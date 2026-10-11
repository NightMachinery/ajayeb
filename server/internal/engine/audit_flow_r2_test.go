package engine_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

// This file is round 1 of the M10.R1 rules audit, lens: turn flow and ages.
// It is black-box: every expected value is derived from docs/rules/ and the
// design reference (rules-reference.md), not from the engine's code. It
// avoids the cases of edgecases_flow_r1_test.go and aims at interactions of
// two or three effects, boundary values and the order of resolution inside
// one action. Helpers are prefixed auditR2Flow.

func auditR2FlowCard(t *testing.T, slug string) catalog.CardID {
	t.Helper()
	c, ok := catalog.CardBySlug(slug)
	if !ok {
		t.Fatalf("test setup: no card %q", slug)
	}
	return c
}

func auditR2FlowToken(t *testing.T, slug string) catalog.TokenID {
	t.Helper()
	tk, ok := catalog.TokenBySlug(slug)
	if !ok {
		t.Fatalf("test setup: no token %q", slug)
	}
	return tk
}

func auditR2FlowCount(ev []engine.Event, kind engine.EventKind) int {
	n := 0
	for _, e := range ev {
		if e.Kind == kind {
			n++
		}
	}
	return n
}

func auditR2FlowHasKey(keys []string, want string) bool {
	return slices.Contains(keys, want)
}

// auditR2FlowSorted returns a sorted copy, so legal sets compare as sets.
func auditR2FlowSorted(keys []string) []string {
	out := slices.Clone(keys)
	slices.Sort(out)
	return out
}

func auditR2FlowKinds(ev []engine.Event) []string {
	out := make([]string, len(ev))
	for i, e := range ev {
		out[i] = e.Kind.String()
	}
	return out
}

// auditR2FlowSixCards are the first six cards of the Age I layout order
// when given to OnlyLeft. The layout (setup.md "Age I") puts them at rows
// 1 (slots 0, 1), 2 (slots 2, 3, 4) and 3 (slot 5, column 3). So workshop
// (slot 2, row 2, column 4, a face-down row) is overlapped only by slot 5
// (row 3, column 3), since row 3 column 5 is empty; the row 2 cards in
// slots 3 and 4 are accessible.
var auditR2FlowSixCards = []string{"altar", "baths", "workshop", "quarry", "clay-pool"}

// auditR2FlowSixWith returns the six OnlyLeft cards with last as the card
// in slot 5 (the one that covers workshop).
func auditR2FlowSixWith(last string) []string {
	return append(slices.Clone(auditR2FlowSixCards), last)
}

// auditR2FlowCheckHiddenWorkshop checks the test position: workshop is in
// the structure, face down and not accessible.
func auditR2FlowCheckHiddenWorkshop(t *testing.T, s *engine.State) int {
	t.Helper()
	i := engine.SlotOf(s, auditR2FlowCard(t, "workshop"))
	if i < 0 || s.Slots[i].FaceUp || engine.Accessible(s)&(1<<i) != 0 {
		t.Fatalf("test setup: workshop should be face down and covered, slot %d, accessible %b", i, engine.Accessible(s))
	}
	return i
}

// TestAuditR2FlowLegalSetAtZeroOneAndTwoCoins pins turn.md "For developers"
// (the legal set in the play phase: a discard for every accessible card, a
// construct key only when the mover can pay, a wonder key only for an
// affordable wonder) and turn.md "2. Discard it for coins" (always
// allowed), at the boundary of exactly 0, 1 and 2 coins. Costs come from the
// catalog: Lumber Yard, Theater and Tavern are free; Stone Pit costs 1 coin;
// Palisade 2 coins; Stable 1 wood, which costs 2 coins to buy when the
// opponent produces no wood (economy.md trade price 2 + opponent's
// production). Every wonder needs at least 4 resource units, so at least 8
// coins with no production.
func TestAuditR2FlowLegalSetAtZeroOneAndTwoCoins(t *testing.T) {
	acc := []string{"lumber-yard", "stone-pit", "stable", "theater", "palisade", "tavern"}
	cases := []struct {
		coins     int
		construct []string
	}{
		{0, []string{"lumber-yard", "theater", "tavern"}},
		{1, []string{"lumber-yard", "stone-pit", "theater", "tavern"}},
		{2, acc},
	}
	for _, c := range cases {
		s := enginetest.New(t, 101).Age(1).Accessible(acc...).Coins(0, c.coins).ToMove(0).State()
		if got := engine.Accessible(&s); auditR2FlowPopcount(got) != 6 {
			t.Fatalf("test setup: %d accessible cards, want 6", auditR2FlowPopcount(got))
		}
		var want []string
		for _, card := range acc {
			want = append(want, "discard:"+card)
		}
		for _, card := range c.construct {
			want = append(want, "construct:"+card)
		}
		got := auditR2FlowSorted(enginetest.Keys(&s))
		if !slices.Equal(got, auditR2FlowSorted(want)) {
			t.Errorf("%d coins: legal %v, want %v", c.coins, got, auditR2FlowSorted(want))
		}
	}
}

// auditR2FlowPopcount counts set bits in a slot mask.
func auditR2FlowPopcount(m uint32) int {
	n := 0
	for ; m != 0; m &= m - 1 {
		n++
	}
	return n
}

