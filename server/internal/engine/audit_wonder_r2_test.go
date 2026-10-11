package engine_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

// Rules audit M10.R1, round 1, lens "wonders, guilds and scoring". These are
// black-box tests written from docs/rules/*.md and the engine design rules
// reference (sections 2 to 8). They aim at what the M2 audit
// (edgecases_wonder_r1_test.go) did not cover: interactions of two or three
// effects, boundary values and the order of resolution inside one action.
// Every expected value is derived from the rules, not from the engine.
//
// Helpers are prefixed auditR2Wonder so they cannot collide with any other
// test file of the package.

const auditR2WonderSeed = 1010202601

func auditR2WonderCard(t *testing.T, slug string) catalog.CardID {
	t.Helper()
	c, ok := catalog.CardBySlug(slug)
	if !ok {
		t.Fatalf("auditR2WonderCard: unknown card %q (test bug)", slug)
	}
	return c
}

func auditR2WonderWonder(t *testing.T, slug string) catalog.WonderID {
	t.Helper()
	w, ok := catalog.WonderBySlug(slug)
	if !ok {
		t.Fatalf("auditR2WonderWonder: unknown wonder %q (test bug)", slug)
	}
	return w
}

// auditR2WonderCardCoins sums the coin rewards that card c paid to seat in
// the events.
func auditR2WonderCardCoins(evs []engine.Event, seat engine.Seat, c catalog.CardID) (int16, bool) {
	var sum int16
	found := false
	for _, e := range evs {
		if e.Kind == engine.EvCoins && e.Seat == seat && e.Card == c {
			sum += e.A
			found = true
		}
	}
	return sum, found
}

func auditR2WonderHas(evs []engine.Event, k engine.EventKind) bool {
	for _, e := range evs {
		if e.Kind == k {
			return true
		}
	}
	return false
}

// auditR2WonderOtherDiscard returns a legal discard key that is not one of
// the excluded keys, so the other seat can pass its turn harmlessly.
func auditR2WonderOtherDiscard(t *testing.T, s *engine.State, exclude ...string) string {
	t.Helper()
	for _, k := range enginetest.Keys(s) {
		if strings.HasPrefix(k, "discard:") && !slices.Contains(exclude, k) {
			return k
		}
	}
	t.Fatalf("auditR2WonderOtherDiscard: no other discard is legal (test bug): %v", enginetest.Keys(s))
	return ""
}

// --- Destroy choices: what is and is not a target ---------------------------

// TestAuditR2WonderZeusIgnoresBrownCardTuckedUnderOpponentWonder: the
// Statue of Zeus destroys a brown card "in your opponent's city"
// (docs/rules/wonders-tokens.md, "The Statue of Zeus and Circus Maximus"); a
// tucked card is not part of any city and "has no effect of any kind"
// (wonders-tokens.md, "Wonders in general"; rules-reference section 8 item 1).
// A yellow card that produces brown resources (Caravansery) is not brown. So
// with a Lumber Yard only under the opponent's Colossus and a Caravansery in
// the city, there is no target: the choice is skipped and the turn passes
// (decisions.md "Forced choices").
func TestAuditR2WonderZeusIgnoresBrownCardTuckedUnderOpponentWonder(t *testing.T) {
	s := enginetest.New(t, auditR2WonderSeed).
		Wonders(0, "statue-of-zeus").
		Wonders(1, "colossus").
		Built(1, "colossus", "lumber-yard").
		City(1, "caravansery", "glassworks").
		Age(2).Accessible("rostrum").
		Coins(0, 60).Pawn(0).ToMove(0).
		State()

	ev := enginetest.Do(t, &s, "wonder:statue-of-zeus:rostrum")
	if s.Phase == engine.PhaseDestroyCard {
		t.Fatalf("destroy phase opened with targets %v; the only brown card of seat 1 is tucked under a wonder and the Caravansery is yellow", enginetest.Keys(&s))
	}
	if !auditR2WonderHas(ev, engine.EvEffectSkipped) {
		t.Fatalf("expected EvEffectSkipped for the Statue of Zeus, events %+v", ev)
	}
	if s.Phase != engine.PhasePlay || s.ToMove != 1 {
		t.Fatalf("phase %s, seat %d to move; want play with seat 1 to move", s.Phase, s.ToMove)
	}
}

