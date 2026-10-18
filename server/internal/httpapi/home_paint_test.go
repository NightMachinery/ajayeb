package httpapi

import (
	"net/http"
	"strings"
	"testing"
	"testing/fstest"
)

func TestHomeStylePreload(t *testing.T) {
	const small = "_app/immutable/assets/small.ab12.css"
	const large = "_app/immutable/assets/main.cd34.css"
	const next = "_app/immutable/assets/main.ef56.css"
	doc := func(css string) []byte {
		return []byte(`<head><link rel="stylesheet" href="./` + small + `"><link href="/` + css + `" rel="stylesheet"></head><body data-hero>Home</body>`)
	}
	files := fstest.MapFS{
		"index.html": {Data: doc(large)},
		"shell.html": {Data: []byte(`<head></head><body>shell</body>`)},
		small:        {Data: []byte("small")},
		large:        {Data: []byte("larger body")},
		next:         {Data: []byte("next content")},
	}
	h := newSPA(files)
	want := "</" + large + ">; rel=preload; as=style"
	for _, target := range []string{"/", "/?play=ai", "/index.html"} {
		rec := do(t, h, http.MethodGet, target, nil)
		if got := rec.Header().Get("Link"); got != want {
			t.Errorf("%s Link = %q, want %q", target, got, want)
		}
		if rec.Body.String() != string(doc(large)) {
			t.Fatal("Home body changed")
		}
	}
	for _, target := range []string{"/play/ABCDEF", "/settings", "/rules", "/shell.html", "/" + large} {
		if got := do(t, h, http.MethodGet, target, nil).Header().Get("Link"); got != "" {
			t.Errorf("%s unexpected preload %q", target, got)
		}
	}
	// A same-size HTML replacement with the same timestamp must not retain a
	// stylesheet URL from an earlier build, even after the policy was cached.
	files["index.html"].Data = doc(next)
	if got := do(t, h, http.MethodGet, "/", nil).Header().Get("Link"); got != "</"+next+">; rel=preload; as=style" {
		t.Fatalf("replacement Link = %q", got)
	}
}

func TestHomeStylePreloadIgnoresUnsafeOrMissingAssets(t *testing.T) {
	files := fstest.MapFS{"_app/immutable/assets/local.hash.css": {Data: []byte("local")}}
	for _, href := range []string{"https://example.test/style.css", "//example.test/style.css", "/private.css", "/_app/immutable/assets/missing.hash.css", "/_app/immutable/assets/local.hash.css?bad=1", "/_app/immutable/assets/../local.hash.css"} {
		doc := []byte(`<head><link rel="stylesheet" href="` + href + `"></head><body data-hero></body>`)
		if got := homeStylePreload(doc, files); got != "" {
			t.Errorf("%q unexpected preload %q", href, got)
		}
	}
	doc := `<head><link rel="stylesheet" href="/_app/immutable/assets/local.hash.css"></head><body>shell</body>`
	if got := homeStylePreload([]byte(doc), files); got != "" {
		t.Errorf("generic shell preload %q", got)
	}
	if got := homeStylePreload([]byte(strings.ReplaceAll(doc, "</head>", "")), files); got != "" {
		t.Errorf("incomplete head preload %q", got)
	}
}
