package engine_test

// Rules-adversary pass, lens: Economy (R1).
//
// These tests are written black-box from the rules reference
// (~/tmp/agents/ajayeb/design/rules-reference.md, sections 2-8) and from
// docs/rules/{economy,decisions,wonders-tokens,ages-scoring,turn,data-rulings}.md,
// against the exported engine API only. Each test cites the rule it encodes.

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

// advR1Card looks up a building's id by slug, failing the test loudly if the
// slug is wrong (so a typo is a clear test failure, not a silent zero value).
func advR1Card(t *testing.T, slug string) catalog.CardID {
	t.Helper()
	id, ok := catalog.CardBySlug(slug)
	if !ok {
		t.Fatalf("advR1Card: unknown card slug %q", slug)
	}
	return id
}

// advR1Wonder looks up a wonder's id by slug.
func advR1Wonder(t *testing.T, slug string) catalog.WonderID {
	t.Helper()
	id, ok := catalog.WonderBySlug(slug)
	if !ok {
		t.Fatalf("advR1Wonder: unknown wonder slug %q", slug)
	}
	return id
}

// --- Trading price formula -------------------------------------------------

// TestAdvEconomyR1_TradePriceCountsBothProducersAndIgnoresFlexible encodes the
// worked example of economy.md "Trading": "your opponent owns a Clay Pool and
// a Brickyard (3 clay in total) and a Caravansery. Each clay unit costs you
// 2 + 3 = 5 coins. The Caravansery adds nothing to the price. A card needing
// 2 clay, when you produce none, costs you 10 coins of trading."
// (Clay Pool makes 1 clay, Brickyard makes 2; rules-reference section 3/5.)
func TestAdvEconomyR1_TradePriceCountsBothProducersAndIgnoresFlexible(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(2).
		City(1, "clay-pool", "brickyard", "caravansery").
		Accessible("statue").
		ToMove(0).
		Coins(0, 30).
		State()

	prices := engine.TradePrices(&s, 0)
	if got := prices[catalog.Clay]; got != 5 {
		t.Fatalf("clay trade price = %d, want 5 (2 + 1 + 2, Caravansery excluded)", got)
	}

	id := advR1Card(t, "statue") // cost CC (2 clay), 0 printed coins
	pay := engine.BuildingPayment(&s, 0, id)
	if pay.Total != 10 {
		t.Fatalf("Statue (2 clay) total = %d, want 10 (2 units * 5)", pay.Total)
	}
}

// TestAdvEconomyR1_FixedPriceIsPersonalOnly encodes economy.md "Fixed prices":
// "A fixed price helps only its owner; it does not change what the opponent
// pays." Stone Reserve and Customs House fix stone, glass and papyrus at 1
// coin for their owner only; neither card produces those resources (TradeFix
// is not EffProduce), so the owner's other trade partner sees the base price.
func TestAdvEconomyR1_FixedPriceIsPersonalOnly(t *testing.T) {
	s := enginetest.New(t, 1).
		City(0, "stone-reserve", "customs-house").
		State()

	own := engine.TradePrices(&s, 0)
	if own[catalog.Stone] != 1 {
		t.Errorf("owner's stone price = %d, want 1 (Stone Reserve)", own[catalog.Stone])
	}
	if own[catalog.Glass] != 1 || own[catalog.Papyrus] != 1 {
		t.Errorf("owner's glass/papyrus price = %d/%d, want 1/1 (Customs House)", own[catalog.Glass], own[catalog.Papyrus])
	}
	if own[catalog.Wood] != 2 || own[catalog.Clay] != 2 {
		t.Errorf("owner's wood/clay price = %d/%d, want 2/2 (not fixed)", own[catalog.Wood], own[catalog.Clay])
	}

	opp := engine.TradePrices(&s, 1)
	if opp[catalog.Stone] != 2 || opp[catalog.Glass] != 2 || opp[catalog.Papyrus] != 2 {
		t.Errorf("opponent's prices = %d/%d/%d, want 2/2/2 (fixed price never spreads to the opponent, and TradeFix is not production)",
			opp[catalog.Stone], opp[catalog.Glass], opp[catalog.Papyrus])
	}
}

// --- Flexible producers ------------------------------------------------

