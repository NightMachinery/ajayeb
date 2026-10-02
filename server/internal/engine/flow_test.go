package engine_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

// Tests of the turn machine through the public API: setup, the draft, the
// three play actions with payment, the end of a turn, the end of an age and
// the starter choice, the end of the game, legality reasons, and the
// scenario builder that most of these tests start from.

// ---------------------------------------------------------------------------
// Setup

// setupDump renders everything NewGame decides, by slug.
func setupDump(s *engine.State) string {
	var b strings.Builder
	cards := func(ids []catalog.CardID) string {
		names := make([]string, len(ids))
		for i, c := range ids {
			names[i] = c.String()
		}
		return strings.Join(names, " ")
	}
	wonders := func(ids []catalog.WonderID) string {
		names := make([]string, len(ids))
		for i, w := range ids {
			names[i] = w.String()
		}
		return strings.Join(names, " ")
	}
	tokens := func(ids []catalog.TokenID) string {
		names := make([]string, len(ids))
		for i, t := range ids {
			names[i] = t.String()
		}
		return strings.Join(names, " ")
	}
	fmt.Fprintf(&b, "first: %d\n", s.FirstSeat)
	fmt.Fprintf(&b, "draft: %s\n", wonders(s.DraftOrder[:]))
	fmt.Fprintf(&b, "box wonders: %s\n", wonders(s.BoxWonders[:]))
	fmt.Fprintf(&b, "board: %s\n", tokens(slices.Collect(s.BoardTokens.All())))
	fmt.Fprintf(&b, "box tokens: %s\n", tokens(s.BoxTokens[:]))
	for a := range engine.NumAges {
		fmt.Fprintf(&b, "age %d: %s\n", a+1, cards(s.Deals[a][:]))
		fmt.Fprintf(&b, "removed %d: %s\n", a+1, cards(s.Removed[a][:]))
	}
	fmt.Fprintf(&b, "box guilds: %s\n", cards(s.BoxGuilds[:]))
	return b.String()
}

// setupGolden pins NewGame's deal for two seeds. A change here changes
// every stored game: it needs a new RulesVersion.
var setupGolden = map[uint64]string{
	1: `first: 0
draft: appian-way great-lighthouse sphinx mausoleum great-library temple-of-artemis statue-of-zeus hanging-gardens
box wonders: colossus circus-maximus pyramids piraeus
board: agriculture economy law strategy theology
box tokens: urbanism masonry architecture philosophy mathematics
age 1: lumber-yard guard-tower logging-camp theater garrison apothecary workshop clay-pit tavern palisade stone-reserve press pharmacist glassworks quarry wood-reserve clay-pool stone-pit altar stable
removed 1: scriptorium clay-reserve baths
age 2: school parade-ground customs-house barracks brewery caravansery library sawmill aqueduct glass-blower forum laboratory statue walls dispensary rostrum temple horse-breeders brickyard archery-range
removed 2: drying-room shelf-quarry courthouse
age 3: academy obelisk arena armory observatory palace chamber-of-commerce lighthouse fortifications university pretorium study merchants-guild port moneylenders-guild pantheon siege-workshop gardens builders-guild senate
removed 3: arsenal circus town-hall
box guilds: scientists-guild shipowners-guild tacticians-guild magistrates-guild
`,
	42: `first: 0
draft: pyramids statue-of-zeus great-lighthouse mausoleum circus-maximus colossus sphinx piraeus
box wonders: temple-of-artemis hanging-gardens appian-way great-library
board: agriculture architecture economy mathematics urbanism
box tokens: law theology masonry philosophy strategy
age 1: scriptorium garrison baths theater palisade wood-reserve guard-tower clay-reserve tavern clay-pool lumber-yard quarry logging-camp stable glassworks apothecary workshop altar stone-reserve pharmacist
removed 1: clay-pit stone-pit press
age 2: parade-ground sawmill barracks library glass-blower drying-room courthouse forum brewery archery-range caravansery walls horse-breeders brickyard temple aqueduct statue rostrum laboratory customs-house
removed 2: dispensary shelf-quarry school
age 3: pantheon academy armory obelisk circus palace port tacticians-guild fortifications study town-hall gardens magistrates-guild arsenal lighthouse chamber-of-commerce siege-workshop merchants-guild senate observatory
removed 3: pretorium university arena
box guilds: builders-guild scientists-guild moneylenders-guild shipowners-guild
`,
}

func TestNewGameSetupGolden(t *testing.T) {
	for seed, want := range setupGolden {
		s := engine.NewGame(seed, engine.Options{})
		if got := setupDump(&s); got != want {
			t.Errorf("seed %d: NewGame deals\n%s\nwant\n%s", seed, got, want)
		}
	}
}

