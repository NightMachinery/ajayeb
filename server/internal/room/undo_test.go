package room_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func boolPtr(b bool) *bool { return &b }

// TestAllowUndoDefault checks that a hot-seat or bot room created without
// allowUndo allows undo, as does one that asks for it, and that an online
// room reports none.
func TestAllowUndoDefault(t *testing.T) {
	h := newHarness(t)
	bot := wire.BotLevelEasy
	cases := []struct {
		name string
		req  wire.Create
		want bool
	}{
		{"hotseat", wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil), true},
		{"ai", wire.NewCreate(wire.ModeAI, &bot, wire.SeatNumber(0), nil), true},
		{"online", wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(0), nil), false},
	}
	explicit := wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil)
	explicit.AllowUndo = boolPtr(true)
	cases = append(cases, struct {
		name string
		req  wire.Create
		want bool
	}{"hotseat allowUndo true", explicit, true})

	for _, c := range cases {
		a := h.client("user-"+c.name, "tab", "Ann")
		code, st := a.create(c.req)
		if st.Room.AllowUndo != c.want {
			t.Errorf("%s: room.allowUndo %v, want %v", c.name, st.Room.AllowUndo, c.want)
		}
		if snap := h.snapshot(code); snap.AllowUndo != c.want {
			t.Errorf("%s: snapshot allowUndo %v, want %v", c.name, snap.AllowUndo, c.want)
		}
		if hdr, err := h.st.PeekHeader(code); err != nil || hdr.NoUndo {
			t.Errorf("%s: stored noUndo %v, %v", c.name, hdr.NoUndo, err)
		}
	}
}

// TestUndoOffRefused checks a hot-seat room created with undo off: every
// frame says so and never offers undo, an undo is refused as undo-off and
// changes nothing, and the setting survives an eviction and a reload.
func TestUndoOffRefused(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	req := wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil)
	req.AllowUndo = boolPtr(false)
	code, st := a.create(req)
	if st.Room.AllowUndo || st.CanUndo {
		t.Fatalf("new room: allowUndo %v, canUndo %v, want both false", st.Room.AllowUndo, st.CanUndo)
	}
	for v := 0; v < 2; v++ {
		a.act(st.View.Choices[0].Key, v, nil)
		st = a.waitState(func(s wire.State) bool { return s.Version == v+1 })
		if st.Room.AllowUndo || st.CanUndo {
			t.Fatalf("after move %d: allowUndo %v, canUndo %v, want both false", v+1, st.Room.AllowUndo, st.CanUndo)
		}
	}
	seq := a.undo(2)
	a.expectReject(seq, wire.RejectCodeUndoOff)
	snap := h.snapshot(code)
	if len(snap.Actions) != 2 || snap.Version != 2 || snap.AllowUndo {
		t.Fatalf("after the refused undo: %d actions, version %d, allowUndo %v", len(snap.Actions), snap.Version, snap.AllowUndo)
	}
	// The refused undo did not use up its seq: the next act may reuse it.
	a.actSeq(seq, st.View.Choices[0].Key, 2, nil)
	st = a.waitState(func(s wire.State) bool { return s.Version == 3 })
	if st.Ack == nil || st.Ack.Seq != seq {
		t.Fatalf("an act with the refused undo's seq: ack %+v", st.Ack)
	}
	a.seq = seq

	if hdr, err := h.st.PeekHeader(code); err != nil || !hdr.NoUndo {
		t.Fatalf("stored room: noUndo %v, %v", hdr.NoUndo, err)
	}
	h.evict(code, a)
	a2 := h.client("user-a", "tab-a2", "Ann")
	st = a2.join(code)
	if st.Room.AllowUndo || st.CanUndo {
		t.Fatalf("reloaded room: allowUndo %v, canUndo %v, want both false", st.Room.AllowUndo, st.CanUndo)
	}
	seq = a2.undo(st.Version)
	a2.expectReject(seq, wire.RejectCodeUndoOff)
}

// TestUndoOffBotRoom checks that a bot room created with undo off refuses
// undo too, after the bot has answered.
func TestUndoOffBotRoom(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	bot := wire.BotLevelEasy
	req := wire.NewCreate(wire.ModeAI, &bot, wire.SeatNumber(0), nil)
	req.AllowUndo = boolPtr(false)
	code, st := a.create(req)
	if st.Room.AllowUndo {
		t.Fatalf("bot room created with undo off: allowUndo true")
	}
	for !st.You.CanAct {
		st = a.waitState(func(s wire.State) bool { return s.Version > st.Version })
	}
	a.act(st.View.Choices[0].Key, st.Version, nil)
	h.waitFor("the bot's answer", func() bool { return len(h.snapshot(code).Actions) >= 2 })
	st = a.waitState(func(s wire.State) bool { return s.Version >= 2 })
	if st.CanUndo {
		t.Fatalf("canUndo true in a bot room with undo off")
	}
	before := h.snapshot(code)
	seq := a.undo(before.Version)
	a.expectReject(seq, wire.RejectCodeUndoOff)
	if after := h.snapshot(code); len(after.Actions) != len(before.Actions) {
		t.Fatalf("the refused undo changed the log: %d actions, want %d", len(after.Actions), len(before.Actions))
	}
}

