package room

import (
	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/view"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// This file builds the state frames: one full snapshot per connection, with
// the view and events of that connection's viewer (engine.ViewFor and
// engine.FilterEvents, the only place hidden information is decided).

// frameCache builds each viewer's view and events once per broadcast. The
// views are never changed after they are built, so connections may share
// them.
type frameCache struct {
	views  [3]*view.View       // by viewer + 1: spectator, seat 0, seat 1
	events [3][]view.GameEvent // likewise
	built  [3]bool
}

func (fc *frameCache) get(s *engine.State, v engine.Viewer, ev []engine.Event) (view.View, []view.GameEvent) {
	i := 0
	if seat, ok := v.Seat(); ok {
		i = int(seat) + 1
	}
	if !fc.built[i] {
		vw := engine.ViewFor(s, v)
		fc.views[i] = &vw
		fc.events[i] = engine.FilterEvents(ev, v)
		fc.built[i] = true
	}
	return *fc.views[i], fc.events[i]
}

// viewer is the viewer whose view a member receives: its seat, the seat to
// move in a hot-seat room, or a spectator.
func (r *Room) viewer(mem member) engine.Viewer {
	switch mem.role {
	case rolePlayer:
		return engine.SeatViewer(mem.seat)
	case roleHotseat:
		return engine.SeatViewer(r.state.ToMove)
	}
	return engine.Spectator
}

// broadcast sends every connection a state frame with the events ev (an
// action's events; nil for a presence change or an undo). sender's frame
// carries ack.
func (r *Room) broadcast(ev []engine.Event, sender *Client, ack *wire.Ack) {
	var fc frameCache
	presence := r.presence()
	for c, mem := range r.clients {
		var a *wire.Ack
		if c == sender {
			a = ack
		}
		c.Out.Push(r.frame(mem, a, ev, presence, &fc))
	}
}

// sendState sends c alone a state frame with no events.
func (r *Room) sendState(c *Client, mem member, ack *wire.Ack) {
	var fc frameCache
	c.Out.Push(r.frame(mem, ack, nil, r.presence(), &fc))
}

// frame builds one connection's state frame.
func (r *Room) frame(mem member, ack *wire.Ack, ev []engine.Event, presence [2]bool, fc *frameCache) wire.State {
	v, events := fc.get(&r.state, r.viewer(mem), ev)
	return wire.NewState(r.code, r.version, r.roomInfo(mem, presence), r.you(mem), ack, events, v)
}

// presence reports, by seat, whether the seat's holder is connected: a bot
// always is, an open seat never, a human when one of the user's
// connections is attached.
func (r *Room) presence() [2]bool {
	var out [2]bool
	for i, s := range r.file.Seats {
		switch s.Kind {
		case store.SeatKindBot:
			out[i] = true
		case store.SeatKindHuman:
			for c := range r.clients {
				if c.UserID == s.UserID {
					out[i] = true
					break
				}
			}
		}
	}
	return out
}

// holds reports whether the member acts for seat.
func (mem member) holds(seat engine.Seat) bool {
	switch mem.role {
	case roleHotseat:
		return true
	case rolePlayer:
		return mem.seat == seat
	}
	return false
}

// roomInfo is the room part of a state frame, for one member.
func (r *Room) roomInfo(mem member, presence [2]bool) wire.RoomInfo {
	seats := make([]wire.SeatInfo, 0, len(r.file.Seats))
	for i, s := range r.file.Seats {
		seats = append(seats, wire.SeatInfo{
			Seat:      i,
			Name:      s.Name,
			Kind:      wire.SeatKind(s.Kind),
			Connected: presence[i],
			You:       mem.holds(engine.Seat(i)),
		})
	}
	spectators := 0
	for _, m := range r.clients {
		if m.role == roleSpectator {
			spectators++
		}
	}
	return wire.NewRoomInfo(wire.Mode(r.file.Mode), wireStatus(r.status()), seats, spectators)
}

// you is what the member may do.
func (r *Room) you(mem member) wire.YouInfo {
	var seats []int
	switch mem.role {
	case roleHotseat:
		seats = []int{0, 1}
	case rolePlayer:
		seats = []int{int(mem.seat)}
	}
	canAct := r.status() == store.StatusPlaying && mem.holds(r.state.ToMove)
	return wire.NewYouInfo(seats, canAct)
}

// wireStatus maps a stored status to the client's. An archived room is
// never loaded, so it never reaches a client; it reads as finished.
func wireStatus(s store.Status) wire.RoomStatus {
	switch s {
	case store.StatusLobby:
		return wire.RoomStatusLobby
	case store.StatusPlaying:
		return wire.RoomStatusPlaying
	}
	return wire.RoomStatusFinished
}
