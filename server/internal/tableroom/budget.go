package tableroom

import "time"

// Defaults of a table's and a lobby's per-user request budget (Config).
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
)

// Budget is a table's or a lobby's per-user request budget
// (M16.table-request-budget). Every request that would change the room
// takes one token from its user's bucket, shared by all of that user's
// connections: burst tokens, refilled one every interval. A request that
// finds the bucket empty is refused with rate before anything is applied,
// saved or sent to anyone else. Only requests that would change something
// are charged: a duplicate or a refusal costs nothing, and bots and the
// room's own changes are never charged.
//
// Buckets live in memory only. A room loaded again (after a restart, or
// an eviction, which needs it to have had no connection) starts every user
// full: at most one more burst per load.
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

// Spend takes one of user's tokens at now, or takes nothing and reports
// false when none is left. Full buckets are forgotten, so the budget holds
// only users who spent in the last burst times interval.
func (b *Budget) Spend(user string, now time.Time) bool {
	for u, at := range b.full {
		if !at.After(now) {
			delete(b.full, u)
		}
	}
	at := b.full[user]
	if at.Before(now) {
		at = now
	}
	at = at.Add(b.interval)
	if at.Sub(now) > time.Duration(b.burst)*b.interval {
		return false
	}
	b.full[user] = at
	return true
}

// Refill fills every user's bucket.
func (b *Budget) Refill() { clear(b.full) }
