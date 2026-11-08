package tableroom

import (
	"os"
	"reflect"
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/roles"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/faults"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func TestSelfRejoin(t *testing.T) {
	h := newTH(t, Config{BotWork: blockBots}, humans(3, 479))
	admin, p, w := tabletest.SeatClient(0), tabletest.SeatClient(1), tabletest.SpectatorClient
	h.want(admin, toBot(1, 1, BotEasy), outAccepted, "move human to observers")
	st := h.latest(p)
	if st.You.Seat != nil || st.You.RejoinSeat == nil || *st.You.RejoinSeat != 1 || st.View.You != nil || st.Pending != nil {
		t.Fatalf("observer offer leaks seat: %+v", st.You)
	}
	if h.latest(w).You.RejoinSeat != nil {
		t.Fatal("offer leaked to another viewer")
	}
	h.want(w, toHuman(1, 1, userOf(1)), outNotAdmin, "cannot rejoin for another user")
	h.want(p, toHuman(1, 2, userOf(1)), outNotAdmin, "cannot take another seat")
	h.want(admin, toHuman(2, 1, "watcher"), outAccepted, "other human takes seat")
	h.want(p, toHuman(1, 1, userOf(1)), outNotAdmin, "cannot displace human")
	h.want(admin, toBot(3, 1, BotEasy), outAccepted, "free former seat")
	h.f.restart()
	h.rejoin()
	if h.latest(p).You.RejoinSeat == nil {
		t.Fatal("restart lost offer")
	}
	h.f.store.inject(faults.FailWrite)
	h.want(p, toHuman(1, 1, userOf(1)), outStorage, "failed write does not take seat")
	if h.latest(p).View.You != nil {
		t.Fatal("failed rejoin exposed hand")
	}
	h.want(p, toHuman(1, 1, userOf(1)), outAccepted, "take own former seat")
	st = h.latest(p)
	if st.You.Seat == nil || *st.You.Seat != 1 || st.View.You == nil || st.History == nil || st.You.RejoinSeat != nil {
		t.Fatal("rejoin did not deliver seat projection and history")
	}
	if h.latest(w).View.You != nil {
		t.Fatal("rejoin exposed hand to unrelated spectator")
	}
	last := st.RoomLog[len(st.RoomLog)-1]
	if last.Kind != "seat" || last.By != userOf(1) || last.Seat == nil || *last.Seat != 1 || last.UserID == nil || *last.UserID != userOf(1) {
		t.Fatalf("log: %+v", last)
	}
	rev := st.Revision
	h.want(p, toHuman(1, 1, userOf(1)), outDuplicate, "rejoin retry")
	if h.latest(p).Revision != rev {
		t.Fatal("retry changed revision")
	}
}

func TestFormat4TablePreservesMetadata(t *testing.T) {
	sp := humans(3, 479)
	for i := range sp.Participants {
		sp.Participants[i].Name = "River"
	}
	h := newTH(t, Config{BotWork: blockBots}, sp)
	if err := h.p.join(h.ctx, "rename", userOf(0), "Brook"); err != nil {
		t.Fatal(err)
	}
	if err := h.p.join(h.ctx, "rename-back", userOf(0), "River"); err != nil {
		t.Fatal(err)
	}
	before := h.record()
	h.rewrite(func(r *Record) {
		r.Format = 4
		r.Roles.History = nil
		r.Roles.Deputies = nil
		r.Roles.Declined = nil
		r.Roles.Elections = 0
	})
	for i := range 3 {
		if err := h.p.join(h.ctx, tabletest.SeatClient(i), userOf(i), "River"); err != nil {
			t.Fatal(err)
		}
	}
	after := h.record()
	if after.Format != recordFormat || !reflect.DeepEqual(before.Names, after.Names) || !reflect.DeepEqual(before.Seqs, after.Seqs) {
		t.Fatal("format-4 migration reset name numbers or client keys")
	}
}

// Captured by the unchanged format-4 writer at 0b948662, using fictional
// names and test identities. The creator reclaimed a duplicate name.
func TestFormat4TableFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/format-4-roles.json")
	if err != nil {
		t.Fatal(err)
	}
	r, err := decodeRecord(data)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = restore(r, defaultBounds); err != nil {
		t.Fatal(err)
	}
	if r.Format != recordFormat || r.Names.Get("user-0").Number != 3 || r.Names.Get("user-1").Number != 2 {
		t.Fatalf("name ledger reset: %+v", r.Names)
	}
}

func TestFormat4RetiredCreator(t *testing.T) {
	h := newTH(t, Config{BotWork: blockBots}, humans(3, 479))
	admin := tabletest.SeatClient(0)
	h.want(admin, wire.TableResign{T: "table-resign", Seq: 1, Confirm: true}, outAccepted, "creator resigns")
	h.rewrite(func(r *Record) {
		r.Format = 4
		r.Roles.Admins = slices.DeleteFunc(r.Roles.Admins, func(a roles.Admin) bool { return a.UserID == userOf(0) })
		r.Retired = append(r.Retired, userOf(0))
	})
	h.joinAs(admin, userOf(0))
	r := h.record()
	if !r.Roles.Real(userOf(0)) || slices.Contains(r.Retired, userOf(0)) {
		t.Fatal("creator was not restored out of retirement")
	}
}