// TestAdvEconomyR1_FlexibleProducerCoversTheDearestUnit encodes economy.md
// "The cheapest payment", step 5: "Let each flexible producer cover the most
// expensive remaining unit it can produce" (see also decisions-reference
// section 8 item 6: "the best choice is the units that would cost most to
// buy"). University costs C G P; with a Forum (glass or papyrus) and the
// opponent producing papyrus but not glass, Forum must cover papyrus (the
// dearer unit), not glass.
func TestAdvEconomyR1_FlexibleProducerCoversTheDearestUnit(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(3).
		City(0, "forum").
		City(1, "press", "drying-room"). // 2 papyrus production, 0 glass
		Accessible("university").
		ToMove(0).
		Coins(0, 20).
		State()

	id := advR1Card(t, "university") // cost C G P
	pay := engine.BuildingPayment(&s, 0, id)

	if pay.NChoice != 1 {
		t.Fatalf("NChoice = %d, want 1 (Forum covers one unit)", pay.NChoice)
	}
	if pay.Choice[0].Res != catalog.Papyrus {
		t.Fatalf("Forum covered %v, want papyrus (price 4) over glass (price 2)", pay.Choice[0].Res)
	}
	if pay.TradeUnits[catalog.Glass] != 1 || pay.TradeUnits[catalog.Papyrus] != 0 {
		t.Errorf("trade units glass=%d papyrus=%d, want 1/0", pay.TradeUnits[catalog.Glass], pay.TradeUnits[catalog.Papyrus])
	}
	// clay(2) + glass(2), papyrus covered free.
	if pay.Total != 4 {
		t.Fatalf("Total = %d, want 4", pay.Total)
	}
}

// TestAdvEconomyR1_TwoFlexibleProducersCoverTwoUnits encodes economy.md
// "Flexible production": "Two flexible producers cover two units, for example
// Forum and Piraeus together can cover one glass and one papyrus."
func TestAdvEconomyR1_TwoFlexibleProducersCoverTwoUnits(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(3).
		Wonders(0, "piraeus", "pyramids", "colossus", "sphinx").
		Built(0, "piraeus", "").
		City(0, "forum").
		Accessible("university").
		ToMove(0).
		Coins(0, 20).
		State()

	id := advR1Card(t, "university") // cost C G P
	pay := engine.BuildingPayment(&s, 0, id)

	if pay.NChoice != 2 {
		t.Fatalf("NChoice = %d, want 2 (Forum and Piraeus each cover one unit)", pay.NChoice)
	}
	if pay.TradeUnits[catalog.Glass] != 0 || pay.TradeUnits[catalog.Papyrus] != 0 {
		t.Errorf("glass/papyrus should both be covered by producers, got trade units %d/%d",
			pay.TradeUnits[catalog.Glass], pay.TradeUnits[catalog.Papyrus])
	}
	if pay.Total != 2 { // only the clay unit is bought
		t.Fatalf("Total = %d, want 2 (just the clay unit)", pay.Total)
	}
}

// --- Masonry and Architecture scope --------------------------------------

// TestAdvEconomyR1_MasonryAppliesOnlyToBlueCards encodes wonders-tokens.md:
// "Masonry: every blue card you construct afterwards costs 2 fewer resource
// units." Walls is red, so Masonry must not touch its price.
func TestAdvEconomyR1_MasonryAppliesOnlyToBlueCards(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(2).
		Tokens(0, "masonry").
		Accessible("walls"). // red, cost SS, no chain
		ToMove(0).
		Coins(0, 10).
		State()

	id := advR1Card(t, "walls")
	pay := engine.BuildingPayment(&s, 0, id)
	if pay.Total != 4 {
		t.Fatalf("Walls total with Masonry = %d, want 4 (unaffected: Walls is red, not blue)", pay.Total)
	}
	if pay.Discount[catalog.Stone] != 0 || pay.DiscountBy != 0 {
		t.Errorf("Discount=%d DiscountBy=%v, want 0/0 (Masonry does not apply to a red card)", pay.Discount[catalog.Stone], pay.DiscountBy)
	}
}

