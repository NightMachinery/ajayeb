package tableroom

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/room/roomtest"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/faults"
)

func TestStableTableNameNumbers(t *testing.T) {
	sp := humans(3, 313)
	for i := range sp.Participants {
		sp.Participants[i].Name = "River"
	}
	h := newTH(t, Config{BotWork: blockBots}, sp)
	admin, b, c := tabletest.SeatClient(0), tabletest.SeatClient(1), tabletest.SeatClient(2)
	want := func(id tabletest.ClientID, user string, n int) {
		t.Helper()
		if got := human(t, h.latest(id), user).NameNo; got != n {
			t.Fatalf("%s: suffix %d, want %d", user, got, n)
		}
	}
	want(admin, userOf(1), 2)
	want(admin, userOf(2), 3)
	h.leave(b)
	h.want(admin, toBot(1, 1, BotEasy), outAccepted, "remove a disconnected human")
	want(admin, userOf(2), 3)
	// An observer keeps its number when seated and when moved again.
	w := tabletest.ClientID("duplicate-watcher")
	if err := h.p.join(h.ctx, w, "new-user", "river"); err != nil {
		t.Fatal(err)
	}
	want(w, "new-user", 4)
	h.want(admin, toHuman(2, 1, "new-user"), outAccepted, "seat the observer")
	want(w, "new-user", 4)
	h.want(admin, toBot(3, 1, BotEasy), outAccepted, "unseat the observer")
	h.want(admin, toBot(4, 2, BotEasy), outAccepted, "free an earlier holder's seat")
	h.want(admin, toHuman(5, 2, "new-user"), outAccepted, "move the observer")
	want(w, "new-user", 4)
	// Rename through fresh connections while the previous tab remains open.
	if err := h.p.join(h.ctx, "rename", userOf(1), "Brook"); err != nil {
		t.Fatal(err)
	}
	if err := h.p.join(h.ctx, "rename-back", userOf(1), "River"); err != nil {
		t.Fatal(err)
	}
	want(w, userOf(1), 5)
	want(w, "new-user", 4)
	h.f.restart()
	if err := h.p.join(h.ctx, w, "new-user", "river"); err != nil {
		t.Fatal(err)
	}
	want(w, "new-user", 4)
	if err := h.p.join(h.ctx, "reconnect", userOf(1), "River"); err != nil {
		t.Fatal(err)
	}
	want(w, userOf(1), 5)
	if err := h.p.join(h.ctx, c, userOf(2), "River"); err != nil {
		t.Fatal(err)
	}
	want(w, userOf(2), 3)
	if err := h.p.join(h.ctx, "new", "last-user", "RIVER"); err != nil {
		t.Fatal(err)
	}
	want(w, "last-user", 6)
}

