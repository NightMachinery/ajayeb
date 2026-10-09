package room_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// maxPlies bounds the game loops below; a real game takes about a hundred
// actions.
const maxPlies = 400

// checkSeat compares one seat of a frame with want, the bot level by value.
func checkSeat(t *testing.T, st wire.State, want wire.SeatInfo) {
	t.Helper()
	if len(st.Room.Seats) != 2 {
		t.Fatalf("room has %d seats, want 2", len(st.Room.Seats))
	}
	got := st.Room.Seats[want.Seat]
	level := func(b *wire.BotLevel) string {
		if b == nil {
			return "<none>"
		}
		return string(*b)
	}
	gotLevel, wantLevel := level(got.Bot), level(want.Bot)
	got.Bot, want.Bot = nil, nil
	if got != want || gotLevel != wantLevel {
		t.Fatalf("seat %d: got %+v with bot %s, want %+v with bot %s", want.Seat, got, gotLevel, want, wantLevel)
	}
}

// checkPlayerFrame checks the parts of a seated player's frame that hold
// whatever the position.
func checkPlayerFrame(t *testing.T, st wire.State, seat int) {
	t.Helper()
	if st.View.Viewer == nil || *st.View.Viewer != seat {
		t.Fatalf("version %d: view is for %v, want seat %d", st.Version, st.View.Viewer, seat)
	}
	toMove := st.View.ToMove != nil && *st.View.ToMove == seat
	if st.You.CanAct != (toMove && st.Room.Status == wire.RoomStatusPlaying) {
		t.Fatalf("version %d: canAct %v for seat %d, to move %v", st.Version, st.You.CanAct, seat, st.View.ToMove)
	}
	if !toMove && len(st.View.Choices) != 0 {
		t.Fatalf("version %d: seat %d is not to move but has %d choices", st.Version, seat, len(st.View.Choices))
	}
	if toMove && len(st.View.Choices) == 0 {
		t.Fatalf("version %d: seat %d is to move but has no choices", st.Version, seat)
	}
}

func checkSpectatorFrame(t *testing.T, st wire.State) {
	t.Helper()
	if st.View.Viewer != nil || len(st.View.Choices) != 0 || len(st.You.Seats) != 0 || st.You.CanAct {
		t.Fatalf("version %d: spectator frame has viewer %v, %d choices, seats %v, canAct %v",
			st.Version, st.View.Viewer, len(st.View.Choices), st.You.Seats, st.You.CanAct)
	}
}

// TestOnlineGameWithSpectator plays a whole online game between two users
// through the actor, with a spectator watching, and checks every frame each
// connection receives and the room file after every action.
func TestOnlineGameWithSpectator(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, st := a.create(wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(0), nil))
	if st.Room.Status != wire.RoomStatusLobby || st.Room.Mode != wire.ModeOnline || st.Version != 0 {
		t.Fatalf("new online room: status %s, mode %s, version %d", st.Room.Status, st.Room.Mode, st.Version)
	}
	checkSeat(t, st, wire.SeatInfo{Seat: 0, Name: "Ann", Kind: wire.SeatKindHuman, Connected: true, You: true})
	checkSeat(t, st, wire.SeatInfo{Seat: 1, Kind: wire.SeatKindOpen})
	if st.You.CanAct || len(st.You.Seats) != 1 || st.You.Seats[0] != 0 {
		t.Fatalf("creator in the lobby: you %+v", st.You)
	}

	// Nobody plays before the second seat is taken.
	seq := a.act("draft:pyramids", 0, nil)
	a.expectReject(seq, wire.RejectCodeNotYourTurn)

	b := h.client("user-b", "tab-b", "Bob")
	stB := b.join(code)
	if stB.Room.Status != wire.RoomStatusPlaying {
		t.Fatalf("after the second join: status %s", stB.Room.Status)
	}
	checkSeat(t, stB, wire.SeatInfo{Seat: 1, Name: "Bob", Kind: wire.SeatKindHuman, Connected: true, You: true})
	checkSeat(t, stB, wire.SeatInfo{Seat: 0, Name: "Ann", Kind: wire.SeatKindHuman, Connected: true})
	stA := a.waitState(func(s wire.State) bool { return s.Room.Status == wire.RoomStatusPlaying })
	checkSeat(t, stA, wire.SeatInfo{Seat: 1, Name: "Bob", Kind: wire.SeatKindHuman, Connected: true})

	c := h.client("user-c", "tab-c", "Cy")
	stC := c.join(code)
	checkSpectatorFrame(t, stC)
	if stC.Room.Spectators != 1 {
		t.Fatalf("spectators: %d, want 1", stC.Room.Spectators)
	}
	seq = c.act("draft:pyramids", stC.Version, nil)
	c.expectReject(seq, wire.RejectCodeNotYourTurn)

	players := [2]*tclient{a, b}
	version := 0
	for ply := 0; ; ply++ {
		if ply > maxPlies {
			t.Fatalf("the game is not over after %d actions", maxPlies)
		}
		h.snapshot(code)
		var frames [2]wire.State
		for seat, p := range players {
			frames[seat] = p.current()
			if frames[seat].Version != version {
				t.Fatalf("seat %d sees version %d, want %d", seat, frames[seat].Version, version)
			}
			checkPlayerFrame(t, frames[seat], seat)
		}
		spec := c.current()
		checkSpectatorFrame(t, spec)
		if spec.Version != version || spec.Room.Spectators != 1 {
			t.Fatalf("spectator: version %d, spectators %d", spec.Version, spec.Room.Spectators)
		}
		hdr, err := h.st.PeekHeader(code)
		if err != nil {
			t.Fatalf("PeekHeader: %v", err)
		}
		if len(hdr.Actions) != version {
			t.Fatalf("room file has %d actions at version %d", len(hdr.Actions), version)
		}
		if frames[0].Room.Status == wire.RoomStatusFinished {
			break
		}
		toMove := *frames[0].View.ToMove
		players[toMove].act(frames[toMove].View.Choices[0].Key, version, nil)
		version++
	}

	snap := h.snapshot(code)
	if snap.Status != store.StatusFinished || !snap.State.Over() || snap.Version != version {
		t.Fatalf("after the game: status %s, over %v, version %d (want %d)", snap.Status, snap.State.Over(), snap.Version, version)
	}
	for seat, p := range players {
		if p.current().View.Result == nil {
			t.Fatalf("seat %d's last frame has no result", seat)
		}
	}
	f, s, err := h.st.Load(code)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if f.Status != store.StatusFinished || f.Result == nil || engine.HashString(&s) != engine.HashString(&snap.State) {
		t.Fatalf("stored room: status %s, result %v, hash %s vs %s", f.Status, f.Result, engine.HashString(&s), engine.HashString(&snap.State))
	}

	seq = a.act("draft:pyramids", version, nil)
	a.expectReject(seq, wire.RejectCodeIllegal)
}

