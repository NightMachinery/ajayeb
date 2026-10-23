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
	if st.Phase != PhaseTurn || st.Age < 1 || st.Age > NumAges || st.Turn < 1 || st.Turn > TurnsPerAge {
		return *st, evs, fmt.Errorf("rules: no ordinary turn")
	}
	var order [MaxSeats]Seat
	for i := uint8(0); i < st.Seats; i++ {
		order[i] = Seat(i)
		if reverse {
			order[i] = Seat(st.Seats - 1 - i)
		}
	}
	for i := st.Seats; i < MaxSeats; i++ {
		if picks[i] != (Action{}) {
			return *st, evs, fmt.Errorf("rules: pick outside table")
		}
	}
	next := *st
	var terms [MaxSeats]construction
	var choose, play, income, placedIncome [MaxSeats]Event
	var delta [MaxSeats]int32
	// 1. Choose. The chosen design leaves exactly once; sixth-turn leftovers
	// enter the pile before placement. Babylon's kept-card path is a later slice.
	for _, seat := range order[:st.Seats] {
		a := picks[seat]
		if !a.Valid() || (a.Kind != ActBuild && a.Kind != ActStage && a.Kind != ActSell) {
			return *st, evs, fmt.Errorf("rules: invalid turn action")
		}
		if !takeCard(&next.Hands[seat], a.Design) {
			return *st, evs, fmt.Errorf("rules: chosen card missing from hand")
		}
		if st.Turn == TurnsPerAge {
			leftover := next.Hands[seat][0]
			if leftover == 0 || next.Hands[seat][1] != 0 {
				return *st, evs, fmt.Errorf("rules: invalid sixth-turn hand")
			}
			next.Discard[leftover]++
			next.Hands[seat] = [HandSize]catalog.DesignID{}
			choose[seat] = Event{Kind: EvLeftoverDiscarded, Seat: seat, Design: leftover, Private: true}
		}
		// 2. Check is the caller's frozen-position membership check. Resolve still
		// validates the union/placement bounds before indexing or changing a value.
		if a.Kind != ActSell {
			q, ok := constructionFor(st, seat, a.Kind, a.Design)
			if !ok {
				return *st, evs, fmt.Errorf("rules: invalid construction")
			}
			terms[seat] = q
		}
	}
	// 3. Pay. Every debit is derived from F, never same-turn receipts or caches.
	// Deltas commute even when two neighbours buy the same physical symbol.
	for _, seat := range order[:st.Seats] {
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
	for _, seat := range order[:st.Seats] {
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
		coins := int32(st.Cities[seat].Coins) + delta[seat]
		if coins < 0 || coins > 32767 {
			return *st, evs, fmt.Errorf("rules: coin overflow")
		}
		c.Coins = int16(coins)
	}
	for _, seat := range order[:st.Seats] {
		c, err := cityCaches(next.Cities[seat])
		if err != nil {
			return *st, evs, err
		}
		next.Cities[seat] = c
	}
	// 5. Count the fully placed public cities, never the frozen or partially
	// placed cities. Earlier coin effects cannot change card/stage counts.
	for _, seat := range order[:st.Seats] {
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
	// 6. Discard-build/last-card scheduling remains the wonder-abilities slice.
	// This intermediate engine is not releasable until that slice lands.
	var automatic [MaxSeats + 1]Event
	count := 0
	// 7. Pass or end the Age, then deal or close the game automatically.
	if st.Turn < TurnsPerAge {
		var passed [MaxSeats][HandSize]catalog.DesignID
		for _, seat := range order[:st.Seats] {
			destination := seat.Left(st.Seats)
			if st.Age == 2 {
				destination = seat.Right(st.Seats)
			}
			passed[destination] = next.Hands[seat]
		}
		next.Hands = passed
		next.Turn++
		automatic[0] = Event{Kind: EvHandsPassed, Age: st.Age, Turn: st.Turn}
		count = 1
	} else {
		// Compare each edge once; each city takes part in exactly two comparisons.
		for _, seat := range order[:st.Seats] {
			other := seat.Left(st.Seats)
			event := Event{Kind: EvMilitary, Seat: seat, Other: other, Age: st.Age}
			switch {
			case next.Cities[seat].Shields > next.Cities[other].Shields:
				event.Outcome = ConflictSeatWins
				next.Cities[seat].Victories[st.Age-1]++
				next.Cities[other].Defeats++
			case next.Cities[seat].Shields < next.Cities[other].Shields:
				event.Outcome = ConflictOtherWins
				next.Cities[other].Victories[st.Age-1]++
				next.Cities[seat].Defeats++
			}
			automatic[seat] = event
		}
		count = int(st.Seats)
		if st.Age < NumAges {
			// The temporary Game has no history. Only this State transfer matters to
			// search; the checked Game wrapper records the newly dealt packet.
			transfer := Game{State: next}
			if err := dealAge(&transfer, st.Age+1); err != nil {
				return *st, evs, err
			}
			next = transfer.State
			automatic[count] = Event{Kind: EvAgeStart, Age: next.Age}
		} else {
			finishScored(&next)
			automatic[count] = Event{Kind: EvGameOver}
		}
		count++
	}
	next.Points++
	for _, bucket := range [4][MaxSeats]Event{choose, play, income, placedIncome} {
		for _, e := range bucket[:st.Seats] {
			if e.Kind != EvNone {
				evs = append(evs, e)
			}
		}
	}
	evs = append(evs, automatic[:count]...)
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
