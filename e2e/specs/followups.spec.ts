// M10.hints-followups: player-facing token values, the result review and static recap.
import { expect, type Page, type TestInfo } from '@playwright/test';
import { test } from '../fixtures/base';
import { GameDriver } from '../fixtures/driver';
import { watchHygiene } from '../fixtures/hygiene';
import { currentView, openHotseatRoom, playKey } from '../fixtures/scenario';
import { createDevRoom, recordGame, recordPolicy } from '../fixtures/ws-game';
import type { Choice, Resource, View } from '../../web/src/lib/wire';

async function photograph(
  page: Page,
  info: TestInfo,
  scene: string,
  selector?: string
): Promise<void> {
  await page.mouse.move(0, 0);
  await expect(page.locator('[data-tooltip-layer] [role="tooltip"]')).toBeHidden();
  await page.evaluate(() => document.fonts.ready);
  const originalViewport = page.viewportSize();
  if (selector && originalViewport) {
    const box = await page.locator(selector).boundingBox();
    if (box && box.height > originalViewport.height) {
      // The gallery scrolls inside its main element. Grow the viewport so
      // it paints the entire grid rather than clipping a tall screenshot.
      await page.setViewportSize({
        width: originalViewport.width,
        height: Math.ceil(box.height + 48)
      });
      await page.locator(selector).scrollIntoViewIfNeeded();
    }
  }
  for (const theme of ['light', 'dark']) {
    await page.evaluate((value) => {
      localStorage.setItem('ajayeb:theme', value);
      document.documentElement.dataset['theme'] = value;
    }, theme);
    const target = selector ? page.locator(selector) : page;
    await target.screenshot({ path: info.outputPath(`${scene}-${theme}.png`) });
  }
  if (originalViewport) await page.setViewportSize(originalViewport);
}

class ProducerFound extends Error {
  readonly choice: Choice;
  readonly resource: Resource;
  constructor(choice: Choice, resource: Resource) {
    super('producer found');
    this.choice = choice;
    this.resource = resource;
  }
}

// Fixed seed; stop before a server-offered producer that raises the other
// seat's price. No rules or predictions are reproduced in this test.
async function beforeProducer(
  origin: string
): Promise<{ keys: string[]; key: string; resource: Resource }> {
  const keys: string[] = [];
  try {
    await recordGame(origin, '11', (view: View) => {
      const choice =
        view.phase === 'play' && !view.pending
          ? view.choices.find(
              (c) =>
                c.kind === 'construct' &&
                c.outcome?.prices?.some((p) => p.seat !== view.toMove && p.to > p.from)
            )
          : undefined;
      const price = choice?.outcome?.prices?.find((p) => p.seat !== view.toMove && p.to > p.from);
      if (choice && price) throw new ProducerFound(choice, price.resource);
      const c = recordPolicy(view);
      keys.push(c.key);
      return c;
    });
  } catch (error) {
    if (error instanceof ProducerFound)
      return { keys, key: error.choice.key, resource: error.resource };
    throw error;
  }
  throw new Error('seed 11 did not offer a producer');
}

