package engine

import (
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// The ten per-token tests: each progress token's immediate and lasting
// effects, through Apply where a flow reaches them and through the hooks
// where only the wonder code (slice S5b) will (Theology, the Great Library
// pick). Helpers come from effects_test.go.

// fxAtPairChoice returns a game in Age II in which seat 0 has just built the
// Laboratory next to its Workshop, completing the set-square pair, and must
// now take one of the board tokens named. sc supplies the players; the rest
// of the scenario is filled in.
func fxAtPairChoice(t *testing.T, board []string, sc Scenario) State {
	t.Helper()
	ws := fxWonders
	sc.Age, sc.FirstPlayer, sc.Wonders = 2, fxSeat(0), &ws
	sc.Accessible = append(sc.Accessible, "laboratory")
	sc.BoardTokens = board
	sc.Players[0].City = append(sc.Players[0].City, "workshop")
	if sc.Players[0].Coins == nil {
		sc.Players[0].Coins = fxInt(20)
	}
	s := fxNew(t, sc)
	ev := fxDo(t, &s, "construct:laboratory")
	if s.Phase != PhasePickBoardToken || len(fxOfKind(ev, EvPair)) != 1 {
		t.Fatalf("no pair choice: phase %s, events %+v", s.Phase, ev)
	}
	return s
}

// fxBoard is a full board of five tokens that holds the given ones.
func fxBoard(t *testing.T, want ...string) []string {
	t.Helper()
	board := append([]string(nil), want...)
	for d := range catalog.Tokens() {
		if len(board) == NumBoardTokens {
			break
		}
		if !slices.Contains(board, d.Slug) {
			board = append(board, d.Slug)
		}
	}
	return board
}

func TestTokenAgriculture(t *testing.T) {
	agri := fxToken(t, "agriculture")
	s := fxAtPairChoice(t, fxBoard(t, "agriculture"), Scenario{})
	coins := s.Players[0].Coins
	ev := fxDo(t, &s, "token:agriculture")
	want := []Event{
		{Kind: EvToken, Seat: 0, Token: agri},
		{Kind: EvCoins, Seat: 0, Token: agri, A: 6, B: coins + 6},
	}
	if !slices.Equal(ev[:2], want) {
		t.Fatalf("events %+v, want %+v first", ev, want)
	}
	if !s.Players[0].Tokens.Has(agri) || s.BoardTokens.Has(agri) || s.Players[0].Coins != coins+6 {
		t.Fatalf("tokens %v, board %v, coins %d", s.Players[0].Tokens, s.BoardTokens, s.Players[0].Coins)
	}
	if s.Phase != PhasePlay || s.ToMove != 1 {
		t.Fatalf("after the pick: phase %s, seat %d to move", s.Phase, s.ToMove)
	}
	if got := Scores(&s)[0].Tokens; got != 4 {
		t.Fatalf("Agriculture scores %d, want 4", got)
	}
}

func TestTokenArchitecture(t *testing.T) {
	ws := fxWonders
	quotes := func(tokens ...string) (wonder, building Quote) {
		s := fxNew(t, Scenario{Wonders: &ws, Players: [2]ScenarioPlayer{{Coins: fxInt(20), Tokens: tokens}}})
		return WonderQuote(&s, 0, catalog.WonderPyramids), BuildingQuote(&s, 0, cardID(t, "baths"))
	}
	w0, b0 := quotes()
	w1, b1 := quotes("architecture")
	if w0.Total != 8 || w1.Total != 4 {
		t.Fatalf("the pyramids (SSS P) cost %d, and %d with Architecture; want 8 and 4", w0.Total, w1.Total)
	}
	if b0 != b1 {
		t.Fatalf("Architecture changed a building's cost: %+v and %+v", b0, b1)
	}
}

func TestTokenEconomy(t *testing.T) {
	ws := fxWonders
	build := func(owner int, card string) (State, []Event) {
		sc := Scenario{Age: 2, FirstPlayer: fxSeat(0), Wonders: &ws, Accessible: []string{card},
			Players: [2]ScenarioPlayer{{Coins: fxInt(20)}, {Coins: fxInt(3)}}}
		if owner >= 0 {
			sc.Players[owner].Tokens = []string{"economy"}
		}
		s := fxNew(t, sc)
		return s, fxDo(t, &s, "construct:"+card)
	}
	// The Caravansery: 2 printed coins to the bank, glass and papyrus bought
	// for 2 each.
	s, ev := build(1, "caravansery")
	if s.Players[0].Coins != 14 || s.Players[1].Coins != 7 {
		t.Fatalf("coins %d and %d, want 14 and 7 (the trading part, 4, to the owner of Economy)", s.Players[0].Coins, s.Players[1].Coins)
	}
	if pays := fxOfKind(ev, EvPay); !slices.Equal(pays, []Event{{Kind: EvPay, Seat: 0, A: 6, B: 4}}) {
		t.Fatalf("payment events %+v", pays)
	}
	// Economy of the builder: the bank keeps everything.
	if s, _ := build(0, "caravansery"); s.Players[0].Coins != 14 || s.Players[1].Coins != 3 {
		t.Fatalf("own Economy: coins %d and %d", s.Players[0].Coins, s.Players[1].Coins)
	}
	// Printed coins only: nothing goes to the owner.
	if s, _ := build(1, "customs-house"); s.Players[0].Coins != 16 || s.Players[1].Coins != 3 {
		t.Fatalf("the Customs House: coins %d and %d", s.Players[0].Coins, s.Players[1].Coins)
	}
}

func TestTokenLaw(t *testing.T) {
	law := fxToken(t, "law")
	ws := fxWonders
	s := fxNew(t, Scenario{Age: 2, Wonders: &ws})
	addToken(&s, 0, law)
	if s.Players[0].Sci[catalog.Scales] != 1 || DistinctSymbols(&s.Players[0]) != 1 {
		t.Fatalf("Law gives the science counts %v", s.Players[0].Sci)
	}
	var ev []Event
	if o := resolveTokenEffects(&s, 0, law, &ev); o != outcomeFinish || len(ev) != 0 || s.Over() {
		t.Fatalf("Law alone: outcome %s, events %+v", o, ev)
	}
	// With five card symbols, Law's scales are the sixth.
	s = fxNew(t, Scenario{Age: 3, Wonders: &ws, Players: [2]ScenarioPlayer{{},
		{City: fxCity("scriptorium", "pharmacist", "workshop", "apothecary", "academy")}}})
	addToken(&s, 1, law)
	ev = ev[:0]
	if o := resolveTokenEffects(&s, 1, law, &ev); o != outcomeOver {
		t.Fatalf("the sixth symbol from Law: outcome %s", o)
	}
	if s.Result.Kind != VictoryScience || s.Result.Winner != 1 || len(fxOfKind(ev, EvPair)) != 0 {
		t.Fatalf("result %+v, events %+v", s.Result, ev)
	}
}

func TestTokenMasonry(t *testing.T) {
	ws := fxWonders
	quotes := func(tokens ...string) [3]int16 {
		s := fxNew(t, Scenario{Age: 2, Wonders: &ws, Players: [2]ScenarioPlayer{{Coins: fxInt(20), Tokens: tokens}}})
		return [3]int16{
			BuildingQuote(&s, 0, cardID(t, "courthouse")).Total,     // blue, WW G
			BuildingQuote(&s, 0, cardID(t, "horse-breeders")).Total, // red, C W
			WonderQuote(&s, 0, catalog.WonderPyramids).Total,
		}
	}
	if got, want := quotes(), [3]int16{6, 4, 8}; got != want {
		t.Fatalf("without Masonry: %v, want %v", got, want)
	}
	if got, want := quotes("masonry"), [3]int16{2, 4, 8}; got != want {
		t.Fatalf("with Masonry: %v, want %v (only the blue card is cheaper)", got, want)
	}
}

func TestTokenMathematicsAndPhilosophy(t *testing.T) {
	cases := []struct {
		tokens []string
		want   int16
	}{
		{[]string{"mathematics"}, 3},
		{[]string{"philosophy"}, 7},
		{[]string{"mathematics", "agriculture"}, 6 + 4},
		{[]string{"mathematics", "philosophy", "law"}, 9 + 7},
		{[]string{"economy", "masonry", "strategy", "theology", "urbanism", "architecture", "law"}, 0},
	}
	ws := fxWonders
	for _, c := range cases {
		s := fxNew(t, Scenario{Wonders: &ws, Players: [2]ScenarioPlayer{{Coins: fxInt(0)}, {Coins: fxInt(0), Tokens: c.tokens}}})
		sc := Scores(&s)
		if sc[1].Tokens != c.want || sc[1].Total != c.want || sc[0].Tokens != 0 {
			t.Errorf("tokens %v: score %+v, want %d token points", c.tokens, sc[1], c.want)
		}
	}
}

func TestTokenStrategy(t *testing.T) {
	ws := fxWonders
	game := func(owner Seat, city ...string) State {
		sc := Scenario{FirstPlayer: fxSeat(0), Wonders: &ws, Accessible: []string{"guard-tower", "theater"}}
		sc.Players[owner].Tokens = []string{"strategy"}
		sc.Players[0].City = city
		return fxNew(t, sc)
	}
	s := game(0)
	ev := fxDo(t, &s, "construct:guard-tower")
	if s.Military != 2 || !slices.Equal(fxOfKind(ev, EvPawn), []Event{{Kind: EvPawn, Seat: 0, A: 0, B: 2}}) {
		t.Fatalf("a red building with Strategy: pawn %d, events %+v", s.Military, ev)
	}
	s = game(0)
	fxDo(t, &s, "construct:theater")
	if s.Military != 0 {
		t.Fatalf("a blue building with Strategy moved the pawn to %d", s.Military)
	}
	s = game(1)
	fxDo(t, &s, "construct:guard-tower")
	if s.Military != 1 {
		t.Fatalf("the opponent's Strategy: pawn %d, want 1", s.Military)
	}
	// Taking Strategy does not reach back to the red buildings already built.
	s = fxNew(t, Scenario{Wonders: &ws, Players: [2]ScenarioPlayer{{City: fxCity("stable", "garrison")}}})
	strategy := fxToken(t, "strategy")
	addToken(&s, 0, strategy)
	ev = ev[:0]
	if o := resolveTokenEffects(&s, 0, strategy, &ev); o != outcomeFinish || len(ev) != 0 || s.Military != 0 {
		t.Fatalf("taking Strategy: outcome %s, events %+v, pawn %d", o, ev, s.Military)
	}
	// Nor does it add to a wonder's shields.
	wonderShields(&s, 0, catalog.WonderColossus, nil)
	if s.Military != 2 {
		t.Fatalf("the Colossus with Strategy: pawn %d, want 2", s.Military)
	}
}

func TestTokenTheology(t *testing.T) {
	ws := fxWonders
	s := fxNew(t, Scenario{Wonders: &ws, Players: [2]ScenarioPlayer{{}, {Tokens: []string{"theology"}}}})
	if !wondersPlayAgain(&s.Players[1]) || wondersPlayAgain(&s.Players[0]) {
		t.Fatal("wondersPlayAgain does not follow Theology")
	}
	before := s
	var ev []Event
	if o := resolveTokenEffects(&s, 1, fxToken(t, "theology"), &ev); o != outcomeFinish || len(ev) != 0 || s != before {
		t.Fatalf("taking Theology: outcome %s, events %+v", o, ev)
	}
}

func TestTokenUrbanism(t *testing.T) {
	urbanism := fxToken(t, "urbanism")
	t.Run("taken from the board", func(t *testing.T) {
		s := fxAtPairChoice(t, fxBoard(t, "urbanism"), Scenario{})
		coins := s.Players[0].Coins
		ev := fxDo(t, &s, "token:urbanism")
		if s.Players[0].Coins != coins+6 || !slices.Contains(ev, Event{Kind: EvCoins, Seat: 0, Token: urbanism, A: 6, B: coins + 6}) {
			t.Fatalf("coins %d (from %d), events %+v", s.Players[0].Coins, coins, ev)
		}
	})
	ws := fxWonders
	build := func(owner Seat, card string, city ...string) (State, []Event) {
		sc := Scenario{Age: 2, FirstPlayer: fxSeat(0), Wonders: &ws, Accessible: []string{card}}
		sc.Players[owner].Tokens = []string{"urbanism"}
		sc.Players[0].City = city
		sc.Players[0].Coins = fxInt(10)
		s := fxNew(t, sc)
		return s, fxDo(t, &s, "construct:"+card)
	}
	t.Run("a chain build pays 4", func(t *testing.T) {
		s, ev := build(0, "statue", "theater")
		if s.Players[0].Coins != 14 || !slices.Contains(ev, Event{Kind: EvCoins, Seat: 0, Token: urbanism, A: 4, B: 14}) {
			t.Fatalf("coins %d, events %+v", s.Players[0].Coins, ev)
		}
	})
	t.Run("a free card that is not chained pays nothing", func(t *testing.T) {
		if s, ev := build(0, "brewery"); s.Players[0].Coins != 16 || len(fxOfKind(ev, EvCoins)) != 1 {
			t.Fatalf("coins %d (the brewery's 6 only), events %+v", s.Players[0].Coins, ev)
		}
	})
	t.Run("a bought card pays nothing", func(t *testing.T) {
		if s, ev := build(0, "statue"); s.Players[0].Coins != 6 || len(fxOfKind(ev, EvCoins)) != 0 {
			t.Fatalf("coins %d, events %+v", s.Players[0].Coins, ev)
		}
	})
	t.Run("the opponent's Urbanism pays the builder nothing", func(t *testing.T) {
		if s, ev := build(1, "statue", "theater"); s.Players[0].Coins != 10 || s.Players[1].Coins != StartCoins || len(fxOfKind(ev, EvCoins)) != 0 {
			t.Fatalf("coins %d and %d, events %+v", s.Players[0].Coins, s.Players[1].Coins, ev)
		}
	})
	// The Mausoleum's free build is not a chain: TestRebuiltBuildingEffects.
}

// TestTokenImmediateEffects takes each token directly and checks its
// immediate coins, its points, and that nothing else changes.
func TestTokenImmediateEffects(t *testing.T) {
	want := map[string]struct{ coins, points int16 }{
		"agriculture": {6, 4}, "architecture": {0, 0}, "economy": {0, 0}, "law": {0, 0}, "masonry": {0, 0},
		"mathematics": {0, 3}, "philosophy": {0, 7}, "strategy": {0, 0}, "theology": {0, 0}, "urbanism": {6, 0},
	}
	ws := fxWonders
	seen := 0
	for d := range catalog.Tokens() {
		w, ok := want[d.Slug]
		if !ok {
			t.Fatalf("token %s has no expectation", d.Slug)
		}
		seen++
		s := fxNew(t, Scenario{Age: 2, Wonders: &ws, Players: [2]ScenarioPlayer{{Coins: fxInt(5)}}})
		s.BoardTokens.Remove(d.ID)
		addToken(&s, 0, d.ID)
		before := s
		if o := resolveTokenEffects(&s, 0, d.ID, nil); o != outcomeFinish {
			t.Fatalf("%s: outcome %s", d.Slug, o)
		}
		if s.Players[0].Coins != 5+w.coins {
			t.Errorf("%s: coins %d, want %d", d.Slug, s.Players[0].Coins, 5+w.coins)
		}
		s.Players[0].Coins = before.Players[0].Coins
		if s != before {
			t.Errorf("%s: taking it changed more than the coins", d.Slug)
		}
		if got := Scores(&s)[0].Tokens; got != w.points {
			t.Errorf("%s: %d token points, want %d", d.Slug, got, w.points)
		}
	}
	if seen != catalog.NumTokens {
		t.Fatalf("%d tokens, want %d", seen, catalog.NumTokens)
	}
}
