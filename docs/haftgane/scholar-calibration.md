# Scholar calibration evidence

@OpenAI, 2026-11-08. Scholar passed the frozen held-out strength gate.
The experimental row is fixed at two attempts, eight candidates and 1,048,576 sampler checks per attempt. The
existing 3/4 rank plus 1/4 win reward, four rollout preferences and 1,500
key-indexed policy noise are unchanged. New rooms now use this row as the
version-one Scholar preset (integration evidence below). The
offline `scholar-calibrate` command measures explicit `AgeWork` rows,
independently of released levels. [The bot plan](bot-plan.md) defines
the gates; [the rollout guide](rollouts.md) describes the search.

## Held-out strength gate: passed

The frozen epoch-one gate completed all 500 count-blocks and 11,300 unique
games: 100 independent deals at each player count, with every main, inverse
and mixed treatment. The strength results were inspected once after the
complete run. No held-out outcome was used to tune the row. These seeds are
spent; an informed policy or work change needs a fresh held-out epoch.

The chosen row is `a2-c8`: two attempts, at most eight candidates and
1,048,576 sampler checks per attempt, in full for every bot. Reward remains
3/4 rank plus 1/4 win (scale 3,360), with the balanced, economic, science and
military rollout preferences and key-indexed noise 1,500. The frozen source
is the commit "sim: freeze Scholar candidate after worker preflight"
(`a57b7c38` on master; the run used its pre-rebase twin, whose server tree
differs only in an unrelated lobby test). The binary embeds VCS stamping,
so its hash cannot be reproduced byte for byte from the merged commit. The
preserved binary SHA-256 is
`d6f5453fc239e5570f32bcaea30368a0d2c64de5ce149d9e2c687a45b632f5d2`
and catalog SHA-256 is
`613971b06c4616f0e41c4fb6515735042019432a993ccd9608ee3a1a89976ecb`.

Below, rank reward measures the focal seat's relative finish, and win credit
shares tied wins. Paired rank improvement subtracts the same seat's Greedy
result on the paired deal. The inverse candidate is an all-Scholar table,
so averaging every focal seat gives exactly 0.5 rank and 1/N win credit in
every block. Its paired improvement still measures the effect of replacing
Greedy and has a nondegenerate interval. The 0.65 and 1/N+0.10 thresholds
apply only to main-field means; inverse and mixed require nonnegative mean
paired improvement at every count. All intervals below use 5,000 deal-block
bootstrap resamples, retaining correlated seat rotations and reflections.

### 3 seats

100 independent blocks; main rank target 0.65 and main win target 0.433333.

- Main: rank 0.731667 (95% interval 0.691667 to 0.770000);
  win credit 0.590000 (95% interval 0.536667 to 0.646667);
  paired rank improvement 0.231667 (95% interval 0.191667 to 0.270000).
- Inverse: rank 0.500000 (95% interval 0.500000 to 0.500000);
  win credit 0.333333 (95% interval 0.333333 to 0.333333);
  paired rank improvement 0.096667 (95% interval 0.056667 to 0.138333).
- Mixed: rank 0.548333 (95% interval 0.528333 to 0.569167);
  win credit 0.383333 (95% interval 0.361667 to 0.405000);
  paired rank improvement 0.164167 (95% interval 0.135000 to 0.195000).

Main mean targets and the positive paired-interval gate pass. Inverse and
mixed mean non-regression gates pass; their complete paired intervals are
also above zero.

### 4 seats

100 independent blocks; main rank target 0.65 and main win target 0.350000.

- Main: rank 0.771667 (95% interval 0.740833 to 0.802500);
  win credit 0.562500 (95% interval 0.512500 to 0.615000);
  paired rank improvement 0.271667 (95% interval 0.240833 to 0.302500).
- Inverse: rank 0.500000 (95% interval 0.500000 to 0.500000);
  win credit 0.250000 (95% interval 0.250000 to 0.250000);
  paired rank improvement 0.072500 (95% interval 0.035833 to 0.109167).
- Mixed: rank 0.524167 (95% interval 0.511944 to 0.536389);
  win credit 0.269167 (95% interval 0.257500 to 0.280833);
  paired rank improvement 0.138472 (95% interval 0.118333 to 0.158333).

Main mean targets and the positive paired-interval gate pass. Inverse and
mixed mean non-regression gates pass; their complete paired intervals are
also above zero.

### 5 seats

100 independent blocks; main rank target 0.65 and main win target 0.300000.

- Main: rank 0.744500 (95% interval 0.724000 to 0.765500);
  win credit 0.442000 (95% interval 0.406000 to 0.482000);
  paired rank improvement 0.244500 (95% interval 0.224000 to 0.265500).
- Inverse: rank 0.500000 (95% interval 0.500000 to 0.500000);
  win credit 0.200000 (95% interval 0.200000 to 0.200000);
  paired rank improvement 0.068000 (95% interval 0.040500 to 0.095500).
