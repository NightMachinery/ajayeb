package catalog

// The 7 guilds (ids 67 to 73). They belong to Age III, where 3 of them are
// shuffled into the deck, and they have their own back (GuildBack, derived at
// registration).
//
// A guild counts the city that has the most of what it counts: its owner's
// or the opponent's, decided separately for each effect and at the moment the
// effect applies (coins when the guild is built, points at the end of the
// game). The Shipowners Guild adds brown and grey cards together and takes the
// city with the larger total.
var guildCards = []CardDef{
	{ID: 67, Slug: "merchants-guild", Name: "Merchants Guild", AltNames: []string{"Traders Guild"}, Color: Purple,
		Cost:    MustCost("C W G P"),
		Effects: Fx(CoinsPerCard(ScopeMaxCity, 1, Yellow), PointsPerCard(ScopeMaxCity, 1, Yellow))},
	{ID: 68, Slug: "shipowners-guild", Name: "Shipowners Guild", Color: Purple,
		Cost:    MustCost("C S G P"),
		Effects: Fx(CoinsPerCard(ScopeMaxCity, 1, Brown, Grey), PointsPerCard(ScopeMaxCity, 1, Brown, Grey))},
	{ID: 69, Slug: "builders-guild", Name: "Builders Guild", Color: Purple,
		Cost:    MustCost("SS C W G"),
		Effects: Fx(PointsPerWonder(ScopeMaxCity, 2))},
	{ID: 70, Slug: "magistrates-guild", Name: "Magistrates Guild", Color: Purple,
		Cost:    MustCost("WW C P"),
		Effects: Fx(CoinsPerCard(ScopeMaxCity, 1, Blue), PointsPerCard(ScopeMaxCity, 1, Blue))},
	{ID: 71, Slug: "scientists-guild", Name: "Scientists Guild", Color: Purple,
		Cost:    MustCost("CC WW"),
		Effects: Fx(CoinsPerCard(ScopeMaxCity, 1, Green), PointsPerCard(ScopeMaxCity, 1, Green))},
	{ID: 72, Slug: "moneylenders-guild", Name: "Moneylenders Guild", Color: Purple,
		Cost:    MustCost("SS WW"),
		Effects: Fx(PointsPerCoins(ScopeMaxCity, 3))},
	{ID: 73, Slug: "tacticians-guild", Name: "Tacticians Guild", Color: Purple,
		Cost:    MustCost("SS C P"),
		Effects: Fx(CoinsPerCard(ScopeMaxCity, 1, Red), PointsPerCard(ScopeMaxCity, 1, Red))},
}

var _ = registerCards(3, guildCards)
