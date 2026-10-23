package rules

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

type wonderCity struct {
	board     catalog.BoardID
	side      catalog.Side
	stages    uint8
	buildings []catalog.DesignID
}

// Explicit conserving worlds, with partial historical records as supported by
// Scenario. Reserve pile cards and requested hands before allocating fillers.
func abilityScenario(t *testing.T, age, turn uint8, specs [MaxSeats]wonderCity, wanted [MaxSeats][]catalog.DesignID, pile ...catalog.DesignID) Game {
	t.Helper()
	var demand [catalog.NumDesigns + 1]uint8
	for _, s := range specs {
		for _, id := range s.buildings {
			demand[id]++
		}
	}
	for _, h := range wanted {
		for _, id := range h {
			demand[id]++
		}
	}
	for _, id := range pile {
		demand[id]++
	}
	var g Game
	found := false
	for seed := uint64(0); seed < 100; seed++ {
		var err error
		g, err = NewGame(Setup{Seats: 7, Seed: seed, Options: Options{Sides: SidesRandom}})
		if err != nil {
			t.Fatal(err)
		}
		ok := true
		for _, id := range g.State.BoxGuilds {
			if id != 0 && demand[id] != 0 {
				ok = false
			}
		}
		if ok {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("requested scenario guilds not drawn")
	}
	for s, h := range g.State.Hands {
		for _, id := range h {
			g.State.Discard[id]++
		}
		g.State.Hands[s] = [HandSize]catalog.DesignID{}
	}
	for a := uint8(2); a <= age; a++ {
		g.Record.Packets[a-1] = g.State.Deals[a-1]
		for _, h := range g.State.Deals[a-1] {
			for _, id := range h {
				if id != 0 {
					g.State.Discard[id]++
				}
			}
		}
		g.State.Deals[a-1] = [MaxSeats][HandSize]catalog.DesignID{}
	}
	var used [catalog.NumBoards + 1]bool
	for _, s := range specs {
		if s.board != 0 {
			if used[s.board] {
				t.Fatal("duplicate forced board")
			}
			used[s.board] = true
		}
	}
	var reserve [catalog.NumDesigns + 1]uint8
	for _, h := range wanted {
		for _, id := range h {
			reserve[id]++
		}
	}
	for _, id := range pile {
		reserve[id]++
	}
	for seat, s := range specs {
		if s.board == 0 {
			for b := catalog.BoardID(1); b <= catalog.NumBoards; b++ {
				if !used[b] {
					s.board = b
					used[b] = true
					break
				}
			}
		}
		if s.side == catalog.NoSide {
			s.side = catalog.Day
		}
		c := City{Board: s.board, Side: s.side, Stages: s.stages, Coins: 30}
		for _, id := range s.buildings {
			if g.State.Discard[id] == 0 {
				t.Fatal("building supply", id)
			}
			g.State.Discard[id]--
			c.Buildings.Add(id)
		}
		g.State.Cities[seat] = c
	}
	for seat, c := range g.State.Cities {
		for stage := uint8(0); stage < c.Stages; stage++ {
			found := false
			for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
				if g.State.Discard[id] > reserve[id] {
					g.State.Discard[id]--
					g.State.Cities[seat].Tucked[stage] = id
					found = true
					break
				}
			}
			if !found {
				t.Fatal("tucked supply")
			}
		}
	}
	for seat, h := range wanted {
		for k, id := range h {
			if g.State.Discard[id] == 0 {
				t.Fatal("wanted hand supply", id)
			}
			g.State.Discard[id]--
			reserve[id]--
			g.State.Hands[seat][k] = id
		}
	}
	for seat := 0; seat < 7; seat++ {
		for k := len(wanted[seat]); k < 8-int(turn); k++ {
			found := false
			for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
				d, _ := catalog.DesignByID(id)
				if d.Age == age && g.State.Discard[id] > reserve[id] {
					g.State.Discard[id]--
					g.State.Hands[seat][k] = id
					found = true
					break
				}
			}
			if !found {
				t.Fatal("hand supply", age, turn, seat, k)
			}
		}
	}
	g.State.Phase, g.State.Age, g.State.Turn, g.State.Decider = PhaseTurn, age, turn, NoSeat
	g.State.Points = 0
	alignScenarioHistory(t, &g)
	g.State.Result = Result{}
	built, err := NewGameFromScenario(g.State.Seed, g.State.Opts, ScenarioFromGame(g))
	if err != nil {
		t.Fatal("ability scenario", err)
	}
	return built
}
func actionBatch(g Game, seat Seat, a Action) Batch {
	b := salesBatch(g)
	for i, p := range b.Picks {
		if p.Seat == seat {
			b.Picks[i].Key = a.Key()
		}
	}
	return b
}
func stagePick(t *testing.T, g Game, seat Seat, id catalog.DesignID) Action {
	t.Helper()
	for _, c := range g.Legal(seat) {
		if c.Kind == ActStage && c.Design == id {
			return c.Action
		}
	}
	t.Fatal("stage not offered", g.Point(), seat, id)
	return Action{}
}
func requiredBatch(g Game, seat Seat, a Action) Batch {
	return Batch{Point: g.Point().ID(), Picks: []Pick{{Seat: seat, Key: a.Key()}}}
}

