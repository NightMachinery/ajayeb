// The move review and the first-time tips (M10.HS5), through the real UI.
//
// - Against the bot (the advisor is on by default there): at the
//   military-supremacy scenario, where building Walls wins at once, the
//   player discards Walls instead. The summary of the bot's reply ends with
//   the review: what the player did, and that building Walls would have
//   won the game.
// - Hot-seat (the advisor is off by default): the same weak move gets no
//   review, in the view or on the screen.
// - First-time tips: at the science-pair scenario the Library's pair pip
//   is on the board; its tip shows once, "Got it" closes it, and after a
//   reload it does not show again.
//
// The base fixture turns the summary popup and the tips off for every
// spec; these tests turn back on what they need.

import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { GameDriver } from '../fixtures/driver';
import { watchHygiene } from '../fixtures/hygiene';
import { currentView, openHotseatRoom, playKey, SCENARIO_SEED } from '../fixtures/scenario';

const POPUP = '[data-dismiss-layer] [data-move-summary]';

/** Turns the opponent move summary popup back on for every page load. */
async function summaryOn(page: Page): Promise<void> {
  await page.addInitScript(() => {
    try {
      const raw = localStorage.getItem('ajayeb:settings');
      const current = raw ? (JSON.parse(raw) as Record<string, unknown>) : {};
      current['showOpponentMove'] = true;
      localStorage.setItem('ajayeb:settings', JSON.stringify(current));
    } catch {
      // Storage may be blocked; the default is on anyway.
    }
  });
}

/** Turns the first-time tips on, with none seen, once per tab (a reload keeps what the page saved). */
async function tipsOn(page: Page): Promise<void> {
  await page.addInitScript(() => {
    try {
      if (sessionStorage.getItem('ajayeb:e2e-tips-on')) return;
      sessionStorage.setItem('ajayeb:e2e-tips-on', '1');
      localStorage.setItem('ajayeb:tips', JSON.stringify({ off: false, seen: [] }));
    } catch {
      // Storage may be blocked; the default is on anyway.
    }
  });
}

test.describe('move review', () => {
  test("against the bot: the summary of the bot's reply reviews the player's weak move", async ({
    page
  }) => {
    test.setTimeout(90_000);
    const problems = watchHygiene(page);
    await summaryOn(page);
    const params = new URLSearchParams({
      mode: 'ai',
      level: 'easy',
      seat: '0',
      seed: String(SCENARIO_SEED),
      scenario: 'military-supremacy'
    });
    await page.goto(`/dev/new?${params.toString()}`);
    await page.waitForURL(/\/play\/[A-Z0-9]+$/);
    const driver = new GameDriver(page);
    await driver.waitReady();
    const room = await page.evaluate(() => window.__ajayeb?.room() ?? null);
    expect(room?.hints.advisor, 'a bot room starts with the advisor on').toBe(true);

    // Walls wins at once; the player discards it instead.
    const { before } = await playKey(driver, 'discard:walls');
    expect(before.choices.some((c) => c.key === 'construct:walls')).toBe(true);

    // The bot replies; the popup lists its move and ends with the review.
    await driver.nextTurn();
    const popup = page.locator(POPUP);
    await expect(popup).toBeVisible();
    await expect(popup).toHaveAttribute('data-move-summary', '1');
    const review = popup.locator('[data-move-review]');
    await expect(review).toHaveAttribute('data-move-review', 'better');
    await expect(review.locator('[data-review-played]')).toHaveText('You discarded Walls.');
    await expect(review.locator('[data-review-verdict]')).toHaveText(
      'Building Walls would have won the game.'
    );
    const view = await currentView(page);
    expect(view.review?.played).toBe('discard:walls');
    expect(view.review?.better).toBe('construct:walls');
    expect(view.review?.reason).toBe('wins');

    await page.keyboard.press('Escape');
    await expect(page.locator(POPUP)).toHaveCount(0);
    expect(problems).toEqual([]);
  });

  test('hot-seat: no review by default', async ({ page }) => {
    test.setTimeout(90_000);
    const problems = watchHygiene(page);
    await summaryOn(page);
    const driver = await openHotseatRoom(page, 'military-supremacy');
    const room = await page.evaluate(() => window.__ajayeb?.room() ?? null);
    expect(room?.hints.advisor, 'a hot-seat room starts with the advisor off').toBe(false);

    await playKey(driver, 'discard:walls');
    // The other player takes the device: the popup shows the discard.
    const second = await driver.nextTurn();
    const popup = page.locator(POPUP);
    await expect(popup).toBeVisible();
    await expect(page.locator('[data-move-review]')).toHaveCount(0);
    await page.keyboard.press('Escape');
    await expect(popup).toHaveCount(0);
    // They move; the first player's turn comes back with the summary of
    // that move, and no review in it or in the view.
    await driver.play(second);
    await driver.nextTurn();
    await expect(popup).toBeVisible();
    await expect(popup.locator('[data-summary-line]').first()).toBeVisible();
    await expect(page.locator('[data-move-review]')).toHaveCount(0);
    expect((await currentView(page)).review).toBeUndefined();
    await page.keyboard.press('Escape');
    expect(problems).toEqual([]);
  });
});

test.describe('first-time tips', () => {
  test('the pair tip shows once, and not again after a reload', async ({ page }) => {
    test.setTimeout(90_000);
    const problems = watchHygiene(page);
    await tipsOn(page);
    const driver = await openHotseatRoom(page, 'science-pair');
    await driver.nextTurn();

    const pip = page.locator(
      '.card-structure [data-card="library"] [data-hint="pair"][data-hint-side="own"]'
    );
    await expect(pip.first()).toBeVisible();
    const tip = page.locator('[data-dismiss-layer] [data-first-tip="pair"]');
    await expect(tip).toBeVisible();
    await expect(tip).toContainText('Each science symbol is on two cards.');
    await tip.locator('[data-tip-ok]').click();
    await expect(page.locator('[data-first-tip]')).toHaveCount(0);
    const seen = await page.evaluate(() => localStorage.getItem('ajayeb:tips'));
    expect(JSON.parse(seen ?? '{}')).toEqual({ off: false, seen: ['pair'] });

    // After a reload the pip is back, and its tip is not.
    await page.reload();
    await driver.waitReady();
    await driver.nextTurn();
    await expect(pip.first()).toBeVisible();
    // Give a due tip the time to open before checking that none did.
    await page.waitForTimeout(500);
    await expect(page.locator('[data-first-tip]')).toHaveCount(0);
    expect(problems).toEqual([]);
  });
});
