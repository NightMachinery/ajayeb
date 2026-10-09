// Package protocoltest end-to-end tests the real server stack: package
// store persisting rooms, package room running them, package ws serving
// /ws, and package httpapi assembling the HTTP surface, all wired together
// the way cmd/ajayeb wires them, and driven over real WebSocket connections
// through an httptest.Server. It is a black-box client of the protocol
// documented in docs/protocol.md: it never reaches into a room's state
// directly except through room.Manager.Snapshot, the same escape hatch a
// development tool would use, and only to establish ground truth for the
// leak and replay checks below.
//
// # What lives here
//
//   - harness_test.go boots a server (newServer) and can stop and rebuild
//     one on the same data directory (restart persistence).
//   - client_test.go is a minimal WebSocket client: it dials, decodes
//     frames into the real wire types, and captures every raw byte it
//     receives so the leak checks in redact_test.go have something to
//     scan.
//   - redact_test.go computes, from a room.Manager.Snapshot's ground-truth
//     engine.State, every card, wonder and token id that must not be
//     visible to a given viewer right now (a face-down slot, a removed or
//     undealt card, a box guild or wonder, an unrevealed draft round two,
//     or the Great Library offer when the viewer is not its decider), plus
//     the room's seed, which the view package never includes at all. This
//     is a best-effort reconstruction from the exported, non-test engine
//     API (the engine package itself exports no redaction helper), not a
//     byte-for-byte mirror of view.go's own whitelist logic, but precise
//     enough to catch a real leak: every id it tracks is unique within one
//     game.
//   - drive_test.go plays full games end to end (online with two distinct
//     users plus a spectator, hot-seat with one user holding both seats,
//     and ai with a bot seat), picking a uniformly random legal choice
//     each turn, and checks for leaks after every turn.
//   - fullgame_test.go, reconnect_test.go, restart_test.go,
//     idempotency_test.go and undo_test.go hold the tests themselves.
//   - nullarrays_test.go scans internal/wire/testdata/messages for a JSON
//     null where the "never nil slices" rule (doc.go of package wire)
//     promises an empty array instead.
//
// # Rules
//
//   - Every server this package starts is also stopped: sockets, the
//     httptest server and the room manager, in that order, usually from
//     t.Cleanup so a failing assertion still cleans up.
//   - No fmt.Print anywhere (golangci-lint forbidigo); use t.Log/t.Fatal.
//   - A room's bots run with a one-iteration budget and no think delay, so
//     full games finish in milliseconds; this governs correctness, not
//     bot strength.
//   - Tests run under -race (make test-race and this package's own
//     acceptance criterion), so state shared with a reader goroutine goes
//     through a channel or a mutex, never a bare field.
package protocoltest
