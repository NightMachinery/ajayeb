package rules

import (
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

func oneRecordedTurn(t *testing.T) Game {
	t.Helper()
	g, err := NewGame(Setup{Seats: 7, Seed: 99, Options: Options{Sides: SidesDay}})
	if err != nil {
		t.Fatal(err)
	}
	next, _, err := g.Resolve(salesBatch(g))
	if err != nil {
		t.Fatal(err)
	}
	return next
}
func rejectsHistory(t *testing.T, g Game, reason string) {
	t.Helper()
	if err := g.CheckInvariants(); err == nil || !strings.Contains(err.Error(), reason) {
		t.Fatal("corruption accepted or rejected for another reason", reason, err)
	}
	if _, err := NewGameFromScenario(g.State.Seed, g.State.Opts, ScenarioFromGame(g)); err == nil {
		t.Fatal("scenario builder accepted corrupt history")
	}
}
func TestHistoryPacketOrigins(t *testing.T) {
	t.Run("opening-hand-swap", func(t *testing.T) {
		g, err := NewGame(Setup{Seats: 7, Seed: 99, Options: Options{Sides: SidesDay}})
		if err != nil {
			t.Fatal(err)
		}
		g.State.Hands[0], g.State.Hands[1] = g.State.Hands[1], g.State.Hands[0]
		rejectsHistory(t, g, "packet")
	})
	t.Run("packet-row-swap", func(t *testing.T) {
		g, err := NewGame(Setup{Seats: 7, Seed: 99, Options: Options{Sides: SidesDay}})
		if err != nil {
			t.Fatal(err)
		}
		g.Record.Packets[0][0], g.Record.Packets[0][1] = g.Record.Packets[0][1], g.Record.Packets[0][0]
		rejectsHistory(t, g, "packet")
	})
	t.Run("alien-pick", func(t *testing.T) {
		g := oneRecordedTurn(t)
		var own catalog.DesignSet
		for _, id := range g.Record.Packets[0][0] {
			own.Add(id)
		}
		for _, id := range g.Record.Packets[0][3] {
			if !own.Has(id) {
				g.Record.Picks[0][0][0] = id
				rejectsHistory(t, g, "packet")
				return
			}
		}
		t.Fatal("fixture has no foreign card")
	})
	t.Run("pick-plus-hand-exceeds-packet", func(t *testing.T) {
		g := oneRecordedTurn(t)
		for _, id := range g.State.Hands[1] {
			if id == 0 {
				continue
			}
			n := 0
			for _, other := range g.Record.Packets[0][0] {
				if other == id {
					n++
				}
			}
			if n == 1 {
				g.Record.Picks[0][0][0] = id
				rejectsHistory(t, g, "packet")
				return
			}
		}
		t.Fatal("fixture has no unique packet card")
	})
	t.Run("box-guild-pick", func(t *testing.T) {
		g := scenarioPhases(t)[4]
		id := g.State.BoxGuilds[0]
		if id == 0 {
			t.Fatal("box fixture")
		}
		g.Record.Picks[2][0][0] = id
		g.Record.Plays[2][0][0] = Play{Kind: ActSell}
		rejectsHistory(t, g, "packet")
	})
}
func TestHistoryPointCounts(t *testing.T) {
	for _, g := range scenarioPhases(t) {
		if g.State.Points > 0 {
			g.State.Points--
			rejectsHistory(t, g, "point count")
		}
	}
	g := oneRecordedTurn(t)
	g.State.Points++
	rejectsHistory(t, g, "point count")
	// Partial histories preserve unknown earlier points, but never go below the
	// phase floor or invent more known uses of a packet than it contains.
	g.Record.Picks[0][0][0] = 0
	g.Record.Plays[0][0][0] = Play{}
	if err := g.CheckInvariants(); err != nil {
		t.Fatal("partial history rejected", err)
	}
}
func TestHistoryMilitaryBalance(t *testing.T) {
	g, err := NewGame(Setup{Seats: 7, Seed: 99, Options: Options{Sides: SidesDay}})
	if err != nil {
		t.Fatal(err)
	}
	for turn := 0; turn < 6; turn++ {
		g, _, err = g.Resolve(salesBatch(g))
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, victory := range []bool{false, true} {
		bad := g
		if victory {
			bad.State.Cities[0].Victories[0]++
		} else {
			bad.State.Cities[0].Defeats++
		}
		rejectsHistory(t, bad, "victory/defeat")
	}
}
