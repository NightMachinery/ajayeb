// The move and event animations (M10.animations-moves and
// M10.animations-events, web/src/lib/fx/anim): the four acceptance checks of
// the UX design's section 12, "M4: animation and game feel", read with the
// game's own Animation setting (D21), and the log's Replay last move.
//
//   a. With Animation off nothing animates for more than 10 ms, and every
//      change is on screen the moment the frame lands.
//   b. Animations are purely visual: the scripted sequences (moves, a
//      looting, a science pair's token, the Great Library, a destroy, a
//      rebuild, an age change, a draft pick and a game end) played once at
//      normal speed and once with Animation off leave the same game screen.
//   c. Input is back within 450 ms of a frame's swap at normal speed (the
//      bot's dwell and pick, before the swap, are deliberate pacing),
//      including an age deal and the first move after it.
//   d. @perf: no long task over 50 ms overlaps an animation on a CPU slowed
//      4 times, over at least ten moves, an age deal and a game end.
//   e. Replay last move plays the move again and leaves the same screen; with
//      Animation off it is disabled and its tooltip says why.
//   f. @perf, information only: the gaps between animation frames while
//      animations run, at full CPU speed (case d reports them at 4x).
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
  animationFrameGaps,
  describeFrameGaps,
  draftKeys,
  findPosition,
  fxRecords,
  fxCues,
  frameTimes,
  inputAt,
  longTasks,
  motionLog,
  openPosition,
  playThrough,
  recordFrames,
  recordInputBack,
  recordLongTasks,
  recordMotion,
  screenDom,
  traceIfAsked,
  useSettings,
  viewOf,
  waitSettled,
  type AnimRecord,
  type LongTask,
  type Position,
  type Speed
} from '../fixtures/anim';
import type { GameDriver } from '../fixtures/driver';

const SEED = '11';

/** One scripted sequence: where it starts, and the moves it plays there. */
interface Sequence {
  name: string;
  at: Position;
  /** The keys to play, one after another, each read from the view on screen. */
  picks: ((view: View) => string)[];
  /** A vs-AI room: after the move, the bot answers before the check. */
  bot?: boolean;
}

/** A sequence at a named scenario that plays its trigger, key by key. */
function scenario(name: string, scenario: string, keys: string[]): Sequence {
  return {
    name,
    at: { mode: 'hotseat', seed: SEED, scenario, keys: [] },
    picks: keys.map((key) => () => key)
  };
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

/** The sequences of check b (also played by a), found once per worker. */
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
      { name: 'a build that trades and reveals a card', at: trade.at, picks: [() => trade.key] },
      {
        name: 'a discard',
        at: { mode: 'hotseat', seed: SEED, keys: draft },
        picks: [(view) => firstOf(view, 'discard')]
      },
      scenario('a wonder that moves coins', 'appian-economy', ['wonder:appian-way:rostrum']),
      scenario('a military build that loots twice', 'looting-double', ['construct:arsenal']),
      {
        name: 'a bot reply',
        at: { mode: 'ai', seed: SEED, keys: draft },
        picks: [(view) => firstOf(view, 'construct')],
        bot: true
      },
      {
        name: 'a science pair and its token',
        at: { mode: 'hotseat', seed: SEED, scenario: 'science-pair', keys: [] },
        picks: [() => 'construct:library', (view) => firstOf(view, 'token')]
      },
      scenario('the Great Library', 'great-library', ['wonder:great-library:rostrum', 'token:law']),
      scenario('a destroy', 'zeus-destroy', ['wonder:statue-of-zeus:rostrum', 'destroy:quarry']),
      scenario('a rebuild from the discard pile', 'mausoleum', [
        'wonder:mausoleum:rostrum',
        'revive:baths'
      ]),
      scenario('an age change and its starter', 'age-transition-chooser', [
        'discard:theater',
        'starter:0'
      ]),
      {
        name: 'a draft pick',
        at: { mode: 'hotseat', seed: SEED, keys: [] },
        picks: [(view) => firstOf(view, 'draft'), (view) => firstOf(view, 'draft')]
      },
      scenario('a game end', 'civilian-end', ['discard:academy'])
    ];
  })();
  return sequences;
}

