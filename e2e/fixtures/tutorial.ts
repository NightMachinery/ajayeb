// The tutorial's shared walk (M10.tutorial-client), for tutorial.spec.ts and
// tutorial-viewports.spec.ts. It reads only the DOM contract of the tutorial
// design (section 7) and the test hook's tutorial() and coach():
//
// - the bubble: [data-coach] with data-coach-step, data-coach-kind,
//   data-coach-turn and data-coach-placement, its buttons [data-coach-next]
//   (info lessons) and [data-coach-pause];
// - the note inside a sheet: [data-coach-note] with data-coach-step and
//   data-coach-kind;
// - exactly one [data-coach-target] (with data-coach-region on a region);
// - [data-tutorial-locked] on a move the rails refuse;
// - Home's [data-tutorial-offer], [data-control="tutorial-start"] and
//   [data-control="tutorial-decline"], and the completion card's
//   [data-tutorial-done] with its three buttons.
//
// The walk policy: while a coach step shows, press Next on an info lesson,
// otherwise press its target (a note with no target closes its sheet with
// Escape); while the move summary is open, close it with a tap on an empty
// part of the top bar; a first-time tip is closed with Got it, and its id
// recorded; in the free part, open the first accessible card and press its
// first enabled move (build, else discard), else the first enabled move on
// the page; answer a starter choice with starter:0; stop at the completion
// card (or where the caller says). tools/chrome/tutorial-walk.mjs is the
// same walk without Playwright, for the player reviewer.

import { expect, type Page } from '@playwright/test';
import type { TutorialState } from '../../web/src/lib/wire';
import { createDevRoom } from './ws-game';

/** The script as GET /api/dev/tutorial answers it (package tutorial). */
export interface TutorialScript {
  name: string;
  human: number;
  entries: {
    seat: number;
    key: string;
    lessons: { id: string }[];
    action?: { id: string; key: string };
  }[];
  outro: { id: string }[];
}

export interface Script {
  raw: TutorialScript;
  /** Each human entry's info lessons and action id, then the outro. */
  order: string[];
  /** The guided keys, in order. */
  guided: string[];
  /** Every key of the script, both seats. */
  keys: string[];
}

export async function tutorialScript(origin: string): Promise<Script> {
  const res = await fetch(`${origin}/api/dev/tutorial`);
  const raw = (await res.json()) as TutorialScript & { ok: boolean };
  if (!raw.ok) throw new Error('GET /api/dev/tutorial failed');
  const order: string[] = [];
  const guided: string[] = [];
  for (const e of raw.entries) {
    if (e.seat !== raw.human) continue;
    for (const l of e.lessons) order.push(l.id);
    if (e.action) {
      order.push(e.action.id);
      guided.push(e.action.key);
    }
  }
  for (const l of raw.outro) order.push(l.id);
  return { raw, order, guided, keys: raw.entries.map((e) => e.key) };
}

/** The device state a first-time player has, written once per tab (a reload keeps what the page saved). */
export async function freshDevice(
  page: Page,
  opts: {
    tips?: boolean;
    /** Tips already seen on this device. */
    seen?: string[];
    summary?: boolean;
    offer?: 'new' | 'declined';
    speed?: string;
  } = {}
): Promise<void> {
  await page.addInitScript(
    ({ tips, seen, summary, offer, speed }) => {
      try {
        if (sessionStorage.getItem('ajayeb:e2e-tutorial-device')) return;
        sessionStorage.setItem('ajayeb:e2e-tutorial-device', '1');
        let settings: Record<string, unknown> = {};
        try {
          settings = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}') as Record<
            string,
            unknown
          >;
        } catch {
          settings = {};
        }
        localStorage.setItem(
          'ajayeb:settings',
          JSON.stringify({ ...settings, showOpponentMove: summary, animationSpeed: speed })
        );
        localStorage.setItem(
          'ajayeb:tips',
          JSON.stringify({ off: !tips, seen, tutorial: { offer } })
        );
        localStorage.setItem('ajayeb:name', 'Learner');
      } catch {
        // Storage may be blocked; nothing to set.
      }
    },
    {
      tips: opts.tips ?? true,
      seen: opts.seen ?? [],
      summary: opts.summary ?? true,
      offer: opts.offer ?? 'new',
      speed: opts.speed ?? 'off'
    }
  );
}