func TestFormat3TableNameFixture(t *testing.T) {
	// Captured using master's format-3 writer at a8f1f372.
	data, err := os.ReadFile("testdata/format-3-names.json")
	if err != nil {
		t.Fatal(err)
	}
	r, err := decodeRecord(data)
	if err != nil {
		t.Fatal(err)
	}
	f := newFixture(t, Config{BotWork: blockBots})
	if _, _, err := restore(r, f.tables.bounds()); err != nil {
		t.Fatal(err)
	}
	if r.Format != recordFormat || r.Names.Get(userOf(1)).Number != 2 {
		t.Fatalf("migration: %+v", r.Names)
	}
	f.close()
	if err := os.WriteFile(filepath.Join(f.store.Dir(), r.Code+".json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	f.start()
	c := newClient(userOf(1), "old", "river")
	if err := f.m.Join(testContext(t), c, r.Code); err != nil {
		t.Fatal(err)
	}
	if got := human(t, lastFrame(t, c), userOf(1)).NameNo; got != 2 {
		t.Fatalf("suffix %d", got)
	}
	f.restart()
	c = newClient(userOf(1), "again", "river")
	if err := f.m.Join(testContext(t), c, r.Code); err != nil {
		t.Fatal(err)
	}
	if got := human(t, lastFrame(t, c), userOf(1)).NameNo; got != 2 {
		t.Fatalf("restart suffix %d", got)
	}
}

func TestTableNameWriteFailure(t *testing.T) {
	h := newTH(t, Config{BotWork: blockBots}, humans(3, 313))
	h.f.store.inject(faults.FailWrite)
	if err := h.p.join(h.ctx, "failed", userOf(1), "Brook"); !errors.Is(err, room.ErrStorage) {
		t.Fatalf("join: %v", err)
	}
	r := h.record()
	if r.Names.Get(userOf(1)).Name != "P1" {
		t.Fatal("failed assignment leaked")
	}
	if err := h.p.join(h.ctx, "retry", userOf(1), "Brook"); err != nil {
		t.Fatal(err)
	}
	if got := human(t, h.latest("retry"), userOf(1)).NameNo; got != 0 {
		t.Fatalf("retry suffix %d", got)
	}
	r = h.record()
	r.Names.Users[0].Name = strings.Repeat("x", store.MaxFileBytes)
	if _, err := r.encode(); !errors.Is(err, store.ErrTooLarge) {
		t.Fatalf("oversize: %v", err)
	}
}

func TestTableNameJoinRenameLoopsStayBounded(t *testing.T) {
	h := newTH(t, Config{BotWork: blockBots}, humans(3, 9))
	before := h.record()
	for i := range 300 {
		id := tabletest.ClientID(fmt.Sprint("watch", i))
		if err := h.p.join(h.ctx, id, fmt.Sprintf("visitor%04d", i), strings.Repeat("x", 60)); err != nil {
			t.Fatal(err)
		}
		h.leave(id)
	}
	after := h.record()
	if len(after.Names.Users) != len(before.Names.Users) || len(after.Names.Names) != len(before.Names.Names) {
		t.Fatal("spectators consumed history")
	}
	for i := range maxNameCounters + 10 {
		id := tabletest.ClientID(fmt.Sprint("rename", i))
		if err := h.p.join(h.ctx, id, userOf(1), fmt.Sprint("Name", i)); err != nil {
			t.Fatal(err)
		}
		h.leave(id)
	}
	r := h.record()
	if len(r.Names.Names) != maxNameCounters || len(r.Names.Users) > maxNameUsers {
		t.Fatal("unbounded ledger")
	}
	others := []string{userOf(0), userOf(2)}
	if err := h.p.join(h.ctx, "overflow-watcher", "overflow-watcher", "Overflow"); err != nil {
		t.Fatal(err)
	}
	if err := h.p.join(h.ctx, "overflow-rename", userOf(1), "Overflow"); err != nil {
		t.Fatal(err)
	}
	if got := h.record().Names.Get(userOf(1)); got.Name != "Overflow" || got.Number != 2 {
		t.Fatal("rename at capacity failed", got)
	}
	if human(t, h.latest("overflow-watcher"), "overflow-watcher").NameNo != 0 {
		t.Fatal("rename renumbered spectator")
	}
	for _, u := range others {
		if h.record().Names.Get(u) != r.Names.Get(u) {
			t.Fatal("rename renumbered retained holder")
		}
	}
	data, err := r.encode()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) > store.MaxFileBytes/4 {
		t.Fatalf("simple table grew to %d", len(data))
	}
	sn := h.snap(tabletest.SeatClient(0))
	h.expect(h.send(tabletest.SeatClient(0), 1, sn.Ref, sn.Legal[0], true), tabletest.Accepted, "confirmed pick with a full ledger")
	// A never-before-seated observer can still be promoted at capacity.
	if err := h.p.join(h.ctx, "new-authority", "new-authority", "Brand new name"); err != nil {
		t.Fatal(err)
	}
	h.want(tabletest.SeatClient(0), toBot(2, 2, BotEasy), outAccepted, "free a seat")
	n := human(t, h.latest("new-authority"), "new-authority").NameNo
	h.want(tabletest.SeatClient(0), toHuman(3, 2, "new-authority"), outAccepted, "seat a new name with full history")
	if human(t, h.latest("new-authority"), "new-authority").NameNo != n {
		t.Fatal("promotion renumbered visitor")
	}
}

func TestTableNameRetention(t *testing.T) {
	clk := roomtest.NewFakeClock(time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC))
	h := newTH(t, Config{Clock: clk, BotWork: blockBots}, humans(3, 9))
	old := h.record().Updated
	clk.Advance(time.Minute)
	if err := h.p.join(h.ctx, "rename", userOf(1), "Brook"); err != nil {
		t.Fatal(err)
	}
	if r := h.record(); !r.Updated.Equal(old) || r.Names.Get(userOf(1)).Name != "Brook" {
		t.Fatal("rename refreshed retention")
	}
	a, err := h.p.actor()
	if err != nil {
		t.Fatal(err)
	}
	if err := a.call(h.ctx, func() { a.st.rec.Status = store.StatusArchived }); err != nil {
		t.Fatal(err)
	}
	before, _, err := h.f.store.PeekGame(h.p.code, Game)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.p.join(h.ctx, "archived", userOf(1), "Archived rename"); err != nil {
		t.Fatal(err)
	}
	after, _, err := h.f.store.PeekGame(h.p.code, Game)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) || h.record().Names.Get(userOf(1)).Name != "Brook" {
		t.Fatal("archived record renamed")
	}
}

