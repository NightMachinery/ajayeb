package rules

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

func salesBatch(g Game) Batch {
	b := Batch{Point: g.Point().ID()}
	for s := Seat(0); s < Seat(g.State.Seats); s++ {
		if !g.Point().Required.Has(s) {
			continue
		}
		key := "side:day"
		if g.State.Phase != PhaseSides {
			key = (Action{Kind: ActSell, Design: g.State.Hands[s][0]}).Key()
		}
		b.Picks = append(b.Picks, Pick{Seat: s, Key: key})
	}
	return b
}

func TestResolvePassingAndSixthTurn(t *testing.T) {
	g, err := NewGame(Setup{Seats: 3, Seed: 42, Options: Options{Sides: SidesDay}})
	if err != nil {
		t.Fatal(err)
	}
	for age := uint8(1); age <= 3; age++ {
		for turn := uint8(1); turn <= 6; turn++ {
			before := g
			b := salesBatch(g)
			next, events, err := g.Resolve(b)
			if err != nil {
				t.Fatal(err)
			}
			if g != before {
				t.Fatal("Resolve mutated caller")
			}
			if turn < 6 {
				for s := Seat(0); s < 3; s++ {
					destination := s.Left(3)
					if age == 2 {
						destination = s.Right(3)
					}
					want := [HandSize]catalog.DesignID{}
					copy(want[:], g.State.Hands[s][1:])
					if next.State.Hands[destination] != want {
						t.Fatalf("age%d turn%d wrong passing direction", age, turn)
					}
				}
				if next.Point().ID() != fmt.Sprintf("age-%d-turn-%d", age, turn+1) {
					t.Fatal("wrong next point", next.Point())
				}
			} else {
				var leftovers, conflicts int
				for _, e := range events {
					if e.Kind == EvLeftoverDiscarded {
						leftovers++
						if !e.Private {
							t.Fatal("leftover identity public")
						}
					}
					if e.Kind == EvMilitary {
						conflicts++
					}
				}
				if leftovers != 3 || conflicts != 3 {
					t.Fatal("sixth-turn leftovers/conflicts", leftovers, conflicts)
				}
				if age < 3 && (next.State.Age != age+1 || next.State.Hands != next.Record.Packets[age] || next.State.Deals[age] != ([MaxSeats][HandSize]catalog.DesignID{})) {
					t.Fatal("next Age not transferred once")
				}
			}
			for s := Seat(0); s < 3; s++ {
				if next.State.Cities[s].Coins != before.State.Cities[s].Coins+3 {
					t.Fatal("sale not three coins")
				}
				id, _ := ParseKey(b.Picks[s].Key)
				if next.Record.Picks[age-1][turn-1][s] != id.Design || next.Record.Plays[age-1][turn-1][s] != (Play{Kind: ActSell}) {
					t.Fatal("private/public record split")
				}
			}
			if err := next.CheckInvariants(); err != nil {
				t.Fatalf("age%d turn%d: %v", age, turn, err)
			}
			g = next
		}
	}
	if g.State.Phase != PhaseOver || g.Point().Kind != PointNone {
		t.Fatal("game did not terminate")
	}
}

func TestResolveRejectsInvalidBatchAtomically(t *testing.T) {
	g, err := NewGame(Setup{Seats: 3, Seed: 9, Options: Options{Sides: SidesDay}})
	if err != nil {
		t.Fatal(err)
	}
	for _, edit := range []func(*Batch){
		func(b *Batch) { b.Point = "age-1-turn-2" },
		func(b *Batch) { b.Picks = b.Picks[:2] },
		func(b *Batch) { b.Picks[1].Seat = 0 },
		func(b *Batch) { b.Picks[0], b.Picks[1] = b.Picks[1], b.Picks[0] },
		func(b *Batch) { b.Picks[0].Key = "sell:palace" },
		func(b *Batch) { b.Picks[0].Key = "build:palace:0:0" },
		func(b *Batch) { b.Picks[0].Key = "sell:garbage" },
	} {
		b := salesBatch(g)
		edit(&b)
		before := g
		next, events, err := g.Resolve(b)
		if err == nil || next != before || g != before || len(events) != 0 {
			t.Fatal("bad batch changed game", err)
		}
	}
}

