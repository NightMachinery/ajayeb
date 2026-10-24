# Internal simple-bot room evidence

This records the baseline milestone, not the public Scholar release or a
capacity decision. [AI policies and tournaments](ai.md) retains the published
held-out intervals. [Room bots](room-bots.md) describes acceptance, persistence
and cancellation. No policy, budget, scheduler or admission limit was changed
for these measurements.

## Frozen identities and visible names

The released Random, Apprentice and Greedy all use policy version 1 and preset
version 1. Policy version fixes the input/tape contract, action grouping,
ranking and deterministic decision procedure. Preset version fixes evaluation
constants, softmax temperature and fixed work. Apprentice's temperature is
1,500 milli-VP. Every decision completes one pass with at most 217 offered
choices, 12 cost probes and 2,616 quote calls. An allowance is per bot, never
shared or divided among six bots.

Rooms persist preset set `haftgane-bot-2`, both numeric versions, algorithm,
room code, exact allowance and bounds in each effective row. They also freeze
all three level mappings at Start. Changing semantics requires a new relevant
version, explicit support or rejection of persisted rows, determinism/privacy
checks and new strength/latency evidence under that exact live preset. A
constant or mapping change cannot silently take over an existing game's rows.
Rules version 1 and table record format 2 are separate compatibility identities.
Unsupported development records archive rather than acquire current defaults.

The internal wire levels, under preset set 2, are:

- `easy`, visible persona Apprentice: Apprentice.
- `normal`, visible persona Scholar: Greedy, a stand-in until Scholar.
- `hard`, visible persona Master: Greedy, a stand-in until Scholar.

