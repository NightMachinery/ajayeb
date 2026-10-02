// Package enginetest holds helpers for tests of the rules engine and of the
// code built on it: a fluent builder for scenario positions (New), scripts
// that apply actions by key or by a short human-friendly step (Do,
// MustFail), and playout policies that play whole games (RandomPolicy,
// SimplePolicy, Playout).
//
// The helpers report failures through testing.TB and stop the test, so a
// test reads as a list of steps. They follow the engine's purity rules: no
// I/O, no clocks, no maps; randomness comes from a seeded engine.RNG.
package enginetest

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// Builder builds a scenario position step by step:
//
//	s := enginetest.New(t, 7).
//		Wonders(0, "mausoleum", "pyramids", "sphinx", "colossus").
//		Age(2).City(0, "lumber-yard", "glassworks").Coins(0, 9).
//		Tokens(0, "masonry").Discard("baths").Pawn(-3).
//		Accessible("temple").ToMove(0).State()
//
// Every method returns the builder. Slugs are checked when State (or
// Scenario) is called.
type Builder struct {
	tb      testing.TB
	seed    uint64
	opts    engine.Options
	sc      engine.Scenario
	wonders [2][]string
	setW    [2]bool
}

// New starts a builder for a game with this seed and the default options.
func New(tb testing.TB, seed uint64) *Builder {
	tb.Helper()
	return &Builder{tb: tb, seed: seed}
}

// Options sets the game options.
func (b *Builder) Options(o engine.Options) *Builder { b.opts = o; return b }

// Name names the scenario (it shows in error messages and logs).
func (b *Builder) Name(name string) *Builder { b.sc.Name = name; return b }

// FirstPlayer fixes the first player.
func (b *Builder) FirstPlayer(seat engine.Seat) *Builder {
	b.sc.FirstPlayer = &seat
	return b
}

// Wonders gives seat these wonders (up to four, in slot order) and skips the
// draft. Missing wonders, of this seat or of a seat never given any, are
// filled with the lowest-numbered wonders nobody has.
func (b *Builder) Wonders(seat engine.Seat, slugs ...string) *Builder {
	b.wonders[seat] = append([]string(nil), slugs...)
	b.setW[seat] = true
	return b
}

// Age lays out this age's structure (1 to 3).
func (b *Builder) Age(n int) *Builder { b.sc.Age = n; return b }

// City adds buildings to seat's city.
func (b *Builder) City(seat engine.Seat, slugs ...string) *Builder {
	b.sc.Players[seat].City = append(b.sc.Players[seat].City, slugs...)
	return b
}

// Coins sets seat's coins.
func (b *Builder) Coins(seat engine.Seat, n int) *Builder {
	b.sc.Players[seat].Coins = &n
	return b
}

// Tokens gives seat these progress tokens.
func (b *Builder) Tokens(seat engine.Seat, slugs ...string) *Builder {
	b.sc.Players[seat].Tokens = append(b.sc.Players[seat].Tokens, slugs...)
	return b
}

// Built builds seat's wonder with the named card tucked under it ("" picks
// any card out of play).
func (b *Builder) Built(seat engine.Seat, wonder, tucked string) *Builder {
	b.sc.Players[seat].Built = append(b.sc.Players[seat].Built, engine.ScenarioWonder{Wonder: wonder, Tucked: tucked})
	return b
}

// Discard puts cards on the discard pile.
func (b *Builder) Discard(slugs ...string) *Builder {
	b.sc.Discard = append(b.sc.Discard, slugs...)
	return b
}

// BoardTokens sets exactly which tokens are on the board.
func (b *Builder) BoardTokens(slugs ...string) *Builder {
	b.sc.BoardTokens = append([]string(nil), slugs...)
	return b
}

// BoxOrder puts these tokens at the front of the box, in order.
func (b *Builder) BoxOrder(slugs ...string) *Builder {
	b.sc.BoxOrder = append([]string(nil), slugs...)
	return b
}

// Accessible moves these cards into accessible, face-up slots.
func (b *Builder) Accessible(slugs ...string) *Builder {
	b.sc.Accessible = append(b.sc.Accessible, slugs...)
	return b
}

// OnlyLeft empties the structure except for these cards.
func (b *Builder) OnlyLeft(slugs ...string) *Builder {
	b.sc.OnlyLeft = append([]string(nil), slugs...)
	return b
}

// Pawn places the conflict pawn (-8 to 8).
func (b *Builder) Pawn(p int) *Builder { b.sc.Pawn = &p; return b }

// ToMove makes seat the active seat, to move.
func (b *Builder) ToMove(seat engine.Seat) *Builder { b.sc.ToMove = &seat; return b }

// Scenario returns the scenario built so far (a copy the builder no longer
// changes).
func (b *Builder) Scenario() *engine.Scenario {
	b.tb.Helper()
	sc := b.sc
	if b.setW[0] || b.setW[1] {
		ws, err := fillWonders(b.wonders)
		if err != nil {
			b.tb.Fatal(err)
		}
		sc.Wonders = &ws
	}
	return &sc
}

// State creates the position. It fails the test if the scenario is invalid.
func (b *Builder) State() engine.State {
	b.tb.Helper()
	s, err := engine.NewGameFromScenario(b.seed, b.opts, b.Scenario())
	if err != nil {
		b.tb.Fatalf("enginetest: %v", err)
	}
	return s
}

// Log returns a fresh log for the position, so a test can record and replay
// what it plays from it.
func (b *Builder) Log() *engine.Log {
	b.tb.Helper()
	return engine.NewLog(b.seed, b.opts, b.Scenario())
}

// fillWonders completes the given wonder lists to four each with the
// lowest-numbered wonders not given to anyone.
func fillWonders(given [2][]string) ([2][engine.WondersPerPlayer]string, error) {
	var out [2][engine.WondersPerPlayer]string
	var used [catalog.NumWonders + 1]bool
	for seat := range given {
		if len(given[seat]) > engine.WondersPerPlayer {
			return out, errorf("seat %d is given %d wonders", seat, len(given[seat]))
		}
		for _, slug := range given[seat] {
			w, ok := catalog.WonderBySlug(slug)
			if !ok {
				return out, errorf("unknown wonder %q", slug)
			}
			used[w] = true
		}
	}
	next := catalog.WonderID(1)
	for seat := range given {
		copy(out[seat][:], given[seat])
		for k := len(given[seat]); k < engine.WondersPerPlayer; k++ {
			for int(next) <= catalog.NumWonders && used[next] {
				next++
			}
			if int(next) > catalog.NumWonders {
				return out, errorf("not enough wonders to fill seat %d", seat)
			}
			used[next] = true
			out[seat][k] = catalog.Wonder(next).Slug
		}
	}
	return out, nil
}
