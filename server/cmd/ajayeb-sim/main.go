// Command ajayeb-sim runs the bots offline: tournaments between two bots,
// the calibration of the level presets, the AI benchmark, and replay checks
// of recorded games.
//
// Usage:
//
//	ajayeb-sim match [-a BOT] [-b BOT] [-games N] [-seed S] [-p N] [-scale F] [-clock] [-wa FILE] [-wb FILE] [-shard K/N] [-json FILE] [-min R] [-logs DIR] [-v]
//	ajayeb-sim tally [-min R] [-v] FILE...
//	ajayeb-sim levels [-games N] [-seed S] [-moves N] [-fixed] [-scale F]
//	ajayeb-sim weights
//	ajayeb-sim bench [-games N] [-seed S] [-p N] [-positions N] [-playouts N]
//	ajayeb-sim replay [-q] FILE...
//
// match plays N games between bot A and bot B with paired seeds: each deal
// is played twice, with A in seat 0 and then in seat 1 (ai.PairedGame), so
// neither bot profits from the deal or from moving first. It prints both
// bots' wins by victory kind, A's score with a 95% interval, and move times.
// medium and hard search their fixed budgets (ai.Level.FixedBudget, an
// iteration count), so the results depend only on the flags, not on -p or
// on the machine; -clock gives them their clock budgets of a live game
// instead (ai.Level.Budget), which makes the results depend on the machine
// and its load. -scale multiplies the iteration budgets of medium and hard,
// on both sides, for a quicker proxy of a tournament. -wa and -wb give A or
// B evaluation weights from a JSON file (as printed by weights; fields left
// out keep their defaults), for tuning.
//
// -shard K/N plays only the deals whose index is K modulo N, so N machines
// can share a long tournament; -json writes the results, and tally merges
// the shards' files into one report. -min R makes match or tally fail (exit
// 1) when A scores less than R (a share from 0 to 1). -logs writes every
// game as an ajayeb-log/1 file, which replay can check.
//
// A BOT is random, easy, greedy, medium or hard, optionally followed by ":N":
// for easy and greedy N is the determinizations per move (for example
// greedy:8), for medium and hard the iterations per move instead of the
// fixed budget (for example medium:5000). greedy is the deterministic Greedy
// bot of the tournaments; easy is Greedy with softmax sampling, the Easy
// level.
//
// levels measures the clock budgets of medium and hard on positions from
// Greedy self-play: iterations per move (by age), time per move and memory;
// it also times easy.
// The fixed budgets are set from its medians. With -fixed it measures the
// fixed budgets instead (times -scale).
//
// weights prints the default evaluation weights as JSON, the format that -wa
// and -wb read.
//
// bench (make bench runs it) measures the engine's hot paths (random
// playouts, determinization, legal actions, evaluation), the move time of
// each bot level, and plays short tournaments of greedy and easy against
// random.
//
// replay checks recorded games: each FILE (or - for standard input) holds an
// ajayeb-log/1 log, or an object whose "log" field holds one (the engine's
// golden replays). The game is replayed with the engine's own legality
// checks; the final state's hash is printed and compared with the log's
// finalHash when it has one.
//
// Exit status: 0 on success, 1 when a game or a replay fails or a score is
// below -min, 2 for a usage error.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

const usage = `Usage:
  ajayeb-sim match [flags]       play a paired tournament between two bots
  ajayeb-sim tally FILE...       merge the -json results of a sharded match
  ajayeb-sim levels [flags]      measure the levels' iterations, time and memory per move
  ajayeb-sim weights             print the default evaluation weights as JSON
  ajayeb-sim bench [flags]       time the engine and the bots, play short tournaments
  ajayeb-sim replay FILE...      replay ajayeb-log/1 files and check their final hash
Run ajayeb-sim <subcommand> -help for the flags.
`

// errUsage marks a command-line mistake that has already been reported.
var errUsage = errors.New("usage")

// errFailed marks a failure that has already been reported in the output.
var errFailed = errors.New("failed")

// run is main without the process exit, so tests can drive it.
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	var err error
	switch args[0] {
	case "match":
		err = runMatch(args[1:], stdout, stderr)
	case "tally":
		err = runTally(args[1:], stdout, stderr)
	case "levels":
		err = runLevels(args[1:], stdout, stderr)
	case "weights":
		err = runWeights(args[1:], stdout, stderr)
	case "bench":
		err = runBench(args[1:], stdout, stderr)
	case "replay":
		err = runReplay(args[1:], stdin, stdout, stderr)
	case "-h", "-help", "--help", "help":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		fmt.Fprintf(stderr, "ajayeb-sim: unknown subcommand %q\n%s", args[0], usage)
		return 2
	}
	switch {
	case err == nil, errors.Is(err, flag.ErrHelp):
		return 0
	case errors.Is(err, errUsage):
		return 2
	case errors.Is(err, errFailed):
		return 1
	}
	fmt.Fprintf(stderr, "ajayeb-sim: %v\n", err)
	return 1
}

