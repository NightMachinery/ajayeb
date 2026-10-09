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
		return stateFrame(t, "DRAFTXX", &s, nil, wire.ModeHotseat, wire.RoomStatusPlaying, nil)
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
		return stateFrame(t, "PLAYXXX", &s, lastEv, wire.ModeHotseat, wire.RoomStatusPlaying, &wire.Ack{ClientID: "c1", Seq: 10})
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
		return stateFrame(t, "TOKENXX", &s, ev, wire.ModeOnline, wire.RoomStatusPlaying, nil)
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
		return stateFrame(t, "OVERXXX", &s, ev, wire.ModeAI, wire.RoomStatusFinished, nil)
	}},
}

// stateFrame builds a wire.State the way the room package will: the full
// view and filtered events for seat 0, wrapped in room/you/ack envelopes
// representative of their kind.
func stateFrame(t *testing.T, code string, s *engine.State, ev []engine.Event, mode wire.Mode, status wire.RoomStatus, ack *wire.Ack) wire.State {
	t.Helper()
	v := engine.ViewFor(s, engine.SeatViewer(0))
	events := engine.FilterEvents(ev, engine.SeatViewer(0))
	room := wire.NewRoomInfo(mode, status, []wire.SeatInfo{
		{Seat: 0, Name: "Sara", Kind: wire.SeatKindHuman, Connected: true, You: true},
		{Seat: 1, Name: "Bot", Kind: wire.SeatKindBot, Connected: true, You: false},
	}, 0)
	you := wire.NewYouInfo([]int{0}, !s.Over() && s.ToMove == 0)
	return wire.NewState(code, int(s.Ply), room, you, ack, events, v)
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
