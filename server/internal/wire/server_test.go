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
	if st.CanUndo || st.Thinking != nil {
		t.Fatalf("NewState: canUndo %v, thinking %v; want false and absent", st.CanUndo, st.Thinking)
	}
	data, err := json.Marshal(st)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	s := string(data)
	for _, want := range []string{`"canUndo":false`, `"kind":"bot","bot":"hard"`} {
		if !strings.Contains(s, want) {
			t.Errorf("frame lacks %s: %s", want, s)
		}
	}
	if strings.Contains(s, `"thinking"`) || strings.Count(s, `"bot":`) != 1 {
		t.Errorf("frame has thinking, or a bot level on the human seat: %s", s)
	}

	seat := 1
	st.CanUndo, st.Thinking = true, &seat
	data, err = json.Marshal(st)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
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
