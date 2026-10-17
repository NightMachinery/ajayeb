# tools/chrome: headless Chrome over the DevTools protocol

Small, dependency-free Node 24 scripts (built-in `WebSocket` and `fetch`) for screenshots, layout probes and SVG contact sheets. They start their own headless Google Chrome with a throwaway profile and close it when they finish. They never touch an everyday browser profile. The Playwright e2e suite in `e2e/` is separate; use these scripts when a whole test run would be more than you need.

Run every launching command inside the guard's browser lock, which keeps the machine to one Chrome at a time:

```sh
python3 tools/heavy.py run --kind browser -- node tools/chrome/shot.mjs --url / --probe --out /tmp/home.png
```

Without the lock (the guard sets `AJAYEB_HEAVY_KIND=browser`), the scripts refuse to launch and exit 2. `--force` exists as a manual escape hatch, but repository checks and agent runs must use the guard.

## The scripts

- `cdp.mjs`: the CDP client library. `CDP.connect(wsUrl)`, then `send(method, params)`, `on(event, fn)` and `waitFor(event)`. `openPage(browser)` opens a tab in a fresh browser context, so storage never leaks between runs, and `evaluate(page, fn, ...args)` runs a function in it.
- `launch.mjs`: starts Chrome on `127.0.0.1:$CHROME_PORT` (default 38959) with a profile in the OS temp directory named `ajayeb-chrome-<pid>-XXXXXX`. Its `close()` kills Chrome's process group and any process still naming the profile, then deletes the profile; exit and SIGINT, SIGTERM or SIGHUP do the same. Run it alone as a self-test, or with `--hold SECONDS` to keep one browser open for several runs (export the `CHROME_CDP` it prints).
- `shot.mjs`: one page, one screenshot and an optional probe. See below.
- `svg-render.mjs`: a labelled contact sheet of SVG files at several sizes on several backgrounds, plus optional metadata. See below.
- `tutorial-walk.mjs`: walks the guided first game from Home as a fresh player would, with a screenshot and checks at every coach step. See below.

Environment:

- `CHROME_CDP=http://127.0.0.1:PORT` attaches to a Chrome that is already running instead of launching one. The tools then open their own tab in a fresh context and never kill that browser.
- `CHROME_PORT` changes the DevTools port (0 lets Chrome pick one).
- `CHROME_BIN` names the Chrome executable. On macOS the default is the copy in `/Applications`; on Linux the scripts look for `google-chrome` or `chromium` on `PATH`.
- `AJAYEB_ORIGIN` changes the default app origin of `shot.mjs`.

## shot.mjs

```sh
python3 tools/heavy.py run --kind browser -- node tools/chrome/shot.mjs --origin http://127.0.0.1:38951 --url /play/ABCD \
  --token ajayebShotTokenA00000A --name Ana --vp 390x844@2 --theme dark --probe --out /tmp/a.png
```

- `--url` is relative to `--origin` (default `http://127.0.0.1:38951`, the dev server).
- `--token`, `--name` and `--theme light|dark|system` are written to the localStorage keys `ajayeb:token`, `ajayeb:name` and `ajayeb:theme` before any page script runs. The server derives the player from the token, so a token shows exactly the seat that a driver client using the same token holds. It must have the real shape, the base64url form of 16 bytes: 22 characters of `A-Z`, `a-z`, `0-9`, `-` and `_`, whose last character carries only two bits and so is `A`, `Q`, `g` or `w`, such as `ajayebShotTokenA00000A`. The client replaces any other stored value with a fresh token, and the page then opens as a stranger, so the script refuses any other `--token`.
- `--vp WxH[@dpr]`, or the presets `mobile` (390x844@2) and `desktop` (1280x800@1). The default is 390x844@2. A short side under 600 px also turns on mobile emulation and touch. The viewport is set with `Emulation.setDeviceMetricsOverride`, never by resizing a window.
- `--storage JSON` writes more localStorage keys the same way, before any page script runs, so a screenshot can use settings other than the defaults. The value is a JSON object of keys to values: a string value is written as it is, any other value as its JSON, so `--storage '{"ajayeb:settings":{"animationSpeed":"off","showOpponentMove":false}}'` stores the settings object the app reads. The flag can be given more than once, a later key wins over an earlier one, and `--token`, `--name` and `--theme` win over the same key. Read the key names and value shapes from the app's code (for example `web/src/lib/ui/settings.svelte.ts` and `web/src/lib/session/undo-default.svelte.ts`); a value the app does not accept falls back to its default, as it would for a player.
- `--scheme light|dark` emulates the system colour scheme, which the `system` theme follows. `--motion` defaults to `reduce`, but the app deliberately follows its own Animation setting rather than that OS preference. Use `--storage` with `animationSpeed` to control game motion.
- `--wait-ready` waits for `data-ready` on `<html>` (at most 2.5 s); otherwise the script waits a fixed `--settle` of 800 ms. Both then wait for fonts and two animation frames.
- `--full` captures the whole page instead of the viewport.
- `--selector CSS` captures one element instead: the script scrolls the first match into view, waits two animation frames for layout, and clips the PNG to its bounding box plus a `--pad PX` margin (default 8). It needs `--out` and does not mix with `--full`. It fails with a clear message, and exit 1, when the selector is invalid, matches nothing, or matches an element with no size; with several matches it takes the first and says so. An element taller than the viewport makes the script grow the emulated viewport's height (never its width) until the element fits, and the status line reports the grown size. That is needed because the app shell scrolls inside a container, not the window, so the browser does not paint content below the viewport, and the reason `--full` is unreliable on pages such as `/dev/gallery`. Growing the height can change layouts that depend on it (the `short:` variant), so prefer a selector box that fits the viewport when that matters. Example: `--url /dev/gallery --selector '.board-gallery > section:nth-of-type(3)'` shoots the resources and science section.

