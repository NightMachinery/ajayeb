package room_test

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// TestFramesCarryHints checks the frame cache's hints (M10.HS1): a live
// game's state frame carries the hints object and the core fields, and a
// finished game's frame carries no hints at all.
func TestFramesCarryHints(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	_, st := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	if st.View.Hints == nil {
		t.Fatal("a live game's frame carries no hints")
	}
	if st.View.Draft == nil || len(st.View.Draft.Order) != 4 {
		t.Fatalf("the draft view lists no pick order: %+v", st.View.Draft)
	}
	a.act("resign:"+string(rune('0'+*st.View.ToMove)), 0, nil)
	st = a.waitState(func(s wire.State) bool { return s.Room.Status == wire.RoomStatusFinished })
	if st.View.Hints != nil || st.View.PlayingAgain {
		t.Fatalf("a finished game's frame carries hints: %+v, playing again %v", st.View.Hints, st.View.PlayingAgain)
	}
}
