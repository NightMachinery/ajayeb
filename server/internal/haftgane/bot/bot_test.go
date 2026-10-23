package bot

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

func testObservation(age, turn uint8) rules.Observation {
	o := rules.Observation{Now: rules.Current{Seat: 0, Seats: 3, Age: age, Turn: turn, Phase: rules.PhaseTurn, Decider: rules.NoSeat}}
	for s := 0; s < 3; s++ {
		o.Now.Cities[s] = rules.PublicCity{Board: catalog.BoardID(s + 1), Side: catalog.Day, Coins: 12}
	}
	return o
}
func testBuild(id catalog.DesignID, left, right uint8) rules.Choice {
	return rules.Choice{Action: rules.Action{Kind: rules.ActBuild, Design: id, Left: left, Right: right}, Total: uint16(left) + uint16(right)}
}
func testPick(t *testing.T, o rules.Observation, cs []rules.Choice) Decision {
	t.Helper()
	d, err := Choose(context.Background(), Greedy, o, cs, 71, DefaultWork())
	if err != nil || !d.Complete {
		t.Fatal(d, err)
	}
	return d
}

// Public-observation tactics isolate a portfolio or threshold. Offered costs
// are supplied facts, not recalculated by the policy. Full real-game sweeps
// separately prove that all returned keys belong to the engine's real list.
func TestGreedyScienceAndMilitary(t *testing.T) {
	t.Run("science set", func(t *testing.T) {
		o := testObservation(3, 6)
		o.Now.Cities[0].Buildings = catalog.DesignSetOf(25, 26)
		cs := []rules.Choice{testBuild(68, 0, 0), testBuild(60, 0, 0)}
		if d := testPick(t, o, cs); d.Key != cs[0].Key() {
			t.Fatal("complete set before six blue points", d)
		}
	})
	t.Run("two conflict thresholds", func(t *testing.T) {
		o := testObservation(3, 6)
		o.Now.Cities[0].Buildings = catalog.DesignSetOf(16)
		o.Now.Cities[1].Buildings = catalog.DesignSetOf(14)
		o.Now.Cities[2].Buildings = catalog.DesignSetOf(15)
		cs := []rules.Choice{testBuild(51, 0, 0), testBuild(60, 0, 0)}
		if d := testPick(t, o, cs); d.Key != cs[0].Key() {
			t.Fatal("win both conflicts before six points", d)
		}
	})
	t.Run("overkill has no linear shield reward", func(t *testing.T) {
		o := testObservation(3, 6)
		o.Now.Cities[0].Buildings = catalog.DesignSetOf(51, 35)
		cs := []rules.Choice{testBuild(52, 0, 0), testBuild(60, 0, 0)}
		if d := testPick(t, o, cs); d.Key != cs[1].Key() {
			t.Fatal("six points before useless extra shields", d)
		}
	})
}
func TestGreedyPaymentsAndAbilities(t *testing.T) {
	t.Run("pay the weaker neighbour", func(t *testing.T) {
		o := testObservation(3, 5)
		o.Now.Cities[o.Now.Seat.Left(o.Now.Seats)].Buildings = catalog.DesignSetOf(56, 57, 58)
		cs := []rules.Choice{testBuild(60, 2, 0), testBuild(60, 0, 2)}
		if d := testPick(t, o, cs); d.Key != cs[1].Key() {
			t.Fatal("avoid enriching the left leader", d)
		}
	})
	t.Run("Babylon early extra cards", func(t *testing.T) {
		o := testObservation(1, 2)
		o.Now.Cities[0].Board = catalog.BoardBabylon
		o.Now.Cities[1].Board = catalog.BoardAlexandria
		o.Now.Cities[0].Side = catalog.Night
		cs := []rules.Choice{{Action: rules.Action{Kind: rules.ActStage, Design: 17}, Stage: 1}, testBuild(18, 0, 0)}
		if d := testPick(t, o, cs); d.Key != cs[0].Key() {
			t.Fatal("three future extra cards before three points", d)
		}
	})
	t.Run("Halik offer uses exact own portfolio", func(t *testing.T) {
		o := testObservation(3, 4)
		o.Now.Phase = rules.PhaseDiscardBuild
		o.Now.Decider = 0
		o.Now.Cities[0].Buildings = catalog.DesignSetOf(25, 26)
		cs := []rules.Choice{{Action: rules.Action{Kind: rules.ActBuildDiscarded, Design: 68}, Free: rules.FreeDiscard}, {Action: rules.Action{Kind: rules.ActBuildDiscarded, Design: 17}, Free: rules.FreeDiscard}}
		if d := testPick(t, o, cs); d.Key != cs[0].Key() {
			t.Fatal("free science set before altar", d)
		}
	})
}
func TestRepeatabilityWorkAndExactFallback(t *testing.T) {
	o := testObservation(2, 3)
	o.Now.Hand = [rules.HandSize]catalog.DesignID{39, 40}
	cs := []rules.Choice{testBuild(39, 0, 0), testBuild(40, 0, 0), {Action: rules.Action{Kind: rules.ActSell, Design: 39}}}
	for _, level := range []Level{Random, Greedy, Apprentice} {
		first, err := Choose(context.Background(), level, o, cs, 91, DefaultWork())
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 5; i++ {
			got, err := Choose(context.Background(), level, o.Clone(), cs, 91, DefaultWork())
			if err != nil || !reflect.DeepEqual(got, first) {
				t.Fatal("same input changed", level, first, got, err)
			}
		}
		if first.Counters.Iterations != 1 || first.Counters.Ranked > rules.MaxTurnChoices || first.Counters.Quotes > MaxQuoteCalls {
			t.Fatal("unbounded work", first)
		}
	}
	got, err := Fallback(context.Background(), o, cs)
	if err != nil || got.Key != testPick(t, o, cs).Key {
		t.Fatal("fallback differs from greedy", got, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := Choose(ctx, Greedy, o, cs, 1, DefaultWork()); err == nil || got.Complete || got.Key != "" {
		t.Fatal("cancel accepted a partial key", got, err)
	}
	bad := DefaultWork()
	bad.Iterations = 2
	if _, err := Choose(context.Background(), Greedy, o, cs, 1, bad); err == nil {
		t.Fatal("unsupported work row accepted")
	}
}

// Err is queried at deterministic candidate boundaries, with no clock race.
type cancelDuringRank struct {
	context.Context
	calls int
}

func (c *cancelDuringRank) Err() error {
	c.calls++
	if c.calls >= 4 {
		return context.Canceled
	}
	return nil
}
func TestOrderConcurrentRepeatabilityAndCancellation(t *testing.T) {
	o := testObservation(2, 3)
	o.Now.Hand = [rules.HandSize]catalog.DesignID{39, 39, 40}
	cs := []rules.Choice{testBuild(39, 0, 0), testBuild(40, 0, 0), {Action: rules.Action{Kind: rules.ActSell, Design: 39}}}
	before := o
	original := append([]rules.Choice(nil), cs...)
	reversed := []rules.Choice{cs[2], cs[1], cs[0]}
	for level := Random; level < NumLevels; level++ {
		want, err := Choose(context.Background(), level, o, cs, 91, DefaultWork())
		if err != nil {
			t.Fatal(err)
		}
		for _, offers := range [][]rules.Choice{cs, reversed} {
			got, err := Choose(context.Background(), level, o, offers, 91, DefaultWork())
			if err != nil || got != want {
				t.Fatal("offer order changed result", level, want, got, err)
			}
		}
		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Go(func() {
				got, err := Choose(context.Background(), level, o, cs, 91, DefaultWork())
				if err != nil || got != want {
					t.Error("concurrent repeat changed result", level, got, err)
				}
			})
		}
		wg.Wait()
	}
	if o != before || !reflect.DeepEqual(original, cs) {
		t.Fatal("policy mutated permitted inputs")
	}
	ctx := &cancelDuringRank{Context: context.Background()}
	prefix := []Ranked{{Key: "caller-prefix"}}
	ranked, counts, err := Rank(ctx, o, cs, prefix)
	if !errors.Is(err, context.Canceled) || !reflect.DeepEqual(ranked, prefix) || counts.Ranked == 0 || counts.Iterations != 0 {
		t.Fatal("cancelled rank exposed partial work", ranked, counts, err)
	}
	ctx = &cancelDuringRank{Context: context.Background()}
	got, err := Choose(ctx, Greedy, o, cs, 91, DefaultWork())
	if !errors.Is(err, context.Canceled) || got != (Decision{}) {
		t.Fatal("cancelled choose exposed partial incumbent", got, err)
	}
}
