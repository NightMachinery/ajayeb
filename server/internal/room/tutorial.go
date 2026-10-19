package room

import (
	"context"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/tutorial"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// This file is the room's side of a tutorial room (package tutorial): an ai
// room whose file names a tutorial script. Everything here is derived from
// the file's action log, never stored: the player's frames carry the
// tutorial object, the bot plays the script's replies while the log is on
// it, and a move the script asked for gets no move review. A room whose
// script this build does not know plays on as a normal game.

// script returns the room's tutorial script, or nil for any other room.
func (r *Room) script() *tutorial.Script {
	if r.file.Tutorial == "" || r.file.Scenario != "" {
		return nil
	}
	sc, ok := tutorial.ByName(r.file.Tutorial)
	if !ok {
		return nil
	}
	return sc
}

// tutorialState is the tutorial object of mem's frame: set only in a
// tutorial room, for a connection that holds the script's player seat (the
// bot and spectators get none).
func (r *Room) tutorialState(mem member) *wire.TutorialState {
	sc := r.script()
	if sc == nil || mem.role != rolePlayer || mem.seat != sc.Human {
		return nil
	}
	seats := make([]engine.Seat, len(r.hist))
	for i, e := range r.hist {
		seats[i] = e.seat
	}
	ts := tutorial.Derive(sc, r.file.Actions, seats, &r.state)
	return &ts
}

// guidedLastMove reports whether any decision in the reviewed turn was
// requested by the tutorial script.
func (r *Room) guidedLastMove(seat engine.Seat) bool {
	sc := r.script()
	if sc == nil {
		return false
	}
	if !r.reviews.loaded {
		r.loadReviews()
	}
	sl := &r.reviews.seats[seat]
	if !sl.set {
		return false
	}
	for _, step := range sl.steps {
		for i := len(r.hist) - 1; i >= 0; i-- {
			if r.hist[i].ply == step.pre.Ply && tutorial.GuidedMove(sc, r.file.Actions, i) {
				return true
			}
		}
	}
	return false
}

// scriptedBot returns a bot that plays the script's next key, when the room
// is a tutorial room on its script and that key is the bot's.
func (r *Room) scriptedBot() (Bot, bool) {
	sc := r.script()
	if sc == nil {
		return nil, false
	}
	key, ok := tutorial.BotKey(sc, r.file.Actions)
	if !ok {
		return nil, false
	}
	return scriptBot{key: key}, true
}

// scriptBot is a Bot that answers with one fixed action key, parsed by the
// engine. The room still applies it through the legal set (botResult), so
// a scripted key that is not legal would be logged and replaced by the
// first legal action, never applied; script_test.go makes sure none is.
type scriptBot struct{ key string }

func (b scriptBot) Choose(context.Context, engine.Observation, ai.Budget, func(ai.Progress)) (engine.Action, error) {
	return engine.ParseKey(b.key)
}
