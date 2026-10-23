package rules

import (
	"fmt"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/view"
)

type recordVisitor func(before *State, picks *[MaxSeats]Action, after *State, evs []Event)

// A materialized setup makes history independent of the setup root and of
// undealt hidden cards. It also supports valid explicit scenario deals. Missing
// resolved records in a partial scenario are reported, never guessed.
func (g *Game) recordStart() (State, error) {
	if g == nil || AllSeats(g.State.Seats) == 0 {
		return State{}, fmt.Errorf("rules: invalid history game")
	}
	st := State{Opts: g.State.Opts, Seats: g.State.Seats, Decider: NoSeat}
	for seat := Seat(0); seat < Seat(st.Seats); seat++ {
		side := g.State.Cities[seat].Side
		if st.Opts.Sides == SidesChoose {
			side = 0
		}
		city, err := cityCaches(City{Board: g.State.Cities[seat].Board, Side: side, Coins: 3})
		if err != nil {
			return State{}, err
		}
		st.Cities[seat] = city
	}
	for age := 0; age < NumAges; age++ {
		st.Deals[age] = g.Record.Packets[age]
		if st.Deals[age] == ([MaxSeats][HandSize]catalog.DesignID{}) {
			st.Deals[age] = g.State.Deals[age]
		}
	}
	if st.Opts.Sides != SidesChoose {
		transfer := Game{State: st}
		if err := dealAge(&transfer, 1); err != nil {
			return State{}, err
		}
		st = transfer.State
	}
	return st, nil
}

func (g *Game) walkRecord(visit recordVisitor) error {
	st, err := g.recordStart()
	if err != nil {
		return err
	}
	if g.State.Points > MaxPoints {
		return fmt.Errorf("rules: history exceeds point bound")
	}
	for step := uint16(0); step < g.State.Points; step++ {
		before := st
		point := (&Game{State: st}).Point()
		if point.Kind == PointNone {
			return fmt.Errorf("rules: history continues past game end")
		}
		var picks [MaxSeats]Action
		for seat := Seat(0); seat < Seat(st.Seats); seat++ {
			if !point.Required.Has(seat) {
				continue
			}
			switch st.Phase {
			case PhaseSides:
				picks[seat] = Action{Kind: ActSide, Side: g.State.Cities[seat].Side}
			case PhaseTurn, PhaseLastCard:
				p := g.Record.Plays[st.Age-1][st.Turn-1][seat]
				picks[seat] = Action{Kind: p.Kind, Design: g.Record.Picks[st.Age-1][st.Turn-1][seat], Left: p.Left, Right: p.Right}
			case PhaseDiscardBuild:
				picks[seat] = Action{Kind: ActBuildDiscarded, Design: g.Record.Rebuilt[st.Age-1][st.Turn-1][seat]}
			}
			if !picks[seat].Valid() {
				return fmt.Errorf("rules: incomplete record at %s seat %d", point.ID(), seat)
			}
		}
		var buf [32]Event
		next, events, err := ResolveState(&st, &picks, buf[:0])
		if err != nil {
			return fmt.Errorf("rules: history %s: %w", point.ID(), err)
		}
		if visit != nil {
			visit(&before, &picks, &next, events)
		}
		st = next
	}
	return nil
}

// History preserves the live entitlement of every event, even after an end
// reveal. Revelation is a separate view field, not a retroactive event filter.
func (g *Game) History(viewer Seat) ([]view.HistoryEntry, error) {
	out := make([]view.HistoryEntry, 0, MaxPoints+1)
	if g == nil {
		return out, fmt.Errorf("rules: nil history game")
	}
	if g.State.Opts.Sides != SidesChoose {
		out = append(out, view.HistoryEntry{Point: "age-1-start", Events: FilterEvents([]Event{{Kind: EvAgeStart, Age: 1}}, viewer)})
	}
	err := g.walkRecord(func(before *State, _ *[MaxSeats]Action, _ *State, evs []Event) {
		out = append(out, view.HistoryEntry{Point: (&Game{State: *before}).Point().ID(), Events: FilterEvents(evs, viewer)})
	})
	if err != nil {
		return []view.HistoryEntry{}, err
	}
	return out, nil
}
