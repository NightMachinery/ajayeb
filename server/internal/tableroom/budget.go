package tableroom

import (
	"maps"
	"sync"
	"time"
)

// Defaults of a table's and a lobby's per-user limits (Config).
const (
	// DefaultRequestBurst and DefaultRequestInterval are each user's
	// request budget: 16 requests at once, then one a second. A table
	// also fills every budget when its game reaches a point it had not
	// reached before. A player changing and confirming a pick spends a few
	// at each point, an admin reseating all seven seats seven, and a
	// reconnecting tab's resent requests are duplicates, which cost
	// nothing.
	DefaultRequestBurst    = 16
	DefaultRequestInterval = time.Second
	// DefaultMaxUserClients bounds one user's connections to a table or
	// a lobby, seated or watching.
	DefaultMaxUserClients = 8
	// DefaultBudgetCacheSize bounds the buckets, and rooms, a
	// BudgetCache holds, and separately the rooms it remembers dropping.
	DefaultBudgetCacheSize = 4096
)

// Budget is a table's or a lobby's per-user request budget
// (M16.table-request-budget). Every request that would change the room
// takes one token from its user's bucket, shared by all of that user's
// connections: burst tokens, refilled one every interval. A request is
// checked (Allows) once its cheap checks pass and before any other work,
// and charged (Charge) only once it is applied, so a request that finds
// the bucket empty is refused with rate before anything is computed,
// saved or sent to anyone else. A duplicate or a refusal costs nothing,
// and bots and the room's own changes are never charged.
//
// A room that leaves memory hands its buckets that are not full to its
// kind's BudgetCache, and takes them back when it is loaded again.
type Budget struct {
	burst    int
	interval time.Duration
	// full holds, for each user who spent tokens, when their bucket is full
	// again (the theoretical arrival time of the generic cell rate
	// algorithm). A user not in it is full, or full at floor.
	full map[string]time.Time
	// floor, while after now, is when every bucket is full at the
	// latest: the room's buckets that the BudgetCache had to drop, unknown
	// user by unknown user, were all full by then. The table's high-water
	// mark may have been dropped with them, so while it stands a new point
	// fills nothing (Refill).
	floor time.Time
}

// NewBudget returns a budget of burst tokens for each user, refilling one
// every interval.
func NewBudget(burst int, interval time.Duration) *Budget {
	return &Budget{burst: burst, interval: interval, full: map[string]time.Time{}}
}

// Allows reports whether user has a token at now. It takes nothing.
// Buckets that are full are forgotten, so the budget holds only users who
// spent in the last burst times interval.
func (b *Budget) Allows(user string, now time.Time) bool {
	b.prune(now)
	return b.fullAt(user, now).Add(b.interval).Sub(now) <= time.Duration(b.burst)*b.interval
}

// Charge takes one of user's tokens at now, which Allows found.
func (b *Budget) Charge(user string, now time.Time) {
	b.full[user] = b.fullAt(user, now).Add(b.interval)
}

// fullAt is when user's bucket is full, now if it is.
func (b *Budget) fullAt(user string, now time.Time) time.Time {
	at := b.full[user]
	if at.Before(b.floor) {
		at = b.floor
	}
	if at.Before(now) {
		at = now
	}
	return at
}

// Refill fills every user's bucket, unless a floor stands at now: then
// the point may not be new at all (its mark may have been dropped), and
// every bucket stays as it is until the floor has passed.
func (b *Budget) Refill(now time.Time) {
	if b.floor.After(now) {
		return
	}
	clear(b.full)
	b.floor = time.Time{}
}

func (b *Budget) prune(now time.Time) {
	maps.DeleteFunc(b.full, func(_ string, at time.Time) bool { return !at.After(now) })
}

// BudgetCache keeps the budgets of rooms that left memory, so that
// evicting a room and loading it again does not fill them. A room hands
// over its buckets that are not full as it leaves memory, and a table
// also the most batches it reached when an undo left it above its
// batches, which a new point is measured against; the room's next load
// takes them back. An entry lasts until its last bucket is full, or, with
// a high-water mark, at least ttl (burst times interval) from the moment
// it was handed over. Past size buckets in all (counting one for each
// room), the entry that would expire first, the arriving one included, is
// dropped.
//
// A dropped entry leaves a note of when its buckets would all have been
// full (and its high-water mark), and the room's next load starts every
// user's bucket full only from then (the budget's floor). Past size notes,
// the notes that would expire first are dropped, and until the last of
// them would have expired every load that finds neither an entry nor a
// note starts from its time, its mark unknown. While a floor stands a new
// point fills nothing, so a dropped mark cannot make an old point refill
// before the floor has passed. Tables and Lobbies each own a cache for
// the life of the server.
type BudgetCache struct {
	mu    sync.Mutex
	size  int
	ttl   time.Duration
	rooms map[string]budgetEntry
	held  int
	// lost holds the notes of dropped entries, and blind the latest time a
	// dropped note was until.
	lost  map[string]budgetNote
	blind time.Time
}

