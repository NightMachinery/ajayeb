# Decisions

A short record of the decisions that shape Ajayeb. Each entry gives the decision, the reasons for it, and the options we rejected. Settled decisions are not reopened casually: change one only with a new entry that says what changed and why, and mark the old entry as superseded.

All entries below were accepted on 2026-10-01 unless they say otherwise. D30 to D60 belong to Haftgane, Ajayeb's second game, and were accepted together on 2026-10-21 with its design ([milestone M12](roadmap.md), [haftgane/README.md](../haftgane/README.md)). Each says what it applies to and which earlier decision it extends or supersedes; Duel keeps D01 to D29 as written, except that D61 (2026-10-22, after the review of the client entry) changes how D51 enters a Duel room. A call marked provisional stands until the owner overturns it.

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

D28 clarifies the persistence hash rule: within the same explicit rules version, a complete legal replay is canonical and a differing stored hash is refreshed. Downloaded game logs still validate their final hash strictly. The historical rationale above records the original policy.

## D12. Guild cards have their own public back

**Decision.** Face-down cards expose only their back: `age1`, `age2`, `age3`, or `guild` for the guilds in Age III.

**Rationale.** The licensed online implementation draws face-down guilds with a distinct back, and the physical guild cards are reported to have their own back too (still to be confirmed against a physical copy). So seeing that a face-down card is a guild is public information. The back is a single catalog constant, so redaction and determinization follow from it automatically, and it can be changed in one place if a physical copy says otherwise.

**Rejected.** Hiding guilds behind the Age III back (it would give players less information than the physical game does).

## D13. Persistence is one JSON file per room

**Decision.** Each room is a JSON file under the data directory, written atomically after every accepted action. A reaper deletes finished rooms after 7 days and idle rooms after 14.

**Rationale.** The data is small and per room. Files need no database process, are easy to inspect and attach to bug reports, and survive a restart.

**Rejected.** SQLite or a database server (more moving parts for data this small).

D28 adds the room compatibility policy and deployment preflight to this file-based design; the retention periods above are unchanged by that decision.

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


## D27. Sound is on by default

**Decision (2026-10-18).** Sound defaults to on, with the existing Quiet, Medium and Loud volume steps. A fresh device hears cues only after its first audio-unlocking gesture. A stored `sound: false` stays off through the existing settings parser; there is no migration or extra choice marker. The board's top bar has a direct sound switch at every width, and full Settings remains in the game menu.

**Rationale.** Sound makes a wonder, a token, looting, an extra turn and the game's end easier to notice. The owner wants that feedback on the first game. Distinct cues mark these events, with a maximum of three cue kinds per move so routine payments and reveals do not drown out the larger moments. Online turn alerts require a live rival turn handing the board back to the viewer, and defeat is personal to a losing player. Shared results and spectators use a neutral chime; hot-seat celebrates the winner.

**Tradeoff.** The first gesture allows later game sounds without an opt-in step. A player who wants silence can mute directly on the board, and the choice persists. Browser autoplay restrictions still apply, so the page neither creates an AudioContext nor fetches sound files before a gesture.

**Rejected.** Keeping Sound off by default (players can miss the feedback entirely). Migrating stored false values or adding an explicit-choice marker (there are no existing players to migrate, and stored test-device preferences may remain as they are).


## D28. Saved rooms are canonical within an explicit rules version

**Decision (2026-10-19).** A saved room is its seed, options, optional scenario and canonical action keys within the same `engine.RulesVersion`. Restoration selects each ordinary key by membership in the engine's legal actions, then applies the engine's own action. It uses the room's existing exception for resignation: either engine-generated `ResignAction` is accepted only while legal, because bots must never find resignation in their action list.

A complete replay under the same rules version restores a playable room even when the stored state hash differs. The store atomically refreshes only that hash and logs a warning with the old and new values. It preserves `Updated`, identity, seat/link epochs, settings, action history and other metadata, including unknown additive fields. A failed refresh is a load failure. An explicit rules-version mismatch or an unreplayable key archives the room durably with `rules-version` or `replay-failed` and an archive timestamp. Archiving keeps the existing retention clock behavior: `Updated` becomes the archive time. Existing archives stay archived without a rewrite, including legacy archives without a reason. Unknown file schemas are refused and left byte-for-byte intact.

An incompatible change to setup, the RNG, catalog ordering or meaning, action semantics or rules must bump `engine.RulesVersion`, even when every old key remains legal. A layout change or a new derived state field alone does not require a bump. Replay downloads and engine log validation retain their strict final-hash check.

Before replacing a production binary, setup/redeploy run the candidate's read-only room check against the target data while the old server still runs. They refuse in-progress rooms that would archive, unless the operator explicitly passes `--force-room-archive`. Shipping forwards that one-shot override. Read, schema and operational errors always refuse, including with force. A second check after stopping and releasing the old port catches moves made during the first scan; a failed second check preserves the old binary and data and attempts to restart that binary on its prior configuration. Neither check initializes identity, creates directories, archives files or listens on a socket.

**Rationale.** Raw state hashes can change with representation and derived fields even when the saved decisions still describe a valid game. Keeping those games playable avoids ending live rooms for an internal representation change. An explicit semantic version is the compatibility boundary. A preflight exposes deliberate incompatibility before avoidable downtime, while the stopped check narrows the race with active players.

**Tradeoff.** Accepting a changed hash loses detection of some accidental corruption or tampering: a changed seed, options or sequence that still replays legally can pass. FNV hashes never authenticated the data or protected it against a writer who could replace the hash too. Strict membership validation catches malformed, noncanonical and illegal keys, but cannot prove that an old seed still means the same deal. Compatibility therefore depends on deliberate version discipline and backups. The running preflight is a changing-directory scan, not a transaction; the stopped check still assumes the service stop has quiesced writers and cannot protect against another process editing data afterward. Restart recovery can fail and is reported without claiming success.

