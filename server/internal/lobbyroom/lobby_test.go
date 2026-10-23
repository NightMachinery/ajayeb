package lobbyroom

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ident"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/room/roomtest"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/tableroom"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

const wait = 20 * time.Second

// fixture is the server side without the transport: the store with the
// lobby and table codecs, a Manager with Duel, the table kind and the
// lobby kind, and device links on.
type fixture struct {
	t       *testing.T
	dir     string
	clock   room.Clock
	secret  ident.Secret
	st      *store.Store
	m       *room.Manager
	tables  *tableroom.Tables
	lobbies *Lobbies
	// replaceFails makes the tables' next ReplaceGame fail.
	replaceFails bool
	// saves holds, for a code or a game, how many of its next writes pass
	// and how many after them fail (failSaves).
	mu    sync.Mutex
	saves map[string]*saveFault
	// codes are the Manager's next NewCode answers (codeStore).
	codes []string
}

type saveFault struct{ pass, fail int }

func newFixture(t *testing.T, clock room.Clock) *fixture {
	t.Helper()
	f := &fixture{t: t, dir: t.TempDir(), clock: clock}
	var err error
	if f.secret, err = ident.LoadSecret(f.dir); err != nil {
		t.Fatal(err)
	}
	f.start()
	t.Cleanup(f.close)
	return f
}

// failingStore fails ReplaceGame and SaveGame while the fixture says so.
type failingStore struct {
	*store.Store
	f *fixture
}

func (s failingStore) SaveGame(code, game string, data []byte) error {
	if s.f.failing(code) || s.f.failing(game) {
		return errors.New("injected")
	}
	return s.Store.SaveGame(code, game, data)
}

// codeStore answers the Manager's NewCode with the fixture's queued codes
// first, so a test can make two creators draw the same code.
type codeStore struct {
	*store.Store
	f *fixture
}

func (s codeStore) NewCode() (string, error) {
	s.f.mu.Lock()
	defer s.f.mu.Unlock()
	if len(s.f.codes) > 0 {
		code := s.f.codes[0]
		s.f.codes = s.f.codes[1:]
		return code, nil
	}
	return s.Store.NewCode()
}

// failSaves lets the next pass writes of key (a room code, or a game: a
// kind's every room) through, then fails the n after them.
func (f *fixture) failSaves(key string, pass, n int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.saves == nil {
		f.saves = map[string]*saveFault{}
	}
	f.saves[key] = &saveFault{pass: pass, fail: n}
}

func (f *fixture) failing(key string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	sf := f.saves[key]
	switch {
	case sf == nil || sf.fail == 0:
		return false
	case sf.pass > 0:
		sf.pass--
		return false
	}
	sf.fail--
	return true
}

func (s failingStore) CreateGame(code, game string, data []byte) ([]byte, error) {
	if s.f.failing(code) || s.f.failing(game) {
		return nil, errors.New("injected")
	}
	return s.Store.CreateGame(code, game, data)
}

func (s failingStore) ReplaceGame(code, from, game string, data []byte) error {
	if s.f.replaceFails {
		s.f.replaceFails = false
		return errors.New("injected")
	}
	return s.Store.ReplaceGame(code, from, game, data)
}

func (f *fixture) start() {
	f.t.Helper()
	f.tables = tableroom.New(tableroom.Config{Clock: f.clock, Creation: true})
	f.lobbies = New(Config{Clock: f.clock, Tables: f.tables, Creation: true})
	kinds, err := room.NewKinds(room.DuelKind(), f.tables.Kind(), f.lobbies.Kind())
	if err != nil {
		f.t.Fatal(err)
	}
	if f.st, err = store.OpenWithCodecs(f.dir, nil, kinds.Codecs()); err != nil {
		f.t.Fatal(err)
	}
	f.m, err = room.NewManager(room.Config{Store: codeStore{Store: f.st, f: f}, Kinds: kinds, KindOf: f.st.Game, Claims: f.secret, Clock: f.clock})
	if err != nil {
		f.t.Fatal(err)
	}
	f.tables.Bind(f.m, failingStore{Store: f.st, f: f})
	f.lobbies.Bind(f.m, failingStore{Store: f.st, f: f})
}

