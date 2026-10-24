package tableroom

import (
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/room/roomtest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// TestOnlyTheSeatsUserLinksIt: in a table only a seat's own user may link
// or relink it. An admin, real or temporary, seated or watching, is
// refused with not-admin, receives no claim, and nothing changes: no log
// entry, no revision, no epoch. Before, an admin got any human seat's
// claim with no trace, and a device joining with it saw that seat's hand
// and pending pick and could pick or resign for it, logged as the seat's
// own user (M16.haftgane-security-review). Each user still links their
// own seat, admin or not.
func TestOnlyTheSeatsUserLinksIt(t *testing.T) {
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
		if out, claim := h.linkOf(x.id, x.seat, x.reset); out != outNotAdmin || claim != "" {
			t.Errorf("%s's link (reset %v) of seat %d: %s", x.who, x.reset, x.seat, out)
		}
	}
	after := h.record()
	if len(after.RoomLog) != len(before.RoomLog) || after.Public != before.Public || after.Seats[1].Epoch != before.Seats[1].Epoch || after.Seats[2].Epoch != before.Seats[2].Epoch {
		t.Fatalf("a refused link changed the table: log %d -> %d, public %d -> %d", len(before.RoomLog), len(after.RoomLog), before.Public, after.Public)
	}
	if out, _ := h.linkOf(admin, 0, false); out != outAccepted {
		t.Fatalf("the creator's link of their own seat: %s", out)
	}
	if out, _ := h.linkOf(c1, 1, false); out != outAccepted {
		t.Fatalf("a player's link of their own seat: %s", out)
	}

	// The temporary admin, a player, gets no other seat's link either.
	h.leave(admin, spec)
	clk.Advance(5 * time.Minute)
	h.until(c1, "user-1 as the temporary admin", func(fr wire.TableState) bool { return human(t, fr, userOf(1)).Temporary })
	for _, reset := range []bool{false, true} {
		if out, _ := h.linkOf(c1, 2, reset); out != outNotAdmin {
			t.Errorf("the temporary admin's link (reset %v) of seat 2: %s", reset, out)
		}
	}
}
