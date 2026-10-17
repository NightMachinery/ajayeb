#!/usr/bin/env node
// tutorial-walk.mjs: walks the guided first game in our own headless Chrome,
// from Home, using only the tutorial's DOM contract (docs/testing.md, "The
// tutorial"), as a fresh player would: a screenshot per coach step and a
// JSON report of each step's target, bubble and checks. It stops at the
// completion card and exits non-zero on the first failed check. It is the
// M11 player reviewer's tool; the Playwright specs (e2e/specs/tutorial*.ts)
// check the same contract in the suite.
//
// Run it inside the guard's browser lock, against a test build served with
// the dev endpoints (the step order is read from GET /api/dev/tutorial):
//
//   python3 tools/heavy.py run --kind browser -- node tools/chrome/tutorial-walk.mjs \
//     --origin http://127.0.0.1:38951 --vp 360x740@2 --theme light --locale en \
//     --animation off --out /tmp/walk
//
// The walk policy: while a coach step shows, press Next on an info step,
// otherwise click its target (the one [data-coach-target]); while the move
// summary is open, tap an empty part of the top bar (Escape when there is
// none); close a first-time tip with Got it; in the free part, open the
// first accessible face-up card and press its first enabled move button
// (build, else discard), else the first enabled move button on the page;
// answer a starter choice with starter:0; stop at the completion card, and
// press its New game, which must open the vs-AI sheet on Home. Every press
// first checks that the point is still on its element, and waits 150 ms
// after (a person's pace: two emulated taps inside the double-tap window
// can be taken as one gesture).
//
// The checks at every coach step: the target is in the viewport and
// elementFromPoint at its centre hits it; the bubble is inside the viewport,
// does not overlap an element target and covers at most 40% of a region
// target; its buttons are at least 44 px tall; the page does not scroll
// sideways. The report also has the placement and the milliseconds from
// the frame that brought the step to the bubble.

import fs from 'node:fs';
import path from 'node:path';
import { parseArgs } from 'node:util';
import { CDP, evaluate, openPage, sleep } from './cdp.mjs';
import { isMain, launch, reportAndExit } from './launch.mjs';

const USAGE = `Usage: node tools/chrome/tutorial-walk.mjs [options]

  --origin URL          the app (default $AJAYEB_ORIGIN or http://127.0.0.1:38951)
  --vp WxH[@dpr]        the viewport (default 390x844@2); a short side under 600 px is touch
  --theme light|dark    the app theme (default light)
  --locale en|fa        the app language (default en)
  --animation normal|off  the game's Animation setting (default off)
  --summary on|off      the opponent move summary popup (default on, as a player has it)
  --out DIR             where the PNGs and report.json go (required)
  --max-actions N       give up after this many moves (default 120)
  --no-shots            skip the screenshots (the report only)
  --trace               record what each press hit and the state after it
  --force               launch without the guard's browser lock
`;

function vpOf(s) {
  const m = /^(\d+)x(\d+)(?:@(\d+(?:\.\d+)?))?$/.exec(s);
  if (!m) throw new Error(`bad --vp ${s}`);
  const width = +m[1];
  const height = +m[2];
  return { width, height, dpr: +(m[3] ?? 1), touch: Math.min(width, height) < 600 };
}

/** A fresh client token of the real shape (16 random bytes, base64url). */
function newToken() {
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  return Buffer.from(bytes).toString('base64url');
}

async function navigate(page, url) {
  const loaded = page.waitFor('Page.loadEventFired', { timeoutMs: 20_000 });
  loaded.catch(() => {});
  const r = await page.send('Page.navigate', { url });
  if (r.loaderId) await loaded;
}

/** Waits until fn (run in the page) returns a truthy value. */
async function until(page, fn, arg, timeoutMs = 15_000, what = 'condition') {
  const end = Date.now() + timeoutMs;
  for (;;) {
    const v = await evaluate(page, fn, arg);
    if (v) return v;
    if (Date.now() > end) throw new Error(`timed out waiting for ${what}`);
    await sleep(50);
  }
}

