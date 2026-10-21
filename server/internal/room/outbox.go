package room

import (
	"sync"

	"github.com/NightMachinery/ajayeb/server/internal/view"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// Outbox limits.
const (
	// MaxPendingFrames bounds the frames other than state frames (created,
	// reject, error) waiting in one outbox. A client that lets more pile up
	// is not reading; the outbox then reports Overflowed and the transport
	// should drop the connection.
	MaxPendingFrames = 32
	// MaxPendingStates bounds the rooms whose state frames wait in one
	// outbox at once (each room keeps only its newest one). A connection is
	// in one room at a time, so a few are enough to cover a switch from one
	// room to another; a client that lets more pile up is not reading, and
	// the outbox reports Overflowed.
	MaxPendingStates = 4
	// MaxMergedEvents bounds the events a pending state frame accumulates
	// from the frames it replaced (see Outbox.Push); the oldest go first.
	MaxMergedEvents = 256
)

// Outbox is one connection's queue of server frames: the room pushes, and
// the connection's writer takes. Push never blocks, so a slow client can
// never stall a room.
//
// State frames are full snapshots, so the outbox keeps only the newest
// state frame of each room (latest wins): pushing a state frame removes an
// older one for the same room code that has not been taken yet. The newer
// frame inherits, when it has no ack of its own, the older one's ack, and,
// when its position follows from the older one's, the older one's events
// (in front of its own, so the client can still animate everything that
// happened; see continues). The history some frames carry (a join's, a
// resync's, an undo's) is never lost either: a newer frame without one
// inherits the older one's, so its own events follow that history's last
// entry; a newer frame with its own history needs none of the older one's
// events, since its history already holds them. Every other frame is kept,
// in order. Both kinds are bounded: MaxPendingFrames other frames and the
// state frames of MaxPendingStates rooms; past either, the frame is dropped
// and the outbox reports Overflowed.
//
// An Outbox is safe for one pusher and one taker at a time, from different
// goroutines.
type Outbox struct {
	mu       sync.Mutex
	queue    []wire.ServerMessage
	other    int // non-snapshot frames in queue
	states   int // typed snapshots, one per room across all families
	overflow bool
	closed   bool
	ready    chan struct{} // capacity 1: a token means "queue may be non-empty"
}

// NewOutbox returns an empty outbox.
func NewOutbox() *Outbox {
	return &Outbox{ready: make(chan struct{}, 1)}
}

// Push queues m for the connection. After Close it does nothing.
func (o *Outbox) Push(m wire.ServerMessage) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed {
		return
	}
	switch st := m.(type) {
	case wire.State:
		o.dropOtherFamily(st.Code, "duel")
		o.pushState(st)
	case wire.TableState:
		o.dropOtherFamily(st.Code, "haftgane")
		o.pushTableState(st)
	case wire.LobbyState:
		o.dropOtherFamily(st.Code, "lobby")
		o.pushLobbyState(st)
	default:
		if o.other >= MaxPendingFrames {
			o.overflow = true
		} else {
			o.queue = append(o.queue, m)
			o.other++
		}
	}
	o.signal()
}

// pushState replaces the pending state frame of the same room, if any, and
// appends st. A state frame of one more room than MaxPendingStates is
// dropped and sets overflow. The caller holds mu.
func (o *Outbox) pushState(st wire.State) {
	replaced := false
	for i, m := range o.queue {
		old, ok := m.(wire.State)
		if !ok || old.Code != st.Code {
			continue
		}
		switch {
		case st.History != nil:
			// st's history already holds old's events.
		case old.History != nil && continues(old, st):
			st.History = old.History
			fallthrough
		case len(old.Events) > 0 && continues(old, st):
			merged := make([]view.GameEvent, 0, len(old.Events)+len(st.Events))
			merged = append(merged, old.Events...)
			merged = append(merged, st.Events...)
			if len(merged) > MaxMergedEvents {
				merged = merged[len(merged)-MaxMergedEvents:]
			}
			st.Events = merged
		}
		if st.Ack == nil {
			st.Ack = old.Ack
		}
		o.queue = append(o.queue[:i], o.queue[i+1:]...)
		replaced = true
		break
	}
	if !replaced {
		if o.states >= MaxPendingStates {
			o.overflow = true
			return
		}
		o.states++
	}
	o.queue = append(o.queue, st)
}

// continues reports whether next's position follows from prev's by
// forward moves only, so that prev's events still lead towards next. Events
// animate the way from the position the client last took to the new one:
//
//   - a newer move (version and ply both up) continues;
//   - a frame for the same position (same version: a presence change, a
//     resync, the reply to a duplicate) continues, so a presence change
//     never swallows a move's events;
//   - an undo moves the version on but takes the ply back: the undone
//     moves' events would animate something that no longer happened, so
//     they are dropped and the client simply jumps to the new position;
//   - a version that went back (which a room never sends, since it stores
//     its version) is treated the same way.
func continues(prev, next wire.State) bool {
	return next.Version >= prev.Version && next.View.Ply >= prev.View.Ply
}

// signal leaves a token in ready. The caller holds mu.
func (o *Outbox) signal() {
	select {
	case o.ready <- struct{}{}:
	default:
	}
}

// Ready returns a channel that receives a value whenever frames may be
// waiting. The taker waits on it and then calls Take.
func (o *Outbox) Ready() <-chan struct{} { return o.ready }

// Take removes and returns every waiting frame, oldest first. It returns
// nil when nothing is waiting.
func (o *Outbox) Take() []wire.ServerMessage {
	o.mu.Lock()
	defer o.mu.Unlock()
	if len(o.queue) == 0 {
		return nil
	}
	out := o.queue
	o.queue = nil
	o.other = 0
	o.states = 0
	return out
}

// Overflowed reports whether a frame was dropped because the client was not
// reading; the transport should then close the connection.
func (o *Outbox) Overflowed() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.overflow
}

// Close drops whatever is waiting and makes later pushes do nothing.
func (o *Outbox) Close() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.closed = true
	o.queue = nil
	o.other = 0
	o.states = 0
	o.signal()
}
