// The move review and the first-time tips (M10.HS5), through the real UI.
//
// - Against the bot (the advisor is on by default there): at the
//   military-supremacy scenario, where building Walls wins at once, the
//   player discards Walls instead. The summary of the bot's reply ends with
//   the review: what the player did, and that building Walls would have
//   won the game.
// - Hot-seat (the advisor is off by default): the same weak move gets no
//   review, in the view or on the screen.
// - The advisor's strength: the game's admin picks Deep in the game menu's
//   Hints sheet (two-stage is the default); the room keeps it, this device
//   remembers it, and the review of the next weak move is a deep one.
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

/** Opens a bot game at the military-supremacy scenario, where building Walls wins at once. */
async function openBotGame(page: Page): Promise<GameDriver> {
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
  return driver;
}

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

  test(
    'against the bot: a play-again wonder turn names the tucked card and both moves',
    { tag: ['@mobile'] },
    async ({ page }) => {
      test.setTimeout(90_000);
      const problems = watchHygiene(page);
      await summaryOn(page);
      const params = new URLSearchParams({
        mode: 'ai',
        level: 'easy',
        seat: '0',
        seed: String(SCENARIO_SEED),
        scenario: 'theology-chain'
      });
      await page.goto(`/dev/new?${params.toString()}`);
      await page.waitForURL(/\/play\/[A-Z0-9]+$/);
      const driver = new GameDriver(page);
      await driver.waitReady();

      // Theology grants an extra turn when Pyramids is built with Rostrum;
      // the extra turn builds Statue, handing the turn to the bot.
      await playKey(driver, 'wonder:pyramids:rostrum');
      await playKey(driver, 'construct:statue');

      await driver.nextTurn();
      const popup = page.locator(POPUP);
      await expect(popup).toBeVisible();
      const review = popup.locator('[data-move-review]');
      await expect(review).toBeVisible();
      await expect(review.locator('[data-review-played]')).toHaveText([
        'You built the wonder Pyramids using Rostrum.',
        'You built Statue.'
      ]);
      const view = await currentView(page);
      expect(view.review?.played).toBe('wonder:pyramids:rostrum construct:statue');
      if (!view.review?.better) {
        await expect(review).toHaveAttribute('data-move-review', 'none');
        await expect(review.locator('[data-review-verdict]')).toHaveCount(0);
      }

      await page.keyboard.press('Escape');
      await expect(page.locator(POPUP)).toHaveCount(0);
      expect(problems).toEqual([]);
    }
  );
});

