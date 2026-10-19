// The final move on the result screen (M11.gameover-last-move): at the end
// of a game no move summary popup opens; the final move first plays out on
// the board, then the result dialog opens and shows that turn itself, in a
// "final move" section above the score sheet built from the move summary's
// own rows, with the viewer's move review under them.
//
// - The order: with Animation on and the summary popup on, every animation
//   of the final frame has ended before the dialog opens, and no popup
//   ever shows.
// - An instant win (military supremacy, the viewer's own Walls against the
//   bot): "Your final move", the Walls row with its pawn chip, the review,
//   no row for the end of the game (the headline says it). It stays
//   through a reload and "View final board", and a rematch starts clean.
// - A civilian win by the other side (the bot's last Age III card, the
//   viewer in seat 1) and in hot-seat (named, never "Your").
//
// On the phone (the mobile project, held upright) the section sits above the
// score sheet; on the desktop (1280x720) the dialog widens and puts it
// beside the sheet. On both the score sheet stays in view without
// scrolling the dialog, as it did before the section came.

import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { clearFx, fxRecords, useSettings } from '../fixtures/anim';
import { GameDriver } from '../fixtures/driver';
import { watchHygiene } from '../fixtures/hygiene';
import { openHotseatRoom, playKey, SCENARIO_SEED } from '../fixtures/scenario';

const POPUP = '[data-dismiss-layer] [data-move-summary]';
const SECTION = 'dialog[open] [data-last-move]';
const TITLE = 'dialog[open] #game-over-title';

/** Turns the opponent move summary popup on for every page load. */
async function summaryOn(page: Page): Promise<void> {
  await page.addInitScript(() => {
    const raw = localStorage.getItem('ajayeb:settings');
    const current = raw ? (JSON.parse(raw) as Record<string, unknown>) : {};
    localStorage.setItem(
      'ajayeb:settings',
      JSON.stringify({ ...current, showOpponentMove: true, confirmMoves: 'never' })
    );
    // A returning player's name: without one, a reload or a rematch of a
    // finished bot room can stop at the name prompt (seen on the desktop
    // project; not this spec's subject).
    if (!localStorage.getItem('ajayeb:name')) localStorage.setItem('ajayeb:name', 'Ana');
  });
}

async function openBotRoom(page: Page, scenario: string, seat: 0 | 1): Promise<GameDriver> {
  const params = new URLSearchParams({
    mode: 'ai',
    level: 'easy',
    seat: String(seat),
    seed: String(SCENARIO_SEED),
    scenario
  });
  await page.goto(`/dev/new?${params.toString()}`);
  await page.waitForURL(/\/play\/[A-Z0-9]+$/);
  const driver = new GameDriver(page);
  await driver.waitReady();
  return driver;
}

/**
 * The section sits above the score sheet, and on a phone held upright and
 * on a desktop the whole sheet shows in the dialog without scrolling.
 */
async function expectLayout(page: Page): Promise<void> {
  const geometry = await page.evaluate(() => {
    const section = document.querySelector('dialog[open] [data-last-move]');
    const table = document.querySelector('dialog[open] [data-score-table]');
    const body = document.querySelector('dialog[open] [data-sheet-body]');
    if (!section || !table || !body) return null;
    const t = table.getBoundingClientRect();
    const b = body.getBoundingClientRect();
    const sec = section.getBoundingClientRect();
    return {
      beside: sec.top < t.bottom && t.top < sec.bottom,
      above: !!(section.compareDocumentPosition(table) & Node.DOCUMENT_POSITION_FOLLOWING),
      scrolled: body.scrollTop,
      tableInView: t.top >= b.top - 0.5 && t.bottom <= b.bottom + 0.5
    };
  });
  expect(geometry, 'the section and the score sheet are in the dialog').not.toBeNull();
  expect(geometry!.above, 'the final move comes before the score sheet').toBe(true);
  const size = page.viewportSize()!;
  expect(geometry!.beside, 'beside the score sheet on a desktop, above it on a phone').toBe(
    size.width >= 1024
  );
  if (size.height >= 700) {
    expect(geometry!.scrolled).toBe(0);
    expect(
      geometry!.tableInView,
      `the score sheet is unscrolled at ${size.width}x${size.height}`
    ).toBe(true);
  }
}

