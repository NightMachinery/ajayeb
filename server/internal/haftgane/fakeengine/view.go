package fakeengine

import "github.com/NightMachinery/ajayeb/server/internal/haftgane/fakeengine/fakeview"

// ViewFor builds one viewer's view by whitelist: it copies in what the
// viewer may see and never deletes from a full object. Everyone sees the
// point, each seat's public city and latest public play, the pile's count,
// the latest resolution's events filtered for the viewer, and the result
// once over. A seat also sees its hand, kept and tucked cards, its offered
// keys, the pile's contents during its own salvage point, and its notebook.
// Nobody sees the seed, a fingerprint, another seat's hand, offer or
// private cards, the pile's contents or Age 2's deal and box, until the end
// reveal (off with Options.HideAtEnd). Every card list is sorted in catalog
// order. A viewer whose seat is not in the game sees what a spectator sees.
func (g *Game) ViewFor(v Viewer) fakeview.Table {
	st := &g.State
	p := g.Point()
	t := fakeview.Table{
		Point:    p.ID,
		Kind:     p.Kind.String(),
		Required: seatInts(p.Required),
		Phase:    st.Phase.String(),
		Age:      int(st.Age),
		Turn:     int(st.Turn),
		Pass:     passName(st.Age),
		Seats:    make([]fakeview.Seat, 0, st.Seats),
		Events:   []fakeview.Event{},
	}
	for d := DesignID(1); d <= NumDesigns; d++ {
		t.Pile += int(st.Pile[d])
	}
	for s := Seat(0); uint8(s) < st.Seats; s++ {
		c := &st.Cities[s]
		t.Seats = append(t.Seats, fakeview.Seat{
			Seat:     int(s),
			Board:    c.Board.Slug(),
			Coins:    int(c.Coins),
			Built:    builtSlugs(c.Built),
			Stages:   int(c.Stages),
			HandSize: int(handCount(&st.Hands[s])),
			HasKept:  c.Kept != 0,
			Last:     g.lastPlay(s),
		})
	}
	if seated(st, v) {
		t.You = g.private(v.Seat)
	}
	if raw := g.rawHistory(); len(raw) > 0 {
		t.Events = filterEvents(raw[len(raw)-1], v)
	}
	if st.Result.Over {
		t.Result = result(st)
		if !st.Opts.HideAtEnd {
			t.Reveal = reveal(st)
		}
	}
	return t
}

// History rebuilds the whole game's events for one viewer, one entry per
// resolved point, filtered exactly as they were live. It replays the game
// from its own setup; no events are stored.
func (g *Game) History(v Viewer) []fakeview.HistoryEntry {
	out := []fakeview.HistoryEntry{}
	for k, evs := range g.rawHistory() {
		r := &g.Record.Log[k]
		out = append(out, fakeview.HistoryEntry{Point: pointID(r.Kind, r.Age, r.Turn), Events: filterEvents(evs, v)})
	}
	return out
}

// rawHistory replays the record from setup and returns each resolution's
// unfiltered events.
func (g *Game) rawHistory() [][]Event {
	out := [][]Event{}
	r, err := NewGame(g.setup())
	if err != nil {
		return out
	}
	for k := uint8(0); k < g.Record.NLog; k++ {
		e := &g.Record.Log[k]
		acts := e.Actions
		out = append(out, r.apply(r.Point(), &acts, make([]Event, 0, MaxEvents)))
	}
	return out
}

// seated reports whether the viewer holds a seat of this game.
func seated(st *State, v Viewer) bool { return !v.Spectator && uint8(v.Seat) < st.Seats }

// private builds a seat's own section.
func (g *Game) private(s Seat) *fakeview.Private {
	st := &g.State
	c := &st.Cities[s]
	p := &fakeview.Private{
		Seat:    int(s),
		Hand:    designSlugs(st.Hands[s][:]),
		Kept:    c.Kept.Slug(),
		Tucked:  designSlugs(c.Tucked[:]),
		Choices: []fakeview.Choice{},
		Offer:   []fakeview.OfferEntry{},
		Seen:    []fakeview.SeenEntry{},
	}
	for _, ch := range g.Legal(s) {
		p.Choices = append(p.Choices, fakeview.Choice{
			Key: ch.Key, Kind: ch.Kind.String(), Design: ch.Design.Slug(), Left: int(ch.Left), Right: int(ch.Right),
		})
	}
	if st.Phase == PhaseSalvage && st.Decider == s {
		for d := DesignID(1); d <= NumDesigns; d++ {
			if st.Pile[d] > 0 {
				p.Offer = append(p.Offer, fakeview.OfferEntry{Design: d.Slug(), Count: int(st.Pile[d]), Legal: !c.built(d)})
			}
		}
	}
	for k := uint8(0); k < g.Record.NLog; k++ {
		e := &g.Record.Log[k]
		if !e.Required.Has(s) {
			continue
		}
		a := e.Actions[s]
		entry := fakeview.SeenEntry{
			Point: pointID(e.Kind, e.Age, e.Turn), Hand: []string{}, Played: a.Design.Slug(), Action: a.Kind.String(),
		}
		switch e.Kind {
		case PointTurn:
			entry.Hand = designSlugs(g.Record.Held[e.Age-1][e.Turn-1][s][:])
		case PointLastCard:
			entry.Hand = []string{a.Design.Slug()}
		}
		p.Seen = append(p.Seen, entry)
	}
	return p
}

