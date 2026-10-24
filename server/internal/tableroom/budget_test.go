package tableroom

import (
	"bytes"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/room/roomtest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/faults"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

var budgetT0 = time.Date(2026, 10, 24, 15, 0, 0, 0, time.UTC)

// TestBudget: a user's burst is taken at once, then one token comes back
// each interval; asking takes nothing; users do not share buckets; Refill
// fills every bucket; full buckets are forgotten.
func TestBudget(t *testing.T) {
	b := NewBudget(3, time.Second)
	now := budgetT0
	spend := func(u string, at time.Time) bool {
		if !b.Allows(u, at) {
			return false
		}
		b.Charge(u, at)
		return true
	}
	for i := range 3 {
		if !b.Allows("u", now) || !spend("u", now) {
			t.Fatalf("token %d of the burst refused", i)
		}
	}
	if spend("u", now) || !spend("v", now) {
		t.Fatal("past the burst, or another user's bucket")
	}
	if spend("u", now.Add(999*time.Millisecond)) || !spend("u", now.Add(time.Second)) || spend("u", now.Add(time.Second)) {
		t.Fatal("one token per interval")
	}
	b.Refill()
	for i := range 3 {
		if !spend("u", now.Add(time.Second)) {
			t.Fatalf("token %d after a refill refused", i)
		}
	}
	spend("w", now.Add(time.Hour))
	if len(b.full) != 1 {
		t.Fatalf("%d buckets remembered, want only the one that is not full", len(b.full))
	}
}

// changes sends n private pick changes for seat's user, alternating
// between two keys and between the clients ids, and expects each accepted.
// It returns the next seq of each client.
func (h *th) changes(n int, ref string, keys [2]string, ids []tabletest.ClientID, seqs map[tabletest.ClientID]uint64, confirmLast bool) {
	h.t.Helper()
	for i := range n {
		id := ids[i%len(ids)]
		seqs[id]++
		confirm := confirmLast && i == n-1
		h.expect(h.send(id, seqs[id], ref, keys[i%2], confirm), tabletest.Accepted, fmt.Sprintf("change %d", i+1))
	}
}

// TestUserBudgetIsShared: a user's requests, from all of their tabs, draw
// on one budget. Past it a request is refused with rate and changes
// nothing: the record is not written, no one else gets a frame, and the
// seq is not used up, so the same request is accepted once a token has
// come back. Another user's budget is their own. Before
// (M16.table-request-budget), nothing but each connection's frame rate
// bounded a user's writes and broadcasts.
func TestUserBudgetIsShared(t *testing.T) {
	clk := roomtest.NewFakeClock(budgetT0)
	h := newTH(t, Config{Clock: clk, BotWork: blockBots}, humans(3, 401))
	c1, tab := tabletest.SeatClient(1), tabletest.ClientID("tab-1")
	h.joinAs(tab, userOf(1))
	s := h.snap(c1)
	keys := [2]string{s.Legal[0], s.Legal[1]}
	seqs := map[tabletest.ClientID]uint64{}
	h.changes(DefaultRequestBurst, s.Ref, keys, []tabletest.ClientID{c1, tab}, seqs, false)
	raw, pick := h.raw(), h.snap(c1).Pick
	quiet := h.quiet(tabletest.SeatClient(0), tabletest.SeatClient(2), tabletest.SpectatorClient)
	for _, id := range []tabletest.ClientID{c1, tab} {
		h.expect(h.send(id, seqs[id]+1, s.Ref, keys[0], true), tabletest.Rate, "a request past the budget from "+string(id))
	}
	if !bytes.Equal(raw, h.raw()) || h.snap(c1).Pick != pick {
		t.Fatal("a request past the budget changed the table")
	}
	quiet()
	c2 := tabletest.SeatClient(2)
	h.expect(h.send(c2, 1, s.Ref, h.snap(c2).Legal[0], false), tabletest.Accepted, "another user's request")
	clk.Advance(DefaultRequestInterval)
	h.expect(h.send(c1, seqs[c1]+1, s.Ref, keys[0], true), tabletest.Accepted, "the refused request, a second later")
	h.expect(h.send(tab, seqs[tab]+1, s.Ref, keys[1], false), tabletest.Rate, "the other tab, with the token spent")
}

// TestEveryChangeDrawsOnOneBudget: a relink, seat switches, a role change
// and a pick all draw on the sender's one budget; past it a seat switch is
// refused with rate, and neither the record nor anyone's frames change.
func TestEveryChangeDrawsOnOneBudget(t *testing.T) {
	clk := roomtest.NewFakeClock(budgetT0)
	h := newTH(t, Config{Clock: clk, BotWork: blockBots}, humans(3, 402))
	admin := tabletest.SeatClient(0)
	if out, _ := h.linkOf(admin, 0, true); out != outAccepted {
		t.Fatalf("the admin's relink of their seat: %s", out)
	}
	levels := [2]string{BotEasy, BotNormal}
	seq := 0
	for i := range DefaultRequestBurst - 3 {
		seq++
		h.want(admin, toBot(seq, 1, levels[i%2]), outAccepted, "a seat switch")
	}
	seq++
	h.want(admin, roleOp(seq, "promote", "watcher"), outAccepted, "a promotion")
	s := h.snap(admin)
	seq++
	h.expect(h.send(admin, uint64(seq), s.Ref, s.Legal[0], false), tabletest.Accepted, "a pick")
	raw := h.raw()
	quiet := h.quiet(tabletest.SeatClient(1), tabletest.SeatClient(2), tabletest.SpectatorClient)
	seq++
	if out := h.do(admin, toBot(seq, 1, levels[(DefaultRequestBurst-3)%2])); out != outRate {
		t.Fatalf("a seat switch past the budget: %s", out)
	}
	if !bytes.Equal(raw, h.raw()) {
		t.Fatal("a refused switch changed the table")
	}
	quiet()
}

// TestNewPointRefillsTheBudget: a point the game reaches for the first
// time fills every budget, so a player who spent theirs at one point has
// all of it at the next, whatever the clock says.
func TestNewPointRefillsTheBudget(t *testing.T) {
	clk := roomtest.NewFakeClock(budgetT0)
	h := newTH(t, Config{Clock: clk, BotWork: blockBots}, humans(3, 403))
	c1 := tabletest.SeatClient(1)
	s := h.snap(c1)
	seqs := map[tabletest.ClientID]uint64{}
	h.changes(DefaultRequestBurst, s.Ref, [2]string{s.Legal[0], s.Legal[1]}, []tabletest.ClientID{c1}, seqs, true)
	for _, id := range []tabletest.ClientID{tabletest.SeatClient(0), tabletest.SeatClient(2)} {
		sn := h.snap(id)
		h.send(id, 1, sn.Ref, sn.Legal[0], true)
	}
	next := h.snap(c1)
	if next.Ref == s.Ref {
		t.Fatal("the point did not resolve")
	}
	h.changes(DefaultRequestBurst, next.Ref, [2]string{next.Legal[0], next.Legal[1]}, []tabletest.ClientID{c1}, seqs, false)
	h.expect(h.send(c1, seqs[c1]+1, next.Ref, next.Legal[0], false), tabletest.Rate, "past the new point's budget")
}

// TestUndoDoesNotRefillTheBudget: reaching a point again after an undo is
// not a new point, so an undo and a pick cannot be cycled to refill the
// budget.
func TestUndoDoesNotRefillTheBudget(t *testing.T) {
	work := faults.NewWork(true)
	clk := roomtest.NewFakeClock(budgetT0)
	h := newTH(t, Config{Clock: clk, BotWork: work.Run}, soloSpec(3, 23, BotEasy))
	me := tabletest.SeatClient(0)
	fr := h.until(me, "the first point", func(st wire.TableState) bool { return st.Point != nil })
	h.pickSelf(1, fr.View.You.Choices[0].Key)
	work.Release()
	work.Release()
	h.until(me, "the first batch", func(st wire.TableState) bool { return st.Batch == 1 })
	// The budget is full again at the second point: spend 13, then undo.
	s := h.snap(me)
	seqs := map[tabletest.ClientID]uint64{me: 1}
	h.changes(DefaultRequestBurst-3, s.Ref, [2]string{s.Legal[0], s.Legal[1]}, []tabletest.ClientID{me}, seqs, false)
	_, _, cancelled := work.Counts()
	seqs[me]++
	if out := h.undo(int(seqs[me]), h.latest(me).Revision); out != outAccepted {
		t.Fatalf("the undo: %s", out)
	}
	// The second point's search is cancelled before any release, so both
	// releases go to the first point's searches.
	waitFor(t, "the cancelled search", func() bool { _, _, c := work.Counts(); return c > cancelled })
	fr = h.until(me, "the first point again", func(st wire.TableState) bool { return st.Batch == 0 && st.Point != nil })
	seqs[me]++
	h.pickSelf(int(seqs[me]), fr.View.You.Choices[0].Key)
	work.Release()
	work.Release()
	h.until(me, "the first batch again", func(st wire.TableState) bool { return st.Batch == 1 && st.Point != nil })
	s = h.snap(me)
	h.changes(1, s.Ref, [2]string{s.Legal[1], s.Legal[0]}, []tabletest.ClientID{me}, seqs, false)
	h.expect(h.send(me, seqs[me]+1, s.Ref, s.Legal[1], false), tabletest.Rate, "past the budget after the undo")
}

// TestRefusedRequestsResolveNothing: a request past the budget is refused
// before the engine works on it. A seat switch that hands a bot a pick
// completing the batch does not resolve it, and neither does a completing
// confirmation; a second later each resolves. Before, both resolved the
// batch and threw it away (found by the slice's review).
func TestRefusedRequestsResolveNothing(t *testing.T) {
	var resolved atomic.Int64
	clk := roomtest.NewFakeClock(budgetT0)
	h := newTH(t, Config{Clock: clk, BotWork: blockBots, Resolving: func() { resolved.Add(1) }}, humans(3, 454))
	admin, c1, c2 := tabletest.SeatClient(0), tabletest.SeatClient(1), tabletest.SeatClient(2)
	want := func(n int64, what string) {
		t.Helper()
		if got := resolved.Load(); got != n {
			t.Fatalf("%s: %d resolutions, want %d", what, got, n)
		}
	}
	seqs := map[tabletest.ClientID]uint64{}
	next := func(id tabletest.ClientID) uint64 { seqs[id]++; return seqs[id] }

	// Seat 1 confirms and seat 2 drafts; the admin spends the budget on its
	// own pick, ending confirmed. A bot in seat 2 would confirm the draft.
	s1, s2 := h.snap(c1), h.snap(c2)
	h.expect(h.send(c1, next(c1), s1.Ref, s1.Legal[0], true), tabletest.Accepted, "seat 1 confirms")
	h.expect(h.send(c2, next(c2), s2.Ref, s2.Legal[0], false), tabletest.Accepted, "seat 2's draft")
	s := h.snap(admin)
	h.changes(DefaultRequestBurst, s.Ref, [2]string{s.Legal[0], s.Legal[1]}, []tabletest.ClientID{admin}, seqs, true)
	seq := int(seqs[admin]) + 1
	for i := range 3 {
		if out := h.do(admin, toBot(seq, 2, BotEasy)); out != outRate {
			t.Fatalf("switch %d past the budget: %s", i, out)
		}
	}
	want(0, "refused switches")
	clk.Advance(DefaultRequestInterval)
	if out := h.do(admin, toBot(seq, 2, BotEasy)); out != outResolved {
		t.Fatalf("the switch a second later: %s", out)
	}
	want(1, "the accepted switch")
	seqs[admin]++

	// At the next point, every budget full again: user-2 back in seat 2,
	// seats 1 and 2 confirm, and the admin's changes leave its
	// confirmation, the completing one, past the budget.
	h.want(admin, toHuman(int(next(admin)), 2, userOf(2)), outAccepted, "user-2 back in seat 2")
	s1, s2, s = h.snap(c1), h.snap(c2), h.snap(admin)
	h.expect(h.send(c1, next(c1), s1.Ref, s1.Legal[0], true), tabletest.Accepted, "seat 1 confirms")
	h.expect(h.send(c2, next(c2), s2.Ref, s2.Legal[0], true), tabletest.Accepted, "seat 2 confirms")
	h.changes(DefaultRequestBurst-1, s.Ref, [2]string{s.Legal[0], s.Legal[1]}, []tabletest.ClientID{admin}, seqs, false)
	for i := range 3 {
		h.expect(h.send(admin, seqs[admin]+1, s.Ref, s.Legal[0], true), tabletest.Rate, fmt.Sprintf("completing confirmation %d past the budget", i))
	}
	want(1, "refused confirmations")
	clk.Advance(DefaultRequestInterval)
	h.expect(h.send(admin, seqs[admin]+1, s.Ref, s.Legal[0], true), tabletest.Resolved, "the confirmation a second later")
	want(2, "the accepted confirmation")
}
