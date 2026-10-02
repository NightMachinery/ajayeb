package engine

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// Tests for cost.go. The table cases follow the cost group of the engine
// test plan; cost_oracle_test.go compares cheapest with an exhaustive search.
//
// Every helper in this file starts with "cost" so that it cannot collide with
// the helpers of other slices in the same package.

// costTestCards, costTestWonders and costTestTokens are real items (ids,
// costs and effects as in the rules reference), enough to exercise the
// adapters while the full catalog is absent. costUseCatalog swaps them in.
func costTestCards() []catalog.CardDef {
	mc, fx := catalog.MustCost, catalog.Fx
	const (
		brown, grey, red, green = catalog.Brown, catalog.Grey, catalog.Red, catalog.Green
		yellow, blue, purple    = catalog.Yellow, catalog.Blue, catalog.Purple
		w, c, s, g, p           = catalog.Wood, catalog.Clay, catalog.Stone, catalog.Glass, catalog.Papyrus
	)
	prod := catalog.Produce
	return []catalog.CardDef{
		{ID: 1, Slug: "lumber-yard", Name: "Lumber Yard", Age: 1, Color: brown, Effects: fx(prod(w, 1))},
		{ID: 2, Slug: "stone-pit", Name: "Stone Pit", Age: 1, Color: brown, Cost: mc("1c"), Effects: fx(prod(s, 1))},
		{ID: 3, Slug: "clay-pool", Name: "Clay Pool", Age: 1, Color: brown, Effects: fx(prod(c, 1))},
		{ID: 4, Slug: "logging-camp", Name: "Logging Camp", Age: 1, Color: brown, Cost: mc("1c"), Effects: fx(prod(w, 1))},
		{ID: 5, Slug: "quarry", Name: "Quarry", Age: 1, Color: brown, Effects: fx(prod(s, 1))},
		{ID: 6, Slug: "clay-pit", Name: "Clay Pit", Age: 1, Color: brown, Cost: mc("1c"), Effects: fx(prod(c, 1))},
		{ID: 7, Slug: "glassworks", Name: "Glassworks", Age: 1, Color: grey, Cost: mc("1c"), Effects: fx(prod(g, 1))},
		{ID: 8, Slug: "press", Name: "Press", Age: 1, Color: grey, Cost: mc("1c"), Effects: fx(prod(p, 1))},
		{ID: 9, Slug: "stable", Name: "Stable", Age: 1, Color: red, Cost: mc("W"), ChainOut: catalog.ChainHorseshoe,
			Effects: fx(catalog.Shields(1))},
		{ID: 10, Slug: "garrison", Name: "Garrison", Age: 1, Color: red, Cost: mc("C"), ChainOut: catalog.ChainSword,
			Effects: fx(catalog.Shields(1))},
		{ID: 17, Slug: "tavern", Name: "Tavern", Age: 1, Color: yellow, ChainOut: catalog.ChainAmphora,
			Effects: fx(catalog.Coins(4))},
		{ID: 18, Slug: "stone-reserve", Name: "Stone Reserve", Age: 1, Color: yellow, Cost: mc("3c"),
			Effects: fx(catalog.TradeFix(s))},
		{ID: 19, Slug: "clay-reserve", Name: "Clay Reserve", Age: 1, Color: yellow, Cost: mc("3c"),
			Effects: fx(catalog.TradeFix(c))},
		{ID: 20, Slug: "wood-reserve", Name: "Wood Reserve", Age: 1, Color: yellow, Cost: mc("3c"),
			Effects: fx(catalog.TradeFix(w))},
		{ID: 21, Slug: "theater", Name: "Theater", Age: 1, Color: blue, ChainOut: catalog.ChainMask,
			Effects: fx(catalog.Points(3))},
		{ID: 23, Slug: "baths", Name: "Baths", Age: 1, Color: blue, Cost: mc("S"), ChainOut: catalog.ChainDrop,
			Effects: fx(catalog.Points(3))},
		{ID: 24, Slug: "sawmill", Name: "Sawmill", Age: 2, Color: brown, Cost: mc("2c"), Effects: fx(prod(w, 2))},
		{ID: 25, Slug: "shelf-quarry", Name: "Shelf Quarry", Age: 2, Color: brown, Cost: mc("2c"), Effects: fx(prod(s, 2))},
		{ID: 26, Slug: "brickyard", Name: "Brickyard", Age: 2, Color: brown, Cost: mc("2c"), Effects: fx(prod(c, 2))},
		{ID: 27, Slug: "glass-blower", Name: "Glass-Blower", Age: 2, Color: grey, Effects: fx(prod(g, 1))},
		{ID: 28, Slug: "drying-room", Name: "Drying Room", Age: 2, Color: grey, Effects: fx(prod(p, 1))},
		{ID: 29, Slug: "horse-breeders", Name: "Horse Breeders", Age: 2, Color: red, Cost: mc("C W"),
			ChainIn: catalog.ChainHorseshoe, Effects: fx(catalog.Shields(1))},
		{ID: 31, Slug: "walls", Name: "Walls", Age: 2, Color: red, Cost: mc("SS"), Effects: fx(catalog.Shields(2))},
		{ID: 35, Slug: "laboratory", Name: "Laboratory", Age: 2, Color: green, Cost: mc("W GG"),
			ChainOut: catalog.ChainLamp, Effects: fx(catalog.Science(catalog.SetSquare), catalog.Points(1))},
		{ID: 39, Slug: "forum", Name: "Forum", Age: 2, Color: yellow, Cost: mc("3c + C"),
			Effects: fx(catalog.ProduceChoice(g, p))},
		{ID: 40, Slug: "caravansery", Name: "Caravansery", Age: 2, Color: yellow, Cost: mc("2c + G P"),
			Effects: fx(catalog.ProduceChoice(w, c, s))},
		{ID: 41, Slug: "customs-house", Name: "Customs House", Age: 2, Color: yellow, Cost: mc("4c"),
			Effects: fx(catalog.TradeFix(g, p))},
		{ID: 43, Slug: "statue", Name: "Statue", Age: 2, Color: blue, Cost: mc("CC"), ChainIn: catalog.ChainMask,
			ChainOut: catalog.ChainColumn, Effects: fx(catalog.Points(4))},
		{ID: 45, Slug: "aqueduct", Name: "Aqueduct", Age: 2, Color: blue, Cost: mc("SSS"), ChainIn: catalog.ChainDrop,
			Effects: fx(catalog.Points(5))},
		{ID: 59, Slug: "lighthouse", Name: "Lighthouse", Age: 3, Color: yellow, Cost: mc("CC G"),
			ChainIn: catalog.ChainAmphora, Effects: fx(catalog.Points(3), catalog.CoinsPerCard(catalog.ScopeSelf, 1, yellow))},
		{ID: 61, Slug: "palace", Name: "Palace", Age: 3, Color: blue, Cost: mc("C S W GG"), Effects: fx(catalog.Points(7))},
		{ID: 64, Slug: "town-hall", Name: "Town Hall", Age: 3, Color: blue, Cost: mc("SSS WW"), Effects: fx(catalog.Points(7))},
		{ID: 69, Slug: "builders-guild", Name: "Builders Guild", Age: 3, Color: purple, Cost: mc("SS C W G"),
			Effects: fx(catalog.PointsPerWonder(catalog.ScopeMaxCity, 2))},
	}
}

