package lobbyroom

import (
	"slices"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// broadcast shares public metadata for one publication only. Viewer-specific
// filtering must copy its slices before changing an element.
type broadcast struct {
	a      *actor
	seats  []wire.PlatformSeat
	humans []wire.PlatformHuman
}

func (a *actor) newBroadcast() *broadcast {
	seats, humans := a.meta()
	return &broadcast{a: a, seats: seats, humans: humans}
}

// frame builds a standalone snapshot (join, resync or duplicate).
func (a *actor) frame(c *room.Client, ack *wire.PlatformAck) wire.LobbyState {
	return a.newBroadcast().frame(c, ack)
}

// frame is c's whole lobby snapshot; ack is set only on the sender's own
// connection.
func (b *broadcast) frame(c *room.Client, ack *wire.PlatformAck) wire.LobbyState {
	a := b.a
	r := &a.rec
	st := wire.NewLobbyState(a.code)
	st.Revision = r.Revision
	st.SeatCount = r.SeatCount
	st.MaxSeats = a.l.maxSeats()
	st.Options = cloneOptions(r.Options)
	st.Seats, st.Humans = b.seats, b.humans
	user := a.userOf(c)
	admin := a.isAdmin(user)
	if admin && !r.Roles.Real(user) {
		st.Humans = slices.Clone(st.Humans)
		for i := range st.Humans {
			if r.Roles.Real(st.Humans[i].UserID) {
				st.Humans[i].Linkable = nil
			}
		}
	}
	st.You = wire.PlatformYou{UserID: user}
	if i := slotOf(r, user, SlotHuman); i >= 0 {
		st.You.Seat = &i
	}
	st.You.RejoinSeat = rejoinSeat(r, user)
	if admin {
		room := r.Roles.HasRoom(user)
		st.You.CanPromote = &room
	}
	st.CanStart = a.canStart(user)
	st.Ack = ack
	st.RoomLog = append([]wire.PlatformLogEntry{}, r.RoomLog...)
	return st
}

// meta is the public metadata: the slots, and the humans (those in slots
// in slot order, a reservation's user too, then the other connected users
// in the order they arrived) with presence and roles.
func (a *actor) meta() ([]wire.PlatformSeat, []wire.PlatformHuman) {
	r := &a.rec
	seats := make([]wire.PlatformSeat, 0, len(r.Slots))
	humans := []wire.PlatformHuman{}
	add := func(userID, name string) {
		if slices.ContainsFunc(humans, func(h wire.PlatformHuman) bool { return h.UserID == userID }) {
			return
		}
		if assigned := a.assignment(userID); assigned.UserID != "" {
			name = assigned.Name
		}
		h := wire.PlatformHuman{UserID: userID, Name: name, Connected: a.connected(userID), Creator: userID == r.Roles.Creator,
			Admin: r.Roles.Real(userID), Temporary: r.Roles.Temp(userID) && a.isAdmin(userID), PromotedBy: r.Roles.PromotedBy(userID)}
		h.NameNo = a.assignment(userID).Number
		if r.Names.Get(userID).UserID != "" {
			yes := true
			h.Linkable = &yes
		}
		humans = append(humans, h)
	}
	for i, s := range r.Slots {
		ps := wire.PlatformSeat{Seat: i, Kind: s.Kind}
		switch s.Kind {
		case SlotHuman, SlotReserved:
			id := s.UserID
			ps.UserID = &id
			add(s.UserID, s.Name)
		case SlotBot:
			level := creationLevel(s.Bot, r.SeatCount)
			ps.Bot = &level
		}
		seats = append(seats, ps)
	}
	for _, c := range a.conns {
		add(a.userOf(c), c.Name)
	}
	return seats, humans
}

// publish sends every connection its frame; ack goes to the sender only.
func (a *actor) publish(sender *room.Client, ack *wire.PlatformAck) {
	b := a.newBroadcast()
	for _, c := range a.conns {
		if c == sender {
			c.Out.Push(b.frame(c, ack))
		} else {
			c.Out.Push(b.frame(c, nil))
		}
	}
}

// broadcastExcept sends a presence change to every connection but except.
func (a *actor) broadcastExcept(except *room.Client) {
	b := a.newBroadcast()
	for _, c := range a.conns {
		if c != except {
			c.Out.Push(b.frame(c, nil))
		}
	}
}