- Mixed: rank 0.554250 (95% interval 0.536500 to 0.573000);
  win credit 0.240000 (95% interval 0.219000 to 0.262000);
  paired rank improvement 0.179500 (95% interval 0.155250 to 0.204375).

Main mean targets and the positive paired-interval gate pass. Inverse and
mixed mean non-regression gates pass; their complete paired intervals are
also above zero.

### 6 seats

100 independent blocks; main rank target 0.65 and main win target 0.266667.

- Main: rank 0.729833 (95% interval 0.707000 to 0.753167);
  win credit 0.398333 (95% interval 0.356667 to 0.440000);
  paired rank improvement 0.229833 (95% interval 0.207000 to 0.253167).
- Inverse: rank 0.500000 (95% interval 0.500000 to 0.500000);
  win credit 0.166667 (95% interval 0.166667 to 0.166667);
  paired rank improvement 0.102667 (95% interval 0.077000 to 0.127167).
- Mixed: rank 0.529100 (95% interval 0.521233 to 0.536967);
  win credit 0.175833 (95% interval 0.166667 to 0.185000);
  paired rank improvement 0.139033 (95% interval 0.125233 to 0.152600).

Main mean targets and the positive paired-interval gate pass. Inverse and
mixed mean non-regression gates pass; their complete paired intervals are
also above zero.

### 7 seats

100 independent blocks; main rank target 0.65 and main win target 0.242857.

- Main: rank 0.717262 (95% interval 0.697619 to 0.736667);
  win credit 0.358571 (95% interval 0.321429 to 0.395714);
  paired rank improvement 0.217262 (95% interval 0.197619 to 0.236667).
- Inverse: rank 0.500000 (95% interval 0.500000 to 0.500000);
  win credit 0.142857 (95% interval 0.142857 to 0.142857);
  paired rank improvement 0.094643 (95% interval 0.071905 to 0.117381).
- Mixed: rank 0.565000 (95% interval 0.551190 to 0.578333);
  win credit 0.173214 (95% interval 0.155714 to 0.190714);
  paired rank improvement 0.169940 (95% interval 0.152262 to 0.187798).

Main mean targets and the positive paired-interval gate pass. Inverse and
mixed mean non-regression gates pass; their complete paired intervals are
also above zero.

### Combined suite

Equal weight across player counts and then the three fields gives rank
0.594385 (95% interval 0.589140 to 0.599837), win credit
0.312387 (95% interval 0.304552 to 0.320673), and paired rank improvement
0.161368 (95% interval 0.152365 to 0.170322).
The combined interval is wholly above zero, passing its gate. All per-count
strength gates pass. This is an offline strength result, not room release
approval.

The run started at 01:01 and ended at 09:06 local time on 2026-11-08,
about 8.1 hours against the final 5.79-hour (about 5.8-hour) preflight
forecast. The completed summary's own full-gate projection is 8.05 hours,
above the approximate eight-hour planning limit. The forecast underestimated
the completed run; desktop load was uncontrolled, so these measurements do
not establish a cause for the difference.

## Development selection before freezing

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
limit. This was an early forecast; validation and the final implementation
measurements below preceded freezing.

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
seat's draws. In format `scholar-calibration/1` the policy roots depend on
the epoch but not the partition, so development, validation and held-out
blocks with the same index share policy tapes and differ only in their
deals. That was harmless here (only two rows were compared and the margins
are large), but a wider search should mix the partition into the policy
root under a new format version first. Any change informed by held-out outcomes requires a fresh
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

The held-out six-bot sample is the seven-seat lineup with seat zero as
an instant Greedy human proxy and six Scholars, with 100 independent games.
Quantiles are nearest-rank sample quantiles; 100 games provide limited
precision in a population p99. All times below include normal desktop load,
with no added load, normal GC (`GOGC=100`), no memory-limit override,
`GOMAXPROCS=1`, Go 1.26.3 and an Apple M2 (arm64). This is a single-thread
CPU-budget proxy, not a phone measurement.

The sequence in each line is p50 / p95 / p99 / maximum:

- Scholar decision across all seven-seat fields (191,965 calls):
  45.196 / 163.129 / 246.990 / 2,577.305 ms.
- Decision at the six-bot table (10,982 calls):
  47.457 / 178.734 / 273.930 / 625.421 ms.
- Serialized six-bot service per turn or decision point (1,982 points):
  255.711 / 1,042.702 / 1,608.108 / 3,286.261 ms.
- Total six-bot decision service over a whole game (100 games):
  6.274 / 11.016 / 11.745 / 19.832 seconds.
- Whole one-human-table game compute, including engine and harness
  (100 games): 6.281 / 11.029 / 11.765 / 19.847 seconds.