// TestAdvEconomyR1_ArchitectureAppliesOnlyToWonders encodes wonders-tokens.md:
// "Architecture: every wonder you build afterwards costs 2 fewer resource
// units," which (per economy.md "Discounts") never touches a building's cost,
// blue or not. It also checks the positive case: Architecture does reduce a
// wonder's cost.
func TestAdvEconomyR1_ArchitectureAppliesOnlyToWonders(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(1).
		Tokens(0, "architecture").
		Wonders(0, "pyramids", "colossus", "sphinx", "mausoleum").
		Accessible("baths"). // blue, cost S
		ToMove(0).
		Coins(0, 10).
		State()

	bathsID := advR1Card(t, "baths")
	bpay := engine.BuildingPayment(&s, 0, bathsID)
	if bpay.Total != 2 {
		t.Fatalf("Baths total with Architecture = %d, want 2 (unaffected: a building, not a wonder)", bpay.Total)
	}
	if bpay.DiscountBy != 0 {
		t.Errorf("DiscountBy = %v, want 0 (Architecture does not apply to buildings)", bpay.DiscountBy)
	}

	pyr := advR1Wonder(t, "pyramids") // cost SSS P, 4 units, each priced 2
	wpay := engine.WonderPayment(&s, 0, pyr)
	if wpay.Total != 4 {
		t.Fatalf("Pyramids total with Architecture = %d, want 4 (8 - 2*2 waived)", wpay.Total)
	}
	if wpay.DiscountBy != catalog.TokenArchitecture {
		t.Errorf("DiscountBy = %v, want architecture", wpay.DiscountBy)
	}
}

// TestAdvEconomyR1_MasonryDiscountCapitalizesOnFewerThanTwoRemainingUnits
// encodes economy.md "Discounts": "If fewer than two units remain to be paid
// after your own production and your flexible producers, the unused part of
// the discount is lost. It is never refunded in coins." Baths costs a single
// stone unit.
func TestAdvEconomyR1_MasonryDiscountCapitalizesOnFewerThanTwoRemainingUnits(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(1).
		Tokens(0, "masonry").
		Accessible("baths"). // blue, cost S (1 stone only)
		ToMove(0).
		Coins(0, 10).
		State()

	id := advR1Card(t, "baths")
	pay := engine.BuildingPayment(&s, 0, id)
	if pay.Total != 0 {
		t.Fatalf("Baths total with Masonry = %d, want 0 (the single stone unit is waived)", pay.Total)
	}
	if pay.Discount[catalog.Stone] != 1 {
		t.Fatalf("Discount[stone] = %d, want 1 (only one unit existed to waive, never 2 and never a coin refund)", pay.Discount[catalog.Stone])
	}
	if pay.DiscountBy != catalog.TokenMasonry {
		t.Errorf("DiscountBy = %v, want masonry", pay.DiscountBy)
	}
}

// TestAdvEconomyR1_MasonryWaivesAfterOwnProduction encodes economy.md "The
// cheapest payment" steps 3 then 6: own production is subtracted first, and
// only the remaining units are priced and then discounted. Aqueduct costs
// SSS; with one own stone producer, one unit is already covered before
// Masonry waives the rest.
func TestAdvEconomyR1_MasonryWaivesAfterOwnProduction(t *testing.T) {
	base := func(withMasonry bool) engine.Payment {
		b := enginetest.New(t, 1).
			Age(2).
			City(0, "quarry"). // 1 stone production
			Accessible("aqueduct")
		if withMasonry {
			b = b.Tokens(0, "masonry")
		}
		s := b.ToMove(0).Coins(0, 10).State()
		id := advR1Card(t, "aqueduct") // blue, cost SSS
		return engine.BuildingPayment(&s, 0, id)
	}

	without := base(false)
	if without.Own[catalog.Stone] != 1 {
		t.Fatalf("own production = %d, want 1", without.Own[catalog.Stone])
	}
	if without.Total != 4 { // 2 remaining units at price 2 each
		t.Fatalf("Total without Masonry = %d, want 4", without.Total)
	}

	with := base(true)
	if with.Own[catalog.Stone] != 1 {
		t.Fatalf("own production with Masonry = %d, want 1 (unaffected by the discount)", with.Own[catalog.Stone])
	}
	if with.Discount[catalog.Stone] != 2 {
		t.Fatalf("Discount[stone] = %d, want 2 (both remaining units waived)", with.Discount[catalog.Stone])
	}
	if with.Total != 0 {
		t.Fatalf("Total with Masonry = %d, want 0", with.Total)
	}
}