func TestResolveMilitaryRing(t *testing.T) {
	for _, n := range []uint8{3, 4, 5, 6, 7} {
		for age := uint8(1); age <= 3; age++ {
			st := State{Seats: n, Phase: PhaseTurn, Age: age, Turn: 6, Decider: NoSeat}
			var picks [MaxSeats]Action
			for s := Seat(0); s < Seat(n); s++ {
				c, err := cityCaches(City{Board: catalog.BoardID(s + 1), Side: catalog.Day})
				if err != nil {
					t.Fatal(err)
				}
				// Different red buildings produce 0,1,2,3 shields without forged caches.
				switch s % 4 {
				case 1:
					c.Buildings.Add(14)
				case 2:
					c.Buildings.Add(35)
				case 3:
					c.Buildings.Add(51)
				}
				c, err = cityCaches(c)
				if err != nil {
					t.Fatal(err)
				}
				st.Cities[s] = c
				st.Hands[s] = [HandSize]catalog.DesignID{17, 18}
				picks[s] = Action{Kind: ActSell, Design: 17}
				if age < 3 {
					st.Deals[age][s] = [HandSize]catalog.DesignID{28, 29, 30, 31, 32, 33, 34}
					if age == 2 {
						st.Deals[age][s] = [HandSize]catalog.DesignID{51, 52, 53, 54, 55, 56, 57}
					}
				}
			}
			next, events, err := ResolveState(&st, &picks, nil)
			if err != nil {
				t.Fatal(err)
			}
			var pairs int
			for _, e := range events {
				if e.Kind == EvMilitary {
					if e.Seat != Seat(pairs) || e.Other != e.Seat.Left(n) || e.Age != age {
						t.Fatal("military not one canonical ring pass", e)
					}
					pairs++
				}
			}
			if pairs != int(n) {
				t.Fatal("wrong conflict count", pairs)
			}
			for s := Seat(0); s < Seat(n); s++ {
				var wins, losses uint8
				for _, other := range []Seat{s.Left(n), s.Right(n)} {
					if st.Cities[s].Shields > st.Cities[other].Shields {
						wins++
					}
					if st.Cities[s].Shields < st.Cities[other].Shields {
						losses++
					}
				}
				if next.Cities[s].Victories[age-1] != wins || next.Cities[s].Defeats != losses {
					t.Fatal("military tokens", n, age, s)
				}
			}
		}
	}
}

func TestResolveSweepsReplayCoreAndOrder(t *testing.T) {
	// This intermediate engine deliberately omits coin/scoring/wonder effects.
	// Sweep assertions are termination, conservation and invariants, not rules completeness.
	for n := uint8(3); n <= 7; n++ {
		for seed := uint64(0); seed < 200; seed++ {
			setup := Setup{Seats: n, Seed: seed, Options: Options{Sides: SideRule(seed % 3)}}
			g, err := NewGame(setup)
			if err != nil {
				t.Fatal(err)
			}
			var batches []Batch
			for steps := 0; g.Point().Kind != PointNone; steps++ {
				if steps >= MaxPoints {
					t.Fatal("did not terminate")
				}
				b := Batch{Point: g.Point().ID()}
				var picks [MaxSeats]Action
				for s := Seat(0); s < Seat(n); s++ {
					cs := g.Legal(s)
					if len(cs) == 0 {
						t.Fatal("empty required offer")
					}
					c := cs[(int(seed)+steps*3+int(s)*7)%len(cs)]
					b.Picks = append(b.Picks, Pick{Seat: s, Key: c.Key()})
					picks[s] = c.Action
				}
				frozen := g
				core, events, err := ResolveState(&g.State, &picks, nil)
				if err != nil {
					t.Fatal(err)
				}
				reversed, revEvents, err := resolveStateOrder(&g.State, &picks, nil, true)
				if err != nil || reversed != core || !reflect.DeepEqual(events, revEvents) {
					t.Fatal("seat loop changed state/events", err)
				}
				next, wrapperEvents, err := g.Resolve(b)
				if err != nil || next.State != core || !reflect.DeepEqual(wrapperEvents, events) || frozen != g {
					t.Fatal("Game/core mismatch", err)
				}
				if err := next.CheckInvariants(); err != nil {
					t.Fatalf("n%d seed%d step%d: %v", n, seed, steps, err)
				}
				batches = append(batches, b)
				g = next
			}
			replay, err := Replay(setup, batches)
			if err != nil || replay != g || replay.Fingerprint() != g.Fingerprint() {
				t.Fatal("replay differs", err)
			}
		}
	}
}

