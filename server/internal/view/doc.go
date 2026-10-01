// Package view defines the data transfer objects (DTOs) for a seat's view of
// a game: the structure of cards, both players' cities and cost breakdowns,
// military, tokens, the discard pile, pending choices, the legal choices of the
// deciding seat, the score preview and the result.
//
// The engine fills these types; this package only declares them. It is also
// an input to tygo, which turns it into TypeScript for the web client (see
// tygo.yaml).
//
// # Rules
//
//   - Types and constants only, plus trivial helpers. No rules logic, and no
//     import of the engine: the engine imports view, never the reverse.
//   - The same import bans as the engine apply (no time, os, net, log, sync,
//     unsafe, syscall, crypto/rand, io/ioutil or math/rand), enforced by
//     depguard.
//   - A view is a whitelist. The engine copies in only what the viewing seat
//     may see: a face-down card exposes only its back; the seed, removed
//     cards, future deals and box contents never appear before game over; the
//     Great Library offer appears only in the deciding seat's view.
//   - Every exported field has an explicit json tag. omitempty is used only on
//     pointer fields, which tygo maps to optional TypeScript properties; a
//     pointer without omitempty would marshal as null and break that mapping.
//   - Slices are initialised to empty, never nil, so JSON never carries null
//     arrays.
//   - Identifiers on the wire are kebab-case slugs. The server sends ids,
//     codes and structured values, never display strings; the client
//     localises everything.
//   - 64-bit integers that JavaScript cannot represent exactly (the seed) are
//     sent as decimal strings and tagged tstype:"string".
package view
