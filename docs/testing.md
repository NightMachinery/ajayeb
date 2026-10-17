# Testing

Ajayeb is tested in layers, from the rules themselves up to a full game
played over a real WebSocket against a real server binary. Each layer trusts
the one below it and adds what that layer cannot catch on its own.

## Engine and AI unit tests

`server/internal/engine` and `server/internal/engine/catalog` carry the
rules tests, written from `docs/rules/` before the code that implements
each rule. `server/internal/ai` and `server/cmd/ajayeb-sim` cover bot
strength and the playout sweep. These run as part of `make test-go`, in
process, with no server, no socket and no browser; they are the cheapest
layer and catch almost everything.

## Fuzzing

The module has six fuzz targets:

- `FuzzPlayout` (`server/internal/engine/fuzz_test.go`): plays a seed's game
  by fuzzed choices among the legal actions and checks the invariants after
  every action;
- `FuzzApplyGarbage` (same file): applies an arbitrary action at a state a
  few random steps in, which must either be legal and apply cleanly or be
  refused with the state untouched;
- `FuzzActionKey` (same file) and `FuzzParseKey`
  (`server/internal/engine/action_test.go`): action keys parse only when
  they round-trip exactly;
- `FuzzDecodeClient` (`server/internal/wire/decode_fuzz_test.go`): any bytes
  as a client frame, which must be refused with `ErrBadFrame` or decode to
  the frame type its `"t"` names, pass the same checks again once
  re-encoded, and round-trip unchanged;
- `FuzzHintsLeak` (`server/internal/engine/hints_leak_test.go`): plays a
  seed's game by fuzzed choices and runs the hints leak guard on the states
  it passes: each viewer's view with every hint family on must be the same
  on determinizations of that viewer's observation (see architecture.md,
  "Hints").

Their seeds, and the crashers that were fixed, run as ordinary tests in
every `make test-go`. To fuzz one locally, under the guard's `fuzz` kind
(`nice 10`):

```sh
python3 tools/heavy.py run --kind fuzz -- go -C server test -run '^$' \
  -fuzz '^FuzzDecodeClient$' -fuzztime 30s -parallel 2 ./internal/wire
```

(`make fuzz FUZZTIME=5m` runs `FuzzPlayout`; `make fuzz FUZZ=FuzzDecodeClient FUZZ_PKG=./internal/wire` runs another target.) The nightly CI run
(`.github/workflows/ci.yml`, jobs `fuzz-targets` and `fuzz`) finds every
`func Fuzz...` in the module's test files by itself and fuzzes each one on
its own runner for 20 minutes, in parallel, so a new target needs no
workflow change. The corpus a fuzzer grows stays in the Go cache, carried
from night to night in the Actions cache (and dropped once it passes
256 MB); it is never committed, so the `testdata/fuzz/` directories hold
only hand-picked seeds and fixed crashers. A crasher is uploaded as the
artifact `fuzz-crasher-<target>`, with its repository path. To reproduce
one, copy the file into that path (for example
`server/internal/engine/testdata/fuzz/FuzzPlayout/<hash>`) and run
`make test-go GO_PKGS=./internal/engine GO_RUN=FuzzPlayout`; commit it with
the fix, so it stays a regression test. The push-triggered CI runs only the
seeds, as part of the ordinary tests.

## The view and protocol fixtures

`server/internal/engine/testdata/protocol/` holds one golden JSON file per
named position: what every viewer (`seat0`, `seat1`, `spectator`) sees
there, full view and filtered events side by side. `TestProtocolFixtures`
in `server/internal/engine/view_test.go` writes and checks them
(`-update` after a deliberate change). [protocol.md](protocol.md) describes
the format in full; this page only says where it sits among the other
layers and how `ajayeb-gen fixtures` turns it into something the web
client's own type checker reads (below).

## `internal/protocoltest`: full games over the real stack

`server/internal/protocoltest` plays whole games over real WebSocket
connections against an `httptest` server built from the actual `ws`,
`room` and `store` packages, the same wiring `cmd/ajayeb` uses. Nothing
here is mocked: a room is a real room actor, a bot is the real AI package
(with a one-iteration budget, so it is fast and deterministic without being
a different code path), and the socket is the real protocol.

It covers, per game mode (online with a spectator, hot-seat, versus a bot):

- a full playthrough to a result, with the replayed action log hashing to
  the same state as the room's own final state (`engine.Hash`);
- that no hidden information (a face-down card's identity, a later age's
  undealt deck, the cards set aside from every age, box guilds, wonders and
  tokens, the Great Library offer for anyone but the seat choosing from
  it, the seed) appears in a connection's raw frames before it is entitled
  to see it. The check reconstructs what should still be hidden from the
  room's own ground-truth state (`room.Manager.Snapshot`) and scans every
  frame byte for it. It is written independently of
  `internal/engine/view.go` from that file's documented redaction rules,
  not copied from its logic. A dedicated test starts an online room at the
  Great Library's draw, the one secret only one seat may see, so that case
  is checked in every run rather than only when a random game builds it.
  The drawer may see its drawn tokens anywhere in the frames of the
  `pickLibraryToken` step, but after it only in its own `libraryDraw`
  events (live and in its history); the two tokens it put back are a
  leak anywhere else in its frames;
- reconnecting mid-game (the socket drops, `hello` and `join` again, and
  resending the last `act` with the same `seq` does not apply it twice);
- restarting the server on the same data directory mid-game and finishing
  the same game, with the same final hash;
- a stale `base`, a duplicate `seq`, an illegal key and an out-of-turn
  action, each rejected with the right code;
- undo in hot-seat and versus-bot rooms, checked by the resulting turn and
  choices rather than by the room's version number (undo does not rewind
  the version; it is a monotonic counter that goes up on every accepted
  request, undo included).

Run it with the race detector, through the guard, from `server/`:

    python3 ../tools/heavy.py run --kind test -- go test -race ./internal/protocoltest/...

## `ajayeb-drive`: a thin client against a running binary

`server/cmd/ajayeb-drive` is a small WebSocket client that plays full games
against a server that is already running, the way a real client would: it
never computes legality or cost, it only picks a random choice from the
view it is sent. It takes `--origin`, `--mode online|hotseat|ai`, `--games`
and `--seed`, exits non-zero on any protocol error or a game that ends
without a result, and prints one summary line per game. `--seed` seeds the
driver's own random choices (game i uses seed plus i); the server deals each
game from a seed of its own.

