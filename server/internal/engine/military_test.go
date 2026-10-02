package engine_test

import (
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/enginetest"
)

// Tests of the military track through the public API: shields moving the
// pawn, Strategy, the looting tokens, military supremacy, and the weaker
// player choosing the next age's starter after real shields. The pawn's
// points at the end are in score_test.go.

// militaryEvents returns the events of the given kinds, in order.
func militaryEvents(ev []engine.Event, ks ...engine.EventKind) []engine.Event {
	out := []engine.Event{}
	for _, e := range ev {
		if slices.Contains(ks, e.Kind) {
			out = append(out, e)
		}
	}
	return out
}

func TestMilitaryPawnAndLooting(t *testing.T) {
	loot := func(seat engine.Seat, lost, value int16) engine.Event {
		return engine.Event{Kind: engine.EvLoot, Seat: seat, A: lost, B: value}
	}
	cases := []struct {
		name     string
		age      int
		pawn     int
		seat     engine.Seat
		card     string
		strategy bool
		coins    [2]int
		// expected
		to      int8
		after   [2]int16
		loot    []engine.Event
		looting uint8
	}{
		{name: "one shield for seat 0", age: 1, pawn: 0, seat: 0, card: "guard-tower", coins: [2]int{7, 7},
			to: 1, after: [2]int16{7, 7}, looting: engine.LootAll},
		{name: "one shield for seat 1", age: 1, pawn: 0, seat: 1, card: "guard-tower", coins: [2]int{7, 7},
			to: -1, after: [2]int16{7, 7}, looting: engine.LootAll},
		{name: "entering space 3 takes 2 coins", age: 1, pawn: 2, seat: 0, card: "guard-tower", coins: [2]int{7, 7},
			to: 3, after: [2]int16{7, 5}, loot: []engine.Event{loot(1, 2, 2)}, looting: engine.LootAll &^ engine.LootPlus3},
		{name: "the loss is floored at the coins held", age: 1, pawn: 2, seat: 0, card: "guard-tower", coins: [2]int{7, 1},
			to: 3, after: [2]int16{7, 0}, loot: []engine.Event{loot(1, 1, 2)}, looting: engine.LootAll &^ engine.LootPlus3},
		{name: "a player with no coins still loses the token", age: 1, pawn: 2, seat: 0, card: "guard-tower", coins: [2]int{7, 0},
			to: 3, after: [2]int16{7, 0}, loot: []engine.Event{loot(1, 0, 2)}, looting: engine.LootAll &^ engine.LootPlus3},
		{name: "seat 1 takes seat 0's token", age: 1, pawn: -2, seat: 1, card: "guard-tower", coins: [2]int{7, 7},
			to: -3, after: [2]int16{5, 7}, loot: []engine.Event{loot(0, 2, 2)}, looting: engine.LootAll &^ engine.LootMinus3},
		{name: "both tokens in one move, with Strategy", age: 3, pawn: 2, seat: 0, card: "pretorium", strategy: true, coins: [2]int{10, 4},
			to: 6, after: [2]int16{2, 0}, loot: []engine.Event{loot(1, 2, 2), loot(1, 2, 5)}, looting: engine.LootMinus3 | engine.LootMinus6},
		{name: "both tokens of seat 0's side", age: 3, pawn: -2, seat: 1, card: "pretorium", strategy: true, coins: [2]int{9, 10},
			to: -6, after: [2]int16{2, 2}, loot: []engine.Event{loot(0, 2, 2), loot(0, 5, 5)}, looting: engine.LootPlus3 | engine.LootPlus6},
		{name: "only the far token is left", age: 3, pawn: 4, seat: 0, card: "pretorium", coins: [2]int{10, 7},
			to: 7, after: [2]int16{2, 2}, loot: []engine.Event{loot(1, 5, 5)}, looting: engine.LootMinus3 | engine.LootMinus6},
		{name: "back toward the centre takes nothing", age: 1, pawn: 4, seat: 1, card: "palisade", coins: [2]int{7, 7},
			to: 3, after: [2]int16{7, 5}, looting: engine.LootAll &^ engine.LootPlus3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := enginetest.New(t, 21).Age(c.age).Pawn(c.pawn).ToMove(c.seat).Accessible(c.card).
				Coins(0, c.coins[0]).Coins(1, c.coins[1])
			if c.strategy {
				b.Tokens(c.seat, "strategy")
			}
			s := b.State()
			ev := enginetest.Do(t, &s, "construct "+c.card)
			if s.Military != c.to || s.Looting != c.looting {
				t.Fatalf("pawn %d, looting %#x; want %d, %#x", s.Military, s.Looting, c.to, c.looting)
			}
			if got := [2]int16{s.Players[0].Coins, s.Players[1].Coins}; got != c.after {
				t.Fatalf("coins %v, want %v", got, c.after)
			}
			want := append([]engine.Event{{Kind: engine.EvPawn, Seat: c.seat, A: int16(c.pawn), B: int16(c.to)}}, c.loot...)
			if got := militaryEvents(ev, engine.EvPawn, engine.EvLoot); !slices.Equal(got, want) {
				t.Fatalf("military events %+v, want %+v", got, want)
			}
			if s.Over() || s.ToMove != c.seat.Opp() {
				t.Fatalf("phase %s, seat %d to move", s.Phase, s.ToMove)
			}
		})
	}
}

