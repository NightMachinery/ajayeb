// The opponent move summary (M10.S03b), through the real UI on the desktop
// and mobile projects.
//
// - Against the bot: once the bot has moved, a popup lists its moves (the
//   bot's last run of history entries, consolidated); a tap on a card of
//   the structure closes it without acting (the card's sheet does not
//   open), and Escape closes it too. With Settings' "Show opponent move in
//   detail" off, the popup no longer opens by itself, but the top bar's
//   button opens the latest summary.
// - Hot-seat with the pass-device screen: the popup waits for the next
//   player to take the device, then shows the previous player's move.
//
// The base fixture turns the popup off for every other spec (its taps would
// eat a driver's clicks), so this spec turns it back on for its pages.

import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { GameDriver, type Snapshot } from '../fixtures/driver';
import { watchHygiene } from '../fixtures/hygiene';

const POPUP = '[data-dismiss-layer] [data-move-summary]';

/** Turns the popup back on (its default) for every page load of this test. */
async function summaryOn(page: Page): Promise<void> {
  // A page-level init script runs after the base fixture's context-level one.
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

/** The plies of the other side's moves since `seat` last moved, as the page's history has them. */
async function pliesSince(page: Page, seat: number): Promise<string> {
  return page.evaluate((viewer) => {
    const history = window.__ajayeb?.history() ?? [];
    let i = history.length;
    while (i > 0 && history[i - 1]!.seat !== viewer) i--;
    return history
      .slice(i)
      .map((e) => e.ply)
      .join(' ');
  }, seat);
}

/** The seat of the latest history entry, or null before any move. */
async function lastMover(page: Page): Promise<number | null> {
  return page.evaluate(() => {
    const history = window.__ajayeb?.history() ?? [];
    return history[history.length - 1]?.seat ?? null;
  });
}

/** Taps (on touch) or clicks a locator; `force` presses even a covered, inert card. */
async function press(
  page: Page,
  isMobile: boolean,
  selector: string,
  force = false
): Promise<void> {
  const target = page.locator(selector).filter({ visible: true }).first();
  if (isMobile) await target.tap({ force });
  else await target.click({ force });
}

/** Plays this client's moves until the bot has just moved in play; returns the turn then. */
async function untilBotMoved(driver: GameDriver, bot: number): Promise<Snapshot> {
  for (let guard = 0; guard < 30; guard++) {
    const snap = await driver.nextTurn();
    if (snap.phase === 'play' && snap.pending === null && (await lastMover(driver.page)) === bot) {
      return snap;
    }
    // A popup over a choice of ours (a new age's starter) would eat the click.
    if ((await driver.page.locator(POPUP).count()) > 0) await driver.page.keyboard.press('Escape');
    await driver.play(snap);
  }
  throw new Error('the bot did not move in play within 30 turns');
}

test.describe('opponent move summary', { tag: ['@mobile'] }, () => {
  test('against the bot: shows after its move, a tap closes it without acting, the button reopens it', async ({
    page,
    isMobile
  }) => {
    test.setTimeout(150_000);
    const problems = watchHygiene(page);
    await summaryOn(page);
    await page.goto('/');
    await page.getByRole('button', { name: /Play vs AI/ }).click();
    const sheet = page.locator('dialog[open]');
    await sheet.getByText('Apprentice', { exact: true }).click();
    await sheet.getByText('Sun', { exact: true }).click();
    await sheet.getByRole('button', { name: 'Start game' }).click();
    await page.waitForURL(/\/play\/[A-Z0-9]+$/);
    const driver = new GameDriver(page);
    await driver.waitReady();

    // No popup in the wonder draft, even when the bot picks.
    const first = await driver.nextTurn();
    expect(first.phase).toBe('wonderDraft');
    await expect(page.locator(POPUP)).toHaveCount(0);

    // The bot has just moved: the popup lists its moves, as the history has them.
    const snap = await untilBotMoved(driver, 1);
    const popup = page.locator(POPUP);
    await expect(popup).toBeVisible();
    await expect(popup).toHaveAttribute('data-move-summary', '1');
    await expect(popup).toHaveAttribute('data-plies', await pliesSince(page, 0));
    await expect(popup.locator('[data-summary-line]').first()).toBeVisible();
    await expect(popup.locator('h2')).toContainText("'s move");

    // A tap on a takeable card closes it, and the card's sheet does not open.
    const card = snap.choices.find((c) => c.kind === 'construct' || c.kind === 'discard')?.card;
    if (!card) throw new Error('no card to take');
    const cardSelector = `.card-structure [data-card="${card}"]`;
    await press(page, isMobile, cardSelector);
    await expect(page.locator(POPUP)).toHaveCount(0);
    await expect(page.locator('dialog[open] [data-card-sheet]')).toHaveCount(0);
    expect((await driver.snapshot()).version).toBe(snap.version);
    // The next tap acts as usual.
    await press(page, isMobile, cardSelector);
    await expect(page.locator(`dialog[open] [data-card-sheet="${card}"]`)).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(page.locator('dialog[open] [data-card-sheet]')).toHaveCount(0);

    // The button opens it again; Escape closes it.
    await press(page, isMobile, '[data-control="move-summary"]');
    await expect(page.locator(POPUP)).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(page.locator(POPUP)).toHaveCount(0);

    // Option off: the popup no longer opens by itself after the bot's move...
    // Settings through the game menu (a narrow phone's top bar has no gear).
    await press(page, isMobile, '[data-control="menu"]');
    await press(page, isMobile, '[data-control="menu-settings"]');
    const option = page.locator('dialog[open] [data-setting="show-opponent-move"]');
    await expect(option).toBeChecked();
    await option.setChecked(false);
    await page.keyboard.press('Escape');
    await expect(page.locator('dialog[open]')).toHaveCount(0);
    const saved = await page.evaluate(() => localStorage.getItem('ajayeb:settings') ?? '{}');
    expect(JSON.parse(saved)).toMatchObject({ showOpponentMove: false });

    await driver.play(await driver.nextTurn());
    const later = await untilBotMoved(driver, 1);
    expect(later.ply).toBeGreaterThan(snap.ply);
    await page.waitForTimeout(300);
    await expect(page.locator(POPUP)).toHaveCount(0);

    // ...but the button near the log still shows the latest one.
    await press(page, isMobile, '[data-control="move-summary"]');
    await expect(page.locator(POPUP)).toBeVisible();
    await expect(page.locator(POPUP)).toHaveAttribute('data-plies', await pliesSince(page, 0));
    await page.keyboard.press('Escape');
    await expect(page.locator(POPUP)).toHaveCount(0);
    expect(problems).toEqual([]);
  });

  test('hot-seat: after the pass-device screen, the previous player’s move', async ({
    page,
    isMobile
  }) => {
    test.setTimeout(150_000);
    const problems = watchHygiene(page);
    await summaryOn(page);
    await page.goto('/');
    await page.getByRole('button', { name: /^Pass and play/ }).click();
    const sheet = page.locator('dialog[open]');
    await expect(sheet.getByRole('heading', { name: 'Pass and play' })).toBeVisible();
    await sheet.getByRole('checkbox').setChecked(true);
    await sheet.getByRole('button', { name: 'Start game' }).click();
    await page.waitForURL(/\/play\/[A-Z0-9]+$/);
    const driver = new GameDriver(page);
    await driver.waitReady();

    // Play until a move in Age I hands the device to the other player.
    for (let guard = 0; guard < 30; guard++) {
      await driver.waitTurn();
      const snap = await driver.snapshot();
      if (snap.pass) {
        if (snap.phase === 'play') break;
        await driver.passDevice();
        continue;
      }
      await driver.play(snap);
    }
    const pass = page.locator('[data-pass-device]');
    await expect(pass).toBeVisible();
    // Not while the device is being passed...
    await expect(page.locator(POPUP)).toHaveCount(0);
    const next = Number(await pass.getAttribute('data-pass-device'));
    const previous = next === 0 ? 1 : 0;
    await driver.passDevice();

    // ...but once the next player has it: the previous player's move.
    const popup = page.locator(POPUP);
    await expect(popup).toBeVisible();
    await expect(popup).toHaveAttribute('data-move-summary', String(previous));
    await expect(popup).toHaveAttribute('data-plies', await pliesSince(page, next));

    // Any tap closes it without acting.
    const version = (await driver.snapshot()).version;
    await press(page, isMobile, '.card-structure [data-card]', true);
    await expect(page.locator(POPUP)).toHaveCount(0);
    await expect(page.locator('dialog[open]')).toHaveCount(0);
    expect((await driver.snapshot()).version).toBe(version);
    expect(problems).toEqual([]);
  });
});
