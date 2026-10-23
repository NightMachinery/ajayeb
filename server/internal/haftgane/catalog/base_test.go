package catalog

// Structural tests of the base catalog: the counts of docs/haftgane/effects.md
// ("Counts that must check out" and "Sizes the engine must hold") and the
// test list of slice haftgane-catalog-data in
// docs/haftgane/engine-architecture.md section 14.

import (
	"slices"
	"strconv"
	"strings"
	"testing"
)

func baseDesign(t *testing.T, id DesignID) Design {
	t.Helper()
	d, ok := DesignByID(id)
	if !ok {
		t.Fatalf("design %d not registered", id)
	}
	return d
}

func baseDesignBySlug(t *testing.T, slug string) Design {
	t.Helper()
	id, ok := DesignBySlug(slug)
	if !ok {
		t.Fatalf("no design %s", slug)
	}
	return baseDesign(t, id)
}

func baseBoard(t *testing.T, id BoardID) Board {
	t.Helper()
	b, ok := BoardByID(id)
	if !ok {
		t.Fatalf("board %d not registered", id)
	}
	return b
}

func baseCopies(d Design) int {
	n := 0
	for _, m := range d.Copies {
		if m != 0 {
			n++
		}
	}
	if d.Color == Purple {
		return 1 // a guild is one physical card
	}
	return n
}

// 80 designs, 77 names and 148 physical cards: Age I 49, Age II 49, Age III
// 40 and 10 guilds; 27, 23, 20 and 10 designs.
func TestBaseCounts(t *testing.T) {
	var designs, copies [4]int
	var guilds, guildCopies int
	names := map[NameID]bool{}
	for id := DesignID(1); id <= NumDesigns; id++ {
		d := baseDesign(t, id)
		names[d.NameID] = true
		if d.Color == Purple {
			guilds++
			guildCopies += baseCopies(d)
			continue
		}
		designs[d.Age]++
		copies[d.Age] += baseCopies(d)
	}
	t.Logf("designs %v guilds %d; copies %v guild cards %d; names %d", designs[1:], guilds, copies[1:], guildCopies, len(names))
	if designs != [4]int{0, 27, 23, 20} || guilds != NumGuilds {
		t.Errorf("designs per Age %v and %d guilds, want [27 23 20] and 10", designs[1:], guilds)
	}
	if copies != [4]int{0, 49, 49, 40} || guildCopies != 10 {
		t.Errorf("copies per Age %v and %d guild cards, want [49 49 40] and 10", copies[1:], guildCopies)
	}
	if total := copies[1] + copies[2] + copies[3] + guildCopies; total != 148 {
		t.Errorf("%d physical cards, want 148", total)
	}
	if len(names) != NumNames {
		t.Errorf("%d names, want %d", len(names), NumNames)
	}
	if T.All.Count() != NumDesigns || T.Guilds.Count() != NumGuilds {
		t.Errorf("compiled sets: %d designs, %d guilds", T.All.Count(), T.Guilds.Count())
	}
}

// Every Age holds 7N cards for N players, 3 to 7; Age III holds N + 2 guilds
// among them, so 6N - 2 other cards.
func TestDeckSizes(t *testing.T) {
	for n := uint8(3); n <= 7; n++ {
		var perAge [4]int
		for id := DesignID(1); id <= NumDesigns; id++ {
			d := baseDesign(t, id)
			perAge[d.Age] += int(CopiesAt(id, n))
		}
		guilds := int(n) + 2
		t.Logf("N=%d: Age I %d, Age II %d, Age III %d + %d guilds", n, perAge[1], perAge[2], perAge[3], guilds)
		if perAge[1] != 7*int(n) || perAge[2] != 7*int(n) || perAge[3]+guilds != 7*int(n) {
			t.Errorf("N=%d: decks %v + %d guilds, want %d each", n, perAge[1:], guilds, 7*n)
		}
	}
	if CopiesAt(0, 7) != 0 || CopiesAt(NumDesigns+1, 7) != 0 {
		t.Error("CopiesAt accepted an invalid id")
	}
}

