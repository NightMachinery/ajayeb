package rules

import "github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"

// The assignment is derived, not stored in a frozen value. Views can report
// both assigned symbols and the final counts without offering a scoring move.
type scienceAssignment struct {
	Points  int16
	Counts  [catalog.NumSymbols]uint8
	Choices [catalog.NumSymbols]uint8
}

// scienceScore tries every distribution of choices. Permuting identical
// choices has no effect, so this covers 3^k ordered assignments in O(k^2).
// Equal optima prefer more compass, then gear, matching the first lexicographic
// ordered assignment. Real cities have at most two choices; tests stress seven.
func scienceScore(counts [catalog.NumSymbols]uint8, choices uint8) scienceAssignment {
	best := scienceAssignment{Points: -1}
	for compass := int(choices); compass >= 0; compass-- {
		for gear := int(choices) - compass; gear >= 0; gear-- {
			picked := [catalog.NumSymbols]uint8{catalog.Compass: uint8(compass), catalog.Gear: uint8(gear), catalog.Tablet: choices - uint8(compass+gear)}
			final := counts
			for s := catalog.Compass; s <= catalog.Tablet; s++ {
				final[s] += picked[s]
			}
			a, b, c := int16(final[catalog.Compass]), int16(final[catalog.Gear]), int16(final[catalog.Tablet])
			points := a*a + b*b + c*c + 7*min(a, b, c)
			if points > best.Points {
				best = scienceAssignment{Points: points, Counts: final, Choices: picked}
			}
		}
	}
	return best
}

func endPoints(st *State, seat Seat, e catalog.Effect) int16 {
	if e.Kind.Timings()&catalog.EndOfGame == 0 {
		return 0
	}
	switch e.Kind {
	case catalog.EffPoints:
		return int16(e.N)
	case catalog.EffPointsPerCard:
		return int16(e.N) * scopeCount(st, seat, e.Scope, e.Colors, false)
	case catalog.EffPointsPerStage:
		return int16(e.N) * scopeCount(st, seat, e.Scope, 0, true)
	case catalog.EffPointsIfWonderComplete:
		c := &st.Cities[seat]
		board, _ := catalog.BoardByID(c.Board)
		if c.Side != catalog.NoSide && c.Stages == board.Sides[c.Side-1].NumStages {
			return int16(e.N)
		}
		return 0
	case catalog.EffScience, catalog.EffScienceChoice:
		// Science always belongs in green, even when a guild or stage holds it.
		return 0
	default:
		panic("rules: unhandled end-of-game effect")
	}
}

// victoryPoints is a victory token's worth by Age; a defeat token is -1.
var victoryPoints = [NumAges]int16{1, 3, 5}

// scoreCity follows score-pad order. Continuous caches provide science; the
// public holders provide their end points. No tucked card contributes a card
// effect, and scoring never repeats a when-placed income effect.
func scoreCity(st *State, seat Seat) Score {
	c := &st.Cities[seat]
	score := Score{Treasure: c.Coins / 3, Military: victoryPoints[0]*int16(c.Victories[0]) + victoryPoints[1]*int16(c.Victories[1]) + victoryPoints[2]*int16(c.Victories[2]) - int16(c.Defeats)}
	if c.Stages != 0 {
		board, _ := catalog.BoardByID(c.Board)
		for stage := uint8(0); stage < c.Stages; stage++ {
			for _, e := range board.Sides[c.Side-1].Stages[stage].Effects {
				score.Wonder += endPoints(st, seat, e)
			}
		}
	}
	for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
		if !c.Buildings.Has(id) {
			continue
		}
		d, _ := catalog.DesignByID(id)
		var points int16
		for _, e := range d.Effects {
			points += endPoints(st, seat, e)
		}
		switch d.Color {
		case catalog.Blue:
			score.Blue += points
		case catalog.Yellow:
			score.Yellow += points
		case catalog.Purple:
			score.Purple += points
		default:
			if points != 0 {
				panic("rules: unhandled points holder")
			}
		}
	}
	score.Green = scienceScore(c.Science, c.ScienceChoice).Points
	score.Total = score.Wonder + score.Treasure + score.Military + score.Blue + score.Yellow + score.Green + score.Purple
	return score
}

func scoreResult(st *State) Result {
	result := Result{Over: true}
	for seat := Seat(0); seat < Seat(st.Seats); seat++ {
		result.Scores[seat] = scoreCity(st, seat)
	}
	for seat := Seat(0); seat < Seat(st.Seats); seat++ {
		rank := uint8(1)
		for other := Seat(0); other < Seat(st.Seats); other++ {
			if result.Scores[other].Total > result.Scores[seat].Total || (result.Scores[other].Total == result.Scores[seat].Total && st.Cities[other].Coins > st.Cities[seat].Coins) {
				rank++
			}
		}
		result.Rank[seat] = rank
		if rank == 1 {
			result.Winners |= uint8(seat.Mask())
		}
	}
	return result
}

// topTie reads a finished ranking: how a tie on the top total ended, and the
// seats in it. Every winner holds the top total, so coins decided ("coins")
// exactly when fewer seats won than tied, and otherwise the tied seats share
// the victory ("shared"). One seat alone at the top is no tie: "" and none.
func topTie(r *Result, seats uint8) (string, SeatSet) {
	var top int16
	for seat := Seat(0); seat < Seat(seats); seat++ {
		if seat == 0 || r.Scores[seat].Total > top {
			top = r.Scores[seat].Total
		}
	}
	var tied SeatSet
	for seat := Seat(0); seat < Seat(seats); seat++ {
		if r.Scores[seat].Total == top {
			tied |= seat.Mask()
		}
	}
	switch {
	case tied.Count() < 2:
		return "", 0
	case SeatSet(r.Winners).Count() < tied.Count():
		return "coins", tied
	default:
		return "shared", tied
	}
}

func finishScored(st *State) {
	st.Phase = PhaseOver
	st.Decider = NoSeat
	st.Result = scoreResult(st)
}
