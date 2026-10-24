package lobbyroom

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/room/roomtest"
	"github.com/NightMachinery/ajayeb/server/internal/tableroom"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// raw is the lobby's stored record, byte for byte.
func (f *fixture) raw(code string) []byte {
	f.t.Helper()
	data, _, err := f.st.PeekGame(code, Game)
	if err != nil {
		f.t.Fatal(err)
	}
	return data
}

// frameCount is how many frames k has received.
func (k *conn) frameCount() int {
	k.drain()
	return len(k.frames)
}

// TestLobbyBudgetIsShared: an admin's edits, from all of their tabs, draw
// on one budget. Past it an edit is refused with rate and changes nothing:
// the record is not written and no one else gets a frame; once a token has
// come back the same seq is accepted. Before (M16.table-request-budget),
// nothing but each connection's frame rate bounded a user's writes.
func TestLobbyBudgetIsShared(t *testing.T) {
	clk := roomtest.NewFakeClock(time.Date(2026, 10, 24, 15, 0, 0, 0, time.UTC))
	f := newFixture(t, clk)
	admin, tab, other := f.conn(user(0), "c0", "P0"), f.conn(user(0), "c0-tab", "P0"), f.conn(user(1), "c1", "P1")
	code := f.create(admin, 4, wire.LobbyOptions{Seating: ptr("lobby")})
	f.join(tab, code)
	f.join(other, code)
	// Seat 3 is open: the two tabs add a bot there and remove it in turn.
	toggle := func(seq, i int) wire.LobbyEdit { return toggleBot(code, seq, i) }
	ks := [2]*conn{admin, tab}
	seqs := [2]int{}
	for i := range tableroom.DefaultRequestBurst {
		k := ks[i%2]
		seqs[i%2]++
		f.send(k, toggle(seqs[i%2], i))
		k.acked(seqs[i%2])
	}
	f.settle(code)
	raw, frames := f.raw(code), other.frameCount()
	for j, k := range ks {
		f.send(k, toggle(seqs[j]+1, tableroom.DefaultRequestBurst))
		if got := k.reject(seqs[j] + 1); got != outRate {
			t.Fatalf("an edit past the budget from %s: %s", k.c.ClientID, got)
		}
	}
	f.settle(code)
	if !bytes.Equal(raw, f.raw(code)) || other.frameCount() != frames {
		t.Fatal("an edit past the budget changed the lobby")
	}
	clk.Advance(tableroom.DefaultRequestInterval)
	f.send(admin, toggle(seqs[0]+1, tableroom.DefaultRequestBurst))
	if st := admin.acked(seqs[0] + 1); st.Seats[3].Kind != SlotBot {
		t.Fatalf("the refused edit a second later: %v", kinds(st))
	}
}

// toggleBot is edit seq: the i-th of a series that adds a bot to seat 3
// and removes it in turn.
func toggleBot(code string, seq, i int) wire.LobbyEdit {
	e := edit(code, seq, "remove-bot")
	if i%2 == 0 {
		e.Op, e.Bot = "add-bot", ptr(tableroom.BotEasy)
	}
	e.Seat = ptr(3)
	return e
}

// TestLobbyEvictionKeepsTheBudget: a lobby pushed out of memory and
// loaded again, twice, keeps its admin's spent budget, which refills with
// time as before. Before, every load started every user full (found by
// the slice's review).
func TestLobbyEvictionKeepsTheBudget(t *testing.T) {
	clk := roomtest.NewFakeClock(time.Date(2026, 10, 24, 15, 0, 0, 0, time.UTC))
	f := newFixture(t, clk)
	f.maxRooms = 1
	f.restart()
	admin := f.conn(user(0), "c0", "P0")
	code := f.create(admin, 4, wire.LobbyOptions{Seating: ptr("lobby")})
	for i := range tableroom.DefaultRequestBurst {
		f.send(admin, toggleBot(code, i+1, i))
		admin.acked(i + 1)
	}
	seq := tableroom.DefaultRequestBurst + 1
	for cycle := range 2 {
		f.send(admin, toggleBot(code, seq, seq-1))
		if got := admin.reject(seq); got != outRate {
			t.Fatalf("past the budget after %d evictions: %s", cycle, got)
		}
		f.m.Leave(admin.c, code)
		f.settle(code)
		other := tableroom.Spec{Creator: "x", Participants: []tableroom.Participant{{UserID: "x", Name: "X"}, {Bot: tableroom.BotEasy}, {Bot: tableroom.BotEasy}}}
		if _, err := f.tables.CreateTable(other); err != nil {
			t.Fatal(err)
		}
		if f.m.Loaded(code) {
			t.Fatal("the lobby was not evicted")
		}
		admin = f.conn(user(0), "c0", "P0")
		f.joinEvicting(admin, code)
		admin.lobby("the lobby, loaded again", any2)
	}
	f.send(admin, toggleBot(code, seq, seq-1))
	if got := admin.reject(seq); got != outRate {
		t.Fatalf("past the budget after 2 evictions: %s", got)
	}
	clk.Advance(tableroom.DefaultRequestInterval)
	f.send(admin, toggleBot(code, seq, seq-1))
	admin.acked(seq)
}

