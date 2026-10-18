# Decisions

A short record of the decisions that shape Ajayeb. Each entry gives the decision, the reasons for it, and the options we rejected. Settled decisions are not reopened casually: change one only with a new entry that says what changed and why, and mark the old entry as superseded.

All entries below were accepted on 2026-10-01 unless they say otherwise.

## D01. The Go server holds the only rules engine; the client is thin

**Decision.** The rules engine is written once, in Go, and runs only on the server. The SvelteKit client runs no rules code: it renders the view the server sends and sends back action keys. Every mode, including games against a bot and hot-seat games on one device, is a server room over WebSocket.

**Rationale.**

- One engine means one source of truth, so there is nothing to keep in sync and nothing to drift.
- Hidden information is enforced by the server. A browser never receives the identity of a face-down card, so no client can cheat, and redaction is tested in one place.
- The AI is faster in Go: a fixed-size state that clones with a plain copy, no garbage on the hot paths, and goroutines for parallel search.
- The result is one small binary that also serves the web app, with a small memory footprint.

**Rejected.**

- A TypeScript engine shared by the browser and a Node server. Hidden information would still have to stay on the server, so the browser copy would add a second place for legality and cost bugs, a Node server costs more memory, and search in JavaScript is slower.
- A Go server running an embedded JavaScript engine. Two runtimes, slow interpretation and painful debugging, for no gain.
- Two engines (Go on the server, TypeScript in the browser for instant feedback). Every rule would be written and tested twice, and the two would drift.

## D02. Moves are action keys, validated by set membership

**Decision.** A move is a canonical key string such as `construct:baths` or `wonder:pyramids:baths`. The engine produces the legal keys for the current state; the server accepts a key only if it is in that set, and then applies the engine's own action for it.

**Rationale.** The server never builds an action from client data, so a malformed or malicious message can only be "not in the set". Keys make logs, replays and test scripts readable, and the client can drive the UI purely from the listed `choices`. Payment is never part of a move: the engine always applies the cheapest payment, which is never worse for the player, so there is nothing for the client to choose or forge.

**Rejected.** Structured action objects validated field by field (a larger attack surface, and duplicated validation logic). Letting the client choose how to pay (no benefit to the player, and more to validate).

## D03. Every update is a full redacted snapshot, not a diff

**Decision.** After each accepted action, the server sends each connection the complete view for that viewer. Animations are driven by a separate list of structured `events` in the same message.

**Rationale.** The view is small (a few kilobytes), so diffs save little. A full snapshot makes reconnect the same code path as a normal update, rules out "missed patch" bugs, and lets a slow connection skip straight to the newest state (the outbox keeps only the latest). The client stays a pure function of its last snapshot.

**Rejected.** JSON patches or custom diffs (ordering and resync complexity, and a corrupt client state is possible). An event-sourced client that rebuilds state from events (it would need rules code).

## D04. WebSocket library: coder/websocket

**Decision.** Use `github.com/coder/websocket`.

**Rationale.** It has a context-aware API, writes are safe from several goroutines, and it has built-in ping, read limits and origin patterns. It is actively maintained.

**Rejected.** `gorilla/websocket`: it needs a single writer goroutine per connection and manual ping and pong handling. It remains a fallback if the module cannot be fetched, but the two must never be mixed in one codebase.

## D05. The web app is a SvelteKit static SPA

**Decision.** SvelteKit with `adapter-static`, `ssr = false` and an `index.html` fallback. The Go binary serves the build (embedded in production) with a SPA fallback and long-lived caching for immutable assets.

**Rationale.** All game state arrives over the WebSocket, so server rendering would add nothing. A static build needs no Node process in production, which keeps deployment to one binary behind Caddy and saves memory.

**Rejected.** SvelteKit with a Node adapter (an extra process and runtime for no benefit). A single SvelteKit app that also holds the game logic (ruled out by D01).

## D06. Browser testing: Playwright with the system Chrome, plus dependency-free CDP tools

