package wire_test

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/tutorial"
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
	{"reject-not-admin", func(t *testing.T) wire.ServerMessage {
		return wire.NewReject(5, wire.RejectCodeNotAdmin, "only the game's admin may change its hints")
	}},
	{"reject-undo-off", func(t *testing.T) wire.ServerMessage {
		return wire.NewReject(9, wire.RejectCodeUndoOff, "this game was created with undo off")
	}},
	{"state-hints-changed", func(t *testing.T) wire.ServerMessage {
		// An online game whose admin (Sara) turned the prices and threats
		// off and the advisor on after the fourth move: the frame the other
		// player (Bob, seat 1) receives.
		s := engine.NewGame(7, engine.Options{FirstPlayer: engine.FirstPlayerSeat0})
		for range 4 {
			engine.ApplyTrusted(&s, engine.LegalActions(&s, nil)[0])
		}
		st := stateFrame(t, "N4HT8Q", &s, nil, wire.ModeOnline, wire.RoomStatusPlaying, nil)
		v := engine.ViewFor(&s, engine.SeatViewer(1))
		st.View = v
		st.You = wire.NewYouInfo([]int{1}, s.ToMove == 1)
		for i := range st.Room.Seats {
			st.Room.Seats[i].You = i == 1
		}
		wire.SetHint(&st.Room.Hints, wire.HintFamilyPrices, false)
		wire.SetHint(&st.Room.Hints, wire.HintFamilyThreats, false)
		wire.SetHint(&st.Room.Hints, wire.HintFamilyAdvisor, true)
		seat := 0
		st.RoomLog = []wire.RoomLogEntry{{
			Kind: wire.RoomLogHints,
			At:   int(s.Ply),
			Seat: &seat,
			On:   []wire.HintFamily{wire.HintFamilyAdvisor},
			Off:  []wire.HintFamily{wire.HintFamilyPrices, wire.HintFamilyThreats},
		}}
		return st
	}},
	{"linked", func(t *testing.T) wire.ServerMessage {
		return wire.NewLinked("N4HT8Q", 1, "q3Vb8Kx0Zr7TnW2mYc5JhA")
	}},
	{"created-claim", func(t *testing.T) wire.ServerMessage {
		// The rematch room's code, as a connection that joined the old room
		// with a device link receives it: with its seat's claim there.
		f := wire.NewCreated("R7MQ3X")
		claim := "Hw4pZ9cN1sKe6uTb0LgQvY"
		f.Claim = &claim
		return f
	}},
	{"error-bad-claim", func(t *testing.T) wire.ServerMessage {
		return wire.NewError(wire.ErrorCodeBadClaim, "that device link does not work in this room (any more)")
	}},
	{"state-names-numbered", func(t *testing.T) wire.ServerMessage {
		// An online game between two players who both call themselves
		// Sara: the second to join is numbered. Sara 2 (seat 1, not the
		// admin) reset her own device link before the first move.
		s := engine.NewGame(8, engine.Options{FirstPlayer: engine.FirstPlayerSeat0})
		st := stateFrame(t, "S2NM4Q", &s, nil, wire.ModeOnline, wire.RoomStatusPlaying, nil)
		st.Room.Seats[1].Name = "Sara"
		st.Room.Seats[1].NameNo = 2
		st.View = engine.ViewFor(&s, engine.SeatViewer(1))
		st.You = wire.NewYouInfo([]int{1}, s.ToMove == 1)
		for i := range st.Room.Seats {
			st.Room.Seats[i].You = i == 1
		}
		seat, target := 1, 1
		st.RoomLog = []wire.RoomLogEntry{{
			Kind:   wire.RoomLogLink,
			At:     0,
			Seat:   &seat,
			On:     []wire.HintFamily{},
			Off:    []wire.HintFamily{},
			Target: &target,
		}}
		return st
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
		st.UndoUnavailable = nil
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
		st.UndoUnavailable = nil
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
	{"state-join-history", func(t *testing.T) wire.ServerMessage {
		// The first frame of a join: no events, and the whole history,
		// each entry filtered for the viewer (seat 0 of an online room).
		s := engine.NewGame(5, engine.Options{FirstPlayer: engine.FirstPlayerSeat0})
		history := []wire.HistoryEntry{}
		for range 8 {
			legal := engine.LegalActions(&s, nil)
			ply, seat := int(s.Ply), int(s.ToMove)
			var ev []engine.Event
			if err := engine.Apply(&s, legal[0], &ev); err != nil {
				t.Fatalf("Apply: %v", err)
			}
			history = append(history, wire.HistoryEntry{Ply: ply, Seat: seat, Events: engine.FilterEvents(ev, engine.SeatViewer(0))})
		}
		st := stateFrame(t, "J9HQ3T", &s, nil, wire.ModeOnline, wire.RoomStatusPlaying, nil)
		st.History = &history
		return st
	}},
	{"state-tutorial", func(t *testing.T) wire.ServerMessage {
		// A tutorial room, seat 0's frame after the bot's first Age I
		// reply: the reveal lesson and the buildOwn action.
		sc := &tutorial.FirstGame
		s := engine.NewGame(sc.Seed, sc.Options)
		keys := []string{}
		seats := []engine.Seat{}
		var ev []engine.Event
		for _, e := range sc.Entries[:8] {
			a, err := engine.ParseKey(e.Key)
			if err != nil {
				t.Fatalf("ParseKey: %v", err)
			}
			seats = append(seats, s.ToMove)
			ev = nil
			if err := engine.Apply(&s, a, &ev); err != nil {
				t.Fatalf("Apply %s: %v", e.Key, err)
			}
			keys = append(keys, e.Key)
		}
		st := stateFrame(t, "T7UQ2R", &s, ev, wire.ModeAI, wire.RoomStatusPlaying, nil)
		easy := wire.BotLevelEasy
		st.Room.Seats[1].Bot = &easy
		st.Room.Tutorial = true
		ts := tutorial.Derive(sc, keys, seats, &s)
		st.Tutorial = &ts
		st.CanUndo = true
		st.UndoUnavailable = nil
		return st
	}},
	{"state-forfeit", func(t *testing.T) wire.ServerMessage {
		// Seat 1 resigned while seat 0 was to move: the forfeit's frame.
		s := engine.NewGame(6, engine.Options{FirstPlayer: engine.FirstPlayerSeat0})
		for range 7 {
			engine.ApplyTrusted(&s, engine.LegalActions(&s, nil)[0])
		}
		var ev []engine.Event
		if err := engine.Apply(&s, engine.ResignAction(1), &ev); err != nil {
			t.Fatalf("resign: %v", err)
		}
		return stateFrame(t, "F2RF8K", &s, ev, wire.ModeOnline, wire.RoomStatusFinished, nil)
	}},
}

// stateFrame builds a wire.State the way the room package does: the seats
// of the mode (Sara on seat 0; in a hot-seat room she holds seat 1 too and
// sees the view of the seat to move, in an ai room seat 1 is a bot, online
// it is Bob), and the view and filtered events of Sara's viewer. Sara
// created the room, so she is its admin, its hint settings are the mode's
// defaults, and a hot-seat or ai room allows undo. Cases set CanUndo and
// Thinking themselves.
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
	info := wire.NewRoomInfo(mode, status, seats, 0)
	info.Hints = room.DefaultHints(store.Mode(mode))
	info.AllowUndo = mode != wire.ModeOnline
	you := wire.NewYouInfo(mine, canAct)
	you.Admin = true // Sara created the room
	return wire.NewState(code, int(s.Ply), info, you, ack, events, v)
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
