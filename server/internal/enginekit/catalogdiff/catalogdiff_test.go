package catalogdiff

import (
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

func TestNameKey(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"The Great Library", "greatlibrary"},
		{"great library", "greatlibrary"},
		{"Great-Library", "greatlibrary"},
		{"  The   Sphinx ", "sphinx"},
		{"Glass-Blower", "glassblower"},
		{"Glass-blower", "glassblower"},
		{"Glassblower", "glassblower"},
		{"Chamber Of Commerce", "chamberofcommerce"},
		{"Theater", "theater"},
		{"Theology", "theology"},
		{"The", "the"},
	} {
		if got := NameKey(c.in); got != c.want {
			t.Errorf("NameKey(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// diffItems runs Diff and lists its mismatches as item/field.
func diffItems(key []KeyColumn, srcs []Table) []string {
	var got []string
	for _, m := range Diff("w", key, srcs) {
		got = append(got, m.Item+"/"+m.Field)
	}
	return got
}

// TestDiffReports checks that the diff sees every kind of disagreement, so a
// zero from a game's check means agreement and not a blind comparison.
func TestDiffReports(t *testing.T) {
	byName := []KeyColumn{{Name: "name", Normalize: NameKey}}
	header := []string{"id", "name", "vp"}
	tbl := func(source string, rows ...[]string) Table {
		return Table{Source: source, Header: header, Rows: rows}
	}
	same := tbl("x", []string{"1", "Pyramids", "9"}, []string{"10", "Sphinx", "6"})
	if got := Diff("w", byName, []Table{same, same, same}); len(got) != 0 {
		t.Fatalf("identical tables gave %v", got)
	}

	cases := []struct {
		name   string
		srcs   []Table
		expect []string // the mismatches, as item/field
	}{
		{"changed cell",
			[]Table{
				tbl("ours", []string{"10", "Sphinx", "6"}),
				tbl("v1", []string{"10", "Sphinx", ""}),
				tbl("v2", []string{"10", "Sphinx", "6"}),
			},
			[]string{"Sphinx/vp"}},
		{"join ignores a leading The, case, spaces and hyphens, but the name still differs",
			[]Table{
				tbl("ours", []string{"1", "Great Library", "4"}),
				tbl("v1", []string{"1", "The Great-Library", "4"}),
				tbl("v2", []string{"1", "great library", "4"}),
			},
			[]string{"Great Library/name"}},
		{"row missing in one source and extra in another",
			[]Table{
				tbl("ours", []string{"1", "Pyramids", "9"}),
				tbl("v1", []string{"1", "Pyramids", "9"}, []string{"2", "Colossus", "3"}),
				tbl("v2", []string{"2", "Colossus", "3"}),
			},
			[]string{"Pyramids/(row)", "Colossus/(row)"}},
		{"duplicate name",
			[]Table{
				tbl("ours", []string{"1", "Pyramids", "9"}),
				tbl("v1", []string{"1", "Pyramids", "9"}, []string{"1", "The Pyramids", "9"}),
				tbl("v2", []string{"1", "Pyramids", "9"}),
			},
			[]string{"The Pyramids/(duplicate)"}},
		{"different columns",
			[]Table{
				tbl("ours", []string{"1", "Pyramids", "9"}),
				{Source: "v1", Header: []string{"id", "name"}, Rows: [][]string{{"1", "Pyramids"}}},
				tbl("v2", []string{"1", "Pyramids", "9"}),
			},
			[]string{"(header)/columns", "Pyramids/vp"}},
	}
	for _, c := range cases {
		if got := diffItems(byName, c.srcs); !slices.Equal(got, c.expect) {
			t.Errorf("%s: mismatches %q, want %q", c.name, got, c.expect)
		}
	}
}

// TestDiffCompositeKey joins by a name and an Age, as a catalog whose names
// repeat across Ages needs: rows that share a name but not an Age are
// different items.
func TestDiffCompositeKey(t *testing.T) {
	byNameAge := []KeyColumn{{Name: "name", Normalize: NameKey}, {Name: "age"}}
	header := []string{"name", "age", "vp"}
	tbl := func(source string, rows ...[]string) Table {
		return Table{Source: source, Header: header, Rows: rows}
	}
	twins := tbl("x", []string{"Glassworks", "1", "0"}, []string{"Glassworks", "2", "0"})
	if got := Diff("w", byNameAge, []Table{twins, twins, twins}); len(got) != 0 {
		t.Fatalf("a name in two Ages gave %v", got)
	}

	cases := []struct {
		name   string
		srcs   []Table
		expect []string
		values []string // the values of the first mismatch, when checked
	}{
		{"a changed cell names the item by both key cells",
			[]Table{
				tbl("ours", []string{"Glassworks", "1", "0"}, []string{"Glassworks", "2", "0"}),
				tbl("v1", []string{"glass works", "1", "0"}, []string{"Glassworks", "2", "1"}),
				tbl("v2", []string{"Glassworks", "1", "0"}, []string{"Glassworks", "2", "0"}),
			},
			[]string{"Glassworks/1/name", "Glassworks/2/vp"},
			[]string{`ours="Glassworks"`, `v1="glass works"`, `v2="Glassworks"`}},
		{"the same name in another Age is a missing row, not a match",
			[]Table{
				tbl("ours", []string{"Glassworks", "1", "0"}),
				tbl("v1", []string{"Glassworks", "1", "0"}),
				tbl("v2", []string{"Glassworks", "2", "0"}),
			},
			[]string{"Glassworks/1/(row)", "Glassworks/2/(row)"},
			[]string{"ours=present", "v1=present", "v2=missing"}},
		{"a duplicate needs both parts equal after normalising",
			[]Table{
				tbl("ours", []string{"Glassworks", "1", "0"}),
				tbl("v1", []string{"Glassworks", "1", "0"}, []string{"The Glass-Works", "1", "0"}),
				tbl("v2", []string{"Glassworks", "1", "0"}),
			},
			[]string{"The Glass-Works/1/(duplicate)"},
			[]string{"v1 lists glassworks/1 twice"}},
		{"a source without one key column is reported and left out",
			[]Table{
				tbl("ours", []string{"Glassworks", "1", "0"}),
				{Source: "v1", Header: []string{"name", "vp"}, Rows: [][]string{{"Glassworks", "0"}}},
				tbl("v2", []string{"Glassworks", "1", "0"}),
			},
			[]string{"(header)/columns", "(header)/age", "Glassworks/1/(row)"},
			nil},
	}
	for _, c := range cases {
		ms := Diff("w", byNameAge, c.srcs)
		var got []string
		for _, m := range ms {
			got = append(got, m.Item+"/"+m.Field)
		}
		if !slices.Equal(got, c.expect) {
			t.Errorf("%s: mismatches %q, want %q", c.name, got, c.expect)
			continue
		}
		if c.values != nil && !slices.Equal(ms[0].Values, c.values) {
			t.Errorf("%s: values %q, want %q", c.name, ms[0].Values, c.values)
		}
	}
}

// thisPackage is the import path no non-test file may use.
const thisPackage = "github.com/NightMachinery/ajayeb/server/internal/enginekit/catalogdiff"

// nonTestImporters lists the non-test Go files under root that import this
// package, and how many files it read.
func nonTestImporters(root string) (bad []string, scanned int, err error) {
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); name == "testdata" || (strings.HasPrefix(name, ".") && path != root) {
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
		scanned++
		for _, imp := range f.Imports {
			if p, err := strconv.Unquote(imp.Path.Value); err == nil && p == thisPackage {
				bad = append(bad, filepath.ToSlash(path))
			}
		}
		return nil
	})
	return bad, scanned, err
}

// TestOnlyTestsImport keeps this file-reading helper out of every program
// and every pure package: only _test.go files may import it.
func TestOnlyTestsImport(t *testing.T) {
	// From internal/enginekit/catalogdiff, three levels up is the module root.
	bad, scanned, err := nonTestImporters(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if scanned < 100 {
		t.Fatalf("read only %d non-test Go files; the walk is not seeing the module", scanned)
	}
	if len(bad) > 0 {
		t.Errorf("non-test files import %s: %q", thisPackage, bad)
	}
}

// TestOnlyTestsImportCatches shows the walk finds an offending file.
func TestOnlyTestsImportCatches(t *testing.T) {
	dir := t.TempDir()
	write := func(name, src string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("ok_test.go", "package x\n\nimport _ \""+thisPackage+"\"\n")
	write("bad.go", "package x\n\nimport (\n\t\"fmt\"\n\tcd \""+thisPackage+"\"\n)\n")
	bad, scanned, err := nonTestImporters(dir)
	if err != nil {
		t.Fatal(err)
	}
	if scanned != 1 || len(bad) != 1 || !strings.HasSuffix(bad[0], "/bad.go") {
		t.Errorf("got bad=%q scanned=%d, want only bad.go of 1 file", bad, scanned)
	}
}
