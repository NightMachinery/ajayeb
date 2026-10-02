package ai_test

import (
	"math"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

// The tests of eval.go.

const evalEps = 1e-9

// TestEvaluateIsZeroSum: the value for one seat is minus the value for the
// other, in every position of a few games, finished ones included.
func TestEvaluateIsZeroSum(t *testing.T) {
	for seed := uint64(1); seed <= 4; seed++ {
		for _, s := range aiPositions(t, seed) {
			v0, v1 := ai.Evaluate(&s, 0), ai.Evaluate(&s, 1)
			if math.IsNaN(v0) || math.IsInf(v0, 0) || math.Abs(v0+v1) > evalEps {
				t.Fatalf("game %d ply %d: values %v and %v", seed, s.Ply, v0, v1)
			}
		}
	}
}

// TestEvaluateReadsOnlyPublicInformation: the value of an observation's
// public state equals the value of the real state and of any
// determinization.
func TestEvaluateReadsOnlyPublicInformation(t *testing.T) {
	r := engine.NewRNG(5)
	for seed := uint64(1); seed <= 3; seed++ {
		for _, s := range aiPositions(t, seed) {
			for seat := range engine.Seat(2) {
				o := engine.Observe(&s, seat)
				want := ai.Evaluate(&s, seat)
				if got := ai.Evaluate(o.Public(), seat); got != want {
					t.Fatalf("game %d ply %d seat %d: public %v, real %v", seed, s.Ply, seat, got, want)
				}
				d := o.Determinize(&r)
				if got := ai.Evaluate(&d, seat); got != want {
					t.Fatalf("game %d ply %d seat %d: determinized %v, real %v", seed, s.Ply, seat, got, want)
				}
			}
		}
	}
}

func TestEvaluateTerminalValues(t *testing.T) {
	w := ai.DefaultWeights()
	for _, name := range []string{"military-supremacy", "science-supremacy", "civilian-end", "draw"} {
		sc := engine.NamedScenario(name)
		s := aiNamed(t, name, len(sc.Trigger))
		if !s.Over() {
			t.Fatalf("%s: not over", name)
		}
		for seat := range engine.Seat(2) {
			v := w.Evaluate(&s, seat)
			switch s.Result.Winner {
			case int8(seat):
				if v < w.Win*0.9 {
					t.Errorf("%s: the winner's value is %v", name, v)
				}
			case engine.SharedVictory:
				if math.Abs(v) > w.Win*0.1 {
					t.Errorf("%s: a shared victory is worth %v", name, v)
				}
			default:
				if v > -w.Win*0.9 {
					t.Errorf("%s: the loser's value is %v", name, v)
				}
			}
		}
	}
}

// evalBase is a quiet Age II position for the feature tests: seat 0 to move,
// both seats with 10 coins and no buildings.
func evalBase(tb testing.TB) *enginetest.Builder {
	return enginetest.New(tb, 9).Age(2).FirstPlayer(0).ToMove(0).Coins(0, 10).Coins(1, 10)
}

// TestEvaluateFeatures checks the direction of each feature: a position
// better for seat 0 in one respect is worth more to seat 0.
func TestEvaluateFeatures(t *testing.T) {
	base := evalBase(t).State()
	v0 := ai.Evaluate(&base, 0)
	cases := []struct {
		name string
		s    engine.State
	}{
		{"more coins", evalBase(t).Coins(0, 16).State()},
		{"the pawn on the opponent's side", evalBase(t).Pawn(4).State()},
		{"a blue card", evalBase(t).City(0, "statue").State()},
		{"production", evalBase(t).City(0, "sawmill").State()},
		{"a flexible producer", evalBase(t).City(0, "caravansery").State()},
		{"three science symbols", evalBase(t).City(0, "scriptorium", "pharmacist", "workshop").State()},
		{"a built wonder", evalBase(t).Wonders(0, "pyramids").Built(0, "pyramids", "").State()},
		{"strategy", evalBase(t).Tokens(0, "strategy").State()},
	}
	for _, c := range cases {
		if v := ai.Evaluate(&c.s, 0); v <= v0 {
			t.Errorf("%s: %v, not above the base %v", c.name, v, v0)
		}
		if v := ai.Evaluate(&c.s, 1); v >= -v0 {
			t.Errorf("%s: %v for the opponent, not below %v", c.name, v, -v0)
		}
	}
}

// TestEvaluateMilitaryThreat: the nearer the pawn to the opponent's capital,
// the more it is worth, with the threat bonus on the last three spaces.
func TestEvaluateMilitaryThreat(t *testing.T) {
	prev := math.Inf(-1)
	for pawn := -8; pawn <= 8; pawn++ {
		s := evalBase(t).Pawn(pawn).State()
		v := ai.Evaluate(&s, 0)
		if v <= prev {
			t.Fatalf("pawn %d: %v, not above %v at pawn %d", pawn, v, prev, pawn-1)
		}
		prev = v
	}
	near, far := evalBase(t).Pawn(8).State(), evalBase(t).Pawn(5).State()
	if d := ai.Evaluate(&near, 0) - ai.Evaluate(&far, 0); d < 20 {
		t.Errorf("one space from the capital is worth only %v more than four spaces", d)
	}
}

// TestEvaluateDeadScience: five symbols are worth less once the opponent
// holds every card of the two symbols still missing (and Law is gone).
func TestEvaluateDeadScience(t *testing.T) {
	five := []string{"scriptorium", "pharmacist", "workshop", "apothecary", "academy"}
	alive := enginetest.New(t, 9).Age(3).FirstPlayer(0).ToMove(0).City(0, five...).
		BoardTokens("agriculture", "architecture", "economy", "masonry", "mathematics").State()
	// The globe is the missing symbol (with the scales, whose Law is not on
	// the board); the opponent takes both globe cards.
	globes := evalSymbolCards(t, catalog.Globe)
	dead := enginetest.New(t, 9).Age(3).FirstPlayer(0).ToMove(0).City(0, five...).City(1, globes...).
		BoardTokens("agriculture", "architecture", "economy", "masonry", "mathematics").State()
	va, vd := ai.Evaluate(&alive, 0), ai.Evaluate(&dead, 0)
	// The opponent's two green cards add points and symbols of their own;
	// the drop in seat 0's science value must outweigh them.
	if va-vd < 5 {
		t.Errorf("five symbols with the sixth alive: %v; with it dead: %v", va, vd)
	}
}

// evalSymbolCards returns the slugs of the green cards showing sym.
func evalSymbolCards(tb testing.TB, sym catalog.Symbol) []string {
	tb.Helper()
	var out []string
	for c := range catalog.T.All.All() {
		if catalog.T.Sym[c] == sym {
			out = append(out, catalog.Card(c).Slug)
		}
	}
	if len(out) != 2 {
		tb.Fatalf("%d cards show %s", len(out), sym)
	}
	return out
}

func TestEvaluateAllocatesNothing(t *testing.T) {
	states := aiPositions(t, 2)
	s := states[len(states)/2]
	if n := testing.AllocsPerRun(100, func() { ai.Evaluate(&s, 0) }); n != 0 {
		t.Fatalf("Evaluate allocates %v times", n)
	}
}

func BenchmarkEvaluate(b *testing.B) {
	states := aiPositions(b, 2)
	s := states[len(states)/2]
	for b.Loop() {
		ai.Evaluate(&s, 0)
	}
}
