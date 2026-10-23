package rules

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

var updateSetupGoldens = flag.Bool("update-haftgane-setup", false, "rewrite Haftgane setup goldens")

func setupPacks(g Game) [NumAges][MaxSeats][HandSize]catalog.DesignID {
	packs := g.State.Deals
	if g.State.Phase != PhaseSides {
		packs[0] = g.Record.Packets[0]
	}
	return packs
}

// rules.md Setup: 7N cards per Age, N+2 guilds, distinct boards, three coins.
// Rulings 16/26: side choice is simultaneous and precedes the first hand.
func TestSetupRules(t *testing.T) {
	for n := uint8(MinSeats); n <= MaxSeats; n++ {
		g, err := NewGame(Setup{Seed: 42, Seats: n})
		if err != nil {
			t.Fatal(err)
		}
		if err := g.CheckInvariants(); err != nil {
			t.Fatalf("n%d: %v", n, err)
		}
		if g.State.Phase != PhaseSides || g.State.Age != 0 || g.State.Turn != 0 || g.State.Decider != NoSeat || g.Point() != (Point{Kind: PointSides, Required: AllSeats(n)}) {
			t.Fatalf("side point %+v", g.State)
		}
		var boards [catalog.NumBoards + 1]bool
		for s := Seat(0); s < Seat(n); s++ {
			c := g.State.Cities[s]
			if c.Coins != 3 || c.Side != catalog.NoSide || !c.Board.Valid() || boards[c.Board] {
				t.Fatalf("city %+v", c)
			}
			boards[c.Board] = true
			if g.State.Hands[s] != ([HandSize]catalog.DesignID{}) || g.Record.Packets[0][s] != ([HandSize]catalog.DesignID{}) {
				t.Fatal("hand exposed before sides")
			}
			actions, ok := sideActions(&g.State, s)
			if !ok || actions[0].Key() != "side:day" || actions[1].Key() != "side:night" {
				t.Fatalf("side offer %v", actions)
			}
		}
		var counts [catalog.NumDesigns + 1]uint8
		for age, row := range g.State.Deals {
			for seat, hand := range row {
				for k, id := range hand {
					if seat >= int(n) {
						if id != 0 {
							t.Fatal("unused seat dealt")
						}
						continue
					}
					d, ok := catalog.DesignByID(id)
					if !ok || d.Age != uint8(age+1) || (k > 0 && hand[k-1] > id) {
						t.Fatalf("bad deal age%d seat%d %v", age, seat, hand)
					}
					counts[id]++
				}
			}
		}
		var ageCounts [NumAges]int
		for age, row := range g.State.Deals {
			for _, hand := range row {
				for _, id := range hand {
					if id != 0 {
						ageCounts[age]++
					}
				}
			}
		}
		for age, total := range ageCounts {
			if total != 7*int(n) {
				t.Fatalf("n%d age%d total%d want%d", n, age+1, total, 7*int(n))
			}
		}
		guilds := 0
		for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
			d, _ := catalog.DesignByID(id)
			if d.Color == catalog.Purple {
				guilds += int(counts[id])
			}
		}
		if ageCounts[2]-guilds != 6*int(n)-2 {
			t.Fatal("non-guild Age III total", ageCounts[2]-guilds)
		}
		if guilds != int(n)+2 {
			t.Fatal("guild draw", guilds)
		}
		box := 0
		for _, id := range g.State.BoxGuilds {
			if id != 0 {
				box++
				counts[id]++
			}
		}
		if box != 8-int(n) {
			t.Fatal("box size", box)
		}
		for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
			d, _ := catalog.DesignByID(id)
			if d.Color == catalog.Purple && counts[id] != 1 {
				t.Fatal("guild lost or duplicated", id)
			}
		}
	}
}

