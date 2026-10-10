package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
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
	parallel int               // games played at once
	clock    bool              // medium and hard use their clock budgets
	scale    float64           // factor on the iteration budgets of medium and hard
	weights  [2]*ai.Weights    // A's and B's evaluation weights; nil for the defaults
	search   [2]*ai.MCTSConfig // A's and B's search settings; nil for their levels'
	shard    shard             // which deals to play
	logDir   string            // where to write each game's log, or ""
}

// budgets returns A's and B's budgets per move.
func (cfg *matchConfig) budgets() [2]ai.Budget {
	return [2]ai.Budget{cfg.a.budget(cfg.clock, cfg.scale), cfg.b.budget(cfg.clock, cfg.scale)}
}

// indexes returns the games this run plays: all of them, or a shard's.
func (cfg *matchConfig) indexes() []int {
	var out []int
	for i := range cfg.games {
		if cfg.shard.has(i) {
			out = append(out, i)
		}
	}
	return out
}

// shard selects the deals one run of a long match plays: those whose index
// is k modulo n. The zero shard (n 0) selects all of them.
type shard struct{ k, n int }

// parseShard parses "K/N" with 0 <= K < N; the empty string is the zero
// shard.
func parseShard(s string) (shard, error) {
	if s == "" {
		return shard{}, nil
	}
	kText, nText, ok := strings.Cut(s, "/")
	k, err1 := strconv.Atoi(kText)
	n, err2 := strconv.Atoi(nText)
	if !ok || err1 != nil || err2 != nil || n < 1 || n > 1000 || k < 0 || k >= n {
		return shard{}, fmt.Errorf("-shard %q is not K/N with 0 <= K < N <= 1000", s)
	}
	return shard{k: k, n: n}, nil
}

// has reports whether game i (of deal i/2) belongs to the shard.
func (s shard) has(i int) bool { return s.n <= 1 || (i/2)%s.n == s.k }

// String returns "K/N", or "" for the zero shard.
func (s shard) String() string {
	if s.n == 0 {
		return ""
	}
	return fmt.Sprintf("%d/%d", s.k, s.n)
}

// gameResult is one game of a match. Arrays by side hold bot A first, then
// bot B. It is also the per-game record of the -json file.
type gameResult struct {
	Index    int              `json:"game"`
	GameSeed uint64           `json:"gameSeed"`
	ASeat    engine.Seat      `json:"aSeat"`
	Result   engine.Result    `json:"result"`
	Plies    int              `json:"plies"`
	Moves    [2]int           `json:"moves"`      // moves chosen by the bot, by side
	MoveTime [2]time.Duration `json:"moveTimeNs"` // total thinking time, by side
	MaxMove  [2]time.Duration `json:"maxMoveNs"`  // longest move, by side
	err      error
}

// sideOf returns the side (0 for A, 1 for B) of seat in this game.
func (g *gameResult) sideOf(seat engine.Seat) int {
	if seat == g.ASeat {
		return 0
	}
	return 1
}

// playMatch plays the games listed in idx, cfg.parallel at a time, and
// returns them in that order. A game's error is kept in its result.
func playMatch(ctx context.Context, cfg *matchConfig, idx []int) []gameResult {
	results := make([]gameResult, len(idx))
	next := make(chan int)
	var wg sync.WaitGroup
	for range max(1, min(cfg.parallel, len(idx))) {
		wg.Go(func() {
			for j := range next {
				results[j] = playOne(ctx, cfg, idx[j])
			}
		})
	}
	for j := range idx {
		next <- j
	}
	close(next)
	wg.Wait()
	return results
}

