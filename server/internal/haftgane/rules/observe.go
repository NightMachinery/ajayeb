package rules

import (
	"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// Observation is the bot's entire immutable input. No field refers back to a
// Game, seed, unknown place, or cached production symbol.
type Observation struct {
	Now  Current
	Know Knowledge
}
type Current struct {
	Seat      Seat
	Opts      Options
	Seats     uint8
	Phase     Phase
	Age, Turn uint8
	Points    uint16
	Decider   Seat
	Cities    [MaxSeats]PublicCity
	Pile      uint8
	Hand      [HandSize]catalog.DesignID
	Kept      catalog.DesignID
	Tucked    [MaxStages]catalog.DesignID
	Offer     [catalog.NumDesigns + 1]uint8
	Result    Result
}
type PublicCity struct {
	Board     catalog.BoardID
	Side      catalog.Side
	Stages    uint8
	Buildings catalog.DesignSet
	Coins     int16
	Victories [NumAges]uint8
	Defeats   uint8
	HandSize  uint8
	HasKept   bool
}
type Knowledge struct {
	Seen      [NumAges][MaxTurns][HandSize]catalog.DesignID
	Picks     [NumAges][MaxTurns]catalog.DesignID
	Plays     [NumAges][MaxTurns][MaxSeats]Play
	Rebuilt   [NumAges][TurnsPerAge][MaxSeats]catalog.DesignID
	PileLooks [MaxPileLooks]PileLook
}
type PileLook struct {
	Age, Turn uint8
	Count     [catalog.NumDesigns + 1]uint8
}

func (o Observation) Clone() Observation { return o }

// Project is usable on sampled states. Only this seat's current private cards
// are read; the public cities contain neither private holders nor caches.
func Project(st *State, seat Seat) Current {
	out := Current{Seat: NoSeat, Decider: NoSeat}
	if st == nil || AllSeats(st.Seats) == 0 {
		return out
	}
	out.Opts, out.Seats, out.Phase = st.Opts, st.Seats, st.Phase
	out.Age, out.Turn, out.Points, out.Decider = st.Age, st.Turn, st.Points, st.Decider
	if st.Phase == PhaseOver {
		out.Result = st.Result
	}
	for s := Seat(0); s < Seat(st.Seats); s++ {
		c := &st.Cities[s]
		p := PublicCity{Board: c.Board, Side: c.Side, Stages: c.Stages, Buildings: c.Buildings, Coins: c.Coins, Victories: c.Victories, Defeats: c.Defeats, HasKept: c.Kept != 0}
		for _, id := range st.Hands[s] {
			if id != 0 {
				p.HandSize++
			}
		}
		out.Cities[s] = p
	}
	for _, n := range st.Discard {
		out.Pile += n
	}
	if seat.Valid(st.Seats) {
		out.Seat = seat
		out.Hand, out.Kept, out.Tucked = st.Hands[seat], st.Cities[seat].Kept, st.Cities[seat].Tucked
		canonicalIDs(out.Hand[:])
		canonicalIDs(out.Tucked[:])
		if st.Phase == PhaseDiscardBuild && st.Decider == seat {
			out.Offer = st.Discard
		}
	}
	return out
}
func (g *Game) Observe(seat Seat) Observation {
	if g == nil {
		return Observation{Now: Project(nil, seat)}
	}
	return Observation{Now: Project(&g.State, seat), Know: g.knowledge(seat)}
}

// publicState rebuilds only derived public facts, for quotes and presentation.
// Its missing private holders are intentional; it is not a playable world.
func (c *Current) publicState() (State, bool) {
	if c == nil || AllSeats(c.Seats) == 0 {
		return State{}, false
	}
	st := State{Opts: c.Opts, Seats: c.Seats, Phase: c.Phase, Age: c.Age, Turn: c.Turn, Points: c.Points, Decider: c.Decider, Result: c.Result}
	for seat := Seat(0); seat < Seat(c.Seats); seat++ {
		p := c.Cities[seat]
		city, err := cityCaches(City{Board: p.Board, Side: p.Side, Stages: p.Stages, Buildings: p.Buildings, Coins: p.Coins, Victories: p.Victories, Defeats: p.Defeats})
		if err != nil {
			return State{}, false
		}
		st.Cities[seat] = city
	}
	return st, true
}
func (o *Observation) Quote(seat Seat, cost catalog.Cost) Quote {
	if o == nil {
		return Quote{}
	}
	return o.Now.Quote(seat, cost)
}
func (c *Current) Quote(seat Seat, cost catalog.Cost) Quote {
	st, ok := c.publicState()
	if !ok {
		return Quote{}
	}
	return QuoteState(&st, seat, cost)
}

// ObservationEncodingSize is the explicit field encoding, independent of Go
// padding. Keep the representation's size local until the limits gate freezes it.
const ObservationEncodingSize = 1693

func (o Observation) AppendEncoding(dst []byte) []byte {
	w := gameEncoder{buf: dst}
	c := &o.Now
	w.u8(uint8(c.Seat), uint8(c.Opts.Sides), uint8(c.Opts.Seating))
	w.boolean(c.Opts.HideAtEnd)
	w.u8(c.Seats, uint8(c.Phase), c.Age, c.Turn)
	w.u16(c.Points)
	w.u8(uint8(c.Decider))
	for _, p := range c.Cities {
		w.u8(uint8(p.Board), uint8(p.Side), p.Stages)
		for _, word := range p.Buildings {
			w.u64(word)
		}
		w.u16(uint16(p.Coins))
		w.u8(p.Victories[:]...)
		w.u8(p.Defeats, p.HandSize)
		w.boolean(p.HasKept)
	}
	w.u8(c.Pile)
	w.ids(c.Hand[:])
	w.u8(uint8(c.Kept))
	w.ids(c.Tucked[:])
	w.u8(c.Offer[:]...)
	w.result(&c.Result)
	k := &o.Know
	for _, age := range k.Seen {
		for _, hand := range age {
			w.ids(hand[:])
		}
	}
	for _, age := range k.Picks {
		w.ids(age[:])
	}
	for _, age := range k.Plays {
		for _, turn := range age {
			for _, p := range turn {
				w.u8(uint8(p.Kind), uint8(p.Design), p.Stage, p.Left, p.Right)
			}
		}
	}
	for _, age := range k.Rebuilt {
		for _, turn := range age {
			w.ids(turn[:])
		}
	}
	for _, look := range k.PileLooks {
		w.u8(look.Age, look.Turn)
		w.u8(look.Count[:]...)
	}
	return w.buf
}
func (o Observation) Fingerprint() string {
	var buf [ObservationEncodingSize]byte
	return rng.Hex(rng.Digest(o.AppendEncoding(buf[:0])))
}

var _ = func(a, b Observation) bool { return a == b }
