// Package catalogdiff is the three-way catalog data check shared by the
// games' catalog tests. Two verifiers build each catalog table blind, without
// seeing the Go data; the check joins their rows with the catalog's own
// rendering of the table by a normalised key and compares every field.
//
// A game supplies its schema as a Config: the verifiers, the tables with the
// join key of each, and its own rendered tables. Its fixtures, its expected
// row counts and its rulings stay with the game. The package reads files, so
// only tests may import it (TestOnlyTestsImport checks this).
package catalogdiff

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Table is one source's version of one table.
type Table struct {
	Source string
	Header []string
	Rows   [][]string
}

// Mismatch is one disagreement: the item (our name when we have the item),
// the field, and each source's value, ours first.
type Mismatch struct {
	Table, Item, Field string
	Values             []string
}

func (m Mismatch) String() string {
	return fmt.Sprintf("%s: %s: %s: %s", m.Table, m.Item, m.Field, strings.Join(m.Values, " | "))
}

// KeyColumn is one column of a join key. Normalize maps the column's cell to
// its part of the key; nil keeps the cell as written.
type KeyColumn struct {
	Name      string
	Normalize func(string) string
}

// Spec is one table of a game's catalog and the key its rows join by, such
// as the name alone, or the name and the Age.
type Spec struct {
	Name string
	Key  []KeyColumn
}

// Config is one game's check.
type Config struct {
	// Dir holds the verifier tables, <source>_<table>.csv, and nothing else.
	Dir string
	// Sources are the verifiers, in the order mismatches print them.
	Sources []string
	// Tables are the tables every source provides.
	Tables []Spec
	// Ours renders the catalog's own tables, by table name, the way the
	// game's generator writes them.
	Ours func() (map[string]Table, error)
}

// NameKey is the usual normalisation of a name: lower case, without a
// leading "The", and with spaces and hyphens removed. "The Great Library",
// "great library" and "Great-Library" all become "greatlibrary", while
// "Theater" keeps its "the".
func NameKey(name string) string {
	words := strings.Fields(strings.ToLower(name))
	if len(words) > 1 && words[0] == "the" {
		words = words[1:]
	}
	return strings.ReplaceAll(strings.Join(words, ""), "-", "")
}

// Parse reads a CSV table: the header row, then the item rows.
func Parse(source string, data []byte) (Table, error) {
	records, err := csv.NewReader(bytes.NewReader(data)).ReadAll()
	if err != nil {
		return Table{}, fmt.Errorf("%s: %w", source, err)
	}
	if len(records) == 0 {
		return Table{}, fmt.Errorf("%s: no header row", source)
	}
	return Table{Source: source, Header: records[0], Rows: records[1:]}, nil
}

// Run is the whole check. It insists that cfg.Dir holds exactly the files
// the comparison reads, reads every verifier table, compares each with ours,
// reports every mismatch through t as table, item, field and each source's
// value, and returns the number of our rows compared, which the game checks
// against its own counts.
func Run(t testing.TB, cfg Config) int {
	t.Helper()
	want := []string{}
	for _, src := range cfg.Sources {
		for _, tbl := range cfg.Tables {
			want = append(want, src+"_"+tbl.Name+".csv")
		}
	}
	slices.Sort(want)
	entries, err := os.ReadDir(cfg.Dir)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	if !slices.Equal(got, want) {
		t.Fatalf("%s holds %q, want exactly %q", cfg.Dir, got, want)
	}

	ours, err := cfg.Ours()
	if err != nil {
		t.Fatal(err)
	}
	total, items := 0, 0
	for _, tbl := range cfg.Tables {
		o, ok := ours[tbl.Name]
		if !ok {
			t.Fatalf("our tables have no %s table", tbl.Name)
		}
		srcs := []Table{o}
		for _, src := range cfg.Sources {
			path := filepath.Join(cfg.Dir, src+"_"+tbl.Name+".csv")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.HasSuffix(data, []byte("\n")) || bytes.Contains(data, []byte("\r")) {
				t.Errorf("%s must use bare newlines and end with one", path)
			}
			vt, err := Parse(src, data)
			if err != nil {
				t.Fatal(err)
			}
			srcs = append(srcs, vt)
		}
		for _, m := range Diff(tbl.Name, tbl.Key, srcs) {
			t.Errorf("mismatch %s", m)
			total++
		}
		items += len(o.Rows)
	}
	t.Logf("verify: %d items in %d tables against %d verifiers: %d mismatches",
		items, len(cfg.Tables), len(cfg.Sources), total)
	return items
}

