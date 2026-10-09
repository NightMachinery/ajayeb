package ws

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/ident"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/room/roomtest"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// wsWait is a safety net for a hung test, never a timing assumption: every
// wait returns as soon as what it waits for has happened.
const wsWait = 20 * time.Second

var wsEpoch = time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)

// harness is a Server over a real room.Manager (temporary store, fake
// clock) behind an httptest server.
type harness struct {
	t     *testing.T
	clock *roomtest.FakeClock
	rooms *room.Manager
	srv   *Server
	hs    *httptest.Server
	url   string
	logs  *lockedBuffer
	httpc *http.Client // no keep-alive, so refused upgrades leave no idle connection behind
}

// newHarness starts a server; opts adjust its config. Cleanup shuts it
// down and checks that no goroutine was left behind.
func newHarness(t *testing.T, opts ...func(*Config)) *harness {
	t.Helper()
	baseline := runtime.NumGoroutine()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	clock := roomtest.NewFakeClock(wsEpoch)
	var seed atomic.Uint64
	rooms, err := room.NewManager(room.Config{
		Store:  st,
		Clock:  clock,
		NewBot: func(store.BotLevel, uint64) room.Bot { return firstLegalBot{} },
		Seed:   func() uint64 { return 7000 + seed.Add(1) },
	})
	if err != nil {
		t.Fatal(err)
	}
	secret, err := ident.LoadSecret(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	logs := &lockedBuffer{}
	cfg := Config{
		Rooms:    rooms,
		Identity: secret,
		Version:  "test-1",
		Clock:    clock,
		Logger:   slog.New(slog.NewTextHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug})),
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	srv, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	hs := httptest.NewServer(srv)
	transport := &http.Transport{DisableKeepAlives: true}
	h := &harness{
		t:     t,
		clock: clock,
		rooms: rooms,
		srv:   srv,
		hs:    hs,
		url:   "ws" + strings.TrimPrefix(hs.URL, "http") + "/ws",
		logs:  logs,
		httpc: &http.Client{Transport: transport},
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), wsWait)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			t.Errorf("Shutdown: %v", err)
		}
		hs.Close()
		rooms.Close()
		transport.CloseIdleConnections()
		checkGoroutines(t, baseline)
	})
	return h
}

// checkGoroutines fails the test if more goroutines run than before it.
func checkGoroutines(t *testing.T, baseline int) {
	t.Helper()
	deadline := time.Now().Add(wsWait)
	for runtime.NumGoroutine() > baseline {
		if time.Now().After(deadline) {
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, true)
			t.Fatalf("%d goroutines still running, %d before the test:\n%s", runtime.NumGoroutine(), baseline, buf[:n])
		}
		runtime.Gosched()
	}
}

// waitUntil spins until cond holds, for conditions another goroutine brings
// about (the server's handler registering a socket, a pong arriving).
func waitUntil(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(wsWait)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		runtime.Gosched()
	}
}

// waitConns waits until the server has n open sockets.
func (h *harness) waitConns(n int) {
	h.t.Helper()
	waitUntil(h.t, "the open sockets to settle", func() bool { return h.srv.Conns() == n })
}

// conns returns the server's open connections.
func (h *harness) conns() []*conn {
	h.srv.mu.Lock()
	defer h.srv.mu.Unlock()
	out := make([]*conn, 0, len(h.srv.conns))
	for c := range h.srv.conns {
		out = append(out, c)
	}
	return out
}

// serverConn returns the server's connection for a client id.
func (h *harness) serverConn(clientID string) *conn {
	h.t.Helper()
	var found *conn
	waitUntil(h.t, "the server's connection for "+clientID, func() bool {
		for _, c := range h.conns() {
			c.mu.Lock()
			id := c.clientID
			c.mu.Unlock()
			if id == clientID {
				found = c
				return true
			}
		}
		return false
	})
	return found
}

// dial opens a socket with extra request headers.
func (h *harness) dial(header http.Header) (*websocket.Conn, *http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), wsWait)
	defer cancel()
	ws, resp, err := websocket.Dial(ctx, h.url, &websocket.DialOptions{HTTPHeader: header, HTTPClient: h.httpc})
	if ws != nil {
		ws.SetReadLimit(1 << 20)
	}
	return ws, resp, err
}

