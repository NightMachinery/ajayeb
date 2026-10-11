package room_test

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/view"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// The history (docs/protocol.md, the state frame): the first frame of a
// join, every answer to a resync and the frames after an undo carry every
// applied action of the game so far, each with its ply, its seat and its
// events filtered for the viewer. Live frames do not.

// resync asks for a fresh state and returns it.
func (c *tclient) resync() wire.State {
	c.t.Helper()
	if err := c.h.m.Resync(context.Background(), c.Client, c.code); err != nil {
		c.t.Fatalf("Resync: %v", err)
	}
	c.h.snapshot(c.code)
	return c.current()
}

// historyEvents concatenates the events of a history.
func historyEvents(h []wire.HistoryEntry) []view.GameEvent {
	out := []view.GameEvent{}
	for _, e := range h {
		out = append(out, e.Events...)
	}
	return out
}

// playOnline plays n actions of an online room between a and b, taking the
// first choice each time, and returns every live event each of them
// received, in order.
func playOnline(t *testing.T, h *harness, a, b *tclient, n int) (evA, evB []view.GameEvent, seats []int) {
	t.Helper()
	evA, evB = []view.GameEvent{}, []view.GameEvent{}
	for v := range n {
		h.snapshot(a.code)
		stA, stB := a.current(), b.current()
		if stA.History != nil || stB.History != nil {
			if v > 0 {
				t.Fatalf("version %d: a live frame carries a history", stA.Version)
			}
		}
		if v > 0 {
			evA, evB = append(evA, stA.Events...), append(evB, stB.Events...)
		}
		mover := a
		if *stA.View.ToMove == 1 {
			mover = b
		}
		st := mover.last
		seats = append(seats, *st.View.ToMove)
		mover.act(st.View.Choices[0].Key, st.Version, nil)
	}
	h.snapshot(a.code)
	stA, stB := a.current(), b.current()
	if stA.History != nil || stB.History != nil {
		t.Fatalf("the last live frame carries a history")
	}
	return append(evA, stA.Events...), append(evB, stB.Events...), seats
}

func TestHistoryOnJoinAndResync(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, st := a.create(wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(0), nil))
	if st.History == nil || len(*st.History) != 0 {
		t.Fatalf("the creator's first frame: history %v, want an empty one", st.History)
	}
	b := h.client("user-b", "tab-b", "Bob")
	if st := b.join(code); st.History == nil || len(*st.History) != 0 {
		t.Fatalf("the joiner's first frame: history %v", st.History)
	}
	// The creator learns of the join through a presence frame, without
	// the history.
	h.snapshot(code)
	if st := a.current(); st.History != nil {
		t.Fatalf("a presence frame carries a history")
	}

	const n = 14
	evA, evB, seats := playOnline(t, h, a, b, n)

	for _, c := range []struct {
		who  *tclient
		live []view.GameEvent
	}{{a, evA}, {b, evB}} {
		st := c.who.resync()
		if st.History == nil || len(*st.History) != n || len(st.Events) != 0 {
			t.Fatalf("resync: history %v, events %d", st.History, len(st.Events))
		}
		for i, e := range *st.History {
			if e.Ply != i || e.Seat != seats[i] {
				t.Fatalf("entry %d: ply %d seat %d, want ply %d seat %d", i, e.Ply, e.Seat, i, seats[i])
			}
		}
		if got := historyEvents(*st.History); !reflect.DeepEqual(got, c.live) {
			t.Fatalf("the history's events differ from the live ones:\n got %+v\nwant %+v", got, c.live)
		}
	}

	// A spectator who joins late sees the same game, filtered for a
	// spectator, and a second tab of a player its own seat's.
	sp := h.client("user-c", "tab-c", "Cy")
	if st := sp.join(code); st.History == nil || len(*st.History) != n {
		t.Fatalf("the spectator's first frame: history %v", st.History)
	}
	a2 := h.client("user-a", "tab-a2", "Ann")
	st = a2.join(code)
	if st.History == nil || !reflect.DeepEqual(historyEvents(*st.History), evA) {
		t.Fatalf("a second tab's history differs from the first tab's live events")
	}
	// Joining again a room the connection is in also answers the history.
	if err := h.m.Join(context.Background(), a2.Client, code); err != nil {
		t.Fatal(err)
	}
	h.snapshot(code)
	if st := a2.current(); st.History == nil || len(*st.History) != n {
		t.Fatalf("a repeated join: history %v", st.History)
	}
}