`--concurrency` plays several games at once against the one server, which
is how we measure the server's resident memory under load: start a test
binary and run `ajayeb-drive --games 20 --concurrency 20 --server-pid PID`,
which samples that process's resident memory with `ps` every 100 ms while
the games run and prints the peak. Each connection sends a
synthetic `X-Forwarded-For` address, so a server started with `-trust-proxy`
naming the driver's own address can be driven past its default
per-address socket cap (16) for a true 20-connections-at-once run; without
`-trust-proxy` the header is simply ignored. A measured run of 20
concurrent bot games against a local test binary peaked at about 19 to 20
MB resident, comfortably under the working target of 40 MB for this
scenario (see [dev/ram-policy.md](dev/ram-policy.md) for the server's other
budgets).

## The TypeScript fixture check

The client's generated types (`web/src/lib/wire/*.gen.ts`, from `tygo`)
describe the wire shapes, but nothing used to prove that a real server
frame actually fits them. `ajayeb-gen fixtures` closes that gap:

- it reads every recorded frame in
  `server/internal/wire/testdata/messages/` and every viewer's view from
  `server/internal/engine/testdata/protocol/`, and splices them, almost
  verbatim, into `web/src/lib/wire/fixtures.gen.ts` as two `satisfies`
  assertions: `fixtures satisfies ServerMessage[]` and `protocolFixtures
  satisfies View[]`;
- `pnpm -C web check` (`svelte-check`, part of `make check-web`) then fails
  the moment a real recorded shape no longer fits the generated types,
  rather than at runtime in a browser;
- `web/src/lib/wire/fixtures.test.ts` adds a few runtime checks on top: the
  arrays are non-empty, every recorded frame type is represented, and
  nothing in a real frame or view is a JSON `null` (the engine's own
  convention is an empty array or an absent key, never `null`; a nil Go
  slice marshaled without this convention is exactly the kind of mistake
  this check exists to catch).

`ajayeb-gen` with no subcommand runs `fixtures` with its defaults, so
`make gen` and `make gen-check` regenerate and check this file along with
everything `tygo` produces.

Running `ajayeb-gen fixtures` for the first time (while
`server/internal/wire` and `server/internal/engine` still used `tygo`'s
fallback `never` type for a cross-package field, `view.GameEvent`) is what
actually found that gap: `tygo.yaml` needed a `type_mappings` entry for it,
exactly as the file's own comment already said it would. That is the
fixture check doing its job: a real recorded `state` frame's `events` field
stopped type-checking the moment something tried to use it as more than an
opaque blob.

## The web client

The client is thin, so its tests check plumbing rather than rules: that it
says the right frames to the server, and shows what the server sent. They
are Vitest unit tests next to the code (`make test-web`), in Node, with no
browser and no server:

- `src/lib/catalog/index.test.ts`: the presentation catalog
  (`web/src/lib/wire/catalog.gen.ts`, written by `ajayeb-gen`) holds the
  whole base game, links chains both ways, carries the same `catalogHash`
  as every recorded view, and knows every slug the recorded views mention.
- `src/lib/session/ws-transport.test.ts`: the WebSocket transport against
  a fake socket and fake timers. Reconnect delays double from 500 ms up to
  a 15 s cap with jitter (each delay lies between half its ceiling and the
  ceiling), the state reads `offline` after several failures in a row, a
  1008 close with reason `protocol` or `bad-hello` stops the retries for
  good while every other close retries, and `close()` never reconnects.
  The browser's `offline` event drops an open socket at once (no close
  event may come for a long time, and a move sent on it would be lost),
  and its `online` event retries without waiting out the backoff.