// mustDial dials and fails the test on error.
func (h *harness) mustDial(header http.Header) *websocket.Conn {
	h.t.Helper()
	ws, _, err := h.dial(header)
	if err != nil {
		h.t.Fatalf("dial: %v", err)
	}
	h.t.Cleanup(func() { _ = ws.CloseNow() })
	return ws
}

// dialStatus dials expecting the upgrade to be refused, and returns the
// HTTP status and the error code of the JSON body (if any).
func (h *harness) dialStatus(header http.Header) (int, string) {
	h.t.Helper()
	ws, resp, err := h.dial(header)
	if err == nil {
		_ = ws.CloseNow()
		h.t.Fatalf("dial succeeded, want it refused")
	}
	if resp == nil {
		h.t.Fatalf("dial failed without a response: %v", err)
	}
	var body httpError
	if resp.Body != nil {
		_ = json.NewDecoder(resp.Body).Decode(&body)
		_ = resp.Body.Close()
	}
	return resp.StatusCode, body.Code
}

// frame is a server frame as a test reads it: the fields of every frame
// type that the tests look at, plus the raw bytes.
type frame struct {
	raw     []byte
	T       string `json:"t"`
	Code    string `json:"code"` // a room code (created, state) or a reject or error code
	Seq     int    `json:"seq"`
	Msg     string `json:"msg"`
	V       int    `json:"v"`
	Version int    `json:"version"`
	CanUndo bool   `json:"canUndo"`
	Server  *struct {
		Version string `json:"version"`
		Rules   int    `json:"rules"`
	} `json:"server"`
	Ack *struct {
		ClientID string `json:"clientId"`
		Seq      int    `json:"seq"`
	} `json:"ack"`
	Room *struct {
		Mode       string `json:"mode"`
		Status     string `json:"status"`
		Spectators int    `json:"spectators"`
	} `json:"room"`
	You *struct {
		Seats  []int `json:"seats"`
		CanAct bool  `json:"canAct"`
	} `json:"you"`
	View *struct {
		Ply     int  `json:"ply"`
		ToMove  *int `json:"toMove"`
		Choices []struct {
			Key string `json:"key"`
		} `json:"choices"`
	} `json:"view"`
}

// tclient is a test connection whose frames a background goroutine reads,
// so it answers pings and closes the way a browser does.
type tclient struct {
	t        *testing.T
	h        *harness
	ws       *websocket.Conn
	frames   chan frame
	stop     chan struct{}
	done     chan struct{} // closed when the reader has stopped
	err      error         // why the reader stopped; read after done
	raw      *lockedBuffer // every frame received, for leak checks
	token    string
	clientID string
	code     string // the room last created or joined
	seq      int
}

var clientCounter atomic.Int64

// connect dials with extra headers and starts reading, without hello.
func (h *harness) connect(header http.Header) *tclient {
	h.t.Helper()
	ws := h.mustDial(header)
	token, err := ident.NewToken()
	if err != nil {
		h.t.Fatal(err)
	}
	c := &tclient{
		t:        h.t,
		h:        h,
		ws:       ws,
		frames:   make(chan frame, 256),
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
		raw:      &lockedBuffer{},
		token:    token,
		clientID: "tab-" + strconv.FormatInt(clientCounter.Add(1), 10),
	}
	go c.read()
	h.t.Cleanup(func() {
		close(c.stop)
		_ = ws.CloseNow()
		<-c.done
	})
	return c
}

// client connects and says hello.
func (h *harness) client() *tclient {
	h.t.Helper()
	return h.clientWith(nil)
}

// clientWith connects with extra headers and says hello.
func (h *harness) clientWith(header http.Header) *tclient {
	h.t.Helper()
	c := h.connect(header)
	c.hello()
	return c
}

func (c *tclient) read() {
	defer close(c.done)
	for {
		_, data, err := c.ws.Read(context.Background())
		if err != nil {
			c.err = err
			return
		}
		_, _ = c.raw.Write(data)
		var f frame
		if err := json.Unmarshal(data, &f); err != nil {
			c.err = err
			return
		}
		f.raw = data
		select {
		case c.frames <- f:
		case <-c.stop:
			return
		}
	}
}

