package webdist

import (
	"io/fs"
	"strings"
	"testing"
)

func TestPlaceholderHasIndex(t *testing.T) {
	b, err := fs.ReadFile(Placeholder(), "index.html")
	if err != nil {
		t.Fatalf("placeholder index.html: %v", err)
	}
	if !strings.Contains(string(b), "no web client") {
		t.Errorf("placeholder does not explain that the web build is missing: %q", b)
	}
}

func TestFSMatchesBuildTag(t *testing.T) {
	fsys, ok := FS()
	if !Embedded {
		if ok || fsys != nil {
			t.Fatalf("FS() = %v, %v without the embedweb tag; want nil, false", fsys, ok)
		}
		return
	}
	// With the tag, FS succeeds exactly when the copied build has an
	// index.html (a fresh clone has only dist/.gitkeep).
	if ok {
		if _, err := fs.Stat(fsys, "index.html"); err != nil {
			t.Fatalf("embedded FS reported ok but has no index.html: %v", err)
		}
	} else if fsys != nil {
		t.Fatalf("FS() returned a non-nil FS with ok=false")
	}
}
