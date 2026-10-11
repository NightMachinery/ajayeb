package engine_test

// Rules audit M10.R1, round 2, lens: economy (file prefix auditR3Econ).
//
// Black-box tests of costs, trading, flexible production, chains, discounts,
// Economy and Urbanism. They aim at what the earlier economy audits
// (edgecases_econ_r1_test.go and audit_econ_r2_test.go) did not cover: a
// building's or a wonder's own coins that come too late to pay for it, a
// fixed price competing with flexible producers, both discount tokens owned
// at once, the opponent's tokens, a destroy that removes two units of
// production, a destroyed producer revived by its victim, a tucked yellow
// card, a revived Reserve used in the Theology extra turn, the Economy token
// taken from the Great Library, the Moneylenders Guild after the last
// card's Economy income, and the view's cost for the seat not to move.
// Every expected value is derived from docs/rules/, never from what the
// engine currently does.

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// auditR3EconCard looks up a building by slug and fails the test on a typo.
func auditR3EconCard(t *testing.T, slug string) catalog.CardID {
	t.Helper()
	id, ok := catalog.CardBySlug(slug)
	if !ok {
		t.Fatalf("auditR3EconCard: unknown card %q", slug)
	}
	return id
}

// auditR3EconWonder looks up a wonder by slug and fails the test on a typo.
func auditR3EconWonder(t *testing.T, slug string) catalog.WonderID {
	t.Helper()
	id, ok := catalog.WonderBySlug(slug)
	if !ok {
		t.Fatalf("auditR3EconWonder: unknown wonder %q", slug)
	}
	return id
}

// auditR3EconToken looks up a progress token by slug.
func auditR3EconToken(t *testing.T, slug string) catalog.TokenID {
	t.Helper()
	id, ok := catalog.TokenBySlug(slug)
	if !ok {
		t.Fatalf("auditR3EconToken: unknown token %q", slug)
	}
	return id
}

// auditR3EconFirst returns the index of the first event of this kind for
// seat, or -1.
func auditR3EconFirst(ev []engine.Event, kind engine.EventKind, seat engine.Seat) int {
	for i, e := range ev {
		if e.Kind == kind && e.Seat == seat {
			return i
		}
	}
	return -1
}

// auditR3EconDiscounted sums the units a payment's discount waived.
func auditR3EconDiscounted(p engine.Payment) int {
	n := 0
	for _, d := range p.Discount {
		n += int(d)
	}
	return n
}

// auditR3EconSlotCosts returns both seats' costs that the view shows for
// the accessible card with this slug.
func auditR3EconSlotCosts(t *testing.T, v *view.View, slug string) *[2]view.CostView {
	t.Helper()
	for i := range v.Structure {
		sl := &v.Structure[i]
		if sl.Card != nil && *sl.Card == slug {
			if sl.Costs == nil {
				t.Fatalf("the view shows no costs for %s", slug)
			}
			return sl.Costs
		}
	}
	t.Fatalf("the view's structure has no face-up %s", slug)
	return nil
}

// --- A card's or a wonder's own coins come after its payment ----------------

// TestAuditR3EconCardCoinsCannotPayForTheCard: turn.md "1. Construct the
// building": you may construct a card only if you can pay its whole cost
// now; decisions.md "Effect order": the payment comes first, the card's coin
// rewards after it. The Port (W G P) costs 6 coins with no production on
// either side, and pays 2 coins per brown card of its owner's city
// (economy.md "Coin rewards"): 3 brown cards make 6. With 5 coins it is not
// affordable even though it would pay 6; with exactly 6 the builder goes to
// 0, then receives 6.
func TestAuditR3EconCardCoinsCannotPayForTheCard(t *testing.T) {
	build := func(coins int) engine.State {
		return enginetest.New(t, 3).
			Age(3).
			City(0, "clay-pool", "quarry", "stone-pit").
			Accessible("port").
			Coins(0, coins).
			ToMove(0).
			State()
	}

	short := build(5)
	if r := enginetest.MustFail(t, &short, "construct:port"); r != engine.ReasonUnaffordable {
		t.Fatalf("Port with 5 coins: reason %v, want ReasonUnaffordable (its 6 coins of reward come after the payment)", r)
	}

	s := build(6)
	ev := enginetest.Do(t, &s, "construct:port")
	if s.Players[0].Coins != 6 {
		t.Fatalf("coins after the Port = %d, want 6 (6 - 6, then 2 per brown card)", s.Players[0].Coins)
	}
	pay, reward := auditR3EconFirst(ev, engine.EvPay, 0), auditR3EconFirst(ev, engine.EvCoins, 0)
	if pay < 0 || reward < 0 || pay > reward {
		t.Fatalf("events: EvPay at %d, EvCoins at %d; want the payment before the reward", pay, reward)
	}
	if ev[pay].A != 6 || ev[reward].A != 6 || ev[reward].B != 6 {
		t.Fatalf("paid %d, gained %d to %d; want 6, 6 and 6", ev[pay].A, ev[reward].A, ev[reward].B)
	}
}

