import { expect, type Page, type BrowserContext } from '@playwright/test';
import { test } from '../fixtures/base';
import { GameDriver } from '../fixtures/driver';
import {
  createOnlineGame,
  joinByLink,
  newDeviceContext,
  startOnlineGame,
  readRoomFile
} from '../fixtures/online';
import { lobbyReady, lobbyAction } from '../fixtures/lobby';
import { watchHygiene } from '../fixtures/hygiene';

async function menu(page: Page, seat: number) {
  const details = page.locator(`[data-seat="${seat}"] details`);
  if (!(await details.evaluate((el) => (el as HTMLDetailsElement).open)))
    await page.locator(`[data-act="lobby-seat-menu:${seat}"]`).click();
}
async function hints(page: Page) {
  const driver = new GameDriver(page);
  await driver.dismissToasts();
  await driver.visible('[data-control="menu"]').click();
  await driver.visible('[data-control="menu-hints"]').click();
  return page.locator('dialog[open]').filter({ has: page.locator('[data-game-hints]') });
}

test.describe('shared online lobby', { tag: ['@full', '@mobile'] }, () => {
  test('explicit Start carries both tabs and a spectator with hints-only frozen admins', async ({
    browser
  }, info) => {
    const contexts: BrowserContext[] = [];
    try {
      for (let i = 0; i < 3; i++) contexts.push(await newDeviceContext(browser, info));
      const owner = await contexts[0]!.newPage(),
        guest = await contexts[1]!.newPage(),
        watch = await contexts[2]!.newPage();
      const problems = [owner, guest, watch].map(watchHygiene);
      const hellos: number[] = [],
        joins: number[] = [];
      guest.on('websocket', (socket) => {
        let version = 0;
        socket.on('framesent', ({ payload }) => {
          const m = JSON.parse(String(payload));
          if (m.t === 'hello') {
            version = m.v;
            hellos.push(version);
          }
          if (m.t === 'join') joins.push(version);
        });
      });
      const code = await createOnlineGame(owner, 'Host');
      await expect(owner.locator('[data-act="lobby-first-player"]')).toHaveValue('random');
      await joinByLink(guest, owner.url(), 'Guest');
      await joinByLink(watch, owner.url(), 'Watcher');
      const twin = await contexts[1]!.newPage();
      problems.push(watchHygiene(twin));
      await twin.goto(owner.url());
      await lobbyReady(twin);
      expect(await guest.evaluate(() => window.__ajayebTable!.lobby()!.you.seat)).toBe(1);
      expect(await twin.evaluate(() => window.__ajayebTable!.lobby()!.you.seat)).toBe(1);
      expect(await watch.evaluate(() => window.__ajayebTable!.lobby()!.you.seat)).toBeUndefined();
      // Fully occupied lobbies wait for an explicit Start, never a friend's join.
      await expect(owner.locator('[data-duel-settings]')).toBeVisible();
      expect((readRoomFile(code) as unknown as { game: string }).game).toBe('lobby');
      const guestId = await guest.evaluate(() => window.__ajayebTable!.lobby()!.you.userId);
      const watchId = await watch.evaluate(() => window.__ajayebTable!.lobby()!.you.userId);
      await menu(owner, 1);
      await lobbyAction(owner, `lobby-promote:${guestId}`);
      await owner.locator('[data-act="lobby-spectators"]').click();
      await lobbyAction(owner, `lobby-promote:${watchId}`);
      await owner.locator('[data-act="lobby-roles"]').click();
      await lobbyAction(owner, 'lobby-step-down');
      await expect(owner.locator('[data-act="lobby-start"]')).toHaveCount(0);
      await startOnlineGame(guest);
      for (const page of [owner, guest, twin, watch]) {
        await new GameDriver(page).waitReady();
        expect(new URL(page.url()).pathname).toBe(`/play/${code}`);
        await expect(page.locator('[data-lobby]')).toHaveCount(0);
        await expect(page.locator('[data-role-tags]')).toHaveCount(0);
      }
      expect(await guest.evaluate(() => window.__ajayeb!.you()!.seats)).toEqual([1]);
      expect(await twin.evaluate(() => window.__ajayeb!.you()!.seats)).toEqual([1]);
      expect(await watch.evaluate(() => window.__ajayeb!.you()!.seats)).toEqual([]);
      expect(await owner.evaluate(() => window.__ajayeb!.you()!.admin)).toBe(false);
      expect(await guest.evaluate(() => window.__ajayeb!.you()!.admin)).toBe(true);
      expect(await watch.evaluate(() => window.__ajayeb!.you()!.admin)).toBe(true);
      expect(hellos).toContain(2);
      // An unknown invite already opened the app's v1 socket before discovery.
      // Duel reuses that socket at Start rather than saying hello again.
      expect(hellos).toContain(1);
      expect(joins.at(-1)).toBe(1);
      // The promoted spectator can edit hints, without receiving any choices.
      const sheet = await hints(watch);
      await expect(sheet.locator('[data-hint-editor]')).toHaveAttribute('data-hint-editor', 'edit');
      await sheet.locator('[data-hint-family="threats"]').getByRole('switch').click();
      for (const page of [owner, guest, twin, watch])
        await expect
          .poll(() => page.evaluate(() => window.__ajayeb?.room()?.hints.threats))
          .toBe(false);
      await watch.keyboard.press('Escape');
      await expect(
        new GameDriver(guest).visible('[data-control="device-link"][data-seat="0"]')
      ).toHaveCount(0);
      await guest.reload();
      await new GameDriver(guest).waitReady();
      expect(await guest.evaluate(() => window.__ajayeb!.you()!.seats)).toEqual([1]);
      expect(await guest.evaluate(() => window.__ajayeb!.you()!.admin)).toBe(true);
      for (const errors of problems) expect(errors).toEqual([]);
    } finally {
      for (const context of contexts) await context.close();
    }
  });

  test('the seat count switches panels and Duel bot levels round trip before Start', async ({
    page
  }) => {
    const problems = watchHygiene(page);
    const code = await createOnlineGame(page, 'Host');
    await page.locator('[data-act="lobby-first-player"]').selectOption('1');
    await lobbyReady(page);
    await page.locator('[data-act="lobby-seats"]').selectOption('3');
    await lobbyReady(page);
    await expect(page.locator('[data-haftgane-settings]')).toBeVisible();
    await expect(page.locator('[data-act="lobby-sides"]')).toHaveValue('day');
    const hints = await page.evaluate(() => window.__ajayebTable!.lobby()!.options.hints);
    for (const sides of ['choose', 'day', 'random']) {
      await page.locator('[data-act="lobby-sides"]').selectOption(sides);
      await lobbyReady(page);
      await expect(page.locator('[data-act="lobby-sides"]')).toHaveValue(sides);
      const options = await page.evaluate(() => window.__ajayebTable!.lobby()!.options);
      expect(options.sides).toBe(sides);
      expect(options.firstPlayer).toBe(1);
      expect(options.hints).toEqual(hints);
    }
    await expect(page.locator('[data-duel-settings]')).toHaveCount(0);
    await page.locator('[data-act="lobby-seats"]').selectOption('2');
    await lobbyReady(page);
    await expect(page.locator('[data-duel-settings]')).toBeVisible();
    await expect(page.locator('[data-haftgane-settings]')).toHaveCount(0);
    await expect(page.locator('[data-act="lobby-sides"]')).toHaveCount(0);
    expect(await page.evaluate(() => window.__ajayebTable!.lobby()!.options.sides)).toBe('random');
    await expect(page.locator('[data-act="lobby-first-player"]')).toHaveValue('1');
    await menu(page, 1);
    await page.locator('[data-act="lobby-bot-level:1"]').selectOption('hard');
    await lobbyAction(page, 'lobby-add-bot:1');
    await menu(page, 1);
    await page.locator('[data-act="lobby-bot-level:1"]').selectOption('normal');
    await lobbyReady(page);
    expect(await page.evaluate(() => window.__ajayebTable!.lobby()!.seats[1]!.bot)).toBe('normal');
    await startOnlineGame(page);
    expect(new URL(page.url()).pathname).toBe(`/play/${code}`);
    expect(await page.evaluate(() => window.__ajayeb!.room()!.allowUndo)).toBe(false);
    const file = readRoomFile(code) as unknown as {
      schema: number;
      mode: string;
      seats: { kind: string; bot?: string }[];
      options: { firstPlayer: string };
    };
    expect(file.schema).toBe(1);
    expect(file.mode).toBe('online');
    expect(file.seats[1]).toMatchObject({ kind: 'bot', bot: 'normal' });
    expect(file.options.firstPlayer).toBe('seat1');
    await expect
      .poll(() => page.evaluate(() => window.__ajayeb?.history()?.length ?? 0))
      .toBeGreaterThan(0);
    expect(problems).toEqual([]);
  });
});
