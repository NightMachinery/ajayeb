// Small UI follow-ups: keep server summaries current and narrow layouts readable.
import { expect } from '@playwright/test';
import { test } from '../fixtures/base';
import { GameDriver } from '../fixtures/driver';
import { watchHygiene } from '../fixtures/hygiene';
import { settle } from '../fixtures/layout-probes';
import { playKey } from '../fixtures/scenario';

test.describe('small UI follow-ups', { tag: '@mobile' }, () => {
  test('Home receives the level and age, then removes a room finished in another tab', async ({
    page
  }) => {
    const problems = watchHygiene(page);
    await page.goto('/');
    const other = await page.context().newPage();
    const otherProblems = watchHygiene(other);
    try {
      await other.goto('/dev/new?mode=ai&level=easy&seat=0&seed=7&scenario=military-supremacy');
      await other.waitForURL(/\/play\/[A-Z0-9]+$/);
      const driver = new GameDriver(other);
      await driver.waitReady();
      const code = await other.evaluate(() => window.__ajayeb?.room()?.code);
      const entry = page.getByRole('button', { name: `Continue: vs AI · ${code}` });
      await expect(entry.locator('[data-continue-details]')).toHaveText('Apprentice · Age II');
      // Home has never opened a socket. This update must come from storage.
      expect(await page.evaluate(() => window.__ajayeb?.room())).toBeUndefined();
      await playKey(driver, 'construct:walls');
      await expect
        .poll(() => other.evaluate(() => window.__ajayeb?.view()?.phase))
        .toBe('gameOver');
      await expect(entry).toHaveCount(0);
      await page.reload();
      await expect(page.getByRole('button', { name: /^Continue:/ })).toHaveCount(0);
      expect(otherProblems).toEqual([]);
    } finally {
      await other.close();
    }
    expect(problems).toEqual([]);
  });

  test('the fixture link query preserves the board and disables moves', async ({ page }) => {
    const problems = watchHygiene(page);
    await page.goto('/dev/fixture/age-1-start-seat0?link=offline');
    await expect(page.locator('html')).toHaveAttribute('data-ready', '1');
    await expect(page.locator('.card-structure')).toBeVisible();
    await expect(
      page.getByRole('status').filter({ hasText: 'Offline. Trying to reconnect' }).first()
    ).toBeVisible();
    expect(await page.evaluate(() => window.__ajayeb?.ready())).toBe(false);
    expect(problems).toEqual([]);
  });
});

for (const viewport of [
  { width: 360, height: 740 },
  { width: 390, height: 844 },
  { width: 740, height: 360 },
  { width: 1280, height: 800 }
]) {
  test(`small UI layouts at ${viewport.width}x${viewport.height}`, async ({ page }, info) => {
    const problems = watchHygiene(page);
    await page.setViewportSize(viewport);
    await page.goto('/dev/fixture/online-age-start-seat0');
    await settle(page);
    await expect(page.locator('[data-peek-bar="inline"]')).toBeVisible();
    const overlaps = await page.locator('[data-peek-bar]').evaluate((bar) => {
      const b = bar.getBoundingClientRect();
      return [...document.querySelectorAll('.card-structure [data-card][data-accessible]')].filter(
        (card) => {
          const c = card.getBoundingClientRect();
          return c.left < b.right && c.right > b.left && c.top < b.bottom && c.bottom > b.top;
        }
      ).length;
    });
    expect(overlaps).toBe(0);
    await page.screenshot({ path: info.outputPath(`age-start-${viewport.width}.png`) });

    await page.goto('/dev/fixture/online-away-seat0');
    await settle(page);
    if (viewport.width === 1280) {
      await expect(page.locator('.city-panel .presence')).toHaveCount(1);
      await expect(page.locator('.city-panel .presence[data-presence="away"]')).toBeVisible();
    }
    await page.screenshot({ path: info.outputPath(`presence-${viewport.width}.png`) });

    await page.goto(
      '/dev/gallery?art=wonder/statue-of-zeus,wonder/temple-of-artemis,wonder/hanging-gardens,wonder/great-library'
    );
    const gallery = page.locator('[data-art-frames]');
    await gallery.scrollIntoViewIfNeeded();
    await page.evaluate(() => document.fonts.ready);
    await expect(gallery.getByRole('heading', { name: 'Art frames' })).toBeVisible();
    const clipped = await gallery
      .locator('.wonder-tile[data-size="city"] .font-display')
      .evaluateAll((names) =>
        names
          .filter((name) => name.scrollHeight > name.clientHeight + 1)
          .map((name) => name.textContent)
      );
    expect(clipped).toEqual([]);
    await page.locator('[data-art-sample="wonder/statue-of-zeus"]').scrollIntoViewIfNeeded();
    await page.screenshot({ path: info.outputPath(`wonder-gallery-${viewport.width}.png`) });

    await page.goto('/dev/new?mode=ai&level=easy&seat=0&seed=7&scenario=military-supremacy');
    await page.waitForURL(/\/play\/[A-Z0-9]+$/);
    const driver = new GameDriver(page);
    await driver.waitReady();
    await playKey(driver, 'construct:walls');
    const table = page.locator('[data-score-table]');
    await expect(table).toBeVisible();
    const name = table.locator('thead [dir="auto"]').filter({ hasText: 'Apprentice' });
    await expect(name).toHaveText('Apprentice');
    expect(
      await name.evaluate(
        (el) => el.scrollWidth <= el.clientWidth && el.scrollHeight <= el.clientHeight + 1
      )
    ).toBe(true);
    await table.screenshot({ path: info.outputPath(`score-table-${viewport.width}.png`) });
    expect(problems).toEqual([]);
  });
}
