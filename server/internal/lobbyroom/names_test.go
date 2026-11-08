package lobbyroom

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/namenum"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/room/roomtest"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/tableroom"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func wantNameNo(t *testing.T, humans []wire.PlatformHuman, user string, number int) {
	t.Helper()
	for _, h := range humans {
		if h.UserID == user {
			if h.NameNo != number {
				t.Fatalf("%s: suffix %d, want %d", user, h.NameNo, number)
			}
			return
		}
	}
	t.Fatalf("missing human %s", user)
}

func TestStableLobbyNameNumbers(t *testing.T) {
	f := newFixture(t, nil)
	a, b, c := f.conn(user(0), "a", "River"), f.conn(user(1), "b", "river"), f.conn(user(2), "c", "RIVER")
	code := f.create(a, 3, wire.LobbyOptions{})
	f.join(b, code)
	f.join(c, code)
	w := f.conn(user(3), "watch", "River")
	f.join(w, code)
	wantNameNo(t, w.last().Humans, user(3), 4)
	// A departure and a release do not renumber anyone. The observer takes
	// the vacated seat and retains the suffix assigned before seating.
	f.m.Leave(b.c, code)
	f.settle(code)
	rm := edit(code, 1, "remove-human")
	rm.UserID = ptr(user(1))
	f.send(a, rm)
	a.acked(1)
	// Move the third human into the vacated second slot.
	f.send(c, edit(code, 1, "release"))
	c.acked(1)
	seat := edit(code, 2, "seat")
	seat.Seat, seat.UserID = ptr(1), ptr(user(2))
	f.send(a, seat)
	st := a.acked(2)
	wantNameNo(t, st.Humans, user(2), 3)
	seat = edit(code, 3, "seat")
	seat.Seat, seat.UserID = ptr(2), ptr(user(3))
	f.send(a, seat)
	st = a.acked(3)
	wantNameNo(t, st.Humans, user(3), 4)
	// Renaming only reallocates the renamer. Returning to a historical name
	// consumes its next unused number, even while an older holder is away.
	b2 := f.conn(user(1), "b2", "Brook")
	f.join(b2, code)
	b2.last()
	b3 := f.conn(user(1), "b3", "River")
	f.join(b3, code)
	wantNameNo(t, b3.last().Humans, user(1), 5)
	f.restart()
	w2 := f.conn(user(3), "watch2", "River")
	f.join(w2, code)
	st = w2.last()
	wantNameNo(t, st.Humans, user(3), 4)
	wantNameNo(t, st.Humans, user(2), 3)
	b4 := f.conn(user(1), "b4", "River")
	f.join(b4, code)
	wantNameNo(t, b4.last().Humans, user(1), 5)
	next := f.conn(user(4), "next", "river")
	f.join(next, code)
	wantNameNo(t, next.last().Humans, user(4), 6)
}

func TestNameNumbersSurviveLobbyStart(t *testing.T) {
	for _, seats := range []int{2, 3} {
		t.Run(string(rune('0'+seats)), func(t *testing.T) {
			f := newFixture(t, nil)
			a, b := f.conn(user(0), "a", "River"), f.conn(user(1), "b", "river")
			code := f.create(a, seats, wire.LobbyOptions{Seating: ptr("random")})
			f.join(b, code)
			b.last()
			// The first slot now has a higher suffix than the second slot.
			x := f.conn(user(0), "x", "Brook")
			f.join(x, code)
			x.last()
			y := f.conn(user(0), "y", "River")
			f.join(y, code)
			wantNameNo(t, y.last().Humans, user(0), 3)
			if seats == 3 {
				third := f.conn(user(2), "third", "River")
				f.join(third, code)
				third.last()
				gone := f.conn(user(10), "gone", "River")
				f.join(gone, code)
				gone.last()
				watcher := f.conn(user(11), "stay", "River")
				f.join(watcher, code)
				wantNameNo(t, watcher.last().Humans, user(11), 6)
				f.m.Leave(gone.c, code)
				f.settle(code)
			}
			f.send(y, wire.LobbyStart{T: "lobby-start", Code: code, Seq: 1, FillBots: true})
			if seats == 2 {
				duelFrame(y)
				r, _, err := f.st.Load(code)
				if err != nil {
					t.Fatal(err)
				}
				if r.Seats[0].NameNo != 3 || r.Seats[1].NameNo != 2 {
					t.Fatalf("Duel changed suffixes: %+v", r.Seats)
				}
			} else {
				st := y.until("table", func(v any) bool { _, ok := v.(wire.TableState); return ok }).(wire.TableState)
				wantNameNo(t, st.Humans, user(0), 3)
				wantNameNo(t, st.Humans, user(1), 2)
				wantNameNo(t, st.Humans, user(11), 6)
				if f.tableRecord(code).Names.Get(user(11)).UserID != "" {
					t.Fatal("Start persisted spectator")
				}
			}
		})
	}
}

