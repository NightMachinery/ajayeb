# Architecture

## The one big idea

**The server owns the truth.** The rules engine exists once, in Go, and runs only on the server. After every accepted action, the server sends each connection a full snapshot of the game as that connection is allowed to see it. The browser is a pure function of the last snapshot it received: it draws the board, offers the moves listed in the snapshot, and sends back the key of the move the player picked.

Three consequences follow, and most of the design is built on them:

- **One engine.** There is no second copy of the rules in TypeScript that could drift from the Go one.
- **Hidden information stays on the server.** A face-down card's identity never reaches a browser, so no client can cheat by reading memory or network frames.
- **Every mode is the same machine.** A game against a bot, a hot-seat game and an online game are all rooms on the server. They differ only in who controls which seat.

## System diagram

```
  Browser tab (SvelteKit SPA, no rules code)          Go binary "ajayeb" (one process)
 +-----------------------------------------+         +------------------------------------------------+
 | components: pure functions of           |         | httpapi: /health, /api/rooms/:code, the SPA,   |
 |   presenter.view + local ui state       |         |          /api/dev/* (only with --dev-endpoints)|
 |        ^                                |         |                                                |
 | Presenter: paced view, animation queue  |         | ws: upgrade, origin check, read limit,         |
 |        ^                                |         |     rate limit, latest-wins outbox per socket  |
 | Session: latest authoritative view,     |  JSON   |        | client messages      ^ state snapshots |
 |   seq, pending action, connection state |<------->|        v                      |                 |
 |        ^                                |  over   | room actor: one goroutine per room             |
 | Transport: WebSocket (or fixtures       |  /ws    |   seats, mode, seq dedup, undo, bot turns      |
 |   for screenshots and tests)            |         |     |              |               |           |
 +-----------------------------------------+         |     v              v               v           |
                                                     |  engine         ai              store          |
                                                     |  pure rules,    Random, Greedy, rooms/<code>   |
                                                     |  Legal, Apply,  ISMCTS on an    .json:         |
                                                     |  ViewFor        Observation     seed + keys    |
                                                     |                                                |
                                                     |  ident: browser token -> HMAC user id          |
                                                     +------------------------------------------------+
```

In production, Caddy terminates TLS and reverse-proxies everything to the one Go process, which also serves the built web app from an embedded copy.

## Server layering

The Go module lives in `server/` (`github.com/NightMachinery/ajayeb/server`). Dependencies point inward: the engine knows nothing about rooms, sockets, files or clocks; the room knows nothing about sockets; only `ws` and `httpapi` touch the network.

- **`internal/engine/catalog`**: the typed card, wonder and token data and the small effect language used to describe them. Numeric ids follow a public reference numbering (cards 1 to 73, wonders 1 to 12, tokens 1 to 10) and travel on the wire as kebab-case slugs.
- **`internal/engine`**: the rules. Setup (`NewGame`), legality, `Apply`, payment and its breakdown, military, science, wonders and the pending choices they open, ages, scoring, the view builder (`ViewFor`) and the per-viewer event filter, the bot observation and determinization, replay and hashing, and the scenario API used by tests. It imports only the standard library, `catalog` and the view types.
- **`internal/view`**: the view DTO types, with no logic. The builder lives in the engine because only the engine can read the hidden state; the types live apart so that `tygo` can read them without pulling in the rules.
- **`internal/ai`**: the bots. A bot receives an `Observation`, never the real state (see below).
- **`internal/room`**: one actor goroutine per room with an inbox channel, so room logic needs no locks. It owns the seats and the mode, deduplicates client sequence numbers, checks versions, schedules bot turns, implements undo, and writes the room file after every accepted action.
- **`internal/store`**: one JSON file per room, written atomically (temporary file, sync, rename), plus a reaper for old rooms with an injectable clock.
- **`internal/ident`**: the server secret and the HMAC-derived user ids.
- **`internal/wire`**: the client and server message types.
- **`internal/ws`**: the WebSocket transport (`github.com/coder/websocket`): origin patterns, read limits, per-connection rate limits, pings, and an outbox that keeps only the newest snapshot for a slow client.
- **`internal/httpapi`**: `/health`, the room lookup API, the development endpoints (refused unless the server runs with `--dev`), and the static handler with its SPA fallback.
- **`internal/webdist`**: embeds the web build only under the `embedweb` build tag, so a fresh clone builds and tests without building the web app first.
- **`cmd/ajayeb`** wires everything together. The other commands are `ajayeb-sim` (tournaments, the AI benchmark, level calibration, weight sweeps and the replay verifier; see [ai.md](ai.md)), `ajayeb-drive` (plays full games over WebSocket against a running server) and `ajayeb-gen`, the TypeScript generator. `ajayeb-gen` today is only `ajayeb-gen catalog -format csv|json|prose`, the catalog dump described in [docs/rules/sources.md](rules/sources.md); the generated-TypeScript subcommands in the next section arrive with their own slices.

