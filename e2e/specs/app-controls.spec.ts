// Installing and fullscreen (M18.installable-and-fullscreen). The manifest
// the Go binary serves has what Chrome on Android and Safari on iOS read,
// and Chromium's own installability check finds nothing wrong with it but
// the off-the-record context Playwright always plays in. Settings offers
// Install only while the browser holds an install prompt, shows the Add to
// Home Screen steps on iOS instead, and nothing in an installed app. The
// Fullscreen toggle of Settings and the game menus shows where the API
// exists, keeps fullscreen across the game's own navigation, and leaves it.
//
// A real profile is the one place Chromium fires beforeinstallprompt, and
// a second browser is not allowed beside the worker's own, so that probe
// runs only when asked for alone: AJAYEB_INSTALL_PROBE=1 make e2e
// GREP=install-probe (docs/testing.md).

import { mkdtempSync, rmSync } from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { openHotseatRoom } from '../fixtures/scenario';

interface Icon {
  src: string;
  sizes: string;
  type: string;
  purpose: string;
}

/** Makes the page look like iPhone Safari: navigator.standalone, a touch
 *  screen, and no element fullscreen. `installed` is a home screen launch. */
function iphone(installed: boolean): void {
  const as = (value: unknown) => ({ get: () => value, configurable: true });
  Object.defineProperty(Navigator.prototype, 'standalone', as(installed));
  Object.defineProperty(Navigator.prototype, 'maxTouchPoints', as(5));
  Object.defineProperty(Document.prototype, 'fullscreenEnabled', as(false));
}

/** Hands the page a stand-in for Chrome's install prompt event, as app.html
 *  keeps it, and counts the prompts. */
function offerInstall(): void {
  const w = window as unknown as { __ajayebInstall: unknown; __prompts: number };
  w.__prompts = 0;
  w.__ajayebInstall = Object.assign(new Event('beforeinstallprompt'), {
    prompt: () => {
      w.__prompts++;
      return Promise.resolve();
    }
  });
}

const isFullscreen = (page: Page) => page.evaluate(() => document.fullscreenElement !== null);

async function openSettings(page: Page): Promise<void> {
  await page.goto('/settings');
  await expect(page.locator('[data-setting="language"]')).toBeVisible();
}