// TestHotseatGame plays a whole hot-seat game: one connection holds both
// seats and always receives the deciding seat's view.
func TestHotseatGame(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, st := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), []string{"Ann", "Ali"}))
	if st.Room.Status != wire.RoomStatusPlaying {
		t.Fatalf("hot-seat room starts as %s", st.Room.Status)
	}
	checkSeat(t, st, wire.SeatInfo{Seat: 0, Name: "Ann", Kind: wire.SeatKindHuman, Connected: true, You: true})
	checkSeat(t, st, wire.SeatInfo{Seat: 1, Name: "Ali", Kind: wire.SeatKindHuman, Connected: true, You: true})
	if len(st.You.Seats) != 2 {
		t.Fatalf("hot-seat you.seats = %v", st.You.Seats)
	}

	// Only the owner may enter.
	b := h.client("user-b", "tab-b", "Bob")
	if e, err := b.joinErr(code); e.Code != wire.ErrorCodeForbidden || !errors.Is(err, room.ErrForbidden) {
		t.Fatalf("stranger joining a hot-seat room: %v, %v", e, err)
	}

	// An act naming the waiting seat is refused.
	toMove := *st.View.ToMove
	seq := a.act(st.View.Choices[0].Key, 0, seatPtr(1-toMove))
	a.expectReject(seq, wire.RejectCodeNotYourTurn)

	for ply := 0; ; ply++ {
		if ply > maxPlies {
			t.Fatalf("the game is not over after %d actions", maxPlies)
		}
		h.snapshot(code)
		st := a.current()
		if st.Version != ply {
			t.Fatalf("version %d after %d actions", st.Version, ply)
		}
		if st.Room.Status == wire.RoomStatusFinished {
			break
		}
		toMove := *st.View.ToMove
		if *st.View.Viewer != toMove || !st.You.CanAct || len(st.View.Choices) == 0 {
			t.Fatalf("version %d: view for %d, to move %d, canAct %v, %d choices",
				st.Version, *st.View.Viewer, toMove, st.You.CanAct, len(st.View.Choices))
		}
		a.act(st.View.Choices[0].Key, st.Version, seatPtr(toMove))
	}
	if snap := h.snapshot(code); snap.Status != store.StatusFinished {
		t.Fatalf("status %s after the game", snap.Status)
	}
}

// TestAIGameWithRealBots plays a whole game against package ai's bot,
// through the default bot factory and a search pool.
func TestAIGameWithRealBots(t *testing.T) {
	h := newHarness(t, func(c *room.Config) {
		c.NewBot = nil
		c.BotPool = ai.NewPool(1)
	})
	a := h.client("user-a", "tab-a", "Ann")
	level := wire.BotLevelEasy
	code, st := a.create(wire.NewCreate(wire.ModeAI, &level, wire.SeatNumber(0), []string{"Ann"}))
	checkSeat(t, st, wire.SeatInfo{Seat: 0, Name: "Ann", Kind: wire.SeatKindHuman, Connected: true, You: true})
	checkSeat(t, st, wire.SeatInfo{Seat: 1, Kind: wire.SeatKindBot, Bot: &level, Connected: true})

	// A stranger may not enter a bot room.
	b := h.client("user-b", "tab-b", "Bob")
	if e, _ := b.joinErr(code); e.Code != wire.ErrorCodeForbidden {
		t.Fatalf("stranger joining a bot room: %v", e)
	}

	next := 0
	for ply := 0; ; ply++ {
		if ply > maxPlies {
			t.Fatalf("the game is not over after %d human actions", maxPlies)
		}
		st := a.waitState(func(s wire.State) bool {
			return s.Room.Status == wire.RoomStatusFinished || (s.You.CanAct && s.Version >= next)
		})
		if st.Room.Status == wire.RoomStatusFinished {
			break
		}
		checkPlayerFrame(t, st, 0)
		a.act(st.View.Choices[0].Key, st.Version, nil)
		next = st.Version + 1
	}

	snap := h.snapshot(code)
	if snap.Status != store.StatusFinished || snap.DroppedBotResults != 0 || snap.Version != len(snap.Actions) {
		t.Fatalf("after the game: status %s, dropped %d, version %d, %d actions",
			snap.Status, snap.DroppedBotResults, snap.Version, len(snap.Actions))
	}
	// The bot really played: replay the log and count its decisions.
	s := engine.NewGame(snap.State.Seed, snap.State.Opts)
	botMoves := 0
	for _, key := range snap.Actions {
		if s.ToMove == 1 {
			botMoves++
		}
		a, err := engine.ParseKey(key)
		if err != nil {
			t.Fatalf("ParseKey(%q): %v", key, err)
		}
		if err := engine.Apply(&s, a, nil); err != nil {
			t.Fatalf("replaying %s: %v", key, err)
		}
	}
	if botMoves == 0 || engine.HashString(&s) != engine.HashString(&snap.State) {
		t.Fatalf("replay: %d bot moves, hash %s vs %s", botMoves, engine.HashString(&s), engine.HashString(&snap.State))
	}
}

// TestDuplicateAndStaleActs checks idempotency: a seq already used only
// re-sends the state (acknowledged), and a base that is not the current
// version is rejected as stale and followed by a fresh state.
func TestDuplicateAndStaleActs(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, st := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	key := st.View.Choices[0].Key

	seq := a.act(key, 0, nil)
	st1 := a.waitState(func(s wire.State) bool { return s.Version == 1 })
	if st1.Ack == nil || st1.Ack.Seq != seq || st1.Ack.ClientID != "tab-a" {
		t.Fatalf("the reply to an act has ack %+v", st1.Ack)
	}

	// The same seq again, as after a double click or a reconnect.
	for _, dup := range []int{seq, seq - 1} {
		a.actSeq(dup, key, 0, nil)
		h.snapshot(code)
		frames := a.drain()
		if len(frames) != 1 {
			t.Fatalf("duplicate seq %d: %d frames, want one state: %#v", dup, len(frames), frames)
		}
		st, ok := frames[0].(wire.State)
		if !ok || st.Version != 1 || st.Ack == nil || st.Ack.Seq != dup {
			t.Fatalf("duplicate seq %d: got %#v", dup, frames[0])
		}
	}
	if snap := h.snapshot(code); snap.Version != 1 || len(snap.Actions) != 1 {
		t.Fatalf("after duplicates: version %d, %d actions", snap.Version, len(snap.Actions))
	}

	// A stale base, with a new seq.
	seq = a.act(st1.View.Choices[0].Key, 0, nil)
	a.expectReject(seq, wire.RejectCodeStale)
	fresh := a.waitState(anyState)
	if fresh.Version != 1 || fresh.Ack != nil {
		t.Fatalf("state after a stale reject: version %d, ack %+v", fresh.Version, fresh.Ack)
	}

	// Keys that are not in the legal set, well formed or not.
	for _, bad := range []string{"construct:no-such-card", "starter:0", "nonsense"} {
		seq = a.act(bad, 1, nil)
		a.expectReject(seq, wire.RejectCodeIllegal)
	}
	if snap := h.snapshot(code); snap.Version != 1 || len(snap.Actions) != 1 {
		t.Fatalf("after rejects: version %d, %d actions", snap.Version, len(snap.Actions))
	}

	// A connection that never joined is refused.
	stranger := h.client("user-a", "tab-z", "Ann")
	stranger.code = code
	err := h.m.Act(context.Background(), stranger.Client, wire.NewAct(code, 1, 1, key, nil))
	if !errors.Is(err, room.ErrNotJoined) {
		t.Fatalf("act without joining: %v", err)
	}
	stranger.expectReject(1, wire.RejectCodeBadRequest)
}

