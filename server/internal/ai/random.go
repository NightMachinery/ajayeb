package ai

import (
	"context"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// RandomBot plays a uniformly random legal action. It is the baseline of
// the tournaments and the Random level of the rooms.
type RandomBot struct {
	seed uint64
}

// NewRandom returns a random bot seeded with seed.
func NewRandom(seed uint64) *RandomBot { return &RandomBot{seed: seed} }

// Choose picks one legal action uniformly at random. It ignores the budget,
// the context and the progress callback: it never searches.
func (r *RandomBot) Choose(_ context.Context, obs engine.Observation, _ Budget, _ func(Progress)) (engine.Action, error) {
	var buf [engine.MaxLegalActions]engine.Action
	legal, err := choiceSetup(&obs, buf[:0])
	if err != nil {
		return engine.Action{}, err
	}
	if len(legal) == 1 {
		return legal[0], nil
	}
	rng := callRNG(r.seed, &obs)
	return legal[rng.IntN(len(legal))], nil
}
