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

/**
 * What the element at `over` covers among the elements `targets` matches,
 * by bounding box, as "name WxH" lines for a failure message; empty when it
 * covers none. Elements inside `over`, and boxes of 1 px or less (the
 * screen-reader-only labels), do not count.
 */
async function covered(page: Page, over: string, targets: string): Promise<string[]> {
  return page.evaluate(
    ([over, targets]) => {
      const top = document.querySelector(over);
      if (!top) return [`nothing matches ${over}`];
      const a = top.getBoundingClientRect();
      const hits: string[] = [];
      for (const el of document.querySelectorAll(targets)) {
        if (top.contains(el) || el.contains(top)) continue;
        const b = el.getBoundingClientRect();
        if (b.width <= 1 || b.height <= 1) continue;
        const w = Math.min(a.right, b.right) - Math.max(a.left, b.left);
        const h = Math.min(a.bottom, b.bottom) - Math.max(a.top, b.top);
        if (w <= 0.5 || h <= 0.5) continue;
        const name =
          el.getAttribute('data-card') ??
          `${el.tagName.toLowerCase()}.${String(el.getAttribute('class') ?? '').split(' ')[0]}`;
        hits.push(`${name} ${Math.round(w)}x${Math.round(h)}`);
      }
      return hits;
    },
    [over, targets] as const
  );
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
    const bar = '[data-game-over] [data-peek-bar="float"] [role="status"]';
    await expect(page.locator(bar)).toBeVisible();
    await expect(page.locator('.card-structure')).toBeVisible();
    await photograph(page, info, 'final-board');
    // The floating bar keeps its headline and its 44 px button at every
    // layout (an inline-size container on the bar itself once squeezed it to
    // a 22 px sliver), and lies over the turn bar's row, clear of every chip,
    // icon and label in both strips.
    const sizes =
      info.project.name === 'mobile'
        ? [
            { width: 360, height: 740 },
            { width: 390, height: 844 },
            { width: 844, height: 390 },
            { width: 740, height: 360 }
          ]
        : [{ width: 1280, height: 800 }];
    for (const size of sizes) {
      await page.setViewportSize(size);
      const at = `${size.width}x${size.height}`;
      const text = page.locator(`${bar} [data-peek-text]`);
      await expect(text, at).toHaveText(/\S/);
      const fit = await text.evaluate((el) => ({
        width: el.getBoundingClientRect().width,
        cut: el.scrollWidth > el.clientWidth + 1
      }));
      expect(fit.width, `${at}: headline width`).toBeGreaterThan(60);
      expect(fit.cut, `${at}: headline cut short`).toBe(false);
      const open = await page.locator(`${bar} [data-peek-open]`).boundingBox();
      expect(open?.height ?? 0, `${at}: open button height`).toBeGreaterThanOrEqual(44);
      expect(
        (open?.y ?? 0) + (open?.height ?? 0),
        `${at}: open button on screen`
      ).toBeLessThanOrEqual(size.height);
      const strips = '.area-home :not(:has(*)), .area-away :not(:has(*))';
      expect(await covered(page, bar, strips), `${at}: strip elements under the bar`).toEqual([]);
    }
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

  test('a phone-landscape skipped-choice toast covers neither strip, the turn bar nor a card to take', async ({
    page
  }, info) => {
    // In phone landscape the own column is only as wide as a strip, so a long
    // sentence there grew up over it; the toast takes the first lane clear of
    // the cards to take and both strips: here the structure's head, where no
    // card can be taken at this point of the age.
    const cases = [
      { width: 844, height: 390, locale: 'en' },
      { width: 740, height: 360, locale: 'en' },
      { width: 740, height: 360, locale: 'fa' }
    ];
    for (const { width, height, locale } of cases) {
      const at = `${width}x${height} ${locale}`;
      await page.setViewportSize({ width, height });
      if (locale !== 'en') {
        // The app's own language setting; the earlier cases left the page on its origin.
        await page.evaluate((value) => {
          const raw = localStorage.getItem('ajayeb:settings');
          const settings = raw ? (JSON.parse(raw) as Record<string, unknown>) : {};
          localStorage.setItem('ajayeb:settings', JSON.stringify({ ...settings, locale: value }));
        }, locale);
      }
      const driver = await openHotseatRoom(page, 'mausoleum-empty');
      await driver.nextTurn();
      await playKey(driver, 'wonder:mausoleum:rostrum', 'wonderFirst');
      const toast = '.toast-lane [data-toast]';
      await expect(page.locator(toast), at).toBeVisible();
      await expect(page.locator('.toast-lane'), at).toHaveAttribute('data-toast-lane', 'head');
      if (width === 844) await photograph(page, info, 'landscape-toast');
      const takeable = (await currentView(page)).choices.flatMap((c) =>
        c.card ? [`.card-structure [data-card="${c.card}"]`] : []
      );
      expect(takeable.length, `${at}: cards to take`).toBeGreaterThan(0);
      const targets = [
        '.area-home .player-strip',
        '.area-away .player-strip',
        '.area-turn',
        ...takeable
      ];
      expect(await covered(page, toast, targets.join(', ')), at).toEqual([]);
    }
  });

  test('late in an age a phone-landscape toast leaves the upper-row cards to take uncovered', async ({
    page
  }, info) => {
    test.setTimeout(120_000);
    // Late in an age the cards to take are near the structure's head, where
    // the toast sat. Seat 0 keeps the Statue of Zeus unbuilt while both
    // players discard the lowest card offered, until a card near the head is
    // offered; then Zeus is built and its destroy choice is skipped (the
    // opponent owns no brown card), which raises the toast.
    const cases = [
      { width: 844, height: 390, locale: 'en' },
      { width: 740, height: 360, locale: 'fa' }
    ];
    for (const { width, height, locale } of cases) {
      const at = `${width}x${height} ${locale}`;
      await page.setViewportSize({ width, height });
      if (locale !== 'en') {
        await page.evaluate((value) => {
          const raw = localStorage.getItem('ajayeb:settings');
          const settings = raw ? (JSON.parse(raw) as Record<string, unknown>) : {};
          localStorage.setItem('ajayeb:settings', JSON.stringify({ ...settings, locale: value }));
        }, locale);
      }
      const driver = await openHotseatRoom(page, 'zeus-no-target');
      /** How far below the structure's top the highest card to take starts, and the lowest one's key. */
      const offered = async (): Promise<{
        headGap: number;
        lowest: string | null;
        toMove: number;
      }> => {
        const view = await currentView(page);
        const cards = view.choices.flatMap((c) =>
          c.kind === 'discard' && c.card ? [{ card: c.card, key: c.key }] : []
        );
        const boxes = await page.evaluate(
          (ids) => {
            const top = document.querySelector('.area-structure')!.getBoundingClientRect().top;
            return ids.map((id) => {
              const r = document
                .querySelector(`.card-structure [data-card="${id}"]`)!
                .getBoundingClientRect();
              return { top: r.top - top, bottom: r.bottom - top };
            });
          },
          cards.map((c) => c.card)
        );
        let lowest: string | null = null;
        let bottom = -Infinity;
        boxes.forEach((b, i) => {
          if (b.bottom > bottom) [bottom, lowest] = [b.bottom, cards[i]!.key];
        });
        return {
          headGap: Math.min(...boxes.map((b) => b.top)),
          lowest,
          toMove: view.toMove ?? -1
        };
      };
      let state = await offered();
      for (let moves = 0; state.headGap > 40 || state.toMove !== 0; moves++) {
        expect(moves, `${at}: a card near the head is offered within an age`).toBeLessThan(20);
        expect(state.lowest, at).not.toBeNull();
        await playKey(driver, state.lowest!);
        state = await offered();
      }
      const zeus = (await currentView(page)).choices.find((c) =>
        c.key.startsWith('wonder:statue-of-zeus:')
      );
      expect(zeus, `${at}: Zeus can still be built`).toBeTruthy();
      await playKey(driver, zeus!.key, 'wonderFirst');
      const toast = '.toast-lane [data-toast]';
      await expect(page.locator(toast), at).toBeVisible();
      const takeable = (await currentView(page)).choices.flatMap((c) =>
        c.card ? [`.card-structure [data-card="${c.card}"]`] : []
      );
      expect(takeable.length, `${at}: cards to take`).toBeGreaterThan(0);
      const targets = [
        '.area-home .player-strip',
        '.area-away .player-strip',
        '.area-turn',
        ...takeable
      ];
      expect(await covered(page, toast, targets.join(', ')), at).toEqual([]);
      await expect(page.locator('.toast-lane'), at).not.toHaveAttribute('data-toast-lane', 'head');
      if (width === 844) await photograph(page, info, 'landscape-toast-late');
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
