package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
	"github.com/NightMachinery/ajayeb/server/internal/store"
)

// playOut plays seed from the deal to the end, always taking the first legal
// action, and returns the keys applied and the final state.
func playOut(t *testing.T, seed uint64, opts engine.Options) ([]string, engine.State) {
	t.Helper()
	s := engine.NewGame(seed, opts)
	var keys []string
	for !s.Over() {
		legal := engine.LegalActions(&s, nil)
		if len(legal) == 0 {
			t.Fatalf("seed %d: no legal action in a running game", seed)
		}
		if err := engine.Apply(&s, legal[0], nil); err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		keys = append(keys, legal[0].Key())
	}
	return keys, s
}

type logFixture struct {
	h     http.Handler
	st    *store.Store
	now   time.Time
	final engine.State
}

// newLogFixture serves a store holding a finished room (FNSHED), a running
// one (PRAYNG), an online lobby (WBBYXX), a finished game that hides its
// secrets at the end (HDDENX) and an archived room (ARCHVD).
func newLogFixture(t *testing.T, trusted ...netip.Prefix) *logFixture {
	t.Helper()
	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	fx := &logFixture{st: st, now: time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)}
	keys, final := playOut(t, 5, engine.Options{})
	fx.final = final
	hidden := engine.Options{HideAtEnd: true}
	hiddenKeys, _ := playOut(t, 6, hidden)
	rooms := []struct {
		code   string
		mode   store.Mode
		opts   engine.Options
		seed   uint64
		keys   []string
		status store.Status
	}{
		{"FNSHED", store.ModeHotseat, engine.Options{}, 5, keys, store.StatusFinished},
		{"PRAYNG", store.ModeAI, engine.Options{}, 5, keys[:4], store.StatusPlaying},
		{"WBBYXX", store.ModeOnline, engine.Options{}, 5, []string{}, store.StatusLobby},
		{"HDDENX", store.ModeHotseat, hidden, 6, hiddenKeys, store.StatusFinished},
		{"ARCHVD", store.ModeHotseat, engine.Options{}, 5, keys, store.StatusArchived},
	}
	for _, r := range rooms {
		f := store.New(r.code, r.mode, r.opts, r.seed, fx.now)
		f.Actions = r.keys
		f.Status = r.status
		if err := st.Save(f); err != nil {
			t.Fatalf("Save %s: %v", r.code, err)
		}
	}
	h, err := New(Config{Version: "test", Logs: st, TrustedProxies: trusted, Clock: func() time.Time { return fx.now }}, testFS())
	if err != nil {
		t.Fatal(err)
	}
	fx.h = h
	return fx
}

func errCode(t *testing.T, rec *httptest.ResponseRecorder) string {
	t.Helper()
	var body errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	if body.OK {
		t.Errorf("an error body says ok")
	}
	return body.Code
}

func TestGameLogFinished(t *testing.T) {
	fx := newLogFixture(t)
	// The suggested date is UTC even on a server whose clock uses another zone.
	fx.now = fx.now.In(time.FixedZone("download-zone", 14*60*60))
	for _, path := range []string{"/api/rooms/FNSHED/log", "/api/rooms/fnshed/log"} {
		rec := do(t, fx.h, http.MethodGet, path, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d, want 200: %s", path, rec.Code, rec.Body.String())
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type %q", ct)
		}
		if cd := rec.Header().Get("Content-Disposition"); cd != `attachment; filename="ajayeb-FNSHED-2026-10-01.json"` {
			t.Errorf("Content-Disposition %q", cd)
		}
		var l engine.Log
		if err := json.Unmarshal(rec.Body.Bytes(), &l); err != nil {
			t.Fatalf("decode the log: %v", err)
		}
		if l.Format != engine.LogFormat || l.Seed != 5 || l.FinalHash == "" {
			t.Errorf("log header: format %q seed %d finalHash %q", l.Format, l.Seed, l.FinalHash)
		}
		s, err := engine.Replay(&l)
		if err != nil {
			t.Fatalf("Replay: %v", err)
		}
		if engine.HashString(&s) != engine.HashString(&fx.final) || !s.Over() {
			t.Errorf("the downloaded log does not replay to the finished game")
		}
	}
}

