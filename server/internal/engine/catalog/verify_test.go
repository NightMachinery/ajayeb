package catalog_test

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
)

// The three-way data check. Two verifiers built the catalog's tables blind,
// without seeing our Go data: V1 from the rulebook's card lists and V2 from
// the Board Game Arena data cross-checked against 7wd-io. Their tables live in
// testdata/verify/<source>_<table>.csv. TestVerify renders our tables through
// the same code path as `ajayeb-gen catalog -format csv`, joins the rows of
// all three sources by a normalised name and compares every field.
//
// The verifier files are regression fixtures, not goldens: no flag rewrites
// them. A row in them changes only by a ruling recorded in
// docs/rules/data-rulings.md.

// verifySources are the verifiers, in the order mismatches print them.
var verifySources = []string{"v1", "v2"}

// verifyTableNames are the tables every source provides.
var verifyTableNames = []string{"cards", "wonders", "tokens"}

// verifyTable is one source's version of one table.
type verifyTable struct {
	source string
	header []string
	rows   [][]string
}

// verifyMismatch is one disagreement: the item (our name when we have the
// item), the field, and each source's value, ours first.
type verifyMismatch struct {
	table, item, field string
	values             []string
}

func (m verifyMismatch) String() string {
	return fmt.Sprintf("%s: %s: %s: %s", m.table, m.item, m.field, strings.Join(m.values, " | "))
}

// verifyKey is the join key of a name: lower case, without a leading "The",
// and with spaces and hyphens removed. "The Great Library", "great library"
// and "Great-Library" all become "greatlibrary", while "Theater" keeps its
// "the".
func verifyKey(name string) string {
	words := strings.Fields(strings.ToLower(name))
	if len(words) > 1 && words[0] == "the" {
		words = words[1:]
	}
	return strings.ReplaceAll(strings.Join(words, ""), "-", "")
}

// verifyParse reads a CSV table: the header row, then the item rows.
func verifyParse(source string, data []byte) (verifyTable, error) {
	records, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
	if err != nil {
		return verifyTable{}, fmt.Errorf("%s: %w", source, err)
	}
	if len(records) == 0 {
		return verifyTable{}, fmt.Errorf("%s: no header row", source)
	}
	return verifyTable{source: source, header: records[0], rows: records[1:]}, nil
}

// verifyOurs renders our tables exactly as ajayeb-gen does, CSV bytes
// included, and parses them back.
func verifyOurs(t *testing.T) map[string]verifyTable {
	t.Helper()
	tables, err := catalog.RecordTables()
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]verifyTable{}
	for _, tb := range tables {
		var buf bytes.Buffer
		if err := tb.WriteCSV(&buf); err != nil {
			t.Fatal(err)
		}
		vt, err := verifyParse("ours", buf.Bytes())
		if err != nil {
			t.Fatal(err)
		}
		out[tb.Name] = vt
	}
	return out
}

// verifyDiff joins the sources' rows by verifyKey of the name column and
// returns every disagreement. srcs[0] is ours and fixes the column order and
// the item order; items only the verifiers have come after ours. A column a
// source lacks, a row a source lacks and a name a source lists twice are
// mismatches too, so nothing can drop out of the comparison unnoticed.
func verifyDiff(table string, srcs []verifyTable) []verifyMismatch {
	var out []verifyMismatch
	report := func(item, field string, values []string) {
		out = append(out, verifyMismatch{table: table, item: item, field: field, values: values})
	}

	// The columns of each source, by name.
	cols := make([]map[string]int, len(srcs))
	for i, s := range srcs {
		cols[i] = map[string]int{}
		for j, h := range s.header {
			cols[i][h] = j
		}
	}
	for _, s := range srcs[1:] {
		if !slices.Equal(s.header, srcs[0].header) {
			report("(header)", "columns", []string{
				srcs[0].source + "=" + strings.Join(srcs[0].header, ","),
				s.source + "=" + strings.Join(s.header, ","),
			})
		}
	}
	nameCol := func(i int) (int, bool) {
		j, ok := cols[i]["name"]
		return j, ok
	}

	// Every source's rows, by join key; the item order is ours, then new
	// keys in source order.
	byKey := make([]map[string][]string, len(srcs))
	var order []string
	seen := map[string]bool{}
	for i, s := range srcs {
		byKey[i] = map[string][]string{}
		nc, ok := nameCol(i)
		if !ok {
			report("(header)", "name", []string{s.source + " has no name column"})
			continue
		}
		for _, row := range s.rows {
			if nc >= len(row) {
				report("(row)", "name", []string{s.source + "=short row " + strings.Join(row, ",")})
				continue
			}
			k := verifyKey(row[nc])
			if _, dup := byKey[i][k]; dup {
				report(row[nc], "(duplicate)", []string{s.source + " lists " + k + " twice"})
				continue
			}
			byKey[i][k] = row
			if !seen[k] {
				seen[k] = true
				order = append(order, k)
			}
		}
	}

	cell := func(i int, row []string, col string) string {
		j, ok := cols[i][col]
		if !ok {
			return "(no column)"
		}
		if j >= len(row) {
			return "(short row)"
		}
		return row[j]
	}
	for _, k := range order {
		item := k
		for i := range srcs {
			if row, ok := byKey[i][k]; ok {
				nc, _ := nameCol(i)
				item = row[nc]
				break
			}
		}
		presence := make([]string, len(srcs))
		missing := false
		for i, s := range srcs {
			presence[i] = s.source + "=present"
			if _, ok := byKey[i][k]; !ok {
				presence[i] = s.source + "=missing"
				missing = true
			}
		}
		if missing {
			report(item, "(row)", presence)
			continue
		}
		for _, col := range srcs[0].header {
			vals := make([]string, len(srcs))
			for i := range srcs {
				vals[i] = cell(i, byKey[i][k], col)
			}
			if slices.ContainsFunc(vals[1:], func(v string) bool { return v != vals[0] }) {
				labelledVals := make([]string, len(vals))
				for i, v := range vals {
					labelledVals[i] = fmt.Sprintf("%s=%q", srcs[i].source, v)
				}
				report(item, col, labelledVals)
			}
		}
	}
	return out
}

