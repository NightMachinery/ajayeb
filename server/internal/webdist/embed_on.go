//go:build embedweb

package webdist

import (
	"embed"
	"io/fs"
)

// dist holds the web build copied into this package's dist directory. The
// all: prefix keeps files whose names start with "." or "_", such as
// SvelteKit's _app directory.
//
//go:embed all:dist
var dist embed.FS

// Embedded reports whether this binary was compiled with the embedweb tag.
const Embedded = true

// FS returns the embedded web build, rooted so that "index.html" is at the
// top, and true. It returns nil and false when the embedded directory holds
// no index.html, for example when the binary was built with the tag before
// the web build was copied in.
func FS() (fs.FS, bool) {
	sub, err := fs.Sub(dist, "dist")
	if err != nil || !hasIndex(sub) {
		return nil, false
	}
	return sub, true
}

// hasIndex reports whether fsys has a regular index.html at its root.
func hasIndex(fsys fs.FS) bool {
	info, err := fs.Stat(fsys, "index.html")
	return err == nil && info.Mode().IsRegular()
}