/** Clicks or taps at a point, as a real pointer would. */
async function press(page, x, y, touch) {
  if (touch) {
    await page.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x, y }] });
    await sleep(40);
    await page.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] });
  } else {
    await page.send('Input.dispatchMouseEvent', { type: 'mouseMoved', x, y });
    await page.send('Input.dispatchMouseEvent', { type: 'mousePressed', x, y, button: 'left', clickCount: 1 });
    await page.send('Input.dispatchMouseEvent', { type: 'mouseReleased', x, y, button: 'left', clickCount: 1 });
  }
}

/** The centre of the first visible match, after scrolling it into view. */
function centreOf(selector) {
  for (const el of document.querySelectorAll(selector)) {
    const r = el.getBoundingClientRect();
    if (r.width <= 0 || r.height <= 0) continue;
    if (r.bottom < 0 || r.top > innerHeight) el.scrollIntoView({ block: 'nearest' });
    const s = el.getBoundingClientRect();
    return { x: s.x + s.width / 2, y: s.y + s.height / 2 };
  }
  return null;
}

/** What the page shows now (run in the page). */
function snapshot() {
  const box = (el) => {
    if (!el) return null;
    const r = el.getBoundingClientRect();
    return { x: r.x, y: r.y, width: r.width, height: r.height };
  };
  const visible = (el) => {
    if (!el) return false;
    const r = el.getBoundingClientRect();
    return r.width > 0 && r.height > 0 && getComputedStyle(el).visibility !== 'hidden';
  };
  const hook = window.__ajayeb;
  const bubble = document.querySelector('[data-coach][data-placed]');
  const notes = [...document.querySelectorAll('[data-coach-note]')].filter(visible);
  const targets = [...document.querySelectorAll('[data-coach-target]')];
  const target = targets[0] ?? null;
  const tb = box(target);
  let hit = null;
  if (tb) {
    const at = document.elementFromPoint(tb.x + tb.width / 2, tb.y + tb.height / 2);
    hit = !!at && (at === target || target.contains(at));
  }
  const buttons = bubble ? [...bubble.querySelectorAll('button')].map((b) => b.getBoundingClientRect().height) : [];
  const room = hook?.room();
  return {
    url: location.pathname,
    ready: hook?.ready() ?? false,
    version: room?.version ?? null,
    tutorial: hook?.tutorial?.() ?? null,
    coach: hook?.coach?.() ?? null,
    bubble: bubble && visible(bubble)
      ? {
          step: bubble.getAttribute('data-coach-step'),
          kind: bubble.getAttribute('data-coach-kind'),
          turn: Number(bubble.getAttribute('data-coach-turn')),
          placement: bubble.getAttribute('data-coach-placement'),
          box: box(bubble),
          buttons
        }
      : null,
    note: notes[0]
      ? { step: notes[0].getAttribute('data-coach-step'), kind: notes[0].getAttribute('data-coach-kind') }
      : null,
    targets: targets.length,
    target: tb ? { box: tb, region: target.hasAttribute('data-coach-region'), hit } : null,
    summary: !!document.querySelector('[data-move-summary]'),
    tip: document.querySelector('[data-first-tip]')?.getAttribute('data-first-tip') ?? null,
    done: visible(document.querySelector('[data-tutorial-done]')),
    starter: !!document.querySelector('[data-act^="starter:"]:not(:disabled)'),
    vw: document.documentElement.clientWidth,
    vh: innerHeight,
    hscroll: document.documentElement.scrollWidth > innerWidth,
    phase: hook?.view()?.phase ?? null
  };
}

/** The first move the free part makes (run in the page): a selector to press. */
function freeMove() {
  const enabled = (el) => !el.disabled && el.getBoundingClientRect().width > 0;
  const sheetAct = [...document.querySelectorAll('dialog[open] [data-act]')].filter(enabled);
  const build = sheetAct.find((b) => b.dataset.act.startsWith('construct:'));
  const discard = sheetAct.find((b) => b.dataset.act.startsWith('discard:'));
  const pick = build ?? discard;
  if (pick) return `[data-act="${pick.dataset.act}"]`;
  if (document.querySelector('dialog[open]')) return null;
  const slot = document.querySelector('[data-tut="structure"] [data-card][data-accessible]');
  if (slot) return `[data-tut="structure"] [data-card="${slot.dataset.card}"]`;
  const any = [...document.querySelectorAll('[data-act]')].find(enabled);
  return any ? `[data-act="${any.dataset.act}"]` : null;
}