// --- The Masonry worked example (economy.md "Discounts") ------------------
//
// "The Palace costs one clay, one stone, one wood and two glass. Suppose you
// produce none of these, and your opponent produces one stone and one glass.
// Your prices are 2 for clay and wood and 3 for stone and glass, so the
// Palace costs 2 + 3 + 2 + 3 + 3 = 13 coins. With Masonry, two of the 3-coin
// units are waived, for 7 coins. With Masonry and a Forum, the Forum covers
// one glass and Masonry waives the stone and the other glass, leaving only
// the clay and the wood, for 4 coins."

func TestAdvEconomyR1_PalaceMasonryExampleBase(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(3).
		City(1, "quarry", "glassworks"). // opponent: 1 stone, 1 glass
		Accessible("palace").
		ToMove(0).
		Coins(0, 50).
		State()

	id := advR1Card(t, "palace")
	pay := engine.BuildingPayment(&s, 0, id)
	if pay.Total != 13 {
		t.Fatalf("Palace base total = %d, want 13 (the economy.md worked example)", pay.Total)
	}
}

func TestAdvEconomyR1_PalaceMasonryExampleWithMasonry(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(3).
		Tokens(0, "masonry").
		City(1, "quarry", "glassworks").
		Accessible("palace").
		ToMove(0).
		Coins(0, 50).
		State()

	id := advR1Card(t, "palace")
	pay := engine.BuildingPayment(&s, 0, id)
	if pay.Total != 7 {
		t.Fatalf("Palace total with Masonry = %d, want 7 (the economy.md worked example)", pay.Total)
	}
	if pay.DiscountBy != catalog.TokenMasonry {
		t.Errorf("DiscountBy = %v, want masonry", pay.DiscountBy)
	}
}

func TestAdvEconomyR1_PalaceMasonryExampleWithMasonryAndForum(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(3).
		Tokens(0, "masonry").
		City(0, "forum").
		City(1, "quarry", "glassworks").
		Accessible("palace").
		ToMove(0).
		Coins(0, 50).
		State()

	id := advR1Card(t, "palace")
	pay := engine.BuildingPayment(&s, 0, id)
	if pay.NChoice != 1 || pay.Choice[0].Res != catalog.Glass {
		t.Fatalf("Forum should cover one glass unit, got NChoice=%d choice=%v", pay.NChoice, pay.Choice[:pay.NChoice])
	}
	if pay.Total != 4 {
		t.Fatalf("Palace total with Masonry and Forum = %d, want 4 (the economy.md worked example)", pay.Total)
	}
}

// --- Chains ----------------------------------------------------------------

// TestAdvEconomyR1_ChainFreeRegardlessOfAffordability encodes economy.md
// "Chains": a chained construction is free "ignoring both its coins and its
// resources", so it is legal even with 0 coins. Stable (chain-out horseshoe)
// frees Horse Breeders (chain-in horseshoe).
func TestAdvEconomyR1_ChainFreeRegardlessOfAffordability(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(2).
		City(0, "stable").
		Accessible("horse-breeders").
		ToMove(0).
		Coins(0, 0).
		State()

	id := advR1Card(t, "horse-breeders")
	pay := engine.BuildingPayment(&s, 0, id)
	if !pay.ChainFree {
		t.Fatalf("ChainFree = false, want true (Stable provides the horseshoe chain)")
	}
	if pay.Total != 0 || !pay.Affordable {
		t.Fatalf("Total=%d Affordable=%v, want 0/true even with 0 coins", pay.Total, pay.Affordable)
	}
	if !enginetest.IsLegal(&s, "construct:horse-breeders") {
		t.Fatalf("construct:horse-breeders should be legal with 0 coins through the chain")
	}
}

