package catalog

// Representative effect holders from docs/haftgane/effects.md. These are
// type-contract fixtures, not the complete catalog or verified card costs.

import (
	"fmt"
	"testing"
)

func inventoryCost(coins uint8, units ...Resource) Cost {
	c := Cost{Coins: coins}
	for _, r := range units {
		c.Res[r]++
	}
	return c
}

// Every effect shape effects.md lists, on its real holder kind and colour.
func inventoryDesigns() []Design {
	type item struct {
		slug  string
		color Color
		age   uint8
		cost  Cost
		eff   []Effect
	}
	all := Self | Left | Right
	items := []item{
		{"lumber-yard", Brown, 1, Cost{}, []Effect{{Kind: EffProduce, N: 1, Res: Wood, ForSale: true}}},
		{"sawmill", Brown, 2, inventoryCost(1), []Effect{{Kind: EffProduce, N: 2, Res: Wood, ForSale: true}}},
		{"glassworks-age-1", Grey, 1, Cost{}, []Effect{{Kind: EffProduce, N: 1, Res: Glass, ForSale: true}}},
		{"loom-age-1", Grey, 1, Cost{}, []Effect{{Kind: EffProduce, N: 1, Res: Cloth, ForSale: true}}},
		{"tree-farm", Brown, 1, inventoryCost(1), []Effect{{Kind: EffProduceChoice, Mask: Wood.Mask() | Clay.Mask(), ForSale: true}}},
		{"mine", Brown, 1, inventoryCost(1), []Effect{{Kind: EffProduceChoice, Mask: Stone.Mask() | Ore.Mask(), ForSale: true}}},
		{"caravansery", Yellow, 2, inventoryCost(0, Wood, Wood), []Effect{{Kind: EffProduceChoice, Mask: Raw}}},
		{"forum", Yellow, 2, inventoryCost(0, Clay, Clay), []Effect{{Kind: EffProduceChoice, Mask: Manufactured}}},
		{"west-trading-post", Yellow, 1, Cost{}, []Effect{{Kind: EffTradeDiscount, N: 1, Mask: Raw, Scope: Left}}},
		{"east-trading-post", Yellow, 1, Cost{}, []Effect{{Kind: EffTradeDiscount, N: 1, Mask: Raw, Scope: Right}}},
		{"marketplace", Yellow, 1, Cost{}, []Effect{{Kind: EffTradeDiscount, N: 1, Mask: Manufactured, Scope: Left | Right}}},
		{"stockade", Red, 1, inventoryCost(0, Wood), []Effect{{Kind: EffShields, N: 1}}},
		{"siege-workshop", Red, 3, inventoryCost(0, Wood, Clay, Clay, Clay), []Effect{{Kind: EffShields, N: 3}}},
		{"apothecary", Green, 1, inventoryCost(0, Cloth), []Effect{{Kind: EffScience, Symbol: Compass}}},
		{"workshop", Green, 1, inventoryCost(0, Glass), []Effect{{Kind: EffScience, Symbol: Gear}}},
		{"scriptorium", Green, 1, inventoryCost(0, Papyrus), []Effect{{Kind: EffScience, Symbol: Tablet}}},
		{"scientists-guild", Purple, 3, inventoryCost(0, Wood, Wood, Ore, Ore, Papyrus), []Effect{{Kind: EffScienceChoice}}},
		{"tavern", Yellow, 1, Cost{}, []Effect{{Kind: EffCoins, N: 5}}},
		{"vineyard", Yellow, 2, Cost{}, []Effect{{Kind: EffCoinsPerCard, N: 1, Colors: Brown.Mask(), Scope: all}}},
		{"bazar", Yellow, 2, Cost{}, []Effect{{Kind: EffCoinsPerCard, N: 2, Colors: Grey.Mask(), Scope: all}}},
		{"lighthouse", Yellow, 3, inventoryCost(0, Wood, Wood, Glass), []Effect{{Kind: EffCoinsPerCard, N: 1, Colors: Yellow.Mask(), Scope: Self}, {Kind: EffPointsPerCard, N: 1, Colors: Yellow.Mask(), Scope: Self}}},
		{"chamber-of-commerce", Yellow, 3, inventoryCost(0, Clay, Clay, Papyrus), []Effect{{Kind: EffCoinsPerCard, N: 2, Colors: Grey.Mask(), Scope: Self}, {Kind: EffPointsPerCard, N: 2, Colors: Grey.Mask(), Scope: Self}}},
		{"ludus", Yellow, 3, inventoryCost(0, Stone, Ore), []Effect{{Kind: EffCoinsPerCard, N: 1, Colors: Red.Mask(), Scope: Self}, {Kind: EffPointsPerCard, N: 1, Colors: Red.Mask(), Scope: Self}}},
		{"arena", Yellow, 3, inventoryCost(0, Stone, Stone, Ore), []Effect{{Kind: EffCoinsPerStage, N: 3, Scope: Self}, {Kind: EffPointsPerStage, N: 1, Scope: Self}}},
		{"workers-guild", Purple, 3, inventoryCost(0, Ore, Ore, Clay, Stone, Wood), []Effect{{Kind: EffPointsPerCard, N: 1, Colors: Brown.Mask(), Scope: Left | Right}}},
		{"shipowners-guild", Purple, 3, inventoryCost(0, Wood, Wood, Wood, Glass, Papyrus), []Effect{{Kind: EffPointsPerCard, N: 1, Colors: Brown.Mask() | Grey.Mask() | Purple.Mask(), Scope: Self}}},
		{"builders-guild", Purple, 3, inventoryCost(0, Stone, Stone, Clay, Clay, Glass), []Effect{{Kind: EffPointsPerStage, N: 1, Scope: all}}},
		{"decorators-guild", Purple, 3, inventoryCost(0, Ore, Ore, Stone, Cloth), []Effect{{Kind: EffPointsIfWonderComplete, N: 7}}},
		{"well", Blue, 1, Cost{}, []Effect{{Kind: EffPoints, N: 3}}},
		{"palace", Blue, 3, inventoryCost(0, Wood, Clay, Stone, Ore, Glass, Papyrus, Cloth), []Effect{{Kind: EffPoints, N: 7}}},
	}
	out := make([]Design, 0, len(items))
	for i, it := range items {
		d := Design{ID: DesignID(i + 1), Slug: it.slug, Name: it.slug, NameID: NameID(i + 1), Age: it.age, Color: it.color, Cost: it.cost, Effects: it.eff}
		if it.color != Purple {
			d.Copies = [3]uint8{3, 5}
		}
		out = append(out, d)
	}
	return out
}

