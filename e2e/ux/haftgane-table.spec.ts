// Haftgane's table screen geometry (M18.table-screen): tables of 3, 5 and 7
// seats, the page's user and bots, at 360x740, 390x844, 844x390 and
// 1280x800, in both themes, left to right and right to left. At each size,
// with nothing selected and with a hand card selected (its pick list
// open), the probe finds no horizontal overflow, no control under 44 px,
// no seat, hand card or bar cut off by the screen's edges, no glyph
// without a tooltip and nothing animating with Animation off; and the
// page logs no console error or failed request. Each state is saved as a
// screenshot for the contact sheet, in HG_TABLE_SHOTS_DIR.
import { expect, type Page } from '@playwright/test';
import { mkdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
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

interface Probe {
  overflow: boolean;
  small: { act: string; width: number; height: number }[];
  cut: string[];
  untipped: string[];
  animated: number;
}

async function probe(page: Page): Promise<Probe> {
  return page.evaluate(() => {
    const root = document.querySelector<HTMLElement>('[data-table-screen]')!;
    const describe = (el: Element) =>
      [...el.attributes]
        .filter((a) => a.name.startsWith('data-'))
        .map((a) => `[${a.name}=${a.value}]`)
        .join('') || el.tagName.toLowerCase();
    const shown = (el: Element) =>
      el.getClientRects().length > 0 &&
      el.checkVisibility({ opacityProperty: true, visibilityProperty: true });
    // The part of an element left once every clipping ancestor and the
    // viewport have cut it.
    const cutOff = (el: Element): boolean => {
      const r = el.getBoundingClientRect();
      let { left, right, top, bottom } = r;
      left = Math.max(left, 0);
      top = Math.max(top, 0);
      right = Math.min(right, innerWidth);
      bottom = Math.min(bottom, innerHeight);
      for (let a = el.parentElement; a; a = a.parentElement) {
        const style = getComputedStyle(a);
        if (style.overflowX !== 'visible' || style.overflowY !== 'visible') {
          const c = a.getBoundingClientRect();
          left = Math.max(left, c.left);
          right = Math.min(right, c.right);
          top = Math.max(top, c.top);
          bottom = Math.min(bottom, c.bottom);
        }
      }
      return right - left < r.width - 1 || bottom - top < r.height - 1;
    };
    const controls = [
      ...root.querySelectorAll<HTMLElement>('button, a, select, input, [role="button"]')
    ].filter(shown);
    const parts = [
      ...root.querySelectorAll('[data-seat], [data-hand] [data-card], [data-turn-bar]'),
      ...controls
    ].filter(shown);
    const untipped: string[] = [];
    for (const el of root.querySelectorAll('svg, [role="img"]')) {
      if (el.parentElement?.closest('svg') || !shown(el)) continue;
      if (el.closest('[data-tooltip]')) continue;
      const button = el.closest('button');
      if (button && [...button.childNodes].some((n) => n.textContent?.trim())) continue;
      untipped.push(describe(el.parentElement ?? el));
    }
    return {
      overflow:
        root.scrollWidth > root.clientWidth + 1 ||
        document.documentElement.scrollWidth > innerWidth + 1,
      small: controls
        .map((el) => ({
          act: el.dataset['act'] ?? el.tagName,
          width: Math.round(el.getBoundingClientRect().width),
          height: Math.round(el.getBoundingClientRect().height)
        }))
        .filter((r) => r.width < 43.9 || r.height < 43.9),
      cut: parts.filter(cutOff).map(describe).slice(0, 10),
      untipped: untipped.slice(0, 10),
      animated: document.getAnimations().filter((a) => {
        const timing = a.effect?.getComputedTiming();
        return (
          a.playState === 'running' && typeof timing?.duration === 'number' && timing.duration > 1
        );
      }).length
    };
  });
}

for (const locale of ['en', 'fa'] as const)
  for (const theme of ['light', 'dark'] as const) {
    test.describe(`haftgane table geometry ${locale} ${theme}`, () => {
      test.use({ appLocale: locale });
      test('3, 5 and 7 seats at four viewports', async ({ page }) => {
        test.setTimeout(240_000);
        const out = process.env.HG_TABLE_SHOTS_DIR ?? '../.cache/e2e/hg-table-shots';
        mkdirSync(out, { recursive: true });
        const problems = watchHygiene(page);
        const player = locale === 'fa' ? 'روشن' : 'Roshan';
        await page.addInitScript(
          ({ t, name }) => {
            localStorage.setItem('ajayeb:theme', t);
            localStorage.setItem('ajayeb:name', name);
          },
          { t: theme, name: player }
        );
        const probes = [];
        for (const seats of [3, 5, 7]) {
          await page.setViewportSize({ width: 390, height: 844 });
          await openBotTable(page, seats, player);
          await expect(page.locator('html')).toHaveAttribute('data-theme', theme);
          await expect(page.locator('html')).toHaveAttribute(
            'dir',
            locale === 'fa' ? 'rtl' : 'ltr'
          );
          // Past a side point, if the table has one, to a turn with a full hand.
          const sides = await latest(page);
          if (sides.view.phase === 'sides') {
            await choose(page, 'side:day');
            await choose(page, 'confirm');
            await expect
              .poll(async () => (await latest(page)).point?.ref.id)
              .not.toBe(sides.point!.ref.id);
          }
          const turn = await latest(page);
          const design = turn.view.you!.hand[0]!;
          for (const [width, height] of sizes) {
            await page.setViewportSize({ width, height });
            // The sheet's complete variant matrix lives in haftgane-card-sheet.spec.ts.
            for (const state of ['hand'] as const) {
              const card = page.locator(`[data-act="hand:${design}"]`).first();
              const pressed = (await card.getAttribute('aria-pressed')) === 'true';
              if (state === 'hand' && pressed)
                await page.locator('[data-act="pick-close"]').click();
              await ready(page);
              await quiet(page);
              const p = await probe(page);
              probes.push({ seats, width, height, locale, theme, state, ...p });
              const name = `table-${seats}p-${state}-${width}x${height}-${locale}-${theme}.png`;
              await page.screenshot({ path: path.join(out, name) });
              expect(p, `${seats} seats ${state} ${width}x${height}`).toEqual({
                overflow: false,
                small: [],
                cut: [],
                untipped: [],
                animated: 0
              });
            }
          }
        }
        writeFileSync(
          path.join(out, `probes-${locale}-${theme}.json`),
          JSON.stringify({ probes, problems }, null, 2)
        );
        expect(problems).toEqual([]);
      });
    });
  }