// TestNewGameSetupOrder derives each deal independently from the documented
// order of RNG use (setup.go) and compares.
func TestNewGameSetupOrder(t *testing.T) {
	ageCards := func(age uint8) []catalog.CardID {
		var out []catalog.CardID
		for d := range catalog.Cards() {
			if d.Age == age && !d.IsGuild() {
				out = append(out, d.ID)
			}
		}
		return out
	}
	var guilds []catalog.CardID
	for d := range catalog.Cards() {
		if d.IsGuild() {
			guilds = append(guilds, d.ID)
		}
	}
	for seed := uint64(0); seed < 64; seed++ {
		for _, opts := range []engine.Options{{}, {FirstPlayer: engine.FirstPlayerSeat0}, {FirstPlayer: engine.FirstPlayerSeat1}} {
			s := engine.NewGame(seed, opts)
			r := engine.NewRNG(seed)
			w := make([]catalog.WonderID, catalog.NumWonders)
			for i := range w {
				w[i] = catalog.WonderID(i + 1)
			}
			engine.ShuffleSlice(&r, w)
			tk := make([]catalog.TokenID, catalog.NumTokens)
			for i := range tk {
				tk[i] = catalog.TokenID(i + 1)
			}
			engine.ShuffleSlice(&r, tk)
			var want engine.State
			copy(want.DraftOrder[:], w[:8])
			copy(want.BoxWonders[:], w[8:])
			want.BoardTokens = catalog.TokenSetOf(tk[:5]...)
			copy(want.BoxTokens[:], tk[5:])
			for age := uint8(1); age <= 2; age++ {
				c := ageCards(age)
				engine.ShuffleSlice(&r, c)
				copy(want.Deals[age-1][:], c[:20])
				copy(want.Removed[age-1][:], c[20:])
			}
			third := ageCards(3)
			engine.ShuffleSlice(&r, third)
			copy(want.Removed[2][:], third[17:])
			g := slices.Clone(guilds)
			engine.ShuffleSlice(&r, g)
			copy(want.BoxGuilds[:], g[3:])
			deck := append(slices.Clone(third[:17]), g[:3]...)
			engine.ShuffleSlice(&r, deck)
			copy(want.Deals[2][:], deck)
			switch opts.FirstPlayer {
			case engine.FirstPlayerSeat0:
				want.FirstSeat = 0
			case engine.FirstPlayerSeat1:
				want.FirstSeat = 1
			default:
				want.FirstSeat = engine.Seat(r.IntN(2))
			}
			if s.DraftOrder != want.DraftOrder || s.BoxWonders != want.BoxWonders || s.BoardTokens != want.BoardTokens ||
				s.BoxTokens != want.BoxTokens || s.Deals != want.Deals || s.Removed != want.Removed ||
				s.BoxGuilds != want.BoxGuilds || s.FirstSeat != want.FirstSeat {
				t.Fatalf("seed %d %+v: NewGame does not follow the documented order:\n%s\nwant\n%s", seed, opts, setupDump(&s), setupDump(&want))
			}
		}
	}
}

func TestNewGameStart(t *testing.T) {
	seen := [2]int{}
	for seed := uint64(1); seed <= 200; seed++ {
		s := engine.NewGame(seed, engine.Options{})
		if err := engine.CheckInvariants(&s); err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		if s.Phase != engine.PhaseWonderDraft || s.Age != 0 || s.Ply != 0 || s.Seed != seed {
			t.Fatalf("seed %d: phase %s age %d ply %d", seed, s.Phase, s.Age, s.Ply)
		}
		if s.ToMove != s.FirstSeat || s.Active != s.FirstSeat || s.ExtraTurn {
			t.Fatalf("seed %d: first %d, to move %d, active %d", seed, s.FirstSeat, s.ToMove, s.Active)
		}
		for i, p := range s.Players {
			if p.Coins != engine.StartCoins || !p.City.Empty() || !p.Tokens.Empty() || p.Wonders != ([4]engine.WonderSlot{}) {
				t.Fatalf("seed %d: seat %d starts with %+v", seed, i, p)
			}
		}
		if s.Military != 0 || s.Looting != engine.LootAll || s.WondersBuilt != 0 || engine.Occupied(&s) != 0 {
			t.Fatalf("seed %d: pawn %d looting %#x built %d", seed, s.Military, s.Looting, s.WondersBuilt)
		}
		if s.BoardTokens.Count() != 5 {
			t.Fatalf("seed %d: %d tokens on the board", seed, s.BoardTokens.Count())
		}
		seen[s.FirstSeat]++

		// Fixing the first player changes nothing else in the deal.
		for _, fp := range []engine.FirstPlayerRule{engine.FirstPlayerSeat0, engine.FirstPlayerSeat1} {
			f := engine.NewGame(seed, engine.Options{FirstPlayer: fp})
			want := s
			want.Opts.FirstPlayer = fp
			want.FirstSeat = engine.Seat(fp - engine.FirstPlayerSeat0)
			want.ToMove, want.Active = want.FirstSeat, want.FirstSeat
			if f != want {
				t.Fatalf("seed %d: the %s option changed more than the first player", seed, fp)
			}
		}
	}
	if seen[0] < 70 || seen[1] < 70 {
		t.Errorf("first player over 200 seeds: %v", seen)
	}
}

// ---------------------------------------------------------------------------
// The wonder draft

func TestDraftOrder(t *testing.T) {
	for _, first := range []engine.Seat{0, 1} {
		opts := engine.Options{FirstPlayer: engine.FirstPlayerSeat0 + engine.FirstPlayerRule(first)}
		s := engine.NewGame(5, opts)
		order := s.DraftOrder
		p1, p2 := first, first.Opp()
		var movers []engine.Seat
		var offers []int
		var ev []engine.Event
		for s.Phase == engine.PhaseWonderDraft {
			movers = append(movers, s.ToMove)
			legal := engine.LegalActions(&s, nil)
			offers = append(offers, len(legal))
			// Each pick takes the first wonder on offer.
			if err := engine.Apply(&s, legal[0], &ev); err != nil {
				t.Fatal(err)
			}
			if err := engine.CheckInvariants(&s); err != nil {
				t.Fatal(err)
			}
		}
		if want := []engine.Seat{p1, p2, p2, p2, p1, p1}; !slices.Equal(movers, want) {
			t.Errorf("first %d: movers %v, want %v", first, movers, want)
		}
		if want := []int{4, 3, 2, 4, 3, 2}; !slices.Equal(offers, want) {
			t.Errorf("first %d: offers %v, want %v", first, offers, want)
		}
		want := []engine.Event{
			{Kind: engine.EvDraft, Seat: p1, Wonder: order[0]},
			{Kind: engine.EvDraft, Seat: p2, Wonder: order[1]},
			{Kind: engine.EvDraft, Seat: p2, Wonder: order[2]},
			{Kind: engine.EvDraft, Seat: p1, Wonder: order[3], A: 1},
			{Kind: engine.EvDraftRound, A: 2},
			{Kind: engine.EvDraft, Seat: p2, Wonder: order[4]},
			{Kind: engine.EvDraft, Seat: p1, Wonder: order[5]},
			{Kind: engine.EvDraft, Seat: p1, Wonder: order[6]},
			{Kind: engine.EvDraft, Seat: p2, Wonder: order[7], A: 1},
			{Kind: engine.EvAgeStart, A: 1},
		}
		if !slices.Equal(ev, want) {
			t.Errorf("first %d: events\n%+v\nwant\n%+v", first, ev, want)
		}
		ids := func(seat engine.Seat) [4]catalog.WonderID {
			var out [4]catalog.WonderID
			for k, ws := range s.Players[seat].Wonders {
				out[k] = ws.ID
			}
			return out
		}
		if got, want := ids(p1), [4]catalog.WonderID{order[0], order[3], order[5], order[6]}; got != want {
			t.Errorf("first %d: the first player has %v, want %v", first, got, want)
		}
		if got, want := ids(p2), [4]catalog.WonderID{order[1], order[2], order[4], order[7]}; got != want {
			t.Errorf("first %d: the second player has %v, want %v", first, got, want)
		}
		if s.Phase != engine.PhasePlay || s.Age != 1 || s.ToMove != p1 || s.Active != p1 || s.Ply != 6 {
			t.Errorf("first %d: after the draft phase %s age %d to move %d active %d ply %d", first, s.Phase, s.Age, s.ToMove, s.Active, s.Ply)
		}
		if engine.Occupied(&s) != 1<<engine.SlotsPerAge-1 {
			t.Errorf("first %d: Age I is not laid out", first)
		}
	}
}

