# Fixed-preset Haftgane room bots

@OpenAI, 2026-11-08. The table actor runs observation-only policies inside
its existing pool worker. Duel's scheduler, bot goldens and review semaphore
are unchanged.

## Levels and frozen work

New tables offer two levels:

- `easy`, shown as Apprentice, uses Greedy with fixed-point softmax noise at
  1.5 VP (D46).
- `normal`, shown as Scholar, uses the calibrated paired Age rollout policy:
  two sample attempts, at most eight candidates and 1,048,576 sampler checks
  per attempt. Every bot receives the complete row, including six-bot tables.

Master (`hard`) is unavailable for new Haftgane tables and lobby additions.
The shared lobby still offers Duel's hard level at two seats. A Duel hard
bot carried into a larger lobby, or a legacy hard bot in a rematch, becomes
Scholar for the new table; its parent table remains unchanged. Random and
exact Greedy remain simulator baselines. [Scholar calibration](scholar-calibration.md)
records the held-out strength evidence and measured waits.

A new table freezes both levels under preset set `haftgane-bot-3`, including
levels not yet seated. Both use numeric policy version 1 and preset version 1.
Each `botPresets` row and open-point `work` row records the algorithm, level,
allowance, maximum 217 choices, 12 probes and 2,616 quote calls. Apprentice's
allowance 1 means one complete ranking pass; its `candidates` and
`samplerChecks` fields are zero. Scholar's allowance 2 means two complete
sample/comparison attempts, with `candidates: 8` and `samplerChecks: 1048576`.

`bot.ScholarWork()` supplies the named frozen work row. `ScholarPresetVersion`
freezes the entire recipe: Age horizon, shortlist and canonical ties, 3/4 rank
plus 1/4 win reward (scale 3,360), four rollout preferences and key noise
1,500. Changing the recipe needs a new version and new validation and held-out
evidence. Tests pin the version-one constants.

An admin selects a frozen level when switching a seat to a bot. A switch
retains and confirms an existing pick; otherwise its worker receives the full
row. Resignation and lobby fill use Apprentice. New points, switches, restore
and undo copy the table's frozen rows, never the latest deployed default.
There is no reduced six-bot allowance or donated work. Policy and preset
versions enter policy entropy.

## Stored compatibility

The record remains schema 2, format 6. A small supported-set registry accepts
sets 2 and 3 by complete equality, including bounds. Unsupported versions,
algorithms or edited rows fail closed. Set 2 remains exactly as written:
Apprentice at `easy`, Greedy at `normal` and `hard`, one pass per seat. Its
historical JSON lacks the two new Scholar fields; both decode as zero.
Existing normal and hard seats continue as Greedy, and old tables retain
all three frozen switch levels. Their entropy domain is unchanged.

Set 1 (Random/Apprentice/Greedy) and development records without supported
frozen presets still archive as `replay-failed`, as before. Format 2 records
are re-keyed to format 3 without replacing their bot configuration. Older
readers reject unsupported presets rather than reinterpreting Scholar as a
baseline. Replay uses accepted action keys and never reruns a bot.

## Admission, cancellation and privacy

At most one job is queued or executing for a room. It takes the existing
`ai.Pool` permit under the address of the player whose point it is: the lowest
required seat whose human player has a connection, else the table creator's,
else the smallest attached human participant's (the protocol's Limits). It
searches one seat, releases, then queues the next seat. Queue waiting consumes
no fixed policy allowance and no watchdog time. No new deployment-wide
scheduler or speculative capacity reservation is introduced. Host-wide
room-work admission is limited to one worker before shared-pool acquisition,
as measured in [release capacity](capacity.md).

Scholar calls `bot.ChooseAge` only after pool admission. The actor copies its
seat's observation and exact offered choices, and derives a per-point policy
tape. The worker alone owns sampled and simulated states. Simulated seats see
only their projected observation and own rollout notebook. The sampler and
rollout cancellation tests pause within that work and prove that the actor
still handles requests. [The sampler guide](sampler.md) and
[rollout guide](rollouts.md) describe the information boundary.

A job receives neither a Game nor a State, seed, other seat's pending pick,
wire revision or task generation. Entropy uses a domain-separated policy root,
logical point, seat, permitted-observation fingerprint and frozen versions.
Changing hidden future packets or another pending pick leaves the complete
choice and counters unchanged. The policy's input copy is immutable.