func inventoryStage(cost Cost, eff ...Effect) Stage { return Stage{Cost: cost, Effects: eff} }

func inventoryBoards() []Board {
	pts := func(n uint8) Effect { return Effect{Kind: EffPoints, N: n} }
	coins := func(n uint8) Effect { return Effect{Kind: EffCoins, N: n} }
	shields := func(n uint8) Effect { return Effect{Kind: EffShields, N: n} }
	free := func(w FreeCondition) Effect { return Effect{Kind: EffFreeBuild, When: w} }
	discard := Effect{Kind: EffBuildFromDiscard}
	day3 := func(mid ...Effect) SideDef {
		return SideDef{NumStages: 3, Stages: [MaxStages]Stage{inventoryStage(inventoryCost(0, Stone, Stone), pts(3)), inventoryStage(inventoryCost(0, Ore, Ore, Ore), mid...), inventoryStage(inventoryCost(0, Glass, Glass), pts(7))}}
	}
	b := []Board{
		{ID: BoardAlexandria, Start: Glass, Sides: [2]SideDef{day3(Effect{Kind: EffProduceChoice, Mask: Raw}), {NumStages: 3, Stages: [MaxStages]Stage{
			inventoryStage(inventoryCost(0, Clay, Clay), Effect{Kind: EffProduceChoice, Mask: Raw}),
			inventoryStage(inventoryCost(0, Wood, Wood), Effect{Kind: EffProduceChoice, Mask: Manufactured}),
			inventoryStage(inventoryCost(0, Stone, Stone, Stone), pts(7))}}}},
		{ID: BoardBabylon, Start: Wood, Sides: [2]SideDef{day3(Effect{Kind: EffScienceChoice}), {NumStages: 2, Stages: [MaxStages]Stage{
			inventoryStage(inventoryCost(0, Stone, Stone), Effect{Kind: EffPlayLastCard}),
			inventoryStage(inventoryCost(0, Clay, Clay, Clay, Glass), Effect{Kind: EffScienceChoice})}}}},
		{ID: BoardEphesos, Start: Papyrus, Sides: [2]SideDef{day3(coins(9)), {NumStages: 3, Stages: [MaxStages]Stage{
			inventoryStage(inventoryCost(0, Stone, Stone), coins(4), pts(2)), inventoryStage(inventoryCost(0, Wood, Wood), coins(4), pts(3)),
			inventoryStage(inventoryCost(0, Papyrus, Cloth, Glass), coins(4), pts(5))}}}},
		{ID: BoardGizah, Start: Stone, Sides: [2]SideDef{day3(pts(5)), {NumStages: 4, Stages: [MaxStages]Stage{
			inventoryStage(inventoryCost(0, Wood, Wood), pts(3)), inventoryStage(inventoryCost(0, Stone, Stone, Stone), pts(5)),
			inventoryStage(inventoryCost(0, Clay, Clay, Clay), pts(5)), inventoryStage(inventoryCost(0, Stone, Stone, Stone, Stone, Papyrus), pts(7))}}}},
		{ID: BoardHalikarnassos, Start: Cloth, Sides: [2]SideDef{day3(discard), {NumStages: 3, Stages: [MaxStages]Stage{
			inventoryStage(inventoryCost(0, Ore, Ore), discard, pts(2)), inventoryStage(inventoryCost(0, Clay, Clay, Clay), discard, pts(1)),
			inventoryStage(inventoryCost(0, Wood, Wood, Wood), discard)}}}},
		{ID: BoardOlympia, Start: Clay, Sides: [2]SideDef{day3(free(FirstOfColour)), {NumStages: 3, Stages: [MaxStages]Stage{
			inventoryStage(inventoryCost(0, Wood, Wood), free(FirstTurnOfAge)), inventoryStage(inventoryCost(0, Stone, Stone), free(LastTurnOfAge)),
			inventoryStage(inventoryCost(0, Ore, Ore, Cloth), pts(5))}}}},
		{ID: BoardRhodos, Start: Ore, Sides: [2]SideDef{day3(shields(2)), {NumStages: 2, Stages: [MaxStages]Stage{
			inventoryStage(inventoryCost(0, Stone, Stone, Stone), shields(1), coins(3), pts(3)),
			inventoryStage(inventoryCost(0, Ore, Ore, Ore, Ore), shields(1), coins(4), pts(4))}}}},
	}
	for i := range b {
		b[i].Slug = b[i].ID.String()
		b[i].Name = "City " + b[i].Slug
		b[i].Wonder = "Wonder " + b[i].Slug
	}
	return b
}

