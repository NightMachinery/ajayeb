package tableroom_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/NightMachinery/ajayeb/server/internal/httpapi"
	"github.com/NightMachinery/ajayeb/server/internal/ident"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/tableroom"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest"
	"github.com/NightMachinery/ajayeb/server/internal/tabletest/leakcheck"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
	"github.com/NightMachinery/ajayeb/server/internal/ws"
)

// These tests drive tables over the real stack, wired as cmd/ajayeb wires
// it: the schema-2 store, the room manager with both kinds, package ws on
// /ws and package httpapi for the log download, behind an httptest.Server,
// through real WebSocket connections. The tabletest.Table adapter sees
// only frames: a snapshot is the client's latest frame after a resync, and
// the canonical history is the ajayeb-log/2 download.

const wsWait = 30 * time.Second

type stack struct {
	t       *testing.T
	dir     string
	secret  ident.Secret
	st      *store.Store
	m       *room.Manager
	tables  *tableroom.Tables
	sockets *ws.Server
	hs      *httptest.Server
	url     string
	logs    *lockedBuffer
}

type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

func newStack(t *testing.T, dir string) *stack {
	t.Helper()
	baseline := runtime.NumGoroutine()
	s := &stack{t: t, dir: dir, logs: &lockedBuffer{}}
	logger := slog.New(slog.NewJSONHandler(s.logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	s.tables = tableroom.New(tableroom.Config{Logger: logger, Creation: true})
	kinds, err := room.NewKinds(room.DuelKind(), s.tables.Kind())
	if err != nil {
		t.Fatal(err)
	}
	if s.st, err = store.OpenWithCodecs(dir, nil, kinds.Codecs()); err != nil {
		t.Fatal(err)
	}
	if s.secret, err = ident.LoadSecret(dir); err != nil {
		t.Fatal(err)
	}
	s.m, err = room.NewManager(room.Config{Store: s.st, Kinds: kinds, KindOf: s.st.Game, Logger: logger, Claims: s.secret})
	if err != nil {
		t.Fatal(err)
	}
	s.tables.Bind(s.m, s.st)
	s.sockets, err = ws.New(ws.Config{Rooms: s.m, Identity: s.secret, Version: "tables", Logger: logger,
		Limits: ws.Limits{FrameRate: 100000, FrameBurst: 100000, MaxConns: 4096, MaxConnsPerIP: 4096, JoinMissesPerIP: 4096}})
	if err != nil {
		t.Fatal(err)
	}
	// Every schedule downloads its log, more often than the per-address
	// limit allows in real time; each reading of this clock is one refill
	// interval later, so the limiter never binds.
	var ticks atomic.Int64
	clock := func() time.Time {
		return time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(ticks.Add(1)) * 2 * time.Minute)
	}
	handler, err := httpapi.New(httpapi.Config{Version: "tables", WebSocket: s.sockets, Logs: s.st, Tables: s.tables, Clock: clock}, nil)
	if err != nil {
		t.Fatal(err)
	}
	s.hs = httptest.NewServer(handler)
	s.url = "ws" + strings.TrimPrefix(s.hs.URL, "http") + "/ws"
	t.Cleanup(func() {
		s.close()
		checkGoroutines(t, baseline)
	})
	return s
}

func (s *stack) close() {
	if s.hs == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), wsWait)
	defer cancel()
	_ = s.sockets.Shutdown(ctx)
	s.hs.Close()
	s.m.Close()
	s.hs = nil
}

// checkGoroutines: once the stack is closed, nothing it started still runs.
func checkGoroutines(t *testing.T, baseline int) {
	t.Helper()
	deadline := time.Now().Add(wsWait)
	for runtime.NumGoroutine() > baseline {
		if time.Now().After(deadline) {
			buf := make([]byte, 1<<16)
			t.Errorf("%d goroutines still running, %d before:\n%s", runtime.NumGoroutine(), baseline, buf[:runtime.Stack(buf, true)])
			return
		}
		time.Sleep(time.Millisecond)
	}
}

// --- a WebSocket client ----------------------------------------------------

type client struct {
	t        *testing.T
	conn     *websocket.Conn
	token    string
	clientID string
	user     string
	name     string
	in       chan []byte
	states   []wire.TableState
	raw      [][]byte // every table-state frame, as received
	rejects  []wire.TableReject
	other    []string
}

