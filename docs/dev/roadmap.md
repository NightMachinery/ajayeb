# Roadmap

Ajayeb is built in twelve milestones, M0 to M11. Each milestone ends green (its acceptance criteria pass), committed and pushed. This file changes only when a milestone ends; day-to-day progress is tracked in the private run state.

**Slice ids.** Work inside a milestone is cut into bounded slices (see [orchestration.md](orchestration.md)). A slice id is the milestone plus the slice, such as `M2.S5a`, and it appears in the commit subject and in a `Slice:` line of the commit body. The engine and AI slices keep the numbering below. Other milestones number their slices in order (`M4.S01`, `M4.S02` and so on) when they are planned.

**Standing gates.** On top of each milestone's own criteria, a milestone is done only when `make verify-quick` and `python3 -m unittest discover -s tests` are green, no headless Chrome is left running, and the docs describe what was built.

## M0: Bootstrap

Status: done on 2026-10-01. Every acceptance criterion passed, and the public repository was created and pushed.

Scope:

- Repository skeleton and metadata: `.gitignore`, `.editorconfig`, `.nvmrc`, LICENSE, NOTICE.md, README.
- The agent guide and development docs: AGENTS.md, CLAUDE.md, `docs/dev/*`.
- `tools/heavy.py`, the memory and run-mode guard, with its tests, and the skeleton `Makefile`.
- The Go module skeleton: `cmd/ajayeb` with a health endpoint and the SPA handler, package stubs, the embedded web build, the linter config.
- The SvelteKit SPA skeleton with lint and format configs, and the pnpm workspace.
- The Playwright skeleton with a health smoke test.
- The headless Chrome tools (launch, screenshot, matrix, SVG render), the fast SVG renderer, and the `chrome-cdp` skill.
- GitHub Actions CI for Go, web and e2e.
- The public GitHub repository, created and pushed.

Acceptance:

- `make verify-quick` and the Python tests are green.
- `make e2e-smoke` passes and leaves no Chrome behind.
- `python3 tools/heavy.py status` reports the current memory pressure and the matching slot count correctly.
- CI runs on the public repository.

## M1: Rules and data

Status: done on 2026-10-02, merged with M2's data gate. Two blind verifiers built independent tables of all 95 items: one from the official rulebook, one from the BGA implementation cross-checked with 7wd-io. The three-way check (`catalog/verify_test.go`) compares them with our catalog field by field and reports zero mismatches. Each ruling is recorded in [docs/rules/data-rulings.md](../rules/data-rulings.md). The rulebook prints no cost for the Sphinx, so that cost rests on the two implementations and our rules reference. One reviewer checked `docs/rules/` both for completeness and for copied wording (by matching seven-word sequences against the rulebook text), and a separate prose audit compared every rendered effect with the rules. No owner questions remain open.

Scope:

- Extract the card, wonder and token data from several independent sources, each extractor blind to the others, then reconcile them field by field. Every disagreement is settled by independent refuters, and anything still unresolved becomes a question for the owner.
- Write `docs/rules/` in our own words, with the rulings on ambiguous points in `docs/rules/decisions.md` and provenance and licenses in `docs/rules/sources.md`.
- Produce the reconciled reference data, with the source of every field recorded, which the M2 catalog slices encode and test against.

Acceptance:

- The reconciliation reports zero unresolved fields, or each one is recorded as an owner question with a provisional ruling.
- The counts check out: 23 Age I, 23 Age II and 20 Age III cards, 7 guilds, 12 wonders and 10 progress tokens.
- Two reviewers pass `docs/rules/`: one for completeness, one for the absence of rulebook wording.
- `docs/rules/sources.md` lists every source with its license.

## M2: Engine

Status: done on 2026-10-02. Every slice below landed after an independent review. Measured at the end: engine coverage 94.7% and catalog 97.5%; a sweep of 2000 random and 2000 simple-policy games green with invariants after every action and all three victory types reached; and ten minutes of fuzzing (about 1.9 million playouts) found no crasher. One deviation from the plan: the rules-audit loop ran a single round, in which four finders wrote 94 black-box tests from the rules and none failed, and it stopped there to save model quota instead of waiting for two empty rounds. A deeper audit with stronger finders is queued for M11.

Scope, as slices (each one green and committed on its own):

