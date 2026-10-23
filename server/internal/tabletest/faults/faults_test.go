package faults_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/haftgane/fakeengine"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/direct"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/faults"
)

func setup(seats uint8, seed uint64) fakeengine.Setup {
	return fakeengine.Setup{Seed: seed, Seats: seats}
}

func stored(t *testing.T, s fakeengine.Setup) (*direct.Table, *faults.FileStore, direct.Options) {
	t.Helper()
	store := faults.NewFileStore(t.TempDir())
	opts := direct.Options{Setup: s, Store: store, Crashed: faults.IsCrash}
	tb, err := direct.New(t.Context(), opts)
	if err != nil {
		t.Fatal(err)
	}
	return tb, store, opts
}

func snap(t *testing.T, tb tabletest.Table, seat int) tabletest.Snapshot {
	t.Helper()
	sn, err := tb.Snapshot(t.Context(), tabletest.SeatClient(seat))
	if err != nil {
		t.Fatal(err)
	}
	return sn
}

func submit(t *testing.T, tb tabletest.Table, seat int, seq uint64, key string, confirm bool) tabletest.Outcome {
	t.Helper()
	o, err := tb.Submit(t.Context(), tabletest.SeatClient(seat), tabletest.Submission{Seq: seq, Ref: snap(t, tb, seat).Ref, Key: key, Confirm: confirm})
	if err != nil {
		t.Fatalf("seat %d seq %d: %v", seat, seq, err)
	}
	return o
}

