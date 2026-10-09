package room

import "container/list"

// seqLRU remembers the last seq seen from each client id, for at most cap
// clients: the least recently used entry goes first when it is full. A
// forgotten client is not a safety problem, because a re-sent act also
// carries its old base and is then refused as stale; the LRU only saves the
// client a reject.
type seqLRU struct {
	cap   int
	order *list.List               // front is the most recently used; values are *seqEntry
	byID  map[string]*list.Element // client id -> its element in order
}

type seqEntry struct {
	clientID string
	seq      uint64
}

func newSeqLRU(capacity int) *seqLRU {
	if capacity < 1 {
		capacity = 1
	}
	return &seqLRU{cap: capacity, order: list.New(), byID: make(map[string]*list.Element, capacity)}
}

// get returns the last seq of clientID and whether it is known, and marks
// the client as recently used.
func (l *seqLRU) get(clientID string) (uint64, bool) {
	e, ok := l.byID[clientID]
	if !ok {
		return 0, false
	}
	l.order.MoveToFront(e)
	return e.Value.(*seqEntry).seq, true
}

// put records seq as clientID's last seq, evicting the least recently used
// client if the LRU is full.
func (l *seqLRU) put(clientID string, seq uint64) {
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

// len returns the number of clients remembered.
func (l *seqLRU) len() int { return l.order.Len() }

// snapshot returns the remembered seqs as a fresh map, for the room file.
func (l *seqLRU) snapshot() map[string]uint64 {
	out := make(map[string]uint64, l.order.Len())
	for e := l.order.Front(); e != nil; e = e.Next() {
		se := e.Value.(*seqEntry)
		out[se.clientID] = se.seq
	}
	return out
}