// Physical cards per colour at 7 players (effects.md), and designs per
// colour, per Age.
func TestColourCountsPerAge(t *testing.T) {
	wantCopies := [4][NumColors]int{
		1: {Brown: 14, Grey: 6, Blue: 8, Yellow: 9, Red: 6, Green: 6},
		2: {Brown: 8, Grey: 6, Blue: 8, Yellow: 10, Red: 9, Green: 8},
		3: {Blue: 10, Yellow: 10, Red: 10, Green: 10, Purple: 10},
	}
	wantDesigns := [4][NumColors]int{
		1: {Brown: 10, Grey: 3, Blue: 4, Yellow: 4, Red: 3, Green: 3},
		2: {Brown: 4, Grey: 3, Blue: 4, Yellow: 4, Red: 4, Green: 4},
		3: {Blue: 5, Yellow: 5, Red: 5, Green: 5, Purple: 10},
	}
	var copies, designs [4][NumColors]int
	for id := DesignID(1); id <= NumDesigns; id++ {
		d := baseDesign(t, id)
		designs[d.Age][d.Color]++
		if d.Color == Purple {
			copies[d.Age][d.Color]++
		} else {
			copies[d.Age][d.Color] += int(CopiesAt(id, 7))
		}
	}
	for age := 1; age <= 3; age++ {
		if copies[age] != wantCopies[age] {
			t.Errorf("Age %d copies at 7 players per colour %v, want %v", age, copies[age], wantCopies[age])
		}
		if designs[age] != wantDesigns[age] {
			t.Errorf("Age %d designs per colour %v, want %v", age, designs[age], wantDesigns[age])
		}
	}
	for c := Brown; c < NumColors; c++ {
		n := 0
		for age := 1; age <= 3; age++ {
			n += designs[age][c]
		}
		if T.ByColor[c].Count() != n {
			t.Errorf("compiled %s set has %d designs, want %d", c, T.ByColor[c].Count(), n)
		}
	}
}

// 24 chain symbols, numbered in Age order (by the id of the design each
// frees), 25 provider links, Forum the only design with two providers, every
// provider of an earlier Age, and two links from Age I straight to Age III.
func TestChainSymbols(t *testing.T) {
	links, doubles, skip := 0, 0, 0
	var targets [NumChains + 1]DesignID
	for id := DesignID(1); id <= NumDesigns; id++ {
		d := baseDesign(t, id)
		if d.ChainIn != 0 {
			if targets[d.ChainIn] != 0 {
				t.Errorf("symbol %d frees two designs", d.ChainIn)
			}
			targets[d.ChainIn] = id
		}
		for _, c := range d.ChainOut {
			if c == 0 {
				continue
			}
			links++
			target := baseDesign(t, T.ChainTarget[c])
			if target.Age <= d.Age {
				t.Errorf("%s provides symbol %d of %s, not an earlier Age", d.Slug, c, target.Slug)
			}
			if target.Age-d.Age == 2 {
				skip++
			}
		}
	}
	previous := DesignID(0)
	for c := Chain(1); c <= NumChains; c++ {
		if targets[c] == 0 {
			t.Fatalf("symbol %d frees nothing", c)
		}
		if targets[c] <= previous {
			t.Errorf("symbol %d frees %d after symbol %d freed %d: not in Age order", c, targets[c], c-1, previous)
		}
		previous = targets[c]
		if T.ChainTarget[c] != targets[c] {
			t.Errorf("compiled target of %d", c)
		}
		if T.ChainProviders[c][1] != 0 {
			doubles++
			if baseDesign(t, targets[c]).Slug != "forum" {
				t.Errorf("symbol %d has two providers", c)
			}
		}
	}
	t.Logf("%d symbols, %d provider links, %d doubles, %d Age I to Age III links", NumChains, links, doubles, skip)
	if links != 25 || doubles != 1 || skip != 2 {
		t.Errorf("links %d doubles %d skips %d, want 25, 1 and 2", links, doubles, skip)
	}
	if baseDesign(t, targets[1]).Age != 2 || baseDesign(t, targets[10]).Age != 2 || baseDesign(t, targets[11]).Age != 3 {
		t.Error("ten Age II targets should come first")
	}
	for _, link := range [][2]string{{"altar", "pantheon"}, {"theater", "gardens"}} {
		from, to := baseDesignBySlug(t, link[0]), baseDesignBySlug(t, link[1])
		if from.ChainOut[0] != to.ChainIn || to.ChainIn == 0 {
			t.Errorf("%s should free %s", link[0], link[1])
		}
	}
}

