// Package ai implements the bots: Random, Greedy (one ply plus the opponent's
// best reply) and SO-ISMCTS (single-observer information-set Monte Carlo tree
// search) with blind reveals, evaluation cut-offs and root-parallel workers.
//
// # Rules
//
//   - A bot never sees the real game state. It receives an engine
//     Observation, which holds only what its seat may know, and samples the
//     hidden information with Observation.Determinize. A fairness test checks
//     that two observations with the same public information lead to the same
//     choice under the same seed.
//   - Bots are deterministic for a given seed and iteration budget. A wall
//     clock budget is the only allowed source of nondeterminism. The room
//     derives the seed from the room seed and the ply, so a game against a
//     bot can be reproduced.
//   - Every search honours context cancellation, and a server-wide pool
//     (a semaphore) bounds how many searches run at once.
//   - No I/O: non-test code must not import os, net, log (including
//     log/slog), io/ioutil, unsafe or math/rand (v1). Randomness comes from
//     the engine's seeded RNG. depguard enforces the import bans.
//   - Unlike the engine, the ai package may use goroutines, time budgets and
//     floating point (the evaluation function is in VP equivalents).
package ai
