// Helpers for the specs that start a hot-seat game at a named scenario (or a
// fixed seed) and play its trigger through the real UI: victory.spec.ts and
// flows.spec.ts.
//
// A room is opened the way a developer opens one, through the test build's
// /dev/new page, which asks the server's POST /api/dev/rooms for it and then
// shows /play/<code>. Every room here names its seed, so none of them uses up
// a seed from the server's --seed-base counter: the games the other specs
// create over the socket stay the same whether or not these specs ran first.
//
// Nothing here sends a frame or decides what is legal. The helpers only read
// the page's test hook (window.__ajayeb) and click what a player clicks,
// through the shared GameDriver.

import { expect, type Page } from '@playwright/test';
import type { Choice, View } from '../../web/src/lib/wire';
import { GameDriver, type Flow, type Snapshot } from './driver';

/** The seed every scenario room is dealt from (its random parts: tokens, decks). */
export const SCENARIO_SEED = 7;

/** The English seat names of a hot-seat room whose players were not named. */
export const CREST_NAMES = ['Sun', 'Moon'] as const;

/**
 * Opens a hot-seat room at a named scenario, or at the ordinary start when
 * `scenario` is null, and waits for its first state. Returns the driver.
 */
export async function openHotseatRoom(
  page: Page,
  scenario: string | null,
  seed = SCENARIO_SEED
): Promise<GameDriver> {
  const params = new URLSearchParams({ mode: 'hotseat', seed: String(seed) });
  if (scenario !== null) params.set('scenario', scenario);
  await page.goto(`/dev/new?${params.toString()}`);
  await page.waitForURL(/\/play\/[A-Z0-9]+$/);
  const driver = new GameDriver(page);
  await driver.waitReady();
  const room = await page.evaluate(() => window.__ajayeb?.room() ?? null);
  expect(room?.mode, 'a dev room opened from a scenario is a hot-seat room').toBe('hotseat');
  return driver;
}

/** The view on screen (a copy). */
export async function currentView(page: Page): Promise<View> {
  const view = await page.evaluate(() => window.__ajayeb?.view() ?? null);
  if (!view) throw new Error('no view on screen');
  return view;
}

/**
 * Plays one action key of a scenario's trigger through the UI, by the flow a
 * player would use for it, and waits for the room to move on. The key must
 * be one of the view's choices; the snapshot it was chosen in is returned
 * with the choice.
 */
export async function playKey(
  driver: GameDriver,
  key: string,
  flow?: Flow
): Promise<{ before: Snapshot; choice: Choice }> {
  const before = await driver.nextTurn();
  const choice = before.choices.find((c) => c.key === key);
  expect(
    choice,
    `${key} is one of the choices: ${before.choices.map((c) => c.key).join(', ')}`
  ).toBeTruthy();
  if (!choice) throw new Error(`${key} is not a choice`);
  await driver.perform({ flow: flow ?? defaultFlow(before, choice), choice }, before);
  await driver.waitAdvance(before.version);
  return { before, choice };
}

/** The flow a player uses for a choice: the panel, the draft, the card or the wonder. */
export function defaultFlow(snap: Snapshot, choice: Choice): Flow {
  if (snap.pending) return 'pending';
  switch (choice.kind) {
    case 'draft':
      return 'draft';
    case 'construct':
      return 'construct';
    case 'discard':
      return 'discard';
    case 'wonder':
      return 'wonderCardFirst';
    default:
      return 'pending';
  }
}

/**
 * The game log's lines, read from the log sheet the top bar opens (after
 * closing any toast that could cover the button), then closed again.
 */
export async function logLines(driver: GameDriver): Promise<string[]> {
  const { page } = driver;
  await driver.dismissToasts();
  await driver.visible('[data-control="log"]').click();
  const sheet = page.locator('dialog[open]').filter({ has: page.locator('#log-sheet-title') });
  await expect(sheet).toBeVisible();
  const lines = (await sheet.locator('li').allTextContents()).map((s) => s.trim());
  await page.keyboard.press('Escape');
  await expect(sheet).toHaveCount(0);
  return lines;
}

/** What the result screen should say, from the scenario's Expect. */
export interface ExpectedResult {
  victory: 'civilian' | 'military' | 'science' | 'shared';
  /** The winning seat, or null for a shared victory. */
  winner: 0 | 1 | null;
  /** The totals tie and blue points broke it. */
  tieBreak?: boolean;
}

/** The score sheet's lines, in the order the table shows them. */
const SCORE_ROWS = [
  'blue',
  'green',
  'yellow',
  'purple',
  'wonders',
  'tokens',
  'coins',
  'military',
  'total'
] as const;

/**
 * Checks the result screen of a finished game: the server's result has the
 * expected kind and winner, the screen says the same (its data attributes,
 * kind line and headline), the tie-break note shows exactly when expected,
 * and the score table shows the server's score sheet line by line. Returns
 * the server's result.
 */
export async function expectResultScreen(
  page: Page,
  expected: ExpectedResult,
  names: readonly [string, string] = CREST_NAMES
): Promise<NonNullable<View['result']>> {
  const view = await currentView(page);
  expect(view.phase).toBe('gameOver');
  const result = view.result;
  expect(result, 'a finished game carries its result').toBeTruthy();
  if (!result) throw new Error('no result');

  // The server's own result.
  expect(result.shared).toBe(expected.victory === 'shared');
  if (expected.victory !== 'shared') expect(result.victory).toBe(expected.victory);
  expect(result.winner ?? null).toBe(expected.winner);

  // The screen.
  const over = page.locator('[data-game-over]');
  await expect(over).toHaveAttribute('data-victory', expected.victory);
  if (expected.winner === null) {
    await expect(over).not.toHaveAttribute('data-winner');
  } else {
    await expect(over).toHaveAttribute('data-winner', String(expected.winner));
  }
  const dialog = page.locator('dialog[open]').filter({ has: page.locator('[data-score-table]') });
  await expect(dialog).toBeVisible();
  const kindText = {
    civilian: 'Civilian victory',
    military: 'Military supremacy',
    science: 'Scientific supremacy',
    shared: 'Shared victory'
  }[expected.victory];
  await expect(dialog.locator('header')).toContainText(kindText);
  await expect(dialog.locator('h2')).toHaveText(
    expected.winner === null ? 'Both players win' : `${names[expected.winner]} wins`
  );
  await expect(dialog.locator('[data-tie-break]')).toHaveCount(expected.tieBreak ? 1 : 0);

  const table = dialog.locator('[data-score-table]');
  await expect(table.locator('thead')).toContainText(names[0]);
  await expect(table.locator('thead')).toContainText(names[1]);
  for (const row of SCORE_ROWS) {
    await expect(table.locator(`[data-score-row="${row}"] td`)).toHaveText([
      String(result.scores[0][row]),
      String(result.scores[1][row])
    ]);
  }
  return result;
}
