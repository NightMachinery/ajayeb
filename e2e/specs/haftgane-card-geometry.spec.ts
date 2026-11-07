import { expect } from '@playwright/test';
import { hgDesignSlugs } from '../../web/src/lib/wire/haftgane-catalog.gen';
import { test } from '../fixtures/base';

for (const locale of ['en', 'fa'] as const) {
  test.describe(`Haftgane card geometry ${locale}`, () => {
    test.use({ appLocale: locale });
    test('every effect fits the minimum phone card @mobile', async ({ page }) => {
      await page.goto('/dev/gallery');
      const cards = page.locator('[data-gallery-all-cards] [data-card]');
      await expect(cards).toHaveCount(hgDesignSlugs.length);
      // 72 px in portrait, 44 px in short landscape. Measure descendants,
      // since an overflow-hidden card can conceal a spilling effect group.
      for (const width of [72, 44]) {
        await cards.evaluateAll((cards, width) => {
          for (const card of cards) {
            const box = card.parentElement!;
            box.style.width = `${width}px`;
            box.style.height = `${width / 0.66}px`;
          }
        }, width);
        await page.evaluate(
          () =>
            new Promise<void>((resolve) =>
              requestAnimationFrame(() => requestAnimationFrame(() => resolve()))
            )
        );
        const clipped = await cards.evaluateAll((cards) =>
          cards.flatMap((card) => {
            const face = card.getBoundingClientRect();
            const band = card.querySelector('.band')!.getBoundingClientRect();
            const fits = (r: DOMRect, outer: DOMRect) =>
              r.left >= outer.left - 0.5 &&
              r.right <= outer.right + 0.5 &&
              r.top >= outer.top - 0.5 &&
              r.bottom <= outer.bottom + 0.5;
            const parts = [...card.querySelectorAll('[data-effect], [data-effect] *')];
            return parts
              .filter(
                (el) =>
                  !fits(el.getBoundingClientRect(), band) || !fits(el.getBoundingClientRect(), face)
              )
              .map(
                (el) =>
                  `${card.getAttribute('data-card')}: ${el.getAttribute('data-effect') ?? el.tagName}`
              );
          })
        );
        expect(clipped, `${width}px effects`).toEqual([]);
      }
    });
  });
}