func TestFormat3LobbyNameFixture(t *testing.T) {
	// Captured using master's format-3 writer at a8f1f372, before adding
	// the ledger. Fictional names and test identities only.
	data, err := os.ReadFile("testdata/format-3-names.json")
	if err != nil {
		t.Fatal(err)
	}
	r, err := decodeRecord(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := validate(r, defaultSeqClients); err != nil {
		t.Fatal(err)
	}
	if r.Format != recordFormat || r.Names.Get(user(0)).Number != 0 || r.Names.Get(user(1)).Number != 2 {
		t.Fatalf("migration: %+v", r.Names)
	}
	f := newFixture(t, nil)
	f.close()
	if err := os.WriteFile(filepath.Join(f.st.Dir(), r.Code+".json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	f.start()
	k := f.conn(user(1), "reload", "river")
	f.join(k, r.Code)
	wantNameNo(t, k.last().Humans, user(1), 2)
	// Any subsequent write persists format 4 without re-keying format-3 seqs.
	f.send(k, edit(r.Code, 1, "release"))
	k.acked(1)
	f.restart()
	k = f.conn(user(1), "again", "river")
	f.join(k, r.Code)
	wantNameNo(t, k.last().Humans, user(1), 2)
}

func TestLobbyNameWriteFailure(t *testing.T) {
	f := newFixture(t, nil)
	a := f.conn(user(0), "a", "River")
	code := f.create(a, 2, wire.LobbyOptions{})
	f.failSaves(code, 0, 1)
	b := f.conn(user(0), "b", "river-new")
	if err := f.m.Join(f.ctx(), b.c, code); !errors.Is(err, room.ErrStorage) {
		t.Fatalf("join: %v", err)
	}
	r := f.record(code)
	if r.Names.Get(user(0)).Name != "River" {
		t.Fatal("failed assignment was published")
	}
	f.join(b, code)
	wantNameNo(t, b.last().Humans, user(0), 0)
	// The encoder is shared by actor saves, creation, and rematch creation.
	r.Names.Users[0].Name = strings.Repeat("x", store.MaxFileBytes)
	if _, err := r.encode(); !errors.Is(err, store.ErrTooLarge) {
		t.Fatalf("oversize: %v", err)
	}
}

func TestFullLobbyNameLedgerStillStarts(t *testing.T) {
	f := newFixture(t, nil)
	a := f.conn(user(0), "creator", "River")
	code := f.create(a, 3, wire.LobbyOptions{})
	b := f.conn(user(1), "second", "River")
	f.join(b, code)
	b.last()
	c := f.conn(user(2), "third", "River")
	f.join(c, code)
	c.last()
	before := f.record(code)
	for i := range 300 {
		w := f.conn(user(1000+i), fmt.Sprint("watch", i), strings.Repeat("x", 60))
		f.join(w, code)
		w.last()
		f.m.Leave(w.c, code)
		f.settle(code)
	}
	if after := f.record(code); len(after.Names.Users) != len(before.Names.Users) || len(after.Names.Names) != len(before.Names.Names) {
		t.Fatal("spectators consumed name history")
	}
	for i := range maxNameCounters + 10 {
		k := f.conn(user(1), fmt.Sprint("rename", i), fmt.Sprint("Name", i))
		f.join(k, code)
		k.last()
		f.m.Leave(k.c, code)
		f.settle(code)
	}
	r := f.record(code)
	if len(r.Names.Names) != maxNameCounters {
		t.Fatalf("history: %d", len(r.Names.Names))
	}
	name := r.Names.Get(user(1))
	k := f.conn(user(1), "frozen", "Overflow")
	f.join(k, code)
	k.last()
	if got := f.record(code).Names.Get(user(1)); got != name {
		t.Fatalf("rename at capacity: %+v", got)
	}
	f.send(a, wire.LobbyStart{T: "lobby-start", Code: code, Seq: 1, FillBots: true})
	st := a.until("table", func(v any) bool { _, ok := v.(wire.TableState); return ok }).(wire.TableState)
	wantNameNo(t, st.Humans, user(1), name.Number)
}

func TestLobbyNameRetention(t *testing.T) {
	clk := roomtest.NewFakeClock(time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC))
	f := newFixture(t, clk)
	a := f.conn(user(0), "a", "River")
	code := f.create(a, 2, wire.LobbyOptions{})
	old := f.record(code).Updated
	clk.Advance(time.Minute)
	b := f.conn(user(0), "b", "Brook")
	f.join(b, code)
	b.last()
	if r := f.record(code); !r.Updated.Equal(old) || r.Names.Get(user(0)).Name != "Brook" {
		t.Fatal("rename refreshed retention", r.Updated)
	}
	actor := f.lobbies.lookup(code)
	if err := actor.call(f.ctx(), func() { actor.rec.Status = store.StatusArchived }); err != nil {
		t.Fatal(err)
	}
	before, _, err := f.st.PeekGame(code, Game)
	if err != nil {
		t.Fatal(err)
	}
	c := f.conn(user(0), "c", "Archived rename")
	f.join(c, code)
	c.last()
	after, _, err := f.st.PeekGame(code, Game)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || f.record(code).Names.Get(user(0)).Name != "Brook" {
		t.Fatal("archived record renamed")
	}
}

func TestRematchNameAssignments(t *testing.T) {
	for _, duel := range []bool{false, true} {
		t.Run(fmt.Sprint(duel), func(t *testing.T) {
			f := newFixture(t, nil)
			code := "NEWAAA"
			assigned := []namenum.Assignment{{UserID: user(0), Name: "River", Number: 5}, {UserID: user(1), Name: "River", Number: 2}}
			if duel {
				rm := room.DuelRematch{Parent: "PRNAAA", Creator: user(0)}
				for i, x := range assigned {
					rm.Seats[i] = store.Seat{Kind: store.SeatKindHuman, UserID: x.UserID, Name: x.Name, NameNo: x.Number}
				}
				if err := f.lobbies.RematchDuel(f.m, code, rm); err != nil {
					t.Fatal(err)
				}
			} else {
				rm := tableroom.Rematch{Parent: "PRNAAA", Creator: user(0), Names: assigned, Slots: []tableroom.StartSlot{{UserID: user(1), Name: "Stale"}, {Bot: tableroom.BotEasy}, {UserID: user(0), Name: "Stale"}}}
				if err := f.lobbies.Rematch(f.m, code, rm); err != nil {
					t.Fatal(err)
				}
			}
			for _, x := range assigned {
				k := f.conn(x.UserID, "tab", x.Name)
				f.join(k, code)
				wantNameNo(t, k.last().Humans, x.UserID, x.Number)
			}
			f.restart()
			k := f.conn(user(0), "reload", "River")
			f.join(k, code)
			wantNameNo(t, k.last().Humans, user(0), 5)
		})
	}
}
