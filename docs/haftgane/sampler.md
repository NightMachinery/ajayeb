# Haftgane observation-only sampler

Implemented by @OpenAI for M21.haftgane-sampler. This engine facility does not
enable a bot level or change the observation-only policies.

`rules.Determinize(o, rng)` accepts a seated engine observation and a caller-owned
random tape. It returns an opaque `World` and a success flag. `World.State()`
returns an independent value with seed zero. The complete `World` occupies 1,200 bytes on the measured arm64 build.
The private witness retains each
Age's packets indexed by removal turn, with the seventh slot holding the
leftover. The bot has no accessor for that history and receives no fabricated
`Game`. Observations from partial-history development scenarios are outside
this contract: a complete legal past is required.

## Constraints and bounded work

Packet sightings partition the places into stretches with exact multiset
supplies. Public constructions and the observer's own picks fix individual
places. Remaining supply comes from the player-count deck, including each
undealt Age and the excluded guilds. The box accepts guilds only. Both pass
directions, returning packets, repeated designs and Babylon's kept cards use
the same removal slots.

Pile entries keep their creation times. A rebuilt design consumes an entry
present at that moment. At a skipped opportunity, every surviving entry must
have a name already in the owner's public city after simultaneous placement.
A still-open discard-build point requires at least one entry with a legal name,
even for an observer who cannot see the pile. An owner-only pile look fixes the
complete multiset before the rebuild, including a skipped opportunity's look.

The source search backtracks over exchangeable entry classes. After choosing
consumed entries, remove them from each private look. The remaining look sets
are nested; differences between successive sets give disjoint exact supplies.
The filler checks unit-capacity max-flow matchings for packet groups and look
buckets, plus a per-design flow coupling group supplies to bucket demands.
It fills exchangeable places in sorted order during search, then shuffles
within each class after success. Unrestricted groups are shuffled directly.
This removes equivalent failed permutations while retaining witness support.

`MaxSamplerChecks` is 1,048,576. Every source branch and every feasibility flow
counts, including failed branches. All loops, arrays and recursion depth are
also bounded by game dimensions. The bound limits deterministic work, not
elapsed time. A false return is exactly `(World{}, false)`, with no partial
world. A sampled bot must fall back to the existing legal Greedy policy;
it must never relax a constraint or read authoritative hidden cards. The
worker integration below provides that fallback without enabling a level.

`DeterminizeControlled(o, rng, limit, stop)` is the worker-facing variant.
The caller's budget is clamped to 0 through `MaxSamplerChecks`. Its
`SamplerCost` reports checks, a stop, or budget exhaustion separately. A pure
stop callback is polled before setup, every 16 source/flow checks, and around
final materialization. A stopped call returns a zero world. An uninterrupted
false callback preserves the ordinary sampler's exact world and RNG tape, with
zero allocations. Rules code reads no context or clock. The caller supplies
cancellation as input.

A callback was chosen instead of resumable search state: the worker needs only
a prompt abort, and preserving recursive branch stacks across calls would
complicate ownership, RNG continuation and scratch storage. An explicit check
budget still supports deterministic fallback experiments. The original
`Determinize` API remains the non-cancellable convenience wrapper.

`MatchesObservation(o, w)` is the separate debug/test oracle. It reconstructs a
complete setup from the private witness, replays checked `Game.Resolve` batches,
checks invariants, compares the final state and observes back to exactly `o`.
It calls no sampler constraint code. Its replay allocations are intentionally
outside the search hot path.

## Support and distribution

Before the first discard opportunity, forced picks and packet stretches leave
unrestricted shuffles and an unrestricted guild-box choice. These are uniform
over labeled copy arrangements, assuming all consistent initial deals equally
likely. Repeated designs and packet sorting quotient those arrangements; no
uniformity over distinct normalized states is implied.

After an opportunity, the randomized first-feasible search is **not uniform**.
It is not a posterior conditioned on opponents' strategic preferences. A small
constraint example has supply A, B, C and three domains AB, AC, ABC. The sampler
assigns ACB probability 1/2 and BAC and BCA probability 1/4 each. Its exact total
variation distance from uniform is 1/6. `TestSamplerConstrainedDistribution`
measures this behavior with 12,000 tapes: counts 6,002, 3,071 and 2,927,
with empirical distance 0.16683 from uniform.

