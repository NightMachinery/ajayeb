package room

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// The move review's own checks (review.go): what it says in positions
// built for one fact, that it is deterministic, the leak guard (the same
// review on every state the mover cannot tell apart from the real one),
// that it never names a card the mover had not seen, and its cost.

// reviewPositions returns play-phase positions from random games, every
// third one, each with a random legal play action to review.
func reviewPositions(tb testing.TB, games int) ([]engine.State, []engine.Action) {
	tb.Helper()
	var states []engine.State
	var acts []engine.Action
	for g := range games {
		seed := uint64(500 + 17*g)
		s := engine.NewGame(seed, engine.Options{FirstPlayer: engine.FirstPlayerRule(g % 3)})
		r := engine.NewRNG(seed)
		pols := [2]enginetest.Policy{enginetest.NewRandomPolicy(seed), enginetest.NewRandomPolicy(seed ^ 0x77)}
		n := 0
		err := enginetest.Playout(&s, pols, 300, func(engine.Action, []engine.Event) error {
			if s.Over() || s.Phase != engine.PhasePlay {
				return nil
			}
			n++
			if n%3 != 0 {
				return nil
			}
			legal := engine.LegalActions(&s, nil)
			states = append(states, s)
			acts = append(acts, legal[r.IntN(len(legal))])
			return nil
		})
		if err != nil {
			tb.Fatalf("game %d: %v", g, err)
		}
	}
	return states, acts
}