// TestAuditR3EconWonderCoinsCannotPayForTheWonder: turn.md "3. Build a
// wonder" (pay the wonder's cost) and decisions.md "Effect order" (the
// payment, then the wonder's coins). The Temple of Artemis (W S G P) costs 8
// with no production; its 12 coins cannot help pay for it, so with 7 coins
// no Temple key is legal. With exactly 8 the builder ends at 0 + 12.
func TestAuditR3EconWonderCoinsCannotPayForTheWonder(t *testing.T) {
	build := func(coins int) engine.State {
		return enginetest.New(t, 3).
			Age(1).
			Wonders(0, "temple-of-artemis", "pyramids", "colossus", "sphinx").
			Accessible("lumber-yard").
			Coins(0, coins).
			ToMove(0).
			State()
	}

	short := build(7)
	if r := enginetest.MustFail(t, &short, "wonder:temple-of-artemis:lumber-yard"); r != engine.ReasonUnaffordable {
		t.Fatalf("Temple of Artemis with 7 coins: reason %v, want ReasonUnaffordable", r)
	}

	s := build(8)
	enginetest.Do(t, &s, "wonder:temple-of-artemis:lumber-yard")
	if s.Players[0].Coins != 12 {
		t.Fatalf("coins after the Temple of Artemis = %d, want 12 (8 - 8 + 12)", s.Players[0].Coins)
	}
	if s.ToMove != 0 {
		t.Fatalf("to move = %d, want 0 (the Temple of Artemis grants another turn)", s.ToMove)
	}
}

// --- Fixed prices against flexible producers ------------------------------

// TestAuditR3EconFlexibleProducersSkipTheFixedPriceUnits: economy.md "The
// cheapest payment" steps 4 and 5: prices are set first (1 coin for a
// resource with a fixed price, otherwise 2 plus the opponent's production),
// then each flexible producer covers the dearest unit it can make. Arsenal
// costs CCC WW. The Wood Reserve fixes wood at 1 although the opponent's
// Sawmill would make it 4; the opponent's Brickyard makes clay 4. The
// Caravansery and the Great Lighthouse must cover two clays, leaving one
// clay at 4 and two woods at 1: 6 coins, all to the opponent's Economy
// (economy.md "The Economy token", fixed prices included). Covering the
// woods instead would cost 12. The pawn starts at -3, so the Arsenal's 3
// shields return it to the centre without looting anyone.
func TestAuditR3EconFlexibleProducersSkipTheFixedPriceUnits(t *testing.T) {
	s := enginetest.New(t, 3).
		Age(3).
		Wonders(0, "great-lighthouse", "pyramids", "colossus", "sphinx").
		Built(0, "great-lighthouse", "").
		City(0, "wood-reserve", "caravansery").
		City(1, "sawmill", "brickyard").
		Tokens(1, "economy").
		Accessible("arsenal").
		Pawn(-3). // the Arsenal's 3 shields bring it back to the centre, looting nobody
		Coins(0, 10).
		Coins(1, 0).
		ToMove(0).
		State()

	pay := engine.BuildingPayment(&s, 0, auditR3EconCard(t, "arsenal"))
	if pay.UnitPrice[catalog.Wood] != 1 || pay.UnitPrice[catalog.Clay] != 4 {
		t.Fatalf("unit prices wood=%d clay=%d, want 1 and 4", pay.UnitPrice[catalog.Wood], pay.UnitPrice[catalog.Clay])
	}
	if pay.NChoice != 2 || pay.TradeUnits[catalog.Clay] != 1 || pay.TradeUnits[catalog.Wood] != 2 || pay.Total != 6 || pay.ToOpponent != 6 {
		t.Fatalf("Arsenal NChoice=%d clay bought=%d wood bought=%d Total=%d ToOpponent=%d, want 2/1/2/6/6",
			pay.NChoice, pay.TradeUnits[catalog.Clay], pay.TradeUnits[catalog.Wood], pay.Total, pay.ToOpponent)
	}
	enginetest.Do(t, &s, "construct:arsenal")
	if s.Players[0].Coins != 4 || s.Players[1].Coins != 6 {
		t.Fatalf("coins %d and %d, want 4 and 6", s.Players[0].Coins, s.Players[1].Coins)
	}
}