func TestEffectHolderInventory(t *testing.T) {
	designs, boards := inventoryDesigns(), inventoryBoards()
	if err := ValidateCatalog(designs, boards); err != nil {
		t.Fatal(err)
	}
	for _, d := range designs {
		t.Run(d.Slug, func(t *testing.T) {
			if err := d.Validate(); err != nil {
				t.Fatal(err)
			}
		})
	}
	stages := 0
	for _, b := range boards {
		t.Run(b.Slug, func(t *testing.T) {
			if err := b.Validate(); err != nil {
				t.Fatal(err)
			}
		})
		stages += int(b.Sides[0].NumStages + b.Sides[1].NumStages)
	}
	t.Logf("inventory shapes: %d designs and %d boards (%d stages) validate", len(designs), len(boards), stages)
	if stages != 41 {
		t.Errorf("stage count %d", stages)
	}
}

// A synthetic catalog at full size with the real graph shape: 80 designs
// (27/23/20 + 10 guilds), 77 names via three Age I/II twin pairs, 24 chain
// symbols with 25 provider links (one double, two Age I -> Age III links).
func inventoryFullCatalog() []Design {
	var ds []Design
	add := func(age uint8, color Color) *Design {
		id := DesignID(len(ds) + 1)
		d := Design{ID: id, Slug: fmt.Sprintf("design-%02d", id), Name: fmt.Sprintf("Design %02d", id), Age: age, Color: color, Effects: []Effect{{Kind: EffPoints, N: 1}}}
		if color != Purple {
			d.Copies = [3]uint8{3, 6}
		}
		ds = append(ds, d)
		return &ds[len(ds)-1]
	}
	for i := 0; i < 27; i++ {
		add(1, Blue)
	}
	for i := 0; i < 23; i++ {
		add(2, Blue)
	}
	for i := 0; i < 20; i++ {
		add(3, Blue)
	}
	for i := 0; i < 10; i++ {
		add(3, Purple)
	}
	nameID := NameID(4)
	for i := range ds {
		ds[i].NameID = nameID
		nameID++
	}
	twin := func(a, b int, slug string, n NameID) {
		for _, i := range []int{a, b} {
			ds[i].Color = Grey
			ds[i].Effects = []Effect{{Kind: EffProduce, N: 1, Res: Glass + Resource(n-1), ForSale: true}}
			ds[i].Name = slug
			ds[i].NameID = n
			ds[i].Slug = fmt.Sprintf("%s-age-%d", slug, ds[i].Age)
		}
	}
	twin(0, 27, "glassworks", 1)
	twin(1, 28, "press", 2)
	twin(2, 29, "loom", 3)
	// Renumber the remaining name ids densely to 4..77.
	next := NameID(4)
	for i := range ds {
		if ds[i].NameID >= 4 {
			ds[i].NameID = next
			next++
		}
	}
	chain := Chain(1)
	// Ten Age II targets (ids 31..40) from Age I providers (ids 4..13); id 14 doubles the first.
	for k := 0; k < 10; k++ {
		ds[30+k].ChainIn = chain
		ds[3+k].ChainOut[0] = chain
		chain++
	}
	ds[13].ChainOut[0] = 1
	// Two Age I -> Age III links (ids 15, 16 -> 51, 52).
	for k := 0; k < 2; k++ {
		ds[50+k].ChainIn = chain
		ds[14+k].ChainOut[0] = chain
		chain++
	}
	// Twelve Age III targets (53..64) from Age II providers 41..50, two of them giving two.
	providers := []int{40, 40, 41, 41, 42, 43, 44, 45, 46, 47, 48, 49}
	for k, p := range providers {
		ds[52+k].ChainIn = chain
		if ds[p].ChainOut[0] == 0 {
			ds[p].ChainOut[0] = chain
		} else {
			ds[p].ChainOut[1] = chain
		}
		chain++
	}
	return ds
}

