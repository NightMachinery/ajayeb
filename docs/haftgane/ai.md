# Haftgane baseline bots

Random, Greedy and Apprentice live in `internal/haftgane/bot`. They are
simulator policies for the internal milestone. Room integration and the
public release follow the sequence in [the bot plan](bot-plan.md).
Duel's bots and difficulty presets remain in `internal/ai`.

## Inputs and exact rules facts

A decision receives its own `rules.Observation`, the engine's exact offered
`Choice` values, independently derived policy entropy and a `Work` row. It
returns one offered key, a completion flag and deterministic work counters.
It has no game, state, setup seed, hidden hand, full-state hash, clock or
pending submission. The call tape mixes supplied policy entropy with the
engine's canonical `Observation.Fingerprint()`. Legitimate remembered
information is part of that fingerprint. `enginekit/rng.MixSeed` also backs
Duel's existing compatibility wrapper, with unchanged output.

`Current.Facts` gives exact public production, discounts, chains, science,
shields, abilities, score categories and current neighbour-conflict outcomes.
`Observation.Preview` applies one offered placement, its effective payment
and its own income through the engine's caches, effects and scoring. It
credits public trade payments to the two recipients. Other seats' unknown
simultaneous placements and payments remain absent. The returned public
current is a counterfactual for quotes, not a legal world or a point
transition. It never advances hands, knowledge, military or ability timing.
Only the engine's real offered set authorizes a move.

## Policy and preset version 1

Random chooses a hand design in proportion to its physical multiplicity,
then an action kind uniformly, then an offered payment uniformly. Side and
ability decisions choose uniformly among offered keys. Payment-rich designs
therefore get no extra weight. Input order does not affect the random tape.

Greedy ranks the complete offer once, in integer milli-VP (1,000 is one VP),
then breaks ties by canonical key. Its named feature breakdown contains:

- Exact own score change, including optimal wildcard science and guild scopes.
- Military outcomes against each actual neighbour, with diminishing confidence
  from shield margin and a small discounted carry into later Ages.
- Unearned science growth, diminishing production and flexible units, trade
  discounts, eligible chain targets and declining short-term coin liquidity.
- Discounted unbuilt stages and the remaining uses of Babylon and Olympia
  abilities. A Halik stage estimates the coming private pile opportunity;
  when the actual offer arrives, each offered rebuild uses its exact own score.
- Structural affordability from engine quotes for at most 12 public costs,
  including the own hand, next stage and four fixed catalog probes.
- Bounded denial in the real Age pass direction and recipient-specific
  payment externalities, using public portfolios only.

Same-round effects in an isolated preview hold other public cities fixed.
Future utility is a policy estimate, not engine truth. Constants are frozen
in `bot/eval.go` as preset 1; changing them requires a new preset version.

Apprentice samples the same Greedy ranking at temperature 1.5 VP. Integer
weights approximate `exp(-delta/temperature)` at 1/16 exponent steps, using
a fixed recurrence and a capped tail. This is deterministic fixed-precision
softmax, rather than a claim of exact real-valued exponential probabilities.
The initial temperature is frozen for these measurements without tuning.

`DefaultWork()` is `{PolicyVersion:1, PresetVersion:1, Iterations:1}` for
each individual bot. A ranking performs at most 217 previews and 2,616 quote
calls (12 initial plus 12 for each candidate). Random needs one draw at
side/ability points and three at an ordinary turn. No allowance is divided
by table size, missing submissions or other bots. Unsupported rows fail.
Cancellation returns no partial key or partial ranked incumbent.
`Fallback` calls this same deterministic Greedy over the supplied exact offer.

## Reproducibility and seed separation

These partitions were declared before strength measurement:

- Development deals: base `0x6d170001`; correctness sweeps and quick smoke only.
- Validation deals: base `0x76616c6964617465`, reserved for calibration tooling.
- Release deals: base `0x72656c6561736531`, reserved for held-out tournaments.
- Policy entropy: independent base `0x706f6c6963797631`.
- Interval resampling: independent base `0x626f6f7473747261`.

Tournaments must record their precise derivation, block ids, lineups,
rotations and versions. Release measurements never drive tuning on those
same seeds. The small development smoke measures implementation behaviour;
it cannot establish the release strength targets.

