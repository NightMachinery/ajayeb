package engine_test

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

// Adversarial black-box tests for the "Wonder" lens: the 12 wonders, "play
// again", Theology, the seven-wonder rule, the Mausoleum, the Statue of
// Zeus and Circus Maximus, the Great Library's private draw, and the Appian
// Way. Every test is built only from docs/rules/*.md and
// design/rules-reference.md sections 2-8; none of it reads the engine's
// implementation. Helpers are prefixed awr1 so they cannot collide with any
// other slice's test helpers in this package.
//
// Seed is fixed and arbitrary; no test depends on its value beyond it being
// a valid NewGame seed.
const awr1Seed = 530900001

func awr1Card(t *testing.T, slug string) catalog.CardID {
	t.Helper()
	id, ok := catalog.CardBySlug(slug)
	if !ok {
		t.Fatalf("awr1Card: unknown card slug %q (test bug)", slug)
	}
	return id
}

func awr1Wonder(t *testing.T, slug string) catalog.WonderID {
	t.Helper()
	id, ok := catalog.WonderBySlug(slug)
	if !ok {
		t.Fatalf("awr1Wonder: unknown wonder slug %q (test bug)", slug)
	}
	return id
}

func awr1Token(t *testing.T, slug string) catalog.TokenID {
	t.Helper()
	id, ok := catalog.TokenBySlug(slug)
	if !ok {
		t.Fatalf("awr1Token: unknown token slug %q (test bug)", slug)
	}
	return id
}

// TestAdvWonderR1_TuckedCardGrantsNoChain checks that a card tucked under a
// wonder gives no chain discount to a later matching card. Rule:
// docs/rules/wonders-tokens.md ("A wonder can be built... the tucked card
// has no effect of any kind... its chain symbol does nothing") and
// design/rules-reference.md section 8, item 1.
//
// Theater (chain-out "mask") is tucked under seat 0's Pyramids. Statue
// (chain-in "mask") must then still cost its full printed price: with 0
// coins and no clay production, building it must fail as unaffordable, not
// succeed for free.
func TestAdvWonderR1_TuckedCardGrantsNoChain(t *testing.T) {
	s := enginetest.New(t, awr1Seed).
		Wonders(0, "pyramids").
		Built(0, "pyramids", "theater").
		Age(2).
		Accessible("statue").
		ToMove(0).
		Coins(0, 0).
		State()

	statue := awr1Card(t, "statue")
	q := engine.BuildingQuote(&s, 0, statue)
	if q.ChainFree {
		t.Errorf("Statue must not be chain-free: Theater was tucked under a wonder, not constructed, so its chain symbol does nothing (wonders-tokens.md)")
	}
	if reason := enginetest.MustFail(t, &s, "construct:statue"); reason != engine.ReasonUnaffordable {
		t.Errorf("construct:statue: want ReasonUnaffordable (no chain discount from a tucked card), got %v", reason)
	}
}

// TestAdvWonderR1_TuckedScienceCardDoesNotCompleteAPair checks that tucking
// the matching half of a science pair under a wonder does not complete the
// pair. Rule: wonders-tokens.md ("the tucked card has no effect of any
// kind") and design/rules-reference.md section 4 (pairs).
//
// Seat 0 owns Library (quill) in their city and tucks Scriptorium (quill)
// under a wonder. The pair must not complete: Sci[quill] stays at 1, not 2,
// and Scriptorium itself is not part of the city.
func TestAdvWonderR1_TuckedScienceCardDoesNotCompleteAPair(t *testing.T) {
	s := enginetest.New(t, awr1Seed).
		Wonders(0, "pyramids").
		City(0, "library").
		Built(0, "pyramids", "scriptorium").
		ToMove(0).
		State()

	if got := s.Players[0].Sci[catalog.Quill]; got != 1 {
		t.Errorf("Sci[quill]: want 1 (Library only; the tucked Scriptorium must not count), got %d", got)
	}
	scriptorium := awr1Card(t, "scriptorium")
	if s.Players[0].City.Has(scriptorium) {
		t.Errorf("a tucked card must not appear in the builder's city")
	}
	if s.Phase == engine.PhasePickBoardToken {
		t.Errorf("no pair was completed, so no board-token choice should be open")
	}
}

