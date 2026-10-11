package room

import (
	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/view"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// This file builds the state frames: one full snapshot per connection, with
// the view and events of that connection's viewer (engine.ViewFor and
// engine.FilterEvents, the only place hidden information is decided), and,
// for the frames that carry it, the game's history filtered the same way.

// frameCache builds each viewer's view, events and history once per
// broadcast. They are never changed after they are built, so connections
// may share them.
type frameCache struct {
	views   [3]*view.View       // by viewer + 1: spectator, seat 0, seat 1
	events  [3][]view.GameEvent // likewise
	built   [3]bool
	history [3]*[]wire.HistoryEntry // likewise, built only when a frame needs it
}

// viewerIndex is a viewer's index in a frameCache.
func viewerIndex(v engine.Viewer) int {
	if seat, ok := v.Seat(); ok {
		return int(seat) + 1
	}
	return 0
}

func (fc *frameCache) get(s *engine.State, v engine.Viewer, ev []engine.Event) (view.View, []view.GameEvent) {
	i := viewerIndex(v)
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

// historyFor returns viewer v's history: every entry of the room's history
// (the newest wire.MaxHistory), each with its events filtered for v.
func (fc *frameCache) historyFor(hist []histEntry, v engine.Viewer) *[]wire.HistoryEntry {
	i := viewerIndex(v)
	if fc.history[i] == nil {
		hist = hist[max(0, len(hist)-wire.MaxHistory):]
		out := make([]wire.HistoryEntry, 0, len(hist))
		for _, e := range hist {
			out = append(out, wire.HistoryEntry{Ply: int(e.ply), Seat: int(e.seat), Events: engine.FilterEvents(e.events, v)})
		}
		fc.history[i] = &out
	}
	return fc.history[i]
}

// historyTo says which connections of a broadcast receive the history.
type historyTo func(c *Client) bool

// noHistory sends the history to nobody (live frames and presence changes).
func noHistory(*Client) bool { return false }

// allHistory sends the history to everyone (after an undo, which takes
// back moves every connection has seen).
func allHistory(*Client) bool { return true }

// broadcast sends every connection a state frame with the events ev (an
// action's events; nil for a presence change or an undo). sender's frame
// carries ack, and the connections withHistory names receive the history
// too (a frame with the history never carries events).
func (r *Room) broadcast(ev []engine.Event, sender *Client, ack *wire.Ack, withHistory historyTo) {
	var fc frameCache
	presence := r.presence()
	for c, mem := range r.clients {
		var a *wire.Ack
		if c == sender {
			a = ack
		}
		c.Out.Push(r.frame(mem, a, ev, presence, &fc, withHistory(c)))
	}
}

// sendState sends c alone a state frame with no events, and with the
// history when withHistory is set (a join, a resync).
func (r *Room) sendState(c *Client, mem member, ack *wire.Ack, withHistory bool) {
	var fc frameCache
	c.Out.Push(r.frame(mem, ack, nil, r.presence(), &fc, withHistory))
}

// frame builds one connection's state frame.
func (r *Room) frame(mem member, ack *wire.Ack, ev []engine.Event, presence [2]bool, fc *frameCache, withHistory bool) wire.State {
	viewer := r.viewer(mem)
	v, events := fc.get(&r.state, viewer, ev)
	if withHistory {
		// The history already holds the last action's events.
		events = nil
	}
	st := wire.NewState(r.code, r.version, r.roomInfo(mem, presence), r.you(mem), ack, events, v)
	st.CanUndo = r.canUndo(mem)
	st.Thinking = r.thinking()
	if withHistory {
		st.History = fc.historyFor(r.hist, viewer)
	}
	return st
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
		info := wire.SeatInfo{
			Seat:      i,
			Name:      s.Name,
			Kind:      wire.SeatKind(s.Kind),
			Connected: presence[i],
			You:       mem.holds(engine.Seat(i)),
		}
		if s.Kind == store.SeatKindBot {
			level := wire.BotLevel(s.Bot)
			info.Bot = &level
		}
		seats = append(seats, info)
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
