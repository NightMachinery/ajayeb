// Helpers for animation.spec.ts (M10.animations-moves): turning the game's
// own Animation setting on for one tab, opening the scripted positions the
// spec plays from, and reading what the page's animation runner recorded
// (window.__ajayebFx.frames() and anims(), web/src/lib/fx/index.ts).
//
// The base fixture (base.ts) forces Animation off once per tab, under a
// sessionStorage guard. useSettings() loads one page first, so that script
// has run and set its guard, then writes this spec's settings; every later
// load in the same tab leaves them alone.
//
// Positions come from the server: a seed's draft is played over the socket
// (recordGame, stopped at the first move of Age I), and the named scenarios
// (server/internal/engine/scenarios_named.go) give the rest. Nothing here
// decides what is legal; every key comes from the view's choices.

import { mkdir, open } from 'node:fs/promises';
import { dirname } from 'node:path';
import { expect, type CDPSession, type Page, type TestInfo } from '@playwright/test';
import type { Choice, View } from '../../web/src/lib/wire';
import { GameDriver } from './driver';
import { defaultFlow } from './scenario';
import { createDevRoom, recordGame } from './ws-game';

export type Speed = 'normal' | 'fast' | 'relaxed' | 'off';

/** What the runner recorded for a frame (web/src/lib/fx/anim/records.ts). */
export interface FrameRecord {
  version: number;
  swapAt: number;
  releasedAt: number | null;
  steps: number;
}

/** What the runner recorded for an animation. */
export interface AnimRecord {
  version: number;
  kind:
    | 'take'
    | 'rebuild'
    | 'destroy'
    | 'token'
    | 'draft'
    | 'flip'
    | 'coins'
    | 'pawn'
    | 'loot'
    | 'glow'
    | 'draw'
    | 'deal'
    | 'victory';
  startAt: number;
  endAt: number | null;
  interrupted: boolean;
}

interface FxHook {
  cues(): { cue: string; delayMs: number; version: number; replay: boolean }[];
  frames(): FrameRecord[];
  anims(): AnimRecord[];
  clear(): void;
}

/** The settings every case here plays with, besides the speed. */
const BASE_SETTINGS = {
  hotseatPassDevice: false,
  showOpponentMove: false,
  confirmMoves: 'never'
};

/**
 * Sets this tab's Animation speed (and no pass-device screen, no move
 * summary, no confirm step), after the base fixture's own override ran.
 */
export async function useSettings(page: Page, speed: Speed): Promise<void> {
  await page.goto('/settings');
  await page.evaluate(
    ({ speed, base }) => {
      let current: Record<string, unknown> = {};
      try {
        current = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}') as Record<
          string,
          unknown
        >;
      } catch {
        current = {};
      }
      localStorage.setItem(
        'ajayeb:settings',
        JSON.stringify({ ...current, ...base, animationSpeed: speed })
      );
      localStorage.setItem('ajayeb:name', 'Anim');
    },
    { speed, base: BASE_SETTINGS }
  );
}

/** A position to open: a seeded game after some keys, or a named scenario. */
export interface Position {
  mode: 'hotseat' | 'ai';
  seed: string;
  scenario?: string;
  keys: string[];
}

class Stop extends Error {
  readonly key: string;
  constructor(key = '') {
    super('found');
    this.key = key;
  }
}

/**
 * Plays a seed's game over the socket from the start of Age I by `advance`
 * until `find` names a key, and returns the position there with that key.
 */
export async function findPosition(
  origin: string,
  seed: string,
  find: (view: View) => string | null,
  advance: (view: View) => Choice
): Promise<{ at: Position; key: string }> {
  const draft = await draftKeys(origin, seed);
  const keys: string[] = [];
  try {
    await recordGame(origin, seed, (view: View): Choice => {
      if (keys.length < draft.length) {
        const key = draft[keys.length]!;
        const c = view.choices.find((x) => x.key === key);
        if (!c) throw new Error(`draft key ${key} is not a choice`);
        keys.push(key);
        return c;
      }
      const key = find(view);
      if (key) throw new Stop(key);
      const c = advance(view);
      keys.push(c.key);
      return c;
    });
  } catch (err) {
    if (err instanceof Stop) return { at: { mode: 'hotseat', seed, keys }, key: err.key };
    throw err;
  }
  throw new Error(`seed ${seed}: no such position`);
}

const drafts = new Map<string, Promise<string[]>>();

