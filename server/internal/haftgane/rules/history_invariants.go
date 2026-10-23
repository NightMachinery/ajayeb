package rules

import (
	"fmt"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// packetOrigin follows the packet, not the chosen card. Babylon's seventh
// play still comes from its sixth packet, which was not passed again.
func packetOrigin(seat Seat, n, age, turn uint8) Seat {
	shift := min(turn, TurnsPerAge) - 1
	if age == 2 {
		return Seat((uint8(seat) + shift) % n)
	}
	return Seat((uint8(seat) + n - shift%n) % n)
}

// Shape checks precede this: all ids, seats and packet rows are valid. Unknown
// past picks remain zero in partial histories; every known pick and current
// card must still fit its own packet's remaining multiset.
func (g *Game) checkHistory() error {
	st := &g.State
	if st.Phase == PhaseSides {
		return nil
	}
	var remaining [NumAges][MaxSeats][catalog.NumDesigns + 1]uint8
	for age, row := range g.Record.Packets {
		for origin, packet := range row {
			for _, id := range packet {
				if id != 0 {
					remaining[age][origin][id]++
				}
			}
		}
	}
	consume := func(age int, origin Seat, id catalog.DesignID) error {
		if remaining[age][origin][id] == 0 {
			return fmt.Errorf("rules: card %d exceeds packet age %d origin %d", id, age+1, origin)
		}
		remaining[age][origin][id]--
		return nil
	}
	ordinary := uint16(TurnsPerAge) * uint16(st.Age-1)
	current := uint8(0)
	switch st.Phase {
	case PhaseTurn:
		current = st.Turn - 1
	case PhaseDiscardBuild:
		current = st.Turn
	case PhaseLastCard, PhaseOver:
		current = TurnsPerAge
	}
	ordinary += uint16(current)
	side := uint16(0)
	if st.Opts.Sides == SidesChoose {
		side = 1
	}
	extra := uint16(0)
	complete := true
	for age, row := range g.Record.Picks {
		for turn, seats := range row {
			resolved := age < int(st.Age)-1 || age == int(st.Age)-1 && turn < int(current)
			for seat := Seat(0); seat < Seat(st.Seats); seat++ {
				id := seats[seat]
				if turn < TurnsPerAge && resolved && id == 0 {
					complete = false
				}
				if id == 0 {
					continue
				}
				origin := packetOrigin(seat, st.Seats, uint8(age+1), uint8(turn+1))
				if err := consume(age, origin, id); err != nil {
					return err
				}
				if turn == MaxTurns-1 {
					extra++
				}
			}
		}
	}
	for _, row := range g.Record.Rebuilt {
		for _, seats := range row {
			for _, id := range seats {
				if id != 0 {
					extra++
				}
			}
		}
	}
	for seat := Seat(0); seat < Seat(st.Seats); seat++ {
		origin := packetOrigin(seat, st.Seats, st.Age, st.Turn)
		for _, id := range st.Hands[seat] {
			if id != 0 {
				if err := consume(int(st.Age-1), origin, id); err != nil {
					return err
				}
			}
		}
		if id := st.Cities[seat].Kept; id != 0 {
			if err := consume(int(st.Age-1), origin, id); err != nil {
				return err
			}
		}
	}
	want := side + ordinary + extra
	if st.Points < want || complete && st.Points != want {
		return fmt.Errorf("rules: point count %d inconsistent with history (minimum %d, complete %t)", st.Points, want, complete)
	}
	var wins, losses uint16
	for _, c := range st.Cities[:st.Seats] {
		for _, v := range c.Victories {
			wins += uint16(v)
		}
		losses += uint16(c.Defeats)
	}
	if wins != losses {
		return fmt.Errorf("rules: victory/defeat totals differ: %d/%d", wins, losses)
	}
	return nil
}
