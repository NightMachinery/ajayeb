package engine

// Hook stub for the pending choices (destroy, build from the discard pile,
// board token, Great Library token). Slice S5b owns this file and replaces
// the stub. Until then only the board token choice can open (a science
// pair, slice S5a), and the stub resolves it through takeBoardToken
// (science.go); the wonder choices never open. The contract is in hooks.go.

// resolvePending resolves the open pending choice with a legal action. Stub
// (S5b): only the board token choice of a science pair is resolved.
func resolvePending(s *State, a Action, ev *[]Event) outcome {
	if s.Phase == PhasePickBoardToken {
		return takeBoardToken(s, s.ToMove, a.Token, ev)
	}
	return outcomeFinish
}
