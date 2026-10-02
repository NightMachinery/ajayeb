package engine

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The purity guard. Rules code must be deterministic: no I/O, clocks,
// goroutine synchronization, unseeded randomness, floating point or map
// iteration. golangci-lint (depguard) bans the same imports; this test keeps
// the guarantee even where the linter does not run.

// bannedImports are forbidden in every non-test file under internal/engine,
// each with every package below it (os also bans os/exec, log bans
// log/slog, and so on).
var bannedImports = []string{"time", "os", "net", "log", "sync", "unsafe", "syscall", "crypto/rand", "io/ioutil"}

// rngOnlyImport may be imported only by rngFile; math/rand (v1) is banned
// everywhere.
const (
	rngOnlyImport = "math/rand/v2"
	rngFile       = "rng.go"
)

// mapFiles may use map types. They build the view DTOs, whose maps
// encoding/json writes in sorted key order; rules logic never uses maps.
var mapFiles = []string{"view.go", "eventfilter.go"}

// purityViolations checks one non-test source file. rel is its path relative
// to internal/engine, with forward slashes.
func purityViolations(rel string, src []byte) []string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, rel, src, parser.SkipObjectResolution)
	if err != nil {
		return []string{err.Error()}
	}
	var out []string
	bad := func(pos token.Pos, msg string) { out = append(out, fset.Position(pos).String()+": "+msg) }
	for _, imp := range f.Imports {
		path, err := strconv.Unquote(imp.Path.Value)
		if err != nil {
			bad(imp.Pos(), "unreadable import")
			continue
		}
		for _, b := range bannedImports {
			if path == b || strings.HasPrefix(path, b+"/") {
				bad(imp.Pos(), "imports "+path)
			}
		}
		if path == "math/rand" {
			bad(imp.Pos(), "imports math/rand (v1); use the engine RNG")
		}
		if path == rngOnlyImport && rel != rngFile {
			bad(imp.Pos(), "imports math/rand/v2 outside "+rngFile+"; use the engine RNG")
		}
	}
	mapsOK := false
	for _, m := range mapFiles {
		if rel == m {
			mapsOK = true
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.MapType:
			if !mapsOK {
				bad(n.Pos(), "uses a map; rules logic uses arrays indexed by enums")
			}
		case *ast.BasicLit:
			if n.Kind == token.FLOAT || n.Kind == token.IMAG {
				bad(n.Pos(), "has a floating-point literal "+n.Value)
			}
		case *ast.Ident:
			switch n.Name {
			case "float32", "float64", "complex64", "complex128":
				bad(n.Pos(), "uses "+n.Name)
			}
		}
		return true
	})
	return out
}

// TestEngineImports scans every non-test Go file of internal/engine and its
// subpackages (catalog, enginetest).
func TestEngineImports(t *testing.T) {
	var files, catalogFiles int
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel := filepath.ToSlash(path)
		files++
		if strings.HasPrefix(rel, "catalog/") {
			catalogFiles++
		}
		for _, v := range purityViolations(rel, src) {
			t.Error(v)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Guard against a scan that silently checks nothing.
	if files < 10 || catalogFiles < 3 {
		t.Fatalf("scanned %d files (%d in catalog); the walk is not finding the sources", files, catalogFiles)
	}
	t.Logf("scanned %d files (%d in catalog)", files, catalogFiles)
}

// TestPurityCheckCatches checks the checker itself on small sources.
func TestPurityCheckCatches(t *testing.T) {
	cases := []struct {
		rel, src string
		bad      int
	}{
		{"x.go", `package engine; import "time"`, 1},
		{"x.go", `package engine; import "os/exec"`, 1},
		{"x.go", `package engine; import "log/slog"`, 1},
		{"x.go", `package engine; import "sync/atomic"`, 1},
		{"x.go", `package engine; import "net/http"`, 1},
		{"x.go", `package engine; import "unsafe"`, 1},
		{"x.go", `package engine; import "crypto/rand"`, 1},
		{"x.go", `package engine; import "math/rand"`, 1},
		{"x.go", `package engine; import "math/rand/v2"`, 1},
		{"catalog/rng.go", `package catalog; import "math/rand/v2"`, 1},
		{"rng.go", `package engine; import "math/rand/v2"`, 0},
		{"x.go", `package engine; import ("math/bits"; "strings"; "fmt"; "errors"; "iter")`, 0},
		{"x.go", `package engine; import "timeutil"`, 0}, // not time/...
		{"x.go", `package engine; var m map[string]int`, 1},
		{"x.go", `package engine; func f() { _ = make(map[uint8]bool) }`, 1},
		{"view.go", `package engine; var m map[string]int`, 0},
		{"catalog/view.go", `package catalog; var m map[string]int`, 1},
		{"x.go", `package engine; var f = 1.5`, 1},
		{"x.go", `package engine; var f = 2i`, 1},
		{"x.go", `package engine; var f float64`, 1},
		{"x.go", `package engine; func f(x int) float32 { return float32(x) }`, 2},
		{"x.go", `package engine; var n = 1e3`, 1},
		{"x.go", `package engine; var n = 1 << 3`, 0},
		{"x.go", `package engine; import "time"; var m map[int]float64`, 3},
		{"x.go", `package engine; import`, 1}, // a parse error is a violation
	}
	for _, c := range cases {
		got := purityViolations(c.rel, []byte(c.src))
		if len(got) != c.bad {
			t.Errorf("%s %q: %d violations, want %d: %v", c.rel, c.src, len(got), c.bad, got)
		}
	}
}