// TestCreateOnlineRefusesAllowUndo checks that an online room has no undo
// setting: a create that names one, true or false, is refused and makes no
// room.
func TestCreateOnlineRefusesAllowUndo(t *testing.T) {
	h := newHarness(t)
	for _, allow := range []bool{true, false} {
		a := h.client("user-a", "tab-a", "Ann")
		req := wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(0), nil)
		req.AllowUndo = boolPtr(allow)
		_, err := h.m.Create(context.Background(), a.Client, req)
		if !errors.Is(err, room.ErrInvalidSpec) {
			t.Fatalf("online create with allowUndo %v: err %v, want ErrInvalidSpec", allow, err)
		}
		if _, ok := a.next().(wire.Error); !ok {
			t.Fatalf("online create with allowUndo %v: no error frame", allow)
		}
	}
	_, err := h.m.CreateFromSpec(room.Spec{
		Mode:   store.ModeOnline,
		Seats:  [2]store.Seat{{Kind: store.SeatKindOpen}, {Kind: store.SeatKindOpen}},
		NoUndo: true,
	})
	if !errors.Is(err, room.ErrInvalidSpec) {
		t.Fatalf("online spec with NoUndo: err %v, want ErrInvalidSpec", err)
	}
}

// TestOldRoomAllowsUndo checks that a room file without the undo field (as
// every file written before it existed) loads with undo allowed.
func TestOldRoomAllowsUndo(t *testing.T) {
	h := newHarness(t)
	f := store.New("QXDVND", store.ModeHotseat, engine.Options{}, 5, roomEpoch)
	f.Seats = [2]store.Seat{
		{Kind: store.SeatKindHuman, UserID: "user-a", Name: "Ann"},
		{Kind: store.SeatKindHuman, UserID: "user-a", Name: "Bob"},
	}
	f.Status = store.StatusPlaying
	if err := h.st.Save(f); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(h.st.Dir(), "QXDVND.json"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(data), "noUndo") {
		t.Fatalf("the old-style file has the key noUndo:\n%s", data)
	}
	a := h.client("user-a", "tab-a", "Ann")
	st := a.join("QXDVND")
	if !st.Room.AllowUndo {
		t.Fatalf("an old room loads with allowUndo false")
	}
	a.act(st.View.Choices[0].Key, st.Version, nil)
	st = a.waitState(func(s wire.State) bool { return s.Version == 1 })
	if !st.CanUndo {
		t.Fatalf("an old room after a move: canUndo false")
	}
	seq := a.undo(1)
	st = a.waitState(func(s wire.State) bool { return s.Ack != nil && s.Ack.Seq == seq })
	if st.Version != 2 {
		t.Fatalf("undo in an old room: version %d, want 2", st.Version)
	}
}

// TestRematchKeepsUndoOff checks that the rematch of a game created with
// undo off has undo off too.
func TestRematchKeepsUndoOff(t *testing.T) {
	h := newHarness(t)
	code, err := h.m.CreateFromSpec(room.Spec{
		Mode: store.ModeHotseat,
		Seats: [2]store.Seat{
			{Kind: store.SeatKindHuman, UserID: "user-a", Name: "Ann"},
			{Kind: store.SeatKindHuman, UserID: "user-a", Name: "Bob"},
		},
		Seed:     3,
		Scenario: "military-supremacy",
		Actions:  mustScenarioTrigger(t, "military-supremacy"),
		NoUndo:   true,
	})
	if err != nil {
		t.Fatalf("CreateFromSpec: %v", err)
	}
	a := h.client("user-a", "tab-a", "Ann")
	if st := a.join(code); st.Room.AllowUndo {
		t.Fatalf("the finished room allows undo")
	}
	code2, err := h.m.Rematch(context.Background(), a.Client, code)
	if err != nil {
		t.Fatalf("Rematch: %v", err)
	}
	if snap := h.snapshot(code2); snap.AllowUndo {
		t.Fatalf("the rematch allows undo")
	}
}