func (f *fixture) close() {
	if f.m != nil {
		f.m.Close()
		f.m = nil
	}
}

func (f *fixture) restart() {
	f.close()
	f.start()
}

func user(i int) string { return "user-" + strconv.Itoa(i) }

// conn is one connection and every frame it received.
type conn struct {
	t      *testing.T
	c      *room.Client
	frames []any
}

func (f *fixture) conn(u, id, name string) *conn {
	c := room.NewClient(u, id, name)
	c.ProtocolVersion = wire.PlatformProtocolVersion
	c.Addr = "addr-" + u
	return &conn{t: f.t, c: c}
}

func (k *conn) drain() {
	for _, m := range k.c.Out.Take() {
		k.frames = append(k.frames, m)
	}
}

// until waits for a frame satisfying ok among those not yet seen, and
// returns it.
func (k *conn) until(what string, ok func(any) bool) any {
	k.t.Helper()
	deadline := time.After(wait)
	seen := 0
	for {
		k.drain()
		for ; seen < len(k.frames); seen++ {
			if ok(k.frames[seen]) {
				fr := k.frames[seen]
				k.frames = k.frames[seen+1:]
				return fr
			}
		}
		select {
		case <-k.c.Out.Ready():
		case <-deadline:
			k.t.Fatalf("waiting for %s; got %d frames", what, len(k.frames))
		}
	}
}

func (k *conn) lobby(what string, ok func(wire.LobbyState) bool) wire.LobbyState {
	k.t.Helper()
	return k.until(what, func(v any) bool {
		st, is := v.(wire.LobbyState)
		return is && ok(st)
	}).(wire.LobbyState)
}

func (k *conn) table(what string, ok func(wire.TableState) bool) wire.TableState {
	k.t.Helper()
	return k.until(what, func(v any) bool {
		st, is := v.(wire.TableState)
		return is && ok(st)
	}).(wire.TableState)
}

func (k *conn) reject(seq int) string {
	k.t.Helper()
	return k.until("a reject", func(v any) bool {
		r, is := v.(wire.LobbyReject)
		return is && r.Seq == seq
	}).(wire.LobbyReject).Code
}

func (k *conn) acked(seq int) wire.LobbyState {
	k.t.Helper()
	return k.lobby("the ack of "+strconv.Itoa(seq), func(st wire.LobbyState) bool { return st.Ack != nil && st.Ack.Seq == seq })
}

func any2(wire.LobbyState) bool { return true }

// last is the latest lobby frame received so far; the frames before it
// are consumed.
func (k *conn) last() wire.LobbyState {
	k.t.Helper()
	k.drain()
	for i := len(k.frames) - 1; i >= 0; i-- {
		if st, ok := k.frames[i].(wire.LobbyState); ok {
			k.frames = k.frames[i+1:]
			return st
		}
	}
	k.t.Fatal("no lobby frame")
	return wire.LobbyState{}
}

func (f *fixture) ctx() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	f.t.Cleanup(cancel)
	return ctx
}

func (f *fixture) create(k *conn, seats int, opts wire.LobbyOptions) string {
	f.t.Helper()
	code, err := f.m.CreatePlatform(f.ctx(), k.c, wire.LobbyCreate{T: "lobby-create", SeatCount: seats, Options: opts})
	if err != nil {
		f.t.Fatal(err)
	}
	k.lobby("the creator's frame", any2)
	return code
}

func (f *fixture) join(k *conn, code string) {
	f.t.Helper()
	if err := f.m.Join(f.ctx(), k.c, code); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) send(k *conn, msg wire.ClientMessage) {
	f.t.Helper()
	if err := f.m.DispatchPlatform(f.ctx(), k.c, msg); err != nil {
		f.t.Fatal(err)
	}
}

