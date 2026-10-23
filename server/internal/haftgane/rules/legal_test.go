package rules

import (
	"reflect"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// Synthetic cards isolate the rule from production costs. The two resource
// cards deliberately have identical costs but different names, colours/chains.
func freeFixture(t *testing.T, reverse bool) State {
	t.Helper()
	defs := []catalog.Design{
		{ID: 1, NameID: 1, Slug: "provider", Name: "Provider", Age: 1, Color: catalog.Blue, Copies: catalog.At(3), Effects: catalog.Fx(catalog.Points(1)), ChainOut: catalog.Out(1)},
		{ID: 2, NameID: 2, Slug: "chained", Name: "Chained", Age: 2, Color: catalog.Red, Copies: catalog.At(3), Cost: catalog.MustCost("W"), ChainIn: 1, Effects: catalog.Fx(catalog.Shields(1))},
		{ID: 3, NameID: 3, Slug: "ordinary", Name: "Ordinary", Age: 2, Color: catalog.Blue, Copies: catalog.At(3), Cost: catalog.MustCost("W"), Effects: catalog.Fx(catalog.Points(1))},
		{ID: 4, NameID: 4, Slug: "printed", Name: "Printed", Age: 2, Color: catalog.Brown, Copies: catalog.At(3), Cost: catalog.MustCost("1c"), Effects: catalog.Fx(catalog.Produce(catalog.Wood, 1))},
	}
	boards := catalog.BaseBoards()
	boards[catalog.BoardOlympia-1].Sides[catalog.Day-1].Stages[2].Cost = catalog.MustCost("W")
	t.Cleanup(catalog.ReplaceForTest(defs, boards))
	st := State{Seats: 3, Phase: PhaseTurn, Age: 2, Turn: 1, Decider: NoSeat}
	for s := 0; s < 3; s++ {
		c, err := cityCaches(City{Board: catalog.BoardOlympia, Side: catalog.Day})
		if err != nil {
			t.Fatal(err)
		}
		st.Cities[s] = c
	}
	// Remove starting clay only for this flat-core fixture: no source sells wood.
	st.Hands[0] = [HandSize]catalog.DesignID{2, 3, 4}
	if reverse {
		st.Hands[0][0], st.Hands[0][1] = st.Hands[0][1], st.Hands[0][0]
	}
	return st
}

func buildChoices(cs []Choice, kind ActionKind, id catalog.DesignID) []Choice {
	var out []Choice
	for _, c := range cs {
		if c.Kind == kind && c.Design == id {
			out = append(out, c)
		}
	}
	return out
}

func TestLegalFreeChecksBeforeCostCache(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		for _, rule := range []string{"chain", "colour", "first-turn", "last-turn"} {
			t.Run(rule+map[bool]string{false: "/forward", true: "/reverse"}[reverse], func(t *testing.T) {
				st := freeFixture(t, reverse)
				switch rule {
				case "chain":
					st.Cities[0].Chains = 1 << 1
				case "colour":
					st.Cities[0].Abilities = AbilityFreeFirstOfColour
					st.Cities[0].Colors[catalog.Blue] = 1
				case "first-turn":
					st.Cities[0].Abilities = AbilityFreeFirstTurn
				case "last-turn":
					st.Cities[0].Abilities = AbilityFreeLastTurn
					st.Turn = 6
				}
				before := st
				cs := LegalState(&st, 0, nil)
				free := buildChoices(cs, ActBuild, 2)
				if len(free) != 1 || free[0].Left != 0 || free[0].Right != 0 || free[0].Printed != 0 || free[0].Total != 0 || free[0].Free == FreeNone {
					t.Fatalf("missing free card: %+v", free)
				}
				other := buildChoices(cs, ActBuild, 3)
				if rule == "chain" || rule == "colour" {
					if len(other) != 0 {
						t.Fatal("free result leaked across same-cost designs", other)
					}
				} else if len(other) != 1 {
					t.Fatal("turn condition did not free both buildings")
				}
				if st != before {
					t.Fatal("Legal mutated its frozen input")
				}
			})
		}
	}
}

