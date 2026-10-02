package engine

import "github.com/NightMachinery/ajayeb/server/internal/engine/catalog"

// Hook stubs for the effects of buildings and tokens, the military track,
// science and scoring. Slice S5a owns this file and replaces the stubs (it may
// move them into military.go, science.go and score.go). Until then buildings
// and tokens have no effect beyond entering the city, nothing ever ends the
// game early, and every finished game is a shared civilian victory on zero
// points. The contracts are in hooks.go.

// resolveBuildingEffects applies a new building's effects. Stub (S5a).
func resolveBuildingEffects(s *State, seat Seat, card catalog.CardID, via buildVia, ev *[]Event) outcome {
	return outcomeFinish
}

// resolveTokenEffects applies a new token's immediate effects. Stub (S5a).
func resolveTokenEffects(s *State, seat Seat, token catalog.TokenID, ev *[]Event) outcome {
	return outcomeFinish
}

// movePawn moves the conflict pawn toward seat's opponent. Stub (S5a): the
// pawn does not move.
func movePawn(s *State, seat Seat, shields uint8, ev *[]Event) outcome {
	return outcomeFinish
}

// checkSupremacy ends the game on a military or science supremacy. Stub
// (S5a): never.
func checkSupremacy(s *State, ev *[]Event) bool {
	return false
}

// civilianResult scores the game after Age III. Stub (S5a): a shared victory
// with all scores zero.
func civilianResult(s *State) Result {
	return Result{Kind: VictoryCivilian, Winner: SharedVictory, Scores: Scores(s)}
}

// Scores returns both players' scores as if the game ended now. Stub (S5a):
// all zero.
func Scores(s *State) [2]Score {
	return [2]Score{}
}