func TestDraftOffersOnlyTheCurrentRound(t *testing.T) {
	s := engine.NewGame(8, engine.Options{FirstPlayer: engine.FirstPlayerSeat0})
	order := s.DraftOrder
	for _, w := range order[4:] {
		if r := enginetest.MustFail(t, &s, "draft "+w.String()); r != engine.ReasonNotInDraft {
			t.Errorf("round 2 wonder %s in round 1: reason %s", w, r)
		}
	}
	for _, w := range s.BoxWonders {
		if r := enginetest.MustFail(t, &s, "draft "+w.String()); r != engine.ReasonNotInDraft {
			t.Errorf("box wonder %s: reason %s", w, r)
		}
	}
	// Take the last wonder of round 1 first: the automatic pick then gets
	// the one left over.
	enginetest.Do(t, &s, "draft "+order[3].String())
	if r := enginetest.MustFail(t, &s, "draft "+order[3].String()); r != engine.ReasonNotInDraft {
		t.Errorf("a taken wonder: reason %s", r)
	}
	ev := enginetest.Do(t, &s, "draft "+order[0].String(), "draft "+order[2].String())
	if s.Players[0].Wonders[1].ID != order[1] || ev[2] != (engine.Event{Kind: engine.EvDraft, Seat: 0, Wonder: order[1], A: 1}) {
		t.Errorf("the automatic pick: seat 0 has %+v, events %+v", s.Players[0].Wonders, ev)
	}
	for _, step := range []string{"construct:baths", "token:law", "starter:0", "destroy:quarry", "revive:baths", "discard:baths"} {
		if r := enginetest.MustFail(t, &s, step); r != engine.ReasonWrongPhase {
			t.Errorf("%s during the draft: reason %s", step, r)
		}
	}
	if got := enginetest.Keys(&s); len(got) != 4 {
		t.Errorf("round 2 offers %v", got)
	}
}

// ---------------------------------------------------------------------------
// Taking a card

func TestDiscard(t *testing.T) {
	s := enginetest.New(t, 1).FirstPlayer(0).
		City(0, "stone-reserve", "clay-reserve").Accessible("theater").State()
	coins := s.Players[0].Coins
	if v := engine.DiscardValue(&s, 0); v != 4 {
		t.Fatalf("discard value %d, want 4 (2 plus 2 yellow cards)", v)
	}
	i := engine.SlotOf(&s, cardID(t, "theater"))
	ev := enginetest.Do(t, &s, "discard theater")
	if s.Players[0].Coins != coins+4 || !s.Discard.Has(cardID(t, "theater")) || s.Slots[i] != (engine.Slot{}) {
		t.Fatalf("after the discard: coins %d, discard %v, slot %+v", s.Players[0].Coins, s.Discard, s.Slots[i])
	}
	if s.Players[0].City.Has(cardID(t, "theater")) {
		t.Fatal("the discarded card entered the city")
	}
	if ev[0] != (engine.Event{Kind: engine.EvDiscard, Seat: 0, Card: cardID(t, "theater"), A: 4}) {
		t.Fatalf("events %+v", ev)
	}
	if s.ToMove != 1 || s.Active != 1 || s.Ply != 1 {
		t.Fatalf("to move %d, active %d, ply %d", s.ToMove, s.Active, s.Ply)
	}
}

func TestConstructPaysTheCheapestWay(t *testing.T) {
	for _, economy := range []bool{false, true} {
		b := enginetest.New(t, 2).FirstPlayer(0).Coins(0, 10).Coins(1, 5).
			City(1, "quarry").Accessible("baths")
		if economy {
			b.Tokens(1, "economy")
		}
		s := b.State()
		q := engine.BuildingQuote(&s, 0, cardID(t, "baths"))
		if q.Total != 3 { // one stone at 2 plus the opponent's quarry
			t.Fatalf("baths costs %d, want 3", q.Total)
		}
		ev := enginetest.Do(t, &s, "construct:baths")
		to := int16(0)
		if economy {
			to = 3
		}
		if s.Players[0].Coins != 7 || s.Players[1].Coins != 5+to {
			t.Errorf("economy %v: coins %d and %d", economy, s.Players[0].Coins, s.Players[1].Coins)
		}
		if !s.Players[0].City.Has(cardID(t, "baths")) || s.Players[0].ColorN[catalog.Blue] != 1 {
			t.Errorf("economy %v: baths is not in the city", economy)
		}
		want := []engine.Event{
			{Kind: engine.EvBuild, Seat: 0, Card: cardID(t, "baths")},
			{Kind: engine.EvPay, Seat: 0, A: 3, B: to},
		}
		if !slices.Equal(ev[:2], want) {
			t.Errorf("economy %v: events %+v", economy, ev)
		}
		if s.ToMove != 1 {
			t.Errorf("economy %v: seat %d to move", economy, s.ToMove)
		}
	}
}