// TestAdvWonderR1_SeventhWonderRuleRemovesWhicheverPlayersWonderAndItScoresNothing
// checks the seven-wonder rule: "When the seventh wonder is built, by
// either player, the one wonder still unbuilt goes back to the box at
// once, whichever player drafted it." Rule: wonders-tokens.md ("The
// seven-wonder rule") and design/rules-reference.md section 2, "Seven
// Wonders, not eight".
//
// Seat 0 builds the seventh wonder overall. The removed, never-built
// eighth wonder belongs to seat 1 (not the builder), and it must score no
// victory points.
func TestAdvWonderR1_SeventhWonderRuleRemovesWhicheverPlayersWonderAndItScoresNothing(t *testing.T) {
	s := enginetest.New(t, awr1Seed).
		Wonders(0, "pyramids", "colossus", "great-lighthouse", "temple-of-artemis").
		Wonders(1, "mausoleum", "great-library", "piraeus", "hanging-gardens").
		Built(0, "pyramids", "theater").
		Built(0, "colossus", "altar").
		Built(0, "great-lighthouse", "baths").
		Built(1, "mausoleum", "circus").
		Built(1, "great-library", "arsenal").
		Built(1, "piraeus", "academy").
		Age(2).
		Accessible("rostrum").
		ToMove(0).
		Coins(0, 20).
		State()

	if s.WondersBuilt != 6 {
		t.Fatalf("awr1 setup bug: want 6 wonders built before the test action, got %d", s.WondersBuilt)
	}

	hangingGardens := awr1Wonder(t, "hanging-gardens")
	evs := enginetest.Do(t, &s, "wonder:temple-of-artemis:rostrum")

	if s.WondersBuilt != 7 {
		t.Errorf("WondersBuilt: want 7 after the seventh wonder is built, got %d", s.WondersBuilt)
	}

	var removedSlot *engine.WonderSlot
	for i := range s.Players[1].Wonders {
		if s.Players[1].Wonders[i].ID == hangingGardens {
			removedSlot = &s.Players[1].Wonders[i]
		}
	}
	if removedSlot == nil {
		t.Fatalf("awr1 setup bug: seat 1 does not have hanging-gardens")
	}
	if !removedSlot.Removed {
		t.Errorf("seat 1's never-built Hanging Gardens must be marked Removed once seat 0 builds the seventh wonder, even though seat 0 did not build it")
	}
	if removedSlot.Built {
		t.Errorf("the removed wonder must not be marked Built")
	}

	foundRemovedEvent := false
	for _, e := range evs {
		if e.Kind == engine.EvWonderRemoved {
			foundRemovedEvent = true
			if e.Seat != 1 {
				t.Errorf("EvWonderRemoved.Seat: want 1 (the owner of the removed wonder), got %v", e.Seat)
			}
			if e.Wonder != hangingGardens {
				t.Errorf("EvWonderRemoved.Wonder: want hanging-gardens, got %v", e.Wonder)
			}
		}
	}
	if !foundRemovedEvent {
		t.Errorf("expected an EvWonderRemoved event when the seventh wonder is built")
	}

	sc := engine.Scores(&s)
	// seat 1 actually built: mausoleum(2) + great-library(4) + piraeus(2) = 8.
	// hanging-gardens (3VP) must NOT be counted: it was never built.
	if sc[1].Wonders != 8 {
		t.Errorf("seat 1's Wonders score: want 8 (the removed, never-built Hanging Gardens scores nothing), got %d", sc[1].Wonders)
	}
	// seat 0 built all four: pyramids(9)+colossus(3)+great-lighthouse(4)+temple-of-artemis(0) = 16.
	if sc[0].Wonders != 16 {
		t.Errorf("seat 0's Wonders score: want 16, got %d", sc[0].Wonders)
	}
}

// TestAdvWonderR1_SeventhWonderRuleStillLetsThatWondersOwnChoiceResolve
// checks that when the wonder completing the seven-wonder count itself
// opens a pending choice, the removal and the choice both happen
// correctly: wonders-tokens.md, "The order of a wonder's effects" (the
// seven-wonder rule is step 2, the special choice is step 7, so applying
// the rule does not block or skip the choice that follows it in the same
// turn).
func TestAdvWonderR1_SeventhWonderRuleStillLetsThatWondersOwnChoiceResolve(t *testing.T) {
	s := enginetest.New(t, awr1Seed).
		Wonders(0, "pyramids", "colossus", "great-lighthouse", "statue-of-zeus").
		Wonders(1, "mausoleum", "great-library", "piraeus", "hanging-gardens").
		Built(0, "pyramids", "theater").
		Built(0, "colossus", "altar").
		Built(0, "great-lighthouse", "baths").
		Built(1, "mausoleum", "circus").
		Built(1, "great-library", "arsenal").
		Built(1, "piraeus", "academy").
		City(1, "quarry").
		Age(2).
		Accessible("rostrum").
		ToMove(0).
		Coins(0, 30).
		State()

	enginetest.Do(t, &s, "wonder:statue-of-zeus:rostrum")

	if s.WondersBuilt != 7 {
		t.Errorf("WondersBuilt: want 7, got %d", s.WondersBuilt)
	}
	hangingGardens := awr1Wonder(t, "hanging-gardens")
	for i := range s.Players[1].Wonders {
		if s.Players[1].Wonders[i].ID == hangingGardens && !s.Players[1].Wonders[i].Removed {
			t.Errorf("seat 1's Hanging Gardens should have been removed by the seven-wonder rule")
		}
	}
	if s.Phase != engine.PhaseDestroyCard {
		t.Fatalf("Phase: want PhaseDestroyCard (Statue of Zeus's own choice must still open), got %v", s.Phase)
	}
	if s.Destroy != catalog.Brown {
		t.Errorf("Destroy colour: want brown, got %v", s.Destroy)
	}
	if !enginetest.IsLegal(&s, "destroy:quarry") {
		t.Fatalf("destroy:quarry should be legal here")
	}

	enginetest.Do(t, &s, "destroy:quarry")
	quarry := awr1Card(t, "quarry")
	if s.Players[1].City.Has(quarry) {
		t.Errorf("quarry should have been destroyed")
	}
	if s.Phase == engine.PhaseDestroyCard {
		t.Errorf("the destroy choice should be resolved by now")
	}
}

