package catalog

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// The data tests of the full catalog: counts, colours, chains, science,
// production, the mistakes other implementations made, the derived tables
// and the golden verification CSVs. Run with -update to rewrite the goldens
// after a deliberate data change, then review the diff.

var update = flag.Bool("update", false, "rewrite testdata/*.golden.csv from the catalog")

// card returns the registered card with this slug, failing the test if there
// is none.
func card(t *testing.T, slug string) *CardDef {
	t.Helper()
	id, ok := CardBySlug(slug)
	if !ok {
		t.Fatalf("no card %q", slug)
	}
	return Card(id)
}

func wonder(t *testing.T, slug string) *WonderDef {
	t.Helper()
	id, ok := WonderBySlug(slug)
	if !ok {
		t.Fatalf("no wonder %q", slug)
	}
	return Wonder(id)
}

func token(t *testing.T, slug string) *TokenDef {
	t.Helper()
	id, ok := TokenBySlug(slug)
	if !ok {
		t.Fatalf("no token %q", slug)
	}
	return Token(id)
}

// slugOf derives the wire slug from an English name: lower case, words joined
// by hyphens (a hyphen in the name, as in Glass-Blower, stays).
func slugOf(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), " ", "-")
}

func TestCounts(t *testing.T) {
	var perAge [4]int
	guilds := 0
	for d := range Cards() {
		if d.IsGuild() {
			guilds++
			continue
		}
		perAge[d.Age]++
	}
	if perAge != [4]int{0, 23, 23, 20} {
		t.Errorf("non-guild cards per age = %v, want [0 23 23 20]", perAge)
	}
	if guilds != NumGuilds {
		t.Errorf("guilds = %d, want %d", guilds, NumGuilds)
	}
	n := 0
	for range Wonders() {
		n++
	}
	if n != NumWonders {
		t.Errorf("wonders = %d, want %d", n, NumWonders)
	}
	n = 0
	for range Tokens() {
		n++
	}
	if n != NumTokens {
		t.Errorf("tokens = %d, want %d", n, NumTokens)
	}
}

// TestIDsContiguous checks that every id from 1 to the count is registered,
// and that the building ids run by age: 1-23 Age I, 24-46 Age II, 47-66
// Age III and 67-73 the guilds, as in the reference numbering.
func TestIDsContiguous(t *testing.T) {
	for id := CardID(1); id <= NumCards; id++ {
		d := Card(id)
		if d == nil {
			t.Errorf("card %d is not registered", id)
			continue
		}
		var wantAge uint8
		switch {
		case id <= 23:
			wantAge = 1
		case id <= 46:
			wantAge = 2
		default:
			wantAge = 3
		}
		if d.Age != wantAge {
			t.Errorf("card %d (%s) is in age %d, want %d", id, d.Slug, d.Age, wantAge)
		}
		if d.IsGuild() != (id >= 67) {
			t.Errorf("card %d (%s): guild = %v", id, d.Slug, d.IsGuild())
		}
	}
	for id := WonderID(1); id <= NumWonders; id++ {
		if Wonder(id) == nil {
			t.Errorf("wonder %d is not registered", id)
		}
	}
	for id := TokenID(1); id <= NumTokens; id++ {
		if Token(id) == nil {
			t.Errorf("token %d is not registered", id)
		}
	}
	if Card(0) != nil || Card(NumCards+1) != nil || Wonder(NumWonders+1) != nil || Token(NumTokens+1) != nil {
		t.Error("an id outside the range is registered")
	}
}

// TestNamesAndSlugs checks that slugs and names are unique across all items
// and that every slug is derived from its name.
func TestNamesAndSlugs(t *testing.T) {
	slugs := map[string]string{}
	names := map[string]string{}
	add := func(kind, slug, name string, alt []string) {
		if prev, ok := slugs[slug]; ok {
			t.Errorf("slug %q used by %s and %s", slug, prev, kind)
		}
		slugs[slug] = kind
		if prev, ok := names[name]; ok {
			t.Errorf("name %q used by %s and %s", name, prev, kind)
		}
		names[name] = kind
		if got := slugOf(name); got != slug {
			t.Errorf("%s %q: slug %q, want %q from its name", kind, name, slug, got)
		}
		if strings.HasPrefix(name, "The ") {
			t.Errorf("%s name %q starts with \"The\"", kind, name)
		}
		for _, a := range alt {
			if a == "" || a == name {
				t.Errorf("%s %q: bad alternative name %q", kind, name, a)
			}
		}
	}
	for d := range Cards() {
		add("card", d.Slug, d.Name, d.AltNames)
	}
	for d := range Wonders() {
		add("wonder", d.Slug, d.Name, nil)
	}
	for d := range Tokens() {
		add("token", d.Slug, d.Name, nil)
	}
	if len(slugs) != NumCards+NumWonders+NumTokens {
		t.Errorf("%d slugs, want %d", len(slugs), NumCards+NumWonders+NumTokens)
	}
	// Alternative names that other sources print.
	for slug, want := range map[string][]string{
		"courthouse":      {"Tribunal"},
		"pretorium":       {"Courthouse"},
		"merchants-guild": {"Traders Guild"},
	} {
		if got := card(t, slug).AltNames; !slices.Equal(got, want) {
			t.Errorf("%s AltNames = %q, want %q", slug, got, want)
		}
	}
}

