package httpapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"
)

// immutablePrefix is where SvelteKit writes content-hashed build assets. A
// file there never changes under the same name, so browsers may cache it for
// a year without revalidating.
const immutablePrefix = "_app/immutable/"

const (
	cacheImmutable  = "public, max-age=31536000, immutable"
	cacheRevalidate = "no-cache"
)

// contentTypes pins the media types of the files a web build contains, so the
// answer does not depend on the host's mime.types files. Other extensions fall
// back to mime.TypeByExtension and then to content sniffing.
var contentTypes = map[string]string{
	".avif":        "image/avif",
	".css":         "text/css; charset=utf-8",
	".html":        "text/html; charset=utf-8",
	".ico":         "image/vnd.microsoft.icon",
	".js":          "text/javascript; charset=utf-8",
	".json":        "application/json",
	".map":         "application/json",
	".mjs":         "text/javascript; charset=utf-8",
	".png":         "image/png",
	".svg":         "image/svg+xml",
	".txt":         "text/plain; charset=utf-8",
	".wasm":        "application/wasm",
	".webmanifest": "application/manifest+json",
	".webp":        "image/webp",
	".woff":        "font/woff",
	".woff2":       "font/woff2",
}

// spa serves a static web build as a single-page app.
//
// Resolution for a request path P (cleaned, without the leading slash):
//
//  1. P itself, if it is a regular file.
//  2. If P has no extension: P.html, then P/index.html (prerendered pages).
//  3. If P has no extension and is not under _app/ or .well-known/, the app
//     shell index.html, so client-side routes such as /play/K7QX2M work on a
//     reload or a shared link.
//  4. Otherwise 404. A missing /missing.js must not turn into an HTML page,
//     or the browser would report a confusing parse error instead of a 404.
//
// Files under _app/immutable/ are cached for a year. Everything else, the app
// shell included, carries Cache-Control: no-cache and a content-hash ETag, so
// browsers revalidate cheaply and pick up a new deployment immediately.
type spa struct {
	fsys fs.FS

	mu    sync.Mutex
	etags map[string]etagEntry
}

// etagEntry caches a file's ETag for as long as its size and modification
// time stay the same. Embedded files have a zero time and never change; files
// under --static change their time when rewritten.
type etagEntry struct {
	size int64
	mod  time.Time
	tag  string
}

func newSPA(fsys fs.FS) *spa {
	return &spa{fsys: fsys, etags: make(map[string]etagEntry)}
}

func (s *spa) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name, ok := fileName(r.URL.Path)
	if !ok || s.fsys == nil {
		notFound(w)
		return
	}
	for _, candidate := range candidates(name) {
		if s.serveFile(w, r, candidate) {
			return
		}
	}
	if isAppRoute(name) && s.serveFile(w, r, "index.html") {
		return
	}
	notFound(w)
}

// fileName maps a URL path to a file name inside the build. It rejects names
// with a segment that starts with a dot (except .well-known), so files such
// as .gitkeep or .env are never served.
func fileName(urlPath string) (string, bool) {
	name := strings.TrimPrefix(path.Clean("/"+urlPath), "/")
	if name == "" {
		return "index.html", true
	}
	for seg := range strings.SplitSeq(name, "/") {
		if strings.HasPrefix(seg, ".") && seg != ".well-known" {
			return "", false
		}
	}
	return name, fs.ValidPath(name)
}

// candidates lists the files that may answer a request for name, in order.
func candidates(name string) []string {
	if path.Ext(name) != "" {
		return []string{name}
	}
	return []string{name, name + ".html", name + "/index.html"}
}

// isAppRoute reports whether a missing name should get the app shell: a path
// without an extension that is not reserved for build assets or well-known
// files.
func isAppRoute(name string) bool {
	if path.Ext(name) != "" {
		return false
	}
	for _, reserved := range []string{"_app", ".well-known"} {
		if name == reserved || strings.HasPrefix(name, reserved+"/") {
			return false
		}
	}
	return true
}

// serveFile writes name if it is a regular file and reports whether it
// answered the request (including with an error status).
func (s *spa) serveFile(w http.ResponseWriter, r *http.Request, name string) bool {
	f, err := s.fsys.Open(name)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return false
	}

	content, ok := f.(io.ReadSeeker)
	if !ok {
		b, err := io.ReadAll(f)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return true
		}
		content = bytes.NewReader(b)
	}

	h := w.Header()
	if ct := contentType(name); ct != "" {
		h.Set("Content-Type", ct)
	}
	if strings.HasPrefix(name, immutablePrefix) {
		h.Set("Cache-Control", cacheImmutable)
	} else {
		tag, err := s.etag(name, info, content)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return true
		}
		h.Set("Cache-Control", cacheRevalidate)
		h.Set("ETag", tag)
	}
	// ServeContent handles HEAD, Range, If-None-Match and If-Modified-Since.
	http.ServeContent(w, r, name, info.ModTime(), content)
	return true
}

// etag returns a strong ETag derived from the file's SHA-256, reusing the
// cached value while the size and modification time are unchanged. It leaves
// content positioned at the start.
func (s *spa) etag(name string, info fs.FileInfo, content io.ReadSeeker) (string, error) {
	s.mu.Lock()
	cached, ok := s.etags[name]
	s.mu.Unlock()
	if ok && cached.size == info.Size() && cached.mod.Equal(info.ModTime()) {
		return cached.tag, nil
	}

	hash := sha256.New()
	if _, err := io.Copy(hash, content); err != nil {
		return "", err
	}
	if _, err := content.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	tag := `"` + base64.RawURLEncoding.EncodeToString(hash.Sum(nil)[:16]) + `"`

	s.mu.Lock()
	s.etags[name] = etagEntry{size: info.Size(), mod: info.ModTime(), tag: tag}
	s.mu.Unlock()
	return tag, nil
}

func contentType(name string) string {
	ext := strings.ToLower(path.Ext(name))
	if ct, ok := contentTypes[ext]; ok {
		return ct
	}
	return mime.TypeByExtension(ext)
}

func notFound(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", cacheRevalidate)
	http.Error(w, "404 page not found", http.StatusNotFound)
}
