// Package puritycheck supplies file and type guards for engine tests only.
// It performs I/O and reflection and must never be imported by rules code.
package puritycheck

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ImportRule forbids Path and its subpackages unless Exact is set. AllowedFiles
// lists exact root-relative paths permitted to use the import (for example the
// game's one generator file). Message overrides the default "imports <path>".
type ImportRule struct {
	Path         string
	Exact        bool
	AllowedFiles []string
	Message      string
}

// Config holds a game's policy. Packages are directories relative to Scan's
// root; each is walked recursively, excluding testdata and *_test.go. MapFiles
// are exact root-relative paths whose DTOs may use maps.
type Config struct {
	Packages []string
	Imports  []ImportRule
	MapFiles []string
}

// Report retains the scanned file paths so each game can enforce its own
// minimum scan counts, independently of the shared walk.
type Report struct {
	Files      []string
	Violations []string
}

// Violations checks one non-test source. rel uses forward slashes relative to
// the configured root. Parse errors are violations, never a skipped file.
func (c Config) Violations(rel string, src []byte) []string {
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
		for _, rule := range c.Imports {
			if path != rule.Path && (rule.Exact || !strings.HasPrefix(path, rule.Path+"/")) {
				continue
			}
			if contains(rule.AllowedFiles, rel) {
				continue
			}
			message := rule.Message
			if message == "" {
				message = "imports " + path
			}
			bad(imp.Pos(), message)
		}
		// Even an otherwise allowed import cannot bring this impure test helper
		// back into a non-test engine file. This applies to every game's config.
		const helper = "github.com/NightMachinery/ajayeb/server/internal/enginekit/puritycheck"
		if path == helper || strings.HasPrefix(path, helper+"/") {
			bad(imp.Pos(), "imports "+path+" (test helper only)")
		}
	}
	mapsOK := contains(c.MapFiles, rel)
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

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// Scan fails on traversal/read errors. The caller must also check Report.Files
// against game-local minimum counts, so an empty or partial tree cannot pass.
func (c Config) Scan(root string) (Report, error) {
	var report Report
	if len(c.Packages) == 0 {
		return report, errors.New("puritycheck: no packages configured")
	}
	seen := make(map[string]bool)
	for _, pkg := range c.Packages {
		err := filepath.WalkDir(filepath.Join(root, pkg), func(path string, d fs.DirEntry, err error) error {
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
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if seen[rel] {
				return nil
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			seen[rel] = true
			report.Files = append(report.Files, rel)
			report.Violations = append(report.Violations, c.Violations(rel, src)...)
			return nil
		})
		if err != nil {
			return report, err
		}
	}
	return report, nil
}