// TestAuditR3EconFourWoodProducersAtTheCoinBoundary: economy.md "Trading":
// each unit costs 2 plus every unit of that resource the opponent's brown
// and grey cards produce, a 2-unit card counting as 2. The opponent's Lumber
// Yard, Logging Camp and Sawmill make 4 wood, so wood costs 6; the Siege
// Workshop (WWW G) costs 18 + 2 = 20. It is legal with exactly 20 coins and
// not with 19 (economy.md "The cheapest payment" step 7).
func TestAuditR3EconFourWoodProducersAtTheCoinBoundary(t *testing.T) {
	build := func(coins int) engine.State {
		return enginetest.New(t, 3).
			Age(3).
			City(1, "lumber-yard", "logging-camp", "sawmill").
			Accessible("siege-workshop").
			Coins(0, coins).
			ToMove(0).
			State()
	}
	short := build(19)
	if got := engine.TradePrices(&short, 0)[catalog.Wood]; got != 6 {
		t.Fatalf("wood price = %d, want 6 (2 + 1 + 1 + 2)", got)
	}
	if r := enginetest.MustFail(t, &short, "construct:siege-workshop"); r != engine.ReasonUnaffordable {
		t.Fatalf("Siege Workshop with 19 coins: reason %v, want ReasonUnaffordable", r)
	}
	s := build(20)
	enginetest.Do(t, &s, "construct:siege-workshop")
	if s.Players[0].Coins != 0 {
		t.Fatalf("coins after the Siege Workshop = %d, want 0", s.Players[0].Coins)
	}
}

// --- Discount tokens and whose tokens count --------------------------------

// TestAuditR3EconMasonryAndArchitectureNeverStack: economy.md "Discounts":
// Masonry waives 2 units of every blue card and Architecture 2 units of
// every wonder. Owning both gives 2 on each, never 4, and neither applies to
// the other's kind or to a red card. With no production anywhere every unit
// costs 2: the Palace (C S W GG) is 10 - 4 = 6 by Masonry, the Pyramids
// (SSS P) 8 - 4 = 4 by Architecture, the Arsenal (CCC WW) stays 10. With
// Masonry alone the Pyramids stay at 8.
func TestAuditR3EconMasonryAndArchitectureNeverStack(t *testing.T) {
	s := enginetest.New(t, 3).
		Age(3).
		Wonders(0, "pyramids", "colossus", "sphinx", "mausoleum").
		Tokens(0, "masonry", "architecture").
		ToMove(0).
		State()

	palace := engine.BuildingPayment(&s, 0, auditR3EconCard(t, "palace"))
	if palace.Total != 6 || auditR3EconDiscounted(palace) != 2 || palace.DiscountBy != auditR3EconToken(t, "masonry") {
		t.Errorf("Palace Total=%d waived=%d DiscountBy=%v, want 6/2/masonry", palace.Total, auditR3EconDiscounted(palace), palace.DiscountBy)
	}
	pyr := engine.WonderPayment(&s, 0, auditR3EconWonder(t, "pyramids"))
	if pyr.Total != 4 || auditR3EconDiscounted(pyr) != 2 || pyr.DiscountBy != auditR3EconToken(t, "architecture") {
		t.Errorf("Pyramids Total=%d waived=%d DiscountBy=%v, want 4/2/architecture", pyr.Total, auditR3EconDiscounted(pyr), pyr.DiscountBy)
	}
	if ars := engine.BuildingPayment(&s, 0, auditR3EconCard(t, "arsenal")); ars.Total != 10 || auditR3EconDiscounted(ars) != 0 {
		t.Errorf("Arsenal Total=%d waived=%d, want 10/0 (a red card)", ars.Total, auditR3EconDiscounted(ars))
	}

	m := enginetest.New(t, 3).
		Age(3).
		Wonders(0, "pyramids", "colossus", "sphinx", "mausoleum").
		Tokens(0, "masonry").
		ToMove(0).
		State()
	if p := engine.WonderPayment(&m, 0, auditR3EconWonder(t, "pyramids")); p.Total != 8 || auditR3EconDiscounted(p) != 0 {
		t.Errorf("Pyramids with Masonry only: Total=%d waived=%d, want 8/0 (Masonry is for blue cards)", p.Total, auditR3EconDiscounted(p))
	}
}

// TestAuditR3EconOpponentsTokensNeverApplyToYou: wonders-tokens.md
// "Progress tokens" and economy.md: Masonry, Architecture and Urbanism
// change their owner's constructions only. Seat 1 owns all three. Seat 0
// pays full price for the Palace (10) and the Pyramids (8), and its
// chain-free Pantheon (the Temple's sun) pays nobody Urbanism's 4 coins:
// neither seat 0, who lacks the token, nor seat 1, who did not build.
func TestAuditR3EconOpponentsTokensNeverApplyToYou(t *testing.T) {
	s := enginetest.New(t, 3).
		Age(3).
		Wonders(0, "pyramids", "colossus", "sphinx", "mausoleum").
		City(0, "temple").
		Tokens(1, "masonry", "architecture", "urbanism").
		Accessible("pantheon", "palace").
		Coins(0, 10).
		Coins(1, 7).
		ToMove(0).
		State()

	if p := engine.BuildingPayment(&s, 0, auditR3EconCard(t, "palace")); p.Total != 10 || p.DiscountBy != 0 {
		t.Errorf("seat 0's Palace Total=%d DiscountBy=%v, want 10/none", p.Total, p.DiscountBy)
	}
	if p := engine.WonderPayment(&s, 0, auditR3EconWonder(t, "pyramids")); p.Total != 8 || p.DiscountBy != 0 {
		t.Errorf("seat 0's Pyramids Total=%d DiscountBy=%v, want 8/none", p.Total, p.DiscountBy)
	}
	p := engine.BuildingPayment(&s, 0, auditR3EconCard(t, "pantheon"))
	if !p.ChainFree || p.UrbanismBonus != 0 {
		t.Fatalf("seat 0's Pantheon ChainFree=%v UrbanismBonus=%d, want true/0", p.ChainFree, p.UrbanismBonus)
	}
	enginetest.Do(t, &s, "construct:pantheon")
	if s.Players[0].Coins != 10 || s.Players[1].Coins != 7 {
		t.Fatalf("coins %d and %d after the chained Pantheon, want 10 and 7 (no Urbanism for anyone)", s.Players[0].Coins, s.Players[1].Coins)
	}
}