func edit(code string, seq int, op string) wire.LobbyEdit {
	return wire.LobbyEdit{T: "lobby-edit", Code: code, Seq: seq, Op: op}
}

func ptr[T any](v T) *T { return &v }

func kinds(st wire.LobbyState) []string {
	var out []string
	for _, s := range st.Seats {
		out = append(out, s.Kind)
	}
	return out
}

// settle waits until the lobby's actor has handled everything delivered
// to it so far (a detach included, so its timers are armed).
func (f *fixture) settle(code string) {
	f.t.Helper()
	a := f.lobbies.lookup(code)
	if a == nil {
		f.t.Fatal("the lobby is not running")
	}
	if err := a.call(f.ctx(), func() {}); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fixture) record(code string) Record {
	f.t.Helper()
	data, _, err := f.st.PeekGame(code, Game)
	if err != nil {
		f.t.Fatal(err)
	}
	r, err := decodeRecord(data)
	if err != nil {
		f.t.Fatal(err)
	}
	return *r
}

// TestLobbyEditsAndStart walks a lobby from creation to Start: joins take
// open slots, a removed human stays out when they come back, an admin
// adds bots, seats a spectator and changes the seat count, a human
// releases their slot, refusals come in the documented order, a repeated
// seq is only acknowledged, and Start hands every connection, the
// spectator's included, the table under the same code with the roles
// carried on.
func TestLobbyEditsAndStart(t *testing.T) {
	f := newFixture(t, nil)
	ks := make([]*conn, 5)
	for i := range ks {
		ks[i] = f.conn(user(i), "c"+strconv.Itoa(i), "P"+strconv.Itoa(i))
	}
	code := f.create(ks[0], 4, wire.LobbyOptions{Seating: ptr("lobby")})
	for _, k := range ks[1:] {
		f.join(k, code)
	}
	st := ks[4].lobby("the spectator's frame", any2)
	if !slices.Equal(kinds(st), []string{SlotHuman, SlotHuman, SlotHuman, SlotHuman}) || st.You.Seat != nil {
		t.Fatalf("after the joins: %v, you %+v", kinds(st), st.You)
	}
	if !ks[0].lobby("the creator's frame", any2).CanStart {
		t.Fatal("the seated creator cannot start a full lobby")
	}

	f.send(ks[1], edit(code, 1, "remove-human"))
	if got := ks[1].reject(1); got != outNotAdmin {
		t.Fatalf("a non-admin's edit: %s", got)
	}
	e := edit(code, 1, "remove-human")
	e.UserID = ptr(user(3))
	f.send(ks[0], e)
	if st := ks[0].acked(1); st.Seats[3].Kind != SlotOpen {
		t.Fatalf("after the removal: %v", kinds(st))
	}
	f.send(ks[0], e)
	if st := ks[0].acked(1); st.Seats[3].Kind != SlotOpen {
		t.Fatal("a repeated seq changed something")
	}
	f.m.Leave(ks[3].c, code)
	f.join(ks[3], code)
	if st := ks[3].last(); st.Seats[3].Kind != SlotOpen || st.You.Seat != nil {
		t.Fatalf("the removed human took a slot again: %v", kinds(st))
	}
	ab := edit(code, 2, "add-bot")
	ab.Seat, ab.Bot = ptr(1), ptr(tableroom.BotHard)
	f.send(ks[0], ab)
	if got := ks[0].reject(2); got != outBad {
		t.Fatalf("a bot in a human's slot: %s", got)
	}
	ab.Seat = ptr(3)
	f.send(ks[0], ab)
	ks[0].acked(2)
	f.send(ks[2], edit(code, 1, "release"))
	if st := ks[2].acked(1); st.Seats[2].Kind != SlotOpen || st.You.Seat != nil {
		t.Fatalf("after the release: %v", kinds(st))
	}
	se := edit(code, 3, "seat")
	se.Seat, se.UserID = ptr(2), ptr(user(4))
	f.send(ks[0], se)
	if st := ks[4].lobby("the spectator seated", func(st wire.LobbyState) bool { return st.You.Seat != nil }); *st.You.Seat != 2 {
		t.Fatalf("the spectator's seat: %v", *st.You.Seat)
	}
	sc := edit(code, 4, "seats")
	sc.SeatCount = ptr(5)
	f.send(ks[0], sc)
	ks[0].acked(4)
	ro := wire.LobbyRole{T: "lobby-role", Code: code, Seq: 5, Op: "promote", UserID: ptr(user(1))}
	f.send(ks[0], ro)
	ks[0].acked(5)

	// Slot 4 is open: Start needs fillBots, and the sender must be a
	// seated admin.
	f.send(ks[3], wire.LobbyStart{T: "lobby-start", Code: code, Seq: 1, FillBots: true})
	if got := ks[3].reject(1); got != outNotAdmin {
		t.Fatalf("a spectator's Start: %s", got)
	}
	f.send(ks[1], wire.LobbyStart{T: "lobby-start", Code: code, Seq: 1})
	if got := ks[1].reject(1); got != outForbidden {
		t.Fatalf("a Start with an open slot: %s", got)
	}
	f.send(ks[1], wire.LobbyStart{T: "lobby-start", Code: code, Seq: 2, FillBots: true})
	for i, k := range ks {
		st := k.table("the table's first frame", func(wire.TableState) bool { return true })
		if st.Code != code || len(st.Seats) != 5 || st.History == nil {
			t.Fatalf("connection %d's first table frame: %+v", i, st)
		}
		if spectator := i == 2 || i == 3; spectator != (st.You.Seat == nil) {
			t.Fatalf("connection %d's seat: %v", i, st.You.Seat)
		}
	}
	if g, err := f.st.Game(code); err != nil || g != tableroom.Game {
		t.Fatalf("the room file after Start: %q %v", g, err)
	}
	if f.lobbies.lookup(code) != nil {
		t.Fatal("the lobby's actor still runs")
	}
	// Lobby order seating: slot s is seat s. The roles came along.
	f.m.Leave(ks[3].c, code)
	st0 := ks[0].table("user-3's departure", func(st wire.TableState) bool {
		i := slices.IndexFunc(st.Humans, func(h wire.PlatformHuman) bool { return h.UserID == user(3) })
		return i < 0
	})
	want := []string{"human", "human", "human", "bot", "bot"}
	for s, seat := range st0.Seats {
		if seat.Kind != want[s] {
			t.Fatalf("seat %d: %s, want %s", s, seat.Kind, want[s])
		}
	}
	i := slices.IndexFunc(st0.Humans, func(h wire.PlatformHuman) bool { return h.UserID == user(1) })
	if i < 0 || !st0.Humans[i].Admin || st0.Humans[i].PromotedBy == nil || *st0.Humans[i].PromotedBy != user(0) {
		t.Fatalf("user-1's role in the table: %+v", st0.Humans)
	}
	// A lobby request to the table is refused in its own family.
	_ = f.m.DispatchPlatform(f.ctx(), ks[0].c, wire.LobbyEdit{T: "lobby-edit", Code: code, Seq: 9, Op: "release"})
	if got := ks[0].reject(9); got != outBad {
		t.Fatalf("a lobby request to the table: %s", got)
	}
}

