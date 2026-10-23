package fakeengine

// EventKind is what an event reports.
type EventKind uint8

const (
	EvPlay           EventKind = iota + 1 // a seat played a card (its design private for a tuck or sale)
	EvLeftover                            // a seat's last card went to the pile (design private)
	EvKept                                // the orchard's holder kept its last card (design private)
	EvSalvage                             // the archive's holder built a design from the pile (public)
	EvSalvageNothing                      // the archive's holder tucked but the pile held nothing for it
	EvPass                                // the hands passed (Dir: 1 left, 2 right)
	EvAgeStart                            // an Age was dealt
	EvGameOver                            // the game ended and was scored
)

var eventNames = [...]string{
	EvPlay: "play", EvLeftover: "leftover", EvKept: "kept", EvSalvage: "salvage",
	EvSalvageNothing: "salvage-nothing", EvPass: "pass", EvAgeStart: "age-start", EvGameOver: "game-over",
}

// String returns the event kind's wire name.
func (k EventKind) String() string {
	if k == 0 || int(k) >= len(eventNames) {
		return ""
	}
	return eventNames[k]
}

// Event is one thing that happened, in its full (unfiltered) form. Design
// is private for a tucked or sold play, a leftover and a kept card; ViewFor
// and History filter it per viewer. Events never carry a key.
type Event struct {
	Kind   EventKind
	Seat   Seat
	Play   ActionKind // EvPlay only
	Design DesignID
	Stage  uint8 // the stage a tuck filled, 1 or 2
	Left   uint8 // coins a build paid to the left neighbour
	Right  uint8
	Age    uint8 // EvAgeStart
	Dir    uint8 // EvPass: 1 left, 2 right
}

// Resolve resolves one complete batch for the open point and returns the
// next game and the resolution's events in canonical order. It checks that
// the batch names the open point, holds exactly one pick per required seat
// sorted by seat, and that every key is in that seat's Legal on this
// frozen position. Every price and condition is read from that position, so
// the order of seats inside the batch decides nothing. g never changes.
func (g *Game) Resolve(b Batch) (Game, []Event, error) {
	p := g.Point()
	if p.Kind == PointNone {
		return Game{}, nil, ErrOver
	}
	if b.Point != p.ID {
		return Game{}, nil, ErrWrongPoint
	}
	if len(b.Picks) != p.Required.Count() {
		return Game{}, nil, ErrBatch
	}
	var acts [MaxSeats]Action
	for i, pk := range b.Picks {
		if !p.Required.Has(pk.Seat) || (i > 0 && pk.Seat <= b.Picks[i-1].Seat) {
			return Game{}, nil, ErrBatch
		}
		a, ok := g.legalAction(pk.Seat, pk.Key)
		if !ok {
			return Game{}, nil, ErrIllegal
		}
		acts[pk.Seat] = a
	}
	n := *g
	evs := n.apply(p, &acts, make([]Event, 0, MaxEvents))
	return n, evs, nil
}

// legalAction returns the engine's own action for a key offered to s: the
// server applies it, never one built from client data.
func (g *Game) legalAction(s Seat, key string) (Action, bool) {
	for _, c := range g.Legal(s) {
		if c.Key == key {
			return Action{Kind: c.Kind, Design: c.Design, Left: c.Left, Right: c.Right}, true
		}
	}
	return Action{}, false
}

// apply resolves a validated point. History replays through it too.
func (g *Game) apply(p Point, acts *[MaxSeats]Action, evs []Event) []Event {
	st := &g.State
	g.Record.Log[g.Record.NLog] = Resolved{Kind: p.Kind, Age: st.Age, Turn: st.Turn, Required: p.Required, Actions: *acts}
	g.Record.NLog++
	st.Points++
	switch p.Kind {
	case PointTurn:
		return g.resolveTurn(acts, evs)
	case PointSalvage:
		d := acts[st.Decider].Design
		st.Pile[d]--
		st.Cities[st.Decider].Built |= 1 << d
		evs = append(evs, Event{Kind: EvSalvage, Seat: st.Decider, Design: d})
		return g.afterTurn(evs)
	case PointLastCard:
		s := st.Decider
		evs = g.play(s, acts[s], evs)
		st.Cities[s].Kept = 0
		return g.endAge(evs)
	}
	return evs
}

