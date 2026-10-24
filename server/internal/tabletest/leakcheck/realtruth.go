package leakcheck

// Real truth is read from the deal, public state and canonical batches,
// never from ViewFor, FilterEvents or History. Its projection checks use a
// separate whitelist from the wire DTOs.
import (
	"fmt"
	"slices"
	"strconv"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules"
)

func design(id catalog.DesignID) string { d, _ := catalog.DesignByID(id); return d.Slug }
func realSlugs(ids []catalog.DesignID) []string {
	is := slices.Clone(ids)
	slices.Sort(is)
	out := []string{}
	for _, id := range is {
		if id.Valid() {
			out = append(out, design(id))
		}
	}
	return out
}
func realSeats(ss rules.SeatSet) []int {
	out := []int{}
	for s := rules.Seat(0); s < rules.MaxSeats; s++ {
		if ss.Has(s) {
			out = append(out, int(s))
		}
	}
	return out
}
func realKind(k rules.PointKind) string {
	return [...]string{"none", "sides", "turn", "discard-build", "last-card"}[k]
}
func actionKind(k rules.ActionKind) string {
	return [...]string{"", "side", "build", "stage", "sell", "build-discarded"}[k]
}
func FromReal(setup rules.Setup, bs []rules.Batch) (*Truth, error) {
	g, err := rules.NewGame(setup)
	if err != nil {
		return nil, err
	}
	t := &Truth{Seats: int(setup.Seats), Real: true}
	for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
		t.Slugs = append(t.Slugs, design(id))
	}
	t.Secrets = append(t.Secrets, Secret{"seed", fmt.Sprintf("%016x", setup.Seed)})
	if v := strconv.FormatUint(setup.Seed, 10); len(v) >= 6 {
		t.Secrets = append(t.Secrets, Secret{"seed", v})
	}
	if setup.Options.Sides != rules.SidesChoose {
		t.Initial = []Event{{Kind: "age-start", Age: 1}}
	}
	for k, b := range bs {
		t.addReal(&g)
		p := g.Point()
		res := Resolution{Point: p.ID(), Kind: realKind(p.Kind), Required: realSeats(p.Required), Held: make([][]string, t.Seats), Actions: make([]Action, t.Seats), Events: []Event{}}
		for _, pk := range b.Picks {
			a, e := rules.ParseKey(pk.Key)
			if e != nil {
				return nil, e
			}
			res.Actions[pk.Seat] = Action{Kind: actionKind(a.Kind), Design: design(a.Design), Left: int(a.Left), Right: int(a.Right)}
			if p.Kind == rules.PointTurn {
				res.Held[pk.Seat] = realSlugs(g.State.Hands[pk.Seat][:])
			}
			if p.Kind == rules.PointLastCard {
				res.Held[pk.Seat] = realSlugs([]catalog.DesignID{g.State.Cities[pk.Seat].Kept})
			}
		}
		n, events, e := g.Resolve(b)
		if e != nil {
			return nil, fmt.Errorf("batch %d: %w", k, e)
		}
		for _, e := range events {
			res.Events = append(res.Events, truthEvent(e, &n.State))
		}
		t.Resolutions = append(t.Resolutions, res)
		g = n
	}
	t.addReal(&g)
	return t, nil
}

