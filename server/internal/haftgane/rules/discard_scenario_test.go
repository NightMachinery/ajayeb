package rules

import (
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// A conserving partial history just after the sixth ordinary turn: Halik's
// new stage opens the discard point, Babylon keeps its other card, and every
// other leftover is already in the pile (rules.md rulings 1, 24 and 25).
func sixthTurnDiscardScenario(t *testing.T) (Game, Seat, Seat) {
	t.Helper()
	g := scenarioNightGame(t)
	var halik, babylon Seat
	for seat, c := range g.State.Cities {
		switch c.Board {
		case catalog.BoardHalikarnassos:
			halik = Seat(seat)
		case catalog.BoardBabylon:
			babylon = Seat(seat)
		}
	}
	g.State.Phase, g.State.Turn, g.State.Decider = PhaseDiscardBuild, TurnsPerAge, halik
	g.State.Points = 7
	pop := func(hand *[HandSize]catalog.DesignID) catalog.DesignID {
		id := hand[0]
		copy(hand[:], hand[1:])
		hand[HandSize-1] = 0
		return id
	}
	for seat := Seat(0); seat < MaxSeats; seat++ {
		hand := &g.State.Hands[seat]
		for turn := 0; turn < 5; turn++ {
			id := pop(hand)
			if seat == babylon && turn == 0 {
				g.State.Cities[seat].Stages = 1
				g.State.Cities[seat].Tucked[0] = id
			} else {
				g.State.Discard[id]++
			}
		}
		id := pop(hand)
		g.Record.Picks[0][5][seat] = id
		if seat == halik {
			g.State.Cities[seat].Stages = 1
			g.State.Cities[seat].Tucked[0] = id
			g.Record.Plays[0][5][seat] = Play{Kind: ActStage, Stage: 1}
		} else {
			g.State.Discard[id]++
			g.Record.Plays[0][5][seat] = Play{Kind: ActSell}
		}
		leftover := pop(hand)
		if seat == babylon {
			g.State.Cities[seat].Kept = leftover
		} else {
			g.State.Discard[leftover]++
		}
		c, err := cityCaches(g.State.Cities[seat])
		if err != nil {
			t.Fatal(err)
		}
		g.State.Cities[seat] = c
	}
	alignScenarioHistory(t, &g)
	return g, halik, babylon
}

func TestScenarioSixthTurnDiscardHandsAndKept(t *testing.T) {
	g, halik, babylon := sixthTurnDiscardScenario(t)
	t.Run("empty-hands-and-kept", func(t *testing.T) {
		if err := g.CheckInvariants(); err != nil {
			t.Fatal(err)
		}
		out, err := NewGameFromScenario(g.State.Seed, g.State.Opts, ScenarioFromGame(g))
		if err != nil || out != g {
			t.Fatal("valid sixth-turn discard scenario rejected", err)
		}
	})
	t.Run("leftover-in-hand", func(t *testing.T) {
		bad := g
		for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
			if bad.State.Discard[id] > 0 {
				bad.State.Discard[id]--
				bad.State.Hands[halik][0] = id
				break
			}
		}
		if err := bad.CheckInvariants(); err == nil || !strings.Contains(err.Error(), "hand seat") {
			t.Fatal("leftover in hand accepted or rejected for another reason", err)
		}
		if _, err := NewGameFromScenario(bad.State.Seed, bad.State.Opts, ScenarioFromGame(bad)); err == nil {
			t.Fatal("builder accepted sixth-turn hand")
		}
	})
	t.Run("missing-babylon-kept", func(t *testing.T) {
		bad := g
		bad.State.Discard[bad.State.Cities[babylon].Kept]++
		bad.State.Cities[babylon].Kept = 0
		if err := bad.CheckInvariants(); err == nil || !strings.Contains(err.Error(), "kept") {
			t.Fatal("missing kept card accepted or rejected for another reason", err)
		}
		if _, err := NewGameFromScenario(bad.State.Seed, bad.State.Opts, ScenarioFromGame(bad)); err == nil {
			t.Fatal("builder accepted missing kept card")
		}
	})
}