// send writes one frame: a value to marshal, or raw bytes.
func (c *tclient) send(v any) {
	c.t.Helper()
	b, ok := v.([]byte)
	if !ok {
		var err error
		if b, err = json.Marshal(v); err != nil {
			c.t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), wsWait)
	defer cancel()
	if err := c.ws.Write(ctx, websocket.MessageText, b); err != nil {
		c.t.Fatalf("write: %v", err)
	}
}

// next returns the next frame.
func (c *tclient) next() frame {
	c.t.Helper()
	timer := time.NewTimer(wsWait)
	defer timer.Stop()
	select {
	case f := <-c.frames:
		return f
	case <-c.done:
		select {
		case f := <-c.frames:
			return f
		default:
		}
		c.t.Fatalf("client %s: the socket closed (%v) while a frame was expected", c.clientID, c.err)
	case <-timer.C:
		c.t.Fatalf("client %s: no frame arrived", c.clientID)
	}
	return frame{}
}

// expect returns the next frame, which must be of type typ.
func (c *tclient) expect(typ string) frame {
	c.t.Helper()
	f := c.next()
	if f.T != typ {
		c.t.Fatalf("client %s: want a %s frame, got %s", c.clientID, typ, f.raw)
	}
	return f
}

// until skips frames until one satisfies ok.
func (c *tclient) until(what string, ok func(frame) bool) frame {
	c.t.Helper()
	for {
		if f := c.next(); ok(f) {
			return f
		}
	}
}

// expectReject skips state frames until a reject, which must be for seq
// with code.
func (c *tclient) expectReject(seq int, code wire.RejectCode) frame {
	c.t.Helper()
	f := c.until("a reject", func(f frame) bool { return f.T != "state" })
	if f.T != "reject" || f.Seq != seq || f.Code != string(code) {
		c.t.Fatalf("client %s: want reject %s for seq %d, got %s", c.clientID, code, seq, f.raw)
	}
	return f
}

// closed waits for the socket to close, skipping frames, and returns the
// close status the server sent (-1 when it closed without one).
func (c *tclient) closed() websocket.StatusCode {
	c.t.Helper()
	timer := time.NewTimer(wsWait)
	defer timer.Stop()
	for {
		select {
		case <-c.frames:
		case <-c.done:
			return websocket.CloseStatus(c.err)
		case <-timer.C:
			c.t.Fatalf("client %s: the socket did not close", c.clientID)
		}
	}
}

// hello says hello and checks the welcome.
func (c *tclient) hello() {
	c.t.Helper()
	c.send(wire.NewHello(wire.ProtocolVersion, c.token, "Sara", c.clientID))
	w := c.expect("welcome")
	if w.V != wire.ProtocolVersion || w.Server == nil || w.Server.Version != "test-1" || w.Server.Rules != engine.RulesVersion {
		c.t.Fatalf("welcome = %s", w.raw)
	}
}

// create creates a room and returns its first state frame.
func (c *tclient) create(mode wire.Mode) frame {
	c.t.Helper()
	c.send(wire.NewCreate(mode, nil, wire.SeatNumber(0), []string{"Sara", "Omid"}))
	created := c.expect("created")
	c.code = created.Code
	return c.until("the new room's state", func(f frame) bool { return f.T == "state" && f.Code == created.Code })
}

// join joins a room and returns its state.
func (c *tclient) join(code string) frame {
	c.t.Helper()
	c.send(wire.NewJoin(code))
	c.code = code
	return c.until("the joined room's state", func(f frame) bool { return f.T == "state" && f.Code == code })
}

// act sends an act with the next seq and returns the seq.
func (c *tclient) act(key string, base int) int {
	c.t.Helper()
	c.seq++
	c.send(wire.NewAct(c.code, c.seq, base, key, nil))
	return c.seq
}

// lockedBuffer is a bytes.Buffer safe for concurrent writers.
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

// firstLegalBot plays the first legal action of its observation.
type firstLegalBot struct{}

func (firstLegalBot) Choose(_ context.Context, obs engine.Observation, _ ai.Budget, _ func(ai.Progress)) (engine.Action, error) {
	legal := engine.LegalActions(obs.Public(), nil)
	if len(legal) == 0 {
		return engine.Action{}, errors.New("no legal action")
	}
	return legal[0], nil
}