// TestAuditR2FlowWonderKeysAppearAtExactlyTheirCost pins turn.md "3. Build a
// wonder" and "For developers" (a wonder key for each accessible card when
// the wonder is affordable) at the coin boundary. The Pyramids cost 3 stone
// and 1 papyrus; with no production and an opponent who produces nothing,
// each unit costs 2 coins (economy.md), so 8 coins. With exactly 8 coins
// every accessible card can be tucked; with 7 no wonder key is legal at all
// (no wonder costs fewer than 4 resource units).
func TestAuditR2FlowWonderKeysAppearAtExactlyTheirCost(t *testing.T) {
	acc := []string{"lumber-yard", "stone-pit", "stable", "theater", "palisade", "tavern"}
	s := enginetest.New(t, 102).Age(1).Wonders(0, "pyramids").Accessible(acc...).Coins(0, 8).ToMove(0).State()
	keys := enginetest.Keys(&s)
	for _, card := range acc {
		if !auditR2FlowHasKey(keys, "wonder:pyramids:"+card) {
			t.Errorf("8 coins: wonder:pyramids:%s missing from %v", card, keys)
		}
	}
	s = enginetest.New(t, 102).Age(1).Wonders(0, "pyramids").Accessible(acc...).Coins(0, 7).ToMove(0).State()
	for _, k := range enginetest.Keys(&s) {
		if strings.HasPrefix(k, "wonder:") {
			t.Errorf("7 coins: %s is legal, but no wonder costs 7 coins or less here", k)
		}
	}
}

// TestAuditR2FlowFaceDownCardIsNotPlayableThenRevealedForTheExtraTurn pins
// turn.md "Taking a card" (only accessible cards; a face-down card is not
// taken) and "Extra turns", Timing: "the extra turn begins after the newly
// accessible cards have been revealed, so you can use what was just turned
// over" (also rules-reference.md section 2 and section 8 item 2). After the
// extra turn the alternation resumes (turn.md "Whose turn it is").
func TestAuditR2FlowFaceDownCardIsNotPlayableThenRevealedForTheExtraTurn(t *testing.T) {
	s := enginetest.New(t, 103).Age(1).OnlyLeft(auditR2FlowSixWith("lumber-yard")...).
		Wonders(0, "sphinx").Coins(0, 30).ToMove(0).State()
	slot := auditR2FlowCheckHiddenWorkshop(t, &s)
	for _, step := range []string{"construct:workshop", "discard:workshop", "wonder:sphinx:workshop"} {
		if r := enginetest.MustFail(t, &s, step); r != engine.ReasonNotAccessible {
			t.Errorf("%s on a covered face-down card: reason %s, want notAccessible", step, r)
		}
	}
	ev := enginetest.Do(t, &s, "wonder:sphinx:lumber-yard")
	revealed := false
	for _, e := range ev {
		if e.Kind == engine.EvReveal && int(e.A) == slot && e.Card == auditR2FlowCard(t, "workshop") {
			revealed = true
		}
	}
	if !revealed || !s.Slots[slot].FaceUp {
		t.Fatalf("workshop was not revealed before the extra turn: events %v, face up %v", auditR2FlowKinds(ev), s.Slots[slot].FaceUp)
	}
	if s.Phase != engine.PhasePlay || s.ToMove != 0 || s.Active != 0 {
		t.Fatalf("extra turn: phase %s, to move %d, active %d; want seat 0 to play again", s.Phase, s.ToMove, s.Active)
	}
	if keys := enginetest.Keys(&s); !auditR2FlowHasKey(keys, "discard:workshop") {
		t.Fatalf("the revealed card is not usable in the extra turn: legal %v", keys)
	}
	enginetest.Do(t, &s, "discard:workshop")
	if s.ToMove != 1 || s.Active != 1 {
		t.Fatalf("after the extra turn: to move %d, active %d; want seat 1", s.ToMove, s.Active)
	}
}

// TestAuditR2FlowTheologyZeusRevealsAfterTheDestroyThenPlaysAgain pins the
// end-of-turn order of turn.md "The end of a turn" with three effects in
// one turn: Theology gives the Statue of Zeus an extra turn
// (wonders-tokens.md "Theology"), the destroy choice is settled first, the
// freed face-down card is revealed only after it (decisions.md "Reveal
// timing"), and the extra turn then starts with the revealed card
// available.
func TestAuditR2FlowTheologyZeusRevealsAfterTheDestroyThenPlaysAgain(t *testing.T) {
	s := enginetest.New(t, 104).Age(1).OnlyLeft(auditR2FlowSixWith("lumber-yard")...).
		Wonders(0, "statue-of-zeus").Tokens(0, "theology").Coins(0, 30).
		City(1, "stone-pit").ToMove(0).State()
	slot := auditR2FlowCheckHiddenWorkshop(t, &s)
	ev := enginetest.Do(t, &s, "wonder:statue-of-zeus:lumber-yard")
	if s.Phase != engine.PhaseDestroyCard || s.ToMove != 0 {
		t.Fatalf("after Zeus: phase %s, to move %d; want the destroy choice for seat 0", s.Phase, s.ToMove)
	}
	if auditR2FlowCount(ev, engine.EvReveal) != 0 || s.Slots[slot].FaceUp {
		t.Fatalf("workshop was revealed before the destroy choice: events %v", auditR2FlowKinds(ev))
	}
	if auditR2FlowCount(ev, engine.EvExtraTurn) != 1 {
		t.Fatalf("Theology should give Zeus one extra turn: events %v", auditR2FlowKinds(ev))
	}
	ev = enginetest.Do(t, &s, "destroy:stone-pit")
	if auditR2FlowCount(ev, engine.EvReveal) != 1 || !s.Slots[slot].FaceUp {
		t.Fatalf("workshop not revealed after the choice: events %v", auditR2FlowKinds(ev))
	}
	if s.Phase != engine.PhasePlay || s.ToMove != 0 || s.Active != 0 {
		t.Fatalf("after the choice: phase %s, to move %d, active %d; want seat 0's extra turn", s.Phase, s.ToMove, s.Active)
	}
	if keys := enginetest.Keys(&s); !auditR2FlowHasKey(keys, "discard:workshop") {
		t.Fatalf("the revealed card is not offered in the extra turn: %v", keys)
	}
}

