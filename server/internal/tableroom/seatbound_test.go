package tableroom

import (
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/faults"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// on runs f on the actor's goroutine and drains the frames it pushed.
func (h *th) on(f func(a *actor)) {
	h.t.Helper()
	a, err := h.p.actor()
	if err != nil {
		h.t.Fatal(err)
	}
	if err := a.call(h.ctx, func() { f(a) }); err != nil {
		h.t.Fatal(err)
	}
	h.p.drain()
}

func seatRef(s int) *int { return &s }

// TestRequestsNameTheirSeat: a pick or a resignation may name the seat it
// was made for. One sent before an admin moved its user to another seat,
// and arriving after, is refused with forbidden and changes nothing; before,
// it acted on the user's new seat (M16.haftgane-security-review). Naming
// the current seat, or no seat, is judged as before, and a refusal uses up
// no seq.
func TestRequestsNameTheirSeat(t *testing.T) {
	h := newTH(t, Config{BotWork: blockBots}, humans(4, 104))
	admin, c1 := tabletest.SeatClient(0), tabletest.SeatClient(1)
	s1, s3 := h.snap(c1), h.snap(tabletest.SeatClient(3))
	key := ""
	for _, k := range s1.Legal {
		if slices.Contains(s3.Legal, k) {
			key = k
			break
		}
	}
	if key == "" {
		t.Fatal("seats 1 and 3 share no key at this seed, so a pick could not land on the wrong seat")
	}
	// user-1 moves from seat 1 to seat 3 while requests for seat 1 are on their way.
	h.want(admin, toBot(1, 1, BotEasy), outAccepted, "seat 1 to a bot")
	h.want(admin, toHuman(2, 3, userOf(1)), outAccepted, "user-1 into seat 3")
	pick := func(seq int, seat *int) string {
		var out string
		h.on(func(a *actor) {
			out = a.submit(h.p.clients[c1], wire.TablePick{T: "table-pick", Code: h.p.code, Seq: seq, Point: parseRef(s1.Ref), Key: key, Confirm: true, Seat: seat})
		})
		return out
	}
	before := h.record()
	if out := pick(1, seatRef(1)); out != outForbidden {
		t.Fatalf("a pick for seat 1 arriving after the move: %s", out)
	}
	if out := h.do(c1, wire.TableResign{T: "table-resign", Seq: 1, Confirm: true, Seat: seatRef(1)}); out != outForbidden {
		t.Fatalf("a resignation of seat 1 arriving after the move: %s", out)
	}
	after := h.record()
	if len(after.Pending) != len(before.Pending) || after.Seats[3].Kind != SeatHuman || after.Seats[3].UserID != userOf(1) ||
		after.Public != before.Public || len(after.Seqs) != len(before.Seqs) {
		t.Fatalf("a refused request changed the table: %+v -> %+v", before.Seats, after.Seats)
	}
	if rj := h.p.rejects[c1]; len(rj) != 2 || rj[0].Code != "forbidden" || rj[1].Code != "forbidden" {
		t.Fatalf("the sender's rejects: %+v", rj)
	}
	if out := pick(1, seatRef(3)); out != outAccepted {
		t.Fatalf("the pick naming seat 3: %s", out)
	}
	if out := h.do(c1, wire.TableResign{T: "table-resign", Seq: 2, Confirm: true}); out != outAccepted || h.record().Seats[3].Kind != SeatBot {
		t.Fatalf("a resignation naming no seat: %s", out)
	}
}

// TestUndoNamesItsSeat: an undo naming another seat than the sender's is
// refused with forbidden; naming the sender's own is judged as before.
func TestUndoNamesItsSeat(t *testing.T) {
	work := faults.NewWork(true)
	h := newTH(t, Config{BotWork: work.Run}, soloSpec(3, 23, BotEasy))
	me := tabletest.SeatClient(0)
	fr := h.until(me, "the first point", func(st wire.TableState) bool { return st.Point != nil })
	h.pickSelf(1, fr.View.You.Choices[0].Key)
	work.Release()
	work.Release()
	fr = h.until(me, "the first batch", func(st wire.TableState) bool { return st.Batch == 1 })
	undo := func(seq int, seat *int) string {
		var out string
		h.on(func(a *actor) {
			out = a.undo(h.p.clients[me], wire.TableUndo{T: "table-undo", Code: h.p.code, Seq: seq, Base: fr.Revision, Seat: seat})
		})
		return out
	}
	if out := undo(2, seatRef(1)); out != outForbidden {
		t.Fatalf("an undo naming seat 1: %s", out)
	}
	if out := undo(2, seatRef(0)); out != outAccepted {
		t.Fatalf("an undo naming the sender's seat: %s", out)
	}
}
