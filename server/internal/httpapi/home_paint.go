package httpapi

import (
	"bytes"
	"io/fs"
	"regexp"
	"strings"
)

var (
	homeLinkTag        = regexp.MustCompile(`<link\b[^>]*>`)
	homeStylesheetRel  = regexp.MustCompile(`\brel=["']stylesheet["']`)
	homeStylesheetHref = regexp.MustCompile(`\bhref=["']([^"']+)["']`)
	homeAssetCSS       = regexp.MustCompile(`^_app/immutable/assets/[A-Za-z0-9_.-]+\.css$`)
)

// homeStylePreload discovers the largest render-blocking stylesheet from the
// generated Home document. Starting it at response headers avoids its waiting
// behind smaller CSS files, fonts and modules on a cold connection. The actual
// stylesheet tags retain their cascade order. No URL or size is hard-coded.
func homeStylePreload(doc []byte, fsys fs.FS) string {
	end := bytes.Index(doc, []byte("</head>"))
	if end < 0 || !bytes.Contains(doc[end:], []byte("data-hero")) {
		return ""
	}
	var largest string
	var size int64
	for _, tag := range homeLinkTag.FindAll(doc[:end], -1) {
		if !homeStylesheetRel.Match(tag) {
			continue
		}
		match := homeStylesheetHref.FindSubmatch(tag)
		if len(match) != 2 {
			continue
		}
		name := strings.TrimPrefix(strings.TrimPrefix(string(match[1]), "./"), "/")
		// Only generated same-origin CSS is eligible, never a remote or arbitrary
		// document-supplied URL. This also keeps Link header delimiters out.
		if !homeAssetCSS.MatchString(name) {
			continue
		}
		info, err := fs.Stat(fsys, name)
		if err == nil && info.Mode().IsRegular() && info.Size() > size {
			largest, size = name, info.Size()
		}
	}
	if largest == "" {
		return ""
	}
	return "</" + largest + ">; rel=preload; as=style"
}