// TestAdvWonderR1_StatueOfZeusSkipsDestroyWhenOpponentHasNoBrownCard checks
// decisions.md "Forced choices" and wonders-tokens.md: "If the opponent has
// no card of that colour, nothing is destroyed," and the wonder is still
// built. The shield still moves the pawn even though the destroy is
// skipped (wonders-tokens.md: the shield moves the pawn before the choice).
func TestAdvWonderR1_StatueOfZeusSkipsDestroyWhenOpponentHasNoBrownCard(t *testing.T) {
	s := enginetest.New(t, awr1Seed).
		Wonders(0, "statue-of-zeus").
		Age(2).
		Accessible("rostrum").
		ToMove(0).
		Pawn(0).
		Coins(0, 30).
		State()

	if !s.Players[1].City.Empty() {
		t.Fatalf("awr1 setup bug: seat 1's city should start empty")
	}

	evs := enginetest.Do(t, &s, "wonder:statue-of-zeus:rostrum")

	zeus := awr1Wonder(t, "statue-of-zeus")
	built := false
	for _, ws := range s.Players[0].Wonders {
		if ws.ID == zeus {
			built = ws.Built
		}
	}
	if !built {
		t.Errorf("the Statue of Zeus must still be built when there is nothing to destroy")
	}
	if s.Phase == engine.PhaseDestroyCard {
		t.Errorf("no destroy choice should be open when the opponent has no brown card")
	}
	if s.Military != 1 && s.Military != -1 {
		t.Errorf("the shield should still move the pawn by 1 space even though the destroy is skipped, got Military=%d", s.Military)
	}

	foundSkip := false
	for _, e := range evs {
		if e.Kind == engine.EvEffectSkipped && engine.Phase(e.A) == engine.PhaseDestroyCard {
			foundSkip = true
		}
	}
	if !foundSkip {
		t.Errorf("expected an EvEffectSkipped event naming PhaseDestroyCard")
	}
}

// TestAdvWonderR1_StatueOfZeusSupremacyEndsBeforeTheDestroyChoice checks
// decisions.md "Supremacy timing" and military-science.md "Military
// supremacy": if the Statue of Zeus's shield carries the pawn into the
// opponent's capital, the game ends immediately and the destroy choice
// never happens, even though the opponent does own a brown card (so the
// choice would otherwise have an option).
func TestAdvWonderR1_StatueOfZeusSupremacyEndsBeforeTheDestroyChoice(t *testing.T) {
	s := enginetest.New(t, awr1Seed).
		Wonders(0, "statue-of-zeus").
		City(1, "quarry").
		Age(2).
		Accessible("rostrum").
		ToMove(0).
		Pawn(8).
		Coins(0, 30).
		State()

	evs := enginetest.Do(t, &s, "wonder:statue-of-zeus:rostrum")

	if !s.Over() {
		t.Fatalf("the game should be over: the pawn should have reached seat 1's capital")
	}
	if s.Result.Kind != engine.VictoryMilitary {
		t.Errorf("Result.Kind: want VictoryMilitary, got %v", s.Result.Kind)
	}
	if s.Result.Winner != 0 {
		t.Errorf("Result.Winner: want seat 0 (pushed the pawn into seat 1's capital), got %v", s.Result.Winner)
	}
	if s.Phase == engine.PhaseDestroyCard {
		t.Errorf("Phase must not be PhaseDestroyCard: the destroy choice must never open after a supremacy")
	}
	quarry := awr1Card(t, "quarry")
	if !s.Players[1].City.Has(quarry) {
		t.Errorf("seat 1's quarry must still be in their city: the destroy choice never ran")
	}
	for _, e := range evs {
		if e.Kind == engine.EvDestroy {
			t.Errorf("no EvDestroy event should fire: the game ended before the choice")
		}
	}
}

// TestAdvWonderR1_CircusMaximusDestroysOnlyGreyCardsNotBrown checks that
// Circus Maximus targets grey cards, not brown ones, even when the
// opponent has both. Rule: wonders-tokens.md ("The Statue of Zeus and
// Circus Maximus"): "brown for the Statue of Zeus, grey for Circus
// Maximus."
func TestAdvWonderR1_CircusMaximusDestroysOnlyGreyCardsNotBrown(t *testing.T) {
	s := enginetest.New(t, awr1Seed).
		Wonders(0, "circus-maximus").
		City(1, "quarry", "glassworks").
		Age(2).
		Accessible("rostrum").
		ToMove(0).
		Coins(0, 30).
		State()

	enginetest.Do(t, &s, "wonder:circus-maximus:rostrum")

	if s.Phase != engine.PhaseDestroyCard {
		t.Fatalf("Phase: want PhaseDestroyCard, got %v", s.Phase)
	}
	if s.Destroy != catalog.Grey {
		t.Errorf("Destroy colour: want grey, got %v", s.Destroy)
	}
	if enginetest.IsLegal(&s, "destroy:quarry") {
		t.Errorf("destroy:quarry must not be legal: Circus Maximus destroys grey cards, not brown")
	}
	if !enginetest.IsLegal(&s, "destroy:glassworks") {
		t.Fatalf("destroy:glassworks should be legal")
	}

	enginetest.Do(t, &s, "destroy:glassworks")
	quarry := awr1Card(t, "quarry")
	glassworks := awr1Card(t, "glassworks")
	if s.Players[1].City.Has(glassworks) {
		t.Errorf("glassworks should have been destroyed")
	}
	if !s.Players[1].City.Has(quarry) {
		t.Errorf("quarry (brown) must be untouched by Circus Maximus")
	}
}