// TestStartRefusals: two seats wait for Duel's side of the lobby, a
// reservation must be opened first, and a failed write leaves the lobby
// as it was.
func TestStartRefusals(t *testing.T) {
	f := newFixture(t, nil)
	k := f.conn(user(0), "c0", "P0")
	code := f.create(k, 2, wire.LobbyOptions{})
	f.send(k, wire.LobbyStart{T: "lobby-start", Code: code, Seq: 1, FillBots: true})
	if got := k.reject(1); got != outForbidden {
		t.Fatalf("a two-seat Start: %s", got)
	}
	sc := edit(code, 2, "seats")
	sc.SeatCount = ptr(3)
	f.send(k, sc)
	k.acked(2)
	f.replaceFails = true
	f.send(k, wire.LobbyStart{T: "lobby-start", Code: code, Seq: 3, FillBots: true})
	if got := k.reject(3); got != outStorage {
		t.Fatalf("a Start whose write fails: %s", got)
	}
	if g, _ := f.st.Game(code); g != Game || f.lobbies.lookup(code) == nil {
		t.Fatalf("after the failed Start: %q", g)
	}
	// A reservation (a rematch's) blocks Start until an admin opens it.
	a := f.lobbies.lookup(code)
	if err := a.call(f.ctx(), func() {
		cand := a.rec.clone()
		cand.Slots[2] = Slot{Kind: SlotReserved, UserID: "someone", Name: "S"}
		if err := a.commit("test", cand); err != nil {
			t.Error(err)
		}
	}); err != nil {
		t.Fatal(err)
	}
	f.send(k, wire.LobbyStart{T: "lobby-start", Code: code, Seq: 4, FillBots: true})
	if got := k.reject(4); got != outForbidden {
		t.Fatalf("a Start with a reservation: %s", got)
	}
	op := edit(code, 5, "open-reservation")
	op.Seat = ptr(2)
	f.send(k, op)
	k.acked(5)
	f.send(k, wire.LobbyStart{T: "lobby-start", Code: code, Seq: 6, FillBots: true})
	k.table("the table", func(wire.TableState) bool { return true })
}