// TestAuditR2FlowStarterChooserReadsThePawnAfterTheLastCardsShields pins
// ages-scoring.md "The end of an age" (the age ends when the last card has
// been taken and that turn has finished) and "Who starts the next age" (the
// weaker player, on whose side the pawn stands, chooses). The last card of
// Age II is Walls (2 shields): it carries the pawn from seat 0's side (-1)
// to seat 1's side (+1), so seat 1 is now weaker and chooses, although seat
// 0 was weaker when the turn began. Then the chosen starter plays first and
// the alternation resumes.
func TestAuditR2FlowStarterChooserReadsThePawnAfterTheLastCardsShields(t *testing.T) {
	s := enginetest.New(t, 105).Age(2).OnlyLeft("walls").Pawn(-1).Coins(0, 20).ToMove(0).State()
	enginetest.Do(t, &s, "construct:walls")
	if s.Military != 1 {
		t.Fatalf("pawn %d, want +1 after 2 shields from -1", s.Military)
	}
	if s.Age != 3 || s.Phase != engine.PhaseChooseStarter || s.ToMove != 1 || s.Active != 0 {
		t.Fatalf("age %d, phase %s, to move %d, active %d; want Age III, seat 1 (weaker) choosing, seat 0 last active",
			s.Age, s.Phase, s.ToMove, s.Active)
	}
	enginetest.Do(t, &s, "starter:1")
	if s.ToMove != 1 {
		t.Fatalf("seat 1 chose itself but seat %d is to move", s.ToMove)
	}
	keys := enginetest.Keys(&s)
	var discard string
	for _, k := range keys {
		if strings.HasPrefix(k, "discard:") {
			discard = k
			break
		}
	}
	enginetest.Do(t, &s, discard)
	if s.ToMove != 0 {
		t.Fatalf("after the starter's move, seat %d is to move; want seat 0", s.ToMove)
	}
}

// TestAuditR2FlowLastCardShieldCentresThePawnUnderLastActiveStarts pins
// ages-scoring.md "Who starts the next age" and "For developers" with the
// literal centre-pawn option (decisions.md "Centred pawn", alternative):
// the pawn is read when the age ends. A 1-shield last card that brings the
// pawn back to the centre makes the position centred, so under
// LastActiveStarts the last active player starts with no choice phase. Read
// before the shield, the pawn would be on the mover's side and open a
// choice instead.
func TestAuditR2FlowLastCardShieldCentresThePawnUnderLastActiveStarts(t *testing.T) {
	for _, c := range []struct {
		seat engine.Seat
		pawn int
	}{{1, 1}, {0, -1}} {
		s := enginetest.New(t, 106).Options(engine.Options{CenterPawn: engine.LastActiveStarts}).
			Age(1).OnlyLeft("garrison").Pawn(c.pawn).Coins(c.seat, 10).ToMove(c.seat).State()
		ev := enginetest.Do(t, &s, "construct:garrison")
		if s.Military != 0 {
			t.Fatalf("seat %d: pawn %d, want 0", c.seat, s.Military)
		}
		if s.Age != 2 || s.Phase != engine.PhasePlay || s.ToMove != c.seat || s.Active != c.seat {
			t.Errorf("seat %d: age %d, phase %s, to move %d, active %d; want Age II started directly by seat %d",
				c.seat, s.Age, s.Phase, s.ToMove, s.Active, c.seat)
		}
		if auditR2FlowCount(ev, engine.EvStarterChosen) != 0 {
			t.Errorf("seat %d: a starter was chosen under LastActiveStarts with a centred pawn", c.seat)
		}
	}
}

// TestAuditR2FlowExtraTurnUsedOnTheLastCardIsNotLost pins turn.md "Extra
// turns" and ages-scoring.md "The end of an age": only an extra turn earned
// on the last turn of an age is lost. An extra turn earned on the second to
// last card is played, and the player takes the last card in it; nothing is
// lost then, and with the pawn centred that player, the last active one,
// chooses the next starter (decisions.md "Centred pawn").
func TestAuditR2FlowExtraTurnUsedOnTheLastCardIsNotLost(t *testing.T) {
	s := enginetest.New(t, 107).Age(1).OnlyLeft("altar", "baths").Wonders(0, "sphinx").Coins(0, 30).ToMove(0).State()
	enginetest.Do(t, &s, "wonder:sphinx:altar")
	if s.ToMove != 0 || s.Age != 1 {
		t.Fatalf("after the Sphinx: to move %d, age %d", s.ToMove, s.Age)
	}
	ev := enginetest.Do(t, &s, "discard:baths")
	if n := auditR2FlowCount(ev, engine.EvExtraTurnLost); n != 0 {
		t.Errorf("an extra turn was reported lost although it had been played: events %v", auditR2FlowKinds(ev))
	}
	if s.Age != 2 || s.Phase != engine.PhaseChooseStarter || s.ToMove != 0 || s.Active != 0 {
		t.Errorf("age %d, phase %s, to move %d, active %d; want seat 0 choosing the Age II starter", s.Age, s.Phase, s.ToMove, s.Active)
	}
}

