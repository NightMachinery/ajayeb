// Package httpapi assembles the server's HTTP surface: the health check, the
// JSON API under /api/, the WebSocket endpoint at /ws, and the single-page app
// (SPA) that serves the web client for every other path.
//
// With Logs, it serves the replay download of a finished game at
// GET /api/rooms/{code}/log (gamelog.go).
//
// This package routes requests and serves files. Game logic lives in the
// engine and room packages, and connection handling in package ws, whose
// handler New mounts at /ws. With DevEndpoints (which requires Dev), it also
// serves the development endpoints under /api/dev/ (devrooms.go), which
// create rooms at chosen positions for end-to-end tests, to clients on the
// same machine only; a production server never enables them.
package httpapi

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/netip"
	"time"
)

// Config holds the settings the HTTP layer needs. The zero value is a valid
// production configuration whose /ws is not served yet (501).
type Config struct {
	// Version is the build version reported by /health.
	Version string

	// Dev marks a development server. It is required for DevEndpoints.
	Dev bool

	// DevEndpoints enables the test-only endpoints under /api/dev/. It
	// requires Dev and DevRooms, and New refuses a Config that sets it
	// without them.
	DevEndpoints bool

	// WebSocket serves /ws (a *ws.Server). Nil keeps the path reserved,
	// answering 501.
	WebSocket http.Handler

	// DevRooms creates the rooms of POST /api/dev/rooms (a *room.Manager).
	DevRooms DevRooms

	// Seed returns the seed of a development room whose request names
	// none. Default: crypto/rand.
	Seed func() uint64

	// Logs serves the replay download, GET /api/rooms/{code}/log (a
	// *store.Store). Nil leaves the path answering 404 like any unknown API
	// path.
	Logs GameLogs

	// TrustedProxies are the reverse proxies whose X-Forwarded-For the
	// per-address limit of the log download believes, as for /ws.
	TrustedProxies []netip.Prefix

	// Clock drives the log download's rate limit. Default: time.Now.
	Clock func() time.Time
}

// ErrDevEndpointsWithoutDev is returned by New when DevEndpoints is set but
// Dev is not.
var ErrDevEndpointsWithoutDev = errors.New("httpapi: DevEndpoints requires Dev")

// ErrDevEndpointsWithoutRooms is returned by New when DevEndpoints is set
// without DevRooms.
var ErrDevEndpointsWithoutRooms = errors.New("httpapi: DevEndpoints requires DevRooms")

// New returns the root handler. static holds the web build (index.html at its
// root, SvelteKit assets under _app/). A nil static file system serves 404 for
// every non-API path.
func New(cfg Config, static fs.FS) (http.Handler, error) {
	if cfg.DevEndpoints && !cfg.Dev {
		return nil, ErrDevEndpointsWithoutDev
	}
	if cfg.DevEndpoints && cfg.DevRooms == nil {
		return nil, ErrDevEndpointsWithoutRooms
	}

	mux := http.NewServeMux()
	mux.Handle("GET /health", healthHandler(cfg.Version))
	mux.HandleFunc("/api", apiNotFound)
	mux.HandleFunc("/api/", apiNotFound)
	if cfg.DevEndpoints {
		d := newDevAPI(cfg.DevRooms, cfg.Seed)
		mux.Handle("/api/dev/rooms", localOnly(http.HandlerFunc(d.createRoom)))
		mux.Handle("/api/dev/scenarios", localOnly(http.HandlerFunc(d.scenarios)))
		mux.Handle("/api/dev/tutorial", localOnly(http.HandlerFunc(d.tutorialScript)))
	}
	if cfg.Logs != nil {
		mux.HandleFunc("/api/rooms/{code}/log", newLogAPI(cfg.Logs, cfg.TrustedProxies, cfg.Clock).serve)
	}
	if cfg.WebSocket != nil {
		mux.Handle("/ws", cfg.WebSocket)
	} else {
		mux.HandleFunc("/ws", wsNotImplemented)
	}
	policy := newSecurityPolicy(cfg.Dev)
	site := newSPA(static)
	site.policy = policy
	mux.Handle("/", site)
	return withCommonHeaders(policy, mux), nil
}

// withCommonHeaders sets the headers that every response carries: the
// security headers of security.go, with the base content security policy
// (the static handler replaces it with the document's own for an HTML
// file).
func withCommonHeaders(policy securityPolicy, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		policy.setHeaders(w.Header())
		next.ServeHTTP(w, r)
	})
}

type healthBody struct {
	OK      bool   `json:"ok"`
	Version string `json:"version"`
}

func healthHandler(version string) http.Handler {
	body := healthBody{OK: true, Version: version}
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, body)
	})
}

// errorBody is the JSON shape of every HTTP API error. Code is a stable
// machine-readable identifier; Msg is a developer hint, not a display string.
type errorBody struct {
	OK   bool   `json:"ok"`
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, errorBody{Code: code, Msg: msg})
}

func apiNotFound(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotFound, "not-found", "no such API endpoint")
}

// wsNotImplemented reserves /ws for a server built without a WebSocket
// handler.
func wsNotImplemented(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusNotImplemented, "not-implemented", "this server has no WebSocket endpoint")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	h := w.Header()
	h.Set("Content-Type", "application/json")
	h.Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	// A write error means the client went away; there is nothing to do.
	_, _ = w.Write(append(b, '\n'))
}
