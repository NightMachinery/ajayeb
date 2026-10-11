package engine_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// This file is the M10.R1 rules audit, round 2, through the military and
// science lens. It is black-box: it builds positions with the scenario
// builder and checks the engine against the rules in docs/rules/ and the
// rulebook. It goes after combinations of three or more effects that the
// earlier rounds (edgecases_milsci_r1_test.go, audit_milsci_r2_test.go and
// the other audit files) did not put together: a Mausoleum revive that wins
// on the last card of an age, looting on the way into a capital, a
// Theology extra turn that a supremacy cancels, Economy income followed by
// two looting tokens, Law going back to the box after a Great Library pick,
// and the view's track and science panels after such moves.

// auditR3MilsciCard returns the id of a card slug.
func auditR3MilsciCard(t *testing.T, slug string) catalog.CardID {
	t.Helper()
	c, ok := catalog.CardBySlug(slug)
	if !ok {
		t.Fatalf("unknown card %q", slug)
	}
	return c
}

// auditR3MilsciToken returns the id of a token slug.
func auditR3MilsciToken(t *testing.T, slug string) catalog.TokenID {
	t.Helper()
	tk, ok := catalog.TokenBySlug(slug)
	if !ok {
		t.Fatalf("unknown token %q", slug)
	}
	return tk
}

// auditR3MilsciOfKind returns the events of one kind, in order.
func auditR3MilsciOfKind(ev []engine.Event, k engine.EventKind) []engine.Event {
	var out []engine.Event
	for _, e := range ev {
		if e.Kind == k {
			out = append(out, e)
		}
	}
	return out
}

// auditR3MilsciIndex returns the index of the first event of kind k, or -1.
func auditR3MilsciIndex(ev []engine.Event, k engine.EventKind) int {
	for i, e := range ev {
		if e.Kind == k {
			return i
		}
	}
	return -1
}

// auditR3MilsciDiscardOther discards any accessible card other than the
// named ones, as a filler move for the opponent.
func auditR3MilsciDiscardOther(t *testing.T, s *engine.State, keep ...string) {
	t.Helper()
	for _, k := range enginetest.Keys(s) {
		slug, ok := strings.CutPrefix(k, "discard:")
		if ok && !slices.Contains(keep, slug) {
			enginetest.Do(t, s, k)
			return
		}
	}
	t.Fatalf("no discard is legal apart from %v: %v", keep, enginetest.Keys(s))
}

// auditR3MilsciEndsWith checks that the action's events end with exactly one
// game-over event of the given kind and winner, and that the action neither
// reports a lost extra turn nor lays out a new age.
func auditR3MilsciEndsWith(t *testing.T, ev []engine.Event, kind engine.VictoryKind, winner engine.Seat) {
	t.Helper()
	if len(ev) == 0 {
		t.Fatalf("no events")
	}
	want := engine.Event{Kind: engine.EvGameOver, A: int16(kind), B: int16(winner)}
	if last := ev[len(ev)-1]; last != want {
		t.Fatalf("last event %+v, want %+v; events %+v", last, want, ev)
	}
	if n := len(auditR3MilsciOfKind(ev, engine.EvGameOver)); n != 1 {
		t.Fatalf("%d game-over events, want 1: %+v", n, ev)
	}
	for _, k := range []engine.EventKind{engine.EvExtraTurnLost, engine.EvAgeStart} {
		if i := auditR3MilsciIndex(ev, k); i >= 0 {
			t.Fatalf("event %s after a supremacy: %+v", k, ev)
		}
	}
}

// auditR3MilsciSortedKeys returns the legal keys, sorted.
func auditR3MilsciSortedKeys(s *engine.State) []string {
	keys := enginetest.Keys(s)
	slices.Sort(keys)
	return keys
}

// TestAuditR3MilsciRevivedRedCardWithStrategyWinsOnTheLastCardOfAgeIII: the
// last card of Age III goes under the Mausoleum, built by a player who owns
// Strategy and Theology, with the pawn on space 7. Theology notes an extra
// turn; the revived Garrison gives 1 shield plus Strategy's 1, which reach
// the capital. The game ends by military supremacy, not by civilian
// scoring, and the extra turn is neither played nor reported lost.
// Rule: docs/rules/military-science.md "Where shields come from" (Strategy
// includes a red building rebuilt with the Mausoleum) and "Military
// supremacy"; docs/rules/decisions.md "Supremacy timing" ("on the last card
// of Age III it wins without civilian scoring"); docs/rules/ages-scoring.md
// "The end of the game".
func TestAuditR3MilsciRevivedRedCardWithStrategyWinsOnTheLastCardOfAgeIII(t *testing.T) {
	s := enginetest.New(t, 3001).Age(3).
		Wonders(0, "mausoleum").
		City(0, "clay-pool", "clay-pit", "glassworks", "glass-blower", "press").
		Tokens(0, "strategy", "theology").
		Discard("garrison").Pawn(7).
		OnlyLeft("obelisk").ToMove(0).State()

	ev := enginetest.Do(t, &s, "wonder:mausoleum:obelisk")
	if s.Over() || s.Phase != engine.PhaseBuildFromDiscard {
		t.Fatalf("phase %s, result %+v; want the Mausoleum's choice before the game ends", s.Phase, s.Result)
	}
	if len(auditR3MilsciOfKind(ev, engine.EvExtraTurn)) != 1 {
		t.Fatalf("want Theology's extra turn noted: %+v", ev)
	}
	ev = enginetest.Do(t, &s, "revive:garrison")

	if !s.Over() || s.Result.Kind != engine.VictoryMilitary || s.Result.Winner != 0 || s.Military != 9 {
		t.Fatalf("pawn %d, phase %s, result %+v; want seat 0's military victory at 9", s.Military, s.Phase, s.Result)
	}
	if s.ExtraTurn {
		t.Errorf("an extra turn is still pending after the game ended")
	}
	auditR3MilsciEndsWith(t, ev, engine.VictoryMilitary, 0)
}

