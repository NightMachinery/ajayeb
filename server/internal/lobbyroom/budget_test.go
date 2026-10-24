package lobbyroom

import (
	"bytes"
	"testing"
	"time"

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
	toggle := func(seq, i int) wire.LobbyEdit {
		e := edit(code, seq, "remove-bot")
		if i%2 == 0 {
			e.Op, e.Bot = "add-bot", ptr(tableroom.BotEasy)
		}
		e.Seat = ptr(3)
		return e
	}
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
