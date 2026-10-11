package room_test

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/view"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// The forfeit (docs/protocol.md, act): an act whose key is resign:<seat>
// ends the game at once with the other seat's victory. A seated player
// forfeits their own seat at any time (in a hot-seat room, the seat to
// move); a spectator cannot, and nothing can after the end.

// checkForfeit checks a finished frame: a forfeit won by winner, with the
// resign and gameOver events, and no undo.
func checkForfeit(t *testing.T, st wire.State, winner int) {
	t.Helper()
	r := st.View.Result
	if st.Room.Status != wire.RoomStatusFinished || r == nil || r.Victory != view.VictoryKindForfeit || r.Winner == nil || *r.Winner != winner {
		t.Fatalf("after the forfeit: status %s, result %+v", st.Room.Status, r)
	}
	if st.CanUndo || st.You.CanAct || st.View.Revealed == nil {
		t.Fatalf("after the forfeit: canUndo %v, canAct %v, revealed %v", st.CanUndo, st.You.CanAct, st.View.Revealed != nil)
	}
	if len(st.Events) != 2 || st.Events[0].Kind != view.EventKindResign || *st.Events[0].Seat != 1-winner ||
		st.Events[1].Kind != view.EventKindGameOver {
		t.Fatalf("the forfeit's events: %+v", st.Events)
	}
}

func TestForfeitOnline(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, _ := a.create(wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(0), nil))

	// Not in the lobby: the game has not started.
	seq := a.act("resign:0", 0, nil)
	a.expectReject(seq, wire.RejectCodeNotYourTurn)

	b := h.client("user-b", "tab-b", "Bob")
	st := b.join(code)
	sp := h.client("user-c", "tab-c", "Cy")
	sp.join(code)
	// Two moves, so the forfeit comes in the middle of the draft.
	for v := range 2 {
		h.snapshot(code)
		mover := a
		if *a.current().View.ToMove == 1 {
			mover = b
		}
		b.current()
		mover.act(mover.last.View.Choices[0].Key, v, nil)
	}
	h.snapshot(code)
	st = b.current()
	a.current()
	sp.drain()

	// Nobody forfeits another seat, and a spectator none at all.
	seq = a.act("resign:1", st.Version, nil)
	a.expectReject(seq, wire.RejectCodeNotYourTurn)
	seq = a.act("resign:0", st.Version, seatPtr(1))
	a.expectReject(seq, wire.RejectCodeNotYourTurn)
	seq = sp.act("resign:1", st.Version, nil)
	sp.expectReject(seq, wire.RejectCodeNotYourTurn)
	// A stale base is refused, as for any act.
	seq = b.act("resign:1", st.Version-1, nil)
	b.expectReject(seq, wire.RejectCodeStale)

	// Seat 1 forfeits whether or not it is to move.
	h.snapshot(code)
	b.drain()
	seq = b.act("resign:1", st.Version, nil)
	h.snapshot(code)
	got := b.current()
	if got.Ack == nil || got.Ack.Seq != seq {
		t.Fatalf("the forfeit's ack: %+v", got.Ack)
	}
	checkForfeit(t, got, 0)
	checkForfeit(t, a.current(), 0)
	checkForfeit(t, sp.current(), 0)

	snap := h.snapshot(code)
	if snap.Status != store.StatusFinished || snap.Actions[len(snap.Actions)-1] != "resign:1" || snap.History != len(snap.Actions) {
		t.Fatalf("the room after the forfeit: status %s, actions %v", snap.Status, snap.Actions)
	}

	// After the end nothing more, a forfeit included.
	seq = a.act("resign:0", got.Version, nil)
	a.expectReject(seq, wire.RejectCodeIllegal)

	// The forfeit is stored: a reload replays it.
	f, s, err := h.st.Load(code)
	if err != nil || f.Status != store.StatusFinished || f.Result == nil || f.Result.Kind != engine.VictoryForfeit || !s.Over() {
		t.Fatalf("the stored room: %v, status %s, result %+v", err, f.Status, f.Result)
	}
}

