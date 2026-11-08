// A grouped resource/science summary cannot satisfy the symbol contract.
// Exercise the actual gesture and game-specific More in Rules link in
// both languages, with mobile using touch and desktop using hover.
import { expect, type Locator, type Page } from '@playwright/test';
import { mkdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { latest, openBotTable } from '../fixtures/table';

async function missing(page: Page): Promise<string[]> {
  return page
    .locator('[data-table-screen]')
    .evaluate((root) =>
      [...root.querySelectorAll('[data-res], [data-science-symbol], [data-effect-symbol]')]
        .filter((el) => el.getClientRects().length && !el.hasAttribute('data-tooltip'))
        .map(
          (el) =>
            el.getAttribute('data-res') ??
            el.getAttribute('data-science-symbol') ??
            el.getAttribute('data-effect-symbol') ??
            '?'
        )
    );
}

async function show(anchor: Locator, touch: boolean): Promise<void> {
  if (touch) await anchor.tap();
  else await anchor.hover();
}

for (const locale of ['en', 'fa'] as const) {
  test.describe(
    `Haftgane symbol tooltips ${locale}`,
    { tag: ['@full', '@mobile', '@haftgane-tooltips'] },
    () => {
      test.use({ appLocale: locale });
      test('each symbol explains itself and opens Haftgane rules', async ({ page, isMobile }) => {
        const problems = watchHygiene(page);
        const code = await openBotTable(page, 5);
        const frame = await latest(page);
        const own = frame.view.seats.find((s) => s.seat === frame.you.seat)!;
        const bubble = page.locator('[data-tooltip-bubble]');
        const names =
          locale === 'en' ? ['Compass', 'Gear', 'Tablet'] : ['پرگار', 'چرخ‌دنده', 'لوح'];
        const symbols = ['compass', 'gear', 'tablet'];
        expect(await missing(page)).toEqual([]);
        // Positive control: removing just the symbol's marker must fail,
        // even though its group's tooltip is still present.
        const target = page.locator('[data-science-symbol="compass"]');
        await target.evaluate((el) => el.removeAttribute('data-tooltip'));
        expect(await missing(page)).toContain('compass');
        await target.evaluate((el) => el.setAttribute('data-tooltip', ''));
        const resource = page.locator('[data-making] [data-res]').first();
        const res = (await resource.getAttribute('data-res'))!;
        await resource.evaluate((el) => el.removeAttribute('data-tooltip'));
        expect(await missing(page)).toContain(res);
        await resource.evaluate((el) => el.setAttribute('data-tooltip', ''));
        for (const [i, symbol] of symbols.entries()) {
          const anchor = page.locator(`[data-science-symbol="${symbol}"]`);
          await show(anchor, isMobile);
          await expect(bubble).toBeVisible();
          await expect(bubble).toContainText(names[i]!);
          await expect(bubble).toContainText(
            new Intl.NumberFormat(locale).format(own.science.counts[i] ?? 0)
          );
          for (const other of names.filter((_, j) => i !== j))
            await expect(bubble).not.toContainText(other);
          await expect(bubble.locator('[data-tooltip-rules]')).toHaveAttribute(
            'data-tooltip-rules',
            'science'
          );
          await page.keyboard.press('Escape');
        }
        await show(resource, isMobile);
        await expect(bubble).toHaveAttribute('data-rules-id', 'resources');
        await expect(bubble).toContainText(new Intl.NumberFormat(locale).format(1));
        await page.locator('[data-tooltip-rules="resources"]').click();
        await expect(page.locator('[data-rules-panel="haftgane"]')).toBeVisible();
        await expect(
          page.locator('[data-rules-game="haftgane"] [data-rules-section="resources"]')
        ).toBeInViewport();
        await page.locator('[data-rules-panel="haftgane"] [data-control="rules-close"]').click();
        await page.locator(`[data-act="city:${frame.you.seat}"]`).click();
        const city = page.locator('[data-city-sheet]');
        await expect(city).toBeVisible();
        for (const [i, symbol] of symbols.entries()) {
          const anchor = city.locator(`[data-science-symbol="${symbol}"]`);
          await expect(anchor).toHaveAttribute('data-tooltip');
          await show(anchor, isMobile);
          await expect(bubble).toContainText(names[i]!);
          for (const other of names.filter((_, j) => i !== j))
            await expect(bubble).not.toContainText(other);
          await expect(bubble).toHaveAttribute('data-rules-id', 'science');
          await page.keyboard.press('Escape');
        }
        await city.locator('[data-act="city-close"]').click();
        // Preserve one live room identity for the standalone tools/chrome capture.
        const out = process.env.HG_SYMBOL_SHOTS_DIR;
        if (out && locale === 'en' && !isMobile) {
          mkdirSync(out, { recursive: true });
          writeFileSync(
            path.join(out, 'room.json'),
            JSON.stringify({
              code,
              url: page.url(),
              token: await page.evaluate(() => localStorage.getItem('ajayeb:token'))
            })
          );
        }
        expect(problems).toEqual([]);
      });
    }
  );
}