// TestHistoryAfterReload: a room loaded from disk rebuilds its history by
// replay, so a join after an eviction sees the whole game.
func TestHistoryAfterReload(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, st := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	for v := range 9 {
		a.act(st.View.Choices[0].Key, v, seatPtr(*st.View.ToMove))
		st = a.waitState(func(s wire.State) bool { return s.Version == v+1 })
	}
	before := a.resync()

	h.m.Leave(a.Client, code)
	h.snapshot(code)
	h.clock.Advance(room.DefaultIdleTimeout)
	h.waitFor("eviction", func() bool { return !h.m.Loaded(code) })

	b := h.client("user-a", "tab-b", "Ann")
	after := b.join(code)
	if before.History == nil || after.History == nil || !reflect.DeepEqual(*before.History, *after.History) {
		t.Fatalf("the reloaded room's history differs:\nbefore %+v\nafter  %+v", before.History, after.History)
	}
	if len(*after.History) != 9 {
		t.Fatalf("%d entries after the reload, want 9", len(*after.History))
	}
}

// TestHistoryAfterUndo: every connection's frame after an undo carries the
// shortened history, and the undone move's events are gone from it.
func TestHistoryAfterUndo(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, st := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	a2 := h.client("user-a", "tab-a2", "Ann")
	a2.join(code)
	for v := range 3 {
		a.act(st.View.Choices[0].Key, v, nil)
		st = a.waitState(func(s wire.State) bool { return s.Version == v+1 })
	}
	h.snapshot(code)
	a2.drain()
	a.undo(3)
	h.snapshot(code)
	for _, c := range []*tclient{a2, a} {
		st = c.current()
		if st.History == nil || len(*st.History) != 2 || len(st.Events) != 0 {
			t.Fatalf("after the undo: history %v, events %d", st.History, len(st.Events))
		}
	}
	// The next move's frame is live again.
	a.act(st.View.Choices[0].Key, 4, nil)
	h.snapshot(code)
	if st := a2.current(); st.History != nil || len(st.Events) == 0 {
		t.Fatalf("the move after an undo: history %v, events %d", st.History, len(st.Events))
	}
}

// TestHistoryBound: a game never has more actions than a history holds.
func TestHistoryBound(t *testing.T) {
	if engine.MaxGameActions > wire.MaxHistory {
		t.Fatalf("a game may take %d actions, but a history holds only %d", engine.MaxGameActions, wire.MaxHistory)
	}
}

// TestHistoryOfAWholeGame plays a hot-seat game to its end and checks that
// the final resync holds one entry per action, below the bound, and that
// the frame stays a reasonable size.
func TestHistoryOfAWholeGame(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab-a", "Ann")
	code, st := a.create(wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil))
	for st.Room.Status != wire.RoomStatusFinished {
		a.act(st.View.Choices[0].Key, st.Version, nil)
		v := st.Version + 1
		st = a.waitState(func(s wire.State) bool { return s.Version == v })
	}
	st = a.resync()
	snap := h.snapshot(code)
	if st.History == nil || len(*st.History) != len(snap.Actions) || snap.History != len(snap.Actions) {
		t.Fatalf("history %d entries, room %d, actions %d", len(*st.History), snap.History, len(snap.Actions))
	}
	b, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	live, err := json.Marshal(wire.NewState(st.Code, st.Version, st.Room, st.You, nil, nil, st.View))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("a whole game: %d actions; the final frame is %d bytes with its history, %d without", len(snap.Actions), len(b), len(live))
	// A sanity bound, far above a real game's frame (about 100 KB).
	if len(b) > 512<<10 {
		t.Fatalf("the history frame is %d bytes", len(b))
	}
}
