// Board-level hints (the hints design, M10.HS4), on the desktop project.
//
// Seeded hot-seat games are played over the WebSocket driver until each of
// four positions turns up, and the browser then opens the same game there:
//
//   dead       a seat holds no card of a science symbol and none can come:
//              its tracker slot is struck (data-symbol-state="dead")
//   hourglass  six wonders are built, so only one more can be: each
//              unbuilt wonder's row carries the hourglass
//   ghost      the seat to move has a takeable card whose build moves the
//              pawn: selecting it draws the ghost pawn on the board's track
//   wonder     the seat to move can build a wonder with a takeable card: the
//              card sheet's wonder step puts an outcome line under its button
//
// The policies decide nothing on their own: every key they return is
// copied from the state's choices, and they read only what the server sent.

import { expect, type Page } from '@playwright/test';
import type { Choice, View } from '../../web/src/lib/wire';
import { test } from '../fixtures/base';
import { GameDriver } from '../fixtures/driver';
import { watchHygiene } from '../fixtures/hygiene';
import { createDevRoom, recordGame, recordPolicy } from '../fixtures/ws-game';

/** Seeds tried in turn until one reaches the position. */
const SEEDS = ['51', '52', '53', '54', '55', '56', '57', '58'];

interface Position {
  seed: string;
  /** The keys that lead to the position. */
  keys: string[];
  /** What the position is about: a seat, a symbol, a card, a key. */
  seat: 0 | 1;
  symbol?: string;
  card?: string;
  key?: string;
}

type Found = Omit<Position, 'seed' | 'keys'>;

/** Thrown by the policy to stop the recording at the wanted position. */
class Stop extends Error {
  readonly found: Found;
  constructor(found: Found) {
    super('found');
    this.found = found;
  }
}

function playing(view: View): boolean {
  return view.phase === 'play' && !view.pending;
}

function takeable(view: View, card: string | undefined): boolean {
  return view.structure.some((s) => s.card === card && s.state === 'faceUp' && s.accessible);
}

const FIND: Record<string, (view: View) => Found | null> = {
  dead(view) {
    if (!playing(view)) return null;
    for (const seat of [0, 1] as const) {
      const dead = view.players[seat].hints?.symbols.find(
        (s) => s.symbol !== 'scales' && s.state === 'dead'
      );
      if (dead) return { seat, symbol: dead.symbol };
    }
    return null;
  },
  hourglass(view) {
    if (!playing(view) || view.hints?.wondersLeft !== 1) return null;
    for (const seat of [0, 1] as const) {
      const open = view.players[seat].wonders.find((w) => !w.built && !w.removed);
      if (open) return { seat, card: open.wonder };
    }
    return null;
  },
  ghost(view) {
    if (!playing(view)) return null;
    const seat = view.toMove === 1 ? 1 : 0;
    const c = view.choices.find(
      (x) =>
        x.kind === 'construct' &&
        takeable(view, x.card) &&
        x.outcome?.pawn !== undefined &&
        x.outcome.pawn.to !== x.outcome.pawn.from
    );
    return c?.card ? { seat, card: c.card, key: c.key } : null;
  },
  wonder(view) {
    if (!playing(view)) return null;
    const seat = view.toMove === 1 ? 1 : 0;
    const c = view.choices.find(
      (x) =>
        x.kind === 'wonder' &&
        takeable(view, x.card) &&
        x.outcome !== undefined &&
        (x.outcome.score[seat] !== 0 || x.outcome.tempo !== undefined)
    );
    return c?.card ? { seat, card: c.card, key: c.key } : null;
  }
};

/** Builds wonders whenever it can, so the race for the last one comes. */
function wonderPolicy(view: View): Choice {
  return view.choices.find((c) => c.kind === 'wonder') ?? recordPolicy(view);
}

const recorded = new Map<string, Promise<Position>>();