/** Whether the element under a point is (inside) a match of the selector (run in the page). */
function pointIsOn({ x, y, selector }) {
  const el = document.elementFromPoint(x, y);
  return !!el && [...document.querySelectorAll(selector)].some((m) => m === el || m.contains(el));
}

/** The element under a point, as a short description (run in the page). */
function describeAt({ x, y }) {
  const el = document.elementFromPoint(x, y);
  if (!el) return null;
  const attrs = ['data-act', 'data-control', 'data-card', 'data-pick', 'data-coach-next']
    .filter((a) => el.closest(`[${a}]`))
    .map((a) => `${a}=${el.closest(`[${a}]`).getAttribute(a)}`);
  return `${el.tagName.toLowerCase()}${el.id ? '#' + el.id : ''} ${attrs.join(' ')}`.trim();
}

function overlap(a, b) {
  const w = Math.min(a.x + a.width, b.x + b.width) - Math.max(a.x, b.x);
  const h = Math.min(a.y + a.height, b.y + b.height) - Math.max(a.y, b.y);
  return w > 0 && h > 0 ? w * h : 0;
}

/** The checks of one coach step; returns the failures. */
function check(s) {
  const fail = [];
  if (s.hscroll) fail.push('the page scrolls sideways');
  if (s.targets > 1) fail.push(`${s.targets} elements carry data-coach-target`);
  const t = s.target;
  if (t) {
    const b = t.box;
    const outside = b.x < -1 || b.y < -1 || b.x + b.width > s.vw + 1 || b.y + b.height > s.vh + 1;
    // A region may be taller than a short screen: it is enough that most of
    // what can fit is on screen.
    const shownHeight = Math.min(b.y + b.height, s.vh) - Math.max(b.y, 0);
    const regionShown = t.region && shownHeight >= 0.85 * Math.min(b.height, s.vh);
    if (outside && !regionShown) fail.push('the target is not inside the viewport');
    if (!t.hit && !t.region) fail.push('the target is covered at its centre');
  }
  if (s.bubble) {
    const b = s.bubble.box;
    if (b.x < -1 || b.y < -1 || b.x + b.width > s.vw + 1 || b.y + b.height > s.vh + 1) {
      fail.push('the bubble is not inside the viewport');
    }
    if (t) {
      const share = overlap(b, t.box) / Math.max(1, t.box.width * t.box.height);
      if (!t.region && share > 0) fail.push('the bubble overlaps its target');
      if (t.region && share > 0.4) fail.push(`the bubble covers ${Math.round(share * 100)}% of its region`);
    }
    for (const h of s.bubble.buttons) if (h < 44 - 0.5) fail.push(`a bubble button is ${h.toFixed(1)} px tall`);
  }
  return fail;
}

