// Who is who in an online room (M10.ID1, docs/protocol.md "Device links"),
// through the real UI.
//
// Device links: a player copies their own seat's link with the compact
// button beside their name, a fresh device (its own identity token, no
// name) opens it, plays that seat, and keeps it across a reload because the
// claim stays in the address. The game's admin copies the other seat's
// link, and resets it: the device on the old link is let go, the old link
// fails on another device, and the new one works.
//
// Equal names: two players who both call themselves Sara show as "Sara"
// and "Sara 2", and the numbers stay when the first one leaves.

import { expect, type BrowserContext, type Page } from '@playwright/test';
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

const CLAIM_FRAGMENT = /^#claim=[A-Za-z0-9_-]{22}$/;

/** The visible seat panel (strip or city panel) of one seat. */
function seatPanel(driver: GameDriver, seat: 0 | 1) {
  return driver.visible(`.seat-panel[data-seat="${seat}"]`);
}

/** The open device link sheet of one seat. */
function linkSheet(page: Page, seat: 0 | 1) {
  return page
    .locator('dialog[open]')
    .filter({ has: page.locator(`[data-device-link-sheet="${seat}"]`) });
}

/**
 * Opens a seat's device link with its button beside the name, checks the
 * link was copied (the sheet says so, and the clipboard holds what the
 * field shows), closes the sheet and returns the link.
 */
async function copyDeviceLink(driver: GameDriver, seat: 0 | 1): Promise<string> {
  const { page } = driver;
  await driver.dismissToasts();
  await driver.visible(`[data-control="device-link"][data-seat="${seat}"]`).click();
  const sheet = linkSheet(page, seat);
  await expect(sheet).toBeVisible();
  await expect(sheet.locator('[data-device-link-status]')).toHaveAttribute(
    'data-device-link-status',
    'copied'
  );
  await expect(sheet.locator('[role="status"]')).toHaveText('Link copied.');
  const shown = await sheet.locator('input[data-device-link]').inputValue();
  const copied = await page.evaluate(() => navigator.clipboard.readText());
  expect(copied, 'the clipboard holds the link the sheet shows').toBe(shown);
  await page.keyboard.press('Escape');
  await expect(sheet).toHaveCount(0);
  return copied;
}

/** Plays moves with `others` until `seat` is to move, then one move with `driver`. */
async function playOneAs(driver: GameDriver, seat: 0 | 1, other: GameDriver): Promise<void> {
  for (let i = 0; i < 6; i++) {
    const toMove = await driver.page.evaluate(() => window.__ajayeb?.view()?.toMove ?? null);
    if (toMove === seat) break;
    const snap = await other.nextTurn();
    await other.play(snap, onlinePolicy);
    await driver.waitAdvance(snap.version);
  }
  const snap = await driver.nextTurn();
  expect(snap.toMove).toBe(seat);
  await driver.play(snap, onlinePolicy);
  await other.waitAdvance(snap.version);
}

async function youSeats(page: Page): Promise<number[]> {
  return page.evaluate(() => window.__ajayeb?.you()?.seats ?? []);
}

/**
 * Plays the wonder draft to its end with the two players' drivers: the
 * strips and city panels, where names and the self mark are, show only
 * after it.
 */
async function finishDraft(a: GameDriver, b: GameDriver): Promise<void> {
  const seatA = (await youSeats(a.page))[0];
  for (let i = 0; i < 12; i++) {
    const view = await a.page.evaluate(() => {
      const v = window.__ajayeb?.view();
      return v ? { phase: v.phase, toMove: v.toMove } : null;
    });
    if (!view || view.phase !== 'wonderDraft') return;
    const [mover, other] = view.toMove === seatA ? [a, b] : [b, a];
    const snap = await mover.nextTurn();
    await mover.play(snap, onlinePolicy);
    await other.waitAdvance(snap.version);
  }
  throw new Error('the wonder draft did not end');
}

