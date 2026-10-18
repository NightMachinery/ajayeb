package room_test

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/ident"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/room/roomtest"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// waitLimit is a safety net for a hung test, never a timing assumption:
// every wait below returns as soon as the frame or condition it waits for
// is there. The race detector slows the room actor and any real bot search
// enough to need a wider margin, so it is bigger under raceEnabled.
var waitLimit = 20 * time.Second

func init() {
	if raceEnabled {
		waitLimit = 60 * time.Second
	}
}

// roomEpoch is the fake clock's starting time.
var roomEpoch = time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)

// harness is a Manager over a temporary store and a fake clock.
type harness struct {
	t      *testing.T
	m      *room.Manager
	st     *store.Store
	clock  *roomtest.FakeClock
	cfg    room.Config
	secret ident.Secret // the Claims of device links
}

// newHarness starts a Manager whose bots play the first legal action, with
// no think delay. opts adjust the config. Cleanup closes the Manager and
// checks that no goroutine was left behind.
func newHarness(t *testing.T, opts ...func(*room.Config)) *harness {
	t.Helper()
	baseline := runtime.NumGoroutine()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	secret, err := ident.LoadSecret(t.TempDir())
	if err != nil {
		t.Fatalf("ident.LoadSecret: %v", err)
	}
	clock := roomtest.NewFakeClock(roomEpoch)
	var seed atomic.Uint64
	cfg := room.Config{
		Store:  st,
		Clock:  clock,
		NewBot: func(store.BotLevel, uint64) room.Bot { return firstLegalBot{} },
		Seed:   func() uint64 { return 1000 + seed.Add(1) },
		Claims: secret,
		// Small searches keep the two-stage reviews (the default) fast.
		ReviewBudget: room.ReviewBudget{Compare: 200, Search: 400},
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	m, err := room.NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	h := &harness{t: t, m: m, st: st, clock: clock, cfg: cfg, secret: secret}
	t.Cleanup(func() {
		m.Close()
		checkGoroutines(t, baseline)
	})
	return h
}

// checkGoroutines fails the test if more goroutines run than before it.
func checkGoroutines(t *testing.T, baseline int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for runtime.NumGoroutine() > baseline {
		if time.Now().After(deadline) {
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, true)
			t.Fatalf("%d goroutines still running, %d before the test:\n%s", runtime.NumGoroutine(), baseline, buf[:n])
		}
		runtime.Gosched()
	}
}

// snapshot is a barrier as well as a read: the room handles messages in
// order, so once it answers, every earlier request has been handled and its
// frames are in the outboxes.
func (h *harness) snapshot(code string) room.Snapshot {
	h.t.Helper()
	snap, err := h.m.Snapshot(context.Background(), code)
	if err != nil {
		h.t.Fatalf("Snapshot(%s): %v", code, err)
	}
	return snap
}

// waitFor polls cond (through room snapshots, which are cheap) until it
// holds, for conditions that another goroutine (a bot) brings about.
func (h *harness) waitFor(what string, cond func() bool) {
	h.t.Helper()
	deadline := time.Now().Add(waitLimit)
	for !cond() {
		if time.Now().After(deadline) {
			h.t.Fatalf("timed out waiting for %s", what)
		}
		runtime.Gosched()
	}
}

// tclient is a test connection: a room.Client plus a reader of its outbox.
type tclient struct {
	*room.Client
	t    *testing.T
	h    *harness
	buf  []wire.ServerMessage
	seq  int
	code string
	last *wire.State // the last state frame read
}

func (h *harness) client(userID, clientID, name string) *tclient {
	return &tclient{Client: room.NewClient(userID, clientID, name), t: h.t, h: h}
}

// create sends a create frame and returns the new room's code, after the
// created frame and the first state frame.
func (c *tclient) create(req wire.Create) (string, wire.State) {
	c.t.Helper()
	code, err := c.h.m.Create(context.Background(), c.Client, req)
	if err != nil {
		c.t.Fatalf("Create: %v", err)
	}
	created, ok := c.next().(wire.Created)
	if !ok || created.Code != code {
		c.t.Fatalf("first frame after create: want created %s, got %#v", code, created)
	}
	c.code = code
	return code, c.waitState(anyState)
}

// join joins code and returns the first state frame.
func (c *tclient) join(code string) wire.State {
	c.t.Helper()
	if err := c.h.m.Join(context.Background(), c.Client, code); err != nil {
		c.t.Fatalf("Join(%s): %v", code, err)
	}
	c.code = code
	return c.waitState(func(s wire.State) bool { return s.Code == code })
}

// joinErr joins code expecting it to fail, and returns the error frame.
func (c *tclient) joinErr(code string) (wire.Error, error) {
	c.t.Helper()
	err := c.h.m.Join(context.Background(), c.Client, code)
	if err == nil {
		c.t.Fatalf("Join(%s): want an error, got none", code)
	}
	e, ok := c.next().(wire.Error)
	if !ok {
		c.t.Fatalf("Join(%s) failed with %v but sent no error frame", code, err)
	}
	return e, err
}

// act sends an act frame with the next seq and returns that seq.
func (c *tclient) act(key string, base int, seat *int) int {
	c.t.Helper()
	c.seq++
	if err := c.h.m.Act(context.Background(), c.Client, wire.NewAct(c.code, c.seq, base, key, seat)); err != nil {
		c.t.Fatalf("Act: %v", err)
	}
	return c.seq
}