// TestLobbyRolesAcrossRestart: promotions, the temporary admin under the
// injected clock and the unseated list persist across a restart, and the
// temporary admin, a seated human, may start; the table carries the role.
func TestLobbyRolesAcrossRestart(t *testing.T) {
	t0 := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	clk := roomtest.NewFakeClock(t0)
	f := newFixture(t, clk)
	ks := make([]*conn, 4)
	for i := range ks {
		ks[i] = f.conn(user(i), "c"+strconv.Itoa(i), "P"+strconv.Itoa(i))
	}
	code := f.create(ks[0], 3, wire.LobbyOptions{})
	for _, k := range ks[1:] {
		f.join(k, code)
	}
	f.send(ks[0], wire.LobbyRole{T: "lobby-role", Code: code, Seq: 1, Op: "promote", UserID: ptr(user(3))})
	ks[0].acked(1)
	f.send(ks[3], wire.LobbyRole{T: "lobby-role", Code: code, Seq: 1, Op: "demote", UserID: ptr(user(0))})
	if got := ks[3].reject(1); got != outForbidden {
		t.Fatalf("demoting the creator: %s", got)
	}
	f.m.Leave(ks[0].c, code)
	f.m.Leave(ks[3].c, code)
	ks[1].lobby("the admins away", func(st wire.LobbyState) bool {
		return !slices.ContainsFunc(st.Humans, func(h wire.PlatformHuman) bool { return h.Admin && h.Connected })
	})
	f.settle(code)
	clk.Advance(5 * time.Minute)
	ks[2].lobby("user-1 as the temporary admin", func(st wire.LobbyState) bool {
		return slices.ContainsFunc(st.Humans, func(h wire.PlatformHuman) bool { return h.UserID == user(1) && h.Temporary })
	})
	before := f.record(code)
	f.restart()
	after := f.record(code)
	if after.Roles.Temporary != user(1) || len(after.Roles.Admins) != 2 || len(after.Roles.Away) != 2 || !after.Roles.Away[0].Since.Equal(before.Roles.Away[0].Since) {
		t.Fatalf("roles across the restart: %+v", after.Roles)
	}
	for _, a := range after.Roles.Admins {
		if a.UserID == user(3) && (a.PromotedBy == nil || *a.PromotedBy != user(0)) {
			t.Fatalf("user-3's promoter across the restart: %+v", a)
		}
	}
	ks[1], ks[2] = f.conn(user(1), "c1b", "P1"), f.conn(user(2), "c2b", "P2")
	f.join(ks[1], code)
	f.join(ks[2], code)
	if !ks[1].lobby("the temporary admin's frame", any2).CanStart {
		t.Fatal("the seated temporary admin cannot start")
	}
	f.send(ks[1], wire.LobbyStart{T: "lobby-start", Code: code, Seq: 1, FillBots: true})
	st := ks[2].table("the table", func(wire.TableState) bool { return true })
	if !slices.ContainsFunc(st.Humans, func(h wire.PlatformHuman) bool { return h.UserID == user(1) && h.Temporary }) {
		t.Fatalf("the table lost the temporary admin: %+v", st.Humans)
	}
	// The creator's return demotes the temporary admin in the table.
	ks[0] = f.conn(user(0), "c0b", "P0")
	f.join(ks[0], code)
	ks[2].table("the creator's return", func(st wire.TableState) bool {
		return !slices.ContainsFunc(st.Humans, func(h wire.PlatformHuman) bool { return h.Temporary })
	})
}