- `src/lib/session/session.test.ts`: the Session against a scripted
  transport (`scripted-transport.ts`). It says `hello` on every new
  socket; acts carry the next `seq`, the current version as `base` and, in
  a hot-seat room, the seat to move; only a key from the view's `choices`
  is sent; a state frame that acks the `seq` (with this tab's client id)
  or a reject that names it settles the request; a pending request turns
  `slow` after 300 ms; after a reconnect the Session joins again and
  re-sends the pending request unchanged; a `reject rate` for the pending
  request re-sends it after a pause, three times at most; reject and error codes map to
  message keys; `canAct` comes from the frame alone.
- `src/lib/session/identity.test.ts`, `rooms.test.ts`,
  `presenter.test.ts` and `fixture-transport.test.ts`: the token (22
  base64url characters, made once, a malformed stored value replaced), the
  per-load client id, display-name cleaning, the saved rooms list, the
  Presenter's queue, minimum thinking dwell and fast-forward, and the
  fixture transport and the dev room endpoint client.

### The client test hook

Every build of the client defines `window.__ajayeb`, a read-only window
onto the game the page shows, for end-to-end drivers:

- `view()`: the view on screen, or `null` before the first state;
- `choices()`: that view's legal choices; each `key` is also the
  `data-act` of a button on the page;
- `room()`: the room on screen with its `code`, `version`, `canUndo` and
  `thinking` seat;
- `you()`: the seats this connection may act for, and `canAct`;
- `conn()`: `connecting`, `open`, `reconnecting` or `offline`;
- `ready()`: true when the page shows the latest state and takes input
  (welcomed, nothing queued in the Presenter, no request pending);
- `history()`: the game's history as the session holds it, one entry per
  applied action (the `history` field of the state frame, kept across live
  frames), so a spec can check the log after a reload.

A second hook, `window.__ajayebFx` (`web/src/lib/fx/index.ts`), records
the sound cues and vibrations that would play, because CI has no audio
device: `cues()` (each with its delay and the frame version that made
it), `haptics()` and `clear()`. A cue is recorded only while Sound is on
and a vibration only while Haptics is on, which is what "would play"
means.

It is safe in production because it exposes only what this client already
holds: the redacted view the server sent this viewer. It never exposes the
identity token. Every value is a copy, so a driver cannot change the page.

A driver opens a room (for a hot-seat game, `/dev/new?mode=hotseat` in a
build with the dev routes), then repeats: wait for `ready()`, read
`choices()`, click the `[data-act="<key>"]` button of one of them, until
`view().phase` is `gameOver`. `e2e/fixtures/driver.ts` is that driver, with the
clicks a player makes for each kind of move (see "End-to-end specs" below).

### Dev pages for the client

With the dev routes on (they are on in the test-mode build from `make build-web-test`, and in any build with `PUBLIC_DEV_ROUTES=1`):

- `/dev/new?mode=hotseat|ai|online&level=easy|normal|hard&seat=0|1|random`
  creates a room over the WebSocket and opens it at `/play/<code>`. With
  `seed=` or `scenario=` it uses `POST /api/dev/rooms` instead, which needs
  a server started with `--dev --dev-endpoints`.
- `/dev/fixture` lists the recorded game states, and
  `/dev/fixture/<name>` shows one (such as `state-play` or
  `pick-library-token-seat1`) through the fixture transport with no
  server. A view is named by its scenario file and viewer joined by a
  dash, never a dot, since the server answers a path with a dot with 404.
  It sets `data-ready` on `<html>` once the state is on screen and
  the fonts have loaded, for screenshots. Append `?link=connecting`,
  `?link=reconnecting`, `?link=offline` or `?link=closed` to show the link
  state after loading that snapshot; the default is `open`. The
  `online-age-start-seat0` scene includes the recorded starter choice event
  so its who-starts bar can be checked without playing an age.

A token written into `localStorage` for a tool (`ajayeb:token`) must have
the real shape, the base64url form of 16 bytes: 22 characters whose last
is `A`, `Q`, `g` or `w`. The client replaces any other value with a fresh
token, and `tools/chrome/shot.mjs --token` refuses one.

## End-to-end specs (Playwright)

`e2e/` holds the Playwright specs. They run against the real server binary
and the test-mode web build, which the Playwright config starts itself
(`server/bin/ajayeb --dev --dev-endpoints --bot-delay 0 --seed-base 1000`
on port 38954, with a fresh data directory under `.cache/e2e/`), in the
installed Chrome locally and in Playwright's Chromium on CI. There are two
projects: `desktop` (Desktop Chrome, a mouse) runs every spec, and `mobile`
(Pixel 7, touch) runs the specs tagged `@mobile`.

Run them through the guard, which holds the machine's one browser lock:

- `make e2e` builds the server and the test bundle, then runs every spec;
- `make e2e GREP=hot-seat` or `make e2e GREP=layout` narrows the run with
  Playwright's `--grep`, and `PW_ARGS='--project desktop'` picks one
  project;
- `E2E_BUILD=0` skips the two builds when they are already current;
- `make e2e-smoke` runs only the `@smoke` spec.

Reports, traces and failure screenshots land in `.cache/e2e/` (ignored by
git). The config stops the server and the browser at the end of a run. A
click or other action gives up after 20 s (`actionTimeout`), so a control
that something covers fails with its call log instead of waiting out the
whole test.

### The shared helpers (`e2e/fixtures/`)

- `driver.ts`, the game driver of the hot-seat, vs-AI and online
  playthroughs. `GameDriver` waits for this client's
  turn (`waitTurn`: `window.__ajayeb.ready()`, and `you().canAct` for a
  seat of its own to move, or the game over), reads a `Snapshot` (version,
  ply, phase, age, the seat to move, the pending choice, `choices()`, the
  result, whether the pass-device screen is up, this connection's seats,
  the thinking seat and `canUndo`), asks a `Policy` for a move, performs it
  by clicking what a player clicks, and waits for the room's version to
  pass the old one and the page to be ready again. It never sends a frame
  and never decides what is legal: every key it clicks is the `data-act`
  of one of the view's choices. The flows it knows:
  - the draft: the tile's Take button where tiles show it, else the tile
    (`data-pick`) and then Take in its sheet;
  - a pending choice: the option's tile (`data-pick`) when the panel has
    tiles, then the panel's `data-act` button;
  - build and discard: the structure card (`data-card`), then the card
    sheet's button;
  - a wonder, card first: the card, "Build a wonder"
    (`data-control="wonder-step"`), then the wonder's button;
  - a wonder, wonder first: the wonder in the mover's panel
    (`.seat-panel[data-seat] button[data-wonder]`), "Build this wonder"
    (`data-control="wonder-build"`), the marked card (`data-marked`), then
    the confirmation in the card sheet;
  - the pass-device screen: "I'm ready".

  With `confirm: true` (online rooms, where the settings' "Confirm moves"
  is on by default) every move button takes the confirm step: the first
  tap must arm it (`data-armed`), and a second tap after the button's
  350 ms double-tap guard sends the move. That guard is the one fixed wait
  in the driver. The pending choice panels' commit buttons take the step
  too, since every move button follows the same rule.

  It closes toasts with their dismiss buttons before tapping the board, as
  a player would, and the bar that says who starts a new age (its close
  button): on a phone toasts float over the player's own strip, wonders
  included, the who-starts bar takes height from it, and toasts stack, so
  it looks again after each one it closes.
  It counts every flow, every pending kind, every pass screen, every undo
  and every confirmed move in `driver.stats`.

  Against the bot, `waitTurn` is what waits out the bot's turn: after the
  player's move the page shows the bot's thinking frame, the Presenter
  holds the bot's answer for its dwell and previews its pick, and only
  then is the page ready with a move of the player's to make. Nothing in
  the driver sleeps a fixed time. The other entry points are `nextTurn`
  (the next move to make, past any pass-device screen), `play` (one
  policy move), `playToEnd`, `undo` (the top bar's undo, after closing any
  toast that covers it) and `watchThinking` with `thinkingSeen`, which
  count from the page's first load how often the bot's thinking indicator
  appeared: the turn bar's thinking state (`data-turn="aiThinking"`) and
  a seat panel's thinking dots (`.seat-panel[data-thinking]`), each on
  the change from absent to shown.
- `hotseatPolicy` in `driver.ts`: deterministic, and chosen to walk every
  flow rather than to play well. The draft takes the first wonder on offer;
  a pending choice takes its first option; in play a wonder is built
  whenever one is legal until both wonder flows have been used (card first,
  then wonder first) and after that on every fourth ply; every fifth ply
  discards; otherwise the first legal construct, else the first discard.
- `ws-game.ts`, a minimal protocol client in Node (hello, join, act, with
  `seq` and `base`, paced under the frame rate limit). `recordGame` plays a
  seeded hot-seat game over the real socket with a fixed policy (odd plies
  discard, so the game reaches Age III) and returns its keys with the
  positions a layout check wants: the draft, a full Age I, Age II and Age
  III structure, and the end. `createDevRoom` then opens the same game at
  any of them through `POST /api/dev/rooms` with the seed and a prefix of
  the keys. `recordOnlineGame` plays a whole online game from a seeded dev
  room (optionally at a scenario) with two players and a spectator, each
  its own user on its own socket, and returns every frame each of them
  received, for the leak check's own test.
- `leak-check.ts`, the frame leak check (see "The frame leak check"
  below), and `online.ts`, the online specs' helpers: a browser context
  per device with the project's viewport and touch (`newDeviceContext`),
  the frame capture (`FrameLog`, from `page.on('websocket')` and
  `framereceived`, across reloads), Home's create flow, the invite read
  from the lobby, the name prompt, the server's room file and the online
  policy (the hot-seat policy, but it drafts and builds the Great Library
  whenever it can).
- `layout-probes.ts`, the objective probes: horizontal overflow (the
  document's `scrollWidth`, and the visible part of every element, clipped
  by its scrolling ancestors, past either edge), page scroll, the size of
  the takeable structure cards and of the enabled primary controls (move
  buttons, panel and draft tiles, the top bar, the wonder flow, the result
  screen, the pass-device screen and every button of an open sheet).
- `hygiene.ts`: console errors, uncaught exceptions, failed requests and
  HTTP errors, collected for a spec to expect empty.
- `scenario.ts`, the helpers of the scenario specs below.
  `openHotseatRoom` opens a hot-seat room at a named scenario (or at the
  ordinary start) through the test build's `/dev/new` page, always with an
  explicit seed (7 unless a spec names another), so these rooms never use
  up a seed from the server's `--seed-base` counter and the games the other
  specs create over the socket stay the same whether or not these ran
  first. `playKey` plays one action key of a scenario's trigger with the
  driver's clicks (the panel for a pending choice, the card sheet for a
  build or discard, card first for a wonder unless the spec asks for wonder
  first) and waits for the room to move on. `logLines` reads the game log
  from the top bar's log sheet. `expectResultScreen` checks a finished
  game: the server's result has the expected kind and winner, the result
  screen says the same (`data-victory`, `data-winner`, the kind line and
  the headline), the tie-break note shows exactly when the totals tie, and
  the score table shows the server's score sheet line by line.

