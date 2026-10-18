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

func chainTooltipRow(t *testing.T, v view.View, slug string) view.CardTooltipView {
	t.Helper()
	for _, row := range v.CardTooltips {
		if row.Card == slug {
			return row
		}
	}
	t.Fatalf("missing public tooltip row %s", slug)
	return view.CardTooltipView{}
}

func TestCardTooltipsLinkedWithoutChainCosts(t *testing.T) {
	s := enginetest.New(t, 7).Age(2).City(0, "baths").City(1, "stone-pit").Coins(0, 7).Coins(1, 1).Accessible("aqueduct").ToMove(0).State()
	before := s
	v := engine.ViewFor(&s, 0)
	out := chainTooltipRow(t, v, "baths").ChainOut
	if out == nil || out.Costs == nil || out.Place != view.PublicPlaceFaceUp || !out.Accessible || out.Slot == nil {
		t.Fatalf("missing target preview: %+v", out)
	}
	if a, b := out.Costs[0], out.Costs[1]; a.Total != 9 || a.ChainFree || a.Affordable || a.Shortfall != 2 || !a.NeedsTrade || b.Total != 4 || b.ChainFree || b.Affordable || b.Shortfall != 3 || !b.NeedsTrade {
		t.Fatalf("wrong without-chain quotes: %+v", *out.Costs)
	}
	if !chainTooltipRow(t, v, "aqueduct").Costs[0].ChainFree {
		t.Fatal("ordinary quote must remain chain-free")
	}
	in := chainTooltipRow(t, v, "aqueduct").ChainIn
	if in == nil || in.Costs != nil || in.Place != view.PublicPlaceCity || in.Seat == nil || *in.Seat != 0 {
		t.Fatalf("wrong reverse source: %+v", in)
	}
	if s != before {
		t.Fatal("tooltip assembly mutated state")
	}
	s.ToMove = 1
	if got := chainTooltipRow(t, engine.ViewFor(&s, 1), "baths").ChainOut; !reflect.DeepEqual(got, out) {
		t.Fatal("linked quotes depend on turn or viewer")
	}

	// Bypassing a chain must still use the full cost solver, including
	// production, fixed prices, flexible producers and the blue discount.
	for _, tc := range []struct {
		name   string
		city   []string
		tokens []string
		want   int
		trade  bool
	}{
		{"own production", []string{"baths", "quarry"}, nil, 6, true},
		{"fixed price", []string{"baths", "stone-reserve"}, nil, 3, true},
		{"flexible producer", []string{"baths", "caravansery"}, nil, 6, true},
		{"masonry retained", []string{"baths"}, []string{"masonry"}, 3, true},
		{"production and masonry", []string{"baths", "caravansery"}, []string{"masonry"}, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x := enginetest.New(t, 7).Age(2).City(0, tc.city...).City(1, "stone-pit").Tokens(0, tc.tokens...).Coins(0, 20).Accessible("aqueduct").ToMove(0).State()
			got := chainTooltipRow(t, engine.ViewFor(&x, 0), "baths").ChainOut.Costs[0]
			if got.Total != tc.want || got.NeedsTrade != tc.trade || got.ChainFree || !got.Affordable || got.Shortfall != 0 {
				t.Fatalf("wrong quote: %+v", got)
			}
		})
	}
}

