// Package roomtest holds test helpers for code that drives rooms: a fake
// clock that satisfies room.Clock, so tests move time forward by hand
// instead of sleeping.
package roomtest

import (
	"slices"
	"sync"
	"time"
)

// FakeClock is a manual clock. Time stands still until Advance moves it;
// Advance then runs, in the calling goroutine and in time order, every
// AfterFunc callback that has come due. It satisfies room.Clock and is safe
// for concurrent use.
type FakeClock struct {
	mu      sync.Mutex
	now     time.Time
	timers  []*fakeTimer
	changed chan struct{} // closed and replaced whenever a timer is added
}

type fakeTimer struct {
	at   time.Time
	f    func()
	done bool // fired or stopped
}

// NewFakeClock returns a clock that reads start.
func NewFakeClock(start time.Time) *FakeClock {
	return &FakeClock{now: start, changed: make(chan struct{})}
}

// Now returns the clock's current time.
func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// AfterFunc schedules f to run when Advance moves the clock d or more past
// the current time, and returns its cancel function.
func (c *FakeClock) AfterFunc(d time.Duration, f func()) func() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &fakeTimer{at: c.now.Add(d), f: f}
	c.timers = append(c.timers, t)
	close(c.changed)
	c.changed = make(chan struct{})
	return func() bool {
		c.mu.Lock()
		defer c.mu.Unlock()
		if t.done {
			return false
		}
		t.done = true
		c.prune()
		return true
	}
}

// prune drops finished timers. The caller holds mu.
func (c *FakeClock) prune() {
	c.timers = slices.DeleteFunc(c.timers, func(t *fakeTimer) bool { return t.done })
}

// Advance moves the clock forward by d and runs the callbacks that came
// due, earliest first.
func (c *FakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	var due []*fakeTimer
	for _, t := range c.timers {
		if !t.done && !t.at.After(c.now) {
			t.done = true
			due = append(due, t)
		}
	}
	c.prune()
	c.mu.Unlock()
	slices.SortStableFunc(due, func(a, b *fakeTimer) int { return a.at.Compare(b.at) })
	for _, t := range due {
		t.f()
	}
}

// Pending returns the number of timers that have neither fired nor been
// stopped.
func (c *FakeClock) Pending() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.timers)
}

// WaitForTimers blocks until at least n timers are pending, or until
// timeout (real time, a safety net against a hung test) has passed, and
// reports whether n were reached. Use it before Advance when the code under
// test arms its timer from another goroutine.
func (c *FakeClock) WaitForTimers(n int, timeout time.Duration) bool {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	for {
		c.mu.Lock()
		if len(c.timers) >= n {
			c.mu.Unlock()
			return true
		}
		ch := c.changed
		c.mu.Unlock()
		select {
		case <-ch:
		case <-deadline.C:
			return false
		}
	}
}