// TestAdvEconomyR1_ChainRequiresOwnCityNotOpponents encodes economy.md
// "Chains": "Only your own city counts. Your opponent's chain cards do
// nothing for you."
func TestAdvEconomyR1_ChainRequiresOwnCityNotOpponents(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(2).
		City(1, "stable"). // the OPPONENT has Stable, not the mover
		Accessible("horse-breeders").
		ToMove(0).
		Coins(0, 20).
		State()

	id := advR1Card(t, "horse-breeders")
	pay := engine.BuildingPayment(&s, 0, id)
	if pay.ChainFree {
		t.Fatalf("ChainFree = true, want false (the chain card is in the opponent's city)")
	}
	if pay.Total == 0 {
		t.Fatalf("Total = 0, want a positive price (Horse Breeders costs C W)")
	}
}

// TestAdvEconomyR1_ChainDoesNotApplyToTuckedCard encodes turn.md "3. Build a
// wonder": "The tucked card has no effect of any kind ... its chain symbol
// does nothing." A Stable tucked under a wonder must not free Horse Breeders.
func TestAdvEconomyR1_ChainDoesNotApplyToTuckedCard(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(2).
		Wonders(0, "pyramids", "colossus", "sphinx", "mausoleum").
		Built(0, "pyramids", "stable"). // Stable is tucked, not in the city
		Accessible("horse-breeders").
		ToMove(0).
		Coins(0, 20).
		State()

	id := advR1Card(t, "horse-breeders")
	pay := engine.BuildingPayment(&s, 0, id)
	if pay.ChainFree {
		t.Fatalf("ChainFree = true, want false (a tucked card's chain symbol does nothing)")
	}
}

// --- Urbanism ----------------------------------------------------------

// TestAdvEconomyR1_UrbanismPaysOnChainFreeBuild encodes wonders-tokens.md:
// "Urbanism: ... each building you get for free through a chain also pays you
// 4 coins."
func TestAdvEconomyR1_UrbanismPaysOnChainFreeBuild(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(2).
		City(0, "stable").
		Tokens(0, "urbanism").
		Accessible("horse-breeders").
		ToMove(0).
		Coins(0, 10).
		State()

	id := advR1Card(t, "horse-breeders")
	pay := engine.BuildingPayment(&s, 0, id)
	if !pay.ChainFree || pay.Total != 0 {
		t.Fatalf("ChainFree=%v Total=%d, want true/0", pay.ChainFree, pay.Total)
	}
	if pay.UrbanismBonus != 4 {
		t.Fatalf("UrbanismBonus = %d, want 4", pay.UrbanismBonus)
	}
}

// TestAdvEconomyR1_UrbanismDoesNotPayForMausoleumRebuild encodes
// decisions.md "Urbanism and the Mausoleum": "Urbanism's 4-coin bonus applies
// only to free constructions through a chain. A card rebuilt for free by the
// Mausoleum does not earn it."
func TestAdvEconomyR1_UrbanismDoesNotPayForMausoleumRebuild(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(1).
		Wonders(0, "mausoleum", "pyramids", "colossus", "sphinx").
		Tokens(0, "urbanism").
		Discard("clay-pool"). // brown, no coin-reward effect of its own
		Accessible("stone-pit").
		ToMove(0).
		Coins(0, 100).
		State()

	before := s.Players[0].Coins
	enginetest.Do(t, &s, "wonder:mausoleum:stone-pit")
	afterBuild := s.Players[0].Coins
	if s.Phase != engine.PhaseBuildFromDiscard {
		t.Fatalf("phase = %v, want PhaseBuildFromDiscard (the discard pile is not empty)", s.Phase)
	}

	enginetest.Do(t, &s, "revive:clay-pool")
	afterRevive := s.Players[0].Coins

	if afterRevive != afterBuild {
		t.Fatalf("coins changed by %d reviving a card from the discard pile, want 0 (no Urbanism bonus; Clay Pool has no coin reward)",
			afterRevive-afterBuild)
	}
	_ = before
}

// --- The Economy token ------------------------------------------------

