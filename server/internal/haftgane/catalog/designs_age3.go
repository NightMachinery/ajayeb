package catalog

// The 20 Age III designs other than the guilds, in id order: colour in the
// enum's order, then English name.
var ageThreeDesigns = []Design{
	// Red: military.
	{ID: 51, Slug: "arsenal", Name: "Arsenal", Color: Red, Cost: MustCost("WWOL"), Copies: At(3, 5), Effects: Fx(Shields(3))},
	{ID: 52, Slug: "castrum", Name: "Castrum", Color: Red, Cost: MustCost("WCCP"), Copies: At(4, 7), Effects: Fx(Shields(3))},
	{ID: 53, Slug: "circus", Name: "Circus", Color: Red, Cost: MustCost("SSSO"), ChainIn: ChainCircus, Copies: At(4, 6), Effects: Fx(Shields(3))},
	{ID: 54, Slug: "fortifications", Name: "Fortifications", Color: Red, Cost: MustCost("COOO"), ChainIn: ChainFortifications, Copies: At(3, 7), Effects: Fx(Shields(3))},
	{ID: 55, Slug: "siege-workshop", Name: "Siege Workshop", Color: Red, Cost: MustCost("WCCC"), ChainIn: ChainSiegeWorkshop, Copies: At(3, 5), Effects: Fx(Shields(3))},

	// Blue: civic buildings, points.
	{ID: 56, Slug: "gardens", Name: "Gardens", Color: Blue, Cost: MustCost("WCC"), ChainIn: ChainGardens, Copies: At(3, 4), Effects: Fx(Points(5))},
	{ID: 57, Slug: "palace", Name: "Palace", Color: Blue, Cost: MustCost("WSCOGPL"), Copies: At(3, 7), Effects: Fx(Points(8))},
	{ID: 58, Slug: "pantheon", Name: "Pantheon", Color: Blue, Cost: MustCost("CCOGPL"), ChainIn: ChainPantheon, Copies: At(3, 6), Effects: Fx(Points(7))},
	{ID: 59, Slug: "senate", Name: "Senate", Color: Blue, Cost: MustCost("WWSO"), ChainIn: ChainSenate, Copies: At(3, 5), Effects: Fx(Points(6))},
	{ID: 60, Slug: "town-hall", Name: "Town Hall", Color: Blue, Cost: MustCost("SSOG"), Copies: At(3, 6), Effects: Fx(Points(6))},

	// Yellow: commerce.
	{ID: 61, Slug: "arena", Name: "Arena", Color: Yellow, Cost: MustCost("SSO"), ChainIn: ChainArena, Copies: At(3, 5), Effects: Fx(CoinsPerStage(Self, 3), PointsPerStage(Self, 1))},
	{ID: 62, Slug: "chamber-of-commerce", Name: "Chamber of Commerce", Color: Yellow, Cost: MustCost("CCP"), Copies: At(4, 6), Effects: Fx(CoinsPerCard(Self, 2, Grey), PointsPerCard(Self, 2, Grey))},
	{ID: 63, Slug: "haven", Name: "Haven", Color: Yellow, Cost: MustCost("WOL"), ChainIn: ChainHaven, Copies: At(3, 4), Effects: Fx(CoinsPerCard(Self, 1, Brown), PointsPerCard(Self, 1, Brown))},
	{ID: 64, Slug: "lighthouse", Name: "Lighthouse", Color: Yellow, Cost: MustCost("SG"), ChainIn: ChainLighthouse, Copies: At(3, 6), Effects: Fx(CoinsPerCard(Self, 1, Yellow), PointsPerCard(Self, 1, Yellow))},
	{ID: 65, Slug: "ludus", Name: "Ludus", Color: Yellow, Cost: MustCost("SO"), Copies: At(5, 7), Effects: Fx(CoinsPerCard(Self, 3, Red), PointsPerCard(Self, 1, Red))},

	// Green: science.
	{ID: 66, Slug: "academy", Name: "Academy", Color: Green, Cost: MustCost("SSSG"), ChainIn: ChainAcademy, Copies: At(3, 7), Effects: Fx(Science(Compass))},
	{ID: 67, Slug: "lodge", Name: "Lodge", Color: Green, Cost: MustCost("CCPL"), ChainIn: ChainLodge, Copies: At(3, 6), Effects: Fx(Science(Compass))},
	{ID: 68, Slug: "observatory", Name: "Observatory", Color: Green, Cost: MustCost("OOGL"), ChainIn: ChainObservatory, Copies: At(3, 7), Effects: Fx(Science(Gear))},
	{ID: 69, Slug: "study", Name: "Study", Color: Green, Cost: MustCost("WPL"), ChainIn: ChainStudy, Copies: At(3, 5), Effects: Fx(Science(Gear))},
	{ID: 70, Slug: "university", Name: "University", Color: Green, Cost: MustCost("WWGP"), ChainIn: ChainUniversity, Copies: At(3, 4), Effects: Fx(Science(Tablet))},
}
