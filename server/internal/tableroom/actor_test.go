package tableroom

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/faults"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

type th struct {
	t   *testing.T
	ctx context.Context
	f   *fixture
	p   *inproc
}

func newTH(t *testing.T, cfg Config, spec Spec) *th {
	t.Helper()
	ctx := testContext(t)
	f := newFixture(t, cfg)
	p, err := f.openInproc(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	return &th{t: t, ctx: ctx, f: f, p: p}
}

func (h *th) snap(c tabletest.ClientID) tabletest.Snapshot {
	h.t.Helper()
	sn, err := h.p.Snapshot(h.ctx, c)
	if err != nil {
		h.t.Fatal(err)
	}
	return sn
}

func (h *th) send(c tabletest.ClientID, seq uint64, ref, key string, confirm bool) tabletest.Outcome {
	h.t.Helper()
	o, err := h.p.Submit(h.ctx, c, tabletest.Submission{Seq: seq, Ref: ref, Key: key, Confirm: confirm})
	if err != nil {
		h.t.Fatal(err)
	}
	return o
}

func (h *th) expect(got, want tabletest.Outcome, what string) {
	h.t.Helper()
	if got != want {
		h.t.Fatalf("%s: %s, want %s", what, got, want)
	}
}

// record reads the table's stored record.
func (h *th) record() Record {
	h.t.Helper()
	data, _, err := h.f.tables.store().PeekGame(h.p.code, Game)
	if err != nil {
		h.t.Fatal(err)
	}
	r, err := decodeRecord(data)
	if err != nil {
		h.t.Fatal(err)
	}
	return *r
}

// framesSince returns the frames client c received since index i.
func (h *th) framesSince(c tabletest.ClientID, i int) []wire.TableState {
	h.p.drain()
	return h.p.frames[c][i:]
}

func (h *th) frameCount(c tabletest.ClientID) int {
	h.p.drain()
	return len(h.p.frames[c])
}

// rejoin attaches the driver's clients again after a restart.
func (h *th) rejoin() {
	h.t.Helper()
	old := h.p.clients
	h.p.clients = map[tabletest.ClientID]*room.Client{}
	for id, c := range old {
		if err := h.p.join(h.ctx, id, c.UserID, c.Name); err != nil {
			h.t.Fatalf("rejoin %s: %v", id, err)
		}
	}
}

// TestAcceptanceRules mirrors the reference model's: the refusals in the
// design's order, duplicates, a changed pick and a withdrawn confirmation,
// and the completing confirmation resolving once.
func TestAcceptanceRules(t *testing.T) {
	h := newTH(t, Config{}, humans(3, 77))
	a, b, c := tabletest.SeatClient(0), tabletest.SeatClient(1), tabletest.SeatClient(2)
	sa := h.snap(a)
	ref := sa.Ref
	k0, k1 := sa.Legal[0], sa.Legal[1]
	h.expect(h.send(tabletest.SpectatorClient, 1, ref, k0, true), tabletest.Forbidden, "a spectator")
	h.expect(h.send(a, 1, "age-1-turn-1@99", k0, false), tabletest.Closed, "another reference")
	h.expect(h.send(a, 1, ref, "salvage:amber", false), tabletest.Illegal, "an unoffered key")
	h.expect(h.send(a, 1, ref, k0, true), tabletest.Accepted, "a pick (refusals used up no seq)")
	h.expect(h.send(a, 1, ref, k1, false), tabletest.Duplicate, "the same seq again")
	if sn := h.snap(a); sn.Pick != k0 || !sn.Confirm {
		t.Fatalf("a duplicate changed the pick: %+v", sn)
	}
	h.expect(h.send(a, 2, ref, k1, false), tabletest.Accepted, "a change, withdrawing the confirmation")
	if sn := h.snap(b); len(sn.Confirmed) != 0 || sn.Pick != "" {
		t.Fatalf("seat 1 sees %+v", sn)
	}
	h.expect(h.send(a, 3, ref, k1, true), tabletest.Accepted, "confirming again")
	h.expect(h.send(b, 1, ref, h.snap(b).Legal[0], true), tabletest.Accepted, "seat 1")
	h.expect(h.send(c, 1, ref, h.snap(c).Legal[0], true), tabletest.Resolved, "the completing confirmation")
	h.expect(h.send(a, 4, ref, k0, true), tabletest.Closed, "a change after the last confirmation")
	can, _ := h.p.Canonical(h.ctx)
	if len(can.Batches) != 1 || can.Batches[0].Picks[0].Key != k1 {
		t.Fatalf("batches %+v", can.Batches)
	}
	if sn := h.snap(a); sn.Ref == ref || sn.Pick != "" || len(sn.Confirmed) != 0 {
		t.Fatalf("the next point %+v reuses the reference or keeps a pick", sn)
	}
	// Each refusal reached its submitter only, with the design's codes.
	var codes []string
	for _, r := range h.p.rejects[a] {
		codes = append(codes, r.Code)
	}
	if !slices.Equal(codes, []string{"decision-closed", "illegal", "decision-closed"}) || len(h.p.rejects[b]) != 0 {
		t.Fatalf("seat 0's rejects %v, seat 1's %v", codes, h.p.rejects[b])
	}
	if r := h.p.rejects[tabletest.SpectatorClient]; len(r) != 1 || r[0].Code != "forbidden" {
		t.Fatalf("the spectator's rejects %v", r)
	}
}

// TestPrivateEditsChangeNobodyElsesFrame: an unconfirmed pick or a change
// reaches only the seat's own connections; others receive no frame and no
// revision moves for them. A confirmation flip is public.
func TestPrivateEditsChangeNobodyElsesFrame(t *testing.T) {
	h := newTH(t, Config{}, humans(4, 5))
	a, b := tabletest.SeatClient(0), tabletest.SeatClient(1)
	tab2 := tabletest.ClientID("seat-0-tab-2")
	if err := h.p.join(h.ctx, tab2, userOf(0), "P0"); err != nil {
		t.Fatal(err)
	}
	sa := h.snap(a)
	nb, ns, n2 := h.frameCount(b), h.frameCount(tabletest.SpectatorClient), h.frameCount(tab2)
	revB := h.p.frames[b][nb-1].Revision
	h.expect(h.send(a, 1, sa.Ref, sa.Legal[0], false), tabletest.Accepted, "a draft")
	h.expect(h.send(a, 2, sa.Ref, sa.Legal[1], false), tabletest.Accepted, "a change")
	if len(h.framesSince(b, nb)) != 0 || len(h.framesSince(tabletest.SpectatorClient, ns)) != 0 {
		t.Fatal("another viewer received a frame for a private edit")
	}
	own := h.framesSince(tab2, n2)
	if len(own) != 2 || own[1].Pending == nil || own[1].Pending.Key != sa.Legal[1] || own[1].Ack != nil {
		t.Fatalf("the seat's other tab got %+v", own)
	}
	if mine := h.p.frames[a][len(h.p.frames[a])-1]; mine.Ack == nil || mine.Ack.Seq != 2 || mine.Revision <= revB {
		t.Fatalf("the submitter's frame %+v", mine)
	}
	h.expect(h.send(a, 3, sa.Ref, sa.Legal[1], true), tabletest.Accepted, "a confirmation")
	fb := h.framesSince(b, nb)
	if len(fb) != 1 || fb[0].Pending != nil || !slices.Equal(fb[0].Point.Confirmed, []int{0}) || fb[0].Revision != revB+1<<editBits {
		t.Fatalf("seat 1 after the confirmation: %+v", fb)
	}
	if fb[0].Point.Ref.Revision != h.snapRef(b).Revision {
		t.Fatal("a confirmation replaced the point reference")
	}
}

func (h *th) snapRef(c tabletest.ClientID) wire.PointRef { return parseRef(h.snap(c).Ref) }

// TestTwoTabsAndForgottenSeqs: two tabs of a seat keep their own seqs.
// With room for one client per user, a tab pushes out the other tab's
// acknowledgement, and the slot's floor still answers for it: the seq that
// last changed the pick is acknowledged again, an older one is stale, and
// neither reverts the newer pick, before or after a restart. Seq 0 is a
// seq like any other.
func TestTwoTabsAndForgottenSeqs(t *testing.T) {
	h := newTH(t, Config{SeqClients: 1}, humans(3, 77))
	a, b := tabletest.SeatClient(0), tabletest.SeatClient(1)
	tab2 := tabletest.ClientID("seat-0-tab-2")
	if err := h.p.join(h.ctx, tab2, userOf(0), "P0"); err != nil {
		t.Fatal(err)
	}
	sa := h.snap(a)
	h.expect(h.send(a, 0, sa.Ref, sa.Legal[0], false), tabletest.Accepted, "a draft at seq 0")
	h.expect(h.send(tab2, 0, sa.Ref, sa.Legal[1], false), tabletest.Accepted, "tab two, forgetting tab one")
	quiet := h.quiet(tab2, b, tabletest.SpectatorClient)
	h.expect(h.send(a, 0, sa.Ref, sa.Legal[0], false), tabletest.Duplicate, "tab one's seq 0 again")
	quiet()
	h.expect(h.send(a, 1, sa.Ref, sa.Legal[2], false), tabletest.Accepted, "tab one changes it")
	h.expect(h.send(tab2, 1, sa.Ref, sa.Legal[1], false), tabletest.Accepted, "tab two changes it back")
	for _, restart := range []bool{false, true} {
		if restart {
			h.f.restart()
			h.rejoin()
		}
		quiet := h.quiet(tab2, b, tabletest.SpectatorClient)
		h.expect(h.send(a, 1, sa.Ref, sa.Legal[2], false), tabletest.Duplicate, "tab one's last seq again")
		h.expect(h.send(a, 0, sa.Ref, sa.Legal[0], false), tabletest.Stale, "tab one's older seq")
		quiet()
		if sn := h.snap(a); sn.Pick != sa.Legal[1] {
			t.Fatalf("restart %v: a forgotten seq changed the pick to %q", restart, sn.Pick)
		}
	}
	if r := h.p.rejects[a]; len(r) == 0 || r[len(r)-1].Code != "stale" {
		t.Fatalf("rejects %+v", r)
	}
	if fr := h.latest(a); fr.Ack == nil || fr.Ack.Seq != 1 || fr.Ack.ClientID != string(a) {
		t.Fatalf("the duplicate's acknowledgement: %+v", fr.Ack)
	}
	h.expect(h.send(tab2, 2, sa.Ref, sa.Legal[0], true), tabletest.Accepted, "tab two's newer seq")
	if sn := h.snap(a); sn.Pick != sa.Legal[0] || !sn.Confirm {
		t.Fatalf("tab one sees %+v", sn)
	}
}

// TestSeqMemoryIsPerUser: other users' private drafts from many tabs
// never push out a user's acknowledgements, so an unchanged retry from a
// user's older tab is the same harmless duplicate with or without them,
// before and after a restart, and never replaces the newer pick of the
// user's other tab.
func TestSeqMemoryIsPerUser(t *testing.T) {
	for _, churn := range []bool{false, true} {
		h := newTH(t, Config{}, humans(3, 2027))
		a := tabletest.SeatClient(0)
		own, other := h.snap(a), h.snap(tabletest.SeatClient(1))
		h.expect(h.send(a, 0, own.Ref, own.Legal[0], false), tabletest.Accepted, "own draft")
		if err := h.p.join(h.ctx, "own-newer", userOf(0), "P0"); err != nil {
			t.Fatal(err)
		}
		h.expect(h.send("own-newer", 0, own.Ref, own.Legal[1], false), tabletest.Accepted, "a newer draft from a second tab")
		revision := h.latest(a).Revision
		if churn {
			for _, s := range []int{1, 2} {
				// 64 drafts in all: twice the old table-wide memory, and
				// within each seat's floor bound.
				h.tabs(s, DefaultPickClients, func(id tabletest.ClientID) {
					h.expect(h.send(id, 0, other.Ref, h.snap(id).Legal[0], false), tabletest.Accepted, "another user's draft")
				})
			}
		}
		for _, restart := range []bool{false, true} {
			if restart {
				h.f.restart()
				h.rejoin()
			}
			quiet := h.quiet("own-newer", tabletest.SeatClient(1), tabletest.SeatClient(2), tabletest.SpectatorClient)
			h.expect(h.send(a, 0, own.Ref, own.Legal[0], false), tabletest.Duplicate,
				fmt.Sprintf("churn %v restart %v: the first tab's unchanged retry", churn, restart))
			quiet()
			if fr := h.latest(a); fr.Pending == nil || fr.Pending.Key != own.Legal[1] || fr.Revision != revision {
				t.Fatalf("churn %v restart %v: the retry changed seat 0's frame: %+v at %d, want %d",
					churn, restart, fr.Pending, fr.Revision, revision)
			}
		}
	}
}

// TestTabChurnStaysRestorable: a pick keeps a floor for each client that
// changed it, at most PickClients per user, and a new tab past that is
// refused with rate before anything is written. Every accepted record
// restores, a restart loses nothing, a tab that already has a floor can
// still change the pick, and once the point resolves new tabs are welcome
// again.
func TestTabChurnStaysRestorable(t *testing.T) {
	h := newTH(t, Config{}, humans(3, 2026))
	sn := h.snap(tabletest.SeatClient(0))
	accepted, refused := 0, 0
	h.tabs(0, DefaultPickClients+4, func(id tabletest.ClientID) {
		before := h.raw()
		switch o := h.send(id, 1, sn.Ref, sn.Legal[0], false); o {
		case tabletest.Accepted:
			accepted++
		case tabletest.Rate:
			refused++
			if !bytes.Equal(before, h.raw()) {
				t.Fatal("a refused tab changed the record")
			}
		default:
			t.Fatalf("a new tab: %s", o)
		}
		r := h.record()
		if _, _, err := restore(&r, defaultBounds); err != nil {
			t.Fatalf("after %d accepted tabs the record does not restore: %v", accepted, err)
		}
	})
	if accepted != DefaultPickClients || refused != 4 {
		t.Fatalf("%d tabs accepted, %d refused", accepted, refused)
	}
	h.f.restart()
	h.rejoin()
	first := tabletest.ClientID("seat-0-new-tab-0")
	if err := h.p.join(h.ctx, first, userOf(0), "P0"); err != nil {
		t.Fatalf("a join after the restart: %v", err)
	}
	h.expect(h.send(first, 1, sn.Ref, sn.Legal[0], false), tabletest.Duplicate, "the first tab's seq again")
	h.expect(h.send(first, 2, sn.Ref, sn.Legal[1], true), tabletest.Accepted, "the first tab changes and confirms")
	h.expect(h.send(tabletest.SeatClient(0), 1, sn.Ref, sn.Legal[0], false), tabletest.Rate, "a client without a floor")
	if r := h.p.rejects[tabletest.SeatClient(0)]; len(r) != 1 || r[0].Code != "rate" || r[0].Seq != 1 {
		t.Fatalf("rejects %+v", r)
	}
	for _, s := range []int{1, 2} {
		o := h.snap(tabletest.SeatClient(s))
		h.send(tabletest.SeatClient(s), 1, o.Ref, o.Legal[0], true)
	}
	next := h.snap(tabletest.SeatClient(0))
	if next.Ref == sn.Ref {
		t.Fatal("the point did not resolve")
	}
	h.expect(h.send(tabletest.SeatClient(0), 1, next.Ref, next.Legal[0], false), tabletest.Accepted, "the refused seq at the next point")
}

// TestWritesKeepRestoreBounds: restore and every write check the same
// bounds. A record with one floor or acknowledgement more than a write
// keeps does not restore, and a candidate past them is refused before it
// is saved.
func TestWritesKeepRestoreBounds(t *testing.T) {
	h := newTH(t, Config{PickClients: 2, SeqClients: 2}, humans(3, 31))
	sn := h.snap(tabletest.SeatClient(0))
	h.expect(h.send(tabletest.SeatClient(0), 1, sn.Ref, sn.Legal[0], false), tabletest.Accepted, "a draft")
	bounds := seqBounds{acks: 2, floors: 2}
	many := func(n int) []SeqRecord {
		var xs []SeqRecord
		for i := range n {
			xs = append(xs, SeqRecord{User: userOf(0), Client: fmt.Sprintf("tab-%d", i), Seq: 1})
		}
		return xs
	}
	for _, tc := range []struct {
		name   string
		mutate func(r *Record)
		ok     bool
	}{
		{"floors at the bound", func(r *Record) { r.Pending[0].Seqs = many(2) }, true},
		{"floors past the bound", func(r *Record) { r.Pending[0].Seqs = many(3) }, false},
		{"acknowledgements at the bound", func(r *Record) { r.Seqs = many(2) }, true},
		{"floors of as many users as a slot keeps", func(r *Record) {
			r.Pending[0].Seqs = nil
			for u := range maxSlotUsers {
				r.Pending[0].Seqs = append(r.Pending[0].Seqs, SeqRecord{User: userOf(u), Client: "c", Seq: 1})
			}
		}, true},
		{"floors of more users than a slot keeps", func(r *Record) {
			r.Pending[0].Seqs = nil
			for u := range maxSlotUsers + 1 {
				r.Pending[0].Seqs = append(r.Pending[0].Seqs, SeqRecord{User: userOf(u), Client: "c", Seq: 1})
			}
		}, false},
		{"acknowledgements of a user with no authority", func(r *Record) {
			r.Seqs = append(r.Seqs, SeqRecord{User: "watcher", Client: "c", Seq: 1})
		}, false},
		{"acknowledgements of a retired user", func(r *Record) {
			r.Seqs = append(r.Seqs, SeqRecord{User: "watcher", Client: "c", Seq: 1})
			r.Retired = append(r.Retired, "watcher")
		}, true},
	} {
		r := h.record()
		tc.mutate(&r)
		if _, _, err := restore(&r, bounds); (err == nil) != tc.ok {
			t.Errorf("%s: restore: %v", tc.name, err)
		}
		a, err := h.p.actor()
		if err != nil {
			t.Fatal(err)
		}
		before := h.raw()
		var cerr error
		if err := a.call(h.ctx, func() {
			cand := a.st.rec.clone()
			tc.mutate(&cand)
			if !tc.ok {
				cerr = a.commit("probe", cand, a.st.game)
			}
		}); err != nil {
			t.Fatal(err)
		}
		if !tc.ok && (cerr == nil || !bytes.Equal(before, h.raw())) {
			t.Errorf("%s: the write was not refused (%v)", tc.name, cerr)
		}
	}
}

// raw is the table's stored record, byte for byte.
func (h *th) raw() []byte {
	h.t.Helper()
	data, _, err := h.f.tables.store().PeekGame(h.p.code, Game)
	if err != nil {
		h.t.Fatal(err)
	}
	return data
}

// tabs opens n tabs of seat's user one after another, each a new client
// that runs fn and closes before the next opens.
func (h *th) tabs(seat, n int, fn func(id tabletest.ClientID)) {
	h.t.Helper()
	for i := range n {
		id := tabletest.ClientID(fmt.Sprintf("seat-%d-new-tab-%d", seat, i))
		if err := h.p.join(h.ctx, id, userOf(seat), fmt.Sprintf("P%d", seat)); err != nil {
			h.t.Fatal(err)
		}
		fn(id)
		h.f.m.Leave(h.p.clients[id], h.p.code)
		delete(h.p.clients, id)
	}
}

// quiet records the clients' frame counts; the returned check fails if any
// of them received a frame since.
func (h *th) quiet(cs ...tabletest.ClientID) func() {
	before := make([]int, len(cs))
	for i, c := range cs {
		before[i] = h.frameCount(c)
	}
	return func() {
		h.t.Helper()
		for i, c := range cs {
			if n := h.frameCount(c); n != before[i] {
				h.t.Fatalf("%s received %d frames", c, n-before[i])
			}
		}
	}
}

// TestReloadRestoresPick: a pick and its confirmed state survive a restart,
// with the same point reference; the seq acknowledgement survives too.
func TestReloadRestoresPick(t *testing.T) {
	h := newTH(t, Config{}, humans(3, 9))
	a, b := tabletest.SeatClient(0), tabletest.SeatClient(1)
	sa, sb := h.snap(a), h.snap(b)
	h.expect(h.send(a, 1, sa.Ref, sa.Legal[1], true), tabletest.Accepted, "seat 0 confirms")
	h.expect(h.send(b, 1, sb.Ref, sb.Legal[0], false), tabletest.Accepted, "seat 1 drafts")
	h.f.restart()
	h.rejoin()
	if sn := h.snap(a); sn.Ref != sa.Ref || sn.Pick != sa.Legal[1] || !sn.Confirm || !slices.Equal(sn.Confirmed, []int{0}) {
		t.Fatalf("seat 0 after a restart: %+v", sn)
	}
	if sn := h.snap(b); sn.Pick != sb.Legal[0] || sn.Confirm {
		t.Fatalf("seat 1 after a restart: %+v", sn)
	}
	if sn := h.snap(tabletest.SeatClient(2)); sn.Pick != "" {
		t.Fatalf("seat 2 sees a pick: %+v", sn)
	}
	h.expect(h.send(a, 1, sa.Ref, sa.Legal[0], false), tabletest.Duplicate, "the acknowledged seq after a restart")
}

// TestStorageFaults: a failed write changes nothing and answers storage; a
// crash before the rename loses the pick, after it keeps it, and the
// retry of the same seq is then a harmless duplicate that cannot resolve
// twice.
func TestStorageFaults(t *testing.T) {
	h := newTH(t, Config{}, humans(3, 11))
	a, b, c := tabletest.SeatClient(0), tabletest.SeatClient(1), tabletest.SeatClient(2)
	sa := h.snap(a)
	h.f.store.inject(faults.FailWrite)
	h.expect(h.send(a, 1, sa.Ref, sa.Legal[0], true), tabletest.StorageError, "a failed write")
	if sn := h.snap(b); len(sn.Confirmed) != 0 {
		t.Fatal("a failed write was published")
	}
	if r := h.p.rejects[a]; len(r) != 1 || r[0].Code != "storage" {
		t.Fatalf("rejects %+v", r)
	}
	h.expect(h.send(a, 1, sa.Ref, sa.Legal[0], true), tabletest.Accepted, "the retry")

	// Crash before the rename: the pick is lost; the restored table takes
	// the retry.
	sb := h.snap(b)
	h.f.store.inject(faults.CrashBeforeRename)
	if _, err := h.p.Submit(h.ctx, b, tabletest.Submission{Seq: 1, Ref: sb.Ref, Key: sb.Legal[0], Confirm: true}); !faults.IsCrash(err) {
		t.Fatalf("crash before rename: %v", err)
	}
	h.f.restart()
	h.rejoin()
	if sn := h.snap(b); sn.Pick != "" || !slices.Equal(sn.Confirmed, []int{0}) {
		t.Fatalf("after a crash before the rename: %+v", sn)
	}
	h.expect(h.send(b, 1, sb.Ref, sb.Legal[0], true), tabletest.Accepted, "the retry after a crash before the rename")

	// Crash after the rename, on the completing confirmation: the batch is
	// durable; the retry is a duplicate and nothing resolves twice.
	sc := h.snap(c)
	h.f.store.inject(faults.CrashAfterRename)
	if _, err := h.p.Submit(h.ctx, c, tabletest.Submission{Seq: 1, Ref: sc.Ref, Key: sc.Legal[0], Confirm: true}); !faults.IsCrash(err) {
		t.Fatalf("crash after rename: %v", err)
	}
	h.f.restart()
	h.rejoin()
	h.expect(h.send(c, 1, sc.Ref, sc.Legal[0], true), tabletest.Duplicate, "the retry after a crash after the rename")
	h.expect(h.send(c, 2, sc.Ref, sc.Legal[0], true), tabletest.Closed, "a new seq for the resolved point")
	can, _ := h.p.Canonical(h.ctx)
	if len(can.Batches) != 1 {
		t.Fatalf("%d batches after the crash", len(can.Batches))
	}
	temps, _ := filepath.Glob(filepath.Join(h.f.store.Dir(), "*tmp*"))
	_ = temps // a crash before the rename may leave a temporary file; the store ignores it
}

// TestRestorePendingSubsets: every subset of required seats with a pending
// pick (confirmed or not, never all confirmed) restores exactly; every
// invalid pending block archives the table as replay-failed.
func TestRestorePendingSubsets(t *testing.T) {
	h := newTH(t, Config{}, humans(3, 13))
	sa := h.snap(tabletest.SeatClient(0))
	legal := map[int]string{}
	for s := range 3 {
		legal[s] = h.snap(tabletest.SeatClient(s)).Legal[0]
	}
	base := h.record()
	h.p.leave()
	h.f.restart()
	write := func(code string, r Record) {
		r.Code = code
		data, err := r.encode()
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(h.f.store.Dir(), code+".json"), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	required := []int{0, 1, 2}
	subsets := faults.PendingSubsets(required, func(s int) string { return legal[s] })
	for i, sub := range subsets {
		code := testCode("PENA", i)
		r := base.clone()
		r.Pending = []PendingSlot{}
		for _, sl := range sub {
			r.Pending = append(r.Pending, PendingSlot{Seat: sl.Seat, Key: sl.Key, Confirmed: sl.Confirmed, Seqs: []SeqRecord{}})
		}
		write(code, r)
		for s := range 3 {
			c := newClient(userOf(s), "restore", "P")
			if err := h.f.m.Join(h.ctx, c, code); err != nil {
				t.Fatalf("subset %d: %v", i, err)
			}
			fr := lastFrame(t, c)
			var want *PendingSlot
			for k := range r.Pending {
				if r.Pending[k].Seat == s {
					want = &r.Pending[k]
				}
			}
			switch {
			case want == nil && fr.Pending != nil, want != nil && (fr.Pending == nil || fr.Pending.Key != want.Key || fr.Pending.Confirm != want.Confirmed):
				t.Fatalf("subset %d, seat %d: pending %+v, want %+v", i, s, fr.Pending, want)
			case refString(fr.Point.Ref) != sa.Ref:
				t.Fatalf("subset %d: the reference changed", i)
			}
			h.f.m.Leave(c, code)
		}
	}
	for i, bad := range faults.InvalidPendings(required, func(s int) string { return legal[s] }, "salvage:amber", 5) {
		code := testCode("PENB", i)
		r := base.clone()
		r.Pending = []PendingSlot{}
		for _, sl := range bad.Pending {
			r.Pending = append(r.Pending, PendingSlot{Seat: sl.Seat, Key: sl.Key, Confirmed: sl.Confirmed, Seqs: []SeqRecord{}})
		}
		write(code, r)
		c := newClient(userOf(0), "restore", "P")
		err := h.f.m.Join(h.ctx, c, code)
		if !errors.Is(err, room.ErrRoomArchived) {
			t.Fatalf("%s: join %v, want archived", bad.Name, err)
		}
		if _, archived, err := h.f.store.PeekGame(code, Game); err != nil || !archived {
			t.Fatalf("%s: not archived (%v)", bad.Name, err)
		}
		data, _ := os.ReadFile(filepath.Join(h.f.store.ArchiveDir(), code+".json"))
		var m map[string]any
		if json.Unmarshal(data, &m) != nil || m["archiveReason"] != string(store.ArchiveReplayFailed) {
			t.Fatalf("%s: archive reason %v", bad.Name, m["archiveReason"])
		}
	}
}

// testCode is a valid room code: prefix and two characters for i.
func testCode(prefix string, i int) string {
	const alphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	return prefix + string(alphabet[i/len(alphabet)]) + string(alphabet[i%len(alphabet)])
}

func lastFrame(t *testing.T, c *room.Client) wire.TableState {
	t.Helper()
	var last *wire.TableState
	for _, m := range c.Out.Take() {
		if st, ok := m.(wire.TableState); ok {
			last = &st
		}
	}
	if last == nil {
		t.Fatal("no frame")
	}
	return *last
}

// TestCompatibility: a hash-only mismatch refreshes the hash in place and
// keeps every other field; another rules version archives as
// rules-version; a file of an unknown schema or game is refused and left
// byte for byte.
func TestCompatibility(t *testing.T) {
	h := newTH(t, Config{}, humans(3, 15))
	sa := h.snap(tabletest.SeatClient(0))
	h.expect(h.send(tabletest.SeatClient(0), 1, sa.Ref, sa.Legal[0], false), tabletest.Accepted, "a draft")
	base := h.record()
	h.p.leave()
	h.f.restart()
	dir := h.f.store.Dir()
	put := func(code string, mutate func(map[string]any)) []byte {
		r := base.clone()
		r.Code = code
		data, _ := r.encode()
		var m map[string]any
		_ = json.Unmarshal(data, &m)
		mutate(m)
		data, _ = json.Marshal(m)
		if err := os.WriteFile(filepath.Join(dir, code+".json"), data, 0o600); err != nil {
			t.Fatal(err)
		}
		return data
	}
	put("CMPAAA", func(m map[string]any) { m["hash"] = "stale"; m["futureField"] = "kept" })
	c := newClient(userOf(0), "x", "P0")
	if err := h.f.m.Join(h.ctx, c, "CMPAAA"); err != nil {
		t.Fatal(err)
	}
	if fr := lastFrame(t, c); fr.Pending == nil {
		t.Fatal("the refreshed table lost its pending pick")
	}
	data, _ := os.ReadFile(filepath.Join(dir, "CMPAAA.json"))
	var m map[string]any
	_ = json.Unmarshal(data, &m)
	if m["hash"] != base.Hash || m["futureField"] != "kept" || m["updated"] != base.Updated.Format("2006-01-02T15:04:05.999999999Z07:00") {
		t.Fatalf("refreshed record: hash %v, future %v, updated %v", m["hash"], m["futureField"], m["updated"])
	}
	h.f.m.Leave(c, "CMPAAA")

	put("CMPAAB", func(m map[string]any) { m["rules"] = RulesVersion + 1 })
	if err := h.f.m.Join(h.ctx, newClient(userOf(0), "x", "P0"), "CMPAAB"); !errors.Is(err, room.ErrRoomArchived) {
		t.Fatalf("another rules version: %v", err)
	}
	data, _ = os.ReadFile(filepath.Join(h.f.store.ArchiveDir(), "CMPAAB.json"))
	_ = json.Unmarshal(data, &m)
	if m["archiveReason"] != string(store.ArchiveRulesVersion) {
		t.Fatalf("archive reason %v", m["archiveReason"])
	}

	for code, mutate := range map[string]func(map[string]any){
		"CMPAAC": func(m map[string]any) { m["schema"] = 3 },
		"CMPAAD": func(m map[string]any) { m["game"] = "unknown" },
		"CMPAAE": func(m map[string]any) { m["format"] = recordFormat + 1 },
	} {
		before := put(code, mutate)
		if err := h.f.m.Join(h.ctx, newClient(userOf(0), "x", "P0"), code); err == nil {
			t.Fatalf("%s loaded", code)
		}
		after, _ := os.ReadFile(filepath.Join(dir, code+".json"))
		if string(after) != string(before) {
			t.Fatalf("%s was rewritten", code)
		}
	}
}

// TestUpgradeFormatZero: the room actor's record from before the lobby,
// without a format, Start slots, previous seats, retired users, epochs or
// away times, loads explicitly upgraded, and its next write is the
// current format. One whose seat is still open is archived, not misread.
func TestUpgradeFormatZero(t *testing.T) {
	h := newTH(t, Config{}, soloSpec(3, 16, BotEasy))
	sa := h.snap(tabletest.SeatClient(0))
	h.expect(h.send(tabletest.SeatClient(0), 1, sa.Ref, sa.Legal[0], false), tabletest.Accepted, "a draft")
	base := h.record()
	h.p.leave()
	h.f.restart()
	old := func(code string, open bool) {
		r := base.clone()
		r.Code = code
		data, _ := r.encode()
		var m map[string]any
		_ = json.Unmarshal(data, &m)
		for _, k := range []string{"format", "start", "previous", "retired", "successor"} {
			delete(m, k)
		}
		roles := m["roles"].(map[string]any)
		delete(roles, "temporary")
		delete(roles, "away")
		for _, seat := range m["seats"].([]any) {
			delete(seat.(map[string]any), "taken")
			delete(seat.(map[string]any), "epoch")
		}
		if open {
			seat := m["seats"].([]any)[1].(map[string]any)
			seat["kind"], seat["bot"] = "open", ""
		}
		data, _ = json.Marshal(m)
		if err := os.WriteFile(filepath.Join(h.f.store.Dir(), code+".json"), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	old("UPGAAA", false)
	c := newClient(userOf(0), "x", "P0")
	if err := h.f.m.Join(h.ctx, c, "UPGAAA"); err != nil {
		t.Fatal(err)
	}
	if fr := lastFrame(t, c); fr.Pending == nil || fr.Pending.Key != sa.Legal[0] {
		t.Fatal("the upgraded table lost its pending pick")
	}
	a := h.f.tables.lookup("UPGAAA")
	var pick wire.TablePick
	if err := a.call(h.ctx, func() {
		pick = wire.TablePick{T: "table-pick", Code: "UPGAAA", Seq: 2, Point: *a.st.rec.Point, Key: sa.Legal[0], Confirm: true}
		a.submit(c, pick)
	}); err != nil {
		t.Fatal(err)
	}
	data, _, err := h.f.store.PeekGame("UPGAAA", Game)
	if err != nil {
		t.Fatal(err)
	}
	r, err := decodeRecord(data)
	if err != nil || r.Format != recordFormat || len(r.Start) != 3 || r.Start[0].UserID != userOf(0) || r.Start[1].Bot != BotEasy {
		t.Fatalf("the upgraded record: %v %+v", err, r)
	}
	h.f.m.Leave(c, "UPGAAA")
	old("UPGAAB", true)
	if err := h.f.m.Join(h.ctx, newClient(userOf(0), "x", "P0"), "UPGAAB"); !errors.Is(err, room.ErrRoomArchived) {
		t.Fatalf("an open seat: %v", err)
	}
}
