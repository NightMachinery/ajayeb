package tableroom

import (
	"slices"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/roles"
	"github.com/NightMachinery/ajayeb/server/internal/room/roomtest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/faults"
)

// TestRolesRefusedWrites: the store refuses each of the temporary-admin
// rule's changes in turn (the grant, a real admin's return, the passing
// on). Until a retry saves the change, the temporary role fails closed:
// the user it would demote or replace has no powers (no seat switch, no
// device link of the creator's seat), the frames show no temporary admin
// the actor would not obey, and the user it would make one has none
// either. Each retry makes the change once.
func TestRolesRefusedWrites(t *testing.T) {
	clk := roomtest.NewFakeClock(time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC))
	h := newTH(t, Config{Clock: clk, BotWork: blockBots}, humans(3, 97))
	admin, c1, c2 := tabletest.SeatClient(0), tabletest.SeatClient(1), tabletest.SeatClient(2)
	seq := map[tabletest.ClientID]int{}
	// switchSeat asks for seat 2 at the bot level it does not have.
	switchSeat := func(id tabletest.ClientID) string {
		seq[id]++
		level := BotEasy
		if sr := h.record().Seats[2]; sr.Kind == SeatBot && sr.Bot == BotEasy {
			level = BotNormal
		}
		return h.do(id, toBot(seq[id], 2, level))
	}
	temporary := func() string {
		h.leave()
		return h.record().Roles.Temporary
	}
	h.leave(admin, tabletest.SpectatorClient)

	// The grant is refused: user-1 has no powers until the retry.
	h.f.store.inject(faults.FailWrite)
	clk.Advance(5 * time.Minute)
	if out := switchSeat(c1); out != outNotAdmin || temporary() != "" {
		t.Fatalf("after a refused grant: %s, temporary %q", out, temporary())
	}
	clk.Advance(roles.Retry)
	if temporary() != userOf(1) || !human(t, h.latest(c2), userOf(1)).Temporary {
		t.Fatalf("the retried grant: %q", temporary())
	}
	if out := switchSeat(c1); out != outAccepted {
		t.Fatalf("the temporary admin's switch: %s", out)
	}

	// The creator's return is refused: user-1 loses the powers at once,
	// cannot mint the creator's claim, and no frame calls them temporary.
	h.f.store.inject(faults.FailWrite)
	if err := h.p.join(h.ctx, admin, userOf(0), "P0"); err != nil {
		t.Fatal(err)
	}
	if out := switchSeat(c1); out != outNotAdmin {
		t.Fatalf("a demoted temporary admin's switch: %s", out)
	}
	if out, _ := h.linkOf(c1, 0, false); out != outNotAdmin {
		t.Fatalf("a demoted temporary admin's link of the creator's seat: %s", out)
	}
	if out, _ := h.linkOf(c1, 0, true); out != outNotAdmin {
		t.Fatalf("a demoted temporary admin's relink of the creator's seat: %s", out)
	}
	if temporary() != userOf(1) || human(t, h.latest(admin), userOf(1)).Temporary || human(t, h.latest(c2), userOf(1)).Temporary {
		t.Fatal("the frames show a temporary admin the table no longer obeys")
	}
	clk.Advance(roles.Retry)
	if temporary() != "" {
		t.Fatalf("the retried demotion: %q", temporary())
	}

	// The passing on is refused, and user-1 comes back before the retry,
	// whose write is refused too: neither user-1 nor user-2 has the
	// powers until a write saves user-2's.
	h.leave(admin)
	clk.Advance(5 * time.Minute)
	if temporary() != userOf(1) {
		t.Fatalf("the second grant: %q", temporary())
	}
	h.leave(c1)
	h.f.store.inject(faults.FailWrite, faults.FailWrite)
	clk.Advance(5 * time.Minute)
	if err := h.p.join(h.ctx, c1, userOf(1), "P1"); err != nil {
		t.Fatal(err)
	}
	if out1, out2 := switchSeat(c1), switchSeat(c2); out1 != outNotAdmin || out2 != outNotAdmin || temporary() != userOf(1) {
		t.Fatalf("after a refused passing on: user-1 %s, user-2 %s, temporary %q", out1, out2, temporary())
	}
	clk.Advance(roles.Retry)
	if temporary() != userOf(2) {
		t.Fatalf("the retried passing on: %q", temporary())
	}
	if out1, out2 := switchSeat(c1), switchSeat(c2); out1 != outNotAdmin || out2 != outAccepted {
		t.Fatalf("after the passing on: user-1 %s, user-2 %s", out1, out2)
	}
	var log []string
	for _, e := range h.sameLogEverywhere(len(h.latest(c2).RoomLog)) {
		if e.Kind == "role" {
			log = append(log, *e.UserID+"="+*e.Role)
		}
	}
	want := []string{"user-1=temporary", "user-1=human", "user-1=temporary", "user-1=human", "user-2=temporary"}
	if !slices.Equal(log, want) {
		t.Fatalf("role log %v, want %v", log, want)
	}
}
