// Package httpapi assembles the server's HTTP surface: the health check, the
// JSON API under /api/, the WebSocket endpoint at /ws, and the single-page app
// (SPA) that serves the web client for every other path.
//
// This package routes requests and serves files. Game logic lives in the
// engine and room packages, and connection handling in package ws.
package httpapi

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"net/netip"
)

// Config holds the settings the HTTP layer needs. The zero value is a valid
// production configuration with no allowed cross-site origins.
type Config struct {
	// Version is the build version reported by /health.
	Version string

	// Dev enables development behaviour. The WebSocket layer then also
	// accepts loopback origins on any port, so the Vite dev server works.
	Dev bool

	// DevEndpoints enables the test-only endpoints under /api/dev/. It
	// requires Dev, and New refuses a Config that sets it without Dev.
	DevEndpoints bool

	// Origins lists the site origins (scheme://host[:port]) allowed to open
	// WebSockets, in addition to the request's own host.
	Origins []string

	// TrustedProxies lists the peers whose X-Forwarded-For header is believed
	// when finding a client's IP address.
	TrustedProxies []netip.Prefix
}

// ErrDevEndpointsWithoutDev is returned by New when DevEndpoints is set but
// Dev is not.
var ErrDevEndpointsWithoutDev = errors.New("httpapi: DevEndpoints requires Dev")

// New returns the root handler. static holds the web build (index.html at its
// root, SvelteKit assets under _app/). A nil static file system serves 404 for
// every non-API path.
func New(cfg Config, static fs.FS) (http.Handler, error) {
	if cfg.DevEndpoints && !cfg.Dev {
		return nil, ErrDevEndpointsWithoutDev
	}

	mux := http.NewServeMux()
	mux.Handle("GET /health", healthHandler(cfg.Version))
	mux.HandleFunc("/api", apiNotFound)
	mux.HandleFunc("/api/", apiNotFound)
	mux.HandleFunc("/ws", wsNotImplemented)
	mux.Handle("/", newSPA(static))
	return withCommonHeaders(mux), nil
}

// withCommonHeaders sets headers that every response should carry.
func withCommonHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "same-origin")
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

func apiNotFound(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusNotFound, errorBody{Code: "not-found", Msg: "no such API endpoint"})
}

// wsNotImplemented reserves /ws until package ws serves it.
func wsNotImplemented(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusNotImplemented, errorBody{Code: "not-implemented", Msg: "the WebSocket endpoint is not implemented yet"})
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
