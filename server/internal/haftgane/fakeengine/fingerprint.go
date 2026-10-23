package fakeengine

// encoder is FNV-1a over an explicit little-endian encoding, written field
// by field (never raw memory or padding) and hashed as it goes.
type encoder struct{ h uint64 }

func newEncoder() encoder { return encoder{h: 14695981039346656037} }

func (e *encoder) u8(b uint8) {
	e.h ^= uint64(b)
	e.h *= 1099511628211
}

func (e *encoder) u16(v uint16) { e.u8(uint8(v)); e.u8(uint8(v >> 8)) }
func (e *encoder) u32(v uint32) { e.u16(uint16(v)); e.u16(uint16(v >> 16)) }
func (e *encoder) u64(v uint64) { e.u32(uint32(v)); e.u32(uint32(v >> 32)) }

func (e *encoder) bool(b bool) {
	if b {
		e.u8(1)
	} else {
		e.u8(0)
	}
}

func (e *encoder) designs(ds []DesignID) {
	for _, d := range ds {
		e.u8(uint8(d))
	}
}

func (e *encoder) options(o Options) {
	e.u8(uint8(o.Seating))
	e.bool(o.HideAtEnd)
}

func (e *encoder) action(a Action) {
	e.u8(uint8(a.Kind))
	e.u8(uint8(a.Design))
	e.u8(a.Left)
	e.u8(a.Right)
}

func (e *encoder) result(r *Result) {
	e.bool(r.Over)
	for i := range r.Scores {
		sc := &r.Scores[i]
		e.u16(uint16(sc.Built))
		e.u16(uint16(sc.Stages))
		e.u16(uint16(sc.Treasure))
		e.u16(uint16(sc.Total))
	}
	for _, rk := range r.Rank {
		e.u8(rk)
	}
	e.u8(uint8(r.Winners))
}

// hex returns the digest as 16 lower-case hex digits.
func (e *encoder) hex() string {
	const digits = "0123456789abcdef"
	var b [16]byte
	for i := 15; i >= 0; i-- {
		b[i] = digits[e.h&0xf]
		e.h >>= 4
	}
	return string(b[:])
}

// Fingerprint is FNV-1a over the explicit encoding of the game, the state
// then the record, as 16 hex digits. It fingerprints, it does not
// authenticate. It covers the seed, so it never goes into a frame or log.
func (g *Game) Fingerprint() string {
	e := newEncoder()
	st := &g.State
	e.u64(st.Seed)
	e.options(st.Opts)
	e.u8(st.Seats)
	e.u8(uint8(st.Phase))
	e.u8(st.Age)
	e.u8(st.Turn)
	e.u16(st.Points)
	e.u8(uint8(st.Decider))
	for i := range st.Cities {
		c := &st.Cities[i]
		e.u8(uint8(c.Board))
		e.u32(c.Built)
		e.designs(c.Tucked[:])
		e.u8(c.Stages)
		e.u16(uint16(c.Coins))
		e.u8(uint8(c.Kept))
	}
	for i := range st.Hands {
		e.designs(st.Hands[i][:])
	}
	for i := range st.Deals {
		e.designs(st.Deals[i][:])
	}
	e.designs(st.Box[:])
	for _, c := range st.Pile {
		e.u8(c)
	}
	e.result(&st.Result)
	r := &g.Record
	for a := range r.Held {
		for t := range r.Held[a] {
			for s := range r.Held[a][t] {
				e.designs(r.Held[a][t][s][:])
			}
		}
	}
	for i := range r.Log {
		l := &r.Log[i]
		e.u8(uint8(l.Kind))
		e.u8(l.Age)
		e.u8(l.Turn)
		e.u8(uint8(l.Required))
		for j := range l.Actions {
			e.action(l.Actions[j])
		}
	}
	e.u8(r.NLog)
	return e.hex()
}
