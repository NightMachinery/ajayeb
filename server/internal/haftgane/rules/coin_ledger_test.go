package rules

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// Literal formulas from rules.md, independent of Effects, Colors caches,
// scopeCount and applyWhenPlaced. A discard build has the same income as an
// ordinary building, but reads the later position at its own point (ruling 28).
func referenceIncome(st *State, seat Seat, a Action) int32 {
	c := &st.Cities[seat]
	count := func(s Seat, color catalog.Color) int32 {
		var n int32
		for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
			if st.Cities[s].Buildings.Has(id) {
				d, _ := catalog.DesignByID(id)
				if d.Color == color {
					n++
				}
			}
		}
		return n
	}
	left, right := Seat((uint8(seat)+1)%st.Seats), Seat((uint8(seat)+st.Seats-1)%st.Seats)
	switch a.Kind {
	case ActBuild, ActBuildDiscarded:
		switch a.Design {
		case 23:
			return 5 // Tavern
		case 43:
			return 2 * (count(seat, catalog.Grey) + count(left, catalog.Grey) + count(right, catalog.Grey)) // Bazar
		case 46:
			return count(seat, catalog.Brown) + count(left, catalog.Brown) + count(right, catalog.Brown) // Vineyard
		case 61:
			return 3 * int32(c.Stages) // Arena
		case 62:
			return 2 * count(seat, catalog.Grey) // Chamber of Commerce
		case 63:
			return count(seat, catalog.Brown) // Haven
		case 64:
			return count(seat, catalog.Yellow) // Lighthouse, including itself
		case 65:
			return 3 * count(seat, catalog.Red) // Ludus
		}
	case ActStage:
		switch {
		case c.Board == catalog.BoardEphesos && c.Side == catalog.Day && c.Stages == 2:
			return 9
		case c.Board == catalog.BoardEphesos && c.Side == catalog.Night:
			return 4
		case c.Board == catalog.BoardRhodos && c.Side == catalog.Night && c.Stages == 1:
			return 3
		case c.Board == catalog.BoardRhodos && c.Side == catalog.Night && c.Stages == 2:
			return 4
		}
	}
	return 0
}

// Check every seat, including nonacting trade recipients. EvPlay supplies only
// the public payment amounts; income and sales are predicted from the chosen
// actions and raw placed buildings, never borrowed from EvCoins or the result.
// Slot zero of the returned coverage counts means a stage; other slots are ids.
func checkCoinLedger(t *testing.T, before, after *State, picks *[MaxSeats]Action, events []Event) [catalog.NumDesigns + 1]int {
	t.Helper()
	var want, placedIncome, eventIncome [MaxSeats]int32
	var fired [catalog.NumDesigns + 1]int
	for seat := Seat(0); seat < Seat(before.Seats); seat++ {
		want[seat] = int32(before.Cities[seat].Coins)
	}
	for _, e := range events {
		switch e.Kind {
		case EvPlay:
			if !e.Seat.Valid(before.Seats) || picks[e.Seat].Kind != e.Action || picks[e.Seat].Design != e.Design {
				t.Fatal("payment event disagrees with selected action", e)
			}
			want[e.Seat] -= int32(e.Printed) + int32(e.Left) + int32(e.Right)
			want[(uint8(e.Seat)+1)%before.Seats] += int32(e.Left)
			want[(uint8(e.Seat)+before.Seats-1)%before.Seats] += int32(e.Right)
		case EvCoins:
			if !e.Seat.Valid(before.Seats) {
				t.Fatal("income event outside table", e)
			}
			eventIncome[e.Seat] += int32(e.Delta)
		}
	}
	for seat := Seat(0); seat < Seat(before.Seats); seat++ {
		a := picks[seat]
		income := referenceIncome(after, seat, a)
		placedIncome[seat] = income
		if income != 0 {
			if a.Kind == ActStage {
				fired[0]++
			} else {
				fired[a.Design]++
			}
		}
		if a.Kind == ActSell {
			placedIncome[seat] += 3
		}
		want[seat] += placedIncome[seat]
		if got := int32(after.Cities[seat].Coins); got != want[seat] {
			t.Fatalf("coin ledger age%d turn%d phase%s seat%d: got%d want%d (income%d, action%+v)", before.Age, before.Turn, before.Phase, seat, got, want[seat], placedIncome[seat], a)
		}
		if eventIncome[seat] != placedIncome[seat] {
			t.Fatalf("income event ledger age%d turn%d phase%s seat%d: got%d want%d", before.Age, before.Turn, before.Phase, seat, eventIncome[seat], placedIncome[seat])
		}
	}
	return fired
}
