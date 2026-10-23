package rng

import "testing"

// Pinned before refactoring the public Duel seed derivation, including both
// additions' uint64 wraparound cases.
func TestMixSeedGolden(t *testing.T) {
	for _, c := range []struct{ seed, tag, want uint64 }{
		{0x0000000000000000, 0x0000000000000000, 0xe220a8397b1dcdaf},
		{0x0000000000000000, 0x0000000000000001, 0x6e789e6aa1b965f4},
		{0x0000000000000000, 0x0000000000000002, 0x06c45d188009454f},
		{0x0000000000000000, 0x000000000000007f, 0x12d05c4045a39c19},
		{0x0000000000000000, 0xffffffffffffffff, 0x0000000000000000},
		{0x0000000000000001, 0x0000000000000000, 0x910a2dec89025cc1},
		{0x0000000000000001, 0x0000000000000001, 0xbeeb8da1658eec67},
		{0x0000000000000001, 0x0000000000000002, 0xf893a2eefb32555e},
		{0x0000000000000001, 0x000000000000007f, 0x6524e51ffe73cb44},
		{0x0000000000000001, 0xffffffffffffffff, 0x5692161d100b05e5},
		{0x123456789abcdef0, 0x0000000000000000, 0x161922c645ce50e8},
		{0x123456789abcdef0, 0x0000000000000001, 0xad760cafa1697b60},
		{0x123456789abcdef0, 0x0000000000000002, 0x3501ff44902ca50d},
		{0x123456789abcdef0, 0x000000000000007f, 0x8e059f98e9c6069b},
		{0x123456789abcdef0, 0xffffffffffffffff, 0x9629f58e8ec5b906},
		{0xffffffffffffffff, 0x0000000000000000, 0xe4d971771b652c20},
		{0xffffffffffffffff, 0x0000000000000001, 0xe99ff867dbf682c9},
		{0xffffffffffffffff, 0x0000000000000002, 0x382ff84cb27281e9},
		{0xffffffffffffffff, 0x000000000000007f, 0x9b041196862f792c},
		{0xffffffffffffffff, 0xffffffffffffffff, 0xb4d055fcf2cbbd7b},
	} {
		if got := MixSeed(c.seed, c.tag); got != c.want {
			t.Fatalf("MixSeed(%x,%x)=%x want%x", c.seed, c.tag, got, c.want)
		}
	}
}