for (const locale of ['en', 'fa'] as const) {
  test.describe(`every decision review ${locale}`, () => {
    test.use({ appLocale: locale });

    test(
      'a missed winning progress token has its own advice',
      { tag: ['@mobile'] },
      async ({ page }) => {
        test.setTimeout(90_000);
        const problems = watchHygiene(page);
        await summaryOn(page);
        const params = new URLSearchParams({
          mode: 'ai',
          level: 'easy',
          seat: '0',
          seed: String(SCENARIO_SEED),
          scenario: 'law-supremacy'
        });
        await page.goto(`/dev/new?${params.toString()}`);
        await page.waitForURL(/\/play\/[A-Z0-9]+$/);
        const driver = new GameDriver(page);
        await driver.waitReady();
        await playKey(driver, 'construct:academy');
        expect((await currentView(page)).review).toBeUndefined();
        await playKey(driver, 'token:agriculture');
        await driver.nextTurn();
        const review = page.locator('[data-move-review]');
        await expect(review).toBeVisible();
        await expect(review.locator('[data-review-played]')).toHaveCount(2);
        await expect
          .poll(async () =>
            ((await review.locator('[data-review-verdict]').textContent()) ?? '').replace(
              /[\u200e\u200f\u2066-\u2069]/gu,
              ''
            )
          )
          .toBe(
            locale === 'en'
              ? 'Taking Law would have won the game.'
              : 'با گرفتن قانون بازی را می‌بردید.'
          );
        await expect
          .poll(async () => (await currentView(page)).review?.entries.map((entry) => entry.played))
          .toEqual(['construct:academy', 'token:agriculture']);
        const entries = (await currentView(page)).review?.entries ?? [];
        expect(entries[0]?.better).toBeUndefined();
        expect(entries[1]).toMatchObject({ better: 'token:law', reason: 'wins' });
        const width = await review.evaluate((el) => ({
          content: el.scrollWidth,
          box: el.clientWidth
        }));
        expect(width.content).toBeLessThanOrEqual(width.box + 1);
        await page.keyboard.press('Escape');
        expect(problems).toEqual([]);
      }
    );

    test(
      'taking the winning token lists both decisions in the final move, without a review',
      { tag: ['@mobile'] },
      async ({ page }) => {
        await summaryOn(page);
        const params = new URLSearchParams({
          mode: 'ai',
          level: 'easy',
          seat: '0',
          seed: String(SCENARIO_SEED),
          scenario: 'law-supremacy'
        });
        await page.goto(`/dev/new?${params.toString()}`);
        await page.waitForURL(/\/play\/[A-Z0-9]+$/);
        const driver = new GameDriver(page);
        await driver.waitReady();
        await playKey(driver, 'construct:academy');
        await playKey(driver, 'token:law');
        // The result's final move has both decisions as rows. With no better
        // move, a review would only repeat them, so none is shown
        // (M11.recheck-polish).
        const section = page.locator('dialog[open] [data-last-move]');
        await expect(section).toHaveAttribute('data-last-move', '0');
        await expect(
          section.locator('[data-summary-line="build"] [data-card="academy"]')
        ).toBeVisible();
        const token = section.locator('[data-summary-line="token"]');
        await expect(token).toHaveCount(1);
        await expect(token).toContainText(locale === 'en' ? 'Law' : 'قانون');
        await expect(section.locator('[data-end-review]')).toHaveCount(0);
        await expect(page.locator('[data-move-review]')).toHaveCount(0);
        expect((await currentView(page)).review?.entries.map((entry) => entry.played)).toEqual([
          'construct:academy',
          'token:law'
        ]);
      }
    );
  });
}

test.describe('advisor strength', () => {
  test(
    'the admin picks Deep in the Hints sheet, and the next review is a deep one',
    { tag: ['@mobile'] },
    async ({ page }) => {
      test.setTimeout(90_000);
      const problems = watchHygiene(page);
      await summaryOn(page);
      const driver = await openBotGame(page);
      const room = await page.evaluate(() => window.__ajayeb?.room() ?? null);
      expect(room?.hints.advisorStrength, 'two-stage is the default').toBe('twoStage');

      await driver.dismissToasts();
      await driver.visible('[data-control="menu"]').click();
      await driver.visible('[data-control="menu-hints"]').click();
      const sheet = page.locator('dialog[open]').filter({ has: page.locator('[data-game-hints]') });
      await expect(sheet).toBeVisible();
      const picker = sheet.locator('[data-hint-family="advisor"] [data-advisor-strength]');
      await expect(picker).toHaveAttribute('data-advisor-strength', 'twoStage');
      await expect(picker.locator('[data-strength="twoStage"]')).toHaveAttribute('data-on', '');
      await picker.locator('[data-strength="deep"]').click();
      await page.waitForFunction(() => window.__ajayeb?.room()?.hints.advisorStrength === 'deep');
      await expect(picker).toHaveAttribute('data-advisor-strength', 'deep');
      await expect(picker.locator('[data-strength-hint]')).toHaveText(
        'A deeper search looks for the better move itself. Slowest, and most thorough.'
      );
      // This device starts its next game from it.
      const saved = await page.evaluate(() => localStorage.getItem('ajayeb:hints'));
      expect(JSON.parse(saved ?? 'null')).toMatchObject({ hints: { advisorStrength: 'deep' } });
      await expect(async () => {
        if ((await sheet.count()) > 0) await page.keyboard.press('Escape');
        await expect(sheet).toHaveCount(0, { timeout: 1_500 });
      }).toPass({ timeout: 8_000 });

      // Walls wins at once; the player discards it instead. The deep
      // review comes after the move, and fills in the summary of the
      // bot's reply. Its own search picks the winning move (Walls, or
      // another card whose shields also reach the capital).
      await playKey(driver, 'discard:walls');
      await driver.nextTurn();
      const popup = page.locator(POPUP);
      await expect(popup).toBeVisible();
      const review = popup.locator('[data-move-review]');
      await expect(review).toHaveAttribute('data-move-review', 'better');
      await expect(review.locator('[data-review-verdict]')).toHaveText(
        /^Building .+ would have won the game\.$/
      );
      const view = await currentView(page);
      expect(view.review?.strength).toBe('deep');
      expect(view.review?.reason).toBe('wins');
      expect(view.review?.gap, 'a deep review has no points').toBe(0);
      expect(view.review?.better).toMatch(/^construct:/);
      await page.keyboard.press('Escape');
      await expect(popup).toHaveCount(0);

      // The room keeps it.
      await page.reload();
      await driver.waitReady();
      expect(await page.evaluate(() => window.__ajayeb?.room()?.hints.advisorStrength)).toBe(
        'deep'
      );
      expect(problems).toEqual([]);
    }
  );
});