## Actions and views

**Actions are canonical key strings,** so logs, replays and test scripts are readable:

- `draft:<wonder>` takes a wonder during the draft.
- `construct:<card>` builds a card into the city.
- `discard:<card>` discards a card for coins.
- `wonder:<wonder>:<card>` builds a wonder, using the card.
- `destroy:<card>` and `revive:<card>` resolve the destroy and build-from-discard choices.
- `token:<token>` picks a progress token (from the board, or from the Great Library offer).
- `starter:<seat>` chooses who starts the next age.

The engine produces each key from its own action value and parses keys strictly. The server accepts a key only if it appears in the legal list for the current state, and then applies the engine's own action for that key. Payment is never part of an action: the engine always applies the cheapest payment and reports its breakdown.

**The view is built by whitelist.** `ViewFor(state, viewer)` copies into a fresh DTO only the fields that viewer may see, so a field nobody remembered to redact cannot leak. Besides the board, cities and military track, the view carries what the thin client needs to explain the game without knowing the rules:

- `choices`: the legal keys for the decider, each with its kind, its cost breakdown, and disabled options with a reason, so the UI can say why a move is not available;
- both players' cost breakdowns for every accessible card, discard values, wonder costs and trade prices;
- the score preview, the pending choice, `events` (structured, for animations and the live region), the log, `canUndo`, who is thinking, and presence.

## Wire types and generation

The TypeScript types in the client are generated from the Go structs, never written twice:

1. **Types.** `tygo` turns `internal/view` into `web/src/lib/wire/view.gen.ts` and `internal/wire` into `web/src/lib/wire/types.gen.ts`. tygo writes one file per package, and `types.gen.ts` imports `view.gen.ts` as `view`; every view type that a wire struct references needs a `type_mappings` entry in `server/tygo.yaml`, or it comes out as `never`. A hand-written `index.ts` next to them re-exports both and defines the message unions.
2. **Data.** `ajayeb-gen` will write `ids.gen.ts` (the id lists as `as const` arrays, so a translation table missing a card fails to compile) and `catalog.gen.ts` (display data such as colour, cost and chain symbols, so the component gallery can draw cards with no server). This subcommand does not exist yet; today `ajayeb-gen` only dumps the catalog for review (see [docs/rules/sources.md](rules/sources.md)).
3. **Contract fixtures.** A Go test will play a scripted game and record the real outgoing messages, and `ajayeb-gen` will turn them into `fixtures.gen.ts`, typed with `satisfies` against the message union, so `svelte-check` fails if real server JSON does not fit the TypeScript types (a stray `null`, a missing field, a wrong enum string). This also arrives with a later slice.
4. **Hygiene.** A Go reflection test walks every outgoing type: each field has an explicit `json` tag, `omitempty` appears only on pointers, and recorded messages contain no `null` arrays.

`make gen-check` will regenerate everything and fail if regenerating changes the files in the working tree (on a clean checkout such as CI, that means the committed files). It joins `make verify-quick` once the `ids.gen.ts`/`catalog.gen.ts`/`fixtures.gen.ts` subcommands of `ajayeb-gen` exist; until then `verify-quick` skips it.

## The client: Transport, Session, Presenter

```
WebSocket -> Transport -> Session (authoritative) -> Presenter (paced) -> components
```

- **Transport** delivers server messages and sends client messages. The WebSocket transport reconnects with backoff and jitter. A fixture transport replays recorded views, so screenshots and component tests need neither the Go server nor a dev server.
- **Session** holds the latest authoritative state: the room, the view (in `$state.raw`, because it is large and always replaced whole), the connection status, the next sequence number and the pending action. It works out `canAct` from the view and the seats this browser controls.
- **Presenter** queues incoming snapshots and decides when the UI sees each one. Before swapping in a new view it records where things were on screen, then plays the animations described by the snapshot's `events` (FLIP and the Web Animations API). It holds an AI move on screen for a minimum "thinking" time, fast-forwards when updates pile up (for example after a reconnect), and applies updates at once when reduced motion is on. Moves are enabled only when the presenter is idle and the session can act.
- **Components** are pure functions of `presenter.view` plus local UI state (selection, open sheet, armed confirmation). Every card element has `data-card="<id>"` and every actionable control has `data-act="<key>"`; the end-to-end tests drive the game through these attributes.

