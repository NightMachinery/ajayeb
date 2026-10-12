package room

import (
	"context"
	"strings"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// This file holds who is who in a room beyond the seats themselves: device
// links (claims, package ident), with which one seat's user plays from
// another device, and the numbers that tell equal display names apart.

// --- device links ------------------------------------------------------------

// claimHolder returns the user of the seat claim was made for, among the
// human seats of room code at their current epochs, and whether there is
// one. Every seat is checked, in constant time each (Claims.VerifyClaim),
// whatever the first one says. Without Claims nothing is a claim.
func claimHolder(cl Claims, code string, seats [2]store.Seat, claim string) (string, bool) {
	if cl == nil {
		return "", false
	}
	user := ""
	for _, s := range seats {
		if s.Kind != store.SeatKindHuman || s.UserID == "" {
			continue
		}
		if cl.VerifyClaim(claim, code, s.UserID, s.Epoch) && user == "" {
			user = s.UserID
		}
	}
	return user, user != ""
}

// Link routes a link frame (a request for one seat's device link) to c's
// room.
func (m *Manager) Link(ctx context.Context, c *Client, l wire.Link) error {
	r, err := m.joinedRoom(c, l.Code)
	if err == nil {
		err = r.post(ctx, msgLink{c: c, seat: l.Seat})
	}
	return m.requestFailed(c, 0, err)
}

// Relink routes a relink frame (the reset of one seat's device link) to
// c's room.
func (m *Manager) Relink(ctx context.Context, c *Client, l wire.Relink) error {
	r, err := m.joinedRoom(c, l.Code)
	if err == nil {
		err = r.post(ctx, msgLink{c: c, seat: l.Seat, reset: true})
	}
	return m.requestFailed(c, 0, err)
}

// link answers a link frame, or with reset a relink frame, in this order:
// the sender must be attached and not a spectator; device links must be on
// (Config.Claims); a human must hold the seat; and the sender must act as
// that seat's user or be the game's admin. A reset then raises the seat's
// epoch, which revokes every claim made for it before, logs the reset for
// every viewer, sends every connection a fresh state, and ends the
// connections that joined with a revoked claim (revokeClaims). Either way
// the sender alone receives the seat's current claim, in a linked frame.
// A link never changes the room's version: no position changed.
func (r *Room) link(c *Client, seat int, reset bool) {
	mem, ok := r.clients[c]
	if !ok {
		c.Out.Push(wire.NewReject(0, wire.RejectCodeBadRequest, "join the room first"))
		return
	}
	if mem.role == roleSpectator {
		c.Out.Push(wire.NewReject(0, wire.RejectCodeNotAdmin, "only a seat's player or the game's admin may have its link"))
		return
	}
	cl := r.m.cfg.Claims
	if cl == nil {
		c.Out.Push(wire.NewReject(0, wire.RejectCodeBadRequest, "device links are off on this server"))
		return
	}
	s := r.file.Seats[seat]
	if s.Kind != store.SeatKindHuman || s.UserID == "" {
		c.Out.Push(wire.NewReject(0, wire.RejectCodeBadRequest, "no player holds that seat"))
		return
	}
	if mem.user != s.UserID && mem.user != r.file.Admin {
		c.Out.Push(wire.NewReject(0, wire.RejectCodeNotAdmin, "only a seat's player or the game's admin may have its link"))
		return
	}
	if reset {
		r.raiseEpoch(s.UserID)
		r.logLink(mem, seat)
		r.persist()
		r.log.Info("device link reset", "seat", seat)
	}
	claim := cl.Claim(r.code, s.UserID, r.file.Seats[seat].Epoch)
	if reset && mem.claim != "" && mem.user == s.UserID {
		// The sender joined with the claim it has just revoked: it keeps
		// its place under the new one, which it is about to receive.
		mem.claim = claim
		r.clients[c] = mem
	}
	c.Out.Push(wire.NewLinked(r.code, seat, claim))
	if reset {
		r.revokeClaims()
		r.broadcast(nil, nil, nil, noHistory)
	}
}

// raiseEpoch revokes every claim made for user in this room: it raises the
// epoch of every seat user holds (both seats of a hot-seat room, whose
// claims would otherwise live on through the other seat) to one past the
// highest of them, so no claim of an earlier epoch verifies for any.
func (r *Room) raiseEpoch(user string) {
	top := 0
	for _, s := range r.file.Seats {
		if s.Kind == store.SeatKindHuman && s.UserID == user {
			top = max(top, s.Epoch)
		}
	}
	for i := range r.file.Seats {
		if s := &r.file.Seats[i]; s.Kind == store.SeatKindHuman && s.UserID == user {
			s.Epoch = top + 1
		}
	}
}

// revokeClaims ends every connection that joined with a claim which is no
// longer one of the room's (after a link reset): it receives error
// bad-claim and is detached, like a connection that left. The transport
// still thinks it joined, so its next request is refused as "join the room
// first", and a join with its own identity (or a fresh link) is what lets
// it back in.
func (r *Room) revokeClaims() {
	for c, mem := range r.clients {
		if mem.claim == "" {
			continue
		}
		if _, ok := claimHolder(r.m.cfg.Claims, r.code, r.file.Seats, mem.claim); ok {
			continue
		}
		delete(r.clients, c)
		c.Out.Push(wire.NewError(wire.ErrorCodeBadClaim, "that device link was reset"))
	}
	if len(r.clients) == 0 {
		r.cancelBot()
		r.emptySince = r.m.cfg.Clock.Now()
		r.armIdle()
		r.publishIdle()
	}
}

// logLink records the reset of seat's link in the room log, by the member
// mem (a seat's player, or the admin; no seat in a hot-seat room).
func (r *Room) logLink(mem member, seat int) {
	by := -1
	if mem.role == rolePlayer {
		by = int(mem.seat)
	}
	target := seat
	r.file.RoomLog = append(r.file.RoomLog, store.LogEntry{
		Kind:   string(wire.RoomLogLink),
		At:     int(r.state.Ply),
		Seat:   by,
		On:     []string{},
		Off:    []string{},
		Target: &target,
	})
	if n := len(r.file.RoomLog); n > wire.MaxRoomLog {
		r.file.RoomLog = append([]store.LogEntry{}, r.file.RoomLog[n-wire.MaxRoomLog:]...)
	}
}

// rematchCreated is the created frame that tells cl the rematch room's
// code. A connection that joined this room with a claim also receives the
// claim of its user's seat there (the rematch keeps the seats and starts
// their epochs at 0), since its own identity holds no seat in the rematch
// room.
func (r *Room) rematchCreated(cl *Client, mem member) wire.Created {
	f := wire.NewCreated(r.rematch)
	if mem.claim != "" && r.m.cfg.Claims != nil {
		claim := r.m.cfg.Claims.Claim(r.rematch, mem.user, 0)
		f.Claim = &claim
	}
	return f
}

// --- display names -----------------------------------------------------------

// nameKey is the form in which two display names are compared: without
// case and outer spaces.
func nameKey(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// numberSeat gives seat i its name number (store.Seat.NameNo), when it has
// just taken its name: 0 unless the other seat is a human whose name is
// equal, which then came first and keeps it, and seat i takes the next
// number after the other's (2 after an unnumbered name). Only seat i
// changes, so a number, once given, stays until its own seat is renamed.
func numberSeat(seats *[2]store.Seat, i int) {
	s := &seats[i]
	s.NameNo = 0
	key := nameKey(s.Name)
	o := seats[1-i]
	if s.Kind != store.SeatKindHuman || key == "" || o.Kind != store.SeatKindHuman || nameKey(o.Name) != key {
		return
	}
	s.NameNo = max(o.NameNo, 1) + 1
}

// numberNewSeats numbers seat 1 when both seats came with equal names and
// no number yet: a hot-seat room created with two equal names, or a room
// file written before numbers existed. Seat 0 counts as the first holder.
// Rooms with numbers already (a rematch keeps them) are left as they are.
func numberNewSeats(seats *[2]store.Seat) {
	if seats[0].NameNo == 0 && seats[1].NameNo == 0 {
		numberSeat(seats, 1)
	}
}

// rename gives an online room's seat the name its user now says hello
// with, and reports whether it changed: the user renamed themselves since
// they took the seat. Only that seat's number is recomputed. An empty name
// (no name chosen) keeps the seat's; hot-seat and bot rooms keep the names
// chosen when they were created.
func (r *Room) rename(seat engine.Seat, name string) bool {
	s := &r.file.Seats[seat]
	if r.file.Mode != store.ModeOnline || name == "" || name == s.Name {
		return false
	}
	s.Name = name
	numberSeat(&r.file.Seats, int(seat))
	return true
}
