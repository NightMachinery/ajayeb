package catalog

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
)

// sampleCards, sampleWonders and sampleTokens are a few real items, enough to
// exercise the registry without the full data.
func sampleCards() []CardDef {
	return []CardDef{
		{ID: 68, Slug: "shipowners-guild", Name: "Shipowners Guild", Age: 3, Color: Purple, Cost: MustCost("C S G P"),
			Effects: Fx(CoinsPerCard(ScopeMaxCity, 1, Brown, Grey), PointsPerCard(ScopeMaxCity, 1, Brown, Grey))},
		{ID: 1, Slug: "lumber-yard", Name: "Lumber Yard", Age: 1, Color: Brown, Effects: Fx(Produce(Wood, 1))},
		{ID: 9, Slug: "stable", Name: "Stable", Age: 1, Color: Red, Cost: MustCost("W"), ChainOut: ChainHorseshoe,
			Effects: Fx(Shields(1))},
		{ID: 29, Slug: "horse-breeders", Name: "Horse Breeders", Age: 2, Color: Red, Cost: MustCost("C W"),
			ChainIn: ChainHorseshoe, Effects: Fx(Shields(1))},
		{ID: 59, Slug: "lighthouse", Name: "Lighthouse", Age: 3, Color: Yellow, Cost: MustCost("CC G"),
			ChainIn: ChainAmphora, Effects: Fx(Points(3), CoinsPerCard(ScopeSelf, 1, Yellow))},
	}
}

func sampleWonders() []WonderDef {
	return []WonderDef{
		{ID: WonderStatueOfZeus, Slug: "statue-of-zeus", Name: "Statue of Zeus", Cost: MustCost("PP C W S"),
			Effects: Fx(Destroy(Brown), Shields(1), Points(3))},
		{ID: WonderPyramids, Slug: "pyramids", Name: "Pyramids", Cost: MustCost("SSS P"), Effects: Fx(Points(9))},
	}
}

func sampleTokens() []TokenDef {
	return []TokenDef{
		{ID: TokenUrbanism, Slug: "urbanism", Name: "Urbanism", Effects: Fx(Coins(6), ChainBonus(4))},
		{ID: TokenLaw, Slug: "law", Name: "Law", Effects: Fx(Science(Scales))},
	}
}

func useSample(t *testing.T) {
	t.Helper()
	t.Cleanup(ReplaceForTest(sampleCards(), sampleWonders(), sampleTokens()))
}

func TestCardSet(t *testing.T) {
	var s CardSet
	if !s.Empty() || s.Count() != 0 {
		t.Fatal("zero CardSet is not empty")
	}
	ids := []CardID{1, 2, 63, 64, 65, 73, 127}
	for _, id := range ids {
		s.Add(id)
	}
	if s.Count() != len(ids) {
		t.Fatalf("Count = %d, want %d", s.Count(), len(ids))
	}
	for id := 0; id < 256; id++ {
		want := slices.Contains(ids, CardID(id))
		if got := s.Has(CardID(id)); got != want {
			t.Errorf("Has(%d) = %v, want %v", id, got, want)
		}
	}
	if s.Has(Unknown) {
		t.Error("Has(Unknown) = true")
	}
	if got := slices.Collect(s.All()); !slices.Equal(got, ids) {
		t.Errorf("All = %v, want %v", got, ids)
	}
	var viaNext []CardID
	for id, ok := s.Next(1); ok; id, ok = s.Next(id + 1) {
		viaNext = append(viaNext, id)
	}
	if !slices.Equal(viaNext, ids) {
		t.Errorf("Next loop = %v, want %v", viaNext, ids)
	}
	if id, ok := s.Next(66); !ok || id != 73 {
		t.Errorf("Next(66) = %d, %v; want 73, true", id, ok)
	}
	if _, ok := s.Next(128); ok {
		t.Error("Next(128) found a card")
	}
	if _, ok := s.Next(255); ok {
		t.Error("Next(255) found a card")
	}
	s.Remove(64)
	s.Remove(64) // removing an absent card is a no-op
	if s.Has(64) || s.Count() != len(ids)-1 {
		t.Error("Remove(64) failed")
	}

	a, b := CardSetOf(1, 2, 70), CardSetOf(2, 3, 70, 71)
	if got := a.And(b); got != CardSetOf(2, 70) {
		t.Errorf("And = %v", slices.Collect(got.All()))
	}
	if got := a.Or(b); got != CardSetOf(1, 2, 3, 70, 71) {
		t.Errorf("Or = %v", slices.Collect(got.All()))
	}
	if got := a.AndNot(b); got != CardSetOf(1) {
		t.Errorf("AndNot = %v", slices.Collect(got.All()))
	}
	// Early exit from the iterator.
	for id := range a.All() {
		if id != 1 {
			t.Errorf("first id = %d", id)
		}
		break
	}
	mustPanic(t, "CardSet.Add(Unknown)", func() { var x CardSet; x.Add(Unknown) })
}