// TestAuditR2WonderCircusMaximusIgnoresYellowGlassProducers: Circus Maximus
// destroys a grey card (wonders-tokens.md). The Forum and the Caravansery
// produce glass or papyrus but are yellow, and brown cards are not grey, so
// an opponent with only those has no target and the choice is skipped.
func TestAuditR2WonderCircusMaximusIgnoresYellowGlassProducers(t *testing.T) {
	s := enginetest.New(t, auditR2WonderSeed).
		Wonders(0, "circus-maximus").
		City(1, "forum", "caravansery", "quarry").
		Age(2).Accessible("rostrum").
		Coins(0, 60).Pawn(0).ToMove(0).
		State()

	ev := enginetest.Do(t, &s, "wonder:circus-maximus:rostrum")
	if s.Phase == engine.PhaseDestroyCard {
		t.Fatalf("destroy phase opened with targets %v; seat 1 has no grey card", enginetest.Keys(&s))
	}
	if !auditR2WonderHas(ev, engine.EvEffectSkipped) || s.ToMove != 1 {
		t.Fatalf("expected the choice to be skipped and the turn to pass; seat %d to move, events %+v", s.ToMove, ev)
	}
}

// TestAuditR2WonderZeusNeverTargetsTheBuildersOwnBrownCards: the targets are
// cards "in the opponent's city" only (wonders-tokens.md, "The Statue of Zeus
// and Circus Maximus"; for developers: "the legal keys are the opponent's
// cards of the named colour"). The builder owns two brown cards and the
// opponent one: exactly one key is legal, and destroying one's own card is
// refused.
func TestAuditR2WonderZeusNeverTargetsTheBuildersOwnBrownCards(t *testing.T) {
	s := enginetest.New(t, auditR2WonderSeed).
		Wonders(0, "statue-of-zeus").
		City(0, "quarry", "clay-pool").
		City(1, "lumber-yard").
		Age(2).Accessible("rostrum").
		Coins(0, 60).Pawn(0).ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:statue-of-zeus:rostrum")
	if s.Phase != engine.PhaseDestroyCard {
		t.Fatalf("phase %s, want destroyCard", s.Phase)
	}
	if got := enginetest.Keys(&s); !slices.Equal(got, []string{"destroy:lumber-yard"}) {
		t.Fatalf("legal keys %v, want only destroy:lumber-yard", got)
	}
	if r := enginetest.MustFail(t, &s, "destroy:quarry"); r != engine.ReasonNotTarget {
		t.Fatalf("destroying one's own quarry: reason %s, want notTarget", r)
	}
}

// --- Order of resolution inside one wonder ----------------------------------

// TestAuditR2WonderCircusMaximusShieldLootsBeforeTheDestroyChoice: the
// wonder's shield moves the pawn (step 4, "which can trigger looting") before
// the special choice opens (step 7) (wonders-tokens.md, "The order of a
// wonder's effects"; "The wonder's shield moves the pawn before the
// choice"). From pawn +2, the single shield reaches +3: seat 1 loses the
// 2-coin looting token at once, and only then is the destroy choice open.
func TestAuditR2WonderCircusMaximusShieldLootsBeforeTheDestroyChoice(t *testing.T) {
	s := enginetest.New(t, auditR2WonderSeed).
		Wonders(0, "circus-maximus").
		City(1, "glassworks").
		Age(2).Accessible("rostrum").
		Coins(0, 60).Coins(1, 5).Pawn(2).ToMove(0).
		State()

	ev := enginetest.Do(t, &s, "wonder:circus-maximus:rostrum")
	if s.Military != 3 {
		t.Fatalf("pawn %d, want 3", s.Military)
	}
	if s.Players[1].Coins != 3 {
		t.Fatalf("seat 1 has %d coins, want 3 (5 minus the 2-coin looting token)", s.Players[1].Coins)
	}
	if !auditR2WonderHas(ev, engine.EvLoot) {
		t.Fatalf("no looting event, events %+v", ev)
	}
	if s.Phase != engine.PhaseDestroyCard || s.ToMove != 0 {
		t.Fatalf("phase %s, seat %d to move; want the destroy choice for seat 0 after the looting", s.Phase, s.ToMove)
	}
	ev = enginetest.Do(t, &s, "destroy:glassworks")
	if auditR2WonderHas(ev, engine.EvLoot) || s.Players[1].Coins != 3 {
		t.Fatalf("the looting happened again or coins changed at the destroy: coins %d, events %+v", s.Players[1].Coins, ev)
	}
}

// TestAuditR2WonderTheologyExtraTurnSurvivesTheDestroyChoice: Theology makes
// every later wonder grant an extra turn (wonders-tokens.md, Theology); the
// extra turn is noted at step 5, before the special choice at step 7, and
// the turn (with its extra turn) ends only after the choice. So after the
// Statue of Zeus's destroy, the builder plays again.
func TestAuditR2WonderTheologyExtraTurnSurvivesTheDestroyChoice(t *testing.T) {
	s := enginetest.New(t, auditR2WonderSeed).
		Wonders(0, "statue-of-zeus").
		Tokens(0, "theology").
		City(1, "quarry").
		Age(2).Accessible("rostrum").
		Coins(0, 60).Pawn(0).ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:statue-of-zeus:rostrum")
	if s.Phase != engine.PhaseDestroyCard {
		t.Fatalf("phase %s, want destroyCard", s.Phase)
	}
	enginetest.Do(t, &s, "destroy:quarry")
	if s.Phase != engine.PhasePlay || s.ToMove != 0 || s.Active != 0 {
		t.Fatalf("phase %s, seat %d to move (active %d); want seat 0 to play again thanks to Theology", s.Phase, s.ToMove, s.Active)
	}
}

