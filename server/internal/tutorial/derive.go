package tutorial

import (
	"slices"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// This file derives the tutorial object of a state frame from the script
// and the action log. With n the script's length and p the log's:
//
//   - prefix: the log's first min(p, n) keys are the script's;
//   - finished: prefix and p >= n; left: not prefix;
//   - the status is complete once Age II (or a later age) is in play or the
//     game is over, guided while prefix and p < n, free otherwise;
//   - while guided, turn counts the player's entries up to the next one
//     (at most turns, the player's entries in all), with entry p's lessons
//     and action when it is the player's and nothing when it is the bot's;
//     once finished, the outro at p == n with the player to move; once
//     left, the leftPath lesson until the player moves again.
//
// It costs O(n) comparisons, so the room derives it for every frame.

// matched returns how many of the log's first keys match the script's, up
// to min(len(keys), len(sc.Entries)).
func matched(sc *Script, keys []string) int {
	m := min(len(keys), len(sc.Entries))
	for i := range m {
		if keys[i] != sc.Entries[i].Key {
			return i
		}
	}
	return m
}

// onScript reports whether keys are a prefix of the script, or the whole
// script followed by more keys.
func onScript(sc *Script, keys []string) bool {
	return matched(sc, keys) == min(len(keys), len(sc.Entries))
}

// Derive returns the tutorial object for the script's player, for the log
// keys (seats[i] is the seat that took keys[i]) that led to s.
func Derive(sc *Script, keys []string, seats []engine.Seat, s *engine.State) wire.TutorialState {
	n, p := len(sc.Entries), len(keys)
	m := matched(sc, keys)
	prefix := m == min(p, n)
	out := wire.TutorialState{
		Script:  sc.Name,
		Status:  wire.TutorialStatusFree,
		Left:    !prefix,
		Lessons: []wire.TutorialLesson{},
	}
	switch {
	case s.Over() || (s.Age >= 2 && s.Phase == engine.PhasePlay):
		out.Status = wire.TutorialStatusComplete
	case prefix && p < n:
		out.Status = wire.TutorialStatusGuided
		out.Turns = sc.Turns()
		// The guided turn ahead; on the bot's reply to the last one, the
		// last (never "turn 11 of 10").
		out.Turn = min(sc.humanBefore(p)+1, out.Turns)
		if e := &sc.Entries[p]; e.Seat == sc.Human {
			out.Lessons = slices.Clone(e.Lessons)
			if e.Action != nil {
				a := *e.Action
				out.Action = &a
			}
		}
	case prefix:
		// Finished: the outro, once, at the first position after the
		// script with the player to move.
		if p == n && s.ToMove == sc.Human {
			out.Lessons = slices.Clone(sc.Outro)
		}
	default:
		// Left the path at action m: the leftPath lesson until the player
		// moves again.
		if !humanMovedAfter(sc, seats, m, p) {
			out.Lessons = []wire.TutorialLesson{{ID: wire.TutorialLessonLeftPath}}
		}
	}
	return out
}

// humanMovedAfter reports whether the player took one of the log's actions
// after index k (up to p). A seat the caller did not give counts as the
// player's, so a short seats slice never shows leftPath by mistake.
func humanMovedAfter(sc *Script, seats []engine.Seat, k, p int) bool {
	for j := k + 1; j < p; j++ {
		if j >= len(seats) || seats[j] == sc.Human {
			return true
		}
	}
	return false
}

// BotKey returns the script's next key when the log is on script and that
// key is the bot's: the room plays it instead of searching.
func BotKey(sc *Script, keys []string) (string, bool) {
	p := len(keys)
	if p >= len(sc.Entries) || !onScript(sc, keys) || sc.Entries[p].Seat == sc.Human {
		return "", false
	}
	return sc.Entries[p].Key, true
}

// GuidedMove reports whether the log's action i was a move of the script:
// i is inside the script and the log matched the script up to and including
// it. The room shows no move review for such a move, since the tutorial
// asked for it.
func GuidedMove(sc *Script, keys []string, i int) bool {
	if i < 0 || i >= len(keys) || i >= len(sc.Entries) {
		return false
	}
	return matched(sc, keys[:i+1]) == i+1
}
