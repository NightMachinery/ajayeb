# Experimental Scholar calibration

@OpenAI, 2026-11-07/08. Work in progress: no preset is selected or exposed to
rooms. The offline `scholar-calibrate` command measures explicit `AgeWork`
rows, independently of released levels. [The bot plan](bot-plan.md) defines
the gates; [the rollout guide](rollouts.md) describes the search.

## Current development evidence

Two candidate rows were screened on three development blocks per player
count: one attempt with four candidates (`1x4`), and two attempts with eight
candidates (`2x8`). Every attempt retains `MaxSamplerChecks=1048576`. Work
is per bot and is never divided by the number of bots.

The cheaper row's main-field rank reward was 0.633 at five seats and 0.635
at seven, below 0.65. The stronger row was then measured on nine development
blocks per count, using all treatments: 1,017 unique games. Its main-field
rank reward was 0.741, 0.787, 0.722, 0.652 and 0.735 at three through seven
seats. Six-seat win credit was 0.259 against the 0.267 target. Nine blocks
cannot establish the required 100-block held-out result. The first three
blocks overlap the screen and must not be counted twice.

The complete development schedule projected 7.24 hours for one 100-block
held-out gate over all counts. The range obtained from the cheapest and most
expensive observed Scholar seat-game rates was 3.88 to 15.66 hours. This is
a throughput extrapolation, not a confidence interval or an execution bound.
The coordinator accepted the mean estimate as within the eight-hour run
limit. Validation and cancellation acceptance still precede any freeze;
held-out games have not been run.

## Validation before freezing

The unchanged `2x8` row was validated on nine separate validation blocks per
count, again 1,017 unique games. Main rank reward at three through seven
seats was 0.759/0.824/0.717/0.709/0.743 and win credit was
0.593/0.583/0.400/0.352/0.397. Six-seat win credit's 95% block interval was
0.296 to 0.407, above the 0.267 target. All main paired-improvement
intervals were positive. Inverse and mixed mean improvements were positive
at every count; the six-seat inverse interval still crossed zero.

The combined paired improvement was 0.174 (95% interval 0.159 to 0.190).
These nine-block results support taking the same row to the predeclared
held-out gate, not a claim that it has passed that gate. The validation
throughput projection was 6.24 hours for the full gate, with observed-rate
range 4.09 to 8.35 hours. The validation binary preceded the final storage
optimization; final waits must use the frozen implementation.

## Experiment design and interpretation

A block fixes a physical deal, seating and board sides. The side strata
cycle through day, night and random. Each player count rotates the focal
replacement through every physical seat. Main comparisons replace one
Greedy with Scholar in an all-Greedy field. Inverse comparisons replace one
Greedy in a field of Scholars. Mixed comparisons alternate the two policies
around the ring and include every offset and both reflections under
`-suite all`. `screen` retains one offset and both reflections. Identical
lineups are played once within a block and shared between comparisons.

Reported rank reward and win credit use exact final scores, then coins, as
in the baseline simulator. Intervals resample independent deal blocks 5,000
times, retaining all correlated rotations and reflections within each block.
The combined suite gives each player count equal weight and, within each
count, each of main, inverse and mixed equal weight. One-block reports mark
intervals as unestimable rather than claiming precision. Development and
validation intervals guide selection; they are not held-out gate outcomes.

The strength gate requires one frozen row, at least 100 blocks at every
count, main-field mean rank reward >=0.65, mean win credit >=1/N+0.10,
positive paired-improvement intervals, no inverse/mixed mean regression at
any count, and a positive combined paired interval. The command's gate flag
covers strength only. It does not approve room latency, cancellation,
watchdogs, privacy, persistence or release.

Development, validation and held-out deals have separate seed namespaces
and an explicit epoch. Policy entropy is independent of setup entropy and
is indexed by logical point and seat, so a treatment cannot consume another
seat's draws. Any change informed by held-out outcomes requires a fresh
epoch. Do not inspect partial held-out outcomes: interim summaries omit
strength contrasts until every planned block is present.

## Clocks, load and memory

These measurements use Go 1.26.3 on arm64 and `GOMAXPROCS=1`, a one-thread
CPU-budget proxy on this machine, not phone hardware. The tournaments add
no competing CPU goroutine; other desktop activity is uncontrolled. There
is one guarded experiment at a time. Every bot receives the complete row.

