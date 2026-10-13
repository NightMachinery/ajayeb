package ai

import (
	"context"
	"runtime"
	"slices"
	"sync"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// Pool caps how many searches run at once across the whole server, so that
// many bot rooms thinking together cannot take every core (or, with the tree
// search's arenas, too much memory). A search takes one slot for its whole
// duration, whatever its worker count. The zero Pool is not usable; make one
// with NewPool or NewPoolPerKey. A Pool is safe for concurrent use.
//
// The slots are shared fairly between keys (the server's key is the
// player's network address, as its per-address limits count it). A key
// holds at most PerKey slots at once, and its further searches wait without
// holding anything. When a slot frees up it goes to a waiting search whose
// key holds the fewest slots; among those, to the key that was last given a
// slot longest ago (a key that has not had one since it was last idle
// counts as never), and within a key to its earliest search. So however
// many searches one address starts, a search of another address waits for
// at most one slot to free up, never for that address's whole queue. The
// empty key, for a search no address is known for, has no PerKey cap; it
// still yields to keys that hold fewer slots.
type Pool struct {
	size   int
	perKey int

	mu      sync.Mutex
	inUse   int
	held    map[string]int    // slots held per key; a key holding none is absent
	waiting []*poolWaiter     // in arrival order
	granted map[string]uint64 // when each busy key last got a slot (a count of grants)
	grants  uint64            // the number of slots granted so far
}

// poolWaiter is a search waiting for a slot. ready is closed when it is
// granted one; granted is set (under the Pool's mutex) at the same moment.
type poolWaiter struct {
	key     string
	ready   chan struct{}
	granted bool
}

// DefaultPoolSize is the pool size for a server: half the CPUs, at least
// one.
func DefaultPoolSize() int { return max(1, runtime.NumCPU()/2) }

// DefaultPerKey is how many slots one key may hold at once by default.
const DefaultPerKey = 2

// NewPool returns a pool of n slots, at most DefaultPerKey of them held by
// one key; n <= 0 means DefaultPoolSize.
func NewPool(n int) *Pool { return NewPoolPerKey(n, DefaultPerKey) }

// NewPoolPerKey returns a pool of n slots, at most perKey of them held by
// one key at once; n <= 0 means DefaultPoolSize and perKey <= 0 means
// DefaultPerKey.
func NewPoolPerKey(n, perKey int) *Pool {
	if n <= 0 {
		n = DefaultPoolSize()
	}
	if perKey <= 0 {
		perKey = DefaultPerKey
	}
	return &Pool{size: n, perKey: perKey, held: make(map[string]int), granted: make(map[string]uint64)}
}

// Size returns the number of slots.
func (p *Pool) Size() int { return p.size }

// PerKey returns how many slots one key may hold at once.
func (p *Pool) PerKey() int { return p.perKey }

// InUse returns the number of slots taken right now.
func (p *Pool) InUse() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.inUse
}

// Waiting returns the number of searches waiting for a slot right now.
func (p *Pool) Waiting() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.waiting)
}

// Acquire is AcquireFor with the empty key.
func (p *Pool) Acquire(ctx context.Context) error { return p.AcquireFor(ctx, "") }

// Release is ReleaseFor with the empty key.
func (p *Pool) Release() { p.ReleaseFor("") }

// AcquireFor takes a slot for key, waiting until one is free, key holds
// fewer than PerKey slots, and its turn has come (see Pool). It returns
// ctx's error, without a slot, if ctx is done first; the search then leaves
// the queue at once.
func (p *Pool) AcquireFor(ctx context.Context, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p.mu.Lock()
	// A free slot with searches still waiting means none of them may take
	// it (their keys are at the cap), so a newcomer that may takes it.
	if p.inUse < p.size && p.mayHold(key) {
		p.take(key)
		p.mu.Unlock()
		return nil
	}
	w := &poolWaiter{key: key, ready: make(chan struct{})}
	p.waiting = append(p.waiting, w)
	p.mu.Unlock()

	select {
	case <-w.ready:
		return nil
	case <-ctx.Done():
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if w.granted {
		// The slot came first: the caller has it, and its search will see
		// the done context soon enough.
		return nil
	}
	if i := slices.Index(p.waiting, w); i >= 0 {
		p.waiting = slices.Delete(p.waiting, i, i+1)
	}
	p.forgetIfIdle(key)
	return ctx.Err()
}

// ReleaseFor frees a slot that AcquireFor took for key, and hands it to the
// next waiting search. It panics if key holds no slot.
func (p *Pool) ReleaseFor(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.held[key] == 0 {
		panic("ai: Pool.Release without Acquire")
	}
	p.inUse--
	if p.held[key]--; p.held[key] == 0 {
		delete(p.held, key)
	}
	p.dispatch()
	p.forgetIfIdle(key)
}

// mayHold reports whether key may take one more slot. The caller holds mu.
func (p *Pool) mayHold(key string) bool { return key == "" || p.held[key] < p.perKey }

// take gives key a slot. The caller holds mu and has checked there is one.
func (p *Pool) take(key string) {
	p.inUse++
	p.held[key]++
	p.grants++
	p.granted[key] = p.grants
}

// forgetIfIdle drops key's record once it holds no slot and has no search
// waiting, so the Pool remembers only the keys that are busy. The caller
// holds mu.
func (p *Pool) forgetIfIdle(key string) {
	if p.held[key] > 0 || slices.ContainsFunc(p.waiting, func(w *poolWaiter) bool { return w.key == key }) {
		return
	}
	delete(p.granted, key)
}

// before reports whether a waiting search of key a goes before one of key
// b: a holds fewer slots, or as many but got its last one earlier (a key
// with no record never got one). The caller holds mu.
func (p *Pool) before(a, b string) bool {
	if p.held[a] != p.held[b] {
		return p.held[a] < p.held[b]
	}
	return p.granted[a] < p.granted[b]
}

// dispatch grants free slots to waiting searches, in the order the Pool
// describes. The caller holds mu.
func (p *Pool) dispatch() {
	for p.inUse < p.size {
		best := -1
		for i, w := range p.waiting {
			if !p.mayHold(w.key) {
				continue
			}
			if best < 0 || p.before(w.key, p.waiting[best].key) {
				best = i
			}
		}
		if best < 0 {
			return
		}
		w := p.waiting[best]
		p.waiting = slices.Delete(p.waiting, best, best+1)
		p.take(w.key)
		w.granted = true
		close(w.ready)
	}
}

// Choose is ChooseFor with the empty key.
func (p *Pool) Choose(ctx context.Context, bot Bot, obs engine.Observation, b Budget, onProgress func(Progress)) (engine.Action, error) {
	return p.ChooseFor(ctx, "", bot, obs, b, onProgress)
}

// ChooseFor runs bot.Choose inside a slot taken for key: it waits for its
// turn, searches, and frees the slot. If ctx is done before it has a slot,
// it returns ctx's error and no action.
func (p *Pool) ChooseFor(ctx context.Context, key string, bot Bot, obs engine.Observation, b Budget, onProgress func(Progress)) (engine.Action, error) {
	if err := p.AcquireFor(ctx, key); err != nil {
		return engine.Action{}, err
	}
	defer p.ReleaseFor(key)
	return bot.Choose(ctx, obs, b, onProgress)
}
