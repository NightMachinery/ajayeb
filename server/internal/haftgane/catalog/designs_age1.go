package catalog

// The 27 Age I designs, in id order: colour in the enum's order, then
// English name. Costs use the notation of MustCost; a missing Cost is free.
// Copies lists each copy's minimum player count.
var ageOneDesigns = []Design{
	// Brown: raw resources.
	{ID: 1, Slug: "clay-pit", Name: "Clay Pit", Color: Brown, Cost: MustCost("1c"), Copies: At(3), Effects: Fx(ProduceChoice(Clay, Ore))},
	{ID: 2, Slug: "clay-pool", Name: "Clay Pool", Color: Brown, Copies: At(3, 5), Effects: Fx(Produce(Clay, 1))},
	{ID: 3, Slug: "excavation", Name: "Excavation", Color: Brown, Cost: MustCost("1c"), Copies: At(4), Effects: Fx(ProduceChoice(Stone, Clay))},
	{ID: 4, Slug: "forest-cave", Name: "Forest Cave", Color: Brown, Cost: MustCost("1c"), Copies: At(5), Effects: Fx(ProduceChoice(Wood, Ore))},
	{ID: 5, Slug: "lumber-yard", Name: "Lumber Yard", Color: Brown, Copies: At(3, 4), Effects: Fx(Produce(Wood, 1))},
	{ID: 6, Slug: "mine", Name: "Mine", Color: Brown, Cost: MustCost("1c"), Copies: At(6), Effects: Fx(ProduceChoice(Stone, Ore))},
	{ID: 7, Slug: "ore-vein", Name: "Ore Vein", Color: Brown, Copies: At(3, 4), Effects: Fx(Produce(Ore, 1))},
	{ID: 8, Slug: "stone-pit", Name: "Stone Pit", Color: Brown, Copies: At(3, 5), Effects: Fx(Produce(Stone, 1))},
	{ID: 9, Slug: "timber-yard", Name: "Timber Yard", Color: Brown, Cost: MustCost("1c"), Copies: At(3), Effects: Fx(ProduceChoice(Wood, Stone))},
	{ID: 10, Slug: "tree-farm", Name: "Tree Farm", Color: Brown, Cost: MustCost("1c"), Copies: At(6), Effects: Fx(ProduceChoice(Wood, Clay))},

	// Grey: manufactured resources.
	{ID: 11, Slug: "glassworks-age-1", Name: "Glassworks", Color: Grey, Copies: At(3, 6), Effects: Fx(Produce(Glass, 1))},
	{ID: 12, Slug: "loom-age-1", Name: "Loom", Color: Grey, Copies: At(3, 6), Effects: Fx(Produce(Cloth, 1))},
	{ID: 13, Slug: "press-age-1", Name: "Press", Color: Grey, Copies: At(3, 6), Effects: Fx(Produce(Papyrus, 1))},

	// Red: military.
	{ID: 14, Slug: "barracks", Name: "Barracks", Color: Red, Cost: MustCost("O"), Copies: At(3, 5), Effects: Fx(Shields(1))},
	{ID: 15, Slug: "guard-tower", Name: "Guard Tower", Color: Red, Cost: MustCost("C"), Copies: At(3, 4), Effects: Fx(Shields(1))},
	{ID: 16, Slug: "stockade", Name: "Stockade", Color: Red, Cost: MustCost("W"), Copies: At(3, 7), Effects: Fx(Shields(1))},

	// Blue: civic buildings, points.
	{ID: 17, Slug: "altar", Name: "Altar", Color: Blue, ChainOut: Out(ChainPantheon), Copies: At(3, 5), Effects: Fx(Points(3))},
	{ID: 18, Slug: "baths", Name: "Baths", Color: Blue, Cost: MustCost("S"), ChainOut: Out(ChainAqueduct), Copies: At(3, 7), Effects: Fx(Points(3))},
	{ID: 19, Slug: "theater", Name: "Theater", Color: Blue, ChainOut: Out(ChainGardens), Copies: At(3, 6), Effects: Fx(Points(3))},
	{ID: 20, Slug: "well", Name: "Well", Color: Blue, ChainOut: Out(ChainStatue), Copies: At(4, 7), Effects: Fx(Points(3))},

	// Yellow: commerce.
	{ID: 21, Slug: "east-trading-post", Name: "East Trading Post", Color: Yellow, ChainOut: Out(ChainForum), Copies: At(3, 7), Effects: Fx(TradeDiscount(Raw, Right))},
	{ID: 22, Slug: "marketplace", Name: "Marketplace", Color: Yellow, ChainOut: Out(ChainCaravansery), Copies: At(3, 6), Effects: Fx(TradeDiscount(Manufactured, Left|Right))},
	{ID: 23, Slug: "tavern", Name: "Tavern", Color: Yellow, Copies: At(4, 5, 7), Effects: Fx(Coins(5))},
	{ID: 24, Slug: "west-trading-post", Name: "West Trading Post", Color: Yellow, ChainOut: Out(ChainForum), Copies: At(3, 7), Effects: Fx(TradeDiscount(Raw, Left))},

	// Green: science.
	{ID: 25, Slug: "apothecary", Name: "Apothecary", Color: Green, Cost: MustCost("L"), ChainOut: Out(ChainStables, ChainDispensary), Copies: At(3, 5), Effects: Fx(Science(Compass))},
	{ID: 26, Slug: "scriptorium", Name: "Scriptorium", Color: Green, Cost: MustCost("P"), ChainOut: Out(ChainCourthouse, ChainLibrary), Copies: At(3, 4), Effects: Fx(Science(Tablet))},
	{ID: 27, Slug: "workshop", Name: "Workshop", Color: Green, Cost: MustCost("G"), ChainOut: Out(ChainArcheryRange, ChainLaboratory), Copies: At(3, 7), Effects: Fx(Science(Gear))},
}
