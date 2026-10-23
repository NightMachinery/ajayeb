package tableroom

import (
	"slices"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/faults"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// soloSpec is one human (lobby slot 0, seat 0 under chosen seating) and
// n-1 bots of the given level.
func soloSpec(n int, seed uint64, level string) Spec {
	sp := Spec{Seed: &seed, Creator: userOf(0), AllowUndo: true}
	sp.Participants = append(sp.Participants, Participant{UserID: userOf(0), Name: "P0"})
	for range n - 1 {
		sp.Participants = append(sp.Participants, Participant{Bot: level})
	}
	return sp
}

// until waits for client id's latest frame to satisfy ok, draining frames
// as they arrive. It never sleeps on a guess: it wakes on the outbox.
func (h *th) until(id tabletest.ClientID, what string, ok func(wire.TableState) bool) wire.TableState {
	h.t.Helper()
	c := h.p.clients[id]
	deadline := time.After(wait)
	for {
		h.p.drain()
		if fs := h.p.frames[id]; len(fs) > 0 && ok(fs[len(fs)-1]) {
			return fs[len(fs)-1]
		}
		select {
		case <-c.Out.Ready():
		case <-deadline:
			h.t.Fatalf("waiting for %s", what)
		}
	}
}

func (h *th) latest(id tabletest.ClientID) wire.TableState {
	h.p.drain()
	fs := h.p.frames[id]
	return fs[len(fs)-1]
}

func (h *th) pickSelf(seq int, key string) {
	h.t.Helper()
	a, err := h.p.actor()
	if err != nil {
		h.t.Fatal(err)
	}
	fr := h.latest(tabletest.SeatClient(0))
	pick := wire.TablePick{T: "table-pick", Code: h.p.code, Seq: seq, Point: fr.Point.Ref, Key: key, Confirm: true}
	if err := a.call(h.ctx, func() { a.submit(h.p.clients[tabletest.SeatClient(0)], pick) }); err != nil {
		h.t.Fatal(err)
	}
}

func (h *th) undo(seq, base int) string {
	h.t.Helper()
	a, err := h.p.actor()
	if err != nil {
		h.t.Fatal(err)
	}
	var out string
	u := wire.TableUndo{T: "table-undo", Code: h.p.code, Seq: seq, Base: base}
	if err := a.call(h.ctx, func() { out = a.undo(h.p.clients[tabletest.SeatClient(0)], u) }); err != nil {
		h.t.Fatal(err)
	}
	h.p.drain()
	return out
}

// TestBotsPlayAGame: one human against bots at 3 to 7 seats; the bots pick
// through the pool, each confirmed, and the game ends.
func TestBotsPlayAGame(t *testing.T) {
	for n := MinSeats; n <= MaxSeats; n++ {
		pool := ai.NewPool(1)
		h := newTH(t, Config{Pool: pool}, soloSpec(n, uint64(40+n), DefaultBotLevel))
		me := tabletest.SeatClient(0)
		for seq := 1; ; seq++ {
			fr := h.until(me, "my turn or the end", func(st wire.TableState) bool {
				return st.Status == "finished" || (st.Point != nil && slices.Contains(st.Point.Required, 0) && st.Pending == nil)
			})
			if fr.Status == "finished" {
				break
			}
			if seq > 50 {
				t.Fatal("the game did not end")
			}
			h.pickSelf(seq, fr.View.You.Choices[0].Key)
		}
		if pool.InUse() != 0 || pool.Waiting() != 0 {
			t.Fatalf("%d seats: the pool holds %d, %d waiting", n, pool.InUse(), pool.Waiting())
		}
		r := h.record()
		if r.Status != "finished" || len(r.Work) != 0 || r.Point != nil {
			t.Fatalf("%d seats: finished record %+v", n, r.Status)
		}
	}
}

// TestNoHumanPause: a table whose only human is away runs no bot work, a
// spectator does not change that, the work starts when the human attaches
// and is cancelled when they leave, without leaking a pool permit. A
// restart restores the bots' accepted picks and their frozen work rows.
func TestNoHumanPause(t *testing.T) {
	work := faults.NewWork(true)
	pool := ai.NewPool(1)
	ctx := testContext(t)
	f := newFixture(t, Config{Pool: pool, BotWork: work.Run})
	code, err := f.tables.CreateTable(soloSpec(4, 21, BotNormal))
	if err != nil {
		t.Fatal(err)
	}
	p := &inproc{f: f, code: code, clients: map[tabletest.ClientID]*room.Client{}, frames: map[tabletest.ClientID][]wire.TableState{}, rejects: map[tabletest.ClientID][]wire.TableReject{}}
	h := &th{t: t, ctx: ctx, f: f, p: p}
	busy := func() bool {
		a, err := p.actor()
		if err != nil {
			t.Fatal(err)
		}
		var b bool
		if err := a.call(ctx, func() { b = a.botBusy }); err != nil {
			t.Fatal(err)
		}
		return b
	}
	if err := p.join(ctx, tabletest.SpectatorClient, "watcher", "Watcher"); err != nil {
		t.Fatal(err)
	}
	if started, _, _ := work.Counts(); started != 0 || busy() {
		t.Fatal("a spectator alone started bot work")
	}
	if err := p.join(ctx, tabletest.SeatClient(0), userOf(0), "P0"); err != nil {
		t.Fatal(err)
	}
	first := <-work.Started()
	f.m.Leave(p.clients[tabletest.SeatClient(0)], code)
	waitFor(t, "the cancellation", func() bool { _, _, c := work.Counts(); return c == 1 })
	waitFor(t, "the permit", func() bool { return pool.InUse() == 0 && pool.Waiting() == 0 })
	if busy() {
		t.Fatal("a bot job survived the last human leaving")
	}
	if err := p.join(ctx, tabletest.SeatClient(0), userOf(0), "P0"); err != nil {
		t.Fatal(err)
	}
	again := <-work.Started()
	if again != first {
		t.Fatalf("the work resumed for %+v, then %+v", first, again)
	}
	work.Release()
	h.until(tabletest.SeatClient(0), "the first bot's confirmation", func(st wire.TableState) bool {
		return st.Point != nil && len(st.Point.Confirmed) == 1
	})
	<-work.Started() // the second bot's search holds now
	rows := h.record().Work
	if len(rows) != 3 || rows[0].Level != BotNormal || rows[0].Allowance != 2 || rows[0].Policy != policyVersion {
		t.Fatalf("work rows %+v", rows)
	}
	f.restart()
	h.rejoin()
	r := h.record()
	if !slices.Equal(workSeats(r.Work), workSeats(rows)) || len(r.Pending) != 1 || !r.Pending[0].Confirmed || r.Pending[0].Seat != first.Seat {
		t.Fatalf("after a restart: work %+v, pending %+v", r.Work, r.Pending)
	}
	if next := <-work.Started(); next.Seat == first.Seat {
		t.Fatalf("the restored table searched again for seat %d, whose pick it had accepted", next.Seat)
	}
}

func workSeats(rows []WorkRow) []int {
	var out []int
	for _, w := range rows {
		out = append(out, w.Seat)
	}
	return out
}

func waitFor(t *testing.T, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(wait)
	for !ok() {
		if time.Now().After(deadline) {
			t.Fatalf("waiting for %s", what)
		}
		time.Sleep(time.Millisecond)
	}
}

// TestUndo: the original sole human undoes back before their last decided
// batch; the reopened point has a fresh reference, so a stale submission
// is decision-closed; a stale base, a duplicate seq, a spectator, a
// second starting human and the undo-off flag are each refused.
func TestUndo(t *testing.T) {
	work := faults.NewWork(true)
	h := newTH(t, Config{BotWork: work.Run}, soloSpec(3, 23, BotEasy))
	me := tabletest.SeatClient(0)
	fr := h.until(me, "the first point", func(st wire.TableState) bool { return st.Point != nil })
	if !fr.AllowUndo || fr.StartingHumanCount != 1 || fr.UndoOff {
		t.Fatalf("undo policy in the frame: %+v", fr)
	}
	oldRef := fr.Point.Ref
	h.pickSelf(1, fr.View.You.Choices[0].Key)
	work.Release() // seat 1
	work.Release() // seat 2: the batch resolves; seat 1's next search then holds
	fr = h.until(me, "the first batch", func(st wire.TableState) bool { return st.Batch == 1 })
	if got := h.undo(2, fr.Revision-1); got != outStale {
		t.Fatalf("a stale base: %s", got)
	}
	if got := h.undo(3, fr.Revision); got != outAccepted {
		t.Fatalf("undo: %s", got)
	}
	fr = h.latest(me)
	if fr.Batch != 0 || fr.History == nil || len(*fr.History) != 0 || fr.Point.Ref == oldRef || fr.Point.Ref.ID != oldRef.ID || fr.Ack == nil || fr.Ack.Seq != 3 {
		t.Fatalf("after undo: %+v", fr)
	}
	if sp := h.latest(tabletest.SpectatorClient); sp.History == nil || sp.Batch != 0 {
		t.Fatal("the spectator's undo frame carries no history")
	}
	if got := h.undo(3, fr.Revision); got != outDuplicate {
		t.Fatalf("the undo seq again: %s", got)
	}
	o, err := h.p.Submit(h.ctx, me, tabletest.Submission{Seq: 4, Ref: refString(oldRef), Key: fr.View.You.Choices[0].Key, Confirm: true})
	if err != nil || o != tabletest.Closed {
		t.Fatalf("a submission for the undone opening: %s %v", o, err)
	}
	if got := h.undo(5, h.latest(me).Revision); got != outIllegal {
		t.Fatalf("undo with nothing to undo: %s", got)
	}
	a, _ := h.p.actor()
	_ = a.call(h.ctx, func() { a.st.rec.UndoOff = true })
	if got := h.undo(6, h.latest(me).Revision); got != outForbidden {
		t.Fatalf("undo switched off: %s", got)
	}
	// Two starting humans: nobody may undo.
	two := newTH(t, Config{}, humans(3, 24))
	if got := two.undo(1, two.latest(tabletest.SeatClient(0)).Revision); got != outForbidden {
		t.Fatalf("undo with two starting humans: %s", got)
	}
}
