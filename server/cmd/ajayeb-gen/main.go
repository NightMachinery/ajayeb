// Command ajayeb-gen writes Ajayeb's generated files.
//
// Usage:
//
//	ajayeb-gen
//	ajayeb-gen fixtures [-out PATH] [-messages DIR] [-protocol DIR]
//	ajayeb-gen catalog [-game duel|haftgane] [-format csv|json|prose|ts] [-out DIR] [-table TABLE]
//
// Without a subcommand it regenerates the TypeScript under web/src/lib/wire
// that make gen is responsible for, after tygo has already written the
// types themselves: the fixtures subcommand with its defaults
// (fixtures.gen.ts), and the presentation catalog as a TypeScript module
// (catalog.gen.ts, the same as catalog -format ts). A later generator that
// also belongs in that default sweep gets a call here alongside them. The
// separate platform DTO pass writes platform.gen.ts without changing Duel output.
//
// fixtures turns real recorded server output into a TypeScript fixture file
// that make check-web type-checks against the generated wire types:
// internal/wire/testdata/messages/*.json (whole ServerMessage frames) become
// `export const fixtures`, and every viewer's View from
// internal/engine/testdata/protocol/*.json becomes `export const
// protocolFixtures`. See fixtures.go for the format in full.
//
// The catalog subcommand dumps the card, wonder and token catalog of the
// engine in one of four formats:
//
//   - csv (the default): the three verification tables, cards, wonders and
//     tokens, in the schema shared with the independent verification tables.
//     They are byte for byte the goldens in
//     internal/engine/catalog/testdata/*.golden.csv.
//   - json: the presentation catalog for the client (ids, slugs, names,
//     colour, age, back, cost, declarative effects and chains).
//   - prose: every item with each effect written out in plain English, for a
//     semantic review of the data against the rules.
//   - ts: the json data as a typed TypeScript module keyed by slug, with the
//     engine's catalog hash (see catalogts.go). make gen writes it to
//     web/src/lib/wire/catalog.gen.ts.
//
// With -out DIR the output goes to files in DIR, which is created if needed:
// cards.csv, wonders.csv and tokens.csv; catalog.json; catalog.txt; or
// catalog.gen.ts.
// Without -out it goes to stdout; for csv that is the three tables one after
// the other, each starting with its header row. -table picks one csv table.
//
// -game picks the game whose catalog is dumped: duel (the default, all of the
// above) or haftgane, whose csv format writes the record tables cards,
// wonders and boards, byte for byte the goldens in
// internal/haftgane/catalog/testdata/*.golden.csv, and whose prose format
// writes every design and wonder stage in plain English (haftgane.go).
// Haftgane has no json or ts format yet.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

const usage = `Usage:
  ajayeb-gen                     regenerate the TypeScript under web/src/lib/wire (fixtures.gen.ts, catalog.gen.ts and platform.gen.ts)
  ajayeb-gen fixtures [flags]    write the wire and protocol fixtures (ajayeb-gen fixtures -help for the flags)
  ajayeb-gen catalog [flags]     dump the catalog (ajayeb-gen catalog -help for the flags)
`

// errUsage marks a command-line mistake that has already been reported.
var errUsage = errors.New("usage")

// run is main without the process exit, so tests can drive it. It returns 0
// on success, 1 when writing the output failed and 2 for a usage error.
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		// make gen's no-subcommand call: every generator that belongs in
		// that default sweep runs here, with its own defaults.
		if err := runDefault(stdout, defaultPaths); err != nil {
			fmt.Fprintf(stderr, "ajayeb-gen: %v\n", err)
			return 1
		}
		return 0
	}
	var err error
	switch args[0] {
	case "fixtures":
		err = runFixturesCmd(args[1:], stdout, stderr)
	case "catalog":
		err = runCatalog(args[1:], stdout, stderr)
	case "-h", "-help", "--help", "help":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		fmt.Fprintf(stderr, "ajayeb-gen: unknown subcommand %q\n%s", args[0], usage)
		return 2
	}
	switch {
	case err == nil:
		return 0
	case errors.Is(err, flag.ErrHelp):
		return 0
	case errors.Is(err, errUsage):
		return 2
	default:
		fmt.Fprintf(stderr, "ajayeb-gen: %v\n", err)
		return 1
	}
}

// genPaths are the inputs and outputs of the default run.
type genPaths struct {
	fixturesOut, catalogOut, messagesDir, protocolDir string
	platformOut                                       string
}

// defaultPaths are relative to server/, where make gen runs this command.
var defaultPaths = genPaths{
	fixturesOut: defaultFixturesOut,
	platformOut: "../web/src/lib/wire/platform.gen.ts",
	catalogOut:  defaultCatalogTSOut,
	messagesDir: defaultMessagesDir,
	protocolDir: defaultProtocolDir,
}