// TestAuditR3MilsciRevivedSixthSymbolOnTheLastCardOfAgeIEndsTheGameInAgeI:
// the last card of Age I goes under the Mausoleum of a Theology owner with
// five different symbols. The revived University is the sixth symbol: the
// game ends at once in Age I. No Age II is laid out, and the extra turn
// Theology noted is simply dropped, not reported lost.
// Rule: docs/rules/military-science.md "Scientific supremacy" (the sixth
// symbol can come from a card rebuilt with the Mausoleum);
// docs/rules/ages-scoring.md "The end of an age" ("If a supremacy happens
// on the last card of an age, the game is over at once and no new age
// begins"); docs/rules/decisions.md "Supremacy timing".
func TestAuditR3MilsciRevivedSixthSymbolOnTheLastCardOfAgeIEndsTheGameInAgeI(t *testing.T) {
	s := enginetest.New(t, 3002).Age(1).
		Wonders(0, "mausoleum").
		City(0, "scriptorium", "workshop", "pharmacist", "apothecary", "study",
			"clay-pool", "clay-pit", "glassworks", "glass-blower", "press").
		Tokens(0, "theology").
		Discard("university").
		OnlyLeft("altar").ToMove(0).State()
	if n := engine.DistinctSymbols(&s.Players[0]); n != 5 {
		t.Fatalf("setup: %d distinct symbols, want 5", n)
	}

	ev := enginetest.Do(t, &s, "wonder:mausoleum:altar", "revive:university")

	if !s.Over() || s.Result.Kind != engine.VictoryScience || s.Result.Winner != 0 {
		t.Fatalf("phase %s, result %+v; want seat 0's science victory", s.Phase, s.Result)
	}
	if s.Age != 1 {
		t.Errorf("age %d, want 1 (no new age after a supremacy)", s.Age)
	}
	if s.ExtraTurn {
		t.Errorf("an extra turn is still pending after the game ended")
	}
	auditR3MilsciEndsWith(t, ev, engine.VictoryScience, 0)
}

// TestAuditR3MilsciLootOnTheWayIntoTheCapital: from space 5, Arsenal with
// Strategy moves the pawn 4 spaces. On the way it enters space 6, where the
// 5-coin token is still waiting, and then reaches the capital. The pawn is
// walked space by space, so the token is taken first and the game ends
// after it, as the last event of the action.
// Rule: docs/rules/military-science.md "Looting tokens" (one move can
// trigger tokens it passes) and "For developers" ("Pawn moves are walked
// space by space ... reaching 9 ends the game immediately"). The rulebook
// does not say whether looting on the way into a capital happens at all,
// since the game is over either way; this follows our own-words rule. The
// winner does not depend on it.
func TestAuditR3MilsciLootOnTheWayIntoTheCapital(t *testing.T) {
	s := enginetest.New(t, 3003).Age(3).
		City(0, "clay-pool", "clay-pit", "brickyard", "lumber-yard", "logging-camp").
		Tokens(0, "strategy").
		Coins(1, 7).Pawn(5).
		Accessible("arsenal").ToMove(0).State()

	ev := enginetest.Do(t, &s, "construct:arsenal")

	if !s.Over() || s.Result.Kind != engine.VictoryMilitary || s.Result.Winner != 0 || s.Military != 9 {
		t.Fatalf("pawn %d, phase %s, result %+v; want seat 0's military victory at 9", s.Military, s.Phase, s.Result)
	}
	pawn := auditR3MilsciOfKind(ev, engine.EvPawn)
	if len(pawn) != 1 || pawn[0].A != 5 || pawn[0].B != 9 {
		t.Fatalf("pawn events %+v, want one move from 5 to 9", pawn)
	}
	loots := auditR3MilsciOfKind(ev, engine.EvLoot)
	if want := []engine.Event{{Kind: engine.EvLoot, Seat: 1, A: 5, B: 5}}; !slices.Equal(loots, want) {
		t.Fatalf("loot events %+v, want %+v", loots, want)
	}
	if iP, iL := auditR3MilsciIndex(ev, engine.EvPawn), auditR3MilsciIndex(ev, engine.EvLoot); iP > iL {
		t.Fatalf("the loot comes before the pawn move: %+v", ev)
	}
	if s.Players[1].Coins != 2 {
		t.Errorf("seat 1 coins %d, want 2", s.Players[1].Coins)
	}
	auditR3MilsciEndsWith(t, ev, engine.VictoryMilitary, 0)
}

