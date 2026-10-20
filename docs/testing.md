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
- resync history and live events checked against an independent replay through
  the received frame's ply. The outbox may merge a pending resync with later
  bot moves: its history is then a prefix and its live events supply the
  suffix. A deterministic outbox regression checks this interleaving and
  rejects missing, duplicated, reordered or corrupted history/events;
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
  `server/internal/engine/testdata/protocol/`. It preserves every JSON value
  in `web/src/lib/wire/fixtures.gen.ts`, pooling repeated view fields with
  their own type assertions and cloning each use. The complete arrays retain
  their `satisfies` assertions: `fixtures satisfies ServerMessage[]` and `protocolFixtures
satisfies View[]`;
- `pnpm -C web check` (`svelte-check`, part of `make check-web`) then fails
  the moment a real recorded shape no longer fits the generated types,
  rather than at runtime in a browser;
- `web/src/lib/wire/fixtures.test.ts` adds a few runtime checks on top: the
  arrays are non-empty, every recorded frame type is represented, and
  nothing in a real frame or view is a JSON `null` (the engine's own
  convention is an empty array or an absent key, never `null`; a nil Go
  slice marshaled without this convention is exactly the kind of mistake
  this check exists to catch). It also compares every generated frame and
  view to its JSON source and checks that snapshots share no mutable objects.

See [dev/protocol-fixtures.md](dev/protocol-fixtures.md) for compact storage
and regeneration instructions.

`ajayeb-gen` with no subcommand writes both the fixtures and the presentation catalog, so
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
  frames), so a spec can check the log after a reload;
- `tutorial()` and `coach()`: the tutorial state of the displayed frame and
  the visible coach step, or `null` outside their respective contexts.

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
on port 38954, with a fresh data directory under `.cache/e2e/`), in
Playwright's own Chromium (Chrome for Testing, `channel: 'chromium'`) both
locally and on CI, never the installed Google Chrome (D26). CI splits the
suite into two shards (`--shard=1/2` and `--shard=2/2`), each a job with its
own server, and keeps each shard's report and traces when it fails or runs
out of time. The shards leave out the `@perf` case, which runs in its own
job, `e2e-perf`, described under `animation.spec.ts` below. There are two projects: `desktop` (Desktop Chrome, a mouse) runs the standard specs, and `mobile`
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

### Opt-in WebKit smoke on macOS

The manual Safari/device check is **deferred-do-no-ask-user-again** by the
owner's decision. Do not request another Safari run. The local WebKit suite
adds engine coverage that the standard Chromium projects cannot provide.

```sh
make install-webkit  # explicit Playwright WebKit download, separate from install
make e2e-webkit      # build, then run the curated headless WebKit project
```

Use the usual detached guard wrapper when working unattended:

```sh
HEAVY_NOTIFY=auto python3 tools/heavy.py run --kind light -- make e2e-webkit
```

`E2E_BUILD=0` reuses a current test build. `GREP` and `PW_ARGS` can narrow the
curated run, for example `PW_ARGS=audio-engine.spec.ts`. The project is enabled
only by that target, `AJAYEB_WEBKIT_PROJECT=1` or an explicit `--project webkit`
(`--project=webkit` also works). It is absent from default project selection and
production smoke; explicit use with `E2E_BASE_URL` is refused. The local
browser download does not alter default `make install` and retains other
cached browser versions. No package script change is needed.

WebKit runs headless under the same `heavy.py --kind browser` lock as Chromium.
Its launch options contain no Chromium channel or switches. It never launches
the installed Safari app or `safaridriver`. The existing smoke, viewport layout
and Persian RTL specs run in WebKit, alongside `audio-engine.spec.ts`.

The shared console check ignores two specific browser notices only in WebKit:
the development server's report-only CSP missing a `report-to` directive, and
the unsupported `interactive-widget` viewport key. Fixed message prefixes and
the CSP notice's exact report-only explanation keep actual violations failing.
Every other console error, page exception, failed request and HTTP error stays
fatal. The smoke spec exercises the collector with browser-free event sources
to prove an unrelated WebKit error still fails and Chromium keeps both notices.

The audio spec observes native AudioContext creation, decode results and source
starts without replacing WebAudio behavior. It verifies no audio context,
sound metadata or file request before a gesture; a real first tap unlocks the
context and loads the hashed metadata and AAC files. Each run attaches the
observed path, AAC buffer playback or synthesized fallback, and decode results.
Forced missing-file and native decode-rejection cases prove fallback separately. Sound off prevents new
sources and loading, whereas Animation off removes animation without muting
sound. Both Animation settings are exercised through the real game controls.

The visibility case explicitly simulates `hidden`/`visible` page events and
suspends a real AudioContext, then checks that a subsequent tap resumes game
audio. This tests the app's recovery path, not real OS backgrounding: headless
Playwright pages behave as active pages. Source starts demonstrate a functioning
audio graph, not sound measured at the speakers.