test.describe('who is who', () => {
  test(
    'a device link moves a seat, the admin copies and resets it',
    { tag: ['@mobile'] },
    async ({ browser }, testInfo) => {
      test.setTimeout(150_000);
      const contexts: BrowserContext[] = [];
      try {
        const ctxA = await newDeviceContext(browser, testInfo, { clipboard: true });
        contexts.push(ctxA);
        const pageA = await ctxA.newPage();
        const problemsA = watchHygiene(pageA);
        await createOnlineGame(pageA, 'Roxana', 'Sun');
        // The lobby already offers the admin's own link.
        await expect(
          pageA.locator('[data-lobby] [data-control="device-link"][data-seat="0"]')
        ).toBeVisible();
        const invite = await readInvite(pageA);

        const ctxB = await newDeviceContext(browser, testInfo, { clipboard: true });
        contexts.push(ctxB);
        const pageB = await ctxB.newPage();
        const problemsB = watchHygiene(pageB);
        await joinByLink(pageB, invite.link, 'Babak');
        const driverA = new GameDriver(pageA, { confirm: true });
        const driverB = new GameDriver(pageB, { confirm: true });
        await driverA.waitReady();
        await driverB.waitReady();
        await finishDraft(driverA, driverB);

        // Each player is marked as "you" on their own seat only.
        await expect(seatPanel(driverB, 1).locator('[data-self-mark]')).toBeVisible();
        await expect(seatPanel(driverB, 0).locator('[data-self-mark]')).toHaveCount(0);
        await expect(seatPanel(driverA, 0).locator('[data-self-mark]')).toBeVisible();
        await expect(seatPanel(driverA, 1).locator('[data-self-mark]')).toHaveCount(0);

        // Babak may copy only his own seat's link; it carries a claim, not his token.
        await expect(pageB.locator('[data-control="device-link"][data-seat="0"]')).toHaveCount(0);
        const linkB = await copyDeviceLink(driverB, 1);
        const urlB = new URL(linkB);
        expect(urlB.pathname).toBe(`/play/${invite.code}`);
        expect(urlB.hash).toMatch(CLAIM_FRAGMENT);
        const tokenB = await pageB.evaluate(() => localStorage.getItem('ajayeb:token'));
        expect(tokenB).toBeTruthy();
        expect(linkB).not.toContain(tokenB!);

        // A fresh device opens it: no name prompt, and it plays seat 1.
        const ctxC = await newDeviceContext(browser, testInfo);
        contexts.push(ctxC);
        const pageC = await ctxC.newPage();
        const problemsC = watchHygiene(pageC);
        await pageC.goto(linkB);
        const driverC = new GameDriver(pageC, { confirm: true });
        await driverC.waitReady();
        await expect(pageC.locator('[data-name-prompt]')).toHaveCount(0);
        expect(await youSeats(pageC)).toEqual([1]);
        await expect(seatPanel(driverC, 1).locator('[data-self-mark]')).toBeVisible();
        expect(await pageC.evaluate(() => localStorage.getItem('ajayeb:token'))).not.toBe(tokenB);

        // It moves as Babak, and keeps the seat across a reload.
        await playOneAs(driverC, 1, driverA);
        await pageC.reload();
        await driverC.waitReady();
        expect(new URL(pageC.url()).hash).toBe(urlB.hash);
        expect(await youSeats(pageC)).toEqual([1]);
        await playOneAs(driverC, 1, driverA);
        await driverB.waitReady();

        // The admin copies Babak's link: the same one.
        const adminLink = await copyDeviceLink(driverA, 1);
        expect(adminLink).toBe(linkB);

        // The admin resets it.
        await driverA.visible('[data-control="device-link"][data-seat="1"]').click();
        const sheet = linkSheet(pageA, 1);
        await expect(sheet.locator('[data-device-link-status]')).toHaveAttribute(
          'data-device-link-status',
          'copied'
        );
        const reset = sheet.locator('[data-control="device-link-reset"]');
        await reset.click();
        await expect(reset).toHaveAttribute('data-armed', '');
        await pageA.waitForTimeout(400);
        await reset.click();
        await expect(sheet.locator('[data-device-link-status]')).toHaveAttribute(
          'data-device-link-status',
          'reset'
        );
        const freshLink = await sheet.locator('input[data-device-link]').inputValue();
        expect(freshLink).not.toBe(linkB);
        expect(new URL(freshLink).hash).toMatch(CLAIM_FRAGMENT);
        expect(await pageA.evaluate(() => navigator.clipboard.readText())).toBe(freshLink);
        await pageA.keyboard.press('Escape');
        await expect(sheet).toHaveCount(0);

        // The device on the old link is told and let go.
        await expect(pageC.getByText('This device link no longer works.')).toBeVisible();

        // Both players' logs record the reset.
        const line = 'Roxana reset the device link of Babak.';
        expect((await logLines(driverA)).some((l) => l.includes(line))).toBe(true);
        expect((await logLines(driverB)).some((l) => l.includes(line))).toBe(true);

        // The old link fails on another device; the new one works.
        const ctxD = await newDeviceContext(browser, testInfo);
        contexts.push(ctxD);
        const pageD = await ctxD.newPage();
        const problemsD = watchHygiene(pageD);
        await pageD.goto(linkB);
        await expect(pageD.getByText('This device link no longer works.')).toBeVisible();
        // Opened over the old one (only the fragment changes), then reloaded.
        await pageD.goto(freshLink);
        const driverD = new GameDriver(pageD, { confirm: true });
        await driverD.waitReady();
        expect(await youSeats(pageD)).toEqual([1]);
        await pageD.reload();
        await driverD.waitReady();
        expect(new URL(pageD.url()).hash).toBe(new URL(freshLink).hash);
        expect(await youSeats(pageD)).toEqual([1]);

        // Babak's own device never depended on a link.
        await pageB.reload();
        await driverB.waitReady();
        expect(await youSeats(pageB)).toEqual([1]);

        expect(problemsA).toEqual([]);
        expect(problemsB).toEqual([]);
        expect(problemsC).toEqual([]);
        expect(problemsD).toEqual([]);
      } finally {
        for (const c of contexts) await c.close();
      }
    }
  );

  test('equal names are numbered, and the numbers stay', async ({ browser }, testInfo) => {
    test.setTimeout(90_000);
    const contexts: BrowserContext[] = [];
    try {
      const ctxA = await newDeviceContext(browser, testInfo, { clipboard: true });
      contexts.push(ctxA);
      const pageA = await ctxA.newPage();
      await createOnlineGame(pageA, 'Sara', 'Sun');
      const invite = await readInvite(pageA);

      const ctxB = await newDeviceContext(browser, testInfo);
      contexts.push(ctxB);
      const pageB = await ctxB.newPage();
      const problemsB = watchHygiene(pageB);
      await joinByLink(pageB, invite.link, 'sara');
      const driverA = new GameDriver(pageA, { confirm: true });
      const driverB = new GameDriver(pageB, { confirm: true });
      await driverA.waitReady();
      await driverB.waitReady();
      await finishDraft(driverA, driverB);

      // The name line only: the panels also carry hint lines that name the
      // other seat ("toward sara 2's capital").
      const nameOf = (driver: GameDriver, seat: 0 | 1) =>
        seatPanel(driver, seat).locator('[dir="auto"]').first().textContent();
      const names = async (driver: GameDriver): Promise<[string, string]> => [
        (await nameOf(driver, 0)) ?? '',
        (await nameOf(driver, 1)) ?? ''
      ];
      for (const driver of [driverA, driverB]) {
        const [first, second] = await names(driver);
        expect(first).toContain('Sara');
        expect(first).not.toMatch(/sara 2/i);
        expect(second).toContain('sara 2');
      }

      // Sara leaves; the second Sara keeps her number, also after a reload.
      await ctxA.close();
      contexts.shift();
      await expect(seatPanel(driverB, 0)).toHaveAttribute('data-presence', 'away');
      await pageB.reload();
      await driverB.waitReady();
      const [first, second] = await names(driverB);
      expect(first).not.toMatch(/sara 2/i);
      expect(second).toContain('sara 2');
      expect(
        await pageB.evaluate(() => window.__ajayeb?.room()?.seats.map((s) => s.nameNo))
      ).toEqual([0, 2]);
      expect(problemsB).toEqual([]);
    } finally {
      for (const c of contexts) await c.close();
    }
  });
});
