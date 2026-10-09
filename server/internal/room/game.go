package room

import (
	"fmt"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
)

// This file holds the room's thin layer over the engine: building the start
// state of a room file, replaying an action log (for undo), and turning a
// client's key into the engine's own action by set membership.

// startState returns the state f's game starts from: engine.NewGame, or
// the named scenario's start for a room made from one. It mirrors the
// replay of package store, which is the authority on what a file means.
func startState(f *store.File) (engine.State, error) {
	if f.Scenario == "" {
		return engine.NewGame(f.Seed, f.Options), nil
	}
	sc := engine.NamedScenario(f.Scenario)
	if sc == nil {
		return engine.State{}, fmt.Errorf("room %s: unknown scenario %q", f.Code, f.Scenario)
	}
	s, err := engine.NewGameFromScenario(f.Seed, f.Options, sc)
	if err != nil {
		return engine.State{}, fmt.Errorf("room %s: scenario %q: %w", f.Code, f.Scenario, err)
	}
	return s, nil
}

// replayKeys replays keys from f's start state and returns the state
// reached and, for each key, the seat that decided it (the seat to move
// just before it was applied).
func replayKeys(f *store.File, keys []string) (engine.State, []engine.Seat, error) {
	s, err := startState(f)
	if err != nil {
		return engine.State{}, nil, err
	}
	deciders := make([]engine.Seat, 0, len(keys))
	for i, key := range keys {
		a, ok := legalByKey(&s, key)
		if !ok {
			return engine.State{}, nil, fmt.Errorf("room %s: action %d (%q) is not legal on replay", f.Code, i, key)
		}
		deciders = append(deciders, s.ToMove)
		if err := engine.Apply(&s, a, nil); err != nil {
			return engine.State{}, nil, fmt.Errorf("room %s: action %d (%s): %w", f.Code, i, key, err)
		}
	}
	return s, deciders, nil
}

// legalByKey returns the engine's own legal action whose key is key: the
// room never builds an action from client data, it looks the key up in the
// legal set.
func legalByKey(s *engine.State, key string) (engine.Action, bool) {
	if s.Over() {
		return engine.Action{}, false
	}
	for _, a := range engine.LegalActions(s, nil) {
		if a.Key() == key {
			return a, true
		}
	}
	return engine.Action{}, false
}
