package catalog_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/engine/catalog"
	"github.com/NightMachinery/ajayeb/server/internal/enginekit/catalogdiff"
)

// The three-way data check. Two verifiers built the catalog's tables blind,
// without seeing our Go data: V1 from the rulebook's card lists and V2 from
// the Board Game Arena data cross-checked against 7wd-io. Their tables live in
// testdata/verify/<source>_<table>.csv. TestVerify renders our tables through
// the same code path as `ajayeb-gen catalog -format csv`, joins the rows of
// all three sources by a normalised name and compares every field. The
// mechanics live in enginekit/catalogdiff, shared with Haftgane's catalog;
// this file holds Duel's schema.
//
// The verifier files are regression fixtures, not goldens: no flag rewrites
// them. A row in them changes only by a ruling recorded in
// docs/rules/data-rulings.md.

// verifyConfig is Duel's schema: the verifiers, in the order mismatches
// print them, and the tables every source provides, each joined by its
// normalised name.
func verifyConfig() catalogdiff.Config {
	byName := []catalogdiff.KeyColumn{{Name: "name", Normalize: catalogdiff.NameKey}}
	return catalogdiff.Config{
		Dir:     filepath.Join("testdata", "verify"),
		Sources: []string{"v1", "v2"},
		Tables: []catalogdiff.Spec{
			{Name: "cards", Key: byName},
			{Name: "wonders", Key: byName},
			{Name: "tokens", Key: byName},
		},
		Ours: verifyOurs,
	}
}

// verifyOurs renders our tables exactly as ajayeb-gen does, CSV bytes
// included, and parses them back.
func verifyOurs() (map[string]catalogdiff.Table, error) {
	tables, err := catalog.RecordTables()
	if err != nil {
		return nil, err
	}
	out := map[string]catalogdiff.Table{}
	for _, tb := range tables {
		var buf bytes.Buffer
		if err := tb.WriteCSV(&buf); err != nil {
			return nil, err
		}
		vt, err := catalogdiff.Parse("ours", buf.Bytes())
		if err != nil {
			return nil, err
		}
		out[tb.Name] = vt
	}
	return out, nil
}

// TestVerify is the three-way diff. It prints every mismatch as table, item,
// field and each source's value, and passes only when there are none.
func TestVerify(t *testing.T) {
	items := catalogdiff.Run(t, verifyConfig())
	if items != catalog.NumCards+catalog.NumWonders+catalog.NumTokens {
		t.Errorf("compared %d items, want %d", items, catalog.NumCards+catalog.NumWonders+catalog.NumTokens)
	}
}
