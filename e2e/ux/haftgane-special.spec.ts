// Legal-game ViewFor captures: all special choices, own and waiting views.
import { expect } from '@playwright/test';
import { mkdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { test } from '../fixtures/base';
import { quiet } from '../fixtures/layout-probes';
import { watchHygiene } from '../fixtures/hygiene';

for (const locale of ['en', 'fa'] as const)
  for (const theme of ['light', 'dark'] as const) {
    test.describe(`special decisions ${locale} ${theme}`, () => {
      test.use({ appLocale: locale });
      test('special-decision matrix', async ({ page }) => {
        test.setTimeout(600_000);
        const out = process.env.HG_SPECIAL_SHOTS_DIR ?? '../.cache/e2e/hg-special-shots';
        mkdirSync(out, { recursive: true });
        const problems = watchHygiene(page);
        await page.addInitScript((t) => localStorage.setItem('ajayeb:theme', t), theme);
        const probes = [];
        for (const seats of [3, 5, 7])
          for (const state of [
            'side',
            'discard',
            'sixth',
            'kept',
            'last',
            'first-colour',
            'first-turn',
            'last-turn'
          ])
            for (const waiting of [false, true]) {
              await page.goto(
                `/dev/gallery?hgSpecial=${state}&seats=${seats}&waiting=${waiting ? 1 : 0}`
              );
              await expect(page.locator('[data-gallery-special]')).toBeVisible();
              if (!waiting && (state === 'side' || state === 'discard')) {
                const sheet = page.locator(
                  state === 'side' ? '[data-side-picker]' : '[data-discard-picker]'
                );
                await expect(sheet).toBeVisible();
                const pick = sheet
                  .locator('[data-pick], button[data-card]:not([aria-disabled="true"])')
                  .first();
                await pick.click();
                await expect(sheet.locator('[data-special-confirm]')).toBeEnabled();
              }
              for (const [width, height] of [
                [360, 740],
                [390, 844],
                [844, 390],
                [1280, 800]
              ]) {
                await page.setViewportSize({ width: width!, height: height! });
                await quiet(page);
                const probe = await page.evaluate(() => {
                  const root =
                    document.querySelector<HTMLElement>('dialog[open]') ??
                    document.querySelector<HTMLElement>('[data-gallery-special]')!;
                  const r = root.getBoundingClientRect();
                  return {
                    overflow:
                      document.documentElement.scrollWidth > innerWidth + 1 ||
                      root.scrollWidth > root.clientWidth + 1,
                    outside:
                      r.x < -1 ||
                      r.y < -1 ||
                      r.right > innerWidth + 1 ||
                      r.bottom > innerHeight + 1,
                    small: [...root.querySelectorAll<HTMLElement>('button:not(:disabled)')]
                      .filter((e) => e.checkVisibility())
                      .map((e) => ({
                        key: e.dataset['act'],
                        w: e.getBoundingClientRect().width,
                        h: e.getBoundingClientRect().height
                      }))
                      .filter((e) => e.w < 43.9 || e.h < 43.9),
                    untipped: [...root.querySelectorAll('svg, [role="img"]')]
                      .filter(
                        (e) =>
                          e.checkVisibility() &&
                          !e.parentElement?.closest('svg') &&
                          !e.closest('[data-tooltip]')
                      )
                      .map((e) => e.outerHTML.slice(0, 100)),
                    animated: document
                      .getAnimations()
                      .filter(
                        (a) =>
                          a.playState === 'running' &&
                          Number(a.effect?.getComputedTiming().duration) > 1
                      ).length
                  };
                });
                probes.push({ seats, state, waiting, width, height, ...probe });
                await page.screenshot({
                  path: path.join(
                    out,
                    `${seats}p-${state}-${waiting ? 'waiting' : 'own'}-${width}x${height}-${locale}-${theme}.png`
                  )
                });
                expect(probe).toEqual({
                  overflow: false,
                  outside: false,
                  small: [],
                  untipped: [],
                  animated: 0
                });
              }
              if (!waiting && (state === 'side' || state === 'discard')) {
                await page.locator('[data-special-confirm]').click();
                await expect(page.locator('[data-gallery-confirmed]')).toBeVisible();
                await expect(page.locator('dialog[open]')).toHaveCount(0);
              }
              if (waiting)
                await expect(
                  page.locator('[data-discard-picker], [data-side-picker], [data-card-sheet]')
                ).toHaveCount(0);
              if (!waiting && state === 'last') {
                await page.locator('[data-kept-card] button').click();
                await expect(page.locator('[data-card-sheet]')).toBeVisible();
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
