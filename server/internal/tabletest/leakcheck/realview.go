package leakcheck

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

func (c *Checker) checkRealView(r *report, d Doc) {
	var v realTable
	if err := strict(d.Data, &v); err != nil {
		r.add(UnknownField, "", err.Error())
		return
	}
	// Keep the existing detectors for hands, cards, key membership/order,
	// history, event redaction and the raw frame. Normalize only after the
	// entire richer document has passed its own strict whitelist.
	old := wTable{Point: v.Point, Kind: v.Kind, Required: v.Required, Phase: v.Phase, Age: v.Age, Turn: v.Turn, Pass: v.Pass, Pile: v.Pile, Events: v.Events}
	for _, s := range v.Seats {
		city := wSeat{Seat: s.Seat, Board: s.Board, Coins: s.Coins, Built: s.Built, Stages: s.Stages, HandSize: s.HandSize, HasKept: s.HasKept}
		if s.Last != nil {
			city.Last = &wPlay{Kind: s.Last.Kind, Design: s.Last.Design, Stage: s.Last.Stage, Left: s.Last.Left, Right: s.Last.Right}
		}
		old.Seats = append(old.Seats, city)
	}
	if y := v.You; y != nil {
		p := &wPrivate{Seat: y.Seat, Hand: y.Hand, Kept: y.Kept, Tucked: y.Tucked}
		for _, ch := range y.Choices {
			p.Choices = append(p.Choices, wChoice{Key: ch.Key, Kind: ch.Kind, Design: ch.Design, Left: ch.Left, Right: ch.Right})
		}
		for _, o := range y.Offer {
			p.Offer = append(p.Offer, wOffer(o))
		}
		for _, s := range y.Notebook.Seen {
			p.Seen = append(p.Seen, wSeen(s))
		}
		old.You = p
	}
	if v.Result != nil {
		old.Result = &wResult{Rank: v.Result.Rank, Winners: v.Result.Winners}
		for i, s := range v.Result.Scores {
			old.Result.Scores = append(old.Result.Scores, wScore{Seat: i, Treasure: s.Treasure, Total: s.Total})
		}
	}
	if v.Reveal != nil {
		old.Reveal = &wReveal{Tucked: v.Reveal.Tucked, Pile: v.Reveal.Pile, Box: v.Reveal.Box}
	}
	data, _ := json.Marshal(old)
	nd := d
	nd.Data = data
	c.checkLegacyView(r, nd)
	c.realExtras(r, d, &v)
}
func (c *Checker) realExtras(r *report, d Doc, v *realTable) {
	g := &c.Truth.RealGames[d.Moment]
	st := &g.State
	same := func(path string, a, b any) {
		if !reflect.DeepEqual(a, b) {
			r.add(Mismatch, path, fmt.Sprintf("%v, want %v", a, b))
		}
	}
	same("canDownload", v.CanDownload, c.Truth.Moments[d.Moment].Reveal)
	observer := rules.NoSeat
	if d.Viewer >= 0 {
		observer = rules.Seat(d.Viewer)
	}
	obs := c.Truth.RealObservations[d.Moment][d.Viewer+1]
	for i, s := range v.Seats {
		if i >= int(st.Seats) {
			continue
		}
		city := st.Cities[i]
		same(fmt.Sprintf("seats[%d].board", i), s.Board, city.Board.String())
		same(fmt.Sprintf("seats[%d].side", i), s.Side, city.Side.String())
		relative, relation := i, "spectator"
		if observer.Valid(st.Seats) {
			relative = (i - d.Viewer + int(st.Seats)) % int(st.Seats)
			relation = "other"
			switch rules.Seat(i) {
			case observer:
				relation = "self"
			case observer.Left(st.Seats):
				relation = "left"
			case observer.Right(st.Seats):
				relation = "right"
			}
		}
		same("relative", s.Relative, relative)
		same("relation", s.Relation, relation)
		same("victories", s.Victories, []int{int(city.Victories[0]), int(city.Victories[1]), int(city.Victories[2])})
		same("defeats", s.Defeats, int(city.Defeats))
		same("shields", s.Shields, int(city.Shields))
		facts := c.Truth.RealPublicFacts[d.Moment][i]
		same("score", s.Score, scoreValue(facts.Score))
		same("science", s.Science, c.Truth.realScience[d.Moment][i])
		// No scalar string field above can be used as an uninspected card holder.
		if s.Last != nil {
			printed := 0
			for _, res := range c.Truth.Resolutions[:d.Moment] {
				for _, e := range res.Events {
					if e.Seat == i && (e.Kind == "play" || e.Kind == "discard-build") {
						printed = e.Printed
					}
				}
			}
			same("last.printed", s.Last.Printed, printed)
		}
	}
	if v.Result != nil {
		want := []realScore{}
		rank := []int{}
		winners := []int{}
		for s := 0; s < int(st.Seats); s++ {
			want = append(want, scoreValue(st.Result.Scores[s]))
			rank = append(rank, int(st.Result.Rank[s]))
			if st.Result.Winners&(1<<s) != 0 {
				winners = append(winners, s)
			}
		}
		same("result.scores", v.Result.Scores, want)
		same("result.rank", v.Result.Rank, rank)
		same("result.winners", v.Result.Winners, winners)
	}
	if v.Reveal != nil {
		left := []realLeftover{}
		for _, res := range c.Truth.Resolutions[:d.Moment] {
			for _, e := range res.Events {
				if e.Kind == "leftover-discarded" || e.Kind == "card-kept" {
					// Age is the age of the point, not a hidden record index.
					age := 0
					_, _ = fmt.Sscanf(res.Point, "age-%d-", &age)
					left = append(left, realLeftover{Age: age, Seat: e.Seat, Design: e.Design, Kept: e.Kind == "card-kept"})
				}
			}
		}
		same("reveal.leftovers", v.Reveal.Leftovers, left)
	}
	if v.You == nil || d.Viewer < 0 || v.You.Seat != d.Viewer {
		return
	}
	y := v.You
	choices := c.Truth.RealChoices[d.Moment][d.Viewer]
	for i, ch := range y.Choices {
		if i >= len(choices) {
			break
		}
		w := choices[i]
		same("choice.side", ch.Side, func() string {
			if w.Kind == rules.ActSide {
				return w.Side.String()
			}
			return ""
		}())
		same("choice.stage", ch.Stage, int(w.Stage))
		same("choice.printed", ch.Printed, int(w.Printed))
		same("choice.total", ch.Total, int(w.Total))
		gain := 0
		if w.Kind == rules.ActSell {
			gain = 3
		}
		same("choice.gain", ch.Gain, gain)
		same("choice.free", ch.Free, [...]string{"", "chain", "first-colour", "first-turn", "last-turn", "discard"}[w.Free])
		witness := []realPaymentUnit{}
		for _, u := range w.Payment.Witness[:w.Payment.Units] {
			witness = append(witness, realPaymentUnit{Resource: u.Resource.String(), Source: [...]string{"", "own-fixed", "own-choice", "left-fixed", "left-choice", "right-fixed", "right-choice"}[u.Source], Producer: int(u.Producer), Price: int(u.Price)})
		}
		same("choice.witness", ch.Witness, witness)
	}
	allowed := c.Truth.Moments[d.Moment].Hands[d.Viewer]
	if st.Phase == rules.PhaseLastCard {
		allowed = []string{c.Truth.Moments[d.Moment].Kept[d.Viewer]}
	}
	for _, x := range y.Disabled {
		if !slices.Contains(allowed, x.Design) {
			r.add(HiddenCard, "disabled.design", "not a card held by this seat")
		}
		if !slices.Contains([]string{"build", "stage"}, x.Kind) || !slices.Contains([]string{"already-built", "wonder-complete", "cannot-cover", "insufficient-coins"}, x.Reason) {
			r.add(Mismatch, "disabled", "unknown kind or reason")
		}
	}
	same("disabled", y.Disabled, c.Truth.realDisabled[d.Moment][d.Viewer])
	// Memory and inference use the separately reviewed permitted observation,
	// not ViewFor's DTO builder or its output. Current private cards and seen
	// hands above are independently checked from deals and accepted batches.
	looks := []realPileLook{}
	for _, l := range obs.Know.PileLooks {
		if l.Age == 0 {
			continue
		}
		x := realPileLook{Point: (rules.Point{Kind: rules.PointDiscardBuild, Age: l.Age, Turn: l.Turn}).ID(), Cards: []realDesignCount{}}
		for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
			if l.Count[id] > 0 {
				x.Cards = append(x.Cards, realDesignCount{Design: design(id), Count: int(l.Count[id])})
			}
		}
		looks = append(looks, x)
	}
	same("notebook.pileLooks", y.Notebook.PileLooks, looks)
	stats := c.Truth.RealStatistics[d.Moment][d.Viewer]
	wantStats := realStatistics{Circulating: []realColourBound{}, Returning: []realDesignBound{}, NeighbourPicks: []realKnownPick{}}
	for color := catalog.Brown; color < catalog.NumColors; color++ {
		b := stats.Circulating[color]
		wantStats.Circulating = append(wantStats.Circulating, realColourBound{Colour: color.String(), Bounds: realBounds{Certain: int(b.Certain), AtMost: int(b.AtMost)}})
	}
	for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
		b := stats.Returning[id]
		if b.AtMost > 0 {
			wantStats.Returning = append(wantStats.Returning, realDesignBound{Design: design(id), Bounds: realBounds{Certain: int(b.Certain), AtMost: int(b.AtMost)}})
		}
	}
	for _, p := range stats.Deductions[:stats.NumDeductions] {
		wantStats.NeighbourPicks = append(wantStats.NeighbourPicks, realKnownPick{Point: (rules.Point{Kind: rules.PointTurn, Age: p.Age, Turn: p.Turn}).ID(), Seat: int(p.Seat), Design: design(p.Design)})
	}
	same("notebook.statistics", y.Notebook.Statistics, wantStats)
	same("neighbours", y.Neighbours, c.Truth.realNeighbours[d.Moment][d.Viewer])

}
func scoreValue(s rules.Score) realScore {
	return realScore{Wonder: int(s.Wonder), Treasure: int(s.Treasure), Military: int(s.Military), Blue: int(s.Blue), Yellow: int(s.Yellow), Green: int(s.Green), Purple: int(s.Purple), Total: int(s.Total)}
}