func TestLegalStageNeverFreeAndPrintedFreeCostsNothing(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		t.Run(map[bool]string{false: "forward", true: "reverse"}[reverse], func(t *testing.T) {
			st := freeFixture(t, reverse)
			st.Cities[0].Stages = 2
			st.Cities[0].Abilities = AbilityFreeFirstOfColour | AbilityFreeFirstTurn | AbilityFreeLastTurn
			st.Turn = 6
			cs := LegalState(&st, 0, nil)
			if len(buildChoices(cs, ActStage, 2)) != 0 || len(buildChoices(cs, ActStage, 3)) != 0 {
				t.Fatal("building free condition leaked into same-cost stage")
			}
			paid := buildChoices(cs, ActBuild, 4)
			if len(paid) != 1 || paid[0].Total != 0 || paid[0].Printed != 0 {
				t.Fatalf("free printed coin cost: %+v", paid)
			}
			st.Turn = 2 // Test the debit without an automatic Age transition.
			st.Hands[0] = [HandSize]catalog.DesignID{4, 2}
			for s := 1; s < 3; s++ {
				st.Hands[s] = [HandSize]catalog.DesignID{3, 4}
			}
			picks := [MaxSeats]Action{{Kind: ActBuild, Design: 4}, {Kind: ActSell, Design: 3}, {Kind: ActSell, Design: 3}}
			next, evs, err := ResolveState(&st, &picks, nil)
			if err != nil || next.Cities[0].Coins != 0 {
				t.Fatal("free build paid printed coins", err, next.Cities[0].Coins)
			}
			for _, e := range evs {
				if e.Kind == EvPlay && e.Seat == 0 && (e.Printed != 0 || e.Left != 0 || e.Right != 0) {
					t.Fatal("free play charged", e)
				}
			}
		})
	}
}

func TestLegalFreePrecedenceAndNameRule(t *testing.T) {
	st := freeFixture(t, false)
	st.Cities[0].Chains = 1 << 1
	st.Cities[0].Abilities = AbilityFreeFirstOfColour | AbilityFreeFirstTurn
	cs := LegalState(&st, 0, nil)
	if got := buildChoices(cs, ActBuild, 2); len(got) != 1 || got[0].Free != FreeChain {
		t.Fatal("chain must be the attached reason first", got)
	}
	st.Cities[0].Buildings.Add(2)
	if len(buildChoices(LegalState(&st, 0, nil), ActBuild, 2)) != 0 {
		t.Fatal("free build bypassed name check")
	}
	if len(buildChoices(LegalState(&st, 0, nil), ActSell, 2)) != 1 {
		t.Fatal("duplicate building could not be sold")
	}
}

func TestCostAssessmentDistinguishesResourcesAndTreasure(t *testing.T) {
	st := State{Seats: 3}
	cost := catalog.MustCost("W")
	if got := AssessCostState(&st, 0, cost); got.Coverable || got.Affordable {
		t.Fatal("missing wood called coverable", got)
	}
	st.Cities[1].ForSale[catalog.Wood] = 1
	if got := AssessCostState(&st, 0, cost); !got.Coverable || got.Cheapest != 2 || got.Affordable {
		t.Fatal("treasure shortage lost cheapest total", got)
	}
	st.Cities[0].Coins = 2
	if got := AssessCostState(&st, 0, cost); !got.Coverable || got.Cheapest != 2 || !got.Affordable {
		t.Fatal("affordable quote", got)
	}
	st.Cities[0].Coins = 0
	if got := AssessCostState(&st, 0, catalog.MustCost("1c")); !got.Coverable || got.Cheapest != 1 || got.Affordable {
		t.Fatal("printed coins lost coverability", got)
	}
	if got := QuoteState(&st, 0, cost); got != (Quote{}) {
		t.Fatal("existing bot quote behavior changed", got)
	}
}