// TestDuplicateSeqSurvivesReload checks that the last seq of each client is
// stored with the room, so a re-sent act is still recognized after the room
// was evicted and reloaded.
func TestDuplicateSeqSurvivesReload(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, st := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	seq := a.act(st.View.Choices[0].Key, 0, nil)
	a.waitState(func(s wire.State) bool { return s.Version == 1 })
	h.m.Disconnect(a.Client)
	h.snapshot(code)
	h.clock.Advance(room.DefaultIdleTimeout)
	h.waitFor("eviction", func() bool { return !h.m.Loaded(code) })

	a2 := h.client("user-a", "tab-a", "Ann") // the same tab, reconnected
	a2.join(code)
	a2.actSeq(seq, st.View.Choices[0].Key, 0, nil)
	h.snapshot(code)
	frames := a2.drain()
	if len(frames) != 1 {
		t.Fatalf("re-sent act after reload: %#v", frames)
	}
	if st, ok := frames[0].(wire.State); !ok || st.Ack == nil || st.Ack.Seq != seq || st.Version != 1 {
		t.Fatalf("re-sent act after reload: %#v", frames[0])
	}
}

// TestUndoHotseat checks that undo cuts the log back one decision at a time
// and replays from the seed, and refuses what it cannot do.
func TestUndoHotseat(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, st := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))

	var keys []string
	for v := 0; v < 3; v++ {
		key := st.View.Choices[0].Key
		keys = append(keys, key)
		a.act(key, v, nil)
		st = a.waitState(func(s wire.State) bool { return s.Version == v+1 })
	}

	seq := a.undo(3)
	st = a.waitState(func(s wire.State) bool { return s.Version == 4 })
	if st.Ack == nil || st.Ack.Seq != seq || len(st.Events) != 0 {
		t.Fatalf("undo reply: ack %+v, %d events", st.Ack, len(st.Events))
	}
	snap := h.snapshot(code)
	if len(snap.Actions) != 2 || snap.Actions[0] != keys[0] || snap.Actions[1] != keys[1] {
		t.Fatalf("after undo the log is %v, want %v", snap.Actions, keys[:2])
	}
	want := engine.NewGame(snap.State.Seed, snap.State.Opts)
	for _, key := range keys[:2] {
		act, _ := engine.ParseKey(key)
		if err := engine.Apply(&want, act, nil); err != nil {
			t.Fatalf("Apply(%s): %v", key, err)
		}
	}
	if engine.HashString(&want) != engine.HashString(&snap.State) {
		t.Fatalf("after undo the state is not the replay of the first two actions")
	}
	f, _, err := h.st.Load(code)
	if err != nil || len(f.Actions) != 2 {
		t.Fatalf("stored room after undo: %v, %v", f, err)
	}

	// A stale undo.
	seq = a.undo(3)
	a.expectReject(seq, wire.RejectCodeStale)
	a.waitState(func(s wire.State) bool { return s.Version == 4 })

	// Undo down to the start, then once more.
	for v := 4; v < 6; v++ {
		a.undo(v)
		a.waitState(func(s wire.State) bool { return s.Version == v+1 })
	}
	seq = a.undo(6)
	a.expectReject(seq, wire.RejectCodeIllegal)
	if snap := h.snapshot(code); len(snap.Actions) != 0 || snap.Version != 6 {
		t.Fatalf("after undoing everything: %d actions, version %d", len(snap.Actions), snap.Version)
	}

	// Play goes on from the replayed start.
	st = a.current()
	a.act(st.View.Choices[0].Key, 6, nil)
	a.waitState(func(s wire.State) bool { return s.Version == 7 })
}

// TestUndoRefusedOnline checks that online rooms have no undo.
func TestUndoRefusedOnline(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, _ := a.create(wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(0), nil))
	b := h.client("user-b", "tab-b", "Bob")
	b.join(code)
	h.snapshot(code)
	st := a.current()
	toMove := *st.View.ToMove
	mover := [2]*tclient{a, b}[toMove]
	mover.act(mover.current().View.Choices[0].Key, 0, nil)
	mover.waitState(func(s wire.State) bool { return s.Version == 1 })
	seq := mover.undo(1)
	mover.expectReject(seq, wire.RejectCodeBadRequest)
	if snap := h.snapshot(code); len(snap.Actions) != 1 {
		t.Fatalf("online undo changed the log: %v", snap.Actions)
	}
}

// aiSpec is a bot room whose human seat is open (the first joiner takes it)
// and whose seat 0 moves first.
func aiSpec(first engine.FirstPlayerRule, level store.BotLevel) room.Spec {
	return room.Spec{
		Mode:    store.ModeAI,
		Seats:   [2]store.Seat{{Kind: store.SeatKindOpen}, {Kind: store.SeatKindBot, Bot: level}},
		Seed:    7,
		Options: engine.Options{FirstPlayer: first},
	}
}

// TestUndoDropsStaleBotResult checks the bot's staleness rule: an undo
// while the bot thinks cancels the search, its late answer is dropped, and
// the game goes on from the undone position.
func TestUndoDropsStaleBotResult(t *testing.T) {
	gate := newGateBot()
	h := newHarness(t, func(c *room.Config) {
		c.NewBot = func(store.BotLevel, uint64) room.Bot { return gate }
	})
	code, err := h.m.CreateFromSpec(aiSpec(engine.FirstPlayerSeat0, store.BotNormal))
	if err != nil {
		t.Fatalf("CreateFromSpec: %v", err)
	}
	a := h.client("user-a", "tab-a", "Ann")
	st := a.join(code)
	if st.Room.Status != wire.RoomStatusPlaying || !st.You.CanAct {
		t.Fatalf("after claiming the human seat: status %s, canAct %v", st.Room.Status, st.You.CanAct)
	}
	checkSeat(t, st, wire.SeatInfo{Seat: 0, Name: "Ann", Kind: wire.SeatKindHuman, Connected: true, You: true})
	key := st.View.Choices[0].Key

	a.act(key, 0, nil)
	if obs := gate.waitStarted(t); obs.Seat() != 1 || obs.Public().Seed != 0 {
		t.Fatalf("the bot observes seat %d with seed %d", obs.Seat(), obs.Public().Seed)
	}
	st = a.waitState(func(s wire.State) bool { return s.Version == 1 })
	if st.You.CanAct {
		t.Fatalf("the human can act while the bot is to move")
	}
	if snap := h.snapshot(code); !snap.BotThinking {
		t.Fatalf("no bot search is running")
	}

	seq := a.undo(1)
	st = a.waitState(func(s wire.State) bool { return s.Version == 2 })
	if st.Ack == nil || st.Ack.Seq != seq || !st.You.CanAct {
		t.Fatalf("after undo: ack %+v, canAct %v", st.Ack, st.You.CanAct)
	}
	h.waitFor("the stale bot result to be dropped", func() bool { return h.snapshot(code).DroppedBotResults == 1 })
	if snap := h.snapshot(code); snap.Version != 2 || len(snap.Actions) != 0 || snap.BotThinking {
		t.Fatalf("after the stale result: version %d, actions %v, thinking %v", snap.Version, snap.Actions, snap.BotThinking)
	}

	// The game goes on: the human plays again and the bot answers.
	a.act(key, 2, nil)
	gate.waitStarted(t)
	gate.release <- struct{}{}
	a.waitState(func(s wire.State) bool { return s.Version == 4 })
	if snap := h.snapshot(code); len(snap.Actions) != 2 || snap.Actions[0] != key {
		t.Fatalf("after the bot's answer: %v", snap.Actions)
	}
	// The bot is now searching its second pick; Close must cancel it.
}

