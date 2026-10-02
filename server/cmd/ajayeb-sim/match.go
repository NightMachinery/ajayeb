package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// matchConfig is one tournament between bot A and bot B.
type matchConfig struct {
	a, b     botSpec
	games    int // rounded up to an even number
	seed     uint64
	parallel int       // games played at once
	budget   ai.Budget // every move's budget, for both bots
	logDir   string    // where to write each game's log, or ""
}

// gameResult is one game of a match. Side 0 is bot A, side 1 bot B.
type gameResult struct {
	index    int
	gameSeed uint64
	aSeat    engine.Seat
	result   engine.Result
	plies    int
	moves    [2]int
	moveTime [2]time.Duration // total thinking time
	maxMove  [2]time.Duration
	err      error
}

// sideOf returns the side (0 for A, 1 for B) of seat in this game.
func (g *gameResult) sideOf(seat engine.Seat) int {
	if seat == g.aSeat {
		return 0
	}
	return 1
}

// playMatch plays the match's games, cfg.parallel at a time, and returns
// them in game order. A game's error is kept in its result.
func playMatch(ctx context.Context, cfg *matchConfig) []gameResult {
	results := make([]gameResult, cfg.games)
	next := make(chan int)
	var wg sync.WaitGroup
	for range max(1, min(cfg.parallel, cfg.games)) {
		wg.Go(func() {
			for i := range next {
				results[i] = playOne(ctx, cfg, i)
			}
		})
	}
	for i := range cfg.games {
		next <- i
	}
	close(next)
	wg.Wait()
	return results
}

// playOne plays game i of the match.
func playOne(ctx context.Context, cfg *matchConfig, i int) gameResult {
	gameSeed, aSeat, seeds := ai.PairedGame(cfg.seed, i)
	g := gameResult{index: i, gameSeed: gameSeed, aSeat: aSeat}
	var bots [2]ai.Bot
	bots[aSeat] = cfg.a.make(seeds[0])
	bots[aSeat.Opp()] = cfg.b.make(seeds[1])
	opts := engine.Options{}
	var log *engine.Log
	if cfg.logDir != "" {
		log = engine.NewLog(gameSeed, opts, nil)
	}
	onMove := func(m ai.MoveInfo) {
		side := g.sideOf(m.Seat)
		g.moves[side]++
		g.moveTime[side] += m.Elapsed
		g.maxMove[side] = max(g.maxMove[side], m.Elapsed)
	}
	end, err := ai.PlayGame(ctx, engine.NewGame(gameSeed, opts), bots, [2]ai.Budget{cfg.budget, cfg.budget}, log, onMove)
	g.result, g.plies = end.Result, int(end.Ply)
	if err != nil {
		g.err = err
		return g
	}
	if log != nil {
		g.err = writeLog(filepath.Join(cfg.logDir, fmt.Sprintf("game-%04d.json", i)), log)
	}
	return g
}

