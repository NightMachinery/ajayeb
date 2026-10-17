package room_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"sync"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/tutorial"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// Tutorial rooms (tutorial.go, package tutorial): an ai room on the
// script's seed and options, the player in seat 0 against the Easy bot. The
// player's frames carry the tutorial object derived from the log, the bot
// plays the script's replies while the log is on it, a move the script
// asked for gets no move review, and the rematch is a normal game.

// botCalls records the levels of the bots the room asks Config.NewBot for.
type botCalls struct {
	mu     sync.Mutex
	levels []store.BotLevel
}

func (b *botCalls) get() []store.BotLevel {
	b.mu.Lock()
	defer b.mu.Unlock()
	return slices.Clone(b.levels)
}

// newTutorialHarness is newHarness whose bots, still playing the first
// legal action, are counted.
func newTutorialHarness(t *testing.T) (*harness, *botCalls) {
	calls := &botCalls{}
	h := newHarness(t, func(cfg *room.Config) {
		cfg.NewBot = func(level store.BotLevel, _ uint64) room.Bot {
			calls.mu.Lock()
			calls.levels = append(calls.levels, level)
			calls.mu.Unlock()
			return firstLegalBot{}
		}
	})
	return h, calls
}

// tutorialCreate is the create frame of the client's tutorial entry.
func tutorialCreate() wire.Create {
	bot := wire.BotLevelEasy
	c := wire.NewCreate(wire.ModeAI, &bot, wire.SeatNumber(0), []string{"Ann"})
	yes := true
	c.Tutorial = &yes
	return c
}

// scriptKeys returns the script's first p keys.
func scriptKeys(sc *tutorial.Script, p int) []string {
	keys := []string{}
	for _, e := range sc.Entries[:p] {
		keys = append(keys, e.Key)
	}
	return keys
}

// follow drives a tutorial room along the script until its log holds p
// keys: the player sends the key each frame's tutorial object asks for,
// and the bot answers by itself. Every frame of the player's guided turns
// asks for the script's key and carries no move review. It returns the
// player's latest frame.
func follow(t *testing.T, h *harness, a *tclient, code string, sc *tutorial.Script, p int) wire.State {
	t.Helper()
	for {
		n := len(h.snapshot(code).Actions)
		st := a.current()
		if n >= p {
			return st
		}
		if sc.Entries[n].Seat != sc.Human {
			h.waitFor("the bot's scripted reply", func() bool { return len(h.snapshot(code).Actions) > n })
			continue
		}
		ts := st.Tutorial
		if ts == nil || ts.Status != wire.TutorialStatusGuided || ts.Action == nil || ts.Action.Key != sc.Entries[n].Key || ts.Left {
			t.Fatalf("at %d: tutorial %+v, want the guided turn of %s", n, ts, sc.Entries[n].Key)
		}
		if !reflect.DeepEqual(ts.Lessons, sc.Entries[n].Lessons) {
			t.Fatalf("at %d: lessons %+v, want %+v", n, ts.Lessons, sc.Entries[n].Lessons)
		}
		checkNoReview(t, "a guided turn", st)
		a.act(ts.Action.Key, st.Version, nil)
	}
}

func TestTutorialCreate(t *testing.T) {
	h, _ := newTutorialHarness(t)
	sc := tutorial.Current
	a := h.client("user-a", "tab-a", "Ann")
	code, st := a.create(tutorialCreate())

	// The script's seed and options, whatever the server's seed; the
	// player in seat 0 against the Easy bot.
	snap := h.snapshot(code)
	if snap.State.Seed != sc.Seed || snap.State.Opts != sc.Options || snap.Mode != store.ModeAI {
		t.Fatalf("tutorial room: seed %d options %+v mode %s", snap.State.Seed, snap.State.Opts, snap.Mode)
	}
	if s := snap.Seats; s[0].Kind != store.SeatKindHuman || s[0].UserID != "user-a" || s[0].Name != "Ann" ||
		s[1].Kind != store.SeatKindBot || s[1].Bot != store.BotEasy {
		t.Fatalf("tutorial seats: %+v", s)
	}
	if hdr, err := h.st.PeekHeader(code); err != nil || hdr.Tutorial != sc.Name || hdr.Seed != sc.Seed {
		t.Fatalf("stored tutorial room: %+v, %v", hdr, err)
	}
	if !st.Room.Tutorial || !st.Room.AllowUndo || !st.Room.Hints.Advisor {
		t.Fatalf("room info: tutorial %v, undo %v, advisor %v", st.Room.Tutorial, st.Room.AllowUndo, st.Room.Hints.Advisor)
	}
	ts := st.Tutorial
	if ts == nil || ts.Script != sc.Name || ts.Status != wire.TutorialStatusGuided || ts.Turn != 1 || ts.Turns != 10 ||
		ts.Action == nil || ts.Action.Key != sc.Entries[0].Key || len(ts.Lessons) != 1 || ts.Lessons[0].ID != wire.TutorialLessonDraftIntro {
		t.Fatalf("the first frame's tutorial: %+v", ts)
	}

	// The same player on another device, through a device link, gets it
	// too; another user cannot enter the room at all.
	claim := a.link(0, false).Claim
	phone := h.client("user-p", "tab-p", "Phone")
	if got := phone.joinClaim(code, claim); got.Tutorial == nil || !reflect.DeepEqual(*got.Tutorial, *ts) {
		t.Fatalf("the device link's frame: %+v", got.Tutorial)
	}
	b := h.client("user-b", "tab-b", "Bob")
	if e, err := b.joinErr(code); e.Code != wire.ErrorCodeForbidden || !errors.Is(err, room.ErrForbidden) {
		t.Fatalf("another user joining the tutorial: %v, %v", e, err)
	}

	// An ordinary bot room has no tutorial.
	easy := wire.BotLevelEasy
	_, plain := h.client("user-c", "tab-c", "Cy").create(wire.NewCreate(wire.ModeAI, &easy, wire.SeatNumber(0), nil))
	if plain.Room.Tutorial || plain.Tutorial != nil {
		t.Fatalf("an ordinary bot room: tutorial %v, %+v", plain.Room.Tutorial, plain.Tutorial)
	}
}

