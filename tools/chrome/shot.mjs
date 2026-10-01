#!/usr/bin/env node
// Screenshot and DOM probe for one page of the running app, over CDP.
//
//   python3 tools/heavy.py run --kind browser -- node tools/chrome/shot.mjs \
//     --url /play/ABCD --token tok-a --name Ana --vp 390x844@3 --theme dark --probe --out x.png
//
// Flags (`--flag value` or `--flag=value`):
//   --origin URL     app origin (default $AJAYEB_ORIGIN or http://127.0.0.1:38951)
//   --url PATH       page to open, relative to --origin (default /); a full URL also works
//   --token T        written to localStorage ajayeb:token before the page loads
//   --name N         written to localStorage ajayeb:name
//   --theme T        light | dark | system, written to localStorage ajayeb:theme
//   --scheme S       light | dark: the emulated OS colour scheme (what "system" follows)
//   --vp V           WxH[@dpr] or a preset: mobile (390x844@2), desktop (1280x800@1);
//                    default 390x844@2. A short side under 600 px also emulates a phone
//                    (mobile viewport and touch).
//   --motion M       reduce (default) | no-preference: prefers-reduced-motion
//   --wait-ready     wait for [data-ready] on <html>, at most 2.5 s (else a fixed settle)
//   --settle MS      the fixed settle without --wait-ready (default 800)
//   --probe          print a JSON report on stdout (see README.md)
//   --out PATH       write a PNG of the viewport; --full captures the whole page
//   --force          launch Chrome without the heavy.py browser lock
//
// Identity works as in the app: the token, name and theme go into
// localStorage on the page's origin before any of its scripts run, so the
// app starts as that player. Each run uses a fresh browser context, so
// nothing carries over between runs.

import fs from 'node:fs';
import path from 'node:path';
import { parseArgs } from 'node:util';
import { CDP, evaluate, openPage, sleep } from './cdp.mjs';
import { isMain, launch, printHeader, reportAndExit } from './launch.mjs';

const PRESETS = { mobile: '390x844@2', desktop: '1280x800@1' };
const THEMES = ['light', 'dark', 'system'];

export function parseViewport(spec) {
  const m = /^(\d+)x(\d+)(?:@(\d+(?:\.\d+)?))?$/.exec(PRESETS[spec] ?? spec);
  if (!m) throw new Error(`--vp: expected WxH[@dpr], mobile or desktop, got ${spec}`);
  const width = Number(m[1]);
  const height = Number(m[2]);
  const dpr = Number(m[3] ?? 1);
  if (width < 1 || height < 1 || dpr <= 0) throw new Error(`--vp: bad size ${spec}`);
  return { width, height, dpr, mobile: Math.min(width, height) < 600 };
}

/** Navigates and waits for the load event of a cross-document navigation. */
async function navigate(page, url, timeoutMs = 15_000) {
  const loaded = page.waitFor('Page.loadEventFired', { timeoutMs });
  loaded.catch(() => {}); // a same-document navigation never fires it
  const r = await page.send('Page.navigate', { url });
  if (r.errorText) throw new Error(`navigate ${url}: ${r.errorText}`);
  if (r.loaderId) await loaded;
}