// TestAuditR2WonderTheologyExtraTurnSurvivesTheMausoleumRevive: the same as
// above for the Mausoleum: the extra turn from Theology is kept through the
// rebuild-from-discard choice (wonders-tokens.md, Theology and "The order of
// a wonder's effects").
func TestAuditR2WonderTheologyExtraTurnSurvivesTheMausoleumRevive(t *testing.T) {
	s := enginetest.New(t, auditR2WonderSeed).
		Wonders(0, "mausoleum").
		Tokens(0, "theology").
		Discard("baths").
		Age(2).Accessible("rostrum").
		Coins(0, 60).ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:mausoleum:rostrum")
	if s.Phase != engine.PhaseBuildFromDiscard {
		t.Fatalf("phase %s, want buildFromDiscard", s.Phase)
	}
	enginetest.Do(t, &s, "revive:baths")
	if s.Phase != engine.PhasePlay || s.ToMove != 0 {
		t.Fatalf("phase %s, seat %d to move; want seat 0 to play again thanks to Theology", s.Phase, s.ToMove)
	}
}

// TestAuditR2WonderExtraTurnIsNotCarriedPastAZeusBuiltInIt: extra turns
// chain only when the wonder built in the extra turn grants one itself
// (decisions.md "Extra turns in a row"). The Sphinx grants one; the Statue
// of Zeus, built in it, does not. After its destroy choice the turn passes
// to the opponent: the Sphinx's extra turn must not be reused.
func TestAuditR2WonderExtraTurnIsNotCarriedPastAZeusBuiltInIt(t *testing.T) {
	s := enginetest.New(t, auditR2WonderSeed).
		Wonders(0, "sphinx", "statue-of-zeus").
		City(1, "quarry").
		Age(1).Accessible("altar", "baths").
		Coins(0, 80).Pawn(0).ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:sphinx:altar")
	if s.ToMove != 0 || s.Phase != engine.PhasePlay {
		t.Fatalf("after the Sphinx: phase %s, seat %d to move; want seat 0 again", s.Phase, s.ToMove)
	}
	enginetest.Do(t, &s, "wonder:statue-of-zeus:baths", "destroy:quarry")
	if s.Phase != engine.PhasePlay || s.ToMove != 1 || s.ExtraTurn {
		t.Fatalf("after the Statue of Zeus: phase %s, seat %d to move, extra turn %v; want seat 1 to move", s.Phase, s.ToMove, s.ExtraTurn)
	}
}

// TestAuditR2WonderPiraeusProductionServesItsOwnExtraTurn: the wonder's
// production starts at step 2 ("its production (if any) starts"), and the
// extra turn begins after the turn ends (wonders-tokens.md, "The order of a
// wonder's effects"; Piraeus "from now on, produces one unit of glass or
// papyrus each turn"). So during Piraeus's own extra turn, a building whose
// only missing unit is glass costs nothing: seat 0 has 0 coins, the Sawmill
// covers the Courthouse's WW and Piraeus its G.
func TestAuditR2WonderPiraeusProductionServesItsOwnExtraTurn(t *testing.T) {
	s := enginetest.New(t, auditR2WonderSeed).
		Wonders(0, "piraeus").
		City(0, "sawmill", "clay-pool", "quarry").
		Age(2).Accessible("rostrum", "courthouse").
		Coins(0, 0).ToMove(0).
		State()

	if q := engine.WonderQuote(&s, 0, auditR2WonderWonder(t, "piraeus")); q.Total != 0 || !q.Affordable {
		t.Fatalf("setup: Piraeus quote %+v, want free from own production", q)
	}
	enginetest.Do(t, &s, "wonder:piraeus:rostrum")
	if s.ToMove != 0 || s.Phase != engine.PhasePlay {
		t.Fatalf("phase %s, seat %d to move; want Piraeus's extra turn", s.Phase, s.ToMove)
	}
	courthouse := auditR2WonderCard(t, "courthouse")
	if q := engine.BuildingQuote(&s, 0, courthouse); q.Total != 0 || !q.Affordable {
		t.Fatalf("Courthouse quote %+v in Piraeus's extra turn; want 0 coins (WW from the Sawmill, G from Piraeus)", q)
	}
	enginetest.Do(t, &s, "construct:courthouse")
	if !s.Players[0].City.Has(courthouse) {
		t.Fatal("the Courthouse is not in seat 0's city")
	}
}

