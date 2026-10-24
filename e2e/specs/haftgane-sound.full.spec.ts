// Haftgane's sound over the real server, from Home's lobby
// (M18.sound-and-motion). CI has no audio device, so the spec reads what
// would have played from the page's fx test hook, window.__ajayebFx
// (web/src/lib/fx/index.ts), as Duel's sound specs do; the cue mapper
// itself is unit-tested (web/src/lib/games/haftgane/table/cues.test.ts).
// This checks the wiring. The creator opens a three-seat table with bots,
// and a friend watches it. The creator plays through the end of Age I,
// and for every batch:
//
// - each page records the batch's cues once, tagged with the batch, at
//   most three, 90 ms apart, none twice;
// - the spectator hears exactly what the table shares, read from the
//   batch's events: one flip for the reveal, the drum of the Age's
//   conflicts and the next Age's deal;
// - the creator hears the same, plus their own play (a slide for a
//   building, the wonder for a stage, a coin for a sale) while the budget
//   leaves room for it, and nothing that is not the viewer's own;
// - the creator's confirmation, when it reaches the table before the
//   batch resolves, is one tick.
//
// Then a reload plays nothing (its frame restates the game) and the next
// batch sounds again; the spectator turns Sound off in the table menu, and
// the next batch records nothing for them while the creator still hears it,
// until they turn it back on; and with Sound off in Settings the creator's
// table never loads the player. A new Age's deal sweeps the pass cues.
//
// A second table seats two people with bots, who decide at once. Each
// person in turn confirms first: they see the check and hear one tick, and
// the other, now the only one the table waits on, hears their turn once.
// Nothing depends on a particular deal.
import { readFileSync } from 'node:fs';
import { expect, type BrowserContext, type Page } from '@playwright/test';
import type { TableEvent } from '../../web/src/lib/wire/platform.gen';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { newDeviceContext } from '../fixtures/online';
import { createTableFromHome } from '../fixtures/lobby';
import { choose, latest, openBotTable, openMenu, ready } from '../fixtures/table';

const manifest = JSON.parse(
  readFileSync(
    new URL('../../web/.svelte-kit/output/client/.vite/manifest.json', import.meta.url),
    'utf8'
  )
) as Record<string, { file: string }>;
/** The lazy chunk that brings the app's player to the table. */
const PLAYER = '/' + manifest['src/lib/games/haftgane/table/sound-player.ts']!.file;

const SEATS = 3;
const SPACING_MS = 90;
const BUDGET = 3;
/** How long a page has to record a cue that must not come. */
const QUIET_MS = 1000;

interface Recorded {
  cue: string;
  delayMs: number;
  version: number;
  replay: boolean;
}

async function recorded(page: Page): Promise<Recorded[]> {
  return page.evaluate(
    () => (window as unknown as { __ajayebFx?: { cues(): Recorded[] } }).__ajayebFx?.cues() ?? []
  );
}

async function clearRecorded(page: Page): Promise<void> {
  await page.evaluate(() =>
    (window as unknown as { __ajayebFx?: { clear(): void } }).__ajayebFx?.clear()
  );
}

/** Whether the player has loaded and installed its hook in this page. */
async function armed(page: Page): Promise<boolean> {
  return page.evaluate(() => '__ajayebFx' in window);
}

/** The resolved batch's events, as the page holds them in its history. */
async function batchEvents(page: Page, batch: number): Promise<TableEvent[]> {
  const entry = await page.evaluate(
    (b) => window.__ajayebTable!.history().find((h) => (h as { batch: number }).batch === b),
    batch
  );
  expect(entry, `batch ${batch} is in the history`).toBeDefined();
  return (entry as { events: TableEvent[] }).events;
}

/** The cues every viewer of a batch hears, in their playing order. */
function sharedCues(events: readonly TableEvent[]): string[] {
  const has = (kind: string) => events.some((e) => e.kind === kind);
  const out: string[] = [];
  if (has('play') || has('side-chosen')) out.push('flip');
  if (has('discard-build')) out.push('rebuild');
  else if (has('discard-build-nothing')) out.push('tick');
  if (has('military')) out.push('drum');
  if (has('age-start')) out.push('ageStart');
  return out;
}

