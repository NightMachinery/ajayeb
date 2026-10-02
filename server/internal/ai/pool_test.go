package ai_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// The tests of pool.go.

// poolBlockingBot counts the searches running inside it and blocks each one
// until release is closed.
type poolBlockingBot struct {
	running, peak atomic.Int32
	release       chan struct{}
}

func (b *poolBlockingBot) Choose(ctx context.Context, obs engine.Observation, _ ai.Budget, _ func(ai.Progress)) (engine.Action, error) {
	n := b.running.Add(1)
	for {
		p := b.peak.Load()
		if n <= p || b.peak.CompareAndSwap(p, n) {
			break
		}
	}
	defer b.running.Add(-1)
	select {
	case <-b.release:
	case <-ctx.Done():
	}
	return engine.LegalActions(obs.Public(), nil)[0], nil
}

func TestPoolCapsConcurrentSearches(t *testing.T) {
	const size, searches = 2, 6
	p := ai.NewPool(size)
	if p.Size() != size {
		t.Fatalf("Size = %d", p.Size())
	}
	bot := &poolBlockingBot{release: make(chan struct{})}
	s := engine.NewGame(1, engine.Options{})
	obs := engine.Observe(&s, s.ToMove)
	var wg sync.WaitGroup
	errs := make(chan error, searches)
	for range searches {
		wg.Go(func() {
			if _, err := p.Choose(t.Context(), bot, obs, ai.Budget{}, nil); err != nil {
				errs <- err
			}
		})
	}
	deadline := time.Now().Add(5 * time.Second)
	for p.InUse() < size || bot.running.Load() < size {
		if time.Now().After(deadline) {
			t.Fatalf("the pool never filled: in use %d, running %d", p.InUse(), bot.running.Load())
		}
		time.Sleep(time.Millisecond)
	}
	time.Sleep(20 * time.Millisecond) // give a wrongly admitted search time to start
	if got := bot.running.Load(); got != size {
		t.Fatalf("%d searches run at once in a pool of %d", got, size)
	}
	close(bot.release)
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
	if bot.peak.Load() > size {
		t.Errorf("peak %d searches in a pool of %d", bot.peak.Load(), size)
	}
	if p.InUse() != 0 {
		t.Errorf("%d slots still taken", p.InUse())
	}
}

func TestPoolAcquireHonoursTheContext(t *testing.T) {
	p := ai.NewPool(1)
	if err := p.Acquire(t.Context()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()
	if err := p.Acquire(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Acquire on a full pool: %v", err)
	}
	s := engine.NewGame(1, engine.Options{})
	if _, err := p.Choose(ctx, ai.NewRandom(1), engine.Observe(&s, s.ToMove), ai.Budget{}, nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Choose on a full pool with a done context: %v", err)
	}
	p.Release()
	if err := p.Acquire(t.Context()); err != nil {
		t.Fatalf("Acquire after Release: %v", err)
	}
	p.Release()
}

func TestPoolReleaseWithoutAcquirePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("Release on an empty pool did not panic")
		}
	}()
	ai.NewPool(1).Release()
}

func TestNewPoolDefaultSize(t *testing.T) {
	if n := ai.NewPool(0).Size(); n != ai.DefaultPoolSize() || n < 1 {
		t.Fatalf("NewPool(0).Size() = %d, DefaultPoolSize() = %d", n, ai.DefaultPoolSize())
	}
}
