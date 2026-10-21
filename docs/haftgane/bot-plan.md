# Bots for Haftgane (7 Wonders, second edition)

@OpenAI, 2026-10-21. Revised M12 design against the platform revision and engine design sections 11 and 16. This is design only. Timing, memory and strength claims below are proposed checks or explicitly illustrative estimates, not measurements. Duel's bots and presets stay unchanged.

## Recommendation and scope

Offer Apprentice, Scholar and Master with game-specific presets: noisy Greedy, sampled simultaneous play to the Age's end, and full-game rollouts respectively. Random and deterministic Greedy are simulator baselines. Observation-only Greedy and Apprentice first pass their strength, privacy and latency gates as an internal milestone. The public release waits for the sampler and Scholar, as the owner chose on 2026-10-21; Master may follow later.

Use `server/internal/haftgane/bot`, called directly by the separate `server/internal/tableroom` actor and typed to Haftgane's engine. Reuse existing `ai.Pool` permits. The Haftgane bot owns its work/budget type; nothing is shared with Duel's `ai.Budget`. Its evaluator, alternating-turn search, hidden-card sampler and numeric presets do not transfer.

The owner settled one room kind on the same site and binary as Duel, with a game picker on Home. A room has 3 to 7 seats; the creator may add bots to any available seats, combine them with humans joining by link, or play alone with bots. This permits zero to six starting bots. Spectators work as in Duel; pass-the-phone is deferred. The common lobby's ordinary Start requires every seat filled and may explicitly fill empty slots with bots. Bot configuration freezes at Start.

Follow the platform's Undo decision: only a game that started with exactly one human may allow undo, unless the creator disabled it, and undo turns off for good once a second human takes a seat. Disconnects, resignations and seat switches never turn it back on. Eligible undo replays to before the human decision, drops later batches and pending picks, rebuilds private knowledge and deterministic bot inputs, and cancels obsolete work. It never changes a seat's controller or rerolls the deal. No engine checkpoints or retained bot-result cache are needed.

