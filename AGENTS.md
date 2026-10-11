# Ajayeb: agent guide

Ajayeb is an unofficial, original-art web implementation of the two-player board game 7 Wonders Duel. It has three modes (versus a bot, hot-seat on one device, and online between two devices), and it must play well on a phone. This file is the working agreement for every coding agent and contributor. Read it fully before you change anything.

## Architecture in one paragraph

The **Go server holds the only rules engine.** The SvelteKit client is thin: it renders the view the server sends and sends back **action keys** (canonical strings such as `construct:baths`, `wonder:pyramids:baths`, `token:law`, `starter:1`). Every mode is a server room over WebSocket, run by one actor goroutine per room. The engine is pure and deterministic: all randomness is resolved when a game is created from its seed, so a game is fully described by its seed, options and action keys. After every accepted action the room sends each connection a full, per-viewer redacted snapshot built by whitelist (`ViewFor`), including the list of legal `choices` with their cost breakdowns. The client never decides what is legal and never computes a cost; it shows what `choices` says. [docs/architecture.md](docs/architecture.md) has the diagram, the data flow of one move, and the hidden-information model.

```
browser: Transport -> Session -> Presenter -> components      (no rules code)
           ^  state snapshots            |  {"t":"act", "key":"construct:baths", ...}
           |                             v
server:  ws -> room actor -> engine (Legal, Apply, ViewFor) + ai -> store (seed + keys)
```

## Where things live

- `server/internal/engine` and `server/internal/engine/catalog`: the rules and the card, wonder and token data. Pure Go.
- `server/internal/view` and `server/internal/wire`: the DTO and message types that `tygo` turns into TypeScript.
- `server/internal/{ai, room, store, ident, ws, httpapi, webdist}`: bots, room actors, persistence, identity, transport, HTTP endpoints, the embedded web build.
- `server/cmd/{ajayeb, ajayeb-sim, ajayeb-drive, ajayeb-gen}`: the server, the simulator, the WebSocket game driver, the code generator.
- `web/`: the SvelteKit client. `web/src/lib/wire/*.gen.ts` is generated; never edit it by hand.
- `e2e/`: Playwright playthroughs (`specs/`) and the UX screenshot loop (`ux/`).
- `tools/`: `heavy.py` (the memory and run-mode guard), `chrome/` (headless Chrome tools), `art/` (art scripts).
- `art/`: art direction, the asset manifest, style references, approved originals and the vector specs and logs.
- `docs/`: start at [docs/README.md](docs/README.md).

## Running things

**Use `make <target>` for everything heavy.** The targets wrap their commands in `tools/heavy.py`, which enforces the memory budget and the run mode for every agent on the machine at once. Never run raw heavy commands such as `go test ./...`, `go build`, `golangci-lint run`, `pnpm install`, `pnpm build`, `vite build`, `svelte-check` or `playwright test`.

- The main targets are `make install`, `make dev`, `make verify-quick`, `make verify`, `make verify-full`, `make e2e` (with `GREP=@full` and similar), `make e2e-smoke`, `make gen-check`, `make build` and `make shots`. Read the `Makefile` for the full list.
- For targeted Go tests, use the target's variables: `make test-go GO_PKGS=./internal/engine/... GO_RUN=TestMilitary`.
- When no target fits, wrap the command yourself: `python3 tools/heavy.py run --kind test -- go -C server test ./internal/engine -run TestMilitary` (the Go module lives in `server/`). Kinds are `test`, `build`, `check`, `lint`, `browser`, `codex`, `install`, `fuzz`, `bench` and `light`; `python3 tools/heavy.py --help` is authoritative.
- **Exit code 75 means the guard refused** (memory pressure, a busy lock, or a paused or out-of-window run mode). Through `make`, a refused step makes `make` itself exit 2; look for "Error 75" or a line starting with "heavy: refused" in the output. Do not retry in a tight loop. Report it, or wait with `--wait`.
- Run **one heavy command at a time.** Do not start a second build while a test run is going.
- `python3 tools/heavy.py status` shows pressure, mode, window and who holds each slot.

**Browsers.**

