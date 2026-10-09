package ai_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

// The tests of the tree search through its public face, MCTSBot.Choose:
// the levels, legal output, determinism (with one worker and with several),
// equal observations, one-move tactics, cancellation, time budgets,
// progress and allocations.

// mctsFast is the budget of the fast tests' searches.
var mctsFast = ai.Budget{MaxIterations: 300}

func TestLevelsUseTheSearch(t *testing.T) {
	for _, l := range []ai.Level{ai.Medium, ai.Hard} {
		if _, ok := ai.New(l, 1).(*ai.MCTSBot); !ok {
			t.Errorf("%s is a %T", l, ai.New(l, 1))
		}
	}
}

// TestMCTSChoosesLegalActions: in positions of every phase, with one worker
// and with two, the search returns an action legal in the real state.
func TestMCTSChoosesLegalActions(t *testing.T) {
	var phases [8]int
	bot := ai.NewMCTS(1, ai.MCTSConfig{})
	for seed := uint64(1); seed <= 2; seed++ {
		for i, s := range aiPositions(t, seed) {
			if s.Over() || (i%5 != 0 && s.Phase == engine.PhasePlay) {
				continue
			}
			phases[s.Phase]++
			for _, workers := range []int{1, 2} {
				a := aiChoose(t, bot, &s, ai.Budget{MaxIterations: 100, Workers: workers})
				if err := engine.IsLegal(&s, a); err != nil {
					t.Fatalf("game %d ply %d (%s), %d workers: chose %s: %v", seed, s.Ply, s.Phase, workers, a, err)
				}
			}
		}
	}
	t.Logf("positions by phase: %v", phases)
}

func TestMCTSChooseErrors(t *testing.T) {
	states := aiPositions(t, 3)
	over := states[len(states)-1]
	mid := states[len(states)/2]
	bot := ai.NewMCTS(1, ai.MCTSConfig{})
	if _, err := bot.Choose(t.Context(), engine.Observe(&over, 0), mctsFast, nil); !errors.Is(err, ai.ErrGameOver) {
		t.Errorf("at game over: %v, want ErrGameOver", err)
	}
	if _, err := bot.Choose(t.Context(), engine.Observe(&mid, mid.ToMove.Opp()), mctsFast, nil); !errors.Is(err, ai.ErrNotToMove) {
		t.Errorf("for the waiting seat: %v, want ErrNotToMove", err)
	}
}

// mctsFinal runs bot on s and returns its choice and its final progress
// report.
func mctsFinal(tb testing.TB, bot ai.Bot, s *engine.State, b ai.Budget) (engine.Action, ai.Progress) {
	tb.Helper()
	var last ai.Progress
	a, err := bot.Choose(tb.Context(), engine.Observe(s, s.ToMove), b, func(p ai.Progress) { last = p })
	if err != nil {
		tb.Fatal(err)
	}
	return a, last
}

// TestMCTSIsDeterministic: under an iteration budget the same bot gives the
// same choice and the same final statistics on every run, with one worker
// and with several (whatever the goroutine scheduling), and two games
// between the same bots are the same game.
func TestMCTSIsDeterministic(t *testing.T) {
	s := greedyMid(t)
	bot := ai.NewMCTS(12, ai.MCTSConfig{})
	for _, workers := range []int{1, 3} {
		b := ai.Budget{MaxIterations: 3000, Workers: workers}
		a, p := mctsFinal(t, bot, &s, b)
		if p.Iterations != b.MaxIterations || p.Best != a {
			t.Fatalf("%d workers: final report %+v for %s", workers, p, a)
		}
		for range 2 {
			if a2, p2 := mctsFinal(t, bot, &s, b); a2 != a || p2.Iterations != p.Iterations || p2.BestShare != p.BestShare {
				t.Fatalf("%d workers: %s (share %v), then %s (share %v)", workers, a, p.BestShare, a2, p2.BestShare)
			}
		}
		t.Logf("%d workers: %s with %.1f%% of the visits", workers, a, 100*p.BestShare)
	}

	play := func() []engine.LogEntry {
		log := engine.NewLog(4, engine.Options{}, nil)
		bots := [2]ai.Bot{ai.NewMCTS(4, ai.MCTSConfig{}), ai.NewGreedy(5, ai.GreedyConfig{})}
		budgets := [2]ai.Budget{{MaxIterations: 150, Workers: 2}, aiFastGreedy}
		if _, err := ai.PlayGame(t.Context(), engine.NewGame(4, engine.Options{}), bots, budgets, log, nil); err != nil {
			t.Fatal(err)
		}
		return log.Actions
	}
	if g1, g2 := play(), play(); !slices.Equal(g1, g2) {
		t.Fatal("two games between the same bots differ")
	}
}