- `--vision greyscale|deuteranopia|protanopia|tritanopia` renders the page as seen with that colour vision. The three deficiencies use Chrome's own emulation (`Emulation.setEmulatedVisionDeficiency`); `greyscale` sets a `grayscale(1)` filter on `<html>` just before the capture, so it covers fixed layers and sheets too. Use it to check that something reads without colour, such as the badge states, the category glyphs or the crests. The status line names the vision.
- `--no-autofocus` keeps an autofocused field from scrolling the page. A script installed before the page's own makes every `focus()` call pass `preventScroll` and ignores the `autofocus` property (which Svelte sets), and the active element is blurred before the probe and the capture, so no focus ring shows either. Without it, the name prompt's input on `/dev/gallery` scrolls the gallery's scrolling container to itself as the page loads, and a `--full` capture comes out blank. Leave it off when focus is what you are checking.

`--probe` prints one JSON object on stdout; status lines go to stderr. Its fields:

- `hscroll`, `iw`, `sw`: horizontal scroll, `innerWidth` and `scrollWidth` (`vscroll`, `ih` and `sh` are the vertical counterparts).
- `body`: the first 400 characters of visible text, for checking state without pixels.
- `overflow`: up to 15 visible elements that stick out of the viewport, outermost first, skipping those clipped by an on-screen scroll or overflow container.
- `tap`: the number of enabled controls, the smallest one (`el`, `w`, `h`, `min` in CSS px; a visually hidden input counts through its label), and how many are under 24 and under 44 px.
- `theme` (the `data-theme` on `<html>`), `ready` (`data-ready`), `storage` (the `ajayeb:*` keys, with the token shown only by its length), `url`, `title` and `vp`.
- `errors` and `failed`: console errors, uncaught exceptions, and requests that failed or returned 400 or more.

## svg-render.mjs

```sh
python3 tools/heavy.py run --kind browser -- node tools/chrome/svg-render.mjs --in web/src/lib/icons/svg --sizes 16,24,48,128 \
  --bg '#f4ecdc,#1c1915' --cols 6 --out /tmp/sheet.png --meta /tmp/sheet.json
```

It inlines every SVG into one HTML page through `Page.setDocumentContent` and takes a single screenshot. Each background is a band whose text colour is what `currentColor` and the `var(--i1, currentColor)` fallbacks resolve to; `--fg` overrides the automatic choice. Ids are prefixed in every copy, so gradients and other `url(#id)` references keep working. `--dpr` defaults to 1, which shows the real rasterization at small sizes. The metadata JSON lists, per file, its viewBox, width and height attributes, element and shape counts, bytes, the colours it uses, and warnings such as a missing viewBox or a `<text>` element.

For the inner loop on a single icon, `tools/art/render-fast.sh in.svg SIZE out.png [bg] [ink]` renders with `rsvg-convert` in milliseconds and needs no browser. librsvg differs from Chrome in places, so check each finished batch with `svg-render.mjs`.

## tutorial-walk.mjs

```sh
python3 tools/heavy.py run --kind browser -- node tools/chrome/tutorial-walk.mjs \
  --origin http://127.0.0.1:38951 --vp 360x740@2 --theme light --locale en --out /tmp/walk
```

The standalone tutorial probe follows the same policy as [the tutorial specs](../../docs/testing.md#the-tutorial-m10tutorial-client). It needs a test build served with `--dev --dev-endpoints`, because it reads the expected lesson order from `GET /api/dev/tutorial`. It seeds a fresh device (a new token, the name Walker, the theme, the Animation setting, the move summary, and the first-time tips on with the tutorial offer new), opens Home, presses Play the tutorial and walks with the same policy as the Playwright specs, through the free part to the completion card, whose New game must open the vs-AI sheet on Home. It uses touch when the viewport's short side is under 600 px, else a mouse.

- `--vp WxH[@dpr]`, `--theme light|dark`, `--locale en|fa`, `--animation normal|off` (default off) and `--summary on|off` (default on) set up the device.
- `--out DIR` gets one PNG per coach step (`NN-<step>-<kind>[-note].png`), the Home offer, the completion card and the Home it leads to, and `report.json`: per step its id, kind, surface, placement, target and bubble boxes, occlusion verdict, the milliseconds from the frame that brought it to the bubble, and the failed checks; then the expected and observed lesson order, the first-time tips seen, console errors and the first failure.
- `--no-shots` skips the PNGs, `--trace` records what every press hit, and `--max-actions N` (default 120) bounds the free part.

The checks at every coach step: a bubble with a ringed target; the target in the viewport (a region taller than a short screen needs most of what fits on screen) and, for an element target, hit by `elementFromPoint` at its centre; the full settled ring inside a sheet’s viewport and clipping ancestors (ending at its top-layer dialog); the bubble inside the viewport, off an element target and over at most 40% of a region target; its buttons at least 44 px tall; no horizontal scroll; exactly one `[data-coach-target]`. It exits 1 on the first failed check, a lesson order that differs from the script, or a console error, and 0 when the walk is clean.

## Checking that nothing is left running

Our profiles are named `ajayeb-chrome-*` and Playwright's `playwright_chromiumdev_profile-*`, so this prints nothing when no browser of ours is left:

```sh
/bin/ps -axo pid=,command= | grep -F -e ajayeb-chrome -e playwright_chromiumdev_profile | grep -v grep
```

Prefer it to `pgrep -fl 'ajayeb-chrome|--headless'`: `pgrep -f` also matches the shell that runs it when the pattern is part of that shell's command line, and `--headless` matches other projects' browsers too.
