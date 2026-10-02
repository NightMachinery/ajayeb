package engine

// Hook stub for the pending choices (destroy, build from the discard pile,
// board token, Great Library token). Slice S5b owns this file and replaces
// the stub. Until then no hook opens a pending choice, so the stub is never
// reached in a real game. The contract is in hooks.go.

// resolvePending resolves the open pending choice with a legal action. Stub
// (S5b): it resolves nothing and lets the turn finish.
func resolvePending(s *State, a Action, ev *[]Event) outcome {
	return outcomeFinish
}