// TestDetachAcrossStart: a detach the Manager delivered to the lobby
// after Start handed its connections over reaches the table, so the
// connection gets no table frame after it; and one that races Start is
// never lost either way.
func TestDetachAcrossStart(t *testing.T) {
	for round := range 20 {
		f := newFixture(t, nil)
		ks := make([]*conn, 3)
		for i := range ks {
			ks[i] = f.conn(user(i), "c"+strconv.Itoa(i), "P"+strconv.Itoa(i))
		}
		code := f.create(ks[0], 3, wire.LobbyOptions{})
		f.join(ks[1], code)
		f.join(ks[2], code)
		la := f.lobbies.lookup(code)
		done := make(chan struct{})
		go func() {
			defer close(done)
			if round%2 == 0 {
				f.m.Leave(ks[1].c, code)
			}
		}()
		f.send(ks[0], wire.LobbyStart{T: "lobby-start", Code: code, Seq: 1, FillBots: true})
		ks[0].table("the table", func(wire.TableState) bool { return true })
		<-done
		if round%2 == 1 {
			// After the handover: straight to the lobby's hook, as a
			// Manager that looked the code up just before the swap would.
			la.detachHook(ks[1].c)
		}
		// The resync's answer comes after the table's first frames went to
		// every connection handed over, and after the detach.
		if err := f.m.Resync(f.ctx(), ks[0].c, code); err != nil {
			t.Fatal(err)
		}
		ks[0].table("the resync", func(st wire.TableState) bool { return st.History != nil })
		ks[1].drain()
		seen := len(ks[1].frames)
		f.m.Leave(ks[2].c, code)
		ks[0].table("user-2's departure", func(st wire.TableState) bool {
			return slices.ContainsFunc(st.Humans, func(h wire.PlatformHuman) bool { return h.UserID == user(2) && !h.Connected })
		})
		// The resync's answer comes after that broadcast has gone to all.
		if err := f.m.Resync(f.ctx(), ks[0].c, code); err != nil {
			t.Fatal(err)
		}
		ks[0].table("the resync", func(st wire.TableState) bool { return st.History != nil })
		ks[1].drain()
		if len(ks[1].frames) != seen {
			t.Fatalf("round %d: a detached connection got a table frame", round)
		}
		f.close()
	}
}

