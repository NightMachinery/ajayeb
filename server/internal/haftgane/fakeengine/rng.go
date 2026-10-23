package fakeengine

// stream is a splitmix64 generator. NewGame draws every random element from
// its own stream, derived from the seed and a fixed tag, as the real engine
// does (engine-architecture.md section 5), so an option that skips a draw
// cannot shift another element's draws. Nothing else in the package draws.
type stream struct{ x uint64 }

// The stream tags. Changing one changes every game's setup.
const (
	tagSeating uint64 = 0x5ea7_0000_0000_0001
	tagBoards  uint64 = 0xb0a2_0000_0000_0002
	tagDeck1   uint64 = 0xdec1_0000_0000_0003
	tagDeck2   uint64 = 0xdec2_0000_0000_0004
)

func newStream(seed, tag uint64) stream {
	s := stream{x: seed ^ tag}
	s.next() // decorrelate seeds that differ in few bits
	return s
}

func (s *stream) next() uint64 {
	s.x += 0x9e3779b97f4a7c15
	z := s.x
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

// below returns a uniform integer in [0, n) by rejection, n > 0.
func (s *stream) below(n uint64) uint64 {
	limit := ^uint64(0) - (^uint64(0) % n)
	for {
		v := s.next()
		if v < limit {
			return v % n
		}
	}
}

// shuffleDesigns is Fisher-Yates over a slice of designs.
func (s *stream) shuffleDesigns(xs []DesignID) {
	for i := len(xs) - 1; i > 0; i-- {
		j := s.below(uint64(i + 1))
		xs[i], xs[j] = xs[j], xs[i]
	}
}

// shuffleBytes is Fisher-Yates over a slice of small integers.
func (s *stream) shuffleBytes(xs []uint8) {
	for i := len(xs) - 1; i > 0; i-- {
		j := s.below(uint64(i + 1))
		xs[i], xs[j] = xs[j], xs[i]
	}
}
