package wire_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// TestRecordMessages builds real server messages from the engine (not hand
// constructed JSON) and compares them, byte for byte, against
// testdata/messages/*.json. A later cmd/ajayeb-gen turns that directory into
// web/src/lib/wire/fixtures.gen.ts, which svelte-check type-checks against
// the generated TypeScript, so real server output -- not an idealized
// example -- is what the client's types are validated against. Run with
// -update after a deliberate change to the wire or view shapes:
//
//	go test ./internal/wire/ -run TestRecordMessages -update
var updateMessages = flag.Bool("update", false, "rewrite testdata/messages from the current engine and wire packages")

type recordCase struct {
	name  string
	build func(t *testing.T) wire.ServerMessage
}

var recordCases = []recordCase{
	{"welcome", func(t *testing.T) wire.ServerMessage {
		return wire.NewWelcome(wire.ProtocolVersion, wire.ServerInfo{Version: "dev", Rules: engine.RulesVersion})
	}},
	{"created", func(t *testing.T) wire.ServerMessage {
		return wire.NewCreated("K7QX2M")
	}},
	{"error-room-not-found", func(t *testing.T) wire.ServerMessage {
		return wire.NewError(wire.ErrorCodeRoomNotFound, "no room with that code")
	}},
	{"reject-stale", func(t *testing.T) wire.ServerMessage {
		return wire.NewReject(17, wire.RejectCodeStale, "base 41 is behind the current version 43")
	}},
	{"state-draft", func(t *testing.T) wire.ServerMessage {
		s := engine.NewGame(1, engine.Options{})
		return stateFrame(t, "DRAFT2", &s, nil, wire.ModeHotseat, wire.RoomStatusPlaying, nil)
	}},
	{"state-play", func(t *testing.T) wire.ServerMessage {
		s := engine.NewGame(1, engine.Options{})
		var lastEv []engine.Event
		for range 10 {
			if s.Over() {
				break
			}
			legal := engine.LegalActions(&s, nil)
			if len(legal) == 0 {
				t.Fatalf("no legal actions")
			}
			var ev []engine.Event
			if err := engine.Apply(&s, legal[0], &ev); err != nil {
				t.Fatalf("Apply: %v", err)
			}
			lastEv = ev
		}
		st := stateFrame(t, "H8TQ4R", &s, lastEv, wire.ModeHotseat, wire.RoomStatusPlaying, &wire.Ack{ClientID: "c1", Seq: 10})
		st.CanUndo = true
		return st
	}},
	{"state-ai-thinking", func(t *testing.T) wire.ServerMessage {
		s := engine.NewGame(4, engine.Options{FirstPlayer: engine.FirstPlayerSeat0})
		legal := engine.LegalActions(&s, nil)
		var ev []engine.Event
		if err := engine.Apply(&s, legal[0], &ev); err != nil {
			t.Fatalf("Apply: %v", err)
		}
		if s.ToMove != 1 {
			t.Fatalf("after seat 0's first pick seat %d is to move, want the bot's seat 1", s.ToMove)
		}
		st := stateFrame(t, "B4TQ7N", &s, ev, wire.ModeAI, wire.RoomStatusPlaying, &wire.Ack{ClientID: "c1", Seq: 1})
		bot := 1
		st.CanUndo, st.Thinking = true, &bot
		return st
	}},
	{"state-pending-token", func(t *testing.T) wire.ServerMessage {
		sc := engine.NamedScenario("science-pair")
		if sc == nil {
			t.Fatalf("named scenario %q not found", "science-pair")
		}
		s, err := engine.NewGameFromScenario(2, engine.Options{}, sc)
		if err != nil {
			t.Fatalf("NewGameFromScenario: %v", err)
		}
		var ev []engine.Event
		if err := engine.RunTrigger(&s, sc, &ev); err != nil {
			t.Fatalf("RunTrigger: %v", err)
		}
		return stateFrame(t, "W3NB9P", &s, ev, wire.ModeOnline, wire.RoomStatusPlaying, nil)
	}},
	{"state-game-over", func(t *testing.T) wire.ServerMessage {
		sc := engine.NamedScenario("military-supremacy")
		if sc == nil {
			t.Fatalf("named scenario %q not found", "military-supremacy")
		}
		s, err := engine.NewGameFromScenario(3, engine.Options{}, sc)
		if err != nil {
			t.Fatalf("NewGameFromScenario: %v", err)
		}
		var ev []engine.Event
		if err := engine.RunTrigger(&s, sc, &ev); err != nil {
			t.Fatalf("RunTrigger: %v", err)
		}
		if !s.Over() {
			t.Fatalf("military-supremacy's trigger should end the game")
		}
		return stateFrame(t, "G5VCK2", &s, ev, wire.ModeAI, wire.RoomStatusFinished, nil)
	}},
}

