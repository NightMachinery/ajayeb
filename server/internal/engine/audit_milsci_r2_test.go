package engine_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

// This file is the M10.R1 rules audit, round 1, through the military and
// science lens. It is black-box: it builds positions with the scenario
// builder and checks the engine against the rules in docs/rules/ and the
// rulebook. It aims at interactions between two or three effects, at
// boundary values (an empty purse, the last token, the ninth shield, the
// last card of an age) and at the order of resolution inside one action. It
// does not repeat the cases of edgecases_milsci_r1_test.go.

// auditR2MilsciCard returns the id of a card slug.
func auditR2MilsciCard(t *testing.T, slug string) catalog.CardID {
	t.Helper()
	c, ok := catalog.CardBySlug(slug)
	if !ok {
		t.Fatalf("unknown card %q", slug)
	}
	return c
}

// auditR2MilsciToken returns the id of a token slug.
func auditR2MilsciToken(t *testing.T, slug string) catalog.TokenID {
	t.Helper()
	tk, ok := catalog.TokenBySlug(slug)
	if !ok {
		t.Fatalf("unknown token %q", slug)
	}
	return tk
}

// auditR2MilsciOfKind returns the events of one kind, in order.
func auditR2MilsciOfKind(ev []engine.Event, k engine.EventKind) []engine.Event {
	var out []engine.Event
	for _, e := range ev {
		if e.Kind == k {
			out = append(out, e)
		}
	}
	return out
}

// auditR2MilsciIndex returns the index of the first event of kind k, or -1.
func auditR2MilsciIndex(ev []engine.Event, k engine.EventKind) int {
	for i, e := range ev {
		if e.Kind == k {
			return i
		}
	}
	return -1
}

