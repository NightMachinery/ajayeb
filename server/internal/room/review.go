package room

import (
	"context"
	"math"
	"slices"
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
// The game's hint settings choose how hard the review looks (the advisor
// strength, view.AdvisorStrength):
//
//   - quick: Greedy's ranking (ai.RankActions: one ply, the opponent's
//     best reply) names the better move, by a lead of at least
//     reviewMinGap of its victory-point equivalents.
//   - two-stage, the default: Greedy proposes the better move as quick
//     does, and a fixed-budget tree search of the Scholar's kind
//     (ai.CompareActions, SO-ISMCTS on one worker) plays the proposed move
//     and the move played against each other. The review names the
//     proposed move only when the search also rates it clearly better, by
//     reviewSearchMinGap of its mean reward; otherwise it says no clearly
//     better move was found.
//   - deep: the tree search ranks every move (ai.SearchActions), and its
//     most visited move, when it is not the move played, is checked the
//     same way.
//
// A reason that is a sure fact about the next moves (exactReason: the
// better move wins at once, or the move played left the opponent a win)
// needs no confirmation: the facts are read from what the mover saw, and
// the search cannot tell such a move from one that merely leads to a
// likely win, since its reward is near 1 for both (a pawn one space from
// the capital).
//
// The points figure (Gap) is always Greedy's: a deep review has none, and
// the client words a search's verdict without a number.
//
// It uses only what the mover knew when moving. Every ranking and search
// runs on the mover's Observation of the position before the move, which
// samples the hidden cards, and the reason comes from engine.ReviewFacts on
// one such sample, which reads only the cards face up before the move. So
// the review is the same on every state the mover could not tell apart
// from the real one (TestReviewLeakGuard, at every strength), and it never
// names a card the move turned over.
//
// It is deterministic for the room's seed, the action log and the ply: the
// seeds come from reviewSeed, Greedy's budget is a count of evaluations
// (reviewRank) and the search's a count of iterations on one worker
// (ReviewBudget). The clocks are safety nets: a ranking or search they cut
// short is dropped and the move gets no review, never a guessed one.
//
// A quick review costs a few milliseconds, so the room actor computes it
// in place, under reviewTimeout. A two-stage or deep review runs a search
// of tens to hundreds of milliseconds, so it runs off the actor (a
// reviewJob on a goroutine of its own, waiting for one of the Manager's
// review slots, which bound how many run at once across every room), and
// its result comes back through the inbox. The room then sends the
// mover's connections a fresh state frame that carries it.
//
// The room keeps, for each seat, the play actions of its last turn and the
// position before each one (reviewSlot), from the commit or, after a load
// or an undo, from a replay of the log. When a wonder grants an extra turn,
// the follow-up play action is appended to the same turn rather than
// overwriting the wonder. The review itself is computed, or started, the
// first time a view of that seat needs it, while the advisor family is on,
// with the strength the settings have then, and kept until the seat's next
// turn.

// The review's parameters.
const (
	// reviewMinGap is the smallest lead, in the evaluation's victory-point
	// equivalents, that makes another move clearly better.
	reviewMinGap = 3.0
	// reviewNear is how close to the best move another must be for the
	// review to name it instead, when it has a clearer reason.
	reviewNear = 1.0
	// reviewTimeout bounds the wall clock of one quick review. The
	// evaluation budget keeps a review to a few milliseconds; this only
	// guards a starved machine.
	reviewTimeout = 50 * time.Millisecond
	// reviewSearchMinGap is the smallest lead, in the tree search's mean
	// reward (between 0 and 1), by which the search must rate the better
	// move above the move played. Near an even game it is about 1.3 of the
	// evaluation's victory points at the rollouts' horizon, and several
	// times the comparison's sampling noise at the default budget
	// (docs/ai.md).
	reviewSearchMinGap = 0.04
	// reviewSearchTimeout bounds the wall clock of one step of a two-stage
	// or deep review, the wait for a review slot included. The iteration
	// budget keeps a review well under a second; this only guards a
	// starved or crowded server.
	reviewSearchTimeout = 10 * time.Second
)

// reviewRank is the ranking's budget: DefaultRankK determinizations, and
// at most 120,000 evaluations (BenchmarkReview: a few milliseconds on a
// laptop).
var reviewRank = ai.RankConfig{K: ai.DefaultRankK, MaxEvals: 120_000}

// ReviewBudget is the tree search's budget in the two-stage and deep
// reviews: iterations on one worker, with no clock.
type ReviewBudget struct {
	// Compare is the comparison of the better move with the move played
	// (two-stage and deep), shared equally between the two.
	Compare int
	// Search is the deep review's search over every move.
	Search int
}

// DefaultReviewBudget is the budget a Manager uses when its Config sets
// none. docs/ai.md has its measurement.
var DefaultReviewBudget = ReviewBudget{Compare: 6_000, Search: 20_000}

// reviewConfig is how one review is computed.
type reviewConfig struct {
	strength view.AdvisorStrength
	rank     ai.RankConfig
	budget   ReviewBudget
}

// The seed tags of the review's searches, mixed into reviewSeed.
const (
	reviewCompareTag = 1
	reviewSearchTag  = 2
)

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
// when there is none to show: no ranking, or a budget that ran out). While
// a review job runs for it, pending is set, gen names the job and cancel
// stops it.
type reviewSlot struct {
	set     bool
	pre     engine.State
	action  engine.Action
	steps   []reviewStep
	done    bool
	result  *view.ReviewView
	pending bool
	gen     uint64
	cancel  context.CancelFunc
}

// stop cancels the slot's review job, if one is running; its result, if
// it still comes, no longer matches the slot.
func (sl *reviewSlot) stop() {
	if sl.cancel != nil {
		sl.cancel()
	}
	sl.pending, sl.cancel = false, nil
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
// otherwise a starts a new turn. Either way a review job still running
// for the slot is stopped.
func (rs *reviewState) recordMove(pre *engine.State, a engine.Action) {
	if !reviewable(pre, a) {
		return
	}
	sl := &rs.seats[pre.ToMove]
	sl.stop()
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

// forgetReviews drops the reviews and stops their jobs, after an undo took
// moves back; the next view that needs one replays the shortened log.
func (r *Room) forgetReviews() {
	for i := range r.reviews.seats {
		r.reviews.seats[i].stop()
	}
	r.reviews = reviewState{}
}

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

// reviewConfig returns how the room reviews a move now: with the game's
// advisor strength and the Manager's search budget.
func (r *Room) reviewConfig() reviewConfig {
	st := r.file.Hints.AdvisorStrength
	if !st.Valid() {
		st = DefaultAdvisorStrength
	}
	return reviewConfig{strength: st, rank: reviewRank, budget: r.m.cfg.ReviewBudget}
}

// review returns the review of seat's last turn of play actions, or nil
// when there is none, or none yet. The first call computes a quick review
// in place, and starts a two-stage or deep one off the actor (startReview)
// once the seat's turn is over: a search started while a wonder's extra
// turn is still to come would only be cancelled by it.
func (r *Room) review(seat engine.Seat) *view.ReviewView {
	if !r.reviews.loaded {
		r.loadReviews()
	}
	sl := &r.reviews.seats[seat]
	if !sl.set {
		return nil
	}
	if !sl.done && !sl.pending {
		cfg := r.reviewConfig()
		if cfg.strength != view.AdvisorStrengthQuick {
			if r.turnOver(seat) {
				r.startReview(seat, cfg)
			}
			return nil
		}
		sl.done = true
		ctx, cancel := context.WithTimeout(r.ctx, reviewTimeout*time.Duration(max(1, len(sl.steps))))
		rv, ok := buildTurnReview(ctx, sl.steps, r.file.Seed, cfg)
		cancel()
		if ok {
			sl.result = &rv
		}
	}
	return sl.result
}

// turnOver reports whether seat's last turn is over: the game ended, the
// other seat is to move, or seat is to move in a turn of its own that no
// extra turn granted (the other seat has moved since, or a new age began).
func (r *Room) turnOver(seat engine.Seat) bool {
	return r.state.Over() || r.state.ToMove != seat || !r.state.PlayingAgain
}

// prefetchReviews starts the two-stage or deep review of every seat whose
// turn is over and whose review is not started yet, while the advisor is
// on, so that it is usually ready by the time that seat's view needs it.
// In a hot-seat room that view comes only after the other seat's move; in
// the other rooms the mover's own frame would start it anyway.
func (r *Room) prefetchReviews() {
	if !r.reviews.loaded || !r.file.Hints.Advisor {
		return
	}
	cfg := r.reviewConfig()
	if cfg.strength == view.AdvisorStrengthQuick {
		return
	}
	for seat := range engine.Seat(2) {
		sl := &r.reviews.seats[seat]
		if sl.set && !sl.done && !sl.pending && r.turnOver(seat) && !r.guidedLastMove(seat) {
			r.startReview(seat, cfg)
		}
	}
}

// reviewJob is one two-stage or deep review, with everything it needs
// copied out of the room so that its goroutine never touches room state.
type reviewJob struct {
	seat     engine.Seat
	gen      uint64
	steps    []reviewStep
	roomSeed uint64
	cfg      reviewConfig
	slots    chan struct{} // the Manager's review slots: a send takes one
	inbox    chan<- any
	done     <-chan struct{}
}

// startReview starts the review of seat's last turn on a goroutine of its
// own, under a context that the slot's stop cancels (a newer move, an
// undo) and that ends with the room.
func (r *Room) startReview(seat engine.Seat, cfg reviewConfig) {
	sl := &r.reviews.seats[seat]
	ctx, cancel := context.WithTimeout(r.ctx, reviewSearchTimeout*time.Duration(len(sl.steps)))
	r.reviewGen++
	sl.pending, sl.gen, sl.cancel = true, r.reviewGen, cancel
	job := reviewJob{
		seat:     seat,
		gen:      r.reviewGen,
		steps:    slices.Clone(sl.steps),
		roomSeed: r.file.Seed,
		cfg:      cfg,
		slots:    r.m.reviewSlots,
		inbox:    r.inbox,
		done:     r.done,
	}
	r.m.wg.Add(1)
	go func() {
		defer r.m.wg.Done()
		defer cancel()
		job.run(ctx)
	}()
}

// run waits for a review slot, reviews the turn and posts the result. A
// job whose context ends first posts a review that says nothing; the room
// drops it if the slot has moved on.
func (j reviewJob) run(ctx context.Context) {
	var rv view.ReviewView
	ok := false
	select {
	case j.slots <- struct{}{}:
		rv, ok = buildTurnReview(ctx, j.steps, j.roomSeed, j.cfg)
		<-j.slots
	case <-ctx.Done():
	}
	select {
	case j.inbox <- msgReviewResult{seat: j.seat, gen: j.gen, review: rv, ok: ok}:
	case <-j.done:
	}
}

// reviewResult takes a review job's result if it is the one the seat's
// slot is waiting for, and sends the seat's own connections a fresh state
// frame that carries it. Any other result is stale (a newer move or an
// undo stopped its job) and dropped.
func (r *Room) reviewResult(m msgReviewResult) {
	if !r.reviews.loaded {
		return
	}
	sl := &r.reviews.seats[m.seat]
	if !sl.pending || sl.gen != m.gen {
		return
	}
	sl.pending, sl.cancel, sl.done = false, nil, true
	if !m.ok {
		return
	}
	rv := m.review
	sl.result = &rv
	if !r.file.Hints.Advisor {
		return
	}
	var fc frameCache
	presence := r.presence()
	for c, mem := range r.clients {
		if seat, ok := r.viewer(mem).Seat(); ok && seat == m.seat {
			c.Out.Push(r.frame(mem, nil, nil, presence, &fc, false))
		}
	}
}

// buildTurnReview reviews the play actions of one turn in order, each from
// the mover's observation before that step. Played lists every step's
// action key (space-separated when an extra turn gave more than one move).
// When any step had a clearly better move, the step with the strongest
// reason (and largest gap on ties) is reported; for a later step in a
// multi-step turn, Better prefixes the earlier steps' played keys so the
// client can tell which step of the turn the advice is about.
func buildTurnReview(ctx context.Context, steps []reviewStep, roomSeed uint64, cfg reviewConfig) (view.ReviewView, bool) {
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
		return view.ReviewView{Ply: lastPly, Played: played, Reason: view.ReviewReasonNone, Strength: cfg.strength}, true
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
// pre, from that seat's observation of pre only, at cfg's strength. It
// reports false when it has nothing to say: a ranking or search failed or
// ran out of time.
func buildReview(ctx context.Context, pre *engine.State, played engine.Action, seed uint64, cfg reviewConfig) (view.ReviewView, bool) {
	if !reviewable(pre, played) {
		return view.ReviewView{}, false
	}
	var rv view.ReviewView
	ok := false
	switch cfg.strength {
	case view.AdvisorStrengthDeep:
		rv, ok = deepReview(ctx, pre, played, seed, cfg.budget)
	case view.AdvisorStrengthTwoStage:
		var better engine.Action
		rv, better, ok = greedyReview(ctx, pre, played, seed, cfg.rank)
		if ok && rv.Better != nil && !exactReason(rv.Reason) {
			obs := engine.Observe(pre, pre.ToMove)
			confirmed, err := searchConfirms(ctx, obs, seed, better, played, cfg.budget.Compare)
			switch {
			case err != nil:
				ok = false
			case !confirmed:
				rv = view.ReviewView{Ply: rv.Ply, Played: rv.Played, Reason: view.ReviewReasonNone}
			}
		}
	default:
		rv, _, ok = greedyReview(ctx, pre, played, seed, cfg.rank)
	}
	if !ok {
		return view.ReviewView{}, false
	}
	rv.Strength = cfg.strength
	if rv.Strength == "" {
		rv.Strength = view.AdvisorStrengthQuick
	}
	return rv, true
}

// searchConfirms reports whether the tree search, comparing better with
// played on obs (ai.CompareActions, with the seed's comparison tag),
// rates better clearly higher: by at least reviewSearchMinGap of mean
// reward.
func searchConfirms(ctx context.Context, obs engine.Observation, seed uint64, better, played engine.Action, iterations int) (bool, error) {
	cmp, err := ai.CompareActions(ctx, obs, ai.MixSeed(seed, reviewCompareTag), better, played, iterations)
	if err != nil {
		return false, err
	}
	return cmp.Values[0]-cmp.Values[1] >= reviewSearchMinGap, nil
}

// deepReview is the deep review of played in pre: the tree search's most
// visited move, when it is not the move played and the comparison rates
// it clearly better, with the reason the facts give, and no points.
func deepReview(ctx context.Context, pre *engine.State, played engine.Action, seed uint64, budget ReviewBudget) (view.ReviewView, bool) {
	obs := engine.Observe(pre, pre.ToMove)
	rv := view.ReviewView{Ply: int(pre.Ply), Played: played.Key(), Reason: view.ReviewReasonNone}
	rng := engine.NewRNG(seed ^ 0x6661637473)
	d := obs.Determinize(&rng)
	if _, wins := engine.WinsNow(&d, played, engine.FaceUpSlots(&d)); wins {
		return rv, true
	}
	sr, err := ai.SearchActions(ctx, obs, ai.MixSeed(seed, reviewSearchTag), budget.Search)
	if err != nil {
		return view.ReviewView{}, false
	}
	bi := sr.Best()
	if bi < 0 || sr.Index(played) < 0 {
		return view.ReviewView{}, false
	}
	better := sr.Actions[bi]
	if better == played {
		return rv, true
	}
	fb := engine.ReviewFacts(&d, better)
	fp := engine.ReviewFacts(&d, played)
	reason := reviewReason(&fb, &fp)
	if !exactReason(reason) {
		confirmed, err := searchConfirms(ctx, obs, seed, better, played, budget.Compare)
		if err != nil {
			return view.ReviewView{}, false
		}
		if !confirmed {
			return rv, true
		}
	}
	setReason(&rv, better.Key(), reason, &fb, &fp)
	return rv, true
}

// exactReason reports whether reason r is a sure fact about the next
// moves, read from what the mover saw, which the search need not confirm:
// the better move wins, or the move played left the opponent a win that
// the better move does not.
func exactReason(r view.ReviewReason) bool {
	return r == view.ReviewReasonWins || r == view.ReviewReasonStopsWin
}

// greedyReview is the quick review of played in pre, from Greedy's
// ranking of the mover's observation, and the better move it names (the
// zero action when none). It reports false when the ranking failed or ran
// out of time.
func greedyReview(ctx context.Context, pre *engine.State, played engine.Action, seed uint64, cfg ai.RankConfig) (view.ReviewView, engine.Action, bool) {
	seat := pre.ToMove
	obs := engine.Observe(pre, seat)
	rv := view.ReviewView{Ply: int(pre.Ply), Played: played.Key(), Reason: view.ReviewReasonNone}
	rk, err := ai.RankActions(ctx, obs, seed, cfg)
	if err != nil {
		return view.ReviewView{}, engine.Action{}, false
	}
	pi := -1
	for i, a := range rk.Actions {
		if a.Key() == rv.Played {
			pi = i
			break
		}
	}
	if pi < 0 {
		return view.ReviewView{}, engine.Action{}, false
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
		return rv, engine.Action{}, true
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
	setReason(&rv, rk.Actions[pick].Key(), pickReason, &pickFacts, &fp)
	rv.Gap = max(1, int(math.Round(rk.Values[pick]-rk.Values[pi])))
	return rv, rk.Actions[pick], true
}

// setReason names the better move with key in rv, why it is better, and
// that reason's details, from the better move's facts fb and the played
// move's fp.
func setReason(rv *view.ReviewView, key string, reason view.ReviewReason, fb, fp *engine.MoveFacts) {
	rv.Better = &key
	rv.Reason = reason
	switch reason {
	case view.ReviewReasonStopsWin:
		rv.Threat = fp.OppWin
	case view.ReviewReasonPair, view.ReviewReasonDeniesPair:
		sym := view.ScienceSymbol(fb.Symbol.String())
		rv.Symbol = &sym
	case view.ReviewReasonLoot:
		coins := fb.Loot
		rv.Coins = &coins
	}
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
