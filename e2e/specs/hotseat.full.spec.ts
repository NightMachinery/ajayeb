// M5 acceptance: a whole hot-seat game played through the real UI, from
// Home to the result screen and back, on the desktop and mobile projects.
//
// The game is created the way a player creates it (Home, the pass-and-play
// sheet, Start), and every move is a click on the visible controls, chosen
// by the shared driver's hot-seat policy (fixtures/driver.ts). The mobile
// project keeps the pass-device screen on, as a touch device does by
// default; the desktop project turns it off and plays with the fixed
// orientation, so both ways of sharing a device are walked.

import { expect } from '@playwright/test';
import { test } from '../fixtures/base';
import { GameDriver } from '../fixtures/driver';
import { watchHygiene } from '../fixtures/hygiene';

const NAMES = ['Darius', 'Roxana'] as const;

test.describe('hot-seat', { tag: ['@mobile', '@full'] }, () => {
  test('a full game from Home to the result and back', async ({ page, isMobile }, testInfo) => {
    test.setTimeout(420_000);
    const problems = watchHygiene(page);
    const started = Date.now();

    await page.goto('/');
    await expect(page.getByRole('heading', { level: 1, name: 'Ajayeb' })).toBeVisible();

    // The pass-and-play sheet.
    await page.getByRole('button', { name: /Pass and play/ }).click();
    const sheet = page.locator('dialog[open]');
    await expect(sheet.getByRole('heading', { name: 'Pass and play' })).toBeVisible();
    await sheet.getByLabel('Player 1 (Sun crest)').fill(NAMES[0]);
    await sheet.getByLabel('Player 2 (Moon crest)').fill(NAMES[1]);
    await sheet.getByRole('checkbox').setChecked(isMobile);
    await sheet.getByText(isMobile ? 'Face active player' : 'Fixed', { exact: true }).click();
    await sheet.getByRole('button', { name: 'Start game' }).click();

    await page.waitForURL(/\/play\/[A-Z0-9]+$/);
    const driver = new GameDriver(page);
    await driver.waitReady();
    const room = await page.evaluate(() => window.__ajayeb?.room() ?? null);
    expect(room?.mode).toBe('hotseat');
    expect(room?.seats.map((s) => s.name)).toEqual([...NAMES]);
    expect(await page.evaluate(() => window.__ajayeb?.you()?.seats)).toEqual([0, 1]);

    const end = await driver.playToEnd();
    const { stats } = driver;
    const seconds = Math.round((Date.now() - started) / 1000);
    const summary = {
      project: testInfo.project.name,
      seconds,
      actions: stats.actions,
      flows: stats.flows,
      pendingKinds: stats.pendingKinds,
      passScreens: stats.passScreens,
      victory: end.result?.victory,
      winner: end.result?.winner ?? null,
      totals: end.result?.scores.map((s) => s.total)
    };
    console.log(`hot-seat game: ${JSON.stringify(summary)}`);
    await testInfo.attach('game-summary.json', {
      body: JSON.stringify(summary, null, 2),
      contentType: 'application/json'
    });

    // Every flow the policy promises was walked.
    expect(stats.flows.draft).toBe(6);
    expect(stats.flows.construct).toBeGreaterThan(0);
    expect(stats.flows.discard).toBeGreaterThan(0);
    expect(stats.flows.wonderCardFirst).toBeGreaterThan(0);
    expect(stats.flows.wonderFirst).toBeGreaterThan(0);
    expect(stats.passScreens > 0).toBe(isMobile);

    // The result screen: the server's victory and winner, and its score sheet.
    const result = end.result;
    expect(result, 'a finished game carries its result').toBeTruthy();
    if (!result) return;
    const over = page.locator('[data-game-over]');
    await expect(over).toHaveAttribute('data-victory', result.shared ? 'shared' : result.victory);
    if (result.winner !== undefined) {
      await expect(over).toHaveAttribute('data-winner', String(result.winner));
    }
    const dialog = page.locator('dialog[open]').filter({ has: page.locator('[data-score-table]') });
    await expect(dialog).toBeVisible();
    const winnerName = result.winner === 0 ? NAMES[0] : result.winner === 1 ? NAMES[1] : /win/i;
    await expect(dialog.locator('h2')).toContainText(winnerName);
    const table = dialog.locator('[data-score-table]');
    await expect(table.locator('thead')).toContainText(NAMES[0]);
    await expect(table.locator('thead')).toContainText(NAMES[1]);
    const categories = [
      'blue',
      'green',
      'yellow',
      'purple',
      'wonders',
      'tokens',
      'coins',
      'military',
      'total'
    ] as const;
    for (const category of categories) {
      const cells = table.locator(`[data-score-row="${category}"] td`);
      await expect(cells).toHaveText([
        String(result.scores[0][category]),
        String(result.scores[1][category])
      ]);
    }

    // Home.
    await dialog.locator('[data-home]').click();
    await page.waitForURL((url) => url.pathname === '/');
    await expect(page.getByRole('heading', { level: 1, name: 'Ajayeb' })).toBeVisible();

    expect(problems, 'console errors, page errors or failed requests').toEqual([]);
  });
});
