package webdist

import "strings"

// The Cache-Control policies of the web build's files. The server's static
// handler (internal/httpapi) asks CacheControl for each file it serves.
const (
	// CacheImmutable is for files whose URL names their content: SvelteKit's
	// content-hashed build assets under _app/immutable/, and art requested
	// with a matching content-hash v query parameter (art/cards/baths.avif?v=
	// followed by the first hex digits of the file's SHA-256, as the
	// generated art index writes every path). Such a URL never serves other
	// bytes, so browsers keep it for a year without asking again. The
	// handler sends no ETag with it.
	CacheImmutable = "public, max-age=31536000, immutable"

	// CacheArt is for the shipped bitmap art under art/ requested without a
	// matching content hash. The app names all of its art with one (the
	// generated art index, and the table textures in CSS, carry ?v=), so
	// this is left to the app icons that static HTML and the web app
	// manifest name by a stable URL, to a stale v from an older deployment,
	// and to a request without v. Such a URL does not name its content
	// (art/cards/baths.avif stays that name when the art is regenerated), so
	// it cannot be immutable; but it changes rarely. A browser uses its copy
	// for a day without asking, then for up to a week more it still shows the
	// copy at once while it revalidates in the background (a cheap 304
	// against the ETag), so replaced art shows up within a day or on the next
	// visit.
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
// "art/cards/baths.avif". versioned reports whether the request named the
// file's exact content, that is, carried a v query parameter that matches
// the file's content hash; it only matters under art/.
func CacheControl(name string, versioned bool) string {
	switch {
	case strings.HasPrefix(name, immutablePrefix):
		return CacheImmutable
	case strings.HasPrefix(name, artPrefix) && versioned:
		return CacheImmutable
	case strings.HasPrefix(name, artPrefix):
		return CacheArt
	default:
		return CacheRevalidate
	}
}
