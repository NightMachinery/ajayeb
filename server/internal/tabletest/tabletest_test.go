package tabletest_test

import (
	"bytes"
	"context"
	"math/rand/v2"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/fakeengine"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/direct"
)

func open(seats uint8, seed uint64, seqClients int) tabletest.Factory {
	return direct.Factory(direct.Options{Setup: fakeengine.Setup{Seed: seed, Seats: seats}, SeqClients: seqClients})
}

func play(t *testing.T, f tabletest.Factory, cfg tabletest.Config) tabletest.Run {
	t.Helper()
	tb, err := f(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	r, err := tabletest.Play(t.Context(), tb, cfg)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

var busy = tabletest.ScriptOptions{Drafts: 50, Resends: 40}

// TestSameSeedsSameSchedule: the same seeds give the same submissions in
// the same order, the same outcomes and the same canonical history.
func TestSameSeedsSameSchedule(t *testing.T) {
	for n := 3; n <= 7; n++ {
		cfg := tabletest.Config{Seats: n, PickSeed: 11, ArrivalSeed: 12, Script: busy}
		a := play(t, open(uint8(n), 5, 2), cfg)
		b := play(t, open(uint8(n), 5, 2), cfg)
		if !bytes.Equal(a.TraceBytes(), b.TraceBytes()) || !bytes.Equal(a.Canonical.Bytes(), b.Canonical.Bytes()) {
			t.Fatalf("%d seats: the same seeds gave two schedules", n)
		}
		cfg.ArrivalSeed = 13
		c := play(t, open(uint8(n), 5, 2), cfg)
		if bytes.Equal(a.TraceBytes(), c.TraceBytes()) {
			t.Fatalf("%d seats: another arrival seed gave the same trace", n)
		}
	}
}

// TestArrivalOrderDecidesNothing plays each game under many arrival seeds,
// with drafts, re-sent seqs and a seq memory of two clients (so stale seqs
// occur), and requires one canonical history, which replays from setup to
// the same hash. It also checks that the sweep met every outcome the rules
// allow a well-behaved client to meet.
func TestArrivalOrderDecidesNothing(t *testing.T) {
	seen := map[tabletest.Outcome]int{}
	for n := 3; n <= 7; n++ {
		for game := uint64(0); game < 6; game++ {
			f := open(uint8(n), 100*uint64(n)+game, 2)
			cfg := tabletest.Config{Seats: n, PickSeed: game, Script: busy}
			runs, err := tabletest.CompareSchedules(t.Context(), f, cfg, []uint64{1, 2, 3, 4, 5, 6, 7, 8})
			if err != nil {
				t.Fatalf("%d seats, game %d: %v", n, game, err)
			}
			c := runs[0].Canonical
			g, err := fakeengine.Replay(fakeengine.Setup{Seed: 100*uint64(n) + game, Seats: uint8(n)}, engineBatches(c.Batches))
			if err != nil || g.Fingerprint() != c.Hash || g.Point().Kind != fakeengine.PointNone {
				t.Fatalf("%d seats, game %d: the canonical history does not replay to its hash: %v", n, game, err)
			}
			for _, r := range runs {
				for _, a := range r.Trace {
					seen[a.Outcome]++
				}
			}
		}
	}
	for _, o := range []tabletest.Outcome{tabletest.Accepted, tabletest.Resolved, tabletest.Duplicate, tabletest.Stale, tabletest.Closed} {
		if seen[o] == 0 {
			t.Errorf("no schedule met outcome %s: %v", o, seen)
		}
	}
}

// TestEveryArrivalOrder replays small points under every arrival order
// that keeps each client's own order: the first turn at three and four
// seats with every seat changing its pick once, the same with re-sent seqs,
// and a one-seat point with re-sent seqs. Every order gives one batch.
func TestEveryArrivalOrder(t *testing.T) {
	cases := []struct {
		name  string
		seats int
		seed  uint64
		opts  tabletest.ScriptOptions
		min   int
	}{
		{"three seats, drafts", 3, 1, tabletest.ScriptOptions{Drafts: 100}, 90},
		{"four seats, drafts", 4, 2, tabletest.ScriptOptions{Drafts: 100}, 2520},
		{"three seats, drafts and resends", 3, 3, tabletest.ScriptOptions{Drafts: 100, Resends: 100}, 90},
		{"one-seat point, resends", 5, 0, tabletest.ScriptOptions{Drafts: 100, Resends: 100}, 1},
	}
	for _, c := range cases {
		seed, prefix := c.seed, []tabletest.Batch(nil)
		if strings.HasPrefix(c.name, "one-seat") {
			seed, prefix = oneSeatPrefix(t, uint8(c.seats))
		}
		for _, seqClients := range []int{32, 1} {
			f := open(uint8(c.seats), seed, seqClients)
			cfg := tabletest.Config{Seats: c.seats, PickSeed: 9, Script: c.opts}
			n, distinct, err := tabletest.ExhaustPoint(t.Context(), f, prefix, cfg, 200000)
			if err != nil {
				t.Fatalf("%s, seq memory %d: %v", c.name, seqClients, err)
			}
			if n < c.min || len(distinct) != 1 {
				t.Fatalf("%s, seq memory %d: %d orders, %d distinct batches", c.name, seqClients, n, len(distinct))
			}
		}
	}
}

// TestRacesFollowArrival: when seats confirm and then change or withdraw,
// whether the change lands depends on arrival, so the orders of one point
// give more than one batch, and every one of them still passes the checks.
func TestRacesFollowArrival(t *testing.T) {
	most := 0
	for pick := uint64(1); pick <= 8; pick++ {
		cfg := tabletest.Config{Seats: 3, PickSeed: pick, Script: tabletest.ScriptOptions{Race: true}}
		n, distinct, err := tabletest.ExhaustPoint(t.Context(), open(3, 4, 32), nil, cfg, 200000)
		if err != nil {
			t.Fatal(err)
		}
		if n != 1680 {
			t.Fatalf("%d orders, want 9!/(3!)^3 = 1680", n)
		}
		most = max(most, len(distinct))
	}
	if most < 2 {
		t.Fatal("no race changed the batch")
	}
	for _, seed := range []uint64{1, 2, 3} {
		play(t, open(5, 7, 32), tabletest.Config{Seats: 5, PickSeed: 3, ArrivalSeed: seed, Script: tabletest.ScriptOptions{Race: true, Resends: 30}})
	}
}

// lying wraps a table and falsifies what it reports, to prove the checks
// can fail.
type lying struct {
	tabletest.Table
	outcome func(n int, s tabletest.Submission, o tabletest.Outcome) tabletest.Outcome
	canon   func(c tabletest.Canonical) tabletest.Canonical
	n       int
}

func (l *lying) Submit(ctx context.Context, c tabletest.ClientID, s tabletest.Submission) (tabletest.Outcome, error) {
	o, err := l.Table.Submit(ctx, c, s)
	l.n++
	if l.outcome != nil && err == nil {
		o = l.outcome(l.n, s, o)
	}
	return o, err
}

func (l *lying) Canonical(ctx context.Context) (tabletest.Canonical, error) {
	c, err := l.Table.Canonical(ctx)
	if l.canon != nil {
		c = l.canon(c)
	}
	return c, err
}

func TestChecksCatchALyingTable(t *testing.T) {
	cases := map[string]*lying{
		"a duplicate reported as accepted": {outcome: func(_ int, _ tabletest.Submission, o tabletest.Outcome) tabletest.Outcome {
			if o == tabletest.Duplicate {
				return tabletest.Accepted
			}
			return o
		}},
		"a stale seq reported as accepted": {outcome: func(_ int, _ tabletest.Submission, o tabletest.Outcome) tabletest.Outcome {
			if o == tabletest.Stale {
				return tabletest.Accepted
			}
			return o
		}},
		"a resolution reported as accepted": {outcome: func(_ int, _ tabletest.Submission, o tabletest.Outcome) tabletest.Outcome {
			if o == tabletest.Resolved {
				return tabletest.Accepted
			}
			return o
		}},
		"an accepted pick reported as a duplicate": {outcome: func(n int, _ tabletest.Submission, o tabletest.Outcome) tabletest.Outcome {
			if n == 1 {
				return tabletest.Duplicate
			}
			return o
		}},
		"an open point reported as closed": {outcome: func(n int, s tabletest.Submission, o tabletest.Outcome) tabletest.Outcome {
			if o == tabletest.Accepted && !s.Confirm {
				return tabletest.Closed
			}
			return o
		}},
		"a canonical batch that differs from the trace": {canon: func(c tabletest.Canonical) tabletest.Canonical {
			c.Batches = append([]tabletest.Batch{}, c.Batches...)
			c.Batches[0] = tabletest.Batch{Point: c.Batches[0].Point, Picks: append([]tabletest.Pick{}, c.Batches[0].Picks...)}
			c.Batches[0].Picks[0].Key = "sell:nothing"
			return c
		}},
	}
	for name, l := range cases {
		tb, err := open(4, 21, 1)(t.Context(), nil)
		if err != nil {
			t.Fatal(err)
		}
		l.Table = tb
		if _, err := tabletest.Play(t.Context(), l, tabletest.Config{Seats: 4, PickSeed: 1, ArrivalSeed: 1, Script: busy}); err == nil {
			t.Errorf("%s: Play passed", name)
		}
	}
}

func TestCountInterleavings(t *testing.T) {
	scripts := []tabletest.Script{{Ops: make([]tabletest.Op, 2)}, {Ops: make([]tabletest.Op, 2)}, {Ops: make([]tabletest.Op, 2)}}
	if n := tabletest.CountInterleavings(scripts, 1000); n != 90 {
		t.Fatalf("CountInterleavings = %d, want 90", n)
	}
	seen := 0
	tabletest.Interleavings(scripts, func([]tabletest.Op) bool { seen++; return true })
	if seen != 90 {
		t.Fatalf("Interleavings visited %d, want 90", seen)
	}
	if n := tabletest.CountInterleavings(scripts, 10); n != 11 {
		t.Fatalf("CountInterleavings over the limit = %d, want 11", n)
	}
}

// oneSeatPrefix finds a seeded random game that reaches a one-seat point and
// returns its seed and the batches before that point.
func oneSeatPrefix(t *testing.T, seats uint8) (uint64, []tabletest.Batch) {
	t.Helper()
	for seed := uint64(0); seed < 500; seed++ {
		g, _ := fakeengine.NewGame(fakeengine.Setup{Seed: seed, Seats: seats})
		rng := rand.New(rand.NewPCG(seed, 2))
		var prefix []tabletest.Batch
		for g.Point().Kind != fakeengine.PointNone {
			p := g.Point()
			if p.Kind == fakeengine.PointSalvage {
				return seed, prefix
			}
			eb := fakeengine.Batch{Point: p.ID}
			b := tabletest.Batch{Point: p.ID}
			for _, s := range p.Required.Seats() {
				cs := g.Legal(s)
				k := cs[rng.IntN(len(cs))].Key
				eb.Picks = append(eb.Picks, fakeengine.Pick{Seat: s, Key: k})
				b.Picks = append(b.Picks, tabletest.Pick{Seat: int(s), Key: k})
			}
			g, _, _ = g.Resolve(eb)
			prefix = append(prefix, b)
		}
	}
	t.Fatal("no game reached a salvage point")
	return 0, nil
}

func engineBatches(bs []tabletest.Batch) []fakeengine.Batch {
	var out []fakeengine.Batch
	for _, b := range bs {
		eb := fakeengine.Batch{Point: b.Point}
		for _, p := range b.Picks {
			eb.Picks = append(eb.Picks, fakeengine.Pick{Seat: fakeengine.Seat(p.Seat), Key: p.Key})
		}
		out = append(out, eb)
	}
	return out
}
