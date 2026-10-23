package rules

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

var legalChoicesSink []Choice
var resolvedStateSink State
var resolvedEventsSink []Event

func benchmarkTurn(b *testing.B) Game {
	b.Helper()
	g, err := NewGame(Setup{Seats: 7, Seed: 42, Options: Options{Sides: SidesDay}})
	if err != nil {
		b.Fatal(err)
	}
	return g
}

func BenchmarkLegalSevenTypical(b *testing.B) {
	g := benchmarkTurn(b)
	var buf [MaxTurnChoices]Choice
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for seat := Seat(0); seat < 7; seat++ {
			legalChoicesSink = LegalState(&g.State, seat, buf[:0])
		}
	}
}

func BenchmarkGameLegalSevenTypical(b *testing.B) {
	g := benchmarkTurn(b)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for seat := Seat(0); seat < 7; seat++ {
			legalChoicesSink = g.Legal(seat)
		}
	}
}

func benchmarkWorstTurn(b *testing.B) State {
	b.Helper()
	// Deliberate capacity stress, not a reachable-city claim. Seven distinct
	// seven-unit costs plus an eighth stage cost, with maximal flexible caches.
	designs := catalog.BaseDesigns()
	var hand [HandSize]catalog.DesignID
	for i := 0; i < HandSize; i++ {
		id := catalog.DesignID(51 + i)
		hand[i] = id
		cost := catalog.Cost{Res: [catalog.NumResources]uint8{1, 1, 1, 1, 1, 1, 1}}
		if i > 0 {
			cost.Res[i-1] = 0
			cost.Res[i] = 2
		}
		designs[id-1].Cost = cost
	}
	boards := catalog.BaseBoards()
	for i := range boards {
		boards[i].Sides[0].Stages[0].Cost = catalog.MustCost("WWWWWWW")
	}
	b.Cleanup(catalog.ReplaceForTest(designs, boards))
	st := State{Seats: 7, Phase: PhaseTurn, Age: 3, Turn: 1, Decider: NoSeat}
	for s := 0; s < 7; s++ {
		st.Hands[s] = hand
		c := City{Board: catalog.BoardID(s + 1), Side: catalog.Day, Coins: 30, NChoice: 3, NChoiceForSale: 6}
		for i := uint8(0); i < c.NChoice; i++ {
			c.Choice[i] = catalog.AllResources
		}
		for i := uint8(0); i < c.NChoiceForSale; i++ {
			c.ChoiceForSale[i] = catalog.AllResources
		}
		for r := range c.Discount {
			if s%2 == r {
				c.Discount[r] = catalog.Raw
			}
		}
		st.Cities[s] = c
	}
	return st
}

func BenchmarkLegalSevenWorst(b *testing.B) {
	st := benchmarkWorstTurn(b)
	var buf [MaxTurnChoices]Choice
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for seat := Seat(0); seat < 7; seat++ {
			legalChoicesSink = LegalState(&st, seat, buf[:0])
		}
	}
}

func BenchmarkGameLegalSevenWorst(b *testing.B) {
	g := Game{State: benchmarkWorstTurn(b)}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for seat := Seat(0); seat < 7; seat++ {
			legalChoicesSink = g.Legal(seat)
		}
	}
}

func BenchmarkResolveSeven(b *testing.B) {
	g := benchmarkTurn(b)
	var picks [MaxSeats]Action
	for s := Seat(0); s < 7; s++ {
		cs := g.Legal(s)
		picks[s] = cs[0].Action
	}
	var buf [32]Event
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		st, evs, err := ResolveState(&g.State, &picks, buf[:0])
		if err != nil {
			b.Fatal(err)
		}
		resolvedStateSink, resolvedEventsSink = st, evs
	}
}