`TestSamplerEnumerableReplay` conditions a real, complete seven-seat history on
the complement of six variable places, then enumerates every distinct
permutation. Its independent side rebuilds full deals and replays legal keys;
it shares no packet, skip or pile-look constraint code. Both sides accept the
same assignments at ten owner/nonowner prefixes covering two nonempty
all-duplicate skips, later additions, a pending rebuild and its completion.
Support sizes are 12, 2, 6, 1, 2, 1, 2, 1, 2 and 4. The recorded measurement
draws 2,048 tapes per case (the default suite now uses 256); every accepted
assignment is reached and no outside assignment is drawn.
The empirical distance from uniform ranges from 0 to 0.04297 in those cases.
Sampling error contributes to those distances; these measurements neither
establish uniformity nor bound bias for other observations. The conditional
six-place comparisons do not exhaust the full game's world space.

Every accepted enumerable world checks the notebook's certain deductions and
circulating/returning bounds. The same checks run on the sweep samples, using
sampled future removal slots for the cards that return.

## Measurements and reproduction

A sweep of 2,000 games per player count (10,000 games) checks every seat at every
prefix: 204,093 positions and 1,024,245 worlds. Setup cycles through all side
options; policy strata cover random, simple production/stage play and a policy
biased toward stages and sales. Phase counts are 3,335 side points, 180,000
ordinary turns, 5,710 discard points, 5,048 last-card points and 10,000 ended
positions. Every sample replayed consistently, every notebook check passed and
none exhausted the bound. The worst used 156,244 checks, at seven players,
sweep seed index 1218, point 21, seat 1. That tail position took about 0.27 s in
an isolated test; later bot calibration must account for it.

On an Apple M2 arm64 build, the sweep took 800.52 s. Seven-seat benchmarks
measured 191,768 ns/op initially and 596,148 ns/op at a later natural prefix,
with 0 B/op and 0 allocs/op in both. These are observations, not latency targets.
The skipped-look fixtures also assert zero allocations. A 60-second run of
`FuzzMatchesObservation` passed 19,663 cases without a crasher. Leak tests exchange
future packets, a box guild and the real seed while preserving observations;
the complete samples and consumed RNG tapes match across 314 state pairs.

Submit heavy commands one at a time through the asynchronous guard. The normal
Go suite includes small sweeps, the exhaustive conditional comparisons,
distribution checks, failure and allocation tests, and saved regression cases.
The long sweep and benchmark command is:

```sh
HEAVY_NOTIFY=auto HAFTGANE_SAMPLER_GAMES=2000 GOTOOLCHAIN=go1.26.3 \
  python3 tools/heavy.py run --kind test --wait 3600 -- \
  go -C server test ./internal/haftgane/rules -run '^TestSamplerSweep$' \
    -bench '^BenchmarkDeterminize' -benchmem -count=1 -v -timeout=45m
```

The fuzz target generates complete legal prefixes, samples from their
observations, checks replay consistency and validates notebook statistics:

```sh
HEAVY_NOTIFY=auto GOTOOLCHAIN=go1.26.3 \
  python3 tools/heavy.py run --kind fuzz --wait 3600 -- \
  go -C server test ./internal/haftgane/rules -run '^$' \
    -fuzz '^FuzzMatchesObservation$' -fuzztime=60s -parallel=1
```

## Worker integration and cancellation evidence

`bot.SampleRoot` accepts only the seated observation, exact offer, independent
policy entropy and a deterministic check budget. It consumes the controlled
sampler and `World.State()` by value. Its ready state belongs to the root
worker; a future simulated-seat policy must receive `Project` only. On a
no-world or budget-out result it invokes the existing exact Greedy once, with
separate attempts, worlds, checks, no-world fallback and budget fallback
counters. Cancellation returns no root state or replacement key. No released
policy, frozen preset, stored game or frame changes in this integration slice.
The dormant `ChooseAge` runner now supplies paired Age comparisons;
[the rollout guide](rollouts.md) explains multi-sample failure aggregation and
its remaining calibration gates. No level is enabled.

The room worker hook test pauses inside sampling while the actor serves a
request, then changes the bot's controller and verifies physical cancellation,
no accepted replacement and no watchdog diagnostic. The baseline policies keep
their observation-only boundary; only the root preparation file may name a
sampled State. Forbidden-information pairs now compare successful samples and
budget fallback diagnostics as well as baseline choices. The tabletest leak
oracle also checks sampling counters with an injected-seed negative control.

The review identified a real conflict: the fixed check bound did not make a
single call promptly cancellable. A measured natural call took 256 ms, and
running to the bound could take seconds. Polling resolves cancellation without
reducing an uninterrupted decision's prescribed work. It does not make full
samples cheap.

A guarded arm64 measurement used `GOMAXPROCS=4`, one test job, and 64 tapes at
each of nine saved seat/position fixtures. External desktop load was not
controlled. The two earlier five/six-seat regressions averaged 2.580 and
2.464 ms per completed call. At the seven-seat tail position, the seven seats'
means were 10.490, 170.731, 5.873, 15.824, 13.193, 14.310 and 12.409 ms. Seat
1's median was 203.640 ms, p95 221.674 ms, maximum 228.181 ms, and its largest
check count was 157,271. Other seats' maxima ranged from 20.157 to 30.946 ms.
These costs are at discard-constrained positions, unlike the initial/later
unconstrained benchmarks above.

