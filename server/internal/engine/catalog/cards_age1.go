package catalog

// The 23 Age I buildings. Ids follow the reference implementation's numbering
// (1 to 23 in this age). Costs are written in the notation of MustCost; a
// missing Cost is free. ChainOut is the chain symbol a card shows, which makes
// one later card free (see compiled.go's ChainTarget).
var ageOneCards = []CardDef{
	// Brown: raw goods, one unit each.
	{ID: 1, Slug: "lumber-yard", Name: "Lumber Yard", Color: Brown, Effects: Fx(Produce(Wood, 1))},
	{ID: 2, Slug: "stone-pit", Name: "Stone Pit", Color: Brown, Cost: MustCost("1c"), Effects: Fx(Produce(Stone, 1))},
	{ID: 3, Slug: "clay-pool", Name: "Clay Pool", Color: Brown, Effects: Fx(Produce(Clay, 1))},
	{ID: 4, Slug: "logging-camp", Name: "Logging Camp", Color: Brown, Cost: MustCost("1c"), Effects: Fx(Produce(Wood, 1))},
	{ID: 5, Slug: "quarry", Name: "Quarry", Color: Brown, Effects: Fx(Produce(Stone, 1))},
	{ID: 6, Slug: "clay-pit", Name: "Clay Pit", Color: Brown, Cost: MustCost("1c"), Effects: Fx(Produce(Clay, 1))},

	// Grey: manufactured goods.
	{ID: 7, Slug: "glassworks", Name: "Glassworks", Color: Grey, Cost: MustCost("1c"), Effects: Fx(Produce(Glass, 1))},
	{ID: 8, Slug: "press", Name: "Press", Color: Grey, Cost: MustCost("1c"), Effects: Fx(Produce(Papyrus, 1))},

	// Red: military.
	{ID: 9, Slug: "stable", Name: "Stable", Color: Red, Cost: MustCost("W"), ChainOut: ChainHorseshoe,
		Effects: Fx(Shields(1))},
	{ID: 10, Slug: "garrison", Name: "Garrison", Color: Red, Cost: MustCost("C"), ChainOut: ChainSword,
		Effects: Fx(Shields(1))},
	{ID: 11, Slug: "palisade", Name: "Palisade", Color: Red, Cost: MustCost("2c"), ChainOut: ChainTower,
		Effects: Fx(Shields(1))},
	{ID: 12, Slug: "guard-tower", Name: "Guard Tower", Color: Red, Effects: Fx(Shields(1))},

	// Green: science.
	{ID: 13, Slug: "scriptorium", Name: "Scriptorium", Color: Green, Cost: MustCost("2c"), ChainOut: ChainBook,
		Effects: Fx(Science(Quill))},
	{ID: 14, Slug: "workshop", Name: "Workshop", Color: Green, Cost: MustCost("P"),
		Effects: Fx(Science(SetSquare), Points(1))},
	{ID: 15, Slug: "pharmacist", Name: "Pharmacist", Color: Green, Cost: MustCost("2c"), ChainOut: ChainGear,
		Effects: Fx(Science(Mortar))},
	{ID: 16, Slug: "apothecary", Name: "Apothecary", Color: Green, Cost: MustCost("G"),
		Effects: Fx(Science(Wheel), Points(1))},

	// Yellow: commerce.
	{ID: 17, Slug: "tavern", Name: "Tavern", Color: Yellow, ChainOut: ChainAmphora, Effects: Fx(Coins(4))},
	{ID: 18, Slug: "stone-reserve", Name: "Stone Reserve", Color: Yellow, Cost: MustCost("3c"),
		Effects: Fx(TradeFix(Stone))},
	{ID: 19, Slug: "clay-reserve", Name: "Clay Reserve", Color: Yellow, Cost: MustCost("3c"),
		Effects: Fx(TradeFix(Clay))},
	{ID: 20, Slug: "wood-reserve", Name: "Wood Reserve", Color: Yellow, Cost: MustCost("3c"),
		Effects: Fx(TradeFix(Wood))},

	// Blue: civilian.
	{ID: 21, Slug: "theater", Name: "Theater", Color: Blue, ChainOut: ChainMask, Effects: Fx(Points(3))},
	{ID: 22, Slug: "altar", Name: "Altar", Color: Blue, ChainOut: ChainMoon, Effects: Fx(Points(3))},
	{ID: 23, Slug: "baths", Name: "Baths", Color: Blue, Cost: MustCost("S"), ChainOut: ChainDrop,
		Effects: Fx(Points(3))},
}

var _ = registerCards(1, ageOneCards)