func TestColorsPerAge(t *testing.T) {
	// Counts per colour, in the order brown, grey, red, green, yellow, blue.
	want := [4][6]int{
		1: {6, 2, 4, 4, 4, 3},
		2: {3, 2, 5, 4, 4, 5},
		3: {0, 0, 5, 4, 5, 6},
	}
	var got [4][6]int
	for d := range Cards() {
		if d.IsGuild() {
			if d.Age != 3 {
				t.Errorf("guild %s is in age %d", d.Slug, d.Age)
			}
			continue
		}
		got[d.Age][d.Color-Brown]++
	}
	if got != want {
		t.Errorf("colours per age (brown grey red green yellow blue) = %v, want %v", got[1:], want[1:])
	}
	if T.ByColor[Purple].Count() != NumGuilds || T.ByColor[Purple] != T.Guilds {
		t.Error("the purple cards are not exactly the guilds")
	}
}

func TestBacks(t *testing.T) {
	for d := range Cards() {
		want := AgeBack(d.Age)
		if d.IsGuild() {
			want = GuildBack
		}
		if d.Back != want {
			t.Errorf("%s: back %s, want %s", d.Slug, d.Back, want)
		}
	}
}

// chainPairs is every chain of the base game: the symbol, the card showing
// it and the later card it makes free.
var chainPairs = []struct {
	chain    Chain
	from, to string
}{
	{ChainHorseshoe, "stable", "horse-breeders"},
	{ChainSword, "garrison", "barracks"},
	{ChainTower, "palisade", "fortifications"},
	{ChainBook, "scriptorium", "library"},
	{ChainGear, "pharmacist", "dispensary"},
	{ChainAmphora, "tavern", "lighthouse"},
	{ChainMask, "theater", "statue"},
	{ChainMoon, "altar", "temple"},
	{ChainDrop, "baths", "aqueduct"},
	{ChainTarget, "archery-range", "siege-workshop"},
	{ChainHelmet, "parade-ground", "circus"},
	{ChainLyre, "school", "university"},
	{ChainLamp, "laboratory", "observatory"},
	{ChainBarrel, "brewery", "arena"},
	{ChainSun, "temple", "pantheon"},
	{ChainColumn, "statue", "gardens"},
	{ChainTempleFacade, "rostrum", "senate"},
}

func TestChains(t *testing.T) {
	if len(chainPairs) != int(NumChains)-1 {
		t.Fatalf("the test lists %d chains, the enum has %d", len(chainPairs), NumChains-1)
	}
	var outs, ins [NumChains]int
	for d := range Cards() {
		outs[d.ChainOut]++
		ins[d.ChainIn]++
	}
	for c := ChainHorseshoe; c < NumChains; c++ {
		if outs[c] != 1 || ins[c] != 1 {
			t.Errorf("chain %s is shown by %d cards and frees %d, want 1 and 1", c, outs[c], ins[c])
		}
	}
	nOut, nIn := 0, 0
	for d := range Cards() {
		if d.ChainOut != NoChain {
			nOut++
		}
		if d.ChainIn != NoChain {
			nIn++
		}
	}
	if nOut != 17 || nIn != 17 {
		t.Errorf("%d cards show a chain and %d are freed by one, want 17 and 17", nOut, nIn)
	}
	for _, p := range chainPairs {
		from, to := card(t, p.from), card(t, p.to)
		if from.ChainOut != p.chain || to.ChainIn != p.chain {
			t.Errorf("%s -> %s: ChainOut %s, ChainIn %s, want %s", p.from, p.to, from.ChainOut, to.ChainIn, p.chain)
		}
		if to.Age <= from.Age {
			t.Errorf("%s (age %d) frees %s (age %d), which is not in a later age", p.from, from.Age, p.to, to.Age)
		}
		if T.ChainSource[p.chain] != from.ID || T.ChainTarget[p.chain] != to.ID {
			t.Errorf("chain %s: tables give %s -> %s", p.chain, T.ChainSource[p.chain], T.ChainTarget[p.chain])
		}
		if from.IsGuild() || to.IsGuild() || from.Color == Brown || from.Color == Grey {
			t.Errorf("chain %s involves a guild or a resource card", p.chain)
		}
	}
	if T.ChainSource[NoChain] != 0 || T.ChainTarget[NoChain] != 0 {
		t.Error("NoChain has a source or a target")
	}
}

func TestScience(t *testing.T) {
	want := map[Symbol][]string{
		Globe:     {"university", "observatory"},
		Sundial:   {"academy", "study"},
		Mortar:    {"pharmacist", "dispensary"},
		SetSquare: {"workshop", "laboratory"},
		Quill:     {"scriptorium", "library"},
		Wheel:     {"apothecary", "school"},
	}
	got := map[Symbol][]string{}
	for d := range Cards() {
		var syms []Symbol
		for _, e := range d.Effects {
			if e.Kind == EffScience {
				syms = append(syms, e.Symbol)
			}
		}
		if d.Color == Green {
			if len(syms) != 1 {
				t.Errorf("green card %s has %d science symbols, want 1", d.Slug, len(syms))
				continue
			}
			got[syms[0]] = append(got[syms[0]], d.Slug)
		} else if len(syms) != 0 {
			t.Errorf("%s card %s has a science symbol", d.Color, d.Slug)
		}
		if (T.Sym[d.ID] != NoSymbol) != (d.Color == Green) {
			t.Errorf("%s: table symbol %s", d.Slug, T.Sym[d.ID])
		}
	}
	for s, cards := range want {
		slices.Sort(cards)
		g := got[s]
		slices.Sort(g)
		if !slices.Equal(g, cards) {
			t.Errorf("symbol %s on %v, want %v", s, g, cards)
		}
	}
	if _, ok := got[Scales]; ok {
		t.Error("a building shows the scales")
	}
	// Scales is only on Law, and Law has nothing else.
	for d := range Tokens() {
		e, ok := FindEffect(d.Effects, EffScience)
		if ok != (d.ID == TokenLaw) {
			t.Errorf("token %s: science = %v", d.Slug, ok)
		}
		if ok && (e.Symbol != Scales || len(d.Effects) != 1) {
			t.Errorf("Law effects = %v", d.Effects)
		}
	}
	for d := range Wonders() {
		if _, ok := FindEffect(d.Effects, EffScience); ok {
			t.Errorf("wonder %s has a science symbol", d.Slug)
		}
	}
}

