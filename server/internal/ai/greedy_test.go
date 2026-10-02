package ai_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

// The tests of greedy.go: legal output, determinism, equal observations,
// one-move tactics (taking a supremacy, blocking the opponent's), the
// budget, cancellation and progress.

// greedyBots are the Greedy variants the fast tests run.
func greedyBots(seed uint64) []ai.Bot {
	return []ai.Bot{
		ai.NewGreedy(seed, ai.GreedyConfig{}),
		ai.NewGreedy(seed, ai.GreedyConfig{Temperature: ai.EasyTemperature}),
	}
}

// greedyMid returns a play-phase position from the middle of a random game
// with many legal actions.
func greedyMid(tb testing.TB) engine.State {
	tb.Helper()
	for seed := uint64(20); seed < 40; seed++ {
		for _, s := range aiPositions(tb, seed) {
			if s.Age == 2 && s.Phase == engine.PhasePlay && len(engine.LegalActions(&s, nil)) >= 14 {
				return s
			}
		}
	}
	tb.Fatal("no Age II position with 14 legal actions")
	return engine.State{}
}

// TestBotsChooseLegalActions: in positions of every phase from random games,
// every bot returns an action that is legal in the real state.
func TestBotsChooseLegalActions(t *testing.T) {
	var phases [8]int
	for seed := uint64(1); seed <= 4; seed++ {
		bots := append(greedyBots(seed), ai.NewRandom(seed), ai.New(ai.Hard, seed))
		for i, s := range aiPositions(t, seed) {
			if s.Over() || (i%4 != 0 && s.Phase == engine.PhasePlay) {
				continue
			}
			phases[s.Phase]++
			for _, bot := range bots {
				a := aiChoose(t, bot, &s, aiFastGreedy)
				if err := engine.IsLegal(&s, a); err != nil {
					t.Fatalf("game %d ply %d (%s): %T chose %s: %v", seed, s.Ply, s.Phase, bot, a, err)
				}
			}
		}
	}
	for _, p := range []engine.Phase{engine.PhaseWonderDraft, engine.PhasePlay, engine.PhaseChooseStarter} {
		if phases[p] == 0 {
			t.Errorf("no %s position was tried", p)
		}
	}
	t.Logf("positions by phase: %v", phases)
}

// TestBotsAreDeterministic: two bots with the same seed play the same game,
// move for move, and a bot asked twice gives the same answer.
func TestBotsAreDeterministic(t *testing.T) {
	for seed := uint64(1); seed <= 2; seed++ {
		play := func() []engine.LogEntry {
			log := engine.NewLog(seed, engine.Options{}, nil)
			bots := [2]ai.Bot{
				ai.NewGreedy(seed, ai.GreedyConfig{Temperature: ai.EasyTemperature}),
				ai.NewGreedy(seed+100, ai.GreedyConfig{}),
			}
			if _, err := ai.PlayGame(t.Context(), engine.NewGame(seed, engine.Options{}), bots, [2]ai.Budget{aiFastGreedy, aiFastGreedy}, log, nil); err != nil {
				t.Fatal(err)
			}
			return log.Actions
		}
		a, b := play(), play()
		if !slices.Equal(a, b) {
			t.Fatalf("game %d: two runs differ", seed)
		}
	}
	s := greedyMid(t)
	for _, bot := range greedyBots(9) {
		first := aiChoose(t, bot, &s, ai.Budget{MaxIterations: 2})
		for range 3 {
			if again := aiChoose(t, bot, &s, ai.Budget{MaxIterations: 2}); again != first {
				t.Fatalf("%T chose %s, then %s", bot, first, again)
			}
		}
	}
}

