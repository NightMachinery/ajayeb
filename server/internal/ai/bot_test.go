package ai_test

import (
	"context"
	"errors"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

// The tests of bot.go: levels, the errors of Choose, the single-action
// shortcut, PlayGame and its logs. The shared helpers of the package's tests
// are here too, prefixed with ai.

// aiFastGreedy is the Greedy budget of the fast tests: one determinization
// per move.
var aiFastGreedy = ai.Budget{MaxIterations: 1}

// aiPositions plays a seeded game with random policies and returns every
// state before an action, and the final state.
func aiPositions(tb testing.TB, seed uint64) []engine.State {
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

// aiNamed builds named scenario name and runs the first steps of its
// trigger.
func aiNamed(tb testing.TB, name string, steps int) engine.State {
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

// aiChoose runs bot for the seat to move in s.
func aiChoose(tb testing.TB, bot ai.Bot, s *engine.State, b ai.Budget) engine.Action {
	tb.Helper()
	a, err := bot.Choose(tb.Context(), engine.Observe(s, s.ToMove), b, nil)
	if err != nil {
		tb.Fatalf("ply %d (phase %s): %v", s.Ply, s.Phase, err)
	}
	return a
}

func TestParseLevel(t *testing.T) {
	for _, l := range []ai.Level{ai.Random, ai.Easy, ai.Medium, ai.Hard} {
		b, err := l.MarshalText()
		if err != nil {
			t.Fatal(err)
		}
		var back ai.Level
		if err := back.UnmarshalText(b); err != nil || back != l {
			t.Errorf("%s: round trip gave %v, %v", l, back, err)
		}
		if got, err := ai.ParseLevel(l.String()); err != nil || got != l {
			t.Errorf("ParseLevel(%q) = %v, %v", l.String(), got, err)
		}
	}
	for _, bad := range []string{"", "Easy", "greedy", "hard ", "level#9"} {
		if _, err := ai.ParseLevel(bad); err == nil {
			t.Errorf("ParseLevel(%q) accepted", bad)
		}
	}
	if _, err := ai.Level(9).MarshalText(); err == nil {
		t.Error("an invalid level marshalled")
	}
}

func TestNewEveryLevelPlaysLegally(t *testing.T) {
	states := aiPositions(t, 5)
	for _, l := range []ai.Level{ai.Random, ai.Easy, ai.Medium, ai.Hard} {
		bot := ai.New(l, 7)
		for i := 0; i < len(states)-1; i += 9 {
			s := states[i]
			a := aiChoose(t, bot, &s, aiFastGreedy)
			if err := engine.IsLegal(&s, a); err != nil {
				t.Fatalf("%s at ply %d chose %s: %v", l, s.Ply, a, err)
			}
		}
	}
}

func TestNewPanicsOnABadLevel(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("New(9) did not panic")
		}
	}()
	ai.New(9, 1)
}

func TestChooseErrors(t *testing.T) {
	states := aiPositions(t, 3)
	over := states[len(states)-1]
	if !over.Over() {
		t.Fatal("the last state is not over")
	}
	mid := states[len(states)/2]
	for _, bot := range []ai.Bot{ai.NewRandom(1), ai.NewGreedy(1, ai.GreedyConfig{})} {
		if _, err := bot.Choose(t.Context(), engine.Observe(&over, 0), aiFastGreedy, nil); !errors.Is(err, ai.ErrGameOver) {
			t.Errorf("%T at game over: %v, want ErrGameOver", bot, err)
		}
		if _, err := bot.Choose(t.Context(), engine.Observe(&mid, mid.ToMove.Opp()), aiFastGreedy, nil); !errors.Is(err, ai.ErrNotToMove) {
			t.Errorf("%T for the waiting seat: %v, want ErrNotToMove", bot, err)
		}
	}
}

// TestSingleActionReturnsAtOnce: the Statue of Zeus against a city with one
// brown card leaves a single legal action, which every bot returns without
// searching: no progress report, even with a cancelled context.
func TestSingleActionReturnsAtOnce(t *testing.T) {
	s := aiNamed(t, "zeus-destroy", 1)
	if keys := enginetest.Keys(&s); len(keys) != 1 || keys[0] != "destroy:quarry" {
		t.Fatalf("legal = %v, want just destroy:quarry", keys)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	for _, l := range []ai.Level{ai.Random, ai.Easy, ai.Medium, ai.Hard} {
		reported := false
		a, err := ai.New(l, 3).Choose(ctx, engine.Observe(&s, s.ToMove), ai.Budget{MaxIterations: 1000}, func(ai.Progress) { reported = true })
		if err != nil || a.Key() != "destroy:quarry" {
			t.Errorf("%s: %s, %v", l, a, err)
		}
		if reported {
			t.Errorf("%s reported progress for a forced move", l)
		}
	}
}

func TestMixSeedSpreads(t *testing.T) {
	seen := map[uint64]bool{}
	for seed := range uint64(50) {
		for tag := range uint64(50) {
			v := ai.MixSeed(seed, tag)
			if seen[v] {
				t.Fatalf("MixSeed(%d, %d) repeats %#x", seed, tag, v)
			}
			seen[v] = true
		}
	}
}

// TestPlayGameLogReplays: a game between two bots replays from its log to
// the sealed hash, and both seats moved.
func TestPlayGameLogReplays(t *testing.T) {
	for seed := uint64(1); seed <= 3; seed++ {
		start := engine.NewGame(seed, engine.Options{})
		log := engine.NewLog(seed, engine.Options{}, nil)
		var moves [2]int
		bots := [2]ai.Bot{ai.NewGreedy(seed, ai.GreedyConfig{}), ai.NewRandom(seed)}
		end, err := ai.PlayGame(t.Context(), start, bots, [2]ai.Budget{aiFastGreedy, {}}, log, func(m ai.MoveInfo) { moves[m.Seat]++ })
		if err != nil {
			t.Fatal(err)
		}
		if !end.Over() || moves[0] == 0 || moves[1] == 0 || len(log.Actions) != moves[0]+moves[1] {
			t.Fatalf("game %d: over %v, moves %v, log %d", seed, end.Over(), moves, len(log.Actions))
		}
		back, err := engine.Replay(log)
		if err != nil {
			t.Fatalf("game %d: replay: %v", seed, err)
		}
		if back != end {
			t.Fatalf("game %d: the replay does not reach the played state", seed)
		}
	}
}

func TestPlayGameStopsOnACancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	s := engine.NewGame(4, engine.Options{})
	bots := [2]ai.Bot{ai.NewRandom(1), ai.NewRandom(2)}
	end, err := ai.PlayGame(ctx, s, bots, [2]ai.Budget{}, nil, nil)
	if !errors.Is(err, context.Canceled) || end != s {
		t.Fatalf("PlayGame with a cancelled context: %v, ply %d", err, end.Ply)
	}
}

// aiBadBot returns a fixed action, legal or not.
type aiBadBot struct{ a engine.Action }

func (b aiBadBot) Choose(context.Context, engine.Observation, ai.Budget, func(ai.Progress)) (engine.Action, error) {
	return b.a, nil
}

func TestPlayGameRejectsAnIllegalChoice(t *testing.T) {
	bad := aiBadBot{engine.StarterAction(0)} // never legal in the draft
	s := engine.NewGame(4, engine.Options{FirstPlayer: engine.FirstPlayerSeat0})
	if _, err := ai.PlayGame(t.Context(), s, [2]ai.Bot{bad, bad}, [2]ai.Budget{}, nil, nil); err == nil {
		t.Fatal("an illegal choice was played")
	}
}
