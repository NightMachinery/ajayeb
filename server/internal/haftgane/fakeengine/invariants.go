package fakeengine

import (
	"errors"
	"strconv"
)

// CheckInvariants checks the game value for tests and restore; the engine
// never calls it itself. It checks that every copy is in exactly one place,
// hand sizes by phase, sorted hands, coins, stages, the phase, Age, turn and
// decider, the result, and that the record replays to this very value.
func (g *Game) CheckInvariants() error {
	st := &g.State
	fail := func(msg string) error { return errors.New("fakeengine: invariant: " + msg) }
	if st.Seats < MinSeats || st.Seats > MaxSeats {
		return fail("seat count " + strconv.Itoa(int(st.Seats)))
	}
	if st.Phase < PhaseTurn || st.Phase > PhaseOver {
		return fail("phase " + strconv.Itoa(int(st.Phase)))
	}
	over := st.Phase == PhaseOver
	if st.Age < 1 || st.Age > NumAges || st.Turn < 1 || st.Turn > TurnsPerAge {
		return fail("age " + strconv.Itoa(int(st.Age)) + " turn " + strconv.Itoa(int(st.Turn)))
	}
	if st.Result.Over != over {
		return fail("result set " + strconv.FormatBool(st.Result.Over) + " in phase " + st.Phase.String())
	}
	if g.Record.NLog > MaxPoints || int(st.Points) != int(g.Record.NLog) {
		return fail("points resolved differ from the record")
	}

	// Every copy in exactly one place.
	var count [NumDesigns + 1]int
	add := func(ds []DesignID) {
		for _, d := range ds {
			if d > NumDesigns {
				count[0] = -1000
			} else if d != 0 {
				count[d]++
			}
		}
	}
	for s := Seat(0); uint8(s) < st.Seats; s++ {
		c := &st.Cities[s]
		add(st.Hands[s][:])
		add(st.Deals[s][:])
		add(c.Tucked[:])
		add([]DesignID{c.Kept})
		for d := DesignID(1); d <= NumDesigns; d++ {
			if c.built(d) {
				count[d]++
			}
		}
		if c.Built&^(((1<<(NumDesigns+1))-1)&^1) != 0 {
			return fail("seat " + strconv.Itoa(int(s)) + " built a design outside the catalog")
		}
	}
	add(st.Box[:])
	for d := DesignID(1); d <= NumDesigns; d++ {
		count[d] += int(st.Pile[d])
	}
	if st.Pile[0] != 0 || count[0] != 0 {
		return fail("a card outside the catalog")
	}
	for d := DesignID(1); d <= NumDesigns; d++ {
		if count[d] != int(copiesFor(d, st.Seats)) {
			return fail("design " + d.Slug() + " has " + strconv.Itoa(count[d]) + " copies in play, want " +
				strconv.Itoa(int(copiesFor(d, st.Seats))))
		}
	}

	// Hands, cities and seats beyond the game.
	want := uint8(0)
	switch st.Phase {
	case PhaseTurn:
		want = HandSize + 1 - st.Turn
	case PhaseSalvage: // picks have left, passing has not happened, and the last turn's leftovers are gone
		if st.Turn < TurnsPerAge {
			want = HandSize - st.Turn
		}
	}
	for s := Seat(0); s < MaxSeats; s++ {
		c := &st.Cities[s]
		if uint8(s) >= st.Seats {
			if *c != (City{}) || st.Hands[s] != [HandSize]DesignID{} || st.Deals[s] != [HandSize]DesignID{} {
				return fail("seat " + strconv.Itoa(int(s)) + " is beyond the game but not empty")
			}
			continue
		}
		h := st.Hands[s]
		if handCount(&h) != want {
			return fail("seat " + strconv.Itoa(int(s)) + " holds " + strconv.Itoa(int(handCount(&h))) + " cards in phase " + st.Phase.String())
		}
		sorted := h
		sortHand(&sorted)
		if sorted != h {
			return fail("seat " + strconv.Itoa(int(s)) + "'s hand is not sorted")
		}
		if c.Coins < 0 {
			return fail("seat " + strconv.Itoa(int(s)) + " has negative coins")
		}
		if c.Stages > MaxStages {
			return fail("seat " + strconv.Itoa(int(s)) + " has too many stages")
		}
		for i := uint8(0); i < MaxStages; i++ {
			if (i < c.Stages) != (c.Tucked[i] != 0) {
				return fail("seat " + strconv.Itoa(int(s)) + "'s tucked cards disagree with its stages")
			}
		}
		if c.Kept != 0 && (c.Board != BoardOrchard || (st.Phase != PhaseLastCard && st.Phase != PhaseSalvage)) {
			return fail("seat " + strconv.Itoa(int(s)) + " keeps a card outside its last-card point")
		}
		if (st.Age == 1) != (st.Deals[s] != [HandSize]DesignID{}) {
			return fail("Age 2's deal is present in Age 2 or missing in Age 1")
		}
	}
	switch st.Phase {
	case PhaseSalvage:
		if uint8(st.Decider) >= st.Seats || st.Cities[st.Decider].Board != BoardArchive || !st.salvageable(st.Decider) {
			return fail("a salvage point for a seat that cannot salvage")
		}
	case PhaseLastCard:
		if uint8(st.Decider) >= st.Seats || st.Cities[st.Decider].Board != BoardOrchard ||
			st.Cities[st.Decider].Kept == 0 || st.Turn != TurnsPerAge {
			return fail("a last-card point without a kept card")
		}
	default:
		if st.Decider != 0 {
			return fail("a decider outside a one-seat point")
		}
	}

	// The record replays to this value.
	r, err := NewGame(g.setup())
	if err != nil {
		return fail("the setup is invalid")
	}
	for k := uint8(0); k < g.Record.NLog; k++ {
		e := &g.Record.Log[k]
		p := r.Point()
		if p.Kind != e.Kind || p.Required != e.Required || r.State.Age != e.Age || r.State.Turn != e.Turn {
			return fail("record entry " + strconv.Itoa(int(k)) + " is not the point it resolved")
		}
		for _, s := range p.Required.Seats() {
			if _, ok := r.legalAction(s, e.Actions[s].Key()); !ok {
				return fail("record entry " + strconv.Itoa(int(k)) + " holds an action that was not legal")
			}
		}
		acts := e.Actions
		r.apply(p, &acts, nil)
	}
	if r != *g {
		return fail("the record does not replay to this game")
	}
	return nil
}