func costTestWonders() []catalog.WonderDef {
	mc, fx := catalog.MustCost, catalog.Fx
	return []catalog.WonderDef{
		{ID: catalog.WonderPyramids, Slug: "pyramids", Name: "Pyramids", Cost: mc("SSS P"), Effects: fx(catalog.Points(9))},
		{ID: catalog.WonderColossus, Slug: "colossus", Name: "Colossus", Cost: mc("CCC G"),
			Effects: fx(catalog.Shields(2), catalog.Points(3))},
		{ID: catalog.WonderGreatLighthouse, Slug: "great-lighthouse", Name: "Great Lighthouse", Cost: mc("PP S W"),
			Effects: fx(catalog.ProduceChoice(catalog.Wood, catalog.Clay, catalog.Stone), catalog.Points(4))},
		{ID: catalog.WonderPiraeus, Slug: "piraeus", Name: "Piraeus", Cost: mc("WW C S"),
			Effects: fx(catalog.ProduceChoice(catalog.Glass, catalog.Papyrus), catalog.PlayAgain(), catalog.Points(2))},
		{ID: catalog.WonderSphinx, Slug: "sphinx", Name: "Sphinx", Cost: mc("S C GG"),
			Effects: fx(catalog.PlayAgain(), catalog.Points(6))},
	}
}

func costTestTokens() []catalog.TokenDef {
	fx := catalog.Fx
	return []catalog.TokenDef{
		{ID: catalog.TokenArchitecture, Slug: "architecture", Name: "Architecture",
			Effects: fx(catalog.Discount(catalog.ForWonders, 2))},
		{ID: catalog.TokenEconomy, Slug: "economy", Name: "Economy", Effects: fx(catalog.TradeIncome())},
		{ID: catalog.TokenMasonry, Slug: "masonry", Name: "Masonry", Effects: fx(catalog.Discount(catalog.ForBlue, 2))},
		{ID: catalog.TokenPhilosophy, Slug: "philosophy", Name: "Philosophy", Effects: fx(catalog.Points(7))},
		{ID: catalog.TokenUrbanism, Slug: "urbanism", Name: "Urbanism",
			Effects: fx(catalog.Coins(6), catalog.ChainBonus(4))},
	}
}

// costUseCatalog swaps in the test catalog until the test ends.
func costUseCatalog(tb testing.TB) {
	tb.Helper()
	tb.Cleanup(catalog.ReplaceForTest(costTestCards(), costTestWonders(), costTestTokens()))
}

// costCardID, costWonderID and costTokenID look an item up by slug.
func costCardID(tb testing.TB, slug string) catalog.CardID {
	tb.Helper()
	id, ok := catalog.CardBySlug(slug)
	if !ok {
		tb.Fatalf("no card %q in the test catalog", slug)
	}
	return id
}

func costWonderID(tb testing.TB, slug string) catalog.WonderID {
	tb.Helper()
	id, ok := catalog.WonderBySlug(slug)
	if !ok {
		tb.Fatalf("no wonder %q in the test catalog", slug)
	}
	return id
}

func costTokenID(tb testing.TB, slug string) catalog.TokenID {
	tb.Helper()
	id, ok := catalog.TokenBySlug(slug)
	if !ok {
		tb.Fatalf("no token %q in the test catalog", slug)
	}
	return id
}

// costRecompute rebuilds the caches of p that cost.go reads (Prod, Choice,
// NChoice, TradeFix, ColorN, Chains) from its city and built wonders, as
// city.go maintains them. The science counts are not needed here.
func costRecompute(tb testing.TB, p *Player) {
	tb.Helper()
	p.Prod, p.Choice, p.NChoice, p.TradeFix = [catalog.NumResources]uint8{}, [MaxChoice]catalog.ResMask{}, 0, 0
	p.ColorN, p.Chains = [catalog.NumColors]uint8{}, 0
	addChoice := func(m catalog.ResMask) {
		if p.NChoice >= MaxChoice {
			tb.Fatalf("more than %d flexible producers", MaxChoice)
			return
		}
		p.Choice[p.NChoice] = m
		p.NChoice++
	}
	for id := range p.City.All() {
		d := catalog.Card(id)
		if d == nil {
			tb.Fatalf("card %d is not registered", id)
			return
		}
		p.ColorN[d.Color]++
		p.Chains |= d.ChainOut.Bit()
		for _, e := range d.Effects {
			switch e.Kind {
			case catalog.EffProduce:
				p.Prod[e.Res] += e.N
			case catalog.EffProduceChoice:
				addChoice(e.Mask)
			case catalog.EffTradeFix:
				p.TradeFix |= e.Mask
			}
		}
	}
	for _, ws := range p.Wonders {
		if !ws.Built {
			continue
		}
		d := catalog.Wonder(ws.ID)
		if d == nil {
			tb.Fatalf("wonder %d is not registered", ws.ID)
			return
		}
		if e, ok := catalog.FindEffect(d.Effects, catalog.EffProduceChoice); ok {
			addChoice(e.Mask)
		}
	}
}

// costGive adds the named cards to seat's city.
func costGive(tb testing.TB, s *State, seat Seat, slugs ...string) {
	tb.Helper()
	p := &s.Players[seat]
	for _, slug := range slugs {
		p.City.Add(costCardID(tb, slug))
	}
	costRecompute(tb, p)
}

// costTake removes the named card from seat's city, as a destroy would.
func costTake(tb testing.TB, s *State, seat Seat, slug string) {
	tb.Helper()
	p := &s.Players[seat]
	id := costCardID(tb, slug)
	if !p.City.Has(id) {
		tb.Fatalf("seat %d has no %s to take", seat, slug)
	}
	p.City.Remove(id)
	costRecompute(tb, p)
}

// costBuild gives seat the named wonders, built, in its first free slots.
func costBuild(tb testing.TB, s *State, seat Seat, slugs ...string) {
	tb.Helper()
	p := &s.Players[seat]
	for _, slug := range slugs {
		placed := false
		for i := range p.Wonders {
			if p.Wonders[i].ID == 0 {
				p.Wonders[i] = WonderSlot{ID: costWonderID(tb, slug), Built: true}
				placed = true
				break
			}
		}
		if !placed {
			tb.Fatalf("seat %d has no free wonder slot for %s", seat, slug)
		}
	}
	costRecompute(tb, p)
}

// costTokens gives seat the named progress tokens.
func costTokens(tb testing.TB, s *State, seat Seat, slugs ...string) {
	tb.Helper()
	for _, slug := range slugs {
		s.Players[seat].Tokens.Add(costTokenID(tb, slug))
	}
}

// costRes returns the resource units of a cost written in the reference
// notation; costIn uses it for production as well as for costs.
func costRes(s string) [catalog.NumResources]uint8 { return catalog.MustCost(s).Res }

