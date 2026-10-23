package engine

import (
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/puritycheck"
)

// Duel owns its policy; the shared helper only parses and walks files. DTO
// maps stay confined to these exact files; all randomness uses enginekit/rng.
var duelPurity = puritycheck.Config{
	Packages: []string{"."},
	Imports: []puritycheck.ImportRule{
		{Path: "time"}, {Path: "os"}, {Path: "net"}, {Path: "log"},
		{Path: "sync"}, {Path: "unsafe"}, {Path: "syscall"},
		{Path: "crypto/rand"}, {Path: "io/ioutil"},
		{Path: "math/rand", Exact: true, Message: "imports math/rand (v1); use the engine RNG"},
		{Path: "math/rand/v2", Exact: true, Message: "imports math/rand/v2; use the engine RNG"},
	},
	MapFiles: []string{"view.go", "eventfilter.go"},
}

func purityViolations(rel string, src []byte) []string {
	return duelPurity.Violations(rel, src)
}

// TestEngineImports scans every non-test Go file of internal/engine and its
// subpackages (catalog, enginetest), including imports of impure test helpers.
func TestEngineImports(t *testing.T) {
	report, err := duelPurity.Scan(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range report.Violations {
		t.Error(v)
	}
	files, catalogFiles := len(report.Files), 0
	for _, rel := range report.Files {
		if strings.HasPrefix(rel, "catalog/") {
			catalogFiles++
		}
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
		{"rng.go", `package engine; import "math/rand/v2"`, 1},
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

func TestPurityHelperIsTestOnly(t *testing.T) {
	src := []byte(`package engine; import "github.com/NightMachinery/ajayeb/server/internal/enginekit/puritycheck"`)
	bad := purityViolations("rules.go", src)
	if len(bad) != 1 || !strings.Contains(bad[0], "test helper only") {
		t.Fatalf("Duel admitted the impure test helper: %v", bad)
	}
	t.Log(bad[0])
}
