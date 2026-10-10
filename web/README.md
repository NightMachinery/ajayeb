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
- `src/lib/ui/` holds UI state classes, such as the theme (`theme.svelte.ts`) and the safe-area override (`safe-area.ts`), and small display mappers such as the military track's side (`track.ts`) and the online helpers (`online.ts`: the room-code input normaliser and alphabet, the invite link, seat presence and its changes, the QR path).
- `src/lib/dev/flags.ts` decides whether the `/dev` routes exist.
- `src/routes/` holds the pages. `/` is Home, `/play/<code>` is a room, and `/dev/*` holds the developer pages.
- `static/` is served as is: the fonts with their licenses, and the favicon.
- `font-sources/` keeps the original TTF fonts and `subset.sh`, which rebuilds the shipped WOFF2 subsets in `static/fonts/`.

## Session: talking to the server

`src/lib/session/` is the client's whole connection layer. It decides nothing about the game: what may be played is the view's `choices`, what it costs is their cost breakdowns, and whether this client may act comes from the state frame. [docs/protocol.md](../docs/protocol.md) is the contract it implements.

- `transport.ts` defines the Transport (send, onMessage, onState, close, and an optional `reconnect` that drops the socket and opens a new one). `ws-transport.ts` is the WebSocket to `/ws` on the page's origin, with reconnects (exponential backoff with jitter, capped at 15 s; no retry after a 1008 `protocol` or `bad-hello` close). `fixture-transport.ts` answers from recorded frames, for dev pages and tests, and `scripted-transport.ts` is driven by hand in unit tests.
- `identity.ts`: the identity token in `localStorage` (`ajayeb:token`, 22 base64url characters, never logged), a random client id per page load, and the display name (`ajayeb:name`).
- `session.svelte.ts`: the Session. It says `hello` on every socket, creates and joins rooms, and sends acts and undos with the tab's `seq` counter and the current version as `base`. It keeps the latest state frame and view in `$state.raw` (replaced whole), the connection state (`connecting`, `open`, `reconnecting`, `offline`), the pending request with a `slow` flag after 300 ms, the last refusal as a message key, and `canAct`. After a reconnect it joins again and re-sends the pending request unchanged. A request the server dropped under its frame rate limit (`reject rate` naming its seq) is re-sent unchanged after a short pause, up to three times, before the refusal shows. `catalogStale` is set when a view's `catalogHash` differs from the client's catalog.
  - `setName(name)` changes the display name the next `hello` sends. When `hello` already went out under another name, it reconnects, so the room shows the new name at once.
  - In a bot room an undo can cross the bot's answer: the server refuses it as `stale` because the answer moved the version. The Session then waits for the fresh state and, if that state still offers undo, sends the undo once more with the same seq and the fresh version as `base`. Only a second refusal shows the stale message.
  - `rematch()` asks for the rematch (a spectator only follows one). `rematchAsked` says this tab asked, so the room page moves to the rematch room as soon as the server's `created` names it; the other player sees the offer on the game-over dialog instead. A request lost to a reconnect is sent again after the join.
- `presenter.svelte.ts`: the Presenter, which holds the frame on screen and moves it forward one queued frame at a time (capture hook, swap, tick, play hook). It has no animations yet, but it paces a bot's answer: a frame that answers a thinking frame waits until the thinking frame has been up `minThinkingMs`, and when the answer takes a card or drafts a wonder it sits in `preview` for `preRollMs` while the old frame stays up. Both settings may be functions; `app.svelte.ts` reads them from the animation speed (`botDwellMs` and `botPickMs` in `ui/motion.svelte.ts`: 700 ms and 300 ms at normal speed, 300 ms each with animation off). A frame with others queued behind it is never paced, `fastForwardAfter` catches up after a reconnect, `hurry()` shows a held answer at once (an undo pressed during the dwell) and `clear()` forgets the room when the game screen goes away. Input is open when `presenter.idle && session.canAct`, so it stays locked through the dwell and the preview.
- `rooms.svelte.ts`: the rooms this browser has played (`ajayeb:rooms`: code, mode and last seen).
- `context.ts` shares a page's Session and Presenter with its components (`setGameContext`, `getSession`, `getPresenter`); `app.svelte.ts` holds the app's one live connection, opened the first time a page needs it.
- `test-hook.ts` defines `window.__ajayeb` for end-to-end drivers (see [docs/testing.md](../docs/testing.md)).
- `DebugBoard.svelte` is a plain view of a room with every choice as a `data-act` button. No page uses it any more; it stays for debugging.