func TestCheapestTable(t *testing.T) {
	raw, man := catalog.RawGoods, catalog.Manufactured
	cases := []struct {
		name         string
		in           costInput
		total, trade int16
		toOpp        int16
		affordable   bool
		check        func(t *testing.T, p *Payment)
	}{
		{name: "free card at 0 coins is affordable", in: costInput{}, affordable: true},
		{name: "coin-only cost is unaffordable at 0 coins", in: costInput{printed: 1}, total: 1},
		{name: "coin-only cost is affordable with exactly its coins", in: costInput{printed: 1, purse: 1}, total: 1, affordable: true},
		{
			name: "opponent with Clay Pool and Brickyard makes clay cost 5",
			in:   costInput{purse: 20, need: costRes("C"), oppProd: costRes("CCC")}, total: 5, trade: 5, affordable: true,
			check: func(t *testing.T, p *Payment) {
				if p.UnitPrice != [5]uint8{2, 5, 2, 2, 2} || p.TradeUnits != costRes("C") {
					t.Errorf("prices %v, bought %v", p.UnitPrice, p.TradeUnits)
				}
			},
		},
		{
			name: "two clay against 3 clay costs 10",
			in:   costInput{purse: 10, need: costRes("CC"), oppProd: costRes("CCC")}, total: 10, trade: 10, affordable: true,
		},
		{
			name:  "Clay Reserve fixes clay at 1 whatever the opponent makes",
			in:    costInput{purse: 2, need: costRes("CC"), oppProd: costRes("CCC"), fix: catalog.Clay.Mask()},
			total: 2, trade: 2, affordable: true,
			check: func(t *testing.T, p *Payment) {
				if p.FixedPrice != catalog.Clay.Mask() || p.UnitPrice[catalog.Clay] != 1 {
					t.Errorf("fixed %b, clay price %d", p.FixedPrice, p.UnitPrice[catalog.Clay])
				}
			},
		},
		{
			name: "Customs House fixes glass and papyrus at 1",
			in: costInput{purse: 9, need: costRes("W GG"), oppProd: costRes("W GG"),
				fix: catalog.MaskOf(catalog.Glass, catalog.Papyrus)},
			total: 5, trade: 5, affordable: true, // wood 3, glass 1 + 1
		},
		{
			name:  "own production covers first",
			in:    costInput{purse: 9, need: costRes("WW G"), own: costRes("W G"), oppProd: costRes("W")},
			total: 3, trade: 3, affordable: true,
			check: func(t *testing.T, p *Payment) {
				if p.Own != costRes("W G") || p.TradeUnits != costRes("W") {
					t.Errorf("own %v, bought %v", p.Own, p.TradeUnits)
				}
			},
		},
		{
			name: "own production beyond the need is not used",
			in:   costInput{purse: 0, need: costRes("S"), own: costRes("SSS WW")}, total: 0, affordable: true,
			check: func(t *testing.T, p *Payment) {
				if p.Own != costRes("S") {
					t.Errorf("own %v, want 1 stone", p.Own)
				}
			},
		},
		{
			name: "Forum covers the dearer unit",
			in: costInput{purse: 9, printed: 2, need: costRes("G P"), oppProd: costRes("G"),
				choice: [MaxChoice]catalog.ResMask{man}, nChoice: 1},
			total: 4, trade: 2, affordable: true, // the Forum takes the 3-coin glass, papyrus is bought for 2
			check: func(t *testing.T, p *Payment) {
				if p.NChoice != 1 || p.Choice[0] != (ChoiceUse{Mask: man, Res: catalog.Glass}) {
					t.Errorf("choice %v (%d)", p.Choice, p.NChoice)
				}
			},
		},
		{
			name:  "a producer never covers the other class",
			in:    costInput{purse: 9, need: costRes("G"), choice: [MaxChoice]catalog.ResMask{raw}, nChoice: 1},
			total: 2, trade: 2, affordable: true,
			check: func(t *testing.T, p *Payment) {
				if p.NChoice != 0 {
					t.Errorf("an idle producer was recorded: %v", p.Choice)
				}
			},
		},
		{
			name: "two raw producers cover the two dearest raw units",
			in: costInput{purse: 9, need: costRes("C SS"), oppProd: costRes("S"),
				choice: [MaxChoice]catalog.ResMask{raw, man, raw}, nChoice: 3},
			total: 2, trade: 2, affordable: true,
			check: func(t *testing.T, p *Payment) {
				if p.NChoice != 2 || p.Choice[0].Res != catalog.Stone || p.Choice[1].Res != catalog.Stone {
					t.Errorf("choice %v (%d)", p.Choice, p.NChoice)
				}
			},
		},
		{
			name:       "a producer is not used for a unit own production covers",
			in:         costInput{need: costRes("G"), own: costRes("G"), choice: [MaxChoice]catalog.ResMask{man}, nChoice: 1},
			affordable: true,
			check: func(t *testing.T, p *Payment) {
				if p.NChoice != 0 || p.Own != costRes("G") {
					t.Errorf("own %v, choice %v (%d)", p.Own, p.Choice, p.NChoice)
				}
			},
		},
		{
			name:  "ties go to the lower resource index",
			in:    costInput{purse: 9, need: costRes("W C"), choice: [MaxChoice]catalog.ResMask{raw}, nChoice: 1},
			total: 2, trade: 2, affordable: true,
			check: func(t *testing.T, p *Payment) {
				if p.Choice[0].Res != catalog.Wood || p.TradeUnits != costRes("C") {
					t.Errorf("choice %v, bought %v", p.Choice[0], p.TradeUnits)
				}
			},
		},
		// The Palace (C S W GG) with own clay and stone against an opponent
		// making wood and glass: 9 coins, 3 with Masonry, 0 with a Forum too.
		{
			name:  "Palace without Masonry",
			in:    costInput{purse: 9, need: costRes("C S W GG"), own: costRes("C S"), oppProd: costRes("W G")},
			total: 9, trade: 9, affordable: true,
		},
		{
			name: "Palace with Masonry",
			in: costInput{purse: 9, need: costRes("C S W GG"), own: costRes("C S"), oppProd: costRes("W G"),
				discount: 2},
			total: 3, trade: 3, affordable: true,
			check: func(t *testing.T, p *Payment) {
				if p.Discount != costRes("W G") || p.TradeUnits != costRes("G") {
					t.Errorf("discount %v, bought %v", p.Discount, p.TradeUnits)
				}
			},
		},
		{
			name: "Palace with Masonry and Forum",
			in: costInput{purse: 0, need: costRes("C S W GG"), own: costRes("C S"), oppProd: costRes("W G"),
				discount: 2, choice: [MaxChoice]catalog.ResMask{man}, nChoice: 1},
			total: 0, affordable: true,
		},
		// The example of docs/rules/economy.md: no own production, the
		// opponent makes one stone and one glass: 13, 7 with Masonry, 4 with
		// Masonry and a Forum.
		{
			name:  "docs example: Palace",
			in:    costInput{purse: 13, need: costRes("C S W GG"), oppProd: costRes("S G")},
			total: 13, trade: 13, affordable: true,
		},
		{
			name:  "docs example: Palace with Masonry",
			in:    costInput{purse: 13, need: costRes("C S W GG"), oppProd: costRes("S G"), discount: 2},
			total: 7, trade: 7, affordable: true,
			check: func(t *testing.T, p *Payment) {
				if p.Discount != costRes("S G") || p.TradeUnits != costRes("C W G") {
					t.Errorf("discount %v, bought %v", p.Discount, p.TradeUnits)
				}
			},
		},
		{
			name: "docs example: Palace with Masonry and Forum",
			in: costInput{purse: 13, need: costRes("C S W GG"), oppProd: costRes("S G"), discount: 2,
				choice: [MaxChoice]catalog.ResMask{man}, nChoice: 1},
			total: 4, trade: 4, affordable: true,
			check: func(t *testing.T, p *Payment) {
				if p.Choice[0].Res != catalog.Glass || p.Discount != costRes("S G") || p.TradeUnits != costRes("C W") {
					t.Errorf("choice %v, discount %v, bought %v", p.Choice[0], p.Discount, p.TradeUnits)
				}
			},
		},
		{
			name:  "a discount larger than the need is not refunded",
			in:    costInput{purse: 0, need: costRes("S"), discount: 2},
			total: 0, affordable: true,
			check: func(t *testing.T, p *Payment) {
				if p.Discount != costRes("S") || p.Total != 0 || p.TradeTotal != 0 {
					t.Errorf("discount %v, total %d, trade %d", p.Discount, p.Total, p.TradeTotal)
				}
			},
		},
		{
			name:  "a discount never reduces printed coins",
			in:    costInput{purse: 9, printed: 3, need: costRes("C"), discount: 2},
			total: 3, affordable: true,
		},
		{
			name:       "chain with Urbanism at 0 coins",
			in:         costInput{need: costRes("SSS"), printed: 4, chainFree: true, chainBonus: 4},
			affordable: true,
			check: func(t *testing.T, p *Payment) {
				if !p.ChainFree || p.UrbanismBonus != 4 || p.PrintedCoins != 0 || p.Own != [5]uint8{} {
					t.Errorf("chain payment %+v", *p)
				}
			},
		},
		{
			name:       "chain without Urbanism pays no bonus",
			in:         costInput{need: costRes("SSS"), chainFree: true},
			affordable: true,
			check: func(t *testing.T, p *Payment) {
				if !p.ChainFree || p.UrbanismBonus != 0 {
					t.Errorf("chain payment %+v", *p)
				}
			},
		},
		{
			name: "Urbanism pays nothing on a build that is not chain-free",
			in:   costInput{need: costRes("S"), own: costRes("S"), chainBonus: 4}, affordable: true,
			check: func(t *testing.T, p *Payment) {
				if p.UrbanismBonus != 0 || p.ChainFree {
					t.Errorf("payment %+v", *p)
				}
			},
		},
		{
			name: "Economy with Caravansery: pay 7, opponent gets 5, bank 2",
			in: costInput{purse: 7, printed: 2, need: costRes("G P"), oppProd: costRes("G"),
				tradeIncome: true},
			total: 7, trade: 5, toOpp: 5, affordable: true,
		},
		{
			name: "Economy at the 1-coin fixed price",
			in: costInput{purse: 9, need: costRes("CC"), oppProd: costRes("CCC"), fix: catalog.Clay.Mask(),
				tradeIncome: true},
			total: 2, trade: 2, toOpp: 2, affordable: true,
		},
		{
			name:  "Economy gets nothing when nothing is bought",
			in:    costInput{purse: 9, printed: 3, tradeIncome: true},
			total: 3, affordable: true,
		},
		{
			name: "total equal to the coins is affordable",
			in:   costInput{purse: 6, need: costRes("SS"), oppProd: costRes("S")}, total: 6, trade: 6, affordable: true,
		},
		{
			name: "total one above the coins is not",
			in:   costInput{purse: 5, need: costRes("SS"), oppProd: costRes("S")}, total: 6, trade: 6,
		},
		{
			name: "a destroyed brown card lowers the price at once",
			in:   costInput{purse: 9, need: costRes("C"), oppProd: costRes("C")}, total: 3, trade: 3, affordable: true,
		},
		{
			name: "NChoice above MaxChoice is clamped",
			in: costInput{purse: 9, need: costRes("W"), choice: [MaxChoice]catalog.ResMask{man, man, man, raw},
				nChoice: 9},
			affordable: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var p Payment
			total, trade := cheapest(&c.in, &p)
			if total != c.total || trade != c.trade {
				t.Errorf("cheapest = (%d, %d), want (%d, %d)", total, trade, c.total, c.trade)
			}
			if p.Total != total || p.TradeTotal != trade || p.ToOpponent != c.toOpp || p.Affordable != c.affordable {
				t.Errorf("payment total %d, trade %d, to opponent %d, affordable %v; want %d, %d, %d, %v",
					p.Total, p.TradeTotal, p.ToOpponent, p.Affordable, c.total, c.trade, c.toOpp, c.affordable)
			}
			if t2, tr2 := cheapest(&c.in, nil); t2 != total || tr2 != trade {
				t.Errorf("without a breakdown: (%d, %d), with one: (%d, %d)", t2, tr2, total, trade)
			}
			if q := quote(&c.in); q != p.Quote() {
				t.Errorf("quote %+v, payment says %+v", q, p.Quote())
			}
			for _, bad := range costPaymentProblems(&c.in, &p) {
				t.Error(bad)
			}
			if c.check != nil {
				c.check(t, &p)
			}
		})
	}
}

