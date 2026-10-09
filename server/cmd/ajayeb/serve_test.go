package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/NightMachinery/ajayeb/server/internal/ident"
	"github.com/NightMachinery/ajayeb/server/internal/store"
	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

// serveWait is a safety net for a hung test, not a timing assumption.
const serveWait = 20 * time.Second

func TestSeedSource(t *testing.T) {
	if seedSource(0) != nil {
		t.Error("seed base 0 should leave the random default")
	}
	next := seedSource(1000)
	for want := uint64(1000); want < 1003; want++ {
		if got := next(); got != want {
			t.Errorf("seed = %d, want %d", got, want)
		}
	}
}

// TestServeEndToEnd starts the real server on a free port, creates a room
// through the dev endpoint, plays a move over a WebSocket, and shuts down:
// the socket must close with 1012 and the move must be in the room file.
func TestServeEndToEnd(t *testing.T) {
	baseline := runtime.NumGoroutine()
	dataDir := t.TempDir()
	opts := options{
		addr:         "127.0.0.1:0",
		dataDir:      dataDir,
		dev:          true,
		devEndpoints: true,
		origins:      []string{},
		trustProxy:   []netip.Prefix{},
		botSearches:  1,
		seedBase:     500,
	}
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&lockedWriter{w: &logs}, &slog.HandlerOptions{Level: slog.LevelDebug}))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	addrCh := make(chan string, 1)
	done := make(chan error, 1)
	go func() { done <- serve(ctx, opts, logger, func(addr string) { addrCh <- addr }) }()
	var addr string
	select {
	case addr = <-addrCh:
	case err := <-done:
		t.Fatalf("serve returned early: %v", err)
	case <-time.After(serveWait):
		t.Fatal("the server did not start")
	}
	transport := &http.Transport{DisableKeepAlives: true}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: serveWait}

	// Health.
	resp, err := client.Get("http://" + addr + "/health")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/health = %d", resp.StatusCode)
	}

	// A dev room, seeded from the seed base.
	resp, err = client.Post("http://"+addr+"/api/dev/rooms", "application/json", strings.NewReader(`{"mode":"hotseat"}`))
	if err != nil {
		t.Fatal(err)
	}
	var created struct {
		Code string `json:"code"`
		Seed string `json:"seed"`
	}
	err = json.NewDecoder(resp.Body).Decode(&created)
	_ = resp.Body.Close()
	if err != nil || resp.StatusCode != http.StatusCreated || created.Seed != "500" {
		t.Fatalf("dev room: %d %+v (%v)", resp.StatusCode, created, err)
	}

	// Play one move over the socket.
	dialCtx, dialCancel := context.WithTimeout(context.Background(), serveWait)
	defer dialCancel()
	conn, _, err := websocket.Dial(dialCtx, "ws://"+addr+"/ws", &websocket.DialOptions{HTTPClient: client})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.CloseNow() }()
	conn.SetReadLimit(1 << 20)
	token, err := ident.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	send := func(v any) {
		t.Helper()
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		if err := conn.Write(dialCtx, websocket.MessageText, b); err != nil {
			t.Fatal(err)
		}
	}
	type stateFrame struct {
		T       string `json:"t"`
		Version int    `json:"version"`
		Ack     *struct {
			Seq int `json:"seq"`
		} `json:"ack"`
		View *struct {
			Choices []struct {
				Key string `json:"key"`
			} `json:"choices"`
		} `json:"view"`
	}
	read := func() stateFrame {
		t.Helper()
		_, b, err := conn.Read(dialCtx)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		var f stateFrame
		if err := json.Unmarshal(b, &f); err != nil {
			t.Fatal(err)
		}
		return f
	}
	send(wire.NewHello(wire.ProtocolVersion, token, "Sara", "tab-1"))
	if f := read(); f.T != "welcome" {
		t.Fatalf("first frame %q, want welcome", f.T)
	}
	send(wire.NewJoin(created.Code))
	st := read()
	if st.T != "state" || st.View == nil || len(st.View.Choices) == 0 {
		t.Fatalf("join answered %+v", st)
	}
	key := st.View.Choices[0].Key
	send(wire.NewAct(created.Code, 1, st.Version, key, nil))
	for {
		f := read()
		if f.T == "state" && f.Ack != nil && f.Ack.Seq == 1 {
			break
		}
	}

	// SIGTERM, as a cancelled context: the socket closes with 1012.
	cancel()
	for {
		if _, _, err := conn.Read(dialCtx); err != nil {
			if status := websocket.CloseStatus(err); status != websocket.StatusServiceRestart {
				t.Errorf("close status %d (%v), want 1012", status, err)
			}
			break
		}
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serve: %v", err)
		}
	case <-time.After(serveWait):
		t.Fatal("serve did not return after shutdown")
	}

	// The move is on disk, and so is the identity secret.
	st2, err := store.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	f, err := st2.PeekHeader(created.Code)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Actions) != 1 || f.Actions[0] != key || f.Seed != 500 {
		t.Errorf("room file: actions %q seed %d, want [%s] and 500", f.Actions, f.Seed, key)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "secret.key")); err != nil {
		t.Errorf("secret.key: %v", err)
	}
	if strings.Contains(logs.String(), token) {
		t.Error("the log contains the client's token")
	}

	_ = conn.CloseNow()
	transport.CloseIdleConnections()
	deadline := time.Now().Add(serveWait)
	for runtime.NumGoroutine() > baseline {
		if time.Now().After(deadline) {
			buf := make([]byte, 1<<16)
			n := runtime.Stack(buf, true)
			t.Fatalf("%d goroutines still running, %d before:\n%s", runtime.NumGoroutine(), baseline, buf[:n])
		}
		runtime.Gosched()
	}
}

func TestServeRefusesABadDataDir(t *testing.T) {
	file := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	opts := options{addr: "127.0.0.1:0", dataDir: file, botSearches: 1}
	err := serve(context.Background(), opts, slog.New(slog.NewTextHandler(io.Discard, nil)), nil)
	if err == nil {
		t.Fatal("serve with a file as its data directory succeeded")
	}
}

// lockedWriter serialises writes from the server's goroutines.
type lockedWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (l *lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}