func TestConstructFreeAndChained(t *testing.T) {
	s := enginetest.New(t, 3).Age(2).FirstPlayer(0).Coins(0, 0).
		City(0, "theater").Accessible("statue").State()
	ev := enginetest.Do(t, &s, "construct statue")
	if s.Players[0].Coins != 0 || !s.Players[0].City.Has(cardID(t, "statue")) {
		t.Fatalf("the chained statue: coins %d", s.Players[0].Coins)
	}
	for _, e := range ev {
		if e.Kind == engine.EvPay {
			t.Fatalf("a free build reported a payment: %+v", ev)
		}
	}
	// A card with a coin cost is not affordable with no coins.
	s = enginetest.New(t, 3).FirstPlayer(0).Coins(0, 1).Accessible("palisade").State()
	if r := enginetest.MustFail(t, &s, "construct palisade"); r != engine.ReasonUnaffordable {
		t.Fatalf("palisade with 1 coin: reason %s", r)
	}
	if !enginetest.IsLegal(&s, "discard palisade") {
		t.Fatal("discarding is not legal")
	}
}

func TestWonderTucksAnyAccessibleCard(t *testing.T) {
	// The Sphinx (S C GG) is fully covered by the city's production, so it
	// costs nothing; the tucked Sawmill (2 coins) could not be afforded.
	s := enginetest.New(t, 4).Age(2).FirstPlayer(0).Coins(0, 0).
		Wonders(0, "sphinx", "pyramids", "colossus", "piraeus").
		City(0, "quarry", "clay-pool", "glassworks", "glass-blower").
		Accessible("sawmill").State()
	if r := enginetest.MustFail(t, &s, "construct sawmill"); r != engine.ReasonUnaffordable {
		t.Fatalf("construct sawmill: reason %s", r)
	}
	if enginetest.IsLegal(&s, "wonder pyramids sawmill") {
		t.Fatal("the pyramids are affordable with no coins and no papyrus")
	}
	sawmill := cardID(t, "sawmill")
	ev := enginetest.Do(t, &s, "wonder sphinx sawmill")
	ws := s.Players[0].Wonders[0]
	if !ws.Built || ws.Tucked != sawmill || s.WondersBuilt != 1 {
		t.Fatalf("the sphinx: %+v, built %d", ws, s.WondersBuilt)
	}
	if s.Players[0].City.Has(sawmill) || s.Discard.Has(sawmill) || engine.SlotOf(&s, sawmill) >= 0 {
		t.Fatal("the tucked card is somewhere else too")
	}
	if ev[0] != (engine.Event{Kind: engine.EvWonder, Seat: 0, Wonder: catalog.WonderSphinx, Card: sawmill}) {
		t.Fatalf("events %+v", ev)
	}
	if r := enginetest.MustFail(t, &s, "wonder sphinx "+s.Slots[firstAccessible(&s)].Card.String()); r != engine.ReasonWonderUnavailable {
		t.Fatalf("the opponent builds seat 0's sphinx: reason %s", r)
	}
}

func TestWonderPayment(t *testing.T) {
	s := enginetest.New(t, 5).FirstPlayer(1).Coins(1, 20).Wonders(1, "pyramids").State()
	q := engine.WonderQuote(&s, 1, catalog.WonderPyramids)
	c := s.Slots[firstAccessible(&s)].Card
	ev := enginetest.Do(t, &s, "wonder:pyramids:"+c.String())
	if s.Players[1].Coins != 20-q.Total || q.Total <= 0 {
		t.Fatalf("coins %d after paying %d", s.Players[1].Coins, q.Total)
	}
	if ev[1] != (engine.Event{Kind: engine.EvPay, Seat: 1, A: q.Total}) {
		t.Fatalf("events %+v", ev)
	}
	// A built wonder cannot be built again, and the box wonders are no one's.
	s.ToMove, s.Active = 1, 1
	c = s.Slots[firstAccessible(&s)].Card
	for _, w := range []catalog.WonderID{catalog.WonderPyramids, s.BoxWonders[0], s.Players[0].Wonders[0].ID} {
		if r := enginetest.MustFail(t, &s, "wonder "+w.String()+" "+c.String()); r != engine.ReasonWonderUnavailable {
			t.Errorf("wonder %s: reason %s", w, r)
		}
	}
}

