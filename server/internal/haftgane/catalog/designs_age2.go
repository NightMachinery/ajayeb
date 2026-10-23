package catalog

// The 23 Age II designs, in id order: colour in the enum's order, then
// English name.
var ageTwoDesigns = []Design{
	// Brown: raw resources.
	{ID: 28, Slug: "brickyard", Name: "Brickyard", Color: Brown, Cost: MustCost("1c"), Copies: At(3, 4), Effects: Fx(Produce(Clay, 2))},
	{ID: 29, Slug: "foundry", Name: "Foundry", Color: Brown, Cost: MustCost("1c"), Copies: At(3, 4), Effects: Fx(Produce(Ore, 2))},
	{ID: 30, Slug: "quarry", Name: "Quarry", Color: Brown, Cost: MustCost("1c"), Copies: At(3, 4), Effects: Fx(Produce(Stone, 2))},
	{ID: 31, Slug: "sawmill", Name: "Sawmill", Color: Brown, Cost: MustCost("1c"), Copies: At(3, 4), Effects: Fx(Produce(Wood, 2))},

	// Grey: manufactured resources.
	{ID: 32, Slug: "glassworks-age-2", Name: "Glassworks", Color: Grey, Copies: At(3, 5), Effects: Fx(Produce(Glass, 1))},
	{ID: 33, Slug: "loom-age-2", Name: "Loom", Color: Grey, Copies: At(3, 5), Effects: Fx(Produce(Cloth, 1))},
	{ID: 34, Slug: "press-age-2", Name: "Press", Color: Grey, Copies: At(3, 5), Effects: Fx(Produce(Papyrus, 1))},

	// Red: military.
	{ID: 35, Slug: "archery-range", Name: "Archery Range", Color: Red, Cost: MustCost("WWO"), ChainIn: ChainArcheryRange, Copies: At(3, 6), Effects: Fx(Shields(2))},
	{ID: 36, Slug: "stables", Name: "Stables", Color: Red, Cost: MustCost("WCO"), ChainIn: ChainStables, Copies: At(3, 5), Effects: Fx(Shields(2))},
	{ID: 37, Slug: "training-ground", Name: "Training Ground", Color: Red, Cost: MustCost("WOO"), ChainOut: Out(ChainCircus), Copies: At(4, 6, 7), Effects: Fx(Shields(2))},
	{ID: 38, Slug: "walls", Name: "Walls", Color: Red, Cost: MustCost("SSS"), ChainOut: Out(ChainFortifications), Copies: At(3, 7), Effects: Fx(Shields(2))},

	// Blue: civic buildings, points.
	{ID: 39, Slug: "aqueduct", Name: "Aqueduct", Color: Blue, Cost: MustCost("SSS"), ChainIn: ChainAqueduct, Copies: At(3, 7), Effects: Fx(Points(5))},
	{ID: 40, Slug: "courthouse", Name: "Courthouse", Color: Blue, Cost: MustCost("CCL"), ChainIn: ChainCourthouse, Copies: At(3, 5), Effects: Fx(Points(4))},
	{ID: 41, Slug: "statue", Name: "Statue", Color: Blue, Cost: MustCost("WOO"), ChainIn: ChainStatue, Copies: At(3, 7), Effects: Fx(Points(4))},
	{ID: 42, Slug: "temple", Name: "Temple", Color: Blue, Cost: MustCost("WCG"), Copies: At(3, 6), Effects: Fx(Points(4))},

	// Yellow: commerce.
	{ID: 43, Slug: "bazar", Name: "Bazar", Color: Yellow, Copies: At(4, 7), Effects: Fx(CoinsPerCard(Self|Left|Right, 2, Grey))},
	{ID: 44, Slug: "caravansery", Name: "Caravansery", Color: Yellow, Cost: MustCost("WW"), ChainIn: ChainCaravansery, ChainOut: Out(ChainLighthouse), Copies: At(3, 5, 6), Effects: Fx(ProduceFamily(Raw))},
	{ID: 45, Slug: "forum", Name: "Forum", Color: Yellow, Cost: MustCost("CC"), ChainIn: ChainForum, ChainOut: Out(ChainHaven), Copies: At(3, 6, 7), Effects: Fx(ProduceFamily(Manufactured))},
	{ID: 46, Slug: "vineyard", Name: "Vineyard", Color: Yellow, Copies: At(3, 6), Effects: Fx(CoinsPerCard(Self|Left|Right, 1, Brown))},

	// Green: science.
	{ID: 47, Slug: "dispensary", Name: "Dispensary", Color: Green, Cost: MustCost("OOG"), ChainIn: ChainDispensary, ChainOut: Out(ChainArena, ChainLodge), Copies: At(3, 4), Effects: Fx(Science(Compass))},
	{ID: 48, Slug: "laboratory", Name: "Laboratory", Color: Green, Cost: MustCost("CCP"), ChainIn: ChainLaboratory, ChainOut: Out(ChainSiegeWorkshop, ChainObservatory), Copies: At(3, 5), Effects: Fx(Science(Gear))},
	{ID: 49, Slug: "library", Name: "Library", Color: Green, Cost: MustCost("SSL"), ChainIn: ChainLibrary, ChainOut: Out(ChainSenate, ChainUniversity), Copies: At(3, 6), Effects: Fx(Science(Tablet))},
	{ID: 50, Slug: "school", Name: "School", Color: Green, Cost: MustCost("WP"), ChainOut: Out(ChainAcademy, ChainStudy), Copies: At(3, 7), Effects: Fx(Science(Tablet))},
}
