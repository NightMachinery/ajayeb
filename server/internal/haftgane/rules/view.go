package rules

import (
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/view"
)

// ViewFor is a whitelist of screen facts. It has no setup root or fingerprint.
// Invalid seats have the spectator entitlement. Download generation belongs to
// the platform; CanDownload is the end-policy permission, not a log payload.
func (g *Game) ViewFor(viewer Seat) view.Table {
	out := view.Table{Required: []int{}, Seats: []view.Seat{}, Events: []view.Event{}}
	if g == nil {
		return out
	}
	current := Project(&g.State, viewer)
	public, ok := current.publicState()
	if !ok {
		return out
	}
	p := g.Point()
	out.Point, out.Kind, out.Phase = p.ID(), pointKindName(p.Kind), current.Phase.String()
	out.Age, out.Turn, out.Pile = int(current.Age), int(current.Turn), int(current.Pile)
	if current.Age > 0 {
		out.Pass = "left"
		if current.Age == 2 {
			out.Pass = "right"
		}
	}
	for s := Seat(0); s < Seat(current.Seats); s++ {
		if p.Required.Has(s) {
			out.Required = append(out.Required, int(s))
		}
	}
	var latest [MaxSeats]view.Play
	var hasLatest [MaxSeats]bool
	var lastBuffer [64]Event
	lastRaw := lastBuffer[:0]
	leftovers := make([]view.Leftover, 0, NumAges*int(current.Seats))
	knowledge, historyErr := g.readKnowledge(viewer, func(before *State, _ *[MaxSeats]Action, _ *State, events []Event) {
		lastRaw = append(lastRaw[:0], events...)
		for _, raw := range events {
			if raw.Kind != EvPlay && raw.Kind != EvDiscardBuild {
				continue
			}
			e, allowed := filterEvent(raw, NoSeat)
			if !allowed {
				continue
			}
			if e.Kind == EvPlay.String() || e.Kind == EvDiscardBuild.String() {
				kind := e.Play
				if e.Kind == EvDiscardBuild.String() {
					kind = "build-discarded"
				}
				hasLatest[e.Seat] = true
				latest[e.Seat] = view.Play{Kind: kind, Design: e.Design, Stage: e.Stage, Left: e.Left, Right: e.Right, Printed: e.Printed}
			}
		}
		if current.Phase == PhaseOver && !current.Opts.HideAtEnd {
			for _, e := range events {
				if e.Kind == EvLeftoverDiscarded || e.Kind == EvCardKept {
					leftovers = append(leftovers, view.Leftover{Age: int(before.Age), Seat: int(e.Seat), Design: designSlug(e.Design), Kept: e.Kind == EvCardKept})
				}
			}
		}
	})
	if historyErr != nil {
		out.Events = []view.Event{}
		hasLatest = [MaxSeats]bool{}
		leftovers = []view.Leftover{}
	}
	if historyErr == nil {
		out.Events = FilterEvents(lastRaw, viewer)
	}
	if current.Points == 0 && current.Opts.Sides != SidesChoose {
		out.Events = FilterEvents([]Event{{Kind: EvAgeStart, Age: 1}}, viewer)
	}
	for seat := Seat(0); seat < Seat(current.Seats); seat++ {
		c := &public.Cities[seat]
		pc := current.Cities[seat]
		board, _ := catalog.BoardByID(c.Board)
		science := scienceScore(c.Science, c.ScienceChoice)
		relative, relation := int(seat), "spectator"
		if viewer.Valid(current.Seats) {
			relative = int((uint8(seat) + current.Seats - uint8(viewer)) % current.Seats)
			relation = "other"
			switch seat {
			case viewer:
				relation = "self"
			case viewer.Left(current.Seats):
				relation = "left"
			case viewer.Right(current.Seats):
				relation = "right"
			}
		}
		var last *view.Play
		if hasLatest[seat] {
			last = &latest[seat]
		}
		out.Seats = append(out.Seats, view.Seat{Seat: int(seat), Board: board.Slug, Side: c.Side.String(), Stages: int(c.Stages), Built: builtSlugs(c.Buildings), Coins: int(c.Coins), Victories: []int{int(c.Victories[0]), int(c.Victories[1]), int(c.Victories[2])}, Defeats: int(c.Defeats), Shields: int(c.Shields), HandSize: int(pc.HandSize), HasKept: pc.HasKept, Relative: relative, Relation: relation, Last: last, Score: scoreDTO(scoreCity(&public, seat)), Science: view.Science{Counts: symbolCounts(science.Counts), Choices: symbolCounts(science.Choices), Points: int(science.Points)}})
	}
	if viewer.Valid(current.Seats) {
		o := Observation{Now: current, Know: knowledge}
		out.You = g.privateView(&o, &public)
	}
	if current.Phase == PhaseOver {
		out.Result = &view.Result{Scores: make([]view.Score, 0, current.Seats), Rank: make([]int, 0, current.Seats), Winners: []int{}}
		for seat := Seat(0); seat < Seat(current.Seats); seat++ {
			out.Result.Scores = append(out.Result.Scores, scoreDTO(current.Result.Scores[seat]))
			out.Result.Rank = append(out.Result.Rank, int(current.Result.Rank[seat]))
			if current.Result.Winners&uint8(seat.Mask()) != 0 {
				out.Result.Winners = append(out.Result.Winners, int(seat))
			}
		}
		out.CanDownload = !current.Opts.HideAtEnd
		if !current.Opts.HideAtEnd {
			r := &view.Reveal{Tucked: make([][]string, 0, current.Seats), Leftovers: leftovers, Pile: []string{}, Box: designSlugs(g.State.BoxGuilds[:])}
			for seat := Seat(0); seat < Seat(current.Seats); seat++ {
				r.Tucked = append(r.Tucked, designSlugs(g.State.Cities[seat].Tucked[:]))
			}
			for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
				for n := uint8(0); n < g.State.Discard[id]; n++ {
					r.Pile = append(r.Pile, designSlug(id))
				}
			}
			out.Reveal = r
		}
	}
	return out
}
func pointKindName(k PointKind) string {
	if k < NumPointKinds {
		return ([NumPointKinds]string{"none", "sides", "turn", "discard-build", "last-card"})[k]
	}
	return ""
}
func designSlugs(ids []catalog.DesignID) []string {
	var counts [catalog.NumDesigns + 1]uint8
	for _, id := range ids {
		if id.Valid() {
			counts[id]++
		}
	}
	out := make([]string, 0, len(ids))
	for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
		for n := uint8(0); n < counts[id]; n++ {
			out = append(out, designSlug(id))
		}
	}
	return out
}
func builtSlugs(set catalog.DesignSet) []string {
	out := []string{}
	for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
		if set.Has(id) {
			out = append(out, designSlug(id))
		}
	}
	return out
}
func symbolCounts(counts [catalog.NumSymbols]uint8) []int {
	return []int{int(counts[catalog.Compass]), int(counts[catalog.Gear]), int(counts[catalog.Tablet])}
}
func scoreDTO(s Score) view.Score {
	return view.Score{Wonder: int(s.Wonder), Treasure: int(s.Treasure), Military: int(s.Military), Blue: int(s.Blue), Yellow: int(s.Yellow), Green: int(s.Green), Purple: int(s.Purple), Total: int(s.Total)}
}
func choiceDTO(c Choice) view.Choice {
	v := view.Choice{Key: c.Key(), Kind: actionName(c.Kind), Design: designSlug(c.Design), Stage: int(c.Stage), Left: int(c.Left), Right: int(c.Right), Printed: int(c.Printed), Total: int(c.Total), Witness: make([]view.PaymentUnit, 0, c.Payment.Units)}
	if c.Kind == ActSide {
		v.Side = c.Side.String()
	}
	if c.Kind == ActSell {
		v.Gain = 3
	}
	if c.Free <= FreeDiscard {
		v.Free = ([6]string{"", "chain", "first-colour", "first-turn", "last-turn", "discard"})[c.Free]
	}
	names := [7]string{"", "own-fixed", "own-choice", "left-fixed", "left-choice", "right-fixed", "right-choice"}
	for _, u := range c.Payment.Witness[:c.Payment.Units] {
		v.Witness = append(v.Witness, view.PaymentUnit{Resource: u.Resource.String(), Source: names[u.Source], Producer: int(u.Producer), Price: int(u.Price)})
	}
	return v
}
func (g *Game) privateView(o *Observation, public *State) *view.Private {
	c := &o.Now
	seat := c.Seat
	v := &view.Private{Seat: int(seat), Hand: designSlugs(c.Hand[:]), Kept: designSlug(c.Kept), Tucked: designSlugs(c.Tucked[:]), Choices: []view.Choice{}, Disabled: []view.Disabled{}, Offer: []view.OfferEntry{}, Notebook: notebookDTO(o), Neighbours: neighbourDTO(public, seat)}
	choices := g.Legal(seat)
	for _, ch := range choices {
		v.Choices = append(v.Choices, choiceDTO(ch))
	}
	if c.Phase == PhaseDiscardBuild && c.Decider == seat {
		legal := discardBuilds(&g.State, seat)
		for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
			if c.Offer[id] > 0 {
				v.Offer = append(v.Offer, view.OfferEntry{Design: designSlug(id), Count: int(c.Offer[id]), Legal: legal.Has(id)})
			}
		}
	}
	if g.Point().Required.Has(seat) && (c.Phase == PhaseTurn || c.Phase == PhaseLastCard) {
		hand := c.Hand
		if c.Phase == PhaseLastCard {
			hand = [HandSize]catalog.DesignID{c.Kept}
		}
		var seen catalog.DesignSet
		for _, id := range hand {
			if !id.Valid() || seen.Has(id) {
				continue
			}
			seen.Add(id)
			for _, kind := range [2]ActionKind{ActBuild, ActStage} {
				d := view.Disabled{Kind: actionName(kind), Design: designSlug(id)}
				terms, ok := constructionFor(&g.State, seat, kind, id)
				if !ok {
					if kind == ActBuild {
						d.Reason = "already-built"
					} else {
						d.Reason = "wonder-complete"
					}
				} else if terms.free == FreeNone {
					d.Stage = int(terms.stage)
					status := AssessCostState(&g.State, seat, terms.cost)
					if !status.Coverable {
						d.Reason = "cannot-cover"
					} else if !status.Affordable {
						d.Reason = "insufficient-coins"
						d.Cheapest = int(status.Cheapest)
						d.ShortBy = int(status.Cheapest) - int(c.Cities[seat].Coins)
					}
				}
				if d.Reason != "" {
					v.Disabled = append(v.Disabled, d)
				}
			}
		}
	}
	return v
}
func notebookDTO(o *Observation) view.Notebook {
	out := view.Notebook{Seen: []view.SeenEntry{}, PileLooks: []view.PileLook{}, Statistics: view.Statistics{Circulating: []view.ColourBound{}, Returning: []view.DesignBound{}, NeighbourPicks: []view.KnownPick{}}}
	k := &o.Know
	for age := uint8(1); age <= NumAges; age++ {
		for turn := uint8(1); turn <= MaxTurns; turn++ {
			id := k.Picks[age-1][turn-1]
			if id == 0 {
				continue
			}
			kind := PointTurn
			if turn == MaxTurns {
				kind = PointLastCard
			}
			out.Seen = append(out.Seen, view.SeenEntry{Point: (Point{Kind: kind, Age: age, Turn: turn}).ID(), Hand: designSlugs(k.Seen[age-1][turn-1][:]), Played: designSlug(id), Action: actionName(k.Plays[age-1][turn-1][o.Now.Seat].Kind)})
		}
	}
	for _, look := range k.PileLooks {
		if look.Age == 0 {
			continue
		}
		v := view.PileLook{Point: (Point{Kind: PointDiscardBuild, Age: look.Age, Turn: look.Turn}).ID(), Cards: []view.DesignCount{}}
		for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
			if look.Count[id] > 0 {
				v.Cards = append(v.Cards, view.DesignCount{Design: designSlug(id), Count: int(look.Count[id])})
			}
		}
		out.PileLooks = append(out.PileLooks, v)
	}
	stats := o.Statistics()
	for colour := catalog.Brown; colour < catalog.NumColors; colour++ {
		b := stats.Circulating[colour]
		out.Statistics.Circulating = append(out.Statistics.Circulating, view.ColourBound{Colour: colour.String(), Bounds: view.Bounds{Certain: int(b.Certain), AtMost: int(b.AtMost)}})
	}
	for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
		b := stats.Returning[id]
		if b.AtMost > 0 {
			out.Statistics.Returning = append(out.Statistics.Returning, view.DesignBound{Design: designSlug(id), Bounds: view.Bounds{Certain: int(b.Certain), AtMost: int(b.AtMost)}})
		}
	}
	for _, pick := range stats.Deductions[:stats.NumDeductions] {
		out.Statistics.NeighbourPicks = append(out.Statistics.NeighbourPicks, view.KnownPick{Point: (Point{Kind: PointTurn, Age: pick.Age, Turn: pick.Turn}).ID(), Seat: int(pick.Seat), Design: designSlug(pick.Design)})
	}
	return out
}