// TestAuditR3EconOwnEconomyRefundsNothing: economy.md "The Economy token":
// the coins your OPPONENT pays for resources come to you. Your own trading
// still goes to the bank. Seat 0 owns Economy and buys two stones at 4 (the
// opponent's Shelf Quarry) for the Walls: it loses 8 and seat 1 gains
// nothing. Then seat 1 buys clay and wood for the Horse Breeders (2 + 2):
// those 4 coins go to seat 0.
func TestAuditR3EconOwnEconomyRefundsNothing(t *testing.T) {
	s := enginetest.New(t, 3).
		Age(2).
		Tokens(0, "economy").
		City(1, "shelf-quarry").
		Accessible("walls", "horse-breeders").
		Coins(0, 10).
		Coins(1, 5).
		ToMove(0).
		State()

	if p := engine.BuildingPayment(&s, 0, auditR3EconCard(t, "walls")); p.Total != 8 || p.ToOpponent != 0 {
		t.Fatalf("seat 0's Walls Total=%d ToOpponent=%d, want 8/0", p.Total, p.ToOpponent)
	}
	enginetest.Do(t, &s, "construct:walls")
	if s.Players[0].Coins != 2 || s.Players[1].Coins != 5 {
		t.Fatalf("coins %d and %d after seat 0's Walls, want 2 and 5", s.Players[0].Coins, s.Players[1].Coins)
	}
	enginetest.Do(t, &s, "construct:horse-breeders")
	if s.Players[0].Coins != 6 || s.Players[1].Coins != 1 {
		t.Fatalf("coins %d and %d after seat 1's Horse Breeders, want 6 and 1", s.Players[0].Coins, s.Players[1].Coins)
	}
}

// TestAuditR3EconUrbanismChainBuildGivesEconomyNothing: economy.md "Chains"
// (a chained card ignores both its coins and its resources) and "The
// Economy token" (only coins paid for resources go to the owner). The
// chained Horse Breeders buy nothing, so the opponent's Economy receives
// nothing and no payment is reported, while Urbanism pays its owner 4 from
// the bank (wonders-tokens.md "Urbanism").
func TestAuditR3EconUrbanismChainBuildGivesEconomyNothing(t *testing.T) {
	s := enginetest.New(t, 3).
		Age(2).
		City(0, "stable").
		Tokens(0, "urbanism").
		City(1, "brickyard", "sawmill").
		Tokens(1, "economy").
		Accessible("horse-breeders").
		Coins(0, 0).
		Coins(1, 0).
		ToMove(0).
		State()

	ev := enginetest.Do(t, &s, "construct:horse-breeders")
	if s.Players[0].Coins != 4 || s.Players[1].Coins != 0 {
		t.Fatalf("coins %d and %d, want 4 and 0", s.Players[0].Coins, s.Players[1].Coins)
	}
	if i := auditR3EconFirst(ev, engine.EvPay, 0); i >= 0 {
		t.Fatalf("a chain build reported a payment: %+v", ev[i])
	}
}

// --- Destroyed producers ----------------------------------------------------

