package bot

import (
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/puritycheck"
)

var packagePolicy = puritycheck.Config{
	Packages: []string{"."}, Module: "github.com/NightMachinery/ajayeb/server",
	ModuleImports: []puritycheck.ModuleImportRule{{Dir: ".", Allowed: []string{"github.com/NightMachinery/ajayeb/server/internal/haftgane/rules", "github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog", "github.com/NightMachinery/ajayeb/server/internal/enginekit/rng"}}},
	Imports:       []puritycheck.ImportRule{{Path: "time"}, {Path: "os"}, {Path: "net"}, {Path: "log"}, {Path: "sync"}, {Path: "unsafe"}, {Path: "syscall"}, {Path: "crypto/rand"}, {Path: "math/rand"}, {Path: "io/ioutil"}},
}

func TestPackagePurity(t *testing.T) {
	report, err := packagePolicy.Scan(".")
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Files) < 1 {
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