func canonical(t *testing.T, tb tabletest.Table) tabletest.Canonical {
	t.Helper()
	c, err := tb.Canonical(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestStorageErrorDiscardsTheCandidate(t *testing.T) {
	tb, store, _ := stored(t, setup(3, 1))
	before, _ := store.Load()
	key := snap(t, tb, 0).Legal[0]
	store.Inject(faults.FailWrite)
	if o := submit(t, tb, 0, 1, key, true); o != tabletest.StorageError {
		t.Fatalf("outcome %s, want %s", o, tabletest.StorageError)
	}
	after, _ := store.Load()
	if !bytes.Equal(before, after) {
		t.Fatal("a failed write changed the record")
	}
	if sn := snap(t, tb, 0); sn.Pick != "" || len(sn.Confirmed) != 0 {
		t.Fatalf("a failed write changed the table: %+v", sn)
	}
	if o := submit(t, tb, 0, 1, key, true); o != tabletest.Accepted {
		t.Fatalf("the retry of the same seq: %s, want accepted (a refusal uses up no seq)", o)
	}
}

// crashOnLastConfirmation confirms seats 1.. and crashes the write of seat
// 0's completing confirmation.
func crashOnLastConfirmation(t *testing.T, fault faults.Fault) (*direct.Table, *faults.FileStore, direct.Options, string) {
	t.Helper()
	tb, store, opts := stored(t, setup(3, 2))
	for s := 1; s < 3; s++ {
		if o := submit(t, tb, s, 1, snap(t, tb, s).Legal[0], true); o != tabletest.Accepted {
			t.Fatalf("seat %d: %s", s, o)
		}
	}
	key := snap(t, tb, 0).Legal[0]
	store.Inject(fault)
	_, err := tb.Submit(t.Context(), tabletest.SeatClient(0), tabletest.Submission{Seq: 1, Ref: snap(t, tb, 1).Ref, Key: key, Confirm: true})
	if !faults.IsCrash(err) {
		t.Fatalf("Submit = %v, want a crash", err)
	}
	if _, err := tb.Snapshot(t.Context(), tabletest.SeatClient(0)); !errors.Is(err, direct.ErrDead) {
		t.Fatalf("the crashed table still answers: %v", err)
	}
	data, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := direct.Restore(opts, data)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	return restored, store, opts, key
}

func TestCrashBeforeRename(t *testing.T) {
	tb, store, _, key := crashOnLastConfirmation(t, faults.CrashBeforeRename)
	if temps, _ := store.Temps(); len(temps) != 1 {
		t.Fatalf("temporary files %v, want the one the crash left", temps)
	}
	sn := snap(t, tb, 0)
	if len(canonical(t, tb).Batches) != 0 || !slices.Equal(sn.Confirmed, []int{1, 2}) || sn.Pick != "" {
		t.Fatalf("restored %+v; want the point open with seats 1 and 2 confirmed and no pick for seat 0", sn)
	}
	if o := submit(t, tb, 0, 1, key, true); o != tabletest.Resolved {
		t.Fatalf("the retry: %s, want resolved", o)
	}
	if n := len(canonical(t, tb).Batches); n != 1 {
		t.Fatalf("%d batches after the retry, want 1", n)
	}
}

func TestCrashAfterRename(t *testing.T) {
	tb, _, opts, key := crashOnLastConfirmation(t, faults.CrashAfterRename)
	c := canonical(t, tb)
	if len(c.Batches) != 1 || c.Batches[0].Point != "age-1-turn-1" {
		t.Fatalf("restored %+v; want the point resolved", c.Batches)
	}
	// The client never heard back, so it sends the same seq again: the
	// restored seqs make it a duplicate, and nothing resolves twice.
	o, err := tb.Submit(t.Context(), tabletest.SeatClient(0), tabletest.Submission{Seq: 1, Ref: "age-1-turn-1@0", Key: key, Confirm: true})
	if err != nil || o != tabletest.Duplicate {
		t.Fatalf("the retry: %s %v, want duplicate", o, err)
	}
	if n := len(canonical(t, tb).Batches); n != 1 {
		t.Fatalf("%d batches after the retry, want 1", n)
	}

	// A pick that does not complete the point, crashed after its rename,
	// is restored, and its retry is a duplicate too.
	store := faults.NewFileStore(t.TempDir())
	opts.Store = store
	data, _ := json.Marshal(tb.Record())
	if err := store.Save(t.Context(), data); err != nil {
		t.Fatal(err)
	}
	tb, err = direct.Restore(opts, data)
	if err != nil {
		t.Fatal(err)
	}
	seat := snap(t, tb, 0).Required[0] // the next point may be a one-seat point
	k := snap(t, tb, seat).Legal[0]
	store.Inject(faults.CrashAfterRename)
	if _, err := tb.Submit(t.Context(), tabletest.SeatClient(seat), tabletest.Submission{Seq: 2, Ref: snap(t, tb, seat).Ref, Key: k}); !faults.IsCrash(err) {
		t.Fatalf("Submit = %v, want a crash", err)
	}
	data, _ = store.Load()
	tb, err = direct.Restore(opts, data)
	if err != nil {
		t.Fatal(err)
	}
	if sn := snap(t, tb, seat); sn.Pick != k || sn.Confirm {
		t.Fatalf("restored seat %d %+v, want its unconfirmed pick", seat, sn)
	}
	if o := submit(t, tb, seat, 2, k, false); o != tabletest.Duplicate {
		t.Fatalf("the retry: %s, want duplicate", o)
	}
}

// onePointPrefix returns the batches leading to the first point of kind k
// in a seeded random game, and that point's decider.
func onePointPrefix(t *testing.T, seats uint8, kind fakeengine.PointKind) (fakeengine.Setup, []tabletest.Batch, int) {
	t.Helper()
	for seed := uint64(0); seed < 500; seed++ {
		s := setup(seats, seed)
		g, _ := fakeengine.NewGame(s)
		rng := rand.New(rand.NewPCG(seed, 1))
		var prefix []tabletest.Batch
		for g.Point().Kind != fakeengine.PointNone {
			p := g.Point()
			if p.Kind == kind {
				return s, prefix, int(p.Required.Seats()[0])
			}
			b := fakeengine.Batch{Point: p.ID}
			tb := tabletest.Batch{Point: p.ID}
			for _, seat := range p.Required.Seats() {
				cs := g.Legal(seat)
				k := cs[rng.IntN(len(cs))].Key
				b.Picks = append(b.Picks, fakeengine.Pick{Seat: seat, Key: k})
				tb.Picks = append(tb.Picks, tabletest.Pick{Seat: int(seat), Key: k})
			}
			g, _, _ = g.Resolve(b)
			prefix = append(prefix, tb)
		}
	}
	t.Fatalf("no %d-seat game reached a point of kind %d", seats, kind)
	return fakeengine.Setup{}, nil, 0
}

// TestRestoreEveryPendingSubset restores every valid pending block of a
// turn point at every seat count and of both one-seat points, checks that
// each seat sees its own pick and confirmed flag back, then completes the
// point and checks the batch.
func TestRestoreEveryPendingSubset(t *testing.T) {
	type pointCase struct {
		name   string
		setup  fakeengine.Setup
		prefix []tabletest.Batch
	}
	var cases []pointCase
	for n := uint8(fakeengine.MinSeats); n <= fakeengine.MaxSeats; n++ {
		cases = append(cases, pointCase{"turn", setup(n, uint64(n)), nil})
	}
	s, p, _ := onePointPrefix(t, 5, fakeengine.PointSalvage)
	cases = append(cases, pointCase{"salvage", s, p})
	s, p, _ = onePointPrefix(t, 4, fakeengine.PointLastCard)
	cases = append(cases, pointCase{"last card", s, p})

	for _, pc := range cases {
		opts := direct.Options{Setup: pc.setup}
		open, err := direct.Factory(opts)(t.Context(), pc.prefix)
		if err != nil {
			t.Fatal(err)
		}
		base := open.(*direct.Table).Record()
		legal := map[int][]string{}
		for _, seat := range snap(t, open, 0).Required {
			legal[seat] = snap(t, open, seat).Legal
		}
		required := snap(t, open, 0).Required
		key := func(seat int) string { return legal[seat][len(legal[seat])-1] }
		subsets := faults.PendingSubsets(required, key)
		if want := pow3(len(required)) - 1; len(subsets) != want {
			t.Fatalf("%s: %d subsets, want %d", pc.name, len(subsets), want)
		}
		for _, pend := range subsets {
			rec := base
			rec.Pending = toPending(pend)
			tb, err := direct.RestoreRecord(opts, rec)
			if err != nil {
				t.Fatalf("%s %+v: %v", pc.name, pend, err)
			}
			want := tabletest.Batch{Point: snap(t, tb, required[0]).Point}
			var confirmed []int
			for _, seat := range required {
				got := snap(t, tb, seat)
				k, c := "", false
				for _, sl := range pend {
					if sl.Seat == seat {
						k, c = sl.Key, sl.Confirmed
					}
				}
				if got.Pick != k || got.Confirm != c {
					t.Fatalf("%s %+v: seat %d sees %q/%t", pc.name, pend, seat, got.Pick, got.Confirm)
				}
				if c {
					confirmed = append(confirmed, seat)
				}
				if k == "" {
					k = legal[seat][0]
				}
				want.Picks = append(want.Picks, tabletest.Pick{Seat: seat, Key: k})
			}
			if !slices.Equal(snap(t, tb, required[0]).Confirmed, confirmed) {
				t.Fatalf("%s %+v: confirmed %v, want %v", pc.name, pend, snap(t, tb, required[0]).Confirmed, confirmed)
			}
			for _, pk := range want.Picks {
				if slices.Contains(confirmed, pk.Seat) {
					continue
				}
				submit(t, tb, pk.Seat, 1, pk.Key, true)
			}
			c := canonical(t, tb)
			if len(c.Batches) != len(pc.prefix)+1 || !c.Batches[len(c.Batches)-1].Equal(want) {
				t.Fatalf("%s %+v: batches %+v, want the last to be %+v", pc.name, pend, c.Batches, want)
			}
			g, err := fakeengine.Replay(pc.setup, engineBatches(c.Batches))
			if err != nil || g.Fingerprint() != c.Hash {
				t.Fatalf("%s %+v: the canonical history does not replay to its hash: %v", pc.name, pend, err)
			}
		}
	}
}

func TestRestoreRefusesInvalidRecords(t *testing.T) {
	s, prefix, decider := onePointPrefix(t, 4, fakeengine.PointSalvage)
	opts := direct.Options{Setup: s}
	open, err := direct.Factory(opts)(t.Context(), prefix)
	if err != nil {
		t.Fatal(err)
	}
	base := open.(*direct.Table).Record()
	key := func(int) string { return snap(t, open, decider).Legal[0] }
	outsider := (decider + 1) % 4
	for _, bad := range faults.InvalidPendings([]int{decider}, key, "sell:amber", outsider) {
		rec := base
		rec.Pending = toPending(bad.Pending)
		if _, err := direct.RestoreRecord(opts, rec); !errors.Is(err, direct.ErrRestore) {
			t.Errorf("%s: RestoreRecord = %v, want ErrRestore", bad.Name, err)
		}
	}
	turn := direct.Options{Setup: setup(3, 9)}
	open, _ = direct.Factory(turn)(t.Context(), nil)
	tbase := open.(*direct.Table).Record()
	for _, bad := range faults.InvalidPendings([]int{0, 1, 2}, func(seat int) string { return snap(t, open, seat).Legal[0] }, "salvage:amber", 6) {
		rec := tbase
		rec.Pending = toPending(bad.Pending)
		if _, err := direct.RestoreRecord(turn, rec); !errors.Is(err, direct.ErrRestore) {
			t.Errorf("turn point, %s: RestoreRecord = %v, want ErrRestore", bad.Name, err)
		}
	}
	for name, mutate := range map[string]func(*direct.Record){
		"a wrong hash":                        func(r *direct.Record) { r.Hash = "0000000000000000" },
		"another point reference":             func(r *direct.Record) { r.Ref = "age-2-turn-1@0" },
		"a reference newer than the revision": func(r *direct.Record) { r.Ref = r.Ref[:len(r.Ref)-1] + "9" },
		"an unoffered key in a batch": func(r *direct.Record) {
			r.Batches = append(r.Batches, tabletest.Batch{Point: "age-1-turn-1", Picks: []tabletest.Pick{
				{Seat: 0, Key: "sell:rune"}, {Seat: 1, Key: "sell:rune"}, {Seat: 2, Key: "sell:rune"},
			}})
		},
	} {
		rec := tbase
		rec.Batches = slices.Clone(tbase.Batches)
		mutate(&rec)
		if _, err := direct.RestoreRecord(turn, rec); !errors.Is(err, direct.ErrRestore) {
			t.Errorf("%s: RestoreRecord = %v, want ErrRestore", name, err)
		}
	}
}

// TestCancellationLeaksNothing cancels a bot search in flight: no pick is
// made, no permit is held and nobody waits; a later run lets every bot
// pick.
func TestCancellationLeaksNothing(t *testing.T) {
	work := faults.NewWork(true)
	permits := faults.NewPermits(1)
	tb, err := direct.New(t.Context(), direct.Options{Setup: setup(4, 3), Bots: []int{1, 2, 3}, Permits: permits, Work: work.Run})
	if err != nil {
		t.Fatal(err)
	}
	type result struct {
		n   int
		err error
	}
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan result)
	go func() {
		n, err := tb.RunBots(ctx)
		done <- result{n, err}
	}()
	if st := <-work.Started(); st.Seat != 1 || st.Point != "age-1-turn-1" {
		t.Fatalf("the first search is %+v", st)
	}
	cancel()
	r := <-done
	if !errors.Is(r.err, context.Canceled) || r.n != 0 {
		t.Fatalf("RunBots = %d, %v; want 0, context.Canceled", r.n, r.err)
	}
	if permits.Held() != 0 || permits.Waiting() != 0 {
		t.Fatalf("after cancellation %d permits held, %d waiting", permits.Held(), permits.Waiting())
	}
	if started, finished, cancelled := work.Counts(); started != 1 || finished != 0 || cancelled != 1 {
		t.Fatalf("work counts %d/%d/%d, want 1 started, 1 cancelled", started, finished, cancelled)
	}
	if sn := snap(t, tb, 0); len(sn.Confirmed) != 0 {
		t.Fatalf("a cancelled search picked: %+v", sn)
	}

	go func() {
		n, err := tb.RunBots(t.Context())
		done <- result{n, err}
	}()
	for range 3 {
		<-work.Started()
		work.Release()
	}
	r = <-done
	if r.err != nil || r.n != 3 {
		t.Fatalf("RunBots = %d, %v; want 3 picks", r.n, r.err)
	}
	if sn := snap(t, tb, 0); !slices.Equal(sn.Confirmed, []int{1, 2, 3}) || permits.Held() != 0 {
		t.Fatalf("after the bots: %+v, %d permits held", sn, permits.Held())
	}
}

// TestNoHumanPause: with no human participant present no search starts,
// and a spectator does not change that; once a human is back the bots pick.
func TestNoHumanPause(t *testing.T) {
	work := faults.NewWork(false)
	tb, err := direct.New(t.Context(), direct.Options{Setup: setup(3, 4), Bots: []int{1, 2}, Work: work.Run})
	if err != nil {
		t.Fatal(err)
	}
	tb.Detach(tabletest.SeatClient(0))
	tb.Detach(tabletest.SpectatorClient)
	if n, err := tb.RunBots(t.Context()); !errors.Is(err, direct.ErrNoHuman) || n != 0 {
		t.Fatalf("RunBots with nobody = %d, %v; want ErrNoHuman", n, err)
	}
	tb.Attach("watcher", -1)
	if n, err := tb.RunBots(t.Context()); !errors.Is(err, direct.ErrNoHuman) || n != 0 {
		t.Fatalf("RunBots with only a spectator = %d, %v; want ErrNoHuman", n, err)
	}
	if started, _, _ := work.Counts(); started != 0 {
		t.Fatalf("%d searches started with no human present", started)
	}
	tb.Attach(tabletest.SeatClient(0), 0)
	if n, err := tb.RunBots(t.Context()); err != nil || n != 2 {
		t.Fatalf("RunBots with the human back = %d, %v; want 2", n, err)
	}
	if o := submit(t, tb, 0, 1, snap(t, tb, 0).Legal[0], true); o != tabletest.Resolved {
		t.Fatalf("the human's confirmation: %s, want resolved", o)
	}
}

func TestPendingSubsetsCount(t *testing.T) {
	for k := 1; k <= 7; k++ {
		req := make([]int, k)
		for i := range req {
			req[i] = i
		}
		subs := faults.PendingSubsets(req, func(int) string { return "k" })
		if len(subs) != pow3(k)-1 || len(subs[0]) != 0 {
			t.Fatalf("%d seats: %d subsets", k, len(subs))
		}
		for _, p := range subs {
			c := 0
			for _, s := range p {
				if s.Confirmed {
					c++
				}
			}
			if c == k {
				t.Fatalf("%d seats: an all-confirmed block %+v", k, p)
			}
		}
	}
}

func pow3(k int) int {
	n := 1
	for range k {
		n *= 3
	}
	return n
}

func toPending(p faults.Pending) []direct.PendingSlot {
	out := []direct.PendingSlot{}
	for _, s := range p {
		out = append(out, direct.PendingSlot{Seat: s.Seat, Key: s.Key, Confirmed: s.Confirmed, Seqs: []direct.ClientSeq{}})
	}
	return out
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