// lastPlay returns a seat's latest play in its public form, or nil.
func (g *Game) lastPlay(s Seat) *fakeview.Play {
	stage := 0
	var last *fakeview.Play
	for k := uint8(0); k < g.Record.NLog; k++ {
		e := &g.Record.Log[k]
		if !e.Required.Has(s) {
			continue
		}
		a := e.Actions[s]
		p := &fakeview.Play{Kind: a.Kind.String(), Left: int(a.Left), Right: int(a.Right)}
		switch a.Kind {
		case ActBuild, ActSalvage:
			p.Design = a.Design.Slug()
		case ActTuck:
			stage++
			p.Stage = stage
		}
		last = p
	}
	return last
}

// filterEvents copies each event's public fields, and the design only where
// the viewer may know it: always for a build or salvage, and for a tuck, a
// sale, a leftover or a kept card only to the seat it belongs to.
func filterEvents(evs []Event, v Viewer) []fakeview.Event {
	out := make([]fakeview.Event, 0, len(evs))
	for _, e := range evs {
		f := fakeview.Event{
			Kind: e.Kind.String(), Seat: int(e.Seat), Play: e.Play.String(),
			Stage: int(e.Stage), Left: int(e.Left), Right: int(e.Right), Age: int(e.Age),
		}
		switch e.Dir {
		case 1:
			f.Pass = "left"
		case 2:
			f.Pass = "right"
		}
		public := e.Kind == EvSalvage || (e.Kind == EvPlay && e.Play == ActBuild)
		own := !v.Spectator && v.Seat == e.Seat
		if public || (own && (e.Kind == EvPlay || e.Kind == EvLeftover || e.Kind == EvKept)) {
			f.Design = e.Design.Slug()
		}
		out = append(out, f)
	}
	return out
}

func result(st *State) *fakeview.Result {
	r := &fakeview.Result{Scores: []fakeview.Score{}, Rank: []int{}, Winners: seatInts(st.Result.Winners)}
	for s := Seat(0); uint8(s) < st.Seats; s++ {
		sc := st.Result.Scores[s]
		r.Scores = append(r.Scores, fakeview.Score{
			Seat: int(s), Built: int(sc.Built), Stages: int(sc.Stages), Treasure: int(sc.Treasure), Total: int(sc.Total),
		})
		r.Rank = append(r.Rank, int(st.Result.Rank[s]))
	}
	return r
}

func reveal(st *State) *fakeview.Reveal {
	r := &fakeview.Reveal{Tucked: [][]string{}, Pile: []string{}, Box: designSlugs(st.Box[:])}
	for s := Seat(0); uint8(s) < st.Seats; s++ {
		r.Tucked = append(r.Tucked, designSlugs(st.Cities[s].Tucked[:]))
	}
	for d := DesignID(1); d <= NumDesigns; d++ {
		for i := uint8(0); i < st.Pile[d]; i++ {
			r.Pile = append(r.Pile, d.Slug())
		}
	}
	return r
}

// designSlugs returns the non-empty designs' slugs in catalog order.
func designSlugs(ds []DesignID) []string {
	var count [NumDesigns + 1]uint8
	for _, d := range ds {
		if d != 0 {
			count[d]++
		}
	}
	out := []string{}
	for d := DesignID(1); d <= NumDesigns; d++ {
		for i := uint8(0); i < count[d]; i++ {
			out = append(out, d.Slug())
		}
	}
	return out
}

func builtSlugs(built uint32) []string {
	out := []string{}
	for d := DesignID(1); d <= NumDesigns; d++ {
		if built&(1<<d) != 0 {
			out = append(out, d.Slug())
		}
	}
	return out
}

func seatInts(ss SeatSet) []int {
	out := []int{}
	for _, s := range ss.Seats() {
		out = append(out, int(s))
	}
	return out
}

func passName(age uint8) string {
	if age == 2 {
		return "right"
	}
	return "left"
}
