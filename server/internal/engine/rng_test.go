package engine

import (
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"
)

// TestRNGGolden pins the output of fixed seeds. Setup shuffles with this
// generator, so a change here would change every seeded game, golden replay
// and stored room. If it fails after a Go upgrade, the PCG stream or our own
// algorithms changed: do not update the values, fix the generator.
func TestRNGGolden(t *testing.T) {
	r := NewRNG(1)
	wantU := []uint64{0x8707a01d6329783f, 0x07df1bd4a477b564, 0xaef8fc52ab960cb2, 0x26b0f2f5c337080e}
	for i, want := range wantU {
		if got := r.Uint64(); got != want {
			t.Errorf("NewRNG(1) draw %d = %#x, want %#x", i, got, want)
		}
	}
	r = NewRNG(42)
	wantN := []int{4, 9, 7, 3, 8, 9, 2, 3, 3, 8, 6, 7, 6, 5, 8, 5, 4, 7, 5, 8}
	for i, want := range wantN {
		if got := r.IntN(10); got != want {
			t.Errorf("NewRNG(42) IntN(10) draw %d = %d, want %d", i, got, want)
		}
	}
	r = NewRNG(7)
	s := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	ShuffleSlice(&r, s)
	if want := []int{0, 7, 3, 1, 9, 6, 4, 8, 2, 5}; !slices.Equal(s, want) {
		t.Errorf("NewRNG(7) shuffle = %v, want %v", s, want)
	}
	r = NewRNG2(1, 2)
	wantBig := []uint64{0x627ad2c32b777a88, 0x4ee761d683bef636, 0x646823029897c044, 0x65f6e06e5b1d60cd}
	for i, want := range wantBig {
		if got := r.Uint64N(1<<63 + 1); got != want {
			t.Errorf("NewRNG2(1, 2) Uint64N(2^63+1) draw %d = %#x, want %#x", i, got, want)
		}
	}
}

// Comparing Duel and shared values without conversion requires matching types;
// golden values above additionally pin the constructors and generic wrapper.
func TestSharedRNGCompatibility(t *testing.T) {
	duel := NewRNG2(17, 39)
	shared := rng.NewRNG2(17, 39)
	for range 100 {
		if x, y := duel.Uint64(), shared.Uint64(); x != y {
			t.Fatalf("alias stream %x != %x", x, y)
		}
	}
	words := []string{"one", "two", "three", "four"}
	copy := slices.Clone(words)
	ShuffleSlice(&duel, words)
	rng.ShuffleSlice(&shared, copy)
	if !slices.Equal(words, copy) {
		t.Fatal("generic shuffle wrapper changed")
	}
	if duel != shared {
		t.Fatal("wrapper consumed different draws")
	}
	if RulesVersion != 1 {
		t.Fatalf("rules version changed to %d", RulesVersion)
	}
	if got := CatalogHash(); got != "64943c7be6864ff4" {
		t.Fatalf("catalog hash changed: %s", got)
	}
}
