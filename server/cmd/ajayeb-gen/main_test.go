package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// goldenDir holds the catalog's verification goldens, which the csv format
// must reproduce byte for byte.
var goldenDir = filepath.Join("..", "..", "internal", "engine", "catalog", "testdata")

func golden(t *testing.T, table string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(goldenDir, table+".golden.csv"))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// runOK runs the command and fails the test unless it exits 0.
func runOK(t *testing.T, args ...string) (stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	if code := run(args, &out, &errOut); code != 0 {
		t.Fatalf("run %q = %d, stderr:\n%s", args, code, errOut.String())
	}
	return out.String(), errOut.String()
}

func TestCSVOutMatchesGoldens(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new", "dir") // -out creates missing directories
	stdout, _ := runOK(t, "catalog", "-format", "csv", "-out", dir)
	if stdout != "" {
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
	if strings.Join(names, " ") != "cards.csv tokens.csv wonders.csv" {
		t.Errorf("files = %v", names)
	}
	for _, table := range tables {
		got, err := os.ReadFile(filepath.Join(dir, table+".csv"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, golden(t, table)) {
			t.Errorf("%s.csv differs from %s.golden.csv", table, table)
		}
	}
}

func TestCSVStdout(t *testing.T) {
	// The default format is csv, and all three tables go to stdout in order.
	all, _ := runOK(t, "catalog")
	want := string(golden(t, "cards")) + string(golden(t, "wonders")) + string(golden(t, "tokens"))
	if all != want {
		t.Error("stdout is not the three goldens one after the other")
	}
	for _, table := range tables {
		got, _ := runOK(t, "catalog", "-format", "csv", "-table", table)
		if got != string(golden(t, table)) {
			t.Errorf("-table %s differs from its golden", table)
		}
	}
	dir := t.TempDir()
	runOK(t, "catalog", "-table", "wonders", "-out", dir)
	if entries, _ := os.ReadDir(dir); len(entries) != 1 || entries[0].Name() != "wonders.csv" {
		t.Errorf("-table wonders -out wrote %v", entries)
	}
}

func TestJSON(t *testing.T) {
	out, _ := runOK(t, "catalog", "-format", "json")
	if strings.Contains(out, "null") {
		t.Error("the JSON contains null")
	}
	if !strings.HasSuffix(out, "}\n") {
		t.Error("the JSON does not end with a newline")
	}
	var cat jsonCatalog
	dec := json.NewDecoder(strings.NewReader(out))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cat); err != nil {
		t.Fatal(err)
	}
	if cat.Schema != jsonSchema || len(cat.Cards) != 73 || len(cat.Wonders) != 12 || len(cat.Tokens) != 10 {
		t.Fatalf("schema %d with %d cards, %d wonders, %d tokens", cat.Schema, len(cat.Cards), len(cat.Wonders), len(cat.Tokens))
	}
	for i, c := range cat.Cards {
		if c.ID != i+1 || c.Slug == "" || c.Name == "" || c.Back == "" || c.Color == "" || c.AltNames == nil || c.Effects == nil {
			t.Errorf("card %d is incomplete: %+v", i+1, c)
		}
		if (c.ChainIn == nil) != (c.ChainFrom == nil) || (c.ChainOut == nil) != (c.ChainTo == nil) {
			t.Errorf("card %s: a chain symbol without its card", c.Slug)
		}
	}
	bySlug := map[string]jsonCard{}
	for _, c := range cat.Cards {
		bySlug[c.Slug] = c
	}
	lh := bySlug["lighthouse"]
	if lh.Age != 3 || lh.Back != "age3" || lh.Color != "yellow" || lh.Cost != (jsonCost{Clay: 2, Glass: 1}) ||
		*lh.ChainIn != "amphora" || *lh.ChainFrom != "tavern" || lh.ChainOut != nil {
		t.Errorf("lighthouse = %+v", lh)
	}
	effects, _ := json.Marshal(lh.Effects)
	if string(effects) != `[{"kind":"points","n":3},{"kind":"coins-per-card","n":1,"colors":["yellow"],"scope":"self"}]` {
		t.Errorf("lighthouse effects = %s", effects)
	}
	lumber := bySlug["lumber-yard"]
	effects, _ = json.Marshal(lumber.Effects)
	if string(effects) != `[{"kind":"produce","n":1,"resource":"wood"}]` || lumber.Cost != (jsonCost{}) {
		t.Errorf("lumber-yard effects = %s, cost %+v", effects, lumber.Cost)
	}
	ship := bySlug["shipowners-guild"]
	effects, _ = json.Marshal(ship.Effects)
	if ship.Back != "guild" || !strings.Contains(string(effects), `"colors":["brown","grey"],"scope":"max"`) {
		t.Errorf("shipowners-guild = %+v, effects %s", ship, effects)
	}
	if got := bySlug["courthouse"].AltNames; len(got) != 1 || got[0] != "Tribunal" {
		t.Errorf("courthouse altNames = %v", got)
	}
	if tm := bySlug["temple"]; *tm.ChainFrom != "altar" || *tm.ChainTo != "pantheon" {
		t.Errorf("temple chains = %v -> %v", *tm.ChainFrom, *tm.ChainTo)
	}
	gl := cat.Wonders[2]
	effects, _ = json.Marshal(gl.Effects)
	if gl.Slug != "great-lighthouse" || string(effects) != `[{"kind":"produce-choice","resources":["wood","clay","stone"]},{"kind":"points","n":4}]` {
		t.Errorf("great-lighthouse = %+v, effects %s", gl, effects)
	}
	masonry := cat.Tokens[4]
	effects, _ = json.Marshal(masonry.Effects)
	if masonry.Slug != "masonry" || string(effects) != `[{"kind":"discount","n":2,"target":"blue"}]` {
		t.Errorf("masonry effects = %s", effects)
	}
	dir := t.TempDir()
	runOK(t, "catalog", "-format", "json", "-out", dir)
	if b, err := os.ReadFile(filepath.Join(dir, "catalog.json")); err != nil || string(b) != out {
		t.Errorf("catalog.json differs from stdout (%v)", err)
	}
}

func TestProse(t *testing.T) {
	out, _ := runOK(t, "catalog", "-format", "prose")
	for _, want := range []string{
		"1. Lumber Yard (lumber-yard): Age I brown building, age1 back. Cost: free.",
		"39. Forum (forum): Age II yellow building, age2 back. Cost: 3 coins and 1 clay.",
		"Produces 1 unit of your choice of glass or papyrus every turn.",
		"59. Lighthouse (lighthouse): Age III yellow building, age3 back. Cost: 2 clay and 1 glass.",
		"you gain 1 coin for each yellow card in your city, this card included.",
		"you gain 2 coins for each brown card in your city.",
		"Free to construct if you already own Tavern, which shows the amphora chain symbol.",
		"Shows the lyre chain symbol, which makes University free to construct later.",
		"68. Shipowners Guild (shipowners-guild): Age III purple guild, guild back.",
		"for each brown and each grey card in the city that has the most brown and grey cards together (yours or your opponent's).",
		"worth 2 victory points for each wonder built in the city that has built the most wonders",
		"worth 1 victory point for every full 3 coins in the city that has the most coins",
		"you gain 2 coins for each wonder you have built.",
		"Also printed as Traders Guild.",
		"4. Temple of Artemis (temple-of-artemis): wonder. Cost: 1 wood, 1 stone, 1 glass and 1 papyrus.",
		"You gain 12 coins once, when built.",
		"You choose one brown card in your opponent's city",
		"Your opponent loses 3 coins to the bank, never going below zero.",
		"You draw 3 of the progress tokens set aside at setup",
		"You gain 6 coins once, when you take this token.",
		"Every blue card you construct from now on costs 2 resource units fewer, of your choice.",
		"Every wonder you construct from now on costs 2 resource units fewer, of your choice.",
		"worth 3 victory points for each progress token you own, this one included.",
		"Every red building you construct from now on gives 1 shield more.",
		"Shows the scales science symbol.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("prose lacks %q", want)
		}
	}
	for _, bad := range []string{"%!", "Unknown effect", "unregistered", "none"} {
		if strings.Contains(out, bad) {
			t.Errorf("prose contains %q", bad)
		}
	}
	if n := strings.Count(out, "\n   - "); n < 73+12+10 {
		t.Errorf("only %d effect lines", n)
	}
	dir := t.TempDir()
	runOK(t, "catalog", "-format", "prose", "-out", dir)
	if b, err := os.ReadFile(filepath.Join(dir, "catalog.txt")); err != nil || string(b) != out {
		t.Errorf("catalog.txt differs from stdout (%v)", err)
	}
}

