package engine_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

// Rules audit M10.R1, round 2, lens "wonders, guilds and scoring". These are
// black-box tests written from docs/rules/*.md and the engine design rules
// reference (sections 2 to 8). They go past the M2 audit
// (edgecases_wonder_r1_test.go) and round 1 (audit_wonder_r2_test.go):
// chains of two or three effects inside one turn, special choices on the
// last card of an age, and the final tally after those choices. Every
// expected value is derived from the rules, not from the engine.
//
// Helpers are prefixed auditR3Wonder so they cannot collide with any other
// test file of the package.

const auditR3WonderSeed = 1010202602

func auditR3WonderCard(t *testing.T, slug string) catalog.CardID {
	t.Helper()
	c, ok := catalog.CardBySlug(slug)
	if !ok {
		t.Fatalf("auditR3WonderCard: unknown card %q (test bug)", slug)
	}
	return c
}

func auditR3WonderWonder(t *testing.T, slug string) catalog.WonderID {
	t.Helper()
	w, ok := catalog.WonderBySlug(slug)
	if !ok {
		t.Fatalf("auditR3WonderWonder: unknown wonder %q (test bug)", slug)
	}
	return w
}

// auditR3WonderCardCoins sums the coin rewards that card c paid to seat in
// the events, and reports whether any was found.
func auditR3WonderCardCoins(evs []engine.Event, seat engine.Seat, c catalog.CardID) (int16, bool) {
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

// auditR3WonderIndex returns the index of the first event of kind k, or -1.
func auditR3WonderIndex(evs []engine.Event, k engine.EventKind) int {
	for i, e := range evs {
		if e.Kind == k {
			return i
		}
	}
	return -1
}

// auditR3WonderSortedKeys returns the legal keys, sorted.
func auditR3WonderSortedKeys(s *engine.State) []string {
	keys := enginetest.Keys(s)
	slices.Sort(keys)
	return keys
}

// --- Chains of choices inside one turn and across extra turns --------------

// TestAuditR3WonderZeusDestroyFeedsATheologyMausoleum: a card destroyed by
// the Statue of Zeus "goes to the discard pile, so the Mausoleum can bring it
// back later" (wonders-tokens.md, "The Statue of Zeus and Circus Maximus";
// rules-reference section 8 item 8), Theology gives every later wonder an
// extra turn (wonders-tokens.md, Theology), and extra turns chain
// (decisions.md, "Extra turns in a row"). Seat 0 builds Zeus and destroys
// seat 1's only brown card, the Quarry; in the extra turn it builds the
// Mausoleum, whose pool must offer that Quarry; the Quarry enters seat 0's
// city; and the Mausoleum's own Theology extra turn follows.
func TestAuditR3WonderZeusDestroyFeedsATheologyMausoleum(t *testing.T) {
	s := enginetest.New(t, auditR3WonderSeed).
		Wonders(0, "statue-of-zeus", "mausoleum").
		Tokens(0, "theology").
		City(1, "quarry").
		Age(1).Accessible("altar", "theater").
		Coins(0, 60).Pawn(0).ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:statue-of-zeus:altar")
	if s.Phase != engine.PhaseDestroyCard || s.ToMove != 0 {
		t.Fatalf("after Zeus: phase %s, to move %d; want the destroy choice for seat 0", s.Phase, s.ToMove)
	}
	enginetest.Do(t, &s, "destroy:quarry")
	if s.Phase != engine.PhasePlay || s.ToMove != 0 {
		t.Fatalf("after the destroy: phase %s, to move %d; want seat 0's Theology extra turn", s.Phase, s.ToMove)
	}
	enginetest.Do(t, &s, "wonder:mausoleum:theater")
	if s.Phase != engine.PhaseBuildFromDiscard {
		t.Fatalf("after the Mausoleum: phase %s, want the rebuild choice", s.Phase)
	}
	if !slices.Contains(enginetest.Keys(&s), "revive:quarry") {
		t.Fatalf("the destroyed Quarry is not offered; legal: %v", enginetest.Keys(&s))
	}
	enginetest.Do(t, &s, "revive:quarry")
	q := auditR3WonderCard(t, "quarry")
	if !s.Players[0].City.Has(q) || s.Players[1].City.Has(q) || s.Discard.Has(q) {
		t.Fatalf("the Quarry is not in seat 0's city alone (seat 0 %v, seat 1 %v, discard %v)",
			s.Players[0].City.Has(q), s.Players[1].City.Has(q), s.Discard.Has(q))
	}
	if s.Phase != engine.PhasePlay || s.ToMove != 0 {
		t.Fatalf("after the revive: phase %s, to move %d; want seat 0 to play again (Theology on the Mausoleum)", s.Phase, s.ToMove)
	}
}

// TestAuditR3WonderCircusMaximusOnTheLastAgeICardDestroysBeforeTheStarterChoice:
// an age ends when its last card is taken "and that turn has finished,
// including any pending choices it opened" (ages-scoring.md, "The end of an
// age"), and the weaker player, on whose side the pawn stands, chooses who
// starts the next age ("Who starts the next age"). Circus Maximus, built
// with the last Age I card from the centre, moves the pawn to +1 (seat 1
// is now weaker). The destroy choice is seat 0's and comes first; only then
// is Age II laid out and seat 1 asked to choose.
func TestAuditR3WonderCircusMaximusOnTheLastAgeICardDestroysBeforeTheStarterChoice(t *testing.T) {
	s := enginetest.New(t, auditR3WonderSeed).
		Wonders(0, "circus-maximus").
		City(1, "glassworks").
		Age(1).OnlyLeft("altar").
		Coins(0, 40).Pawn(0).ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:circus-maximus:altar")
	if s.Phase != engine.PhaseDestroyCard || s.ToMove != 0 || s.Age != 1 {
		t.Fatalf("after Circus Maximus: phase %s, to move %d, age %d; want seat 0's destroy choice still in Age I", s.Phase, s.ToMove, s.Age)
	}
	if s.Military != 1 {
		t.Fatalf("pawn %d, want 1 (the wonder's shield)", s.Military)
	}
	ev := enginetest.Do(t, &s, "destroy:glassworks")
	if !s.Discard.Has(auditR3WonderCard(t, "glassworks")) {
		t.Fatal("the destroyed Glassworks is not on the discard pile")
	}
	if s.Age != 2 || s.Phase != engine.PhaseChooseStarter || s.ToMove != 1 {
		t.Fatalf("after the destroy: age %d, phase %s, to move %d; want Age II with seat 1 (weaker) choosing", s.Age, s.Phase, s.ToMove)
	}
	if d, a := auditR3WonderIndex(ev, engine.EvDestroy), auditR3WonderIndex(ev, engine.EvAgeStart); d < 0 || a < 0 || d > a {
		t.Fatalf("destroy at event %d, age start at %d; want the destroy first. Events %+v", d, a, ev)
	}
}

// TestAuditR3WonderZeusOnTheLastAgeIIICardScoresAfterTheDestroy: the game
// ends after the last Age III card "and that turn has finished, including its
// pending choices"; an extra turn earned on it is lost (ages-scoring.md, "The
// end of the game"); guild points use "the cities at the end" ("Guilds").
// Seat 0 (Theology) builds Zeus with the last card and destroys one of seat
// 1's two brown cards. Seat 1's Shipowners Guild then counts 1 (1 against
// seat 0's 1), not 2; seat 0 scores Zeus's 3 points and 2 military points
// (pawn +1); the Theology extra turn is reported lost.
func TestAuditR3WonderZeusOnTheLastAgeIIICardScoresAfterTheDestroy(t *testing.T) {
	s := enginetest.New(t, auditR3WonderSeed).
		Wonders(0, "statue-of-zeus").
		Tokens(0, "theology").
		City(0, "clay-pool").
		City(1, "shipowners-guild", "lumber-yard", "quarry").
		Age(3).OnlyLeft("obelisk").
		Coins(0, 40).Pawn(0).ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:statue-of-zeus:obelisk")
	if s.Phase != engine.PhaseDestroyCard {
		t.Fatalf("phase %s, want the destroy choice before the game ends", s.Phase)
	}
	if got, want := auditR3WonderSortedKeys(&s), []string{"destroy:lumber-yard", "destroy:quarry"}; !slices.Equal(got, want) {
		t.Fatalf("legal %v, want %v", got, want)
	}
	ev := enginetest.Do(t, &s, "destroy:quarry")
	if !s.Over() || s.Result.Kind != engine.VictoryCivilian {
		t.Fatalf("phase %s, result %+v; want a civilian end", s.Phase, s.Result)
	}
	if auditR3WonderIndex(ev, engine.EvExtraTurnLost) < 0 {
		t.Fatalf("no extra-turn-lost event; events %+v", ev)
	}
	a, b := s.Result.Scores[0], s.Result.Scores[1]
	if b.Purple != 1 {
		t.Fatalf("seat 1 Shipowners points %d, want 1 (counted after the destroy)", b.Purple)
	}
	if a.Wonders != 3 || a.Military != 2 {
		t.Fatalf("seat 0 wonders %d, military %d; want 3 and 2", a.Wonders, a.Military)
	}
}

// --- The Mausoleum's revived card ending the game ---------------------------

// TestAuditR3WonderMausoleumRevivedSixthSymbolWinsAtOnce: the revived card
// gets "its science symbol (which can complete a pair or a scientific
// supremacy)" (wonders-tokens.md, "The Mausoleum"), and a supremacy ends the
// game the moment it happens (decisions.md, "Supremacy timing"), so the
// pending Theology extra turn never comes. Seat 0 owns five different
// symbols; reviving the University adds the globe, the sixth.
func TestAuditR3WonderMausoleumRevivedSixthSymbolWinsAtOnce(t *testing.T) {
	s := enginetest.New(t, auditR3WonderSeed).
		Wonders(0, "mausoleum").
		Tokens(0, "theology").
		City(0, "scriptorium", "workshop", "pharmacist", "apothecary", "academy").
		Discard("university", "baths").
		Age(3).Accessible("obelisk").
		Coins(0, 60).Pawn(0).ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:mausoleum:obelisk")
	enginetest.Do(t, &s, "revive:university")
	if !s.Over() || s.Result.Kind != engine.VictoryScience || s.Result.Winner != 0 {
		t.Fatalf("phase %s, result %+v; want a science victory for seat 0", s.Phase, s.Result)
	}
}

// TestAuditR3WonderMausoleumRevivedRedCardWithStrategyReachesTheCapital: the
// revived card gets "shields (with Strategy's bonus)" (wonders-tokens.md, "The
// Mausoleum"); Strategy adds 1 shield to each red building ("The ten
// tokens"); a pawn reaching the capital wins at once (military-science.md,
// "Military supremacy"). From +7, the revived Garrison (1 shield) plus
// Strategy (1) reaches +9: seat 0 wins. Without Strategy it would stop at +8.
func TestAuditR3WonderMausoleumRevivedRedCardWithStrategyReachesTheCapital(t *testing.T) {
	s := enginetest.New(t, auditR3WonderSeed).
		Wonders(0, "mausoleum").
		Tokens(0, "strategy").
		Discard("garrison", "baths").
		Age(2).Accessible("rostrum").
		Coins(0, 60).Pawn(7).ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:mausoleum:rostrum")
	enginetest.Do(t, &s, "revive:garrison")
	if !s.Over() || s.Result.Kind != engine.VictoryMilitary || s.Result.Winner != 0 {
		t.Fatalf("pawn %d, phase %s, result %+v; want a military victory for seat 0", s.Military, s.Phase, s.Result)
	}
}

// TestAuditR3WonderMausoleumRevivedBlueWinsTheTieBreakOnTheLastCard: the last
// card's pending choice is resolved before scoring (ages-scoring.md, "The end
// of the game"), the revived card brings its victory points
// (wonders-tokens.md, "The Mausoleum"), and equal totals are broken by blue
// points ("Tie-break and shared victory"). Seat 0 pays the Mausoleum with
// production (0 coins) and revives the Palace: 2 + 7 = 9. Seat 1 has the
// Pyramids (9) and 2 coins (no point). Equal totals; seat 0 wins on blue,
// 7 to 0.
func TestAuditR3WonderMausoleumRevivedBlueWinsTheTieBreakOnTheLastCard(t *testing.T) {
	s := enginetest.New(t, auditR3WonderSeed).
		Wonders(0, "mausoleum").
		Wonders(1, "pyramids").
		Built(1, "pyramids", "").
		City(0, "clay-pool", "clay-pit", "glassworks", "glass-blower", "press").
		Discard("palace").
		Age(3).OnlyLeft("obelisk").
		Coins(0, 0).Coins(1, 2).Pawn(0).ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:mausoleum:obelisk")
	if s.Phase != engine.PhaseBuildFromDiscard {
		t.Fatalf("phase %s, want the rebuild choice", s.Phase)
	}
	enginetest.Do(t, &s, "revive:palace")
	if !s.Over() || s.Result.Kind != engine.VictoryCivilian {
		t.Fatalf("phase %s, result %+v; want a civilian end", s.Phase, s.Result)
	}
	a, b := s.Result.Scores[0], s.Result.Scores[1]
	if a.Total != 9 || b.Total != 9 || a.Blue != 7 || b.Blue != 0 {
		t.Fatalf("scores %+v vs %+v; want totals 9 and 9, blue 7 and 0", a, b)
	}
	if s.Result.Winner != 0 {
		t.Fatalf("winner %d, want seat 0 (more blue points)", s.Result.Winner)
	}
}

// TestAuditR3WonderMausoleumRevivedLighthouseCountsItself: the revived card
// gets its yellow coin reward (wonders-tokens.md, "The Mausoleum"), and the
// Lighthouse pays "1 coin per yellow card you own, the Lighthouse included"
// (economy.md, "Coin rewards"; ages-scoring.md, "For developers": computed
// after the card has entered the city). With a Tavern already built, the
// revived Lighthouse pays 2.
func TestAuditR3WonderMausoleumRevivedLighthouseCountsItself(t *testing.T) {
	s := enginetest.New(t, auditR3WonderSeed).
		Wonders(0, "mausoleum").
		City(0, "tavern").
		Discard("lighthouse").
		Age(3).Accessible("obelisk").
		Coins(0, 60).Pawn(0).ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:mausoleum:obelisk")
	ev := enginetest.Do(t, &s, "revive:lighthouse")
	got, ok := auditR3WonderCardCoins(ev, 0, auditR3WonderCard(t, "lighthouse"))
	if !ok || got != 2 {
		t.Fatalf("revived Lighthouse paid %d (found %v), want 2 (Tavern and itself); events %+v", got, ok, ev)
	}
}

// TestAuditR3WonderReviveKeysAreExactlyTheDiscardPile: the pool "is every
// card on the discard pile ... from any age and from either player. It never
// includes the cards removed at setup or the cards tucked under wonders"
// (wonders-tokens.md, "The Mausoleum"); the legal keys are "all the cards on
// the discard pile" ("For developers"). The pile holds an Age I card, an Age
// II card and a guild; the opponent's city card, the card tucked under the
// opponent's wonder and the cards removed at setup are not keys.
func TestAuditR3WonderReviveKeysAreExactlyTheDiscardPile(t *testing.T) {
	s := enginetest.New(t, auditR3WonderSeed).
		Wonders(0, "mausoleum").
		Wonders(1, "pyramids").
		Built(1, "pyramids", "temple").
		City(1, "theater").
		Discard("baths", "courthouse", "merchants-guild").
		Age(3).Accessible("obelisk").
		Coins(0, 60).Pawn(0).ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:mausoleum:obelisk")
	want := []string{"revive:baths", "revive:courthouse", "revive:merchants-guild"}
	if got := auditR3WonderSortedKeys(&s); !slices.Equal(got, want) {
		t.Fatalf("legal %v, want exactly %v", got, want)
	}
	for a := range s.Removed {
		for _, c := range s.Removed[a] {
			if c == 0 {
				continue
			}
			if engine.IsLegal(&s, engine.ReviveAction(c)) == nil {
				t.Fatalf("removed card %s can be revived", catalog.Card(c).Slug)
			}
		}
	}
	enginetest.MustFail(t, &s, "revive:temple")
	enginetest.MustFail(t, &s, "revive:theater")
}

// --- Coins of wonders on the last card, and guilds at the end --------------

// TestAuditR3WonderAppianWayOnTheLastCardSetsTheFinalCoins: the Appian Way
// takes 3 coins from the bank and the opponent loses 3 (wonders-tokens.md);
// those effects still happen on the last card, whose extra turn is lost
// (ages-scoring.md, "The end of the game"); coins score 1 point per full 3
// ("The eight scoring categories"); the Moneylenders Guild scores per full 3
// coins of "the richer player at the end" ("Guilds"). Seat 0: 5 + 3 = 8
// coins (2 points). Seat 1: 14 - 3 = 11 (3 points), so the guild counts 11
// coins: 3 points (it would be 4 before the loss).
func TestAuditR3WonderAppianWayOnTheLastCardSetsTheFinalCoins(t *testing.T) {
	s := enginetest.New(t, auditR3WonderSeed).
		Wonders(0, "appian-way").
		City(0, "moneylenders-guild", "quarry", "stone-pit", "clay-pool", "clay-pit", "press").
		Age(3).OnlyLeft("obelisk").
		Coins(0, 5).Coins(1, 14).Pawn(0).ToMove(0).
		State()

	ev := enginetest.Do(t, &s, "wonder:appian-way:obelisk")
	if !s.Over() || s.Result.Kind != engine.VictoryCivilian {
		t.Fatalf("phase %s, result %+v; want a civilian end", s.Phase, s.Result)
	}
	if auditR3WonderIndex(ev, engine.EvExtraTurnLost) < 0 {
		t.Fatalf("no extra-turn-lost event; events %+v", ev)
	}
	if s.Players[0].Coins != 8 || s.Players[1].Coins != 11 {
		t.Fatalf("coins %d and %d, want 8 and 11", s.Players[0].Coins, s.Players[1].Coins)
	}
	a, b := s.Result.Scores[0], s.Result.Scores[1]
	if a.Coins != 2 || b.Coins != 3 || a.Purple != 3 || a.Wonders != 3 {
		t.Fatalf("seat 0 %+v, seat 1 %+v; want coin points 2 and 3, Moneylenders 3, wonders 3", a, b)
	}
}

// TestAuditR3WonderEconomyOnTheLastCardMakesTheMoneylendersOwnerRicher: with
// Economy, the trading coins the opponent pays come to the owner
// (wonders-tokens.md, Economy; economy.md), and the Moneylenders Guild uses
// the coins at the end (ages-scoring.md, "Guilds"). Seat 1 constructs the
// last card, the Obelisk (SS G), with no production, trading against seat
// 0's Quarry and Glassworks at 3 per unit: 9 coins, 12 to 3. Seat 0 (6
// coins) receives them: 15. Seat 0 is now the richer player: Moneylenders 5,
// coins 5, total 10; seat 1: Obelisk 5 + coins 1 = 6.
func TestAuditR3WonderEconomyOnTheLastCardMakesTheMoneylendersOwnerRicher(t *testing.T) {
	s := enginetest.New(t, auditR3WonderSeed).
		Tokens(0, "economy").
		City(0, "moneylenders-guild", "quarry", "glassworks").
		Age(3).OnlyLeft("obelisk").
		Coins(0, 6).Coins(1, 12).Pawn(0).ToMove(1).
		State()

	enginetest.Do(t, &s, "construct:obelisk")
	if !s.Over() || s.Result.Kind != engine.VictoryCivilian {
		t.Fatalf("phase %s, result %+v; want a civilian end", s.Phase, s.Result)
	}
	if s.Players[0].Coins != 15 || s.Players[1].Coins != 3 {
		t.Fatalf("coins %d and %d, want 15 and 3", s.Players[0].Coins, s.Players[1].Coins)
	}
	a, b := s.Result.Scores[0], s.Result.Scores[1]
	if a.Purple != 5 || a.Coins != 5 || a.Total != 10 || b.Blue != 5 || b.Coins != 1 || b.Total != 6 {
		t.Fatalf("seat 0 %+v, seat 1 %+v; want seat 0 purple 5, coins 5, total 10; seat 1 blue 5, coins 1, total 6", a, b)
	}
	if s.Result.Winner != 0 {
		t.Fatalf("winner %d, want seat 0", s.Result.Winner)
	}
}

// TestAuditR3WonderTuckedGuildPaysAndScoresNothing: a tucked card "has no
// effect of any kind: it is not part of your city" (turn.md, "Build a
// wonder"; rules-reference section 8 item 1). The Merchants Guild tucked
// under the Pyramids pays no coins (seat 1 has 2 yellow cards) and scores no
// purple points; the builder pays only the wonder's cost.
func TestAuditR3WonderTuckedGuildPaysAndScoresNothing(t *testing.T) {
	s := enginetest.New(t, auditR3WonderSeed).
		Wonders(0, "pyramids").
		City(1, "tavern", "brewery").
		Age(3).Accessible("merchants-guild").
		Coins(0, 60).Pawn(0).ToMove(0).
		State()

	cost := engine.WonderQuote(&s, 0, auditR3WonderWonder(t, "pyramids")).Total
	ev := enginetest.Do(t, &s, "wonder:pyramids:merchants-guild")
	g := auditR3WonderCard(t, "merchants-guild")
	if got, ok := auditR3WonderCardCoins(ev, 0, g); ok {
		t.Fatalf("the tucked guild paid %d coins; events %+v", got, ev)
	}
	if want := int16(60) - int16(cost); s.Players[0].Coins != want {
		t.Fatalf("seat 0 has %d coins, want %d (60 minus the wonder's %d)", s.Players[0].Coins, want, cost)
	}
	if s.Players[0].City.Has(g) {
		t.Fatal("the tucked guild is in seat 0's city")
	}
	if sc := engine.Scores(&s); sc[0].Purple != 0 || sc[1].Purple != 0 {
		t.Fatalf("purple points %d and %d, want 0 and 0", sc[0].Purple, sc[1].Purple)
	}
}

// TestAuditR3WonderTempleOfArtemisIsABuiltWonderWorthNothing: the Temple of
// Artemis is worth 0 points (wonders-tokens.md), but it is a built wonder:
// the Arena pays "2 coins per wonder you have built" (economy.md, "Coin
// rewards"), and the Builders Guild scores 2 points per wonder of the player
// with the most built wonders (ages-scoring.md, "Guilds"). Seat 0 has built
// only the Temple: the Arena pays 2; seat 1's Builders Guild scores 2 (seat
// 0's one wonder against seat 1's none); seat 0's wonder points are 0.
func TestAuditR3WonderTempleOfArtemisIsABuiltWonderWorthNothing(t *testing.T) {
	s := enginetest.New(t, auditR3WonderSeed).
		Wonders(0, "temple-of-artemis").
		Built(0, "temple-of-artemis", "").
		City(1, "builders-guild").
		Age(3).Accessible("arena").
		Coins(0, 60).Pawn(0).ToMove(0).
		State()

	ev := enginetest.Do(t, &s, "construct:arena")
	if got, ok := auditR3WonderCardCoins(ev, 0, auditR3WonderCard(t, "arena")); !ok || got != 2 {
		t.Fatalf("Arena paid %d (found %v), want 2", got, ok)
	}
	sc := engine.Scores(&s)
	if sc[1].Purple != 2 || sc[0].Wonders != 0 {
		t.Fatalf("seat 1 Builders Guild %d, seat 0 wonder points %d; want 2 and 0", sc[1].Purple, sc[0].Wonders)
	}
}

// TestAuditR3WonderFinalTallyEveryCategory: the eight categories and their
// total (ages-scoring.md, "The eight scoring categories"), after a final
// discard worth 2 coins plus 1 per yellow card (turn.md, "Discard it for
// coins").
//
// Seat 0 discards the last card with 2 yellow cards: 5 + 4 = 9 coins.
//
//   - Seat 0: blue 3 (Baths); green 2 (Library); yellow 6 (Lighthouse and
//     Arena, 3 each); purple 3 (Magistrates Guild: seat 1's 3 blue cards);
//     wonders 9 (Pyramids 9, Temple of Artemis 0); tokens 20 (Mathematics
//     3 x 3, Agriculture 4, Philosophy 7); coins 3; military 5 (pawn +4).
//     Total 51.
//   - Seat 1: blue 17 (Palace 7, Town Hall 7, Altar 3); wonders 6 (Sphinx);
//     tokens 0 (Law); coins 6 (20 coins); nothing else. Total 29.
func TestAuditR3WonderFinalTallyEveryCategory(t *testing.T) {
	s := enginetest.New(t, auditR3WonderSeed).
		Wonders(0, "pyramids", "temple-of-artemis").
		Wonders(1, "sphinx").
		Built(0, "pyramids", "").Built(0, "temple-of-artemis", "").
		Built(1, "sphinx", "").
		City(0, "baths", "library", "lighthouse", "arena", "magistrates-guild").
		City(1, "palace", "town-hall", "altar").
		Tokens(0, "mathematics", "agriculture", "philosophy").
		Tokens(1, "law").
		Age(3).OnlyLeft("obelisk").
		Coins(0, 5).Coins(1, 20).Pawn(4).ToMove(0).
		State()

	enginetest.Do(t, &s, "discard:obelisk")
	if !s.Over() || s.Result.Kind != engine.VictoryCivilian {
		t.Fatalf("phase %s, result %+v; want a civilian end", s.Phase, s.Result)
	}
	want := [2]engine.Score{
		{Blue: 3, Green: 2, Yellow: 6, Purple: 3, Wonders: 9, Tokens: 20, Coins: 3, Military: 5, Total: 51},
		{Blue: 17, Wonders: 6, Coins: 6, Total: 29},
	}
	if s.Result.Scores != want {
		t.Fatalf("scores\n%+v\n%+v\nwant\n%+v\n%+v", s.Result.Scores[0], s.Result.Scores[1], want[0], want[1])
	}
	if s.Result.Winner != 0 {
		t.Fatalf("winner %d, want seat 0", s.Result.Winner)
	}
}

// --- Production and the seven-wonder rule -----------------------------------

// TestAuditR3WonderPiraeusGivesOneUnitPerTurn: Piraeus "produces one unit of
// glass or papyrus each turn" (wonders-tokens.md), not one of each. Seat 0
// has the Sawmill (WW) and Piraeus; the Hanging Gardens (WW G P) still needs
// one unit at 2 coins (economy.md, trading with an opponent who produces
// none). At 1 coin the wonder is illegal; at exactly 2 it is legal, leaving
// 0 before its own 6 coins.
func TestAuditR3WonderPiraeusGivesOneUnitPerTurn(t *testing.T) {
	build := func(coins int) engine.State {
		return enginetest.New(t, auditR3WonderSeed).
			Wonders(0, "piraeus", "hanging-gardens").
			Built(0, "piraeus", "").
			City(0, "sawmill").
			Age(2).Accessible("rostrum").
			Coins(0, coins).Coins(1, 0).Pawn(0).ToMove(0).
			State()
	}
	s := build(1)
	enginetest.MustFail(t, &s, "wonder:hanging-gardens:rostrum")
	s = build(2)
	ev := enginetest.Do(t, &s, "wonder:hanging-gardens:rostrum")
	paid := int16(-1)
	for _, e := range ev {
		if e.Kind == engine.EvPay && e.Seat == 0 {
			paid = e.A
		}
	}
	if paid != 2 || s.Players[0].Coins != 6 {
		t.Fatalf("paid %d, now %d coins; want 2 paid and 0 + 6 = 6", paid, s.Players[0].Coins)
	}
}

// TestAuditR3WonderSixthWonderRemovesNothingSeventhRemovesTheLast: "When the
// seventh wonder is built, by either player, the one wonder still unbuilt goes
// back to the box at once" (wonders-tokens.md, "The seven-wonder rule"). With
// five built, seat 0's sixth (the Great Lighthouse) removes nothing, and
// both seats keep their last unbuilt wonder. Seat 1's seventh (Circus
// Maximus, whose destroy is skipped: seat 0 has no grey card) removes seat
// 0's Statue of Zeus, which seat 0 then cannot build.
func TestAuditR3WonderSixthWonderRemovesNothingSeventhRemovesTheLast(t *testing.T) {
	s := enginetest.New(t, auditR3WonderSeed).
		Wonders(0, "pyramids", "colossus", "great-lighthouse", "statue-of-zeus").
		Wonders(1, "mausoleum", "great-library", "sphinx", "circus-maximus").
		Built(0, "pyramids", "").Built(0, "colossus", "").
		Built(1, "mausoleum", "").Built(1, "great-library", "").Built(1, "sphinx", "").
		Age(1).Accessible("altar", "theater", "baths").
		Coins(0, 60).Coins(1, 60).Pawn(0).ToMove(0).
		State()

	ev := enginetest.Do(t, &s, "wonder:great-lighthouse:altar")
	if i := auditR3WonderIndex(ev, engine.EvWonderRemoved); i >= 0 {
		t.Fatalf("the sixth wonder removed %+v", ev[i])
	}
	if s.WondersBuilt != 6 || s.Players[0].Wonders[3].Removed || s.Players[1].Wonders[3].Removed {
		t.Fatalf("after the sixth: %d built, removed flags %v and %v; want 6 and none", s.WondersBuilt,
			s.Players[0].Wonders[3].Removed, s.Players[1].Wonders[3].Removed)
	}
	if s.ToMove != 1 {
		t.Fatalf("to move %d, want seat 1", s.ToMove)
	}
	ev = enginetest.Do(t, &s, "wonder:circus-maximus:theater")
	i := auditR3WonderIndex(ev, engine.EvWonderRemoved)
	if i < 0 || ev[i].Seat != 0 || ev[i].Wonder != auditR3WonderWonder(t, "statue-of-zeus") {
		t.Fatalf("want seat 0's Statue of Zeus removed; events %+v", ev)
	}
	if !s.Players[0].Wonders[3].Removed || s.Players[0].Wonders[3].Built {
		t.Fatalf("Statue of Zeus slot %+v, want removed and unbuilt", s.Players[0].Wonders[3])
	}
	if s.ToMove != 0 || s.Phase != engine.PhasePlay {
		t.Fatalf("phase %s, to move %d; want seat 0 to play", s.Phase, s.ToMove)
	}
	enginetest.MustFail(t, &s, "wonder:statue-of-zeus:baths")
	for _, k := range enginetest.Keys(&s) {
		if strings.HasPrefix(k, "wonder:") {
			t.Fatalf("a wonder key is still legal: %s", k)
		}
	}
}

// TestAuditR3WonderHangingGardensCoinsArriveBeforeItsExtraTurn: the wonder's
// coins are taken at step 3 and the extra turn begins after the turn ends
// (wonders-tokens.md, "The order of a wonder's effects"). Seat 0 pays the
// Hanging Gardens with exactly its 4 coins (G and P at 2 each), is left with
// 0, receives 6, and in the extra turn can construct the Customs House
// (4 coins), which only the wonder's coins can pay for.
func TestAuditR3WonderHangingGardensCoinsArriveBeforeItsExtraTurn(t *testing.T) {
	s := enginetest.New(t, auditR3WonderSeed).
		Wonders(0, "hanging-gardens").
		City(0, "sawmill").
		Age(2).Accessible("rostrum", "customs-house").
		Coins(0, 4).Coins(1, 0).Pawn(0).ToMove(0).
		State()

	ev := enginetest.Do(t, &s, "wonder:hanging-gardens:rostrum")
	if i := auditR3WonderIndex(ev, engine.EvPay); i < 0 || ev[i].A != 4 {
		t.Fatalf("want the wonder to cost all 4 coins; events %+v", ev)
	}
	if s.Players[0].Coins != 6 || s.ToMove != 0 || s.Phase != engine.PhasePlay {
		t.Fatalf("coins %d, to move %d, phase %s; want 0 + 6 = 6 and seat 0's extra turn", s.Players[0].Coins, s.ToMove, s.Phase)
	}
	enginetest.Do(t, &s, "construct:customs-house")
	if s.Players[0].Coins != 2 {
		t.Fatalf("coins %d after the Customs House, want 2", s.Players[0].Coins)
	}
}

// TestAuditR3WonderAppianWayWithTheologyOnTheLastAgeIICard: Theology does not
// stack with a wonder's own extra turn ("A wonder that already grants one
// still grants only one", wonders-tokens.md, Theology; rules-reference
// section 8 item 3), an extra turn earned on the last card is lost
// (ages-scoring.md, "The end of an age"), the opponent's loss stops at 0
// (rules-reference section 8 item 11), and with the pawn in the centre the
// last active player chooses who starts (ages-scoring.md; decisions.md,
// "Centred pawn"). Seat 0 (0 coins) pays the Appian Way with production
// and gets 3; seat 1 drops from 2 to 0; one extra turn is noted and lost;
// Age III is laid out and seat 0 chooses.
func TestAuditR3WonderAppianWayWithTheologyOnTheLastAgeIICard(t *testing.T) {
	s := enginetest.New(t, auditR3WonderSeed).
		Wonders(0, "appian-way").
		Tokens(0, "theology").
		City(0, "quarry", "stone-pit", "clay-pool", "clay-pit", "press").
		Age(2).OnlyLeft("rostrum").
		Coins(0, 0).Coins(1, 2).Pawn(0).ToMove(0).
		State()

	ev := enginetest.Do(t, &s, "wonder:appian-way:rostrum")
	if s.Players[0].Coins != 3 || s.Players[1].Coins != 0 {
		t.Fatalf("coins %d and %d, want 3 and 0", s.Players[0].Coins, s.Players[1].Coins)
	}
	var extra, lost int
	for _, e := range ev {
		switch e.Kind {
		case engine.EvExtraTurn:
			extra++
		case engine.EvExtraTurnLost:
			lost++
		}
	}
	if extra != 1 || lost != 1 {
		t.Fatalf("%d extra-turn and %d extra-turn-lost events, want 1 and 1; events %+v", extra, lost, ev)
	}
	if s.Age != 3 || s.Phase != engine.PhaseChooseStarter || s.ToMove != 0 {
		t.Fatalf("age %d, phase %s, to move %d; want Age III with seat 0 (last active, centred pawn) choosing", s.Age, s.Phase, s.ToMove)
	}
}

// TestAuditR3WonderGuildTiesCountTheSharedNumber: "When both cities have the
// same number, the count is the same either way" (ages-scoring.md,
// "Guilds"; rules-reference section 8 item 13). Seat 0 owns three guilds,
// each facing a tie: Moneylenders with 9 coins each (3 points), Builders
// with 2 wonders each (4 points), Shipowners with 2 brown or grey cards each,
// counted per city (2 points). Purple 9; the guilds are seat 0's alone.
func TestAuditR3WonderGuildTiesCountTheSharedNumber(t *testing.T) {
	s := enginetest.New(t, auditR3WonderSeed).
		Wonders(0, "pyramids", "colossus").
		Wonders(1, "sphinx", "piraeus").
		Built(0, "pyramids", "").Built(0, "colossus", "").
		Built(1, "sphinx", "").Built(1, "piraeus", "").
		City(0, "moneylenders-guild", "builders-guild", "shipowners-guild", "lumber-yard", "quarry").
		City(1, "glassworks", "press").
		Age(3).Coins(0, 9).Coins(1, 9).Pawn(0).
		State()

	sc := engine.Scores(&s)
	if sc[0].Purple != 9 || sc[1].Purple != 0 {
		t.Fatalf("purple points %d and %d, want 9 (3 + 4 + 2) and 0", sc[0].Purple, sc[1].Purple)
	}
}

// --- Mirror direction: seat 1 builds ----------------------------------------

// TestAuditR3WonderSeat1CircusMaximusLootsSeat0ThenDestroys: the same rules
// as for seat 0, in the other direction. The wonder's shield moves the pawn
// before the choice (wonders-tokens.md, "The order of a wonder's effects");
// a looting token at -3 is taken from seat 0 and a loss stops at 0
// (military-science.md, looting; rules-reference section 8 item 11); the
// target is a grey card in the opponent's city. Seat 1 builds Circus
// Maximus from -2: the pawn reaches -3, seat 0 loses its only coin, then
// seat 1 destroys seat 0's Press (seat 1's own Glassworks is not a target),
// and the turn passes to seat 0.
func TestAuditR3WonderSeat1CircusMaximusLootsSeat0ThenDestroys(t *testing.T) {
	s := enginetest.New(t, auditR3WonderSeed).
		Wonders(1, "circus-maximus").
		City(0, "press").
		City(1, "glassworks").
		Age(1).Accessible("altar").
		Coins(0, 1).Coins(1, 40).Pawn(-2).ToMove(1).
		State()

	enginetest.Do(t, &s, "wonder:circus-maximus:altar")
	if s.Military != -3 || s.Players[0].Coins != 0 {
		t.Fatalf("pawn %d, seat 0 coins %d; want -3 and 0 (looted, floored)", s.Military, s.Players[0].Coins)
	}
	if s.Phase != engine.PhaseDestroyCard || s.ToMove != 1 {
		t.Fatalf("phase %s, to move %d; want seat 1's destroy choice", s.Phase, s.ToMove)
	}
	if got, want := enginetest.Keys(&s), []string{"destroy:press"}; !slices.Equal(got, want) {
		t.Fatalf("legal %v, want %v", got, want)
	}
	enginetest.Do(t, &s, "destroy:press")
	if s.Players[0].City.Has(auditR3WonderCard(t, "press")) || !s.Discard.Has(auditR3WonderCard(t, "press")) {
		t.Fatal("the Press did not move from seat 0's city to the discard pile")
	}
	if s.Phase != engine.PhasePlay || s.ToMove != 0 {
		t.Fatalf("phase %s, to move %d; want seat 0 to play", s.Phase, s.ToMove)
	}
}

// TestAuditR3WonderSeat1MausoleumRevivesTacticiansCountingSeat0: the pool
// holds cards "from either player", the revived card enters the builder's
// city, and a revived guild pays its coins (wonders-tokens.md, "The
// Mausoleum"); the Tacticians Guild pays 1 coin per red card in the city
// with the most red cards (ages-scoring.md, "Guilds"). Seat 1 revives the
// Tacticians Guild while seat 0 has 3 red cards and seat 1 has 1: seat 1
// gets 3 coins, and the guild is in seat 1's city.
func TestAuditR3WonderSeat1MausoleumRevivesTacticiansCountingSeat0(t *testing.T) {
	s := enginetest.New(t, auditR3WonderSeed).
		Wonders(1, "mausoleum").
		City(0, "stable", "garrison", "palisade").
		City(1, "guard-tower").
		Discard("tacticians-guild").
		Age(3).Accessible("obelisk").
		Coins(1, 60).Pawn(0).ToMove(1).
		State()

	enginetest.Do(t, &s, "wonder:mausoleum:obelisk")
	if s.Phase != engine.PhaseBuildFromDiscard || s.ToMove != 1 {
		t.Fatalf("phase %s, to move %d; want seat 1's rebuild choice", s.Phase, s.ToMove)
	}
	ev := enginetest.Do(t, &s, "revive:tacticians-guild")
	g := auditR3WonderCard(t, "tacticians-guild")
	if got, ok := auditR3WonderCardCoins(ev, 1, g); !ok || got != 3 {
		t.Fatalf("revived Tacticians Guild paid seat 1 %d (found %v), want 3; events %+v", got, ok, ev)
	}
	if !s.Players[1].City.Has(g) {
		t.Fatal("the guild is not in seat 1's city")
	}
	if sc := engine.Scores(&s); sc[1].Purple != 3 || sc[0].Purple != 0 {
		t.Fatalf("purple points %d and %d, want 0 and 3", sc[0].Purple, sc[1].Purple)
	}
}