**Decision.** End-to-end playthroughs use `@playwright/test` with the installed Google Chrome (`channel: 'chrome'`, headless); CI uses Playwright's bundled Chromium. (Superseded in part by D26: both now use Playwright's Chromium.) Screenshots, probes, screenshot matrices and SVG renders use small dependency-free Node scripts in `tools/chrome/` that launch their own headless Chrome with a throwaway profile over the DevTools protocol. Only one Chrome runs at a time, and it is closed after every run.

**Rationale.** Playwright is the reliable way to get isolated browser contexts (two online players in one browser), automatic waiting and WebSocket frame capture. Using the installed Chrome avoids a browser download. The CDP scripts cover the cases where Playwright is more than we need, at a fraction of the memory.

**Rejected.**

- Driving the owner's running Brave browser. It would share the owner's profile and storage, steal focus, offer no viewport emulation, and its scripting bridge is flaky. Saving one browser's memory is not worth it.
- Downloading Playwright's browsers on the laptop (a large download we do not need locally).
- A DevTools MCP server as the main screenshot path (it proved flaky in a sibling project).

## D07. Shipped art is AVIF; approved originals are committed

**Decision.** Bitmap art ships as AVIF only, under `web/static/art/`. The approved originals are committed under `art/originals/` as lossless WebP, pixel-identical to the generated PNG (checked by script when written), and the shipped files are derived from them by script. Rejected attempts stay outside the repository.

**Rationale.** The owner asked for both. For painted art, AVIF gives smaller files than WebP or JPEG at the same visual quality, and every modern browser supports it. Keeping the originals means any size or quality can be regenerated later without generating the art again, and nothing approved can be lost. Keeping rejects out bounds the repository's size. Codex returns PNGs of about 3.7 MB, so 109 PNG originals would add about 380 MB to a public repository whose history keeps every version; lossless WebP keeps the same pixels about 30% smaller (default chosen on 2026-10-10 while the owner's answer was pending).