/** The keys of a seed's wonder draft (the first offer each time), up to Age I. */
export function draftKeys(origin: string, seed: string): Promise<string[]> {
  let p = drafts.get(seed);
  if (!p) {
    p = (async () => {
      const keys: string[] = [];
      try {
        await recordGame(origin, seed, (view: View): Choice => {
          if (view.phase !== 'wonderDraft') throw new Stop();
          const c = view.choices[0];
          if (!c) throw new Error('a draft with no choice');
          keys.push(c.key);
          return c;
        });
      } catch (err) {
        if (!(err instanceof Stop)) throw err;
      }
      return keys;
    })();
    drafts.set(seed, p);
  }
  return p;
}

/**
 * Opens a position in the page (whose settings useSettings set) and waits
 * until it is ready. An ai room's bot is easy, its human seat 0.
 */
export async function openPosition(page: Page, origin: string, at: Position): Promise<GameDriver> {
  let path: string;
  if (at.mode === 'ai') {
    const res = await fetch(`${origin}/api/dev/rooms`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        mode: 'ai',
        bot: 'easy',
        seed: at.seed,
        ...(at.scenario ? { scenario: at.scenario } : {}),
        actions: at.keys
      })
    });
    const data = (await res.json()) as { ok?: boolean; path?: string };
    if (!data.ok || !data.path) throw new Error(`ai dev room: ${JSON.stringify(data)}`);
    path = data.path;
  } else {
    const room = await createDevRoom(origin, {
      mode: 'hotseat',
      seed: at.seed,
      ...(at.scenario ? { scenario: at.scenario } : {}),
      actions: at.keys
    });
    path = room.path;
  }
  await page.goto(path);
  const driver = new GameDriver(page);
  await driver.waitReady();
  return driver;
}

/** The view on screen. */
export async function viewOf(page: Page): Promise<View> {
  const view = await page.evaluate(() => window.__ajayeb?.view() ?? null);
  if (!view) throw new Error('no view on screen');
  return view;
}

/**
 * Plays one key through the UI the way a player would, and waits until
 * the room moved on and the page takes input again.
 */
export async function playThrough(driver: GameDriver, key: string): Promise<void> {
  const before = await driver.snapshot();
  const choice = before.choices.find((c) => c.key === key);
  if (!choice) throw new Error(`${key} is not a choice: ${before.choices.map((c) => c.key)}`);
  await driver.perform({ flow: defaultFlow(before, choice), choice }, before);
  await driver.waitAdvance(before.version);
}

/** Sound cues recorded by live frames and visual replays. */
export async function fxCues(page: Page): Promise<ReturnType<FxHook['cues']>> {
  return page.evaluate(
    () => (window as unknown as { __ajayebFx?: FxHook }).__ajayebFx?.cues() ?? []
  );
}

/** The runner's records since the last clearFx(). */
export async function fxRecords(
  page: Page
): Promise<{ frames: FrameRecord[]; anims: AnimRecord[] }> {
  return page.evaluate(() => {
    const fx = (window as unknown as { __ajayebFx?: FxHook }).__ajayebFx;
    return { frames: fx?.frames() ?? [], anims: fx?.anims() ?? [] };
  });
}

export async function clearFx(page: Page): Promise<void> {
  await page.evaluate(() => (window as unknown as { __ajayebFx?: FxHook }).__ajayebFx?.clear());
}

/**
 * Waits until every animation the runner started has ended and nothing it
 * drew is left: no clone, flip or float in the FX layer or its top part
 * (the one over dialogs), nothing hidden. Then two frames more: the last
 * thing to end may have changed the layout (a counter's text written back,
 * a sheet closed), and the structure refits to its box only in the next
 * frame's ResizeObserver callback, so reading the DOM at once can catch the
 * old fit.
 */
export async function waitSettled(page: Page, timeout = 10_000): Promise<void> {
  await page.waitForFunction(
    () => {
      const fx = (window as unknown as { __ajayebFx?: FxHook }).__ajayebFx;
      const running = (fx?.anims() ?? []).some((a) => a.endAt === null);
      return (
        window.__ajayeb?.ready() === true &&
        !running &&
        document.querySelectorAll('[data-fx-clone], .fx-flip, .fx-float, [data-fx-hidden]')
          .length === 0 &&
        (document.querySelector('.fx-top')?.childElementCount ?? 0) === 0
      );
    },
    null,
    { timeout, polling: 50 }
  );
  await page.evaluate(
    () =>
      new Promise<void>((resolve) => {
        requestAnimationFrame(() => requestAnimationFrame(() => resolve()));
      })
  );
}