test.describe('web follow-ups', { tag: '@mobile' }, () => {
  let problems: string[] = [];
  test.afterEach(() => {
    expect(problems).toEqual([]);
  });
  test.beforeEach(async ({ page, isMobile }) => {
    problems = watchHygiene(page);
    await page.setViewportSize(
      isMobile ? { width: 360, height: 740 } : { width: 1280, height: 800 }
    );
    await page.addInitScript(() => {
      const settings = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}');
      localStorage.setItem(
        'ajayeb:settings',
        JSON.stringify({ ...settings, hotseatPassDevice: false })
      );
      localStorage.setItem('ajayeb:name', 'Ana');
    });
  });

  test('a board token opens its sheet with hint values; the pair pick shows both seats', async ({
    page
  }, info) => {
    const driver = await openHotseatRoom(page, 'science-pair');
    await driver.nextTurn();
    await page.locator('[data-token-open]').first().click();
    const sheet = page.locator('dialog[open] [data-token-sheet]');
    await expect(sheet).toBeVisible();
    await expect(sheet.locator('[data-token-value]').first()).toBeVisible();
    await photograph(page, info, 'token-sheet');
    await page.keyboard.press('Escape');
    await expect(sheet).toBeHidden();
    await playKey(driver, 'construct:library');
    const pick = page.locator('[data-token-choice="board"]');
    await expect(pick).toBeVisible();
    const view = await currentView(page);
    expect(view.hints?.tokens.length).toBeGreaterThan(0);
    await expect(
      pick.locator('[data-token-values]').first().locator('[data-token-value]')
    ).toHaveCount(2);
    await photograph(page, info, 'token-choice');
  });

  test('a token sheet from the previous game closes when the player starts a rematch', async ({
    page
  }) => {
    await openHotseatRoom(page, 'military-supremacy');
    const previous = page.url();
    await page.locator('[data-token-open]').first().click();
    await expect(page.locator('dialog[open] [data-token-sheet]')).toBeVisible();

    // The same player can have two tabs open. End the game from the other
    // tab while this tab is inspecting a token, then rematch through the UI.
    const other = await page.context().newPage();
    try {
      await other.goto(previous);
      const driver = new GameDriver(other);
      await driver.waitReady();
      await playKey(driver, 'construct:walls');
      await expect(page.locator('[data-game-over] [data-rematch]')).toBeVisible();
      await page.locator('[data-rematch]').click();
      await page.waitForURL((url) => url.href !== previous);
      await new GameDriver(page).waitReady();
      await expect(page.locator('[data-phase="wonderDraft"]')).toBeVisible();
      await expect(page.locator('dialog[open] [data-token-sheet]')).toHaveCount(0);
    } finally {
      await other.close();
    }
  });

  test('the viewer ending the game sees their own review and a floating final-board bar', async ({
    page
  }, info) => {
    await page.goto('/dev/new?mode=ai&level=easy&seat=0&seed=7&scenario=military-supremacy');
    await page.waitForURL(/\/play\/[A-Z0-9]+$/);
    const driver = new GameDriver(page);
    await driver.waitReady();
    await playKey(driver, 'construct:walls');
    const review = page.locator('[data-game-over] [data-end-review] [data-move-review]');
    await expect(review).toBeVisible();
    await expect(review.locator('[data-review-played]')).toHaveText('You built Walls.');
    await photograph(page, info, 'game-over-review');
    const button = page.locator('[data-view-board]');
    expect(await button.evaluate((el) => el.scrollWidth <= el.clientWidth)).toBe(true);
    if (info.project.name === 'mobile') {
      await page.setViewportSize({ width: 390, height: 740 });
      expect(await button.evaluate((el) => el.scrollWidth <= el.clientWidth)).toBe(true);
      await page.setViewportSize({ width: 360, height: 740 });
    }
    await button.click();
    await expect(page.locator('[data-game-over] [data-peek-bar]')).toHaveAttribute(
      'data-peek-bar',
      'float'
    );
    await expect(page.locator('.card-structure')).toBeVisible();
    await photograph(page, info, 'final-board');
  });

  test('a producer raises the next seat’s trade chip and leaves a slot trace until that seat moves', async ({
    page,
    baseURL
  }, info) => {
    if (!baseURL) throw new Error('no baseURL');
    const at = await beforeProducer(baseURL);
    const room = await createDevRoom(baseURL, { mode: 'hotseat', seed: '11', actions: at.keys });
    await page.goto(room.path);
    const driver = new GameDriver(page);
    await driver.waitReady();
    const before = await currentView(page);
    const card = before.choices.find((c) => c.key === at.key)?.card;
    const slot = before.structure.find((s) => s.card === card)?.index;
    await playKey(driver, at.key);
    await driver.nextTurn();
    const caret = page
      .locator(`[data-resource="${at.resource}"] [data-recap-tick="up"]`)
      .filter({ visible: true });
    await expect(caret.first()).toBeVisible();
    await expect(page.locator(`[data-slot="${slot}"][data-recap-trace]`)).toBeVisible();
    await photograph(page, info, 'recap-caret-trace');
    const next = await driver.nextTurn();
    await driver.play(next);
    await driver.nextTurn();
    await expect(page.locator(`[data-slot="${slot}"][data-recap-trace]`)).toHaveCount(0);
  });

  test('the own strip keeps Ana readable beside YOU, a device link and a full-colour token', async ({
    page
  }, info) => {
    await page.goto('/dev/new?mode=ai&level=easy&seat=0&seed=7&scenario=theology-chain');
    await page.waitForURL(/\/play\/[A-Z0-9]+$/);
    await new GameDriver(page).waitReady();
    const strip = page.locator('.player-strip').filter({ has: page.locator('[data-self-mark]') });
    // Desktop uses city panels; keep this check on the phone strip.
    if (await strip.count()) {
      const name = strip.locator('[data-control="open-city"] [dir="auto"]');
      await expect(name).toHaveText('Ana');
      expect(await name.evaluate((el) => el.scrollWidth <= el.clientWidth)).toBe(true);
      await expect(strip.locator('[data-control="device-link"]')).toBeVisible();
      const token = strip.locator('[data-owned]');
      await expect(token).toBeVisible();
      expect(await token.evaluate((el) => getComputedStyle(el).filter)).toBe('none');
    }
    await photograph(page, info, 'own-strip');
  });

  test('a phone-landscape skipped-choice toast stays clear of the own strip and turn controls', async ({
    page
  }, info) => {
    for (const size of [
      { width: 844, height: 390 },
      { width: 740, height: 360 }
    ]) {
      await page.setViewportSize(size);
      const driver = await openHotseatRoom(page, 'mausoleum-empty');
      await driver.nextTurn();
      await playKey(driver, 'wonder:mausoleum:rostrum', 'wonderFirst');
      const toast = page.locator('[data-toast]');
      await expect(toast).toBeVisible();
      if (size.width === 844) await photograph(page, info, 'landscape-toast');
      const overlap = await toast.evaluate((el) => {
        const t = el.getBoundingClientRect();
        const strip = document.querySelector('.area-home .player-strip')!.getBoundingClientRect();
        const turn = document.querySelector('.area-turn')!.getBoundingClientRect();
        const stripOverlap =
          t.top < strip.bottom &&
          t.bottom > strip.top &&
          t.left < strip.right &&
          t.right > strip.left;
        const turnOverlap =
          t.top < turn.bottom && t.bottom > turn.top && t.left < turn.right && t.right > turn.left;
        return stripOverlap || turnOverlap;
      });
      expect(overlap).toBe(false);
    }
  });

  test('every 48 px mini card keeps its cost and effect band inside one row', async ({
    page
  }, info) => {
    await page.goto('/dev/gallery');
    const grid = page.locator('[data-card-grid="48"]');
    await expect(grid).toBeVisible();
    await grid.scrollIntoViewIfNeeded();
    const overflow = await grid.evaluate((el) =>
      [...el.querySelectorAll('.card-face')].flatMap((card, i) => {
        const problems: string[] = [];
        for (const selector of ['.cost', '.band']) {
          const row = card.querySelector<HTMLElement>(selector)!;
          const box = row.getBoundingClientRect();
          for (const part of row.querySelectorAll<HTMLElement>('[data-chain], [data-effect]')) {
            const r = part.getBoundingClientRect();
            if (
              r.top < box.top - 0.5 ||
              r.bottom > box.bottom + 0.5 ||
              r.left < box.left - 0.5 ||
              r.right > box.right + 0.5
            ) {
              problems.push(
                `${card.parentElement?.dataset['miniCard'] ?? i} ${selector} ${part.dataset['chain'] ?? part.dataset['effect']} ${JSON.stringify({ box: { x: box.x, y: box.y, width: box.width, height: box.height }, part: { x: r.x, y: r.y, width: r.width, height: r.height } })}`
              );
            }
          }
        }
        return problems;
      })
    );
    await photograph(page, info, 'mini-cards', '[data-card-grid="48"]');
    expect(overflow).toEqual([]);
  });
});
