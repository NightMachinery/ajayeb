// The decider's hints (the hints design, M10.HS3), on the desktop and
// mobile projects.
//
// A seeded hot-seat game is played over the WebSocket driver until the seat
// to move has a takeable card whose build wins the game (the server's
// Choice.outcome.wins) and that it may also discard. The browser then opens
// the same game at that position, with Confirm moves on. The card carries
// its win pip, and its card sheet puts a crown and the win first in the
// outcome line under Build. The first tap on Discard arms it, and the
// confirm step then warns that the mover could win this turn instead; the
// warning is hidden until then.
//
// The policy decides nothing on its own: every key it returns is copied
// from the state's choices, and it reads only what the server sent.

import { expect } from '@playwright/test';
import type { Choice, View } from '../../web/src/lib/wire';
import { test } from '../fixtures/base';
import { GameDriver } from '../fixtures/driver';
import { currentView } from '../fixtures/scenario';
import { watchHygiene } from '../fixtures/hygiene';
import { createDevRoom, recordGame, recordPolicy } from '../fixtures/ws-game';

/** Seeds tried in turn until one reaches a winning build. */
const SEEDS = ['31', '32', '33', '34', '35', '36', '37', '38', '39', '40', '41', '42'];

interface WinPosition {
  seed: string;
  /** The keys that lead to the position. */
  keys: string[];
  /** The seat to move there. */
  seat: 0 | 1;
  /** The card whose build wins. */
  card: string;
}

/** Thrown by the policy to stop the recording at the wanted position. */
class Found extends Error {
  readonly position: Omit<WinPosition, 'seed' | 'keys'>;
  constructor(position: Omit<WinPosition, 'seed' | 'keys'>) {
    super('found');
    this.position = position;
  }
}

function winPosition(view: View): Omit<WinPosition, 'seed' | 'keys'> | null {
  if (view.phase !== 'play' || view.pending) return null;
  const seat = view.toMove === 1 ? 1 : 0;
  const win = view.choices.find(
    (c) =>
      c.kind === 'construct' &&
      // A supremacy, which the board's win pip also marks (a civilian win
      // on the last card is said by the end term instead).
      (c.outcome?.wins?.kind === 'military' || c.outcome?.wins?.kind === 'science') &&
      view.choices.some((d) => d.key === `discard:${c.card}`)
  );
  return win?.card ? { seat, card: win.card } : null;
}

/**
 * Seat 0 builds what moves the pawn its way, then what adds a science
 * symbol; seat 1 discards. A win for seat 0 turns up before the game ends.
 */
function racePolicy(view: View): Choice {
  const found = winPosition(view);
  if (found) throw new Found(found);
  const builds = view.choices.filter((c) => c.kind === 'construct');
  if (view.toMove === 0 && view.phase === 'play') {
    const pawn = builds.find((c) => c.outcome?.pawn && c.outcome.pawn.to > c.outcome.pawn.from);
    const science = builds.find((c) => c.outcome?.distinct !== undefined);
    if (pawn ?? science) return (pawn ?? science)!;
  }
  if (view.toMove === 1 && view.phase === 'play') {
    const discard = view.choices.find((c) => c.kind === 'discard');
    if (discard) return discard;
  }
  return recordPolicy(view);
}

test('a card’s score preview labels the total and explains coin points @mobile', async ({
  page,
  baseURL
}) => {
  if (!baseURL) throw new Error('no baseURL');
  await page.addInitScript(() => {
    localStorage.setItem('ajayeb:name', 'Hints');
    const settings = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}');
    localStorage.setItem(
      'ajayeb:settings',
      JSON.stringify({ ...settings, hotseatPassDevice: false })
    );
  });
  const room = await createDevRoom(baseURL, { mode: 'hotseat', seed: '7', scenario: 'mausoleum' });
  await page.goto(room.path);
  const driver = new GameDriver(page);
  await driver.waitReady();
  const view = await currentView(page);
  const seat = view.toMove === 1 ? 1 : 0;
  const choice = view.choices.find((c) => c.kind === 'construct' && c.outcome?.score[seat]);
  if (!choice?.card) throw new Error('mausoleum scenario has no build with a score change');
  await driver.openCard(choice.card);
  const line = page.locator(`dialog[open] [data-outcome="${choice.key}"]`);
  await expect(line.locator('[data-outcome-term="vp"]')).toContainText('total VP');
  await expect(line.locator('.sr-only')).toContainText('total VP (coins included)');
});