func TestResolveFrozenProductionAndPurse(t *testing.T) {
	// Player 0 sells a card while player 2 builds a free stockade, buying an ore
	// from 0. The receipt cannot pay for an otherwise unaffordable build by 0.
	st := State{Seats: 3, Phase: PhaseTurn, Age: 1, Turn: 1, Decider: NoSeat}
	for seat, board := range []catalog.BoardID{catalog.BoardRhodos, catalog.BoardGizah, catalog.BoardEphesos} {
		c, err := cityCaches(City{Board: board, Side: catalog.Day, Coins: 0})
		if err != nil {
			t.Fatal(err)
		}
		st.Cities[seat] = c
	}
	st.Cities[1].Coins = 2
	st.Hands[0] = [HandSize]catalog.DesignID{1, 17}
	st.Hands[1] = [HandSize]catalog.DesignID{14, 18}
	st.Hands[2] = [HandSize]catalog.DesignID{7, 19}
	cs := LegalState(&st, 0, nil)
	if len(buildChoices(cs, ActBuild, 1)) != 0 {
		t.Fatal("spent same-turn receipts before receiving them")
	}
	// Seat 1 may buy 0's ore. Seat 2's newly constructed Ore Vein must not be
	// offered as another source while prices/legality are read from F.
	choices := buildChoices(LegalState(&st, 1, nil), ActBuild, 14)
	if len(choices) != 1 || choices[0].Right != 2 || choices[0].Left != 0 {
		t.Fatal("same-turn production used", choices)
	}
	picks := [MaxSeats]Action{{Kind: ActSell, Design: 1}, choices[0].Action, {Kind: ActBuild, Design: 7}}
	next, events, err := ResolveState(&st, &picks, nil)
	if err != nil {
		t.Fatal(err)
	}
	if next.Cities[0].Coins != 5 || next.Cities[1].Coins != 0 || next.Cities[2].Coins != 0 {
		t.Fatal("trade payment/sale not additive", next.Cities)
	}
	if next.Cities[2].Produce[catalog.Ore] != 1 || !next.Cities[1].Buildings.Has(14) {
		t.Fatal("placements/cache")
	}
	if events[0].Kind != EvPlay || events[0].Seat != 0 || events[0].Design != 1 || !events[0].Private {
		t.Fatal("play order/private sale", events)
	}
	if events[3] != (Event{Kind: EvCoins, Seat: 0, Source: CoinSale, Delta: 3}) {
		t.Fatal("sale income event", events)
	}
	// Trusted core still rejects an overdraw and preserves an existing prefix.
	bad := picks
	bad[0] = Action{Kind: ActBuild, Design: 1}
	prefix := []Event{{Kind: EvAgeStart, Age: 1}}
	original := st
	got, evs, err := ResolveState(&st, &bad, prefix)
	if err == nil || got != original || st != original || !reflect.DeepEqual(evs, prefix) {
		t.Fatal("failed core changed values/prefix", err)
	}
}