func TestProduction(t *testing.T) {
	for d := range Cards() {
		e, produces := FindEffect(d.Effects, EffProduce)
		switch d.Color {
		case Brown, Grey:
			if d.Age == 3 {
				t.Errorf("%s card %s is in Age III", d.Color, d.Slug)
			}
			if !produces || len(d.Effects) != 1 {
				t.Errorf("%s: effects %v, want a single production", d.Slug, d.Effects)
				continue
			}
			class, units := RawGoods, d.Age // Age I brown makes 1 unit, Age II brown makes 2
			if d.Color == Grey {
				class, units = Manufactured, 1
			}
			if !class.Has(e.Res) || e.N != units {
				t.Errorf("%s produces %d %s, want %d units of its class", d.Slug, e.N, e.Res, units)
			}
			if T.Prod[d.ID][e.Res] != e.N {
				t.Errorf("%s: table production %v", d.Slug, T.Prod[d.ID])
			}
		default:
			if produces || T.Prod[d.ID] != [NumResources]uint8{} {
				t.Errorf("%s card %s has fixed production", d.Color, d.Slug)
			}
		}
	}
	// Each resource has exactly the expected producers.
	wantProducers := [NumResources][]string{
		Wood:    {"lumber-yard", "logging-camp", "sawmill"},
		Clay:    {"clay-pool", "clay-pit", "brickyard"},
		Stone:   {"stone-pit", "quarry", "shelf-quarry"},
		Glass:   {"glassworks", "glass-blower"},
		Papyrus: {"press", "drying-room"},
	}
	for r := range NumResources {
		var got []string
		for d := range Cards() {
			if T.Prod[d.ID][r] > 0 {
				got = append(got, d.Slug)
			}
		}
		if !slices.Equal(got, wantProducers[r]) {
			t.Errorf("%s producers = %v, want %v", r, got, wantProducers[r])
		}
	}
	for d := range Wonders() {
		if _, ok := FindEffect(d.Effects, EffProduce); ok {
			t.Errorf("wonder %s has fixed production", d.Slug)
		}
	}
}

func TestChoiceAndTradeFix(t *testing.T) {
	choice := map[string]ResMask{}
	for d := range Cards() {
		if e, ok := FindEffect(d.Effects, EffProduceChoice); ok {
			choice["card "+d.Slug] = e.Mask
		}
	}
	for d := range Wonders() {
		if e, ok := FindEffect(d.Effects, EffProduceChoice); ok {
			choice["wonder "+d.Slug] = e.Mask
		}
	}
	want := map[string]ResMask{
		"card forum":              Manufactured,
		"card caravansery":        RawGoods,
		"wonder great-lighthouse": RawGoods,
		"wonder piraeus":          Manufactured,
	}
	if len(choice) != len(want) {
		t.Errorf("flexible producers = %v, want %v", choice, want)
	}
	for k, m := range want {
		if choice[k] != m {
			t.Errorf("%s: choice mask %05b, want %05b", k, choice[k], m)
		}
	}
	// The fast payment method needs every mask to be exactly one class.
	for k, m := range choice {
		if m != RawGoods && m != Manufactured {
			t.Errorf("%s: mask %05b is neither raw goods nor manufactured goods", k, m)
		}
	}
	if T.Choice[card(t, "forum").ID] != Manufactured || T.WChoice[WonderGreatLighthouse] != RawGoods {
		t.Error("the choice tables are wrong")
	}

	fix := map[string]ResMask{}
	for d := range Cards() {
		if e, ok := FindEffect(d.Effects, EffTradeFix); ok {
			fix[d.Slug] = e.Mask
			if d.Color != Yellow {
				t.Errorf("%s card %s fixes a price", d.Color, d.Slug)
			}
		}
		if T.TradeFix[d.ID] != fix[d.Slug] {
			t.Errorf("%s: table trade fix %05b", d.Slug, T.TradeFix[d.ID])
		}
	}
	wantFix := map[string]ResMask{
		"stone-reserve": Stone.Mask(),
		"clay-reserve":  Clay.Mask(),
		"wood-reserve":  Wood.Mask(),
		"customs-house": Manufactured,
	}
	if len(fix) != len(wantFix) {
		t.Errorf("trade fixes = %v, want %v", fix, wantFix)
	}
	for k, m := range wantFix {
		if fix[k] != m {
			t.Errorf("%s fixes %05b, want %05b", k, fix[k], m)
		}
	}
}

