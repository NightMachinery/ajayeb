package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

const (
	indexHTML = "<!doctype html><title>shell</title>"
	startJS   = "console.log('start')"
)

func testFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html":                           {Data: []byte(indexHTML)},
		"rules.html":                           {Data: []byte("<!doctype html><title>rules</title>")},
		"guide/index.html":                     {Data: []byte("<!doctype html><title>guide</title>")},
		"favicon.svg":                          {Data: []byte("<svg xmlns='http://www.w3.org/2000/svg'/>")},
		"art/cards/baths.avif":                 {Data: []byte("\x00\x00\x00\x1cftypavif")},
		"fonts/body.woff2":                     {Data: []byte("wOF2")},
		"_app/version.json":                    {Data: []byte(`{"version":"1"}`)},
		"_app/immutable/entry/start.Bx81.js":   {Data: []byte(startJS)},
		"_app/immutable/assets/app.Q2c9.css":   {Data: []byte("body{}")},
		".gitkeep":                             {Data: nil},
		".well-known/security.txt":             {Data: []byte("Contact: https://example.invalid\n")},
		"_app/immutable/chunks/nested/a.D1.js": {Data: []byte("export{}")},
	}
}

func newTestHandler(t *testing.T) http.Handler {
	t.Helper()
	h, err := New(Config{Version: "test-1"}, testFS())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return h
}

