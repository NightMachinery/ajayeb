// Package wire defines the JSON messages exchanged between the browser and
// the server: client frames (hello, create, join, act, undo, resync, leave,
// rematch), server frames (welcome, created, state, reject, error), and the
// payloads of the small HTTP API.
//
// It is an input to tygo, which generates the TypeScript types the web client
// uses (see tygo.yaml), so the client and the server share one definition.
// The full game view embedded in state frames comes from package view.
//
// # Rules
//
//   - Plain data types and constants only, plus decoding and validation
//     helpers. No I/O, no rules logic and no room state.
//   - Every frame has a "t" discriminator whose TypeScript type is a literal,
//     set with a tstype tag such as tstype:"'state'".
//   - Every exported field has an explicit json tag. omitempty is used only on
//     pointer fields (optional in TypeScript), and slices are never nil, so
//     recorded messages contain no null arrays. A reflection test enforces
//     both rules.
//   - The server sends ids, error codes and structured events, never display
//     strings. This is what makes the client fully localisable.
//   - Tokens travel only inside the hello frame and never appear in any
//     server frame.
//   - hello and welcome carry the protocol version "v"; an incompatible change
//     bumps it.
//   - testdata/messages holds real messages recorded from a scripted game
//     (refreshed with go test -update). They become a TypeScript fixture that
//     the web type checker validates against the generated types.
package wire
