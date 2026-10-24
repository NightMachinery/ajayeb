# Roadmap

Ajayeb's first game, Duel, was built in twelve milestones, M0 to M11, and reached v1 with M11. Its second game, Haftgane, was designed in M12 and is built in M13 to M23. Each milestone ends green (its acceptance criteria pass), committed and pushed. This file changes only when a milestone ends; day-to-day progress is tracked in the private run state.

**Slice ids and other names.** Every shorthand name must tell a reader what it is without a lookup.

- A milestone is `M0` to `M23`. The first mention in any report, brief or document also gives its title, such as "M10 (game feel, tutorial and polish)".
- Work inside a milestone is cut into bounded slices (see [orchestration.md](orchestration.md)). A slice id is the milestone plus a short kebab-case name of the slice's outcome: two or three plain words, such as `M10.security-pass`, `M10.tooltip-coverage` or `M10.persian-rtl`. No letter-and-number codes. When the same kind of slice recurs, the name takes a number: `M10.hints-followups-2`.
- The slice id appears in the commit subject, as in `server: security headers on every response (M10.security-pass)`, and in a `Slice: M10.security-pass` line of the commit body. Its branch is `slice/security-pass` and its worktree directory `M10-security-pass`.
- Slices launched together form a wave, numbered within its milestone and named by its theme on first mention: "M10 wave 4 (security, tooltips, offline shell, Persian)".
- Older history keeps the codes it was committed with: the engine and AI slices below (`M2.S5a`), and the early M10 slices (`M10.HS4`, `M10.F1`). Published commits are never renamed.

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

Status: done on 2026-10-02. Every slice below landed after an independent review. Measured at the end: engine coverage 94.7% and catalog 97.5%; a sweep of 2000 random and 2000 simple-policy games green with invariants after every action and all three victory types reached; and ten minutes of fuzzing (about 1.9 million playouts) found no crasher. One deviation from the plan: the rules-audit loop ran a single round, in which four finders wrote 94 black-box tests from the rules and none failed, and it stopped there to save model quota instead of waiting for two empty rounds. A deeper audit with stronger finders is queued for M10, beside the UI build.

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

Status: done on 2026-10-10. The production host is small and shared (one vCPU, little free memory and disk, and an older Go than the server needs), so instead of building there, `./self_host.py ship TARGET URL` builds the web bundle and a static Linux binary on the developer's machine, uploads the binary and `self_host.py`, and runs `setup` or `redeploy --prebuilt` on the host. A new binary is checked before the running server is stopped, the previous one is kept for a rollback, and the Caddyfile edit is validated before Caddy reloads (with a backup). The self-host tests (154) are green. On the public URL, `/health` answers with the deployed version, plain HTTP redirects to HTTPS and the development endpoints answer 404; the server's resident memory is about 8 MB. `ajayeb-drive` completed an online, a hot-seat and a Hard bot game against production (the Hard bot takes about 2 seconds a move on the single vCPU). The `@prod` Playwright specs (`make e2e-prod`) passed twice against it: two browsers create and join an online game through the UI and play five moves with the confirm step, and a third test plays against the Apprentice. When `E2E_BASE_URL` points at a real site, the config runs only `@prod` specs and starts no local server.

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

Status: done on 2026-10-11, with four bitmaps left on their placeholders. The vector set was drawn by Opus agents in the render, screenshot and critique loop described in `art/vector/STYLE.md`: resources, economy and military, science, chain symbols, category glyphs, ornaments, the military track, the wordmark and the favicon, 75 files that `make vec-lint` checks in `verify-quick` and in CI. In truly blind readings by fresh agents that saw only the sheets and a list of names, resources scored 14 of 14 at 12 px (DPR 2) and 16 px (DPR 1), science 7 of 7 at three sizes, chain symbols 17 of 17 and category glyphs 15 of 15; a critic pass and a polish round followed, and a fresh blind round after the polish still scored 100%. Every set's specimen was checked in greyscale and under deuteranopia, protanopia and tritanopia emulation; a colour-vision pass over whole board fixtures moves to the M11 review loop. The bitmaps came from a 27-image pilot whose critique was folded into `art/style.md` and the subjects, then a batch of 109 assets in which an independent critic judged every image, for up to three rounds of regeneration. 105 were approved and shipped. Four are marked `needs-human` in `art/manifest.json` and keep their procedural placeholders: the Statue of Zeus, the Masonry token, the Tavern and the home hero. The shipped AVIF art totals 2.64 MB in 200 files (budget 5 MB), the per-age prefetch is 106 to 138 KB (budget 200 KB), every per-asset budget is met, and the approved originals are committed as lossless WebP, pixel-identical to the generated images. The full e2e suite, including `layout.spec`, passed on CI with the art in place (78 passed). An in-app check found no broken images, console errors or failed requests. Left for M10: the avatars, victory scenes, textures, app icon and home hero have no consumer in the app yet (about 229 KB shipped unused); owned progress tokens render greyed; some names break mid-word on medium cards; the phone card sheet squeezes the art for long names; two wonder crops; and a follow-up art round for the four placeholders and the Great Library (its large gold dome breaks the style's gold rule).

The owner merged the review passes of the old M10 (UX polish) and M11 (game feel, tutorial and hardening) on 2026-10-11 ([D22](decisions.md)): the features are built first, then one review loop runs over the finished game. M10 and M11 below follow that split.

## M10: Game feel, tutorial and polish (build)

Scope, built as ordinary slices:

