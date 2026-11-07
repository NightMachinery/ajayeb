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
world. A future sampled bot must fall back to the existing legal Greedy policy;
it must never relax a constraint or read authoritative hidden cards. No bot
integration is part of this slice.

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
Support sizes are 12, 2, 6, 1, 2, 1, 2, 1, 2 and 4. Each case also draws 2,048
tapes; every accepted assignment is reached and no outside assignment is drawn.
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
