package tableroom

import (
	"bytes"
	"context"
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/fakeengine"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/direct"
)

// TestDriverInProcess: Play and CompareSchedules pass unchanged over the
// actor at 3 to 7 seats, with drafts, resends and races, and every arrival
// seed gives the same canonical batches and hash.
func TestDriverInProcess(t *testing.T) {
	ctx := testContext(t)
	for n := MinSeats; n <= MaxSeats; n++ {
		f := newFixture(t, Config{})
		cfg := tabletest.Config{Seats: n, PickSeed: uint64(n), Script: tabletest.ScriptOptions{Drafts: 50, Resends: 30}}
		runs, err := tabletest.CompareSchedules(ctx, f.factory(n, uint64(100+n), false), cfg, []uint64{1, 2, 3, 4})
		if err != nil {
			t.Fatalf("%d seats: %v", n, err)
		}
		if len(runs) != 4 || len(runs[0].Canonical.Batches) == 0 {
			t.Fatalf("%d seats: %d runs", n, len(runs))
		}
		race := cfg
		race.Script = tabletest.ScriptOptions{Race: true, Resends: 30}
		race.ArrivalSeed = 5
		tb, err := f.factory(n, uint64(200+n), true)(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tabletest.Play(ctx, tb, race); err != nil {
			t.Fatalf("%d seats, races: %v", n, err)
		}
	}
}

// TestEveryArrivalOrderInProcess: ExhaustPoint over the actor, as over the
// reference model: every arrival order of small points gives one batch.
func TestEveryArrivalOrderInProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("thousands of tables")
	}
	ctx := testContextFor(t, 10*time.Minute)
	cases := []struct {
		name  string
		seats int
		seed  uint64
		opts  tabletest.ScriptOptions
		min   int
	}{
		{"three seats, drafts", 3, 1, tabletest.ScriptOptions{Drafts: 100}, 90},
		{"three seats, drafts and resends", 3, 3, tabletest.ScriptOptions{Drafts: 100, Resends: 100}, 90},
		{"four seats, drafts", 4, 2, tabletest.ScriptOptions{Drafts: 100}, 2520},
		{"one-seat point, resends", 5, 0, tabletest.ScriptOptions{Drafts: 100, Resends: 100}, 1},
	}
	for _, c := range cases {
		seed, prefix := c.seed, []tabletest.Batch(nil)
		if strings.HasPrefix(c.name, "one-seat") {
			seed, prefix = salvagePrefix(t, c.seats)
		}
		for _, seqClients := range []int{32, 1} {
			f := newFixtureOn(t, Config{SeqClients: seqClients}, true)
			cfg := tabletest.Config{Seats: c.seats, PickSeed: 9, Script: c.opts}
			n, distinct, err := tabletest.ExhaustPoint(ctx, f.factory(c.seats, seed, false), prefix, cfg, 200000)
			if err != nil {
				t.Fatalf("%s, seq memory %d: %v", c.name, seqClients, err)
			}
			if n < c.min || len(distinct) != 1 {
				t.Fatalf("%s, seq memory %d: %d orders, %d distinct batches", c.name, seqClients, n, len(distinct))
			}
		}
	}
	// Races: a change or a withdrawn confirmation racing the last
	// confirmation lands or not by arrival alone.
	f := newFixtureOn(t, Config{}, true)
	most := 0
	for pick := uint64(1); pick <= 3; pick++ {
		cfg := tabletest.Config{Seats: 3, PickSeed: pick, Script: tabletest.ScriptOptions{Race: true}}
		n, distinct, err := tabletest.ExhaustPoint(ctx, f.factory(3, 4, false), nil, cfg, 200000)
		if err != nil {
			t.Fatal(err)
		}
		if n != 1680 {
			t.Fatalf("%d orders, want 1680", n)
		}
		most = max(most, len(distinct))
	}
	if most < 2 {
		t.Fatal("no race changed the batch")
	}
}