// Diff joins the sources' rows by key and returns every disagreement.
// srcs[0] is ours and fixes the column order and the item order; items only
// the verifiers have come after ours. A column a source lacks, a row a
// source lacks and a key a source lists twice are mismatches too, so nothing
// can drop out of the comparison unnoticed. An item is labelled by its key
// cells as written, joined with "/", and a key by its normalised parts.
func Diff(table string, key []KeyColumn, srcs []Table) []Mismatch {
	var out []Mismatch
	report := func(item, field string, values []string) {
		out = append(out, Mismatch{Table: table, Item: item, Field: field, Values: values})
	}

	// The columns of each source, by name.
	cols := make([]map[string]int, len(srcs))
	for i, s := range srcs {
		cols[i] = map[string]int{}
		for j, h := range s.Header {
			cols[i][h] = j
		}
	}
	for _, s := range srcs[1:] {
		if !slices.Equal(s.Header, srcs[0].Header) {
			report("(header)", "columns", []string{
				srcs[0].Source + "=" + strings.Join(srcs[0].Header, ","),
				s.Source + "=" + strings.Join(s.Header, ","),
			})
		}
	}

	// The key columns of each source; nil when it lacks one of them.
	keyCols := make([][]int, len(srcs))
	label := func(i int, row []string) string {
		cells := make([]string, len(key))
		for j, c := range keyCols[i] {
			cells[j] = row[c]
		}
		return strings.Join(cells, "/")
	}

	// Every source's rows, by join key; the item order is ours, then new
	// keys in source order. A map key joins the normalised parts with a
	// byte no cell holds.
	byKey := make([]map[string][]string, len(srcs))
	var order []string
	seen := map[string]bool{}
	for i, s := range srcs {
		byKey[i] = map[string][]string{}
		idx := make([]int, 0, len(key))
		for _, k := range key {
			if j, ok := cols[i][k.Name]; ok {
				idx = append(idx, j)
			} else {
				report("(header)", k.Name, []string{s.Source + " has no " + k.Name + " column"})
			}
		}
		if len(idx) < len(key) {
			continue
		}
		keyCols[i] = idx
	rows:
		for _, row := range s.Rows {
			parts := make([]string, len(key))
			for j, k := range key {
				if idx[j] >= len(row) {
					report("(row)", k.Name, []string{s.Source + "=short row " + strings.Join(row, ",")})
					continue rows
				}
				parts[j] = row[idx[j]]
				if k.Normalize != nil {
					parts[j] = k.Normalize(parts[j])
				}
			}
			mk := strings.Join(parts, "\x00")
			if _, dup := byKey[i][mk]; dup {
				report(label(i, row), "(duplicate)", []string{s.Source + " lists " + strings.Join(parts, "/") + " twice"})
				continue
			}
			byKey[i][mk] = row
			if !seen[mk] {
				seen[mk] = true
				order = append(order, mk)
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
	for _, mk := range order {
		var item string
		for i := range srcs {
			if row, ok := byKey[i][mk]; ok {
				item = label(i, row)
				break
			}
		}
		presence := make([]string, len(srcs))
		missing := false
		for i, s := range srcs {
			presence[i] = s.Source + "=present"
			if _, ok := byKey[i][mk]; !ok {
				presence[i] = s.Source + "=missing"
				missing = true
			}
		}
		if missing {
			report(item, "(row)", presence)
			continue
		}
		for _, col := range srcs[0].Header {
			vals := make([]string, len(srcs))
			for i := range srcs {
				vals[i] = cell(i, byKey[i][mk], col)
			}
			if slices.ContainsFunc(vals[1:], func(v string) bool { return v != vals[0] }) {
				labelledVals := make([]string, len(vals))
				for i, v := range vals {
					labelledVals[i] = fmt.Sprintf("%s=%q", srcs[i].Source, v)
				}
				report(item, col, labelledVals)
			}
		}
	}
	return out
}
