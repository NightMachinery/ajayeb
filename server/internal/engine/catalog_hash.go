package engine

import (
	"hash/fnv"

	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// catalogHashFormat versions the encoding below. Changing the encoding (not
// the data) bumps it, so that the hash still changes exactly when either
// changes.
const catalogHashFormat = 1

// CatalogHash fingerprints the catalog the engine plays with: 16 lower-case
// hex digits of FNV-1a (64-bit) over a canonical encoding of every card,
// wonder and progress token, with all their fields (id, slug, names, age,
// back, colour, cost, chains and every field of every effect), in id order.
// Every view carries it, so a client whose presentation catalog was
// generated from other data can tell that it is stale. Any change to the
// data changes the hash in practice; the same data always gives the same
// hash.
func CatalogHash() string {
	h := fnv.New64a()
	b := make([]byte, 0, 6<<10) // the base game encodes to about 4.5 KB
	b = append(b, 'a', 'j', 'c', catalogHashFormat)
	for d := range catalog.Cards() {
		b = append(b, 'c', byte(d.ID))
		b = hashString(b, d.Slug)
		b = hashString(b, d.Name)
		b = append(b, byte(len(d.AltNames)))
		for _, n := range d.AltNames {
			b = hashString(b, n)
		}
		b = append(b, d.Age, byte(d.Back), byte(d.Color))
		b = hashCost(b, d.Cost)
		b = append(b, byte(d.ChainIn), byte(d.ChainOut))
		b = hashEffects(b, d.Effects)
	}
	for d := range catalog.Wonders() {
		b = append(b, 'w', byte(d.ID))
		b = hashString(b, d.Slug)
		b = hashString(b, d.Name)
		b = hashCost(b, d.Cost)
		b = hashEffects(b, d.Effects)
	}
	for d := range catalog.Tokens() {
		b = append(b, 't', byte(d.ID))
		b = hashString(b, d.Slug)
		b = hashString(b, d.Name)
		b = hashEffects(b, d.Effects)
	}
	_, _ = h.Write(b) // a hash.Hash never returns an error
	const digits = "0123456789abcdef"
	sum := h.Sum64()
	var out [16]byte
	for i := 15; i >= 0; i-- {
		out[i] = digits[sum&15]
		sum >>= 4
	}
	return string(out[:])
}

// hashString appends a string with its length, so that no two different
// sequences of strings encode the same.
func hashString(b []byte, s string) []byte {
	n := len(s)
	b = append(b, byte(n), byte(n>>8))
	return append(b, s...)
}

// hashCost appends a printed cost.
func hashCost(b []byte, c catalog.Cost) []byte {
	b = append(b, c.Coins)
	return append(b, c.Res[:]...)
}

// hashEffects appends a list of effects with every field.
func hashEffects(b []byte, fx []catalog.Effect) []byte {
	b = append(b, byte(len(fx)))
	for _, e := range fx {
		b = append(b, byte(e.Kind), e.N, byte(e.Res), byte(e.Mask), byte(e.Colors),
			byte(e.Symbol), byte(e.Scope), byte(e.Target))
	}
	return b
}
