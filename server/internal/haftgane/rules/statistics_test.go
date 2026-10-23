package rules

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// Section 10's notebook: bounds concern remembered packets, not an estimate of
// the unseen deck. These observations are hand-written, not engine playouts.
func notebookExample(age uint8) Observation {
	o := Observation{Now: Current{Seat: 0, Seats: 3, Phase: PhaseTurn, Age: age, Turn: 3}}
	o.Know.Seen[age-1][0] = [HandSize]catalog.DesignID{1, 2, 7, 8, 11, 17, 19}
	o.Know.Picks[age-1][0] = 19
	o.Know.Plays[age-1][0][0] = Play{Kind: ActSell}
	return o
}
func TestNotebookCirculatingBounds(t *testing.T) {
	for _, age := range []uint8{1, 2} {
		o := notebookExample(age)
		next := Seat(0).Left(3)
		if age == 2 {
			next = Seat(0).Right(3)
		}
		o.Know.Plays[age-1][1][next] = Play{Kind: ActBuild, Design: 17}
		s := o.Statistics()
		if s.Circulating[catalog.Brown] != (Bounds{Certain: 4, AtMost: 4}) || s.Circulating[catalog.Grey] != (Bounds{Certain: 1, AtMost: 1}) || s.Circulating[catalog.Blue] != (Bounds{}) {
			t.Fatal("known construction must remove its exact colour", age, s)
		}
		o.Know.Plays[age-1][1][next] = Play{Kind: ActStage, Stage: 1}
		s = o.Statistics()
		if s.Circulating[catalog.Brown] != (Bounds{Certain: 3, AtMost: 4}) || s.Circulating[catalog.Grey] != (Bounds{Certain: 0, AtMost: 1}) || s.Circulating[catalog.Blue] != (Bounds{Certain: 0, AtMost: 1}) {
			t.Fatal("anonymous removal cannot name its colour", age, s)
		}
	}
}
func TestNotebookReturningBounds(t *testing.T) {
	o := notebookExample(1)
	o.Now.Turn = 2
	// The packet was passed to the left on turn 1 and comes back on turn 4.
	s := o.Statistics()
	if s.Returning[1] != (Bounds{AtMost: 1}) || s.Returning[19] != (Bounds{}) {
		t.Fatal("returning bound", s)
	}
	// Identical copies give a certain survivor even after two unknown picks.
	o.Know.Seen[0][0] = [HandSize]catalog.DesignID{1, 1, 1, 1, 17, 19, 23}
	o.Know.Picks[0][0] = 23
	s = o.Statistics()
	if s.Returning[1] != (Bounds{Certain: 2, AtMost: 4}) {
		t.Fatal("certain returning copies", s.Returning[1])
	}
	o.Now.Seats = 6
	if o.Statistics().Returning != ([catalog.NumDesigns + 1]Bounds{}) {
		t.Fatal("six seats cannot return a packet")
	}
	o.Now.Seats, o.Now.Turn = 3, 5
	if o.Statistics().Returning != ([catalog.NumDesigns + 1]Bounds{}) {
		t.Fatal("packet returns after Age end")
	}
}
func TestNotebookNeighbourDeductions(t *testing.T) {
	for _, age := range []uint8{1, 2} {
		o := notebookExample(age)
		o.Now.Turn = 4
		first, second := Seat(0).Left(3), Seat(0).Right(3)
		if age == 2 {
			first, second = second, first
		}
		o.Know.Plays[age-1][1][first] = Play{Kind: ActSell}
		o.Know.Plays[age-1][2][second] = Play{Kind: ActBuild, Design: 17}
		o.Now.Hand = [HandSize]catalog.DesignID{1, 2, 7, 8}
		s := o.Statistics()
		if s.NumDeductions != 1 || s.Deductions[0] != (KnownPick{Age: age, Turn: 2, Seat: first, Design: 11}) {
			t.Fatal("single missing card identifies neighbour pick", age, s)
		}
		// The exact returning hand replaces its earlier uncertain bounds, once.
		if s.Circulating[catalog.Brown] != (Bounds{Certain: 4, AtMost: 4}) {
			t.Fatal("returned packet double counted or not refined", s)
		}
		o.Know.Plays[age-1][2][second] = Play{Kind: ActStage, Stage: 1}
		if o.Statistics().NumDeductions != 0 {
			t.Fatal("two different anonymous cards are ambiguous")
		}
		// If both missing cards have the same design, both picks are certain.
		o.Know.Seen[age-1][0] = [HandSize]catalog.DesignID{1, 2, 7, 8, 11, 11, 19}
		s = o.Statistics()
		if s.NumDeductions != 2 || s.Deductions[0].Design != 11 || s.Deductions[1].Design != 11 {
			t.Fatal("indistinguishable copies should be certain", s)
		}
	}
}
func TestNotebookDoesNotCountPacketTwice(t *testing.T) {
	o := notebookExample(1)
	o.Now.Turn = 5
	o.Know.Seen[0][3] = [HandSize]catalog.DesignID{1, 2, 7, 8}
	o.Know.Picks[0][3] = 8
	o.Know.Plays[0][3][0] = Play{Kind: ActSell}
	o.Know.Plays[0][1][1] = Play{Kind: ActSell}
	o.Know.Plays[0][2][2] = Play{Kind: ActBuild, Design: 17}
	s := o.Statistics()
	if s.Circulating[catalog.Brown] != (Bounds{Certain: 3, AtMost: 3}) {
		t.Fatal("latest sighting must replace same packet", s)
	}
	o.Now.Phase = PhaseLastCard
	if s := o.Statistics(); s.Circulating != ([catalog.NumColors]Bounds{}) || s.Returning != ([catalog.NumDesigns + 1]Bounds{}) {
		t.Fatal("nothing circulates after turn six")
	}
	o.Now.Phase, o.Now.Seat = PhaseTurn, NoSeat
	if o.Statistics() != (Statistics{}) {
		t.Fatal("spectator has no notebook")
	}
}