/** A tutorial room at a position of the script (POST /api/dev/rooms), opened in the page. */
export async function openTutorialRoom(
  page: Page,
  origin: string,
  actions: string[] = []
): Promise<string> {
  const res = await fetch(`${origin}/api/dev/rooms`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ mode: 'ai', bot: 'easy', tutorial: true, actions })
  });
  const data = (await res.json()) as { ok?: boolean; code?: string; path?: string };
  if (!data.ok || !data.code) throw new Error(`POST /api/dev/rooms: ${JSON.stringify(data)}`);
  await page.goto(data.path ?? `/play/${data.code}`);
  await page.waitForFunction(() => window.__ajayeb?.ready() === true);
  return data.code;
}

// createDevRoom is re-exported for a non-tutorial room in the same spec.
export { createDevRoom };

/** Waits until the page is ready and its tutorial has left the script (or not). */
export async function waitLeft(page: Page, left: boolean): Promise<void> {
  await page.waitForFunction((left) => {
    const hook = (
      window as unknown as {
        __ajayeb?: { ready(): boolean; tutorial(): TutorialState | null };
      }
    ).__ajayeb;
    return hook?.ready() === true && hook.tutorial()?.left === left;
  }, left);
}

/** What the page shows now (run in the page). */
export interface CoachSnapshot {
  version: number | null;
  ready: boolean;
  tutorial: TutorialState | null;
  bubble: {
    step: string;
    kind: string;
    turn: number;
    placement: string;
    box: Box;
    buttons: number[];
    arrow: Box | null;
  } | null;
  note: { step: string; kind: string } | null;
  targets: number;
  target: { box: Box; region: boolean; hit: boolean } | null;
  summary: boolean;
  tip: string | null;
  done: boolean;
  starter: boolean;
  vw: number;
  vh: number;
  hscroll: boolean;
}

export interface Box {
  x: number;
  y: number;
  width: number;
  height: number;
}

export function snapshot(page: Page): Promise<CoachSnapshot> {
  return page.evaluate(() => {
    const box = (el: Element | null) => {
      if (!el) return null;
      const r = el.getBoundingClientRect();
      return { x: r.x, y: r.y, width: r.width, height: r.height };
    };
    const visible = (el: Element | null) => {
      if (!el) return false;
      const r = el.getBoundingClientRect();
      return r.width > 0 && r.height > 0 && getComputedStyle(el).visibility !== 'hidden';
    };
    const hook = (
      window as unknown as {
        __ajayeb?: {
          ready(): boolean;
          room(): { version: number } | null;
          tutorial(): TutorialState | null;
        };
      }
    ).__ajayeb;
    const bubble = document.querySelector('[data-coach][data-placed]');
    const note = [...document.querySelectorAll('[data-coach-note]')].find(visible) ?? null;
    const targets = [...document.querySelectorAll('[data-coach-target]')];
    const target = targets[0] ?? null;
    const tb = box(target);
    let hit = false;
    if (target && tb) {
      const at = document.elementFromPoint(tb.x + tb.width / 2, tb.y + tb.height / 2);
      hit = !!at && (at === target || target.contains(at));
    }
    const arrow = bubble?.querySelector('[data-coach-arrow]') ?? null;
    return {
      version: hook?.room()?.version ?? null,
      ready: hook?.ready() ?? false,
      tutorial: hook?.tutorial() ?? null,
      bubble:
        bubble && visible(bubble)
          ? {
              step: bubble.getAttribute('data-coach-step') ?? '',
              kind: bubble.getAttribute('data-coach-kind') ?? '',
              turn: Number(bubble.getAttribute('data-coach-turn')),
              placement: bubble.getAttribute('data-coach-placement') ?? '',
              box: box(bubble)!,
              buttons: [...bubble.querySelectorAll('button')].map(
                (b) => b.getBoundingClientRect().height
              ),
              arrow: visible(arrow) ? box(arrow) : null
            }
          : null,
      note: note
        ? {
            step: note.getAttribute('data-coach-step') ?? '',
            kind: note.getAttribute('data-coach-kind') ?? ''
          }
        : null,
      targets: targets.length,
      target: tb ? { box: tb, region: target!.hasAttribute('data-coach-region'), hit } : null,
      summary: !!document.querySelector('[data-move-summary]'),
      tip: document.querySelector('[data-first-tip]')?.getAttribute('data-first-tip') ?? null,
      done: visible(document.querySelector('[data-tutorial-done]')),
      starter: !!document.querySelector('[data-act="starter:0"]:not(:disabled)'),
      vw: document.documentElement.clientWidth,
      vh: innerHeight,
      hscroll: document.documentElement.scrollWidth > innerWidth
    };
  });
}

