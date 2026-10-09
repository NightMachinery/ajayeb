package room

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/view"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// outboxState is a state frame at version whose game is at ply version (as
// in a room that has seen no undo), with an event of each kind.
func outboxState(code string, version int, ack *wire.Ack, kinds ...view.EventKind) wire.State {
	return outboxStateAt(code, version, version, ack, kinds...)
}

// outboxStateAt is outboxState with a ply of its own, for frames after an
// undo, which moves the version on and the ply back.
func outboxStateAt(code string, version, ply int, ack *wire.Ack, kinds ...view.EventKind) wire.State {
	events := make([]view.GameEvent, 0, len(kinds))
	for _, k := range kinds {
		events = append(events, view.GameEvent{Kind: k})
	}
	return wire.NewState(code, version, wire.NewRoomInfo(wire.ModeOnline, wire.RoomStatusPlaying, nil, 0),
		wire.NewYouInfo(nil, false), ack, events, view.View{Ply: ply})
}

// outboxKinds lists the kinds of a frame's events.
func outboxKinds(st wire.State) []view.EventKind {
	out := make([]view.EventKind, 0, len(st.Events))
	for _, e := range st.Events {
		out = append(out, e.Kind)
	}
	return out
}

// TestOutboxMergeFollowsThePosition pins which events a replacing state
// frame inherits. Events animate the way from the position the client last
// took to the new one, so they are kept while the game only moves forward
// (a newer move, or a presence change or resync at the same version and
// ply) and dropped when the replacing frame does not continue from the
// replaced one: an undo (version up, ply back) or a version that went back.
func TestOutboxMergeFollowsThePosition(t *testing.T) {
	build, pay, reveal := view.EventKindBuild, view.EventKindPay, view.EventKindReveal
	cases := []struct {
		name   string
		frames []wire.State
		want   []view.EventKind
	}{
		{"same position keeps them", []wire.State{
			outboxState("AAAAAA", 5, nil, build), outboxState("AAAAAA", 5, nil)}, []view.EventKind{build}},
		{"a forward move appends", []wire.State{
			outboxState("AAAAAA", 5, nil, build), outboxState("AAAAAA", 6, nil, pay)}, []view.EventKind{build, pay}},
		{"an undo drops them", []wire.State{
			outboxStateAt("AAAAAA", 5, 5, nil, build), outboxStateAt("AAAAAA", 6, 4, nil)}, []view.EventKind{}},
		{"a move after an undo keeps only its own", []wire.State{
			outboxStateAt("AAAAAA", 5, 5, nil, build), outboxStateAt("AAAAAA", 6, 4, nil), outboxStateAt("AAAAAA", 7, 5, nil, reveal)},
			[]view.EventKind{reveal}},
		{"an older version drops them", []wire.State{
			outboxState("AAAAAA", 5, nil, build), outboxState("AAAAAA", 4, nil, pay)}, []view.EventKind{pay}},
	}
	for _, tc := range cases {
		o := NewOutbox()
		ack := &wire.Ack{ClientID: "c", Seq: 9}
		for i, f := range tc.frames {
			if i == 0 {
				f.Ack = ack
			}
			o.Push(f)
		}
		got := o.Take()
		if len(got) != 1 {
			t.Fatalf("%s: %d frames, want 1", tc.name, len(got))
		}
		st := got[0].(wire.State)
		if st.Ack != ack {
			t.Fatalf("%s: the ack was lost: %+v", tc.name, st.Ack)
		}
		kinds := outboxKinds(st)
		if len(kinds) != len(tc.want) {
			t.Fatalf("%s: events %v, want %v", tc.name, kinds, tc.want)
		}
		for i := range kinds {
			if kinds[i] != tc.want[i] {
				t.Fatalf("%s: events %v, want %v", tc.name, kinds, tc.want)
			}
		}
	}
}