// TestAuditR3MilsciStrategyLandsOnSixAndTheViewFollows: the pawn stands on
// space 3, where the 2-coin token is already gone. Walls (2 shields) with
// Strategy (1 more) land exactly on space 6: only the 5-coin token is
// taken. The public view then shows the pawn on 6, the two tokens of seat
// 1's side gone and seat 0's two still there, and a score preview of 10
// military points for seat 0 and none for seat 1.
// Rule: docs/rules/military-science.md "Looting tokens" (a token triggers
// once; the space-6 token is worth 5), "Military points at the end" (zone
// 6 to 8 is worth 10 to the player ahead) and the track diagram (tokens at
// 3 and 6 on each side); docs/rules/ages-scoring.md "The score preview".
func TestAuditR3MilsciStrategyLandsOnSixAndTheViewFollows(t *testing.T) {
	s := enginetest.New(t, 3004).Age(2).
		City(0, "shelf-quarry").Tokens(0, "strategy").
		Coins(1, 7).Pawn(3).
		Accessible("walls").ToMove(0).State()

	ev := enginetest.Do(t, &s, "construct:walls")

	if s.Military != 6 {
		t.Fatalf("pawn %d, want 6", s.Military)
	}
	if loots := auditR3MilsciOfKind(ev, engine.EvLoot); !slices.Equal(loots, []engine.Event{{Kind: engine.EvLoot, Seat: 1, A: 5, B: 5}}) {
		t.Fatalf("loot events %+v, want only the 5-coin token", loots)
	}
	if s.Players[1].Coins != 2 {
		t.Fatalf("seat 1 coins %d, want 2", s.Players[1].Coins)
	}

	v := engine.ViewFor(&s, engine.SeatViewer(1))
	if v.Military.Position != 6 {
		t.Errorf("view pawn %d, want 6", v.Military.Position)
	}
	want := []view.LootingView{
		{At: -6, Seat: 0, Value: 5, Present: true},
		{At: -3, Seat: 0, Value: 2, Present: true},
		{At: 3, Seat: 1, Value: 2, Present: false},
		{At: 6, Seat: 1, Value: 5, Present: false},
	}
	if !slices.Equal(v.Military.Looting, want) {
		t.Errorf("view looting %+v, want %+v", v.Military.Looting, want)
	}
	if v.ScorePreview[0].Military != 10 || v.ScorePreview[1].Military != 0 {
		t.Errorf("preview military %d / %d, want 10 / 0", v.ScorePreview[0].Military, v.ScorePreview[1].Military)
	}
	if v.Players[1].Coins != 2 {
		t.Errorf("view: seat 1 coins %d, want 2", v.Players[1].Coins)
	}
}

// TestAuditR3MilsciForcedLawOnTheLastCardOfAgeIIIWinsByScience: the last
// card of Age III completes a pair for a player with five different
// symbols, and the board holds only Law. The pick is forced to Law, which
// is the sixth symbol: a science victory, not civilian scoring.
// Rule: docs/rules/military-science.md "Pairs" (the pick is forced when a
// token is available) and "Scientific supremacy" (Law taken from the board
// after a pair can be the sixth symbol); docs/rules/ages-scoring.md "The
// end of the game" (the last turn finishes with its pending choices before
// scoring); docs/rules/decisions.md "Supremacy timing".
func TestAuditR3MilsciForcedLawOnTheLastCardOfAgeIIIWinsByScience(t *testing.T) {
	s := enginetest.New(t, 3005).Age(3).
		City(0, "scriptorium", "workshop", "pharmacist", "apothecary", "study",
			"lumber-yard", "quarry", "glassworks", "glass-blower").
		Tokens(1, "mathematics", "strategy", "theology", "urbanism").
		BoardTokens("law").
		OnlyLeft("academy").ToMove(0).State()

	enginetest.Do(t, &s, "construct:academy")
	if s.Over() || s.Phase != engine.PhasePickBoardToken || s.ToMove != 0 {
		t.Fatalf("phase %s, to move %d, result %+v; want seat 0's token pick before the game ends", s.Phase, s.ToMove, s.Result)
	}
	if keys := enginetest.Keys(&s); !slices.Equal(keys, []string{"token:law"}) {
		t.Fatalf("legal keys %v, want only token:law", keys)
	}
	ev := enginetest.Do(t, &s, "token:law")

	if !s.Over() || s.Result.Kind != engine.VictoryScience || s.Result.Winner != 0 {
		t.Fatalf("phase %s, result %+v; want seat 0's science victory", s.Phase, s.Result)
	}
	auditR3MilsciEndsWith(t, ev, engine.VictoryScience, 0)
}

