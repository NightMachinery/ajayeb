package protocoltest

import (
	"math/rand/v2"
	"strconv"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/view"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// The forfeit over the real stack: an act whose key is resign:<seat>, in
// each room mode, with the leak check running on every frame, histories
// included.

// checkForfeitFrame checks a state frame after seat loser forfeited.
func checkForfeitFrame(t *testing.T, who string, st wire.State, loser int) {
	t.Helper()
	r := st.View.Result
	if st.Room.Status != wire.RoomStatusFinished || r == nil || r.Victory != view.VictoryKindForfeit ||
		r.Winner == nil || *r.Winner != 1-loser || st.View.Revealed == nil || st.CanUndo {
		t.Fatalf("%s after seat %d's forfeit: status %s, result %+v, canUndo %v", who, loser, st.Room.Status, r, st.CanUndo)
	}
}

func TestForfeitOnlineOverTheWire(t *testing.T) {
	srv := newServer(t, t.TempDir(), 9_970_000_000_001)
	rng := rand.New(rand.NewPCG(41, 42))
	a := dial(t, srv, "A")
	a.hello()
	a.create(wire.ModeOnline, nil, wire.SeatNumber(0))
	code := a.code
	b := dial(t, srv, "B")
	b.hello()
	b.join(code)
	spec := dial(t, srv, "Spectator")
	spec.hello()
	spec.join(code)
	conns := []*tclient{a, b, spec}
	syncTo(conns, maxVersion(conns))
	bySeat := map[int]*tclient{0: a, 1: b}

	snap := srv.snapshot(code)
	lt := newLeakTracker(t, snap.State.Seed)
	checkAll := func() {
		resyncAll(conns...)
		s := srv.snapshot(code)
		lt.check(spec, "spectator", s.Version, hiddenFor(t, s, nil))
		for seat, c := range bySeat {
			var viewer *engine.Seat
			if !s.State.Over() {
				viewer = seatRef(seat)
			}
			lt.check(c, "seat"+strconv.Itoa(seat), s.Version, hiddenFor(t, s, viewer))
		}
	}
	// Play into Age I.
	for a.latest.View.Age < 1 || a.latest.View.Ply < 12 {
		actor := bySeat[*a.latest.View.ToMove]
		seq := actor.act(pickChoice(rng, actor.latest.View.Choices), nil)
		actor.untilAck(seq)
		syncTo(conns, maxVersion(conns))
	}
	checkAll()

	// A spectator cannot forfeit; a player cannot forfeit the other seat.
	seq := spec.act("resign:0", nil)
	if r := spec.untilReject(seq); r.Code != wire.RejectCodeNotYourTurn {
		t.Fatalf("a spectator's forfeit: %s", r.Code)
	}
	seq = a.act("resign:1", nil)
	if r := a.untilReject(seq); r.Code != wire.RejectCodeNotYourTurn {
		t.Fatalf("seat 0 forfeiting seat 1: %s", r.Code)
	}

	// The seat not to move forfeits.
	waiting := 1 - *a.latest.View.ToMove
	loser := bySeat[waiting]
	seq = loser.act(engine.ResignAction(engine.Seat(waiting)).Key(), nil)
	st := loser.untilAck(seq)
	checkForfeitFrame(t, "the loser", st, waiting)
	syncTo(conns, st.Version)
	for _, c := range conns {
		checkForfeitFrame(t, c.name, c.latest, waiting)
	}
	checkAll()
	for _, c := range conns {
		h := *c.latest.History
		if last := h[len(h)-1]; last.Seat != waiting || last.Events[0].Kind != view.EventKindResign {
			t.Fatalf("%s: the history's last entry %+v", c.name, last)
		}
	}
	checkReplay(t, srv, code)
}

func TestForfeitHotseatOverTheWire(t *testing.T) {
	srv := newServer(t, t.TempDir(), 9_970_000_000_101)
	rng := rand.New(rand.NewPCG(43, 44))
	c := dial(t, srv, "Hotseat")
	c.hello()
	c.create(wire.ModeHotseat, nil, wire.SeatNumber(0))
	code := c.code
	lt := newLeakTracker(t, srv.snapshot(code).State.Seed)
	for range 20 {
		toMove := *c.latest.View.ToMove
		seq := c.act(pickChoice(rng, c.latest.View.Choices), seatInt(toMove))
		c.untilAck(seq)
	}
	toMove := *c.latest.View.ToMove
	// The player who is not to move cannot forfeit on the shared device.
	seq := c.act(engine.ResignAction(engine.Seat(1-toMove)).Key(), nil)
	if r := c.untilReject(seq); r.Code != wire.RejectCodeNotYourTurn {
		t.Fatalf("forfeiting the waiting seat: %s", r.Code)
	}
	seq = c.act(engine.ResignAction(engine.Seat(toMove)).Key(), seatInt(toMove))
	checkForfeitFrame(t, "hotseat", c.untilAck(seq), toMove)
	// Undo cannot take a forfeit back.
	seq = c.undo()
	if r := c.untilReject(seq); r.Code != wire.RejectCodeIllegal {
		t.Fatalf("an undo after the forfeit: %s", r.Code)
	}
	c.resync()
	s := srv.snapshot(code)
	lt.check(c, "hotseat", s.Version, hiddenFor(t, s, nil))
	checkReplay(t, srv, code)
}

func TestForfeitAIOverTheWire(t *testing.T) {
	srv := newServer(t, t.TempDir(), 9_970_000_000_201)
	rng := rand.New(rand.NewPCG(45, 46))
	c := dial(t, srv, "AI-human")
	c.hello()
	bot := wire.BotLevelEasy
	c.create(wire.ModeAI, &bot, wire.SeatNumber(0))
	code := c.code
	lt := newLeakTracker(t, srv.snapshot(code).State.Seed)
	for played := 0; played < 6; {
		if c.latest.You.CanAct {
			seq := c.act(pickChoice(rng, c.latest.View.Choices), nil)
			c.untilAck(seq)
			played++
		} else {
			c.untilState(c.latest.Version + 1)
		}
	}
	// The bot's seat is not the human's to forfeit.
	seq := c.act("resign:1", nil)
	if r := c.untilReject(seq); r.Code != wire.RejectCodeNotYourTurn {
		t.Fatalf("forfeiting the bot's seat: %s", r.Code)
	}
	// The human forfeits, whoever is to move; a stale base is judged afresh.
	for {
		seq = c.act("resign:0", nil)
		f := c.until("the forfeit's answer", func(f any) bool {
			if r, ok := f.(wire.Reject); ok && r.Seq == seq {
				return true
			}
			st, ok := f.(wire.State)
			return ok && st.Ack != nil && st.Ack.Seq == seq
		})
		if r, ok := f.(wire.Reject); ok {
			if r.Code != wire.RejectCodeStale {
				t.Fatalf("the human's forfeit: %s", r.Code)
			}
			c.seq-- // a refusal does not use the seq up
			c.untilState(c.latest.Version)
			continue
		}
		checkForfeitFrame(t, "ai-human", f.(wire.State), 0)
		break
	}
	c.resync()
	s := srv.snapshot(code)
	lt.check(c, "ai-human", s.Version, hiddenFor(t, s, nil))
	checkReplay(t, srv, code)
}