function position(origin: string, what: keyof typeof FIND): Promise<Position> {
  let p = recorded.get(what);
  if (!p) {
    p = (async () => {
      const find = FIND[what]!;
      const policy = what === 'hourglass' ? wonderPolicy : recordPolicy;
      for (const seed of SEEDS) {
        const keys: string[] = [];
        try {
          await recordGame(origin, seed, (view) => {
            const found = find(view);
            if (found) throw new Stop(found);
            const choice = policy(view);
            keys.push(choice.key);
            return choice;
          });
        } catch (err) {
          // The policy threw before its key was pushed, so `keys` is
          // exactly what was applied.
          if (err instanceof Stop) return { seed, keys, ...err.found };
        }
      }
      throw new Error(`no seed of ${SEEDS.join(', ')} reached a ${what} position`);
    })();
    recorded.set(what, p);
  }
  return p;
}

/** Opens the hot-seat game at a recorded position. */
async function open(page: Page, origin: string, at: Position): Promise<GameDriver> {
  await page.addInitScript(() => {
    try {
      localStorage.setItem(
        'ajayeb:settings',
        JSON.stringify({
          hotseatPassDevice: false,
          animationSpeed: 'off',
          showOpponentMove: false,
          confirmMoves: 'never'
        })
      );
      localStorage.setItem('ajayeb:name', 'Hints');
    } catch {
      // Storage may be blocked; the room then asks for a name and fails.
    }
  });
  const room = await createDevRoom(origin, { mode: 'hotseat', seed: at.seed, actions: at.keys });
  await page.goto(room.path);
  const driver = new GameDriver(page);
  await driver.waitReady();
  await driver.dismissToasts();
  return driver;
}

test.describe('board hints', () => {
  test('the science tracker strikes a symbol that can no longer come', async ({
    page,
    baseURL
  }) => {
    test.setTimeout(150_000);
    if (!baseURL) throw new Error('no baseURL');
    const problems = watchHygiene(page);
    const at = await position(baseURL, 'dead');
    await open(page, baseURL, at);
    const slot = page.locator(
      `.seat-panel[data-seat="${at.seat}"] .science-tracker [data-symbol="${at.symbol}"]`
    );
    await expect(slot).toHaveAttribute('data-symbol-state', 'dead');
    await expect(slot.locator('.dead-strike')).toBeVisible();
    expect(problems).toEqual([]);
  });

  test('once one wonder is left, every unbuilt wonder shows the hourglass', async ({
    page,
    baseURL
  }) => {
    test.setTimeout(150_000);
    if (!baseURL) throw new Error('no baseURL');
    const problems = watchHygiene(page);
    const at = await position(baseURL, 'hourglass');
    await open(page, baseURL, at);
    const row = page.locator(`.seat-panel[data-seat="${at.seat}"] [data-wonder-row="${at.card}"]`);
    await expect(row.locator('[data-hint="hourglass"]')).toBeVisible();
    expect(problems).toEqual([]);
  });

  test('selecting a card that moves the pawn draws the ghost pawn on the board', async ({
    page,
    baseURL
  }) => {
    test.setTimeout(150_000);
    if (!baseURL) throw new Error('no baseURL');
    const problems = watchHygiene(page);
    const at = await position(baseURL, 'ghost');
    const driver = await open(page, baseURL, at);
    await driver.openCard(at.card!);
    // The board's own track (not the card sheet's copy) draws the ghost.
    await expect
      .poll(() =>
        page.evaluate(() =>
          [...document.querySelectorAll('[data-ghost-pawn]')].some((e) => !e.closest('dialog'))
        )
      )
      .toBe(true);
    expect(problems).toEqual([]);
  });

  test('the wonder step puts an outcome line under each wonder', async ({ page, baseURL }) => {
    test.setTimeout(150_000);
    if (!baseURL) throw new Error('no baseURL');
    const problems = watchHygiene(page);
    const at = await position(baseURL, 'wonder');
    const driver = await open(page, baseURL, at);
    await driver.openCard(at.card!);
    await page.locator('dialog[open] [data-control="wonder-step"]').click();
    const step = page.locator('dialog[open] [data-wonder-step]');
    await expect(step).toBeVisible();
    const line = step.locator(`[data-act-group] [data-outcome="${at.key}"]`);
    await expect(line.locator('[data-outcome-term]').first()).toBeVisible();
    expect(problems).toEqual([]);
  });
});