// TestEffectShapes checks that each colour carries only the kinds of effect
// that colour can have, so a misplaced effect stands out.
func TestEffectShapes(t *testing.T) {
	kinds := func(fx []Effect) []EffectKind {
		var ks []EffectKind
		for _, e := range fx {
			ks = append(ks, e.Kind)
		}
		return ks
	}
	for d := range Cards() {
		ks := kinds(d.Effects)
		ok := true
		switch d.Color {
		case Red:
			ok = slices.Equal(ks, []EffectKind{EffShields})
		case Green:
			ok = slices.Equal(ks, []EffectKind{EffScience}) || slices.Equal(ks, []EffectKind{EffScience, EffPoints})
		case Blue:
			ok = slices.Equal(ks, []EffectKind{EffPoints})
		case Yellow:
			switch d.Age {
			case 1, 2:
				ok = len(ks) == 1 && (ks[0] == EffCoins || ks[0] == EffTradeFix || ks[0] == EffProduceChoice)
			case 3:
				ok = len(ks) == 2 && d.Effects[0] == Points(3) &&
					(ks[1] == EffCoinsPerCard || ks[1] == EffCoinsPerWonder) && d.Effects[1].Scope == ScopeSelf
			}
		case Purple:
			for _, e := range d.Effects {
				if e.Scope != ScopeMaxCity {
					ok = false
				}
			}
			switch {
			case len(ks) == 2 && ks[0] == EffCoinsPerCard && ks[1] == EffPointsPerCard:
				// Coins and points count the same colours.
				ok = ok && d.Effects[0].Colors == d.Effects[1].Colors && d.Effects[0].N == 1 && d.Effects[1].N == 1
			case slices.Equal(ks, []EffectKind{EffPointsPerWonder}), slices.Equal(ks, []EffectKind{EffPointsPerCoins}):
			default:
				ok = false
			}
		}
		if !ok {
			t.Errorf("%s %s card %s has unexpected effects %v", ageName(d.Age), d.Color, d.Slug, d.Effects)
		}
	}
	for d := range Wonders() {
		for _, e := range d.Effects {
			switch e.Kind {
			case EffPoints, EffShields, EffCoins, EffProduceChoice, EffPlayAgain, EffDestroy,
				EffBuildFromDiscard, EffDrawTokens, EffOpponentLosesCoins:
			default:
				t.Errorf("wonder %s has a %s effect", d.Slug, e.Kind)
			}
		}
	}
	for d := range Cards() {
		seen := map[EffectKind]bool{}
		for _, e := range d.Effects {
			if seen[e.Kind] {
				t.Errorf("%s has two %s effects", d.Slug, e.Kind)
			}
			seen[e.Kind] = true
		}
	}
}

func ageName(age uint8) string { return [...]string{"?", "Age I", "Age II", "Age III"}[age] }

// TestKnownPitfalls pins the values that other implementations got wrong
// (docs/rules/sources.md, "Known mistakes in other implementations").
func TestKnownPitfalls(t *testing.T) {
	costs := map[string]string{
		"walls":             "SS",
		"palisade":          "2c",
		"forum":             "3c + C",
		"parade-ground":     "CC G",
		"senate":            "CC S P",
		"builders-guild":    "SS C W G",
		"tacticians-guild":  "SS C P",
		"caravansery":       "2c + G P",
		"customs-house":     "4c",
		"magistrates-guild": "WW C P",
	}
	for slug, want := range costs {
		if got := card(t, slug).Cost; got != MustCost(want) {
			t.Errorf("%s costs %s, want %s", slug, got, want)
		}
	}
	if d := card(t, "senate"); !slices.Equal(d.Effects, Fx(Points(5))) {
		t.Errorf("senate effects = %v, want 5 VP", d.Effects)
	}
	if g := card(t, "builders-guild").Cost.Res[Glass]; g != 1 {
		t.Errorf("builders-guild needs %d glass, want 1", g)
	}
	if c := card(t, "tacticians-guild").Cost; c.Res[Wood] != 0 || c.Res[Clay] != 1 {
		t.Errorf("tacticians-guild cost %s has wood or lacks clay", c)
	}
	for _, slug := range []string{"statue", "gardens"} {
		if c := card(t, slug).Color; c != Blue {
			t.Errorf("%s is %s, want blue", slug, c)
		}
	}
	for _, slug := range []string{"workshop", "apothecary", "dispensary", "forum", "caravansery", "customs-house", "study"} {
		if d := card(t, slug); d.ChainOut != NoChain {
			t.Errorf("%s shows the %s chain symbol, want none", slug, d.ChainOut)
		}
	}
	for _, slug := range []string{"study", "workshop", "apothecary"} {
		if d := card(t, slug); d.ChainIn != NoChain || d.ChainOut != NoChain {
			t.Errorf("%s has a chain (%s in, %s out), want none", slug, d.ChainIn, d.ChainOut)
		}
	}
	for _, p := range []struct{ from, to string }{
		{"school", "university"}, {"brewery", "arena"}, {"tavern", "lighthouse"},
	} {
		if T.ChainTarget[card(t, p.from).ChainOut] != card(t, p.to).ID {
			t.Errorf("%s does not chain to %s", p.from, p.to)
		}
	}
	if card(t, "tavern").ChainOut == NoChain || card(t, "brewery").ChainOut == NoChain {
		t.Error("tavern and brewery must show a chain symbol")
	}
}

