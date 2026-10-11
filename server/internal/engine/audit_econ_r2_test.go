package engine_test

// Rules audit M10.R1, round 1, lens: economy (file prefix auditR2Econ).
//
// Black-box tests of costs, trading, flexible production, chains, discounts,
// Economy and Urbanism, aimed at the interactions of two or three effects in
// one payment, at boundary values (exact coins, the last card of an age, the
// last progress token, the ninth shield) and at the order of resolution
// inside one action. The cases of edgecases_econ_r1_test.go are not
// repeated. Every expected value is derived from docs/rules/ (and the
// rules reference), never from what the engine currently does.

import (
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// auditR2EconCard looks up a building by slug and fails the test on a typo.
func auditR2EconCard(t *testing.T, slug string) catalog.CardID {
	t.Helper()
	id, ok := catalog.CardBySlug(slug)
	if !ok {
		t.Fatalf("auditR2EconCard: unknown card %q", slug)
	}
	return id
}

// auditR2EconWonder looks up a wonder by slug and fails the test on a typo.
func auditR2EconWonder(t *testing.T, slug string) catalog.WonderID {
	t.Helper()
	id, ok := catalog.WonderBySlug(slug)
	if !ok {
		t.Fatalf("auditR2EconWonder: unknown wonder %q", slug)
	}
	return id
}

// auditR2EconToken looks up a progress token by slug.
func auditR2EconToken(t *testing.T, slug string) catalog.TokenID {
	t.Helper()
	id, ok := catalog.TokenBySlug(slug)
	if !ok {
		t.Fatalf("auditR2EconToken: unknown token %q", slug)
	}
	return id
}

// auditR2EconCoinEvents returns the EvCoins events of seat, in order.
func auditR2EconCoinEvents(ev []engine.Event, seat engine.Seat) []engine.Event {
	var out []engine.Event
	for _, e := range ev {
		if e.Kind == engine.EvCoins && e.Seat == seat {
			out = append(out, e)
		}
	}
	return out
}

// --- Wonder payments: own production, tucked cards, flexible producers ----

// TestAuditR2EconGreatLighthouseDoesNotPayForItself: a wonder's production
// works "from now on", from the moment it is built (wonders-tokens.md "The
// twelve wonders" and "The order of a wonder's effects": the cost is paid in
// step 1, production starts in step 2). So the Great Lighthouse's own wood,
// clay or stone cannot cover one of its own cost units (PP S W): with no
// production on either side it costs 4 units at 2 coins, 8 coins.
func TestAuditR2EconGreatLighthouseDoesNotPayForItself(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(1).
		Wonders(0, "great-lighthouse", "pyramids", "colossus", "sphinx").
		Accessible("lumber-yard").
		Coins(0, 20).
		ToMove(0).
		State()

	pay := engine.WonderPayment(&s, 0, auditR2EconWonder(t, "great-lighthouse"))
	if pay.NChoice != 0 || pay.Total != 8 {
		t.Fatalf("Great Lighthouse payment NChoice=%d Total=%d, want 0/8 (its own production starts only after it is paid for)", pay.NChoice, pay.Total)
	}
	enginetest.Do(t, &s, "wonder:great-lighthouse:lumber-yard")
	if s.Players[0].Coins != 12 {
		t.Fatalf("coins after building the Great Lighthouse = %d, want 12 (20 - 8)", s.Players[0].Coins)
	}
	// From now on it covers a raw unit: Baths (S) becomes free.
	if p := engine.BuildingPayment(&s, 0, auditR2EconCard(t, "baths")); p.Total != 0 || p.NChoice != 1 {
		t.Fatalf("Baths after the Great Lighthouse: Total=%d NChoice=%d, want 0/1", p.Total, p.NChoice)
	}
}

// TestAuditR2EconTuckedBrownCardProducesNothing: turn.md "3. Build a wonder"
// says the tucked card has no effect of any kind (the r1 audit checked its
// chain symbol only). Tucking a Brickyard must neither give its builder 2
// clay nor raise the opponent's clay price (economy.md "Trading": the price
// counts the opponent's brown and grey production).
func TestAuditR2EconTuckedBrownCardProducesNothing(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(2).
		Wonders(0, "pyramids", "colossus", "sphinx", "mausoleum").
		Accessible("brickyard").
		Coins(0, 30).
		ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:pyramids:brickyard")
	if got := engine.TradePrices(&s, 1)[catalog.Clay]; got != 2 {
		t.Errorf("opponent's clay price = %d, want 2 (a tucked Brickyard produces nothing)", got)
	}
	statue := engine.BuildingPayment(&s, 0, auditR2EconCard(t, "statue")) // CC
	if statue.Own[catalog.Clay] != 0 || statue.Total != 4 {
		t.Errorf("Statue for the tucker: Own[clay]=%d Total=%d, want 0/4 (no clay from the tucked card)", statue.Own[catalog.Clay], statue.Total)
	}
}

// TestAuditR2EconFourFlexibleProducersPayAWholeWonderAtZeroCoins: economy.md
// "Flexible production" lists four flexible producers (Forum, Caravansery,
// the Great Lighthouse, Piraeus), each covering one unit per turn. With all
// four, the Temple of Artemis (W S G P) is fully covered, so it costs 0 and
// can be built with exactly 0 coins (economy.md "The cheapest payment",
// step 7: allowed when the total is no more than your coins). It then pays
// its 12 coins (wonders-tokens.md).
func TestAuditR2EconFourFlexibleProducersPayAWholeWonderAtZeroCoins(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(2).
		Wonders(0, "piraeus", "great-lighthouse", "temple-of-artemis", "sphinx").
		Built(0, "piraeus", "").
		Built(0, "great-lighthouse", "").
		City(0, "forum", "caravansery").
		City(1, "sawmill", "shelf-quarry", "glass-blower", "drying-room"). // high prices, so a missed unit shows
		Accessible("brickyard").
		Coins(0, 0).
		ToMove(0).
		State()

	pay := engine.WonderPayment(&s, 0, auditR2EconWonder(t, "temple-of-artemis"))
	if pay.NChoice != 4 || pay.Total != 0 || !pay.Affordable {
		t.Fatalf("Temple of Artemis: NChoice=%d Total=%d Affordable=%v, want 4/0/true", pay.NChoice, pay.Total, pay.Affordable)
	}
	enginetest.Do(t, &s, "wonder:temple-of-artemis:brickyard")
	if s.Players[0].Coins != 12 {
		t.Fatalf("coins = %d, want 12 (paid 0, then the Temple's 12)", s.Players[0].Coins)
	}
}

// TestAuditR2EconTwoManufacturedProducersCoverTwoGlass: economy.md "Flexible
// production": "Forum and Piraeus together can cover one glass and one
// papyrus, or two glass." Laboratory costs W GG, so only the wood is bought.
func TestAuditR2EconTwoManufacturedProducersCoverTwoGlass(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(2).
		Wonders(0, "piraeus", "pyramids", "colossus", "sphinx").
		Built(0, "piraeus", "").
		City(0, "forum").
		Accessible("laboratory").
		Coins(0, 10).
		ToMove(0).
		State()

	pay := engine.BuildingPayment(&s, 0, auditR2EconCard(t, "laboratory"))
	if pay.NChoice != 2 || pay.TradeUnits[catalog.Glass] != 0 || pay.Total != 2 {
		t.Fatalf("Laboratory: NChoice=%d glass bought=%d Total=%d, want 2/0/2 (Forum and Piraeus each cover a glass)", pay.NChoice, pay.TradeUnits[catalog.Glass], pay.Total)
	}
}

// TestAuditR2EconRawProducersCoverTheDearRawUnits: economy.md "The cheapest
// payment" step 5 (each flexible producer covers the dearest remaining unit
// it can make), on the raw side and with a wonder as a producer. Circus
// costs CC SS; the opponent's Brickyard makes clay cost 4, stone costs 2.
// The Caravansery and the Great Lighthouse must take both clays, leaving
// 2 stone at 2: 4 coins (covering the stones would leave 8).
func TestAuditR2EconRawProducersCoverTheDearRawUnits(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(3).
		Wonders(0, "great-lighthouse", "pyramids", "colossus", "sphinx").
		Built(0, "great-lighthouse", "").
		City(0, "caravansery").
		City(1, "brickyard").
		Accessible("circus").
		Coins(0, 20).
		ToMove(0).
		State()

	pay := engine.BuildingPayment(&s, 0, auditR2EconCard(t, "circus"))
	if pay.NChoice != 2 || pay.TradeUnits[catalog.Clay] != 0 || pay.Total != 4 {
		t.Fatalf("Circus: NChoice=%d clay bought=%d Total=%d, want 2/0/4", pay.NChoice, pay.TradeUnits[catalog.Clay], pay.Total)
	}
	before := s.Players[0].Coins
	enginetest.Do(t, &s, "construct:circus")
	if d := before - s.Players[0].Coins; d != 4 {
		t.Fatalf("coins spent = %d, want 4", d)
	}
}

// TestAuditR2EconOpponentWonderProducersNeverRaisePrices: economy.md
// "Trading": "Your opponent's flexible production (yellow cards and wonders)
// never raises your price"; wonders-tokens.md says the same of the Great
// Lighthouse and Piraeus. An opponent with all four flexible producers and
// no brown or grey card leaves every price at 2.
func TestAuditR2EconOpponentWonderProducersNeverRaisePrices(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(2).
		Wonders(1, "piraeus", "great-lighthouse", "pyramids", "sphinx").
		Built(1, "piraeus", "").
		Built(1, "great-lighthouse", "").
		City(1, "forum", "caravansery").
		ToMove(0).
		State()

	prices := engine.TradePrices(&s, 0)
	for r := range catalog.NumResources {
		if prices[r] != 2 {
			t.Errorf("price of resource %d = %d, want 2 (flexible production never counts)", r, prices[r])
		}
	}
}

// --- Fixed prices with discounts and Economy -----------------------------

// TestAuditR2EconFixedPriceArchitectureAndEconomyOnAWonder combines three
// rules in one wonder payment:
//   - economy.md "Fixed prices": Stone Reserve sets stone to 1 coin "whatever
//     your opponent produces" (the opponent's Shelf Quarry would make it 4);
//     the fixed price applies to every cost, wonders included;
//   - economy.md "Discounts": Architecture waives the 2 units that would cost
//     the most, here the papyrus (2) and one stone (1);
//   - economy.md "The Economy token": the opponent receives the trading part
//     "at the price actually paid, including the 1-coin fixed prices and
//     after any discount".
//
// Pyramids (SSS P): 2 stone left at 1 coin, 2 coins, all to the opponent.
func TestAuditR2EconFixedPriceArchitectureAndEconomyOnAWonder(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(2).
		Wonders(0, "pyramids", "colossus", "sphinx", "mausoleum").
		City(0, "stone-reserve").
		Tokens(0, "architecture").
		City(1, "shelf-quarry").
		Tokens(1, "economy").
		Accessible("sawmill").
		Coins(0, 10).
		Coins(1, 0).
		ToMove(0).
		State()

	pay := engine.WonderPayment(&s, 0, auditR2EconWonder(t, "pyramids"))
	if pay.UnitPrice[catalog.Stone] != 1 {
		t.Fatalf("stone unit price = %d, want 1 (Stone Reserve beats the opponent's production)", pay.UnitPrice[catalog.Stone])
	}
	if pay.Total != 2 || pay.ToOpponent != 2 {
		t.Fatalf("Pyramids Total=%d ToOpponent=%d, want 2/2 (papyrus and a stone waived, 2 stone bought at 1)", pay.Total, pay.ToOpponent)
	}
	enginetest.Do(t, &s, "wonder:pyramids:sawmill")
	if s.Players[0].Coins != 8 || s.Players[1].Coins != 2 {
		t.Fatalf("coins after Pyramids = %d and %d, want 8 and 2", s.Players[0].Coins, s.Players[1].Coins)
	}
}

// TestAuditR2EconMasonryWaivesDearUnitsNotFixedPriceUnits: economy.md
// "Discounts": "Ajayeb waives the two units that would otherwise cost you
// the most." Town Hall costs SSS WW. With Stone Reserve, stone costs 1 even
// though the opponent makes 2 stone; wood costs 2. Masonry must waive the
// two woods, leaving 3 stone at 1: 3 coins, all to the opponent's Economy
// (waiving two stones would leave 1 + 4 = 5).
func TestAuditR2EconMasonryWaivesDearUnitsNotFixedPriceUnits(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(3).
		City(0, "stone-reserve").
		Tokens(0, "masonry").
		City(1, "shelf-quarry").
		Tokens(1, "economy").
		Accessible("town-hall").
		Coins(0, 10).
		Coins(1, 0).
		ToMove(0).
		State()

	pay := engine.BuildingPayment(&s, 0, auditR2EconCard(t, "town-hall"))
	if pay.Discount[catalog.Wood] != 2 || pay.Total != 3 || pay.ToOpponent != 3 {
		t.Fatalf("Town Hall Discount[wood]=%d Total=%d ToOpponent=%d, want 2/3/3", pay.Discount[catalog.Wood], pay.Total, pay.ToOpponent)
	}
	enginetest.Do(t, &s, "construct:town-hall")
	if s.Players[0].Coins != 7 || s.Players[1].Coins != 3 {
		t.Fatalf("coins = %d and %d, want 7 and 3", s.Players[0].Coins, s.Players[1].Coins)
	}
}

// TestAuditR2EconCustomsHouseForumAndMasonryOnThePalace: the economy.md
// "Discounts" worked example, varied with a fixed price. Palace costs C S W
// GG. The Customs House makes glass 1 coin; the opponent's Shelf Quarry makes
// stone 4; clay and wood cost 2. The Forum can only cover a glass (step 5);
// Masonry then waives the stone (4) and one 2-coin unit (step 6). Left: one
// 2-coin unit and one glass at 1, for 3 coins.
func TestAuditR2EconCustomsHouseForumAndMasonryOnThePalace(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(3).
		City(0, "customs-house", "forum").
		Tokens(0, "masonry").
		City(1, "shelf-quarry").
		Accessible("palace").
		Coins(0, 10).
		ToMove(0).
		State()

	pay := engine.BuildingPayment(&s, 0, auditR2EconCard(t, "palace"))
	if pay.Discount[catalog.Stone] != 1 {
		t.Errorf("Discount[stone] = %d, want 1 (the dearest unit is waived)", pay.Discount[catalog.Stone])
	}
	if pay.Total != 3 {
		t.Fatalf("Palace Total = %d, want 3 (one 2-coin unit and one glass at the fixed price)", pay.Total)
	}
}

// TestAuditR2EconExactCoinsWithEconomyAndFixedPrices: economy.md "The
// cheapest payment" step 7: a construction is allowed when the total is no
// more than your coins, so exactly enough is enough and one short is not.
// Caravansery costs 2 printed coins plus G P; with the Customs House both are
// bought at 1, for 4 in all. The opponent owns Economy and receives only the
// 2 trading coins (economy.md "The Economy token"), and the payer ends at
// exactly 0 coins (economy.md "Coins": never below zero).
func TestAuditR2EconExactCoinsWithEconomyAndFixedPrices(t *testing.T) {
	build := func(coins int) engine.State {
		return enginetest.New(t, 1).
			Age(2).
			City(0, "customs-house").
			City(1, "glass-blower", "drying-room").
			Tokens(1, "economy").
			Accessible("caravansery").
			Coins(0, coins).
			Coins(1, 0).
			ToMove(0).
			State()
	}

	short := build(3)
	if r := enginetest.MustFail(t, &short, "construct:caravansery"); r != engine.ReasonUnaffordable {
		t.Fatalf("with 3 coins: reason %v, want ReasonUnaffordable (the total is 4)", r)
	}

	s := build(4)
	pay := engine.BuildingPayment(&s, 0, auditR2EconCard(t, "caravansery"))
	if pay.Total != 4 || pay.ToOpponent != 2 || !pay.Affordable {
		t.Fatalf("Caravansery Total=%d ToOpponent=%d Affordable=%v, want 4/2/true", pay.Total, pay.ToOpponent, pay.Affordable)
	}
	enginetest.Do(t, &s, "construct:caravansery")
	if s.Players[0].Coins != 0 || s.Players[1].Coins != 2 {
		t.Fatalf("coins = %d and %d, want 0 and 2 (printed 2 to the bank, trading 2 to Economy)", s.Players[0].Coins, s.Players[1].Coins)
	}
}

// --- Order of resolution: tokens gained during the action ------------------

// TestAuditR2EconArchitectureFromTheGreatLibraryDoesNotDiscountTheLibrary:
// wonders-tokens.md "Progress tokens": lasting effects "apply to everything
// the player does afterwards, never to what they did before", and "The order
// of a wonder's effects" pays the cost (step 1) before the special choice
// (step 7). Taking Architecture from the Great Library's offer must not
// discount the Great Library itself (WWW G P, 10 coins), but discounts the
// next wonder (Pyramids SSS P: 4 coins).
func TestAuditR2EconArchitectureFromTheGreatLibraryDoesNotDiscountTheLibrary(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(1).
		Wonders(0, "great-library", "pyramids", "colossus", "sphinx").
		BoardTokens("agriculture", "economy", "mathematics", "philosophy", "strategy").
		BoxOrder("architecture", "masonry", "law").
		Accessible("lumber-yard").
		Coins(0, 20).
		ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:great-library:lumber-yard")
	if s.Players[0].Coins != 10 {
		t.Fatalf("coins after the Great Library = %d, want 10 (5 units at 2, no discount yet)", s.Players[0].Coins)
	}
	if s.Phase != engine.PhasePickLibraryToken {
		t.Fatalf("phase = %v, want PhasePickLibraryToken", s.Phase)
	}
	enginetest.Do(t, &s, "token:architecture")
	if s.Players[0].Coins != 10 {
		t.Fatalf("coins after taking Architecture = %d, want 10 (no refund for the Library)", s.Players[0].Coins)
	}
	pay := engine.WonderPayment(&s, 0, auditR2EconWonder(t, "pyramids"))
	if pay.Total != 4 || pay.DiscountBy != auditR2EconToken(t, "architecture") {
		t.Fatalf("next wonder: Total=%d DiscountBy=%v, want 4/architecture", pay.Total, pay.DiscountBy)
	}
}

// TestAuditR2EconUrbanismTakenThroughAChainPairPaysNoChainBonus: the
// Dispensary is built free through the Pharmacist's chain, and its mortar
// completes a pair with the Pharmacist's. Urbanism is the last token on the
// board. The token is taken after the card (decisions.md "Effect order":
// coin rewards come before the science symbol and the pair), and lasting
// effects never apply to what was done before (wonders-tokens.md "Progress
// tokens"), so the player gains only Urbanism's 6 coins, not 6 + 4.
func TestAuditR2EconUrbanismTakenThroughAChainPairPaysNoChainBonus(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(2).
		City(0, "pharmacist").
		Tokens(1, "philosophy", "mathematics", "theology", "strategy").
		BoardTokens("urbanism").
		Accessible("dispensary").
		Coins(0, 5).
		ToMove(0).
		State()

	if p := engine.BuildingPayment(&s, 0, auditR2EconCard(t, "dispensary")); !p.ChainFree {
		t.Fatalf("Dispensary should be free through the Pharmacist's chain")
	}
	enginetest.Do(t, &s, "construct:dispensary")
	if s.Players[0].Coins != 5 {
		t.Fatalf("coins after the chain build = %d, want 5 (no Urbanism yet)", s.Players[0].Coins)
	}
	if s.Phase != engine.PhasePickBoardToken {
		t.Fatalf("phase = %v, want PhasePickBoardToken (two mortars make a pair)", s.Phase)
	}
	enginetest.Do(t, &s, "token:urbanism")
	if s.Players[0].Coins != 11 {
		t.Fatalf("coins after taking Urbanism = %d, want 11 (6 only, no retroactive chain bonus)", s.Players[0].Coins)
	}
}

// TestAuditR2EconUrbanismPaysWhenOwnProductionWouldCoverTheCost:
// wonders-tokens.md "Urbanism": each building you get for free through a
// chain pays 4 coins; economy.md "Chains" says the chained card is free
// "ignoring both its coins and its resources". Horse Breeders (C W) is
// chained from the Stable even though the builder's own Clay Pool and
// Lumber Yard would cover it at 0 coins anyway; it is still a chain
// construction, so Urbanism pays 4. (Ambiguity note: the physical rules let
// a player pay the normal cost instead; with a 0-coin normal cost nobody
// would decline 4 coins, and Ajayeb always takes the chain.)
func TestAuditR2EconUrbanismPaysWhenOwnProductionWouldCoverTheCost(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(2).
		City(0, "stable", "clay-pool", "lumber-yard").
		Tokens(0, "urbanism").
		Accessible("horse-breeders").
		Coins(0, 5).
		ToMove(0).
		State()

	pay := engine.BuildingPayment(&s, 0, auditR2EconCard(t, "horse-breeders"))
	if !pay.ChainFree || pay.UrbanismBonus != 4 {
		t.Fatalf("Horse Breeders ChainFree=%v UrbanismBonus=%d, want true/4", pay.ChainFree, pay.UrbanismBonus)
	}
	enginetest.Do(t, &s, "construct:horse-breeders")
	if s.Players[0].Coins != 9 {
		t.Fatalf("coins = %d, want 9 (5 + Urbanism's 4)", s.Players[0].Coins)
	}
}

// TestAuditR2EconUrbanismThenLighthouseCoinsInOrder: decisions.md "Effect
// order": a building's coin rewards come "first Urbanism's chain bonus, then
// the card's own coins". The Lighthouse, chained from the Tavern, pays 1 coin
// per yellow card including itself (economy.md "Coin rewards"): Tavern,
// Brewery and Lighthouse make 3. From 0 coins: +4, then +3.
func TestAuditR2EconUrbanismThenLighthouseCoinsInOrder(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(3).
		City(0, "tavern", "brewery").
		Tokens(0, "urbanism").
		Accessible("lighthouse").
		Coins(0, 0).
		ToMove(0).
		State()

	ev := enginetest.Do(t, &s, "construct:lighthouse")
	if s.Players[0].Coins != 7 {
		t.Fatalf("coins = %d, want 7 (4 from Urbanism and 3 from the Lighthouse)", s.Players[0].Coins)
	}
	coins := auditR2EconCoinEvents(ev, 0)
	if len(coins) != 2 {
		t.Fatalf("coin events %+v, want two", coins)
	}
	if coins[0].Token != auditR2EconToken(t, "urbanism") || coins[0].A != 4 || coins[0].B != 4 {
		t.Errorf("first coin event %+v, want Urbanism +4 to 4", coins[0])
	}
	if coins[1].Card != auditR2EconCard(t, "lighthouse") || coins[1].A != 3 || coins[1].B != 7 {
		t.Errorf("second coin event %+v, want Lighthouse +3 to 7", coins[1])
	}
}

// TestAuditR2EconMausoleumRevivedChainCardFreesItsChainWithUrbanism:
// wonders-tokens.md "The Mausoleum": the revived card enters the city with
// all its effects, chain symbols included. The revive itself earns no
// Urbanism bonus (decisions.md "Urbanism and the Mausoleum"), but the
// Stable it brings back then frees Horse Breeders through its chain, and
// that chain construction does earn Urbanism's 4 coins.
func TestAuditR2EconMausoleumRevivedChainCardFreesItsChainWithUrbanism(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(2).
		Wonders(0, "mausoleum", "pyramids", "colossus", "sphinx").
		Tokens(0, "urbanism").
		Discard("stable").
		Accessible("horse-breeders", "sawmill").
		Coins(0, 30).
		ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:mausoleum:sawmill") // CC GG P: 10 coins
	afterWonder := s.Players[0].Coins
	enginetest.Do(t, &s, "revive:stable")
	if s.Players[0].Coins != afterWonder {
		t.Fatalf("reviving the Stable changed coins by %d, want 0", s.Players[0].Coins-afterWonder)
	}
	// The opponent plays a discard that leaves Horse Breeders in place.
	step := ""
	for _, k := range enginetest.Keys(&s) {
		if strings.HasPrefix(k, "discard:") && k != "discard:horse-breeders" {
			step = k
			break
		}
	}
	if step == "" {
		t.Fatalf("no discard for the opponent among %v", enginetest.Keys(&s))
	}
	enginetest.Do(t, &s, step)
	if s.ToMove != 0 {
		t.Fatalf("to move = %d, want seat 0", s.ToMove)
	}
	before := s.Players[0].Coins
	enginetest.Do(t, &s, "construct:horse-breeders")
	if d := s.Players[0].Coins - before; d != 4 {
		t.Fatalf("Horse Breeders through the revived Stable's chain: coins changed by %d, want +4 (free, plus Urbanism)", d)
	}
}

// TestAuditR2EconEconomyGetsNothingFromAMausoleumRevive: economy.md "The
// Economy token": the owner receives the coins the opponent pays for
// trading. The Mausoleum's own cost (CC GG P, 10 coins bought) is trading
// and goes to the Economy owner; the revived Palace is built for free
// (wonders-tokens.md "The Mausoleum"), so it involves no trading and pays
// the Economy owner nothing.
func TestAuditR2EconEconomyGetsNothingFromAMausoleumRevive(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(3).
		Wonders(0, "mausoleum", "pyramids", "colossus", "sphinx").
		Tokens(1, "economy").
		Discard("palace").
		Accessible("arsenal").
		Coins(0, 30).
		Coins(1, 0).
		ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:mausoleum:arsenal")
	if s.Players[0].Coins != 20 || s.Players[1].Coins != 10 {
		t.Fatalf("after the Mausoleum: coins %d and %d, want 20 and 10", s.Players[0].Coins, s.Players[1].Coins)
	}
	enginetest.Do(t, &s, "revive:palace")
	if s.Players[0].Coins != 20 || s.Players[1].Coins != 10 {
		t.Fatalf("after reviving the Palace: coins %d and %d, want 20 and 10 (a free build has no trading)", s.Players[0].Coins, s.Players[1].Coins)
	}
}

// --- Destroying production mid-action -------------------------------------

// TestAuditR2EconStatueOfZeusPaysTheOldPriceThenThePriceFalls:
// wonders-tokens.md "The order of a wonder's effects" pays the cost (step 1)
// before the destroy choice (step 7), and "The Statue of Zeus and Circus
// Maximus": the builder's price for the destroyed card's resource drops at
// once. The Statue of Zeus (PP C W S) is paid with clay at 4 (the
// opponent's Brickyard): 12 coins, with no refund afterwards; then clay is
// back to 2.
func TestAuditR2EconStatueOfZeusPaysTheOldPriceThenThePriceFalls(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(2).
		Wonders(0, "statue-of-zeus", "pyramids", "colossus", "sphinx").
		City(1, "brickyard").
		Accessible("sawmill").
		Coins(0, 30).
		ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:statue-of-zeus:sawmill")
	if s.Players[0].Coins != 18 {
		t.Fatalf("coins after the Statue of Zeus = %d, want 18 (30 - 12, clay at 4)", s.Players[0].Coins)
	}
	enginetest.Do(t, &s, "destroy:brickyard")
	if got := engine.TradePrices(&s, 0)[catalog.Clay]; got != 2 {
		t.Fatalf("clay price after destroying the Brickyard = %d, want 2", got)
	}
	if s.Players[0].Coins != 18 {
		t.Fatalf("coins after the destroy = %d, want 18 (no refund)", s.Players[0].Coins)
	}
}

// TestAuditR2EconCircusMaximusEconomyIncomeKeptAfterTheDestroy: the
// opponent owns Economy and a Glass-Blower. Circus Maximus (SS W G) costs
// 2 + 2 + 2 + 3 = 9, all trading, so the opponent receives 9 (economy.md
// "The Economy token"). Destroying the Glass-Blower afterwards drops the
// builder's glass price to 2 at once but takes nothing back from the
// opponent (wonders-tokens.md "The Statue of Zeus and Circus Maximus").
func TestAuditR2EconCircusMaximusEconomyIncomeKeptAfterTheDestroy(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(2).
		Wonders(0, "circus-maximus", "pyramids", "colossus", "sphinx").
		City(1, "glass-blower").
		Tokens(1, "economy").
		Accessible("sawmill").
		Coins(0, 20).
		Coins(1, 0).
		ToMove(0).
		State()

	enginetest.Do(t, &s, "wonder:circus-maximus:sawmill", "destroy:glass-blower")
	if s.Players[0].Coins != 11 || s.Players[1].Coins != 9 {
		t.Fatalf("coins %d and %d, want 11 and 9", s.Players[0].Coins, s.Players[1].Coins)
	}
	if got := engine.TradePrices(&s, 0)[catalog.Glass]; got != 2 {
		t.Fatalf("glass price after the destroy = %d, want 2", got)
	}
}

// --- Boundaries: the ninth shield and the last card of the game ----------

// TestAuditR2EconEconomyIncomeArrivesBeforeTheNinthShield: decisions.md
// "Effect order": the payment, with the trading part to an Economy owner,
// comes before the shields, and a military supremacy ends the game inside
// the pawn's move. Walls (SS, 2 shields) bought by seat 1 with the pawn at
// -7 takes it to seat 0's capital; the 4 trading coins still reached seat 0.
func TestAuditR2EconEconomyIncomeArrivesBeforeTheNinthShield(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(2).
		Tokens(0, "economy").
		Coins(0, 0).
		Coins(1, 20).
		Pawn(-7).
		Accessible("walls").
		ToMove(1).
		State()

	enginetest.Do(t, &s, "construct:walls")
	if !s.Over() || s.Result.Kind != engine.VictoryMilitary || s.Result.Winner != 1 {
		t.Fatalf("result %+v, want a military victory for seat 1", s.Result)
	}
	if s.Players[0].Coins != 4 || s.Players[1].Coins != 16 {
		t.Fatalf("coins %d and %d, want 4 and 16 (the trading income was paid before the supremacy)", s.Players[0].Coins, s.Players[1].Coins)
	}
}

// TestAuditR2EconDiscardOfTheLastCardCountsInTheTreasury: turn.md "2.
// Discard it for coins" (2 plus one per yellow card) and economy.md
// "Coins": 1 point per 3 coins at the end. Discarding the very last card of
// Age III with 7 coins and one yellow card gives 3 coins, so the final
// score sees 10 coins: 3 points, not 2.
func TestAuditR2EconDiscardOfTheLastCardCountsInTheTreasury(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(3).
		City(0, "tavern").
		OnlyLeft("arsenal").
		Coins(0, 7).
		ToMove(0).
		State()

	enginetest.Do(t, &s, "discard:arsenal")
	if !s.Over() || s.Result.Kind != engine.VictoryCivilian {
		t.Fatalf("result %+v, want the civilian end of the game", s.Result)
	}
	if s.Players[0].Coins != 10 || s.Result.Scores[0].Coins != 3 {
		t.Fatalf("coins %d, coin points %d, want 10 and 3", s.Players[0].Coins, s.Result.Scores[0].Coins)
	}
}

// TestAuditR2EconEconomyIncomeOnTheLastCardCountsInTheTreasury: the trading
// coins an Economy owner receives on the last card of Age III arrive with
// the payment (decisions.md "Effect order"), before the game is scored, so
// they count for 1 point per 3 coins (economy.md "Coins"). Obelisk (SS G)
// bought for 6; the owner goes from 8 coins (2 points) to 14 (4 points),
// and the payer from 20 (6 points) to 14 (4 points).
func TestAuditR2EconEconomyIncomeOnTheLastCardCountsInTheTreasury(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(3).
		Tokens(1, "economy").
		OnlyLeft("obelisk").
		Coins(0, 20).
		Coins(1, 8).
		ToMove(0).
		State()

	enginetest.Do(t, &s, "construct:obelisk")
	if !s.Over() {
		t.Fatalf("the game should be over after the last card of Age III")
	}
	if s.Players[1].Coins != 14 || s.Result.Scores[1].Coins != 4 {
		t.Fatalf("Economy owner: coins %d, coin points %d, want 14 and 4", s.Players[1].Coins, s.Result.Scores[1].Coins)
	}
	if s.Players[0].Coins != 14 || s.Result.Scores[0].Coins != 4 {
		t.Fatalf("payer: coins %d, coin points %d, want 14 and 4", s.Players[0].Coins, s.Result.Scores[0].Coins)
	}
}

// TestAuditR2EconPiraeusAndLighthouseWithArchitectureOnStatueOfZeus: a wonder
// paid with two wonder producers and Architecture together. Statue of Zeus
// costs PP C W S; the opponent's Shelf Quarry makes stone 4. Piraeus covers a
// papyrus and the Great Lighthouse the dearest raw unit (stone, step 5);
// Architecture waives two of the three 2-coin units left (step 6); one unit
// is bought: 2 coins.
func TestAuditR2EconPiraeusAndLighthouseWithArchitectureOnStatueOfZeus(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(2).
		Wonders(0, "piraeus", "great-lighthouse", "statue-of-zeus", "sphinx").
		Built(0, "piraeus", "").
		Built(0, "great-lighthouse", "").
		Tokens(0, "architecture").
		City(1, "shelf-quarry").
		ToMove(0).
		State()

	pay := engine.WonderPayment(&s, 0, auditR2EconWonder(t, "statue-of-zeus"))
	if pay.NChoice != 2 || pay.TradeUnits[catalog.Stone] != 0 {
		t.Errorf("NChoice=%d stone bought=%d, want 2/0", pay.NChoice, pay.TradeUnits[catalog.Stone])
	}
	waived := 0
	for r := range catalog.NumResources {
		waived += int(pay.Discount[r])
	}
	if waived != 2 || pay.Total != 2 {
		t.Fatalf("waived=%d Total=%d, want 2/2", waived, pay.Total)
	}
}

// --- What the client is shown matches what is charged ---------------------

// auditR2EconTownHall builds the Town Hall position of
// TestAuditR2EconMasonryWaivesDearUnitsNotFixedPriceUnits with the given
// coins for the payer.
func auditR2EconTownHall(t *testing.T, coins int) engine.State {
	t.Helper()
	return enginetest.New(t, 1).
		Age(3).
		City(0, "stone-reserve").
		Tokens(0, "masonry").
		City(1, "shelf-quarry").
		Tokens(1, "economy").
		Accessible("town-hall").
		Coins(0, coins).
		Coins(1, 0).
		ToMove(0).
		State()
}

// TestAuditR2EconViewChoiceShowsTheChargedPayment: the client never computes
// a cost and shows what the engine sends (economy.md "For developers": the
// view carries each breakdown, each trade price and each discard value).
// The construct choice for the Town Hall must show the payment that is then
// charged: 3 coins, all to the opponent's Economy, Masonry waiving the two
// woods, 3 stone bought at the fixed price of 1. The discard choice gains
// 2 + 1 yellow card (the Stone Reserve) = 3 (turn.md "2. Discard it for
// coins").
func TestAuditR2EconViewChoiceShowsTheChargedPayment(t *testing.T) {
	s := auditR2EconTownHall(t, 10)
	v := engine.ViewFor(&s, engine.SeatViewer(0))

	var build, discard *view.Choice
	for i := range v.Choices {
		switch v.Choices[i].Key {
		case "construct:town-hall":
			build = &v.Choices[i]
		case "discard:town-hall":
			discard = &v.Choices[i]
		}
	}
	if build == nil || build.Cost == nil || discard == nil || discard.Gain == nil {
		t.Fatalf("missing construct or discard choice for the Town Hall: %+v", v.Choices)
	}
	c := build.Cost
	if c.Total != 3 || c.ToOpponent != 3 || c.PrintedCoins != 0 || c.TradeTotal != 3 {
		t.Errorf("cost Total=%d ToOpponent=%d Printed=%d Trade=%d, want 3/3/0/3", c.Total, c.ToOpponent, c.PrintedCoins, c.TradeTotal)
	}
	if c.Discount == nil || c.Discount.Token != "masonry" || c.Discount.Units.Wood != 2 {
		t.Errorf("discount %+v, want masonry waiving 2 wood", c.Discount)
	}
	if len(c.Trade) != 1 || c.Trade[0].Units != 3 || c.Trade[0].UnitPrice != 1 || !c.Trade[0].Fixed || c.Trade[0].Coins != 3 {
		t.Errorf("trade %+v, want 3 stone at the fixed price of 1", c.Trade)
	}
	if *discard.Gain != 3 || v.Players[0].DiscardValue != 3 {
		t.Errorf("discard gain %d, discard value %d, want 3 and 3", *discard.Gain, v.Players[0].DiscardValue)
	}
	if v.Players[0].TradePrices.Stone != 1 || v.Players[1].TradePrices.Stone != 2 {
		t.Errorf("stone prices %d and %d, want 1 (fixed) and 2 (seat 0 makes no stone)", v.Players[0].TradePrices.Stone, v.Players[1].TradePrices.Stone)
	}

	before0, before1 := s.Players[0].Coins, s.Players[1].Coins
	enginetest.Do(t, &s, build.Key)
	if int(before0-s.Players[0].Coins) != c.Total || int(s.Players[1].Coins-before1) != c.ToOpponent {
		t.Fatalf("charged %d (opponent got %d), but the view showed %d (%d)",
			before0-s.Players[0].Coins, s.Players[1].Coins-before1, c.Total, c.ToOpponent)
	}
}

// TestAuditR2EconViewShortfallIsExact: economy.md "The cheapest payment"
// step 7 allows a construction only when its total is no more than the
// player's coins. With 2 coins the Town Hall (3 coins) is not a choice; it
// is a disabled play with the same cost and a shortfall of exactly 1.
func TestAuditR2EconViewShortfallIsExact(t *testing.T) {
	s := auditR2EconTownHall(t, 2)
	v := engine.ViewFor(&s, engine.SeatViewer(0))
	for _, ch := range v.Choices {
		if ch.Key == "construct:town-hall" {
			t.Fatalf("construct:town-hall is offered with 2 coins; it costs 3")
		}
	}
	var d *view.DisabledChoice
	for i := range v.Disabled {
		if v.Disabled[i].Key == "construct:town-hall" {
			d = &v.Disabled[i]
		}
	}
	if d == nil || d.Cost == nil {
		t.Fatalf("construct:town-hall missing from the disabled plays: %+v", v.Disabled)
	}
	if d.Cost.Affordable || d.Cost.Total != 3 || d.Cost.Shortfall != 1 {
		t.Fatalf("disabled cost Affordable=%v Total=%d Shortfall=%d, want false/3/1", d.Cost.Affordable, d.Cost.Total, d.Cost.Shortfall)
	}
}
