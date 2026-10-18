// Undo without the own-move toast (M10.U1), through the real UI on the
// desktop and mobile projects.
//
// - A player's own move shows no toast, in a hot-seat and in a bot game
//   (the one exception, a choice the server skipped, is checked by the
//   skip flows in flows.spec.ts).
// - Undo is in the game menu ("Undo last move"), greyed out with a reason
//   while there is nothing to take back, and U undoes too.
// - A game created with "Allow undo" off shows no undo anywhere: not in the
//   top bar, the menu, the pass-device screen or the keyboard; its settings
//   say undo is off; the sheet remembers the choice; and the server itself
//   refuses an undo in such a game (probed over the protocol).

import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { GameDriver, type Snapshot } from '../fixtures/driver';
import { watchHygiene } from '../fixtures/hygiene';
import { probeUndoOff } from '../fixtures/ws-game';

/** Opens the pass-and-play sheet from Home and returns it. */
async function openHotseatSheet(page: Page) {
  await page.goto('/');
  await page.getByRole('button', { name: /^Pass and play/ }).click();
  const sheet = page.locator('dialog[open]');
  await expect(sheet.getByRole('heading', { name: 'Pass and play' })).toBeVisible();
  return sheet;
}

/** The room on screen, as the test hook reports it. */
async function room(page: Page) {
  const r = await page.evaluate(() => window.__ajayeb?.room() ?? null);
  if (!r) throw new Error('no room on screen');
  return r;
}

/** No toast is on screen. */
async function expectNoToast(page: Page, what: string): Promise<void> {
  await expect(page.locator('[data-toast]'), `no toast after ${what}`).toHaveCount(0);
}

/** Plays this client's moves until one has been made in Age I, checking for toasts after each. */
async function playIntoAgeOne(driver: GameDriver): Promise<Snapshot> {
  for (let guard = 0; guard < 20; guard++) {
    const snap = await driver.nextTurn();
    const move = await driver.play(snap);
    await expectNoToast(driver.page, `the move ${move.choice.key}`);
    if (snap.phase === 'play') return driver.nextTurn();
  }
  throw new Error('no move in Age I within 20 moves');
}

/** Opens the game menu and returns its undo item (absent in a game without undo). */
async function openMenu(driver: GameDriver) {
  await driver.dismissToasts();
  await driver.visible('[data-control="menu"]').click();
  await expect(driver.page.locator('[data-game-menu]:popover-open')).toHaveCount(1);
  return driver.page.locator('[data-game-menu] [data-control="menu-undo"]');
}

async function closeMenu(page: Page): Promise<void> {
  await page.keyboard.press('Escape');
  await expect(page.locator('[data-game-menu]:popover-open')).toHaveCount(0);
}

