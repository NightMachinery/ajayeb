package ai

import (
	"runtime"
	"time"

	"github.com/NightMachinery/ajayeb/server/internal/engine"
)

// This file holds the level presets: how much each level the rooms offer
// searches per move. Slice S12 froze them after measured tournaments, and
// slice S13 gave Hard rollouts to the end of the game (HardRollout) with a
// fixed budget re-measured for them; docs/ai.md has the measurements and
// the commands that rerun them.
//
// Every level has two budgets:
//
//   - the clock budget (Level.Budget), which a live game uses: the tree
//     search thinks for a fixed time, so a player never waits longer than
//     that, and the bot plays as well as the machine allows;
//   - the fixed budget (Level.FixedBudget), the clock budget's iteration
//     equivalent: about the work the clock budget buys on the reference
//     machine (an Apple M2, ajayeb-sim levels measures it), as a number of
//     iterations. A search under it is deterministic, so tournaments and
//     tests use it, and their results are the same on any machine and at
//     any load.
//
// The presets:
//
//   - Random plays uniformly at random. It needs no budget.
//   - Easy is Greedy with softmax sampling at EasyTemperature, over
//     DefaultGreedyK determinizations per move (a few milliseconds).
//   - Medium is the tree search for MediumTime on MediumWorkers worker;
//     its fixed budget is MediumIterations.
//   - Hard is the tree search with HardRollout for HardTime on up to
//     HardWorkers workers, never more than GOMAXPROCS in a live game; its
//     fixed budget is HardIterations over exactly HardWorkers workers,
//     whatever the machine, since the worker count changes the trees and so
//     the moves.
//
// A fixed budget is a median: Medium's search runs about twice as many
// iterations per second late in Age III as in the draft, and Hard's, whose
// rollouts are longest early on, about six times as many, so in a live game
// Medium and Hard search less than their fixed budgets early on and more
// late in the game.

// The presets of the tree-search levels.
const (
	MediumTime       = 400 * time.Millisecond // Medium's thinking time per move
	MediumWorkers    = 1                      // Medium's search workers
	MediumIterations = 40_000                 // Medium's fixed budget: iterations per move

	HardTime       = 2 * time.Second // Hard's thinking time per move
	HardWorkers    = 4               // Hard's search workers (in a live game, at most GOMAXPROCS)
	HardIterations = 200_000         // Hard's fixed budget: iterations per move, over all the workers
)

// HardRollout is Hard's rollout: the default policy played to the end of
// the game instead of being cut off at the end of the age or after 16 plies
// and scored by the evaluation. docs/ai.md explains why.
var HardRollout = RolloutConfig{Plies: MaxGamePlies, Ages: engine.NumAges}

// Budget returns the level's clock budget, the default budget of the bot
// that New returns for it: what a live game uses.
func (l Level) Budget() Budget {
	switch l {
	case Easy:
		return Budget{MaxIterations: DefaultGreedyK}
	case Medium:
		return Budget{MaxTime: MediumTime, Workers: MediumWorkers}
	case Hard:
		return Budget{MaxTime: HardTime, Workers: min(HardWorkers, runtime.GOMAXPROCS(0))}
	}
	return Budget{}
}

// FixedBudget returns the level's fixed budget: the iteration equivalent of
// its clock budget, under which its bot is deterministic. Tournaments and
// tests pass it to Choose.
func (l Level) FixedBudget() Budget {
	switch l {
	case Easy:
		return Budget{MaxIterations: DefaultGreedyK}
	case Medium:
		return Budget{MaxIterations: MediumIterations, Workers: MediumWorkers}
	case Hard:
		return Budget{MaxIterations: HardIterations, Workers: HardWorkers}
	}
	return Budget{}
}

// SearchConfig returns the tree-search settings of Medium or Hard, with the
// level's clock budget as the default budget; the zero MCTSConfig for the
// other levels. New builds the level's bot from it.
func (l Level) SearchConfig() MCTSConfig {
	switch l {
	case Medium:
		return MCTSConfig{Budget: l.Budget()}
	case Hard:
		return MCTSConfig{Budget: l.Budget(), Rollout: HardRollout}
	}
	return MCTSConfig{}
}
