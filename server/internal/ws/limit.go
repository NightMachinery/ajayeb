package ws

import (
	"time"
)

// rate is a token bucket in its GCRA form (the generic cell rate
// algorithm): instead of a token count and a refill timestamp, a bucket is
// one instant, the theoretical arrival time (TAT) of the next request. A
// request at now is allowed while TAT is at most tolerance ahead of now, and
// each allowed request pushes TAT on by interval. That admits burst requests
// at once and then one per interval, exactly like a bucket of burst tokens
// refilled at one token per interval, with integer time arithmetic only.
type rate struct {
	interval  time.Duration // time to earn one request back: period / count
	tolerance time.Duration // (burst - 1) * interval
}

// newRate returns a limit of count requests per period with a burst of
// burst. count and burst must be positive.
func newRate(count int, period time.Duration, burst int) rate {
	interval := period / time.Duration(count)
	return rate{interval: interval, tolerance: time.Duration(burst-1) * interval}
}

// bucket is one limited subject's state under a rate: its TAT. The zero
// bucket is full.
type bucket struct {
	tat time.Time
}

// next returns the TAT the bucket would have after allowing a request at
// now, and whether the request is allowed.
func (r rate) next(b bucket, now time.Time) (time.Time, bool) {
	tat := b.tat
	if tat.Before(now) {
		tat = now
	}
	if tat.Sub(now) > r.tolerance {
		return b.tat, false
	}
	return tat.Add(r.interval), true
}

// allow reports whether a request at now is within the limit, and records
// it when it is. A refused request leaves the bucket unchanged.
func (r rate) allow(b *bucket, now time.Time) bool {
	tat, ok := r.next(*b, now)
	if ok {
		b.tat = tat
	}
	return ok
}

// full reports whether the bucket has refilled completely by now, so that
// forgetting it changes nothing.
func (b bucket) full(now time.Time) bool { return !b.tat.After(now) }

// keyedLimiter applies one rate to many keys (user ids, IP addresses). It
// remembers at most capacity keys: when a new key arrives then, the keys
// whose buckets have refilled are forgotten first (they are equivalent to a
// fresh bucket), and if every remembered key is still limited the new key is
// refused. Refusing (rather than evicting a limited key, which would reset
// its limit) keeps the limit sound under a flood of distinct keys; it can
// only refuse newcomers while capacity keys are all at their limit at once.
//
// A keyedLimiter is not safe for concurrent use; the Server guards its
// limiters with one mutex.
type keyedLimiter[K comparable] struct {
	rate     rate
	capacity int
	buckets  map[K]bucket
}

func newKeyedLimiter[K comparable](r rate, capacity int) *keyedLimiter[K] {
	return &keyedLimiter[K]{rate: r, capacity: capacity, buckets: make(map[K]bucket)}
}

// check reports whether a request for key at now would be allowed, without
// recording it.
func (l *keyedLimiter[K]) check(key K, now time.Time) bool {
	b, known := l.buckets[key]
	if !known && len(l.buckets) >= l.capacity {
		l.prune(now)
		if len(l.buckets) >= l.capacity {
			return false
		}
	}
	_, ok := l.rate.next(b, now)
	return ok
}

// limited reports whether key is over its limit at now. A key the limiter
// does not remember is never limited: its bucket would be full. Unlike
// check, it never refuses a newcomer, for limits that must not lock out
// everyone else while a flood of keys fills the table (see record).
func (l *keyedLimiter[K]) limited(key K, now time.Time) bool {
	b, known := l.buckets[key]
	if !known {
		return false
	}
	_, ok := l.rate.next(b, now)
	return !ok
}

// record counts a request for key at now, if the limiter can remember the
// key: when it remembers capacity keys whose buckets are all still
// refilling, a new key goes uncounted rather than pushing out another key's
// limit. Only a flood of capacity distinct keys gets that far.
func (l *keyedLimiter[K]) record(key K, now time.Time) {
	if _, known := l.buckets[key]; !known && len(l.buckets) >= l.capacity {
		l.prune(now)
		if len(l.buckets) >= l.capacity {
			return
		}
	}
	l.take(key, now)
}

// take records a request for key at now. The caller has just checked that
// it is allowed.
func (l *keyedLimiter[K]) take(key K, now time.Time) {
	b := l.buckets[key]
	if l.rate.allow(&b, now) {
		l.buckets[key] = b
	}
}

// prune forgets every key whose bucket is full again.
func (l *keyedLimiter[K]) prune(now time.Time) {
	for k, b := range l.buckets {
		if b.full(now) {
			delete(l.buckets, k)
		}
	}
}

// len returns the number of remembered keys.
func (l *keyedLimiter[K]) len() int { return len(l.buckets) }
