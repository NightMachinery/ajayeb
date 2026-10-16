// The right-to-left smoke fixture (M10.persian-rtl): the app in Persian, a
// right-to-left locale whose table is a stub (the rest falls back to
// English per key). At Home, Settings, a hot-seat game at Age I and at its
// result screen:
//
// - <html> is lang="fa" dir="rtl";
// - no horizontal scroll: scrollWidth <= innerWidth, and nothing visible
//   past the left or right edge (fixtures/layout-probes.ts);
// - the app chrome is mirrored: a control measured in the same state in
//   English sits at the opposite inline edge;
// - the board stays left to right: the card structure and the military
//   track compute direction ltr, and a card's slot is where it is in English;
// - one full hot-seat move works, and nothing logs a console error.
//
// The language comes from the base fixture's appLocale option, which sets
// the app's own setting before the page loads. The English runs open the
// same states in a second browser context with the default language.

import { expect, type Browser, type Page, type TestInfo } from '@playwright/test';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { probeLayout, settle, type LayoutReport } from '../fixtures/layout-probes';
import { newDeviceContext } from '../fixtures/online';
import { GameDriver } from '../fixtures/driver';
import { createDevRoom, recordGame, type RecordedGame } from '../fixtures/ws-game';

/** Seeds tried in turn until one game is recorded to its end. */
const SEEDS = ['11', '12', '13', '14', '15'];

let recorded: Promise<RecordedGame> | null = null;

/** One recorded hot-seat game, played once per worker over the protocol. */
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

interface Rect {
  x: number;
  y: number;
  w: number;
  h: number;
}

/** The first visible match's box, rounded to a tenth of a pixel. */
async function rect(page: Page, selector: string): Promise<Rect> {
  const box = await page.locator(selector).filter({ visible: true }).first().boundingBox();
  if (!box) throw new Error(`no box for ${selector}`);
  const r = (n: number) => Math.round(n * 10) / 10;
  return { x: r(box.x), y: r(box.y), w: r(box.width), h: r(box.height) };
}

/** The centre of a box mirrored across the page's vertical axis. */
function mirroredCentre(box: Rect, width: number): number {
  return width - (box.x + box.w / 2);
}

/** The page's width without a classic scrollbar, the axis of the mirror. */
async function pageWidth(page: Page): Promise<number> {
  return page.evaluate(() => document.documentElement.clientWidth);
}

/** Two positions within 2 px of each other. */
function expectNear(actual: number, expected: number, what: string): void {
  expect(Math.abs(actual - expected), `${what}: ${actual} vs ${expected}`).toBeLessThanOrEqual(2);
}

async function documentDir(page: Page): Promise<{ dir: string; lang: string }> {
  return page.evaluate(() => ({
    dir: document.documentElement.dir,
    lang: document.documentElement.lang
  }));
}

async function computedDir(page: Page, selector: string): Promise<string> {
  const el = page.locator(selector).filter({ visible: true }).first();
  return el.evaluate((node) => getComputedStyle(node).direction);
}

function expectNoHorizontalScroll(report: LayoutReport, where: string): void {
  expect
    .soft(report.scrollWidth, `${where}: horizontal scroll`)
    .toBeLessThanOrEqual(report.innerWidth);
  expect.soft(report.overflowing, `${where}: elements past the left or right edge`).toEqual([]);
}

/** Room pages: a stored name (the room would ask for one) and no pass-device
 *  screen, merged into the settings the base fixture already wrote. */
async function prepareRooms(page: Page): Promise<void> {
  await page.addInitScript(() => {
    try {
      const raw = localStorage.getItem('ajayeb:settings');
      const current = raw ? (JSON.parse(raw) as Record<string, unknown>) : {};
      current['hotseatPassDevice'] = false;
      localStorage.setItem('ajayeb:settings', JSON.stringify(current));
      localStorage.setItem('ajayeb:name', 'Rtl');
    } catch {
      // Storage may be blocked; the defaults then apply.
    }
  });
}