func TestTokenSet(t *testing.T) {
	var s TokenSet
	if !s.Empty() {
		t.Fatal("zero TokenSet is not empty")
	}
	for _, id := range []TokenID{TokenLaw, TokenAgriculture, TokenUrbanism} {
		s.Add(id)
	}
	want := []TokenID{TokenAgriculture, TokenLaw, TokenUrbanism}
	if got := slices.Collect(s.All()); !slices.Equal(got, want) {
		t.Errorf("All = %v, want %v", got, want)
	}
	if s.Count() != 3 || !s.Has(TokenLaw) || s.Has(TokenEconomy) || s.Has(Unknown) {
		t.Error("membership is wrong")
	}
	if s != TokenSetOf(want...) {
		t.Error("TokenSetOf disagrees with Add")
	}
	s.Remove(TokenLaw)
	s.Remove(Unknown) // never a member: no-op
	if s.Has(TokenLaw) || s.Count() != 2 {
		t.Error("Remove failed")
	}
	if _, ok := s.Next(TokenUrbanism + 1); ok {
		t.Error("Next past the last token found one")
	}
	if _, ok := s.Next(16); ok {
		t.Error("Next(16) found a token")
	}
	mustPanic(t, "TokenSet.Add(Unknown)", func() { var x TokenSet; x.Add(Unknown) })
}

func TestEnumsAndMasks(t *testing.T) {
	if RawGoods&Manufactured != 0 || RawGoods|Manufactured != AllResources || AllResources != 0b11111 {
		t.Error("resource classes are not a partition of the five resources")
	}
	if MaskOf(Wood, Clay, Stone) != RawGoods || MaskOf(Glass, Papyrus) != Manufactured {
		t.Error("MaskOf disagrees with the class masks")
	}
	if !RawGoods.Has(Stone) || RawGoods.Has(Glass) || RawGoods.Has(NumResources) {
		t.Error("ResMask.Has is wrong")
	}
	letters := ""
	for r := range NumResources {
		letters += string(r.Letter())
	}
	if letters != "WCSGP" || NumResources.Letter() != '?' {
		t.Errorf("letters = %q", letters)
	}
	if Glass.String() != "glass" || Color(99).String() != "color#99" {
		t.Error("stringers are wrong")
	}
	if !ColorsOf(Brown, Grey).Has(Grey) || ColorsOf(Brown).Has(Grey) || ColorsOf(Purple).Has(NumColors) {
		t.Error("ColorMask.Has is wrong")
	}
	if ChainTempleFacade.String() != "temple-facade" || SetSquare.String() != "set-square" || Purple.String() != "purple" {
		t.Error("kebab names are wrong")
	}
	if NoChain.Bit() != 0 || ChainSword.Bit() != 1<<2 || NumChains > 32 {
		t.Error("chain bits are wrong")
	}
	for _, c := range []struct {
		age  uint8
		want Back
	}{{0, NoBack}, {1, BackAge1}, {2, BackAge2}, {3, BackAge3}, {4, NoBack}} {
		if got := AgeBack(c.age); got != c.want {
			t.Errorf("AgeBack(%d) = %s, want %s", c.age, got, c.want)
		}
	}
	if GuildBack.String() != "guild" || BackAge2.String() != "age2" {
		t.Error("back names are wrong")
	}
	// Every name table is complete and every name is kebab-case.
	for _, names := range [][]string{resourceNames[:], colorNames[:], symbolNames[:], chainNames[:],
		backNames[:], scopeNames[:], targetNames[:], effectKindNames[:]} {
		for i, n := range names {
			if err := checkSlug(n); err != nil {
				t.Errorf("name %d %q: %v", i, n, err)
			}
		}
	}
	if WonderCircusMaximus != NumWonders || TokenUrbanism != NumTokens {
		t.Error("the id constants do not cover the counts")
	}
	seen := map[string]bool{}
	for _, s := range append(wonderSlugs[1:], tokenSlugs[1:]...) {
		if err := checkSlug(s); err != nil || seen[s] {
			t.Errorf("canonical slug %q is malformed or repeated", s)
		}
		seen[s] = true
	}
}

