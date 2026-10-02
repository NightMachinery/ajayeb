# Protocol

This page is a stub. The full WebSocket message contract between the client
and the server (`hello`, `join`, `act`, `state`, `reject`, and so on) arrives
with milestone M4, once `internal/wire` exists. For now it documents two
pieces of the contract that already exist in the engine and that other
slices (the room, the store, `ajayeb-sim`, `ajayeb-drive`) need to agree on
today: the recorded game log and the scenario format. Both are plain JSON, so
a tool outside Go can read and write them without linking the engine.

## The game log

A game is fully described by its seed, its options, an optional starting
scenario and the keys of every action applied to it. The log format
`ajayeb-log/1` stores exactly that, nothing more, so a log is small and a
replay reproduces the game exactly:

    {
      "format": "ajayeb-log/1",
      "rules": 1,
      "seed": "1234567890123",
      "options": {
        "firstPlayer": "random",
        "centerPawn": "lastActiveChooses",
        "hideAtEnd": false
      },
      "scenario": null,
      "actions": [
        {"ply": 0, "seat": 0, "action": "draft:pyramids"},
        {"ply": 1, "seat": 1, "action": "draft:gardens"}
      ],
      "finalHash": "9f2c0e5a17b3d4c8"
    }

Field by field:

- `format` is always the string `ajayeb-log/1`. A reader checks it and
  refuses a log in a format it does not know, rather than guessing.
- `rules` is the engine's `RulesVersion`, an integer that goes up whenever a
  change to the rules or to setup could make an old log replay differently.
  A reader refuses a log whose `rules` does not match the engine it is
  running, instead of silently replaying it under different rules.
- `seed` is the 64-bit setup seed, written as a JSON string (not a number) so
  that a JavaScript reader keeps every bit of it; a double cannot hold a full
  64-bit integer exactly.
- `options` are the engine's `Options`: `firstPlayer` (how the first player
  of the game is chosen), `centerPawn` (the ruling used when the pawn starts
  an age centred) and `hideAtEnd` (whether removed cards, box guilds, box
  wonders and box tokens stay hidden once the game is over, instead of being
  revealed for review).
- `scenario` is `null` for an ordinary game. For a game that started from a
  named or ad hoc starting position, it is the Scenario object described
  below, and the log is replayed by building that scenario first instead of
  calling plain setup.
- `actions` is the list of applied actions in order. Each entry names the
  `ply` the game was at and the `seat` that was to move when the action was
  applied (a reader checks both before applying it, which catches a log that
  was edited or reordered by hand), and `action` is the action's canonical
  key, such as `construct:baths` or `wonder:pyramids:baths`.
- `finalHash` is optional: 16 lowercase hex digits of the FNV-1a hash the
  engine computes over the final state's fixed-size binary encoding. When
  present, a replay that reaches a different hash is reported as a mismatch
  rather than silently accepted, which catches a log doctored by hand or a
  rules change that was not meant to affect old games.

A store that persists a room, a tool that replays or verifies a game, and a
script that generates fixtures from a scripted game all read and write this
same shape, so none of them needs to agree with the others out of band.

## The scenario format

A Scenario describes a starting position that is not the plain start of the
game: a particular age, particular cities, particular tokens, and so on. It
is used to build fixed positions for tests, for the developer-only endpoint
that creates a room at a chosen position, and for the named scenarios the
engine ships for its own tests. Every field is optional, and the zero
Scenario is the ordinary start of Age I after a draft in which each player
took the wonders offered to it in order. Names of cards, wonders and tokens
are the kebab-case wire slugs, not numeric ids.

The fields:

- `name`: a label for a named scenario; not used by `NewGameFromScenario`
  itself.
- `firstPlayer`: overrides which seat started the draft and Age I, and which
  seat is to move unless `toMove` says otherwise.
- `wonders`: each seat's four wonders, in slot order. When it is left out,
  the draft is completed automatically, each pick taking the first wonder
  still on offer.
- `age`: which age's structure is laid out, 1 to 3 (0 means 1). Earlier
  ages' cards that the scenario does not place elsewhere go out of play.
- `players`: each seat's `coins` (default the starting amount), `city` (the
  cards it has built, as slugs), `tokens` (progress tokens it holds) and
  `built` (the wonders it has built, each with the wonder's slug and,
  optionally, the slug of the card tucked under it; when a built wonder's
  tucked card is left out, an unnamed card is used).
- `boardTokens`: when given, exactly the set of tokens on the board;
  otherwise the board keeps its dealt tokens minus any given to a player or
  placed in `boxTokenOrder`.
- `boxTokenOrder`: puts these tokens at the front of the box, in this order,
  so that a scripted Great Library draw is reproducible (it draws the first
  three). The rest of the box follows in its dealt order.
- `discard`: cards in the discard pile.
- `accessible`: cards of the current age moved into accessible, face-up
  slots of the structure.
- `onlyLeft`: empties the structure down to just these cards of the current
  age, placed in the first slots in layout order, for setting up the end of
  an age.
- `pawn`: the conflict pawn's position, -8 to 8 (positive toward seat 1's
  capital); any looting token the pawn has already reached is gone.
- `toMove`: the seat that is to move.
- `trigger` and `expect`: for a named scenario's own tests, a script of
  action keys to apply once the scenario is built, and a short description
  of what that script should lead to. `NewGameFromScenario` ignores both;
  they exist only for the engine's own test suite.

`NewGameFromScenario(seed, options, scenario)` builds the position by running
an ordinary `NewGame(seed, options)` first, so that everything the scenario
leaves unspecified is still random but reproducible from the seed, and then
rearranging exactly what the scenario names. The result always satisfies
`CheckInvariants`; a scenario that asks for an impossible position (a card
named twice, a wonder a seat was not dealt, and so on) is rejected with an
error instead of producing a broken state.
