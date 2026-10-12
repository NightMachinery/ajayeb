package room_test

import (
	"errors"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// TestFinishedHotseatReopens: a finished hot-seat game reopens for the user
// who created it, from a new connection (another tab of the same browser,
// so the same user id), both while the room is in memory and after it was
// evicted and comes back from its file. Another user (another browser, so
// another token) is refused as forbidden, also from the file's header: a
// hot-seat room belongs to its creator. Its device link is the way to open
// it on another device, and it still works once the game is over.
func TestFinishedHotseatReopens(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, st := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), []string{"Ann", "Ali"}))
	for ply := 0; st.Room.Status != wire.RoomStatusFinished; ply++ {
		if ply > maxPlies {
			t.Fatalf("the game is not over after %d actions", maxPlies)
		}
		a.act(st.View.Choices[0].Key, st.Version, seatPtr(*st.View.ToMove))
		h.snapshot(code)
		st = a.current()
	}
	claim := a.link(0, false).Claim
	finished := st.Version

	reopen := func(when string, c *tclient) {
		t.Helper()
		got := c.join(code)
		if got.Room.Status != wire.RoomStatusFinished || got.View.Result == nil || got.Version != finished || len(got.You.Seats) != 2 {
			t.Fatalf("%s: status %s, result %v, version %d (want %d), seats %v",
				when, got.Room.Status, got.View.Result != nil, got.Version, finished, got.You.Seats)
		}
	}
	stranger := func(when string) {
		t.Helper()
		b := h.client("user-b", "tab-b-"+when, "Bob")
		if e, err := b.joinErr(code); e.Code != wire.ErrorCodeForbidden || !errors.Is(err, room.ErrForbidden) {
			t.Fatalf("%s: another user opening the finished hot-seat game: %v, %v", when, e, err)
		}
	}

	a2 := h.client("user-a", "tab-a2", "Ann")
	reopen("in memory", a2)
	stranger("in memory")

	h.evict(code, a, a2)
	stranger("evicted")
	if h.m.Loaded(code) {
		t.Fatalf("a refused join loaded the finished room")
	}
	reopen("evicted", h.client("user-a", "tab-a3", "Ann"))

	phone := h.client("user-c", "tab-c", "Phone")
	got := phone.joinClaim(code, claim)
	if got.Room.Status != wire.RoomStatusFinished || len(got.You.Seats) != 2 {
		t.Fatalf("the device link of the finished game: status %s, seats %v", got.Room.Status, got.You.Seats)
	}
}
