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
	b.Refill(now.Add(time.Second))
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

// TestBudgetCache: a room's spent buckets and high-water mark come back
// on its next load, once; what has refilled by then does not; an entry
// lasts until its last bucket is full, or ttl with a mark; a full budget
// with no mark leaves nothing; past its size the entry that would expire
// first goes, however late it came, so a nearly full entry cannot push
// out a deeply spent one.
func TestBudgetCache(t *testing.T) {
	c := NewBudgetCache(5, 3*time.Second)
	spent := func(at time.Time, users ...string) *Budget {
		b := NewBudget(3, time.Second)
		for _, u := range users {
			b.Charge(u, at)
		}
		return b
	}
	now := budgetT0
	c.Put("A", spent(now, "u", "u", "u", "v"), 7, now)
	// A second later v's bucket is full again and u's is not.
	b := NewBudget(3, time.Second)
	if got := c.Take("A", b, now.Add(time.Second)); got != 7 || len(b.full) != 1 || b.full["u"] != now.Add(3*time.Second) {
		t.Fatalf("taken back: reached %d, buckets %v", got, b.full)
	}
	if got := c.Take("A", NewBudget(3, time.Second), now.Add(time.Second)); got != 0 {
		t.Fatal("an entry taken twice")
	}
	// One token spent: gone once it is back, a second later.
	c.Put("B", spent(now, "u"), 0, now)
	if len(c.rooms) != 1 {
		t.Fatal("a spent budget was not kept")
	}
	c.Put("C", spent(now.Add(time.Second), "u"), 0, now.Add(time.Second))
	if _, ok := c.rooms["B"]; ok {
		t.Fatal("an entry outlived its last bucket")
	}
	c.Put("C", NewBudget(3, time.Second), 0, now)
	if len(c.rooms) != 0 {
		t.Fatal("a full budget with no mark was kept")
	}
	// A mark alone lasts ttl.
	c.Put("M", NewBudget(3, time.Second), 4, now)
	if got := c.Take("M", NewBudget(3, time.Second), now.Add(2*time.Second)); got != 4 {
		t.Fatalf("the mark two seconds later: %d", got)
	}
	// Size 5: D (two buckets and the room, full in three seconds) and E
	// (one, full in a second, though handed over later) fit; F (one, full
	// in two) pushes out E, which would expire first, not D.
	c.Put("D", spent(now, "u", "u", "u", "v"), 0, now)
	later := now.Add(500 * time.Millisecond)
	c.Put("E", spent(later, "u"), 0, later)
	c.Put("F", spent(later, "u", "u"), 0, later)
	_, d := c.rooms["D"]
	_, e := c.rooms["E"]
	if !d || e || c.held != 5 {
		t.Fatalf("after the size bound: D kept %v, E kept %v, %d held", d, e, c.held)
	}
}

