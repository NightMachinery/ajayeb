package ai

import (
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// The tests of parallel.go and of the budget plan: how a budget is split
// over the workers, how root statistics merge, and which child is played.

func TestPlanResolvesTheBudget(t *testing.T) {
	m := NewMCTS(1, MCTSConfig{})
	if p := m.plan(Budget{}); p.iterations != DefaultMCTSIterations || p.maxTime != 0 || p.workers != 1 {
		t.Errorf("a bot with no defaults plans %+v", p)
	}
	hard := NewMCTS(1, MCTSConfig{Budget: Budget{MaxTime: 2 * time.Second, Workers: 4}})
	if p := hard.plan(Budget{}); p.iterations != 0 || p.maxTime != 2*time.Second || p.workers != 4 {
		t.Errorf("the bot's default budget plans %+v", p)
	}
	// A budget that sets iterations replaces the default time limit, and
	// keeps the default worker count unless it sets its own.
	if p := hard.plan(Budget{MaxIterations: 100}); p.iterations != 100 || p.maxTime != 0 || p.workers != 4 {
		t.Errorf("an iteration budget plans %+v", p)
	}
	if p := hard.plan(Budget{MaxIterations: 2}); p.workers != 2 {
		t.Errorf("2 iterations get %d workers", p.workers)
	}
	if p := hard.plan(Budget{MaxIterations: 1000, Workers: 100}); p.workers != MaxSearchWorkers {
		t.Errorf("100 workers are capped at %d", p.workers)
	}
	if p := hard.plan(Budget{MaxTime: time.Second, Workers: -3}); p.workers != 4 || p.iterations != 0 {
		t.Errorf("a negative worker count plans %+v", p)
	}
}

// TestPlanSplitsTheIterations: the workers' shares add up to the budget,
// differ by at most one, and size each arena to the share plus the root.
func TestPlanSplitsTheIterations(t *testing.T) {
	for _, c := range []struct{ iterations, workers int }{{1, 1}, {10, 3}, {1000, 4}, {7, 7}, {100000, 8}} {
		p := searchPlan{iterations: c.iterations, workers: c.workers}
		total := 0
		for i := range c.workers {
			limit, nodes := p.worker(i)
			if limit < c.iterations/c.workers || limit > c.iterations/c.workers+1 {
				t.Errorf("%+v: worker %d runs %d", c, i, limit)
			}
			if nodes != min(limit+1, MaxArenaNodes) {
				t.Errorf("%+v: worker %d has %d nodes for %d iterations", c, i, nodes, limit)
			}
			total += limit
		}
		if total != c.iterations {
			t.Errorf("%+v: the shares add up to %d", c, total)
		}
	}
	unbounded := searchPlan{workers: 2}
	if limit, nodes := unbounded.worker(1); limit != 0 || nodes != MaxArenaNodes {
		t.Errorf("an unbounded worker gets %d iterations and %d nodes", limit, nodes)
	}
}

func TestRootStatsBest(t *testing.T) {
	var rs rootStats
	if j := rs.best(5); j != -1 || rs.share(j, 5) != 0 {
		t.Errorf("no visits: best %d", j)
	}
	rs.visits[1], rs.value[1] = 10, 4
	rs.visits[3], rs.value[3] = 10, 6
	rs.visits[4], rs.value[4] = 10, 6
	rs.visits[0], rs.value[0] = 9, 9
	if j := rs.best(5); j != 3 {
		t.Errorf("best = %d, want 3 (most visits, then most reward, then first)", j)
	}
	if j := rs.best(3); j != 1 {
		t.Errorf("best of the first 3 = %d, want 1", j)
	}
	if s := rs.share(3, 5); s != 10.0/39 {
		t.Errorf("share = %v", s)
	}
	var sum rootStats
	sum.add(&rs)
	sum.add(&rs)
	if sum.visits[3] != 20 || sum.value[0] != 18 || sum.best(5) != 3 {
		t.Errorf("the sum of two copies is %+v", sum)
	}
}

// TestSearchCancelledBeforeItStarts: a search whose context is done before
// the first iteration plays the rollout policy's move, which is legal.
func TestSearchCancelledBeforeItStarts(t *testing.T) {
	s := itestMid(t)
	m := NewMCTS(4, MCTSConfig{})
	x := itestNewSearch(t, t.Context(), m, &s, searchPlan{iterations: 1000, workers: 2}, nil)
	x.run.stop.Store(true)
	x.run.runWorkers()
	a := x.run.result()
	if x.run.merged.iterations != 0 {
		t.Fatalf("%d iterations ran", x.run.merged.iterations)
	}
	if err := engine.IsLegal(&s, a); err != nil {
		t.Fatalf("the fallback move %s: %v", a, err)
	}
}
