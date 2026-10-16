package engine_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// Payment ends before income effects, including Urbanism. These actual
// applications distinguish the projection from the final post-action purse.
func TestViewCoinsAfterBeforeEffects(t *testing.T) {
	cases := []struct {
		name, key             string
		state                 engine.State
		afterPay, afterAction int
	}{
		{"free income", "construct:tavern", enginetest.New(t, 7).Age(1).Coins(0, 5).Accessible("tavern").ToMove(0).State(), 5, 9},
		{"trade", "construct:baths", enginetest.New(t, 7).Age(1).Coins(0, 5).Accessible("baths").ToMove(0).State(), 3, 3},
		{"Urbanism then income", "construct:arena", enginetest.New(t, 7).Age(3).Coins(0, 2).City(0, "brewery").Tokens(0, "urbanism").Accessible("arena").ToMove(0).State(), 2, 6},
		{"wonder income", "wonder:appian-way:rostrum", enginetest.New(t, 7).Age(2).Wonders(0, "appian-way").Coins(0, 20).Accessible("rostrum").ToMove(0).State(), 10, 13},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.state
			v := engine.ViewFor(&s, 0)
			var cost *view.CostView
			for _, c := range v.Choices {
				if c.Key == tc.key {
					cost = c.Cost
				}
			}
			if cost == nil || cost.CoinsAfter == nil || *cost.CoinsAfter != tc.afterPay {
				t.Fatalf("cost %+v, want coinsAfter %d", cost, tc.afterPay)
			}
			enginetest.Do(t, &s, tc.key)
			if got := int(s.Players[0].Coins); got != tc.afterAction {
				t.Fatalf("final purse %d, want %d", got, tc.afterAction)
			}
		})
	}
}

func TestViewCoinsAfterUnaffordableAndRedaction(t *testing.T) {
	s := enginetest.New(t, 7).Age(1).Coins(0, 0).Coins(1, 7).Accessible("baths").ToMove(0).State()
	for _, viewer := range []engine.Viewer{0, 1, engine.Spectator} {
		v := engine.ViewFor(&s, viewer)
		for _, slot := range v.Structure {
			if slot.Card == nil {
				if slot.Costs != nil {
					t.Fatal("hidden card has costs")
				}
				continue
			}
			for seat, c := range slot.Costs {
				if c.Affordable != (c.CoinsAfter != nil) {
					t.Fatalf("viewer %d seat %d: coinsAfter presence differs from affordability", viewer, seat)
				}
				if c.CoinsAfter != nil && *c.CoinsAfter != int(s.Players[seat].Coins)-c.Total {
					t.Fatal("wrong public seat purse")
				}
			}
		}
		if viewer != 0 && (len(v.Choices) != 0 || len(v.Disabled) != 0) {
			t.Fatal("private choices leaked")
		}
		for _, d := range v.Disabled {
			if d.Cost != nil && !d.Cost.Affordable {
				b, err := json.Marshal(d.Cost)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(string(b), "coinsAfter") || d.Cost.Shortfall <= 0 {
					t.Fatalf("unaffordable preview %s", b)
				}
			}
		}
	}
}