// TestBudgetCacheOverflow: a room whose entry the cache dropped to keep
// within its size starts every user's bucket, known or not, full only
// from when the dropped buckets would all have been: the deepest debt it
// held. Once the notes of dropped entries overflow too, every room
// without an entry or a note starts from the latest dropped note. A new
// point's refill fills nothing until the floor has passed.
func TestBudgetCacheOverflow(t *testing.T) {
	const burst = 4
	c := NewBudgetCache(3, burst*time.Second)
	spent := func(at time.Time, n int, users ...string) *Budget {
		b := NewBudget(burst, time.Second)
		for _, u := range users {
			for range n {
				b.Charge(u, at)
			}
		}
		return b
	}
	tokens := func(b *Budget, u string, at time.Time) int {
		n := 0
		for b.Allows(u, at) {
			b.Charge(u, at)
			n++
		}
		return n
	}
	t0 := budgetT0
	// V: u spent its burst and v one token; W, spent later, pushes it out.
	v := spent(t0, burst, "u")
	v.Charge("v", t0)
	c.Put("V", v, 3, t0)
	c.Put("W", spent(t0.Add(time.Millisecond), burst, "w"), 0, t0.Add(time.Millisecond))
	if _, ok := c.rooms["V"]; ok {
		t.Fatal("V was not dropped")
	}
	at := t0.Add(time.Second)
	b := NewBudget(burst, time.Second)
	if got := c.Take("V", b, at); got != 3 {
		t.Fatalf("V's mark after the drop: %d", got)
	}
	for _, u := range []string{"u", "v", "x"} {
		if got := tokens(b, u, at); got != 1 {
			t.Fatalf("%s a second after V's whole burst: %d tokens, want 1", u, got)
		}
	}
	b = NewBudget(burst, time.Second)
	c.Take("V", b, at)
	if got := tokens(b, "u", at); got != burst {
		t.Fatalf("a note taken twice: %d tokens", got)
	}
	// Notes past the size: R0 to R2 are dropped in turn and noted, then R3
	// drops the note of W, which would have expired first.
	for i := range 4 {
		at := t0.Add(time.Duration(2+i) * time.Millisecond)
		c.Put(fmt.Sprintf("R%d", i), spent(at, burst, "u"), 0, at)
	}
	// W was full at t0 + 1 ms + burst seconds, so one token is back by a
	// second and 1 ms after its burst.
	late := t0.Add(time.Second + 2*time.Millisecond)
	for _, code := range []string{"W", "a room never seen"} {
		b := NewBudget(burst, time.Second)
		c.Take(code, b, late)
		if got := tokens(b, "u", late); got != 1 {
			t.Fatalf("%s after the notes overflowed: %d tokens, want 1", code, got)
		}
	}
	b = NewBudget(burst, time.Second)
	c.Take("a room never seen", b, t0.Add(burst*time.Second+time.Millisecond))
	if got := tokens(b, "u", t0.Add(burst*time.Second+time.Millisecond)); got != burst {
		t.Fatalf("once the dropped note would have expired: %d tokens", got)
	}
	// A refill under a standing floor fills nothing; once the floor has
	// passed it fills every bucket.
	b = NewBudget(burst, time.Second)
	mid := t0.Add(2 * time.Second)
	c.Take("R0", b, mid)
	if got := tokens(b, "u", mid); got != 1 {
		t.Fatalf("two seconds after R0's burst: %d tokens, want 1", got)
	}
	b.Refill(mid)
	if got := tokens(b, "u", mid); got != 0 {
		t.Fatalf("a refill under the floor: %d tokens, want 0", got)
	}
	past := t0.Add(burst*time.Second + 100*time.Millisecond)
	b.Refill(past)
	if got := tokens(b, "u", past); got != burst {
		t.Fatalf("a refill once the floor has passed: %d tokens", got)
	}
}

// TestCachePressureKeepsDebt: a user who spent their whole burst at a
// room that left memory keeps that debt however many other rooms' entries
// arrive (DefaultBudgetCacheSize, as served). Entries that are nearly full
// expire first and leave first; entries deeper than it push it out, and
// its note keeps the debt. Before, every entry lasted burst times interval
// from its hand-over, so 2048 later one-token entries pushed it out, and
// the load that missed started full (found by the slice's re-check).
func TestCachePressureKeepsDebt(t *testing.T) {
	for _, depth := range []int{1, DefaultRequestBurst} {
		t.Run(fmt.Sprintf("others spent %d", depth), func(t *testing.T) {
			now := budgetT0
			c := NewBudgetCache(DefaultBudgetCacheSize, DefaultRequestBurst*DefaultRequestInterval)
			b := NewBudget(DefaultRequestBurst, DefaultRequestInterval)
			for range DefaultRequestBurst {
				b.Charge("u", now)
			}
			c.Put("victim", b, 0, now)
			later := now.Add(time.Millisecond)
			crowd(c, DefaultBudgetCacheSize/2, depth, later)
			restored := NewBudget(DefaultRequestBurst, DefaultRequestInterval)
			c.Take("victim", restored, later)
			if restored.Allows("u", later) {
				t.Fatal("other rooms' entries gave an exhausted user a token")
			}
			second := now.Add(DefaultRequestInterval)
			if !restored.Allows("u", second) {
				t.Fatal("no token a second after the burst")
			}
			restored.Charge("u", second)
			if restored.Allows("u", second) {
				t.Fatal("two tokens a second after the burst")
			}
		})
	}
}