func TestFullSizeCatalogShape(t *testing.T) {
	ds := inventoryFullCatalog()
	names := map[NameID]bool{}
	symbols, links := 0, 0
	for _, d := range ds {
		names[d.NameID] = true
		if d.ChainIn != 0 {
			symbols++
		}
		for _, c := range d.ChainOut {
			if c != 0 {
				links++
			}
		}
	}
	t.Logf("full-size synthetic: %d designs, %d names, %d chain symbols, %d provider links", len(ds), len(names), symbols, links)
	if err := ValidateCatalog(ds, inventoryBoards()); err != nil {
		t.Fatal(err)
	}
	restore := ReplaceForTest(ds, inventoryBoards())
	defer restore()
	pairs := 0
	for id := DesignID(1); id <= NumDesigns; id++ {
		if Twin[id] != 0 {
			pairs++
			if Twin[Twin[id]] != id {
				t.Fatalf("asymmetric twin %d", id)
			}
		}
	}
	t.Logf("twin entries %d (3 pairs expected)", pairs)
	if _, ok := DesignBySlug("glassworks-age-2"); !ok {
		t.Fatal("twin slug lookup")
	}
	// A second double-provider symbol is rejected; a third provider too.
	bad := inventoryFullCatalog()
	bad[16].ChainOut[0] = 2
	if ValidateCatalog(bad, nil) == nil {
		t.Error("second double symbol passed")
	}
	// A provider in the same Age as its target is rejected (Age II -> Age II).
	bad = inventoryFullCatalog()
	bad[13].ChainOut[0] = 0
	bad[49].ChainOut[1] = 1
	if ValidateCatalog(bad, nil) == nil {
		t.Error("same-age provider passed")
	}
}