func TestSevenWonders(t *testing.T) {
	// Six wonders are built; seat 1 builds the seventh and its last one
	// goes back to the box.
	s := enginetest.New(t, 6).Age(2).FirstPlayer(1).Coins(1, 40).
		Wonders(0, "pyramids", "colossus", "great-lighthouse", "temple-of-artemis").
		Wonders(1, "mausoleum", "great-library", "piraeus", "hanging-gardens").
		Built(0, "pyramids", "").Built(0, "colossus", "").Built(0, "great-lighthouse", "").Built(0, "temple-of-artemis", "").
		Built(1, "mausoleum", "").Built(1, "great-library", "").State()
	if s.WondersBuilt != 6 {
		t.Fatalf("%d built", s.WondersBuilt)
	}
	c := s.Slots[firstAccessible(&s)].Card
	ev := enginetest.Do(t, &s, "wonder piraeus "+c.String())
	if s.WondersBuilt != 7 || !s.Players[1].Wonders[3].Removed || s.Players[1].Wonders[3].Built {
		t.Fatalf("after the seventh: built %d, %+v", s.WondersBuilt, s.Players[1].Wonders[3])
	}
	found := false
	for _, e := range ev {
		if e == (engine.Event{Kind: engine.EvWonderRemoved, Seat: 1, Wonder: catalog.WonderHangingGardens}) {
			found = true
		}
	}
	if !found {
		t.Fatalf("no EvWonderRemoved in %+v", ev)
	}
	s.ToMove, s.Active = 1, 1
	for _, k := range enginetest.Keys(&s) {
		if strings.HasPrefix(k, "wonder:") {
			t.Fatalf("a wonder is still legal: %s", k)
		}
	}
	// A scenario with seven built wonders removes the eighth itself.
	s = enginetest.New(t, 6).
		Wonders(0, "pyramids", "colossus", "great-lighthouse", "temple-of-artemis").
		Wonders(1, "mausoleum", "great-library", "piraeus", "hanging-gardens").
		Built(0, "pyramids", "").Built(0, "colossus", "").Built(0, "great-lighthouse", "").Built(0, "temple-of-artemis", "").
		Built(1, "mausoleum", "").Built(1, "great-library", "").Built(1, "hanging-gardens", "").State()
	if !s.Players[1].Wonders[2].Removed {
		t.Fatal("the scenario did not apply the seven-wonder rule")
	}
}

// firstAccessible returns the first accessible slot.
func firstAccessible(s *engine.State) int {
	acc := engine.Accessible(s)
	for i := range engine.SlotsPerAge {
		if acc&(1<<i) != 0 {
			return i
		}
	}
	return -1
}

// ---------------------------------------------------------------------------
// The end of a turn, of an age and of the game

func TestExtraTurn(t *testing.T) {
	s := enginetest.New(t, 7).FirstPlayer(0).State()
	s.ExtraTurn = true
	enginetest.Do(t, &s, "discard "+s.Slots[firstAccessible(&s)].Card.String())
	if s.ToMove != 0 || s.Active != 0 || s.ExtraTurn {
		t.Fatalf("after an extra turn: to move %d, active %d, extra %v", s.ToMove, s.Active, s.ExtraTurn)
	}
	enginetest.Do(t, &s, "discard "+s.Slots[firstAccessible(&s)].Card.String())
	if s.ToMove != 1 {
		t.Fatalf("the turn did not pass: seat %d to move", s.ToMove)
	}
}

func TestEndOfAge(t *testing.T) {
	cases := []struct {
		name    string
		pawn    int
		last    engine.Seat // who takes the last card
		literal bool        // Options.CenterPawn = LastActiveStarts
		chooser engine.Seat
		choice  bool
	}{
		{"centre: the last active player chooses", 0, 0, false, 0, true},
		{"centre: the last active player chooses (seat 1)", 0, 1, false, 1, true},
		{"centre, literal: the last active player starts", 0, 1, true, 1, false},
		{"seat 1 is weaker and chooses", 2, 0, false, 1, true},
		{"seat 1 is weaker and chooses (literal)", 4, 0, true, 1, true},
		{"seat 0 is weaker and chooses", -1, 1, false, 0, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			opts := engine.Options{}
			if c.literal {
				opts.CenterPawn = engine.LastActiveStarts
			}
			s := enginetest.New(t, 8).Options(opts).OnlyLeft("tavern").Pawn(c.pawn).ToMove(c.last).State()
			ev := enginetest.Do(t, &s, "discard tavern")
			if s.Age != 2 || engine.Occupied(&s) != 1<<engine.SlotsPerAge-1 {
				t.Fatalf("age %d is not laid out", s.Age)
			}
			if ev[len(ev)-1] != (engine.Event{Kind: engine.EvAgeStart, A: 2}) {
				t.Fatalf("events %+v", ev)
			}
			if !c.choice {
				if s.Phase != engine.PhasePlay || s.ToMove != c.last || s.Active != c.last {
					t.Fatalf("phase %s, to move %d", s.Phase, s.ToMove)
				}
				return
			}
			if s.Phase != engine.PhaseChooseStarter || s.ToMove != c.chooser || s.Active != c.last {
				t.Fatalf("phase %s, to move %d, active %d", s.Phase, s.ToMove, s.Active)
			}
			if got := enginetest.Keys(&s); !slices.Equal(got, []string{"starter:0", "starter:1"}) {
				t.Fatalf("legal %v", got)
			}
			if r := enginetest.MustFail(t, &s, "discard "+s.Slots[firstAccessible(&s)].Card.String()); r != engine.ReasonWrongPhase {
				t.Fatalf("a discard in the starter choice: reason %s", r)
			}
			for _, starter := range []engine.Seat{0, 1} {
				x := s
				ev := enginetest.Do(t, &x, fmt.Sprintf("starter %d", starter))
				if x.Phase != engine.PhasePlay || x.ToMove != starter || x.Active != starter {
					t.Fatalf("starter %d: phase %s, to move %d, active %d", starter, x.Phase, x.ToMove, x.Active)
				}
				if !slices.Equal(ev, []engine.Event{{Kind: engine.EvStarterChosen, Seat: c.chooser, A: int16(starter)}}) {
					t.Fatalf("starter %d: events %+v", starter, ev)
				}
			}
		})
	}
}

func TestExtraTurnLostAtTheEndOfAnAge(t *testing.T) {
	s := enginetest.New(t, 9).Age(2).OnlyLeft("brewery").ToMove(1).State()
	s.ExtraTurn = true
	ev := enginetest.Do(t, &s, "discard brewery")
	if s.ExtraTurn || s.Phase != engine.PhaseChooseStarter || s.Age != 3 {
		t.Fatalf("extra %v, phase %s, age %d", s.ExtraTurn, s.Phase, s.Age)
	}
	want := []engine.EventKind{engine.EvDiscard, engine.EvExtraTurnLost, engine.EvAgeStart}
	if got := kinds(ev); !slices.Equal(got, want) {
		t.Fatalf("events %v, want %v", got, want)
	}
	if ev[1].Seat != 1 {
		t.Fatalf("seat %d lost the extra turn", ev[1].Seat)
	}
}