// TestTutorialCreateRefusals sends tutorial creates the wire would refuse
// straight to the Manager: it refuses them too.
func TestTutorialCreateRefusals(t *testing.T) {
	h, _ := newTutorialHarness(t)
	normal, easy := wire.BotLevelNormal, wire.BotLevelEasy
	for i, req := range []wire.Create{
		wire.NewCreate(wire.ModeOnline, nil, wire.SeatNumber(0), nil),
		wire.NewCreate(wire.ModeHotseat, nil, wire.SeatNumber(0), nil),
		wire.NewCreate(wire.ModeAI, &normal, wire.SeatNumber(0), nil),
		wire.NewCreate(wire.ModeAI, &easy, wire.SeatNumber(1), nil),
		wire.NewCreate(wire.ModeAI, &easy, wire.SeatRandom, nil),
	} {
		yes := true
		req.Tutorial = &yes
		c := h.client("user-a", "tab-"+string(rune('a'+i)), "Ann")
		if _, err := h.m.Create(context.Background(), c.Client, req); !errors.Is(err, room.ErrInvalidSpec) {
			t.Fatalf("case %d: Create = %v, want ErrInvalidSpec", i, err)
		}
		if _, ok := c.next().(wire.Error); !ok {
			t.Fatalf("case %d: no error frame", i)
		}
	}
	if h.m.Len() != 0 {
		t.Fatalf("%d rooms after refused creates", h.m.Len())
	}
}

// TestTutorialFollowsTheScript plays the player's script keys through the
// room: the bot answers with exactly the script's keys and never searches,
// no guided move is reviewed, the end shows the outro, and the first free
// move is reviewed and answered by the Easy bot.
func TestTutorialFollowsTheScript(t *testing.T) {
	h, calls := newTutorialHarness(t)
	sc := tutorial.Current
	a := h.client("user-a", "tab-a", "Ann")
	code, _ := a.create(tutorialCreate())
	n := len(sc.Entries)
	st := follow(t, h, a, code, sc, n)

	if got := h.snapshot(code).Actions; !slices.Equal(got, scriptKeys(sc, n)) {
		t.Fatalf("the log after the script:\n%q\nwant\n%q", got, scriptKeys(sc, n))
	}
	if got := calls.get(); len(got) != 0 {
		t.Fatalf("the bot searched during the script: %v", got)
	}
	ts := st.Tutorial
	if ts == nil || ts.Status != wire.TutorialStatusFree || ts.Left || ts.Action != nil || ts.Turn != 0 || !reflect.DeepEqual(ts.Lessons, sc.Outro) {
		t.Fatalf("after the script: %+v, want free with the outro", ts)
	}
	checkNoReview(t, "after the script", st)

	// A free move: reviewed, and the Easy bot replies.
	ply := st.View.Ply
	a.act(st.View.Choices[0].Key, st.Version, nil)
	h.waitFor("the Easy bot's reply", func() bool { return len(h.snapshot(code).Actions) > n+1 })
	st = a.current()
	checkReview(t, "a free move", st, ply)
	if got := calls.get(); !slices.Equal(got, []store.BotLevel{store.BotEasy}) {
		t.Fatalf("bots asked for after the script: %v, want one easy", got)
	}
	if ts := st.Tutorial; ts == nil || ts.Status != wire.TutorialStatusFree || len(ts.Lessons) != 0 {
		t.Fatalf("after a free move: %+v", ts)
	}
}