// --- The Mausoleum ----------------------------------------------------------

// TestAuditR2WonderMausoleumRevivedArenaCountsTheMausoleumItself: the
// Mausoleum is marked built at step 2, before its choice at step 7
// (wonders-tokens.md, "The order of a wonder's effects"), and the revived
// card gets "all its effects", coin rewards from yellow cards included
// ("The Mausoleum"). The Arena pays 2 coins per wonder its owner has built
// (rules-reference section 5, Age III). With the Pyramids already built,
// the revived Arena must pay 2 x 2 = 4 coins.
func TestAuditR2WonderMausoleumRevivedArenaCountsTheMausoleumItself(t *testing.T) {
	s := enginetest.New(t, auditR2WonderSeed).
		Wonders(0, "mausoleum", "pyramids").
		Built(0, "pyramids", "").
		Discard("arena").
		Age(3).Accessible("palace").
		Coins(0, 60).ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:mausoleum:palace")
	before := s.Players[0].Coins
	ev := enginetest.Do(t, &s, "revive:arena")
	got, ok := auditR2WonderCardCoins(ev, 0, auditR2WonderCard(t, "arena"))
	if !ok || got != 4 {
		t.Fatalf("revived Arena paid %d coins (found %v); want 4 (2 per built wonder: the Pyramids and the Mausoleum). Events %+v", got, ok, ev)
	}
	if s.Players[0].Coins != before+4 {
		t.Fatalf("coins %d, want %d", s.Players[0].Coins, before+4)
	}
}

// TestAuditR2WonderMausoleumRevivedGuildPaysItsCoins: a revived card gets
// "coin rewards from yellow cards and guilds" (wonders-tokens.md, "The
// Mausoleum"), and the Merchants Guild pays 1 coin per yellow card in the
// city with the most yellow cards (ages-scoring.md, "Guilds"). The opponent
// has 3 yellow cards and the builder 1: the revived guild pays 3.
func TestAuditR2WonderMausoleumRevivedGuildPaysItsCoins(t *testing.T) {
	s := enginetest.New(t, auditR2WonderSeed).
		Wonders(0, "mausoleum").
		City(0, "wood-reserve").
		City(1, "tavern", "brewery", "forum").
		Discard("merchants-guild").
		Age(3).Accessible("palace").
		Coins(0, 60).ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:mausoleum:palace")
	ev := enginetest.Do(t, &s, "revive:merchants-guild")
	got, ok := auditR2WonderCardCoins(ev, 0, auditR2WonderCard(t, "merchants-guild"))
	if !ok || got != 3 {
		t.Fatalf("revived Merchants Guild paid %d coins (found %v); want 3 (seat 1's 3 yellow cards). Events %+v", got, ok, ev)
	}
	if sc := engine.Scores(&s); sc[0].Purple != 3 {
		t.Fatalf("purple points %d, want 3", sc[0].Purple)
	}
}

// TestAuditR2WonderMausoleumRevivesAnUnaffordableCardAtZeroCoins: the
// revived card is constructed "for free: no coins and no resources"
// (wonders-tokens.md, "The Mausoleum"). The builder pays the Mausoleum
// entirely with production and is left with exactly 0 coins; the Palace,
// which it could never construct (no stone, no wood, no coins), can still be
// revived, scoring its 7 blue points.
func TestAuditR2WonderMausoleumRevivesAnUnaffordableCardAtZeroCoins(t *testing.T) {
	s := enginetest.New(t, auditR2WonderSeed).
		Wonders(0, "mausoleum").
		City(0, "brickyard", "glassworks", "glass-blower", "press").
		Discard("palace").
		Age(2).Accessible("rostrum").
		Coins(0, 0).ToMove(0).
		State()

	if q := engine.WonderQuote(&s, 0, auditR2WonderWonder(t, "mausoleum")); q.Total != 0 || !q.Affordable {
		t.Fatalf("setup: Mausoleum quote %+v, want free from own production", q)
	}
	palace := auditR2WonderCard(t, "palace")
	if q := engine.BuildingQuote(&s, 0, palace); q.Affordable {
		t.Fatalf("setup: the Palace should be unaffordable at 0 coins, quote %+v", q)
	}
	enginetest.Do(t, &s, "wonder:mausoleum:rostrum")
	if s.Phase != engine.PhaseBuildFromDiscard {
		t.Fatalf("phase %s, want buildFromDiscard", s.Phase)
	}
	enginetest.Do(t, &s, "revive:palace")
	if !s.Players[0].City.Has(palace) || s.Players[0].Coins != 0 {
		t.Fatalf("palace in city %v, coins %d; want the Palace built with 0 coins spent", s.Players[0].City.Has(palace), s.Players[0].Coins)
	}
	if sc := engine.Scores(&s); sc[0].Blue != 7 {
		t.Fatalf("blue points %d, want 7 (the Palace)", sc[0].Blue)
	}
}

