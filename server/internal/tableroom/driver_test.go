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
			f.tabs = true
			cfg := tabletest.Config{Seats: c.seats, PickSeed: 9, Script: c.opts}
			cfg.Script.Tabs = true
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

// TestDifferentialAgainstDirect: the same seeded scripts, sent from two
// tabs per seat to the reference model (tabletest/direct) and to the actor
// arrival by arrival, give identical outcomes, batches and hashes,
// including with a seq memory of one client per user, small enough for
// evictions and stale seqs. After every arrival every viewer (each seat's
// two tabs and the spectator) sees the same point, required and confirmed
// seats, offer, pick and confirmation in both.
func TestDifferentialAgainstDirect(t *testing.T) {
	ctx := testContext(t)
	seen := map[tabletest.Outcome]int{}
	for n := MinSeats; n <= MaxSeats; n++ {
		for _, seqClients := range []int{32, 1} {
			for game := range uint64(3) {
				differential(ctx, t, n, 300+game*7+uint64(n), seqClients, game, seen)
			}
		}
	}
	for _, o := range []tabletest.Outcome{tabletest.Accepted, tabletest.Resolved, tabletest.Duplicate, tabletest.Stale, tabletest.Closed} {
		if seen[o] == 0 {
			t.Errorf("no arrival met outcome %s: %v", o, seen)
		}
	}
}

func differential(ctx context.Context, t *testing.T, n int, seed uint64, seqClients int, salt uint64, seen map[tabletest.Outcome]int) {
	t.Helper()
	where := fmt.Sprintf("%d seats, seed %d, seq memory %d", n, seed, seqClients)
	// direct's setup uses the engine's default (random) seating.
	ref, err := direct.New(ctx, direct.Options{Setup: fakeengine.Setup{Seed: seed, Seats: uint8(n)}, SeqClients: seqClients, Tabs: true})
	if err != nil {
		t.Fatal(err)
	}
	f := newFixture(t, Config{SeqClients: seqClients})
	f.tabs = true
	sp := humans(n, seed)
	sp.RandomSeating = true
	act, err := f.openInproc(ctx, sp)
	if err != nil {
		t.Fatal(err)
	}
	clients := make([]tabletest.ClientID, n)
	viewers := []tabletest.ClientID{tabletest.SpectatorClient}
	for s := range clients {
		clients[s] = tabletest.SeatClient(s)
		viewers = append(viewers, tabletest.SeatClient(s), tabletest.TabClient(s))
	}
	seqs := tabletest.Seqs{}
	opts := []tabletest.ScriptOptions{{Drafts: 60, Resends: 40, Tabs: true}, {Race: true, Resends: 40, Tabs: true}, {Drafts: 100, Tabs: true}}
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
		}
		compareViews(ctx, t, fmt.Sprintf("%s, point %d", where, idx), ref, act, viewers, dsnaps[0].Ref, asnaps[0].Ref)
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
			seen[do]++
			compareViews(ctx, t, fmt.Sprintf("%s, point %s, arrival %d", where, dsnaps[0].Point, i), ref, act, viewers, dsnaps[0].Ref, asnaps[0].Ref)
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

// compareViews requires every viewer's snapshot to agree between the
// reference model and the actor: the point and whether it is over, the
// required and confirmed seats, the viewer's seat, offer, pick and
// confirmation, and whether the reference is still the one the point
// opened with (the two number references differently).
func compareViews(ctx context.Context, t *testing.T, where string, ref, act tabletest.Table, viewers []tabletest.ClientID, dRef, aRef string) {
	t.Helper()
	for _, c := range viewers {
		d, err := ref.Snapshot(ctx, c)
		if err != nil {
			t.Fatal(err)
		}
		a, err := act.Snapshot(ctx, c)
		if err != nil {
			t.Fatal(err)
		}
		if d.Point != a.Point || d.Over != a.Over || d.Seat != a.Seat || d.Pick != a.Pick || d.Confirm != a.Confirm ||
			(d.Ref == dRef) != (a.Ref == aRef) || fmt.Sprint(d.Required, d.Confirmed, d.Legal) != fmt.Sprint(a.Required, a.Confirmed, a.Legal) {
			t.Fatalf("%s, %s: snapshots differ:\n direct %+v\n actor  %+v", where, c, d, a)
		}
	}
}

// TestDifferentialSwitches: the same seeded scripts reach the reference
// model and the actor around seat switches, at 3 to 7 seats. At the second
// point the spectator takes seat 1, and the former controller's resends
// and fresh picks are forbidden from then on. At the third, halfway
// through the arrivals, seat 2 goes to a bot, which confirms a kept pick
// (resolving the point when it was the last), so its former clients'
// remaining picks are forbidden or closed; after the arrivals it goes back
// to its human. Every outcome, the switches' included, and every viewer's
// snapshot agree after every arrival, and the canonical histories end
// equal.
func TestDifferentialSwitches(t *testing.T) {
	ctx := testContext(t)
	kept := 0
	for n := MinSeats; n <= MaxSeats; n++ {
		for game := range uint64(2) {
			differentialSwitches(ctx, t, n, 700+game*11+uint64(n), game, &kept)
		}
	}
	if kept == 0 {
		t.Fatal("no bot took over a kept, unconfirmed pick")
	}
}