## Checks and measurements

The source guard rejects runtime I/O, clocks, maps, floats, unrelated
modules and engine-state selectors. Tests cover all counts and point kinds,
engine-backed tactics, exact fallback, order-independent and concurrent
repeatability, cancellation during ranking, and equal decisions, features,
rankings and work on paired forbidden-information changes. Pairs include
opponent hands/root seeds, future deals, excluded guilds, hidden sales and
legally reached hidden tucks.

Run the focused suite through the guard:

```sh
make test-go GO_PKGS='./internal/haftgane/bot ./internal/haftgane/rules'
```

The opt-in development smoke and allocation benchmark also go through the
guard (detach long runs with `HEAVY_NOTIFY=auto`):

```sh
M17_BOT_SMOKE=1 python3 tools/heavy.py run --kind test -- \
  go -C server test ./internal/haftgane/bot -run '^TestStrengthSmoke$' -v
python3 tools/heavy.py run --kind bench -- \
  go -C server test ./internal/haftgane/bot -run '^$' \
  -bench '^BenchmarkSevenSeatPolicy$' -benchmem
```

The first development smoke (2026-10-23, 300 complete games) used four
independent deal blocks per N, each with the focal policy in every seat.
Against Random, Greedy's mean rank rewards at N=3 to 7 were 1.000, 1.000,
0.975, 1.000 and 0.994; its win credits were 1.000, 1.000, 0.900, 1.000 and
0.964. Apprentice's rank rewards were 1.000, 0.833, 0.838, 0.925 and 0.821;
its win credits were 1.000, 0.688, 0.650, 0.792 and 0.500. Four blocks give
wide uncertainty: Apprentice's N=7 win-credit 95% block-bootstrap interval
was 0.286 to 0.714. These are implementation smoke measurements, not release
claims or evidence of passing the planned strength gates.

Across the same smoke, ordinary-turn Greedy active time was 0.199 ms median,
0.570 ms p95 and 0.979 ms p99. Apprentice was 0.199, 0.503 and 0.887 ms;
Random was 0.0106, 0.0152 and 0.0420 ms. Greedy/Apprentice discard decisions
had p95 0.740/1.301 ms. All 300 games completed with no fallback.

On the loaded Apple M2/arm64 host, three seven-seat mid-Age-I benchmark runs
measured Greedy at 0.673 to 0.916 ms, Apprentice at 0.281 to 0.397 ms and
Random at 0.0115 to 0.0202 ms. All allocated 2,144 bytes in 17 allocations,
including the canonical observation fingerprint and offered-key strings.
The sequential runs experienced different host load; their relative timing
does not imply Apprentice's identical ranking pass is faster. Room queueing,
presentation and full-table scheduling are outside this simulator slice.
Seat-balanced held-out release gates belong to the next simulator slice.


## Seat-balanced simulator

The explicit `tournament -game haftgane` command runs these baseline policies.
Duel continues to use its existing `match`, `tally`, `levels`, `weights`,
`bench` and `replay` commands and report formats. The new command is separate;
its flags and JSONL format do not change Duel's defaults.

Each independent deal block materializes the decks, guild selection, physical
boards and sides once. Blocks cycle through all-day, all-night and random-side
strata. Side selection is complete before any policy runs. Every candidate
replaces one baseline seat on that same setup, keeping all other policies and
their call tapes fixed. Focal seats cover the whole ring. Mixed backgrounds
cycle Greedy, Apprentice and Random as evenly as the count permits, with every
relative offset and both reflected lineups. Reflections never reverse the
engine's actual Age pass direction.

The `release` suite has Greedy against Random, Apprentice against Random and
a paired Greedy-for-Apprentice replacement in balanced mixed backgrounds.
The `diagnostics` suite has the inverse fields and homogeneous self-play for
all three policies. `all` runs both. Identical complete physical lineups share
one game within a block; every declared comparison, focal seat, background
offset and reflection remains in the manifest. This saves duplicate work
without treating those references as independent observations.