// TestAuditR2WonderMausoleumRevivedChainSymbolMakesALaterCardFree: a revived
// card enters the city with all its effects, "chain symbols" included
// (wonders-tokens.md, "The Mausoleum"). Baths carries the drop symbol that
// makes the Aqueduct free (rules-reference section 5). After reviving Baths,
// seat 0 (no stone, 0 coins) constructs the Aqueduct for free on its next
// turn.
func TestAuditR2WonderMausoleumRevivedChainSymbolMakesALaterCardFree(t *testing.T) {
	s := enginetest.New(t, auditR2WonderSeed).
		Wonders(0, "mausoleum").
		City(0, "brickyard", "glassworks", "glass-blower", "press").
		Discard("baths").
		Age(2).Accessible("rostrum", "aqueduct").
		Coins(0, 0).ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:mausoleum:rostrum", "revive:baths")
	if s.ToMove != 1 {
		t.Fatalf("seat %d to move after the revive, want 1", s.ToMove)
	}
	enginetest.Do(t, &s, auditR2WonderOtherDiscard(t, &s, "discard:aqueduct"))
	aqueduct := auditR2WonderCard(t, "aqueduct")
	q := engine.BuildingQuote(&s, 0, aqueduct)
	if !q.ChainFree || q.Total != 0 || !q.Affordable {
		t.Fatalf("Aqueduct quote %+v; want free through the drop chain of the revived Baths", q)
	}
	enginetest.Do(t, &s, "construct:aqueduct")
	if !s.Players[0].City.Has(aqueduct) {
		t.Fatal("the Aqueduct is not in seat 0's city")
	}
}

// TestAuditR2WonderMausoleumPairWithAnEmptyBoardGivesNothing: a revived
// green card can complete a science pair (wonders-tokens.md, "The
// Mausoleum"), but "if the board has no tokens left, the pair gives
// nothing" (rules-reference section 8 item 12; decisions.md "Forced
// choices": a choice with no option is skipped). With the five board tokens
// all owned by the opponent, reviving the Library next to the Scriptorium
// pairs the quill and the turn simply passes; the box tokens are never
// offered.
func TestAuditR2WonderMausoleumPairWithAnEmptyBoardGivesNothing(t *testing.T) {
	s := enginetest.New(t, auditR2WonderSeed).
		Wonders(0, "mausoleum").
		City(0, "scriptorium").
		Tokens(1, "agriculture", "architecture", "economy", "masonry", "mathematics").
		BoxOrder("law", "philosophy", "strategy", "theology", "urbanism").
		Discard("library").
		Age(2).Accessible("rostrum").
		Coins(0, 60).ToMove(0).
		State()

	if !s.BoardTokens.Empty() {
		t.Fatalf("setup: the board should hold no token, has %v", s.BoardTokens)
	}
	enginetest.Do(t, &s, "wonder:mausoleum:rostrum", "revive:library")
	if s.Phase != engine.PhasePlay || s.ToMove != 1 {
		t.Fatalf("phase %s, seat %d to move; want the turn to pass with no token pick (keys %v)", s.Phase, s.ToMove, enginetest.Keys(&s))
	}
	if !s.Players[0].Tokens.Empty() {
		t.Fatalf("seat 0 gained tokens %v from a pair with an empty board", s.Players[0].Tokens)
	}
}

// TestAuditR2WonderMausoleumOnTheLastAgeIIICardScoresTheRevivedCard: the
// game ends after the last card of Age III "and that turn has finished,
// including its pending choices" (ages-scoring.md, "The end of the game").
// The Mausoleum built with the last card still opens its choice, and the
// final score includes the revived Town Hall's 7 blue points.
func TestAuditR2WonderMausoleumOnTheLastAgeIIICardScoresTheRevivedCard(t *testing.T) {
	s := enginetest.New(t, auditR2WonderSeed).
		Wonders(0, "mausoleum").
		Discard("town-hall").
		Age(3).OnlyLeft("palace").
		Coins(0, 60).Pawn(0).ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:mausoleum:palace")
	if s.Over() || s.Phase != engine.PhaseBuildFromDiscard {
		t.Fatalf("phase %s; want the Mausoleum's choice before the game ends", s.Phase)
	}
	enginetest.Do(t, &s, "revive:town-hall")
	if !s.Over() || s.Result.Kind != engine.VictoryCivilian {
		t.Fatalf("phase %s, result %+v; want a civilian end", s.Phase, s.Result)
	}
	if got := s.Result.Scores[0]; got.Blue != 7 || got.Wonders != 2 {
		t.Fatalf("seat 0 final score %+v; want blue 7 (Town Hall) and wonders 2 (Mausoleum)", got)
	}
}