**Decided: equal work at every table size.** Give each difficulty a fixed deterministic allowance per bot decision. Adding bots does not divide or reduce anyone's allowance. This preserves the selected difficulty's policy and computational effort; it does not promise identical win rates against different opponents or at different seat counts. The cost is additional CPU and serialized turn latency. The owner chose this on 2026-10-21 (the platform's Bot quality paragraph).

## Evidence and dependencies

`docs/ai.md`, M3 in `docs/dev/roadmap.md`, and `server/internal/ai/levels.go` establish Duel's pattern: Random, noisy Greedy, short rollouts and full-game rollouts, with separate measured and fixed budgets. Duel Medium uses 40,000 fixed iterations; Hard uses 200,000 with four workers. These units and counts do not transfer. M3's first stronger-budget Hard reached only 53.5% against Medium; full-game rollouts later reached 96.0% on the recorded 200-game tournament. Test rollout quality before assuming more iterations buy strength.

`server/internal/ai/{bot,greedy,eval,rollout,mcts,pool,rank,review}.go` and `server/internal/engine/observe.go` supply useful patterns: observation-only input, repeatable call-local randomness, bounded buffers, cancellation and paired comparisons. Their two-player utilities, opponent-reply minimization and nearly shared hidden information do not describe drafting hands. Duel review continues to use its existing separate semaphore.

The internal milestone needs offered keys, observation quotes, permitted observations, resolution, abilities and scoring; only the stage before the public release adds sampled worlds and sampler integration. The platform supplies durable pending picks, the frozen policy/work ledger and direct pool admission. The rules, effects, edition comparison and reuse audit remain inputs; verified catalog data and tested maxima are implementation prerequisites. No private draft catalog data appears here.

## Draft decision: honest information and bounded sampling

**Decision.** Adopt [engine section 11](engine-architecture.md#11-bots-the-observation-the-state-only-core-and-sampled-worlds) for `Observation`, permitted `Knowledge`, canonical multiset identity and `Determinize`, with the concrete contract confirmed in [section 16](engine-architecture.md#16-the-platform-contract). Every level receives the same honest information: public state, its own current hand and legitimately retained hand/pile knowledge. Difficulty changes computation and noise, never access to real hidden cards or another seat's pending pick. Past possession is not a live view; a seat switch preserves only that seat's knowledge. The notebook statistics a human sees are functions of this observation, so a bot may compute the same figures and receives no other input.

Use section 11's packet groups and pile classes, including skipped-discard constraints. Charge its declared `MaxSamplerChecks` bound in `limits.go` within each comparison iteration. Adopt its enumerable-world distribution checks, without claiming an exact preference-conditioned posterior. Never sample authoritative hidden state.

On feasibility-bound exhaustion, receive no world, never an inconsistent one. Stop that search and use the deterministic observation-only Greedy fallback; do not retry the failed world. Target zero exhaustion on engine sweeps and ordinary release positions; the earlier 1% eligible-pick calibration threshold is only an outer investigation limit, not a substitute for that zero-exhaustion target. Keep this diagnostic separate from forced choices and watchdog fallback.

`Determinize(o, rng) (World, bool)` returns an opaque world on success. Search uses `w.State()`, `LegalState`, `ResolveState` and `QuoteState`; choose actions from `LegalState` on the same frozen state passed to `ResolveState`. Each policy receives only `Project(&sample, seat)` and its new rollout memory; only the root retains supplied `Knowledge`. Omitting opponents' pre-root private memory is an explicit approximation. Search cannot read the world's sampled history or fabricate a `Game`; no policy sees another hand or unseen future deal.

Use `MatchesObservation(o, w)` for test/debug consistency, including historical replay, not in the search hot path. Accept the engine's conservation and distribution checks. Bot tests cover no-world fallback, restricted `Project` inputs, returning hands, hidden sales/burials, duplicate designs, guilds, pile looks and both pass directions. Include skipped-discard constraints: nonempty all-duplicate piles, multiple skips, later additions and rebuilds. Canonical knowledge and keys never expose deal order.

**Reasons.** One engine-owned contract keeps inference consistent with views and replay; bounded failure preserves honest play. State-only rollouts avoid an unnecessary private-history reconstruction dependency.

**Rejected.** A second sampler; real hidden state with random mistakes; shared real-bot hand caches; unbounded whole-sample retries; claiming exact preference-conditioned inference.

## Draft decision: families and simultaneous search

**Decision.** Start with root Monte Carlo comparison rather than an information-set tree. A **comparison sweep** evaluates every shortlisted root key once on the same belief sample and shared policy-randomness tapes. A **simulated round** gathers all required choices from one frozen position, resolves the complete seat-sorted batch, then follows engine-defined ability points. No seat responds to another pick from the same simultaneous round.

- **Random, simulator baseline.** Choose a hand design proportional to its multiplicity, then uniformly among its legal action kinds, then among the kind's offered payment outcomes. For side/ability decisions, choose uniformly among offered keys. This avoids over-weighting a card merely because it has more payment variants. Publish the definition with results.
- **Greedy, baseline and legal fallback.** Rank offered keys by own estimated final-score gain, payment externalities and bounded passing/denial value. It uses only the observation and engine facts/quotes; uncertain same-round benefits remain estimates. It needs no successful hidden-hand sample.
- **Apprentice.** Sample from Greedy values with softmax noise. Begin tuning at 1.5 victory-point equivalents, borrowing the interpretable scale from Duel rather than claiming its temperature transfers. Freeze the temperature after held-out calibration.
- **Scholar.** Run paired sweeps with observation-limited policies for every simulated seat. Continue through the current Age's conflicts and abilities, then evaluate future value. Select highest mean multiplayer reward; canonical key order breaks exact ties.
- **Master.** Use the same framework with a larger fixed allowance and full-game rollouts, including hypothetical future Ages. Exact final scoring and the coin tie-break determine rewards. Demonstrate its improvement at its actual production preset.

Use independently self-interested opponent policies drawn from a small economic, science, military and balanced mixture. A preference persists within a rollout so a seat can complete a strategy. It depends on that simulated seat's public city and own information, not a real opponent difficulty oracle, another bot's pending key or a human-targeting bonus. At every simultaneous point build all observations before generating any picks, use seat-specific random streams, and resolve once. Sequential ability and side choices follow their own permitted observations; side selection cannot inspect future hands.

Search all root keys when the bounded offered set makes that practical. Otherwise retain at most 16 keys: Greedy's best, best construction/stage/sale where available, then high-valued alternatives including differing payment recipients and useful denial cards. Use only offered canonical keys. Equivalent producer assignments and indistinguishable duplicate copies do not create new strategies; recipient-distinct Pareto payment outcomes can. Never search an unoffered overspending variant. Record exclusions and compare 8, 16 and all-key variants offline.

Only complete sweeps contribute means. If none completes, use Greedy. Randomness is indexed by sweep, logical future point and seat, so changing a root candidate cannot shift an unrelated opponent's random tape. Fixed termination, rather than elapsed time, determines which comparisons count.

For scores ordered by total points then coins, define **rank reward** as `(L + T/2)/(N-1)`, where L opponents finish strictly below the seat and T tie it exactly. It is 1 for sole first and 0 for sole last. Define **win credit** as `1/k` for each of k tied winners, otherwise zero. Start with terminal reward `0.75*rank reward + 0.25*win credit`; tune and freeze that blend with the preset. At Scholar's cutoff apply the same ordering to each seat's estimated final score using its own simulated observation. Exact terminal scoring replaces estimates.

**Reasons.** Paired root comparison has bounded storage and a clear information boundary. Age versus full-game horizon gives the levels a substantive difference. Rank reward values improving a losing position without abandoning the chance to win.

**Rejected.** Duel's alternating minimax or `r`/`1-r` backup; a coalition of opponents; enumerating every joint action combination; more iterations as the sole proof of a stronger level. A simultaneous information-set tree is a later candidate only if it wins at equal per-bot work and justifies its memory and complexity.

## Draft decision: fixed work, existing permits and frozen rows

**Decision.** Freeze a fixed iteration allowance for each difficulty and decision kind under a versioned preset. For Greedy/Apprentice an iteration is a complete bounded ranking pass; for Scholar/Master it is one bounded paired comparison attempt. Each attempt charges `MaxSamplerChecks` and has a rollout length bounded by verified remaining game/ability maxima. Normal accepted play completes the preset's prescribed work. Neither table size, bot count, missing submissions, queue pressure nor acceptance order reduces that allowance.

Actual iteration constants require the implemented engine and measurements; this design invents none. Calibrate at seven seats and verify both strength and wait there before freezing the preset. Pin deterministic candidate/rollout bounds and the engine sampler's bound/version with it. The same iteration count may take more CPU at a larger table because each simulated round has more seats; equal effort settings are not an equal-duration guarantee. Pool wait, pauses and presentation pacing are outside the allowance.

**Illustrative latency only.** If a Master pick takes 150 ms active search time, two serialized bots take about 300 ms and six about 900 ms, before queueing, legality, persistence, abilities and presentation. A seven-bot continuation after resignations would take about 1.05 seconds on that assumption. These are arithmetic estimates, not measured presets or latency commitments. Actual turn wait depends on the host, engine and competing rooms; measure it. If full-table wait is unacceptable, bring the measured strength-versus-wait tradeoff to the owner rather than silently dividing the allowance.

For each seat at each logical point, persist an **effective work row** before launch: policy version, difficulty/preset, exact iteration allowance and the versioned bounds needed to reproduce it. Rows and accepted picks survive restart. A restored accepted pick is not searched again. An unaccepted seat retains its original row regardless of which other seats committed first. Work is never donated or repartitioned. A switch to a bot keeps the seat's pick and confirms it; otherwise the new bot gets the full fixed preset row, persisted before launch. Such a switch adds its fixed work to the turn's total. It uses the preset of the level the admin chose, frozen in the game's configuration, not the latest deployed default.

Support persisted policy/preset versions or reject incompatible creation/load explicitly. A deployment cannot silently replace an open point's policy or work. Undo rebuilds the corresponding logical-point inputs from the frozen configuration and durable control history, retaining the seat switches. Replay of completed games uses accepted keys and never needs to rerun bots. Do not retain bot-result caches.

The Haftgane actor uses `ai.Pool.AcquireFor(ctx,key)` and `ReleaseFor(key)` directly, with at most one queued or active bot job per room. Release after one pick and requeue for the next seat. Use the smallest attached human participant address for existing address fairness; unrelated spectators cannot choose it. Follow the platform's measured Haftgane bot-room-work limit without occupying an AI permit while waiting for room-work admission. Duel's `ChooseFor`, pool settings and separate review semaphore remain unchanged.

Use one runtime task generation. On return, check generation, logical point, seat, current controller, pending vacancy and offered-key membership. Unrelated presence or revision changes do not invalidate an otherwise valid answer. Cancel on point closure/undo, a controller change affecting the task, shutdown or loss of the last attached human participant. An original participant watching after resignation may keep play active; unrelated spectators cannot. Preserve every other seat's durable pick.

Keep root statistics and bounded scratch rather than a tree or retained sample cache. Derive storage bounds from verified engine maxima, candidate count and the section-11 sampler contract, then measure peak heap, allocations and pool hold time. No speculative MiB reservation or deployment-wide scheduling change is a prerequisite.

Cancellation checks occur before admission, between candidates, within bounded sampler chunks and after every simulated batch/ability, including inside a full-game rollout. Propose p99 cancellation return below 50 ms wall time under a specified ordinary host load; verify and publish the load rather than imply a universal guarantee. Queued cancellation removes its waiter and releases any room-work permission.

A cancelled or watchdog-truncated search yields an incomplete status, never a partial ranked incumbent. Structural cancellation accepts no replacement. On a still-valid point, an exceptional watchdog may submit the deterministic observation-only fallback after normal checks; persist its actual key and a bounded safe diagnostic reason. Require zero watchdog fallbacks in the ordinary release workload. Sampler-bound fallback is separately deterministic and measured. Do not spin-retry expensive failed searches.

**Reasons.** Fixed work preserves difficulty when the creator adds bots. Existing permits already supply admission fairness; one room cannot queue six requests ahead of another. Durable rows prevent restart or a seat switch from silently changing a pending choice.

**Rejected.** Dividing one room allowance; time-limited strength as normal policy; simultaneous searches within a room; recomputing work from the missing seats after restart; cheaper work for a bot switched into a seat; expanding Duel's review or bot scheduling as part of this feature.

## Draft decision: evaluate this game's scoring and economy

**Decision.** Rules, legal payments and exact scoring stay in the engine. The bot adds transparent future-value estimates in victory-point equivalents and named diagnostic feature deltas. Avoid counting exact current points again as future bonuses.

- **Earned points and coins.** Use exact score categories and terminal `floor(coins/3)`. Earlier, add a diminishing liquidity premium for useful affordable future choices. Price reserve payments and coins sent to each neighbour. Same-round income cannot fund the current pick. Avoid rewarding endless sales through a constant coin premium.
- **Military per neighbour.** Forecast each adjacent conflict separately, using shield differences and that Age's actual win/defeat values. Loss-to-tie differs from tie-to-win. Account for remaining opponent growth; replace forecasts with realized conflict points at Age end. There is no linear shield reward or supremacy bonus. At three seats count both other seats once each.
- **Science squares and sets.** Exact marginal scoring includes squares, complete sets and optimal wildcard assignment. Forecast attainable missing/duplicate symbols using eligible multiplicities, own hand and legitimate memory, discounted for competition and pass distance. Babylon/Scientists Guild flexibility is not a fixed symbol or Duel-style distinct-symbol threat.
- **Production, trade and chains.** Value marginal future affordability, seven resource types, diminishing duplicates, choice production, left/right discounts and chain opportunities. Owner-only yellow/wonder production is not tradeable. A neighbour's use does not consume supply available to others. Dependence costs liquidity and may enrich a rival; it does not risk the removal of permanent production. A chain with no reachable eligible target has no option value.
- **Wonder progress.** Include built effects once, then discounted attainable stages after costs and consumed picks. Distinguish two-, three- and four-stage sides and completion guild value. Value Babylon by remaining extra cards, Halikarnassos by a permitted offer or sampled discard distribution, Olympia by eligible future free builds and Alexandria by coverage. The engine handles their actual timing and legality.
- **Guilds and commerce.** Apply the correct own/neighbour/stage scopes, including self-counting cards through engine scoring. Forecast only incremental growth. Same-round Vineyard/Bazar income uses sampled joint placement and correct ability order, not actual hidden picks. Duel's maximum-of-two scope does not transfer.
- **Passing and denial.** Use the Age's actual receiver. Withholding a useful card can matter against a specialist or leader, but cap the heuristic term so it cannot dominate one's own city. At smaller tables hands can return; passing a card is not always permanent loss. Full-game rollouts should capture much of this effect.

Begin with interpretable weights and independent scenarios for science completion, military thresholds, manufactured-resource scarcity, recipient-distinct payments, completion guilds and second-edition wonder abilities. Keep evaluator, rollout mixture, utility blend, shortlist and sampler parameters separate. Use paired equal-preset experiments and feature ablations, not Random-only tuning. Compare Age/full-game horizons and own-score/rank utilities explicitly.

Partition seeds into development, validation and untouched release sets, stratified by N and board side. Freeze weights, temperature, mixture, sampler, horizon and all work bounds as one bot-policy/preset version, separate from engine rules/catalog versions. Publish measured strength, time, memory, fallback rate and reproducible guarded commands in future game-specific AI documentation. No learning from live private games is needed.

**Reasons.** Nonlinear portfolios, two neighbours and directional drafting need different incentives from Duel. Transparent features expose strategic mistakes before more search conceals them.

**Rejected.** Duel weights; early raw-score-only play; linear shields; all production treated as tradeable; full value for every unbuilt stage; tuning on final acceptance seeds.

## Draft decision: seat-balanced strength and privacy evaluation

**Decision.** Extend `ajayeb-sim` with explicitly game-selected N-player tournaments, calibration, ablation and replay, preserving Duel commands/output. Identify policy/preset/catalog versions, exact work rows, lineup, player count, seed block, board/seat assignment, scores/coins/ranks, accepted keys and per-pick work/time/fallback diagnostics in controlled experiment results. Private observations and low-entropy hashes never enter ordinary server logs or client frames.

For every N from 3 through 7, run one stronger bot against N-1 weaker bots, the inverse field, balanced mixed lineups and homogeneous self-play. Include directed military/science specialists, neighbour placements, liquidity traps and both pass directions. Self-play's average midrank `(N+1)/2` is a reporting sanity check, not strength evidence.

A **deal block** fixes all decks, guild selection, board sides and physical seat assignment. Compare candidate and baseline on exactly that setup, keeping policy randomness stable for unchanged seats. Rotate the focal policy through every seat and rotate background relative offsets to cover both neighbours and distance. Add reflected lineups as separate treatments while preserving the real Age pass rules. Balance board sides across blocks. Changing a policy must not change its paired deal.

Shard by block id; merging fails on missing/duplicate blocks, lineups or rotations. Report average midrank, rank reward, fractional win credit, score/coin distributions and paired changes, broken down by N, seat, board side and adjacent policies. Include score categories and sale frequency for diagnosis. Use 95% block-bootstrap intervals that keep a deal's correlated rotations together; treating every rotated game as independent overstates certainty.

Predeclare these provisional release targets in M3's spirit, then leave release seeds untouched:

- **Greedy/Apprentice against Random:** at every N, Greedy rank reward at least 0.85 and win credit 0.70; Apprentice at least 0.75 and 0.50 respectively, over at least 100 independent deal blocks per N. Greedy must outperform Apprentice in their paired mixed-lineup comparison.
- **Scholar against Greedy:** rank reward at least 0.65 and win credit at least `1/N+0.10`, over at least 100 blocks per N. The 95% interval for paired rank improvement over replacing that seat with Greedy must be above zero.
- **Master against Scholar:** rank reward at least 0.58 and win credit at least `1/N+0.05`, over at least 200 blocks per N, with the paired-improvement interval above zero.
- **Inverse/mixed fields:** positive paired rank improvement with a 95% interval above zero in the combined predeclared suite, and no player count with a point-estimate regression.

Run every gate at the same fixed per-bot preset used in live play. There is no separate smaller six-bot allowance. Show point estimates and intervals; mark a numerical target uncertain if its interval crosses the threshold even when its point estimate passes. Never average away a failing N or substitute a reduced-budget proxy for the production result.

Measure seven-seat/six-bot starting tables, all-bot continuation after resignations, side/ability points, mixed difficulties, every pending-pick restore subset and mid-point seat switches. Record active search time, pool wait/hold time, room-work wait, accepted-pick-to-reveal latency, grouped presentation delay, heap/allocations, serialization, restart time and fallback reasons separately. Compare mixed Duel/Haftgane p50/p95/p99 action and bot latency against Duel-only baseline. Choose the platform's one Haftgane bot-room-work concurrency setting from this evidence, not a speculative resident-room cap. Bring measured unacceptable waits or regressions to review before enabling a difficulty.

### Determinism and cancellation acceptance

Derive policy entropy independently from the setup root under a domain label. Derive the call random tape from game id, policy version, logical point, seat and `Observation.Fingerprint()`, the canonical permitted-observation hash defined in [engine section 11](engine-architecture.md#11-bots-the-observation-the-state-only-core-and-sampled-worlds). Bots receive only derived entropy, never the setup root. Exclude the real full-state hash, pending keys, wire point reference, runtime task generation, clock and arrival order. Use that engine encoding directly, including canonical remembered information, rather than invent a second bot hash. This matches the platform and section 16.

For fixed observation, entropy, policy/preset and effective work row, repeated calls, restart and surrounding goroutine schedules must produce the same key, ranking and work counters. Production uses one worker. Cross-architecture comparisons require stable rounding or fixed-precision comparisons if floating-point ties diverge. A future parallel simulator may distribute indexed complete sweeps only if deterministic merging leaves results unchanged.

The fairness condition is **same permitted observation, policy entropy and work row, same result**. Equal public cities alone are insufficient: a different own hand or legitimate memory may change the decision. Construct reachable pairs with identical public history, own hand, historical hand/pile knowledge and offered keys, but different forbidden opponent hands, sold/tucked cards, future deals or excluded guilds. Assert equal observations, samples under identical sampler randomness, feature diagnostics, rankings and choices for every level.

Also vary accepted hidden opponent keys, arrival order, public confirmation timing, a human's pick changes and withdrawn confirmations, presence, transport revision and task interleavings without changing permitted bot inputs. Results must match. Change own legitimate knowledge in separate positive tests to prove it was not erased. Check no cache or progress callback transfers private information across real bot seats.

Inject cancellation before pool acquisition, inside sampling, between candidates, within long rollouts/abilities and just before delivery. Test undo, replacement, shutdown and participant loss; verify prompt return, no permit/waiter leaks, no stale/partial acceptance and preservation of other durable picks. For restore and seat switches, assert exact persisted policy/work rows and accepted keys, not merely that play eventually resumes. Force exceptional watchdog fallback separately and require none during ordinary acceptance runs.

**Reasons.** Neighbours and drafting direction can reverse rankings; paired deal blocks reduce noise without pretending rotations are independent. Fixed-preset all-count tests separate strength from scheduling and latent information leaks.

**Rejected.** Two-seat swaps as the only pairing; raw score alone; high-budget offline evidence standing in for live settings; hidden-state-derived randomness; privacy tests that omit legitimate own memory.

## Draft decision: defer hints and private post-round advice

**Decision.** Ship no Haftgane hint families or advisor initially, matching the revised platform. Duel's D23 settings/review stay unchanged. Later work may retain D23's room-owned admin settings and local hiding, but hand-dependent facts must be built inside the owning seat's projection from permitted information. No shared private hint object or pre-pick best-move suggestion is introduced.

A future advisor is private post-move learning. Propose its default on for games starting with one human and off with several, preserving an explicit room choice thereafter. Deliver only after the simultaneous batch and dependent ability points resolve. Alternatives and private explanations go only to the reviewed human, never spectators or other seats.

Review the immutable pre-commit observation. Compare played/proposed keys on paired belief samples with identical sampled opponent picks at that simultaneous point and shared future policy tapes. Do not condition on actual hidden submissions or later revelations. Each ability review uses its own observation; an earlier stage choice cannot borrow a later private pile offer. Bot interfaces needed are bounded root ranking, paired comparison, completion/uncertainty status and visible-fact feature deltas. Calibrate a meaningful multiplayer gap on independent samples; forced, incomplete or uncertain comparisons remain quiet.

Any future advisor needs its own scheduling/latency review before implementation and must not delay acknowledgement or resolution. This plan changes no Duel review path. Factual legal quotes and current score/conflict deltas can precede predictive advice because they need engine facts rather than extra search.

**Reasons.** Honest playable bots and measured full-table performance come before optional learning searches. Post-round delivery and pre-commit information avoid hindsight criticism and disclosure during simultaneous choice.

**Rejected.** Enabling Duel hint ids without implementation; public criticism exposing a hand; true-deal counterfactuals; reviewing bot moves; copying Duel's numerical confidence threshold.

## Ordered implementation slices

These are cohesive code-and-doc groups; LIGHT means focused independent review, FULL also covers rules/information and resources. Follow [engine section 14](engine-architecture.md#14-slices-and-the-release-sequence): the internal milestone covers 1 to 4 with applicable evidence from 9; the public release also needs 5 to 7 and 9, with Scholar the strongest level offered; 8 may follow later. Sampler integration gates only sampled difficulties, never Greedy/Apprentice.

1. **Observed drafting knowledge**, FULL. Engine-owned immutable observation and historical knowledge with canonical catalog multisets. Accept reachable privacy pairs, replay/undo/restart equality, no raw private-history exposure and tested maxima.
2. **Greedy baseline and Apprentice**, LIGHT. Random, deterministic Greedy, noisy Apprentice and exact offered-key fallback. Accept all-count/all-phase legal play, science/conflict/payment/wonder scenarios, bounded ranking and repeatability. Internal play waits for the following tournament tooling and strength gates; public play waits for Scholar.
3. **Seat-balanced tournament tooling**, LIGHT. Fixed deal blocks, neighbour/seat rotations, independent policy entropy, metrics and complete shard tallying. Accept synthetic outcomes and unchanged Duel reports. This precedes tuning or public strength claims.
4. **Fixed-preset room integration**, FULL. Call the Haftgane bot package directly from the separate actor using existing pool permits, one job per room and durable effective work rows. Accept zero-to-six starting bots, seventh-bot continuation, mixed humans, restore subsets, seat switches in both directions with full presets, pick preservation, deterministic undo and harmless late results. No Duel scheduling changes.
5. **Engine sampler integration**, FULL, before the public release. Consume `Determinize(o, rng) (World, bool)` and `w.State()`. Accept `MaxSamplerChecks`, no-world fallback, `MatchesObservation(o, w)`, skipped-discard constraints, distribution checks and restricted `Project` inputs.
6. **Simultaneous Age rollouts**, FULL. Paired root sweeps and independent observation-limited policies. Accept frozen simultaneous choices, ability order, common randomness, shortlist ablations, work/storage bounds and cancellation within rollouts.
7. **Scholar calibration**, FULL. Choose fixed iteration constants from full-table measurements, tune on development/validation seeds and run held-out strength/latency/privacy gates. Accept exact persisted preset versions, bounded storage, prompt cancellation and zero ordinary watchdog fallbacks. Equal per-bot work, as the owner decided.
8. **Master full-game rollouts**, FULL. Complete hypothetical future Ages and terminal scoring under its fixed preset. Accept all-count strength, endgame/ability/wildcard scenarios and measured six-bot/mixed-load waits. If it fails, leave Master unavailable rather than silently reducing its work when bots are added.
9. **Release evidence and documentation**, LIGHT. Publish frozen versions, complete tournaments, uncertainty, phase-level time/memory/fallback diagnostics and guarded rerun instructions. Accept no hidden-data disclosure, honest unresolved failures and unchanged Duel regression coverage for shared seams.
10. **Private facts and post-round advisor**, FULL, deferred. Separate facts from estimated comparisons and delivery. Accept D23 ownership, private projection, paired pre-commit reasoning, later-offer isolation, omission/cancellation and independently reviewed runtime cost before enabling it.

No builds, tests, tournaments, host probes or product changes were performed for this revision. Implementation measurements must use the repository's guarded make/heavy workflow. No owner choice remains open here; actual calibration follows implementation.

## Owner answers (2026-10-21)

- **Name.** Haftgane in the title and prose; package `server/internal/haftgane/bot`.
- **Release order.** Greedy and Apprentice form an internal milestone; the public release waits for the sampler and Scholar (slices 5 to 7 and 9); Master may follow.
- **Bot strength.** Equal fixed work per bot, decided.
- **Undo.** Off for good once a second human takes a seat.
- **Admin powers.** Seat switches in both directions replace the takeover in the work rows, restore checks and slice 4.
- **Commitment and notebook.** Human pick changes and withdrawn confirmations are among the inputs that must not change a bot's decision; the notebook statistics give bots nothing new.
