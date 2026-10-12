// Per-card hint pips (the hints design, M10.HS2), on the desktop and mobile
// projects.
//
// A seeded hot-seat game is played over the WebSocket driver until the seat
// to move has a takeable card that completes one of its science pairs while
// a progress token is on the board. The browser then opens the same game at
// that position. On the mover's turn the card carries a solid pip
// (`data-hint="pair"`, `data-hint-side="own"`) and its card sheet says so;
// after the mover discards another card the other seat is to move, and the
// same card carries an outlined pip for the first seat (a deny target).
// Switching the science family off in the game menu removes every pair pip.
//
// The policy decides nothing on its own: every key it returns is copied
// from the state's choices, and it reads only the hints the server sent.

import { expect, type Locator } from '@playwright/test';
import type { Choice, View } from '../../web/src/lib/wire';
import { test } from '../fixtures/base';
import { GameDriver } from '../fixtures/driver';
import { watchHygiene } from '../fixtures/hygiene';
import { playKey } from '../fixtures/scenario';
import { createDevRoom, recordGame, recordPolicy } from '../fixtures/ws-game';

/** Seeds tried in turn until one reaches a pair position. */
const SEEDS = ['21', '22', '23', '24', '25', '26'];

interface PairPosition {
  seed: string;
  /** The keys that lead to the position. */
  keys: string[];
  /** The seat to move there. */
  seat: 0 | 1;
  /** The card that completes the mover's pair. */
  card: string;
  /** Another takeable card, for the mover to discard. */
  other: string;
}

/** Thrown by the policy to stop the recording at the wanted position. */
class Found extends Error {
  constructor(readonly position: Omit<PairPosition, 'seed' | 'keys'>) {
    super('found');
  }
}

function pairPosition(view: View): Omit<PairPosition, 'seed' | 'keys'> | null {
  if (view.phase !== 'play' || view.pending) return null;
  const seat = view.toMove === 1 ? 1 : 0;
  const takeable = view.structure.filter((s) => s.state === 'faceUp' && s.accessible && s.card);
  const pair = takeable.find((s) => s.hints?.[seat].codes.includes('pair'));
  // The other seat must not have its own pair pip on that card, so the pip
  // shown on its turn is the first seat's.
  if (!pair || pair.hints?.[seat === 0 ? 1 : 0].codes.includes('pair')) return null;
  const other = takeable.find((s) => s.card !== pair.card);
  if (!other) return null;
  const discard = view.choices.some((c) => c.key === `discard:${other.card}`);
  return discard ? { seat, card: pair.card!, other: other.card! } : null;
}

/** Builds science cards when it can, so a pair turns up early. */
function sciencePolicy(view: View): Choice {
  const found = pairPosition(view);
  if (found) throw new Found(found);
  const science = view.choices.find(
    (c) =>
      c.kind === 'construct' &&
      view.structure.some(
        (s) => s.card === c.card && s.hints?.[view.toMove === 1 ? 1 : 0].effect?.science
      )
  );
  return science ?? recordPolicy(view);
}

let recorded: Promise<PairPosition> | null = null;

function position(origin: string): Promise<PairPosition> {
  recorded ??= (async () => {
    for (const seed of SEEDS) {
      const keys: string[] = [];
      try {
        await recordGame(origin, seed, (view) => {
          const choice = sciencePolicy(view);
          keys.push(choice.key);
          return choice;
        });
      } catch (err) {
        if (err instanceof Found) {
          // The policy threw before its key was pushed, so `keys` is exactly
          // what was applied.
          return { seed, keys, ...err.position };
        }
      }
    }
    throw new Error(`no seed of ${SEEDS.join(', ')} reached a pair position`);
  })();
  return recorded;
}

function pipOn(driver: GameDriver, card: string): Locator {
  return driver.page.locator(`.card-structure [data-card="${card}"] [data-hint]`);
}

test.describe('hint pips', { tag: ['@mobile'] }, () => {
  test('a pair pip, solid for the mover and outlined for the other seat', async ({
    page,
    baseURL
  }) => {
    test.setTimeout(150_000);
    if (!baseURL) throw new Error('no baseURL');
    const problems = watchHygiene(page);
    const at = await position(baseURL);
    // The room comes from this test's own HTTP request, so a stored name
    // keeps the room page from asking for one, and no pass-device screen
    // sits between the turns (as in layout.spec.ts, which also explains why
    // animationSpeed is repeated here).
    await page.addInitScript(() => {
      try {
        localStorage.setItem(
          'ajayeb:settings',
          JSON.stringify({ hotseatPassDevice: false, animationSpeed: 'off' })
        );
        localStorage.setItem('ajayeb:name', 'Hints');
      } catch {
        // Storage may be blocked; the room then asks for a name and fails.
      }
    });
    const room = await createDevRoom(baseURL, {
      mode: 'hotseat',
      seed: at.seed,
      actions: at.keys
    });
    await page.goto(room.path);
    const driver = new GameDriver(page);
    await driver.waitReady();
    expect(await page.evaluate(() => window.__ajayeb?.view()?.toMove)).toBe(at.seat);

    // The mover's own pair: solid.
    const pip = pipOn(driver, at.card);
    await expect(pip).toHaveAttribute('data-hint', 'pair');
    await expect(pip).toHaveAttribute('data-hint-side', 'own');
    await expect(pip).toHaveAttribute('data-hint-seat', String(at.seat));
    await expect(pip).toHaveAttribute('data-style', 'solid');

    // Its card sheet says the same, in the mover's section.
    await driver.dismissToasts();
    await page.locator(`.card-structure [data-card="${at.card}"]`).click();
    const sheet = page.locator('dialog[open] [data-card-sheet]');
    await expect(sheet).toBeVisible();
    await expect(
      sheet.locator(`[data-hint-section][data-hint-seat="${at.seat}"] [data-hint-line="pair"]`)
    ).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(page.locator('dialog[open]')).toHaveCount(0);

    // The mover discards another card; on the other seat's turn the same
    // card is a deny target: the first seat's pip, outlined.
    await playKey(driver, `discard:${at.other}`);
    expect(await page.evaluate(() => window.__ajayeb?.view()?.toMove)).toBe(at.seat === 0 ? 1 : 0);
    await expect(pip).toHaveAttribute('data-hint', 'pair');
    await expect(pip).toHaveAttribute('data-hint-side', 'opp');
    await expect(pip).toHaveAttribute('data-hint-seat', String(at.seat));
    await expect(pip).toHaveAttribute('data-style', 'outline');

    // Science switched off: no pair pip anywhere.
    await driver.dismissToasts();
    await driver.visible('[data-control="menu"]').click();
    await driver.visible('[data-control="menu-hints"]').click();
    const hints = page.locator('dialog[open]').filter({ has: page.locator('[data-game-hints]') });
    await expect(hints).toBeVisible();
    const science = hints.locator('[data-hint-family="science"]');
    await science.getByRole('switch').click();
    await expect(science).not.toHaveAttribute('data-on');
    // The switch keeps focus and is disabled until the server answers; an
    // Escape that lands on it then is lost, so press again until the sheet
    // closes (hints.spec.ts closeSheet has the same race).
    await expect(async () => {
      await page.keyboard.press('Escape');
      await expect(hints).toHaveCount(0, { timeout: 1_000 });
    }).toPass({ timeout: 10_000 });
    await page.waitForFunction(() => window.__ajayeb?.room()?.hints.science === false);
    await expect(page.locator('.card-structure [data-hint="pair"]')).toHaveCount(0);

    expect(problems).toEqual([]);
  });
});
