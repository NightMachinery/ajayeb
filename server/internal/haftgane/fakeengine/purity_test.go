package fakeengine

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

// The fake follows AGENTS.md "Go engine purity", so the actor is built
// against an engine with the real one's properties. Its non-test files may
// import only these packages: no clocks, I/O, synchronisation or
// randomness (rng.go is its own splitmix64), and nothing outside the
// standard library but its own DTO package.
var allowedImports = []string{
	"errors",
	"strconv",
	"strings",
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/fakeengine/fakeview",
}

// purityViolations checks one non-test source file: its imports, map types
// (rules logic uses arrays) and floating point.
func purityViolations(name string, src []byte) []string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution)
	if err != nil {
		return []string{err.Error()}
	}
	var out []string
	bad := func(pos token.Pos, msg string) { out = append(out, fset.Position(pos).String()+": "+msg) }
	for _, imp := range f.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		ok := false
		for _, a := range allowedImports {
			ok = ok || path == a
		}
		if !ok {
			bad(imp.Pos(), "imports "+path)
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.MapType:
			bad(n.Pos(), "uses a map")
		case *ast.BasicLit:
			if n.Kind == token.FLOAT || n.Kind == token.IMAG {
				bad(n.Pos(), "has a floating-point literal")
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

func TestPurity(t *testing.T) {
	scanned := 0
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		scanned++
		for _, v := range purityViolations(path, src) {
			t.Error(v)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if scanned < 10 {
		t.Fatalf("scanned only %d files; is the walk rooted right?", scanned)
	}
}

// TestPurityCatches proves each rule can fail.
func TestPurityCatches(t *testing.T) {
	for name, src := range map[string]string{
		"time":  "package x\nimport \"time\"\nvar _ = time.Now",
		"rand":  "package x\nimport \"math/rand/v2\"\nvar _ = rand.IntN",
		"map":   "package x\nvar m map[int]int",
		"float": "package x\nvar f = 1.5",
		"type":  "package x\nvar f float64",
	} {
		if len(purityViolations(name+".go", []byte(src))) == 0 {
			t.Errorf("%s: no violation reported", name)
		}
	}
}
