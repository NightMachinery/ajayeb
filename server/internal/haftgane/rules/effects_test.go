package rules

import (
	"reflect"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// A conserving terminal scenario with two cards of each ordinary colour in
// the owner and each neighbour, one owner guild, and 2/1/3 built stages. Each
// card case comes from real supply; no production interpreter shapes the wants.
// Seat 3 holds brown, grey and a stage to detect non-neighbour scope leaks.
func effectWorld(t *testing.T, board catalog.BoardID, side catalog.Side, stages uint8, card catalog.DesignID) Game {
	t.Helper()
	var g Game
	found := false
	for seed := uint64(0); seed < 100; seed++ {
		var err error
		g, err = NewGame(Setup{Seats: 7, Seed: seed, Options: Options{Sides: SidesDay}})
		if err != nil {
			t.Fatal(err)
		}
		boxed := false
		for _, id := range g.State.BoxGuilds {
			if id == 71 || (card >= 71 && id == card) {
				boxed = true
			}
		}
		if !boxed {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("scenario guilds not drawn")
	}
	for seat, hand := range g.State.Hands {
		for _, id := range hand {
			g.State.Discard[id]++
		}
		g.State.Hands[seat] = [HandSize]catalog.DesignID{}
	}
	for age := 1; age < NumAges; age++ {
		g.Record.Packets[age] = g.State.Deals[age]
		for _, hand := range g.State.Deals[age] {
			for _, id := range hand {
				if id != 0 {
					g.State.Discard[id]++
				}
			}
		}
		g.State.Deals[age] = [MaxSeats][HandSize]catalog.DesignID{}
	}
	order := [MaxSeats]catalog.BoardID{board}
	next := 1
	for b := catalog.BoardID(1); b <= catalog.NumBoards; b++ {
		if b != board {
			order[next] = b
			next++
		}
	}
	for s, b := range order {
		g.State.Cities[s] = City{Board: b, Side: catalog.Day, Coins: 3}
	}
	g.State.Cities[0].Side = side
	g.State.Cities[0].Stages = stages
	g.State.Cities[1].Stages = 1
	g.State.Cities[3].Stages = 1
	g.State.Cities[6].Stages = 3
	baseline := [MaxSeats][]catalog.DesignID{
		0: {2, 5, 11, 12, 14, 15, 17, 18, 21, 24, 25, 26, 71},
		1: {7, 8, 11, 13, 16, 35, 19, 20, 22, 23, 27, 47},
		3: {1, 34},
		6: {28, 29, 12, 13, 36, 37, 39, 40, 44, 45, 48, 49},
	}
	for s, ids := range baseline {
		for _, id := range ids {
			if g.State.Discard[id] == 0 {
				t.Fatal("baseline copy missing", id)
			}
			g.State.Discard[id]--
			g.State.Cities[s].Buildings.Add(id)
		}
	}
	if card != 0 && !g.State.Cities[0].Buildings.Has(card) {
		d, _ := catalog.DesignByID(card)
		// A twin replaces the same-name building, never duplicates it.
		for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
			old, _ := catalog.DesignByID(id)
			if old.NameID == d.NameID && g.State.Cities[0].Buildings.Has(id) {
				g.State.Cities[0].Buildings.Remove(id)
				g.State.Discard[id]++
			}
		}
		if g.State.Discard[card] == 0 {
			replaced := false
			for s := 1; s < 7 && !replaced; s++ {
				if !g.State.Cities[s].Buildings.Has(card) {
					continue
				}
				g.State.Cities[s].Buildings.Remove(card)
				g.State.Discard[card]++
				for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
					other, _ := catalog.DesignByID(id)
					if id != card && g.State.Discard[id] > 0 && other.Color == d.Color && g.State.Cities[s].Buildings.And(catalog.T.ByName[other.NameID]).Empty() {
						g.State.Discard[id]--
						g.State.Cities[s].Buildings.Add(id)
						replaced = true
						break
					}
				}
			}
			if !replaced {
				t.Fatal("scenario cannot free copy", card)
			}
		}
		g.State.Discard[card]--
		g.State.Cities[0].Buildings.Add(card)
	}
	for seat, c := range g.State.Cities {
		for k := uint8(0); k < c.Stages; k++ {
			for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
				if g.State.Discard[id] > 0 {
					g.State.Discard[id]--
					g.State.Cities[seat].Tucked[k] = id
					break
				}
			}
		}
	}
	g.State.Phase, g.State.Age, g.State.Turn, g.State.Decider = PhaseOver, 3, 6, NoSeat
	g.State.Result = Result{Over: true, Winners: uint8(AllSeats(7))}
	for s := range g.State.Result.Rank {
		g.State.Result.Rank[s] = 1
	}
	// A terminal scenario must now carry the real public-city scores. Wants
	// above remain literal and the sweep uses an independent scoring oracle.
	for seat, c := range g.State.Cities {
		var err error
		g.State.Cities[seat], err = cityCaches(c)
		if err != nil {
			t.Fatal(err)
		}
	}
	g.State.Result = scoreResult(&g.State)
	built, err := NewGameFromScenario(g.State.Seed, g.State.Opts, ScenarioFromGame(g))
	if err != nil {
		t.Fatal("effect scenario", err)
	}
	return built
}

