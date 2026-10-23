package lobbyroom

import (
	"slices"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// frame is c's whole lobby snapshot; ack is set only on the sender's own
// connection.
func (a *actor) frame(c *room.Client, ack *wire.PlatformAck) wire.LobbyState {
	r := &a.rec
	st := wire.NewLobbyState(a.code)
	st.Revision = r.Revision
	st.SeatCount = r.SeatCount
	st.Options = cloneOptions(r.Options)
	st.Seats, st.Humans = a.meta()
	user := a.userOf(c)
	st.You = wire.PlatformYou{UserID: user}
	if i := slotOf(r, user, SlotHuman); i >= 0 {
		st.You.Seat = &i
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
		h := wire.PlatformHuman{UserID: userID, Name: name, Connected: a.connected(userID), Creator: userID == r.Roles.Creator,
			Admin: r.Roles.Real(userID), Temporary: r.Roles.Temporary == userID && a.isAdmin(userID), PromotedBy: r.Roles.PromotedBy(userID)}
		same := 0
		for _, o := range humans {
			if o.Name == name {
				same++
			}
		}
		if same > 0 {
			h.NameNo = same + 1
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
			level := s.Bot
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
	for _, c := range a.conns {
		if c == sender {
			c.Out.Push(a.frame(c, ack))
		} else {
			c.Out.Push(a.frame(c, nil))
		}
	}
}

// broadcastExcept sends a presence change to every connection but except.
func (a *actor) broadcastExcept(except *room.Client) {
	for _, c := range a.conns {
		if c != except {
			c.Out.Push(a.frame(c, nil))
		}
	}
}
