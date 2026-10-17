package protocoltest

import (
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/view"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// Pin the resync/live-move interleaving without scheduler timing: the real
// outbox merges two moves and a presence update into a pending resync.
func TestResyncHistoryAfterOutboxMerge(t *testing.T) {
	want := []wire.HistoryEntry{
		{Ply: 0, Seat: 0, Events: []view.GameEvent{{Kind: view.EventKindDraft}}},
		{Ply: 1, Seat: 1, Events: []view.GameEvent{{Kind: view.EventKindPay}, {Kind: view.EventKindBuild}}},
		{Ply: 2, Seat: 0, Events: []view.GameEvent{{Kind: view.EventKindDiscard}}},
	}
	frame := func(ply int, events []view.GameEvent) wire.State {
		return wire.NewState("HISTRY", ply, wire.NewRoomInfo(wire.ModeAI, wire.RoomStatusPlaying, nil, 0),
			wire.NewYouInfo(nil, false), nil, events, view.View{Ply: ply})
	}
	prefix := want[:1]
	resync := frame(1, nil)
	resync.History = &prefix
	o := room.NewOutbox()
	o.Push(resync)
	o.Push(frame(2, want[1].Events))
	o.Push(frame(3, want[2].Events))
	o.Push(frame(3, nil)) // a same-position update retains both moves
	frames := o.Take()
	if len(frames) != 1 {
		t.Fatalf("got %d pending frames, want 1", len(frames))
	}
	merged := frames[0].(wire.State)
	if merged.History == nil || len(*merged.History) != 1 || merged.View.Ply != 3 {
		t.Fatalf("expected a one-entry history prefix at ply 3: %+v", merged)
	}
	if err := resyncHistoryError(merged, want); err != nil {
		t.Fatal(err)
	}

	// A fresh resync includes both moves in its history and clears the
	// live suffix, so it passes the same complete-content check.
	complete := frame(3, nil)
	complete.History = &want
	o.Push(merged)
	o.Push(complete)
	if err := resyncHistoryError(o.Take()[0].(wire.State), want); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{
		"no history", "missing move", "duplicate move", "reordered moves",
		"history past ply", "wrong history ply", "wrong history seat", "wrong history event",
	} {
		t.Run(name, func(t *testing.T) {
			bad := merged
			history := slices.Clone(*merged.History)
			bad.History = &history
			bad.Events = slices.Clone(merged.Events)
			switch name {
			case "no history":
				bad.History = nil
			case "missing move":
				bad.Events = bad.Events[:len(bad.Events)-1]
			case "duplicate move":
				bad.Events = append(bad.Events, bad.Events[0])
			case "reordered moves":
				bad.Events[0], bad.Events[2] = bad.Events[2], bad.Events[0]
			case "history past ply":
				history = append(history, want...)
			case "wrong history ply":
				history[0].Ply++
			case "wrong history seat":
				history[0].Seat = 1
			case "wrong history event":
				history[0].Events = []view.GameEvent{{Kind: view.EventKindBuild}}
			}
			if err := resyncHistoryError(bad, want); err == nil {
				t.Fatal("accepted corrupted history/events")
			}
		})
	}
}