// TestSpotValues checks values that are easy to get wrong in other ways:
// the coin and point amounts of specific cards, wonders and tokens.
func TestSpotValues(t *testing.T) {
	if !slices.Equal(card(t, "lighthouse").Effects, Fx(Points(3), CoinsPerCard(ScopeSelf, 1, Yellow))) {
		t.Error("lighthouse effects are wrong")
	}
	if !slices.Equal(card(t, "arena").Effects, Fx(Points(3), CoinsPerWonder(ScopeSelf, 2))) {
		t.Error("arena effects are wrong")
	}
	if !slices.Equal(card(t, "chamber-of-commerce").Effects, Fx(Points(3), CoinsPerCard(ScopeSelf, 3, Grey))) {
		t.Error("chamber-of-commerce effects are wrong")
	}
	if !slices.Equal(card(t, "port").Effects, Fx(Points(3), CoinsPerCard(ScopeSelf, 2, Brown))) {
		t.Error("port effects are wrong")
	}
	if !slices.Equal(card(t, "shipowners-guild").Effects,
		Fx(CoinsPerCard(ScopeMaxCity, 1, Brown, Grey), PointsPerCard(ScopeMaxCity, 1, Brown, Grey))) {
		t.Error("shipowners-guild must count brown and grey together")
	}
	if !slices.Equal(card(t, "builders-guild").Effects, Fx(PointsPerWonder(ScopeMaxCity, 2))) {
		t.Error("builders-guild effects are wrong")
	}
	if !slices.Equal(card(t, "moneylenders-guild").Effects, Fx(PointsPerCoins(ScopeMaxCity, 3))) {
		t.Error("moneylenders-guild effects are wrong")
	}
	if card(t, "tavern").Effects[0] != Coins(4) || card(t, "brewery").Effects[0] != Coins(6) {
		t.Error("tavern or brewery coins are wrong")
	}
	if card(t, "pretorium").Cost != MustCost("8c") || card(t, "pretorium").Effects[0] != Shields(3) {
		t.Error("pretorium is wrong")
	}

	wonders := []struct {
		slug, cost string
		fx         []Effect
	}{
		{"pyramids", "SSS P", Fx(Points(9))},
		{"colossus", "CCC G", Fx(Shields(2), Points(3))},
		{"great-lighthouse", "PP S W", Fx(ProduceChoice(Wood, Clay, Stone), Points(4))},
		{"temple-of-artemis", "W S G P", Fx(Coins(12), PlayAgain())},
		{"mausoleum", "CC GG P", Fx(BuildFromDiscard(), Points(2))},
		{"great-library", "WWW G P", Fx(DrawTokens(3), Points(4))},
		{"piraeus", "WW C S", Fx(ProduceChoice(Glass, Papyrus), PlayAgain(), Points(2))},
		{"hanging-gardens", "WW G P", Fx(Coins(6), PlayAgain(), Points(3))},
		{"statue-of-zeus", "PP C W S", Fx(Destroy(Brown), Shields(1), Points(3))},
		{"sphinx", "S C GG", Fx(PlayAgain(), Points(6))},
		{"appian-way", "SS CC P", Fx(Coins(3), OpponentLosesCoins(3), PlayAgain(), Points(3))},
		{"circus-maximus", "SS W G", Fx(Destroy(Grey), Shields(1), Points(3))},
	}
	for i, w := range wonders {
		d := wonder(t, w.slug)
		if int(d.ID) != i+1 {
			t.Errorf("wonder %s has id %d, want %d", w.slug, d.ID, i+1)
		}
		if d.Cost != MustCost(w.cost) {
			t.Errorf("wonder %s costs %s, want %s", w.slug, d.Cost, w.cost)
		}
		if !slices.Equal(d.Effects, w.fx) {
			t.Errorf("wonder %s effects = %v, want %v", w.slug, d.Effects, w.fx)
		}
	}

	tokens := []struct {
		slug string
		fx   []Effect
	}{
		{"agriculture", Fx(Coins(6), Points(4))},
		{"architecture", Fx(Discount(ForWonders, 2))},
		{"economy", Fx(TradeIncome())},
		{"law", Fx(Science(Scales))},
		{"masonry", Fx(Discount(ForBlue, 2))},
		{"mathematics", Fx(PointsPerToken(3))},
		{"philosophy", Fx(Points(7))},
		{"strategy", Fx(ExtraShield(1))},
		{"theology", Fx(WondersPlayAgain())},
		{"urbanism", Fx(Coins(6), ChainBonus(4))},
	}
	for i, k := range tokens {
		d := token(t, k.slug)
		if int(d.ID) != i+1 {
			t.Errorf("token %s has id %d, want %d", k.slug, d.ID, i+1)
		}
		if !slices.Equal(d.Effects, k.fx) {
			t.Errorf("token %s effects = %v, want %v", k.slug, d.Effects, k.fx)
		}
	}
}