Decision time brackets `ChooseAge`, after observation and legality. Its
context timestamps every poll, so the instrumentation is included. For the
six-bot table, seat zero is an instant Greedy human proxy and the other six
seats run Scholar. Per-point service sums those actual six decisions at the
same table. Whole-game compute time includes engine and harness work but
excludes human thought, actor persistence, pool admission, transport and
presentation. It is not an end-to-end room wait.

In the nine-block `2x8` development sample, six-bot decision p50/p95/p99 was
42.30/152.86/247.95 ms (988 calls). Serialized per-point service was
235.78/880.35/1468.42 ms (178 points). Whole-game p50/p95/p99 was
5.60/9.45/9.45 seconds (nine games). Nine games do not estimate a stable
population p99. These are measured sums, not the costliest seat multiplied
by six. Allocation counters include full-game engine/harness work;
heap-after-game samples are not peak RSS. Across the nine six-bot games,
whole-game allocation p50/max was 74,813,776/99,600,416 bytes in
875,800/1,097,799 allocations. The largest heap-after-game sample was
3,980,008 bytes. Those small live-heap samples do not remove the substantial
allocation churn or its contribution to game time.

The cancellation acceptance probe takes the largest poll gap over each
complete call, including entry/first poll and last poll/return. Its fixed
fixtures include all five player counts at turn three of all three Ages,
plus the costly seven-seat discard root, with eight tapes each: 128 calls
per load. Load one adds a continuously computing goroutine on the same Go
CPU thread. A timer-cancel probe is reported separately; a short unwind
cannot substitute for the complete-call poll gap.

Initial loaded poll-gap p99 was 50.975 ms, failing the 50 ms target. Tracing
identified sampler matching and multi-quote evaluation stretches. Polls
now split sampler matching traversals and individual evaluation quotes,
including cutoff evaluation. These polls neither charge additional sampler
checks nor change features, candidate ordering or random draws. The same
normal-GC probe still failed at loaded p99 51.857 ms (max 57.829 ms).

Two diagnostic controls then isolated runtime effects. With GC disabled in
a separate process, the unchanged probe reached loaded p99/max
30.072/30.108 ms, with zero collections and 296,706,584 allocated bytes
across all complete and cancelled calls. A synthetic loop, calibrated to
the 128 real calls' 5.143-second idle duration and polling about every
317 ns, took 10.486 seconds under the same competing goroutine. It allocated
zero bytes, had no collections, and reached p99/max 30.058/30.064 ms. These
controls support investigating allocation and GC; neither satisfies the
normal-GC gate. Scheduler trace delay totals aggregate all goroutines and
cannot by themselves locate the compute inside a long poll interval.

The allocation change uses engine-owned canonical key bytes in reusable
rollout buffers, shares the existing feature calculation with public
rankings, and reuses one bounded legal-choice buffer across all seats,
rounds, candidates and attempts in a decision. It changes neither work nor
key-indexed noise. Tests matched 5,532 old/new policy selections, reversed
offer order, and complete decisions/counters at all five player counts and
the costly root. The private selector allocates zero bytes on its initial
seven-seat fixture.

The final initial seven-seat `2x8` benchmark used 13,992 bytes and 50
allocations per decision, 55.80 ms/op over four measured iterations. Earlier
rollouts allocated about 1.2 to 1.8 MB and 15,000 to 23,000 objects per
initial decision. This benchmark is a local observation, not a full-game
latency bound.

The unchanged normal-GC probe passed after both storage changes. Across
128 complete calls per load, idle p99/max was 0.989/2.082 ms and loaded
p99/max was 30.070/52.407 ms. One loaded maximum exceeded 50 ms; the declared
p99 target passed. Separate timer-cancel p99 was 0.021 ms across 105 loaded
cancelled calls, with 23 calls completing before the timer. The maximum-gap
result remains the acceptance measure. No diagnostic GC settings were used
in this final probe. Full verification and the held-out gate remain pending.

## Measured rollout approximations

The notebook probe uses 40 checked development histories (eight per count),
with side strata and a policy that prioritizes ability stages. It tests turn
three onward and the first turn after each Age boundary. Each seat's full
observation is compared with that same observation after stripping only its
notebook. Legal offers, preference and policy entropy stay fixed. No real
opponent notebook is passed to the runtime search.

