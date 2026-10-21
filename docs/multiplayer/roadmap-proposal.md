# Roadmap proposal: building the second game (M13 onward)

@Opus5.5 [2026-10-21 Wed]. A proposal for the milestones after v1, in the style of [docs/dev/roadmap.md](../dev/roadmap.md). It merges the slice plans of [platform-architecture.md](platform-architecture.md), [engine-architecture.md](engine-architecture.md), [bot-plan.md](bot-plan.md), [phone-layout.md](phone-layout.md) and [art-reuse.md](art-reuse.md) into one plan ordered by dependency. When the owner approves it, its milestones move into `docs/dev/roadmap.md`. The slice plan in [reuse-audit.md](reuse-audit.md) (its "Question 2", with `M13.*` names) is superseded by this one: the platform review replaced its "generalize Duel's room" approach with registered room kinds (D31).

## How to read this plan

- Slice ids follow the roadmap's convention: the milestone plus a short kebab-case name of the outcome. Where a draft already named a slice, the name is kept.
- Each slice carries its review level from [docs/dev/orchestration.md](../dev/orchestration.md): **FULL** (stored games, the engine or rules, the protocol, deploy scripts, security, and anything written by a weaker model, reviewed by the other model family), **LIGHT** (ordinary UI, one cheaper reviewer) or **NONE** (tests, tooling, docs, chores; the coordinator reads the diff). A slice written by a weaker model is FULL whatever its label.
- "Needs" lists what must land first. "Owner first" names an answer the milestone needs before the slice it gates; the questions are numbered and restated in full in [README.md](README.md).
- **Release A** is the playable game: rules, views, the room, Apprentice bots, the client, art, a review loop and a deploy (M13 to M21). **Release B** adds the sampled bots, Scholar and Master (M22).
- **Duel unchanged** means: Duel's full e2e suite and frame leak check, its protocol tests under `-race`, its golden replays and catalog hash, `engine.RulesVersion`, the schema-1 golden files, the recorded version-1 game frames and Duel's generated wire files are all unchanged and green.

## Standing gates

On top of each milestone's own criteria and the roadmap's standing gates (`make verify-quick` and the Python tests green, no headless Chrome left running, the docs describing what was built):

- **No build slice starts before the owner approves this design package.**
- Every slice that touches code Duel runs keeps Duel unchanged.
- Cold Home stays at or below 200,000 gzip bytes, and cold Home and cold Duel entry stay at or below their recorded baseline plus 5,000 gzip bytes, measured with service workers blocked.
- No unverified card value (a draft cost, effect or mockup value) is committed before M13's reconciliation, and no public text shares a run of seven or more words with a saved source.
- New-game rooms cannot be created in production until M21 turns creation on.

## Where work runs in parallel

Four tracks can move at once once the design is approved:

- **Data:** M13, feeding `M15.seven-catalog-data`.
- **Shared layers and room:** M14, then the room actor of M16 (against a fake engine), then the client of M18.
- **Engine and bots:** M15 from its first slice (the core types and the payment solver need no data), then `M16.new-game-engine-contract`, M17, and after `M15.seven-rules-gate` the sampler work of M22.
- **Art:** M19 from M13's names on, one Codex generation at a time.