/** What the viewer's own play in a batch sounds like. */
const OWN_PLAY: Record<string, string> = { build: 'slide', stage: 'wonder', sell: 'coin' };
/** The cues a seated viewer may add to the shared ones. */
const OWN = new Set(['extraTurn', 'yourTurn', 'wonder', 'coin', 'slide', 'tick']);

/** Waits for a page's cues of `batch`, then checks the frame's shape. */
async function batchCues(page: Page, batch: number): Promise<string[]> {
  await expect
    .poll(async () => (await recorded(page)).filter((c) => c.version === batch).length)
    .toBeGreaterThan(0);
  const mine = (await recorded(page)).filter((c) => c.version === batch);
  expect(mine.length).toBeLessThanOrEqual(BUDGET);
  expect(mine.map((c) => c.delayMs)).toEqual(mine.map((_, i) => i * SPACING_MS));
  expect(mine.every((c) => !c.replay)).toBe(true);
  return mine.map((c) => c.cue);
}

/** The choice kinds the creator tries in turn, so stages and sales come up too. */
const ROTATION = ['build', 'stage', 'sell'] as const;

/**
 * The creator decides when the point asks them (or waits for the bots) and
 * the batch resolves; returns the new batch.
 */
async function playBatch(a: Page, me: number, turn: number): Promise<number> {
  const f = await latest(a);
  expect(f.status).toBe('playing');
  if (f.point?.required.includes(me) && !f.point.confirmed.includes(me)) {
    const cs = f.view.you!.choices;
    const kind = ROTATION[turn % ROTATION.length];
    const pick = cs.find((c) => c.kind === kind) ?? cs.find((c) => c.kind === 'build') ?? cs[0]!;
    await choose(a, pick.key);
    await choose(a, 'confirm');
  }
  await expect.poll(async () => (await latest(a)).batch).toBeGreaterThan(f.batch);
  await ready(a);
  return (await latest(a)).batch;
}

/** Waits until the spectator shows `batch` and takes input. */
async function caughtUp(b: Page, batch: number): Promise<void> {
  await expect.poll(async () => (await latest(b)).batch).toBe(batch);
  await ready(b);
}

/** Checks both pages' cues of a batch against its events; returns the creator's. */
async function checkBatch(a: Page, b: Page, me: number, batch: number): Promise<string[]> {
  const events = await batchEvents(a, batch);
  const shared = sharedCues(events);
  // Before this batch, the creator's own confirmation is one tick at most.
  const before = (await recorded(a)).filter((c) => c.version < batch).map((c) => c.cue);
  expect(before.length <= 1 && before.every((c) => c === 'tick'), before.join()).toBe(true);

  const creator = await batchCues(a, batch);
  expect(new Set(creator).size, creator.join()).toBe(creator.length);
  expect(creator.filter((c) => c === 'flip')).toHaveLength(shared.includes('flip') ? 1 : 0);
  for (const c of shared) expect(creator, `${batch}: ${creator.join()}`).toContain(c);
  for (const c of creator) expect(shared.includes(c) || OWN.has(c), c).toBe(true);
  const own = events.find((e) => e.kind === 'play' && e.seat === me);
  const notices = creator.filter((c) => c === 'extraTurn' || c === 'yourTurn').length;
  if (own && OWN_PLAY[own.play] && shared.length + notices < BUDGET)
    expect(creator, `${batch}: own ${own.play}`).toContain(OWN_PLAY[own.play]);

  await caughtUp(b, batch);
  if (shared.length > 0) expect(await batchCues(b, batch)).toEqual(shared);
  await b.waitForTimeout(QUIET_MS);
  expect((await recorded(b)).map((c) => c.cue)).toEqual(shared);
  await clearRecorded(a);
  await clearRecorded(b);
  return creator;
}

