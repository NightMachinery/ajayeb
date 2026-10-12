package engine

import (
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"strconv"
)

// This file holds the game log and replay. A game is fully described by its
// seed, its options, an optional scenario and its action keys, so the log
// stores exactly that, and Replay rebuilds every state from it. Hash
// fingerprints a state, so a replay can be checked against the hash recorded
// when the game was played.
//
// The log format, version ajayeb-log/1, as JSON:
//
//	{"format":"ajayeb-log/1","rules":1,"seed":"1234567890123",
//	 "options":{"firstPlayer":"random","centerPawn":"lastActiveChooses","hideAtEnd":false},
//	 "scenario":null,
//	 "actions":[{"ply":0,"seat":0,"action":"draft:pyramids"}, ...],
//	 "finalHash":"9f2c0e5a17b3d4c8"}
//
// The seed is a string so that JavaScript readers keep all 64 bits. Each
// entry records the ply it was applied at and the seat that was to move, and
// replay checks both. finalHash (16 hex digits of Hash) is optional.

// LogFormat is the format tag of the logs this version writes and reads.
const LogFormat = "ajayeb-log/1"

// RulesVersion numbers the rules the engine plays. It changes whenever a
// change to the rules (or to setup) could make an old log replay
// differently; logs record it and Replay rejects a log of another version.
const RulesVersion = 1

// Log is a recorded game.
type Log struct {
	Format    string     `json:"format"`
	Rules     int        `json:"rules"`
	Seed      uint64     `json:"seed,string"`
	Options   Options    `json:"options"`
	Scenario  *Scenario  `json:"scenario"` // nil for a real game
	Actions   []LogEntry `json:"actions"`
	FinalHash string     `json:"finalHash,omitempty"`
}

// LogEntry is one applied action.
type LogEntry struct {
	Ply    uint16 `json:"ply"`  // State.Ply before the action
	Seat   Seat   `json:"seat"` // the seat that was to move
	Action Action `json:"action"`
}

// NewLog starts the log of a game created with NewGame(seed, opts), or with
// NewGameFromScenario(seed, opts, sc) when sc is not nil.
func NewLog(seed uint64, opts Options, sc *Scenario) *Log {
	return &Log{Format: LogFormat, Rules: RulesVersion, Seed: seed, Options: opts, Scenario: sc, Actions: []LogEntry{}}
}

// Record appends action a, about to be applied to s, with s's ply and seat
// to move. Call it before Apply, and only for an action Apply accepts.
func (l *Log) Record(s *State, a Action) {
	l.Actions = append(l.Actions, LogEntry{Ply: s.Ply, Seat: s.ToMove, Action: a})
}

// Seal records the hash of the final state s.
func (l *Log) Seal(s *State) { l.FinalHash = HashString(s) }

// Start returns the initial state the log describes, after checking its
// format and rules version.
func (l *Log) Start() (State, error) {
	if l.Format != LogFormat {
		return State{}, fmt.Errorf("engine: log format %s, want %q", truncateString(l.Format), LogFormat)
	}
	if l.Rules != RulesVersion {
		return State{}, fmt.Errorf("engine: log of rules version %d, this engine plays version %d", l.Rules, RulesVersion)
	}
	if l.Scenario != nil {
		return NewGameFromScenario(l.Seed, l.Options, l.Scenario)
	}
	return NewGame(l.Seed, l.Options), nil
}

// Replay rebuilds the final state of a log. See ReplaySteps.
func Replay(l *Log) (State, error) { return ReplaySteps(l, nil) }

// ReplaySteps rebuilds the states of a log one action at a time. Before each
// action it checks that the entry's ply and seat match the state (the seat
// must be the one to move), then applies the action with Apply, which
// rejects an illegal one. After each action it calls step, if not nil, with
// the entry's index, the new state and the action's events; an error from
// step stops the replay. At the end it compares FinalHash, when the log has
// one. It returns the last state reached, also on error.
func ReplaySteps(l *Log, step func(i int, s *State, ev []Event) error) (State, error) {
	s, err := l.Start()
	if err != nil {
		return s, err
	}
	var ev []Event
	for i, e := range l.Actions {
		if e.Ply != s.Ply {
			return s, fmt.Errorf("engine: log entry %d is for ply %d, but the game is at ply %d", i, e.Ply, s.Ply)
		}
		if e.Seat != s.ToMove {
			return s, fmt.Errorf("engine: log entry %d (%s) is seat %d's, but seat %d is to move", i, e.Action, e.Seat, s.ToMove)
		}
		ev = ev[:0]
		if err := Apply(&s, e.Action, &ev); err != nil {
			return s, fmt.Errorf("engine: log entry %d: %w", i, err)
		}
		if step != nil {
			if err := step(i, &s, ev); err != nil {
				return s, err
			}
		}
	}
	if l.FinalHash != "" {
		if h := HashString(&s); h != l.FinalHash {
			return s, fmt.Errorf("%w: the replay ends with hash %s, the log says %s", ErrHashMismatch, h, truncateString(l.FinalHash))
		}
	}
	return s, nil
}

// ErrHashMismatch is wrapped by the Replay error for a log whose final state
// does not match its recorded hash.
var ErrHashMismatch = errors.New("engine: replay hash mismatch")

// Hash fingerprints a state: FNV-1a (64-bit) over its fixed-size binary
// encoding (encoding/binary, little-endian), up to the derived counters at
// the end of State (Shields, PlayingAgain). Equal states have equal hashes,
// and any change to any other field, hidden ones included, changes the hash
// in practice. The counters are left out because they follow from the
// action keys anyway, and so that the hashes recorded before they existed
// (saved rooms, game logs, golden replays) still match.
func Hash(s *State) uint64 {
	buf := make([]byte, 0, stateEncodedSize)
	buf, err := binary.Append(buf, binary.LittleEndian, s)
	if err != nil {
		panic("engine: State is not fixed-size: " + err.Error())
	}
	h := fnv.New64a()
	_, _ = h.Write(buf[:stateHashedSize])
	return h.Sum64()
}

// stateEncodedSize is the length of a state's binary encoding, and
// stateHashedSize the part of it Hash digests: everything before the
// derived counters, which are the last fields of State.
var (
	stateEncodedSize = binary.Size(State{})
	stateHashedSize  = stateEncodedSize - binary.Size(State{}.Shields) - binary.Size(State{}.PlayingAgain)
)

// HashString returns Hash as 16 lower-case hex digits.
func HashString(s *State) string {
	const digits = "0123456789abcdef"
	h := Hash(s)
	var b [16]byte
	for i := 15; i >= 0; i-- {
		b[i] = digits[h&15]
		h >>= 4
	}
	return string(b[:])
}

// truncateString bounds untrusted text quoted in an error message.
func truncateString(s string) string {
	if len(s) > 64 {
		return strconv.Quote(s[:64]) + "..."
	}
	return strconv.Quote(s)
}