func do(t *testing.T, h http.Handler, method, target string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	req.RemoteAddr = "127.0.0.1:40000" // a local client, which the development endpoints require
	for k, vs := range header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestHealth(t *testing.T) {
	h := newTestHandler(t)

	rec := do(t, h, http.MethodGet, "/health", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", cc)
	}
	var body struct {
		OK      bool   `json:"ok"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %q: %v", rec.Body.String(), err)
	}
	if !body.OK || body.Version != "test-1" {
		t.Errorf("body = %+v, want ok=true version=test-1", body)
	}

	if rec := do(t, h, http.MethodHead, "/health", nil); rec.Code != http.StatusOK {
		t.Errorf("HEAD /health status = %d, want 200", rec.Code)
	}
}

func TestAppRoutesFallBackToIndex(t *testing.T) {
	h := newTestHandler(t)
	for _, target := range []string{"/", "/play/ABC", "/play/K7QX2M", "/play/ABC/", "/settings", "/index.html"} {
		t.Run(target, func(t *testing.T) {
			rec := do(t, h, http.MethodGet, target, nil)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			if got := rec.Body.String(); got != indexHTML {
				t.Errorf("body = %q, want the index.html shell", got)
			}
			if cc := rec.Header().Get("Cache-Control"); cc != "no-cache" {
				t.Errorf("Cache-Control = %q, want no-cache", cc)
			}
			if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
				t.Errorf("Content-Type = %q, want text/html", ct)
			}
		})
	}
}

func TestPrerenderedPages(t *testing.T) {
	h := newTestHandler(t)
	for target, want := range map[string]string{
		"/rules":  "<title>rules</title>",
		"/guide":  "<title>guide</title>",
		"/guide/": "<title>guide</title>",
	} {
		rec := do(t, h, http.MethodGet, target, nil)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), want) {
			t.Errorf("GET %s = %d %q, want 200 containing %q", target, rec.Code, rec.Body.String(), want)
		}
	}
}

func TestImmutableAssetsAreCachedForAYear(t *testing.T) {
	h := newTestHandler(t)
	for target, wantType := range map[string]string{
		"/_app/immutable/entry/start.Bx81.js":   "text/javascript; charset=utf-8",
		"/_app/immutable/assets/app.Q2c9.css":   "text/css; charset=utf-8",
		"/_app/immutable/chunks/nested/a.D1.js": "text/javascript; charset=utf-8",
	} {
		rec := do(t, h, http.MethodGet, target, nil)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s status = %d, want 200", target, rec.Code)
			continue
		}
		if cc := rec.Header().Get("Cache-Control"); cc != "public, max-age=31536000, immutable" {
			t.Errorf("GET %s Cache-Control = %q, want immutable", target, cc)
		}
		if ct := rec.Header().Get("Content-Type"); ct != wantType {
			t.Errorf("GET %s Content-Type = %q, want %q", target, ct, wantType)
		}
	}
}

func TestOtherFilesRevalidate(t *testing.T) {
	h := newTestHandler(t)
	for target, wantType := range map[string]string{
		"/_app/version.json":        "application/json",
		"/favicon.svg":              "image/svg+xml",
		"/art/cards/baths.avif":     "image/avif",
		"/fonts/body.woff2":         "font/woff2",
		"/.well-known/security.txt": "text/plain; charset=utf-8",
	} {
		rec := do(t, h, http.MethodGet, target, nil)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s status = %d, want 200", target, rec.Code)
			continue
		}
		if cc := rec.Header().Get("Cache-Control"); cc != "no-cache" {
			t.Errorf("GET %s Cache-Control = %q, want no-cache", target, cc)
		}
		if rec.Header().Get("ETag") == "" {
			t.Errorf("GET %s has no ETag", target)
		}
		if ct := rec.Header().Get("Content-Type"); ct != wantType {
			t.Errorf("GET %s Content-Type = %q, want %q", target, ct, wantType)
		}
	}
}

func TestMissingFilesAre404NotTheShell(t *testing.T) {
	h := newTestHandler(t)
	for _, target := range []string{
		"/missing.js",
		"/missing.css",
		"/art/cards/missing.avif",
		"/play/ABC/missing.png",
		"/_app/immutable/entry/missing.js",
		"/_app/immutable/nothing",
		"/_app",
		"/_app/version",
		"/.gitkeep",
		"/.env",
		"/play/.hidden",
		"/.well-known/missing",
	} {
		t.Run(target, func(t *testing.T) {
			rec := do(t, h, http.MethodGet, target, nil)
			if rec.Code != http.StatusNotFound {
				t.Fatalf("status = %d, want 404", rec.Code)
			}
			if strings.Contains(rec.Body.String(), "<title>shell</title>") {
				t.Errorf("404 body is the app shell")
			}
		})
	}
}

func TestETagRevalidation(t *testing.T) {
	h := newTestHandler(t)

	first := do(t, h, http.MethodGet, "/play/ABC", nil)
	tag := first.Header().Get("ETag")
	if tag == "" {
		t.Fatal("app shell has no ETag")
	}
	if other := do(t, h, http.MethodGet, "/", nil).Header().Get("ETag"); other != tag {
		t.Errorf("ETag for / = %q, want the same as /play/ABC (%q)", other, tag)
	}

	rec := do(t, h, http.MethodGet, "/play/XYZ", http.Header{"If-None-Match": {tag}})
	if rec.Code != http.StatusNotModified {
		t.Errorf("revalidation status = %d, want 304", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("304 has a body: %q", rec.Body.String())
	}
}

func TestStaticDirETagFollowsContent(t *testing.T) {
	dir := t.TempDir()
	index := filepath.Join(dir, "index.html")
	if err := os.WriteFile(index, []byte("version one"), 0o600); err != nil {
		t.Fatal(err)
	}
	h, err := New(Config{}, os.DirFS(dir))
	if err != nil {
		t.Fatal(err)
	}

	first := do(t, h, http.MethodGet, "/play/ABC", nil)
	if first.Body.String() != "version one" {
		t.Fatalf("body = %q, want version one", first.Body.String())
	}
	if err := os.WriteFile(index, []byte("version two!"), 0o600); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Minute)
	if err := os.Chtimes(index, later, later); err != nil {
		t.Fatal(err)
	}

	second := do(t, h, http.MethodGet, "/play/ABC", http.Header{"If-None-Match": {first.Header().Get("ETag")}})
	if second.Code != http.StatusOK || second.Body.String() != "version two!" {
		t.Fatalf("after rewrite: %d %q, want 200 with the new content", second.Code, second.Body.String())
	}
	if second.Header().Get("ETag") == first.Header().Get("ETag") {
		t.Error("ETag did not change after the file changed")
	}
}

func TestAPIUnknownIsJSON404(t *testing.T) {
	h := newTestHandler(t)
	for _, tc := range []struct{ method, target string }{
		{http.MethodGet, "/api"},
		{http.MethodGet, "/api/"},
		{http.MethodGet, "/api/nope"},
		{http.MethodGet, "/api/rooms/ABC/missing.js"},
		{http.MethodPost, "/api/rooms"},
	} {
		rec := do(t, h, tc.method, tc.target, nil)
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s %s status = %d, want 404", tc.method, tc.target, rec.Code)
			continue
		}
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("%s %s Content-Type = %q, want application/json", tc.method, tc.target, ct)
		}
		var body errorBody
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.OK || body.Code != "not-found" {
			t.Errorf("%s %s body = %q (err %v), want ok=false code=not-found", tc.method, tc.target, rec.Body.String(), err)
		}
	}
}

func TestWSIsReserved(t *testing.T) {
	h := newTestHandler(t)
	rec := do(t, h, http.MethodGet, "/ws", nil)
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501", rec.Code)
	}
	var body errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body.Code != "not-implemented" {
		t.Errorf("body = %q (err %v), want code=not-implemented", rec.Body.String(), err)
	}
}

func TestNonGetMethodsOnStaticPaths(t *testing.T) {
	h := newTestHandler(t)
	for _, target := range []string{"/", "/play/ABC", "/health", "/favicon.svg"} {
		rec := do(t, h, http.MethodPost, target, nil)
		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("POST %s status = %d, want 405", target, rec.Code)
		}
	}
	if rec := do(t, h, http.MethodHead, "/play/ABC", nil); rec.Code != http.StatusOK || rec.Body.Len() != 0 {
		t.Errorf("HEAD /play/ABC = %d with %d body bytes, want 200 and no body", rec.Code, rec.Body.Len())
	}
}

func TestCommonHeaders(t *testing.T) {
	h := newTestHandler(t)
	for _, target := range []string{"/health", "/play/ABC", "/missing.js", "/api/x"} {
		rec := do(t, h, http.MethodGet, target, nil)
		if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
			t.Errorf("GET %s X-Content-Type-Options = %q, want nosniff", target, got)
		}
	}
}

func TestNilStaticFS(t *testing.T) {
	h, err := New(Config{Version: "v"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if rec := do(t, h, http.MethodGet, "/health", nil); rec.Code != http.StatusOK {
		t.Errorf("/health status = %d, want 200", rec.Code)
	}
	if rec := do(t, h, http.MethodGet, "/play/ABC", nil); rec.Code != http.StatusNotFound {
		t.Errorf("/play/ABC status = %d, want 404", rec.Code)
	}
}

func TestDevEndpointsRequireDev(t *testing.T) {
	rooms := &fakeRooms{}
	if _, err := New(Config{DevEndpoints: true, DevRooms: rooms}, nil); !errors.Is(err, ErrDevEndpointsWithoutDev) {
		t.Errorf("New(DevEndpoints without Dev) error = %v, want ErrDevEndpointsWithoutDev", err)
	}
	if _, err := New(Config{Dev: true, DevEndpoints: true}, nil); !errors.Is(err, ErrDevEndpointsWithoutRooms) {
		t.Errorf("New(DevEndpoints without DevRooms) error = %v, want ErrDevEndpointsWithoutRooms", err)
	}
	if _, err := New(Config{Dev: true, DevEndpoints: true, DevRooms: rooms}, nil); err != nil {
		t.Errorf("New(Dev, DevEndpoints and DevRooms) error = %v, want nil", err)
	}
}

func TestWebSocketHandlerServesWS(t *testing.T) {
	ws := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	h, err := New(Config{WebSocket: ws}, testFS())
	if err != nil {
		t.Fatal(err)
	}
	if rec := do(t, h, http.MethodGet, "/ws", nil); rec.Code != http.StatusTeapot {
		t.Errorf("GET /ws status = %d, want the WebSocket handler's 418", rec.Code)
	}
	if rec := do(t, h, http.MethodGet, "/ws/other", nil); rec.Code == http.StatusTeapot {
		t.Error("/ws/other reached the WebSocket handler")
	}
}

func TestFileName(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
		ok   bool
	}{
		{"/", "index.html", true},
		{"", "index.html", true},
		{"/play/ABC", "play/ABC", true},
		{"/play/ABC/", "play/ABC", true},
		{"/a/../../etc/passwd", "etc/passwd", true},
		{"/../../index.html", "index.html", true},
		{"//double//slash", "double/slash", true},
		{"/.git/config", "", false},
		{"/x/.env", "", false},
		{"/.well-known/security.txt", ".well-known/security.txt", true},
	} {
		got, ok := fileName(tc.in)
		if got != tc.want || ok != tc.ok {
			t.Errorf("fileName(%q) = %q, %v; want %q, %v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}