// TestTutorialLeavesThePath makes another move at a guided turn: the Easy
// bot replies, the frame shows leftPath, and an undo brings the guided turn
// back, after which the bot plays the script again.
func TestTutorialLeavesThePath(t *testing.T) {
	h, calls := newTutorialHarness(t)
	sc := tutorial.Current
	a := h.client("user-a", "tab-a", "Ann")
	code, _ := a.create(tutorialCreate())
	const at = 6 // the first Age I turn
	if sc.Entries[at].Seat != sc.Human {
		t.Fatalf("entry %d is not a guided turn", at)
	}
	st := follow(t, h, a, code, sc, at)
	guided := *st.Tutorial

	other := ""
	for _, c := range st.View.Choices {
		if c.Key != sc.Entries[at].Key {
			other = c.Key
			break
		}
	}
	a.act(other, st.Version, nil)
	h.waitFor("the Easy bot's reply", func() bool { return len(h.snapshot(code).Actions) > at+1 })
	st = a.current()
	ts := st.Tutorial
	if ts == nil || !ts.Left || ts.Status != wire.TutorialStatusFree || ts.Action != nil ||
		len(ts.Lessons) != 1 || ts.Lessons[0].ID != wire.TutorialLessonLeftPath {
		t.Fatalf("after leaving the path: %+v", ts)
	}
	if got := calls.get(); !slices.Equal(got, []store.BotLevel{store.BotEasy}) {
		t.Fatalf("bots asked for after leaving the path: %v, want one easy", got)
	}

	// Undo goes back before the player's move: on the script again.
	a.undo(st.Version)
	h.snapshot(code)
	st = a.current()
	if got := h.snapshot(code).Actions; !slices.Equal(got, scriptKeys(sc, at)) {
		t.Fatalf("the log after the undo: %q", got)
	}
	if st.Tutorial == nil || !reflect.DeepEqual(*st.Tutorial, guided) {
		t.Fatalf("after the undo: %+v, want %+v", st.Tutorial, guided)
	}
	follow(t, h, a, code, sc, at+4)
	if got := h.snapshot(code).Actions; !slices.Equal(got, scriptKeys(sc, at+4)) {
		t.Fatalf("back on the script: %q", got)
	}
	if got := calls.get(); len(got) != 1 {
		t.Fatalf("the bot searched back on the script: %v", got)
	}
}

// TestTutorialSurvivesReload evicts a tutorial room mid-script: reloaded
// from its file, it shows the same lesson.
func TestTutorialSurvivesReload(t *testing.T) {
	h, _ := newTutorialHarness(t)
	sc := tutorial.Current
	a := h.client("user-a", "tab-a", "Ann")
	code, _ := a.create(tutorialCreate())
	before := follow(t, h, a, code, sc, 8).Tutorial
	h.evict(code, a)
	after := h.client("user-a", "tab-a2", "Ann").join(code).Tutorial
	if before == nil || after == nil || !reflect.DeepEqual(*before, *after) {
		t.Fatalf("before the reload %+v, after %+v", before, after)
	}
	if after.Action == nil || after.Action.Key != sc.Entries[8].Key || after.Lessons[0].ID != wire.TutorialLessonReveal {
		t.Fatalf("after the reload: %+v", after)
	}
}

// TestTutorialForfeitAndRematch: a forfeit completes the tutorial, and the
// rematch is a normal bot room, with a fresh deal and default options.
func TestTutorialForfeitAndRematch(t *testing.T) {
	h, _ := newTutorialHarness(t)
	sc := tutorial.Current
	a := h.client("user-a", "tab-a", "Ann")
	code, _ := a.create(tutorialCreate())
	st := follow(t, h, a, code, sc, 6)
	a.act("resign:0", st.Version, nil)
	h.snapshot(code)
	st = a.current()
	if st.Room.Status != wire.RoomStatusFinished || st.Tutorial == nil || st.Tutorial.Status != wire.TutorialStatusComplete {
		t.Fatalf("after the forfeit: status %s, tutorial %+v", st.Room.Status, st.Tutorial)
	}

	code2, err := h.m.Rematch(context.Background(), a.Client, code)
	if err != nil {
		t.Fatalf("Rematch: %v", err)
	}
	st = a.join(code2)
	if st.Room.Tutorial || st.Tutorial != nil {
		t.Fatalf("the rematch is a tutorial: %v, %+v", st.Room.Tutorial, st.Tutorial)
	}
	snap := h.snapshot(code2)
	if snap.State.Opts != (engine.Options{}) || snap.State.Seed == sc.Seed || snap.Mode != store.ModeAI ||
		snap.Seats[0].UserID != "user-a" || snap.Seats[1].Bot != store.BotEasy {
		t.Fatalf("the rematch: seed %d options %+v mode %s seats %+v", snap.State.Seed, snap.State.Opts, snap.Mode, snap.Seats)
	}
	if hdr, err := h.st.PeekHeader(code2); err != nil || hdr.Tutorial != "" {
		t.Fatalf("the rematch's file: %+v, %v", hdr, err)
	}
}