// TestAuditR2WonderTempleOfArtemisOnTheLastAgeIIICardStillPaysItsCoins: an
// extra turn earned on the last card is lost (ages-scoring.md), but the
// wonder's other effects are not: the 12 coins are taken at step 3
// (wonders-tokens.md) and are counted in the coin category of the final
// score (1 point per full 3 coins). The Temple of Artemis itself scores 0.
func TestAuditR2WonderTempleOfArtemisOnTheLastAgeIIICardStillPaysItsCoins(t *testing.T) {
	s := enginetest.New(t, auditR2WonderSeed).
		Wonders(0, "temple-of-artemis").
		Age(3).OnlyLeft("palace").
		Coins(0, 40).Pawn(0).ToMove(0).
		State()

	q := engine.WonderQuote(&s, 0, auditR2WonderWonder(t, "temple-of-artemis"))
	ev := enginetest.Do(t, &s, "wonder:temple-of-artemis:palace")
	if !s.Over() || s.Result.Kind != engine.VictoryCivilian {
		t.Fatalf("phase %s, result %+v; want a civilian end", s.Phase, s.Result)
	}
	wantCoins := 40 - q.Total + 12
	if s.Players[0].Coins != wantCoins {
		t.Fatalf("coins %d, want %d (40 - %d + 12)", s.Players[0].Coins, wantCoins, q.Total)
	}
	if got := s.Result.Scores[0]; got.Coins != wantCoins/3 || got.Wonders != 0 {
		t.Fatalf("seat 0 final score %+v; want coins %d and wonders 0", got, wantCoins/3)
	}
	if !auditR2WonderHas(ev, engine.EvExtraTurnLost) {
		t.Fatalf("expected the extra turn to be reported lost, events %+v", ev)
	}
}

// --- The Great Library ------------------------------------------------------

// TestAuditR2WonderGreatLibraryMathematicsCountsEveryTokenIncludingItself:
// the kept token "sits with the builder's other tokens" (wonders-tokens.md,
// "The Great Library"), and Mathematics scores 3 per token owned,
// Mathematics included (ages-scoring.md category 6). Seat 0 owns Law and
// Economy (worth no points) and keeps Mathematics from the library: 9
// token points.
func TestAuditR2WonderGreatLibraryMathematicsCountsEveryTokenIncludingItself(t *testing.T) {
	s := enginetest.New(t, auditR2WonderSeed).
		Wonders(0, "great-library").
		Tokens(0, "law", "economy").
		BoardTokens("agriculture", "architecture", "masonry", "philosophy", "strategy").
		BoxOrder("mathematics", "theology", "urbanism").
		Age(1).Accessible("baths").
		Coins(0, 60).ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:great-library:baths")
	if s.Phase != engine.PhasePickLibraryToken {
		t.Fatalf("phase %s, want pickLibraryToken", s.Phase)
	}
	enginetest.Do(t, &s, "token:mathematics")
	if sc := engine.Scores(&s); sc[0].Tokens != 9 {
		t.Fatalf("token points %d, want 9 (3 tokens x 3)", sc[0].Tokens)
	}
}

// --- Guilds -----------------------------------------------------------------

// TestAuditR2WonderShipownersGuildCountsOnlyBrownAndGreyCards: the
// Shipowners Guild counts brown and grey cards (ages-scoring.md, "Guilds").
// Yellow producers (Forum, Caravansery) and a producing wonder (the Great
// Lighthouse) are not brown or grey cards. Seat 0 has one Lumber Yard plus
// those, seat 1 nothing: 1 coin and 1 point.
func TestAuditR2WonderShipownersGuildCountsOnlyBrownAndGreyCards(t *testing.T) {
	s := enginetest.New(t, auditR2WonderSeed).
		Wonders(0, "great-lighthouse").
		Built(0, "great-lighthouse", "").
		City(0, "forum", "caravansery", "lumber-yard").
		Age(3).Accessible("shipowners-guild").
		Coins(0, 60).ToMove(0).
		State()

	ev := enginetest.Do(t, &s, "construct:shipowners-guild")
	got, _ := auditR2WonderCardCoins(ev, 0, auditR2WonderCard(t, "shipowners-guild"))
	if got != 1 {
		t.Fatalf("Shipowners Guild paid %d coins, want 1 (only the Lumber Yard)", got)
	}
	if sc := engine.Scores(&s); sc[0].Purple != 1 {
		t.Fatalf("purple points %d, want 1", sc[0].Purple)
	}
}

