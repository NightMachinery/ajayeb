// The game's history (docs/protocol.md, the state frame's `history`): a
// hot-seat game is played a dozen moves into Age I through the real UI, then
// the page reloads. The join frame of the new socket carries every applied
// action, so the log sheet shows the whole game again, line for line, and
// the Session holds one history entry per ply. (online.full.spec.ts checks
// the same across an online player's reload.)

import { expect } from '@playwright/test';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { currentView, logLines, openHotseatRoom, playKey } from '../fixtures/scenario';

test.describe('history', () => {
  test('a hot-seat reload mid-game shows the whole log', async ({ page }) => {
    const problems = watchHygiene(page);
    const driver = await openHotseatRoom(page, null, 4343);

    // The draft (eight picks), then a few moves of Age I.
    for (let moves = 0; moves < 12; moves++) {
      const view = await currentView(page);
      const choice = view.choices.find((c) => c.kind !== 'wonder') ?? view.choices[0];
      if (!choice) throw new Error(`no choice at ply ${view.ply}`);
      await playKey(driver, choice.key);
    }
    await driver.nextTurn();
    const view = await currentView(page);
    expect(view.age).toBe(1);

    const before = await logLines(driver);
    const historyBefore = await page.evaluate(() => window.__ajayeb?.history() ?? []);
    expect(historyBefore.map((e) => e.ply)).toEqual([...Array(view.ply).keys()]);
    // Every draft pick and every move of Age I has its line.
    expect(before.length).toBeGreaterThanOrEqual(12);
    expect(before.filter((l) => / drafted /.test(l))).toHaveLength(8);

    await page.reload();
    await driver.waitReady();
    await driver.nextTurn();
    const historyAfter = await page.evaluate(() => window.__ajayeb?.history() ?? []);
    expect(historyAfter).toEqual(historyBefore);
    expect(await logLines(driver)).toEqual(before);

    expect(problems).toEqual([]);
  });
});