// TestBotsSeeOnlyTheObservation: a state that differs only in hidden
// information (a determinization of the same observation) gives the same
// choice. The observations are equal values, so this holds as long as a
// choice depends on nothing but the bot and the observation.
func TestBotsSeeOnlyTheObservation(t *testing.T) {
	r := engine.NewRNG(77)
	checked := 0
	for seed := uint64(5); seed <= 6; seed++ {
		bots := append(greedyBots(seed), ai.NewRandom(seed))
		for i, s := range aiPositions(t, seed) {
			if s.Over() || i%6 != 2 {
				continue
			}
			o := engine.Observe(&s, s.ToMove)
			other := o.Determinize(&r)
			if engine.Observe(&other, s.ToMove) != o {
				t.Fatal("a determinization changed the observation")
			}
			for _, bot := range bots {
				a1 := aiChoose(t, bot, &s, aiFastGreedy)
				a2 := aiChoose(t, bot, &other, aiFastGreedy)
				if a1 != a2 {
					t.Fatalf("game %d ply %d: %T chose %s and %s for equal observations", seed, s.Ply, bot, a1, a2)
				}
			}
			checked++
		}
	}
	if checked < 10 {
		t.Fatalf("only %d positions checked", checked)
	}
}

// TestGreedyTakesASupremacy: in the named supremacy scenarios, Greedy and
// Easy play a move that wins at once, or, for Law, the move that opens the
// winning token choice and then the winning token.
func TestGreedyTakesASupremacy(t *testing.T) {
	for _, name := range []string{"military-supremacy", "military-via-wonder", "science-supremacy", "law-supremacy"} {
		for _, bot := range greedyBots(3) {
			s := aiNamed(t, name, 0)
			me := s.ToMove
			for n := 0; !s.Over() && s.ToMove == me && n < 3; n++ {
				a := aiChoose(t, bot, &s, aiFastGreedy)
				if err := engine.Apply(&s, a, nil); err != nil {
					t.Fatal(err)
				}
			}
			if !s.Over() || s.Result.Winner != int8(me) {
				t.Errorf("%s: %T did not win at once (phase %s, result %+v)", name, bot, s.Phase, s.Result)
			}
		}
	}
}

// TestGreedyBlocksAMilitaryWin: the pawn is two spaces from seat 0's
// capital and the Walls (two shields) are on offer, so whatever seat 0
// leaves, seat 1 builds the Walls and wins. Greedy, too poor to build
// them, must take them away (discard them); richer, it may also build
// them. Either way seat 1 must have no winning reply.
func TestGreedyBlocksAMilitaryWin(t *testing.T) {
	for _, coins := range []int{0, 12} {
		build := func() engine.State {
			return enginetest.New(t, 31).Age(2).FirstPlayer(0).ToMove(0).
				Coins(0, coins).Coins(1, 40).Pawn(-7).
				OnlyLeft("walls", "statue", "library").State()
		}
		probe := build()
		enginetest.Do(t, &probe, "discard:statue")
		if !greedyOppCanWin(&probe) {
			t.Fatal("the scenario has no threat: seat 1 cannot win after a harmless discard")
		}
		for _, bot := range greedyBots(4) {
			s := build()
			a := aiChoose(t, bot, &s, aiFastGreedy)
			if err := engine.Apply(&s, a, nil); err != nil {
				t.Fatal(err)
			}
			if greedyOppCanWin(&s) {
				t.Errorf("coins %d: %T played %s and left seat 1 a winning reply", coins, bot, a)
			}
		}
	}
}

// greedyOppCanWin reports whether the seat to move in s has an action that
// wins the game by military supremacy at once.
func greedyOppCanWin(s *engine.State) bool {
	mover := s.ToMove
	for _, a := range engine.LegalActions(s, nil) {
		x := *s
		engine.ApplyTrusted(&x, a)
		if x.Over() && x.Result.Kind == engine.VictoryMilitary && x.Result.Winner == int8(mover) {
			return true
		}
	}
	return false
}