**Rejected.** Archiving every hash difference (needlessly ends compatible games). Ignoring rules-version mismatches or replay errors (can silently change a game). Relaxing downloaded-log validation (would weaken its separate verification contract). Automatically rewriting newer schemas (loses fields on rollback). Silently accepting a failed check as an empty directory, or persisting the force flag (hides operational failures or turns future data loss into an unattended default).


## D29. Keep bounded diagnostic archives outside the live room directory

**Decision (2026-10-19).** Archived rooms move from `data/rooms/<code>.json` to `data/rooms-archive/<code>.json`, retaining their reason, archive timestamp and complete metadata. Persist the archive marker first, then link the file into the archive directory before unlinking the live path. Both directories must be on the same filesystem. The move never overwrites another record. A failed move leaves a closed record to retry; an interrupted move may leave two links to the same file, which the next load or sweep reconciles. Per-code store locks serialize moves, saves, restores and cleanup for that code. Different rooms remain independent.

The reaper keeps archives for 30 days after `ArchivedAt`, using `Updated` for legacy archives without that field. Migrating an existing archive changes no bytes and never resets its clock. Startup and hourly sweeps remove expired archives, then the oldest remaining archives until both limits hold: 1000 files and 64 MiB of JSON bytes. Archive time orders eviction, with code breaking ties. Finished rooms still expire after seven days without an update and unfinished rooms after fourteen days. These archive rules supersede the archive portion of D13 and refine D28's storage location and retention period.

Live listings and code scans exclude the archive folder. Direct code lookups still return the typed archived-room error, and new code allocation checks both folders. The read-only candidate check scans both directories; an archive record never contributes to live-room loss. Unknown schemas, malformed files and nonarchived records misplaced in the archive folder fail preflight and remain untouched by cleanup. Unrelated filenames and directories are ignored.

Deployment room backups remain snapshots of `rooms/`, with a separate policy: at most ten backups, none older than thirty days by filesystem modification time. Every backup attempt prunes existing copies, even when there are no live rooms to snapshot. It does not repeatedly copy the diagnostic archive directory. Operators who need a longer investigation window must copy the relevant diagnostic record elsewhere before expiry.

**Rationale.** A separate folder keeps live scans small while preserving the room code and failure reason for bug reports. Thirty days gives an investigation window; count and byte limits bound ordinary archive growth. Oldest-first eviction favors recent incidents. Keeping finished and idle retention unchanged preserves the existing game-download and abandoned-room behavior.

**Tradeoff.** Cleanup runs at startup and hourly, so managed archives can temporarily exceed the limits between sweeps. Foreign or unreadable records are preserved outside those limits rather than deleted using a schema this binary cannot interpret; they require operator repair and can consume extra space. A flood can shorten the diagnostic window below thirty days. Hard links require a shared filesystem. Backup expiry uses modification time, so manually copying or touching a backup can change its age. These are diagnostic retention and deployment recovery policies, not a permanent history or a full data backup.

**Rejected.** Keeping archives mixed with live rooms (live scans grow with failed games). Keeping every archive forever (unbounded storage). Reusing finished-room expiry (too little time for a delayed bug report). Retaining only by age (a burst can fill storage), or silently deleting unfamiliar records to meet the caps (unsafe on rollback).

## D30. Haftgane is 7 Wonders, second edition, on the same site in one kind of room

**Decision (2026-10-21).** Ajayeb's second game, **Haftgane** (Persian هفتگانه, "the seven"; next to Ajayeb, عجایب, the two words make the Persian name for the Seven Wonders of the World), is the base game of 7 Wonders, second edition (2020), for 3 to 7 players. Its game id and room kind are `haftgane`. The owner decided that it lives on the same site and in the same binary as Duel, with a game picker on Home, and that it has one kind of room: 3 to 7 seats, in which the creator may add any number of bots, so a table can be the creator alone with bots or any mix of humans and bots. Spectators work as in Duel. Pass-the-phone (hot-seat) play is deferred. The base game only, decided on 2026-10-21: no expansion is built, and the design keeps only cheap extension points.

**Rationale.** One site keeps one identity, one deployment and one set of room links. A single room kind, with bots added seat by seat, covers playing alone and playing with friends without a mode choice.

**Rejected.** A separate site or binary (two deployments to maintain for one owner). Separate bot and online modes, as Duel has (they contradict "any mix of humans and bots"). Hot-seat for 3 to 7 players in the first version (a pass-the-device flow for up to seven people, deferred by the owner).

**Relation.** Haftgane only. Extends D05 (one static SPA served by one binary, now for two games). D01 holds unchanged. D15's bot and hot-seat modes do not exist in this game; D43 replaces them here. Its online lobby also starts Duel at two seats (D60).

Source: the owner's answers of 2026-10-21; [platform-architecture.md](../haftgane/platform-architecture.md), summary.

## D31. Room kinds are registered side by side, and Duel's room stays as it is

**Decision (2026-10-21).** The server registers **room kinds**, each a complete game implementation: a file codec, a room constructor, an inbound frame decoder and a lazy client entry. Duel registers its existing actor, schema-1 codec and client unchanged. Haftgane gets its own actor (`server/internal/tableroom`) typed directly to its engine. The two share the Manager's lifetime bookkeeping, connection queues, claims, clocks, atomic file operations and the bot permit pool through narrow seams. Some orchestration is duplicated at first and is extracted only once both actors have tests that prove it identical.

**Rationale.** Duel is finished and verified against stored games. Generalizing its actor, Session, review, tutorial, hot-seat, undo and bot scheduling before a second caller exists would put all of it at risk for code that Haftgane does not need.

**Rejected.** A universal engine interface or factory with a capability matrix. Two transaction modes inside one actor. Rewriting Duel's orchestration in advance.

