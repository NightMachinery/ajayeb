// Package engine implements the rules of the game: setup, the turn machine,
// legal actions, applying actions, payment, scoring, replay and hashing, and
// the per-seat view that the server sends to clients.
//
// The engine is the only implementation of the rules in the project. The web
// client runs no rules code. The server applies an action only when its
// canonical key (for example "construct:baths" or "wonder:the-pyramids:baths")
// appears in the engine's legal actions for the seat that is to move, and it
// stores the keys, not client-built objects, in the room log.
//
// Subpackages: catalog holds the typed card, wonder and token data and the
// effect vocabulary; enginetest holds test builders, scripts and playout
// policies.
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
//     holds no RNG. Only rng.go may import math/rand/v2; it wraps PCG with its
//     own IntN and Shuffle so replays do not depend on library helper
//     algorithms.
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