test.describe('the final move on the result screen', () => {
  test('the final move plays out on the board before the result opens, with no popup', async ({
    page
  }) => {
    test.setTimeout(90_000);
    const problems = watchHygiene(page);
    await useSettings(page, 'normal');
    await summaryOn(page);
    const driver = await openBotRoom(page, 'military-supremacy', 0);
    await page.evaluate(() => {
      const seen = { opened: null as number | null, popup: false };
      (window as unknown as { __lastMove: typeof seen }).__lastMove = seen;
      const check = (): void => {
        if (seen.opened === null && document.querySelector('dialog[open] #game-over-title'))
          seen.opened = performance.now();
        if (document.querySelector('[data-dismiss-layer] [data-move-summary]')) seen.popup = true;
      };
      new MutationObserver(check).observe(document.body, {
        subtree: true,
        childList: true,
        attributes: true,
        attributeFilter: ['open']
      });
    });
    await clearFx(page);
    await playKey(driver, 'construct:walls');
    await expect(page.locator(TITLE)).toBeVisible({ timeout: 20_000 });
    await expect(page.locator(SECTION)).toBeVisible();
    const seen = await page.evaluate(
      () =>
        (window as unknown as { __lastMove: { opened: number | null; popup: boolean } }).__lastMove
    );
    const { anims } = await fxRecords(page);
    expect(anims.length, 'the final move animates').toBeGreaterThan(0);
    expect(anims.map((a) => a.kind)).toContain('pawn');
    expect(anims.every((a) => a.endAt !== null)).toBe(true);
    const lastEnd = Math.max(...anims.map((a) => a.endAt ?? Infinity));
    expect(seen.opened, 'the result dialog opened').not.toBeNull();
    expect(seen.opened!, 'every animation ended before the result opened').toBeGreaterThanOrEqual(
      lastEnd
    );
    expect(seen.popup, 'no move summary popup at the end of the game').toBe(false);
    await expect(page.locator(POPUP)).toHaveCount(0);
    expect(problems).toEqual([]);
  });

  test("an instant win by the viewer's own move: its row, its review, through reload and rematch", async ({
    page
  }) => {
    test.setTimeout(90_000);
    const problems = watchHygiene(page);
    await summaryOn(page);
    const driver = await openBotRoom(page, 'military-supremacy', 0);
    await playKey(driver, 'construct:walls');
    await expect(page.locator(TITLE)).toBeVisible();

    const expectSection = async (): Promise<void> => {
      const section = page.locator(SECTION);
      await expect(section).toHaveAttribute('data-last-move', '0');
      await expect(section.locator('h3')).toHaveText('Your final move');
      const rows = section.locator('[data-summary-line]');
      await expect(rows).toHaveCount(1);
      await expect(rows.first()).toHaveAttribute('data-summary-line', 'build');
      await expect(rows.first().locator('[data-card="walls"]')).toBeVisible();
      await expect(rows.first().locator('[data-chip="pawn"]')).toBeVisible();
      await expect(section.locator('[data-summary-line="gameOver"]')).toHaveCount(0);
      await expect(section.locator('[data-end-review] [data-move-review]')).toBeVisible();
      await expect(page.locator(POPUP)).toHaveCount(0);
      await expectLayout(page);
    };
    await expectSection();

    // A reload on the result screen brings the same section back.
    await page.reload();
    await driver.waitReady();
    await expect(page.locator(TITLE)).toBeVisible();
    await expectSection();

    // "View final board" and back.
    await page.locator('dialog[open] [data-view-board]').click();
    await expect(page.locator(TITLE)).toHaveCount(0);
    await page.locator('[data-game-over] [data-peek-open]').click();
    await expectSection();

    // A rematch starts a new game with no result and no popup.
    const previous = page.url();
    await page.locator('dialog[open] [data-rematch]').click();
    await page.waitForURL((url) => url.href !== previous);
    await new GameDriver(page).waitReady();
    await expect(page.locator('[data-phase="wonderDraft"]')).toBeVisible();
    await expect(page.locator('[data-game-over]')).toHaveCount(0);
    await expect(page.locator(POPUP)).toHaveCount(0);
    expect(problems).toEqual([]);
  });

  test("a civilian win by the other side's move: the bot's last Age III card", async ({ page }) => {
    test.setTimeout(90_000);
    const problems = watchHygiene(page);
    await summaryOn(page);
    // The viewer takes seat 1; the bot, in seat 0, is to move and ends the game.
    await openBotRoom(page, 'civilian-end', 1);
    await expect(page.locator(TITLE)).toBeVisible({ timeout: 20_000 });
    const section = page.locator(SECTION);
    await expect(section).toHaveAttribute('data-last-move', '0');
    await expect(section.locator('h3')).toHaveText(/\S's final move$/);
    await expect(section.locator('h3')).not.toHaveText('Your final move');
    await expect(section.locator('[data-summary-line]').first()).toBeVisible();
    await expect(section).toContainText('Academy');
    await expect(section.locator('[data-summary-line="gameOver"]')).toHaveCount(0);
    await expect(page.locator(POPUP)).toHaveCount(0);
    await expect(page.locator('[data-game-over]')).toHaveAttribute('data-victory', 'civilian');
    await expectLayout(page);
    expect(problems).toEqual([]);
  });

  test('a civilian win in hot-seat names the player who ended it', async ({ page }) => {
    const problems = watchHygiene(page);
    await summaryOn(page);
    const driver = await openHotseatRoom(page, 'civilian-end');
    await playKey(driver, 'discard:academy');
    await expect(page.locator(TITLE)).toBeVisible();
    const section = page.locator(SECTION);
    await expect(section).toHaveAttribute('data-last-move', '0');
    // Hot-seat names the player (the stored name or the crest's), never "Your".
    await expect(section.locator('h3')).toHaveText(/^(Ana|Sun)'s final move$/);
    const row = section.locator('[data-summary-line="discard"]');
    await expect(row).toHaveCount(1);
    await expect(row.locator('[data-card="academy"]')).toBeVisible();
    await expect(page.locator(POPUP)).toHaveCount(0);
    await expectLayout(page);
    expect(problems).toEqual([]);
  });
});
