package tableroom

import (
	"slices"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/room/roomtest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// do runs one seat, role or resignation request from client id on the
// actor and returns its outcome.
func (h *th) do(id tabletest.ClientID, f wire.ClientMessage) string {
	h.t.Helper()
	a, err := h.p.actor()
	if err != nil {
		h.t.Fatal(err)
	}
	var out string
	if err := a.call(h.ctx, func() {
		c := h.p.clients[id]
		switch f := f.(type) {
		case wire.TableSeat:
			out = a.switchSeat(c, f)
		case wire.TableRole:
			out = a.role(c, f)
		case wire.TableResign:
			out = a.resign(c, f)
		}
	}); err != nil {
		h.t.Fatal(err)
	}
	h.p.drain()
	return out
}

func (h *th) want(id tabletest.ClientID, f wire.ClientMessage, want, what string) {
	h.t.Helper()
	if got := h.do(id, f); got != want {
		h.t.Fatalf("%s: %s, want %s", what, got, want)
	}
}

func toBot(seq, seat int, level string) wire.TableSeat {
	return wire.TableSeat{T: "table-seat", Seq: seq, Seat: seat, Control: SeatBot, Bot: &level}
}

func toHuman(seq, seat int, user string) wire.TableSeat {
	return wire.TableSeat{T: "table-seat", Seq: seq, Seat: seat, Control: SeatHuman, UserID: &user}
}

func roleOp(seq int, op, user string) wire.TableRole {
	r := wire.TableRole{T: "table-role", Seq: seq, Op: op}
	if user != "" {
		r.UserID = &user
	}
	return r
}

// human returns the frame's metadata for user, or fails.
func human(t *testing.T, fr wire.TableState, user string) wire.PlatformHuman {
	t.Helper()
	i := slices.IndexFunc(fr.Humans, func(x wire.PlatformHuman) bool { return x.UserID == user })
	if i < 0 {
		t.Fatalf("%s is not in the humans: %+v", user, fr.Humans)
	}
	return fr.Humans[i]
}

// sameLogEverywhere: every connection's latest frame carries the same
// room log, of want entries, and no connection's revision ever went back.
func (h *th) sameLogEverywhere(want int) []wire.PlatformLogEntry {
	h.t.Helper()
	h.p.drain()
	var log []wire.PlatformLogEntry
	for id, frames := range h.p.frames {
		if h.p.clients[id] == nil || len(frames) == 0 {
			continue
		}
		for i := 1; i < len(frames); i++ {
			if frames[i].Revision < frames[i-1].Revision {
				h.t.Fatalf("%s's revision went from %d to %d", id, frames[i-1].Revision, frames[i].Revision)
			}
		}
		got := frames[len(frames)-1].RoomLog
		if log == nil {
			log = got
		}
		if len(got) != want || !slices.EqualFunc(got, log, sameEntry) {
			h.t.Fatalf("%s's room log: %d entries, want %d: %+v", id, len(got), want, got)
		}
	}
	return log
}

func sameEntry(x, y wire.PlatformLogEntry) bool {
	eq := func(p, q *string) bool { return (p == nil) == (q == nil) && (p == nil || *p == *q) }
	return x.Kind == y.Kind && x.At == y.At && x.By == y.By && eq(x.UserID, y.UserID) && eq(x.Control, y.Control) &&
		eq(x.Role, y.Role) && (x.Seat == nil) == (y.Seat == nil) && (x.Seat == nil || *x.Seat == *y.Seat)
}

// leave detaches the clients, waiting until the actor has seen it.
func (h *th) leave(ids ...tabletest.ClientID) {
	h.t.Helper()
	for _, id := range ids {
		h.f.m.Leave(h.p.clients[id], h.p.code)
		delete(h.p.clients, id)
	}
	a, err := h.p.actor()
	if err == nil {
		err = a.call(h.ctx, func() {})
	}
	if err != nil {
		h.t.Fatal(err)
	}
	h.p.drain()
}

func (h *th) botBusy() bool {
	h.t.Helper()
	a, err := h.p.actor()
	if err != nil {
		h.t.Fatal(err)
	}
	var busy bool
	if err := a.call(h.ctx, func() { busy = a.botBusy }); err != nil {
		h.t.Fatal(err)
	}
	return busy
}

// TestSeatSwitches: an admin switches seats before and after a pick and
// its confirmation. A bot taking a seat confirms its pending pick, or gets
// a work row and searches; a spectator seated in a bot's or a human's seat
// takes the pick over, still changeable, and the human replaced watches.
// Only admins see the previous-seat hint, every switch is in every
// viewer's log, no viewer's revision goes back, and a restart keeps it all.
func TestSeatSwitches(t *testing.T) {
	h := newTH(t, Config{BotWork: blockBots}, humans(4, 41))
	admin, spec := tabletest.SeatClient(0), tabletest.SpectatorClient
	c1, c2, c3 := tabletest.SeatClient(1), tabletest.SeatClient(2), tabletest.SeatClient(3)
	h.want(c1, toBot(1, 2, BotEasy), outNotAdmin, "a non-admin's switch")
	h.want(admin, toHuman(1, 2, "nobody"), outBad, "seating a user who is not here")
	h.want(admin, toHuman(2, 2, userOf(1)), outBad, "seating a seated user")

	// Seat 2 drafts, then goes to a bot, which confirms the draft.
	s2 := h.snap(c2)
	h.expect(h.send(c2, 1, s2.Ref, s2.Legal[0], false), tabletest.Accepted, "seat 2's draft")
	h.want(admin, toBot(3, 2, BotNormal), outAccepted, "seat 2 to a bot")
	fr := h.latest(spec)
	if fr.Seats[2].Kind != SeatBot || *fr.Seats[2].Bot != BotNormal || !slices.Contains(fr.Point.Confirmed, 2) {
		t.Fatalf("after seat 2 went to a bot: %+v, confirmed %v", fr.Seats[2], fr.Point.Confirmed)
	}
	if fr2 := h.latest(c2); fr2.You.Seat != nil || fr2.Pending != nil || fr2.View.You != nil {
		t.Fatalf("the replaced human still sees seat 2: %+v", fr2.You)
	}
	h.expect(h.send(c2, 2, s2.Ref, s2.Legal[0], true), tabletest.Forbidden, "the replaced human's pick")

	// Seat 3 has no pick: its bot gets a work row and searches.
	h.want(admin, toBot(4, 3, BotEasy), outAccepted, "seat 3 to a bot")
	r := h.record()
	if i := slices.IndexFunc(r.Work, func(w WorkRow) bool { return w.Seat == 3 }); i < 0 || r.Work[i].Level != BotEasy {
		t.Fatalf("seat 3's work rows: %+v", r.Work)
	}
	if !h.botBusy() {
		t.Fatal("seat 3's bot is not searching")
	}
	// The spectator takes seat 3 from the bot, before the bot picked.
	h.want(admin, toHuman(5, 3, "watcher"), outAccepted, "the spectator into seat 3")
	if r := h.record(); slices.ContainsFunc(r.Work, func(w WorkRow) bool { return w.Seat == 3 }) || h.botBusy() {
		t.Fatalf("seat 3 kept its bot: %+v", r.Work)
	}
	ss := h.snap(spec)
	if ss.Seat != 3 || len(ss.Legal) == 0 {
		t.Fatalf("the spectator's view of seat 3: %+v", ss)
	}

	// Seat 1 confirms; user-2, now a spectator, takes seat 1 over and may
	// still withdraw its confirmation. user-1 watches.
	s1 := h.snap(c1)
	h.expect(h.send(c1, 1, s1.Ref, s1.Legal[1], true), tabletest.Accepted, "seat 1's confirmation")
	h.want(admin, toHuman(6, 1, userOf(2)), outAccepted, "user-2 into seat 1")
	if fr := h.latest(c2); fr.You.Seat == nil || *fr.You.Seat != 1 || fr.Pending == nil || fr.Pending.Key != s1.Legal[1] || !fr.Pending.Confirm {
		t.Fatalf("user-2 in seat 1: %+v", fr.Pending)
	}
	h.expect(h.send(c2, 3, s1.Ref, s1.Legal[1], false), tabletest.Accepted, "user-2 withdraws seat 1's confirmation")
	// A private edit by the new controller changes nobody else's frame.
	others := []tabletest.ClientID{admin, c1, c3, spec}
	counts := make([]int, len(others))
	for i, id := range others {
		counts[i] = h.frameCount(id)
	}
	h.expect(h.send(c2, 4, s1.Ref, s1.Legal[0], false), tabletest.Accepted, "user-2 changes seat 1's draft")
	for i, id := range others {
		if n := h.frameCount(id); n != counts[i] {
			t.Fatalf("%s got %d frames from user-2's private edit", id, n-counts[i])
		}
	}

	// The previous-seat hint: user-3 and user-1 held seats 3 and 1 and
	// watch now; user-2 holds a seat again, so it has none. Admins only.
	hint := []wire.PreviousSeat{{UserID: userOf(3), Seat: 3}, {UserID: userOf(1), Seat: 1}}
	if ps := h.latest(admin).You.PreviousSeats; ps == nil || !slices.Equal(*ps, hint) {
		t.Fatalf("the admin's previous-seat hint: %v", ps)
	}
	for _, id := range []tabletest.ClientID{c1, c2, spec} {
		if ps := h.latest(id).You.PreviousSeats; ps != nil {
			t.Fatalf("%s got the previous-seat hint %v", id, *ps)
		}
	}
	log := h.sameLogEverywhere(4)
	if e := log[3]; e.Kind != "seat" || e.By != userOf(0) || *e.Seat != 1 || *e.Control != SeatHuman || *e.UserID != userOf(2) || e.At != 0 {
		t.Fatalf("the last entry: %+v", e)
	}

	// A restart keeps the seats, the pick, the hint and the log.
	before := h.record()
	h.f.restart()
	h.rejoin()
	after := h.record()
	if !slices.Equal(after.Seats, before.Seats) || !slices.Equal(after.Previous, before.Previous) || len(after.RoomLog) != 4 {
		t.Fatalf("across a restart: %+v, then %+v", before.Seats, after.Seats)
	}
	if sn := h.snap(c2); sn.Seat != 1 || sn.Pick != s1.Legal[0] || sn.Confirm {
		t.Fatalf("seat 1 after the restart: %+v", sn)
	}
	h.sameLogEverywhere(4)
}

// TestUndoOffWhenASecondHumanSits: a sole-human table loses undo for good
// once a second human takes a seat.
func TestUndoOffWhenASecondHumanSits(t *testing.T) {
	h := newTH(t, Config{BotWork: blockBots}, soloSpec(3, 7, BotEasy))
	me := tabletest.SeatClient(0)
	if fr := h.latest(me); !fr.AllowUndo || fr.UndoOff || fr.StartingHumanCount != 1 {
		t.Fatalf("the start: %+v", fr)
	}
	h.want(me, toHuman(1, 1, "watcher"), outAccepted, "the spectator into a bot's seat")
	h.want(me, toBot(2, 1, BotHard), outAccepted, "back to a bot")
	if fr := h.latest(me); !fr.UndoOff || !fr.AllowUndo {
		t.Fatalf("undo after a second human sat: off %t", fr.UndoOff)
	}
	if out := h.undo(3, h.latest(me).Revision); out != outForbidden {
		t.Fatalf("undo: %s", out)
	}
}

// TestRolesAndTemporaryAdmin: promotion, demotion and their limits, then
// the temporary admin under an injected clock: made once every real admin
// has been away five minutes, kept across a restart, passed on after its
// own five minutes away, and demoted when a real admin returns. Every
// change is in every viewer's log.
func TestRolesAndTemporaryAdmin(t *testing.T) {
	t0 := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	clk := roomtest.NewFakeClock(t0)
	h := newTH(t, Config{Clock: clk, BotWork: blockBots}, humans(3, 43))
	admin, c1, c2, spec := tabletest.SeatClient(0), tabletest.SeatClient(1), tabletest.SeatClient(2), tabletest.SpectatorClient
	h.want(c1, roleOp(1, "promote", userOf(2)), outNotAdmin, "a non-admin's promotion")
	h.want(admin, roleOp(1, "promote", userOf(1)), outAccepted, "user-1's promotion")
	h.want(c1, roleOp(2, "promote", "watcher"), outAccepted, "the spectator's promotion by user-1")
	if hm := human(t, h.latest(c2), "watcher"); !hm.Admin || hm.PromotedBy == nil || *hm.PromotedBy != userOf(1) {
		t.Fatalf("the spectator after promotion: %+v", hm)
	}
	h.want(c1, roleOp(3, "demote", userOf(0)), outForbidden, "demoting the creator")
	h.want(spec, roleOp(1, "demote", userOf(1)), outNotAdmin, "demoting one's promoter")
	h.want(admin, roleOp(2, "demote", userOf(1)), outAccepted, "the creator demotes user-1")
	if hm := human(t, h.latest(c2), "watcher"); !hm.Admin {
		t.Fatal("demoting user-1 demoted the admin it promoted")
	}
	h.want(c1, toBot(4, 2, BotEasy), outNotAdmin, "a demoted admin's switch")

	// Both real admins leave; seats 1 and 2 stay. Five minutes after the
	// last one left, the human holding a seat longest (seat order breaks
	// the tie) becomes the temporary admin.
	h.leave(admin, spec)
	if r := h.record(); len(r.Roles.Away) != 2 || !r.Roles.Away[0].Since.Equal(t0) {
		t.Fatalf("away times: %+v", r.Roles.Away)
	}
	clk.Advance(5*time.Minute - time.Second)
	if fr := h.latest(c1); human(t, fr, userOf(1)).Temporary {
		t.Fatal("a temporary admin before five minutes")
	}
	clk.Advance(time.Second)
	h.until(c2, "user-1 as the temporary admin", func(fr wire.TableState) bool { return human(t, fr, userOf(1)).Temporary })
	h.want(c1, roleOp(5, "promote", userOf(2)), outNotAdmin, "a temporary admin's promotion")

	// A restart keeps the temporary admin, the promoters and the away times.
	h.f.restart()
	h.rejoin()
	r := h.record()
	if r.Roles.Temporary != userOf(1) || len(r.Roles.Away) != 2 || !r.Roles.Away[1].Since.Equal(t0) {
		t.Fatalf("roles across the restart: %+v", r.Roles)
	}
	promoted := false
	for _, a := range r.Roles.Admins {
		promoted = promoted || a.UserID == "watcher" && a.PromotedBy != nil && *a.PromotedBy == userOf(1)
	}
	if !promoted {
		t.Fatalf("the spectator's promoter across the restart: %+v", r.Roles.Admins)
	}
	h.want(c1, toBot(6, 2, BotEasy), outAccepted, "the temporary admin's switch")

	// The temporary admin leaves: five minutes on, the role passes to the
	// spectator connected longest (user-2, whose seat went to a bot).
	h.leave(c1)
	clk.Advance(5 * time.Minute)
	h.until(c2, "user-2 as the temporary admin", func(fr wire.TableState) bool {
		return human(t, fr, userOf(2)).Temporary && !slices.ContainsFunc(fr.Humans, func(x wire.PlatformHuman) bool { return x.Temporary && x.UserID != userOf(2) })
	})

	// A real admin's return demotes the temporary admin at once.
	if err := h.p.join(h.ctx, admin, userOf(0), "P0"); err != nil {
		t.Fatal(err)
	}
	if hm := human(t, h.latest(c2), userOf(2)); hm.Temporary {
		t.Fatal("the temporary admin stayed after a real admin returned")
	}
	log := h.sameLogEverywhere(8)
	var roles []string
	for _, e := range log {
		if e.Kind == "role" {
			roles = append(roles, *e.UserID+"="+*e.Role)
		}
	}
	want := []string{"user-1=admin", "watcher=admin", "user-1=human", "user-1=temporary", "user-1=human", "user-2=temporary", "user-2=human"}
	if !slices.Equal(roles, want) {
		t.Fatalf("role log %v, want %v", roles, want)
	}
}

// TestResignation: a seat resigns to a bot at the lowest level, which
// confirms its pending pick; nobody wins by it. An original participant
// watching keeps the bots running; once nobody who played is attached,
// they pause.
func TestResignation(t *testing.T) {
	h := newTH(t, Config{BotWork: blockBots}, humans(3, 47))
	c1, c2 := tabletest.SeatClient(1), tabletest.SeatClient(2)
	s1 := h.snap(c1)
	h.expect(h.send(c1, 1, s1.Ref, s1.Legal[0], false), tabletest.Accepted, "seat 1's draft")
	h.want(c1, wire.TableResign{T: "table-resign", Seq: 2, Confirm: false}, outBad, "an unconfirmed resignation")
	h.want(c1, wire.TableResign{T: "table-resign", Seq: 3, Confirm: true}, outAccepted, "seat 1 resigns")
	fr := h.latest(c2)
	if fr.Status != "playing" || fr.Seats[1].Kind != SeatBot || *fr.Seats[1].Bot != DefaultBotLevel || !slices.Contains(fr.Point.Confirmed, 1) {
		t.Fatalf("after the resignation: %+v, confirmed %v", fr.Seats[1], fr.Point.Confirmed)
	}
	h.want(c1, wire.TableResign{T: "table-resign", Seq: 3, Confirm: true}, outDuplicate, "the resignation again")
	h.want(c1, wire.TableResign{T: "table-resign", Seq: 4, Confirm: true}, outForbidden, "a spectator's resignation")
	if e := h.sameLogEverywhere(1)[0]; e.Kind != "resign" || e.By != userOf(1) || *e.Seat != 1 || *e.Control != SeatBot {
		t.Fatalf("the log entry: %+v", e)
	}
	// Seat 2 resigns with no pick: its bot searches while user-2, an
	// original participant, watches alone, and pauses once nobody who
	// played is attached.
	h.want(c2, wire.TableResign{T: "table-resign", Seq: 1, Confirm: true}, outAccepted, "seat 2 resigns")
	h.leave(tabletest.SeatClient(0), c1, tabletest.SpectatorClient)
	if !h.botBusy() {
		t.Fatal("no bot search with an original participant watching")
	}
	h.leave(c2)
	if err := h.p.join(h.ctx, tabletest.SpectatorClient, "watcher", "Watcher"); err != nil {
		t.Fatal(err)
	}
	if h.botBusy() {
		t.Fatal("a bot searches with only an unrelated spectator attached")
	}
}