func TestDeterministic(t *testing.T) {
	for _, format := range formats {
		a, _ := runOK(t, "catalog", "-format", format)
		b, _ := runOK(t, "catalog", "-format", format)
		if a != b || a == "" {
			t.Errorf("format %s is not deterministic", format)
		}
	}
}

func TestCommandLine(t *testing.T) {
	// A regular file where -out needs a directory makes the write fail.
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		args []string
		code int
		msg  string // a substring of stderr
	}{
		{nil, 0, "no TypeScript generators yet"},
		{[]string{"-help"}, 0, ""},
		{[]string{"catalog", "-help"}, 0, "-format"},
		{[]string{"nope"}, 2, `unknown subcommand "nope"`},
		{[]string{"catalog", "-format", "xml"}, 2, `unknown format "xml"`},
		{[]string{"catalog", "-format", "json", "-table", "cards"}, 2, "-table applies only to -format csv"},
		{[]string{"catalog", "-table", "guilds"}, 2, `unknown table "guilds"`},
		{[]string{"catalog", "extra"}, 2, `unexpected argument "extra"`},
		{[]string{"catalog", "-bogus"}, 2, "-bogus"},
		{[]string{"catalog", "-out", filepath.Join(blocker, "sub")}, 1, "not a directory"},
	}
	for _, c := range cases {
		var out, errOut bytes.Buffer
		code := run(c.args, &out, &errOut)
		if code != c.code || !strings.Contains(errOut.String(), c.msg) {
			t.Errorf("run %q = %d, stderr %q; want %d and %q", c.args, code, errOut.String(), c.code, c.msg)
		}
	}
	var out bytes.Buffer
	run([]string{"-help"}, &out, &bytes.Buffer{})
	if !strings.Contains(out.String(), "ajayeb-gen catalog") {
		t.Errorf("help = %q", out.String())
	}
}
