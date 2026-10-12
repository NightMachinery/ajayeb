// M5 acceptance: the game screen fits every viewport of the UX plan.
//
// One seeded hot-seat game is played once over the protocol (fixtures/
// ws-game.ts) to learn its action keys; each viewport then opens the same
// game at five positions through POST /api/dev/rooms with the seed and a
// prefix of those keys: the wonder draft, a fresh Age I, Age II and Age III
// structure, and the result screen. At each one:
//
// - no horizontal scroll: scrollWidth <= innerWidth, and no visible part of
//   any element past the left or right edge;
// - no page scroll: scrollHeight <= innerHeight;
// - every takeable card at least 44 x 44 px (the plan requires it at
//   360 x 740 in all three ages; it is checked at every viewport);
// - on a phone (portrait or landscape), neither player strip scrolls
//   inside: both fit their box whole;
// - every enabled primary control at least 44 px tall, on the screen and in
//   the card sheet of the first takeable card;
// - no console errors, page errors or failed requests.
//
// The desktop project checks all nine viewports with a mouse; the mobile
// project checks the phone and tablet ones with touch.

import { expect } from '@playwright/test';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { probeLayout, quiet, settle, type LayoutReport } from '../fixtures/layout-probes';
import { createDevRoom, recordGame, type GameMarks, type RecordedGame } from '../fixtures/ws-game';

const VIEWPORTS = [
  { width: 360, height: 740 },
  { width: 390, height: 844 },
  { width: 412, height: 915 },
  { width: 430, height: 932 },
  { width: 844, height: 390 },
  { width: 740, height: 360 },
  { width: 768, height: 1024 },
  { width: 1024, height: 768 },
  { width: 1280, height: 800 },
  { width: 1920, height: 1080 }
] as const;

/** The viewports the mobile project checks: phones and the tablet. */
const TOUCH_WIDTHS = new Set([360, 390, 412, 430, 844, 740, 768]);

const POSITIONS = [
  'draft',
  'age1',
  'age2',
  'age3',
  'gameOver'
] as const satisfies readonly (keyof GameMarks)[];

const MIN_TAP = 44;

/** Seeds tried in turn until one game reaches Age III under the recording policy. */
const SEEDS = ['11', '12', '13', '14', '15'];

let recorded: Promise<RecordedGame> | null = null;

/** The recorded game, played once per worker. */
function game(origin: string): Promise<RecordedGame> {
  recorded ??= (async () => {
    let last: unknown;
    for (const seed of SEEDS) {
      try {
        return await recordGame(origin, seed);
      } catch (err) {
        last = err;
      }
    }
    throw last;
  })();
  return recorded;
}

function summary(report: LayoutReport): string {
  return JSON.stringify({
    ...report,
    takeable: report.takeable.length,
    controls: report.controls.length
  });
}

/** One line per position for the run log: the layout, the smallest card and control. */
function measure(report: LayoutReport): string {
  const min = (boxes: { w: number; h: number }[], f: (b: { w: number; h: number }) => number) =>
    boxes.length > 0 ? Math.min(...boxes.map(f)) : null;
  return [
    `layout ${report.layout ?? '-'}`,
    `cards ${report.takeable.length} (min ${min(report.takeable, (b) => b.w)}x${min(report.takeable, (b) => b.h)})`,
    `controls ${report.controls.length} (min h ${min(report.controls, (b) => b.h)})`,
    `strips overflowing ${report.stripOverflow.length}`,
    `scroll ${report.scrollWidth}x${report.scrollHeight}`
  ].join(', ');
}

function check(report: LayoutReport, where: string): void {
  console.log(`${where}: ${measure(report)}`);
  expect
    .soft(report.scrollWidth, `${where}: horizontal scroll (${summary(report)})`)
    .toBeLessThanOrEqual(report.innerWidth);
  expect.soft(report.overflowing, `${where}: elements past the left or right edge`).toEqual([]);
  expect
    .soft(report.scrollHeight, `${where}: page scroll (${summary(report)})`)
    .toBeLessThanOrEqual(report.innerHeight);
  const smallCards = report.takeable.filter((b) => b.w < MIN_TAP || b.h < MIN_TAP);
  expect.soft(smallCards, `${where}: takeable cards under ${MIN_TAP} px`).toEqual([]);
  const smallControls = report.controls.filter((b) => b.h < MIN_TAP);
  expect.soft(smallControls, `${where}: primary controls under ${MIN_TAP} px tall`).toEqual([]);
  if (report.layout?.startsWith('phone')) {
    expect.soft(report.stripOverflow, `${where}: a player strip scrolls inside`).toEqual([]);
  }
}

for (const viewport of VIEWPORTS) {
  const name = `${viewport.width}x${viewport.height}`;
  test.describe(`layout at ${name}`, { tag: '@mobile' }, () => {
    test.use({ viewport });

    test('draft, Age I, II and III, and the result fit', async ({ page, baseURL, isMobile }) => {
      test.skip(isMobile && !TOUCH_WIDTHS.has(viewport.width), 'desktop-only viewport');
      test.setTimeout(120_000);
      if (!baseURL) throw new Error('no baseURL');
      const recording = await game(baseURL);
      const problems = watchHygiene(page);
      // The layout under test is the board itself, without the pass-device
      // screen between turns. The rooms come from this test's own HTTP
      // request, so the browser has not seen them: a stored name keeps the
      // room page from asking for one first. This replaces the whole
      // ajayeb:settings object (a page-level init script runs after the
      // base fixture's context-level one, fixtures/base.ts), so it repeats
      // animationSpeed: 'off' and showOpponentMove: false itself rather than
      // relying on that order.
      await page.addInitScript(() => {
        try {
          localStorage.setItem(
            'ajayeb:settings',
            JSON.stringify({
              hotseatPassDevice: false,
              animationSpeed: 'off',
              showOpponentMove: false
            })
          );
          localStorage.setItem('ajayeb:name', 'Layout');
        } catch {
          // Storage may be blocked; the default only matters on touch screens.
        }
      });

      for (const position of POSITIONS) {
        await test.step(position, async () => {
          const keys = recording.keys.slice(0, recording.marks[position]);
          const room = await createDevRoom(baseURL, {
            mode: 'hotseat',
            seed: recording.seed,
            actions: keys
          });
          await page.goto(room.path);
          await settle(page);
          const phase = await page.evaluate(() => window.__ajayeb?.view()?.phase);
          const age = await page.evaluate(() => window.__ajayeb?.view()?.age);
          if (position === 'draft') expect(phase).toBe('wonderDraft');
          else if (position === 'gameOver') expect(phase).toBe('gameOver');
          else expect([phase, age]).toEqual(['play', Number(position.slice(3))]);

          const report = await probeLayout(page);
          check(report, `${name} ${position}`);
          if (position.startsWith('age')) {
            expect(report.takeable.length, 'a fresh structure offers cards').toBeGreaterThan(0);
            // The card sheet of the first takeable card, and its buttons.
            await page.locator('.card-structure [data-card][data-accessible]').first().click();
            await expect(page.locator('dialog[open] [data-card-sheet]')).toBeVisible();
            await quiet(page);
            check(await probeLayout(page), `${name} ${position} card sheet`);
            await page.keyboard.press('Escape');
            await expect(page.locator('dialog[open]')).toHaveCount(0);
          }
        });
      }

      expect(problems, 'console errors, page errors or failed requests').toEqual([]);
    });
  });
}
