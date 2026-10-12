package ai

import (
	"context"
	"errors"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// This file ranks the legal actions of an observed position with the Greedy
// search (greedy.go), for the move review the rooms show a player after
// their move (the advisor hint family). Greedy's bot plays only its best
// action; RankActions keeps the value of every action, so the review can
// compare the move the player made with the best one.
//
// The ranking is deterministic: the determinizations come from the seed
// and the observation (callRNG), and the budget is a count of evaluations,
// never a clock. A ranking cut short by the budget keeps only the
// determinizations it finished, every action valued on each of them, so
// the comparison stays paired; one cut short before any finished reports
// ErrRankBudget. The context is only a safety net for the caller (a room
// that must not stall): a cancelled ranking reports the context's error,
// and the caller drops it rather than use a partial result.

// Defaults of RankActions.
const (
	DefaultRankK        = 16      // determinizations: enough that a gap of a few VP is above sampling noise
	DefaultRankMaxEvals = 200_000 // evaluations, about 20 ms in the largest positions on a laptop
)

// RankConfig configures RankActions. The zero value uses the defaults.
type RankConfig struct {
	K        int      // determinizations; 0 means DefaultRankK
	MaxEvals int      // evaluations before the ranking stops; 0 means DefaultRankMaxEvals
	Weights  *Weights // the evaluation's weights; nil means DefaultWeights
}

// Ranking is the value of every legal action of an observed position for
// the observer, by Greedy's search: Values[i] is the mean, over the
// finished determinizations, of the value of Actions[i] in victory-point
// equivalents (a won game is worth Weights.Win).
type Ranking struct {
	Actions    []engine.Action // the legal actions, in the engine's order
	Values     []float64       // by action
	Iterations int             // determinizations finished
	Evals      int             // evaluations spent, the cut-short determinization included
}

// Best returns the index of the action of highest value, the first one on
// a tie.
func (r *Ranking) Best() int { return argmax(r.Values) }

// Index returns the index of action a in the ranking, or -1.
func (r *Ranking) Index(a engine.Action) int {
	for i, b := range r.Actions {
		if b == a {
			return i
		}
	}
	return -1
}

// ErrRankBudget is RankActions's error when the evaluation budget ran out
// before the first determinization was finished.
var ErrRankBudget = errors.New("ai: the ranking's budget ran out")

// RankActions values every legal action of the observed position for the
// observer, who must be to move, as Greedy would (one ply, the opponent's
// best reply, an extra ply for an extra turn or a pending choice), over
// cfg.K determinizations drawn from seed and the observation. It never
// reads the real state: only obs and its determinizations.
func RankActions(ctx context.Context, obs engine.Observation, seed uint64, cfg RankConfig) (Ranking, error) {
	if err := ctx.Err(); err != nil {
		return Ranking{}, err
	}
	legal, err := choiceSetup(&obs, nil)
	if err != nil {
		return Ranking{}, err
	}
	k := cfg.K
	if k <= 0 {
		k = DefaultRankK
	}
	maxEvals := cfg.MaxEvals
	if maxEvals <= 0 {
		maxEvals = DefaultRankMaxEvals
	}
	w := defaultWeights
	if cfg.Weights != nil {
		w = *cfg.Weights
	}
	n := len(legal)
	out := Ranking{Actions: legal, Values: make([]float64, n)}
	sums := make([]float64, n)
	vals := make([]float64, n)
	rng := callRNG(seed, &obs)
	sr := greedySearch{me: obs.Seat(), w: &w, ctx: ctx}
	for it := 0; it < k; it++ {
		d := obs.Determinize(&rng)
		finished := true
		for i, a := range legal {
			if sr.evals >= maxEvals || sr.stopped {
				finished = false
				break
			}
			c := d
			engine.ApplyTrusted(&c, a)
			vals[i] = sr.mine(&c, 0, greedyExtraPlies)
		}
		if sr.stopped {
			out.Evals = sr.evals
			if err := ctx.Err(); err != nil {
				return out, err
			}
			return out, context.Canceled
		}
		if !finished {
			break
		}
		for i := range n {
			sums[i] += vals[i]
		}
		out.Iterations++
	}
	out.Evals = sr.evals
	if out.Iterations == 0 {
		return out, ErrRankBudget
	}
	for i := range n {
		out.Values[i] = sums[i] / float64(out.Iterations)
	}
	return out, nil
}