// playOne plays game i of the match.
func playOne(ctx context.Context, cfg *matchConfig, i int) gameResult {
	gameSeed, aSeat, seeds := ai.PairedGame(cfg.seed, i)
	g := gameResult{Index: i, GameSeed: gameSeed, ASeat: aSeat}
	var bots [2]ai.Bot
	var budgets [2]ai.Budget
	sides := cfg.budgets()
	bots[aSeat], budgets[aSeat] = cfg.a.make(seeds[0], cfg.weights[0], cfg.search[0]), sides[0]
	bots[aSeat.Opp()], budgets[aSeat.Opp()] = cfg.b.make(seeds[1], cfg.weights[1], cfg.search[1]), sides[1]
	opts := engine.Options{}
	var log *engine.Log
	if cfg.logDir != "" {
		log = engine.NewLog(gameSeed, opts, nil)
	}
	onMove := func(m ai.MoveInfo) {
		side := g.sideOf(m.Seat)
		g.Moves[side]++
		g.MoveTime[side] += m.Elapsed
		g.MaxMove[side] = max(g.MaxMove[side], m.Elapsed)
	}
	end, err := ai.PlayGame(ctx, engine.NewGame(gameSeed, opts), bots, budgets, log, onMove)
	g.Result, g.Plies = end.Result, int(end.Ply)
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

// rate returns A's share of the points, from 0 to 1.
func (st *matchStats) rate() float64 {
	if st.games == 0 {
		return 0
	}
	return st.score / float64(st.games)
}

// summarize adds up the results.
func summarize(results []gameResult) matchStats {
	var st matchStats
	for i := range results {
		g := &results[i]
		st.games++
		st.plies += g.Plies
		for side := range 2 {
			st.moves[side] += g.Moves[side]
			st.moveTime[side] += g.MoveTime[side]
			st.maxMove[side] = max(st.maxMove[side], g.MaxMove[side])
		}
		if w := g.Result.Winner; w == engine.SharedVictory {
			st.shared++
			st.score += 0.5
		} else {
			side := g.sideOf(engine.Seat(w))
			st.wins[side][g.Result.Kind]++
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

// matchHeader is what identifies a match: the same for every shard of it.
type matchHeader struct {
	A       string            `json:"a"`
	B       string            `json:"b"`
	Budgets [2]ai.Budget      `json:"budgets"` // A's and B's budgets per move
	Weights [2]*ai.Weights    `json:"weights"` // A's and B's evaluation weights; null for the defaults
	Search  [2]*ai.MCTSConfig `json:"search"`  // A's and B's search settings; null for their levels'
	Seed    uint64            `json:"seed"`
	Games   int               `json:"games"` // in the whole match, all shards together
	Clock   bool              `json:"clock"` // medium and hard used their clock budgets
	Scale   float64           `json:"scale"`

	names [2]string  // labels for the report
	specs [2]botSpec // A and B, parsed
}

// header returns the match's header.
func (cfg *matchConfig) header() matchHeader {
	return matchHeader{A: cfg.a.String(), B: cfg.b.String(), Budgets: cfg.budgets(), Weights: cfg.weights, Search: cfg.search,
		Seed: cfg.seed, Games: cfg.games, Clock: cfg.clock, Scale: cfg.scale,
		names: labels(cfg.a, cfg.b), specs: [2]botSpec{cfg.a, cfg.b}}
}

// labels names the two sides for the report, telling apart two bots of the
// same kind.
func labels(a, b botSpec) [2]string {
	as, bs := a.String(), b.String()
	if as == bs {
		return [2]string{as + " (A)", bs + " (B)"}
	}
	return [2]string{as, bs}
}

// describeBudget says what a bot searches per move.
func describeBudget(spec botSpec, b ai.Budget) string {
	if _, search := spec.level(); !search {
		if spec.name == "random" {
			return "no search"
		}
		k := b.MaxIterations
		if k == 0 {
			k = ai.DefaultGreedyK
		}
		return plural(k, "determinization")
	}
	workers := plural(max(b.Workers, 1), "worker")
	if b.MaxTime > 0 {
		return fmt.Sprintf("%s on %s", b.MaxTime, workers)
	}
	return fmt.Sprintf("%s on %s", plural(b.MaxIterations, "iteration"), workers)
}

// plural writes n and a noun, with an s unless n is 1.
func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(n) + " " + noun + "s"
}

// printMatch writes the report of a match, or of the part of it that st
// sums up. parallel is the number of games played at once, or 0 when the
// timing is unknown (tally).
func printMatch(w io.Writer, h *matchHeader, st *matchStats, elapsed time.Duration, parallel int) {
	names := h.names
	part := ""
	if st.games != h.Games {
		part = fmt.Sprintf(" of %d", h.Games)
	}
	fmt.Fprintf(w, "match %s vs %s: %d%s games (%d deals, each played with both seatings), seed %d\n",
		names[0], names[1], st.games, part, (st.games+1)/2, h.Seed)
	fmt.Fprintf(w, "budgets per move: %s %s; %s %s\n", names[0], describeBudget(h.specs[0], h.Budgets[0]),
		names[1], describeBudget(h.specs[1], h.Budgets[1]))
	for side, wt := range h.Weights {
		if wt != nil {
			fmt.Fprintf(w, "%s uses evaluation weights other than the defaults\n", names[side])
		}
	}
	for side, sc := range h.Search {
		if sc != nil {
			fmt.Fprintf(w, "%s uses search settings other than its level's: %+v\n", names[side], *sc)
		}
	}
	width := max(len(names[0]), len(names[1]), len("shared"))
	for side := range 2 {
		k := &st.wins[side]
		total := k[engine.VictoryMilitary] + k[engine.VictoryScience] + k[engine.VictoryCivilian]
		fmt.Fprintf(w, "  %-*s %4d wins: military %d, science %d, civilian %d\n", width, names[side], total,
			k[engine.VictoryMilitary], k[engine.VictoryScience], k[engine.VictoryCivilian])
	}
	fmt.Fprintf(w, "  %-*s %4d\n", width, "shared", st.shared)
	p := st.rate()
	lo, hi := wilson(p, st.games)
	fmt.Fprintf(w, "%s scores %.1f%% (95%% interval %.1f%% to %.1f%%; a shared victory counts half)\n",
		names[0], 100*p, 100*lo, 100*hi)
	fmt.Fprintf(w, "moves: %s; %s\n", moveTimes(names[0], st, 0), moveTimes(names[1], st, 1))
	avg := 0.0
	if st.games > 0 {
		avg = float64(st.plies) / float64(st.games)
	}
	if parallel > 0 {
		fmt.Fprintf(w, "games last %.1f actions on average; %s in all (%d in parallel)\n",
			avg, elapsed.Round(time.Millisecond), parallel)
	} else {
		fmt.Fprintf(w, "games last %.1f actions on average\n", avg)
	}
}

// checkMin compares A's score with a minimum share (from 0 to 1; a negative
// minimum checks nothing) and prints the verdict. It returns errFailed when
// the score is below the minimum or some of the h.Games games are missing.
func checkMin(w io.Writer, h *matchHeader, st *matchStats, minRate float64) error {
	if minRate < 0 {
		return nil
	}
	p := st.rate()
	switch {
	case st.games < h.Games:
		fmt.Fprintf(w, "FAIL: only %d of %d games were played\n", st.games, h.Games)
		return errFailed
	case p < minRate:
		fmt.Fprintf(w, "FAIL: %s scored %.1f%%, below the minimum of %.1f%%\n", h.names[0], 100*p, 100*minRate)
		return errFailed
	}
	fmt.Fprintf(w, "PASS: %s scored %.1f%%, at least the minimum of %.1f%%\n", h.names[0], 100*p, 100*minRate)
	return nil
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
func printGame(w io.Writer, names [2]string, g *gameResult) {
	r := &g.Result
	outcome := "shared victory"
	if r.Winner != engine.SharedVictory {
		outcome = names[g.sideOf(engine.Seat(r.Winner))] + " wins by " + r.Kind.String()
	}
	fmt.Fprintf(w, "game %d: seed %d, %s in seat 0: %s, %d-%d, %d actions\n", g.Index, g.GameSeed,
		names[g.sideOf(0)], outcome, r.Scores[0].Total, r.Scores[1].Total, g.Plies)
}

// matchFormat names the format of the -json files.
const matchFormat = "ajayeb-match/1"

// matchFile is the -json output of match and the input of tally.
type matchFile struct {
	Format string `json:"format"`
	matchHeader
	Shard   string       `json:"shard"` // "K/N", or "" for a whole match
	Results []gameResult `json:"results"`
}

// writeMatchFile writes a match's results as JSON.
func writeMatchFile(path string, h *matchHeader, sh shard, results []gameResult) error {
	f := matchFile{Format: matchFormat, matchHeader: *h, Shard: sh.String(), Results: results}
	b, err := json.MarshalIndent(&f, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

// runMatch is the match subcommand.
func runMatch(args []string, stdout, stderr io.Writer) error {
	fs := newFlags("match", stderr)
	aText := fs.String("a", "greedy", "bot A")
	bText := fs.String("b", "random", "bot B")
	games := fs.Int("games", 100, "games to play (rounded up to an even number: each deal is played twice)")
	seed := fs.Uint64("seed", 1, "base seed of the deals and the bots")
	parallel := fs.Int("p", 2, "games played at once")
	scale := fs.Float64("scale", 1, "factor on the iteration budgets of medium and hard, on both sides")
	clock := fs.Bool("clock", false, "give medium and hard their clock budgets (results then depend on the machine)")
	waFile := fs.String("wa", "", "JSON file of evaluation weights for bot A (see ajayeb-sim weights)")
	wbFile := fs.String("wb", "", "JSON file of evaluation weights for bot B")
	saFile := fs.String("sa", "", "JSON file of search settings (ai.MCTSConfig) for bot A, a medium or hard bot")
	sbFile := fs.String("sb", "", "JSON file of search settings for bot B")
	shardText := fs.String("shard", "", "play only the deals whose index is K modulo N (K/N)")
	jsonFile := fs.String("json", "", "write the results to this file, for tally")
	minRate := fs.Float64("min", -1, "fail when A scores less than this share, from 0 to 1 (negative: no check)")
	logDir := fs.String("logs", "", "write each game's ajayeb-log/1 file into this directory")
	verbose := fs.Bool("v", false, "print one line per game")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "ajayeb-sim match: unexpected argument %q\n", fs.Arg(0))
		return errUsage
	}
	cfg := matchConfig{seed: *seed, parallel: *parallel, clock: *clock, scale: *scale, logDir: *logDir}
	var err error
	if cfg.a, err = parseBot(*aText); err == nil {
		cfg.b, err = parseBot(*bText)
	}
	if err == nil {
		cfg.shard, err = parseShard(*shardText)
	}
	switch {
	case err != nil:
	case *games < 1 || *games > 100_000:
		err = fmt.Errorf("-games %d is not 1 to 100000", *games)
	case *parallel < 1 || *parallel > 256:
		err = fmt.Errorf("-p %d is not 1 to 256", *parallel)
	case !(*scale > 0 && *scale <= 100):
		err = fmt.Errorf("-scale %v is not more than 0 and at most 100", *scale)
	case *minRate > 1:
		err = fmt.Errorf("-min %v is more than 1 (it is a share, from 0 to 1)", *minRate)
	}
	if err == nil {
		if cfg.weights[0], err = readWeights(*waFile); err == nil {
			cfg.weights[1], err = readWeights(*wbFile)
		}
	}
	if err == nil {
		if cfg.search[0], err = readSearch(*saFile, cfg.a); err == nil {
			cfg.search[1], err = readSearch(*sbFile, cfg.b)
		}
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

	idx := cfg.indexes()
	start := time.Now()
	results := playMatch(context.Background(), &cfg, idx)
	elapsed := time.Since(start)
	failed := false
	for i := range results {
		if err := results[i].err; err != nil {
			fmt.Fprintf(stderr, "ajayeb-sim match: game %d (seed %d): %v\n", results[i].Index, results[i].GameSeed, err)
			failed = true
		}
	}
	if failed {
		return errFailed
	}
	h := cfg.header()
	if *verbose {
		for i := range results {
			printGame(stdout, h.names, &results[i])
		}
	}
	if *jsonFile != "" {
		if err := writeMatchFile(*jsonFile, &h, cfg.shard, results); err != nil {
			return err
		}
	}
	st := summarize(results)
	printMatch(stdout, &h, &st, elapsed, max(1, min(cfg.parallel, len(idx))))
	if cfg.shard.n > 1 {
		h.Games = st.games // a shard checks -min on its own games
	}
	return checkMin(stdout, &h, &st, *minRate)
}

// runTally is the tally subcommand: it merges the -json files of a match's
// shards into one report.
func runTally(args []string, stdout, stderr io.Writer) error {
	fs := newFlags("tally", stderr)
	minRate := fs.Float64("min", -1, "fail when A scores less than this share, from 0 to 1, or games are missing (negative: no check)")
	verbose := fs.Bool("v", false, "print one line per game")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(stderr, "ajayeb-sim tally: name at least one -json file of match")
		return errUsage
	}
	if *minRate > 1 {
		fmt.Fprintf(stderr, "ajayeb-sim tally: -min %v is more than 1 (it is a share, from 0 to 1)\n", *minRate)
		return errUsage
	}
	h, results, err := readMatchFiles(fs.Args())
	if err != nil {
		return err
	}
	if *verbose {
		for i := range results {
			printGame(stdout, h.names, &results[i])
		}
	}
	st := summarize(results)
	printMatch(stdout, h, &st, 0, 0)
	if st.games < h.Games {
		fmt.Fprintf(stdout, "missing: %d of the match's %d games\n", h.Games-st.games, h.Games)
	}
	return checkMin(stdout, h, &st, *minRate)
}

// readMatchFiles reads and merges the -json files of one match, in game
// order. Every file must describe the same match, and no game may appear
// twice.
func readMatchFiles(paths []string) (*matchHeader, []gameResult, error) {
	var h *matchHeader
	var ref []byte
	var results []gameResult
	seen := make([]string, 0) // the file of each game seen, by index
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
		var f matchFile
		if err := json.Unmarshal(data, &f); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", path, err)
		}
		if f.Format != matchFormat {
			return nil, nil, fmt.Errorf("%s: format %q, want %q", path, f.Format, matchFormat)
		}
		key, err := json.Marshal(&f.matchHeader)
		if err != nil {
			return nil, nil, err
		}
		if h == nil {
			a, errA := parseBot(f.A)
			b, errB := parseBot(f.B)
			if errA != nil || errB != nil || f.Games < 1 || f.Games > 100_000 {
				return nil, nil, fmt.Errorf("%s: not a match of %q and %q over 1 to 100000 games", path, f.A, f.B)
			}
			hh := f.matchHeader
			hh.specs, hh.names = [2]botSpec{a, b}, labels(a, b)
			h, ref = &hh, key
			seen = make([]string, f.Games)
		} else if string(key) != string(ref) {
			return nil, nil, fmt.Errorf("%s: not the same match as %s (bots, budgets, weights, seed or games differ)", path, paths[0])
		}
		for _, g := range f.Results {
			if g.Index < 0 || g.Index >= h.Games {
				return nil, nil, fmt.Errorf("%s: game %d is outside the match's %d games", path, g.Index, h.Games)
			}
			if other := seen[g.Index]; other != "" {
				return nil, nil, fmt.Errorf("%s: game %d is also in %s", path, g.Index, other)
			}
			seen[g.Index] = path
			results = append(results, g)
		}
	}
	slices.SortFunc(results, func(x, y gameResult) int { return x.Index - y.Index })
	return h, results, nil
}
