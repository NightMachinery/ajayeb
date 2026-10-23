package rules

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// Forced scenarios choose their own deal, not necessarily the seed's original
// packets. Allocate that explicit deal around the known picks and current
// hands, retaining the exact Age supply. Real playout tests never call this.
func alignScenarioHistory(t *testing.T, g *Game) {
	t.Helper()
	if g.State.Phase == PhaseSides {
		return
	}
	age := g.State.Age - 1
	var supply [catalog.NumDesigns + 1]uint8
	for _, packet := range g.Record.Packets[age] {
		for _, id := range packet {
			if id != 0 {
				supply[id]++
			}
		}
	}
	var packets [MaxSeats][HandSize]catalog.DesignID
	var sizes [MaxSeats]int
	add := func(origin Seat, id catalog.DesignID) {
		if supply[id] == 0 || sizes[origin] == HandSize {
			t.Fatal("forced scenario packet cannot hold known cards", origin, id)
		}
		packets[origin][sizes[origin]] = id
		sizes[origin]++
		supply[id]--
	}
	for turn, seats := range g.Record.Picks[age] {
		for seat, id := range seats {
			if id != 0 {
				add(packetOrigin(Seat(seat), g.State.Seats, g.State.Age, uint8(turn+1)), id)
			}
		}
	}
	for seat := Seat(0); seat < Seat(g.State.Seats); seat++ {
		origin := packetOrigin(seat, g.State.Seats, g.State.Age, g.State.Turn)
		for _, id := range g.State.Hands[seat] {
			if id != 0 {
				add(origin, id)
			}
		}
		if id := g.State.Cities[seat].Kept; id != 0 {
			add(origin, id)
		}
	}
	for origin := Seat(0); origin < Seat(g.State.Seats); origin++ {
		for id := catalog.DesignID(1); id <= catalog.NumDesigns && sizes[origin] < HandSize; id++ {
			for supply[id] > 0 && sizes[origin] < HandSize {
				add(origin, id)
			}
		}
		canonicalIDs(packets[origin][:])
	}
	g.Record.Packets[age] = packets
	points := uint16(6) * uint16(g.State.Age-1)
	if g.State.Opts.Sides == SidesChoose {
		points++
	}
	switch g.State.Phase {
	case PhaseTurn:
		points += uint16(g.State.Turn - 1)
	case PhaseDiscardBuild:
		points += uint16(g.State.Turn)
	case PhaseLastCard, PhaseOver:
		points += 6
	}
	for _, row := range g.Record.Rebuilt {
		for _, seats := range row {
			for _, id := range seats {
				if id != 0 {
					points++
				}
			}
		}
	}
	for _, row := range g.Record.Picks {
		for _, id := range row[MaxTurns-1] {
			if id != 0 {
				points++
			}
		}
	}
	g.State.Points = points
}
