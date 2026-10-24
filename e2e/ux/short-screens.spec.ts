// Short screens (M18.short-screen-status): a phone whose browser shares the
// screen with another app, such as a video call, at 390x480 to 390x620 and
// 360x520, and at each game's floor height, below which the game shows a
// notice instead. At each size, in both themes and once in Persian, a live
// Duel game, Duel's age summary and a live Haftgane table must keep each
// player's status, the page's own science symbols, the cards to take and
// the turn bar fully on screen and uncovered, an open dialog's primary
// action inside the viewport, and the page free of horizontal scroll.
// Below the floor the notice is on screen instead.
// Each state is saved as a screenshot for the contact sheet, in
// SHORT_SHOTS_DIR (default .cache/e2e/short-screens/).
import { expect, type Page } from '@playwright/test';
import { mkdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { test } from '../fixtures/base';
import { openBotTable } from '../fixtures/table';
import { reachScreen } from './screens';

/** The smallest portrait height each game supports (`.too-short` in GameScreen and TableScreen). */
const DUEL_FLOOR = 460;
const TABLE_FLOOR = 480;
type Size = readonly [width: number, height: number];
const SIZES: readonly Size[] = [
  [390, 480],
  [390, 560],
  [390, 620],
  [360, 520]
];
/** The brief's sizes and the game's floor, once each. */
const sizes = (floor: number): Size[] =>
  SIZES.some(([, h]) => h === floor) ? [...SIZES] : [...SIZES, [390, floor]];
const dir = path.resolve(process.env.SHORT_SHOTS_DIR ?? '../.cache/e2e/short-screens');

interface Probe {
  overflow: boolean;
  /** Required parts that are missing, cut off by the viewport or a clipping
   *  ancestor, or covered by another element. */
  cut: string[];
  rects: Record<string, [number, number, number, number]>;
  /** Where the screen's main parts are, for the record (not checked). */
  layout: Record<string, [number, number, number, number]>;
}

const LAYOUT = [
  '.area-top',
  '.area-away',
  '.area-board',
  '.area-structure',
  '.area-home',
  '.area-turn',
  '.card-structure [data-card]',
  '[data-hand] [data-card]',
  '.player-strip',
  '[data-table-screen] [data-hand]',
  '[data-table-screen] [data-neighbour]',
  '[data-table-screen] [data-turn-bar]',
  'dialog:modal',
  'dialog:modal [data-sheet-footer]'
];

/** Each required selector (label, CSS) must match at least one shown element, fully visible. */
async function probe(page: Page, required: readonly (readonly [string, string])[]): Promise<Probe> {
  return page.evaluate(
    ([required, layoutSelectors]) => {
      const shown = (el: Element) =>
        el.getClientRects().length > 0 &&
        el.checkVisibility({ opacityProperty: true, visibilityProperty: true });
      const cutOff = (el: Element): boolean => {
        const r = el.getBoundingClientRect();
        let { left, right, top, bottom } = r;
        left = Math.max(left, 0);
        top = Math.max(top, 0);
        right = Math.min(right, innerWidth);
        bottom = Math.min(bottom, innerHeight);
        for (let a = el.parentElement; a; a = a.parentElement) {
          const style = getComputedStyle(a);
          if (style.overflowX !== 'visible' || style.overflowY !== 'visible') {
            const c = a.getBoundingClientRect();
            left = Math.max(left, c.left);
            right = Math.min(right, c.right);
            top = Math.max(top, c.top);
            bottom = Math.min(bottom, c.bottom);
          }
        }
        return right - left < r.width - 1 || bottom - top < r.height - 1;
      };
      // Four points near the corners, far enough in to clear a rounded
      // corner, must hit the part itself (or an ancestor, for a part that
      // lets the pointer through). Returns what covers it, for the report.
      const covered = (el: Element): string | null => {
        const r = el.getBoundingClientRect();
        const d = Math.min(r.width, r.height) / 4;
        for (const x of [r.left + d, r.right - d]) {
          for (const y of [r.top + d, r.bottom - d]) {
            const hit = document.elementFromPoint(x, y);
            if (!hit) return 'nothing';
            if (!(el.contains(hit) || hit.contains(el)))
              return `${hit.tagName.toLowerCase()}.${[...hit.classList].slice(0, 3).join('.')}`;
          }
        }
        return null;
      };
      const cut: string[] = [];
      const rects: Record<string, [number, number, number, number]> = {};
      for (const [label, selector] of required) {
        const els = [...document.querySelectorAll(selector)].filter(shown);
        if (els.length === 0) {
          cut.push(`${label}: missing`);
          continue;
        }
        for (const el of els) {
          const r = el.getBoundingClientRect();
          rects[label] = [
            Math.round(r.x),
            Math.round(r.y),
            Math.round(r.width),
            Math.round(r.height)
          ];
          if (cutOff(el)) cut.push(`${label}: cut off`);
          else {
            const by = covered(el);
            if (by) cut.push(`${label}: covered by ${by}`);
          }
        }
      }
      const layout: Record<string, [number, number, number, number]> = {};
      for (const selector of layoutSelectors) {
        const el = [...document.querySelectorAll(selector)].find(shown);
        if (!el) continue;
        const r = el.getBoundingClientRect();
        layout[selector] = [
          Math.round(r.x),
          Math.round(r.y),
          Math.round(r.width),
          Math.round(r.height)
        ];
      }
      return {
        layout,
        overflow:
          document.documentElement.scrollWidth > innerWidth + 1 ||
          document.body.scrollWidth > innerWidth + 1,
        cut,
        rects
      };
    },
    [required, LAYOUT] as const
  );
}

/** The parts every short screen of a game must keep on screen. */
const DUEL: readonly (readonly [string, string])[] = [
  ['status 0', '[data-player-status="0"]'],
  ['status 1', '[data-player-status="1"]'],
  ['science', '[data-player-status] [data-science-symbols]'],
  ['cards', '.area-structure'],
  ['turn', '.area-turn [data-turn], .area-turn [data-peek-bar]']
];
const TABLE: readonly (readonly [string, string])[] = [
  ['own status', '[data-table-screen] [data-player-status]'],
  ['science', '[data-table-screen] [data-player-status] [data-science-symbols]'],
  ['neighbours', '[data-table-screen] [data-neighbour]'],
  ['hand', '[data-table-screen] [data-hand]'],
  ['turn', '[data-table-screen] [data-turn-bar]']
];
/** An open modal dialog's pinned actions: Sheet's footer, the row a footer-less
 *  sheet marks data-sheet-actions, or a choice panel's commit button. */
const DIALOG: readonly (readonly [string, string])[] = [
  [
    'dialog action',
    [
      'dialog:modal [data-sheet-footer] button',
      'dialog:modal button[data-sheet-actions]',
      'dialog:modal [data-sheet-actions] button',
      'dialog:modal [data-commit] button'
    ].join(', ')
  ]
];
const NOTICE: readonly (readonly [string, string])[] = [['notice', '[data-too-short]']];

async function check(
  page: Page,
  name: string,
  width: number,
  height: number,
  required: readonly (readonly [string, string])[],
  floor = DUEL_FLOOR
): Promise<void> {
  await page.setViewportSize({ width, height });
  // Two frames for the layout, the structure's fit and the media queries.
  await page.evaluate(
    () => new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)))
  );
  mkdirSync(dir, { recursive: true });
  const file = `${name}-${width}x${height}`;
  await page.screenshot({ path: path.join(dir, `${file}.png`) });
  const below = height < floor;
  const p = await probe(page, below ? NOTICE : required);
  writeFileSync(path.join(dir, `${file}.json`), JSON.stringify(p, null, 1));
  expect.soft(p.overflow, `${file}: horizontal scroll`).toBe(false);
  expect.soft(p.cut, `${file}: off screen`).toEqual([]);
}