// TestAuditR3MilsciUrbanismCoinsComeBeforeThePair: Dispensary built free
// through Pharmacist's chain by an Urbanism owner with 0 coins completes the
// mortar pair. Urbanism's 4 coins are paid before the pair is reported; the
// pick then adds Agriculture's 6, and the turn passes.
// Rule: docs/rules/decisions.md "Effect order" (a building: coin rewards,
// first Urbanism's chain bonus, then shields, then the science symbol with a
// possible pair); docs/rules/wonders-tokens.md "Urbanism" and "Agriculture".
func TestAuditR3MilsciUrbanismCoinsComeBeforeThePair(t *testing.T) {
	s := enginetest.New(t, 3006).Age(2).
		City(0, "pharmacist").Tokens(0, "urbanism").Coins(0, 0).
		BoardTokens("agriculture", "philosophy", "law", "strategy", "theology").
		Accessible("dispensary").ToMove(0).State()

	ev := enginetest.Do(t, &s, "construct:dispensary")
	if s.Players[0].Coins != 4 {
		t.Fatalf("seat 0 coins %d, want 4 (Urbanism, nothing paid)", s.Players[0].Coins)
	}
	iCoins, iPair := auditR3MilsciIndex(ev, engine.EvCoins), auditR3MilsciIndex(ev, engine.EvPair)
	if iCoins < 0 || iPair < 0 || iCoins > iPair {
		t.Fatalf("want Urbanism's coins before the pair: %+v", ev)
	}
	if c := ev[iCoins]; c.Token != auditR3MilsciToken(t, "urbanism") || c.A != 4 {
		t.Fatalf("first coin event %+v, want Urbanism's 4", c)
	}
	if s.Phase != engine.PhasePickBoardToken {
		t.Fatalf("phase %s, want pickBoardToken", s.Phase)
	}
	enginetest.Do(t, &s, "token:agriculture")
	if s.Players[0].Coins != 10 {
		t.Fatalf("seat 0 coins %d, want 10", s.Players[0].Coins)
	}
	if s.Phase != engine.PhasePlay || s.ToMove != 1 {
		t.Fatalf("phase %s, seat %d to move; want seat 1's turn", s.Phase, s.ToMove)
	}
}

// TestAuditR3MilsciNinthShieldFromAWonderCancelsTheologysExtraTurn: a
// Theology owner builds a destroying wonder whose shield carries the pawn
// into the capital. The game ends inside the shield step, so neither the
// extra turn (noted after the shields) nor the destroy choice happens. The
// second case is seat 1 with Circus Maximus on the last card of Age II: no
// Age III is laid out and no extra turn is reported lost.
// Rule: docs/rules/wonders-tokens.md "The order of a wonder's effects"
// (shields at step 4, the extra turn at step 5, the special choice at step
// 7); docs/rules/military-science.md "Military supremacy" ("Whatever the
// action would still have done after the move is skipped", including on
// the last card of an age).
func TestAuditR3MilsciNinthShieldFromAWonderCancelsTheologysExtraTurn(t *testing.T) {
	t.Run("statue of zeus, seat 0", func(t *testing.T) {
		s := enginetest.New(t, 3007).Age(1).
			Wonders(0, "statue-of-zeus").
			City(0, "lumber-yard", "clay-pool", "quarry", "press", "drying-room").
			Tokens(0, "theology").
			City(1, "logging-camp").
			Pawn(8).Accessible("baths").ToMove(0).State()

		ev := enginetest.Do(t, &s, "wonder:statue-of-zeus:baths")

		if !s.Over() || s.Result.Kind != engine.VictoryMilitary || s.Result.Winner != 0 {
			t.Fatalf("phase %s, result %+v; want seat 0's military victory", s.Phase, s.Result)
		}
		if n := len(auditR3MilsciOfKind(ev, engine.EvExtraTurn)); n != 0 || s.ExtraTurn {
			t.Errorf("%d extra-turn events (pending %v): the extra turn comes after the shield that ended the game", n, s.ExtraTurn)
		}
		if !s.Players[1].City.Has(auditR3MilsciCard(t, "logging-camp")) {
			t.Errorf("the opponent's brown card was destroyed after the game ended")
		}
		auditR3MilsciEndsWith(t, ev, engine.VictoryMilitary, 0)
	})
	t.Run("circus maximus, seat 1, last card of age II", func(t *testing.T) {
		s := enginetest.New(t, 3008).Age(2).
			Wonders(1, "circus-maximus").
			City(1, "shelf-quarry", "lumber-yard", "glassworks").
			Tokens(1, "theology").
			City(0, "press").
			Pawn(-8).OnlyLeft("temple").ToMove(1).State()

		ev := enginetest.Do(t, &s, "wonder:circus-maximus:temple")

		if !s.Over() || s.Result.Kind != engine.VictoryMilitary || s.Result.Winner != 1 || s.Military != -9 {
			t.Fatalf("pawn %d, phase %s, result %+v; want seat 1's military victory at -9", s.Military, s.Phase, s.Result)
		}
		if s.Age != 2 {
			t.Errorf("age %d, want 2", s.Age)
		}
		if n := len(auditR3MilsciOfKind(ev, engine.EvExtraTurn)); n != 0 || s.ExtraTurn {
			t.Errorf("%d extra-turn events (pending %v), want none", n, s.ExtraTurn)
		}
		if !s.Players[0].City.Has(auditR3MilsciCard(t, "press")) {
			t.Errorf("the opponent's grey card was destroyed after the game ended")
		}
		auditR3MilsciEndsWith(t, ev, engine.VictoryMilitary, 1)
	})
}