// Science assignments use only public symbols, with the documented compass,
// then gear tie break. Enumerate the small choice multiset independently.
func scienceValue(fixed [catalog.NumSymbols]uint8, wild uint8) realScience {
	best := realScience{Points: -1}
	picked := [3]int{}
	var walk func(int)
	walk = func(left int) {
		if left > 0 {
			for i := range 3 {
				picked[i]++
				walk(left - 1)
				picked[i]--
			}
			return
		}
		counts := []int{int(fixed[catalog.Compass]) + picked[0], int(fixed[catalog.Gear]) + picked[1], int(fixed[catalog.Tablet]) + picked[2]}
		score := counts[0]*counts[0] + counts[1]*counts[1] + counts[2]*counts[2] + 7*min(counts[0], counts[1], counts[2])
		if score > best.Points || (score == best.Points && (picked[0] > best.Choices[0] || (picked[0] == best.Choices[0] && picked[1] > best.Choices[1]))) {
			best = realScience{Counts: counts, Choices: []int{picked[0], picked[1], picked[2]}, Points: score}
		}
	}
	walk(int(wild))
	return best
}
func publicNeighbours(st *rules.State, seat rules.Seat) []realNeighbour {
	out := []realNeighbour{}
	for direction, other := range [2]rules.Seat{seat.Left(st.Seats), seat.Right(st.Seats)} {
		c := st.Cities[other]
		mine := st.Cities[seat]
		n := realNeighbour{Seat: int(other), Direction: [2]string{"left", "right"}[direction], Units: []realTradeUnit{}}
		if st.Age > 0 {
			if mine.Shields > c.Shields {
				n.Outlook = [3]int{1, 3, 5}[st.Age-1]
			} else if mine.Shields < c.Shields {
				n.Outlook = -1
			}
		}
		price := func(resource catalog.Resource) realTradePrice {
			p := realTradePrice{Resource: resource.String(), Price: 2}
			if !mine.Discount[direction].Has(resource) {
				return p
			}
			p.Price = 1
			scope := [2]catalog.SeatScope{catalog.Left, catalog.Right}[direction]
			match := func(e catalog.Effect) bool {
				return e.Kind == catalog.EffTradeDiscount && e.Scope&scope != 0 && e.Mask.Has(resource)
			}
			for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
				if mine.Buildings.Has(id) {
					d, _ := catalog.DesignByID(id)
					for _, e := range d.Effects {
						if match(e) {
							p.Discount = &realDiscount{Design: d.Slug}
							return p
						}
					}
				}
			}
			if mine.Side != catalog.NoSide {
				b, _ := catalog.BoardByID(mine.Board)
				for i := uint8(0); i < mine.Stages; i++ {
					for _, e := range b.Sides[mine.Side-1].Stages[i].Effects {
						if match(e) {
							p.Discount = &realDiscount{Stage: int(i + 1)}
							return p
						}
					}
				}
			}
			return p
		}
		for resource := catalog.Resource(0); resource < catalog.NumResources; resource++ {
			if c.ForSale[resource] > 0 {
				n.Units = append(n.Units, realTradeUnit{Count: int(c.ForSale[resource]), Options: []realTradePrice{price(resource)}})
			}
		}
		for i, mask := range c.ChoiceForSale[:c.NChoiceForSale] {
			u := realTradeUnit{Choice: true, Producer: i, Count: 1, Options: []realTradePrice{}}
			for resource := catalog.Resource(0); resource < catalog.NumResources; resource++ {
				if mask.Has(resource) {
					u.Options = append(u.Options, price(resource))
				}
			}
			n.Units = append(n.Units, u)
		}
		out = append(out, n)
	}
	return out
}