- At most **one headless Chrome** runs on the machine at a time. The guard's `browser` lock enforces this for Playwright and for `tools/chrome/*`.
- Playwright uses the installed Chrome (`channel: 'chrome'`, headless). The `tools/chrome` scripts launch their own headless Chrome with a throwaway profile and kill it on exit. The `chrome-cdp` skill (`.claude/skills/chrome-cdp/SKILL.md`) explains how to use them for screenshots, layout probes and SVG contact sheets.
- Every browser is closed at the end of its run. Check with `/bin/ps -axo pid=,command= | grep -F -e ajayeb-chrome -e playwright_chromiumdev_profile | grep -v grep`, which prints nothing when no browser of ours is left. Do not use `pgrep -f` for this: it also matches the shell whose command line holds the pattern, and a `--headless` pattern matches other projects' browsers.
- **Never automate the owner's Brave browser.** It shares the owner's profile and storage, steals focus and cannot emulate viewports. At most, open a URL in it for the owner to look at.

**Processes.**

- Stop every process you started before you return, by PID. Never leave a dev server, a Go server or a browser running.
- No dev servers in worktrees, ever, and no Vite dev server during unattended work: tests and screenshots use the test build served by the Go binary.
- Local ports are 38951 to 38959: 38951 Go server, 38952 Vite dev, 38953 `vite preview`, 38954 e2e server, 38959 Chrome DevTools for the tools. Vite dev and preview forward `/ws`, `/api` and `/health` to `AJAYEB_SERVER` (default `http://127.0.0.1:38951`).
- Set `NO_PROXY=127.0.0.1,localhost` for local HTTP when a proxy is configured.

**Shell traps.**

- On macOS the userland is BSD (`sed`, `grep`, `date`, `find`, `stat`). In scripts, prefix load-bearing binaries with `command` so a user alias or function cannot shadow them.
- Use `go tool air`, never bare `air` (a personal shell function shadows it). Call the real `codex` binary (scripts read `CODEX_BIN`), never the shell wrapper of the same name.
- Some global git excludes files ignore any top-level `server/` directory; the repo's `.gitignore` re-includes ours. If a new file does not show up in `git status`, run `git check-ignore -v <path>` to see which rule hides it.
- Never run `sudo`.

## Conventions

### Go engine purity

The engine is the single source of truth for the rules, so it must be deterministic and easy to replay.

