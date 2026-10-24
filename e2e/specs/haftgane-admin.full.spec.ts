// Haftgane's admin controls, resignation and spectator view over the real
// server, from Home's lobby: the creator starts a five-seat table alone
// with bots (so undo is on), two friends join the running table as
// spectators, and then, checked as it goes:
//
// - a spectator's hand region is the spectator panel, and following a
//   seat centres the ring on it (its neighbours are that seat's);
// - the creator seats a spectator in a bot's seat through the seat's
//   admin menu, and the creator's undo button disappears with a notice
//   that says why;
// - the creator gives that seat back to a bot: the former player reads
//   that a bot plays their seat, and the creator's seat menu names the
//   seat they held, while a non-admin's menu offers no seat and no hint;
// - the creator seats them again, promotes and demotes them (the role
//   tag follows), promotes them again; they step down, then resign to a
//   bot from the game menu;
// - the creator promotes and demotes a spectator from the game menu;
// - every table frame each page received passes the pending-ownership
//   oracle, gives no seat a private field and no page another seat's
//   private view, and carries the previous-seat hint only when its viewer
//   is an admin;
// - no console error or failed request happens.
//
// Nothing depends on a particular deal: the creator's seat is random, and
// every seat used is read from the frames.
import { expect, type BrowserContext, type Page } from '@playwright/test';
import type { TableState } from '../../web/src/lib/wire/platform.gen';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { newDeviceContext } from '../fixtures/online';
import {
  choose,
  latest,
  menuAction,
  openBotTable,
  openMenu,
  pendingOwnership,
  ready,
  seatAction,
  tableStates,
  traffic,
  type Traffic
} from '../fixtures/table';

/** The fields a public seat may carry. */
const PUBLIC_SEAT = new Set([
  'seat',
  'board',
  'side',
  'stages',
  'built',
  'coins',
  'victories',
  'defeats',
  'shields',
  'handSize',
  'hasKept',
  'relative',
  'relation',
  'last',
  'score',
  'science'
]);

/**
 * The per-frame leak oracle: no seat carries a private field, the private
 * view is the viewer's own seat's, and the previous-seat hint reaches only
 * a frame whose viewer the same frame lists as an admin.
 */
function frameLeaks(name: string, log: readonly Traffic[]): string[] {
  const out: string[] = [];
  for (const i of tableStates(log)) {
    const f = log[i]!.msg as unknown as TableState;
    for (const s of f.view.seats) {
      const extra = Object.keys(s).filter((k) => !PUBLIC_SEAT.has(k));
      if (extra.length) out.push(`${name} ${i}: seat ${s.seat} carries ${extra.join(', ')}`);
    }
    if (f.view.you && f.view.you.seat !== f.you.seat)
      out.push(`${name} ${i}: private view of seat ${f.view.you.seat}`);
    const me = f.humans.find((h) => h.userId === f.you.userId);
    if (f.you.previousSeats !== undefined && !(me?.admin || me?.temporary))
      out.push(`${name} ${i}: previous seats sent to a non-admin`);
  }
  return out;
}

/** The frames a page received that carried previous-seat hints. */
function hinted(log: readonly Traffic[]): number {
  return tableStates(log).filter(
    (i) => (log[i]!.msg as unknown as TableState).you.previousSeats !== undefined
  ).length;
}

async function named(context: BrowserContext, name: string): Promise<Page> {
  await context.addInitScript((n) => localStorage.setItem('ajayeb:name', n), name);
  return context.newPage();
}

async function frame(page: Page): Promise<TableState> {
  return latest(page);
}

/** The first played point resolves: every seat confirms, the bots on their own. */
async function playTurn(page: Page): Promise<void> {
  const before = (await frame(page)).batch;
  const key = (await frame(page)).view.you!.choices[0]!.key;
  await choose(page, key);
  await choose(page, 'confirm');
  await expect.poll(async () => (await frame(page)).batch).toBeGreaterThan(before);
}

