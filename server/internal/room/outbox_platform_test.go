package room

import (
	"fmt"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func TestTableOutboxSlowClientAndUndo(t *testing.T) {
	history := []wire.TableHistoryEntry{{Batch: 1, Events: []wire.TableEvent{{}}}}
	first := wire.NewTableState("REGAAA")
	first.Revision, first.Batch = 3, 1
	first.History = &history
	first.Ack = &wire.PlatformAck{ClientID: "tab", Seq: 1}
	next := wire.NewTableState(first.Code)
	next.Revision, next.Batch = 4, 2
	next.Events = []wire.TableEvent{{}}
	presence := next
	presence.Revision = 5
	presence.Events = []wire.TableEvent{}
	presence.Pending = &wire.TablePending{Key: "owner-only", Confirm: false}
	o := NewOutbox()
	o.Push(first)
	o.Push(next)
	o.Push(presence)
	frames := o.Take()
	if len(frames) != 1 {
		t.Fatalf("slow client retained %d snapshots", len(frames))
	}
	got := frames[0].(wire.TableState)
	if got.Revision != 5 || got.Batch != 2 || got.History != &history || len(got.Events) != 1 || got.Ack != first.Ack || got.Pending != presence.Pending {
		t.Fatalf("lost forward history/event/ack or latest private pick: %+v", got)
	}

	for _, tc := range []struct {
		name            string
		revision, batch int
	}{
		{"undo", 6, 0}, {"backward-revision", 2, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			o.Push(got)
			rollback := wire.NewTableState(first.Code)
			rollback.Revision, rollback.Batch = tc.revision, tc.batch
			o.Push(rollback)
			st := o.Take()[0].(wire.TableState)
			if st.History != nil || len(st.Events) != 0 || st.Pending != nil || st.Ack != first.Ack {
				t.Fatalf("rollback inherited stale history/events/private pick: %+v", st)
			}
		})
	}
	// A resync's own complete history supersedes all accumulated old events.
	o.Push(got)
	full := wire.NewTableState(first.Code)
	full.Revision, full.Batch = 6, 2
	replacement := []wire.TableHistoryEntry{{Batch: 2, Events: []wire.TableEvent{}}}
	full.History = &replacement
	full.Ack = &wire.PlatformAck{ClientID: "tab", Seq: 2}
	o.Push(full)
	st := o.Take()[0].(wire.TableState)
	if st.History != &replacement || len(st.Events) != 0 || st.Ack != full.Ack {
		t.Fatal("full history or newest ack lost")
	}
}

func TestTableOutboxBoundsAndFamilySwitch(t *testing.T) {
	o := NewOutbox()
	for i := range MaxMergedEvents + 3 {
		st := wire.NewTableState("REGAAA")
		st.Revision, st.Batch = i, i
		st.Events = []wire.TableEvent{{}}
		o.Push(st)
	}
	if n := len(o.Take()[0].(wire.TableState).Events); n != MaxMergedEvents {
		t.Fatalf("merged %d events", n)
	}
	if o.Overflowed() {
		t.Fatal("event truncation disconnected slow client")
	}
	// A common code has one snapshot slot, including a lobby-to-game handoff.
	lobby := wire.NewLobbyState("REGAAA")
	lobby.Ack = &wire.PlatformAck{ClientID: "old-lobby", Seq: 9}
	o.Push(lobby)
	o.Push(wire.NewTableState("REGAAA"))
	frames := o.Take()
	if len(frames) != 1 || frames[0].(wire.TableState).Ack != nil {
		t.Fatal("cross-family state/ack merge")
	}
	o.Push(wire.NewTableState("REGAAA"))
	o.Push(wire.State{Code: "REGAAA"})
	if frames := o.Take(); len(frames) != 1 {
		t.Fatal("Duel handoff retained old-family snapshot")
	}
	// All three concrete families share the existing four-room bound.
	for i := range MaxPendingStates {
		code := fmt.Sprintf("ROOM%d", i)
		switch i % 3 {
		case 0:
			o.Push(wire.State{Code: code})
		case 1:
			o.Push(wire.NewTableState(code))
		case 2:
			o.Push(wire.NewLobbyState(code))
		}
	}
	o.Push(wire.NewTableState("EXTRA"))
	if !o.Overflowed() || len(o.Take()) != MaxPendingStates {
		t.Fatal("snapshot families escaped common bound")
	}
}

func TestLobbyOutboxLatestAndClose(t *testing.T) {
	o := NewOutbox()
	old := wire.NewLobbyState("REGAAA")
	old.Ack = &wire.PlatformAck{ClientID: "tab", Seq: 1}
	newer := wire.NewLobbyState(old.Code)
	newer.Revision = 2
	o.Push(old)
	o.Push(newer)
	frames := o.Take()
	if len(frames) != 1 || frames[0].(wire.LobbyState).Ack != old.Ack || frames[0].(wire.LobbyState).Revision != 2 {
		t.Fatal("lobby latest/ack lost")
	}
	o.Close()
	o.Push(newer)
	if len(o.Take()) != 0 {
		t.Fatal("closed outbox retained snapshot")
	}
}
