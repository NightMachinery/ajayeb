# Haftgane engine contract in rooms

The table actor uses real `rules.Game` values and whitelist DTOs from
`haftgane/view`. No engine value, setup root or observation fingerprint is
serialized into a live frame. Legal keys are the engine's canonical offers;
the actor never reconstructs an action from client fields.

## Setup and replay

A schema-2 table stores seed, seat count, seating, sides and end policy.
`options.sides` in the lobby accepts `choose`, `day` and `random`; absence
means `day`, including older lobby records. Start carries that effective
choice into the table, and rematch preserves it. Chosen sides open one
simultaneous side point before dealing; day/random sides open Age I directly.
The seeded engine owns board assignment, side draws, guild membership and
all three deals. Seat shuffling remains an independent stream.

The real rules version is 1. Development fake records use 1000 and archive
as `rules-version`; no fake batch is interpreted as real play. Compatible
real records replay setup and canonical batches, then restore the unresolved
private pending block without applying it. The finished-game log download
uses the same side-aware setup decoder as creation, restore, validation and
undo. New records store the effective side rule explicitly, including `day`
when the caller omits it. Hash-only refresh and unknown-field
preservation retain the existing store policy.

History entry i is always batch i. Entry zero is `game-start`, empty for
chosen sides and carrying Age I's start for day/random sides. Every resolved
side or ability point is a batch. Discard construction precedes a kept-card
decision at the sixth turn; the next Age waits for both. An actor test proves
this with a natural, replayable seven-seat prefix rather than a checkpoint.

## Projection and limits

The platform carries the real engine's public cities, complete choice cost
facts and payment witnesses, own hand/kept/tucked cards, disabled explanations,
notebook seen hands/pile looks/inference, neighbour production and full scores.
Filtered events retain their complete public fields and live entitlements;
the platform derives `pass` from the Age only for `hands-passed` events.
End reveal is a separate field and does not retroactively reveal history.
With final reveal disabled, results remain public but hidden holders and the
log download remain unavailable.

Card lists use catalog design ids in catalog order, never copy ids or deal
positions. Maximum resource cost is seven units: fifteen undominated payment
outcomes per construction, and at most 217 ordinary-turn keys. Key length is
bounded by 64 bytes. The real rules gate proves these constants; actor tests
also check offered groups, effective payment totals and keys while playing.

One immutable projection/history per seat plus spectator is cached for the
current engine fingerprint, at most eight entries. Envelope metadata and
pending picks remain outside that cache. A resolution or undo invalidates it.
The engine value itself is 2,208 bytes on the measured arm64 build.
A sample on 2026-10-24 retained 75,656 heap bytes per new seven-seat table,
averaged over 32 tables with eight attached clients and drained outboxes.
A separate eight-table sample played seven-seat tables to completion, then
resynced every role and drained the outboxes. It retained 656,173 heap bytes
per table, including eight cached projections and histories. Those cached
values serialized to an average 468,707 JSON bytes per table; the largest
single role's view plus history was 61,513 bytes. These are distinct measures:
heap includes the actor and connections, while JSON counts just the cached
view and history values, without the full frame envelope. Finished caches stay
resident while the table has a connection, because no later batch invalidates
them. The last connection to leave drops them (M16.haftgane-security-review), so
a table that stays in memory with no one watching, for up to the idle timeout,
holds no cached view, and the cache is bounded by the sockets attached: each
holds one role's view at most, about 61 KB of JSON at the largest. The capacity
slice will decide whether histories should instead live only for a broadcast.
One seven-seat game averaged 5.79 ms per resolved last-confirmation transaction
across 19 batches, including storage, projections and driver overhead. Its
largest observed offer had 34 keys; 217 is the proven ceiling, not that sample's
maximum. Retained heap grows with notebook/history and queued frames; these
samples are not peak-capacity claims.

## Bots and verification

Room bots use the released Random, Apprentice and Greedy policies under
frozen versioned work rows, through the existing pool and cancellation seam.
[Room bots](room-bots.md) describes the internal level mapping, compatibility,
privacy boundary, watchdog and acceptance checks.

The fake engine stays as isolated test infrastructure for its own rules,
small reference model and fault fixtures. The actor's differential model,
sweeps, privacy frames and fuzzer use real rules. The independent oracle reads
state/deals/batches and its own field whitelist, with permitted observations
for memory/inference; it never uses a production view as its expected answer.
Real socket tests include every seat and a spectator, drafts, resends,
controller changes, end policies and restarts. Contaminated-frame canaries
remain mandatory. [Testing](../testing.md#the-real-haftgane-engine-contract)
names the checks and commands.