// TestTutorialFromSpec makes tutorial rooms the way the development
// endpoint does: at a position of the script, with the script's seed and
// options whatever the spec says, and refuses one that does not fit.
func TestTutorialFromSpec(t *testing.T) {
	h, calls := newTutorialHarness(t)
	sc := tutorial.Current
	open := store.Seat{Kind: store.SeatKindOpen}
	easy := store.Seat{Kind: store.SeatKindBot, Bot: store.BotEasy}
	code, err := h.m.CreateFromSpec(room.Spec{
		Mode: store.ModeAI, Seats: [2]store.Seat{open, easy}, Seed: 1,
		Actions: scriptKeys(sc, 7), Tutorial: sc.Name,
	})
	if err != nil {
		t.Fatalf("CreateFromSpec: %v", err)
	}
	a := h.client("user-a", "tab-a", "Ann")
	st := a.join(code)
	if snap := h.snapshot(code); snap.State.Seed != sc.Seed || snap.State.Opts != sc.Options {
		t.Fatalf("seed %d options %+v, want the script's", snap.State.Seed, snap.State.Opts)
	}
	// The bot was to move: it plays the script's reply.
	h.waitFor("the bot's scripted reply", func() bool { return len(h.snapshot(code).Actions) > 7 })
	st = a.current()
	if got := h.snapshot(code).Actions; !slices.Equal(got, scriptKeys(sc, 8)) || len(calls.get()) != 0 {
		t.Fatalf("the log %q, bots %v", got, calls.get())
	}
	if st.Tutorial == nil || st.Tutorial.Action == nil || st.Tutorial.Action.Key != sc.Entries[8].Key || !st.Room.Tutorial {
		t.Fatalf("the dev tutorial room's frame: %+v", st.Tutorial)
	}

	normal := store.Seat{Kind: store.SeatKindBot, Bot: store.BotNormal}
	for name, spec := range map[string]room.Spec{
		"unknown script": {Mode: store.ModeAI, Seats: [2]store.Seat{open, easy}, Tutorial: "no-such-script"},
		"hotseat":        {Mode: store.ModeHotseat, Seats: [2]store.Seat{open, open}, Tutorial: sc.Name},
		"normal bot":     {Mode: store.ModeAI, Seats: [2]store.Seat{open, normal}, Tutorial: sc.Name},
		"bot in seat 0":  {Mode: store.ModeAI, Seats: [2]store.Seat{easy, open}, Tutorial: sc.Name},
		"scenario":       {Mode: store.ModeAI, Seats: [2]store.Seat{open, easy}, Scenario: "mausoleum", Tutorial: sc.Name},
	} {
		if _, err := h.m.CreateFromSpec(spec); !errors.Is(err, room.ErrInvalidSpec) {
			t.Errorf("%s: CreateFromSpec = %v, want ErrInvalidSpec", name, err)
		}
	}
}

// TestTutorialUnknownScript loads a room whose file names a script this
// build does not ship: it plays on as a normal game.
func TestTutorialUnknownScript(t *testing.T) {
	h, calls := newTutorialHarness(t)
	sc := tutorial.Current
	code, err := h.st.NewCode()
	if err != nil {
		t.Fatal(err)
	}
	f := store.New(code, store.ModeAI, sc.Options, sc.Seed, roomEpoch)
	f.Seats = [2]store.Seat{
		{Kind: store.SeatKindHuman, UserID: "user-a", Name: "Ann"},
		{Kind: store.SeatKindBot, Bot: store.BotEasy},
	}
	f.Admin = "user-a"
	f.Actions = scriptKeys(sc, 1)
	f.Tutorial = "retired-script"
	if err := h.st.Save(f); err != nil {
		t.Fatal(err)
	}
	a := h.client("user-a", "tab-a", "Ann")
	if st := a.join(code); st.Room.Tutorial || st.Tutorial != nil {
		t.Fatalf("a retired script's room: %v, %+v", st.Room.Tutorial, st.Tutorial)
	}
	h.waitFor("the bot's reply", func() bool { return len(h.snapshot(code).Actions) > 1 })
	if got := calls.get(); len(got) == 0 || got[0] != store.BotEasy {
		t.Fatalf("the bot of a retired script's room did not search: %v", got)
	}
}