let recorded: Promise<WinPosition> | null = null;

function position(origin: string): Promise<WinPosition> {
  recorded ??= (async () => {
    for (const seed of SEEDS) {
      const keys: string[] = [];
      try {
        await recordGame(origin, seed, (view) => {
          const choice = racePolicy(view);
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
    throw new Error(`no seed of ${SEEDS.join(', ')} reached a winning build`);
  })();
  return recorded;
}

test.describe('decider hints', { tag: ['@mobile'] }, () => {
  test('a winning build shows its crown and outcome line, and a discard warns', async ({
    page,
    baseURL
  }) => {
    test.setTimeout(150_000);
    if (!baseURL) throw new Error('no baseURL');
    const problems = watchHygiene(page);
    const at = await position(baseURL);
    // As in hint-pips.spec.ts: a stored name, no pass-device screen, no move
    // summary popup, and here Confirm moves on, so the confirm step shows.
    await page.addInitScript(() => {
      try {
        localStorage.setItem(
          'ajayeb:settings',
          JSON.stringify({
            hotseatPassDevice: false,
            animationSpeed: 'off',
            showOpponentMove: false,
            confirmMoves: 'always'
          })
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

    await page.setViewportSize({ width: 740, height: 360 });

    // The board pip says the build wins.
    const card = page.locator(`.card-structure [data-card="${at.card}"]`);
    await expect(card.locator('[data-hint]')).toHaveAttribute('data-hint', 'win');

    // The sheet: the crown and the win lead the outcome line under Build.
    await driver.dismissToasts();
    await card.click();
    const sheet = page.locator('dialog[open] [data-card-sheet]');
    await expect(sheet).toBeVisible();
    const build = sheet.locator(`[data-outcome="construct:${at.card}"]`);
    const winTerm = build.locator('[data-outcome-term]').first();
    await expect(winTerm).toHaveAttribute('data-outcome-term', 'win');
    await expect(winTerm.locator('svg')).toBeVisible();
    await expect(build.locator('.sr-only')).toContainText('Wins');

    // Discarding it instead: the warning shows only once Discard is armed.
    const discard = sheet.locator(`[data-outcome="discard:${at.card}"]`);
    const warning = discard.locator('[data-blunder]');
    await expect(warning).toBeHidden();
    await sheet.locator(`[data-act="discard:${at.card}"]`).click();
    await expect(sheet.locator(`[data-act="discard:${at.card}"]`)).toHaveAttribute(
      'data-armed',
      ''
    );
    await expect(warning).toBeVisible();
    await expect(warning).toContainText('could win this turn instead');
    // A warning can grow after arming. It must not push the action row
    // out of a short landscape sheet or consume its scrolling details.
    const geometry = await sheet.evaluate((el) => {
      const panel = el.closest('.sheet-panel') ?? el.querySelector('.sheet-panel') ?? el;
      const bounds = panel.getBoundingClientRect();
      return [
        ...el.querySelectorAll<HTMLElement>('button[data-act], [data-control="wonder-step"]')
      ].every((button) => {
        const r = button.getBoundingClientRect();
        const hit = document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2);
        return (
          r.height >= 44 &&
          r.top >= bounds.top &&
          r.bottom <= bounds.bottom &&
          !!hit &&
          button.contains(hit)
        );
      });
    });
    expect(geometry).toBe(true);
    expect(
      (await page.locator('dialog[open] [data-sheet-body]').boundingBox())!.height
    ).toBeGreaterThanOrEqual(44);
    // Nothing was sent: the seat is still to move.
    expect(await page.evaluate(() => window.__ajayeb?.view()?.toMove)).toBe(at.seat);

    expect(problems).toEqual([]);
  });
});
