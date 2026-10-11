// Objective layout probes (UX design section 9, "Objective probes"), run in
// the page: horizontal overflow, page scroll, the size of takeable cards and
// of the primary controls. Each returns plain data, so a spec can assert on
// it and print what went wrong.

import type { Page } from '@playwright/test';

export interface Box {
  what: string;
  x: number;
  y: number;
  w: number;
  h: number;
}

export interface LayoutReport {
  innerWidth: number;
  innerHeight: number;
  scrollWidth: number;
  scrollHeight: number;
  /** Visible elements whose visible part reaches past the left or right edge. */
  overflowing: Box[];
  /** Takeable structure cards. */
  takeable: Box[];
  /** Enabled, visible primary controls. */
  controls: Box[];
  /** The game screen's layout class, if one is on screen. */
  layout: string | null;
  /** Seat panels (the player strips on a phone) whose content is taller
   *  than their box, so they would scroll inside; `h` is the hidden part. */
  stripOverflow: Box[];
}

/**
 * The primary controls: move buttons (data-act), panel and draft tiles
 * (data-pick), the top bar and wonder-flow controls, the panels' Peek and
 * Begin buttons, the result screen's buttons, the pass-device screen's
 * buttons, and every button of an open sheet.
 */
export const PRIMARY_CONTROLS = [
  '[data-act]',
  '[data-pick]',
  '[data-control="undo"]',
  '[data-control="log"]',
  '[data-control="settings"]',
  '[data-control="wonder-step"]',
  '[data-control="wonder-build"]',
  '[data-control="wonder-pick-cancel"]',
  '[data-control="wonder-back"]',
  '[data-peek]',
  '[data-begin-age]',
  '[data-game-over] button',
  '[data-pass-device] button',
  'dialog[open] button'
].join(', ');

/** Measures the page as it is now. */
export async function probeLayout(page: Page): Promise<LayoutReport> {
  return page.evaluate((controlsSelector) => {
    const round = (n: number) => Math.round(n * 10) / 10;
    const describe = (el: Element): string => {
      const data = [...el.attributes]
        .filter((a) => a.name.startsWith('data-'))
        .map((a) => `[${a.name}${a.value ? `=${a.value}` : ''}]`)
        .join('');
      const cls = typeof el.className === 'string' ? el.className.trim().split(/\s+/)[0] : '';
      return `${el.tagName.toLowerCase()}${cls ? '.' + cls : ''}${data}`.slice(0, 120);
    };
    const box = (
      el: Element,
      r: DOMRect
    ): { what: string; x: number; y: number; w: number; h: number } => ({
      what: describe(el),
      x: round(r.x),
      y: round(r.y),
      w: round(r.width),
      h: round(r.height)
    });
    const shown = (el: Element): boolean =>
      el.checkVisibility({ opacityProperty: true, visibilityProperty: true }) &&
      !el.closest('[inert]') &&
      !el.closest('dialog:not([open])');

    // The part of an element that is actually painted: its box clipped by
    // every ancestor that clips its overflow.
    const visiblePart = (el: Element): DOMRect | null => {
      const r = el.getBoundingClientRect();
      let left = r.left;
      let right = r.right;
      let top = r.top;
      let bottom = r.bottom;
      for (let a = el.parentElement; a && a !== document.documentElement; a = a.parentElement) {
        const style = getComputedStyle(a);
        if (style.overflowX !== 'visible' || style.overflowY !== 'visible') {
          const c = a.getBoundingClientRect();
          if (style.overflowX !== 'visible') {
            left = Math.max(left, c.left);
            right = Math.min(right, c.right);
          }
          if (style.overflowY !== 'visible') {
            top = Math.max(top, c.top);
            bottom = Math.min(bottom, c.bottom);
          }
        }
        if (style.position === 'fixed') break;
      }
      if (right - left <= 0.5 || bottom - top <= 0.5) return null;
      return new DOMRect(left, top, right - left, bottom - top);
    };

    const overflowing = [];
    for (const el of document.body.querySelectorAll('*')) {
      if (el.closest('svg') && el.tagName.toLowerCase() !== 'svg') continue;
      if (!shown(el)) continue;
      const part = visiblePart(el);
      if (!part) continue;
      if (part.right > innerWidth + 0.5 || part.left < -0.5) overflowing.push(box(el, part));
      if (overflowing.length >= 10) break;
    }

    const takeable = [...document.querySelectorAll('.card-structure [data-card][data-accessible]')]
      .filter(shown)
      .map((el) => box(el, el.getBoundingClientRect()));

    const controls = [...document.querySelectorAll(controlsSelector)]
      .filter(
        (el) =>
          shown(el) &&
          !(el as HTMLButtonElement).disabled &&
          el.getAttribute('aria-disabled') !== 'true'
      )
      .map((el) => box(el, el.getBoundingClientRect()));

    const stripOverflow = [...document.querySelectorAll('.game-screen .seat-panel')]
      .filter(shown)
      .filter((el) => el.scrollHeight > el.clientHeight + 1)
      .map((el) => {
        const r = el.getBoundingClientRect();
        return { ...box(el, r), h: el.scrollHeight - el.clientHeight };
      });

    const root = document.documentElement;
    return {
      innerWidth,
      innerHeight,
      scrollWidth: Math.max(root.scrollWidth, document.body.scrollWidth),
      scrollHeight: Math.max(root.scrollHeight, document.body.scrollHeight),
      overflowing,
      takeable,
      controls,
      layout: document.querySelector<HTMLElement>('.game-screen')?.dataset['layout'] ?? null,
      stripOverflow
    };
  }, PRIMARY_CONTROLS);
}

/** Waits for running transitions and animations to end, then two painted frames. */
export async function quiet(page: Page): Promise<void> {
  await page.waitForFunction(
    () => document.getAnimations().every((a) => a.playState !== 'running'),
    null,
    { timeout: 10_000 }
  );
  await page.evaluate(async () => {
    await new Promise<void>((resolve) =>
      requestAnimationFrame(() => requestAnimationFrame(() => resolve()))
    );
  });
}

/** Waits for the page to settle: the game ready, fonts loaded, nothing moving. */
export async function settle(page: Page): Promise<void> {
  await page.waitForFunction(
    () => window.__ajayeb?.ready() === true && document.documentElement.dataset['ready'] === '1',
    null,
    { timeout: 20_000 }
  );
  await page.evaluate(() => document.fonts.ready.then(() => undefined));
  await quiet(page);
}