/**
 * The game screen's DOM as text, for comparing two runs: the room code,
 * the FX attributes, transforms in style attributes, the ids tooltips
 * point at and whether an art image has loaded yet are taken out, because
 * they differ by run, not by state.
 */
export async function screenDom(page: Page): Promise<string> {
  const { html, code } = await page.evaluate(() => {
    const screen = document.querySelector('.game-screen');
    const layer = document.querySelector('.fx-layer');
    const top = document.querySelector('.fx-top');
    if (!screen) return { html: '', code: '' };
    const copy = screen.cloneNode(true) as HTMLElement;
    for (const el of [copy, ...copy.querySelectorAll('*')]) {
      // The anchors (data-fx-band and the like) are markup and stay.
      for (const name of ['data-fx-hidden', 'aria-describedby', 'id', 'for']) {
        el.removeAttribute(name);
      }
      // An image's own load (ArtImage's `loaded`) races the screenshot, not the FX.
      if (el.classList.contains('art-image')) el.classList.remove('loaded');
      const style = el.getAttribute('style');
      if (style !== null) {
        const kept = style
          .split(';')
          .map((s) => s.trim())
          .filter((s) => s && !/^(transform|translate|will-change)\s*:/.test(s));
        if (kept.length > 0) el.setAttribute('style', kept.join('; '));
        else el.removeAttribute('style');
      }
    }
    return {
      html:
        copy.outerHTML +
        `\n<!-- fx layer children: ${layer?.childElementCount ?? -1}, ` +
        `top: ${top?.childElementCount ?? -1} -->`,
      code: window.__ajayeb?.room()?.code ?? ''
    };
  });
  expect(html, 'the game screen is on the page').not.toBe('');
  // One tag per line, so a difference shows as the lines that differ.
  return (code ? html.split(code).join('ROOM') : html).replaceAll('><', '>\n<');
}

/**
 * Installs, before any app script, a recorder of every animation and
 * transition the page starts: Element.prototype.animate (WAAPI) and the
 * animationstart and transitionrun events (CSS). Each records its end time
 * (delay plus active duration) in window.__motionLog.
 */
export async function recordMotion(page: Page): Promise<void> {
  await page.addInitScript(() => {
    const log: { what: string; ms: number }[] = [];
    (window as unknown as { __motionLog: typeof log }).__motionLog = log;
    const endOf = (a: Animation) => {
      const t = a.effect?.getComputedTiming();
      return typeof t?.endTime === 'number' ? t.endTime : Number(t?.endTime ?? 0);
    };
    const original = Element.prototype.animate;
    Element.prototype.animate = function (this: Element, ...args: Parameters<Element['animate']>) {
      const anim = original.apply(this, args);
      log.push({ what: `waapi ${this.tagName.toLowerCase()}.${this.className}`, ms: endOf(anim) });
      return anim;
    };
    const css = (event: AnimationEvent | TransitionEvent) => {
      const target = event.target;
      if (!(target instanceof Element)) return;
      const name = 'animationName' in event ? event.animationName : event.propertyName;
      for (const anim of target.getAnimations()) {
        log.push({ what: `css ${name} on ${target.tagName.toLowerCase()}`, ms: endOf(anim) });
      }
    };
    document.addEventListener('animationstart', css, true);
    document.addEventListener('transitionrun', css, true);
  });
}

/** What recordMotion saw since the page loaded or the last reset. */
export async function motionLog(
  page: Page,
  reset = false
): Promise<{ what: string; ms: number }[]> {
  return page.evaluate((reset) => {
    const log = (window as unknown as { __motionLog?: { what: string; ms: number }[] }).__motionLog;
    const out = [...(log ?? [])];
    if (reset && log) log.length = 0;
    return out;
  }, reset);
}

/**
 * Installs, before any app script, a recorder of the first animation frame
 * at which the page takes input again after each version: the test hook
 * ready() (presenter idle, the latest state, nothing pending) and you()
 * canAct. window.__inputAt[version] is that frame's performance.now().
 */
export async function recordInputBack(page: Page): Promise<void> {
  await page.addInitScript(() => {
    const at: Record<number, number> = {};
    (window as unknown as { __inputAt: typeof at }).__inputAt = at;
    const tick = () => {
      const hook = window.__ajayeb;
      const version = hook?.room()?.version;
      if (hook && version !== undefined && at[version] === undefined) {
        if (hook.ready() && hook.you()?.canAct) at[version] = performance.now();
      }
      requestAnimationFrame(tick);
    };
    requestAnimationFrame(tick);
  });
}

export async function inputAt(page: Page): Promise<Record<number, number>> {
  return page.evaluate(
    () => (window as unknown as { __inputAt?: Record<number, number> }).__inputAt ?? {}
  );
}

