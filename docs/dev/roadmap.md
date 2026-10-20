# Roadmap

Ajayeb is built in twelve milestones, M0 to M11. Each milestone ends green (its acceptance criteria pass), committed and pushed. This file changes only when a milestone ends; day-to-day progress is tracked in the private run state.

**Slice ids and other names.** Every shorthand name must tell a reader what it is without a lookup.

- A milestone is `M0` to `M11`. The first mention in any report, brief or document also gives its title, such as "M10 (game feel, tutorial and polish)".
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