// TestTables recomputes every derived table from the definitions.
func TestTables(t *testing.T) {
	var all, guilds CardSet
	var ages [4]CardSet
	var byColor [NumColors]CardSet
	for d := range Cards() {
		all.Add(d.ID)
		byColor[d.Color].Add(d.ID)
		if d.IsGuild() {
			guilds.Add(d.ID)
		} else {
			ages[d.Age].Add(d.ID)
		}
		id := d.ID
		if T.Color[id] != d.Color || T.Age[id] != d.Age || T.Back[id] != d.Back || T.Cost[id] != d.Cost ||
			T.ChainIn[id] != d.ChainIn || T.ChainOut[id] != d.ChainOut {
			t.Errorf("%s: basic tables disagree with the definition", d.Slug)
		}
		var prod [NumResources]uint8
		var choice, fix ResMask
		var shields, vp, coins uint8
		sym := NoSymbol
		for _, e := range d.Effects {
			switch e.Kind {
			case EffProduce:
				prod[e.Res] += e.N
			case EffProduceChoice:
				choice |= e.Mask
			case EffTradeFix:
				fix |= e.Mask
			case EffShields:
				shields += e.N
			case EffScience:
				sym = e.Symbol
			case EffPoints:
				vp += e.N
			case EffCoins:
				coins += e.N
			}
		}
		if T.Prod[id] != prod || T.Choice[id] != choice || T.TradeFix[id] != fix || T.Shields[id] != shields ||
			T.Sym[id] != sym || T.VP[id] != vp || T.Coins[id] != coins {
			t.Errorf("%s: effect tables disagree with the definition", d.Slug)
		}
	}
	if T.All != all || T.Guilds != guilds || T.AgeCards != ages || T.ByColor != byColor {
		t.Error("card sets disagree with the definitions")
	}
	if T.All.Count() != NumCards || T.AgeCards[1].Count() != 23 || T.AgeCards[2].Count() != 23 ||
		T.AgeCards[3].Count() != 20 || T.Guilds.Count() != NumGuilds || !T.AgeCards[0].Empty() {
		t.Error("card set sizes are wrong")
	}
	if !T.AgeCards[3].And(T.ByColor[Brown].Or(T.ByColor[Grey])).Empty() {
		t.Error("Age III has a brown or grey card")
	}
	for d := range Wonders() {
		id := d.ID
		var w struct {
			choice             ResMask
			shields, coins, vp uint8
			again, rebuild     bool
			destroy            Color
			draw, oppLoses     uint8
		}
		for _, e := range d.Effects {
			switch e.Kind {
			case EffProduceChoice:
				w.choice = e.Mask
			case EffShields:
				w.shields = e.N
			case EffCoins:
				w.coins = e.N
			case EffPoints:
				w.vp = e.N
			case EffPlayAgain:
				w.again = true
			case EffBuildFromDiscard:
				w.rebuild = true
			case EffDestroy:
				for c := Brown; c < NumColors; c++ {
					if e.Colors.Has(c) {
						w.destroy = c
					}
				}
			case EffDrawTokens:
				w.draw = e.N
			case EffOpponentLosesCoins:
				w.oppLoses = e.N
			}
		}
		if T.WCost[id] != d.Cost || T.WChoice[id] != w.choice || T.WShields[id] != w.shields ||
			T.WCoins[id] != w.coins || T.WVP[id] != w.vp || T.WPlayAgain[id] != w.again ||
			T.WRebuild[id] != w.rebuild || T.WDestroy[id] != w.destroy || T.WDraw[id] != w.draw ||
			T.WOppLoses[id] != w.oppLoses {
			t.Errorf("wonder %s: tables disagree with the definition", d.Slug)
		}
	}
	if T.WDestroy[WonderStatueOfZeus] != Brown || T.WDestroy[WonderCircusMaximus] != Grey || !T.WRebuild[WonderMausoleum] ||
		T.WDraw[WonderGreatLibrary] != 3 || T.WOppLoses[WonderAppianWay] != 3 {
		t.Error("wonder special tables are wrong")
	}
	for d := range Tokens() {
		var coins, vp uint8
		if e, ok := FindEffect(d.Effects, EffCoins); ok {
			coins = e.N
		}
		if e, ok := FindEffect(d.Effects, EffPoints); ok {
			vp = e.N
		}
		if T.TokenCoins[d.ID] != coins || T.TokenVP[d.ID] != vp {
			t.Errorf("token %s: tables disagree with the definition", d.Slug)
		}
	}
}

// TestTablesFollowReplace checks that ReplaceForTest rebuilds the tables and
// that restoring brings the full tables back.
func TestTablesFollowReplace(t *testing.T) {
	full := T
	restore := ReplaceForTest([]CardDef{
		{ID: 1, Slug: "lumber-yard", Name: "Lumber Yard", Age: 1, Color: Brown, Effects: Fx(Produce(Wood, 1))},
	}, nil, nil)
	if T.All != CardSetOf(1) || T.Prod[1][Wood] != 1 || T.VP[61] != 0 {
		t.Errorf("tables were not rebuilt for the replacement catalog")
	}
	restore()
	if T != full {
		t.Error("restoring the catalog did not restore the tables")
	}
}

