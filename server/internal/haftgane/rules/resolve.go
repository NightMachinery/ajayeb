package rules

import (
	"fmt"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// ResolveState is the state-only search core. Its caller guarantees that every
// action came from LegalState on this same frozen State. It does not call Legal
// or the payment solver. Game.Resolve is the checked external entry point.
// On failure the original state and the caller's event prefix are returned.
func ResolveState(st *State, picks *[MaxSeats]Action, evs []Event) (State, []Event, error) {
	return resolveStateOrder(st, picks, evs, false)
}

// reverse is an internal test seam: every independent seat calculation can run
// backwards; events are still emitted by pipeline step and ascending seat.
func resolveStateOrder(st *State, picks *[MaxSeats]Action, evs []Event, reverse bool) (State, []Event, error) {
	if st == nil {
		return State{}, evs, fmt.Errorf("rules: nil state")
	}
	if picks == nil || AllSeats(st.Seats) == 0 || st.Points >= MaxPoints {
		return *st, evs, fmt.Errorf("rules: invalid resolution input")
	}
	if st.Phase == PhaseSides {
		next, events, err := resolveSides(Game{State: *st}, *picks)
		if err != nil {
			return *st, evs, err
		}
		return next.State, append(evs, events...), nil
	}
	if st.Phase == PhaseDiscardBuild {
		return resolveDiscardState(st, picks, evs, reverse)
	}
	last := st.Phase == PhaseLastCard
	if st.Age < 1 || st.Age > NumAges || (!last && (st.Phase != PhaseTurn || st.Turn < 1 || st.Turn > TurnsPerAge)) || (last && (st.Turn != MaxTurns || !st.Decider.Valid(st.Seats))) {
		return *st, evs, fmt.Errorf("rules: no card point")
	}
	required := AllSeats(st.Seats)
	if last {
		required = st.Decider.Mask()
	}
	var order [MaxSeats]Seat
	for i := uint8(0); i < st.Seats; i++ {
		order[i] = Seat(i)
		if reverse {
			order[i] = Seat(st.Seats - 1 - i)
		}
	}
	var acting [MaxSeats]Seat
	active := 0
	for _, seat := range order[:st.Seats] {
		if required.Has(seat) {
			acting[active] = seat
			active++
		}
	}
	for seat, a := range picks {
		if !required.Has(Seat(seat)) && a != (Action{}) {
			return *st, evs, fmt.Errorf("rules: pick outside required seats")
		}
	}
	next := *st
	var terms [MaxSeats]construction
	var choose, play, income, placedIncome [MaxSeats]Event
	var delta [MaxSeats]int32
	// 1. Choose using F, including an ability stage being built this turn.
	// 2. The caller checked membership; validate placement bounds here.
	for _, seat := range acting[:active] {
		a := picks[seat]
		if !a.Valid() || (a.Kind != ActBuild && a.Kind != ActStage && a.Kind != ActSell) {
			return *st, evs, fmt.Errorf("rules: invalid card action")
		}
		if a.Kind != ActSell {
			q, ok := constructionFor(st, seat, a.Kind, a.Design)
			if !ok {
				return *st, evs, fmt.Errorf("rules: invalid construction")
			}
			terms[seat] = q
		}
		if last {
			if next.Cities[seat].Kept != a.Design {
				return *st, evs, fmt.Errorf("rules: chosen card is not kept")
			}
			next.Cities[seat].Kept = 0
		} else {
			if !takeCard(&next.Hands[seat], a.Design) {
				return *st, evs, fmt.Errorf("rules: chosen card missing from hand")
			}
			if st.Turn == TurnsPerAge {
				leftover := next.Hands[seat][0]
				if leftover == 0 || next.Hands[seat][1] != 0 {
					return *st, evs, fmt.Errorf("rules: invalid sixth-turn hand")
				}
				keep := st.Cities[seat].Abilities&AbilityPlayLastCard != 0 || a.Kind == ActStage && stageHasEffect(&st.Cities[seat], terms[seat].stage, catalog.EffPlayLastCard)
				event := Event{Kind: EvLeftoverDiscarded, Seat: seat, Design: leftover, Private: true}
				if keep {
					next.Cities[seat].Kept = leftover
					event.Kind = EvCardKept
				} else {
					next.Discard[leftover]++
				}
				next.Hands[seat] = [HandSize]catalog.DesignID{}
				choose[seat] = event
			}
		}
	}
	// 3. Pay. Every debit is derived from F, never same-turn receipts or caches.
	// Deltas commute even when two neighbours buy the same physical symbol.
	for _, seat := range acting[:active] {
		a := picks[seat]
		q := terms[seat]
		printed := q.cost.Coins
		if q.free != FreeNone {
			printed = 0
			if a.Left != 0 || a.Right != 0 {
				return *st, evs, fmt.Errorf("rules: nonzero free payment")
			}
		}
		total := int32(printed) + int32(a.Left) + int32(a.Right)
		if total > int32(st.Cities[seat].Coins) {
			return *st, evs, fmt.Errorf("rules: payment exceeds frozen purse")
		}
		delta[seat] -= total
		delta[seat.Left(st.Seats)] += int32(a.Left)
		delta[seat.Right(st.Seats)] += int32(a.Right)
		play[seat] = Event{Kind: EvPlay, Seat: seat, Action: a.Kind, Design: a.Design, Stage: q.stage, Left: a.Left, Right: a.Right, Printed: printed, Private: a.Kind != ActBuild}
	}
	// 4. Place all cards before recomputing caches. Selling remains ordinary
	// placement income, distinct from the when-placed effect pipeline.
	for _, seat := range acting[:active] {
		a := picks[seat]
		c := &next.Cities[seat]
		switch a.Kind {
		case ActBuild:
			c.Buildings.Add(a.Design)
		case ActStage:
			c.Tucked[c.Stages] = a.Design
			c.Stages++
		case ActSell:
			next.Discard[a.Design]++
			delta[seat] += 3
			income[seat] = Event{Kind: EvCoins, Seat: seat, Source: CoinSale, Delta: 3}
		}
	}
	// Credit all recipients, including neighbours outside a one-seat point.
	for _, seat := range order[:st.Seats] {
		coins := int32(st.Cities[seat].Coins) + delta[seat]
		if coins < 0 || coins > 32767 {
			return *st, evs, fmt.Errorf("rules: coin overflow")
		}
		next.Cities[seat].Coins = int16(coins)
		c, err := cityCaches(next.Cities[seat])
		if err != nil {
			return *st, evs, err
		}
		next.Cities[seat] = c
	}
	// 5. Count the fully placed public cities, never the frozen or partially
	// placed cities. Earlier coin effects cannot change card/stage counts.
	for _, seat := range acting[:active] {
		a := picks[seat]
		var effects []catalog.Effect
		event := Event{Kind: EvCoins, Seat: seat}
		switch a.Kind {
		case ActBuild:
			d, _ := catalog.DesignByID(a.Design)
			effects = d.Effects
			event.Source, event.Design = CoinDesign, a.Design
		case ActStage:
			c := &next.Cities[seat]
			board, _ := catalog.BoardByID(c.Board)
			effects = board.Sides[c.Side-1].Stages[c.Stages-1].Effects
			event.Source, event.Stage = CoinStage, c.Stages
		default:
			continue
		}
		amount, err := applyWhenPlaced(&next, seat, effects)
		if err != nil {
			return *st, evs, err
		}
		if amount != 0 {
			event.Delta = amount
			placedIncome[seat] = event
		}
	}
	// 6. Every discard stage makes a pile look, even if the name rule leaves
	// nothing legal. The stage play plus its rebuilt/no-build outcome records
	// this fact for history/knowledge reconstruction without another state field.
	var automatic [MaxSeats + 2]Event
	tail := automatic[:0]
	opened := false
	for _, seat := range acting[:active] {
		a := picks[seat]
		if a.Kind != ActStage || !stageHasEffect(&next.Cities[seat], terms[seat].stage, catalog.EffBuildFromDiscard) {
			continue
		}
		next.Phase, next.Decider = PhaseDiscardBuild, seat
		if hasDiscardChoice(&next) {
			opened = true
		} else {
			next.Decider = NoSeat
			tail = append(tail, Event{Kind: EvDiscardBuildNothing, Seat: seat})
		}
	}
	if !opened {
		var err error
		tail, err = advanceAfterTurn(&next, tail, reverse)
		if err != nil {
			return *st, evs, err
		}
	}
	next.Points++
	for _, bucket := range [4][MaxSeats]Event{choose, play, income, placedIncome} {
		for _, e := range bucket[:st.Seats] {
			if e.Kind != EvNone {
				evs = append(evs, e)
			}
		}
	}
	evs = append(evs, tail...)
	return next, evs, nil
}

func takeCard(hand *[HandSize]catalog.DesignID, id catalog.DesignID) bool {
	for i, d := range hand {
		if d == id {
			copy(hand[i:], hand[i+1:])
			hand[HandSize-1] = 0
			return true
		}
	}
	return false
}
