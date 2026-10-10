package room_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// These tests pin the room package's defences against a client that tries
// to exhaust the server: the cap on rooms in memory, the bot pool, the seq
// table and the outbox.

// abuseHotseat is the create frame of a plain hot-seat room.
func abuseHotseat() wire.Create {
	return wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil)
}

// TestForbiddenJoinLoadsNothing checks that a join the room would refuse
// (another user's hot-seat room) is refused from the file's header, without
// loading the room: otherwise anyone who knows a few hundred codes could
// fill the rooms in memory with joins that are all refused.
func TestForbiddenJoinLoadsNothing(t *testing.T) {
	const maxRooms = 3
	h := newHarness(t, func(c *room.Config) { c.MaxRooms = maxRooms })
	owner := h.client("user-a", "tab-a", "Ann")
	codes := make([]string, 0, maxRooms)
	for range maxRooms {
		code, _ := owner.create(abuseHotseat())
		codes = append(codes, code)
	}
	h.m.Disconnect(owner.Client)
	for _, code := range codes {
		h.snapshot(code) // every detach is handled and every countdown armed
	}
	h.clock.Advance(room.DefaultIdleTimeout)
	h.waitFor("every room to be evicted", func() bool { return h.m.Len() == 0 })

	stranger := h.client("user-x", "tab-x", "Xe")
	for _, code := range codes {
		if e, err := stranger.joinErr(code); e.Code != wire.ErrorCodeForbidden || !errors.Is(err, room.ErrForbidden) {
			t.Fatalf("a stranger's join of %s: %v, %v", code, e, err)
		}
		if h.m.Loaded(code) {
			t.Fatalf("a refused join loaded room %s into memory", code)
		}
	}
	victim := h.client("user-v", "tab-v", "Vi")
	victim.create(abuseHotseat())

	// The owner still gets the room back.
	if st := owner.join(codes[0]); len(st.You.Seats) != 2 {
		t.Fatalf("the owner's rejoin: you %+v", st.You)
	}
}

// TestFullManagerEvictsIdleRooms checks that rooms nobody is attached to
// never lock anyone out: when the Manager holds MaxRooms rooms, a create or
// a load evicts the room that has been idle longest (it is on disk and comes
// back on the next join) instead of answering busy. Only a server whose
// rooms all have connections is busy.
func TestFullManagerEvictsIdleRooms(t *testing.T) {
	h := newHarness(t, func(c *room.Config) { c.MaxRooms = 3 })
	a := h.client("user-a", "tab-a", "Ann")
	first, _ := a.create(abuseHotseat())
	h.clock.Advance(time.Second)
	second, _ := a.create(abuseHotseat()) // a leaves first, which goes idle
	h.clock.Advance(time.Second)
	third, _ := a.create(abuseHotseat()) // and then second
	h.clock.Advance(time.Second)
	h.m.Leave(a.Client, third)
	for _, code := range []string{first, second, third} {
		h.snapshot(code)
	}

	// Full: the next create evicts the room idle longest, first.
	b := h.client("user-b", "tab-b", "Bob")
	fourth, _ := b.create(abuseHotseat())
	if h.m.Loaded(first) || !h.m.Loaded(second) || !h.m.Loaded(third) || !h.m.Loaded(fourth) {
		t.Fatalf("after the fourth create: loaded %v %v %v %v, want first evicted",
			h.m.Loaded(first), h.m.Loaded(second), h.m.Loaded(third), h.m.Loaded(fourth))
	}
	// The evicted room comes back on its owner's join, evicting second.
	if st := a.join(first); len(st.You.Seats) != 2 || st.Version != 0 {
		t.Fatalf("the evicted room after a reload: version %d, you %+v", st.Version, st.You)
	}
	if h.m.Loaded(second) || !h.m.Loaded(third) {
		t.Fatalf("after the reload: second loaded %v, third loaded %v", h.m.Loaded(second), h.m.Loaded(third))
	}

	// Every room in memory now has a connection but third: one more create
	// fits, and the next one is busy.
	c := h.client("user-c", "tab-c", "Cy")
	c.create(abuseHotseat())
	if h.m.Loaded(third) {
		t.Fatalf("the idle room third was not evicted")
	}
	d := h.client("user-d", "tab-d", "Di")
	if _, err := h.m.Create(context.Background(), d.Client, abuseHotseat()); !errors.Is(err, room.ErrTooManyRooms) {
		t.Fatalf("a create when every room has a connection: %v", err)
	}
	if f, ok := d.next().(wire.Error); !ok || f.Code != wire.ErrorCodeBusy {
		t.Fatalf("a create when every room has a connection sent %#v", f)
	}
	if h.m.Len() != 3 {
		t.Fatalf("%d rooms in memory, want 3", h.m.Len())
	}
}

