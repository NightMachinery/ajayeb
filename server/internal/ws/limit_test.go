package ws

import (
	"fmt"
	"net/netip"
	"testing"
	"time"
)

var limitEpoch = time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)

// limitCount allows requests at now until one is refused and returns how
// many were allowed (at most limit+1 tries).
func limitCount(r rate, b *bucket, now time.Time, limit int) int {
	n := 0
	for range limit + 1 {
		if !r.allow(b, now) {
			break
		}
		n++
	}
	return n
}

func TestFrameRateBurstAndRefill(t *testing.T) {
	r := newRate(DefaultFrameRate, time.Second, DefaultFrameBurst)
	var b bucket
	now := limitEpoch

	if got := limitCount(r, &b, now, 100); got != 20 {
		t.Fatalf("a fresh bucket allowed %d frames at once, want the burst of 20", got)
	}
	if r.allow(&b, now) {
		t.Fatal("the 21st frame at the same instant was allowed")
	}
	// A refused frame costs nothing: still exactly one frame per 100 ms.
	now = now.Add(99 * time.Millisecond)
	if r.allow(&b, now) {
		t.Fatal("a frame 99 ms later was allowed, before one was earned back")
	}
	now = now.Add(time.Millisecond)
	if !r.allow(&b, now) {
		t.Fatal("a frame 100 ms later was refused")
	}
	if r.allow(&b, now) {
		t.Fatal("two frames were allowed for one 100 ms refill")
	}
	// Ten per second, sustained.
	allowed := 0
	for range 50 {
		now = now.Add(100 * time.Millisecond)
		if r.allow(&b, now) {
			allowed++
		}
	}
	if allowed != 50 {
		t.Fatalf("at exactly 10 frames per second, %d of 50 were allowed", allowed)
	}
	// After 2 s of silence the whole burst is back, and no more.
	now = now.Add(10 * time.Second)
	if !b.full(now) {
		t.Fatal("the bucket is not full after 10 s of silence")
	}
	if got := limitCount(r, &b, now, 100); got != 20 {
		t.Fatalf("after a long pause %d frames were allowed, want 20", got)
	}
}

func TestCreateRateIsTwentyPerHour(t *testing.T) {
	r := newRate(DefaultCreatesPerUser, time.Hour, DefaultCreatesPerUser)
	var b bucket
	now := limitEpoch
	if got := limitCount(r, &b, now, 100); got != 20 {
		t.Fatalf("allowed %d creates at once, want 20", got)
	}
	now = now.Add(3*time.Minute - time.Second)
	if r.allow(&b, now) {
		t.Fatal("a create was allowed before 3 minutes had passed")
	}
	now = now.Add(time.Second)
	if !r.allow(&b, now) {
		t.Fatal("a create was refused after 3 minutes (one per 60/20 minutes)")
	}
}

func TestKeyedLimiterKeysAreIndependent(t *testing.T) {
	l := newKeyedLimiter[string](newRate(2, time.Hour, 2), 16)
	now := limitEpoch
	for i := range 2 {
		if !l.check("a", now) {
			t.Fatalf("a's create %d was refused", i+1)
		}
		l.take("a", now)
	}
	if l.check("a", now) {
		t.Fatal("a's third create was allowed")
	}
	if !l.check("b", now) {
		t.Fatal("b was refused because of a")
	}
	// check alone uses nothing up.
	for range 5 {
		if !l.check("b", now) {
			t.Fatal("check used up b's allowance")
		}
	}
}

func TestKeyedLimiterIsBounded(t *testing.T) {
	const capacity = 8
	l := newKeyedLimiter[netip.Addr](newRate(1, time.Hour, 1), capacity)
	now := limitEpoch
	addr := func(i int) netip.Addr { return netip.MustParseAddr(fmt.Sprintf("192.0.2.%d", i)) }

	for i := range capacity {
		if !l.check(addr(i), now) {
			t.Fatalf("key %d refused below capacity", i)
		}
		l.take(addr(i), now)
	}
	if l.len() != capacity {
		t.Fatalf("remembers %d keys, want %d", l.len(), capacity)
	}
	// Every remembered key is still limited: a newcomer is refused rather
	// than evicting one (which would reset that key's limit).
	if l.check(addr(100), now) {
		t.Fatal("a new key was allowed while every remembered key is limited")
	}
	if l.len() != capacity {
		t.Fatalf("remembers %d keys after a refusal, want %d", l.len(), capacity)
	}
	// Once the buckets have refilled they are forgotten to make room.
	now = now.Add(time.Hour)
	if !l.check(addr(100), now) {
		t.Fatal("a new key was refused after the old ones refilled")
	}
	l.take(addr(100), now)
	if l.len() != 1 {
		t.Fatalf("remembers %d keys after pruning, want 1", l.len())
	}
}

func TestLimitsDefaults(t *testing.T) {
	got := Limits{}.withDefaults()
	want := Limits{
		HelloTimeout:   10 * time.Second,
		FrameRate:      10,
		FrameBurst:     20,
		MaxRateRejects: 20,
		MaxConns:       1024,
		MaxConnsPerIP:  16,
		CreatesPerUser: 20,
		CreatesPerIP:   60,
		LimiterKeys:    4096,
		PingInterval:   25 * time.Second,
		MaxMissedPongs: 2,
		WriteTimeout:   10 * time.Second,
	}
	if got != want {
		t.Errorf("defaults = %+v\nwant %+v", got, want)
	}
	custom := Limits{FrameRate: 3, PingInterval: time.Second}.withDefaults()
	if custom.FrameRate != 3 || custom.PingInterval != time.Second || custom.FrameBurst != 20 {
		t.Errorf("explicit limits were not kept: %+v", custom)
	}
}