for (const theme of ['light', 'dark'] as const) {
  for (const locale of ['en', 'fa'] as const) {
    if (locale === 'fa' && theme === 'dark') continue;
    test.describe(`short screens ${theme} ${locale}`, () => {
      test.use({
        viewport: { width: 390, height: 844 },
        isMobile: true,
        hasTouch: true,
        deviceScaleFactor: 1,
        appLocale: locale
      });
      test.beforeEach(async ({ page }) => {
        await page.addInitScript((theme) => {
          localStorage.setItem('ajayeb:theme', theme);
          localStorage.setItem('ajayeb:name', 'Player');
        }, theme);
      });
      const tag = `${theme}-${locale}`;

      test('Duel live game', async ({ page, baseURL }) => {
        await reachScreen(page, baseURL!, 'board-age3');
        for (const [w, h] of sizes(DUEL_FLOOR)) await check(page, `duel-${tag}`, w, h, DUEL);
        await check(page, `duel-${tag}`, 390, DUEL_FLOOR - 1, DUEL);
      });

      test('Duel age summary', async ({ page, baseURL }) => {
        // The owner's case: the summary asks this player who starts the next age.
        await reachScreen(page, baseURL!, 'choice-starter');
        for (const [w, h] of sizes(DUEL_FLOOR)) await check(page, `age-${tag}`, w, h, DIALOG);
        // Peeked at the board, the summary's bar must still offer it back
        // above the turn bar.
        await page.setViewportSize({ width: 390, height: 844 });
        await page.locator('dialog:modal [data-age-continue]').click();
        await expect(page.locator('[data-peek-bar] [data-peek-open]')).toBeVisible();
        for (const [w, h] of sizes(DUEL_FLOOR)) await check(page, `age-peek-${tag}`, w, h, DUEL);
      });

      test('Duel science symbols', async ({ page, baseURL }) => {
        // The progress token choice comes from a pair of symbols; peeked at
        // the board, the strip shows them, each with its own tooltip.
        await reachScreen(page, baseURL!, 'choice-progress-token');
        await page.locator('dialog:modal [data-peek]').click();
        const paired = page.locator('[data-player-status] [data-science-symbols] [data-paired]');
        await expect(paired).toBeVisible();
        for (const [w, h] of sizes(DUEL_FLOOR)) await check(page, `science-${tag}`, w, h, DUEL);
        await page.setViewportSize({ width: 390, height: 480 });
        await paired.first().tap();
        await expect(page.locator('[data-tooltip-layer] [data-tooltip-bubble]')).toBeVisible();
        await page.screenshot({ path: path.join(dir, `science-tip-${tag}-390x480.png`) });
      });

      test('Duel age summary to begin', async ({ page, baseURL }) => {
        await reachScreen(page, baseURL!, 'end-age-summary');
        for (const [w, h] of sizes(DUEL_FLOOR)) await check(page, `begin-${tag}`, w, h, DIALOG);
      });

      // SHORT_SWEEP=1 records the layout every 20 px from 360 to 640 px
      // high, unchecked, to measure where the floor belongs.
      test('floor sweep', async ({ page, baseURL }) => {
        test.skip(!process.env.SHORT_SWEEP || tag !== 'light-en');
        test.setTimeout(180_000);
        const sweep = async (name: string, required: typeof DUEL) => {
          for (let h = 360; h <= 640; h += 20) {
            await page.setViewportSize({ width: 390, height: h });
            await page.evaluate(
              () => new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)))
            );
            mkdirSync(path.join(dir, 'sweep'), { recursive: true });
            const file = path.join(dir, 'sweep', `${name}-390x${h}`);
            await page.screenshot({ path: `${file}.png` });
            writeFileSync(`${file}.json`, JSON.stringify(await probe(page, required), null, 1));
          }
        };
        // A narrow window wider than it is high gets the landscape layout.
        const wide = async (name: string, required: typeof DUEL) => {
          for (const [w, h] of [
            [390, 380],
            [480, 400],
            [512, 400],
            [540, 320],
            [568, 320],
            [600, 400],
            [640, 320],
            [640, 360],
            [667, 320],
            [667, 340],
            [667, 350],
            [667, 360],
            [667, 375],
            [740, 600]
          ] satisfies Size[]) {
            await page.setViewportSize({ width: w, height: h });
            await page.evaluate(
              () => new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)))
            );
            const file = path.join(dir, 'sweep', `${name}-${w}x${h}`);
            await page.screenshot({ path: `${file}.png` });
            writeFileSync(`${file}.json`, JSON.stringify(await probe(page, required), null, 1));
          }
        };
        await reachScreen(page, baseURL!, 'board-age3');
        await sweep('duel', DUEL);
        await wide('duel', DUEL);
        await page.setViewportSize({ width: 390, height: 844 });
        await openBotTable(page, 7);
        await sweep('table7', TABLE);
        await wide('table7', TABLE);
      });

      test('Haftgane table', async ({ page }) => {
        test.setTimeout(120_000);
        await openBotTable(page, 5);
        for (const [w, h] of sizes(TABLE_FLOOR))
          await check(page, `table-${tag}`, w, h, TABLE, TABLE_FLOOR);
        await check(page, `table-${tag}`, 390, TABLE_FLOOR - 1, TABLE, TABLE_FLOOR);
        // A hand card's sheet keeps its action in view.
        await page.setViewportSize({ width: 390, height: 844 });
        await page.locator('[data-hand] [data-card]').first().click();
        await expect(page.locator('dialog:modal [data-sheet-footer]')).toBeVisible();
        for (const [w, h] of sizes(TABLE_FLOOR))
          await check(page, `table-card-${tag}`, w, h, DIALOG, TABLE_FLOOR);
      });
    });
  }
}

// Every kind of sheet, once: its primary action stays in view (Sheet pins
// a footer, or the row its caller marks data-sheet-actions).
test.describe('short screens dialogs', () => {
  test.use({ viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true });
  for (const screen of [
    'new-game',
    'join',
    'card-buildable',
    'choice-progress-token',
    'choice-destroy',
    'ending-science'
  ] as const) {
    test(screen, async ({ page, baseURL }) => {
      await reachScreen(page, baseURL!, screen);
      for (const [w, h] of sizes(DUEL_FLOOR)) await check(page, `sheet-${screen}`, w, h, DIALOG);
    });
  }
});