func TestEveryCardIncomeAndScoring(t *testing.T) {
	// Literal expectations from rules.md/effects.md. The fixed fixture has six
	// brown/grey neighbour-plus-self cards, two own brown/grey/red, three own
	// yellow after a new yellow card, and six total wonder stages in scope.
	coins := [catalog.NumDesigns + 1]int16{23: 5, 43: 12, 46: 6, 61: 6, 62: 4, 63: 2, 64: 3, 65: 6}
	points := [catalog.NumDesigns + 1]int16{17: 3, 18: 3, 19: 3, 20: 3, 39: 5, 40: 4, 41: 4, 42: 4, 56: 5, 57: 8, 58: 7, 59: 6, 60: 6, 61: 2, 62: 4, 63: 2, 64: 3, 65: 2, 71: 6, 72: 8, 74: 4, 75: 4, 77: 6, 78: 4, 79: 4, 80: 4}
	green := [catalog.NumDesigns + 1]int16{27: 10, 47: 5, 48: 10, 49: 5, 50: 5, 66: 5, 67: 5, 68: 10, 69: 10, 70: 5, 76: 10}
	for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
		d, _ := catalog.DesignByID(id)
		t.Run(d.Slug, func(t *testing.T) {
			g := effectWorld(t, catalog.BoardGizah, catalog.Day, 2, id)
			before := g
			st := g.State
			got, err := applyWhenPlaced(&st, 0, d.Effects)
			if err != nil || got != coins[id] || st.Cities[0].Coins != 3+coins[id] {
				t.Fatalf("income %d want %d: %v", got, coins[id], err)
			}
			var p int16
			for _, e := range d.Effects {
				p += endPoints(&g.State, 0, e)
			}
			if p != points[id] {
				t.Fatalf("points %d want %d", p, points[id])
			}
			score := scoreCity(&g.State, 0)
			wantBlue := int16(6)
			if d.Color == catalog.Blue && id != 17 && id != 18 {
				wantBlue += points[id]
			}
			wantYellow := int16(0)
			if d.Color == catalog.Yellow {
				wantYellow = points[id]
			}
			wantPurple := int16(6)
			if d.Color == catalog.Purple && id != 71 {
				wantPurple += points[id]
			}
			wantGreen := int16(2)
			if green[id] != 0 {
				wantGreen = green[id]
			}
			if score.Wonder != 8 || score.Treasure != 1 || score.Military != 0 || score.Blue != wantBlue || score.Yellow != wantYellow || score.Green != wantGreen || score.Purple != wantPurple {
				t.Fatalf("holder categories: %+v", score)
			}
			if g != before {
				t.Fatal("scoring mutated the game")
			}
		})
	}
}

