package engine

import (
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// viewCardTooltips indexes only identities already exposed in the whitelist
// output. Hidden slots, deals, removed cards and OutOfPlay are never read.
// The payment solver reads the two public cities; whose turn it is does
// not affect a quote. A fixed array deduplicates and orders rows by card id.
func viewCardTooltips(s *State, public *view.View) []view.CardTooltipView {
	var rows [catalog.NumCards + 1]*view.CardTooltipView
	add := func(slug string, place view.PublicPlace, seat, slot *int, accessible bool) {
		id, ok := catalog.CardBySlug(slug)
		if !ok || rows[id] != nil {
			return
		}
		row := &view.CardTooltipView{Card: slug, Place: place, Seat: seat, Slot: slot, Accessible: accessible}
		for i := range row.Costs {
			pay := BuildingPayment(s, Seat(i), id)
			row.Costs[i] = view.CardTooltipCostView{
				Total: int(pay.Total), Affordable: pay.Affordable,
				ChainFree: pay.ChainFree, NeedsTrade: pay.TradeTotal > 0,
			}
			if !pay.Affordable {
				row.Costs[i].Shortfall = max(0, int(pay.Total)-int(s.Players[i].Coins))
			}
		}
		rows[id] = row
	}
	for _, slot := range public.Structure {
		if slot.State == view.SlotStateFaceUp && slot.Card != nil {
			add(*slot.Card, view.PublicPlaceFaceUp, nil, dtoInt(slot.Index), slot.Accessible)
		}
	}
	for i, player := range public.Players {
		for _, slug := range player.City {
			add(slug, view.PublicPlaceCity, dtoInt(i), nil, false)
		}
		for _, wonder := range player.Wonders {
			if wonder.Tucked != nil {
				add(*wonder.Tucked, view.PublicPlaceTucked, dtoInt(i), nil, false)
			}
		}
	}
	for _, slug := range public.Discard {
		add(slug, view.PublicPlaceDiscard, nil, nil, false)
	}
	// Pending card choices are subsets of the public cities/discard, and
	// play choices are subsets of the structure. They need no extra rows.
	out := []view.CardTooltipView{}
	for _, row := range rows {
		if row != nil {
			out = append(out, *row)
		}
	}
	return out
}
