package room

import (
	"context"
	"math"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// This file is the move review (the advisor hint family; the hints design,
// owner answer Q2): after a seat's play action, whether a clearly better
// one existed in that position, and why, for that seat only. There is no
// way to ask before moving: the review speaks only of a move already made,
// and the client shows it inside the summary of the opponent's next move.
//
// It uses only what the mover knew when moving. The actions are ranked
// with ai.RankActions on the mover's Observation of the position before
// the move, which samples the hidden cards, and the reason comes from
// engine.ReviewFacts on one such sample, which reads only the cards face up
// before the move. So the review is the same on every state the mover
// could not tell apart from the real one (TestReviewLeakGuard), and it
// never names a card the move turned over.
//
// It is deterministic for the room's seed, the action log and the ply: the
// ranking's seed is reviewSeed, and its budget is a count of evaluations
// (reviewRank). The only clock is reviewTimeout, a safety net so that the
// room actor never stalls: a ranking it cuts short is dropped and the move
// gets no review, never a guessed one.
//
// The room keeps, for each seat, the position before its last play action
// and that action (reviewSlot), from the commit or, after a load or an
// undo, from a replay of the log. The review itself is computed the first
// time a view of that seat needs it, while the advisor family is on, and
// kept until the seat's next play action.

// The review's parameters.
const (
	// reviewMinGap is the smallest lead, in the evaluation's victory-point
	// equivalents, that makes another move clearly better.
	reviewMinGap = 3.0
	// reviewNear is how close to the best move another must be for the
	// review to name it instead, when it has a clearer reason.
	reviewNear = 1.0
	// reviewTimeout bounds the wall clock of one review. The evaluation
	// budget keeps a review to a few milliseconds; this only guards a
	// starved machine.
	reviewTimeout = 50 * time.Millisecond
)

// reviewRank is the ranking's budget: DefaultRankK determinizations, and
// at most 120,000 evaluations (BenchmarkReview: a few milliseconds on a
// laptop).
var reviewRank = ai.RankConfig{K: ai.DefaultRankK, MaxEvals: 120_000}

// reviewSeed is the seed of the review of the move at ply: a function of
// the room's seed and the position only.
func reviewSeed(roomSeed uint64, ply uint16) uint64 {
	return ai.MixSeed(roomSeed^0x7265766965770000, uint64(ply))
}

// reviewSlot is one seat's last play action: the position it was made in
// and the action, and its review once computed (nil when there is none to
// show: no ranking, or a budget that ran out).
type reviewSlot struct {
	set    bool
	pre    engine.State
	action engine.Action
	done   bool
	result *view.ReviewView
}

// reviewState is the room's reviews. Until loaded is set, seats is not
// known: a loaded room and an undo leave it unset, and the first view that
// needs a review replays the log (loadReviews).
type reviewState struct {
	loaded bool
	seats  [2]reviewSlot
}

// reviewable reports whether a, taken in s, is a move the review speaks
// of: a play action (build, discard or wonder) of the seat to move.
func reviewable(s *engine.State, a engine.Action) bool {
	if s.Over() || s.Phase != engine.PhasePlay {
		return false
	}
	switch a.Type {
	case engine.ActConstructBuilding, engine.ActDiscardBuilding, engine.ActConstructWonder:
		return true
	}
	return false
}

// noteMove records action a, about to be applied in pre, as its seat's
// last play action when it is one. It is a no-op while the reviews are not
// loaded: the replay that loads them will see the move in the log.
func (r *Room) noteMove(pre *engine.State, a engine.Action) {
	if !r.reviews.loaded || !reviewable(pre, a) {
		return
	}
	r.reviews.seats[pre.ToMove] = reviewSlot{set: true, pre: *pre, action: a}
}

// forgetReviews drops the reviews, after an undo took moves back; the next
// view that needs one replays the shortened log.
func (r *Room) forgetReviews() { r.reviews = reviewState{} }

// loadReviews finds each seat's last play action by replaying the log.
func (r *Room) loadReviews() {
	r.reviews = reviewState{loaded: true}
	s, err := startState(r.file)
	if err != nil {
		return
	}
	for _, key := range r.file.Actions {
		a, ok := actionByKey(&s, key)
		if !ok {
			return
		}
		if reviewable(&s, a) {
			r.reviews.seats[s.ToMove] = reviewSlot{set: true, pre: s, action: a}
		}
		if engine.Apply(&s, a, nil) != nil {
			return
		}
	}
}

// review returns the review of seat's last play action, computing it the
// first time, or nil when there is none.
func (r *Room) review(seat engine.Seat) *view.ReviewView {
	if !r.reviews.loaded {
		r.loadReviews()
	}
	sl := &r.reviews.seats[seat]
	if !sl.set {
		return nil
	}
	if !sl.done {
		sl.done = true
		ctx, cancel := context.WithTimeout(r.ctx, reviewTimeout)
		rv, ok := buildReview(ctx, &sl.pre, sl.action, reviewSeed(r.file.Seed, sl.pre.Ply), reviewRank)
		cancel()
		if ok {
			sl.result = &rv
		}
	}
	return sl.result
}

// buildReview reviews action played, a play action of the seat to move in
// pre, from that seat's observation of pre only. It reports false when it
// has nothing to say: the ranking failed or ran out of time.
func buildReview(ctx context.Context, pre *engine.State, played engine.Action, seed uint64, cfg ai.RankConfig) (view.ReviewView, bool) {
	if !reviewable(pre, played) {
		return view.ReviewView{}, false
	}
	seat := pre.ToMove
	obs := engine.Observe(pre, seat)
	rv := view.ReviewView{Ply: int(pre.Ply), Played: played.Key(), Reason: view.ReviewReasonNone}
	rk, err := ai.RankActions(ctx, obs, seed, cfg)
	if err != nil {
		return view.ReviewView{}, false
	}
	pi := -1
	for i, a := range rk.Actions {
		if a.Key() == rv.Played {
			pi = i
			break
		}
	}
	if pi < 0 {
		return view.ReviewView{}, false
	}
	bi := rk.Best()
	win := ai.DefaultWeights().Win
	if cfg.Weights != nil {
		win = cfg.Weights.Win
	}
	// Nothing to learn when the move was as good as any, when another is
	// better by less than the noise, or in a decided position: the move
	// keeps a won game won, or every move loses.
	if bi == pi || rk.Values[bi]-rk.Values[pi] < reviewMinGap || rk.Values[pi] >= win/2 || rk.Values[bi] <= -win/2 {
		return rv, true
	}

	// The reason, from the facts that tell a better move from the one
	// played, read on a determinization of the observation (so on what the
	// mover knew). Among the moves that are clearly better than the one
	// played and within reviewNear of the best (or that win for sure), the
	// one with the strongest reason is named: the ranking cannot tell such
	// moves apart, and a reason teaches more than a number.
	rng := engine.NewRNG(seed ^ 0x6661637473)
	d := obs.Determinize(&rng)
	fp := engine.ReviewFacts(&d, played)
	pick, pickReason := bi, view.ReviewReasonPoints
	var pickFacts engine.MoveFacts
	first := true
	mask := engine.FaceUpSlots(&d)
	for i, a := range rk.Actions {
		if i == pi || rk.Values[i]-rk.Values[pi] < reviewMinGap {
			continue
		}
		if rk.Values[i] < rk.Values[bi]-reviewNear {
			// Far from the best only a sure win is named, which the
			// ranking may value a little below a win through a sampled card.
			if _, wins := engine.WinsNow(&d, a, mask); !wins {
				continue
			}
		}
		fb := engine.ReviewFacts(&d, a)
		r := reviewReason(&fb, &fp)
		if first || reasonRank(r) < reasonRank(pickReason) || (r == pickReason && rk.Values[i] > rk.Values[pick]) {
			pick, pickReason, pickFacts, first = i, r, fb, false
		}
	}
	key := rk.Actions[pick].Key()
	rv.Better = &key
	rv.Gap = max(1, int(math.Round(rk.Values[pick]-rk.Values[pi])))
	rv.Reason = pickReason
	switch pickReason {
	case view.ReviewReasonStopsWin:
		rv.Threat = fp.OppWin
	case view.ReviewReasonPair, view.ReviewReasonDeniesPair:
		sym := view.ScienceSymbol(pickFacts.Symbol.String())
		rv.Symbol = &sym
	case view.ReviewReasonLoot:
		coins := pickFacts.Loot
		rv.Coins = &coins
	}
	return rv, true
}

// reviewReason is why a move with facts fb is better than the move played,
// with facts fp: the first fact, in order of priority, that holds for the
// one and not for the other, or points when none does.
func reviewReason(fb, fp *engine.MoveFacts) view.ReviewReason {
	switch {
	case fb.Wins && !fp.Wins:
		return view.ReviewReasonWins
	case fp.OppKnown && fp.OppWin != nil && fb.OppKnown && fb.OppWin == nil:
		return view.ReviewReasonStopsWin
	case fb.Pair && !fp.Pair:
		return view.ReviewReasonPair
	case fb.DeniesPair && !fp.DeniesPair:
		return view.ReviewReasonDeniesPair
	case fb.Loot > fp.Loot:
		return view.ReviewReasonLoot
	}
	return view.ReviewReasonPoints
}

// reasonRank orders the reasons, the strongest first.
func reasonRank(r view.ReviewReason) int {
	switch r {
	case view.ReviewReasonWins:
		return 0
	case view.ReviewReasonStopsWin:
		return 1
	case view.ReviewReasonPair:
		return 2
	case view.ReviewReasonDeniesPair:
		return 3
	case view.ReviewReasonLoot:
		return 4
	}
	return 5
}
