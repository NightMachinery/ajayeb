package rules

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

func assertFree(t *testing.T, cs []Choice, id catalog.DesignID, reason FreeReason) {
	t.Helper()
	offers := buildChoices(cs, ActBuild, id)
	if len(offers) != 1 || offers[0].Free != reason || offers[0].Total != 0 || offers[0].Printed != 0 || offers[0].Left != 0 || offers[0].Right != 0 {
		t.Fatal("free building must have exactly one zero-payment offer", id, reason, offers)
	}
}
func assertNotFree(t *testing.T, cs []Choice, id catalog.DesignID) {
	t.Helper()
	for _, c := range buildChoices(cs, ActBuild, id) {
		if c.Free != FreeNone {
			t.Fatal("free window leaked", id, c)
		}
	}
}

func TestRulings6And21OlympiaColourAtStageReveal(t *testing.T) {
	g := abilityScenario(t, 2, 2, [MaxSeats]wonderCity{
		0: {board: catalog.BoardOlympia, stages: 1, buildings: []catalog.DesignID{5, 17}},
		1: {board: catalog.BoardBabylon},
	}, [MaxSeats][]catalog.DesignID{0: {39}, 1: {28, 35, 40}})
	stage := stagePick(t, g, 0, 39)
	// Stage 2 grants the ability only after placement, never to its carrier.
	assertNotFree(t, g.Legal(0), 39)
	next, _, err := resolveAbility(t, g, actionBatch(g, 0, stage))
	if err != nil {
		t.Fatal(err)
	}
	assertFree(t, next.Legal(0), 35, FreeFirstColour)
	assertNotFree(t, next.Legal(0), 40) // Altar already filled blue before stage 2.
	red := buildChoices(next.Legal(0), ActBuild, 35)[0]
	built, _, err := resolveAbility(t, next, actionBatch(next, 0, red.Action))
	if err != nil {
		t.Fatal(err)
	}
	// A second red design is paid even though it has the same resource cost.
	st := built.State
	st.Hands[0] = [HandSize]catalog.DesignID{36}
	assertNotFree(t, LegalState(&st, 0, nil), 36)
	if err := built.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
}

func TestRulings7And21OlympiaFirstTurnNeverStored(t *testing.T) {
	for _, stageOnFirst := range []bool{false, true} {
		stages := uint8(1)
		if stageOnFirst {
			stages = 0
		}
		g := abilityScenario(t, 2, 1, [MaxSeats]wonderCity{0: {board: catalog.BoardOlympia, side: catalog.Night, stages: stages, buildings: []catalog.DesignID{7}}, 1: {board: catalog.BoardRhodos}}, [MaxSeats][]catalog.DesignID{0: {39}})
		var a Action
		if stageOnFirst {
			assertNotFree(t, g.Legal(0), 39)
			a = stagePick(t, g, 0, 39)
		} else {
			assertFree(t, g.Legal(0), 39, FreeFirstTurn)
			a = Action{Kind: ActSell, Design: 39}
		}
		next, _, err := resolveAbility(t, g, actionBatch(g, 0, a))
		if err != nil {
			t.Fatal(err)
		}
		if next.State.Turn != 2 {
			t.Fatal("did not advance")
		}
		for _, c := range next.Legal(0) {
			if c.Free == FreeFirstTurn {
				t.Fatal("first-turn allowance stored")
			}
		}
		if err := next.CheckInvariants(); err != nil {
			t.Fatal(err)
		}
	}
	// The following Age has a new first-turn window. Selling in this one loses
	// it, even when the player could have constructed a free building instead.
	g := abilityScenario(t, 2, 6, [MaxSeats]wonderCity{0: {board: catalog.BoardOlympia, side: catalog.Night, stages: 1}}, [MaxSeats][]catalog.DesignID{})
	next, _, err := resolveAbility(t, g, salesBatch(g))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range next.State.Hands[0] {
		if id != 0 {
			assertFree(t, next.Legal(0), id, FreeFirstTurn)
		}
	}
	after, _, err := resolveAbility(t, next, salesBatch(next))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range after.Legal(0) {
		if c.Free == FreeFirstTurn {
			t.Fatal("first-turn allowance carried forward")
		}
	}
	if err := after.CheckInvariants(); err != nil {
		t.Fatal(err)
	}
}

