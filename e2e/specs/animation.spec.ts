// The move animations (M10.animations-moves, web/src/lib/fx/anim): the four
// acceptance checks of the UX design's section 12, "M4: animation and game
// feel", read with the game's own Animation setting (D21).
//
//   a. With Animation off nothing animates for more than 10 ms, and every
//      change is on screen the moment the frame lands.
//   b. Animations are purely visual: five scripted sequences played once at
//      normal speed and once with Animation off leave the same game screen.
//   c. Input is back within 450 ms of a frame's swap at normal speed (the
//      bot's dwell and pick, before the swap, are deliberate pacing).
//   d. @perf: no long task over 50 ms overlaps an animation on a CPU slowed
//      4 times, over at least ten moves.
//
// Every case turns the setting on for its own tab (fixtures/anim.ts
// useSettings, after the base fixture's motion-off override) and opens its
// own rooms from a fixed seed or a named scenario through the dev endpoint.

import { expect, type Page } from '@playwright/test';
import type { View } from '../../web/src/lib/wire';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { recordPolicy } from '../fixtures/ws-game';
import {
  draftKeys,
  findPosition,
  fxRecords,
  inputAt,
  longTasks,
  motionLog,
  openPosition,
  playThrough,
  recordInputBack,
  recordLongTasks,
  recordMotion,
  screenDom,
  useSettings,
  viewOf,
  waitSettled,
  type Position,
  type Speed
} from '../fixtures/anim';
import type { GameDriver } from '../fixtures/driver';

const SEED = '11';

/** One scripted sequence: where it starts, and the move it plays there. */
interface Sequence {
  name: string;
  at: Position;
  /** The key to play, read from the view on screen. */
  pick: (view: View) => string;
  /** A vs-AI room: after the move, the bot answers before the check. */
  bot?: boolean;
}

/** Whether taking `card` turns a face-down card up. */
function reveals(view: View, card: string | undefined): boolean {
  const slot = view.structure.find((s) => s.card === card);
  if (!slot) return false;
  return view.structure.some(
    (s) => s.state === 'faceDown' && s.coveredBy.length === 1 && s.coveredBy[0] === slot.index
  );
}

function firstOf(view: View, kind: string): string {
  const c = view.choices.find((x) => x.kind === kind);
  if (!c) throw new Error(`no ${kind} choice: ${view.choices.map((x) => x.key)}`);
  return c.key;
}

let sequences: Promise<Sequence[]> | null = null;

/** The five sequences of check b (also played by a), found once per worker. */
function scripted(origin: string): Promise<Sequence[]> {
  sequences ??= (async () => {
    const draft = await draftKeys(origin, SEED);
    const trade = await findPosition(
      origin,
      SEED,
      (view) =>
        view.phase === 'play' && !view.pending
          ? (view.choices.find(
              (c) =>
                c.kind === 'construct' &&
                !!c.cost?.affordable &&
                (c.cost?.tradeTotal ?? 0) > 0 &&
                reveals(view, c.card)
            )?.key ?? null)
          : null,
      recordPolicy
    );
    return [
      { name: 'a build that trades and reveals a card', at: trade.at, pick: () => trade.key },
      {
        name: 'a discard',
        at: { mode: 'hotseat', seed: SEED, keys: draft },
        pick: (view) => firstOf(view, 'discard')
      },
      {
        name: 'a wonder that moves coins',
        at: { mode: 'hotseat', seed: SEED, scenario: 'appian-economy', keys: [] },
        pick: () => 'wonder:appian-way:rostrum'
      },
      {
        name: 'a military build that moves the pawn',
        at: { mode: 'hotseat', seed: SEED, scenario: 'looting-double', keys: [] },
        pick: () => 'construct:arsenal'
      },
      {
        name: 'a bot reply',
        at: { mode: 'ai', seed: SEED, keys: draft },
        pick: (view) => firstOf(view, 'construct'),
        bot: true
      }
    ];
  })();
  return sequences;
}

/** Plays a sequence in a fresh room, and waits until everything has settled. */
async function playSequence(page: Page, origin: string, seq: Sequence): Promise<GameDriver> {
  const driver = await openPosition(page, origin, seq.at);
  await driver.waitTurn();
  await driver.dismissToasts();
  await playThrough(driver, seq.pick(await viewOf(page)));
  if (seq.bot) await driver.waitTurn();
  await waitSettled(page);
  return driver;
}

