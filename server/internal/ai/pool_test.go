package ai_test

import (
	"context"
	"errors"
	"fmt"
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

// acquireAsync starts AcquireFor(key) in a goroutine and returns a channel
// that receives its result once it returns.
func acquireAsync(ctx context.Context, p *ai.Pool, key string) <-chan error {
	done := make(chan error, 1)
	go func() { done <- p.AcquireFor(ctx, key) }()
	return done
}

func waitWaiting(t *testing.T, p *ai.Pool, n int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for p.Waiting() != n {
		if time.Now().After(deadline) {
			t.Fatalf("waiting = %d, want %d", p.Waiting(), n)
		}
		time.Sleep(time.Millisecond)
	}
}

func expectGranted(t *testing.T, done <-chan error, what string) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("%s: %v", what, err)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("%s: never got a slot", what)
	}
}

func expectWaiting(t *testing.T, done <-chan error, what string) {
	t.Helper()
	select {
	case err := <-done:
		t.Fatalf("%s got a slot (%v), want it to wait", what, err)
	case <-time.After(20 * time.Millisecond):
	}
}

// Eight rooms of one address cannot delay another address's search by more
// than one search: it gets the very next slot that frees up, ahead of the
// first address's queue, whatever the pool size.
func TestPoolOtherKeyWaitsForAtMostOneSearch(t *testing.T) {
	for _, size := range []int{1, 2} {
		p := ai.NewPoolPerKey(size, 2)
		held := min(size, p.PerKey())
		for range held {
			if err := p.AcquireFor(t.Context(), "A"); err != nil {
				t.Fatal(err)
			}
		}
		// The rest of A's eight searches queue up; each, once it has a slot,
		// reports it and holds it until drain closes.
		var wg sync.WaitGroup
		got := make(chan struct{}, 8)
		drain := make(chan struct{})
		for range 8 - held {
			wg.Go(func() {
				if err := p.AcquireFor(t.Context(), "A"); err != nil {
					t.Error(err)
					return
				}
				got <- struct{}{}
				<-drain
				p.ReleaseFor("A")
			})
		}
		waitWaiting(t, p, 8-held)
		other := acquireAsync(t.Context(), p, "B")
		waitWaiting(t, p, 9-held)
		expectWaiting(t, other, "B on a pool A fills")

		p.ReleaseFor("A") // one search of A ends
		expectGranted(t, other, fmt.Sprintf("size %d: B after one release", size))
		if len(got) != 0 || p.Waiting() != 8-held {
			t.Fatalf("size %d: a queued search of A took a slot before B: waiting %d", size, p.Waiting())
		}

		close(drain)
		p.ReleaseFor("B")
		for range held - 1 {
			p.ReleaseFor("A")
		}
		wg.Wait()
		if p.InUse() != 0 || p.Waiting() != 0 || len(got) != 8-held {
			t.Fatalf("size %d: in use %d, waiting %d, %d of A's queue ran", size, p.InUse(), p.Waiting(), len(got))
		}
	}
}

// One key holds at most PerKey slots, even with slots free; its further
// searches wait without blocking another key, and get a slot when one of
// its own ends.
func TestPoolPerKeyCap(t *testing.T) {
	p := ai.NewPoolPerKey(4, 2)
	for range 2 {
		if err := p.AcquireFor(t.Context(), "A"); err != nil {
			t.Fatal(err)
		}
	}
	third := acquireAsync(t.Context(), p, "A")
	waitWaiting(t, p, 1)
	expectWaiting(t, third, "A's third search")
	if err := p.AcquireFor(t.Context(), "B"); err != nil {
		t.Fatal(err)
	}
	if p.InUse() != 3 || p.Waiting() != 1 {
		t.Fatalf("in use %d, waiting %d, want 3 and 1", p.InUse(), p.Waiting())
	}
	p.ReleaseFor("B")
	expectWaiting(t, third, "A's third search after B's ended")
	p.ReleaseFor("A")
	expectGranted(t, third, "A's third search after one of A's ended")
	p.ReleaseFor("A")
	p.ReleaseFor("A")
	if p.InUse() != 0 || p.Waiting() != 0 {
		t.Fatalf("in use %d, waiting %d at the end", p.InUse(), p.Waiting())
	}
}

// A cancelled waiter leaves the queue and never takes a slot, and the pool
// keeps handing slots out afterwards (a room evicted, or its search gone
// stale, while it waited).
func TestPoolCancelledWaiterLeavesTheQueue(t *testing.T) {
	p := ai.NewPoolPerKey(1, 2)
	if err := p.AcquireFor(t.Context(), "A"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	gone := acquireAsync(ctx, p, "B")
	later := acquireAsync(t.Context(), p, "C")
	waitWaiting(t, p, 2)
	cancel()
	select {
	case err := <-gone:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled AcquireFor = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the cancelled AcquireFor did not return")
	}
	waitWaiting(t, p, 1)
	p.ReleaseFor("A")
	expectGranted(t, later, "C after the cancelled B")
	p.ReleaseFor("C")
	if p.InUse() != 0 || p.Waiting() != 0 {
		t.Fatalf("in use %d, waiting %d at the end", p.InUse(), p.Waiting())
	}
	defer func() {
		if recover() == nil {
			t.Error("releasing the cancelled B's slot did not panic: it had one")
		}
	}()
	p.ReleaseFor("B")
}

// Many keys acquiring and releasing at once never exceed the pool size or
// a key's cap, and every search gets its turn (run it with -race).
func TestPoolStress(t *testing.T) {
	const size, perKey, keys, each = 3, 2, 5, 40
	p := ai.NewPoolPerKey(size, perKey)
	var total atomic.Int32
	var per [keys]atomic.Int32
	var wg sync.WaitGroup
	errs := make(chan string, keys*each)
	for k := range keys {
		key := fmt.Sprint("k", k)
		for i := range each {
			wg.Go(func() {
				ctx := t.Context()
				if i%7 == 0 {
					var cancel context.CancelFunc
					ctx, cancel = context.WithTimeout(ctx, time.Duration(i%3)*time.Millisecond)
					defer cancel()
				}
				if err := p.AcquireFor(ctx, key); err != nil {
					return
				}
				if n := total.Add(1); n > size {
					errs <- fmt.Sprintf("%d searches in a pool of %d", n, size)
				}
				if n := per[k].Add(1); n > perKey {
					errs <- fmt.Sprintf("key %s holds %d slots, cap %d", key, n, perKey)
				}
				time.Sleep(50 * time.Microsecond)
				per[k].Add(-1)
				total.Add(-1)
				p.ReleaseFor(key)
			})
		}
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Error(e)
	}
	if p.InUse() != 0 || p.Waiting() != 0 {
		t.Fatalf("in use %d, waiting %d at the end", p.InUse(), p.Waiting())
	}
}

func TestNewPoolPerKeyDefaults(t *testing.T) {
	if p := ai.NewPool(3); p.PerKey() != ai.DefaultPerKey {
		t.Errorf("NewPool PerKey = %d, want %d", p.PerKey(), ai.DefaultPerKey)
	}
	if p := ai.NewPoolPerKey(0, 0); p.Size() != ai.DefaultPoolSize() || p.PerKey() != ai.DefaultPerKey {
		t.Errorf("NewPoolPerKey(0, 0) = size %d, per key %d", p.Size(), p.PerKey())
	}
}