// 7 boards, 14 sides, 41 stages; day sides have 3 stages, night sides 2 to 4.
// Every stage costs resources only, at least one unit and at most 7 (review
// nit 3 of the core types).
func TestBoardsAndStages(t *testing.T) {
	wantNight := [NumBoards + 1]uint8{BoardAlexandria: 3, BoardBabylon: 2, BoardEphesos: 3, BoardGizah: 4,
		BoardHalikarnassos: 3, BoardOlympia: 3, BoardRhodos: 2}
	sides, stages := 0, 0
	for id := BoardID(1); id <= NumBoards; id++ {
		b := baseBoard(t, id)
		if b.Slug != id.String() {
			t.Errorf("board %d slug %s", id, b.Slug)
		}
		if b.Sides[0].NumStages != 3 || b.Sides[1].NumStages != wantNight[id] {
			t.Errorf("%s: %d day and %d night stages, want 3 and %d", b.Slug, b.Sides[0].NumStages, b.Sides[1].NumStages, wantNight[id])
		}
		for side, def := range b.Sides {
			sides++
			for i := range def.NumStages {
				stages++
				c := def.Stages[i].Cost
				if c.Coins != 0 || c.Units() == 0 || c.Units() > MaxCostUnits {
					t.Errorf("%s side %d stage %d costs %s: want resources only, 1 to %d units", b.Slug, side, i+1, CostCell(c), MaxCostUnits)
				}
			}
		}
	}
	t.Logf("%d boards, %d sides, %d stages", NumBoards, sides, stages)
	if sides != 14 || stages != 41 {
		t.Errorf("%d sides and %d stages, want 14 and 41", sides, stages)
	}
}

// No design costs both coins and resources, none costs more than U = 7
// units, and only ten cost coins: 1 coin each, the six Age I brown choice
// designs and the four Age II double brown designs.
func TestDesignCosts(t *testing.T) {
	var coinDesigns []string
	maxUnits := uint16(0)
	for id := DesignID(1); id <= NumDesigns; id++ {
		d := baseDesign(t, id)
		if d.Cost.Coins != 0 && d.Cost.Units() != 0 {
			t.Errorf("%s costs coins and resources", d.Slug)
		}
		if d.Cost.Units() > MaxCostUnits {
			t.Errorf("%s costs %d units", d.Slug, d.Cost.Units())
		}
		maxUnits = max(maxUnits, d.Cost.Units())
		if d.Cost.Coins != 0 {
			coinDesigns = append(coinDesigns, d.Slug)
			if d.Cost.Coins != 1 || d.Color != Brown {
				t.Errorf("%s costs %d coins", d.Slug, d.Cost.Coins)
			}
		}
	}
	want := []string{"clay-pit", "excavation", "forest-cave", "mine", "timber-yard", "tree-farm", "brickyard", "foundry", "quarry", "sawmill"}
	if !slices.Equal(coinDesigns, want) {
		t.Errorf("coin-cost designs %v, want %v", coinDesigns, want)
	}
	if maxUnits != MaxCostUnits {
		t.Errorf("largest cost %d units, want %d (the Palace)", maxUnits, MaxCostUnits)
	}
}

// Glassworks, Press and Loom are twins: one name, an Age I and an Age II
// design, identical but for Age and copies. No other name has two designs.
func TestTwins(t *testing.T) {
	pairs := 0
	for id := DesignID(1); id <= NumDesigns; id++ {
		d := baseDesign(t, id)
		other := Twin[id]
		if T.ByName[d.NameID].Count() != map[bool]int{true: 2, false: 1}[other != 0] {
			t.Errorf("%s: name set has %d designs", d.Slug, T.ByName[d.NameID].Count())
		}
		if other == 0 {
			continue
		}
		pairs++
		o := baseDesign(t, other)
		if Twin[other] != id || o.NameID != d.NameID || o.Name != d.Name {
			t.Errorf("%s: asymmetric twin", d.Slug)
		}
		a, b := d, o
		a.ID, a.Slug, a.Age, a.Copies = 0, "", 0, [3]uint8{}
		b.ID, b.Slug, b.Age, b.Copies = 0, "", 0, [3]uint8{}
		if !sameTwin(a, b) || d.Slug != strings.ToLower(d.Name)+"-age-"+strconv.Itoa(int(d.Age)) {
			t.Errorf("%s and %s differ beyond Age and copies", d.Slug, o.Slug)
		}
	}
	if pairs != 6 {
		t.Errorf("%d twin entries, want 6 (three pairs)", pairs)
	}
	for _, name := range []string{"glassworks", "press", "loom"} {
		a, b := baseDesignBySlug(t, name+"-age-1"), baseDesignBySlug(t, name+"-age-2")
		if Twin[a.ID] != b.ID || a.Age != 1 || b.Age != 2 || a.Copies == b.Copies {
			t.Errorf("%s twins", name)
		}
	}
}

