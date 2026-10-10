---
name: chrome-cdp
description: Screenshot, probe and inspect Ajayeb pages, and render SVG contact sheets, with our own headless Chrome over the DevTools protocol (tools/chrome). Use it for visual checks of a page or game state at a given viewport, theme and seat; for layout questions such as horizontal scroll, overflowing elements, tap-target sizes and on-screen text; and for icon and vector review. Use Playwright (e2e/, make e2e) for full playthroughs instead.
---

# chrome-cdp: our own headless Chrome, driven over CDP

`tools/chrome/` holds dependency-free Node 24 scripts that talk the Chrome DevTools Protocol (CDP) over Node's built-in WebSocket. They launch **their own** headless Google Chrome with a throwaway profile and kill it when they finish. `tools/chrome/README.md` has every flag; this file is the know-how.

## Rules

- **Never drive the owner's Brave or any everyday browser.** It shares their profile and storage, steals focus and has no viewport emulation. The scripts start a private headless Chrome; `CHROME_CDP` attaches only to a browser you started yourself.
- **Always run under the browser lock**, one Chrome on the machine at a time:
  `python3 tools/heavy.py run --kind browser -- node tools/chrome/shot.mjs ...`
  Without it (`AJAYEB_HEAVY_KIND=browser` is unset) the scripts refuse with exit 2. `--force` is only for when you know no other browser job runs. If the guard waits on free swap or memory, let it wait; do not force.
- **Always close the browser.** The scripts do this themselves, including on errors and Ctrl-C. If you used `launch.mjs --hold`, stop it when done. Before you hand back, check that nothing you started is left: our profiles are named `ajayeb-chrome-*` and Playwright's `playwright_chromiumdev_profile-*`, so `/bin/ps -axo pid=,command= | grep -F -e ajayeb-chrome -e playwright_chromiumdev_profile | grep -v grep` must print nothing. (`pgrep -f PATTERN` also matches the shell that runs it, and `--headless` matches other projects' browsers.) Kill leftovers by PID, never with `pkill -f`.
- Never start a Vite dev server for screenshots. Serve the test build with the Go binary instead (below).

## Serve something to look at

`shot.mjs` defaults to `--origin http://127.0.0.1:38951`. If the owner's dev servers (`self_host.py dev-start`) already serve that port, use them. Otherwise serve the test build with the Go binary on 38951:

```sh
make build-server build-web-test
server/bin/ajayeb --addr 127.0.0.1:38951 --data .cache/shots-data --static web/build --dev --dev-endpoints --bot-delay 0
```

Run it in the background, note its PID, and kill that PID when you are done. Use `NO_PROXY=127.0.0.1,localhost` for anything that talks HTTP to it.

## Screenshots and probes

```sh
python3 tools/heavy.py run --kind browser -- node tools/chrome/shot.mjs \
  --url /play/ABCD --token tok-a --name Ana --vp 390x844@2 --theme dark --probe --out /tmp/a.png
```

Then look at the PNG with the Read tool. What bites:

1. **Viewport is emulated, not resized.** `--vp WxH[@dpr]` (or `mobile`, `desktop`) goes through `Emulation.setDeviceMetricsOverride`. A short side under 600 px also turns on mobile emulation and touch, so coarse-pointer layouts apply. Never try to size a window.
2. **Identity lives in localStorage.** `--token`, `--name` and `--theme` set `ajayeb:token`, `ajayeb:name` and `ajayeb:theme` on the page's origin before any page script runs. The server HMACs the token into the user id, so the same token always means the same seat. Each run uses a fresh browser context, so one perspective never leaks into the next; capture each seat in its own run.
3. **`--theme system` follows `--scheme light|dark`**, the emulated OS preference. Shots default to reduced motion.
4. **Wait for the page.** Fixture routes set `data-ready` on `<html>` when fonts, images and layout are done; pass `--wait-ready`. Other pages get a fixed `--settle` (800 ms by default).
5. **Shoot one section with `--selector CSS`** (and `--pad PX`, default 8) rather than `--full`. It scrolls the first match into view and clips the PNG to its box, which costs far fewer image tokens than a whole page. The app shell scrolls inside a container, so `--full` misses everything below the first screen (on `/dev/gallery`, for one); `--selector` instead grows the emulated viewport's height until a tall element fits, and says so on stderr. A selector that matches nothing fails loudly, so a renamed class cannot hand you a blank shot. Example: `--url /dev/gallery --selector '.board-gallery > section:nth-of-type(3)'`.

6. **Colour vision.** `--vision greyscale|deuteranopia|protanopia|tritanopia` renders the page as seen with that colour vision (Chrome's emulation for the three deficiencies, a `grayscale(1)` filter on `<html>` for greyscale). Shoot the parts where colour carries meaning (badges, category glyphs, crests, band colours) once per vision you care about.
7. **Autofocus moves the page.** On `/dev/gallery` the name prompt's autofocused input scrolls the gallery's scrolling container as the page loads, so a `--full` capture comes out blank and a capture lands somewhere unexpected. Pass `--no-autofocus`: focus never scrolls, the autofocus property is ignored, and the active element is blurred before the capture. Leave it off when focus itself is under test.

`--probe` prints JSON on stdout: `hscroll`, `iw`, `sw`, the first 400 characters of `body` text, `overflow` (elements sticking out of the viewport), `tap` (the smallest enabled tap target and the counts under 24 and 44 px), `theme`, `ready`, `storage` (token masked), and `errors` and `failed` (console errors, exceptions and failed requests). Use it to assert state and layout without spending image tokens; open the PNG when the look matters.

## Driving a game into a state

Reach the state on the server, then photograph one seat:

- **Server dev endpoints** (the server must run with `--dev --dev-endpoints`): `POST /api/dev/rooms {mode, seed, scenario | actions}` creates a room already at a given point of a game. They arrive with milestone M4.
- **`ajayeb-drive`** (also M4) plays seats over WebSocket with stable tokens such as `tok-a` and `tok-b`; `--hold-at` keeps the sockets open and prints the room code. Then shoot the seat you want: `shot.mjs --url /play/<code> --token tok-a`.

Until these exist, only the static pages (home, settings, `/dev` routes in a test build) can be photographed.

## SVG contact sheets

- Inner loop, one icon, no browser: `tools/art/render-fast.sh in.svg 512 out.png [bg] [ink]` (rsvg-convert, about 10 MB).
- Once per batch, real browser fidelity:

  ```sh
  python3 tools/heavy.py run --kind browser -- node tools/chrome/svg-render.mjs \
    --in web/src/lib/icons/svg --sizes 16,24,48,128 --bg '#f4ecdc,#1c1915' --out /tmp/sheet.png --meta /tmp/sheet.json
  ```

  It inlines the SVGs (so `currentColor` and `var(--i1)` resolve against each band) into one labelled grid and takes one screenshot. `--dpr 1` (the default) shows the true small-size raster; `--dpr 2` is for a closer look. The meta JSON lists viewBox, element and shape counts, bytes, colours and warnings per file.

## More than screenshots

`tools/chrome/cdp.mjs` is a tiny client: `CDP.connect(wsUrl)`, `openPage(browser)` for an isolated tab, then `page.send(method, params)`, `page.on(event, fn)` and `evaluate(page, fn, ...args)`. Use `launch()` from `launch.mjs` to get the browser, and close both in a `finally`. Any CDP method works, for example `Input.dispatchMouseEvent` to click (for colour-blindness checks, `shot.mjs --vision` already wraps `Emulation.setEmulatedVisionDeficiency`).
