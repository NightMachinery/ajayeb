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
          .every((el) => el instanceof HTMLImageElement && el.complete && el.naturalWidth > 0)
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
              ratio: Number(img.getAttribute('width')) / Number(img.getAttribute('height'))
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
      if (pick.kind === 'side' || pick.kind === 'build-discarded') {
        await page.locator('[data-special-confirm]').click();
      } else await choose(page, 'confirm');
      if (pick.kind === 'stage') tucked = true;
      await expect.poll(async () => (await latest(page)).point?.ref.id).not.toBe(f.point!.ref.id);
    }
    expect([...seen]).toEqual([1, 2, 3]);
    expect(tucked, 'the live game built a stage with a concealed identity').toBe(true);
    expect(requests.some((url) => url.includes('/art/backs/guild'))).toBe(false);
    expect(problems).toEqual([]);
  });
});

// All seven real headers, including the longest name in both locales, at
// the smallest supported phone width. A single text range must stay on one
// line and inside its panel, so nowrap cannot hide an overflowing name.
for (const [locale, longest] of [
  ['en', 'Halikarnassos'],
  ['fa', 'هالیکارناسوس']
] as const) {
  test.describe(`wonder names ${locale}`, { tag: ['@full', '@haftgane-table'] }, () => {
    test.use({ appLocale: locale });
    test('all board names stay unbroken at 360px', async ({ page }) => {
      const problems = watchHygiene(page);
      await page.setViewportSize({ width: 360, height: 844 });
      await page.addInitScript(() => localStorage.setItem('ajayeb:name', 'Painter'));
      await page.goto('/dev/gallery');
      const boards = page.locator('[data-gallery-wonder-boards] [data-wonder-board]');
      await expect(boards).toHaveCount(7);
      await expect(boards.locator('[data-wonder-name]').filter({ hasText: longest })).toHaveCount(
        1
      );
      await page.evaluate(() => document.fonts.ready);
      for (const board of await boards.all()) {
        await board.scrollIntoViewIfNeeded();
        const name = board.locator('[data-wonder-name]');
        const probe = await name.evaluate((el) => {
          const range = document.createRange();
          range.selectNodeContents(el);
          const rects = [...range.getClientRects()].filter((r) => r.width > 0 && r.height > 0);
          const box = el.closest('[data-wonder-board]')!.getBoundingClientRect();
          return {
            text: el.textContent,
            lines: new Set(rects.map((r) => Math.round(r.top))).size,
            inside: rects.every((r) => r.left >= box.left && r.right <= box.right)
          };
        });
        expect(probe.lines, probe.text ?? '').toBe(1);
        expect(probe.inside, probe.text ?? '').toBe(true);
      }
      expect(problems).toEqual([]);
    });
  });
}