// Ids follow registration order: the three Ages in turn, the guilds last;
// within each, colour in the enum's order, then English name. Name ids
// number the names in order of their first design. Slugs are the kebab-case
// English name, with the Age for twins, at most 30 bytes.
func TestRegistrationOrder(t *testing.T) {
	group := func(d Design) int {
		if d.Color == Purple {
			return 4
		}
		return int(d.Age)
	}
	nextName := NameID(1)
	for id := DesignID(1); id <= NumDesigns; id++ {
		d := baseDesign(t, id)
		if d.ID != id {
			t.Fatalf("design at %d has id %d", id, d.ID)
		}
		if id > 1 {
			p := baseDesign(t, id-1)
			if g, pg := group(d), group(p); g < pg || g == pg && (d.Color < p.Color || d.Color == p.Color && d.Name <= p.Name) {
				t.Errorf("%s (%d) is out of order after %s", d.Slug, id, p.Slug)
			}
		}
		if Twin[id] != 0 && Twin[id] < id {
			if d.NameID != baseDesign(t, Twin[id]).NameID {
				t.Errorf("%s: twin name id", d.Slug)
			}
		} else {
			if d.NameID != nextName {
				t.Errorf("%s: name id %d, want %d", d.Slug, d.NameID, nextName)
			}
			nextName++
		}
		want := strings.ReplaceAll(strings.ToLower(d.Name), " ", "-")
		if Twin[id] != 0 {
			want += "-age-" + strconv.Itoa(int(d.Age))
		}
		if d.Slug != want || len(d.Slug) > MaxSlugBytes {
			t.Errorf("slug %q, want %q within %d bytes", d.Slug, want, MaxSlugBytes)
		}
		if back, ok := DesignBySlug(d.Slug); !ok || back != id {
			t.Errorf("%s does not round-trip", d.Slug)
		}
		if g := group(d); g == 4 && (d.Age != 3 || T.Ages[3].Has(id)) || g < 4 && !T.Ages[g].Has(id) {
			t.Errorf("%s: compiled Age sets", d.Slug)
		}
	}
	if nextName != NumNames+1 {
		t.Errorf("name ids end at %d", nextName-1)
	}
}

// The number of items holding each effect kind, from the inventory of
// effects.md: designs and stages, not physical cards.
func TestEffectInventoryCounts(t *testing.T) {
	var items [NumEffectKinds]int
	var conditions [NumFreeConditions]int
	count := func(fx []Effect) {
		var seen [NumEffectKinds]bool
		for _, e := range fx {
			if !seen[e.Kind] {
				items[e.Kind]++
				seen[e.Kind] = true
			}
			if e.Kind == EffFreeBuild {
				conditions[e.When]++
			}
		}
	}
	for id := DesignID(1); id <= NumDesigns; id++ {
		count(baseDesign(t, id).Effects)
	}
	for id := BoardID(1); id <= NumBoards; id++ {
		for _, def := range baseBoard(t, id).Sides {
			for i := range def.NumStages {
				count(def.Stages[i].Effects)
			}
		}
	}
	want := [NumEffectKinds]int{
		EffProduce: 14, EffProduceChoice: 11, EffTradeDiscount: 3, EffShields: 15, EffScience: 12,
		EffScienceChoice: 3, EffCoins: 7, EffCoinsPerCard: 6, EffCoinsPerStage: 1, EffPoints: 43,
		EffPointsPerCard: 11, EffPointsPerStage: 2, EffPointsIfWonderComplete: 1, EffPlayLastCard: 1,
		EffBuildFromDiscard: 4, EffFreeBuild: 3,
	}
	for k := EffNone + 1; k < NumEffectKinds; k++ {
		if items[k] != want[k] {
			t.Errorf("%s on %d items, want %d", k, items[k], want[k])
		}
	}
	if conditions != [NumFreeConditions]int{0, 1, 1, 1} {
		t.Errorf("free-build conditions %v, want one of each", conditions)
	}
}

// The compiled tables equal a fresh compile, and the record tables render
// every effect of the catalog.
func TestCompiledTables(t *testing.T) {
	if T != compile(reg) {
		t.Error("T differs from a fresh compile")
	}
	for id := DesignID(1); id <= NumDesigns; id++ {
		var bits uint32
		for _, c := range baseDesign(t, id).ChainOut {
			bits |= c.Bit()
		}
		if T.ChainOutBits[id] != bits {
			t.Errorf("design %d chain bits", id)
		}
	}
	if _, err := RecordTables(); err != nil {
		t.Fatal(err)
	}
}

// The cost notation round-trips and rejects malformed costs.
func TestCostNotation(t *testing.T) {
	for _, s := range []string{"", "1c", "WCSOGPL", "SSOG", "WWWW"} {
		c, err := ParseCost(s)
		if err != nil || CostString(c) != s {
			t.Errorf("%q: %v, %q", s, err, CostString(c))
		}
	}
	for _, s := range []string{"c", "0c", "1cW", "X", "w", "256c"} {
		if _, err := ParseCost(s); err == nil {
			t.Errorf("%q accepted", s)
		}
	}
}
