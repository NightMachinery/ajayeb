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
// frame inherits the older one's events (in front of its own, so the client
// can still animate everything that happened) and, when it has no ack of
// its own, the older one's ack. Every other frame is kept, in order.
//
// An Outbox is safe for one pusher and one taker at a time, from different
// goroutines.
type Outbox struct {
	mu       sync.Mutex
	queue    []wire.ServerMessage
	other    int // frames in queue that are not wire.State
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
	if st, ok := m.(wire.State); ok {
		o.pushState(st)
	} else {
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
// appends st. The caller holds mu.
func (o *Outbox) pushState(st wire.State) {
	for i, m := range o.queue {
		old, ok := m.(wire.State)
		if !ok || old.Code != st.Code {
			continue
		}
		if len(old.Events) > 0 {
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
		break
	}
	o.queue = append(o.queue, st)
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
	o.signal()
}