func TestCardTooltipsLinkedPublicWhereabouts(t *testing.T) {
	for _, tc := range []struct {
		name       string
		state      engine.State
		root       string
		place      view.PublicPlace
		accessible bool
	}{
		{"accessible faceup", enginetest.New(t, 7).Age(2).City(0, "baths").Accessible("aqueduct").ToMove(0).State(), "baths", view.PublicPlaceFaceUp, true},
		{"rival city", enginetest.New(t, 7).Age(2).City(0, "baths").City(1, "aqueduct").ToMove(0).State(), "baths", view.PublicPlaceCity, false},
		{"discard", enginetest.New(t, 7).Age(2).City(0, "baths").Discard("aqueduct").ToMove(0).State(), "baths", view.PublicPlaceDiscard, false},
		{"later age", enginetest.New(t, 7).Age(1).City(0, "baths").ToMove(0).State(), "baths", view.PublicPlaceUnseenLaterAge, false},
		{"ended age absence", enginetest.New(t, 7).Age(3).City(0, "baths").ToMove(0).State(), "baths", view.PublicPlaceRemovedKnown, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, viewer := range redactViewers {
				got := chainTooltipRow(t, engine.ViewFor(&tc.state, viewer), tc.root).ChainOut
				if got == nil || got.Place != tc.place || got.Accessible != tc.accessible {
					t.Fatalf("wrong public whereabouts: %+v", got)
				}
				if tc.place == view.PublicPlaceCity && (got.Seat == nil || *got.Seat != 1) {
					t.Fatal("rival city lost actual owner")
				}
			}
		})
	}
	// A public but covered card has a known identity and a distinct status.
	s := enginetest.New(t, 7).Age(2).City(0, "baths").Accessible("aqueduct").ToMove(0).State()
	for i := range s.Slots {
		if catalog.Card(s.Slots[i].Card) != nil && catalog.Card(s.Slots[i].Card).Slug == "aqueduct" {
			// Reuse an occupied, face-up but inaccessible slot's topology.
			for j, other := range s.Slots {
				if other.Card != 0 && other.FaceUp && !engine.ViewFor(&s, 0).Structure[j].Accessible {
					s.Slots[i].Card, s.Slots[j].Card = s.Slots[j].Card, s.Slots[i].Card
					break
				}
			}
			break
		}
	}
	if got := chainTooltipRow(t, engine.ViewFor(&s, 0), "baths").ChainOut; got.Place != view.PublicPlaceFaceUp || got.Accessible || got.Slot == nil {
		t.Fatalf("covered face-up target lost: %+v", got)
	}
}

func TestCardTooltipsLinkedHiddenRemovedAndScenarioEquivalent(t *testing.T) {
	s := enginetest.New(t, 7).Age(2).City(0, "baths").Accessible("aqueduct").ToMove(0).State()
	id, _ := catalog.CardBySlug("aqueduct")
	visible, hidden := -1, -1
	for i, sl := range s.Slots {
		if sl.Card == id {
			visible = i
		}
		if sl.Card != 0 && !sl.FaceUp {
			hidden = i
		}
	}
	if visible < 0 || hidden < 0 || s.Removed[1][0] == 0 {
		t.Fatal("fixture needs target, hidden slot, and removed card")
	}
	s.Slots[visible].Card, s.Slots[hidden].Card = s.Slots[hidden].Card, s.Slots[visible].Card
	removed := s
	removed.Slots[hidden].Card, removed.Removed[1][0] = removed.Removed[1][0], removed.Slots[hidden].Card
	outOfPlay := removed
	outOfPlay.OutOfPlay.Add(id)
	for _, viewer := range redactViewers {
		a := engine.ViewFor(&s, viewer)
		for _, changed := range []engine.State{removed, outOfPlay} {
			b := engine.ViewFor(&changed, viewer)
			if !reflect.DeepEqual(a.CardTooltips, b.CardTooltips) {
				t.Fatal("hidden, removed, or scenario status affected complete tooltip index")
			}
		}
		if got := chainTooltipRow(t, a, "baths").ChainOut; got.Place != view.PublicPlaceUnseenThisAge || got.Seat != nil || got.Slot != nil || got.Accessible {
			t.Fatalf("current unseen target exposed: %+v", got)
		}
		for _, row := range a.CardTooltips {
			if row.Card == "aqueduct" {
				t.Fatal("hidden target gained a root row")
			}
		}
	}
}

func TestCardTooltipsLinkedAbsenceAfterFinalAge(t *testing.T) {
	s := enginetest.New(t, 7).Age(3).City(0, "rostrum").OnlyLeft("town-hall").Coins(0, 30).ToMove(0).State()
	if got := chainTooltipRow(t, engine.ViewFor(&s, 0), "rostrum").ChainOut.Place; got != view.PublicPlaceUnseenThisAge {
		t.Fatalf("scenario absence while age is in progress: %s", got)
	}
	early := s
	enginetest.Do(t, &early, engine.ResignAction(0).Key())
	for _, viewer := range redactViewers {
		if got := chainTooltipRow(t, engine.ViewFor(&early, viewer), "rostrum").ChainOut.Place; got != view.PublicPlaceUnseenThisAge {
			t.Fatalf("early forfeit must not finish the current age: %s", got)
		}
	}
	enginetest.Do(t, &s, "construct:town-hall")
	for _, viewer := range redactViewers {
		if got := chainTooltipRow(t, engine.ViewFor(&s, viewer), "rostrum").ChainOut.Place; got != view.PublicPlaceRemovedKnown {
			t.Fatalf("absent target after emptied final age: %s", got)
		}
	}
}
