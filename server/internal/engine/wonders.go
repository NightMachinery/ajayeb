package engine

import "github.com/NightMachinery/ajayeb/server/internal/engine/catalog"

// Hook stub for the effects of wonders and the seven-wonders rule. Slice S5b
// owns this file and replaces the stub. Until then a built wonder has no
// effect beyond being built. The contract is in hooks.go.

// resolveWonderEffects applies a newly built wonder's effects. Stub (S5b).
func resolveWonderEffects(s *State, seat Seat, wonder catalog.WonderID, ev *[]Event) outcome {
	return outcomeFinish
}
