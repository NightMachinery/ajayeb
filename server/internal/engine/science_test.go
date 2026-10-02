package engine_test

import (
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

// Tests of science through the public API: pairs and the board token choice
// they open, a pair with an empty board, the symbols that do not pair, and
// scientific supremacy from a card, from Law and on the last card of the
// game.

// scienceFive is a city with five different symbols (quill, mortar, set
// square, wheel, sundial).
var scienceFive = []string{"scriptorium", "pharmacist", "workshop", "apothecary", "academy"}

func TestSciencePairOpensTheBoardChoice(t *testing.T) {
	s := enginetest.New(t, 31).Age(2).FirstPlayer(0).Coins(0, 20).City(0, "workshop").
		BoardTokens("agriculture", "economy", "law", "philosophy", "theology").Accessible("laboratory").State()
	lab := cardID(t, "laboratory")
	ev := enginetest.Do(t, &s, "construct laboratory")
	want := []engine.Event{
		{Kind: engine.EvBuild, Seat: 0, Card: lab},
		{Kind: engine.EvPay, Seat: 0, A: 6},
		{Kind: engine.EvPair, Seat: 0, A: int16(catalog.SetSquare)},
	}
	if !slices.Equal(ev, want) {
		t.Fatalf("events %+v, want %+v (no reveal before the choice)", ev, want)
	}
	if s.Phase != engine.PhasePickBoardToken || s.ToMove != 0 || s.Active != 0 {
		t.Fatalf("phase %s, to move %d, active %d", s.Phase, s.ToMove, s.Active)
	}
	keys := []string{"token:agriculture", "token:economy", "token:law", "token:philosophy", "token:theology"}
	if got := enginetest.Keys(&s); !slices.Equal(got, keys) {
		t.Fatalf("legal %v, want %v", got, keys)
	}
	if r := enginetest.MustFail(t, &s, "discard "+s.Slots[firstAccessible(&s)].Card.String()); r != engine.ReasonWrongPhase {
		t.Fatalf("a discard during the choice: reason %s", r)
	}
	if r := enginetest.MustFail(t, &s, "token masonry"); r != engine.ReasonTokenUnavailable {
		t.Fatalf("a token not on the board: reason %s", r)
	}
	philosophy, _ := catalog.TokenBySlug("philosophy")
	ev = enginetest.Do(t, &s, "token philosophy")
	if ev[0] != (engine.Event{Kind: engine.EvToken, Seat: 0, Token: philosophy}) {
		t.Fatalf("events %+v", ev)
	}
	if !s.Players[0].Tokens.Has(philosophy) || s.BoardTokens.Has(philosophy) || s.BoardTokens.Count() != 4 {
		t.Fatalf("tokens %v, board %v", s.Players[0].Tokens, s.BoardTokens)
	}
	if s.Phase != engine.PhasePlay || s.ToMove != 1 {
		t.Fatalf("after the pick: phase %s, seat %d to move", s.Phase, s.ToMove)
	}
}

// TestSciencePairThroughAChain: a green card built free through its chain
// completes a pair like any other.
func TestSciencePairThroughAChain(t *testing.T) {
	s := enginetest.New(t, 32).Age(2).FirstPlayer(0).Coins(0, 0).City(0, "pharmacist").Accessible("dispensary").State()
	ev := enginetest.Do(t, &s, "construct dispensary")
	if s.Phase != engine.PhasePickBoardToken || !slices.Contains(ev, engine.Event{Kind: engine.EvPair, Seat: 0, A: int16(catalog.Mortar)}) {
		t.Fatalf("phase %s, events %+v", s.Phase, ev)
	}
}

func TestScienceNoPair(t *testing.T) {
	cases := []struct {
		name  string
		mine  []string
		their []string
		build string
	}{
		{"the other card of the symbol is the opponent's", nil, []string{"workshop"}, "laboratory"},
		{"a first card of a symbol", []string{"workshop", "scriptorium"}, nil, "school"},
		{"a pair made earlier does not fire again", []string{"scriptorium", "library"}, nil, "school"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := enginetest.New(t, 33).Age(2).FirstPlayer(0).Coins(0, 20).City(0, c.mine...).City(1, c.their...).
				Accessible(c.build).State()
			ev := enginetest.Do(t, &s, "construct "+c.build)
			for _, e := range ev {
				if e.Kind == engine.EvPair {
					t.Fatalf("a pair: %+v", ev)
				}
			}
			if s.Phase != engine.PhasePlay || s.ToMove != 1 {
				t.Fatalf("phase %s, seat %d to move", s.Phase, s.ToMove)
			}
		})
	}
}

