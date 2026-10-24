package store

import (
	"errors"
	"fmt"
	"slices"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// ErrRulesVersion is wrapped by the error Load returns when a file's Rules
// does not match the engine's current engine.RulesVersion.
var ErrRulesVersion = errors.New("store: rules version mismatch")

// ErrHashMismatch is used by strict replay downloads. Room restoration
// refreshes a differing hash after a full replay within the same RulesVersion.
var ErrHashMismatch = errors.New("store: replay hash mismatch")

// ErrSchema is wrapped by the error Load and PeekHeader return for a file
// whose schema/codec is unsupported (including a missing schema). ErrGame also
// matches it. Such files are left untouched and counted as preflight errors.
var ErrSchema = errors.New("store: unknown room file schema")

// replay rebuilds the state f's Seed, Options, Scenario and Actions
// describe, by replaying every action through the engine in order. It does
// not look at f.Hash: Load treats a clean same-version replay as canonical,
// while replay downloads validate their stored hash separately.
func replay(f *File) (engine.State, error) {
	var sc *engine.Scenario
	if f.Scenario != "" {
		sc = engine.NamedScenario(f.Scenario)
		if sc == nil {
			return engine.State{}, fmt.Errorf("store: room %s: unknown scenario %q", f.Code, f.Scenario)
		}
	}

	var s engine.State
	if sc != nil {
		built, err := engine.NewGameFromScenario(f.Seed, f.Options, sc)
		if err != nil {
			return engine.State{}, fmt.Errorf("store: room %s: starting scenario %q: %w", f.Code, f.Scenario, err)
		}
		s = built
	} else {
		s = engine.NewGame(f.Seed, f.Options)
	}

	for i, key := range f.Actions {
		a, ok := replayAction(&s, key)
		if !ok {
			return s, fmt.Errorf("store: room %s: action %d (%q) is not legal on replay", f.Code, i, key)
		}
		if err := engine.Apply(&s, a, nil); err != nil {
			return s, fmt.Errorf("store: room %s: action %d (%s): %w", f.Code, i, key, err)
		}
	}
	return s, nil
}

// replayAction matches the room's replay path: ordinary keys come from LegalActions.
// Resignation is deliberately absent from that list (bots must not choose it),
// so accept only the engine's own legal resignation for either seat.
func replayAction(s *engine.State, key string) (engine.Action, bool) {
	if s.Over() {
		return engine.Action{}, false
	}
	for _, a := range engine.LegalActions(s, nil) {
		if a.Key() == key {
			return a, true
		}
	}
	for seat := range engine.Seat(2) {
		if a := engine.ResignAction(seat); a.Key() == key && engine.IsLegal(s, a) == nil {
			return a, true
		}
	}
	return engine.Action{}, false
}

// ArchiveReason is a bounded, durable reason suitable for the client. Details
// (including keys, hashes and engine errors) belong only in server logs.
type ArchiveReason string

const (
	ArchiveRulesVersion ArchiveReason = "rules-version"
	ArchiveReplayFailed ArchiveReason = "replay-failed"
	ArchiveUnavailable  ArchiveReason = "unavailable" // legacy archive, or an unknown future reason
)

// SafeArchiveReason never forwards arbitrary persisted text to clients.
func SafeArchiveReason(reason ArchiveReason) ArchiveReason {
	switch reason {
	case ArchiveRulesVersion, ArchiveReplayFailed:
		return reason
	}
	return ArchiveUnavailable
}

// ErrArchived identifies an unavailable archived room, distinct from a missing file.
var ErrArchived = errors.New("store: room archived")

// ArchivedError retains private diagnostic details and a separately bounded reason.
type ArchivedError struct {
	Reason ArchiveReason
	Cause  error
}

func (e *ArchivedError) Error() string {
	return fmt.Sprintf("%s: %s: %v", ErrArchived, e.Reason, e.Cause)
}
func (e *ArchivedError) Unwrap() error        { return e.Cause }
func (e *ArchivedError) Is(target error) bool { return target == ErrArchived }

// MaxHintAdmins bounds a shared Duel game's frozen HintAdmins: the lobby's
// real admins (package roles allows 32) and its one temporary admin.
const MaxHintAdmins = 33

// ValidHintAdmins reports whether ids can be a File's HintAdmins: at most
// MaxHintAdmins user ids, none empty, longer than 64 bytes or there twice.
// An empty set is a legacy room's.
func ValidHintAdmins(ids []string) bool {
	if len(ids) > MaxHintAdmins {
		return false
	}
	sorted := slices.Sorted(slices.Values(ids))
	for i, id := range sorted {
		if id == "" || len(id) > 64 || i > 0 && sorted[i-1] == id {
			return false
		}
	}
	return true
}

// assess is shared by restoration and read-only deployment checks. It never writes.
func assess(f *File) (engine.State, ArchiveReason, error) {
	if f.Rules != engine.RulesVersion {
		return engine.State{}, ArchiveRulesVersion, fmt.Errorf("%w: file %d, engine %d", ErrRulesVersion, f.Rules, engine.RulesVersion)
	}
	if !ValidHintAdmins(f.HintAdmins) {
		// Not replay data, but no build wrote it, so the file is damaged.
		return engine.State{}, ArchiveReplayFailed, fmt.Errorf("store: room %s: invalid hint admins", f.Code)
	}
	s, err := replay(f)
	if err != nil {
		return engine.State{}, ArchiveReplayFailed, err
	}
	return s, "", nil
}
