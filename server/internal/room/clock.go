package room

import "time"

// Clock is the room package's only source of time: the bot's think delay and
// the idle eviction of rooms both go through it, so tests drive them with a
// fake clock (see package roomtest) instead of sleeping.
type Clock interface {
	// Now returns the current time.
	Now() time.Time
	// AfterFunc calls f, in a goroutine of the clock's choosing, once d has
	// passed, and returns a function that cancels the call. The cancel
	// function reports whether it stopped the call before it ran, like
	// time.Timer.Stop.
	AfterFunc(d time.Duration, f func()) (stop func() bool)
}

// SystemClock is the wall clock: time.Now and time.AfterFunc.
type SystemClock struct{}

// Now returns time.Now().
func (SystemClock) Now() time.Time { return time.Now() }

// AfterFunc is time.AfterFunc.
func (SystemClock) AfterFunc(d time.Duration, f func()) func() bool {
	return time.AfterFunc(d, f).Stop
}