// auditR2MilsciDiscardOther discards any accessible card other than the
// named ones, for the opponent's filler move between two turns of a test.
func auditR2MilsciDiscardOther(t *testing.T, s *engine.State, keep ...string) {
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

// TestAuditR2MilsciLootingAnEmptyPurseStillSpendsTheToken: when the pawn
// enters space 3 while the victim has 0 coins, the victim loses nothing, but
// the token still leaves the game, so re-entering space 3 later (when the
// victim has coins again) takes nothing.
// Rule: docs/rules/military-science.md "Looting tokens": the token "leaves
// the game for good", "a player who has fewer coins loses all they have and
// no more", and "a token triggers only once"; rulebook p.12 (looting tokens
// are discarded after use).
func TestAuditR2MilsciLootingAnEmptyPurseStillSpendsTheToken(t *testing.T) {
	s := enginetest.New(t, 2001).Age(1).
		City(0, "lumber-yard", "clay-pool").
		Coins(0, 7).Coins(1, 0).
		Pawn(2).
		Accessible("stable", "baths", "theater", "guard-tower", "garrison").
		ToMove(0).State()

	enginetest.Do(t, &s, "construct:stable") // seat 0: pawn 2 -> 3, seat 1 has nothing to lose
	if s.Military != 3 || s.Players[1].Coins != 0 {
		t.Fatalf("after Stable: pawn %d, seat 1 coins %d; want 3 and 0", s.Military, s.Players[1].Coins)
	}
	if s.Looting&engine.LootPlus3 != 0 {
		t.Fatalf("the 2-coin token on seat 1's side is still on the track after the pawn reached space 3 (looting %04b)", s.Looting)
	}
	enginetest.Do(t, &s, "discard:baths") // seat 1 gains 2 coins
	enginetest.Do(t, &s, "discard:theater")
	enginetest.Do(t, &s, "construct:guard-tower") // seat 1: pawn 3 -> 2
	coins := s.Players[1].Coins
	if s.Military != 2 || coins == 0 {
		t.Fatalf("setup: pawn %d, seat 1 coins %d", s.Military, coins)
	}
	ev := enginetest.Do(t, &s, "construct:garrison") // seat 0: pawn 2 -> 3 again
	if s.Military != 3 {
		t.Fatalf("pawn %d, want 3", s.Military)
	}
	if loots := auditR2MilsciOfKind(ev, engine.EvLoot); len(loots) != 0 || s.Players[1].Coins != coins {
		t.Fatalf("re-entering space 3 looted seat 1 (coins %d -> %d, loot events %+v); the token was spent against an empty purse",
			coins, s.Players[1].Coins, loots)
	}
}

// TestAuditR2MilsciSeatOneTakesBothTokensFromAPartialPurse: seat 1 (pushing
// toward seat 0's capital) crosses both looting tokens of seat 0's side in
// one move, nearer token first. Seat 0 holds 6 coins: the 2-coin token takes
// 2, the 5-coin token takes the 4 that are left.
// Rule: docs/rules/military-science.md "Looting tokens" (one move can
// trigger both tokens, one after the other; a player with fewer coins loses
// all they have and no more) and "Where shields come from" (Strategy).
func TestAuditR2MilsciSeatOneTakesBothTokensFromAPartialPurse(t *testing.T) {
	s := enginetest.New(t, 2002).Age(3).
		City(1, "clay-pool", "clay-pit", "brickyard", "lumber-yard", "logging-camp").
		Tokens(1, "strategy").
		Pawn(-2).Coins(0, 6).
		Accessible("arsenal").ToMove(1).State()

	ev := enginetest.Do(t, &s, "construct:arsenal") // 3 shields + 1 from Strategy: -2 -> -6

	if s.Military != -6 {
		t.Fatalf("pawn %d, want -6", s.Military)
	}
	loots := auditR2MilsciOfKind(ev, engine.EvLoot)
	want := []engine.Event{
		{Kind: engine.EvLoot, Seat: 0, A: 2, B: 2},
		{Kind: engine.EvLoot, Seat: 0, A: 4, B: 5},
	}
	if !slices.Equal(loots, want) {
		t.Fatalf("loot events %+v, want %+v", loots, want)
	}
	if s.Players[0].Coins != 0 {
		t.Fatalf("seat 0 coins %d, want 0", s.Players[0].Coins)
	}
	if s.Looting&(engine.LootMinus3|engine.LootMinus6) != 0 || s.Looting&(engine.LootPlus3|engine.LootPlus6) != engine.LootPlus3|engine.LootPlus6 {
		t.Fatalf("looting bits %04b: seat 0's two tokens should be gone and seat 1's two still there", s.Looting)
	}
}

// TestAuditR2MilsciCrossingTheCentreLootsOnlyTheOpponent: a move that starts
// on the mover's own side and ends in the opponent's 5-point zone takes the
// opponent's 2-coin token, and never a token of the mover's own side, which
// the pawn only moves away from.
// Rule: docs/rules/military-science.md "Looting tokens" (a token triggers
// when the pawn enters space 3 or 6 on a player's side, and that player
// loses the coins).
func TestAuditR2MilsciCrossingTheCentreLootsOnlyTheOpponent(t *testing.T) {
	s := enginetest.New(t, 2003).Age(3).
		City(0, "clay-pool", "clay-pit", "brickyard", "lumber-yard", "logging-camp").
		Tokens(0, "strategy").
		Pawn(-1).Coins(0, 7).Coins(1, 7).
		Accessible("arsenal").ToMove(0).State()

	ev := enginetest.Do(t, &s, "construct:arsenal") // 4 shields: -1 -> 3

	if s.Military != 3 {
		t.Fatalf("pawn %d, want 3", s.Military)
	}
	loots := auditR2MilsciOfKind(ev, engine.EvLoot)
	if len(loots) != 1 || loots[0].Seat != 1 || loots[0].A != 2 {
		t.Fatalf("loot events %+v, want one 2-coin loss for seat 1", loots)
	}
	if s.Players[0].Coins != 7 || s.Players[1].Coins != 5 {
		t.Fatalf("coins %d / %d, want 7 / 5", s.Players[0].Coins, s.Players[1].Coins)
	}
	if s.Looting&(engine.LootMinus3|engine.LootMinus6) != engine.LootMinus3|engine.LootMinus6 {
		t.Fatalf("looting bits %04b: seat 0's own tokens must stay on the track", s.Looting)
	}
}

// TestAuditR2MilsciStrategySuppliesTheNinthShield: the pawn starts on space
// 7. Stable's own shield takes it to 8, and Strategy's extra shield is the
// one that reaches the capital: military supremacy, at once.
// Rule: docs/rules/military-science.md "Where shields come from" (Strategy
// gives 1 extra shield to each red building) and "Military supremacy".
func TestAuditR2MilsciStrategySuppliesTheNinthShield(t *testing.T) {
	s := enginetest.New(t, 2004).Age(1).
		City(0, "lumber-yard").Tokens(0, "strategy").
		Pawn(7).Accessible("stable").ToMove(0).State()

	enginetest.Do(t, &s, "construct:stable")

	if s.Military != 9 || !s.Over() || s.Result.Kind != engine.VictoryMilitary || s.Result.Winner != 0 {
		t.Fatalf("pawn %d, phase %s, result %+v; want pawn 9 and seat 0's military victory", s.Military, s.Phase, s.Result)
	}
}

// TestAuditR2MilsciColossusWithStrategyStopsOneShortOfTheCapital: from space
// 6, the Colossus's 2 shields take the pawn to space 8. Strategy adds nothing
// to a wonder, so the game must go on; a wrong extra shield would end it.
// Rule: docs/rules/wonders-tokens.md "Strategy" ("Wonders get no extra
// shield"); rules-reference.md section 8 item 4.
func TestAuditR2MilsciColossusWithStrategyStopsOneShortOfTheCapital(t *testing.T) {
	s := enginetest.New(t, 2005).Age(2).
		Wonders(0, "colossus").
		City(0, "clay-pool", "brickyard", "glassworks").
		Tokens(0, "strategy").
		Pawn(6).Accessible("temple").ToMove(0).State()

	enginetest.Do(t, &s, "wonder:colossus:temple")

	if s.Over() || s.Military != 8 {
		t.Fatalf("pawn %d, phase %s, result %+v; want pawn 8 and the game going on", s.Military, s.Phase, s.Result)
	}
	if s.ToMove != 1 {
		t.Fatalf("seat %d to move, want 1", s.ToMove)
	}
}

// TestAuditR2MilsciEconomyIncomeBeforeTheColossusLoots: a wonder's payment
// (with the trading coins going to an opponent who owns Economy) comes
// before its shields. The opponent starts with 0 coins, receives 2 trading
// coins, and then the Colossus's shields reach space 3 and the 2-coin token
// takes them back. The other order would leave the opponent with 2.
// Rule: docs/rules/wonders-tokens.md "The order of a wonder's effects"
// (step 1 the payment and the Economy income, step 4 the shields and
// looting); docs/rules/decisions.md "Effect order".
func TestAuditR2MilsciEconomyIncomeBeforeTheColossusLoots(t *testing.T) {
	s := enginetest.New(t, 2006).Age(2).
		Wonders(0, "colossus").
		City(0, "clay-pool", "brickyard"). // 3 clay; the glass is bought for 2
		Tokens(1, "economy").
		Coins(0, 5).Coins(1, 0).
		Pawn(2).Accessible("temple").ToMove(0).State()

	enginetest.Do(t, &s, "wonder:colossus:temple")

	if s.Military != 4 {
		t.Fatalf("pawn %d, want 4", s.Military)
	}
	if s.Players[0].Coins != 3 {
		t.Fatalf("seat 0 coins %d, want 3 (5 - 2 for the glass)", s.Players[0].Coins)
	}
	if s.Players[1].Coins != 0 {
		t.Fatalf("seat 1 coins %d, want 0 (2 from Economy, then 2 lost to looting)", s.Players[1].Coins)
	}
}

// TestAuditR2MilsciZeusLootsThenDestroysThenTheologyPlaysAgain: the Statue
// of Zeus, built by a Theology owner from space 2, does three things in
// order: its shield reaches space 3 and loots 2 coins, the extra turn is
// noted, and the destroy choice opens. Once the card is destroyed, the
// builder plays again.
// Rule: docs/rules/wonders-tokens.md "The order of a wonder's effects"
// (shields 4, extra turn 5, special choice 7) and "Theology".
func TestAuditR2MilsciZeusLootsThenDestroysThenTheologyPlaysAgain(t *testing.T) {
	s := enginetest.New(t, 2007).Age(1).
		Wonders(0, "statue-of-zeus").
		City(0, "press", "drying-room", "clay-pool", "lumber-yard", "quarry").
		Tokens(0, "theology").
		City(1, "logging-camp").Coins(1, 5).
		Pawn(2).Accessible("stone-pit").ToMove(0).State()

	ev := enginetest.Do(t, &s, "wonder:statue-of-zeus:stone-pit")

	if s.Military != 3 || s.Players[1].Coins != 3 {
		t.Fatalf("pawn %d, seat 1 coins %d; want 3 and 3", s.Military, s.Players[1].Coins)
	}
	iLoot, iExtra := auditR2MilsciIndex(ev, engine.EvLoot), auditR2MilsciIndex(ev, engine.EvExtraTurn)
	if iLoot < 0 || iExtra < 0 || iLoot > iExtra {
		t.Fatalf("want the loot before the extra turn: events %+v", ev)
	}
	if s.Phase != engine.PhaseDestroyCard || s.ToMove != 0 {
		t.Fatalf("phase %s, seat %d to move; want destroyCard for seat 0", s.Phase, s.ToMove)
	}
	enginetest.Do(t, &s, "destroy:logging-camp")
	if s.Phase != engine.PhasePlay || s.ToMove != 0 || s.Active != 0 {
		t.Fatalf("after the destroy: phase %s, to move %d, active %d; want seat 0 to play again", s.Phase, s.ToMove, s.Active)
	}
}

// TestAuditR2MilsciUrbanismCoinsThenStrategyShieldsOnAChainedRedCard: Horse
// Breeders built free through Stable's chain by a player who owns Urbanism
// and Strategy: Urbanism pays 4 coins, then the card's shield plus
// Strategy's shield move the pawn 2 spaces, from 1 to 3, which loots 2 coins
// from the opponent. The coins come before the shields.
// Rule: docs/rules/decisions.md "Effect order" (a building: coin rewards,
// first Urbanism's chain bonus, then shields with looting);
// docs/rules/wonders-tokens.md "Urbanism" and "Strategy".
func TestAuditR2MilsciUrbanismCoinsThenStrategyShieldsOnAChainedRedCard(t *testing.T) {
	s := enginetest.New(t, 2008).Age(2).
		City(0, "stable").Tokens(0, "urbanism", "strategy").
		Coins(0, 0).Coins(1, 7).
		Pawn(1).Accessible("horse-breeders").ToMove(0).State()

	ev := enginetest.Do(t, &s, "construct:horse-breeders")

	if s.Players[0].Coins != 4 {
		t.Fatalf("seat 0 coins %d, want 4 (Urbanism's chain bonus, nothing paid)", s.Players[0].Coins)
	}
	if s.Military != 3 || s.Players[1].Coins != 5 {
		t.Fatalf("pawn %d, seat 1 coins %d; want 3 and 5", s.Military, s.Players[1].Coins)
	}
	iCoins, iPawn := auditR2MilsciIndex(ev, engine.EvCoins), auditR2MilsciIndex(ev, engine.EvPawn)
	if iCoins < 0 || iPawn < 0 || iCoins > iPawn {
		t.Fatalf("want Urbanism's coins before the pawn moves: events %+v", ev)
	}
}

// TestAuditR2MilsciTheLastBoardTokenThenAPairGivesNothing: the board holds a
// single token. The first pair must take it (and only it: neither a token in
// the box nor one the opponent owns can be taken). The next pair finds the
// board empty and gives nothing, and the turn passes normally.
// Rule: docs/rules/military-science.md "Pairs" ("If no tokens are left on
// the board, the pair gives nothing"; the pick is forced when a token is
// available) and "For developers" (the legal keys are the tokens still on
// the board).
func TestAuditR2MilsciTheLastBoardTokenThenAPairGivesNothing(t *testing.T) {
	s := enginetest.New(t, 2009).Age(2).
		City(0, "pharmacist", "apothecary", "lumber-yard", "press", "drying-room").
		Tokens(1, "mathematics", "strategy", "theology", "urbanism").
		BoardTokens("philosophy").
		Accessible("dispensary", "school").ToMove(0).State()

	enginetest.Do(t, &s, "construct:dispensary") // free through Pharmacist's chain; mortar pair
	if s.Phase != engine.PhasePickBoardToken {
		t.Fatalf("phase %s, want pickBoardToken", s.Phase)
	}
	if keys := enginetest.Keys(&s); !slices.Equal(keys, []string{"token:philosophy"}) {
		t.Fatalf("legal keys %v, want only token:philosophy", keys)
	}
	enginetest.MustFail(t, &s, "token:mathematics") // the opponent's
	if box := s.BoxTokens[0]; box != 0 {
		enginetest.MustFail(t, &s, "token:"+catalog.Token(box).Slug) // in the box
	}
	enginetest.Do(t, &s, "token:philosophy")
	if !s.BoardTokens.Empty() {
		t.Fatalf("board %v, want empty", s.BoardTokens)
	}

	auditR2MilsciDiscardOther(t, &s, "school")
	ev := enginetest.Do(t, &s, "construct:school") // wheel pair, no token left
	if len(auditR2MilsciOfKind(ev, engine.EvPair)) != 1 {
		t.Fatalf("want one pair event: %+v", ev)
	}
	skipped := auditR2MilsciOfKind(ev, engine.EvEffectSkipped)
	if len(skipped) != 1 || engine.Phase(skipped[0].A) != engine.PhasePickBoardToken {
		t.Fatalf("want the token pick reported as skipped: %+v", ev)
	}
	if s.Phase != engine.PhasePlay || s.ToMove != 1 {
		t.Fatalf("phase %s, seat %d to move; want play and seat 1", s.Phase, s.ToMove)
	}
	if s.Players[0].Tokens.Count() != 1 {
		t.Fatalf("seat 0 owns %d tokens, want 1", s.Players[0].Tokens.Count())
	}
}

// TestAuditR2MilsciGreatLibraryWithTheologyPicksThenPlaysAgain: a Theology
// owner builds the Great Library. The offer holds exactly the first three
// box tokens (a board token or the fourth box token cannot be taken), the
// kept Agriculture pays its 6 coins, the other two go back to the box, and
// only then does the builder take the extra turn.
// Rule: docs/rules/wonders-tokens.md "The Great Library" (keep one of three
// drawn from the box, return the other two; the kept token takes effect at
// once), "Theology", and "The order of a wonder's effects" (the extra turn
// is noted before the special choice opens).
func TestAuditR2MilsciGreatLibraryWithTheologyPicksThenPlaysAgain(t *testing.T) {
	s := enginetest.New(t, 2010).Age(1).
		Wonders(0, "great-library").
		City(0, "lumber-yard", "sawmill", "glassworks", "press").
		Tokens(0, "theology").Coins(0, 7).
		BoardTokens("architecture", "economy", "masonry", "mathematics", "strategy").
		BoxOrder("agriculture", "law", "philosophy").
		Accessible("baths").ToMove(0).State()

	ev := enginetest.Do(t, &s, "wonder:great-library:baths")
	if s.Phase != engine.PhasePickLibraryToken || s.ToMove != 0 {
		t.Fatalf("phase %s, seat %d to move", s.Phase, s.ToMove)
	}
	if len(auditR2MilsciOfKind(ev, engine.EvExtraTurn)) != 1 {
		t.Fatalf("want one extra turn from Theology: %+v", ev)
	}
	enginetest.MustFail(t, &s, "token:architecture") // on the board, not in the offer
	enginetest.MustFail(t, &s, "token:urbanism")     // in the box, not drawn
	coins := s.Players[0].Coins

	enginetest.Do(t, &s, "token:agriculture")

	if s.Players[0].Coins != coins+6 {
		t.Fatalf("seat 0 coins %d, want %d (Agriculture's 6)", s.Players[0].Coins, coins+6)
	}
	var box []catalog.TokenID
	for _, tk := range s.BoxTokens {
		if tk != 0 {
			box = append(box, tk)
		}
	}
	for _, slug := range []string{"law", "philosophy", "urbanism"} {
		if !slices.Contains(box, auditR2MilsciToken(t, slug)) {
			t.Errorf("box %v lacks %s", box, slug)
		}
	}
	if len(box) != 3 || s.BoardTokens.Count() != 5 {
		t.Errorf("box %v (want 3 tokens), board %v (want unchanged)", box, s.BoardTokens)
	}
	if s.Phase != engine.PhasePlay || s.ToMove != 0 || s.Active != 0 {
		t.Fatalf("after the pick: phase %s, to move %d, active %d; want seat 0 to play again", s.Phase, s.ToMove, s.Active)
	}
}

// TestAuditR2MilsciMausoleumPairTheologyIsNotRetroactive: the Mausoleum
// rebuilds Dispensary, which completes the mortar pair, and the builder
// picks Theology from the board. Theology reaches only wonders built
// afterwards, so the Mausoleum (built earlier in this very turn) grants no
// extra turn: the turn passes.
// Rule: docs/rules/wonders-tokens.md "Progress tokens" (lasting effects
// "apply to everything the player does afterwards, never to what they did
// before"), "Theology" ("every wonder you build afterwards") and "The
// Mausoleum" (the rebuilt card's science symbol can complete a pair).
func TestAuditR2MilsciMausoleumPairTheologyIsNotRetroactive(t *testing.T) {
	s := enginetest.New(t, 2011).Age(2).
		Wonders(0, "mausoleum").
		City(0, "pharmacist", "clay-pool", "clay-pit", "glassworks", "glass-blower", "press").
		Discard("dispensary").
		BoardTokens("theology", "philosophy", "law", "economy", "agriculture").
		Accessible("temple").ToMove(0).State()

	ev := enginetest.Do(t, &s, "wonder:mausoleum:temple", "revive:dispensary")
	if s.Phase != engine.PhasePickBoardToken {
		t.Fatalf("phase %s, want pickBoardToken after the rebuilt pair", s.Phase)
	}
	ev = append(ev, enginetest.Do(t, &s, "token:theology")...)

	if n := len(auditR2MilsciOfKind(ev, engine.EvExtraTurn)); n != 0 {
		t.Fatalf("%d extra turns: Theology taken after the Mausoleum was built must not reach it", n)
	}
	if s.Phase != engine.PhasePlay || s.ToMove != 1 {
		t.Fatalf("phase %s, seat %d to move; want the turn to pass to seat 1", s.Phase, s.ToMove)
	}
}

// TestAuditR2MilsciMausoleumPairWithAnEmptyBoardEndsTheTurn: the rebuilt
// green card completes a pair, but no token is left on the board, so the
// pair gives nothing; the pending choice of the Mausoleum must not chain
// into an empty token choice, and the turn passes.
// Rule: docs/rules/military-science.md "Pairs" ("If no tokens are left on
// the board, the pair gives nothing"; a pair through the Mausoleum counts
// like any other); docs/rules/decisions.md "Forced choices" (a choice with
// no option is skipped).
func TestAuditR2MilsciMausoleumPairWithAnEmptyBoardEndsTheTurn(t *testing.T) {
	s := enginetest.New(t, 2012).Age(2).
		Wonders(0, "mausoleum").
		City(0, "pharmacist", "clay-pool", "clay-pit", "glassworks", "glass-blower", "press").
		Tokens(1, "mathematics", "philosophy", "strategy", "theology", "urbanism").
		BoxOrder("agriculture", "architecture", "economy", "law", "masonry").
		Discard("dispensary").
		Accessible("temple").ToMove(0).State()
	if !s.BoardTokens.Empty() {
		t.Fatalf("setup: board %v", s.BoardTokens)
	}

	ev := enginetest.Do(t, &s, "wonder:mausoleum:temple", "revive:dispensary")

	if len(auditR2MilsciOfKind(ev, engine.EvPair)) != 1 {
		t.Fatalf("want one pair event: %+v", ev)
	}
	dispensary := auditR2MilsciCard(t, "dispensary")
	skipped := auditR2MilsciOfKind(ev, engine.EvEffectSkipped)
	if len(skipped) != 1 || skipped[0].Card != dispensary || engine.Phase(skipped[0].A) != engine.PhasePickBoardToken {
		t.Fatalf("want the board pick reported as skipped for Dispensary: %+v", skipped)
	}
	if s.Phase != engine.PhasePlay || s.ToMove != 1 {
		t.Fatalf("phase %s, seat %d to move; want play and seat 1", s.Phase, s.ToMove)
	}
}

// TestAuditR2MilsciPairOnTheLastCardOfTheGameIsScored: the last card of Age
// III completes a pair. The game is not scored until the token is picked;
// the picked Agriculture then counts both its 6 coins (2 coin points) and
// its 4 points, and Mathematics counts it as a second token.
// Rule: docs/rules/ages-scoring.md "The end of the game" (the game ends when
// the last card of Age III "has been taken and that turn has finished,
// including its pending choices") and "The eight scoring categories" (6.
// progress tokens: Agriculture 4, Mathematics 3 per token including itself;
// 7. coins: 1 point per full 3).
func TestAuditR2MilsciPairOnTheLastCardOfTheGameIsScored(t *testing.T) {
	s := enginetest.New(t, 2013).Age(3).
		City(0, "study", "quarry", "lumber-yard", "glassworks", "glass-blower").
		Tokens(0, "mathematics").Coins(0, 0).
		BoardTokens("agriculture", "philosophy", "law", "strategy", "theology").
		OnlyLeft("academy").ToMove(0).State()

	enginetest.Do(t, &s, "construct:academy")
	if s.Over() || s.Phase != engine.PhasePickBoardToken {
		t.Fatalf("phase %s, result %+v; want the token pick before the game ends", s.Phase, s.Result)
	}
	enginetest.Do(t, &s, "token:agriculture")

	if !s.Over() || s.Result.Kind != engine.VictoryCivilian {
		t.Fatalf("phase %s, result %+v; want a civilian end", s.Phase, s.Result)
	}
	sc := s.Result.Scores[0]
	if sc.Tokens != 4+3*2 {
		t.Errorf("token points %d, want 10 (Agriculture 4 + Mathematics 3 x 2 tokens)", sc.Tokens)
	}
	if sc.Coins != 2 {
		t.Errorf("coin points %d, want 2 (Agriculture's 6 coins)", sc.Coins)
	}
}

// TestAuditR2MilsciLastCardShieldsSetTheFinalMilitaryPoints: the last card of
// Age III is a red building. Its shields (and the looting they cause) apply
// before the civilian scoring, so the military points come from the pawn's
// new zone and the victim's coin points from the purse after the loot.
// Rule: docs/rules/ages-scoring.md "The eight scoring categories" (8.
// military: 2, 5 or 10 by zone for the player ahead; 7. coins) and "The end
// of the game"; docs/rules/military-science.md "Looting tokens".
func TestAuditR2MilsciLastCardShieldsSetTheFinalMilitaryPoints(t *testing.T) {
	t.Run("seat 0 into the 5-point zone", func(t *testing.T) {
		s := enginetest.New(t, 2014).Age(3).
			Coins(0, 8).Coins(1, 3).Pawn(0).
			OnlyLeft("pretorium").ToMove(0).State()
		enginetest.Do(t, &s, "construct:pretorium") // 8 coins, 3 shields: 0 -> 3
		if !s.Over() || s.Result.Kind != engine.VictoryCivilian || s.Military != 3 {
			t.Fatalf("phase %s, pawn %d, result %+v", s.Phase, s.Military, s.Result)
		}
		sc := s.Result.Scores
		if sc[0].Military != 5 || sc[1].Military != 0 {
			t.Errorf("military points %d / %d, want 5 / 0", sc[0].Military, sc[1].Military)
		}
		if s.Players[1].Coins != 1 || sc[1].Coins != 0 {
			t.Errorf("seat 1 coins %d (coin points %d), want 1 (0 points) after the 2-coin loot", s.Players[1].Coins, sc[1].Coins)
		}
	})
	t.Run("seat 1 into the 10-point zone", func(t *testing.T) {
		s := enginetest.New(t, 2015).Age(3).
			City(1, "clay-pool", "clay-pit", "brickyard", "lumber-yard", "logging-camp").
			Coins(0, 5).Pawn(-4).
			OnlyLeft("arsenal").ToMove(1).State()
		enginetest.Do(t, &s, "construct:arsenal") // 3 shields: -4 -> -7, taking the 5-coin token
		if !s.Over() || s.Result.Kind != engine.VictoryCivilian || s.Military != -7 {
			t.Fatalf("phase %s, pawn %d, result %+v", s.Phase, s.Military, s.Result)
		}
		sc := s.Result.Scores
		if sc[1].Military != 10 || sc[0].Military != 0 {
			t.Errorf("military points %d / %d, want 0 / 10", sc[0].Military, sc[1].Military)
		}
		if s.Players[0].Coins != 0 || sc[0].Coins != 0 {
			t.Errorf("seat 0 coins %d (coin points %d), want 0 after the 5-coin loot", s.Players[0].Coins, sc[0].Coins)
		}
	})
}

// TestAuditR2MilsciStarterChooserUsesThePawnAfterTheLastCard: the weaker
// player who chooses the next age's starter is decided by the pawn where it
// stands once the age's last turn is over, after that turn's shields.
// Rule: docs/rules/ages-scoring.md "Who starts the next age" (the weaker
// player chooses; on the centre space the last active player chooses) and
// "The end of an age" (the age ends when its last card has been taken and
// that turn has finished).
func TestAuditR2MilsciStarterChooserUsesThePawnAfterTheLastCard(t *testing.T) {
	t.Run("crossing the centre", func(t *testing.T) {
		s := enginetest.New(t, 2016).Age(1).
			City(0, "lumber-yard").Tokens(0, "strategy").
			Pawn(-1).OnlyLeft("stable").ToMove(0).State()
		enginetest.Do(t, &s, "construct:stable") // 2 shields: -1 -> 1, seat 1 is now weaker
		if s.Phase != engine.PhaseChooseStarter || s.ToMove != 1 {
			t.Fatalf("phase %s, seat %d to move; want seat 1 (weaker after the move) to choose", s.Phase, s.ToMove)
		}
	})
	t.Run("reaching the centre", func(t *testing.T) {
		s := enginetest.New(t, 2017).Age(1).
			City(0, "lumber-yard").
			Pawn(-1).OnlyLeft("stable").ToMove(0).State()
		enginetest.Do(t, &s, "construct:stable") // 1 shield: -1 -> 0
		if s.Phase != engine.PhaseChooseStarter || s.ToMove != 0 {
			t.Fatalf("phase %s, seat %d to move; want seat 0 (last active, centred pawn) to choose", s.Phase, s.ToMove)
		}
	})
}

// TestAuditR2MilsciSixGreenCardsAreNotSixSymbols: a player who owns six green
// cards but only three different symbols (three pairs) has not won;
// supremacy needs six different symbols. The sixth green card completes a
// pair and opens the token pick instead.
// Rule: docs/rules/military-science.md "Scientific supremacy" ("six
// different science symbols"); rulebook p.12.
func TestAuditR2MilsciSixGreenCardsAreNotSixSymbols(t *testing.T) {
	s := enginetest.New(t, 2018).Age(2).
		City(0, "pharmacist", "dispensary", "workshop", "laboratory", "apothecary",
			"lumber-yard", "press", "drying-room").
		Accessible("school").ToMove(0).State()

	enginetest.Do(t, &s, "construct:school")

	if s.Over() {
		t.Fatalf("the game ended (%+v) with three different symbols", s.Result)
	}
	if n := engine.DistinctSymbols(&s.Players[0]); n != 3 {
		t.Fatalf("distinct symbols %d, want 3", n)
	}
	if s.Phase != engine.PhasePickBoardToken {
		t.Fatalf("phase %s, want pickBoardToken for the wheel pair", s.Phase)
	}
}

// TestAuditR2MilsciLawAsTheFifthSymbolThenACardAsTheSixth: Law taken after a
// pair adds the scales as a fifth different symbol, which does not win; the
// next new card symbol is the sixth and wins at once.
// Rule: docs/rules/military-science.md "Scientific supremacy" (the scales
// count as one of the seven; six different symbols win; a card that
// completes a pair repeats a symbol).
func TestAuditR2MilsciLawAsTheFifthSymbolThenACardAsTheSixth(t *testing.T) {
	s := enginetest.New(t, 2019).Age(3).
		City(0, "scriptorium", "workshop", "apothecary", "study",
			"quarry", "lumber-yard", "glassworks", "glass-blower", "clay-pool", "press").
		BoardTokens("law", "philosophy", "agriculture", "strategy", "theology").
		Accessible("academy", "university").ToMove(0).State()

	enginetest.Do(t, &s, "construct:academy", "token:law")
	if s.Over() {
		t.Fatalf("the game ended (%+v) with five different symbols", s.Result)
	}
	if n := engine.DistinctSymbols(&s.Players[0]); n != 5 {
		t.Fatalf("distinct symbols %d, want 5 (quill, set square, wheel, sundial, scales)", n)
	}
	if s.ToMove != 1 {
		t.Fatalf("seat %d to move, want 1", s.ToMove)
	}
	auditR2MilsciDiscardOther(t, &s, "university")
	enginetest.Do(t, &s, "construct:university")
	if !s.Over() || s.Result.Kind != engine.VictoryScience || s.Result.Winner != 0 {
		t.Fatalf("phase %s, result %+v; want seat 0's science victory", s.Phase, s.Result)
	}
}

// TestAuditR2MilsciLibraryLawWinsDespiteTheologysExtraTurn: a Theology owner
// with five different card symbols builds the Great Library and keeps Law.
// The extra turn was noted before the pick, but the sixth symbol wins at
// once, so the game ends and no extra turn is left pending.
// Rule: docs/rules/military-science.md "Scientific supremacy" (Law from the
// Great Library offer can be the sixth symbol); docs/rules/decisions.md
// "Supremacy timing" (a supremacy ends the game the moment it happens).
func TestAuditR2MilsciLibraryLawWinsDespiteTheologysExtraTurn(t *testing.T) {
	s := enginetest.New(t, 2020).Age(3).
		Wonders(0, "great-library").
		City(0, "scriptorium", "workshop", "apothecary", "pharmacist", "academy",
			"lumber-yard", "sawmill", "glassworks", "press").
		Tokens(0, "theology").
		BoardTokens("architecture", "economy", "masonry", "mathematics", "strategy").
		BoxOrder("philosophy", "law", "agriculture").
		Accessible("obelisk").ToMove(0).State()

	enginetest.Do(t, &s, "wonder:great-library:obelisk", "token:law")

	if !s.Over() || s.Result.Kind != engine.VictoryScience || s.Result.Winner != 0 {
		t.Fatalf("phase %s, result %+v; want seat 0's science victory", s.Phase, s.Result)
	}
	if s.ExtraTurn {
		t.Fatalf("an extra turn is still pending after the game ended")
	}
}

// TestAuditR2MilsciPairDuringAnExtraTurnThenTheTurnPasses: the Sphinx grants
// an extra turn; in it the builder completes a pair and picks a token. The
// pick finishes that extra turn, and play passes to the opponent: the extra
// turn is used once, not carried over the token choice.
// Rule: docs/rules/turn.md "The end of a turn" (a pending choice belongs to
// the turn that opened it); docs/rules/wonders-tokens.md "The Sphinx" (take
// an extra turn); docs/rules/military-science.md "Pairs".
func TestAuditR2MilsciPairDuringAnExtraTurnThenTheTurnPasses(t *testing.T) {
	s := enginetest.New(t, 2021).Age(2).
		Wonders(0, "sphinx").
		City(0, "workshop", "quarry", "clay-pool", "glassworks", "glass-blower", "lumber-yard").
		BoardTokens("philosophy", "agriculture", "law", "strategy", "theology").
		Accessible("temple", "laboratory").ToMove(0).State()

	enginetest.Do(t, &s, "wonder:sphinx:temple")
	if s.ToMove != 0 {
		t.Fatalf("seat %d to move after the Sphinx, want 0", s.ToMove)
	}
	enginetest.Do(t, &s, "construct:laboratory")
	if s.Phase != engine.PhasePickBoardToken || s.ToMove != 0 {
		t.Fatalf("phase %s, seat %d to move; want seat 0's token pick", s.Phase, s.ToMove)
	}
	enginetest.Do(t, &s, "token:philosophy")
	if s.Phase != engine.PhasePlay || s.ToMove != 1 || s.Active != 1 {
		t.Fatalf("phase %s, to move %d, active %d; want seat 1's turn", s.Phase, s.ToMove, s.Active)
	}
}

// TestAuditR2MilsciStrategyFromARebuiltPairReachesTheTheologyExtraTurn: a
// Theology owner builds the Mausoleum (extra turn noted), rebuilds
// Dispensary (mortar pair) and picks Strategy. Strategy is owned from then
// on, so the red building constructed in the extra turn that follows gets
// its extra shield.
// Rule: docs/rules/wonders-tokens.md "Progress tokens" (lasting effects
// apply to everything the player does afterwards), "Strategy", "Theology"
// and "The Mausoleum"; docs/rules/military-science.md "Where shields come
// from".
func TestAuditR2MilsciStrategyFromARebuiltPairReachesTheTheologyExtraTurn(t *testing.T) {
	s := enginetest.New(t, 2022).Age(2).
		Wonders(0, "mausoleum").
		City(0, "pharmacist", "clay-pool", "clay-pit", "glassworks", "glass-blower", "press", "lumber-yard").
		Tokens(0, "theology").
		Discard("dispensary").
		BoardTokens("strategy", "philosophy", "law", "economy", "agriculture").
		Pawn(0).Accessible("temple", "horse-breeders").ToMove(0).State()

	enginetest.Do(t, &s, "wonder:mausoleum:temple", "revive:dispensary", "token:strategy")
	if s.Phase != engine.PhasePlay || s.ToMove != 0 {
		t.Fatalf("phase %s, seat %d to move; want seat 0's extra turn", s.Phase, s.ToMove)
	}
	enginetest.Do(t, &s, "construct:horse-breeders")
	if s.Military != 2 {
		t.Fatalf("pawn %d, want 2 (Horse Breeders' shield plus Strategy's)", s.Military)
	}
	if s.ToMove != 1 {
		t.Fatalf("seat %d to move, want 1", s.ToMove)
	}
}
