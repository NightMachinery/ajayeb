package catalog

// The seven boards, in id order. Each has a day side of three stages and a
// night side of two to four; the starting resource is the board's, the same
// on both sides.
var boardDefs = []Board{
	{ID: BoardAlexandria, Slug: "alexandria", Name: "Alexandria", Wonder: "the Lighthouse", Start: Glass, Sides: [2]SideDef{
		SideOf(
			StageOf(MustCost("SS"), Points(3)),
			StageOf(MustCost("OO"), ProduceFamily(Raw)),
			StageOf(MustCost("PL"), Points(7)),
		),
		SideOf(
			StageOf(MustCost("CC"), ProduceFamily(Raw)),
			StageOf(MustCost("OOO"), ProduceFamily(Manufactured)),
			StageOf(MustCost("WWWW"), Points(7)),
		),
	}},
	{ID: BoardBabylon, Slug: "babylon", Name: "Babylon", Wonder: "the Hanging Gardens", Start: Wood, Sides: [2]SideDef{
		SideOf(
			StageOf(MustCost("CC"), Points(3)),
			StageOf(MustCost("OOL"), ScienceChoice()),
			StageOf(MustCost("WWWW"), Points(7)),
		),
		SideOf(
			StageOf(MustCost("SS"), PlayLastCard()),
			StageOf(MustCost("CCCG"), ScienceChoice()),
		),
	}},
	{ID: BoardEphesos, Slug: "ephesos", Name: "Ephesos", Wonder: "the Temple of Artemis", Start: Papyrus, Sides: [2]SideDef{
		SideOf(
			StageOf(MustCost("CC"), Points(3)),
			StageOf(MustCost("WW"), Coins(9)),
			StageOf(MustCost("OOG"), Points(7)),
		),
		SideOf(
			StageOf(MustCost("SS"), Coins(4), Points(2)),
			StageOf(MustCost("WW"), Coins(4), Points(3)),
			StageOf(MustCost("OOL"), Coins(4), Points(5)),
		),
	}},
	{ID: BoardGizah, Slug: "gizah", Name: "Gizah", Wonder: "the Pyramids", Start: Stone, Sides: [2]SideDef{
		SideOf(
			StageOf(MustCost("WW"), Points(3)),
			StageOf(MustCost("CCL"), Points(5)),
			StageOf(MustCost("SSSS"), Points(7)),
		),
		SideOf(
			StageOf(MustCost("WW"), Points(3)),
			StageOf(MustCost("SSS"), Points(5)),
			StageOf(MustCost("CCC"), Points(5)),
			StageOf(MustCost("SSSSP"), Points(7)),
		),
	}},
	{ID: BoardHalikarnassos, Slug: "halikarnassos", Name: "Halikarnassos", Wonder: "the Mausoleum", Start: Cloth, Sides: [2]SideDef{
		SideOf(
			StageOf(MustCost("OO"), Points(3)),
			StageOf(MustCost("GP"), BuildFromDiscard()),
			StageOf(MustCost("SSS"), Points(7)),
		),
		SideOf(
			StageOf(MustCost("CC"), BuildFromDiscard(), Points(2)),
			StageOf(MustCost("GP"), BuildFromDiscard(), Points(1)),
			StageOf(MustCost("WWW"), BuildFromDiscard()),
		),
	}},
	{ID: BoardOlympia, Slug: "olympia", Name: "Olympia", Wonder: "the Statue of Zeus", Start: Clay, Sides: [2]SideDef{
		SideOf(
			StageOf(MustCost("SS"), Points(3)),
			StageOf(MustCost("WW"), FreeBuild(FirstOfColour)),
			StageOf(MustCost("CCC"), Points(7)),
		),
		SideOf(
			StageOf(MustCost("OO"), FreeBuild(FirstTurnOfAge), Points(2)),
			StageOf(MustCost("CCC"), FreeBuild(LastTurnOfAge), Points(3)),
			StageOf(MustCost("GPL"), Points(5)),
		),
	}},
	{ID: BoardRhodos, Slug: "rhodos", Name: "Rhodos", Wonder: "the Colossus", Start: Ore, Sides: [2]SideDef{
		SideOf(
			StageOf(MustCost("WW"), Points(3)),
			StageOf(MustCost("CCC"), Shields(2)),
			StageOf(MustCost("OOOO"), Points(7)),
		),
		SideOf(
			StageOf(MustCost("SSS"), Coins(3), Points(3), Shields(1)),
			StageOf(MustCost("OOOO"), Coins(4), Points(4), Shields(1)),
		),
	}},
}