// TestAuditR3MilsciColossusThenStrategyInTheExtraTurnTakesBothTokens: a
// player with Strategy and Theology builds the Colossus from space 2: 2
// shields, no Strategy bonus, so the pawn lands on 4 and the 2-coin token
// takes 2 of the victim's 4 coins. In the extra turn, Walls with Strategy
// (3 shields) take the pawn from 4 to 7, past the 5-coin token, which takes
// the 2 coins left. The 2-coin token does not fire again.
// Rule: docs/rules/wonders-tokens.md "Strategy" (wonders get no extra
// shield) and "Theology"; docs/rules/military-science.md "Looting tokens"
// (only as much as the victim has; each token once).
func TestAuditR3MilsciColossusThenStrategyInTheExtraTurnTakesBothTokens(t *testing.T) {
	s := enginetest.New(t, 3009).Age(2).
		Wonders(0, "colossus").
		City(0, "clay-pool", "brickyard", "glassworks", "shelf-quarry").
		Tokens(0, "strategy", "theology").
		Coins(1, 4).Pawn(2).
		Accessible("temple", "walls").ToMove(0).State()

	ev := enginetest.Do(t, &s, "wonder:colossus:temple")
	if s.Military != 4 {
		t.Fatalf("pawn %d after the Colossus, want 4 (Strategy adds nothing to a wonder)", s.Military)
	}
	if loots := auditR3MilsciOfKind(ev, engine.EvLoot); !slices.Equal(loots, []engine.Event{{Kind: engine.EvLoot, Seat: 1, A: 2, B: 2}}) {
		t.Fatalf("loot events %+v after the Colossus", loots)
	}
	if s.Phase != engine.PhasePlay || s.ToMove != 0 {
		t.Fatalf("phase %s, seat %d to move; want seat 0's extra turn", s.Phase, s.ToMove)
	}

	ev = enginetest.Do(t, &s, "construct:walls")
	if s.Military != 7 {
		t.Fatalf("pawn %d after Walls, want 7", s.Military)
	}
	if loots := auditR3MilsciOfKind(ev, engine.EvLoot); !slices.Equal(loots, []engine.Event{{Kind: engine.EvLoot, Seat: 1, A: 2, B: 5}}) {
		t.Fatalf("loot events %+v after Walls, want only the 5-coin token taking 2", loots)
	}
	if s.Players[1].Coins != 0 || s.ToMove != 1 {
		t.Fatalf("seat 1 coins %d, seat %d to move; want 0 and seat 1", s.Players[1].Coins, s.ToMove)
	}
}

// TestAuditR3MilsciEconomyIncomeThenBothTokensFromAThinPurse: seat 1 owns
// Economy and holds 1 coin. Seat 0 buys one clay (2 coins, paid to seat 1,
// who then has 3) and builds Arsenal with Strategy from space 2 to space 6.
// The 2-coin token takes 2, and the 5-coin token takes the 1 that is left.
// Rule: docs/rules/decisions.md "Effect order" (the Economy income is paid
// before the shields and their looting); docs/rules/military-science.md
// "Looting tokens" (both tokens in one move, nearer first; never more than
// the victim has).
func TestAuditR3MilsciEconomyIncomeThenBothTokensFromAThinPurse(t *testing.T) {
	s := enginetest.New(t, 3010).Age(3).
		City(0, "lumber-yard", "logging-camp", "clay-pool", "clay-pit").
		Tokens(0, "strategy").Tokens(1, "economy").
		Coins(0, 7).Coins(1, 1).Pawn(2).
		Accessible("arsenal").ToMove(0).State()

	ev := enginetest.Do(t, &s, "construct:arsenal")

	if pays := auditR3MilsciOfKind(ev, engine.EvPay); !slices.Equal(pays, []engine.Event{{Kind: engine.EvPay, Seat: 0, A: 2, B: 2}}) {
		t.Fatalf("pay events %+v, want 2 coins, all to seat 1", pays)
	}
	if s.Military != 6 {
		t.Fatalf("pawn %d, want 6", s.Military)
	}
	want := []engine.Event{
		{Kind: engine.EvLoot, Seat: 1, A: 2, B: 2},
		{Kind: engine.EvLoot, Seat: 1, A: 1, B: 5},
	}
	if loots := auditR3MilsciOfKind(ev, engine.EvLoot); !slices.Equal(loots, want) {
		t.Fatalf("loot events %+v, want %+v", loots, want)
	}
	if s.Players[0].Coins != 5 || s.Players[1].Coins != 0 {
		t.Fatalf("coins %d / %d, want 5 / 0", s.Players[0].Coins, s.Players[1].Coins)
	}
}

