package wire

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/view"
)

// TestStateExtrasJSON pins how the state frame's room-filled extras look on
// the wire: canUndo is always present, thinking only while a bot searches,
// and a seat's bot level only on a bot seat.
func TestStateExtrasJSON(t *testing.T) {
	hard := BotLevelHard
	room := NewRoomInfo(ModeAI, RoomStatusPlaying, []SeatInfo{
		{Seat: 0, Name: "Sara", Kind: SeatKindHuman, Connected: true, You: true},
		{Seat: 1, Kind: SeatKindBot, Bot: &hard, Connected: true},
	}, 0)
	st := NewState("K7QX2M", 3, room, NewYouInfo([]int{0}, false), nil, nil, view.View{})
	if st.CanUndo || st.Thinking != nil || st.UndoUnavailable == nil || *st.UndoUnavailable != UndoUnavailableDisabled {
		t.Fatalf("NewState: canUndo %v, thinking %v; want false and absent", st.CanUndo, st.Thinking)
	}
	data, err := json.Marshal(st)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	s := string(data)
	for _, want := range []string{`"canUndo":false`, `"undoUnavailable":"disabled"`, `"kind":"bot","bot":"hard"`} {
		if !strings.Contains(s, want) {
			t.Errorf("frame lacks %s: %s", want, s)
		}
	}
	if strings.Contains(s, `"thinking"`) || strings.Count(s, `"bot":`) != 1 {
		t.Errorf("frame has thinking, or a bot level on the human seat: %s", s)
	}

	seat := 1
	st.CanUndo, st.Thinking = true, &seat
	st.UndoUnavailable = nil
	data, err = json.Marshal(st)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(string(data), `"undoUnavailable"`) {
		t.Fatalf("available undo still has a reason: %s", data)
	}
	for _, want := range []string{`"canUndo":true`, `"thinking":1`} {
		if !strings.Contains(string(data), want) {
			t.Errorf("frame lacks %s: %s", want, data)
		}
	}
}

// TestErrorCodeBusy pins the code a client sees when the server is full.
func TestErrorCodeBusy(t *testing.T) {
	data, err := json.Marshal(NewError(ErrorCodeBusy, "try again later"))
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(data), `"code":"busy"`) {
		t.Fatalf("busy error frame: %s", data)
	}
}

// The constructor has no room history: it starts unavailable, names the
// first public restriction, and otherwise defaults to no human decision.
func TestUndoUnavailableConstructorReasons(t *testing.T) {
	cases := []struct {
		mode  Mode
		allow bool
		seats []int
		phase view.Phase
		want  UndoUnavailableReason
	}{
		{ModeOnline, false, nil, view.PhaseGameOver, UndoUnavailableOnline},
		{ModeHotseat, false, nil, view.PhaseGameOver, UndoUnavailableDisabled},
		{ModeAI, true, nil, view.PhaseGameOver, UndoUnavailableSpectator},
		{ModeHotseat, true, []int{0, 1}, view.PhaseGameOver, UndoUnavailableGameOver},
		{ModeAI, true, []int{0}, view.PhaseWonderDraft, UndoUnavailableNoHumanDecision},
	}
	for _, tc := range cases {
		room := NewRoomInfo(tc.mode, RoomStatusPlaying, nil, 0)
		room.AllowUndo = tc.allow
		st := NewState("K7QX2M", 0, room, NewYouInfo(tc.seats, false), nil, nil, view.View{Phase: tc.phase})
		if st.CanUndo || st.UndoUnavailable == nil || *st.UndoUnavailable != tc.want {
			t.Fatalf("state %+v, want %s", st, tc.want)
		}
		b, err := json.Marshal(st)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), `"undoUnavailable":"`+string(tc.want)+`"`) {
			t.Fatalf("missing code: %s", b)
		}
	}
}
