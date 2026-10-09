package main

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"runtime/metrics"
	"slices"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// The levels subcommand measures what the levels do per move: how many
// iterations the clock budgets of the tree-search levels buy on this
// machine (their fixed budgets are set from the medians), how long a move
// takes, and how much memory a search holds. Easy is timed too.

// levelsNodeBytes is the size of one search-tree node (arena.go). A search
// adds at most one node per iteration.
const levelsNodeBytes = 24

// heapMetric is the runtime metric of the bytes held by heap objects.
const heapMetric = "/memory/classes/heap/objects:bytes"

// levelMove is the measurement of one search.
type levelMove struct {
	age        int // 0 for the wonder draft
	iterations int
	elapsed    time.Duration
	heap       uint64 // the most heap the search held above the baseline, as sampled
}

// runLevels is the levels subcommand.
func runLevels(args []string, stdout, stderr io.Writer) error {
	fs := newFlags("levels", stderr)
	games := fs.Int("games", 12, "Greedy self-play games to take positions from")
	seed := fs.Uint64("seed", 1, "base seed of the games")
	moves := fs.Int("moves", 60, "positions to search, spread evenly over the games' moves")
	fixed := fs.Bool("fixed", false, "measure the fixed budgets instead of the clock budgets")
	scale := fs.Float64("scale", 1, "factor on the fixed budgets (with -fixed)")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	switch {
	case fs.NArg() > 0:
		fmt.Fprintf(stderr, "ajayeb-sim levels: unexpected argument %q\n", fs.Arg(0))
		return errUsage
	case *games < 1 || *games > 10_000 || *moves < 1 || *moves > 100_000 || !(*scale > 0 && *scale <= 100):
		fmt.Fprintln(stderr, "ajayeb-sim levels: -games must be 1 to 10000, -moves 1 to 100000 and -scale more than 0 and at most 100")
		return errUsage
	}
	ctx := context.Background()
	pos, err := levelPositions(ctx, *seed, *games, *moves)
	if err != nil {
		return err
	}
	kind := "clock"
	if *fixed {
		kind = "fixed"
	}
	fmt.Fprintf(stdout, "%s budgets on %d positions from %d Greedy games (GOMAXPROCS %d)\n", kind, len(pos), *games, runtime.GOMAXPROCS(0))
	for _, l := range []ai.Level{ai.Easy, ai.Medium, ai.Hard} {
		spec, _ := parseBot(l.String())
		b := spec.budget(!*fixed, *scale)
		if l == ai.Easy {
			b = l.Budget()
		}
		printLevel(stdout, l, b, measureLevel(ctx, l, b, pos, *seed))
	}
	return nil
}

// levelPositions plays Greedy self-play games and returns up to want of the
// positions met, spread evenly over all the moves that had a choice.
func levelPositions(ctx context.Context, seed uint64, games, want int) ([]engine.State, error) {
	var all []engine.State
	buf := make([]engine.Action, 0, engine.MaxLegalActions)
	for g := range games {
		gameSeed := ai.MixSeed(seed, uint64(g))
		bots := [2]ai.Bot{ai.NewGreedy(ai.MixSeed(gameSeed, 1), ai.GreedyConfig{}), ai.NewGreedy(ai.MixSeed(gameSeed, 2), ai.GreedyConfig{})}
		s := engine.NewGame(gameSeed, engine.Options{})
		for !s.Over() {
			if len(engine.LegalActions(&s, buf[:0])) > 1 {
				all = append(all, s)
			}
			a, err := bots[s.ToMove].Choose(ctx, engine.Observe(&s, s.ToMove), ai.Budget{}, nil)
			if err != nil {
				return nil, fmt.Errorf("game %d, ply %d: %w", g, s.Ply, err)
			}
			if err := engine.Apply(&s, a, nil); err != nil {
				return nil, fmt.Errorf("game %d, ply %d: %w", g, s.Ply, err)
			}
		}
	}
	if len(all) <= want {
		return all, nil
	}
	out := make([]engine.State, want)
	for i := range out {
		out[i] = all[i*len(all)/want]
	}
	return out, nil
}