// TestVerify is the three-way diff. It prints every mismatch as table, item,
// field and each source's value, and passes only when there are none.
func TestVerify(t *testing.T) {
	// Every file in testdata/verify must be one the comparison reads.
	want := []string{}
	for _, src := range verifySources {
		for _, tbl := range verifyTableNames {
			want = append(want, src+"_"+tbl+".csv")
		}
	}
	slices.Sort(want)
	entries, err := os.ReadDir(filepath.Join("testdata", "verify"))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	if !slices.Equal(got, want) {
		t.Fatalf("testdata/verify holds %q, want exactly %q", got, want)
	}

	ours := verifyOurs(t)
	total, items := 0, 0
	for _, tbl := range verifyTableNames {
		o, ok := ours[tbl]
		if !ok {
			t.Fatalf("catalog.RecordTables has no %s table", tbl)
		}
		srcs := []verifyTable{o}
		for _, src := range verifySources {
			path := filepath.Join("testdata", "verify", src+"_"+tbl+".csv")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.HasSuffix(data, []byte("\n")) || bytes.Contains(data, []byte("\r")) {
				t.Errorf("%s must use bare newlines and end with one", path)
			}
			vt, err := verifyParse(src, data)
			if err != nil {
				t.Fatal(err)
			}
			srcs = append(srcs, vt)
		}
		for _, m := range verifyDiff(tbl, srcs) {
			t.Errorf("mismatch %s", m)
			total++
		}
		items += len(o.rows)
	}
	t.Logf("verify: %d items in %d tables against %d verifiers: %d mismatches",
		items, len(verifyTableNames), len(verifySources), total)
	if items != catalog.NumCards+catalog.NumWonders+catalog.NumTokens {
		t.Errorf("compared %d items, want %d", items, catalog.NumCards+catalog.NumWonders+catalog.NumTokens)
	}
}

func TestVerifyKey(t *testing.T) {
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
		if got := verifyKey(c.in); got != c.want {
			t.Errorf("verifyKey(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestVerifyDiffReports checks that the diff sees every kind of
// disagreement, so a zero from TestVerify means agreement and not a blind
// comparison.
func TestVerifyDiffReports(t *testing.T) {
	header := []string{"id", "name", "vp"}
	tbl := func(source string, rows ...[]string) verifyTable {
		return verifyTable{source: source, header: header, rows: rows}
	}
	same := tbl("x", []string{"1", "Pyramids", "9"}, []string{"10", "Sphinx", "6"})
	if got := verifyDiff("w", []verifyTable{same, same, same}); len(got) != 0 {
		t.Fatalf("identical tables gave %v", got)
	}

	cases := []struct {
		name   string
		srcs   []verifyTable
		expect []string // the mismatches, as item/field
	}{
		{"changed cell",
			[]verifyTable{
				tbl("ours", []string{"10", "Sphinx", "6"}),
				tbl("v1", []string{"10", "Sphinx", ""}),
				tbl("v2", []string{"10", "Sphinx", "6"}),
			},
			[]string{"Sphinx/vp"}},
		{"join ignores a leading The, case, spaces and hyphens, but the name still differs",
			[]verifyTable{
				tbl("ours", []string{"1", "Great Library", "4"}),
				tbl("v1", []string{"1", "The Great-Library", "4"}),
				tbl("v2", []string{"1", "great library", "4"}),
			},
			[]string{"Great Library/name"}},
		{"row missing in one source and extra in another",
			[]verifyTable{
				tbl("ours", []string{"1", "Pyramids", "9"}),
				tbl("v1", []string{"1", "Pyramids", "9"}, []string{"2", "Colossus", "3"}),
				tbl("v2", []string{"2", "Colossus", "3"}),
			},
			[]string{"Pyramids/(row)", "Colossus/(row)"}},
		{"duplicate name",
			[]verifyTable{
				tbl("ours", []string{"1", "Pyramids", "9"}),
				tbl("v1", []string{"1", "Pyramids", "9"}, []string{"1", "The Pyramids", "9"}),
				tbl("v2", []string{"1", "Pyramids", "9"}),
			},
			[]string{"The Pyramids/(duplicate)"}},
		{"different columns",
			[]verifyTable{
				tbl("ours", []string{"1", "Pyramids", "9"}),
				{source: "v1", header: []string{"id", "name"}, rows: [][]string{{"1", "Pyramids"}}},
				tbl("v2", []string{"1", "Pyramids", "9"}),
			},
			[]string{"(header)/columns", "Pyramids/vp"}},
	}
	for _, c := range cases {
		var got []string
		for _, m := range verifyDiff("w", c.srcs) {
			got = append(got, m.item+"/"+m.field)
		}
		if !slices.Equal(got, c.expect) {
			t.Errorf("%s: mismatches %q, want %q", c.name, got, c.expect)
		}
	}
}
