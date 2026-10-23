package tableroom

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ident"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/faults"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/leakcheck"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

const wait = 20 * time.Second

// faultyStore is the real room store with injected faults: a failed write,
// or a crash before or after the rename, done by tabletest/faults'
// FileStore on the live file's own path.
type faultyStore struct {
	*store.Store
	mu     sync.Mutex
	queued []faults.Fault
	crashs int
}

func (s *faultyStore) inject(f ...faults.Fault) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queued = append(s.queued, f...)
}

func (s *faultyStore) SaveGame(code, game string, data []byte) error {
	s.mu.Lock()
	f := faults.None
	if len(s.queued) > 0 {
		f, s.queued = s.queued[0], s.queued[1:]
	}
	s.mu.Unlock()
	if f == faults.None {
		return s.Store.SaveGame(code, game, data)
	}
	fs := &faults.FileStore{Path: filepath.Join(s.Dir(), code+".json")}
	fs.Inject(f)
	err := fs.Save(context.Background(), data)
	if faults.IsCrash(err) {
		s.mu.Lock()
		s.crashs++
		s.mu.Unlock()
	}
	return err
}

// memStore keeps records in memory, for the exhaustive arrival-order tests
// that open thousands of tables and for the fuzzer: the actor's logic is
// the same, and the real store's writes are tested everywhere else. It
// outlives a fixture's restarts, and loading through it refuses any
// record that would need a refresh or an archive.
type memStore struct {
	mu   sync.Mutex
	recs map[string][]byte
}

func (s *memStore) LoadGame(code, _ string, assess func([]byte) (store.Assessment, error)) ([]byte, error) {
	data, _, err := s.PeekGame(code, Game)
	if err != nil {
		return nil, err
	}
	if v, err := assess(data); err != nil || v.Archive != "" || len(v.Refresh) > 0 {
		return nil, fmt.Errorf("memStore: unexpected assessment %+v (%v)", v, err)
	}
	return data, nil
}

func (s *memStore) SaveGame(code, _ string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recs[code] = slices.Clone(data)
	return nil
}

func (s *memStore) ReplaceGame(code, _, _ string, data []byte) error {
	return s.SaveGame(code, Game, data)
}

func (s *memStore) PeekGame(code, _ string) ([]byte, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.recs[code]
	if !ok {
		return nil, false, fs.ErrNotExist
	}
	return slices.Clone(data), false, nil
}

// logBuffer captures a table's log for the log-capture tests.
type logBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *logBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *logBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

// fixture is a whole server side without the transport: the store with the
// table codec, a room Manager with Duel and the table kind, and the
// tables.
type fixture struct {
	t      testing.TB
	dir    string
	store  *faultyStore
	m      *room.Manager
	tables *Tables
	logs   *logBuffer
	cfg    Config
	mem    bool      // keep tables in a memStore
	ms     *memStore // that store, which outlives restarts
	tabs   bool      // openInproc also joins each human seat's second tab
	secret ident.Secret
	succ   Successors // the rematch maker, kept across restarts
}

func newFixture(t testing.TB, cfg Config) *fixture {
	t.Helper()
	return newFixtureOn(t, cfg, false)
}

func newFixtureOn(t testing.TB, cfg Config, mem bool) *fixture {
	t.Helper()
	f := &fixture{t: t, dir: t.TempDir(), cfg: cfg, mem: mem}
	var err error
	if f.secret, err = ident.LoadSecret(f.dir); err != nil {
		t.Fatal(err)
	}
	f.start()
	t.Cleanup(f.close)
	return f
}