/**
 * Installs, before any app script, a PerformanceObserver that keeps every
 * long task (over 50 ms) in window.__longTasks.
 */
export async function recordLongTasks(page: Page): Promise<void> {
  await page.addInitScript(() => {
    const tasks: { start: number; duration: number }[] = [];
    (window as unknown as { __longTasks: typeof tasks }).__longTasks = tasks;
    try {
      new PerformanceObserver((list) => {
        for (const e of list.getEntries()) tasks.push({ start: e.startTime, duration: e.duration });
      }).observe({ type: 'longtask', buffered: true });
    } catch {
      // No long task timing in this browser: the test says so.
    }
  });
}

/** A long task the page recorded (recordLongTasks). */
export interface LongTask {
  start: number;
  duration: number;
}

export async function longTasks(page: Page): Promise<LongTask[]> {
  return page.evaluate(
    () =>
      (window as unknown as { __longTasks?: { start: number; duration: number }[] }).__longTasks ??
      []
  );
}

/**
 * Installs, before any app script, a recorder of every animation frame's
 * timestamp (the requestAnimationFrame time) in window.__frames, so the gaps
 * between frames can be read against the animation windows: a gap over one
 * frame is a frame the page did not draw, which a player sees as stutter.
 */
export async function recordFrames(page: Page): Promise<void> {
  await page.addInitScript(() => {
    const frames: number[] = [];
    (window as unknown as { __frames: typeof frames }).__frames = frames;
    const tick = (t: number) => {
      frames.push(t);
      requestAnimationFrame(tick);
    };
    requestAnimationFrame(tick);
  });
}

export async function frameTimes(page: Page): Promise<number[]> {
  return page.evaluate(() => (window as unknown as { __frames?: number[] }).__frames ?? []);
}

/** The gaps between frames that overlap an animation window. */
export function animationFrameGaps(frames: number[], anims: AnimRecord[]): number[] {
  const gaps: number[] = [];
  for (let i = 1; i < frames.length; i++) {
    const from = frames[i - 1]!;
    const to = frames[i]!;
    if (anims.some((a) => a.endAt !== null && from < a.endAt && to > a.startAt)) {
      gaps.push(to - from);
    }
  }
  return gaps;
}

/** A one-line summary of frame gaps: how many, the share over 34 ms, the worst. */
export function describeFrameGaps(gaps: number[]): string {
  const over = gaps.filter((g) => g > 34).length;
  const share = gaps.length === 0 ? 0 : Math.round((over / gaps.length) * 1000) / 10;
  return (
    `frames during animations ${gaps.length}, over 34 ms ${over} (${share}%), ` +
    `worst gap ${Math.round(Math.max(0, ...gaps))} ms`
  );
}

/**
 * With AJAYEB_PERF_TRACE=1, records a Chrome performance trace (timeline and
 * CPU samples) until the returned function runs, and attaches it to the
 * test as chrome-trace.json: a diagnostic for finding what runs during an
 * animation, never an acceptance measure (profiling slows the page). Off,
 * it does nothing.
 */
export async function traceIfAsked(cdp: CDPSession, info: TestInfo): Promise<() => Promise<void>> {
  if (process.env.AJAYEB_PERF_TRACE !== '1') return async () => {};
  await cdp.send('Tracing.start', {
    categories:
      'toplevel,devtools.timeline,disabled-by-default-devtools.timeline,' +
      'disabled-by-default-devtools.timeline.frame,disabled-by-default-v8.cpu_profiler,' +
      'v8,blink.user_timing',
    options: 'sampling-frequency=1000',
    transferMode: 'ReturnAsStream'
  });
  return async () => {
    const completed = new Promise<{ stream?: string }>((resolve) =>
      cdp.once('Tracing.tracingComplete', resolve)
    );
    await cdp.send('Tracing.end');
    const { stream } = await completed;
    if (!stream) throw new Error('Chrome returned no trace stream');
    const path = info.outputPath('chrome-trace.json');
    await mkdir(dirname(path), { recursive: true });
    const file = await open(path, 'w');
    try {
      for (;;) {
        const chunk = await cdp.send('IO.read', { handle: stream });
        await file.writeFile(chunk.base64Encoded ? Buffer.from(chunk.data, 'base64') : chunk.data);
        if (chunk.eof) break;
      }
    } finally {
      await file.close();
      await cdp.send('IO.close', { handle: stream });
    }
    await info.attach('chrome trace', { path, contentType: 'application/json' });
  };
}