// TestAdvWonderR1_MausoleumSkippedWhenDiscardPileIsEmpty checks
// decisions.md "Forced choices": "If the discard pile is empty, nothing
// happens and the Wonder is still built."
func TestAdvWonderR1_MausoleumSkippedWhenDiscardPileIsEmpty(t *testing.T) {
	s := enginetest.New(t, awr1Seed).
		Wonders(0, "mausoleum").
		Age(2).
		Accessible("rostrum").
		ToMove(0).
		Coins(0, 20).
		State()

	if !s.Discard.Empty() {
		t.Fatalf("awr1 setup bug: discard pile should start empty")
	}

	evs := enginetest.Do(t, &s, "wonder:mausoleum:rostrum")

	mausoleum := awr1Wonder(t, "mausoleum")
	built := false
	for _, ws := range s.Players[0].Wonders {
		if ws.ID == mausoleum {
			built = ws.Built
		}
	}
	if !built {
		t.Errorf("the Mausoleum must still be built when the discard pile is empty")
	}
	if s.Phase == engine.PhaseBuildFromDiscard {
		t.Errorf("no rebuild choice should be open: the discard pile is empty")
	}
	foundSkip := false
	for _, e := range evs {
		if e.Kind == engine.EvEffectSkipped && engine.Phase(e.A) == engine.PhaseBuildFromDiscard {
			foundSkip = true
		}
	}
	if !foundSkip {
		t.Errorf("expected an EvEffectSkipped event naming PhaseBuildFromDiscard")
	}
}

// TestAdvWonderR1_MausoleumRevivedCardEntersBuildersCity checks
// wonders-tokens.md: the discard pool is "from any age and from either
// player," and the builder "constructs one card of their choice... for
// free." The revived card must land in the BUILDER's city, regardless of
// who originally discarded it.
func TestAdvWonderR1_MausoleumRevivedCardEntersBuildersCity(t *testing.T) {
	s := enginetest.New(t, awr1Seed).
		Wonders(0, "mausoleum").
		Discard("quarry").
		Age(2).
		Accessible("rostrum").
		ToMove(0).
		Coins(0, 20).
		State()

	enginetest.Do(t, &s, "wonder:mausoleum:rostrum")
	if s.Phase != engine.PhaseBuildFromDiscard {
		t.Fatalf("Phase: want PhaseBuildFromDiscard, got %v", s.Phase)
	}
	if !enginetest.IsLegal(&s, "revive:quarry") {
		t.Fatalf("revive:quarry should be legal")
	}
	enginetest.Do(t, &s, "revive:quarry")

	quarry := awr1Card(t, "quarry")
	if !s.Players[0].City.Has(quarry) {
		t.Errorf("the revived card must enter the builder's (seat 0's) city")
	}
	if s.Players[1].City.Has(quarry) {
		t.Errorf("the revived card must not appear in seat 1's city")
	}
	if s.Discard.Has(quarry) {
		t.Errorf("the revived card must leave the discard pile")
	}
}

// TestAdvWonderR1_MausoleumExcludesCardsTuckedUnderWonders checks
// decisions.md "The Mausoleum": the pool "never includes... the cards
// tucked under wonders." A card tucked under seat 1's Colossus must not be
// offered, so with nothing else on the discard pile, the Mausoleum's
// choice must be skipped entirely.
func TestAdvWonderR1_MausoleumExcludesCardsTuckedUnderWonders(t *testing.T) {
	s := enginetest.New(t, awr1Seed).
		Wonders(0, "mausoleum").
		Wonders(1, "colossus").
		Built(1, "colossus", "quarry").
		Age(2).
		Accessible("rostrum").
		ToMove(0).
		Coins(0, 30).
		State()

	if !s.Discard.Empty() {
		t.Fatalf("awr1 setup bug: discard pile should be empty (quarry is tucked, not discarded)")
	}

	enginetest.Do(t, &s, "wonder:mausoleum:rostrum")

	if s.Phase == engine.PhaseBuildFromDiscard {
		t.Errorf("a card tucked under a wonder must not be offered by the Mausoleum; legal keys: %v", enginetest.Keys(&s))
	}
}

