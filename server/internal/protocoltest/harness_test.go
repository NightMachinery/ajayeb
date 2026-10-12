package protocoltest

import (
	"context"
	"net/http/httptest"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/httpapi"
	"github.com/NightMachinery/ajayeb/server/internal/ident"
	"github.com/NightMachinery/ajayeb/server/internal/room"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/ws"
)

// wait is a safety net for a hung test, never a timing assumption: every
// helper that waits returns as soon as what it waits for has happened.
const wait = 20 * time.Second

// tinyBudget makes a bot's search deterministic and essentially free: one
// iteration, no wall-clock limit. It governs correctness, not strength.
var tinyBudget = ai.Budget{MaxIterations: 1}

// server is the full stack this package drives over real WebSocket
// connections: store, room manager, ws server and the httpapi handler that
// mounts it, behind an httptest.Server. Build one with newServer; stop it
// with close (done automatically through t.Cleanup unless the test calls
// close itself, as the restart test does to rebuild on the same data
// directory).
type server struct {
	t       *testing.T
	dataDir string
	store   *store.Store
	rooms   *room.Manager
	sockets *ws.Server
	hs      *httptest.Server
	wsURL   string
	seed    atomic.Uint64
}

// newServer boots a server whose rooms persist under dataDir (t.TempDir()
// for a fresh one, or an existing directory to resume on, as the restart
// test does). seedBase seeds its rooms deterministically and distinctly
// from any other server in the same test, so two servers in one test (the
// restart test) never hand out the same seed to two different rooms.
func newServer(t *testing.T, dataDir string, seedBase uint64) *server {
	t.Helper()
	baseline := runtime.NumGoroutine()
	t.Cleanup(func() { checkGoroutines(t, baseline) })
	st, err := store.Open(dataDir)
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	secret, err := ident.LoadSecret(dataDir)
	if err != nil {
		t.Fatalf("ident.LoadSecret: %v", err)
	}

	srv := &server{t: t, dataDir: dataDir, store: st}
	rooms, err := room.NewManager(room.Config{
		Store:     st,
		NewBot:    room.DefaultBotFactory,
		BotBudget: func(store.BotLevel) ai.Budget { return tinyBudget },
		BotDelay:  0,
		Seed:      func() uint64 { return seedBase + srv.seed.Add(1) },
		Claims:    secret,
	})
	if err != nil {
		t.Fatalf("room.NewManager: %v", err)
	}
	sockets, err := ws.New(ws.Config{
		Rooms:    rooms,
		Identity: secret,
		Version:  "protocoltest",
		// The default per-connection rate limit (10 frames/s, burst 20) is
		// sized for a human clicking; this package drives full games as
		// fast as the server accepts them, which is the point (the RAM
		// test plays 20 at once), and rate limiting itself is package ws's
		// own concern, not this package's.
		Limits: ws.Limits{FrameRate: 100000, FrameBurst: 100000},
	})
	if err != nil {
		rooms.Close()
		t.Fatalf("ws.New: %v", err)
	}
	handler, err := httpapi.New(httpapi.Config{Version: "protocoltest", WebSocket: sockets}, nil)
	if err != nil {
		rooms.Close()
		t.Fatalf("httpapi.New: %v", err)
	}
	hs := httptest.NewServer(handler)

	srv.rooms = rooms
	srv.sockets = sockets
	srv.hs = hs
	srv.wsURL = "ws" + hs.URL[len("http"):] + "/ws"
	t.Cleanup(srv.close)
	return srv
}

// close stops the server: refuses new sockets, closes every open one, and
// stops the room manager. It is idempotent (t.Cleanup calls it even after
// the restart test has already called it by hand).
func (s *server) close() {
	if s.hs == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	_ = s.sockets.Shutdown(ctx)
	s.hs.Close()
	s.rooms.Close()
	s.hs = nil
}

// checkGoroutines fails the test if more goroutines are running than before
// this server started, once close has had a chance to settle (a room's bot
// search, its persistence write and the ws server's writer goroutines all
// stop asynchronously).
func checkGoroutines(t *testing.T, baseline int) {
	t.Helper()
	deadline := time.Now().Add(wait)
	for runtime.NumGoroutine() > baseline {
		if time.Now().After(deadline) {
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, true)
			t.Errorf("%d goroutines still running, %d before this server started:\n%s", runtime.NumGoroutine(), baseline, buf[:n])
			return
		}
		runtime.Gosched()
	}
}

// snapshot returns the room's ground-truth state and action log, the way a
// development tool would: never through a client's own (redacted) view.
// It is used only to compute what must stay hidden (redact_test.go) and to
// check a replay's hash (restart_test.go), never to shortcut playing the
// game itself.
func (s *server) snapshot(code string) room.Snapshot {
	s.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	snap, err := s.rooms.Snapshot(ctx, code)
	if err != nil {
		s.t.Fatalf("Snapshot(%s): %v", code, err)
	}
	return snap
}