func TestEveryStageCoinAndPointEffects(t *testing.T) {
	type want struct {
		board         catalog.BoardID
		side          catalog.Side
		points, coins []int16
	}
	for _, tc := range []want{
		{1, catalog.Day, []int16{3, 0, 7}, []int16{0, 0, 0}}, {1, catalog.Night, []int16{0, 0, 7}, []int16{0, 0, 0}},
		{2, catalog.Day, []int16{3, 0, 7}, []int16{0, 0, 0}}, {2, catalog.Night, []int16{0, 0}, []int16{0, 0}},
		{3, catalog.Day, []int16{3, 0, 7}, []int16{0, 9, 0}}, {3, catalog.Night, []int16{2, 3, 5}, []int16{4, 4, 4}},
		{4, catalog.Day, []int16{3, 5, 7}, []int16{0, 0, 0}}, {4, catalog.Night, []int16{3, 5, 5, 7}, []int16{0, 0, 0, 0}},
		{5, catalog.Day, []int16{3, 0, 7}, []int16{0, 0, 0}}, {5, catalog.Night, []int16{2, 1, 0}, []int16{0, 0, 0}},
		{6, catalog.Day, []int16{3, 0, 7}, []int16{0, 0, 0}}, {6, catalog.Night, []int16{2, 3, 5}, []int16{0, 0, 0}},
		{7, catalog.Day, []int16{3, 0, 7}, []int16{0, 0, 0}}, {7, catalog.Night, []int16{3, 4}, []int16{3, 4}},
	} {
		total := int16(0)
		for i, want := range tc.points {
			total += want
			g := effectWorld(t, tc.board, tc.side, uint8(i+1), 0)
			b, _ := catalog.BoardByID(tc.board)
			st := g.State
			got, err := applyWhenPlaced(&st, 0, b.Sides[tc.side-1].Stages[i].Effects)
			if err != nil || got != tc.coins[i] || scoreCity(&g.State, 0).Wonder != total {
				t.Fatalf("board%s side%s stage%d: income%d points%+v err%v", tc.board, tc.side, i+1, got, scoreCity(&g.State, 0), err)
			}
			if tc.board == catalog.BoardBabylon && i >= 1 && scoreCity(&g.State, 0).Green != 10 {
				t.Fatal("stage science choice must score in green")
			}
		}
	}
}

func ageTwoCoinScenario(t *testing.T, old catalog.DesignID) Game {
	t.Helper()
	g, err := NewGame(Setup{Seats: 7, Seed: 8, Options: Options{Sides: SidesDay}})
	if err != nil {
		t.Fatal(err)
	}
	for s, hand := range g.State.Hands {
		for _, id := range hand {
			g.State.Discard[id]++
		}
		g.State.Hands[s] = [HandSize]catalog.DesignID{}
	}
	if err := dealAge(&g, 2); err != nil {
		t.Fatal(err)
	}
	if old != 0 {
		g.State.Discard[old]--
		g.State.Cities[0].Buildings.Add(old)
	}
	g, err = NewGameFromScenario(g.State.Seed, g.State.Opts, ScenarioFromGame(g))
	if err != nil {
		t.Fatal(err)
	}
	return g
}
func forceHand(t *testing.T, g *Game, seat Seat, id catalog.DesignID) {
	t.Helper()
	for _, got := range g.State.Hands[seat] {
		if got == id {
			return
		}
	}
	for s, hand := range g.State.Hands {
		for k, got := range hand {
			if got == id {
				g.State.Hands[s][k], g.State.Hands[seat][0] = g.State.Hands[seat][0], got
				return
			}
		}
	}
	t.Fatal("card absent from current packets", id)
}