- S1 foundation: the frozen core types (catalog types and effect language, state, actions, events), age layouts, the RNG and the import guard.
- S2a catalog skeleton: all cards, wonders and tokens with ids, slugs, names, ages and colours.
- S2b catalog complete: costs, effects and chains, plus the catalog dump command.
- S2v verification tables: two blind, independent tables of the data, used by the three-way diff gate.
- S3 cost core: the cheapest-payment solver and its breakdown, checked against an exhaustive oracle.
- S4 flow: setup, the wonder draft, structures and reveals, the three basic actions with payment, end of turn, age transitions with the starter choice, basic scoring, invariants, replay and hash, and the scenario core with test helpers.
- S5a effects: building and token effects, military and looting, science and pairs, scoring and the tie-break.
- S5b wonders: wonder effects, the pending choices they open, and the seven-wonder rule.
- S6 gate: the three-way data diff, data rulings, black-box rules-adversary tests, property tests at full rules, golden replays and fuzz corpora.
- S7 view: `ViewFor`, the per-viewer event filter, the catalog hash and the protocol fixtures.
- S8 observation: the bot `Observation` and `Determinize`.
- S9 scenarios: the named scenarios with their triggers and expectations, used by tests, the dev endpoints and fixtures.
- Then the rules-audit loop: lens-specific finders write suspected divergences as failing tests, refuters check them, fixes land one at a time, until two rounds in a row find nothing new.

Acceptance:

- A sweep of 2000 seeded bot games is green, with all invariants checked after every action (card conservation, coins never negative, termination, replay hash, redaction).
- Ten minutes of fuzzing find no crasher.
- The golden replays are stable, and engine test coverage is at least 90%.
- Every victory type (military, science, civilian) occurs in the sweep.
- The three-way data diff is green.
- The redaction scan finds no hidden id and no seed in any view.

## M3: AI

