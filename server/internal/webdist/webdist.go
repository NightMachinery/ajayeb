// Package webdist provides the web client's static build to the server.
//
// The build is embedded only when the binary is compiled with the embedweb
// build tag, after the web build has been copied into this package's dist
// directory. Without the tag (a fresh clone, go test, go vet) the package
// still builds and FS reports that nothing is embedded, so the server can fall
// back to a directory given with --static or to the built-in placeholder page.
package webdist

import (
	"embed"
	"io/fs"
)

//go:embed placeholder/index.html
var placeholderFiles embed.FS

// Placeholder returns a one-page file system whose index.html explains that
// the web build is missing. The server serves it when it has neither a
// --static directory nor an embedded build.
func Placeholder() fs.FS {
	sub, err := fs.Sub(placeholderFiles, "placeholder")
	if err != nil {
		// fs.Sub fails only for an invalid directory name, which is a
		// compile-time constant here.
		panic("webdist: placeholder: " + err.Error())
	}
	return sub
}
