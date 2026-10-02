package catalog

// The 20 Age III buildings (ids 47 to 66), guilds excluded (see guilds.go).
// Age III has no brown or grey cards. The yellow cards pay their coins once,
// when built, counting only their owner's city, after the new card has joined
// it (so the Lighthouse counts itself).
var ageThreeCards = []CardDef{
	// Red: military.
	{ID: 47, Slug: "circus", Name: "Circus", Color: Red, Cost: MustCost("CC SS"), ChainIn: ChainHelmet,
		Effects: Fx(Shields(2))},
	{ID: 48, Slug: "arsenal", Name: "Arsenal", Color: Red, Cost: MustCost("CCC WW"), Effects: Fx(Shields(3))},
	{ID: 49, Slug: "siege-workshop", Name: "Siege Workshop", Color: Red, Cost: MustCost("WWW G"), ChainIn: ChainTarget,
		Effects: Fx(Shields(2))},
	{ID: 50, Slug: "fortifications", Name: "Fortifications", Color: Red, Cost: MustCost("SS C P"), ChainIn: ChainTower,
		Effects: Fx(Shields(2))},
	{ID: 51, Slug: "pretorium", Name: "Pretorium", AltNames: []string{"Courthouse"}, Color: Red, Cost: MustCost("8c"),
		Effects: Fx(Shields(3))},

	// Green: science.
	{ID: 52, Slug: "academy", Name: "Academy", Color: Green, Cost: MustCost("S W GG"),
		Effects: Fx(Science(Sundial), Points(3))},
	{ID: 53, Slug: "university", Name: "University", Color: Green, Cost: MustCost("C G P"), ChainIn: ChainLyre,
		Effects: Fx(Science(Globe), Points(2))},
	{ID: 54, Slug: "study", Name: "Study", Color: Green, Cost: MustCost("WW G P"),
		Effects: Fx(Science(Sundial), Points(3))},
	{ID: 55, Slug: "observatory", Name: "Observatory", Color: Green, Cost: MustCost("S PP"), ChainIn: ChainLamp,
		Effects: Fx(Science(Globe), Points(2))},

	// Yellow: commerce.
	{ID: 56, Slug: "arena", Name: "Arena", Color: Yellow, Cost: MustCost("C S W"), ChainIn: ChainBarrel,
		Effects: Fx(Points(3), CoinsPerWonder(ScopeSelf, 2))},
	{ID: 57, Slug: "chamber-of-commerce", Name: "Chamber of Commerce", Color: Yellow, Cost: MustCost("PP"),
		Effects: Fx(Points(3), CoinsPerCard(ScopeSelf, 3, Grey))},
	{ID: 58, Slug: "port", Name: "Port", Color: Yellow, Cost: MustCost("W G P"),
		Effects: Fx(Points(3), CoinsPerCard(ScopeSelf, 2, Brown))},
	{ID: 59, Slug: "lighthouse", Name: "Lighthouse", Color: Yellow, Cost: MustCost("CC G"), ChainIn: ChainAmphora,
		Effects: Fx(Points(3), CoinsPerCard(ScopeSelf, 1, Yellow))},
	{ID: 60, Slug: "armory", Name: "Armory", Color: Yellow, Cost: MustCost("SS G"),
		Effects: Fx(Points(3), CoinsPerCard(ScopeSelf, 1, Red))},

	// Blue: civilian.
	{ID: 61, Slug: "palace", Name: "Palace", Color: Blue, Cost: MustCost("C S W GG"), Effects: Fx(Points(7))},
	{ID: 62, Slug: "gardens", Name: "Gardens", Color: Blue, Cost: MustCost("CC WW"), ChainIn: ChainColumn,
		Effects: Fx(Points(6))},
	{ID: 63, Slug: "pantheon", Name: "Pantheon", Color: Blue, Cost: MustCost("C W PP"), ChainIn: ChainSun,
		Effects: Fx(Points(6))},
	{ID: 64, Slug: "town-hall", Name: "Town Hall", Color: Blue, Cost: MustCost("SSS WW"), Effects: Fx(Points(7))},
	{ID: 65, Slug: "senate", Name: "Senate", Color: Blue, Cost: MustCost("CC S P"), ChainIn: ChainTempleFacade,
		Effects: Fx(Points(5))},
	{ID: 66, Slug: "obelisk", Name: "Obelisk", Color: Blue, Cost: MustCost("SS G"), Effects: Fx(Points(5))},
}

var _ = registerCards(3, ageThreeCards)
