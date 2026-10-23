package rules

import (
	"fmt"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

func stageHasEffect(c *City, stage uint8, kind catalog.EffectKind) bool {
	b, ok := catalog.BoardByID(c.Board)
	if !ok || c.Side < catalog.Day || c.Side > catalog.Night || stage < 1 || stage > b.Sides[c.Side-1].NumStages {
		return false
	}
	for _, e := range b.Sides[c.Side-1].Stages[stage-1].Effects {
		if e.Kind == kind {
			return true
		}
	}
	return false
}

func resolveDiscardState(st *State, picks *[MaxSeats]Action, evs []Event, reverse bool) (State, []Event, error) {
	if st.Age < 1 || st.Age > NumAges || st.Turn < 1 || st.Turn > TurnsPerAge || !st.Decider.Valid(st.Seats) {
		return *st, evs, fmt.Errorf("rules: invalid discard point")
	}
	seat := st.Decider
	a := picks[seat]
	for s, other := range picks {
		if Seat(s) != seat && other != (Action{}) {
			return *st, evs, fmt.Errorf("rules: pick outside discard decider")
		}
	}
	d, ok := catalog.DesignByID(a.Design)
	if !ok || !a.Valid() || a.Kind != ActBuildDiscarded || st.Discard[a.Design] == 0 || !st.Cities[seat].Buildings.And(catalog.T.ByName[d.NameID]).Empty() {
		return *st, evs, fmt.Errorf("rules: invalid discard construction")
	}
	next := *st
	next.Discard[a.Design]--
	next.Cities[seat].Buildings.Add(a.Design)
	city, err := cityCaches(next.Cities[seat])
	if err != nil {
		return *st, evs, err
	}
	next.Cities[seat] = city
	amount, err := applyWhenPlaced(&next, seat, d.Effects)
	if err != nil {
		return *st, evs, err
	}
	var buf [MaxSeats + 3]Event
	tail := append(buf[:0], Event{Kind: EvDiscardBuild, Seat: seat, Design: a.Design})
	if amount != 0 {
		tail = append(tail, Event{Kind: EvCoins, Seat: seat, Design: a.Design, Source: CoinDesign, Delta: amount})
	}
	tail, err = advanceAfterTurn(&next, tail, reverse)
	if err != nil {
		return *st, evs, err
	}
	next.Points++
	return next, append(evs, tail...), nil
}

// advanceAfterTurn resumes only after the turn's discard opportunity. Kept
// cards are independent points with a fresh frozen position, before military.
func advanceAfterTurn(st *State, evs []Event, reverse bool) ([]Event, error) {
	st.Decider = NoSeat
	if st.Turn < TurnsPerAge {
		var passed [MaxSeats][HandSize]catalog.DesignID
		for seat := Seat(0); seat < Seat(st.Seats); seat++ {
			destination := seat.Left(st.Seats)
			if st.Age == 2 {
				destination = seat.Right(st.Seats)
			}
			passed[destination] = st.Hands[seat]
		}
		st.Hands = passed
		evs = append(evs, Event{Kind: EvHandsPassed, Age: st.Age, Turn: st.Turn})
		st.Phase = PhaseTurn
		st.Turn++
		return evs, nil
	}
	for seat, c := range st.Cities {
		if c.Kept == 0 {
			continue
		}
		if st.Decider != NoSeat {
			return evs, fmt.Errorf("rules: multiple kept-card owners")
		}
		st.Decider = Seat(seat)
	}
	if st.Decider != NoSeat {
		st.Phase, st.Turn = PhaseLastCard, MaxTurns
		return evs, nil
	}
	var conflicts [MaxSeats]Event
	for i := uint8(0); i < st.Seats; i++ {
		seat := Seat(i)
		if reverse {
			seat = Seat(st.Seats - 1 - i)
		}
		other := seat.Left(st.Seats)
		e := Event{Kind: EvMilitary, Seat: seat, Other: other, Age: st.Age}
		switch {
		case st.Cities[seat].Shields > st.Cities[other].Shields:
			e.Outcome = ConflictSeatWins
			st.Cities[seat].Victories[st.Age-1]++
			st.Cities[other].Defeats++
		case st.Cities[seat].Shields < st.Cities[other].Shields:
			e.Outcome = ConflictOtherWins
			st.Cities[other].Victories[st.Age-1]++
			st.Cities[seat].Defeats++
		}
		conflicts[seat] = e
	}
	evs = append(evs, conflicts[:st.Seats]...)
	if st.Age < NumAges {
		transfer := Game{State: *st}
		if err := dealAge(&transfer, st.Age+1); err != nil {
			return evs, err
		}
		*st = transfer.State
		evs = append(evs, Event{Kind: EvAgeStart, Age: st.Age})
	} else {
		finishScored(st)
		evs = append(evs, Event{Kind: EvGameOver})
	}
	return evs, nil
}