// TestBotThinkDelay checks that the bot never answers before the think
// delay has passed on the room's clock.
func TestBotThinkDelay(t *testing.T) {
	h := newHarness(t, func(c *room.Config) { c.BotDelay = room.DefaultBotDelay })
	code, err := h.m.CreateFromSpec(aiSpec(engine.FirstPlayerSeat0, store.BotEasy))
	if err != nil {
		t.Fatalf("CreateFromSpec: %v", err)
	}
	a := h.client("user-a", "tab-a", "Ann")
	st := a.join(code)
	a.act(st.View.Choices[0].Key, 0, nil)
	a.waitState(func(s wire.State) bool { return s.Version == 1 })

	if !h.clock.WaitForTimers(1, waitLimit) {
		t.Fatalf("the bot never armed its think delay")
	}
	h.clock.Advance(room.DefaultBotDelay - time.Millisecond)
	if snap := h.snapshot(code); snap.Version != 1 {
		t.Fatalf("the bot moved before its think delay: version %d", snap.Version)
	}
	h.clock.Advance(time.Millisecond)
	a.waitState(func(s wire.State) bool { return s.Version == 2 })
}

// TestBotSeedAndObservation checks what a bot is given: its own seat's
// observation (seed hidden) and a seed derived from the room seed and the
// ply. It also checks that a bot does not start while its room waits for
// the human.
func TestBotSeedAndObservation(t *testing.T) {
	type made struct {
		level store.BotLevel
		seed  uint64
	}
	madeCh := make(chan made, 8)
	gate := newGateBot()
	h := newHarness(t, func(c *room.Config) {
		c.NewBot = func(level store.BotLevel, seed uint64) room.Bot {
			madeCh <- made{level, seed}
			return gate
		}
	})
	spec := aiSpec(engine.FirstPlayerSeat1, store.BotHard)
	code, err := h.m.CreateFromSpec(spec)
	if err != nil {
		t.Fatalf("CreateFromSpec: %v", err)
	}
	if snap := h.snapshot(code); snap.Status != store.StatusLobby || snap.BotThinking {
		t.Fatalf("before the human joins: status %s, thinking %v", snap.Status, snap.BotThinking)
	}
	a := h.client("user-a", "tab-a", "Ann")
	st := a.join(code)
	if st.You.CanAct {
		t.Fatalf("the human may act on the bot's turn")
	}
	obs := gate.waitStarted(t)
	got := <-madeCh
	if got.level != store.BotHard || got.seed != ai.MixSeed(spec.Seed, 0) {
		t.Fatalf("bot made with %+v, want level hard and seed %d", got, ai.MixSeed(spec.Seed, 0))
	}
	if obs.Seat() != 1 || obs.Public().Seed != 0 {
		t.Fatalf("the bot observes seat %d, seed %d", obs.Seat(), obs.Public().Seed)
	}
	// The human cannot act for the bot.
	seq := a.act("draft:pyramids", 0, nil)
	a.expectReject(seq, wire.RejectCodeNotYourTurn)
	gate.release <- struct{}{}
	a.waitState(func(s wire.State) bool { return s.Version == 1 })
}

// TestIdleEvictionAndReload checks that a room with no connection leaves
// memory after the idle timeout, never before, and comes back from disk
// unchanged on the next join.
func TestIdleEvictionAndReload(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, st := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	for v := 0; v < 2; v++ {
		a.act(st.View.Choices[0].Key, v, nil)
		st = a.waitState(func(s wire.State) bool { return s.Version == v+1 })
	}
	before := h.snapshot(code)

	// A connected room is never evicted.
	h.clock.Advance(2 * room.DefaultIdleTimeout)
	h.snapshot(code)
	if !h.m.Loaded(code) {
		t.Fatalf("a room with a connection was evicted")
	}

	h.m.Leave(a.Client, code)
	h.snapshot(code) // the detach is handled and the countdown armed
	h.clock.Advance(room.DefaultIdleTimeout - time.Second)
	h.snapshot(code)
	if !h.m.Loaded(code) {
		t.Fatalf("evicted before the idle timeout")
	}
	h.clock.Advance(time.Second)
	h.waitFor("eviction", func() bool { return !h.m.Loaded(code) })
	if h.m.Len() != 0 {
		t.Fatalf("%d rooms in memory after eviction", h.m.Len())
	}

	b := h.client("user-a", "tab-b", "Ann")
	st = b.join(code)
	if st.Version != 2 || st.Room.Status != wire.RoomStatusPlaying {
		t.Fatalf("reloaded room: version %d, status %s", st.Version, st.Room.Status)
	}
	after := h.snapshot(code)
	if engine.HashString(&after.State) != engine.HashString(&before.State) || len(after.Actions) != 2 {
		t.Fatalf("the reloaded room differs from the evicted one")
	}
	b.act(st.View.Choices[0].Key, 2, nil)
	b.waitState(func(s wire.State) bool { return s.Version == 3 })
}

// TestSaveFailureKeepsRoom checks that a room whose write failed stays in
// memory (and playable) until a write succeeds, so nothing is lost.
func TestSaveFailureKeepsRoom(t *testing.T) {
	var flaky *flakyStore
	h := newHarness(t, func(c *room.Config) {
		flaky = &flakyStore{Store: c.Store.(*store.Store)}
		c.Store = flaky
	})
	a := h.client("user-a", "tab-a", "Ann")
	code, st := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))

	flaky.fail.Store(true)
	a.act(st.View.Choices[0].Key, 0, nil)
	a.waitState(func(s wire.State) bool { return s.Version == 1 })
	if snap := h.snapshot(code); !snap.Dirty {
		t.Fatalf("a failed write did not mark the room dirty")
	}
	h.m.Leave(a.Client, code)
	h.snapshot(code)
	h.clock.Advance(room.DefaultIdleTimeout)
	if snap := h.snapshot(code); !snap.Dirty || !h.m.Loaded(code) {
		t.Fatalf("a dirty room was evicted (dirty %v, loaded %v)", snap.Dirty, h.m.Loaded(code))
	}

	flaky.fail.Store(false)
	h.clock.Advance(room.DefaultIdleTimeout)
	h.waitFor("eviction", func() bool { return !h.m.Loaded(code) })
	if snap := h.snapshot(code); len(snap.Actions) != 1 || snap.Version != 1 {
		t.Fatalf("reloaded after a failed write: %d actions, version %d", len(snap.Actions), snap.Version)
	}
}

// TestSpectatorsAndPresence checks seats, spectators and the connected
// flags as connections come and go.
func TestSpectatorsAndPresence(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, _ := a.create(wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(1), nil))
	b := h.client("user-b", "tab-b", "Bob")
	if st := b.join(code); !st.Room.Seats[0].You || st.You.Seats[0] != 0 {
		t.Fatalf("the second user did not get the open seat 0: %+v", st.Room.Seats)
	}
	c := h.client("user-c", "tab-c", "Cy")
	d := h.client("user-d", "tab-d", "Di")
	c.join(code)
	d.join(code)
	h.snapshot(code)
	if st := b.current(); st.Room.Spectators != 2 {
		t.Fatalf("spectators %d, want 2", st.Room.Spectators)
	}

	h.m.Disconnect(a.Client)
	h.snapshot(code)
	st := b.current()
	checkSeat(t, st, wire.SeatInfo{Seat: 1, Name: "Ann", Kind: wire.SeatKindHuman, Connected: false})

	// Ann comes back from another tab and gets her seat, not a spectator's.
	a2 := h.client("user-a", "tab-a2", "Ann")
	st = a2.join(code)
	checkSeat(t, st, wire.SeatInfo{Seat: 1, Name: "Ann", Kind: wire.SeatKindHuman, Connected: true, You: true})
	if st.Room.Spectators != 2 {
		t.Fatalf("spectators %d after Ann's return, want 2", st.Room.Spectators)
	}

	h.m.Leave(c.Client, code)
	h.snapshot(code)
	if st := b.current(); st.Room.Spectators != 1 {
		t.Fatalf("spectators %d after one left, want 1", st.Room.Spectators)
	}
	if snap := h.snapshot(code); snap.Clients != 3 || snap.Spectators != 1 {
		t.Fatalf("snapshot: %d clients, %d spectators", snap.Clients, snap.Spectators)
	}
}