The tracks share one machine: the heavy guard queues builds and tests, and only one headless Chrome runs at a time, so parallel tracks save agent time, not machine time. Slices that edit the same seam (the four FULL slices of M14, or the engine's frozen files) run one at a time.

## Owner decision points

- Before any build: approval of this package.
- Before `M13.printed-card-reading`: who reads the printed cards (README, open data checks).
- Before M14: when shared-layer work starts. The coordinator earlier recommended letting real Duel games accumulate first; M13 and the first M15 slices touch no Duel code and can start at once. At the end of M14: the go for a Duel-only deploy.
- Before `M15.seven-core-types`: questions 2, 3 and 4 (side choice, payments, sixth-turn timing), because the key grammar and the turn steps freeze there.
- Before `M15.seven-views-and-observation`: questions 6 and 14 (the notebook, running points), because both are view fields.
- Before `M16.new-game-room-actor`: questions 5, 7, 8 and 9 (commitment, recovery, seating, undo).
- Before `M18.home-picker-and-lobby`: questions 12, 13, 15 and 16 (name, crests, recap, bot names).
- Before M21: question 11 (bot levels at the first release) and the go to deploy.
- Before M22 freezes its presets: question 10 (bot quality), with measured strength and waits.
- Question 1 (expansions) blocks nothing while the answer is "base game only"; choosing an expansion adds milestones after M22.

## M13: Rules and data for the second game

Needs: the owner's approval; for the final rules docs, the answers to questions 2 to 4.

Scope:

- `M13.blind-table-publisher` (NONE). A verifier builds a table of all 80 designs and 41 stages from the publisher's card and chain sheet, effects sheet and rulebook. It works blind: it never sees the private drafts, the two held-back working notes (the effect inventory and the edition differences) or the other verifier.
- `M13.blind-table-wiki` (NONE). A second verifier, equally blind, builds the same table from the second-edition fan wiki's costs and boards, cross-checked against second-edition implementations.
- `M13.printed-card-reading` (owner task, NONE). The four unconfirmed costs, the twelve wonder sides the rulebook does not draw and the direction of each trading post's arrow are read from printed cards or the licensed online version. Reading every cost is better still: no publisher sheet prints costs, so the two tables are not independent there.
- `M13.data-reconciliation` (NONE). A field-by-field reconciliation of both tables, the printed reading and the drafts; independent refuters settle each disagreement, and anything left becomes an owner question with a provisional ruling. The reconciled reference and the text of the data rulings stay in the private run directory until `M15.seven-catalog-data` encodes and commits them.
- `M13.rules-docs-final` (FULL). The rules docs in their final place: rules.md with the owner's rules answers, sources.md, the two held-back working notes once reconciled, and the data rulings. Two reviewers: one for completeness, one for rulebook wording by seven-word matching.

Acceptance:

- The reconciliation reports zero unresolved fields, or each one is an owner question with a provisional ruling.
- The counts check out: 80 designs, 77 names and 148 copies; 7 cards per player in every Age at every player count from 3 to 7, with N + 2 guilds in Age III; 24 chain symbols, each provided only in an earlier Age than the card it frees; 7 boards, 14 sides and 41 stages.
- Every field the printed reading was asked for has a recorded value and source, and each trading post's direction agrees with ruling 30, or the ruling changes.
- Two reviewers pass the rules docs, and `sources.md` lists every source with its licence.

## M14: Shared layers, with Duel unchanged

Needs: the owner's approval and start signal. These slices change Duel's production code paths, so they run in this order, one at a time.

Scope:

1. `M14.platform-acceptance-baselines` (NONE). Record schema-1 golden files, version-1 game-frame fixtures, Duel's entry and import graph, cold Home and Duel bytes, the service worker's install transfer and the available mixed-load baselines. No behaviour changes; missing measurements are reported, not invented.
2. `M14.room-file-game-dispatch` (FULL). A bounded schema and game header decides the codec before decoding or replay; schema 1 goes byte for byte to today's path; the production schema-2 table stays empty; `CheckDataDir` dispatches through the existing counters, with the preflight script and report unchanged.
3. `M14.room-kind-registry` (FULL). Create, load and attach dispatch injected into the Manager, with Duel alone registered; the diff stays within the composition root and the Manager's seams.
4. `M14.protocol-game-frames` (FULL). Hello accepts versions 1 and 2; Duel keeps one game-frame contract; the new-game frame DTOs and strict decoders are added (not yet reachable); the Outbox gains a typed `table-state` branch beside Duel's.
5. `M14.client-game-entry` (FULL). Routing by the authoritative snapshot, the one-frame bootstrap behind the existing name prompt, the hand-off to Duel's unchanged Session, an empty lazy entry for the new game, and Saved Rooms' optional game tag.
6. `M14.client-root-import-boundaries` (LIGHT, optional). Break the root's eager Duel card-preview and game-message imports if slice 5 is too large for them.
7. `M14.duel-only-deploy` (NONE, the owner's go). Ship the shared layers with only Duel registered and run `make e2e-prod`, so real Duel games exercise the new seams before any new-game code relies on them.

Acceptance:

- D28 and D29 suites green; schema-1 golden writes unchanged; an unknown game is refused; an old binary loads, reaps and checks a data directory holding schema-2 fixtures without changing their bytes.
- Recorded version-1 game bytes unchanged; Duel plays under both hello versions; a version-1 client joining a new-game room gets the reload path; `make gen-check` green with Duel's generated files unchanged.
- The full Duel e2e and leak suite, generic-entry reconnect and two-tab tests, and old-tab Saved Rooms rewriting are green; a cold invite to an online Duel room with no stored name sends no hello or join and claims no seat until the name is submitted.
- Both cold-entry budgets hold, the precache is unchanged and its install transfer is recorded.
- After the deploy, `/health` answers and Duel's `@prod` specs pass.

## M15: The rules engine

Needs: questions 2 to 4 before slice 2; M13's reconciled tables before slice 5; questions 6 and 14 before slice 11. Slices 1, 4 and 6 move Duel code with no behaviour change, each just before its first caller. Slices 3 and 5 can run in parallel after slice 2.

Scope (the engine design's slices, in its numbering):

1. `M15.purity-guard-sharing` (NONE). Duel's import checker and fixed-value shape checker become test helpers configured per game.
2. `M15.seven-core-types` (FULL). Catalog types, the effect language with per-kind validation, the frozen state, record, action, event and point types, the seat ring and the key grammar.
3. `M15.seven-payment-solver` (FULL). The flat solver, its witness, `Quote`, an exhaustive oracle written without the solver's reasoning, and `FuzzPayment`.
4. `M15.catalog-diff-sharing` (NONE). The three-way diff's mechanics become a test helper with per-game schemas and join keys.
5. `M15.seven-catalog-data` (FULL). The verified data files, registry and compiled tables, the blind tables as the diff's fixtures, structural tests, regression pins for every first-edition value and known source error, the golden CSV and prose dump through `ajayeb-gen catalog -game seven-wonders-2e`, and the list of fields still unconfirmed.
6. `M15.rng-sharing` (FULL). The generator, bounded draws, shuffle and digest helper move into `enginekit`, with Duel's names kept as wrappers.
7. `M15.seven-setup-and-deal` (FULL). Setup, the tagged streams, `SeatOrder`, decks at every N, boards, the three side rules and the side point, dealing, the scenario builder, invariants and the fingerprint encoding.
8. `M15.seven-turn-resolution` (FULL). Points, `LegalState` with per-action free checks before the quote cache, `ResolveState` in seven steps, passing, the sixth-turn leftover, military, Age transitions, the record and replay.
9. `M15.seven-effects-and-scoring` (FULL). Every coin and point kind, science with choice symbols, categories and ranks.
10. `M15.seven-wonder-abilities` (FULL). Babylon's last card, the discard build and its pile look (also when skipped), Olympia's free builds, the Decorators Guild on every side length.
11. `M15.seven-views-and-observation` (FULL). Knowledge, `ViewFor`, disabled reasons, the notebook, both end policies, events, `FilterEvents`, `History`, the DTOs, the observation and its encoding, `Project` and `o.Quote`. It also carries what the screens need and the engine design does not yet list: per neighbour, the units the viewer may buy, its price per unit and the discount's source, and the military outlook (D55); running scores if question 14 is answered yes; and relative seat positions. The bot plan's first slice, "Observed drafting knowledge", is this slice.
12. `M15.seven-rules-gate` (FULL). Golden replays for every board, side and ability path, full sweeps, fuzzers, coverage, benchmarks, a complete `limits.go`, a rules-audit round, and `docs/architecture.md` and the rules docs updated.

Acceptance:

- A sweep of 2000 seeded games per player count, under random and simple policies and every side option, is green with invariants after every point; every effect kind, free condition, point kind and ability path is reached or forced by a scenario.
- `FuzzPlayout`, `FuzzKeys` and `FuzzPayment` run ten minutes each without a crasher; the payment oracle agrees exactly and no construction offers more than 15 outcomes; the science oracle agrees.
- Coverage of `rules` and `catalog` is at least 90%; the golden replays are stable.
- The three-way diff reports zero unresolved mismatches, and the unconfirmed list is empty or owner-accepted.
- The redaction and noninterference checks pass for every seat and the spectator under both end policies, with negative fixtures that prove they can fail; `History` equals the live filtered streams; no view, observation or download carries the seed.
- `limits.go` publishes the tested maxima, and the benchmarks are recorded against the engine design's targets.
- Duel unchanged, including `TestRNGGolden`, the setup and replay goldens and a byte-identical fixed-seed `ajayeb-sim match`.
- The rules-audit round finds nothing new, or its findings land first.

## M16: The new room and its protocol

Needs: M14 slices 2 to 4 for slices 1 and 2 (which run against a fake engine, beside M15); M15 complete for slice 3; questions 5, 7, 8 and 9 before slice 2.

Scope:

1. `M16.platform-interleaving-harness` (NONE). An N-seat `ajayeb-drive` that submits from a common snapshot with randomized arrival, the fake engine contract, failure-injection fixtures, and an independent hand-game leak oracle whose negative fixtures plant a shuffle-derived id, a deal-ordered hand, a raw sold or tucked key, a pending or observation hash and an opponent's offer. Duel's two-seat driver is unchanged.
2. `M16.new-game-room-actor` (FULL). The separate actor with the schema-2 codec, the mixed lobby, private pending picks, point references, replay and undo, claims, recovery, resignation and the one-successor rematch, behind a creation flag that is off. It also fixes the replacement bot's level for takeovers, which no draft specifies (recommended: the lowest level the release offers, frozen at Start).
3. `M16.new-game-engine-contract` (FULL). The real engine replaces the fake; offered-payment bounds, catalog ids and order, memory, setup, the side point, ability order, the end policy and replay compatibility are confirmed against the actor.
4. `M16.new-game-security-review` (FULL). A security pass in M4's three lenses: hidden information across seven seats and a spectator, identity and authorization (claims, takeover, former controllers, rematch), and denial of service and limits. Confirmed findings are fixed, each with a test that fails first. No draft planned this; it follows M4 and M10.

Acceptance:

- Every arrival order on small points gives the same canonical batches and hashes; duplicate and evicted seqs, two tabs, stale submissions after undo, and takeover before and after a locked pick behave as designed.
- Storage errors, a crash before and after rename, restore of every pending subset and the no-human pause are tested; sweeps run at 3 to 7 seats; actor operations and ability-point boundaries are fuzzed with replay invariants.
- Protocol tests play full tables at 3 to 7 seats with a spectator over the real WebSocket, room and store stack, and pass under `-race`.
- The leak oracle catches every planted leak and finds none in real frames; log-capture tests find no unrevealed key, hand, seed or observation hash.
- Duel unchanged; `make gen-check` green.

## M17: Simple bots (Release A)

Needs: M15 for slices 1 and 2 (beside M16); M16 slice 3 for slice 3. Questions 10 and 11 shape slice 3 and the release.

Scope (the bot plan's slices 2, 3, 4 and the Release A part of 9):

1. `M17.greedy-and-apprentice` (LIGHT). Random, deterministic Greedy, noisy Apprentice and the exact offered-key fallback. The seed mixer moves into `enginekit/rng` here, with the same unchanged-output proof as `M15.rng-sharing`.
2. `M17.seat-balanced-tournaments` (LIGHT). Game-selected tournaments in `ajayeb-sim` with deal blocks, seat and neighbour rotations, metrics and complete shard tallies; Duel's commands and reports unchanged.
3. `M17.fixed-preset-room-bots` (FULL). The actor calls the bot package directly through the existing permit pool, one job per room, with durable work rows.
4. `M17.simple-bots-evidence` (LIGHT). The new game's bot documentation: frozen versions, tournament results with intervals, time, memory and fallback figures, and guarded rerun commands.

Acceptance:

- Against Random, over at least 100 deal blocks per player count: Greedy's rank reward is at least 0.85 and its win credit at least 0.70; Apprentice's at least 0.75 and 0.50. Greedy beats Apprentice in their paired comparison.
- The same observation, entropy and work row give the same key and counters across restarts and schedules; changing forbidden information changes nothing.
- Zero to six starting bots, an all-bot continuation, mixed humans, every restore subset, takeover with a full preset row and a kept locked pick, deterministic undo and harmless late results all pass.
- Cancellation leaks no permit or waiter; ordinary runs record zero watchdog fallbacks; the six-bot wait is measured and recorded.

## M18: The client for the new game

Needs: M14 slice 5; M16 slice 2 for the first screens (fake engine) and slice 3 for real play; questions 12, 13, 15 and 16 before slice 2. Procedural placeholders stand in for art until M19, as in M5. The platform design lists the client as one FULL slice; it is split here so that only the hidden-information boundary is FULL and the screens are LIGHT.

Scope (components as named in [phone-layout.md](phone-layout.md)):

1. `M18.table-session` (FULL). The lazy entry's Session and Presenter, typed `table-state` frames, `{seq, point, key}` submissions, locked picks, reconnect and pending restore, and former-controller tabs.
2. `M18.home-picker-and-lobby` (LIGHT). The game picker, New table, the seat list with bots and invites, the settings (seating, sides, end reveal, undo), Start, the rematch lobby and spectating.
3. `M18.table-screen` (LIGHT). The seat ring, far chips, neighbour panels with trade chips and the military outlook, the pass cues, your strip with stage pills, the hand grid, the turn bar and status badges, in portrait, landscape and desktop.
4. `M18.card-sheet-and-payments` (LIGHT). The card sheet with the payment options, the stage payment list, the sale, confirmation and the chain, free and cannot-build variants.
5. `M18.reveal-and-recap` (LIGHT). The grouped reveal, coin changes, the turn recap and the hand pass, with Animation off and the live region.
6. `M18.special-decisions` (LIGHT). The side picker, the discard pick, Babylon's last card and Olympia's free builds.
7. `M18.age-military-and-scores` (LIGHT). The end-of-Age conflicts, the score sheet with its tie-break sentence, the final board and the download under either end policy.
8. `M18.city-sheet-and-notebook` (LIGHT). The city sheet with ring tabs, and the Seen hands sheet if question 6 is answered yes.
9. `M18.recovery-resign-and-spectator` (LIGHT). The takeover bar and its confirmation, resignation, the returning former player and the spectator view, which phone-layout.md describes but does not draw.
10. `M18.sound-and-motion` (LIGHT). Cue planning for this game's events (a decision opening, the reveal, conflicts, the end) within the cue budget, and the animations under D21.
11. `M18.rules-reference` (LIGHT). The new game's rules reference, from Home and the game menu, as Duel has. No draft planned it.
12. `M18.persian-messages` (LIGHT). Persian for every new message, with the native check deferred as it is for Duel. No draft planned it.
13. `M18.table-playthroughs` (NONE). The e2e playthroughs below and a layout spec.

Acceptance:

- Playthroughs green twice in a row on the desktop and mobile projects: the creator alone with six bots; two humans in isolated contexts with bots, a spectator, a reload, an offline blip, a takeover and a resignation; an undo in a solo game; every special decision.
- The leak oracle checks every frame for seven seats and a spectator under both end policies.
- At 3 to 7 seats and at 360x740, 390x844, 844x390 and 1280x800, in both themes and right to left: no horizontal scroll, every control at least 44 px, no console errors or failed requests.
- Tooltip coverage, the i18n lint and the logical-CSS lint are clean; with Animation off every change is instant and visible.
- Both cold-entry budgets hold; a deploy in mid-game keeps the open page's lazy imports working under the all-build precache; Duel unchanged.

## M19: Art for the new game

Needs: M13's verified names for the bitmap subjects; nothing else. Runs beside M15 to M18. Bitmaps are generated one at a time under `heavy.py --kind codex`; vectors are drawn by Opus agents in the render and critique loop (D18).

Scope (counts from [art-reuse.md](art-reuse.md), plus the crests of phone-layout.md):

1. `M19.art-pipeline-game-inputs` (NONE). The manifest, subjects, index and prefetch inputs gain a game-qualified namespace; Duel's outputs and URLs stay byte-identical. art-reuse.md expects this from the platform plan, which does not include it; the reuse audit proposed it.
2. `M19.vectors-resources-and-science` (LIGHT). Ore and cloth, compass and tablet at both sizes, and a small gear: nine files.
3. `M19.vectors-chains` (LIGHT). Ten new chain drawings, and the fourteen reused or aliased ones checked at chip size.
4. `M19.vectors-conflicts-passing-abilities` (LIGHT). Victory and defeat pennants, the pass-hand mark and five ability glyphs: eight files.
5. `M19.vectors-seat-crests` (LIGHT). The seat crests, the seven seat colours and the ore and cloth tokens. phone-layout.md counts five new crests beside Duel's sun and moon, while art-reuse.md marks day and night sides with a sun or moon; this slice settles one meaning per glyph, and the recommendation is seven new crests (two more drawings) so the sun and moon keep their day and night meaning.
6. `M19.bitmaps-pilot` (LIGHT). Ore Vein, Loom and Clay Pit.
7. `M19.bitmaps-production-sites` (LIGHT). Six mixed-production sites.
8. `M19.bitmaps-civic-and-commercial` (LIGHT). Six places.
9. `M19.bitmaps-scholars-and-guilds` (LIGHT). Six subjects.
10. `M19.art-wiring` (LIGHT). Aliases, the approved index, a prefetch plan built from the public catalog only, the Age III back for every concealed Age III card, neutral tucked-stage marks, the wonder board panel, and the placeholders replaced.

Acceptance:

- In blind identification at 12 and 16 px, resources and science score 100% and chain symbols at least 90%; a family review and a colour-vision review pass.
- Every new asset is approved or has a recorded placeholder; per-image byte budgets hold (D24); originals are lossless WebP, pixel-identical to the generated images; there are no broken images.
- Duel's art files, ids and hashes are unchanged; no prefetch, back or image choice depends on hidden information; the per-game icon closure meets its byte gate.

## M20: The review loop for the new game

Needs: M17, M18 and M19. It follows D22 and M11.

Scope:

- `M20.shots-for-the-table` (NONE). `make shots` captures the new game's screenshot set over fixtures and live tables at 3 to 7 seats, light and dark, and right to left.
- Rounds of the loop: scripted probes (horizontal scroll, tap targets, axe, console errors, budgets), a player reviewer who plays real tables at phone and desktop size (alone with six bots, and a mixed table), and a screens reviewer who reads the screenshot set. Each fix is its own slice with before and after shots, reviewed by risk.

Acceptance:

- The loop stops when a round finds only small issues that do not affect real play, with no open P0 or P1.
- axe reports zero serious or critical violations in both themes; a colour-vision review of table fixtures passes.
- Cold Home and Duel budgets hold; the new game's cold entry and the seven-seat table's DOM node count meet budgets set in M18 (proposed: Duel's 2,500 nodes); Home's largest contentful paint stays at or below 2.5 s on a throttled phone.

## M21: Capacity and Release A

Needs: M20; question 11; the owner's go to deploy.

Scope:

1. `M21.new-game-capacity-release` (FULL). Host measurements of seven seats with six bots, an all-bot continuation, ability points, restore, the end reveal, a spectator burst and slow clients, alone and mixed with Duel load (p50, p95 and p99 against the Duel-only baseline); the bot-room-work limit chosen from them; candidate and rollback scans with schema-2 files; retention and backups; creation turned on by default.
2. `M21.release-a-deploy` (NONE; FULL if `self_host.py` changes). Ship with `./self_host.py ship`, add `@prod` specs for the new game, and run them.

Acceptance:

- Zero watchdog fallbacks in the ordinary workload; the chosen limit and its measurements are recorded; serialization and memory stay within bounds derived from the published maxima.
- The installer refuses a rollback while schema-2 live files exist.
- Against production: `/health` answers; a creator alone with bots plays several turns; two browsers at one table with bots play several turns with the confirm step; Duel's `@prod` specs pass.

## M22: Sampled bots (Release B)

Needs: `M15.seven-rules-gate` and M17's tooling; it can start beside M16 to M21, and it ships after M21. Question 10 is answered with the measurements of slices 4 and 5.

Scope:

1. `M22.seven-sampler` (FULL). `Determinize`, `World`, `MatchesObservation`, the constraint model and `MaxSamplerChecks` (the engine design's slice 13).
2. `M22.sampler-integration` (FULL). The bot consumes `Determinize(o, rng) (World, bool)` and `w.State()`, with the no-world Greedy fallback.
3. `M22.simultaneous-age-rollouts` (FULL). Paired root sweeps with observation-limited policies for every simulated seat.
4. `M22.scholar-calibration` (FULL). Fixed iteration constants from full-table measurements, tuning on development and validation seeds, and held-out gates.
5. `M22.master-full-game-rollouts` (FULL). Full-game rollouts under Master's fixed preset; if it fails its gate, Master stays unavailable.
6. `M22.sampled-bots-evidence` (LIGHT). Frozen versions, tournaments, intervals and diagnostics published.
7. `M22.release-b-deploy` (NONE; FULL if `self_host.py` changes). Capacity re-measured with Scholar and Master tables, then shipped.

Acceptance:

- `MatchesObservation` holds for every sample of the sweeps at every player count; on enumerable scenarios the sampler's possible worlds equal those an independent replay accepts, including skipped discard opportunities; the distribution is measured and published; no sweep position exhausts the bound.
- Scholar against Greedy: rank reward at least 0.65 and win credit at least 1/N + 0.10 over at least 100 blocks per player count, with the paired-improvement interval above zero. Master against Scholar: at least 0.58 and 1/N + 0.05 over at least 200 blocks, interval above zero. No player count regresses in the inverse and mixed fields.
- Six-bot waits at the live presets are measured and, if unacceptable, taken to the owner before a level is enabled; p99 cancellation is below 50 ms under a stated load; zero ordinary watchdog fallbacks.
- After the deploy, the `@prod` specs pass with a Master at the table.

## Later, not scheduled

- The private post-round advisor and any hints (bot plan slice 10, D50).
- Pass-the-phone play, deferred by the owner.
- Expansions, if the owner asks for one.
- A guided first game for the new game.
- A host-chosen board deal, which the rulebook allows and the engine's separate board stream makes cheap.
- Crests chosen in the lobby, if question 13 is answered that way later.
- Selective precache, only if install-transfer measurements show a material cost.
