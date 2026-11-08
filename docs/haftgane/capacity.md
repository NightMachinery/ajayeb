# Release capacity and safety

@OpenAI, 2026-11-08. Use **one admitted Haftgane bot-room worker**, with
the complete released `2x8` Scholar row for every bot. Creation stays
**off by default**. The owner accepted the measured host waits through the
release coordinator. Enabling production creation is a separate deploy.

## Host and clocks

The Linux amd64 host has two logical CPUs and 4,106,379,264 bytes of RAM.
Before testing, available memory was about 1.50 GB, load average 0.05 and
production had no established game connections. Go 1.26.3 binaries were
cross-compiled locally, then run with `GOMAXPROCS=2`, normal GC and niceness
10. The host built nothing. Scratch processes used free loopback ports and
separate data. A bounded supervisor checked production connections and
available memory, stopped on a player connection or less than 512 MiB
available, and terminated its own processes on success or failure.

The production WebSocket driver created two seven-seat tables with six
Scholar bots each. `TestCapacityMixedEvidence` completed two Scholar
tables while two Duel Apprentice actors continuously requested full
decisions, then replayed their completed game counts without tables.
Limit one completed 207 Duel games, limit two 13. This is a saturated
throughput comparison with different baseline sample sizes, not a
confidence interval or matched temporal trace. An earlier four-trial,
alternating short probe agreed on the limit choice, but ended before
limit-one tables reached ordinary turns.

`TestCapacityWebSocketEvidence` uses actual store, actors, codecs, JSON
and WebSockets. Cases cover cold mid-game restore, a burst of 24
spectators plus four peers that stop reading after hello/join, and seven-bot
continuation after the original human resigns but watches. These cases
also run with two continuously active Duel rooms. Private logs retain all
aggregates and commands, without collecting production-game data.

Quantiles below use nearest rank: **p50/p95/p99 in milliseconds**, unless
stated otherwise. A small sample's p99 may be its maximum. No measurement
includes browser animation or presentation delay. Active search, pool
wait, pool hold and room-work wait are separate clocks; their quantiles
must not be added together.

## Why one worker

Two complete tables plus two Duel rooms, admission limit one:

- Duel acknowledgement: **2.988/12.051/26.780**, 6,771 samples.
  Duel-only baseline: **2.935/6.013/14.128**, 6,735 samples.
- Duel dispatch-to-bot-completion: **0.917/7.377/22.339**, 7,483 samples.
  Baseline: **0.596/2.979/5.113**, 7,442 samples. Dispatch includes observation
  construction, pool wait and search, excluding the subsequent commit.
- Haftgane active search: **132.322/472.508/597.319**, 216 ordinary calls.
  Pool hold: **132.325/472.510/597.320**. Pool wait:
  **0.000595/0.001219/0.001580**. Room-work wait:
  **115.157/464.234/580.656**.
- Accepted-pick-to-spectator-outbox: **585.009/3,605.213/5,179.213**, 272
  picks, maximum **6,234.348**. This starts after durable save and includes
  waiting for remaining confirmations.

Limit two improved table reveal p99 to **2,280.668**, but Duel
dispatch-to-completion rose to **28.181/179.173/327.977**, versus baseline
**0.638/3.443/5.231**. Duel acknowledgement p99 was **35.532**, versus
**18.007**. Both table jobs could occupy the shared pool and delay Duel.
One slot preserves substantially more Duel service under load.

`-haftgane-bot-rooms=1` is the single Haftgane admission setting. A worker
takes it before requesting an address-fair shared pool permit, releases
the pool permit after one seat's decision, then yields to the next room.
A queued table holds no shared permit and runs no watchdog. Cancellation
removes either waiter and releases every acquired permit. Existing
per-room serialization, pool size and address cap still apply. No bot work
is reduced by table size and no speculative room cap was added.

## Solo, restore, abilities and delivery

Solo WebSocket trajectory, including one cold mid-game restart:

- Human-send-to-resolving-frame: **531.087/2,710.232/2,710.232**, 19 points.
  Ordinary active search: **101.950/372.038/481.658**, 108 calls;
  pool hold **101.953/372.041/481.662**.
- Pool wait p99 **0.001299**, room-work wait p99 **0.037454**. Sequential
  search dominates the seconds-long wait. Aggregates cannot partition
  the exact slowest point into individual searches and I/O.
- Cold shutdown, new stack, hello and rejoin: **7.445**, one sample.
  A confirmed pick survived when its point stayed open.
- Side search: **26.620/36.873/36.873**, six calls; last-card ability:
  **5.564/7.959/7.959**, two calls.
- Wire frame bytes p50/p95/p99/max: **9,645/13,983/15,362/15,734**;
  final reveal **13,033** bytes.

With the spectator burst and four non-reading peers, human-to-reveal was
**589.185/2,632.562/2,632.562**, 19 points. Spectator join:
**2.506/6.714/9.370**, 24 samples. Largest creator frame: **19,629** bytes;
final frame **16,928**. Non-reading peers did not stall progress. This
short run does not establish TCP buffer exhaustion or spectator capacity.

Seven-bot continuation finished without another human pick. Discard-build
search: **88.577/170.316/170.316**, two calls; last-card:
**6.328/7.799/7.799**, three calls. The original human remained attached,
as required by the participation rule. Final spectator frame: **9,558** bytes.