// One payment pinned field by field, so a change to the breakdown format is
// a deliberate one.
func TestCheapestBreakdownPinned(t *testing.T) {
	in := costInput{
		purse: 10, printed: 2,
		need: costRes("C S W GG"), own: costRes("W"), oppProd: costRes("S GG"), fix: catalog.Clay.Mask(),
		choice: [MaxChoice]catalog.ResMask{catalog.Manufactured, catalog.RawGoods}, nChoice: 2,
		discount: 2, tradeIncome: true,
	}
	var got Payment
	cheapest(&in, &got)
	// Prices: wood 2, clay 1 (fixed), stone 3, glass 4, papyrus 2. Own wood
	// covers the wood; the manufactured producer takes a glass (4) and the
	// raw producer the stone (3); the discount waives the other glass (4)
	// and the clay (1); nothing is left to buy.
	want := Payment{
		Affordable:   true,
		PrintedCoins: 2,
		Own:          costRes("W"),
		Choice: [MaxChoice]ChoiceUse{
			{Mask: catalog.Manufactured, Res: catalog.Glass},
			{Mask: catalog.RawGoods, Res: catalog.Stone},
		},
		NChoice:    2,
		Discount:   costRes("C G"),
		UnitPrice:  [5]uint8{2, 1, 3, 4, 2},
		FixedPrice: catalog.Clay.Mask(),
		Total:      2,
	}
	if got != want {
		t.Errorf("payment\n got %+v\nwant %+v", got, want)
	}
}

func TestUnitPrices(t *testing.T) {
	opp := costRes("WWWW CC G")
	got := unitPrices(catalog.Clay.Mask()|catalog.Papyrus.Mask(), &opp)
	if want := [5]uint8{6, 1, 2, 3, 1}; got != want {
		t.Errorf("unitPrices = %v, want %v", got, want)
	}
	var huge [catalog.NumResources]uint8
	for r := range huge {
		huge[r] = 255
	}
	if got := unitPrices(0, &huge); got != [5]uint8{255, 255, 255, 255, 255} {
		t.Errorf("unitPrices does not saturate: %v", got)
	}
}

// Large inputs saturate instead of wrapping around.
func TestCheapestSaturates(t *testing.T) {
	in := costInput{printed: 255}
	for r := range catalog.NumResources {
		in.need[r] = 255
		in.oppProd[r] = 255
	}
	total, trade := cheapest(&in, nil)
	if total != maxInt16 || trade != maxInt16 {
		t.Errorf("cheapest = (%d, %d), want both %d", total, trade, maxInt16)
	}
}