// stateFrame builds a wire.State the way the room package does: the seats
// of the mode (Sara on seat 0; in a hot-seat room she holds seat 1 too and
// sees the view of the seat to move, in an ai room seat 1 is a bot, online
// it is Bob), and the view and filtered events of Sara's viewer. Cases set
// CanUndo and Thinking themselves.
func stateFrame(t *testing.T, code string, s *engine.State, ev []engine.Event, mode wire.Mode, status wire.RoomStatus, ack *wire.Ack) wire.State {
	t.Helper()
	viewer := engine.Seat(0)
	seats := []wire.SeatInfo{{Seat: 0, Name: "Sara", Kind: wire.SeatKindHuman, Connected: true, You: true}}
	mine := []int{0}
	switch mode {
	case wire.ModeHotseat:
		viewer = s.ToMove
		seats = append(seats, wire.SeatInfo{Seat: 1, Name: "Omid", Kind: wire.SeatKindHuman, Connected: true, You: true})
		mine = []int{0, 1}
	case wire.ModeAI:
		level := wire.BotLevelNormal
		seats = append(seats, wire.SeatInfo{Seat: 1, Kind: wire.SeatKindBot, Bot: &level, Connected: true})
	default:
		seats = append(seats, wire.SeatInfo{Seat: 1, Name: "Bob", Kind: wire.SeatKindHuman, Connected: true})
	}
	v := engine.ViewFor(s, engine.SeatViewer(viewer))
	events := engine.FilterEvents(ev, engine.SeatViewer(viewer))
	canAct := false
	for _, seat := range mine {
		canAct = canAct || (!s.Over() && int(s.ToMove) == seat)
	}
	room := wire.NewRoomInfo(mode, status, seats, 0)
	return wire.NewState(code, int(s.Ply), room, wire.NewYouInfo(mine, canAct), ack, events, v)
}

// TestRecordMessages drives recordCases and checks each against its golden
// file under testdata/messages.
func TestRecordMessages(t *testing.T) {
	for _, c := range recordCases {
		t.Run(c.name, func(t *testing.T) {
			msg := c.build(t)
			checkMessageFile(t, filepath.Join("testdata", "messages", c.name+".json"), msg)
		})
	}
}

func checkMessageFile(t *testing.T, path string, got wire.ServerMessage) {
	t.Helper()
	data, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	data = append(data, '\n')
	if *updateMessages {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir for %s: %v", path, err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v (run with -update to create it)", path, err)
	}
	if !bytes.Equal(data, want) {
		t.Fatalf("%s is stale; run go test ./internal/wire/ -run TestRecordMessages -update to refresh it:\n--- got ---\n%s\n--- on disk ---\n%s",
			path, data, want)
	}
}

// TestRecordedMessagesHaveDiscriminator is a light sanity check that every
// recorded message at least marshals with a non-empty "t", catching a case
// whose build function returned a zero-ish value by mistake.
func TestRecordedMessagesHaveDiscriminator(t *testing.T) {
	for _, c := range recordCases {
		t.Run(c.name, func(t *testing.T) {
			data, err := json.Marshal(c.build(t))
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var head struct {
				T string `json:"t"`
			}
			if err := json.Unmarshal(data, &head); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if head.T == "" {
				t.Fatalf("recorded message has no \"t\"")
			}
		})
	}
}
