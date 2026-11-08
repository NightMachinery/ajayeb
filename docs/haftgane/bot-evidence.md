# Release bot evidence

@OpenAI, 2026-11-08. New Haftgane tables offer Apprentice and Scholar under
frozen preset set `haftgane-bot-3`. Master is unavailable. The strength gates
pass; this page does not approve public deployment or concurrent-room capacity.
[Room bots](room-bots.md) defines persistence, admission and compatibility.
The [guarded rerun instructions](bot-evidence-reruns.md) use existing entry points;
this documentation audit ran no new tournament.

## Frozen levels and strength

- **Apprentice (`easy`):** policy version 1, preset version 1, algorithm
  `apprentice`, temperature 1,500 milli-VP. Its stored work row has
  `allowance: 1`, `maxChoices: 217`, `maxProbes: 12`, `maxQuotes: 2616`,
  `candidates: 0`, `samplerChecks: 0`. Every bot ranks the complete offer once.
  The held-out gate required rank reward at least 0.75 and win credit at least
  0.50 against Random, over 100 independent deal blocks at every N=3..7.
  All counts passed, including interval lower bounds. At seven seats, rank
  reward was 0.8729 (95% interval 0.8576 to 0.8876), and win credit 0.6257
  (0.5943 to 0.6571). Greedy also beat Apprentice in the paired mixed fields
  at every count. [All baseline intervals](ai.md#first-held-out-tournament)
  cover 30,600 unique games and all 25 declared checks.
- **Scholar (`normal`):** policy version 1, preset version 1, algorithm
  `scholar`. The stored row has `allowance: 2`, `candidates: 8`,
  `samplerChecks: 1048576`, and the same 217/12/2616 choice/probe/quote bounds.
  `bot.ScholarWork()` pins two sample attempts and at most eight candidates,
  the Age horizon, 3/4 rank plus 1/4 win reward, four rollout preferences and
  key-indexed noise 1,500. The epoch-one held-out gate covered 100 independent
  blocks at every N=3..7, 500 count-blocks and 11,300 unique games. Every
  main-field mean and its interval lower bound cleared rank 0.65 and win
  `1/N + 0.10`. Every main, inverse and mixed paired-improvement interval
  was above zero. Combined paired rank improvement was 0.161368 (95% interval
  0.152365 to 0.170322). [All Scholar intervals and frozen binary/catalog
  identities](scholar-calibration.md#held-out-strength-gate-passed) are retained
  with the development/validation history and spent-seed warning.

Both receive full per-bot work even with six bots. Set-2 stored tables retain
their complete Apprentice/Greedy rows and tape domain; they do not acquire
Scholar on restore. Rule and record versions are separate from policy/preset
versions. Any informed retuning needs new versions and a fresh held-out epoch.

## Waits, cancellation, fallback and storage

- **Apprentice room baseline:** a seven-seat, six-Apprentice game measured
  active ordinary-decision p50/p95/p99 0.243/0.611/0.683 ms and
  accepted-pick-to-spectator-receipt 24.472/52.566/67.147 ms (127 picks).
  The latter starts after durable save, includes remaining picks, and is
  neither human-submit latency nor browser presentation. This historical
  set-1 run used the same policy/work as today's Apprentice: Go 1.26.2,
  Darwin arm64, eight logical CPUs, `GOMAXPROCS=4`, uncontrolled desktop load.
  Whole-game process allocation was 37,466,160 bytes in 322,518 allocations;
  post-GC retained delta 140,168 bytes. These include the harness, store and
  extra frame marshals. Ordinary fallbacks and watchdogs were zero across
  the baseline room cases. [Baseline evidence](simple-bots-evidence.md)
  includes ability phases, pool wait/hold, serialization, restore and mixed
  Duel load. Apprentice cancellation has correctness tests, but no published
  Apprentice-specific cancellation-latency distribution.
- **Scholar actor measurement:** twenty one-human, six-Scholar games on
  Apple M2, Darwin arm64, Go 1.26.3, normal GC, `GOMAXPROCS=1`, one pool
  permit, no competing test rooms or synthetic CPU workers. External desktop
  load was uncontrolled. Human-submit-to-resolving-outbox p50/p95/p99/max was
  315.565/996.731/1,323.614/1,610.321 ms over 366 points. Whole-game wall
  time was 6.697/8.991/9.062/9.062 seconds. All 2,229 decisions used the full
  two attempts; watchdog and sample-fallback counts were zero.
- **Scholar cancellation:** on that machine, Go and GC configuration, with
  one competing computing goroutine on the same Go thread, the per-call
  maximum poll-gap p99 was 46.356 ms and maximum 55.492 ms over 176 complete
  calls. Fixtures include turn one at three and seven seats in every Age,
  turn three at every count, and the costly discard root. Idle p99 was
  0.255 ms. The loaded p99 passes 50 ms by only 3.6 ms and is the
  second-largest observation; repeat-run scheduler noise can change the
  result. This is no worst-case guarantee. Separate loaded timer-cancel
  p99 was 0.024 ms over 157 cancelled calls; it cannot replace the
  complete-call measure. [Room integration evidence](scholar-calibration.md#room-preset-integration)
  records both probes and their different clocks.
- **Scholar simulator resources:** the 100 held-out six-bot games used
  `GOMAXPROCS=1` on Apple M2, normal GC and no added CPU workers. Serialized
  six-bot service per point had p99 1,608.108 ms; whole-game compute p99 was
  11.765 seconds. These exclude actor admission, persistence and delivery.
  Whole-game allocated bytes p50/p99 were 2,116,496/2,216,384, allocation
  counts 8,180/9,451, and heap-after-game 3,690,176/5,501,536 bytes.
  These include engine/harness work and are not peak heap or RSS. The
  initial seven-seat `2x8` microbenchmark used 13,992 B/op and 50 allocations.
  Across the complete gate there were zero Greedy sample fallbacks, budget-out
  samples or no-world samples; simulator counters do not measure watchdogs.
  [Calibration clocks and memory](scholar-calibration.md#clocks-load-and-memory)
  gives current phase-level simulator timings, the wider-root long poll gaps
  and older allocation measurements.

There is no separate room-work queue in these runs, so its wait is not a
measured zero. Current Scholar phase-specific actor wait/hold/allocation
breakdowns, browser presentation, production-host peak RSS, concurrent Scholar
rooms and mixed Scholar/Duel capacity remain unmeasured here. Historical
Greedy mixed-load data cannot establish Scholar capacity. Keeping full work
preserves the demonstrated strength at the measured wait; reducing work
requires a new preset and strength evidence.

## Sampler distribution and M21 acceptance

The [sampler evidence](sampler.md#support-and-distribution) distinguishes
support (which worlds can be drawn) from distribution (their probabilities).
Before discard constraints it samples uniformly over labeled arrangements,
not normalized states. Later first-feasible search is nonuniform: the small
three-domain example has exact distance 1/6 from uniform, empirical 0.16683
over 12,000 tapes. Ten independently replayed conditional six-place universes
had identical support, with all supported worlds reached in 2,048 draws per
case; empirical distances from uniform were 0 to 0.04297. This is measured
bias, not a claim of a strategic posterior or full-game uniformity.

The three [M21 acceptance bullets](../dev/roadmap.md#m21-sampled-bots-up-to-scholar)
stand as follows:

- **Sampler: met for the declared sweeps.** All 1,024,245 worlds over 204,093
  positions in 10,000 games replayed with `MatchesObservation`; notebook
  checks passed, no position exhausted 1,048,576 checks, and the worst used
  156,244. Independent enumerable replay includes skipped opportunities.
  The expanded support sweep checked 40,879 true worlds and 2,264,520
  assignments across 3,500 conditional universes. Distribution is published
  above, including its bias and finite-sample limits.
- **Scholar strength: met.** At every N the 100-block main targets passed,
  paired-improvement intervals were positive, and inverse/mixed fields had
  no regression. The complete per-count results are linked above; no failing
  player count was averaged away.
- **Waits, cancellation and watchdogs: partly.** Six-bot waits are measured
  at the released work row, loaded p99 cancellation passes narrowly, and
  ordinary watchdog counts are zero in the twenty-game actor probe.
  Wait acceptability needs an owner disposition before public enabling is
  treated as approved; this evidence does not record that decision. Host
  capacity and public deployment remain M22 gates. The measurements do not
  provide a universal 50 ms cancellation bound or concurrent-room promise.

## Evidence privacy audit

This audit checked the baseline, sampler, rollout, calibration and room guides
and the committed decision golden. Published measurements contain aggregate
counts, quantiles and fixture metadata, with no recorded live hands,
opponent pending keys, credentials or per-seat secrets. Full tournament
ledgers contain accepted keys for replay and remain in private experiment
storage, outside the repository.

The [decision golden](../../server/internal/tableroom/testdata/frozen-bot-decisions.golden.json)
has 30 rows: five synthetic development positions, three policies and two
tape domains. Each row stores only the deciding seat's chosen action key and
work counters as decision output, plus fixture identity, preset and derived
policy entropy. It stores no hand, observation, notebook, opponent submission
or sampled world. Its public fixture seed reconstructs a synthetic test game;
it is not a live-game secret. Set-2 Scholar and set-3 Greedy rows deliberately
test tape domains, not supported room mappings. The normal golden test checks
equality without regenerating the file. Observation-equivalent hidden-state
pairs test complete decisions and counters, independently of this disclosure
audit; [room verification](room-bots.md#verification) lists those checks.

