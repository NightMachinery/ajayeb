package ai

import (
	"context"
	"math"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

// The internal tests of the tree search (mcts.go and parallel.go): the
// search without Choose's sure-win shortcut, the determinism of the merged
// statistics, the arena bound, the allocations of an iteration, and the
// benchmark of one iteration. The helpers shared by the internal test files
// (arena_test.go, rollout_test.go, parallel_test.go) are here, prefixed with
// itest.

// itestPositions plays a seeded game with random policies and returns every
// state before an action, and the final state.
func itestPositions(tb testing.TB, seed uint64) []engine.State {
	tb.Helper()
	s := engine.NewGame(seed, engine.Options{FirstPlayer: engine.FirstPlayerRule(seed % 3)})
	states := []engine.State{s}
	pols := [2]enginetest.Policy{enginetest.NewRandomPolicy(seed), enginetest.NewRandomPolicy(seed ^ 0x5a5a)}
	err := enginetest.Playout(&s, pols, 300, func(engine.Action, []engine.Event) error {
		states = append(states, s)
		return nil
	})
	if err != nil {
		tb.Fatalf("game %d: %v", seed, err)
	}
	return states
}

// itestNamed builds named scenario name and runs the first steps of its
// trigger.
func itestNamed(tb testing.TB, name string, steps int) engine.State {
	tb.Helper()
	sc := engine.NamedScenario(name)
	if sc == nil {
		tb.Fatalf("no named scenario %q", name)
	}
	s, err := engine.NewGameFromScenario(1, engine.Options{}, sc)
	if err != nil {
		tb.Fatal(err)
	}
	for i := range steps {
		a, err := engine.ParseKey(sc.Trigger[i])
		if err != nil {
			tb.Fatal(err)
		}
		if err := engine.Apply(&s, a, nil); err != nil {
			tb.Fatalf("%s step %d: %v", name, i, err)
		}
	}
	return s
}

// itestPendingPositions returns one position of each pending phase, from
// the named scenarios: destroy, revive, a board token and a Great Library
// token.
func itestPendingPositions(tb testing.TB) []engine.State {
	tb.Helper()
	out := []engine.State{
		itestNamed(tb, "zeus-destroy", 1),
		itestNamed(tb, "mausoleum", 1),
		itestNamed(tb, "science-pair", 1),
		itestNamed(tb, "great-library", 1),
	}
	want := []engine.Phase{engine.PhaseDestroyCard, engine.PhaseBuildFromDiscard, engine.PhasePickBoardToken, engine.PhasePickLibraryToken}
	for i, s := range out {
		if s.Phase != want[i] {
			tb.Fatalf("pending position %d is in phase %s, want %s", i, s.Phase, want[i])
		}
	}
	return out
}

// itestMid returns a play-phase position from the middle of a random game
// with many legal actions: Age II, at least 14 of them.
func itestMid(tb testing.TB) engine.State {
	tb.Helper()
	for seed := uint64(20); seed < 40; seed++ {
		for _, s := range itestPositions(tb, seed) {
			if s.Age == 2 && s.Phase == engine.PhasePlay && len(engine.LegalActions(&s, nil)) >= 14 {
				return s
			}
		}
	}
	tb.Fatal("no Age II position with 14 legal actions")
	return engine.State{}
}

// itestOppCanWin reports whether the seat to move in s has an action that
// wins the game at once.
func itestOppCanWin(s *engine.State) bool {
	mover := s.ToMove
	for _, a := range engine.LegalActions(s, nil) {
		x := *s
		engine.ApplyTrusted(&x, a)
		if x.Over() && x.Result.Winner == int8(mover) {
			return true
		}
	}
	return false
}

// itestSearch is a search set up as MCTSBot.search sets it up, for the seat
// to move in s, but run step by step so a test can look inside.
type itestSearch struct {
	obs   engine.Observation
	legal []engine.Action
	first engine.State
	run   *searchRun
}

// itestNewSearch prepares a search of s by m under plan p.
func itestNewSearch(tb testing.TB, ctx context.Context, m *MCTSBot, s *engine.State, p searchPlan, onProgress func(Progress)) *itestSearch {
	tb.Helper()
	x := &itestSearch{obs: engine.Observe(s, s.ToMove)}
	legal, err := choiceSetup(&x.obs, make([]engine.Action, 0, engine.MaxLegalActions))
	if err != nil {
		tb.Fatal(err)
	}
	x.legal = legal
	rng := callRNG(m.seed, &x.obs)
	seeds := [2]uint64{rng.Uint64(), rng.Uint64()}
	x.first = x.obs.Determinize(&rng)
	x.run = m.newSearchRun(ctx, &x.obs, x.legal, p, seeds, &x.first, onProgress)
	tb.Cleanup(x.run.finish)
	return x
}

// itestSearchMove runs a whole search of s, as Choose would but without the
// sure-win shortcut, and returns the move.
func itestSearchMove(tb testing.TB, m *MCTSBot, s *engine.State, iterations, workers int) engine.Action {
	tb.Helper()
	x := itestNewSearch(tb, tb.Context(), m, s, searchPlan{iterations: iterations, workers: workers}, nil)
	x.run.runWorkers()
	return x.run.result()
}

// TestSearchTakesASupremacy: the search alone (no sure-win shortcut) plays
// a move that wins at once in the named supremacy scenarios, or, for Law,
// the move that opens the winning token choice and then the winning token.
func TestSearchTakesASupremacy(t *testing.T) {
	for _, name := range []string{"military-supremacy", "military-via-wonder", "science-supremacy", "law-supremacy"} {
		s := itestNamed(t, name, 0)
		me := s.ToMove
		m := NewMCTS(3, MCTSConfig{})
		for n := 0; !s.Over() && s.ToMove == me && n < 3; n++ {
			a := itestSearchMove(t, m, &s, 1500, 1)
			if err := engine.Apply(&s, a, nil); err != nil {
				t.Fatal(err)
			}
		}
		if !s.Over() || s.Result.Winner != int8(me) {
			t.Errorf("%s: the search did not win at once (phase %s, result %+v)", name, s.Phase, s.Result)
		}
	}
}

// TestSearchIsDeterministic: under an iteration budget, two searches of the
// same position by the same bot give the same merged root statistics, bit
// for bit, with one worker and with several; and every worker runs its
// share of the iterations.
func TestSearchIsDeterministic(t *testing.T) {
	s := itestMid(t)
	m := NewMCTS(11, MCTSConfig{})
	for _, workers := range []int{1, 3} {
		const iterations = 2000
		var stats [2]rootStats
		var moves [2]engine.Action
		for k := range stats {
			x := itestNewSearch(t, t.Context(), m, &s, searchPlan{iterations: iterations, workers: workers}, nil)
			x.run.runWorkers()
			moves[k] = x.run.result()
			stats[k] = x.run.merged
		}
		if stats[0] != stats[1] || moves[0] != moves[1] {
			t.Fatalf("%d workers: two searches differ: %s and %s", workers, moves[0], moves[1])
		}
		if stats[0].iterations != iterations {
			t.Errorf("%d workers ran %d iterations, want %d", workers, stats[0].iterations, iterations)
		}
		total := uint64(0)
		for _, v := range stats[0].visits {
			total += v
		}
		if total != iterations {
			t.Errorf("%d workers: the root's children have %d visits, want %d", workers, total, iterations)
		}
	}
}

// TestSearchArenaStaysInBudget: no worker's tree has more nodes than its
// share of the iterations plus the root, and a time limit sizes the arena
// too.
func TestSearchArenaStaysInBudget(t *testing.T) {
	s := itestMid(t)
	m := NewMCTS(5, MCTSConfig{})
	p := searchPlan{iterations: 1000, workers: 3}
	x := itestNewSearch(t, t.Context(), m, &s, p, nil)
	x.run.runWorkers()
	for i := range x.run.workers {
		w := &x.run.workers[i]
		limit, nodes := p.worker(i)
		if w.done != limit || nodes != limit+1 || int(w.tree.limit) != nodes {
			t.Errorf("worker %d: ran %d of %d iterations, arena limit %d (plan %d)", i, w.done, limit, w.tree.limit, nodes)
		}
		if n := w.tree.len(); n > limit+1 || n < 2 {
			t.Errorf("worker %d: %d nodes after %d iterations", i, n, limit)
		}
	}
	timed := searchPlan{maxTime: 50e6, workers: 2} // 50 ms
	if _, nodes := timed.worker(0); nodes != int(0.05*arenaNodesPerSecond)+1 {
		t.Errorf("a 50 ms search sizes its arena to %d nodes", nodes)
	}
}

// TestSearchSurvivesAFullArena: a tree that cannot grow any more still
// searches, and the root keeps the visits.
func TestSearchSurvivesAFullArena(t *testing.T) {
	s := itestMid(t)
	m := NewMCTS(6, MCTSConfig{})
	x := itestNewSearch(t, t.Context(), m, &s, searchPlan{iterations: 500, workers: 1}, nil)
	w := &x.run.workers[0]
	w.tree.limit = 8 // the root and seven children
	x.run.runWorkers()
	if w.done != 500 || w.tree.len() != 8 || w.tree.at(0).visits != 500 {
		t.Fatalf("done %d, nodes %d, root visits %d", w.done, w.tree.len(), w.tree.at(0).visits)
	}
	if err := engine.IsLegal(&s, x.run.result()); err != nil {
		t.Fatal(err)
	}
}

// TestSearchBacksUpForTheMover: every node's value is a sum of rewards in
// [0, 1], so it lies between 0 and its visit count, and a node has no more
// visits than availabilities.
func TestSearchBacksUpForTheMover(t *testing.T) {
	s := itestMid(t)
	m := NewMCTS(7, MCTSConfig{})
	x := itestNewSearch(t, t.Context(), m, &s, searchPlan{iterations: 1500, workers: 1}, nil)
	x.run.runWorkers()
	tr := &x.run.workers[0].tree
	for i := int32(1); int(i) < tr.len(); i++ {
		n := tr.at(i)
		if n.visits == 0 || n.visits > n.avail || n.value < 0 || float64(n.value) > float64(n.visits)*(1+1e-6) || math.IsNaN(float64(n.value)) {
			t.Fatalf("node %d: %+v", i, *n)
		}
	}
}

// TestIterationAllocatesNothing: once the arena has its memory, an
// iteration (determinize, descend, expand, roll out, back up) allocates
// nothing.
func TestIterationAllocatesNothing(t *testing.T) {
	s := itestMid(t)
	m := NewMCTS(8, MCTSConfig{})
	x := itestNewSearch(t, t.Context(), m, &s, searchPlan{iterations: 1 << 14, workers: 1}, nil)
	w := &x.run.workers[0]
	for len(w.tree.chunks) < cap(w.tree.chunks) {
		w.tree.chunks = append(w.tree.chunks, new(nodeChunk))
	}
	for range 500 {
		w.iterate()
	}
	if n := testing.AllocsPerRun(2000, w.iterate); n != 0 {
		t.Fatalf("an iteration allocates %v times", n)
	}
}

// itestStages returns a position of each stage of a game, for the
// benchmarks: the draft, Age I, Age II (itestMid) and late Age III.
func itestStages(tb testing.TB) []struct {
	name string
	s    engine.State
} {
	tb.Helper()
	out := []struct {
		name string
		s    engine.State
	}{{"draft", engine.NewGame(71, engine.Options{})}}
	game := itestPositions(tb, 71)
	for _, s := range game {
		if s.Age == 1 && s.Phase == engine.PhasePlay && s.Ply >= 12 {
			out = append(out, struct {
				name string
				s    engine.State
			}{"age1", s})
			break
		}
	}
	out = append(out, struct {
		name string
		s    engine.State
	}{"age2", itestMid(tb)})
	for i := len(game) - 1; i >= 0; i-- {
		if s := game[i]; s.Age == 3 && s.Phase == engine.PhasePlay && len(engine.LegalActions(&s, nil)) >= 6 && i+8 < len(game) {
			out = append(out, struct {
				name string
				s    engine.State
			}{"age3", s})
			break
		}
	}
	if len(out) != 4 {
		tb.Fatalf("found %d of the 4 stages", len(out))
	}
	return out
}

// BenchmarkMCTSIteration measures one iteration at each stage of a game, in
// trees of up to DefaultMCTSIterations nodes. The design's target is 40 µs.
func BenchmarkMCTSIteration(b *testing.B) {
	for _, st := range itestStages(b) {
		b.Run(st.name, func(b *testing.B) {
			m := NewMCTS(9, MCTSConfig{})
			x := itestNewSearch(b, b.Context(), m, &st.s, searchPlan{iterations: DefaultMCTSIterations, workers: 1}, nil)
			w := &x.run.workers[0]
			b.ReportAllocs()
			for b.Loop() {
				if w.tree.full() {
					w.tree.init(DefaultMCTSIterations + 1)
					w.tree.alloc(0)
				}
				w.iterate()
			}
		})
	}
}