`src/lib/catalog/` has typed lookups (`cardBySlug`, `wonderBySlug`, `tokenBySlug`) over `src/lib/wire/catalog.gen.ts`, the presentation catalog that `make gen` writes from the engine's data.

The pages: `/play/<code>` joins a room and shows the game screen (`Room.svelte`). A first visit with no stored name, to a room this browser has not played, shows `screens/NamePrompt.svelte` first, and only then joins. A room that cannot be opened shows `screens/RoomError.svelte`, with a friendly message for the server's code and a way Home. Home's "Play online" opens `screens/OnlineSheet.svelte` (name, crest, Create, which sends `create` with mode online) and "Join with a code" opens `screens/JoinSheet.svelte` (six characters from the server's alphabet, case-insensitive; a pasted invite link works too); `/dev/new?mode=hotseat|ai|online` creates a room and opens it (with `seed=` or `scenario=`, through the server's dev endpoint); `/dev/fixture/<name>` shows a recorded game state in the game screen with no server.

## The game screen

`src/lib/components/game/GameScreen.svelte` is the board of every mode. It takes the page's Session and Presenter and an `onhome` callback, and decides nothing about the game.

- **Layout.** One markup, with CSS grid areas chosen by `breakpoints.viewport` (`src/lib/layout/breakpoints.svelte.ts`) through a `data-layout` attribute. Phone portrait: top bar, opponent strip, military track and progress tokens, structure (taking the remaining height), own strip, turn bar. Phone landscape: the compact track and tokens move into the top bar, and the strips take side columns. Tablet: stacked, with city bands under each strip. Desktop: city panels in 300 px side columns, 380 px from 1600 px wide. The shell is `100dvh` and never scrolls. During the wonder draft the area between the top bar and the turn bar holds the Draft screen.
- **Who sits where.** The board region is `dir="ltr"`. The player is at the bottom (or on the left); a spectator sees seat 0 there. In hot-seat the board faces the player to move on phones and stays fixed elsewhere, with the active panel lit; the setting overrides both.
- **The card sheet** (`CardSheet.svelte`) opens on a tapped slot. It shows the card at full tier, its category and effect, the cost breakdown from the matching choice (or disabled choice), the opponent's price and the discard gain, and the Build, Discard and Build a wonder buttons, each with its key as `data-act`. A covered card shows its cover count and no actions. Build a wonder steps to `WonderStep.svelte`, which lists the player's wonders with cost and affordability. `card-actions.ts` only sorts the view's choices for one card; it never parses keys or computes costs.
- **Wonder first.** Tapping one of your wonders opens `sheets/WonderSheet.svelte`. "Build this wonder" starts the pick mode: the turn bar asks for a card (with Cancel), takeable cards carry a wonder badge, and tapping one opens the card sheet to confirm. Both flows send `wonder:<wonder>:<card>`.
- **Committing.** Hot-seat and bot rooms send on the tap; online rooms use `ConfirmButton` when the confirm-moves setting asks for it, in the card sheet and in the wonder draft. On send the sheet closes, the slot shows the pending outline and input locks until the ack; after 300 ms the turn bar shows the slow state. A refusal shows its message above the turn bar.
- **Toasts** never cover the opponent's strip or the military track. On a phone and a tablet they sit just above the turn bar. On a desktop and in phone landscape (outside the draft) they sit at the foot of this player's own column, clear of the structure's bottom row, the cards a player can take.
- **Undo** is in the top bar, and in the toast that names the move just made (hot-seat and bot rooms only; online rooms have no undo). In a bot room it also takes back the bot's reply (the server does that); pressed while a bot answer is held for its dwell, the answer shows first and is then taken back.
- **Against the bot.** The bot's seat is named by its persona (`names.ts`: Apprentice for easy, Scholar for normal, Master for hard) wherever a seat name shows. While it thinks, the turn bar reads "<persona> is thinking" and its strip or panel shows `hud/ThinkingDots.svelte`, which stands still under reduced motion. During the Presenter's preview the slot it takes, or the draft offer it picks, carries `data-bot-pick` and a ring in its crest colour (`botPick` in `card-actions.ts` finds it from the answer's events).
- **The military track** has the capital of the seat at the bottom (or on the left) at its left end: a player at the Moon sees it mirrored, with the pawn, the zones and the looting tokens following (`ui/track.ts`). Spectators and the Sun see it as the view states it.
- **The game menu** (the top bar's first button, every mode) holds Home, Settings and the log. Home leaves the room; the room stays on Home's Continue list until its game ends, and the hint under Home that says so is gone once the game is over.
- **Online rooms.**
  - While the room is in the lobby, the game screen shows `screens/Lobby.svelte`: the code in large type, the invite link with Copy link (and a toast) and Share (only where `navigator.share` exists), and a QR code drawn from `uqr`, which loads only when the lobby shows. The lobby turns into the draft when the friend's join arrives.
  - The other player's strip or panel shows `hud/PresenceDot.svelte`: a filled dot when connected, a dashed ring and the word "away" when not, so it reads in greyscale. A toast says when they join, leave or come back, and while they are away the turn bar's waiting text says so.
  - While this tab's own connection is down, the ConnectionPill (icon only in the top bar below desktop widths) and the turn bar say so, the board stays open for inspection, and every move button reads "Waiting for connection" and is disabled. The fresh state after the reconnect restores the board.
  - A spectator sees the board read-only, with "Watching" in the turn bar, a note in the card sheet and no actions. Players see the spectator count (`hud/SpectatorCount.svelte`) in the top bar.
  - Rematch on the game-over dialog asks for the rematch room and moves there. The other player sees "<name> wants a rematch" and a Join the rematch button; a spectator gets Watch the rematch.
- **Hot-seat** shows the pass-device screen between turns of different players when the setting is on (by default on touch screens), with the last move's summary and an undo for the previous player.
- **Draft, choices and game over.** `screens/Draft.svelte` fills the main area during the wonder draft. `choices/ChoiceHost.svelte` stays mounted through the play (it shows pending choices and the bar that says who starts a new age), except while the pass-device screen is up, since its modal panels would cover that screen. `screens/GameOver.svelte` is its own centred dialog once the game is over. They take `{view, canAct, onact}` and `{view, onrematch?, onhome}`.
- **Test contract.** Structure cards keep `data-card` and action buttons `data-act`. Further hooks for drivers: `data-control="wonder-step"`, `"wonder-build"`, `"wonder-pick-cancel"`, `"undo"`, `"log"`, `"settings"`, `"open-city"`, `"menu"`, `"menu-home"`, `"menu-settings"`, `"menu-log"`; `data-wonder` on wonder buttons; `data-marked` on pickable slots; `data-bot-pick` on the slot or draft offer a bot is about to take; `data-thinking` (the bot's seat) on the game screen and on the thinking seat's panel; `data-turn` (the TurnState kind) on the turn bar; `data-left-seat` and `data-mirrored` on the military track; `data-pass-device` on the pass screen; `data-toast` on toasts. Online: `data-lobby`, `data-room-code`, `data-invite-link`, `data-qr` and `data-lobby-seat` in the lobby, `data-control="copy-link"`, `"share-link"`, `"lobby-home"`; `data-presence` (`connected` or `away`) on presence dots and seat panels; `data-away` on the turn bar while the waited-for player is away; `data-spectators` on the spectator count; `data-watching` on the spectator's card-sheet note; `data-rematch` on the rematch button; `data-home-hint` on the menu's Home hint; `data-name-prompt`, `data-name-input`, `data-control="name-submit"`; `data-online-sheet`, `data-control="online-create"`, `"online-join"`; `data-join-sheet`, `data-code-input`, `data-control="join-submit"`, `"home-join"`; `data-room-error` and `data-control="error-home"`.

The `/dev/gallery` page has a Game screen section that shows recorded states (draft, Age I, Age II, Age III, a pending choice, game over, and the online lobby, opponent-away and watching states) in frames of a phone and a desktop size. Its Online section shows the lobby (a fixed fake code and QR, also reconnecting), the name prompt, the online and join sheets, the room error, and the connection states of the pill, the presence dot and the turn bar.

## Styling

- Tailwind's default colour palette is switched off. Use the semantic utilities, such as `bg-table`, `bg-panel`, `text-fg`, `text-fg-muted`, `border-rule`, `bg-accent` and `text-on-accent`, so every colour follows the theme. The palette and game colours are there too (`bg-card-red`, `text-sun-ink`, `bg-res-wood` and so on).
- `dark:` follows `data-theme="dark"` on `<html>`, which reflects the in-app choice, not only the system setting.
- Two screen-level variants exist: `short:` (a height of 32rem or less, such as a phone in landscape) and `split:` (room for two columns: 64rem wide, or a short landscape screen).
- Use logical properties only (`ms-*`, `pe-*`, `text-start` and so on). Padding by the physical safe-area insets is the one deliberate exception.
- The two players' crests are fixed by seat: seat 0 is the Sun (turquoise, `sun` colours) and seat 1 is the Moon (pomegranate, `moon` colours), in every mode and on every screen.

## Strings

Every user-visible string goes through `t(key, params)` from `$lib/i18n`. The English tables in `src/lib/i18n/en/` define the keys, and the compiler checks each call: an unknown key, a missing placeholder value, or a count that is not a number is a type error. Plural messages are objects of forms (`one`, `other`, exact `=0` and so on), chosen with `Intl.PluralRules`; numbers are formatted with `Intl.NumberFormat`. `i18n.locale` and `i18n.dir` are reactive, and the root layout copies them to `<html lang dir>`.

## Icons

`src/lib/icons/svg/*.svg` are the hand-authored icon sources: one standalone `<svg viewBox="0 0 24 24">` (or `0 0 12 12` for a `-sm` variant) per file, two-tone through `fill="var(--i1, currentColor)"` for the primary shape and `var(--i2, rgba(128,128,128,0.65))` for the secondary accent. The `--i2` fallback is an opaque neutral grey, not `currentColor` or `transparent`: nothing in the app sets `--i1`/`--i2` yet, and a transparent or same-colour fallback made the secondary detail invisible wherever it sits over an `--i1`-filled region of the same resolved colour (opacity cannot separate two coincident fills of one colour). `names.ts` lists every icon name; `sprite.ts` builds them into one inline sprite (`import.meta.glob('./svg/*.svg', { query: '?raw' })`) and throws at build time if a name has no source file or a source file is missing from the list. `IconSprite.svelte` inlines the sprite once, in the root layout; `Icon.svelte` renders one icon (`<Icon name="resource-wood" />`), sized in `em`, decorative (`aria-hidden`) unless given a `label`.

This is the placeholder set for M5: simple, clean, geometric shapes, never copied or traced from official iconography. The M9 milestone replaces it through its own Opus render-critique loop (see `art/vector/` once that milestone starts); `/dev/gallery` renders every icon at a glance for a visual check in both themes.

## UI lints

`scripts/lint-ui.mjs` (plain Node, no new dependencies; run by `pnpm -C web lint`, so `make lint-web` runs it too) checks two project rules that ESLint and Prettier do not:

- **Logical CSS only.** No physical Tailwind utility (`ml-`, `mr-`, `pl-`, `pr-`, `left-`, `right-`, `text-left`, `text-right`, `rounded-l`, `rounded-r`, `border-l`, `border-r`, `scroll-ml-`/`mr-`/`pl-`/`pr-`, `origin-left`/`right`, `float-left`/`right`) and no physical CSS property (`margin-left`, `margin-right`, `padding-left`, `padding-right`, `border-left`, `border-right`, bare `left:`/`right:`) in `.svelte` or `.css` files under `src/`. Mark a deliberate exception with `physical-ok` on the same line (the board region and safe-area insets are the known exceptions).
- **No hard-coded English.** Every template text node and every `aria-label`, `title`, `alt` and `placeholder` string literal in a `.svelte` file must go through `t()`. The rule reads the whole template, so it also catches text on a line of its own and text next to an expression (`{count} coins`). `<code>` and `<kbd>` content is exempt (it is literal, not translatable), and a line can opt out with `i18n-ok` when the text is genuinely not shipped UI copy, such as a `/dev` fixture label.

Both rules are pure functions (`findPhysicalCssViolations`, `findHardcodedTextViolations`) exported from `lint-ui.mjs` and unit-tested from `src/lib/lint-ui.test.ts` (kept under `src/` so Vitest's `include` picks it up).

## Fonts

Fraunces (600, headings) and Inter (400, 600 and 700, the interface) are self-hosted under the SIL Open Font License 1.1. The license texts ship next to the fonts in `static/fonts/`. The WOFF2 files are Latin subsets made by `font-sources/subset.sh` from the original TTFs in `font-sources/`. Upstream: Fraunces 1.000 from github.com/undercasetype/Fraunces and Inter 4.001 from github.com/rsms/inter.

## Environment

`PUBLIC_DEV_ROUTES=1` turns on the `/dev` routes in a build; they are always on under `vite dev` and in the test-mode build (`make build-web-test`). When it is unset they answer 404. See `.env.example`.
