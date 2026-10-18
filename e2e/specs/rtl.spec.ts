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

import { expect, type Browser, type Locator, type Page, type TestInfo } from '@playwright/test';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { probeLayout, settle, type LayoutReport } from '../fixtures/layout-probes';
import { newDeviceContext } from '../fixtures/online';
import { GameDriver } from '../fixtures/driver';
import { playKey } from '../fixtures/scenario';
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

/** Check the browser's paragraph direction and visual order, not just dir attributes.
 * Latin letters in an interpolated name must sit to the right of the next
 * Persian word, while the name's own letters still read left to right. */
async function expectLatinNameFirstInRtl(line: Locator, name: string): Promise<void> {
  await expect(line).toBeVisible();
  const visual = await line.evaluate((node, name) => {
    const text = node.textContent ?? '';
    const start = text.indexOf(name);
    const next = text.slice(start + name.length).search(/[\u0620-\u06ff]/u);
    function characterBox(index: number): { x: number; y: number } {
      const walker = document.createTreeWalker(node, NodeFilter.SHOW_TEXT);
      let textNode: Node | null;
      while ((textNode = walker.nextNode())) {
        const length = textNode.textContent?.length ?? 0;
        if (index < length) {
          const range = document.createRange();
          range.setStart(textNode, index);
          range.setEnd(textNode, index + 1);
          const box = range.getBoundingClientRect();
          return { x: box.x, y: box.y };
        }
        index -= length;
      }
      throw new Error('The sentence has no character at the requested index');
    }
    if (start < 0 || next < 0) throw new Error('Expected a Latin name followed by Persian text');
    return {
      text: text.replace(/[\u200e\u200f\u2066-\u2069]/gu, '').trim(),
      direction: getComputedStyle(node).direction,
      first: characterBox(start),
      last: characterBox(start + name.length - 1),
      persian: characterBox(start + name.length + next)
    };
  }, name);
  expect(visual.text.startsWith(name), 'the translated sentence starts with the Latin name').toBe(
    true
  );
  expect(visual.direction).toBe('rtl');
  expect(
    Math.abs(visual.first.y - visual.persian.y),
    'name and first Persian word share a line'
  ).toBeLessThan(3);
  expect(
    visual.first.x,
    'the leading name is visually before the Persian words in RTL'
  ).toBeGreaterThan(visual.persian.x);
  expect(visual.first.x, 'the Latin name itself still reads LTR').toBeLessThan(visual.last.x);
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

  test('Settings switches text and direction together, and it holds across a reload', async ({
    page
  }) => {
    const problems = watchHygiene(page);
    // A persisted Persian choice stays English until its messages load.
    // The first RTL document must already contain the Persian heading.
    await page.addInitScript(() => {
      const w = window as unknown as { __firstRtl?: string };
      new MutationObserver(() => {
        if (w.__firstRtl === undefined && document.documentElement?.dir === 'rtl') {
          w.__firstRtl = document.querySelector('h1')?.textContent ?? '';
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
    await expect(page.locator('h1')).toHaveText('تنظیمات');
    expect(await documentDir(page)).toEqual({ dir: 'rtl', lang: 'fa' });
    expect(await page.evaluate(() => (window as { __firstRtl?: string }).__firstRtl)).toBe(
      'تنظیمات'
    );
    expectNoHorizontalScroll(await probeLayout(page), 'Settings');

    await language('en').click();
    expect(await documentDir(page)).toEqual({ dir: 'ltr', lang: 'en' });
    await expect(page.locator('h1')).toHaveText('Settings');

    await language('fa').click();
    expect(await documentDir(page)).toEqual({ dir: 'rtl', lang: 'fa' });
    await expect(page.locator('h1')).toHaveText('تنظیمات');

    await page.reload();
    await expect(page.locator('h1')).toHaveText('تنظیمات');
    await expect(language('fa')).toBeVisible();
    expect(await documentDir(page)).toEqual({ dir: 'rtl', lang: 'fa' });
    const stored = await page.evaluate(() => localStorage.getItem('ajayeb:settings') ?? '{}');
    expect((JSON.parse(stored) as { locale?: string }).locale).toBe('fa');
    expect(problems, 'console errors, page errors or failed requests').toEqual([]);
  });

  test('a Latin name leads a Persian pass-device sentence from the right', async ({
    page,
    baseURL
  }) => {
    if (!baseURL) throw new Error('no baseURL');
    const problems = watchHygiene(page);
    await page.addInitScript(() => {
      localStorage.setItem('ajayeb:name', 'Player');
      const settings = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}');
      localStorage.setItem(
        'ajayeb:settings',
        JSON.stringify({ ...settings, hotseatPassDevice: true })
      );
    });
    const room = await createDevRoom(baseURL, {
      mode: 'hotseat',
      seed: '7',
      scenario: 'mausoleum'
    });
    await page.goto(room.path);
    const driver = new GameDriver(page);
    await driver.waitReady();
    await playKey(driver, 'discard:rostrum');
    await expectLatinNameFirstInRtl(page.locator('[data-pass-device] li').first(), 'Player');
    expect(problems, 'console errors, page errors or failed requests').toEqual([]);
  });

  test('Latin names lead Persian starter options from the right', async ({ page }) => {
    const problems = watchHygiene(page);
    await page.addInitScript(() => {
      const settings = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}');
      localStorage.setItem(
        'ajayeb:settings',
        JSON.stringify({ ...settings, confirmMoves: 'never' })
      );
    });
    await page.goto('/dev/fixture/choose-starter-seat0');
    await expect(page.locator('html')).toHaveAttribute('data-ready', '1');
    const option = page.locator('[data-choice-host="chooseStarter"] [data-act="starter:1"] span');
    await expectLatinNameFirstInRtl(option, 'Omid');
    const rawName = page
      .locator('[data-age-interstitial] [dir="auto"]')
      .filter({ hasText: 'Sara' });
    await expect(rawName).toBeVisible();
    expect(await rawName.evaluate((node) => getComputedStyle(node).direction)).toBe('ltr');
    expect(problems, 'console errors, page errors or failed requests').toEqual([]);
  });

  test('lobby sharing labels stay on one line at the four review sizes', async ({ page }) => {
    const problems = watchHygiene(page);
    await page.addInitScript(() => {
      Object.defineProperty(navigator, 'share', { value: async () => {} });
    });
    for (const [width, height] of [
      [360, 740],
      [390, 844],
      [740, 360],
      [1280, 800]
    ] as const) {
      await page.setViewportSize({ width, height });
      await page.goto('/dev/fixture/online-lobby-seat0');
      await expect(page.locator('html')).toHaveAttribute('data-ready', '1');
      for (const control of ['copy-link', 'share-link']) {
        const button = page.locator(`[data-control="${control}"]`);
        await expect(button).toBeVisible();
        const layout = await button.evaluate((node) => {
          const text = [...node.childNodes].find(
            (child) => child.nodeType === Node.TEXT_NODE && child.textContent?.trim()
          );
          if (!text) throw new Error('No sharing button label');
          const value = text.textContent ?? '';
          const range = document.createRange();
          range.setStart(text, value.length - value.trimStart().length);
          range.setEnd(text, value.trimEnd().length);
          return {
            lines: range.getClientRects().length,
            height: node.getBoundingClientRect().height
          };
        });
        expect(layout.lines, `${control} at ${width}px`).toBe(1);
        expect(layout.height).toBeGreaterThanOrEqual(44);
      }
      expectNoHorizontalScroll(await probeLayout(page), `lobby at ${width}px`);
    }
    expect(problems, 'console errors, page errors or failed requests').toEqual([]);
  });

  test('translated labels in spatial diagram rows follow Persian direction', async ({ page }) => {
    const problems = watchHygiene(page);
    await page.goto('/rules');
    for (const name of ['chain', 'military', 'discard']) {
      const diagram = page.locator(`[data-rules-diagram="${name}"]`).first();
      await expect(diagram).toBeVisible();
      const spatial = diagram.locator('[dir="ltr"]').first();
      expect(await spatial.evaluate((node) => getComputedStyle(node).direction)).toBe('ltr');
      const labels = await diagram.locator('[dir="ltr"]').evaluateAll((rows) => {
        const seen = new Set<Node>();
        const labels: { text: string; direction: string }[] = [];
        for (const row of rows) {
          const walker = document.createTreeWalker(row, NodeFilter.SHOW_TEXT);
          let node: Node | null;
          while ((node = walker.nextNode())) {
            const text = node.textContent?.trim() ?? '';
            if (seen.has(node) || !/[\u0620-\u06d3]/u.test(text) || !node.parentElement) continue;
            seen.add(node);
            labels.push({ text, direction: getComputedStyle(node.parentElement).direction });
          }
        }
        return labels;
      });
      expect(
        labels.length,
        `${name} has translated labels inside its spatial rows`
      ).toBeGreaterThan(0);
      for (const label of labels) expect(label.direction, label.text).toBe('rtl');
    }
    expect(problems, 'console errors, page errors or failed requests').toEqual([]);
  });

  test('the translated victory caption follows Persian direction inside the spatial FX layer', async ({
    page,
    baseURL
  }) => {
    const problems = watchHygiene(page);
    if (!baseURL) throw new Error('no baseURL');
    await page.addInitScript(() => {
      localStorage.setItem('ajayeb:name', 'Player');
      const settings = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}');
      localStorage.setItem(
        'ajayeb:settings',
        JSON.stringify({ ...settings, animationSpeed: 'normal' })
      );
      // Record the real label while the ordinary animation owns it. Its
      // direction is the assertion; this test adds no timing requirement.
      new MutationObserver(() => {
        const label = document.querySelector('.fx-banner-text');
        const layer = label?.closest('.fx-top');
        if (label && layer) {
          (
            window as { __bidiVictory?: { text: string; label: string; layer: string } }
          ).__bidiVictory = {
            text: label.textContent ?? '',
            label: getComputedStyle(label).direction,
            layer: getComputedStyle(layer).direction
          };
        }
      }).observe(document, { childList: true, subtree: true });
    });
    const room = await createDevRoom(baseURL, {
      mode: 'hotseat',
      seed: '7',
      scenario: 'civilian-end'
    });
    await page.goto(room.path);
    const driver = new GameDriver(page);
    await driver.waitReady();
    await playKey(driver, 'discard:academy');
    const caption = await page.evaluate(
      () =>
        (window as { __bidiVictory?: { text: string; label: string; layer: string } }).__bidiVictory
    );
    expect(caption?.text).toMatch(/[\u0620-\u06ff]/u);
    expect(caption?.label).toBe('rtl');
    expect(caption?.layer).toBe('ltr');
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
    let enTokens: string[];
    try {
      await openAt(en, baseURL, rec, rec.marks.age1);
      expect(await documentDir(en)).toEqual({ dir: 'ltr', lang: 'en' });
      const card = en.locator('.card-structure [data-card][data-accessible]').first();
      cardId = (await card.getAttribute('data-card')) ?? '';
      expect(cardId).not.toBe('');
      enWidth = await pageWidth(en);
      enMenu = await rect(en, '[data-control="menu"]');
      enCard = await rect(en, `.card-structure [data-card="${cardId}"]`);
      enTokens = await en
        .locator('.progress-token-row')
        .filter({ visible: true })
        .first()
        .locator('[data-fx-token]')
        .evaluateAll((els) => els.map((el) => el.getAttribute('data-fx-token') ?? ''));
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
      const backLabel = page.locator('.card-structure .card-back text').first();
      await expect(backLabel).toBeVisible();
      expect(await backLabel.evaluate((node) => getComputedStyle(node).direction)).toBe('rtl');
      expect(await computedDir(page, '.military-track')).toBe('ltr');
      expect(await computedDir(page, '.progress-token-row')).toBe('ltr');
      const tokens = page
        .locator('.progress-token-row')
        .filter({ visible: true })
        .first()
        .locator('[data-fx-token]');
      expect(
        await tokens.evaluateAll((els) => els.map((el) => el.getAttribute('data-fx-token') ?? ''))
      ).toEqual(enTokens);
      const xs = await tokens.evaluateAll((els) => els.map((el) => el.getBoundingClientRect().x));
      expect(xs).toEqual([...xs].sort((a, b) => a - b));
      const card = await rect(page, `.card-structure [data-card="${cardId}"]`);
      expectNear(card.x, enCard.x, 'the card slot x');
      expectNear(card.y, enCard.y, 'the card slot y');
      expectNear(card.w, enCard.w, 'the card width');
    });

    await test.step('the card sheet preserves the board card geometry', async () => {
      await page.locator('.card-structure [data-card][data-accessible]').first().click();
      const face = page.locator('dialog[open] [data-card-sheet] .card-face').first();
      await expect(face).toBeVisible();
      expect(await face.evaluate((el) => getComputedStyle(el).direction)).toBe('ltr');
      expect(await computedDir(page, 'dialog[open] [data-card-sheet] .name-text')).toBe('rtl');
      expect(
        await face
          .locator(':scope > span')
          .last()
          .evaluate((node) => getComputedStyle(node).direction)
      ).toBe('rtl');
      const rows = face.locator('.cost, .band');
      for (const row of await rows.all())
        expect(await row.evaluate((el) => getComputedStyle(el).direction)).toBe('ltr');
      await page.keyboard.press('Escape');
      await expect(page.locator('dialog[open] [data-card-sheet]')).toHaveCount(0);
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
      const result = 'dialog[open]:has([data-score-table])';
      await expect(page.locator(result)).toBeVisible();
      expect(await documentDir(page)).toEqual({ dir: 'rtl', lang: 'fa' });
      expect(await computedDir(page, result)).toBe('rtl');
      expectNoHorizontalScroll(await probeLayout(page), 'result');
    });

    expect(problems, 'console errors, page errors or failed requests').toEqual([]);
  });
});