// salvagePrefix finds a game that reaches a one-seat salvage point and
// returns its seed and the batches before it (chosen seating, as the
// fixture's tables use).
func salvagePrefix(t *testing.T, seats int) (uint64, []tabletest.Batch) {
	t.Helper()
	for seed := uint64(0); seed < 500; seed++ {
		g, _ := newGame(makeSetup(seed, seats, false, false))
		rng := rand.New(rand.NewPCG(seed, 2))
		var prefix []tabletest.Batch
		for !pointOver(g.Point()) {
			p := g.Point()
			if p.Kind == fakeengine.PointSalvage {
				return seed, prefix
			}
			eb := batch{Point: p.ID}
			b := tabletest.Batch{Point: p.ID}
			for _, s := range p.Required.Seats() {
				keys := legalKeys(&g, int(s))
				k := keys[rng.IntN(len(keys))]
				eb.Picks = append(eb.Picks, pick{Seat: s, Key: k})
				b.Picks = append(b.Picks, tabletest.Pick{Seat: int(s), Key: k})
			}
			g, _, _ = g.Resolve(eb)
			prefix = append(prefix, b)
		}
	}
	t.Fatal("no game reached a salvage point")
	return 0, nil
}

// TestDifferentialAgainstDirect: the same seeded scripts, delivered to the
// reference model (tabletest/direct) and to the actor arrival by arrival,
// give identical outcomes, batches and hashes, including with a seq
// memory small enough for evictions and stale seqs.
func TestDifferentialAgainstDirect(t *testing.T) {
	ctx := testContext(t)
	for n := MinSeats; n <= MaxSeats; n++ {
		for _, seqClients := range []int{32, 2} {
			for game := range uint64(3) {
				differential(ctx, t, n, 300+game*7+uint64(n), seqClients, game)
			}
		}
	}
}

func differential(ctx context.Context, t *testing.T, n int, seed uint64, seqClients int, salt uint64) {
	t.Helper()
	where := fmt.Sprintf("%d seats, seed %d, seq memory %d", n, seed, seqClients)
	// direct's setup uses the engine's default (random) seating.
	ref, err := direct.New(ctx, direct.Options{Setup: fakeengine.Setup{Seed: seed, Seats: uint8(n)}, SeqClients: seqClients})
	if err != nil {
		t.Fatal(err)
	}
	f := newFixture(t, Config{SeqClients: seqClients})
	sp := humans(n, seed)
	sp.RandomSeating = true
	act, err := f.openInproc(ctx, sp)
	if err != nil {
		t.Fatal(err)
	}
	clients := make([]tabletest.ClientID, n)
	for s := range clients {
		clients[s] = tabletest.SeatClient(s)
	}
	seqs := tabletest.Seqs{}
	opts := []tabletest.ScriptOptions{{Drafts: 60, Resends: 40}, {Race: true, Resends: 40}, {Drafts: 100}}
	for idx := 0; ; idx++ {
		if idx > 100 {
			t.Fatalf("%s: the game did not end", where)
		}
		dsnaps := make([]tabletest.Snapshot, n)
		asnaps := make([]tabletest.Snapshot, n)
		for s, c := range clients {
			if dsnaps[s], err = ref.Snapshot(ctx, c); err != nil {
				t.Fatal(err)
			}
			if asnaps[s], err = act.Snapshot(ctx, c); err != nil {
				t.Fatal(err)
			}
			d, a := dsnaps[s], asnaps[s]
			if d.Point != a.Point || d.Over != a.Over || d.Seat != a.Seat || fmt.Sprint(d.Required, d.Confirmed, d.Legal) != fmt.Sprint(a.Required, a.Confirmed, a.Legal) {
				t.Fatalf("%s, point %d: snapshots differ:\n direct %+v\n actor  %+v", where, idx, d, a)
			}
		}
		if dsnaps[0].Over {
			break
		}
		scripts, err := tabletest.Scripts(dsnaps, clients, seed*31+uint64(idx)+salt, opts[(idx+int(salt))%len(opts)], seqs)
		if err != nil {
			t.Fatal(err)
		}
		for i, op := range tabletest.Interleave(scripts, seed+uint64(idx)*17) {
			do, err := ref.Submit(ctx, op.Client, op.Sub)
			if err != nil {
				t.Fatal(err)
			}
			sub := op.Sub
			if sub.Ref == dsnaps[0].Ref {
				sub.Ref = asnaps[0].Ref
			}
			ao, err := act.Submit(ctx, op.Client, sub)
			if err != nil {
				t.Fatal(err)
			}
			if do != ao {
				t.Fatalf("%s, point %s, arrival %d (%s seq %d confirm=%t): direct %s, actor %s",
					where, dsnaps[0].Point, i, op.Client, op.Sub.Seq, op.Sub.Confirm, do, ao)
			}
		}
	}
	dc, _ := ref.Canonical(ctx)
	ac, err := act.Canonical(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(dc.Bytes(), ac.Bytes()) {
		t.Fatalf("%s: canonical histories differ:\n%s\n%s", where, dc.Bytes(), ac.Bytes())
	}
}