func (s *stack) dial(ctx context.Context, token, clientID, name string) *client {
	s.t.Helper()
	conn, _, err := websocket.Dial(ctx, s.url, nil)
	if err != nil {
		s.t.Fatal(err)
	}
	conn.SetReadLimit(4 << 20)
	user, err := s.secret.UserID(token)
	if err != nil {
		s.t.Fatal(err)
	}
	c := &client{t: s.t, conn: conn, token: token, clientID: clientID, user: user, name: name, in: make(chan []byte, 1<<14)}
	go func() {
		defer close(c.in)
		for {
			_, data, err := conn.Read(context.Background())
			if err != nil {
				return
			}
			c.in <- data
		}
	}()
	s.t.Cleanup(func() { _ = conn.CloseNow() })
	c.send(ctx, wire.NewHello(wire.PlatformProtocolVersion, token, name, clientID))
	if got := c.next(ctx); got != "welcome" {
		s.t.Fatalf("hello answered %s", got)
	}
	return c
}

func (c *client) send(ctx context.Context, v any) {
	c.t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		c.t.Fatal(err)
	}
	if err := c.conn.Write(ctx, websocket.MessageText, data); err != nil {
		c.t.Fatal(err)
	}
}

// next reads one frame, strictly decoded into the real wire type, and
// returns its type.
func (c *client) next(ctx context.Context) string {
	c.t.Helper()
	var data []byte
	select {
	case d, ok := <-c.in:
		if !ok {
			c.t.Fatal("the connection closed")
		}
		data = d
	case <-ctx.Done():
		c.t.Fatal("no frame came")
	}
	var head struct {
		T string `json:"t"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		c.t.Fatal(err)
	}
	strict := func(v any) {
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		if err := dec.Decode(v); err != nil {
			c.t.Fatalf("%s: %v: %s", head.T, err, data)
		}
	}
	switch head.T {
	case "table-state":
		var st wire.TableState
		strict(&st)
		c.states = append(c.states, st)
		c.raw = append(c.raw, data)
	case "table-reject":
		var r wire.TableReject
		strict(&r)
		c.rejects = append(c.rejects, r)
	case "welcome":
	default:
		c.other = append(c.other, string(data))
	}
	return head.T
}

// untilState reads until a table-state frame satisfies ok.
func (c *client) untilState(ctx context.Context, what string, ok func(wire.TableState) bool) wire.TableState {
	c.t.Helper()
	for {
		if c.next(ctx) == "table-state" && ok(c.states[len(c.states)-1]) {
			return c.states[len(c.states)-1]
		}
		if len(c.other) > 0 {
			c.t.Fatalf("waiting for %s: %s", what, c.other[len(c.other)-1])
		}
	}
}

func (c *client) join(ctx context.Context, code string) wire.TableState {
	c.send(ctx, wire.NewJoin(code))
	return c.untilState(ctx, "the join frame", func(st wire.TableState) bool { return st.History != nil })
}

// --- tabletest.Table over the socket ----------------------------------------

type wsTable struct {
	s       *stack
	code    string
	clients map[tabletest.ClientID]*client
	acked   map[tabletest.ClientID]uint64
}

// tokens are the stack's users: seat s's is tokens[s], the spectator's the
// last.
func newTokens(t *testing.T, n int) []string {
	out := make([]string, n)
	for i := range out {
		tok, err := ident.NewToken()
		if err != nil {
			t.Fatal(err)
		}
		out[i] = tok
	}
	return out
}

// open creates a table of n humans (chosen seating, so lobby slot s is
// seat s) and joins each seat's client and a spectator.
func (s *stack) open(ctx context.Context, tokens []string, n int, seed uint64, prefix []tabletest.Batch) (*wsTable, error) {
	sp := tableroom.Spec{Seed: &seed}
	for i := range n {
		user, _ := s.secret.UserID(tokens[i])
		sp.Participants = append(sp.Participants, tableroom.Participant{UserID: user, Name: fmt.Sprintf("P%d", i)})
	}
	sp.Creator = sp.Participants[0].UserID
	for _, b := range prefix {
		br := tableroom.BatchRecord{Point: b.Point}
		for _, pk := range b.Picks {
			br.Picks = append(br.Picks, tableroom.PickRecord{Seat: pk.Seat, Key: pk.Key})
		}
		sp.Prefix = append(sp.Prefix, br)
	}
	code, err := s.tables.CreateTable(sp)
	if err != nil {
		return nil, err
	}
	w := &wsTable{s: s, code: code, clients: map[tabletest.ClientID]*client{}, acked: map[tabletest.ClientID]uint64{}}
	for i := range n {
		id := tabletest.SeatClient(i)
		w.clients[id] = s.dial(ctx, tokens[i], string(id), fmt.Sprintf("P%d", i))
		w.clients[id].join(ctx, code)
	}
	w.clients[tabletest.SpectatorClient] = s.dial(ctx, tokens[len(tokens)-1], "spectator", "Watcher")
	w.clients[tabletest.SpectatorClient].join(ctx, code)
	return w, nil
}

func (w *wsTable) closeAll() {
	for _, c := range w.clients {
		_ = c.conn.Close(websocket.StatusNormalClosure, "")
	}
}

// Snapshot resyncs: the resync's answer, the next frame with history, is
// sent after everything this table did before it, so it is up to date.
// Nothing else in a driver run sends history (no joins, no undo).
func (w *wsTable) Snapshot(ctx context.Context, id tabletest.ClientID) (tabletest.Snapshot, error) {
	c := w.clients[id]
	c.send(ctx, wire.NewResync(w.code))
	st := c.untilState(ctx, "the resync", func(st wire.TableState) bool { return st.History != nil })
	return tableroom.SnapshotOf(st), nil
}

// Submit sends a table-pick and reads its answer: a table-reject with its
// seq, or a frame acknowledging it. Only an accepted submission records a
// seq, so an acknowledged seq at or below the client's highest accepted
// one is a duplicate (exact while the server's seq memory holds every
// client, as here); an accepted one resolved the point when its frame's
// reference is no longer the submitted one.
func (w *wsTable) Submit(ctx context.Context, id tabletest.ClientID, s tabletest.Submission) (tabletest.Outcome, error) {
	c := w.clients[id]
	c.send(ctx, wire.TablePick{T: "table-pick", Code: w.code, Seq: int(s.Seq), Point: tableroom.ParseRef(s.Ref), Key: s.Key, Confirm: s.Confirm})
	for {
		switch c.next(ctx) {
		case "table-reject":
			r := c.rejects[len(c.rejects)-1]
			if r.Seq != int(s.Seq) {
				return "", fmt.Errorf("a reject of seq %d while waiting for %d", r.Seq, s.Seq)
			}
			switch r.Code {
			case "decision-closed":
				return tabletest.Closed, nil
			case "illegal":
				return tabletest.Illegal, nil
			case "stale":
				return tabletest.Stale, nil
			case "forbidden":
				return tabletest.Forbidden, nil
			case "storage":
				return tabletest.StorageError, nil
			}
			return "", fmt.Errorf("reject %q", r.Code)
		case "table-state":
			st := c.states[len(c.states)-1]
			if st.Ack == nil || st.Ack.ClientID != c.clientID || st.Ack.Seq != int(s.Seq) {
				continue
			}
			if s.Seq <= w.acked[id] {
				return tabletest.Duplicate, nil
			}
			w.acked[id] = s.Seq
			if st.Point == nil || tableroom.RefString(st.Point.Ref) != s.Ref {
				return tabletest.Resolved, nil
			}
			return tabletest.Accepted, nil
		default:
			return "", fmt.Errorf("unexpected frame %s", c.other[len(c.other)-1])
		}
	}
}

// Canonical is the end-of-game log download.
func (w *wsTable) Canonical(ctx context.Context) (tabletest.Canonical, error) {
	l, status, err := w.s.download(ctx, w.code)
	if err != nil {
		return tabletest.Canonical{}, err
	}
	if status != http.StatusOK {
		return tabletest.Canonical{}, fmt.Errorf("the log download answered %d", status)
	}
	can := tabletest.Canonical{Hash: l.Hash, Batches: []tabletest.Batch{}}
	for _, b := range l.Batches {
		tb := tabletest.Batch{Point: b.Point, Picks: []tabletest.Pick{}}
		for _, pk := range b.Picks {
			tb.Picks = append(tb.Picks, tabletest.Pick{Seat: pk.Seat, Key: pk.Key})
		}
		can.Batches = append(can.Batches, tb)
	}
	return can, nil
}

func (s *stack) download(ctx context.Context, code string) (*tableroom.Log, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.hs.URL+"/api/rooms/"+code+"/log", nil)
	if err != nil {
		return nil, 0, err
	}
	resp, err := s.hs.Client().Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil, resp.StatusCode, err
	}
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	var l tableroom.Log
	if err := dec.Decode(&l); err != nil {
		return nil, resp.StatusCode, err
	}
	return &l, resp.StatusCode, nil
}

// factory opens fresh tables of one setup, closing the previous one's
// connections.
func (s *stack) factory(tokens []string, n int, seed uint64) tabletest.Factory {
	var prev *wsTable
	return func(ctx context.Context, prefix []tabletest.Batch) (tabletest.Table, error) {
		if prev != nil {
			prev.closeAll()
		}
		w, err := s.open(ctx, tokens, n, seed, prefix)
		prev = w
		return w, err
	}
}

func wsContext(t *testing.T, d time.Duration) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	t.Cleanup(cancel)
	return ctx
}

func viewerOf(id tabletest.ClientID) int {
	if id == tabletest.SpectatorClient {
		return leakcheck.Spectator
	}
	var s int
	_, _ = fmt.Sscanf(string(id), "seat-%d", &s)
	return s
}

// TestWebSocketTables: at 3 to 7 seats plus a spectator, whole games with
// drafts, resends and races pass the driver over the real stack, every
// arrival seed gives the same canonical history as the download, and
// every frame every connection received, byte for byte, passes the leak
// oracle; the log names no hidden value.
func TestWebSocketTables(t *testing.T) {
	ctx := wsContext(t, 5*time.Minute)
	s := newStack(t, t.TempDir())
	var truths []*leakcheck.Truth
	var users []string
	for n := tableroom.MinSeats; n <= tableroom.MaxSeats; n++ {
		tokens := newTokens(t, n+1)
		for _, tok := range tokens {
			u, _ := s.secret.UserID(tok)
			users = append(users, u)
		}
		seed := uint64(900 + n)
		cfg := tabletest.Config{Seats: n, PickSeed: uint64(n), Script: tabletest.ScriptOptions{Drafts: 50, Resends: 30}}
		runs, err := tabletest.CompareSchedules(ctx, s.factory(tokens, n, seed), cfg, []uint64{1, 2})
		if err != nil {
			t.Fatalf("%d seats: %v", n, err)
		}
		race := cfg
		race.Script, race.ArrivalSeed = tabletest.ScriptOptions{Race: true, Resends: 30}, 3
		tb, err := s.open(ctx, tokens, n, seed+50, nil)
		if err != nil {
			t.Fatal(err)
		}
		run, err := tabletest.Play(ctx, tb, race)
		if err != nil {
			t.Fatalf("%d seats, races: %v", n, err)
		}
		for _, c := range tb.clients {
			c.send(ctx, wire.NewResync(tb.code))
			c.untilState(ctx, "the final resync", func(st wire.TableState) bool { return st.History != nil })
		}
		truth := tableroom.TruthFrom(t, run.Canonical, n, seed+50, false)
		truths = append(truths, truth)
		checker := leakcheck.New(truth, tableroom.FrameConfig)
		for id, c := range tb.clients {
			docs := tableroom.LeakDocs(t, fmt.Sprintf("%d seats %s", n, id), viewerOf(id), c.states, truth)
			for i, raw := range c.raw {
				docs = append(docs, leakcheck.Doc{Label: fmt.Sprintf("%d seats %s raw %d", n, id, i), Kind: leakcheck.Frame, Viewer: viewerOf(id), Data: raw})
			}
			if leaks := checker.Check(docs...); len(leaks) != 0 {
				t.Fatalf("%d seats, %s: %v", n, id, leaks[:min(5, len(leaks))])
			}
		}
		tb.closeAll()
		if len(runs[0].Canonical.Batches) == 0 {
			t.Fatalf("%d seats: no batches", n)
		}
	}
	// The transport logs a connection's user id, a public identity rather
	// than a game secret; every other hex token still fails.
	text := s.logs.String()
	for _, u := range users {
		text = strings.ReplaceAll(text, u, "user-id")
	}
	logs := []byte(text)
	for _, truth := range truths {
		if leaks := leakcheck.New(truth, leakcheck.DefaultConfig).Check(leakcheck.Doc{Label: "log", Kind: leakcheck.Log, Data: logs}); len(leaks) != 0 {
			t.Fatalf("the log: %v", leaks[:min(5, len(leaks))])
		}
	}
}

// TestWebSocketEveryArrivalOrder: ExhaustPoint over the real stack on the
// first point of a three-seat table with drafts: all 90 orders give one
// batch.
func TestWebSocketEveryArrivalOrder(t *testing.T) {
	if testing.Short() {
		t.Skip("ninety tables over the socket")
	}
	ctx := wsContext(t, 10*time.Minute)
	s := newStack(t, t.TempDir())
	tokens := newTokens(t, 4)
	n, distinct, err := tabletest.ExhaustPoint(ctx, s.factory(tokens, 3, 1), nil,
		tabletest.Config{Seats: 3, PickSeed: 9, Script: tabletest.ScriptOptions{Drafts: 100}}, 1000)
	if err != nil || n < 90 || len(distinct) != 1 {
		t.Fatalf("%d orders, %d distinct batches: %v", n, len(distinct), err)
	}
}

// TestWebSocketRestart: a pick and its confirmation survive a server
// restart on the same data directory; the reference is the same, the
// running game's log stays refused, and play goes on to the end.
func TestWebSocketRestart(t *testing.T) {
	ctx := wsContext(t, 2*time.Minute)
	dir := t.TempDir()
	s := newStack(t, dir)
	tokens := newTokens(t, 4)
	tb, err := s.open(ctx, tokens, 3, 33, nil)
	if err != nil {
		t.Fatal(err)
	}
	sn, _ := tb.Snapshot(ctx, tabletest.SeatClient(1))
	if o, err := tb.Submit(ctx, tabletest.SeatClient(1), tabletest.Submission{Seq: 1, Ref: sn.Ref, Key: sn.Legal[0], Confirm: true}); err != nil || o != tabletest.Accepted {
		t.Fatalf("the pick: %s %v", o, err)
	}
	if _, status, _ := s.download(ctx, tb.code); status != http.StatusConflict {
		t.Fatalf("a running table's log: %d", status)
	}
	s.close()

	s2 := newStack(t, dir)
	again := reopen(ctx, s2, tokens, 3, tb.code)
	got, _ := again.Snapshot(ctx, tabletest.SeatClient(1))
	if got.Ref != sn.Ref || got.Pick != sn.Legal[0] || !got.Confirm || !slices.Equal(got.Confirmed, []int{1}) {
		t.Fatalf("after the restart: %+v", got)
	}
	finish(ctx, t, again, 3)
	l, status, err := s2.download(ctx, tb.code)
	if err != nil || status != http.StatusOK {
		t.Fatalf("the log: %d %v", status, err)
	}
	if b := l.Batches[0]; b.Picks[1] != (tableroom.PickRecord{Seat: 1, Key: sn.Legal[0]}) {
		t.Fatalf("the first batch lost seat 1's pick from before the restart: %+v", b)
	}
}

// reopen joins an existing table's seats and a spectator, each on a new
// client id, so the seats' seqs start again.
func reopen(ctx context.Context, s *stack, tokens []string, n int, code string) *wsTable {
	w := &wsTable{s: s, code: code, clients: map[tabletest.ClientID]*client{}, acked: map[tabletest.ClientID]uint64{}}
	for i := range n {
		id := tabletest.SeatClient(i)
		w.clients[id] = s.dial(ctx, tokens[i], string(id)+"-again", fmt.Sprintf("P%d", i))
		w.clients[id].join(ctx, code)
	}
	w.clients[tabletest.SpectatorClient] = s.dial(ctx, tokens[len(tokens)-1], "spectator-again", "Watcher")
	w.clients[tabletest.SpectatorClient].join(ctx, code)
	return w
}

// finish plays to the end: at each point every required seat that has not
// confirmed confirms its first offered key, and the last one resolves.
func finish(ctx context.Context, t *testing.T, w *wsTable, n int) {
	t.Helper()
	seqs := map[tabletest.ClientID]uint64{}
	for point := 0; ; point++ {
		if point > 100 {
			t.Fatal("the game did not end")
		}
		sn, _ := w.Snapshot(ctx, tabletest.SeatClient(0))
		if sn.Over {
			return
		}
		var todo []int
		for _, s := range sn.Required {
			if !slices.Contains(sn.Confirmed, s) {
				todo = append(todo, s)
			}
		}
		for i, s := range todo {
			id := tabletest.SeatClient(s)
			mine, _ := w.Snapshot(ctx, id)
			seqs[id]++
			o, err := w.Submit(ctx, id, tabletest.Submission{Seq: seqs[id], Ref: mine.Ref, Key: mine.Legal[0], Confirm: true})
			if want := map[bool]tabletest.Outcome{false: tabletest.Accepted, true: tabletest.Resolved}[i == len(todo)-1]; err != nil || o != want {
				t.Fatalf("point %s, seat %d: %s, want %s (%v)", sn.Point, s, o, want, err)
			}
		}
	}
}
