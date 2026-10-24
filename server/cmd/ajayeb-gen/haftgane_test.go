package main

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// haftganeGoldenDir holds Haftgane's catalog goldens: the record tables,
// which the csv format must reproduce byte for byte, and the prose dump.
var haftganeGoldenDir = filepath.Join("..", "..", "internal", "haftgane", "catalog", "testdata")

var updateProse = flag.Bool("update", false, "rewrite Haftgane's catalog.golden.txt from the prose format")

func haftganeGolden(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(haftganeGoldenDir, name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestHaftganeCSVMatchesGoldens(t *testing.T) {
	dir := t.TempDir()
	if stdout, _ := runOK(t, "catalog", "-game", "haftgane", "-out", dir); stdout != "" {
		t.Errorf("stdout = %q, want nothing with -out", stdout)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	if strings.Join(names, " ") != "boards.csv cards.csv wonders.csv" {
		t.Errorf("files = %v", names)
	}
	var all []byte
	for _, table := range haftganeTables {
		got, err := os.ReadFile(filepath.Join(dir, table+".csv"))
		if err != nil {
			t.Fatal(err)
		}
		want := haftganeGolden(t, table+".golden.csv")
		all = append(all, want...)
		if !bytes.Equal(got, want) {
			t.Errorf("%s.csv differs from %s.golden.csv", table, table)
		}
		if one, _ := runOK(t, "catalog", "-game", "haftgane", "-table", table); one != string(want) {
			t.Errorf("-table %s differs from its golden", table)
		}
	}
	if stdout, _ := runOK(t, "catalog", "-game", "haftgane", "-format", "csv"); stdout != string(all) {
		t.Error("stdout is not the three goldens one after the other")
	}
}

// TestHaftganeProse pins the prose dump in catalog.golden.txt; run with
// -update to rewrite it, then review the diff against docs/haftgane/rules.md.
func TestHaftganeProse(t *testing.T) {
	out, _ := runOK(t, "catalog", "-game", "haftgane", "-format", "prose")
	path := filepath.Join(haftganeGoldenDir, "catalog.golden.txt")
	if *updateProse {
		if err := os.WriteFile(path, []byte(out), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if out != string(haftganeGolden(t, "catalog.golden.txt")) {
		t.Errorf("the prose dump differs from %s; if the change is deliberate, rerun with -update and review the diff", path)
	}
	for _, want := range []string{
		"1. Clay Pit (clay-pit): Age I brown card, one copy, from 3 players. Cost: 1 coin.",
		"Produces 1 unit of your choice of clay or ore every turn. Neighbours may buy it.",
		"Produces 1 unit of your choice of glass, papyrus or cloth every turn. Neighbours may not buy it.",
		"You buy wood, clay, stone and ore from your left neighbour at 1 coin per unit instead of 2.",
		"Shares its name with the Age II Glassworks: a city that holds either may not construct the other.",
		"Free to construct if you already own East Trading Post or West Trading Post, which shows chain symbol 7.",
		"you gain 2 coins for each grey card in your city and your two neighbours' cities.",
		"worth 1 victory point for each brown, each grey and each purple card in your city, this card included.",
		"worth 7 victory points if every stage of your wonder side is built.",
		"Age III purple guild (N + 2 of the 10 guilds are drawn for N players)",
		"2. Babylon, the Hanging Gardens (babylon). Produces 1 wood every turn from the start, which neighbours may buy.",
		"Night side, stage 1 of 2. Cost: 2 stone. From this Age on, you keep the card left over on the sixth turn",
		"Night side, stage 4 of 4. Cost: 4 stone and 1 papyrus. Worth 7 victory points at the end of the game.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("prose lacks %q", want)
		}
	}
	for _, bad := range []string{"%!", "Unknown effect", "unknown set", "none"} {
		if strings.Contains(out, bad) {
			t.Errorf("prose contains %q", bad)
		}
	}
	// 85 effect lines (five designs have two effects), 6 twin lines, 24
	// chain-in and 25 chain-out lines, and 41 stages.
	if n := strings.Count(out, "\n   - "); n != 85+6+24+25+41 {
		t.Errorf("%d item lines, want %d", n, 85+6+24+25+41)
	}
	dir := t.TempDir()
	runOK(t, "catalog", "-game", "haftgane", "-format", "prose", "-out", dir)
	if b, err := os.ReadFile(filepath.Join(dir, "catalog.txt")); err != nil || string(b) != out {
		t.Errorf("catalog.txt differs from stdout (%v)", err)
	}
}

func TestHaftganeCommandLine(t *testing.T) {
	cases := []struct {
		args []string
		code int
		msg  string
	}{
		{[]string{"catalog", "-game", "chess"}, 2, `unknown game "chess"`},
		{[]string{"catalog", "-game", "haftgane", "-format", "json"}, 2, `format "json" is not available for -game haftgane`},
		{[]string{"catalog", "-game", "haftgane", "-table", "tokens"}, 2, `unknown table "tokens" (want cards, wonders, boards)`},
		{[]string{"catalog", "-game", "haftgane", "-format", "prose", "-table", "cards"}, 2, "-table applies only to -format csv"},
		{[]string{"catalog", "-table", "boards"}, 2, `unknown table "boards" (want cards, wonders, tokens)`},
	}
	for _, c := range cases {
		var out, errOut bytes.Buffer
		code := run(c.args, &out, &errOut)
		if code != c.code || !strings.Contains(errOut.String(), c.msg) {
			t.Errorf("run %q = %d, stderr %q; want %d and %q", c.args, code, errOut.String(), c.code, c.msg)
		}
	}
	for _, format := range haftganeFormats {
		a, _ := runOK(t, "catalog", "-game", "haftgane", "-format", format)
		b, _ := runOK(t, "catalog", "-game", "haftgane", "-format", format)
		if a != b || a == "" {
			t.Errorf("format %s is not deterministic", format)
		}
	}
}
