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
	// BudgetCache holds.
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
// kind's BudgetCache, and takes them back when it is loaded again, so
// only a restart of the server starts every user full.
type Budget struct {
	burst    int
	interval time.Duration
	// full holds, for each user who spent tokens, when their bucket is full
	// again (the theoretical arrival time of the generic cell rate
	// algorithm). A user not in it is full.
	full map[string]time.Time
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
	at := b.full[user]
	if at.Before(now) {
		at = now
	}
	return at.Add(b.interval).Sub(now) <= time.Duration(b.burst)*b.interval
}

// Charge takes one of user's tokens at now, which Allows found.
func (b *Budget) Charge(user string, now time.Time) {
	at := b.full[user]
	if at.Before(now) {
		at = now
	}
	b.full[user] = at.Add(b.interval)
}

// Refill fills every user's bucket.
func (b *Budget) Refill() { clear(b.full) }

func (b *Budget) prune(now time.Time) {
	maps.DeleteFunc(b.full, func(_ string, at time.Time) bool { return !at.After(now) })
}

// BudgetCache keeps the budgets of rooms that left memory, so that
// evicting a room and loading it again does not fill them. A room hands
// over its buckets that are not full as it leaves memory, and a table
// also the most batches it reached when an undo left it above its
// batches, which a new point is measured against; the room's next load
// takes them back. An entry lasts burst times interval from the moment
// it was handed over, by when each of its buckets would have filled
// anyway. Past size buckets in all (counting one for each room), the
// entries that would expire first are dropped first. Tables and Lobbies
// each own one for the life of the server, so a restart is the one thing
// that starts every budget full.
type BudgetCache struct {
	mu    sync.Mutex
	size  int
	ttl   time.Duration
	rooms map[string]budgetEntry
	held  int
}

type budgetEntry struct {
	until   time.Time
	reached int
	full    map[string]time.Time
}

// NewBudgetCache returns a cache of at most size buckets whose entries
// last ttl.
func NewBudgetCache(size int, ttl time.Duration) *BudgetCache {
	return &BudgetCache{size: size, ttl: ttl, rooms: map[string]budgetEntry{}}
}

// Put keeps code's buckets that are not full at now, and reached, the
// most batches the table reached (0 for none), as the room leaves memory.
// A room with neither leaves nothing.
func (c *BudgetCache) Put(code string, b *Budget, reached int, now time.Time) {
	b.prune(now)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.drop(code)
	if len(b.full) == 0 && reached == 0 {
		return
	}
	e := budgetEntry{until: now.Add(c.ttl), reached: reached, full: maps.Clone(b.full)}
	for _, at := range e.full {
		if at.After(e.until) {
			e.until = at
		}
	}
	n := len(e.full) + 1
	if n > c.size {
		return
	}
	for code, old := range c.rooms {
		if !old.until.After(now) {
			c.drop(code)
		}
	}
	for c.held+n > c.size {
		first := ""
		for code, old := range c.rooms {
			if first == "" || old.until.Before(c.rooms[first].until) {
				first = code
			}
		}
		c.drop(first)
	}
	c.rooms[code] = e
	c.held += n
}

// Take moves code's entry, if it has not expired at now, into b, and
// returns the most batches it reached (0 for none).
func (c *BudgetCache) Take(code string, b *Budget, now time.Time) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.rooms[code]
	if !ok {
		return 0
	}
	c.drop(code)
	if !e.until.After(now) {
		return 0
	}
	for u, at := range e.full {
		if at.After(now) {
			b.full[u] = at
		}
	}
	return e.reached
}

func (c *BudgetCache) drop(code string) {
	if e, ok := c.rooms[code]; ok {
		c.held -= len(e.full) + 1
		delete(c.rooms, code)
	}
}
