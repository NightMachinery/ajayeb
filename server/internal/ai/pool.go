package ai

import (
	"context"
	"runtime"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// Pool caps how many searches run at once across the whole server, so that
// many bot rooms thinking together cannot take every core (or, with the tree
// search's arenas, too much memory). It is a counting semaphore: a search
// takes one slot for its whole duration, whatever its worker count. The zero
// Pool is not usable; make one with NewPool. A Pool is safe for concurrent
// use.
type Pool struct {
	sem chan struct{}
}

// DefaultPoolSize is the pool size for a server: half the CPUs, at least
// one.
func DefaultPoolSize() int { return max(1, runtime.NumCPU()/2) }

// NewPool returns a pool of n slots; n <= 0 means DefaultPoolSize.
func NewPool(n int) *Pool {
	if n <= 0 {
		n = DefaultPoolSize()
	}
	return &Pool{sem: make(chan struct{}, n)}
}

// Size returns the number of slots.
func (p *Pool) Size() int { return cap(p.sem) }

// InUse returns the number of slots taken right now.
func (p *Pool) InUse() int { return len(p.sem) }

// Acquire takes a slot, waiting until one is free. It returns ctx's error,
// without a slot, if ctx is done first.
func (p *Pool) Acquire(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case p.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Release frees a slot taken by Acquire. It panics if no slot is taken.
func (p *Pool) Release() {
	select {
	case <-p.sem:
	default:
		panic("ai: Pool.Release without Acquire")
	}
}

// Choose runs bot.Choose inside a slot: it waits for a free slot, searches,
// and frees the slot. If ctx is done before a slot is free, it returns ctx's
// error and no action.
func (p *Pool) Choose(ctx context.Context, bot Bot, obs engine.Observation, b Budget, onProgress func(Progress)) (engine.Action, error) {
	if err := p.Acquire(ctx); err != nil {
		return engine.Action{}, err
	}
	defer p.Release()
	return bot.Choose(ctx, obs, b, onProgress)
}