/** Checks that what the frame on screen says is what the page shows. */
async function expectStateShown(page: Page, what: string): Promise<void> {
  const view = await viewOf(page);
  const shown = await page.evaluate(() => {
    const coins = [0, 1].map(
      (seat) =>
        document.querySelector(
          `.game-screen .seat-panel[data-seat="${seat}"] [data-tally="coins"] [data-fx-count]`
        )?.textContent ?? null
    );
    const pawn = document.querySelector('.game-screen [data-fx-pawn]')?.getAttribute('data-pawn');
    const faces = [...document.querySelectorAll('.game-screen .card-structure [data-slot]')]
      .filter((el) => el.getAttribute('data-state') === 'faceUp')
      .map((el) => el.getAttribute('data-card'));
    return {
      coins,
      pawn: pawn === null || pawn === undefined ? null : Number(pawn),
      faces,
      hidden: document.querySelectorAll('[data-fx-hidden]').length,
      drawn: document.querySelector('.fx-layer')?.childElementCount ?? 0
    };
  });
  expect(shown.hidden, `${what}: nothing hidden`).toBe(0);
  expect(shown.drawn, `${what}: nothing in the FX layer`).toBe(0);
  expect(shown.coins, `${what}: the coins`).toEqual(
    view.players.map((p) => new Intl.NumberFormat('en').format(p.coins))
  );
  if (view.phase === 'play') {
    expect(shown.pawn, `${what}: the pawn`).toBe(view.military.position);
    expect(shown.faces.sort(), `${what}: the face-up cards`).toEqual(
      view.structure
        .filter((s) => s.state === 'faceUp')
        .map((s) => s.card ?? '')
        .sort()
    );
  }
}