// TestAuditR2FlowPairOnTheLastCardHoldsTheAgeOpen pins ages-scoring.md "The
// end of an age" ("including any pending choices it opened") and turn.md
// "The end of a turn" for a science pair completed by the last card of Age
// I: the next age is not laid out while the token choice is open. Taking
// Law then completes six different symbols (military-science.md
// "Scientific supremacy": the token after a pair can be the sixth) and the
// game ends in Age I with no new age; taking another token lets the age
// end, and with the pawn centred the last active player chooses.
func TestAuditR2FlowPairOnTheLastCardHoldsTheAgeOpen(t *testing.T) {
	for _, tc := range []struct {
		token string
		win   bool
	}{{"law", true}, {"agriculture", false}} {
		t.Run(tc.token, func(t *testing.T) {
			s := enginetest.New(t, 108).Age(1).OnlyLeft("workshop").
				City(0, "laboratory", "scriptorium", "pharmacist", "apothecary", "academy").
				BoardTokens("law", "agriculture", "economy", "masonry", "philosophy").
				Coins(0, 10).ToMove(0).State()
			ev := enginetest.Do(t, &s, "construct:workshop")
			if s.Phase != engine.PhasePickBoardToken || s.Age != 1 || auditR2FlowCount(ev, engine.EvAgeStart) != 0 {
				t.Fatalf("after the pair on the last card: phase %s, age %d, events %v; want the token choice still in Age I",
					s.Phase, s.Age, auditR2FlowKinds(ev))
			}
			ev = enginetest.Do(t, &s, "token:"+tc.token)
			if tc.win {
				if !s.Over() || s.Result.Kind != engine.VictoryScience || s.Result.Winner != 0 || s.Age != 1 {
					t.Fatalf("Law as the sixth symbol: phase %s, result %+v, age %d", s.Phase, s.Result, s.Age)
				}
				if auditR2FlowCount(ev, engine.EvAgeStart) != 0 {
					t.Fatalf("a new age started after a science supremacy: %v", auditR2FlowKinds(ev))
				}
				return
			}
			if s.Over() || s.Age != 2 || s.Phase != engine.PhaseChooseStarter || s.ToMove != 0 {
				t.Fatalf("after the token: over %v, age %d, phase %s, to move %d", s.Over(), s.Age, s.Phase, s.ToMove)
			}
			if s.Players[0].Coins != 10-2+6 {
				t.Fatalf("coins %d, want 14 (10, minus 2 for papyrus, plus Agriculture's 6)", s.Players[0].Coins)
			}
		})
	}
}

// TestAuditR2FlowChainedChoicesOnTheLastCardOfAgeIIIAreScored pins
// ages-scoring.md "The end of the game" (civilian scoring after the last
// card of Age III "and that turn has finished, including its pending
// choices"; the extra turn earned is lost) with a chain of two choices
// (turn.md "The end of a turn": "One choice can open another"): the
// Mausoleum, given an extra turn by Theology, rebuilds University, which
// pairs with Observatory and opens a token pick. The game must not end
// before the token is taken, and Philosophy's 7 points must be scored.
func TestAuditR2FlowChainedChoicesOnTheLastCardOfAgeIIIAreScored(t *testing.T) {
	s := enginetest.New(t, 109).Age(3).OnlyLeft("palace").
		Wonders(0, "mausoleum").Tokens(0, "theology").City(0, "observatory").Discard("university").
		BoardTokens("philosophy", "agriculture", "economy", "law", "masonry").
		Coins(0, 40).ToMove(0).State()
	ev := enginetest.Do(t, &s, "wonder:mausoleum:palace")
	if s.Over() || s.Phase != engine.PhaseBuildFromDiscard {
		t.Fatalf("after the Mausoleum on the last card: over %v, phase %s", s.Over(), s.Phase)
	}
	all := slices.Clone(ev)
	ev = enginetest.Do(t, &s, "revive:university")
	all = append(all, ev...)
	if s.Over() || s.Phase != engine.PhasePickBoardToken {
		t.Fatalf("after the rebuild: over %v, phase %s; want the token pick", s.Over(), s.Phase)
	}
	ev = enginetest.Do(t, &s, "token:philosophy")
	all = append(all, ev...)
	if !s.Over() || s.Result.Kind != engine.VictoryCivilian {
		t.Fatalf("after the token: phase %s, result %+v; want civilian scoring", s.Phase, s.Result)
	}
	if auditR2FlowCount(all, engine.EvExtraTurnLost) != 1 {
		t.Errorf("Theology's extra turn earned on the last card should be reported lost once: %v", auditR2FlowKinds(all))
	}
	sc := s.Result.Scores[0]
	if sc.Tokens != 7 || sc.Green != 4 || sc.Wonders != 2 {
		t.Errorf("seat 0 score %+v; want tokens 7 (Philosophy), green 4 (University and Observatory), wonders 2 (Mausoleum)", sc)
	}
	if s.Result.Scores != engine.Scores(&s) {
		t.Errorf("result scores %+v differ from the final position's %+v", s.Result.Scores, engine.Scores(&s))
	}
}

// TestAuditR2FlowLibraryLawOnTheLastCardOfAgeIIIWinsByScience pins
// decisions.md "Supremacy timing" ("on the last card of Age III it wins
// without civilian scoring") and military-science.md "Scientific
// supremacy" (Law from the Great Library can be the sixth symbol).
func TestAuditR2FlowLibraryLawOnTheLastCardOfAgeIIIWinsByScience(t *testing.T) {
	s := enginetest.New(t, 110).Age(3).OnlyLeft("palace").
		Wonders(0, "great-library").City(0, "scriptorium", "pharmacist", "workshop", "apothecary", "academy").
		BoardTokens("agriculture", "economy", "masonry", "philosophy", "theology").BoxOrder("law").
		Coins(0, 40).ToMove(0).State()
	enginetest.Do(t, &s, "wonder:great-library:palace")
	if s.Over() || s.Phase != engine.PhasePickLibraryToken {
		t.Fatalf("after the Great Library: over %v, phase %s", s.Over(), s.Phase)
	}
	ev := enginetest.Do(t, &s, "token:law")
	if !s.Over() || s.Result.Kind != engine.VictoryScience || s.Result.Winner != 0 {
		t.Fatalf("result %+v; want a science victory for seat 0, not civilian scoring", s.Result)
	}
	if n := auditR2FlowCount(ev, engine.EvGameOver); n != 1 {
		t.Fatalf("%d game-over events, want 1: %v", n, auditR2FlowKinds(ev))
	}
}