func TestLegalCanonicalAndBufferReuse(t *testing.T) {
	g, err := NewGame(Setup{Seats: 7, Seed: 2, Options: Options{Sides: SidesDay}})
	if err != nil {
		t.Fatal(err)
	}
	before := g
	for seat := Seat(0); seat < 7; seat++ {
		var buf [MaxTurnChoices]Choice
		cs := LegalState(&g.State, seat, buf[:0])
		if len(cs) == 0 || len(cs) > MaxTurnChoices || &cs[0] != &buf[0] {
			t.Fatal("offer size/buffer", len(cs))
		}
		if !reflect.DeepEqual(cs, g.Legal(seat)) {
			t.Fatal("Game Legal differs from state core")
		}
		seen := map[Action]bool{}
		for _, c := range cs {
			if c.Key() == "" || seen[c.Action] {
				t.Fatal("noncanonical/duplicate action", c)
			}
			seen[c.Action] = true
			if c.Total != uint16(c.Printed)+uint16(c.Left)+uint16(c.Right) {
				t.Fatal("breakdown arithmetic", c)
			}
		}
	}
	if g != before || len(LegalState(&g.State, NoSeat, nil)) != 0 || len(LegalState(nil, 0, nil)) != 0 {
		t.Fatal("invalid seat or mutation")
	}
	g.State.Phase = PhaseOver
	if len(g.Legal(0)) != 0 {
		t.Fatal("offer after end")
	}
}

func TestLegalTwinNameAndStageCompletion(t *testing.T) {
	st := State{Seats: 3, Phase: PhaseTurn, Age: 2, Turn: 1, Decider: NoSeat}
	c := City{Board: catalog.BoardGizah, Side: catalog.Day, Stages: 3, Coins: 30}
	c.Buildings.Add(11) // Age I Glassworks blocks the Age II twin, not a sale.
	var err error
	st.Cities[0], err = cityCaches(c)
	if err != nil {
		t.Fatal(err)
	}
	st.Hands[0] = [HandSize]catalog.DesignID{32, 32}
	cs := LegalState(&st, 0, nil)
	if len(cs) != 1 || cs[0].Action != (Action{Kind: ActSell, Design: 32}) {
		t.Fatal("twin/complete-stage rule or duplicate-copy dedup", cs)
	}
}

func TestLegalOlympiaFrozenConditions(t *testing.T) {
	st := freeFixture(t, false)
	st.Cities[0].Abilities = AbilityFreeFirstTurn | AbilityFreeLastTurn
	for turn := uint8(1); turn <= 7; turn++ {
		st.Turn = turn
		free := buildChoices(LegalState(&st, 0, nil), ActBuild, 2)
		if (len(free) == 1) != (turn == 1 || turn == 6) {
			t.Fatal("free turn condition", turn, free)
		}
	}
	st.Turn = 1
	st.Cities[0].Abilities = 0
	st.Cities[0].Stages = 0
	if len(buildChoices(LegalState(&st, 0, nil), ActBuild, 2)) != 0 {
		t.Fatal("unbuilt Olympia stage granted a free building")
	}
}

func TestLegalFreeAfterCachedOrdinaryRefusal(t *testing.T) {
	// Canonical id order, not just hand order, makes the failed ordinary quote
	// enter the cache before the free action. Test both directions of leakage.
	for _, reverse := range []bool{false, true} {
		for _, chain := range []bool{false, true} {
			t.Run(map[bool]string{false: "colour", true: "chain"}[chain]+map[bool]string{false: "/forward", true: "/reverse"}[reverse], func(t *testing.T) {
				st := freeFixture(t, reverse)
				var defs []catalog.Design
				for id := catalog.DesignID(1); id <= 4; id++ {
					d, _ := catalog.DesignByID(id)
					defs = append(defs, d)
				}
				defs[1].ChainIn, defs[1].Color = 0, catalog.Blue
				defs[2].ChainIn, defs[2].Color = 1, catalog.Red
				t.Cleanup(catalog.ReplaceForTest(defs, catalog.BaseBoards()))
				if chain {
					st.Cities[0].Chains = 1 << 1
				} else {
					st.Cities[0].Abilities = AbilityFreeFirstOfColour
					st.Cities[0].Colors[catalog.Blue] = 1
				}
				cs := LegalState(&st, 0, nil)
				if len(buildChoices(cs, ActBuild, 2)) != 0 || len(buildChoices(cs, ActBuild, 3)) != 1 {
					t.Fatal("cached ordinary refusal disabled the free action", cs)
				}
			})
		}
	}
}