The injected-stop measurement cancels after 32 checks inside search, then
deliberately runs the next 16-check interval before the callback reports it.
It measures that early interval plus stop unwinding, not later windows,
preparation or the materialization tail. Across 576 calls per load condition,
this early-window cancellation-to-return latency was:

- No test-local CPU workers: p50 0.055 ms, p95 0.252 ms, p99 0.415 ms,
  maximum 1.309 ms.
- Two additional continuously computing goroutines: p50 0.044 ms,
  p95 0.177 ms, p99 0.302 ms, maximum 0.514 ms.

Both early-window p99 values meet the 50 ms target under those stated
conditions. Every cancelled call returned no world, and the CPU workers are
joined before the test exits. These numbers alone do not describe the
slowest polling interval of a call.

The opt-in `TestSamplerPollGapMeasurements` timestamps every stop poll during
complete calls and records the maximum gap per call, including entry to the
first poll and the final poll to return. It covers all seats of the three
saved fixtures with 64 tapes each, then every seat/prefix of fresh games.
A cancellation just after a poll must wait for the next poll, so these gaps
cover later search windows, preparation and materialization that the
early-window injection omits. They measure the wait until the next poll;
stop unwinding is exercised separately by the injected-stop test. Neither
measurement is a universal host-latency guarantee.

The guarded arm64 poll-gap run used `GOMAXPROCS=4`, no test-local CPU load
workers and uncontrolled external desktop load. Across 21,629 complete calls
(1,152 saved-fixture calls plus every seat/prefix of 40 fresh games per player
count), the per-call maximum gaps had p50 0.136 ms, p99 0.264 ms, p99.9
0.371 ms and maximum 1.270 ms. The maximum occurred at the saved seven-seat
position, seat 1, tape 11; the slowest full call took 231.358 ms. These
percentiles describe the distribution of each call's largest interval, rather
than all individual intervals pooled together. The observed p99 is below the
50 ms target under this stated load.

At the costly seat's observed mean, 16 full samples cost about 2.73 seconds
per decision before rollouts. For this measured table, with the human at seat
0 and bots at seats 1 through 6, the corresponding serialized sum is
`16 × (170.731 + 5.873 + 15.824 + 13.193 + 14.310 + 12.409)` ms, about
3.72 seconds for all six bots. A 16.4-second total would be a hypothetical
table where all six bots cost as much as the costliest seat; no such table
was measured. These are arithmetic estimates from per-seat means, not an
approved preset. Scholar cannot assume tens of samples are cheap at every
position. Calibration must measure tails and full-table waits at one fixed
preset, or improve the sampler, before enabling a level. It must not silently
reduce work for six bots or use elapsed time to select completed comparisons.

The strengthened support probe fixes the entire real world at every seat and
prefix, then separately varies random pile entries across all Ages, including
potential rebuild sources. Its default ten games per player count checked
5,088 true-world cases and 337,140 assignments in 27.32 s. The larger
80-game-per-count run accepted all 40,879 true worlds and found exact replay
agreement on 2,264,520 assignments (12,445 accepted) across 3,500 conditional
universes, with 1,430 varied potential rebuild sources. That probe took
206.55 s; measurements plus probe took 222.990 s. The older fixed fixture's
exhaustive assignments remain; only its default empirical tape count was
reduced to keep ordinary validation short.

Reproduce both the larger probe and measurements with:

```sh
HEAVY_NOTIFY=auto HAFTGANE_SAMPLER_GAMES=80 HAFTGANE_SAMPLER_MEASURE=1 \
  GOMAXPROCS=4 GOTOOLCHAIN=go1.26.3 \
  python3 tools/heavy.py run --kind test --wait 3600 -- \
  go -C server test ./internal/haftgane/rules \
    -run '^TestSampler(SupportSweep|CancellationMeasurements)$' \
    -count=1 -v -timeout=20m
```

To reproduce the complete-call poll-gap run and rerun the early-window stop
measurement without the support sweep:

```sh
HEAVY_NOTIFY=auto HAFTGANE_SAMPLER_GAMES=40 HAFTGANE_SAMPLER_MEASURE=1 \
  GOMAXPROCS=4 GOTOOLCHAIN=go1.26.3 \
  python3 tools/heavy.py run --kind test --wait 3600 -- \
  go -C server test ./internal/haftgane/rules \
    -run '^TestSampler(Cancellation|PollGap)Measurements$' \
    -count=1 -v -timeout=10m
```
