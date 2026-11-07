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