// TestGreedyStopsOnCancel: a search with an endless iteration budget returns
// soon after its context is cancelled, with a legal action.
func TestGreedyStopsOnCancel(t *testing.T) {
	s := greedyMid(t)
	ctx, cancel := context.WithCancel(t.Context())
	type result struct {
		a   engine.Action
		err error
		at  time.Time
	}
	done := make(chan result, 1)
	go func() {
		a, err := ai.NewGreedy(1, ai.GreedyConfig{}).Choose(ctx, engine.Observe(&s, s.ToMove), ai.Budget{MaxIterations: 1 << 30}, nil)
		done <- result{a, err, time.Now()}
	}()
	time.Sleep(30 * time.Millisecond)
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
		lat := r.at.Sub(cancelled)
		t.Logf("returned %v after the cancel", lat)
		if lat > 100*time.Millisecond { // the design's target is 20 ms; the slack is for a loaded machine
			t.Errorf("returned %v after the cancel, want under 20 ms", lat)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Choose did not return after the cancel")
	}
}

// TestGreedyMaxTimeAndProgress: a time budget ends the search on time, and
// the progress callback fires along the way and once at the end.
func TestGreedyMaxTimeAndProgress(t *testing.T) {
	s := greedyMid(t)
	const budget = 350 * time.Millisecond
	var reports []ai.Progress
	start := time.Now()
	a, err := ai.NewGreedy(2, ai.GreedyConfig{}).Choose(t.Context(), engine.Observe(&s, s.ToMove),
		ai.Budget{MaxIterations: 1 << 30, MaxTime: budget}, func(p ai.Progress) { reports = append(reports, p) })
	elapsed := time.Since(start)
	if err != nil {
		t.Fatal(err)
	}
	if err := engine.IsLegal(&s, a); err != nil {
		t.Fatal(err)
	}
	if elapsed < budget || elapsed > budget+300*time.Millisecond {
		t.Errorf("a %v budget took %v", budget, elapsed)
	}
	if len(reports) < 3 {
		t.Fatalf("%d progress reports in %v, want one about every 100 ms and a final one", len(reports), elapsed)
	}
	for i, p := range reports {
		if i > 0 && (p.Iterations < reports[i-1].Iterations || p.Elapsed < reports[i-1].Elapsed) {
			t.Errorf("report %d goes backwards: %+v after %+v", i, p, reports[i-1])
		}
		if p.BestShare < 0 || p.BestShare > 1 || engine.IsLegal(&s, p.Best) != nil {
			t.Errorf("report %d: %+v", i, p)
		}
	}
	last := reports[len(reports)-1]
	if last.Best != a || last.Iterations < 1 {
		t.Errorf("the final report %+v does not match the choice %s", last, a)
	}
	t.Logf("%d iterations in %v, %d reports", last.Iterations, elapsed, len(reports))
}

// TestGreedyAllocatesLittle: the search itself allocates nothing per
// evaluated move, so a whole Choose makes only a handful of allocations
// (design section 11, group 12: no allocations per iteration in a steady
// state).
func TestGreedyAllocatesLittle(t *testing.T) {
	s := greedyMid(t)
	bot := ai.NewGreedy(1, ai.GreedyConfig{})
	obs := engine.Observe(&s, s.ToMove)
	ctx := t.Context()
	n := testing.AllocsPerRun(5, func() {
		if _, err := bot.Choose(ctx, obs, ai.Budget{MaxIterations: 3}, nil); err != nil {
			t.Fatal(err)
		}
	})
	if n > 8 {
		t.Fatalf("Choose allocates %v times", n)
	}
}

func BenchmarkGreedyChoose(b *testing.B) {
	s := greedyMid(b)
	bot := ai.NewGreedy(1, ai.GreedyConfig{})
	obs := engine.Observe(&s, s.ToMove)
	for b.Loop() {
		if _, err := bot.Choose(b.Context(), obs, aiFastGreedy, nil); err != nil {
			b.Fatal(err)
		}
	}
}
