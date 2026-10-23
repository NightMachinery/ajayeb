package rng

// MixSeed derives a seed from a base and an indexed tag using the splitmix64
// finalizer. It is a pure derivation, not a draw from any game or policy stream.
// The constants and overflow arithmetic preserve Duel's original ai.MixSeed.
func MixSeed(seed, tag uint64) uint64 {
	z := seed + 0x9e3779b97f4a7c15*(tag+1)
	z = (z ^ z>>30) * 0xbf58476d1ce4e5b9
	z = (z ^ z>>27) * 0x94d049bb133111eb
	return z ^ z>>31
}
