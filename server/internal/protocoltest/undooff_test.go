package protocoltest

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// TestUndoOffOverTheWire checks allowUndo end to end: a hot-seat room
// created with it off says so in every frame, never offers undo and
// answers an undo with reject undo-off; an online create that names it is
// answered with reject bad-request and makes no room.
func TestUndoOffOverTheWire(t *testing.T) {
	srv := newServer(t, t.TempDir(), 9_800_000_000_001)
	c := dial(t, srv, "NoUndo")
	c.hello()

	off := false
	req := wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), []string{c.name})
	req.AllowUndo = &off
	c.send(req)
	created, ok := c.next().(wire.Created)
	if !ok {
		t.Fatalf("create with allowUndo false: no created frame")
	}
	c.code = created.Code
	st := c.until("the new room's state", func(f any) bool {
		s, ok := f.(wire.State)
		return ok && s.Code == created.Code
	}).(wire.State)
	if st.Room.AllowUndo || st.CanUndo {
		t.Fatalf("new room: allowUndo %v, canUndo %v", st.Room.AllowUndo, st.CanUndo)
	}
	seq := c.act(st.View.Choices[0].Key, seatInt(*st.View.ToMove))
	st = c.untilAck(seq)
	if st.Room.AllowUndo || st.CanUndo {
		t.Fatalf("after a move: allowUndo %v, canUndo %v", st.Room.AllowUndo, st.CanUndo)
	}
	useq := c.undo()
	if r := c.untilReject(useq); r.Code != wire.RejectCodeUndoOff {
		t.Fatalf("undo in a room with undo off: reject %s, want %s", r.Code, wire.RejectCodeUndoOff)
	}

	// An online room has no undo, so the field is refused there.
	on := true
	online := wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(0), []string{c.name})
	online.AllowUndo = &on
	c.send(online)
	f := c.next()
	r, ok := f.(wire.Reject)
	if !ok || r.Code != wire.RejectCodeBadRequest {
		t.Fatalf("online create with allowUndo: got %#v, want reject bad-request", f)
	}

	// A plain hot-seat room allows undo.
	plain := c.create(wire.ModeHotseat, nil, wire.SeatNumber(0))
	if !plain.Room.AllowUndo {
		t.Fatalf("a hot-seat room created without allowUndo: allowUndo false")
	}
}