func TestRuling1BabylonStageBuiltSixthTurnEveryAge(t *testing.T) {
	for age := uint8(1); age <= 3; age++ {
		cards := [3][2]catalog.DesignID{{17, 18}, {39, 40}, {56, 57}}
		g := abilityScenario(t, age, 6, [MaxSeats]wonderCity{0: {board: catalog.BoardBabylon, side: catalog.Night, buildings: []catalog.DesignID{8}}, 1: {board: catalog.BoardGizah}}, [MaxSeats][]catalog.DesignID{0: cards[age-1][:]})
		a := stagePick(t, g, 0, cards[age-1][0])
		next, events, err := resolveAbility(t, g, actionBatch(g, 0, a))
		if err != nil {
			t.Fatal(err)
		}
		if next.State.Phase != PhaseLastCard || next.Point().ID() != fmt.Sprintf("age-%d-last-card", age) || next.Point().Required != Seat(0).Mask() || next.State.Cities[0].Kept != cards[age-1][1] {
			t.Fatal("new stage did not keep the sixth leftover", next.Point())
		}
		kept := false
		for _, e := range events {
			if e.Kind == EvCardKept && e.Seat == 0 && e.Design == cards[age-1][1] && e.Private {
				kept = true
			}
			if e.Kind == EvMilitary || e.Kind == EvAgeStart || e.Kind == EvGameOver {
				t.Fatal("Age advanced before kept card")
			}
		}
		if !kept {
			t.Fatal("missing private kept event")
		}
		if err := next.CheckInvariants(); err != nil {
			t.Fatal(err)
		}
		done, _, err := resolveAbility(t, next, requiredBatch(next, 0, Action{Kind: ActSell, Design: cards[age-1][1]}))
		if err != nil {
			t.Fatal(err)
		}
		if done.State.Cities[0].Kept != 0 || done.Record.Picks[age-1][6][0] != cards[age-1][1] {
			t.Fatal("kept record/clear")
		}
		if err := done.CheckInvariants(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestRulings4And22BabylonThreeActionsNoDecline(t *testing.T) {
	g := abilityScenario(t, 3, 6, [MaxSeats]wonderCity{0: {board: catalog.BoardBabylon, side: catalog.Night, stages: 1, buildings: []catalog.DesignID{2, 28, 30, 11}}}, [MaxSeats][]catalog.DesignID{0: {51, 64}})
	pending, _, err := resolveAbility(t, g, actionBatch(g, 0, Action{Kind: ActSell, Design: 51}))
	if err != nil {
		t.Fatal(err)
	}
	if pending.State.Phase != PhaseLastCard {
		t.Fatal("missing last point")
	}
	for _, kind := range []ActionKind{ActBuild, ActStage, ActSell} {
		choices := buildChoices(pending.Legal(0), kind, 64)
		if len(choices) == 0 {
			t.Fatal("missing kept action", kind)
		}
		done, _, err := resolveAbility(t, pending, requiredBatch(pending, 0, choices[0].Action))
		if err != nil {
			t.Fatal(err)
		}
		if done.State.Phase != PhaseOver || done.State.Cities[0].Kept != 0 || done.Record.Plays[2][6][0].Kind != kind {
			t.Fatal("last-card resolution/record")
		}
		if err := done.CheckInvariants(); err != nil {
			t.Fatal(err)
		}
	}
	if len(pending.Legal(1)) != 0 {
		t.Fatal("non-decider offered a kept move")
	}
	for _, key := range []string{"", "pass", "side:day"} {
		if _, _, err := resolveAbility(t, pending, Batch{Point: pending.Point().ID(), Picks: []Pick{{Seat: 0, Key: key}}}); err == nil {
			t.Fatal("kept card declined", key)
		}
	}
}

func TestRulings2_3_24_25_27OwnLeftoverDiscardBuild(t *testing.T) {
	g := abilityScenario(t, 1, 6, [MaxSeats]wonderCity{0: {board: catalog.BoardHalikarnassos, side: catalog.Night, buildings: []catalog.DesignID{2}}, 1: {board: catalog.BoardOlympia, buildings: []catalog.DesignID{18}}}, [MaxSeats][]catalog.DesignID{0: {17, 18}})
	if g.State.Discard[18] != 0 {
		t.Fatal("own leftover fixture already in pile")
	}
	a := stagePick(t, g, 0, 17)
	pending, events, err := resolveAbility(t, g, actionBatch(g, 0, a))
	if err != nil {
		t.Fatal(err)
	}
	if pending.State.Phase != PhaseDiscardBuild || pending.Point().ID() != "age-1-turn-6-discard-build" || pending.State.Discard[18] != 1 {
		t.Fatal("own leftover not offered before Age end", pending.Point())
	}
	for _, e := range events {
		if e.Kind == EvMilitary {
			t.Fatal("military before discard build")
		}
		if e.Kind == EvPlay && e.Seat == 0 && (e.Left != a.Left || e.Right != a.Right || e.Stage != 1 || !e.Private) {
			t.Fatal("stage payment contract", e)
		}
	}
	if err := pending.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
	choices := pending.Legal(0)
	for _, c := range choices {
		if c.Kind != ActBuildDiscarded || c.Total != 0 || c.Printed != 0 || c.Left != 0 || c.Right != 0 {
			t.Fatal("discard point offered a non-free/decline action")
		}
	}
	if len(buildChoices(choices, ActBuildDiscarded, 18)) != 1 || len(pending.Legal(1)) != 0 {
		t.Fatal("discard required seats or legal own leftover")
	}
	done, _, err := resolveAbility(t, pending, requiredBatch(pending, 0, Action{Kind: ActBuildDiscarded, Design: 18}))
	if err != nil {
		t.Fatal(err)
	}
	if !done.State.Cities[0].Buildings.Has(18) || done.State.Discard[18] != 0 || done.Record.Rebuilt[0][5][0] != 18 || done.State.Age != 2 {
		t.Fatal("own leftover rebuild did not finish before deal")
	}
	if err := done.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
}

func TestRuling23PileLookEvenWhenSkipped(t *testing.T) {
	for rotation := 0; rotation < MaxSeats; rotation++ {
		for _, duplicates := range []bool{false, true} {
			old := []catalog.DesignID{2, 5, 7, 8, 11, 12, 13}
			var specs [MaxSeats]wonderCity
			specs[0] = wonderCity{board: catalog.BoardHalikarnassos, side: catalog.Night}
			var wanted [MaxSeats][]catalog.DesignID
			wanted[0] = []catalog.DesignID{17}
			if duplicates {
				specs[0].buildings = old
				for s := 1; s < 7; s++ {
					wanted[s] = []catalog.DesignID{old[s-1]}
				}
			} else {
				for s, id := range old {
					specs[s].buildings = []catalog.DesignID{id}
				}
				for s, id := range []catalog.DesignID{21, 22, 24, 23, 19, 20} {
					wanted[s+1] = []catalog.DesignID{id}
				}
			}
			specs[1].board = catalog.BoardOlympia
			var rotatedSpecs [MaxSeats]wonderCity
			var rotatedHands [MaxSeats][]catalog.DesignID
			for seat := range specs {
				rotatedSpecs[(seat+rotation)%MaxSeats] = specs[seat]
				rotatedHands[(seat+rotation)%MaxSeats] = wanted[seat]
			}
			owner := Seat(rotation)
			g := abilityScenario(t, 1, 2, rotatedSpecs, rotatedHands)
			for id, n := range g.State.Discard {
				if n != 0 {
					t.Fatal("skip fixture must begin with empty pile", id, n)
				}
			}
			b := actionBatch(g, owner, stagePick(t, g, owner, 17))
			for s := 1; s < 7; s++ {
				kind := ActBuild
				if duplicates {
					kind = ActSell
				}
				b.Picks[(s+rotation)%MaxSeats].Key = (Action{Kind: kind, Design: wanted[s][0]}).Key()
			}
			next, events, err := resolveAbility(t, g, b)
			if err != nil {
				t.Fatal(err)
			}
			if next.State.Phase != PhaseTurn || next.State.Turn != 3 || next.State.Decider != NoSeat {
				t.Fatal("empty/all-duplicate pile opened a point")
			}
			seen := 0
			for _, e := range events {
				if e.Kind == EvDiscardBuildNothing {
					if e.Seat != owner {
						t.Fatal("wrong pile look owner")
					}
					seen++
				}
			}
			if seen != 1 || next.Record.Plays[0][1][owner].Kind != ActStage || next.Record.Rebuilt[0][1][owner] != 0 {
				t.Fatal("skipped look not recorded by stage/outcome")
			}
			if duplicates {
				count := 0
				for _, n := range next.State.Discard {
					count += int(n)
				}
				if count != 6 {
					t.Fatal("nonempty all-duplicate pile lost cards")
				}
			}
			if err := next.CheckInvariants(); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Log("forced skip sweep: 7 empty piles, 7 nonempty all-duplicate piles, every decider seat")
}

func TestRulings5And29DiscardBeforeKeptCardUsesFreshProduction(t *testing.T) {
	g := abilityScenario(t, 1, 6, [MaxSeats]wonderCity{
		0: {board: catalog.BoardHalikarnassos, side: catalog.Night, buildings: []catalog.DesignID{2}},
		1: {board: catalog.BoardBabylon, side: catalog.Night, stages: 1}, 3: {buildings: []catalog.DesignID{7}}, 4: {buildings: []catalog.DesignID{14}}, 6: {board: catalog.BoardOlympia},
	}, [MaxSeats][]catalog.DesignID{0: {17, 7}, 1: {14, 23}})
	g.State.Cities[1].Coins = 0
	b := actionBatch(g, 0, stagePick(t, g, 0, 17))
	b.Picks[1].Key = (Action{Kind: ActSell, Design: 23}).Key()
	pending, _, err := resolveAbility(t, g, b)
	if err != nil {
		t.Fatal(err)
	}
	if pending.State.Phase != PhaseDiscardBuild || pending.State.Cities[1].Kept != 14 || pending.State.Cities[1].Coins != 3 {
		t.Fatal("discard was not first")
	}
	if len(buildChoices(pending.Legal(0), ActBuildDiscarded, 14)) != 0 {
		t.Fatal("Babylon kept card entered the pile")
	}
	last, _, err := resolveAbility(t, pending, requiredBatch(pending, 0, Action{Kind: ActBuildDiscarded, Design: 7}))
	if err != nil {
		t.Fatal(err)
	}
	if last.State.Phase != PhaseLastCard || last.State.Decider != 1 {
		t.Fatal("kept point did not follow discard")
	}
	offers := buildChoices(last.Legal(1), ActBuild, 14)
	if len(offers) != 1 || offers[0].Right != 2 || offers[0].Left != 0 {
		t.Fatal("kept card could not spend sale income on new neighbour production", offers)
	}
	done, _, err := resolveAbility(t, last, requiredBatch(last, 1, offers[0].Action))
	if err != nil {
		t.Fatal(err)
	}
	if done.State.Cities[1].Coins != 1 || done.State.Cities[1].Victories[0] != 2 || done.Record.Rebuilt[0][5][0] != 7 || done.Record.Picks[0][6][1] != 14 {
		t.Fatal("kept payment, military or record")
	}
	if done.State.Cities[0].Coins != g.State.Cities[0].Coins {
		t.Fatal("one-seat trade receipt was lost")
	}
	if err := last.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
	if err := done.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
	// Replay the exact forced start, including its explicit scenario.
	replay, err := Replay(Setup{Seed: g.State.Seed, Seats: 7, Options: g.State.Opts, Scenario: ScenarioFromGame(g)}, []Batch{b, requiredBatch(pending, 0, Action{Kind: ActBuildDiscarded, Design: 7}), requiredBatch(last, 1, offers[0].Action)})
	if err != nil || replay != done {
		t.Fatal("ability replay mismatch", err)
	}
}

func TestRuling28DiscardIncomeAfterEarlierEffects(t *testing.T) {
	g := abilityScenario(t, 2, 4, [MaxSeats]wonderCity{0: {board: catalog.BoardHalikarnassos, side: catalog.Night, buildings: []catalog.DesignID{2, 23}}, 1: {board: catalog.BoardOlympia}}, [MaxSeats][]catalog.DesignID{0: {39}, 1: {28}, 6: {29}}, 46)
	b := actionBatch(g, 0, stagePick(t, g, 0, 39))
	b.Picks[1].Key = (Action{Kind: ActBuild, Design: 28}).Key()
	b.Picks[6].Key = (Action{Kind: ActBuild, Design: 29}).Key()
	pending, _, err := resolveAbility(t, g, b)
	if err != nil {
		t.Fatal(err)
	}
	done, events, err := resolveAbility(t, pending, requiredBatch(pending, 0, Action{Kind: ActBuildDiscarded, Design: 46}))
	if err != nil {
		t.Fatal(err)
	}
	if done.State.Cities[0].Coins != 31 {
		t.Fatal("discard Vineyard omitted new neighbours or replayed Tavern", done.State.Cities[0].Coins)
	}
	if len(events) < 2 || events[0] != (Event{Kind: EvDiscardBuild, Seat: 0, Design: 46}) || events[1] != (Event{Kind: EvCoins, Seat: 0, Design: 46, Source: CoinDesign, Delta: 3}) {
		t.Fatal("discard effect event order", events)
	}
	if err := done.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
	// Bazar resolves in step 5, before the owner's newly rebuilt Glassworks.
	g = abilityScenario(t, 2, 4, [MaxSeats]wonderCity{0: {board: catalog.BoardHalikarnassos, side: catalog.Night, buildings: []catalog.DesignID{2}}, 1: {board: catalog.BoardOlympia}}, [MaxSeats][]catalog.DesignID{0: {39}, 1: {43}, 2: {32}}, 11)
	b = actionBatch(g, 0, stagePick(t, g, 0, 39))
	b.Picks[1].Key = (Action{Kind: ActBuild, Design: 43}).Key()
	b.Picks[2].Key = (Action{Kind: ActBuild, Design: 32}).Key()
	pending, _, err = resolveAbility(t, g, b)
	if err != nil {
		t.Fatal(err)
	}
	before := pending.State.Cities[1].Coins
	done, events, err = resolveAbility(t, pending, requiredBatch(pending, 0, Action{Kind: ActBuildDiscarded, Design: 11}))
	if err != nil {
		t.Fatal(err)
	}
	// Two trade coins from the stage payment and two Bazar coins.
	if before != 34 || done.State.Cities[1].Coins != before {
		t.Fatal("earlier Bazar effect recomputed", "before", before, "after", done.State.Cities[1].Coins, "events", events)
	}
	for _, e := range events {
		if e.Kind == EvCoins && e.Seat == 1 {
			t.Fatal("old Bazar emitted income")
		}
	}
}

func TestRuling19DecoratorsEveryBoardSideLength(t *testing.T) {
	for board := catalog.BoardID(1); board <= catalog.NumBoards; board++ {
		for _, side := range []catalog.Side{catalog.Day, catalog.Night} {
			b, _ := catalog.BoardByID(board)
			numStages := b.Sides[side-1].NumStages
			for _, stages := range []uint8{numStages - 1, numStages} {
				g := abilityScenario(t, 3, 6, [MaxSeats]wonderCity{0: {board: board, side: side, stages: stages, buildings: []catalog.DesignID{73}}}, [MaxSeats][]catalog.DesignID{})
				d, _ := catalog.DesignByID(73)
				want := int16(0)
				if stages == numStages {
					want = 7
				}
				if got := endPoints(&g.State, 0, d.Effects[0]); got != want {
					t.Fatal("decorators side length", board, side, stages, "got", got, "want", want, "expected stages", numStages, "board", b, "actual city", g.State.Cities[0])
				}
			}
		}
	}
}

func TestDiscardNameRuleAndNoDecline(t *testing.T) {
	g := abilityScenario(t, 2, 4, [MaxSeats]wonderCity{0: {board: catalog.BoardHalikarnassos, side: catalog.Night, buildings: []catalog.DesignID{2, 11}}, 1: {board: catalog.BoardOlympia}}, [MaxSeats][]catalog.DesignID{0: {39}}, 32)
	pending, _, err := resolveAbility(t, g, actionBatch(g, 0, stagePick(t, g, 0, 39)))
	if err != nil {
		t.Fatal(err)
	}
	if pending.State.Phase != PhaseDiscardBuild || pending.State.Discard[32] == 0 {
		t.Fatal("twin pile fixture")
	}
	if len(buildChoices(pending.Legal(0), ActBuildDiscarded, 32)) != 0 {
		t.Fatal("discard build bypassed twin name rule")
	}
	before := pending
	next, events, err := resolveAbility(t, pending, requiredBatch(pending, 0, Action{}))
	if err == nil || next != before || pending != before || len(events) != 0 {
		t.Fatal("discard could be declined")
	}
	if !reflect.DeepEqual(pending.State.Hands, before.State.Hands) {
		t.Fatal("pending point passed hands")
	}
}

// Forced-point sweeps check all intermediate states, not just the final score.
func resolveAbility(t *testing.T, g Game, b Batch) (Game, []Event, error) {
	t.Helper()
	before := g
	next, events, err := g.Resolve(b)
	if err != nil {
		if next != before || len(events) != 0 || g != before {
			t.Fatal("failed ability transition mutated values")
		}
		return next, events, err
	}
	var picks [MaxSeats]Action
	for _, p := range b.Picks {
		a, e := ParseKey(p.Key)
		if e != nil {
			t.Fatal(e)
		}
		picks[p.Seat] = a
	}
	core, coreEvents, e := ResolveState(&g.State, &picks, nil)
	reverse, reverseEvents, re := resolveStateOrder(&g.State, &picks, nil, true)
	if e != nil || re != nil || core != next.State || reverse != core || !reflect.DeepEqual(events, coreEvents) || !reflect.DeepEqual(events, reverseEvents) || g != before {
		t.Fatal("forced ability core/order mismatch", e, re)
	}
	checkCoinLedger(t, &g.State, &next.State, &picks, events)
	if err := next.CheckInvariants(); err != nil {
		t.Fatal("immediately after ability transition", err)
	}
	for seat := Seat(0); seat < Seat(next.State.Seats); seat++ {
		if (len(next.Legal(seat)) > 0) != next.Point().Required.Has(seat) {
			t.Fatal("required seat/offer mismatch")
		}
	}
	replay, e := Replay(Setup{Seed: g.State.Seed, Seats: g.State.Seats, Options: g.State.Opts, Scenario: ScenarioFromGame(g)}, []Batch{b})
	if e != nil || replay != next {
		t.Fatal("forced ability replay differs", e)
	}
	return next, events, nil
}

func TestDiscardMilitaryAndFinalScoringWait(t *testing.T) {
	g := abilityScenario(t, 3, 6, [MaxSeats]wonderCity{0: {board: catalog.BoardHalikarnassos, side: catalog.Night, buildings: []catalog.DesignID{2}}, 1: {board: catalog.BoardOlympia}}, [MaxSeats][]catalog.DesignID{0: {51, 56}})
	pending, events, err := resolveAbility(t, g, actionBatch(g, 0, stagePick(t, g, 0, 56)))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range events {
		if e.Kind == EvMilitary || e.Kind == EvGameOver {
			t.Fatal("end before discard point")
		}
	}
	done, events, err := resolveAbility(t, pending, requiredBatch(pending, 0, Action{Kind: ActBuildDiscarded, Design: 51}))
	if err != nil {
		t.Fatal(err)
	}
	if done.State.Phase != PhaseOver || done.State.Cities[0].Victories[2] != 2 || done.State.Result.Scores[0].Military != 10 || done.State.Result != referenceResult(&done.State) {
		t.Fatal("discard shields omitted from final conflict/score")
	}
	if events[0].Kind != EvDiscardBuild || events[len(events)-1].Kind != EvGameOver {
		t.Fatal("discard/military/end order")
	}
}