export async function walk(opts) {
  const vp = vpOf(opts.vp);
  fs.mkdirSync(opts.out, { recursive: true });
  const script = await (await fetch(new URL('/api/dev/tutorial', opts.origin))).json();
  if (!script.ok) throw new Error('GET /api/dev/tutorial failed: is the server running with --dev-endpoints?');
  const expected = [];
  for (const e of script.entries) {
    if (e.seat !== script.human) continue;
    for (const l of e.lessons) expected.push(l.id);
    if (e.action) expected.push(e.action.id);
  }
  for (const l of script.outro) expected.push(l.id);

  const chrome = await launch({ force: opts.force });
  const browser = await CDP.connect(chrome.wsUrl);
  const report = { vp: opts.vp, theme: opts.theme, locale: opts.locale, animation: opts.animation, steps: [], errors: [], expected, observed: [], tips: [], trace: [], retries: [] };
  let failed = null;
  try {
    const page = await openPage(browser);
    page.on('Runtime.exceptionThrown', (p) =>
      report.errors.push(p.exceptionDetails.exception?.description ?? p.exceptionDetails.text)
    );
    page.on('Runtime.consoleAPICalled', (p) => {
      if (p.type === 'error') report.errors.push(p.args.map((a) => a.value ?? a.description).join(' '));
    });
    await Promise.all(['Page', 'Runtime'].map((d) => page.send(`${d}.enable`)));
    await page.send('Emulation.setDeviceMetricsOverride', {
      width: vp.width,
      height: vp.height,
      deviceScaleFactor: vp.dpr,
      mobile: vp.touch
    });
    await page.send('Emulation.setTouchEmulationEnabled', { enabled: vp.touch, maxTouchPoints: 5 });
    const storage = {
      'ajayeb:token': newToken(),
      'ajayeb:name': 'Walker',
      'ajayeb:theme': opts.theme,
      'ajayeb:settings': JSON.stringify({
        animationSpeed: opts.animation,
        showOpponentMove: opts.summary,
        locale: opts.locale
      }),
      'ajayeb:tips': JSON.stringify({ off: false, seen: [], tutorial: { offer: 'new' } })
    };
    await page.send('Page.addScriptToEvaluateOnNewDocument', {
      source: `(() => { try { if (sessionStorage.getItem('walk-init')) return; sessionStorage.setItem('walk-init', '1');
        for (const [k, v] of Object.entries(${JSON.stringify(storage)})) localStorage.setItem(k, v); } catch (e) {} })();`
    });
    await navigate(page, new URL('/', opts.origin).href);

    const shoot = async (name) => {
      if (opts.noShots) return;
      const { data } = await page.send('Page.captureScreenshot', { format: 'png' });
      fs.writeFileSync(path.join(opts.out, `${name}.png`), Buffer.from(data, 'base64'));
    };
    const pressSel = async (selector) => {
      // The point must still be on the element (a scroll into view may move
      // it after the centre was read), else look again.
      let c = null;
      for (let i = 0; i < 6; i++) {
        c = await evaluate(page, centreOf, selector);
        if (c && (await evaluate(page, pointIsOn, { ...c, selector }))) break;
        if (c) report.retries.push({ selector, x: Math.round(c.x), y: Math.round(c.y) });
        await sleep(100);
      }
      if (!c) throw new Error(`nothing to press for ${selector}`);
      const at = opts.trace ? await evaluate(page, describeAt, c) : null;
      await press(page, c.x, c.y, vp.touch);
      // A person's pace: never a second tap within the double-tap window.
      await sleep(150);
      if (opts.trace) {
        const after = await evaluate(page, () => ({
          dialog: !!document.querySelector('dialog[open]'),
          coach: window.__ajayeb?.coach?.() ?? null
        }));
        report.trace.push({ selector, x: Math.round(c.x), y: Math.round(c.y), at, after });
      }
    };

    await until(page, () => !!document.querySelector('[data-tutorial-offer]'), null, 15_000, 'the Home offer');
    await shoot('00-home-offer');
    await pressSel('[data-control="tutorial-start"]');
    await until(page, () => location.pathname.startsWith('/play/'), null, 15_000, 'the tutorial room');

    let n = 0;
    let actions = 0;
    let lastVersion = null;
    let versionAt = Date.now();
    let lastStep = null;
    let lastAction = null;
    for (;;) {
      const s = await evaluate(page, snapshot);
      if (s.version !== lastVersion) {
        lastVersion = s.version;
        versionAt = Date.now();
      }
      if (s.done) {
        await shoot(`${String(++n).padStart(2, '0')}-done`);
        report.done = true;
        // New game: Home with the vs-AI sheet open.
        report.newGame = { before: await evaluate(page, () => document.querySelectorAll('[data-dismiss-layer]').length) };
        await pressSel('[data-control="tutorial-new"]');
        await sleep(1000);
        report.newGame.url = await evaluate(page, () => location.pathname + location.search);
        report.newGame.aiSheet = await evaluate(page, () => !!document.querySelector('[data-control="ai-tutorial"]'));
        report.newGame.layers = await evaluate(page, () => [...document.querySelectorAll('[data-dismiss-layer]')].map((e) => e.outerHTML.slice(0, 120)));
        await shoot(`${String(++n).padStart(2, '0')}-new-game`);
        if (report.newGame.url !== '/' || !report.newGame.aiSheet) failed = 'New game did not open the vs-AI sheet on Home';
        break;
      }
      if (actions > opts.maxActions) throw new Error(`no completion card after ${actions} moves`);
      if (s.summary) {
        // An empty part of the top bar closes the summary.
        const c = await evaluate(page, () => {
          const r = document.querySelector('.top-bar')?.getBoundingClientRect();
          if (!r) return null;
          // A point on nothing that acts or has a tooltip (a tap there would
          // open the tooltip instead), scanning out from the middle.
          const y = r.y + r.height / 2;
          const busy = '[data-tooltip], button, a, input, [data-act]';
          for (let d = 0; d < r.width / 2; d += 6) {
            for (const x of [r.x + r.width / 2 + d, r.x + r.width / 2 - d]) {
              const el = document.elementFromPoint(x, y);
              if (el && el.closest('.top-bar') && !el.closest(busy)) return { x, y };
            }
          }
          return null;
        });
        if (c) await press(page, c.x, c.y, vp.touch);
        else {
          // No empty point: Escape closes the summary too.
          await page.send('Input.dispatchKeyEvent', { type: 'keyDown', key: 'Escape', code: 'Escape', windowsVirtualKeyCode: 27 });
          await page.send('Input.dispatchKeyEvent', { type: 'keyUp', key: 'Escape', code: 'Escape', windowsVirtualKeyCode: 27 });
        }
        await sleep(150);
        continue;
      }
      if (s.tip) {
        report.tips.push(s.tip);
        await pressSel('[data-tip-ok]');
        continue;
      }
      const shown = s.bubble ?? (s.note ? { ...s.note, placement: 'note' } : null);
      if (shown) {
        const key = `${s.version}:${shown.step}:${shown.kind}:${s.note ? 'note' : 'bubble'}`;
        if (key !== lastStep) {
          lastStep = key;
          // Check once the boxes are still: a sheet may still be sliding in,
          // and a note inside it moves with it (two reads 50 ms apart agree,
          // for at most 2 s).
          await sleep(50);
          let settled = await evaluate(page, snapshot);
          for (const until = Date.now() + 2000; Date.now() < until; ) {
            await sleep(50);
            const next = await evaluate(page, snapshot);
            const same = JSON.stringify([next.target?.box, next.bubble?.box, next.note?.step]) ===
              JSON.stringify([settled.target?.box, settled.bubble?.box, settled.note?.step]);
            settled = next;
            if (same) break;
          }
          const fails = check(settled);
          const name = `${String(++n).padStart(2, '0')}-${shown.step}-${shown.kind}${s.note ? '-note' : ''}`;
          await shoot(name);
          const entry = {
            n,
            step: shown.step,
            kind: shown.kind,
            surface: s.note ? 'note' : 'bubble',
            placement: settled.bubble?.placement ?? (s.note ? 'note' : null),
            target: settled.target,
            bubble: settled.bubble?.box ?? null,
            occluded: settled.target ? !settled.target.hit : null,
            msFromFrame: Date.now() - versionAt,
            failures: fails,
            shot: opts.noShots ? null : `${name}.png`
          };
          report.steps.push(entry);
          // The lesson order: each info lesson, and each action once per turn.
          if (shown.kind === 'info') report.observed.push(shown.step);
          else if (lastAction !== `${s.version}:${shown.step}`) {
            lastAction = `${s.version}:${shown.step}`;
            report.observed.push(shown.step);
          }
          if (fails.length > 0) {
            failed = `step ${n} (${shown.step}, ${shown.kind}): ${fails.join('; ')}`;
            break;
          }
        }
        if (shown.kind === 'info') {
          await pressSel('[data-coach-next]');
        } else if (shown.kind === 'note' || !s.target) {
          // A note with no target: close the sheet (Escape) and look again.
          await page.send('Input.dispatchKeyEvent', { type: 'keyDown', key: 'Escape', code: 'Escape', windowsVirtualKeyCode: 27 });
          await page.send('Input.dispatchKeyEvent', { type: 'keyUp', key: 'Escape', code: 'Escape', windowsVirtualKeyCode: 27 });
        } else {
          await pressSel('[data-coach-target]');
          if (shown.kind === 'act') {
            actions++;
            // The move's frame: wait for it, so the next look is not the
            // frame the move was made from.
            const from = s.version;
            const end = Date.now() + 3000;
            while (Date.now() < end && (await evaluate(page, () => window.__ajayeb?.room()?.version ?? null)) === from) {
              await sleep(50);
            }
          }
        }
        await sleep(120);
        continue;
      }
      if (!s.ready) {
        await sleep(80);
        continue;
      }
      const tut = s.tutorial;
      if (tut && tut.status === 'guided' && tut.action) {
        // A guided turn whose step has not shown yet: give it a moment.
        if (Date.now() - versionAt > 8000) throw new Error(`no coach step for ${tut.action.key}`);
        await sleep(80);
        continue;
      }
      // A short grace for a frame's info lesson (the outro lessons stay
      // listed once acknowledged, so their presence alone proves nothing).
      if (tut && tut.status !== 'complete' && Date.now() - versionAt < 600) {
        await sleep(80);
        continue;
      }
      if (s.starter) {
        await pressSel('[data-act="starter:0"]');
        actions++;
        await sleep(200);
        continue;
      }
      // The free part: a normal game against the Easy bot.
      const sel = await evaluate(page, freeMove);
      if (sel) {
        await pressSel(sel);
        if (sel.startsWith('[data-act')) actions++;
      }
      await sleep(200);
    }
    if (!failed) {
      const exp = expected.join(' ');
      const obs = report.observed.slice(0, expected.length).join(' ');
      if (exp !== obs) failed = `the lesson order differs: expected ${exp}, observed ${obs}`;
    }
    if (!failed && report.errors.length > 0) failed = `console errors: ${report.errors.join(' | ')}`;
  } catch (err) {
    failed = String(err?.message ?? err);
  } finally {
    report.failed = failed;
    fs.writeFileSync(path.join(opts.out, 'report.json'), JSON.stringify(report, null, 2));
    browser.close();
    await chrome.close();
  }
  return report;
}