Status: done on 2026-10-10. Presets are frozen in `server/internal/ai/levels.go` and described in [docs/ai.md](../ai.md). The authoritative measurement is the CI tournament at the levels' fixed budgets (run 38020386167): Greedy beat Random in 100% of 100 paired games, Medium beat Greedy in 68.0% of 100 (95% interval 58.3% to 76.3%), and Hard beat Medium in 96.0% of 200 (interval 92.3% to 98.0%). The first full run had Hard at only 53.5%, because more iterations bought little once the search converged: the leaf value was the limit. Follow-up slice S13 made Hard's rollouts play to the end of the game, and since each iteration now costs about twice as much, its fixed budget went from 500,000 to 200,000 iterations (the low end of the measured range) for the same 2 seconds. The step from Medium to Hard is now large. A fourth level between them is planned, but only after the game is finished (the owner's call).

Scope:

- S10: the bot interface, the Random and Greedy bots, the evaluation function, the search pool, and self-play tournaments in `ajayeb-sim`.
- S11: single-observer ISMCTS with rollouts, root-parallel workers, time and iteration budgets, cancellation and progress reports.
- S12: tuning sweeps, frozen difficulty presets (Easy, Medium, Hard), and the measured win rates recorded in the docs.

Acceptance:

- Greedy wins at least 90% of games against Random.
- Medium wins at least 65% against Greedy, and Hard wins at least 58% against Medium.
- Bots are deterministic under an iteration budget and stop promptly when cancelled.
- The fairness test passes: the same public information with different hidden cards gives the same choice.

## M4: Server

Status: done on 2026-10-10. The protocol tests in `server/internal/protocoltest` play full online, hot-seat and bot games over the real WebSocket, room and store stack. They cover no-leak capture (face-down cards, every age's removed cards and the Great Library offer, checked for both seats and a spectator), reconnect, idempotent resends, undo and restart persistence, and they pass under `-race` together with the room, ws and httpapi packages. `ajayeb-drive` finished 20 concurrent bot games, plus online and hot-seat runs, and the server's peak resident memory during the 20 games was about 19 MB (`--server-pid` samples it). Recorded server messages are generated into `web/src/lib/wire/fixtures.gen.ts` and type-checked against the generated TypeScript types; on its first run this check found a missing type mapping for game events. The security review reported 13 findings across its three lenses. After merging duplicates, 11 were confirmed and fixed, each with a test that failed first, and two suggested extras were refuted; the remaining risks are listed in [docs/protocol.md](../protocol.md). `self_host.py dev-start` has been wired since M0 and its server flags match, but it was not run here, because unattended work never starts a Vite dev server.

Scope:

- Wire types and `tygo` generation; identity and the room store; the room actor with modes, deduplication, undo and bot turns; the WebSocket transport with its limits; the dev endpoints.
- Protocol tests that play full online, hot-seat and bot games; the `ajayeb-drive` command; `docs/protocol.md`.
- `self_host.py dev-start` for local development.
- A security review with three lenses (hidden-information leaks, identity and authorization, denial of service and limits), with refuted findings dropped and confirmed ones fixed.

Acceptance:

- All protocol tests are green, including no-leak capture, reconnect and restart persistence, and the room and transport packages pass under `-race`.
- `ajayeb-drive --games 20` is green.
- The server stays below 40 MB resident with 20 live games.
- `make gen-check` is green.

## M5: Web core and hot-seat

Status: done on 2026-10-10. The client is built from the session layer up: a WebSocket transport with reconnects, the Session and Presenter, the card catalog generated from the engine, placeholder icons in one inline sprite, the card structure sized by `fitStructure`, the board chrome, the game screen with one markup for every viewport class, the card sheet with build, discard and both wonder flows, the draft, every pending-choice panel, the end-of-age summary, the game-over screen, the home screen with the hot-seat and vs-AI setup, settings and the pass-device screen. Two lints keep the client honest: no physical CSS and no hard-coded English. `hotseat.full` plays a whole game from the home screen by tapping the real UI, on the desktop and the mobile project (72 and 71 moves, every pending kind and both wonder flows covered), and `layout.spec` checks nine viewports at the draft, all three ages and game over. The smallest takeable card at 360 by 740 is 46 by 71 pixels, in Age III. The e2e suite passed twice in a row, with no console errors or failed requests. Placeholder art and icons stand in until M9.

Scope:

- The app shell and design tokens, i18n scaffolding, connection and identity, the board components, the choice flows, the result screen, the test hooks and the component gallery.
- Placeholder icons and procedural placeholder art, so the game is fully playable before any real art exists.
- The hot-seat full-playthrough end-to-end test.

Acceptance:

- `hotseat.full` is green on the desktop and mobile projects, and `layout.spec` is green.
- `svelte-check` reports zero errors, and the unit tests for the layout fitting and the display mappers pass.
- The i18n lint (no hard-coded English) and the logical-CSS lint are clean.
- At every tested viewport the board has no horizontal scroll and no page scroll, cards that can be taken are at least 44 by 44 pixels at 360 by 740 in all three ages, and there are no console errors or failed requests.

## M6: Versus the computer

Scope:

- The difficulty picker, bot turns with a visible thinking state and a minimum dwell time, and undo.
- The vs-AI full-playthrough end-to-end test.

Acceptance:

- `ai.full` is green.

Status: done on 2026-10-10. Bots are named by level (Apprentice, Scholar and Master), and while one thinks the turn bar and its strip or panel say so. The client holds the bot's answer until the thinking state has shown for a minimum dwell (700 ms at normal speed, 300 ms with animation off) and then highlights the card the bot takes for a moment before the board changes, so a player can follow every bot move. Undo in a bot room takes back the bot's reply with the player's move. A game menu in every mode leads Home, to Settings and to the log, and a room left that way comes back from Home's Continue until its game ends. The military track is mirrored for a player at the Moon, so their capital is always on their side. `ai.full` plays a whole game against the Apprentice on the desktop project (as the Sun) and the mobile project (as the Moon), with an undo, the thinking indicator, the bot's choice of who starts the next age and the result screen checked; a second test leaves a Scholar game and continues it with the board restored exactly. The whole e2e suite passed twice in a row. Found along the way and queued: an undo pressed in the instant the bot answers can be refused as stale (once in about ten tries), and the move toast covers the bot's strip on phones while it thinks; both are queued for M7's client slice.

## M7: Online

Scope:

- Create, invite (copy, share, QR), join, presence, the reconnect banner and spectators.
- The online full-playthrough test with two isolated browser contexts, a page reload and an offline blip mid-game, and frame capture checked against the room's hidden ids.
- One seeded scenario test per victory type, and a flows test for every special choice: the wonder draft, progress-token choice, the Great Library, reviving from the discard pile, the destroy wonders, the next-age starter choice and extra turns.

Acceptance:

- All end-to-end specs are green twice in a row.

Status: done on 2026-10-10. Online play works end to end in the client: Play online creates a room and shows a lobby with the room code, a copy-link button, the system share sheet where there is one and a QR code of the link; a friend opens the link (or types the code on Home), gives a name and joins, and the wonder draft starts. Moves go through a confirm step, the opponent's strip shows whether they are connected, a lost connection shows the offline or reconnecting state with every move disabled until it is back, a third visitor watches as a spectator with no choices, and a finished game offers a rematch to both players. `online.full` plays a whole game between two isolated browser contexts on the desktop and mobile projects, with a spectator joining partway, a page reload, four seconds offline, and the room's own action log checked against the moves the pages sent. A frame leak check runs over every WebSocket frame each page received: using what the game end reveals and each face-down card's identity once it turns up, it confirms no viewer ever got a hidden card, a removed card, a box guild, wonder or token, the seed, or another player's Great Library offer before they were entitled to it (about 730 face-down checks per game). The checker has its own test that plants each kind of leak in copies of real frames and catches it. `victory.spec` reaches every ending from named scenarios (military and science supremacy, the civilian count, a tie on blue cards and a draw) and `flows.spec` covers every special choice (the draft, progress tokens, the Great Library, reviving from the discard pile, destroying a card, the next age's starter and extra turns). The whole suite, 73 tests, passed twice in a row. One client bug was found and fixed on the way: the transport ignored the browser's offline event while its socket was open, so a move tapped while offline could be lost.

## M8: Deploy

Scope:

- `self_host.py` production mode with its tests, and `docs/self-hosting.md`.
- Deploy to <https://ajayeb.pinky.lilf.ir>, as approved by the owner once online play passes its end-to-end tests.

Acceptance:

- The self-host tests are green.
- `/health` answers on the public URL.
- `ajayeb-drive` completes an online game against production, and the Playwright `@smoke` run (create, join, five moves) passes against it.

## M9: Art

Scope:

- The vector set, made in the Opus render and critique loop: resources and economy, science, chain symbols, UI and category glyphs, ornaments, card frames, badges, the military track, the logo and the favicon.
- Bitmap art from Codex `image_gen`: a pilot in which the owner picks the style anchors, then the batch run with critique and regeneration.
- Approved originals committed under `art/originals/`; AVIF derivatives shipped under `web/static/art/`; contact sheets in `docs/art/`; the art license file in `art/`.

Acceptance:

- In blind identification tests, resource and science icons are recognized 100% of the time and chain symbols at least 90% of the time. A family review signs off on stroke weight, palette and silhouette, and a colour-vision review of board fixtures passes.
- Every asset in the manifest is approved, or has an explicit placeholder recorded.
- Per-asset byte budgets are met, the shipped art totals 5 MB or less, and there are no missing images.
- Mini cards stay legible with art in place: the M5 layout checks and the UX rubric still pass.

## M10: UX polish

Scope:

- The UX review loop: a screenshot matrix over fixtures and live rooms in light and dark themes, objective probes, lens critics, refuters, fixers and a before-and-after judge.

Acceptance:

- The loop runs until two rounds in a row find no major issue, with no open P0 or P1 finding.
- axe reports zero serious or critical violations in both themes.
- The initial JavaScript is below 200 KB gzipped, the board renders 2500 DOM nodes or fewer, and the home page's largest contentful paint is 2.5 s or less on a throttled mobile profile.

## M11: Game feel, tutorial and hardening

Scope:

- Animations driven by server events (card flights, flips, coins, the pawn, looting, tokens, the age deal, the draft, victory), sound, haptics and a speed setting.
- A guided first game against the easiest bot, first-time tips, and a rules reference built from the real components.
- A Persian locale stub with a right-to-left smoke fixture, and app-shell caching.
- Hardening: nightly fuzzing at scale, a further security pass, replay download, and complete docs.
- A deeper rules audit, carried over from M2: lens-specific finders with the strongest model write black-box tests until two rounds in a row find no new divergence.

Acceptance:

- With reduced motion on, no animation runs longer than 10 ms and every state change stays visible.
- Animations are purely visual: the final DOM is the same with animation on and off for scripted sequences, input is re-enabled within 450 ms of an update at normal speed, and no task longer than 50 ms runs during an animation on a throttled CPU.
- A fresh agent that sees only the UI completes the tutorial, and every coach-mark target is visible at every tested viewport.
- The right-to-left fixture has no horizontal scroll, the app chrome is mirrored, and the board stays left-to-right.
- Nightly CI fuzzing and sweeps are green.
