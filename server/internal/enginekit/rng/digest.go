package rng

import "hash/fnv"

// Digest returns the project's FNV-1a 64-bit fingerprint of exactly b. Callers
// own their encodings and domain prefixes. This is not a cryptographic hash.
func Digest(b []byte) uint64 {
	h := fnv.New64a()
	_, _ = h.Write(b)
	return h.Sum64()
}

// Hex formats a digest as exactly 16 lower-case hexadecimal digits.
func Hex(h uint64) string {
	const digits = "0123456789abcdef"
	var b [16]byte
	for i := 15; i >= 0; i-- {
		b[i] = digits[h&15]
		h >>= 4
	}
	return string(b[:])
}
