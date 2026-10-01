// Package ws serves the /ws WebSocket endpoint using github.com/coder/websocket.
//
// It upgrades connections, decodes and validates client frames into wire
// types, routes them to rooms, and writes server frames back.
//
// # Responsibilities
//
//   - Origin checks at upgrade: only the configured site origins, plus
//     localhost and 127.0.0.1 on any port when the server runs with --dev.
//   - Abuse limits: an 8 KiB read limit, a token bucket of 10 frames per
//     second with a burst of 20, a 10 second deadline for hello, and a cap on
//     sockets per client IP. X-Forwarded-For is believed only when the peer is
//     a configured trusted proxy.
//   - Liveness: a ping every 25 seconds; a connection that misses two pongs is
//     dropped.
//   - A latest-wins outbox per connection. State frames are full snapshots, so
//     a slow client only ever receives the newest one.
//   - On shutdown, sockets are closed with status 1012 (service restart) so
//     clients reconnect with backoff and no game is lost.
//
// # Rules
//
//   - This package holds no game state and never interprets rules; rooms do.
//   - Logging goes only through an injected *slog.Logger. Tokens are never
//     logged.
package ws
