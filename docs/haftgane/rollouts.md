# Paired simultaneous Age rollouts

@OpenAI. Implemented in M21.simultaneous-age-rollouts. `bot.ChooseAge` is an
experimental worker entry point for later Scholar calibration. No released
level calls it, and no policy/preset version, stored game, frame, Duel module
or web asset changes. [The bot plan](bot-plan.md) defines the release gates.

## Work and failure

The input is one seated `Observation`, its exact engine offers, independent
policy entropy and explicit `AgeWork`: sampling attempts, candidate cap and
sampler checks per attempt. There is no default or calibrated work row.
The allowance is per bot; there is no bot-count input or division by table
size. Zero attempts or zero sampler checks can exercise deterministic fallback.

One initial Greedy ranking supplies both the shortlist and the legal fallback.
The shortlist retains Greedy's best, the best build/stage/sale when space
permits, then other ranked keys up to the caller's cap. Payment-recipient
variants remain separate offered strategies. A cap of `MaxTurnChoices` keeps
all offered keys. Ties use canonical key order.

An attempt samples one world through the engine. Every shortlisted root key
is then simulated on a copy of that same state, with the same policy tapes.
Only a complete sweep adds rewards to the means. Failed samples contribute
no rewards; other successful sweeps still count. Attempts use independent,
indexed tapes, without retrying a failed world. If no sweep completes, reuse
the initial exact Greedy ranking once. This avoids calling `SampleRoot` and
recomputing a whole-decision fallback for each failed sample.

`AgeCounters` aggregates sample attempts, worlds, sampler checks, no-world
samples, budget-out samples, completed sweeps, comparisons, resolved batches,
policy choices, shortlist exclusions and Greedy fallback decisions. Budget-out
samples are also no-world samples. They are separate from exceptional room
watchdog fallbacks. `AgeCandidate` holds reward sums/squares and paired
sum/squares of the difference from `ReferenceKey` (the Greedy candidate).
All ranked candidates have exactly `CompletedSweeps` observations. The highest
sum wins, with canonical ties. These are local search results; calibration
logs export safe counters, without raw keys or sampled cards.

Cancellation anywhere returns the zero decision and its context error, even
after earlier complete sweeps. An incomplete incumbent is never delivered.
Invalid work, an unoffered simulated action or engine failure returns an
error, without accepting a partial sweep or weakening legality.

## Simultaneous play and information

The runner first projects observations for every seat from one frozen state.
Only then does it gather every required action. Actions come from
`LegalState` on that same state and are checked for membership before one
`ResolveState` call. No opponent reacts to the forced root pick in the same
round. Side choices and discard/last-card abilities follow the engine's
required seats and order.

`rollout_policy.go` receives only that seat's `Observation`, own offered
choices, preference and policy tape. It cannot name the state-only engine
APIs or acquire state through runner factories; AST guards enforce this
separation. The orchestration files alone own sampled State values. No bot
constructs a Game or reads the world's private witness.

The root retains its supplied notebook. Other simulated seats start with a
new notebook; their pre-root private memories are omitted, an explicit
approximation. Each seat subsequently remembers its own sightings and picks,
public plays/rebuilds and its own private pile looks. Another seat's sold or
tucked identity never enters its notebook. Checked full-game history tests
exercise both passing directions, private looks, skipped/pending discard
points and Babylon's kept cards. Side policies receive no future hand.

Every seat draws one persistent preference per sweep: balanced, economic,
science or military. Each ranks its own estimated gain using the existing
evaluator, with extra weight on the preference's features. Uniform additive
noise uses the existing Apprentice milli-VP scale. This exploratory mixture
is neither the released Apprentice policy nor a calibrated Scholar preset.
It is independently self-interested, without a coalition or human-targeting
term.

Policy randomness is indexed by sweep, Age, turn, phase, ability decider,
seat and canonical action key. It is not a mutable stream shared between
candidates. A changed offer/ranking cannot shift draws for common actions,
and an extra ability cannot consume another logical point's tape. A preference
persists across the whole rollout. Root sampling mixes only permitted input
and independent entropy. Production code reads no clock or runtime state.

## Horizon and utility

A rollout resolves the current Age through its conflicts and dependent
abilities. A side-selection root includes Age I. It stops when the Age advances
or the game ends; no policy chooses from the next Age. At cutoff each seat's
own simulated observation supplies exact earned score plus the existing
future-value estimates. Terminal observations use exact final scores. Coins
break score ties.

Reward follows the design's exploratory blend: three quarters pairwise rank
reward plus one quarter tied-win credit. Integer scale 3,360 represents every
N=3..7 and possible tie exactly. Mean reward, not a two-player minimax backup,
selects the root. No strength or release-wait claim follows from this choice.

## Bounds and evidence

At most 65,535 attempts fit the explicit uint16 input, each with at most
`MaxSamplerChecks`. At most `MaxTurnChoices` (217) candidates are retained.
Each candidate resolves at most `MaxPoints` (25) batches, with at most seven
seat policies per batch and one final evaluation per seat. Legal/ranking
buffers are bounded by the engine's offered-choice maximum. Reward sum and
square accumulators fit int64 at these bounds.

