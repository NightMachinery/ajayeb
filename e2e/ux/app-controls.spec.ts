// Captures of the app controls (M18.installable-and-fullscreen): Settings
// with the Install button (a stand-in for Chrome's install prompt) and the
// Fullscreen switch, Settings as iPhone Safari shows it (the Add to Home
// Screen steps, no Fullscreen), and the Duel and Haftgane game menus with
// their Fullscreen item. At 390x844 and 1280x800, light and dark, and once
// in Persian; the PNGs go to APP_SHOTS_DIR for the contact sheet.

import { mkdirSync } from 'node:fs';
import path from 'node:path';
import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { openHotseatRoom } from '../fixtures/scenario';
import { openBotTable } from '../fixtures/table';

const out = process.env.APP_SHOTS_DIR ?? '../.cache/e2e/app-shots';

function iphone(): void {
  const as = (value: unknown) => ({ get: () => value, configurable: true });
  Object.defineProperty(Navigator.prototype, 'standalone', as(false));
  Object.defineProperty(Navigator.prototype, 'maxTouchPoints', as(5));
  Object.defineProperty(Document.prototype, 'fullscreenEnabled', as(false));
}

function offerInstall(): void {
  (window as unknown as { __ajayebInstall: unknown }).__ajayebInstall = Object.assign(
    new Event('beforeinstallprompt'),
    { prompt: () => Promise.resolve() }
  );
}

async function settings(page: Page, row: string): Promise<void> {
  await page.goto('/settings');
  const el = page.locator(`[data-setting="${row}"]`);
  await el.scrollIntoViewIfNeeded();
  await expect(el).toBeVisible();
  await page.evaluate(() => document.fonts.ready.then(() => undefined));
}

const runs = [
  ...(['light', 'dark'] as const).flatMap((theme) =>
    (
      [
        [390, 844],
        [1280, 800]
      ] as const
    ).map(([width, height]) => ({ theme, width, height, locale: 'en' }))
  ),
  { theme: 'light', width: 390, height: 844, locale: 'fa' }
] as const;

for (const { theme, width, height, locale } of runs) {
  const tag = `${width}x${height}-${locale}-${theme}`;
  test.describe(`app controls ${tag}`, () => {
    test.use({ appLocale: locale, viewport: { width, height } });
    test.beforeEach(async ({ page }) => {
      mkdirSync(out, { recursive: true });
      await page.addInitScript((t) => localStorage.setItem('ajayeb:theme', t), theme);
    });

    test('settings with Install and Fullscreen', async ({ page }) => {
      await page.addInitScript(offerInstall);
      await settings(page, 'install');
      await expect(page.locator('[data-setting="fullscreen"]')).toBeVisible();
      await page.screenshot({ path: path.join(out, `settings-install-${tag}.png`) });
    });

    test('settings on iPhone Safari', async ({ page }) => {
      await page.addInitScript(iphone);
      await settings(page, 'install-ios');
      await page.screenshot({ path: path.join(out, `settings-ios-${tag}.png`) });
    });

    test('Duel game menu', async ({ page }) => {
      await openHotseatRoom(page, null);
      await page.locator('.game-screen [data-control="menu"]').click();
      await expect(page.locator('[data-control="menu-fullscreen"]')).toBeVisible();
      await page.screenshot({ path: path.join(out, `duel-menu-${tag}.png`) });
    });

    test('Haftgane table menu', async ({ page }) => {
      await openBotTable(page, 3, locale === 'fa' ? 'روشن' : 'Roshan');
      await page.locator('[data-act="table-menu"]').click();
      await expect(page.locator('[data-act="fullscreen"]')).toBeVisible();
      await page.screenshot({ path: path.join(out, `table-menu-${tag}.png`) });
    });
  });
}