function overlap(a: Box, b: Box): number {
  const w = Math.min(a.x + a.width, b.x + b.width) - Math.max(a.x, b.x);
  const h = Math.min(a.y + a.height, b.y + b.height) - Math.max(a.y, b.y);
  return w > 0 && h > 0 ? w * h : 0;
}

function inside(b: Box, vw: number, vh: number): boolean {
  return b.x >= -1 && b.y >= -1 && b.x + b.width <= vw + 1 && b.y + b.height <= vh + 1;
}

/** The checks of one coach step (the design's acceptance); returns the failures. */
export function stepFailures(s: CoachSnapshot): string[] {
  const fail: string[] = [];
  if (s.hscroll) fail.push('the page scrolls sideways');
  if (s.targets > 1) fail.push(`${s.targets} elements carry data-coach-target`);
  const t = s.target;
  if (t) {
    const b = t.box;
    // A region may be taller than a short screen: most of what fits is enough.
    const shown = Math.min(b.y + b.height, s.vh) - Math.max(b.y, 0);
    const regionShown = t.region && shown >= 0.85 * Math.min(b.height, s.vh);
    if (!inside(b, s.vw, s.vh) && !regionShown) fail.push('the target is not inside the viewport');
    if (!t.hit && !t.region) fail.push('the target is covered at its centre');
  }
  if (s.bubble) {
    const b = s.bubble.box;
    if (!inside(b, s.vw, s.vh)) fail.push('the bubble is not inside the viewport');
    if (t) {
      const share = overlap(b, t.box) / Math.max(1, t.box.width * t.box.height);
      if (!t.region && share > 0) fail.push('the bubble overlaps its target');
      if (t.region && share > 0.4) {
        fail.push(`the bubble covers ${Math.round(share * 100)}% of its region`);
      }
      // The arrow sits on the bubble's side that faces the target.
      if (s.bubble.arrow && !t.region) {
        const a = s.bubble.arrow;
        const towards =
          s.bubble.placement === 'above'
            ? a.y + a.height / 2 <= t.box.y + 1
            : a.y + a.height / 2 >= t.box.y + t.box.height - 1;
        if (!towards) fail.push('the arrow does not point towards the target');
      }
    }
    for (const h of s.bubble.buttons) {
      if (h < 44 - 0.5) fail.push(`a bubble button is ${h.toFixed(1)} px tall`);
    }
  }
  return fail;
}

export interface WalkStep {
  step: string;
  kind: string;
  surface: 'bubble' | 'note';
  placement: string;
  version: number | null;
  failures: string[];
}

export interface WalkResult {
  steps: WalkStep[];
  /** Lesson ids in order: each info lesson, and each action once per turn. */
  observed: string[];
  /** Every action key the page sent, in order. */
  sent: string[];
  /** The first-time tips seen, in order. */
  tips: string[];
  /** The tips seen while the tutorial was guided. */
  guidedTips: string[];
  /** Whether a starter choice came up. */
  starter: boolean;
  done: boolean;
}

export interface WalkOptions {
  /** Touch (tap) or a mouse (click). */
  touch: boolean;
  /** Stop after this step's Next (an info lesson), or at the completion card. */
  until?: string;
  /** Moves in the free part before giving up. */
  maxMoves?: number;
  /** Called at each new coach step, after the checks. */
  onStep?: (s: CoachSnapshot, step: WalkStep) => Promise<void> | void;
}

