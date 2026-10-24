// Real-engine captured sheet states, all seat counts, viewports, themes and
// directions. The scrolling body may scroll vertically; its sticky actions
// remain reachable, and every control keeps its 44 px target.
import { expect } from '@playwright/test';
import { mkdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { quiet } from '../fixtures/layout-probes';

for (const locale of ['en', 'fa'] as const)
  for (const theme of ['light', 'dark'] as const) {
    test.describe(`card sheet ${locale} ${theme}`, () => {
      test.use({ appLocale: locale });
      test('all card-sheet variants at 3, 5 and 7 seats', async ({ page }) => {
        test.setTimeout(600_000);
        const out = process.env.HG_SHEET_SHOTS_DIR ?? '../.cache/e2e/hg-sheet-shots';
        mkdirSync(out, { recursive: true });
        const problems = watchHygiene(page);
        await page.addInitScript((t) => localStorage.setItem('ajayeb:theme', t), theme);
        const probes = [];
        for (const seats of [3, 5, 7]) {
          for (const state of ['trade', 'stage', 'sale', 'chain', 'free', 'blocked']) {
            await page.goto(`/dev/gallery?hgSheet=${state}&seats=${seats}`);
            const sheet = page.locator('[data-card-sheet]');
            await expect(sheet).toBeVisible();
            await expect(page.locator('html')).toHaveAttribute('data-theme', theme);
            await expect(page.locator('html')).toHaveAttribute(
              'dir',
              locale === 'fa' ? 'rtl' : 'ltr'
            );
            for (const [width, height] of [
              [360, 740],
              [390, 844],
              [844, 390],
              [1280, 800]
            ]) {
              await page.setViewportSize({ width: width!, height: height! });
              if (state === 'stage' || state === 'sale')
                await sheet
                  .locator(state === 'stage' ? '#hg-stage-title' : '#hg-sale-title')
                  .scrollIntoViewIfNeeded();
              else await sheet.locator('[data-sheet-body]').evaluate((body) => body.scrollTo(0, 0));
              await quiet(page);
              const p = await sheet.evaluate((root) => {
                const dialog = root.closest('dialog')!;
                const r = dialog.getBoundingClientRect();
                const small = [
                  ...root.querySelectorAll<HTMLElement>('button:not(:disabled), [role="radio"]')
                ]
                  .filter((e) => !e.closest('[inert]') && e.checkVisibility())
                  .map((e) => ({
                    key: e.dataset['act'] ?? e.dataset['pick'],
                    w: e.getBoundingClientRect().width,
                    h: e.getBoundingClientRect().height
                  }))
                  .filter((e) => e.w < 43.9 || e.h < 43.9);
                const untipped = [...root.querySelectorAll('svg, [role="img"]')]
                  .filter(
                    (e) =>
                      !e.parentElement?.closest('svg') &&
                      !e.closest('[data-tooltip]') &&
                      !e.closest('[inert]') &&
                      e.checkVisibility()
                  )
                  .map((e) => e.outerHTML.slice(0, 100));
                const footer = root.querySelector('.sheet-footer')!.getBoundingClientRect();
                return {
                  crampedTitle: root.querySelector('h2')!.getBoundingClientRect().width < 100,
                  overflow:
                    document.documentElement.scrollWidth > innerWidth + 1 ||
                    root.scrollWidth > root.clientWidth + 1,
                  small,
                  untipped,
                  outside:
                    r.x < -1 || r.y < -1 || r.right > innerWidth + 1 || r.bottom > innerHeight + 1,
                  footerOutside: footer.bottom > innerHeight + 1 || footer.top < 0,
                  animated: document
                    .getAnimations()
                    .filter(
                      (a) =>
                        a.playState === 'running' &&
                        Number(a.effect?.getComputedTiming().duration) > 1
                    ).length
                };
              });
              probes.push({ seats, state, width, height, locale, theme, ...p });
              await page.screenshot({
                path: path.join(out, `${seats}p-${state}-${width}x${height}-${locale}-${theme}.png`)
              });
              expect(p).toEqual({
                crampedTitle: false,
                overflow: false,
                small: [],
                untipped: [],
                outside: false,
                footerOutside: false,
                animated: 0
              });
            }
            if (state === 'trade') {
              const options = sheet.locator('[data-pick^="build:"]');
              expect(await options.count()).toBeGreaterThan(1);
              await options.last().focus();
              await page.keyboard.press('Home');
              await expect(options.first()).toBeFocused();
              await expect(options.first()).toHaveAttribute('aria-checked', 'true');
            }
            if (state === 'chain' || state === 'free')
              await expect(
                sheet.locator('[aria-labelledby="hg-build-title"] [role="radiogroup"]')
              ).toHaveCount(0);
            if (state === 'blocked')
              await expect(sheet.locator('[data-cannot-build]')).toBeVisible();
            // Visit the entire scroll body, not just its initial viewport.
            await sheet
              .locator('[data-sheet-body]')
              .evaluate((body) => body.scrollTo(0, body.scrollHeight));
            await expect(sheet.locator('[data-act^="sell:"]')).toBeVisible();
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

test('card-sheet confirmation resets when the payment changes', async ({ page }) => {
  await page.goto('/dev/gallery?hgSheet=trade&seats=5');
  const sheet = page.locator('[data-card-sheet]');
  const action = sheet.locator('[data-act^="build:"]');
  const original = await action.getAttribute('data-act');
  await action.click();
  await expect(action).toHaveAttribute('data-armed', '');
  await sheet.locator('[data-pick^="build:"]').last().click();
  await expect(action).not.toHaveAttribute('data-act', original!);
  await expect(action).not.toHaveAttribute('data-armed');
  await action.click();
  await page.waitForTimeout(360); // the existing deliberate double-tap guard
  await action.click();
  await expect(sheet).toHaveCount(0);
  await expect(page.locator('[data-gallery-confirmed]')).toBeVisible();
});
