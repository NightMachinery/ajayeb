package engine

import (
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// Tests of the effects of buildings at the level of the hooks: the coins a
// building pays (fixed, per card, per wonder, with the self and max-city
// scopes), the order of a building's effects, the effects of a building
// rebuilt from the discard pile, the coin helpers, the pawn's extreme moves,
// the once-per-symbol pair and the end-of-turn supremacy check. They are in
// package engine to reach the hooks; tokens_test.go shares their helpers
// (prefixed fx). The scripted tests of the same rules through the public API
// are in military_test.go, science_test.go and score_test.go.

// fxNew builds a scenario position and checks its invariants.
func fxNew(t *testing.T, sc Scenario) State {
	t.Helper()
	s, err := NewGameFromScenario(11, Options{}, &sc)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// fxDo applies action keys in order, checking the invariants after each, and
// returns all their events.
func fxDo(t *testing.T, s *State, keys ...string) []Event {
	t.Helper()
	var all []Event
	for _, k := range keys {
		a, err := ParseKey(k)
		if err != nil {
			t.Fatal(err)
		}
		var ev []Event
		if err := Apply(s, a, &ev); err != nil {
			t.Fatalf("%s: %v", k, err)
		}
		if err := CheckInvariants(s); err != nil {
			t.Fatalf("after %s: %v", k, err)
		}
		all = append(all, ev...)
	}
	return all
}

func fxInt(n int) *int                { return &n }
func fxSeat(s Seat) *Seat             { return &s }
func fxCity(slugs ...string) []string { return slugs }

// fxToken returns the id of a token slug.
func fxToken(t *testing.T, slug string) catalog.TokenID {
	t.Helper()
	tk, ok := catalog.TokenBySlug(slug)
	if !ok {
		t.Fatalf("no token %q", slug)
	}
	return tk
}

// fxOfKind returns the events of one kind, in order.
func fxOfKind(ev []Event, k EventKind) []Event {
	var out []Event
	for _, e := range ev {
		if e.Kind == k {
			out = append(out, e)
		}
	}
	return out
}

// fxCardCoins returns the coins card c paid in the events (0 if it reported
// none), failing if it reported more than one gain.
func fxCardCoins(t *testing.T, ev []Event, c catalog.CardID) int16 {
	t.Helper()
	var got []Event
	for _, e := range fxOfKind(ev, EvCoins) {
		if e.Card == c {
			got = append(got, e)
		}
	}
	switch len(got) {
	case 0:
		return 0
	case 1:
		return got[0].A
	}
	t.Fatalf("%s reported %d coin events: %+v", c, len(got), got)
	return 0
}

// The wonders every fx scenario gives the players, so that built wonders can
// be named.
var fxWonders = [2][WondersPerPlayer]string{
	{"pyramids", "sphinx", "colossus", "piraeus"},
	{"mausoleum", "great-library", "appian-way", "circus-maximus"},
}

func TestBuildingCoins(t *testing.T) {
	built := func(ws ...string) []ScenarioWonder {
		var out []ScenarioWonder
		for _, w := range ws {
			out = append(out, ScenarioWonder{Wonder: w})
		}
		return out
	}
	cases := []struct {
		name   string
		age    int
		build  string
		mine   []string
		theirs []string
		myW    []ScenarioWonder
		theirW []ScenarioWonder
		want   int16
	}{
		{name: "tavern: 4 coins", age: 1, build: "tavern", want: 4},
		{name: "brewery: 6 coins", age: 2, build: "brewery", want: 6},
		{name: "a building with no coin effect", age: 1, build: "theater", want: 0},
		{name: "lighthouse counts itself", age: 3, build: "lighthouse",
			mine: fxCity("tavern", "stone-reserve"), theirs: fxCity("clay-reserve", "wood-reserve", "forum", "customs-house"), want: 3},
		{name: "arena: own built wonders only", age: 3, build: "arena",
			myW: built("pyramids", "sphinx"), theirW: built("mausoleum", "great-library", "appian-way"), want: 4},
		{name: "arena with no wonder", age: 3, build: "arena", theirW: built("mausoleum"), want: 0},
		{name: "chamber of commerce: 3 per own grey card", age: 3, build: "chamber-of-commerce",
			mine: fxCity("press", "drying-room", "glassworks"), theirs: fxCity("glass-blower"), want: 9},
		{name: "port: 2 per own brown card", age: 3, build: "port",
			mine: fxCity("lumber-yard", "quarry"), theirs: fxCity("clay-pool", "clay-pit", "stone-pit"), want: 4},
		{name: "armory: 1 per own red card", age: 3, build: "armory",
			mine: fxCity("stable", "garrison", "walls"), theirs: fxCity("palisade", "guard-tower", "barracks", "horse-breeders"), want: 3},
		{name: "merchants guild: the city with more yellow cards", age: 3, build: "merchants-guild",
			mine: fxCity("tavern"), theirs: fxCity("stone-reserve", "clay-reserve", "wood-reserve"), want: 3},
		{name: "merchants guild: own city when it has more", age: 3, build: "merchants-guild",
			mine: fxCity("tavern", "brewery"), theirs: fxCity("stone-reserve"), want: 2},
		{name: "shipowners guild: brown and grey together, from one city", age: 3, build: "shipowners-guild",
			mine: fxCity("lumber-yard", "quarry", "clay-pool"), theirs: fxCity("logging-camp", "glassworks", "press"), want: 3},
		{name: "shipowners guild: the opponent's larger total", age: 3, build: "shipowners-guild",
			mine: fxCity("lumber-yard"), theirs: fxCity("logging-camp", "glassworks", "press", "drying-room"), want: 4},
		{name: "magistrates guild: blue cards", age: 3, build: "magistrates-guild",
			mine: fxCity("theater", "altar"), theirs: fxCity("baths"), want: 2},
		{name: "scientists guild: green cards", age: 3, build: "scientists-guild",
			mine: fxCity("workshop"), theirs: fxCity("scriptorium", "pharmacist"), want: 2},
		{name: "tacticians guild: red cards", age: 3, build: "tacticians-guild",
			mine: fxCity("stable", "garrison"), theirs: fxCity("palisade"), want: 2},
		{name: "builders guild pays no coins", age: 3, build: "builders-guild",
			myW: built("pyramids"), theirW: built("mausoleum", "great-library"), want: 0},
		{name: "moneylenders guild pays no coins", age: 3, build: "moneylenders-guild", want: 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ws := fxWonders
			sc := Scenario{
				Age: c.age, FirstPlayer: fxSeat(0), Wonders: &ws, Accessible: []string{c.build},
				Players: [2]ScenarioPlayer{
					{Coins: fxInt(40), City: c.mine, Built: c.myW},
					{City: c.theirs, Built: c.theirW},
				},
			}
			s := fxNew(t, sc)
			card := cardID(t, c.build)
			coins := s.Players[0].Coins
			q := BuildingQuote(&s, 0, card)
			ev := fxDo(t, &s, "construct:"+c.build)
			if got := fxCardCoins(t, ev, card); got != c.want {
				t.Fatalf("%s paid %d coins, want %d (events %+v)", c.build, got, c.want, ev)
			}
			if s.Players[0].Coins != coins-q.Total+c.want {
				t.Fatalf("purse %d, want %d - %d + %d", s.Players[0].Coins, coins, q.Total, c.want)
			}
		})
	}
}

// TestBuildingEffectOrder checks the order of a building's events: the
// build, the payment, Urbanism's chain bonus, the card's own coins.
func TestBuildingEffectOrder(t *testing.T) {
	ws := fxWonders
	s := fxNew(t, Scenario{
		Age: 3, FirstPlayer: fxSeat(0), Wonders: &ws, Accessible: []string{"lighthouse"},
		Players: [2]ScenarioPlayer{{Coins: fxInt(0), City: fxCity("tavern"), Tokens: []string{"urbanism"}}},
	})
	lighthouse := cardID(t, "lighthouse")
	urbanism := fxToken(t, "urbanism")
	ev := fxDo(t, &s, "construct:lighthouse")
	want := []Event{
		{Kind: EvBuild, Seat: 0, Card: lighthouse},
		{Kind: EvCoins, Seat: 0, Token: urbanism, A: 4, B: 4},
		{Kind: EvCoins, Seat: 0, Card: lighthouse, A: 2, B: 6},
	}
	if !slices.Equal(ev[:len(want)], want) {
		t.Fatalf("events %+v, want %+v first", ev, want)
	}
	if s.Players[0].Coins != 6 {
		t.Fatalf("coins %d, want 6", s.Players[0].Coins)
	}
}

// TestRebuiltBuildingEffects calls resolveBuildingEffects the way the
// Mausoleum does (viaDiscard): every effect applies except Urbanism's bonus.
func TestRebuiltBuildingEffects(t *testing.T) {
	ws := fxWonders
	t.Run("a green card completes a pair", func(t *testing.T) {
		s := fxNew(t, Scenario{
			Age: 2, FirstPlayer: fxSeat(0), Wonders: &ws, Discard: []string{"laboratory"},
			Players: [2]ScenarioPlayer{{City: fxCity("workshop")}},
		})
		lab := cardID(t, "laboratory")
		s.Discard.Remove(lab)
		addCard(&s, 0, lab)
		var ev []Event
		if o := resolveBuildingEffects(&s, 0, lab, viaDiscard, &ev); o != outcomeChoice {
			t.Fatalf("outcome %s, want choice", o)
		}
		if s.Phase != PhasePickBoardToken || s.ToMove != 0 {
			t.Fatalf("phase %s, to move %d", s.Phase, s.ToMove)
		}
		if !slices.Equal(ev, []Event{{Kind: EvPair, Seat: 0, A: int16(catalog.SetSquare)}}) {
			t.Fatalf("events %+v", ev)
		}
	})
	t.Run("a sixth symbol wins", func(t *testing.T) {
		s := fxNew(t, Scenario{
			Age: 3, FirstPlayer: fxSeat(0), Wonders: &ws, Discard: []string{"observatory"},
			Players: [2]ScenarioPlayer{{City: fxCity("scriptorium", "pharmacist", "workshop", "apothecary", "academy")}},
		})
		obs := cardID(t, "observatory")
		s.Discard.Remove(obs)
		addCard(&s, 0, obs)
		if o := resolveBuildingEffects(&s, 0, obs, viaDiscard, nil); o != outcomeOver {
			t.Fatalf("outcome %s, want over", o)
		}
		if s.Result.Kind != VictoryScience || s.Result.Winner != 0 || !s.Over() {
			t.Fatalf("result %+v", s.Result)
		}
	})
	t.Run("no Urbanism bonus, Strategy applies", func(t *testing.T) {
		s := fxNew(t, Scenario{
			Age: 2, FirstPlayer: fxSeat(0), Wonders: &ws, Discard: []string{"horse-breeders"},
			Players: [2]ScenarioPlayer{{Coins: fxInt(5), City: fxCity("stable"), Tokens: []string{"urbanism", "strategy"}}},
		})
		hb := cardID(t, "horse-breeders")
		s.Discard.Remove(hb)
		addCard(&s, 0, hb)
		var ev []Event
		if o := resolveBuildingEffects(&s, 0, hb, viaDiscard, &ev); o != outcomeFinish {
			t.Fatalf("outcome %s", o)
		}
		if s.Players[0].Coins != 5 || len(fxOfKind(ev, EvCoins)) != 0 {
			t.Fatalf("coins %d, events %+v: a rebuilt card paid Urbanism", s.Players[0].Coins, ev)
		}
		if s.Military != 2 {
			t.Fatalf("pawn %d, want 2 (1 shield and Strategy)", s.Military)
		}
	})
	t.Run("the same card through its chain pays Urbanism", func(t *testing.T) {
		s := fxNew(t, Scenario{
			Age: 2, FirstPlayer: fxSeat(0), Wonders: &ws, Accessible: []string{"horse-breeders"},
			Players: [2]ScenarioPlayer{{Coins: fxInt(5), City: fxCity("stable"), Tokens: []string{"urbanism"}}},
		})
		ev := fxDo(t, &s, "construct:horse-breeders")
		if s.Players[0].Coins != 9 || len(fxOfKind(ev, EvCoins)) != 1 {
			t.Fatalf("coins %d, events %+v", s.Players[0].Coins, ev)
		}
	})
}

func TestWonderHelpers(t *testing.T) {
	ws := fxWonders
	s := fxNew(t, Scenario{
		Age: 1, FirstPlayer: fxSeat(0), Wonders: &ws,
		Players: [2]ScenarioPlayer{{Coins: fxInt(1), Tokens: []string{"strategy"}}},
	})
	var ev []Event
	if o := wonderShields(&s, 0, catalog.WonderColossus, &ev); o != outcomeFinish || s.Military != 2 {
		t.Fatalf("the colossus with Strategy: outcome %s, pawn %d, want 2 (no extra shield for a wonder)", o, s.Military)
	}
	if o := wonderShields(&s, 0, catalog.WonderPyramids, &ev); o != outcomeFinish || s.Military != 2 {
		t.Fatalf("the pyramids moved the pawn to %d", s.Military)
	}
	wonderCoins(&s, 0, catalog.WonderTempleOfArtemis, &ev)
	wonderCoins(&s, 0, catalog.WonderSphinx, &ev)
	want := []Event{
		{Kind: EvPawn, Seat: 0, A: 0, B: 2},
		{Kind: EvCoins, Seat: 0, Wonder: catalog.WonderTempleOfArtemis, A: 12, B: 13},
	}
	if !slices.Equal(ev, want) {
		t.Fatalf("events %+v, want %+v", ev, want)
	}
	if wondersPlayAgain(&s.Players[0]) || wondersPlayAgain(&s.Players[1]) {
		t.Fatal("Theology is reported without the token")
	}
}

func TestCoinHelpers(t *testing.T) {
	s := fxNew(t, Scenario{Players: [2]ScenarioPlayer{{Coins: fxInt(3)}, {Coins: fxInt(0)}}})
	var ev []Event
	awardCoins(&s, 0, 0, fromCard(17), &ev)
	awardCoins(&s, 0, -2, fromCard(17), &ev)
	if len(ev) != 0 || s.Players[0].Coins != 3 {
		t.Fatalf("a zero or negative award: coins %d, events %+v", s.Players[0].Coins, ev)
	}
	if lost := loseCoins(&s, 0, 2, fromWonder(catalog.WonderAppianWay), &ev); lost != 2 || s.Players[0].Coins != 1 {
		t.Fatalf("lost %d, coins %d", lost, s.Players[0].Coins)
	}
	if lost := loseCoins(&s, 0, 3, fromWonder(catalog.WonderAppianWay), &ev); lost != 1 || s.Players[0].Coins != 0 {
		t.Fatalf("lost %d, coins %d: the loss is not floored at 0", lost, s.Players[0].Coins)
	}
	if lost := loseCoins(&s, 1, 3, fromWonder(catalog.WonderAppianWay), &ev); lost != 0 || s.Players[1].Coins != 0 {
		t.Fatalf("a player with no coins lost %d", lost)
	}
	if lost := takeCoins(&s, 0, -4); lost != 0 || s.Players[0].Coins != 0 {
		t.Fatalf("a negative loss changed the purse: %d", lost)
	}
	want := []Event{
		{Kind: EvCoins, Seat: 0, Wonder: catalog.WonderAppianWay, A: -2, B: 1},
		{Kind: EvCoins, Seat: 0, Wonder: catalog.WonderAppianWay, A: -1, B: 0},
	}
	if !slices.Equal(ev, want) {
		t.Fatalf("events %+v, want %+v", ev, want)
	}
	s.Players[1].Coins = 32760
	awardCoins(&s, 1, 100, fromToken(1), nil)
	if s.Players[1].Coins != maxInt16 {
		t.Fatalf("coins %d: the award does not saturate", s.Players[1].Coins)
	}
}

func TestMovePawnEdges(t *testing.T) {
	t.Run("zero shields", func(t *testing.T) {
		s := fxNew(t, Scenario{})
		var ev []Event
		if o := movePawn(&s, 0, 0, &ev); o != outcomeFinish || len(ev) != 0 || s.Military != 0 {
			t.Fatalf("outcome %s, events %+v, pawn %d", o, ev, s.Military)
		}
	})
	t.Run("a huge move takes both tokens and stops at the capital", func(t *testing.T) {
		s := fxNew(t, Scenario{Players: [2]ScenarioPlayer{{}, {Coins: fxInt(6)}}})
		var ev []Event
		if o := movePawn(&s, 0, 255, &ev); o != outcomeOver {
			t.Fatalf("outcome %s", o)
		}
		want := []Event{
			{Kind: EvPawn, Seat: 0, A: 0, B: 9},
			{Kind: EvLoot, Seat: 1, A: 2, B: 2},
			{Kind: EvLoot, Seat: 1, A: 4, B: 5},
			{Kind: EvGameOver, A: int16(VictoryMilitary), B: 0},
		}
		if !slices.Equal(ev, want) {
			t.Fatalf("events %+v, want %+v", ev, want)
		}
		if s.Military != 9 || s.Looting != LootMinus3|LootMinus6 || s.Players[1].Coins != 0 {
			t.Fatalf("pawn %d, looting %#x, coins %d", s.Military, s.Looting, s.Players[1].Coins)
		}
		if err := CheckInvariants(&s); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("the other side's tokens stay when the pawn comes back", func(t *testing.T) {
		s := fxNew(t, Scenario{Pawn: fxInt(-7), Players: [2]ScenarioPlayer{{Coins: fxInt(9)}, {Coins: fxInt(9)}}})
		var ev []Event
		if o := movePawn(&s, 0, 12, &ev); o != outcomeFinish {
			t.Fatalf("outcome %s", o)
		}
		if s.Military != 5 || s.Looting != LootPlus6 || s.Players[0].Coins != 9 || s.Players[1].Coins != 7 {
			t.Fatalf("pawn %d, looting %#x, coins %d and %d", s.Military, s.Looting, s.Players[0].Coins, s.Players[1].Coins)
		}
	})
}

// TestPairOncePerSymbol checks that only the second copy of a symbol is a
// pair. In the base game a symbol is on two cards only; a count above 2 is
// forced here to show that it would not pair again.
func TestPairOncePerSymbol(t *testing.T) {
	s := fxNew(t, Scenario{Age: 2})
	p := &s.Players[0]
	for n, want := range []outcome{1: outcomeFinish, 2: outcomeChoice, 3: outcomeFinish} {
		if n == 0 {
			continue
		}
		x := s
		x.Players[0].Sci[catalog.Quill] = uint8(n)
		var ev []Event
		if got := gainSymbol(&x, 0, catalog.Quill, 13, &ev); got != want {
			t.Errorf("count %d: outcome %s, want %s", n, got, want)
		}
		if pairs := len(fxOfKind(ev, EvPair)); (pairs == 1) != (n == 2) {
			t.Errorf("count %d: %d pair events", n, pairs)
		}
	}
	if p.Sci[catalog.Quill] != 0 {
		t.Fatal("the test changed its base state")
	}
}

func TestCheckSupremacy(t *testing.T) {
	s := fxNew(t, Scenario{Age: 2})
	if checkSupremacy(&s, nil) || s.Over() {
		t.Fatal("a supremacy in a fresh age")
	}
	cases := []struct {
		name   string
		set    func(s *State)
		kind   VictoryKind
		winner int8
	}{
		{"the pawn on seat 1's capital", func(s *State) { s.Military = 9 }, VictoryMilitary, 0},
		{"the pawn on seat 0's capital", func(s *State) { s.Military = -9 }, VictoryMilitary, 1},
		{"six symbols for the active seat", func(s *State) {
			for sym := catalog.Globe; sym <= catalog.Quill; sym++ { // globe to quill: six symbols
				s.Players[s.Active].Sci[sym] = 1
			}
		}, VictoryScience, 0},
		{"six symbols for the other seat", func(s *State) {
			for sym := catalog.Scales; sym < catalog.NumSymbols; sym++ {
				s.Players[s.Active.Opp()].Sci[sym] = 1
			}
		}, VictoryScience, 1},
	}
	for _, c := range cases {
		x := s
		x.Active, x.ToMove = 0, 0
		x.ExtraTurn = true
		c.set(&x)
		var ev []Event
		if !checkSupremacy(&x, &ev) {
			t.Fatalf("%s: no supremacy", c.name)
		}
		if x.Result.Kind != c.kind || x.Result.Winner != c.winner || x.Phase != PhaseGameOver || x.ExtraTurn {
			t.Fatalf("%s: result %+v, phase %s, extra turn %v", c.name, x.Result, x.Phase, x.ExtraTurn)
		}
		if x.Result.Scores != Scores(&x) {
			t.Fatalf("%s: the result's scores are not Scores", c.name)
		}
		if !slices.Equal(ev, []Event{{Kind: EvGameOver, A: int16(c.kind), B: int16(c.winner)}}) {
			t.Fatalf("%s: events %+v", c.name, ev)
		}
		ev = ev[:0]
		if !checkSupremacy(&x, &ev) || len(ev) != 0 || x.Result.Kind != c.kind {
			t.Fatalf("%s: a second check changed the finished game: %+v", c.name, ev)
		}
	}
}