func TestBuildingPaymentRules(t *testing.T) {
	costUseCatalog(t)
	type setup func(t *testing.T, s *State)
	cases := []struct {
		name   string
		setup  setup
		card   string // built by seat 0
		total  int16
		toOpp  int16
		afford bool
		check  func(t *testing.T, s *State, p *Payment)
	}{
		{
			name: "coin-only card is unaffordable at 0 coins", card: "stone-pit", total: 1,
			setup: func(t *testing.T, s *State) { s.Players[0].Coins = 0 },
		},
		{
			name: "coin-only card is affordable with its coins", card: "stone-pit", total: 1, afford: true,
			setup: func(t *testing.T, s *State) { s.Players[0].Coins = 1 },
		},
		{
			name: "opponent with Clay Pool and Brickyard makes clay cost 5", card: "garrison", total: 5, afford: true,
			setup: func(t *testing.T, s *State) { costGive(t, s, 1, "clay-pool", "brickyard") },
		},
		{
			name: "the opponent's Caravansery does not raise prices", card: "garrison", total: 2, afford: true,
			setup: func(t *testing.T, s *State) { costGive(t, s, 1, "caravansery") },
		},
		{
			name: "the opponent's Great Lighthouse does not raise prices", card: "garrison", total: 2, afford: true,
			setup: func(t *testing.T, s *State) { costBuild(t, s, 1, "great-lighthouse") },
		},
		{
			name: "my own production does not change my price", card: "walls", total: 3, afford: true,
			setup: func(t *testing.T, s *State) {
				costGive(t, s, 0, "quarry")
				costGive(t, s, 1, "stone-pit")
			},
		},
		{
			name: "Clay Reserve gives clay at 1", card: "garrison", total: 1, afford: true,
			setup: func(t *testing.T, s *State) {
				costGive(t, s, 0, "clay-reserve")
				costGive(t, s, 1, "clay-pool", "brickyard")
			},
			check: func(t *testing.T, s *State, p *Payment) {
				if p.FixedPrice != catalog.Clay.Mask() || p.UnitPrice[catalog.Clay] != 1 {
					t.Errorf("fixed %b, prices %v", p.FixedPrice, p.UnitPrice)
				}
			},
		},
		{
			name: "the opponent's Clay Reserve does not fix my price", card: "garrison", total: 2, afford: true,
			setup: func(t *testing.T, s *State) { costGive(t, s, 1, "clay-reserve") },
		},
		{
			name: "Customs House gives glass and papyrus at 1", card: "caravansery", total: 4, afford: true,
			setup: func(t *testing.T, s *State) {
				costGive(t, s, 0, "customs-house")
				costGive(t, s, 1, "glassworks", "glass-blower", "press")
			},
		},
		{
			name: "Forum covers the dearer unit", card: "caravansery", total: 4, afford: true,
			setup: func(t *testing.T, s *State) {
				costGive(t, s, 0, "forum")
				costGive(t, s, 1, "glassworks")
			},
			check: func(t *testing.T, s *State, p *Payment) {
				want := ChoiceUse{Card: costCardID(t, "forum"), Mask: catalog.Manufactured, Res: catalog.Glass}
				if p.NChoice != 1 || p.Choice[0] != want || p.TradeUnits != costRes("P") {
					t.Errorf("choice %v (%d), bought %v", p.Choice, p.NChoice, p.TradeUnits)
				}
			},
		},
		{
			name: "Masonry and Palace", card: "palace", total: 7, afford: true,
			setup: func(t *testing.T, s *State) {
				costTokens(t, s, 0, "masonry")
				costGive(t, s, 1, "quarry", "glassworks")
			},
			check: func(t *testing.T, s *State, p *Payment) {
				if p.DiscountBy != catalog.TokenMasonry || p.Discount != costRes("S G") {
					t.Errorf("discount %v by %v", p.Discount, p.DiscountBy)
				}
			},
		},
		{
			name: "Masonry, Forum and Palace", card: "palace", total: 4, afford: true,
			setup: func(t *testing.T, s *State) {
				costTokens(t, s, 0, "masonry")
				costGive(t, s, 0, "forum")
				costGive(t, s, 1, "quarry", "glassworks")
			},
		},
		{
			name: "Masonry, Forum and Palace with own clay and stone", card: "palace", total: 0, afford: true,
			setup: func(t *testing.T, s *State) {
				s.Players[0].Coins = 0
				costTokens(t, s, 0, "masonry")
				costGive(t, s, 0, "forum", "clay-pool", "quarry")
				costGive(t, s, 1, "lumber-yard", "glassworks")
			},
		},
		{
			name: "Masonry is ignored on a non-blue card", card: "walls", total: 6, afford: true,
			setup: func(t *testing.T, s *State) {
				costTokens(t, s, 0, "masonry")
				costGive(t, s, 1, "quarry")
			},
			check: func(t *testing.T, s *State, p *Payment) {
				if p.DiscountBy != 0 || p.Discount != [5]uint8{} {
					t.Errorf("discount %v by %v on a red card", p.Discount, p.DiscountBy)
				}
			},
		},
		{
			name: "Architecture is ignored on a building", card: "palace", total: 10, afford: true,
			setup: func(t *testing.T, s *State) { costTokens(t, s, 0, "architecture") },
		},
		{
			name: "the opponent's Masonry does nothing for me", card: "palace", total: 10, afford: true,
			setup: func(t *testing.T, s *State) { costTokens(t, s, 1, "masonry") },
		},
		{
			name: "Masonry with nothing left to waive", card: "baths", total: 0, afford: true,
			setup: func(t *testing.T, s *State) {
				s.Players[0].Coins = 0
				costTokens(t, s, 0, "masonry")
				costGive(t, s, 0, "quarry")
			},
			check: func(t *testing.T, s *State, p *Payment) {
				if p.DiscountBy != catalog.TokenMasonry || p.Discount != [5]uint8{} {
					t.Errorf("discount %v by %v", p.Discount, p.DiscountBy)
				}
			},
		},
		{
			name: "chain and Urbanism at 0 coins", card: "aqueduct", total: 0, afford: true,
			setup: func(t *testing.T, s *State) {
				s.Players[0].Coins = 0
				costGive(t, s, 0, "baths")
				costTokens(t, s, 0, "urbanism")
				costGive(t, s, 1, "shelf-quarry")
			},
			check: func(t *testing.T, s *State, p *Payment) {
				if !p.ChainFree || p.ChainFrom != costCardID(t, "baths") || p.UrbanismBonus != 4 {
					t.Errorf("chain %v from %v, bonus %d", p.ChainFree, p.ChainFrom, p.UrbanismBonus)
				}
			},
		},
		{
			name: "chain without Urbanism", card: "lighthouse", total: 0, afford: true,
			setup: func(t *testing.T, s *State) {
				s.Players[0].Coins = 0
				costGive(t, s, 0, "tavern")
				costTokens(t, s, 0, "masonry")
			},
			check: func(t *testing.T, s *State, p *Payment) {
				if !p.ChainFree || p.ChainFrom != costCardID(t, "tavern") || p.UrbanismBonus != 0 || p.DiscountBy != 0 {
					t.Errorf("payment %+v", *p)
				}
			},
		},
		{
			name: "the opponent's chain card does not chain for me", card: "aqueduct", total: 6, afford: true,
			setup: func(t *testing.T, s *State) {
				costGive(t, s, 1, "baths")
				costTokens(t, s, 0, "urbanism")
			},
			check: func(t *testing.T, s *State, p *Payment) {
				if p.ChainFree || p.UrbanismBonus != 0 {
					t.Errorf("payment %+v", *p)
				}
			},
		},
		{
			name: "Urbanism does not pay for a card that is free by cost", card: "theater", total: 0, afford: true,
			setup: func(t *testing.T, s *State) { costTokens(t, s, 0, "urbanism") },
			check: func(t *testing.T, s *State, p *Payment) {
				if p.ChainFree || p.UrbanismBonus != 0 {
					t.Errorf("payment %+v", *p)
				}
			},
		},
		{
			name: "chain wins over Masonry", card: "statue", total: 0, afford: true,
			setup: func(t *testing.T, s *State) {
				costGive(t, s, 0, "theater")
				costTokens(t, s, 0, "masonry")
			},
			check: func(t *testing.T, s *State, p *Payment) {
				if !p.ChainFree || p.Discount != [5]uint8{} || p.DiscountBy != 0 {
					t.Errorf("payment %+v", *p)
				}
			},
		},
		{
			name: "Economy with Caravansery: pay 7, opponent gets 5", card: "caravansery", total: 7, toOpp: 5, afford: true,
			setup: func(t *testing.T, s *State) {
				s.Players[0].Coins = 7
				costTokens(t, s, 1, "economy")
				costGive(t, s, 1, "glassworks")
			},
		},
		{
			name: "my own Economy does not redirect my payment", card: "caravansery", total: 6, afford: true,
			setup: func(t *testing.T, s *State) { costTokens(t, s, 0, "economy") },
		},
		{
			name: "Economy at the 1-coin fixed price", card: "garrison", total: 1, toOpp: 1, afford: true,
			setup: func(t *testing.T, s *State) {
				costGive(t, s, 0, "clay-reserve")
				costTokens(t, s, 1, "economy")
				costGive(t, s, 1, "clay-pool", "brickyard")
			},
		},
		{
			name: "total equal to the coins is affordable", card: "walls", total: 6, afford: true,
			setup: func(t *testing.T, s *State) {
				s.Players[0].Coins = 6
				costGive(t, s, 1, "quarry")
			},
		},
		{
			name: "total one above the coins is not", card: "walls", total: 6,
			setup: func(t *testing.T, s *State) {
				s.Players[0].Coins = 5
				costGive(t, s, 1, "quarry")
			},
		},
		{
			name: "a destroyed brown card lowers the price at once", card: "garrison", total: 3, afford: true,
			setup: func(t *testing.T, s *State) {
				costGive(t, s, 1, "clay-pool", "brickyard")
				if p := BuildingPayment(s, 0, costCardID(t, "garrison")); p.Total != 5 {
					t.Fatalf("before the destroy: %d, want 5", p.Total)
				}
				costTake(t, s, 1, "brickyard")
			},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var s State
			s.Players[0].Coins, s.Players[1].Coins = 20, 20
			if c.setup != nil {
				c.setup(t, &s)
			}
			id := costCardID(t, c.card)
			p := BuildingPayment(&s, 0, id)
			if p.Total != c.total || p.ToOpponent != c.toOpp || p.Affordable != c.afford {
				t.Errorf("%s: total %d, to opponent %d, affordable %v; want %d, %d, %v\n%+v",
					c.card, p.Total, p.ToOpponent, p.Affordable, c.total, c.toOpp, c.afford, p)
			}
			if q := BuildingQuote(&s, 0, id); q != p.Quote() {
				t.Errorf("quote %+v, payment says %+v", q, p.Quote())
			}
			if c.check != nil {
				c.check(t, &s, &p)
			}
		})
	}
}

