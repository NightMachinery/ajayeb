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

- `driver.ts`, the game driver of the hot-seat and vs-AI playthroughs
  (the online one will reuse it). `GameDriver` waits for this client's
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

  It closes a toast with its dismiss button before tapping the board, as a
  player would, and counts every flow, every pending kind, every pass
  screen and every undo in `driver.stats`.

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
  the keys.
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
- mobile: 67 actions (6 draft picks, 4 pending choices, 32 builds, 20
  discards, 2 wonders card first and 3 wonder first) and 58 pass-device
  screens, about 60 s.

The spec prints its counts on one `hot-seat game:` line and attaches them
as `game-summary.json`. A whole `make e2e` (both projects, every spec,
after the builds) takes about six and a half minutes.

### `ai.full.spec.ts` (`@mobile @full`)

Games against the bot through the real UI, on both projects. The bot's
turns are waited out with the driver's `waitTurn`, never with sleeps. The
config's `--bot-delay 0` makes the server answer at once, so the pacing a
player sees is the client's own: the thinking frame stays up for the
Presenter's dwell (300 ms under reduced motion, which the config sets),
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
- mobile (Moon): 72 plies, 36 actions by the player (37 sent, one undone)
  and 36 by the bot; 3 draft picks, 3 pending choices, 19 builds, 8
  discards, 2 wonders card first and 2 wonder first; the undo at ply 14;
  a civilian win for the bot, 49 to 46; about 55 s.

The Scholar test takes about 9 s on each project. The first test prints
its counts on one `vs-AI game:` line and attaches them as
`game-summary.json`.

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
over and no move is offered, and that the page logged no error. Each test
takes 2 to 4 s.

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
  viewer, so the waiting side of this secret is covered by
  `internal/protocoltest` until an online spec covers it in the browser;
- reviving from the discard pile (`mausoleum`), and the Mausoleum with an
  empty pile (`mausoleum-empty`), whose choice is skipped with a log line;
- destroying a card (`zeus-destroy` for a brown card, `circus-maximus` for
  a grey one), and the Statue of Zeus with nothing to destroy
  (`zeus-no-target`), skipped with a log line;
- the next age's starter (`age-transition-chooser`): the trailing seat
  chooses, not the one who took the last card, and the panel says why;
- the extra turn: lost on an age's last card (`extra-turn-lost`, with the
  log line and the starter choice that follows), and an ordinary one from
  the Appian Way (`appian-economy`): the same seat moves again, the log
  says so, and after that move the turn passes.

Each test takes 3 to 8 s (the draft is the longest). On mobile the
pass-device screen is on, and the driver passes it wherever it appears.

Measured on the laptop with `make e2e E2E_BUILD=0 GREP='victory|flows'`
(both projects, 36 tests), green on two runs in a row, about two and a
half minutes each:

- desktop: `victory` about 26 s for its 7 tests, `flows` about 46 to 48 s
  for its 11;
- mobile: `victory` about 21 to 22 s, `flows` about 46 s.