// TestAuditR3EconZeusOnATwoUnitProducerLowersThePriceByTwo: economy.md
// "Trading" (a card that produces 2 units counts as 2; prices fall at once
// when the Statue of Zeus destroys a producer). The opponent's Clay Pool,
// Clay Pit and Brickyard make 4 clay: clay costs 6, and the Statue of Zeus
// (PP C W S) costs 2 + 2 + 6 + 2 + 2 = 14. Destroying the Brickyard leaves
// 2 clay: the builder's price is 4, not 2 or 5. The victim's own clay
// production drops to 2 as well, so its Arsenal (CCC WW) covers 2 clays
// itself and buys the third.
func TestAuditR3EconZeusOnATwoUnitProducerLowersThePriceByTwo(t *testing.T) {
	s := enginetest.New(t, 3).
		Age(2).
		Wonders(0, "statue-of-zeus", "pyramids", "colossus", "sphinx").
		City(1, "clay-pool", "clay-pit", "brickyard").
		Accessible("sawmill").
		Coins(0, 20).
		ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:statue-of-zeus:sawmill")
	if s.Players[0].Coins != 6 {
		t.Fatalf("coins after the Statue of Zeus = %d, want 6 (20 - 14)", s.Players[0].Coins)
	}
	enginetest.Do(t, &s, "destroy:brickyard")
	if got := engine.TradePrices(&s, 0)[catalog.Clay]; got != 4 {
		t.Fatalf("clay price after destroying the Brickyard = %d, want 4 (two clay producers are left)", got)
	}
	ars := engine.BuildingPayment(&s, 1, auditR3EconCard(t, "arsenal"))
	if ars.Own[catalog.Clay] != 2 || ars.TradeUnits[catalog.Clay] != 1 {
		t.Fatalf("victim's Arsenal: own clay %d, clay bought %d, want 2 and 1", ars.Own[catalog.Clay], ars.TradeUnits[catalog.Clay])
	}
}

// TestAuditR3EconVictimRevivesItsDestroyedProducer: a destroyed card goes to
// the discard pile (rules reference section 8 item 8), so the Mausoleum can
// bring it back, with every effect of the building (wonders-tokens.md "The
// Mausoleum"). Seat 0's Statue of Zeus destroys seat 1's Brickyard. Seat 1
// then pays for its Mausoleum (CC GG P) without its clay: 5 units at 2 = 10
// (it would have been 6 with the Brickyard). It revives the Brickyard, which
// makes clay again: seat 0's clay price is back to 4 at once (economy.md
// "Trading": prices rise when the opponent's production rises).
func TestAuditR3EconVictimRevivesItsDestroyedProducer(t *testing.T) {
	s := enginetest.New(t, 3).
		Age(2).
		Wonders(0, "statue-of-zeus", "pyramids", "colossus", "sphinx").
		Wonders(1, "mausoleum", "appian-way", "piraeus", "great-library").
		City(1, "brickyard").
		Accessible("sawmill", "glass-blower").
		Coins(0, 20).
		Coins(1, 10).
		ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:statue-of-zeus:sawmill")
	if s.Players[0].Coins != 8 {
		t.Fatalf("coins after the Statue of Zeus = %d, want 8 (20 - 12, clay at 4)", s.Players[0].Coins)
	}
	enginetest.Do(t, &s, "destroy:brickyard")
	if s.ToMove != 1 {
		t.Fatalf("to move = %d, want 1", s.ToMove)
	}
	maus := engine.WonderPayment(&s, 1, auditR3EconWonder(t, "mausoleum"))
	if maus.Own[catalog.Clay] != 0 || maus.Total != 10 || !maus.Affordable {
		t.Fatalf("Mausoleum after the destroy: own clay %d, Total %d, Affordable %v, want 0/10/true", maus.Own[catalog.Clay], maus.Total, maus.Affordable)
	}
	enginetest.Do(t, &s, "wonder:mausoleum:glass-blower", "revive:brickyard")
	if s.Players[1].Coins != 0 {
		t.Fatalf("seat 1's coins = %d, want 0 (10 - 10; a revive is free)", s.Players[1].Coins)
	}
	if got := engine.TradePrices(&s, 0)[catalog.Clay]; got != 4 {
		t.Fatalf("seat 0's clay price after the revive = %d, want 4", got)
	}
	if p := engine.BuildingPayment(&s, 1, auditR3EconCard(t, "statue")); p.Own[catalog.Clay] != 2 || p.Total != 0 {
		t.Fatalf("seat 1's Statue (CC) after the revive: own clay %d, Total %d, want 2/0", p.Own[catalog.Clay], p.Total)
	}
}

