package room_test

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/view"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// The hint settings (docs/dev/decisions.md D23): part of each room, the
// same for every connection, edited by the game's admin (its creator) with
// a hints frame while the game runs, and logged for both players.

// sendHints sends a hints frame with the next seq and returns that seq.
func (c *tclient) sendHints(h view.HintSettings) int {
	c.t.Helper()
	c.seq++
	if err := c.h.m.Hints(context.Background(), c.Client, wire.NewHints(c.code, c.seq, h)); err != nil {
		c.t.Fatalf("Hints: %v", err)
	}
	return c.seq
}

// families joins hint families for a compact comparison.
func families(fs []wire.HintFamily) string {
	out := make([]string, 0, len(fs))
	for _, f := range fs {
		out = append(out, string(f))
	}
	return strings.Join(out, ",")
}

func withHint(h view.HintSettings, f wire.HintFamily, on bool) view.HintSettings {
	wire.SetHint(&h, f, on)
	return h
}

func TestHintDefaultsPerMode(t *testing.T) {
	all := room.DefaultHints(store.ModeAI)
	for _, f := range wire.HintFamilies() {
		if !wire.HintOn(all, f) {
			t.Errorf("ai default: family %s is off", f)
		}
	}
	for _, mode := range []store.Mode{store.ModeHotseat, store.ModeOnline} {
		h := room.DefaultHints(mode)
		if h != withHint(all, wire.HintFamilyAdvisor, false) {
			t.Errorf("%s default %+v: want every family but the advisor on", mode, h)
		}
	}

	h := newHarness(t)
	bot := wire.BotLevelEasy
	cases := []struct {
		req  wire.Create
		mode store.Mode
	}{
		{wire.NewCreate(wire.ModeAI, &bot, wire.SeatNumber(0), nil), store.ModeAI},
		{wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil), store.ModeHotseat},
		{wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(1), nil), store.ModeOnline},
	}
	for i, c := range cases {
		a := h.client("user-a", "tab-"+string(rune('a'+i)), "Ann")
		code, st := a.create(c.req)
		if st.Room.Hints != room.DefaultHints(c.mode) {
			t.Errorf("%s room: hints %+v, want the defaults", c.mode, st.Room.Hints)
		}
		if !st.You.Admin {
			t.Errorf("%s room: the creator is not the admin", c.mode)
		}
		if snap := h.snapshot(code); snap.Admin != "user-a" {
			t.Errorf("%s room: admin %q", c.mode, snap.Admin)
		}
		if len(st.RoomLog) != 0 || st.RoomLog == nil {
			t.Errorf("%s room: room log %#v, want empty", c.mode, st.RoomLog)
		}
	}
}

func TestCreateWithHints(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	want := withHint(room.DefaultHints(store.ModeHotseat), wire.HintFamilyPrices, false)
	want.Advisor = true
	req := wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil)
	req.Hints = &want
	code, st := a.create(req)
	if st.Room.Hints != want {
		t.Fatalf("hints %+v, want %+v", st.Room.Hints, want)
	}
	if len(st.RoomLog) != 0 {
		t.Fatalf("the settings of the create are not a change: room log %+v", st.RoomLog)
	}
	if snap := h.snapshot(code); snap.Hints != want {
		t.Fatalf("snapshot hints %+v", snap.Hints)
	}
}

