//go:build slow

package ai_test

import (
	"sync"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// The slow tournaments (go test -tags slow): seeded matches with paired
// seeds, each seed played twice with the seats swapped, so that neither bot
// profits from the deal or from moving first.

// tourResult is one game's outcome from the first bot's side.
type tourResult struct {
	score float64 // 1 for a win, 0.5 for a shared victory, 0 for a loss
	kind  engine.VictoryKind
}

// tourMatch plays games games (an even number) between the bots made by
// newA and newB, on two goroutines, and returns each game's result for A.
// The games are set up by ai.PairedGame, as ajayeb-sim match sets them up,
// so "ajayeb-sim match -a greedy -b random -games 100 -seed 1" replays the
// Greedy tournament below.
func tourMatch(t *testing.T, games int, seed uint64, newA, newB func(seed uint64) ai.Bot, budgets [2]ai.Budget) []tourResult {
	t.Helper()
	results := make([]tourResult, games)
	errs := make([]error, games)
	next := make(chan int)
	var wg sync.WaitGroup
	for range 2 {
		wg.Go(func() {
			for i := range next {
				gameSeed, aSeat, botSeeds := ai.PairedGame(seed, i)
				var bots [2]ai.Bot
				var b [2]ai.Budget
				bots[aSeat], b[aSeat] = newA(botSeeds[0]), budgets[0]
				bots[aSeat.Opp()], b[aSeat.Opp()] = newB(botSeeds[1]), budgets[1]
				end, err := ai.PlayGame(t.Context(), engine.NewGame(gameSeed, engine.Options{}), bots, b, nil, nil)
				if err != nil {
					errs[i] = err
					continue
				}
				r := tourResult{kind: end.Result.Kind}
				switch end.Result.Winner {
				case int8(aSeat):
					r.score = 1
				case engine.SharedVictory:
					r.score = 0.5
				}
				results[i] = r
			}
		})
	}
	for i := range games {
		next <- i
	}
	close(next)
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("game %d: %v", i, err)
		}
	}
	return results
}

// tourScore sums A's results and logs them: the share of points, and A's
// wins by victory kind.
func tourScore(t *testing.T, label string, results []tourResult) float64 {
	t.Helper()
	total := 0.0
	var kinds [4]int
	for _, r := range results {
		total += r.score
		if r.score == 1 {
			kinds[r.kind]++
		}
	}
	rate := total / float64(len(results))
	t.Logf("%s: %.1f of %d (%.1f%%); wins by military %d, science %d, civilian %d", label, total, len(results),
		100*rate, kinds[engine.VictoryMilitary], kinds[engine.VictoryScience], kinds[engine.VictoryCivilian])
	return rate
}

func tourGreedy(seed uint64) ai.Bot { return ai.NewGreedy(seed, ai.GreedyConfig{}) }
func tourEasy(seed uint64) ai.Bot {
	return ai.NewGreedy(seed, ai.GreedyConfig{Temperature: ai.EasyTemperature})
}
func tourRandom(seed uint64) ai.Bot { return ai.NewRandom(seed) }

// TestTournamentGreedyBeatsRandom: the deterministic Greedy bot, at its
// default budget, scores at least 90% against Random over 100 paired games
// (measured: 100%).
func TestTournamentGreedyBeatsRandom(t *testing.T) {
	rate := tourScore(t, "Greedy vs Random", tourMatch(t, 100, 1, tourGreedy, tourRandom, [2]ai.Budget{}))
	if rate < 0.9 {
		t.Errorf("Greedy scored %.1f%% against Random, want at least 90%%", 100*rate)
	}
}

// TestTournamentEasyBeatsRandom: Easy errs but still beats Random clearly:
// at least 85% over 100 paired games (measured: 95%).
func TestTournamentEasyBeatsRandom(t *testing.T) {
	rate := tourScore(t, "Easy vs Random", tourMatch(t, 100, 1, tourEasy, tourRandom, [2]ai.Budget{}))
	if rate < 0.85 {
		t.Errorf("Easy scored %.1f%% against Random, want at least 85%%", 100*rate)
	}
}

// TestTournamentGreedyBeatsEasy: the softmax makes Easy weaker than Greedy:
// Greedy scores at least 65% over 100 paired games (measured: 81%).
func TestTournamentGreedyBeatsEasy(t *testing.T) {
	rate := tourScore(t, "Greedy vs Easy", tourMatch(t, 100, 1, tourGreedy, tourEasy, [2]ai.Budget{}))
	if rate < 0.65 {
		t.Errorf("Greedy scored %.1f%% against Easy, want at least 65%%", 100*rate)
	}
}
