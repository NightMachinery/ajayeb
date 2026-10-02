package main

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"slices"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// runBench is the bench subcommand: engine timings, bot move times, and
// short tournaments.
func runBench(args []string, stdout, stderr io.Writer) error {
	fs := newFlags("bench", stderr)
	games := fs.Int("games", 40, "games per tournament (rounded up to an even number)")
	seed := fs.Uint64("seed", 1, "base seed")
	parallel := fs.Int("p", 2, "tournament games played at once")
	positions := fs.Int("positions", 60, "positions for the per-call timings and the move times")
	playouts := fs.Int("playouts", 500, "random playouts for the engine timing")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	switch {
	case fs.NArg() > 0:
		fmt.Fprintf(stderr, "ajayeb-sim bench: unexpected argument %q\n", fs.Arg(0))
		return errUsage
	case *games < 2 || *parallel < 1 || *positions < 1 || *playouts < 1:
		fmt.Fprintln(stderr, "ajayeb-sim bench: -games must be at least 2, and -p, -positions and -playouts at least 1")
		return errUsage
	}

	start := time.Now()
	fmt.Fprintf(stdout, "engine, one core (%d random playouts):\n", *playouts)
	pos := benchPlayouts(stdout, *seed, *playouts, *positions)
	benchCalls(stdout, pos)

	fmt.Fprintf(stdout, "move times over %d positions (default budgets, one core):\n", len(pos))
	for _, name := range []string{"random", "easy", "greedy", "hard"} {
		spec, _ := parseBot(name)
		if err := benchMoves(stdout, spec, pos, *seed); err != nil {
			return err
		}
	}

	for _, pair := range [][2]string{{"greedy", "random"}, {"easy", "random"}, {"greedy", "easy"}} {
		a, _ := parseBot(pair[0])
		b, _ := parseBot(pair[1])
		cfg := matchConfig{a: a, b: b, games: *games + *games%2, seed: *seed, parallel: *parallel}
		t0 := time.Now()
		results := playMatch(context.Background(), &cfg)
		for i := range results {
			if err := results[i].err; err != nil {
				return fmt.Errorf("%s vs %s, game %d: %w", pair[0], pair[1], i, err)
			}
		}
		st := summarize(results)
		fmt.Fprintln(stdout)
		printMatch(stdout, &cfg, &st, time.Since(t0))
	}
	fmt.Fprintf(stdout, "\nbench done in %s\n", time.Since(start).Round(time.Millisecond))
	return nil
}

// benchPlayouts times random playouts (LegalActions and ApplyTrusted only)
// and returns up to want positions met along the way, from every phase.
func benchPlayouts(w io.Writer, seed uint64, n, want int) []engine.State {
	rng := engine.NewRNG(seed)
	legal := make([]engine.Action, 0, engine.MaxLegalActions)
	var pos []engine.State
	plies := 0
	start := time.Now()
	for i := range n {
		s := engine.NewGame(ai.MixSeed(seed, uint64(i)), engine.Options{})
		pick := rng.IntN(90)
		for !s.Over() {
			if len(pos) < want && int(s.Ply) == pick {
				pos = append(pos, s)
			}
			legal = engine.LegalActions(&s, legal[:0])
			engine.ApplyTrusted(&s, legal[rng.IntN(len(legal))])
			plies++
		}
	}
	el := time.Since(start)
	fmt.Fprintf(w, "  random playout     %8.1f us per game, %.1f actions per game, %.2f M actions/s\n",
		float64(el.Microseconds())/float64(n), float64(plies)/float64(n), float64(plies)/el.Seconds()/1e6)
	if len(pos) == 0 {
		pos = append(pos, engine.NewGame(seed, engine.Options{}))
	}
	return pos
}

// benchCalls times the per-call hot paths over the positions.
func benchCalls(w io.Writer, pos []engine.State) {
	const reps = 20000
	obs := make([]engine.Observation, len(pos))
	for i := range pos {
		obs[i] = engine.Observe(&pos[i], pos[i].ToMove)
	}
	rng := engine.NewRNG(7)
	time1 := func(name string, f func(i int)) {
		start := time.Now()
		for i := range reps {
			f(i % len(pos))
		}
		fmt.Fprintf(w, "  %-18s %8.1f ns per call\n", name, float64(time.Since(start).Nanoseconds())/reps)
	}
	time1("Determinize", func(i int) {
		d := obs[i].Determinize(&rng)
		runtime.KeepAlive(d)
	})
	buf := make([]engine.Action, 0, engine.MaxLegalActions)
	time1("LegalActions", func(i int) {
		buf = engine.LegalActions(&pos[i], buf[:0])
		runtime.KeepAlive(buf)
	})
	time1("Evaluate", func(i int) {
		runtime.KeepAlive(ai.Evaluate(&pos[i], pos[i].ToMove))
	})
	time1("Observe", func(i int) {
		o := engine.Observe(&pos[i], 0)
		runtime.KeepAlive(o)
	})
}

// benchMoves times one bot's choices over the positions that are not over.
func benchMoves(w io.Writer, spec botSpec, pos []engine.State, seed uint64) error {
	bot := spec.make(seed)
	var times []time.Duration
	for i := range pos {
		s := &pos[i]
		if s.Over() {
			continue
		}
		t0 := time.Now()
		if _, err := bot.Choose(context.Background(), engine.Observe(s, s.ToMove), ai.Budget{}, nil); err != nil {
			return fmt.Errorf("%s at ply %d: %w", spec, s.Ply, err)
		}
		times = append(times, time.Since(t0))
	}
	if len(times) == 0 {
		return nil
	}
	slices.Sort(times)
	var total time.Duration
	for _, d := range times {
		total += d
	}
	fmt.Fprintf(w, "  %-8s mean %s, median %s, max %s\n", spec, ms(total/time.Duration(len(times))),
		ms(times[len(times)/2]), ms(times[len(times)-1]))
	return nil
}
