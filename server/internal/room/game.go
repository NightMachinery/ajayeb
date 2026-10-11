package room

import (
	"fmt"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
)

// This file holds the room's thin layer over the engine: building the start
// state of a room file, replaying an action log (for undo, and to rebuild a
// loaded room's history), and turning a client's key into the engine's own
// action by set membership.

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

// histEntry is one applied action, as the room remembers it: the ply it
// was applied at, the seat that took it, and its events, unfiltered. The
// state frames' history is built from these, filtered per viewer.
type histEntry struct {
	ply    uint16
	seat   engine.Seat // the seat to move, or for a resign the seat that resigned
	events []engine.Event
}

// actor is the seat that takes action a in s: the seat to move, or the
// seat a resign names.
func actor(s *engine.State, a engine.Action) engine.Seat {
	if a.Type == engine.ActResign {
		return a.Starter
	}
	return s.ToMove
}

// replayKeys replays keys from f's start state and returns the state
// reached and the history of every key: its ply, the seat that took it and
// its events.
func replayKeys(f *store.File, keys []string) (engine.State, []histEntry, error) {
	s, err := startState(f)
	if err != nil {
		return engine.State{}, nil, err
	}
	hist := make([]histEntry, 0, len(keys))
	for i, key := range keys {
		a, ok := actionByKey(&s, key)
		if !ok {
			return engine.State{}, nil, fmt.Errorf("room %s: action %d (%q) is not legal on replay", f.Code, i, key)
		}
		e := histEntry{ply: s.Ply, seat: actor(&s, a)}
		if err := engine.Apply(&s, a, &e.events); err != nil {
			return engine.State{}, nil, fmt.Errorf("room %s: action %d (%s): %w", f.Code, i, key, err)
		}
		hist = append(hist, e)
	}
	return s, hist, nil
}

// actionByKey returns the engine's own action for a key of the room's log:
// a legal action (legalByKey), or a forfeit (resignByKey).
func actionByKey(s *engine.State, key string) (engine.Action, bool) {
	if a, ok := resignByKey(s, key); ok {
		return a, true
	}
	return legalByKey(s, key)
}

// resignByKey returns the forfeit action of a resign:<seat> key while the
// game is not over. A resign is never among the legal actions (bots never
// pick one), so the room checks who may send it itself (see Room.resign).
func resignByKey(s *engine.State, key string) (engine.Action, bool) {
	if s.Over() {
		return engine.Action{}, false
	}
	for seat := range engine.Seat(2) {
		if a := engine.ResignAction(seat); a.Key() == key {
			return a, engine.IsLegal(s, a) == nil
		}
	}
	return engine.Action{}, false
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
