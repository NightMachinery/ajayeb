package fakeengine

// Observation is what a bot in one seat may know, built by whitelist and
// never by redacting a copy of State: no seed, the pile as a count except
// during the seat's own salvage point, and only the seat's own private
// cards and remembered hands.
type Observation struct {
	Now  Current
	Know Knowledge
}

// Current is the position now, as the seat may see it.
type Current struct {
	Seat    Seat
	Opts    Options // there is no seed anywhere in an observation
	Seats   uint8
	Phase   Phase
	Age     uint8
	Turn    uint8
	Points  uint16
	Decider Seat
	Cities  [MaxSeats]PublicCity
	Pile    uint8                 // the pile's card count, public
	Hand    [HandSize]DesignID    // own hand, sorted
	Kept    DesignID              // own kept card
	Tucked  [MaxStages]DesignID   // own tucked cards, by stage
	Offer   [NumDesigns + 1]uint8 // the pile by design, only during the seat's own salvage point
	Result  Result                // zero until over
}

// PublicCity is one seat's public city.
type PublicCity struct {
	Board    BoardID
	Built    uint32
	Stages   uint8
	Coins    int16
	HandSize uint8
	HasKept  bool
}

// Knowledge is what the seat remembers: the hand it held at each of its
// turns, its own actions and everyone's public plays, per resolved point.
type Knowledge struct {
	Seen  [NumAges][TurnsPerAge][HandSize]DesignID
	Own   [MaxPoints]Action           // the seat's own actions (zero where it was not required)
	Plays [MaxPoints][MaxSeats]Action // public forms: the design only for a build or salvage
}

// Observe returns seat s's observation. Bots only; nothing random.
func (g *Game) Observe(s Seat) Observation {
	st := &g.State
	var o Observation
	n := &o.Now
	n.Seat, n.Opts, n.Seats, n.Phase = s, st.Opts, st.Seats, st.Phase
	n.Age, n.Turn, n.Points, n.Decider = st.Age, st.Turn, st.Points, st.Decider
	for i := Seat(0); uint8(i) < st.Seats; i++ {
		c := &st.Cities[i]
		n.Cities[i] = PublicCity{
			Board: c.Board, Built: c.Built, Stages: c.Stages, Coins: c.Coins,
			HandSize: handCount(&st.Hands[i]), HasKept: c.Kept != 0,
		}
	}
	for d := DesignID(1); d <= NumDesigns; d++ {
		n.Pile += st.Pile[d]
	}
	if uint8(s) >= st.Seats {
		return o
	}
	n.Hand = st.Hands[s]
	n.Kept = st.Cities[s].Kept
	n.Tucked = st.Cities[s].Tucked
	if st.Phase == PhaseSalvage && st.Decider == s {
		n.Offer = st.Pile
	}
	if st.Result.Over {
		n.Result = st.Result
	}
	k := &o.Know
	for a := 0; a < NumAges; a++ {
		for t := 0; t < TurnsPerAge; t++ {
			k.Seen[a][t] = g.Record.Held[a][t][s]
		}
	}
	for i := uint8(0); i < g.Record.NLog; i++ {
		e := &g.Record.Log[i]
		if e.Required.Has(s) {
			k.Own[i] = e.Actions[s]
		}
		for j := Seat(0); uint8(j) < st.Seats; j++ {
			if !e.Required.Has(j) {
				continue
			}
			a := e.Actions[j]
			if a.Kind != ActBuild && a.Kind != ActSalvage {
				a.Design = 0
			}
			k.Plays[i][j] = a
		}
	}
	return o
}

// Fingerprint digests the observation's canonical encoding: the fields in
// declaration order, little-endian, written explicitly. Two histories that
// leave the seat the same knowledge give the same digest. It is the
// platform's "permitted-observation hash": it never goes into a frame, a
// log or a download.
func (o *Observation) Fingerprint() string {
	e := newEncoder()
	n := &o.Now
	e.u8(uint8(n.Seat))
	e.options(n.Opts)
	e.u8(n.Seats)
	e.u8(uint8(n.Phase))
	e.u8(n.Age)
	e.u8(n.Turn)
	e.u16(n.Points)
	e.u8(uint8(n.Decider))
	for i := range n.Cities {
		c := &n.Cities[i]
		e.u8(uint8(c.Board))
		e.u32(c.Built)
		e.u8(c.Stages)
		e.u16(uint16(c.Coins))
		e.u8(c.HandSize)
		e.bool(c.HasKept)
	}
	e.u8(n.Pile)
	e.designs(n.Hand[:])
	e.u8(uint8(n.Kept))
	e.designs(n.Tucked[:])
	for _, c := range n.Offer {
		e.u8(c)
	}
	e.result(&n.Result)
	k := &o.Know
	for a := range k.Seen {
		for t := range k.Seen[a] {
			e.designs(k.Seen[a][t][:])
		}
	}
	for i := range k.Own {
		e.action(k.Own[i])
	}
	for i := range k.Plays {
		for j := range k.Plays[i] {
			e.action(k.Plays[i][j])
		}
	}
	return e.hex()
}
