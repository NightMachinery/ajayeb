package rules

import (
	"fmt"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// CheckInvariants is diagnostic, not a step in ordinary resolution. Scenarios
// call it at their trust boundary. History legality will be checked by replay
// once Resolve exists; here the record's shapes and public placements are checked.
func (g *Game) CheckInvariants() error {
	if g == nil {
		return fmt.Errorf("rules: nil game")
	}
	st := &g.State
	if st.Seats < MinSeats || st.Seats > MaxSeats {
		return fmt.Errorf("rules: invalid seat count")
	}
	if err := st.Opts.Validate(); err != nil {
		return err
	}
	if st.Phase >= NumPhases || st.Points > MaxPoints {
		return fmt.Errorf("rules: invalid phase/point count")
	}
	switch st.Phase {
	case PhaseSides:
		if st.Opts.Sides != SidesChoose || st.Age != 0 || st.Turn != 0 || st.Decider != NoSeat || st.Points != 0 {
			return fmt.Errorf("rules: invalid side point")
		}
	case PhaseTurn, PhaseDiscardBuild:
		if st.Age < 1 || st.Age > NumAges || st.Turn < 1 || st.Turn > TurnsPerAge {
			return fmt.Errorf("rules: invalid Age/turn")
		}
		if (st.Phase == PhaseTurn && st.Decider != NoSeat) || (st.Phase == PhaseDiscardBuild && !st.Decider.Valid(st.Seats)) {
			return fmt.Errorf("rules: invalid decider")
		}
	case PhaseLastCard:
		if st.Age < 1 || st.Age > NumAges || st.Turn != MaxTurns || !st.Decider.Valid(st.Seats) {
			return fmt.Errorf("rules: invalid last card point")
		}
	case PhaseOver:
		if st.Age != NumAges || (st.Turn != TurnsPerAge && st.Turn != MaxTurns) || st.Decider != NoSeat {
			return fmt.Errorf("rules: invalid finished position")
		}
	}
	if err := g.Point().Validate(st.Seats); err != nil {
		return err
	}
	if st.Phase != PhaseOver && st.Result != (Result{}) {
		return fmt.Errorf("rules: premature result")
	}
	if st.Phase == PhaseOver {
		if err := checkResult(st); err != nil {
			return err
		}
	}
	var copies [catalog.NumDesigns + 1]uint16
	var boards [catalog.NumBoards + 1]bool
	for seat, c := range st.Cities {
		if seat >= int(st.Seats) {
			if c != (City{}) || st.Hands[seat] != ([HandSize]catalog.DesignID{}) {
				return fmt.Errorf("rules: nonzero unused seat %d", seat)
			}
			continue
		}
		if !c.Board.Valid() || boards[c.Board] {
			return fmt.Errorf("rules: missing/duplicate board")
		}
		boards[c.Board] = true
		if c.Coins < 0 {
			return fmt.Errorf("rules: negative coins")
		}
		if (st.Phase == PhaseSides && c.Side != catalog.NoSide) || (st.Phase != PhaseSides && c.Side != catalog.Day && c.Side != catalog.Night) {
			return fmt.Errorf("rules: invalid revealed side")
		}
		if !c.Buildings.AndNot(catalog.T.All).Empty() || c.Buildings.Count()+int(c.Stages) > 21 {
			return fmt.Errorf("rules: invalid city building set")
		}
		rebuilt, err := cityCaches(c)
		if err != nil {
			return err
		}
		if rebuilt != c {
			return fmt.Errorf("rules: seat %d cache mismatch", seat)
		}
		var names [catalog.NumNames + 1]bool
		for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
			if !c.Buildings.Has(id) {
				continue
			}
			d, ok := catalog.DesignByID(id)
			if !ok || d.Age > st.Age || names[d.NameID] {
				return fmt.Errorf("rules: invalid building Age or duplicate name")
			}
			names[d.NameID] = true
			copies[id]++
		}
		for i, id := range c.Tucked {
			if i >= int(c.Stages) {
				if id != 0 {
					return fmt.Errorf("rules: nonzero unused tucked slot")
				}
				continue
			}
			d, ok := catalog.DesignByID(id)
			if !ok || d.Age > st.Age {
				return fmt.Errorf("rules: invalid tucked card")
			}
			copies[id]++
		}
		if c.Kept != 0 {
			d, ok := catalog.DesignByID(c.Kept)
			allowed := st.Phase == PhaseDiscardBuild && st.Turn == TurnsPerAge || st.Phase == PhaseLastCard && Seat(seat) == st.Decider
			if !ok || d.Age != st.Age || !allowed || c.Abilities&AbilityPlayLastCard == 0 {
				return fmt.Errorf("rules: invalid kept card")
			}
			copies[c.Kept]++
		}
		if st.Phase == PhaseLastCard && Seat(seat) == st.Decider && c.Kept == 0 {
			return fmt.Errorf("rules: last card missing")
		}
		complete := int(st.Age) - 1
		if st.Phase == PhaseSides {
			complete = 0
		}
		if st.Phase == PhaseOver {
			complete = NumAges
		}
		if c.Defeats > uint8(2*complete) {
			return fmt.Errorf("rules: too many defeat tokens")
		}
		for age, v := range c.Victories {
			if v > 2 || (age >= complete && v != 0) {
				return fmt.Errorf("rules: invalid victory tokens")
			}
		}
		want := 0
		switch st.Phase {
		case PhaseTurn:
			want = 8 - int(st.Turn)
		case PhaseDiscardBuild:
			want = 7 - int(st.Turn)
		}
		if _, err := checkIDs(st.Hands[seat][:], st.Age, want); err != nil {
			return fmt.Errorf("rules: hand seat %d: %w", seat, err)
		}
		for _, id := range st.Hands[seat] {
			if id != 0 {
				copies[id]++
			}
		}
	}
	if st.Discard[0] != 0 {
		return fmt.Errorf("rules: discard sentinel is nonzero")
	}
	for id, n := range st.Discard {
		if n == 0 {
			continue
		}
		d, ok := catalog.DesignByID(catalog.DesignID(id))
		if !ok || d.Age > st.Age {
			return fmt.Errorf("rules: invalid discard Age")
		}
		copies[id] += uint16(n)
	}
	var box catalog.DesignSet
	if _, err := checkIDs(st.BoxGuilds[:], 3, 8-int(st.Seats)); err != nil {
		return fmt.Errorf("rules: box: %w", err)
	}
	for _, id := range st.BoxGuilds {
		if id == 0 {
			continue
		}
		d, _ := catalog.DesignByID(id)
		if d.Color != catalog.Purple || box.Has(id) {
			return fmt.Errorf("rules: invalid box guild")
		}
		box.Add(id)
		copies[id]++
	}
	for age, row := range st.Deals {
		future := age >= int(st.Age)
		if !future && row != ([MaxSeats][HandSize]catalog.DesignID{}) {
			return fmt.Errorf("rules: dealt Age still in future storage")
		}
		if future {
			if err := checkPackets(row, uint8(age+1), st.Seats, box); err != nil {
				return fmt.Errorf("rules: future deal: %w", err)
			}
		}
		for _, hand := range row {
			for _, id := range hand {
				if id != 0 {
					copies[id]++
				}
			}
		}
	}
	for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
		d, ok := catalog.DesignByID(id)
		if !ok {
			return fmt.Errorf("rules: incomplete catalog")
		}
		want := catalog.CopiesAt(id, st.Seats)
		if d.Color == catalog.Purple {
			want = 1
		}
		if copies[id] != uint16(want) {
			return fmt.Errorf("rules: design %s has %d copies, want %d", d.Slug, copies[id], want)
		}
	}
	if st.Phase == PhaseDiscardBuild && !hasDiscardChoice(st) {
		return fmt.Errorf("rules: discard point has no choice or trigger")
	}
	if err := g.checkRecord(box); err != nil {
		return err
	}
	// Public-city shapes and caches are validated above, so scoring is safe.
	if st.Phase == PhaseOver && st.Result != scoreResult(st) {
		return fmt.Errorf("rules: final scores differ from public cities")
	}
	return nil
}

