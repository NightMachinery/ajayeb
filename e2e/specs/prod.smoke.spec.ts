// The M8 production smoke: a few short checks played through the real UI
// against a real deployment, never a local build or a dev endpoint. Tagged
// `@prod`, which playwright.config.ts forces as the only grep whenever
// E2E_BASE_URL is set (see its header comment), so these are the only specs
// that can ever run against a real site, and every other spec is kept off
// it the same way. `make e2e-prod` is the normal way to run this file
// (docs/testing.md, "Production smoke").
//
// Four tests:
// - GET /health answers ok with a version;
// - Home renders in both themes with no console errors, failed requests or
//   horizontal scroll;
// - two isolated browser contexts play a short online game: A creates it
//   through the UI, B joins through the invite link, and they play at
//   least five moves (the draft picks among them) through the confirm
//   step, each acting only on its own turn, agreeing on the room version
//   after every move, then both leave through the game menu;
// - a short vs-AI game at Apprentice, started from Home, two moves with the
//   bot answering.
//
// Nothing here calls /api/dev or a /dev route: a production server answers
// 404 for the first and turns the second off entirely (AGENTS.md,
// protocol.md). The online and vs-AI tests reuse GameDriver and the online
// helpers (e2e/fixtures/driver.ts, e2e/fixtures/online.ts) exactly as the
// local full-playthrough specs do, waiting on room versions and ready()
// rather than sleeping.
//
// Every room a run creates is left for the server's reaper (protocol.md
// does not expose a delete). A full run of this file creates at most two:
// one online room and one vs-AI room.