func TestIncomeVineyardBazarSameTurnNeighbours(t *testing.T) {
	for _, tc := range []struct {
		design, old, left, right catalog.DesignID
		income                   int16
	}{{46, 2, 28, 29, 3}, {43, 11, 32, 33, 6}} {
		g := ageTwoCoinScenario(t, tc.old)
		forceHand(t, &g, 0, tc.design)
		forceHand(t, &g, 1, tc.left)
		forceHand(t, &g, 6, tc.right)
		for s := range g.State.Hands {
			canonicalIDs(g.State.Hands[s][:])
		}
		b := salesBatch(g)
		for _, a := range []struct {
			seat Seat
			id   catalog.DesignID
		}{{0, tc.design}, {1, tc.left}, {6, tc.right}} {
			b.Picks[a.seat].Key = (Action{Kind: ActBuild, Design: a.id}).Key()
		}
		next, events, err := g.Resolve(b)
		if err != nil {
			t.Fatal(err)
		}
		if next.State.Cities[0].Coins != 3+tc.income {
			t.Fatal("same-turn count omitted self/neighbours", tc.design, next.State.Cities[0].Coins)
		}
		found := false
		for _, e := range events {
			if e.Kind == EvCoins && e.Seat == 0 {
				if e.Source != CoinDesign || e.Design != tc.design || e.Delta != tc.income {
					t.Fatal("when-placed event", e)
				}
				found = true
			}
		}
		if !found {
			t.Fatal("missing card income event")
		}
		if err := next.CheckInvariants(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestIncomeNotReplayedOrSpendableEarly(t *testing.T) {
	g := ageTwoCoinScenario(t, 23)
	next, events, err := g.Resolve(salesBatch(g))
	if err != nil {
		t.Fatal(err)
	}
	if next.State.Cities[0].Coins != 6 {
		t.Fatal("old Tavern income replayed")
	}
	for _, e := range events {
		if e.Kind == EvCoins && e.Source == CoinDesign {
			t.Fatal("old building emitted new income")
		}
	}
	// A stage can grant coins after it is paid, never as part of the purse.
	st := State{Seats: 3, Phase: PhaseTurn, Age: 1, Turn: 1, Decider: NoSeat}
	for s, b := range []catalog.BoardID{catalog.BoardEphesos, catalog.BoardGizah, catalog.BoardHalikarnassos} {
		c, err := cityCaches(City{Board: b, Side: catalog.Night})
		if err != nil {
			t.Fatal(err)
		}
		st.Cities[s] = c
		st.Hands[s] = [HandSize]catalog.DesignID{17, 18}
	}
	st.Cities[1].ForSale[catalog.Stone] = 2
	if len(buildChoices(LegalState(&st, 0, nil), ActStage, 17)) != 0 {
		t.Fatal("spent the stage's own income")
	}
	st.Cities[0].Coins = 4
	choices := buildChoices(LegalState(&st, 0, nil), ActStage, 17)
	if len(choices) != 1 {
		t.Fatal("stage should now be affordable", choices)
	}
	picks := [MaxSeats]Action{choices[0].Action, {Kind: ActSell, Design: 17}, {Kind: ActSell, Design: 17}}
	got, evs, err := ResolveState(&st, &picks, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Cities[0].Coins != 4 {
		t.Fatal("Ephesos income not applied after payment")
	}
	want := Event{Kind: EvCoins, Seat: 0, Source: CoinStage, Stage: 1, Delta: 4}
	found := false
	for _, e := range evs {
		if e == want {
			found = true
		}
		if e.Kind == EvCoins && e.Source == CoinStage && e.Design != 0 {
			t.Fatal("stage income revealed tucked design")
		}
	}
	if !found {
		t.Fatal("stage income event", evs)
	}
}

func TestChoiceProducersSaleability(t *testing.T) {
	st := State{Seats: 3}
	for seat, board := range []catalog.BoardID{catalog.BoardRhodos, catalog.BoardEphesos, catalog.BoardHalikarnassos} {
		c, err := cityCaches(City{Board: board, Side: catalog.Day, Coins: 20})
		if err != nil {
			t.Fatal(err)
		}
		st.Cities[seat] = c
	}
	st.Cities[1].Buildings.Add(44)
	st.Cities[1].Buildings.Add(45)
	var err error
	st.Cities[1], err = cityCaches(st.Cities[1])
	if err != nil {
		t.Fatal(err)
	}
	if p := PaymentsState(&st, 1, catalog.MustCost("WG")); p.Count != 1 || p.Outcomes[0].Left != 0 || p.Outcomes[0].Right != 0 {
		t.Fatal("own yellow producers unusable", p)
	}
	if p := PaymentsState(&st, 0, catalog.MustCost("W")); p.Count != 0 {
		t.Fatal("bought a yellow producer", p)
	}
	st.Cities[1].Buildings.Add(1)
	st.Cities[1], err = cityCaches(st.Cities[1])
	if err != nil {
		t.Fatal(err)
	}
	if p := PaymentsState(&st, 0, catalog.MustCost("C")); p.Count != 1 || p.Outcomes[0].Left != 2 {
		t.Fatal("could not buy brown choice production", p)
	}
	c, err := cityCaches(City{Board: catalog.BoardAlexandria, Side: catalog.Night, Stages: 2})
	if err != nil {
		t.Fatal(err)
	}
	if c.NChoice != 2 || c.NChoiceForSale != 0 || c.ForSale[catalog.Glass] != 1 {
		t.Fatal("wonder production saleability", c)
	}
}

func TestIncomeOverflowAtomic(t *testing.T) {
	g := ageTwoCoinScenario(t, 0)
	g.State.Cities[0].Coins = 32767
	forceHand(t, &g, 0, 46)
	for s := range g.State.Hands {
		canonicalIDs(g.State.Hands[s][:])
	}
	g.State.Cities[1].Buildings.Add(2)
	g.State.Cities[1], _ = cityCaches(g.State.Cities[1]) // flat-core overflow sentinel
	var picks [MaxSeats]Action
	for s := Seat(0); s < 7; s++ {
		picks[s] = Action{Kind: ActSell, Design: g.State.Hands[s][0]}
	}
	picks[0] = Action{Kind: ActBuild, Design: 46}
	before := g.State
	prefix := []Event{{Kind: EvAgeStart, Age: 2}}
	got, events, err := ResolveState(&g.State, &picks, prefix)
	if err == nil || got != before || g.State != before || !reflect.DeepEqual(events, prefix) {
		t.Fatal("overflow was not atomic", err)
	}
}

func TestIncomeTimingAndAggregation(t *testing.T) {
	g := effectWorld(t, catalog.BoardGizah, catalog.Day, 2, 0)
	st := g.State
	effects := catalog.Fx(catalog.Coins(1), catalog.CoinsPerCard(catalog.Self, 2, catalog.Brown), catalog.CoinsPerStage(catalog.Self, 3), catalog.Points(100), catalog.Produce(catalog.Wood, 2))
	income, err := applyWhenPlaced(&st, 0, effects)
	if err != nil || income != 11 || st.Cities[0].Coins != 14 || st.Cities[0].Produce != g.State.Cities[0].Produce {
		t.Fatal("wrong timing or aggregation", income, err)
	}
	st = g.State
	st.Cities[0].Coins = 130
	tavern, _ := catalog.DesignByID(23)
	income, err = applyWhenPlaced(&st, 0, tavern.Effects)
	if err != nil || income != 5 || st.Cities[0].Coins != 135 {
		t.Fatal("physical reserve capped income", income, err)
	}
}

func TestIncomeSixthTurnBeforeFinalScoring(t *testing.T) {
	g := effectWorld(t, catalog.BoardGizah, catalog.Day, 2, 0)
	g.State.Phase, g.State.Turn = PhaseTurn, 6
	g.State.Result = Result{}
	g.State.Hands[0][0] = 64
	g.State.Discard[64]--
	for seat := 0; seat < 7; seat++ {
		for k := 0; k < 2; k++ {
			if g.State.Hands[seat][k] != 0 {
				continue
			}
			found := false
			for id := catalog.DesignID(51); id <= catalog.NumDesigns; id++ {
				if g.State.Discard[id] > 0 {
					g.State.Discard[id]--
					g.State.Hands[seat][k] = id
					found = true
					break
				}
			}
			if !found {
				t.Fatal("Age III scenario supply")
			}
		}
	}
	for seat := range g.State.Hands {
		canonicalIDs(g.State.Hands[seat][:])
	}
	var err error
	g, err = NewGameFromScenario(g.State.Seed, g.State.Opts, ScenarioFromGame(g))
	if err != nil {
		t.Fatal(err)
	}
	batch := salesBatch(g)
	batch.Picks[0].Key = (Action{Kind: ActBuild, Design: 64}).Key()
	next, events, err := g.Resolve(batch)
	if err != nil {
		t.Fatal(err)
	}
	if next.State.Cities[0].Coins != 6 || next.State.Result.Scores[0].Treasure != 2 || next.State.Phase != PhaseOver {
		t.Fatal("income omitted from final treasure", next.State.Result)
	}
	found := false
	for _, e := range events {
		if e.Kind == EvCoins && e.Source == CoinDesign && e.Design == 64 && e.Delta == 3 {
			found = true
		}
	}
	if !found {
		t.Fatal("missing final-turn placement income")
	}
	if err := next.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
}