// writeLog writes a game log as indented JSON.
func writeLog(path string, log *engine.Log) error {
	b, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

// matchStats sums up a match.
type matchStats struct {
	games    int
	wins     [2][4]int // by side and victory kind
	shared   int
	score    float64 // A's points: 1 per win, 0.5 per shared victory
	plies    int
	moves    [2]int
	moveTime [2]time.Duration
	maxMove  [2]time.Duration
}

// summarize adds up the results.
func summarize(results []gameResult) matchStats {
	var st matchStats
	for i := range results {
		g := &results[i]
		st.games++
		st.plies += g.plies
		for side := range 2 {
			st.moves[side] += g.moves[side]
			st.moveTime[side] += g.moveTime[side]
			st.maxMove[side] = max(st.maxMove[side], g.maxMove[side])
		}
		if w := g.result.Winner; w == engine.SharedVictory {
			st.shared++
			st.score += 0.5
		} else {
			side := g.sideOf(engine.Seat(w))
			st.wins[side][g.result.Kind]++
			if side == 0 {
				st.score++
			}
		}
	}
	return st
}

// wilson returns the 95% Wilson score interval of a proportion p observed
// over n trials.
func wilson(p float64, n int) (lo, hi float64) {
	if n == 0 {
		return 0, 1
	}
	const z = 1.959963984540054
	nf := float64(n)
	den := 1 + z*z/nf
	centre := (p + z*z/(2*nf)) / den
	half := z * math.Sqrt(p*(1-p)/nf+z*z/(4*nf*nf)) / den
	return max(0, centre-half), min(1, centre+half)
}

// labels names the two sides for the report, telling apart two bots of the
// same kind.
func labels(cfg *matchConfig) [2]string {
	a, b := cfg.a.String(), cfg.b.String()
	if a == b {
		return [2]string{a + " (A)", b + " (B)"}
	}
	return [2]string{a, b}
}

// printMatch writes the report of a match.
func printMatch(w io.Writer, cfg *matchConfig, st *matchStats, elapsed time.Duration) {
	names := labels(cfg)
	fmt.Fprintf(w, "match %s vs %s: %d games (%d deals, each played with both seatings), seed %d\n",
		names[0], names[1], st.games, st.games/2, cfg.seed)
	width := max(len(names[0]), len(names[1]), len("shared"))
	for side := range 2 {
		k := &st.wins[side]
		total := k[engine.VictoryMilitary] + k[engine.VictoryScience] + k[engine.VictoryCivilian]
		fmt.Fprintf(w, "  %-*s %4d wins: military %d, science %d, civilian %d\n", width, names[side], total,
			k[engine.VictoryMilitary], k[engine.VictoryScience], k[engine.VictoryCivilian])
	}
	fmt.Fprintf(w, "  %-*s %4d\n", width, "shared", st.shared)
	p := 0.0
	if st.games > 0 {
		p = st.score / float64(st.games)
	}
	lo, hi := wilson(p, st.games)
	fmt.Fprintf(w, "%s scores %.1f%% (95%% interval %.1f%% to %.1f%%; a shared victory counts half)\n",
		names[0], 100*p, 100*lo, 100*hi)
	fmt.Fprintf(w, "moves: %s; %s\n", moveTimes(names[0], st, 0), moveTimes(names[1], st, 1))
	avg := 0.0
	if st.games > 0 {
		avg = float64(st.plies) / float64(st.games)
	}
	fmt.Fprintf(w, "games last %.1f actions on average; %s in all (%d in parallel)\n",
		avg, elapsed.Round(time.Millisecond), max(1, min(cfg.parallel, cfg.games)))
}

// moveTimes describes one side's thinking times.
func moveTimes(name string, st *matchStats, side int) string {
	mean := time.Duration(0)
	if st.moves[side] > 0 {
		mean = st.moveTime[side] / time.Duration(st.moves[side])
	}
	return fmt.Sprintf("%s %s mean, %s max (%d moves)", name, ms(mean), ms(st.maxMove[side]), st.moves[side])
}

// ms formats a duration in milliseconds with three decimals.
func ms(d time.Duration) string {
	return fmt.Sprintf("%.3f ms", float64(d)/float64(time.Millisecond))
}

// printGame writes one line about a game, for -v.
func printGame(w io.Writer, cfg *matchConfig, g *gameResult) {
	names := labels(cfg)
	r := &g.result
	outcome := "shared victory"
	if r.Winner != engine.SharedVictory {
		outcome = names[g.sideOf(engine.Seat(r.Winner))] + " wins by " + r.Kind.String()
	}
	fmt.Fprintf(w, "game %d: seed %d, %s in seat 0: %s, %d-%d, %d actions\n", g.index, g.gameSeed,
		names[g.sideOf(0)], outcome, r.Scores[0].Total, r.Scores[1].Total, g.plies)
}

// runMatch is the match subcommand.
func runMatch(args []string, stdout, stderr io.Writer) error {
	fs := newFlags("match", stderr)
	aText := fs.String("a", "greedy", "bot A")
	bText := fs.String("b", "random", "bot B")
	games := fs.Int("games", 100, "games to play (rounded up to an even number: each deal is played twice)")
	seed := fs.Uint64("seed", 1, "base seed of the deals and the bots")
	parallel := fs.Int("p", 2, "games played at once")
	iters := fs.Int("iters", 0, "iterations per move for both bots (0: each bot's default)")
	moveTime := fs.Duration("movetime", 0, "wall-clock limit per move (0: none; a limit makes results depend on the machine)")
	logDir := fs.String("logs", "", "write each game's ajayeb-log/1 file into this directory")
	verbose := fs.Bool("v", false, "print one line per game")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "ajayeb-sim match: unexpected argument %q\n", fs.Arg(0))
		return errUsage
	}
	cfg := matchConfig{seed: *seed, parallel: *parallel, logDir: *logDir,
		budget: ai.Budget{MaxIterations: *iters, MaxTime: *moveTime}}
	var err error
	if cfg.a, err = parseBot(*aText); err == nil {
		cfg.b, err = parseBot(*bText)
	}
	switch {
	case err != nil:
	case *games < 1 || *games > 100_000:
		err = fmt.Errorf("-games %d is not 1 to 100000", *games)
	case *parallel < 1 || *parallel > 256:
		err = fmt.Errorf("-p %d is not 1 to 256", *parallel)
	case *iters < 0 || *moveTime < 0:
		err = fmt.Errorf("-iters and -movetime cannot be negative")
	}
	if err != nil {
		fmt.Fprintf(stderr, "ajayeb-sim match: %v\n", err)
		return errUsage
	}
	cfg.games = *games + *games%2
	if cfg.logDir != "" {
		if err := os.MkdirAll(cfg.logDir, 0o755); err != nil {
			return err
		}
	}

	start := time.Now()
	results := playMatch(context.Background(), &cfg)
	elapsed := time.Since(start)
	failed := false
	for i := range results {
		if err := results[i].err; err != nil {
			fmt.Fprintf(stderr, "ajayeb-sim match: game %d (seed %d): %v\n", i, results[i].gameSeed, err)
			failed = true
		}
	}
	if failed {
		return errFailed
	}
	if *verbose {
		for i := range results {
			printGame(stdout, &cfg, &results[i])
		}
	}
	st := summarize(results)
	printMatch(stdout, &cfg, &st, elapsed)
	return nil
}