// differentialSwitches plays one game; kept counts the bot switches that
// found a kept, unconfirmed pick to confirm.
func differentialSwitches(ctx context.Context, t *testing.T, n int, seed, salt uint64, kept *int) {
	t.Helper()
	where := fmt.Sprintf("%d seats, seed %d", n, seed)
	ref, err := direct.New(ctx, direct.Options{Setup: fakeengine.Setup{Seed: seed, Seats: uint8(n)}, Tabs: true})
	if err != nil {
		t.Fatal(err)
	}
	f := newFixture(t, Config{BotWork: blockBots})
	f.tabs = true
	sp := humans(n, seed)
	sp.RandomSeating = true
	act, err := f.openInproc(ctx, sp)
	if err != nil {
		t.Fatal(err)
	}
	h := &th{t: t, ctx: ctx, f: f, p: act}
	// The creator switches from a connection of its own, so the scripts'
	// seqs never meet the switches'.
	if err := act.join(ctx, "admin", userOf(0), "P0"); err != nil {
		t.Fatal(err)
	}
	adminSeq := 0
	switchTo := func(what string, seat int, user string, clients ...tabletest.ClientID) {
		t.Helper()
		do, err := ref.Switch(ctx, seat, clients...)
		if err != nil {
			t.Fatal(err)
		}
		adminSeq++
		var ao string
		if user == "" {
			ao = h.do("admin", toBot(adminSeq, seat, BotEasy))
		} else {
			ao = h.do("admin", toHuman(adminSeq, seat, user))
		}
		if string(do) != ao {
			t.Fatalf("%s, %s: direct %s, actor %s", where, what, do, ao)
		}
	}
	const k1, k2 = 1, 2
	human2 := act.clients[tabletest.SeatClient(k2)].UserID
	clients := make([]tabletest.ClientID, n)
	viewers := []tabletest.ClientID{tabletest.SpectatorClient}
	for s := range clients {
		clients[s] = tabletest.SeatClient(s)
		viewers = append(viewers, tabletest.SeatClient(s), tabletest.TabClient(s))
	}
	seqs := tabletest.Seqs{}
	snaps := func() ([]tabletest.Snapshot, []tabletest.Snapshot) {
		t.Helper()
		d, a := make([]tabletest.Snapshot, n), make([]tabletest.Snapshot, n)
		for s, c := range clients {
			if d[s], err = ref.Snapshot(ctx, c); err != nil {
				t.Fatal(err)
			}
			if a[s], err = act.Snapshot(ctx, c); err != nil {
				t.Fatal(err)
			}
		}
		return d, a
	}
	both := func(what string, c tabletest.ClientID, sub tabletest.Submission, dRef, aRef string) tabletest.Outcome {
		t.Helper()
		do, err := ref.Submit(ctx, c, sub)
		if err != nil {
			t.Fatal(err)
		}
		if sub.Ref == dRef {
			sub.Ref = aRef
		}
		ao, err := act.Submit(ctx, c, sub)
		if err != nil {
			t.Fatal(err)
		}
		if do != ao {
			t.Fatalf("%s, %s (%s seq %d confirm=%t): direct %s, actor %s", where, what, c, sub.Seq, sub.Confirm, do, ao)
		}
		compareViews(ctx, t, where+", "+what, ref, act, viewers, dRef, aRef)
		return do
	}
	forbidden := 0
	for idx := 0; ; idx++ {
		if idx > 100 {
			t.Fatalf("%s: the game did not end", where)
		}
		dsnaps, asnaps := snaps()
		compareViews(ctx, t, fmt.Sprintf("%s, point %d", where, idx), ref, act, viewers, dsnaps[0].Ref, asnaps[0].Ref)
		if dsnaps[0].Over {
			break
		}
		if idx == 1 {
			switchTo("the spectator to seat 1", k1, "watcher", tabletest.SpectatorClient)
			old := clients[k1]
			clients[k1] = tabletest.SpectatorClient
			dsnaps, asnaps = snaps()
			if sn := dsnaps[0]; len(sn.Legal) > 0 {
				for _, c := range []tabletest.ClientID{old, tabletest.TabClient(k1)} {
					seqs[c]++
					if o := both("a former controller's pick", c, tabletest.Submission{Seq: seqs[c], Ref: sn.Ref, Key: sn.Legal[0], Confirm: true}, dsnaps[0].Ref, asnaps[0].Ref); o == tabletest.Forbidden {
						forbidden++
					}
				}
			}
		}
		scripts, err := tabletest.Scripts(dsnaps, clients, seed*37+uint64(idx)+salt, tabletest.ScriptOptions{Drafts: 70, Resends: 30}, seqs)
		if err != nil {
			t.Fatal(err)
		}
		ops := tabletest.Interleave(scripts, seed+uint64(idx)*13)
		for i, op := range ops {
			if idx == 2 && i == len(ops)/2 {
				if sn, _ := ref.Snapshot(ctx, clients[k2]); sn.Pick != "" && !sn.Confirm {
					*kept++
				}
				switchTo("seat 2 to a bot", k2, "")
			}
			if o := both(fmt.Sprintf("point %s, arrival %d", dsnaps[0].Point, i), op.Client, op.Sub, dsnaps[0].Ref, asnaps[0].Ref); o == tabletest.Forbidden {
				forbidden++
			}
		}
		if idx == 2 {
			switchTo("seat 2 back to its human", k2, human2, tabletest.SeatClient(k2), tabletest.TabClient(k2))
		}
	}
	if forbidden == 0 {
		t.Fatalf("%s: no former controller was refused", where)
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