Whole-game bot service is derived by summing actual Scholar decisions in
each saved game, not multiplying a slow seat by six. The game-compute clock
also includes the instant Greedy proxy and engine/harness overhead. Neither
includes human thought, actor persistence, pool admission, network or UI.
The two game figures are different clocks on the same 100 games, not
independent samples.

For those same 100 six-bot games, whole-game allocated bytes were
2,116,496 / 2,203,888 / 2,216,384 / 2,222,680, in
8,180 / 9,162 / 9,451 / 9,513 allocations. Heap-after-game bytes were
3,690,176 / 5,215,080 / 5,501,536 / 5,774,392. These counters include engine
and harness work. Across all 11,300 unique games, allocations totalled
14,442,493,472 bytes in 61,827,957 objects; the maximum sampled heap after a
game was 22,071,560 bytes. Heap-after-game is not peak heap or RSS. Before
the storage optimization, nine development six-bot games had median/max
allocation 74,813,776/99,600,416 bytes in 875,800/1,097,799 allocations;
those older timings and allocations do not describe the frozen binary.

The cancellation acceptance probe takes the largest poll gap over each
complete call, including entry/first poll and last poll/return. Its fixed
fixtures include all five player counts at turn three of all three Ages,
plus the costly seven-seat discard root, with eight tapes each: 128 calls
per load. Load one adds a continuously computing goroutine on the same Go
CPU thread. A timer-cancel probe is reported separately; a short unwind
cannot substitute for the complete-call poll gap.

Initial loaded poll-gap p99 was 50.975 ms, failing the 50 ms target. Slow
traced intervals ended near matching and multi-quote evaluation, without
establishing where that interval was spent. Polls
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
p99/max was 30.070/52.407 ms, over the turn-three fixtures of each Age
plus the seven-seat discard root. Turn-one roots cost about twice as much,
so they are not covered; the preset slice adds them. One loaded maximum exceeded 50 ms; the declared
p99 target passed. Separate timer-cancel p99 was 0.021 ms across 105 loaded
cancelled calls, with 23 calls completing before the timer. The maximum-gap
result remains the acceptance measure. No diagnostic GC settings were used
in this final probe. `make verify-quick` passed.

The held-out run additionally measured complete-call poll gaps with normal
GC and no added load across actual tournament roots. At three through seven
seats, p99 was 0.638 / 5.205 / 4.921 / 21.391 / 8.558 ms; maxima were
36.240 / 316.219 / 140.819 / 307.799 / 447.258 ms. These broader roots and
uncontrolled desktop conditions differ from the controlled loaded probe.
They do not change its declared p99 pass, but the observed long gaps mean
there is no 50 ms worst-case cancellation guarantee. Across all held-out
Scholar decisions there were zero Greedy sample fallbacks, zero budget-out
samples and zero no-world samples. Those simulator counters do not measure
room watchdogs; the original shadow worker probe below measured those.
The integration probe now measures actual Scholar play.

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
SHA-256, catalog hash, exact work rows, partition/epoch, block count, Go
CPU budget and load are part of the immutable experiment identity. The
shard schedule and the CPU model are not; keep them fixed by hand.
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

For a fresh frozen held-out epoch, use `-partition heldout -frozen`, one
nonzero profile, `-blocks 100`, counts 3 through 7 and `-suite all`. The full
manifest has 800/1600/2200/3700/3000 unique games, respectively, or 11,300 in
total. Ten shards assign `(profile, count, block)` ordinals modulo ten and
cover ten independent deals per count each. They can run sequentially here
or on isolated CI workers with the same binary, GOMAXPROCS and load (the
config does not record the CPU model, so timings from different machines
would pool silently; strength is unaffected, since play is deterministic). Share
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

## Original shadow worker watchdog probe

Before room integration, the opt-in `TestScholarWorkerWatchdogMeasurements`
shadowed the full `2x8`
row inside the existing admitted-worker hook, under a one-permit pool and
the unchanged 30-second watchdog. One seven-seat table runs at a time.
Released Greedy chose accepted actions after the shadow search, so
these trajectories measured worker completion and watchdog behavior, not
Scholar strength. Every shadow call must complete both attempts without
a Greedy fallback; worker reports and durable records must contain no
ordinary watchdog fallback. The room integration below replaces this test
with twenty actual Scholar trajectories under the same opt-in flag.

Across three seven-seat shadow games at `GOMAXPROCS=1`, all 349 calls
completed their full work with zero watchdog or Greedy sample fallbacks.
The maximum active worker duration was 116.0 ms. The same three trajectories
passed under the race detector with zero fallbacks; its slower clocks are
not used for latency estimates. No other tables or artificial CPU workers
competed for the one pool permit.

## Frozen preflight and release recommendation

