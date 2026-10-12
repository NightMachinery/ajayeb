package httpapi

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/NightMachinery/ajayeb/server/internal/webdist"
)

// The art file of testFS and its content hash, as tools/art/index.mjs writes
// it into the art index (the first 10 hex digits of the SHA-256).
const artBody = "\x00\x00\x00\x1cftypavif"

func artVersion() string {
	sum := sha256.Sum256([]byte(artBody))
	return hex.EncodeToString(sum[:])[:10]
}

// The art index names every art file with ?v=<content hash>: such a request
// is cached for a year and needs no revalidation, so it carries no ETag.
func TestVersionedArtIsImmutable(t *testing.T) {
	h := newTestHandler(t)
	for _, v := range []string{artVersion(), strings.ToUpper(artVersion()), fullHash()} {
		target := "/art/cards/baths.avif?v=" + v
		rec := do(t, h, http.MethodGet, target, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200", target, rec.Code)
		}
		if cc := rec.Header().Get("Cache-Control"); cc != webdist.CacheImmutable {
			t.Errorf("GET %s Cache-Control = %q, want %q", target, cc, webdist.CacheImmutable)
		}
		if tag := rec.Header().Get("ETag"); tag != "" {
			t.Errorf("GET %s ETag = %q, want none", target, tag)
		}
		if ct := rec.Header().Get("Content-Type"); ct != "image/avif" {
			t.Errorf("GET %s Content-Type = %q, want image/avif", target, ct)
		}
		if rec.Body.String() != artBody {
			t.Errorf("GET %s body = %q, want the file", target, rec.Body.String())
		}
	}
}

func fullHash() string {
	sum := sha256.Sum256([]byte(artBody))
	return hex.EncodeToString(sum[:])
}

// Art requested without a content hash (an icon named by static HTML or the
// web app manifest, a texture named in CSS), or with a v that does not match
// the file (left over from an older deployment, or too short to be a hash),
// keeps the shorter art policy and the ETag, so it is revalidated in time.
func TestUnversionedArtIsRevalidated(t *testing.T) {
	h := newTestHandler(t)
	for _, target := range []string{
		"/art/cards/baths.avif",
		"/art/cards/baths.avif?v=",
		"/art/cards/baths.avif?v=0123456789",
		"/art/cards/baths.avif?v=" + artVersion()[:7],
		"/art/cards/baths.avif?v=" + fullHash() + "00",
		"/art/cards/baths.avif?x=" + artVersion(),
	} {
		rec := do(t, h, http.MethodGet, target, nil)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200", target, rec.Code)
		}
		if cc := rec.Header().Get("Cache-Control"); cc != webdist.CacheArt {
			t.Errorf("GET %s Cache-Control = %q, want %q", target, cc, webdist.CacheArt)
		}
		tag := rec.Header().Get("ETag")
		if tag == "" {
			t.Errorf("GET %s has no ETag", target)
			continue
		}
		again := do(t, h, http.MethodGet, target, http.Header{"If-None-Match": {tag}})
		if again.Code != http.StatusNotModified {
			t.Errorf("GET %s revalidation status = %d, want 304", target, again.Code)
		}
	}
}

// A v parameter outside art/ changes nothing: the shell and other static
// files still revalidate, and the immutable build assets stay immutable.
func TestVersionOutsideArtChangesNothing(t *testing.T) {
	fsys := testFS()
	fsys["fonts/body.woff2"] = &fstest.MapFile{Data: []byte(artBody)}
	h, err := New(Config{Version: "test-1"}, fsys)
	if err != nil {
		t.Fatal(err)
	}
	for target, want := range map[string]string{
		"/fonts/body.woff2?v=" + artVersion():                   webdist.CacheRevalidate,
		"/favicon.svg?v=" + artVersion():                        webdist.CacheRevalidate,
		"/?v=" + artVersion():                                   webdist.CacheRevalidate,
		"/play/ABC?v=" + artVersion():                           webdist.CacheRevalidate,
		"/_app/immutable/entry/start.Bx81.js?v=" + artVersion(): webdist.CacheImmutable,
	} {
		rec := do(t, h, http.MethodGet, target, nil)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s status = %d, want 200", target, rec.Code)
			continue
		}
		if cc := rec.Header().Get("Cache-Control"); cc != want {
			t.Errorf("GET %s Cache-Control = %q, want %q", target, cc, want)
		}
		hasTag := rec.Header().Get("ETag") != ""
		if hasTag != (want != webdist.CacheImmutable) {
			t.Errorf("GET %s ETag present = %v, want %v", target, hasTag, want != webdist.CacheImmutable)
		}
	}
}

func TestVersionMatches(t *testing.T) {
	hash := fullHash()
	for _, c := range []struct {
		v    string
		want bool
	}{
		{hash[:10], true},
		{hash[:8], true},
		{hash, true},
		{strings.ToUpper(hash[:10]), true},
		{hash[:7], false},
		{"", false},
		{"zzzzzzzzzz", false},
		{hash + "0", false},
	} {
		if got := versionMatches(c.v, hash); got != c.want {
			t.Errorf("versionMatches(%q) = %v, want %v", c.v, got, c.want)
		}
	}
	if versionMatches(hash[:10], "") {
		t.Error("versionMatches with no hash = true, want false")
	}
}