func checkIDs(ids []catalog.DesignID, age uint8, want int) (int, error) {
	count := 0
	zero := false
	var prior catalog.DesignID
	for _, id := range ids {
		if id == 0 {
			zero = true
			continue
		}
		d, ok := catalog.DesignByID(id)
		if zero || !ok || id < prior || (age != 0 && d.Age != age) {
			return count, fmt.Errorf("invalid id, Age, order or padding")
		}
		prior = id
		count++
	}
	if want >= 0 && count != want {
		return count, fmt.Errorf("has %d cards, want %d", count, want)
	}
	return count, nil
}

func checkPackets(row [MaxSeats][HandSize]catalog.DesignID, age, n uint8, box catalog.DesignSet) error {
	var counts [catalog.NumDesigns + 1]uint8
	for seat, hand := range row {
		want := 0
		if seat < int(n) {
			want = HandSize
		}
		if _, err := checkIDs(hand[:], age, want); err != nil {
			return err
		}
		for _, id := range hand {
			if id != 0 {
				counts[id]++
			}
		}
	}
	for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
		d, _ := catalog.DesignByID(id)
		var want uint8
		if d.Age == age {
			want = catalog.CopiesAt(id, n)
			if d.Color == catalog.Purple && !box.Has(id) {
				want = 1
			}
		}
		if counts[id] != want {
			return fmt.Errorf("age %d packet supply differs for %s", age, d.Slug)
		}
	}
	return nil
}

func hasDiscardChoice(st *State) bool {
	c := &st.Cities[st.Decider]
	b, _ := catalog.BoardByID(c.Board)
	if c.Stages == 0 || c.Side == catalog.NoSide {
		return false
	}
	trigger := false
	for _, e := range b.Sides[c.Side-1].Stages[c.Stages-1].Effects {
		if e.Kind == catalog.EffBuildFromDiscard {
			trigger = true
		}
	}
	if !trigger {
		return false
	}
	for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
		if st.Discard[id] == 0 {
			continue
		}
		d, _ := catalog.DesignByID(id)
		if c.Buildings.And(catalog.T.ByName[d.NameID]).Empty() {
			return true
		}
	}
	return false
}