// TestAdminChangesHintsOnline checks the change frame in an online room
// whose creator took seat 1: both players and a spectator receive the new
// settings and the log entry; only the admin may send it; a repeated seq
// is only acknowledged.
func TestAdminChangesHintsOnline(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, _ := a.create(wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(1), nil))

	// In the lobby already.
	lobby := withHint(room.DefaultHints(store.ModeOnline), wire.HintFamilyAdvisor, true)
	seq := a.sendHints(lobby)
	st := a.waitState(func(s wire.State) bool { return s.Ack != nil && s.Ack.Seq == seq })
	if st.Room.Status != wire.RoomStatusLobby || st.Room.Hints != lobby {
		t.Fatalf("lobby change: status %s, hints %+v", st.Room.Status, st.Room.Hints)
	}

	b := h.client("user-b", "tab-b", "Bob")
	bst := b.join(code)
	if bst.You.Admin {
		t.Fatalf("the joiner is the admin")
	}
	if bst.Room.Hints != lobby {
		t.Fatalf("the joiner sees hints %+v, want %+v", bst.Room.Hints, lobby)
	}
	if len(bst.RoomLog) != 1 || families(bst.RoomLog[0].On) != "advisor" || bst.RoomLog[0].Seat == nil || *bst.RoomLog[0].Seat != 1 || bst.RoomLog[0].At != 0 {
		t.Fatalf("the joiner's room log: %+v", bst.RoomLog)
	}
	sp := h.client("user-c", "tab-c", "Cy")
	sp.join(code)

	// Mid-game: one move, then the admin turns two families off.
	h.snapshot(code)
	mover := a
	if *b.current().View.ToMove == 0 {
		mover = b
	}
	a.current()
	mover.act(mover.last.View.Choices[0].Key, 0, nil)
	h.snapshot(code)
	a.current()
	b.current()
	sp.current()

	next := withHint(withHint(lobby, wire.HintFamilyPrices, false), wire.HintFamilyThreats, false)
	seq = a.sendHints(next)
	h.snapshot(code)
	ast := a.current()
	if ast.Ack == nil || ast.Ack.Seq != seq || ast.Version != 1 {
		t.Fatalf("the admin's frame: ack %+v, version %d (a change of hints is not a move)", ast.Ack, ast.Version)
	}
	for _, c := range []*tclient{a, b, sp} {
		st := c.current()
		if st.Room.Hints != next {
			t.Fatalf("%s sees hints %+v", c.ClientID, st.Room.Hints)
		}
		if len(st.RoomLog) != 2 {
			t.Fatalf("%s's room log: %+v", c.ClientID, st.RoomLog)
		}
		e := st.RoomLog[1]
		if e.Kind != wire.RoomLogHints || e.At != 1 || e.Seat == nil || *e.Seat != 1 || families(e.Off) != "prices,threats" || len(e.On) != 0 {
			t.Fatalf("%s's room log entry: %+v", c.ClientID, e)
		}
		if c != a && st.Ack != nil {
			t.Fatalf("%s's frame carries the admin's ack", c.ClientID)
		}
	}

	// The same seq again: only an ack, no second entry.
	a.Out.Take()
	if err := h.m.Hints(context.Background(), a.Client, wire.NewHints(code, seq, lobby)); err != nil {
		t.Fatalf("Hints: %v", err)
	}
	h.snapshot(code)
	again := a.current()
	if again.Ack == nil || again.Ack.Seq != seq || again.Room.Hints != next || len(again.RoomLog) != 2 {
		t.Fatalf("a repeated seq: ack %+v, hints %+v, log %d", again.Ack, again.Room.Hints, len(again.RoomLog))
	}

	// The other player and the spectator are refused.
	bseq := b.sendHints(lobby)
	b.expectReject(bseq, wire.RejectCodeNotAdmin)
	sseq := sp.sendHints(lobby)
	sp.expectReject(sseq, wire.RejectCodeNotAdmin)
	if snap := h.snapshot(code); snap.Hints != next {
		t.Fatalf("a refused change changed the hints: %+v", snap.Hints)
	}
	// A refused request does not use up its seq: sent again, it is judged
	// afresh and refused again.
	b.Out.Take()
	if err := h.m.Hints(context.Background(), b.Client, wire.NewHints(code, bseq, lobby)); err != nil {
		t.Fatalf("Hints: %v", err)
	}
	b.expectReject(bseq, wire.RejectCodeNotAdmin)

	// Not joined: bad-request.
	d := h.client("user-d", "tab-d", "Dee")
	d.code = code
	if err := h.m.Hints(context.Background(), d.Client, wire.NewHints(code, 1, lobby)); err == nil {
		t.Fatalf("Hints from a client that did not join: no error")
	}
	d.expectReject(1, wire.RejectCodeBadRequest)
}

