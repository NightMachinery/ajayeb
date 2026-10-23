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
- **`internal/enginekit`**: shared deterministic infrastructure. `rng` owns the value PCG generator, bounded draws, Fisher-Yates shuffle and FNV-1a byte digest with fixed-width hex formatting. Duel keeps its RNG names through aliases and wrappers and owns its state/catalog encoding and hashed prefix. `puritycheck` and `catalogdiff` are test-only helpers for source/shape guards and three-way catalog verification.
- **`internal/view`**: the view DTO types, with no logic. The builder lives in the engine because only the engine can read the hidden state; the types live apart so that `tygo` can read them without pulling in the rules.
- **`internal/ai`**: the bots. A bot receives an `Observation`, never the real state (see below). Its `Pool` bounds the searches running at once and shares them fairly between client addresses, so one address's many bot rooms cannot hold up another's bot.
- **`internal/room`**: one actor goroutine per room with an inbox channel, so room logic needs no locks. It owns the seats and the mode, deduplicates client sequence numbers, checks versions, schedules bot turns, implements undo and the forfeit (`resign:<seat>`, a terminal action never offered in `choices`), keeps each applied action's events for the history (rebuilt by replay on load), and writes the room file after every accepted action.
- **`internal/store`**: one JSON file per room, written atomically (temporary file, sync, rename), plus a reaper for old rooms with an injectable clock. It refuses to read a room file larger than a cap derived from what a room can hold.
- **`internal/ident`**: the server secret and the HMAC-derived user ids. The secret file must be a regular file, and one other users could read is tightened to mode 0600.
- **`internal/wire`**: the client and server message types.
- **`internal/ws`**: the WebSocket transport (`github.com/coder/websocket`): origin patterns, read limits, per-connection rate limits, pings, and an outbox that keeps only the newest snapshot for a slow client.
- **`internal/httpapi`**: `/health`, the room lookup API, the development endpoints (enabled only with both `--dev` and `--dev-endpoints`), and the static handler with its SPA fallback. Every response carries the security headers (no framing, same-origin opener and resource policies, a Permissions-Policy that turns off unused features) and a content security policy that allows only same-origin scripts plus the SHA-256 hash of each inline script of the HTML file served, computed from the served bytes (under `--dev`, CSP is report-only and same-origin pages may frame the app); see [protocol.md](protocol.md#http-endpoints).
- **`internal/webdist`**: embeds the web build only under the `embedweb` build tag, so a fresh clone builds and tests without building the web app first. It also defines the build's cache policy (`CacheControl`, which the static handler in `internal/httpapi` applies): SvelteKit's content-hashed `_app/immutable/` files are cached for a year, and so is bitmap art under `art/` requested with a `v` query parameter that matches the start of the file's SHA-256 (the generated art index writes every art path that way); art without a matching hash is cached for a day and then revalidated in the background for up to a week; everything else, including the app shell and the stable app icon PNGs under `icons/`, is revalidated on every load.
- **`cmd/ajayeb`** wires everything together. The other commands are `ajayeb-sim` (tournaments, the AI benchmark, level calibration, weight sweeps and the replay verifier; see [ai.md](ai.md)), `ajayeb-drive` (plays full games over WebSocket against a running server) and `ajayeb-gen`, the TypeScript generator. `ajayeb-gen` writes the presentation catalog and contract fixtures described below, and exports the catalog as CSV, JSON, prose or TypeScript for review (see [rules/sources.md](rules/sources.md)).

### Room-kind composition

`cmd/ajayeb` builds one closed `room.Kinds` table, currently containing only
`DuelKind`. Each stable game id binds create, cold-join admission and load
callbacks to its file codec. `NewKinds` rejects duplicate or incomplete
registrations and mismatched codec ids before startup. Its `Codecs()` table is
passed to the store, reaper and read-only preflight; there is no second list of
games in those paths. Schema-1 Duel remains required and schema 2 has no
production registration.

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
Production still registers Duel alone; no new view or rules implementation
is hidden in the platform envelope. See [protocol.md](protocol.md) for the
frozen families and the named M15 projection placeholders.

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
2. **Data.** `ajayeb-gen` writes `catalog.gen.ts`: typed slug lists and display data such as colour, cost, effects and chain links. The slug unions make incomplete translation tables fail to compile, and the gallery can draw cards without a server. There is no separate `ids.gen.ts`.
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

`lib/games/haftgane` is an empty lazy entry for table/lobby envelopes, with no
rules, renderer, catalog, actions or invented view. It retains the already
welcomed bootstrap connection and requests resync after envelope validation.
Its snapshot stays unknown until M15 supplies the typed game projection.
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

Every build of the client exposes `window.__ajayeb` (the displayed view, the room and version, the current choices, the controlled seats, connection state, history and tutorial state). It contains only what the client already received, which is safe because the client never holds hidden information.

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
resource/colour/science vocabularies and sixteen validated effect kinds. It
currently has no card data: synthetic catalogs exercise validation and slug
lookup. Registration checks field use, saleable production, copy markers,
chains and twin semantics; the later data slice pins the verified edition's
complete counts and order. The existing Duel packages and data stay unchanged.

`rules` defines fixed-value State, Record, Game, actions, events and decision
points. It contains the seat ring and canonical key grammar, with no setup,
legality, resolution, views or bots yet. Left is the next seat modulo the seat
count, and right is the previous. Public plays and private picks
have separate record arrays. Events have typed source fields and no raw key.
Point ids are computed from kind, Age and turn; pending choices stay in the room.
Actions name design slugs and payment outcomes, never copies or seats. Parsing
checks canonical syntax and the tagged union; only future offer membership can
make an action legal. The rules version starts at 1.

The ordinary payment solver uses frozen own production, neighbours' saleable
production, per-side prices and the starting purse. It returns at most fifteen
undominated `(left, right)` outcomes, ordered by total then left, each with a
canonical physical-symbol witness. Fixed own symbols are used first; a proven
maximum-own-coverage filter, bounded resource splits and separate choice-symbol
matchings keep the hot path allocation-free. `QuoteState` and `Game.Quote`
return the cheapest affordable ordinary total without building witnesses.
Chains and free-build abilities bypass this solver per action; stages use their
ordinary cost. An independent unit-by-unit oracle checks the small-universe
sweep and 100,000 deterministic random inputs, and `FuzzPayment` checks outcomes
and every witness against it.

Both the shared `enginekit/puritycheck` and a dedicated depguard rule enforce
the purity boundary. Haftgane cannot import either math/rand version, even in
rng.go; the shared generator is provided by `enginekit/rng`. The shape guard
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

`internal/store` is the only reader/writer of room files. A bounded header selects the codec before game decoding or replay: schema 1 defaults to `duel` when `game` is absent and delegates to the existing Duel `File`, while schema 2 requires an explicit registered game id. The complete input retains the existing `MaxFileBytes` cap; the schema discriminator is at most 10 JSON bytes and the game id is at most 32 lowercase ASCII letters, digits or hyphens (starting with a letter), in at most 34 JSON bytes. Discriminator order is unrestricted; duplicate schema/game fields, including case aliases, fail closed. `NewCodecs` builds an immutable composition-time table, rejecting duplicate or invalid registrations. Its production schema-2 table is empty. Codecs own their file types and game rules checks; shared inspection exposes only retention metadata and read-only replay assessment. `OpenWithCodecs` and `CheckDataDirWithCodecs` take the same table. Duel's load/save APIs retain their typed signatures and schema-1 write bytes.

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
