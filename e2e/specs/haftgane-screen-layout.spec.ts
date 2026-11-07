import { expect } from '@playwright/test';
import { test } from '../fixtures/base';

for (const locale of ['en', 'fa'] as const) {
  test.describe(`Haftgane screen layout ${locale}`, () => {
    test.use({ appLocale: locale });
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
