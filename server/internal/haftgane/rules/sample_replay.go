package rules

import "github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"

// MatchesObservation is an independent, checked full-history replay. It never
// calls the constraint builder, source search or matching code. Intended for
// tests/debugging, not the search hot path (Resolve and Observe allocate).
func MatchesObservation(o Observation, w World) bool {
	c := &o.Now
	if !c.Seat.Valid(c.Seats) || w.state.Seed != 0 || c.Points > MaxPoints {
		return false
	}
	initial := State{Seats: c.Seats, Opts: c.Opts, Decider: NoSeat, BoxGuilds: w.state.BoxGuilds}
	for s := Seat(0); s < Seat(c.Seats); s++ {
		side := c.Cities[s].Side
		if c.Opts.Sides == SidesChoose {
			side = catalog.NoSide
		}
		initial.Cities[s] = City{Board: c.Cities[s].Board, Side: side, Coins: 3}
	}
	for age := range w.cards {
		initial.Deals[age] = w.cards[age]
		for s := range initial.Deals[age] {
			canonicalIDs(initial.Deals[age][s][:])
		}
	}
	g := Game{State: initial}
	if c.Opts.Sides != SidesChoose {
		if err := dealAge(&g, 1); err != nil {
			return false
		}
	}
	var err error
	g, err = NewGameFromScenario(0, c.Opts, ScenarioFromGame(g))
	if err != nil {
		return false
	}
	for step := uint16(0); step < c.Points; step++ {
		st := &g.State
		point := g.Point()
		if point.Kind == PointNone {
			return false
		}
		var picks [MaxSeats]Pick
		count := 0
		for s := Seat(0); s < Seat(c.Seats); s++ {
			if !point.Required.Has(s) {
				continue
			}
			var a Action
			switch st.Phase {
			case PhaseSides:
				a = Action{Kind: ActSide, Side: c.Cities[s].Side}
			case PhaseTurn, PhaseLastCard:
				p := o.Know.Plays[st.Age-1][st.Turn-1][s]
				turn := min(st.Turn, uint8(TurnsPerAge))
				origin := packetOrigin(s, c.Seats, st.Age, turn)
				id := w.cards[st.Age-1][origin][st.Turn-1]
				if p.Kind == ActBuild && id != p.Design {
					return false
				}
				a = Action{Kind: p.Kind, Design: id, Left: p.Left, Right: p.Right}
			case PhaseDiscardBuild:
				a = Action{Kind: ActBuildDiscarded, Design: o.Know.Rebuilt[st.Age-1][st.Turn-1][s]}
			default:
				return false
			}
			picks[count] = Pick{Seat: s, Key: a.Key()}
			count++
		}
		next, _, resolveErr := g.Resolve(Batch{Point: point.ID(), Picks: picks[:count]})
		if resolveErr != nil {
			return false
		}
		g = next
	}
	return g.State == w.state && g.Observe(c.Seat) == o && g.CheckInvariants() == nil
}
