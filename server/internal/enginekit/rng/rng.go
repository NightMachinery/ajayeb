// Package rng supplies the shared deterministic generator and byte digest.
package rng

import (
	"math/bits"
	"math/rand/v2"
)

// RNG is the shared random number generator: PCG (from math/rand/v2,
// stored by value) with our own bounded-integer and shuffle algorithms, so
// that a seed gives the same games on every platform and Go version even if
// the standard library's helpers change. The engine uses it only in NewGame;
// bots use it for determinization and rollouts.
//
// RNG is a small value: copying it forks the stream (both copies produce the
// same numbers). Only this package's rng.go may import math/rand/v2.
// Legacy engine-prefixed panic text is retained for Duel compatibility.
type RNG struct {
	pcg rand.PCG
}

// NewRNG returns a generator seeded from one 64-bit seed. The second PCG word
// is a fixed function of the seed.
func NewRNG(seed uint64) RNG {
	return NewRNG2(seed, seed^0x9e3779b97f4a7c15)
}

// NewRNG2 returns a generator seeded from two 64-bit words.
func NewRNG2(seed1, seed2 uint64) RNG {
	var r RNG
	r.pcg.Seed(seed1, seed2)
	return r
}

// Uint64 returns the next 64 random bits.
func (r *RNG) Uint64() uint64 { return r.pcg.Uint64() }

// Uint64N returns a uniform integer in [0, n), by Lemire's nearly
// divisionless method: multiply a random 64-bit word by n and keep the high
// word, rejecting the few low words that would bias the result. It panics if
// n is 0.
func (r *RNG) Uint64N(n uint64) uint64 {
	if n == 0 {
		panic("engine: RNG.Uint64N(0)")
	}
	hi, lo := bits.Mul64(r.pcg.Uint64(), n)
	if lo < n {
		threshold := -n % n // 2^64 mod n
		for lo < threshold {
			hi, lo = bits.Mul64(r.pcg.Uint64(), n)
		}
	}
	return hi
}

// IntN returns a uniform integer in [0, n). It panics if n <= 0.
func (r *RNG) IntN(n int) int {
	if n <= 0 {
		panic("engine: RNG.IntN with n <= 0")
	}
	return int(r.Uint64N(uint64(n)))
}

// Shuffle permutes n elements by Fisher-Yates, from the last element down,
// calling swap(i, j) to exchange elements i and j. It panics if n < 0.
func (r *RNG) Shuffle(n int, swap func(i, j int)) {
	if n < 0 {
		panic("engine: RNG.Shuffle with n < 0")
	}
	for i := n - 1; i > 0; i-- {
		swap(i, r.IntN(i+1))
	}
}

// ShuffleSlice permutes s in place, exactly as Shuffle(len(s), ...) would.
func ShuffleSlice[T any](r *RNG, s []T) {
	r.Shuffle(len(s), func(i, j int) { s[i], s[j] = s[j], s[i] })
}

// MarshalBinary encodes the generator's state.
func (r *RNG) MarshalBinary() ([]byte, error) { return r.pcg.MarshalBinary() }

// UnmarshalBinary restores a state written by MarshalBinary.
func (r *RNG) UnmarshalBinary(b []byte) error { return r.pcg.UnmarshalBinary(b) }