// TestMCTSSeesOnlyTheObservation: states that differ only in hidden
// information give the same choice.
func TestMCTSSeesOnlyTheObservation(t *testing.T) {
	r := engine.NewRNG(78)
	bot := ai.NewMCTS(6, ai.MCTSConfig{})
	checked := 0
	for i, s := range aiPositions(t, 6) {
		if s.Over() || i%8 != 3 {
			continue
		}
		o := engine.Observe(&s, s.ToMove)
		other := o.Determinize(&r)
		a1 := aiChoose(t, bot, &s, mctsFast)
		a2 := aiChoose(t, bot, &other, mctsFast)
		if a1 != a2 {
			t.Fatalf("ply %d: chose %s and %s for equal observations", s.Ply, a1, a2)
		}
		checked++
	}
	if checked < 5 {
		t.Fatalf("only %d positions checked", checked)
	}
}

// TestMCTSTakesASupremacy: Choose plays a move that wins at once in the
// named supremacy scenarios (or, for Law, opens the winning token choice
// and then takes Law).
func TestMCTSTakesASupremacy(t *testing.T) {
	for _, name := range []string{"military-supremacy", "military-via-wonder", "science-supremacy", "law-supremacy"} {
		s := aiNamed(t, name, 0)
		me := s.ToMove
		bot := ai.NewMCTS(3, ai.MCTSConfig{})
		for n := 0; !s.Over() && s.ToMove == me && n < 3; n++ {
			a := aiChoose(t, bot, &s, mctsFast)
			if err := engine.Apply(&s, a, nil); err != nil {
				t.Fatal(err)
			}
		}
		if !s.Over() || s.Result.Winner != int8(me) {
			t.Errorf("%s: did not win at once (phase %s, result %+v)", name, s.Phase, s.Result)
		}
	}
}

// TestMCTSBlocksAMilitaryWin: the opponent wins with the Walls if seat 0
// leaves them (see TestGreedyBlocksAMilitaryWin); the search must take them
// away.
func TestMCTSBlocksAMilitaryWin(t *testing.T) {
	for _, coins := range []int{0, 12} {
		s := enginetest.New(t, 31).Age(2).FirstPlayer(0).ToMove(0).
			Coins(0, coins).Coins(1, 40).Pawn(-7).
			OnlyLeft("walls", "statue", "library").State()
		a := aiChoose(t, ai.NewMCTS(4, ai.MCTSConfig{}), &s, ai.Budget{MaxIterations: 2000})
		if err := engine.Apply(&s, a, nil); err != nil {
			t.Fatal(err)
		}
		if greedyOppCanWin(&s) {
			t.Errorf("coins %d: played %s and left seat 1 a winning reply", coins, a)
		}
	}
}

// TestMCTSStopsOnCancel: a search with an endless budget returns soon after
// its context is cancelled, with a legal action: within 20 ms (the design's
// target) in the median of three tries, and never later than 100 ms (slack
// for a loaded machine).
func TestMCTSStopsOnCancel(t *testing.T) {
	s := greedyMid(t)
	var lats []time.Duration
	for try, workers := range []int{1, 3, 2} {
		ctx, cancel := context.WithCancel(t.Context())
		type result struct {
			a   engine.Action
			err error
			at  time.Time
		}
		done := make(chan result, 1)
		go func() {
			a, err := ai.NewMCTS(uint64(try), ai.MCTSConfig{}).Choose(ctx, engine.Observe(&s, s.ToMove), ai.Budget{MaxIterations: 1 << 30, Workers: workers}, nil)
			done <- result{a, err, time.Now()}
		}()
		time.Sleep(60 * time.Millisecond)
		cancelled := time.Now()
		cancel()
		select {
		case r := <-done:
			if r.err != nil {
				t.Fatal(r.err)
			}
			if err := engine.IsLegal(&s, r.a); err != nil {
				t.Fatalf("after cancel: %s: %v", r.a, err)
			}
			lats = append(lats, r.at.Sub(cancelled))
		case <-time.After(5 * time.Second):
			t.Fatal("Choose did not return after the cancel")
		}
	}
	t.Logf("returned %v after the cancel", lats)
	sorted := slices.Clone(lats)
	slices.Sort(sorted)
	if sorted[1] > 20*time.Millisecond || sorted[2] > 100*time.Millisecond {
		t.Errorf("returned %v after the cancel, want under 20 ms", lats)
	}
}