**Relation.** The server that hosts both games; Duel behaves exactly as before. New. It constrains how D13, D16 and D28 are implemented for two games (see D41) but changes none of them for Duel.

Source: [platform-architecture.md](../haftgane/platform-architecture.md), "Room registration and package boundaries".

## D32. Haftgane's rules live in their own pure packages; only infrastructure is shared

**Decision (2026-10-21).** The rules live in `server/internal/haftgane/` (`catalog`, `rules`, `view`, test helpers in `rulestest`, bots in `bot`), pure under AGENTS.md's engine rules, with one stricter rule: no file of the new engine imports `math/rand/v2`. Code that means exactly the same thing in both games moves out of Duel into `server/internal/enginekit/`, each piece just before its first new caller and with Duel's goldens, replays, stored rooms and rules version proven unchanged: the seeded generator and shuffle, the seed mixer, the digest helper, the import and fixed-value shape checkers (test helpers) and the three-way catalog diff (a test helper). The bot permit pool is shared by use, not moved. State, catalog types, effects, payment, military, science, scoring, views, replay helpers, test harnesses and bot search are written per game.

**Rationale.** The same words (production, chains, science, military) name different rules in the two games, and Duel's engine is tied to stored rooms through its hashes. Infrastructure with identical meaning should exist once, so that a fix reaches both games and copies cannot drift. This answers the owner's question about sharing code between the engines.

**Rejected.** Generalizing `server/internal/engine`. A shared rules core with per-game plug-ins. Copying the infrastructure. One package for catalog, rules and DTOs.

**Relation.** Haftgane only, apart from the infrastructure that moves out of Duel unchanged. Extends D01: Haftgane's rules also run only on the server. The new engine follows D11's determinism through D36.

Source: [engine-architecture.md](../haftgane/engine-architecture.md), sections 1 and 15.

## D33. Card designs are the rules identity; physical copies exist only while decks are built

**Decision (2026-10-21).** The catalog has one entry per **design** (a card as printed for one Age) and a separate name identity for the one-building-per-name rule. Hands, cities, tucked cards, the discard pile, keys, events, views and observations hold design ids. Physical copies are counted only when the decks for N players are built, so no id, order or key can encode a deal position. Every list of cards a viewer receives is sorted by design id.

**Rationale.** Copies of a design are indistinguishable in play. Copy ids would carry no information, would need a canonical-copy rule for keys, would enlarge what bots sample, and could leak the shuffle.

**Rejected.** Copy ids in the state or on the wire. One design per name (it would blur which Age a card belongs to).

**Relation.** Haftgane only. New.

Source: [engine-architecture.md](../haftgane/engine-architecture.md), section 2 and its draft entry; [platform-architecture.md](../haftgane/platform-architecture.md), "Views, events and history".

## D34. A construction key names its payment outcome

**Decision (2026-10-21).** A construction key says how many coins go to each neighbour: `build:<design>:<left>:<right>` or `stage:<design>:<left>:<right>`. The engine offers exactly the affordable outcomes that no other outcome beats (paying no more to either neighbour and less to one), at most 15 per construction, cheapest first, and a free construction only as free. Buying a unit one's own production covers is never offered. The server accepts only an offered key and applies the engine's own action.

**Rationale.** In Duel the cheapest payment is never worse, so D02 keeps payment out of moves. Here trade coins go to one of two rival neighbours and the rules let the buyer choose, so which rival gets paid is a real decision. Leaving out beaten outcomes is a deliberate restriction (rules.md ruling 10): it keeps the list short and its size proven.

