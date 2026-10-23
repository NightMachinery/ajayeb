package rules

import "github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"

// knowledge records only this seat's sightings and actions plus public facts.
// Every pile opportunity is captured after ordinary placements, before a
// discard build, including the publicly skipped opportunity.
func (g *Game) knowledge(viewer Seat) Knowledge {
	var out Knowledge
	if g == nil || !viewer.Valid(g.State.Seats) {
		return out
	}
	looks := 0
	err := g.walkRecord(func(before *State, picks *[MaxSeats]Action, after *State, _ []Event) {
		switch before.Phase {
		case PhaseTurn, PhaseLastCard:
			age, turn := before.Age-1, before.Turn-1
			for seat := Seat(0); seat < Seat(before.Seats); seat++ {
				a := picks[seat]
				if a.Kind == ActNone {
					continue
				}
				p := Play{Kind: a.Kind}
				if a.Kind == ActBuild {
					p.Design = a.Design
				}
				if a.Kind == ActBuild || a.Kind == ActStage {
					p.Left, p.Right = a.Left, a.Right
				}
				if a.Kind == ActStage {
					p.Stage = before.Cities[seat].Stages + 1
				}
				out.Plays[age][turn][seat] = p
				if seat == viewer {
					if before.Phase == PhaseLastCard {
						out.Seen[age][turn][0] = before.Cities[seat].Kept
					} else {
						out.Seen[age][turn] = before.Hands[seat]
					}
					canonicalIDs(out.Seen[age][turn][:])
					out.Picks[age][turn] = a.Design
				}
			}
			if picks[viewer].Kind == ActStage && stageHasEffect(&after.Cities[viewer], after.Cities[viewer].Stages, catalog.EffBuildFromDiscard) && looks < MaxPileLooks {
				out.PileLooks[looks] = PileLook{Age: before.Age, Turn: before.Turn, Count: after.Discard}
				looks++
			}
		case PhaseDiscardBuild:
			out.Rebuilt[before.Age-1][before.Turn-1][before.Decider] = picks[before.Decider].Design
		}
	})
	if err != nil {
		return Knowledge{}
	}
	return out
}
