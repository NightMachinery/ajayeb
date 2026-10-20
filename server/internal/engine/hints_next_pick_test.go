package engine

import (
	"fmt"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// A normal reply consumes the penultimate card; Play again instead gives
// the mover the next pick immediately (rules/turn.md).
func TestHintsNextPickInAge(t *testing.T) {
	cards := [3][]string{{"theater", "baths", "altar"}, {"brewery", "school", "library"}, {"town-hall", "palace", "academy"}}
	for age := 1; age <= 3; age++ {
		for _, n := range []int{1, 2, 3} {
			for _, wonder := range []string{"pyramids", "sphinx"} {
				t.Run(fmt.Sprintf("age%d/left%d/%s", age, n, wonder), func(t *testing.T) {
					s := hnNew(t, Scenario{Age: age, OnlyLeft: cards[age-1][:n], Wonders: wonderFill([]string{wonder}, nil), Players: [2]ScenarioPlayer{{Coins: intPtr(40)}}, ToMove: seatPtr(0)})
					var a Action
					for _, c := range LegalActions(&s, nil) {
						if c.Type == ActConstructWonder {
							a = c
							break
						}
					}
					if a.Type != ActConstructWonder {
						t.Fatal("missing wonder action")
					}
					want := n >= 3 || (n >= 2 && wonder == "sphinx")
					if got := nextPickInAge(&s, a); got != want {
						t.Fatalf("next pick %v, want %v", got, want)
					}
					x := s
					ApplyTrusted(&x, a)
					if !x.Over() && x.Age == s.Age && x.ToMove != s.ToMove {
						ApplyTrusted(&x, LegalActions(&x, nil)[0])
					}
					got := !x.Over() && x.Age == s.Age && x.Phase == PhasePlay && x.ToMove == s.ToMove
					if got != want {
						t.Fatalf("actual next pick %v, want %v", got, want)
					}
				})
			}
		}
	}
}

// Discarding can fund a win that is not affordable now, but on the last
// two cards the mover will not get a chance to use those coins this age.
func TestHintsNextTurnThreatEndsWithAge(t *testing.T) {
	cards := [3][]string{{"palisade", "theater", "baths"}, {"horse-breeders", "brewery", "school"}, {"arsenal", "town-hall", "palace"}}
	for age := 1; age <= 3; age++ {
		for _, n := range []int{2, 3} {
			t.Run(fmt.Sprintf("age%d/left%d", age, n), func(t *testing.T) {
				s := hnNew(t, Scenario{Age: age, OnlyLeft: cards[age-1][:n], Pawn: intPtr(8), ToMove: seatPtr(0)})
				// Keep the small endgame fixture's remaining cards face up
				// and independent, so both a target and another discard exist.
				var slots [SlotsPerAge]Slot
				for i, c := range cards[age-1][:n] {
					slots[5+i] = Slot{Card: hnCard(t, c), FaceUp: true}
				}
				s.Slots = slots
				target := cards[age-1][0]
				quote := BuildingQuote(&s, 0, hnCard(t, target))
				s.Players[0].Coins = quote.Total - 2
				v := hoView(t, &s)
				var blocked *view.DisabledChoice
				for i := range v.Disabled {
					if v.Disabled[i].Key == "construct:"+target {
						blocked = &v.Disabled[i]
					}
				}
				if blocked == nil {
					t.Fatal("target must be unaffordable")
				}
				if got := blocked.DiscardToAfford != nil; got != (n == 3) {
					t.Fatalf("discard advice %v, want %v", got, n == 3)
				}
				if blocked.DiscardToAfford != nil && *blocked.DiscardToAfford != 2 {
					t.Fatalf("discard gain %d, want 2", *blocked.DiscardToAfford)
				}
				o := hoOutcome(t, &s, "discard:"+cards[age-1][1])
				if got := len(hoThreats(o.Threatens)) > 0; got != (n == 3) {
					t.Fatalf("next-turn threats %v with %d left", hoThreats(o.Threatens), n)
				}
				f := hnAllHints
				f.Threats = false
				h := ComputeHints(&s, f)
				economy := ViewWithHints(&s, SeatViewer(0), &h)
				for _, c := range economy.Disabled {
					if c.Key == blocked.Key && (c.DiscardToAfford != nil) != (n == 3) {
						t.Fatal("discard advice depends on threats family")
					}
				}
				f.Economy = false
				h = ComputeHints(&s, f)
				v = ViewWithHints(&s, SeatViewer(0), &h)
				for _, c := range v.Disabled {
					if c.DiscardToAfford != nil {
						t.Fatal("economy off still offers discard advice")
					}
				}
			})
		}
	}
}

// Equivalent to the tutorial's final Palisade: enough cards may remain in
// the age, yet there is no different takeable card to discard for coins.
func TestHintsDiscardNeedsAnotherTakeableCard(t *testing.T) {
	s := hnNew(t, Scenario{Age: 1, OnlyLeft: []string{"theater", "baths", "palisade"}, Players: [2]ScenarioPlayer{{Coins: intPtr(1)}}, ToMove: seatPtr(0)})
	s.Slots = [SlotsPerAge]Slot{}
	for i, c := range []string{"theater", "baths", "palisade"} {
		slot := []int{0, 2, 5}[i] // one chain of coverers, only Palisade takeable
		s.Slots[slot] = Slot{Card: hnCard(t, c), FaceUp: true}
	}
	for _, c := range hoView(t, &s).Disabled {
		if c.Key == "construct:palisade" {
			if c.DiscardToAfford != nil {
				t.Fatal("discard advice without another takeable card")
			}
			return
		}
	}
	t.Fatal("missing blocked Palisade")
}
