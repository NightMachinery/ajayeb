package leakcheck

import (
	"fmt"
	"strconv"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/fakeengine"
)

// FromFake builds a fake game's truth by replaying its batches from setup
// and reading the engine's state, never its views. The secrets are the seed
// (as hex, and as decimal when it has six digits or more), the game's
// fingerprint at every moment and every seat's observation fingerprint at
// every moment.
func FromFake(setup fakeengine.Setup, batches []fakeengine.Batch) (*Truth, error) {
	g, err := fakeengine.NewGame(setup)
	if err != nil {
		return nil, err
	}
	t := &Truth{Seats: int(setup.Seats)}
	for _, d := range fakeengine.Designs() {
		t.Slugs = append(t.Slugs, d.Slug)
	}
	t.Secrets = append(t.Secrets, Secret{"seed", fmt.Sprintf("%016x", setup.Seed)})
	if dec := strconv.FormatUint(setup.Seed, 10); len(dec) >= 6 {
		t.Secrets = append(t.Secrets, Secret{"seed", dec})
	}
	for k, b := range batches {
		t.add(&g)
		p := g.Point()
		res := Resolution{
			Point: p.ID, Kind: p.Kind.String(), Required: seatList(p.Required),
			Held: make([][]string, t.Seats), Actions: make([]Action, t.Seats), Events: []Event{},
		}
		for _, s := range p.Required.Seats() {
			switch p.Kind {
			case fakeengine.PointTurn:
				res.Held[s] = slugs(g.State.Hands[s][:])
			case fakeengine.PointLastCard:
				res.Held[s] = slugs([]fakeengine.DesignID{g.State.Cities[s].Kept})
			default:
				res.Held[s] = []string{}
			}
		}
		next, evs, err := g.Resolve(b)
		if err != nil {
			return nil, fmt.Errorf("batch %d: %w", k, err)
		}
		for _, pk := range b.Picks {
			a, _ := fakeengine.ParseKey(pk.Key)
			res.Actions[pk.Seat] = Action{Kind: a.Kind.String(), Design: a.Design.Slug(), Left: int(a.Left), Right: int(a.Right)}
		}
		for _, e := range evs {
			ev := Event{
				Kind: e.Kind.String(), Seat: int(e.Seat), Play: e.Play.String(), Design: e.Design.Slug(),
				Stage: int(e.Stage), Left: int(e.Left), Right: int(e.Right), Age: int(e.Age),
			}
			switch e.Dir {
			case 1:
				ev.Pass = "left"
			case 2:
				ev.Pass = "right"
			}
			res.Events = append(res.Events, ev)
		}
		t.Resolutions = append(t.Resolutions, res)
		g = next
	}
	t.add(&g)
	return t, nil
}

// add appends the moment of g and its secrets.
func (t *Truth) add(g *fakeengine.Game) {
	st := &g.State
	p := g.Point()
	m := Moment{
		Point: p.ID, Kind: p.Kind.String(), Required: seatList(p.Required), Phase: st.Phase.String(),
		Age: int(st.Age), Turn: int(st.Turn), Over: st.Result.Over, Reveal: st.Result.Over && !st.Opts.HideAtEnd,
		Decider: -1, Pile: []string{}, Box: slugs(st.Box[:]),
	}
	if st.Phase == fakeengine.PhaseSalvage {
		m.Decider = int(st.Decider)
	}
	for d := 1; d <= fakeengine.NumDesigns; d++ {
		for range st.Pile[d] {
			m.Pile = append(m.Pile, fakeengine.DesignID(d).Slug())
		}
	}
	t.Secrets = append(t.Secrets, Secret{fmt.Sprintf("game fingerprint at moment %d", len(t.Moments)), g.Fingerprint()})
	for s := 0; s < t.Seats; s++ {
		c := &st.Cities[s]
		m.Coins = append(m.Coins, int(c.Coins))
		m.Stages = append(m.Stages, int(c.Stages))
		m.Hands = append(m.Hands, slugs(st.Hands[s][:]))
		m.Kept = append(m.Kept, c.Kept.Slug())
		m.Tucked = append(m.Tucked, slugs(c.Tucked[:]))
		var built []fakeengine.DesignID
		for d := 1; d <= fakeengine.NumDesigns; d++ {
			if c.Built&(1<<d) != 0 {
				built = append(built, fakeengine.DesignID(d))
			}
		}
		m.Built = append(m.Built, slugs(built))
		keys := []string{}
		for _, ch := range g.Legal(fakeengine.Seat(s)) {
			keys = append(keys, ch.Key)
		}
		m.Legal = append(m.Legal, keys)
		o := g.Observe(fakeengine.Seat(s))
		t.Secrets = append(t.Secrets, Secret{fmt.Sprintf("seat %d's observation hash at moment %d", s, len(t.Moments)), o.Fingerprint()})
	}
	t.Moments = append(t.Moments, m)
}

// slugs returns the non-empty designs' slugs in catalog order.
func slugs(ds []fakeengine.DesignID) []string {
	sorted := make([]fakeengine.DesignID, 0, len(ds))
	for _, d := range ds {
		if d != 0 {
			sorted = append(sorted, d)
		}
	}
	for i := 1; i < len(sorted); i++ {
		for j := i; j > 0 && sorted[j] < sorted[j-1]; j-- {
			sorted[j], sorted[j-1] = sorted[j-1], sorted[j]
		}
	}
	out := []string{}
	for _, d := range sorted {
		out = append(out, d.Slug())
	}
	return out
}

func seatList(ss fakeengine.SeatSet) []int {
	out := []int{}
	for _, s := range ss.Seats() {
		out = append(out, int(s))
	}
	return out
}