// TestMilitaryTokenTriggersOnce: once a looting token is taken, the pawn can
// leave its zone and enter it again without taking anything.
func TestMilitaryTokenTriggersOnce(t *testing.T) {
	s := enginetest.New(t, 26).Pawn(3).ToMove(1).Accessible("guard-tower", "palisade").State()
	ev := enginetest.Do(t, &s, "construct guard-tower", "construct palisade")
	if s.Military != 3 || s.Players[1].Coins != engine.StartCoins || len(militaryEvents(ev, engine.EvLoot)) != 0 {
		t.Fatalf("pawn %d, seat 1 coins %d, events %+v", s.Military, s.Players[1].Coins, ev)
	}
	want := []engine.Event{
		{Kind: engine.EvPawn, Seat: 1, A: 3, B: 2},
		{Kind: engine.EvPawn, Seat: 0, A: 2, B: 3},
	}
	if got := militaryEvents(ev, engine.EvPawn); !slices.Equal(got, want) {
		t.Fatalf("pawn events %+v, want %+v", got, want)
	}
}

// TestMilitaryEconomyBeforeLooting: the trading coins reach the owner of
// Economy before the looting token takes coins from them.
func TestMilitaryEconomyBeforeLooting(t *testing.T) {
	s := enginetest.New(t, 22).FirstPlayer(0).Pawn(2).Accessible("garrison").
		Tokens(1, "economy").Coins(1, 0).State()
	garrison := cardID(t, "garrison")
	ev := enginetest.Do(t, &s, "construct garrison")
	want := []engine.Event{
		{Kind: engine.EvBuild, Seat: 0, Card: garrison},
		{Kind: engine.EvPay, Seat: 0, A: 2, B: 2},
		{Kind: engine.EvPawn, Seat: 0, A: 2, B: 3},
		{Kind: engine.EvLoot, Seat: 1, A: 2, B: 2},
	}
	if !slices.Equal(ev[:len(want)], want) {
		t.Fatalf("events %+v, want %+v first", ev, want)
	}
	if s.Players[0].Coins != 5 || s.Players[1].Coins != 0 {
		t.Fatalf("coins %d and %d", s.Players[0].Coins, s.Players[1].Coins)
	}
}