### `hotseat.full.spec.ts` (`@mobile @full`)

A whole hot-seat game through the real UI, on both projects. From Home it
opens the pass-and-play sheet, names the players, sets the pass-device
screen on for the mobile project (with the "face active player"
orientation) and off for the desktop one (fixed orientation), and starts.
The driver then plays to the end with the hot-seat policy and the spec
checks:

- every flow was walked: six draft picks, builds, discards, at least one
  wonder card first and one wonder first, and pass-device screens exactly
  in the mobile project;
- the result screen names the server's victory and winner, and its score
  table shows the server's score sheet, category by category, under both
  names;
- Home leads back to the home page;
- no console error, page error or failed request on the way.

The server's `--seed-base 1000` deals the rooms created over the socket
from seeds 1000, 1001 and so on, in the order they are created, so the
games are the same on every run of the same set of specs (a narrower
`GREP` creates fewer rooms first and so deals other games). Measured in a
full `make e2e` on the laptop, the same on two runs in a row:

- desktop: 69 actions (6 draft picks, 3 pending choices, 34 builds, 21
  discards, 2 wonders card first and 3 wonder first), about 60 s;
- mobile: 70 actions (6 draft picks, 4 pending choices, 39 builds, 14
  discards, 4 wonders card first and 3 wonder first, the Great Library's
  draw among the choices) and 57 pass-device screens, about 35 s.

Since the online spec's first game (desktop project) takes a seed of its
own, the mobile project's games here and in `ai.full` were re-measured
with it in the run (M7.S02); the desktop ones did not change.

The spec prints its counts on one `hot-seat game:` line and attaches them
as `game-summary.json`.

### `ai.full.spec.ts` (`@mobile @full`)

Games against the bot through the real UI, on both projects. The bot's
turns are waited out with the driver's `waitTurn`, never with sleeps. The
config's `--bot-delay 0` makes the server answer at once, so the pacing a
player sees is the client's own: the thinking frame stays up for the
Presenter's dwell (300 ms with Animation off, which the e2e setup forces),
then the bot's pick is previewed for as long again.

The first test plays a whole game against the Apprentice (`easy`). From
Home it opens Play vs AI, picks Apprentice and a crest (the Sun on the
desktop project, the Moon on the mobile one) and starts, then plays to the
end with the hot-seat policy for the player's seat. It checks:

- the room is an `ai` room with an `easy` bot in the other seat, and this
  connection holds only the chosen seat;
- the military track puts the player's capital at its left end, mirrored
  (`data-mirrored`) at the Moon;
- one undo in the middle of Age I or II, from the top bar, after the bot
  has answered: the bot's reply goes with the player's move, so the ply,
  the structure and the choices are those from before the move, nobody is
  thinking, and the turn bar says it is the player's turn;
- the bot's thinking indicator appeared, both in the turn bar and in the
  bot's strip or panel (counted with `watchThinking`);
- the result screen shows the server's victory and winner ("You win",
  "Apprentice wins" or a shared victory), and its score table shows the
  server's score sheet category by category;
- Home no longer offers to continue the finished room: a finished room
  leaves Home's Continue list (and `ajayeb:rooms`) as soon as a state
  frame says its game is over;
- no console error, page error or failed request on the way.

The second test starts a game against the Scholar (`normal`), plays three
moves, leaves with the game menu (`data-control="menu"`, then Home), and
comes back through its "Continue: vs AI" entry on Home. The board must be
restored exactly: the same room version, ply, phase, seat to move,
choices and `canUndo`. It then plays one more move to show the game goes
on.

The Apprentice is deterministic (its random numbers come from the room
seed), so the first test replays the same game on every full run.
Measured in a full `make e2e`, the same on two runs in a row:

- desktop (Sun): 71 plies, 34 actions by the player (35 sent, one undone)
  and 37 by the bot; 3 draft picks, 1 pending choice, 20 builds, 8
  discards, 1 wonder card first and 2 wonder first; the undo at ply 15;
  the thinking indicator shown about 30 times; a civilian win for the
  player, 63 to 45; about 60 s;
- mobile (Moon): 70 plies, 35 actions by the player (36 sent, one undone)
  and 36 by the bot; 3 draft picks, 2 pending choices, 16 builds, 11
  discards, 2 wonders card first and 1 wonder first; the undo at ply 15;
  a civilian win for the bot; about 35 s.

The Scholar test takes about 9 s on each project. The first test prints
its counts on one `vs-AI game:` line and attaches them as
`game-summary.json`.

### `layout.spec.ts` (`@mobile`)