// TestAuditR3MilsciLawReturnedByTheLibraryCannotBePickedAfterAPair: the
// Great Library offers Law, Philosophy and Urbanism; the builder keeps
// Philosophy. Law and Urbanism go back to the box, not to the board, so a
// pair the same player completes later can only take a board token.
// Rule: docs/rules/wonders-tokens.md "The Great Library" (the other two go
// back to the box); docs/rules/military-science.md "Pairs" and "For
// developers" (during a board pick, the legal keys are the tokens still on
// the board).
func TestAuditR3MilsciLawReturnedByTheLibraryCannotBePickedAfterAPair(t *testing.T) {
	board := []string{"agriculture", "economy", "masonry", "mathematics", "strategy"}
	s := enginetest.New(t, 3011).Age(2).
		Wonders(0, "great-library").
		City(0, "lumber-yard", "sawmill", "glassworks", "press", "pharmacist").
		BoardTokens(board...).
		BoxOrder("law", "philosophy", "urbanism").
		Accessible("temple", "dispensary").ToMove(0).State()

	enginetest.Do(t, &s, "wonder:great-library:temple")
	if got, want := auditR3MilsciSortedKeys(&s), []string{"token:law", "token:philosophy", "token:urbanism"}; !slices.Equal(got, want) {
		t.Fatalf("library offer keys %v, want %v", got, want)
	}
	enginetest.Do(t, &s, "token:philosophy")
	auditR3MilsciDiscardOther(t, &s, "dispensary")
	enginetest.Do(t, &s, "construct:dispensary") // free through Pharmacist; mortar pair

	if s.Phase != engine.PhasePickBoardToken {
		t.Fatalf("phase %s, want pickBoardToken", s.Phase)
	}
	var want []string
	for _, slug := range board {
		want = append(want, "token:"+slug)
	}
	slices.Sort(want)
	if got := auditR3MilsciSortedKeys(&s); !slices.Equal(got, want) {
		t.Fatalf("pair keys %v, want the board tokens %v", got, want)
	}
	enginetest.MustFail(t, &s, "token:law")
	enginetest.MustFail(t, &s, "token:urbanism")
	enginetest.MustFail(t, &s, "token:philosophy")
	var box []catalog.TokenID
	for _, tk := range s.BoxTokens {
		if tk != 0 {
			box = append(box, tk)
		}
	}
	for _, slug := range []string{"law", "urbanism"} {
		if !slices.Contains(box, auditR3MilsciToken(t, slug)) {
			t.Errorf("box %v lacks %s", box, slug)
		}
	}
	// Five tokens were in the box; three were drawn and one kept.
	if len(box) != 4 {
		t.Errorf("box %v holds %d tokens, want 4", box, len(box))
	}
	if v := engine.ViewFor(&s, engine.SeatViewer(1)); v.BoxTokenCount != 4 {
		t.Errorf("view box count %d, want 4", v.BoxTokenCount)
	}
}

