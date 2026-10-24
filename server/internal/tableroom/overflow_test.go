package tableroom

import (
	"fmt"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/room/roomtest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/faults"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// TestArrivingEntryExpiringFirstIsNoted: when the cache is full, an
// arriving entry that would expire before every cached one is the one
// noted; it does not push out a deeper entry. Before, only cached entries
// were compared (found by the slice's second re-check).
func TestArrivingEntryExpiringFirstIsNoted(t *testing.T) {
	now := budgetT0
	c := NewBudgetCache(2, 16*time.Second)
	deep := NewBudget(16, time.Second)
	for range 16 {
		deep.Charge("u", now)
	}
	c.Put("deep", deep, 0, now)
	later := now.Add(time.Millisecond)
	shallow := NewBudget(16, time.Second)
	shallow.Charge("u", later)
	c.Put("shallow", shallow, 0, later)
	if _, ok := c.rooms["deep"]; !ok {
		t.Fatal("an arriving one-token entry pushed out a sixteen-token one")
	}
	b := NewBudget(16, time.Second)
	c.Take("shallow", b, later)
	n := 0
	for b.Allows("u", later) {
		b.Charge("u", later)
		n++
	}
	if n != 15 {
		t.Fatalf("the noted room's user, who spent one token: %d tokens, want 15", n)
	}
}

// TestNoteOverflowCannotRefillAnOldPoint: at a table with undo, the human
// spends 13 tokens and undoes (14). The table is evicted, and the cache
// overflows its entries and then its notes, so the table's note, and with
// it its high-water mark, is gone. Loaded again two seconds later, the
// table takes the dropped note's time as its floor and does not know the
// point it completes again is old; the floor holds through that point's
// refill, so the human has at most the 3 requests keeping the entry would
// have left. Before, the refill cleared the floor and gave a fresh burst
// (found by the slice's second re-check).
func TestNoteOverflowCannotRefillAnOldPoint(t *testing.T) {
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
	// The second point's search is cancelled before any release, and the
	// first point's search by the eviction, so both releases below go to
	// the first point's searches once it is loaded again.
	waitFor(t, "the cancelled search", func() bool { _, _, c := work.Counts(); return c > cancelled })
	started, _, _ := work.Counts()
	h.until(me, "the first point again", func(st wire.TableState) bool { return st.Batch == 0 && st.Point != nil })
	// The first point's search starts only once the cancelled one has
	// reported back, which may come after the state; evicted before it
	// starts, the table would have no search to cancel.
	waitFor(t, "the first point's search", func() bool { s, _, _ := work.Counts(); return s > started })
	_, _, cancelled = work.Counts()
	h.push(24)
	// Every filler is due to refill after the table's entry, so the table's
	// entry is dropped first, and then its note.
	later := clk.Now().Add(time.Millisecond)
	for i := range DefaultBudgetCacheSize + DefaultBudgetCacheSize/2 + 2 {
		b := NewBudget(DefaultRequestBurst, DefaultRequestInterval)
		for range DefaultRequestBurst {
			b.Charge("u", later)
		}
		h.f.tables.budgets.Put(fmt.Sprintf("filler-%d", i), b, 0, later)
	}
	clk.Advance(2 * time.Second)
	h.rejoin()
	waitFor(t, "the search cancelled by the eviction", func() bool { _, _, c := work.Counts(); return c > cancelled })
	fr = h.until(me, "the first point, loaded again", func(st wire.TableState) bool { return st.Batch == 0 && st.Point != nil })
	seqs[me]++
	h.pickSelf(int(seqs[me]), fr.View.You.Choices[0].Key)
	work.Release()
	work.Release()
	h.until(me, "the first batch again", func(st wire.TableState) bool { return st.Batch == 1 && st.Point != nil })
	s = h.snap(me)
	accepted := 0
	for i := range 4 {
		seqs[me]++
		if h.send(me, seqs[me], s.Ref, s.Legal[i%2], false) == tabletest.Accepted {
			accepted++
		}
	}
	if accepted > 3 {
		t.Fatalf("after the notes overflowed, the old point's refill allowed %d more requests; keeping the entry allows at most 3", accepted)
	}
}