func TestRulings8And21OlympiaSixthTurnWindow(t *testing.T) {
	for _, onSixth := range []bool{false, true} {
		turn := uint8(5)
		if onSixth {
			turn = 6
		}
		g := abilityScenario(t, 2, turn, [MaxSeats]wonderCity{
			0: {board: catalog.BoardOlympia, side: catalog.Night, stages: 1, buildings: []catalog.DesignID{28}},
		}, [MaxSeats][]catalog.DesignID{0: {39}, 1: {35, 40}})
		a := stagePick(t, g, 0, 39)
		if a.Left != 0 || a.Right != 0 {
			t.Fatal("own clay should pay the stage")
		}
		// An ability granted by this very stage cannot make its building carrier
		// free. The next stage also never inherits a free-building condition.
		assertNotFree(t, g.Legal(0), 39)
		next, _, err := resolveAbility(t, g, actionBatch(g, 0, a))
		if err != nil {
			t.Fatal(err)
		}
		if onSixth {
			if next.State.Age != 3 {
				t.Fatal("unexpected extra point")
			}
			for _, c := range next.Legal(0) {
				if c.Free == FreeLastTurn {
					t.Fatal("sixth-turn window survived into next Age")
				}
			}
		} else {
			if next.State.Turn != 6 {
				t.Fatal("missing sixth turn")
			}
			assertFree(t, next.Legal(0), 40, FreeLastTurn)
			for _, c := range next.Legal(0) {
				if c.Kind == ActStage && c.Free != FreeNone {
					t.Fatal("free stage")
				}
			}
			done, _, err := resolveAbility(t, next, salesBatch(next))
			if err != nil {
				t.Fatal(err)
			}
			if done.State.Age != 3 || done.State.Cities[0].Kept != 0 {
				t.Fatal("Olympia kept its leftover")
			}
			for _, c := range done.Legal(0) {
				if c.Free == FreeLastTurn {
					t.Fatal("last-turn allowance stored")
				}
			}
			if err := done.CheckInvariants(); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestWonderStagesEveryBoardAndSide(t *testing.T) {
	hits := 0
	for board := catalog.BoardID(1); board <= catalog.NumBoards; board++ {
		for _, side := range []catalog.Side{catalog.Day, catalog.Night} {
			definition, _ := catalog.BoardByID(board)
			length := definition.Sides[side-1].NumStages
			for stage := uint8(0); stage < length; stage++ {
				g := abilityScenario(t, 3, 3, [MaxSeats]wonderCity{0: {board: board, side: side, stages: stage, buildings: []catalog.DesignID{2, 4, 5, 7, 8, 11, 12, 13, 28, 29, 30, 31}}}, [MaxSeats][]catalog.DesignID{0: {51}})
				next, _, err := resolveAbility(t, g, actionBatch(g, 0, stagePick(t, g, 0, 51)))
				if err != nil {
					t.Fatal(board, side, stage, err)
				}
				if next.State.Cities[0].Stages != stage+1 {
					t.Fatal("stage not placed")
				}
				if err := next.CheckInvariants(); err != nil {
					t.Fatal(board, side, stage, "at ability point", err)
				}
				if next.State.Phase == PhaseDiscardBuild {
					next, _, err = resolveAbility(t, next, salesBatch(next))
					if err != nil {
						t.Fatal(err)
					}
					if err := next.CheckInvariants(); err != nil {
						t.Fatal(err)
					}
				}
				hits++
			}
		}
	}
	if hits != 41 {
		t.Fatal("missing stage", hits)
	}
	t.Logf("forced stage paths: %d, all 14 board/side combinations", hits)
}