Structural cancellation invalidates the generation immediately, then waits
for physical worker completion before starting a replacement. Returns must
match the actor incarnation, worker generation, point and seat, and still
match the controller, pending vacancy and offered set. Unrelated presence
or role changes do not invalidate valid work. A new actor cannot accept an
answer from a previous incarnation, even if its local generation number is
the same. Accepted picks survive restore without another search. Undo retains
controllers and uses frozen rows to reproduce the logical decision inputs.

Losing the last attached human participant cancels work. An original human
watching after resignation may keep a seven-bot continuation running;
unrelated spectators cannot start it. Shutdown waits for cancelled workers,
and both queued and active cancellation release their pool resources.

An exceptional watchdog defaults to 30 seconds of admitted search time.
It is separate from deterministic work. Structural cancellation accepts
nothing. A watchdog on a still-valid point may use the released complete
Greedy fallback, through the same legality and generation checks. Its accepted
key stays in normal pending/batch storage; a bounded diagnostic stores only
batch index, logical point, seat and reason `watchdog`. A deterministic
no-world result uses Greedy with reason `no-world`; a failed or incomplete
Scholar decision uses Greedy with reason `scholar-error`. Undo removes diagnostics
for discarded points. None enters frames, ordinary logs or downloads.
Ordinary acceptance workloads require zero watchdog fallbacks.

`BotFinished` is an optional seedless diagnostic callback with completed work
counters, active time, pool wait and fallback/cancellation status. It runs
outside the actor and carries no key, observation or fingerprint. Its callback
must be thread-safe and return promptly. Tests use it for measurement; it
changes neither policy inputs nor work.

## Verification

The room tests cover complete seven-seat games with zero through six starting
bots and mixed humans/difficulties, seven-bot continuation, all 64 six-bot
pending subsets through cold loads, human none/draft/confirmed variants,
side/discard-build/last-card policy calls, kept-pick switches, deterministic
undo and late results. Forced queued/active cancellation and delayed cleanup
check permits, waiters and strict per-room serialization. A watchdog fixture
checks the exact Greedy result and the persisted safe reason.

The ordinary suite covers released policy calls and legacy set-2 continuation.
Broad controller/persistence sweeps use Apprentice under the race detector;
focused worker tests still execute actual Scholar decisions. The full normal
wait measurement is opt-in with `HAFTGANE_SCHOLAR_WORKER=1`.
`TestFrozenBotDecisionGolden` pins keys and counters at five small development
positions for Scholar, Greedy and Apprentice under both tape domains. It runs
without race instrumentation and updates only with
`UPDATE_HAFTGANE_BOT_GOLDEN=1` after versioning any changed policy and retaining
the implementation needed by stored rooms. Baseline work compatibility also
has a direct test against what `bot.Choose` accepts.
`TestForbiddenInformationPairs` compares the complete Scholar result and
counters at the released row across different hidden future packets, opponent
hands, setup roots, private sales/tucks and excluded guilds. Only permitted
observations cross the policy boundary.

`TestScholarWorkerWatchdogMeasurements` is opt-in
(`HAFTGANE_SCHOLAR_WORKER=1`). It plays twenty actual one-human, six-Scholar
games through the actor, store, frames and one shared pool permit. The human
submits the first legal choice as soon as each point is offered; the wait ends
when the resolving batch reaches that human's outbox. Bot-only ability points
advance naturally and have no human-pick wait. Sides cycle through day, random
and choose; the test emits per-game and aggregate quantiles.

On 2026-11-08, Apple M2, Darwin arm64, Go 1.26.3, normal GC and
`GOMAXPROCS=1`, with no other test rooms or synthetic CPU workers:

- 366 human-pick waits: p50 315.565 ms, p95 996.731 ms, p99 1,323.614 ms,
  maximum 1,610.321 ms.
- Twenty complete games: p50 6.697 s, p95 8.991 s, p99 and maximum 9.062 s.
- 2,229 complete Scholar decisions; zero watchdogs and zero sample fallbacks.

These are host measurements, including filesystem and scheduling noise, not a
capacity promise for concurrent rooms. All work remains fixed. Reducing it
would change the preset that passed the strength gate. The original baseline
and mixed-Duel measurements remain in [simple-bot evidence](simple-bots-evidence.md).

The cancellation probe includes turn one of every Age at three and seven
seats, turn three at all counts, and a discard-build root. With one competing
CPU worker, complete-call maximum poll-gap p99 was 46.356 ms and maximum
55.492 ms over 176 calls. The p99 pass is marginal: only 3.6 ms below
the 50 ms target. With 176 calls, nearest-rank p99 is the second-largest
value, so scheduler noise may move a repeat run across the target. The
maximum is not a hard bound. [Scholar calibration](scholar-calibration.md#room-preset-integration)
gives the idle measurements and guarded commands.