// TestRematch: a finished table's original human asks for a rematch: one
// successor lobby for every request, with the old seat count, options and
// Start-time bots, the human's slot reserved and the requester its only
// admin. A connection that joined the table with the human's device link
// gets the successor's claim, which takes the reserved slot without
// renaming it. A spectator who never played may not ask.
func TestRematch(t *testing.T) {
	f := newFixture(t, nil)
	k := f.conn(user(0), "c0", "P0")
	code := f.create(k, 3, wire.LobbyOptions{Seating: ptr("lobby"), AllowUndo: ptr(false), FinalReveal: ptr(false)})
	ab := edit(code, 1, "add-bot")
	ab.Seat, ab.Bot = ptr(2), ptr(tableroom.BotHard)
	f.send(k, ab)
	k.acked(1)
	f.send(k, wire.LobbyStart{T: "lobby-start", Code: code, Seq: 2, FillBots: true})
	st := k.table("the table", func(wire.TableState) bool { return true })
	claim := f.secret.Claim(code, user(0), 0)
	dev := f.conn("device", "d0", "Phone")
	if err := f.m.JoinClaim(f.ctx(), dev.c, code, &claim); err != nil {
		t.Fatal(err)
	}
	spec := f.conn(user(9), "c9", "Watcher")
	f.join(spec, code)
	seq := 1
	for st.Status != "finished" {
		if st.Point != nil && slices.Contains(st.Point.Required, 0) && !slices.Contains(st.Point.Confirmed, 0) && st.View.You != nil && len(st.View.You.Choices) > 0 {
			ref := st.Point.Ref
			f.send(k, wire.TablePick{T: "table-pick", Code: code, Seq: seq, Point: ref, Key: st.View.You.Choices[0].Key, Confirm: true})
			seq++
			st = k.table("the pick's answer", func(n wire.TableState) bool {
				return n.Status == "finished" || n.Point == nil || n.Point.Ref != ref || slices.Contains(n.Point.Confirmed, 0)
			})
			continue
		}
		st = k.table("the next point", func(wire.TableState) bool { return true })
	}
	if _, err := f.m.Rematch(f.ctx(), spec.c, code); err != nil {
		t.Fatal(err)
	}
	spec.until("the spectator's refusal", func(v any) bool { r, is := v.(wire.TableReject); return is && r.Code == outForbidden })
	if _, err := f.m.Rematch(f.ctx(), k.c, code); err != nil {
		t.Fatal(err)
	}
	created := func(c *conn) wire.Created {
		return c.until("created", func(v any) bool { _, is := v.(wire.Created); return is }).(wire.Created)
	}
	next := created(k)
	if next.Code == code || next.Claim != nil {
		t.Fatalf("the requester's created frame: %+v", next)
	}
	if c := created(spec); c.Code != next.Code || c.Claim != nil {
		t.Fatalf("the spectator's created frame: %+v", c)
	}
	dc := created(dev)
	if dc.Code != next.Code || dc.Claim == nil || *dc.Claim != f.secret.Claim(next.Code, user(0), 0) {
		t.Fatalf("the device's created frame: %+v", dc)
	}
	if _, err := f.m.Rematch(f.ctx(), dev.c, code); err != nil {
		t.Fatal(err)
	}
	if again := created(dev); again.Code != next.Code {
		t.Fatalf("a second rematch: %s, then %s", next.Code, again.Code)
	}
	r := f.record(next.Code)
	if r.SeatCount != 3 || r.Slots[0] != (Slot{Kind: SlotReserved, UserID: user(0), Name: "P0"}) || r.Slots[1].Bot != tableroom.BotEasy ||
		r.Slots[2].Bot != tableroom.BotHard || *r.Options.Seating != "lobby" || *r.Options.AllowUndo || *r.Options.FinalReveal ||
		r.Roles.Creator != user(0) || len(r.Roles.Admins) != 1 {
		t.Fatalf("the successor: %+v", r)
	}
	if err := f.m.JoinClaim(f.ctx(), dev.c, next.Code, dc.Claim); err != nil {
		t.Fatal(err)
	}
	ls := dev.lobby("the device in the successor", any2)
	if ls.You.UserID != user(0) || ls.You.Seat == nil || *ls.You.Seat != 0 || ls.Humans[0].Name != "P0" || !ls.Humans[0].Admin {
		t.Fatalf("the claimed slot: %+v %+v", ls.You, ls.Humans)
	}
}
