package engine

import (
	"math/big"
	"math/rand/v2"
	"slices"
	"testing"
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

// TestRNGIsPCG checks that the raw stream is math/rand/v2's PCG with the
// documented seeding.
func TestRNGIsPCG(t *testing.T) {
	for _, seed := range []uint64{0, 1, 42, 1<<64 - 1} {
		r := NewRNG(seed)
		p := rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)
		for i := range 100 {
			if a, b := r.Uint64(), p.Uint64(); a != b {
				t.Fatalf("seed %d draw %d: %#x, PCG gives %#x", seed, i, a, b)
			}
		}
	}
}

// TestUint64NMatchesReference compares Uint64N with a big-integer statement
// of Lemire's method on the same raw stream: take x * n as a 128-bit number,
// reject it while its low 64 bits are below 2^64 mod n, return its high 64
// bits. The large bounds reject often, which exercises the retry loop.
func TestUint64NMatchesReference(t *testing.T) {
	two64 := new(big.Int).Lsh(big.NewInt(1), 64)
	bounds := []uint64{1, 2, 3, 7, 10, 20, 23, 1 << 32, 1<<32 + 1, 1<<63 - 1, 1 << 63, 1<<63 + 1,
		(1<<64-1)/3 + 1, 1<<64 - 2, 1<<64 - 1}
	for _, n := range bounds {
		r := NewRNG2(n, 99)
		p := rand.NewPCG(n, 99)
		bn := new(big.Int).SetUint64(n)
		threshold := new(big.Int).Mod(two64, bn)
		rejected := 0
		for i := range 300 {
			var want uint64
			for {
				prod := new(big.Int).Mul(new(big.Int).SetUint64(p.Uint64()), bn)
				lo := new(big.Int).Mod(prod, two64)
				if lo.Cmp(threshold) >= 0 {
					want = new(big.Int).Rsh(prod, 64).Uint64()
					break
				}
				rejected++
			}
			got := r.Uint64N(n)
			if got != want {
				t.Fatalf("Uint64N(%d) draw %d = %d, reference %d", n, i, got, want)
			}
			if got >= n {
				t.Fatalf("Uint64N(%d) = %d is out of range", n, got)
			}
		}
		if n == 1<<63+1 && rejected == 0 {
			t.Errorf("Uint64N(2^63+1) never rejected; the retry loop went untested")
		}
	}
}

func TestRNGPanics(t *testing.T) {
	r := NewRNG(1)
	for name, f := range map[string]func(){
		"Uint64N(0)":  func() { r.Uint64N(0) },
		"IntN(0)":     func() { r.IntN(0) },
		"IntN(-1)":    func() { r.IntN(-1) },
		"Shuffle(-1)": func() { r.Shuffle(-1, func(int, int) {}) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s did not panic", name)
				}
			}()
			f()
		}()
	}
	r.Shuffle(0, func(int, int) { t.Error("Shuffle(0) swapped") })
	r.Shuffle(1, func(int, int) { t.Error("Shuffle(1) swapped") })
}

func TestRNGCopyForksAndMarshal(t *testing.T) {
	a := NewRNG(5)
	a.Uint64()
	b := a // a copy continues the same stream
	for i := range 10 {
		if x, y := a.Uint64(), b.Uint64(); x != y {
			t.Fatalf("draw %d: copies diverged", i)
		}
	}
	blob, err := a.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	want := a.Uint64()
	var c RNG
	if err := c.UnmarshalBinary(blob); err != nil {
		t.Fatal(err)
	}
	if got := c.Uint64(); got != want {
		t.Errorf("restored generator gave %#x, want %#x", got, want)
	}
	if err := c.UnmarshalBinary([]byte("junk")); err == nil {
		t.Error("UnmarshalBinary accepted junk")
	}
}

func TestShuffleIsAPermutationAndUniform(t *testing.T) {
	r := NewRNG(2026)
	// Shuffle and ShuffleSlice consume the stream identically.
	r2 := r
	x := []int{0, 1, 2, 3, 4, 5, 6, 7}
	y := slices.Clone(x)
	ShuffleSlice(&r, x)
	r2.Shuffle(len(y), func(i, j int) { y[i], y[j] = y[j], y[i] })
	if !slices.Equal(x, y) {
		t.Fatalf("ShuffleSlice %v and Shuffle %v differ", x, y)
	}
	sorted := slices.Clone(x)
	slices.Sort(sorted)
	if !slices.Equal(sorted, []int{0, 1, 2, 3, 4, 5, 6, 7}) {
		t.Fatalf("shuffle lost elements: %v", x)
	}
	// All 6 orders of 3 elements appear about equally often. The seed is
	// fixed, so this is deterministic; the bound is about 6 standard
	// deviations.
	const draws = 60000
	var counts [6]int
	perms := [][3]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}}
	for range draws {
		p := [3]int{0, 1, 2}
		ShuffleSlice(&r, p[:])
		counts[slices.Index(perms, p)]++
	}
	for i, n := range counts {
		if n < draws/6-550 || n > draws/6+550 {
			t.Errorf("order %v came up %d times of %d", perms[i], n, draws)
		}
	}
}
