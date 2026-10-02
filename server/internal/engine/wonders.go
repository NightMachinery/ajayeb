package engine

import "github.com/NightMachinery/ajayeb/server/internal/engine/catalog"

// Hook stub for the effects of wonders and the seven-wonder rule. Slice S5b
// owns this file and replaces the stub. Until then a built wonder has no
// effect beyond being built, except that the seven-wonder rule is applied
// (through city.go's sevenWonderRule), so that the turn machine's
// invariants hold. The contract is in hooks.go.

// resolveWonderEffects applies a newly built wonder's effects. Stub (S5b):
// only the seven-wonder rule.
func resolveWonderEffects(s *State, seat Seat, wonder catalog.WonderID, ev *[]Event) outcome {
	sevenWonderRule(s, ev)
	return outcomeFinish
}
