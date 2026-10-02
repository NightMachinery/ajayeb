package ai

import (
	"context"
	"math"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// This file is the Greedy bot: one ply of its own plus the opponent's best
// reply, averaged over a few determinizations.
//
// For each determinization d of the observation (an iteration), and for
// each legal action a, Greedy plays a on a copy of d and values the result:
//
//   - a finished game is worth its terminal value;
//   - if the opponent is to move, the value is the minimum, over the
//     opponent's replies b, of the evaluation after b: the opponent answers
//     with the reply that is worst for Greedy;
//   - if Greedy is still to move (an extra turn, a pending choice of its
//     own, or the starter choice of a new age), the value is the maximum
//     over its next actions, valued the same way.
//
// A line may add at most greedyExtraPlies such plies beyond the action and
// the reply (Greedy's own follow-up, or the opponent's: an extra turn or a
// pending choice after its reply). Where a line stops with the same player
// to move again, that player gets Weights.Tempo.
//
// Before that search, Greedy checks for a sure win: an action that wins at
// once, or through a choice of its own that it opens, using only public
// information. It plays such a win without searching (see sureWin).
//
// The value of an action is its mean over the determinizations. The
// deterministic Greedy (Temperature 0, the tournament bot) plays the action
// of highest value, the first one in legal order on a tie. Easy samples an
// action with probability proportional to exp((value - best) /
// Temperature), so it usually plays well but sometimes errs, more often when
// the alternatives are close. Each reply considered means the opponent's
// choices, and the cards a move uncovers for it, are part of the value; the
// determinizations average over the face-down cards a move may reveal.
//
// The work per iteration is about (actions) x (replies) evaluations, a few
// thousand at most, and the memory is a few fixed-size buffers: nothing
// grows with the budget.

// Defaults of the Greedy bot.
const (
	DefaultGreedyK   = 3   // determinizations per move
	EasyTemperature  = 1.5 // the Easy level's softmax temperature, in VP
	greedyExtraPlies = 1   // plies a line may add for extra turns and pending choices
	greedyMaxDepth   = 2 + greedyExtraPlies
)

// GreedyConfig configures a Greedy bot. The zero value is the deterministic
// tournament Greedy with DefaultGreedyK determinizations and the default
// weights.
type GreedyConfig struct {
	K           int      // determinizations per move when the budget sets no MaxIterations; 0 means DefaultGreedyK
	Temperature float64  // softmax temperature in VP; 0 plays the best action (deterministic Greedy)
	Weights     *Weights // the evaluation's weights; nil means DefaultWeights
}

// GreedyBot is the Greedy bot (and, with a temperature, the Easy level). It
// holds no mutable state, so one bot may serve several goroutines.
type GreedyBot struct {
	seed uint64
	k    int
	temp float64
	w    Weights
}

// NewGreedy returns a Greedy bot seeded with seed.
func NewGreedy(seed uint64, cfg GreedyConfig) *GreedyBot {
	g := &GreedyBot{seed: seed, k: cfg.K, temp: cfg.Temperature, w: DefaultWeights()}
	if g.k <= 0 {
		g.k = DefaultGreedyK
	}
	if g.temp < 0 {
		g.temp = 0
	}
	if cfg.Weights != nil {
		g.w = *cfg.Weights
	}
	return g
}

// Choose picks an action as described at the top of this file. An
// iteration of the budget is one determinization; Workers is ignored.
func (g *GreedyBot) Choose(ctx context.Context, obs engine.Observation, b Budget, onProgress func(Progress)) (engine.Action, error) {
	var rootBuf [engine.MaxLegalActions]engine.Action
	legal, err := choiceSetup(&obs, rootBuf[:0])
	if err != nil {
		return engine.Action{}, err
	}
	if len(legal) == 1 {
		return legal[0], nil
	}
	start := time.Now()
	k := g.k
	if b.MaxIterations > 0 {
		k = b.MaxIterations
	}
	rng := callRNG(g.seed, &obs)
	sr := greedySearch{me: obs.Seat(), w: &g.w, ctx: ctx}
	if b.MaxTime > 0 {
		sr.deadline = start.Add(b.MaxTime)
	}
	first := obs.Determinize(&rng)
	if i, ok := sr.sureWin(&first, legal); ok {
		return legal[i], nil
	}

	n := len(legal)
	var sums, vals [engine.MaxLegalActions]float64
	var firsts [engine.MaxLegalActions]int // iterations in which each action was the best
	done, partial := 0, 0
	lastProgress := start
	report := func(final bool) {
		if onProgress == nil {
			return
		}
		now := time.Now()
		if !final && now.Sub(lastProgress) < progressEvery {
			return
		}
		lastProgress = now
		best := g.best(sums[:n], vals[:partial], done)
		share := 0.0
		if done > 0 {
			share = float64(firsts[best]) / float64(done)
		}
		onProgress(Progress{Iterations: done, Elapsed: now.Sub(start), Best: legal[best], BestShare: share})
	}

	for it := 0; it < k && !sr.stopped; it++ {
		d := first
		if it > 0 {
			d = obs.Determinize(&rng)
		}
		partial = 0
		for i, a := range legal {
			c := d
			engine.ApplyTrusted(&c, a)
			v := sr.mine(&c, 0, greedyExtraPlies)
			if sr.stopped {
				break
			}
			vals[i] = v
			partial = i + 1
			report(false)
		}
		if sr.stopped {
			break
		}
		for i := range n {
			sums[i] += vals[i]
		}
		firsts[argmax(vals[:n])]++
		done++
		partial = 0
	}
	if done > 0 {
		partial = 0 // a cut-short iteration is dropped when a full one exists
	}
	report(true)

	if g.temp > 0 && (done > 0 || partial > 0) {
		return legal[g.sample(&rng, sums[:n], vals[:partial], done)], nil
	}
	return legal[g.best(sums[:n], vals[:partial], done)], nil
}

// best returns the index of the best action so far: by the sums of the
// finished iterations, or, if none finished, by the values of the actions
// the first iteration reached (partial), or else the first action.
func (g *GreedyBot) best(sums, partial []float64, done int) int {
	switch {
	case done > 0:
		return argmax(sums)
	case len(partial) > 0:
		return argmax(partial)
	}
	return 0
}

// sample draws an action index by a softmax of the mean values at the
// bot's temperature.
func (g *GreedyBot) sample(rng *engine.RNG, sums, partial []float64, done int) int {
	vals := partial
	scale := 1.0
	if done > 0 {
		vals = sums
		scale = 1 / float64(done)
	}
	top := vals[argmax(vals)] * scale
	var weights [engine.MaxLegalActions]float64
	total := 0.0
	for i, v := range vals {
		weights[i] = math.Exp((v*scale - top) / g.temp)
		total += weights[i]
	}
	u := unitFloat(rng) * total
	for i := range vals {
		u -= weights[i]
		if u < 0 {
			return i
		}
	}
	return len(vals) - 1
}

// argmax returns the index of the largest value, the first one on a tie.
func argmax(vals []float64) int {
	best := 0
	for i, v := range vals {
		if v > vals[best] {
			best = i
		}
	}
	return best
}

// unitFloat returns a uniform float64 in [0, 1).
func unitFloat(r *engine.RNG) float64 {
	return float64(r.Uint64()>>11) / (1 << 53)
}

// greedySearch is the state of one Greedy search: the evaluation, the
// stopping rule and one legal-action buffer per depth.
type greedySearch struct {
	me       engine.Seat
	w        *Weights
	ctx      context.Context
	deadline time.Time // zero: none
	evals    int
	stopped  bool
	bufs     [greedyMaxDepth][engine.MaxLegalActions]engine.Action
}

// mine values c, reached by one of Greedy's own actions. extra is how many
// more plies the line may add.
func (sr *greedySearch) mine(c *engine.State, depth, extra int) float64 {
	if c.Over() {
		return sr.leaf(c, 0)
	}
	if c.ToMove != sr.me {
		return sr.reply(c, depth, extra)
	}
	if extra == 0 {
		return sr.leaf(c, sr.w.Tempo)
	}
	best := math.Inf(-1)
	legal := engine.LegalActions(c, sr.bufs[depth][:0])
	for _, a := range legal {
		x := *c
		engine.ApplyTrusted(&x, a)
		v := sr.mine(&x, depth+1, extra-1)
		if sr.stopped {
			return 0
		}
		best = max(best, v)
		if sr.won(&x, sr.me) {
			break
		}
	}
	return best
}

// reply values c, in which the opponent is to move: the minimum over its
// replies.
func (sr *greedySearch) reply(c *engine.State, depth, extra int) float64 {
	worst := math.Inf(1)
	opp := sr.me.Opp()
	legal := engine.LegalActions(c, sr.bufs[depth][:0])
	for _, a := range legal {
		x := *c
		engine.ApplyTrusted(&x, a)
		var v float64
		switch {
		case x.Over() || x.ToMove == sr.me:
			v = sr.leaf(&x, 0)
		case extra > 0:
			v = sr.reply(&x, depth+1, extra-1)
		default:
			v = sr.leaf(&x, -sr.w.Tempo)
		}
		if sr.stopped {
			return 0
		}
		worst = min(worst, v)
		if sr.won(&x, opp) {
			break
		}
	}
	return worst
}

// sureWin looks for a root action that wins whatever the hidden information:
// one that ends the game with Greedy's victory at once, or opens a choice of
// Greedy's own (before any card is revealed, a new age is dealt or the Great
// Library draws) in which some option wins. Such a win depends only on
// public information, so one determinization d proves it. It returns the
// action's index in legal.
//
// Without this check a sure win could lose to a line that wins only in the
// sampled determinization (an extra turn that happens to reveal the card
// the line needs), with the sample's luck counted as certainty.
func (sr *greedySearch) sureWin(d *engine.State, legal []engine.Action) (int, bool) {
	for i, a := range legal {
		x := *d
		engine.ApplyTrusted(&x, a)
		if sr.won(&x, sr.me) {
			return i, true
		}
		if x.Over() || x.ToMove != sr.me || !sameHidden(d, &x) {
			continue
		}
		for _, b := range engine.LegalActions(&x, sr.bufs[0][:0]) {
			y := x
			engine.ApplyTrusted(&y, b)
			if sr.won(&y, sr.me) {
				return i, true
			}
		}
	}
	return 0, false
}

// sameHidden reports whether going from state a to state b used no hidden
// information: no card was revealed, no age was dealt and no Great Library
// offer was drawn.
func sameHidden(a, b *engine.State) bool {
	return a.Age == b.Age && faceDown(a) == faceDown(b) && b.Phase != engine.PhasePickLibraryToken
}

// faceDown counts the face-down cards of the structure.
func faceDown(s *engine.State) int {
	n := 0
	for i := range s.Slots {
		if s.Slots[i].Card != 0 && !s.Slots[i].FaceUp {
			n++
		}
	}
	return n
}

// won reports whether x is a finished game won by seat.
func (sr *greedySearch) won(x *engine.State, seat engine.Seat) bool {
	return x.Over() && x.Result.Winner == int8(seat)
}

// leaf evaluates x for Greedy, plus a tempo bonus, and checks every
// checkEvery evaluations whether the search must stop.
func (sr *greedySearch) leaf(x *engine.State, tempo float64) float64 {
	sr.evals++
	if sr.evals%checkEvery == 0 {
		sr.checkStop()
	}
	v := sr.w.Evaluate(x, sr.me)
	if !x.Over() {
		v += tempo
	}
	return v
}

// checkStop sets stopped when the context is done or the deadline passed.
func (sr *greedySearch) checkStop() {
	if sr.ctx.Err() != nil || (!sr.deadline.IsZero() && time.Now().After(sr.deadline)) {
		sr.stopped = true
	}
}
