package catalog

// The 23 Age II buildings (ids 24 to 46). ChainIn is the chain symbol that
// makes a card free when its owner already shows that symbol.
var ageTwoCards = []CardDef{
	// Brown: raw goods, two units each.
	{ID: 24, Slug: "sawmill", Name: "Sawmill", Color: Brown, Cost: MustCost("2c"), Effects: Fx(Produce(Wood, 2))},
	{ID: 25, Slug: "shelf-quarry", Name: "Shelf Quarry", Color: Brown, Cost: MustCost("2c"), Effects: Fx(Produce(Stone, 2))},
	{ID: 26, Slug: "brickyard", Name: "Brickyard", Color: Brown, Cost: MustCost("2c"), Effects: Fx(Produce(Clay, 2))},

	// Grey: manufactured goods.
	{ID: 27, Slug: "glass-blower", Name: "Glass-Blower", Color: Grey, Effects: Fx(Produce(Glass, 1))},
	{ID: 28, Slug: "drying-room", Name: "Drying Room", Color: Grey, Effects: Fx(Produce(Papyrus, 1))},

	// Red: military.
	{ID: 29, Slug: "horse-breeders", Name: "Horse Breeders", Color: Red, Cost: MustCost("C W"), ChainIn: ChainHorseshoe,
		Effects: Fx(Shields(1))},
	{ID: 30, Slug: "barracks", Name: "Barracks", Color: Red, Cost: MustCost("3c"), ChainIn: ChainSword,
		Effects: Fx(Shields(1))},
	{ID: 31, Slug: "walls", Name: "Walls", Color: Red, Cost: MustCost("SS"), Effects: Fx(Shields(2))},
	{ID: 32, Slug: "archery-range", Name: "Archery Range", Color: Red, Cost: MustCost("S W P"), ChainOut: ChainTarget,
		Effects: Fx(Shields(2))},
	{ID: 33, Slug: "parade-ground", Name: "Parade Ground", Color: Red, Cost: MustCost("CC G"), ChainOut: ChainHelmet,
		Effects: Fx(Shields(2))},

	// Green: science.
	{ID: 34, Slug: "school", Name: "School", Color: Green, Cost: MustCost("W PP"), ChainOut: ChainLyre,
		Effects: Fx(Science(Wheel), Points(1))},
	{ID: 35, Slug: "laboratory", Name: "Laboratory", Color: Green, Cost: MustCost("W GG"), ChainOut: ChainLamp,
		Effects: Fx(Science(SetSquare), Points(1))},
	{ID: 36, Slug: "dispensary", Name: "Dispensary", Color: Green, Cost: MustCost("CC S"), ChainIn: ChainGear,
		Effects: Fx(Science(Mortar), Points(2))},
	{ID: 37, Slug: "library", Name: "Library", Color: Green, Cost: MustCost("S W G"), ChainIn: ChainBook,
		Effects: Fx(Science(Quill), Points(2))},

	// Yellow: commerce.
	{ID: 38, Slug: "brewery", Name: "Brewery", Color: Yellow, ChainOut: ChainBarrel, Effects: Fx(Coins(6))},
	{ID: 39, Slug: "forum", Name: "Forum", Color: Yellow, Cost: MustCost("3c + C"),
		Effects: Fx(ProduceChoice(Glass, Papyrus))},
	{ID: 40, Slug: "caravansery", Name: "Caravansery", Color: Yellow, Cost: MustCost("2c + G P"),
		Effects: Fx(ProduceChoice(Wood, Clay, Stone))},
	{ID: 41, Slug: "customs-house", Name: "Customs House", Color: Yellow, Cost: MustCost("4c"),
		Effects: Fx(TradeFix(Glass, Papyrus))},

	// Blue: civilian.
	{ID: 42, Slug: "temple", Name: "Temple", Color: Blue, Cost: MustCost("W P"), ChainIn: ChainMoon, ChainOut: ChainSun,
		Effects: Fx(Points(4))},
	{ID: 43, Slug: "statue", Name: "Statue", Color: Blue, Cost: MustCost("CC"), ChainIn: ChainMask, ChainOut: ChainColumn,
		Effects: Fx(Points(4))},
	{ID: 44, Slug: "courthouse", Name: "Courthouse", AltNames: []string{"Tribunal"}, Color: Blue, Cost: MustCost("WW G"),
		Effects: Fx(Points(5))},
	{ID: 45, Slug: "aqueduct", Name: "Aqueduct", Color: Blue, Cost: MustCost("SSS"), ChainIn: ChainDrop,
		Effects: Fx(Points(5))},
	{ID: 46, Slug: "rostrum", Name: "Rostrum", Color: Blue, Cost: MustCost("S W"), ChainOut: ChainTempleFacade,
		Effects: Fx(Points(4))},
}

var _ = registerCards(2, ageTwoCards)