// TestAuditR3EconTheologyZeusExtraTurnBuysAtTheLoweredPrice: wonders-tokens.md
// "The order of a wonder's effects" (pay, then the destroy) and economy.md
// "Trading" (the price falls at once), with Theology's extra turn after the
// destroy and the opponent's Economy on both payments. Seat 1 owns a
// Brickyard (clay 4) and Economy. The Statue of Zeus (PP C W S) costs 12,
// all to seat 1. After the destroy, the extra turn's Dispensary (CC S)
// costs 2 + 2 + 2 = 6, again to seat 1, and the view's construct choice
// shows that price (it would be 10 at the old price).
func TestAuditR3EconTheologyZeusExtraTurnBuysAtTheLoweredPrice(t *testing.T) {
	s := enginetest.New(t, 3).
		Age(2).
		Wonders(0, "statue-of-zeus", "pyramids", "colossus", "sphinx").
		Tokens(0, "theology").
		City(1, "brickyard").
		Tokens(1, "economy").
		Accessible("sawmill", "dispensary").
		Coins(0, 20).
		Coins(1, 0).
		ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:statue-of-zeus:sawmill", "destroy:brickyard")
	if s.Players[0].Coins != 8 || s.Players[1].Coins != 12 {
		t.Fatalf("coins %d and %d after the Statue of Zeus, want 8 and 12", s.Players[0].Coins, s.Players[1].Coins)
	}
	if s.ToMove != 0 {
		t.Fatalf("to move = %d, want 0 (Theology's extra turn)", s.ToMove)
	}
	v := engine.ViewFor(&s, engine.SeatViewer(0))
	var cost *view.CostView
	for i := range v.Choices {
		if v.Choices[i].Key == "construct:dispensary" {
			cost = v.Choices[i].Cost
		}
	}
	if cost == nil || cost.Total != 6 || cost.ToOpponent != 6 {
		t.Fatalf("view's Dispensary cost %+v, want Total 6 and ToOpponent 6", cost)
	}
	enginetest.Do(t, &s, "construct:dispensary")
	if s.Players[0].Coins != 2 || s.Players[1].Coins != 18 {
		t.Fatalf("coins %d and %d after the Dispensary, want 2 and 18", s.Players[0].Coins, s.Players[1].Coins)
	}
}

// --- Tucked and revived economy cards ---------------------------------------

// TestAuditR3EconTuckedYellowCardIsNotInTheCity: turn.md "3. Build a
// wonder": the card tucked under a wonder has no effect of any kind, and
// turn.md "2. Discard it for coins": 2 coins plus 1 per yellow card in your
// city. A Forum tucked under the Sphinx neither produces glass or papyrus
// nor counts as a yellow card. In the Sphinx's extra turn the Laboratory
// (W GG) still costs 6, and discarding gives 2 + 1 (the Tavern) = 3.
func TestAuditR3EconTuckedYellowCardIsNotInTheCity(t *testing.T) {
	s := enginetest.New(t, 3).
		Age(2).
		Wonders(0, "sphinx", "pyramids", "colossus", "mausoleum").
		City(0, "tavern").
		Accessible("forum", "laboratory").
		Coins(0, 20).
		ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:sphinx:forum")
	if s.Players[0].Coins != 12 || s.ToMove != 0 {
		t.Fatalf("after the Sphinx: coins %d, to move %d, want 12 and 0", s.Players[0].Coins, s.ToMove)
	}
	if p := engine.BuildingPayment(&s, 0, auditR3EconCard(t, "laboratory")); p.NChoice != 0 || p.Total != 6 {
		t.Errorf("Laboratory NChoice=%d Total=%d, want 0/6 (a tucked Forum produces nothing)", p.NChoice, p.Total)
	}
	if got := engine.DiscardValue(&s, 0); got != 3 {
		t.Errorf("discard value = %d, want 3 (the tucked Forum is not a yellow card of the city)", got)
	}
	enginetest.Do(t, &s, "discard:laboratory")
	if s.Players[0].Coins != 15 {
		t.Fatalf("coins after the discard = %d, want 15 (12 + 3)", s.Players[0].Coins)
	}
}

// TestAuditR3EconRevivedReserveFixesThePriceInTheTheologyExtraTurn:
// decisions.md "The fixed-price delay" (a fixed price takes effect at once)
// and wonders-tokens.md "The Mausoleum" (the revived card applies all its
// effects). Seat 0 pays 10 for the Mausoleum (CC GG P), revives the Wood
// Reserve, and in Theology's extra turn buys the Siege Workshop (WWW G) with
// wood at 1 instead of 4 (the opponent's Sawmill): 3 + 2 = 5, exactly the
// coins left.
func TestAuditR3EconRevivedReserveFixesThePriceInTheTheologyExtraTurn(t *testing.T) {
	s := enginetest.New(t, 3).
		Age(3).
		Wonders(0, "mausoleum", "pyramids", "colossus", "sphinx").
		Tokens(0, "theology").
		City(1, "sawmill").
		Discard("wood-reserve").
		Accessible("siege-workshop", "palace").
		Coins(0, 15).
		ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:mausoleum:palace", "revive:wood-reserve")
	if s.Players[0].Coins != 5 || s.ToMove != 0 {
		t.Fatalf("after the Mausoleum: coins %d, to move %d, want 5 and 0", s.Players[0].Coins, s.ToMove)
	}
	p := engine.BuildingPayment(&s, 0, auditR3EconCard(t, "siege-workshop"))
	if !p.FixedPrice.Has(catalog.Wood) || p.Total != 5 || !p.Affordable {
		t.Fatalf("Siege Workshop fixed wood=%v Total=%d Affordable=%v, want true/5/true", p.FixedPrice.Has(catalog.Wood), p.Total, p.Affordable)
	}
	enginetest.Do(t, &s, "construct:siege-workshop")
	if s.Players[0].Coins != 0 {
		t.Fatalf("coins after the Siege Workshop = %d, want 0", s.Players[0].Coins)
	}
}

