package catalog

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The record tables are pinned byte for byte in testdata/*.golden.csv; their
// row order is the design id order, which is part of the rules version. Run
// with -update to rewrite them, then review the diff.
var update = flag.Bool("update", false, "rewrite testdata/*.golden.csv from the catalog")

// TestGoldenCSV compares the record tables with testdata/*.golden.csv.
// ajayeb-gen catalog -game haftgane writes the same bytes.
func TestGoldenCSV(t *testing.T) {
	tables, err := RecordTables()
	if err != nil {
		t.Fatal(err)
	}
	for _, tb := range tables {
		var buf bytes.Buffer
		if err := tb.WriteCSV(&buf); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join("testdata", tb.Name+".golden.csv")
		if *update {
			if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%v (run go test -run TestGoldenCSV -update to create it)", err)
		}
		if !bytes.Equal(buf.Bytes(), want) {
			got := strings.Split(buf.String(), "\n")
			exp := strings.Split(string(want), "\n")
			for i := range max(len(got), len(exp)) {
				var g, e string
				if i < len(got) {
					g = got[i]
				}
				if i < len(exp) {
					e = exp[i]
				}
				if g != e {
					t.Errorf("%s line %d:\n got  %q\n want %q", path, i+1, g, e)
				}
			}
			t.Errorf("%s differs from the catalog; if the change is deliberate, rerun with -update and review the diff", path)
		}
		if !strings.HasSuffix(buf.String(), "\n") || strings.Contains(buf.String(), "\r") {
			t.Errorf("%s must use bare newlines and end with one", path)
		}
	}
}
