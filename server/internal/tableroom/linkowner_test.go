package tableroom

import (
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/room/roomtest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// TestAdminLinksArePublic covers seated, watching and temporary admins, and
// the non-admin refusal. Each disclosure persists a log visible to everyone.
func TestAdminLinksArePublic(t *testing.T) {
	clk := roomtest.NewFakeClock(time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC))
	h := newTH(t, Config{Clock: clk, BotWork: blockBots}, humans(3, 105))
	admin, c1, spec := tabletest.SeatClient(0), tabletest.SeatClient(1), tabletest.SpectatorClient
	h.want(admin, roleOp(1, "promote", "watcher"), outAccepted, "promote the spectator")
	before := h.record()
	for _, x := range []struct {
		id    tabletest.ClientID
		seat  int
		reset bool
		who   string
	}{{admin, 1, false, "the seated creator"}, {admin, 1, true, "the seated creator"}, {spec, 2, false, "a watching admin"}, {spec, 2, true, "a watching admin"}, {c1, 2, false, "a player"}} {
		out, claim := h.linkOf(x.id, x.seat, x.reset)
		if x.id == c1 {
			if out != outNotAdmin || claim != "" {
				t.Fatalf("non-admin got link: %s", out)
			}
		} else if out != outAccepted || claim == "" {
			t.Fatalf("admin refused: %s", out)
		}

	}
	after := h.record()
	if len(after.RoomLog) != len(before.RoomLog)+4 || after.Public != before.Public+4 {
		t.Fatal("disclosure was not persisted")
	}
	log := h.sameLogEverywhere(len(after.RoomLog))
	for _, e := range log[len(before.RoomLog):] {
		if (e.Kind != "link-copy" && e.Kind != "link") || e.UserID == nil || e.Seat == nil {
			t.Fatalf("missing public identity: %+v", e)
		}
	}
	if out, _ := h.linkOf(admin, 0, false); out != outAccepted {
		t.Fatalf("the creator's link of their own seat: %s", out)
	}
	if out, _ := h.linkOf(c1, 1, false); out != outAccepted {
		t.Fatalf("a player's link of their own seat: %s", out)
	}

	// The temporary admin, a player, gets no other seat's link either.
	h.on(func(a *actor) { a.st.rec.Roles.History = []string{userOf(1)} })
	h.leave(admin, spec)
	clk.Advance(5 * time.Minute)
	h.until(c1, "user-1 as the temporary admin", func(fr wire.TableState) bool { return human(t, fr, userOf(1)).Temporary })
	for _, reset := range []bool{false, true} {
		if out, _ := h.linkOf(c1, 2, reset); out != outAccepted {
			t.Errorf("the temporary admin's link (reset %v) of seat 2: %s", reset, out)
		}
	}
}