func TestOutboxLatestStateWins(t *testing.T) {
	o := NewOutbox()
	ack := &wire.Ack{ClientID: "c", Seq: 3}
	o.Push(outboxState("AAAAAA", 1, ack, view.EventKindBuild))
	o.Push(wire.NewReject(4, wire.RejectCodeStale, ""))
	o.Push(outboxState("AAAAAA", 2, nil, view.EventKindPay, view.EventKindReveal))
	select {
	case <-o.Ready():
	default:
		t.Fatalf("Ready did not signal")
	}
	got := o.Take()
	if len(got) != 2 {
		t.Fatalf("Take returned %d frames, want 2: %#v", len(got), got)
	}
	if r, ok := got[0].(wire.Reject); !ok || r.Seq != 4 {
		t.Fatalf("first frame: %#v", got[0])
	}
	st, ok := got[1].(wire.State)
	if !ok || st.Version != 2 {
		t.Fatalf("second frame: %#v", got[1])
	}
	if st.Ack != ack {
		t.Fatalf("the replaced frame's ack was lost: %+v", st.Ack)
	}
	want := []view.EventKind{view.EventKindBuild, view.EventKindPay, view.EventKindReveal}
	if len(st.Events) != len(want) {
		t.Fatalf("events %v, want kinds %v", st.Events, want)
	}
	for i, k := range want {
		if st.Events[i].Kind != k {
			t.Fatalf("event %d is %s, want %s", i, st.Events[i].Kind, k)
		}
	}
	if o.Take() != nil {
		t.Fatalf("Take after Take returned frames")
	}
}

func TestOutboxKeepsNewerAck(t *testing.T) {
	o := NewOutbox()
	o.Push(outboxState("AAAAAA", 1, &wire.Ack{ClientID: "c", Seq: 1}))
	o.Push(outboxState("AAAAAA", 2, &wire.Ack{ClientID: "c", Seq: 2}))
	got := o.Take()
	if len(got) != 1 || got[0].(wire.State).Ack.Seq != 2 {
		t.Fatalf("got %#v", got)
	}
}

func TestOutboxKeepsOtherRoomsStates(t *testing.T) {
	o := NewOutbox()
	o.Push(outboxState("AAAAAA", 5, nil, view.EventKindBuild))
	o.Push(wire.NewCreated("BBBBBB"))
	o.Push(outboxState("BBBBBB", 0, nil))
	got := o.Take()
	if len(got) != 3 {
		t.Fatalf("got %d frames, want 3", len(got))
	}
	if st := got[2].(wire.State); st.Code != "BBBBBB" || len(st.Events) != 0 {
		t.Fatalf("a state frame of another room inherited events: %#v", st)
	}
}

func TestOutboxMergedEventsAreBounded(t *testing.T) {
	o := NewOutbox()
	for v := range MaxMergedEvents + 10 {
		o.Push(outboxState("AAAAAA", v, nil, view.EventKindPay))
	}
	got := o.Take()
	if len(got) != 1 || len(got[0].(wire.State).Events) != MaxMergedEvents {
		t.Fatalf("got %d frames with %d events", len(got), len(got[0].(wire.State).Events))
	}
}

func TestOutboxOverflowAndClose(t *testing.T) {
	o := NewOutbox()
	for i := range MaxPendingFrames {
		o.Push(wire.NewReject(i, wire.RejectCodeRate, ""))
	}
	if o.Overflowed() {
		t.Fatalf("overflowed at the limit")
	}
	o.Push(wire.NewReject(99, wire.RejectCodeRate, ""))
	if !o.Overflowed() {
		t.Fatalf("not overflowed past the limit")
	}
	if n := len(o.Take()); n != MaxPendingFrames {
		t.Fatalf("Take returned %d frames, want %d", n, MaxPendingFrames)
	}
	o.Close()
	o.Push(wire.NewCreated("AAAAAA"))
	if o.Take() != nil {
		t.Fatalf("Push after Close queued a frame")
	}
}

func TestSeqLRU(t *testing.T) {
	l := newSeqLRU(2)
	l.put("a", 1)
	l.put("b", 2)
	if _, ok := l.get("a"); !ok { // a is now the most recently used
		t.Fatalf("a missing")
	}
	l.put("c", 3) // evicts b
	if _, ok := l.get("b"); ok {
		t.Fatalf("b was not evicted")
	}
	if seq, ok := l.get("a"); !ok || seq != 1 {
		t.Fatalf("a = %d, %v", seq, ok)
	}
	l.put("a", 7)
	snap := l.snapshot()
	if len(snap) != 2 || snap["a"] != 7 || snap["c"] != 3 || l.len() != 2 {
		t.Fatalf("snapshot %v", snap)
	}
}