/** Plays a sequence in a fresh room, and waits until everything has settled. */
async function playSequence(page: Page, origin: string, seq: Sequence): Promise<GameDriver> {
  const driver = await openPosition(page, origin, seq.at);
  await driver.waitTurn();
  await driver.dismissToasts();
  for (const pick of seq.picks) {
    await driver.dismissToasts();
    await playThrough(driver, pick(await viewOf(page)));
  }
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
      drawn:
        (document.querySelector('.fx-layer')?.childElementCount ?? 0) +
        (document.querySelector('.fx-top')?.childElementCount ?? 0)
    };
  });
  expect(shown.hidden, `${what}: nothing hidden`).toBe(0);
  expect(shown.drawn, `${what}: nothing in the FX layer`).toBe(0);
  // The draft shows no coins.
  if (view.phase !== 'wonderDraft') {
    expect(shown.coins, `${what}: the coins`).toEqual(
      view.players.map((p) => new Intl.NumberFormat('en').format(p.coins))
    );
  }
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

/** What one room of a performance run recorded. */
interface PerfRoom {
  name: string;
  /** performance.now() at the room's perf-room mark. */
  markAt: number;
  anims: AnimRecord[];
  tasks: LongTask[];
  frames: number[];
}

/**
 * The performance run of cases d and f, with the CPU slowed `rate` times:
 * ten hot-seat moves, each after the last one's animations ended, then an
 * age change with the move after it, then a game's last card. Each room is
 * its own page load, so its records are read before the next.
 */
async function perfRun(
  page: Page,
  baseURL: string,
  rate: number
): Promise<{ moves: number; rooms: PerfRoom[] }> {
  await recordLongTasks(page);
  await recordFrames(page);
  await useSettings(page, 'normal');
  const draft = await draftKeys(baseURL, SEED);
  const rooms: PerfRoom[] = [];
  const collect = async (name: string) => {
    // A mark that ties this page's clock to a trace's (traceIfAsked).
    const markAt = await page.evaluate((room) => {
      performance.mark(`perf-room:${room}`);
      return performance.now();
    }, name);
    rooms.push({
      name,
      markAt,
      anims: (await fxRecords(page)).anims,
      tasks: await longTasks(page),
      frames: await frameTimes(page)
    });
  };
  // A deliberate player: the next tap comes after the move has played.
  const play = async (driver: GameDriver, key: string) => {
    await driver.dismissToasts();
    await playThrough(driver, key);
    await waitSettled(page, 20_000);
  };
  const cdp = await page.context().newCDPSession(page);
  await cdp.send('Emulation.setCPUThrottlingRate', { rate });
  const stopTrace = await traceIfAsked(cdp, test.info());
  let moves = 0;
  try {
    const driver = await openPosition(page, baseURL, { mode: 'hotseat', seed: SEED, keys: draft });
    while (moves < 10) {
      const snap = await driver.nextTurn();
      if (snap.phase !== 'play') break;
      const view = await viewOf(page);
      await play(driver, snap.pending ? (view.choices[0]?.key ?? '') : recordPolicy(view).key);
      moves += 1;
    }
    await collect('Age I');

    // An age deal, and the first move after it.
    const age = await openPosition(page, baseURL, {
      mode: 'hotseat',
      seed: SEED,
      scenario: 'age-transition-chooser',
      keys: []
    });
    for (const key of ['discard:theater', 'starter:0']) {
      await age.nextTurn();
      await play(age, key);
    }
    await age.nextTurn();
    await play(age, recordPolicy(await viewOf(page)).key);
    await collect('age change');

    // A game end, with its victory banner.
    const end = await openPosition(page, baseURL, {
      mode: 'hotseat',
      seed: SEED,
      scenario: 'civilian-end',
      keys: []
    });
    await end.nextTurn();
    await play(end, 'discard:academy');
    await collect('game end');
  } finally {
    await stopTrace();
    await cdp.send('Emulation.setCPUThrottlingRate', { rate: 1 });
  }
  if (process.env.AJAYEB_PERF_TRACE === '1') {
    await test.info().attach('perf rooms', {
      body: JSON.stringify({ rate, rooms }),
      contentType: 'application/json'
    });
  }
  return { moves, rooms };
}

