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

// reviewOf reviews key k in s with the room's budget, without a clock.
func reviewOf(t *testing.T, s *engine.State, k string) view.ReviewView {
	t.Helper()
	rv, ok := buildReview(context.Background(), s, mustKey(t, s, k), reviewSeed(9, s.Ply), reviewRank)
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
		rv, ok := buildTurnReview(context.Background(), steps, 9, reviewRank)
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

// TestReviewDeterministic checks that the same position and move give the
// same review, and that a review is a pure function of the seed: a second
// room built the same way says the same.
func TestReviewDeterministic(t *testing.T) {
	states, acts := reviewPositions(t, 2)
	for i := range states {
		s := &states[i]
		one, ok1 := buildReview(context.Background(), s, acts[i], reviewSeed(11, s.Ply), reviewRank)
		two, ok2 := buildReview(context.Background(), s, acts[i], reviewSeed(11, s.Ply), reviewRank)
		if ok1 != ok2 || !reflect.DeepEqual(one, two) {
			t.Fatalf("position %d (%s): %+v then %+v", i, acts[i], one, two)
		}
	}
}

// TestReviewLeakGuard is the leak guard over the review: for random
// positions and moves, the review is the same on the real state and on
// every determinization of the mover's observation, which differ in every
// card the mover has not seen.
func TestReviewLeakGuard(t *testing.T) {
	states, acts := reviewPositions(t, 3)
	said := 0
	reasons := map[view.ReviewReason]int{}
	for i := range states {
		s := &states[i]
		seed := reviewSeed(13, s.Ply)
		want, ok := buildReview(context.Background(), s, acts[i], seed, reviewRank)
		if !ok {
			t.Fatalf("position %d: no review", i)
		}
		if want.Better != nil {
			said++
		}
		reasons[want.Reason]++
		obs := engine.Observe(s, s.ToMove)
		r := engine.NewRNG(uint64(i))
		for range 2 {
			d := obs.Determinize(&r)
			got, ok := buildReview(context.Background(), &d, acts[i], seed, reviewRank)
			if !ok || !reflect.DeepEqual(got, want) {
				t.Fatalf("position %d (%s): the real state says %+v, a determinization %+v", i, acts[i], want, got)
			}
		}
	}
	if said == 0 {
		t.Fatal("no review named a better move: the guard checked nothing")
	}
	t.Logf("%d reviews of random moves, by reason: %v", len(states), reasons)
}

// TestReviewNamesOnlySeenCards checks that a review names only what the
// mover saw before moving: the better move's card is one of the face-up
// cards it could take, and a threat names a face-up card or a wonder,
// never a card the move turned over.
func TestReviewNamesOnlySeenCards(t *testing.T) {
	states, acts := reviewPositions(t, 4)
	for i := range states {
		s := &states[i]
		rv, ok := buildReview(context.Background(), s, acts[i], reviewSeed(17, s.Ply), reviewRank)
		if !ok || rv.Better == nil {
			continue
		}
		faceUp := map[string]bool{}
		for _, sl := range engine.ViewFor(s, engine.SeatViewer(s.ToMove)).Structure {
			if sl.Card != nil {
				faceUp[*sl.Card] = true
			}
		}
		better, err := engine.ParseKey(*rv.Better)
		if err != nil || engine.IsLegal(s, better) != nil {
			t.Fatalf("position %d: better %q is not a legal move", i, *rv.Better)
		}
		if rv.Threat != nil && rv.Threat.Card != nil && !faceUp[*rv.Threat.Card] {
			t.Fatalf("position %d: the threat names %s, which was not face up", i, *rv.Threat.Card)
		}
	}
}

// TestReviewBudget checks that a review whose ranking runs out of budget
// or time says nothing rather than guess.
func TestReviewBudget(t *testing.T) {
	s := namedBefore(t, "military-supremacy")
	a := mustKey(t, &s, "discard:walls")
	if _, ok := buildReview(context.Background(), &s, a, 1, ai.RankConfig{K: 4, MaxEvals: 1}); ok {
		t.Fatal("a review with no budget said something")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, ok := buildReview(ctx, &s, a, 1, reviewRank); ok {
		t.Fatal("a cancelled review said something")
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
		if _, ok := buildReview(context.Background(), s, acts[i], reviewSeed(19, s.Ply), reviewRank); !ok {
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

// BenchmarkReview measures one review of a mid-game position.
func BenchmarkReview(b *testing.B) {
	states, acts := reviewPositions(b, 2)
	for i := 0; b.Loop(); i++ {
		k := i % len(states)
		s := &states[k]
		buildReview(context.Background(), s, acts[k], reviewSeed(23, s.Ply), reviewRank)
	}
}