func TestWonderPaymentRules(t *testing.T) {
	costUseCatalog(t)
	pyramids := costWonderID(t, "pyramids")
	newState := func() State {
		var s State
		s.Players[0].Coins, s.Players[1].Coins = 20, 20
		costGive(t, &s, 1, "quarry") // stone costs seat 0 three, papyrus two
		return s
	}

	s := newState()
	if p := WonderPayment(&s, 0, pyramids); p.Total != 11 || p.DiscountBy != 0 {
		t.Errorf("Pyramids alone: %d by %v, want 11", p.Total, p.DiscountBy)
	}

	s = newState()
	costBuild(t, &s, 0, "great-lighthouse")
	if p := WonderPayment(&s, 0, pyramids); p.Total != 8 {
		t.Errorf("Pyramids with Great Lighthouse: %d, want 8", p.Total)
	}

	s = newState()
	costTokens(t, &s, 0, "architecture")
	if p := WonderPayment(&s, 0, pyramids); p.Total != 5 || p.DiscountBy != catalog.TokenArchitecture {
		t.Errorf("Pyramids with Architecture: %d by %v, want 5", p.Total, p.DiscountBy)
	}

	// Architecture and the Great Lighthouse: the Lighthouse covers a stone,
	// Architecture waives the other two, and only the papyrus is bought.
	s = newState()
	costTokens(t, &s, 0, "architecture")
	costBuild(t, &s, 0, "great-lighthouse")
	p := WonderPayment(&s, 0, pyramids)
	wantUse := ChoiceUse{Wonder: catalog.WonderGreatLighthouse, Mask: catalog.RawGoods, Res: catalog.Stone}
	if p.Total != 2 || p.NChoice != 1 || p.Choice[0] != wantUse || p.Discount != costRes("SS") ||
		p.TradeUnits != costRes("P") || p.DiscountBy != catalog.TokenArchitecture {
		t.Errorf("Pyramids with Architecture and Great Lighthouse: %+v", p)
	}
	if q := WonderQuote(&s, 0, pyramids); q != p.Quote() {
		t.Errorf("quote %+v, payment says %+v", q, p.Quote())
	}

	s = newState()
	costTokens(t, &s, 0, "masonry")
	if p := WonderPayment(&s, 0, pyramids); p.Total != 11 || p.DiscountBy != 0 {
		t.Errorf("Masonry applied to a wonder: %d by %v", p.Total, p.DiscountBy)
	}

	// Wonders never chain, whatever chain symbols the city holds.
	s = newState()
	s.Players[0].Chains = ^uint32(0)
	costTokens(t, &s, 0, "urbanism")
	if p := WonderPayment(&s, 0, pyramids); p.ChainFree || p.Total != 11 || p.UrbanismBonus != 0 {
		t.Errorf("a wonder was chain-free: %+v", p)
	}

	// An unbuilt wonder of my own produces nothing for me.
	s = newState()
	s.Players[0].Wonders[0] = WonderSlot{ID: catalog.WonderGreatLighthouse}
	costRecompute(t, &s.Players[0])
	if p := WonderPayment(&s, 0, pyramids); p.Total != 11 {
		t.Errorf("an unbuilt Great Lighthouse produced: %d, want 11", p.Total)
	}

	// Economy for a wonder: the trading part goes to the opponent.
	s = newState()
	costTokens(t, &s, 1, "economy")
	if p := WonderPayment(&s, 0, pyramids); p.Total != 11 || p.ToOpponent != 11 {
		t.Errorf("Economy on a wonder: total %d, to opponent %d, want 11 and 11", p.Total, p.ToOpponent)
	}
}

// Two flexible producers of one class are named by their own sources, and
// producers of the other player never count.
func TestChoiceSourcesNamed(t *testing.T) {
	costUseCatalog(t)
	var s State
	s.Players[0].Coins = 20
	costGive(t, &s, 0, "caravansery")
	costBuild(t, &s, 0, "great-lighthouse", "piraeus")
	costGive(t, &s, 1, "forum", "shelf-quarry")
	// Town Hall: SSS WW. Stone costs 4, wood 2: the two raw producers take
	// two stones, the rest is bought (one stone and two wood).
	p := BuildingPayment(&s, 0, costCardID(t, "town-hall"))
	if p.Total != 8 || p.NChoice != 2 {
		t.Fatalf("Town Hall: %+v", p)
	}
	got := [2]ChoiceUse{p.Choice[0], p.Choice[1]}
	want := [2]ChoiceUse{
		{Card: costCardID(t, "caravansery"), Mask: catalog.RawGoods, Res: catalog.Stone},
		{Wonder: catalog.WonderGreatLighthouse, Mask: catalog.RawGoods, Res: catalog.Stone},
	}
	if got != want {
		t.Errorf("choices %+v, want %+v", got, want)
	}
	// Reordering the cache does not change the names.
	pl := &s.Players[0]
	pl.Choice[0], pl.Choice[2] = pl.Choice[2], pl.Choice[0]
	if p2 := BuildingPayment(&s, 0, costCardID(t, "town-hall")); p2.Total != p.Total ||
		[2]ChoiceUse{p2.Choice[0], p2.Choice[1]} != want {
		t.Errorf("after reordering the cache: %+v", p2)
	}
}

func TestDiscardValueAndTradePrices(t *testing.T) {
	costUseCatalog(t)
	var s State
	if v := DiscardValue(&s, 0); v != 2 {
		t.Errorf("discard with no yellow card: %d, want 2", v)
	}
	costGive(t, &s, 0, "tavern", "forum", "clay-reserve", "lumber-yard")
	costGive(t, &s, 1, "caravansery", "customs-house", "sawmill", "glassworks", "clay-pool")
	if v := DiscardValue(&s, 0); v != 5 {
		t.Errorf("discard with 3 yellow cards: %d, want 5", v)
	}
	if v := DiscardValue(&s, 1); v != 4 {
		t.Errorf("discard with 2 yellow cards: %d, want 4", v)
	}
	// Seat 0 fixes clay; seat 1 makes 2 wood, 1 clay and 1 glass (its
	// Caravansery does not count).
	if got, want := TradePrices(&s, 0), [5]uint8{4, 1, 2, 3, 2}; got != want {
		t.Errorf("seat 0 trade prices %v, want %v", got, want)
	}
	// Seat 1 fixes glass and papyrus; seat 0 makes 1 wood.
	if got, want := TradePrices(&s, 1), [5]uint8{3, 2, 2, 1, 1}; got != want {
		t.Errorf("seat 1 trade prices %v, want %v", got, want)
	}
	if p := BuildingPayment(&s, 0, costCardID(t, "palace")); p.UnitPrice != TradePrices(&s, 0) ||
		p.FixedPrice != s.Players[0].TradeFix {
		t.Errorf("payment prices %v (fixed %b) differ from TradePrices", p.UnitPrice, p.FixedPrice)
	}
}

