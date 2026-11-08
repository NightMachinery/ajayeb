package tableroom

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/room"
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
	if err := h.p.join(h.ctx, "failed", "new-user", "P1"); !errors.Is(err, room.ErrStorage) {
		t.Fatalf("join: %v", err)
	}
	r := h.record()
	if r.Names.Get("new-user").UserID != "" {
		t.Fatal("failed assignment leaked")
	}
	if err := h.p.join(h.ctx, "retry", "new-user", "P1"); err != nil {
		t.Fatal(err)
	}
	if got := human(t, h.latest("retry"), "new-user").NameNo; got != 2 {
		t.Fatalf("retry suffix %d", got)
	}
	r = h.record()
	r.Names.Users[0].Name = strings.Repeat("x", store.MaxFileBytes)
	if _, err := r.encode(); !errors.Is(err, store.ErrTooLarge) {
		t.Fatalf("oversize: %v", err)
	}
}