func TestParseCost(t *testing.T) {
	cost := func(coins uint8, res string) Cost {
		c := Cost{Coins: coins}
		for i := 0; i < len(res); i++ {
			c.Res[strings.IndexByte(resourceLetters, res[i])]++
		}
		return c
	}
	good := []struct {
		in   string
		want Cost
	}{
		{"", Cost{}},
		{"free", Cost{}},
		{"  free ", Cost{}},
		{"W", cost(0, "W")},
		{"3c", cost(3, "")},
		{"8c", cost(8, "")},
		{"10c", cost(10, "")},
		{"3c + C", cost(3, "C")},
		{"2c + G P", cost(2, "GP")},
		{"2c G P", cost(2, "GP")},
		{"SS C W G", cost(0, "SSCWG")},
		{"CCC WW", cost(0, "CCCWW")},
		{"C S W GG", cost(0, "CSWGG")},
		{"PP C W S", cost(0, "PPCWS")},
		{"SSS P", cost(0, "SSSP")},
		{"S\tW  P", cost(0, "SWP")},
	}
	for _, c := range good {
		got, err := ParseCost(c.in)
		if err != nil || got != c.want {
			t.Errorf("ParseCost(%q) = %+v, %v; want %+v", c.in, got, err, c.want)
		}
		if again, err := ParseCost(got.String()); err != nil || again != got {
			t.Errorf("round trip of %q via %q = %+v, %v", c.in, got.String(), again, err)
		}
	}
	bad := []string{
		"3", "c", "0c", "03c", "256c", "-1c", "3c 2c", "C 3c", "3c +", "+ 3c", "3c + + C",
		"+", "SW", "S S", "SS S", "w", "x", "Free", "FREE", "free W", "3C", "3c+C", "S,W",
		"1.5c", "WWWx",
	}
	for _, in := range bad {
		if got, err := ParseCost(in); err == nil {
			t.Errorf("ParseCost(%q) = %+v, want an error", in, got)
		} else if !errors.Is(err, ErrBadCost) {
			t.Errorf("ParseCost(%q) error %v does not wrap ErrBadCost", in, err)
		}
	}
	if got := cost(2, "GP").String(); got != "2c G P" {
		t.Errorf("String = %q, want %q", got, "2c G P")
	}
	if got := cost(0, "SSCWG").String(); got != "W C SS G" {
		t.Errorf("String = %q, want %q", got, "W C SS G")
	}
	if (Cost{}).String() != "free" || !(Cost{}).Free() || cost(1, "").Free() {
		t.Error("free cost handling is wrong")
	}
	if cost(3, "SSCWG").Units() != 5 {
		t.Error("Units is wrong")
	}
	mustPanic(t, `MustCost("SW")`, func() { MustCost("SW") })
}

