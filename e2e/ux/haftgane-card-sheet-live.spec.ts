// Home -> five-seat lobby -> real actor -> sheet over the live table.
// Freeze the human's point by drafting without confirming while capturing.
import { expect } from '@playwright/test';
import { mkdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { quiet } from '../fixtures/layout-probes';
import { probeCardSheet } from '../fixtures/card-sheet-probe';
import { choose, dismissMilitary, latest, openBotTable, ready } from '../fixtures/table';

for (const [locale, theme] of [
  ['en', 'light'],
  ['fa', 'dark']
] as const) {
  test.describe(`live Haftgane card sheet ${locale} ${theme}`, () => {
    test.use({ appLocale: locale });
    test('trade and an alternate use over a live five-seat table', async ({ page }) => {
      test.setTimeout(240_000);
      const problems = watchHygiene(page);
      await page.addInitScript((t) => localStorage.setItem('ajayeb:theme', t), theme);
      const code = await openBotTable(page, 5, 'Player');
      await expect(page.locator('html')).toHaveAttribute('dir', locale === 'fa' ? 'rtl' : 'ltr');
      await expect(page.locator('html')).toHaveAttribute('data-theme', theme);
      let frame = await latest(page);
      // Production rooms choose their seed. Advance with actual offered keys
      // until this hand has a paid trade build, without injecting state.
      for (let turn = 0; turn < 20; turn++) {
        const trade = frame.view.you!.choices.find(
          (c) => c.kind === 'build' && (c.left > 0 || c.right > 0)
        );
        if (trade) break;
        const pick =
          frame.view.you!.choices.find((c) => c.kind === 'build') ?? frame.view.you!.choices[0]!;
        const point = frame.point!.ref.id;
        await choose(page, pick.key);
        await choose(page, 'confirm');
        await expect.poll(async () => (await latest(page)).point?.ref.id).not.toBe(point);
        frame = await latest(page);
      }
      const trade = frame.view.you!.choices.find(
        (c) => c.kind === 'build' && (c.left > 0 || c.right > 0)
      );
      expect(trade, 'a real paid trade build').toBeDefined();
      const alternate =
        frame.view.you!.choices.find((c) => c.design === trade!.design && c.kind === 'stage') ??
        frame.view.you!.choices.find((c) => c.design === trade!.design && c.kind === 'sell');
      expect(alternate, 'a real stage or sale of the same card').toBeDefined();
      const out = process.env.HG_SHEET_LIVE_SHOTS_DIR ?? '../.cache/e2e/hg-sheet-live-shots';
      mkdirSync(out, { recursive: true });
      const probes = [];
      await dismissMilitary(page);
      for (const choice of [trade!, alternate!]) {
        await page.locator(`[data-act="hand:${choice.design}"]`).first().click();
        await page.locator(`[data-card-sheet] [data-pick="${choice.key}"]`).click();
        await ready(page);
        for (const [width, height] of [
          [360, 740],
          [390, 844],
          [844, 390],
          [1280, 800]
        ]) {
          await page.setViewportSize({ width: width!, height: height! });
          if (choice.kind !== 'build')
            await page
              .locator(choice.kind === 'stage' ? '#hg-stage-title' : '#hg-sale-title')
              .scrollIntoViewIfNeeded();
          else await page.locator('[data-sheet-body]').evaluate((body) => body.scrollTo(0, 0));
          await quiet(page);
          const p = await probeCardSheet(page);
          const tableOverflow = await page
            .locator('[data-table-screen]')
            .evaluate((el) => el.scrollWidth > el.clientWidth + 1);
          probes.push({
            code,
            point: frame.point!.ref.id,
            key: choice.key,
            seats: frame.view.seats.length,
            width,
            height,
            locale,
            theme,
            tableOverflow,
            ...p
          });
          await page.screenshot({
            path: path.join(out, `5p-${choice.kind}-${width}x${height}-${locale}-${theme}.png`)
          });
          expect(tableOverflow).toBe(false);
          expect(p).toEqual({
            lowContrast: false,
            crampedTitle: false,
            overflow: false,
            small: [],
            untipped: [],
            outside: false,
            footerOutside: false,
            animated: 0
          });
        }
        await page.locator('[data-act="pick-close"]').click();
      }
      writeFileSync(
        path.join(out, `probes-${locale}-${theme}.json`),
        JSON.stringify({ probes, problems }, null, 2)
      );
      expect(problems).toEqual([]);
    });
  });
}
