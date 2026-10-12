package ai_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// The tests of rank.go: every legal action valued, determinism for a fixed
// seed, the same ranking on equal observations, the evaluation budget, a
// win ranked first.

// rankOf ranks the actions of the seat to move in s.
func rankOf(tb testing.TB, s *engine.State, seed uint64, cfg ai.RankConfig) ai.Ranking {
	tb.Helper()
	rk, err := ai.RankActions(context.Background(), engine.Observe(s, s.ToMove), seed, cfg)
	if err != nil {
		tb.Fatalf("ply %d: %v", s.Ply, err)
	}
	return rk
}

// TestRankActionsIsDeterministic: a fixed seed gives the same ranking, of
// every legal action in the engine's order, and an observation drawn from
// it (a determinization, whose hidden cards differ) gives the same one.
func TestRankActionsIsDeterministic(t *testing.T) {
	checked := 0
	for _, s := range aiPositions(t, 5) {
		if s.Over() || s.Phase != engine.PhasePlay || s.Ply%5 != 0 {
			continue
		}
		one := rankOf(t, &s, 42, ai.RankConfig{})
		two := rankOf(t, &s, 42, ai.RankConfig{})
		if !reflect.DeepEqual(one, two) {
			t.Fatalf("ply %d: two rankings differ", s.Ply)
		}
		if !reflect.DeepEqual(one.Actions, engine.LegalActions(&s, nil)) || len(one.Values) != len(one.Actions) {
			t.Fatalf("ply %d: the ranking does not list the legal actions", s.Ply)
		}
		if one.Iterations != ai.DefaultRankK {
			t.Fatalf("ply %d: %d determinizations, want %d", s.Ply, one.Iterations, ai.DefaultRankK)
		}
		obs := engine.Observe(&s, s.ToMove)
		r := engine.NewRNG(uint64(s.Ply))
		d := obs.Determinize(&r)
		if other := rankOf(t, &d, 42, ai.RankConfig{}); !reflect.DeepEqual(one, other) {
			t.Fatalf("ply %d: a determinization of the observation ranks differently", s.Ply)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no position checked")
	}
}

// TestRankActionsFindsAWin: the move that wins at once is valued as a won
// game, far ahead of discarding its card.
func TestRankActionsFindsAWin(t *testing.T) {
	s := aiNamed(t, "military-supremacy", 0)
	rk := rankOf(t, &s, 1, ai.RankConfig{})
	walls := rk.Index(engine.ConstructAction(wallsCard(t)))
	discard := rk.Index(engine.DiscardAction(wallsCard(t)))
	if walls < 0 || discard < 0 {
		t.Fatal("walls is not in the ranking")
	}
	win := ai.DefaultWeights().Win
	if rk.Values[walls] < win/2 || rk.Values[rk.Best()] < rk.Values[walls] {
		t.Fatalf("building walls is worth %v, the best %v", rk.Values[walls], rk.Values[rk.Best()])
	}
	if rk.Values[walls]-rk.Values[discard] < win/2 {
		t.Fatalf("the win leads the discard by %v", rk.Values[walls]-rk.Values[discard])
	}
}

// TestRankActionsBudget: a budget that runs out keeps only the finished
// determinizations, and one that runs out before the first reports
// ErrRankBudget; a cancelled context reports its error.
func TestRankActionsBudget(t *testing.T) {
	s := greedyMid(t)
	obs := engine.Observe(&s, s.ToMove)
	full := rankOf(t, &s, 7, ai.RankConfig{K: 4})
	cut, err := ai.RankActions(context.Background(), obs, 7, ai.RankConfig{K: 4, MaxEvals: full.Evals / 2})
	if err != nil || cut.Iterations < 1 || cut.Iterations >= 4 {
		t.Fatalf("half the budget: %d determinizations, %v", cut.Iterations, err)
	}
	if _, err := ai.RankActions(context.Background(), obs, 7, ai.RankConfig{MaxEvals: 1}); !errors.Is(err, ai.ErrRankBudget) {
		t.Fatalf("no budget: %v, want ErrRankBudget", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ai.RankActions(ctx, obs, 7, ai.RankConfig{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled: %v", err)
	}
}

// BenchmarkRankActions ranks a mid-game position with the default budget.
func BenchmarkRankActions(b *testing.B) {
	s := greedyMid(b)
	obs := engine.Observe(&s, s.ToMove)
	for b.Loop() {
		if _, err := ai.RankActions(context.Background(), obs, 1, ai.RankConfig{}); err != nil {
			b.Fatal(err)
		}
	}
}

// wallsCard is the Walls card's id.
func wallsCard(tb testing.TB) catalog.CardID {
	tb.Helper()
	a, err := engine.ParseKey("construct:walls")
	if err != nil {
		tb.Fatal(err)
	}
	return a.Card
}