Derivation version 1 uses `tag = 5*block + (N-3)`. The setup seed is
`MixSeed(deal_base, tag)`, and the independent policy root is
`MixSeed(policy_base, tag)`. A call receives
`MixSeed(policy_root, 1024*point_kind + 128*age + 8*turn + seat)`, followed by
the bot's observation-fingerprint mixing. Lineup identity, accepted-batch
count, runtime scheduling and the setup seed never enter the policy tape.
An extra ability point cannot shift the tape for a later ordinary turn.

### Complete reports and intervals

Reports stream one block at a time as JSONL, optionally gzip-compressed.
The header records the game, rules and catalog digest, policy/preset/work,
temperature, seed partition and domains, side schedule, counts and suite.
Each block records its fixed setup identity, boards/sides, full comparison
manifest, unique games, accepted batches, terminal score categories, coins
and ranks. Each accepted pick has a compact diagnostic tuple:
`[seat, point_kind, active_ns, iterations, ranked, previews, quotes, draws,
fallback_reason]`. Keys are in the corresponding batch in seat order.
Reason zero means normal completion; this runner aborts on incomplete
policies instead of silently replacing their prescribed work.

A footer pins the block-stream SHA-256 and block count. Tallying requires every
shard and assigned block, rejects duplicates or changed headers, rotations,
lineups and game sets, then replays every accepted key and checks terminal
results, diagnostic cardinality and bounded work. Missing footers, truncated
gzip streams and changed block bytes fail. Equivalent cached lineups are
counted once in work/latency statistics.

Strength uses mean focal rank reward and win credit per deal block, with
all that block's correlated rotations kept together in 5,000 bootstrap
resamples. Paired changes are computed within the same physical seat and
background before block averaging. Reports include per-N intervals,
inverse-field changes, self-play midrank sanity and score/coin distributions
by policy and full seat/board/side/neighbour context. These diagnostic
breakdowns cover unique games; they are not the balanced focal strength
estimators. Phase latency uses one-microsecond bins, exact active-time sums
and maxima. Heap allocation is sampled after each block; this is not peak RSS.

A complete held-out gate report requires the release partition, at least
100 independent blocks at every N from 3 to 7, and the release suite.
Greedy/Apprentice against Random check the predeclared mean targets and
show whether their interval crosses the target. Greedy-for-Apprentice requires
the paired rank-improvement interval above zero at each N. The combined
release flag passes only when all 25 checks pass. A small development or
partial-shard run cannot pass that flag. Failed or uncertain gates are
reported without tuning on the same seeds.

### Guarded commands

Keep controlled experiment files outside the repository. For a short
development check:

```sh
report_dir="${AJAYEB_RUN_DIR:-$HOME/tmp/agents/ajayeb}/logs/haftgane-tournaments"
command mkdir -p "$report_dir"
HEAVY_NOTIFY=auto python3 tools/heavy.py run --kind test -- \
  go -C server run ./cmd/ajayeb-sim tournament -game haftgane \
  -partition development -blocks 2 -suite all \
  -out "$report_dir/development.jsonl.gz" \
  -summary "$report_dir/development-summary.json"
```

For held-out evidence, run shards serially, waiting for each completion
notification before starting the next. Change `0/5` and `release-0` below
to `1/5` through `4/5` and `release-1` through `release-4`:

```sh
HEAVY_NOTIFY=auto python3 tools/heavy.py run --kind test -- \
  go -C server run ./cmd/ajayeb-sim tournament -game haftgane \
  -partition release -blocks 100 -suite all -shard 0/5 \
  -out "$report_dir/release-0.jsonl.gz"
```

After every shard completes, validate and tally the complete set:

```sh
HEAVY_NOTIFY=auto python3 tools/heavy.py run --kind test -- \
  go -C server run ./cmd/ajayeb-sim tournament-tally -game haftgane \
  -check-gates -summary "$report_dir/release-summary.json" \
  "$report_dir"/release-{0,1,2,3,4}.jsonl.gz
```

Output files must not already exist. An interrupted file stays available for
inspection but has no complete footer and cannot enter a tally. Gate failure
returns status 1 after writing the honest summary; invalid command flags
return status 2. Normal run/tally status zero means the experiment is complete,
unless the explicitly requested `-check-gates` also requires strength success.