// TestLobbyCachePressureKeepsTheBudget: a lobby evicted with its admin
// past the budget, whose entry the cache then drops for 2048 other rooms'
// deeper debts, still refuses the admin when loaded again, and gives back
// one token a second as before. Before, the load that found no entry
// started every user full (found by the slice's re-check).
func TestLobbyCachePressureKeepsTheBudget(t *testing.T) {
	clk := roomtest.NewFakeClock(time.Date(2026, 10, 24, 15, 0, 0, 0, time.UTC))
	f := newFixture(t, clk)
	f.maxRooms = 1
	f.restart()
	admin := f.conn(user(0), "c0", "P0")
	code := f.create(admin, 4, wire.LobbyOptions{Seating: ptr("lobby")})
	for i := range tableroom.DefaultRequestBurst {
		f.send(admin, toggleBot(code, i+1, i))
		admin.acked(i + 1)
	}
	seq := tableroom.DefaultRequestBurst + 1
	f.m.Leave(admin.c, code)
	f.settle(code)
	other := tableroom.Spec{Creator: "x", Participants: []tableroom.Participant{{UserID: "x", Name: "X"}, {Bot: tableroom.BotEasy}, {Bot: tableroom.BotEasy}}}
	if _, err := f.tables.CreateTable(other); err != nil {
		t.Fatal(err)
	}
	if f.m.Loaded(code) {
		t.Fatal("the lobby was not evicted")
	}
	clk.Advance(time.Millisecond)
	for i := range tableroom.DefaultBudgetCacheSize / 2 {
		b := tableroom.NewBudget(tableroom.DefaultRequestBurst, tableroom.DefaultRequestInterval)
		for range tableroom.DefaultRequestBurst {
			b.Charge("u", clk.Now())
		}
		f.lobbies.budgets.Put(fmt.Sprintf("crowd-%d", i), b, 0, clk.Now())
	}
	admin = f.conn(user(0), "c0", "P0")
	f.joinEvicting(admin, code)
	admin.lobby("the lobby, loaded again", any2)
	f.send(admin, toggleBot(code, seq, seq-1))
	if got := admin.reject(seq); got != outRate {
		t.Fatalf("past the budget after the cache overflowed: %s", got)
	}
	clk.Advance(tableroom.DefaultRequestInterval)
	f.send(admin, toggleBot(code, seq, seq-1))
	admin.acked(seq)
	f.send(admin, toggleBot(code, seq+1, seq))
	if got := admin.reject(seq + 1); got != outRate {
		t.Fatalf("with that token spent: %s", got)
	}
}

// joinEvicting joins k to code, which pushes another room out of memory.
// A room just made tells the Manager it may be evicted from its own
// goroutine, so a join right after it can find no room to evict yet.
func (f *fixture) joinEvicting(k *conn, code string) {
	f.t.Helper()
	deadline := time.Now().Add(wait)
	for {
		err := f.m.Join(f.ctx(), k.c, code)
		if err == nil {
			return
		}
		if !errors.Is(err, room.ErrTooManyRooms) || time.Now().After(deadline) {
			f.t.Fatal(err)
		}
		time.Sleep(time.Millisecond)
	}
}