Storage retains root statistics and one candidate's state/notebooks, without
a search tree, sampled-history reconstruction or retained world cache. Root
arrays use at most 217 choices, 217 rankings, 217 reward values and 217
candidate summaries. A rollout uses seven notebooks and observations, one
217-choice buffer and a 32-event engine buffer. On the measured arm64 build,
State is 1,048 bytes, Knowledge 1,278, Current 456, Choice 42, Ranked 104,
AgeCandidate 48 and the complete AgeDecision 10,552. String storage and Go
stack/garbage collection add overhead; fixed value sizes are not an RSS cap.

The 50-game sweep (ten games per player count) made 4,669 decisions on 4,669
worlds. Every sampled witness passed `MatchesObservation` outside the hot
path; no sample or Greedy fallback occurred. It covered 100 side decisions,
4,500 ordinary turns, 37 discard-build and 32 last-card decisions, resolving
70,603 simulated batches and charging 441,472 sampler checks in 38.24 s.
The ordinary suite uses one game per player count. Separate memory checks
matched checked history in 40 seven-seat games, including 39 pile looks and
57 last-card points. Forbidden-information pairs compare entire rankings and
counters across future deals, opponent hands/seeds, hidden sales/tucks and
excluded guilds. Room worker tests pause inside the actual rollout, serve an
actor request, then cancel without accepting a replacement or watchdog
fallback.

A predeclared paired-noise experiment used 64 samples, eight candidates and
one initial position per N=3..7. It compared variance of paired reward
differences against `var(A) + var(B)`, the independent-draw variance with the
same empirical marginals. Ratios were 0.3298, 0.5015, 0.3947, 0.2370 and
0.2119, with aggregate 0.3446. Common randomness reduced comparison noise in
these fixtures. It does not guarantee lower variance for every pair or
establish playing strength.

A guarded arm64 run used `GOMAXPROCS=4`, one test job and uncontrolled external
desktop load. Whole-call context polls cover ranking, sampling, every seat,
batches, abilities and cutoff. For each complete decision, the test records
its largest gap, including entry/first poll and final poll/return. With two
attempts per decision, 1,058 calls per load condition covered fresh prefixes
and the saved costly seven-seat ability root. Per-call maximum-gap results:

- No test-local CPU workers: p50 0.157 ms, p99 2.618 ms, maximum 27.054 ms.
- Two continuously computing goroutines: p50 0.231 ms, p99 4.099 ms,
  maximum 16.472 ms.

These p99 values meet 50 ms under the stated load. Gaps measure waiting for
the next cancellation poll over whole calls; actual stop unwinding is tested
separately. This is not a universal host-latency guarantee. CPU workers are
joined before the test exits.

The saved root is an Age III discard-build decision at logical point 21,
seat 1; inactive seats have no decision to measure. Two-attempt whole-call
means with no test-local CPU workers were 411.508 ms at eight candidates,
430.946 ms at sixteen and 463.985 ms at all 65 offered candidates. Maxima
were 545.108, 591.100 and 632.738 ms. The loaded means were 411.672, 440.758
and 430.864 ms. Sampling dominates this late root, which has no later seat
policy choices. These are exploratory worker service times, excluding pool
wait and room delivery, not a calibrated six-bot wait.

A short initial seven-seat benchmark with two attempts measured:

- Eight candidates: 78.58 ms/op, 1,219,500 B/op, 15,622 allocations/op.
- Sixteen cap: twelve actual candidates, 111.80 ms/op, 1,830,053 B/op,
  23,535 allocations/op.
- All keys: the same twelve candidates, 110.91 ms/op and the same allocation
  counts. This fixture cannot distinguish sixteen from an all-key shortlist.

A separate, GC-started storage probe sampled `HeapAlloc` every 128 context
polls and at return. At eight candidates its baseline was 221,640 bytes and
sampled high-water mark 1,426,848 bytes (1,205,208 bytes above baseline).
At twelve candidates, both the sixteen cap and all-key cases rose from
221,488 to 2,028,688 bytes (1,807,200 above baseline). This is a sampled Go-heap
high-water mark for one two-attempt call, including temporary garbage, not an
exact process peak or RSS bound. The probe adds sampling overhead and does
not supply the latency figures above.

The few benchmark repetitions and host scheduling limit precision. Rebuilding
keys once per policy/shortlist reduced temporary allocation without changing
tapes or measured paired rewards. Calibration later cut the eight-candidate
row to 13,992 B/op and 50 allocations/op without changing any choice
([scholar-calibration.md](scholar-calibration.md)); calibration must assess full-table waits, quality and memory before freezing
any preset. Work is never reduced because more bots sit at the table.

## Reproduce

The larger legal sweep, timing/noise probes and allocation benchmarks:

```sh
HEAVY_NOTIFY=auto HAFTGANE_ROLLOUT_GAMES=10 HAFTGANE_ROLLOUT_MEASURE=1 \
  GOMAXPROCS=4 GOTOOLCHAIN=go1.26.3 \
  python3 tools/heavy.py run --kind test --wait 3600 -- \
  go -C server test ./internal/haftgane/bot \
    -run '^TestAge' -bench '^BenchmarkAgeDecision$' \
    -benchtime=300ms -benchmem -count=1 -v -timeout=15m
```

The simulated-seat mixture, evaluator, reward blend, shortlist and work bounds
must be calibrated and versioned together in M21.scholar-calibration. This
slice supplies the comparison engine and diagnostics, not those release rows.