// TestSciencePairWithAnEmptyBoard: with no token left on the board, the pair
// gives nothing and the turn goes on.
func TestSciencePairWithAnEmptyBoard(t *testing.T) {
	s := enginetest.New(t, 34).Age(2).FirstPlayer(0).Coins(0, 20).City(0, "workshop").Accessible("laboratory").
		Tokens(1, "mathematics", "philosophy", "strategy", "theology", "urbanism").
		BoxOrder("agriculture", "architecture", "economy", "law", "masonry").State()
	if !s.BoardTokens.Empty() {
		t.Fatalf("the board holds %v", s.BoardTokens)
	}
	lab := cardID(t, "laboratory")
	ev := enginetest.Do(t, &s, "construct laboratory")
	want := []engine.Event{
		{Kind: engine.EvPair, Seat: 0, A: int16(catalog.SetSquare)},
		{Kind: engine.EvEffectSkipped, Seat: 0, Card: lab, A: int16(engine.PhasePickBoardToken)},
	}
	if !slices.Equal(ev[2:4], want) {
		t.Fatalf("events %+v, want %+v after the build and the payment", ev, want)
	}
	if s.Phase != engine.PhasePlay || s.ToMove != 1 {
		t.Fatalf("phase %s, seat %d to move", s.Phase, s.ToMove)
	}
}

func TestScienceSupremacy(t *testing.T) {
	cases := []struct {
		name   string
		seat   engine.Seat
		city   []string
		tokens []string
		build  string
	}{
		{"a sixth symbol from a card", 0, scienceFive, nil, "observatory"},
		{"a sixth symbol for seat 1", 1, scienceFive, nil, "university"},
		{"Law is one of the six", 0, scienceFive[:4], []string{"law"}, "academy"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := enginetest.New(t, 35).Age(3).ToMove(c.seat).Coins(c.seat, 30).City(c.seat, c.city...).
				Tokens(c.seat, c.tokens...).Accessible(c.build).State()
			if engine.DistinctSymbols(&s.Players[c.seat]) != 5 {
				t.Fatalf("%d symbols before the build", engine.DistinctSymbols(&s.Players[c.seat]))
			}
			ev := enginetest.Do(t, &s, "construct "+c.build)
			if !s.Over() || s.Result.Kind != engine.VictoryScience || s.Result.Winner != int8(c.seat) {
				t.Fatalf("phase %s, result %+v", s.Phase, s.Result)
			}
			if ev[len(ev)-1] != (engine.Event{Kind: engine.EvGameOver, A: int16(engine.VictoryScience), B: int16(c.seat)}) {
				t.Fatalf("events %+v", ev)
			}
			if s.Result.Scores != engine.Scores(&s) || engine.DistinctSymbols(&s.Players[c.seat]) != 6 {
				t.Fatalf("scores %+v", s.Result.Scores)
			}
		})
	}
	// Five symbols are not enough.
	s := enginetest.New(t, 35).Age(3).ToMove(0).Coins(0, 30).City(0, scienceFive[:4]...).Accessible("observatory").State()
	enginetest.Do(t, &s, "construct observatory")
	if s.Over() || engine.DistinctSymbols(&s.Players[0]) != 5 {
		t.Fatalf("five symbols: phase %s", s.Phase)
	}
}

// TestScienceSupremacyFromLawAfterAPair: the pair's token choice can give
// Law, the sixth symbol.
func TestScienceSupremacyFromLawAfterAPair(t *testing.T) {
	s := enginetest.New(t, 36).Age(3).ToMove(1).Coins(1, 30).City(1, scienceFive...).
		BoardTokens("law", "agriculture", "economy", "masonry", "urbanism").Accessible("study").State()
	ev := enginetest.Do(t, &s, "construct study")
	if s.Phase != engine.PhasePickBoardToken || !slices.Contains(ev, engine.Event{Kind: engine.EvPair, Seat: 1, A: int16(catalog.Sundial)}) {
		t.Fatalf("phase %s, events %+v", s.Phase, ev)
	}
	law, _ := catalog.TokenBySlug("law")
	ev = enginetest.Do(t, &s, "token law")
	want := []engine.Event{
		{Kind: engine.EvToken, Seat: 1, Token: law},
		{Kind: engine.EvGameOver, A: int16(engine.VictoryScience), B: 1},
	}
	if !slices.Equal(ev, want) {
		t.Fatalf("events %+v, want %+v", ev, want)
	}
	if !s.Over() || s.Result.Winner != 1 || s.Result.Scores != engine.Scores(&s) {
		t.Fatalf("result %+v", s.Result)
	}
}

// TestScienceSupremacyOnTheLastCard: a science supremacy on the last card of
// Age III wins without civilian scoring.
func TestScienceSupremacyOnTheLastCard(t *testing.T) {
	s := enginetest.New(t, 37).Age(3).ToMove(0).Coins(0, 30).City(0, scienceFive...).OnlyLeft("observatory").State()
	ev := enginetest.Do(t, &s, "construct observatory")
	if s.Result.Kind != engine.VictoryScience || s.Result.Winner != 0 {
		t.Fatalf("result %+v, events %+v", s.Result, ev)
	}
}

func TestDistinctSymbols(t *testing.T) {
	s := enginetest.New(t, 38).Age(2).City(0, "scriptorium", "library", "workshop").Tokens(0, "law").State()
	if got := engine.DistinctSymbols(&s.Players[0]); got != 3 {
		t.Fatalf("quill twice, set square and Law's scales: %d symbols, want 3", got)
	}
	if got := engine.DistinctSymbols(&s.Players[1]); got != 0 {
		t.Fatalf("an empty city has %d symbols", got)
	}
}