func checkResult(st *State) error {
	r := &st.Result
	if !r.Over || r.Winners == 0 || r.Winners&^uint8(AllSeats(st.Seats)) != 0 {
		return fmt.Errorf("rules: invalid final result")
	}
	for seat, score := range r.Scores {
		if seat >= int(st.Seats) {
			if score != (Score{}) || r.Rank[seat] != 0 {
				return fmt.Errorf("rules: nonzero unused result seat")
			}
			continue
		}
		sum := int32(score.Wonder) + int32(score.Treasure) + int32(score.Military) + int32(score.Blue) + int32(score.Yellow) + int32(score.Green) + int32(score.Purple)
		if sum != int32(score.Total) {
			return fmt.Errorf("rules: score total differs")
		}
		rank := uint8(1)
		for other := uint8(0); other < st.Seats; other++ {
			os := r.Scores[other]
			if os.Total > score.Total || os.Total == score.Total && st.Cities[other].Coins > st.Cities[seat].Coins {
				rank++
			}
		}
		if r.Rank[seat] != rank || ((r.Winners&(1<<seat)) != 0) != (rank == 1) {
			return fmt.Errorf("rules: invalid score rank/winner")
		}
	}
	return nil
}

func (g *Game) checkRecord(box catalog.DesignSet) error {
	st := &g.State
	for age, row := range g.Record.Packets {
		if age >= int(st.Age) {
			if row != ([MaxSeats][HandSize]catalog.DesignID{}) {
				return fmt.Errorf("rules: premature record packet")
			}
		} else if err := checkPackets(row, uint8(age+1), st.Seats, box); err != nil {
			return fmt.Errorf("rules: record packet: %w", err)
		}
	}
	var picked [NumAges][catalog.NumDesigns + 1]uint8
	for age, turns := range g.Record.Picks {
		for turn, seats := range turns {
			for seat, id := range seats {
				p := g.Record.Plays[age][turn][seat]
				allowed := age < int(st.Age)-1 || age == int(st.Age)-1 && (turn < int(st.Turn)-1 || st.Phase == PhaseDiscardBuild && turn < int(st.Turn) || st.Phase == PhaseOver && turn < int(st.Turn))
				if st.Phase == PhaseLastCard && age == int(st.Age)-1 {
					allowed = turn < TurnsPerAge
				}
				if seat >= int(st.Seats) || !allowed {
					if id != 0 || p != (Play{}) {
						return fmt.Errorf("rules: future/unused record pick")
					}
					continue
				}
				if id == 0 {
					if p != (Play{}) {
						return fmt.Errorf("rules: play missing private pick")
					}
					continue
				}
				d, ok := catalog.DesignByID(id)
				if !ok || d.Age != uint8(age+1) {
					return fmt.Errorf("rules: invalid private record pick")
				}
				picked[age][id]++
				switch p.Kind {
				case ActBuild:
					if p.Design != id || p.Stage != 0 || !st.Cities[seat].Buildings.Has(id) {
						return fmt.Errorf("rules: invalid public build record")
					}
				case ActStage:
					if p.Design != 0 || p.Stage < 1 || p.Stage > st.Cities[seat].Stages || st.Cities[seat].Tucked[p.Stage-1] != id {
						return fmt.Errorf("rules: invalid public stage record")
					}
				case ActSell:
					if p.Design != 0 || p.Stage != 0 || p.Left != 0 || p.Right != 0 {
						return fmt.Errorf("rules: invalid public sale record")
					}
				default:
					return fmt.Errorf("rules: invalid recorded action kind")
				}
				if uint16(p.Left)+uint16(p.Right) > 2*catalog.MaxCostUnits {
					return fmt.Errorf("rules: invalid recorded payment")
				}
				if turn == MaxTurns-1 && st.Cities[seat].Abilities&AbilityPlayLastCard == 0 {
					return fmt.Errorf("rules: last-card record outside ability seat")
				}
			}
		}
	}
	for age, row := range picked {
		for id, n := range row {
			if n == 0 {
				continue
			}
			d, _ := catalog.DesignByID(catalog.DesignID(id))
			want := catalog.CopiesAt(catalog.DesignID(id), st.Seats)
			if d.Color == catalog.Purple {
				want = 1
			}
			if n > want {
				return fmt.Errorf("rules: Age %d record uses a copy twice", age+1)
			}
		}
	}
	for age, turns := range g.Record.Rebuilt {
		for turn, seats := range turns {
			for seat, id := range seats {
				if id == 0 {
					continue
				}
				allowed := age < int(st.Age)-1 || age == int(st.Age)-1 && (turn < int(st.Turn)-1 || st.Phase == PhaseLastCard || st.Phase == PhaseOver)
				if seat >= int(st.Seats) || !allowed {
					return fmt.Errorf("rules: future/unused rebuild record")
				}
				d, ok := catalog.DesignByID(id)
				if !ok || d.Age > uint8(age+1) || !st.Cities[seat].Buildings.Has(id) {
					return fmt.Errorf("rules: invalid rebuilt design")
				}
			}
		}
	}
	return nil
}