The game screen at the UX plan's viewports: 360x740, 390x844, 412x915,
430x932, 844x390, 740x360, 768x1024, 1024x768, 1280x800 and 1920x1080.
The desktop project checks all ten; the mobile project checks the eight
phone and tablet sizes through 1024 px with touch, and skips the rest. A seeded game
is recorded once per worker (`recordGame`), and each viewport opens it at
five positions: the wonder draft, a fresh Age I, Age II and Age III
structure, and the result screen, with the pass-device screen off. At each
position, and again with the card sheet of the first takeable card open
in each age:

- no horizontal scroll and nothing visible past either edge;
- no page scroll;
- every takeable card at least 44 by 44 px (the plan asks for this at
  360x740 in all three ages; the spec checks it everywhere);
- every enabled primary control at least 44 px tall;
- no console error, page error or failed request.

The spec logs one line per position with the layout class, the smallest
takeable card and control, and the document's scroll size. At 360x740 the
smallest takeable cards measured 54x83 px in Ages I and II and 46x71 px in
Age III.

### `victory.spec.ts` (`@mobile`)

One test per way a game ends, each from a named scenario
(`server/internal/engine/scenarios_named.go`, listed in
[dev/scenarios.md](dev/scenarios.md)) whose trigger the spec plays
through the UI:

- military supremacy: `military-supremacy` (Walls' two shields reach the
  capital) and `military-via-wonder` (the Statue of Zeus's shield, played
  wonder first, ends the game before its destroy choice can open);
- scientific supremacy: `science-supremacy` (University's globe as a
  sixth distinct symbol) and `law-supremacy` (a science pair's board token
  choice, then Law's scales as the sixth);
- the civilian count at the end of Age III: `civilian-end`;
- the ties: `tie-blue` (equal totals, broken on blue points, with the
  tie-break note) and `draw` (equal totals and equal blue points: a shared
  victory, with no winner).

Each test checks the result screen with `expectResultScreen` against the
server's result, then a few facts of the scenario's Expect from the final
view (the pawn in the capital and no destroy choice, Law in the winner's
tokens, the equal totals of a tie), that the turn bar says the game is
over and no move is offered, and that the page logged no error. The
`military-supremacy` test also checks the replay download: the game menu
has no "Download replay" item while the game runs, has one once it is
over, and the file it saves is named `ajayeb-<code>.json` and holds the
room's `ajayeb-log/1` log (format, seed, scenario, one entry per action
played, final hash). Each test takes 2 to 4 s.

### `animation.spec.ts` (`@mobile`, one case `@perf`)

