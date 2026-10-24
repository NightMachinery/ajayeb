// Haftgane's confirmed pick (M18.sound-and-motion): two people at a table
// with bots, and the first to confirm waits on the other. Right after the
// confirmation shows, nothing animates with Animation off; then at
// 360x740, 390x844, 844x390 and 1280x800, in English and light and in
// Persian and dark, the first person's turn bar shows the check on the
// confirmed pick, inside the screen, with no horizontal overflow and no
// console error or failed request. Each state is saved as a screenshot
// for the contact sheet, in HG_CONFIRM_SHOTS_DIR.
import { expect, type Page } from '@playwright/test';
import { mkdirSync } from 'node:fs';
import path from 'node:path';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { createTableFromHome } from '../fixtures/lobby';
import { newDeviceContext } from '../fixtures/online';
import { choose, latest, ready } from '../fixtures/table';

const sizes = [
  [360, 740],
  [390, 844],
  [844, 390],
  [1280, 800]
] as const;

interface Probe {
  overflow: boolean;
  check: boolean;
  barInside: true | string;
  animated: number;
}

async function probe(page: Page): Promise<Probe> {
  return page.evaluate(() => {
    const check = document.querySelector('[data-pending-confirm="true"] .check');
    const bar = document.querySelector('[data-turn-bar]')!.getBoundingClientRect();
    return {
      overflow: document.documentElement.scrollWidth > innerWidth + 1,
      check: !!check && check.getClientRects().length > 0,
      barInside:
        bar.left >= -1 && bar.right <= innerWidth + 1 && bar.bottom <= innerHeight + 1
          ? true
          : `${Math.round(bar.left)},${Math.round(bar.right)},${Math.round(bar.bottom)} in ${innerWidth}x${innerHeight}`,
      animated: document.getAnimations().filter((a) => {
        const timing = a.effect?.getComputedTiming();
        return (
          a.playState === 'running' && typeof timing?.duration === 'number' && timing.duration > 1
        );
      }).length
    };
  });
}

for (const [locale, theme] of [
  ['en', 'light'],
  ['fa', 'dark']
] as const) {
  test.describe(`haftgane confirmation ${locale} ${theme}`, () => {
    test.use({ appLocale: locale });
    test('the confirmed pick at four viewports', async ({ page, browser }, info) => {
      test.setTimeout(120_000);
      const out = process.env.HG_CONFIRM_SHOTS_DIR ?? '../.cache/e2e/hg-confirm-shots';
      mkdirSync(out, { recursive: true });
      const other = await newDeviceContext(browser, info);
      try {
        const b = await other.newPage();
        await b.addInitScript(() => localStorage.setItem('ajayeb:name', 'Bo'));
        const problems = watchHygiene(page);
        const name = locale === 'fa' ? 'روشن' : 'Roshan';
        await page.addInitScript(
          ({ t, n }) => {
            localStorage.setItem('ajayeb:theme', t);
            localStorage.setItem('ajayeb:name', n);
          },
          { t: theme, n: name }
        );
        await page.setViewportSize({ width: 390, height: 844 });
        const code = await createTableFromHome(page, name);
        await b.goto(`/play/${code}`);
        await expect
          .poll(() =>
            page.evaluate(
              () => window.__ajayebTable?.lobby()?.seats.filter((s) => s.kind === 'human').length
            )
          )
          .toBe(2);
        await choose(page, 'lobby-start');
        await expect(page.locator('[data-table-screen]')).toBeVisible();
        await expect(page.locator('html')).toHaveAttribute('data-theme', theme);
        const f = await latest(page);
        const bots = f.seats.filter((s) => s.kind === 'bot').map((s) => s.seat);
        // The bots decide at once; once they have, the table waits on the people.
        await expect
          .poll(async () => {
            const p = (await latest(page)).point;
            return bots.every((s) => p?.confirmed.includes(s));
          })
          .toBe(true);
        await choose(page, f.view.you!.choices[0]!.key);
        await choose(page, 'confirm');
        await expect(page.locator('[data-pending-confirm="true"] .check')).toBeVisible();
        // At once, before anything could have finished on its own.
        expect((await probe(page)).animated, 'nothing animates with Animation off').toBe(0);
        for (const [width, height] of sizes) {
          await page.setViewportSize({ width, height });
          await ready(page);
          // Two frames, so the layout has followed the new size.
          await page.evaluate(
            () =>
              new Promise<void>((resolve) =>
                requestAnimationFrame(() => requestAnimationFrame(() => resolve()))
              )
          );
          const p = await probe(page);
          await page.screenshot({
            path: path.join(out, `confirm-${width}x${height}-${locale}-${theme}.png`)
          });
          expect(p, `${width}x${height}`).toEqual({
            overflow: false,
            check: true,
            barInside: true,
            animated: 0
          });
        }
        expect(problems).toEqual([]);
      } finally {
        await other.close();
      }
    });
  });
}