// newFlags returns a flag set for a subcommand that reports its errors to
// stderr.
func newFlags(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet("ajayeb-sim "+name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs
}

// parseFlags parses args, mapping a parse error to errUsage (the flag
// package has printed it) and keeping flag.ErrHelp.
func parseFlags(fs *flag.FlagSet, args []string) error {
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return err
		}
		return errUsage
	}
	return nil
}

// botSpec is a parsed BOT argument: a bot name and an optional number.
type botSpec struct {
	name string // random, easy, greedy, medium or hard
	n    int    // easy and greedy: determinizations per move; medium and hard: iterations per move; 0 for the preset
}

// maxBotN bounds the N of a BOT argument, for each kind of bot.
const (
	maxGreedyK     = 1000
	maxSearchIters = 100_000_000
)

// parseBot parses a BOT argument such as "greedy", "easy:5" or
// "medium:5000".
func parseBot(s string) (botSpec, error) {
	name, nText, hasN := strings.Cut(s, ":")
	spec := botSpec{name: name}
	top := 0
	switch name {
	case "random":
	case "easy", "greedy":
		top = maxGreedyK
	case "medium", "hard":
		top = maxSearchIters
	default:
		return botSpec{}, fmt.Errorf("unknown bot %q (want random, easy, greedy, medium or hard)", s)
	}
	if hasN {
		if top == 0 {
			return botSpec{}, fmt.Errorf("bot %q: random takes no :N", s)
		}
		n, err := strconv.Atoi(nText)
		if err != nil || n < 1 || n > top {
			return botSpec{}, fmt.Errorf("bot %q: N must be a whole number from 1 to %d", s, top)
		}
		spec.n = n
	}
	return spec, nil
}

// String returns the spec as it was written.
func (b botSpec) String() string {
	if b.n > 0 {
		return b.name + ":" + strconv.Itoa(b.n)
	}
	return b.name
}

// level returns the ai level of a tree-search spec (medium or hard), and
// false for the others.
func (b botSpec) level() (ai.Level, bool) {
	switch b.name {
	case "medium":
		return ai.Medium, true
	case "hard":
		return ai.Hard, true
	}
	return 0, false
}

// make returns the bot, seeded with seed. w, if not nil, replaces the
// evaluation weights of the bots that use them (all but random).
func (b botSpec) make(seed uint64, w *ai.Weights) ai.Bot {
	switch b.name {
	case "random":
		return ai.NewRandom(seed)
	case "easy":
		return ai.NewGreedy(seed, ai.GreedyConfig{K: b.n, Temperature: ai.EasyTemperature, Weights: w})
	case "greedy":
		return ai.NewGreedy(seed, ai.GreedyConfig{K: b.n, Weights: w})
	}
	l, _ := b.level()
	return ai.NewMCTS(seed, ai.MCTSConfig{Budget: l.Budget(), Weights: w})
}

// budget returns the bot's budget per move. A tree-search bot searches its
// level's fixed budget, or N iterations, times scale (at least one), or,
// with clock, its level's clock budget. Greedy's budget is its
// determinizations; Random needs none.
func (b botSpec) budget(clock bool, scale float64) ai.Budget {
	l, search := b.level()
	switch {
	case !search:
		return ai.Budget{MaxIterations: b.n}
	case clock:
		return l.Budget()
	}
	bud := l.FixedBudget()
	if b.n > 0 {
		bud.MaxIterations = b.n
	}
	bud.MaxIterations = max(1, int(math.Round(float64(bud.MaxIterations)*scale)))
	return bud
}

// readWeights reads evaluation weights from a JSON file: the default
// weights, with the fields the file sets replaced. An empty path gives nil
// (the defaults).
func readWeights(path string) (*ai.Weights, error) {
	if path == "" {
		return nil, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	w := ai.DefaultWeights()
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&w); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &w, nil
}

// runWeights is the weights subcommand.
func runWeights(args []string, stdout, stderr io.Writer) error {
	fs := newFlags("weights", stderr)
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "ajayeb-sim weights: unexpected argument %q\n", fs.Arg(0))
		return errUsage
	}
	b, err := json.MarshalIndent(ai.DefaultWeights(), "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(stdout, "%s\n", b)
	return err
}
