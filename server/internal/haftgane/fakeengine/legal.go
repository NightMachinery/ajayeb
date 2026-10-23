package fakeengine

// Point returns the open decision point. Turns require every seat; a
// salvage or last-card point requires its one decider. Once the game is over
// the kind is PointNone, the id empty and no seat required.
func (g *Game) Point() Point {
	st := &g.State
	switch st.Phase {
	case PhaseTurn:
		return Point{ID: pointID(PointTurn, st.Age, st.Turn), Kind: PointTurn, Required: SeatSet(1<<st.Seats - 1)}
	case PhaseSalvage:
		return Point{ID: pointID(PointSalvage, st.Age, st.Turn), Kind: PointSalvage, Required: SeatSet(1 << st.Decider)}
	case PhaseLastCard:
		return Point{ID: pointID(PointLastCard, st.Age, st.Turn), Kind: PointLastCard, Required: SeatSet(1 << st.Decider)}
	}
	return Point{Kind: PointNone}
}

// Legal returns exactly the keys offered to seat s at the open point, in
// canonical order: builds, then tucks, then sales, each by design id, a
// build's payment splits by coins to the left; or the salvage keys by design
// id. A seat that is not required gets an empty list. An open point always
// offers each required seat at least one key: selling is always legal, and
// a salvage point opens only when the pile holds a design the decider may
// build.
func (g *Game) Legal(s Seat) []Choice {
	out := []Choice{}
	if !g.Point().Required.Has(s) {
		return out
	}
	st := &g.State
	c := &st.Cities[s]
	if st.Phase == PhaseSalvage {
		for d := DesignID(1); d <= NumDesigns; d++ {
			if st.Pile[d] > 0 && !c.built(d) {
				out = append(out, choice(Action{Kind: ActSalvage, Design: d}))
			}
		}
		return out
	}
	var cards []DesignID
	if st.Phase == PhaseLastCard {
		cards = []DesignID{c.Kept}
	} else {
		for _, d := range st.Hands[s] {
			if d != 0 && (len(cards) == 0 || cards[len(cards)-1] != d) {
				cards = append(cards, d)
			}
		}
	}
	for _, d := range cards {
		cost := designs[d].Cost
		if c.built(d) || int16(cost) > c.Coins {
			continue
		}
		for l := uint8(0); l <= cost; l++ {
			out = append(out, choice(Action{Kind: ActBuild, Design: d, Left: l, Right: cost - l}))
		}
	}
	if c.Stages < MaxStages {
		for _, d := range cards {
			out = append(out, choice(Action{Kind: ActTuck, Design: d}))
		}
	}
	for _, d := range cards {
		out = append(out, choice(Action{Kind: ActSell, Design: d}))
	}
	return out
}

func choice(a Action) Choice {
	return Choice{Key: a.Key(), Kind: a.Kind, Design: a.Design, Left: a.Left, Right: a.Right}
}

// salvageable reports whether the pile holds a design seat s has not built.
func (st *State) salvageable(s Seat) bool {
	for d := DesignID(1); d <= NumDesigns; d++ {
		if st.Pile[d] > 0 && !st.Cities[s].built(d) {
			return true
		}
	}
	return false
}