test.describe('animations', { tag: ['@mobile'] }, () => {
  test('a. with Animation off nothing animates and every change shows at once', async ({
    page,
    baseURL
  }) => {
    test.setTimeout(240_000);
    if (!baseURL) throw new Error('no baseURL');
    const problems = watchHygiene(page);
    await recordMotion(page);
    await useSettings(page, 'off');
    for (const seq of await scripted(baseURL)) {
      const driver = await openPosition(page, baseURL, seq.at);
      await driver.waitTurn();
      await driver.dismissToasts();
      await motionLog(page, true);
      const key = seq.pick(await viewOf(page));
      const before = await driver.snapshot();
      await playThrough(driver, key);
      // The moment the frame lands: no waiting for anything to settle.
      await expectStateShown(page, seq.name);
      if (seq.bot) {
        await driver.waitTurn();
        await expectStateShown(page, `${seq.name}, the bot's answer`);
      }
      const { frames, anims } = await fxRecords(page);
      expect(anims, `${seq.name}: the runner started nothing`).toEqual([]);
      for (const f of frames.filter((x) => x.version > before.version)) {
        expect(f.steps, `${seq.name}: frame ${f.version} planned nothing`).toBe(0);
      }
      const long = (await motionLog(page)).filter((m) => m.ms > 10);
      expect(long, `${seq.name}: animations or transitions over 10 ms`).toEqual([]);
    }
    expect(problems, 'console errors, page errors or failed requests').toEqual([]);
  });

  test('b. animations are purely visual: the same screen with them on and off', async ({
    page,
    baseURL
  }) => {
    test.setTimeout(300_000);
    if (!baseURL) throw new Error('no baseURL');
    const problems = watchHygiene(page);
    for (const seq of await scripted(baseURL)) {
      const dom: Partial<Record<Speed, string>> = {};
      for (const speed of ['normal', 'off'] as const) {
        await useSettings(page, speed);
        await playSequence(page, baseURL, seq);
        if (speed === 'normal') {
          const { anims } = await fxRecords(page);
          expect(anims.length, `${seq.name}: something animated at normal speed`).toBeGreaterThan(
            0
          );
        }
        dom[speed] = await screenDom(page);
      }
      expect(dom.normal, `${seq.name}: the screen after the animations`).toBe(dom.off);
    }
    expect(problems, 'console errors, page errors or failed requests').toEqual([]);
  });

  test.describe(() => {
    // Both measure wall time, so a loaded machine can push one run over: up
    // to three runs, each one's numbers in the failure message or the
    // annotations (docs/testing.md says how to read them).
    test.describe.configure({ retries: 2 });

    test('c. input comes back within 450 ms of a frame at normal speed', async ({
      page,
      baseURL
    }) => {
      test.setTimeout(240_000);
      if (!baseURL) throw new Error('no baseURL');
      const problems = watchHygiene(page);
      await recordInputBack(page);
      await useSettings(page, 'normal');
      const draft = await draftKeys(baseURL, SEED);
      const waits: { room: string; version: number; ms: number; steps: number }[] = [];

      const measure = async (room: string) => {
        const { frames } = await fxRecords(page);
        const at = await inputAt(page);
        for (const f of frames) {
          const back = at[f.version];
          if (back === undefined || f.steps === 0) continue;
          if (waits.some((w) => w.room === room && w.version === f.version)) continue;
          waits.push({ room, version: f.version, ms: Math.round(back - f.swapAt), steps: f.steps });
        }
      };

      // Hot-seat: every move is the next player's, so every frame counts.
      const hot = await openPosition(page, baseURL, { mode: 'hotseat', seed: SEED, keys: draft });
      for (let i = 0; i < 6; i++) {
        const snap = await hot.nextTurn();
        if (snap.phase !== 'play' || snap.pending) break;
        const view = await viewOf(page);
        await hot.dismissToasts();
        await playThrough(hot, recordPolicy(view).key);
        await measure('hotseat');
      }

      // Against a bot: its answers are the frames that hand input back.
      const ai = await openPosition(page, baseURL, { mode: 'ai', seed: SEED, keys: draft });
      for (let i = 0; i < 3; i++) {
        const snap = await ai.nextTurn();
        if (snap.phase !== 'play' || snap.pending) break;
        await ai.dismissToasts();
        await playThrough(ai, recordPolicy(await viewOf(page)).key);
        await ai.waitTurn();
        await measure('ai');
      }

      test.info().annotations.push({
        type: 'input back (ms)',
        description: waits.map((w) => `${w.room} v${w.version}: ${w.ms}`).join(', ')
      });
      expect(waits.filter((w) => w.room === 'hotseat').length).toBeGreaterThanOrEqual(4);
      expect(waits.filter((w) => w.room === 'ai').length).toBeGreaterThanOrEqual(2);
      const all = waits.map((w) => `${w.room} v${w.version}: ${w.ms}`).join(', ');
      for (const w of waits) {
        expect(
          w.ms,
          `${w.room} frame ${w.version}: input back after ${w.ms} ms (${all})`
        ).toBeLessThanOrEqual(450);
      }
      expect(problems, 'console errors, page errors or failed requests').toEqual([]);
    });

    test(
      'd. no long task over 50 ms during an animation at 4x CPU',
      { tag: ['@perf'] },
      async ({ page, baseURL }) => {
        test.setTimeout(300_000);
        if (!baseURL) throw new Error('no baseURL');
        const problems = watchHygiene(page);
        await recordLongTasks(page);
        await useSettings(page, 'normal');
        const draft = await draftKeys(baseURL, SEED);
        const driver = await openPosition(page, baseURL, {
          mode: 'hotseat',
          seed: SEED,
          keys: draft
        });
        const cdp = await page.context().newCDPSession(page);
        await cdp.send('Emulation.setCPUThrottlingRate', { rate: 4 });
        let moves = 0;
        try {
          while (moves < 10) {
            const snap = await driver.nextTurn();
            if (snap.phase !== 'play') break;
            await driver.dismissToasts();
            const view = await viewOf(page);
            const key = snap.pending ? (view.choices[0]?.key ?? '') : recordPolicy(view).key;
            await playThrough(driver, key);
            // A deliberate player: the next tap comes after the move has played.
            await waitSettled(page, 20_000);
            moves += 1;
          }
        } finally {
          await cdp.send('Emulation.setCPUThrottlingRate', { rate: 1 });
        }
        const { anims } = await fxRecords(page);
        const tasks = await longTasks(page);
        const overlapping = tasks.filter((task) =>
          anims.some(
            (a) =>
              a.endAt !== null && task.start < a.endAt && task.start + task.duration > a.startAt
          )
        );
        const worst = Math.max(0, ...overlapping.map((t) => Math.round(t.duration)));
        test.info().annotations.push({
          type: 'perf',
          description:
            `moves ${moves}, animations ${anims.length}, long tasks ${tasks.length}, ` +
            `overlapping ${overlapping.length}, worst overlapping ${worst} ms, ` +
            `worst overall ${Math.max(0, ...tasks.map((t) => Math.round(t.duration)))} ms`
        });
        expect(moves).toBeGreaterThanOrEqual(10);
        expect(anims.length).toBeGreaterThan(10);
        expect(overlapping, `long tasks during animations (worst ${worst} ms)`).toEqual([]);
        expect(problems, 'console errors, page errors or failed requests').toEqual([]);
      }
    );
  });
});
