package rules

// Additional rule-derived order and record scenarios from the independent review.

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// Ruling 2: a card sold this turn by another seat is in the pile at step 6.
// The ordinary hands wait unpassed during the discard point.
func TestWonderSameTurnSaleOffered(t *testing.T) {
	old := []catalog.DesignID{2, 5, 7, 8, 11, 12, 13}
	var specs [MaxSeats]wonderCity
	var wanted [MaxSeats][]catalog.DesignID
	specs[0] = wonderCity{board: catalog.BoardHalikarnassos, side: catalog.Night}
	for s, id := range old {
		specs[s].buildings = []catalog.DesignID{id}
	}
	specs[1].board = catalog.BoardOlympia
	wanted[0] = []catalog.DesignID{17}
	for s, id := range []catalog.DesignID{21, 22, 24, 23, 19, 20} {
		wanted[s+1] = []catalog.DesignID{id}
	}
	g := abilityScenario(t, 1, 2, specs, wanted)
	for id, n := range g.State.Discard {
		if n != 0 {
			t.Fatal("fixture pile not empty", id, n)
		}
	}
	b := actionBatch(g, 0, stagePick(t, g, 0, 17))
	for s := 1; s < 7; s++ {
		kind := ActBuild
		if wanted[s][0] == 23 {
			kind = ActSell
		}
		b.Picks[s].Key = (Action{Kind: kind, Design: wanted[s][0]}).Key()
	}
	pending, _, err := resolveAbility(t, g, b)
	if err != nil {
		t.Fatal(err)
	}
	if pending.State.Phase != PhaseDiscardBuild {
		t.Fatal("same-turn sale did not open the discard point", pending.State.Phase)
	}
	cs := pending.Legal(0)
	if len(cs) != 1 || cs[0].Design != 23 || cs[0].Kind != ActBuildDiscarded {
		t.Fatal("expected only the same-turn Tavern", cs)
	}
	// Hands wait: each seat holds its turn-2 hand minus its pick, unrotated.
	for s := 0; s < 7; s++ {
		h := g.State.Hands[s]
		picked := wanted[s][0]
		takeCard(&h, picked)
		if pending.State.Hands[s] != h {
			t.Fatal("hands moved during the discard point", s)
		}
	}
	coins := pending.State.Cities[0].Coins
	done, events, err := resolveAbility(t, pending, requiredBatch(pending, 0, cs[0].Action))
	if err != nil {
		t.Fatal(err)
	}
	if done.State.Cities[0].Coins != coins+5 || done.State.Turn != 3 || done.State.Phase != PhaseTurn {
		t.Fatal("rebuilt Tavern income or pass", done.State.Cities[0].Coins, coins)
	}
	if events[len(events)-1].Kind != EvHandsPassed {
		t.Fatal("hands did not pass after the discard build", events)
	}
	t.Logf("same-turn sale offered: %v", cs[0].Key())
}

// Ruling 5: the kept card may buy production a neighbour built in step 4 of
// the sixth turn, and spend the sixth turn's sale income.
func TestWonderKeptCardBuysNeighbourSixthTurnBuilding(t *testing.T) {
	g := abilityScenario(t, 1, 6, [MaxSeats]wonderCity{
		0: {board: catalog.BoardGizah},
		1: {board: catalog.BoardBabylon, side: catalog.Night, stages: 1},
		2: {board: catalog.BoardEphesos},
		3: {buildings: []catalog.DesignID{7}},
		4: {buildings: []catalog.DesignID{14}},
	}, [MaxSeats][]catalog.DesignID{0: {7, 18}, 1: {14, 23}})
	g.State.Cities[1].Coins = 0
	if offers := buildChoices(g.Legal(1), ActBuild, 14); len(offers) != 0 {
		t.Fatal("Barracks should be unaffordable on turn 6", offers)
	}
	b := actionBatch(g, 0, Action{Kind: ActBuild, Design: 7})
	b.Picks[1].Key = (Action{Kind: ActSell, Design: 23}).Key()
	last, _, err := resolveAbility(t, g, b)
	if err != nil {
		t.Fatal(err)
	}
	if last.State.Phase != PhaseLastCard || last.State.Cities[1].Kept != 14 || last.State.Cities[1].Coins != 3 {
		t.Fatal("kept point", last.State.Phase, last.State.Cities[1].Kept, last.State.Cities[1].Coins)
	}
	offers := buildChoices(last.Legal(1), ActBuild, 14)
	if len(offers) != 1 || offers[0].Right != 2 || offers[0].Left != 0 {
		t.Fatal("kept card cannot buy the neighbour's sixth-turn Ore Vein", offers)
	}
	before0 := last.State.Cities[0].Coins
	done, _, err := resolveAbility(t, last, requiredBatch(last, 1, offers[0].Action))
	if err != nil {
		t.Fatal(err)
	}
	if done.State.Cities[0].Coins != before0+2 || done.State.Cities[1].Coins != 1 || done.State.Age != 2 {
		t.Fatal("payment or transition")
	}
}

