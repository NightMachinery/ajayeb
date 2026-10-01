//go:build !embedweb

package webdist

import "io/fs"

// Embedded reports whether this binary was compiled with the embedweb tag.
const Embedded = false

// FS reports that no web build is embedded. Build with -tags embedweb, after
// copying the web build into dist, to embed it.
func FS() (fs.FS, bool) {
	return nil, false
}