Across 2,800 seat observations and 11,200 paired policy choices, no action
changed in this sample. Cutoff estimates changed in 130 observations, by up
to 2.7 VP, and 26 of 6,160 seat-pair orderings changed. The future-score
estimate ordered 2,287 pairs differently from exact current points and
2,025 differently from the eventual scores in these histories. Future-score
terms deliberately differ from current points; this experiment quantifies
that approximation rather than validating it as an exact score predictor.
Stripping the entire notebook at every tested point also differs from the
runtime approximation, which starts opponents empty at the root and then
records their simulated observations. Zero observed policy changes do not
prove that notebook information is irrelevant.

The later-position noise probe uses turn three of all three Ages at every
player count, 64 samples per position and at most eight candidates. It
compares paired candidate-minus-Greedy variance with the sum of their two
marginal variances, representing independent draws with those marginals.
Ratios for Ages I/II/III were:

- Three seats: 0.318/0.168/1.000.
- Four seats: 0.385/0.433/undefined (both variances zero).
- Five seats: 0.630/0.190/0.890.
- Six seats: 0.478/0.711/0.156.
- Seven seats: 0.348/0.175/0.502.

The aggregate variance ratio was 0.364. Pairing reduced aggregate measured
noise, but not every position benefited. These are selected reachable
development prefixes, not a claim about every possible root.

## Running and resuming

Build one binary through the guard and preserve it for every shard. Its
SHA-256, catalog hash, exact work rows, partition/epoch, schedule, block
count, Go CPU budget and load are part of the immutable experiment identity.
Rebuilding a changed binary requires a new output directory.

```sh
GOTOOLCHAIN=go1.26.3 HEAVY_NOTIFY=auto python3 tools/heavy.py run \
  --kind build --wait 3600 -- go -C server build \
  -o /tmp/ajayeb-scholar ./cmd/ajayeb-sim
```

After the build completes, inspect a development plan before starting work:

```sh
GOMAXPROCS=1 /tmp/ajayeb-scholar scholar-calibrate -mode plan \
  -partition development -epoch 1 -blocks 9 -profiles 2x8 -suite all
```

Run one guarded shard at a time on this machine:

```sh
GOMAXPROCS=1 HEAVY_NOTIFY=auto python3 tools/heavy.py run \
  --kind bench --wait 3600 -- /tmp/ajayeb-scholar scholar-calibrate \
  -partition development -epoch 1 -blocks 9 -profiles 2x8 -suite all \
  -out /tmp/scholar-development -shard 0/1
```

`run` resumes by validating and skipping completed blocks. `-mode tally`
replays the ledger without rerunning decisions. Each block is published
atomically and exclusively, with a digest, canonical filename/identity and
all required unique lineups. Accepted batches are replayed through the
checked engine and their final outcome/hash verified before publication and
on resume. Missing, corrupt, renamed or incompatible blocks are refused.
The private ledger contains accepted keys for replay and must not be sent
in ordinary client frames or bot diagnostics.

For a future frozen held-out run, use `-partition heldout -frozen`, one
nonzero profile, `-blocks 100`, counts 3 through 7 and `-suite all`. The full
manifest has 800/1600/2200/3700/3000 unique games, respectively, or 11,300 in
total. Ten shards assign `(profile, count, block)` ordinals modulo ten and
cover ten independent deals per count each. They can run sequentially here
or on isolated CI workers with the same binary and CPU/load identity. Share
or collect immutable block files under their original names; tally only
after all 500 blocks are present. Never shorten work to meet a timer.

The opt-in diagnostics run through the guard:

```sh
GOMAXPROCS=1 GOTOOLCHAIN=go1.26.3 HEAVY_NOTIFY=auto \
  python3 tools/heavy.py run --kind test --wait 3600 -- \
  env HAFTGANE_SCHOLAR_MEASURE=1 go -C server test \
  ./internal/haftgane/bot -run '^TestScholar.*Measurements$' \
  -count=1 -v -timeout=10m
```

`HAFTGANE_SCHOLAR_TRACE=1` enables the separate expensive call-site tracer.
Scheduler tracing also perturbs timing. Diagnostic trace runs do not count
as cancellation acceptance evidence.