// TestAuditR2FlowLastBoardTokenThenAPairOnAnEmptyBoard pins
// military-science.md "Pairs" ("If no tokens are left on the board, the
// pair gives nothing") and decisions.md "Forced choices" (a token must be
// taken if the board has any) at the boundary of the last board token: the
// first pair offers exactly that token; the second pair, with the board
// empty, opens nothing and the turn passes.
func TestAuditR2FlowLastBoardTokenThenAPairOnAnEmptyBoard(t *testing.T) {
	s := enginetest.New(t, 111).Age(2).City(0, "scriptorium", "pharmacist").
		Accessible("library", "dispensary").
		Tokens(1, "agriculture", "architecture", "mathematics", "urbanism").
		BoardTokens("philosophy").ToMove(0).State()
	enginetest.Do(t, &s, "construct:library")
	if s.Phase != engine.PhasePickBoardToken {
		t.Fatalf("phase %s after the quill pair, want the board token pick", s.Phase)
	}
	if keys := enginetest.Keys(&s); !slices.Equal(keys, []string{"token:philosophy"}) {
		t.Fatalf("legal %v, want exactly the last board token", keys)
	}
	enginetest.Do(t, &s, "token:philosophy")
	if !s.BoardTokens.Empty() || s.ToMove != 1 {
		t.Fatalf("board %v, to move %d after the pick", s.BoardTokens, s.ToMove)
	}
	var discard string
	for _, k := range enginetest.Keys(&s) {
		if strings.HasPrefix(k, "discard:") && k != "discard:dispensary" {
			discard = k
			break
		}
	}
	if discard == "" {
		t.Fatalf("test setup: seat 1 has no card to discard other than dispensary: %v", enginetest.Keys(&s))
	}
	enginetest.Do(t, &s, discard)
	ev := enginetest.Do(t, &s, "construct:dispensary")
	if s.Phase != engine.PhasePlay || s.ToMove != 1 || auditR2FlowCount(ev, engine.EvToken) != 0 {
		t.Fatalf("mortar pair on an empty board: phase %s, to move %d, events %v; want nothing and seat 1 to move",
			s.Phase, s.ToMove, auditR2FlowKinds(ev))
	}
	if s.Players[0].Tokens.Count() != 1 {
		t.Fatalf("seat 0 has %d tokens, want 1", s.Players[0].Tokens.Count())
	}
}

// TestAuditR2FlowTheologyFromAMausoleumPairDoesNotReachTheMausoleum pins
// wonders-tokens.md "Progress tokens" ("lasting effects apply to everything
// the player does afterwards, never to what they did before") and
// "Theology" (every wonder you build afterwards): Theology taken from a
// pair that the Mausoleum's rebuild completed, within the Mausoleum's own
// turn, does not give the Mausoleum an extra turn.
func TestAuditR2FlowTheologyFromAMausoleumPairDoesNotReachTheMausoleum(t *testing.T) {
	s := enginetest.New(t, 112).Age(2).Wonders(0, "mausoleum").City(0, "workshop").Discard("laboratory").
		BoardTokens("theology", "agriculture", "economy", "law", "masonry").
		Accessible("rostrum").Coins(0, 40).ToMove(0).State()
	all := enginetest.Do(t, &s, "wonder:mausoleum:rostrum", "revive:laboratory")
	if s.Phase != engine.PhasePickBoardToken {
		t.Fatalf("phase %s, want the pair's token pick", s.Phase)
	}
	all = append(all, enginetest.Do(t, &s, "token:theology")...)
	if auditR2FlowCount(all, engine.EvExtraTurn) != 0 || s.ExtraTurn {
		t.Errorf("Theology taken after the Mausoleum gave it an extra turn: events %v", auditR2FlowKinds(all))
	}
	if s.ToMove != 1 || s.Active != 1 {
		t.Errorf("to move %d, active %d; want seat 1's turn", s.ToMove, s.Active)
	}
}

// TestAuditR2FlowSeventhWonderRemovesTheBuildersOwnLastWonder pins
// wonders-tokens.md "The seven-wonder rule" (the one wonder still unbuilt
// goes back to the box "whichever player drafted it", here the builder's
// own; from then on no wonder can be built; it scores nothing) together
// with the Sphinx's extra turn, which is still played: the extra turn
// offers no wonder key.
func TestAuditR2FlowSeventhWonderRemovesTheBuildersOwnLastWonder(t *testing.T) {
	s := enginetest.New(t, 113).Age(1).
		Wonders(0, "sphinx", "hanging-gardens", "appian-way", "piraeus").
		Wonders(1, "pyramids", "colossus", "great-lighthouse", "temple-of-artemis").
		Built(1, "pyramids", "").Built(1, "colossus", "").Built(1, "great-lighthouse", "").Built(1, "temple-of-artemis", "").
		Built(0, "hanging-gardens", "").Built(0, "appian-way", "").
		Accessible("altar", "baths").Coins(0, 40).ToMove(0).State()
	ev := enginetest.Do(t, &s, "wonder:sphinx:altar")
	removed := false
	for _, e := range ev {
		if e.Kind == engine.EvWonderRemoved && e.Seat == 0 && e.Wonder.String() == "piraeus" {
			removed = true
		}
	}
	if !removed || !s.Players[0].Wonders[3].Removed {
		t.Fatalf("Piraeus (seat 0's last unbuilt wonder) was not removed: events %v", auditR2FlowKinds(ev))
	}
	if s.Phase != engine.PhasePlay || s.ToMove != 0 {
		t.Fatalf("the Sphinx's extra turn: phase %s, to move %d", s.Phase, s.ToMove)
	}
	for _, k := range enginetest.Keys(&s) {
		if strings.HasPrefix(k, "wonder:") {
			t.Fatalf("%s is legal after seven wonders", k)
		}
	}
	if got := engine.Scores(&s)[0].Wonders; got != 6+3+3 {
		t.Errorf("seat 0 wonder points %d, want 12 (Sphinx 6, Hanging Gardens 3, Appian Way 3; Piraeus removed)", got)
	}
}