// TestRematchAsksBeforeCreating checks the creation hook of a rematch: the
// room asks the requester's AllowCreate before it creates the rematch room,
// a refusal is answered with reject rate and creates nothing, and a rematch
// that already exists is handed out without asking.
func TestRematchAsksBeforeCreating(t *testing.T) {
	h := newHarness(t)
	code, err := h.m.CreateFromSpec(room.Spec{
		Mode: store.ModeOnline,
		Seats: [2]store.Seat{
			{Kind: store.SeatKindHuman, UserID: "user-a", Name: "Ann"},
			{Kind: store.SeatKindHuman, UserID: "user-b", Name: "Bob"},
		},
		Seed:     3,
		Scenario: "military-supremacy",
		Actions:  mustScenarioTrigger(t, "military-supremacy"),
	})
	if err != nil {
		t.Fatalf("CreateFromSpec: %v", err)
	}
	asked := 0
	a := h.client("user-a", "tab-a", "Ann")
	a.AllowCreate = func() bool { asked++; return false }
	b := h.client("user-b", "tab-b", "Bob")
	b.AllowCreate = func() bool { return true }
	a.join(code)
	b.join(code)

	if _, err := h.m.Rematch(context.Background(), a.Client, code); !errors.Is(err, room.ErrRateLimited) {
		t.Fatalf("a refused rematch: %v", err)
	}
	a.expectReject(0, wire.RejectCodeRate)
	if snap := h.snapshot(code); snap.Rematch != "" || asked != 1 {
		t.Fatalf("after a refused rematch: rematch %q, asked %d times", snap.Rematch, asked)
	}

	code2, err := h.m.Rematch(context.Background(), b.Client, code)
	if err != nil {
		t.Fatalf("an allowed rematch: %v", err)
	}
	again, err := h.m.Rematch(context.Background(), a.Client, code)
	if err != nil || again != code2 || asked != 1 {
		t.Fatalf("the existing rematch for a refused user: %q, %v, asked %d times", again, err, asked)
	}
}

// TestSeqsAreScopedByUser checks that the seq table is keyed by user as
// well as by client id: another user who picks the same client id cannot
// make the room take a player's new requests for old ones.
func TestSeqsAreScopedByUser(t *testing.T) {
	h := newHarness(t)
	a := h.client("user-a", "tab", "Ann")
	code, _ := a.create(wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(0), nil))
	b := h.client("user-b", "tab", "Bob")
	b.join(code)
	h.snapshot(code)
	st := a.current()
	mover, other := a, b
	if *st.View.ToMove == 1 {
		mover, other = b, a
	}
	st = mover.current()
	mover.actSeq(1_000_000, st.View.Choices[0].Key, 0, nil)
	st = other.waitState(func(s wire.State) bool { return s.Version == 1 })

	other.actSeq(1, st.View.Choices[0].Key, 1, nil)
	if snap := h.snapshot(code); snap.Version != 2 {
		t.Fatalf("the other user's act with the same client id was not applied: version %d", snap.Version)
	}
	hdr, err := h.st.PeekHeader(code)
	if err != nil {
		t.Fatalf("PeekHeader: %v", err)
	}
	if len(hdr.LastSeq) != 2 || hdr.LastSeq[mover.UserID+":tab"] != 1_000_000 || hdr.LastSeq[other.UserID+":tab"] != 1 {
		t.Fatalf("stored seqs %v", hdr.LastSeq)
	}
}

// TestBotSearchesOnlyWhileAttached checks that a bot thinks only while its
// human is connected: a room nobody is attached to runs no search (so rooms
// left behind cannot fill the bot pool), the last detach cancels a running
// one, and the next attach starts it again.
func TestBotSearchesOnlyWhileAttached(t *testing.T) {
	gate := newGateBot()
	h := newHarness(t, func(c *room.Config) {
		c.NewBot = func(store.BotLevel, uint64) room.Bot { return gate }
	})
	code, err := h.m.CreateFromSpec(room.Spec{
		Mode:    store.ModeAI,
		Seats:   [2]store.Seat{{Kind: store.SeatKindHuman, UserID: "user-a", Name: "Ann"}, {Kind: store.SeatKindBot, Bot: store.BotHard}},
		Seed:    7,
		Options: engine.Options{FirstPlayer: engine.FirstPlayerSeat1},
	})
	if err != nil {
		t.Fatalf("CreateFromSpec: %v", err)
	}
	if snap := h.snapshot(code); snap.Status != store.StatusPlaying || snap.BotThinking {
		t.Fatalf("with nobody attached: status %s, thinking %v", snap.Status, snap.BotThinking)
	}

	a := h.client("user-a", "tab-a", "Ann")
	if st := a.join(code); st.Thinking == nil || *st.Thinking != 1 {
		t.Fatalf("the first state after the join does not show the bot thinking: %v", st.Thinking)
	}
	gate.waitStarted(t)
	h.m.Disconnect(a.Client)
	h.waitFor("the search to be cancelled and its result dropped", func() bool {
		snap := h.snapshot(code)
		return !snap.BotThinking && snap.DroppedBotResults == 1
	})
	if snap := h.snapshot(code); snap.Version != 0 {
		t.Fatalf("the bot moved for a room nobody is attached to: version %d", snap.Version)
	}

	a2 := h.client("user-a", "tab-a2", "Ann")
	a2.join(code)
	gate.waitStarted(t)
	gate.release <- struct{}{}
	a2.waitState(func(s wire.State) bool { return s.Version == 1 })
}