[Playwright's WebKit](https://playwright.dev/docs/browsers#webkit) is a patched
upstream build, not the installed Safari version; macOS provides relevant codec
coverage, but other platforms can differ. This suite cannot stand in for iOS's
silent switch, call interruptions, device speaker routing or home-screen app
mode. Those manual checks remain deferred. See [Playwright's page model](https://playwright.dev/docs/pages#multiple-pages)
for the headless visibility limit.

The native audio spec also runs in Chromium, so its observer and assertions can
be verified when WebKit cannot launch. This does not count as a WebKit pass.
On macOS 14, Playwright can select frozen WebKit revision 2251 rather than
its current engine. The local macOS 14.3 run crashed with SIGBUS in the
Objective-C loader before any page or audio test could run; no WebKit app
behavior was validated there. Older host versions can fail in the loader before opening a
page ([upstream startup report](https://github.com/microsoft/playwright/issues/35896));
recent clients also have a documented mismatch with that frozen build
([upstream compatibility report](https://github.com/microsoft/playwright/issues/42964)).
Keep such a run failed and report the installed revision and host version in
private results. Do not mask it by skipping browser tests or substituting
Chromium for the WebKit target. It is not evidence of an app regression, and
it does not trigger another manual Safari request.

The separate Ubuntu `e2e-webkit` CI job installs the workspace, builds the
same test server and bundle, then installs WebKit with `--with-deps` and runs
only that project. It runs after Go/web checks on master pushes, pull requests
and manual dispatch, separately from Chromium, with a 20-minute job limit.
It uploads its JSON report, native audio attachments and failure traces even
on successful runs, and lists observed audio paths in the job summary.
Linux AAC decoding depends on the runner's GStreamer plugins. A decoder-less
run must prove actual synthesized sources start and report fallback coverage;
it is not counted as AAC decode coverage. Successful decoding must produce
native file-buffer playback. The initial Linux WebKit run passed all six native
audio cases: normal loads decoded all 17 shipped AAC cues and started native
file buffers; forced missing-file and invalid-AAC cases started synthesized
fallback sources. This validates Linux WebKit's audio graph, alongside the
separate macOS startup limitation and deferred iPhone checks described above.

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

  The move-summary dismissal spec compares the popup's box with every
  takeable card's box and chooses a card outside the popup for its first
  tap. This proves that a tap intended for the board closes the popup
  without opening the card sheet or sending a move. Only when every
  takeable card overlaps the popup does it use `force: true`; that fallback
  may tap the popup itself and checks dismissal without a move. The next
  tap always uses normal actionability checks and must open the sheet.

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
`military-supremacy` test also checks replay availability: the menu entry
is visible but disabled during play, and its focusable wrapper shows the
custom tooltip "Available when this game ends." The eligible finished
game enables that entry and shows a result-screen download button. Both
save byte-identical files named by the server as
`ajayeb-<code>-YYYY-MM-DD.json`, with the UTC download date. The decoded
`ajayeb-log/1` record has the room's format, seed, scenario, action count
and final hash. The spec also captures the running menu at 360x740,
390x844, 740x360 and 1280x800, checking that the entry can be scrolled
into view.

### `replay-download.spec.ts` (`@mobile`)

A fresh spectator, controlling neither seat, downloads an eligible
finished room's record from the result screen and compares the saved
bytes with the direct HTTP response. It checks the dated filename and
the enabled menu, with result-action and menu screenshots at the four
sizes above. A `hideAtEnd` room instead returns 403 without a seed, omits
the result download button, explains that the record stays hidden, and
keeps the menu entry disabled with that explanation in the custom tooltip.

The HTTP tests in `server/internal/httpapi/gamelog_test.go` separately
cover 409 for running games, 403 for hidden records and the other refusal
cases. `TestGameLogFinished` decodes the downloaded record, calls
`engine.Replay` and requires the same terminal state hash. The filename
test uses a clock in UTC+14 across a date boundary to check that the
server uses UTC. See [the format and access rules](protocol.md#downloading-a-games-log).

### `animation.spec.ts` (`@mobile`, two cases `@perf`)

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
rebuilding one (`mausoleum`), the last card of Age I, which deals Age II,
and the starter's answer (`age-transition-chooser`), two draft picks, and a
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
  change (its last card, whose frame deals the new age before the starter
  choice, and the first move of that age) and three bot answers that
  animated, that moment minus the frame's `swapAt` (`__ajayebFx.frames()`)
  must be at most 450 ms. The frame that opens Age II holds input until its
  deal ends, so that the starter choice's dialog opens over a dealt board
  (M11.age-start-order): that frame alone is measured from the end of its
  deal, every dealt card must have landed uncut, and input may not come
  back before it. Every other frame, Age I's deal included, counts from its
  swap. The
  bot's dwell and pick come before the swap and are not counted. The
  failure message lists every measurement. Every exercised bot reply must
  supply a timed animation sample. The case uses the configured retry policy;
  use `--retries=0` when measuring first-attempt acceptance.
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
  overall one for how loaded the machine was. It ends with the frame
  measure at 4x (see f). Every run also attaches `perf rooms`, with the
  room-specific task and animation records, including unprofiled failures.
  The case uses the configured retry policy. Repeated acceptance runs use
  `--retries=0 --repeat-each=10`; every first attempt must pass. On CI it runs in its own job,
  `e2e-perf`, once per project with no retries, and writes each run's
  `perf` annotation to the run summary. That job does not fail the
  workflow: GitHub's runners do not meet the 50 ms limit yet, which M10
  closed as an open criterion carried into M11 (see the roadmap).
- **e. Replay last move.** With Animation off the log sheet's control is
  disabled and its tooltip (keyboard focus on desktop, a tap on the
  phone) says to turn Animation on. At normal speed it is disabled before
  any move, then after the Arsenal's looting it replays: new animations
  are recorded, among them the looting, no move is sent (the version stays)
  and the screen afterwards is the screen before.

- **f. Frame gaps at full speed (`@perf`, information only).** Stutter at
  normal CPU speed is not the same thing as long tasks at 4x, so the run of
  d is played again unthrottled while `recordFrames` notes every
  `requestAnimationFrame` time. Its `perf` annotation gives the gaps between
  two frames that overlap an animation's window: how many, the share over
  34 ms (at least one frame missed at 60 Hz) and the worst gap. Case d
  reports the same at 4x. Neither has a threshold; CI's `e2e-perf` job
  writes both to the run summary.

With `AJAYEB_PERF_TRACE=1`, cases d and f also record a Chrome performance
trace (the timeline and JavaScript samples) and attach it with the rooms'
records, each room marked `perf-room:<name>` in the trace so its clock can
be matched. It is a diagnostic for finding what runs during an animation;
profiling slows the page, so a traced run's numbers are not a measurement.

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

Historical M6 measurements on the laptop with `make e2e E2E_BUILD=0 GREP='victory|flows'`
(both projects, 36 tests), green on two runs in a row, about two and a
half minutes each:

- desktop: `victory` about 26 s for its then 7 tests, `flows` about 46 to 48 s
  for its 11;
- mobile: `victory` about 21 to 22 s, `flows` about 46 s.

### The frame leak check (`fixtures/leak-check.ts`)

`checkLeaks` takes every WebSocket frame each viewer of one game received,
in order, and proves that no frame named a card, wonder or token before
that viewer was entitled to see it. It is written from the rules of what
is hidden, not from the server's redaction code, and its ground truth is
what the game itself ends up revealing:

- at game over every view of these games (created without `hideAtEnd`) carries a `revealed` section (the cards set
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

The existing `unexpected-reconnect` annotation records either player opening
more than one connection before B's intentional reload. Its socket-lifetime
attachment includes both players' bounded browser network events, and a
`connection diagnostics` attachment is written even when the full game fails
before that checkpoint. `newDeviceContext` captures the last 64 main-page
`document`, `online`, `offline` and `pagehide` events in Node across reloads.
Each record contains only its kind, epoch milliseconds, `navigator.onLine`
and `isTrusted` (`false` for `dispatchEvent`; browser-generated events,
including `context.setOffline`, are trusted). A document being destroyed may
lose its last report, so a missing `pagehide` does not prove it stayed open.
No URL, identity token, claim or outgoing frame enters this diagnostic record. The focused reload
test checks that event evidence survives reload and the strict baseline+1
socket assertion remains intact.

A paired reconnect does not by itself establish a server timeout. The client
has no post-open liveness timer: its 10-second watchdog skips an already-open
socket even if the main thread delayed its open callback. A browser `offline`
event deliberately closes an open socket with 1000, then retries after the
first 250 to 500 ms jitter, including on loopback where the socket may still
work. By comparison the server's hello and per-frame write deadlines are
10 seconds, and unanswered pings take about 75 seconds. Match the diagnostic
browser events and peer-close log lines before changing any timeout. A
`socket closed` status alone is incomplete evidence: the read-loop cleanup
can record normal closure after a read error, so inspect the preceding
`peer closed`, `reading failed` and `writing failed` lines too.

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
  Library; a military win for seat 1. Frames: A 80, B 78, the spectator 67. The leak check made 730 face-down checks against 60 learned slots
  and covered 9 removed cards, 4 box guilds, 4 box wonders, 4 box tokens,
  the 4 second-round wonders and 20 cards per age. About 63 s.
- mobile (A at the Moon): 70 plies, 38 moves by A and 32 by B; spectator
  at ply 11, reload at ply 19, offline at ply 28; seat 1 drew mathematics,
  law and architecture; a civilian win for seat 1, 62 to 45. Frames: A
  79, B 77, the spectator 65; 719 face-down checks. About 60 s.

These playthrough counts and timings were measured during M7. The suite
has grown since then; use the current Playwright report for its test count
and duration. For a bounded local run, select files with
`make e2e PW_ARGS='specs/online.full.spec.ts'`. After a successful build,
subsequent batches may use `E2E_BUILD=0` while those build inputs stay unchanged.

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
on top, never used in place of the forced one, so it can narrow the production
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
since Playwright treats an _array_ of patterns as OR, not AND). Since
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

### The production checks

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
once on the result screen (its row, and no review repeating it), and a fixed seed's producer leaving a
trade-price caret and an emptied-slot trace until the next player's move.
`gameover-last-move.spec.ts` (both projects) checks the result screen's final
move section (M11.gameover-last-move): with Animation normal every animation of
the final frame ends before the dialog opens and no move summary popup ever
shows. The ordering case runs at 1x, 4x and 6x CPU on both projects. It captures
the FX records immediately before native `showModal`, checks natural completion
(including the victory banner and landing pulses), and still compares every
end timestamp with the open timestamp. Polling for a visible dialog cannot
hide an early open. Its JSON attachment and `final-move-order` annotation
report the evidence. Unit tests also cover a never-ending manual run: the
all-kind safety limit finishes it with a warning after the paint gate opens,
while natural completion and interruption cancel that limit. The remaining
cases check that the viewer's own military win shows "Your final move" with
the Walls row and its pawn chip, and no review repeating that move (no better
move was there), through a reload, View final board and a rematch;
the bot's civilian win (the viewer in seat 1) and a hot-seat one name the mover;
and the section comes before the score sheet, above it on a phone held upright
and beside it where the `split` query matches. The mobile project's phone,
390x844 and the desktop at 1280x720 show everything without scrolling; at
360x740, 740x360 and 844x390 scrolling the body reaches the totals with the
footer's buttons in view. In phone landscape (740x360 and 844x390) the score
sheet stands beside the headline and the final move, and its totals show as
the dialog opens, with no scroll (M11.round3-layout-nits). After a viewport change the helper re-measures the same
scroll, visibility and 44 px button bounds until responsive transitions
settle, including the nominal 0.01 ms transitions with Animation off. The
native result-open ordering case still takes its snapshot immediately and
never polls to hide an early open. The describe is tagged `@mobile`, so it
runs on both projects.
It also checks a short own name beside the YOU mark, device link and owned
token, and every mini card at 48 by 74 px. Two checks are geometric, with one
helper that lists what an element covers by bounding box:

- The floating final-board bar keeps a headline wider than 60 px that is not
  cut short and a 44 px open button, and covers no chip, icon or label of
  either strip, at 360 by 740, 390 by 844, 844 by 390 and 740 by 360 px on
  the phone project and 1280 by 800 px on the desktop project. An
  inline-size container on the bar itself once squeezed it to a 22 px sliver
  that only the screenshots showed.
- A long skipped-choice toast in phone landscape covers neither strip, the
  turn bar nor any card the view offers to take, at 844 by 390 px and at 740
  by 360 px in English and in Persian: early in an age at the structure's
  head, and late in an age, when both players have discarded the lowest card
  until one near the head is offered and the Statue of Zeus then skips its
  destroy choice (`zeus-no-target`), in another lane. With that toast on
  screen the viewport then goes through 740x360, 844x390, 700x320, 740x360,
  360x740 (portrait) and back, and a per-frame sampler finds no landscape
  frame where it covers an accessible card, a strip or the turn bar, and the
  settled toast is back on screen in a clear lane.

Each scene saves phone and desktop screenshots in light and dark.

`layout.spec.ts` probes the actual hit span of compact strip controls,
including their transparent extended tap areas. It moves the pointer away
and waits for a delayed tooltip to close before testing Escape on a card
sheet: an open tooltip intentionally takes the first Escape itself. The
1024 by 768 regression also runs on the mobile project with touch.
On the desktop project it also opens a bot room at 1280x800 with a long
player name and checks that neither seat panel truncates its name, at rest
and with the selection's ghost tallies (M11.round3-layout-nits): the tallies
wrap to their own line instead.
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

`age-start-order.spec.ts` (both projects, M11.age-start-order) checks
that the start of Age II comes in order: the deal and its sound, then the
"who starts" choice. A MutationObserver notes when the age dialog, the
pass-device screen or the move summary first covers the board, and each
must come after every dealt card landed, uncut (`__ajayebFx.anims()`).
When the hot-seat player chooses, with Animation normal and off, the
dialog opens on the choice, the age sounds once on the frame that opened
it, and the answer deals and sounds nothing. When the bot chooses, the
dialog opens on its outcome. A reload during the choice replays nothing,
and its answer deals nothing. Two harder paths: a swap whose render blocks
the main thread for 1.4 s when the starter host mounts (the whole nominal
hold) still opens the dialog only after the deal; and online, with the move
summary on, the viewer's opening frame dropped at the socket (its request's
ack handed to the next frame) recovers the deal on the first play frame,
and the summary opens only after it, then the age dialog.

`rtl.spec.ts` checks the card sheet’s card layout and the progress-token
row’s physical order as well as the board. `tooltip-coverage.spec.ts`
accepts adjacent visible labels only when they accompany a single image
branch; an unrelated glyph elsewhere in a labelled row cannot pass. A whole-card
fallback cannot excuse a symbol without its own explanation, and the Age III
position must stay below 2500 DOM elements.
`tooltips.spec.ts` exercises the linked bubble’s narrow hit region alongside
the existing touch, long-press, keyboard and rules-link paths. Its card-sheet
regression uses real touchscreen and CDP touch events in English and Persian,
with Confirm moves on and off. It requires a held discard to show its diagram,
its release and a dismissing tap to send nothing, and the next short tap (plus
confirmation when enabled) to send exactly one discard and advance one version.
The discard group explicitly follows its button's enabled state for touch
classification because the tooltip anchor wraps the button rather than being
its descendant. The shared gesture rules and diagram placement stay unchanged.

`card-tooltip.spec.ts` checks full-card art hover and real touch long-press in
English and Persian, inner-symbol precedence, keyboard descriptions, both
server quote lines, explicit free Mausoleum rebuild payments, draft wonder
art/effect precedence and lazy city-card disclosures. The rebuild regression
covers the game fixture and the standalone gallery choice host. Gallery
readiness waits for fonts and quiet paint; it has no game-ready hook.
Real-touch covered-card art, an inner cost symbol and a face-down back retain
their short-tap inspection action, while long-press release only shows the
tooltip. A full back in a short phone sheet must fit without covering its card.
Focused client tests cover printed versus current costs, asymmetric
chain/free/shortfall wording, builder names and draft wonder summaries.
Placement tests keep a phone card clear whenever a vertical or lateral
placement fits. Engine tests compare the compact public quote index against
the solver at all protocol positions and prove that swapping a hidden card
with a removed card leaves the index identical. The ordinary redaction sweep
also scans this field for hidden identities.

### Linked chain tooltip checks

The card tooltip server tests compare the complete public index after swapping
an unseen linked target between a face-down slot and the age's removed cards.
They also check accessible and covered public targets, named city ownership,
discard, later/current unseen cards and absence after an ended age. Outgoing
quotes bypass only the chain alternative, preserving production, flexible
producers, fixed trade prices and Masonry. Reverse previews omit costs.

The chain tooltip browser cases use public fixed-scenario data for outgoing
city links and incoming structure links, including the chain affordability
badge. They cover English and Persian, real touch long-press and desktop hover,
with counting hints disabled, plus 360 px fit and card avoidance. The existing
Age III UX DOM budget still applies while city disclosures are closed.

## The tutorial (M10.tutorial-client)

Two specs walk the guided first game, both through the tutorial's DOM contract only (web/README.md, "The tutorial") and the test hook's `tutorial()` and `coach()`. They share one walk in `e2e/fixtures/tutorial.ts`, which reads the expected lesson order and guided keys from `GET /api/dev/tutorial`, so a changed script needs no change in the tests.

- **The walk policy.** While a coach step shows, press Next on an info lesson, otherwise press its target (a note with no target closes its sheet with Escape). While the opponent move summary is open, tap an empty part of the top bar (Escape when there is none: the centre of the bar may hold a tooltip anchor). Close a first-time tip with Got it and record its id. In the free part, open the first accessible card and press its first enabled move (build, else discard), else the first enabled move on the page. Answer a starter choice with `starter:0`. Stop at the completion card, or where the caller says. Every press waits 150 ms after it, a person's pace: two emulated taps inside the double-tap window can be taken as one gesture and close a sheet without acting.
- **The checks at every coach step** (`stepFailures`): a bubble with a ringed target (a bubble with none showed before its target was drawn); the full settled ring inside a sheet’s viewport and vertical clipping ancestors up to its top-layer dialog (outer ancestors do not clip the dialog); the target on screen (a region taller than the screen needs 85% of what fits shown) and, for an element target, the top element at its centre; the bubble inside the viewport, off an element target, over at most 40% of a region target, with its arrow towards the target; its buttons at least 44 px tall; no horizontal scroll; exactly one `[data-coach-target]`. A step gets a 600 ms grace to appear after a frame lands.
- **`tutorial.spec.ts`** (`@mobile`, both projects, about two minutes): the whole walk from Home through the real create, with the first-time tips and the move summary on, as a real player has them (the lesson ids and the guided keys sent in the script's order, no tip during the guided part, the pawn's and the new age's tips never, Play on my own absent on the `playOn` outro, then the completion card and its New game opening the vs-AI sheet on Home); the new age's tip in an ordinary game, held back while the age-start sheet is open and shown once the player continues; Play on my own and Resume tutorial; a different move leaving the path (`leftPath` once) and Undo back onto it; a reload mid-lesson; Home's No thanks and the Settings reset; Persian at 390x844 and 1280x800; Animation off (no animation over 10 ms) and Animation normal (the bubble waits for the board to settle, its target box final within 2 px).
- **`tutorial-viewports.spec.ts`** (`@mobile`, about five minutes): the walk until the last lesson at the ten viewports of the UX plan, with Animation off, the desktop project with a mouse at all ten and the mobile project with touch at the touch widths, and no console errors, page errors or failed requests (`watchHygiene`).
- **The device.** `freshDevice` seeds a new token, a name, the tips, the move summary, the Animation setting and the tutorial offer once per page; the base fixture declines the offer, so every other spec sees Home without its panel. `openTutorialRoom` opens a tutorial room at any position through `POST /api/dev/rooms` with `tutorial: true` and an `actions` prefix.
- **The probe.** `tools/chrome/tutorial-walk.mjs` is the same walk without Playwright, for the player reviewer: one screenshot per coach step and a `report.json` (tools/chrome/README.md). It needs a test build served with `--dev --dev-endpoints` and runs under the browser lock:

```sh
python3 tools/heavy.py run --kind browser -- node tools/chrome/tutorial-walk.mjs \
  --origin http://127.0.0.1:38951 --vp 360x740@2 --out /tmp/walk
```

## Text contrast regressions

`make e2e PW_ARGS='specs/contrast.spec.ts'` checks the
warning and error text, discard filter counts, compact tutorial progress,
and age-summary seat names with axe in both themes. It uses phone layouts
for the compact text and desktop layouts for the danger chip and offline
caption. Only colour contrast is checked here, so unrelated accessibility
findings keep their own regression coverage. The screenshot matrix below
adds all four sizes and the Persian captures. Disabled labels are exempt
from axe's contrast rule, so the focused spec measures their painted text
and background colours separately for primary, secondary and danger
buttons. Those labels are also inspected in the captures.

The disabled-button checks run on both browser projects in both themes. They
cover the tutorial's locked Discard, an unaffordable Build, both progress-token
pickers, Join and an offline destructive choice. Each disabled button must use
the neutral sunken panel and muted ink, a dashed rule border and full opacity,
with a measured text contrast of at least 4.5:1. Accent-soft remains the
selection treatment, so unavailable actions do not look selected.

## UX screenshot matrix and probes

Run `make shots` to build the test client and Go binary, start the test
server, and generate the opt-in Playwright `ux` project. It captures 38
screens at 360x740, 390x844, 740x360 and 1280x800 in English, light and
dark, plus Persian at 390x844 and 1280x800 in light. The three phone
sizes emulate touch and a mobile viewport, including landscape. Every
capture uses Animation off, a fresh context, loaded fonts and decoded visible
images, and a device scale factor of one so its PNG pixels match CSS pixels.

The ordinary `make e2e` and CI commands do not configure the `ux` project.
It is enabled only when the invocation explicitly requests `--project ux`
(or `--project=ux`); that choice is passed to Playwright's worker processes
in `AJAYEB_UX_PROJECT`. Do not export that internal flag in the shell.
Production-smoke mode never enables it.

Output goes to the gitignored `e2e/ux-out/` by default. Set `SHOTS_DIR`
to a different directory (an absolute path avoids ambiguity):

```sh
SHOTS_DIR=/tmp/ajayeb-shots-before make shots
SHOTS_DIR=/tmp/ajayeb-shots-after make shots PW_ARGS='--grep board-age2'
```

The filter matches stable screen ids in test names; add the viewport,
theme or locale to narrow further, for example
`PW_ARGS='--grep "390x844 light en.*board-age2"'`. `E2E_BUILD=0`
skips rebuilding when the existing test build is current. Each invocation
replaces its report and known matrix PNGs in its output directory, so use
separate directories to preserve before and after evidence. Never point
`SHOTS_DIR` at a directory holding other screenshot runs you need to keep.

Open `index.html` directly from disk. Its contact sheet groups the captures
by screen, links each image at full size, and lists that capture's findings.
Each PNG is named `<screen>__<viewport>__<theme>__<locale>.png`.
`probes.json` contains one record per capture and the run's budgets;
`summary.md` lists unreachable screens, budget misses and probe findings,
with severe findings first. P0 marks critical accessibility errors and uncaught
exceptions; P1 marks serious accessibility errors, hygiene, scroll, covered
actions and budget misses; P2 marks overflow and targets below 24 px; P3 marks
targets below the recommended 44 px. Serious and critical axe findings use the same
WCAG tags as the smoke spec. Geometry and tap measurements share
`tools/chrome/probe.mjs` with `shot.mjs --probe`.

Each capture measures horizontal scroll, painted elements crossing a viewport
edge using the root layout viewport (`clientWidth` and `clientHeight`), the smallest enabled control and counts below 24 and 44 CSS pixels,
and covered enabled `button[data-act]` centres. Hidden, modal-inert and
clipped content is excluded; vertically scrolled content wholly outside the
viewport is not an overflow finding. Console errors, uncaught exceptions,
HTTP errors and failed requests are recorded. Normal cancelled navigation
requests are excluded. Findings do not fail the project. A missing state,
wrong pending choice, wrong ending or missing distinguishing UI does fail it.
The probes are evidence for a reviewer, not a substitute for playing the game.

Most screens come from the existing recorded `/dev/fixture` states.
The blue-card tie, tutorial steps and pass-device screen use seeded dev rooms
and the existing scenario/tutorial helpers. `discard-pile` opens the existing
DiscardSheet specimen in `/dev/gallery`: the current app has no live-game
opener for that component. The modal is the actual component, with the gallery
inert behind it. Its axe scan is scoped to the open discard dialog to exclude
unrelated gallery demonstrations; geometry and hygiene still cover the page.
Every other axe scan covers the document. Capture metadata records the source
route and axe scope. This provenance is a known review-loop finding. Tutorial
outro captures the first outro lesson, `tokens`, after all scripted actions.

The `run budgets` test runs once in a full invocation. It sums gzipped bodies
of the scripts loaded by a cold Home load (200 KB, or 200,000 bytes, limit), measures the full
Age III board after the fixed-seed recorded game's first two ages (2500 DOM
nodes at 1280x800 and 390x844), and takes the median of three cold Home LCP
loads at 390x844 (2.5 seconds). LCP uses CDP: 4x CPU slowdown, 1.6 Mbps
download, 750 Kbps upload, 150 ms latency, cache disabled and a fresh context
per load. Fonts and images must finish within bounded waits; all measured
samples and profile parameters are in the report. Timing varies by machine.
A screen-only grep skips budgets for fast reshoots; include `run budgets` in
the grep, or run the full command, when budget evidence is needed.

The capture settles running transitions before probing, including the open
dialog's entry opacity. Settings, Join and tutorial sheets also wait for their
distinguishing controls. Incidental pointer hover is cleared; tutorial target
focus and the two intentional open tooltips are preserved. Tap probes measure
text inputs themselves and use labels for radio, checkbox and visually hidden
inputs. Hidden lazy images in closed sheets do not delay a capture.

The shared probe retains `iw` and `ih` as `innerWidth` and `innerHeight` for
CLI compatibility. Its explicit `viewportWidth` and `viewportHeight` drive
scroll, bounds and covered-action checks: mobile Chrome can expand
`innerWidth` when content overflows, which would otherwise hide the problem.

## Accessible control regressions

`aria-targets.spec.ts` checks the city bands' named group, the waiting wonder
draft's tab stop and keyboard scrolling in short landscape, and the actual
44 px boxes of city, wonder and device-link buttons, segmented options and
cost disclosures. It clicks the After this pick summary's padding and toggles
it with Enter and Space, then reloads to check the saved preference. This
catches a visually large box with a small active line and duplicate toggles
from competing native and reactive updates.
It also checks the unchecked checkbox and radio surfaces in both themes and
keeps their native Space and arrow-key selection behaviour covered.

The dense 26 px progress-token tooltip buttons in short landscape remain
information controls with long-press access. Inline Rules and Home links
remain text links. The UX probes continue to count these under 44 px; neither
case is hidden from the report.

### Persian sentence direction

`rtl.spec.ts` measures the browser's computed direction and character positions
for Persian pass-device log sentences and starter options beginning with Latin
player names. The name appears to the right of the next Persian word, while its
own letters read left to right. Card geometry remains left to right and its
translated name, category and card-back ordinal follow the locale. Diagram labels inside spatial
rows also follow the locale without reversing those rows. Bare names and name fields keep automatic
direction. A MutationObserver records the ordinary victory caption while it
is in the spatial FX layer, checking locale direction without adding an
animation timing assertion. The spec also checks single-line lobby sharing labels, 44 px button
heights and no horizontal overflow at the four review sizes. Longer card action
phrases may wrap to preserve readable text and action spacing.

## Returning-player flow regressions (M11.player-flow)

`player-flow.spec.ts` (`@mobile`) checks an online opponent move received while
offline, then the reconnect summary, a reload without replaying seen moves,
hidden incoming frames followed by reload, and a visibility wake without a new
frame. Hidden-but-connected reception uses the document visibility event so it
can test that received frames do not advance the seen marker. The same spec
checks that a first-time tip lets an intended touch card tap open its sheet,
that the new-age tip is absent on an Age III resume (unit tests also cover late
Age II), and that Escape returns native-sheet focus without a tooltip taking the
next touch. A deliberate Tab
focus still opens a keyboard tooltip.

`move-summary.spec.ts` keeps the summary's tap-swallowing contract and checks
that a hot-seat pass screen replaces the automatic popup while the eye button
still opens it. On a fixture page it also checks a card picture's tooltip: the
board's card detail (the printed cost line and the card's face in the stack) on
a hover, and on touch after a long-press that keeps the popup, whose next tap
closes only the tooltip. On the gallery's extra-turn scene it checks that the
Piraeus picture's tooltip has the wonder's effect and printed cost. In phone
landscape (844x390 and 740x360, a real bot game) it plays until two popups have
followed a move that made a new card takeable, and checks that every popup on
the way covers no card to take, neither strip and not the turn bar, in a lane
other than `none` (M11.round3-board-fixes). It also replays round 3's
Apprentice game (seed 2001, its first 13 actions) at both sizes and builds
Baths, Clay Reserve and Palisade, after which the bot's popup is too tall for
any lane: each popup covers nothing, shows its title whole and its first line begun,
and, when it scrolls inside, carries `data-scroll-more` (the fade) until its
foot is reached (M11.recheck-polish). `undo.spec.ts` checks the bot undo's visible confirmation in the
toast's polite live region. Unit tests cover the bounded, lenient seen-marker
storage, seat and room separation, blocked storage and undo rewinding markers.

## Sheet layout regressions

`sheet-layout.spec.ts` checks card actions before and after keyboard scrolling
in short landscape, armed confirmation labels, and entering and leaving the
wonder step. The existing `hint-outcome.spec.ts` also checks the action row
and usable detail height with an armed blunder warning in landscape.
The pinned tutorial action also keeps its coach note visible in the scrolling
detail body.
Choice footer hit tests sample the panel's bottom strip at three
scroll positions; the last option remains focusable and selectable. Phone
checks compare coach bubbles with the measured top bar and hint pips with
neighboring chip borders. A scrolled card detail body must scroll upward
before a downward drag can dismiss its sheet.

### Home loading and prerendered paint

`home-loading.spec.ts` runs on both desktop and mobile. It checks Home with
JavaScript disabled, including the visible priority hero and its responsive
preload, stylesheet cascade order before fonts/hero, and low-priority eager
module preloads, plus the Home-only critical stylesheet HTTP preload. Go tests
cover its generated URL selection, content-bound invalidation and route isolation.
The browser spec enforces the normal
Go CSP while a hero fails before hydration,
and confirms game, Settings and Rules responses contain the generic
shell rather than Home. With scripts enabled it checks that the cold English
load contains no Persian message table and opens no WebSocket; lazy sheets
retain their choices and restore focus; real AI, hot-seat, online, join and
query-driven AI starts work; and Continue reuses the live session.

The early-activation cases hold every immutable JavaScript response for at
least 1.5 seconds and until the pre-hydration assertions finish. They inspect
Home immediately after its heading paints: all mode, tutorial, join and
Settings buttons are natively disabled, the theme radios are disabled, and
“Getting ready…” is visible. Pointer presses during this state cannot silently
consume an enabled action. After release, each action opens its sheet or starts
the tutorial; mobile uses touch, Settings uses Enter, and the theme case checks
both pointer selection and arrow-key selection. A held-start layout case captures
all four review sizes and checks that the status stays on screen without covering
any button, radio or link. The deliberately chosen
tradeoff is visible unavailability until mount rather than queued activation.
No new inline script or CSP permission is needed. The no-script case keeps
Home readable with the same unavailable state; regular links still work.
A separate case starts the production-style online click immediately after the
heading and releases scripts only after the click has had a chance to land.
Production helpers retain their immediate heading-then-click behavior.

A returning-device case combines stored Continue details, a completed and
declined tutorial, Persian, dark theme and Animation off. It records hydration
warnings and any frame in which the first-launch tutorial offer is visible (the
build-time Home carries it, hidden on such a device). Another case
holds the Persian network import while switching back to English, then checks
a successful switch and reload. Locale unit tests also cover shared pending
loads and failures that preserve the working language. Image lifecycle unit
tests cover cached successes, pre-hydration failures, ordinary lazy events
and cancellation of listeners and queued work when a source changes. `rtl.spec.ts` requires
the first RTL document to already contain translated text, rather than relying
on a language bootstrap before the messages arrive.

Two cold-load cases guard Home against layout shift (M11.home-cold-cls). Each
makes three visits from a new device, in fresh contexts with the service worker
blocked, the cache disabled, 4x CPU slowdown and a 1.6 Mbps down, 750 Kbps up,
150 ms network, one in English with the light theme and one in Persian with the
dark theme. Each visit sums the layout-shift entries until Home is ready, the
language has switched and the fonts have loaded, plus 1.5 seconds; the median
of the three must stay under 0.1. The visits must also show the tutorial offer,
so the case keeps covering the element that once moved the page. A Persian case
holds the Persian messages and checks that a “Getting ready…” status stays
readable while Home's text waits hidden, including after the app has started,
and goes once the language switches. The unit test `web/src/app-bootstrap.test.ts`
runs `app.html`'s inline bootstrap against the app's own storage parsing, so its
copy of the tutorial offer test cannot drift.

When changing rendering or fallback filenames, also run `shell.spec.ts` for
worker installation and real offline Home and game deep links. The Go static
handler tests keep prerendered Home separate from the generic shell, preserve
legacy builds and retain missing-asset 404 behavior. Use the unchanged UX budget
test for timing evidence: it reports measurements without asserting its limits,
so inspect the three LCP samples, their median and the script-byte breakdown.

## Completed ages and result actions

`age-summary.spec.ts` runs on desktop and mobile. Real bot games with Animation
normal and off prove that an opponent's starter decision cannot close the
completed-age summary or replace its server-supplied totals. With the move
summary on (M11.round3-board-fixes), in both Animation modes: with seed 1 the
bot chooses itself and takes the first card of Age II, and the recap opens
first with Continue, then the summary with the new age's line, the starter
choice and that move; with seed 7 the bot chooses the viewer, and a reload
while the recap holds the summary back still opens that summary, with no deal.
`age-start-order.spec.ts`'s skipped-opener case checks the same recap-first
order when the opening snapshot never reached the viewer. Two real online
browsers cover closing before and after the opponent chooses, an opponent's
first card, presence-only updates, reconnect/resync, and the final move shown in
the result dialog itself, with no summary popup. Hidden boundaries and results wait for wake and the
unseen opponent summary; a dialog already open stays open when hidden. Reconnect
cases drop only the real server WebSocket while relaying its unchanged messages,
so intentional disconnects do not interrupt unrelated idle art requests. Recorded-transport cases queue both ended ages, verify Continue-only older recaps, expose current starter controls only after their matching recap is reached, and keep each recap stable through result/presence frames. A focused transition also checks that
game over leaves an unread age open with Continue and cannot stack its result
dialog above it. The summary is retained across ordinary snapshots,
not reconstructed from a historical score after a full page reload.

The full online playthrough leaves the spectator's first recap unread while
later ages advance, checks its frozen totals, then explicitly reads every
reached age before the result. Recorded cases always reach both boundaries.
Complete hot-seat/AI/online playthroughs exercise the driver's continuation
loop, including a handoff released by Continue. The AI undo checks that the
visible turn state is restored exactly, including an existing wonder replay.

Hot-seat cases run with passing enabled and disabled. With passing enabled,
opponent summaries are on: the pass screen shows the last move before the
next holder's age dialog, without a duplicate popup. The chooser's original
dialog stays mounted when choosing the other starter, until explicit continuation
releases the handoff. Its summary includes both the final card and starter choice
from the existing opponent run. Opening and dismissal are scoped to each holder;
returning the device does not reopen a summary that holder explicitly closed. Automatic handoffs do not reopen an older opponent run after undo. The cases also cover undo into the starter
choice, Escape and backdrop dismissal, replacement-button focus, quiet focus
return and room leave. Native close can temporarily retain focus inside the
closed dialog before dropping it to the page body. Age and result callbacks
recognize that closed-dialog focus too, preserving a live opener and quietly
falling back to the game menu otherwise. Explicit Escape assertions cover both
the outstanding starter choice and the result dialog. The existing driver acknowledges completed recaps when advancing to the next playable turn or finished result; it never dismisses a pending starter choice.

The footer cases cover 360x740, 390x844, 740x360 and 1280x800, both themes and
English/Persian, for starter choice, completed age and civilian game over.
They require every reserved action to remain hit-testable at a minimum of
44 by 44 CSS pixels before and after detail scrolling, with a usable separate
detail body and no horizontal page overflow. The UX matrix's end-age-summary
screen now captures the summary as it opens automatically.

Unit tests for `ui/age-summary.svelte.ts` cover a burst of frames before
presentation, frozen totals and starter identity, dismissal across duplicate,
presence and resync frames, undo within and across the boundary, room isolation,
an unread age surviving game over and separate hot-seat holders. These checks exercise local
presentation state; server legality and cost calculation remain unchanged.

Card-sheet clarity checks cover English and Persian on both browser projects.
At 740x360 the single cost breakdown and shortfall must fit beside the face
above the action footer. Keyboard End and PageUp verify that the detail fade
tracks remaining scroll content. At 1280x800 an expanded After this pick line
must fit or have a scroll-edge cue. A real tutorial room advances past the final
guided lesson into Age I free play, asserts that no coach is showing, and opens the
Workshop to verify that an unbuilt Mausoleum does not introduce advice yet.
Focused hint tests retain ordinary discard advice when Mausoleum explanations
are hidden. Conditional sentences remain in each locale's message table.

The room screenshot barrier retains its font-ready promise only after all registered font faces settle. New faces, pending loads and locale changes recheck font readiness. A later load clears the current stamp, and a stale presented-frame effect cancels its pending animation frame. This barrier is separate from the input-ready hook and the animation performance windows.

The animation suite also checks focus after confirmed card removal at normal
speed and with Animation off. It opens and reopens the card sheet with Enter,
checks Escape and backdrop dismissal restore its opener, injects a rejected
action at the transport boundary, and checks that only a confirmed removal
moves focus to the named game region. Tab must still reach a board control.
Native focus/removal probes are diagnostic evidence, separate from unprofiled
performance acceptance.

## Screenshot tool argument checks

`python3 tools/heavy.py run --kind light -- node --test tools/chrome/cli.test.mjs`
checks that invalid `shot.mjs` arguments fail before launching Chromium,
print a short message and exit non-zero. `AJAYEB_STACK=1` enables diagnostic
stacks. `make test-tools` runs these checks, and `make verify-quick` includes
them. The mini card specimen lives in `tools/art/vector/specimen.mjs`;
its chain chips are at least 14 px, with 17% cost and 23% effect bands.

## Extra-turn reload

The Appian Way flow in `e2e/specs/flows.spec.ts` first reloads during an extra
turn with its full history, preserving the wonder name in the turn bar. It
then reloads with reconnect event history stripped. Its real server view must
still show `playingAgain`, and the turn bar must still say Play again.
The session unit test separately checks that a new session keeps that
field even when it has never received the granting event. The pure turn
mapper also tests the server-view fallback directly and prefers the named
wonder from history when that history is available.

## Original icon paint and modal cost

`icons.spec.ts` compares every shipped icon with its original sprite rendering
in light and dark themes, using inherited ink and explicit two-tone paints.
It uses 48 CSS pixels (a multiple of the source grids), fixed cells and a pixel
scale of one for exact image equality. Ordinary tooltip and animation specs
retain each project's device scale and input settings. Local clipping references
and unique ids are checked too. The source unit tests cover the raw budget,
paint id scoping, and the limits of fill-only group removal.

The client draws original icon shapes directly, avoiding a shared sprite and
SVG use shadow trees during native modal style recalculation. FX copies restore
their own clipping definitions after interaction ids are stripped. The existing
Age III DOM limit remains below 2,500 elements; neither the long-task threshold
nor the measured animation windows change. See
[the attribution](dev/master-e2e-failures.md) for the rejected alternatives.