**The game screen** (`web/src/lib/components/game/GameScreen.svelte`) is the one component every room page mounts. It keeps one markup and switches CSS grid areas by viewport class: a phone in portrait stacks the top bar, the opponent's strip, the track and tokens, the structure, the player's own strip and the turn bar; a phone in landscape puts the strips in side columns and the compact track in the top bar; a tablet stacks as on a phone with city bands; a desktop has city panels in 300 px columns (380 px from 1600 px wide). The board region is `dir="ltr"`, and "you" sit at the bottom or on the left. Its local UI state (open sheet, selected slot, the wonder chosen for a wonder-first build, the hot-seat device holder, the session's event log) lives in `GameUi` (`web/src/lib/ui/ui.svelte.ts`). Every move it sends is a key from `choices`: the card sheet shows a card's build, discard and wonder choices with the server's cost breakdowns, and a disabled choice shows the server's reason.

 `window.__ajayeb` (the last message, the version, the current choices, the controlled seats, the mode). It contains only what the client already received, which is safe because the client never holds hidden information.

## Data flow of one move

An online game, seat 0 builds the Baths:

1. The player taps the Baths. The client opens the card sheet and shows the cost breakdown from the `construct:baths` entry in `choices`.
2. The player taps Build (online games ask for a second tap to confirm). The Session sends `{"t":"act","code":"K7QX2M","seq":17,"base":42,"key":"construct:baths"}` and marks the action pending. The card gets a "lifting" outline and every move control is disabled.
3. The `ws` read loop checks the frame size and the rate limit, decodes the message and passes it to the room's inbox.
4. The room goroutine checks, in order:
   - the connection has joined the room as a player, not as a spectator;
   - `seq` is newer than the last `seq` applied for this user and client id. An older or equal one is answered with the current state and an `ack` for that `seq`, and nothing is applied. This check comes before the turn check, so an act re-sent after a reconnect resolves even when the turn has passed;
   - the game is running (a finished game answers `reject illegal`, an online room still waiting for its second player `reject not-your-turn`) and the sender holds the seat that must decide (otherwise `reject not-your-turn`);
   - `base` equals the room's version (otherwise it replies `reject stale` followed by a fresh state);
   - the key is in the engine's legal list (otherwise `reject illegal`).

   Only an applied act uses up its `seq`. After a refusal the client may send the same `seq` again, and the room judges it afresh. The room stores its version with the action log, so a version never comes back after an undo, not even after the room was evicted from memory and reloaded.
5. The room applies the engine's own action for that key. `Apply` updates the state and returns the events of the move (coins paid, card built, cards revealed, and so on).
6. The room appends the key to its action log, bumps the version and writes the room file atomically.
7. For each connection, the room builds `ViewFor(state, viewer)` and filters the events for that viewer, then hands the snapshot to that connection's outbox. If an older snapshot is still waiting there, the new one replaces it.
8. The Session receives the `state` message, replaces its view, and clears the pending action, because the `ack` names its `seq` and the version has moved past its `base`.
9. The Presenter animates the events (the card flies to the city, coins move, newly uncovered cards flip) and then shows the new view. The TurnBar now says it is the opponent's turn.

**A bot turn** works the same way from step 4 on. When a bot must act, the room starts a search in its own goroutine and gives it an `Observation` of the bot's seat. The bot's seed is derived from the room seed and the game's ply, so the same position always gets the same answer. The bot's choice comes back to the inbox tagged with the generation of its search, a counter the room bumps for every search it starts; the room drops any choice that does not come from the search running for the current version, such as one an undo made stale or one cancelled when its human left and came back. A server-wide semaphore limits how many searches run at once. A bot searches only while its room has a connection: the last connection to leave cancels the search and the next one to join starts it again, so rooms left behind never keep the semaphore busy.

**Hot-seat** rooms belong to one browser that controls both seats; the room sends the view of whichever seat must decide. Hot-seat and bot rooms commit moves on the first tap and offer undo until the game is over. Online rooms have no undo.

## Determinism and replay

- **All chance happens at setup.** `NewGame(seed, options)` shuffles everything once: the wonder order for the draft, the three age decks and the cards removed from each, the guilds that enter Age III, the progress tokens on the board and the order of those left in the box, and the first player. After setup, `Apply` uses no randomness.
- **So a game is its seed plus its action keys.** The room file stores exactly that (plus seats and options), and the server restores a room by replaying it. Replay is fast because a game has roughly a hundred actions.
- **The state is a fixed-size value struct** of arrays, bitsets and small integers, with no pointers, maps or slices. Copying it is a clone, and hashing its bytes (FNV-1a) gives a stable fingerprint. A loaded room is checked against its stored hash; a room that fails to replay, or whose rules version is incompatible, is archived instead of silently changing.
- **Rules code avoids every source of nondeterminism:** no map iteration in logic, no floats, no clocks, no I/O, no shared mutable state. An import-guard test and the linter enforce this.
- **Undo** (bot and hot-seat rooms only, and never once the game is over) truncates the action log back to before the human's last decision and replays from the seed. Because hidden cards were fixed at setup, an undo cannot reroll them.
- **Golden replays** pin known games (seed, keys, winner, victory type, scores, final hash), and property sweeps replay thousands of seeded bot games, checking invariants after every action.
- **Bots are reproducible too.** A bot's random stream is seeded from the room seed and the game's ply (not the room version, which an undo moves on), and searches with an iteration budget give the same answer every time.

## Hidden information

Hidden information is decided entirely by the engine's view builder and event filter. The rules of the game hide a few things from **both** players:

- the **seed**, which would let anyone recompute the whole deal;
- the **face-down cards in the structure**: a face-down slot exposes only its back (`age1`, `age2`, `age3` or `guild`), never its card;
- the **cards removed from each age deck** at setup (only their count is shown) and the deals of later ages;
- the **guilds left in the box**, which never enter play;
- the **wonders left in the box** (the four never offered in the draft), and the second draft round's wonders until the first round is over;
- the **progress tokens in the box**: only their count, never their identity or order (players can still work out which tokens they are by elimination, as at a real table).

Everything else is public, as it is at the table: face-up cards, both cities, coins, the military track, the board tokens, the cards tucked under wonders, and the discard pile.

**The one asymmetric secret is the Great Library offer.** When a player builds the Great Library, three tokens are drawn from the box and offered to that player alone. While the choice is open, only the decider's view contains the offer, and the event that announces the draw carries the engine's `Private` flag, so the per-viewer event filter keeps it out of the other seat's event stream the same way the view keeps the offer itself out of their snapshot. The token chosen then joins that player's city like any other token. The two that were not chosen go back to the box and stay hidden from both players. (In a hot-seat game both people may see the offer on the shared screen; that is accepted.)

**At game over** the view reveals the removed cards and the box guilds, wonders and tokens, so players can review the game, and the result includes the seed and the full action log for a downloadable replay.

**Bots cannot cheat.** A bot receives an `Observation`, which wraps the real state in unexported fields: the hidden cards are replaced by "unknown", the seed is zeroed, and the library offer is present only if it is the bot's own. To search, the bot calls `Determinize`, which samples a complete state consistent with everything the bot has seen. A fairness test checks that two observations with the same public information but different hidden cards lead to the same choice.

**How this is checked:**

- a redaction sweep over thousands of seeded games asserts that no hidden card id, and never the seed, appears anywhere in either seat's serialized view, and that the library offer appears only in the decider's view;
- protocol tests capture every byte the server sends and check it against the room's true hidden ids at that moment;
- the online end-to-end test captures the WebSocket frames in the browser and compares them with a development-only endpoint that lists the room's hidden ids.

## Rooms, identity and reconnect

- **Identity.** The browser keeps a random token in `localStorage`. The server derives the user id as an HMAC of the token with a server secret, so tokens never appear in logs or URLs. An invite link is just `/play/<code>`.
- **Seats.** In an online room the first two distinct users take seats 0 and 1, and later visitors watch as spectators with a public view. Hot-seat and bot rooms accept only their creator. Leaving a room or disconnecting only detaches that connection: the seat stays with its user, who gets it back on the next join. Spectators can never lock a player out of a full room.
- **Rematch.** Once the game is over, either player may ask for a rematch. The server creates one new room with the same mode and seats and a fresh deal, stores its code with the old room, and sends every connection of the old room a `created` frame with that one code. Every later request, from either player and even after the old room was reloaded, gets the same code. Creating the rematch room counts against the requester's room creation limits, like a `create`.
- **Idempotency.** Every action carries a per-client `seq` and the `base` version it was made against, so a double tap or a re-send after a reconnect can never apply twice. The room keys seqs by user id and client id together, so another user who picks the same client id cannot make it ignore a player's requests.
- **Rooms in memory.** A room with no connection leaves memory after 30 minutes, and sooner when the server holds its maximum of rooms and needs the slot (the one idle longest goes first); it is on disk and comes back on the next join. A join that the room would refuse (another user's hot-seat or bot room) is refused from the room file's header, without loading the room.
- **Reconnect.** After a drop, the client sends `hello` and `join` again and receives a fresh snapshot. A pending action is resolved if the version moved past its base; otherwise the client re-sends it with the same `seq`.
- **Restarts.** On shutdown the server closes sockets with "service restart" and clients reconnect with backoff. Rooms are on disk, so a redeploy loses no game.

The full message contract will be in `protocol.md` (milestone M4).