// lateBot is a bot whose searches all run until their context is
// cancelled and then fail. The first one also waits for late before it
// returns, so that it can answer after a later search has started.
type lateBot struct {
	started chan int // the number of each search, as it starts
	late    chan struct{}
	calls   atomic.Int32
}

func (b *lateBot) Choose(ctx context.Context, _ engine.Observation, _ ai.Budget, _ func(ai.Progress)) (engine.Action, error) {
	n := int(b.calls.Add(1))
	b.started <- n
	<-ctx.Done()
	if n == 1 {
		<-b.late
	}
	return engine.Action{}, ctx.Err()
}

// TestCancelledSearchStaysDropped checks that a search cancelled by the last
// detach can never be applied, even when its human comes back and a new
// search for the same version starts before the cancelled one answers.
// Otherwise a client could leave and rejoin its own bot room on purpose to
// make the bot play a truncated search, or the first legal action.
func TestCancelledSearchStaysDropped(t *testing.T) {
	bot := &lateBot{started: make(chan int, 4), late: make(chan struct{})}
	release := sync.OnceFunc(func() { close(bot.late) })
	defer release() // never leave the first search waiting, even on a failure
	h := newHarness(t, func(c *room.Config) {
		c.NewBot = func(store.BotLevel, uint64) room.Bot { return bot }
	})
	code, err := h.m.CreateFromSpec(room.Spec{
		Mode:    store.ModeAI,
		Seats:   [2]store.Seat{{Kind: store.SeatKindHuman, UserID: "user-a", Name: "Ann"}, {Kind: store.SeatKindBot, Bot: store.BotHard}},
		Seed:    7,
		Options: engine.Options{FirstPlayer: engine.FirstPlayerSeat1},
	})
	if err != nil {
		t.Fatalf("CreateFromSpec: %v", err)
	}
	waitSearch := func(want int) {
		t.Helper()
		select {
		case n := <-bot.started:
			if n != want {
				t.Fatalf("search %d started, want %d", n, want)
			}
		case <-time.After(waitLimit):
			t.Fatalf("search %d never started", want)
		}
	}

	a := h.client("user-a", "tab-a", "Ann")
	a.join(code)
	waitSearch(1)
	h.m.Disconnect(a.Client)
	h.waitFor("the detach to cancel the search", func() bool { return !h.snapshot(code).BotThinking })

	a2 := h.client("user-a", "tab-a2", "Ann")
	a2.join(code)
	waitSearch(2)
	release()
	h.waitFor("the cancelled search to answer", func() bool {
		snap := h.snapshot(code)
		return snap.DroppedBotResults == 1 || snap.Version != 0
	})
	if snap := h.snapshot(code); snap.Version != 0 || !snap.BotThinking {
		t.Fatalf("the cancelled search's result was applied: version %d, thinking %v", snap.Version, snap.BotThinking)
	}
}

// TestNewRoomIsNotEvictedFirst checks the order of eviction for a room that
// nobody has joined yet: it has been idle since it was made, not forever,
// so an older idle room goes first.
func TestNewRoomIsNotEvictedFirst(t *testing.T) {
	h := newHarness(t, func(c *room.Config) { c.MaxRooms = 2 })
	open := store.Seat{Kind: store.SeatKindOpen}
	spec := room.Spec{Mode: store.ModeOnline, Seats: [2]store.Seat{open, open}, Seed: 5}
	older, err := h.m.CreateFromSpec(spec)
	if err != nil {
		t.Fatalf("CreateFromSpec: %v", err)
	}
	a := h.client("user-a", "tab-a", "Ann")
	a.join(older)
	h.m.Leave(a.Client, older)
	h.snapshot(older) // idle since now
	h.clock.Advance(time.Second)
	newer, err := h.m.CreateFromSpec(spec)
	if err != nil {
		t.Fatalf("CreateFromSpec: %v", err)
	}
	if _, err := h.m.CreateFromSpec(spec); err != nil {
		t.Fatalf("CreateFromSpec on a full Manager: %v", err)
	}
	if h.m.Loaded(older) || !h.m.Loaded(newer) {
		t.Fatalf("older loaded %v, newer loaded %v: want the older idle room evicted", h.m.Loaded(older), h.m.Loaded(newer))
	}
}
