// Package tabletest is the N-seat interleaving harness for Haftgane's
// decision points (docs/haftgane/platform-architecture.md, "Decision points
// and acceptance"): it submits every required seat's picks from one common
// snapshot, in a seeded random or exhaustively enumerated arrival order,
// with drafts, re-sent (duplicate or stale) seqs and racing confirmations,
// checks each arrival's outcome against the acceptance rules, and compares
// the canonical batches and the final rules hash across schedules.
//
// It talks to a table only through the Table interface, so the transport
// is pluggable: today package direct implements it in process over the fake
// engine (a small model of the actor's acceptance rules); the Haftgane room
// actor (M16.haftgane-room-actor) implements it in its tests directly, and
// over WebSocket once its frames exist. Nothing here imports an engine.
//
// The harness is deterministic: a schedule is a function of the
// snapshots, the pick seed and the arrival seed, so the same seeds give
// the same submissions in the same order and, with the same table, the same
// trace and canonical batches.
//
// Subpackages: direct (the in-process table), leakcheck (the hand-game
// leak oracle) and faults (storage, crash, pending-restore, cancellation and
// no-human fixtures). All of it is test infrastructure; see fakeengine's
// package documentation for who may import it.
package tabletest