func TestForfeitHotseat(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, st := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	for v := range 3 {
		a.act(st.View.Choices[0].Key, v, nil)
		st = a.waitState(func(s wire.State) bool { return s.Version == v+1 })
	}
	toMove := *st.View.ToMove
	// Only the player to move forfeits on a shared device.
	seq := a.act(engine.ResignAction(engine.Seat(1-toMove)).Key(), st.Version, nil)
	a.expectReject(seq, wire.RejectCodeNotYourTurn)
	seq = a.act(engine.ResignAction(engine.Seat(toMove)).Key(), st.Version, seatPtr(1-toMove))
	a.expectReject(seq, wire.RejectCodeNotYourTurn)

	seq = a.act(engine.ResignAction(engine.Seat(toMove)).Key(), st.Version, seatPtr(toMove))
	st = a.waitState(func(s wire.State) bool { return s.Ack != nil && s.Ack.Seq == seq })
	checkForfeit(t, st, 1-toMove)

	// A forfeit cannot be undone.
	seq = a.undo(st.Version)
	a.expectReject(seq, wire.RejectCodeIllegal)
	if snap := h.snapshot(code); len(snap.Actions) != 4 {
		t.Fatalf("actions after the refused undo: %v", snap.Actions)
	}
	// A join after the end carries the whole history, the forfeit last.
	b := h.client("user-a", "tab-b", "Ann")
	st = b.join(code)
	last := (*st.History)[len(*st.History)-1]
	if len(*st.History) != 4 || last.Seat != toMove || last.Events[0].Kind != view.EventKindResign {
		t.Fatalf("the history after a forfeit: %+v", st.History)
	}
}

// TestForfeitAI: the human forfeits while the bot is thinking; the search
// is cancelled and its late answer dropped.
func TestForfeitAI(t *testing.T) {
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
	st = a.waitState(func(s wire.State) bool { return s.Version == 1 })
	if st.Thinking == nil {
		t.Fatalf("the bot is not thinking")
	}
	// The bot's seat is not the human's to forfeit.
	seq := a.act("resign:1", 1, nil)
	a.expectReject(seq, wire.RejectCodeNotYourTurn)

	seq = a.act("resign:0", 1, nil)
	st = a.waitState(func(s wire.State) bool { return s.Ack != nil && s.Ack.Seq == seq })
	checkForfeit(t, st, 1)
	if st.Thinking != nil {
		t.Fatalf("the bot still thinks after the forfeit")
	}
	h.waitFor("the cancelled search to answer", func() bool { return h.snapshot(code).DroppedBotResults == 1 })
	if snap := h.snapshot(code); len(snap.Actions) != 2 || snap.BotThinking {
		t.Fatalf("after the forfeit: actions %v, thinking %v", snap.Actions, snap.BotThinking)
	}
}

// TestForfeitReplaysOnLoad: a room file whose log ends in a forfeit loads
// back, with its history.
func TestForfeitReplaysOnLoad(t *testing.T) {
	h := newHarness(t)
	code, err := h.m.CreateFromSpec(room.Spec{
		Mode:    store.ModeHotseat,
		Seats:   [2]store.Seat{{Kind: store.SeatKindOpen}, {Kind: store.SeatKindOpen}},
		Seed:    3,
		Actions: []string{},
	})
	if err != nil {
		t.Fatal(err)
	}
	a := h.client("user-a", "tab-a", "Ann")
	st := a.join(code)
	toMove := *st.View.ToMove
	a.act(engine.ResignAction(engine.Seat(toMove)).Key(), 0, nil)
	st = a.waitState(func(s wire.State) bool { return s.Version == 1 })
	checkForfeit(t, st, 1-toMove)

	h.m.Leave(a.Client, code)
	h.snapshot(code)
	h.clock.Advance(room.DefaultIdleTimeout)
	h.waitFor("eviction", func() bool { return !h.m.Loaded(code) })
	b := h.client("user-a", "tab-b", "Ann")
	st = b.join(code)
	if st.View.Result == nil || st.View.Result.Victory != view.VictoryKindForfeit || len(*st.History) != 1 {
		t.Fatalf("the reloaded forfeit: result %+v, history %+v", st.View.Result, st.History)
	}
}
