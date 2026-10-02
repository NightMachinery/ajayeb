package enginetest

import (
	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// Policy picks one of the legal actions of a state. Policies are for test
// playouts; the real bots live in internal/ai.
type Policy interface {
	Choose(s *engine.State, legal []engine.Action) engine.Action
}

// RandomPolicy picks uniformly among the legal actions.
type RandomPolicy struct {
	RNG engine.RNG
}

// NewRandomPolicy returns a random policy seeded with seed.
func NewRandomPolicy(seed uint64) *RandomPolicy {
	return &RandomPolicy{RNG: engine.NewRNG(seed)}
}

// Choose picks a legal action at random.
func (p *RandomPolicy) Choose(_ *engine.State, legal []engine.Action) engine.Action {
	return legal[p.RNG.IntN(len(legal))]
}

// SimplePolicy plays a little like a beginner: in the play phase it usually
// builds a wonder when it can, otherwise the building worth the most
// (points, shields, a science symbol, production), and discards only when
// nothing is affordable; one move in Explore it plays at random instead.
// Every other choice is random. Ties are broken at random, so the games vary
// with the seed.
type SimplePolicy struct {
	RNG     engine.RNG
	Explore int // one move in Explore is random; 0 means 5
}

// NewSimplePolicy returns a simple policy seeded with seed.
func NewSimplePolicy(seed uint64) *SimplePolicy {
	return &SimplePolicy{RNG: engine.NewRNG(seed)}
}

// Choose picks the best-looking legal action.
func (p *SimplePolicy) Choose(s *engine.State, legal []engine.Action) engine.Action {
	explore := p.Explore
	if explore <= 0 {
		explore = 5
	}
	if s.Phase != engine.PhasePlay || p.RNG.IntN(explore) == 0 {
		return legal[p.RNG.IntN(len(legal))]
	}
	best, bestScore, ties := 0, -1, 0
	for i, a := range legal {
		sc := simpleScore(a)
		switch {
		case sc > bestScore:
			best, bestScore, ties = i, sc, 1
		case sc == bestScore:
			// Reservoir sampling keeps a uniform choice among the ties.
			ties++
			if p.RNG.IntN(ties) == 0 {
				best = i
			}
		}
	}
	return legal[best]
}

// simpleScore rates a play-phase action.
func simpleScore(a engine.Action) int {
	switch a.Type {
	case engine.ActConstructWonder:
		return 100 + int(catalog.T.WVP[a.Wonder])
	case engine.ActConstructBuilding:
		t := &catalog.T
		c := a.Card
		v := 10 + int(t.VP[c]) + 2*int(t.Shields[c])
		if t.Sym[c] != catalog.NoSymbol {
			v += 3
		}
		for _, n := range t.Prod[c] {
			v += 2 * int(n)
		}
		if t.Choice[c] != 0 {
			v += 2
		}
		return v
	}
	return 0 // discard
}

// Playout plays *s with the given policies (indexed by seat) until the game
// is over or maxPlies actions have been played, whichever comes first. After
// each action it calls after, if not nil, with the action and its events; an
// error from after stops the playout. It returns an error when an action
// fails, after fails, or the game is not over after maxPlies actions.
func Playout(s *engine.State, pols [2]Policy, maxPlies int, after func(a engine.Action, ev []engine.Event) error) error {
	legal := make([]engine.Action, 0, engine.MaxLegalActions)
	var ev []engine.Event
	for n := 0; !s.Over(); n++ {
		if n >= maxPlies {
			return errorf("the game is not over after %d actions (phase %s, age %d)", maxPlies, s.Phase, s.Age)
		}
		legal = engine.LegalActions(s, legal[:0])
		if len(legal) == 0 {
			return errorf("no legal action in phase %s at ply %d", s.Phase, s.Ply)
		}
		a := pols[s.ToMove].Choose(s, legal)
		ev = ev[:0]
		if err := engine.Apply(s, a, &ev); err != nil {
			return err
		}
		if after != nil {
			if err := after(a, ev); err != nil {
				return err
			}
		}
	}
	return nil
}
