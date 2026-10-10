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
  is checked in every run rather than only when a random game builds it;
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
  (welcomed, nothing queued in the Presenter, no request pending).

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
  the fonts have loaded, for screenshots.

A token written into `localStorage` for a tool (`ajayeb:token`) must have
the real shape, 22 base64url characters; the client replaces any other
value with a fresh token.

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
git). The config stops the server and the browser at the end of a run.

### The shared helpers (`e2e/fixtures/`)

- `driver.ts`, the game driver, which later milestones reuse for the vs-AI
  and online playthroughs. `GameDriver` waits for `window.__ajayeb.ready()`,
  reads a `Snapshot` (version, ply, phase, age, the seat to move, the
  pending choice, `choices()`, the result, and whether the pass-device
  screen is up), asks a `Policy` for a move, performs it by clicking what a
  player clicks, and waits for the room's version to pass the old one and
  the page to be ready again. It never sends a frame and never decides what
  is legal: every key it clicks is the `data-act` of one of the view's
  choices. The flows it knows:
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

  It closes a toast with its dismiss button before tapping the board, as a
  player would, and counts every flow, every pending kind and every pass
  screen in `driver.stats`.
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
  the keys.
- `layout-probes.ts`, the objective probes: horizontal overflow (the
  document's `scrollWidth`, and the visible part of every element, clipped
  by its scrolling ancestors, past either edge), page scroll, the size of
  the takeable structure cards and of the enabled primary controls (move
  buttons, panel and draft tiles, the top bar, the wonder flow, the result
  screen, the pass-device screen and every button of an open sheet).
- `hygiene.ts`: console errors, uncaught exceptions, failed requests and
  HTTP errors, collected for a spec to expect empty.

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
from seeds 1000, 1001 and so on, in order, so the games are the same on
every run of the same set of specs; in a full `make e2e` the desktop game
gets seed 1000 and the mobile game seed 1001. Measured on the laptop:

- desktop: 72 actions (6 draft picks, 6 pending choices covering all five
  kinds, 33 builds, 20 discards, 4 wonders card first and 3 wonder first),
  about 30 s;
- mobile: 71 actions (6 draft picks, 5 pending choices, 34 builds, 19
  discards, 3 wonders card first and 4 wonder first) and 58 pass-device
  screens, about 42 s.

The spec prints its counts on one `hot-seat game:` line and attaches them
as `game-summary.json`. A whole `make e2e` (both projects, the smoke spec
included, after the builds) takes about two and a half minutes.

### `layout.spec.ts` (`@mobile`)

The game screen at the UX plan's viewports: 360x740, 390x844, 430x932,
844x390, 740x360, 768x1024, 1024x768, 1280x800 and 1920x1080. The desktop
project checks all nine; the mobile project checks the six phone and
tablet ones (up to 844 wide) with touch, and skips the rest. A seeded game
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