import { expect, type BrowserContext, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { GameDriver, type Snapshot } from '../fixtures/driver';
import { watchHygiene } from '../fixtures/hygiene';
import {
  createOnlineGame,
  joinByLink,
  newDeviceContext,
  onlinePolicy,
  readInvite,
  type Crest
} from '../fixtures/online';

async function hook<T>(page: Page, read: () => T): Promise<T> {
  return page.evaluate(read);
}

/** Home, the vs-AI sheet, a level and a crest, Start game; returns the room code. */
async function startAiGame(page: Page, level: string, crest: Crest): Promise<string> {
  await page.goto('/');
  await expect(page.getByRole('heading', { level: 1, name: 'Ajayeb' })).toBeVisible();
  await page.getByRole('button', { name: /Play vs AI/ }).click();
  const sheet = page.locator('dialog[open]');
  await expect(sheet.getByRole('heading', { name: 'Choose your rival' })).toBeVisible();
  await sheet.getByText(level, { exact: true }).click();
  await expect(sheet.getByRole('radio', { name: new RegExp(level) })).toBeChecked();
  await sheet.getByText(crest, { exact: true }).click();
  await expect(sheet.getByRole('radio', { name: crest, exact: true })).toBeChecked();
  await sheet.getByRole('button', { name: 'Start game' }).click();
  await page.waitForURL(/\/play\/[A-Z0-9]+$/);
  const code = new URL(page.url()).pathname.split('/').pop();
  if (!code) throw new Error(`no room code in ${page.url()}`);
  return code;
}

/** The game menu's Home item, closing the room and returning to '/'. */
async function leaveThroughMenu(driver: GameDriver): Promise<void> {
  await driver.dismissToasts();
  await driver.visible('[data-control="menu"]').click();
  await driver.visible('[data-game-menu] [data-control="menu-home"]').click();
  await driver.page.waitForURL((url) => url.pathname === '/');
}

test.describe('production smoke', { tag: ['@prod'] }, () => {
  test('GET /health answers ok with a version', async ({ request }) => {
    const res = await request.get('/health');
    expect(res.ok()).toBe(true);
    const body: unknown = await res.json();
    expect(body).toMatchObject({ ok: true });
    const version = (body as { version?: unknown }).version;
    expect(typeof version).toBe('string');
    expect((version as string).length).toBeGreaterThan(0);
    console.log(`prod smoke: /health version ${JSON.stringify(version)}`);
  });

  for (const colorScheme of ['light', 'dark'] as const) {
    test.describe(`home, ${colorScheme}`, () => {
      test.use({ colorScheme });

      test('renders with no console errors, failed requests or horizontal scroll', async ({
        page
      }) => {
        const problems = watchHygiene(page);
        await page.goto('/');
        await expect(page.getByRole('heading', { level: 1, name: 'Ajayeb' })).toBeVisible();
        await expect(page.locator('html')).toHaveAttribute('data-theme', colorScheme);
        await page.evaluate(async () => {
          await document.fonts.ready;
        });

        const { sw, iw } = await page.evaluate(() => ({
          sw: document.documentElement.scrollWidth,
          iw: window.innerWidth
        }));
        expect(sw, `horizontal scroll: scrollWidth ${sw} > innerWidth ${iw}`).toBeLessThanOrEqual(
          iw
        );

        expect(problems, 'console errors, page errors or failed requests').toEqual([]);
      });
    });
  }

  test('two browsers create, join and play an online game', async ({ browser }, testInfo) => {
    const contexts: BrowserContext[] = [];
    try {
      const ctxA = await newDeviceContext(browser, testInfo, { clipboard: true });
      contexts.push(ctxA);
      const pageA = await ctxA.newPage();
      const problemsA = watchHygiene(pageA);
      const code = await createOnlineGame(pageA, 'Roxana', 'Sun');
      const invite = await readInvite(pageA);
      expect(invite.code).toBe(code);

      const ctxB = await newDeviceContext(browser, testInfo);
      contexts.push(ctxB);
      const pageB = await ctxB.newPage();
      const problemsB = watchHygiene(pageB);
      await joinByLink(pageB, invite.link, 'Babak');

      await expect(pageA.locator('[data-lobby]')).toHaveCount(0);
      const roomA = await hook(pageA, () => window.__ajayeb?.room() ?? null);
      expect(roomA?.mode).toBe('online');
      expect(roomA?.status).toBe('playing');

      const driverA = new GameDriver(pageA, { confirm: true });
      const driverB = new GameDriver(pageB, { confirm: true });
      const seatA = (await hook(pageA, () => window.__ajayeb?.you()?.seats ?? []))[0];
      if (seatA !== 0 && seatA !== 1) throw new Error('A holds no seat');
      const bySeat = seatA === 0 ? [driverA, driverB] : [driverB, driverA];
      const others = (mover: GameDriver) => [driverA, driverB].filter((d) => d !== mover);

      const MIN_MOVES = 5;
      let moves = 0;
      for (let guard = 0; moves < MIN_MOVES; guard++) {
        if (guard > 60) throw new Error(`fewer than ${MIN_MOVES} moves after ${guard} tries`);
        await driverA.waitReady();
        const now = await driverA.snapshot();
        if (now.phase === 'gameOver') throw new Error(`the game ended after only ${moves} moves`);
        if (now.toMove !== 0 && now.toMove !== 1) throw new Error(`no seat to move at ${now.ply}`);
        const mover = bySeat[now.toMove]!;

        for (const other of others(mover)) {
          expect(await hook(other.page, () => window.__ajayeb?.you()?.canAct ?? false)).toBe(false);
          expect(await hook(other.page, () => window.__ajayeb?.choices().length ?? 0)).toBe(0);
        }

        const snap = await mover.nextTurn();
        await mover.play(snap, onlinePolicy);
        moves += 1;
        for (const other of others(mover)) await other.waitAdvance(snap.version);

        const versionA = await hook(pageA, () => window.__ajayeb?.room()?.version);
        const versionB = await hook(pageB, () => window.__ajayeb?.room()?.version);
        expect(versionB, `both pages on the same room version after move ${moves}`).toBe(versionA);
      }

      await leaveThroughMenu(driverA);
      await leaveThroughMenu(driverB);

      console.log(
        `prod smoke: online game ${code}, ${moves} moves, ` +
          `flows A ${JSON.stringify(driverA.stats.flows)} B ${JSON.stringify(driverB.stats.flows)}`
      );
      expect(problemsA, 'console errors, page errors or failed requests (A)').toEqual([]);
      expect(problemsB, 'console errors, page errors or failed requests (B)').toEqual([]);
    } finally {
      for (const ctx of contexts) await ctx.close();
    }
  });

  test('a short vs-AI game at Apprentice', async ({ page }) => {
    const problems = watchHygiene(page);
    const driver = new GameDriver(page);
    const code = await startAiGame(page, 'Apprentice', 'Sun');

    await driver.waitReady();
    const room = await hook(page, () => window.__ajayeb?.room() ?? null);
    expect(room?.code).toBe(code);
    expect(room?.mode).toBe('ai');

    const MOVES = 2;
    for (let i = 0; i < MOVES; i++) {
      const snap = await driver.nextTurn();
      if (snap.phase === 'gameOver') break;
      await driver.play(snap);
    }
    expect(
      driver.stats.actions,
      'the player made moves and the bot answered'
    ).toBeGreaterThanOrEqual(1);
    // One more wait shows the bot's reply landed and the page is ready
    // again with either a move of ours or the game over.
    const after: Snapshot = await driver.nextTurn();
    expect(after.ply).toBeGreaterThan(0);

    await leaveThroughMenu(driver);
    console.log(`prod smoke: vs-AI game ${code}, ${driver.stats.actions} player moves`);
    expect(problems, 'console errors, page errors or failed requests').toEqual([]);
  });
});