The final implementation repeated one development block at each count
(113 games) and matched every accepted batch, final result and final hash
from the earlier implementation. Its throughput projects 5.79 hours for
the full gate, with observed-rate range 4.09 to 7.40 hours. One six-bot
game took 4.62 seconds; its decision p50/p95/p99 was 34.25/121.10/134.72 ms
(108 calls), and actual per-point service was 190.44/765.81/765.81 ms
(18 points). These are a preflight, not a reliable game-tail estimate.

The row above was fixed before any held-out play. The completed plan used
100 independent blocks per count at three through seven seats, epoch one, all treatments,
11,300 unique games, one preserved binary and ten resumable shards. Normal
GC, `GOMAXPROCS=1` and zero additional CPU workers applied throughout. The
strength look happened once, after all 500 count-blocks completed. Any
subsequent informed change needs fresh held-out seeds.

Retain the measured row as the Scholar strength candidate. Its six-bot turn
p99 of 1.61 seconds and game-compute p99 of 11.76 seconds go to the owner
before Scholar is offered. Keeping full work preserves the demonstrated
strength at the measured wait. Fewer samples or different budgets by Age
could reduce waits, but change the measured preset and require fresh
validation and held-out evidence. The calibration slice did not enable a room preset. The subsequent
integration below enables the same full row; public release still depends on
the remaining release gates.


## Room preset integration

@OpenAI, 2026-11-08. New table preset set `haftgane-bot-3` maps `easy` to
Apprentice and `normal` to Scholar; it does not offer Master. Numeric policy
and Scholar preset versions are both 1. Stored Scholar rows contain
`allowance: 2` (sample attempts), `candidates: 8`, `samplerChecks: 1048576`,
`maxChoices: 217`, `maxProbes: 12` and `maxQuotes: 2616`. The named
`bot.ScholarWork()` and recipe test pin this row and the unchanged reward,
preferences and noise. Set-2 tables keep their complete frozen baseline rows,
including Greedy for existing normal and hard seats, through restore and
future switches. [Room bots](room-bots.md) defines that compatibility contract.

The expanded cancellation probe adds turn-one roots from every Age at three
and seven seats to the original fifteen turn-three fixtures and discard root.
Each has eight tapes: 176 complete calls per load case, normal GC,
`GOMAXPROCS=1`, Go 1.26.3 on Apple M2 (Darwin arm64), same spinning-goroutine
load model as calibration:

- Idle: whole-call maximum poll-gap p50/p95/p99/max
  0.019/0.069/0.255/1.428 ms. Timer-cancel-to-return p99/max 0.024/0.024 ms
  across 43 cancelled calls; 133 calls completed before the timer.
- One competing CPU worker: whole-call maximum poll-gap p50/p95/p99/max
  21.510/29.616/46.356/55.492 ms. Timer-cancel-to-return p99/max
  0.024/20.583 ms across 157 cancelled calls; 19 completed before the timer.

The loaded p99 passes the unchanged target below 50 ms. The maximum exceeds
50 ms, so this is no worst-case guarantee. The complete-call poll gap remains
the acceptance measure; timer cancellation alone can miss completed calls.

The room probe plays twenty games with one human and six actual Scholar bots,
not shadow decisions, under one pool permit and no additional rooms or CPU
workers. It uses normal GC and the same machine and Go settings as above.
Seeds are fixed development fixtures; sides cycle through day/random/choose.
Human picks are submitted immediately when offered, and their wait ends at
the resolving batch's human outbox frame. This includes queueing, six serial
policy calls, persistence and views; bot-only abilities have no human-pick wait.

- Human-pick-to-resolution, 366 points: p50 315.565 ms, p95 996.731 ms,
  p99 1,323.614 ms, maximum 1,610.321 ms.
- Whole-game wall time, twenty games: p50 6.697 s, p95 8.991 s,
  p99 and maximum 9.062 s.
- 2,229 bot decisions completed the full two attempts, with zero watchdog
  fallbacks and zero no-world/sample fallbacks.

These room measurements include real actor and storage overhead. They differ
from the held-out simulator's CPU service sums and use fewer development
games; they do not supersede its strength result or establish mixed-room
capacity. A lower-work row could shorten waits but would require a new preset
and fresh strength evidence.

Both probes remain opt-in, outside `verify-quick`. Run through the guard:

```sh
GOTOOLCHAIN=go1.26.3 GOMAXPROCS=1 HAFTGANE_SCHOLAR_MEASURE=1 \
HAFTGANE_SCHOLAR_WORKER=1 HEAVY_NOTIFY=auto python3 tools/heavy.py run \
  --kind test --wait 3600 -- go -C server test -count=1 -p 1 -v \
  -timeout 20m \
  -run '^(TestScholarCancellationMeasurements|TestScholarWorkerWatchdogMeasurements)$' \
  ./internal/haftgane/bot ./internal/tableroom
```
