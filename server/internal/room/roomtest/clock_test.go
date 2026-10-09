package roomtest

import (
	"testing"
	"time"
)

func TestFakeClock(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewFakeClock(start)
	var order []string
	c.AfterFunc(2*time.Second, func() { order = append(order, "b") })
	c.AfterFunc(time.Second, func() { order = append(order, "a") })
	stop := c.AfterFunc(time.Second, func() { order = append(order, "stopped") })
	if !stop() || stop() {
		t.Fatalf("stop should report true once, then false")
	}
	if c.Pending() != 2 || !c.WaitForTimers(2, time.Second) {
		t.Fatalf("pending %d, want 2", c.Pending())
	}
	c.Advance(999 * time.Millisecond)
	if len(order) != 0 {
		t.Fatalf("timers fired early: %v", order)
	}
	c.Advance(5 * time.Second)
	if len(order) != 2 || order[0] != "a" || order[1] != "b" {
		t.Fatalf("fired %v, want [a b]", order)
	}
	if got := c.Now(); !got.Equal(start.Add(5999 * time.Millisecond)) {
		t.Fatalf("Now %v", got)
	}
	if c.Pending() != 0 || c.WaitForTimers(1, time.Millisecond) {
		t.Fatalf("timers left after firing")
	}
}