/** That a performance run played what it should have. */
function expectPerfRun(moves: number, anims: AnimRecord[]): void {
  expect(moves).toBeGreaterThanOrEqual(10);
  expect(anims.length).toBeGreaterThan(10);
  expect(
    anims.some((a) => a.kind === 'deal'),
    'an age deal played'
  ).toBe(true);
  expect(
    anims.some((a) => a.kind === 'victory'),
    'the victory banner played'
  ).toBe(true);
}

test.describe('animations', { tag: ['@mobile'] }, () => {
  test('a. with Animation off nothing animates and every change shows at once', async ({
    page,
    baseURL
  }) => {
    test.setTimeout(420_000);
    if (!baseURL) throw new Error('no baseURL');
    const problems = watchHygiene(page);
    await recordMotion(page);
    await useSettings(page, 'off');
    for (const seq of await scripted(baseURL)) {
      const driver = await openPosition(page, baseURL, seq.at);
      await driver.waitTurn();
      await driver.dismissToasts();
      await motionLog(page, true);
      const before = await driver.snapshot();
      for (const [n, pick] of seq.picks.entries()) {
        await driver.dismissToasts();
        await playThrough(driver, pick(await viewOf(page)));
        // The moment the frame lands: no waiting for anything to settle.
        await expectStateShown(page, `${seq.name}, move ${n + 1}`);
      }
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
    test.setTimeout(540_000);
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
      test.setTimeout(300_000);
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

      // An age change: the age's last card, the starter's answer (the deal
      // plays then, as the dialog closes), and the first move of the age,
      // which waits only for its own card to be dealt (a hidden card cannot
      // be clicked).
      const age = await openPosition(page, baseURL, {
        mode: 'hotseat',
        seed: SEED,
        scenario: 'age-transition-chooser',
        keys: []
      });
      for (const pick of [
        () => 'discard:theater',
        () => 'starter:0',
        (view: View) => recordPolicy(view).key
      ]) {
        await age.nextTurn();
        await age.dismissToasts();
        await playThrough(age, pick(await viewOf(page)));
        await measure('age');
      }
      const dealt = (await fxRecords(page)).anims.filter((a) => a.kind === 'deal');
      expect(dealt.length, 'the age change dealt its cards').toBeGreaterThan(0);

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
      expect(waits.filter((w) => w.room === 'age').length).toBeGreaterThanOrEqual(3);
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
        test.setTimeout(420_000);
        if (!baseURL) throw new Error('no baseURL');
        const problems = watchHygiene(page);
        const { moves, rooms } = await perfRun(page, baseURL, 4);
        const anims = rooms.flatMap((r) => r.anims);
        const tasks = rooms.flatMap((r) => r.tasks);
        const overlapping = rooms.flatMap((r) =>
          r.tasks.filter((task) =>
            r.anims.some(
              (a) =>
                a.endAt !== null && task.start < a.endAt && task.start + task.duration > a.startAt
            )
          )
        );
        const worst = Math.max(0, ...overlapping.map((t) => Math.round(t.duration)));
        test.info().annotations.push({
          type: 'perf',
          description:
            `moves ${moves} (and an age change, a game end), animations ${anims.length}, ` +
            `long tasks ${tasks.length}, overlapping ${overlapping.length}, ` +
            `worst overlapping ${worst} ms, ` +
            `worst overall ${Math.max(0, ...tasks.map((t) => Math.round(t.duration)))} ms; ` +
            `at 4x: ${describeFrameGaps(rooms.flatMap((r) => animationFrameGaps(r.frames, r.anims)))}`
        });
        expectPerfRun(moves, anims);
        expect(overlapping, `long tasks during animations (worst ${worst} ms)`).toEqual([]);
        expect(problems, 'console errors, page errors or failed requests').toEqual([]);
      }
    );
  });

  // f. Information only (no threshold): stutter at full speed is not the same
  // as long tasks at 4x, so the same run at 1x reports its frame gaps.
  test(
    'f. frame gaps during animations at 1x CPU',
    { tag: ['@perf'] },
    async ({ page, baseURL }) => {
      test.setTimeout(300_000);
      if (!baseURL) throw new Error('no baseURL');
      const problems = watchHygiene(page);
      const { moves, rooms } = await perfRun(page, baseURL, 1);
      test.info().annotations.push({
        type: 'perf',
        description: `at 1x: ${describeFrameGaps(rooms.flatMap((r) => animationFrameGaps(r.frames, r.anims)))}`
      });
      expectPerfRun(
        moves,
        rooms.flatMap((r) => r.anims)
      );
      expect(problems, 'console errors, page errors or failed requests').toEqual([]);
    }
  );

  test('e. Replay last move plays the move again and changes nothing', async ({
    page,
    baseURL
  }) => {
    test.setTimeout(180_000);
    if (!baseURL) throw new Error('no baseURL');
    const problems = watchHygiene(page);
    const log = page.locator('dialog[open]').filter({ has: page.locator('#log-sheet-title') });
    const control = log.locator('[data-control="replay-last-move"]');
    const openLog = async (driver: GameDriver) => {
      await driver.dismissToasts();
      await driver.visible('[data-control="log"]').click();
      await expect(log).toBeVisible();
    };
    const at: Position = { mode: 'hotseat', seed: SEED, scenario: 'looting-double', keys: [] };

    // With Animation off: disabled, and its tooltip says why.
    await useSettings(page, 'off');
    let driver = await openPosition(page, baseURL, at);
    await driver.waitTurn();
    await playThrough(driver, 'construct:arsenal');
    await openLog(driver);
    await expect(control).toBeDisabled();
    const group = log.locator('[data-replay-tooltip]');
    if (test.info().project.use.hasTouch) {
      await group.tap();
    } else {
      // Keyboard focus (focus-visible) shows it: a Tab first, so the focus
      // that follows counts as the keyboard's.
      await page.keyboard.press('Tab');
      await group.focus();
    }
    await expect(page.locator('[data-tooltip-layer] [role="tooltip"]')).toContainText(
      'Turn Animation on'
    );
    await page.keyboard.press('Escape');

    // At normal speed: the move's animations play again, the screen stays.
    await useSettings(page, 'normal');
    await page.evaluate(() => {
      const current = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}') as Record<
        string,
        unknown
      >;
      localStorage.setItem('ajayeb:settings', JSON.stringify({ ...current, sound: true }));
    });
    driver = await openPosition(page, baseURL, at);
    await driver.waitTurn();
    await openLog(driver);
    await expect(control, 'no move to replay yet').toBeDisabled();
    await page.keyboard.press('Escape');
    await expect(log).toHaveCount(0);
    await playThrough(driver, 'construct:arsenal');
    await waitSettled(page);
    const before = await screenDom(page);
    const version = (await driver.snapshot()).version;
    const played = (await fxRecords(page)).anims.length;
    const cueCount = (await fxCues(page)).length;
    await openLog(driver);
    await expect(control).toBeEnabled();
    await control.click();
    await expect(log).toHaveCount(0);
    await expect
      .poll(async () => (await fxRecords(page)).anims.length, { message: 'the replay animated' })
      .toBeGreaterThan(played);
    const kinds = new Set((await fxRecords(page)).anims.slice(played).map((a) => a.kind));
    const replayCues = (await fxCues(page)).slice(cueCount);
    expect(replayCues.length).toBeGreaterThan(0);
    expect(replayCues.every((cue) => cue.replay && cue.version === version)).toBe(true);
    expect(replayCues.map((cue) => cue.cue)).toContain('drum');
    expect([...kinds], 'the replay played the looting').toContain('loot');
    await waitSettled(page);
    expect((await driver.snapshot()).version, 'the replay sent nothing').toBe(version);
    expect(await screenDom(page), 'the screen after the replay').toBe(before);
    expect(problems, 'console errors, page errors or failed requests').toEqual([]);
  });
});