/** A page in English (the app's default language), the same device as the project. */
async function englishPage(browser: Browser, testInfo: TestInfo): Promise<Page> {
  const context = await newDeviceContext(browser, testInfo);
  const page = await context.newPage();
  await prepareRooms(page);
  return page;
}

/** Opens a hot-seat room at the given prefix of the recorded game. */
async function openAt(page: Page, origin: string, rec: RecordedGame, plies: number): Promise<void> {
  const room = await createDevRoom(origin, {
    mode: 'hotseat',
    seed: rec.seed,
    actions: rec.keys.slice(0, plies)
  });
  await page.goto(room.path);
  await settle(page);
}

test.describe('right to left (Persian)', { tag: ['@mobile'] }, () => {
  test.use({ appLocale: 'fa' });

  test('Home is right to left and mirrored, with no horizontal scroll', async ({
    page,
    browser
  }, testInfo) => {
    const problems = watchHygiene(page);
    await page.goto('/');
    await expect(page.locator('main nav button').first()).toBeVisible();
    expect(await documentDir(page)).toEqual({ dir: 'rtl', lang: 'fa' });
    expectNoHorizontalScroll(await probeLayout(page), 'Home');

    // The Settings button and a mode button's icon sit at the opposite edge.
    const width = await pageWidth(page);
    const faGear = await rect(page, 'main button[aria-label="تنظیمات"]');
    const faIcon = await rect(page, 'main nav button svg');

    const en = await englishPage(browser, testInfo);
    try {
      await en.goto('/');
      await expect(en.locator('main nav button').first()).toBeVisible();
      expect(await documentDir(en)).toEqual({ dir: 'ltr', lang: 'en' });
      const enGear = await rect(en, 'main button[aria-label="Settings"]');
      const enIcon = await rect(en, 'main nav button svg');
      expect(await pageWidth(en)).toBe(width);
      expectNear(faGear.x + faGear.w / 2, mirroredCentre(enGear, width), 'the Settings button');
      expectNear(faIcon.x + faIcon.w / 2, mirroredCentre(enIcon, width), 'a mode icon');
    } finally {
      await en.context().close();
    }
    expect(problems, 'console errors, page errors or failed requests').toEqual([]);
  });

  test('Settings switches the language at once, and it holds across a reload', async ({ page }) => {
    const problems = watchHygiene(page);
    // When <html> first turns right to left: while the document is still
    // being parsed means src/app.html's inline bootstrap did it, before the
    // app's own scripts and the first paint.
    await page.addInitScript(() => {
      const w = window as unknown as { __firstRtl?: string };
      new MutationObserver(() => {
        if (w.__firstRtl === undefined && document.documentElement?.dir === 'rtl') {
          w.__firstRtl = document.readyState;
        }
      }).observe(document, {
        subtree: true,
        childList: true,
        attributes: true,
        attributeFilter: ['dir']
      });
    });
    const language = (value: string) =>
      page.locator(`label:has(> input[name="settings-language"][value="${value}"])`);

    await page.goto('/settings');
    await expect(language('fa')).toBeVisible();
    expect(await documentDir(page)).toEqual({ dir: 'rtl', lang: 'fa' });
    expect(await page.evaluate(() => (window as { __firstRtl?: string }).__firstRtl)).toBe(
      'loading'
    );
    await expect(page.locator('h1')).toHaveText('تنظیمات');
    expectNoHorizontalScroll(await probeLayout(page), 'Settings');

    await language('en').click();
    expect(await documentDir(page)).toEqual({ dir: 'ltr', lang: 'en' });
    await expect(page.locator('h1')).toHaveText('Settings');

    await language('fa').click();
    expect(await documentDir(page)).toEqual({ dir: 'rtl', lang: 'fa' });
    await expect(page.locator('h1')).toHaveText('تنظیمات');

    await page.reload();
    await expect(language('fa')).toBeVisible();
    expect(await documentDir(page)).toEqual({ dir: 'rtl', lang: 'fa' });
    const stored = await page.evaluate(() => localStorage.getItem('ajayeb:settings') ?? '{}');
    expect((JSON.parse(stored) as { locale?: string }).locale).toBe('fa');
    expect(problems, 'console errors, page errors or failed requests').toEqual([]);
  });

  test('a hot-seat game: mirrored chrome and a left-to-right board at Age I, one move, and the result', async ({
    page,
    browser,
    baseURL
  }, testInfo) => {
    test.setTimeout(120_000);
    if (!baseURL) throw new Error('no baseURL');
    const rec = await game(baseURL);
    const problems = watchHygiene(page);
    await prepareRooms(page);

    // Age I in English: where the menu button and a takeable card are.
    const en = await englishPage(browser, testInfo);
    let enMenu: Rect;
    let enCard: Rect;
    let enWidth: number;
    let cardId: string;
    try {
      await openAt(en, baseURL, rec, rec.marks.age1);
      expect(await documentDir(en)).toEqual({ dir: 'ltr', lang: 'en' });
      const card = en.locator('.card-structure [data-card][data-accessible]').first();
      cardId = (await card.getAttribute('data-card')) ?? '';
      expect(cardId).not.toBe('');
      enWidth = await pageWidth(en);
      enMenu = await rect(en, '[data-control="menu"]');
      enCard = await rect(en, `.card-structure [data-card="${cardId}"]`);
    } finally {
      await en.context().close();
    }

    await test.step('Age I', async () => {
      await openAt(page, baseURL, rec, rec.marks.age1);
      expect(await page.evaluate(() => window.__ajayeb?.view()?.age)).toBe(1);
      expect(await documentDir(page)).toEqual({ dir: 'rtl', lang: 'fa' });
      expectNoHorizontalScroll(await probeLayout(page), 'Age I');

      // The chrome is mirrored: the menu button is at the other edge.
      expect(await pageWidth(page)).toBe(enWidth);
      const menu = await rect(page, '[data-control="menu"]');
      expectNear(menu.x + menu.w / 2, mirroredCentre(enMenu, enWidth), 'the menu button');
      expect(await computedDir(page, '[data-turn]')).toBe('rtl');

      // The board is not: same direction, same slot.
      expect(await computedDir(page, '.card-structure')).toBe('ltr');
      expect(await computedDir(page, '.military-track')).toBe('ltr');
      const card = await rect(page, `.card-structure [data-card="${cardId}"]`);
      expectNear(card.x, enCard.x, 'the card slot x');
      expectNear(card.y, enCard.y, 'the card slot y');
      expectNear(card.w, enCard.w, 'the card width');
    });

    await test.step('one move', async () => {
      const driver = new GameDriver(page);
      const before = await driver.nextTurn();
      expect(before.phase).toBe('play');
      await driver.play(before);
      const after = await driver.snapshot();
      expect(after.ply).toBe(before.ply + 1);
      expect(await documentDir(page)).toEqual({ dir: 'rtl', lang: 'fa' });
      expectNoHorizontalScroll(await probeLayout(page), 'after one move');
    });

    await test.step('the result screen', async () => {
      await openAt(page, baseURL, rec, rec.marks.gameOver);
      expect(await page.evaluate(() => window.__ajayeb?.view()?.phase)).toBe('gameOver');
      await expect(page.locator('[data-game-over]').filter({ visible: true })).toHaveCount(1);
      expect(await documentDir(page)).toEqual({ dir: 'rtl', lang: 'fa' });
      expect(await computedDir(page, '[data-game-over]')).toBe('rtl');
      expectNoHorizontalScroll(await probeLayout(page), 'result');
    });

    expect(problems, 'console errors, page errors or failed requests').toEqual([]);
  });
});
