package webdist

import "strings"

// The Cache-Control policies of the web build's files. The server's static
// handler (internal/httpapi) asks CacheControl for each file it serves.
const (
	// CacheImmutable is for SvelteKit's content-hashed build assets: a file
	// there never changes under the same name, so browsers keep it for a
	// year without asking again. The handler sends no ETag with it.
	CacheImmutable = "public, max-age=31536000, immutable"

	// CacheArt is for the shipped bitmap art under art/. Its file names are
	// not content-hashed (art/cards/baths.avif stays that name when the art
	// is regenerated), so it cannot be immutable; but it changes rarely and
	// is most of what a game downloads. A browser uses its copy for a day
	// without asking, then for up to a week more it still shows the copy at
	// once while it revalidates in the background (a cheap 304 against the
	// ETag), so replaced art shows up within a day or on the next visit.
	CacheArt = "public, max-age=86400, stale-while-revalidate=604800"

	// CacheRevalidate is for everything else, the app shell included:
	// browsers revalidate every time against the ETag, so a new deployment
	// shows at once.
	CacheRevalidate = "no-cache"
)

const (
	immutablePrefix = "_app/immutable/"
	artPrefix       = "art/"
)

// CacheControl returns the Cache-Control value for a file of the web build,
// named by its path inside the build without a leading slash, such as
// "index.html", "_app/immutable/entry/start.Bx81.js" or
// "art/cards/baths.avif".
func CacheControl(name string) string {
	switch {
	case strings.HasPrefix(name, immutablePrefix):
		return CacheImmutable
	case strings.HasPrefix(name, artPrefix):
		return CacheArt
	default:
		return CacheRevalidate
	}
}