func TestSetupOptionsDoNotMoveDraws(t *testing.T) {
	for n := uint8(MinSeats); n <= MaxSeats; n++ {
		for _, seed := range []uint64{0, 1, 42, 1<<64 - 1} {
			base, err := NewGame(Setup{Seed: seed, Seats: n})
			if err != nil {
				t.Fatal(err)
			}
			for side := SideRule(0); side < NumSideRules; side++ {
				for seating := SeatingRule(0); seating < NumSeatingRules; seating++ {
					for _, hide := range []bool{false, true} {
						g, err := NewGame(Setup{Seed: seed, Seats: n, Options: Options{Sides: side, Seating: seating, HideAtEnd: hide}})
						if err != nil {
							t.Fatal(err)
						}
						if setupPacks(g) != setupPacks(base) || g.State.BoxGuilds != base.State.BoxGuilds {
							t.Fatal("option moved deck/guild draw")
						}
						for s := Seat(0); s < Seat(n); s++ {
							if g.State.Cities[s].Board != base.State.Cities[s].Board {
								t.Fatal("option moved boards")
							}
							if side == SidesDay && g.State.Cities[s].Side != catalog.Day {
								t.Fatal("day option")
							}
							if side == SidesRandom {
								random, err := NewGame(Setup{Seed: seed, Seats: n, Options: Options{Sides: SidesRandom}})
								if err != nil || random.State.Cities[s].Side != g.State.Cities[s].Side {
									t.Fatal("option moved random side draw", err)
								}
							}
						}
						if side != SidesChoose && (g.State.Phase != PhaseTurn || g.State.Age != 1 || g.State.Turn != 1 || g.State.Deals[0] != ([MaxSeats][HandSize]catalog.DesignID{}) || g.State.Hands != g.Record.Packets[0]) {
							t.Fatal("Age I not dealt exactly once")
						}
						if err := g.CheckInvariants(); err != nil {
							t.Fatal(err)
						}
					}
				}
			}
		}
	}
}

func TestSeatOrder(t *testing.T) {
	for n := uint8(MinSeats); n <= MaxSeats; n++ {
		for seed := uint64(0); seed < 100; seed++ {
			got := SeatOrder(seed, n, SeatingRandom)
			chosen := SeatOrder(seed, n, SeatingChosen)
			var seen [MaxSeats]bool
			for s := uint8(0); s < MaxSeats; s++ {
				if s < n {
					if got[s] >= n || seen[got[s]] || chosen[s] != s {
						t.Fatal("not permutation", got)
					}
					seen[got[s]] = true
				} else if got[s] != uint8(NoSeat) || chosen[s] != uint8(NoSeat) {
					t.Fatal("unused seating slots")
				}
			}
			if got != SeatOrder(seed, n, SeatingRandom) {
				t.Fatal("nondeterministic seating")
			}
		}
	}
	for _, n := range []uint8{0, 2, 8, 255} {
		got := SeatOrder(1, n, SeatingRandom)
		for _, s := range got {
			if s != uint8(NoSeat) {
				t.Fatal("invalid seating")
			}
		}
	}
}

func TestSidePointAtomic(t *testing.T) {
	g, err := NewGame(Setup{Seed: 1, Seats: 3})
	if err != nil {
		t.Fatal(err)
	}
	picks := [MaxSeats]Action{{Kind: ActSide, Side: catalog.Day}, {Kind: ActSide, Side: catalog.Night}, {Kind: ActSide, Side: catalog.Day}}
	for _, edit := range []func(*[MaxSeats]Action){func(p *[MaxSeats]Action) { p[1] = Action{} }, func(p *[MaxSeats]Action) { p[1].Side = 3 }, func(p *[MaxSeats]Action) { p[0].Left = 1 }, func(p *[MaxSeats]Action) { p[3] = p[0] }} {
		bad := picks
		edit(&bad)
		before := g
		after, _, err := resolveSides(g, bad)
		if err == nil || g != before || after != before {
			t.Fatal("partial side reveal on invalid batch")
		}
	}
	before := g
	next, ev, err := resolveSides(g, picks)
	if err != nil {
		t.Fatal(err)
	}
	if g != before || next.State.Points != 1 || next.Point() != (Point{Kind: PointTurn, Age: 1, Turn: 1, Required: AllSeats(3)}) || len(ev) != 4 {
		t.Fatal("side resolution")
	}
	for s := 0; s < 3; s++ {
		if next.State.Cities[s].Side != picks[s].Side || ev[s] != (Event{Kind: EvSideChosen, Seat: Seat(s), Side: picks[s].Side}) {
			t.Fatal("reveal order")
		}
	}
	if ev[3] != (Event{Kind: EvAgeStart, Age: 1}) {
		t.Fatal("Age start event")
	}
	if err := next.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := resolveSides(next, picks); err == nil {
		t.Fatal("resolved sides twice")
	}
	if _, ok := sideActions(&next.State, 0); ok {
		t.Fatal("side offer after dealing")
	}
}

