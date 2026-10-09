//go:build slow

package ai_test

import (
	"math"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// The slow tournaments (go test -tags slow): seeded matches with paired
// seeds, each seed played twice with the seats swapped, so that neither bot
// profits from the deal or from moving first. The tree-search levels play
// their fixed budgets (Level.FixedBudget), so every result is the same on
// any machine; docs/ai.md records them.
//
// The tournaments of the levels take long: on an Apple M2 with GOMAXPROCS
// 4, Medium against Greedy takes about 15 minutes and Hard against Medium
// about 6 hours. Run them with a long enough -timeout, or as a quicker proxy
// with these variables:
//
//   - AJAYEB_TOUR_SCALE multiplies the iteration budgets of Medium and Hard
//     (both sides alike; default 1);
//   - AJAYEB_TOUR_GAMES replaces every tournament's number of games;
//   - AJAYEB_TOUR_P is the number of games played at once (default 2).
//
// The thresholds stay the same under a proxy. The nightly CI job plays the
// full tournaments with ajayeb-sim, sharded over several runners, and is the
// authority on them.

// tourResult is one game's outcome from the first bot's side.
type tourResult struct {
	score float64 // 1 for a win, 0.5 for a shared victory, 0 for a loss
	kind  engine.VictoryKind
}

// tourEnv reads a positive number from an environment variable, or returns
// def when it is unset.
func tourEnv(t *testing.T, name string, def float64) float64 {
	t.Helper()
	text := os.Getenv(name)
	if text == "" {
		return def
	}
	v, err := strconv.ParseFloat(text, 64)
	if err != nil || !(v > 0) || math.IsInf(v, 0) {
		t.Fatalf("%s=%q is not a positive number", name, text)
	}
	return v
}

// tourGames returns a tournament's number of games: games, or
// AJAYEB_TOUR_GAMES (rounded up to an even number).
func tourGames(t *testing.T, games int) int {
	t.Helper()
	n := int(tourEnv(t, "AJAYEB_TOUR_GAMES", float64(games)))
	return max(2, n+n%2)
}

// tourLevelBudget returns a tree-search level's fixed budget, times
// AJAYEB_TOUR_SCALE.
func tourLevelBudget(t *testing.T, l ai.Level) ai.Budget {
	t.Helper()
	b := l.FixedBudget()
	b.MaxIterations = max(1, int(math.Round(float64(b.MaxIterations)*tourEnv(t, "AJAYEB_TOUR_SCALE", 1))))
	return b
}

// tourMatch plays games games (an even number) between the bots made by
// newA and newB, AJAYEB_TOUR_P at a time, and returns each game's result
// for A. The games are set up by ai.PairedGame, as ajayeb-sim match sets
// them up, so "ajayeb-sim match -a greedy -b random -games 100 -seed 1"
// replays the Greedy tournament below, and "ajayeb-sim match -a medium -b
// greedy -games 100 -seed 1" the Medium one.
func tourMatch(t *testing.T, games int, seed uint64, newA, newB func(seed uint64) ai.Bot, budgets [2]ai.Budget) []tourResult {
	t.Helper()
	parallel := int(tourEnv(t, "AJAYEB_TOUR_P", 2))
	results := make([]tourResult, games)
	errs := make([]error, games)
	next := make(chan int)
	var wg sync.WaitGroup
	for range max(1, min(parallel, games)) {
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

// tourScore sums A's results and logs them: the share of points with its
// 95% interval, and A's wins by victory kind.
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
	lo, hi := tourWilson(rate, len(results))
	t.Logf("%s: %.1f of %d (%.1f%%, 95%% interval %.1f%% to %.1f%%); wins by military %d, science %d, civilian %d",
		label, total, len(results), 100*rate, 100*lo, 100*hi,
		kinds[engine.VictoryMilitary], kinds[engine.VictoryScience], kinds[engine.VictoryCivilian])
	return rate
}

// tourWilson returns the 95% Wilson score interval of a proportion p
// observed over n trials.
func tourWilson(p float64, n int) (lo, hi float64) {
	const z = 1.959963984540054
	nf := float64(n)
	den := 1 + z*z/nf
	centre := (p + z*z/(2*nf)) / den
	half := z * math.Sqrt(p*(1-p)/nf+z*z/(4*nf*nf)) / den
	return max(0, centre-half), min(1, centre+half)
}

// tourCheck fails the test when rate is below want.
func tourCheck(t *testing.T, a, b string, rate, want float64) {
	t.Helper()
	if rate < want {
		t.Errorf("%s scored %.1f%% against %s, want at least %.0f%%", a, 100*rate, b, 100*want)
	}
}

func tourGreedy(seed uint64) ai.Bot { return ai.NewGreedy(seed, ai.GreedyConfig{}) }
func tourEasy(seed uint64) ai.Bot   { return ai.New(ai.Easy, seed) }
func tourRandom(seed uint64) ai.Bot { return ai.NewRandom(seed) }
func tourMedium(seed uint64) ai.Bot { return ai.New(ai.Medium, seed) }
func tourHard(seed uint64) ai.Bot   { return ai.New(ai.Hard, seed) }

// TestTournamentGreedyBeatsRandom: the deterministic Greedy bot, at its
// default budget, scores at least 90% against Random over 100 paired games.
func TestTournamentGreedyBeatsRandom(t *testing.T) {
	rate := tourScore(t, "Greedy vs Random", tourMatch(t, tourGames(t, 100), 1, tourGreedy, tourRandom, [2]ai.Budget{}))
	tourCheck(t, "Greedy", "Random", rate, 0.9)
}

// TestTournamentEasyBeatsRandom: Easy errs but still beats Random clearly:
// at least 85% over 100 paired games.
func TestTournamentEasyBeatsRandom(t *testing.T) {
	rate := tourScore(t, "Easy vs Random", tourMatch(t, tourGames(t, 100), 1, tourEasy, tourRandom,
		[2]ai.Budget{ai.Easy.FixedBudget(), {}}))
	tourCheck(t, "Easy", "Random", rate, 0.85)
}

// TestTournamentGreedyBeatsEasy: the softmax makes Easy weaker than Greedy:
// Greedy scores at least 65% over 100 paired games.
func TestTournamentGreedyBeatsEasy(t *testing.T) {
	rate := tourScore(t, "Greedy vs Easy", tourMatch(t, tourGames(t, 100), 1, tourGreedy, tourEasy,
		[2]ai.Budget{{}, ai.Easy.FixedBudget()}))
	tourCheck(t, "Greedy", "Easy", rate, 0.65)
}

// TestTournamentMediumBeatsGreedy: Medium, on its fixed budget, scores at
// least 65% against the deterministic Greedy over 100 paired games.
func TestTournamentMediumBeatsGreedy(t *testing.T) {
	rate := tourScore(t, "Medium vs Greedy", tourMatch(t, tourGames(t, 100), 1, tourMedium, tourGreedy,
		[2]ai.Budget{tourLevelBudget(t, ai.Medium), {}}))
	tourCheck(t, "Medium", "Greedy", rate, 0.65)
}

// TestTournamentHardBeatsMedium: Hard scores at least 58% against Medium
// over 200 paired games, both on their fixed budgets.
func TestTournamentHardBeatsMedium(t *testing.T) {
	rate := tourScore(t, "Hard vs Medium", tourMatch(t, tourGames(t, 200), 1, tourHard, tourMedium,
		[2]ai.Budget{tourLevelBudget(t, ai.Hard), tourLevelBudget(t, ai.Medium)}))
	tourCheck(t, "Hard", "Medium", rate, 0.58)
}
