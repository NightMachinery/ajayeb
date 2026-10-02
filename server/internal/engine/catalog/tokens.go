package catalog

// The 10 progress tokens, in our canonical (alphabetical) order.
//
// Coins are paid once, when the token is taken (from the board or through
// the Great Library). The rule effects (discounts, trade income, the extra
// shield, the wonders' extra turn and the chain bonus) apply from then on;
// none of them reaches back to cards or wonders built earlier.
var tokenDefs = []TokenDef{
	{ID: TokenAgriculture, Slug: "agriculture", Name: "Agriculture", Effects: Fx(Coins(6), Points(4))},
	{ID: TokenArchitecture, Slug: "architecture", Name: "Architecture", Effects: Fx(Discount(ForWonders, 2))},
	{ID: TokenEconomy, Slug: "economy", Name: "Economy", Effects: Fx(TradeIncome())},
	{ID: TokenLaw, Slug: "law", Name: "Law", Effects: Fx(Science(Scales))},
	{ID: TokenMasonry, Slug: "masonry", Name: "Masonry", Effects: Fx(Discount(ForBlue, 2))},
	{ID: TokenMathematics, Slug: "mathematics", Name: "Mathematics", Effects: Fx(PointsPerToken(3))},
	{ID: TokenPhilosophy, Slug: "philosophy", Name: "Philosophy", Effects: Fx(Points(7))},
	{ID: TokenStrategy, Slug: "strategy", Name: "Strategy", Effects: Fx(ExtraShield(1))},
	{ID: TokenTheology, Slug: "theology", Name: "Theology", Effects: Fx(WondersPlayAgain())},
	{ID: TokenUrbanism, Slug: "urbanism", Name: "Urbanism", Effects: Fx(Coins(6), ChainBonus(4))},
}

var _ = registerTokens(tokenDefs)