// TestAuditR2FlowPairDuringAnExtraTurnThenTheTurnPasses pins turn.md
// "Extra turns" (one extra turn per granting wonder) and "The end of a
// turn": a pair completed during the extra turn opens its token choice for
// the same player, and once it is settled the turn passes to the opponent,
// because the extra turn has been used up.
func TestAuditR2FlowPairDuringAnExtraTurnThenTheTurnPasses(t *testing.T) {
	s := enginetest.New(t, 114).Age(1).Wonders(0, "sphinx").City(0, "laboratory").
		Accessible("altar", "workshop").Coins(0, 30).ToMove(0).State()
	enginetest.Do(t, &s, "wonder:sphinx:altar", "construct:workshop")
	if s.Phase != engine.PhasePickBoardToken || s.ToMove != 0 {
		t.Fatalf("phase %s, to move %d; want seat 0's token pick", s.Phase, s.ToMove)
	}
	tk, _ := s.BoardTokens.Next(1)
	if tk == auditR2FlowToken(t, "theology") {
		tk, _ = s.BoardTokens.Next(tk + 1)
	}
	enginetest.Do(t, &s, "token:"+tk.String())
	if s.Phase != engine.PhasePlay || s.ToMove != 1 || s.Active != 1 {
		t.Fatalf("after the token: phase %s, to move %d, active %d; want seat 1", s.Phase, s.ToMove, s.Active)
	}
}

// TestAuditR2FlowMilitarySupremacyIsTheLastEventOfTheAction pins
// military-science.md "Military supremacy" ("Whatever the action would
// still have done after the move is skipped") and turn.md "The end of a
// turn" (a supremacy does not wait for the end of the turn): a card that
// reaches the capital and frees a face-down card ends the game with nothing
// after it, so no reveal follows the game-over event.
func TestAuditR2FlowMilitarySupremacyIsTheLastEventOfTheAction(t *testing.T) {
	s := enginetest.New(t, 115).Age(1).OnlyLeft(auditR2FlowSixWith("garrison")...).
		Pawn(8).Coins(0, 10).ToMove(0).State()
	auditR2FlowCheckHiddenWorkshop(t, &s)
	ev := enginetest.Do(t, &s, "construct:garrison")
	if !s.Over() || s.Result.Kind != engine.VictoryMilitary || s.Result.Winner != 0 || s.Military != 9 {
		t.Fatalf("result %+v, pawn %d; want seat 0's military victory", s.Result, s.Military)
	}
	if len(ev) == 0 || ev[len(ev)-1].Kind != engine.EvGameOver {
		t.Fatalf("events %v; the game-over event must be the last", auditR2FlowKinds(ev))
	}
	if auditR2FlowCount(ev, engine.EvReveal) != 0 {
		t.Errorf("a card was revealed in the action that ended the game: %v", auditR2FlowKinds(ev))
	}
}

// TestAuditR2FlowScoringCountsTheLastDiscardsCoins pins ages-scoring.md
// "The end of the game" (the game ends when the last card of Age III has
// been taken and that turn has finished; then the players count) and "The
// eight scoring categories" item 7 (1 point per full 3 coins). Discarding
// the last card with exactly 1 coin gives 2 more (turn.md "2. Discard it
// for coins"), so 3 coins, worth 1 point.
func TestAuditR2FlowScoringCountsTheLastDiscardsCoins(t *testing.T) {
	s := enginetest.New(t, 116).Age(3).OnlyLeft("palace").Coins(0, 1).ToMove(0).State()
	enginetest.Do(t, &s, "discard:palace")
	if !s.Over() || s.Result.Kind != engine.VictoryCivilian {
		t.Fatalf("phase %s, result %+v", s.Phase, s.Result)
	}
	if s.Players[0].Coins != 3 || s.Result.Scores[0].Coins != 1 {
		t.Errorf("coins %d, coin points %d; want 3 coins worth 1 point", s.Players[0].Coins, s.Result.Scores[0].Coins)
	}
}

// TestAuditR2FlowSetupComposition pins setup.md "Setup steps" and "The
// wonder draft" over many seeds: 7 coins each, the pawn centred with four
// looting tokens; five tokens on the board and five different ones in the
// box; eight draft wonders and four box wonders, all twelve different;
// each Age I and Age II deal is 20 of that age's 23 cards with the other 3
// removed; Age III is 17 of its 20 cards plus exactly 3 guilds, with 3
// Age III cards removed and the other 4 guilds in the box.
func TestAuditR2FlowSetupComposition(t *testing.T) {
	for seed := uint64(1); seed <= 150; seed++ {
		s := engine.NewGame(seed, engine.Options{})
		if s.Players[0].Coins != 7 || s.Players[1].Coins != 7 || s.Military != 0 || s.Looting != engine.LootAll {
			t.Fatalf("seed %d: coins %d/%d, pawn %d, looting %#x", seed, s.Players[0].Coins, s.Players[1].Coins, s.Military, s.Looting)
		}
		toks := s.BoardTokens
		for _, tk := range s.BoxTokens {
			if tk == 0 || toks.Has(tk) {
				t.Fatalf("seed %d: box token %d is missing or also on the board", seed, tk)
			}
			toks.Add(tk)
		}
		if s.BoardTokens.Count() != 5 || toks.Count() != 10 {
			t.Fatalf("seed %d: %d board tokens, %d tokens in all", seed, s.BoardTokens.Count(), toks.Count())
		}
		var ws [catalog.NumWonders + 1]bool
		for _, w := range append(s.DraftOrder[:], s.BoxWonders[:]...) {
			if w == 0 || int(w) > catalog.NumWonders || ws[w] {
				t.Fatalf("seed %d: wonder %d missing or repeated", seed, w)
			}
			ws[w] = true
		}
		var seen catalog.CardSet
		for age := 1; age <= 3; age++ {
			guilds := 0
			for _, c := range s.Deals[age-1] {
				d := catalog.Card(c)
				if d == nil || d.Age != uint8(age) || seen.Has(c) {
					t.Fatalf("seed %d: Age %d deal holds card %d", seed, age, c)
				}
				seen.Add(c)
				if d.IsGuild() {
					guilds++
				}
			}
			for _, c := range s.Removed[age-1] {
				d := catalog.Card(c)
				if d == nil || d.Age != uint8(age) || d.IsGuild() || seen.Has(c) {
					t.Fatalf("seed %d: Age %d removed card %d", seed, age, c)
				}
				seen.Add(c)
			}
			want := 0
			if age == 3 {
				want = 3
			}
			if guilds != want {
				t.Fatalf("seed %d: Age %d deal has %d guilds, want %d", seed, age, guilds, want)
			}
		}
		for _, c := range s.BoxGuilds {
			d := catalog.Card(c)
			if d == nil || !d.IsGuild() || seen.Has(c) {
				t.Fatalf("seed %d: box guild %d", seed, c)
			}
			seen.Add(c)
		}
		if seen.Count() != 73 {
			t.Fatalf("seed %d: %d cards placed, want 73", seed, seen.Count())
		}
	}
}

