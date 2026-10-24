package rules

import "testing"

// Military conflicts (rules.md): each event names both strengths, and a
// victory is worth 1, 3 or 5 by Age, a tie nothing. The tokens the events
// give add up to each city's military score.
func TestMilitaryEventsCarryShieldsAndTokens(t *testing.T) {
	worth := [NumAges]int{1, 3, 5}
	wins, ties := 0, 0
	for seed := uint64(1); seed <= 4; seed++ {
		g, err := NewGame(Setup{Seats: 5, Seed: seed, Options: Options{Sides: SidesDay}})
		if err != nil {
			t.Fatal(err)
		}
		var military [MaxSeats]int
		for g.State.Phase != PhaseOver {
			next, events, err := g.Resolve(observationBatch(g, seed))
			if err != nil {
				t.Fatal(err)
			}
			var raw []Event
			for _, e := range events {
				if e.Kind != EvMilitary {
					continue
				}
				raw = append(raw, e)
				left, right := next.State.Cities[e.Seat].Shields, next.State.Cities[e.Other].Shields
				if e.Left != left || e.Right != right {
					t.Fatal("conflict strengths", e, left, right)
				}
				switch {
				case left == right:
					ties++
					if e.Delta != 0 || e.Outcome != ConflictTie {
						t.Fatal("a tie with a token", e)
					}
				case left > right:
					wins++
					military[e.Seat] += worth[e.Age-1]
					military[e.Other]--
					if int(e.Delta) != worth[e.Age-1] || e.Outcome != ConflictSeatWins {
						t.Fatal("victory token", e)
					}
				default:
					wins++
					military[e.Other] += worth[e.Age-1]
					military[e.Seat]--
					if int(e.Delta) != worth[e.Age-1] || e.Outcome != ConflictOtherWins {
						t.Fatal("victory token", e)
					}
				}
			}
			for _, viewer := range []Seat{NoSeat, 0, 3} {
				for i, v := range FilterEvents(raw, viewer) {
					e := raw[i]
					if v.Kind != "military" || v.Left != int(e.Left) || v.Right != int(e.Right) || v.Delta != int(e.Delta) {
						t.Fatal("filtered conflict", viewer, v, e)
					}
				}
			}
			g = next
		}
		for s := Seat(0); s < 5; s++ {
			if int(g.State.Result.Scores[s].Military) != military[s] {
				t.Fatal("tokens and military score disagree", s, military[s], g.State.Result.Scores[s].Military)
			}
		}
	}
	if wins == 0 || ties == 0 {
		t.Fatal("too few conflicts of each kind", wins, ties)
	}
}