if (isMain(import.meta.url)) {
  const { values } = parseArgs({
    options: {
      origin: { type: 'string', default: process.env.AJAYEB_ORIGIN ?? 'http://127.0.0.1:38951' },
      vp: { type: 'string', default: '390x844@2' },
      theme: { type: 'string', default: 'light' },
      locale: { type: 'string', default: 'en' },
      animation: { type: 'string', default: 'off' },
      summary: { type: 'string', default: 'on' },
      out: { type: 'string' },
      'max-actions': { type: 'string', default: '120' },
      'no-shots': { type: 'boolean', default: false },
      trace: { type: 'boolean', default: false },
      force: { type: 'boolean', default: false },
      help: { type: 'boolean', default: false }
    }
  });
  if (values.help || !values.out) {
    process.stdout.write(USAGE);
    process.exit(values.help ? 0 : 2);
  }
  walk({
    origin: values.origin,
    vp: values.vp,
    theme: values.theme,
    locale: values.locale,
    animation: values.animation,
    summary: values.summary !== 'off',
    out: values.out,
    maxActions: Number(values['max-actions']),
    noShots: values['no-shots'],
    trace: values.trace,
    force: values.force
  })
    .then((report) => {
      const bad = report.steps.filter((s) => s.failures.length > 0).length;
      process.stderr.write(
        `tutorial-walk: ${report.steps.length} coach steps, ${bad} failed, done=${!!report.done}${report.failed ? `\nFAILED: ${report.failed}` : ''}\n`
      );
      process.exit(report.failed ? 1 : 0);
    })
    .catch(reportAndExit);
}