The Apprentice label now plays the policy of that name. This milestone
implements neither a Scholar policy nor a Master policy, so those labels
play Greedy and are equally strong for now. Random is a simulator baseline
only (D46) and no room level. Public release remains gated on the sampler and
Scholar. The order is supported by the published held-out results: replacing
Apprentice with Greedy improves paired rank in mixed fields with the complete
interval above zero. The complete means, intervals, inverse comparisons and
self-play checks remain in [AI](ai.md#first-held-out-tournament). The room
row uses the same full policy work as that tournament.

Preset set 1 (`haftgane-bot-1`) mapped `easy` to Random, `normal` to
Apprentice and `hard` to Greedy. The room measurements below ran under set 1
and were not rerun for set 2. Set 2 changed only the room mapping, not any
policy, allowance or bound, and the tournament results are unchanged. So the
evidence still covers every live preset: set 1 already measured six-Apprentice
and six-Greedy tables with the same allowance and bounds. A table
frozen under set 1 archives as `replay-failed` when it loads
([room bots](room-bots.md#levels-and-frozen-work)).

## Rerun the measurements

Run one command at a time and wait for its completion notification. These
opt-in tests write safe aggregate JSON lines, never views, submitted keys,
observations, fingerprints or player data. Keep captured logs outside the repo.

For every room-case figure below, including its phase breakdown, allocations,
serialization, finished cold load and fallback count:

```sh
HEAVY_NOTIFY=auto python3 tools/heavy.py run --kind test --wait 3600 -- \
  env M17_ROOM_EVIDENCE=1 go -C server test -count=1 -v -timeout 20m \
  -run '^TestRoomEvidence$' ./internal/tableroom
```

For the complete pending-subset cold-start distribution:

```sh
HEAVY_NOTIFY=auto python3 tools/heavy.py run --kind test --wait 3600 -- \
  env M17_ROOM_EVIDENCE=1 go -C server test -count=1 -v \
  -run '^TestBotEveryPendingSubsetRestarts$' ./internal/tableroom
```

For all mixed-load figures:

```sh
HEAVY_NOTIFY=auto python3 tools/heavy.py run --kind test --wait 3600 -- \
  env M17_ROOM_EVIDENCE=1 go -C server test -count=1 -v -timeout 20m \
  -run '^TestMixedRoomEvidence$' ./internal/tableroom
```

For the eight-role retained-cache figure and the exceptional fallback check:

```sh
HEAVY_NOTIFY=auto python3 tools/heavy.py run --kind test --wait 3600 -- \
  go -C server test -count=1 -v \
  -run '^TestEngineContractFinishedMemory$|^TestBotWatchdogHasDurableSafeReason$' \
  ./internal/tableroom
```

[AI's guarded tournament commands](ai.md#guarded-commands) reproduce all published
strength intervals and simulator time/heap figures. The earlier development
smoke and microbenchmark also retain their own commands there. No extra
Makefile target is required.

## What the clocks and memory count

The non-race sample was taken on 2026-10-24 with Go 1.26.2, darwin/arm64,
eight logical CPUs and GOMAXPROCS 4. Other host activity was uncontrolled.
Quantiles use nearest rank, without interpolation or a latency confidence
interval. A single observation is reported as a value, not evidence about a
population tail.

Each standalone case has seven seats, a real file store, one shared pool
permit, its attached human and one spectator. Human choices use the first
actual offered key. The all-bot continuation retains its resigned human as a
watcher. Ability cases start from naturally replayable prefixes, then finish
the game; prefix picks are excluded from the new-pick count.

- Active time is the actor's BotReport admitted-work clock. It includes the
  diagnostic hook and timeout bookkeeping; it is wall time, not pure CPU time.
- Pool wait is BotReport's acquisition interval. Pool hold is a separately
  sampled hook-to-report bracket: after acquisition and deadline setup, before
  choice, through permit release to entry into the report callback. It omits
  the acquisition-to-hook prefix and includes the release-to-callback suffix.
  It is an estimate of permit hold, not an exact independently instrumented
  Acquire/Release duration. The two clocks must not be subtracted as CPU facts.
- Accepted-to-receipt starts after the pick's durable SaveGame and ends when
  the spectator's resolved frame is drained. The resolving pick starts at its
  resolving save. Earlier confirmations include waiting for the remaining
  picks. This is sampled server-outbox receipt, with possible latest-state
  coalescing, not network or displayed-animation latency. A cardinality check
  requires one receipt sample for every newly accepted pick.
- Human-submit time includes synchronous actor processing, store work,
  publication and the in-process driver's drain. Store-save time is emitted
  separately. Their durations overlap and must not be added.
- Serialization is an extra JSON marshal of each drained frame, separately
  timed after receipt. Sizes include the whole frame. It measures the two
  actual connections, not an invented frame for each bot seat. Transport I/O,
  compression, browser parsing and rendering are absent.
- Allocated bytes and allocation counts are process-wide deltas for one game,
  including persistence, diagnostic maps/slices, extra marshals and the driver.
  Net retained heap is a post-GC delta after discarding captured frames. It is
  noisy, includes harness state, and is neither an isolated actor size nor a
  peak. Cases have one observation each.
- Cold finished load includes constructing a fresh manager/store registry,
  replaying the file and rejoining both clients with histories. Pending-subset
  timings cover cold start plus joins at the opening point, with accepted
  picks and frozen rows checked unchanged. They exclude shutdown and file
  preparation. They are not a mid-game restart distribution.

There is no separate Haftgane room-work admission queue in this baseline, so
its wait is not applicable and is not reported as a measured zero. Browser
grouped presentation delay is unmeasured. Choosing the release room-work
concurrency setting still belongs to the capacity slice.

## Standalone room results

Within each item, active and accepted-to-receipt triplets are p50/p95/p99 in
milliseconds. Pool wait p95 is in microseconds. Hold and serialization figures
use their explicitly defined brackets above. A case's final cold load is one
observation. These observations do not establish a policy latency ordering.

These cases ran under preset set 1, when the first three were named for the
policy their level played. They are now named for their levels: `six-easy`
plays Apprentice, and `six-normal` and `six-hard` play Greedy. The mixed
cases rotate the three levels across seats, so under set 1 they included
Random seats; under set 2 every seat is Apprentice or Greedy.

- `six-random`: 108 ordinary bot decisions; active 0.029/0.081/0.220 ms.
  Pool wait p95 7.292 microseconds;
  hold bracket p95 0.046 ms.
  Accepted-to-receipt (127 picks) 20.639/42.426/44.469 ms.
  Whole-game allocation 37,281,880 bytes in 324,817 allocations;
  net retained delta 151,136 bytes.
  225 frames: bytes p50/p95/max 7,195/12,531/15,728;
  marshal p50/p95/p99 51.917/127.792/195.833 microseconds.
  Cold finished start and join 3.919 ms; normal fallbacks 0.
  `discard-build` separately: 1 calls, active median/max 0.031/0.031 ms, pool wait max 0.834 microseconds, hold bracket max 0.022 ms. This small phase sample does not estimate a stable tail.

- `six-apprentice`: 108 ordinary bot decisions; active 0.243/0.611/0.683 ms.
  Pool wait p95 1.625 microseconds;
  hold bracket p95 0.600 ms.
  Accepted-to-receipt (127 picks) 24.472/52.566/67.147 ms.
  Whole-game allocation 37,466,160 bytes in 322,518 allocations;
  net retained delta 140,168 bytes.
  241 frames: bytes p50/p95/max 7,131/13,764/16,230;
  marshal p50/p95/p99 48.750/121.292/917.667 microseconds.
  Cold finished start and join 3.685 ms; normal fallbacks 0.
  `discard-build` separately: 1 calls, active median/max 0.593/0.593 ms, pool wait max 0.500 microseconds, hold bracket max 0.583 ms. This small phase sample does not estimate a stable tail.

- `six-greedy`: 108 ordinary bot decisions; active 0.250/0.752/0.971 ms.
  Pool wait p95 1.709 microseconds;
  hold bracket p95 0.737 ms.
  Accepted-to-receipt (127 picks) 23.833/48.062/62.512 ms.
  Whole-game allocation 37,934,344 bytes in 323,174 allocations;
  net retained delta 106,152 bytes.
  243 frames: bytes p50/p95/max 7,074/12,490/16,787;
  marshal p50/p95/p99 48.708/127.416/1039.791 microseconds.
  Cold finished start and join 3.800 ms; normal fallbacks 0.
  `discard-build` separately: 1 calls, active median/max 0.373/0.373 ms, pool wait max 0.292 microseconds, hold bracket max 0.370 ms. This small phase sample does not estimate a stable tail.

- `six-mixed-choose`: 108 ordinary bot decisions; active 0.191/0.529/0.649 ms.
  Pool wait p95 1.458 microseconds;
  hold bracket p95 0.500 ms.
  Accepted-to-receipt (134 picks) 20.634/43.845/50.984 ms.
  Whole-game allocation 40,264,888 bytes in 342,290 allocations;
  net retained delta 77,256 bytes.
  259 frames: bytes p50/p95/max 6,452/12,393/16,821;
  marshal p50/p95/p99 46.709/112.209/216.417 microseconds.
  Cold finished start and join 4.428 ms; normal fallbacks 0.
  `sides` separately: 6 calls, active median/max 0.077/0.151 ms, pool wait max 7.083 microseconds, hold bracket max 0.114 ms. This small phase sample does not estimate a stable tail.
  `discard-build` separately: 1 calls, active median/max 0.659/0.659 ms, pool wait max 0.375 microseconds, hold bracket max 0.652 ms. This small phase sample does not estimate a stable tail.

- `all-bot-continuation`: 126 ordinary bot decisions; active 0.145/0.348/0.475 ms.
  Pool wait p95 1.000 microseconds;
  hold bracket p95 0.341 ms.
  Accepted-to-receipt (126 picks) 21.235/43.597/49.514 ms.
  Whole-game allocation 36,607,280 bytes in 321,834 allocations;
  net retained delta 90,232 bytes.
  257 frames: bytes p50/p95/max 4,453/6,754/12,780;
  marshal p50/p95/p99 36.958/82.625/290.459 microseconds.
  Cold finished start and join 3.622 ms; normal fallbacks 0.

- `discard-build`: 24 ordinary bot decisions; active 0.100/0.349/0.353 ms.
  Pool wait p95 1.542 microseconds;
  hold bracket p95 0.322 ms.
  Accepted-to-receipt (29 picks) 21.647/48.603/48.733 ms.
  Whole-game allocation 12,396,552 bytes in 83,370 allocations;
  net retained delta 64,176 bytes.
  53 frames: bytes p50/p95/max 8,295/15,963/43,718;
  marshal p50/p95/p99 56.458/183.291/421.083 microseconds.
  Cold finished start and join 3.400 ms; normal fallbacks 0.
  `discard-build` separately: 1 calls, active median/max 1.285/1.285 ms, pool wait max 3.291 microseconds, hold bracket max 1.270 ms. This small phase sample does not estimate a stable tail.

- `last-card`: 72 ordinary bot decisions; active 0.105/0.661/1.415 ms.
  Pool wait p95 2.125 microseconds;
  hold bracket p95 0.633 ms.
  Accepted-to-receipt (87 picks) 21.490/46.351/75.846 ms.
  Whole-game allocation 30,546,808 bytes in 236,391 allocations;
  net retained delta 82,536 bytes.
  159 frames: bytes p50/p95/max 8,497/12,231/20,083;
  marshal p50/p95/p99 52.917/133.125/1465.042 microseconds.
  Cold finished start and join 4.591 ms; normal fallbacks 0.
  `last-card` separately: 3 calls, active median/max 0.035/0.035 ms, pool wait max 9.542 microseconds, hold bracket max 0.027 ms. This small phase sample does not estimate a stable tail.

All standalone cases completed without a watchdog fallback or cancelled policy.
The forced watchdog fixture separately proves the durable `watchdog` reason
and exact Greedy fallback; it does not contribute ordinary latency samples.

Across all 64 opening pending subsets, cold-start-plus-join p50/p95/p99 was
1.451/13.466/21.899 ms. Every accepted bot pick and complete work row survived;
accepted bots were not searched again. Human none/draft/confirmed compatibility
variants also retain their assertions, without separate time claims.

The eight-table all-human seven-seat cache fixture, after all eight viewer
roles resynced and outboxes drained, retained 664,963 heap bytes per table.
Cached views and histories serialized to 468,707 bytes per table; the largest
one-role view plus history was 61,513 bytes. This is a separate cache-occupancy
experiment, not the two-connection six-bot room footprint or a frame envelope.
It agrees in scale with the earlier opening/finished comparison in
[the engine contract](engine-contract.md#projection-and-limits). No caching
change was made.

## Mixed Duel and Haftgane load

Four matched deal trials alternate baseline/mixed order. Each cohort plays
eight complete Duel games in total, two real Easy rooms at once, with their
full released three-determinization preset. Both cohorts disable the optional
think delay and advisor; other room semantics and fixed seeds are the same.
The mixed cohort adds one seven-seat, six-Greedy-bot table per trial, choosing
sides first. It runs concurrently until both Duel games finish, then closes.
These background tables may finish only part of their games; the standalone
cases above provide full-game coverage. Their partial accepted-pick receipts
are counted only after resolution. The shared pool has one permit, with
separate participant address keys. No scheduling or budget was tuned.

Duel bot latency here means factory-dispatch to policy completion, including
observation construction and pool wait but excluding subsequent commit and
presentation delay. The human-action clock runs from Manager.Act submission
to its actual outbox ack. The wrapper returns the original bot's answer.
Duel transport JSON encoding is not included. The background Haftgane frames
are separately marshaled as in the standalone probe.

- `duel-only`: 261 human action acks, p50/p95/p99 8.763/15.352/44.513 ms; 279 bot completions, dispatch-to-complete 0.541/2.834/4.946 ms; active Choose 0.357/2.100/3.402 ms.

- `mixed-duel`: 261 human action acks, p50/p95/p99 12.473/22.061/27.657 ms; 279 bot completions, dispatch-to-complete 0.699/4.331/9.370 ms; active Choose 0.407/3.118/6.727 ms.

The concurrent Haftgane sample had 181 completed ordinary
bot decisions: active 0.269/1.283/5.327 ms,
pool wait 0.001/1.479/11.007 ms, and hold bracket
0.253/0.900/5.313 ms. Its 227
resolved accepted picks reached the watcher in 45.307/98.906/128.513 ms.
The ordinary fallback count was zero; no Duel policy error was reported.
Side and ability breakdowns remain in the aggregate JSON output.

Mixed load raised the observed Duel bot p95 and p99 and human ack p50 and p95.
Human ack p99 was lower in this final sample; the baseline contained a long
outlier. An earlier independent run had the opposite p99 ordering. Four
paired trials and these small tail counts cannot establish a stable p99
regression estimate or an admission limit. This is evidence for host/load
calibration, not evidence that contention disappeared or that a release SLA
passed.

## Remaining evidence and release limits

The internal policies and all normal acceptance workloads have zero observed
watchdog fallback. No unresolved implementation failure remains in this
measurement slice. The final run passed Go lint, Duel room/bot regressions,
the simulator, the pure Haftgane bot suite, all opt-in measurements and the
64-subset restore checks. The new harness also passed under race detection.

Unmeasured here: exact permit acquisition/release timestamps, browser grouped
presentation delay, transport/network latency, peak RSS or isolated per-room
allocation, a stable cold mid-game restart distribution, mixed load with Duel
Medium/Hard or advisor work, a production host load sweep, and sampler/Scholar
strength and work. The public labels do not fill those gaps. Spectator bursts,
slow clients, backups, release configuration and a selected room-work limit
remain capacity-release work. No release difficulty or creation flag is enabled
by this evidence.