// Disabled explanations are reconstructed from held cards, public city facts
// and the cost solver, never the production view. Offered constructions also
// account for free-build abilities and chains without copying their policy.
func publicDisabled(st *rules.State, seat rules.Seat, offered []rules.Choice) []realDisabled {
	out := []realDisabled{}
	if st.Phase != rules.PhaseTurn && st.Phase != rules.PhaseLastCard {
		return out
	}
	if st.Phase == rules.PhaseLastCard && st.Decider != seat {
		return out
	}
	held := st.Hands[seat]
	if st.Phase == rules.PhaseLastCard {
		held = [rules.HandSize]catalog.DesignID{st.Cities[seat].Kept}
	}
	city := &st.Cities[seat]
	for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
		if !slices.Contains(held[:], id) {
			continue
		}
		for _, kind := range []rules.ActionKind{rules.ActBuild, rules.ActStage} {
			if slices.ContainsFunc(offered, func(ch rules.Choice) bool { return ch.Kind == kind && ch.Design == id }) {
				continue
			}
			d := realDisabled{Kind: actionKind(kind), Design: design(id)}
			card, _ := catalog.DesignByID(id)
			cost := card.Cost
			if kind == rules.ActBuild {
				if !city.Buildings.And(catalog.T.ByName[card.NameID]).Empty() {
					d.Reason = "already-built"
				}
			} else {
				board, ok := catalog.BoardByID(city.Board)
				if !ok || city.Side < catalog.Day || city.Side > catalog.Night || city.Stages >= board.Sides[city.Side-1].NumStages {
					d.Reason = "wonder-complete"
				} else {
					d.Stage = int(city.Stages) + 1
					cost = board.Sides[city.Side-1].Stages[city.Stages].Cost
				}
			}
			if d.Reason == "" {
				status := rules.AssessCostState(st, seat, cost)
				switch {
				case !status.Coverable:
					d.Reason = "cannot-cover"
				case !status.Affordable:
					d.Reason = "insufficient-coins"
					d.Cheapest = int(status.Cheapest)
					d.ShortBy = d.Cheapest - int(city.Coins)
				}
			}
			if d.Reason != "" {
				out = append(out, d)
			}
		}
	}
	return out
}