test.describe('Haftgane admin controls', { tag: ['@full', '@haftgane-admin'] }, () => {
  test('seat switches, roles, resignation, undo and the spectator view', async ({
    browser
  }, info) => {
    test.setTimeout(240_000);
    const contexts: BrowserContext[] = [];
    try {
      for (let i = 0; i < 3; i++) contexts.push(await newDeviceContext(browser, info));
      const a = await named(contexts[0]!, 'Ana');
      const b = await named(contexts[1]!, 'Bo');
      const c = await named(contexts[2]!, 'Cy');
      const pages = [a, b, c] as const;
      const hygiene = pages.map(watchHygiene);
      const logs = pages.map(traffic);

      // The creator alone with four bots: undo is on once a turn is played.
      const code = await openBotTable(a, 5, 'Ana');
      const start = await frame(a);
      expect(start.startingHumanCount).toBe(1);
      expect(start.allowUndo).toBe(true);
      const seatA = start.you.seat!;
      await expect(a.locator('[data-act="undo"]')).toHaveCount(0);
      await playTurn(a);
      await expect(a.locator('[data-act="undo"]')).toBeVisible();
      await expect(a.locator('[data-role-tag="creator"]').first()).toBeVisible();

      // Two friends open the running table: they watch.
      for (const p of [b, c]) {
        await p.goto(`/play/${code}`);
        await expect(p.locator('[data-spectator-panel]')).toBeVisible();
        expect((await frame(p)).you.seat).toBeUndefined();
      }
      const userB = (await frame(b)).you.userId;
      const userC = (await frame(c)).you.userId;
      await expect(b.locator('[data-last-plays]')).toBeVisible();
      await expect(b.locator(`[data-spectator="${userB}"]`)).toBeVisible();
      await expect(b.locator(`[data-spectator="${userC}"]`)).toBeVisible();

      // A spectator follows a seat: the ring centres on it.
      const n = start.seats.length;
      const followed = (seatA + 2) % n;
      await c.locator(`[data-act="follow:${followed}"]`).click();
      await expect(c.locator(`[data-act="follow:${followed}"]`)).toHaveAttribute(
        'aria-pressed',
        'true'
      );
      await expect(c.locator('[data-neighbour="left"]')).toHaveAttribute(
        'data-seat',
        String((followed + 1) % n)
      );
      await expect(c.locator('[data-neighbour="right"]')).toHaveAttribute(
        'data-seat',
        String((followed + n - 1) % n)
      );

      // A non-admin's game menu lists the spectators but offers no seat.
      await openMenu(c);
      await expect(c.locator('[data-table-menu] [data-spectator-list]')).toBeVisible();
      await expect(c.locator('[data-table-menu] [data-act^="seat-menu:"]')).toHaveCount(0);
      await expect(c.locator('[data-table-menu] [data-act="step-down"]')).toHaveCount(0);
      await c.locator('[data-act="menu-close"]').click();

      // The creator seats a spectator in a bot's seat: undo goes for good.
      const bots = start.seats.filter((s) => s.kind === 'bot').map((s) => s.seat);
      const [seatB, other] = bots as [number, number];
      await seatAction(a, seatB, `seat-user:${seatB}:${userB}`);
      await expect(b.locator('[data-notice="seated"]')).toBeVisible();
      await expect.poll(async () => (await frame(b)).you.seat).toBe(seatB);
      await expect(b.locator('[data-spectator-panel]')).toHaveCount(0);
      await expect(a.locator('[data-notice="undo-off"]')).toBeVisible();
      await expect(a.locator('[data-act="undo"]')).toHaveCount(0);
      expect((await frame(a)).undoOff).toBe(true);
      await a.locator('[data-act="menu-close"]').click();

      // The creator gives the seat back to a bot: the former player
      // watches and reads who plays their seat.
      await seatAction(a, seatB, `seat-bot:${seatB}:normal`);
      await expect(b.locator('[data-notice="watching"]')).toBeVisible();
      await expect(b.locator(`[data-former-seat="${seatB}"]`)).toBeVisible();
      expect((await frame(a)).seats[seatB]).toMatchObject({ kind: 'bot', bot: 'normal' });

      // The admin's seat menu names the seat the former player held; a
      // non-admin's frames never carry it.
      await a.locator(`[data-act="seat-menu:${other}"]`).click();
      const hint = a.locator(
        `[data-seat-menu="${other}"] [data-spectator="${userB}"] [data-previous-seat]`
      );
      await expect(hint).toHaveAttribute('data-previous-seat', String(seatB));
      await expect(hint).toHaveText(`earlier in seat ${seatB + 1}`);
      await a.locator('[data-act="seat-menu-close"]').click();
      await openMenu(c);
      await expect(c.locator('[data-previous-seat]')).toHaveCount(0);
      await c.locator('[data-act="menu-close"]').click();
      expect(hinted(logs[2]!), 'hints sent to a non-admin spectator').toBe(0);

      // Seated again, then promoted and demoted from the seat's menu.
      await seatAction(a, seatB, `seat-user:${seatB}:${userB}`);
      await expect.poll(async () => (await frame(b)).you.seat).toBe(seatB);
      await expect(b.locator('[data-former-seat]')).toHaveCount(0);
      await seatAction(a, seatB, `role-promote:${userB}`);
      await expect(
        a.locator(`[data-seat="${seatB}"] [data-role-tag="admin"]`).first()
      ).toBeVisible();
      await expect.poll(async () => (await frame(b)).you.previousSeats).toBeDefined();
      await seatAction(a, seatB, `role-demote:${userB}`);
      await expect(a.locator(`[data-seat="${seatB}"] [data-role-tag]`)).toHaveCount(0);
      expect((await frame(b)).humans.find((h) => h.userId === userB)?.admin).toBe(false);

      // Promoted again, the player steps down, then resigns to a bot.
      await seatAction(a, seatB, `role-promote:${userB}`);
      await expect
        .poll(async () => (await frame(b)).humans.find((h) => h.userId === userB)?.admin)
        .toBe(true);
      await choose(b, 'step-down');
      await expect
        .poll(async () => (await frame(b)).humans.find((h) => h.userId === userB)?.admin)
        .toBe(false);
      await choose(b, 'resign');
      await expect(b.locator('[data-notice="watching"]')).toBeVisible();
      await expect(b.locator(`[data-former-seat="${seatB}"]`)).toBeVisible();
      expect((await frame(b)).seats[seatB]).toMatchObject({ kind: 'bot', bot: 'easy' });
      expect((await frame(a)).undoOff).toBe(true);
      await expect(a.locator('[data-act="undo"]')).toHaveCount(0);

      // A spectator promoted and demoted from the game menu's list.
      await menuAction(a, `role-promote:${userC}`);
      await expect(
        a.locator(`[data-table-menu] [data-spectator="${userC}"] [data-role-tag="admin"]`)
      ).toBeVisible();
      await menuAction(a, `role-demote:${userC}`);
      await expect(
        a.locator(`[data-table-menu] [data-spectator="${userC}"] [data-role-tag]`)
      ).toHaveCount(0);
      await a.locator('[data-act="menu-close"]').click();

      // The spectator still follows the table as it goes on.
      await playTurn(a);
      await ready(c);
      expect((await frame(c)).batch).toBe((await frame(a)).batch);
      await expect(c.locator(`[data-act="follow:${followed}"]`)).toHaveAttribute(
        'aria-pressed',
        'true'
      );

      const names = ['creator', 'player', 'spectator'];
      for (const [i, log] of logs.entries()) {
        expect(pendingOwnership(names[i]!, log)).toEqual([]);
        expect(frameLeaks(names[i]!, log)).toEqual([]);
      }
      expect(hinted(logs[0]!), 'the creator saw the hint').toBeGreaterThan(0);
      for (const problems of hygiene) expect(problems).toEqual([]);
    } finally {
      for (const ctx of contexts) await ctx.close();
    }
  });
});
