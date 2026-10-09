package ai_test

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// The benchmarks of the engine operations a search iteration is made of,
// against the design's targets (engine design, section 10, on one Apple
// M-series core):
//
//   - Clone (a state copy): 30 ns
//   - LegalActions with 6 accessible cards and 4 wonders: 1 µs
//   - ApplyTrusted: 300 ns
//   - a random playout of a whole game (about 70 plies): 80 µs
//   - Determinize: 1 µs
//   - one search iteration (descent, a 16-ply heuristic rollout and the
//     evaluation): 40 µs (BenchmarkMCTSIteration in mcts_internal_test.go)
//
// Run them with: go test ./internal/ai -run '^$' -bench . -benchtime 2s
// (through tools/heavy.py, kind bench).

// benchSink keeps results alive, so the compiler cannot drop the work.
var benchSink uint16

// benchWide returns a play-phase position with 6 accessible cards and 4
// unbuilt wonders for the seat to move, the design's LegalActions case: of
// those in the first random games, the one with the most legal actions.
func benchWide(b *testing.B) engine.State {
	b.Helper()
	var best engine.State
	most := 0
	for seed := uint64(1); seed < 40; seed++ {
		for _, s := range aiPositions(b, seed) {
			if s.Phase != engine.PhasePlay || popcount(engine.Accessible(&s)) != 6 {
				continue
			}
			open := 0
			for _, w := range s.Players[s.ToMove].Wonders {
				if w.ID != 0 && !w.Built && !w.Removed {
					open++
				}
			}
			if n := len(engine.LegalActions(&s, nil)); open == 4 && n > most {
				best, most = s, n
			}
		}
	}
	if most == 0 {
		b.Fatal("no position with 6 accessible cards and 4 open wonders")
	}
	return best
}

func popcount(m uint32) int {
	n := 0
	for ; m != 0; m &= m - 1 {
		n++
	}
	return n
}

func BenchmarkClone(b *testing.B) {
	s := greedyMid(b)
	var c engine.State
	b.ReportAllocs()
	for b.Loop() {
		c = s.Clone()
		benchSink += c.Ply
	}
}

// BenchmarkLegalActions lists the legal actions of the design's case (6
// accessible cards, 4 unbuilt wonders) and of the widest play position of
// the first random games.
func BenchmarkLegalActions(b *testing.B) {
	widest, most := engine.State{}, 0
	for seed := uint64(1); seed < 40; seed++ {
		for _, s := range aiPositions(b, seed) {
			if n := len(engine.LegalActions(&s, nil)); s.Phase == engine.PhasePlay && n > most {
				widest, most = s, n
			}
		}
	}
	cases := []struct {
		name string
		s    engine.State
	}{{"six-cards-four-wonders", benchWide(b)}, {"widest", widest}}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			buf := make([]engine.Action, 0, engine.MaxLegalActions)
			b.ReportAllocs()
			for b.Loop() {
				buf = engine.LegalActions(&c.s, buf[:0])
			}
			b.ReportMetric(float64(len(buf)), "actions")
		})
	}
}

// BenchmarkApplyTrusted applies each legal action of a mid-game position in
// turn, to a fresh copy (so the time includes one Clone).
func BenchmarkApplyTrusted(b *testing.B) {
	s := greedyMid(b)
	legal := engine.LegalActions(&s, nil)
	var c engine.State
	i := 0
	b.ReportAllocs()
	for b.Loop() {
		c = s
		engine.ApplyTrusted(&c, legal[i])
		if i++; i == len(legal) {
			i = 0
		}
	}
	benchSink += c.Ply
}

// BenchmarkRandomPlayout plays whole games from the start with uniformly
// random legal actions (from 16 deals, so NewGame is not timed).
func BenchmarkRandomPlayout(b *testing.B) {
	var starts [16]engine.State
	for i := range starts {
		starts[i] = engine.NewGame(uint64(i+1), engine.Options{})
	}
	buf := make([]engine.Action, 0, engine.MaxLegalActions)
	r := engine.NewRNG(1)
	plies, games := 0, 0
	b.ReportAllocs()
	for b.Loop() {
		s := starts[games%len(starts)]
		for !s.Over() {
			legal := engine.LegalActions(&s, buf[:0])
			engine.ApplyTrusted(&s, legal[r.IntN(len(legal))])
		}
		plies += int(s.Ply)
		games++
	}
	b.ReportMetric(float64(plies)/float64(games), "plies/game")
}

func BenchmarkDeterminize(b *testing.B) {
	cases := []struct {
		name string
		s    engine.State
	}{
		{"draft", engine.NewGame(71, engine.Options{})},
		{"age2", greedyMid(b)},
	}
	for _, s := range aiPositions(b, 71) {
		if s.Age == 3 && s.Phase == engine.PhasePlay {
			cases = append(cases, struct {
				name string
				s    engine.State
			}{"age3", s})
			break
		}
	}
	for _, c := range cases {
		b.Run(c.name, func(b *testing.B) {
			o := engine.Observe(&c.s, c.s.ToMove)
			r := engine.NewRNG(71)
			b.ReportAllocs()
			for b.Loop() {
				d := o.Determinize(&r)
				benchSink += d.Ply
			}
		})
	}
}