// TestHintsLogFolds checks that changes at the same ply fold into one entry,
// that switching a family back cancels it, and that a move starts a new
// entry; and that an undo moves later entries back to the ply it returns
// to.
func TestHintsLogFolds(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, st := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	if st.You.Admin != true || st.Room.Mode != wire.ModeHotseat {
		t.Fatalf("hot-seat creator: admin %v", st.You.Admin)
	}
	base := st.Room.Hints

	a.sendHints(withHint(base, wire.HintFamilyPrices, false))
	a.sendHints(withHint(withHint(base, wire.HintFamilyPrices, false), wire.HintFamilyAdvisor, true))
	h.snapshot(code)
	st = a.current()
	if len(st.RoomLog) != 1 || families(st.RoomLog[0].Off) != "prices" || families(st.RoomLog[0].On) != "advisor" || st.RoomLog[0].Seat != nil {
		t.Fatalf("two changes at one ply: %+v", st.RoomLog)
	}
	// Back to the start: the entry cancels out.
	a.sendHints(base)
	h.snapshot(code)
	if st = a.current(); len(st.RoomLog) != 0 {
		t.Fatalf("a change undone at the same ply: %+v", st.RoomLog)
	}
	// The same settings again: nothing to log.
	a.sendHints(base)
	h.snapshot(code)
	if st = a.current(); len(st.RoomLog) != 0 {
		t.Fatalf("a change of nothing: %+v", st.RoomLog)
	}

	// A move, then a change at ply 1 and another after a second move.
	a.act(st.View.Choices[0].Key, 0, nil)
	st = a.waitState(func(s wire.State) bool { return s.Version == 1 })
	a.sendHints(withHint(base, wire.HintFamilyScience, false))
	a.act(st.View.Choices[0].Key, 1, nil)
	st = a.waitState(func(s wire.State) bool { return s.Version == 2 })
	a.sendHints(base)
	h.snapshot(code)
	st = a.current()
	if len(st.RoomLog) != 2 || st.RoomLog[0].At != 1 || st.RoomLog[1].At != 2 ||
		families(st.RoomLog[0].Off) != "science" || families(st.RoomLog[1].On) != "science" {
		t.Fatalf("changes at two plies: %+v", st.RoomLog)
	}

	// An undo back to ply 1 moves the second entry back to ply 1.
	a.undo(2)
	st = a.waitState(func(s wire.State) bool { return s.Version == 3 })
	if len(st.RoomLog) != 2 || st.RoomLog[0].At != 1 || st.RoomLog[1].At != 1 {
		t.Fatalf("after the undo: %+v", st.RoomLog)
	}
}

// TestHintsRefusedAfterTheEnd checks that a finished game's settings no
// longer change.
func TestHintsRefusedAfterTheEnd(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, st := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	a.act("resign:"+string(rune('0'+*st.View.ToMove)), 0, nil)
	st = a.waitState(func(s wire.State) bool { return s.Room.Status == wire.RoomStatusFinished })
	if !st.You.Admin {
		t.Fatalf("the admin is still the admin after the end")
	}
	seq := a.sendHints(withHint(st.Room.Hints, wire.HintFamilyPrices, false))
	a.expectReject(seq, wire.RejectCodeGameOver)
	if snap := h.snapshot(code); !snap.Hints.Prices || len(snap.RoomLog) != 0 {
		t.Fatalf("a refused change after the end: %+v, %+v", snap.Hints, snap.RoomLog)
	}
}

