package ai

import (
	"context"
	"errors"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// This file is the tree search's side of the move review (the advisor hint
// family): the search the two-stage and deep advisors run on the mover's
// observation of the position before a move. rank.go is the quick side,
// Greedy's ranking.
//
// Both functions search the way the Medium bot (the Scholar) does, with
// its rollouts, UCB constant and evaluation, on one worker for a fixed
// number of iterations and with no clock, so their answer depends only on
// the seed and the observation. They never read the real state. A search
// cut short (its context done before the last iteration) reports the
// context's error, and the caller says nothing rather than use it.
//
//   - SearchActions searches every legal action, as the bot would, and
//     reports each one's visits and mean reward. The deep advisor takes its
//     most visited action as the move to compare with the one played.
//   - CompareActions measures two given actions against each other: the
//     root plays them in turn, so each gets half the iterations, and each
//     pair of iterations shares one determinization, so the hidden cards
//     are the same for both and the comparison is paired. Below the root
//     the search is the usual one: both players answer as well as the tree
//     has learnt. Visits alone would favour the better action, and leave
//     the worse one with too few iterations to say how much worse it is.

// ErrNotComparable is CompareActions's error when its two actions are the
// same, or one of them is not legal for the observer.
var ErrNotComparable = errors.New("ai: the actions to compare are not two different legal actions")

// SearchRanking is the tree search's verdict on some actions of an
// observed position, for the observer: the visits of each and its mean
// reward, between 0 (a loss) and 1 (a win), as the rollouts score a
// position (rollout.go).
type SearchRanking struct {
	Actions    []engine.Action // the actions searched, in the engine's order (SearchActions) or as given (CompareActions)
	Visits     []int           // iterations through each action
	Values     []float64       // mean reward of each action; 0 for an action never visited
	Iterations int             // iterations run
}

// Best returns the index of the most visited action, then of the higher
// mean reward, then the first; -1 when none was visited.
func (r *SearchRanking) Best() int {
	best := -1
	for j, v := range r.Visits {
		if v == 0 {
			continue
		}
		if best < 0 || v > r.Visits[best] || (v == r.Visits[best] && r.Values[j] > r.Values[best]) {
			best = j
		}
	}
	return best
}

// Index returns the index of action a in the ranking, or -1.
func (r *SearchRanking) Index(a engine.Action) int {
	for i, b := range r.Actions {
		if b == a {
			return i
		}
	}
	return -1
}

// SearchActions runs the tree search on obs, whose observer must be to
// move, for exactly iterations iterations on one worker, and reports what
// it learnt of every legal action.
func SearchActions(ctx context.Context, obs engine.Observation, seed uint64, iterations int) (SearchRanking, error) {
	legal, err := choiceSetup(&obs, nil)
	if err != nil {
		return SearchRanking{}, err
	}
	return runReviewSearch(ctx, &obs, seed, legal, iterations, false)
}

// CompareActions runs the tree search on obs, whose observer must be to
// move, with the root restricted to the legal actions a and b, played in
// turn, for exactly iterations iterations (rounded up to an even number)
// on one worker. Actions is [a, b].
func CompareActions(ctx context.Context, obs engine.Observation, seed uint64, a, b engine.Action, iterations int) (SearchRanking, error) {
	legal, err := choiceSetup(&obs, nil)
	if err != nil {
		return SearchRanking{}, err
	}
	pair := []engine.Action{a, b}
	for _, x := range pair {
		found := false
		for _, l := range legal {
			found = found || l == x
		}
		if !found || a == b {
			return SearchRanking{}, ErrNotComparable
		}
	}
	return runReviewSearch(ctx, &obs, seed, pair, iterations+iterations%2, true)
}

// runReviewSearch runs one review search over the root actions roots:
// every legal action, or the two of a comparison (forced).
func runReviewSearch(ctx context.Context, obs *engine.Observation, seed uint64, roots []engine.Action, iterations int, forced bool) (SearchRanking, error) {
	if err := ctx.Err(); err != nil {
		return SearchRanking{}, err
	}
	iterations = max(iterations, 1)
	m := NewMCTS(seed, Medium.SearchConfig()) // the plan below replaces its clock budget
	rng := callRNG(seed, obs)
	seeds := [2]uint64{rng.Uint64(), rng.Uint64()}
	first := obs.Determinize(&rng)
	r := m.newSearchRun(ctx, obs, roots, searchPlan{iterations: iterations, workers: 1}, seeds, &first, nil)
	defer r.finish()
	w := &r.workers[0]
	if forced {
		w.forced = r.keys[:len(roots)]
	}
	stopOnDone := context.AfterFunc(ctx, func() { r.stop.Store(true) })
	defer stopOnDone()
	r.runWorkers()
	if w.done < iterations {
		if err := ctx.Err(); err != nil {
			return SearchRanking{}, err
		}
		return SearchRanking{}, context.Canceled
	}
	var rs rootStats
	w.collect(&rs)
	out := SearchRanking{
		Actions:    roots,
		Visits:     make([]int, len(roots)),
		Values:     make([]float64, len(roots)),
		Iterations: w.done,
	}
	for j := range roots {
		out.Visits[j] = int(rs.visits[j])
		if rs.visits[j] > 0 {
			out.Values[j] = rs.value[j] / float64(rs.visits[j])
		}
	}
	return out, nil
}