// namedBefore returns the named scenario's position before its trigger.
func namedBefore(t *testing.T, name string) engine.State {
	t.Helper()
	sc := engine.NamedScenario(name)
	if sc == nil {
		t.Fatalf("no named scenario %q", name)
	}
	s, err := engine.NewGameFromScenario(3, engine.Options{}, sc)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// mustKey returns the legal action of s with key k.
func mustKey(t *testing.T, s *engine.State, k string) engine.Action {
	t.Helper()
	a, ok := legalByKey(s, k)
	if !ok {
		t.Fatalf("%s is not legal", k)
	}
	return a
}

// quickReview reviews at the quick strength, with the room's budget.
var quickReview = reviewConfig{strength: view.AdvisorStrengthQuick, rank: reviewRank}

// testBudget is a small search budget, so the tests of the two-stage and
// deep strengths stay fast; what they check does not depend on the size.
var testBudget = ReviewBudget{Compare: 400, Search: 800}

// strengthReview reviews at strength st, with the room's ranking and the
// test search budget.
func strengthReview(st view.AdvisorStrength) reviewConfig {
	return reviewConfig{strength: st, rank: reviewRank, budget: testBudget}
}

// searchStrengths are the strengths that run the tree search.
var searchStrengths = []view.AdvisorStrength{view.AdvisorStrengthTwoStage, view.AdvisorStrengthDeep}

// reviewOf reviews key k in s at the quick strength, without a clock.
func reviewOf(t *testing.T, s *engine.State, k string) view.ReviewView {
	t.Helper()
	return reviewAt(t, s, k, quickReview)
}

// reviewAt reviews key k in s with cfg, without a clock.
func reviewAt(t *testing.T, s *engine.State, k string, cfg reviewConfig) view.ReviewView {
	t.Helper()
	rv, ok := buildReview(context.Background(), s, mustKey(t, s, k), reviewSeed(9, s.Ply), cfg)
	if !ok {
		t.Fatalf("no review of %s", k)
	}
	return rv
}

// TestReviewReasons checks the review in positions built for one fact: a
// missed win, the move that wins (nothing to say), a missed looting.
func TestReviewReasons(t *testing.T) {
	// The pawn is two spaces from seat 1's capital and Walls has two
	// shields: discarding it misses a military win.
	s := namedBefore(t, "military-supremacy")
	rv := reviewOf(t, &s, "discard:walls")
	if rv.Reason != view.ReviewReasonWins || rv.Better == nil || *rv.Better != "construct:walls" || rv.Gap < 3 {
		t.Fatalf("a missed win: %+v", rv)
	}
	if rv.Played != "discard:walls" || rv.Ply != int(s.Ply) {
		t.Fatalf("the reviewed move: %+v", rv)
	}
	rv = reviewOf(t, &s, "construct:walls")
	if rv.Reason != view.ReviewReasonNone || rv.Better != nil || rv.Gap != 0 {
		t.Fatalf("the winning move: %+v", rv)
	}

	// Arsenal's four shields (with Strategy) would take both looting
	// tokens, 7 coins: discarding it misses them.
	s = namedBefore(t, "looting-double")
	rv = reviewOf(t, &s, "discard:arsenal")
	if rv.Better == nil || *rv.Better != "construct:arsenal" {
		t.Fatalf("a missed looting: %+v", rv)
	}
	if rv.Reason == view.ReviewReasonLoot && (rv.Coins == nil || *rv.Coins != 7) {
		t.Fatalf("a missed looting: coins %v", rv.Coins)
	}
}

// TestReviewPlayAgainTurn checks that a play-again wonder turn keeps every
// step of the turn rather than letting the extra action overwrite the wonder,
// and identifies which step of the turn had a clearly better move.
func TestReviewPlayAgainTurn(t *testing.T) {
	wonders := [2][engine.WondersPerPlayer]string{
		{"appian-way", "pyramids", "great-library", "mausoleum"},
		{"hanging-gardens", "sphinx", "great-lighthouse", "piraeus"},
	}
	coins := 40
	pawn := 7
	toMove := engine.Seat(0)
	sc := &engine.Scenario{
		Name:     "review-play-again",
		Age:      2,
		Wonders:  &wonders,
		Players:  [2]engine.ScenarioPlayer{{Coins: &coins}},
		OnlyLeft: []string{"walls", "rostrum", "statue"},
		Pawn:     &pawn,
		ToMove:   &toMove,
	}

	playTurn := func(keys ...string) view.ReviewView {
		t.Helper()
		s, err := engine.NewGameFromScenario(3, engine.Options{}, sc)
		if err != nil {
			t.Fatal(err)
		}
		rs := reviewState{loaded: true}
		for _, k := range keys {
			a := mustKey(t, &s, k)
			rs.recordMove(&s, a)
			if err := engine.Apply(&s, a, nil); err != nil {
				t.Fatalf("apply %s: %v", k, err)
			}
		}
		steps := rs.seats[0].steps
		if len(steps) != len(keys) {
			t.Fatalf("recorded %d steps, want %d", len(steps), len(keys))
		}
		rv, ok := buildTurnReview(context.Background(), steps, 9, quickReview)
		if !ok {
			t.Fatalf("no review of %v", keys)
		}
		return rv
	}

	// First move tucks Walls (which would have won at once) under Appian
	// Way, then the extra turn builds Rostrum: the review keeps the wonder
	// in Played and still flags the missed win on the first move.
	rv := playTurn("wonder:appian-way:walls", "construct:rostrum")
	if rv.Played != "wonder:appian-way:walls construct:rostrum" {
		t.Fatalf("played = %q, want both moves", rv.Played)
	}
	if rv.Reason != view.ReviewReasonWins || rv.Better == nil || *rv.Better != "construct:walls" {
		t.Fatalf("missed win on first move of play-again turn: %+v", rv)
	}

	// First move builds Appian Way with Rostrum, keeping Walls available,
	// and the extra turn discards Walls instead of winning with it: Better
	// prefixes the first move so the advice points to the extra turn.
	rv = playTurn("wonder:appian-way:rostrum", "discard:walls")
	if rv.Played != "wonder:appian-way:rostrum discard:walls" {
		t.Fatalf("played = %q, want both moves", rv.Played)
	}
	if rv.Reason != view.ReviewReasonWins || rv.Better == nil || *rv.Better != "wonder:appian-way:rostrum construct:walls" {
		t.Fatalf("missed win on extra turn: %+v", rv)
	}

	// First move builds Appian Way with Rostrum and the extra turn builds
	// Walls to win the game: no better move.
	rv = playTurn("wonder:appian-way:rostrum", "construct:walls")
	if rv.Played != "wonder:appian-way:rostrum construct:walls" || rv.Better != nil || rv.Reason != view.ReviewReasonNone {
		t.Fatalf("winning play-again turn: %+v", rv)
	}
}

// TestReviewStrengthsOnAMissedWin checks the two search strengths on a
// missed win: discarding Walls when building it takes the capital. The
// two-stage review keeps Greedy's verdict and points once the search
// agrees; the deep review finds the same move by itself and gives no
// points. The winning move itself gets no advice at any strength.
func TestReviewStrengthsOnAMissedWin(t *testing.T) {
	s := namedBefore(t, "military-supremacy")
	for _, st := range searchStrengths {
		rv := reviewAt(t, &s, "discard:walls", strengthReview(st))
		if rv.Reason != view.ReviewReasonWins || rv.Better == nil || *rv.Better != "construct:walls" || rv.Strength != st {
			t.Fatalf("%s: a missed win: %+v", st, rv)
		}
		if (st == view.AdvisorStrengthDeep) != (rv.Gap == 0) {
			t.Fatalf("%s: gap %d", st, rv.Gap)
		}
		rv = reviewAt(t, &s, "construct:walls", strengthReview(st))
		if rv.Better != nil || rv.Reason != view.ReviewReasonNone || rv.Strength != st {
			t.Fatalf("%s: the winning move: %+v", st, rv)
		}
	}
}

// TestReviewConfirmation checks the two-stage rule on random moves, at the
// room's budget: a two-stage review names a better move only when the
// quick review names the same one, with the same reason and points
// (Greedy proposes, the search only confirms), and on these positions the
// search agrees with some of Greedy's proposals and turns others down.
func TestReviewConfirmation(t *testing.T) {
	states, acts := reviewPositions(t, 2)
	cfg := reviewConfig{strength: view.AdvisorStrengthTwoStage, rank: reviewRank, budget: DefaultReviewBudget}
	kept, dropped := 0, 0
	for i := range states {
		s := &states[i]
		seed := reviewSeed(29, s.Ply)
		q, ok1 := buildReview(context.Background(), s, acts[i], seed, quickReview)
		ts, ok2 := buildReview(context.Background(), s, acts[i], seed, cfg)
		if !ok1 || !ok2 {
			t.Fatalf("position %d: no review (quick %v, two-stage %v)", i, ok1, ok2)
		}
		switch {
		case ts.Better != nil:
			if q.Better == nil || *q.Better != *ts.Better || q.Reason != ts.Reason || q.Gap != ts.Gap {
				t.Fatalf("position %d (%s): quick says %+v, two-stage %+v", i, acts[i], q, ts)
			}
			kept++
		case q.Better != nil:
			if ts.Reason != view.ReviewReasonNone || ts.Gap != 0 {
				t.Fatalf("position %d: a two-stage review with no better move: %+v", i, ts)
			}
			if dropped == 0 {
				t.Logf("position %d (ply %d, %s): Greedy proposes %s (%s, %d points), the search does not agree", i, s.Ply, acts[i], *q.Better, q.Reason, q.Gap)
			}
			dropped++
		}
	}
	t.Logf("%d positions: the search confirmed %d of Greedy's proposals and turned down %d", len(states), kept, dropped)
	if kept == 0 || dropped == 0 {
		t.Fatalf("the search confirmed %d proposals and turned down %d; want some of each", kept, dropped)
	}
}

// TestReviewJobWaitsForSlot checks a review job against the Manager's
// review slots: with every slot taken it waits, and a job whose deadline
// passes first posts a review that says nothing; with a slot free it
// reviews, posts the result under its generation and gives the slot back.
func TestReviewJobWaitsForSlot(t *testing.T) {
	s := namedBefore(t, "military-supremacy")
	a := mustKey(t, &s, "discard:walls")
	slots := make(chan struct{}, 1)
	slots <- struct{}{}
	inbox := make(chan any, 1)
	job := reviewJob{
		seat:     s.ToMove,
		gen:      7,
		steps:    []reviewStep{{pre: s, action: a}},
		roomSeed: 1,
		cfg:      strengthReview(view.AdvisorStrengthTwoStage),
		slots:    slots,
		inbox:    inbox,
		done:     make(chan struct{}),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	job.run(ctx)
	if m := (<-inbox).(msgReviewResult); m.ok || m.gen != 7 || m.seat != s.ToMove {
		t.Fatalf("a job that found no slot posted %+v", m)
	}
	<-slots
	job.run(context.Background())
	m := (<-inbox).(msgReviewResult)
	if !m.ok || m.review.Better == nil || m.review.Strength != view.AdvisorStrengthTwoStage {
		t.Fatalf("a job with a free slot posted %+v", m)
	}
	if len(slots) != 0 {
		t.Fatal("the job kept its slot")
	}
}

// TestReviewDeterministic checks, at every strength, that the same
// position and move give the same review, and that a review is a pure
// function of the seed: a second room built the same way, or a replay of
// the log, says the same.
func TestReviewDeterministic(t *testing.T) {
	states, acts := reviewPositions(t, 2)
	for _, cfg := range []reviewConfig{quickReview, strengthReview(view.AdvisorStrengthTwoStage), strengthReview(view.AdvisorStrengthDeep)} {
		for i := range states {
			if cfg.strength != view.AdvisorStrengthQuick && i%4 != 0 {
				continue
			}
			s := &states[i]
			one, ok1 := buildReview(context.Background(), s, acts[i], reviewSeed(11, s.Ply), cfg)
			two, ok2 := buildReview(context.Background(), s, acts[i], reviewSeed(11, s.Ply), cfg)
			if !ok1 || ok1 != ok2 || !reflect.DeepEqual(one, two) {
				t.Fatalf("%s, position %d (%s): %+v then %+v", cfg.strength, i, acts[i], one, two)
			}
			if one.Strength != cfg.strength {
				t.Fatalf("%s, position %d: the review says strength %q", cfg.strength, i, one.Strength)
			}
		}
	}
}

// TestReviewLeakGuard is the leak guard over the review: for random
// positions and moves, the review is the same on the real state and on
// every determinization of the mover's observation, which differ in every
// card the mover has not seen.
//
// It covers every strength: the quick one on every position, the two
// search strengths on every third, with the test budget (the search sees
// only the observation whatever its budget).
func TestReviewLeakGuard(t *testing.T) {
	states, acts := reviewPositions(t, 3)
	for _, cfg := range []reviewConfig{quickReview, strengthReview(view.AdvisorStrengthTwoStage), strengthReview(view.AdvisorStrengthDeep)} {
		said, checked := 0, 0
		reasons := map[view.ReviewReason]int{}
		for i := range states {
			if cfg.strength != view.AdvisorStrengthQuick && i%3 != 0 {
				continue
			}
			checked++
			s := &states[i]
			seed := reviewSeed(13, s.Ply)
			want, ok := buildReview(context.Background(), s, acts[i], seed, cfg)
			if !ok {
				t.Fatalf("%s, position %d: no review", cfg.strength, i)
			}
			if want.Better != nil {
				said++
			}
			reasons[want.Reason]++
			obs := engine.Observe(s, s.ToMove)
			r := engine.NewRNG(uint64(i))
			for range 2 {
				d := obs.Determinize(&r)
				got, ok := buildReview(context.Background(), &d, acts[i], seed, cfg)
				if !ok || !reflect.DeepEqual(got, want) {
					t.Fatalf("%s, position %d (%s): the real state says %+v, a determinization %+v", cfg.strength, i, acts[i], want, got)
				}
			}
		}
		if said == 0 {
			t.Fatalf("%s: no review named a better move: the guard checked nothing", cfg.strength)
		}
		t.Logf("%s: %d reviews of random moves, by reason: %v", cfg.strength, checked, reasons)
	}
}

// TestReviewNamesOnlySeenCards checks that a review names only what the
// mover saw before moving: the better move's card is one of the face-up
// cards it could take, and a threat names a face-up card or a wonder,
// never a card the move turned over.
func TestReviewNamesOnlySeenCards(t *testing.T) {
	states, acts := reviewPositions(t, 4)
	for _, cfg := range []reviewConfig{quickReview, strengthReview(view.AdvisorStrengthDeep)} {
		for i := range states {
			if cfg.strength != view.AdvisorStrengthQuick && i%3 != 0 {
				continue
			}
			checkSeenOnly(t, cfg, i, &states[i], acts[i])
		}
	}
}

// checkSeenOnly checks that the review of a in s names only what the mover
// saw before moving.
func checkSeenOnly(t *testing.T, cfg reviewConfig, i int, s *engine.State, a engine.Action) {
	t.Helper()
	rv, ok := buildReview(context.Background(), s, a, reviewSeed(17, s.Ply), cfg)
	if !ok || rv.Better == nil {
		return
	}
	faceUp := map[string]bool{}
	for _, sl := range engine.ViewFor(s, engine.SeatViewer(s.ToMove)).Structure {
		if sl.Card != nil {
			faceUp[*sl.Card] = true
		}
	}
	better, err := engine.ParseKey(*rv.Better)
	if err != nil || engine.IsLegal(s, better) != nil {
		t.Fatalf("%s, position %d: better %q is not a legal move", cfg.strength, i, *rv.Better)
	}
	if rv.Threat != nil && rv.Threat.Card != nil && !faceUp[*rv.Threat.Card] {
		t.Fatalf("%s, position %d: the threat names %s, which was not face up", cfg.strength, i, *rv.Threat.Card)
	}
}

// TestReviewBudget checks that a review whose ranking runs out of budget
// or time says nothing rather than guess.
func TestReviewBudget(t *testing.T) {
	s := namedBefore(t, "military-supremacy")
	a := mustKey(t, &s, "discard:walls")
	if _, ok := buildReview(context.Background(), &s, a, 1, reviewConfig{strength: view.AdvisorStrengthQuick, rank: ai.RankConfig{K: 4, MaxEvals: 1}}); ok {
		t.Fatal("a review with no budget said something")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, cfg := range []reviewConfig{quickReview, strengthReview(view.AdvisorStrengthTwoStage), strengthReview(view.AdvisorStrengthDeep)} {
		if _, ok := buildReview(ctx, &s, a, 1, cfg); ok {
			t.Fatalf("a cancelled %s review said something", cfg.strength)
		}
	}
	// The search cut short after Greedy has spoken: the deadline comes
	// long before a billion iterations are done. A missed looting is not
	// a sure fact, so the two-stage review needs the search to agree.
	s = namedBefore(t, "looting-double")
	a = mustKey(t, &s, "discard:arsenal")
	if q := reviewOf(t, &s, "discard:arsenal"); q.Better == nil || exactReason(q.Reason) {
		t.Fatalf("the quick review of the looting position: %+v", q)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	huge := ReviewBudget{Compare: 1 << 30, Search: 1 << 30}
	for _, st := range searchStrengths {
		if rv, ok := buildReview(ctx, &s, a, 1, reviewConfig{strength: st, rank: reviewRank, budget: huge}); ok {
			t.Fatalf("a %s review whose search was cut short said %+v", st, rv)
		}
	}
}

// TestReviewCost checks that the largest reviews stay well within the
// room's budget: their cost is reported (go test -v -run TestReviewCost),
// and only a wildly slow one fails, since the machine may be loaded.
func TestReviewCost(t *testing.T) {
	if testing.Short() {
		t.Skip("timing")
	}
	states, acts := reviewPositions(t, 3)
	var total, worst time.Duration
	evals := 0
	for i := range states {
		s := &states[i]
		start := time.Now()
		if _, ok := buildReview(context.Background(), s, acts[i], reviewSeed(19, s.Ply), quickReview); !ok {
			t.Fatalf("position %d: no review", i)
		}
		d := time.Since(start)
		total += d
		worst = max(worst, d)
		obs := engine.Observe(s, s.ToMove)
		rk, _ := ai.RankActions(context.Background(), obs, reviewSeed(19, s.Ply), reviewRank)
		evals = max(evals, rk.Evals)
	}
	t.Logf("%d reviews: mean %v, worst %v, at most %d evaluations", len(states), total/time.Duration(len(states)), worst, evals)
	if worst > time.Second {
		t.Errorf("the worst review took %v", worst)
	}
}

// TestReviewSearchCost reports what the two-stage and deep reviews cost
// at the room's budget (go test -v -run TestReviewSearchCost): the mean
// and worst wall clock of a review, over random moves. docs/ai.md records
// the numbers. Only a wildly slow review fails, since the machine may be
// loaded.
func TestReviewSearchCost(t *testing.T) {
	if testing.Short() {
		t.Skip("timing")
	}
	states, acts := reviewPositions(t, 2)
	for _, st := range searchStrengths {
		cfg := reviewConfig{strength: st, rank: reviewRank, budget: DefaultReviewBudget}
		var total, worst time.Duration
		n := 0
		for i := range states {
			if i%2 != 0 {
				continue
			}
			s := &states[i]
			start := time.Now()
			if _, ok := buildReview(context.Background(), s, acts[i], reviewSeed(31, s.Ply), cfg); !ok {
				t.Fatalf("%s, position %d: no review", st, i)
			}
			d := time.Since(start)
			total += d
			worst = max(worst, d)
			n++
		}
		t.Logf("%s: %d reviews, mean %v, worst %v", st, n, total/time.Duration(n), worst)
		if worst > 10*time.Second {
			t.Errorf("%s: the worst review took %v", st, worst)
		}
	}
}

// BenchmarkReview measures one review of a mid-game position.
func BenchmarkReview(b *testing.B) {
	states, acts := reviewPositions(b, 2)
	for i := 0; b.Loop(); i++ {
		k := i % len(states)
		s := &states[k]
		buildReview(context.Background(), s, acts[k], reviewSeed(23, s.Ply), quickReview)
	}
}

// BenchmarkReviewTwoStage measures one two-stage review at the room's
// budget, of positions where Greedy proposes a better move (so the search
// runs).
func BenchmarkReviewTwoStage(b *testing.B) { benchSearchReview(b, view.AdvisorStrengthTwoStage) }

// BenchmarkReviewDeep measures one deep review at the room's budget.
func BenchmarkReviewDeep(b *testing.B) { benchSearchReview(b, view.AdvisorStrengthDeep) }

func benchSearchReview(b *testing.B, st view.AdvisorStrength) {
	states, acts := reviewPositions(b, 2)
	cfg := reviewConfig{strength: st, rank: reviewRank, budget: DefaultReviewBudget}
	var idx []int
	for i := range states {
		s := &states[i]
		if q, ok := buildReview(context.Background(), s, acts[i], reviewSeed(23, s.Ply), quickReview); ok && (q.Better != nil || st == view.AdvisorStrengthDeep) {
			idx = append(idx, i)
		}
	}
	if len(idx) == 0 {
		b.Fatal("no position to review")
	}
	for i := 0; b.Loop(); i++ {
		k := idx[i%len(idx)]
		s := &states[k]
		buildReview(context.Background(), s, acts[k], reviewSeed(23, s.Ply), cfg)
	}
}