// TestAuditR3MilsciTwinsSplitBetweenCitiesNeverPair: seat 0 owns Pharmacist
// and discards Dispensary, which gives it no symbol. Seat 1 then rebuilds
// Dispensary with the Mausoleum: seat 1 owns one mortar, seat 0 owns one,
// and neither has a pair, so no token pick opens.
// Rule: docs/rules/military-science.md "Pairs" ("If each player owns one
// card of a symbol, neither will ever have that pair"); docs/rules/
// wonders-tokens.md "The Mausoleum" (the pool is every discarded card,
// from either player).
func TestAuditR3MilsciTwinsSplitBetweenCitiesNeverPair(t *testing.T) {
	s := enginetest.New(t, 3012).Age(2).
		City(0, "pharmacist").
		Wonders(1, "mausoleum").
		City(1, "clay-pool", "clay-pit", "glassworks", "glass-blower", "press").
		BoardTokens("agriculture", "philosophy", "law", "strategy", "theology").
		Accessible("dispensary", "temple").ToMove(0).State()

	enginetest.Do(t, &s, "discard:dispensary")
	if n := engine.DistinctSymbols(&s.Players[0]); n != 1 {
		t.Fatalf("seat 0 has %d symbols after discarding, want 1", n)
	}
	ev := enginetest.Do(t, &s, "wonder:mausoleum:temple", "revive:dispensary")

	if len(auditR3MilsciOfKind(ev, engine.EvPair)) != 0 {
		t.Fatalf("a pair was reported: %+v", ev)
	}
	if s.Phase != engine.PhasePlay || s.ToMove != 0 {
		t.Fatalf("phase %s, seat %d to move; want seat 0's turn", s.Phase, s.ToMove)
	}
	if !s.Players[1].City.Has(auditR3MilsciCard(t, "dispensary")) {
		t.Fatalf("Dispensary is not in seat 1's city")
	}
	if a, b := engine.DistinctSymbols(&s.Players[0]), engine.DistinctSymbols(&s.Players[1]); a != 1 || b != 1 {
		t.Fatalf("distinct symbols %d / %d, want 1 / 1", a, b)
	}
	if s.BoardTokens.Count() != 5 || s.Players[1].Tokens.Count() != 0 {
		t.Fatalf("board %v, seat 1 tokens %v: no token should move", s.BoardTokens, s.Players[1].Tokens)
	}
}

// TestAuditR3MilsciViewScienceShowsTheScalesAndThePair: after Dispensary
// completes the mortar pair and Law is taken, the public science panel of
// the player lists the scales once, the mortar twice and the set square
// once: three different symbols.
// Rule: docs/rules/military-science.md "Science symbols" (Law carries the
// scales, a seventh symbol), "Pairs" and "Scientific supremacy" (the scales
// count as one of the seven).
func TestAuditR3MilsciViewScienceShowsTheScalesAndThePair(t *testing.T) {
	s := enginetest.New(t, 3013).Age(2).
		City(0, "pharmacist", "workshop").
		BoardTokens("law", "philosophy", "agriculture", "strategy", "theology").
		Accessible("dispensary").ToMove(0).State()

	enginetest.Do(t, &s, "construct:dispensary", "token:law")

	v := engine.ViewFor(&s, engine.SeatViewer(1))
	sv := v.Players[0].Science
	want := []view.SymbolCount{
		{Symbol: view.ScienceSymbolScales, Count: 1},
		{Symbol: view.ScienceSymbolMortar, Count: 2},
		{Symbol: view.ScienceSymbolSetSquare, Count: 1},
	}
	got := slices.Clone(sv.Symbols)
	slices.SortFunc(got, func(a, b view.SymbolCount) int { return strings.Compare(string(a.Symbol), string(b.Symbol)) })
	slices.SortFunc(want, func(a, b view.SymbolCount) int { return strings.Compare(string(a.Symbol), string(b.Symbol)) })
	if !slices.Equal(got, want) || sv.Distinct != 3 {
		t.Fatalf("seat 0's science %+v, want %v and 3 different symbols", sv, want)
	}
	if !slices.Contains(v.Players[0].Tokens, "law") || slices.Contains(v.BoardTokens, "law") {
		t.Fatalf("tokens %v, board %v: Law should have moved to seat 0", v.Players[0].Tokens, v.BoardTokens)
	}
}

// TestAuditR3MilsciLastCardStrategyCrossesTheCentreForTheFinalPoints: seat 1
// owns Strategy and takes the last card of Age III, Fortifications (2
// shields, 3 with Strategy), with the pawn on space 2 of seat 1's side.
// The pawn crosses the centre to space 1 of seat 0's side: seat 1 now
// scores 2 military points and seat 0 none. No looting token is reached.
// Rule: docs/rules/ages-scoring.md "The eight scoring categories" (8.
// military: only the player ahead scores, by zone); docs/rules/
// military-science.md "Military points at the end" and "Looting tokens".
func TestAuditR3MilsciLastCardStrategyCrossesTheCentreForTheFinalPoints(t *testing.T) {
	s := enginetest.New(t, 3014).Age(3).
		City(1, "clay-pool", "shelf-quarry", "press").
		Tokens(1, "strategy").
		Coins(0, 7).Pawn(2).
		OnlyLeft("fortifications").ToMove(1).State()

	ev := enginetest.Do(t, &s, "construct:fortifications")

	if !s.Over() || s.Result.Kind != engine.VictoryCivilian {
		t.Fatalf("phase %s, result %+v; want a civilian end", s.Phase, s.Result)
	}
	if s.Military != -1 {
		t.Fatalf("pawn %d, want -1", s.Military)
	}
	if sc := s.Result.Scores; sc[0].Military != 0 || sc[1].Military != 2 {
		t.Errorf("military points %d / %d, want 0 / 2", sc[0].Military, sc[1].Military)
	}
	if loots := auditR3MilsciOfKind(ev, engine.EvLoot); len(loots) != 0 || s.Players[0].Coins != 7 {
		t.Errorf("loot events %+v, seat 0 coins %d; want no looting", loots, s.Players[0].Coins)
	}
}