// TestAdvEconomyR1_EconomyCapturesOnlyTradePortion pins the worked example of
// economy.md "The Economy token": "you own Economy and produce one glass.
// Your opponent constructs a Caravansery, which costs 2 coins plus one glass
// and one papyrus, and they produce neither. Glass costs them 3 and papyrus
// 2, so they pay 7 coins in total. The bank receives the 2 printed coins and
// you receive 5."
func TestAdvEconomyR1_EconomyCapturesOnlyTradePortion(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(2).
		City(0, "glassworks"). // economy owner produces 1 glass
		Tokens(0, "economy").
		Accessible("caravansery"). // 2c + G P
		ToMove(1).
		Coins(1, 20).
		State()

	id := advR1Card(t, "caravansery")
	pay := engine.BuildingPayment(&s, 1, id)
	if pay.PrintedCoins != 2 {
		t.Errorf("PrintedCoins = %d, want 2", pay.PrintedCoins)
	}
	if pay.TradeTotal != 5 {
		t.Errorf("TradeTotal = %d, want 5 (glass 3 + papyrus 2)", pay.TradeTotal)
	}
	if pay.Total != 7 {
		t.Errorf("Total = %d, want 7", pay.Total)
	}
	if pay.ToOpponent != 5 {
		t.Fatalf("ToOpponent = %d, want 5 (only the trading part, the printed coins stay with the bank)", pay.ToOpponent)
	}
}

// TestAdvEconomyR1_NoEconomyMeansBankKeepsTrade is the control for the above:
// without the Economy token, nothing is redirected to the opponent.
func TestAdvEconomyR1_NoEconomyMeansBankKeepsTrade(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(2).
		City(0, "glassworks").
		Accessible("caravansery").
		ToMove(1).
		Coins(1, 20).
		State()

	id := advR1Card(t, "caravansery")
	pay := engine.BuildingPayment(&s, 1, id)
	if pay.ToOpponent != 0 {
		t.Fatalf("ToOpponent = %d, want 0 (seat 0 does not own Economy)", pay.ToOpponent)
	}
	if pay.Total != 7 {
		t.Fatalf("Total = %d, want 7", pay.Total)
	}
}

// TestAdvEconomyR1_EconomyIncomeBeforeLooting encodes decisions.md
// "Effect order": payment (Economy income included) happens before shields,
// looting and any later loss, so an Economy owner with 0 coins can still lose
// a subsequent looting token out of the income they just received, rather
// than escaping the loss because they had nothing when it was applied.
// Walls (SS, 2 shields, red) is built by seat 1; seat 0 owns Economy and
// starts at 0 coins, with the pawn one step from the first looting zone.
func TestAdvEconomyR1_EconomyIncomeBeforeLooting(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(2).
		Tokens(0, "economy").
		Coins(0, 0).
		Pawn(-2). // one more shield toward seat 0's capital enters the 3-5 loot zone
		Accessible("walls").
		ToMove(1).
		Coins(1, 20).
		State()

	id := advR1Card(t, "walls")
	pay := engine.BuildingPayment(&s, 1, id)
	if pay.Total != 4 || pay.ToOpponent != 4 {
		t.Fatalf("Walls payment Total=%d ToOpponent=%d, want 4/4 (2 stone bought at price 2, all to seat 0)", pay.Total, pay.ToOpponent)
	}

	enginetest.Do(t, &s, "construct:walls")

	if s.Military != -4 {
		t.Fatalf("pawn = %d, want -4 (2 shields from -2)", s.Military)
	}
	if s.Players[0].Coins != 2 {
		t.Fatalf("seat 0 coins = %d, want 2 (received 4 from trading, then lost 2 to the looting token that was just entered)", s.Players[0].Coins)
	}
}

// TestAdvEconomyR1_EconomyIncomeBeforeAppianWayLoss is the wonder-side
// counterpart: wonders-tokens.md "The order of a wonder's effects" puts the
// payment (Economy income included) before "the Appian Way takes 3 coins
// from the opponent". Seat 1 builds the Appian Way; seat 0 owns Economy and
// starts at 0 coins.
func TestAdvEconomyR1_EconomyIncomeBeforeAppianWayLoss(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(1).
		Tokens(0, "economy").
		Coins(0, 0).
		Wonders(1, "appian-way", "pyramids", "colossus", "sphinx").
		Accessible("lumber-yard"). // tuck fodder
		ToMove(1).
		Coins(1, 100).
		State()

	wid := advR1Wonder(t, "appian-way")
	pay := engine.WonderPayment(&s, 1, wid)
	if pay.ToOpponent <= 0 {
		t.Fatalf("ToOpponent = %d, want a positive trade income to seat 0", pay.ToOpponent)
	}
	income := pay.ToOpponent

	enginetest.Do(t, &s, "wonder:appian-way:lumber-yard")

	want := income - 3
	if want < 0 {
		want = 0
	}
	if int16(s.Players[0].Coins) != want {
		t.Fatalf("seat 0 coins = %d, want %d (received %d from trading, then lost 3 to the Appian Way, floored at 0)",
			s.Players[0].Coins, want, income)
	}
}