func TestMilitarySupremacy(t *testing.T) {
	cases := []struct {
		name     string
		pawn     int
		seat     engine.Seat
		card     string
		strategy bool
		to       int16
	}{
		{"seat 0 reaches seat 1's capital", 6, 0, "pretorium", false, 9},
		{"the move stops at the capital", 8, 0, "pretorium", false, 9},
		{"seat 1 reaches seat 0's capital", -7, 1, "pretorium", false, -9},
		{"Strategy's shield completes the move", 6, 0, "circus", true, 9},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := enginetest.New(t, 23).Age(3).Pawn(c.pawn).ToMove(c.seat).Accessible(c.card).Coins(c.seat, 20)
			if c.strategy {
				b.Tokens(c.seat, "strategy")
			}
			s := b.State()
			ev := enginetest.Do(t, &s, "construct "+c.card)
			if !s.Over() || s.Result.Kind != engine.VictoryMilitary || s.Result.Winner != int8(c.seat) {
				t.Fatalf("phase %s, result %+v", s.Phase, s.Result)
			}
			if int16(s.Military) != c.to || s.Result.Scores != engine.Scores(&s) {
				t.Fatalf("pawn %d, scores %+v", s.Military, s.Result.Scores)
			}
			last := ev[len(ev)-1]
			if last != (engine.Event{Kind: engine.EvGameOver, A: int16(engine.VictoryMilitary), B: int16(c.seat)}) {
				t.Fatalf("last event %+v", last)
			}
			if pawn := militaryEvents(ev, engine.EvPawn); len(pawn) != 1 || pawn[0].B != c.to {
				t.Fatalf("pawn events %+v", pawn)
			}
			if len(militaryEvents(ev, engine.EvReveal)) != 0 {
				t.Fatalf("the turn went on after the supremacy: %+v", ev)
			}
			if got := enginetest.Keys(&s); len(got) != 0 {
				t.Fatalf("legal after the game: %v", got)
			}
		})
	}
}

// TestMilitarySupremacyOnTheLastCard: a supremacy on an age's last card ends
// the game; no age begins and, in Age III, there is no civilian scoring.
func TestMilitarySupremacyOnTheLastCard(t *testing.T) {
	s := enginetest.New(t, 24).OnlyLeft("guard-tower").Pawn(8).ToMove(0).State()
	ev := enginetest.Do(t, &s, "construct guard-tower")
	if !s.Over() || s.Result.Kind != engine.VictoryMilitary || s.Age != 1 || len(militaryEvents(ev, engine.EvAgeStart)) != 0 {
		t.Fatalf("phase %s, age %d, result %+v, events %+v", s.Phase, s.Age, s.Result, ev)
	}
	s = enginetest.New(t, 24).Age(3).OnlyLeft("pretorium").Pawn(-7).ToMove(1).Coins(1, 8).State()
	enginetest.Do(t, &s, "construct pretorium")
	if s.Result.Kind != engine.VictoryMilitary || s.Result.Winner != 1 {
		t.Fatalf("the last card of Age III: result %+v", s.Result)
	}
}

// TestMilitaryWeakerPlayerChooses: shields on the last card of an age leave
// the pawn off centre, and the player on whose side it stands chooses who
// starts the next age.
func TestMilitaryWeakerPlayerChooses(t *testing.T) {
	for _, seat := range []engine.Seat{0, 1} {
		for _, literal := range []bool{false, true} {
			opts := engine.Options{}
			if literal {
				opts.CenterPawn = engine.LastActiveStarts
			}
			s := enginetest.New(t, 25).Options(opts).OnlyLeft("guard-tower").ToMove(seat).State()
			enginetest.Do(t, &s, "construct guard-tower")
			if s.Age != 2 || s.Phase != engine.PhaseChooseStarter || s.ToMove != seat.Opp() || s.Active != seat {
				t.Fatalf("seat %d (literal %v): age %d, phase %s, to move %d, active %d", seat, literal, s.Age, s.Phase, s.ToMove, s.Active)
			}
		}
	}
}
