// Command ajayeb-sim runs the bots offline: tournaments between two bots,
// the AI benchmark, and replay checks of recorded games.
//
// Usage:
//
//	ajayeb-sim match [-a BOT] [-b BOT] [-games N] [-seed S] [-p N] [-iters N] [-movetime D] [-logs DIR] [-v]
//	ajayeb-sim bench [-games N] [-seed S] [-p N] [-positions N] [-playouts N]
//	ajayeb-sim replay [-q] FILE...
//
// match plays N games between bot A and bot B with paired seeds: each deal
// is played twice, with A in seat 0 and then in seat 1 (ai.PairedGame), so
// neither bot profits from the deal or from moving first. It prints both
// bots' wins by victory kind, A's score with a 95% interval, and move times.
// The results depend only on the flags, not on -p, unless -movetime sets a
// wall-clock budget. -logs writes every game as an ajayeb-log/1 file, which
// replay can check.
//
// A BOT is a level name (random, easy, greedy, medium or hard), optionally
// followed by ":K" for the determinizations per move of easy and greedy, for
// example greedy:8. greedy is the deterministic Greedy bot of the
// tournaments; easy is Greedy with softmax sampling. Until the tree search
// lands, medium and hard play Greedy too.
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
// Exit status: 0 on success, 1 when a game or a replay fails, 2 for a usage
// error.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
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

// botSpec is a parsed BOT argument: a level name and an optional number of
// determinizations.
type botSpec struct {
	name string // random, easy, greedy, medium or hard
	k    int    // determinizations per move for easy and greedy; 0 for the default
}

// parseBot parses a BOT argument such as "greedy" or "easy:5".
func parseBot(s string) (botSpec, error) {
	name, kText, hasK := strings.Cut(s, ":")
	spec := botSpec{name: name}
	switch name {
	case "random", "easy", "greedy", "medium", "hard":
	default:
		return botSpec{}, fmt.Errorf("unknown bot %q (want random, easy, greedy, medium or hard)", s)
	}
	if hasK {
		if name != "easy" && name != "greedy" {
			return botSpec{}, fmt.Errorf("bot %q: only easy and greedy take :K", s)
		}
		k, err := strconv.Atoi(kText)
		if err != nil || k < 1 || k > 1000 {
			return botSpec{}, fmt.Errorf("bot %q: K must be a whole number from 1 to 1000", s)
		}
		spec.k = k
	}
	return spec, nil
}

// String returns the spec as it was written.
func (b botSpec) String() string {
	if b.k > 0 {
		return b.name + ":" + strconv.Itoa(b.k)
	}
	return b.name
}

// make returns the bot, seeded with seed.
func (b botSpec) make(seed uint64) ai.Bot {
	switch b.name {
	case "random":
		return ai.NewRandom(seed)
	case "easy":
		return ai.NewGreedy(seed, ai.GreedyConfig{K: b.k, Temperature: ai.EasyTemperature})
	case "greedy":
		return ai.NewGreedy(seed, ai.GreedyConfig{K: b.k})
	case "medium":
		return ai.New(ai.Medium, seed)
	}
	return ai.New(ai.Hard, seed)
}
