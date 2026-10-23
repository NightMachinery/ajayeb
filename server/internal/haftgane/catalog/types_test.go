package catalog

import (
	"reflect"
	"testing"

	duelcatalog "github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// Effect fixtures come from engine-architecture section 3 and effects.md.
func effectFixtures() []struct {
	effect Effect
	timing Timing
} {
	return []struct {
		effect Effect
		timing Timing
	}{
		{Effect{Kind: EffProduce, N: 1, Res: Wood, ForSale: true}, Continuous},
		{Effect{Kind: EffProduceChoice, Mask: Raw, ForSale: false}, Continuous},
		{Effect{Kind: EffTradeDiscount, N: 1, Mask: Raw, Scope: Left}, Continuous},
		{Effect{Kind: EffShields, N: 2}, Continuous},
		{Effect{Kind: EffScience, Symbol: Compass}, Continuous | EndOfGame},
		{Effect{Kind: EffScienceChoice}, Continuous | EndOfGame},
		{Effect{Kind: EffCoins, N: 3}, WhenPlaced},
		{Effect{Kind: EffCoinsPerCard, N: 1, Colors: Brown.Mask(), Scope: Self | Left | Right}, WhenPlaced},
		{Effect{Kind: EffCoinsPerStage, N: 3, Scope: Self}, WhenPlaced},
		{Effect{Kind: EffPoints, N: 3}, EndOfGame},
		{Effect{Kind: EffPointsPerCard, N: 1, Colors: Blue.Mask(), Scope: Left | Right}, EndOfGame},
		{Effect{Kind: EffPointsPerStage, N: 1, Scope: Self | Left | Right}, EndOfGame},
		{Effect{Kind: EffPointsIfWonderComplete, N: 7}, EndOfGame},
		{Effect{Kind: EffPlayLastCard}, Continuous},
		{Effect{Kind: EffBuildFromDiscard}, EndOfTurn},
		{Effect{Kind: EffFreeBuild, When: FirstOfColour}, Continuous},
	}
}

func TestEveryEffectKindAndTiming(t *testing.T) {
	seen := [NumEffectKinds]bool{}
	for _, fixture := range effectFixtures() {
		e := fixture.effect
		if err := e.Validate(); err != nil {
			t.Fatalf("%s: %v", e.Kind, err)
		}
		if e.Kind.Timings() != fixture.timing {
			t.Errorf("%s timing = %v, want %v", e.Kind, e.Kind.Timings(), fixture.timing)
		}
		if seen[e.Kind] {
			t.Fatalf("duplicate kind %s", e.Kind)
		}
		seen[e.Kind] = true
		if e.N > 0 {
			bad := e
			bad.N = 0
			if bad.Validate() == nil {
				t.Errorf("%s accepted zero N", e.Kind)
			}
		}
	}
	for kind := EffNone + 1; kind < NumEffectKinds; kind++ {
		if !seen[kind] || kind.String() == "" || kind.Timings() == 0 {
			t.Errorf("uncovered effect %d", kind)
		}
	}
	for _, kind := range []EffectKind{EffNone, NumEffectKinds, 255} {
		if (Effect{Kind: kind}).Validate() == nil || kind.Timings() != 0 {
			t.Errorf("unknown kind %d passed", kind)
		}
	}
	for _, when := range []FreeCondition{FirstOfColour, FirstTurnOfAge, LastTurnOfAge} {
		if err := (Effect{Kind: EffFreeBuild, When: when}).Validate(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestEffectNegativeRules(t *testing.T) {
	cases := []Effect{
		{Kind: EffProduce}, {Kind: EffProduce, N: 1, Res: NumResources},
		{Kind: EffProduceChoice}, {Kind: EffProduceChoice, Mask: Wood.Mask()},
		{Kind: EffProduceChoice, Mask: 128}, {Kind: EffProduceChoice, Mask: Raw, N: 1},
		{Kind: EffTradeDiscount, Mask: Raw, Scope: Left},
		{Kind: EffTradeDiscount, N: 2, Mask: Raw, Scope: Left},
		{Kind: EffTradeDiscount, N: 1, Mask: Wood.Mask(), Scope: Left},
		{Kind: EffTradeDiscount, N: 1, Mask: Manufactured, Scope: Self},
		{Kind: EffTradeDiscount, N: 1, Mask: Raw, Scope: Left | Beyond},
		{Kind: EffShields}, {Kind: EffScience}, {Kind: EffScience, Symbol: NumSymbols},
		{Kind: EffCoins}, {Kind: EffCoinsPerCard, N: 1, Scope: Self},
		{Kind: EffCoinsPerCard, N: 1, Colors: Brown.Mask()},
		{Kind: EffCoinsPerCard, N: 1, Colors: NoColor.Mask(), Scope: Self},
		{Kind: EffCoinsPerCard, N: 1, Colors: 1 << 15, Scope: Self},
		{Kind: EffCoinsPerStage, N: 1, Scope: Beyond}, {Kind: EffPoints},
		{Kind: EffPointsPerCard, N: 1, Colors: Blue.Mask(), Scope: Beyond},
		{Kind: EffPointsPerStage, Scope: Self}, {Kind: EffPointsIfWonderComplete},
		{Kind: EffFreeBuild}, {Kind: EffFreeBuild, When: NumFreeConditions},
	}
	for _, e := range cases {
		if e.Validate() == nil {
			t.Errorf("accepted invalid effect %+v", e)
		}
	}
	// Every unused field rejects a nonzero value, for every kind, rather than
	// accidentally becoming a hidden new parameter on one effect.
	fields := []string{"N", "Res", "Mask", "Colors", "Scope", "Symbol", "ForSale", "When"}
	used := [][]string{
		{"N", "Res", "ForSale"}, {"Mask", "ForSale"}, {"N", "Mask", "Scope"}, {"N"},
		{"Symbol"}, {}, {"N"}, {"N", "Colors", "Scope"}, {"N", "Scope"}, {"N"},
		{"N", "Colors", "Scope"}, {"N", "Scope"}, {"N"}, {}, {}, {"When"},
	}
	for i, fixture := range effectFixtures() {
		for _, name := range fields {
			allowed := false
			for _, field := range used[i] {
				if name == field {
					allowed = true
				}
			}
			if allowed {
				continue
			}
			e := fixture.effect
			field := reflect.ValueOf(&e).Elem().FieldByName(name)
			if field.Kind() == reflect.Bool {
				field.SetBool(true)
			} else {
				field.SetUint(1)
			}
			if e.Validate() == nil {
				t.Errorf("%s accepted unused %s", e.Kind, name)
			}
		}
	}
}

func syntheticDesign() Design {
	return Design{ID: 1, Slug: "sample-alpha", Name: "Sample Alpha", NameID: 1, Age: 1, Color: Blue, Copies: [3]uint8{3}, Effects: []Effect{{Kind: EffPoints, N: 3}}}
}
func syntheticBoard() Board {
	b := Board{ID: BoardAlexandria, Slug: "alexandria", Name: "Alexandria", Wonder: "Synthetic monument", Start: Glass}
	for side := range b.Sides {
		b.Sides[side].NumStages = 3
		for i := 0; i < 3; i++ {
			b.Sides[side].Stages[i].Effects = []Effect{{Kind: EffPoints, N: 1}}
		}
	}
	return b
}

func TestSyntheticCatalogValidation(t *testing.T) {
	d, b := syntheticDesign(), syntheticBoard()
	if err := ValidateCatalog([]Design{d}, []Board{b}); err != nil {
		t.Fatal(err)
	}
	changes := []struct {
		name   string
		change func(*Design)
	}{
		{"id-zero", func(d *Design) { d.ID = 0 }}, {"id-range", func(d *Design) { d.ID = NumDesigns + 1 }},
		{"slug-upper", func(d *Design) { d.Slug = "Sample" }}, {"slug-prefix", func(d *Design) { d.Slug = "the-sample" }},
		{"slug-digit", func(d *Design) { d.Slug = "1-sample" }},
		{"slug-empty", func(d *Design) { d.Slug = "" }}, {"slug-long", func(d *Design) { d.Slug = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" }},
		{"slug-hyphen", func(d *Design) { d.Slug = "sample--alpha" }},
		{"slug-start", func(d *Design) { d.Slug = "-sample" }},
		{"slug-end", func(d *Design) { d.Slug = "sample-" }},
		{"slug-character", func(d *Design) { d.Slug = "sample_alpha" }},
		{"name-id-zero", func(d *Design) { d.NameID = 0 }},
		{"age-high", func(d *Design) { d.Age = 4 }},
		{"color-high", func(d *Design) { d.Color = NumColors }},
		{"marker-high", func(d *Design) { d.Copies = [3]uint8{8} }},
		{"chain-out-range", func(d *Design) { d.ChainOut = [2]Chain{NumChains + 1} }},
		{"name-empty", func(d *Design) { d.Name = "" }}, {"name-id", func(d *Design) { d.NameID = NumNames + 1 }},
		{"age", func(d *Design) { d.Age = 0 }}, {"color", func(d *Design) { d.Color = NoColor }},
		{"guild-age", func(d *Design) { d.Color = Purple }}, {"guild-copies", func(d *Design) { d.Color = Purple; d.Age = 3 }},
		{"missing-copies", func(d *Design) { d.Copies = [3]uint8{} }},
		{"marker-range", func(d *Design) { d.Copies = [3]uint8{2} }},
		{"marker-order", func(d *Design) { d.Copies = [3]uint8{4, 3} }},
		{"marker-duplicate", func(d *Design) { d.Copies = [3]uint8{3, 3} }},
		{"marker-padding", func(d *Design) { d.Copies = [3]uint8{3, 0, 5} }},
		{"cost-mix", func(d *Design) { d.Cost.Coins = 1; d.Cost.Res[Wood] = 1 }},
		{"cost-units", func(d *Design) { d.Cost.Res[Wood] = 8 }},
		{"effects-empty", func(d *Design) { d.Effects = nil }},
		{"chain-range", func(d *Design) { d.ChainIn = NumChains + 1 }},
		{"chain-padding", func(d *Design) { d.ChainOut = [2]Chain{0, 1} }},
		{"chain-duplicate", func(d *Design) { d.ChainOut = [2]Chain{1, 1} }},
		{"effect-invalid", func(d *Design) { d.Effects = []Effect{{Kind: EffPoints}} }},
		{"production-sale", func(d *Design) { d.Color = Brown; d.Effects = []Effect{{Kind: EffProduce, N: 1, Res: Wood}} }},
		{"grey-raw", func(d *Design) {
			d.Color = Grey
			d.Effects = []Effect{{Kind: EffProduce, N: 1, Res: Wood, ForSale: true}}
		}},
		{"yellow-sale", func(d *Design) {
			d.Color = Yellow
			d.Effects = []Effect{{Kind: EffProduceChoice, Mask: Raw, ForSale: true}}
		}},
	}
	for _, c := range changes {
		t.Run(c.name, func(t *testing.T) {
			d := syntheticDesign()
			c.change(&d)
			if d.Validate() == nil {
				t.Fatal("accepted invalid design")
			}
		})
	}
	boards := []func(*Board){
		func(b *Board) { b.ID = 0 }, func(b *Board) { b.Slug = "wrong" }, func(b *Board) { b.Name = "" }, func(b *Board) { b.Wonder = "" },
		func(b *Board) { b.Start = NumResources }, func(b *Board) { b.Sides[0].NumStages = 2 }, func(b *Board) { b.Sides[1].NumStages = 5 },
		func(b *Board) { b.Sides[1].Stages[3].Cost.Coins = 1 }, func(b *Board) { b.Sides[0].Stages[0].Effects = nil },
		func(b *Board) {
			b.Sides[0].Stages[0].Effects = []Effect{{Kind: EffProduce, N: 1, Res: Wood, ForSale: true}}
		},
	}
	for i, change := range boards {
		b := syntheticBoard()
		change(&b)
		if b.Validate() == nil {
			t.Errorf("board negative %d passed", i)
		}
	}
	if err := ValidateCatalog([]Design{d, d}, nil); err == nil {
		t.Error("duplicate id passed")
	}
	other := d
	other.ID = 2
	other.NameID = 2
	if err := ValidateCatalog([]Design{d, other}, nil); err == nil {
		t.Error("duplicate slug passed")
	}
	other.Slug = "sample-beta"
	if err := ValidateCatalog([]Design{d, other}, nil); err == nil {
		t.Error("one name with two ids passed")
	}
	other.Name = "Sample Beta"
	other.NameID = 1
	if err := ValidateCatalog([]Design{d, other}, nil); err == nil {
		t.Error("one name id with different names passed")
	}
}

func TestChainsAndTwins(t *testing.T) {
	a := syntheticDesign()
	a.ChainOut = [2]Chain{1}
	b := a
	b.ID = 2
	b.Slug = "sample-beta"
	b.Name = "Sample Beta"
	b.NameID = 2
	b.Age = 2
	b.ChainOut = [2]Chain{}
	b.ChainIn = 1
	if err := ValidateCatalog([]Design{a, b}, nil); err != nil {
		t.Fatal(err)
	}
	bad := b
	bad.Age = 1
	if ValidateCatalog([]Design{a, bad}, nil) == nil {
		t.Error("same-age provider passed")
	}
	if ValidateCatalog([]Design{a}, nil) == nil {
		t.Error("missing chain target passed")
	}
	if ValidateCatalog([]Design{b}, nil) == nil {
		t.Error("missing chain provider passed")
	}
	a = syntheticDesign()
	a.Color = Grey
	a.Effects = []Effect{{Kind: EffProduce, N: 1, Res: Glass, ForSale: true}}
	b = a
	b.ID = 2
	b.Slug = "sample-beta"
	b.Age = 2
	b.Copies = [3]uint8{4}
	if err := ValidateCatalog([]Design{a, b}, nil); err != nil {
		t.Fatal(err)
	}
	restore := ReplaceForTest([]Design{a, b}, nil)
	defer restore()
	if Twin[a.ID] != b.ID || Twin[b.ID] != a.ID {
		t.Fatal("twin index not symmetric")
	}
	b.Cost.Coins = 1
	if ValidateCatalog([]Design{a, b}, nil) == nil {
		t.Error("different twin cost passed")
	}
}

func TestResourceColorAndDesignSet(t *testing.T) {
	expected := []string{"none", "brown", "grey", "red", "blue", "yellow", "green", "purple"}
	for c, name := range expected {
		if Color(c).String() != name {
			t.Errorf("color %d = %q, want %q", c, Color(c), name)
		}
	}
	for r := Resource(0); r < NumResources; r++ {
		if !r.Mask().Has(r) {
			t.Fatal(r)
		}
	}
	for c := Brown; c < NumColors; c++ {
		if !c.Mask().Has(c) {
			t.Fatal(c)
		}
		var match bool
		for other := duelcatalog.Brown; other < duelcatalog.NumColors; other++ {
			if c.String() == other.String() {
				match = true
			}
		}
		if !match {
			t.Errorf("new wire color %s", c)
		}
	}
	set := DesignSetOf(1, 64, 80)
	if set.Count() != 3 || set.Has(0) || set.Has(81) {
		t.Fatal(set)
	}
	var got []DesignID
	for id, ok := set.Next(0); ok; id, ok = set.Next(id + 1) {
		got = append(got, id)
	}
	if !reflect.DeepEqual(got, []DesignID{1, 64, 80}) {
		t.Fatal(got)
	}
	set.Remove(64)
	if set.Has(64) || set.Count() != 2 {
		t.Fatal(set)
	}
}

func TestCatalogGraphNegativesAndHolderRules(t *testing.T) {
	makeDesign := func(id DesignID, age uint8) Design {
		d := syntheticDesign()
		d.ID = id
		d.NameID = NameID(id)
		d.Age = age
		d.Slug = "sample-" + string(rune('a'+id))
		d.Name = d.Slug
		return d
	}
	a, b, c := makeDesign(1, 1), makeDesign(2, 1), makeDesign(3, 2)
	a.ChainOut, b.ChainOut, c.ChainIn = [2]Chain{1}, [2]Chain{1}, 1
	if err := ValidateCatalog([]Design{a, b, c}, nil); err != nil {
		t.Fatal(err)
	}
	extra := makeDesign(4, 1)
	extra.ChainOut = [2]Chain{1}
	if ValidateCatalog([]Design{a, b, c, extra}, nil) == nil {
		t.Error("three chain providers passed")
	}
	extra = makeDesign(4, 3)
	extra.ChainIn = 1
	if ValidateCatalog([]Design{a, b, c, extra}, nil) == nil {
		t.Error("two chain targets passed")
	}
	d, e, f := makeDesign(4, 1), makeDesign(5, 1), makeDesign(6, 2)
	d.ChainOut, e.ChainOut, f.ChainIn = [2]Chain{2}, [2]Chain{2}, 2
	if ValidateCatalog([]Design{a, b, c, d, e, f}, nil) == nil {
		t.Error("two doubled chains passed")
	}
	board := syntheticBoard()
	if ValidateCatalog(nil, []Board{board, board}) == nil {
		t.Error("duplicate board passed")
	}
	for _, effects := range [][]Effect{
		{{Kind: EffProduce, N: 1, Res: Stone, ForSale: true}},
		{{Kind: EffProduceChoice, Mask: Wood.Mask() | Clay.Mask(), ForSale: true}},
	} {
		d := syntheticDesign()
		d.Color = Brown
		d.Effects = effects
		if err := d.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	for _, kind := range []EffectKind{EffFreeBuild, EffPlayLastCard, EffBuildFromDiscard} {
		effect := Effect{Kind: kind}
		if kind == EffFreeBuild {
			effect.When = FirstOfColour
		}
		d := syntheticDesign()
		d.Effects = []Effect{effect}
		if d.Validate() == nil {
			t.Errorf("%s on a card passed", kind)
		}
		b := syntheticBoard()
		b.Sides[0].Stages[0].Effects = []Effect{effect}
		if err := b.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	badBoards := []func(*Board){
		func(b *Board) { b.Sides[1].NumStages = 1 },
		func(b *Board) { b.Sides[0].Stages[0].Cost.Res[Wood] = 8 },
		func(b *Board) { b.Sides[1].Stages[3].Effects = []Effect{{Kind: EffPoints, N: 1}} },
		func(b *Board) {
			b.Sides[0].Stages[0].Effects = []Effect{{Kind: EffProduceChoice, Mask: Raw, ForSale: true}}
		},
	}
	for i, change := range badBoards {
		b := syntheticBoard()
		change(&b)
		if b.Validate() == nil {
			t.Errorf("board rule %d passed", i)
		}
	}
	d = makeDesign(1, 1)
	d.Color = Grey
	d.Effects = []Effect{{Kind: EffProduceChoice, Mask: Raw}}
	if d.Validate() == nil {
		t.Error("grey choice passed")
	}
	d.Color = Brown
	d.Effects = []Effect{{Kind: EffProduceChoice, Mask: Manufactured, ForSale: true}}
	if d.Validate() == nil {
		t.Error("brown manufactured choice passed")
	}
}