// --- Economy gained mid-game, and Economy at the end of the game -----------

// TestAuditR3EconEconomyFromTheGreatLibraryIsNotRetroactive:
// wonders-tokens.md "Progress tokens" (a lasting effect applies to what
// happens afterwards) and "The Great Library" (the chosen token is the
// player's). Seat 0 pays 10 for the Great Library (WWW G P) to the bank and
// keeps Economy: no refund. Seat 1's next purchase, the Stable (W) at 2,
// goes to seat 0.
func TestAuditR3EconEconomyFromTheGreatLibraryIsNotRetroactive(t *testing.T) {
	s := enginetest.New(t, 3).
		Age(1).
		Wonders(0, "great-library", "pyramids", "colossus", "sphinx").
		BoardTokens("agriculture", "law", "mathematics", "philosophy", "strategy").
		BoxOrder("economy", "masonry", "theology").
		Accessible("lumber-yard", "stable").
		Coins(0, 20).
		Coins(1, 7).
		ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:great-library:lumber-yard", "token:economy")
	if s.Players[0].Coins != 10 || s.Players[1].Coins != 7 {
		t.Fatalf("coins %d and %d after the Great Library, want 10 and 7", s.Players[0].Coins, s.Players[1].Coins)
	}
	if s.ToMove != 1 {
		t.Fatalf("to move = %d, want 1", s.ToMove)
	}
	enginetest.Do(t, &s, "construct:stable")
	if s.Players[0].Coins != 12 || s.Players[1].Coins != 5 {
		t.Fatalf("coins %d and %d after seat 1's Stable, want 12 and 5", s.Players[0].Coins, s.Players[1].Coins)
	}
}

// TestAuditR3EconMoneylendersCountsTheLastCardsEconomyIncome: ages-scoring.md
// "Guilds": the Moneylenders Guild scores 1 point per full 3 coins held by
// the richer player at the end; decisions.md "Effect order": the Economy
// income arrives with the payment, before the end of the game. Both seats
// have 12 coins. Seat 0 buys the last card of Age III, the Obelisk (SS G),
// for 6, all to seat 1: 6 and 18 coins. The guild scores 18 / 3 = 6 points
// (not 4 from the coins before the payment), and the coin points are 2 and
// 6.
func TestAuditR3EconMoneylendersCountsTheLastCardsEconomyIncome(t *testing.T) {
	s := enginetest.New(t, 3).
		Age(3).
		City(0, "moneylenders-guild").
		Tokens(1, "economy").
		OnlyLeft("obelisk").
		Coins(0, 12).
		Coins(1, 12).
		ToMove(0).
		State()

	enginetest.Do(t, &s, "construct:obelisk")
	if !s.Over() || s.Result.Kind != engine.VictoryCivilian {
		t.Fatalf("result %+v, want the civilian end of the game", s.Result)
	}
	if s.Players[0].Coins != 6 || s.Players[1].Coins != 18 {
		t.Fatalf("coins %d and %d, want 6 and 18", s.Players[0].Coins, s.Players[1].Coins)
	}
	sc := s.Result.Scores
	if sc[0].Purple != 6 {
		t.Errorf("Moneylenders Guild points = %d, want 6 (the richer player's 18 coins)", sc[0].Purple)
	}
	if sc[0].Coins != 2 || sc[1].Coins != 6 {
		t.Errorf("coin points %d and %d, want 2 and 6", sc[0].Coins, sc[1].Coins)
	}
}

// --- The view for the seat not to move --------------------------------------

// TestAuditR3EconViewCostsUseEachSeatsOwnTokens: economy.md "For developers":
// the view carries both seats' breakdowns for every accessible card and the
// cost of each unbuilt wonder for its owner. Each must use that seat's own
// tokens and its opponent's Economy. Seat 0 (to move) owns Economy; seat 1
// owns Masonry and Architecture. With no production anywhere, the Palace
// costs seat 0 10 (no discount, nothing to seat 1) and seat 1 6 (Masonry,
// all 6 to seat 0); seat 1's Pyramids cost 4 (Architecture, all to seat 0).
func TestAuditR3EconViewCostsUseEachSeatsOwnTokens(t *testing.T) {
	s := enginetest.New(t, 3).
		Age(3).
		Wonders(1, "pyramids", "colossus", "sphinx", "mausoleum").
		Tokens(0, "economy").
		Tokens(1, "masonry", "architecture").
		Accessible("palace").
		Coins(0, 7).
		Coins(1, 7).
		ToMove(0).
		State()

	for _, viewer := range []engine.Seat{0, 1} {
		v := engine.ViewFor(&s, engine.SeatViewer(viewer))
		costs := auditR3EconSlotCosts(t, &v, "palace")
		c0, c1 := costs[0], costs[1]
		if c0.Total != 10 || c0.Discount != nil || c0.ToOpponent != 0 {
			t.Errorf("viewer %d: seat 0's Palace Total=%d Discount=%+v ToOpponent=%d, want 10/none/0", viewer, c0.Total, c0.Discount, c0.ToOpponent)
		}
		if c1.Total != 6 || c1.Discount == nil || c1.Discount.Token != "masonry" || c1.ToOpponent != 6 || !c1.Affordable {
			t.Errorf("viewer %d: seat 1's Palace Total=%d Discount=%+v ToOpponent=%d Affordable=%v, want 6/masonry/6/true", viewer, c1.Total, c1.Discount, c1.ToOpponent, c1.Affordable)
		}
		var pyr *view.CostView
		for _, w := range v.Players[1].Wonders {
			if w.Wonder == "pyramids" {
				pyr = w.Cost
			}
		}
		if pyr == nil || pyr.Total != 4 || pyr.Discount == nil || pyr.Discount.Token != "architecture" || pyr.ToOpponent != 4 {
			t.Errorf("viewer %d: seat 1's Pyramids cost %+v, want Total 4, Architecture, ToOpponent 4", viewer, pyr)
		}
	}
}

