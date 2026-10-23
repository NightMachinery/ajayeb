package catalog

// The 10 guilds, Age III designs with no copy markers: N + 2 of them are
// drawn at random for N players. In id order, by English name.
var guildDesigns = []Design{
	{ID: 71, Slug: "builders-guild", Name: "Builders Guild", Color: Purple, Cost: MustCost("SSSCCG"), Effects: Fx(PointsPerStage(Self|Left|Right, 1))},
	{ID: 72, Slug: "craftsmens-guild", Name: "Craftsmens Guild", Color: Purple, Cost: MustCost("SSOO"), Effects: Fx(PointsPerCard(Left|Right, 2, Grey))},
	{ID: 73, Slug: "decorators-guild", Name: "Decorators Guild", Color: Purple, Cost: MustCost("SOOL"), Effects: Fx(PointsIfWonderComplete(7))},
	{ID: 74, Slug: "magistrates-guild", Name: "Magistrates Guild", Color: Purple, Cost: MustCost("WWWSL"), Effects: Fx(PointsPerCard(Left|Right, 1, Blue))},
	{ID: 75, Slug: "philosophers-guild", Name: "Philosophers Guild", Color: Purple, Cost: MustCost("CCCPL"), Effects: Fx(PointsPerCard(Left|Right, 1, Green))},
	{ID: 76, Slug: "scientists-guild", Name: "Scientists Guild", Color: Purple, Cost: MustCost("WWOOP"), Effects: Fx(ScienceChoice())},
	{ID: 77, Slug: "shipowners-guild", Name: "Shipowners Guild", Color: Purple, Cost: MustCost("WWWGP"), Effects: Fx(PointsPerCard(Self, 1, Brown, Grey, Purple))},
	{ID: 78, Slug: "spies-guild", Name: "Spies Guild", Color: Purple, Cost: MustCost("CCG"), Effects: Fx(PointsPerCard(Left|Right, 1, Red))},
	{ID: 79, Slug: "traders-guild", Name: "Traders Guild", Color: Purple, Cost: MustCost("GPL"), Effects: Fx(PointsPerCard(Left|Right, 1, Yellow))},
	{ID: 80, Slug: "workers-guild", Name: "Workers Guild", Color: Purple, Cost: MustCost("WSCOO"), Effects: Fx(PointsPerCard(Left|Right, 1, Brown))},
}
