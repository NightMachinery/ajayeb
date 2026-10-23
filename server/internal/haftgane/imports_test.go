package haftgane

import (
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/puritycheck"
)

var policy = puritycheck.Config{
	Packages: []string{"."},
	Module:   "github.com/NightMachinery/ajayeb/server",
	ModuleImports: []puritycheck.ModuleImportRule{
		{Dir: "."},
		{Dir: "catalog", Allowed: []string{
			"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng",
		}},
		{Dir: "rules", Allowed: []string{
			"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog",
			"github.com/NightMachinery/ajayeb/server/internal/haftgane/view",
			"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng",
		}},
		{Dir: "view"},
		{Dir: "bot", Allowed: []string{
			"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules",
			"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog",
			"github.com/NightMachinery/ajayeb/server/internal/enginekit/rng",
		}},
		{Dir: "fakeengine", Allowed: []string{
			"github.com/NightMachinery/ajayeb/server/internal/haftgane/fakeengine/fakeview",
		}},
	},
	Imports: []puritycheck.ImportRule{
		{Path: "time"}, {Path: "os"}, {Path: "net"}, {Path: "log"}, {Path: "sync"}, {Path: "unsafe"}, {Path: "syscall"}, {Path: "crypto/rand"}, {Path: "io/ioutil"},
		{Path: "math/rand"}, // Both versions, even a file named rng.go (D32).
		{Path: "github.com/NightMachinery/ajayeb/server/internal/engine"},
		{Path: "github.com/NightMachinery/ajayeb/server/internal/enginekit/catalogdiff"},
	},
}

func TestHaftganeImports(t *testing.T) {
	report, err := policy.Scan(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range report.Violations {
		t.Error(v)
	}
	var catalog, rules int
	for _, file := range report.Files {
		if strings.HasPrefix(file, "catalog/") {
			catalog++
		}
		if strings.HasPrefix(file, "rules/") {
			rules++
		}
	}
	if catalog < 3 || rules < 4 {
		t.Fatalf("incomplete scan %d catalog, %d rules", catalog, rules)
	}
	t.Logf("scanned %d files (%d catalog, %d rules)", len(report.Files), catalog, rules)
}
func TestHaftganePurityNegatives(t *testing.T) {
	for _, src := range []string{`package rules;import "math/rand/v2"`, `package rules;import "math/rand"`, `package rules;import "os"`, `package rules;var m map[uint8]bool`, `package rules;const f=1.5`, `package rules;import "github.com/NightMachinery/ajayeb/server/internal/engine"`, `package rules;import "github.com/NightMachinery/ajayeb/server/internal/enginekit/puritycheck"`, `package rules;import`} {
		if bad := policy.Violations("rules/rng.go", []byte(src)); len(bad) == 0 {
			t.Errorf("accepted %s", src)
		}
	}
}

func TestHaftganeModuleImportBoundary(t *testing.T) {
	for _, tt := range []struct {
		file, pkg string
		allowed   bool
	}{
		{"catalog/data.go", "internal/enginekit/rng", true},
		{"catalog/data.go", "internal/haftgane/rules", false},
		{"catalog/data.go", "internal/haftgane/view", false},
		{"rules/legal.go", "internal/haftgane/catalog", true},
		{"rules/legal.go", "internal/haftgane/view", true},
		{"rules/legal.go", "internal/enginekit/rng", true},
		{"rules/legal.go", "internal/room", false},
		{"rules/legal.go", "internal/wire", false},
		{"rules/legal.go", "internal/enginekit/catalogdiff", false},
		{"rules/legal.go", "internal/haftgane/fakeengine", false},
		{"rules/legal.go", "internal/haftgane/catalog/extra", false},
		{"view/view.go", "internal/haftgane/rules", false},
	} {
		bad := policy.Violations(tt.file, []byte(`package rules;import "github.com/NightMachinery/ajayeb/server/`+tt.pkg+`"`))
		if (len(bad) == 0) != tt.allowed {
			t.Errorf("%s imports %s: %v", tt.file, tt.pkg, bad)
		}
	}
}
