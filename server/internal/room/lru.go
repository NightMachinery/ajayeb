package room

import (
	"container/list"
	"slices"
)

// seqLRU remembers the last applied seq of each client id, for at most cap
// clients: the least recently used entry goes first when it is full. A
// forgotten client is not a safety problem, because a re-sent act also
// carries its old base and is then refused as stale; the LRU only saves the
// client a reject. Seqs are the wire's ints (never negative: package wire
// refuses a negative seq).
type seqLRU struct {
	cap   int
	order *list.List               // front is the most recently used; values are *seqEntry
	byID  map[string]*list.Element // client id -> its element in order
}

type seqEntry struct {
	clientID string
	seq      int
}

func newSeqLRU(capacity int) *seqLRU {
	if capacity < 1 {
		capacity = 1
	}
	return &seqLRU{cap: capacity, order: list.New(), byID: make(map[string]*list.Element, capacity)}
}

// get returns the last seq of clientID and whether it is known, and marks
// the client as recently used.
func (l *seqLRU) get(clientID string) (int, bool) {
	e, ok := l.byID[clientID]
	if !ok {
		return 0, false
	}
	l.order.MoveToFront(e)
	return e.Value.(*seqEntry).seq, true
}

// put records seq as clientID's last seq, evicting the least recently used
// client if the LRU is full.
func (l *seqLRU) put(clientID string, seq int) {
	if e, ok := l.byID[clientID]; ok {
		e.Value.(*seqEntry).seq = seq
		l.order.MoveToFront(e)
		return
	}
	if l.order.Len() >= l.cap {
		oldest := l.order.Back()
		l.order.Remove(oldest)
		delete(l.byID, oldest.Value.(*seqEntry).clientID)
	}
	l.byID[clientID] = l.order.PushFront(&seqEntry{clientID: clientID, seq: seq})
}

// load fills the LRU from a room file's seqs. The file does not record which
// client was used last, so they are put in client-id order: the result does
// not depend on the map's iteration order, and it never holds more than cap
// clients, whatever the file says.
func (l *seqLRU) load(stored map[string]int) {
	ids := make([]string, 0, len(stored))
	for id := range stored {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		l.put(id, stored[id])
	}
}

// len returns the number of clients remembered.
func (l *seqLRU) len() int { return l.order.Len() }

// snapshot returns the remembered seqs as a fresh map, for the room file.
func (l *seqLRU) snapshot() map[string]int {
	out := make(map[string]int, l.order.Len())
	for e := l.order.Front(); e != nil; e = e.Next() {
		se := e.Value.(*seqEntry)
		out[se.clientID] = se.seq
	}
	return out
}