With two continuous Duel rooms, solo/restore human-to-reveal was
**900.228/3,668.419/3,668.419**, 19 points; with the spectator burst and
non-reading peers it was **826.904/4,263.295/4,263.295**, 19 points.
Their Duel acknowledgement p99 values were **25.674/27.475** respectively.
Cold restart/rejoin took **32.853**, one sample. Spectator joins were
**5.077/31.886/67.778**, 24 samples. Seven-bot continuation also finished
under Duel load, with ordinary-search **88.121/350.113/419.169**, 126
calls, and Duel acknowledgement p99 **26.029**, 3,117 samples.

The separate server peaked at **16,720 KiB RSS** with two Scholar games,
sampled every 100 ms (268 samples); the preceding two-Duel run peaked at
**15,488 KiB**. These are sampled peaks, not bounds. The instrumented
WebSocket test process reached 74.25 MB heap during the spectator case
and 87.81 MB in the following continuation case. These totals include
retained driver frames and preceding cases, not server-only memory.
The six-Scholar actor probe allocated about 28.24 MB over one whole game,
including store and extra frame marshals, not retained at once.

Every ordinary probe reported **zero watchdog and zero other policy
fallbacks**. Forced watchdog tests remain separate correctness fixtures.

## Wait disposition

The concurrently driven games each took **26.745/26.770 seconds for the
whole game**, at limit one, including lobby setup and driver pacing.
Their seat jobs interleaved behind the other table. These are not turn
waits. Solo p95 2.710 s and concurrent game times exceeded the coordinator's
2 s/25 s escalation thresholds, reported before acceptance. The owner
accepted the host waits at limit one. The saturated mixed reveal tail
above remains visible; this is no promise of a 2 s turn bound.

## Reduced-budget timing experiment

The requested cheaper-budget experiment was already running when the
owner accepted the full-row waits. A private source copy changed only
the consumed search row to `1x4`, one attempt and four candidates, with
the full sampler-check bound. Its temporary records were experimental;
no production policy, stored preset or CLI switch changed. Solo/restore
human-to-reveal was **152.331/751.657/751.657**, 19 points, versus the full
row's **531.087/2,710.232/2,710.232**. Under two Duel rooms it was
**268.220/1,156.773/1,156.773**, 19 points; with the spectator burst under
Duel load **308.625/1,158.146/1,158.146**, 19 points. All had zero fallbacks.
These are timing trajectories on one fixture, not strength evidence.
Earlier three-block [development screening](scholar-calibration.md#development-selection-before-freezing)
found rank reward 0.633 at five seats and 0.635 at seven for `1x4`, below
the 0.65 gate. That small screen cannot quantify a held-out strength loss.
The released `2x8` row remains unchanged.

## Release switch and stored data

`-haftgane-tables` enables three-to-seven-seat lobbies on an ordinary
server, without requiring or enabling `-dev` or `-dev-endpoints`. Default
is false. Duel's two-seat lobby is unchanged. `GET /health` reports
`haftganeCreation`; Home offers Haftgane only when it is true. Existing
rooms still load with creation disabled.

Production registers Duel, table and lobby codecs for storage, retention
and `--check-rooms`. The installer scans a candidate before stopping and
again after stopping, before backup or swap. This applies to rollback
binaries supplied through `--prebuilt`. Unknown schema-2 games,
unsupported formats or read errors cannot be forced past the scan.
Tests create real schema-2/format-6 records, check old schema-only readers
and future formats, and require unchanged bytes. Rollback staging now
precedes overwriting `.prev`, preserving the candidate that was scanned.

The shared reaper expires finished records after seven days and idle
lobbies/tables after fourteen, like Duel. Archives use the same thirty-day,
1,000-file and 64 MiB limits. Rooms backups include every game kind
unchanged, retaining at most ten backups, each at most thirty days old.
Tests verify schema-2 expiry and byte-exact Duel/table/lobby backup members.

## Rerun

The probes are opt-in and skipped during ordinary verification. Run each
command separately, letting its detached job finish:

```sh
HAFTGANE_CAPACITY=1 HEAVY_NOTIFY=auto python3 tools/heavy.py run --kind light --wait 3600 -- \
  make test-go GO_PKGS=./internal/tableroom GO_RUN='^TestCapacityWebSocketEvidence$'
HAFTGANE_CAPACITY=1 HAFTGANE_CAPACITY_MIXED=1 HEAVY_NOTIFY=auto python3 tools/heavy.py run --kind light --wait 3600 -- \
  make test-go GO_PKGS=./internal/tableroom GO_RUN='^TestCapacityWebSocketEvidence$'
HAFTGANE_CAPACITY=1 HAFTGANE_CAPACITY_ROOMS=1 HEAVY_NOTIFY=auto python3 tools/heavy.py run --kind light --wait 3600 -- \
  make test-go GO_PKGS=./internal/tableroom GO_RUN='^TestCapacityMixedEvidence$'
E2E_RELEASE_SERVER=1 HEAVY_NOTIFY=auto python3 tools/heavy.py run --kind light --wait 3600 -- \
  make e2e PW_ARGS='specs/haftgane.prod.spec.ts'
```

The last command uses an ordinary local server with creation enabled,
without fixture endpoints or predictable seeds. `@prod` specs skip unless
health advertises creation enabled. Host reruns must also repeat the
activity/memory preflight, isolated data and free-port selection, bounded
supervision and process cleanup used above.