// TestHintsSurviveReload checks that the settings, the admin and the room
// log persist with the room, across an eviction and a new Manager.
func TestHintsSurviveReload(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, _ := a.create(wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(0), nil))
	b := h.client("user-b", "tab-b", "Bob")
	b.join(code)
	next := withHint(room.DefaultHints(store.ModeOnline), wire.HintFamilyCounting, false)
	a.sendHints(next)
	h.snapshot(code)
	a.current()

	h.m.Disconnect(a.Client)
	h.m.Disconnect(b.Client)
	h.snapshot(code)
	h.clock.Advance(room.DefaultIdleTimeout)
	h.waitFor("eviction", func() bool { return !h.m.Loaded(code) })

	b2 := h.client("user-b", "tab-b2", "Bob")
	st := b2.join(code)
	if st.Room.Hints != next || st.You.Admin || len(st.RoomLog) != 1 || families(st.RoomLog[0].Off) != "counting" {
		t.Fatalf("after a reload: hints %+v, admin %v, log %+v", st.Room.Hints, st.You.Admin, st.RoomLog)
	}
	a2 := h.client("user-a", "tab-a2", "Ann")
	if st := a2.join(code); !st.You.Admin {
		t.Fatalf("after a reload the creator is not the admin")
	}

	// A new Manager over the same store.
	h.m.Close()
	m2, err := room.NewManager(h.cfg)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	t.Cleanup(m2.Close)
	snap, err := m2.Snapshot(context.Background(), code)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snap.Hints != next || snap.Admin != "user-a" || len(snap.RoomLog) != 1 {
		t.Fatalf("a new Manager: hints %+v, admin %q, log %+v", snap.Hints, snap.Admin, snap.RoomLog)
	}
}

// TestOldRoomFileGetsDefaults checks a room file written before hint
// settings existed: it loads with the defaults of its mode, its creator
// (its first human seat) as the admin, and an empty log, and the admin may
// then change the settings.
func TestOldRoomFileGetsDefaults(t *testing.T) {
	cases := []struct {
		mode  store.Mode
		seats [2]store.Seat
		admin string
	}{
		{store.ModeHotseat, [2]store.Seat{{Kind: store.SeatKindHuman, UserID: "user-a", Name: "Ann"}, {Kind: store.SeatKindHuman, UserID: "user-a", Name: "Omid"}}, "user-a"},
		{store.ModeAI, [2]store.Seat{{Kind: store.SeatKindBot, Bot: store.BotEasy}, {Kind: store.SeatKindHuman, UserID: "user-a", Name: "Ann"}}, "user-a"},
		{store.ModeOnline, [2]store.Seat{{Kind: store.SeatKindOpen}, {Kind: store.SeatKindHuman, UserID: "user-a", Name: "Ann"}}, "user-a"},
		{store.ModeOnline, [2]store.Seat{{Kind: store.SeatKindHuman, UserID: "user-b", Name: "Bob"}, {Kind: store.SeatKindHuman, UserID: "user-a", Name: "Ann"}}, "user-b"},
	}
	for i, c := range cases {
		t.Run(string(c.mode), func(t *testing.T) {
			h := newHarness(t)
			code := []string{"XHDHS2", "XHDA32", "XHDN32", "XHDN42"}[i]
			f := store.New(code, c.mode, engine.Options{}, 77, roomEpoch)
			f.Seats = c.seats
			f.Status = store.StatusPlaying
			// Saved with no admin, no settings and no log: exactly the
			// shape of a file written before those fields existed (they
			// are omitted when empty).
			if err := h.st.Save(f); err != nil {
				t.Fatalf("Save: %v", err)
			}
			hdr, err := h.st.PeekHeader(code)
			if err != nil || hdr.Admin != "" || hdr.Hints != nil {
				t.Fatalf("the old-format file: %+v, %v", hdr, err)
			}

			snap := h.snapshot(code)
			if snap.Hints != room.DefaultHints(c.mode) || snap.Admin != c.admin || len(snap.RoomLog) != 0 {
				t.Fatalf("loaded: hints %+v, admin %q, log %+v", snap.Hints, snap.Admin, snap.RoomLog)
			}
			adm := h.client(c.admin, "tab-x", "X")
			st := adm.join(code)
			if !st.You.Admin || st.Room.Hints != room.DefaultHints(c.mode) {
				t.Fatalf("the admin's frame: admin %v, hints %+v", st.You.Admin, st.Room.Hints)
			}
			next := withHint(st.Room.Hints, wire.HintFamilyTempo, false)
			adm.sendHints(next)
			h.snapshot(code)
			if st := adm.current(); st.Room.Hints != next || len(st.RoomLog) != 1 {
				t.Fatalf("after a change: %+v, %+v", st.Room.Hints, st.RoomLog)
			}
			saved, err := h.st.PeekHeader(code)
			if err != nil || saved.Admin != c.admin || saved.Hints == nil || *saved.Hints != next || len(saved.RoomLog) != 1 {
				t.Fatalf("saved after the change: %+v, %v", saved, err)
			}
		})
	}
}