func TestEffectConstructors(t *testing.T) {
	cases := []struct {
		got, want Effect
	}{
		{Produce(Stone, 2), Effect{Kind: EffProduce, Res: Stone, N: 2}},
		{ProduceChoice(Wood, Clay, Stone), Effect{Kind: EffProduceChoice, Mask: RawGoods}},
		{TradeFix(Glass, Papyrus), Effect{Kind: EffTradeFix, Mask: Manufactured}},
		{Shields(2), Effect{Kind: EffShields, N: 2}},
		{Science(Wheel), Effect{Kind: EffScience, Symbol: Wheel}},
		{Points(7), Effect{Kind: EffPoints, N: 7}},
		{Coins(4), Effect{Kind: EffCoins, N: 4}},
		{CoinsPerCard(ScopeSelf, 1, Yellow), Effect{Kind: EffCoinsPerCard, N: 1, Colors: Yellow.Mask(), Scope: ScopeSelf}},
		{CoinsPerWonder(ScopeSelf, 2), Effect{Kind: EffCoinsPerWonder, N: 2, Scope: ScopeSelf}},
		{PointsPerCard(ScopeMaxCity, 1, Brown, Grey), Effect{Kind: EffPointsPerCard, N: 1, Colors: ColorsOf(Brown, Grey), Scope: ScopeMaxCity}},
		{PointsPerWonder(ScopeMaxCity, 2), Effect{Kind: EffPointsPerWonder, N: 2, Scope: ScopeMaxCity}},
		{PointsPerCoins(ScopeMaxCity, 3), Effect{Kind: EffPointsPerCoins, N: 3, Scope: ScopeMaxCity}},
		{PointsPerToken(3), Effect{Kind: EffPointsPerToken, N: 3}},
		{PlayAgain(), Effect{Kind: EffPlayAgain}},
		{Destroy(Grey), Effect{Kind: EffDestroy, Colors: Grey.Mask()}},
		{BuildFromDiscard(), Effect{Kind: EffBuildFromDiscard}},
		{DrawTokens(3), Effect{Kind: EffDrawTokens, N: 3}},
		{OpponentLosesCoins(3), Effect{Kind: EffOpponentLosesCoins, N: 3}},
		{Discount(ForBlue, 2), Effect{Kind: EffDiscount, N: 2, Target: ForBlue}},
		{TradeIncome(), Effect{Kind: EffTradeIncome}},
		{ExtraShield(1), Effect{Kind: EffExtraShield, N: 1}},
		{WondersPlayAgain(), Effect{Kind: EffWondersPlayAgain}},
		{ChainBonus(4), Effect{Kind: EffChainBonus, N: 4}},
	}
	kinds := map[EffectKind]bool{}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("constructor gave %+v, want %+v", c.got, c.want)
		}
		if err := c.got.Validate(); err != nil {
			t.Errorf("%v: %v", c.got, err)
		}
		if c.got.String() == "" {
			t.Errorf("%+v has an empty String", c.got)
		}
		kinds[c.got.Kind] = true
	}
	if len(kinds) != int(numEffectKinds)-1 {
		t.Errorf("constructors cover %d of %d effect kinds", len(kinds), numEffectKinds-1)
	}
	if got := Fx(Points(1), Coins(2)); len(got) != 2 || got[1] != Coins(2) {
		t.Errorf("Fx = %v", got)
	}
	if e, ok := FindEffect(Fx(Points(3), CoinsPerCard(ScopeSelf, 1, Yellow)), EffCoinsPerCard); !ok || e.N != 1 {
		t.Error("FindEffect missed an effect")
	}
	if _, ok := FindEffect(Fx(Points(3)), EffShields); ok {
		t.Error("FindEffect found a missing effect")
	}
	if got := CoinsPerCard(ScopeSelf, 1, Yellow).String(); got != "coins-per-card 1 yellow self" {
		t.Errorf("String = %q", got)
	}
	if got := Produce(Stone, 2).String(); got != "produce S2" {
		t.Errorf("String = %q", got)
	}
}

func TestEffectValidateRejects(t *testing.T) {
	bad := []Effect{
		{},                                     // EffNone
		{Kind: numEffectKinds},                 // unknown kind
		Produce(Wood, 0),                       // N must be positive
		Produce(NumResources, 1),               // bad resource
		ProduceChoice(),                        // empty mask
		{Kind: EffProduceChoice, Mask: 1 << 6}, // bit outside the resources
		Shields(0),
		{Kind: EffShields, N: 1, Res: Clay}, // field unused by the kind
		{Kind: EffShields, N: 1, Scope: ScopeSelf},   // field unused by the kind
		{Kind: EffPoints, N: 1, Symbol: Globe},       // field unused by the kind
		{Kind: EffPlayAgain, N: 1},                   // field unused by the kind
		{Kind: EffTradeIncome, Target: ForBlue},      // field unused by the kind
		{Kind: EffCoins, N: 1, Colors: Brown.Mask()}, // field unused by the kind
		{Kind: EffCoins, N: 1, Mask: RawGoods},       // field unused by the kind
		Science(NoSymbol),
		Science(NumSymbols),
		CoinsPerCard(NoScope, 1, Yellow), // missing scope
		CoinsPerCard(ScopeSelf, 1),       // no colours
		CoinsPerCard(ScopeSelf, 1, NoColor),
		{Kind: EffCoinsPerCard, N: 1, Scope: ScopeSelf, Colors: NoColor.Mask() | Yellow.Mask()},
		PointsPerCard(Scope(9), 1, Blue),
		PointsPerWonder(NoScope, 2),
		PointsPerCoins(ScopeMaxCity, 0),
		Destroy(NoColor),
		{Kind: EffDestroy, Colors: ColorsOf(Brown, Grey)}, // destroy takes one colour
		Discount(NoTarget, 2),
		Discount(DiscountTarget(7), 2),
		Discount(ForWonders, 0),
	}
	for _, e := range bad {
		if err := e.Validate(); err == nil {
			t.Errorf("Validate(%+v) = nil, want an error", e)
		}
	}
}