// --- Coin rewards inside extra turns ----------------------------------------

// TestAuditR3EconRevivedTavernPaysThenCountsForTheDiscard: wonders-tokens.md
// "The Mausoleum" (a revived card applies all its effects, coins included;
// economy.md "Coin rewards": that includes a construction through the
// Mausoleum) and turn.md "2. Discard it for coins" (2 plus 1 per yellow card
// in your city). Seat 0 pays exactly its 10 coins for the Mausoleum
// (CC GG P), revives the Tavern for its 4 coins, and in Theology's extra
// turn discards for 2 + 1 = 3: 7 coins in the end.
func TestAuditR3EconRevivedTavernPaysThenCountsForTheDiscard(t *testing.T) {
	s := enginetest.New(t, 3).
		Age(2).
		Wonders(0, "mausoleum", "pyramids", "colossus", "sphinx").
		Tokens(0, "theology").
		Discard("tavern").
		Accessible("sawmill", "brickyard").
		Coins(0, 10).
		ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:mausoleum:sawmill")
	if s.Players[0].Coins != 0 {
		t.Fatalf("coins after the Mausoleum = %d, want 0", s.Players[0].Coins)
	}
	enginetest.Do(t, &s, "revive:tavern")
	if s.Players[0].Coins != 4 || s.ToMove != 0 {
		t.Fatalf("after reviving the Tavern: coins %d, to move %d, want 4 and 0", s.Players[0].Coins, s.ToMove)
	}
	if got := engine.DiscardValue(&s, 0); got != 3 {
		t.Fatalf("discard value = %d, want 3 (the revived Tavern is a yellow card of the city)", got)
	}
	enginetest.Do(t, &s, "discard:brickyard")
	if s.Players[0].Coins != 7 {
		t.Fatalf("coins after the discard = %d, want 7", s.Players[0].Coins)
	}
}

// TestAuditR3EconChainedArenaCountsTheWonderOfThisTurn: wonders-tokens.md
// "Urbanism" (4 coins for a chain construction), decisions.md "Effect
// order" (Urbanism's bonus, then the card's own coins) and economy.md "Coin
// rewards" (Arena: 2 coins per wonder you have built, counted when it is
// built). Seat 0 has built the Pyramids and builds the Sphinx (S C GG, 8
// coins) for an extra turn, in which the Arena is free through the
// Brewery's barrel. From 2 coins: +4 by Urbanism, then +4 for two wonders.
func TestAuditR3EconChainedArenaCountsTheWonderOfThisTurn(t *testing.T) {
	s := enginetest.New(t, 3).
		Age(3).
		Wonders(0, "sphinx", "pyramids", "colossus", "mausoleum").
		Built(0, "pyramids", "").
		City(0, "brewery").
		Tokens(0, "urbanism").
		Accessible("arena", "palace").
		Coins(0, 10).
		ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:sphinx:palace")
	if s.Players[0].Coins != 2 || s.ToMove != 0 {
		t.Fatalf("after the Sphinx: coins %d, to move %d, want 2 and 0", s.Players[0].Coins, s.ToMove)
	}
	ev := enginetest.Do(t, &s, "construct:arena")
	if s.Players[0].Coins != 10 {
		t.Fatalf("coins after the Arena = %d, want 10 (2 + 4 + 2 x 2)", s.Players[0].Coins)
	}
	var gains []int16
	for _, e := range ev {
		if e.Kind == engine.EvCoins && e.Seat == 0 {
			gains = append(gains, e.A)
		}
	}
	if len(gains) != 2 || gains[0] != 4 || gains[1] != 4 {
		t.Fatalf("coin gains %v, want [4 4] (Urbanism, then the Arena)", gains)
	}
	if i := auditR3EconFirst(ev, engine.EvPay, 0); i >= 0 {
		t.Fatalf("a chain build reported a payment: %+v", ev[i])
	}
}
