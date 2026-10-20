package room

import (
	"context"
	"reflect"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

func recordReviewChoice(t *testing.T, rs *reviewState, s *engine.State, key string) {
	t.Helper()
	a := mustKey(t, s, key)
	rs.recordMove(s, a)
	if err := engine.Apply(s, a, nil); err != nil {
		t.Fatal(err)
	}
}

func TestReviewTokenChoice(t *testing.T) {
	s := namedBefore(t, "law-supremacy")
	engine.ApplyTrusted(&s, mustKey(t, &s, "construct:academy"))
	for _, strength := range []view.AdvisorStrength{view.AdvisorStrengthQuick, view.AdvisorStrengthTwoStage, view.AdvisorStrengthDeep} {
		t.Run(string(strength), func(t *testing.T) {
			cfg := strengthReview(strength)
			rv, ok := buildTurnReview(context.Background(), []reviewStep{{pre: s, action: mustKey(t, &s, "token:agriculture")}}, 9, cfg)
			if !ok || len(rv.Entries) != 1 {
				t.Fatalf("missing token decision: %+v, ok %v", rv, ok)
			}
			e := rv.Entries[0]
			if e.Played != "token:agriculture" || e.Better == nil || *e.Better != "token:law" || e.Reason != view.ReviewReasonWins {
				t.Fatalf("missed winning token: %+v", e)
			}
			best := reviewAt(t, &s, "token:law", cfg)
			if best.Better != nil || best.Reason != view.ReviewReasonNone {
				t.Fatalf("no better alternative to winning token: %+v", best)
			}
		})
	}
}

func TestReviewAllDecisionKinds(t *testing.T) {
	for _, tc := range []struct{ scenario, first, choice string }{
		{"science-pair", "construct:library", ""},
		{"great-library", "wonder:great-library:rostrum", "token:philosophy"},
		{"mausoleum", "wonder:mausoleum:rostrum", "revive:baths"},
		{"zeus-destroy", "wonder:statue-of-zeus:rostrum", ""},
	} {
		t.Run(tc.scenario, func(t *testing.T) {
			s := namedBefore(t, tc.scenario)
			rs := reviewState{loaded: true}
			seat := s.ToMove
			recordReviewChoice(t, &rs, &s, tc.first)
			key := tc.choice
			if key == "" {
				key = engine.LegalActions(&s, nil)[0].Key()
			}
			recordReviewChoice(t, &rs, &s, key)
			rv, ok := buildTurnReview(context.Background(), rs.seats[seat].steps, 9, quickReview)
			if !ok || len(rv.Entries) != 2 || rv.Entries[0].Played != tc.first || rv.Entries[1].Played != key {
				t.Fatalf("decisions lost or reordered: %+v", rv)
			}
		})
	}
}

func TestReviewWonderTokenExtraTurn(t *testing.T) {
	sc := engine.NamedScenario("great-library")
	sc.Players[0].Tokens = []string{"theology"}
	sc.OnlyLeft = []string{"rostrum", "statue", "walls"}
	s, err := engine.NewGameFromScenario(3, engine.Options{}, sc)
	if err != nil {
		t.Fatal(err)
	}
	rs := reviewState{loaded: true}
	keys := []string{"wonder:great-library:rostrum", "token:philosophy", "construct:statue"}
	for _, key := range keys {
		recordReviewChoice(t, &rs, &s, key)
	}
	rv, ok := buildTurnReview(context.Background(), rs.seats[0].steps, 9, quickReview)
	got := []string{}
	for _, e := range rv.Entries {
		got = append(got, e.Played)
	}
	if !ok || !reflect.DeepEqual(got, keys) {
		t.Fatalf("got %v, want %v (ok %v)", got, keys, ok)
	}
}

func TestReviewDraftAndStarterBoundaries(t *testing.T) {
	s := engine.NewGame(3, engine.Options{})
	rs := reviewState{loaded: true}
	for s.Phase == engine.PhaseWonderDraft {
		a := engine.LegalActions(&s, nil)[0]
		recordReviewChoice(t, &rs, &s, a.Key())
	}
	for seat := range engine.Seat(2) {
		if !rs.seats[seat].set || rs.seats[seat].action.Type != engine.ActDraftWonder {
			t.Fatal("draft decision missing")
		}
	}
	seat := s.ToMove
	recordReviewChoice(t, &rs, &s, engine.LegalActions(&s, nil)[0].Key())
	if len(rs.seats[seat].steps) != 4 {
		t.Fatal("opening card action lost a draft decision")
	}

	sc := engine.NamedScenario("science-pair")
	sc.OnlyLeft = []string{"library"}
	s, err := engine.NewGameFromScenario(3, engine.Options{}, sc)
	if err != nil {
		t.Fatal(err)
	}
	rs = reviewState{loaded: true}
	recordReviewChoice(t, &rs, &s, "discard:library")
	if s.Phase != engine.PhaseChooseStarter {
		t.Fatalf("want starter choice, got %v", s.Phase)
	}
	seat = s.ToMove
	recordReviewChoice(t, &rs, &s, engine.StarterAction(seat).Key())
	if rs.seats[seat].action.Type != engine.ActChooseStarter {
		t.Fatal("starter decision missing")
	}
	if r := (&Room{state: s, reviews: rs}); r.turnOver(seat) {
		t.Fatal("review started before the chosen starter's own card action")
	}
	recordReviewChoice(t, &rs, &s, engine.LegalActions(&s, nil)[0].Key())
	steps := rs.seats[seat].steps
	if len(steps) < 2 || steps[len(steps)-2].action.Type != engine.ActChooseStarter {
		t.Fatal("choosing yourself to start lost the starter decision")
	}
}

func TestReviewDraftWaitsForOpeningCard(t *testing.T) {
	s := engine.NewGame(3, engine.Options{})
	r := &Room{state: s, reviews: reviewState{loaded: true}}
	for s.Phase == engine.PhaseWonderDraft {
		recordReviewChoice(t, &r.reviews, &s, engine.LegalActions(&s, nil)[0].Key())
		r.state = s
		if s.Phase == engine.PhaseWonderDraft {
			for seat := range engine.Seat(2) {
				if r.turnOver(seat) {
					t.Fatalf("seat %d's draft review was ready before the draft ended", seat)
				}
			}
		}
	}
	if r.turnOver(s.ToMove) {
		t.Fatal("opening player's draft review started before their first card")
	}
	if !r.turnOver(s.ToMove.Opp()) {
		t.Fatal("other player's completed draft review is not ready")
	}
}

func TestReviewSingleOptionSkipsSearch(t *testing.T) {
	sc := engine.NamedScenario("zeus-destroy")
	sc.Players[1].City = []string{"quarry"}
	s, err := engine.NewGameFromScenario(3, engine.Options{}, sc)
	if err != nil {
		t.Fatal(err)
	}
	engine.ApplyTrusted(&s, mustKey(t, &s, "wonder:statue-of-zeus:rostrum"))
	legal := engine.LegalActions(&s, nil)
	if len(legal) != 1 {
		t.Fatalf("want one destroy option, got %v", legal)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // any ranking or search would fail, proving this path skips it
	for _, strength := range []view.AdvisorStrength{view.AdvisorStrengthQuick, view.AdvisorStrengthTwoStage, view.AdvisorStrengthDeep} {
		rv, ok := buildReview(ctx, &s, legal[0], 9, strengthReview(strength))
		if !ok || rv.Played != legal[0].Key() || rv.Strength != strength || rv.Reason != view.ReviewReasonNone || rv.Better != nil || rv.Gap != 0 || rv.Entries == nil {
			t.Fatalf("%s: forced decision should need no search: %+v, ok %v", strength, rv, ok)
		}
	}
}

func TestReviewWaitsForPendingChoice(t *testing.T) {
	s := namedBefore(t, "science-pair")
	engine.ApplyTrusted(&s, mustKey(t, &s, "construct:library"))
	r := &Room{state: s}
	if r.turnOver(s.ToMove) {
		t.Fatal("turn considered over while a progress token is pending")
	}
}

// Every token leads to the same civilian win on this last-card choice.
// A different kind of win (Law's science win) is no lesson in a decided game.
func TestReviewTokenDecided(t *testing.T) {
	sc := engine.NamedScenario("law-supremacy")
	sc.OnlyLeft = []string{"academy"}
	coins := 200
	sc.Players[0].Coins = &coins
	s, err := engine.NewGameFromScenario(3, engine.Options{}, sc)
	if err != nil {
		t.Fatal(err)
	}
	engine.ApplyTrusted(&s, mustKey(t, &s, "construct:academy"))
	for _, strength := range []view.AdvisorStrength{view.AdvisorStrengthQuick, view.AdvisorStrengthTwoStage, view.AdvisorStrengthDeep} {
		rv := reviewAt(t, &s, "token:agriculture", strengthReview(strength))
		if rv.Better != nil || rv.Reason != view.ReviewReasonNone {
			t.Fatalf("%s: advised in a decided position: %+v", strength, rv)
		}
	}
}

func TestReviewTokenSkipsBot(t *testing.T) {
	s := namedBefore(t, "law-supremacy")
	engine.ApplyTrusted(&s, mustKey(t, &s, "construct:academy"))
	for _, strength := range []view.AdvisorStrength{view.AdvisorStrengthQuick, view.AdvisorStrengthTwoStage, view.AdvisorStrengthDeep} {
		hints := view.HintSettings{Advisor: true, AdvisorStrength: strength}
		f := &store.File{Hints: &hints}
		f.Seats[s.ToMove].Kind = store.SeatKindBot
		r := &Room{file: f, state: s, reviews: reviewState{loaded: true}}
		a := mustKey(t, &s, "token:agriculture")
		r.reviews.recordMove(&s, a)
		engine.ApplyTrusted(&r.state, a)
		if rv := r.review(s.ToMove); rv != nil || r.reviews.seats[s.ToMove].pending {
			t.Fatalf("%s: bot received or started a token review", strength)
		}
	}
}

// A summary can be opened after the next age starts; carry the age from
// the starter decision's own position, rather than the current snapshot.
func TestReviewStarterCarriesAge(t *testing.T) {
	for _, oldAge := range []int{1, 2} {
		sc := &engine.Scenario{Age: oldAge}
		// Use known age-appropriate final cards instead of depending on a
		// random legal-action order in the fixture.
		sc.OnlyLeft = []string{"library"}
		if oldAge == 1 {
			sc.OnlyLeft = []string{"palisade"}
		}
		s, err := engine.NewGameFromScenario(7, engine.Options{}, sc)
		if err != nil {
			t.Fatal(err)
		}
		engine.ApplyTrusted(&s, mustKey(t, &s, "discard:"+sc.OnlyLeft[0]))
		if s.Phase != engine.PhaseChooseStarter {
			t.Fatal("missing starter decision")
		}
		age := int(s.Age)
		for seat := range engine.Seat(2) {
			rv, ok := buildTurnReview(context.Background(), []reviewStep{{pre: s, action: engine.StarterAction(seat)}}, 7, quickReview)
			if !ok || len(rv.Entries) != 1 {
				t.Fatalf("missing review: %+v", rv)
			}
			if got := rv.Entries[0].Age; got == nil || *got != age {
				t.Fatalf("starter age %v, want %d", got, age)
			}
		}
	}
}