func TestCheckSlug(t *testing.T) {
	for _, s := range []string{"law", "lumber-yard", "statue-of-zeus", "age1", "a-1-b"} {
		if err := checkSlug(s); err != nil {
			t.Errorf("checkSlug(%q) = %v", s, err)
		}
	}
	for _, s := range []string{"", "the-pyramids", "Lumber-yard", "lumber_yard", "lumber yard", "-law", "law-",
		"lumber--yard", "caf\u00e9", "law\n", strings.Repeat("a", maxSlugLen+1)} {
		if err := checkSlug(s); err == nil {
			t.Errorf("checkSlug(%q) = nil, want an error", s)
		}
	}
}

func TestRegistryEmpty(t *testing.T) {
	t.Cleanup(ReplaceForTest(nil, nil, nil))
	if Card(1) != nil || Wonder(1) != nil || Token(1) != nil || Card(0) != nil || Card(Unknown) != nil {
		t.Error("an empty catalog returned an item")
	}
	if _, ok := CardBySlug("lumber-yard"); ok {
		t.Error("CardBySlug found a card in an empty catalog")
	}
	if _, ok := WonderBySlug("pyramids"); ok {
		t.Error("WonderBySlug found a wonder in an empty catalog")
	}
	if _, ok := TokenBySlug("law"); ok {
		t.Error("TokenBySlug found a token in an empty catalog")
	}
	n := 0
	for range Cards() {
		n++
	}
	for range Wonders() {
		n++
	}
	for range Tokens() {
		n++
	}
	if n != 0 {
		t.Errorf("iterators yielded %d items", n)
	}
	if _, err := CardID(1).MarshalText(); err == nil {
		t.Error("MarshalText of an unregistered card succeeded")
	}
	if _, err := WonderID(1).MarshalText(); err == nil {
		t.Error("MarshalText of an unregistered wonder succeeded")
	}
	if _, err := TokenID(1).MarshalText(); err == nil {
		t.Error("MarshalText of an unregistered token succeeded")
	}
	var c CardID
	if err := c.UnmarshalText([]byte("lumber-yard")); err == nil {
		t.Error("UnmarshalText found a card in an empty catalog")
	}
	if CardID(9).String() != "card#9" || WonderID(2).String() != "wonder#2" || TokenID(3).String() != "token#3" {
		t.Error("String of unregistered ids is wrong")
	}
	if CardID(Unknown).String() != "unknown" || WonderID(Unknown).String() != "unknown" || TokenID(Unknown).String() != "unknown" {
		t.Error("String of Unknown is wrong")
	}
}