**Rejected.** WebP as the shipped format (larger files; an earlier draft of the UX plan used it). Shipping PNG or JPEG (far too large). A WebP fallback next to every AVIF (double the files for browsers we do not target). Keeping the originals outside the repository (not reproducible, easy to lose). PNG originals (about 30% larger for the same pixels). Git LFS (GitHub's free storage and bandwidth would run out with public clones).

## D08. Centred pawn at the end of an age: the last active player chooses

**Decision.** When an age ends with the conflict pawn in the centre, the player who took the last card of the age chooses who starts the next age. When the pawn is off centre, the player on whose side it sits (the one with the weaker military) chooses, as the rules say. The next age is dealt before the choice, so the chooser sees its face-up cards. An engine option provides the literal reading, in which that player simply starts.

**Rationale.** The rulebook's wording ("begins") is ambiguous, fan summaries are split, and the licensed online implementation lets the last active player choose. Following it matches what most online players expect. The ruling is recorded in our own words in `docs/rules/decisions.md`.

**Rejected.** Making the last active player start, with no choice (kept only as an option). Leaving it to chance.

## D09. The coin supply is unlimited

**Decision.** There is no cap on the coins in the bank. A player's coins never go below zero.

**Rationale.** The rulebook does not limit the bank, and the licensed online implementation does not either. A cap would add edge cases (what happens when the bank is empty?) that the rules never define.

**Rejected.** Capping the bank at the number of coins in the physical box.

## D10. English first, ready for other languages

**Decision.** The UI ships in English first, but is built for translation from day one: every user-visible string goes through `t()` from typed message tables split by area, the server sends ids and codes instead of display strings, CSS uses logical properties only, user-provided text gets `dir="auto"`, and numbers and plurals use `Intl`. A Persian locale with right-to-left layout is the planned second language.

**Rationale.** The owner asked for English now and other languages later. Retrofitting translation and right-to-left layout into a finished UI is far more work than building them in. Typed per-area tables give compile-time checks that a locale is complete, and many small files avoid merge conflicts between parallel agents.

**Rejected.** A single message file per locale, as Paraglide uses (a merge-conflict hotspot when several agents add strings at once). Hard-coded strings with extraction later.

## D11. All randomness at setup; a game is its seed plus its keys

**Decision.** `NewGame(seed, options)` resolves every random element once. After setup the engine is deterministic. A room is stored as its seed, options and action keys, and restored by replay, with a hash check.

**Rationale.** Replays, golden tests, bug reports, undo and restart persistence all reduce to "replay the keys". Bots can determinize hidden information by reshuffling only the hidden fields. The stored form is tiny, and a rules change that alters an old game is detected by its hash instead of silently changing the game.

**Rejected.** Drawing random numbers during play (replays become fragile and determinization harder). Storing full state snapshots (larger, and they hide rule changes).

## D12. Guild cards have their own public back

**Decision.** Face-down cards expose only their back: `age1`, `age2`, `age3`, or `guild` for the guilds in Age III.

**Rationale.** The licensed online implementation draws face-down guilds with a distinct back, and the physical guild cards are reported to have their own back too (still to be confirmed against a physical copy). So seeing that a face-down card is a guild is public information. The back is a single catalog constant, so redaction and determinization follow from it automatically, and it can be changed in one place if a physical copy says otherwise.

**Rejected.** Hiding guilds behind the Age III back (it would give players less information than the physical game does).

## D13. Persistence is one JSON file per room

**Decision.** Each room is a JSON file under the data directory, written atomically after every accepted action. A reaper deletes finished rooms after 7 days and idle rooms after 14.

**Rationale.** The data is small and per room. Files need no database process, are easy to inspect and attach to bug reports, and survive a restart.

**Rejected.** SQLite or a database server (more moving parts for data this small).

## D14. Identity is an HMAC of a browser token; there are no accounts

**Decision.** The browser stores a random token in `localStorage`. The server derives the user id as an HMAC of that token with a server secret. Tokens never appear in URLs or logs, and an invite link is just the room code.

**Rationale.** Friends can play with no sign-up, while seats stay bound to the browser that took them. The pattern has worked in sibling projects.

**Rejected.** Accounts or OAuth (friction and personal data we do not need). Seats bound to the connection only (a reload would lose the seat).

## D15. Undo exists only in bot and hot-seat rooms

**Decision.** Bot and hot-seat rooms commit moves on the first tap and offer undo, implemented on the server by replaying to before the human's last decision. Online rooms have no undo; they ask for a second tap to confirm instead.

**Rationale.** In casual local play, undo is friendlier than confirmation. Online, undo would let a player take a card, see which face-down cards it uncovers, and then take the move back: a cheap way to peek at hidden information. That is acceptable against a bot or a friend on the same couch, but not between two remote players. Because hidden cards are fixed at setup, an undo can never reroll them.

**Rejected.** Undo by agreement online (more protocol for little benefit). Confirmation dialogs everywhere (slow for local play).

## D16. Wire types are generated from Go and checked against recorded messages

**Decision.** `tygo` generates the TypeScript types from the Go message and view structs. A Go test records real messages from a scripted game, and `svelte-check` type-checks them against the generated types. `make gen-check` fails if anything generated is stale.

**Rationale.** The types are written once. Recording real JSON catches what type generation alone misses, such as a nil slice that marshals as `null`.

**Rejected.** Hand-written TypeScript types (they drift). Protocol Buffers or JSON Schema (more tooling than this protocol needs).

## D17. Every heavy job goes through one guard

**Decision.** `tools/heavy.py` gates every build, test, lint, install, browser and image-generation job by memory pressure and by run mode, with machine-wide locks, and every `Makefile` target uses it.

**Rationale.** The development machine is often short of memory, and several agents may start heavy jobs at once. Scheduled prompts can fire late, so the guard, not the schedule, is what actually keeps heavy work inside the allowed hours. See [ram-policy.md](ram-policy.md).

**Rejected.** Relying on agents to remember the rules (they do not reliably). Relying on cron alone for the nightshift window (late fires would run heavy work in the owner's daytime).

## D18. Vectors by Opus render loops, bitmaps by Codex

**Decision.** Icons, symbols, frames, badges, the military track and the logo are hand-authored SVG, made by Opus subagents that render, look at and critique their own output. Painted art is generated by Codex `image_gen`, one asset at a time.

**Rationale.** The owner asked for this split. Hand-authored SVG gives small, crisp, themeable icons whose shapes can be tested for legibility at 12 to 16 pixels; image generation is good at painted illustrations and poor at precise, consistent symbols.

**Rejected.** Using image generation for icons (inconsistent shapes, raster output). Using a third-party icon set for game symbols (they would not match the art direction; generic interface icons come from Lucide).

## D19. The e2e prerequisite check runs inside the webServer command

**Decision.** `e2e/fixtures/global-setup.ts` checks that `server/bin/ajayeb` and `web/build/index.html` exist and empties the e2e data directory. It runs as the first step of Playwright's `webServer` command (`node fixtures/global-setup.ts && exec ../server/bin/ajayeb ...`), not as the `globalSetup` option, and it builds nothing. `make e2e` builds both pieces before Playwright starts, in a `build` slot of the guard.

**Rationale.** Playwright 1.63 starts the `webServer` before it runs `globalSetup`, so a check there would come after the server had already failed with an unclear message, or after it had opened the data directory. Building outside the browser lock keeps compiling from holding the only browser slot, and a raw `playwright test` gets a clear "build first" message instead of a slow surprise build.

**Rejected.** A `globalSetup` that builds when stale (it runs too late, and it would compile inside the browser lock). Reusing an already running server (`reuseExistingServer` is off, so each run starts from an empty data directory).

## D20. Category glyphs never reuse an effect icon's subject

**Decision.** The seven card-category glyphs are a mountain (brown), an anvil (grey), a dome (blue), an eight-pointed star (green), a tied purse (yellow), two crossed spears (red) and a crown (purple). The UX design's amphora, coin and shield are not used for grey, yellow and red.

**Rationale.** A category glyph sits on the same cards as the effect icons. An amphora would read as the amphora chain symbol, a coin as a coin effect and a shield as a military shield, so a player could mistake the card's colour for one of its effects. The anvil is a workshop, which is what grey cards stand for, and it shares no outline with the glass flask or the amphora. See [art/vector/specs/ui-category.md](../../art/vector/specs/ui-category.md).

**Rejected.** The amphora, coin and shield from the UX design (each collides with an icon of the same family). A neckless jar for grey (still close to the amphora and the barrel at 15 px).

## D21. Animation follows the game's own setting, not the OS

**Decision (2026-10-11).** The owner decided that animation is controlled only by the game's own settings, persisted locally, never by the operating system's reduced-motion preference. Settings > Animation speed (normal, fast, relaxed, off) lives under the `ajayeb:settings` key in `localStorage`, defaults to normal, and is the only switch: the client no longer reads `prefers-reduced-motion` anywhere, in CSS or in script. With Animation off, every animation is still instant and every state change still stays visible, exactly as before; only what triggers that behaviour changed.

**Rationale.** A player's choice inside the game is explicit and easy to find, while the OS preference is often set once, for every app on the device, for a reason that may not apply to a board game the player actually wants to watch move. Making Settings the single source of truth also keeps the app's behaviour predictable from one browser profile to the next, instead of depending on a system setting the player may not remember changing.

**Tradeoff.** A motion-sensitive player who has turned on the OS's reduced-motion preference will still see full animation on their first visit, until they turn Animation off in the game's own Settings. This makes it important that Settings stays reachable from the game's menu on every screen, not just from Home.

**Rejected.** Keeping the `prefers-reduced-motion` override on top of the setting, as before (the OS could force Off but never un-force it): the owner wants the in-game setting to be the only switch, not a second, invisible one layered under it.

e2e no longer sets Playwright's `reducedMotion` context option, which stopped doing anything useful once the client stopped reading it. Instead, a shared context init script (`e2e/fixtures/base.ts`) merges `animationSpeed: 'off'` into the `ajayeb:settings` JSON before the app's first load, so the suite keeps its fast, deterministic pacing through the real game setting.

## D22. Build the game-feel features first, then one review loop

**Decision (2026-10-11).** The owner merged the separate review passes of M10 (UX polish) and M11 (game feel, tutorial and hardening) into one stage. First the features are built as ordinary slices: the queued UX fixes, then animations, sound, haptics, the speed setting, the guided first game, tips, the rules reference and the right-to-left Persian stub. Then a single review loop runs over the finished game, round after round, and stops when a round finds only small issues that do not affect real play. Each round has plain-script probes (horizontal scroll, tap-target sizes, axe, console errors, budgets), a "player" reviewer that plays real games in the browser at phone and desktop size (including the tutorial and the animations), and a "screens" reviewer that reads the screenshot set (layout, looks, accessibility); fixers prove each fix with before and after shots, and the e2e suite catches regressions. The deeper rules audit and the hardening work (fuzzing at scale, the security pass, replay download) run beside the UI build, outside the loop, because they do not touch the screens.

**Rationale.** Reviewing the screens before the animations and the tutorial exist would mean reviewing them again afterwards, so one loop at the end is cheaper. Two reviewers per round instead of a wide panel: only one headless Chrome runs on the machine at a time, so browser reviewers could not run in parallel anyway, and splitting the screenshot set from the played games keeps each reviewer's context small enough to look closely.

**Rejected.** Separate M10 and M11 loops (double review of the same screens). One reviewer for everything (a full screenshot set plus a played game does not fit one context, and a single "find everything" reviewer stops after its first few findings). Refuters and a separate judge per finding (their cost buys little once fixers verify with screenshots and e2e).

## D23. Hint settings belong to each game, edited by its admin

**Decision (2026-10-11).** The owner decided that hint settings are part of each game, not a device preference. Every game carries its own hint settings (one switch per hint family, all on by default), and the game's **admin** may change them at any time, including mid-game. The admin is the player who created the game; other ways to promote admins come later. The client also saves the hint settings locally, and a newly created game starts from the last settings saved on that device. Hint settings live with the room, not in the engine's game options: they never change the rules, so a game is still fully described by its seed, options and action keys. The server stores them with the room, computes only the enabled hint families, and sends the same settings to both players; a change shows in the game log for both.

**Rationale.** Hints carry information that helps a player decide, so in an online game both players should play under the same hints, set by one person and visible to both, instead of each device choosing its own. Keeping them out of the engine options keeps replays and the store format unchanged. Computing only the enabled families makes the setting authoritative: a client cannot show a hint the admin turned off, because the server never sends it.

**Rejected.** Hints as a per-device setting in Settings, as the UX design first had it (players in one online game could play with different information). Hints as an engine option fixed at creation (it would tie a display concern to replays and forbid changing it mid-game).

**Amended (2026-10-11), the owner's answers on three hint questions.** Hints may name the five progress tokens left in the box, since a careful player can work them out by elimination; only their order stays secret. There are no move suggestions. The advisor family is a post-move review: after a player has moved, the summary of the opponent's next move tells that player, and only that player, whether they could have made a better move and why, so players learn from it. It is offered in every mode, on by default in games against the bot and off by default in hot-seat and online games, where the admin may turn it on. The warning before a move that hands over a win or misses one respects the Confirm moves setting: with confirmations off it appears only in the card sheet and the after-pick panel, never as an extra confirm.

## D24. Art is delivered whole, by content-hashed URL, with no total size cap

**Decision (2026-10-12).** The owner asked why the client did not prefetch more and dropped the 5 MB cap on the total shipped art ("considering not all cards are loaded at once, there is no need for the 5mb hard cap on all cards"). Slice AD1 made this the policy:

- Every art URL carries `?v=` and the first 10 hex digits of the file's SHA-256, written by `tools/art/index.mjs` into the generated index. The server caches an `/art/` request whose `v` matches the file's own hash for a year (`public, max-age=31536000, immutable`); a request without `v`, or with a stale one, gets a day plus `stale-while-revalidate` and a strong ETag.
- After the first board paint the client prefetches the whole game's art at idle, one file at a time at low priority, in the order it will be needed (the current age at the card sheet's size, then thumbnails, the next ages, guilds, wonders, tokens, backs, victory scenes), using the same `srcset` and `sizes` strings as the frames so the cache hits. It pauses while the page is hidden and keeps the old small plan under Save-Data or a 2g connection.
- Budgets are per image only. There is no total cap; `index.mjs` prints the shipped total and the prefetch bytes per device class instead (at the time of AD1: 220 files, 4.23 MB shipped; a full prefetch is about 2.6 MB on a phone at DPR 3 and 0.8 MB on a desktop at DPR 1).
- A higher-resolution variant exists only where a measured display needs it. Wonders gained a 1440x960 size; cards did not, because their largest display (the phone card sheet, 142 CSS px at DPR 3) needs 426 px and the 480 px file covers it.

**Rationale.** A URL that changes whenever the file changes cannot be served stale by any cache between the server and the browser, so a year-long cache is safe and repeat visits cost nothing. The total cap measured the wrong thing: no screen loads every file at once, and the per-image budgets already bound what one view costs. Prefetching at idle turns a slow first look at each card into an instant one, at the price of data the player was going to download anyway in a full game.

**Tradeoff.** A player who opens one game and leaves downloads more than before (up to the full plan). Save-Data and slow connections opt out.

**Rejected.** Keeping the 5 MB total (it forbade sharper wonders for no user-visible gain). Hashed filenames instead of `?v=` (the query form needs no renaming and leaves no old files behind; every cache on our path keys on the query string). Prefetching everything at once (it would compete with the board's own first requests).

This supersedes the "budget 5 MB" line in the M9 section of `docs/dev/roadmap.md`, which is corrected at the end of M10.

## D25. The tutorial is a scripted room on a fixed seed

**Decision (2026-10-17).** The guided first game (M10.tutorial-design, built as M10.tutorial-server and M10.tutorial-client) is an ordinary `ai` room against the Easy bot, created with `tutorial: true`, on the seed and options of a frozen script (`server/internal/tutorial`, script `first-game`). The room layer derives the current lesson from the script and the room's action log and sends it to the player as `State.tutorial`. The engine is unchanged. While the log follows the script, the bot plays the script's replies; once the player leaves it, the Easy bot plays. The client enables only the lesson's action key while the game is guided (the rails), with a "Play on my own" exit; the server still accepts any legal key, so the rails are a display rule.

**Rationale.** A fixed seed makes every lesson's precondition (a free build, a trade, a chain, a wonder, a science symbol) a Go test, and scripted replies keep the tutorial stable when the bots are tuned. Deriving the lesson from the log means undo, reload and replay need no extra state, and the engine stays pure.

**Tradeoff.** A new or shorter script needs a new name, a seed search (the harness runs with `AJAYEB_TUTORIAL_SEARCH=1`) and a new golden file. The lesson order follows what one seed can play: the chain lesson comes before science and the tokens lesson moved to the outro, because the design's order fits no seed's coin budget.

**Rejected.** The Easy bot's own moves on the fixed seed (any change to the AI would silently change the tutorial). Rails enforced by the server (more code for no gain in a single-player room). Soft guidance with every move enabled (a beginner gets lost, and the "fresh agent completes it" acceptance test becomes unreliable).

## D26. Agents' browsers are Playwright's Chromium, never the installed Chrome

**Decision (2026-10-18).** The e2e suite (locally and on CI) and the `tools/chrome` scripts run Playwright's own Chromium, Chrome for Testing (`channel: 'chromium'` in `e2e/playwright.config.ts`; `chromium.executablePath()` from the e2e package in `tools/chrome/launch.mjs`), in Chrome's new headless mode. `make install` downloads it with `playwright install chromium --no-shell`. This replaces D06's use of the installed Google Chrome.

**Rationale.** The owner saw their everyday Chrome being launched. No agent had opened its profile, but every headless run started `/Applications/Google Chrome.app`, so macOS registered each run as the Google Chrome app (bundle id `com.google.Chrome`) and it shared that app's crash-report folder. Chrome for Testing is a separate app (`com.google.chrome.for.testing`), so the owner's browser is never involved. Local runs now use the same browser family as CI.

**Tradeoff.** A download of about 150 MB per Playwright version, and the browser is not byte-identical to the Chrome the owner uses, so screenshots can differ slightly from it.

**Rejected.** chrome-headless-shell, the old headless mode (it renders less like real Chrome, and `tools/chrome` would need Playwright's internal registry to find it). Keeping the installed Chrome with more flags (it is the same app, so nothing can hide it).
