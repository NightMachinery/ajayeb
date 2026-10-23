import { expect } from '@playwright/test';
import { mkdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { createTableFromHome, lobbyAction, lobbyReady } from '../fixtures/lobby';
const sizes = [
  [360, 740],
  [390, 844],
  [844, 390],
  [1280, 800]
] as const;
for (const locale of ['en', 'fa'] as const)
  for (const theme of ['light', 'dark']) {
    test.describe(`lobby geometry ${locale} ${theme}`, () => {
      test.use({ appLocale: locale });
      test('four viewports, all targets, no overflow, instant changes', async ({ page }) => {
        const out = process.env.LOBBY_SHOTS_DIR ?? '../.cache/e2e/lobby-shots';
        mkdirSync(out, { recursive: true });
        const problems = watchHygiene(page);
        await page.addInitScript((t) => localStorage.setItem('ajayeb:theme', t), theme);
        const probes = [];
        for (const [width, height] of sizes) {
          await page.setViewportSize({ width, height });
          await createTableFromHome(page, locale === 'fa' ? 'روشن' : 'Roshan');
          await page.locator('[data-act="lobby-seat-menu:1"]').click();
          await lobbyAction(page, 'lobby-add-bot:1');
          await page.locator('[data-act="lobby-seat-menu:1"]').click();
          await page.locator('[data-act="lobby-seating"]').selectOption('lobby');
          await lobbyReady(page);
          await expect(page.locator('[data-act="lobby-seating"]')).toHaveValue('lobby');
          await expect(page.locator('html')).toHaveAttribute(
            'dir',
            locale === 'fa' ? 'rtl' : 'ltr'
          );
          const probe = await page.evaluate(() => {
            const main = document.querySelector('[data-lobby]')!;
            const targets = [
              ...main.querySelectorAll<HTMLElement>('button,select,input,a,summary')
            ].filter((el) => el.getClientRects().length && el.getBoundingClientRect().height > 0);
            return {
              overflow:
                main.scrollWidth > main.clientWidth + 1 ||
                document.documentElement.scrollWidth > innerWidth + 1,
              small: targets
                .map((el) => ({
                  act: el.dataset.act ?? el.tagName,
                  width: el.getBoundingClientRect().width,
                  height: el.getBoundingClientRect().height
                }))
                .filter((r) => r.width < 43.9 || r.height < 43.9),
              animated: main.getAnimations({ subtree: true }).filter((a) => {
                const timing = a.effect?.getComputedTiming();
                return (
                  a.playState === 'running' &&
                  typeof timing?.duration === 'number' &&
                  timing.duration > 1
                );
              }).length
            };
          });
          probes.push({ width, height, locale, theme, ...probe });
          expect(probe).toEqual({ overflow: false, small: [], animated: 0 });
          await page.screenshot({
            path: path.join(out, `lobby-${width}x${height}-${locale}-${theme}.png`),
            fullPage: true
          });
        }
        writeFileSync(
          path.join(out, `probes-${locale}-${theme}.json`),
          JSON.stringify({ probes, problems }, null, 2)
        );
        expect(problems).toEqual([]);
      });
    });
  }
