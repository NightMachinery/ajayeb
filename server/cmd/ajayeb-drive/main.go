// Command ajayeb-drive is a WebSocket client that plays full games against
// a running ajayeb server, the successor to the project's earlier
// drive.mjs: a protocol-level smoke test that needs no browser.
//
// Usage:
//
//	ajayeb-drive --origin http://127.0.0.1:38951 --mode online|hotseat|ai [flags]
//
// Run ajayeb-drive -help for the flags.
//
// Each game picks a uniformly random legal choice every turn (from the
// view's own choices, never computed locally) until the server reports a
// result, exactly as a thin client must. An online game is played against
// itself, over two distinct WebSocket connections with two distinct
// identity tokens, so a single run exercises the real two-seat path. The
// command exits non-zero the moment any game hits a protocol error (an
// unexpected reject or error frame, a closed connection, a frame that does
// not decode, a timeout) or finishes without a result, and it always prints
// one summary line per finished game first.
//
// --concurrency plays that many games at once against the one server, for
// a load or memory sweep (see docs/testing.md's RAM figure): for example,
// --games 20 --concurrency 20 starts 20 games together and reports once all
// have finished, so an operator can sample the server process's resident
// memory while they are all in flight. Each connection also sends a
// synthetic X-Forwarded-For (one per game, two for online's two seats),
// which internal/ws only honors from a trusted proxy address: start the
// server under test with -trust-proxy naming this process's own address
// (-trust-proxy 127.0.0.1 for a driver running on the same host) to raise
// its 16-sockets-per-address default past -concurrency; without it, the
// header is simply ignored and every connection counts against the one
// real address as usual.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/wire"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// errBadFlags reports a flag problem the flag package has already printed,
// along with the usage.
var errBadFlags = errors.New("bad flags")

// options is the parsed command line.
type options struct {
	origin      string
	mode        string
	games       int
	seedBase    uint64
	concurrency int
	bot         wire.BotLevel
	gameTimeout time.Duration
}

var modes = []string{"online", "hotseat", "ai"}
var botLevels = []string{"easy", "normal", "hard"}

func parseFlags(args []string, stderr io.Writer) (options, error) {
	var o options
	var botFlag string
	fs := flag.NewFlagSet("ajayeb-drive", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&o.origin, "origin", "", "the running server's `origin`, e.g. http://127.0.0.1:38951 (required)")
	fs.StringVar(&o.mode, "mode", "", "room mode: "+strings.Join(modes, "|")+" (required)")
	fs.IntVar(&o.games, "games", 1, "number of games to play")
	fs.Uint64Var(&o.seedBase, "seed", 0, "base `seed`; game i uses seed+i (0 picks a random base)")
	fs.IntVar(&o.concurrency, "concurrency", 1, "games to play at once against the server")
	fs.StringVar(&botFlag, "bot", "easy", "bot level for -mode ai: "+strings.Join(botLevels, "|"))
	fs.DurationVar(&o.gameTimeout, "game-timeout", 2*time.Minute, "fail a game that has not finished within this `duration`")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: ajayeb-drive --origin URL --mode %s [flags]\n\nFlags:\n", strings.Join(modes, "|"))
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return options{}, err
		}
		return options{}, errBadFlags
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "ajayeb-drive: unexpected arguments: %s\n", strings.Join(fs.Args(), " "))
		return options{}, errBadFlags
	}
	if o.origin == "" {
		fmt.Fprintln(stderr, "ajayeb-drive: -origin is required")
		return options{}, errBadFlags
	}
	modeOK := false
	for _, m := range modes {
		modeOK = modeOK || o.mode == m
	}
	if !modeOK {
		fmt.Fprintf(stderr, "ajayeb-drive: -mode must be one of %s, got %q\n", strings.Join(modes, "|"), o.mode)
		return options{}, errBadFlags
	}
	switch botFlag {
	case "easy":
		o.bot = wire.BotLevelEasy
	case "normal":
		o.bot = wire.BotLevelNormal
	case "hard":
		o.bot = wire.BotLevelHard
	default:
		fmt.Fprintf(stderr, "ajayeb-drive: -bot must be one of %s, got %q\n", strings.Join(botLevels, "|"), botFlag)
		return options{}, errBadFlags
	}
	if o.games < 1 {
		fmt.Fprintln(stderr, "ajayeb-drive: -games must be at least 1")
		return options{}, errBadFlags
	}
	if o.concurrency < 1 {
		fmt.Fprintln(stderr, "ajayeb-drive: -concurrency must be at least 1")
		return options{}, errBadFlags
	}
	return o, nil
}

// wsURLFrom turns a server origin (http(s)://host[:port]) into its /ws
// WebSocket URL.
func wsURLFrom(origin string) (string, error) {
	u, err := url.Parse(origin)
	if err != nil {
		return "", fmt.Errorf("-origin: %w", err)
	}
	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", fmt.Errorf("-origin: scheme must be http, https, ws or wss, got %q", u.Scheme)
	}
	if u.Host == "" {
		return "", fmt.Errorf("-origin: missing host")
	}
	u.Path = "/ws"
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

// randomSeedBase returns a seed base that is very unlikely to collide with
// another run, when the caller passed -seed 0.
func randomSeedBase() uint64 {
	return uint64(time.Now().UnixNano())
}

// run is main without the process exit, so tests can drive it. It returns
// 0 when every game finished cleanly, 1 when any game failed, and 2 for a
// usage error.
func run(args []string, stdout, stderr io.Writer) int {
	opts, err := parseFlags(args, stderr)
	switch {
	case errors.Is(err, flag.ErrHelp):
		return 0
	case errors.Is(err, errBadFlags):
		return 2
	case err != nil:
		fmt.Fprintf(stderr, "ajayeb-drive: %v\n", err)
		return 2
	}
	wsURL, err := wsURLFrom(opts.origin)
	if err != nil {
		fmt.Fprintf(stderr, "ajayeb-drive: %v\n", err)
		return 2
	}
	seedBase := opts.seedBase
	if seedBase == 0 {
		seedBase = randomSeedBase()
	}

	var out sync.Mutex // serializes stdout/stderr writes across concurrent games
	var failed atomic.Bool
	var wg sync.WaitGroup
	sem := make(chan struct{}, opts.concurrency)

	for i := range opts.games {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()
			seed := seedBase + uint64(i)
			ctx, cancel := context.WithTimeout(context.Background(), opts.gameTimeout)
			defer cancel()
			summary, err := playGame(ctx, wsURL, opts.mode, seed, opts.bot, i)
			out.Lock()
			defer out.Unlock()
			if err != nil {
				failed.Store(true)
				fmt.Fprintf(stdout, "game %d: FAILED: %v\n", i, err)
				return
			}
			fmt.Fprintf(stdout, "game %d: %s\n", i, summary)
		}(i)
	}
	wg.Wait()

	if failed.Load() {
		return 1
	}
	return 0
}
