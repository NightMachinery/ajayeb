package main

import (
	"bytes"
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	hg "github.com/NightMachinery/ajayeb/server/internal/haftgane/catalog"
)

// hgEntries reads back every `'slug': {json},` line of a block that starts
// with the given declaration line.
func hgEntries(t *testing.T, src, start string) map[string]string {
	t.Helper()
	at := strings.Index(src, start)
	if at < 0 {
		t.Fatalf("no %q in the module", start)
	}
	line := regexp.MustCompile(`^  '([a-z0-9-]+)': (\{.*\}),$`)
	out := map[string]string{}
	for _, l := range strings.Split(src[at+len(start):], "\n")[1:] {
		if l == "};" {
			return out
		}
		m := line.FindStringSubmatch(l)
		if m == nil {
			t.Fatalf("unexpected line %q", l)
		}
		out[m[1]] = m[2]
	}
	t.Fatalf("block %q is not closed", start)
	return nil
}

func TestHaftganeCatalogTS(t *testing.T) {
	data, err := haftganeCatalogTS()
	if err != nil {
		t.Fatal(err)
	}
	again, _ := haftganeCatalogTS()
	if !bytes.Equal(data, again) {
		t.Fatal("haftganeCatalogTS is not deterministic")
	}
	src := string(data)
	designs := hgEntries(t, src, "export const hgDesigns")
	if len(designs) != hg.NumDesigns {
		t.Fatalf("%d designs, want %d", len(designs), hg.NumDesigns)
	}
	for id := hg.DesignID(1); id <= hg.NumDesigns; id++ {
		d, _ := hg.DesignByID(id)
		raw, ok := designs[d.Slug]
		if !ok {
			t.Errorf("design %s is missing", d.Slug)
			continue
		}
		var j hgJSONDesign
		if err := json.Unmarshal([]byte(raw), &j); err != nil {
			t.Errorf("design %s: %v", d.Slug, err)
			continue
		}
		if j.Age != int(d.Age) || j.Colour != d.Color.String() || len(j.Effects) != len(d.Effects) {
			t.Errorf("design %s reads back as %+v", d.Slug, j)
		}
		if len(j.Cost.Res) != int(d.Cost.Units()) || j.Cost.Coins != int(d.Cost.Coins) {
			t.Errorf("design %s cost %+v, want %d units and %d coins", d.Slug, j.Cost, d.Cost.Units(), d.Cost.Coins)
		}
		// Codes only: the module names nothing in English.
		if strings.Contains(raw, d.Name) {
			t.Errorf("design %s carries its name: %s", d.Slug, raw)
		}
	}
	boards := hgEntries(t, src, "export const hgBoards")
	if len(boards) != hg.NumBoards {
		t.Fatalf("%d boards, want %d", len(boards), hg.NumBoards)
	}
	for id := hg.BoardID(1); id <= hg.NumBoards; id++ {
		b, _ := hg.BoardByID(id)
		var j hgJSONBoard
		if err := json.Unmarshal([]byte(boards[b.Slug]), &j); err != nil {
			t.Fatalf("board %s: %v", b.Slug, err)
		}
		if j.Start != b.Start.String() || len(j.Day) != int(b.Sides[0].NumStages) || len(j.Night) != int(b.Sides[1].NumStages) {
			t.Errorf("board %s reads back as %+v", b.Slug, j)
		}
	}
}

// Each chain is named after the design it makes free: the design with that
// slug, and only it, takes the chain in.
func TestHaftganeChainNames(t *testing.T) {
	for c := hg.Chain(1); c <= hg.NumChains; c++ {
		name, err := hgChainSlug(c)
		if err != nil {
			t.Fatal(err)
		}
		var takers []string
		for id := hg.DesignID(1); id <= hg.NumDesigns; id++ {
			d, _ := hg.DesignByID(id)
			if d.ChainIn == c {
				takers = append(takers, d.Slug)
			}
		}
		if len(takers) != 1 || takers[0] != name {
			t.Errorf("chain %d is named %q, but %v take it in", c, name, takers)
		}
	}
}

func TestHaftganeCatalogTSCommand(t *testing.T) {
	stdout, _ := runOK(t, "catalog", "-game", "haftgane", "-format", "ts")
	want, _ := haftganeCatalogTS()
	if stdout != string(want) {
		t.Error("catalog -game haftgane -format ts differs from haftganeCatalogTS()")
	}
	dir := t.TempDir()
	runOK(t, "catalog", "-game", "haftgane", "-format", "ts", "-out", dir)
	if entries, _ := os.ReadDir(dir); len(entries) != 1 || entries[0].Name() != "haftgane-catalog.gen.ts" {
		t.Errorf("-format ts -out wrote %v", entries)
	}
}
