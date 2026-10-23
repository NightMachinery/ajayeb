package fakeengine

import (
	"bufio"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// The fake engine and the harness built on it are test infrastructure.
// testOnlyPackages (each with every package below it) may be imported by
// any _test.go file, but by non-test code only inside testOnlyImporters.
// cmd/ajayeb-drive is a test tool that drives the fake in process; the
// server binary, cmd/ajayeb, must never reach any of them.
var (
	testOnlyPackages  = []string{"internal/haftgane/fakeengine", "internal/tabletest"}
	testOnlyImporters = []string{"internal/haftgane/fakeengine", "internal/tabletest", "cmd/ajayeb-drive"}
	serverBinary      = "cmd/ajayeb"
)

// moduleImports maps each package directory of the server module (relative
// to go.mod, with forward slashes) to the module-internal packages its
// non-test files import, also relative.
func moduleImports(t *testing.T) map[string][]string {
	t.Helper()
	root, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	mod, err := os.Open(filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("server/go.mod not found from %s: %v", root, err)
	}
	defer mod.Close()
	sc := bufio.NewScanner(mod)
	sc.Scan()
	module := strings.TrimSpace(strings.TrimPrefix(sc.Text(), "module"))
	if module == "" || strings.ContainsAny(module, " \t") {
		t.Fatalf("cannot read the module path from %s", mod.Name())
	}
	out := map[string][]string{}
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "testdata" || (strings.HasPrefix(d.Name(), ".") && path != root) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, filepath.Dir(path))
		rel = filepath.ToSlash(rel)
		if _, ok := out[rel]; !ok {
			out[rel] = []string{}
		}
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			if strings.HasPrefix(p, module+"/") {
				out[rel] = append(out[rel], strings.TrimPrefix(p, module+"/"))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func under(pkg string, roots []string) bool {
	for _, r := range roots {
		if pkg == r || strings.HasPrefix(pkg, r+"/") {
			return true
		}
	}
	return false
}

// guardViolations reports every non-test import of a test-only package from
// outside the allowed importers, and every test-only package the server
// binary reaches.
func guardViolations(imports map[string][]string) []string {
	var out []string
	dirs := make([]string, 0, len(imports))
	for d := range imports {
		dirs = append(dirs, d)
	}
	slices.Sort(dirs)
	for _, d := range dirs {
		if under(d, testOnlyImporters) {
			continue
		}
		for _, p := range imports[d] {
			if under(p, testOnlyPackages) {
				out = append(out, d+" imports "+p)
			}
		}
	}
	seen := map[string]bool{serverBinary: true}
	queue := []string{serverBinary}
	for len(queue) > 0 {
		d := queue[0]
		queue = queue[1:]
		if under(d, testOnlyPackages) {
			out = append(out, serverBinary+" reaches "+d)
		}
		for _, p := range imports[d] {
			if !seen[p] {
				seen[p] = true
				queue = append(queue, p)
			}
		}
	}
	if len(seen) < 5 {
		out = append(out, serverBinary+" reaches only "+strconv.Itoa(len(seen))+" packages; is the walk rooted right?")
	}
	return out
}

func TestTestOnlyGuard(t *testing.T) {
	for _, v := range guardViolations(moduleImports(t)) {
		t.Error(v)
	}
}

// TestTestOnlyGuardCatches plants an import from a production package and
// from the server binary's closure, and checks that both are reported.
func TestTestOnlyGuardCatches(t *testing.T) {
	imports := moduleImports(t)
	if len(guardViolations(imports)) != 0 {
		t.Skip("the real tree already fails; TestTestOnlyGuard reports it")
	}
	planted := map[string][]string{}
	for d, ps := range imports {
		planted[d] = slices.Clone(ps)
	}
	planted["internal/room"] = append(planted["internal/room"], "internal/tabletest/direct")
	vs := guardViolations(planted)
	joined := strings.Join(vs, "\n")
	if !strings.Contains(joined, "internal/room imports internal/tabletest/direct") ||
		!strings.Contains(joined, "cmd/ajayeb reaches internal/tabletest/direct") {
		t.Fatalf("a planted import was not reported:\n%s", joined)
	}
}