// crowd hands c n other rooms' entries, each one user's depth tokens
// spent at now.
func crowd(c *BudgetCache, n, depth int, now time.Time) {
	for i := range n {
		b := NewBudget(DefaultRequestBurst, DefaultRequestInterval)
		for range depth {
			b.Charge("u", now)
		}
		c.Put(fmt.Sprintf("crowd-%d", i), b, 0, now)
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
func TestUndoDoesNotRefillTheBudget(t *testing.T) { undoRefill(t, false) }

// TestEvictionKeepsTheHighWaterMark: nor can an eviction between the undo
// and the pick, which loads the table at fewer batches than it reached.
func TestEvictionKeepsTheHighWaterMark(t *testing.T) { undoRefill(t, true) }

func undoRefill(t *testing.T, evict bool) {
	work := faults.NewWork(true)
	clk := roomtest.NewFakeClock(budgetT0)
	h := newTHRooms(t, Config{Clock: clk, BotWork: work.Run}, soloSpec(3, 23, BotEasy), 1)
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
	// releases go to the first point's searches; so is the first point's
	// search, if the table is evicted.
	waitFor(t, "the cancelled search", func() bool { _, _, c := work.Counts(); return c > cancelled })
	fr = h.until(me, "the first point again", func(st wire.TableState) bool { return st.Batch == 0 && st.Point != nil })
	if evict {
		_, _, cancelled = work.Counts()
		h.evict(24)
		waitFor(t, "the search cancelled by the eviction", func() bool { _, _, c := work.Counts(); return c > cancelled })
		fr = h.until(me, "the first point, loaded again", func(st wire.TableState) bool { return st.Batch == 0 && st.Point != nil })
	}
	seqs[me]++
	h.pickSelf(int(seqs[me]), fr.View.You.Choices[0].Key)
	work.Release()
	work.Release()
	h.until(me, "the first batch again", func(st wire.TableState) bool { return st.Batch == 1 && st.Point != nil })
	s = h.snap(me)
	h.changes(1, s.Ref, [2]string{s.Legal[1], s.Legal[0]}, []tabletest.ClientID{me}, seqs, false)
	h.expect(h.send(me, seqs[me]+1, s.Ref, s.Legal[1], false), tabletest.Rate, "past the budget after the undo")
}

// newTHRooms is newTH with the Manager holding at most maxRooms rooms, so
// that making another table evicts this one (evict).
func newTHRooms(t *testing.T, cfg Config, spec Spec, maxRooms int) *th {
	t.Helper()
	ctx := testContext(t)
	f := newFixture(t, cfg)
	f.maxRooms = maxRooms
	f.restart()
	p, err := f.openInproc(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	return &th{t: t, ctx: ctx, f: f, p: p}
}

// evict pushes the table out of memory and joins the clients again,
// which loads it back.
func (h *th) evict(seed uint64) {
	h.t.Helper()
	h.push(seed)
	h.rejoin()
}

// push detaches every client and makes another table, which pushes this
// one out of memory (newTHRooms with 1).
func (h *th) push(seed uint64) {
	h.t.Helper()
	h.away()
	other, err := h.f.tables.CreateTable(humans(3, seed))
	if err != nil {
		h.t.Fatal(err)
	}
	// The new table tells the Manager it may be evicted from its own
	// goroutine; until it has, the rejoin finds no room to evict.
	if a := h.f.tables.lookup(other); a != nil {
		_ = a.call(h.ctx, func() {})
	}
	if h.f.m.Loaded(h.p.code) {
		h.t.Fatal("the table was not evicted")
	}
}

// away detaches every client, keeping them for rejoin, and waits until
// the actor has seen it, so the table is idle.
func (h *th) away() {
	h.t.Helper()
	h.p.leave()
	a, err := h.p.actor()
	if err == nil {
		err = a.call(h.ctx, func() {})
	}
	if err != nil {
		h.t.Fatal(err)
	}
}

// TestEvictionKeepsTheBudget: a table pushed out of memory and loaded
// again, however often, keeps its users' spent budgets, which refill with
// time as before; a restart of the server starts them full. Before, every
// load started every user full, so a user past the budget had it all back
// after an eviction, which needs only another room to be made (found by
// the slice's review).
func TestEvictionKeepsTheBudget(t *testing.T) {
	clk := roomtest.NewFakeClock(budgetT0)
	h := newTHRooms(t, Config{Clock: clk, BotWork: blockBots}, humans(3, 452), 1)
	me := tabletest.SeatClient(0)
	s := h.snap(me)
	keys := [2]string{s.Legal[0], s.Legal[1]}
	seqs := map[tabletest.ClientID]uint64{}
	h.changes(DefaultRequestBurst, s.Ref, keys, []tabletest.ClientID{me}, seqs, false)
	next := seqs[me] + 1
	for i := range 3 {
		h.expect(h.send(me, next, s.Ref, keys[0], false), tabletest.Rate, fmt.Sprintf("past the budget after %d evictions", i))
		h.evict(uint64(460 + i))
	}
	h.expect(h.send(me, next, s.Ref, keys[0], false), tabletest.Rate, "past the budget after 3 evictions")
	clk.Advance(DefaultRequestInterval)
	h.expect(h.send(me, next, s.Ref, keys[0], false), tabletest.Accepted, "a second later")
	h.expect(h.send(me, next+1, s.Ref, keys[1], false), tabletest.Rate, "with that token spent")
	h.f.restart()
	h.rejoin()
	h.expect(h.send(me, next+1, s.Ref, keys[1], false), tabletest.Accepted, "after a restart")
}

// TestCachePressureKeepsTheBudget: a table evicted with a user past the
// budget, whose entry the cache then drops for 2048 other rooms' deeper
// debts, still refuses that user when loaded again, and gives back one
// token a second as before. Before, the load that found no entry started
// every user full (found by the slice's re-check).
func TestCachePressureKeepsTheBudget(t *testing.T) {
	clk := roomtest.NewFakeClock(budgetT0)
	h := newTHRooms(t, Config{Clock: clk, BotWork: blockBots}, humans(3, 457), 1)
	me := tabletest.SeatClient(0)
	s := h.snap(me)
	keys := [2]string{s.Legal[0], s.Legal[1]}
	seqs := map[tabletest.ClientID]uint64{}
	h.changes(DefaultRequestBurst, s.Ref, keys, []tabletest.ClientID{me}, seqs, false)
	next := seqs[me] + 1
	h.push(458)
	clk.Advance(time.Millisecond)
	crowd(h.f.tables.budgets, DefaultBudgetCacheSize/2, DefaultRequestBurst, clk.Now())
	h.rejoin()
	h.expect(h.send(me, next, s.Ref, keys[0], false), tabletest.Rate, "past the budget after the cache overflowed")
	clk.Advance(DefaultRequestInterval)
	h.expect(h.send(me, next, s.Ref, keys[0], false), tabletest.Accepted, "a second later")
	h.expect(h.send(me, next+1, s.Ref, keys[1], false), tabletest.Rate, "with that token spent")
}

// TestIdleUnloadKeepsTheBudget: a table that leaves memory when idle keeps
// its spent budgets too, less what refilled meanwhile.
func TestIdleUnloadKeepsTheBudget(t *testing.T) {
	clk := roomtest.NewFakeClock(budgetT0)
	h := newTH(t, Config{Clock: clk, BotWork: blockBots, IdleTimeout: 2 * DefaultRequestInterval}, humans(3, 456))
	me := tabletest.SeatClient(0)
	s := h.snap(me)
	keys := [2]string{s.Legal[0], s.Legal[1]}
	seqs := map[tabletest.ClientID]uint64{}
	h.changes(DefaultRequestBurst, s.Ref, keys, []tabletest.ClientID{me}, seqs, false)
	h.away()
	clk.Advance(2 * DefaultRequestInterval)
	waitFor(t, "the idle table to leave memory", func() bool { return !h.f.m.Loaded(h.p.code) })
	h.rejoin()
	next := seqs[me] + 1
	h.expect(h.send(me, next, s.Ref, keys[0], false), tabletest.Accepted, "the first token back")
	h.expect(h.send(me, next+1, s.Ref, keys[1], false), tabletest.Accepted, "the second token back")
	h.expect(h.send(me, next+2, s.Ref, keys[0], false), tabletest.Rate, "a third, two seconds after the burst")
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