func TestSpectatorNamesOnlyWhilePresent(t *testing.T) {
	sp := humans(3, 9)
	for i := range sp.Participants {
		sp.Participants[i].Name = "River"
	}
	h := newTH(t, Config{BotWork: blockBots}, sp)
	join := func(id tabletest.ClientID, u, name string) {
		t.Helper()
		if err := h.p.join(h.ctx, id, u, name); err != nil {
			t.Fatal(err)
		}
	}
	join("v1", "visitor1", "River")
	join("v2", "visitor2", "River")
	if human(t, h.latest("v2"), "visitor2").NameNo != 5 {
		t.Fatal("duplicate spectator number")
	}
	h.leave("v1")
	if human(t, h.latest("v2"), "visitor2").NameNo != 5 {
		t.Fatal("departure renumbered spectator")
	}
	join("v1-back", "visitor1", "River")
	if human(t, h.latest("v1-back"), "visitor1").NameNo != 6 {
		t.Fatal("reconnect reused present number")
	}
	join("rename-visitor", "visitor1", "Brook")
	if human(t, h.latest("v2"), "visitor1").Name != "Brook" {
		t.Fatal("spectator rename not published")
	}
	// A seated rename must also avoid a spectator's live number.
	join("rename-player", userOf(1), "Brook")
	if human(t, h.latest("v2"), userOf(1)).NameNo != 2 {
		t.Fatal("rename collided with spectator")
	}
	r := h.record()
	if r.Names.Get("visitor1").UserID != "" || r.Names.Get("visitor2").UserID != "" {
		t.Fatal("spectator names persisted")
	}
}

func TestTableTrimmedNameCollision(t *testing.T) {
	h := newTH(t, Config{BotWork: blockBots}, humans(3, 313))
	admin := tabletest.SeatClient(0)
	if err := h.p.join(h.ctx, "honest", "honest", "River"); err != nil {
		t.Fatal(err)
	}
	if err := h.p.join(h.ctx, "padded", "padded", "River "); err != nil {
		t.Fatal(err)
	}
	if human(t, h.latest("padded"), "padded").NameNo != 2 {
		t.Fatal("padded name collided with spectator")
	}
	h.want(admin, toBot(1, 1, BotEasy), outAccepted, "free a seat")
	h.want(admin, toHuman(2, 1, "padded"), outAccepted, "seat a padded name")
	h.want(admin, roleOp(3, "promote", "honest"), outAccepted, "promote the honest spectator")
	h.want(admin, toBot(4, 2, BotEasy), outAccepted, "free another seat")
	h.want(admin, toHuman(5, 2, "honest"), outAccepted, "seat the honest spectator")
	st := h.latest("honest")
	if honest, padded := human(t, st, "honest"), human(t, st, "padded"); honest.NameNo != 0 || padded.NameNo != 2 || padded.Name != "River" {
		t.Fatal("seating changed assignments", honest, padded)
	}
}

func TestTableUnicodeNameLifecycle(t *testing.T) {
	for _, name := range []string{strings.Repeat("م", 40), strings.Repeat("😀", 17)} {
		t.Run(name, func(t *testing.T) {
			sp := humans(3, 313)
			sp.Participants[1].Name = name
			h := newTH(t, Config{BotWork: blockBots}, sp)
			admin := tabletest.SeatClient(0)
			if err := h.p.join(h.ctx, "watcher", "watcher", " "+name+" "); err != nil {
				t.Fatal(err)
			}
			if human(t, h.latest("watcher"), "watcher").NameNo != 2 {
				t.Fatal("Unicode spectator not numbered")
			}
			h.want(admin, roleOp(1, "promote", "watcher"), outAccepted, "promote Unicode name")
			h.want(admin, toBot(2, 2, BotEasy), outAccepted, "free a seat")
			h.want(admin, toHuman(3, 2, "watcher"), outAccepted, "seat Unicode name")
			if human(t, h.latest("watcher"), "watcher").NameNo != 2 {
				t.Fatal("promotion or seating renumbered Unicode name")
			}
			renamed := name + "م"
			if err := h.p.join(h.ctx, "rename", "watcher", renamed); err != nil {
				t.Fatal(err)
			}
			if got := human(t, h.latest("rename"), "watcher"); got.Name != renamed || got.NameNo != 0 {
				t.Fatal("Unicode rename failed", got)
			}
		})
	}
}