func TestGameOverAfterAgeIII(t *testing.T) {
	s := enginetest.New(t, 10).Age(3).OnlyLeft("palace", "senate").FirstPlayer(0).State()
	enginetest.Do(t, &s, "discard palace")
	if s.Over() {
		t.Fatal("the game ended with a card left")
	}
	s.ExtraTurn = true // lost: the age ends
	ev := enginetest.Do(t, &s, "discard senate")
	if !s.Over() || s.Result.Kind != engine.VictoryCivilian || s.Result.Scores != engine.Scores(&s) {
		t.Fatalf("phase %s, result %+v", s.Phase, s.Result)
	}
	want := []engine.EventKind{engine.EvDiscard, engine.EvExtraTurnLost, engine.EvGameOver}
	if got := kinds(ev); !slices.Equal(got, want) {
		t.Fatalf("events %v, want %v", got, want)
	}
	if last := ev[len(ev)-1]; last.A != int16(engine.VictoryCivilian) || last.B != int16(s.Result.Winner) {
		t.Fatalf("EvGameOver %+v for result %+v", last, s.Result)
	}
	if got := enginetest.Keys(&s); len(got) != 0 {
		t.Fatalf("legal after the game: %v", got)
	}
	for _, step := range []string{"discard palace", "starter:0", "token:law", "draft:pyramids"} {
		if r := enginetest.MustFail(t, &s, step); r != engine.ReasonGameOver {
			t.Errorf("%s after the game: reason %s", step, r)
		}
	}
}

// kinds lists the kinds of events.
func kinds(ev []engine.Event) []engine.EventKind {
	out := make([]engine.EventKind, len(ev))
	for i, e := range ev {
		out[i] = e.Kind
	}
	return out
}

// ---------------------------------------------------------------------------
// Legality

func TestIllegalReasons(t *testing.T) {
	s := enginetest.New(t, 11).FirstPlayer(0).Coins(0, 0).
		City(0, "quarry").City(1, "clay-pool", "glassworks").Discard("garrison").
		Accessible("palisade").State()
	covered := s.Slots[0].Card // row 1 is covered at the start
	cases := []struct {
		step string
		want engine.Reason
	}{
		{"draft:pyramids", engine.ReasonWrongPhase},
		{"token:law", engine.ReasonWrongPhase},
		{"starter:1", engine.ReasonWrongPhase},
		{"destroy:clay-pool", engine.ReasonWrongPhase},
		{"revive:garrison", engine.ReasonWrongPhase},
		{"construct " + covered.String(), engine.ReasonNotAccessible},
		{"discard " + covered.String(), engine.ReasonNotAccessible},
		{"discard quarry", engine.ReasonNotAccessible},   // in a city
		{"discard garrison", engine.ReasonNotAccessible}, // in the discard pile
		{"construct palisade", engine.ReasonUnaffordable},
		{"wonder:" + s.Players[1].Wonders[0].ID.String() + ":palisade", engine.ReasonWonderUnavailable},
	}
	for _, c := range cases {
		if r := enginetest.MustFail(t, &s, c.step); r != c.want {
			t.Errorf("%s: reason %s, want %s", c.step, r, c.want)
		}
	}
	for _, a := range []engine.Action{{}, {Type: engine.ActDiscardBuilding}, {Type: 99}, {Type: engine.ActDiscardBuilding, Card: catalog.Unknown}} {
		before := s
		err := engine.Apply(&s, a, nil)
		var ie *engine.IllegalError
		if !errors.As(err, &ie) || ie.Reason != engine.ReasonMalformed || s != before {
			t.Errorf("%v: %v", a, err)
		}
	}
}

// TestPendingChoiceLegality checks the legal set and the reasons of each
// pending phase, in states built by hand (the hooks that open these phases
// belong to later slices).
func TestPendingChoiceLegality(t *testing.T) {
	base := enginetest.New(t, 12).Age(2).FirstPlayer(0).
		City(0, "lumber-yard", "press").City(1, "quarry", "clay-pit", "glass-blower", "baths").
		Discard("stable", "tavern").State()

	s := base
	s.Phase, s.Destroy = engine.PhaseDestroyCard, catalog.Brown
	if got := enginetest.Keys(&s); !slices.Equal(got, []string{"destroy:quarry", "destroy:clay-pit"}) {
		t.Errorf("destroy brown: %v", got)
	}
	for _, step := range []string{"destroy lumber-yard", "destroy glass-blower", "destroy baths", "destroy stable"} {
		if r := enginetest.MustFail(t, &s, step); r != engine.ReasonNotTarget {
			t.Errorf("%s: reason %s", step, r)
		}
	}
	if r := enginetest.MustFail(t, &s, "discard "+s.Slots[firstAccessible(&s)].Card.String()); r != engine.ReasonWrongPhase {
		t.Errorf("discard in the destroy choice: reason %s", r)
	}

	s = base
	s.Phase = engine.PhaseBuildFromDiscard
	if got := enginetest.Keys(&s); !slices.Equal(got, []string{"revive:stable", "revive:tavern"}) {
		t.Errorf("revive: %v", got)
	}
	if r := enginetest.MustFail(t, &s, "revive baths"); r != engine.ReasonNotInDiscard {
		t.Errorf("revive a city card: reason %s", r)
	}

	s = base
	s.Phase = engine.PhasePickBoardToken
	var want []string
	for tk := range s.BoardTokens.All() {
		want = append(want, "token:"+tk.String())
	}
	if got := enginetest.Keys(&s); !slices.Equal(got, want) {
		t.Errorf("board tokens: %v, want %v", got, want)
	}
	if r := enginetest.MustFail(t, &s, "token "+s.BoxTokens[0].String()); r != engine.ReasonTokenUnavailable {
		t.Errorf("a box token from the board: reason %s", r)
	}

	s = base
	s.Phase = engine.PhasePickLibraryToken
	copy(s.Library[:], s.BoxTokens[:3])
	want = []string{"token:" + s.Library[0].String(), "token:" + s.Library[1].String(), "token:" + s.Library[2].String()}
	if got := enginetest.Keys(&s); !slices.Equal(got, want) {
		t.Errorf("library: %v, want %v", got, want)
	}
	board, _ := s.BoardTokens.Next(1)
	for _, tk := range []catalog.TokenID{s.BoxTokens[3], board} {
		if r := enginetest.MustFail(t, &s, "token "+tk.String()); r != engine.ReasonTokenUnavailable {
			t.Errorf("library pick of %s: reason %s", tk, r)
		}
	}
}

