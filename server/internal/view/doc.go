// Package view defines the data transfer objects (DTOs) for a seat's view of
// a game: the structure of cards, both players' cities and cost breakdowns,
// military, tokens, the discard pile, pending choices, the legal choices of the
// deciding seat, the score preview and the result. It also defines the
// per-viewer game events (GameEvent) that accompany each view.
//
// The engine fills these types (engine.ViewFor and engine.FilterEvents); this
// package only declares them. It is also an input to tygo, which turns it
// into TypeScript for the web client (see tygo.yaml).
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
//     A reflection test checks every type of the package.
//   - Slices are initialised to empty, never nil, so JSON never carries null
//     arrays. Maps are not used: a struct such as ResourceCounts gives the
//     client a fixed shape instead.
//   - A pair indexed by seat (both players, both seats' costs, both scores)
//     is a Go array of two, tagged tstype:"Pair<T>" (declared in schema.go)
//     so that TypeScript sees a tuple of exactly two entries.
//   - Numbers are plain int: JSON and JavaScript share them, and a
//     fixed-width integer type would only invite a byte slice, which
//     encoding/json writes as base64.
//   - Identifiers on the wire are kebab-case slugs. The server sends ids,
//     codes and structured values, never display strings; the client
//     localises everything. Small vocabularies (phases, colours, resources
//     and so on) are string types with one constant per value, which tygo
//     turns into TypeScript unions.
//   - 64-bit integers that JavaScript cannot represent exactly (the seed) are
//     sent as decimal strings and tagged tstype:"string". No view carries
//     the seed, not even at game over.
//
// # What a viewer sees
//
// A viewer is seat 0, seat 1 or a spectator. Everything public at a real
// table is in every view: face-up cards, both cities and purses, the cards
// tucked under wonders, the military track and its looting tokens, the board
// tokens, the discard pile, the wonders of the current draft round, the cost
// of every accessible card for both seats, each seat's wonder costs, trade
// prices, production, science and discard value, and the score preview.
//
// Hidden from every viewer until the game is over: the cards in face-down
// slots (only their backs show), the cards set aside from each age's deck
// (only their number), the deals of later ages, the guilds left in the box,
// the four wonders never offered, the second draft round until the first is
// over, and the progress tokens in the box (only their number). At game over
// they appear under revealed, unless the game's options hide them.
//
// One secret is asymmetric: the tokens the Great Library draws. While that
// choice is open only the deciding seat's view lists them (pending.tokens and
// the choices); every other view shows only how many were drawn
// (pending.hiddenTokens), and the libraryDraw events reach the others without
// the token.
//
// choices and disabled are filled only in the view of the seat to move, so a
// spectator and the waiting seat get empty lists.
package view