// resolveTurn plays every seat's card, then sets aside each Age's last
// cards, then checks the archive's salvage, then passes. Payments are sums,
// so the seat order of the loops decides nothing.
func (g *Game) resolveTurn(acts *[MaxSeats]Action, evs []Event) []Event {
	st := &g.State
	archiveTucked := false
	for s := Seat(0); uint8(s) < st.Seats; s++ {
		removeOne(&st.Hands[s], acts[s].Design)
		evs = g.play(s, acts[s], evs)
		if acts[s].Kind == ActTuck && st.Cities[s].Board == BoardArchive {
			archiveTucked = true
		}
	}
	if st.Turn == TurnsPerAge {
		for s := Seat(0); uint8(s) < st.Seats; s++ {
			last := st.Hands[s][0]
			st.Hands[s][0] = 0
			if st.Cities[s].Board == BoardOrchard {
				st.Cities[s].Kept = last
				evs = append(evs, Event{Kind: EvKept, Seat: s, Design: last})
			} else {
				st.Pile[last]++
				evs = append(evs, Event{Kind: EvLeftover, Seat: s, Design: last})
			}
		}
	}
	if archiveTucked {
		s, _ := st.boardHolder(BoardArchive)
		if st.salvageable(s) {
			st.Phase, st.Decider = PhaseSalvage, s
			return evs
		}
		evs = append(evs, Event{Kind: EvSalvageNothing, Seat: s})
	}
	return g.afterTurn(evs)
}

// play applies one seat's action on its card (already out of its hand).
func (g *Game) play(s Seat, a Action, evs []Event) []Event {
	st := &g.State
	c := &st.Cities[s]
	ev := Event{Kind: EvPlay, Seat: s, Play: a.Kind, Design: a.Design}
	switch a.Kind {
	case ActBuild:
		c.Built |= 1 << a.Design
		c.Coins -= int16(a.Left) + int16(a.Right)
		st.Cities[st.left(s)].Coins += int16(a.Left)
		st.Cities[st.right(s)].Coins += int16(a.Right)
		ev.Left, ev.Right = a.Left, a.Right
	case ActTuck:
		c.Tucked[c.Stages] = a.Design
		c.Stages++
		ev.Stage = c.Stages
	case ActSell:
		st.Pile[a.Design]++
		c.Coins += SellCoins
	}
	return append(evs, ev)
}

// afterTurn passes the hands and opens the next turn, or ends the Age.
func (g *Game) afterTurn(evs []Event) []Event {
	st := &g.State
	if st.Turn < TurnsPerAge {
		var next [MaxSeats][HandSize]DesignID
		dir := uint8(1)
		for s := Seat(0); uint8(s) < st.Seats; s++ {
			if st.Age == 1 {
				next[st.left(s)] = st.Hands[s]
			} else {
				next[st.right(s)] = st.Hands[s]
				dir = 2
			}
		}
		st.Hands = next
		evs = append(evs, Event{Kind: EvPass, Dir: dir})
		st.Turn++
		st.Phase, st.Decider = PhaseTurn, 0
		g.recordHeld()
		return evs
	}
	for s := Seat(0); uint8(s) < st.Seats; s++ {
		if st.Cities[s].Kept != 0 {
			st.Phase, st.Decider = PhaseLastCard, s
			return evs
		}
	}
	return g.endAge(evs)
}

// endAge deals the next Age, or scores the game.
func (g *Game) endAge(evs []Event) []Event {
	st := &g.State
	if st.Age < NumAges {
		st.Age++
		st.Turn = 1
		st.Hands = st.Deals
		st.Deals = [MaxSeats][HandSize]DesignID{}
		st.Phase, st.Decider = PhaseTurn, 0
		g.recordHeld()
		return append(evs, Event{Kind: EvAgeStart, Age: st.Age})
	}
	st.Phase = PhaseOver
	st.Decider = 0
	g.score()
	return append(evs, Event{Kind: EvGameOver})
}

// score sets the result: the built designs' points, three per tucked card
// and a point per three coins. Ranks go by total, then coins.
func (g *Game) score() {
	st := &g.State
	r := &st.Result
	r.Over = true
	for s := Seat(0); uint8(s) < st.Seats; s++ {
		c := &st.Cities[s]
		var sc Score
		for d := DesignID(1); d <= NumDesigns; d++ {
			if c.built(d) {
				sc.Built += int16(designs[d].Points)
			}
		}
		sc.Stages = 3 * int16(c.Stages)
		sc.Treasure = c.Coins / 3
		sc.Total = sc.Built + sc.Stages + sc.Treasure
		r.Scores[s] = sc
	}
	for s := Seat(0); uint8(s) < st.Seats; s++ {
		rank := uint8(1)
		for t := Seat(0); uint8(t) < st.Seats; t++ {
			if beats(st, t, s) {
				rank++
			}
		}
		r.Rank[s] = rank
		if rank == 1 {
			r.Winners |= 1 << s
		}
	}
}

// beats reports whether seat a ranks strictly above seat b.
func beats(st *State, a, b Seat) bool {
	ta, tb := st.Result.Scores[a].Total, st.Result.Scores[b].Total
	if ta != tb {
		return ta > tb
	}
	return st.Cities[a].Coins > st.Cities[b].Coins
}