test.describe('undo', { tag: ['@mobile'] }, () => {
  test('a hot-seat game: no toast for my own move, Undo from the menu and with U', async ({
    page
  }) => {
    const problems = watchHygiene(page);
    const sheet = await openHotseatSheet(page);
    // No pass-device screen, so the board and its menu are never covered.
    await sheet.getByRole('checkbox').setChecked(false);
    const allow = sheet.locator('[data-control="allow-undo"]');
    await expect(allow).toBeChecked();
    await expect(sheet.locator('[data-allow-undo]')).toContainText('cannot be changed');
    await sheet.getByRole('button', { name: 'Start game' }).click();
    await page.waitForURL(/\/play\/[A-Z0-9]+$/);
    const driver = new GameDriver(page);
    await driver.waitReady();
    expect((await room(page)).allowUndo).toBe(true);

    // Before any move there is nothing to take back: greyed out, with why.
    let item = await openMenu(driver);
    await expect(item).toBeDisabled();
    await expect(item.locator('[data-undo-reason]')).toHaveText(
      'No move of yours to take back yet.'
    );
    await closeMenu(page);

    const before = await playIntoAgeOne(driver);
    expect(before.canUndo).toBe(true);

    // Undo last move, from the menu.
    item = await openMenu(driver);
    await expect(item).toBeEnabled();
    await expect(item.locator('[data-undo-reason]')).toHaveCount(0);
    await item.click();
    await driver.waitAdvance(before.version);
    const undone = await driver.snapshot();
    expect(undone.ply).toBe(before.ply - 1);
    await expectNoToast(page, 'the undo');

    // One more move, then U takes it back.
    const again = await driver.nextTurn();
    await driver.play(again);
    await expectNoToast(page, 'a move in Age I');
    const moved = await driver.snapshot();
    await page.locator('body').press('u');
    await driver.waitAdvance(moved.version);
    expect((await driver.snapshot()).ply).toBe(again.ply);

    // The game's settings show undo, read-only.
    await driver.visible('[data-control="menu"]').click();
    await driver.visible('[data-control="menu-hints"]').click();
    const hints = page.locator('dialog[open]').filter({ has: page.locator('[data-game-hints]') });
    await expect(hints.locator('[data-undo-setting="on"]')).toContainText(
      'Undo is allowed in this game.'
    );
    // U and Ctrl+Z under the open sheet do not take back the move beneath it.
    const underSheet = await driver.snapshot();
    expect(underSheet.canUndo).toBe(true);
    await page.keyboard.press('u');
    await page.keyboard.press('Control+z');
    await page.waitForTimeout(500);
    expect((await driver.snapshot()).version).toBe(underSheet.version);
    await expect(hints).toBeVisible();
    await page.keyboard.press('Escape');
    expect(problems).toEqual([]);
  });

  test('a bot game: no toast for my own move or the reply, and Undo from the menu', async ({
    page
  }) => {
    test.setTimeout(120_000);
    const problems = watchHygiene(page);
    await page.goto('/');
    await page.getByRole('button', { name: /Play vs AI/ }).click();
    const sheet = page.locator('dialog[open]');
    await expect(sheet.getByRole('heading', { name: 'Choose your rival' })).toBeVisible();
    await expect(sheet.locator('[data-control="allow-undo"]')).toBeChecked();
    await sheet.getByText('Apprentice', { exact: true }).click();
    await sheet.getByText('Sun', { exact: true }).click();
    await sheet.getByRole('button', { name: 'Start game' }).click();
    await page.waitForURL(/\/play\/[A-Z0-9]+$/);
    const driver = new GameDriver(page);
    await driver.waitReady();
    expect((await room(page)).allowUndo).toBe(true);

    // nextTurn waits through the bot's reply, so each check also covers it.
    const before = await playIntoAgeOne(driver);
    await expectNoToast(page, "the bot's reply");
    expect(before.canUndo).toBe(true);

    const item = await openMenu(driver);
    await expect(item).toBeEnabled();
    await item.click();
    await driver.waitAdvance(before.version);
    const after = await driver.nextTurn();
    // The undo takes back my last move together with the bot's reply.
    expect(after.ply).toBeLessThan(before.ply);
    expect(after.toMove).toBe(0);
    const confirmation = page.locator('[data-toast][role="status"]');
    await expect(confirmation).toContainText('Undid your last move.');
    await expect(confirmation.locator('..')).toHaveAttribute('aria-live', 'polite');
    expect(problems).toEqual([]);
  });

  test('a game created with undo off shows no undo anywhere, and the server refuses it', async ({
    page,
    baseURL
  }) => {
    const problems = watchHygiene(page);
    const sheet = await openHotseatSheet(page);
    // The pass-device screen on, to check it offers no undo either.
    await sheet.getByRole('checkbox').setChecked(true);
    await sheet.locator('[data-control="allow-undo"]').setChecked(false);
    await sheet.getByRole('button', { name: 'Start game' }).click();
    await page.waitForURL(/\/play\/[A-Z0-9]+$/);
    const driver = new GameDriver(page);
    await driver.waitReady();
    expect((await room(page)).allowUndo).toBe(false);

    // A move: the pass-device screen comes up without "Undo X's move".
    const first = await driver.nextTurn();
    await driver.play(first);
    await expect(page.locator('[data-pass-device]')).toBeVisible();
    await expect(page.locator('[data-pass-device] button')).toHaveCount(1);
    await expect(page.locator('[data-pass-device]')).not.toContainText('Undo');
    await driver.passDevice();
    await expectNoToast(page, 'the first move');

    // No arrow in the top bar, no menu item, and U does nothing.
    const snap = await driver.nextTurn();
    expect(snap.canUndo).toBe(false);
    await expect(page.locator('[data-control="undo"]')).toHaveCount(0);
    const item = await openMenu(driver);
    await expect(item).toHaveCount(0);
    await closeMenu(page);
    await page.locator('body').press('u');
    await page.waitForTimeout(300);
    expect((await driver.snapshot()).version).toBe(snap.version);

    // The game's settings say undo is off.
    await driver.visible('[data-control="menu"]').click();
    await driver.visible('[data-control="menu-hints"]').click();
    const hints = page.locator('dialog[open]').filter({ has: page.locator('[data-game-hints]') });
    await expect(hints.locator('[data-undo-setting="off"]')).toContainText(
      'Undo is off in this game.'
    );
    await page.keyboard.press('Escape');

    // The sheet remembers the choice for the next game.
    const next = await openHotseatSheet(page);
    await expect(next.locator('[data-control="allow-undo"]')).not.toBeChecked();

    // The server refuses an undo in such a game, whatever the client does.
    if (!baseURL) throw new Error('no baseURL');
    const probe = await probeUndoOff(baseURL);
    expect(probe).toEqual({ allowUndo: [false, false], canUndo: false, undoAnswer: 'undo-off' });
    expect(problems).toEqual([]);
  });
});
