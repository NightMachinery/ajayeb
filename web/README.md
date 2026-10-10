# web: the Ajayeb client

The browser client: a SvelteKit 2 single-page app with Svelte 5 runes, Tailwind 4, Vite 8 and TypeScript 6.0. It is thin on purpose. The Go server holds the only rules engine; this app renders the views the server sends and sends back action keys. See [AGENTS.md](../AGENTS.md) for the conventions and [docs/architecture.md](../docs/architecture.md) for the data flow.

## Running it

Heavy commands go through the root `Makefile`, which wraps them in the memory guard (`tools/heavy.py`):

- `make check-web` runs `svelte-kit sync` and `svelte-check` (the type gate).
- `make lint-web` runs ESLint and `prettier --check`.
- `make test-web` runs the Vitest unit tests (one worker thread).
- `make build-web` writes the static build to `web/build/`; `make build` embeds it in the Go binary.
- `python3 self_host.py dev-start` runs the Go server and the Vite dev server together (see `make dev`).

The package scripts behind them are `dev`, `build`, `preview`, `check`, `lint`, `format` and `test`.

Ports: Vite dev listens on 38952 and `vite preview` on 38953. Both forward `/ws`, `/api` and `/health` to the Go server at `http://127.0.0.1:38951`, or to `AJAYEB_SERVER` when that is set.

## Layout

- `src/app.html` is the page template. Its inline script applies the stored theme before the first paint, so the page never flashes the wrong theme.
- `src/app.css` imports Tailwind and the tokens, declares the fonts, maps the tokens to Tailwind utilities, and holds the base styles (the `100dvh` app shell, the safe-area variables `--safe-t`, `--safe-r`, `--safe-b`, `--safe-l`, focus rings and reduced motion).
- `src/lib/styles/tokens.css` is the single source of colour, type and motion tokens, in OKLCH. Cards keep their paper colours in both themes; only the table surfaces change, from an ivory manuscript page (light) to lapis night (dark).
- `src/lib/i18n/` holds the message tables and `t()`.
- `src/lib/ui/` holds UI state classes, such as the theme (`theme.svelte.ts`) and the safe-area override (`safe-area.ts`).
- `src/lib/dev/flags.ts` decides whether the `/dev` routes exist.
- `src/routes/` holds the pages. `/` is Home, `/play/<code>` is a room, and `/dev/*` holds the developer pages.
- `static/` is served as is: the fonts with their licenses, and the favicon.
- `font-sources/` keeps the original TTF fonts and `subset.sh`, which rebuilds the shipped WOFF2 subsets in `static/fonts/`.

## Session: talking to the server

`src/lib/session/` is the client's whole connection layer. It decides nothing about the game: what may be played is the view's `choices`, what it costs is their cost breakdowns, and whether this client may act comes from the state frame. [docs/protocol.md](../docs/protocol.md) is the contract it implements.

- `transport.ts` defines the Transport (send, onMessage, onState, close). `ws-transport.ts` is the WebSocket to `/ws` on the page's origin, with reconnects (exponential backoff with jitter, capped at 15 s; no retry after a 1008 `protocol` or `bad-hello` close). `fixture-transport.ts` answers from recorded frames, for dev pages and tests, and `scripted-transport.ts` is driven by hand in unit tests.
- `identity.ts`: the identity token in `localStorage` (`ajayeb:token`, 22 base64url characters, never logged), a random client id per page load, and the display name (`ajayeb:name`).
- `session.svelte.ts`: the Session. It says `hello` on every socket, creates and joins rooms, and sends acts and undos with the tab's `seq` counter and the current version as `base`. It keeps the latest state frame and view in `$state.raw` (replaced whole), the connection state (`connecting`, `open`, `reconnecting`, `offline`), the pending request with a `slow` flag after 300 ms, the last refusal as a message key, and `canAct`. After a reconnect it joins again and re-sends the pending request unchanged. A request the server dropped under its frame rate limit (`reject rate` naming its seq) is re-sent unchanged after a short pause, up to three times, before the refusal shows. `catalogStale` is set when a view's `catalogHash` differs from the client's catalog.
- `presenter.svelte.ts`: the Presenter, which holds the frame on screen and moves it forward one queued frame at a time (capture hook, swap, tick, play hook). It has no animations yet; `minThinkingMs` and `fastForwardAfter` are there for the bot dwell and for catching up after a reconnect. Input is open when `presenter.idle && session.canAct`.
- `rooms.svelte.ts`: the rooms this browser has played (`ajayeb:rooms`: code, mode and last seen).
- `context.ts` shares a page's Session and Presenter with its components (`setGameContext`, `getSession`, `getPresenter`); `app.ts` holds the app's one live connection, opened the first time a page needs it.
- `test-hook.ts` defines `window.__ajayeb` for end-to-end drivers (see [docs/testing.md](../docs/testing.md)).
- `DebugBoard.svelte` is a temporary, plain view of a room with every choice as a `data-act` button, until the real game screen exists.

`src/lib/catalog/` has typed lookups (`cardBySlug`, `wonderBySlug`, `tokenBySlug`) over `src/lib/wire/catalog.gen.ts`, the presentation catalog that `make gen` writes from the engine's data.

The pages: `/play/<code>` joins a room and shows it; `/dev/new?mode=hotseat|ai|online` creates a room and opens it (with `seed=` or `scenario=`, through the server's dev endpoint); `/dev/fixture/<name>` shows a recorded game state with no server.

## Styling

- Tailwind's default colour palette is switched off. Use the semantic utilities, such as `bg-table`, `bg-panel`, `text-fg`, `text-fg-muted`, `border-rule`, `bg-accent` and `text-on-accent`, so every colour follows the theme. The palette and game colours are there too (`bg-card-red`, `text-sun-ink`, `bg-res-wood` and so on).
- `dark:` follows `data-theme="dark"` on `<html>`, which reflects the in-app choice, not only the system setting.
- Two screen-level variants exist: `short:` (a height of 32rem or less, such as a phone in landscape) and `split:` (room for two columns: 64rem wide, or a short landscape screen).
- Use logical properties only (`ms-*`, `pe-*`, `text-start` and so on). Padding by the physical safe-area insets is the one deliberate exception.

## Strings

Every user-visible string goes through `t(key, params)` from `$lib/i18n`. The English tables in `src/lib/i18n/en/` define the keys, and the compiler checks each call: an unknown key, a missing placeholder value, or a count that is not a number is a type error. Plural messages are objects of forms (`one`, `other`, exact `=0` and so on), chosen with `Intl.PluralRules`; numbers are formatted with `Intl.NumberFormat`. `i18n.locale` and `i18n.dir` are reactive, and the root layout copies them to `<html lang dir>`.

## Fonts

Fraunces (600, headings) and Inter (400, 600 and 700, the interface) are self-hosted under the SIL Open Font License 1.1. The license texts ship next to the fonts in `static/fonts/`. The WOFF2 files are Latin subsets made by `font-sources/subset.sh` from the original TTFs in `font-sources/`. Upstream: Fraunces 1.000 from github.com/undercasetype/Fraunces and Inter 4.001 from github.com/rsms/inter.

## Environment

`PUBLIC_DEV_ROUTES=1` turns on the `/dev` routes in a build; they are always on under `vite dev`. When it is unset they answer 404. See `.env.example`.