func TestPaymentOfUnknownItems(t *testing.T) {
	costUseCatalog(t)
	var s State
	s.Players[0].Coins = 99
	for _, c := range []catalog.CardID{0, 11, catalog.Unknown} { // 11 is a real id missing from the test catalog
		if p := BuildingPayment(&s, 0, c); p != (Payment{}) {
			t.Errorf("BuildingPayment(%d) = %+v, want the zero Payment", c, p)
		}
		if q := BuildingQuote(&s, 0, c); q != (Quote{}) {
			t.Errorf("BuildingQuote(%d) = %+v, want the zero Quote", c, q)
		}
	}
	for _, w := range []catalog.WonderID{0, catalog.WonderMausoleum, catalog.Unknown} {
		if p := WonderPayment(&s, 0, w); p != (Payment{}) {
			t.Errorf("WonderPayment(%d) = %+v, want the zero Payment", w, p)
		}
		if q := WonderQuote(&s, 0, w); q != (Quote{}) {
			t.Errorf("WonderQuote(%d) = %+v, want the zero Quote", w, q)
		}
	}
}

// Each seat pays with its own production and purse, at prices set by the
// other seat's production.
func TestPaymentUsesTheRightSeat(t *testing.T) {
	costUseCatalog(t)
	var s State
	s.Players[0].Coins, s.Players[1].Coins = 20, 3
	costGive(t, &s, 0, "quarry")
	costGive(t, &s, 1, "clay-pool", "brickyard")
	garrison, walls := costCardID(t, "garrison"), costCardID(t, "walls")
	if p := BuildingPayment(&s, 1, garrison); p.Total != 0 || p.Own != costRes("C") || !p.Affordable {
		t.Errorf("seat 1 Garrison: %+v, want free from its own clay", p)
	}
	if p := BuildingPayment(&s, 0, garrison); p.Total != 5 || p.Own != [5]uint8{} || !p.Affordable {
		t.Errorf("seat 0 Garrison: %+v, want 5 coins of clay", p)
	}
	// Walls (SS): seat 0 makes one stone and buys one at 2; seat 1 buys two
	// at 3 each, more than its 3 coins.
	if p := BuildingPayment(&s, 0, walls); p.Total != 2 || !p.Affordable {
		t.Errorf("seat 0 Walls: %+v, want 2, affordable", p)
	}
	if p := BuildingPayment(&s, 1, walls); p.Total != 6 || p.Affordable {
		t.Errorf("seat 1 Walls: %+v, want 6, unaffordable with 3 coins", p)
	}
}

// The adapters on random cities: each payment is a valid breakdown of the
// inputs read independently from the catalog definitions, its total matches
// the exhaustive search, its producers and chain are named from the payer's
// own city, and the quotes agree with the payments.
func TestPaymentAdaptersRandom(t *testing.T) {
	costUseCatalog(t)
	var cards []catalog.CardID
	for d := range catalog.Cards() {
		cards = append(cards, d.ID)
	}
	var wonders []catalog.WonderID
	for d := range catalog.Wonders() {
		wonders = append(wonders, d.ID)
	}
	var tokens []catalog.TokenID
	for d := range catalog.Tokens() {
		tokens = append(tokens, d.ID)
	}
	rng := NewRNG(31)
	const games = 3000
	var chained, choiced, discounted int
	for range games {
		var s State
		for _, c := range cards {
			if seat := rng.IntN(4); seat < 2 {
				s.Players[seat].City.Add(c)
			}
		}
		ShuffleSlice(&rng, wonders)
		for i, w := range wonders[:2*2] {
			s.Players[i/2].Wonders[i%2] = WonderSlot{ID: w, Built: rng.IntN(2) == 0}
		}
		for _, tok := range tokens {
			if seat := rng.IntN(3); seat < 2 {
				s.Players[seat].Tokens.Add(tok)
			}
		}
		for seat := range s.Players {
			p := &s.Players[seat]
			p.Coins = int16(rng.IntN(16))
			costRecompute(t, p)
			// The adapters must not depend on the order of the cache.
			rng.Shuffle(int(p.NChoice), func(i, j int) { p.Choice[i], p.Choice[j] = p.Choice[j], p.Choice[i] })
		}
		for seat := range Seat(2) {
			for _, c := range cards {
				d := catalog.Card(c)
				target := catalog.NoTarget
				if d.Color == catalog.Blue {
					target = catalog.ForBlue
				}
				in := costExpectedInput(t, &s, seat, d.Cost, target, d.ChainIn)
				pay := BuildingPayment(&s, seat, c)
				costCheckAdapter(t, &s, seat, &in, &pay, d.Slug, target)
				if q := BuildingQuote(&s, seat, c); q != pay.Quote() {
					t.Fatalf("%s: quote %+v, payment says %+v", d.Slug, q, pay.Quote())
				}
				if pay.ChainFree {
					chained++
					if from := catalog.Card(pay.ChainFrom); from == nil || !s.Players[seat].City.Has(pay.ChainFrom) ||
						from.ChainOut != d.ChainIn {
						t.Fatalf("%s: chain from %v, which does not provide %v", d.Slug, pay.ChainFrom, d.ChainIn)
					}
				}
				if pay.NChoice > 0 {
					choiced++
				}
				if pay.Discount != [5]uint8{} {
					discounted++
				}
			}
			for _, w := range wonders {
				d := catalog.Wonder(w)
				in := costExpectedInput(t, &s, seat, d.Cost, catalog.ForWonders, catalog.NoChain)
				pay := WonderPayment(&s, seat, w)
				costCheckAdapter(t, &s, seat, &in, &pay, d.Slug, catalog.ForWonders)
				if q := WonderQuote(&s, seat, w); q != pay.Quote() {
					t.Fatalf("%s: quote %+v, payment says %+v", d.Slug, q, pay.Quote())
				}
			}
		}
	}
	if chained < 1000 || choiced < 1000 || discounted < 1000 {
		t.Errorf("weak coverage: %d chain-free, %d with producers, %d discounted", chained, choiced, discounted)
	}
}

// costExpectedInput reads the cost input for seat straight from the catalog
// definitions of both cities, without the Player caches. The order of the
// producers may differ from the cache order.
func costExpectedInput(tb testing.TB, s *State, seat Seat, cost catalog.Cost, target catalog.DiscountTarget,
	chainIn catalog.Chain) costInput {
	tb.Helper()
	p, o := &s.Players[seat], &s.Players[seat.Opp()]
	in := costInput{purse: p.Coins, printed: int16(cost.Coins), need: cost.Res}
	for id := range p.City.All() {
		d := catalog.Card(id)
		if chainIn != catalog.NoChain && d.ChainOut == chainIn {
			in.chainFree = true
		}
		for _, e := range d.Effects {
			switch e.Kind {
			case catalog.EffProduce:
				in.own[e.Res] += e.N
			case catalog.EffProduceChoice:
				in.choice[in.nChoice] = e.Mask
				in.nChoice++
			case catalog.EffTradeFix:
				in.fix |= e.Mask
			}
		}
	}
	for _, ws := range p.Wonders {
		if ws.Built {
			if e, ok := catalog.FindEffect(catalog.Wonder(ws.ID).Effects, catalog.EffProduceChoice); ok {
				in.choice[in.nChoice] = e.Mask
				in.nChoice++
			}
		}
	}
	for id := range o.City.All() {
		for _, e := range catalog.Card(id).Effects {
			if e.Kind == catalog.EffProduce {
				in.oppProd[e.Res] += e.N
			}
		}
	}
	for tok := range p.Tokens.All() {
		for _, e := range catalog.Token(tok).Effects {
			if e.Kind == catalog.EffDiscount && e.Target == target && target != catalog.NoTarget {
				in.discount += e.N
			}
			if e.Kind == catalog.EffChainBonus {
				in.chainBonus += e.N
			}
		}
	}
	for tok := range o.Tokens.All() {
		if _, ok := catalog.FindEffect(catalog.Token(tok).Effects, catalog.EffTradeIncome); ok {
			in.tradeIncome = true
		}
	}
	if !in.chainFree {
		in.chainBonus = 0
	}
	return in
}

