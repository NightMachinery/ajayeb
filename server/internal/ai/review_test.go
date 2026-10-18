package ai_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// The tests of review.go, the tree search of the move review: every legal
// action searched, the comparison's equal shares, determinism for a fixed
// seed and on equal observations, a search cut short, and the actions a
// comparison refuses.

// TestSearchActionsIsDeterministic: a fixed seed and iteration count give
// the same search, over every legal action in the engine's order, and a
// determinization of the observation (whose hidden cards differ) gives
// the same one.
func TestSearchActionsIsDeterministic(t *testing.T) {
	checked := 0
	for _, s := range aiPositions(t, 3) {
		if s.Over() || s.Phase != engine.PhasePlay || s.Ply%7 != 0 {
			continue
		}
		obs := engine.Observe(&s, s.ToMove)
		one, err := ai.SearchActions(context.Background(), obs, 42, 600)
		if err != nil {
			t.Fatalf("ply %d: %v", s.Ply, err)
		}
		two, _ := ai.SearchActions(context.Background(), obs, 42, 600)
		if !reflect.DeepEqual(one, two) {
			t.Fatalf("ply %d: two searches differ", s.Ply)
		}
		if !reflect.DeepEqual(one.Actions, engine.LegalActions(&s, nil)) || one.Iterations != 600 {
			t.Fatalf("ply %d: the search lists %d actions in %d iterations", s.Ply, len(one.Actions), one.Iterations)
		}
		total := 0
		for _, v := range one.Visits {
			total += v
		}
		if total != 600 || one.Best() < 0 {
			t.Fatalf("ply %d: %d root visits, best %d", s.Ply, total, one.Best())
		}
		r := engine.NewRNG(uint64(s.Ply))
		d := obs.Determinize(&r)
		if other, _ := ai.SearchActions(context.Background(), engine.Observe(&d, d.ToMove), 42, 600); !reflect.DeepEqual(one, other) {
			t.Fatalf("ply %d: a determinization of the observation searches differently", s.Ply)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no position checked")
	}
}

// TestCompareActions: the comparison gives each of its two actions half
// the iterations, is deterministic, and refuses the same action twice or
// an action that is not legal.
func TestCompareActions(t *testing.T) {
	checked := 0
	for _, s := range aiPositions(t, 3) {
		if s.Over() || s.Phase != engine.PhasePlay || s.Ply%7 != 0 {
			continue
		}
		legal := engine.LegalActions(&s, nil)
		if len(legal) < 2 {
			continue
		}
		a, b := legal[0], legal[len(legal)-1]
		obs := engine.Observe(&s, s.ToMove)
		one, err := ai.CompareActions(context.Background(), obs, 7, a, b, 301)
		if err != nil {
			t.Fatalf("ply %d: %v", s.Ply, err)
		}
		if !reflect.DeepEqual(one.Actions, []engine.Action{a, b}) || one.Iterations != 302 ||
			one.Visits[0] != 151 || one.Visits[1] != 151 {
			t.Fatalf("ply %d: %v visits in %d iterations", s.Ply, one.Visits, one.Iterations)
		}
		for _, v := range one.Values {
			if v < 0 || v > 1 {
				t.Fatalf("ply %d: a mean reward of %v", s.Ply, v)
			}
		}
		r := engine.NewRNG(uint64(s.Ply))
		d := obs.Determinize(&r)
		if two, _ := ai.CompareActions(context.Background(), engine.Observe(&d, d.ToMove), 7, a, b, 301); !reflect.DeepEqual(one, two) {
			t.Fatalf("ply %d: a determinization of the observation compares differently", s.Ply)
		}
		if _, err := ai.CompareActions(context.Background(), obs, 7, a, a, 100); !errors.Is(err, ai.ErrNotComparable) {
			t.Fatalf("ply %d: comparing an action with itself: %v", s.Ply, err)
		}
		if _, err := ai.CompareActions(context.Background(), obs, 7, a, engine.Action{Type: engine.ActChooseStarter}, 100); !errors.Is(err, ai.ErrNotComparable) {
			t.Fatalf("ply %d: comparing with an illegal action: %v", s.Ply, err)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no position checked")
	}
}

// TestReviewSearchCutShort: a search whose context ends before its last
// iteration reports the context's error, never a partial result.
func TestReviewSearchCutShort(t *testing.T) {
	var s engine.State
	for _, p := range aiPositions(t, 1) {
		if !p.Over() && p.Phase == engine.PhasePlay {
			s = p
			break
		}
	}
	obs := engine.Observe(&s, s.ToMove)
	legal := engine.LegalActions(&s, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := ai.SearchActions(ctx, obs, 1, 1<<30); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("a search cut short: %v", err)
	}
	if _, err := ai.CompareActions(ctx, obs, 1, legal[0], legal[1], 1<<30); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("a comparison cut short: %v", err)
	}
}
