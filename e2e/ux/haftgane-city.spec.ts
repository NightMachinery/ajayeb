// Haftgane's city sheet and Seen hands sheet geometry
// (M18.city-sheet-and-notebook): one live table each of 3, 5 and 7 seats,
// the page's user and bots, started from Home's lobby. At three moments of
// the game (the first turn, Age II's second turn and the end) the page
// switches through English and Persian in both themes, reloading the
// table, and opens Seen hands (empty at the first turn, with Age I and II
// memory mid-game) and the city sheet on its own city, a neighbour and a
// far seat (mid-game and at the end). At 360x740, 390x844, 844x390 and
// 1280x800 the probe finds no horizontal overflow or scroller, no control
// under 44 px, the sheet inside the screen, no glyph without a tooltip and
// nothing animating with Animation off; and the page logs no console
// error or failed request. Each state is saved as a screenshot for the
// contact sheet, in HG_CITY_SHOTS_DIR. A last test checks the dev
// gallery's city and notebook leaves and opens both sheets from it.
import { expect, type Page } from '@playwright/test';
import { mkdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import type { TableState } from '../../web/src/lib/wire/platform.gen';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { quiet } from '../fixtures/layout-probes';
import { choose, latest, openBotTable, ready } from '../fixtures/table';

const sizes = [
  [360, 740],
  [390, 844],
  [844, 390],
  [1280, 800]
] as const;
const variants = [
  ['en', 'light'],
  ['en', 'dark'],
  ['fa', 'light'],
  ['fa', 'dark']
] as const;

interface Probe {
  overflow: string[];
  small: { act: string; width: number; height: number }[];
  outside: boolean;
  crampedTitle: boolean;
  untipped: string[];
  animated: number;
}

async function probe(page: Page, selector: string): Promise<Probe> {
  return page.locator(selector).evaluate((root) => {
    const dialog = root.closest('dialog') ?? root;
    const box = dialog.getBoundingClientRect();
    const panel = root.getBoundingClientRect();
    const shown = (el: Element) =>
      el.getClientRects().length > 0 &&
      el.checkVisibility({ opacityProperty: true, visibilityProperty: true });
    const describe = (el: Element) =>
      [...el.attributes]
        .filter((a) => a.name.startsWith('data-'))
        .map((a) => `[${a.name}=${a.value}]`)
        .join('') || el.tagName.toLowerCase();
    // Horizontal overflow: the page, the table behind, the sheet, any
    // horizontal scroller inside it, and anything past the sheet's sides.
    const overflow: string[] = [];
    if (document.documentElement.scrollWidth > innerWidth + 1) overflow.push('page');
    const table = document.querySelector('[data-table-screen]');
    if (table && table.scrollWidth > table.clientWidth + 1) overflow.push('table');
    if (root.scrollWidth > root.clientWidth + 1) overflow.push('sheet');
    for (const el of root.querySelectorAll<HTMLElement>('*')) {
      if (!shown(el)) continue;
      const style = getComputedStyle(el);
      if (
        (style.overflowX === 'auto' || style.overflowX === 'scroll') &&
        el.scrollWidth > el.clientWidth + 1
      )
        overflow.push(`scroller ${describe(el)}`);
      const r = el.getBoundingClientRect();
      if (r.width > 0 && (r.left < panel.left - 1 || r.right > panel.right + 1))
        overflow.push(`past the side ${describe(el)}`);
    }
    const small = [...root.querySelectorAll<HTMLElement>('button:not(:disabled), [role="tab"]')]
      .filter((el) => !el.closest('[inert]') && shown(el))
      .map((el) => ({
        act: el.dataset['act'] ?? el.tagName,
        width: Math.round(el.getBoundingClientRect().width),
        height: Math.round(el.getBoundingClientRect().height)
      }))
      .filter((r) => r.width < 43.9 || r.height < 43.9);
    const untipped = [...root.querySelectorAll('svg, [role="img"]')]
      .filter(
        (el) =>
          !el.parentElement?.closest('svg') &&
          !el.closest('[data-tooltip]') &&
          !el.closest('[inert]') &&
          shown(el)
      )
      .map((el) => describe(el.parentElement ?? el));
    const title = root.querySelector('h2');
    return {
      overflow: overflow.slice(0, 10),
      small,
      outside:
        box.x < -1 || box.y < -1 || box.right > innerWidth + 1 || box.bottom > innerHeight + 1,
      crampedTitle: !!title && title.getBoundingClientRect().width < 80,
      untipped: untipped.slice(0, 10),
      animated: document
        .getAnimations()
        .filter(
          (a) => a.playState === 'running' && Number(a.effect?.getComputedTiming().duration) > 1
        ).length
    };
  });
}

/** Plays the page's seat until the frame satisfies `done`: a stage when one
 *  is offered, else a building, else the first choice; bots play the rest. */
async function playUntil(page: Page, done: (f: TableState) => boolean): Promise<TableState> {
  for (let i = 0; i < 120; i++) {
    const f = await latest(page);
    if (done(f) || f.status === 'finished') return f;
    const me = f.you.seat!;
    const ref = f.point!.ref;
    if (f.point!.required.includes(me) && !f.pending?.confirm) {
      const cs = f.view.you!.choices;
      const pick =
        (f.view.phase === 'sides' ? cs.find((c) => c.key === 'side:day') : undefined) ??
        (f.view.turn > 1 ? cs.find((c) => c.kind === 'stage') : undefined) ??
        cs.find((c) => c.kind === 'build') ??
        cs[0]!;
      await choose(page, pick.key);
      await choose(page, 'confirm');
    }
    await expect
      .poll(async () => {
        const g = await latest(page);
        return (
          g.status === 'finished' ||
          g.point?.ref.id !== ref.id ||
          g.point?.ref.revision !== ref.revision
        );
      })
      .toBe(true);
  }
  throw new Error('the game did not reach the moment');
}

for (const seats of [3, 5, 7] as const) {
  test.describe(`haftgane city and Seen hands geometry ${seats} seats`, () => {
    test('three moments, four variants, four viewports', async ({ page }) => {
      test.setTimeout(900_000);
      const out = process.env.HG_CITY_SHOTS_DIR ?? '../.cache/e2e/hg-city-shots';
      mkdirSync(out, { recursive: true });
      const problems = watchHygiene(page);
      await page.addInitScript(() => localStorage.setItem('ajayeb:name', 'Roshan'));
      await page.setViewportSize({ width: 390, height: 844 });
      await openBotTable(page, seats, 'Roshan');
      const probes: Record<string, unknown>[] = [];

      async function capture(moment: string, state: string, selector: string): Promise<void> {
        for (const [width, height] of sizes) {
          await page.setViewportSize({ width, height });
          await page
            .locator(selector)
            .evaluate((root) => (root.querySelector('[data-sheet-body]') ?? root).scrollTo(0, 0));
          await quiet(page);
          const p = await probe(page, selector);
          const v = await page.evaluate(() => [
            document.documentElement.lang,
            document.documentElement.dataset['theme']
          ]);
          const base = `city-${seats}p-${moment}-${state}-${width}x${height}-${v[0]}-${v[1]}`;
          probes.push({ seats, moment, state, width, height, locale: v[0], theme: v[1], ...p });
          await page.screenshot({ path: path.join(out, `${base}.png`) });
          if (width === 390) {
            await page.locator(selector).evaluate((root) => {
              const body = root.querySelector('[data-sheet-body]') ?? root;
              body.scrollTo(0, body.scrollHeight);
            });
            await quiet(page);
            await page.screenshot({ path: path.join(out, `${base}-scrolled.png`) });
          }
          expect(p, `${seats} seats ${moment} ${state} ${width}x${height} ${v}`).toEqual({
            overflow: [],
            small: [],
            outside: false,
            crampedTitle: false,
            untipped: [],
            animated: 0
          });
        }
      }

      async function moment(name: string, seen: 'empty' | 'memory' | null, cities: boolean) {
        const f = await latest(page);
        const order = [...f.view.seats].sort((a, b) => a.relative - b.relative);
        const targets: [string, number][] = [];
        if (cities) {
          targets.push(['own', order[0]!.seat]);
          targets.push(['neighbour', order.find((s) => s.relation === 'left')!.seat]);
          if (seats > 3) targets.push(['far', order.find((s) => s.relation === 'other')!.seat]);
        }
        for (const [locale, theme] of variants) {
          await page.evaluate(
            ({ l, t }) => {
              const settings = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}');
              localStorage.setItem('ajayeb:settings', JSON.stringify({ ...settings, locale: l }));
              localStorage.setItem('ajayeb:theme', t);
            },
            { l: locale, t: theme }
          );
          await page.setViewportSize({ width: 390, height: 844 });
          await page.reload();
          await ready(page);
          await expect(page.locator('html')).toHaveAttribute('data-theme', theme);
          await expect(page.locator('html')).toHaveAttribute(
            'dir',
            locale === 'fa' ? 'rtl' : 'ltr'
          );
          if (seen) {
            await page.locator('[data-act="seen-hands"]').click();
            await expect(page.locator('[data-seen-sheet]')).toBeVisible();
            await expect(page.locator('[data-seen-empty]')).toHaveCount(seen === 'empty' ? 1 : 0);
            await capture(name, `seen-${seen}`, '[data-seen-sheet]');
            await page.setViewportSize({ width: 390, height: 844 });
            await page.locator('[data-act="seen-close"]').click();
            await expect(page.locator('[data-seen-sheet]')).toHaveCount(0);
          }
          for (const [i, [state, seat]] of targets.entries()) {
            await page.setViewportSize({ width: 390, height: 844 });
            if (i === 0) await page.locator(`[data-act="city:${seat}"]`).first().click();
            else await page.locator(`[data-act="city-tab:${seat}"]`).click();
            await expect(page.locator(`[data-city-sheet="${seat}"]`)).toBeVisible();
            await capture(name, `city-${state}`, '[data-city-sheet]');
          }
          if (targets.length > 0) {
            await page.locator('[data-act="city-close"]').click();
            await expect(page.locator('[data-city-sheet]')).toHaveCount(0);
          }
        }
      }

      await playUntil(page, (f) => f.view.phase === 'turn');
      await moment('start', 'empty', false);
      await playUntil(page, (f) => f.view.age === 2 && f.view.turn >= 2);
      await moment('mid', 'memory', true);
      await playUntil(page, () => false);
      expect((await latest(page)).status).toBe('finished');
      await moment('end', null, true);

      writeFileSync(
        path.join(out, `probes-${seats}p.json`),
        JSON.stringify({ probes, problems }, null, 2)
      );
      expect(problems).toEqual([]);
    });
  });
}

