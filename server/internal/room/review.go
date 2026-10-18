package room

import (
	"context"
	"math"
	"strings"
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
// The room keeps, for each seat, the play actions of its last turn and the
// position before each one (reviewSlot), from the commit or, after a load
// or an undo, from a replay of the log. When a wonder grants an extra turn,
// the follow-up play action is appended to the same turn rather than
// overwriting the wonder. The review itself is computed the first time a
// view of that seat needs it, while the advisor family is on, and kept
// until the seat's next turn.

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

// reviewStep is one play action of a seat's turn and the position it was
// made in. A normal turn has one step; a play-again wonder turn has two
// (or more, when chained).
type reviewStep struct {
	pre    engine.State
	action engine.Action
}

// reviewSlot is one seat's last turn of play actions: its latest position
// and action, every step of the turn, and its review once computed (nil
// when there is none to show: no ranking, or a budget that ran out).
type reviewSlot struct {
	set    bool
	pre    engine.State
	action engine.Action
	steps  []reviewStep
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

// recordMove records action a, about to be applied in pre, in its seat's
// review slot when it is a play action. When pre is an extra turn of the
// same seat in the same age, the step is appended to the seat's current
// turn so the wonder that granted the extra turn is not overwritten;
// otherwise a starts a new turn.
func (rs *reviewState) recordMove(pre *engine.State, a engine.Action) {
	if !reviewable(pre, a) {
		return
	}
	sl := &rs.seats[pre.ToMove]
	if pre.PlayingAgain && len(sl.steps) > 0 && sl.steps[len(sl.steps)-1].pre.Age == pre.Age {
		sl.pre = *pre
		sl.action = a
		sl.steps = append(sl.steps, reviewStep{pre: *pre, action: a})
		sl.done = false
		sl.result = nil
		return
	}
	*sl = reviewSlot{
		set:    true,
		pre:    *pre,
		action: a,
		steps:  []reviewStep{{pre: *pre, action: a}},
	}
}

// noteMove records action a, about to be applied in pre, as part of its
// seat's last turn when it is a play action. It is a no-op while the
// reviews are not loaded: the replay that loads them will see the move in
// the log.
func (r *Room) noteMove(pre *engine.State, a engine.Action) {
	if !r.reviews.loaded {
		return
	}
	r.reviews.recordMove(pre, a)
}

// forgetReviews drops the reviews, after an undo took moves back; the next
// view that needs one replays the shortened log.
func (r *Room) forgetReviews() { r.reviews = reviewState{} }

// loadReviews finds each seat's last turn of play actions by replaying the log.
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
		r.reviews.recordMove(&s, a)
		if engine.Apply(&s, a, nil) != nil {
			return
		}
	}
}

// review returns the review of seat's last turn of play actions, computing
// it the first time, or nil when there is none.
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
		ctx, cancel := context.WithTimeout(r.ctx, reviewTimeout*time.Duration(max(1, len(sl.steps))))
		rv, ok := buildTurnReview(ctx, sl.steps, r.file.Seed, reviewRank)
		cancel()
		if ok {
			sl.result = &rv
		}
	}
	return sl.result
}

// buildTurnReview reviews the play actions of one turn in order, each from
// the mover's observation before that step. Played lists every step's
// action key (space-separated when an extra turn gave more than one move).
// When any step had a clearly better move, the step with the strongest
// reason (and largest gap on ties) is reported; for a later step in a
// multi-step turn, Better prefixes the earlier steps' played keys so the
// client can tell which step of the turn the advice is about.
func buildTurnReview(ctx context.Context, steps []reviewStep, roomSeed uint64, cfg ai.RankConfig) (view.ReviewView, bool) {
	if len(steps) == 0 {
		return view.ReviewView{}, false
	}
	if len(steps) == 1 {
		s := &steps[0]
		return buildReview(ctx, &s.pre, s.action, reviewSeed(roomSeed, s.pre.Ply), cfg)
	}
	keys := make([]string, len(steps))
	var pick view.ReviewView
	pickStep := -1
	for i := range steps {
		s := &steps[i]
		keys[i] = s.action.Key()
		rv, ok := buildReview(ctx, &s.pre, s.action, reviewSeed(roomSeed, s.pre.Ply), cfg)
		if !ok {
			return view.ReviewView{}, false
		}
		if rv.Better == nil {
			continue
		}
		if pickStep < 0 || reasonRank(rv.Reason) < reasonRank(pick.Reason) ||
			(rv.Reason == pick.Reason && rv.Gap > pick.Gap) {
			pick, pickStep = rv, i
		}
	}
	last := &steps[len(steps)-1]
	lastPly := int(last.pre.Ply)
	played := strings.Join(keys, " ")
	if _, wins := engine.WinsNow(&last.pre, last.action, engine.FaceUpSlots(&last.pre)); wins || pickStep < 0 {
		return view.ReviewView{Ply: lastPly, Played: played, Reason: view.ReviewReasonNone}, true
	}
	pick.Ply = lastPly
	pick.Played = played
	if pickStep > 0 {
		better := strings.Join(append(append([]string(nil), keys[:pickStep]...), *pick.Better), " ")
		pick.Better = &better
	}
	return pick, true
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
		betterPick := first || reasonRank(r) < reasonRank(pickReason)
		if !betterPick && r == pickReason {
			if r == view.ReviewReasonWins && fb.Win.Steps != pickFacts.Win.Steps {
				betterPick = fb.Win.Steps < pickFacts.Win.Steps
			} else {
				betterPick = rk.Values[i] > rk.Values[pick]
			}
		}
		if betterPick {
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
