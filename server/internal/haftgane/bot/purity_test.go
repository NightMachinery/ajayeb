package bot

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/puritycheck"
)

var packagePolicy = puritycheck.Config{
	Packages: []string{"."}, Module: "github.com/NightMachinery/ajayeb/server",
	ModuleImports: []puritycheck.ModuleImportRule{{Dir: ".", Allowed: []string{"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules", "github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog", "github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"}}},
	Imports:       []puritycheck.ImportRule{{Path: "time"}, {Path: "os"}, {Path: "net"}, {Path: "log"}, {Path: "sync"}, {Path: "unsafe"}, {Path: "syscall"}, {Path: "crypto/rand"}, {Path: "math/rand"}, {Path: "io/ioutil"}, {Path: "reflect"}, {Path: "runtime"}},
}

func TestPackagePurity(t *testing.T) {
	report, err := packagePolicy.Scan(".")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Files) < 3 {
		t.Fatal("incomplete package purity scan", report.Files)
	}
	for _, v := range report.Violations {
		t.Error(v)
	}
}
func TestPackagePurityNegatives(t *testing.T) {
	for _, src := range []string{`package x;import "os"`, `package x;import "math/rand/v2"`, `package x;import "github.com/NightMachinery/ajayeb/server/internal/room"`, `package x;import "github.com/NightMachinery/ajayeb/server/internal/engine"`, `package x;import "github.com/NightMachinery/ajayeb/server/internal/enginekit/puritycheck"`, `package x;var m map[int]int`, `package x;const f=0.5`, `package x;import`} {
		if len(packagePolicy.Violations("probe.go", []byte(src))) == 0 {
			t.Error("purity guard accepted", src)
		}
	}
}

// Forbidding engine-state selectors also rejects aliases, so a policy cannot
// quietly grow a Game/State argument while importing the permitted rules API.
func forbiddenEngineSelectors(src []byte) ([]string, error) {
	f, err := parser.ParseFile(token.NewFileSet(), "policy.go", src, 0)
	if err != nil {
		return nil, err
	}
	aliases := map[string]bool{}
	for _, imp := range f.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		if path == "github.com/NightMachinery/ajayeb/server/internal/haftgane/rules" {
			name := "rules"
			if imp.Name != nil {
				name = imp.Name.Name
			}
			aliases[name] = true
		}
	}
	var bad []string
	ast.Inspect(f, func(n ast.Node) bool {
		s, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		id, ok := s.X.(*ast.Ident)
		if !ok || !aliases[id.Name] {
			return true
		}
		switch s.Sel.Name {
		case "Game", "State", "World", "Scenario", "Setup", "Record", "NewGame", "NewGameFromScenario", "ScenarioFromGame", "ResolveState", "LegalState", "Replay":
			bad = append(bad, s.Sel.Name)
		}
		return true
	})
	return bad, nil
}
func TestObservationOnlyPolicyBoundary(t *testing.T) {
	paths, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		if filepath.Ext(path) != ".go" || len(path) >= 8 && path[len(path)-8:] == "_test.go" {
			continue
		}
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		bad, err := forbiddenEngineSelectors(src)
		if err != nil || len(bad) != 0 {
			t.Error(path, bad, err)
		}
	}
	for _, selector := range []string{"Game", "State", "World", "Scenario", "Setup", "Record", "NewGame", "NewGameFromScenario", "ResolveState", "LegalState", "Replay"} {
		src := []byte("package x; import r \"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules\"; var _ = r." + selector)
		bad, err := forbiddenEngineSelectors(src)
		if err != nil || len(bad) != 1 {
			t.Error("boundary probe accepted", selector, bad, err)
		}
	}
}