// TestBuildTablesRejects checks the structural assertions made when the
// tables are built.
func TestBuildTablesRejects(t *testing.T) {
	brown := func(id CardID, slug string) CardDef {
		return CardDef{ID: id, Slug: slug, Name: slug, Color: Brown, Effects: Fx(Produce(Wood, 1))}
	}
	yellow := func(id CardID, slug string, fx ...Effect) CardDef {
		return CardDef{ID: id, Slug: slug, Name: slug, Color: Yellow, Effects: fx}
	}
	cases := []struct {
		name    string
		cards   []CardDef
		wonders []WonderDef
		tokens  []TokenDef
	}{
		{"mixed choice mask", []CardDef{yellow(39, "forum", ProduceChoice(Wood, Glass))}, nil, nil},
		{"partial choice mask", []CardDef{yellow(39, "forum", ProduceChoice(Wood, Clay))}, nil, nil},
		{"wonder with a mixed choice mask", nil, []WonderDef{{ID: WonderPiraeus, Slug: "piraeus", Name: "Piraeus",
			Effects: Fx(ProduceChoice(Glass, Stone))}}, nil},
		{"token with a choice", nil, nil, []TokenDef{{ID: TokenLaw, Slug: "law", Name: "Law",
			Effects: Fx(ProduceChoice(Glass, Papyrus))}}},
		{"two effects of one kind", []CardDef{yellow(17, "tavern", Coins(4), Coins(1))}, nil, nil},
		{"production on a yellow card", []CardDef{yellow(17, "tavern", Produce(Wood, 1))}, nil, nil},
		{"brown card making glass", []CardDef{{ID: 1, Slug: "a", Name: "a", Color: Brown, Effects: Fx(Produce(Glass, 1))}}, nil, nil},
		{"grey card making wood", []CardDef{{ID: 7, Slug: "a", Name: "a", Color: Grey, Effects: Fx(Produce(Wood, 1))}}, nil, nil},
		{"wonder with production", nil, []WonderDef{{ID: WonderPyramids, Slug: "pyramids", Name: "Pyramids",
			Effects: Fx(Produce(Stone, 1))}}, nil},
		{"chain shown twice", []CardDef{
			{ID: 9, Slug: "a", Name: "a", Color: Red, ChainOut: ChainHorseshoe, Effects: Fx(Shields(1))},
			{ID: 10, Slug: "b", Name: "b", Color: Red, ChainOut: ChainHorseshoe, Effects: Fx(Shields(1))},
		}, nil, nil},
		{"chain freeing twice", []CardDef{
			{ID: 29, Slug: "a", Name: "a", Color: Red, ChainIn: ChainSword, Effects: Fx(Shields(1))},
			{ID: 30, Slug: "b", Name: "b", Color: Red, ChainIn: ChainSword, Effects: Fx(Shields(1))},
		}, nil, nil},
		{"five flexible producers", []CardDef{
			yellow(39, "a", ProduceChoice(Glass, Papyrus)), yellow(40, "b", ProduceChoice(Wood, Clay, Stone)),
			yellow(41, "c", ProduceChoice(Glass, Papyrus)), yellow(42, "d", ProduceChoice(Glass, Papyrus)),
		}, []WonderDef{{ID: WonderPiraeus, Slug: "piraeus", Name: "Piraeus", Effects: Fx(ProduceChoice(Glass, Papyrus))}}, nil},
	}
	for _, c := range cases {
		r := new(registry)
		for _, d := range c.cards {
			if err := r.addCard(1, d); err != nil {
				t.Fatalf("%s: %v", c.name, err)
			}
		}
		for _, d := range c.wonders {
			if err := r.addWonder(d); err != nil {
				t.Fatalf("%s: %v", c.name, err)
			}
		}
		for _, d := range c.tokens {
			if err := r.addToken(d); err != nil {
				t.Fatalf("%s: %v", c.name, err)
			}
		}
		if _, err := buildTables(r); err == nil {
			t.Errorf("%s: buildTables accepted the catalog", c.name)
		}
	}
	// The same shapes are fine when well formed.
	r := new(registry)
	for _, d := range []CardDef{brown(1, "a"), yellow(39, "forum", ProduceChoice(Glass, Papyrus))} {
		if err := r.addCard(1, d); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := buildTables(r); err != nil {
		t.Errorf("a well-formed catalog was rejected: %v", err)
	}
	// The same assertion guards ReplaceForTest. Its panic leaves the bad
	// registry swapped in, so put the real catalog and tables back by hand.
	saved, savedTables := reg, T
	mustPanic(t, "ReplaceForTest with a mixed choice mask", func() {
		ReplaceForTest([]CardDef{{ID: 39, Slug: "forum", Name: "Forum", Age: 2, Color: Yellow,
			Effects: Fx(ProduceChoice(Wood, Glass))}}, nil, nil)
	})
	reg = saved
	loadTables()
	if T != savedTables {
		t.Fatal("the tables were not restored")
	}
}

// TestRecordTables checks the verification tables' shape: the exact schema
// headers, one row per item in id order, and errors for effects the schema
// cannot express.
func TestRecordTables(t *testing.T) {
	tables, err := RecordTables()
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		name, header string
		rows         int
	}{
		{"cards", "id,name,age,back,color,coins,W,C,S,G,P,produce,produce_choice,trade_fix,shields,science,vp,coins_gain,coins_per,points_per,chain_in,chain_out", NumCards},
		{"wonders", "id,name,coins,W,C,S,G,P,vp,shields,coins_gain,produce_choice,play_again,destroy_color,build_from_discard,draw_tokens,opponent_loses", NumWonders},
		{"tokens", "id,name,coins_gain,vp,vp_per_token,discount,trade_income,science,extra_shield,wonders_play_again,chain_bonus", NumTokens},
	}
	if len(tables) != len(want) {
		t.Fatalf("%d tables, want %d", len(tables), len(want))
	}
	for i, w := range want {
		tb := tables[i]
		if tb.Name != w.name || strings.Join(tb.Header, ",") != w.header || len(tb.Rows) != w.rows {
			t.Errorf("table %d: %s with header %q and %d rows; want %s, %q, %d", i, tb.Name, strings.Join(tb.Header, ","),
				len(tb.Rows), w.name, w.header, w.rows)
			continue
		}
		for j, row := range tb.Rows {
			if len(row) != len(tb.Header) || row[0] != strconv.Itoa(j+1) {
				t.Errorf("%s row %d = %q", tb.Name, j, row)
			}
		}
	}
	// A few cells, spelled out.
	cells := []struct {
		table, item, col, want string
	}{
		{"cards", "Caravansery", "produce_choice", "W|C|S"},
		{"cards", "Caravansery", "coins", "2"},
		{"cards", "Caravansery", "G", "1"},
		{"cards", "Customs House", "trade_fix", "G|P"},
		{"cards", "Sawmill", "produce", "W2"},
		{"cards", "Lighthouse", "coins_per", "yellow:1:self"},
		{"cards", "Lighthouse", "chain_in", "amphora"},
		{"cards", "Arena", "coins_per", "wonder:2:self"},
		{"cards", "Shipowners Guild", "coins_per", "brown+grey:1:max"},
		{"cards", "Shipowners Guild", "points_per", "brown+grey:1:max"},
		{"cards", "Shipowners Guild", "back", "guild"},
		{"cards", "Builders Guild", "points_per", "wonder:2:max"},
		{"cards", "Builders Guild", "coins_per", ""},
		{"cards", "Moneylenders Guild", "points_per", "coins3:1:max"},
		{"cards", "Rostrum", "chain_out", "temple-facade"},
		{"cards", "Workshop", "science", "set-square"},
		{"cards", "Lumber Yard", "coins", ""},
		{"wonders", "Great Lighthouse", "produce_choice", "W|C|S"},
		{"wonders", "Temple of Artemis", "vp", ""},
		{"wonders", "Temple of Artemis", "coins_gain", "12"},
		{"wonders", "Appian Way", "opponent_loses", "3"},
		{"wonders", "Circus Maximus", "destroy_color", "grey"},
		{"wonders", "Mausoleum", "build_from_discard", "1"},
		{"wonders", "Great Library", "draw_tokens", "3"},
		{"wonders", "Sphinx", "play_again", "1"},
		{"tokens", "Masonry", "discount", "blue"},
		{"tokens", "Architecture", "discount", "wonder"},
		{"tokens", "Mathematics", "vp_per_token", "3"},
		{"tokens", "Law", "science", "scales"},
		{"tokens", "Urbanism", "chain_bonus", "4"},
		{"tokens", "Urbanism", "coins_gain", "6"},
		{"tokens", "Economy", "trade_income", "1"},
		{"tokens", "Strategy", "extra_shield", "1"},
		{"tokens", "Theology", "wonders_play_again", "1"},
	}
	for _, c := range cells {
		var tb *RecordTable
		for i := range tables {
			if tables[i].Name == c.table {
				tb = &tables[i]
			}
		}
		col := slices.Index(tb.Header, c.col)
		found := false
		for _, row := range tb.Rows {
			if row[1] == c.item {
				found = true
				if row[col] != c.want {
					t.Errorf("%s %s %s = %q, want %q", c.table, c.item, c.col, row[col], c.want)
				}
			}
		}
		if !found || col < 0 {
			t.Errorf("%s: no row %q or column %q", c.table, c.item, c.col)
		}
	}

	// Effects without a column are errors, as are two effects in one cell.
	bad := []struct {
		name    string
		cards   []CardDef
		wonders []WonderDef
		tokens  []TokenDef
	}{
		{"card with an extra turn", []CardDef{{ID: 17, Slug: "a", Name: "a", Age: 1, Color: Yellow, Effects: Fx(PlayAgain())}}, nil, nil},
		{"card with two coin rewards", []CardDef{{ID: 59, Slug: "a", Name: "a", Age: 3, Color: Yellow,
			Effects: Fx(CoinsPerCard(ScopeSelf, 1, Yellow), CoinsPerWonder(ScopeSelf, 2))}}, nil, nil},
		{"wonder with a trade fix", nil, []WonderDef{{ID: WonderPyramids, Slug: "pyramids", Name: "Pyramids",
			Effects: Fx(TradeFix(Wood))}}, nil},
		{"token with shields", nil, nil, []TokenDef{{ID: TokenLaw, Slug: "law", Name: "Law", Effects: Fx(Shields(1))}}},
	}
	for _, c := range bad {
		restore := ReplaceForTest(c.cards, c.wonders, c.tokens)
		_, err := RecordTables()
		restore()
		if err == nil {
			t.Errorf("%s: RecordTables accepted it", c.name)
		}
	}
}