// ---------------------------------------------------------------------------
// Scenarios

func cardID(t *testing.T, slug string) catalog.CardID {
	t.Helper()
	c, ok := catalog.CardBySlug(slug)
	if !ok {
		t.Fatalf("no card %q", slug)
	}
	return c
}

func TestScenarioPlacesEverything(t *testing.T) {
	s := enginetest.New(t, 13).Age(2).FirstPlayer(1).
		Wonders(0, "mausoleum", "sphinx").Wonders(1, "great-library").
		City(0, "lumber-yard", "glassworks", "sawmill").City(1, "guard-tower", "archery-range").
		Coins(0, 9).Coins(1, 0).Tokens(0, "law", "masonry").
		Built(0, "sphinx", "stable").Built(1, "great-library", "").
		Discard("baths", "theater").Pawn(-4).
		Accessible("temple", "rostrum").ToMove(0).State()
	p0, p1 := &s.Players[0], &s.Players[1]
	if s.Age != 2 || s.Phase != engine.PhasePlay || s.FirstSeat != 1 || s.ToMove != 0 || s.Active != 0 {
		t.Fatalf("age %d phase %s first %d to move %d", s.Age, s.Phase, s.FirstSeat, s.ToMove)
	}
	if p0.Wonders[0].ID != catalog.WonderMausoleum || p0.Wonders[1].ID != catalog.WonderSphinx || p1.Wonders[0].ID != catalog.WonderGreatLibrary {
		t.Fatalf("wonders %+v / %+v", p0.Wonders, p1.Wonders)
	}
	for _, slug := range []string{"lumber-yard", "glassworks", "sawmill"} {
		if !p0.City.Has(cardID(t, slug)) {
			t.Errorf("seat 0 lacks %s", slug)
		}
	}
	if p0.City.Count() != 3 || p1.City.Count() != 2 || p0.Coins != 9 || p1.Coins != 0 {
		t.Errorf("cities %d and %d, coins %d and %d", p0.City.Count(), p1.City.Count(), p0.Coins, p1.Coins)
	}
	if p0.Prod[catalog.Wood] != 3 || p0.Sci[catalog.Scales] != 1 || !p0.Tokens.Has(catalog.TokenMasonry) {
		t.Errorf("caches: prod %v sci %v tokens %#x", p0.Prod, p0.Sci, p0.Tokens)
	}
	if !p0.Wonders[1].Built || p0.Wonders[1].Tucked != cardID(t, "stable") || !p1.Wonders[0].Built || p1.Wonders[0].Tucked == 0 || s.WondersBuilt != 2 {
		t.Errorf("built wonders %+v / %+v", p0.Wonders, p1.Wonders)
	}
	if !s.Discard.Has(cardID(t, "baths")) || !s.Discard.Has(cardID(t, "theater")) || s.Discard.Count() != 2 {
		t.Errorf("discard %v", s.Discard)
	}
	if s.Military != -4 || s.Looting != engine.LootAll&^engine.LootMinus3 {
		t.Errorf("pawn %d, looting %#x", s.Military, s.Looting)
	}
	acc := engine.Accessible(&s)
	for _, slug := range []string{"temple", "rostrum"} {
		i := engine.SlotOf(&s, cardID(t, slug))
		if i < 0 || acc&(1<<i) == 0 || !s.Slots[i].FaceUp {
			t.Errorf("%s is not accessible (slot %d)", slug, i)
		}
	}
	// Age I is over: its deal is out of play, except the cards placed.
	if n := s.OutOfPlay.Count(); n < 20-7 || n > 20 {
		t.Errorf("%d cards out of play", n)
	}
	if !enginetest.IsLegal(&s, "construct temple") && !enginetest.IsLegal(&s, "discard temple") {
		t.Error("temple cannot be taken")
	}
}

func TestScenarioOnlyLeft(t *testing.T) {
	s := enginetest.New(t, 14).Age(3).OnlyLeft("palace", "obelisk", "arena").Accessible("arena").State()
	if engine.Occupied(&s) != 0b111 {
		t.Fatalf("occupied %020b", engine.Occupied(&s))
	}
	acc := engine.Accessible(&s)
	for _, slug := range []string{"palace", "obelisk", "arena"} {
		i := engine.SlotOf(&s, cardID(t, slug))
		if i < 0 || !s.Slots[i].FaceUp {
			t.Fatalf("%s is in slot %d", slug, i)
		}
	}
	if i := engine.SlotOf(&s, cardID(t, "arena")); acc&(1<<i) == 0 {
		t.Fatal("arena is not accessible")
	}
	if s.OutOfPlay.Count() != 20+20+17 {
		t.Fatalf("%d cards out of play", s.OutOfPlay.Count())
	}
}