func TestRegistrySample(t *testing.T) {
	useSample(t)

	d := Card(68)
	if d == nil || d.Slug != "shipowners-guild" || d.Age != 3 || d.Back != GuildBack || !d.IsGuild() {
		t.Fatalf("guild = %+v", d)
	}
	if d := Card(29); d == nil || d.Age != 2 || d.Back != BackAge2 || d.IsGuild() {
		t.Errorf("horse-breeders = %+v", d)
	}
	if d := Card(1); d == nil || d.Back != BackAge1 || d.Cost != (Cost{}) {
		t.Errorf("lumber-yard = %+v", d)
	}
	if Card(2) != nil || Card(74) != nil || Card(Unknown) != nil {
		t.Error("an unregistered id returned a card")
	}
	if id, ok := CardBySlug("stable"); !ok || id != 9 {
		t.Errorf("CardBySlug(stable) = %d, %v", id, ok)
	}
	for _, s := range []string{"Stable", "the-stable", " stable", "pyramids", "law", ""} {
		if _, ok := CardBySlug(s); ok {
			t.Errorf("CardBySlug(%q) matched", s)
		}
	}
	if id, ok := WonderBySlug("statue-of-zeus"); !ok || id != WonderStatueOfZeus {
		t.Errorf("WonderBySlug = %d, %v", id, ok)
	}
	if _, ok := WonderBySlug("stable"); ok {
		t.Error("WonderBySlug matched a card slug")
	}
	if id, ok := TokenBySlug("law"); !ok || id != TokenLaw {
		t.Errorf("TokenBySlug = %d, %v", id, ok)
	}

	var ids []CardID
	for d := range Cards() {
		ids = append(ids, d.ID)
	}
	if !slices.Equal(ids, []CardID{1, 9, 29, 59, 68}) {
		t.Errorf("Cards order = %v", ids)
	}
	var wids []WonderID
	for d := range Wonders() {
		wids = append(wids, d.ID)
	}
	if !slices.Equal(wids, []WonderID{WonderPyramids, WonderStatueOfZeus}) {
		t.Errorf("Wonders order = %v", wids)
	}
	var tids []TokenID
	for d := range Tokens() {
		tids = append(tids, d.ID)
	}
	if !slices.Equal(tids, []TokenID{TokenLaw, TokenUrbanism}) {
		t.Errorf("Tokens order = %v", tids)
	}
	for d := range Cards() { // early exit
		_ = d
		break
	}

	// Text and JSON codecs use slugs.
	type wire struct {
		Card   CardID           `json:"card"`
		Wonder WonderID         `json:"wonder"`
		Token  TokenID          `json:"token"`
		Cards  []CardID         `json:"cards"`
		ByCard map[CardID]uint8 `json:"byCard"`
	}
	in := wire{Card: 9, Wonder: WonderPyramids, Token: TokenLaw, Cards: []CardID{1, 68}, ByCard: map[CardID]uint8{59: 2}}
	b, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"card":"stable","wonder":"pyramids","token":"law","cards":["lumber-yard","shipowners-guild"],"byCard":{"lighthouse":2}}`
	if string(b) != want {
		t.Errorf("json = %s\nwant   %s", b, want)
	}
	var out wire
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out.Card != in.Card || out.Wonder != in.Wonder || out.Token != in.Token ||
		!slices.Equal(out.Cards, in.Cards) || out.ByCard[59] != 2 {
		t.Errorf("json round trip = %+v", out)
	}
	for _, bad := range []string{`{"card":"Stable"}`, `{"card":9}`, `{"wonder":"the-pyramids"}`,
		`{"token":"stable"}`, `{"cards":["lumber-yard","nope"]}`} {
		if err := json.Unmarshal([]byte(bad), &out); err == nil {
			t.Errorf("json.Unmarshal(%s) succeeded", bad)
		}
	}
	if _, err := json.Marshal(wire{Card: 2}); err == nil {
		t.Error("json.Marshal of an unregistered card succeeded")
	}
	if CardID(9).String() != "stable" || WonderPyramids.String() != "pyramids" || TokenLaw.String() != "law" {
		t.Error("String of registered ids is wrong")
	}
	var w WonderID
	if err := w.UnmarshalText([]byte(strings.Repeat("x", 500))); err == nil || len(err.Error()) > 200 {
		t.Errorf("long slug error = %v", err)
	}
}

func TestRegistryCopiesInput(t *testing.T) {
	cards := sampleCards()
	t.Cleanup(ReplaceForTest(cards, nil, nil))
	cards[1].Effects[0] = Points(99)
	cards[1].Slug = "changed"
	if d := Card(1); d.Slug != "lumber-yard" || d.Effects[0] != Produce(Wood, 1) {
		t.Errorf("the registry aliases its input: %+v", d)
	}
}

func TestReplaceForTestRestores(t *testing.T) {
	before := reg
	calls := 0
	savedHooks := replaceHooks
	t.Cleanup(func() { replaceHooks = savedHooks })
	afterReplace(func() { calls++ })

	restore := ReplaceForTest(sampleCards(), nil, nil)
	if Card(9) == nil || calls != 1 {
		t.Fatalf("replace: card = %v, hook calls = %d", Card(9), calls)
	}
	inner := ReplaceForTest(nil, sampleWonders(), nil)
	if Card(9) != nil || Wonder(WonderPyramids) == nil || calls != 2 {
		t.Fatal("nested replace did not take effect")
	}
	inner()
	if Card(9) == nil || Wonder(WonderPyramids) != nil || calls != 3 {
		t.Fatal("inner restore did not bring back the outer catalog")
	}
	restore()
	if reg != before || calls != 4 {
		t.Fatalf("restore: registry restored = %v, hook calls = %d", reg == before, calls)
	}
}

func TestRegisterFunctions(t *testing.T) {
	saved := reg
	t.Cleanup(func() { reg = saved })
	reg = new(registry)

	age1 := []CardDef{
		{ID: 1, Slug: "lumber-yard", Name: "Lumber Yard", Color: Brown, Effects: Fx(Produce(Wood, 1))},
		{ID: 9, Slug: "stable", Name: "Stable", Color: Red, Cost: MustCost("W"), ChainOut: ChainHorseshoe, Effects: Fx(Shields(1))},
	}
	guilds := []CardDef{
		{ID: 72, Slug: "moneylenders-guild", Name: "Moneylenders Guild", Color: Purple, Cost: MustCost("SS WW"),
			Effects: Fx(PointsPerCoins(ScopeMaxCity, 3))},
	}
	if !registerCards(1, age1) || !registerCards(3, guilds) ||
		!registerWonders(sampleWonders()) || !registerTokens(sampleTokens()) {
		t.Fatal("register functions returned false")
	}
	if d := Card(9); d == nil || d.Age != 1 || d.Back != BackAge1 {
		t.Errorf("stable = %+v", d)
	}
	if d := Card(72); d == nil || d.Age != 3 || d.Back != GuildBack {
		t.Errorf("moneylenders-guild = %+v", d)
	}
	if Wonder(WonderStatueOfZeus) == nil || Token(TokenUrbanism) == nil {
		t.Error("wonders or tokens missing")
	}
	mustPanic(t, "registerCards twice", func() { registerCards(1, age1) })
	mustPanic(t, "registerWonders twice", func() { registerWonders(sampleWonders()) })
	mustPanic(t, "registerTokens twice", func() { registerTokens(sampleTokens()) })
	mustPanic(t, "ReplaceForTest with a bad card", func() {
		ReplaceForTest([]CardDef{{ID: 1, Slug: "lumber-yard", Name: "Lumber Yard", Color: Brown}}, nil, nil) // Age unset
	})
	mustPanic(t, "ReplaceForTest with a bad wonder", func() {
		ReplaceForTest(nil, []WonderDef{{ID: WonderPyramids, Slug: "sphinx", Name: "Sphinx"}}, nil)
	})
	mustPanic(t, "ReplaceForTest with a bad token", func() {
		ReplaceForTest(nil, nil, []TokenDef{{ID: TokenLaw, Slug: "law"}}) // no name
	})
}

func TestRegistryRejects(t *testing.T) {
	good := func() CardDef {
		return CardDef{ID: 9, Slug: "stable", Name: "Stable", Color: Red, Cost: MustCost("W"),
			ChainOut: ChainHorseshoe, Effects: Fx(Shields(1))}
	}
	cardCases := []struct {
		name string
		age  uint8
		mod  func(*CardDef)
	}{
		{"id 0", 1, func(d *CardDef) { d.ID = 0 }},
		{"id 74", 1, func(d *CardDef) { d.ID = 74 }},
		{"id Unknown", 1, func(d *CardDef) { d.ID = Unknown }},
		{"empty slug", 1, func(d *CardDef) { d.Slug = "" }},
		{"the- slug", 1, func(d *CardDef) { d.Slug = "the-stable" }},
		{"upper-case slug", 1, func(d *CardDef) { d.Slug = "Stable" }},
		{"slug with a space", 1, func(d *CardDef) { d.Slug = "stable yard" }},
		{"slug of a wonder", 1, func(d *CardDef) { d.Slug = "pyramids" }},
		{"slug of a token", 1, func(d *CardDef) { d.Slug = "law" }},
		{"no name", 1, func(d *CardDef) { d.Name = "" }},
		{"age 0", 0, func(*CardDef) {}},
		{"age 4", 4, func(*CardDef) {}},
		{"Age disagrees", 1, func(d *CardDef) { d.Age = 2 }},
		{"no colour", 1, func(d *CardDef) { d.Color = NoColor }},
		{"bad colour", 1, func(d *CardDef) { d.Color = NumColors }},
		{"guild in age 1", 1, func(d *CardDef) { d.Color = Purple }},
		{"Back disagrees", 1, func(d *CardDef) { d.Back = BackAge2 }},
		{"guild with an age back", 3, func(d *CardDef) { d.Color = Purple; d.Back = BackAge3 }},
		{"bad chain in", 1, func(d *CardDef) { d.ChainIn = NumChains }},
		{"bad chain out", 1, func(d *CardDef) { d.ChainOut = NumChains }},
		{"self chain", 1, func(d *CardDef) { d.ChainIn = ChainHorseshoe }},
		{"bad effect", 1, func(d *CardDef) { d.Effects = Fx(Shields(0)) }},
	}
	for _, c := range cardCases {
		r := new(registry)
		if err := r.addWonder(sampleWonders()[1]); err != nil { // pyramids, to collide with
			t.Fatal(err)
		}
		if err := r.addToken(sampleTokens()[1]); err != nil { // law
			t.Fatal(err)
		}
		d := good()
		c.mod(&d)
		if err := r.addCard(c.age, d); err == nil {
			t.Errorf("%s: addCard accepted %+v", c.name, d)
		}
	}
	{
		r := new(registry)
		if err := r.addCard(1, good()); err != nil {
			t.Fatalf("good card rejected: %v", err)
		}
		dupID := good()
		dupID.Slug = "other"
		if err := r.addCard(1, dupID); err == nil {
			t.Error("duplicate card id accepted")
		}
		dupSlug := good()
		dupSlug.ID = 10
		if err := r.addCard(1, dupSlug); err == nil {
			t.Error("duplicate card slug accepted")
		}
		guild := good()
		guild.ID, guild.Slug, guild.Color, guild.ChainOut = 70, "magistrates-guild", Purple, NoChain
		guild.Back = GuildBack
		if err := r.addCard(3, guild); err != nil {
			t.Errorf("guild with an explicit guild back rejected: %v", err)
		}
		wonderOnCardSlug := WonderDef{ID: WonderSphinx, Slug: "sphinx", Name: "Sphinx"}
		if err := r.addWonder(wonderOnCardSlug); err != nil {
			t.Errorf("sphinx rejected: %v", err)
		}
	}

	wonderCases := []WonderDef{
		{ID: 0, Slug: "pyramids", Name: "Pyramids"},
		{ID: NumWonders + 1, Slug: "pyramids", Name: "Pyramids"},
		{ID: WonderPyramids, Slug: "the-pyramids", Name: "Pyramids"},
		{ID: WonderPyramids, Slug: "sphinx", Name: "Sphinx"}, // wrong slug for the id
		{ID: WonderPyramids, Slug: "pyramids"},               // no name
		{ID: WonderPyramids, Slug: "pyramids", Name: "Pyramids", Effects: Fx(Points(0))},
	}
	for _, d := range wonderCases {
		if err := new(registry).addWonder(d); err == nil {
			t.Errorf("addWonder accepted %+v", d)
		}
	}
	{
		r := new(registry)
		w := sampleWonders()[1]
		if err := r.addWonder(w); err != nil {
			t.Fatal(err)
		}
		if err := r.addWonder(w); err == nil {
			t.Error("duplicate wonder accepted")
		}
		if err := r.addCard(1, CardDef{ID: 1, Slug: "pyramids", Name: "X", Color: Brown}); err == nil {
			t.Error("card with a wonder's slug accepted")
		}
	}

	tokenCases := []TokenDef{
		{ID: 0, Slug: "law", Name: "Law"},
		{ID: NumTokens + 1, Slug: "law", Name: "Law"},
		{ID: TokenLaw, Slug: "economy", Name: "Economy"}, // wrong slug for the id
		{ID: TokenLaw, Slug: "law"},                      // no name
		{ID: TokenLaw, Slug: "law", Name: "Law", Effects: Fx(Science(NoSymbol))},
	}
	for _, d := range tokenCases {
		if err := new(registry).addToken(d); err == nil {
			t.Errorf("addToken accepted %+v", d)
		}
	}
	{
		r := new(registry)
		tok := sampleTokens()[1]
		if err := r.addToken(tok); err != nil {
			t.Fatal(err)
		}
		if err := r.addToken(tok); err == nil {
			t.Error("duplicate token accepted")
		}
	}
}

func mustPanic(t *testing.T, what string, f func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Errorf("%s did not panic", what)
		}
	}()
	f()
}