// measureLevel runs one search of level l under budget b on each position.
// A search that ends before searching (a sure win) is left out.
func measureLevel(ctx context.Context, l ai.Level, b ai.Budget, pos []engine.State, seed uint64) []levelMove {
	bot := ai.New(l, seed)
	sample := []metrics.Sample{{Name: heapMetric}}
	heapNow := func() uint64 {
		metrics.Read(sample)
		if sample[0].Value.Kind() != metrics.KindUint64 {
			return 0
		}
		return sample[0].Value.Uint64()
	}
	var out []levelMove
	for i := range pos {
		s := &pos[i]
		// Two collections empty the pool of recycled tree chunks, so the
		// search allocates its tree afresh and the heap shows it.
		runtime.GC()
		runtime.GC()
		base := heapNow()
		var last ai.Progress
		peak := uint64(0)
		reported := false
		start := time.Now()
		_, err := bot.Choose(ctx, engine.Observe(s, s.ToMove), b, func(p ai.Progress) {
			last, reported = p, true
			peak = max(peak, heapNow())
		})
		elapsed := time.Since(start)
		if err != nil || !reported {
			continue
		}
		m := levelMove{age: int(s.Age), iterations: last.Iterations, elapsed: elapsed}
		if peak > base {
			m.heap = peak - base
		}
		out = append(out, m)
	}
	return out
}

// printLevel writes a level's measurements.
func printLevel(w io.Writer, l ai.Level, b ai.Budget, moves []levelMove) {
	spec, _ := parseBot(l.String())
	fmt.Fprintf(w, "\n%s (%s): %d searches\n", l, describeBudget(spec, b), len(moves))
	if len(moves) == 0 {
		return
	}
	iters := make([]int, len(moves))
	times := make([]time.Duration, len(moves))
	heap := make([]uint64, len(moves))
	for i, x := range moves {
		iters[i], times[i], heap[i] = x.iterations, x.elapsed, x.heap
	}
	slices.Sort(times)
	var total time.Duration
	for _, d := range times {
		total += d
	}
	timeLine := fmt.Sprintf("  time per move: mean %s, median %s, max %s\n", ms1(total/time.Duration(len(times))),
		ms1(times[len(times)/2]), ms1(times[len(times)-1]))
	slices.Sort(heap)
	heapLine := fmt.Sprintf("  heap held during a search (sampled every 100 ms and at the end): median %.2f MiB, max %.2f MiB\n",
		float64(heap[len(heap)/2])/(1<<20), float64(heap[len(heap)-1])/(1<<20))
	if _, search := spec.level(); !search {
		fmt.Fprint(w, timeLine, heapLine)
		return
	}
	fmt.Fprintf(w, "  iterations per move: %s\n", intSpread(iters))
	ageNames := [...]string{"draft", "Age I", "Age II", "Age III"}
	for age, name := range ageNames {
		var sub []int
		for _, x := range moves {
			if x.age == age {
				sub = append(sub, x.iterations)
			}
		}
		if len(sub) > 0 {
			fmt.Fprintf(w, "    %-8s %s\n", name+":", intSpread(sub))
		}
	}
	fmt.Fprint(w, timeLine)
	slices.Sort(iters)
	workers := max(b.Workers, 1)
	treeMax := float64((iters[len(iters)-1]+workers)*levelsNodeBytes) / (1 << 20)
	treeMed := float64((iters[len(iters)/2]+workers)*levelsNodeBytes) / (1 << 20)
	fmt.Fprintf(w, "  search trees (%d bytes a node, at most one node per iteration): median %.1f MiB, max %.1f MiB over all workers\n",
		levelsNodeBytes, treeMed, treeMax)
	fmt.Fprint(w, heapLine)
}

// intSpread describes a sample of counts: its 10th percentile, median, mean
// and 90th percentile.
func intSpread(v []int) string {
	s := slices.Clone(v)
	slices.Sort(s)
	total := 0
	for _, x := range s {
		total += x
	}
	at := func(q int) int { return s[min(len(s)-1, q*len(s)/100)] }
	return fmt.Sprintf("p10 %d, median %d, mean %d, p90 %d (%d moves)", at(10), at(50), total/len(s), at(90), len(s))
}

// ms1 formats a duration in milliseconds with one decimal.
func ms1(d time.Duration) string {
	return fmt.Sprintf("%.1f ms", float64(d)/float64(time.Millisecond))
}
