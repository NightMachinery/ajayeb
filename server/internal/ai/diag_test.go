//go:build diag

package ai

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// Diagnostics of the tree search (go test -tags diag): how the searches of
// several settings and budgets agree on a fixed set of positions, how
// concentrated their root visits are and how deep their trees grow.
//
// DIAG_SPECS lists the searches, separated by ";": each is
// "name:iterations:workers[:key=value...]" with keys c (UCB constant),
// plies, eps, mode, ages (the rollout settings) and seed (the bot seed).
// For example:
//
//	DIAG_SPECS='M:40000:1;H:500000:4;H2:500000:4:seed=2' go test -tags diag -run TestDiagSearch -v ./internal/ai
//
// prints, for each search, the mean share of the root's visits that went
// to the move played, the depth of the principal variation and the cost per
// iteration, then how often each pair of searches chose the same move.
// DIAG_POS is the number of positions (default 48), DIAG_P the number of
// positions searched at once (default 1).

type diagSpec struct {
	name    string
	iters   int
	workers int
	cfg     MCTSConfig
	seed    uint64
}

func diagParse(t *testing.T, text string) []diagSpec {
	var out []diagSpec
	for _, part := range strings.Split(text, ";") {
		f := strings.Split(strings.TrimSpace(part), ":")
		if len(f) < 3 {
			t.Fatalf("bad spec %q", part)
		}
		sp := diagSpec{name: f[0], seed: 1}
		var err error
		if sp.iters, err = strconv.Atoi(f[1]); err != nil {
			t.Fatal(err)
		}
		if sp.workers, err = strconv.Atoi(f[2]); err != nil {
			t.Fatal(err)
		}
		for _, kv := range f[3:] {
			k, v, _ := strings.Cut(kv, "=")
			x, err := strconv.ParseFloat(v, 64)
			if err != nil {
				t.Fatalf("%s: %v", kv, err)
			}
			if !diagSet(&sp, k, x) {
				t.Fatalf("unknown key %q", k)
			}
		}
		out = append(out, sp)
	}
	return out
}