/** Runs in the page: the probe report. Keep it self-contained. */
function probePage() {
  const de = document.documentElement;
  const iw = window.innerWidth;
  const ih = window.innerHeight;
  const visible = (el) =>
    el.checkVisibility
      ? el.checkVisibility({ checkOpacity: true, checkVisibilityCSS: true })
      : true;
  const round = (n) => Math.round(n * 10) / 10;
  const describe = (el) => {
    let s = el.tagName.toLowerCase();
    if (el.id) s += `#${el.id}`;
    const cls = [...el.classList].slice(0, 2);
    if (cls.length) s += `.${cls.join('.')}`;
    for (const a of ['data-act', 'data-card', 'data-testid', 'aria-label', 'name']) {
      const v = el.getAttribute(a);
      if (v) {
        s += `[${a}="${v.slice(0, 40)}"]`;
        break;
      }
    }
    const text = (el.innerText || el.value || '').trim().replace(/\s+/g, ' ').slice(0, 30);
    return text ? `${s} "${text}"` : s;
  };
  const outside = (r) => r.right > iw + 0.5 || r.left < -0.5;
  // An overflowing element inside an on-screen ancestor that clips or
  // scrolls horizontally does not widen the page.
  const clipped = (el) => {
    for (let p = el.parentElement; p && p !== document.body; p = p.parentElement) {
      if (getComputedStyle(p).overflowX !== 'visible' && !outside(p.getBoundingClientRect()))
        return true;
    }
    return false;
  };

  const overflow = [];
  const reported = [];
  for (const el of document.body.querySelectorAll('*')) {
    if (el.closest('[inert]') || !visible(el)) continue;
    const r = el.getBoundingClientRect();
    if ((r.width === 0 && r.height === 0) || !outside(r)) continue;
    if (reported.some((p) => p.contains(el)) || clipped(el)) continue;
    reported.push(el);
    overflow.push({
      el: describe(el),
      left: round(r.left),
      right: round(r.right),
      width: round(r.width)
    });
    if (overflow.length >= 15) break;
  }

  const tapSelector = [
    'button',
    'a[href]',
    'input:not([type="hidden"])',
    'select',
    'textarea',
    'summary',
    '[role="button"]',
    '[role="link"]',
    '[role="tab"]',
    '[role="radio"]',
    '[role="checkbox"]',
    '[tabindex]:not([tabindex="-1"])',
    '[data-act]'
  ].join(',');
  let smallest = null;
  let count = 0;
  let under24 = 0;
  let under44 = 0;
  for (const el of document.querySelectorAll(tapSelector)) {
    if (el.matches(':disabled, [aria-disabled="true"]') || el.closest('[inert]')) continue;
    // A visually hidden input is hit through its label.
    let target = el;
    if (el.labels && el.labels.length) target = [...el.labels].find(visible) ?? el;
    if (!visible(target)) continue;
    const r = target.getBoundingClientRect();
    if (r.width <= 1 && r.height <= 1) continue; // screen-reader-only
    count++;
    const min = Math.min(r.width, r.height);
    if (min < 24) under24++;
    if (min < 44) under44++;
    if (!smallest || min < smallest.min) {
      smallest = { el: describe(target), w: round(r.width), h: round(r.height), min: round(min) };
    }
  }

  return {
    url: location.href,
    title: document.title,
    iw,
    ih,
    sw: de.scrollWidth,
    sh: de.scrollHeight,
    hscroll: de.scrollWidth > iw,
    vscroll: de.scrollHeight > ih,
    theme: de.dataset.theme ?? null,
    ready: de.hasAttribute('data-ready'),
    // The app's own keys; a token is shown only by its length.
    storage: (() => {
      try {
        const keys = Object.keys(localStorage).filter((k) => k.startsWith('ajayeb:'));
        return Object.fromEntries(
          keys.sort().map((k) => {
            const v = localStorage.getItem(k) ?? '';
            return [k, k === 'ajayeb:token' ? `<${v.length} chars>` : v];
          })
        );
      } catch {
        return null;
      }
    })(),
    body: (document.body?.innerText ?? '').slice(0, 400),
    overflow,
    tap: { count, smallest, under24, under44 }
  };
}

