package lobbyroom

import (
	"slices"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/tableroom"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func (r *Record) linkEpoch(user string) (int, bool) {
	for _, s := range r.Slots {
		if (s.Kind == SlotHuman || s.Kind == SlotReserved) && s.UserID == user {
			return s.Epoch, true
		}
	}
	for _, l := range r.Links {
		if l.UserID == user {
			return l.Epoch, true
		}
	}
	return 0, false
}
func (r *Record) pruneLinks() {
	r.Links = slices.DeleteFunc(r.Links, func(l tableroom.LinkEpoch) bool {
		return slotOf(r, l.UserID, SlotHuman) >= 0 || slotOf(r, l.UserID, SlotReserved) >= 0 || r.Names.Get(l.UserID).UserID == ""
	})
}
func (a *actor) link(c *room.Client, seat int, target *string, reset bool) string {
	op := "link"
	if reset {
		op = "relink"
	}
	byUser := target != nil
	if target == nil {
		if seat < 0 || seat >= len(a.rec.Slots) || a.rec.Slots[seat].Kind != SlotHuman {
			return a.refuse(c, 0, op, outBad)
		}
		u := a.rec.Slots[seat].UserID
		target = &u
	}
	r, user := &a.rec, a.userOf(c)
	cl := a.m.Claims()
	if cl == nil || r.Names.Get(*target).UserID == "" {
		return a.refuse(c, 0, op, outBad)
	}
	if user != *target && !a.isAdmin(user) {
		return a.refuse(c, 0, op, outNotAdmin)
	}
	seat = slotOf(r, *target, SlotHuman)
	epoch, issued := r.linkEpoch(*target)
	changed := reset || user != *target || !issued
	if changed {
		cand := r.clone()
		cand.Revision++
		if reset || !issued {
			epoch = cand.Revision
			if seat >= 0 {
				cand.Slots[seat].Epoch = epoch
			} else if reserved := slotOf(r, *target, SlotReserved); reserved >= 0 {
				cand.Slots[reserved].Epoch = epoch
			} else {
				cand.Links = slices.DeleteFunc(cand.Links, func(l tableroom.LinkEpoch) bool { return l.UserID == *target })
				cand.Links = append(cand.Links, tableroom.LinkEpoch{UserID: *target, Epoch: epoch})
			}
		}
		entry := wire.PlatformLogEntry{Kind: "link-copy", By: user, UserID: target}
		if reset {
			entry.Kind = "link"
		}
		if seat >= 0 {
			entry.Seat = &seat
		}
		if reset || user != *target {
			appendLog(&cand, entry)
		}
		if !a.affords(c) {
			return a.refuse(c, 0, op, outRate)
		}
		if err := a.commit(op, cand); err != nil {
			return a.refuse(c, 0, op, outStorage)
		}
		a.charge(c)
	}
	claim := cl.Claim(a.code, *target, epoch)
	if own, ok := a.claims[c]; reset && ok && own.user == *target {
		a.claims[c] = claimed{claim: claim, user: *target}
	}
	linked := wire.PlatformLinked{T: "linked", Code: a.code, Seat: seat, Claim: claim}
	if byUser {
		linked.Seat, linked.UserID = -1, target
	}
	c.Out.Push(linked)
	if changed {
		away := a.revokeClaims()
		a.publish(nil, nil)
		if away {
			a.checkRoles()
		}
		a.armIdle()
		a.publishIdle()
	}
	return a.note(op, outAccepted)
}
