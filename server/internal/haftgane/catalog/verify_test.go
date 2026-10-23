package catalog_test

import (
	"bytes"
	"path/filepath"
	"slices"
	"testing"

	"github.com/NightMachinery/ajayeb/server/internal/enginekit/catalogdiff"
	"github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// The three-way data check. In M13 two verifiers built the catalog's tables
// blind, without seeing our drafts or each other's work: "wiki" from the
// second-edition fan wiki checked against second-edition implementations, and
// "publisher" from the publisher's card and chain sheet, effects sheet and
// rulebook. Their tables live in testdata/verify/<source>/<source>_<table>.csv.
// TestVerify renders our record tables through the same code path as
// `ajayeb-gen catalog -game haftgane`, joins every source's rows (designs by
// name and Age, stages by board, side and stage number) and compares every
// field. The mechanics live in enginekit/catalogdiff, shared with Duel.
//
// The publisher's sources print no card costs and draw no night sides, so
// its pass compares a projection of our tables: no cost column for designs,
// and the day sides only (with their costs, which the rulebook draws).
// Everything else, and every field of the wiki's tables, is compared in full.
//
// The verifier files are regression fixtures, not goldens: no flag rewrites
// them. A row in them changes only by a ruling or normalisation recorded in
// docs/haftgane/data-rulings.md.

// verifyColumns are the columns of the M13 blind tables, without their
// provenance columns.
var verifyColumns = map[string][]string{
	"cards":   {"age", "name", "colour", "player_counts", "cost", "chain_from", "chain_to", "effects"},
	"wonders": {"board", "side", "starting_resource", "stage", "cost", "effects"},
}

// A verifier and the part of our tables it speaks to.
type verifier struct {
	source    string
	dropCols  map[string][]string // per table, our columns its sources cannot show
	dayOnly   bool                // its sources show only day sides
	wantItems int
}

var verifiers = []verifier{
	{source: "wiki", wantItems: catalog.NumDesigns + 41},
	{source: "publisher", dropCols: map[string][]string{"cards": {"cost"}}, dayOnly: true, wantItems: catalog.NumDesigns + 21},
}

func verifyConfig(v verifier) catalogdiff.Config {
	byNameAge := []catalogdiff.KeyColumn{{Name: "age"}, {Name: "name", Normalize: catalogdiff.NameKey}}
	byStage := []catalogdiff.KeyColumn{{Name: "board"}, {Name: "side"}, {Name: "stage"}}
	return catalogdiff.Config{
		Dir:     filepath.Join("testdata", "verify", v.source),
		Sources: []string{v.source},
		Tables:  []catalogdiff.Spec{{Name: "cards", Key: byNameAge}, {Name: "wonders", Key: byStage}},
		Ours:    func() (map[string]catalogdiff.Table, error) { return verifyOurs(v) },
	}
}

// verifyOurs renders our tables exactly as ajayeb-gen does, CSV bytes
// included, parses them back and projects them onto what v can see.
func verifyOurs(v verifier) (map[string]catalogdiff.Table, error) {
	tables, err := catalog.RecordTables()
	if err != nil {
		return nil, err
	}
	out := map[string]catalogdiff.Table{}
	for _, tb := range tables {
		cols, ok := verifyColumns[tb.Name]
		if !ok {
			continue
		}
		var buf bytes.Buffer
		if err := tb.WriteCSV(&buf); err != nil {
			return nil, err
		}
		full, err := catalogdiff.Parse("ours", buf.Bytes())
		if err != nil {
			return nil, err
		}
		out[tb.Name] = project(full, cols, v.dropCols[tb.Name], v.dayOnly)
	}
	return out, nil
}

func project(full catalogdiff.Table, cols, drop []string, dayOnly bool) catalogdiff.Table {
	index := map[string]int{}
	for i, h := range full.Header {
		index[h] = i
	}
	p := catalogdiff.Table{Source: full.Source}
	for _, c := range cols {
		if !slices.Contains(drop, c) {
			p.Header = append(p.Header, c)
		}
	}
	for _, row := range full.Rows {
		if side, ok := index["side"]; ok && dayOnly && row[side] != "day" {
			continue
		}
		var r []string
		for _, c := range p.Header {
			r = append(r, row[index[c]])
		}
		p.Rows = append(p.Rows, r)
	}
	return p
}

// TestVerify is the three-way diff. It prints every mismatch as table, item,
// field and each source's value, and passes only when there are none.
func TestVerify(t *testing.T) {
	for _, v := range verifiers {
		t.Run(v.source, func(t *testing.T) {
			if items := catalogdiff.Run(t, verifyConfig(v)); items != v.wantItems {
				t.Errorf("compared %d items, want %d", items, v.wantItems)
			}
		})
	}
}