func TestResolveStageTucksChosenCardAndPaysStageCost(t *testing.T) {
	st := State{Seats: 3, Phase: PhaseTurn, Age: 1, Turn: 2, Decider: NoSeat}
	for seat, board := range []catalog.BoardID{catalog.BoardGizah, catalog.BoardRhodos, catalog.BoardEphesos} {
		c, err := cityCaches(City{Board: board, Side: catalog.Day, Coins: 3})
		if err != nil {
			t.Fatal(err)
		}
		st.Cities[seat] = c
		st.Hands[seat] = [HandSize]catalog.DesignID{1, 17}
	}
	// Gizah's first stage costs two wood, not the tucked Clay Pit's one coin.
	st.Cities[1].ForSale[catalog.Wood] = 2
	offered := buildChoices(LegalState(&st, 0, nil), ActStage, 1)
	if len(offered) != 0 {
		t.Fatal("tucked card's cheap cost used instead of stage cost")
	}
	st.Cities[0].Coins = 4
	offered = buildChoices(LegalState(&st, 0, nil), ActStage, 1)
	if len(offered) != 1 || offered[0].Stage != 1 || offered[0].Printed != 0 || offered[0].Left != 4 {
		t.Fatal("stage quote", offered)
	}
	picks := [MaxSeats]Action{offered[0].Action, {Kind: ActSell, Design: 1}, {Kind: ActSell, Design: 1}}
	next, events, err := ResolveState(&st, &picks, nil)
	if err != nil || next.Cities[0].Stages != 1 || next.Cities[0].Tucked[0] != 1 || !next.Cities[0].Buildings.Empty() || next.Cities[0].Coins != 0 || next.Cities[1].Coins != 10 {
		t.Fatal("stage placement/payment", err, next.Cities)
	}
	if events[0].Stage != 1 || !events[0].Private || events[0].Printed != 0 {
		t.Fatal("stage event", events)
	}
}

func TestResolveSeedAndRecordDoNotEnterCore(t *testing.T) {
	g, err := NewGame(Setup{Seats: 3, Seed: 8, Options: Options{Sides: SidesDay}})
	if err != nil {
		t.Fatal(err)
	}
	before := g
	var picks [MaxSeats]Action
	for s := Seat(0); s < 3; s++ {
		picks[s] = g.Legal(s)[0].Action
	}
	a, events, err := ResolveState(&g.State, &picks, nil)
	if err != nil {
		t.Fatal(err)
	}
	g.State.Seed = ^g.State.Seed
	g.Record.Picks[0][0][0] = 255
	for s := Seat(0); s < 3; s++ {
		if !reflect.DeepEqual(g.Legal(s), before.Legal(s)) {
			t.Fatal("hidden seed/record changed Legal")
		}
	}
	b, otherEvents, err := ResolveState(&g.State, &picks, nil)
	if err != nil {
		t.Fatal(err)
	}
	b.Seed = a.Seed
	if a != b || !reflect.DeepEqual(events, otherEvents) {
		t.Fatal("core depended on seed/record")
	}
}

func TestResolveAppendAndCoreAllocations(t *testing.T) {
	g, err := NewGame(Setup{Seats: 7, Seed: 42, Options: Options{Sides: SidesDay}})
	if err != nil {
		t.Fatal(err)
	}
	var picks [MaxSeats]Action
	for s := Seat(0); s < 7; s++ {
		picks[s] = g.Legal(s)[0].Action
	}
	var buf [32]Event
	buf[0] = Event{Kind: EvAgeStart, Age: 1}
	_, without, err := ResolveState(&g.State, &picks, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, with, err := ResolveState(&g.State, &picks, buf[:1])
	if err != nil || with[0] != buf[0] || !reflect.DeepEqual(with[1:], without) {
		t.Fatal("event append contract", err)
	}
	if allocs := testing.AllocsPerRun(100, func() {
		_, _, err := ResolveState(&g.State, &picks, buf[:0])
		if err != nil {
			panic(err)
		}
	}); allocs != 0 {
		t.Fatalf("state-only Resolve allocates: %v", allocs)
	}
}