type budgetEntry struct {
	until   time.Time
	reached int
	floor   time.Time
	full    map[string]time.Time
}

type budgetNote struct {
	until   time.Time
	reached int
}

// NewBudgetCache returns a cache of at most size buckets, and size notes,
// that keeps a high-water mark at least ttl.
func NewBudgetCache(size int, ttl time.Duration) *BudgetCache {
	return &BudgetCache{size: size, ttl: ttl, rooms: map[string]budgetEntry{}, lost: map[string]budgetNote{}}
}

// Put keeps code's buckets that are not full at now, and reached, the
// most batches the table reached (0 for none), as the room leaves memory.
// A room with neither leaves nothing.
func (c *BudgetCache) Put(code string, b *Budget, reached int, now time.Time) {
	b.prune(now)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.drop(code)
	delete(c.lost, code)
	e := budgetEntry{reached: reached, full: maps.Clone(b.full)}
	if b.floor.After(now) {
		e.floor = b.floor
	}
	if len(e.full) == 0 && e.floor.IsZero() && reached == 0 {
		return
	}
	e.until = e.floor
	for _, at := range e.full {
		if at.After(e.until) {
			e.until = at
		}
	}
	if reached > 0 && e.until.Before(now.Add(c.ttl)) {
		e.until = now.Add(c.ttl)
	}
	for code, old := range c.rooms {
		if !old.until.After(now) {
			c.drop(code)
		}
	}
	n := len(e.full) + 1
	if n > c.size {
		c.note(code, e, now)
		return
	}
	for c.held+n > c.size {
		first := ""
		for k, old := range c.rooms {
			if first == "" || old.until.Before(c.rooms[first].until) {
				first = k
			}
		}
		if first == "" || !c.rooms[first].until.Before(e.until) {
			// The arriving entry would expire first: it is the one noted.
			c.note(code, e, now)
			return
		}
		c.note(first, c.rooms[first], now)
		c.drop(first)
	}
	c.rooms[code] = e
	c.held += n
}

// Take moves code's entry, if it has not expired at now, into b, and
// returns the most batches it reached (0 for none). Without an entry, a
// note of a dropped one, or a dropped note, makes b's buckets full only
// from when the dropped ones would have been.
func (c *BudgetCache) Take(code string, b *Budget, now time.Time) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.rooms[code]; ok {
		c.drop(code)
		if !e.until.After(now) {
			return 0
		}
		for u, at := range e.full {
			if at.After(now) {
				b.full[u] = at
			}
		}
		if e.floor.After(now) {
			b.floor = e.floor
		}
		return e.reached
	}
	n, ok := c.lost[code]
	if ok {
		delete(c.lost, code)
	} else {
		n.until = c.blind
	}
	if !n.until.After(now) {
		return 0
	}
	b.floor = n.until
	return n.reached
}

func (c *BudgetCache) drop(code string) {
	if e, ok := c.rooms[code]; ok {
		c.held -= len(e.full) + 1
		delete(c.rooms, code)
	}
}

// note remembers that code's entry e was dropped at now, dropping the
// notes that would expire first when there are size of them.
func (c *BudgetCache) note(code string, e budgetEntry, now time.Time) {
	for code, old := range c.lost {
		if !old.until.After(now) {
			delete(c.lost, code)
		}
	}
	if !e.until.After(now) {
		return
	}
	for len(c.lost) >= c.size {
		first := ""
		for code, old := range c.lost {
			if first == "" || old.until.Before(c.lost[first].until) {
				first = code
			}
		}
		if c.lost[first].until.After(c.blind) {
			c.blind = c.lost[first].until
		}
		delete(c.lost, first)
	}
	c.lost[code] = budgetNote{until: e.until, reached: e.reached}
}