// TestGoldenCSV compares the verification CSVs with testdata/*.golden.csv.
// ajayeb-gen catalog -format csv writes the same bytes.
func TestGoldenCSV(t *testing.T) {
	tables, err := RecordTables()
	if err != nil {
		t.Fatal(err)
	}
	for _, tb := range tables {
		var buf bytes.Buffer
		if err := tb.WriteCSV(&buf); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join("testdata", tb.Name+".golden.csv")
		if *update {
			if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%v (run go test -run TestGoldenCSV -update to create it)", err)
		}
		if !bytes.Equal(buf.Bytes(), want) {
			got := strings.Split(buf.String(), "\n")
			exp := strings.Split(string(want), "\n")
			for i := 0; i < max(len(got), len(exp)); i++ {
				var g, e string
				if i < len(got) {
					g = got[i]
				}
				if i < len(exp) {
					e = exp[i]
				}
				if g != e {
					t.Errorf("%s line %d:\n got  %q\n want %q", path, i+1, g, e)
				}
			}
			t.Errorf("%s differs from the catalog; if the change is deliberate, rerun with -update and review the diff", path)
		}
		if !strings.HasSuffix(buf.String(), "\n") || strings.Contains(buf.String(), "\r") {
			t.Errorf("%s must use bare newlines and end with one", path)
		}
	}
}
