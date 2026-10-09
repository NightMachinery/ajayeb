// Package ai implements the bots: Random, Greedy (one ply plus the opponent's
// best reply, over a few determinizations) and SO-ISMCTS (single-observer
// information-set Monte Carlo tree search) with blind reveals, evaluation
// cut-offs and root-parallel workers.
//
// # Files
//
//   - bot.go: the Bot interface, Budget and Progress, the levels and New,
//     and PlayGame, which plays a whole game between two bots (for
//     tournaments and tests).
//   - levels.go: the level presets, each a clock budget for live games and
//     its iteration equivalent for deterministic tournaments.
//   - pool.go: Pool, the server-wide semaphore on concurrent searches.
//   - random.go: the Random bot.
//   - eval.go: Evaluate and its Weights, the static evaluation in victory
//     point equivalents.
//   - greedy.go: the Greedy bot; with a softmax temperature it is the Easy
//     level.
//   - arena.go: the search tree's node arena and action keys.
//   - rollout.go: the rollout policy and the reward.
//   - mcts.go: MCTSBot, the SO-ISMCTS search (Medium and Hard).
//   - parallel.go: root-parallel workers, merging and progress.
//
// Medium and Hard are the tree search with clock budgets of 0.4 s on one
// worker and 2 s on up to four workers. The tournaments that check each
// level against the one below are in tournament_test.go, behind the slow
// build tag; the command ajayeb-sim runs any pairing, and docs/ai.md has the
// measured results.
//
// # Rules
//
//   - A bot never sees the real game state. It receives an engine
//     Observation, which holds only what its seat may know, and samples the
//     hidden information with Observation.Determinize. Two observations with
//     the same public information therefore lead to the same choice under
//     the same seed; a test checks it.
//   - Bots are deterministic for a given seed, observation and iteration
//     budget: each Choose call derives its random stream from the bot's seed
//     and the observation, never from earlier calls. A wall-clock budget is
//     the only allowed source of nondeterminism. The room derives the bot's
//     seed from the room seed, so a game against a bot can be reproduced.
//   - Every search honours context cancellation and returns the best action
//     found so far, and a server-wide pool (a semaphore) bounds how many
//     searches run at once. Search memory is fixed-size or sized to the
//     budget, never unbounded.
//   - No I/O: non-test code must not import os, net, log (including
//     log/slog), io/ioutil, unsafe or math/rand (v1). Randomness comes from
//     the engine's seeded RNG. depguard enforces the import bans.
//   - Unlike the engine, the ai package may use goroutines, time budgets and
//     floating point (the evaluation function is in VP equivalents).
package ai
