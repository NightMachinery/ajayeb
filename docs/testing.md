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

The catalog's three-way data check, `TestVerify` in
`server/internal/engine/catalog/verify_test.go`, joins the catalog's own CSV
tables with two blind verifier tables and compares every field (the rulings it
led to are in [rules/data-rulings.md](rules/data-rulings.md)). Its mechanics
live in the test-only helper `server/internal/enginekit/catalogdiff`, shared
with Haftgane's catalog: each game supplies its verifiers, its tables with the
join key of each (Duel's is the name; a key may also combine columns, such as
a name and an Age), its fixtures and its row counts. Only `_test.go` files may
import the helper, which its own `TestOnlyTestsImport` checks.

### Purity and fixed-value guards

`internal/enginekit/puritycheck` is a test helper shared by the rules engines.
It reads and parses every non-test Go file in configured package directories,
recursing through subpackages and excluding `testdata`. Parse and file errors
fail closed. Each game supplies import rules, DTO map exceptions and the
checked root types. Duel and Haftgane forbid both math/rand versions in every local file;
only `enginekit/rng/rng.go` may import math/rand/v2.
Duel's `TestEngineImports` keeps its scan-count minimums locally, including the
catalog minimum, so a broken source walk cannot silently pass. The shared
checker always rejects imports of itself from scanned production sources.
It performs I/O and reflection and belongs only in engine test imports.
The depguard rules enforce that same shared-file exception.

`TestPurityCheckCatches` retains the parser negatives, including the rejection
of v2 even in Duel's local rng.go. `enginekit/rng` owns the exact permitted-file
case and checks the rest of its package for I/O, maps and floating-point code.
`purity_regression_test.go` adds a negative for every existing import,
map, numeric and parse rule, plus forbidden fixed-value kinds and nested
array/struct references. Those fixtures were run against the unchanged master
guards before extraction and again through the shared helper afterwards.
The reflection walker accepts only bools, fixed-width integers, arrays and
structs. Duel keeps its State/Action/Event roots, comparability checks and
State's 512-byte memory limit in `state_test.go`; the helper sets no game size
limit. The original state remains 386 bytes encoded and 408 bytes in memory.

### Module imports and test-only registry replacement

The shared purity checker can restrict module-internal imports by source
directory. It accepts exact package paths; the most specific directory policy
wins, and an unconfigured directory admits no module import when the module
boundary is enabled. Existing I/O, randomness and numeric bans still apply.
Duel permits its current catalog, DTO and RNG dependencies. Haftgane's catalog
permits only `enginekit/rng`; its rules additionally permit its own catalog and
view packages. DTOs and the shared RNG have no module-internal dependencies.
Test helpers keep their existing explicit dependencies. These policies never
admit all of `enginekit` or a similarly named package by prefix.

Strict depguard rules enforce the same production boundaries while permitting
the standard library. Forbidigo resolves import aliases and rejects references
to either catalog's `ReplaceForTest` outside `_test.go` files, including command
packages. The test exemption applies only to that diagnostic; existing print
checks still apply in tests, and command print exemptions do not admit catalog
replacement. Registry validation and test replacement behaviour stay unchanged.

### Seeded generator and byte digests

`internal/enginekit/rng` owns the PCG reference, rejection-sampling, copy/fork,
marshal and permutation tests moved from Duel. `TestRNGGolden` runs in both
packages with identical pinned draws and shuffle values; Duel's alias/wrapper
test also pins rules version 1 and catalog hash `64943c7be6864ff4`. Shared
`TestDigestGolden` pins FNV-1a vectors and 16-digit lower-case hex formatting.
Duel's setup, replay, state-hash, catalog, protocol, view and store fixtures
remain unchanged. Which bytes enter a hash is each game's own contract.

## Fuzzing

The module's fuzz targets include:

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
  "Hints");
- `FuzzTableOps` (`server/internal/tableroom/fuzz_test.go`): fuzzed
  operations on one Haftgane table with the record, replay, revision and
  restart invariants checked after each (see "`internal/tableroom`" above).

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

## `internal/tableroom`: the table actor

Haftgane's table actor is tested through `tabletest.Table`, the driver's
contract (`internal/tabletest`), by two adapters. The in-process adapter
(`harness_test.go`) joins real `room.Client`s through the Manager, submits
on the actor's goroutine and keeps every frame it pushed. The WebSocket
adapter (`ws_test.go`, the external test package) drives the same tables
over the real stack, wired as `cmd/ajayeb` wires it: the schema-2 store,
the Manager with every kind (Duel, the table and the lobby), `ws` and
`httpapi` behind an `httptest` server. It sees only frames: a snapshot is the latest frame after a resync,
an outcome is read from the reject or the acknowledging frame, and the
canonical history is the `ajayeb-log/2` download.

- `TestDriverInProcess` and `TestWebSocketTables` pass `Play` and
  `CompareSchedules` unchanged at 3 to 7 seats, with drafts, resends and
  races; every arrival seed gives the same canonical history.
- `TestEveryArrivalOrderInProcess` runs `ExhaustPoint` on small points (90
  and 2,520 orders, a one-seat salvage point, seq memories of 32 and 1, each
  seat submitting from two tabs) and
  the races of a change or withdrawn confirmation against the last
  confirmation (1,680 orders); `TestWebSocketEveryArrivalOrder` runs the
  90 orders over the socket.
- `TestDifferentialAgainstDirect` delivers the same seeded scripts, from
  two tabs per seat, to the reference model (`tabletest/direct`) and the
  actor, arrival by arrival, at 3 to 7 seats with seq memories of 32 and of
  one client per user: every outcome agrees, after every arrival every
  viewer's snapshot agrees (each seat's two tabs and the spectator: the
  point, required and confirmed seats, offer, pick and confirmation), the
  final canonical bytes agree, and duplicates and stale seqs both occur.
  Each game has its own deadline (each seat count in
  `TestDriverInProcess`), so the sweep, several times slower under the race
  detector, never shares one budget, and a hung game still fails.
  `TestDifferentialSwitches` does the same around seat switches, which the
  model plays through `direct.Switch`: the spectator takes a seat at one
  point, and at the next a seat goes to a bot halfway through the arrivals
  (confirming a kept pick at least once) and back to its human after them;
  the former controllers' picks are forbidden in both, and the switches'
  own outcomes agree.
- `actor_test.go` covers the refusal order, private edits that change no
  other connection's frame (revision included), two tabs and forgotten seqs
  (the pick's floor answers a forgotten seq as a duplicate or stale, seq 0
  included, before and after a restart), a seq memory per user (other
  users' drafts from many tabs change neither a user's duplicate answer nor
  their frames), tab churn (past a pick's floor bound a new tab is refused
  with `rate` before anything is written, every accepted record restores,
  a restart loses nothing), the shared write and restore bounds (a
  candidate past them is refused before it is saved),
  a reload that restores a pick and its confirmation, storage errors and
  crashes before and after the rename (`tabletest/faults`), the restore of
  every pending subset (and an invalid pending block archiving as
  `replay-failed`), and the compatibility paths: hash refresh keeping
  unknown fields, rules-version archive, unknown schema or game untouched.