// runDefault runs every generator of the default sweep.
func runDefault(stdout io.Writer, p genPaths) error {
	if err := generateFixtures(p.fixturesOut, p.messagesDir, p.protocolDir); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "wrote %s\n", p.fixturesOut)
	if err := writeCatalogTS(p.catalogOut); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "wrote %s\n", p.catalogOut)
	if p.platformOut != "" {
		if err := generatePlatform(p.platformOut); err != nil {
			return err
		}
		fmt.Fprintf(stdout, "wrote %s\n", p.platformOut)
	}
	return nil
}

// catalogOptions is the parsed command line of the catalog subcommand.
type catalogOptions struct {
	game   string // duel or haftgane
	format string // csv, json, prose or ts
	out    string // output directory, or "" for stdout
	table  string // csv only: one table, or "" for all three
}

var (
	games   = []string{"duel", "haftgane"}
	formats = []string{"csv", "json", "prose", "ts"}
	tables  = []string{"cards", "wonders", "tokens"}
)

func parseCatalogFlags(args []string, stderr io.Writer) (catalogOptions, error) {
	var o catalogOptions
	fs := flag.NewFlagSet("ajayeb-gen catalog", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&o.game, "game", "duel", "the game whose catalog to dump: "+strings.Join(games, ", "))
	fs.StringVar(&o.format, "format", "csv", "output format: "+strings.Join(formats, ", ")+" (haftgane: csv, prose)")
	fs.StringVar(&o.out, "out", "", "write files into this directory (created if needed) instead of stdout")
	fs.StringVar(&o.table, "table", "", "csv only: write just this table ("+strings.Join(tables, ", ")+"; haftgane: "+strings.Join(haftganeTables, ", ")+")")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: ajayeb-gen catalog [-game duel|haftgane] [-format csv|json|prose|ts] [-out DIR] [-table TABLE]\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return o, err
		}
		return o, errUsage // the flag package has printed the problem and the usage
	}
	bad := func(format string, a ...any) (catalogOptions, error) {
		fmt.Fprintf(stderr, "ajayeb-gen catalog: "+format+"\n", a...)
		return o, errUsage
	}
	if fs.NArg() > 0 {
		return bad("unexpected argument %q", fs.Arg(0))
	}
	if !slices.Contains(games, o.game) {
		return bad("unknown game %q (want %s)", o.game, strings.Join(games, ", "))
	}
	gameFormats, gameTables := formats, tables
	if o.game == "haftgane" {
		gameFormats, gameTables = haftganeFormats, haftganeTables
	}
	if !slices.Contains(gameFormats, o.format) {
		if slices.Contains(formats, o.format) {
			return bad("format %q is not available for -game %s (want %s)", o.format, o.game, strings.Join(gameFormats, ", "))
		}
		return bad("unknown format %q (want %s)", o.format, strings.Join(formats, ", "))
	}
	if o.table != "" {
		if o.format != "csv" {
			return bad("-table applies only to -format csv")
		}
		if !slices.Contains(gameTables, o.table) {
			return bad("unknown table %q (want %s)", o.table, strings.Join(gameTables, ", "))
		}
	}
	return o, nil
}

func runCatalog(args []string, stdout, stderr io.Writer) error {
	o, err := parseCatalogFlags(args, stderr)
	if err != nil {
		return err
	}
	outputs, err := catalogOutputs(o)
	if err != nil {
		return err
	}
	if o.out == "" {
		for _, f := range outputs {
			if _, err := stdout.Write(f.data); err != nil {
				return err
			}
		}
		return nil
	}
	if err := os.MkdirAll(o.out, 0o755); err != nil {
		return err
	}
	for _, f := range outputs {
		if err := os.WriteFile(filepath.Join(o.out, f.name), f.data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// output is one generated file.
type output struct {
	name string
	data []byte
}

// catalogOutputs renders the files the options ask for, in a fixed order.
func catalogOutputs(o catalogOptions) ([]output, error) {
	if o.game == "haftgane" {
		return haftganeOutputs(o)
	}
	switch o.format {
	case "csv":
		return csvOutputs(o.table)
	case "json":
		data, err := presentationJSON()
		if err != nil {
			return nil, err
		}
		return []output{{"catalog.json", data}}, nil
	case "prose":
		return []output{{"catalog.txt", proseCatalog()}}, nil
	case "ts":
		data, err := catalogTS()
		if err != nil {
			return nil, err
		}
		return []output{{"catalog.gen.ts", data}}, nil
	}
	return nil, fmt.Errorf("unknown format %q", o.format)
}
