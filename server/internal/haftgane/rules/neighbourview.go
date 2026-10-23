package rules

import (
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/view"
)

// Fixed symbols and shared choice symbols stay distinct. A buyer may choose
// one option of each choice symbol, not purchase each listed option separately.
func neighbourDTO(st *State, viewer Seat) []view.Neighbour {
	out := make([]view.Neighbour, 0, 2)
	for direction, seat := range [2]Seat{viewer.Left(st.Seats), viewer.Right(st.Seats)} {
		c := &st.Cities[seat]
		v := view.Neighbour{Seat: int(seat), Direction: ([2]string{"left", "right"})[direction], Units: []view.TradeUnit{}}
		if st.Age >= 1 && st.Age <= NumAges {
			if st.Cities[viewer].Shields > c.Shields {
				v.Outlook = ([NumAges]int{1, 3, 5})[st.Age-1]
			} else if st.Cities[viewer].Shields < c.Shields {
				v.Outlook = -1
			}
		}
		for r := catalog.Resource(0); r < catalog.NumResources; r++ {
			if c.ForSale[r] > 0 {
				v.Units = append(v.Units, view.TradeUnit{Producer: 0, Count: int(c.ForSale[r]), Options: []view.TradePrice{tradePrice(st, viewer, direction, r)}})
			}
		}
		for producer, mask := range c.ChoiceForSale[:c.NChoiceForSale] {
			unit := view.TradeUnit{Choice: true, Producer: producer, Count: 1, Options: []view.TradePrice{}}
			for r := catalog.Resource(0); r < catalog.NumResources; r++ {
				if mask.Has(r) {
					unit.Options = append(unit.Options, tradePrice(st, viewer, direction, r))
				}
			}
			v.Units = append(v.Units, unit)
		}
		out = append(out, v)
	}
	return out
}
func tradePrice(st *State, seat Seat, direction int, r catalog.Resource) view.TradePrice {
	v := view.TradePrice{Resource: r.String(), Price: 2}
	c := &st.Cities[seat]
	if !c.Discount[direction].Has(r) {
		return v
	}
	v.Price = 1
	scope := ([2]catalog.SeatScope{catalog.Left, catalog.Right})[direction]
	matches := func(e catalog.Effect) bool {
		return e.Kind == catalog.EffTradeDiscount && e.Scope&scope != 0 && e.Mask.Has(r)
	}
	for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
		if !c.Buildings.Has(id) {
			continue
		}
		d, _ := catalog.DesignByID(id)
		for _, e := range d.Effects {
			if matches(e) {
				v.Discount = &view.Discount{Design: d.Slug}
				return v
			}
		}
	}
	if c.Side != catalog.NoSide {
		b, _ := catalog.BoardByID(c.Board)
		for stage := uint8(0); stage < c.Stages; stage++ {
			for _, e := range b.Sides[c.Side-1].Stages[stage].Effects {
				if matches(e) {
					v.Discount = &view.Discount{Stage: int(stage + 1)}
					return v
				}
			}
		}
	}
	return v
}