func TestGameLogRefusals(t *testing.T) {
	fx := newLogFixture(t)
	cases := []struct {
		method, path string
		status       int
		code         string
	}{
		{http.MethodGet, "/api/rooms/NNNNNN/log", http.StatusNotFound, "not-found"},
		{http.MethodGet, "/api/rooms/bad!/log", http.StatusNotFound, "not-found"},
		{http.MethodGet, "/api/rooms/PRAYNG/log", http.StatusConflict, "game-running"},
		{http.MethodGet, "/api/rooms/WBBYXX/log", http.StatusConflict, "game-running"},
		{http.MethodGet, "/api/rooms/HDDENX/log", http.StatusForbidden, "log-hidden"},
		{http.MethodGet, "/api/rooms/ARCHVD/log", http.StatusGone, "archived"},
		{http.MethodPost, "/api/rooms/FNSHED/log", http.StatusMethodNotAllowed, "method-not-allowed"},
	}
	for _, c := range cases {
		rec := do(t, fx.h, c.method, c.path, nil)
		if rec.Code != c.status {
			t.Errorf("%s %s: status %d, want %d", c.method, c.path, rec.Code, c.status)
			continue
		}
		if got := errCode(t, rec); got != c.code {
			t.Errorf("%s %s: code %q, want %q", c.method, c.path, got, c.code)
		}
		if strings.Contains(rec.Body.String(), `"seed"`) {
			t.Errorf("%s %s: a refusal carries the seed", c.method, c.path)
		}
	}
}

func TestGameLogWithoutStoreIs404(t *testing.T) {
	rec := do(t, newTestHandler(t), http.MethodGet, "/api/rooms/FNSHED/log", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status %d, want 404", rec.Code)
	}
}

func TestGameLogRateLimit(t *testing.T) {
	fx := newLogFixture(t)
	get := func(remote string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, "/api/rooms/NNNNNN/log", nil)
		req.RemoteAddr = remote
		rec := httptest.NewRecorder()
		fx.h.ServeHTTP(rec, req)
		return rec
	}
	// Misses count: a burst, then 429.
	for i := range logBurst {
		if rec := get("192.0.2.1:1000"); rec.Code != http.StatusNotFound {
			t.Fatalf("request %d: status %d, want 404", i, rec.Code)
		}
	}
	rec := get("192.0.2.1:1001")
	if rec.Code != http.StatusTooManyRequests || errCode(t, rec) != "rate" || rec.Header().Get("Retry-After") == "" {
		t.Fatalf("over the burst: status %d, want 429 rate with Retry-After", rec.Code)
	}
	// Another address is unaffected; an IPv6 /64 counts as one address.
	if rec := get("192.0.2.2:1000"); rec.Code != http.StatusNotFound {
		t.Errorf("another address: status %d, want 404", rec.Code)
	}
	for i := range logBurst {
		if rec := get(fmt.Sprintf("[2001:db8::%x]:1000", i+1)); rec.Code != http.StatusNotFound {
			t.Fatalf("IPv6 request %d: status %d", i, rec.Code)
		}
	}
	if rec := get("[2001:db8::ff]:1000"); rec.Code != http.StatusTooManyRequests {
		t.Errorf("the same /64: status %d, want 429", rec.Code)
	}
	// One interval later one more request is allowed.
	fx.now = fx.now.Add(logInterval)
	if rec := get("192.0.2.1:1000"); rec.Code != http.StatusNotFound {
		t.Errorf("after an interval: status %d, want 404", rec.Code)
	}
	if rec := get("192.0.2.1:1000"); rec.Code != http.StatusTooManyRequests {
		t.Errorf("right after: status %d, want 429", rec.Code)
	}
}

func TestGameLogRateLimitTrustedProxy(t *testing.T) {
	fx := newLogFixture(t, netip.MustParsePrefix("10.0.0.1/32"))
	get := func(xff string) int {
		req := httptest.NewRequest(http.MethodGet, "/api/rooms/NNNNNN/log", nil)
		req.RemoteAddr = "10.0.0.1:5000"
		req.Header.Set("X-Forwarded-For", xff)
		rec := httptest.NewRecorder()
		fx.h.ServeHTTP(rec, req)
		return rec.Code
	}
	for range logBurst {
		if got := get("198.51.100.7"); got != http.StatusNotFound {
			t.Fatalf("status %d, want 404", got)
		}
	}
	if got := get("203.0.113.9, 198.51.100.7"); got != http.StatusTooManyRequests {
		t.Errorf("a forged left entry escaped the limit: status %d", got)
	}
	if got := get("198.51.100.8"); got != http.StatusNotFound {
		t.Errorf("another forwarded client: status %d, want 404", got)
	}
}

func TestGameLogLimiterCapacity(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	a := newLogAPI(nil, nil, func() time.Time { return now })
	for i := range logCapacity {
		key := netip.PrefixFrom(netip.AddrFrom4([4]byte{10, byte(i >> 16), byte(i >> 8), byte(i)}), 32)
		if !a.allow(key) {
			t.Fatalf("address %d refused", i)
		}
	}
	newcomer := netip.MustParsePrefix("192.0.2.1/32")
	if a.allow(newcomer) {
		t.Errorf("a newcomer was admitted while every remembered bucket is refilling")
	}
	now = now.Add(logInterval)
	if !a.allow(newcomer) {
		t.Errorf("a newcomer was refused after the buckets refilled")
	}
}