// TestAuditR2WonderScientistsGuildIgnoresTheLawToken: the Scientists Guild
// counts green cards (ages-scoring.md, "Guilds"). Law is a token that counts
// as a science symbol (wonders-tokens.md), not a green card. Seat 0 has one
// green card and Law: 1 coin and 1 point.
func TestAuditR2WonderScientistsGuildIgnoresTheLawToken(t *testing.T) {
	s := enginetest.New(t, auditR2WonderSeed).
		City(0, "scriptorium").
		Tokens(0, "law").
		Age(3).Accessible("scientists-guild").
		Coins(0, 60).ToMove(0).
		State()

	ev := enginetest.Do(t, &s, "construct:scientists-guild")
	got, _ := auditR2WonderCardCoins(ev, 0, auditR2WonderCard(t, "scientists-guild"))
	if got != 1 {
		t.Fatalf("Scientists Guild paid %d coins, want 1 (one green card; Law is not a card)", got)
	}
	if sc := engine.Scores(&s); sc[0].Purple != 1 {
		t.Fatalf("purple points %d, want 1", sc[0].Purple)
	}
}

// TestAuditR2WonderTacticiansGuildCountsRedCardsNotShields: the Tacticians
// Guild pays "1 coin and 1 point per red card" in the city with the most red
// cards (ages-scoring.md, "Guilds"). Seat 0 owns Walls and Parade Ground
// (2 cards, 4 shields); seat 1 owns Stable, Garrison and Palisade (3 cards,
// 3 shields). The count is 3 (seat 1's cards), not 4 (seat 0's shields).
func TestAuditR2WonderTacticiansGuildCountsRedCardsNotShields(t *testing.T) {
	s := enginetest.New(t, auditR2WonderSeed).
		City(0, "walls", "parade-ground").
		City(1, "stable", "garrison", "palisade").
		Age(3).Accessible("tacticians-guild").
		Coins(0, 60).Pawn(0).ToMove(0).
		State()

	ev := enginetest.Do(t, &s, "construct:tacticians-guild")
	got, _ := auditR2WonderCardCoins(ev, 0, auditR2WonderCard(t, "tacticians-guild"))
	if got != 3 {
		t.Fatalf("Tacticians Guild paid %d coins, want 3 (seat 1's three red cards)", got)
	}
	if sc := engine.Scores(&s); sc[0].Purple != 3 {
		t.Fatalf("purple points %d, want 3", sc[0].Purple)
	}
}

// TestAuditR2WonderGuildCoinsAtBuildPointsAfterAZeusDestroy: a guild's coins
// use the counts when it is constructed and its points the counts at the end
// (ages-scoring.md, "Guilds"; rules-reference section 8 item 13). Seat 0
// builds the Shipowners Guild while seat 1 has 3 brown cards (3 coins), then
// the Statue of Zeus destroys one of them: the coins are not taken back, and
// the points now count 2.
func TestAuditR2WonderGuildCoinsAtBuildPointsAfterAZeusDestroy(t *testing.T) {
	s := enginetest.New(t, auditR2WonderSeed).
		Wonders(0, "statue-of-zeus").
		City(1, "quarry", "clay-pool", "lumber-yard").
		Age(3).Accessible("shipowners-guild", "palace").
		Coins(0, 80).Pawn(0).ToMove(0).
		State()

	ev := enginetest.Do(t, &s, "construct:shipowners-guild")
	guild := auditR2WonderCard(t, "shipowners-guild")
	if got, _ := auditR2WonderCardCoins(ev, 0, guild); got != 3 {
		t.Fatalf("Shipowners Guild paid %d coins, want 3", got)
	}
	enginetest.Do(t, &s, auditR2WonderOtherDiscard(t, &s, "discard:palace"))
	if s.ToMove != 0 {
		t.Fatalf("setup: seat %d to move, want 0", s.ToMove)
	}
	enginetest.Do(t, &s, "wonder:statue-of-zeus:palace")
	before := s.Players[0].Coins
	ev = enginetest.Do(t, &s, "destroy:quarry")
	if _, ok := auditR2WonderCardCoins(ev, 0, guild); ok || s.Players[0].Coins != before {
		t.Fatalf("the guild's coins changed after the destroy: %d to %d, events %+v", before, s.Players[0].Coins, ev)
	}
	if sc := engine.Scores(&s); sc[0].Purple != 2 {
		t.Fatalf("purple points %d, want 2 (seat 1 has 2 brown cards left)", sc[0].Purple)
	}
}

