import { expect, type BrowserContext, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { newDeviceContext } from '../fixtures/online';
import { watchHygiene } from '../fixtures/hygiene';
import { createTableFromHome, lobbyAction, lobbyReady } from '../fixtures/lobby';

async function userId(page: Page): Promise<string> {
  return page.evaluate(() => window.__ajayebTable!.lobby()!.you.userId);
}
async function menu(page: Page, seat: number): Promise<void> {
  const item = page.locator(`[data-seat="${seat}"] details`);
  if (!(await item.getAttribute('open'))) {
    // A boolean HTML attribute can be the empty string; inspect its property.
    if (!(await item.evaluate((el) => (el as HTMLDetailsElement).open)))
      await page.locator(`[data-act="lobby-seat-menu:${seat}"]`).click();
  }
}
async function named(context: BrowserContext, name: string): Promise<Page> {
  await context.addInitScript((n) => localStorage.setItem('ajayeb:name', n), name);
  return context.newPage();
}

test.describe('Home and shared Haftgane lobby', { tag: ['@full', '@mobile'] }, () => {
  test('friends, bots, spectators, roles and settings reach the table session', async ({
    browser
  }, info) => {
    const contexts: BrowserContext[] = [];
    try {
      for (let i = 0; i < 3; i++) contexts.push(await newDeviceContext(browser, info));
      const host = await named(contexts[0]!, 'Host');
      const guest = await named(contexts[1]!, 'Guest');
      const watcher = await named(contexts[2]!, 'Watcher');
      const hygiene = [host, guest, watcher].map(watchHygiene);
      const code = await createTableFromHome(host);
      expect(await host.locator('[data-act="lobby-seats"]').inputValue()).toBe('4');
      await menu(host, 1);
      await host.locator('[data-act="lobby-bot-level:1"]').selectOption('normal');
      await lobbyAction(host, 'lobby-add-bot:1');
      await expect(host.locator('[data-seat="1"]')).toHaveAttribute('data-kind', 'bot');
      await menu(host, 1);
      await lobbyAction(host, 'lobby-remove-bot:1');
      await expect(host.locator('[data-seat="1"]')).toHaveAttribute('data-kind', 'open');
      await guest.goto(`/play/${code}`);
      await expect(guest.locator('[data-lobby]')).toBeVisible();
      await lobbyReady(guest);
      const id = await userId(guest);
      await expect(host.locator('[data-seat="1"]')).toHaveAttribute('data-kind', 'human');
      for (const seat of [2, 3]) {
        await menu(host, seat);
        await lobbyAction(host, `lobby-add-bot:${seat}`);
      }
      await watcher.goto(`/play/${code}`);
      await lobbyReady(watcher);
      expect(await watcher.evaluate(() => window.__ajayebTable!.lobby()!.you.seat)).toBeUndefined();
      await watcher.locator('[data-act="lobby-spectators"]').click();
      await expect(watcher.locator('[data-spectators]')).toContainText('Watcher');
      await expect(watcher.locator('[data-act="lobby-start"]')).toHaveCount(0);
      await menu(host, 1);
      await lobbyAction(host, `lobby-promote:${id}`);
      await expect(guest.locator(`[data-role-tags="${id}"] [data-admin]`)).toBeVisible();
      await lobbyAction(host, `lobby-demote:${id}`);
      await expect(guest.locator(`[data-role-tags="${id}"] [data-admin]`)).toHaveCount(0);
      await lobbyAction(host, `lobby-promote:${id}`);
      await guest.locator('[data-act="lobby-roles"]').click();
      await lobbyAction(guest, 'lobby-step-down');
      await expect(guest.locator(`[data-role-tags="${id}"] [data-admin]`)).toHaveCount(0);
      await lobbyAction(host, `lobby-promote:${id}`);
      await host.locator('[data-act="lobby-seating"]').selectOption('lobby');
      await lobbyReady(host);
      await host.locator('[data-act="lobby-reveal"]').selectOption('false');
      await lobbyReady(host);
      await expect(guest.locator('[data-act="lobby-seating"]')).toHaveValue('lobby');
      await expect(guest.locator('[data-act="lobby-reveal"]')).toHaveValue('false');
      await expect(host.locator('[data-act="lobby-undo"]')).toBeDisabled();
      await host.locator('[data-act="lobby-roles"]').click();
      await lobbyAction(host, 'lobby-step-down');
      await expect(host.locator('[data-act="lobby-seating"]')).toBeDisabled();
      await lobbyAction(guest, 'lobby-start');
      await expect
        .poll(() => guest.evaluate(() => window.__ajayebTable?.latest()?.status))
        .toBe('playing');
      expect(await guest.evaluate(() => window.__ajayebTable!.latest()!.finalReveal)).toBe(false);
      await expect(watcher.locator('[data-watching]')).toBeVisible();
      for (const problems of hygiene) expect(problems).toEqual([]);
    } finally {
      for (const context of contexts) await context.close();
    }
  });

  test('seat limits and a cold invite preserve the name gate and confirm displacement', async ({
    page,
    browser
  }, info) => {
    const context = await newDeviceContext(browser, info);
    try {
      const guest = await context.newPage();
      const hygiene = [watchHygiene(page), watchHygiene(guest)];
      const code = await createTableFromHome(page);
      await page.locator('[data-act="lobby-seats"]').selectOption('7');
      await lobbyReady(page);
      await expect(page.locator('[data-seat]')).toHaveCount(7);
      await page.locator('[data-act="lobby-seats"]').selectOption('4');
      await lobbyReady(page);
      for (const seat of [1, 2]) {
        await menu(page, seat);
        await lobbyAction(page, `lobby-add-bot:${seat}`);
      }
      const guestMessages: string[] = [];
      guest.on('websocket', (ws) =>
        ws.on('framesent', (f) => guestMessages.push(String(f.payload)))
      );
      await guest.goto(`/play/${code}`);
      await expect(guest.locator('[data-name-input]')).toBeVisible();
      expect(guestMessages).toEqual([]);
      expect(
        await page.evaluate(
          () => window.__ajayebTable!.lobby()!.seats.filter((s) => s.kind === 'human').length
        )
      ).toBe(1);
      await guest.locator('[data-name-input]').fill('Invited');
      await guest.locator('[data-control="name-submit"]').click();
      await lobbyReady(guest);
      await expect(page.locator('[data-seat="3"]')).toHaveAttribute('data-kind', 'human');
      await page.locator('[data-act="lobby-seats"]').selectOption('3');
      await expect(page.locator('[data-act="lobby-confirm"]')).toBeVisible();
      await page.locator('[data-act="lobby-cancel"]').click();
      expect(await page.evaluate(() => window.__ajayebTable!.lobby()!.seatCount)).toBe(4);
      await expect(page.locator('[data-act="lobby-seats"]')).toHaveValue('4');
      await page.locator('[data-act="lobby-seats"]').selectOption('3');
      await lobbyAction(page, 'lobby-confirm');
      await expect(page.locator('[data-seat]')).toHaveCount(3);
      await expect
        .poll(() => guest.evaluate(() => window.__ajayebTable!.lobby()!.you.seat))
        .toBeUndefined();
      await page.locator('[data-act="lobby-qr"]').click();
      await expect(page.locator('[data-lobby] svg[role="img"] path')).toHaveAttribute('d', /M/);
      for (const problems of hygiene) expect(problems).toEqual([]);
    } finally {
      await context.close();
    }
  });

  test('a former controller creates a rematch lobby with reservations', async ({
    page,
    browser
  }, info) => {
    const extra = await newDeviceContext(browser, info);
    try {
      const guest = await named(extra, 'Returning');
      const hygiene = watchHygiene(page);
      const code = await createTableFromHome(page);
      await guest.goto(`/play/${code}`);
      await lobbyReady(guest);
      await page.locator('[data-act="lobby-seating"]').selectOption('lobby');
      await lobbyReady(page);
      await lobbyAction(page, 'lobby-start');
      await lobbyReady(guest);
      const previousSeat = await guest.evaluate(() => window.__ajayebTable!.latest()!.you.seat!);
      await lobbyAction(page, `seat-bot:${previousSeat}`);
      await expect(guest.locator('[data-watching]')).toBeVisible();
      for (let i = 0; i < 200; i++) {
        await lobbyReady(page);
        await lobbyReady(guest);
        const frame = await page.evaluate(() => window.__ajayebTable!.latest());
        if (frame?.status === 'finished') break;
        for (const tab of [page, guest]) {
          const key = await tab.evaluate(() => {
            const f = window.__ajayebTable!.latest();
            return f?.view.you?.choices[0]?.key ?? null;
          });
          if (key) {
            await lobbyAction(tab, key);
            if (await tab.locator('[data-act="confirm"]').isEnabled())
              await lobbyAction(tab, 'confirm');
          }
        }
      }
      await expect(page.locator('[data-finished]')).toBeVisible();
      await lobbyAction(guest, 'lobby-rematch');
      await guest.waitForURL(
        (url) => /\/play\//.test(url.pathname) && !url.pathname.endsWith(code)
      );
      await expect(guest.locator('[data-lobby]')).toBeVisible();
      await expect(guest.locator('[data-kind="reserved"]')).toHaveCount(1);
      await expect(guest.locator('[data-act="lobby-start"]')).toBeDisabled();
      const seat = await guest.locator('[data-kind="reserved"]').getAttribute('data-seat');
      await menu(guest, Number(seat));
      await lobbyAction(guest, `lobby-open:${seat}`);
      await expect(guest.locator('[data-act="lobby-start"]')).toBeEnabled();
      expect(hygiene).toEqual([]);
    } finally {
      await extra.close();
    }
  });
});