/** Turns Sound on or off from the table menu and closes it. */
async function menuSound(page: Page, on: boolean): Promise<void> {
  await openMenu(page);
  const sound = page.locator('[data-table-menu] [data-act="sound"]');
  await expect(sound).toHaveAttribute('data-sound', on ? 'off' : 'on');
  await sound.click();
  await expect(sound).toHaveAttribute('data-sound', on ? 'on' : 'off');
  await page.locator('[data-act="menu-close"]').click();
  await expect(page.locator('[data-table-menu]')).toHaveCount(0);
}

async function named(context: BrowserContext, name: string): Promise<Page> {
  await context.addInitScript((n) => localStorage.setItem('ajayeb:name', n), name);
  return context.newPage();
}

test.describe('Haftgane sound', { tag: ['@full', '@mobile', '@haftgane-sound'] }, () => {
  test('cues once per batch for a player and a spectator, none on reload or with Sound off', async ({
    browser
  }, info) => {
    test.setTimeout(420_000);
    const contexts: BrowserContext[] = [];
    try {
      for (let i = 0; i < 2; i++) contexts.push(await newDeviceContext(browser, info));
      const a = await named(contexts[0]!, 'Ana');
      const b = await named(contexts[1]!, 'Bo');
      const hygiene = [a, b].map(watchHygiene);
      const requested: string[] = [];
      a.on('request', (r) => requested.push(new URL(r.url()).pathname));

      // Sound is on by default: the table loads the player, and the
      // opening frames are silent.
      const code = await openBotTable(a, SEATS, 'Ana');
      const me = (await latest(a)).you.seat!;
      await b.goto(`/play/${code}`);
      await expect(b.locator('[data-spectator-panel]')).toBeVisible();
      await ready(b);
      for (const page of [a, b]) await expect.poll(() => armed(page)).toBe(true);
      expect(requested).toContain(PLAYER);
      expect(await recorded(a)).toEqual([]);
      expect(await recorded(b)).toEqual([]);

      // Through the end of Age I.
      let turn = 0;
      let ageEnd = false;
      let ownPlays = 0;
      for (; !ageEnd; turn++) {
        expect(turn, 'the end of Age I came in time').toBeLessThan(14);
        const batch = await playBatch(a, me, turn);
        const creator = await checkBatch(a, b, me, batch);
        const events = await batchEvents(a, batch);
        ageEnd = events.some((e) => e.kind === 'military');
        // A new Age's deal sweeps the pass chip and the pass flow once.
        const dealt = events.some((e) => e.kind === 'age-start') ? 1 : 0;
        await expect(a.locator('[data-top-bar] [data-pass][data-fresh]')).toHaveCount(dealt);
        await expect(a.locator('[data-pass-flow][data-fresh]')).toHaveCount(dealt);
        if (creator.some((c) => ['slide', 'wonder', 'coin'].includes(c))) ownPlays++;
      }
      expect(ownPlays, 'the creator heard their own plays').toBeGreaterThan(0);

      // A reload restates the game: nothing plays, and the next batch does.
      for (const page of [a, b]) {
        await page.reload();
        await expect(page.locator('[data-table-screen]')).toBeVisible();
        await ready(page);
        await expect.poll(() => armed(page)).toBe(true);
        await page.waitForTimeout(QUIET_MS);
        expect(await recorded(page), 'a reload plays nothing').toEqual([]);
      }
      await checkBatch(a, b, me, await playBatch(a, me, turn++));

      // The spectator mutes the table: the creator still hears the next
      // batch, the spectator records nothing, also after a reload, until
      // Sound is back on.
      await menuSound(b, false);
      let batch = await playBatch(a, me, turn++);
      await batchCues(a, batch);
      await caughtUp(b, batch);
      await b.waitForTimeout(QUIET_MS);
      expect(await recorded(b), 'Sound off plays nothing').toEqual([]);
      await b.reload();
      await ready(b);
      await openMenu(b);
      await expect(b.locator('[data-table-menu] [data-act="sound"]')).toHaveAttribute(
        'data-sound',
        'off'
      );
      await b.locator('[data-act="menu-close"]').click();
      await menuSound(b, true);
      await clearRecorded(a);
      await checkBatch(a, b, me, await playBatch(a, me, turn++));

      // Settings' Sound switch governs the table too: with it off, the
      // creator's table never loads the player and records nothing.
      await a.goto('/settings');
      await a.locator('input[data-setting="sound"]').uncheck();
      requested.length = 0;
      await a.goto(`/play/${code}`);
      await expect(a.locator('[data-table-screen]')).toBeVisible();
      batch = await playBatch(a, me, turn++);
      await caughtUp(b, batch);
      expect(await batchCues(b, batch)).toEqual(sharedCues(await batchEvents(b, batch)));
      await a.waitForTimeout(QUIET_MS);
      expect(await armed(a), 'no player loads with Sound off').toBe(false);
      expect(await recorded(a)).toEqual([]);
      expect(requested).not.toContain(PLAYER);

      for (const problems of hygiene) expect(problems).toEqual([]);
    } finally {
      for (const ctx of contexts) await ctx.close();
    }
  });

  test('two players: a confirmation ticks, and the last person to choose hears their turn', async ({
    browser
  }, info) => {
    test.setTimeout(240_000);
    const contexts: BrowserContext[] = [];
    try {
      for (let i = 0; i < 2; i++) contexts.push(await newDeviceContext(browser, info));
      const a = await named(contexts[0]!, 'Ana');
      const b = await named(contexts[1]!, 'Bo');
      const hygiene = [a, b].map(watchHygiene);
      const code = await createTableFromHome(a, 'Ana');
      await b.goto(`/play/${code}`);
      await expect
        .poll(() =>
          a.evaluate(
            () => window.__ajayebTable?.lobby()?.seats.filter((s) => s.kind === 'human').length
          )
        )
        .toBe(2);
      await choose(a, 'lobby-start');
      for (const page of [a, b]) {
        await expect(page.locator('[data-table-screen]')).toBeVisible();
        await ready(page);
        await expect.poll(() => armed(page)).toBe(true);
      }

      for (const [first, last] of [
        [a, b],
        [b, a]
      ] as const) {
        // The bots decide at once; once they have, the two people decide last.
        const f = await latest(a);
        const bots = f.seats.filter((s) => s.kind === 'bot').map((s) => s.seat);
        await expect
          .poll(async () => {
            const p = (await latest(a)).point;
            return bots.every((s) => p?.confirmed.includes(s));
          })
          .toBe(true);
        await clearRecorded(a);
        await clearRecorded(b);
        await choose(first, (await latest(first)).view.you!.choices[0]!.key);
        await choose(first, 'confirm');
        await expect(first.locator('[data-pending-confirm="true"] .check')).toBeVisible();
        await expect.poll(async () => (await recorded(first)).map((c) => c.cue)).toEqual(['tick']);
        await expect
          .poll(async () => (await recorded(last)).map((c) => c.cue))
          .toEqual(['yourTurn']);
        await choose(last, (await latest(last)).view.you!.choices[0]!.key);
        await choose(last, 'confirm');
        await expect.poll(async () => (await latest(a)).batch).toBeGreaterThan(f.batch);
        const batch = (await latest(a)).batch;
        for (const page of [a, b]) {
          await caughtUp(page, batch);
          expect((await batchCues(page, batch)).filter((c) => c === 'flip')).toHaveLength(1);
        }
        // The notices of the decision came once each.
        expect((await recorded(first)).filter((c) => c.cue === 'tick')).toHaveLength(1);
        expect((await recorded(last)).filter((c) => c.cue === 'yourTurn')).toHaveLength(1);
      }
      for (const problems of hygiene) expect(problems).toEqual([]);
    } finally {
      for (const ctx of contexts) await ctx.close();
    }
  });
});