async function main() {
  const { values: o } = parseArgs({
    options: {
      origin: { type: 'string', default: process.env.AJAYEB_ORIGIN ?? 'http://127.0.0.1:38951' },
      url: { type: 'string', default: '/' },
      token: { type: 'string' },
      name: { type: 'string' },
      theme: { type: 'string' },
      scheme: { type: 'string' },
      vp: { type: 'string', default: '390x844@2' },
      motion: { type: 'string', default: 'reduce' },
      'wait-ready': { type: 'boolean', default: false },
      settle: { type: 'string', default: '800' },
      probe: { type: 'boolean', default: false },
      out: { type: 'string' },
      full: { type: 'boolean', default: false },
      force: { type: 'boolean', default: false },
      help: { type: 'boolean', short: 'h', default: false }
    }
  });
  if (o.help) {
    printHeader(import.meta.url);
    return;
  }
  if (!o.probe && !o.out) throw new Error('nothing to do: pass --out PATH, --probe, or both');
  if (o.theme && !THEMES.includes(o.theme)) throw new Error(`--theme: one of ${THEMES.join(', ')}`);
  if (o.scheme && !['light', 'dark'].includes(o.scheme)) throw new Error('--scheme: light or dark');
  if (!['reduce', 'no-preference'].includes(o.motion))
    throw new Error('--motion: reduce or no-preference');
  const vp = parseViewport(o.vp);
  const target = new URL(o.url, o.origin);

  const chrome = await launch({ force: o.force });
  let browser = null;
  let page = null;
  try {
    browser = await CDP.connect(chrome.wsUrl);
    page = await openPage(browser);

    // Hygiene, reported by --probe: console errors, exceptions, failed requests.
    const errors = [];
    const failed = [];
    const urls = new Map(); // requestId -> url, for loadingFailed
    page.on('Network.requestWillBeSent', (p) => urls.set(p.requestId, p.request.url));
    page.on('Runtime.consoleAPICalled', (p) => {
      if (p.type === 'error' || p.type === 'assert') {
        errors.push(
          `console.${p.type}: ${p.args.map((a) => a.value ?? a.description ?? '').join(' ')}`
        );
      }
    });
    page.on('Runtime.exceptionThrown', (p) => {
      errors.push(
        `uncaught: ${p.exceptionDetails.exception?.description ?? p.exceptionDetails.text}`
      );
    });
    page.on('Log.entryAdded', (p) => {
      if (p.entry.level === 'error')
        errors.push(`log: ${p.entry.text} ${p.entry.url ?? ''}`.trim());
    });
    page.on('Network.responseReceived', (p) => {
      if (p.response.status >= 400) failed.push(`HTTP ${p.response.status} ${p.response.url}`);
    });
    page.on('Network.loadingFailed', (p) => {
      if (!p.canceled) failed.push(`${p.errorText} ${urls.get(p.requestId) ?? p.type}`);
    });
    await Promise.all(['Page', 'Runtime', 'Log', 'Network'].map((d) => page.send(`${d}.enable`)));

    // Viewport through metrics emulation, never by resizing a window.
    await page.send('Emulation.setDeviceMetricsOverride', {
      width: vp.width,
      height: vp.height,
      deviceScaleFactor: vp.dpr,
      mobile: vp.mobile
    });
    await page.send('Emulation.setTouchEmulationEnabled', {
      enabled: vp.mobile,
      maxTouchPoints: 5
    });
    const features = [{ name: 'prefers-reduced-motion', value: o.motion }];
    if (o.scheme) features.push({ name: 'prefers-color-scheme', value: o.scheme });
    await page.send('Emulation.setEmulatedMedia', { features });

    const storage = [];
    if (o.token !== undefined) storage.push(['ajayeb:token', o.token]);
    if (o.name !== undefined) storage.push(['ajayeb:name', o.name]);
    if (o.theme !== undefined) storage.push(['ajayeb:theme', o.theme]);
    // The keys are written by a new-document script, which runs on the
    // target origin before any of the page's own scripts (the theme
    // bootstrap in app.html included). So the app boots once, already
    // holding this identity, and no extra page load adds noise to the probe.
    let storageScript = null;
    if (storage.length > 0) {
      const source =
        `(() => { if (location.origin !== ${JSON.stringify(target.origin)}) return;` +
        ` try { for (const [k, v] of ${JSON.stringify(storage)}) localStorage.setItem(k, v); }` +
        ` catch (e) { console.error('shot.mjs: cannot write localStorage: ' + e); } })();`;
      ({ identifier: storageScript } = await page.send('Page.addScriptToEvaluateOnNewDocument', {
        source
      }));
    }

    await navigate(page, target.href);
    if (storageScript) {
      await page.send('Page.removeScriptToEvaluateOnNewDocument', { identifier: storageScript });
    }
    if (o['wait-ready']) {
      const deadline = Date.now() + 2_500;
      while (Date.now() < deadline) {
        if (await evaluate(page, () => document.documentElement.hasAttribute('data-ready'))) break;
        await sleep(50);
      }
    } else {
      await sleep(Number(o.settle) || 0);
    }
    await evaluate(page, async () => {
      await document.fonts.ready;
      await new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)));
    });

    if (o.probe) {
      const report = await evaluate(page, probePage);
      report.vp = `${vp.width}x${vp.height}@${vp.dpr}`;
      report.errors = errors;
      report.failed = failed;
      console.log(JSON.stringify(report, null, 2));
    }

    if (o.out) {
      const params = { format: 'png', fromSurface: true };
      if (o.full) {
        const { cssContentSize: s } = await page.send('Page.getLayoutMetrics');
        params.captureBeyondViewport = true;
        params.clip = {
          x: 0,
          y: 0,
          width: Math.ceil(s.width),
          height: Math.ceil(s.height),
          scale: 1
        };
      }
      const { data } = await page.send('Page.captureScreenshot', params);
      fs.mkdirSync(path.dirname(path.resolve(o.out)), { recursive: true });
      fs.writeFileSync(o.out, Buffer.from(data, 'base64'));
      console.error(
        `saved ${o.out} (${vp.width}x${vp.height}@${vp.dpr}${o.full ? ', full page' : ''})`
      );
    }
  } finally {
    await page?.close();
    browser?.close();
    await chrome.close();
  }
}

if (isMain(import.meta.url)) main().catch(reportAndExit);
