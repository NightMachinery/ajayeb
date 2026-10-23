package rng

import "testing"

// Known FNV-1a 64-bit vectors pin both the algorithm and lower-case hex form.
func TestDigestGolden(t *testing.T) {
	for _, tt := range []struct {
		input string
		sum   uint64
		hex   string
	}{
		{"", 0xcbf29ce484222325, "cbf29ce484222325"},
		{"a", 0xaf63dc4c8601ec8c, "af63dc4c8601ec8c"},
		{"hello", 0xa430d84680aabd0b, "a430d84680aabd0b"},
	} {
		t.Run(tt.hex, func(t *testing.T) {
			b := []byte(tt.input)
			if got := Digest(b); got != tt.sum {
				t.Fatalf("digest %x want%x", got, tt.sum)
			}
			if got := Hex(tt.sum); got != tt.hex {
				t.Fatalf("hex %s want%s", got, tt.hex)
			}
		})
	}
	if Digest(nil) != Digest([]byte{}) {
		t.Fatal("empty byte representations differ")
	}
	for _, tt := range []struct {
		sum  uint64
		want string
	}{{0, "0000000000000000"}, {1, "0000000000000001"}, {0xabcdef, "0000000000abcdef"}, {1<<64 - 1, "ffffffffffffffff"}} {
		if got := Hex(tt.sum); got != tt.want {
			t.Fatalf("Hex(%x)=%s want%s", tt.sum, got, tt.want)
		}
	}
}
