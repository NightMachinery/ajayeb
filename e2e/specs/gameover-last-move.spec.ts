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
// The section sits above the score sheet on a phone held upright and beside
// it where two columns fit (the split query: a desktop, or a short
// landscape screen). The mobile project's phone (412x839), 390x844 and the
// desktop (1280x720) show it all without scrolling; a small phone (360x740)
// and phone landscape (740x360) scroll the body to the totals, the footer
// in view.

import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { clearFx, fxRecords, useSettings, type AnimRecord } from '../fixtures/anim';
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

/** The client's `split` variant (web/src/app.css): room for two columns. */
const SPLIT = '(min-width: 64rem), (orientation: landscape) and (max-height: 32rem)';

/**
 * The section comes before the score sheet: beside it where two columns
 * fit (the split query), above it otherwise. Where the dialog has the room
 * (a phone of 800 px or more held upright, a desktop) nothing scrolls;
 * everywhere, scrolling the body reaches the totals while the footer's
 * buttons stay in view.
 */
async function expectLayout(page: Page): Promise<void> {
  const size = page.viewportSize()!;
  const at = `${size.width}x${size.height}`;
  const geometry = await page.evaluate((split) => {
    const section = document.querySelector('dialog[open] [data-last-move]');
    const table = document.querySelector<HTMLTableElement>('dialog[open] [data-score-table]');
    const body = document.querySelector<HTMLElement>('dialog[open] [data-sheet-body]');
    if (!section || !table || !body) return null;
    const t = table.getBoundingClientRect();
    const sec = section.getBoundingClientRect();
    return {
      split: matchMedia(split).matches,
      beside: sec.top < t.bottom && t.top < sec.bottom,
      above: !!(section.compareDocumentPosition(table) & Node.DOCUMENT_POSITION_FOLLOWING)
    };
  }, SPLIT);
  expect(geometry, `${at}: the section and the score sheet are in the dialog`).not.toBeNull();
  expect(geometry!.above, `${at}: the final move comes before the score sheet`).toBe(true);
  expect(geometry!.beside, `${at}: beside the sheet where two columns fit, else above`).toBe(
    geometry!.split
  );
  if (size.height >= 800 || (geometry!.split && size.height >= 700)) {
    // Polled: after a viewport change the dialog settles over a few frames.
    await expect
      .poll(
        () =>
          page.evaluate(() => {
            const body = document.querySelector<HTMLElement>('dialog[open] [data-sheet-body]');
            return !!body && body.scrollHeight <= body.clientHeight + 1;
          }),
        { message: `${at}: the dialog shows everything without scrolling` }
      )
      .toBe(true);
  }
  // The totals are reachable, with the footer in view.
  const reach = await page.evaluate(() => {
    const body = document.querySelector<HTMLElement>('dialog[open] [data-sheet-body]')!;
    const table = document.querySelector<HTMLTableElement>('dialog[open] [data-score-table]')!;
    const from = body.scrollTop;
    body.scrollTop = body.scrollHeight;
    const b = body.getBoundingClientRect();
    const total = table.rows[table.rows.length - 1]!.getBoundingClientRect();
    const buttons = [...document.querySelectorAll('dialog[open] [data-sheet-footer] button')].map(
      (el) => el.getBoundingClientRect()
    );
    const result = {
      totalInView: total.top >= b.top - 0.5 && total.bottom <= b.bottom + 0.5,
      footerInView:
        buttons.length > 0 &&
        buttons.every((r) => r.top >= 0 && r.bottom <= innerHeight && r.height >= 44)
    };
    body.scrollTop = from;
    return result;
  });
  expect(reach.totalInView, `${at}: scrolling reaches the totals`).toBe(true);
  expect(reach.footerInView, `${at}: the footer's buttons stay in view`).toBe(true);
}

test.describe('the final move on the result screen @mobile', () => {
  for (const cpu of [1, 4, 6]) {
    test(`the final move plays out on the board before the result opens, with no popup (${cpu}x CPU)`, async ({
      page,
      context
    }) => {
      test.setTimeout(90_000);
      const cdp = await context.newCDPSession(page);
      await cdp.send('Emulation.setCPUThrottlingRate', { rate: cpu });
      try {
        const problems = watchHygiene(page);
        await useSettings(page, 'normal');
        await summaryOn(page);
        const driver = await openBotRoom(page, 'military-supremacy', 0);
        await page.evaluate(() => {
          const seen = { opened: null as number | null, popup: false, anims: [] as AnimRecord[] };
          (window as unknown as { __lastMove: typeof seen }).__lastMove = seen;
          // Capture before the native call, not a later observer/poll: a
          // slow runner must never hide an early open from this assertion.
          const native = HTMLDialogElement.prototype.showModal;
          HTMLDialogElement.prototype.showModal = function () {
            if (seen.opened === null && this.querySelector('#game-over-title')) {
              seen.opened = performance.now();
              seen.anims =
                (
                  window as unknown as {
                    __ajayebFx?: { anims(): AnimRecord[] };
                  }
                ).__ajayebFx?.anims() ?? [];
            }
            native.call(this);
          };
          new MutationObserver(() => {
            if (document.querySelector('[data-dismiss-layer] [data-move-summary]'))
              seen.popup = true;
          }).observe(document.body, {
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
            (
              window as unknown as {
                __lastMove: { opened: number | null; popup: boolean; anims: AnimRecord[] };
              }
            ).__lastMove
        );
        const { anims } = await fxRecords(page);
        await test.info().attach('final move at native result open', {
          body: JSON.stringify({ cpu, seen, anims }),
          contentType: 'application/json'
        });
        expect(anims.length, 'the final move animates').toBeGreaterThan(0);
        expect(anims.map((a) => a.kind)).toContain('pawn');
        expect(anims.every((a) => a.endAt !== null)).toBe(true);
        expect(
          anims.every((a) => !a.interrupted),
          'the final move completes naturally'
        ).toBe(true);
        expect(seen.anims, 'all recorded effects already ended at native showModal').toEqual(anims);
        const lastEnd = Math.max(...anims.map((a) => a.endAt ?? Infinity));
        expect(seen.opened, 'the result dialog opened').not.toBeNull();
        expect(
          seen.opened!,
          'every animation ended before the result opened'
        ).toBeGreaterThanOrEqual(lastEnd);
        test.info().annotations.push({
          type: 'final-move-order',
          description: `${cpu}x CPU: ${anims.length} naturally finished effects; result opened ${Math.round(seen.opened! - lastEnd)} ms after the last end`
        });
        expect(seen.popup, 'no move summary popup at the end of the game').toBe(false);
        await expect(page.locator(POPUP)).toHaveCount(0);
        expect(problems).toEqual([]);
      } finally {
        await cdp.detach();
      }
    });
  }

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

  test('on a small phone and in phone landscape the totals are a scroll away, the footer in view', async ({
    page
  }) => {
    const problems = watchHygiene(page);
    await summaryOn(page);
    const driver = await openBotRoom(page, 'military-supremacy', 0);
    await playKey(driver, 'construct:walls');
    await expect(page.locator(TITLE)).toBeVisible();
    for (const size of [
      { width: 360, height: 740 },
      { width: 740, height: 360 },
      { width: 390, height: 844 }
    ]) {
      await page.setViewportSize(size);
      await expect(page.locator(SECTION)).toBeVisible();
      await expectLayout(page);
    }
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
