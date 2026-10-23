// Package engine implements the rules of the game: setup, the turn machine,
// legal actions, applying actions, payment, scoring, replay and hashing, and
// the per-seat view that the server sends to clients.
//
// The engine is the only implementation of the rules in the project. The web
// client runs no rules code. The server applies an action only when its
// canonical key (for example "construct:baths" or "wonder:pyramids:baths")
// appears in the engine's legal actions for the seat that is to move, and it
// stores the keys, not client-built objects, in the room log.
//
// Subpackages: catalog holds the typed card, wonder and token data and the
// effect vocabulary; enginetest holds test builders, scripts and playout
// policies for tests of this package and of code built on it.
//
// # Where things are
//
//   - state.go: State and its parts, the fixed-size value the whole game
//     lives in; action.go: Action and its canonical keys (Key, ParseKey);
//     events.go: the events Apply reports. These three files and
//     catalog/types.go are frozen: change them only through the coordinator.
//   - setup.go: NewGame, which resolves every chance event of a game (the
//     wonder draft order, the age deals and their removed cards, the guild
//     draw, the token placement and box order, and the first player) from
//     the seed, in a fixed order a golden test pins.
//   - layout.go: the three age structures (slot positions, which slots cover
//     which, face-up parity, AccessibleMask).
//   - structure.go: the current age's structure in State.Slots: dealing an
//     age, which slots are accessible, taking a card, and turning face up
//     the face-down cards a turn has freed.
//   - city.go: the only writer of a player's derived caches (Chains, Prod,
//     Choice, NChoice, TradeFix, ColorN, Sci), so that they always match a
//     recompute from scratch; CheckInvariants verifies that.
//   - cost.go: the cheapest payment for a building or a wonder, its full
//     breakdown (Payment) for the view, its short form for hot paths
//     (Quote), trade prices and the discard value. The player never chooses
//     how to pay; the engine always applies the cheapest payment.
//   - legal.go: LegalActions and IsLegal, which always agree with each
//     other; the server accepts a key only when LegalActions lists it for
//     the seat to move.
//   - apply.go: the turn machine. Apply and ApplyTrusted dispatch an action
//     by phase, the three play actions take a card from the structure, and
//     finishTurn ends the turn (reveals, the supremacy check, the end of an
//     age or of the game, the extra turn, the next player).
//   - hooks.go: the fixed seams between the turn machine (apply.go) and the
//     effect rules (effects.go, wonders.go, pending.go), with their
//     contracts.
//   - invariants.go: CheckInvariants, which checks that a state (or a
//     scenario built from one) is one the rules can reach and returns every
//     violation it finds; tests and fuzzers call it after every step, the
//     engine itself never does.
//   - replay.go: Log, the ajayeb-log/1 format a game is recorded in (seed,
//     options, an optional scenario and its action keys), Replay and
//     ReplaySteps, which rebuild a game's states from a log, and
//     Hash/HashString, the FNV-1a fingerprint of a state that a replay can
//     be checked against.
//   - scenario.go: Scenario and NewGameFromScenario, which build a test or
//     dev-room starting position (wonders, age, cities, coins, tokens, built
//     wonders, the discard pile, which cards are accessible, the pawn) on
//     top of an ordinary NewGame deal, and RunTrigger, which plays a named
//     scenario's trigger script.
//   - rng.go: aliases and wrappers for enginekit/rng, used by NewGame.
//   - errors.go: IllegalError and its Reason codes, ErrBadKey,
//     ErrMalformedAction.
//
// The enginetest package is a separate, test-only package (its own tests
// live in package engine_test) that other packages' tests import for a
// fluent scenario builder, script helpers that apply actions by key or by a
// short human-friendly step, and playout policies that play whole games.
//
// # Purity rules
//
// The engine is a pure, deterministic library. golangci-lint (depguard) and an
// import-guard test enforce the import bans; reviewers enforce the rest.
//
//   - No I/O of any kind: no files, network, environment, clocks, goroutines
//     or logging. Non-test code must not import time, os, net, log (including
//     log/slog), sync, unsafe, syscall, crypto/rand, io/ioutil or math/rand
//     (v1).
//   - All chance is resolved in NewGame from an explicit seed: wonder order,
//     age deals, removed cards, the guild draw, token placement and the box
//     order, and the first player. Apply uses no randomness and the state
//     holds no RNG. No engine file may import either math/rand version;
//     enginekit/rng wraps PCG with the pinned IntN and Shuffle algorithms so
//     replays do not depend on library helper algorithms.
//   - The same seed, options and action keys give the same states, events,
//     views and hashes on every platform and Go version.
//   - Never iterate a map where the order could reach state, events, views,
//     hashes or errors. Rules logic uses arrays indexed by enums
//     ([NumResources]int8 and the like) and bitsets. (Maps inside view DTOs
//     are acceptable because encoding/json sorts their keys.)
//   - Integers only: no floating point in rules code.
//   - The state is one fixed-size value type made of arrays, bitsets and
//     small fixed-width integers, with no pointers, slices, maps or strings,
//     so Clone is a plain copy and states are comparable and hashable.
//   - Views are built by whitelist, so a forgotten field cannot leak hidden
//     information. Slices in views start empty, never nil, so JSON never
//     carries null arrays.
package engine
