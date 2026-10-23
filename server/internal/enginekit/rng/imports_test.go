package rng

import (
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/puritycheck"
)

// Only the PCG implementation may use v2; the rest of this package stays as
// pure as the calling engines. Duel and Haftgane have no local exception.
var rngPurity = puritycheck.Config{
	Packages:      []string{"."},
	Module:        "github.com/NightMachinery/ajayeb/server",
	ModuleImports: []puritycheck.ModuleImportRule{{Dir: "."}},
	Imports: []puritycheck.ImportRule{
		{Path: "time"}, {Path: "os"}, {Path: "net"}, {Path: "log"}, {Path: "sync"}, {Path: "unsafe"}, {Path: "syscall"}, {Path: "crypto/rand"}, {Path: "io/ioutil"},
		{Path: "math/rand", Exact: true},
		{Path: "math/rand/v2", Exact: true, AllowedFiles: []string{"rng.go"}},
	},
}

func TestRNGImports(t *testing.T) {
	report, err := rngPurity.Scan(".")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Files) < 2 {
		t.Fatalf("expected at least generator and digest, scanned %v", report.Files)
	}
	for _, bad := range report.Violations {
		t.Error(bad)
	}
}

func TestRNGImportException(t *testing.T) {
	for _, tt := range []struct {
		path, src string
		bad       int
	}{
		{"rng.go", `package rng;import "math/rand/v2"`, 0},
		{"digest.go", `package rng;import "math/rand/v2"`, 1},
		{"nested/rng.go", `package nested;import "math/rand/v2"`, 1},
		{"rng.go", `package rng;import "math/rand"`, 1},
		{"rng.go", `package rng;import "crypto/rand"`, 1},
		{"rng.go", `package rng;import "os"`, 1},
		{"rng.go", `package rng;var m map[string]int`, 1},
		{"rng.go", `package rng;var f float64`, 1},
	} {
		if got := rngPurity.Violations(tt.path, []byte(tt.src)); len(got) != tt.bad {
			t.Errorf("%s %s: %v", tt.path, tt.src, got)
		}
	}
	bad := rngPurity.Violations("rng.go", []byte(`package rng;import "github.com/NightMachinery/ajayeb/server/internal/enginekit/puritycheck"`))
	if len(bad) != 2 || !strings.Contains(strings.Join(bad, "\n"), "test helper only") || !strings.Contains(strings.Join(bad, "\n"), "outside the module import allow-list") {
		t.Fatal(bad)
	}
}
