package rules

import "fmt"

// Batch is a complete point, in ascending seat order. Pending picks never
// enter Game. Pick contains a canonical key, not client-supplied rule fields.
type Batch struct {
	Point string
	Picks []Pick
}
type Pick struct {
	Seat Seat
	Key  string
}

// Resolve checks exact point/seat/key membership before handing the engine's
// own actions to the trusted state core. An error changes no value or record.
func (g *Game) Resolve(b Batch) (Game, []Event, error) {
	if g == nil {
		return Game{}, nil, fmt.Errorf("rules: nil game")
	}
	point := g.Point()
	if point.Kind == PointNone || b.Point != point.ID() || len(b.Picks) != point.Required.Count() {
		return *g, nil, fmt.Errorf("rules: batch does not match current point")
	}
	var picks [MaxSeats]Action
	var seen SeatSet
	for i, p := range b.Picks {
		if !point.Required.Has(p.Seat) || seen.Has(p.Seat) || (i > 0 && b.Picks[i-1].Seat >= p.Seat) {
			return *g, nil, fmt.Errorf("rules: batch seats must be exactly required seats in order")
		}
		seen |= p.Seat.Mask()
		// Match the engine-owned action, not an action reconstructed from fields.
		found := false
		for _, c := range g.Legal(p.Seat) {
			if c.Key() == p.Key {
				picks[p.Seat] = c.Action
				found = true
				break
			}
		}
		if !found {
			return *g, nil, fmt.Errorf("rules: key %q not offered to seat %d at %s", p.Key, p.Seat, point.ID())
		}
	}
	if seen != point.Required {
		return *g, nil, fmt.Errorf("rules: incomplete batch")
	}
	next, events, err := ResolveState(&g.State, &picks, nil)
	if err != nil {
		return *g, nil, err
	}
	out := *g
	out.State = next
	if g.State.Phase == PhaseTurn || g.State.Phase == PhaseLastCard {
		age, turn := g.State.Age-1, g.State.Turn-1
		for seat := uint8(0); seat < g.State.Seats; seat++ {
			if !point.Required.Has(Seat(seat)) {
				continue
			}
			a := picks[seat]
			play := Play{Kind: a.Kind, Left: a.Left, Right: a.Right}
			if a.Kind == ActBuild {
				play.Design = a.Design
			}
			if a.Kind == ActStage {
				play.Stage = g.State.Cities[seat].Stages + 1
			}
			out.Record.Picks[age][turn][seat] = a.Design
			out.Record.Plays[age][turn][seat] = play
		}
	}
	if g.State.Phase == PhaseDiscardBuild {
		out.Record.Rebuilt[g.State.Age-1][g.State.Turn-1][g.State.Decider] = picks[g.State.Decider].Design
	}
	if next.Age != g.State.Age && next.Phase == PhaseTurn {
		out.Record.Packets[next.Age-1] = next.Hands
	}
	return out, events, nil
}

func Replay(setup Setup, batches []Batch) (Game, error) {
	g, err := NewGame(setup)
	if err != nil {
		return Game{}, err
	}
	for i, b := range batches {
		next, _, err := g.Resolve(b)
		if err != nil {
			return Game{}, fmt.Errorf("rules: replay batch %d: %w", i, err)
		}
		g = next
	}
	return g, nil
}