- The follow-up art round (the four placeholders and the Great Library), the unused art wired in (bot avatars in the setup sheet and the bot's strip, victory scenes on the game-over screen, the paper and night textures, the app icon and web manifest, the hero on Home) and the in-app art fixes found at the end of M9.
- The queued UX fixes from M5 to M9 (tap targets, raw colours, overlaps and truncations, the ornament, track and wordmark integration).
- Tooltips on every board element and icon-only button, down to each symbol, chip, pip and glyph on a card, through our own helper, following the AGENTS.md rule, with vector concept diagrams where a picture explains more than a sentence; a lint keeps the native `title` attribute out.
- A rules reference built from the real components and the tooltip diagrams: a side drawer on desktop that leaves the board visible and live, a tall bottom sheet on phones, and the same content at `/rules` from Home. Tooltips link to the matching section.
- Animations driven by server events (card flights, flips, coins, the pawn, looting, tokens, the age deal, the draft, victory), sound, haptics and the speed setting, all controlled by the game's own Animation setting ([D21](decisions.md)).
- An opponent move summary: a Settings option, Show opponent move in detail (on by default), opens a popup that summarises the opponent's last move, consolidated when they took several turns in a row, and any tap closes it; a button by the log shows it at any time. The same summary tells a player who reconnects what happened while they were away.
- A guided first game against the easiest bot, and first-time tips that reuse the tooltip layer.
- A Persian locale stub with a right-to-left smoke fixture, and app-shell caching.

Beside the UI build, outside the review loop because they do not touch the screens:

- A deeper rules audit, carried over from M2: lens-specific finders with the strongest model write black-box tests until two rounds in a row find no new divergence.
- Hardening: nightly fuzzing at scale, a further security pass, replay download, and complete docs.

Acceptance:

- With Animation off in Settings, no animation runs longer than 10 ms and every state change stays visible.
- Animations are purely visual: the final DOM is the same with animation on and off for scripted sequences, input is re-enabled within 450 ms of an update at normal speed, and no task longer than 50 ms runs during an animation on a throttled CPU.
- A fresh agent that sees only the UI completes the tutorial, and every coach-mark target is visible at every tested viewport.
- Every board element, every symbol on a card and every icon-only button has a tooltip (or a visible label it would only repeat), no native `title` attribute remains, and the rules reference opens from the game menu at every tested viewport without leaving the game.
- The right-to-left fixture has no horizontal scroll, the app chrome is mirrored, and the board stays left-to-right.
- Every asset in the manifest is approved, or its placeholder is a recorded owner decision.
- Nightly CI fuzzing and sweeps are green, and the rules audit's last two rounds find no new divergence.

Status: done on 2026-10-18, with one acceptance criterion left open: no long task over 50 ms during an animation on a throttled CPU. Every slice merged after an independent review. The follow-up art round closed the four placeholders and the Great Library, so all 109 assets in the manifest are approved. The art the app had no use for is now wired in: bot avatars, victory scenes, the paper and night textures, the app icon and web manifest, and the hero on Home. The shipped art is 220 AVIF files totalling 4.22 MB. D24 dropped the 5 MB total cap; the M9 figures above stay as history. Each game carries its own hint settings, edited by its admin (D23). First-time tips and the opponent move summary reuse the tooltip layer. `tooltip-coverage.spec` finds a tooltip or a visible label on every board element, card symbol and icon-only button at the draft, Age I, Age III and game over on both projects. A lint keeps the native `title` attribute out. The rules reference opens from the game menu as a drawer or a sheet, and at `/rules`. Animations, sound and haptics follow server events and the game's own Animation setting (Normal, Fast, Relaxed or Off, kept on each device). `animation.spec` checks that they are instant when off, that the final DOM is the same with animation on and off, and that input comes back within 450 ms. Its fourth check, no long task over 50 ms overlapping an animation at 4 times CPU throttling, fails: GitHub's runners fail it on every push, and on the development machine it passes or fails with the load. Three bounded rounds of fixes (`M10.animation-long-tasks`) found real costs, such as removing a card button that still had focus, which forced about 600 ms of style work per run, and repeated font-readiness reads. Their corrections removed those costs in native traces, but paired runs still showed overlapping tasks of up to about a second, mostly style, layout and rendering around the age change. The criterion moves to M11. Until it passes, push CI runs that check in its own job, `e2e-perf`, which writes its numbers to the run summary and does not fail the workflow, and a player on a slow device can turn Animation off. The two corrections merge as `M10.long-task-corrections`. The guided first game is a scripted room on a fixed seed (D25), ten guided turns against the Apprentice, which plays its normal moves once the player leaves the script. Its reviewer walked it from Home using only the screen at 360 by 740 (touch) and 1280 by 800 (Animation off). The e2e walk checks every coach-mark target at all ten viewports of the UX plan. The right-to-left spec passes with no horizontal scroll, mirrored chrome and a left-to-right board. The Persian locale stayed a stub through the build. The owner then asked for the whole locale, which merged as its own slice, `M10.persian-names`: all 1,491 messages, with a glossary and the questions left for a native speaker in [docs/i18n/persian.md](../i18n/persian.md). The rules audit ran two rounds of four lens-specific finders, 86 and then 69 black-box tests, and none failed. Hardening added the security pass (headers and a computed content security policy, a size cap on room files, fair bot slots per address), the offline app shell, the replay download and a docs audit. Nightly fuzzing and sweeps are green. Push CI was red for most of a day because a protocol test assumed that a resync frame's history always matches its ply. That is not true when the room merges queued frames, so the test now checks the history against a replay of the action log (`M10.ci-green`). The e2e suite outgrew one 30-minute CI job and now runs in two shards. Found along the way and moved to M11: the build-from-discard icon, the StarterDiagram ruling and a few cosmetic items (the victory ribbon, the deal origin, the MilitaryDiagram wrap, the coin float near the header). The screenshot matrix behind `make shots` became M11's first slice.

## M11: The review loop

Scope:

- One review loop over the finished game, run round after round. Each round has plain-script probes (horizontal scroll, tap-target sizes, axe, console errors, budgets), a "player" reviewer that plays real games in the browser at phone and desktop size, including the tutorial and the animations, and a "screens" reviewer that reads the screenshot set over fixtures and live rooms in light and dark themes (layout, looks, accessibility, colour vision). Fixers prove each fix with before and after shots, and the e2e suite catches regressions. There are no refuters or separate judge ([D22](decisions.md)).

Acceptance:

- The loop stops when a round finds only small issues that do not affect real play, with no open P0 or P1 finding.
- axe reports zero serious or critical violations in both themes.
- A colour-vision review of board fixtures passes.
- The initial JavaScript is below 200 KB gzipped, the board renders 2500 DOM nodes or fewer, and the home page's largest contentful paint is 2.5 s or less on a throttled mobile profile.
- Carried from M10: no task longer than 50 ms runs during an animation on a throttled CPU (`animation.spec` case d), on CI's runners as well as on the development machine.

Status: done on 2026-10-20, and with it the game reached v1. The loop ran three rounds. Rounds 1 and 2 each had the scripted probes, a player reviewer and a screens reviewer; round 3, sized down by the owner, had the probes and one player reviewer, followed by a targeted re-check of its findings in play instead of a fourth round. Every fix merged as its own slice with before and after shots, and its review was sized by risk ([D22](decisions.md) and the owner's review-by-risk rule of 2026-10-19). Between rounds the owner's own requests landed as slices too: the move review lists every decision of a turn and the advisor comments on it, the new age deals with its sound before the one dialog that holds the finished age's summary and who starts, the final move plays out on the board before the result opens and shows inside it, card tooltips in the log, and the crown and looting chips on the military track. Round 3's one P1 was advice that promised a "next turn" on the last cards of an age, when no next turn exists; hints now speak of a next turn only when the opponent's reply and the player's next pick both still fall in this age. The re-check, by a fresh player reviewer, confirmed that fix in the original game's position and found no P0 or P1, only one P2 (a tall move summary in phone landscape that still covers the cards to take when no lane has room) and five P3s, so the loop stopped. The round 3 probes reported no accessibility violations in either theme, no console errors and no horizontal scroll; 76 tap targets are between 24 and 44 px (P3). The colour-vision review of board fixtures passed (round 2). Budgets: 140 KB of initial JavaScript gzipped, 2,463 DOM nodes on the Age III board at 1280x800 (close to the 2,500 cap) and 1,892 at 390x844, and a home LCP of 1.55 s on a throttled phone. Case d, carried from M10, now passes on push CI: the `e2e-perf` job passed on both projects for the last two master pushes. Two slices got it there: the result dialog waits for every final animation (`M11.gameover-test-on-runners`), and an age opening waits for all of its effects and hands modal work to a later task (`M11.case-d-on-runners`). It is not yet fully reliable on the runners: a proof run with ten repeats per project passed 36 of 40 attempts on desktop and 29 of 40 on mobile, with tasks of 306 to 456 ms at points the traces could not attribute, so `e2e-perf` stays non-gating. A profiled CI run then attributed the remaining failures to native work on the runners (major GC, compositor commit waits and renderer frame work, with every application callback under 1 ms), so no client change followed ([master-e2e-failures.md](master-e2e-failures.md)). Left after v1: the P2 above and the re-check's P3s, a flaky Go race test seen once on CI (`TestReviewQuickInPlace`), card tooltips on the log's rows, Home's saved rooms that can still push content down once on a returning device, and the native Persian check and the iPhone Safari check, both deferred by the owner.

## M12: The design of Haftgane

Status: done on 2026-10-21. The owner chose Ajayeb's second game, **Haftgane**: the base game of 7 Wonders, second edition, for 3 to 7 players, on the same site and binary as Duel, with any mix of humans and bots at one table. M12 designed it and built nothing. The design is in [docs/haftgane/](../haftgane/README.md): the rules in our own words with 30 rulings, an audit of what Duel's code can be reused, the platform, engine and bot designs, the phone screens, and the art reuse plan. The owner answered its questions in two rounds on 2026-10-21, named the game and accepted the design. Its decisions are D30 to D60 ([decisions.md](decisions.md)), and its build plan is M13 to M23 below. Review: the platform and engine designs each had a FULL review by the other model family and a targeted recheck (two blockers and fourteen major findings between them, all fixed or answered), and the bot plan was revised twice against them; the screens and the art plan had no independent review, so their gates are in M18 to M20. Left open: the data checks of M13 and a few provisional calls, listed in the design's README.

Scope:

- A rules inventory from the publisher's sheets and rulebook, the fan wiki and several implementations, with the card data kept private until M13 verified it (the effect inventory and the edition differences were then published).
- A reuse audit of Duel's server, client and art, with file and line evidence.
- The platform and engine designs, each reviewed by the other model family, and the bot plan revised against them; the phone screens as rendered mockups, kept private because they show draft card values; the art reuse plan.
- One package that reconciles the documents, the owner's answers folded in, and the move of the decisions and this plan into `docs/dev/`.

Acceptance:

- No public text shares a run of seven or more words with a saved source, and no card value is published before M13 verifies it.
- Every design names its slices, review levels and acceptance checks, merged here into one plan ordered by dependency.
- The owner has answered every question that gates a build slice and accepted the design.

## Building Haftgane: M13 to M23

These milestones build Haftgane from the M12 design. They merge the slice plans of the platform, engine, bot, screen and art designs into one plan ordered by dependency. The reuse audit's own slice plan ("Question 2", with `M13.*` names) is superseded: the platform review replaced its "generalize Duel's room" approach with registered room kinds (D31).

**How to read the plan.**

- Each slice carries its review level from [orchestration.md](orchestration.md): **FULL** (stored games, the engine or rules, the protocol, deploy scripts, security, and anything written by a weaker model, reviewed by the other model family), **LIGHT** (ordinary UI, one cheaper reviewer) or **NONE** (tests, tooling, docs, chores; the coordinator reads the diff). A slice written by a weaker model is FULL whatever its label.
- "Needs" lists what must land first.
- **The internal milestone** is Haftgane playable end to end with Apprentice bots: rules, views, the room, simple bots, the client, art and the review loop (M13 to M20). Nothing is public yet. **The public release** waits for the sampled bots up to Scholar (M21), then capacity and the deploy (M22), as the owner chose. **Master** may follow later (M23).
- **Duel unchanged** means: Duel's full e2e suite and frame leak check, its protocol tests under `-race`, its golden replays and catalog hash, `engine.RulesVersion`, the schema-1 golden files, the recorded version-1 game frames and Duel's generated wire files are all unchanged and green.

**Standing gates for Haftgane,** on top of each milestone's own criteria and the standing gates above:

- Every slice that touches code Duel runs keeps Duel unchanged.
- Cold Home stays at or below 200,000 gzip bytes, and cold Home and cold Duel entry stay at or below their recorded baseline plus 5,000 gzip bytes, measured with service workers blocked.
- A card value is committed only as verified by M13's reconciliation, or marked provisional with its open question (now only Babylon's night side, Q1), never as a draft or mockup value; and no public text shares a run of seven or more words with a saved source.
- Haftgane rooms cannot be created in production until M22 turns creation on.

**Where work runs in parallel.** Four tracks can move at once:

- **Data:** M13, feeding `M15.haftgane-catalog-data`.
- **Shared layers and room:** M14, then the room actor of M16 (against a fake engine), then the client of M18.
- **Engine and bots:** M15 from its first slice (the core types and the payment solver need no data), then `M16.haftgane-engine-contract`, M17, and after `M15.haftgane-rules-gate` the sampler and Scholar work of M21.
- **Art:** M19 from M13's names on, one Codex generation at a time.

The tracks share one machine: the heavy guard queues builds and tests, and only one headless Chrome runs at a time, so parallel tracks save agent time, not machine time. Slices that edit the same seam (the four FULL slices of M14, or the engine's frozen files) run one at a time.

**Owner decision points.**

- Before M14: when shared-layer work starts. Letting real Duel games accumulate first was recommended; M13 and the first M15 slices touch no Duel code and can start at once. At the end of M14: the go for a Duel-only deploy.
- Before a level is enabled: six-bot waits that turn out unacceptable at Scholar's or Master's preset (M21, M23).
- Before M22: the go to deploy the public release.
- The provisional calls listed in the design's README stand unless the owner overturns them; they gate nothing.
- Expansions block nothing: the owner chose the base game only, and an expansion would add milestones after M23.

## M13: Rules and data for Haftgane

Status: done on 2026-10-21. Three blind tables (from the publisher's sheets, from the fan wiki and implementations, and from BoardGameGeek photos of the printed game) and the M12 drafts were reconciled field by field: all 80 card designs and 39 of the 41 wonder stages are confirmed, four draft costs changed (Town Hall, Arena, Circus, the Craftsmens Guild), and the printed arrows confirmed ruling 30. The rules docs in [docs/haftgane/](../haftgane/README.md) are final, with the data rulings, the effect inventory and the edition differences published, after a FULL review and one round of fixes. Q1, Babylon's night side (two stages), which no photo of the printed board shows, was accepted by the owner on 2026-10-22 as provisional, which satisfies the release gate; a clear photo would override it ([data-rulings.md](../haftgane/data-rulings.md#the-night-side-questions)).

Needs: nothing; the design was accepted on 2026-10-21.

Scope:

- `M13.blind-table-publisher` (NONE). A verifier builds a table of all 80 designs and 41 stages from the publisher's card and chain sheet, effects sheet and rulebook. It works blind: it never sees the private drafts, the two working notes then held back (the effect inventory and the edition differences, published since) or the other verifier.
- `M13.blind-table-wiki` (NONE). A second verifier, equally blind, builds the same table from the second-edition fan wiki's costs and boards, cross-checked against second-edition implementations.
- `M13.printed-card-check` (NONE). The four unconfirmed costs, the twelve wonder sides the rulebook does not draw and the direction of each trading post's arrow are read from the printed game; reading every cost is better still, because no publisher sheet prints costs and so the two tables are not independent there. No physical copy is needed. BoardGameGeek lists three fan reference files (wonder cards, an icon reference, a cost sheet) whose downloads need a signed-in browser, so the owner downloads them by hand into the private run directory, and the check reads them when they are present. Otherwise it reads user photos: BoardGameGeek's own pages refuse scripts, but its gallery API and image server answer (tested on 2026-10-21), so photos captioned as boards, card artwork and the like can be fetched. Files and photos are read for facts only (costs, wonder sides, arrows), kept private in the run directory, and never shipped or copied. The licensed online version stays a fallback.
- `M13.data-reconciliation` (NONE). A field-by-field reconciliation of both tables, the printed-card check and the drafts; independent refuters settle each disagreement, and anything left becomes an owner question with a provisional ruling. The reconciled reference stays in the private run directory until `M15.haftgane-catalog-data` encodes and commits it; the data rulings were published by `M13.rules-docs-final`.
- `M13.rules-docs-final` (FULL). The rules docs in their final place: rules.md with the owner's rules answers, sources.md, the effect inventory and the edition differences (published once reconciled), and the data rulings. Two reviewers: one for completeness, one for rulebook wording by seven-word matching.

Acceptance:

- The reconciliation reports zero unresolved fields, or each one is an owner question with a provisional ruling.
- The counts check out: 80 designs, 77 names and 148 copies; 7 cards per player in every Age at every player count from 3 to 7, with N + 2 guilds in Age III; 24 chain symbols, each provided only in an earlier Age than the card it frees; 7 boards, 14 sides and 41 stages.
- Every field the printed-card check was asked for has a recorded value and source (the fan file or photo it was read from, kept private), and each trading post's direction agrees with ruling 30, or the ruling changes.
- Two reviewers pass the rules docs, and `sources.md` lists every source with its licence.

## M14: Shared layers, with Duel unchanged

Status: done on 2026-10-22. Duel runs on the shared layers in production with only Duel registered: the room-file dispatch, the room-kind registry and the version-2 frames on the server, and a lazy client entry. After its FULL review measured what discovery before every entry cost Duel (the opponent saw a leave and a return, a bot move due on entry was lost, reloads were slower), the entry opens a Duel room on its own version-1 connection first and discovers another game only when the server refuses that join (D61). Duel's recorded game bytes, schema-1 golden files and generated wire files are unchanged. Cold Home fell from 141,410 to 109,351 gzip bytes; cold Duel rose by 3,499 to 269,883, within its budget, and two clean builds of one commit are now byte-identical. Slice 6 was not needed: the client entry split the card preview, and splitting the English game messages would save Duel nothing. The first deploy's production smoke found the test hook reporting ready before the lazily loaded Duel screen had mounted; with that fixed, `make e2e-prod` passed 5 of 5 against the main host, and both hosts serve that build. `make verify-quick` now also runs CI's `go mod tidy` check.

Needs: the owner's start signal. These slices change Duel's production code paths, so they run in this order, one at a time.

Scope:

1. `M14.platform-acceptance-baselines` (NONE). Record schema-1 golden files, version-1 game-frame fixtures, Duel's entry and import graph, cold Home and Duel bytes, the service worker's install transfer and the available mixed-load baselines. No behaviour changes; missing measurements are reported, not invented.
2. `M14.room-file-game-dispatch` (FULL). A bounded schema and game header decides the codec before decoding or replay; schema 1 goes byte for byte to today's path; the production schema-2 table stays empty; `CheckDataDir` dispatches through the existing counters, with the preflight script and report unchanged.
3. `M14.room-kind-registry` (FULL). Create, load and attach dispatch injected into the Manager, with Duel alone registered; the diff stays within the composition root and the Manager's seams.
4. `M14.protocol-game-frames` (FULL). Hello accepts versions 1 and 2; Duel keeps one game-frame contract; the Haftgane frame DTOs and strict decoders are added (not yet reachable); the Outbox gains a typed `table-state` branch beside Duel's.
5. `M14.client-game-entry` (FULL). Routing by the authoritative snapshot, the one-frame bootstrap behind the existing name prompt, the hand-off to Duel's unchanged Session, an empty lazy entry for Haftgane, and Saved Rooms' optional game tag.
6. `M14.client-root-import-boundaries` (LIGHT, optional). Break the root's eager Duel card-preview and game-message imports if slice 5 is too large for them.
7. `M14.duel-only-deploy` (NONE, the owner's go). Ship the shared layers with only Duel registered and run `make e2e-prod`, so real Duel games exercise the new seams before any Haftgane code relies on them.

Acceptance:

- D28 and D29 suites green; schema-1 golden writes unchanged; an unknown game is refused; an old binary loads, reaps and checks a data directory holding schema-2 fixtures without changing their bytes.
- Recorded version-1 game bytes unchanged; Duel plays under both hello versions; a version-1 client joining a Haftgane room gets the reload path; `make gen-check` green with Duel's generated files unchanged.
- The full Duel e2e and leak suite, generic-entry reconnect and two-tab tests, and old-tab Saved Rooms rewriting are green; a cold invite to an online Duel room with no stored name sends no hello or join and claims no seat until the name is submitted.
- Both cold-entry budgets hold, the precache is unchanged and its install transfer is recorded.
- After the deploy, `/health` answers and Duel's `@prod` specs pass.

## M15: The Haftgane rules engine

Status: done on 2026-10-23. All twelve slices landed; every FULL slice was reviewed by the other model family. Setup, turn resolution, effects and scoring, and the wonder abilities were built as one stack, reviewed slice by slice and merged together after one round of fixes: the setup review found that `CheckInvariants` expected one card in every hand during a sixth-turn discard build, where the rules leave none. Measured at the gate: coverage 91.9% for `rules` and for `catalog`; a sweep of 60,000 games (2000 per player count, side rule and policy) green with invariants after every point and every effect kind, free condition and ability path reached; `FuzzPlayout`, `FuzzKeys` and `FuzzPayment` ten minutes each without a crasher; the exhaustive payment oracle in agreement; golden replays for every board, side and ability path. The views slice's independent redaction check found no leak in 7,204 per-viewer frames over every seat count, both end policies, every seat and the spectator, and `History` equals the live filtered streams. Seven-seat `Legal` takes about 30 µs typically and about 2.5 ms on the adversarial fixture (target 10 ms); `Resolve` about 3 µs. Duel is unchanged: its goldens pass and a fixed-seed `ajayeb-sim match` writes byte-identical game logs. The rules audit (the gate's FULL review) decided each audited rule from the rules docs before reading the code and found no rule implemented wrongly; it found one clause of ruling 28 guarded only by the long sweep, now pinned by its own test. One target is accepted rather than met: `Quote` on a synthetic shared-capacity fixture takes about 48 µs against 5 µs, but the catalog cannot produce that shape, and 1.46 million quotes from natural games had a median of 0.45 µs and a maximum of about 5 µs; the solver gains a feasibility pre-check only if it shows up in M21's rollout profiles. Q1 stays owner-accepted provisional evidence.

Needs: M13's reconciled tables before slice 5. Slices 1, 4 and 6 move Duel code with no behaviour change, each just before its first caller. Slices 3 and 5 can run in parallel after slice 2.

Scope (the engine design's slices, in its numbering):

1. `M15.purity-guard-sharing` (NONE). Duel's import checker and fixed-value shape checker become test helpers configured per game.
2. `M15.haftgane-core-types` (FULL). Catalog types, the effect language with per-kind validation, the frozen state, record, action, event and point types, the seat ring and the key grammar.
3. `M15.haftgane-payment-solver` (FULL). The flat solver, its witness, `Quote`, an exhaustive oracle written without the solver's reasoning, and `FuzzPayment`.
4. `M15.catalog-diff-sharing` (NONE). The three-way diff's mechanics become a test helper with per-game schemas and join keys.
5. `M15.haftgane-catalog-data` (FULL). The verified data files, registry and compiled tables, the blind tables as the diff's fixtures, structural tests, regression pins for every first-edition value and known source error, the golden CSV and prose dump through `ajayeb-gen catalog -game haftgane`, and the list of fields still unconfirmed.
6. `M15.rng-sharing` (FULL). The generator, bounded draws, shuffle and digest helper move into `enginekit`, with Duel's names kept as wrappers.
7. `M15.haftgane-setup-and-deal` (FULL). Setup, the tagged streams, `SeatOrder`, decks at every N, boards, the three side rules and the side point, dealing, the scenario builder, invariants and the fingerprint encoding.
8. `M15.haftgane-turn-resolution` (FULL). Points, `LegalState` with per-action free checks before the quote cache, `ResolveState` in seven steps, passing, the sixth-turn leftover, military, Age transitions, the record and replay.
9. `M15.haftgane-effects-and-scoring` (FULL). Every coin and point kind, science with choice symbols, categories and ranks.
10. `M15.haftgane-wonder-abilities` (FULL). Babylon's last card, the discard build and its pile look (also when skipped), Olympia's free builds, the Decorators Guild on every side length.
11. `M15.haftgane-views-and-observation` (FULL). Knowledge, `ViewFor`, disabled reasons, the notebook with its statistics, both end policies, events, `FilterEvents`, `History`, the DTOs, the observation and its encoding, `Project` and `o.Quote`. It also carries the screen fields of the engine design's section 10: per neighbour, the units the viewer may buy, its price per unit and the discount's source, and the military outlook (D55); running scores, which the owner chose to show; and relative seat positions. The bot plan's first slice, "Observed drafting knowledge", is this slice.
12. `M15.haftgane-rules-gate` (FULL). Golden replays for every board, side and ability path, full sweeps, fuzzers, coverage, benchmarks, a complete `limits.go`, a rules-audit round, and `docs/architecture.md` and the rules docs updated.

Acceptance:

- A sweep of 2000 seeded games per player count, under random and simple policies and every side option, is green with invariants after every point; every effect kind, free condition, point kind and ability path is reached or forced by a scenario.
- `FuzzPlayout`, `FuzzKeys` and `FuzzPayment` run ten minutes each without a crasher; the payment oracle agrees exactly and no construction offers more than 15 outcomes; the science oracle agrees.
- Coverage of `rules` and `catalog` is at least 90%; the golden replays are stable.
- The three-way diff reports zero unresolved mismatches, and the unconfirmed list is empty or owner-accepted.
- The redaction and noninterference checks pass for every seat and the spectator under both end policies, with negative fixtures that prove they can fail; `History` equals the live filtered streams; no view, observation or download carries the seed.
- `limits.go` publishes the tested maxima, and the benchmarks are recorded against the engine design's targets.
- Duel unchanged, including `TestRNGGolden`, the setup and replay goldens and a byte-identical fixed-seed `ajayeb-sim match`.
- The rules-audit round finds nothing new, or its findings land first.

## M16: The Haftgane room and its protocol

Status: done on 2026-10-24. All four slices landed, each FULL slice reviewed by the other model family, with the shared lobby and the roles of D42 built as `M16.haftgane-lobby-and-roles`. The security pass confirmed and fixed four findings: an optional seat on pick, undo and resign; only a seat's own user links it; cached views dropped when the last connection leaves; and records bounded under the store's read cap, with no write passing it. It deferred two denial-of-service findings, which `M16.table-request-budget` then fixed: a per-user request budget at tables and lobbies that survives unloads and cache pressure and is checked before any engine work, a cap of eight connections per user with a bounded watcher count through every reseat, and bot searches counted under the player whose point it is. `M16.tableroom-race-budget` brought the tableroom race suite back into CI's go job. Left for later, as stated in [protocol.md](../protocol.md): per-room pseudonyms for user ids, requiring the seat on table requests, and promoted ids each getting their own budget.

Needs: M14 slices 2 to 4 for slices 1 and 2 (which run against a fake engine, beside M15); M15 complete for slice 3.

Scope:

1. `M16.platform-interleaving-harness` (NONE). An N-seat `ajayeb-drive` that submits from a common snapshot with randomized arrival, the fake engine contract, failure-injection fixtures, and an independent hand-game leak oracle whose negative fixtures plant a shuffle-derived id, a deal-ordered hand, a raw sold or tucked key, a pending or observation hash and an opponent's offer. Duel's two-seat driver is unchanged.
2. `M16.haftgane-room-actor` (FULL). The separate actor with the schema-2 codec; the shared lobby as its own `lobby` room kind (D60), whose Start hands a filled table to the actor; the roles of D42 (real admins with who promoted them, demotion, stepping down and the temporary admin after five minutes), persisted with each admin's disconnect time and shared by the lobby and the actor; changeable private pending picks with their confirmed flags and reload restore, point references, replay and undo (off once a second human sits down), claims, the admins' seat switches (any spectator into any seat, with the previous-seat hint for admins only), the public switch and role log, resignation and the one-successor rematch, whose requester creates it, behind a creation flag that is off. It also fixes the bot level used when an admin switches a seat to a bot, which no draft specifies (recommended: the admin picks it in the switch, defaulting to the lowest level released).
3. `M16.haftgane-engine-contract` (FULL). The real engine replaces the fake; offered-payment bounds, catalog ids and order, memory, setup, the side point, ability order, the end policy and replay compatibility are confirmed against the actor.
4. `M16.haftgane-security-review` (FULL). A security pass in M4's three lenses: hidden information across seven seats and a spectator (including what a pick change reveals), identity and authorization (claims, the admins' seat switches, promotion, demotion and the temporary admin, former controllers, rematch), and denial of service and limits. Confirmed findings are fixed, each with a test that fails first. No draft planned this; it follows M4 and M10.

Acceptance:

- Every arrival order on small points gives the same canonical batches and hashes; duplicate and evicted seqs, two tabs, stale submissions after undo, seat switches before and after a pick and its confirmation, and changes and withdrawn confirmations racing the last confirmation behave as designed. A reload restores a pick and its confirmed state, and no frame to another viewer changes when a seat edits its pick.
- Storage errors, a crash before and after rename, restore of every pending subset and the no-human pause are tested; sweeps run at 3 to 7 seats; actor operations and ability-point boundaries are fuzzed with replay invariants.
- Protocol tests play full tables at 3 to 7 seats with a spectator over the real WebSocket, room and store stack, and pass under `-race`.
- The leak oracle catches every planted leak and finds none in real frames; log-capture tests find no unrevealed key, hand, seed or observation hash.
- Roles, who promoted whom, temporary status and disconnect times survive a restart. A temporary admin is made only after every real admin has been away for five minutes, passes on after five minutes away, and is demoted the moment a real admin returns. An admin can demote only those they promoted and the creator anyone; nobody demotes the creator. The previous-seat hint reaches admins only, and every seat switch and role change is in every viewer's log.
- Duel unchanged; `make gen-check` green.

## M17: Simple bots (internal milestone)

Status: done on 2026-10-24. All four slices landed, and the FULL slice, `M17.fixed-preset-room-bots`, was approved by the other model family on its first round. The held-out release tournament played 500 deal blocks (30,600 games) and passed all 25 strength gates: against Random, Greedy's rank reward was 0.98 to 0.99 and its win credit 0.93 to 0.98, and Apprentice's 0.86 to 0.94 and 0.63 to 0.89; Greedy beat Apprentice in the paired comparison at every player count. The room bots' acceptance suite passes (repeatability with forbidden information changed, all 64 pending subsets across restarts, schedules, seat switches, undo and late results), and the six-bot wait is measured in [simple-bots-evidence.md](../haftgane/simple-bots-evidence.md). The review found that the room levels did not play the policies their persona names promise, so a follow-up, `M17.persona-level-mapping`, now maps easy (Apprentice) to the Apprentice policy, with normal and hard playing Greedy until the Scholar policy exists (M21). Frozen presets are an equality check against the build, so the next preset change archives every live table unless that slice adds a registry of supported preset sets; [room-bots.md](../haftgane/room-bots.md) describes both options.

Needs: M15 for slices 1 and 2 (beside M16); M16 slice 3 for slice 3.

Scope (the bot plan's slices 2, 3, 4 and the internal part of 9):

1. `M17.greedy-and-apprentice` (LIGHT). Random, deterministic Greedy, noisy Apprentice and the exact offered-key fallback. The seed mixer moves into `enginekit/rng` here, with the same unchanged-output proof as `M15.rng-sharing`.
2. `M17.seat-balanced-tournaments` (LIGHT). Game-selected tournaments in `ajayeb-sim` with deal blocks, seat and neighbour rotations, metrics and complete shard tallies; Duel's commands and reports unchanged.
3. `M17.fixed-preset-room-bots` (FULL). The actor calls the bot package directly through the existing permit pool, one job per room, with durable work rows.
4. `M17.simple-bots-evidence` (LIGHT). Haftgane's bot documentation: frozen versions, tournament results with intervals, time, memory and fallback figures, and guarded rerun commands.

Acceptance:

- Against Random, over at least 100 deal blocks per player count: Greedy's rank reward is at least 0.85 and its win credit at least 0.70; Apprentice's at least 0.75 and 0.50. Greedy beats Apprentice in their paired comparison.
- The same observation, entropy and work row give the same key and counters across restarts and schedules; changing forbidden information changes nothing.
- Zero to six starting bots, an all-bot continuation, mixed humans, every restore subset, seat switches with a full preset row and a kept pick, deterministic undo and harmless late results all pass.
- Cancellation leaks no permit or waiter; ordinary runs record zero watchdog fallbacks; the six-bot wait is measured and recorded.

## M18: The Haftgane client

Needs: M14 slice 5; M16 slice 2 for the first screens (fake engine) and for slice 13, and M16 slice 3 for real play. Procedural placeholders stand in for art until M19, as in M5. The platform design lists the client as one FULL slice; it is split here so that only the hidden-information boundary is FULL and the screens are LIGHT.

Scope (components as named in [phone-layout.md](../haftgane/phone-layout.md)):

1. `M18.table-session` (FULL). The lazy entry's Session and Presenter, typed `table-state` frames, `{seq, point, key, confirm}` submissions, changeable picks with withdrawable confirmation, the refusal after the last confirmation, reconnect and reload restoring the pick, and former-controller tabs.
2. `M18.home-picker-and-lobby` (LIGHT). The game picker, New table, the seat list with bots and invites, the admin menu in the lobby with its role actions and role tags, the spectator list, the settings (seating, sides, end reveal, undo), Start, the rematch lobby and spectating.
3. `M18.table-screen` (LIGHT). The seat ring, far chips, neighbour panels with trade chips and the military outlook, the pass cues, your strip with stage pills, the hand grid, the turn bar and status badges, in portrait, landscape and desktop.
4. `M18.card-sheet-and-payments` (LIGHT). The card sheet with the payment options, the stage payment list, the sale, confirmation and Change, and the chain, free and cannot-build variants.
5. `M18.reveal-and-recap` (LIGHT). The grouped reveal, coin changes, the turn recap and the hand pass, with Animation off and the live region.
6. `M18.special-decisions` (LIGHT). The side picker, the discard pick, Babylon's last card and Olympia's free builds.
7. `M18.age-military-and-scores` (LIGHT). The end-of-Age conflicts, the score sheet with its tie-break sentence, the final board and the download under either end policy.
8. `M18.city-sheet-and-notebook` (LIGHT). The city sheet with ring tabs, and the Seen hands sheet with its statistics.
9. `M18.admin-controls-resign-and-spectator` (LIGHT). The admin's seat menu in a game (a seat to a bot or to a human, seating any spectator in any seat with the previous-seat hint), the role actions (promote, demote, step down) and the role tags with the temporary-admin badge, resignation, undo disappearing when a second human sits down, the returning former player and the spectator view, which phone-layout.md describes but does not draw.
10. `M18.sound-and-motion` (LIGHT). Cue planning for this game's events (a decision opening, the reveal, conflicts, the end) within the cue budget, and the animations under D21.
11. `M18.rules-reference` (LIGHT). Haftgane's rules reference, from Home and the game menu, as Duel has. No draft planned it.
12. `M18.persian-messages` (LIGHT). Persian for every new message, with the native check deferred as it is for Duel. No draft planned it.
13. `M18.shared-lobby` (FULL). Duel's side of the shared lobby (D60): Duel's create path from a lobby started with two seats, carrying the lobby's admins so that any of them may edit the game's hint settings (D23), the two-seat settings panel, and the retirement of Duel's online sheet and its start-when-the-friend-joins flow, with Duel's online e2e updated. The lobby room kind itself comes with `M16.haftgane-room-actor`.
14. `M18.table-playthroughs` (NONE). The e2e playthroughs below and a layout spec.

Acceptance:

- Playthroughs green twice in a row on the desktop and mobile projects: the creator alone with six bots; two humans in isolated contexts with bots, a spectator, a reload that restores a confirmed pick, a pick changed before the last confirmation, an offline blip, an admin switch in each direction, a spectator seated in a human's seat, a promotion, a demotion, a step-down, a temporary admin made and demoted, and a resignation; an undo in a solo game; every special decision.
- The leak oracle checks every frame for seven seats and a spectator under both end policies.
- At 3 to 7 seats and at 360x740, 390x844, 844x390 and 1280x800, in both themes and right to left: no horizontal scroll, every control at least 44 px, no console errors or failed requests.
- Tooltip coverage, the i18n lint and the logical-CSS lint are clean; with Animation off every change is instant and visible.
- Both cold-entry budgets hold; a deploy in mid-game keeps the open page's lazy imports working under the all-build precache; Duel unchanged, except that its online start goes through the shared lobby; a two-seat Start gives the same Duel room file and frames as today's online game, with no admin controls once it starts.

## M19: Art for Haftgane

Needs: M13's verified names for the bitmap subjects; nothing else. Runs beside M15 to M18. Bitmaps are generated one at a time under `heavy.py --kind codex`; vectors are drawn by Opus agents in the render and critique loop (D18).

Scope (counts from [art-reuse.md](../haftgane/art-reuse.md), plus the crests of phone-layout.md):

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

## M20: The review loop for Haftgane

Needs: M17, M18 and M19. It follows D22 and M11.

Scope:

- `M20.shots-for-the-table` (NONE). `make shots` captures Haftgane's screenshot set over fixtures and live tables at 3 to 7 seats, light and dark, and right to left.
- Rounds of the loop: scripted probes (horizontal scroll, tap targets, axe, console errors, budgets), a player reviewer who plays real tables at phone and desktop size (alone with six bots, and a mixed table), and a screens reviewer who reads the screenshot set. Each fix is its own slice with before and after shots, reviewed by risk.

Acceptance:

- The loop stops when a round finds only small issues that do not affect real play, with no open P0 or P1.
- axe reports zero serious or critical violations in both themes; a colour-vision review of table fixtures passes.
- Cold Home and Duel budgets hold; Haftgane's cold entry and the seven-seat table's DOM node count meet budgets set in M18 (proposed: Duel's 2,500 nodes); Home's largest contentful paint stays at or below 2.5 s on a throttled phone.

With M20 the internal milestone is complete: Haftgane plays end to end with Apprentice bots, locally and in the review loop, and nothing is public yet.

## M21: Sampled bots up to Scholar

Needs: `M15.haftgane-rules-gate` and M17's tooling. It can start beside M16 to M20 and must finish before the public release.

Scope:

1. `M21.haftgane-sampler` (FULL). `Determinize`, `World`, `MatchesObservation`, the constraint model and `MaxSamplerChecks` (the engine design's slice 13).
2. `M21.sampler-integration` (FULL). The bot consumes `Determinize(o, rng) (World, bool)` and `w.State()`, with the no-world Greedy fallback.
3. `M21.simultaneous-age-rollouts` (FULL). Paired root sweeps with observation-limited policies for every simulated seat.
4. `M21.scholar-calibration` (FULL). Fixed iteration constants from full-table measurements, at equal work per bot, tuning on development and validation seeds, and held-out gates.
5. `M21.scholar-evidence` (LIGHT). Frozen versions, tournaments, intervals and diagnostics published.

Acceptance:

- `MatchesObservation` holds for every sample of the sweeps at every player count; on enumerable scenarios the sampler's possible worlds equal those an independent replay accepts, including skipped discard opportunities; every certain notebook statistic holds in every such world; the distribution is measured and published; no sweep position exhausts the bound.
- Scholar against Greedy: rank reward at least 0.65 and win credit at least 1/N + 0.10 over at least 100 blocks per player count, with the paired-improvement interval above zero. No player count regresses in the inverse and mixed fields.
- Six-bot waits at Scholar's preset are measured and, if unacceptable, taken to the owner before the level is enabled; p99 cancellation is below 50 ms under a stated load; zero ordinary watchdog fallbacks.

## M22: Capacity and the public release

Needs: M20, M21 and the owner's go to deploy.

Scope:

1. `M22.haftgane-capacity-release` (FULL). Host measurements of seven seats with six Scholar bots, an all-bot continuation, ability points, restore, the end reveal, a spectator burst and slow clients, alone and mixed with Duel load (p50, p95 and p99 against the Duel-only baseline); the bot-room-work limit chosen from them; candidate and rollback scans with schema-2 files; retention and backups; creation turned on by default.
2. `M22.public-release-deploy` (NONE; FULL if `self_host.py` changes). Ship with `./self_host.py ship`, add `@prod` specs for Haftgane, and run them.

Acceptance:

- Zero watchdog fallbacks in the ordinary workload; the chosen limit and its measurements are recorded; serialization and memory stay within bounds derived from the published maxima.
- The installer refuses a rollback while schema-2 live files exist.
- Against production: `/health` answers; a creator alone with Scholar bots plays several turns; two browsers at one table with bots play several turns, one of them changing a confirmed pick; Duel's `@prod` specs pass.

## M23: Master (may follow later)

Needs: M22. Not part of the public release.

Scope:

1. `M23.master-full-game-rollouts` (FULL). Full-game rollouts under Master's fixed preset; if it fails its gate, Master stays unavailable.
2. `M23.master-evidence` (LIGHT). Frozen versions, tournaments, intervals and diagnostics published.
3. `M23.master-deploy` (NONE; FULL if `self_host.py` changes). Capacity re-measured with Master tables, then shipped.

Acceptance:

- Master against Scholar: rank reward at least 0.58 and win credit at least 1/N + 0.05 over at least 200 blocks per player count, interval above zero.
- Six-bot waits at Master's preset are measured and, if unacceptable, taken to the owner first; after the deploy, the `@prod` specs pass with a Master at the table.

## After M23, not scheduled

- The private post-round advisor and any hints (bot plan slice 10, D50).
- Pass-the-phone play, deferred by the owner.
- Expansions, if the owner asks for one.
- A guided first game for Haftgane.
- A host-chosen board deal, which the rulebook allows and the engine's separate board stream makes cheap.
- Crests chosen in the lobby, if the owner ever wants them.
- Selective precache, only if install-transfer measurements show a material cost.
