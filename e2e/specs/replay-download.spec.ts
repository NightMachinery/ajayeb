// Replay availability uses the server's revealed view; fresh visitors may
// download only the finished logs that the existing HTTP endpoint allows.
import { readFile } from 'node:fs/promises';
import { expect } from '@playwright/test';
import { test } from '../fixtures/base';
import { GameDriver } from '../fixtures/driver';
import { currentView } from '../fixtures/scenario';
import { watchHygiene } from '../fixtures/hygiene';
import { newDeviceContext } from '../fixtures/online';

for (const hidden of [false, true]) {
  test(
    `finished spectator replay: hidden=${hidden}`,
    { tag: '@mobile' },
    async ({ page, browser }) => {
      const problems = watchHygiene(page);
      const response = await page.request.post('/api/dev/rooms', {
        data: {
          mode: 'online',
          seed: '7',
          scenario: 'military-supremacy',
          actions: ['construct:walls'],
          options: { hideAtEnd: hidden }
        }
      });
      expect(response.ok()).toBe(true);
      const room = (await response.json()) as { code: string; path: string };
      // Claim both open seats on separate devices, then visit without either identity.
      for (const name of ['Host', 'Guest']) {
        const context = await newDeviceContext(browser, test.info());
        try {
          await context.addInitScript((name) => localStorage.setItem('ajayeb:name', name), name);
          const occupant = await context.newPage();
          await occupant.goto(room.path);
          await new GameDriver(occupant).waitReady();
        } finally {
          await context.close();
        }
      }
      await page.addInitScript(() => localStorage.setItem('ajayeb:name', 'Observer'));
      await page.goto(room.path);
      const driver = new GameDriver(page);
      await driver.waitReady();
      expect(await page.evaluate(() => window.__ajayeb?.you()?.seats)).toEqual([]);
      expect((await currentView(page)).phase).toBe('gameOver');
      const result = page
        .locator('dialog[open]')
        .filter({ has: page.locator('[data-score-table]') });
      await expect(result).toBeVisible();
      const logResponse = await page.request.get(`/api/rooms/${room.code}/log`);
      if (hidden) {
        expect(logResponse.status()).toBe(403);
        expect(await logResponse.text()).not.toContain('"seed"');
        await expect(page.locator('[data-download-replay]')).toHaveCount(0);
        await expect(page.locator('[data-replay-hidden]')).toHaveText(
          'This game keeps its record hidden after the end.'
        );
      } else {
        expect(logResponse.status()).toBe(200);
        const [download] = await Promise.all([
          page.waitForEvent('download'),
          page.locator('[data-download-replay]').filter({ visible: true }).click()
        ]);
        expect(download.suggestedFilename()).toBe(
          `ajayeb-${room.code}-${new Date().toISOString().slice(0, 10)}.json`
        );
        expect(await readFile(await download.path())).toEqual(await logResponse.body());
      }
      // These four views cover the added result CTA and scrollable game menu.
      const sizes = [
        [360, 740],
        [390, 844],
        [740, 360],
        [1280, 800]
      ] as const;
      for (const [width, height] of hidden ? [sizes[0]] : sizes) {
        await page.setViewportSize({ width, height });
        await page.screenshot({ path: test.info().outputPath(`result-${width}x${height}.png`) });
        if (!hidden) {
          const cta = page.locator('[data-download-replay]').filter({ visible: true });
          await cta.scrollIntoViewIfNeeded();
          await expect(cta).toBeInViewport();
          await page.screenshot({
            path: test.info().outputPath(`result-actions-${width}x${height}.png`)
          });
        }
        await page.locator('[data-view-board]').filter({ visible: true }).click();
        await driver.visible('[data-control="menu"]').click();
        const replay = driver.visible('[data-control="menu-replay"]');
        if (hidden) {
          await expect(replay).toBeDisabled();
          await page.keyboard.press('Tab');
          await driver.visible('[data-replay-tooltip]').focus();
          await expect(page.locator('#ajayeb-tooltip')).toHaveText(
            'This game keeps its record hidden after the end.'
          );
          await page.keyboard.press('Escape');
        } else {
          await expect(replay).toBeEnabled();
        }
        await replay.scrollIntoViewIfNeeded();
        await page.screenshot({ path: test.info().outputPath(`menu-${width}x${height}.png`) });
        await page.keyboard.press('Escape');
        await page
          .locator('[data-game-over] [data-peek-bar] button')
          .filter({ visible: true })
          .click();
      }
      expect(problems, 'no console errors or failed requests').toEqual([]);
    }
  );
}