test.describe('installable', { tag: '@mobile' }, () => {
  test.use({ serviceWorkers: 'allow' });

  test('the served manifest has every member installation needs', async ({ request }) => {
    const res = await request.get('/manifest.webmanifest');
    expect(res.status()).toBe(200);
    const manifest = JSON.parse(await res.text()) as Record<string, unknown> & { icons: Icon[] };
    expect(manifest).toMatchObject({
      id: '/',
      name: 'Ajayeb',
      short_name: 'Ajayeb',
      start_url: '/',
      scope: '/',
      display: 'standalone'
    });
    expect(manifest.theme_color).toMatch(/^#[0-9a-f]{6}$/);
    expect(manifest.background_color).toMatch(/^#[0-9a-f]{6}$/);
    const icons = manifest.icons.map((i) => `${i.purpose} ${i.sizes}`).sort();
    expect(icons).toEqual(['any 192x192', 'any 512x512', 'maskable 512x512']);
    for (const icon of manifest.icons) {
      const png = await request.get(icon.src);
      expect(png.status(), icon.src).toBe(200);
      expect(png.headers()['content-type'], icon.src).toBe('image/png');
      const b = await png.body();
      expect(`${b.readUInt32BE(16)}x${b.readUInt32BE(20)}`, icon.src).toBe(icon.sizes);
    }
  });

  test("Chromium's installability check finds only the off-the-record context", async ({
    page
  }) => {
    await page.goto('/');
    await page.evaluate(async () => {
      await navigator.serviceWorker.ready;
    });
    await page.reload();
    const cdp = await page.context().newCDPSession(page);
    const manifest = await cdp.send('Page.getAppManifest');
    expect(manifest.url).toMatch(/\/manifest\.webmanifest$/);
    expect(manifest.errors).toEqual([]);
    const { installabilityErrors } = await cdp.send('Page.getInstallabilityErrors');
    const ids = installabilityErrors.map((e) => e.errorId);
    console.log('installability', JSON.stringify({ ids, manifest: manifest.url }));
    expect(ids.filter((id) => id !== 'in-incognito')).toEqual([]);
  });

  test('install-probe: a real profile gets beforeinstallprompt', async ({
    playwright,
    baseURL
  }) => {
    test.skip(!process.env.AJAYEB_INSTALL_PROBE, 'opt-in: needs its own browser');
    const dir = mkdtempSync(path.join(os.tmpdir(), 'playwright_chromiumdev_profile-install-'));
    const context = await playwright.chromium.launchPersistentContext(dir, {
      channel: 'chromium',
      baseURL,
      args: ['--bypass-app-banner-engagement-checks', '--mute-audio']
    });
    try {
      const page = context.pages()[0] ?? (await context.newPage());
      await page.goto('/');
      await expect
        .poll(
          () => page.evaluate(() => !!(window as { __ajayebInstall?: unknown }).__ajayebInstall),
          {
            timeout: 20_000
          }
        )
        .toBe(true);
      const cdp = await context.newCDPSession(page);
      const { installabilityErrors } = await cdp.send('Page.getInstallabilityErrors');
      console.log('install-probe', JSON.stringify(installabilityErrors));
      expect(installabilityErrors).toEqual([]);
      await openSettings(page);
      await expect(page.locator('[data-control="install"]')).toBeVisible();
    } finally {
      await context.close();
      rmSync(dir, { recursive: true, force: true });
    }
  });
});

test.describe('the Install row', { tag: '@mobile' }, () => {
  test('stays hidden while the browser offers no installation', async ({ page }) => {
    await openSettings(page);
    await expect(page.locator('[data-setting="install"]')).toHaveCount(0);
    await expect(page.locator('[data-setting="install-ios"]')).toHaveCount(0);
  });

  test('prompts once when the browser offers installation', async ({ page }) => {
    await page.addInitScript(offerInstall);
    await openSettings(page);
    await page.locator('[data-control="install"]').click();
    expect(await page.evaluate(() => (window as unknown as { __prompts: number }).__prompts)).toBe(
      1
    );
    await expect(page.locator('[data-setting="install"]')).toHaveCount(0);
  });

  test('shows the Add to Home Screen steps on iOS, and nothing once installed', async ({
    page
  }) => {
    await page.addInitScript(iphone, false);
    await openSettings(page);
    await expect(page.locator('[data-setting="install-ios"] li')).toHaveCount(2);
    await expect(page.locator('[data-control="install"]')).toHaveCount(0);
    await expect(page.locator('[data-setting="fullscreen"]')).toHaveCount(0);

    await page.addInitScript(iphone, true);
    await page.reload();
    await expect(page.locator('[data-setting="language"]')).toBeVisible();
    await expect(page.locator('[data-setting="install-ios"]')).toHaveCount(0);
  });
});

test.describe('the Fullscreen toggle', { tag: '@mobile' }, () => {
  test('keeps fullscreen across the game and Home, and leaves it', async ({ page }) => {
    await openHotseatRoom(page, null);
    const menu = page.locator('.game-screen [data-control="menu"]');
    await menu.click();
    const item = page.locator('[data-control="menu-fullscreen"]');
    await expect(item).toHaveText('Full screen');
    await item.click();
    await expect.poll(() => isFullscreen(page)).toBe(true);
    await menu.click();
    await expect(item).toHaveText('Exit full screen');

    await page.locator('[data-control="menu-home"]').click();
    await expect(page.getByRole('heading', { level: 1, name: 'Ajayeb' })).toBeVisible();
    expect(await isFullscreen(page)).toBe(true);
    await page.getByRole('button', { name: 'Settings', exact: true }).click();
    const box = page.locator('[data-setting="fullscreen"]');
    await expect(box).toBeChecked();
    await box.click();
    await expect.poll(() => isFullscreen(page)).toBe(false);
    await expect(box).not.toBeChecked();
  });

  test('is hidden where the Fullscreen API is missing', async ({ page }) => {
    await page.addInitScript(() =>
      Object.defineProperty(Document.prototype, 'fullscreenEnabled', { get: () => false })
    );
    await openSettings(page);
    await expect(page.locator('[data-setting="fullscreen"]')).toHaveCount(0);
    await openHotseatRoom(page, null);
    await page.locator('.game-screen [data-control="menu"]').click();
    await expect(page.locator('[data-control="menu-home"]')).toBeVisible();
    await expect(page.locator('[data-control="menu-fullscreen"]')).toHaveCount(0);
  });
});