/** Records every act frame the page sends (call before the room opens). */
export function recordSent(page: Page): string[] {
  const sent: string[] = [];
  page.on('websocket', (ws) => {
    ws.on('framesent', (f) => {
      if (typeof f.payload !== 'string') return;
      try {
        const msg = JSON.parse(f.payload) as { t?: string; key?: string };
        if (msg.t === 'act' && typeof msg.key === 'string') sent.push(msg.key);
      } catch {
        // Not JSON: not ours.
      }
    });
  });
  return sent;
}

const sleep = (ms: number) => new Promise<void>((resolve) => setTimeout(resolve, ms));

/** The boxes a step check reads, to tell when they stop moving. */
const boxesOf = (s: CoachSnapshot) =>
  JSON.stringify([s.target?.box ?? null, s.bubble?.box ?? null, s.note?.step ?? null]);

/**
 * A snapshot once the step's boxes are still: a sheet may still be sliding
 * in (Animation normal), and a note inside it moves with it. Two reads 50 ms
 * apart must agree, for at most 2 s; then the last read is checked anyway.
 */
async function stillSnapshot(page: Page): Promise<CoachSnapshot> {
  await sleep(50);
  let last = await snapshot(page);
  for (const until = Date.now() + 2000; Date.now() < until;) {
    await sleep(50);
    const next = await snapshot(page);
    if (boxesOf(next) === boxesOf(last)) return next;
    last = next;
  }
  return last;
}

/** Presses a selector's first visible match as a pointer would, at a person's pace. */
export async function pressOn(page: Page, selector: string, touch: boolean): Promise<void> {
  const target = page.locator(selector).locator('visible=true').first();
  if (touch) await target.tap();
  else await target.click();
  // Never a second tap within the double-tap window (a quick second tap in
  // an open sheet can be taken as part of a double tap).
  await sleep(150);
}