// actSeq sends an act frame with an explicit seq.
func (c *tclient) actSeq(seq int, key string, base int, seat *int) {
	c.t.Helper()
	if err := c.h.m.Act(context.Background(), c.Client, wire.NewAct(c.code, seq, base, key, seat)); err != nil {
		c.t.Fatalf("Act: %v", err)
	}
}

// undo sends an undo frame with the next seq and returns that seq.
func (c *tclient) undo(base int) int {
	c.t.Helper()
	c.seq++
	if err := c.h.m.Undo(context.Background(), c.Client, wire.NewUndo(c.code, c.seq, base)); err != nil {
		c.t.Fatalf("Undo: %v", err)
	}
	return c.seq
}

// fill moves the outbox's frames into buf, waiting for at least one when
// block is set.
func (c *tclient) fill(block bool) {
	c.t.Helper()
	if msgs := c.Out.Take(); len(msgs) > 0 || !block {
		c.buf = append(c.buf, msgs...)
		return
	}
	timer := time.NewTimer(waitLimit)
	defer timer.Stop()
	for {
		select {
		case <-c.Out.Ready():
			if msgs := c.Out.Take(); len(msgs) > 0 {
				c.buf = append(c.buf, msgs...)
				return
			}
		case <-timer.C:
			c.t.Fatalf("client %s: no frame arrived", c.ClientID)
		}
	}
}

// next returns the next frame, waiting for it.
func (c *tclient) next() wire.ServerMessage {
	c.t.Helper()
	if len(c.buf) == 0 {
		c.fill(true)
	}
	m := c.buf[0]
	c.buf = c.buf[1:]
	c.note(m)
	return m
}

// note remembers m if it is a state frame.
func (c *tclient) note(m wire.ServerMessage) {
	if st, ok := m.(wire.State); ok {
		c.last = &st
	}
}

// drain returns every frame already delivered, without waiting. Call it
// after a harness snapshot to see everything a request produced.
func (c *tclient) drain() []wire.ServerMessage {
	c.fill(false)
	out := c.buf
	c.buf = nil
	for _, m := range out {
		c.note(m)
	}
	return out
}

// current reads every frame already delivered and returns the latest state
// frame. A reject or error frame among them fails the test. Call it after a
// harness snapshot, which makes sure every earlier request was handled.
func (c *tclient) current() wire.State {
	c.t.Helper()
	for _, m := range c.drain() {
		switch m.(type) {
		case wire.Reject, wire.Error:
			c.t.Fatalf("client %s: unexpected frame %#v", c.ClientID, m)
		}
	}
	if c.last == nil {
		c.t.Fatalf("client %s: no state frame yet", c.ClientID)
	}
	return *c.last
}

func anyState(wire.State) bool { return true }

// waitState returns the first state frame that satisfies ok, skipping
// earlier state frames. A reject or error frame on the way fails the test.
func (c *tclient) waitState(ok func(wire.State) bool) wire.State {
	c.t.Helper()
	for {
		switch m := c.next().(type) {
		case wire.State:
			if ok(m) {
				return m
			}
		case wire.Reject, wire.Error:
			c.t.Fatalf("client %s: unexpected frame %#v", c.ClientID, m)
		}
	}
}

// expectReject waits for the next frame that is not a state frame and
// checks that it rejects seq with code.
func (c *tclient) expectReject(seq int, code wire.RejectCode) wire.Reject {
	c.t.Helper()
	for {
		m := c.next()
		if _, ok := m.(wire.State); ok {
			continue
		}
		r, ok := m.(wire.Reject)
		if !ok || r.Seq != seq || r.Code != code {
			c.t.Fatalf("client %s: want reject %s for seq %d, got %#v", c.ClientID, code, seq, m)
		}
		return r
	}
}

// firstLegalBot plays the first legal action of its observation.
type firstLegalBot struct{}

func (firstLegalBot) Choose(_ context.Context, obs engine.Observation, _ ai.Budget, _ func(ai.Progress)) (engine.Action, error) {
	legal := engine.LegalActions(obs.Public(), nil)
	if len(legal) == 0 {
		return engine.Action{}, errors.New("no legal action")
	}
	return legal[0], nil
}

// gateBot reports each search on started and then waits for release (or
// for its context to be cancelled) before playing the first legal action.
type gateBot struct {
	started chan engine.Observation
	release chan struct{}
}

func newGateBot() *gateBot {
	return &gateBot{started: make(chan engine.Observation, 16), release: make(chan struct{})}
}

func (g *gateBot) Choose(ctx context.Context, obs engine.Observation, b ai.Budget, p func(ai.Progress)) (engine.Action, error) {
	g.started <- obs
	select {
	case <-g.release:
	case <-ctx.Done():
	}
	return firstLegalBot{}.Choose(ctx, obs, b, p)
}

// waitStarted waits for the next search to start and returns its
// observation.
func (g *gateBot) waitStarted(t *testing.T) engine.Observation {
	t.Helper()
	select {
	case obs := <-g.started:
		return obs
	case <-time.After(waitLimit):
		t.Fatalf("the bot never started a search")
	}
	return engine.Observation{}
}

// flakyStore is a store whose writes fail while fail is set.
type flakyStore struct {
	*store.Store
	fail atomic.Bool
}

func (f *flakyStore) Save(file *store.File) error {
	if f.fail.Load() {
		return errors.New("disk full")
	}
	return f.Store.Save(file)
}

func seatPtr(s int) *int { return &s }

// mustScenarioTrigger returns the trigger keys of a named scenario.
func mustScenarioTrigger(t *testing.T, name string) []string {
	t.Helper()
	sc := engine.NamedScenario(name)
	if sc == nil {
		t.Fatalf("no named scenario %q", name)
	}
	return sc.Trigger
}