- `internal/engine` and `internal/engine/catalog` use the standard library only. Their non-test files must not import `time`, `os`, `net/...`, `log`, `sync`, `unsafe` or `math/rand` (v1). `math/rand/v2` is allowed only inside `rng.go`. An import-guard test and golangci-lint (`depguard`) enforce this.
- **Integers only** in rules code. No floats. (The AI's evaluation function may use floats; it is not rules code.)
- **No maps in rules logic.** Use arrays indexed by enums, such as `[NumResources]int`. Never iterate a map to produce output, because Go randomizes map iteration order. (DTO maps are acceptable because `encoding/json` sorts their keys.)
- All chance is resolved in `NewGame`. `Apply` uses no randomness. The state is a fixed-size value struct, so `Clone` is a plain copy.
- In view and wire types, initialize slices to empty, never nil, because a nil slice marshals as JSON `null`. Give every field an explicit `json` tag, and use `omitempty` only on pointer fields (they become optional fields in TypeScript). A reflection test checks this.
- The server validates moves **by set membership**: it applies the engine's own action for a key that appears in the legal list, and never builds an action from client data.
- The server sends ids, codes and structured events, never display strings. That is what makes the client translatable.
- No `fmt.Print*` outside `cmd/` (golangci-lint `forbidigo`). Code is formatted with `gofmt` and `goimports`.
- Rules tests come first in each engine slice and are written from `docs/rules/`, which states the rules in our own words. Rulings on ambiguous rules live in `docs/rules/decisions.md`.

### Svelte and the web client

- **Svelte 5 runes only:** `$state`, `$state.raw`, `$derived`, `$effect`, `$props`. No `export let`, no `$:`, no stores (`svelte/store`), no `on:` directives (write `onclick={...}`), no `<slot>` (use snippets and `{@render ...}`).
- Prefer `$derived` over `$effect`, and never write `$state` inside an `$effect`. Use `$state.raw` for the large server views, which are always replaced whole.
- The client stays thin: no rules code, no cost arithmetic, no legality checks. Render `choices`; send keys.
- Card elements carry `data-card="<id>"` and every actionable button carries `data-act="<key>"`. These attributes are the end-to-end test contract; do not rename or remove them.
- One component per file. Leaf components take plain props so the gallery can render them standalone.
- **Tailwind, mobile-first:** unprefixed classes style the phone, and `sm:`, `md:`, `lg:` add to them. Two custom screen variants exist: `short:` (little height, such as a phone in landscape) and `split:` (room for two columns: a desktop width of at least 64rem, or a short landscape screen). The board never scrolls horizontally.
- **Tailwind's default colour palette is switched off** (`--color-*: initial` in `web/src/app.css`), so every colour follows the theme tokens. A class such as `bg-blue-500` silently generates nothing. Use the semantic utilities (`bg-table`, `bg-panel`, `text-fg`, `text-fg-muted`, `border-rule`, `bg-accent`, `text-on-accent`) and the game colours (`bg-card-*`, `text-sun-ink`, `bg-res-*`). `web/README.md` lists them.
- **Logical CSS properties only:** `ms-*`, `me-*`, `ps-*`, `pe-*`, `start-*`, `end-*`, `text-start`, `text-end`, `border-s`, `rounded-s` and so on, never `ml`, `mr`, `pl`, `pr`, `left`, `right`, `text-left` or `text-right`. The lint allows a physical property only on a line marked `physical-ok`. The board region itself is `dir="ltr"` on purpose, because its layout is spatial.
- Support **dark mode** and the game's own **Animation** setting everywhere. With Animation off, animations become instant and the state change stays visible. The OS's reduced-motion preference is deliberately not read (D21).
- **Every user-visible string goes through `t()`** from the typed message tables in `web/src/lib/i18n/` (one file per area). That includes `aria-label`, `title` and `alt` text. Card, wonder and token names are keyed by engine id. Use `Intl` for numbers and plurals. Server error codes map to messages in the tables.
- Put `dir="auto"` on every element that shows user-provided text, such as player names.
- **Tooltips use our own helper, never the native `title` attribute** (until the helper lands, add no new `title=`). Every board element and icon-only button gets one; skip it when it would only repeat a visible label. Keep the text short, put it through `t()`, and show only numbers the server sent. A concept tooltip may add a small vector diagram whose labels are translatable text, not baked into the SVG. A fine pointer shows it on hover (after a short delay) and on keyboard focus, and Escape closes it. On touch, a tap shows it on an element with no action; an actionable element keeps its tap action (the card sheet carries the same information) and shows the tooltip on a long-press, whose release never triggers the action. Any tap closes an open tooltip and is swallowed, so it never acts on what lies beneath. One tooltip at a time.

### Assets

- **Vectors** (icons, symbols, frames, badges, the military track, the logo) are hand-authored SVG, made by Opus subagents in a render, screenshot, critique loop: `tools/art/render-fast.sh` for each inner iteration, `tools/chrome/svg-render.mjs` once per batch for browser fidelity. librsvg draws `currentColor` as black, so pass `render-fast.sh` its optional fifth argument (the ink colour) when rendering on a dark background. The rules for drawing them are in `art/vector/STYLE.md`, the per-set specs in `art/vector/specs/`, and the iteration logs in `art/vector/log/`. Icon sources live in `web/src/lib/icons/svg/` and are built into one inline sprite. Codex is never used for vectors.
- **Bitmaps** (card, wonder and token art, backs, textures) are generated by Codex `image_gen` through the `tools/art/` scripts, strictly one at a time under `heavy.py --kind codex`. The art direction is in `art/style.md` and the subjects in `art/subjects.json`.
- **Approved originals are committed** as lossless WebP, pixel-identical to the generated PNG, under `art/originals/<kind>/<id>.webp` (`tools/art/process.mjs` writes and verifies them). Rejected attempts stay outside the repo, in the private run directory.
- **Shipped bitmaps are AVIF only**, under `web/static/art/`. They are derived from the originals by the art processing script. Never hand-edit a shipped file; regenerate it from its original.
- The app references only approved art, through a generated index, and draws procedural placeholders for everything else, so there are no broken images.
- Never copy official art, iconography, card frames, the logo or rulebook text. Fonts must be OFL and self-hosted. See [docs/legal.md](docs/legal.md).

### Documentation

- Keep `docs/` current as you change things, and keep this file and the README current too.
- Write plain, clear prose. No tables (use lists). No em dashes or en dashes as sentence connectors.
- `docs/dev/roadmap.md` changes only at the end of a milestone. Day-to-day progress lives in the private run state.

### Privacy (this repository is public)

- Commit no personal information: no absolute home-directory paths (write `~/...` or repo-relative paths), no email addresses, no machine hostnames, no usernames other than the GitHub organization `NightMachinery`, no tokens or keys.
- **Private run state lives outside the repo** in `~/tmp/agents/ajayeb/` (override with `AJAYEB_RUN_DIR`): `TASK.md` (owner instructions, verbatim), `STATE.md` (progress, newest first), `NOTES.md` (iteration notes and learnings), `mode.json`, and the `heavy/`, `logs/`, `results/`, `briefs/`, `patches/`, `shots/`, `art-work/`, `reports/`, `design/` and `workflows/` directories. Never copy any of it into the repo.

## Commits

- Stay on `master`. Worktree branches are temporary and are integrated by rebasing onto `master`, running `make verify-quick` and fast-forwarding, one branch at a time.
- Commits are **atomic and logically grouped**: one cohesive change per commit, usually one per successful slice. Commit at natural endpoints and push at the end of a milestone.
- **Commit with an explicit pathspec,** because other sessions may share the index. Run `git status` right before committing and `git show --stat HEAD` right after, to confirm that only your paths landed. `git commit -- <paths>` rejects untracked files, so add them first:

```sh
git add -- server/internal/engine/military.go server/internal/engine/military_test.go
git commit -m "engine: add military track and looting (M2.S5a)" \
  -m "Slice: M2.S5a" \
  -m "Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>" \
  -- server/internal/engine/military.go server/internal/engine/military_test.go
```

- The subject format is `area: summary (slice id)`. Areas include `engine`, `ai`, `sim`, `server`, `web`, `e2e`, `ux`, `art`, `tools`, `docs`, `ci` and `chore`. Slice ids look like `M2.S5a` (see [docs/dev/roadmap.md](docs/dev/roadmap.md)).
- Every agent-authored commit ends with the trailer `Co-Authored-By: Claude Opus 5.5 (1M context) <noreply@anthropic.com>`.
- Never rewrite or force-push published history. If a commit picked up someone else's staged work, report it instead of fixing it.
- Some files have a single writer (the coordinator or the integrator): `server/go.mod`, `server/go.sum`, `package.json` files, `pnpm-lock.yaml`, the `Makefile`, `*.gen.ts`, `AGENTS.md`, `docs/protocol.md`, and the frozen engine types (`state.go`, `action.go`, `events.go`). A worker that needs a change there describes it in its result file.

## Owner phrases

The owner steers the long run with a few fixed phrases. Each one is first recorded verbatim in `~/tmp/agents/ajayeb/TASK.md`, then acted on. The procedures are described in [docs/dev/orchestration.md](docs/dev/orchestration.md).

- **"switch to nightshift [HH:MM-HH:MM]"** (the window defaults to 00:00-10:00 local time):
  1. Run `python3 tools/heavy.py mode nightshift --window HH:MM-HH:MM`.
  2. Arm recurring cron jobs on off-minutes: the start procedure at 00:03, the last call at 09:12 and the wind-down at 09:44, plus the hourly recovery ping at minute 17, which is always armed. For another window, shift the three times by the same offsets (start plus 3 minutes, end minus 48 minutes, end minus 16 minutes).
  3. Record the cron ids in `STATE.md`.
  4. If the current time is outside the window, run the wind-down right away.
- **"switch to continuous"** (or "nightshift off"): delete the three window cron jobs, run `python3 tools/heavy.py mode continuous`, and resume work.
- **"pause ajayeb"**: run the wind-down now, then `python3 tools/heavy.py mode paused`. Nothing heavy runs until the owner resumes.
- **"resume ajayeb"**: set the mode back to what it was before the pause (continuous, or nightshift with its window) and run the start procedure.
- **"ajayeb status"** (or "good morning"): give a review rebuilt from `git log`, `heavy.jsonl`, the result files and `STATE.md`, never from memory.

## Working unattended

- Start by reading the Learnings index at the top of `~/tmp/agents/ajayeb/NOTES.md`. Your brief settles the design questions; do not stop to ask.
- Before starting a slice, run `git log --grep "Slice: <id>"`. If it is already committed, report it done. If its owned files are dirty, continue from that state.
- Touch only the files your brief says you own. Ask for changes to anything else in your result.
- Write your result to the path your brief assigns. Report honestly: a partial or failing result says so.
