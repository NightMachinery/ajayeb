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
 |   presenter.view + local ui state       |         |          /api/dev/* (--dev + --dev-endpoints) |
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
- **`internal/engine`**: the rules. Setup (`NewGame`), legality, `Apply`, payment and its breakdown, military, science, wonders and the pending choices they open, ages, scoring, the view builder (`ViewFor`, and `ViewWithHints` with the hints of `hints.go`) and the per-viewer event filter, the bot observation and determinization, replay and hashing, and the scenario API used by tests. It imports only the standard library, `catalog`, the view types and shared deterministic helpers in `enginekit`.
- **`internal/enginekit`**: shared deterministic infrastructure. `rng` owns the value PCG generator, bounded draws, Fisher-Yates shuffle, the policy seed mixer and FNV-1a byte digest with fixed-width hex formatting. Duel keeps its RNG names through aliases and wrappers and owns its state/catalog encoding and hashed prefix. `puritycheck` and `catalogdiff` are test-only helpers for source/shape guards and three-way catalog verification.
- **`internal/view`**: the view DTO types, with no logic. The builder lives in the engine because only the engine can read the hidden state; the types live apart so that `tygo` can read them without pulling in the rules.
- **`internal/ai`**: the bots. A bot receives an `Observation`, never the real state (see below). Its `Pool` bounds the searches running at once and shares them fairly between client addresses, so one address's many bot rooms cannot hold up another's bot.
- **`internal/room`**: one actor goroutine per room with an inbox channel, so room logic needs no locks. It owns the seats and the mode, deduplicates client sequence numbers, checks versions, schedules bot turns, implements undo and the forfeit (`resign:<seat>`, a terminal action never offered in `choices`), keeps each applied action's events for the history (rebuilt by replay on load), and writes the room file after every accepted action.
- **`internal/tableroom`**: Haftgane's table room kind, a separate actor beside Duel's (see "The table actor" below). Its concrete engine seam binds Haftgane rules and whitelisted views.
- **`internal/lobbyroom`**: the shared lobby room kind (D60), where a group gathers before a game; its Start hands two filled seats to Duel, or a larger table to `tableroom`, under the same code (see "The shared lobby" below).
- **`internal/roles`**: the admin roles of D42, shared by the lobby and the table: real admins and who promoted them, demotion and stepping down, and the temporary admin of the five-minute rule, all persisted, with the clock passed in.
- **`internal/store`**: one JSON file per room, written atomically (temporary file, sync, rename), plus a reaper for old rooms with an injectable clock. It refuses to read a room file larger than a cap derived from what a room can hold.
- **`internal/ident`**: the server secret and the HMAC-derived user ids. The secret file must be a regular file, and one other users could read is tightened to mode 0600.
- **`internal/wire`**: the client and server message types.
- **`internal/ws`**: the WebSocket transport (`github.com/coder/websocket`): origin patterns, read limits, per-connection rate limits, pings, and an outbox that keeps only the newest snapshot for a slow client.
- **`internal/httpapi`**: `/health`, the room lookup API, the development endpoints (enabled only with both `--dev` and `--dev-endpoints`), and the static handler with its SPA fallback. Every response carries the security headers (no framing, same-origin opener and resource policies, a Permissions-Policy that turns off unused features) and a content security policy that allows only same-origin scripts plus the SHA-256 hash of each inline script of the HTML file served, computed from the served bytes (under `--dev`, CSP is report-only and same-origin pages may frame the app); see [protocol.md](protocol.md#http-endpoints).
- **`internal/webdist`**: embeds the web build only under the `embedweb` build tag, so a fresh clone builds and tests without building the web app first. It also defines the build's cache policy (`CacheControl`, which the static handler in `internal/httpapi` applies): SvelteKit's content-hashed `_app/immutable/` files are cached for a year, and so is bitmap art under `art/` requested with a `v` query parameter that matches the start of the file's SHA-256 (the generated art index writes every art path that way); art without a matching hash is cached for a day and then revalidated in the background for up to a week; everything else, including the app shell and the stable app icon PNGs under `icons/`, is revalidated on every load.
- **`cmd/ajayeb`** wires everything together. The other commands are `ajayeb-sim` (tournaments, the AI benchmark, level calibration, weight sweeps and the replay verifier; see [ai.md](ai.md)), `ajayeb-drive` (plays full games over WebSocket against a running server) and `ajayeb-gen`, the TypeScript generator. `ajayeb-gen` writes the presentation catalog and contract fixtures described below, and exports the catalog as CSV, JSON, prose or TypeScript for review (see [rules/sources.md](rules/sources.md)).

### Room-kind composition

`cmd/ajayeb` builds one closed `room.Kinds` table: `DuelKind`, Haftgane's
table kind (`tableroom.Tables.Kind`) and the shared lobby
(`lobbyroom.Lobbies.Kind`). Each stable game id binds create, cold-join admission and load
callbacks to its file codec. `NewKinds` rejects duplicate or incomplete
registrations and mismatched codec ids before startup. Its `Codecs()` table is
passed to the store, reaper and read-only preflight; there is no second list of
games in those paths. Schema-1 Duel remains required; schema 2's one
production registration is `haftgane`.

Manager stores opaque lifetime handles and retains its existing capacity
reservations, serialized restore, idle ordering, eviction retries and per-code
identity checks. The store's `Game` selector reads a capped, bounded header and
checks the schema/game pair against the bound codecs before Manager chooses a
loader. Cold admission still runs before a slot is reserved; already loaded
rooms bypass disk selection. Codes remain globally unique across live and
archived files and all room kinds.

Duel's adapter uses the existing `newRoom`, typed store operations, inbox,
attach/detach, delivery and eviction messages. Its actor, constructor, bot/review
scheduling, identity claims and room-file paths remain intact. `CreateHandle`
shares code allocation and capacity bookkeeping; `NewHandle` tracks a separate
actor's run/cancel lifecycle without requiring common rules, state or view types.
A stale handle cannot unregister its replacement. An eviction hook that agrees
to leave unregisters itself before replying; a refusal keeps the handle
registered. Initial Duel idle state is synchronized at registration, independent
of whether its handle was constructed before or after the first save. Existing embedders default to
Duel alone; production explicitly injects the table and store selector. No
protocol or client registry changes are part of this seam.

M14.protocol-game-frames adds negotiated hello versions 1/2, keeping Duel's
DTOs and actor unchanged. New table/lobby envelopes have strict separate
decoding, authenticated `Inbound` delivery and typed snapshot queues.
Join gates v1 before non-Duel admission or attachment. Common resync,
device-link and rematch operations route by the joined handle's actual kind.
At that slice production registered Duel alone; M16.haftgane-room-actor
adds the table kind below. See [protocol.md](protocol.md) for the frozen
families.

### The table actor

`internal/tableroom` is Haftgane's room kind (M16.haftgane-room-actor). Its
`Kind` is always registered, so saved tables load, play and archive; only
creating one is switched off. A lobby's Start makes a table under the lobby's
code (`StartTable`, below). The Go API `CreateTable` (a filled seat list,
optional seed, roles and prefix batches) and the development endpoint
`POST /api/dev/tables` make one directly. Creating a table directly, or a
lobby of three or more seats (creating one, growing a lobby to that size or
starting one), is refused unless the server runs with `-haftgane-tables`
(which needs `-dev` and `-dev-endpoints`), until Haftgane ships. A two-seat
lobby, which is how Home starts Duel's online game, and a shared Duel game's
rematch lobby need no switch, so production can make them. The lobby frame's
`maxSeats` (2 without the switch, 7 with it) tells the client which seat
counts to offer.

- **Engine contract.** `engine.go` binds concrete `haftgane/rules` and
  `haftgane/view` values. The actor accepts only engine-offered keys and
  preserves full payment witnesses, notebook memory and score facts. Rules
  version 1 is separate from development fake version 1000: fake records
  archive as `rules-version`, never migrate into a different game. The fake
  remains isolated test infrastructure and the import guard prohibits the
  production binary from reaching it. [The engine contract](haftgane/engine-contract.md)
  records setup, history numbering, bounds, replay and measurements.
- **Projection cache.** At most eight immutable engine projections, one per
  seat plus the spectator, are retained for the current engine fingerprint.
  Private draft edits, confirmations and presence updates reuse these engine
  facts. A resolution or undo replaces the cache; pending picks, acks,
  presence, roles and viewer metadata are always built for the publication.
- **Record.** One schema-2 file per table, `game: "haftgane"`, with its
  format, rules version, setup (seed, seat count, seating rule, side rule, end reveal),
  seat assignment, the Start-time slots, seats (each with the revision its
  human took it at and its device-link epoch), roles, the seat each former
  player held last, undo policy, the complete canonical batches, the rules hash, the
  two revision counters, the open point's reference, the pending block (each
  required seat's key, confirmed flag and per-client seq floors, bounded per
  user), the seq memory (bounded per user and kept only for users with
  authority, plus a bounded list of retired ones), frozen bot presets and the
  open point's work rows, bounded fallback diagnostics, the
  room log and the rematch's code. The format field names the record's
  semantics: a reader refuses a newer format and leaves the file untouched,
  and upgrades an older one explicitly instead of relying on unknown-field
  lenience. Restore
  validates all of it and replays the batches; a hash mismatch alone is
  refreshed in place, a rules-version mismatch or a failed replay archives the
  file (D29), and an invalid pending block counts as a failed replay. Unknown
  schemas and games stay untouched.
- **Transactions.** Every change clones the record, applies and validates the
  change, checks the candidate with restore's own checks (against the engine
  value it produced instead of a replay, and with the same per-user seq and
  floor bounds), writes the file durably, then swaps the new state in and
  publishes frames. A failed write publishes nothing and answers `storage`, so no frame
  ever shows a state, or a reveal, that is not on disk. The confirmation that
  completes a point's required set appends one sorted batch, resolves it and
  opens the next point in that same write.
- **Revisions and references.** The public revision advances on what every
  viewer sees (confirmation flips, resolutions, undo, seat switches,
  resignations, role changes, relinks); each seat's edit count advances on
  its other accepted picks, which reach only that seat's connections, and
  starts again at every public change. Frames carry the public revision
  times 2^20 plus the viewer's seat's edit count, so a switch never takes a
  connection's revision backwards. A point reference is the logical id and the public revision
  at its opening, so it is fresh after an advance or an undo and unchanged
  across a restart, and it never counts anyone's private edits.
- **Frames.** Each broadcast builds one frame per role (a seated viewer, or
  the one spectator role) through the engine's whitelist view and caches it
  for the current engine fingerprint; a connection's frame adds only its own pending pick and
  ack, and an admin's adds the previous-seat hint. Events and history carry
  public outcomes only, redacted for the viewer's seat; a connection's first
  frame after its seat changes carries its whole history again.
- **Roles, switches and resignation.** Package `roles` decides every role
  change and the temporary admin; a timer on the injected clock rechecks the
  five-minute rule. When the store refuses a role change, the actor keeps
  the unsaved roles, starts its next check from them and retries after five
  seconds, and the temporary role fails closed meanwhile: a temporary admin
  has the powers (and frames show them as one) only while both the saved
  roles and the rule, checked at that moment, name them. So a refused write
  delays a change but never leaves a demoted or replaced temporary admin
  their powers. An admin seats any spectator in any seat or gives a seat
  to a bot; the switch keeps the pending pick (a bot confirms it), rewrites
  the open point's work rows, stops or starts the bot, and logs a public
  entry. A resignation hands the sender's seat to a bot of the lowest level.
  Either revokes the device links of the seat's former user.
- **Fixed bot presets.** Table record format 2 freezes the released policy,
  preset, algorithm and deterministic bounds for every internal level. The
  actor calls the observation-only Haftgane package directly, one job per room,
  through existing pool permits. Cancellation waits for physical completion;
  late results must match the actor incarnation and task. [Room bots](haftgane/room-bots.md)
  describes level mapping, compatibility, watchdog diagnostics and tests.
- **Links and rematch.** A connection joining with a seat's claim acts as
  that seat's user. A claim's epoch is the public revision when the seat's
  human took it or relinked it, so a relink or a change of the seat's user
  revokes the claims before it, and their connections are detached. A
  seat started from a lobby slot keeps epoch 0, the rematch's claim, only if
  its human held that slot without a break; otherwise it starts at the
  table's first public revision. A finished table's original human asks for
  its one rematch in three writes: the table writes down a fresh code
  (`room.Manager.FreshCode`) with the requester, package `lobbyroom` makes
  the successor lobby under it through the `Successors` interface (the
  lobby names the table as its parent), and the table records it as its
  successor. A request after a refused write or a crash between them, by
  any original human, goes on from the written code, so there is never a
  second successor.
- **Undo.** Only the original sole human, while still controlling their seat,
  with undo allowed at creation; it is off for good once a second human takes
  a seat. It cuts back before the latest batch holding that seat's decision
  and reopens that point under a fresh reference.
- **Bots.** One bot job per table at a time, for the first required bot seat
  that has not confirmed. It searches under a permit from the server's shared
  `ai.Pool`, keyed by the smallest attached human participant's address, and
  confirms its pick through the same transaction as a human's. A durable work
  row per required bot seat freezes the policy version, level and allowance
  when the point opens, so a restart resumes the same work. No bot runs while
  no human participant is attached; the last human leaving cancels the job
  and releases its permit. A bot's random tape hashes a policy root derived
  from the seed under its own label, the game, the policy version, the
  logical point, the seat and its observation hash, never arrival order,
  references or pending keys. The seat's level defaults to the lowest
  released level.
- **Logs.** Log lines name the room, the operation, the seat, the public
  revision and the outcome, never a key, a card, an observation, the seed or
  a hash.
- **Game log.** A finished table with the final reveal on serves
  `ajayeb-log/2`: setup, complete batches and the rules hash, checked by a
  strict replay before it is served ([protocol.md](protocol.md#downloading-a-games-log)).

### The shared lobby

`internal/lobbyroom` is the `lobby` room kind (D60, M16.haftgane-lobby-and-roles):
a small actor and a schema-2 record (format, seat count, options, slots that
are open, reserved, human or bot, each with its claim epoch, roles, the users
who released or were removed, the seq memory, a role log and, for a rematch's
lobby, the table it follows), changed by the same clone, validate, write,
publish transaction as the table. The only claims a lobby honours are a
rematch's, of epoch 0, while their human keeps the reservation or the slot it
became; any other seating takes the revision of the change as its epoch, so a
human who lost their slot and is seated again cannot be reached by the old
claim, in the lobby or at the table. It knows no game rules. Start, by a
seated admin, builds the table's spec from the slots and roles and calls
`tableroom.Tables.StartTable`, which writes the table's record in place of the
lobby's (`store.ReplaceGame`, which requires the lobby's live record under that
code) and builds the table's actor with the lobby's connections. The lobby then
swaps its handle for the table's (`room.Manager.Swap`), so the code and every
connection move together. A detach that reaches the lobby during the swap is
forwarded to the table, and a detach always takes effect before a later join
of the same connection, so leaving and quickly joining again leaves it
attached; a join that arrives after it is retried on the
started game, and a lobby request is refused, so nothing is lost or applied
twice. At two seats Start instead calls `room.Manager.StartDuel`: it builds
Duel's ordinary schema-1 game with both seats assigned, carries their claim
epochs, conditionally replaces only a clean live lobby (`store.ReplaceDuel`),
and swaps in the Duel actor. Humans, their other tabs and spectators keep
the same code and assignments. A filled bot slot uses Duel's existing bot
factory while the online game retains its no-undo policy. An unsuccessful
write leaves the lobby intact; a late lobby request cannot change Duel.

Duel stores the lobby's effective admin set as optional `hintAdmins`, frozen
at Start. Those users, including promoted spectators, may edit hints, but
have no role or seat powers and may manage only their own device links.
Shared-game frames send `you.linkAdmin: false`; legacy frames omit it and
retain their existing creator-owned hint and device-link rights. Initial
Duel hint settings can travel in `LobbyOptions.hints` for D23's device defaults.
The engine, replay hash and old protocol fixtures are unchanged. A shared
Duel rematch also returns to a two-seat lobby with original humans reserved
and the requester its only initial admin; old one-tap rematches stay as they
were. As a table does with its reserved code and its successor, the parent
saves the code first as `rematchReserved`, then has the lobby made under it,
then keeps it as `rematch`. A retry or restart before that step finds the
same lobby; another room that took the reserved code is preserved and a new
code is reserved. Once the code is the `rematch`, every request answers it
without asking the lobby again, also after that lobby has started its own
game and holds no lobby any more. The code goes out, in `created` frames and
the answer, only after the file holds it as `rematch`: a failed write keeps
it in memory, answers `busy` and sends nothing, and the next request writes
it again before answering, so nobody can start a lobby the game would forget.

A table's rematch is a new lobby made here,
with the original humans' slots reserved for them: `Lobbies.Rematch` makes
it under the code the table reserved with the store's conditional create, so
checking the code and writing the lobby are one step: it finds the lobby
instead when an earlier attempt for the same table already made it (the
lobby names that table as its parent), and refuses a code any other room
holds, whose bytes stay as they were.

## App shell

The client ships with a service worker, `web/src/service-worker.ts`, served as `/service-worker.js`. It precaches the build of its version (the `_app/immutable/` files, the content-hashed fonts among them, the shell `index.html`, and the favicon, manifest and app icons), answers page loads network first with the cached shell as the offline fallback, caches `?v=`-hashed art at runtime in a bounded cache, and never touches `/ws`, `/api/*` or `/health`. Art-cache lookups and LRU refreshes share a transaction queue, so an offline request cannot mistake an entry being reordered for an uncached file. The app opens without a network and shows its own offline state; playing still needs the server. A new version waits while a game page is open. The server's headers above already fit it, so nothing on the server or a reverse proxy changes: the hashed files are immutable for a year, while the shell, `/service-worker.js`, the manifest, the favicon and the icons keep stable names and are revalidated on every load. [web/README.md](../web/README.md#app-shell) has the details.

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

**The view is built by whitelist.** `ViewFor(state, viewer)` (or `ViewWithHints(state, viewer, hints)`, the same view with the game's hints, see [Hints](#hints)) copies into a fresh DTO only the fields that viewer may see, so a field nobody remembered to redact cannot leak. Besides the board, cities and military track, the view carries what the thin client needs to explain the game without knowing the rules:

- `choices`: the legal keys for the decider, each with its kind and cost breakdown; `disabled` lists unavailable play actions with reasons, so the UI can explain them;
- both players' cost breakdowns for every face-up card (accessible or covered, as if it were accessible), discard values, wonder costs and trade prices;
- the score preview, the pending choice, `events` (structured, for animations and the live region), `canUndo`, who is thinking, and presence;
- on a join, a resync or an undo only, `history`: every applied action's ply, seat and events, filtered for the viewer, so a reloaded client can show the whole game's log. The client never animates it.

## Wire types and generation

The TypeScript types in the client are generated from the Go structs, never written twice:

1. **Types.** `tygo` turns `internal/view` into `web/src/lib/wire/view.gen.ts` and `internal/wire` into `web/src/lib/wire/types.gen.ts`. tygo writes one file per package, and `types.gen.ts` imports `view.gen.ts` as `view`; every view type that a wire struct references needs a `type_mappings` entry in `server/tygo.yaml`, or it comes out as `never`. A hand-written `index.ts` next to them re-exports both and defines the message unions.
2. **Data.** `ajayeb-gen` writes `catalog.gen.ts`: typed slug lists and display data such as colour, cost, effects and chain links. The slug unions make incomplete translation tables fail to compile, and the gallery can draw cards without a server. There is no separate `ids.gen.ts`. For Haftgane it also writes `haftgane-catalog.gen.ts` (the same file as `ajayeb-gen catalog -game haftgane -format ts`): the 80 design slugs and 7 board slugs, each design's Age, colour, printed cost, the chain symbols it takes in and gives out and its effects, and each board's starting resource and the stages of both sides. It holds codes and numbers only, never prose: names come from the translation tables. Only Haftgane's lazy chunk imports it (about 3.7 KB gzip), and the table screen reads it for printed facts, never for a price or a legality, which come from the view.
3. **Contract fixtures.** Go tests record wire messages under `internal/wire/testdata/messages/` and per-viewer protocol views under `internal/engine/testdata/protocol/`. `ajayeb-gen fixtures` turns them into `fixtures.gen.ts`, checked with `satisfies` against `ServerMessage[]` and `View[]`, so `svelte-check` fails if recorded server JSON does not fit the TypeScript types (a stray `null`, a missing field or a wrong enum string).
4. **Hygiene.** A Go reflection test walks every outgoing type: each field has an explicit `json` tag, `omitempty` appears only on pointers, and recorded messages contain no `null` arrays.

`make gen` runs `tygo`, then `ajayeb-gen` with no subcommand to write both the catalog and fixtures. `make gen-check`, included in `make verify-quick`, compares regenerated wire files with a snapshot of the working tree and fails on any difference. On a clean checkout such as CI, this checks the committed files.

## Authoritative client entry

`/play/<code>` keeps the existing name prompt ahead of all connection work.
A device claim or known saved room skips that prompt under the existing rules.
D61 keeps the ordinary v1 Duel Room and Session first, unless a saved game hint
or a room created in this page identifies another kind. Its first state is the
authority. A welcomed v1 join refused with `error protocol` before its first
state reports a wrong-game outcome and loads discovery; other protocol errors
and bad hellos remain fatal reload notices. The server refuses that join before
admitting a seat. Missing or stale hints therefore cannot admit the wrong game.

Discovery and its closed lazy registry load only on this fallback path. Its
v2 hello uses the same page identity and claim, holding one newest snapshot as
`unknown` until its module validates the server's frame family. A stale non-Duel
hint that discovers Duel keeps the bootstrap attached until the ordinary v1
Session receives its first state for that code, then closes without leave.
Discovery's snapshot/events never feed Duel's Presenter. Overlap prevents a
false away notification; ordinary Duel entry wakes the bot only on its real
Session and preserves all resulting events. Each tab retains its own client id.
Rematches and device-link fragment changes reuse Room and its existing socket;
`created` marks a rematch code as Duel before navigation.

`lib/games/haftgane` is the lazy entry for table and lobby frames
(M18.table-session). It holds no rules: it renders what the frames say and
sends keys from `view.you.choices`. The bootstrap hands it the already
welcomed connection with the one snapshot it held, then forwards every table
and lobby frame, their rejects, `linked` and `created`, and reports each new
socket's join; the entry asks for a resync once mounted, since the bootstrap
kept only that snapshot. `session/frames.ts` checks each frame's shape. A
frame that does not match fails the page with a reload notice, because only
a reload mends a client and a server that disagree.

- **TableSession** (`session/session.svelte.ts`) keeps the latest table or
  lobby frame in `$state.raw`, replaced whole, and one request in flight.
  Each page load is a client with its own id and a seq, counted from 1 and
  shared by all its requests; a refusal does not use one up. The counter
  lives beside the page's id (`entrySeqs` in `$lib/entry/identity`), not
  in the Session, so a Session mounted again in the same page (Home and
  back) goes on from the last seq instead of repeating ones the table
  already took; a reload makes a new id, which starts at 1. A pick is
  `{seq, point, key, confirm}`, sent only for a key among the latest
  frame's choices. Confirm and withdraw send the pending pick again with
  the confirmation flipped, so a pick can change until the point resolves.
  A frame whose `ack` names this client and seq settles the request; a
  `table-reject` ends it with its outcome (decision-closed, illegal, stale,
  forbidden, storage), except `rate`, which is retried after a pause up to
  three times. A request the seat itself makes (a pick, an undo, a
  resignation) names no seat, so the table applies it to whatever seat the
  sender holds when it arrives; each is bound to the seat it was chosen
  for. After a new socket's join or a `rate` refusal, it goes out again
  only while the latest frame still shows this tab in that seat (a pick,
  also at the same point), and a frame that moves the tab drops it, so a
  retry never resigns or picks for a seat the tab was given since. The
  server keeps each seat's
  pending pick and its confirmation and sends them back in that seat's
  frames, so a reconnect or a reload restores both. A frame that moves this
  tab off its seat (an admin gave the seat to a bot or another human) turns
  it into a spectator's view with a notice, and carries the history again,
  replacing the old view. A tab whose device link was revoked gets the
  `bad-claim` error and a reload notice. Rooms seen are remembered with the
  `haftgane` game hint, and a finished table leaves Home's Continue list.
- **TablePresenter** (`session/presenter.svelte.ts`) holds the frame on
  screen and moves it forward one queued frame at a time, as Duel's does.
  A frame with history, or one that changes this tab's seat, restates the
  game and is shown without playing events. A frame that only refreshes
  presence replaces the one on screen in place, and a backlog fast-forwards
  to the newest frame. A change without events (a seat confirming, a pick
  withdrawn) reaches a `still` hook after the swap, which nothing waits
  for.
- **Home and the lobby** (`lib/lobby`) keep Duel's one-tap bot, tutorial and
  pass-and-play starts. Play online opens the shared New table sheet with two
  seats; Haftgane's New table defaults to four. The sheet submits a name before
  its short v2 creation connection, then enters the ordinary room route. The
  lobby offers two through seven seats. At two it shows Duel's starter
  (random by default), bot levels and the chosen hint settings; at three or
  more it shows Haftgane's seating, final reveal and undo policy. The former
  online sheet is retired. Joining fills a seat but never starts a game. Its edits and role changes use the Session's existing sequenced
  retry lane; Start uses the server's `canStart` and explicitly fills open
  seats with Apprentice bots. Reservations block Start until their humans
  return or an admin opens them. Haftgane's wonder-side control offers Each
  picks, Day (the absent default) and Random, and replaces options by spreading
  the complete current object, preserving Duel preferences across count changes. The current protocol has no
  atomic seat-order edit, so seat numbers show the listed order without fake
  drag handles. A finished participant can request the rematch lobby; its
  private device claim stays in the successor URL fragment. Saved lobbies
  and live tables return through Home's Continue list with a Haftgane label.
  On a two-seat Start, the lazy bootstrap returns the Duel snapshot to
  discovery instead of passing it to the attached lobby Session. Discovery
  mounts Duel's ordinary Session and keeps the v2 membership until the v1
  join receives its first state, including when that socket already exists.
  Lobby-only English and Persian messages load with the lobby or its New
  table sheet, while Home loads only its picker labels. Native lobby selects
  set their fixed-option values through local attachments. This avoids extra
  shared Svelte runtime exports whose import aliases otherwise increase Duel's
  compressed chunks despite an unchanged eager module set. The development
  creation route `/dev/table` is retired. The legacy Duel waiting lobby
  loads `HintsSummary` only when its hint panel is shown. Its reuse in the
  shared lobby must not make a running hot-seat room fetch that panel.
- **The table screen** (`table/TableScreen.svelte`, M18.table-screen)
  draws [phone-layout.md](haftgane/phone-layout.md)'s main screen from the
  Presenter's frame: the top bar with the game menu, the Age and turn and
  the pass chip; the far chips; the two neighbour panels with their trade
  chips (units, price per unit and the discount's source, all from
  `view.you.neighbours`), the military outlook and the pass mark; the
  anchor's strip with its stage pills; the hand with each card's
  affordability badge; and the turn bar with the chosen count. The anchor
  is the viewer's seat, or seat 0 for a spectator, and the board is
  `dir="ltr"`. The layout is CSS only: a phone in portrait stacks the
  pieces, a short landscape screen moves the neighbours and far chips into
  side columns, and a desktop gives the neighbours full side panels. A
  tap on a hand card opens its [card sheet](haftgane/card-sheet.md), with
  its printed facts, build and stage payment witnesses, sale and action
  buttons carrying the server's keys. Payment radios save private drafts;
  the actions confirm through the Session with the existing confirmation
  preference. Change withdraws confirmation and reopens the card. Wonder
  sides and discard-pile decisions retain a separate choice panel. The game menu holds the rules and, until
  the admin slice, the development screen's seat, undo and resign
  controls. `table/model.ts` derives the seat ring, the badges and the
  printed facts; it groups the server's choices but never adds a cost.
  Each leaf takes plain props, and the dev gallery shows them all. Once
  the game is over, the entry's rematch actions follow the turn bar.
- **The city sheet and Seen hands** (`table/CitySheet.svelte`,
  `table/SeenHandsSheet.svelte`, M18.city-sheet-and-notebook) are
  [phone-layout.md](haftgane/phone-layout.md)'s section 8. A far chip, the
  crest and name of a neighbour panel or of the viewer's strip open the
  city sheet on that seat; ring tabs (`RingTabs`, the anchor first, then
  round the table from its left) switch seats inside it. `CityView` shows
  what the seat's public view holds: wonder and side, points, coins,
  shields, victory tokens by Age, defeats, stages, science, the score by
  category, the last play, each stage with its effect and built state
  (`CityStages`), production with its note, and the buildings by colour
  (`CityRows`). The cards under a wonder are known only as a set, never
  by stage, so the viewer's own city lists `you.tucked` after the stages,
  and other cities list theirs only once the end reveals them
  (`view.reveal.tucked`); the kept card is named only in the viewer's own
  city. The Seen hands sheet opens from the hand header and shows
  `you.notebook`: the statistics (`SeenHandsSummary`: circulating colours
  as exact, at most or a range, neighbours' certain picks and the cards
  that can come back), then tabs by Age listing each turn's hand with the
  played card marked and each pile look (`MiniCard`). A tapped figure
  filters the list. `table/city.ts` orders the tabs, groups a city's rows
  and the notebook, and reads the Age and turn from the point id
  (`age-N-turn-M`, `-discard-build`, `age-N-last-card`); nothing on these
  sheets is computed beyond counting and grouping what the view sent.
- **The end of an Age and the final scores** (`table/AgeMilitary.svelte`,
  `table/ScoreSheet.svelte`, M18.age-military-and-scores) are
  [phone-layout.md](haftgane/phone-layout.md)'s section 6, described in
  [end-of-age-and-scores.md](haftgane/end-of-age-and-scores.md). The
  military dialog opens by itself on the batch whose `military` events
  ended an Age (read from the session history, so a reload opens it
  again) and lists each conflict, its tokens, and the shields only on that
  batch. The score sheet follows at the end, with the verdict from
  `result.winners`, the rows in `result.rank` order, a row opening its
  city, and the log download when `canDownload` allows it. The final
  board keeps buttons that reopen both, and the revealed cards
  (`FinalReveal`) when the end is revealed. `table/end.ts` only reads and
  orders what the view sent.
- **The admin controls, resignation and the spectator view**
  (M18.admin-controls-resign-and-spectator) are phone-layout.md's section
  3. Each control appears only when the frame says this viewer may use it,
  and the server checks every request again. `table/admin.ts` and the
  roles section of `table/model.ts` read the frame: each human's role
  (`RoleTag`: a crown and "creator", "admin" or "temporary admin", the
  crown alone on a far chip), the spectators (the humans who hold no
  seat), the previous-seat hint (`you.previousSeats`, sent to admins
  only), and the role buttons, which follow the lobby's rule: a real admin
  may promote anyone who is not an admin, and may demote an admin they
  promoted, or, as the creator, any admin but the creator. An admin's game
  menu (`TableMenu`) lists every seat with Manage, which opens the seat's
  admin menu (`AdminSeatMenu`, connected by `SeatMenuLayer`) over it; the
  city sheet offers Manage too. A human's seat goes to a bot at any of the
  three levels every table freezes, and any seat takes a connected
  spectator, with the hint beside a former player; a real admin also gets
  the occupant's role buttons. The game menu lists the spectators
  (`SpectatorList`) with their role buttons, and offers "Stop being admin"
  to a real admin and "Resign to a bot" to a seated player. Every action
  shows what it changes first (`AdminConfirm`). Undo moves to the top bar
  while the table allows it, beside a count of the spectators that opens
  the game menu; when a second human sits down, the Session's `undo-off`
  notice says why it went. A spectator's hand region is the
  `SpectatorPanel`: seat buttons that centre the ring on the seat followed
  (`ring(seats, follow)`), the turn recap, and the spectators. A former player first reads who plays their
  seat now, and their ring centres on that seat until they follow another.
  The seat comes from an admin's own hint or from what this tab held
  (`TableSession.held`, kept across a reload in sessionStorage by
  `session/held-seats.ts`), since no other frame names it.
- **The reveal, the turn recap and the hand pass** (M18.reveal-and-recap)
  are phone-layout.md's section 4. `table/recap.ts` reads one resolved
  batch from the Session's `history`, as the server filtered it for this
  viewer: each seat's act (a building, a building from the discard pile, a
  stage, a sale, an empty discard build or a side), each coin change as
  one event states it (the printed cost and the trades from the seat's
  play, the receipts from its neighbours' plays, a sale's or a card's
  income from its `coins` event), a kept or discarded last card, where
  hands went, and the room log's seat switches and role changes dated to
  the turn (`at` from the previous turn up to this one). It adds nothing
  up, and a card the server left unnamed stays unnamed. `recap-text.ts`
  words it. The table screen hooks the Presenter: on the frame that
  resolves a turn it sets the reveal tags (`RevealTag`, hanging from each
  chip, panel and the strip) and the hand's arrival before the swap, then
  waits a short beat (none with Animation off) and opens the recap
  (`RecapLayer` around `TurnRecap`, with `CoinDelta` chips) for a seated
  viewer whose "Show opponent move" setting is on. The military dialog
  and the scores wait until the recap closes. The top bar's recap button
  reopens it, the spectator panel shows it inline, and a polite live
  region speaks it. A reload restates no reveal.
- **Sound and motion** (M18.sound-and-motion). `table/cues.ts` is the
  pure mapper from a live frame and the one before it to cues and a
  vibration pattern, as Duel's `fx/cues.ts` is. A frame with history,
  another room's or seat's, the first one, or one whose revision does not
  move forward plays nothing. Seven seats resolving at once make one
  grouped cue per kind: a flip for the reveal, a drum for the conflicts,
  the next Age's deal, the result. A seated viewer adds only their own
  moments: their play, coins they gain, the new hand, their confirmation,
  a decision that opens for them alone, the table waiting only on them
  while another person sits at it, and a kept last card. At most three
  cues sound, 90 ms apart. `table/sound.ts` plays the plan from the
  Presenter's play and still hooks with the app's player (`$lib/fx`), the
  same CC0 files, fallbacks, settings and test hook as Duel's, through
  `table/sound-player.ts`, a lazy chunk loaded once a table shows with
  Sound on; vibration calls `navigator.vibrate` itself. The game menu has
  the Sound switch. A new Age sweeps the pass chip and the pass flow
  once, a confirmed pick shows a check, and the chosen dots fill
  smoothly; with Animation off each is simply there.
- **The development screen** (`dev/TableDev.svelte`) now shows only what
  comes before a table's first frame.

**Chunks.** Haftgane's code must not change what Home and Duel download.
Its modules import from the root only what the root already loads (the
shell `Button`, the tooltip helper, the translation helpers, motion and
theme), and never a Duel component, Duel's icon module or a Lucide icon a
Duel or Home route uses: a module shared with Duel moves into a chunk
Duel loads, or makes an existing chunk export another name. The Duel
drawings the table reuses are imported raw under a `?hg&raw` query
(`table/glyphs.ts`), which gives them module ids of their own. The table
styles itself with scoped `<style>` blocks and the theme's custom
properties, not new Tailwind utilities, which would grow the stylesheet
every page loads, and `theme.css` (the seat and resource colours) ships
in Haftgane's chunk. It adds no Svelte feature that Duel and the
entry do not already use (such as transitions, a `<select>` value or
`bind:group`), whose runtime helpers would join the shared runtime chunk
and rename Duel's imports. The table reaches Duel's sound player only
through a lazy module of its own (`table/sound-player.ts`) that imports
names the player's chunk already exports, so that chunk keeps its
content. A module Haftgane shares with another lazy route becomes a chunk
of its own, whose name the app's route map and the play page's preload
list carry, so the gallery imports the table's leaves dynamically.
`make shots PW_ARGS=--grep=platform` measures Home and Duel cold bytes
against the committed baseline.

Loader functions stay separate in a frozen table: combining dynamic imports
in a conditional lets the bundler associate preloads with the wrong entry.

Saved Rooms preserves optional `game: "duel" | "haftgane"` hints, with
`mode: "online"` for Haftgane as an old-parser shim. Missing, stale or erased
hints always resolve from snapshots. An old tab's field-by-field rewrite keeps
the room and drops the unknown game field; the next current tab rediscovers it.
No hidden view data is stored. Card tooltip previews now load on demand; shared
gestures and translation lookup stay in the root. The separate optional English
message-table split has not changed the existing translation contract here.
The worker still precaches all build chunks so an open page keeps its lazy
chunks across deploys. Install transfer is measured separately from cold JS.

## The client: Transport, Session, Presenter

```
WebSocket -> Transport -> Session (authoritative) -> Presenter (paced) -> components
```

- **Transport** delivers server messages and sends client messages. The WebSocket transport reconnects with backoff and jitter. A fixture transport replays recorded views, so screenshots and component tests need neither the Go server nor a dev server.
- **Session** holds the latest authoritative state: the room, the view (in `$state.raw`, because it is large and always replaced whole), the connection status, the next sequence number and the pending action. It works out `canAct` from the view and the seats this browser controls.
- **Presenter** queues incoming snapshots and decides when the UI sees each one. Before swapping in a new view it records where things were on screen, then plays the animations described by the snapshot's `events` (FLIP and the Web Animations API). It paces a bot's answer: the bot's thinking snapshot stays up for a minimum dwell (about 700 ms, scaled by the animation speed, with a 300 ms floor when animation is off), then the answer is previewed for about 300 ms, so the screen can highlight the card the bot takes, before the new view shows. It fast-forwards when updates pile up (for example after a reconnect) and never paces a snapshot that has others queued behind it. Moves are enabled only when the presenter is idle and the session can act, so input stays locked through the dwell and the preview.
- **Components** are pure functions of `presenter.view` plus local UI state (selection, open sheet, armed confirmation). Every card element has `data-card="<id>"` and every actionable control has `data-act="<key>"`; the end-to-end tests drive the game through these attributes.

**The Duel game screen** (`web/src/lib/components/game/GameScreen.svelte`) is the component the Duel entry mounts. It keeps one markup and switches CSS grid areas by viewport class: a phone in portrait stacks the top bar, the opponent's strip, the track and tokens, the structure, the player's own strip and the turn bar; a phone in landscape puts the strips in side columns and the compact track in the top bar; a tablet stacks as on a phone with city bands; a desktop has city panels in 300 px columns (380 px from 1600 px wide). The board region is `dir="ltr"`, and "you" sit at the bottom or on the left. Its local UI state (open sheet, selected slot, the wonder chosen for a wonder-first build, the hot-seat device holder, the session's event log) lives in `GameUi` (`web/src/lib/ui/ui.svelte.ts`). Every move it sends is a key from `choices`: the card sheet shows a card's build, discard and wonder choices with the server's cost breakdowns, and a disabled choice shows the server's reason; for a covered card it shows both seats' costs from the slot's `costs`. Each player's wonder button marks a wonder as buildable only when a `wonder` choice names it. The military track puts the capital of the seat at the bottom (or on the left) at its left end, so a player at the Moon sees it mirrored. Against the bot, the turn bar and the bot's strip or panel say the bot (named by its level's persona: Apprentice, Scholar or Master) is thinking, and the slot it takes carries `data-bot-pick` during the preview. The top bar's game menu leads Home, Settings and the log in every mode, and Forfeit while the Session says a forfeit can be sent; after a confirm step it sends `resign:<seat>`; leaving through Home keeps an unfinished room on the Continue list, while a forfeit finishes it and removes it from that list.

**Languages and direction.** The server sends ids and codes, never display strings, so every word on screen comes from the client's message tables (`web/src/lib/i18n/`). English is the complete table; Persian (`fa`) is a stub table, and any key it leaves out falls back to English per key. The player chooses the language in Settings; it is stored with the other settings and applied before the first paint by an inline script in `web/src/app.html`. Persian is written right to left: `<html dir>` follows the locale, and the app chrome mirrors through logical CSS properties. The board region, the card structure and the military track stay `dir="ltr"` in every locale, because their layout is spatial, so a card keeps its slot and the pawn its direction. `web/README.md` ("Languages and direction") has the details.

**Online rooms in the client.** The same game screen serves online play, driven by the state frame's `room` block. While `room.status` is `lobby` it shows the lobby (code, invite link, copy and share, a QR code from a lazily loaded `uqr`) and switches to the draft when the friend's join arrives. The seats' `connected` flags drive the other player's presence dot, the join, leave and back toasts, and the turn bar's "away" wording; the client only reshapes them for display (`web/src/lib/ui/online.ts`). When this tab's own socket is down the board stays inspectable, every move button is disabled and reads "Waiting for connection", and nothing is sent until the reconnect's fresh state arrives. A spectator gets the same board with no actions and "Watching" in the turn bar. Online rooms have no undo and use the confirm step per the setting. A first visit through an invite link, with no stored name, asks for a display name before joining, because a join takes a seat under the name at once; rooms this browser created (remembered on `created`, before the page moves to them) never ask; changing the name later reconnects so the room shows it; opening a device link never asks for a name. After a game, the player who asks for the rematch moves to the new room; the other player and any spectators are offered to follow.

**Tooltips.** Every tooltip in the app goes through one helper, never the native `title` attribute. A component gives an element a tooltip with the `tooltip` attachment (`web/src/lib/ui/tooltip.svelte.ts`), and one layer mounted in the root layout (`components/shell/TooltipLayer.svelte`) listens to the pointer, touch, focus and keyboard events of the whole page and draws the one open tooltip. The innermost registered element under the pointer or finger wins, so a symbol on a card shows its own tooltip, not the card's. A fine pointer shows it after a hover delay and on keyboard focus; on touch, a tap shows it on an element with no action, and an actionable element keeps its tap and shows its tooltip on a long-press whose release never acts. Any tap while a tooltip is open closes it and is swallowed (`ui/dismiss.ts`, which the floating `DismissLayer` panel shares). In the opponent move summary, a panel that any tap closes, the tooltips inside follow the actionable rule: a long-press shows one and keeps the panel open, and the next tap closes only the tooltip. A tooltip can carry a stack: a second panel of linked previews joined to its bubble, such as the card a chain symbol makes free. Like everything else in the client, tooltips only reshape data the client already has (the catalog's printed text, or numbers the server sent).

Every build of the client exposes `window.__ajayeb` (the displayed view, the room and version, the current choices, the controlled seats, connection state, history and tutorial state). It contains only what the client already received, which is safe because the client never holds hidden information. A Haftgane page exposes `window.__ajayebTable` instead (`lib/games/haftgane/session/test-hook.ts`): the frame on screen and the latest one, the lobby frame, the connection, the request in flight, the notice and the history, all of it what the server sent this viewer.

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
7. The room computes the game's shared hints once, with the room's hint settings (`ComputeHints`). For each connection it builds `ViewWithHints(state, viewer, hints)` (at most three distinct views: seat 0, seat 1 and spectators) and filters the events for that viewer, then hands the snapshot to that connection's outbox. If an older snapshot is still waiting there, the new one replaces it.
8. The Session receives the `state` message, replaces its view, and clears the pending action, because the `ack` names its `seq` and the version has moved past its `base`.
9. The Presenter shows the new view in three steps, through its hooks (the animation runner, `web/src/lib/fx/anim/`):
   - **capture**: before the swap, the runner plans the frame's live events into a timeline (`plan.ts`, scaled by the game's Animation setting) and reads what it needs from the old DOM: the rectangle and a copy of the card being taken (its slot is gone after the swap), the face-down cards about to flip, where the pawn stood, the looting tokens the pawn will take, the token, pile card or draft tile a choice takes, and the structure an age replaces. With Animation off it plans nothing and reads nothing;
   - **swap**: the new view is rendered, at once and in full; the state on screen is always the server's;
   - **play**: the runner reads the new rectangles once, then animates from the old ones to the new (FLIP, transform and opacity only, with the Web Animations API): the card flies to the city, a discard into the coins, a wonder's card under its wonder; newly revealed cards flip; a coin float rises and the counter counts; the pawn walks cell by cell and flips the looting tokens it reaches; a progress token or a drafted wonder flies to its new row; a pair's science symbol glows; the Great Library deals its offer (face up to the drawer only, backs to everyone else); a new age's cards are dealt one by one; a victory banner ends the final board sequence before the result screen opens. On ordinary moves, input comes back as the frame's leading flight settles (240 ms into a card's 320 ms flight at normal speed, counted from the swap); the rest plays on, and a new frame finishes it at once. Age II and III deals drain their dealt-card runs before the starter choice opens. The final live frame drains every kind of run, including counters and landing pulses, before the Presenter becomes idle and the result dialog opens. These waits follow actual completion through the paint gate, rather than estimating it from a swap-based timer. The runner keeps the last animated move, so the log's Replay last move can play it again without touching the state.

   The TurnBar now says it is the opponent's turn.

**A bot turn** works the same way from step 4 on. When a bot must act, the room starts a search in its own goroutine and gives it an `Observation` of the bot's seat. The bot's seed is derived from the room seed and the game's ply, so the same position always gets the same answer. The bot's choice comes back to the inbox tagged with the generation of its search, a counter the room bumps for every search it starts; the room drops any choice that does not come from the search running for the current version, such as one an undo made stale or one cancelled when its human left and came back. A server-wide semaphore limits how many searches run at once. A bot searches only while its room has a connection: the last connection to leave cancels the search and the next one to join starts it again, so rooms left behind never keep the semaphore busy.

**Hot-seat** rooms belong to one browser that controls both seats; the room sends the view of whichever seat must decide. Hot-seat and bot rooms commit moves on the first tap and offer undo until the game is over, unless the game was created with undo off (`allowUndo` false in `create`), a setting fixed for the game's life, stored in the room file and shown to every viewer as `room.allowUndo`. Online rooms have no undo.

## Determinism and replay

- **All chance happens at setup.** `NewGame(seed, options)` shuffles everything once: the wonder order for the draft, the three age decks and the cards removed from each, the guilds that enter Age III, the progress tokens on the board and the order of those left in the box, and the first player. After setup, `Apply` uses no randomness.
- **Within a rules version, a game is its seed plus its action keys.** The room file stores exactly that (plus seats and options), and the server restores a room by replaying it. Replay is fast because a game has roughly a hundred actions.
- **The state is a fixed-size value struct** of arrays, bitsets and small integers, with no pointers, maps or slices. Copying it is a clone, and hashing its bytes (FNV-1a) gives a stable fingerprint. The hash covers every byte except the two derived counters at the end of the state (the shields each seat has placed, and whether the current turn is an extra turn): they follow from the action keys anyway, and leaving them out kept every hash recorded before they existed valid, in saved rooms, game logs and golden replays. Within the same explicit `RulesVersion`, a loaded room that replays completely through legal keys is canonical: a differing stored hash is refreshed atomically without changing the last-activity timestamp. A room that fails to replay, or whose rules version is incompatible, is archived. Downloaded replay logs keep strict hash validation (D28).
- **Rules code avoids every source of nondeterminism:** no map iteration in logic, no floats, no clocks, no I/O, no shared mutable state. An import-guard test and the linter enforce this.
- **Undo** (bot and hot-seat rooms not created with undo off, and never once the game is over) truncates the action log back to before the human's last decision and replays from the seed. Because hidden cards were fixed at setup, an undo cannot reroll them.
- **Golden replays** pin known games (seed, keys, winner, victory type, scores, final hash), and property sweeps replay thousands of seeded bot games, checking invariants after every action.
- **Bots are reproducible too.** A bot's random stream is seeded from the room seed and the game's ply (not the room version, which an undo moves on), and searches with an iteration budget give the same answer every time.

## Haftgane engine foundation

Haftgane's pure packages live under `server/internal/haftgane`, independently of
Duel. `catalog` defines 80 design identities, 77 name identities, seven boards,
resource/colour/science vocabularies and sixteen validated effect kinds.
Registration checks field use, saleable production, copy markers, chains and
twin semantics; tests swap in small synthetic catalogs with `ReplaceForTest`.
The existing Duel packages and data stay unchanged.

The catalog holds the base game's verified data (M15.haftgane-catalog-data,
from the M13 reference): the designs in `designs_age1.go`, `designs_age2.go`,
`designs_age3.go` and `guilds.go`, the boards in `boards.go` and the 24 chain
symbols in `chains.go`, written in the constructor language of `dsl.go` and
registered by package-level initializers, so a data error panics at
initialization. Design ids follow registration order: the Ages in turn with
the guilds last, then colour in the enum's order, then English name. Chain
symbols are numbered by the id of the one design each frees, so the Age II
targets come first. Name ids number the names by their first design, so each
Age I and Age II twin (Glassworks, Press, Loom) shares one, and `Twin` maps
each to the other. `T` holds the compiled lookup tables: sets by Age, colour
and name, each chain's target and providers, and each design's provided
symbols. `CopiesAt` counts a design's copies for N players. `RecordTables`
flattens the catalog into the notation of the M13 blind tables plus our ids;
`testdata/*.golden.csv` pins it byte for byte (the row order is part of the
rules version) and `testdata/catalog.golden.txt` pins the prose dump, both
written by `ajayeb-gen catalog -game haftgane`. The catalog's tests check the
edition's counts and shapes, diff it field by field against the two committed
blind tables through `enginekit/catalogdiff`, pin every value of
[edition-differences.md](haftgane/edition-differences.md), and list the
fields no photo has confirmed (Babylon's night side, accepted by the owner)
behind a release-gate test. [data-rulings.md](haftgane/data-rulings.md)
records the rulings and the fixtures' normalisations.

`rules` defines fixed-value State, Record, Game, actions, events and decision
points. It contains setup, the seat ring, canonical keys, turn legality,
resolution, replay, whitelisted views and immutable observations. Bot policies follow in the bot milestone. Left is the next seat modulo the seat count, and right is the previous.
Public plays and private picks have separate record arrays. Events have typed
source fields and no raw key.
Point ids are computed from kind, Age and turn; pending choices stay in the room.
Actions name design slugs and payment outcomes, never copies or seats. Parsing
checks canonical syntax and the tagged union; only membership in the current
offer makes an action legal. The rules version starts at 1.

`NewGame(Setup)` validates 3 to 7 seats and resolves seating, boards, sides,
guilds and each Age deck through separate fixed-tag `enginekit/rng` streams.
`SeatOrder` maps engine seats to lobby slots without moving another draw.
Deck inputs follow design id and copy-marker order, with the N + 2 drawn
guilds appended in id order to Age III; Fisher-Yates is followed by sorted
seven-card packets. Day and random sides deal Age I immediately. Choosing
sides leaves all hands empty until the simultaneous side point reveals every
choice together, then transfers Age I from `Deals` to `Hands` and
`Record.Packets`. Every seat starts with 3 coins. Five setup goldens pin stream
tags, seat order, boards, sides, packets, box guilds and fingerprints; the
invariant sweep covers 10,000 seeds per player count and all option kinds.

`ScenarioFromGame` makes a value snapshot whose explicit `State` and `Record`
fields tests can edit. `NewGameFromScenario` copies it, sorts card lists,
rebuilds city caches and rejects invalid conservation, shape or phase-specific
hand sizes. A sixth-turn discard point requires empty ordinary hands and a
kept card for a Babylon night seat whose last-card stage is built. It never
invents missing cards or a play history. Scenarios may
retain a partial history; the checker validates record shapes and placements,
while `Replay` checks full historical legality for recorded batches.
`CheckInvariants` also checks city names, board and stage bounds, inactive
slots, balanced victory/defeat tokens, real final-score consistency and cache
consistency. Known picks and current cards consume the multiset of their
originating packet, following each Age's pass direction (the kept card stays
with its sixth-turn packet). Point counts meet the phase minimum plus recorded
ability points, and equal that count when all resolved ordinary picks are
present. `Fingerprint` hashes an
explicit 2,157-byte little-endian encoding of every State and Record field in
declaration order, with one-byte booleans and no memory padding, through the
shared FNV-1a digest and hex helper.

`city.go` is the sole cache writer. Production symbols follow board start,
then ascending building design id and effect index, then ascending built
stage number and effect index. `ChoiceForSale` filters that same sequence to
saleable producers. This public-city order lets views identify the producer
behind a payment witness without using acquisition history. Rebuilding applies
continuous effects only, never income or scoring.

`LegalState` offers engine-owned actions from a frozen State into a reusable
choice buffer, ordered by design id, build/stage/sell, then payment total and
left amount. Each `Choice` embeds `Action` (including its canonical `Key()`)
and carries effective printed/trade totals, stage number, free reason and a
payment witness. The name rule is checked first. Chains and Olympia conditions
are checked per building before consulting the ordinary cost cache; a stage
never inherits a free result. A seat's local cache holds at most eight costs
and contains no free flags. It solves each ordinary cost with an unbounded
purse once, retaining the complete frontier before filtering by frozen treasure.
`AssessCostState` exposes `Coverable`, `Cheapest` (including printed coins) and
`Affordable`, preserving the difference between missing production and short
treasure. A view should say "cannot be covered" for `Coverable == false`,
or compute per-resource availability itself if it must name a resource,
because the shortfall may be shared producer capacity. The existing
`QuoteState` API still returns only affordable quotes.

`ResolveState` trusts actions selected by `LegalState` on that same State;
it never invokes legality generation or the payment solver. It removes picks,
computes additive payments from the frozen city, places every card, rebuilds
caches, and passes the remaining hands left in Ages I/III and right in Age II.
Sixth-turn leftovers enter the pile before placement, except Babylon night
keeps its other card when stage 1 already exists or is built on that turn.
After the discard opportunity and any kept-card turn, military compares each
adjacent pair once, then transfers the next Age or closes the game. Events
follow pipeline step and ascending seat even when the independent seat loops
run backwards. The core reads neither seed nor record and appends
to the caller's event buffer. `Game.Resolve` checks the exact point, sorted
required seats and offered keys, applies the engine's own actions, then writes
private picks (including Babylon's seventh row), public plays, discard rebuilds
and newly dealt packets. `Replay` uses those same checked batches; an error changes no caller value.

`applyWhenPlaced` interprets fixed coins, coins per card and coins per stage
once for the selected placement. It runs only after every city/cache reflects
step 4, so Vineyard and Bazar include same-turn neighbour buildings. Counts sum
the cities in the effect's scope; they never use a maximum. Each placement emits
one aggregated coin event in step 5, in ascending seat order, naming the public
building or stage. Stage income never names its tucked card. This income was
not available to pay for the current action, and is credited before military
and final scoring. Existing buildings' income is never replayed and the coin
reserve imposes no cap.

`scoreCity` computes the seven score-pad categories from the public cities:
wonder, treasure (whole groups of three coins), military (Age victory counts
worth 1/3/5 minus defeats), blue, yellow, green and purple. The point interpreter
handles fixed points, points per card, points per stage and completion of the
owner's wonder side. Points follow the holder's category, while science always
scores in green, including the Scientists Guild and Babylon's stage. Science
tries all distributions of choice symbols and maximises squares plus seven
per complete set. Equal optima prefer compass, then gear; the derived assignment
and final symbol counts appear in each public city view. An independent
oracle checks every ordered assignment for fixed counts 0 to 7 and up to seven
choices (real cities have at most two).

`finishScored` runs after Age III's military. It sums the categories, ranks by
total then actual coins, and gives tied seats the same competition rank, with
every rank-one seat in the winners mask. Final-score invariants recompute every
category from the public cities. Literal card/stage scenarios and an independent
raw-building score oracle check the results, with replay/core equivalence and
reversed seat-loop order in the seeded sweep. A separate coin ledger checks
every seat after ordinary, discard-build and kept-card points: frozen treasure
minus public payments, plus trade receipts, sales and literal card/stage income
counted from raw buildings. It also checks income events, independently of the
effect interpreter and production caches. The score oracle uses these checked
coins and the state's military tokens. Forced discard-income scenarios use the
same ledger at each boundary, including ruling 28.

Halikarnassos searches the whole pile after the turn's placements and coin
effects, including sixth-turn leftovers and cards just sold. A discard-build
point requires only its owner and opens only when a legal name exists. Every
offer constructs that design for free; there is no decline. The building's own
income counts the cities at that point without repeating earlier income.
An empty or all-duplicate pile emits `EvDiscardBuildNothing` and continues.
Discard points and rebuild records require the corresponding discard stage
play. Legal, resolution and invariants share the same pile/name predicate.
The stage play and its rebuilt/no-build outcome identify every pile look for
history reconstruction, including skipped searches. Private pile offers and
remembered looks appear only in the owner's view.

Babylon's kept-card point follows that discard opportunity and precedes
military. Only its owner acts, using a fresh frozen position: sixth-turn income
can be spent and newly built neighbour production can be bought. Construction,
stage building and selling are all offered under the ordinary rules. Resolving
one clears `Kept`; neighbours still receive its trade payments. No decline is
offered because selling always exists.

Olympia's free conditions read the frozen city for each building: day stage 2
frees only the first building of an absent colour; night stages free buildings
on turn 1 or turn 6 when the stage already stood at that turn's opening.
Selling or building a stage loses that turn's window. Free constructions have
one zero-payment offer, with chain precedence; stages never inherit it.
Decorators checks the actual side length, including two- and four-stage sides.

The rules now resolve complete games without a deferred ability path. Seeded
games check scores, conservation, record shape, replay and reversed seat order;
forced scenarios check all 41 stages and both skipped-pile outcomes immediately
at ability boundaries. The rules gate adds 60,000 complete games (2,000 seeds
for every player count, random/simple acceptance policy and side option),
1,183,834 resolved points, invariants and an independent literal coin ledger
after every point, independent final scoring and exact replay. All effect,
free-condition, event and point kinds occur; every one of the 41 stages is
also forced. The 75 compact golden replays retain readable keys and SHA-256
hashes of each point's full game value, raw events and offered choices. They
bind materialized scenario builders by an initial-state hash, rather than
copying large states into fixtures. The independent FULL review is the rules
audit; actor, persistence and client integration follow separately.

`Observe(seat)` builds a fixed, comparable whitelist: public cities without
production caches, this seat's sorted hand, tucked and kept cards, an offer
only at its own discard point, and the seat's historical `Knowledge`.
`Project(state, seat)` supplies the same current-position fields to sampled
search without reading a record. `o.Quote(seat, cost)` rebuilds public city
caches and uses the ordinary solver for any seat. The explicit field-order
little-endian observation encoding is 1,693 bytes; its FNV fingerprint remains
internal and never appears in a view, event, ordinary log or download.

Knowledge belongs to the seat. It contains its own held hands and picks,
everyone's public plays and rebuilds, and its own pile opportunities, including
an empty or all-duplicate skipped search. The notebook exposes these sightings
and conservative statistics for the current Age. A named public construction
removes that design from a remembered packet; a sale or stage removes one
unknown card. Per-colour and returning-design counts carry `certain` and
`atMost` bounds. A returned packet identifies a neighbour's private pick only
when every unexplained missing copy has the same design. A newer sighting
replaces the earlier packet, so a second circuit never counts it twice.
Statistics read the observation alone. Past deductions remain remembered when
no packet can circulate or return before the Age ends.

`ViewFor(seat)` produces `view.Table`, with a private `you` section only for a
valid seat. Public fields include each city's running score and optimal science
assignment, relative seat positions, latest public play and hand/kept counts.
Each neighbour panel includes fixed saleable units and separate shared choice
symbols, the viewer's unit prices and discount source, and the Age's conflict
token if military resolved now. Choices carry effective printed/trade totals,
free reasons and canonical producer witnesses. Disabled constructions distinguish
`already-built`, `wonder-complete`, `cannot-cover` and `insufficient-coins`;
the last includes the exact shortfall. Offers list every pile design and mark
which names are legal. Every DTO slice is non-nil, every field has an explicit
JSON name, and only pointer fields use `omitempty`.

`FilterEvents` selects fields by event kind, action and coin source, regardless
of the raw event's `Private` assertion; unknown kinds are dropped. `History`
replays materialized entered packets and public setup facts, rebuilding every
batch from the record without regenerating randomness from a seed. Its stream
always starts with `rules.HistoryStartPoint` (`"game-start"`) at index 0,
representing batch 0. That entry carries Age I's `age-start` for day/random
setup and an empty event list for chosen sides. Entry i is completed batch i,
so the platform uses the index as its batch number and a successful history
has `State.Points + 1` entries. It preserves live privacy after the game ends.
The same bounded walk builds Knowledge, latest
plays and view events using reusable scratch. Explicit scenarios with missing
past picks have no reconstructable history: `History` returns an error and an
empty list, while `Observe`/`ViewFor` keep current facts but omit historical
knowledge and events. They never invent a past.

With the default end policy, final views reveal all tucked cards, every
sixth-turn leftover (including a card kept and later played), the pile and the
box. `HideAtEnd` retains running-game privacy. `canDownload` exposes only the
end-policy permission; the platform owns log generation. There is no download
payload or seed field in these engine DTOs. The platform can embed the seven
raw JSON fixtures in `rules/testdata/views/`; `tygo` generates their types into
`web/src/lib/wire/haftgane.gen.ts`.

Rules-gate measurements on Apple M2 (three 500 ms benchmark runs,
2026-10-23) cover seven seats per operation at Age III turn 4. Seven `Clone`
calls take 440 to 491 ns with zero allocations; seven `Observe` calls take
585 to 619 microseconds, 24,528 bytes and 28 allocations; seven `ViewFor` calls
take 0.981 to 1.218 milliseconds, about 180,210 bytes and 923 allocations.
These are measurements, not timing gates. The acceptance sweep's
largest reachable JSON view is 25,789 bytes across 7,204 views. A deliberately
unreachable serialization stress combines 217 keys, seven witness units per
key, a full notebook and three full pile looks, and measures 185,536 bytes.
The earlier 24 KB design estimate is not a bound or a transport cap. The
rules gate publishes a separate conservative structural bound of 570,406 bytes:
every optional DTO section and list at its proven capacity, all safe-ASCII
strings at 64 bytes and all integers at their widest emitted width. Its test
covers partial-history scenarios as well as natural games, and rejects unknown
slice dimensions, unsafe strings and overflowing dimensions. This bound does
not assert that the mutually exclusive sections can occur together.

The ordinary payment solver uses frozen own production, neighbours' saleable
production, per-side prices and the starting purse. It returns at most fifteen
undominated `(left, right)` outcomes, ordered by total then left, each with a
canonical physical-symbol witness. Fixed own symbols are used first; a proven
maximum-own-coverage filter, bounded resource splits and separate choice-symbol
matchings keep the hot path allocation-free. `QuoteState` returns the cheapest
affordable ordinary total without building witnesses.
Chains and free-build abilities bypass this solver per action; stages use their
ordinary cost. An independent unit-by-unit oracle retains every affordable
covering before testing dominance. It checks 100,000 deterministic random inputs
and an opt-in small-universe sweep, run by
`make test-haftgane-payment-exhaustive` (part of `make verify-full`). The sweep runs need vectors in
parallel. The test oracle collapses permutations of unused identical-mask
producers within one city, retaining each physical capacity; a separate
comparison pins equality with its original uncollapsed traversal. This avoids
Go's ten-second fuzz-worker watchdog on dense duplicate inputs. `FuzzPayment` checks outcomes and every
witness against the same oracle.

Rules-gate payment measurements on the same host and run take 0.745 to
0.846 microseconds for a typical construction, 29.2 to 30.8 microseconds for
feasible capacity and 47.2 to 50.9 microseconds for impossible capacity, all
allocation-free. These meet the 50-microsecond typical and 1-millisecond worst
payment targets. `Quote` takes 0.658 to 0.824 microseconds typical, 3.40 to 3.60
microseconds for feasible capacity and 48.0 to 49.3 microseconds for impossible
capacity; `QuoteState` takes 0.460 to 0.583 microseconds. The impossible fixture
exceeds the design's 5-microsecond `Quote` target by about ten times. This
synthetic, unreachable stress gap is accepted after the rules audit. It needs
six saleable choice symbols that can each supply any resource, while the
catalog's only saleable choice producers are the six brown two-raw cards.
Their shared-capacity infeasibility is confined to raw resources and stays small.

The reviewer measured 1,456,185 inputs across 200 natural games at N=3 to 7,
including 503,052 unaffordable costs. The first-pass median was 0.45 microseconds
and p99 2.0 microseconds. Re-timing the 14,164 inputs over 2 microseconds gave
a median of 0.45, p99 2.4 and maximum 9.2 microseconds; a third pass reduced the
ten still over 5 microseconds to 2.2 to 5.25 microseconds. These slowest inputs
were all feasible and affordable, with 1 to 3 own choice producers. No natural
input approached the impossible fixture, which took 57 to 75 microseconds on
the review's loaded host. The solver is unchanged since its payment review;
these stress timing differences reflect host load, not a solver regression.

Revisit this acceptance if the payment solver occupies a significant share of
profiles in `M21.simultaneous-age-rollouts`. Then add a global feasibility
pre-check: one bipartite matching of units missing after maximal own coverage
against own choice symbols and both neighbours' saleable symbols, ignoring
price. Coverability is exact because a covering exists if and only if a
maximal-own covering does. Verify that change with the exhaustive payment
oracle, a 10-minute `FuzzPayment` run and `BenchmarkQuoteImpossible`.
No pre-check is added while current callers do not need it.

The seven-seat Legal benchmarks distinguish feasible capacity from impossible
shared capacity. In the rules-gate run, `LegalState` takes 28.4 to 30.7
microseconds typical, 1.071 to 1.448 milliseconds for feasible capacity and
2.395 to 2.524 milliseconds for impossible capacity, with zero allocations.
The corresponding `Game.Legal` ranges are 31.8 to 34.2 microseconds (9,856
bytes, 7 allocations), 1.028 to 1.086 milliseconds (66,304 bytes, 21 allocations)
and 2.586 to 3.442 milliseconds (9,856 bytes, 7 allocations). Both APIs meet
the seven-seat 2-millisecond typical and 10-millisecond worst targets.
`ResolveState` for all seven seats takes 2.67 to 2.80 microseconds with no
allocation, below its 0.2-millisecond target. The impossible fixture has seven
units per cost and only six shared flexible symbols, so every construction is
refuted and only 49 sales remain across all seats. These capacity cases are
synthetic stress fixtures, not claims of reachable cities.

Haftgane's baseline policies live in `internal/haftgane/bot`. Each receives
only its observation, exact offered choices, independent policy entropy and
a fixed per-bot work row. Public facts and isolated offered-choice previews
keep exact score, payment and placement logic in the engine. Forward-looking
features remain policy estimates. [The Haftgane AI page](haftgane/ai.md)
documents Random, Greedy, Apprentice and their exact Greedy fallback.

The shared `enginekit/puritycheck` and strict depguard allow-lists enforce
exact module dependencies for each pure package: the catalog can use only
`enginekit/rng`, and rules can also use their own catalog and view. Forbidigo
keeps both games' `ReplaceForTest` registry helpers in test files. Haftgane
cannot import either math/rand version, even in rng.go; the shared generator is
provided by `enginekit/rng`. The shape guard
checks fixed widths, comparability and independent State/Record/Game size
limits. Science's timing mask includes continuous cache construction and
end-game scoring. Wonder free-build effects apply to buildings, never stage
actions. The frozen single-writer files are `rules/state.go`, `rules/keys.go`
(the Action type), `rules/events.go`, `rules/limits.go` (sizing constants),
`rules/seats.go`, `rules/points.go` and `catalog/types.go`. Setup must initialize
an inactive decider to `NoSeat`; ability exits restore it. Stage numbers in
plays and events start at 1. Coin events cover only placement and sale income;
play events carry payments, and neighbours' plays determine trade receipts.

## Saved room compatibility

`internal/store` is the only reader/writer of room files. A bounded header selects the codec before game decoding or replay: schema 1 defaults to `duel` when `game` is absent and delegates to the existing Duel `File`, while schema 2 requires an explicit registered game id. The complete input retains the existing `MaxFileBytes` cap; the schema discriminator is at most 10 JSON bytes and the game id is at most 32 lowercase ASCII letters, digits or hyphens (starting with a letter), in at most 34 JSON bytes. Discriminator order is unrestricted; duplicate schema/game fields, including case aliases, fail closed. `NewCodecs` builds an immutable composition-time table, rejecting duplicate or invalid registrations. Its production schema-2 table holds Haftgane's table codec. Schema-2 records are read and written through game-agnostic `LoadGame`, `SaveGame` and `PeekGame` under the same per-code lock: a load runs the codec's assessment, rewriting only the hash for a refresh (keeping unknown fields) or archiving durably; a save refuses an archived file, a file of another game and a file of an unknown schema. A room's first record is written by a conditional create instead (`CreateGame`, and `Create` for Duel's schema 1, which writes the same bytes as `Save`): under the same lock it writes only when no file holds the code, live or archived, of any schema, readable or not, and otherwise writes nothing and answers `ErrExists` (`CreateGame` with the live bytes it found). Rooms of every kind share one namespace of codes and `NewCode` reserves nothing, so every creation path (Duel's rooms, lobbies, tables and a rematch's lobby) makes its first write this way, and `room.Manager.CreateHandle` draws another code on `ErrExists`. Codecs own their file types and game rules checks; shared inspection exposes only retention metadata and read-only replay assessment. `OpenWithCodecs` and `CheckDataDirWithCodecs` take the same table. Duel's load/save APIs retain their typed signatures and schema-1 write bytes.

Duel restoration selects ordinary keys from `LegalActions`, matching the room's own replay path; resignation uses the room's explicit legal, engine-generated resignation exception. A key parsed into an action is insufficient: it must match the canonical key of that generated action. Within the same `engine.RulesVersion`, seed, options and keys are authoritative. The hash is a diagnostic fingerprint, not an authenticated integrity check. A successful replay with a different hash refreshes only the hash and logs both values. The room manager serializes disk restores through actor registration, so a second restore cannot overwrite a move saved by the first actor. Requests to already-loaded rooms bypass that lock. The atomic rewrite preserves all other metadata, including additive fields unknown to this build, and does not refresh `Updated` or extend idle retention.

A rules-version mismatch archives with reason `rules-version`; an invalid setup/scenario or failed key replay uses `replay-failed`. The archive stores `archiveReason` and `archivedAt`, stamps `Updated`, and retains the action log and room metadata. It then moves the record from `rooms/` to `rooms-archive/`, keeping it out of live scans. Startup/hourly cleanup retains archives for 30 days after archiving, capped at 1000 files or 64 MiB, oldest first (D29). Legacy archives migrate without changing bytes or timestamps. Raw errors and keys stay in server logs. Repeated loads leave archive contents untouched. A legacy archive without a reason remains closed, with the safe client reason `unavailable`. Unknown schemas or games are refused without writing anything, and reaping preserves them outside the managed retention caps. Duel saves refuse a clean bounded header selecting a non-Duel game or schema-2 codec, and retain the existing refusal of a decoded Duel archive. Otherwise the next save repairs damaged live bytes from the room in memory, including unreadable or oversized files, malformed headers, invalid or truncated JSON, and an undecodable Duel body. A refresh or archive write failure is reported as temporary `busy`, never as a missing room, successful restoration or durable archive. The client keeps the Home entry for a later retry. Archived and genuinely missing rooms instead terminate the join intent, clear the stale board and pending/link/rematch work, forget the Home entry, and show the room error screen on both cold opens and reconnects.

Bump `engine.RulesVersion` for incompatible setup, RNG, catalog or rules changes, including changes under which old keys remain legal but describe a different game. A representation-only change or an added derived field does not by itself require a bump. Accepting legal replay over an old hash loses some corruption detection, so this version discipline and backups are essential. [D28](dev/decisions.md#d28-saved-rooms-are-canonical-within-an-explicit-rules-version) records the tradeoff.

The candidate server's `--check-rooms --data DIR` scans without creating a store, identity secret or listener. It scans both `rooms/` and `rooms-archive/` and reports existing archives, valid rooms, refreshable hashes, rooms that would archive and read/schema errors through the selected codecs. Unsupported schemas and games increment the existing `schemaErrors` and `errors` counters. The report remains `ajayeb-room-check/1`; its scalar `rules` continues to report Duel's version and does not select other games. The preflight script is unchanged. Archive-folder records never count as live rooms at risk. Deployment checks this before stopping the existing binary, refuses live-room loss without explicit force, and repeats the check at the stop boundary with recovery to the prior binary on refusal. See [self-hosting](self-hosting.md#room-compatibility-preflight).

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

**At game over** the view reveals the removed cards and the box guilds, wonders and tokens unless the room uses `hideAtEnd`. No WebSocket view carries the seed. A finished game whose cards are revealed permits a replay download from `GET /api/rooms/{code}/log`, for players and spectators alike. The JSON response contains the seed, options and action keys needed to replay it; running games return 409 and hidden records return 403. The menu shows a disabled entry with a custom tooltip during play, then enables it and adds a result-screen button when the record is available. The server names the file `ajayeb-<code>-YYYY-MM-DD.json` using the UTC download date. See [the download contract](protocol.md#downloading-a-games-log).

**Bots cannot cheat.** A bot receives an `Observation`, which wraps the real state in unexported fields: the hidden cards are replaced by "unknown", the seed is zeroed, and the library offer is present only if it is the bot's own. To search, the bot calls `Determinize`, which samples a complete state consistent with everything the bot has seen. A fairness test checks that two observations with the same public information but different hidden cards lead to the same choice.

**How this is checked:**

- a redaction sweep over thousands of seeded games asserts that no hidden card id, and never the seed, appears anywhere in either seat's serialized view, and that the library offer appears only in the decider's view;
- protocol tests capture every byte the server sends and check it against the room's true hidden ids at that moment;
- the online end-to-end test captures the WebSocket frames in the browser and compares them with a development-only endpoint that lists the room's hidden ids.

## Hints

The hints (decision D23; the design is the hints design of M10) are facts the server works out from public information and sends in the view, so the thin client never derives a rule to explain the game. They live in `internal/engine/hints.go`.

**Data flow.** For each broadcast the room's frame cache calls `ComputeHints(state, settings)` once, with the room's hint settings, and passes the result to `ViewWithHints` for each of the at most three views it builds. `ComputeHints` computes the shared hints, which are the same for every viewer; `ViewWithHints` adds the decider's hints about its own choices (below) to the view of the seat to move only. `ViewFor(state, viewer)` is `ViewWithHints(state, viewer, nil)`, so the simulator, the tests and replays never see hints, and a finished game's view carries none. Each part is computed only while its family is on, and the client also drops what its own player hides (`visibleFamilies`).

**Per-card hints (M10.HS2).** Every face-up slot of the structure, covered or not, carries `hints`: a pair of `CardHints`, one per seat, saying what building that card now would do for that seat. They are shared hints, so the three views carry the same values. A `CardHints` holds:

- `codes`: the pip codes, the facts strong enough to mark the card on the board. `win` (building it wins the game, from the threats family), `pair` (it completes a pair of science symbols while a progress token is still on the board, science) and `loot` (its shields take coins from the opponent, military). The client draws one pip per card, in the priority own win, opponent's win, own pair, opponent's pair, own loot, opponent's loot; a pip for the opponent is a deny target.
- `effect`, sent only when it says something: whether the seat can pay for it now, its science symbol and what it does to the seat's set, the pawn move with the coins each looting space would take, a win and how, and, for yellow cards and guilds only, the coins and points it is worth now.
- `priceShift` and `savesInView` (economy): how it would move a trade price, and what the seat's own production would save on the cards in view.
- `lastProducer` (counting): no other producer of its resource can still come (`lastCertain`), or none is in view and unseen cards may hold one (`lastVisible`).

Two pools get the same kind of hints. `hints.discard` lists, for each seat with an open Mausoleum (and for the seat choosing from the discard pile), the discarded cards whose hints say something, with their worth and whether a chain makes them free. A pending destroy choice carries `cardHints`: for each card that may be destroyed, how the trade prices would move, what the destroyer's payments in view would lose, and whether it is the opponent's only source of its resource. All of it comes from `previewBuild`, `priceShift`, `savesInView` and `publicPlace` on copies, so the leak guard below covers these fields with no extra code.

**Board-level hints (M10.HS4).** `hints_board.go` adds the shared facts about the board as a whole and about each seat, each under its family's switch:

- `players[i].hints` (`PlayerHintsView`): coins to the next point and each owned guild's points (scoring); the state of each of the seven science symbols (`paired`, `live` when one card is owned and the other may still come, `owned` when it never can, `open`, `dead`), the symbols still needed, whether enough may still come (`reachable`) and whether Law on the board would be the sixth (science); the shields the opponent still needs to take this seat's capital and whether it can no longer gather them (`toCapital`, `capitalSafe`, military); each chain symbol's target and its public place (counting); and `threats`, the wins this seat would have with one action if it were its turn now (threats).
- `hints` (`HintsView`): the wonders the game still allows (`wondersLeft`), who takes the age's last card if nobody plays again (`lastCardTo`) and who will choose the next starter (`nextStarter`), who would win on points now (`previewWinner`), both cards of each science symbol with their places (`pairs`), what each looting token would take now (`looting`), each board token's value for each seat (`tokens`), the box named by elimination while a Great Library is open (`boxTokens`, the owner's answer to question 1 of the hints design; the box order stays secret), the cards that may lie under each face-down back (`unseen`) and, in Age III, what each guild not yet seen would bring each seat (`guildPreview`).
- `players[i].wonders[k].effect` (`WonderEffectView`): what building that open wonder would do now, from `previewWonder`: coins and points for both seats, the pawn, a win, the extra turn, the wonder the seventh-wonder rule would remove, and the choice it opens with its size.
- `pending.tokenValues`: during a Great Library offer, the value of each token on offer, in the deciding seat's view only (where the offer itself is listed).

The rules behind them: a missing card symbol is open while one of its two cards is face up, unseen, or in the discard pile while the seat's Mausoleum is open (or the seat is choosing its rebuild). The scales are open while Law is on the board and the seat has a pair it may still complete, or while Law is in no city and on no board and the seat's Great Library is open. `capitalSafe` compares the distance with an upper bound of the attacker's shields: its open shield wonders and every red card that may still come to it, each with Strategy's extra shield unless the defender owns Strategy. The threats are computed only in a play turn and a starter choice (the phase gate), on a normalised copy with every card outside the face-up mask scrubbed, and are absent in a pending choice. The guild candidates are every guild in no public place, so they always include the guilds never put in the game. Two property tests play random games to the end and check the claims against what then happens: a capital called safe is never taken, a seat called unable to reach a science victory never wins by science, and a symbol called dead never enters that seat's city; another checks `lastCardTo` against the rest of the age in play turns and pending choices.

On the client, `web/src/lib/view/board-hints.ts` words these facts from the screen's point of view and `view/threats.ts` turns them into the strips' glow levels (none, calm, soft, strong): the science tracker shows each symbol's state with a tooltip naming its two cards and where they are; the military track puts a crown under each capital (an instant win), says on each zone tablet that only the player ahead scores it, shows the shields the opponent still needs at 5 or fewer, and says what each looting token would take now; wonder rows, the wonder button and the wonder sheet show what building a wonder would do now, the hourglass once one wonder is left and the ready mark of a wonder its owner could afford; board tokens say what they are worth to each seat, with a laurel on Law while it would be a sixth symbol; the age chip says who takes the last card and who will choose the next starter; and the age interstitial names a pair the ended age killed. The draft's pick chips follow `draft.order`, and the turn bar falls back to `playingAgain` after a reload. web/README.md lists the pip vocabulary and the test hooks.

Six of these fields are drawn in tooltips, worded by `web/src/lib/view/tips.ts`: `chainTargets` gives the chain chips their status lines (which seat shows the symbol, the card it makes free and where that card is, and what the chain saves today); `coinsToNextVp` and `previewWinner` are lines in the tooltips of a strip's coins and points; `guildPoints` is the tooltip of the purple band in a city panel; and `unseen` and `guildPreview` fill a face-down back's tooltip (the cards that may lie under it, how many of them are set aside, and in Age III what each guild among them would bring each seat). The chain lines and a back's candidates follow the counting family, the other three the scoring family. `view/token-values.ts` supplies the token pick's value lines for both seats on a board pick and for the decider alone during a Great Library offer. `CitySheet.svelte` lists the selected seat's threats in addition to the strongest glow and military badge.

**Core fields.** Three facts that every view carries, whatever the settings, came with the hints: `players[i].shields` (the shields each seat has placed, Strategy's included, which the cities cannot tell because Strategy counts only for red cards built after it), `playingAgain` (the current turn is an extra turn, so the client no longer infers it from the event history) and `draft.order` (the seats of the round's four picks).

**Copy-only simulations.** The hints are built from a few helpers that simulate on a copy of the state and never touch the structure: `previewBuild` and `previewWonder` (what building a card or a wonder would do for a seat now, stopping before any choice it opens; the Great Library preview never draws), `priceShift` and `savesInView` (how a card would move trade prices and the payments in view), `tokenPreview`, `extraTurnOutcome` (from the rules alone, since an applied copy has already used up the extra turn), `publicPlace` (where a card is as far as both players can know: face up, a city, tucked, the discard pile, unseen this age, unseen in a later age, or certainly set aside in an age that is over), and `WinsNow`, which the Greedy bot shares (see [ai.md](ai.md)).

**Public information only.** A hint for a viewer must come out the same on every state that viewer cannot tell apart from the real one. So every simulation runs on a copy whose Great Library offer is cleared, a follow-up after a simulated action may only use the cards that were face up before it (the face-up mask), and `WinsNow` stops as soon as an action has revealed a card, dealt an age or drawn the Great Library. The decider's hints run under a fixed budget of 20,000 Applies per view (`hintApplyBudget`, counted by `simCtx`), a count rather than a clock, so a hint is a pure function of the state and the settings; what the budget cuts is left out, never guessed.

**The leak guard.** `hints_leak_test.go` builds, for random states and every named scenario, each viewer's view with every hint family on, and checks that it is identical on the real state and on several determinizations of that viewer's observation (the decider keeps its Great Library offer). This one equality catches any hint, present or future, that reads a face-down card, the deals, the removed cards, the box guilds, the box order or a hidden offer. A second test traces every simulation and checks that no shared hint simulates for the waiting seat during a Great Library offer and that every follow-up stays inside the face-up mask. The fuzz target `FuzzHintsLeak` runs the same checks on fuzzer-chosen games in the nightly fuzz job.

**Historical cost measurements** (Apple M2, `go test -bench` in `internal/engine`, M10.HS1, with the frame measured again in M10.HS2; these do not measure the complete current hint set):

- `BenchmarkComputeHints/frame`: the hints and the three views of one broadcast. About 42 to 47 microseconds in M10.HS1, when the hints were still empty; about 50 to 60 microseconds, 40 kilobytes and 444 allocations with the per-card hints of M10.HS2.
- `BenchmarkComputeHints/projected`: the then-projected shared work, measured with the helpers (a `previewBuild`, a `priceShift` and a `savesInView` for each face-up card and seat, a `previewWonder` for each open wonder, a `tokenPreview` for each board token and seat, and `publicPlace` for all 73 cards) on a mid-Age II state: about 20 microseconds and 17 allocations.
- `BenchmarkDeciderHints`: the whole of `deciderHints` (M10.HS3) for the seat to move. On the busiest play state of a seeded game (18 actions) it spends 533 Applies in about 122 microseconds; on a mid-age state about 32 microseconds. Over a corpus of 319 states from seeded games the most any view spent was 6854 Applies, so the budget of 20,000 never cut anything; the median was about 38 microseconds and the worst about 1.15 milliseconds. (In M10.HS1, `WinsNow` and `extraTurnOutcome` alone took about 7 microseconds and 34 Applies on the busiest state.)

**Frame size.** The core fields and the empty `hints` object add 56 bytes to a mid-age view: measured on seeds 1 to 3, with both seats played by the simple policy, at the first play state of Age I and of Age II with at most ten cards left, the decider's view went from 15402 to 17989 bytes before to 15458 to 18045 bytes after (seed 1, Age II: 16826 before, 16882 after).

The per-card hints of M10.HS2 add about 2 to 4.4 kilobytes, measured by `TestHintsFrameSize` with every family on, at the same kind of positions (the decider's view without and with the hints):

- seed 1: Age I 15013 to 17623 bytes, Age II 14589 to 18980;
- seed 2: Age I 15190 to 17798, Age II 13597 to 15621;
- seed 3: Age I 17147 to 19245, Age II 13705 to 17752.

The board-level hints of M10.HS4 add about 3.8 to 4.1 kilobytes more at the same positions (`HintsView` about 2.2 to 2.4, the two `players[i].hints` 1 to 1.5, the wonder effects 0.3 to 0.6), so the shared hints now take the decider's view to:

- seed 1: Age I 15013 to 21527 bytes, Age II 14589 to 22941;
- seed 2: Age I 15190 to 21943, Age II 13597 to 19495;
- seed 3: Age I 17147 to 23412, Age II 13705 to 21795.

Compressed with gzip, the whole view with every hint is 2.8 to 3.1 kilobytes, against 1.6 to 1.8 without. The test fails if the shared hints ever add more than 60% (a third before M10.HS4). The decider's outcomes of M10.HS3 add another 401 to 1383 bytes at the same positions, and the test holds them to a sixth of the plain view. Most of the cost is the `effect` of science and military cards; the first draft, which sent every effect and every discarded card, added up to 5.4 kilobytes.

**The decider's hints (M10.HS3).** In the view of the seat to move, each choice carries `outcome` (`OutcomeView`), what that action would do, while a family that needs it is on. They are not shared: the waiting seat and spectators get none, so they cannot read the mover's options. `hints_decider.go` builds them in passes:

- For each choice other than a draft pick, a starter or a resignation: both seats' coins afterwards and the change of each score total (scoring), the pawn move with the looting it takes (military), the mover's different science symbols when they change (science), the choice the action opens (`opens`), the turn order (`tempo`: an extra turn gained or lost on the age's last card, and on the last card of Age I or II the seat that will choose who starts), a win at once or through the mover's own follow-up (`wins`, threats), the game's end on points (`end`), and the trade prices and affordability it changes (`prices` and `flips`, economy; the mover's own lost affordability is left out to keep the frame small).
- A build or wonder the seat cannot pay for carries `DisabledChoice.wins` when it would win by a supremacy with enough coins.
- In the starter choice, each `starter:S` carries in `threatens` the wins seat S would have at once if it started.
- One turn ahead, for an action after which the opponent plays in the same age: `gives` (wins the opponent would have that it does not have now), `blocks` (wins it has now that the action takes away) and `threatens` (wins the mover would have on its next turn). Each runs on a scrubbed copy: every slot outside the face-up mask becomes `catalog.Unknown`, so a nested simulation can never use an unseen card.
- `revealRisk`, where the action turns cards over and the opponent has no win otherwise: the chance, as a reduced fraction `num/den`, that a turned card lets the opponent win. Each candidate from the pool of cards the mover cannot see is tried in one probe slot, and the chance is 1 − C(m−k, n)/C(m, n) for m unseen cards, k of them winning, and n turned over.

The client words all of this in `web/src/lib/view/outcome.ts`: the outcome line under each move button (`OutcomeLine.svelte`), the card sheet's "After this pick" panel (`AfterPickPanel.svelte`, with what the pick uncovers from `view/freed.ts`), the starter choice's threat and opening lines, and the selection ghosts that `MilitaryTrack`, `PlayerStrip` and `CityPanel` draw from a `ghost` prop. A move that hands over a win or misses one is warned about in the confirm step only, so with Confirm moves off the outcome line and the panel say it alone. The only arithmetic in the client is display: the change of a seat's coins is the server's coins after minus the view's coins now.

The leak guard covers the outcomes with no extra code, because it compares whole views. `TestDeciderSimulationsStayPublic` traces the nested simulations and checks that every state they run on is scrubbed, apart from the one probe slot of the reveal risk. The wire uses `Pair<Amount>` for the coin and score pairs (a named element type, as the DTO test requires) and pointers to slices for the optional threat lists.

**The move review (M10.HS5).** The advisor family is one thing: after a seat's play action (a build, a discard or a wonder), whether a clearly better action existed in that position, and why, for that seat alone. There is no way to ask before moving. The room keeps, for each seat, the position before its last play action and the action (`internal/room/review.go`; after a load or an undo it replays the log), and computes the review the first time that seat's view needs it, while the family is on; `frames.go` puts it in that seat's own view only (`View.review`), never in the opponent's or a spectator's.

- The ranking is `ai.RankActions` (`internal/ai/rank.go`): the Greedy search of every legal action over 16 determinizations of the mover's own observation of the position before the move, with a seed derived from the room's seed and the ply, under a budget of 120,000 evaluations. A clock of 50 milliseconds only guards a starved machine: a ranking it cuts short gives no review, never a guessed one. So the review is a pure function of the seed, the log and the ply.
- **Advisor strength (M11.advisor-strength).** The game's hint settings choose how carefully the review checks (`HintSettings.advisorStrength`, edited by the admin under the advisor's switch). `quick` is the Greedy ranking above, computed in the room actor. `twoStage`, the default, lets Greedy propose and then asks the tree search to agree: `ai.CompareActions` (`internal/ai/review.go`) runs the Scholar's SO-ISMCTS on the mover's observation with the root restricted to the proposed move and the move played, taken in turn on a shared determinization per pair, for a fixed iteration count on one worker; the proposal stands only if its mean reward beats the played move's by 0.04. `deep` lets `ai.SearchActions` (the same search over every move) pick the move, then checks it the same way, and gives no points. Both search strengths run off the actor in a `reviewJob`, which waits for one of the Manager's review slots (`-review-searches`, default 1), and post their result to the inbox; the room then sends the mover's connections a fresh frame. A newer move or an undo cancels the job, and a stale result is dropped by its generation. [ai.md](ai.md#the-move-reviews-search) has the budget and its cost.
- The reason comes from `engine.ReviewFacts` on one determinization of the same observation, with the face-up mask: a sure win, a win the move left the opponent (read on a scrubbed copy), a science pair completed or denied, and looting. Among the actions within 1 point of the best that beat the one played by at least 3, the one with the strongest reason is named; otherwise the reason is `points` with the gap. Nothing is said in a decided position (the move kept a win, or every move loses) or below the 3-point gap.
- `TestReviewLeakGuard` checks, at every strength, that the review is the same on several determinizations of the mover's observation, and `TestReviewNamesOnlySeenCards` that it never names a card that was face down before the move. Cost, measured by `TestReviewCost` over 60 positions of seeded games: mean about 1.1 milliseconds, the worst about 18 milliseconds, at most about 36,000 evaluations, so the budget cuts nothing in practice.

The client shows the review inside the summary of the opponent's next move (`components/game/MoveReview.svelte`, worded by `web/src/lib/view/review.ts`), and the same summary lists the trade prices the opponent's moves changed since the viewer's own move, from the snapshots the screen saw (`view/recap.ts`, per seat, restarted by a new age, an undo or a resync). First-time tips (`ui/tips.svelte.ts`) are device-local, not a hint family: each explains a pip once, beside it.

## Rooms, identity and reconnect

- **Identity.** The browser keeps a random token in `localStorage`. The server derives the user id as an HMAC of the token with a server secret, so tokens never appear in logs or URLs. An invite link is just `/play/<code>`. The display name is stored in `localStorage` with the token, so the browser asks for it only once.
- **Device links.** A player moves to another device with a **device link**, `/play/<code>#claim=<claim>`. The claim is an HMAC of the room code, the seat's user id and a per-seat epoch (package `ident`): it works in that room only, reveals neither the token nor the user id, and stays in the fragment, so it never reaches a server log and survives a reload. A join that carries it acts in that room as the seat's user, without touching the device's own identity. The seat's player and the game's admin get a seat's link with a `link` frame (the compact icon button beside each name, `web/src/lib/components/identity/`) and revoke it with `relink`, which raises the epoch and records the reset in the room log. A device link is a password for that seat in that room; resetting it is the remedy for one that leaked. A hot-seat or bot room belongs to the user who created it, so another browser (another token) that opens its plain link, even once the game is over, is refused with `forbidden`, and the room error screen says that the game belongs to another browser and that its device link is the way to open it there. [protocol.md](protocol.md#device-links) has the details.
- **Who is who.** Equal display names in a room (compared without case and outer spaces) get stable numbers, `seats[i].nameNo`: the later seat shows "Sara 2". Only the seat that takes or changes a name is renumbered, so numbers stay when the other player leaves. The client formats every name through one helper (`web/src/lib/view/display-name.ts`). Outside hot-seat games, the viewer's own seat is marked in the strips, the city panel, the lobby and the log with an accent ring on its crest and a "you" mark, so it never relies on colour alone.
- **Seats.** In an online room the first two distinct users take seats 0 and 1, and later visitors watch as spectators with a public view. Hot-seat and bot rooms accept only their creator. Leaving a room or disconnecting only detaches that connection: the seat stays with its user, who gets it back on the next join. Spectators can never lock a player out of a full room.
- **Rematch.** Once the game is over, either player may ask for a rematch. The server creates one new room with the same mode and seats and a fresh deal, stores its code with the old room, and sends every connection of the old room a `created` frame with that one code. Every later request, from either player and even after the old room was reloaded, gets the same code. Creating the rematch room counts against the requester's room creation limits, like a `create`.
- **Hint settings.** Each game has hint settings (decision D23): twelve families of hints, each on or off, the same for both players. They live with the room, not in the engine, because they change what the client shows, never the rules: a game replays the same with any settings, and the game log (`ajayeb-log/1`) leaves them out. The creator is the game's admin, the only user who may change them, mid-game included, with a `hints` frame. Every state frame carries the settings (`room.hints`), whether its viewer is the admin (`you.admin`), and the room log (`roomLog`), the list of changes the game log shows among the moves. The client keeps two things of its own in `localStorage`: the settings its next game starts from (`ajayeb:hints`) and what this player hides on their own screen (`ajayeb:hintsHidden`), which can only take hints away (`visibleFamilies` in `web/src/lib/hints/families.ts`).
- **Idempotency.** Every action carries a per-client `seq` and the `base` version it was made against, so a double tap or a re-send after a reconnect can never apply twice. The room keys seqs by user id and client id together, so another user who picks the same client id cannot make it ignore a player's requests.
- **Rooms in memory.** A room with no connection leaves memory after 30 minutes, and sooner when the server holds its maximum of rooms and needs the slot (the one idle longest goes first); it is on disk and comes back on the next join. A join that the room would refuse (another user's hot-seat or bot room) is refused from the room file's header, without loading the room.
- **Reconnect.** After a drop, the client sends `hello` and `join` again and receives a fresh snapshot. The client re-sends a pending request unchanged, with the same `seq` and `base`; the server either acknowledges the earlier application or judges the request against the current state. A rematch request lost to the drop is sent again too.
- **Undo across a bot answer.** In a bot room the bot may answer between the player's tap on Undo and the server receiving it, so the undo arrives with a stale `base`. The client waits for the fresh snapshot and, if it still offers undo, re-sends the undo once with the fresh version.
- **Restarts.** On shutdown the server closes sockets with "service restart" and clients reconnect with backoff. Rooms are on disk, so a redeploy loses no game.

The full message contract is in [protocol.md](protocol.md).

## Tutorial rooms

The tutorial (decision D25; the design is the tutorial design of M10) is a guided first game against the Easy bot. On the server it is an ordinary `ai` room with one extra name in its file, and the engine is unchanged: a tutorial game is a normal game whose seed and options a script fixes.

- **The script.** `server/internal/tutorial` holds the scripts as literal data. A script is a seed, options (the player is the first player), the player's seat and a list of entries: an action key with its seat, and on the player's entries (the guided turns) the info lessons shown before the move and the action lesson that asks for it. The one script, `first-game`, covers the player's three draft picks and seven Age I turns with the bot's replies in between, and an outro shown once it is done. Lessons are ids such as `military` or `trade`, with the slug of the card or wonder they point at; the client words them. A seed search (`search_test.go`, run on demand) found the seed, and `script_test.go` replays the script and checks every lesson's precondition on the real deal, so a rules change that breaks it fails a test. A shipped script is never edited in place: a changed one gets a new name, and a room whose script name this build does not know plays on as a normal game.
- **Creating one.** `create` with `tutorial: true` (mode `ai`, bot `easy`, seat 0 or absent) builds the room file with the script's seed and options instead of a fresh seed, and stores the script's name in the file's `tutorial` field. Like the hint settings and the undo setting, the name is room data: the replay and the hash ignore it.
- **Derived, never stored.** Every state frame for the connection that holds the player's seat carries `tutorial`, which `tutorial.Derive` computes from the script, the action log and the state: whether the log is still a prefix of the script, the status (`guided` while it is and the script has entries left, `free` after it or once the player left it, `complete` once Age II is in play or the game is over), the lessons to show now and the one key the guided turn asks for. Because it is a function of the log, a reload, a reconnect, an evicted room that comes back and an undo all show the right lesson. `RoomInfo.tutorial` marks the room, for Home's Continue list.
- **The bot's replies.** While the log is on the script and the next entry is the bot's, the room plays that key through a bot that returns it at once, instead of asking the Easy bot to search. The think delay, the search generation and the legality check of a bot answer stay as they are. Off the script the Easy bot plays. See [ai.md](ai.md#what-a-bot-sees).
- **The rails are the client's.** The server accepts any legal key in a tutorial room, as in any other. A different move simply leaves the script; the frames then show the `leftPath` lesson until the player moves again, and the game goes on against the Easy bot. An undo back before that move brings the guided turn back.
- **No review of guided moves.** The move review (the advisor family) speaks of a seat's last play action. When that action was a move of the script, the frame carries no review, because the tutorial asked for it. Free moves are reviewed as usual.
- **The rematch** of a tutorial room is a normal `ai` room: a fresh seed, the default options, the same seats, hint settings and undo setting, and no script.
- **Development routes.** With `--dev-endpoints`, `POST /api/dev/rooms` accepts `tutorial: true` (mode `ai`; the seed and options come from the script, so the request gives neither, nor a scenario) with an optional `actions` prefix, to open the tutorial at any position, and `GET /api/dev/tutorial` returns the current script as JSON (its name, the seed as a decimal string, the options, the player's seat, the entries and the outro), so tests read the expected order instead of copying it.
- **The client's half.** The client words what the frame names and points at it (web/README.md, "The tutorial"). From the frame's `tutorial` object, the info lessons acknowledged on this tab and which sheet is open, a pure step machine picks the one coach step to show: a bubble on the board, or a note inside a sheet, with a ring on its target. During a guided turn the rails disable every other move button, a display rule only. "Play on my own" pauses that for the room, on this device. When the status turns `complete` (Age II), a completion card offers Keep playing, New game and Home, once per room. Nothing about the tutorial is sent back but the action keys.