/** Taps an empty part of the top bar (closes the move summary); Escape when there is none. */
async function tapTopBar(page: Page, touch: boolean): Promise<void> {
  const at = await page.evaluate(() => {
    const r = document.querySelector('.top-bar')?.getBoundingClientRect();
    if (!r) return null;
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
  if (!at) await page.keyboard.press('Escape');
  else if (touch) await page.touchscreen.tap(at.x, at.y);
  else await page.mouse.click(at.x, at.y);
  await sleep(150);
}

/** The free part's next press (run in the page): a selector, or null to wait. */
function freeMove(): string | null {
  const enabled = (el: Element) =>
    !(el as HTMLButtonElement).disabled && el.getBoundingClientRect().width > 0;
  const sheetAct = [...document.querySelectorAll<HTMLElement>('dialog[open] [data-act]')].filter(
    enabled
  );
  const pick =
    sheetAct.find((b) => b.dataset['act']!.startsWith('construct:')) ??
    sheetAct.find((b) => b.dataset['act']!.startsWith('discard:'));
  if (pick) return `[data-act="${pick.dataset['act']}"]`;
  if (document.querySelector('dialog[open]')) return null;
  const slot = document.querySelector<HTMLElement>(
    '[data-tut="structure"] [data-card][data-accessible]'
  );
  if (slot) return `[data-tut="structure"] [data-card="${slot.dataset['card']}"]`;
  const any = [...document.querySelectorAll<HTMLElement>('[data-act]')].find(enabled);
  return any ? `[data-act="${any.dataset['act']}"]` : null;
}

/** Walks the tutorial with the policy above, checking every coach step. */
export async function walkTutorial(
  page: Page,
  opts: WalkOptions,
  sent: string[] = []
): Promise<WalkResult> {
  const result: WalkResult = {
    steps: [],
    observed: [],
    sent,
    tips: [],
    guidedTips: [],
    starter: false,
    done: false
  };
  let lastKey: string | null = null;
  let lastAction: string | null = null;
  let lastVersion: number | null = null;
  let versionAt = Date.now();
  // When the last coach step was pressed: the next one shows two frames
  // later, with no new frame, so it gets the same grace.
  let pressedAt = 0;
  let moves = 0;
  const maxMoves = opts.maxMoves ?? 120;
  const debug = process.env['TUTORIAL_WALK_DEBUG'] === '1';
  const t0 = Date.now();
  const log = (what: string) => {
    if (debug) console.log(`walk +${Date.now() - t0}ms ${what}`);
  };
  for (;;) {
    const s = await snapshot(page);
    log(
      `v${s.version} ready=${s.ready} ${s.tutorial?.status ?? '-'} bubble=${s.bubble?.step ?? '-'} note=${s.note?.step ?? '-'} summary=${s.summary} tip=${s.tip ?? '-'}`
    );
    if (s.version !== lastVersion) {
      lastVersion = s.version;
      versionAt = Date.now();
    }
    if (s.done) {
      result.done = true;
      return result;
    }
    if (moves > maxMoves) throw new Error(`no completion card after ${moves} moves`);
    if (s.summary) {
      await tapTopBar(page, opts.touch);
      continue;
    }
    if (s.tip) {
      result.tips.push(s.tip);
      if (s.tutorial?.status === 'guided') result.guidedTips.push(s.tip);
      await pressOn(page, '[data-tip-ok]', opts.touch);
      continue;
    }
    const shown = s.bubble
      ? {
          step: s.bubble.step,
          kind: s.bubble.kind,
          surface: 'bubble' as const,
          placement: s.bubble.placement
        }
      : s.note
        ? { ...s.note, surface: 'note' as const, placement: 'note' }
        : null;
    if (shown) {
      const key = `${s.version}:${shown.step}:${shown.kind}:${shown.surface}`;
      if (key !== lastKey) {
        lastKey = key;
        const settled = await stillSnapshot(page);
        const step: WalkStep = { ...shown, version: s.version, failures: stepFailures(settled) };
        result.steps.push(step);
        if (shown.kind === 'info') result.observed.push(shown.step);
        else if (lastAction !== `${s.version}:${shown.step}`) {
          lastAction = `${s.version}:${shown.step}`;
          result.observed.push(shown.step);
        }
        expect
          .soft(
            step.failures,
            `coach step ${shown.step} (${shown.kind}, ${shown.surface}): target ${JSON.stringify(settled.target)}, bubble ${JSON.stringify(settled.bubble?.box ?? null)}, viewport ${settled.vw}x${settled.vh}`
          )
          .toEqual([]);
        await opts.onStep?.(settled, step);
      }
      if (shown.kind === 'info') {
        await pressOn(page, '[data-coach-next]', opts.touch);
        pressedAt = Date.now();
        if (opts.until === shown.step) return result;
      } else if (shown.kind === 'note' || !s.target) {
        if (shown.kind === 'note') await page.keyboard.press('Escape');
        await sleep(100);
      } else {
        await pressOn(page, '[data-coach-target]', opts.touch);
        if (shown.kind === 'act') {
          const from = s.version;
          await page
            .waitForFunction(
              (v) =>
                (
                  window as unknown as { __ajayeb?: { room(): { version: number } | null } }
                ).__ajayeb?.room()?.version !== v,
              from,
              { timeout: 5000 }
            )
            .catch(() => {});
        }
      }
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
    // A frame's info lesson shows two frames after the presenter settles
    // (the next one two frames after Next): a short grace before the free
    // part moves on (its outro lessons stay listed once acknowledged, so
    // their presence alone proves nothing).
    if (tut && tut.status !== 'complete' && Date.now() - Math.max(versionAt, pressedAt) < 600) {
      await sleep(80);
      continue;
    }
    if (s.starter) {
      result.starter = true;
      await pressOn(page, '[data-act="starter:0"]', opts.touch);
      moves++;
      continue;
    }
    const sel = await page.evaluate(freeMove);
    if (sel) {
      await pressOn(page, sel, opts.touch);
      if (sel.startsWith('[data-act')) moves++;
    } else {
      await sleep(150);
    }
  }
}