// TestAdvWonderR1_MausoleumRebuildDoesNotTriggerUrbanismEvenWithAMatchingChain
// checks decisions.md "Urbanism and the Mausoleum": "Urbanism's 4-coin
// bonus applies only to free constructions through a chain. A card
// rebuilt by the Mausoleum does not earn it." The revived card (Statue)
// even has a chain-in that seat 0 happens to own (Theater), so a buggy
// "cost was zero, pay the bonus" implementation would wrongly pay it here.
func TestAdvWonderR1_MausoleumRebuildDoesNotTriggerUrbanismEvenWithAMatchingChain(t *testing.T) {
	s := enginetest.New(t, awr1Seed).
		Wonders(0, "mausoleum").
		Tokens(0, "urbanism").
		City(0, "theater").
		Discard("statue").
		Age(2).
		Accessible("rostrum").
		ToMove(0).
		Coins(0, 20).
		State()

	enginetest.Do(t, &s, "wonder:mausoleum:rostrum")
	if s.Phase != engine.PhaseBuildFromDiscard {
		t.Fatalf("Phase: want PhaseBuildFromDiscard, got %v", s.Phase)
	}
	before := s.Players[0].Coins
	enginetest.Do(t, &s, "revive:statue")
	after := s.Players[0].Coins

	if after != before {
		t.Errorf("Urbanism must not pay its 4-coin bonus for a Mausoleum rebuild (even though the revived card, Statue, has a chain-in seat 0 owns): coins before=%d after=%d", before, after)
	}
}

// TestAdvWonderR1_MausoleumRebuildOfARedCardGetsTheStrategyBonus checks
// wonders-tokens.md "Strategy": "every red building you construct
// afterwards gives 1 extra shield. This includes a red building rebuilt
// with the Mausoleum."
func TestAdvWonderR1_MausoleumRebuildOfARedCardGetsTheStrategyBonus(t *testing.T) {
	s := enginetest.New(t, awr1Seed).
		Wonders(0, "mausoleum").
		Tokens(0, "strategy").
		Discard("garrison").
		Age(2).
		Accessible("rostrum").
		ToMove(0).
		Pawn(0).
		Coins(0, 20).
		State()

	enginetest.Do(t, &s, "wonder:mausoleum:rostrum")
	if s.Phase != engine.PhaseBuildFromDiscard {
		t.Fatalf("Phase: want PhaseBuildFromDiscard, got %v", s.Phase)
	}
	before := s.Military
	enginetest.Do(t, &s, "revive:garrison")
	delta := s.Military - before
	if delta < 0 {
		delta = -delta
	}
	if delta != 2 {
		t.Errorf("Garrison gives 1 shield, plus 1 from Strategy, rebuilt through the Mausoleum: want |delta|=2, got %d", delta)
	}
}

// TestAdvWonderR1_MausoleumRebuildPaysAYellowCardsCoinReward checks
// wonders-tokens.md: a Mausoleum rebuild "enters the builder's city with
// all its effects... coin rewards from yellow cards and guilds."
func TestAdvWonderR1_MausoleumRebuildPaysAYellowCardsCoinReward(t *testing.T) {
	s := enginetest.New(t, awr1Seed).
		Wonders(0, "mausoleum").
		Discard("tavern").
		Age(2).
		Accessible("rostrum").
		ToMove(0).
		Coins(0, 20).
		State()

	enginetest.Do(t, &s, "wonder:mausoleum:rostrum")
	if s.Phase != engine.PhaseBuildFromDiscard {
		t.Fatalf("Phase: want PhaseBuildFromDiscard, got %v", s.Phase)
	}
	before := s.Players[0].Coins
	enginetest.Do(t, &s, "revive:tavern")
	after := s.Players[0].Coins

	if after-before != 4 {
		t.Errorf("Tavern pays +4 coins when it enters a city, including through the Mausoleum: want delta=4, got %d", after-before)
	}
}

// TestAdvWonderR1_GreatLibraryOfferIsPrivateToTheBuilder checks
// wonders-tokens.md "The Great Library": "Three of the five tokens in the
// box are drawn at random. Only the builder sees them." The engine's
// EvLibraryDraw events must be Private and belong to the builder's seat.
func TestAdvWonderR1_GreatLibraryOfferIsPrivateToTheBuilder(t *testing.T) {
	s := enginetest.New(t, awr1Seed).
		Wonders(0, "great-library").
		Age(1).
		Accessible("baths").
		ToMove(0).
		Coins(0, 20).
		State()

	evs := enginetest.Do(t, &s, "wonder:great-library:baths")

	if s.Phase != engine.PhasePickLibraryToken {
		t.Fatalf("Phase: want PhasePickLibraryToken, got %v", s.Phase)
	}
	count := 0
	for _, e := range evs {
		if e.Kind != engine.EvLibraryDraw {
			continue
		}
		count++
		if !e.Private {
			t.Errorf("EvLibraryDraw must be Private")
		}
		if e.Seat != 0 {
			t.Errorf("EvLibraryDraw.Seat: want the builder (0), got %v", e.Seat)
		}
		if int(e.A) < 0 || int(e.A) >= engine.LibraryDraw {
			t.Fatalf("EvLibraryDraw.A out of range: %d", e.A)
		}
		if s.Library[e.A] != e.Token {
			t.Errorf("EvLibraryDraw.Token at offer place %d: want %v (matching State.Library), got %v", e.A, s.Library[e.A], e.Token)
		}
	}
	if count != engine.LibraryDraw {
		t.Errorf("expected exactly %d EvLibraryDraw events, got %d", engine.LibraryDraw, count)
	}
}

