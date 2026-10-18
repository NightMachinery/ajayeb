package httpapi

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/webdist"
)

// minVersionLen is the shortest v query parameter that counts as a content
// hash. The generated art index writes 10 hex digits; anything shorter is too
// weak a claim to cache the answer for a year.
const minVersionLen = 8

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
//     shell.html (or index.html for legacy builds), so client-side routes such as /play/K7QX2M work on a
//     reload or a shared link.
//  4. Otherwise 404. A missing /missing.js must not turn into an HTML page,
//     or the browser would report a confusing parse error instead of a 404.
//
// Cache-Control comes from webdist.CacheControl. Files under _app/immutable/,
// and art under art/ requested with a v query parameter that matches the
// start of the file's SHA-256 in hex (the generated art index writes every
// art path that way, and the CSS names its textures with one too), are
// cached for a year with no ETag. Art without a matching v (an app icon
// named by static HTML or the web app manifest, or a stale v left over from
// an older deployment)
// gets the shorter art policy and a content-hash ETag, and everything else,
// the app shell included, carries Cache-Control: no-cache and the ETag, so
// browsers revalidate cheaply and pick up a new deployment immediately.
//
// An HTML file gets its own content security policy, with the hash of each
// of its inline scripts (security.go). The policy is computed with the
// file's digest and cached with it, so it follows the file's content.
type spa struct {
	fsys   fs.FS
	policy securityPolicy

	mu    sync.Mutex
	etags map[string]etagEntry
}

// etagEntry caches a file's ETag and content hash for as long as its size and
// modification time stay the same. Embedded files have a zero time and never
// change; files under --static change their time when rewritten.
type etagEntry struct {
	size          int64
	mod           time.Time
	tag           string
	hash          string // the SHA-256 in lowercase hex
	homeStyleLink string // Home-only critical stylesheet preload, bound to the HTML digest
	csp           string // an HTML file's content security policy; "" for other files
}

func newSPA(fsys fs.FS) *spa {
	return &spa{fsys: fsys, policy: newSecurityPolicy(false), etags: make(map[string]etagEntry)}
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
	if isAppRoute(name) {
		// index.html may be prerendered Home. Only the generic shell can
		// hydrate another route, including a shared game link.
		if s.serveFile(w, r, "shell.html") || s.serveFile(w, r, "index.html") {
			return
		}
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
	var sum etagEntry
	if html := isHTML(name); html || !strings.HasPrefix(name, immutablePrefix) {
		var err error
		if sum, err = s.digest(name, info, content, html); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return true
		}
		if html {
			h.Set(s.policy.header(), sum.csp)
			if sum.homeStyleLink != "" {
				h.Add("Link", sum.homeStyleLink)
			}
		}
	}
	cc := webdist.CacheControl(name, versionMatches(r.URL.Query().Get("v"), sum.hash))
	h.Set("Cache-Control", cc)
	if cc != webdist.CacheImmutable {
		h.Set("ETag", sum.tag)
	}
	// ServeContent handles HEAD, Range, If-None-Match and If-Modified-Since.
	http.ServeContent(w, r, name, info.ModTime(), content)
	return true
}

// immutablePrefix is where SvelteKit writes content-hashed build assets;
// webdist.CacheControl gives them a year, so they need no ETag or hash.
const immutablePrefix = "_app/immutable/"

// versionMatches reports whether a v query parameter names the file whose
// SHA-256 is hash (lowercase hex): it must be a hex prefix of it, at least
// minVersionLen digits long.
func versionMatches(v, hash string) bool {
	return len(v) >= minVersionLen && len(v) <= len(hash) && strings.HasPrefix(hash, strings.ToLower(v))
}

// digest returns a strong ETag and the hex SHA-256 of the file, and for an
// HTML file (html) its content security policy, reusing the cached values
// while size and modification time are unchanged for non-HTML files. HTML
// is always hashed, with its policy cached by content digest, so even a
// same-size rebuild preserving timestamps picks up new scripts. It leaves
// content positioned at the start.
func (s *spa) digest(name string, info fs.FileInfo, content io.ReadSeeker, html bool) (etagEntry, error) {
	s.mu.Lock()
	cached, ok := s.etags[name]
	s.mu.Unlock()
	if !html && ok && cached.size == info.Size() && cached.mod.Equal(info.ModTime()) {
		return cached, nil
	}

	hash := sha256.New()
	var doc bytes.Buffer
	var sink io.Writer = hash
	if html {
		sink = io.MultiWriter(hash, &doc)
	}
	if _, err := io.Copy(sink, content); err != nil {
		return etagEntry{}, err
	}
	if _, err := content.Seek(0, io.SeekStart); err != nil {
		return etagEntry{}, err
	}
	sum := hash.Sum(nil)
	entry := etagEntry{
		size: info.Size(),
		mod:  info.ModTime(),
		tag:  `"` + base64.RawURLEncoding.EncodeToString(sum[:16]) + `"`,
		hash: hex.EncodeToString(sum),
	}
	if html {
		// Hash HTML on every request: a static build may replace a file
		// while preserving its size and timestamp. Reuse the policy only
		// when the content digest itself matches.
		if ok && cached.hash == entry.hash {
			entry.csp = cached.csp
			entry.homeStyleLink = cached.homeStyleLink
		} else {
			entry.csp = s.policy.documentCSP(doc.Bytes())
			if name == "index.html" {
				entry.homeStyleLink = homeStylePreload(doc.Bytes(), s.fsys)
			}
		}
	}

	s.mu.Lock()
	s.etags[name] = entry
	s.mu.Unlock()
	return entry, nil
}

// isHTML reports whether name is an HTML document, which gets its own
// content security policy.
func isHTML(name string) bool {
	return strings.EqualFold(path.Ext(name), ".html")
}

func contentType(name string) string {
	ext := strings.ToLower(path.Ext(name))
	if ct, ok := contentTypes[ext]; ok {
		return ct
	}
	return mime.TypeByExtension(ext)
}

func notFound(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", webdist.CacheRevalidate)
	http.Error(w, "404 page not found", http.StatusNotFound)
}