// diagPositions returns positions from deterministic Greedy self-play.
func diagPositions(t *testing.T, n int) []engine.State {
	var out []engine.State
	for seed := uint64(101); len(out) < n; seed++ {
		s := engine.NewGame(seed, engine.Options{})
		bots := [2]Bot{NewGreedy(seed, GreedyConfig{}), NewGreedy(seed+1, GreedyConfig{})}
		k := 0
		for !s.Over() {
			legal := engine.LegalActions(&s, nil)
			if len(legal) > 1 && k%7 == int(seed%7) && len(out) < n {
				out = append(out, s)
			}
			k++
			a, err := bots[s.ToMove].Choose(context.Background(), engine.Observe(&s, s.ToMove), Budget{MaxIterations: 1}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if err := engine.Apply(&s, a, nil); err != nil {
				t.Fatal(err)
			}
		}
	}
	return out
}

type diagOut struct {
	move   engine.Action
	share  float64
	pv     int     // depth of the principal variation (most visited, at least 64 visits), worker 0
	mean   float64 // mean depth of the nodes, worker 0
	nodes  int
	top3   float64 // share of the three most visited root children
	dur    time.Duration
}

func diagRun(m *MCTSBot, s *engine.State, iters, workers int) diagOut {
	obs := engine.Observe(s, s.ToMove)
	legal, _ := choiceSetup(&obs, make([]engine.Action, 0, engine.MaxLegalActions))
	rng := callRNG(m.seed, &obs)
	seeds := [2]uint64{rng.Uint64(), rng.Uint64()}
	first := obs.Determinize(&rng)
	r := m.newSearchRun(context.Background(), &obs, legal, searchPlan{iterations: iters, workers: workers}, seeds, &first, nil)
	defer r.finish()
	r.runWorkers()
	var o diagOut
	o.move = r.result()
	j := r.merged.best(len(legal))
	o.share = r.merged.share(j, len(legal))
	var sh []float64
	for k := range legal {
		sh = append(sh, r.merged.share(k, len(legal)))
	}
	for range 3 {
		bi := -1
		for k, v := range sh {
			if bi < 0 || v > sh[bi] {
				bi = k
			}
		}
		if bi >= 0 {
			o.top3 += sh[bi]
			sh[bi] = -1
		}
	}
	t := &r.workers[0].tree
	o.nodes = t.len()
	// The mean depth of the nodes, and the principal variation: the most
	// visited child, while it has at least 64 visits.
	type item struct {
		n int32
		d int
	}
	stack := []item{{0, 0}}
	sum := 0
	for len(stack) > 0 {
		it := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		sum += it.d
		for c := t.at(it.n).child; c != 0; c = t.at(c).sibling {
			stack = append(stack, item{c, it.d + 1})
		}
	}
	o.mean = float64(sum) / float64(o.nodes)
	v := int32(0)
	for {
		best, bv := int32(0), uint32(63)
		for c := t.at(v).child; c != 0; c = t.at(c).sibling {
			if n := t.at(c); n.visits > bv {
				best, bv = c, n.visits
			}
		}
		if best == 0 {
			break
		}
		o.pv++
		v = best
	}
	return o
}

func TestDiagSearch(t *testing.T) {
	specs := diagParse(t, os.Getenv("DIAG_SPECS"))
	npos := 48
	if v := os.Getenv("DIAG_POS"); v != "" {
		npos, _ = strconv.Atoi(v)
	}
	par := 1
	if v := os.Getenv("DIAG_P"); v != "" {
		par, _ = strconv.Atoi(v)
	}
	pos := diagPositions(t, npos)
	res := make([][]diagOut, len(specs))
	for i := range res {
		res[i] = make([]diagOut, len(pos))
	}
	var wg sync.WaitGroup
	sem := make(chan struct{}, par)
	for i, sp := range specs {
		m := NewMCTS(sp.seed, sp.cfg)
		for k := range pos {
			sem <- struct{}{}
			wg.Go(func() {
				defer func() { <-sem }()
				start := time.Now()
				res[i][k] = diagRun(m, &pos[k], sp.iters, sp.workers)
				res[i][k].dur = time.Since(start)
			})
		}
	}
	wg.Wait()
	for i, sp := range specs {
		var share, pv, mean, top3, nodes float64
		var dur time.Duration
		for _, o := range res[i] {
			dur += o.dur
			share += o.share
			pv += float64(o.pv)
			mean += o.mean
			top3 += o.top3
			nodes += float64(o.nodes)
		}
		n := float64(len(pos))
		fmt.Printf("%-10s best share %.3f top3 %.3f pv depth %.2f mean node depth %.2f nodes %.0f, %.1f us/iteration-worker\n", sp.name, share/n, top3/n, pv/n, mean/n, nodes/n,
			float64(dur.Microseconds())*float64(sp.workers)/float64(sp.iters)/n)
	}
	fmt.Printf("agreement (share of positions with the same move):\n%-10s", "")
	for _, sp := range specs {
		fmt.Printf(" %8s", sp.name)
	}
	fmt.Println()
	for i, a := range specs {
		fmt.Printf("%-10s", a.name)
		for j := range specs {
			same := 0
			for k := range pos {
				if res[i][k].move == res[j][k].move {
					same++
				}
			}
			fmt.Printf(" %8.2f", float64(same)/float64(len(pos)))
		}
		fmt.Println()
	}
}

func diagSet(sp *diagSpec, k string, x float64) bool {
	switch k {
	case "c":
		sp.cfg.C = x
	case "plies":
		sp.cfg.Rollout.Plies = int(x)
	case "eps":
		sp.cfg.Rollout.Epsilon = x
	case "mode":
		sp.cfg.Rollout.ModeProb = x
	case "ages":
		sp.cfg.Rollout.Ages = int(x)
	case "seed":
		sp.seed = uint64(x)
	default:
		return false
	}
	return true
}