// TestAdvWonderR1_GreatLibraryKeepsTenTokensConservedBetweenBoardAndBox
// checks design/rules-reference.md section 1 ("10 Progress tokens") and
// wonders-tokens.md "The Great Library": of the 3 drawn, 1 is kept and 2
// "go back to the box." The board, the box, and both players' tokens
// together must always total exactly 10, with the board itself unaffected
// by a Great Library draw (it draws from the box, not the board).
func TestAdvWonderR1_GreatLibraryKeepsTenTokensConservedBetweenBoardAndBox(t *testing.T) {
	s := enginetest.New(t, awr1Seed).
		Wonders(0, "great-library").
		Age(1).
		Accessible("baths").
		ToMove(0).
		Coins(0, 20).
		State()

	boardBefore := s.BoardTokens.Count()
	enginetest.Do(t, &s, "wonder:great-library:baths")
	if s.Phase != engine.PhasePickLibraryToken {
		t.Fatalf("Phase: want PhasePickLibraryToken, got %v", s.Phase)
	}
	kept := "token:" + s.Library[0].String()
	enginetest.Do(t, &s, kept)

	if s.BoardTokens.Count() != boardBefore {
		t.Errorf("BoardTokens must be unaffected by the Great Library draw: before=%d after=%d", boardBefore, s.BoardTokens.Count())
	}
	boxCount := 0
	for _, tk := range s.BoxTokens {
		if tk != 0 {
			boxCount++
		}
	}
	total := s.BoardTokens.Count() + s.Players[0].Tokens.Count() + s.Players[1].Tokens.Count() + boxCount
	if total != catalog.NumTokens {
		t.Errorf("token conservation: board(%d)+p0(%d)+p1(%d)+box(%d)=%d, want %d",
			s.BoardTokens.Count(), s.Players[0].Tokens.Count(), s.Players[1].Tokens.Count(), boxCount, total, catalog.NumTokens)
	}
	if s.Players[0].Tokens.Count() != 1 {
		t.Errorf("seat 0 should have kept exactly 1 token, got %d", s.Players[0].Tokens.Count())
	}
}

// TestAdvWonderR1_TheologyFromTheLibraryDoesNotGrantItsOwnExtraTurnButAppliesToTheNext
// checks wonders-tokens.md "The order of a wonder's effects": step 5 ("the
// extra turn, if any, is noted") happens BEFORE step 7 ("the special
// choice, if any, opens"). So picking Theology as the Great Library's own
// special choice must not retroactively grant that same build an extra
// turn (the Great Library is not among Theology's own sources and does
// not itself grant one). Theology must, however, apply starting with the
// very next wonder the player builds (docs/rules/wonders-tokens.md,
// Theology token: "every wonder you build afterwards grants one too").
func TestAdvWonderR1_TheologyFromTheLibraryDoesNotGrantItsOwnExtraTurnButAppliesToTheNext(t *testing.T) {
	s := enginetest.New(t, awr1Seed).
		Wonders(0, "great-library", "pyramids").
		BoardTokens("agriculture", "economy", "law", "masonry", "mathematics").
		BoxOrder("theology", "architecture", "philosophy").
		Age(1).
		Accessible("baths", "scriptorium", "altar").
		ToMove(0).
		Coins(0, 60).
		State()

	enginetest.Do(t, &s, "wonder:great-library:baths")
	if s.Phase != engine.PhasePickLibraryToken {
		t.Fatalf("Phase: want PhasePickLibraryToken, got %v", s.Phase)
	}
	// Read here, while the pending choice still blocks finishTurn: this is
	// step 5's noted value, taken before step 7 (the choice) ever opens.
	if s.ExtraTurn {
		t.Errorf("before the token choice resolves, the Great Library (which has no play-again of its own) must not have set ExtraTurn")
	}
	theology := awr1Token(t, "theology")
	found := false
	for _, tk := range s.Library {
		if tk == theology {
			found = true
		}
	}
	if !found {
		t.Fatalf("awr1 setup bug: theology was not drawn by the Great Library, got %v", s.Library)
	}

	enginetest.Do(t, &s, "token:theology")
	// The turn is now fully finished (no pending choice is left), so the
	// observable signal for "no extra turn" is that play passed to the
	// opponent, not the (by-now-consumed) ExtraTurn field.
	if s.ToMove != 1 {
		t.Errorf("picking Theology as the Great Library's own special choice must not retroactively grant this build an extra turn (effect order: extra turn is noted before the special choice opens): want ToMove=1, got %v", s.ToMove)
	}
	if !s.Players[0].Tokens.Has(theology) {
		t.Fatalf("awr1 setup bug: seat 0 should now own Theology")
	}

	enginetest.Do(t, &s, "discard:scriptorium")
	if s.ToMove != 0 {
		t.Fatalf("awr1 setup bug: play should return to seat 0")
	}

	enginetest.Do(t, &s, "wonder:pyramids:altar")
	if s.ToMove != 0 {
		t.Errorf("owning Theology, seat 0's next wonder (Pyramids, which grants no extra turn of its own) should still grant one, keeping seat 0 to move; got ToMove=%v", s.ToMove)
	}
}

