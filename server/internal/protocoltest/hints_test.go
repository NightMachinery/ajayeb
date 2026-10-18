package protocoltest

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/view"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// TestHintSettingsOverTheSocket drives the hint settings (docs/protocol.md,
// the hints frame) through the real transport: a create with settings, the
// joiner seeing them, the admin changing them (a family and the advisor's
// strength) for both players with a log entry, the other player refused,
// and everything still there after a server restart.
func TestHintSettingsOverTheSocket(t *testing.T) {
	dir := t.TempDir()
	srv1 := newServer(t, dir, 9_950_000_000_001)
	a := dial(t, srv1, "Admin")
	a.hello()
	hints := view.HintSettings{Prices: true, Economy: true, Science: true, Military: true, Threats: true,
		Tempo: true, Wonders: true, Tokens: true, Scoring: true, Counting: true, Recap: false, Advisor: true,
		AdvisorStrength: view.AdvisorStrengthDeep}
	req := wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(1), []string{"Admin"})
	req.Hints = &hints
	a.send(req)
	created, ok := a.next().(wire.Created)
	if !ok {
		t.Fatalf("create: no created frame")
	}
	a.code = created.Code
	st := a.until("the new room's state", func(f any) bool {
		st, ok := f.(wire.State)
		return ok && st.Code == created.Code
	}).(wire.State)
	if st.Room.Hints != hints || !st.You.Admin {
		t.Fatalf("the creator's first frame: hints %+v, admin %v", st.Room.Hints, st.You.Admin)
	}

	b := dial(t, srv1, "Joiner")
	b.hello()
	bst := b.join(a.code)
	if bst.Room.Hints != hints || bst.You.Admin || len(bst.RoomLog) != 0 {
		t.Fatalf("the joiner's first frame: hints %+v, admin %v, log %+v", bst.Room.Hints, bst.You.Admin, bst.RoomLog)
	}

	next := hints
	next.Recap, next.Prices, next.AdvisorStrength = true, false, view.AdvisorStrengthQuick
	seq := a.nextSeq()
	a.send(wire.NewHints(a.code, seq, next))
	ast := a.untilAck(seq)
	if ast.Room.Hints != next || len(ast.RoomLog) != 1 {
		t.Fatalf("the admin's answer: hints %+v, log %+v", ast.Room.Hints, ast.RoomLog)
	}
	bst = b.until("the change", func(f any) bool {
		st, ok := f.(wire.State)
		return ok && len(st.RoomLog) == 1
	}).(wire.State)
	e := bst.RoomLog[0]
	if bst.Room.Hints != next || e.Kind != wire.RoomLogHints || e.Seat == nil || *e.Seat != 1 ||
		len(e.On) != 1 || e.On[0] != wire.HintFamilyRecap || len(e.Off) != 1 || e.Off[0] != wire.HintFamilyPrices {
		t.Fatalf("the joiner's frame: hints %+v, log %+v", bst.Room.Hints, bst.RoomLog)
	}

	bseq := b.nextSeq()
	b.send(wire.NewHints(b.code, bseq, hints))
	if r := b.untilReject(bseq); r.Code != wire.RejectCodeNotAdmin {
		t.Fatalf("the joiner's change: %s", r.Code)
	}

	token, clientID, code := a.token, a.clientID, a.code
	srv1.close()
	srv2 := newServer(t, dir, 9_950_100_000_001)
	r := dial(t, srv2, "Admin-resumed")
	r.token, r.clientID = token, clientID
	r.hello()
	rst := r.join(code)
	if rst.Room.Hints != next || !rst.You.Admin || len(rst.RoomLog) != 1 {
		t.Fatalf("after a restart: hints %+v, admin %v, log %+v", rst.Room.Hints, rst.You.Admin, rst.RoomLog)
	}
}
