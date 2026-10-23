package puritycheck

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestConfiguredRules(t *testing.T) {
	config := Config{
		Imports:  []ImportRule{{Path: "clock"}, {Path: "random", Exact: true}, {Path: "random/v2", AllowedFiles: []string{"generator.go"}}},
		MapFiles: []string{"dto.go"},
	}
	cases := []struct {
		rel, src string
		bad      int
	}{
		{"rules.go", `package rules; import "clock/sub"`, 1},
		{"rules.go", `package rules; import "clockwork"`, 0},
		{"rules.go", `package rules; import "random/other"`, 0},
		{"generator.go", `package rules; import "random/v2"`, 0},
		{"nested/generator.go", `package rules; import "random/v2"`, 1},
		{"dto.go", `package rules; var m map[int]bool`, 0},
		{"nested/dto.go", `package rules; var m map[int]bool`, 1},
		{"rules.go", `package rules; import`, 1},
	}
	for _, c := range cases {
		if bad := config.Violations(c.rel, []byte(c.src)); len(bad) != c.bad {
			t.Errorf("%s %q: got %v, want %d", c.rel, c.src, bad, c.bad)
		}
	}
}

func TestCannotImportTestHelper(t *testing.T) {
	for _, path := range []string{
		"github.com/NightMachinery/ajayeb/server/internal/enginekit/puritycheck",
		"github.com/NightMachinery/ajayeb/server/internal/enginekit/puritycheck/nested",
	} {
		// Even explicitly allowing the helper's file cannot override the test-only
		// boundary; production source is checked without resolving its imports.
		config := Config{Imports: []ImportRule{{Path: path, AllowedFiles: []string{"rules.go"}}}}
		bad := config.Violations("rules.go", []byte(`package rules; import helper "`+path+`"`))
		if len(bad) != 1 || !strings.Contains(bad[0], "test helper only") {
			t.Fatalf("test helper import passed: %v", bad)
		}
	}
}

func TestScan(t *testing.T) {
	root := t.TempDir()
	for path, src := range map[string]string{
		"rules/state.go": "package rules", "catalog/data.go": "package catalog",
		"rules/checks_test.go": "not Go source", "rules/testdata/broken.go": "not Go source",
		"catalog/testdata/broken.go": "not Go source", "rules/notes.txt": "not Go source",
		"foreign/ignored.go": "not Go source",
	} {
		path = filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	config := Config{Packages: []string{"catalog", "rules", "catalog"}}
	report, err := config.Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(report.Files, []string{"catalog/data.go", "rules/state.go"}) || len(report.Violations) != 0 {
		t.Fatalf("scan: %+v", report)
	}
	if _, err := (Config{}).Scan(root); err == nil {
		t.Fatal("empty package list passed")
	}
	if _, err := (Config{Packages: []string{"missing"}}).Scan(root); err == nil {
		t.Fatal("missing package passed")
	}
	// Syntax errors, unlike excluded test fixtures, fail closed in real sources.
	if err := os.WriteFile(filepath.Join(root, "rules/state.go"), []byte("package rules; func broken("), 0o600); err != nil {
		t.Fatal(err)
	}
	report, err = config.Scan(root)
	if err != nil || len(report.Violations) != 1 {
		t.Fatalf("broken source passed: %+v, %v", report, err)
	}
	// An unreadable .go entry must fail the scan, not look like an empty file.
	if err := os.Symlink(filepath.Join(root, "missing.go"), filepath.Join(root, "rules/unreadable.go")); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Scan(root); err == nil {
		t.Fatal("unreadable source passed")
	}
}

func TestFixedShapeAllowedValues(t *testing.T) {
	type fixed struct {
		Flag       bool
		Signed8    int8
		Signed16   int16
		Signed32   int32
		Signed64   int64
		Unsigned8  uint8
		Unsigned16 uint16
		Unsigned32 uint32
		Unsigned64 uint64
		Nested     [2]struct{ Values [3]uint16 }
	}
	if bad := FixedShapeViolations(reflect.TypeFor[fixed](), "fixed"); len(bad) != 0 {
		t.Fatal(bad)
	}
}