// TestJoinErrors checks the refusals of join and create.
func TestJoinErrors(t *testing.T) {
	h := newHarness(t, func(c *room.Config) { c.MaxClients = 2; c.MaxRooms = 2 })
	a := h.client("user-a", "tab-a", "Ann")
	for _, code := range []string{"ABCDEF", "abc", "../../x", ""} {
		if e, err := a.joinErr(code); e.Code != wire.ErrorCodeRoomNotFound || !errors.Is(err, room.ErrRoomNotFound) {
			t.Fatalf("join %q: %v, %v", code, e, err)
		}
	}

	code, _ := a.create(wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(0), nil))
	b := h.client("user-b", "tab-b", "Bob")
	b.join(code)
	c := h.client("user-c", "tab-c", "Cy")
	if e, err := c.joinErr(code); e.Code != wire.ErrorCodeRoomFull || !errors.Is(err, room.ErrRoomFull) {
		t.Fatalf("join of a full room: %v, %v", e, err)
	}

	// The second room fits; the third does not.
	d := h.client("user-d", "tab-d", "Di")
	d.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	e := h.client("user-e", "tab-e", "Ed")
	_, err := h.m.Create(context.Background(), e.Client, wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	if !errors.Is(err, room.ErrTooManyRooms) {
		t.Fatalf("create beyond MaxRooms: %v", err)
	}
	if f, ok := e.next().(wire.Error); !ok || f.Code != wire.ErrorCodeBusy {
		t.Fatalf("create beyond MaxRooms sent %#v", f)
	}
}

// TestRematch checks that a finished room's rematch creates one new room
// with the same seats and tells every connection about it.
func TestRematch(t *testing.T) {
	h := newHarness(t)
	code, err := h.m.CreateFromSpec(room.Spec{
		Mode: store.ModeOnline,
		Seats: [2]store.Seat{
			{Kind: store.SeatKindHuman, UserID: "user-a", Name: "Ann"},
			{Kind: store.SeatKindHuman, UserID: "user-b", Name: "Bob"},
		},
		Seed:     3,
		Scenario: "military-supremacy",
		Actions:  mustScenarioTrigger(t, "military-supremacy"),
	})
	if err != nil {
		t.Fatalf("CreateFromSpec: %v", err)
	}
	a := h.client("user-a", "tab-a", "Ann")
	b := h.client("user-b", "tab-b", "Bob")
	c := h.client("user-c", "tab-c", "Cy")
	if st := a.join(code); st.Room.Status != wire.RoomStatusFinished || st.View.Result == nil {
		t.Fatalf("the scenario room is not finished: %s", st.Room.Status)
	}
	b.join(code)
	c.join(code)

	// A spectator may not ask for a rematch.
	if _, err := h.m.Rematch(context.Background(), c.Client, code); !errors.Is(err, room.ErrForbidden) {
		t.Fatalf("spectator rematch: %v", err)
	}
	h.snapshot(code)
	c.drain()

	code2, err := h.m.Rematch(context.Background(), a.Client, code)
	if err != nil || code2 == code {
		t.Fatalf("Rematch: %q, %v", code2, err)
	}
	again, err := h.m.Rematch(context.Background(), b.Client, code)
	if err != nil || again != code2 {
		t.Fatalf("second Rematch: %q, %v (want %q)", again, err, code2)
	}
	h.snapshot(code)
	for _, cl := range []*tclient{a, b, c} {
		found := false
		for _, m := range cl.drain() {
			if cr, ok := m.(wire.Created); ok && cr.Code == code2 {
				found = true
			}
		}
		if !found {
			t.Fatalf("client %s was not told the rematch code", cl.ClientID)
		}
	}

	st := b.join(code2)
	checkSeat(t, st, wire.SeatInfo{Seat: 1, Name: "Bob", Kind: wire.SeatKindHuman, Connected: true, You: true})
	if st.Version != 0 || st.Room.Status != wire.RoomStatusPlaying {
		t.Fatalf("rematch room: version %d, status %s", st.Version, st.Room.Status)
	}
	st = a.join(code2)
	checkSeat(t, st, wire.SeatInfo{Seat: 0, Name: "Ann", Kind: wire.SeatKindHuman, Connected: true, You: true})

	// A game that is not over has no rematch.
	if _, err := h.m.Rematch(context.Background(), a.Client, code2); !errors.Is(err, room.ErrNotOver) {
		t.Fatalf("rematch of a running game: %v", err)
	}
	a.expectReject(0, wire.RejectCodeBadRequest)
}

// TestCloseIsFinal checks that a closed Manager refuses everything.
func TestCloseIsFinal(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, st := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	h.m.Close()
	if err := h.m.Act(context.Background(), a.Client, wire.NewAct(code, 1, 0, st.View.Choices[0].Key, nil)); !errors.Is(err, room.ErrClosed) {
		t.Fatalf("Act after Close: %v", err)
	}
	if err := h.m.Join(context.Background(), a.Client, code); !errors.Is(err, room.ErrClosed) {
		t.Fatalf("Join after Close: %v", err)
	}
	if h.m.Len() != 0 {
		t.Fatalf("%d rooms after Close", h.m.Len())
	}
	// The game was saved before Close: a new Manager on the same store
	// continues it.
	m2, err := room.NewManager(room.Config{Store: h.st, Clock: h.clock})
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	defer m2.Close()
	snap, err := m2.Snapshot(context.Background(), code)
	if err != nil || snap.Version != 0 || snap.Status != store.StatusPlaying {
		t.Fatalf("reopened room: %+v, %v", snap.Status, err)
	}
}