- `switch_test.go` covers seat switches before and after a pick and its
  confirmation (a bot taking the seat confirms the kept pick or starts
  work; a seated spectator may still change it; the human replaced
  watches), undo turned off for good when a second human sits, roles under
  an injected clock (promotion and demotion limits, the temporary admin
  made after five minutes away, kept across a restart, passed on and
  demoted when a real admin returns) and resignation.
  `rolewrites_test.go` refuses the write of each temporary-admin change and
  checks that the temporary role fails closed (no seat switch, no device
  link of the creator's seat, no temporary admin in any frame) until a
  retry saves it. Each checks that
  only admins see the previous-seat hint, every change reaches every
  viewer's log, and no viewer's revision goes back.
- `links_test.go` covers device links (who may link a seat, a connection
  joining with a claim acting as the seat's user, admin powers included,
  revocation by a relink and by a switch, bad and revoked claims refused)
  and the rematch's refusals and creation limit. `actor_test.go` also
  refuses a record of a newer format untouched and upgrades the room
  actor's format-0 record.
- `bots_test.go` plays one human against bots at 3 to 7 seats through
  `ai.Pool`, checks the no-human pause (a spectator alone starts nothing,
  the last human leaving cancels the job and frees its permit, a restart
  resumes from the durable work rows) and undo's policy and refusals.
- `budget_test.go` (M16.table-request-budget) covers the per-user request
  budget: its bucket arithmetic, asking without taking; two tabs of one
  user sharing it, a request past it refused with `rate` while the record's
  bytes, the pick and every other connection's frames stay as they were,
  and the same seq accepted a second later; a relink, seat switches, a role
  change and a pick drawing on one budget; a new point filling every
  budget; an undo followed by the same point again filling nothing, also
  with an eviction between them; and no engine resolution for a refused
  completing confirmation or a refused switch that hands a bot a
  completing pick (counted by the `Resolving` hook, with a positive check a
  second later). The cache that keeps budgets across loads has unit tests:
  one take, each entry expiring when its last bucket is full, the size bound
  dropping the entry that would expire first, and an overflow (a dropped
  entry's note giving every user of its room only what refilled since the
  deepest debt it held, the notes' own overflow doing the same for every
  room, a refill filling nothing while the floor stands and everything once
  it has passed). At the served size, 2048 other rooms' entries leave an
  exhausted user exhausted, whether they are nearly full or deeper. The
  table keeps a spent budget across three evictions under memory pressure
  (`newTHRooms`, `evict`, `push`), an idle unload and an eviction whose
  entry the cache then drops, and starts full after a restart.
  `overflow_test.go` checks that an arriving entry that would expire first
  is the one noted, and that after both overflows a table loaded again at an
  old point, its mark lost, gives an undoing human no more requests than
  keeping its entry would have. `conns_test.go` refuses a user's
  ninth connection, seated or watching, with `room-full`, and walks a
  seven-seat table through twelve reseatings, promotions, switches to bots
  and a resignation, checking after every step that each user holds at
  most 8 connections, that watching connections number at most 32 plus
  seven less the players and that the table holds at most 88, and that
  the connections closed got `room-full` while each user kept their
  oldest. `poolkey_test.go` holds the point's player's address in the pool
  and checks that the bot's search waits for it, then that the key falls
  back to the next attached player, the creator and the smallest attached
  participant. In `lobbyroom`, `budget_test.go` covers the same budget for
  lobby edits, across two evictions and across an eviction whose entry the
  cache then drops, and `conns_test.go` the ninth
  connection, a player with a slot getting into a lobby that watchers have
  filled while a new watcher is refused, the same bounds through slot
  edits, a smaller seat count and Start (counted at the table from the
  clients' side), and in a rematch lobby.
- `leaks_test.go` passes every frame of whole games, for every seat and the
  spectator under both end policies, to the leak oracle
  (`tabletest/leakcheck`). First each frame's **envelope** is checked
  against its viewer: `you` names the viewer's seat, an acknowledgement
  the viewer's own client, the spectator gets no pending pick, and a seat
  gets only its own (required at the point, offered to it, its confirmation
  the one the public list shows, and, in process, exactly the pick the
  test saw its seat's accepted submissions set). Then the oracle checks the
  whole frame, its view rebuilt in the engine view's shape,
  the frame's own events wherever the rebuilt view did not take them from
  it (a history frame's, say), and its history. `merge_test.go` passes the
  same frames through an outbox drained only every few pushes, as a slow
  connection's is, so a frame carries the events of several resolutions
  and a history frame the events after its history; they pass, and a
  merged frame missing one event does not. A planted leak of each
  kind must be reported: another seat's card, a raw key in an event, a
  fingerprint, a suffixed card id, a deal-ordered hand, another seat's
  offer, an unknown field, and in the envelope another seat's pending pick
  sent to the spectator or given as the viewer's, the viewer's own key it
  never had accepted, a confirmation the public list does not show,
  another seat's `you`, another connection's acknowledgement, and a hidden
  card in a history frame's events. In the whole-frame scan only the
  pending key the envelope vouched for is left out, and only once the raw
  frame holds exactly that key and confirmation and names no field twice;
  everything beside it is scanned. Canaries plant another seat's
  observation hash, another seat's key and the key named twice beside a
  legitimate pending pick, in process and in a frame received over the
  socket, and each must be reported. The captured log is checked by the
  oracle and searched for hidden sentinels (the seed in decimal and hex,
  every key, the rules and observation hashes). `TestWebSocketTables`
  repeats the frame check on the raw bytes each connection received, and
  plants another seat's pending pick in a spectator's and a seat's frame
  received over the socket. The envelope also requires that only an
  admin's frame carries the previous-seat hint and that the hint names no
  seated user. Where viewers change, the oracle takes one viewer per frame
  and requires a connection's first frame for a new seat to carry its
  history: `TestSwitchFramesLeakNothing` checks every frame across
  switches, role changes and a resignation against the viewer the test's
  own model gives it, with planted cases for both rules. The model also
  remembers the seat each user held last, so the envelope requires
  `you.formerSeat` in exactly the frames of an unseated viewer of a game in
  play who held one, naming that seat, and `you.canPromote` only in a real
  admin's frames; `TestPlantedEnvelopeLeaks` plants a former seat in a
  seated viewer's frame, in a never-seated spectator's, and with the wrong
  seat, and the cap's room in a non-admin's frame.
- `TestPrivateViewFieldCanaries` (`viewfields_privacy_test.go`) finds, over
  seeded seven-seat random-side games, a sixth turn whose choices preview a
  kept card and a pile search with a card the searching city may not build,
  opens a real table there and checks every seat's frame. The oracle's own
  truth for a kept card comes from resolving the turn with that choice, and
  for a pile reason from the pile and the city's names. Planted copies must
  be reported: a neighbour's card as the preview, the preview copied into
  the neighbour's frame, a dropped preview, a pile reason in a seat that is
  not searching, a reason naming a card outside the pile, and a dropped
  reason. The oracle also checks each `military` event's shields and token
  worth against the resolved state and `result.tieBreak` and `result.tied`
  against totals and coins, and `engine_privacy_test.go` plants a forged
  tie, forged shields and a forged worth in a finished frame.
- `FuzzTableOps` (`fuzz_test.go`) drives one table through fuzzed picks,
  changes and confirmations from every seat, a second tab and the
  spectator, new tabs that open, pick and close under fresh client ids,
  resends (a closed tab reconnecting under its id to send one), stale
  references, undo, bot picks, detaches and restarts, from the start or a
  salvage or last-card point, with small seq and floor bounds. After every
  operation the record in memory must equal the stored bytes, the stored
  record must restore with a matching rules hash, no frame revision may go
  backwards, and a restart keeps every pick, confirmation and reference. A
  refusal or a duplicate must leave the stored bytes alone. Seat switches to
  bots of any level and to any person, role changes from any client,
  resignations and rematches (through a stub successor maker) are among its
  operations, and bots act only when the fuzzer says so.
- `TestWebSocketRestart` restarts the server on the same data directory with
  a confirmed pick pending and finishes the game: the pick lands in the
  first batch of the downloaded log.
- `TestWebSocketLobbyToTable` (`lobby_ws_test.go`) runs the whole platform
  path over the socket at 3 to 7 seats, with a spectator and two device
  links: a lobby filled by joins and a bot, a lobby promotion, Start
  moving every connection to the table under the same code, a switch to a
  bot that revokes that seat's device, the spectator seated, a demotion and
  a resignation, every connection's log the same, the game played out with
  the bots, and every frame through the oracle against its viewer at that
  frame's log length. Odd seat counts hide the cards at the end: the log
  download answers `log-hidden` and no final frame carries the reveal; even
  counts reveal it in every final frame. Then the rematch reaches every
  connection, the device linked to the creator's seat follows with its new
  claim, and the rematch lobby starts again.

Run the socket tests with the race detector, through the guard, from the
repository root:

    python3 tools/heavy.py run --kind test -- go -C server test -race -short -run WebSocket ./internal/tableroom/

**The race build's budget** (M16.tableroom-race-budget). The race detector
slows this package's long tests two to eight times, so its race build plays
the long sweeps on a sample. `RaceEnabled` (`race_enabled_test.go`,
set by the `race` build tag that `-race` adds) tells the tests apart:

- the seat-count sweeps play 3 and 7 seats (`SweepSeats`):
  `TestDriverInProcess`, `TestDifferentialAgainstDirect`,
  `TestDifferentialSwitches`, `TestFramesLeakNothing`,
  `TestMergedFramesPassTheOracle` (whose full set is 3, 5 and 7) and
  `TestWebSocketTables`; `TestWebSocketLobbyToTable` plays 4 and 7, an even
  and an odd count, so both end policies;
- `TestEngineContractSidesAndReplay` plays its seven-seat game under
  `choose` only, beside all three rules at three seats;
- `TestEngineContractFinishedMemory` measures two finished tables, not
  eight (the measurement is the plain build's);
- `TestEveryArrivalOrderInProcess` is skipped, as `-short` skips it.

The counts and games dropped add game size, not concurrency: the same
goroutines and code paths run at every count. Everything concurrent keeps
its full form under the race detector: actor schedules, restarts, pending
subsets, the bots and every socket test. No assertion changed, and the full
sweeps run without the race detector, in `make verify-quick` and CI's go
job. `TestDifferentialAgainstDirect` and `TestEveryArrivalOrderInProcess`
run in parallel with each other, since each has its own fixtures and
deadlines of minutes. The other tests stay sequential: the socket tests
count the process's goroutines when they clean up, the in-process sweeps
give each game 20 seconds, and the measurement tests read process-wide
clocks and memory.

Measured on the development machine, the race build with `-short`, as
`make test-race` runs it, went from about 680 to 347 seconds; without
`-short`, as CI runs it with every other package in the go job's race step,
it takes 384 seconds. The full run without the race detector is no slower
(about four minutes; the parallel pair overlaps).

`TestRecordClientFrames` (`clientframes_test.go`) records, in order, what
four clients of one table send and receive through the Manager: three
humans and a spectator. Seat 0 drafts, changes, confirms, withdraws and
confirms again; seat 1 sends a key it was not offered, confirms, and
changes its pick just as seat 2's confirmation closes the decision; seat 0
repeats a seq; seat 0, the creator, gives seat 1 to a bot, and seat 1 then
picks from the seat it lost. The recording, with the table's code replaced
by `TBLAAA`, is `testdata/client-frames.json`, one message per line; the
test fails when it changes. Rewrite it with `UPDATE_CLIENT_FRAMES=1` and
review the diff, because the web client's replay test reads it.

## `internal/lobbyroom`: the shared lobby

`shared_duel_test.go` covers two-seat Start under the same code, spectator and
multi-tab handoff, restart and carried device-link epochs, stale lobby requests,
failed-write preservation, and an online bot using the chosen Duel level.
A fixed-clock, fixed-seed comparison checks the new room against the legacy
online create/join path, with only the carried claim epochs, frozen admin set
and optional link capability accounted for. The existing assigned-seat Duel
constructor gives a second full-file byte comparison with identical epochs.
The legacy and new engine/view frame payloads are byte-identical after removing
only the explicit new link-permission marker. The frozen-admin regression
covers promoted spectators, a creator who stepped down, D23 hint preferences,
other-seat link refusals and persistence. A complete-game rematch checks
reservations, fresh roles, a reserved-code collision and restart idempotency.
Another asks for the rematch again after its lobby has started its own game,
loaded and after a restart, and gets the same code.
`TestSharedLobbyDuelRematchFailedPersist` fails the write that confirms the
successor: no `created` frame goes out, and after a crash or a plain retry the
next request answers the same code, whose lobby then starts and is still the
answer afterwards. `TestHintAdminBoundFollowsRoles` keeps the store's bound on
the frozen admin set equal to the roles' admins plus the temporary one. A
complete online game with a bot seat survives a restart in the middle of the
bot's turn (the bot resumes when its player attaches), and its rematch lobby
keeps the bot's slot and level. `TestSharedLobbyDuelWithoutTableCreation` runs the lobbies as
production does, without `-haftgane-tables`: a two-seat lobby is created,
started, played and rematched, `maxSeats` is 2, and creating, editing to or
starting three or more seats is refused, including a lobby made larger while
creation was on. The store's `TestHintAdminsChecked` refuses to save a damaged
frozen admin set and archives a file holding one as `replay-failed`.
Store tests separately refuse missing, damaged, foreign, archived or already
started targets without changing their bytes, and race two conditional Starts.
These extend the tests below; Duel protocol and bot goldens are not regenerated.


`lobby_test.go` runs lobbies in process through the real Manager and store,
beside the table kind. `TestLobbyEditsAndStart` walks a lobby from creation
to Start (joins, a removed human staying out, bots, a seated spectator, a
seat count change, a release, the refusal order, a repeated seq) and checks
that Start hands every connection the table under the same code with the
roles carried on. `TestStartRefusals` covers an open slot without explicit bot filling, a reservation not yet
opened and a failed write that leaves the lobby as it was.
`TestLobbyRolesAcrossRestart` persists promotions, the temporary admin and
the unseated list across a restart, and lets the temporary admin start.
`TestDetachAcrossStart` races a detach against Start twenty times: it is
never lost. `TestRematch` checks the successor lobby a finished table makes.
`TestLobbyRolesRefusedWrites` refuses the write of each temporary-admin
change (the grant, a real admin's return, the passing on) and checks that
the temporary role fails closed until a retry saves it.
`TestRematchClaimsRevoked` revokes rematch claims by removal, release, an
opened reservation and a cut seat count, and checks that seating the human
again, a restart and Start never revive them, while an unbroken claim
reaches its seat at the table. `TestRematchOneSuccessor` refuses each of a
rematch's three writes, restarts and lets the other original human retry:
the table always ends with one successor.
`TestCreationCollisions` races a rematch's lobby against another rematch,
an ordinary lobby, a table and a Duel room drawing the same code, in both
orders: the first to write keeps the code and its bytes, and the other
answers `ErrCodeTaken` or is made under another code.
`TestRematchCollisionAtTheWrite`, from the re-review's probe, plants a
competing lobby at the moment of the rematch's write: the rematch refuses
the code, the competitor's bytes stay as they were, and the code is never
written unconditionally. `TestLeaveThenJoinStaysAttached` holds the actor
while a connection leaves and joins again, twenty times: the connection
always ends attached.
Package `roles` has its own unit tests of the role policy and the
five-minute rules at explicit times.

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

`ajayeb-drive table` is a separate subcommand: the N-seat driver for
Haftgane's simultaneous points, which needs no server (see "The N-seat table
harness" below).

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
- `src/lib/games/haftgane/session/session.test.ts`: Haftgane's
  TableSession over synthetic frames (`test-frames.ts`) and a fake
  connection standing in for the bootstrap (`fake-connection.ts`). It
  covers the handover's first frame; picks only from the choices, each
  with the next seq and the open point; a change, a confirmation and its
  withdrawal; every refusal (decision-closed, illegal, stale, forbidden,
  storage, and rate, retried three times); duplicates answered by a frame;
  a reconnect resending a pick only for the same seat and point; a
  resignation, an undo and a pick each moved to another seat and to
  watching before a reconnect and before a `rate` retry, never sent again,
  and sent again unchanged while the seat stays; a reload restoring the
  pending pick and its confirmation; a Session mounted again in the same
  page going on from the last seq of the page's client id (with the
  server's restored pick), a reload's new id starting at 1, and the
  default counter being the page's; a tab moved off its seat becoming a
  spectator; the lobby's Start; device links; and the saved rooms'
  `haftgane` hint.
- `src/lib/games/haftgane/session/presenter.test.ts`: the TablePresenter
  plays a resolution's events in order, shows a frame with history or a
  changed seat without playing it, replaces a presence refresh in place,
  fast-forwards a backlog, and tells its still hook of a change without
  events (a seat confirming) but never of a restatement or a refresh.
- `src/lib/games/haftgane/table/cues.test.ts`: the table's cue mapper.
  Every event kind (a flip for the plays and the sides, a drum for the
  conflicts, the Age's deal, the result as a fanfare, a defeat or a
  spectator's chime, the viewer's own play, coins and trade receipts, the
  new hand, a kept card, a build from the pile and an empty pile look),
  the no-replay rule (a history, the first frame, another room or seat, a
  revision that does not move forward), the budget over the engine's own
  seven-seat view fixtures, the decisions that notify one viewer (a
  special decision opening, the table waiting on them alone when another
  person sits at it), the viewer's own confirmation and the vibration
  patterns.
- `src/lib/games/haftgane/session/recorded.test.ts`: the same two over the
  frames the real table actor sent (`TestRecordClientFrames`, above).
  Every frame parses; each client's Session, driven through its own API by
  the recorded requests, sends exactly what that client sent; the refusals
  of requests the Session would never make change nothing; and the
  Presenter plays only the resolution.

### Duel through the shared lobby

`shared-lobby.full.spec.ts` runs on desktop and mobile with `@full`. A filled
lobby remains a lobby until Start. Two tabs of one guest and a promoted
spectator follow the same-code Duel handoff; frozen admins can change shared
hints, including a spectator, but cannot mint the other seat's device link.
Reload preserves the guest's seat and hint permission. A separate case
switches two/three-seat panels, round-trips Each picks, Day and Random in
Haftgane's side-rule control while preserving the starter and hint settings,
hides that control at two seats, changes an existing bot's Duel level and starts its online game without Undo.
A third case finishes a shared game by the guest's forfeit and rematches
from the result screen: the owner who asks lands in a two-seat successor
lobby that holds the owner's seat and reserves the guest's, the guest and
the spectator follow it from their own result screens, the guest takes the
reserved seat, only the owner is an admin there (the guest's promotion in
the first game is not kept), and its Start begins the next game under the
successor's code. The bootstrap unit
regression verifies that Duel snapshots return to discovery without reaching
the old lobby Session; the v2 membership stays until the v1 state arrives.

`make shots PW_ARGS=--grep=shared.lobby.geometry` checks the two-seat panel at
360x740, 390x844, 844x390 and 1280x800 in English and Persian, both themes.
It requires no horizontal overflow, controls at least 44 px, instant
Animation-off changes and clean console/network probes.
`SHARED_LOBBY_SHOTS_DIR` selects the captures and probe directory.

D61 and legacy device-link regressions use the explicit
`createLegacyOnlineGame` compatibility fixture through `/dev/new`, so stored
legacy rooms retain their original v1 entry and creator link permissions.
The online playthrough and production smoke use Home's new shared lobby.
The leak checker removes only the lobby's fixed `options.hints` vocabulary,
like Duel's `room.hints`, because `economy` also names a progress token.
A negative canary preserves a planted token name in every other lobby field;
whole-game checks continue scanning all lobby and game frames. The legacy
waiting lobby loads its hint summary lazily; cold running-Duel measurements
therefore exclude the waiting-only component even when the shared lobby
reuses it. The ordinary game menu still needs its hint-summary text helper.

### Home and the Haftgane lobby

`make e2e PW_ARGS='--project=desktop --project=mobile specs/home-lobby.full.spec.ts'`
checks the production Home flow, named creation, adding and removing released
bots, a second isolated human, spectating, promote/demote/step-down, settings,
Start and a finished game's rematch reservations, including an original human
who became a spectator after a controller switch; that game is played
through the table screen with `choose()` from `fixtures/table.ts`, and its
rematch button follows the turn bar. These cases carry `@full`
and `@mobile`. A cold invite sends no hello or join until its name is
submitted. The count regression exercises three through seven seats and
requires confirmation before a decrease displaces a human; cancel keeps both
the roster and the displayed count unchanged. The QR check waits for the
real encoder's rendered path. Existing table-session leak and reconnect tests
also create their lobby from Home instead of the retired `/dev/table` route.

`make shots PW_ARGS=--grep=lobby.geometry` captures 360x740, 390x844, 844x390
and 1280x800 in English and Persian, light and dark themes. Every capture
checks horizontal overflow, 44 px control boxes, page hygiene and the game's
Animation-off setting. `LOBBY_SHOTS_DIR` selects the output directory; the
four `probes-<locale>-<theme>.json` files retain the measurements. The
lobby request tests cover ordered acknowledgements, rejection, reconnect
retries and explicit rematch navigation with a private claim. Local message
tests check complete Persian keys, RTL isolates and Intl number formatting.
`make shots PW_ARGS=--grep=platform` retains its measured resource list even
when a byte assertion fails, without changing the original entry limits.

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
  (welcomed, the game screen mounted, nothing queued in the Presenter, no
  request pending). The screen counts because Duel's screen loads lazily
  after the route: on a real network the live game holds its first state
  before the buttons exist;
- `history()`: the game's history as the session holds it, one entry per
  applied action (the `history` field of the state frame, kept across live
  frames), so a spec can check the log after a reload;
- `checkForUpdate()`: SvelteKit's deployed-version check, without changing
  the game or returning any identity data.
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
(`server/bin/ajayeb --dev --dev-endpoints --haftgane-tables --bot-delay 0
--seed-base 1000` on port 38954, with a fresh data directory under `.cache/e2e/`), in
Playwright's own Chromium (Chrome for Testing, `channel: 'chromium'`) both
locally and on CI, never the installed Google Chrome (D26). CI splits the
suite into four shards (`--shard=1/4` to `--shard=4/4`), each a job with its
own server, and keeps each shard's report and traces when it fails or runs
out of time. The shards leave out the `@perf` case, which runs in its own
job, `e2e-perf`, described under `animation.spec.ts` below, and the
`@slow` case, which waits out a five-minute server rule in real time
(`haftgane-playthrough.full.spec.ts`) and runs locally on demand. There are two projects: `desktop` (Desktop Chrome, a mouse) runs the standard specs, and `mobile`
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
  room-specific task, long animation frame and animation records,
  including unprofiled failures. Each overlapping task gets a
  `perf overlap` annotation, and all of them a `perf overlaps` attachment:
  its room, the frames whose animations it overlapped, its length, and
  what ran in it, from the Long Animation Frames API (`explainLongTask`
  in `fixtures/anim.ts`): each long frame's script entry points (invoker,
  file, position and function, the longest three) and its rendering time.
  The case uses the configured retry policy. Repeated acceptance runs use
  `--retries=0 --repeat-each=10`; every first attempt must pass. On CI it
  runs in its own job, `e2e-perf`, five times per project with no retries.
  There a native stall of the shared runner (a major GC, a compositor
  commit) sometimes lands in an animation, in a different place each time
  ([master-e2e-failures.md](dev/master-e2e-failures.md)), so
  `tools/perf_report.py` judges the attempts together: a place (a room and
  its frames) where a long task overlaps an animation in more than half of
  a project's attempts fails the job, and one that does so in fewer is
  listed as flaky. Any other failure of an attempt fails it too. The script
  writes the verdict, each project's median worst overlapping task, every
  overlap with its attribution and each attempt's `perf` annotation to the
  run summary; `tests/test_perf_report.py` covers it. The job does not fail
  the workflow.
- **e. Replay last move.** With Animation off the log sheet's control is
  disabled and its tooltip (keyboard focus on desktop, a tap on the
  phone) says to turn Animation on. At normal speed it is disabled before
  any move, then after the Arsenal's looting it replays: new animations
  are recorded, among them the looting, no move is sent (the version stays)
  and the screen afterwards is the screen before.

- **f. Frame gaps at full speed (`@perf`, information only).** Stutter at
  normal CPU speed is not the same thing as long tasks at 4x, so the run of
  d is played again unthrottled while `recordFrames` notes every
  `requestAnimationFrame` time. It runs once per run, skipping the repeats
  of `--repeat-each`. Its `perf` annotation gives the gaps between
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

- Context A opens Home, Play online, types a name and creates a two-seat
  shared lobby. It chooses who starts (Sun on desktop, Moon on mobile). In the lobby it presses Copy link and reads the clipboard, which
  must be the link the lobby shows and end in `/play/<code>` with the code
  the lobby shows in large type.
- Context B opens that link, answers the name prompt ("Play as …") and
  takes the open seat. A explicitly presses Start, then both lobbies give
  way to the draft; the room is
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

### `haftgane-table.full.spec.ts` (`@full @mobile @haftgane-table`)

A Haftgane table over the real server and the table screen, on desktop
and mobile, in three browser contexts. A choice of a hand card is clicked after
its card (the pick list shows a selected card's choices), and the admin's
seat controls are in the game menu; `fixtures/table.ts` holds these steps,
the table hook's waits and the oracle below, shared with the next spec.
The creator opens Home's Haftgane picker and New table form, a second
human joins the lobby from its own context,
and the creator starts it with bots in the two empty seats; a spectator
joins the running table. The creator confirms a pick and reloads, and the
pick comes back confirmed; it withdraws, changes and confirms again. The
second human drafts a pick and changes it, drops off the network
(`setOffline`) and comes back to the changed draft, then confirms last,
which resolves the point for every page. Then the creator gives the
second human's seat to a bot, and that tab shows the watching notice and a
spectator's view with no pending pick. The switch goes the other way too:
the creator seats that human again, then seats the spectator in the
human's seat, and the human watches, with the notice and the former seat.
Last, the creator, the table's only real admin, steps down, and the
temporary admin is made at once: the creator, the connected human who has
held a seat longest, whose frames carry the temporary role and no admin
cap, and whose seat shows the hollow role tag on every page.

The leak boundary is checked while both of the second human's drafts are
pending: the spec picks two of its choices that no page of the others has
shown, and neither may appear in the creator's or the spectator's
WebSocket frames, DOM, local or session storage, console, or
`window.__ajayebTable` dump. A pending-pick ownership oracle reads each
page's socket traffic both ways, in order, across reloads and reconnects:
it takes a page's own pick from what the page sent (each `table-pick`
under its hello's client id and seq), once a frame acknowledges that seq,
until the point changes or the page holds no seat, and requires every
`table-state` frame's pending pick to be exactly that, or absent. It runs
over the creator's and the spectator's traffic at the leak boundary and
over all three pages' at the end, and three contaminated copies must fail
it: a creator frame showing the second human's draft, the creator's own
pick with the wrong confirmation, and a spectator frame with a pick.

A second test leaves a solo table for Home and comes back through the
app's own links, in the same page and under the same client id: the
server restores the pending pick, a new pick is applied, and Start and the
two picks carry three rising seqs.

Both run in CI's shards but not in `make e2e-smoke`; run them alone with
`make e2e GREP=@haftgane-table` (which also runs the two specs below).

### `haftgane-table-screen.full.spec.ts` (`@full @haftgane-table`)

The table screen at five seats, one human and four bots, played through
the screen alone. The page creates the table from Home's lobby, sets five
seats and starts it, so the four open seats take bots (`openBotTable` in
`fixtures/table.ts`, which the geometry spec below shares). The board
shows each seat once (the strip, two neighbour panels, two far chips).
When the table lets each seat choose a wonder side, the page picks day,
changes to night and confirms; then three turns of Age I each pick a
card through its hand card and the pick list and confirm, the second turn
changing its pick first. The pick's hand card carries the drafted tag and
is pressed; a trade chip's tooltip shows its price on hover; every glyph
and image on the board sits inside a tooltip anchor or a labelled button;
nothing animates with Animation off. After the side point and after every
turn, the pending-ownership oracle passes over every table frame the page
received, and three contaminated copies must fail it. No console error or
failed request is allowed. Run it with `make e2e GREP=@haftgane-table`.

### `haftgane-city-notebook.full.spec.ts` (`@full @haftgane-table`)

The city sheet and the Seen hands sheet over a live five-seat table, one
human and four bots started from Home's lobby (`openBotTable`). Seen hands
first opens empty. The page then plays three turns, building a wonder
stage when one is offered, and reopens it: each hand the view's notebook
holds is listed under its point with exactly that hand's cards and the
played card marked, the pile looks likewise, no card outside the notebook
appears, and each circulating colour carries the view's bounds; a figure
filters the list and Show all clears it. The city sheet opens from a far
chip, and its ring tabs list the seats from the viewer round the table;
each tab, and the arrow keys, show that seat's city with exactly its
buildings, stages and points by category. The leak oracle: a city other
than the viewer's own has no line about the cards under a wonder, no card
element but its own buildings, and no name of a card the viewer holds,
kept or tucked (unless that city has built a card of the same name). The
viewer's own city names its tucked and kept cards. The neighbour panel
and the strip open their cities too. Every table frame passes the
pending-ownership oracle, no frame gives a seat a field outside the public
seat or the page another seat's private view, and no console error or
failed request is allowed.

### `haftgane-scores.full.spec.ts` (`@full @haftgane-scores`)

The end of each Age and of the game over a live three-seat table, once
with the end revealed and once with it hidden (the lobby's reveal
control). One human opens the table from Home, Start gives the other
seats bots, and a spectator joins the running table. The human sells a
card every turn. At each end of Age, the military dialog of the player
and of the spectator must list exactly the military events of the batch
that ended the Age (read from the page's own socket traffic, since only
the resolving frame carries events), with each outcome, both shields and
the victory token's worth as each event states them (the shields agree
with the frame's seats), the player's two conflicts and every seat's
military points. A reload at that moment must open it again. At the end,
the score sheet follows the last conflicts: its verdict, the tie-break
sentence exactly when `result.tieBreak` states a tie, its rows in the
result's rank order and every category of every seat must match the
result, and a row opens that seat's city over it. Under the revealed end
the log downloads as `ajayeb-log/2`; under the hidden end there is no
Download, the note shows, and the endpoint answers 403 `log-hidden`. The
final board's buttons reopen the revealed cards (the pile and box counts
match the view), the conflicts and the scores. Every frame passes the
pending-ownership oracle, nothing animates, and no console error or
failed request is allowed. Run it with `make e2e GREP=@haftgane-scores`.

The spec writes each policy's frames (the player's and the spectator's
socket traffic) and the room record to
`HG_SCORES_LIVE_DIR` (default `.cache/e2e/hg-scores-live`) as
`frames-revealed.json` and `frames-hidden.json`.
`TestScoresBrowserFrames` in `server/internal/tableroom` reads them from
`AJAYEB_SCORES_BROWSER_FRAMES` (a path list) and runs the leak oracle
over every frame against the truth replayed from the record; it skips
when the variable is unset. Three canaries replace the last frame with a
changed total, a download flag the policy does not grant and, under the
hidden end, a revealed box, and each must fail:

```sh
D=$PWD/.cache/e2e/hg-scores-live
AJAYEB_SCORES_BROWSER_FRAMES="$D/frames-revealed.json:$D/frames-hidden.json" \
  make test-go GO_PKGS=./internal/tableroom GO_RUN=TestScoresBrowserFrames
```

### `haftgane-admin.full.spec.ts` (`@full @mobile @haftgane-admin`)

The admin controls, resignation and spectator view over a live five-seat
table, on three devices. The creator starts it alone with bots from Home's
lobby (`openBotTable`), so undo is on, and plays a turn: the top bar shows
Undo. The creator undoes the turn: the frame goes back to the first batch,
the hand is whole again, the point reopens under a new reference with no
pending pick, and Undo goes until the creator plays the turn again. Two
friends open the running table and watch: their hand region is
the spectator panel, with the turn recap and the spectators, and one of
them follows a seat, so the ring's neighbours become that seat's. A
non-admin's game menu offers no seat. Through the seat's admin menu
(`seatAction`: the game menu, Manage, the action, Confirm) the creator
seats one friend in a bot's seat: that page is seated, and the creator's
Undo disappears with the `undo-off` notice. The creator gives the seat to
a bot: the former player reads that a bot plays their seat, and the
creator's seat menu shows the previous-seat hint beside them, while the
other spectator's menu shows none and none of that page's frames carries
`previousSeats`. Before that, the creator's seat menu offers exactly the
frame's `botLevels`, in order, and the role button its `canPromote`
allows; after it, the former player's frame names the seat in
`you.formerSeat`, and a fresh tab of theirs, which shares no session
memory, shows it too. The creator seats the friend again, promotes and demotes
them from the seat menu (the role tag follows), and promotes them again;
the friend steps down and resigns to a bot from the game menu
(`menuAction`), and the seat goes to the easy bot. The creator promotes
and demotes the spectator from the game menu's list, and the spectator
still follows the table as a turn resolves. Nothing depends on a deal:
every seat used is read from the frames. Every table frame of each page
passes the pending-ownership oracle (which owns the pick a seat keeps when
a page takes it, if it is one of that seat's choices), gives no seat a
field outside the public seat and no page another seat's private view,
and carries `previousSeats` only when the same frame lists its viewer as
an admin, `canPromote` only to a real admin, and `formerSeat` only to an
unseated viewer while the game is in play; the never-seated spectator
receives none. No console error or failed request is allowed.

### `haftgane-recap.full.spec.ts` (`@full @mobile @haftgane-recap`)

The reveal, the turn recap and the hand pass over a live three-seat table
from Home's lobby, for a seated player and a spectator. The creator, with
Settings' "Show opponent move" on, starts the table alone with bots, a
friend watches, and the creator builds, sells and stages in turn through
the end of Age I and two turns of Age II. After every batch the spec reads
that batch's events from each page's history and checks, with its own
reading of the events: the creator's recap opens by itself, with one line
per seat that acted, its act, its card where the page may know it and
every signed coin change, in ring order from the creator, where hands
went, the creator's own discarded last card, and a spoken live region;
once it closes, every seat's reveal tag is on the board, the new hand
carries the side it came from, and the top bar's button reopens the
recap. Choosing a card takes the tags off the board. The spectator's panel
shows the same lines with no card named for a sale or a stage, and the
spectator's board shows a tag per seat. At the end of the Age the recap
comes first and the military dialog only after it closes. The creator
makes the spectator an admin between two turns, and the next turn's recap
lists the change on both pages. Every table frame each page received
passes the pending-ownership oracle, names no other seat's face-down card
in its events, its history or a seat's `last` play, gives no seat a field
outside the public seat and no page another seat's private view. Nothing
animates with Animation off, nothing depends on a deal, and no console
error or failed request is allowed.

### `haftgane-sound.full.spec.ts` (`@full @mobile @haftgane-sound`)

The table's sound wiring over a live three-seat table from Home's lobby,
read from the page's fx test hook (`window.__ajayebFx`), as Duel's
`sound.spec.ts` does; the mapper itself is unit-tested above. The creator
starts the table alone with bots and a friend watches. Through the end of
Age I, for every batch the spec reads the batch's events from each page's
history and checks: each page records the batch's cues once, tagged with
the batch, at most three and 90 ms apart, none twice; the spectator hears
exactly the shared cues (one flip for the reveal, the drum of the Age's
conflicts, the next Age's deal); the creator hears those plus only cues
of their own, including their own play whenever the budget leaves room,
and their confirmation before the batch resolves is one tick at most.
Then a reload records nothing on either page and the next batch sounds
again; the spectator's table menu turns Sound off, the next batch records
nothing for them while the creator hears it, the switch survives a
reload, and turning it back on brings the cues back; and with Sound off
in Settings the creator's table never requests the lazy player chunk and
records nothing. Each new Age's deal marks the pass chip and the pass
flow for their one-shot sweep. A second test seats two people with bots,
who decide at once: each person in turn confirms first, sees the check
on their pick and hears one tick, and the other, now the only one the
table waits on, hears their turn once; the batch then sounds one flip on
both pages. A mutation probe that removes the table's sound call fails
the spec. Nothing depends on a deal, and no console error or failed
request is allowed.

### `haftgane-playthrough.full.spec.ts` (`@full @mobile @haftgane-playthrough`)

M18's acceptance playthroughs that no earlier spec held. The creator opens
a seven-seat table alone from Home, so Start gives the six other seats
bots, once under each end policy, and a spectator joins the running table.
The creator plays every point the table waits on it for: a card build
when one is offered, else a sale, else the point's first choice (a side).
It builds no stage, so no wonder decision of its own follows, and nothing
depends on the deal. Both pages reach the score sheet, with a row and
the right total for every seat. Every frame passes the pending-ownership
oracle, and no console error or failed request is allowed. The spec
writes each run's frames and room record to `HG_PLAYTHROUGH_LIVE_DIR`
(default `.cache/e2e/hg-playthrough-live`) as
`frames-7-<policy>-<project>.json`, in the format of the scores spec's
files, so `TestScoresBrowserFrames` runs the Go leak oracle over every
frame of the seated player and the spectator:

```sh
D=$PWD/.cache/e2e/hg-playthrough-live
AJAYEB_SCORES_BROWSER_FRAMES="$D/frames-7-revealed-desktop.json:$D/frames-7-hidden-desktop.json" \
  make test-go GO_PKGS=./internal/tableroom GO_RUN=TestScoresBrowserFrames
```

Its last test, tagged `@slow`, waits out the temporary-admin rule in real
time. `roles.AwayLimit` is a five-minute constant, and no test clock or
development endpoint can shorten it, so the e2e server gets no such
backdoor. The creator and a second human start a four-seat table with
bots, and the creator's tab closes. Once the room has seen the creator
away for five minutes (the spec checks that at least 290 s passed), the
second human is the temporary admin: the hollow role tag shows, the
frames carry no admin cap, a bot's seat takes another level through the
seat menu, and the away creator's seat menu offers seat powers and no
role button. The creator opens the table again, and the temporary admin
is demoted at once. CI leaves `@slow` out (`--grep-invert '@perf|@slow'`);
run it with `make e2e GREP=@slow`, about six minutes per project.

### M18 acceptance: the spec or test behind each line

The M18 acceptance list in [the roadmap](dev/roadmap.md), line by line,
with what proves it (M18.table-playthroughs). Every spec named here reads
its seats and choices from the frames, so it holds whatever the e2e
server's seed counter deals.

- The creator alone with six bots: `haftgane-playthrough.full.spec.ts`,
  under each end policy.
- Two humans in isolated contexts with bots and a spectator:
  `haftgane-table.full.spec.ts` holds the reload that restores a
  confirmed pick, the pick changed before the last confirmation, the
  offline blip, both directions of the admin seat switch, a spectator
  seated in a human's seat, and the temporary admin made at once when
  the last real admin steps down. `haftgane-admin.full.spec.ts` holds the
  promotion, the demotion, the step-down and the resignation (and a
  spectator seated in a bot's seat). The temporary admin made after five
  minutes away and demoted on the creator's return is the `@slow` test
  above; Go covers the same rule with a fake clock
  (`TestRolesAndTemporaryAdmin` in `internal/tableroom`,
  `TestTemporaryAdmin` in `internal/roles`, `TestLobbyRolesAcrossRestart`
  in `internal/lobbyroom`).
- An undo in a solo game: `haftgane-admin.full.spec.ts`.
- Every special decision: `haftgane-special.full.spec.ts`
  ([special-decisions.md](haftgane/special-decisions.md)), now on mobile
  too.
- A shared Duel rematch from the result screen into its successor lobby:
  `shared-lobby.full.spec.ts`, whose rematch test has the guest forfeit,
  the owner ask for the rematch, the guest and the spectator follow it,
  the successor reserve the guest's seat and keep only the owner as
  admin, and its Start begin the next game under the new code.
- The leak oracle over seven seats and a spectator under both end
  policies: `TestFramesLeakNothing` in `internal/tableroom` (3 to 7 seats,
  every client's every frame, both policies, in `make verify-quick`), and
  `TestScoresBrowserFrames` over the browser frames of the seven-seat
  playthrough above.
- Layout at 3 to 7 seats, four viewports, both themes and right to left:
  `ux/haftgane-table.spec.ts` (`make shots PW_ARGS='--grep "haftgane table
  geometry"'`), which also fails on any console error or failed request.
- Tooltip coverage, the i18n lint and the logical-CSS lint:
  `web/scripts/lint-ui.mjs` in `make lint-web` (no hard-coded text, no
  physical CSS property, no native `title`), and the untipped-glyph probe
  of the Haftgane UX specs. Animation off: the geometry spec's motion
  count and the scores and recap specs' running-animation checks.
- Both cold-entry budgets: `ux/platform-baselines.spec.ts`
  (`make shots PW_ARGS=--grep=platform`).
- A deploy in mid-game: the deploy test in `shell.spec.ts`.
- Duel unchanged except its online start, and a two-seat Start equal to
  a legacy online game: `TestSharedLobbyDuelAssignedFileMatchesLegacy`
  and `TestSharedLobbyDuelStart` in `internal/lobbyroom` (the room file
  and the first frames, byte for byte, apart from the claim epoch and the
  hint admins), and `shared-lobby.full.spec.ts`, where no lobby and no
  role tags show once the game starts.

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
   it through the UI (Home, Play online, a name, Create), reads the
   invite the way a player would (Copy link, then the clipboard: see "The
   clipboard permission" below), and context B opens that link, answers the
   name prompt and takes the open seat. A presses Start. They then play at least five moves
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
position must stay below 2500 DOM elements (the closed log sheet renders no
lines, so a long log does not count). Its log test opens the log at Age I,
on the result screen and after a Great Library's draws, in English and
Persian: every card, wonder and progress token a line names
(`[data-log-item]`) is a tooltip anchor inside its sentence, and its tooltip
has the board's detail (the printed cost) on hover, on keyboard focus and on
a tap, which leaves the log open. With Animation on, a closing log keeps its
height and its lines in every frame of the exit, keeps them when reopened
mid-exit, and holds none once closed. The formatter's unit tests
(`log-format.test.ts`) check that a player name shaped like a placeholder
or a private-use marker is only ever text, in both locales.
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

`ux/haftgane-table.spec.ts` (filter `--grep haftgane`) checks the
Haftgane table screen's geometry outside that matrix: tables of 3, 4, 5,
6 and 7 seats at 360x740, 390x844, 844x390 and 1280x800, light and dark, English
and Persian, each with a full hand and nothing selected. Its probe runs
two frames after each resize, without waiting for motion to end, and
requires no horizontal overflow, no control under 44 px, no seat, hand
card, bar or control cut off by the screen's edges, no glyph without a
tooltip and nothing animating. Then the page plays a turn and counts the
running animations in the first frame that shows the next hand: none with
Animation off. A last test is the positive control: the same count with
Animation on (relaxed) must find the deal, so a zero above means
something. The page may log no console error or failed request. Its
screenshots and `probes-*.json` go to `HG_TABLE_SHOTS_DIR` (default
`.cache/e2e/hg-table-shots`).

`ux/haftgane-polish.spec.ts` (filter `--grep "haftgane table polish"`)
captures the M18.table-polish fixes over live tables from Home: tables of
3, 5 and 7 seats in English and one of 7 in Persian, both themes. On the
7-seat tables it checks, at 360x740 and 1280x800, that the card sheet's
primary button is the drafted outcome, that an empty own city says so
once and counts no empty colour, that a bot seat's admin menu offers the
other bot levels, and that no name in the conflicts list breaks inside a
word and each has a tooltip. On every table, at Age II's last turn with a
card drafted, each hand card is at least 4.5rem wide (44 px on a short
landscape screen) with its pick tag inside it, at all four sizes. Its
checks are soft, so a run on an older build captures every state and
reports each problem; screenshots and `report-*.json` go to
`HG_POLISH_SHOTS_DIR` (default `.cache/e2e/hg-polish-shots`).

`ux/haftgane-persian.spec.ts` (filter `--grep "haftgane in Persian"`)
plays one 7-seat table in Persian from Home's lobby, with sides chosen
by each player and a second device watching. In Age II the admin seats
that device in a bot's seat and gives the seat back to a bot, so the
device watches as a former player, and then asks to resign and goes back.
It captures every table screen at 360x740 and 1280x800 in both themes:
the lobby, the side choice, the table, the card sheet, the own and
another city, the game menu, a bot's seat menu, the spectators, the
rules, the recap, Seen hands, the spectator's view, the admin's
confirmation, the former player's watching view, the resignation's
confirmation, both conflicts dialogs, the scores, the final board, the
revealed cards, and any special decision the deal gives the user. On each, and in
the tooltip of every anchor on the first turn's table at 1280x800, it
collects the Latin text outside people's names (`dir="auto"`) and the
room code, and any Latin comma or semicolon beside Persian letters (Persian
writes «،» and «؛»), and requires none. Screenshots and `report-persian.json` go
to `HG_PERSIAN_SHOTS_DIR` (default `.cache/e2e/hg-persian-shots`).

`ux/haftgane-city.spec.ts` (filter `--grep "haftgane city"`) does the same for the city sheet and Seen hands. One
live table each of 3, 5 and 7 seats is played to three moments: the first
turn (Seen hands empty), Age II's second turn (Seen hands with two Ages of
memory, and the city sheet on the viewer's own city, the left neighbour
and a far seat) and the end (the three cities again). At each moment the
page switches through English and Persian in both themes by setting the
settings and theme in local storage and reloading the table. At 360x740,
390x844, 844x390 and 1280x800 its probe requires no horizontal overflow
of the page, the table or the sheet, no horizontal scroller and nothing
past the sheet's sides, no button or tab under 44 px, the sheet inside
the screen, no glyph without a tooltip and nothing animating. Each state
is saved, and at 390x844 again scrolled to the sheet's end, with
`probes-<seats>p.json`, in `HG_CITY_SHOTS_DIR` (default
`.cache/e2e/hg-city-shots`). A last test loads the dev gallery, checks its
city and notebook leaves (three cities, seven ring tabs, the cards under
a wonder both as the owner's and as revealed, a neighbour's pick), and
opens both sheets from it under the same probe.

`ux/haftgane-scores.spec.ts` (filter `--grep=Haftgane.end`) captures the
end of an Age and of the game. One live table each of 3, 5 and 7 seats,
in English light and in Persian dark, is played from Home to the end of
Age I (the military dialog, at its top and scrolled to its end) and to
the end of the game (the score sheet, the final board and the revealed
cards), each at 360x740, 390x844, 844x390 and 1280x800. Its probe
requires no horizontal overflow of the page or the sheet, no control
under 44 px, no glyph without a tooltip, the panel inside the screen and
nothing animating, and the page may log no console error or failed
request. Two more tests open the gallery's score sheet with each
tie-break sentence (`/dev/gallery?hgEnd=tie|shared&seats=7`), which the
live view cannot show yet. Screenshots and `probes-*.json` go to
`HG_SCORES_SHOTS_DIR` (default `.cache/e2e/hg-scores-shots`).

`ux/haftgane-admin.spec.ts` (filter `--grep "haftgane admin"`) captures the admin controls
and the spectator view. One live table each of 3, 5 and 7 seats
is started from Home's lobby by its creator with bots; a friend joins and
is seated in a bot's seat, and a second friend watches and follows a seat,
each on a device of their own. In English and Persian, in both themes,
every page reloads, and the run captures the admin's table, game menu,
seat menu on the player's seat with its confirmation and seat menu on a
bot's seat; the player's table, game menu and resignation confirmation;
and the spectator's table, spectator panel and game menu. At the end the
player's seat goes to a bot and the former player's table is captured. At
360x740, 390x844, 844x390 and 1280x800 the probe of
`ux/haftgane-city.spec.ts`, without its title check, must find nothing,
and nothing in the table's top bar (`data-top-bar`) may run into its
neighbour: each direct child's box, widened to all of its content, must
not meet another's, so a label spilling past its own box counts (a check
added after the Age and turn ran under the pass chip at 360 px, which the
other checks could not see). The screenshots and `probes-<seats>p.json`
go to `HG_ADMIN_SHOTS_DIR`
(default `.cache/e2e/hg-admin-shots`). A last test checks the dev
gallery's admin leaves and opens both of its seat menus under the probe.

`ux/haftgane-recap.spec.ts` (filter `--grep "haftgane recap"`) captures the
reveal and the turn recap. One live table each of 3, 5 and 7 seats is
started from Home's lobby by its creator with bots and Settings' "Show
opponent move" on, and a friend watches on a device of their own. In
English and Persian, in both themes, both pages reload and the creator
plays a turn, preferring a card bought from a neighbour, so that every
variant reveals one turn live (a reload restates none). The run captures
the player's recap as it opens by itself and the board with every seat's
reveal tag once it closes, and the spectator's tagged board and spectator
panel. At 360x740, 390x844, 844x390 and 1280x800 the probe of
`ux/haftgane-admin.spec.ts` must find nothing. The screenshots and
`probes-<seats>p.json` go to `HG_RECAP_SHOTS_DIR` (default
`.cache/e2e/hg-recap-shots`). A last test checks the dev gallery's reveal
leaves and opens its recap overlay under the probe.

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

`dialog-focus.spec.ts` checks where the focus lands when a sheet opens. A
reload into a finished game and a reload into the starter choice, with no
input since the load, open on the dialog's title heading (`tabindex="-1"`,
computed outline `none`, the dialog's `aria-labelledby`), never on the
scrolling body. Tab then moves to a control inside the dialog that shows
the keyboard ring. Home's settings sheet, opened with a click, does the same,
and the join sheet opens on its autofocus code field.

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
worker installation and real offline Home and game deep links. Its install case
also checks that install leaves out only the files the `/dev` pages alone load
(web/README.md, "App shell"). The Go static
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

## Duel compatibility baselines before Haftgane

@OpenAI, 2026-10-21. The M14 baselines record `46a540d` before shared-layer
changes. They are tests and measurements, with no product behavior changes.
Regeneration is an explicit review action, never a normal test side effect.
Use the agent guide's detached guard form when running unattended.

### Schema-1 writes and loads

`TestPlatformSchema1Baseline` compares actual Store output byte for byte
against `server/internal/store/testdata/platform-schema1/`. Seven cases cover
lobby, unfinished play, a finished game, both unfinished and finished
archives, rules-version archiving and a compatible hash refresh. The goldens
include fixed timestamps, rules/hash fields and non-default room metadata.
The D28 transition cases preserve an unknown additive field. Archive paths
are pinned under `rooms-archive/`, outside live `rooms/` scans (D29).

Each committed fixture is also loaded in a separate fresh store. Loaded
fields and replay hashes must agree, and load must leave its bytes intact.
Existing D28/D29 suites retain their broader failure, retention and
concurrency coverage. To regenerate deliberately:

```sh
UPDATE_PLATFORM_BASELINES=1 python3 tools/heavy.py run --kind test -- \
  make test-go GO_PKGS=./internal/store 'GO_RUN=^TestPlatformSchema1Baseline$'
```

### Room-file codec dispatch

`M14.room-file-game-dispatch` keeps the schema-1 goldens above unchanged.
`TestRoomFileDispatchBeforeDuelDecode` gives unsupported schemas/games payloads
that cannot decode as Duel and checks explicit refusal before that decoder.
`TestRoomFileCodecSelection` registers two fake schema-2 codecs, checks exact
selection and original bytes, and proves unsupported headers call neither.
`TestRoomFileCodecRegistration` rejects duplicate ids, attempts to replace Duel,
invalid ids and missing decoders. Production registers no schema-2 codec.

`TestRoomFileCodecPreflightAndRetention` uses the same injected table for both
preflight and retention. Fake game rules differ from Duel's; the codec supplies
the compatibility verdict. Aggregate counters are unchanged in kind; malformed or Duel-incompatible
headers can now count as schema errors instead of only read errors. The report
format and scalar Duel `rules` value stay unchanged. The test checks read-only scans,
hash-refresh classification, live archive risk, archive replay avoidance,
expiry and byte-preserving legacy archive migration. Malformed discriminators
or retention metadata fail closed in `TestRoomFileMalformedHeadersFailClosed`.
`TestRoomFileSaveRefusesUnsupportedTarget` protects clean bounded headers
selecting a non-Duel game or schema-2 codec from the typed Duel save path.
`TestRoomFileSaveRepairsDamagedLiveFile`, lifted from the FULL review's repair
probe, checks that the next save repairs truncated, empty, garbage, schema-string
and oversized files, plus missing/duplicate discriminators, undecodable Duel
bodies and unreadable files. Each repaired file must load successfully, matching
pre-slice Duel behavior. Decoded Duel archives retain their existing protection.
`TestRoomFileHeaderErrors` checks unexpected EOF for incomplete inputs and a
trailing-data diagnostic without a nil-error suffix. Existing D28/D29 concurrency, restore failures,
retention bounds and read-size limits remain covered by the full store suite.

`TestRoomFileOldBinaryPreservesSchema2` uses only the pre-slice store API. It
loads and peeks at schema-2 live, finished, legacy-archive and archive-folder
fixtures, runs reaping with zero archive caps, then preflights the directory.
All four remain byte-identical at their original paths; preflight reports four
schema errors, with no playable room or archive candidate. The same test was
compiled and run against frozen pre-slice master `31ec171`, extracted with
`git archive` into temporary source space. No checkout, data directory or
fixture regeneration was involved. This proves rollback refusal and preservation
using the old implementation, alongside the current implementation's checks.
`TestRoomFileOldBinaryPreservesIncompatibleSchema2` adds a schema-2 payload whose
numeric seed conflicts with Duel's string field. Both old and new code must
preserve its bytes through load, peek, reaping and preflight, with one error and
no playable room or archive candidate. The old binary counts it as a read error;
the new dispatcher counts it as a schema error. Either error refuses rollback.

Run targeted checks through the guard:

```sh
make test-go GO_PKGS=./internal/store
make test-go GO_PKGS=./internal/store 'GO_RUN=^TestRoomFile'
```

### Room creation

`create_test.go` checks the conditional create every room's first write
uses. `TestCreateGame` and `TestCreate` hold codes with a live record of the
same game, another game's record, a Duel room, damaged bytes, a newer schema
and an archived room: each creation is refused with `ErrExists`, every file
stays byte for byte, and `CreateGame` returns the live bytes it found. A free
code is written as a save would write it. `TestCreateRace` races twelve
creators of both schemas for one code, twenty times: exactly one wins, and
the file is its own.

### Authoritative lazy client entry

`lib/entry/bootstrap.test.ts` checks v2 welcome-before-join, same identity/claim
on retry, latest unknown snapshot replacement, old-server protocol refusal and
close without leave, and an offline initial load resuming on the online event.
Saved Rooms tests cover valid optional game hints,
normalization, cross-room updates, stale-hint correction and an old tab's exact
legacy-field rewrite, which retains `mode: "online"` and drops `game`.

`e2e/specs/game-entry.spec.ts` runs on desktop and mobile. A cold online invite
with no name sends no hello/join and leaves the second seat open until submit.
D61 then takes the ordinary v1 path. Every creator-side state is checked for a
transient away flag after admission, and toast observations catch a false left
notice. Reload allows the old document's one real disconnect, with no second
handoff disconnect after the player returns. Blocked name storage still sends
the submitted name. Duplicate tabs and reconnect retain the same seat.

A stale Haftgane tag for a real Duel room verifies overlapping v2/v1 membership,
identical identity, no leave, and discovery closing after the first v1 state.
An untagged table first refuses v1 before admission, then opens the
Haftgane entry through v2; no Duel board or seat is produced by that refusal.
Its stubbed table frame is the spectator's first frame from the recorded
client frames (`TestRecordClientFrames`), so it stays a valid frame.
Session unit tests limit the wrong-game outcome to a welcomed join before its
first state; all other protocol errors remain fatal. Discovery bad-hello uses
the same fatal reload notice. Rematch and device-fragment tests retain the
actual GameScreen node and one socket, with no leave. The tutorial test from
2912538 remains unchanged and covers a pending bot transition on cold entry.
Online and recovery connection counts retain their pre-slice expectations.

The import fixture removes Duel modules from the root; Home's set is unchanged.
The generic route contains only the lightweight host and identity helpers.
Bootstrap, discovery, registry and all Haftgane modules stay outside its eager
closure. A real Duel invite must not fetch the Haftgane chunk. The frozen
registry's separate loader functions also protect emitted CSS/preload boundaries.

The build version uses the git commit, or a source-content digest when git is
absent, rather than a timestamp. The browser test calls SvelteKit's actual
`updated.check()` through the read-only hook: the deployed version returns
false and an intercepted newer `/_app/version.json` returns true. Two clean
builds must give identical cold-entry and install measurements. The Duel gate
stays 271,384 gzip bytes; this slice additionally retains at least 500 bytes
of margin without regenerating the byte baseline.

Run the unchanged `make shots PW_ARGS=--grep=platform` baseline after building.
Do not regenerate byte budgets. The baselines remain 141,410 gzip bytes for
Home and 266,384 for Duel, with at most 5,000 extra per entry. Record both clean
builds' actual cold bytes and worker install transfer in the slice result.
The all-build precache policy is unchanged; install transfer is measured
separately from code executed on the cold path.

### Protocol-2 game envelopes

`TestPlatformStrictDecoders` and `TestPlatformInvalidValues` cover all frozen
table/lobby request families, missing required fields (zero and false
included), nulls, duplicate/unknown fields, exact spelling, nested options,
operation-specific payloads and input bounds. `TestPlatformLeavesDuelRequestsUnchanged`
compares v1/v2 decoding of the existing request corpus, including the common
Duel device-link defaults. Platform DTOs generate separately into
`web/src/lib/wire/platform.gen.ts`; `web/src/platform-frames.test.ts` compiles
all new discriminator families against its generated unions. `make gen-check`
must leave every existing
Duel generated file and fixture byte-identical.

`TestPlatformDuelPlaysOnBothVersions` completes a real-socket Duel game on
each version, with action/reject/duplicate/undo/reconnect checks.
`TestPlatformFamiliesAtTransportBoundary` checks welcome echoes, v1 platform
refusals and v2 strict family rejects without enabling production creation.
`TestPlatformV1AdmissionBeforeAnySeatClaim` covers cold and loaded non-Duel
handles, a production-unsupported codec, preserved file bytes and refusal
before admission, load or attachment. `TestPlatformDispatchByActualKind`
uses a test-only handle to check authenticated sender routing for common
operations and family isolation. Production registration remains Duel-only.

`TestTableOutboxSlowClientAndUndo` checks forward event/history/ack retention,
latest private pick, full-history replacement, undo and backward revision.
`TestTableOutboxBoundsAndFamilySwitch` checks the 256-event cap, shared
four-room snapshot budget and cross-family replacement under one room code.
The original typed Duel merge and tests remain unchanged.

The unchanged M14 v1 transcript digest manifest is a separate byte-exact
check, without baseline regeneration. Full acceptance also runs the race
suite over wire/ws/room/protocoltest/store/cmd/ajayeb, `make verify-quick`,
and the online full-game/spectator/reconnect and leak Playwright specs on
desktop and mobile with retries disabled. No engine or arbitrary hidden
payload is used to fill the named M15 view/event placeholders.

### Room-kind registry

`TestRoomKindIdlePublicationBeforeRegistration` builds a Duel handle before the
first save and proves its initial idle state remains evictable.
`TestRoomKindUnmatchedSelectorIsLogged` pins the room/game diagnostic for a
selector with no matching registry entry.

`TestRoomKindsRegistration` rejects duplicates, incomplete callbacks, invalid or
mismatched codec ids, replacement of the schema-1 Duel codec and omission of
Duel. `TestRoomKindsCodecBinding` proves the registry supplies the same immutable
codec descriptors to store/preflight, checks header selection without decoding
the game payload, rejects mismatched schema/game pairs and keeps test codecs out
of the production table. `TestRoomGameHeaderSelection` covers the live/archive
lookup, invalid or missing codes, byte preservation and the existing read cap.

`TestRoomKindsDuelDispatch` observes the registered create/load/admission path,
attach/detach and inbox delivery, then forces capacity eviction of the idle Duel
room. `TestRoomKindsAdmissionBeforeReserve` verifies cold refusal before capacity
reservation and unchanged identity/claim arguments; an unregistered game stays
on disk without a loader call. `TestRoomKindsGlobalCodesAndHandleLifetime`
forces a code collision across kinds and checks that a stale actor cannot remove
or mark idle its replacement, and that Close waits for registered actors.
`TestRoomKindsForeignLoadUsesRegisteredHandle` restores a fake game whose payload
cannot decode as Duel, reuses its warm handle, exercises attach/detach and
capacity eviction, and refuses Duel-only requests/snapshots for that actor.

The existing room, WebSocket and protocol suites retain their claims, concurrent
restore, ghost-attach, idle/dirty eviction, rematch, bot/review and shutdown
coverage. Run them under `test-race` along with store. The schema-1 and version-1
baselines above remain unchanged and must run without regeneration. Browser
acceptance uses the existing online full-game/spectator/reconnect scenarios,
room recovery and room/rematch flows on desktop and mobile, through `make e2e`.

### Version-1 frames

`TestPlatformV1TranscriptBaseline` pins every raw frame's length/SHA-256,
its index and kind, plus the full stream length/digest. The 9,128-byte
manifest represents the unchanged 76-frame, 2,252,287-byte game without
storing another copy of those payloads. Failures identify the first changed
frame, save the actual stream and print commands to reproduce master's
stream and compare the bytes. Existing readable engine-view and individual
wire-message goldens retain all named phases, viewer redaction and
additional message shapes. See [dev/protocol-fixtures.md](dev/protocol-fixtures.md)
for capture and deliberate manifest regeneration. These checks run in
`make verify-quick`, without a browser.

### Entry imports and bytes

`web/src/platform-imports.test.ts` parses runtime imports, re-exports,
literal dynamic imports and asset globs, excluding whole-declaration
type-only imports. The fixture `web/src/testdata/platform-imports.json`
keeps only each entry's sorted set of eager modules: root layout 162,
Home 57, Duel route 360 (including external modules/assets). It occupies
23,291 bytes in 596 lines, down from 268,086 bytes and 9,507 lines.

The test computes eager edges and shortest predecessor paths in memory.
An unexpected module fails with its name, entry and full import path;
missing modules and unsorted/duplicate sets also fail. Eager Haftgane code
is forbidden even if mistakenly added to the expected set. Lazy-only
changes do not alter this fixture, and no full edge list is stored.
Unresolved eager local edges and nonliteral runtime imports encountered
in the eager closure fail closed. Review a changed eager module set before
regenerating it:

```sh
UPDATE_PLATFORM_BASELINES=1 python3 tools/heavy.py run --kind test -- \
  pnpm -C web exec vitest run src/platform-imports.test.ts
```

`e2e/ux/platform-baselines.spec.ts` measures fresh Home and deep-linked
Duel entry contexts with service workers blocked and HTTP cache disabled,
summing `gzipSync(response.body()).length` for unique script URLs, as in
the UX budgets test. The Duel case is a seeded hot-seat room, no prior Home
navigation, with a stored name and Animation off, measured through input
readiness and network idle. This is a test-mode build, not a production
network-bandwidth estimate.

The committed `e2e/fixtures/platform-baselines.json` records Home at
141,410 gzip bytes (35 scripts) and Duel at 266,384 (53 scripts). Home
retains its 200,000-byte ceiling; both entries must stay within baseline
plus 5,000 bytes: 146,410 for Home and 271,384 for Duel.

Worker installation is measured separately in an empty context, registering
the actual worker from `/health` so app assets have not warmed the cache.
It makes 108 install requests, including the registration script, and
caches 107 files. Responses contain 2,067,299 body bytes and 2,169,812
bytes including response headers; the registration script is 7,952 bytes.
The measurement excludes the initial health response, outgoing request
headers and transport framing. Local responses are uncompressed. Chromium
exposes precache bodies through request sizes and the registration body
through its response buffer; the test cross-checks precache body sizes
against the actual installed cache. No worker cache policy or install-byte
ceiling is changed.

To record both measurements deliberately, build and run:

```sh
UPDATE_PLATFORM_BASELINES=1 python3 tools/heavy.py run --kind light -- \
  make shots PW_ARGS=--grep=platform
```

Omit the update variable to enforce the committed entry budgets.
`web/src/platform-bytes.test.ts`, included in `make verify-quick`, checks
that recorded totals match every per-file measurement and that the fixed
ceiling and allowance remain intact. The browser check still requires
`make shots PW_ARGS=--grep=platform`; verify-quick does not start a browser.
Byte-budget gates permit the documented allowance rather than requiring
every rebuilt script to have identical bytes.

### Existing load and latency tools

`e2e/fixtures/platform-load-baseline.json` records the local diagnostic
samples alongside the stable compatibility fixtures. These timings are
observations for comparison, not exact runtime goldens or capacity limits.
The existing commands were used without adding a new load harness:

- `ajayeb-drive`: 20 concurrent Easy games, pick seeds 1200 to 1219,
  server seed base 1000, bot delay zero, trusted loopback proxy for the
  driver's synthetic addresses. All games finished. Peak RSS was
  22,960 KiB, sampled 69 times at 100 ms intervals. Whole-game duration
  ranged from 5.924 to 6.820 seconds, median 6.393 and nearest-rank p95
  6.809. Driver pacing is included, so these are not action latencies.
- `make bench BENCH_ARGS='-games 2 -positions 3 -playouts 30 -p 1'`: the
  existing simulator reported 76.8 microseconds per random game, 69.7
  actions/game and 0.91 million actions/second. Per-call timings were
  1,007.6 ns for Determinize, 98.5 for LegalActions, 385.8 for Evaluate
  and 155.5 for Observe. Mean clock-budget move times over three positions
  were 0.071 ms for Easy, 400.999 for Medium and 2,013.213 for Hard. This
  small diagnostic is not a strength or preset calibration.
- `make e2e E2E_BUILD=0 GREP=@perf PW_ARGS='--project=desktop --retries=0'`:
  both existing animation cases passed on their first attempt. At 1x CPU
  the median/p90 frame gap was 17/17 ms, with none over 34 ms. At 4x,
  72 animations had zero overlapping long tasks; the worst overall task
  was 253 ms. Frame gaps had median/p90 17/17 ms, one over 34 ms (0.1%),
  and a 67 ms maximum. Full annotations are retained in the fixture.

For a future driver comparison, build the existing command through the
guard, start a test binary with a fresh data directory and the same seed,
bot-delay and loopback-proxy flags, then pass `--games 20 --concurrency 20
--mode ai --bot easy --seed 1200 --server-pid <pid>` and a local `--origin`.
Stop that server by its PID afterwards. Simulator and perf samples can be
regenerated with the commands above; review and replace the observational
fixture deliberately, never assert identical timings across machines.

There is no Haftgane engine to mix into this baseline. The current driver
does not record per-action/reveal p50/p95/p99 or bot queue/pool-hold times;
the perf specs measure presentation, not server latency. Those measurements
remain unavailable rather than inferred from whole-game time.

## The N-seat table harness

@Claude Opus 5.5, 2026-10-23 (M16.platform-interleaving-harness). Haftgane's
room actor takes simultaneous picks from three to seven seats, so arrival
order must decide nothing, and its hand game has far more hidden information
than Duel. This harness lets the actor be built and tested before M15's
engine exists. All of it is test infrastructure:
`internal/haftgane/fakeengine/guard_test.go` fails if the server binary
(`cmd/ajayeb`) reaches any of it, or if any non-test package outside the
harness and `cmd/ajayeb-drive` imports it.

Run its tests with `make test-go GO_PKGS='./internal/haftgane/...
./internal/tabletest/... ./cmd/ajayeb-drive'`.

### The fake engine (`internal/haftgane/fakeengine`)

A small pure hand game with exactly the functions of
[haftgane/engine-architecture.md](haftgane/engine-architecture.md) section 16:
`NewGame`, `SeatOrder`, `Replay`, `Point`, `Legal`, `Resolve`, `ViewFor`,
`History`, `Observe`, `Fingerprint`, `CheckInvariants`, `Batch` and `Pick`.
Its types are its own (views in `fakeview`). The actor now uses the real engine;
this double remains isolated test infrastructure. The package documentation
states the rules. In short:

- three to seven seats, two Ages of two simultaneous turns, hands of three
  passed left and then right, Age 2 dealt at setup with an undealt box;
- every turn's keys are `build:<design>:<left>:<right>` (the coins paid to
  each neighbour), `tuck:<design>` and `sell:<design>`; a tuck or a sale is
  private, and so are the leftover and kept cards;
- one-seat points: the archive's holder salvages a design from the pile
  after it tucks, and the orchard's holder plays its kept last card;
- the end reveal shows every tucked card, the pile and the box, unless the
  game was created with `HideAtEnd`.

It follows the engine purity rules: standard library only, integers only, no
maps, its own seeded streams, and no chance after `NewGame`. `purity_test.go`
enforces them. `fakeengine_test.go` sweeps 400 random games at every seat
count, under both seating rules and both end policies. After every point it
checks the invariants, that replay matches play, that every view is free of
`null`, and that history matches the live events. Its noninterference
test, early in 200 games, replaces the other seats' hands, Age 2's deal and
box, and the pile's contents. It requires each seat's view, observation and
observation fingerprint, and the spectator's view, to stay the same.

### The driver core (`internal/tabletest`)

The driver speaks to a `Table`: `Snapshot` per client, `Submit` of
`{seq, point, key, confirm}`, and `Canonical`, which returns the batches and
the rules hash. A `Factory` opens a table at a prefix of batches.

- `Play` takes every seat's snapshot and requires one **common snapshot**:
  the same reference, point and required seats, with an offer for each
  required seat only. It then builds each seat's **script** from that
  snapshot with a seeded pick stream: optional drafts (a different pick
  before the final one), optional **resends** (a copy with the same seq,
  delivered later), or, in race mode, a confirmation followed by a change or
  a withdrawal. With **tabs**, a seat's submissions alternate between its
  client and a second tab of the same user (`TabClient`), which a small
  per-user seq memory needs before anything is forgotten. Last, it merges
  the scripts in a seeded **arrival order** that keeps each client's own
  order.
- `CheckPoint` checks every reply against what the rules allow. A seq that
  is not newer is a duplicate, and a refusal uses up no seq. An older seq
  that changed nothing since is stale. Every arrival after the completing
  confirmation is closed. The implied batch must equal the final picks.
- `CompareSchedules` plays one game under many arrival seeds and requires
  byte-identical canonical batches and hashes. `ExhaustPoint` replays one
  point under every arrival order, on a fresh table each time.

`internal/tabletest/direct` is the in-process adapter: a model actor over the
fake that submits in the platform's acceptance order. The order is
membership, then seq dedupe in a recency list per user (`SeqClients`
bounds each user's, so small values with two tabs make stale seqs), then
the point reference, then the requirement and the offer, then the pick's
floor (the same seq a duplicate, an older one stale). Each change is cloned,
validated, persisted and only then swapped in. Its `Record` and `Restore`
are the room file this harness assumes. A WebSocket adapter plugs in as
another `Factory`.

The tests prove that the same seeds give the same trace and canonical
history. Games at three to seven seats under eight arrival seeds, with
drafts, resends, two tabs per seat and a one-client seq memory per user,
must give one history that replays to its hash. Played once more with
races, the sweep must meet every outcome, stale included. Every arrival order of small points must give one
batch: 90 at three seats, 2,520 at four seats, and a one-seat salvage
point. Races must change the batch only through arrival. Six lying tables
(a duplicate, a stale seq or a resolution reported as accepted, a false
duplicate, a false close, and a false canonical batch) must each make
`Play` fail.

### The leak oracle (`internal/tabletest/leakcheck`)

The oracle is independent of the code it checks. It builds the truth from the
setup and the batches by reading the engine's state, never its views. It
judges each document against that truth and against its own whitelist of
the view's fields, decoded with unknown fields refused. Its detectors are:

- `shuffle-id`: a card that is not a bare catalog slug, or a slug with a
  copy or position suffix such as `amber#2`;
- `deal-order`: a card list out of catalog order;
- `raw-key`: an action key anywhere but the viewer's own offer, including in
  events, the history and logs;
- `hidden-card`: a card the viewer may not know there. That covers another
  seat's hand, a sold, tucked, leftover or kept card in a public place, the
  reveal before the end or with the reveal off, and any card in a log line;
- `hash`: the seed, a game fingerprint, any seat's observation hash, or any
  hex token of twelve digits or more;
- `foreign-offer`: another seat's offered keys, salvage offer or private
  section, or a private section shown to a spectator;
- `order`: offered keys out of the engine's order, seats or history out of
  order;
- `unknown-field`: a field outside the whitelist;
- `mismatch`: a public value that differs from the truth.

A `View` or `History` document gets every detector. A whole wire `Frame`
gets the field-independent ones (`raw-key`, `hash`, `shuffle-id`), with its
own key paths allowed in `Config`. A `Log` gets the key, card and secret
checks. A `View` document's `Span` is the number of resolutions whose
events it carries, the latest last: more than one in a frame an outbox
merged for a slow connection. `TestCleanFakePasses` checks every view and history of every viewer
at every moment of forty games and requires no finding. `TestPlantedLeaksFail`
plants 26 leaks in clean documents, covering the five the platform design
names: a shuffle-derived id, a deal-ordered hand, a raw sold or tucked key,
a pending or observation hash, and an opponent's offer, plus the ordering and
whitelist cases. Each planted leak must be reported by its own detector.
`TestLogCapture` checks the in-process table's captured log, then plants a
key and a fingerprint in it. The original fake truth and whitelist remain
for the small harness. Actor tests use `FromReal`, a separate strict real-view whitelist and semantic
checks; they never take `ViewFor` or `History` as the expected answer.
Deals and accepted batches supply hands, picks, ordering and live event
entitlements. The separately reviewed permitted observation supplies private
pile-look memory and inference facts. Rich-field canaries contaminate seen
hands, pile looks, deductions, payment witnesses, science and nested hashes.
String classification is memoized independently of path exemptions; a
regression sends the same key through an allowed and a forbidden path.

### Failure injection (`internal/tabletest/faults`)

- `FileStore` writes the room record the way the room store does (temporary
  file, sync, rename). It can fail a write, or crash before or after the
  rename.
- `PendingSubsets` enumerates every pending block a restore must accept:
  each required seat with no pick, an unconfirmed pick or a confirmed pick,
  minus the block that would have resolved. `InvalidPendings` lists five it
  must refuse.
- `Permits` is a counting work pool that reports held permits and waiters.
  `Work` is a bot search that blocks until it is released or cancelled, and
  counts its starts, finishes and cancellations.

Their tests run them against `direct`:

- a storage error discards the candidate and leaves the pick unapplied;
- a crash before the rename restores the previous record, and the
  submission can be retried;
- a crash after the rename restores the resolved point, and a retried seq
  is a duplicate;
- every pending subset at turn points for three to seven seats, and at a
  salvage and a last-card point, restores and completes to the same hash;
- every invalid block, wrong hash, wrong reference and unoffered batch key
  is refused;
- cancelling bot work leaks no permit or waiter;
- a room with no human attached starts no bot search.

### `ajayeb-drive table`

`ajayeb-drive table` runs the harness from the command line, with no server.
It plays the fake game on the in-process table under `-schedules` arrival
seeds per game, with `-drafts` and `-resends` percentages and a
`-seq-clients` seq memory for each user. With `-tabs` (on by default) each
seat sends its last submission from a second tab, so a small memory forgets
the first tab's seqs and its resends meet stale. It compares the
canonical batches and hashes, optionally replays the first point under
every arrival order (`-exhaust` with an order limit), and runs the leak
oracle over every view, history, snapshot and log line (`-leaks`, on by
default). It prints one summary line per game with the outcome counts and
the hash, and exits 1 if any game failed. For example:

```
ajayeb-drive table -seats 3,4,5,6,7 -games 4 -schedules 8
ajayeb-drive table -seats 4 -games 1 -drafts 100 -resends 0 -exhaust 3000
ajayeb-drive table -seats 5 -race
```

Duel's driver is unchanged: `main` dispatches a first argument `table` to
the new subcommand, and every other invocation goes to the same `run` as
before.


## Haftgane views and permitted observations

`make test-go GO_PKGS=./internal/haftgane/...` runs the view and observation
acceptance tests with the rest of the engine. `TestViewPrivacyAndQuoteSweep`
checks every seat and the spectator under both end policies at 1,194 positions
in 60 complete games, covering every player count and side rule. Its independent
screen model collects private hands, picks, pile looks and public plays when
they become visible; it checks 7,204 serialized views, including notebook,
offer and final-reveal entitlements. Negative fixtures inject an extra public
building, another hand, a fabricated remembered hand, an unsolicited pile
offer, premature reveal/download permission and a spectator private section.
The model must reject each leak. DTO reflection and recursive value walks
require explicit JSON tags, optional pointers only, non-nil slices and no
seed or fingerprint fields. Hidden sentinels also leave every viewer's view
and normalized observation unchanged.

`TestReachableHiddenPlacePairs` and `TestReachableHiddenResolvedChoicePairs`
start from valid materialized initial deals. Different unknown sales, tucked
cards, other hands, future packets and boxed guilds leave the seat's normalized
Current/Knowledge fields, encoding, observation fingerprint, notebook statistics
and JSON view byte-identical; spectators agree too. Resolved sale/stage pairs
are replayed and undo/redone from their own setups. A test-only observation
Greedy witness exercises quotes and remembered bounds; shipping bot policies
remain the bot milestone's work. The game has no pending-pick field, so room
submission-order and draft/withdrawal privacy remain platform harness checks.

On every sweep position and every seat, both quote paths are compared with
`PaymentsState` for all 80 building costs and all 41 stage costs, including
stages not at that table: 727,210 comparisons. An independent packet-label
rotation checks that circulating bounds contain the actual consistent world
and that every certain neighbour pick matches the resolved record. Hand-built
notebook cases cover exact and at-most circulation/returns, ambiguous and
certain neighbour picks, repeated packets, both pass directions and packets
that cannot return. The later sampler gate enumerates all consistent worlds;
this slice makes no sampling-distribution claim.

`TestHistoryEqualsLiveFilteredStreams` compares the whole filtered stream at
every prefix for all seats and the spectator, through 96 complete games and
both end policies. The run includes 34 private pile looks and 44 kept-card
points. A separate reachable empty-pile stage tests a skipped look. History
includes Age I's start and never retroactively reveals private event identities.
A partial-history scenario must fail closed rather than invent remembered cards.

`TestHaftganeViewFixtures` pins seven compact raw JSON game bodies (65,633 bytes
including newlines) under `internal/haftgane/rules/testdata/views/`, spanning
sides, ordinary seat/spectator, discard build, kept card and both end policies.
It also decodes them into the exported DTOs for platform consumption. After
reviewing an intentional shape change, regenerate through the guard with
`UPDATE_HAFTGANE_VIEWS=1 make test-go GO_PKGS=./internal/haftgane/rules GO_RUN=TestHaftganeViewFixtures`.
`TestGallerySpecialCases` does the same for the dev gallery's special-decision
samples (`web/src/lib/dev/gallery/haftgane-special-cases.json`), comparing by
value because Prettier formats that file
([special-decisions.md](haftgane/special-decisions.md) has the recipe).
`make gen` generates the separate Haftgane TypeScript DTO file. Package-local
view and bot import/shape guards have explicit negative cases; the bot package
currently reserves its observation-only boundary, with policies added later.

`TestViewCapacityBytes` measures a conservative serialization stress with 217
choices and seven-unit witnesses. It is not a reachable-city maximum or a cap.
`BenchmarkCloneSeven`, `BenchmarkObserveSeven` and `BenchmarkViewForSeven` record
seven-seat timings and allocations on the same Age III position, with no timing
threshold. Current sizes and measurements are in [architecture.md](architecture.md).


## Haftgane rules gate

`TestRulesGateSweep` defaults to eight seeds for each of five player counts,
three side options and two acceptance policies (240 complete games). Set
`HAFTGANE_GATE_GAMES=2000` for the milestone gate's 60,000 games:

```sh
HAFTGANE_GATE_GAMES=2000 python3 tools/heavy.py run --kind test -- \
  go -C server test -count=1 -v -timeout 90m \
  -run '^TestRulesGateSweep$' ./internal/haftgane/rules
```

Every point checks conservation, caches, history shape and a literal coin
ledger; every game's result is checked against the raw-card scoring reference
and its accepted batches replay exactly. Counts name each effect, free reason,
free condition, point, event, board/side/stage, kept-card action and skipped
pile path. The full run reached 1,183,834 points, all kinds and all 41 stages,
7,036 empty-pile skips and 14 all-duplicate skips. Observed maxima were 72
choices per seat, 10 payments per construction, 29 events, 25 points, 20 built
cards in a city and 126 cards in the pile. The catalog/structure proofs in
`TestRulesGateDimensions` bound these dimensions independently of sampling.

`TestRulesGateReplayGoldens` pins 75 cases in
`internal/haftgane/rules/testdata/replays/replays.jsonl` (one short case per
line, about 158 KB): ten complete natural games; all 41 stages; three sixth-turn
Babylon cases; twelve Halikarnassos empty/duplicate/sixth-turn cases; seven
Olympia windows; shared victory and the coin tie-break at three seats.
Named explicit scenario builders are bound by a full initial-value SHA-256.
Each frame stores its readable point/seat/keys and SHA-256 hashes of every
seat's offers, the complete resulting game value and the ordered raw events.
The scenarios are development roots, including deliberate partial histories,
not production room files or claims of natural reachability. Existing raw view
fixtures cover readable DTO shapes. On a mismatch the test names the first
case/frame and writes the actual JSONL stream to a temporary file for diffing.
After inspecting an intentional change, regenerate with:

```sh
UPDATE_HAFTGANE_REPLAYS=1 make test-go \
  GO_PKGS=./internal/haftgane/rules GO_RUN=TestRulesGateReplayGoldens
```

Run `FuzzPlayout`, `FuzzKeys` and `FuzzPayment` separately for ten minutes each,
serially, under heavy kind `fuzz`. For example, substitute each target name:

```sh
python3 tools/heavy.py run --kind fuzz -- go -C server test \
  ./internal/haftgane/rules -run '^$' -fuzz '^FuzzPlayout$' \
  -fuzztime 10m -parallel 2 -timeout 15m
```

`FuzzPlayout` consumes bytes into valid setups and offered batches, checking
invariants, the state-only core, the coin ledger, replay and atomic malformed-key
refusal. The independent payment oracle compares complete frontiers, quote
minima and every physical witness. It removes only permutations of identical
unused supplier masks, preserving duplicate capacity. A 1,000-world test
compares this symmetry reduction directly with the original physical traversal.
Two dense seeds retain the cases that previously tripped Go's ten-second
instrumented fuzz watchdog. `make test-haftgane-payment-exhaustive` still checks
all 202,448,192 small-universe inputs and remains part of `verify-full`.
`TestScienceOracle` separately enumerates ordered science-symbol assignments.

Measure rules/catalog statement coverage with a guarded `go test -count=1
-coverprofile=... ./internal/haftgane/...`; both must reach 90% (the rules gate measured 91.9% for each). Run the
seven-seat and payment benchmarks with `-benchmem`, then compare with the
published targets in [architecture](architecture.md). Timings are recorded,
not flaky unit-test thresholds. `TestRulesGateViewByteBound` fills every DTO
section at once, checks the conservative 604,948-byte JSON ceiling, and rejects
unknown dimensions or widths. It is not a transport cap or a reachable-view
claim. The view redaction, history and noninterference suite remains mandatory.

The Duel regression proof includes its RNG/setup/replay goldens and twelve
fixed-seed `ajayeb-sim match -a greedy:1 -b random -games 12 -seed 1234567 -p 1`
game logs compared byte for byte before and after the slice. The independent
FULL review of this gate supplies M15's rules audit; automated checks do not
substitute for it.


The approved views-review follow-ups are durable regression tests.
`TestHistoryBatchZeroContract` checks every side option and live prefix: entry 0
is `rules.HistoryStartPoint` (`"game-start"`), with an Age I start for day/random
sides or an empty event list for chosen sides; entry i is batch i. Partial
scenario history still fails closed.
`TestWholeGameGuildSwapNoninterference` compares whole-game reachable pairs
for every viewer who has not seen the swapped guild, including the spectator.
It checks view JSON, observations, fingerprints, statistics, History and Legal,
rejecting public replay divergence. The recorded sweep covers 120 pairs,
2,336 positions and 11,791 equal viewer frames, with 2,261 witness frames
excluded. `TestNotebookAgainstActualWorld` checks notebook bounds against
independently rotated actual packets and the future hands that really return:
1,032 deductions, 1,472,175 Returning bounds and 145,400 circulating bounds.


## Haftgane bot tournaments

The baseline policy tests cover observation-only inputs, all player counts
and decision kinds, exact offered-key fallback, deterministic fixed work,
concurrent repeatability, mid-ranking cancellation and paired forbidden-data
changes. Their seeded sweeps and opt-in smoke are documented in
[the Haftgane AI page](haftgane/ai.md).

`TestHATournamentSyntheticRewards` independently pins score/coin ordering,
fractional wins and tied rank reward. The tournament tests also check fixed
deals and side strata, every focal/offset/reflection treatment, entropy-root
separation, repeated keys/work, incomplete or changed comparison/game sets,
corrupt digests, changed policy versions, truncated plain/gzip streams,
missing/duplicate shards and cancellation without a complete footer.
An exact-threshold test prevents floating-point roundoff from turning an
exact target into failure or a zero paired difference into positive evidence.

```sh
make test-go GO_PKGS=./cmd/ajayeb-sim GO_RUN='^TestHATournament'
make test-race RACE_DIRS='cmd/ajayeb-sim internal/haftgane/bot'
```

Run long experiments serially through the guarded commands in
[the simulator guide](haftgane/ai.md#seat-balanced-simulator). A complete
release tally requires all declared blocks, replays every accepted batch and
keeps a deal's rotations together for its 95% block-bootstrap interval.
A partial shard or short development run cannot pass the release flag.
The first fixed preset passed all 25 targets over 100 held-out blocks at
each N, with 30,600 complete unique games and no normal fallback.

Duel compatibility includes byte-identical help, weights, replay, tally and
twelve full fixed-seed game logs. The match/levels/bench reports and match
JSON are byte-identical after removing only the variable timing/heap
measurement fields; engine outcomes, action counts, iteration counts,
configuration and statistical report text stay pinned.

## The real Haftgane engine contract

`internal/tableroom` now runs `haftgane/rules`, including its actor sweeps,
fuzzer and real WebSocket tables at three to seven seats plus a spectator.
The original `tabletest/direct` fake remains a small acceptance-model test;
`tabletest/real` independently applies the same platform acceptance order to
real rules for the actor's differential tests. Neither harness enters the
production binary.

`TestEngineContractSidesAndReplay` checks day, random and chosen sides at
three and seven seats, a private draft across restart, complete games under
interleaving, and identical replay projections/history for every viewer.
`TestEngineContractAbilityOrder` finds a natural replayable seven-seat game
(seed 0): a sixth-turn discard build precedes Babylon's last-card point,
then the next Age opens. Both intermediate schema-2 records replay.
`TestEngineContractLegacyRules` refuses fake rules version 1000 before real
replay and refuses unknown side rules. Existing compatibility tests preserve
unknown fields on hash refresh and leave unsupported schemas/formats alone.

`TestLobbySidesStartAndRestore` exercises the additive `options.sides` through
cold lobby load and actual Start. Missing old values mean day sides;
`choose` opens a side point, while `day` and `random` deal Age I immediately.
Rematches retain the effective side rule. The lobby-screen control can send
`choose`, `day` or `random` through the generated `LobbyOptions` type.

`TestEngineContractMeasurements` reports retained seven-seat heap (32 live
tables, seven humans and a spectator, empty outboxes, cached real views) and
last-confirmation transaction time, including storage, all eight projections
and driver dispatch/drain overhead. These are observations, not capacity or
latency gates. See [the contract](haftgane/engine-contract.md) for a sample.
Run it with `make test-go GO_PKGS=./internal/tableroom
GO_RUN=TestEngineContractMeasurements GOFLAGS=-count=1`.

Race sweeps give every real game its own bounded context. The differential
uses two minutes per game, including all viewers, to accommodate race and
memory-pressure overhead. The socket sweep
uses five minutes per seat count, including three games and privacy checks;
the full lobby/game/rematch scenario uses five minutes per game. The merged
frame test retains every assertion and drains at 2, 3, 7, 31 and 127 pushes,
so cosmetic traffic cannot keep its multi-resolution canary unreachable.


## Fixed-preset Haftgane room bots

The actor uses the released bot package with frozen versioned work and the
existing permit pool. `TestFixedPresetsRejectUnsupportedWork` corrupts each
version, algorithm, allowance and bound independently. No unsupported row may
be interpreted as the current default. The table's three level presets freeze
before any bot starts, including levels selected by later switches.

`TestBotEveryPendingSubsetRestarts` cold-loads every subset of six accepted bot
picks, preserving every full work row and searching only missing seats. It also
checks human absent, drafted and confirmed variants. The sole fully confirmed
case must already have resolved and is rejected as an open point.
`TestRoomPolicyRepeatabilityAndForbiddenFacts` exchanges real future packets
without changing permitted inputs and compares complete decisions/counters,
then changes another seat's private pending pick. `TestReleasedRoomPoliciesAtSideAndAbilities`
checks actual actor policy calls at side, discard-build and last-card points.

`TestBotSchedulesAndRestartSameBatch` compares actual accepted keys and
complete counters between human-first play and a private draft, three bot
commits, restart and human-last play. The lifecycle tests cover kept-pick
switches, undo equality, late messages
across actor incarnations, unrelated role revisions, cancelled cleanup before
replacement admission, queued waiter removal, active cancellation and a safe
persisted watchdog reason. Full tables exercise zero to six starting bots and
seven-bot continuation. Measurements and their scope are in
[room bots](haftgane/room-bots.md); ordinary workloads require zero watchdogs.
