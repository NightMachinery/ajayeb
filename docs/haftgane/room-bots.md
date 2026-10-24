# Fixed-preset Haftgane room bots

The Haftgane actor calls `haftgane/bot.Choose` directly with one permitted
observation, the exact engine offer, policy entropy and a persisted work row.
Duel's scheduler, bot goldens and review semaphore are unchanged. These are
internal baseline policies; public release still waits for Scholar.

## Levels and frozen work

Each room code plays the policy its persona names, where that policy exists:

- `easy`, shown as Apprentice, plays the Apprentice policy (Greedy with
  softmax noise, D46).
- `normal`, shown as Scholar, plays deterministic Greedy as a stand-in until
  the Scholar policy exists.
- `hard`, shown as Master, also plays Greedy for now.

So normal and hard are equally strong until Scholar lands, and there is no
very weak room opponent. Random stays a simulator baseline only (D46). The
order is still monotone: in the held-out tournament Greedy beats Apprentice
in mixed fields at every seat count ([AI](ai.md#first-held-out-tournament)).

An admin explicitly chooses the level when switching a seat to a bot, and
the switch is refused unless the table froze that level. Resignation, and a
lobby's empty slot, use the lowest level, `easy`. A switch retains and
confirms any existing pick. Without a pick, its bot gets the full preset.

Each table freezes all three presets at creation, including levels not yet
seated. Each row contains preset set `haftgane-bot-2` (the `policy` field),
numeric policy version 1, preset version 1, algorithm, room level, allowance 1, maximum 217
choices, 12 probes and 2,616 quote calls. Allowance 1 means one complete
bounded pass, not one candidate. Apprentice uses the released fixed-point
1.5 VP temperature. There is no reduced six-bot allowance or donated work.
New points and switches copy the game's frozen row, never a default selected
at restore time. Policy and preset versions also enter policy entropy.

Table record format 2 (schema 2) added frozen `botPresets` and bounded
`botFallbacks`; each open-point `work` row embeds its complete preset.
Unsupported versions, algorithms, allowances or bounds fail validation.
Older development records without supported frozen presets archive as
`replay-failed`; their old fallback rows are never interpreted as new bots.
A reader that predates format 2 refuses it rather than reverting an all-human
table's future bot switches to old defaults. Replay still uses accepted keys,
not bot reruns. Unsupported future formats remain untouched. Format 3
(M16.haftgane-security-review) keeps format 2's presets and remembers each
client by a bounded key; a format-2 record is re-keyed on load
([protocol](../protocol.md), "Room files").

Preset set `haftgane-bot-1`, the first one, mapped `easy` to Random, `normal`
to Apprentice and `hard` to Greedy. Set 2 changed only that mapping. The bot
package's numeric policy and preset versions stay 1, because each policy's
behaviour and the published tournament are unchanged. A table frozen under
set 1 fails validation when it loads, so it archives as `replay-failed`
and is never replayed under set 2. That includes all-human tables. Only
development tables existed, so this was accepted before release.

### The cost of the next preset change

The frozen presets are an equality check against the build, not a second
copy that old tables keep playing. On every restore and every commit,
`checkWork` requires the record's presets to equal this build's. A restored
table keeps the same bot behaviour only within one build. The next preset
change, such as adding Scholar or retuning Apprentice, therefore archives
every live Haftgane table, including all-human tables that never seated a
bot, unless the slice making it changes this. That slice must choose one of
two options:

- Add a small registry of supported preset sets. An old table keeps its
  frozen rows and plays on with its old policy under the new build, and the
  registry drops a set only when no live table needs it.
- Accept that every live table archives at that deploy, and say so in the
  release notes. A softer form skips the check for all-human tables until a
  bot seat is first needed, at the cost of the guarantee that every table's
  presets are fixed at its start.

Seat switches and seat records are already checked against the table's
frozen presets rather than the build's, so a registry would not let a switch
pick a level its table never froze.

## Admission, cancellation and privacy

At most one job is queued or executing for a room. It takes the existing
`ai.Pool` permit under the smallest attached human participant address,
searches one seat, releases, then queues the next seat. Queue waiting consumes
no fixed policy allowance and no watchdog time. No new deployment-wide
scheduler or speculative capacity reservation is introduced; host-wide
room-work admission belongs to the measured capacity-release slice.

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
batch index, logical point, seat and reason `watchdog`. Undo removes diagnostics
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

`TestSixBotPresetWaitMeasurements` reports a full real-engine game for each
preset, including active search and pool wait. Each human-point wait includes
serialized bots, storage and real views; ability-only work is included before
the next human point. The first includes creation and attachment, later ones
start before the preceding human's resolving confirmation. These are sample
observations under the test host's load, not a capacity or latency promise.
Run it with `make test-go GO_PKGS=./internal/tableroom
GO_RUN=TestSixBotPresetWaitMeasurements GOFLAGS=-count=1`.

A non-race sample on 2026-10-24, one permit and one seven-seat game per
preset (18 human-point waits and 109 completed decisions each), measured the
following. It ran under preset set 1, so each item names the policy that
played; under set 2 `easy` plays Apprentice and `normal` plays Greedy. It
was not rerun for the remapping, which changed no policy or allowance.

- Random (`easy` under set 1): 38.95 ms mean wait, 63.91 ms maximum; 1.66 ms
  total active search and 0.050 ms total pool wait.
- Apprentice (`normal` under set 1): 36.36 ms mean, 41.00 ms maximum;
  15.99 ms active and 0.024 ms pool wait.
- Greedy (`hard`): 39.60 ms mean, 68.37 ms maximum; 17.33 ms active
  and 0.032 ms pool wait.

Every ordinary decision completed one full pass, with zero watchdogs.
These short host samples include filesystem and scheduling noise; they do
not establish an ordering of preset latency or a mixed-load capacity limit.
A race-instrumented run measured means of 127 to 138 ms. No policy allowance
changed between the two runs.

For the complete version/mapping contract, retained-cache and allocation
figures, phase and mixed-Duel latency evidence, restore times and guarded
reruns, see [internal simple-bot evidence](simple-bots-evidence.md). The Scholar
and Master labels are not claims that those policies exist; both play Greedy.