// TestCreateFromSpecRejectsBadSpecs checks that a spec whose seats do not
// fit its mode, or whose game does not replay, makes no room.
func TestCreateFromSpecRejectsBadSpecs(t *testing.T) {
	h := newHarness(t)
	human := func(id string) store.Seat { return store.Seat{Kind: store.SeatKindHuman, UserID: id} }
	open := store.Seat{Kind: store.SeatKindOpen}
	bot := func(l store.BotLevel) store.Seat { return store.Seat{Kind: store.SeatKindBot, Bot: l} }
	cases := []struct {
		name string
		spec room.Spec
	}{
		{"online with a bot", room.Spec{Mode: store.ModeOnline, Seats: [2]store.Seat{human("a"), bot(store.BotEasy)}}},
		{"online with one user twice", room.Spec{Mode: store.ModeOnline, Seats: [2]store.Seat{human("a"), human("a")}}},
		{"hot-seat with two users", room.Spec{Mode: store.ModeHotseat, Seats: [2]store.Seat{human("a"), human("b")}}},
		{"hot-seat half open", room.Spec{Mode: store.ModeHotseat, Seats: [2]store.Seat{human("a"), open}}},
		{"bot room without a bot", room.Spec{Mode: store.ModeAI, Seats: [2]store.Seat{human("a"), open}}},
		{"bot of an unknown level", room.Spec{Mode: store.ModeAI, Seats: [2]store.Seat{open, bot("wizard")}}},
		{"human seat without a user", room.Spec{Mode: store.ModeOnline, Seats: [2]store.Seat{{Kind: store.SeatKindHuman}, open}}},
		{"unknown mode", room.Spec{Mode: "solo", Seats: [2]store.Seat{open, open}}},
		{"unknown scenario", room.Spec{Mode: store.ModeHotseat, Seats: [2]store.Seat{open, open}, Scenario: "no-such-scenario"}},
		{"illegal action", room.Spec{Mode: store.ModeHotseat, Seats: [2]store.Seat{open, open}, Actions: []string{"construct:baths"}}},
	}
	for _, tc := range cases {
		if code, err := h.m.CreateFromSpec(tc.spec); !errors.Is(err, room.ErrInvalidSpec) {
			t.Errorf("%s: code %q, err %v; want ErrInvalidSpec", tc.name, code, err)
		}
	}
	if h.m.Len() != 0 {
		t.Fatalf("%d rooms made from bad specs", h.m.Len())
	}

	// A hot-seat spec with open seats goes to its first joiner.
	code, err := h.m.CreateFromSpec(room.Spec{Mode: store.ModeHotseat, Seats: [2]store.Seat{open, open}, Seed: 5})
	if err != nil {
		t.Fatalf("CreateFromSpec: %v", err)
	}
	a := h.client("user-a", "tab-a", "Ann")
	st := a.join(code)
	if len(st.You.Seats) != 2 || st.Room.Status != wire.RoomStatusPlaying {
		t.Fatalf("first joiner of an open hot-seat room: you %+v, status %s", st.You, st.Room.Status)
	}
	b := h.client("user-b", "tab-b", "Bob")
	if e, _ := b.joinErr(code); e.Code != wire.ErrorCodeForbidden {
		t.Fatalf("second user in a claimed hot-seat room: %v", e)
	}
}

// TestBotRestartsAfterReload checks that a bot room evicted while its bot
// was to move starts the bot again when it is reloaded.
func TestBotRestartsAfterReload(t *testing.T) {
	gate := newGateBot()
	h := newHarness(t, func(c *room.Config) {
		c.NewBot = func(store.BotLevel, uint64) room.Bot { return gate }
	})
	code, err := h.m.CreateFromSpec(aiSpec(engine.FirstPlayerSeat0, store.BotNormal))
	if err != nil {
		t.Fatalf("CreateFromSpec: %v", err)
	}
	a := h.client("user-a", "tab-a", "Ann")
	st := a.join(code)
	a.act(st.View.Choices[0].Key, 0, nil)
	gate.waitStarted(t)
	h.m.Disconnect(a.Client)
	h.snapshot(code)
	h.clock.Advance(room.DefaultIdleTimeout)
	h.waitFor("eviction", func() bool { return !h.m.Loaded(code) })

	a2 := h.client("user-a", "tab-a2", "Ann")
	st = a2.join(code)
	if st.Version != 1 || st.You.CanAct {
		t.Fatalf("reloaded room: version %d, canAct %v", st.Version, st.You.CanAct)
	}
	gate.waitStarted(t)
	gate.release <- struct{}{}
	a2.waitState(func(s wire.State) bool { return s.Version == 2 })
}

// TestJoiningAnotherRoomLeavesTheFirst checks that a connection is in one
// room at a time.
func TestJoiningAnotherRoomLeavesTheFirst(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	first, _ := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	b := h.client("user-b", "tab-b", "Bob")
	second, _ := b.create(wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(0), nil))
	a.join(second)
	if a.Joined() != second {
		t.Fatalf("joined %q, want %q", a.Joined(), second)
	}
	if snap := h.snapshot(first); snap.Clients != 0 {
		t.Fatalf("the first room still has %d connections", snap.Clients)
	}
	// Requests for the room it left are refused.
	err := h.m.Act(context.Background(), a.Client, wire.NewAct(first, 50, 0, "draft:pyramids", nil))
	if !errors.Is(err, room.ErrNotJoined) {
		t.Fatalf("act in the room it left: %v", err)
	}
	a.expectReject(50, wire.RejectCodeBadRequest)
	if err := h.m.Resync(context.Background(), a.Client, second); err != nil {
		t.Fatalf("Resync: %v", err)
	}
	h.snapshot(second)
	if st := a.current(); st.Code != second {
		t.Fatalf("resync answered with room %s", st.Code)
	}
}

// TestJoinWithCancelledContextLeavesNoMember checks that a join whose
// caller stops waiting (its connection closed mid-join) does not leave the
// connection attached: the room would otherwise keep a member that nobody
// ever detaches, report it as present, and never go idle.
func TestJoinWithCancelledContextLeavesNoMember(t *testing.T) {
	h := newHarness(t)
	owner := h.client("user-a", "tab-a", "Ann")
	code, _ := owner.create(wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(0), nil))
	h.m.Leave(owner.Client, code)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for i := range 50 {
		c := room.NewClient("user-a", "tab-x", "Ann")
		err := h.m.Join(ctx, c, code)
		if err == nil {
			// The attach won the race and the reply arrived: a real member.
			h.m.Disconnect(c)
			continue
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("join %d: %v", i, err)
		}
		if c.Joined() != "" {
			t.Fatalf("join %d failed but the client records room %q", i, c.Joined())
		}
		// The attach may have been handled (and its state frame sent)
		// before the caller gave up; only an error frame would be wrong.
		for _, f := range c.Out.Take() {
			if _, ok := f.(wire.Error); ok {
				t.Fatalf("join %d with a cancelled context sent %#v", i, f)
			}
		}
	}
	if snap := h.snapshot(code); snap.Clients != 0 {
		t.Fatalf("%d connections still attached after cancelled joins", snap.Clients)
	}
}

// evict detaches every given client and advances the clock until the room
// leaves memory.
func (h *harness) evict(code string, clients ...*tclient) {
	h.t.Helper()
	for _, c := range clients {
		h.m.Disconnect(c.Client)
	}
	h.snapshot(code) // the detaches are handled and the countdown armed
	h.clock.Advance(room.DefaultIdleTimeout)
	h.waitFor("eviction of "+code, func() bool { return !h.m.Loaded(code) })
}

// TestVersionSurvivesReloadAfterUndo checks that the room version is stored:
// after an undo the version is ahead of the action count, and a reload must
// not take it back, or an old base could name a different position.
func TestVersionSurvivesReloadAfterUndo(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, st := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	for v := 0; v < 3; v++ {
		a.act(st.View.Choices[0].Key, v, nil)
		st = a.waitState(func(s wire.State) bool { return s.Version == v+1 })
	}
	a.undo(3)
	a.waitState(func(s wire.State) bool { return s.Version == 4 })
	if hdr, err := h.st.PeekHeader(code); err != nil || hdr.Version != 4 || len(hdr.Actions) != 2 {
		t.Fatalf("room file after undo: %+v, %v", hdr, err)
	}
	h.evict(code, a)

	a2 := h.client("user-a", "tab-a2", "Ann")
	st = a2.join(code)
	if st.Version != 4 || len(h.snapshot(code).Actions) != 2 {
		t.Fatalf("reloaded room: version %d, want 4", st.Version)
	}
	// Version 2 was the position before the undone action; it must stay
	// stale even though the log has two actions again.
	seq := a2.act(st.View.Choices[0].Key, 2, nil)
	a2.expectReject(seq, wire.RejectCodeStale)
	a2.act(st.View.Choices[0].Key, 4, nil)
	a2.waitState(func(s wire.State) bool { return s.Version == 5 })
}