The move and event animations' four acceptance checks (UX design section
12), and Replay last move, on both projects. Each case writes its own
Animation speed with `fixtures/anim.ts` `useSettings` (after the base
fixture's motion-off override) and opens its own rooms through `POST
/api/dev/rooms`: seed 11 from its start or after its draft (played over the
socket with the first offer each time), the first position of that game
with an affordable build that trades and reveals a card (`findPosition`),
and named scenarios.

The scripted sequences, each one or more moves: that build, a discard, a
wonder that moves coins (`appian-economy`), the Arsenal looting twice
(`looting-double`), a bot's reply, a science pair and its token
(`science-pair`), the Great Library and its token (`great-library`), the
Statue of Zeus destroying a card (`zeus-destroy`), the Mausoleum
rebuilding one (`mausoleum`), the last card of Age I and the starter's
answer that deals Age II (`age-transition-chooser`), two draft picks, and a
game's last card (`civilian-end`).

- **a. Off is instant.** `recordMotion` wraps `Element.prototype.animate`
  and listens for `animationstart` and `transitionrun` before the app
  loads. Every sequence is played with Animation off: nothing may run
  longer than 10 ms, the runner plans and starts nothing, and the moment
  each frame lands the coins (outside the draft), the pawn and the face-up
  cards on screen are the view's, with nothing hidden and nothing in the
  FX layer or its top part.
- **b. Purely visual.** The same sequences, each once at normal speed and
  once off, in fresh rooms; after the page is ready and every animation has
  ended, the game screen's DOM is serialized one tag per line
  (`screenDom`: the room code, `data-fx-hidden`, ids, `aria-describedby`,
  transforms in style attributes and an art image's `loaded` class taken
  out) and the two must be identical. A difference shows as the lines that
  differ.
- **c. Input back within 450 ms.** `recordInputBack` notes, at each
  animation frame, the first moment the test hook's `ready()` and
  `you().canAct` are true for each version. For six hot-seat moves, an age
  change (its last card, the starter's answer whose frame deals the new
  age, and the first move of that age, which waits only for its own card
  to be dealt) and three bot answers that animated, that moment minus the
  frame's `swapAt` (`__ajayebFx.frames()`) must be at most 450 ms. The
  bot's dwell and pick come before the swap and are not counted. The
  failure message lists every measurement. Like d it measures wall time,
  so it retries up to twice on this often-loaded laptop.
- **d. No long task during an animation (`@perf`).** A CDP session sets
  `Emulation.setCPUThrottlingRate` to 4, a buffered `PerformanceObserver`
  collects `longtask` entries, and ten hot-seat moves are played, each
  after the last one's animations ended, then an age change with the move
  after it, then a game's last card. Each room is its own page load, so
  its records are read before the next. No long task may overlap an
  animation's window (`__ajayebFx.anims()`: from its WAAPI `ready`, the
  first frame it really moves, to its end), and a deal and a victory banner
  must have played. The annotation `perf` gives the moves, animations,
  long tasks, how many overlap and the worst overlapping and overall task
  in ms: read the worst overlapping one against the 50 ms limit, and the
  overall one for how loaded the machine was. This laptop is often busy,
  so the case retries up to twice; a pass with the failed runs within
  20 ms of the limit is acceptable.
- **e. Replay last move.** With Animation off the log sheet's control is
  disabled and its tooltip (keyboard focus on desktop, a tap on the
  phone) says to turn Animation on. At normal speed it is disabled before
  any move, then after the Arsenal's looting it replays: new animations
  are recorded, among them the looting, no move is sent (the version stays)
  and the screen afterwards is the screen before.

To film the animations for a visual check, slow them with CDP
`Animation.setPlaybackRate` (0.2 shows a 320 ms flight over 1.6 s) rather
than shooting faster; a headless screenshot takes 50 to 300 ms here.

### `flows.spec.ts` (`@mobile`)

One test per special choice, each checking that the pending panel opens
for the right seat (`data-choice-host`, `data-decider`, `data-mine`) with
exactly the options the view offers (its `data-pick` tiles and the
`choices` keys), that the choice is made with a player's clicks, and that
it lands in the server's next view:

- the wonder draft of an ordinary new game (seed 4242): six picks by hand
  in the draft order (the first player, then the second twice, and the
  reverse in round 2), each pick on the draft screen and in the picker's
  wonders, and Age I beginning with four wonders each, all eight of them
  the ones offered;
- a progress token from the board (`science-pair`): the board untouched
  while the choice is open, every board token an option, the taken one
  moved to the player;
- the Great Library (`great-library`): the decider's view and panel show
  the three drawn tokens, Law first, with the "only you can see" note;
  after Law is taken, the next view (seat 1's) names the two returned
  tokens nowhere, and neither does the page. A hot-seat room has no other
  viewer, so the waiting side of this secret is covered in the browser by
  `online.full.spec.ts` (below) and over the protocol by
  `internal/protocoltest`;
- reviving from the discard pile (`mausoleum`), and the Mausoleum with an
  empty pile (`mausoleum-empty`), whose choice is skipped: the move toast
  (or, on mobile, the pass-device summary) and the log say why;
- destroying a card (`zeus-destroy` for a brown card, `circus-maximus` for
  a grey one), and the Statue of Zeus with nothing to destroy
  (`zeus-no-target`), skipped with the same explanation;
- the next age's starter (`age-transition-chooser`): the trailing seat
  chooses, not the one who took the last card, and the panel says why;
- the extra turn: lost on an age's last card (`extra-turn-lost`, with the
  log line and the starter choice that follows), and an ordinary one from
  the Appian Way (`appian-economy`): the same seat moves again, the turn
  bar reads "Sun plays again (Appian Way)" (`data-turn="playAgain"`), the
  log says so, and after that move the turn passes. The `theology-chain`
  flow builds Pyramids and Colossus consecutively: Theology keeps the same
  seat active after each wonder, with both builds and extra turns in the log.

Each test takes 3 to 8 s (the draft is the longest). On mobile the
pass-device screen is on, and the driver passes it wherever it appears.

Measured on the laptop with `make e2e E2E_BUILD=0 GREP='victory|flows'`
(both projects, 36 tests), green on two runs in a row, about two and a
half minutes each:

- desktop: `victory` about 26 s for its 7 tests, `flows` about 46 to 48 s
  for its 11;
- mobile: `victory` about 21 to 22 s, `flows` about 46 s.

### The frame leak check (`fixtures/leak-check.ts`)

`checkLeaks` takes every WebSocket frame each viewer of one game received,
in order, and proves that no frame named a card, wonder or token before
that viewer was entitled to see it. It is written from the rules of what
is hidden, not from the server's redaction code, and its ground truth is
what the game itself ends up revealing:

- at game over every view carries a `revealed` section (the cards set
  aside from each age, the deals of ages never laid out, the cards still
  face down, and the guilds, wonders and progress tokens left in the box;
  docs/rules/decisions.md, "Reveal at game end"). None of them may appear
  in any frame before the game is over, and every viewer's section must be
  the same;
- each face-down slot's card is learned when the slot turns face up (a
  `reveal` event, or the slot shown face up later, by any viewer) or from
  the final `revealed.faceDown`; no frame that shows the slot face down
  may name that card;
- an age's cards may not appear before that age is laid out, nor the
  second draft round's wonders during the first round;
- the Great Library's offer (the drawer's own `pending.tokens`) reaches no
  other viewer: their views show an empty `pending.tokens` and
  `pending.hiddenTokens` equal to the offer's size, the tokens put back in
  the box are named by nobody afterwards, the drawer included, and the
  kept one only once it is in the drawer's city;
- the game's seed, which no view carries (the caller reads it from the
  server's room file), appears in no frame, even at the end (checked for
  seeds of four digits or more, since smaller numbers are everywhere);
- and, with no ground truth at all, every slug a frame names must already
  be public to that viewer: shown in a public place of this frame or an
  earlier one (a face-up structure card, a city, a discard pile, a tucked
  card, a wonder, a token on the board or in a city, the draft offer, the
  open choice's options, the reveal).

Frames are parsed and compared value by value: each whole string, each
colon-separated segment of a string (an action key such as
`wonder:great-library:baths`) and each object key, never raw substrings,
since the card `library` is a substring of the wonder `great-library`. The
report lists each leak (viewer, frame, version, ply, id, rule) and what
was covered: frames per viewer, the secrets of the reveal by kind, the
face-down slots whose card was learned and how many (frame, slot) pairs
were checked against it, the cards known per age, and each library offer
with how many frames of the drawer and of the others it was checked in.

`specs/leak-check.spec.ts` tests the checker with no browser. Two protocol
clients and a spectator play a whole online game from the `great-library`
scenario (seed 7, so it uses no seed of the server's counter) with
`recordOnlineGame`. The checker must pass those frames as the server sent
them, with the library offer covered, and must catch a leak planted in a
copy of one of them: a face-down card's identity written into its slot, a
removed card in an event, the library offer in the other player's view, a
seed field and a later age's card. A unit check also shows that
`great-library` does not count as naming `library`. Its six tests take
about 4 s, most of it the recorded game (42 actions, about 45 frames per
viewer, 483 face-down checks).

### `online.full.spec.ts` (`@mobile @full`)

An online game between two browsers, played through the real UI on both
projects. Each player, and later the spectator, has a browser context of
their own in the one browser: their own storage (so their own identity
token and name) and their own network switch.

- Context A opens Home, Play online, types a name, picks a crest (the Sun
  on the desktop project, the Moon on the mobile one) and creates the
  game. In the lobby it presses Copy link and reads the clipboard, which
  must be the link the lobby shows and end in `/play/<code>` with the code
  the lobby shows in large type.
- Context B opens that link, answers the name prompt ("Play as …") and
  takes the open seat. Both lobbies give way to the draft; the room is
  `online` and `playing`, with both names on the right seats.
- The two pages play the game to the end with the online policy and the
  confirm step, each only on its own turn: before every move the other
  page (and the spectator) must have `canAct` false and no choices, and
  after it every other page must reach the new version.
- Once the draft is over (ply 10) a third context opens the same link,
  answers the name prompt and becomes a spectator: no seats, the turn bar
  says `spectating`, no `[data-act]` on the page, and both players see one
  spectator in their top bar. It then sees every move arrive, never with a
  choice.
- On B's turn after ply 18, B reloads: no name prompt, and the same seat,
  version, ply and choices come back on a second socket.
- On B's first turn in Age II, A goes offline (`context.setOffline`) for at
  least 4 s: A's turn bar reads `offline` (or `reconnecting`), its
  connection pill shows, `ready()` is false, B's page sees A's seat go
  away, and B moves meanwhile while A offers no enabled move. Back online,
  A reconnects, catches up with B's move and the pill goes; B sees A back.
- At the end all three pages show the same result, and it is the server's:
  the room file's result, and its action log is exactly the keys the two
  pages sent, one per ply. Each result screen names the victory and the
  winner from its own side ("You win", or the winner's name).
- The leak check runs over the three contexts' frames with the seed from
  the room file, and must find nothing, with the reveal present and every
  face-down slot's card known. The spectator's frames carry no choices.
- No console error, page error or failed request on any page, except A's
  failed reconnects while it is offline.

A second test opens an online room at the `great-library` scenario through
`/dev/new` (seed 7), with seat 1 and a spectator joining through its link.
Seat 0 builds the Great Library wonder first and keeps Law; meanwhile seat
1's and the spectator's views show `pickLibraryToken` with no tokens and
three hidden ones, their panels say what is being chosen without a token,
and no offered token shows on their pages. Before seat 0 keeps Law, its
connection drops for a moment: the panel's commit button is disabled and
reads "Waiting for connection", and once the connection is back the pick
takes the confirm step like any online move. One more move each follows, and
the leak check over the three streams finds nothing, with the offer
covered. So this secret is checked in the browser on every run, whatever
the first game's deal.

The spec prints its counts on one `online game:` line and attaches them as
`game-summary.json`. Measured in a full `make e2e`, the same games on two
runs in a row (the seed depends on how many rooms the specs before it
created, so a narrower `GREP` deals another game):

- desktop (A at the Sun): 71 plies, 33 moves by A and 38 by B, 66 of them
  through the confirm step; the spectator joined at ply 10, B reloaded at
  ply 18, A was offline about 4 s at ply 28 (the page read `offline`,
  opened 5 sockets in all, and caught up about 0.1 s after the network
  came back); seat 1 drew strategy, philosophy and urbanism from the Great
  Library; a military win for seat 1. Frames: A 80, B 78, the spectator
  67. The leak check made 730 face-down checks against 60 learned slots
  and covered 9 removed cards, 4 box guilds, 4 box wonders, 4 box tokens,
  the 4 second-round wonders and 20 cards per age. About 63 s.
- mobile (A at the Moon): 70 plies, 38 moves by A and 32 by B; spectator
  at ply 11, reload at ply 19, offline at ply 28; seat 1 drew mathematics,
  law and architecture; a civilian win for seat 1, 62 to 45. Frames: A
  79, B 77, the spectator 65; 719 face-down checks. About 60 s.

The Great Library test takes about 5 s (frames: seat 0 8, seat 1 7, the
spectator 6). A whole `make e2e` (both projects, every spec) took about 6.5
minutes after the builds, 73 tests passed and 3 skipped (the layout
viewports wider than 844 px on the mobile project).

## Production smoke (`make e2e-prod`, `prod.smoke.spec.ts`, `@prod`)

`e2e/specs/prod.smoke.spec.ts` is the only spec that is ever allowed to run
against a real deployment instead of the local test server. `make e2e-prod`
runs it against `PROD_URL` (default `https://ajayeb.lilf.ir`), builds
nothing and starts nothing local:

    make e2e-prod                               # PROD_URL defaults to the production site
    make e2e-prod PROD_URL=https://staging.example

It is the M8 acceptance check: "the Playwright `@smoke` run (create, join,
five moves) passes against production."