// --- Discard value -----------------------------------------------------

// TestAdvEconomyR1_DiscardValueCountsOwnYellowCardsOnly encodes turn.md
// "2. Discard it for coins": "You gain 2 coins, and 1 more for every yellow
// card your city holds. The discarded card itself is not in your city, so a
// discarded yellow card does not count."
func TestAdvEconomyR1_DiscardValueCountsOwnYellowCardsOnly(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(1).
		City(0, "tavern", "brewery", "lumber-yard"). // 2 yellow, 1 brown
		Accessible("quarry").
		ToMove(0).
		State()

	if v := engine.DiscardValue(&s, 0); v != 4 {
		t.Fatalf("DiscardValue = %d, want 4 (2 + 2 yellow cards)", v)
	}

	before := s.Players[0].Coins
	evs := enginetest.Do(t, &s, "discard:quarry")
	gained := s.Players[0].Coins - before
	if gained != 4 {
		t.Fatalf("coins gained discarding = %d, want 4", gained)
	}
	found := false
	for _, e := range evs {
		if e.Kind == engine.EvDiscard {
			found = true
			if e.A != 4 {
				t.Fatalf("EvDiscard.A = %d, want 4", e.A)
			}
		}
	}
	if !found {
		t.Fatalf("no EvDiscard event found")
	}
}

// --- On-build coins: self scope versus the city with the most cards -------

// TestAdvEconomyR1_ChamberOfCommerceIsSelfScopeNotMaxCity encodes
// economy.md "Coin rewards": "Age III yellow cards count only your own city,
// after the card has entered it," contrasted with the guilds, which use the
// city with the most cards. The opponent is given more grey cards than the
// builder, which a max-city bug would wrongly use.
func TestAdvEconomyR1_ChamberOfCommerceIsSelfScopeNotMaxCity(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(3).
		City(0, "glassworks").                           // builder: 1 grey card
		City(1, "press", "glass-blower", "drying-room"). // opponent: 3 grey cards
		Accessible("chamber-of-commerce").               // cost PP
		ToMove(0).
		Coins(0, 20).
		State()

	id := advR1Card(t, "chamber-of-commerce")
	pay := engine.BuildingPayment(&s, 0, id)

	before := s.Players[0].Coins
	enginetest.Do(t, &s, "construct:chamber-of-commerce")
	delta := s.Players[0].Coins - before

	want := int16(3) - pay.Total // 3 coins per own grey card: just Glassworks
	if delta != want {
		t.Fatalf("coin delta = %d, want %d (reward 3, for the builder's own 1 grey card, not the opponent's 3)", delta, want)
	}
}

// TestAdvEconomyR1_ArenaCountsOwnWondersOnly encodes economy.md "Coin
// rewards": "Arena: 2 coins per wonder you have built", counted in the
// builder's own city (data-rulings.md's semantic audit flags Arena as one of
// the wonder/colour counters scoped to the builder alone). The opponent is
// given more built wonders, which a max-city bug would wrongly use.
func TestAdvEconomyR1_ArenaCountsOwnWondersOnly(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(3).
		Wonders(0, "pyramids", "colossus", "mausoleum", "sphinx").
		Built(0, "pyramids", "").
		Built(0, "colossus", ""). // builder: 2 built wonders
		Wonders(1, "statue-of-zeus", "appian-way", "great-lighthouse", "temple-of-artemis").
		Built(1, "statue-of-zeus", "").
		Built(1, "appian-way", "").
		Built(1, "great-lighthouse", ""). // opponent: 3 built wonders
		Accessible("arena").
		ToMove(0).
		Coins(0, 20).
		State()

	id := advR1Card(t, "arena")
	pay := engine.BuildingPayment(&s, 0, id)

	before := s.Players[0].Coins
	enginetest.Do(t, &s, "construct:arena")
	delta := s.Players[0].Coins - before

	want := int16(2*2) - pay.Total // 2 coins per own built wonder: 2 of them
	if delta != want {
		t.Fatalf("coin delta = %d, want %d (reward 4, for the builder's own 2 wonders, not the opponent's 3)", delta, want)
	}
}