// TestRematchSurvivesReload checks that the rematch code is stored with the
// finished room, so a rematch asked after the room was evicted and reloaded
// leads to the same room instead of creating a second one.
func TestRematchSurvivesReload(t *testing.T) {
	h := newHarness(t)
	code, err := h.m.CreateFromSpec(room.Spec{
		Mode: store.ModeOnline,
		Seats: [2]store.Seat{
			{Kind: store.SeatKindHuman, UserID: "user-a", Name: "Ann"},
			{Kind: store.SeatKindHuman, UserID: "user-b", Name: "Bob"},
		},
		Seed:     3,
		Scenario: "military-supremacy",
		Actions:  mustScenarioTrigger(t, "military-supremacy"),
	})
	if err != nil {
		t.Fatalf("CreateFromSpec: %v", err)
	}
	a := h.client("user-a", "tab-a", "Ann")
	a.join(code)
	code2, err := h.m.Rematch(context.Background(), a.Client, code)
	if err != nil {
		t.Fatalf("Rematch: %v", err)
	}
	if hdr, err := h.st.PeekHeader(code); err != nil || hdr.Rematch != code2 {
		t.Fatalf("room file after the rematch: %+v, %v", hdr, err)
	}
	h.evict(code, a)

	b := h.client("user-b", "tab-b", "Bob")
	b.join(code)
	again, err := h.m.Rematch(context.Background(), b.Client, code)
	if err != nil || again != code2 {
		t.Fatalf("Rematch after reload: %q, %v; want %q", again, err, code2)
	}
	codes, err := h.st.Codes()
	if err != nil || len(codes) != 2 {
		t.Fatalf("rooms on disk: %v, %v; want the room and one rematch", codes, err)
	}
}

// TestRefusedActKeepsSeq checks that only an applied act or undo uses up its
// seq: a refused one (stale, illegal, not your turn) may be sent again with
// the same seq and is then judged afresh, while an applied one sent again is
// only acknowledged.
func TestRefusedActKeepsSeq(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, st := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	key := st.View.Choices[0].Key
	toMove := *st.View.ToMove

	a.actSeq(1, key, 5, nil) // stale
	a.expectReject(1, wire.RejectCodeStale)
	a.actSeq(1, "construct:no-such-card", 0, nil) // illegal
	a.expectReject(1, wire.RejectCodeIllegal)
	a.actSeq(1, key, 0, seatPtr(1-toMove)) // not your turn
	a.expectReject(1, wire.RejectCodeNotYourTurn)

	a.actSeq(1, key, 0, nil) // now right: applied
	st = a.waitState(func(s wire.State) bool { return s.Version == 1 })
	if st.Ack == nil || st.Ack.Seq != 1 {
		t.Fatalf("applied act: ack %+v", st.Ack)
	}
	a.actSeq(1, key, 0, nil) // applied already: only acknowledged
	h.snapshot(code)
	frames := a.drain()
	if len(frames) != 1 {
		t.Fatalf("re-sent applied act: %#v", frames)
	}
	if st, ok := frames[0].(wire.State); !ok || st.Version != 1 || st.Ack == nil || st.Ack.Seq != 1 {
		t.Fatalf("re-sent applied act: %#v", frames[0])
	}

	// The same holds for undo.
	if err := h.m.Undo(context.Background(), a.Client, wire.NewUndo(code, 2, 0)); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	a.expectReject(2, wire.RejectCodeStale)
	if err := h.m.Undo(context.Background(), a.Client, wire.NewUndo(code, 2, 1)); err != nil {
		t.Fatalf("Undo: %v", err)
	}
	st = a.waitState(func(s wire.State) bool { return s.Version == 2 })
	if st.Ack == nil || st.Ack.Seq != 2 {
		t.Fatalf("applied undo: ack %+v", st.Ack)
	}
	if snap := h.snapshot(code); snap.Version != 2 || len(snap.Actions) != 0 {
		t.Fatalf("after the undo: version %d, %d actions", snap.Version, len(snap.Actions))
	}
	if hdr, err := h.st.PeekHeader(code); err != nil || hdr.LastSeq["tab-a"] != 2 {
		t.Fatalf("stored seqs: %+v, %v", hdr, err)
	}
}

// TestLastSeqIsBounded checks that the room remembers the seqs of at most
// SeqClients client ids, the most recently used, in memory and on disk.
func TestLastSeqIsBounded(t *testing.T) {
	h := newHarness(t, func(c *room.Config) { c.SeqClients = 2 })
	owner := h.client("user-a", "tab-0", "Ann")
	code, st := owner.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	tabs := []*tclient{owner}
	for _, id := range []string{"tab-1", "tab-2"} {
		c := h.client("user-a", id, "Ann")
		c.join(code)
		tabs = append(tabs, c)
	}
	for v, c := range tabs {
		h.snapshot(code)
		st = c.current()
		c.act(st.View.Choices[0].Key, v, nil)
		c.waitState(func(s wire.State) bool { return s.Version == v+1 })
	}
	hdr, err := h.st.PeekHeader(code)
	if err != nil {
		t.Fatalf("PeekHeader: %v", err)
	}
	if len(hdr.LastSeq) != 2 || hdr.LastSeq["tab-1"] != 1 || hdr.LastSeq["tab-2"] != 1 {
		t.Fatalf("stored seqs %v, want tab-1 and tab-2 only", hdr.LastSeq)
	}
	h.evict(code, tabs...)

	// tab-2 is remembered after the reload; tab-0 was forgotten, so its
	// re-sent act is judged again, and refused as stale by its old base.
	t2 := h.client("user-a", "tab-2", "Ann")
	t2.join(code)
	t2.actSeq(1, "anything", 2, nil)
	h.snapshot(code)
	if frames := t2.drain(); len(frames) != 1 || frames[0].(wire.State).Ack == nil {
		t.Fatalf("remembered seq after reload: %#v", frames)
	}
	t0 := h.client("user-a", "tab-0", "Ann")
	t0.join(code)
	t0.actSeq(1, "anything", 0, nil)
	t0.expectReject(1, wire.RejectCodeStale)
}