### How `playwright.config.ts` keeps everything else off a real site

Setting `E2E_BASE_URL` (which `make e2e-prod` does, to `PROD_URL`) switches
the whole config to production-smoke mode: `E2E_BASE_URL` becomes `baseURL`
in place of the local server's origin, no `webServer` is started (the site
named is already running, and the config must never start a local one in
its place), the timeouts grow to give a real network round trip room (10
minutes per test, 30 s per `expect`, a 40 s action timeout), and, the part
that actually keeps a local-only spec off a real site, **every project's
`grep` is forced to `@prod`**. `prod.smoke.spec.ts` is the only spec tagged
`@prod`, so it is the only one that can ever be selected this way, whatever
else is passed on the command line: a `--grep` (through `PW_ARGS`, since `e2e-prod` ignores `GREP`) is ANDed in
on top, never used in place of the forced one, so it can narrow the five
tests further but never add a spec that is not `@prod`. Without
`E2E_BASE_URL`, none of this changes; `make e2e` and `make e2e-smoke` run
exactly as before.

The forcing has to be written once at the top level and then **repeated on
every project that sets its own `grep`**, not just once at the top. Playwright
resolves a project's effective `grep` as "this project's own, if it set one,
else the top-level config's" (not as an AND of the two), so a project with
its own `grep` (here, `mobile`'s `/@mobile/`) would otherwise keep matching
every `@mobile` spec regardless of the top-level setting, and defeat the
forcing silently rather than with an error. (That is different from
`--grep` on the command line, which Playwright does AND on top of a
project's `grep`; that is how `make e2e-smoke`'s `--grep '@smoke'` already
narrows both projects at once, and why it was not enough on its own here: a
bare `playwright test` run with `E2E_BASE_URL` set, the way a human might
run it directly, passes no `--grep` at all.) So `playwright.config.ts` gives
the `mobile` project a conditional `grep` too: in production-smoke mode it
requires both `@prod` and `@mobile` (a single regex with two lookaheads,
since Playwright treats an *array* of patterns as OR, not AND). Since
`prod.smoke.spec.ts` carries only `@prod`, the `mobile` project matches none
of it by default, so a plain `make e2e-prod` runs the desktop project only;
tagging a test `@mobile` as well, once it is confirmed to also pass that
way, is what would let it run there too.

This was found the hard way while building this slice: the first attempt
forced only the top-level `grep`, which the `mobile` project's own
`/@mobile/` silently overrode, and a bare `make e2e-prod` ran all 40 local
specs (`@mobile`-tagged ones included) against the real site before it was
caught and killed a few tests in. Two extra, harmless `vs AI` games
(`ai.full.spec.ts`, which never calls a dev route) actually finished against
production before the interrupt, and a third spec got as far as navigating
to `/dev/new`, which the production build does not route, before the kill
landed. No dev-endpoint call ever reached the server (the production build
serves no `/dev/new` route to navigate to, so Playwright's `page.goto`
itself was the only thing that happened), and no corruption resulted, but it
is exactly the failure this section's forcing exists to prevent, and the fix
above (repeating the grep on every project, not just at the top) is what
actually closes it. `pnpm -C e2e exec playwright test --list` with
`E2E_BASE_URL` set is how this was verified afterwards, and is the fast,
network-free way to re-check it after any further change here: it must
print exactly the tests below, and nothing else, regardless of what other
flags are added.

### The four tests

1. `GET /health` answers `ok` with a non-empty `version` (logged).
2. Home renders in both themes (`light`, `dark`) with no console error, page
   error, failed request or HTTP error response, and no horizontal scroll.
3. Two isolated browser contexts play a short online game: context A creates
   it through the UI (Home, Play online, a name, a crest, Create), reads the
   invite the way a player would (Copy link, then the clipboard: see "The
   clipboard permission" below), and context B opens that link, answers the
   name prompt and takes the open seat. They then play at least five moves
   (the wonder-draft picks count) through the confirm step online rooms use,
   each only on its own turn (the other side's `canAct` and `choices()` are
   checked false first), and both pages are checked to agree on the room's
   `version` after every move. Both then leave through the game menu
   (Home), which just sends `leave`, not a delete; the room itself is left
   for the server's reaper.
4. A short game against the bot: Home, Play vs AI, Apprentice, Start, two
   moves with the bot answering, then leave through the game menu the same
   way.

Nothing here calls `/api/dev` or a `/dev` route: those answer 404 and are
turned off entirely on a production server (`-dev-endpoints` is refused
without `-dev`, and neither flag is part of a production run; see
[self-hosting.md](self-hosting.md)). Everything is driven the same way the
local full-playthrough specs are, through `GameDriver`
(`e2e/fixtures/driver.ts`) and the online helpers
(`e2e/fixtures/online.ts`), waiting on `window.__ajayeb.ready()` and room
versions, never a fixed sleep.

A full run creates at most two rooms on the real site (one online, one vs
AI); the Home tests create none. Run twice in a row (the acceptance check),
that is four rooms total, all left for the reaper.

### The clipboard permission

`newDeviceContext(browser, testInfo, { clipboard: true })`
(`e2e/fixtures/online.ts`) grants the `clipboard-read`/`clipboard-write`
permissions so `readInvite` can read back what Copy link put on the
clipboard, the way `online.full.spec.ts` already does locally. Granting a
permission is scoped to an origin, and that scope has to be the context's
real one (`testInfo.project.use.baseURL`, the same value `playwright.config.ts`
sets to `E2E_BASE_URL` in this mode) rather than the fixture file's own
`ORIGIN` constant (always the local test server's address): against a real
site the two differ, and the mismatch fails the clipboard read with
`NotAllowedError: ... Read permission denied` the first time it is used,
without the context ever using the permission it was actually granted. For
every local run `use.baseURL` already equals that same `ORIGIN`, so fixing
this changed nothing there.

### Measured

Run twice in a row against `https://ajayeb.pinky.lilf.ir` (`make e2e-prod`,
no local build), both green, `/health` reporting version `c5b9bfa` both
times:

- run 1: 5 passed in 29.7 s. The online game (room `36VWUY`) took its five
  moves from the wonder draft (A 3 picks, B 2); the vs-AI game (room
  `Z76NY6`) played its two moves with the Apprentice answering.
- run 2: 5 passed in 26.8 s. Online game `SWSPXU` (A 2 draft picks, B 3); vs
  AI game `V69WH9`.

Four rooms in all, every one left for the server's reaper. No stray Chrome
process afterwards (`ps -axo pid=,command= | grep -F -e ajayeb-chrome -e
playwright_chromiumdev_profile`).

## Hint and layout follow-up checks (M10.hints-followups)

`followups.spec.ts` checks board tokens opening TokenSheet with hint values,
a token pick showing both seats' values, an open token sheet closing when
another tab ends the game and this tab starts a rematch, the viewer's winning move showing
its review on the result screen, and a fixed seed's producer leaving a
trade-price caret and an emptied-slot trace until the next player's move.
It also checks a short own name beside the YOU mark, device link and owned
token, the inline final-board bar, a long skipped-choice toast in phone
landscape without strip overlap, and every mini card at 48 by 74 px.
Each scene saves phone and desktop screenshots in light and dark.

`layout.spec.ts` probes the actual hit span of compact strip controls,
including their transparent extended tap areas. It moves the pointer away
and waits for a delayed tooltip to close before testing Escape on a card
sheet: an open tooltip intentionally takes the first Escape itself. The
1024 by 768 regression also runs on the mobile project with touch.
`tooltips.spec.ts` checks the DismissLayer focus trap, Tab cycling and focus
restoration to the opener after Escape.

Unit checks cover the signed-number helper in English and Persian, the
new first-time tips, the recap board mappers and WonderSheet's 280 px art
sizes attribute.

### Small UI follow-ups (`small-ui.spec.ts`)

The spec checks that a Home tab receives a bot game’s level and age and
removes its Continue entry when another tab finishes it. It checks the
fixture link-state option and captures the who-starts bar, online presence,
gallery wonder names and score headers at 360x740, 390x844, 740x360 and
1280x800. The layout assertions check card overlap and clipped names.

### Review follow-up checks

`fx-followups.spec.ts` checks Age I after a draft and Age II after a bot’s
starter choice against the normal zero-delay test server. It requires a
deal to land, without slowing that server. `animation.spec.ts` also checks
that replay cues carry the replay marker while Sound is on and that the
replay changes neither state nor the final screen. Unit checks cover the
Sound-off gate, single vibration timing, cancelled haptics and stale deal
carry on interruption and resync.

`rtl.spec.ts` checks the card sheet’s card layout and the progress-token
row’s physical order as well as the board. `tooltip-coverage.spec.ts`
accepts adjacent visible labels only when they accompany a single image
branch; an unrelated glyph elsewhere in a labelled row cannot pass.
`tooltips.spec.ts` exercises the linked bubble’s narrow hit region alongside
the existing touch, long-press, keyboard and rules-link paths. Its card-sheet
regression uses real touchscreen and CDP touch events in English and Persian,
with Confirm moves on and off. It requires a held discard to show its diagram,
its release and a dismissing tap to send nothing, and the next short tap (plus
confirmation when enabled) to send exactly one discard and advance one version.
The discard group explicitly follows its button's enabled state for touch
classification because the tooltip anchor wraps the button rather than being
its descendant. The shared gesture rules and diagram placement stay unchanged.
