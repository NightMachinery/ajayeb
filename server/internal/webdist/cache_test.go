package webdist

import (
	"strings"
	"testing"
)

func TestCacheControl(t *testing.T) {
	for name, want := range map[string]string{
		"_app/immutable/entry/start.Bx81.js": CacheImmutable,
		"_app/immutable/assets/app.Q2c9.css": CacheImmutable,
		"art/cards/baths.avif":               CacheArt,
		"art/cards/baths@thumb.avif":         CacheArt,
		"art/ui/paper.avif":                  CacheArt,
		"art/victory/military.avif":          CacheArt,
		"index.html":                         CacheRevalidate,
		"_app/version.json":                  CacheRevalidate,
		"favicon.svg":                        CacheRevalidate,
		"manifest.webmanifest":               CacheRevalidate,
		"icons/apple-touch-icon.png":         CacheRevalidate,
		"fonts/inter-400.woff2":              CacheRevalidate,
		"play/art/cards.avif":                CacheRevalidate,
		"artwork.html":                       CacheRevalidate,
		"dev/gallery/index.html":             CacheRevalidate,
		".well-known/security.txt":           CacheRevalidate,
		"_app/immutable":                     CacheRevalidate,
		"_app/immutable.js":                  CacheRevalidate,
		"art":                                CacheRevalidate,
	} {
		if got := CacheControl(name); got != want {
			t.Errorf("CacheControl(%q) = %q, want %q", name, got, want)
		}
	}
}

// The art policy must stay cacheable but never immutable: art file names
// are not content-hashed, so a regenerated asset reuses its name.
func TestArtIsCachedButRevalidated(t *testing.T) {
	if strings.Contains(CacheArt, "immutable") || strings.Contains(CacheArt, "no-cache") {
		t.Fatalf("CacheArt = %q: art must be cached and then revalidated", CacheArt)
	}
	if !strings.Contains(CacheArt, "max-age=") || !strings.Contains(CacheArt, "stale-while-revalidate=") {
		t.Fatalf("CacheArt = %q: want a max-age and a stale-while-revalidate window", CacheArt)
	}
}
