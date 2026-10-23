package rules

import (
	"fmt"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// scopeCount sums selected cities, never the maximum. At least three seats
// makes self, left and right distinct. Counting only public placements makes
// when-placed income independent of the order of other income calculations.
func scopeCount(st *State, seat Seat, scope catalog.SeatScope, colors catalog.ColorMask, stages bool) int16 {
	var count int16
	seats := [3]Seat{seat, seat.Left(st.Seats), seat.Right(st.Seats)}
	scopes := [3]catalog.SeatScope{catalog.Self, catalog.Left, catalog.Right}
	for i, s := range seats {
		if scope&scopes[i] == 0 {
			continue
		}
		c := &st.Cities[s]
		if stages {
			count += int16(c.Stages)
			continue
		}
		for color := catalog.Brown; color < catalog.NumColors; color++ {
			if colors.Has(color) {
				count += int16(c.Colors[color])
			}
		}
	}
	return count
}

// applyWhenPlaced runs only the selected placement's coin effects, once.
// All cities/caches must already reflect every placement of the turn. It
// returns an aggregated income for a single event; no reserve cap is applied.
func applyWhenPlaced(st *State, seat Seat, effects []catalog.Effect) (int16, error) {
	var income int32
	for _, e := range effects {
		if e.Kind.Timings()&catalog.WhenPlaced == 0 {
			continue
		}
		switch e.Kind {
		case catalog.EffCoins:
			income += int32(e.N)
		case catalog.EffCoinsPerCard:
			income += int32(e.N) * int32(scopeCount(st, seat, e.Scope, e.Colors, false))
		case catalog.EffCoinsPerStage:
			income += int32(e.N) * int32(scopeCount(st, seat, e.Scope, 0, true))
		default:
			panic("rules: unhandled when-placed effect")
		}
	}
	coins := int32(st.Cities[seat].Coins) + income
	if coins < 0 || coins > 32767 {
		return 0, fmt.Errorf("rules: when-placed coin overflow")
	}
	st.Cities[seat].Coins = int16(coins)
	return int16(income), nil
}
