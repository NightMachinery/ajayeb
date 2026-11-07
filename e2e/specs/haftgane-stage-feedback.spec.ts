import { expect } from '@playwright/test';
import { test } from '../fixtures/base';

for (const locale of ['en', 'fa'] as const) {
  test.describe(`Haftgane stage feedback ${locale}`, () => {
    test.use({ appLocale: locale });
    test('unaffordable stage shows its server quote and shortfall @mobile', async ({ page }) => {
      // Legal ViewFor capture: stage 1, reason insufficient-coins,
      // cheapest 4, shortBy 1. No synthetic payment or client arithmetic.
      await page.goto('/dev/gallery?hgSheet=stage-shortfall&seats=3');
      const section = page.locator('[aria-labelledby="hg-stage-title"]');
      await expect(section.locator('[data-cost]')).toBeAttached();
      await expect(section.locator('[data-cannot-stage="insufficient-coins"]')).toHaveText(
        locale === 'en'
          ? 'Short by 1 coins. Cheapest payment: 4.'
          : '۱ سکه کم دارید. ارزان‌ترین پرداخت: ۴.'
      );
      await expect(section.locator('[data-pick]')).toHaveCount(0);
      await expect(page.locator('[data-card-sheet] .sheet-footer [data-kind="stage"]')).toHaveCount(
        0
      );
    });
    test('production shortfall retains the unavailable stage cost @mobile', async ({ page }) => {
      await page.goto('/dev/gallery?hgSheet=blocked&seats=3');
      const section = page.locator('[aria-labelledby="hg-stage-title"]');
      await expect(section.locator('[data-cost]')).toBeAttached();
      await expect(section.locator('[data-cannot-stage="cannot-cover"]')).toHaveText(
        locale === 'en'
          ? 'Your production and your neighbours cannot cover this cost.'
          : 'تولید شما و همسایه‌هایتان نمی‌تواند این هزینه را بپوشاند.'
      );
    });
  });
}
