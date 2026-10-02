package catalog

// The 12 wonders, in our canonical order (the id constants in types.go).
// Names drop the leading "The" of the printed titles, like the slugs.
//
// Effects are listed in a fixed order for readability: the special effect
// first (production, destroy, rebuild, token draw), then shields, coins, the
// opponent's coin loss, the extra turn, and the victory points last. The
// engine applies them in its own rules order, not in this order.
var wonderDefs = []WonderDef{
	{ID: WonderPyramids, Slug: "pyramids", Name: "Pyramids", Cost: MustCost("SSS P"),
		Effects: Fx(Points(9))},
	{ID: WonderColossus, Slug: "colossus", Name: "Colossus", Cost: MustCost("CCC G"),
		Effects: Fx(Shields(2), Points(3))},
	{ID: WonderGreatLighthouse, Slug: "great-lighthouse", Name: "Great Lighthouse", Cost: MustCost("PP S W"),
		Effects: Fx(ProduceChoice(Wood, Clay, Stone), Points(4))},
	{ID: WonderTempleOfArtemis, Slug: "temple-of-artemis", Name: "Temple of Artemis", Cost: MustCost("W S G P"),
		Effects: Fx(Coins(12), PlayAgain())},
	{ID: WonderMausoleum, Slug: "mausoleum", Name: "Mausoleum", Cost: MustCost("CC GG P"),
		Effects: Fx(BuildFromDiscard(), Points(2))},
	{ID: WonderGreatLibrary, Slug: "great-library", Name: "Great Library", Cost: MustCost("WWW G P"),
		Effects: Fx(DrawTokens(3), Points(4))},
	{ID: WonderPiraeus, Slug: "piraeus", Name: "Piraeus", Cost: MustCost("WW C S"),
		Effects: Fx(ProduceChoice(Glass, Papyrus), PlayAgain(), Points(2))},
	{ID: WonderHangingGardens, Slug: "hanging-gardens", Name: "Hanging Gardens", Cost: MustCost("WW G P"),
		Effects: Fx(Coins(6), PlayAgain(), Points(3))},
	{ID: WonderStatueOfZeus, Slug: "statue-of-zeus", Name: "Statue of Zeus", Cost: MustCost("PP C W S"),
		Effects: Fx(Destroy(Brown), Shields(1), Points(3))},
	{ID: WonderSphinx, Slug: "sphinx", Name: "Sphinx", Cost: MustCost("S C GG"),
		Effects: Fx(PlayAgain(), Points(6))},
	{ID: WonderAppianWay, Slug: "appian-way", Name: "Appian Way", Cost: MustCost("SS CC P"),
		Effects: Fx(Coins(3), OpponentLosesCoins(3), PlayAgain(), Points(3))},
	{ID: WonderCircusMaximus, Slug: "circus-maximus", Name: "Circus Maximus", Cost: MustCost("SS W G"),
		Effects: Fx(Destroy(Grey), Shields(1), Points(3))},
}

var _ = registerWonders(wonderDefs)