// TestAdvEconomyR1_LighthouseCountsItself encodes economy.md "Coin rewards":
// "Lighthouse: 1 coin per yellow card you own, the Lighthouse included" (also
// data-rulings.md: "only the Lighthouse does [count itself], among the
// per-colour coin effects"). The builder already owns 2 other yellow cards.
func TestAdvEconomyR1_LighthouseCountsItself(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(3).
		City(0, "tavern", "brewery"). // builder already has 2 yellow cards
		Accessible("lighthouse").     // chain-in amphora, from Tavern: free
		ToMove(0).
		Coins(0, 10).
		State()

	id := advR1Card(t, "lighthouse")
	pay := engine.BuildingPayment(&s, 0, id)
	if !pay.ChainFree || pay.Total != 0 {
		t.Fatalf("Lighthouse payment ChainFree=%v Total=%d, want true/0 (Tavern's amphora chain)", pay.ChainFree, pay.Total)
	}

	before := s.Players[0].Coins
	enginetest.Do(t, &s, "construct:lighthouse")
	delta := s.Players[0].Coins - before

	if delta != 3 {
		t.Fatalf("coin delta = %d, want 3 (1 coin per yellow card, 3 after the Lighthouse enters its own city)", delta)
	}
}

// --- Guild coins: a single city, never two cities added together ----------

// TestAdvEconomyR1_ShipownersGuildCountsOneCityNotCombined encodes
// ages-scoring.md "Guilds": "Shipowners Guild: brown and grey cards together.
// 1 coin and 1 point per brown or grey card, counted in the single city with
// the most brown and grey cards combined. Brown cards from one city cannot be
// added to grey cards from the other." (rules-reference section 8 item 13.)
// The builder's own combined count (2) is lower than the opponent's (3); the
// reward must use the opponent's single-city count, never 2 + 3 = 5.
func TestAdvEconomyR1_ShipownersGuildCountsOneCityNotCombined(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(3).
		City(0, "lumber-yard", "stone-pit").             // builder: 2 brown, 0 grey
		City(1, "press", "glass-blower", "drying-room"). // opponent: 0 brown, 3 grey
		Accessible("shipowners-guild").                  // cost C S G P
		ToMove(0).
		Coins(0, 50).
		State()

	id := advR1Card(t, "shipowners-guild")
	pay := engine.BuildingPayment(&s, 0, id)

	before := s.Players[0].Coins
	enginetest.Do(t, &s, "construct:shipowners-guild")
	delta := s.Players[0].Coins - before

	want := int16(3) - pay.Total // max(2, 3) = 3, never 2 + 3 = 5
	if delta != want {
		t.Fatalf("coin delta = %d, want %d (reward 3, the higher single-city count, never the two cities added together)", delta, want)
	}
}

// TestAdvEconomyR1_MerchantsGuildCountsMaxCityEvenWhenOpponentsCity encodes
// ages-scoring.md "Guilds": "a guild looks at both cities... whichever city
// has the most of them, which may be the opponent's city." The opponent is
// given more yellow cards than the builder, so the reward must come from the
// opponent's count, not be capped at the builder's own city.
func TestAdvEconomyR1_MerchantsGuildCountsMaxCityEvenWhenOpponentsCity(t *testing.T) {
	s := enginetest.New(t, 1).
		Age(3).
		City(0, "tavern").                          // builder: 1 yellow
		City(1, "brewery", "forum", "caravansery"). // opponent: 3 yellow
		Accessible("merchants-guild").              // cost C W G P
		ToMove(0).
		Coins(0, 30).
		State()

	id := advR1Card(t, "merchants-guild")
	pay := engine.BuildingPayment(&s, 0, id)

	before := s.Players[0].Coins
	enginetest.Do(t, &s, "construct:merchants-guild")
	delta := s.Players[0].Coins - before

	want := int16(3) - pay.Total // max(1, 3) = 3, the opponent's city
	if delta != want {
		t.Fatalf("coin delta = %d, want %d (reward 3, from the opponent's larger yellow count)", delta, want)
	}
}