// truthEvent copies an engine event's identity. A conflict's shields and
// token worth come from the resolved state instead: shields stand still
// from the comparison to the next Age, and a victory is worth 1, 3 or 5.
func truthEvent(e rules.Event, after *rules.State) Event {
	v := Event{Kind: e.Kind.String()}
	switch e.Kind {
	case rules.EvAgeStart:
		v.Age = int(e.Age)
	case rules.EvHandsPassed:
		v.Age, v.Turn = int(e.Age), int(e.Turn)
		v.Pass = "left"
		if e.Age == 2 {
			v.Pass = "right"
		}
	case rules.EvSideChosen:
		v.Seat, v.Side = int(e.Seat), e.Side.String()
	case rules.EvMilitary:
		v.Seat, v.Other, v.Age = int(e.Seat), int(e.Other), int(e.Age)
		v.Outcome = [...]string{"tie", "seat-wins", "other-wins"}[e.Outcome]
		v.Left, v.Right = int(after.Cities[e.Seat].Shields), int(after.Cities[e.Other].Shields)
		if v.Left != v.Right {
			v.Delta = [...]int{1, 3, 5}[e.Age-1]
		}
	case rules.EvCoins:
		v.Seat, v.Delta = int(e.Seat), int(e.Delta)
		v.Source = [...]string{"", "sale", "design", "stage"}[e.Source]
		if e.Source == rules.CoinDesign {
			v.Design = design(e.Design)
		}
		if e.Source == rules.CoinStage {
			v.Stage = int(e.Stage)
		}
	case rules.EvPlay:
		v.Seat, v.Play, v.Design = int(e.Seat), actionKind(e.Action), design(e.Design)
		if e.Action == rules.ActBuild || e.Action == rules.ActStage {
			v.Left, v.Right, v.Printed = int(e.Left), int(e.Right), int(e.Printed)
		}
		if e.Action == rules.ActStage {
			v.Stage = int(e.Stage)
		}
	case rules.EvLeftoverDiscarded, rules.EvCardKept, rules.EvDiscardBuild:
		v.Seat, v.Design = int(e.Seat), design(e.Design)
	case rules.EvDiscardBuildNothing:
		v.Seat = int(e.Seat)
	}
	return v
}
func (t *Truth) addReal(g *rules.Game) {
	st := &g.State
	p := g.Point()
	over := st.Phase == rules.PhaseOver
	m := Moment{Point: p.ID(), Kind: realKind(p.Kind), Required: realSeats(p.Required), Phase: st.Phase.String(), Age: int(st.Age), Turn: int(st.Turn), Over: over, Reveal: over && !st.Opts.HideAtEnd, Decider: -1, Pile: []string{}, Box: realSlugs(st.BoxGuilds[:])}
	if st.Phase == rules.PhaseDiscardBuild {
		m.Decider = int(st.Decider)
	}
	for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
		for range st.Discard[id] {
			m.Pile = append(m.Pile, design(id))
		}
	}
	observations := []rules.Observation{g.Observe(rules.NoSeat)}
	choices := make([][]rules.Choice, t.Seats)
	facts := make([]rules.PublicFacts, t.Seats)
	stats := make([]rules.Statistics, t.Seats)
	neighbours := make([][]realNeighbour, t.Seats)
	science := make([]realScience, t.Seats)
	disabled := make([][]realDisabled, t.Seats)
	for s := 0; s < t.Seats; s++ {
		c := &st.Cities[s]
		m.Coins = append(m.Coins, int(c.Coins))
		m.Stages = append(m.Stages, int(c.Stages))
		m.Hands = append(m.Hands, realSlugs(st.Hands[s][:]))
		m.Kept = append(m.Kept, design(c.Kept))
		m.Tucked = append(m.Tucked, realSlugs(c.Tucked[:]))
		ids := []catalog.DesignID{}
		for id := catalog.DesignID(1); id <= catalog.NumDesigns; id++ {
			if c.Buildings.Has(id) {
				ids = append(ids, id)
			}
		}
		m.Built = append(m.Built, realSlugs(ids))
		keys := []string{}
		choices[s] = g.Legal(rules.Seat(s))
		for _, ch := range choices[s] {
			keys = append(keys, ch.Key())
		}
		m.Legal = append(m.Legal, keys)
		o := g.Observe(rules.Seat(s))
		observations = append(observations, o)
		facts[s], _ = o.Now.Facts(rules.Seat(s))
		stats[s] = o.Statistics()
		neighbours[s] = publicNeighbours(st, rules.Seat(s))
		disabled[s] = publicDisabled(st, rules.Seat(s), choices[s])
		science[s] = scienceValue(facts[s].Science, facts[s].ScienceChoices)
		t.Secrets = append(t.Secrets, Secret{fmt.Sprintf("seat %d's observation hash at moment %d", s, len(t.Moments)), o.Fingerprint()})
	}
	kept := make([][]string, t.Seats)
	for s := 0; s < t.Seats; s++ {
		kept[s] = trialKept(st, rules.Seat(s), choices)
	}
	t.Secrets = append(t.Secrets, Secret{"game fingerprint", g.Fingerprint()})
	t.Moments = append(t.Moments, m)
	t.RealGames = append(t.RealGames, *g)
	t.RealObservations = append(t.RealObservations, observations)
	t.RealChoices = append(t.RealChoices, choices)
	t.RealPublicFacts = append(t.RealPublicFacts, facts)
	t.RealStatistics = append(t.RealStatistics, stats)
	t.realNeighbours = append(t.realNeighbours, neighbours)
	t.realScience = append(t.realScience, science)
	t.realDisabled = append(t.realDisabled, disabled)
	t.realKept = append(t.realKept, kept)
}

// trialKept is, for each choice offered to seat at a sixth turn, the card the
// seat keeps once the turn resolves with that choice (every other seat taking
// its first offer), or "". It resolves the turn, never reading the preview.
func trialKept(st *rules.State, seat rules.Seat, choices [][]rules.Choice) []string {
	out := make([]string, len(choices[seat]))
	if st.Phase != rules.PhaseTurn || st.Turn != rules.TurnsPerAge {
		return out
	}
	var picks [rules.MaxSeats]rules.Action
	for s := range choices {
		if len(choices[s]) == 0 {
			return out
		}
		picks[s] = choices[s][0].Action
	}
	type move struct {
		kind rules.ActionKind
		id   catalog.DesignID
	}
	tried := map[move]string{}
	for i, ch := range choices[seat] {
		m := move{ch.Kind, ch.Design}
		if kept, ok := tried[m]; ok {
			out[i] = kept
			continue
		}
		picks[seat] = ch.Action
		if next, _, err := rules.ResolveState(st, &picks, nil); err == nil && next.Cities[seat].Kept != 0 {
			out[i] = design(next.Cities[seat].Kept)
		}
		tried[m] = out[i]
	}
	return out
}
