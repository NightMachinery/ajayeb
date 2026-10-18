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
			row.Costs[i] = cardTooltipCost(s, Seat(i), BuildingPayment(s, Seat(i), id))
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
	for id, row := range rows {
		if row != nil {
			if ch := catalog.T.ChainIn[id]; ch != catalog.NoChain && catalog.T.ChainSource[ch] != 0 {
				row.ChainIn = cardChainTooltip(public, &rows, catalog.T.ChainSource[ch])
			}
			if ch := catalog.T.ChainOut[id]; ch != catalog.NoChain && catalog.T.ChainTarget[ch] != 0 {
				target := catalog.T.ChainTarget[ch]
				row.ChainOut = cardChainTooltip(public, &rows, target)
				var costs [2]view.CardTooltipCostView
				for seat := range costs {
					// Remove only the chain alternative before the usual adapter.
					// Clearing the flag after buildingInput would lose Masonry,
					// because that adapter returns early for a chain-free card.
					d := *catalog.Card(target)
					d.ChainIn = catalog.NoChain
					var in costInput
					buildingInput(s, Seat(seat), &d, &in)
					var pay Payment
					cheapest(&in, &pay)
					costs[seat] = cardTooltipCost(s, Seat(seat), pay)
				}
				row.ChainOut.Costs = &costs
			}
			out = append(out, *row)
		}
	}
	return out
}

func cardTooltipCost(s *State, seat Seat, pay Payment) view.CardTooltipCostView {
	out := view.CardTooltipCostView{
		Total: int(pay.Total), Affordable: pay.Affordable,
		ChainFree: pay.ChainFree, NeedsTrade: pay.TradeTotal > 0,
	}
	if !pay.Affordable {
		out.Shortfall = max(0, int(pay.Total)-int(s.Players[seat].Coins))
	}
	return out
}

// Only the already redacted index and catalog age decide whereabouts. In
// particular, scenario OutOfPlay and the identities in hidden slots must
// never distinguish an unseen card from a removed card of the current age.
func cardChainTooltip(public *view.View, rows *[catalog.NumCards + 1]*view.CardTooltipView, id catalog.CardID) *view.CardChainTooltipView {
	if row := rows[id]; row != nil {
		return &view.CardChainTooltipView{Place: row.Place, Seat: row.Seat, Slot: row.Slot, Accessible: row.Accessible}
	}
	place := view.PublicPlaceUnseenThisAge
	age := int(catalog.T.Age[id])
	if age > public.Age {
		place = view.PublicPlaceUnseenLaterAge
	} else if age < public.Age || cardTooltipAgeFinished(public) {
		place = view.PublicPlaceRemovedKnown
	}
	return &view.CardChainTooltipView{Place: place}
}

// An emptied public structure also finishes the current age, including the
// final age. A supremacy or forfeit with cards left does not finish it.
func cardTooltipAgeFinished(public *view.View) bool {
	if len(public.Structure) == 0 {
		return false
	}
	for _, slot := range public.Structure {
		if slot.State != view.SlotStateEmpty {
			return false
		}
	}
	return true
}
