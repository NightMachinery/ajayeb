package engine

import "github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"

// RNG preserves Duel's type identity and method set through the shared seeded
// generator. Copying it still forks the stream by value.
type RNG = rng.RNG

// NewRNG preserves Duel's one-word seeding exactly.
func NewRNG(seed uint64) RNG { return rng.NewRNG(seed) }

// NewRNG2 preserves Duel's two-word seeding exactly.
func NewRNG2(seed1, seed2 uint64) RNG { return rng.NewRNG2(seed1, seed2) }

// ShuffleSlice preserves Duel's Fisher-Yates draw order.
func ShuffleSlice[T any](r *RNG, s []T) { rng.ShuffleSlice(r, s) }