// TestAuditR2WonderBuildersGuildAfterTheSevenWonderRule: the rulebook example
// in our own words (ages-scoring.md, "Guilds"): A has built 4 wonders and B
// has built 3, so whoever owns the Builders Guild scores 8. Here the owner is
// the player with 3 built wonders, whose fourth was removed by the
// seven-wonder rule; the removed wonder counts for nothing
// (wonders-tokens.md, "The seven-wonder rule"). Owner: 8 guild points and
// 9 + 3 + 4 = 16 wonder points.
func TestAuditR2WonderBuildersGuildAfterTheSevenWonderRule(t *testing.T) {
	s := enginetest.New(t, auditR2WonderSeed).
		Wonders(0, "pyramids", "colossus", "great-lighthouse", "temple-of-artemis").
		Wonders(1, "mausoleum", "great-library", "piraeus", "hanging-gardens").
		Built(0, "pyramids", "").
		Built(0, "colossus", "").
		Built(0, "great-lighthouse", "").
		Built(1, "mausoleum", "").
		Built(1, "great-library", "").
		Built(1, "piraeus", "").
		Built(1, "hanging-gardens", "").
		City(0, "builders-guild").
		Age(3).Pawn(0).
		State()

	if s.WondersBuilt != 7 {
		t.Fatalf("setup: %d wonders built, want 7", s.WondersBuilt)
	}
	sc := engine.Scores(&s)
	if sc[0].Purple != 8 {
		t.Fatalf("Builders Guild points %d, want 8 (2 x seat 1's 4 wonders)", sc[0].Purple)
	}
	if sc[0].Wonders != 16 {
		t.Fatalf("seat 0 wonder points %d, want 16 (the removed Temple of Artemis scores nothing)", sc[0].Wonders)
	}
}

// --- Final tally and tie-break ----------------------------------------------

// TestAuditR2WonderTieBreakUsesPrintedBlueNotGuildPoints: the tie-break
// compares "points from blue cards (category 1)" (ages-scoring.md, "Tie-break
// and shared victory"); the Magistrates Guild's points are purple points
// (category 4), even though it counts blue cards. Seat 0: Baths and Altar
// (6 blue) plus the Magistrates Guild (2 blue cards in the larger city, so
// 2 purple), and 2 coins after the final discard (0 points): 8. Seat 1: the
// Palace (7 blue) and 3 coins (1 point): 8. Seat 1 wins on blue, 7 to 6.
func TestAuditR2WonderTieBreakUsesPrintedBlueNotGuildPoints(t *testing.T) {
	s := enginetest.New(t, auditR2WonderSeed).
		City(0, "baths", "altar", "magistrates-guild").
		City(1, "palace").
		Coins(0, 0).Coins(1, 3).
		Age(3).OnlyLeft("obelisk").
		Pawn(0).ToMove(0).
		State()

	enginetest.Do(t, &s, "discard:obelisk")
	if !s.Over() || s.Result.Kind != engine.VictoryCivilian {
		t.Fatalf("phase %s, result %+v; want a civilian end", s.Phase, s.Result)
	}
	a, b := s.Result.Scores[0], s.Result.Scores[1]
	if a.Total != 8 || b.Total != 8 || a.Blue != 6 || b.Blue != 7 || a.Purple != 2 {
		t.Fatalf("scores %+v vs %+v; want totals 8 and 8, blue 6 and 7, seat 0 purple 2", a, b)
	}
	if s.Result.Winner != 1 {
		t.Fatalf("winner %d, want seat 1 (more blue-card points)", s.Result.Winner)
	}
}

// TestAuditR2WonderSharedVictoryHasNoFurtherTieBreak: when the totals and the
// blue-card points are both equal, "both players share the victory"
// (ages-scoring.md); no other category breaks the tie. Seat 0: Baths (3
// blue) and the Pyramids (9), 2 coins after the final discard: 12. Seat 1:
// Altar (3 blue) and 27 coins (9): 12. The categories differ, yet the result
// is shared.
func TestAuditR2WonderSharedVictoryHasNoFurtherTieBreak(t *testing.T) {
	s := enginetest.New(t, auditR2WonderSeed).
		Wonders(0, "pyramids").
		Built(0, "pyramids", "").
		City(0, "baths").
		City(1, "altar").
		Coins(0, 0).Coins(1, 27).
		Age(3).OnlyLeft("obelisk").
		Pawn(0).ToMove(0).
		State()

	enginetest.Do(t, &s, "discard:obelisk")
	if !s.Over() || s.Result.Kind != engine.VictoryCivilian {
		t.Fatalf("phase %s, result %+v; want a civilian end", s.Phase, s.Result)
	}
	a, b := s.Result.Scores[0], s.Result.Scores[1]
	if a.Total != 12 || b.Total != 12 || a.Blue != 3 || b.Blue != 3 {
		t.Fatalf("scores %+v vs %+v; want totals 12 and blue 3 each", a, b)
	}
	if s.Result.Winner != engine.SharedVictory {
		t.Fatalf("winner %d, want a shared victory", s.Result.Winner)
	}
}