**Rejected.** The engine choosing the cheapest split with a fixed tie-break (removes a real choice). A payment field validated by the server (D02's reasons). One key per way of covering a cost (identical effects). Every outcome, beaten ones included (up to 225 per construction).

**Relation.** Haftgane only. Supersedes D02's payment clause for Haftgane only. D02's canonical keys and set membership stay. The owner accepted unbeaten payments only on 2026-10-21.

Source: [engine-architecture.md](../haftgane/engine-architecture.md), sections 7 and 8; [rules.md](../haftgane/rules.md), ruling 10.

## D35. A simultaneous decision resolves as one complete batch from a frozen position

**Decision (2026-10-21).** The engine defines **decision points**: a set of required seats choosing against one frozen position. Each turn is a point for all seats, checked and priced against the position at its opening and applied in seven fixed steps (choose, check, pay, place, when-placed effects, end-of-turn abilities, pass). The Halikarnassos discard build and Babylon's last card follow as one-seat points, in that order. The room holds one pick per required seat and persists every accepted change privately. A seat may replace its pick or withdraw its confirmation until the last required seat confirms; that confirmation produces one seat-sorted batch, which the engine resolves before one durable write and one grouped reveal, and a change arriving after it is refused. Reload and reconnect restore a seat's pick and its confirmed state, and others see only who has confirmed. Complete batches are the canonical history; arrival order never becomes rules order. The owner chose changeable picks on 2026-10-21.

**Rationale.** This makes simultaneous play exact: a key legal at the opening stays legal at resolution, so there is no conflict resolution, and neither arrival order nor seat order decides anything. The last confirmation closes the point at once, so a late change has nothing to race with: it is refused, and a delayed request never reverts a newer pick because it carries an older seq.

**Rejected.** Applying picks in seat order or as they arrive. Appending every arrival to the rules history. Mutating the engine with partial picks. Playing Babylon's last card inside the sixth turn. Picks locked once confirmed (the provisional default, which the owner declined, because a player could not correct a pick while others were still thinking). A visible count of pick changes (it would show hesitation).

**Relation.** Haftgane only. Extends D02 (membership is checked per seat against that seat's frozen legal set) and D11 (a game is its setup plus its complete batches).

Source: [engine-architecture.md](../haftgane/engine-architecture.md), section 6; [platform-architecture.md](../haftgane/platform-architecture.md), "Decision points and acceptance" and "Owner decisions".

## D36. Setup draws from one seeded stream per random element

**Decision (2026-10-21).** All randomness is still resolved when the game is created. Seating, boards, sides, the guild draw and each Age's deck draw from their own stream, derived from the seed and a fixed tag pinned by a golden, with the shared generator.

**Rationale.** With one stream, an option that skips a draw (chosen seating, chosen sides) would shift every later shuffle, so options would change the deals. Separate streams also let a later element be added without moving the others.

**Rejected.** One stream consumed in order, as in Duel.

**Relation.** Haftgane only. Extends D11 for Haftgane. Duel's single stream is unchanged.

Source: [engine-architecture.md](../haftgane/engine-architecture.md), section 5.

## D37. Each viewer's knowledge is derived from the game record and sent by whitelist

**Decision (2026-10-21).** The engine value is the state plus a fixed-size **record** of every dealt packet, every pick, every public play and every discard build. Views, events, history and bot observations are projections of it, built by whitelist for each seat and for one spectator role, never by deleting fields from a full object. A seat's hand, offered keys, pending pick, tucked and sold cards and remembered knowledge reach only that seat; who has confirmed, and when, is public, and a seat's pick changes are not. Late joiners and resyncs get the same whitelist with a rebuilt history. Decided by the owner on 2026-10-21: each seat's view carries a private **notebook** (the hands it held, what it did with each card, and the pile at its own discard opportunities) with summary statistics drawn only from that seat's knowledge, each certain or an upper bound, so humans know what bots know. A former player seated in another seat gets only that seat's knowledge (D42).

**Rationale.** What a seat legitimately knows is a function of the record, so deriving it keeps views, bots and history consistent, and the whole game stays one comparable, hashable value.

**Rejected.** Knowledge stored per seat (derivable, and about three times the size). The record kept by the room. A shared private hint object. Full-state redaction by deletion. No notebook (less UI, but human and bot memory unequal). Statistics that guess (advice built on assumptions).

**Relation.** Haftgane only. Extends D03: still a full per-viewer snapshot after every change, now for up to seven seats and a spectator.

Source: [engine-architecture.md](../haftgane/engine-architecture.md), sections 4 and 10; [platform-architecture.md](../haftgane/platform-architecture.md), "Views, events and history".

## D38. Everything is revealed at the end unless the room hides it, and the engine has no resignation

**Decision (2026-10-21).** By default every hidden card is shown when the game ends and the log can be downloaded as `ajayeb-log/2` (setup, complete batches and the final fingerprint). A creation option keeps hidden cards hidden after the end, and then no download is offered, because the log's keys name sold and tucked cards. There is no victory before the end of Age III. Resigning hands the seat to a bot; it is a room act, so the engine has no resign key.

**Rationale.** It matches Duel's end policy. The score counts only public things, so a hidden end still shows the full result.

**Rejected.** Always revealing. A resignation or surrender victory in the engine.

**Relation.** Haftgane only. Follows D03 and D28 (downloaded logs keep their strict final-hash check). Duel's `ajayeb-log/1` is unchanged.

Source: [engine-architecture.md](../haftgane/engine-architecture.md), sections 10 and 12; [platform-architecture.md](../haftgane/platform-architecture.md), "Persistence, replay and compatibility".

## D39. Wonder sides are a room option

**Decision (2026-10-21).** By default every player chooses a side after seeing every board, in one simultaneous hidden point before Age I is dealt, and the choices are revealed together. Room options give every seat its day side, or draw sides at random.

**Rationale.** rules.md rulings 16 and 26. A simultaneous hidden choice fits the decision model and removes any advantage from choosing last.

**Rejected.** Open choice in seat order (slower online, and the first seat always chooses blind). Dealing Age I before the choice.

**Relation.** Haftgane only. New; Duel's wonder draft is a different mechanism. The owner accepted the simultaneous hidden choice on 2026-10-21.

Source: [engine-architecture.md](../haftgane/engine-architecture.md), section 5; [rules.md](../haftgane/rules.md), rulings 16 and 26.

## D40. The coin reserve is unlimited

**Decision (2026-10-21).** As in Duel: the reserve never runs out, no payment exceeds the treasure in the frozen position, and the base game takes no coins away.

**Rationale.** rules.md ruling 14: the rulebook sets no limit, and no rule covers an empty reserve.

**Rejected.** A cap at the coins in the box.

**Relation.** Haftgane only. Confirms D09 for Haftgane, as a rule of this game rather than an inheritance.

Source: [engine-architecture.md](../haftgane/engine-architecture.md), draft entries.

## D41. Compatibility lives in per-game file codecs and frame families

**Decision (2026-10-21).** Duel's room files stay on schema 1 with today's codec; Haftgane files use schema 2 with their own codec, an explicit game id (`haftgane`) and their own rules version. A bounded header decides the codec before any decoding or replay, and an unknown schema, codec or game is left untouched and unsupported, never archived as a broken Duel room. Hello accepts protocol versions 1 and 2. Duel's game frames keep one unchanged contract on both; version 2 only adds Haftgane frames (`table-state` and its requests), and a version-1 client that tries to join a Haftgane room gets the existing reload path. New DTOs are generated separately, leaving Duel's generated files byte-identical.

**Rationale.** No Duel room, frame or fixture changes, so the risk to Duel is confined to the dispatch seams, and old binaries already refuse schema 2 through their error count.

**Rejected.** Migrating or normalizing Duel records. Discovery messages before join. A second Duel wire contract. A game field added to Duel's state frame.

**Relation.** Both games' room files and frames; Duel's stay byte for byte as they are. Extends D13 and D16 to two games, extends D28's compatibility tuple to (schema and codec, game, game rules version), and keeps D29's archive layout and retention for both games.

Source: [platform-architecture.md](../haftgane/platform-architecture.md), "Persistence, replay and compatibility" and "Protocol and lazy client entry".

## D42. One mixed-seat lobby; Start freezes the table; admins switch who controls a seat, never the table

**Decision (2026-10-21).** A table is created in the shared lobby (D60) with a seat count and options. The creator holds one lobby seat and is its first admin; humans who open the link take free seats or spectate; an admin adds and removes bots, edits options and can move a lobby human to the spectators. There are no ready flags: Start is the consent, every seat must be filled, and an explicit Start option fills open seats with bots. Start applies the seating rule and freezes the seat assignment, the starting human count, the undo flag and the bot configuration. Seating is random, with an admin able to use the lobby order instead. A disconnect never changes neighbours or ends the game.

An admin may switch any seat between human and bot control at any time: a human replaced by a bot becomes a spectator, and the bot plays the seat's pick, confirmed or not. An admin may also seat any spectator in any seat, a bot's or a human's, whose occupant then becomes a spectator; the human seated takes over the seat's pick, still changeable, and the seat's knowledge. For each spectator the admin's seat menu shows the seat they held earlier in this game, if any (the previous-seat hint, sent to admins only). A former player seated elsewhere keeps their own memory of the old seat's hands, which the owner accepts; the server gives each viewer only its current seat's knowledge. Resignation hands one's own seat to a bot. No switch declares a surrender winner.

**Roles.** The real admins are the creator and every human an admin has promoted, and any admin may promote any human in the room, seated or spectating. An admin may demote the admins they promoted; the creator may demote anyone and cannot be demoted. Only when every real admin has been disconnected for five minutes does a temporary admin take the role: the connected human who has held their seat longest, or else the spectator connected longest. A real admin's return demotes the temporary admin at once, and a temporary admin away for five minutes passes the role on by the same rule. Roles, who promoted whom, temporary status and disconnect times persist across restart and reload. Every seat switch and role change shows in everyone's game log. Whoever requests a rematch is the creator of its successor lobby, which reserves seats for the original humans.

**Provisional calls,** which stand until the owner overturns them: any admin may step down; demoting an admin leaves the admins they promoted in place, so that only the creator can then demote those; a temporary admin has the seat powers, the lobby's edits and Start, but cannot promote or demote; and when the last real admin steps down, the temporary-admin rule applies at once instead of after five minutes.

**Rationale.** One lobby serves a solo table and a table of friends alike. Freezing at Start keeps undo, confirmation and bot work from depending on later events. Keeping the table fixed keeps neighbours, and so the rules, stable. The admins, not a timer, decide when an absent player's seat goes to a bot, and can give any seat to a returning or new human. Promotion lets a creator share the job. A temporary admin keeps a table whose admins have all gone from having nobody who can switch a stuck seat, and the real admins get the role back the moment one returns. The previous-seat hint shows an admin when seating someone would join what two seats know; the owner chose to allow that rather than forbid it.

**Rejected.** Separate bot and online modes. Readiness revisions. A connected-but-idle deadline. Permanent takeover by any seated human after two minutes (replaced by the admin powers). A former player only in their own former seat (a provisional rule the owner dropped). Passing the admin role after two minutes to the human seated longest (a provisional rule, replaced by real and temporary admins and five minutes).

**Relation.** Haftgane only, except that the roles also exist in the shared lobby before a Duel game starts (D60). Reuses D14's identity and claims. Extends D23 for Haftgane: an admin's settings belong to every admin, and promotion is one of the other ways to make admins that D23 left for later.

Source: [platform-architecture.md](../haftgane/platform-architecture.md), "Owner decisions" and "Lobby, admin powers and rematch".

## D43. Undo exists only in a game that started with exactly one human seat

**Decision (2026-10-21).** A game that started with exactly one human seat allows undo unless its creator turned it off. Undo turns off for the rest of the game as soon as a second human takes a seat, and no later resignation or switch turns it back on. Undo replays from setup to before the latest completed batch with the human's decision, dropping later batches and the pending block; it never changes a seat's controller or rerolls the deal. Undo after a reveal shows the bots' simultaneous picks, and deterministic bots repeat them.

**Rationale.** A solo player can correct a mistake without any other human's information being rolled back. Peeking at bot replies through undo is the same accepted trade as D15's in Duel's bot rooms.

**Rejected.** No undo in Haftgane (prevents peeking but removes solo correction). Undo in games with several humans.

**Relation.** Haftgane only. Extends D15 to Haftgane, where "bot room" means "started with one human seat".

Source: [platform-architecture.md](../haftgane/platform-architecture.md), "Owner decisions"; [bot-plan.md](../haftgane/bot-plan.md).

## D44. Every bot level gets the same honest information, and samples hidden cards only within proven bounds

**Decision (2026-10-21).** Bots never receive the game. Each gets the engine's `Observation`: the public position, its own hand and its legitimately remembered hands and pile looks, with a canonical encoding. Difficulty changes computation and noise, never access. The stronger levels sample hidden cards with the engine's `Determinize`, which fills every hidden place consistently with the whole observed history (including skipped Halikarnassos abilities), counts its work against a declared bound, and returns no world rather than an inconsistent one; the bot then falls back to Greedy. Policy randomness comes from the game id, policy version, logical point, seat and the observation's fingerprint, under a domain label separate from setup; never from the full state, pending keys, wall clock or arrival order.

**Rationale.** One engine-owned contract keeps what bots infer consistent with views and replay, and a bounded failure keeps play honest. Observation-derived seeds make the same knowledge give the same choice, which is the fairness test.

**Rejected.** Real hidden state with random mistakes. A second sampler in the bot package. Unbounded retries. Seeds from the full-state hash.

**Relation.** Haftgane only. Applies D01's no-cheating principle to bots and carries M3's fairness test (same public information, same choice) to seven seats.

Source: [bot-plan.md](../haftgane/bot-plan.md), "Honest information and bounded sampling"; [engine-architecture.md](../haftgane/engine-architecture.md), section 11; [platform-architecture.md](../haftgane/platform-architecture.md), "Bots: information, work and lifecycle".

## D45. Each bot gets fixed work per decision at its difficulty, through the existing permit pool

**Decision (2026-10-21).** Each difficulty has a fixed, deterministic iteration allowance per bot decision under a versioned preset. Table size, bot count, queue pressure and the order of other picks never reduce it. Before a bot searches, the room persists that seat's work row (policy version, preset, allowance); rows and accepted picks survive restart and are never redistributed, and a bot switched into a seat gets a full row. At most one bot job per room is queued or active, through `ai.Pool.AcquireFor` and `ReleaseFor`, released after each pick. Decided by the owner on 2026-10-21: equal work per bot at every table size, so a full table is slower rather than weaker.

**Rationale.** Fixed work keeps a chosen difficulty meaning the same thing when the creator adds bots. The existing pool already gives fair admission between rooms, and durable rows stop a restart or a seat switch from silently changing a pending choice.

**Rejected.** Dividing one room allowance among its bots (faster turns, weaker bots at full tables). Time-limited strength as normal policy. Simultaneous searches within a room. A new global CPU and memory scheduler. Cheaper work for a bot switched into a seat.

**Relation.** Haftgane only. New. Duel's budgets, presets, pool settings and review lane are unchanged.

Source: [bot-plan.md](../haftgane/bot-plan.md), "Fixed work, existing permits and frozen rows"; [platform-architecture.md](../haftgane/platform-architecture.md), "Owner decisions".

## D46. Bot levels compare root moves by simulating whole simultaneous rounds

**Decision (2026-10-21).** The levels are Apprentice (Greedy with softmax noise), Scholar (paired root comparisons that simulate every seat's simultaneous picks to the end of the current Age) and Master (the same with full-game rollouts). Random and deterministic Greedy are simulator baselines, and Greedy is also the legal fallback. A simulated round gathers every required choice from one frozen position before resolving it; opponents are independently self-interested policies drawn from a small mixture of strategies. Rewards blend rank and win credit. The owner chose to wait for Scholar: Greedy and Apprentice first complete an internal milestone, the public release offers Apprentice and Scholar, and Master may follow later. Persona names per level are decided.

**Rationale.** Paired root comparison has bounded storage and a clear information boundary, and the Age versus full-game horizon gives the levels a real difference.

**Rejected.** Duel's alternating minimax and two-player backup. A coalition of opponents. Enumerating joint actions. A simultaneous information-set tree before it proves itself at equal work.

**Relation.** Haftgane only. New. Duel's bots and their names are unchanged; Haftgane reuses the persona names with its own presets.

Source: [bot-plan.md](../haftgane/bot-plan.md), "Families and simultaneous search".

## D47. The bots' evaluation is written for this game's scoring and economy

**Decision (2026-10-21).** Rules, legal payments and exact scoring stay in the engine. The bot adds transparent future-value estimates in victory-point equivalents: per-neighbour military forecasts, science squares and sets, production and trade with both neighbours, chains, wonder progress, guild scopes, and capped passing and denial terms. Weights, temperature, mixture, horizon and work bounds freeze together as one bot-policy version, separate from the rules version.

**Rationale.** Two neighbours, nonlinear science and directional drafting need different incentives from Duel's.

**Rejected.** Duel's weights. Linear shield rewards. Treating all production as tradeable. Tuning on the release seeds.

**Relation.** Haftgane only. New.

Source: [bot-plan.md](../haftgane/bot-plan.md), "Evaluate this game's scoring and economy".

## D48. Bot strength is measured in seat-balanced deal blocks at the live presets

**Decision (2026-10-21).** `ajayeb-sim` gains game-selected N-player tournaments, with Duel's commands and output unchanged. A **deal block** fixes the decks, guilds, sides and seats; the focal policy rotates through every seat, results are reported by player count, seat and side, and 95% intervals keep a block's rotations together. Release targets are declared in advance (for Apprentice and Greedy against Random, Scholar against Greedy, Master against Scholar) and are measured at the same fixed per-bot presets as live play.

**Rationale.** Neighbours and passing direction can reverse rankings, so paired deal blocks reduce noise without pretending rotated games are independent.

**Rejected.** Two-seat swaps as the only pairing. Raw score alone. Reduced-budget proxies standing in for live settings.

**Relation.** Haftgane only; Duel's tournaments and reports are unchanged. Follows M3's practice of predeclared targets; no existing decision changes.

Source: [bot-plan.md](../haftgane/bot-plan.md), "Seat-balanced strength and privacy evaluation".

## D49. Haftgane capacity limits come from measurements on the host

**Decision (2026-10-21).** Existing room, client and queue limits stay. The only new limit is one configurable maximum of Haftgane rooms doing bot work at once, set from host measurements of the full seven-seat, six-bot case, mixed with Duel load. No room cap, memory reservation, payload ceiling or merge timer is invented before measurement.

**Rationale.** Under the owner's decision a small fixed room cap would refuse the third person who opens a room on the whole site; numbers chosen before measurement are guesses.

**Rejected.** A two-room cap, per-room memory reservations and fixed frame or file ceilings.

**Relation.** Haftgane only. New. Duel's limits are unchanged.

Source: [platform-architecture.md](../haftgane/platform-architecture.md), "Capacity and acceptance".

## D50. Haftgane ships without hints or an advisor

**Decision (2026-10-21).** The first release of Haftgane has no hint families and no post-move advisor. If they come later, the admin owns their settings as in D23, their absence never blocks play, and anything that depends on a hand is computed inside that seat's projection from its permitted information. A future advisor reviews a decision only after its batch resolves, privately to that human.

**Rationale.** Honest playable bots and measured full-table performance come first. Duel's shared hint object cannot carry private hands.

**Rejected.** Enabling Duel's hint ids for Haftgane. Pre-pick move suggestions. Advice that uses the true deal.

**Relation.** Haftgane only. Extends D23 (deferred for this game; ownership by the admin kept for later). Duel's hints and advisor are unchanged.

Source: [bot-plan.md](../haftgane/bot-plan.md), "Defer hints and private post-round advice"; [platform-architecture.md](../haftgane/platform-architecture.md), "Views, events and history".

## D51. Duel's client and the service worker stay as they are; Haftgane loads lazily

**Decision (2026-10-21).** The client picks a lazy game entry from the authoritative snapshot. A generic `/play/<code>` entry runs its hello and join only after the existing name prompt, and hands a Duel room to the unchanged Duel Session through its ordinary version-1 connection. The service worker keeps precaching the whole build, so a Duel player downloads changed Haftgane code once in the background per build; art stays out of the precache (D24). Cold Home stays within 200,000 gzip bytes, and cold Home and Duel entry stay within their baseline plus 5,000 gzip bytes, measured with service workers blocked; import checks prove that Home and Duel never execute Haftgane catalogs or renderers.

**Rationale.** It keeps Duel's Session and cache behaviour untouched and protects an open page's lazy chunks across a deploy. The cost is one extra round trip on generic Duel entry and some background bytes.

**Rejected.** Generalizing Duel's Session. Selective precache before install-transfer evidence exists.

**Relation.** The client that hosts both games; Duel behaves exactly as before. Keeps D05 and D24 as they are; adds the two cold-entry budgets.

**Amendment (2026-10-24).** The install-transfer evidence now exists: the install had grown to about 2.95 MB of body, 1.09 MB of it files that only the `/dev` pages load. The worker now leaves out exactly those files (M18.precache-without-dev-pages). A file is left out only when no real route can reach it, as read from Vite's client manifest and the route-to-node map, so every real page keeps its lazy chunks across a deploy. Dev pages no longer open offline. The rest of this decision stands.

Source: [platform-architecture.md](../haftgane/platform-architecture.md), "Protocol and lazy client entry" and "Capacity and acceptance".

## D52. The seat ring is drawn left to right in every language

**Decision (2026-10-21).** The left neighbour is always on the screen's left, and the other seats run across the top from left to right, also in right-to-left languages. The page chrome, panel contents and sheets follow the page direction; signed numbers are isolated and formatted with `Intl`.

**Rationale.** Left and right are rules terms that name real table directions (passing, the trading posts). Mirroring would put the left neighbour on the right.

**Rejected.** Mirroring the ring. Relabelling neighbours "first" and "second".

**Relation.** Haftgane only. Extends D10 and the existing convention that Duel's board region is left to right.

Source: [phone-layout.md](../haftgane/phone-layout.md), draft entries.

## D53. The hand never scrolls

**Decision (2026-10-21).** The whole hand is always visible as a grid sized by its container: four columns by two rows on phones, one row of seven in short landscape.

**Rationale.** Every card's cost and affordability must be comparable at a glance, and the board never scrolls sideways.

**Rejected.** A scrolling row. A fan. List rows (kept only as a fallback for very small screens).

**Relation.** Haftgane only. New.

Source: [phone-layout.md](../haftgane/phone-layout.md), draft entries.

## D54. Each turn ends with one grouped reveal and a recap

**Decision (2026-10-21).** When the last confirmation arrives, every seat is tagged with what it did at the same moment, and a recap of the turn opens by itself and closes on any tap; a button reopens it during the next turn. There are no per-seat or per-bot pauses. With Animation off the tags and the new hand appear at once and the recap still opens.

**Rationale.** Simultaneous turns should feel simultaneous; with up to six bots a sequence of pauses would add seconds to every turn; a recap makes six actions readable on a phone.

**Rejected.** Sequential reveals per seat. A recap only on request (kept as a setting, as the owner accepted).

**Relation.** Haftgane only. Follows D21: the recap carries information, so it survives Animation off.

Source: [phone-layout.md](../haftgane/phone-layout.md), draft entries; [platform-architecture.md](../haftgane/platform-architecture.md), "Bots: information, work and lifecycle".

## D55. Neighbour panels show server-computed trade prices and the military outlook

**Decision (2026-10-21).** Each seat's view includes, for each neighbour, the units that seat may buy and its price per unit after its discounts, with the discount's source, and the conflict token it would take against that neighbour if the Age ended now. Running scores for every seat join them, as the owner chose. The client shows them and computes none of them.

**Rationale.** D01 keeps the client thin: prices depend on discounts and the outlook on the Age, which is rules arithmetic.

**Rejected.** Deriving them in the client from public city data.

**Relation.** Haftgane only. Applies D01 to the new screens. The engine's view slice must add these fields (`M15.haftgane-views-and-observation` in [roadmap.md](roadmap.md)).

Source: [phone-layout.md](../haftgane/phone-layout.md), draft entries and "Server fields the screens rely on".

## D56. Seat crests are fixed by table position

**Decision (2026-10-21).** Seven crests and seven seat colours are assigned by seat order at Start, each crest a distinct shape.

**Rationale.** A stable identity for the whole game without a lobby picker, and shapes that differ for colour vision.

**Rejected.** Crests chosen in the lobby (more personal, but needs a picker and a rule for duplicates).

**Relation.** Haftgane only. New. The crest set is drawn in the vector loop under D18.

Source: [phone-layout.md](../haftgane/phone-layout.md), draft entries and owner answers.

## D57. Approved paintings are shared through game-specific bindings

**Decision (2026-10-21).** Haftgane reuses approved Duel paintings and their shipped, content-hashed URLs through explicit game-specific aliases (58 of the 80 designs from 55 paintings), and 21 new paintings cover the other 22 designs. New assets live in a game-qualified namespace. Rules ids and display ids stay independent, and Duel's art ids, files and hashes are unchanged byte for byte.

**Rationale.** Reuse saves 55 generations and keeps the two games visually related; new mixed-resource and profession scenes avoid misleading reuse.

**Rejected.** Regenerating all 80 designs. Treating every matching name as a fit. Copying Duel's art into a second directory. Changing Duel's art ids or hashes.

**Relation.** Haftgane only; Duel's art is unchanged. Extends D07, D18 and D24 to two games without changing them.

Source: [art-reuse.md](../haftgane/art-reuse.md), draft decisions.

## D58. Wonder paintings stay intact inside a wide live board

**Decision (2026-10-21).** The seven existing wonder paintings serve all fourteen sides, shown whole inside a wider responsive panel whose stage strip (two to four cells, drawn in CSS from server data) sits below the painting.

**Rationale.** The originals and their three sizes suffice, and the stages stay readable without covering the monuments.

**Rejected.** Cropping or stretching every painting into a wide backdrop. Baking costs and effects into pixels. Fourteen new paintings.

**Relation.** Haftgane only. New, under D07 and D24.

Source: [art-reuse.md](../haftgane/art-reuse.md), draft decisions.

## D59. Concealed card backs follow what the viewer may know

**Decision (2026-10-21).** In Haftgane every concealed Age III card, guilds included, uses the ordinary Age III back; mixed-Age piles and tucked stages are drawn procedurally with a neutral mark. The violet guild back appears only where the category is public, such as a rules preview. The client never chooses a back from a hidden engine id.

**Rationale.** Which guilds are in the game is hidden, so a distinct guild back would leak it.

**Rejected.** Applying Duel's public guild back to every game. Painting new backs to solve a redaction problem.

**Relation.** Haftgane only. Supersedes D12 for Haftgane only; Duel keeps D12.

Source: [art-reuse.md](../haftgane/art-reuse.md), draft decisions.

## D60. Duel and Haftgane share one online lobby, and its seat count picks the game

**Decision (2026-10-21).** Home keeps Duel's one-tap starts unchanged: play the bot, the tutorial, and pass and play. The online start ("play with friends") opens one shared lobby of 2 to 7 seats, whose admins set the seat count until Start. At Start, two filled seats create a Duel game with those two players, human or bot, under the same code and link; three to seven create a Haftgane game, and Haftgane's solo games use the same lobby. The lobby shows the settings of the game its seat count selects: at two seats Duel's (who starts, a bot seat's level), at three or more Haftgane's. It is its own small room kind, whose Start writes the chosen game's room file under the same code and swaps the actor; it replaces a lobby phase of Haftgane's own. The roles of D42 exist in the lobby for both games and carry into a Haftgane game. A Duel game started from the lobby has no admin powers except over D23's hint settings, which any of the lobby's admins may edit (the admin set is frozen at Start: no promotion, demotion or temporary admin inside a Duel game), and otherwise plays exactly like today's online Duel game with the same seats; Duel's current online sheet and its start-when-the-friend-joins flow retire. The undo of D43 and the seat switches of D42 apply to Haftgane games only.

**Rationale.** A group only has to know its size, as the owner suggested, while bot games and the tutorial stay one tap away. One lobby, built once, serves both games.

**Rejected.** (A) Two separate Home entries with Duel untouched (a group of two must know that it wants Duel). (B) Everything through the lobby, with Duel's one-tap starts removed (it costs one-tap bot games and changes more of Duel).

**Relation.** Both games: it changes Duel's online creation through one new create path, under a FULL review, and leaves every Duel game after Start as it is. Extends D31, since the lobby is one more registered room kind. D14's identity and claims carry through the swap.

Source: [platform-architecture.md](../haftgane/platform-architecture.md), "Shared lobby"; the owner's answers of 2026-10-21, including "for hint, any admin can edit them" and, for a Duel game, "you can allow duel with admins too".

## D61. A Duel room is entered on its own connection first

**Decision (2026-10-22).** The `/play/<code>` entry opens the ordinary version-1 Duel Session first, after the same name prompt, unless the room is known to be another game: a saved `game` tag other than `duel`, or a lobby or table code the page itself created. If the server refuses that join with its `protocol` error (it does so before claiming a seat or touching the room file), the page falls back to the discovery of D51 over version 2. Wherever a discovery connection still hands a room over to Duel, it stays open until the Duel Session has its first state, so the seat never shows its player as away. The server's answer to the join remains the authority; a tag only chooses which connection to try first.

**Rationale.** The FULL review of M14.client-game-entry measured what D51's discovery cost Duel: the opponent saw their friend join, leave and come back on every cold invite and every reload, a bot move due on entry ran on the discovery connection and its events were lost, and a reload showed the board 0.3 s later at a 100 ms round trip and 0.9 s later at 300 ms. Trying Duel first removes all three for every Duel room, and only a future room with a stale or missing tag pays one refused join.

**Rejected.** Discovery before every entry (D51 as first written). Only overlapping the two connections (it hides the flash, but the bot still runs on discovery and the delay stays). One version-2 connection shared by both games (it changes Duel's Session and transport more than this needs). A peek join that the server neither counts as presence nor lets wake the bot (it is the room-info operation that D41 rejected).

**Relation.** Amends D51's entry mechanism: Duel's Session gains one case, a `protocol` refusal of a join after a welcome, which it reports as a wrong-game answer for the page instead of a fatal error. D51's budgets and its service-worker rule stand. D60's lobby uses the same overlapping hand-off when its Start turns the room into a Duel game.

Source: the M14.client-game-entry review, 2026-10-22; [platform-architecture.md](../haftgane/platform-architecture.md), "Protocol and lazy client entry".
