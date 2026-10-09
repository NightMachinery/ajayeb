package store

import (
	"errors"
	"fmt"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// ErrRulesVersion is wrapped by the error Load returns when a file's Rules
// does not match the engine's current engine.RulesVersion.
var ErrRulesVersion = errors.New("store: rules version mismatch")

// ErrHashMismatch is wrapped by the error Load returns when a file replays
// cleanly but reaches a state whose hash does not match the file's Hash.
var ErrHashMismatch = errors.New("store: replay hash mismatch")

// ErrSchema is wrapped by the error Load and PeekHeader return for a file
// whose Schema is not this build's (a newer server's file, or one with no
// schema at all). Such a file is left untouched.
var ErrSchema = errors.New("store: unknown room file schema")

// replay rebuilds the state f's Seed, Options, Scenario and Actions
// describe, by replaying every action through the engine in order. It does
// not look at f.Hash; callers that must trust the result check that
// themselves (see Load).
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
		a, err := engine.ParseKey(key)
		if err != nil {
			return s, fmt.Errorf("store: room %s: action %d (%q): %w", f.Code, i, key, err)
		}
		if err := engine.Apply(&s, a, nil); err != nil {
			return s, fmt.Errorf("store: room %s: action %d (%s): %w", f.Code, i, key, err)
		}
	}
	return s, nil
}