// TestAdvWonderR1_AppianWayEconomyIncomeArrivesBeforeItsOwnLoss checks
// decisions.md "Effect order": the Economy income from the Appian Way
// builder's payment (step 1) must be credited to the opponent before the
// Appian Way's own 3-coin loss (step 6) is applied, so the "coins never go
// below zero" floor cannot let the Economy owner dodge the loss by zeroing
// out first. The opponent starts at 0 coins and the builder's payment
// sends them exactly 2 coins of trading income (less than the 3-coin
// loss): with the correct order they end at 0 (2 income, then floored by
// the 3 loss); with the ruled-out alternative order they would end at 2
// (loss floored at 0 first, then the income arrives unclipped).
func TestAdvWonderR1_AppianWayEconomyIncomeArrivesBeforeItsOwnLoss(t *testing.T) {
	s := enginetest.New(t, awr1Seed).
		Wonders(0, "appian-way").
		City(0, "shelf-quarry", "brickyard"). // seat 0 produces SS and CC: Appian Way's SS CC are free, only P must be traded
		Tokens(1, "economy").
		Age(2).
		Accessible("rostrum").
		ToMove(0).
		Coins(0, 10).
		Coins(1, 0).
		State()

	appianWay := awr1Wonder(t, "appian-way")
	q := engine.WonderQuote(&s, 0, appianWay)
	if q.Total != 2 || q.ToOpponent != 2 {
		t.Fatalf("awr1 setup bug: want Total=2 and ToOpponent=2 (1 papyrus traded at 2 coins, all to the Economy-owning opponent), got Total=%d ToOpponent=%d", q.Total, q.ToOpponent)
	}

	enginetest.Do(t, &s, "wonder:appian-way:rostrum")

	if got := s.Players[1].Coins; got != 0 {
		t.Errorf("the Economy income (step 1, +2) must be credited before the Appian Way's own loss (step 6, -3, floored at 0): want seat 1 coins=0, got %d (a value of 2 would mean the loss was floored to 0 BEFORE the income arrived)", got)
	}
}

// TestAdvWonderR1_AppianWayOpponentLossFloorsAtZeroWithoutEconomy checks
// design/rules-reference.md section 8, item 11 / decisions.md "Unlimited
// coins": "A player's coins never go below zero: a loss takes at most what
// the player has."
func TestAdvWonderR1_AppianWayOpponentLossFloorsAtZeroWithoutEconomy(t *testing.T) {
	s := enginetest.New(t, awr1Seed).
		Wonders(0, "appian-way").
		Age(2).
		Accessible("rostrum").
		ToMove(0).
		Coins(0, 30).
		Coins(1, 1).
		State()

	enginetest.Do(t, &s, "wonder:appian-way:rostrum")

	if got := s.Players[1].Coins; got != 0 {
		t.Errorf("seat 1 had 1 coin and loses 3 to the Appian Way: want floored at 0, got %d", got)
	}
}

// TestAdvWonderR1_ArchitectureDiscountsOnlyWondersByTheirTwoPriciestUnits
// checks wonders-tokens.md: "Architecture: every wonder you build
// afterwards costs 2 fewer resource units," and
// design/rules-reference.md section 8, item 6: "The player chooses which 2
// units are discounted. The best choice is the units that would cost most
// to buy." It also checks that Architecture does not discount an ordinary
// building's cost (only Masonry does that, for blue cards).
func TestAdvWonderR1_ArchitectureDiscountsOnlyWondersByTheirTwoPriciestUnits(t *testing.T) {
	s := enginetest.New(t, awr1Seed).
		Wonders(0, "mausoleum").
		Tokens(0, "architecture").
		City(1, "glassworks"). // opponent produces 1 glass, making glass pricier (3) than clay or papyrus (2)
		State()

	architecture := awr1Token(t, "architecture")
	mausoleum := awr1Wonder(t, "mausoleum")
	pay := engine.WonderPayment(&s, 0, mausoleum)

	if pay.DiscountBy != architecture {
		t.Errorf("Payment.DiscountBy: want architecture, got %v", pay.DiscountBy)
	}
	// Mausoleum costs CC GG P. Glass is the single priciest resource
	// (opponent produces it), so the 2 waived units should both be glass.
	if pay.Discount[catalog.Glass] != 2 {
		t.Errorf("Discount[glass]: want 2 (the 2 priciest units), got %d", pay.Discount[catalog.Glass])
	}
	if pay.Discount[catalog.Clay] != 0 {
		t.Errorf("Discount[clay]: want 0, got %d", pay.Discount[catalog.Clay])
	}
	if pay.Discount[catalog.Papyrus] != 0 {
		t.Errorf("Discount[papyrus]: want 0, got %d", pay.Discount[catalog.Papyrus])
	}

	aqueduct := awr1Card(t, "aqueduct") // a blue building, SSS
	bpay := engine.BuildingPayment(&s, 0, aqueduct)
	if bpay.DiscountBy != 0 {
		t.Errorf("Architecture must not discount a building's cost (that is Masonry's job): BuildingPayment.DiscountBy want 0, got %v", bpay.DiscountBy)
	}
}

