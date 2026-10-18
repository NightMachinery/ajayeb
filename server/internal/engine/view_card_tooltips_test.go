package engine_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

func TestCardTooltipsPublicQuotes(t *testing.T) {
	for _, fx := range protoFixtures(t) {
		for _, viewer := range redactViewers {
			v := engine.ViewFor(&fx.state, viewer)
			visible := map[string]view.PublicPlace{}
			for _, slot := range v.Structure {
				if slot.Card != nil {
					visible[*slot.Card] = view.PublicPlaceFaceUp
				}
			}
			for _, p := range v.Players {
				for _, slug := range p.City {
					visible[slug] = view.PublicPlaceCity
				}
				for _, w := range p.Wonders {
					if w.Tucked != nil {
						visible[*w.Tucked] = view.PublicPlaceTucked
					}
				}
			}
			for _, slug := range v.Discard {
				visible[slug] = view.PublicPlaceDiscard
			}
			if v.CardTooltips == nil || len(v.CardTooltips) != len(visible) {
				t.Fatalf("%s: missing or nil public index", fx.name)
			}
			last := catalog.CardID(0)
			for _, row := range v.CardTooltips {
				if place, ok := visible[row.Card]; !ok || row.Place != place {
					t.Fatalf("%s: nonpublic row %+v", fx.name, row)
				}
				id, ok := catalog.CardBySlug(row.Card)
				if !ok || id <= last {
					t.Fatal("tooltip rows are not unique catalog order")
				}
				last = id
				for seat, got := range row.Costs {
					pay := engine.BuildingPayment(&fx.state, engine.Seat(seat), id)
					short := 0
					if !pay.Affordable {
						short = max(0, int(pay.Total)-int(fx.state.Players[seat].Coins))
					}
					want := view.CardTooltipCostView{Total: int(pay.Total), Affordable: pay.Affordable, ChainFree: pay.ChainFree, NeedsTrade: pay.TradeTotal > 0, Shortfall: short}
					if got != want {
						t.Fatalf("%s %s seat %d: %+v, want %+v", fx.name, row.Card, seat, got, want)
					}
				}
			}
		}
	}
}

func TestCardTooltipsHiddenIdentityDoesNotAffectIndex(t *testing.T) {
	s := enginetest.New(t, 7).Age(1).ToMove(0).State()
	changed := s
	slot := -1
	for i, sl := range s.Slots {
		if sl.Card != 0 && !sl.FaceUp {
			slot = i
			break
		}
	}
	if slot < 0 || s.Removed[0][0] == 0 {
		t.Fatal("fixture needs a hidden and a removed card")
	}
	hidden, removed := s.Slots[slot].Card, s.Removed[0][0]
	changed.Slots[slot].Card, changed.Removed[0][0] = removed, hidden
	for _, viewer := range redactViewers {
		a, b := engine.ViewFor(&s, viewer), engine.ViewFor(&changed, viewer)
		if !reflect.DeepEqual(a.CardTooltips, b.CardTooltips) {
			t.Fatal("hidden identities affected public card index")
		}
		for _, row := range a.CardTooltips {
			id, _ := catalog.CardBySlug(row.Card)
			if id == hidden || id == removed || s.OutOfPlay.Has(id) {
				t.Fatalf("hidden card named in %+v", row)
			}
		}
	}
}

func TestCardTooltipsAsymmetricCostsAndOwner(t *testing.T) {
	s := enginetest.New(t, 7).Age(2).City(0, "baths").City(1, "stone-pit").Coins(0, 0).Coins(1, 1).Accessible("aqueduct").ToMove(0).State()
	v := engine.ViewFor(&s, 0)
	for _, row := range v.CardTooltips {
		switch row.Card {
		case "aqueduct":
			if !row.Costs[0].ChainFree || row.Costs[1].Total != 4 || row.Costs[1].Affordable || row.Costs[1].Shortfall != 3 || !row.Costs[1].NeedsTrade || !row.Accessible || row.Slot == nil {
				t.Fatalf("wrong asymmetric quote: %+v", row)
			}
		case "baths":
			if row.Place != view.PublicPlaceCity || row.Seat == nil || *row.Seat != 0 {
				t.Fatalf("wrong owner: %+v", row)
			}
		}
	}
	b, err := json.Marshal(v.CardTooltips)
	if err != nil || len(b) == 0 {
		t.Fatalf("index is not serializable: %v", err)
	}
}

func TestCardTooltipsReviveChoiceIsExplicitlyFree(t *testing.T) {
	found := false
	for _, fx := range protoFixtures(t) {
		if fx.state.Phase != engine.PhaseBuildFromDiscard {
			continue
		}
		v := engine.ViewFor(&fx.state, engine.SeatViewer(fx.state.ToMove))
		for _, choice := range v.Choices {
			if choice.Kind != view.ChoiceKindRevive {
				continue
			}
			found = true
			cost := choice.Cost
			if cost == nil || !cost.Affordable || cost.Total != 0 || cost.ChainFree || cost.Shortfall != 0 || cost.CoinsAfter == nil || *cost.CoinsAfter != int(fx.state.Players[fx.state.ToMove].Coins) || len(cost.Trade) != 0 {
				t.Fatalf("%s: revive must send its actual zero payment, got %+v", fx.name, cost)
			}
		}
	}
	if !found {
		t.Fatal("protocol positions did not exercise a legal revive")
	}
}
