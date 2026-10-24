package tableroom

import (
	"maps"
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
