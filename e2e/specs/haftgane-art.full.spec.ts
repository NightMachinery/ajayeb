// Live art acceptance with optional review captures. Never seed a guild draw:
// the complete public art catalog is independent of what the room dealt.
import { mkdirSync } from 'node:fs';
import path from 'node:path';
import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { choose, dismissMilitary, latest, openBotTable, ready } from '../fixtures/table';

async function images(page: Page) {
  await expect
    .poll(() =>
      page.locator('img').evaluateAll((els) =>
        els
          .filter((el) => {
            const r = el.getBoundingClientRect();
            return r.width && r.height && r.top < innerHeight && r.bottom > 0;
          })
          .every((el) => el.complete && el.naturalWidth > 0)
      )
    )
    .toBe(true);
}

test.describe('Haftgane art', { tag: ['@full', '@haftgane-table'] }, () => {
  test('approved card and intact wonder art through all three Ages', async ({ page }) => {
    test.setTimeout(180_000);
    const problems = watchHygiene(page);
    const requests: string[] = [];
    page.on('request', (r) => {
      if (r.url().includes('/art/')) requests.push(r.url());
    });
    await openBotTable(page, 5, 'Painter');
    const seen = new Set<number>();
    let tucked = false;
    const out = process.env.HG_ART_SHOTS_DIR;
    if (out) mkdirSync(out, { recursive: true });
    for (let move = 0; move < 25; move++) {
      await ready(page);
      await dismissMilitary(page);
      const f = await latest(page);
      if (f.status === 'finished') break;
      if (f.view.phase === 'turn' && !seen.has(f.view.age)) {
        seen.add(f.view.age);
        const variants = out
          ? ([
              ['light', 390, 844],
              ['dark', 390, 844],
              ['light', 844, 390],
              ['dark', 844, 390],
              ['light', 1280, 800],
              ['dark', 1280, 800]
            ] as const)
          : ([['light', 390, 844]] as const);
        for (const [theme, width, height] of variants) {
          await page.setViewportSize({ width, height });
          await page.evaluate((value) => {
            document.documentElement.dataset.theme = value;
          }, theme);
          await images(page);
          expect(await page.locator('[data-card-art] img').count()).toBeGreaterThan(0);
          expect(
            await page.evaluate(() => document.documentElement.scrollWidth > innerWidth + 1)
          ).toBe(false);
          if (out)
            await page.screenshot({
              path: path.join(out, `age-${f.view.age}-${width}x${height}-${theme}.png`)
            });
          await page.locator(`[data-act="city:${f.you.seat}"]`).click();
          const board = page.locator('[data-wonder-board]');
          await expect(board).toHaveAttribute(
            'data-wonder-board',
            f.view.seats[f.you.seat!]!.board
          );
          await expect(board.locator('.painting img')).toBeVisible();
          await images(page);
          const picture = await board
            .locator('.painting img')
            .evaluate((img: HTMLImageElement) => ({
              fit: getComputedStyle(img).objectFit,
              ratio: img.naturalWidth / img.naturalHeight
            }));
          expect(picture).toEqual({ fit: 'contain', ratio: 1.5 });
          await expect(board.locator('[data-stage-built="true"]')).toHaveCount(
            f.view.seats[f.you.seat!]!.stages
          );
          expect(await board.locator('.tuck').count()).toBe(f.view.seats[f.you.seat!]!.stages);
          if (out)
            await page.screenshot({
              path: path.join(out, `wonder-age-${f.view.age}-${width}x${height}-${theme}.png`)
            });
          await page.locator('[data-act="city-close"]').click();
        }
      }
      const choices = f.view.you!.choices;
      const pick =
        choices.find((c) => c.kind === 'stage') ??
        choices.find((c) => c.kind === 'build') ??
        choices[0]!;
      await choose(page, pick.key);
      await choose(page, 'confirm');
      if (pick.kind === 'stage') tucked = true;
      await expect.poll(async () => (await latest(page)).point?.ref.id).not.toBe(f.point!.ref.id);
    }
    expect([...seen]).toEqual([1, 2, 3]);
    expect(tucked, 'the live game built a stage with a concealed identity').toBe(true);
    expect(requests.some((url) => url.includes('/art/backs/guild'))).toBe(false);
    expect(problems).toEqual([]);
  });
});