// TestAuditR2FlowRandomGamesFollowTheTurnRules plays whole games from the
// draft with random and simple policies under both centre-pawn options and
// checks, after every action, the flow rules that any position must obey:
//
//   - turn.md "Taking a card": every accessible card is face up when a turn
//     begins, a face-down card lies on an even row (setup.md "Face-up and
//     face-down rows"), and every play key names an accessible card, each
//     with its discard key (turn.md "There is always a move");
//   - turn.md "The end of a turn": a revealed card was face down before;
//     the active player makes pending choices; after a turn the same player
//     plays again exactly when an extra turn was earned, and at the end of
//     an age an earned extra turn is reported lost;
//   - ages-scoring.md "Who starts the next age": the next age is laid out
//     (20 cards) before the choice, and the chooser is the weaker player,
//     or the last active one with the pawn centred (who starts directly
//     under LastActiveStarts);
//   - setup.md: the first player starts Age I, each age's 20 cards are all
//     taken before it ends, civilian scoring happens only at the end of Age
//     III, and the removed cards and box guilds never appear.
func TestAuditR2FlowRandomGamesFollowTheTurnRules(t *testing.T) {
	var st auditR2FlowStats
	for seed := uint64(1); seed <= 30; seed++ {
		for _, cp := range []engine.CenterPawnRule{engine.LastActiveChooses, engine.LastActiveStarts} {
			opts := engine.Options{CenterPawn: cp}
			auditR2FlowPlayAndCheck(t, seed, opts, enginetest.NewRandomPolicy(seed*7+1), &st)
			auditR2FlowPlayAndCheck(t, seed, opts, enginetest.NewSimplePolicy(seed*7+2), &st)
		}
	}
	if st.extraTurns == 0 || st.reveals == 0 || st.lost == 0 || st.centred == 0 || st.offCentre == 0 || st.guilds == 0 {
		t.Fatalf("the random games did not reach every case: %+v", st)
	}
	t.Logf("reached: %+v", st)
}

func auditR2FlowIsPending(p engine.Phase) bool {
	switch p {
	case engine.PhaseDestroyCard, engine.PhaseBuildFromDiscard, engine.PhasePickBoardToken, engine.PhasePickLibraryToken:
		return true
	}
	return false
}

