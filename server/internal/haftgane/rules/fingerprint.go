package rules

import (
	"encoding/binary"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// No memory padding or JSON enters this encoding. Fields follow State then
// Record declaration order, fixed-width little-endian, booleans as 0/1 bytes.
// A test independently compares encoding/binary's field walk and pins the size.
const gameEncodingSize = 2157

func (g *Game) Fingerprint() string {
	if g == nil {
		return ""
	}
	var buf [gameEncodingSize]byte
	return rng.Hex(rng.Digest(appendGameEncoding(buf[:0], g)))
}

type gameEncoder struct{ buf []byte }

func (w *gameEncoder) u8(v ...uint8) { w.buf = append(w.buf, v...) }
func (w *gameEncoder) u16(v uint16)  { w.buf = binary.LittleEndian.AppendUint16(w.buf, v) }
func (w *gameEncoder) u32(v uint32)  { w.buf = binary.LittleEndian.AppendUint32(w.buf, v) }
func (w *gameEncoder) u64(v uint64)  { w.buf = binary.LittleEndian.AppendUint64(w.buf, v) }
func (w *gameEncoder) boolean(v bool) {
	if v {
		w.u8(1)
	} else {
		w.u8(0)
	}
}
func (w *gameEncoder) ids(ids []catalog.DesignID) {
	for _, id := range ids {
		w.u8(uint8(id))
	}
}
func (w *gameEncoder) city(c *City) {
	w.u8(uint8(c.Board), uint8(c.Side), c.Stages)
	w.ids(c.Tucked[:])
	for _, word := range c.Buildings {
		w.u64(word)
	}
	w.u16(uint16(c.Coins))
	w.u8(c.Victories[:]...)
	w.u8(c.Defeats, uint8(c.Kept))
	w.u8(c.Produce[:]...)
	w.u8(c.ForSale[:]...)
	for _, m := range c.Choice {
		w.u8(uint8(m))
	}
	w.u8(c.NChoice)
	for _, m := range c.ChoiceForSale {
		w.u8(uint8(m))
	}
	w.u8(c.NChoiceForSale)
	for _, m := range c.Discount {
		w.u8(uint8(m))
	}
	w.u8(c.Colors[:]...)
	w.u8(c.Science[:]...)
	w.u8(c.ScienceChoice, c.Shields)
	w.u32(c.Chains)
	w.u8(c.Abilities)
}
func (w *gameEncoder) result(r *Result) {
	w.boolean(r.Over)
	for _, s := range r.Scores {
		for _, v := range [8]int16{s.Wonder, s.Treasure, s.Military, s.Blue, s.Yellow, s.Green, s.Purple, s.Total} {
			w.u16(uint16(v))
		}
	}
	w.u8(r.Rank[:]...)
	w.u8(r.Winners)
}
func appendGameEncoding(dst []byte, g *Game) []byte {
	w := gameEncoder{buf: dst}
	st := &g.State
	w.u64(st.Seed)
	w.u8(uint8(st.Opts.Sides), uint8(st.Opts.Seating))
	w.boolean(st.Opts.HideAtEnd)
	w.u8(st.Seats, uint8(st.Phase), st.Age, st.Turn)
	w.u16(st.Points)
	w.u8(uint8(st.Decider))
	for i := range st.Cities {
		w.city(&st.Cities[i])
	}
	for _, hand := range st.Hands {
		w.ids(hand[:])
	}
	for _, age := range st.Deals {
		for _, hand := range age {
			w.ids(hand[:])
		}
	}
	w.ids(st.BoxGuilds[:])
	w.u8(st.Discard[:]...)
	w.result(&st.Result)
	for _, age := range g.Record.Packets {
		for _, hand := range age {
			w.ids(hand[:])
		}
	}
	for _, age := range g.Record.Picks {
		for _, turn := range age {
			w.ids(turn[:])
		}
	}
	for _, age := range g.Record.Plays {
		for _, turn := range age {
			for _, p := range turn {
				w.u8(uint8(p.Kind), uint8(p.Design), p.Stage, p.Left, p.Right)
			}
		}
	}
	for _, age := range g.Record.Rebuilt {
		for _, turn := range age {
			w.ids(turn[:])
		}
	}
	return w.buf
}
