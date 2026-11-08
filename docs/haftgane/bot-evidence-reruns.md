# Rerun the release bot evidence

These recipes accompany [release bot evidence](bot-evidence.md), which records
the measured versions, outcomes and remaining release gates.

Run from the repository root, after `make install-offline` once for a new
worktree. Submit one heavy job at a time, end the turn and wait for its
completion notification. Queue time is extra; rough durations below are local
observations or estimates, not deadlines. Keep outputs outside the repository.
Use pinned Go 1.26.3 for current reruns; the old Go 1.26.2 baseline clocks and
pre-optimization timings cannot be reproduced exactly by today's binary.

**Scholar strength and simulator time/memory/fallbacks:** about 8.1 hours for
the original full gate, plus build and tally. Preserve one binary for all
shards. These commands reproduce epoch one for audit, not fresh acceptance
after retuning. A changed candidate needs a newly declared epoch. A current
build reproduces the recipe, not the preserved VCS-stamped binary hash listed
in calibration. First build:

```sh
GOTOOLCHAIN=go1.26.3 HEAVY_NOTIFY=auto python3 tools/heavy.py run \
  --kind build --wait 3600 -- go -C server build \
  -o /tmp/ajayeb-scholar ./cmd/ajayeb-sim
```

After it completes, submit each shard separately, changing `0/10` through
`9/10`, with the same private output directory and all other flags unchanged:

```sh
GOMAXPROCS=1 HEAVY_NOTIFY=auto python3 tools/heavy.py run \
  --kind bench --wait 3600 -- /tmp/ajayeb-scholar scholar-calibrate \
  -partition heldout -epoch 1 -blocks 100 -counts 3,4,5,6,7 \
  -profiles 2x8 -sampler-checks 1048576 -suite all -frozen -load 0 \
  -out /tmp/scholar-heldout-audit -shard 0/10
```

Once all 500 count-blocks exist, tally with the same preserved binary:

```sh
GOMAXPROCS=1 HEAVY_NOTIFY=auto python3 tools/heavy.py run \
  --kind test --wait 3600 -- /tmp/ajayeb-scholar scholar-calibrate \
  -mode tally -partition heldout -epoch 1 -blocks 100 -counts 3,4,5,6,7 \
  -profiles 2x8 -sampler-checks 1048576 -suite all -frozen -load 0 \
  -out /tmp/scholar-heldout-audit
```

The command prints aggregate JSON; save it privately. Run/resume validates
existing block identities and replays keys; a different binary requires a new
output directory. [Calibration reproduction](scholar-calibration.md#running-and-resuming)
also gives development, validation and approximation/noise probes.

**Scholar room waits/watchdogs and loaded cancellation:** about three minutes
in the recorded run (142 s actor probe plus 38 s cancellation). This measures
actual accepted Scholar games and the full 176-call cancellation fixtures:

```sh
GOTOOLCHAIN=go1.26.3 GOMAXPROCS=1 HEAVY_NOTIFY=auto \
  python3 tools/heavy.py run --kind test --wait 3600 -- \
  env HAFTGANE_SCHOLAR_MEASURE=1 HAFTGANE_SCHOLAR_WORKER=1 \
  go -C server test -count=1 -p 1 -v -timeout 20m \
  -run '^(TestScholarCancellationMeasurements|TestScholarWorkerWatchdogMeasurements)$' \
  ./internal/haftgane/bot ./internal/tableroom
```

**Current decision allocations:** seconds to a minute, excluding queue/build.
This reruns both the baseline and rollout benchmarks; current optimized values
supersede historical allocation figures:

```sh
GOTOOLCHAIN=go1.26.3 GOMAXPROCS=1 HEAVY_NOTIFY=auto \
  python3 tools/heavy.py run --kind bench --wait 3600 -- \
  go -C server test ./internal/haftgane/bot -run '^$' \
  -bench '^(BenchmarkSevenSeatPolicy|BenchmarkAgeDecision)$' \
  -benchmem -benchtime=300ms -count=1
```

**Sampler support, distribution and bound:** the long replay sweep took
800.52 s, about fourteen minutes; the expanded conditional support and stop
probe took about four minutes. [Sampler commands](sampler.md#measurements-and-reproduction)
give the 2,000-games-per-count sweep/benchmark, 60-second fuzz and expanded
support/poll-gap probes. To repeat the published 2,048-tape enumerable cases
and 12,000-tape constrained distribution (allow a few minutes):

```sh
GOTOOLCHAIN=go1.26.3 HEAVY_NOTIFY=auto python3 tools/heavy.py run \
  --kind test --wait 3600 -- env HAFTGANE_SAMPLER_GAMES=1 \
  go -C server test ./internal/haftgane/rules -count=1 -v -timeout 10m \
  -run '^TestSampler(EnumerableReplay|ConstrainedDistribution)$'
```

**Apprentice strength and room diagnostics:** use the exact
[baseline shard/tally commands](ai.md#guarded-commands) (budget tens of minutes
to an hour for the full 30,600-game suite) and
[room measurement commands](simple-bots-evidence.md#rerun-the-measurements)
(budget several minutes for room cases, pending subsets, mixed Duel load and
retained-cache/watchdog fixtures). The current harness uses set 3; it cannot
recreate the historical Random/Apprentice/Greedy room cohort. There is no
current environment switch for that cohort. The historical source is
`6e88b11f`; the baseline guide records that reproduction limit. Current room
commands can measure Apprentice/Scholar phases and mixed Duel load, but no
new results from them are claimed here. All raw Go examples pin 1.26.3; no
historical timing is promised on a changed host.

**Correctness, privacy and frozen decisions:** `make verify-quick`, submitted
as below, covers ordinary sampler/bot/room tests and unchanged Duel regression
suites. Allow several minutes. It omits opt-in timing probes and tournaments.

```sh
HEAVY_NOTIFY=auto AJAYEB_HEAVY_WAIT=3600 python3 tools/heavy.py run \
  --kind light -- make verify-quick
```

No current measurement entry point publishes an Apprentice-specific
cancellation-latency distribution, production peak RSS, browser presentation
delay or a Scholar mixed-room capacity curve. Those gaps cannot be filled by
rerunning the historical baseline or by treating a correctness test as a
latency measurement.