// TestSeatHolderAdmittedWhenFull checks that spectators cannot lock a player
// out: MaxClients bounds the connections a new spectator may join, while a
// user who holds a seat is refused only once that user alone has MaxClients
// connections.
func TestSeatHolderAdmittedWhenFull(t *testing.T) {
	h := newHarness(t, func(c *room.Config) { c.MaxClients = 3 })
	a := h.client("user-a", "tab-a", "Ann")
	code, _ := a.create(wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(0), nil))
	h.client("user-b", "tab-b", "Bob").join(code)
	h.client("user-c", "tab-c", "Cy").join(code)
	h.m.Disconnect(a.Client)
	h.client("user-d", "tab-d", "Di").join(code) // the room is full again

	a2 := h.client("user-a", "tab-a2", "Ann")
	st := a2.join(code)
	checkSeat(t, st, wire.SeatInfo{Seat: 0, Name: "Ann", Kind: wire.SeatKindHuman, Connected: true, You: true})
	if e, err := h.client("user-e", "tab-e", "Ed").joinErr(code); e.Code != wire.ErrorCodeRoomFull || !errors.Is(err, room.ErrRoomFull) {
		t.Fatalf("a new spectator in a full room: %v, %v", e, err)
	}

	// One user's own tabs are bounded too.
	h.client("user-a", "tab-a3", "Ann").join(code)
	h.client("user-a", "tab-a4", "Ann").join(code)
	if e, _ := h.client("user-a", "tab-a5", "Ann").joinErr(code); e.Code != wire.ErrorCodeRoomFull {
		t.Fatalf("a fourth tab of one user: %v", e)
	}
	if snap := h.snapshot(code); snap.Clients != 6 || snap.Spectators != 2 {
		t.Fatalf("snapshot: %d clients, %d spectators", snap.Clients, snap.Spectators)
	}
}

// TestStateFrameExtras checks the frame fields the client cannot work out
// from the view: each seat's bot level, whether this connection may undo
// now, and which seat's bot is searching.
func TestStateFrameExtras(t *testing.T) {
	gate := newGateBot()
	h := newHarness(t, func(c *room.Config) {
		c.NewBot = func(store.BotLevel, uint64) room.Bot { return gate }
	})

	// A bot room whose bot drafts first: it searches from the first frame.
	code, err := h.m.CreateFromSpec(aiSpec(engine.FirstPlayerSeat1, store.BotHard))
	if err != nil {
		t.Fatalf("CreateFromSpec: %v", err)
	}
	a := h.client("user-a", "tab-a", "Ann")
	st := a.join(code)
	if st.Room.Seats[0].Bot != nil || st.Room.Seats[1].Bot == nil || *st.Room.Seats[1].Bot != wire.BotLevelHard {
		t.Fatalf("seats %+v: want a hard bot on seat 1 only", st.Room.Seats)
	}
	if st.Thinking == nil || *st.Thinking != 1 || st.CanUndo {
		t.Fatalf("bot to move first: thinking %v, canUndo %v", st.Thinking, st.CanUndo)
	}
	gate.waitStarted(t)
	gate.release <- struct{}{}
	st = a.waitState(func(s wire.State) bool { return s.You.CanAct })
	if st.Thinking != nil || st.CanUndo {
		t.Fatalf("after the bot's first pick: thinking %v, canUndo %v (the human has nothing to undo)", st.Thinking, st.CanUndo)
	}
	// The human plays until the bot is to move again.
	for i := 0; ; i++ {
		if i > 4 {
			t.Fatalf("the bot never got the move back")
		}
		v := st.Version
		a.act(st.View.Choices[0].Key, v, nil)
		st = a.waitState(func(s wire.State) bool { return s.Version == v+1 })
		if !st.CanUndo {
			t.Fatalf("version %d: canUndo false after a human move", st.Version)
		}
		if *st.View.ToMove == 1 {
			break
		}
		if st.Thinking != nil {
			t.Fatalf("version %d: the human is to move but seat %d is thinking", st.Version, *st.Thinking)
		}
	}
	if st.Thinking == nil || *st.Thinking != 1 {
		t.Fatalf("version %d: the bot is to move but thinking is %v", st.Version, st.Thinking)
	}
	if snap := h.snapshot(code); !snap.BotThinking {
		t.Fatalf("the frame says thinking but no search runs")
	}
	gate.waitStarted(t)
	gate.release <- struct{}{}

	// Hot-seat: no bot, undo once there is a move to take back.
	b := h.client("user-b", "tab-b", "Bob")
	_, st = b.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	if st.CanUndo || st.Thinking != nil || st.Room.Seats[0].Bot != nil || st.Room.Seats[1].Bot != nil {
		t.Fatalf("new hot-seat room: canUndo %v, thinking %v, seats %+v", st.CanUndo, st.Thinking, st.Room.Seats)
	}
	b.act(st.View.Choices[0].Key, 0, nil)
	if st = b.waitState(func(s wire.State) bool { return s.Version == 1 }); !st.CanUndo || st.Thinking != nil {
		t.Fatalf("hot-seat after a move: canUndo %v, thinking %v", st.CanUndo, st.Thinking)
	}

	// Online: never undo, for players or spectators.
	c := h.client("user-c", "tab-c", "Cy")
	online, _ := c.create(wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(0), nil))
	d := h.client("user-d", "tab-d", "Di")
	d.join(online)
	e := h.client("user-e", "tab-e", "Ed")
	e.join(online)
	h.snapshot(online)
	mover := [2]*tclient{c, d}[*c.current().View.ToMove]
	mover.act(mover.current().View.Choices[0].Key, 0, nil)
	h.snapshot(online)
	for _, cl := range []*tclient{c, d, e} {
		if st := cl.current(); st.Version != 1 || st.CanUndo || st.Thinking != nil {
			t.Fatalf("online %s: version %d, canUndo %v, thinking %v", cl.ClientID, st.Version, st.CanUndo, st.Thinking)
		}
	}
}

// TestNoUndoAfterGameOver checks that a finished game cannot be taken back,
// and that its frames say so.
func TestNoUndoAfterGameOver(t *testing.T) {
	h := newHarness(t)
	code, err := h.m.CreateFromSpec(room.Spec{
		Mode:     store.ModeHotseat,
		Seats:    [2]store.Seat{{Kind: store.SeatKindOpen}, {Kind: store.SeatKindOpen}},
		Seed:     3,
		Scenario: "military-supremacy",
		Actions:  mustScenarioTrigger(t, "military-supremacy"),
	})
	if err != nil {
		t.Fatalf("CreateFromSpec: %v", err)
	}
	a := h.client("user-a", "tab-a", "Ann")
	st := a.join(code)
	if st.Room.Status != wire.RoomStatusFinished || st.CanUndo {
		t.Fatalf("finished hot-seat room: status %s, canUndo %v", st.Room.Status, st.CanUndo)
	}
	seq := a.undo(st.Version)
	a.expectReject(seq, wire.RejectCodeIllegal)
	if snap := h.snapshot(code); len(snap.Actions) != len(mustScenarioTrigger(t, "military-supremacy")) {
		t.Fatalf("the undo changed the log: %v", snap.Actions)
	}
}

// TestUndoFrameCarriesNoUndoneEvents checks the outbox merge end to end: a
// client that has not read the frame of a move before the move is undone
// receives one frame, the undone position, without the move's events.
func TestUndoFrameCarriesNoUndoneEvents(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, st := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	a.act(st.View.Choices[0].Key, 0, nil)
	seq := a.undo(1)
	h.snapshot(code)
	frames := a.drain()
	if len(frames) != 1 {
		t.Fatalf("got %d frames, want the undo's only: %#v", len(frames), frames)
	}
	got, ok := frames[0].(wire.State)
	if !ok || got.Version != 2 || got.View.Ply != 0 || got.Ack == nil || got.Ack.Seq != seq {
		t.Fatalf("frame after act and undo: %#v", frames[0])
	}
	if len(got.Events) != 0 {
		t.Fatalf("the undo's frame carries %d events of the undone move", len(got.Events))
	}
}
