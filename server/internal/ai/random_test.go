package ai_test

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// The tests of random.go.

// TestRandomCoversTheLegalActions: over many seeds the Random bot picks
// every legal action of a position, each about equally often, and one seed
// always gives the same pick.
func TestRandomCoversTheLegalActions(t *testing.T) {
	var s engine.State
	for _, x := range aiPositions(t, 11) {
		if x.Phase == engine.PhasePlay && len(engine.LegalActions(&x, nil)) >= 8 {
			s = x
			break
		}
	}
	legal := engine.LegalActions(&s, nil)
	if len(legal) < 8 {
		t.Fatal("no play position with 8 legal actions")
	}
	const draws = 4000
	counts := make(map[engine.Action]int)
	obs := engine.Observe(&s, s.ToMove)
	for seed := range uint64(draws) {
		bot := ai.NewRandom(seed)
		a, err := bot.Choose(t.Context(), obs, ai.Budget{}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if again, _ := bot.Choose(t.Context(), obs, ai.Budget{}, nil); again != a {
			t.Fatalf("seed %d chose %s, then %s", seed, a, again)
		}
		counts[a]++
	}
	want := draws / len(legal)
	for _, a := range legal {
		if n := counts[a]; n < want/2 || n > want*3/2 {
			t.Errorf("%s chosen %d times of %d, expected about %d", a, n, draws, want)
		}
	}
	if len(counts) != len(legal) {
		t.Errorf("%d different actions chosen, %d legal", len(counts), len(legal))
	}
}