// TestAdvWonderR1_StrategyNeverAddsAShieldToAWonder checks
// design/rules-reference.md section 8, item 4: "Strategy applies only to
// buildings," not to wonders. Colossus gives exactly 2 shields; owning
// Strategy must not make it 3.
func TestAdvWonderR1_StrategyNeverAddsAShieldToAWonder(t *testing.T) {
	s := enginetest.New(t, awr1Seed).
		Wonders(0, "colossus").
		Tokens(0, "strategy").
		Age(2).
		Accessible("rostrum").
		ToMove(0).
		Pawn(0).
		Coins(0, 30).
		State()

	enginetest.Do(t, &s, "wonder:colossus:rostrum")

	delta := s.Military
	if delta < 0 {
		delta = -delta
	}
	if delta != 2 {
		t.Errorf("Colossus gives 2 shields; Strategy must not add to a wonder's shields: want |Military|=2, got %d", delta)
	}
}

// TestAdvWonderR1_ExtraTurnsChainAcrossConsecutiveWonders checks
// decisions.md "Extra turns in a row": "Extra turns chain. A wonder that
// grants an extra turn, built during an extra turn, grants another one."
// Seat 0 builds the Sphinx, then Hanging Gardens during the resulting
// extra turn, and should still be to move for a third action before play
// finally passes to seat 1.
func TestAdvWonderR1_ExtraTurnsChainAcrossConsecutiveWonders(t *testing.T) {
	s := enginetest.New(t, awr1Seed).
		Wonders(0, "sphinx", "hanging-gardens").
		Age(1).
		Accessible("altar", "baths", "theater").
		ToMove(0).
		Coins(0, 40).
		State()

	// The observable sign of an extra turn, once a turn has fully resolved
	// (no pending choice left to read the transient ExtraTurn flag from),
	// is that the same seat remains ToMove instead of passing to the
	// opponent.
	enginetest.Do(t, &s, "wonder:sphinx:altar")
	if s.ToMove != 0 {
		t.Fatalf("awr1 setup bug: Sphinx should grant seat 0 an extra turn, got ToMove=%v", s.ToMove)
	}

	enginetest.Do(t, &s, "wonder:hanging-gardens:baths")
	if s.ToMove != 0 {
		t.Errorf("Hanging Gardens, built during seat 0's extra turn, should grant yet another extra turn, keeping seat 0 to move; got ToMove=%v", s.ToMove)
	}

	enginetest.Do(t, &s, "discard:theater")
	if s.ToMove != 1 {
		t.Errorf("after the chained extra turns are used up, play should finally pass to seat 1, got ToMove=%v", s.ToMove)
	}
}

// TestAdvWonderR1_AnyAccessibleCardCanBeTuckedAndOnlyTheWondersCostIsCharged
// checks turn.md: "Any accessible card can be used, including one you
// could not afford to construct," and "Pay the wonder's cost, not the
// card's." Pretorium costs a flat 8 coins and is accessible but
// unaffordable to construct; it can still be tucked under Colossus, and
// only Colossus's own (cheaper) cost is charged.
func TestAdvWonderR1_AnyAccessibleCardCanBeTuckedAndOnlyTheWondersCostIsCharged(t *testing.T) {
	s := enginetest.New(t, awr1Seed).
		Wonders(0, "colossus").
		City(0, "brickyard"). // produces CC, covering 2 of Colossus's 3 clay
		Age(3).
		Accessible("pretorium").
		ToMove(0).
		Coins(0, 6).
		State()

	if reason := enginetest.MustFail(t, &s, "construct:pretorium"); reason != engine.ReasonUnaffordable {
		t.Fatalf("awr1 setup bug: construct:pretorium should fail as unaffordable (6 coins < 8), got %v", reason)
	}

	colossus := awr1Wonder(t, "colossus")
	q := engine.WonderQuote(&s, 0, colossus)
	if !q.Affordable {
		t.Fatalf("awr1 setup bug: Colossus should be affordable with 6 coins, total=%d", q.Total)
	}
	before := s.Players[0].Coins

	enginetest.Do(t, &s, "wonder:colossus:pretorium")

	pretorium := awr1Card(t, "pretorium")
	built := false
	for _, ws := range s.Players[0].Wonders {
		if ws.ID == colossus {
			built = ws.Built
			if ws.Tucked != pretorium {
				t.Errorf("Tucked: want pretorium, got %v", ws.Tucked)
			}
		}
	}
	if !built {
		t.Errorf("Colossus should be built")
	}
	after := s.Players[0].Coins
	if before-after != q.Total {
		t.Errorf("only Colossus's own cost should be charged (%d), not Pretorium's: coins went from %d to %d (delta %d)", q.Total, before, after, before-after)
	}
}
