package engine

import (
	"fmt"
	"math/bits"
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// Exercise served outcomes, including Play again wins, rather than an
// eligibility helper branch that ordinary next-turn threats never use.
func TestHintsEndAgeOutcomes(t *testing.T) {
	cards := [3][]string{{"theater", "palisade", "baths"}, {"brewery", "horse-breeders", "school"}, {"town-hall", "arsenal", "palace"}}
	for age := 1; age <= 3; age++ {
		for _, n := range []int{1, 2, 3} {
			for _, wonder := range []string{"pyramids", "sphinx"} {
				t.Run(fmt.Sprintf("age%d/left%d/%s", age, n, wonder), func(t *testing.T) {
					left := cards[age-1][:n]
					if n == 3 && age != 2 {
						// The bottom sacrifice covers the face-up military
						// card; taking it frees a known extra-turn win.
						left = []string{cards[age-1][1], cards[age-1][2], cards[age-1][0]}
					}
					s := hnNew(t, Scenario{Age: age, OnlyLeft: left, Wonders: wonderFill([]string{wonder}, nil), Players: [2]ScenarioPlayer{{Coins: intPtr(40)}}, Pawn: intPtr(8), ToMove: seatPtr(0)})
					key := "wonder:" + wonder + ":" + cards[age-1][0]
					o := hoOutcome(t, &s, key)
					win := o.Wins != nil && o.Wins.Kind == view.VictoryKindMilitary && o.Wins.Steps == 2
					wantWin := wonder == "sphinx" && n >= 2
					if win != wantWin {
						t.Fatalf("extra-turn win %+v, want %v", o.Wins, wantWin)
					}
					if wonder == "sphinx" {
						want := view.ExtraTurnStateGain
						if n == 1 {
							want = view.ExtraTurnStateLost
						}
						if o.Tempo == nil || o.Tempo.ExtraTurn != want {
							t.Fatalf("extra-turn outcome %+v, want %v", o.Tempo, want)
						}
					}
					x := s
					hnDo(t, &x, key)
					if !x.Over() && x.Age == s.Age && x.ToMove != s.ToMove {
						ApplyTrusted(&x, LegalActions(&x, nil)[0])
					}
					gotPick := !x.Over() && x.Age == s.Age && x.Phase == PhasePlay && x.ToMove == s.ToMove
					wantPick := n >= 3 || (n >= 2 && wonder == "sphinx")
					if gotPick != wantPick {
						t.Fatalf("actual next pick %v, want %v", gotPick, wantPick)
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
				left := cards[age-1][:n]
				if n == 3 {
					left = []string{cards[age-1][2], cards[age-1][1], cards[age-1][0]}
				}
				s := hnNew(t, Scenario{Age: age, OnlyLeft: left, Pawn: intPtr(8), ToMove: seatPtr(0)})
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
		s.Slots[slot] = Slot{Card: hnCard(t, c), FaceUp: i != 1, Back: cardBack(hnCard(t, c))}
	}
	if err := CheckInvariants(&s); err != nil {
		t.Fatal(err)
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

// Strategy can set up a military win after a science pair. The token
// decision consumes no structure card, unlike the preceding Library pick.
func TestHintsSubdecisionNextTurnThreat(t *testing.T) {
	for _, left := range []int{19, 1} {
		t.Run(fmt.Sprintf("left%d", left), func(t *testing.T) {
			sc := Scenario{
				Age:         2,
				Accessible:  []string{"library", "horse-breeders"},
				BoardTokens: []string{"strategy", "agriculture", "urbanism", "architecture", "economy"},
				Pawn:        intPtr(7),
				Players:     [2]ScenarioPlayer{{Coins: intPtr(40), City: []string{"scriptorium"}}},
				ToMove:      seatPtr(0),
			}
			if left == 1 {
				sc.OnlyLeft = []string{"library", "horse-breeders"}
			}
			s := hnNew(t, sc)
			hnDo(t, &s, "construct:library")
			if s.Phase != PhasePickBoardToken || s.ToMove != 0 {
				t.Fatalf("phase %v, to move %v", s.Phase, s.ToMove)
			}
			if got := bits.OnesCount32(Occupied(&s)); got != left {
				t.Fatalf("cards left %d, want %d", got, left)
			}
			o := hoOutcome(t, &s, "token:strategy")
			want := []string{}
			if left == 19 {
				want = []string{"0:horse-breeders:military"}
			}
			if got := hoThreats(o.Threatens); !slices.Equal(got, want) {
				t.Fatalf("token next-turn threats %v, want %v", got, want)
			}
		})
	}
}