// TestDevRoomAdminIsFirstClaimer checks that a room made with open seats
// (the development endpoint) makes its first player the admin.
func TestDevRoomAdminIsFirstClaimer(t *testing.T) {
	h := newHarness(t)
	hints := withHint(room.DefaultHints(store.ModeOnline), wire.HintFamilyRecap, false)
	code, err := h.m.CreateFromSpec(room.Spec{
		Mode:  store.ModeOnline,
		Seats: [2]store.Seat{{Kind: store.SeatKindOpen}, {Kind: store.SeatKindOpen}},
		Seed:  5,
		Hints: &hints,
	})
	if err != nil {
		t.Fatalf("CreateFromSpec: %v", err)
	}
	if snap := h.snapshot(code); snap.Admin != "" || snap.Hints != hints {
		t.Fatalf("before any join: admin %q, hints %+v", snap.Admin, snap.Hints)
	}
	a := h.client("user-a", "tab-a", "Ann")
	if st := a.join(code); !st.You.Admin || st.Room.Hints != hints {
		t.Fatalf("the first player: admin %v, hints %+v", st.You.Admin, st.Room.Hints)
	}
	b := h.client("user-b", "tab-b", "Bob")
	if st := b.join(code); st.You.Admin {
		t.Fatalf("the second player is the admin")
	}
}

// TestRematchKeepsHints checks that a rematch keeps the admin and the
// current settings, and starts with an empty log.
func TestRematchKeepsHints(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, _ := a.create(wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(0), nil))
	b := h.client("user-b", "tab-b", "Bob")
	b.join(code)
	next := withHint(room.DefaultHints(store.ModeOnline), wire.HintFamilyWonders, false)
	a.sendHints(next)
	h.snapshot(code)
	a.current()
	b.act("resign:1", 0, nil)
	h.snapshot(code)
	newCode, err := h.m.Rematch(context.Background(), b.Client, code)
	if err != nil {
		t.Fatalf("Rematch: %v", err)
	}
	snap := h.snapshot(newCode)
	if snap.Admin != "user-a" || snap.Hints != next || len(snap.RoomLog) != 0 {
		t.Fatalf("the rematch: admin %q, hints %+v, log %+v", snap.Admin, snap.Hints, snap.RoomLog)
	}
}

// TestHintFamiliesMatchSettings checks the family list against the fields of
// view.HintSettings: every field but the last, the advisor's strength, is
// a family, in the same order, under its json name.
func TestHintFamiliesMatchSettings(t *testing.T) {
	rt := reflect.TypeFor[view.HintSettings]()
	fams := wire.HintFamilies()
	if rt.NumField() != len(fams)+1 || rt.Field(len(fams)).Name != "AdvisorStrength" {
		t.Fatalf("%d fields, %d families and the strength", rt.NumField(), len(fams))
	}
	for i, f := range fams {
		if tag := rt.Field(i).Tag.Get("json"); tag != string(f) {
			t.Errorf("field %d is %q, family %q", i, tag, f)
		}
		var h view.HintSettings
		wire.SetHint(&h, f, true)
		if reflect.ValueOf(h).Field(i).Bool() != true || !wire.HintOn(h, f) {
			t.Errorf("SetHint(%s) does not set field %d", f, i)
		}
	}
}
