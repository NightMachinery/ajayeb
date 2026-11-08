// Production-only messages and UI: no dev endpoint or saved-room-file reads.
import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { createTableFromHome, lobbyReady } from '../fixtures/lobby';
import { joinByLink, newDeviceContext } from '../fixtures/online';
import { choose, expectPending, latest, ready } from '../fixtures/table';
import { watchHygiene } from '../fixtures/hygiene';

async function scholars(page: Page, first: number, count: number): Promise<void> {
  for (let seat = first; seat < count; seat++) {
    await page.locator(`[data-act="lobby-seat-menu:${seat}"]`).click();
    await page.locator(`[data-act="lobby-bot-level:${seat}"]`).selectOption('normal');
    await page.locator(`[data-act="lobby-add-bot:${seat}"]`).click();
    await lobbyReady(page);
  }
}

async function turn(page: Page): Promise<void> {
  const frame = await latest(page);
  const choice = frame.view.you?.choices[0];
  expect(choice, 'server offered a human pick').toBeDefined();
  await choose(page, choice!.key);
  await choose(page, 'confirm');
}

test.describe('Haftgane production', { tag: ['@prod', '@mobile'] }, () => {
  test.beforeEach(async ({ request }) => {
    const response = await request.get('/health');
    expect(response.ok()).toBe(true);
    const health = (await response.json()) as { haftganeCreation?: boolean };
    test.skip(health.haftganeCreation !== true, 'Haftgane creation is disabled on this server');
  });

  test('a creator plays several turns with six Scholar bots', async ({ page }) => {
    const problems = watchHygiene(page);
    await createTableFromHome(page);
    await page.locator('[data-act="lobby-seats"]').selectOption('7');
    await lobbyReady(page);
    await scholars(page, 1, 7);
    await choose(page, 'lobby-start');
    await expect(page.locator('[data-table-screen]')).toBeVisible();
    for (let i = 0; i < 3; i++) {
      const before = await latest(page);
      expect(before.seats.filter((s) => s.kind === 'bot')).toHaveLength(6);
      for (const seat of before.seats.filter((s) => s.kind === 'bot'))
        expect(seat.bot).toBe('normal');
      await turn(page);
      await expect.poll(async () => (await latest(page)).batch).toBeGreaterThan(before.batch);
    }
    expect(problems).toEqual([]);
  });

  test('two browsers play with bots and change a confirmed pick', async ({ browser }, info) => {
    const aContext = await newDeviceContext(browser, info);
    const bContext = await newDeviceContext(browser, info);
    try {
      const a = await aContext.newPage(),
        b = await bContext.newPage();
      const problemsA = watchHygiene(a),
        problemsB = watchHygiene(b);
      await createTableFromHome(a);
      await joinByLink(b, a.url(), 'Guest');
      await scholars(a, 2, 4);
      await choose(a, 'lobby-start');
      for (const page of [a, b]) {
        await expect(page.locator('[data-table-screen]')).toBeVisible();
        await ready(page);
      }
      const initial = await latest(a);
      const choices = initial.view.you!.choices;
      expect(choices.length).toBeGreaterThan(1);
      await choose(a, choices[0]!.key);
      await choose(a, 'confirm');
      await expectPending(a, choices[0]!.key, true);
      await choose(a, 'withdraw');
      await choose(a, choices[1]!.key);
      await choose(a, 'confirm');
      await expectPending(a, choices[1]!.key, true);
      expect((await latest(b)).pending).toBeUndefined();
      await turn(b);
      await expect.poll(async () => (await latest(a)).batch).toBeGreaterThan(initial.batch);
      for (let i = 0; i < 2; i++) {
        const before = await latest(a);
        await turn(a);
        await turn(b);
        for (const page of [a, b])
          await expect.poll(async () => (await latest(page)).batch).toBeGreaterThan(before.batch);
      }
      expect(problemsA).toEqual([]);
      expect(problemsB).toEqual([]);
    } finally {
      await aContext.close();
      await bContext.close();
    }
  });
});