test.describe('first-time tips', () => {
  test('the pair and free-chain tips each show once across reloads', async ({ page }) => {
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
    expect(JSON.parse(seen ?? '{}')).toMatchObject({ off: false, seen: ['pair'] });

    // After a reload the pip is back, and its tip is not.
    await page.reload();
    await driver.waitReady();
    await driver.nextTurn();
    await expect(pip.first()).toBeVisible();
    // The pair is remembered, while the next unseen tip explains why
    // Scriptorium lets the Library be built free through its chain.
    await expect(tip).toHaveCount(0);
    const chain = page.locator('[data-first-tip="freeChain"]');
    await expect(chain).toBeVisible();
    await expect(chain).toContainText('A chain badge:');
    await chain.locator('[data-tip-ok]').click();
    const bothSeen = await page.evaluate(() => localStorage.getItem('ajayeb:tips'));
    expect(JSON.parse(bothSeen ?? '{}')).toMatchObject({ off: false, seen: ['pair', 'freeChain'] });

    await page.reload();
    await driver.waitReady();
    await driver.nextTurn();
    await page.waitForTimeout(500);
    await expect(tip).toHaveCount(0);
    await expect(chain).toHaveCount(0);
    expect(problems).toEqual([]);
  });

  test(
    'the don’t-show-again button turns tips off across reloads until Settings turns them on',
    { tag: ['@mobile'] },
    async ({ page }) => {
      test.setTimeout(90_000);
      const problems = watchHygiene(page);
      await tipsOn(page);
      const driver = await openHotseatRoom(page, 'science-pair');
      await driver.nextTurn();

      const tip = page.locator('[data-dismiss-layer] [data-first-tip="pair"]');
      await expect(tip).toBeVisible();
      await expect(tip.locator('[data-tip-off-hint]')).toHaveText('Turn them back on in Settings.');
      await expect(tip.locator('[data-tip-off]')).toHaveText('Don’t show tips again');
      await tip.locator('[data-tip-off]').click();
      await expect(page.locator('[data-first-tip]')).toHaveCount(0);
      const storedOff = await page.evaluate(() => localStorage.getItem('ajayeb:tips'));
      expect(JSON.parse(storedOff ?? '{}')).toMatchObject({ off: true });

      // After a reload all tips stay off, including the next unseen tip (freeChain).
      await page.reload();
      await driver.waitReady();
      await driver.nextTurn();
      await page.waitForTimeout(400);
      await expect(page.locator('[data-first-tip]')).toHaveCount(0);

      // Settings offers "Show first-time tips" next to "Show the tips again", and turns tips back on.
      await driver.dismissToasts();
      await driver.visible('[data-control="menu"]').click();
      await driver.visible('[data-control="menu-settings"]').click();
      const sheet = page.locator('dialog[open]');
      await expect(sheet).toBeVisible();
      const toggle = sheet.locator('[data-setting="tips"]');
      await expect(sheet.locator('[data-settings-hints]')).toContainText('Show first-time tips');
      await expect(sheet.locator('[data-setting="tips-reset"]')).toBeVisible();
      await expect(toggle).not.toBeChecked();
      await toggle.click();
      await expect(toggle).toBeChecked();
      await page.keyboard.press('Escape');
      await expect(sheet).toHaveCount(0);

      await expect(page.locator('[data-dismiss-layer] [data-first-tip="pair"]')).toBeVisible();
      expect(problems).toEqual([]);
    }
  );
});
