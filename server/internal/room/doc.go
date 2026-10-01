// Package room runs game rooms. A room owns one game and its mode (online,
// hot-seat or against a bot), the seats and who holds them, the action log,
// per-client idempotency state, bot scheduling, undo and persistence.
//
// # Actor model
//
// Each room is an actor: a single goroutine owns all room state and processes
// messages from its inbox channel one at a time, so room logic needs no locks.
// Connections, bots and the reaper talk to a room only by sending to its
// inbox. Outbound state goes to each connection's latest-wins outbox, so a
// slow client never blocks the room.
//
// # Rules
//
//   - The room applies an action only when its key appears in the engine's
//     legal actions for the seat to move. It never builds actions from client
//     data.
//   - act frames carry seq and base. A duplicate seq from the same client
//     re-sends the current state without applying anything; a base that is
//     not the current version is rejected as stale and followed by a fresh
//     state.
//   - A bot runs off the actor goroutine on an engine Observation and reports
//     its key back through the inbox together with the version it saw; a
//     result that went stale (for example after an undo) is dropped.
//   - Hot-seat and bot rooms accept only the creator's user id, and only they
//     support undo. Online rooms seat the first two distinct users; later
//     joiners are spectators.
//   - Time comes from an injected clock, never from time.Now in room logic, so
//     bot delays and idle expiry are testable with a fake clock.
//   - Logging goes only through an injected *slog.Logger, and tokens never
//     appear in logs.
package room