func TestSetupRejectsInvalid(t *testing.T) {
	for _, setup := range []Setup{{}, {Seats: 2}, {Seats: 8}, {Seats: 3, Options: Options{Sides: 3}}, {Seats: 3, Options: Options{Seating: 2}}} {
		if g, err := NewGame(setup); err == nil || g != (Game{}) {
			t.Fatalf("accepted %+v", setup)
		}
	}
}

func TestSetupTenThousandPerN(t *testing.T) {
	for n := uint8(MinSeats); n <= MaxSeats; n++ {
		for seed := uint64(0); seed < 10000; seed++ {
			opts := Options{Sides: SideRule(seed % 3), Seating: SeatingRule(seed / 3 % 2), HideAtEnd: seed/6%2 != 0}
			g, err := NewGame(Setup{Seed: seed, Seats: n, Options: opts})
			if err != nil {
				t.Fatal(err)
			}
			if err := g.CheckInvariants(); err != nil {
				t.Fatalf("n%d seed%d opts%+v: %v", n, seed, opts, err)
			}
		}
		t.Logf("n%d:10000 seeded setups validated", n)
	}
}

type setupGolden struct {
	Seed        string                                        `json:"seed"`
	Seats       uint8                                         `json:"seats"`
	Tags        [7]string                                     `json:"stream_tags"`
	Order       [MaxSeats]uint8                               `json:"seat_order"`
	Boards      [MaxSeats]catalog.BoardID                     `json:"boards"`
	Sides       [MaxSeats]catalog.Side                        `json:"sides"`
	Packets     [NumAges][MaxSeats][HandSize]catalog.DesignID `json:"packets"`
	Box         [MaxBoxGuilds]catalog.DesignID                `json:"box"`
	Fingerprint string                                        `json:"fingerprint"`
}

func TestSetupGoldens(t *testing.T) {
	for n := uint8(MinSeats); n <= MaxSeats; n++ {
		const seed = uint64(0x4861667467616e65)
		g, err := NewGame(Setup{Seed: seed, Seats: n, Options: Options{Sides: SidesRandom}})
		if err != nil {
			t.Fatal(err)
		}
		snap := setupGolden{Seed: fmt.Sprintf("%016x", seed), Seats: n, Order: SeatOrder(seed, n, SeatingRandom), Packets: setupPacks(g), Box: g.State.BoxGuilds, Fingerprint: g.Fingerprint()}
		for i, tag := range [7]uint64{streamSeating, streamBoards, streamSides, streamGuilds, streamAge1, streamAge2, streamAge3} {
			snap.Tags[i] = fmt.Sprintf("%016x", tag)
		}
		for s := 0; s < MaxSeats; s++ {
			snap.Boards[s] = g.State.Cities[s].Board
			snap.Sides[s] = g.State.Cities[s].Side
		}
		data, err := json.MarshalIndent(snap, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		data = append(data, '\n')
		path := filepath.Join("testdata", "setup", fmt.Sprintf("n%d.json", n))
		if *updateSetupGoldens {
			if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, data, 0644); err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(data, want) {
			t.Fatalf("setup golden changed:%s; do not update without reviewing rules/tags/draw order", path)
		}
	}
}