func auditR2FlowPlayAndCheck(t *testing.T, seed uint64, opts engine.Options, pol enginetest.Policy, st *auditR2FlowStats) {
	t.Helper()
	s := engine.NewGame(seed, opts)
	var hidden catalog.CardSet
	for a := range s.Removed {
		for _, c := range s.Removed[a] {
			hidden.Add(c)
		}
	}
	for _, c := range s.BoxGuilds {
		hidden.Add(c)
	}
	legal := make([]engine.Action, 0, engine.MaxLegalActions)
	var turnSeat engine.Seat
	var turnAge uint8
	extra := false
	taken := 0
	fail := func(ply int, format string, args ...any) {
		t.Helper()
		t.Fatalf("seed %d, options %+v, ply %d: "+format, append([]any{seed, opts, ply}, args...)...)
	}
	for ply := 0; !s.Over(); ply++ {
		if ply > 600 {
			fail(ply, "the game did not end")
		}
		legal = engine.LegalActions(&s, legal[:0])
		if s.Phase == engine.PhasePlay {
			acc := engine.Accessible(&s)
			for i, sl := range s.Slots {
				if sl.Card == 0 {
					continue
				}
				if hidden.Has(sl.Card) {
					fail(ply, "removed card %s is in the structure", sl.Card)
				}
				if acc&(1<<i) != 0 && !sl.FaceUp {
					fail(ply, "accessible slot %d is face down at the start of a turn", i)
				}
				if !sl.FaceUp && engine.SlotPos(s.Age, i).Row%2 != 0 {
					fail(ply, "slot %d on odd row %d is face down", i, engine.SlotPos(s.Age, i).Row)
				}
			}
			keys := make([]string, 0, len(legal))
			for _, a := range legal {
				keys = append(keys, a.Key())
				i := engine.SlotOf(&s, a.Card)
				if i < 0 || acc&(1<<i) == 0 {
					fail(ply, "legal %s names a card that is not accessible", a.Key())
				}
			}
			for i, sl := range s.Slots {
				if acc&(1<<i) != 0 && !auditR2FlowHasKey(keys, "discard:"+sl.Card.String()) {
					fail(ply, "no discard key for accessible card %s", sl.Card)
				}
			}
		}
		if auditR2FlowIsPending(s.Phase) && s.ToMove != s.Active {
			fail(ply, "seat %d decides a pending choice in seat %d's turn", s.ToMove, s.Active)
		}
		if len(legal) == 0 {
			fail(ply, "no legal action in phase %s", s.Phase)
		}
		a := pol.Choose(&s, legal)
		pre := s
		var ev []engine.Event
		if err := engine.Apply(&s, a, &ev); err != nil {
			fail(ply, "%s: %v", a.Key(), err)
		}
		if pre.Phase == engine.PhaseWonderDraft && s.Phase == engine.PhasePlay {
			if s.Age != 1 || s.ToMove != s.FirstSeat || s.Active != s.FirstSeat {
				fail(ply, "after the draft: age %d, seat %d to move; want the first player %d", s.Age, s.ToMove, s.FirstSeat)
			}
		}
		if pre.Phase == engine.PhasePlay {
			turnSeat, turnAge = pre.Active, pre.Age
			taken++
		}
		for _, e := range ev {
			switch e.Kind {
			case engine.EvExtraTurn:
				extra = true
				st.extraTurns++
			case engine.EvReveal:
				st.reveals++
				i := int(e.A)
				if pre.Slots[i].FaceUp || pre.Slots[i].Card != e.Card {
					fail(ply, "reveal of slot %d (%s), which was not a face-down card", i, e.Card)
				}
			}
		}
		turnOver := (pre.Phase == engine.PhasePlay || auditR2FlowIsPending(pre.Phase)) && !auditR2FlowIsPending(s.Phase)
		if !turnOver {
			continue
		}
		lost := auditR2FlowCount(ev, engine.EvExtraTurnLost)
		switch {
		case s.Over():
			if s.Result.Kind == engine.VictoryCivilian && (turnAge != 3 || engine.Occupied(&s) != 0 || taken != engine.SlotsPerAge) {
				fail(ply, "civilian end in age %d with %d cards taken", turnAge, taken)
			}
		case s.Age != turnAge:
			if s.Age != turnAge+1 || taken != engine.SlotsPerAge {
				fail(ply, "age %d to %d after %d cards", turnAge, s.Age, taken)
			}
			if auditR2FlowPopcount(engine.Occupied(&s)) != engine.SlotsPerAge {
				fail(ply, "age %d is not laid out before its starter is decided", s.Age)
			}
			if (lost == 1) != extra || lost > 1 {
				fail(ply, "extra turn earned %v, lost events %d", extra, lost)
			}
			if s.Active != turnSeat {
				fail(ply, "active %d at the start of age %d, want the last active seat %d", s.Active, s.Age, turnSeat)
			}
			var chooser engine.Seat
			choose := true
			switch {
			case s.Military > 0:
				chooser = 1
			case s.Military < 0:
				chooser = 0
			default:
				chooser = turnSeat
				choose = opts.CenterPawn != engine.LastActiveStarts
			}
			if choose && (s.Phase != engine.PhaseChooseStarter || s.ToMove != chooser) {
				fail(ply, "pawn %d: phase %s, seat %d to move; want seat %d to choose", s.Military, s.Phase, s.ToMove, chooser)
			}
			if !choose && (s.Phase != engine.PhasePlay || s.ToMove != turnSeat) {
				fail(ply, "centred pawn, literal option: phase %s, seat %d to move; want seat %d to start", s.Phase, s.ToMove, turnSeat)
			}
			// decisions.md "Guild back": a guild shows the guild back, every
			// other card its age's back.
			for i, sl := range s.Slots {
				want := catalog.AgeBack(s.Age)
				if catalog.Card(sl.Card).IsGuild() {
					want = catalog.GuildBack
					st.guilds++
				}
				if sl.Back != want {
					fail(ply, "age %d slot %d (%s) shows back %s, want %s", s.Age, i, sl.Card, sl.Back, want)
				}
			}
			if lost == 1 {
				st.lost++
			}
			if s.Military == 0 {
				st.centred++
			} else {
				st.offCentre++
			}
			taken = 0
		default:
			want := turnSeat.Opp()
			if extra {
				want = turnSeat
			}
			if s.Phase != engine.PhasePlay || s.ToMove != want || s.Active != want || lost != 0 {
				fail(ply, "after the turn (extra %v): phase %s, to move %d, active %d, lost %d; want seat %d",
					extra, s.Phase, s.ToMove, s.Active, lost, want)
			}
		}
		extra = false
	}
	var seen catalog.CardSet
	for _, p := range s.Players {
		seen = seen.Or(p.City)
		for _, w := range p.Wonders {
			if w.Tucked != 0 {
				seen.Add(w.Tucked)
			}
		}
	}
	seen = seen.Or(s.Discard)
	if !seen.And(hidden).Empty() {
		t.Fatalf("seed %d: removed cards or box guilds reached play: %v", seed, seen.And(hidden))
	}
}

// TestAuditR2FlowTheologyFromTheLibraryOnTheLastCardLosesNothing pins
// wonders-tokens.md "Theology" (only wonders built afterwards) and
// ages-scoring.md "The end of an age" (only an extra turn earned on that
// turn is lost): the Great Library built with the last card of Age I,
// keeping Theology, earns no extra turn, so none is reported lost, and with
// the pawn centred the builder, the last active player, chooses the next
// starter (decisions.md "Centred pawn").
func TestAuditR2FlowTheologyFromTheLibraryOnTheLastCardLosesNothing(t *testing.T) {
	s := enginetest.New(t, 117).Age(1).OnlyLeft("altar").Wonders(1, "great-library").
		BoardTokens("agriculture", "economy", "masonry", "philosophy", "law").BoxOrder("theology").
		Coins(1, 40).ToMove(1).State()
	all := enginetest.Do(t, &s, "wonder:great-library:altar")
	if s.Phase != engine.PhasePickLibraryToken || s.Age != 1 {
		t.Fatalf("phase %s, age %d; want the library pick still in Age I", s.Phase, s.Age)
	}
	all = append(all, enginetest.Do(t, &s, "token:theology")...)
	if auditR2FlowCount(all, engine.EvExtraTurn) != 0 || auditR2FlowCount(all, engine.EvExtraTurnLost) != 0 {
		t.Errorf("Theology kept from the Great Library touched the Library's own turn: events %v", auditR2FlowKinds(all))
	}
	if s.Age != 2 || s.Phase != engine.PhaseChooseStarter || s.ToMove != 1 || s.Active != 1 {
		t.Errorf("age %d, phase %s, to move %d, active %d; want seat 1 choosing the Age II starter", s.Age, s.Phase, s.ToMove, s.Active)
	}
}

// auditR2FlowStats counts what the random games reached, so the property
// test cannot pass vacuously.
type auditR2FlowStats struct {
	extraTurns, reveals, lost, centred, offCentre, guilds int
}
