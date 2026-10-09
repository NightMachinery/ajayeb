// Command ajayeb-gen writes Ajayeb's generated files.
//
// Usage:
//
//	ajayeb-gen
//	ajayeb-gen fixtures [-out PATH] [-messages DIR] [-protocol DIR]
//	ajayeb-gen catalog [-format csv|json|prose] [-out DIR] [-table cards|wonders|tokens]
//
// Without a subcommand it regenerates the TypeScript under web/src/lib/wire
// that make gen is responsible for, after tygo has already written the
// types themselves: today that is just the fixtures subcommand, run with
// its defaults. A later generator that also belongs in that default sweep
// gets its own subcommand and a call here alongside it.
//
// fixtures turns real recorded server output into a TypeScript fixture file
// that make check-web type-checks against the generated wire types:
// internal/wire/testdata/messages/*.json (whole ServerMessage frames) become
// `export const fixtures`, and every viewer's View from
// internal/engine/testdata/protocol/*.json becomes `export const
// protocolFixtures`. See fixtures.go for the format in full.
//
// The catalog subcommand dumps the card, wonder and token catalog of the
// engine in one of three formats:
//
//   - csv (the default): the three verification tables, cards, wonders and
//     tokens, in the schema shared with the independent verification tables.
//     They are byte for byte the goldens in
//     internal/engine/catalog/testdata/*.golden.csv.
//   - json: the presentation catalog for the client (ids, slugs, names,
//     colour, age, back, cost, declarative effects and chains).
//   - prose: every item with each effect written out in plain English, for a
//     semantic review of the data against the rules.
//
// With -out DIR the output goes to files in DIR, which is created if needed:
// cards.csv, wonders.csv and tokens.csv; catalog.json; or catalog.txt.
// Without -out it goes to stdout; for csv that is the three tables one after
// the other, each starting with its header row. -table picks one csv table.
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
  ajayeb-gen                     regenerate the TypeScript under web/src/lib/wire (today: fixtures, with its defaults)
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
		if err := generateFixtures(defaultFixturesOut, defaultMessagesDir, defaultProtocolDir); err != nil {
			fmt.Fprintf(stderr, "ajayeb-gen: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "wrote %s\n", defaultFixturesOut)
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

// catalogOptions is the parsed command line of the catalog subcommand.
type catalogOptions struct {
	format string // csv, json or prose
	out    string // output directory, or "" for stdout
	table  string // csv only: one table, or "" for all three
}

var (
	formats = []string{"csv", "json", "prose"}
	tables  = []string{"cards", "wonders", "tokens"}
)

func parseCatalogFlags(args []string, stderr io.Writer) (catalogOptions, error) {
	var o catalogOptions
	fs := flag.NewFlagSet("ajayeb-gen catalog", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&o.format, "format", "csv", "output format: "+strings.Join(formats, ", "))
	fs.StringVar(&o.out, "out", "", "write files into this directory (created if needed) instead of stdout")
	fs.StringVar(&o.table, "table", "", "csv only: write just this table ("+strings.Join(tables, ", ")+")")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: ajayeb-gen catalog [-format csv|json|prose] [-out DIR] [-table cards|wonders|tokens]\n\n")
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
	if !slices.Contains(formats, o.format) {
		return bad("unknown format %q (want %s)", o.format, strings.Join(formats, ", "))
	}
	if o.table != "" {
		if o.format != "csv" {
			return bad("-table applies only to -format csv")
		}
		if !slices.Contains(tables, o.table) {
			return bad("unknown table %q (want %s)", o.table, strings.Join(tables, ", "))
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
	}
	return nil, fmt.Errorf("unknown format %q", o.format)
}