func TestScenarioCurrentAgeCity(t *testing.T) {
	// Seven Age I cards in a city at the start of Age I: three come from
	// the removed cards, the others leave the structure from accessible
	// slots.
	s := enginetest.New(t, 15).
		City(0, "lumber-yard", "stone-pit", "clay-pool", "logging-camp").
		City(1, "quarry", "clay-pit", "glassworks").State()
	left := popcount(engine.Occupied(&s))
	for _, r := range s.Removed[0] {
		if r != 0 {
			left++
		}
	}
	if left != 23-7 {
		t.Fatalf("%d Age I cards left in the structure and the removed cards, want 16", left)
	}
	if popcount(engine.Occupied(&s)) >= 20 {
		t.Fatal("no card left the structure")
	}
}

func popcount(m uint32) int {
	n := 0
	for ; m != 0; m &= m - 1 {
		n++
	}
	return n
}

func TestScenarioTokens(t *testing.T) {
	s := enginetest.New(t, 16).Tokens(0, "law").Tokens(1, "economy").
		BoardTokens("agriculture", "urbanism", "theology").
		BoxOrder("philosophy", "strategy").State()
	if s.BoardTokens != catalog.TokenSetOf(catalog.TokenAgriculture, catalog.TokenUrbanism, catalog.TokenTheology) {
		t.Fatalf("board %v", slices.Collect(s.BoardTokens.All()))
	}
	if s.BoxTokens[0] != catalog.TokenPhilosophy || s.BoxTokens[1] != catalog.TokenStrategy {
		t.Fatalf("box %v", s.BoxTokens)
	}
	if !s.Players[0].Tokens.Has(catalog.TokenLaw) || !s.Players[1].Tokens.Has(catalog.TokenEconomy) {
		t.Fatal("tokens not given")
	}
	// Five board tokens with nothing owned: a sixth cannot fit in the box.
	_, err := engine.NewGameFromScenario(16, engine.Options{}, &engine.Scenario{BoardTokens: []string{"law"}})
	if err == nil {
		t.Fatal("a board of one token with nine in the box was accepted")
	}
}

func TestScenarioErrors(t *testing.T) {
	nine, two := 9, engine.Seat(2)
	cases := map[string]engine.Scenario{
		"unknown card":   {Discard: []string{"the-baths"}},
		"unknown token":  {Players: [2]engine.ScenarioPlayer{{Tokens: []string{"luck"}}}},
		"unknown wonder": {Wonders: &[2][4]string{{"pyramids", "sphinx", "colossus", "atlantis"}, {"mausoleum", "piraeus", "great-library", "appian-way"}}},
		"wonder twice":   {Wonders: &[2][4]string{{"pyramids", "sphinx", "colossus", "piraeus"}, {"mausoleum", "piraeus", "great-library", "appian-way"}}},
		"card twice":     {Discard: []string{"baths"}, Players: [2]engine.ScenarioPlayer{{City: []string{"baths"}}}},
		"wrong age":      {Age: 1, Accessible: []string{"palace"}},
		"not left":       {OnlyLeft: []string{"baths"}, Accessible: []string{"theater"}},
		"pawn":           {Pawn: &nine},
		"age":            {Age: 4},
		"to move":        {ToMove: &two},
		"not own wonder": {Players: [2]engine.ScenarioPlayer{{Built: []engine.ScenarioWonder{{Wonder: "pyramids"}}}, {Built: []engine.ScenarioWonder{{Wonder: "pyramids"}}}}},
		"token twice":    {Players: [2]engine.ScenarioPlayer{{Tokens: []string{"law"}}, {Tokens: []string{"law"}}}},
	}
	for name, sc := range cases {
		if _, err := engine.NewGameFromScenario(1, engine.Options{}, &sc); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestScenarioJSONAndDeterminism(t *testing.T) {
	sc := enginetest.New(t, 17).Age(2).City(0, "sawmill").Coins(1, 3).Tokens(1, "law").
		Built(0, "colossus", "").Wonders(0, "colossus").Discard("baths").Pawn(2).
		BoxOrder("economy").Accessible("brewery").Scenario()
	data, err := json.Marshal(sc)
	if err != nil {
		t.Fatal(err)
	}
	var back engine.Scenario
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	a, err := engine.NewGameFromScenario(17, engine.Options{}, sc)
	if err != nil {
		t.Fatal(err)
	}
	b, err := engine.NewGameFromScenario(17, engine.Options{}, &back)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("the scenario does not survive JSON: %s", data)
	}
	if plain, _ := engine.NewGameFromScenario(17, engine.Options{}, nil); plain != engine.NewGame(17, engine.Options{}) {
		t.Fatal("a nil scenario is not NewGame")
	}
	// Every action from the scenario state applies cleanly through a whole
	// game.
	pol := enginetest.NewRandomPolicy(17)
	if err := enginetest.Playout(&a, [2]enginetest.Policy{pol, pol}, 200, func(engine.Action, []engine.Event) error {
		return engine.CheckInvariants(&a)
	}); err != nil {
		t.Fatal(err)
	}
}

func TestRunTrigger(t *testing.T) {
	sc := &engine.Scenario{OnlyLeft: []string{"tavern"}, Trigger: []string{"discard:tavern", "starter:1"}}
	s, err := engine.NewGameFromScenario(18, engine.Options{}, sc)
	if err != nil {
		t.Fatal(err)
	}
	var ev []engine.Event
	if err := engine.RunTrigger(&s, sc, &ev); err != nil {
		t.Fatal(err)
	}
	if s.Age != 2 || s.Phase != engine.PhasePlay || s.ToMove != 1 {
		t.Fatalf("after the trigger: age %d phase %s to move %d", s.Age, s.Phase, s.ToMove)
	}
	sc.Trigger = []string{"discard:tavern", "discard:tavern"}
	s, _ = engine.NewGameFromScenario(18, engine.Options{}, sc)
	if err := engine.RunTrigger(&s, sc, nil); err == nil {
		t.Fatal("an illegal trigger step was applied")
	}
}
