package protocoltest

import (
	"math/rand/v2"
	"strconv"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/view"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// pickChoice returns the key of a uniformly random legal choice. choices is
// always view.View.Choices, the legal set (see view/doc.go); there is no
// "ok" field to filter on, unlike disabled choices.
func pickChoice(rng *rand.Rand, choices []view.Choice) string {
	return choices[rng.IntN(len(choices))].Key
}

// seatRef is a convenience for a function that wants a *engine.Seat
// literal, such as trueHidden.
func seatRef(seat int) *engine.Seat {
	s := engine.Seat(seat)
	return &s
}

// syncTo waits until every connection in conns has seen a state at version
// v or later, so that after a join or an act every client's c.latest
// reflects at least that moment, the way a real client converges after a
// broadcast.
func syncTo(conns []*tclient, v int) {
	for _, c := range conns {
		c.untilState(v)
	}
}

// maxVersion returns the highest Version among the connections' c.latest.
func maxVersion(conns []*tclient) int {
	v := 0
	for _, c := range conns {
		if c.latest.Version > v {
			v = c.latest.Version
		}
	}
	return v
}

// over reports whether c's latest state is a finished game.
func over(c *tclient) bool { return c.latest.View.Result != nil }

// gameResult holds what a full-game driver checked along the way, for a
// test to assert on afterward.
type gameResult struct {
	code  string
	seed  uint64
	final wire.State
}

// playOnline drives a full online game between two distinct users, plus a
// spectator, to completion, checking after every turn that nobody received
// anything that must stay hidden from them. It returns once every
// connection's latest state shows the game over.
func playOnline(t *testing.T, srv *server, rng *rand.Rand) gameResult {
	t.Helper()
	a := dial(t, srv, "A")
	a.hello()
	a.create(wire.ModeOnline, nil, wire.SeatRandom)
	code := a.code

	b := dial(t, srv, "B")
	b.hello()
	b.join(code)

	spec := dial(t, srv, "Spectator")
	spec.hello()
	spec.join(code)

	conns := []*tclient{a, b, spec}
	syncTo(conns, maxVersion(conns))

	if len(a.latest.You.Seats) != 1 || len(b.latest.You.Seats) != 1 {
		t.Fatalf("online: want exactly one seat each for A and B, got A=%v B=%v", a.latest.You.Seats, b.latest.You.Seats)
	}
	bySeat := map[int]*tclient{a.latest.You.Seats[0]: a, b.latest.You.Seats[0]: b}
	if len(bySeat) != 2 {
		t.Fatalf("online: A and B hold the same seat %v", a.latest.You.Seats)
	}

	seed := srv.snapshot(code).State.Seed
	lt := newLeakTracker(t, seed)
	checkAll := func() {
		snap := srv.snapshot(code)
		var viewerSpec *engine.Seat // spectator: always nil
		lt.check(spec, "spectator", trueHidden(&snap.State, viewerSpec))
		for seat, c := range bySeat {
			var viewer *engine.Seat
			if !snap.State.Over() {
				viewer = seatRef(seat)
			}
			lt.check(c, "seat"+strconv.Itoa(seat), trueHidden(&snap.State, viewer))
		}
	}
	checkAll()

	for !over(a) {
		toMove := *a.latest.View.ToMove
		actor, ok := bySeat[toMove]
		if !ok {
			t.Fatalf("online: no connection holds seat %d", toMove)
		}
		choices := actor.latest.View.Choices
		if len(choices) == 0 {
			t.Fatalf("online: seat %d to move with no legal choices", toMove)
		}
		key := pickChoice(rng, choices)
		seq := actor.act(key, nil)
		actor.untilAck(seq)
		syncTo(conns, maxVersion(conns))
		checkAll()
	}
	return gameResult{code: code, seed: seed, final: a.latest}
}

// playHotseat drives a full hot-seat game, one connection holding both
// seats, to completion.
func playHotseat(t *testing.T, srv *server, rng *rand.Rand) gameResult {
	t.Helper()
	c := dial(t, srv, "Hotseat")
	c.hello()
	c.create(wire.ModeHotseat, nil, wire.SeatNumber(0))
	code := c.code

	seed := srv.snapshot(code).State.Seed
	lt := newLeakTracker(t, seed)
	check := func() {
		snap := srv.snapshot(code)
		var viewer *engine.Seat
		if !snap.State.Over() {
			viewer = seatRef(*c.latest.View.ToMove)
		}
		lt.check(c, "hotseat", trueHidden(&snap.State, viewer))
	}
	check()

	for !over(c) {
		toMove := *c.latest.View.ToMove
		choices := c.latest.View.Choices
		if len(choices) == 0 {
			t.Fatalf("hotseat: seat %d to move with no legal choices", toMove)
		}
		key := pickChoice(rng, choices)
		seq := c.act(key, seatInt(toMove))
		c.untilAck(seq)
		check()
	}
	return gameResult{code: code, seed: seed, final: c.latest}
}

// playAI drives a full game against a bot: the human holds seat 0, a bot
// (easy level, tinyBudget) holds seat 1.
func playAI(t *testing.T, srv *server, rng *rand.Rand) gameResult {
	t.Helper()
	c := dial(t, srv, "AI-human")
	c.hello()
	bot := wire.BotLevelEasy
	c.create(wire.ModeAI, &bot, wire.SeatNumber(0))
	code := c.code
	const mySeat = 0

	seed := srv.snapshot(code).State.Seed
	lt := newLeakTracker(t, seed)
	check := func() {
		snap := srv.snapshot(code)
		var viewer *engine.Seat
		if !snap.State.Over() {
			viewer = seatRef(mySeat)
		}
		lt.check(c, "ai-human", trueHidden(&snap.State, viewer))
	}
	check()

	for !over(c) {
		if c.latest.You.CanAct {
			choices := c.latest.View.Choices
			if len(choices) == 0 {
				t.Fatalf("ai: can act but no legal choices")
			}
			key := pickChoice(rng, choices)
			seq := c.act(key, nil)
			c.untilAck(seq)
		} else {
			c.untilState(c.latest.Version + 1)
		}
		check()
	}
	return gameResult{code: code, seed: seed, final: c.latest}
}