// TestMCTSCancelledBeforeItStarts: with a context done from the start, the
// search still returns a legal action at once.
func TestMCTSCancelledBeforeItStarts(t *testing.T) {
	s := greedyMid(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	start := time.Now()
	a, err := ai.NewMCTS(1, ai.MCTSConfig{}).Choose(ctx, engine.Observe(&s, s.ToMove), ai.Budget{MaxIterations: 1 << 30, Workers: 2}, nil)
	if err != nil || engine.IsLegal(&s, a) != nil {
		t.Fatalf("%s, %v", a, err)
	}
	if el := time.Since(start); el > 50*time.Millisecond {
		t.Errorf("took %v", el)
	}
}

// TestMCTSMaxTimeAndProgress: a time budget ends the search on time, on
// several workers, and the progress callback fires about every 100 ms and
// once at the end, with the choice.
func TestMCTSMaxTimeAndProgress(t *testing.T) {
	s := greedyMid(t)
	const budget = 350 * time.Millisecond
	var reports []ai.Progress
	start := time.Now()
	a, err := ai.NewMCTS(2, ai.MCTSConfig{}).Choose(t.Context(), engine.Observe(&s, s.ToMove),
		ai.Budget{MaxTime: budget, Workers: 2}, func(p ai.Progress) { reports = append(reports, p) })
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.IsLegal(&s, a); err != nil {
		t.Fatal(err)
	}
	if elapsed < budget || elapsed > budget+150*time.Millisecond {
		t.Errorf("a %v budget took %v", budget, elapsed)
	}
	if len(reports) < 3 || len(reports) > 6 {
		t.Fatalf("%d progress reports in %v, want one about every 100 ms and a final one", len(reports), elapsed)
	}
	for i, p := range reports {
		if i > 0 && (p.Iterations < reports[i-1].Iterations || p.Elapsed < reports[i-1].Elapsed) {
			t.Errorf("report %d goes backwards: %+v after %+v", i, p, reports[i-1])
		}
		if p.BestShare <= 0 || p.BestShare > 1 || engine.IsLegal(&s, p.Best) != nil {
			t.Errorf("report %d: %+v", i, p)
		}
	}
	last := reports[len(reports)-1]
	if last.Best != a || last.Iterations < 100 {
		t.Errorf("the final report %+v does not match the choice %s", last, a)
	}
	t.Logf("%d iterations on 2 workers in %v (%.0f per second per worker), %d reports",
		last.Iterations, elapsed, float64(last.Iterations)/elapsed.Seconds()/2, len(reports))
}

// TestMCTSChooseAllocatesLittle: a whole search makes a handful of
// allocations, however many iterations it runs: the iterations themselves
// allocate nothing, and the arena's chunks come back from earlier searches.
func TestMCTSChooseAllocatesLittle(t *testing.T) {
	s := greedyMid(t)
	bot := ai.NewMCTS(1, ai.MCTSConfig{})
	obs := engine.Observe(&s, s.ToMove)
	ctx := t.Context()
	choose := func(iterations int) float64 {
		return testing.AllocsPerRun(3, func() {
			if _, err := bot.Choose(ctx, obs, ai.Budget{MaxIterations: iterations}, nil); err != nil {
				t.Fatal(err)
			}
		})
	}
	small, large := choose(500), choose(5000)
	t.Logf("allocations per search: %v at 500 iterations, %v at 5000", small, large)
	if large > 20 || large > small+4 {
		t.Fatalf("a search allocates %v times at 500 iterations and %v at 5000", small, large)
	}
}

func BenchmarkMCTSChoose(b *testing.B) {
	s := greedyMid(b)
	bot := ai.NewMCTS(1, ai.MCTSConfig{})
	obs := engine.Observe(&s, s.ToMove)
	const iterations = 2000
	b.ReportAllocs()
	for b.Loop() {
		if _, err := bot.Choose(b.Context(), obs, ai.Budget{MaxIterations: iterations}, nil); err != nil {
			b.Fatal(err)
		}
	}
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/iterations/1000, "us/iteration")
}