test.describe('haftgane city gallery', () => {
  test('the gallery shows the city and notebook leaves and opens both sheets', async ({ page }) => {
    const out = process.env.HG_CITY_SHOTS_DIR ?? '../.cache/e2e/hg-city-shots';
    mkdirSync(out, { recursive: true });
    const problems = watchHygiene(page);
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto('/dev/gallery');
    const section = page.locator('[data-gallery-city]');
    await section.scrollIntoViewIfNeeded();
    await expect(section.locator('[data-city]')).toHaveCount(3);
    await expect(section.locator('[data-seat-tab]')).toHaveCount(7);
    await expect(section.locator('[data-under="own"]')).toHaveCount(1);
    await expect(section.locator('[data-under="revealed"]')).toHaveCount(1);
    await expect(section.locator('[data-neighbour-pick]')).toHaveCount(1);
    await quiet(page);
    await section.screenshot({ path: path.join(out, 'gallery-city-390x844.png') });
    const opens = section.locator(':scope > .row button');
    await opens.nth(0).click();
    await expect(page.locator('[data-city-sheet]')).toBeVisible();
    await quiet(page);
    expect(await probe(page, '[data-city-sheet]')).toEqual({
      overflow: [],
      small: [],
      outside: false,
      crampedTitle: false,
      untipped: [],
      animated: 0
    });
    await page.locator('[data-act="city-close"]').click();
    await opens.nth(1).click();
    await expect(page.locator('[data-seen-sheet] [data-seen-point]').first()).toBeVisible();
    await quiet(page);
    expect(await probe(page, '[data-seen-sheet]')).toEqual({
      overflow: [],
      small: [],
      outside: false,
      crampedTitle: false,
      untipped: [],
      animated: 0
    });
    await page.screenshot({ path: path.join(out, 'gallery-seen-390x844.png') });
    await page.locator('[data-act="seen-close"]').click();
    expect(problems).toEqual([]);
  });
});
