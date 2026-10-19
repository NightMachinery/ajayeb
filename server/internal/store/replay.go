package store

import (
	"errors"
	"fmt"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// ErrRulesVersion is wrapped by the error Load returns when a file's Rules
// does not match the engine's current engine.RulesVersion.
var ErrRulesVersion = errors.New("store: rules version mismatch")

// ErrHashMismatch is used by strict replay downloads. Room restoration
// refreshes a differing hash after a full replay within the same RulesVersion.
var ErrHashMismatch = errors.New("store: replay hash mismatch")

// ErrSchema is wrapped by the error Load and PeekHeader return for a file
// whose Schema is not this build's (a newer server's file, or one with no
// schema at all). Such a file is left untouched.
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

// assess is shared by restoration and read-only deployment checks. It never writes.
func assess(f *File) (engine.State, ArchiveReason, error) {
	if f.Rules != engine.RulesVersion {
		return engine.State{}, ArchiveRulesVersion, fmt.Errorf("%w: file %d, engine %d", ErrRulesVersion, f.Rules, engine.RulesVersion)
	}
	s, err := replay(f)
	if err != nil {
		return engine.State{}, ArchiveReplayFailed, err
	}
	return s, "", nil
}
