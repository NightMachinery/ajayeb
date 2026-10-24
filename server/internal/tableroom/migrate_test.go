package tableroom

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
)

// rewrite replaces the table's stored record with what edit makes of it
// while the server is down, as an older build would have left it, and
// starts the server again. Every connection must join again.
func (h *th) rewrite(edit func(r *Record)) {
	h.t.Helper()
	r := h.record()
	edit(&r)
	data, err := r.encode()
	if err != nil {
		h.t.Fatal(err)
	}
	h.f.close()
	if err := os.WriteFile(filepath.Join(h.f.dir, "rooms", h.p.code+".json"), data, 0o600); err != nil {
		h.t.Fatal(err)
	}
	h.f.start()
}

// joinAs joins id as user, named after the user.
func (h *th) joinAs(id tabletest.ClientID, user string) {
	h.t.Helper()
	if err := h.p.join(h.ctx, id, user, user); err != nil {
		h.t.Fatalf("%s's join: %v", id, err)
	}
}

// floorsOf lists the clients with a floor in seat's pending pick, as the
// running table holds them.
func (h *th) floorsOf(seat int) (out []SeqRecord) {
	h.on(func(a *actor) { out = slices.Clone(pendingOf(&a.st.rec, seat).Seqs) })
	return out
}

// TestOldFloorsAreTrimmedOnLoad: a pending pick written under the bounds
// before M16.haftgane-security-review, which kept 32 clients of each user,
// loads under today's 8. Restore keeps each user's 8 most recently added
// floors, and a tab whose floor it dropped is refused with rate, as a
// ninth tab is today, so none of its seqs can replace the pick. Before, the
// table archived on load (found by the slice's re-check).
func TestOldFloorsAreTrimmedOnLoad(t *testing.T) {
	// The old build's bounds, and room for this test's ten connections of
	// one user (M16.table-request-budget caps them at 8).
	h := newTH(t, Config{SeqClients: 32, PickClients: 32, MaxUserClients: 32, BotWork: blockBots}, humans(3, 309))
	s := h.snap(tabletest.SeatClient(1))
	tab := func(i int) tabletest.ClientID { return tabletest.ClientID(fmt.Sprintf("old-tab-%d", i)) }
	for i := range 9 {
		h.joinAs(tab(i), userOf(1))
		h.expect(h.send(tab(i), 1, s.Ref, s.Legal[i%len(s.Legal)], false), tabletest.Accepted, "a tab's pick under the old bounds")
	}
	h.f.cfg.SeqClients, h.f.cfg.PickClients = 0, 0
	h.f.restart()
	for i := range 9 {
		h.joinAs(tab(i), userOf(1))
	}
	floors := h.floorsOf(1)
	if len(floors) != DefaultPickClients || slices.ContainsFunc(floors, func(x SeqRecord) bool { return x.Client == string(tab(0)) }) {
		t.Fatalf("the floors kept: %+v", floors)
	}
	pick := h.snap(tab(8)).Pick
	h.expect(h.send(tab(0), 2, s.Ref, s.Legal[1], false), tabletest.Rate, "the tab whose floor was dropped")
	if got := h.snap(tab(8)).Pick; got != pick {
		t.Fatalf("the dropped tab changed the pick from %q to %q", pick, got)
	}
	h.expect(h.send(tab(8), 2, s.Ref, s.Legal[0], false), tabletest.Accepted, "a tab whose floor was kept")
}

// TestOldPickUsersAreTrimmedOnLoad: a pending pick that kept the floors of
// more users than today's 8 (the bound was 16) loads with the floors of
// the seat's current user and of the 7 others who added one most
// recently. A user whose floors were dropped is refused with rate when
// seated there, the seq and tab of their dropped floor included, so their
// old seqs cannot replace the pick either.
func TestOldPickUsersAreTrimmedOnLoad(t *testing.T) {
	h := newTH(t, Config{BotWork: blockBots}, humans(3, 313))
	admin, c1 := tabletest.SeatClient(0), tabletest.SeatClient(1)
	s := h.snap(c1)
	h.expect(h.send(c1, 1, s.Ref, s.Legal[0], false), tabletest.Accepted, "seat 1's pick")
	old := func(i int) string { return fmt.Sprintf("old-user-%d", i) }
	h.rewrite(func(r *Record) {
		r.Format = 2
		slot := pendingOf(r, 1)
		for i := range 9 {
			slot.Seqs = append(slot.Seqs, SeqRecord{User: old(i), Client: "tab", Seq: 5})
		}
	})
	h.joinAs(admin, userOf(0))
	h.joinAs(c1, userOf(1))
	var users []string
	for _, x := range h.floorsOf(1) {
		users = append(users, x.User)
	}
	want := []string{userOf(1), old(2), old(3), old(4), old(5), old(6), old(7), old(8)}
	if !slices.Equal(users, want) {
		t.Fatalf("the users whose floors were kept: %v, want %v", users, want)
	}
	h.joinAs("tab", old(0))
	h.want(admin, toHuman(1, 1, old(0)), outAccepted, "seat a user whose floor was dropped")
	h.expect(h.send("tab", 5, s.Ref, s.Legal[1], false), tabletest.Rate, "their dropped floor's seq")
	h.expect(h.send("tab", 6, s.Ref, s.Legal[1], false), tabletest.Rate, "a newer seq of theirs")
	if got := h.snap("tab").Pick; got != s.Legal[0] {
		t.Fatalf("the pick changed to %q", got)
	}
}

// TestOldClientIdsAreRekeyedOnLoad: a format-2 record kept a long client
// id as sent; on load it is re-keyed by ClientKey, so the client's
// accepted seq is still remembered and an older one is a duplicate. A
// format-3 record is not re-keyed again: after another restart the
// client's retry is still a duplicate. Before, the format-2 record's raw
// id was never found again, and seq 7 replaced the pick accepted at seq 8
// (found by the slice's re-check).
func TestOldClientIdsAreRekeyedOnLoad(t *testing.T) {
	h := newTH(t, Config{BotWork: blockBots}, humans(3, 310))
	id := tabletest.ClientID(strings.Repeat("a", 80))
	h.joinAs(id, userOf(1))
	s := h.snap(id)
	h.expect(h.send(id, 8, s.Ref, s.Legal[0], false), tabletest.Accepted, "the long-id tab's pick")
	key, n := ClientKey(string(id)), 0
	h.rewrite(func(r *Record) {
		r.Format = 2
		raw := func(xs []SeqRecord) {
			for i := range xs {
				if xs[i].Client == key {
					xs[i].Client = string(id)
					n++
				}
			}
		}
		raw(r.Seqs)
		for i := range r.Pending {
			raw(r.Pending[i].Seqs)
		}
	})
	if n != 2 {
		t.Fatalf("the record held the client %d times, want an ack and a floor", n)
	}
	h.joinAs(id, userOf(1))
	h.expect(h.send(id, 7, s.Ref, s.Legal[1], false), tabletest.Duplicate, "an older seq after the format-2 load")
	if got := h.snap(id).Pick; got != s.Legal[0] {
		t.Fatalf("the pick changed to %q", got)
	}
	h.expect(h.send(id, 9, s.Ref, s.Legal[1], false), tabletest.Accepted, "a newer seq")
	h.f.restart()
	h.joinAs(id, userOf(1))
	h.expect(h.send(id, 9, s.Ref, s.Legal[0], false), tabletest.Duplicate, "its retry after a format-3 load")
}