// Rulings 7 and 8 in a later Age: both Olympia night windows recur.
func TestWonderOlympiaNightWindowsLaterAge(t *testing.T) {
	g := abilityScenario(t, 2, 6, [MaxSeats]wonderCity{
		0: {board: catalog.BoardOlympia, side: catalog.Night, stages: 1, buildings: []catalog.DesignID{28}},
	}, [MaxSeats][]catalog.DesignID{0: {39}, 1: {35, 40}})
	next, _, err := resolveAbility(t, g, actionBatch(g, 0, stagePick(t, g, 0, 39)))
	if err != nil {
		t.Fatal(err)
	}
	if next.State.Age != 3 || next.State.Turn != 1 || next.State.Cities[0].Stages != 2 {
		t.Fatal("setup", next.State.Age, next.State.Turn)
	}
	firstHits, lastHits := 0, 0
	for turn := 1; turn <= 6; turn++ {
		for _, c := range next.Legal(0) {
			if c.Kind == ActStage && c.Free != FreeNone {
				t.Fatal("free stage")
			}
			if c.Kind != ActBuild {
				continue
			}
			switch {
			case turn == 1 && c.Free != FreeNone:
				firstHits++
			case turn == 6 && c.Free != FreeNone:
				lastHits++
			case c.Free == FreeFirstTurn || c.Free == FreeLastTurn:
				t.Fatal("window outside turns 1 and 6", turn, c)
			}
		}
		if turn < 6 {
			next, _, err = resolveAbility(t, next, salesBatch(next))
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if firstHits == 0 || lastHits == 0 {
		t.Fatal("Age III windows missing", firstHits, lastHits)
	}
	t.Logf("Age III free offers: turn 1 %d, turn 6 %d", firstHits, lastHits)
}

// Rulings 1 and 29 together: Babylon builds its ability stage on turn 6 while
// Halikarnassos builds a discard stage. Discard point, then last card, then
// military and the next Age.
func TestWonderDiscardThenNewKeptThenMilitary(t *testing.T) {
	g := abilityScenario(t, 2, 6, [MaxSeats]wonderCity{
		0: {board: catalog.BoardHalikarnassos, side: catalog.Night, buildings: []catalog.DesignID{28}},
		1: {board: catalog.BoardBabylon, side: catalog.Night, buildings: []catalog.DesignID{30}},
	}, [MaxSeats][]catalog.DesignID{0: {39, 42}, 1: {40, 41}})
	b := actionBatch(g, 0, stagePick(t, g, 0, 39))
	b.Picks[1].Key = stagePick(t, g, 1, 40).Key()
	pending, events, err := resolveAbility(t, g, b)
	if err != nil {
		t.Fatal(err)
	}
	if pending.Point().ID() != "age-2-turn-6-discard-build" || pending.State.Cities[1].Kept != 41 {
		t.Fatal("expected discard point first with Babylon's new keep", pending.Point().ID(), pending.State.Cities[1].Kept)
	}
	for _, e := range events {
		if e.Kind == EvMilitary || e.Kind == EvAgeStart {
			t.Fatal("military too early")
		}
	}
	for _, c := range pending.Legal(0) {
		if c.Design == 41 && pending.State.Discard[41] == 0 {
			t.Fatal("kept card offered")
		}
	}
	cs := pending.Legal(0)
	last, events, err := resolveAbility(t, pending, requiredBatch(pending, 0, cs[0].Action))
	if err != nil {
		t.Fatal(err)
	}
	if last.Point().ID() != "age-2-last-card" || last.Point().Required != Seat(1).Mask() {
		t.Fatal("expected last card after the discard build", last.Point().ID())
	}
	for _, e := range events {
		if e.Kind == EvMilitary || e.Kind == EvAgeStart {
			t.Fatal("military before the kept card")
		}
	}
	done, events, err := resolveAbility(t, last, salesBatch(last))
	if err != nil {
		t.Fatal(err)
	}
	mil := 0
	for _, e := range events {
		if e.Kind == EvMilitary {
			mil++
		}
	}
	if mil != 7 || events[len(events)-1].Kind != EvAgeStart || done.State.Age != 3 {
		t.Fatal("military and deal after kept card", mil, events)
	}
}

// Everything the views slice needs is in the Record: rebuild every batch from
// the Record and the public sides alone, and replay to the identical game.
func replayAbilityRecord(t *testing.T, final Game, setup Setup) []Batch {
	t.Helper()
	g, err := NewGame(setup)
	if err != nil {
		t.Fatal(err)
	}
	var out []Batch
	for g.State.Phase != PhaseOver && g.State.Points < final.State.Points {
		p := g.Point()
		b := Batch{Point: p.ID()}
		for s := Seat(0); s < Seat(g.State.Seats); s++ {
			if !p.Required.Has(s) {
				continue
			}
			var a Action
			switch p.Kind {
			case PointSides:
				a = Action{Kind: ActSide, Side: final.State.Cities[s].Side}
			case PointTurn, PointLastCard:
				pl := final.Record.Plays[p.Age-1][p.Turn-1][s]
				a = Action{Kind: pl.Kind, Design: final.Record.Picks[p.Age-1][p.Turn-1][s], Left: pl.Left, Right: pl.Right}
			case PointDiscardBuild:
				a = Action{Kind: ActBuildDiscarded, Design: final.Record.Rebuilt[p.Age-1][p.Turn-1][s]}
			}
			b.Picks = append(b.Picks, Pick{Seat: s, Key: a.Key()})
		}
		next, _, err := g.Resolve(b)
		if err != nil {
			t.Fatal("record does not rebuild batch", p.ID(), err)
		}
		out = append(out, b)
		g = next
	}
	if g != final {
		t.Fatal("record replay differs from the played game")
	}
	return out
}

func TestWonderRecordRebuildsEveryKey(t *testing.T) {
	games, looks, kepts := 0, 0, 0
	for n := uint8(3); n <= 7; n++ {
		for seed := uint64(0); seed < 60; seed++ {
			setup := Setup{Seats: n, Seed: seed, Options: Options{Sides: SideRule(seed % 3)}}
			g, err := NewGame(setup)
			if err != nil {
				t.Fatal(err)
			}
			for steps := 0; g.State.Phase != PhaseOver; steps++ {
				p := g.Point()
				b := Batch{Point: p.ID()}
				for s := Seat(0); s < Seat(n); s++ {
					if !p.Required.Has(s) {
						continue
					}
					cs := g.Legal(s)
					c := cs[(int(seed)*5+steps*3+int(s)*7)%len(cs)]
					if seed%2 == 0 {
						for _, o := range cs {
							if o.Kind == ActStage {
								c = o
								break
							}
						}
					}
					b.Picks = append(b.Picks, Pick{Seat: s, Key: c.Key()})
				}
				if p.Kind == PointDiscardBuild {
					looks++
				}
				if p.Kind == PointLastCard {
					kepts++
				}
				next, _, err := g.Resolve(b)
				if err != nil {
					t.Fatal(err)
				}
				if err := next.CheckInvariants(); err != nil {
					t.Fatal(err)
				}
				g = next
			}
			replayAbilityRecord(t, g, setup)
			games++
		}
	}
	t.Logf("record rebuilt %d games, %d discard points, %d last cards", games, looks, kepts)
}

// A skipped search on the sixth turn (nonempty all-duplicate pile, including
// the owner's own leftover) still leads to Babylon's kept card, then military.
func TestWonderSixthTurnSkipThenKeptCard(t *testing.T) {
	hands := [MaxSeats][]catalog.DesignID{0: {17, 5}, 1: {13, 14}, 2: {19, 7}, 3: {20, 8}, 4: {21, 11}, 5: {22, 12}, 6: {23, 2}}
	g := abilityScenario(t, 1, 6, [MaxSeats]wonderCity{
		0: {board: catalog.BoardHalikarnassos, side: catalog.Night, buildings: []catalog.DesignID{1, 2, 5, 7, 8, 11, 12}},
		1: {board: catalog.BoardBabylon, side: catalog.Night, stages: 1},
	}, hands)
	// Empty the pile into cities 1 to 6 without repeating a name or blocking
	// the card each seat builds this turn.
	for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
		for g.State.Discard[id] > 0 {
			d, _ := catalog.DesignByID(id)
			placed := false
			for s := 1; s < 7 && !placed; s++ {
				c := &g.State.Cities[s]
				chosen, _ := catalog.DesignByID(hands[s][0])
				if !c.Buildings.And(catalog.T.ByName[d.NameID]).Empty() || chosen.NameID == d.NameID {
					continue
				}
				c.Buildings.Add(id)
				g.State.Discard[id]--
				placed = true
			}
			if !placed {
				t.Fatal("cannot place", id)
			}
		}
	}
	g, err := NewGameFromScenario(g.State.Seed, g.State.Opts, ScenarioFromGame(g))
	if err != nil {
		t.Fatal(err)
	}
	b := actionBatch(g, 0, stagePick(t, g, 0, 17))
	for s := 1; s < 7; s++ {
		b.Picks[s].Key = (Action{Kind: ActBuild, Design: hands[s][0]}).Key()
	}
	next, events, err := resolveAbility(t, g, b)
	if err != nil {
		t.Fatal(err)
	}
	pile := 0
	for _, n := range next.State.Discard {
		pile += int(n)
	}
	skipped := false
	for _, e := range events {
		if e.Kind == EvDiscardBuildNothing && e.Seat == 0 {
			skipped = true
		}
		if e.Kind == EvMilitary {
			t.Fatal("military before kept card")
		}
	}
	if !skipped || pile != 6 || next.Point().ID() != "age-1-last-card" || next.State.Cities[1].Kept != 14 {
		t.Fatal("sixth-turn skip did not continue to the kept card", skipped, pile, next.Point().ID())
	}
	done, events, err := resolveAbility(t, next, salesBatch(next))
	if err != nil {
		t.Fatal(err)
	}
	if done.State.Age != 2 || events[len(events)-1].Kind != EvAgeStart {
		t.Fatal("no military and deal after kept card")
	}
	replayAbilityRecord(t, done, Setup{Seed: g.State.Seed, Seats: 7, Options: g.State.Opts, Scenario: ScenarioFromGame(g)})
}

func TestDiscardStageHistoryRequired(t *testing.T) {
	g := abilityScenario(t, 2, 3, [MaxSeats]wonderCity{0: {board: catalog.BoardHalikarnassos, side: catalog.Night, stages: 1, buildings: []catalog.DesignID{11, 13}}}, [MaxSeats][]catalog.DesignID{0: {39}})
	pending, _, err := resolveAbility(t, g, actionBatch(g, 0, stagePick(t, g, 0, 39)))
	if err != nil {
		t.Fatal(err)
	}
	for _, play := range []Play{{Kind: ActSell}, {Kind: ActStage, Stage: 1}} {
		bad := pending
		bad.Record.Plays[1][2][0] = play
		rejectsHistory(t, bad, "current discard stage")
	}
	done, _, err := resolveAbility(t, pending, salesBatch(pending))
	if err != nil {
		t.Fatal(err)
	}
	done.Record.Plays[1][2][0] = Play{Kind: ActSell}
	rejectsHistory(t, done, "rebuild lacks discard stage")
}
