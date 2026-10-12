// A game's hint settings (docs/dev/decisions.md D23, M10.HS0), through the
// real UI on the desktop and mobile projects.
//
// Hot-seat: a family switched off in the pass-and-play sheet reaches the
// room (the state frame's room.hints), the device's local default, and the
// game menu's Hints sheet. Online: the creator is the game's admin; the
// lobby shows a summary of the settings, the joiner sees them read-only,
// and a change the admin makes mid-game reaches both players and shows in
// both game logs, also after a reload.
//
// The joiner never sees the lobby (the room starts the moment the second
// seat is taken), so it reads the settings from the game menu instead.

import { expect, type BrowserContext, type Locator, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { GameDriver } from '../fixtures/driver';
import { watchHygiene } from '../fixtures/hygiene';
import {
  createOnlineGame,
  joinByLink,
  newDeviceContext,
  onlinePolicy,
  readInvite
} from '../fixtures/online';
import { logLines } from '../fixtures/scenario';

/** The game menu's Hints sheet, opened from the visible top bar. */
async function openGameHints(driver: GameDriver): Promise<Locator> {
  const { page } = driver;
  await driver.dismissToasts();
  await driver.visible('[data-control="menu"]').click();
  await driver.visible('[data-control="menu-hints"]').click();
  const sheet = page.locator('dialog[open]').filter({ has: page.locator('[data-game-hints]') });
  await expect(sheet).toBeVisible();
  return sheet;
}

/** Closes a sheet with Escape. The sheet is a modal <dialog>: Escape closes
 *  it wherever the focus is inside it, including on a hint switch that is
 *  waiting for the server (aria-disabled, so it keeps the focus). Should a
 *  press still be lost (the dialog not yet the top layer's active one), it
 *  is pressed again, at most a few times; a second press after the sheet
 *  has closed is never sent. */
async function closeSheet(page: Page, sheet: Locator): Promise<void> {
  await expect(sheet).toBeVisible();
  await expect(async () => {
    if ((await sheet.count()) > 0) await page.keyboard.press('Escape');
    await expect(sheet).toHaveCount(0, { timeout: 1_500 });
  }).toPass({ timeout: 8_000 });
}

/** The summary line of the game menu's Hints item, read with the menu open. */
async function menuSummary(driver: GameDriver): Promise<string> {
  await driver.dismissToasts();
  await driver.visible('[data-control="menu"]').click();
  const text = (await driver
    .visible('[data-control="menu-hints"] [data-hints-summary-text]')
    .textContent())!.trim();
  await driver.page.keyboard.press('Escape');
  await expect(driver.page.locator('[data-game-menu]:popover-open')).toHaveCount(0);
  return text;
}

test.describe('hint settings', { tag: ['@mobile'] }, () => {
  test('a hot-seat game starts with a family switched off', async ({ page }) => {
    const problems = watchHygiene(page);
    await page.goto('/');
    await page.getByRole('button', { name: /Pass and play/ }).click();
    const sheet = page.locator('dialog[open]');
    await expect(sheet.getByRole('heading', { name: 'Pass and play' })).toBeVisible();
    // No pass-device screen, so the menu is never covered.
    await sheet.getByRole('checkbox').setChecked(false);

    // The Hints section starts closed, with every family but the advisor on.
    const section = sheet.locator('[data-hints-section]');
    await expect(section.locator('[data-control="hints-section"]')).toContainText(
      'All hints on except Advisor'
    );
    await section.locator('[data-control="hints-section"]').click();
    const prices = section.locator('[data-hint-family="prices"]');
    await expect(prices).toHaveAttribute('data-on', '');
    await prices.getByRole('switch').click();
    await expect(prices).not.toHaveAttribute('data-on');
    // A switch leaves the section open.
    await expect(section).toHaveAttribute('open', '');
    await expect(prices.getByRole('switch')).toBeVisible();
    await expect(section.locator('[data-control="hints-section"]')).toContainText(
      'All hints on except Prices and Advisor'
    );
    await sheet.getByRole('button', { name: 'Start game' }).click();

    await page.waitForURL(/\/play\/[A-Z0-9]+$/);
    const driver = new GameDriver(page);
    await driver.waitReady();
    const room = await page.evaluate(() => window.__ajayeb?.room() ?? null);
    expect(room?.mode).toBe('hotseat');
    expect(room?.hints).toMatchObject({ prices: false, economy: true, advisor: false });
    // The device that created the room holds both seats and is its admin.
    expect(await page.evaluate(() => window.__ajayeb?.you()?.admin)).toBe(true);
    // The next game on this device starts from these settings.
    const saved = await page.evaluate(() => localStorage.getItem('ajayeb:hints'));
    expect(JSON.parse(saved ?? 'null')).toMatchObject({ hints: { prices: false } });

    expect(await menuSummary(driver)).toBe('All hints on except Prices and Advisor');
    const hints = await openGameHints(driver);
    await expect(hints.locator('[data-hint-editor]')).toHaveAttribute('data-hint-editor', 'edit');
    await expect(hints.locator('[data-hint-family="prices"]')).not.toHaveAttribute('data-on');
    await expect(hints.locator('[data-hint-family="economy"]')).toHaveAttribute('data-on', '');
    await expect(hints.locator('[data-hint-local-hide]')).toContainText('On my screen');
    await closeSheet(page, hints);

    expect(problems).toEqual([]);
  });

  test('the admin changes the hints mid-game, and both players see it', async ({
    browser
  }, testInfo) => {
    test.setTimeout(120_000);
    const contexts: BrowserContext[] = [];
    try {
      const ctxA = await newDeviceContext(browser, testInfo, { clipboard: true });
      contexts.push(ctxA);
      const pageA = await ctxA.newPage();
      const problemsA = watchHygiene(pageA);
      await createOnlineGame(pageA, 'Roxana', 'Sun');

      // The lobby says which hints the game has.
      const summary = pageA.locator('[data-lobby] [data-hints-summary]');
      await expect(summary).toBeVisible();
      await expect(summary.locator('[data-hints-summary-text]')).toHaveText(
        'All hints on except Advisor'
      );
      const invite = await readInvite(pageA);

      const ctxB = await newDeviceContext(browser, testInfo);
      contexts.push(ctxB);
      const pageB = await ctxB.newPage();
      const problemsB = watchHygiene(pageB);
      await joinByLink(pageB, invite.link, 'Babak');
      await expect(pageA.locator('[data-lobby]')).toHaveCount(0);

      const driverA = new GameDriver(pageA, { confirm: true });
      const driverB = new GameDriver(pageB, { confirm: true });
      await driverA.waitReady();
      expect(await pageA.evaluate(() => window.__ajayeb?.you()?.admin)).toBe(true);
      expect(await pageB.evaluate(() => window.__ajayeb?.you()?.admin)).toBe(false);

      // One move, so the change lands mid-game.
      const toMove = await pageA.evaluate(() => window.__ajayeb?.view()?.toMove ?? null);
      const [mover, other] = toMove === 0 ? [driverA, driverB] : [driverB, driverA];
      const snap = await mover.nextTurn();
      await mover.play(snap, onlinePolicy);
      await other.waitAdvance(snap.version);

      // The joiner sees the settings read-only.
      expect(await menuSummary(driverB)).toBe('All hints on except Advisor');
      const hintsB = await openGameHints(driverB);
      await expect(hintsB.locator('[data-hint-editor]')).toHaveAttribute(
        'data-hint-editor',
        'readonly'
      );
      await expect(hintsB.getByRole('switch')).toHaveCount(1); // the local board-marks switch only
      await expect(hintsB).toContainText('Only the player who created the game can change them.');
      await closeSheet(pageB, hintsB);

      // The admin switches Threats off.
      const hintsA = await openGameHints(driverA);
      const threats = hintsA.locator('[data-hint-family="threats"]');
      await expect(threats).toHaveAttribute('data-on', '');
      const threatsSwitch = threats.getByRole('switch');
      await threatsSwitch.click();
      await expect(threats).not.toHaveAttribute('data-on');
      // While the change is on its way the switch waits, but keeps the
      // focus, so the Escape right after the click still closes the sheet.
      await expect(threatsSwitch).toBeFocused();
      await closeSheet(pageA, hintsA);

      for (const page of [pageA, pageB]) {
        await page.waitForFunction(() => window.__ajayeb?.room()?.hints.threats === false);
      }
      expect(await menuSummary(driverB)).toBe('All hints on except Threats and Advisor');

      const line = 'Roxana changed the hints: turned off Threats.';
      expect(await logLines(driverA)).toContain(line);
      expect(await logLines(driverB)).toContain(line);

      // The room log is kept with the room.
      await pageB.reload();
      await driverB.waitReady();
      expect(await logLines(driverB)).toContain(line);

      expect(problemsA).toEqual([]);
      expect(problemsB).toEqual([]);
    } finally {
      for (const c of contexts) await c.close();
    }
  });
});
