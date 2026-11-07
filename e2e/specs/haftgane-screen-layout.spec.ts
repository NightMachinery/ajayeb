import { expect } from '@playwright/test';
import { test } from '../fixtures/base';
import { openBotTable } from '../fixtures/table';

for (const locale of ['en', 'fa'] as const) {
  test.describe(`Haftgane screen layout ${locale}`, () => {
    test.use({ appLocale: locale });
    test('generated bot numbers stay visible in seats and city tabs @mobile', async ({ page }) => {
      await page.setViewportSize({ width: 390, height: 844 });
      await openBotTable(page, 7, 'Human 42');
      const suffixes = page.locator('[data-table-screen] [data-bot-suffix]');
      await expect(suffixes).toHaveCount(5);
      async function clipped(scope: typeof suffixes): Promise<number> {
        return scope.evaluateAll(
          (els) =>
            els.filter((el) => {
              const r = el.getBoundingClientRect();
              const name = el.parentElement!.parentElement!.getBoundingClientRect();
              const status = el
                .closest('[data-seat]')
                ?.querySelector('[data-status]')
                ?.getBoundingClientRect();
              const covered =
                status &&
                Math.min(r.right, status.right) > Math.max(r.left, status.left) &&
                Math.min(r.bottom, status.bottom) > Math.max(r.top, status.top);
              return (
                r.width === 0 || r.left < name.left - 0.5 || r.right > name.right + 0.5 || !!covered
              );
            }).length
        );
      }
      expect(await clipped(suffixes)).toBe(0);
      await page.locator('[data-relation="left"] [data-act^="city:"]').first().click();
      const tabs = page.locator('[data-city-sheet] [data-bot-suffix]');
      await expect(tabs).toHaveCount(5);
      expect(await clipped(tabs)).toBe(0);
      const heading = page.locator('#hg-city-title');
      expect(await heading.evaluate((el) => el.scrollWidth <= el.clientWidth + 1)).toBe(true);
      await expect(heading).toHaveAttribute('dir', 'auto');
    });
    for (const state of ['kept', 'last']) {
      test(`kept status clears the card name: ${state} @mobile`, async ({ page }) => {
        await page.goto(`/dev/gallery?hgSpecial=${state}&seats=7`);
        const kept = page.locator('[data-kept-card]');
        await expect(kept).toBeVisible();
        const name = await kept.locator('[data-card] .name').boundingBox();
        const status = await kept.locator('p').boundingBox();
        expect(status!.y).toBeGreaterThanOrEqual(name!.y + name!.height);
      });
    }
    test('card sheets and Seen hands keep card anatomy left to right @mobile', async ({ page }) => {
      await page.goto('/dev/gallery?hgSheet=chain&seats=7');
      const face = page.locator('[data-card-sheet] [data-card]');
      await expect(face).toBeVisible();
      expect(await face.evaluate((el) => getComputedStyle(el).direction)).toBe('ltr');
      expect(await face.locator('.name').evaluate((el) => getComputedStyle(el).direction)).toBe(
        locale === 'fa' ? 'rtl' : 'ltr'
      );
      const cost = await face.locator('.cost').boundingBox();
      const chains = await face.locator('.chains').boundingBox();
      expect(cost!.x).toBeLessThan(chains!.x);
      await page.goto('/dev/gallery');
      await page
        .locator('[data-gallery-city] .open')
        .filter({ hasText: 'Open Seen hands, Age II' })
        .click();
      const mini = page.locator('[data-seen-sheet] .mini').first();
      await expect(mini).toBeVisible();
      expect(await mini.evaluate((el) => getComputedStyle(el).direction)).toBe('ltr');
      const band = await mini.locator('.band').boundingBox();
      const name = await mini.locator('.name').boundingBox();
      expect(band!.x).toBeLessThan(name!.x);
      expect(await mini.locator('.name').evaluate((el) => getComputedStyle(el).direction)).toBe(
        locale === 'fa' ? 'rtl' : 'ltr'
      );
    });
    test('conflict panels retain physical neighbour positions @mobile', async ({ page }) => {
      await page.goto('/dev/gallery?hgEnd=military&seats=7');
      const left = page.locator('[data-my-conflict="left"]');
      const right = page.locator('[data-my-conflict="right"]');
      await expect(left).toBeVisible();
      expect((await left.boundingBox())!.x).toBeLessThan((await right.boundingBox())!.x);
      for (const panel of [left, right]) {
        expect(await panel.evaluate((el) => getComputedStyle(el).direction)).toBe(
          locale === 'fa' ? 'rtl' : 'ltr'
        );
      }
    });
  });
}
