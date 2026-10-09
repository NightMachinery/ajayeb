package ai_test

import (
	"runtime"
	"testing"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/ai"
)

// TestLevelPresetsAreFrozen pins the presets that docs/ai.md measured: a
// change here must come with new tournaments and a new docs/ai.md.
func TestLevelPresetsAreFrozen(t *testing.T) {
	if ai.MediumTime != 400*time.Millisecond || ai.MediumWorkers != 1 || ai.MediumIterations != 40_000 {
		t.Errorf("Medium: %v on %d workers, fixed %d iterations", ai.MediumTime, ai.MediumWorkers, ai.MediumIterations)
	}
	if ai.HardTime != 2*time.Second || ai.HardWorkers != 4 || ai.HardIterations != 500_000 {
		t.Errorf("Hard: %v on %d workers, fixed %d iterations", ai.HardTime, ai.HardWorkers, ai.HardIterations)
	}
	if ai.EasyTemperature != 1.5 || ai.DefaultGreedyK != 3 {
		t.Errorf("Easy: temperature %v, %d determinizations", ai.EasyTemperature, ai.DefaultGreedyK)
	}
}

func TestLevelBudgets(t *testing.T) {
	procs := runtime.GOMAXPROCS(0)
	cases := []struct {
		l            ai.Level
		clock, fixed ai.Budget
	}{
		{ai.Random, ai.Budget{}, ai.Budget{}},
		{ai.Easy, ai.Budget{MaxIterations: 3}, ai.Budget{MaxIterations: 3}},
		{ai.Medium, ai.Budget{MaxTime: 400 * time.Millisecond, Workers: 1}, ai.Budget{MaxIterations: 40_000, Workers: 1}},
		{ai.Hard, ai.Budget{MaxTime: 2 * time.Second, Workers: min(4, procs)}, ai.Budget{MaxIterations: 500_000, Workers: 4}},
	}
	for _, c := range cases {
		if got := c.l.Budget(); got != c.clock {
			t.Errorf("%s: clock budget %+v, want %+v", c.l, got, c.clock)
		}
		if got := c.l.FixedBudget(); got != c.fixed {
			t.Errorf("%s: fixed budget %+v, want %+v", c.l, got, c.fixed)
		}
	}
}

// TestHardFixedBudgetIgnoresTheMachine: on a machine with fewer cores, a
// live Hard search runs fewer workers, but the fixed budget keeps four, so
// a tournament plays the same moves everywhere.
func TestHardFixedBudgetIgnoresTheMachine(t *testing.T) {
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(1))
	if got := ai.Hard.Budget().Workers; got != 1 {
		t.Errorf("with GOMAXPROCS 1, Hard's clock budget has %d workers", got)
	}
	if got := ai.Hard.FixedBudget().Workers; got != ai.HardWorkers {
		t.Errorf("with GOMAXPROCS 1, Hard's fixed budget has %d workers", got)
	}
}

// TestFixedBudgetsAreDeterministic: a level's bot on its fixed budget (here
// scaled down) chooses the same move twice, also with Hard's four workers.
func TestFixedBudgetsAreDeterministic(t *testing.T) {
	s := greedyMid(t)
	for _, l := range []ai.Level{ai.Easy, ai.Medium, ai.Hard} {
		b := l.FixedBudget()
		if l != ai.Easy {
			b.MaxIterations /= 100
		}
		first := aiChoose(t, ai.New(l, 7), &s, b)
		if again := aiChoose(t, ai.New(l, 7), &s, b); again != first {
			t.Errorf("%s chose %s, then %s", l, first, again)
		}
	}
}
