package engine

import (
	"slices"
	"testing"
)

func lastReplyScenario(ownCoins, opponentCoins int) Scenario {
	return Scenario{
		Age:     3,
		Wonders: wonderFill(nil, []string{"pyramids", "sphinx", "colossus", "hanging-gardens"}),
		Players: [2]ScenarioPlayer{
			{Coins: intPtr(ownCoins)},
			{Coins: intPtr(opponentCoins), Built: []ScenarioWonder{{Wonder: "pyramids"}, {Wonder: "sphinx"}, {Wonder: "colossus"}, {Wonder: "hanging-gardens"}}},
		},
		OnlyLeft: []string{"academy", "town-hall"},
		ToMove:   seatPtr(0),
	}
}

// A final reply that wins on points regardless of the card action is not
// a newly offered, card-specific threat (rules/ages-scoring.md).
func TestOutcomeLastReplyUnavoidableCivilianWinQuiet(t *testing.T) {
	s := hnNew(t, lastReplyScenario(0, 200))
	for _, key := range []string{"discard:academy", "discard:town-hall"} {
		x := s
		hnDo(t, &x, key)
		legal := LegalActions(&x, nil)
		if len(legal) < 2 {
			t.Fatal("fixture must include multiple final replies")
		}
		for _, a := range legal {
			y := x
			ApplyTrusted(&y, a)
			if !y.Over() || y.Result.Kind != VictoryCivilian || y.Result.Winner != 1 {
				t.Fatalf("%s: %s does not give the same points win: %+v", key, a.Key(), y.Result)
			}
		}
		if got := hoThreats(hoOutcome(t, &s, key).Gives); len(got) != 0 {
			t.Fatalf("%s names an irrelevant card for an unavoidable points win: %v", key, got)
		}
	}
}

// Keep useful card-specific advice: building Town Hall wins, discarding
// it loses, so this final reply really does depend on which action is taken.
func TestOutcomeLastReplyContestedCivilianWin(t *testing.T) {
	s := hnNew(t, lastReplyScenario(88, 20))
	x := s
	hnDo(t, &x, "discard:academy")
	winners := [2]bool{}
	for i, key := range []string{"construct:town-hall", "discard:town-hall"} {
		y := x
		hnDo(t, &y, key)
		winners[i] = y.Result.Kind == VictoryCivilian && y.Result.Winner == 1
	}
	if winners != [2]bool{true, false} {
		t.Fatalf("fixture must have different final outcomes: %v", winners)
	}
	got := hoThreats(hoOutcome(t, &s, "discard:academy").Gives)
	if !slices.Contains(got, "1:town-hall:civilian") {
		t.Fatalf("lost a useful points threat: %v", got)
	}
}