// TestAuditR3MilsciRevivedRedCardOnTheLastCardDecidesTheStarterChooser: the
// last card of Age I goes under the Mausoleum; the revived Garrison, with
// Strategy, moves the pawn from space 1 on seat 0's side to space 1 on seat
// 1's side. The age ends after the revive, so the weaker player who
// chooses the Age II starter is read from the pawn after it: seat 1.
// Rule: docs/rules/ages-scoring.md "The end of an age" (the age ends when
// the last turn has finished, including its pending choices) and "Who
// starts the next age" (the weaker player chooses); docs/rules/
// military-science.md "Where shields come from" (Strategy on a rebuilt red
// building).
func TestAuditR3MilsciRevivedRedCardOnTheLastCardDecidesTheStarterChooser(t *testing.T) {
	s := enginetest.New(t, 3015).Age(1).
		Wonders(0, "mausoleum").
		City(0, "clay-pool", "clay-pit", "glassworks", "glass-blower", "press").
		Tokens(0, "strategy").
		Discard("garrison").Pawn(-1).
		OnlyLeft("altar").ToMove(0).State()

	enginetest.Do(t, &s, "wonder:mausoleum:altar")
	if s.Phase != engine.PhaseBuildFromDiscard || s.Age != 1 {
		t.Fatalf("phase %s, age %d; want the Mausoleum's choice still in Age I", s.Phase, s.Age)
	}
	enginetest.Do(t, &s, "revive:garrison")

	if s.Military != 1 {
		t.Fatalf("pawn %d, want 1", s.Military)
	}
	if s.Age != 2 || s.Phase != engine.PhaseChooseStarter || s.ToMove != 1 {
		t.Fatalf("age %d, phase %s, seat %d to move; want Age II's starter chosen by seat 1", s.Age, s.Phase, s.ToMove)
	}
}

// TestAuditR3MilsciSixDifferentSymbolsWithAPairAmongSevenCards: a player
// whose city holds a quill pair and four other symbols (six green cards,
// five different symbols) builds University, a sixth different symbol: the
// seventh green card wins at once.
// Rule: docs/rules/military-science.md "Scientific supremacy" (six
// different symbols, whatever else the city holds).
func TestAuditR3MilsciSixDifferentSymbolsWithAPairAmongSevenCards(t *testing.T) {
	s := enginetest.New(t, 3016).Age(3).
		City(0, "scriptorium", "library", "workshop", "pharmacist", "apothecary", "study",
			"clay-pool", "glassworks", "press").
		Accessible("university").ToMove(0).State()
	if n := engine.DistinctSymbols(&s.Players[0]); n != 5 {
		t.Fatalf("setup: %d distinct symbols, want 5", n)
	}

	ev := enginetest.Do(t, &s, "construct:university")

	if !s.Over() || s.Result.Kind != engine.VictoryScience || s.Result.Winner != 0 {
		t.Fatalf("phase %s, result %+v; want seat 0's science victory", s.Phase, s.Result)
	}
	auditR3MilsciEndsWith(t, ev, engine.VictoryScience, 0)
}

// TestAuditR3MilsciTokenPointsAtTheEndAfterALibraryPick: on the last card
// of Age III a player who already owns Agriculture builds the Great Library
// and keeps Mathematics. Scoring counts Agriculture 4 and Mathematics 3 for
// each of the two tokens; Law, owned too, scores nothing as a token.
// Rule: docs/rules/ages-scoring.md "The eight scoring categories" (6.
// progress tokens: Agriculture 4, Philosophy 7, Mathematics 3 per token
// owned including itself; the other tokens are worth no points).
func TestAuditR3MilsciTokenPointsAtTheEndAfterALibraryPick(t *testing.T) {
	s := enginetest.New(t, 3017).Age(3).
		Wonders(0, "great-library").
		City(0, "lumber-yard", "sawmill", "glassworks", "press").
		Tokens(0, "agriculture", "law").
		BoardTokens("architecture", "economy", "masonry", "strategy", "theology").
		BoxOrder("mathematics", "philosophy", "urbanism").
		OnlyLeft("obelisk").ToMove(0).State()

	enginetest.Do(t, &s, "wonder:great-library:obelisk")
	if s.Over() || s.Phase != engine.PhasePickLibraryToken {
		t.Fatalf("phase %s, result %+v; want the library pick before scoring", s.Phase, s.Result)
	}
	enginetest.Do(t, &s, "token:mathematics")

	if !s.Over() || s.Result.Kind != engine.VictoryCivilian {
		t.Fatalf("phase %s, result %+v; want a civilian end", s.Phase, s.Result)
	}
	if got, want := s.Result.Scores[0].Tokens, int16(4+3*3); got != want {
		t.Errorf("token points %d, want %d (Agriculture 4, Mathematics 3 x 3 tokens, Law 0)", got, want)
	}
}
