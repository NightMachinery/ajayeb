package ai

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

// The tests of rollout.go: the policy plays legal moves in every phase,
// takes a supremacy and denies one, a rollout stops at its cut, rewards are
// in range, and nothing allocates.

// itestPolicy returns a rollout policy with configuration c and the default
// weights.
func itestPolicy(c RolloutConfig) *rolloutPolicy {
	w := DefaultWeights()
	p := new(rolloutPolicy)
	p.init(c.resolved(), &w)
	return p
}

func TestRolloutConfigResolved(t *testing.T) {
	d := RolloutConfig{}.resolved()
	if d.Plies != DefaultRolloutPlies || d.Epsilon != DefaultRolloutEpsilon || d.ModeProb != DefaultRolloutModeProb {
		t.Errorf("the zero config resolves to %+v", d)
	}
	n := RolloutConfig{Plies: 5, Epsilon: -1, ModeProb: -1}.resolved()
	if n.Plies != 5 || n.Epsilon != 0 || n.ModeProb != 0 {
		t.Errorf("negative shares resolve to %+v", n)
	}
	c := RolloutConfig{Epsilon: 3, ModeProb: 0.9}.resolved()
	if c.Epsilon != 1 || c.ModeProb != 0.5 {
		t.Errorf("large shares resolve to %+v", c)
	}
}

// TestRolloutPolicyPlaysLegalMoves: in positions of every phase, the
// policy's move is legal, whether it plays by value, partly at random, or
// only at random.
func TestRolloutPolicyPlaysLegalMoves(t *testing.T) {
	var states []engine.State
	for seed := uint64(1); seed <= 6; seed++ {
		states = append(states, itestPositions(t, seed)...)
	}
	states = append(states, itestPendingPositions(t)...)
	r := engine.NewRNG(3)
	var phases [8]int
	for _, c := range []RolloutConfig{{Epsilon: -1, ModeProb: -1}, {}, {Epsilon: 1, ModeProb: 0.5}} {
		p := itestPolicy(c)
		for _, s := range states {
			if s.Over() {
				continue
			}
			phases[s.Phase]++
			for k := range 3 {
				p.mode = [2]rollMode{rollMode(k), rollMode((k + 1) % 3)}
				if a := p.move(&s, &r); engine.IsLegal(&s, a) != nil {
					t.Fatalf("config %+v, ply %d (%s): illegal %s: %v", c, s.Ply, s.Phase, a, engine.IsLegal(&s, a))
				}
			}
		}
	}
	for ph := engine.PhaseWonderDraft; ph < engine.PhaseGameOver; ph++ {
		if phases[ph] == 0 {
			t.Errorf("no %s position was tried", ph)
		}
	}
}

// TestRolloutStopsAtTheCut: a rollout plays until the game ends, the age
// ends, or its plies run out, whichever comes first, and its reward is in
// [0, 1] (exactly 1, 0.5 or 0 at the end of a game).
func TestRolloutStopsAtTheCut(t *testing.T) {
	r := engine.NewRNG(5)
	for _, plies := range []int{1, 4, DefaultRolloutPlies} {
		p := itestPolicy(RolloutConfig{Plies: plies})
		cut, ageEnd, over := 0, 0, 0
		for seed := uint64(1); seed <= 3; seed++ {
			for _, s := range itestPositions(t, seed) {
				if s.Over() {
					continue
				}
				d := s
				reward := p.run(&d, &r)
				played := int(d.Ply - s.Ply)
				switch {
				case d.Over():
					over++
					if reward != 0 && reward != 0.5 && reward != 1 {
						t.Fatalf("a finished game is worth %v", reward)
					}
				case d.Age != s.Age:
					ageEnd++
				case played == plies:
					cut++
				default:
					t.Fatalf("from ply %d, a rollout of %d plies stopped after %d in phase %s", s.Ply, plies, played, d.Phase)
				}
				if played > plies || played < 1 {
					t.Fatalf("a rollout of %d plies played %d", plies, played)
				}
				if reward < 0 || reward > 1 {
					t.Fatalf("reward %v", reward)
				}
				if err := engine.CheckInvariants(&d); err != nil {
					t.Fatalf("after a rollout from ply %d: %v", s.Ply, err)
				}
			}
		}
		if cut == 0 || ageEnd == 0 || over == 0 {
			t.Errorf("%d plies: cut %d, age end %d, game over %d: every way of stopping should occur", plies, cut, ageEnd, over)
		}
	}
}

// TestRolloutTakesASupremacy: with no random moves, the policy plays the
// move that wins at once in the named supremacy scenarios, and takes Law
// when Law completes six symbols.
func TestRolloutTakesASupremacy(t *testing.T) {
	p := itestPolicy(RolloutConfig{Epsilon: -1, ModeProb: -1})
	r := engine.NewRNG(1)
	cases := []struct {
		name  string
		steps int
	}{{"military-supremacy", 0}, {"military-via-wonder", 0}, {"science-supremacy", 0}, {"law-supremacy", 1}}
	for _, c := range cases {
		s := itestNamed(t, c.name, c.steps)
		me := s.ToMove
		engine.ApplyTrusted(&s, p.move(&s, &r))
		if !s.Over() || s.Result.Winner != int8(me) {
			t.Errorf("%s: the policy did not win (phase %s)", c.name, s.Phase)
		}
	}
}

// TestRolloutDeniesASupremacy: the opponent wins with the Walls if seat 0
// leaves them, so the policy takes them away, whether it can afford them or
// not.
func TestRolloutDeniesASupremacy(t *testing.T) {
	p := itestPolicy(RolloutConfig{Epsilon: -1, ModeProb: -1})
	r := engine.NewRNG(1)
	for _, coins := range []int{0, 12} {
		s := enginetest.New(t, 31).Age(2).FirstPlayer(0).ToMove(0).
			Coins(0, coins).Coins(1, 40).Pawn(-7).
			OnlyLeft("walls", "statue", "library").State()
		a := p.move(&s, &r)
		engine.ApplyTrusted(&s, a)
		if itestOppCanWin(&s) {
			t.Errorf("coins %d: the policy played %s and left a winning reply", coins, a)
		}
	}
}

// TestRolloutIsDeterministic: the same state and generator give the same
// rollout.
func TestRolloutIsDeterministic(t *testing.T) {
	s := itestMid(t)
	p := itestPolicy(RolloutConfig{})
	for seed := range uint64(20) {
		a, b := s, s
		ra, rb := engine.NewRNG(seed), engine.NewRNG(seed)
		va, vb := p.run(&a, &ra), p.run(&b, &rb)
		if a != b || va != vb {
			t.Fatalf("seed %d: two rollouts differ", seed)
		}
	}
}

// TestRolloutAllocatesNothing: a rollout, with the evaluation at its end,
// allocates nothing.
func TestRolloutAllocatesNothing(t *testing.T) {
	s := itestMid(t)
	p := itestPolicy(RolloutConfig{})
	r := engine.NewRNG(2)
	var d engine.State
	n := testing.AllocsPerRun(500, func() {
		d = s
		p.run(&d, &r)
	})
	if n != 0 {
		t.Fatalf("a rollout allocates %v times", n)
	}
}

// BenchmarkRollout measures one rollout of up to DefaultRolloutPlies plies
// from a mid-game position, with its evaluation.
func BenchmarkRollout(b *testing.B) {
	s := itestMid(b)
	p := itestPolicy(RolloutConfig{})
	r := engine.NewRNG(2)
	var d engine.State
	b.ReportAllocs()
	for b.Loop() {
		d = s
		p.run(&d, &r)
	}
}
