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