// costCheckAdapter checks one adapter payment against the independently read
// input: a valid breakdown, the optimal total, and producers named from the
// payer's own city and built wonders, each source used once.
func costCheckAdapter(tb testing.TB, s *State, seat Seat, in *costInput, pay *Payment, what string,
	target catalog.DiscountTarget) {
	tb.Helper()
	for _, bad := range costPaymentProblems(in, pay) {
		tb.Fatalf("%s for seat %d: %s\ninput %+v\npayment %+v", what, seat, bad, *in, *pay)
	}
	want := int16(0)
	if !in.chainFree {
		want = in.printed + int16(oracleTrade(in))
	}
	if pay.Total != want {
		tb.Fatalf("%s for seat %d: total %d, exhaustive search %d\ninput %+v\npayment %+v",
			what, seat, pay.Total, want, *in, *pay)
	}
	p := &s.Players[seat]
	var usedCards catalog.CardSet
	var usedWonders [catalog.NumWonders + 1]bool
	for _, u := range pay.Choice[:pay.NChoice] {
		switch {
		case u.Card != 0 && u.Wonder == 0:
			e, ok := catalog.FindEffect(catalog.Card(u.Card).Effects, catalog.EffProduceChoice)
			if !p.City.Has(u.Card) || !ok || e.Mask != u.Mask || usedCards.Has(u.Card) {
				tb.Fatalf("%s for seat %d: bad card source in %+v", what, seat, u)
			}
			usedCards.Add(u.Card)
		case u.Wonder != 0 && u.Card == 0:
			built := false
			for _, ws := range p.Wonders {
				built = built || (ws.ID == u.Wonder && ws.Built)
			}
			e, ok := catalog.FindEffect(catalog.Wonder(u.Wonder).Effects, catalog.EffProduceChoice)
			if !built || !ok || e.Mask != u.Mask || usedWonders[u.Wonder] {
				tb.Fatalf("%s for seat %d: bad wonder source in %+v", what, seat, u)
			}
			usedWonders[u.Wonder] = true
		default:
			tb.Fatalf("%s for seat %d: producer use without exactly one source: %+v", what, seat, u)
		}
	}
	// DiscountBy names the first owned token (by id) whose discount fits the
	// item, unless the build is chain-free.
	wantBy := catalog.TokenID(0)
	if !in.chainFree && target != catalog.NoTarget {
		for tok := range p.Tokens.All() {
			for _, e := range catalog.Token(tok).Effects {
				if e.Kind == catalog.EffDiscount && e.Target == target && wantBy == 0 {
					wantBy = tok
				}
			}
		}
	}
	if pay.DiscountBy != wantBy {
		tb.Fatalf("%s for seat %d: discount by %v, want %v", what, seat, pay.DiscountBy, wantBy)
	}
}

// When the real catalog is loaded, every flexible producer must make exactly
// one resource class: the greedy payment is optimal only then. The design
// has the catalog assert this when it loads as well; this test guards the
// engine's side of the contract and skips while the data is absent.
func TestCostChoiceMasksAreClasses(t *testing.T) {
	n := 0
	check := func(what string, effects []catalog.Effect) {
		for _, e := range effects {
			if e.Kind == catalog.EffProduceChoice {
				n++
				if e.Mask != catalog.RawGoods && e.Mask != catalog.Manufactured {
					t.Errorf("%s produces %b, which is not one resource class", what, e.Mask)
				}
			}
		}
	}
	for d := range catalog.Cards() {
		check(d.Slug, d.Effects)
	}
	for d := range catalog.Wonders() {
		check(d.Slug, d.Effects)
	}
	if n == 0 {
		t.Skip("the catalog has no flexible producers yet (data not loaded)")
	}
	if n != MaxChoice {
		t.Errorf("%d flexible producers in the catalog, want %d", n, MaxChoice)
	}
}

func TestCheapestAllocs(t *testing.T) {
	in := costInput{
		purse: 9, printed: 2, need: costRes("C S W GG"), own: costRes("W"), oppProd: costRes("S GG"),
		fix: catalog.Clay.Mask(), choice: [MaxChoice]catalog.ResMask{catalog.Manufactured, catalog.RawGoods},
		nChoice: 2, discount: 2, tradeIncome: true,
	}
	var pay Payment
	if n := testing.AllocsPerRun(1000, func() {
		cheapest(&in, nil)
		cheapest(&in, &pay)
		_ = quote(&in)
	}); n != 0 {
		t.Errorf("cheapest allocates %.1f times per run, want 0", n)
	}
}

func TestPaymentAdaptersAllocs(t *testing.T) {
	costUseCatalog(t)
	s := costBenchState(t)
	palace, aqueduct := costCardID(t, "palace"), costCardID(t, "aqueduct")
	pyramids := costWonderID(t, "pyramids")
	var sink int16
	if n := testing.AllocsPerRun(1000, func() {
		p := BuildingPayment(&s, 0, palace)
		c := BuildingPayment(&s, 0, aqueduct)
		w := WonderPayment(&s, 0, pyramids)
		q := BuildingQuote(&s, 1, palace)
		v := WonderQuote(&s, 1, pyramids)
		tp := TradePrices(&s, 0)
		sink += p.Total + c.UrbanismBonus + w.Total + q.Total + v.Total + int16(tp[0]) + DiscardValue(&s, 1)
	}); n != 0 {
		t.Errorf("the payment adapters allocate %.1f times per run, want 0", n)
	}
	_ = sink
}

// costBenchState is a mid-game position with producers, fixed prices, a
// chain and the four payment tokens in play.
func costBenchState(tb testing.TB) State {
	tb.Helper()
	var s State
	s.Players[0].Coins, s.Players[1].Coins = 12, 9
	costGive(tb, &s, 0, "forum", "caravansery", "clay-reserve", "baths", "lumber-yard", "press", "tavern")
	costBuild(tb, &s, 0, "great-lighthouse", "piraeus")
	costTokens(tb, &s, 0, "masonry", "architecture", "urbanism")
	costGive(tb, &s, 1, "shelf-quarry", "glassworks", "glass-blower", "brickyard", "customs-house")
	costTokens(tb, &s, 1, "economy")
	return s
}

func BenchmarkCheapest(b *testing.B) {
	in := costInput{
		purse: 9, printed: 2, need: costRes("C S W GG"), own: costRes("W"), oppProd: costRes("S GG"),
		fix: catalog.Clay.Mask(), choice: [MaxChoice]catalog.ResMask{catalog.Manufactured, catalog.RawGoods},
		nChoice: 2, discount: 2, tradeIncome: true,
	}
	b.Run("quote", func(b *testing.B) {
		for b.Loop() {
			cheapest(&in, nil)
		}
	})
	b.Run("breakdown", func(b *testing.B) {
		var pay Payment
		for b.Loop() {
			cheapest(&in, &pay)
		}
	})
}

func BenchmarkPaymentAdapters(b *testing.B) {
	costUseCatalog(b)
	s := costBenchState(b)
	palace := costCardID(b, "palace")
	pyramids := costWonderID(b, "pyramids")
	b.Run("BuildingPayment", func(b *testing.B) {
		for b.Loop() {
			BuildingPayment(&s, 0, palace)
		}
	})
	b.Run("BuildingQuote", func(b *testing.B) {
		for b.Loop() {
			BuildingQuote(&s, 0, palace)
		}
	})
	b.Run("WonderPayment", func(b *testing.B) {
		for b.Loop() {
			WonderPayment(&s, 0, pyramids)
		}
	})
	b.Run("WonderQuote", func(b *testing.B) {
		for b.Loop() {
			WonderQuote(&s, 0, pyramids)
		}
	})
}