func (f *fixture) start() {
	f.t.Helper()
	f.logs = &logBuffer{}
	cfg := f.cfg
	cfg.Creation = true
	cfg.Logger = slog.New(slog.NewJSONHandler(f.logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	f.tables = New(cfg)
	if f.succ != nil {
		f.tables.SetSuccessors(f.succ)
	}
	kinds, err := room.NewKinds(room.DuelKind(), f.tables.Kind())
	if err != nil {
		f.t.Fatal(err)
	}
	st, err := store.OpenWithCodecs(f.dir, nil, kinds.Codecs())
	if err != nil {
		f.t.Fatal(err)
	}
	f.store = &faultyStore{Store: st}
	kindOf := st.Game
	if f.mem && f.ms == nil {
		f.ms = &memStore{recs: map[string][]byte{}}
	}
	if f.mem {
		kindOf = func(code string) (string, error) {
			if _, _, err := f.ms.PeekGame(code, Game); err == nil {
				return Game, nil
			}
			return st.Game(code)
		}
	}
	f.m, err = room.NewManager(room.Config{Store: st, Kinds: kinds, KindOf: kindOf, Logger: cfg.Logger, Claims: f.secret})
	if err != nil {
		f.t.Fatal(err)
	}
	if f.mem {
		f.tables.Bind(f.m, f.ms)
		return
	}
	f.tables.Bind(f.m, f.store)
}

func (f *fixture) close() {
	if f.m != nil {
		f.m.Close()
		f.m = nil
	}
}

// restart drops everything in memory, as a crash does, and starts again on
// the same data directory.
func (f *fixture) restart() {
	f.close()
	f.start()
}

func userOf(seat int) string { return "user-" + strconv.Itoa(seat) }

// humans returns a spec of n human seats in lobby order (chosen seating,
// so lobby slot s is seat s) with the given seed.
func humans(n int, seed uint64) Spec {
	sp := Spec{Seed: &seed, Creator: userOf(0)}
	for s := range n {
		sp.Participants = append(sp.Participants, Participant{UserID: userOf(s), Name: "P" + strconv.Itoa(s)})
	}
	return sp
}

// --- tabletest.Table over the actor, in process ---------------------------

// inproc drives a table through the real Manager: clients join with
// Manager.Join, picks go to the actor's own submit on its goroutine (the
// path room.Inbound takes), and every frame pushed to every client is
// drained and kept for the leak checks.
type inproc struct {
	f       *fixture
	code    string
	clients map[tabletest.ClientID]*room.Client
	frames  map[tabletest.ClientID][]wire.TableState
	rejects map[tabletest.ClientID][]wire.TableReject
	// other holds every other frame each client received (linked,
	// created, error).
	other map[tabletest.ClientID][]wire.ServerMessage
	// track keeps the test's own model of each seat's accepted pending
	// pick, from the outcomes of its submissions, and wants holds, beside
	// each frame, the pick the model says its viewer's seat had when it
	// was sent. Set it only where submissions are the only change (no
	// bots, no undo).
	track bool
	seats map[string]int // user -> seat
	model map[int]wire.TablePending
	wants map[tabletest.ClientID][]pendingWant
	// viewers holds, beside each frame, its viewer as the test's model of
	// seats and admins says (seats, admins) when the frame was drained. A
	// test that switches seats or roles updates the model before the
	// change's frames are drained.
	admins  map[string]bool
	viewers map[tabletest.ClientID][]frameViewer
}

func newClient(user, client, name string) *room.Client {
	c := room.NewClient(user, client, name)
	c.ProtocolVersion = wire.PlatformProtocolVersion
	c.Addr = "addr-" + user
	return c
}

// openInproc creates a table from spec and joins one client per seat of
// the driver (tabletest.SeatClient) and a spectator.
func (f *fixture) openInproc(ctx context.Context, spec Spec) (*inproc, error) {
	code, err := f.tables.CreateTable(spec)
	if err != nil {
		return nil, err
	}
	p := &inproc{f: f, code: code, clients: map[tabletest.ClientID]*room.Client{},
		frames: map[tabletest.ClientID][]wire.TableState{}, rejects: map[tabletest.ClientID][]wire.TableReject{},
		seats: map[string]int{}, model: map[int]wire.TablePending{}, wants: map[tabletest.ClientID][]pendingWant{},
		admins: map[string]bool{spec.Creator: true}, viewers: map[tabletest.ClientID][]frameViewer{}}
	// The driver's client for seat s is the user seated there, whatever the
	// seating rule made of the lobby order.
	var seats []SeatRecord
	a, err := p.actor()
	if err == nil {
		err = a.call(ctx, func() { seats = slices.Clone(a.st.rec.Seats) })
	}
	if err != nil {
		return nil, err
	}
	for s, sr := range seats {
		user, name := sr.UserID, sr.Name
		if sr.Kind == SeatBot {
			continue
		}
		p.seats[user] = s
		if err := p.join(ctx, tabletest.SeatClient(s), user, name); err != nil {
			return nil, err
		}
		if f.tabs {
			if err := p.join(ctx, tabletest.TabClient(s), user, name); err != nil {
				return nil, err
			}
		}
	}
	if err := p.join(ctx, tabletest.SpectatorClient, "watcher", "Watcher"); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *inproc) join(ctx context.Context, id tabletest.ClientID, user, name string) error {
	c := newClient(user, string(id), name)
	if err := p.f.m.Join(ctx, c, p.code); err != nil {
		return err
	}
	p.clients[id] = c
	p.drain()
	return nil
}

// leave detaches every client, so the table may be evicted.
func (p *inproc) leave() {
	for _, c := range p.clients {
		p.f.m.Leave(c, p.code)
	}
}

func (p *inproc) drain() {
	if p.wants == nil {
		p.wants = map[tabletest.ClientID][]pendingWant{}
	}
	if p.viewers == nil {
		p.viewers = map[tabletest.ClientID][]frameViewer{}
	}
	for id, c := range p.clients {
		for _, m := range c.Out.Take() {
			switch m := m.(type) {
			case wire.TableState:
				p.frames[id] = append(p.frames[id], m)
				w := pendingWant{Known: p.track}
				if seat, ok := p.seats[c.UserID]; ok {
					if pd, ok := p.model[seat]; ok {
						w.Pending = &pd
					}
				}
				p.wants[id] = append(p.wants[id], w)
				v := frameViewer{Seat: leakcheck.Spectator, Client: string(id), Admin: p.admins[c.UserID]}
				if seat, ok := p.seats[c.UserID]; ok {
					v.Seat = seat
				}
				p.viewers[id] = append(p.viewers[id], v)
			case wire.TableReject:
				p.rejects[id] = append(p.rejects[id], m)
			default:
				if p.other == nil {
					p.other = map[tabletest.ClientID][]wire.ServerMessage{}
				}
				p.other[id] = append(p.other[id], m)
			}
		}
	}
}

func (p *inproc) actor() (*actor, error) {
	a := p.f.tables.lookup(p.code)
	if a == nil {
		return nil, errors.New("the table is not running")
	}
	return a, nil
}

func (p *inproc) Snapshot(ctx context.Context, id tabletest.ClientID) (tabletest.Snapshot, error) {
	c := p.clients[id]
	if c == nil {
		return tabletest.Snapshot{}, fmt.Errorf("unknown client %s", id)
	}
	a, err := p.actor()
	if err != nil {
		return tabletest.Snapshot{}, err
	}
	var fr wire.TableState
	if err := a.call(ctx, func() { fr = a.newBroadcast(false).frame(c, nil, false) }); err != nil {
		return tabletest.Snapshot{}, err
	}
	return snapshotOf(fr), nil
}

func (p *inproc) Submit(ctx context.Context, id tabletest.ClientID, s tabletest.Submission) (tabletest.Outcome, error) {
	c := p.clients[id]
	if c == nil {
		return "", fmt.Errorf("unknown client %s", id)
	}
	a, err := p.actor()
	if err != nil {
		return "", err
	}
	crashs := p.f.store.crashes()
	var out string
	pick := wire.TablePick{T: "table-pick", Code: p.code, Seq: int(s.Seq), Point: parseRef(s.Ref), Key: s.Key, Confirm: s.Confirm}
	if err := a.call(ctx, func() { out = a.submit(c, pick) }); err != nil {
		return "", err
	}
	if seat, ok := p.seats[c.UserID]; ok {
		switch out {
		case outAccepted:
			p.model[seat] = wire.TablePending{Key: s.Key, Confirm: s.Confirm}
		case outResolved:
			clear(p.model)
		}
	}
	p.drain()
	if p.f.store.crashes() > crashs {
		return "", faults.ErrCrash
	}
	return outcomeOf(out), nil
}

func (s *faultyStore) crashes() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.crashs
}

func (p *inproc) Canonical(ctx context.Context) (tabletest.Canonical, error) {
	a, err := p.actor()
	if err != nil {
		return tabletest.Canonical{}, err
	}
	var c tabletest.Canonical
	err = a.call(ctx, func() {
		c.Hash = a.st.rec.Hash
		for _, b := range a.st.rec.Batches {
			tb := tabletest.Batch{Point: b.Point, Picks: []tabletest.Pick{}}
			for _, pk := range b.Picks {
				tb.Picks = append(tb.Picks, tabletest.Pick{Seat: pk.Seat, Key: pk.Key})
			}
			c.Batches = append(c.Batches, tb)
		}
	})
	if c.Batches == nil {
		c.Batches = []tabletest.Batch{}
	}
	return c, err
}

func outcomeOf(out string) tabletest.Outcome {
	switch out {
	case outAccepted:
		return tabletest.Accepted
	case outResolved:
		return tabletest.Resolved
	case outDuplicate:
		return tabletest.Duplicate
	case outStale:
		return tabletest.Stale
	case outIllegal:
		return tabletest.Illegal
	case outClosed:
		return tabletest.Closed
	case outForbidden:
		return tabletest.Forbidden
	case outRate:
		return tabletest.Rate
	case outStorage:
		return tabletest.StorageError
	}
	return tabletest.Outcome("unknown:" + out)
}

// refString and parseRef carry a wire point reference through the
// driver's string form, id@revision.
func refString(r wire.PointRef) string { return r.ID + "@" + strconv.Itoa(r.Revision) }

func parseRef(s string) wire.PointRef {
	i := strings.LastIndexByte(s, '@')
	if i < 0 {
		return wire.PointRef{ID: s}
	}
	n, err := strconv.Atoi(s[i+1:])
	if err != nil {
		n = 0
	}
	return wire.PointRef{ID: s[:i], Revision: n}
}

// snapshotOf reads the driver's snapshot off a frame: what a client sees
// is exactly what its frames say.
func snapshotOf(st wire.TableState) tabletest.Snapshot {
	sn := tabletest.Snapshot{Required: []int{}, Confirmed: []int{}, Legal: []string{}, Seat: -1}
	if st.You.Seat != nil {
		sn.Seat = *st.You.Seat
	}
	if st.Point == nil {
		sn.Over = st.Status == "finished"
		sn.Ref = "over"
		return sn
	}
	sn.Ref, sn.Point = refString(st.Point.Ref), st.Point.Ref.ID
	sn.Required = append(sn.Required, st.Point.Required...)
	sn.Confirmed = append(sn.Confirmed, st.Point.Confirmed...)
	required := false
	for _, s := range sn.Required {
		required = required || s == sn.Seat
	}
	if required && st.View.You != nil {
		for _, ch := range st.View.You.Choices {
			sn.Legal = append(sn.Legal, ch.Key)
		}
	}
	if st.Pending != nil {
		sn.Pick, sn.Confirm = st.Pending.Key, st.Pending.Confirm
	}
	return sn
}

// factory is a tabletest.Factory of in-process tables of one setup: n
// human seats, the seed, and the prefix resolved. Each new table first
// lets the previous one go, so thousands fit under the Manager's limit.
func (f *fixture) factory(n int, seed uint64, hideAtEnd bool) tabletest.Factory {
	var prev *inproc
	return func(ctx context.Context, prefix []tabletest.Batch) (tabletest.Table, error) {
		if prev != nil {
			prev.leave()
		}
		sp := humans(n, seed)
		sp.HideAtEnd = hideAtEnd
		for _, b := range prefix {
			br := BatchRecord{Point: b.Point}
			for _, pk := range b.Picks {
				br.Picks = append(br.Picks, PickRecord{Seat: pk.Seat, Key: pk.Key})
			}
			sp.Prefix = append(sp.Prefix, br)
		}
		p, err := f.openInproc(ctx, sp)
		prev = p
		return p, err
	}
}

func testContext(t testing.TB) context.Context {
	return testContextFor(t, wait)
}

func testContextFor(t testing.TB, d time.Duration) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	t.Cleanup(cancel)
	return ctx
}
