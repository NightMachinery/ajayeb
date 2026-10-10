// Package ws serves the /ws WebSocket endpoint using github.com/coder/websocket.
//
// It upgrades connections, says hello, decodes and validates client frames
// into wire types, routes them to the rooms (room.Manager, the bounded
// registry of room actors), and writes the rooms' frames back.
//
// # A connection
//
// The first frame must be a hello in the supported protocol version, within
// the hello timeout; the server derives the user id from its token (the
// token goes no further) and answers welcome. A bad hello gets an error
// frame (bad-hello, or protocol for another version) and the socket is
// closed with status 1008. After hello, create, join, act, undo, resync,
// leave and rematch go to the rooms one at a time, in the order they
// arrive; their outcomes come back as frames in the connection's outbox
// (created, state, reject, error), which a writer goroutine drains. The
// outbox is latest-wins for state frames, so a slow client only ever gets
// the newest snapshot, and a client that lets other frames pile up is
// closed (1008). When a socket closes, its client is detached from its
// room; its seat stays with its user.
//
// # Limits
//
//   - Origins: Accept checks the Origin header against the configured site
//     origins (and the request's own host). With Dev, loopback origins on
//     any port are accepted too, for the Vite dev server.
//   - Frames: at most ReadLimit (8 KiB) each, or the socket closes with
//     1009. Text frames only (1003 otherwise). A token bucket of 20 frames
//     refilled at 10 per second per connection; a frame over it is answered
//     with reject rate and not handled, and after 20 such frames in a row the
//     socket closes (1008).
//   - Sockets: at most 16 per client address and 1024 in all; more are
//     refused before the upgrade (429 and 503). The client address is the
//     TCP peer, or, from a trusted proxy only, the rightmost untrusted
//     address of X-Forwarded-For. Every per-address limit counts an IPv4
//     address on its own and an IPv6 address by its /64 (limitKey), since
//     one host can use every address of its /64.
//   - Room creation: token buckets of 20 per hour per user id and 60 per
//     hour per client address; a create over either is answered with reject
//     rate, and so is a rematch that would create its room (the rooms ask
//     room.Client.AllowCreate).
//   - Joins of unknown rooms: a token bucket of 30 per hour per client
//     address, used up by every join that finds no room; past it every join
//     of the address is answered with reject rate. This bounds guessing
//     from one address only. Someone with many /64s gets 30 misses an hour
//     from each, and misses go uncounted once the limiter is full of
//     addresses that are all still refilling (keyedLimiter.record), while a
//     room code has only 31^6 (about 8.9e8) values. A longer code, or an
//     invite secret for online rooms, is an open question.
//   - Undo: a token bucket of 60 per minute per client address, since every
//     applied undo rewrites a room file; an undo over it is answered with
//     reject rate.
//   - The keyed limiters remember a bounded number of keys.
//   - Liveness: a ping every 25 seconds; at the tick after two pings in a
//     row went unanswered the socket is dropped.
//
// Every limit is a field of Limits, and every timer and rate limit reads the
// injected Clock (room.Clock), so tests drive them with a fake clock instead
// of sleeping.
//
// # Shutdown
//
// Server.Shutdown refuses new sockets, closes every open one with status 1012
// (service restart) so clients reconnect with backoff, and waits for the
// connections to finish. The rooms are already on disk.
//
// # Rules
//
//   - This package holds no game state and never interprets rules; rooms do.
//   - Logging goes only through an injected *slog.Logger. Tokens are never
//     logged and never sent back.
//   - Every map and channel is bounded: sockets by the socket limits, the
//     creation limiters by their key capacity, the outbox by room.Outbox's
//     own bounds.
package ws
