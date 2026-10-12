// The rules reference (M10.RR1): the /rules page and the in-game overlay
// (web/src/lib/components/rules/).
//
// - /rules fits a 360 px phone and a 1280 px desktop with no horizontal
//   scroll, lists every section with its deep-link id, and a deep link
//   (/rules#military) lands on its section. Home links to it.
// - Inside a hot-seat game, the game menu's Rules item opens the rules
//   without leaving the game: a side panel at 1280 (the board stays visible
//   and a card stays clickable beside it) and a bottom sheet at 360 and on
//   the mobile project's phone. A contents chip jumps to its section, the
//   panel scrolls on its own, Close (or Escape) closes it, and the game is
//   exactly as it was: same URL, same room version, same view.

import { expect, type Locator, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { openHotseatRoom } from '../fixtures/scenario';

/** The section ids of components/rules/sections.ts, in order. */
const SECTION_IDS = [
  'goal',
  'setup',
  'turn',
  'costs',
  'military',
  'science',
  'wonders',
  'tokens',
  'guilds',
  'age-end',
  'scoring',
  'hints',
  'glossary'
];

const PHONE = { width: 360, height: 740 };
const DESKTOP = { width: 1280, height: 800 };

/** The page's horizontal overflow: the document and the rules page's own scroller. */
async function horizontalOverflow(page: Page): Promise<number> {
  return page.evaluate(() => {
    const doc = document.documentElement.scrollWidth - window.innerWidth;
    const main = document.querySelector<HTMLElement>('[data-rules-page]');
    const own = main ? main.scrollWidth - main.clientWidth : 0;
    return Math.max(doc, own);
  });
}

/** Where `target` sits relative to the top of the box of `scroller`, in px. */
async function offsetInScroller(target: Locator, scroller: Locator): Promise<number> {
  const [t, s] = await Promise.all([target.boundingBox(), scroller.boundingBox()]);
  if (!t || !s) throw new Error('no box to measure');
  return t.y - s.y;
}

test.describe('rules page', () => {
  for (const vp of [PHONE, DESKTOP]) {
    test(`/rules at ${vp.width} has every section and no horizontal scroll`, async ({ page }) => {
      const problems = watchHygiene(page);
      await page.setViewportSize(vp);
      await page.goto('/rules');
      const main = page.locator('[data-rules-page]');
      await expect(main).toBeVisible();

      const ids = await page
        .locator('[data-rules-page] [data-rules-section]')
        .evaluateAll((els) => els.map((el) => el.id));
      expect(ids).toEqual(SECTION_IDS);
      await expect(page.locator('[data-rules-toc="list"] [data-rules-toc-item]')).toHaveCount(
        SECTION_IDS.length
      );
      expect(await horizontalOverflow(page)).toBeLessThanOrEqual(0);

      // The contents link to each section: following one lands on it.
      await page.locator('[data-rules-toc-item="scoring"]').click();
      await expect(page).toHaveURL(/\/rules#scoring$/);
      const scoring = page.locator('#scoring');
      await expect(scoring).toBeInViewport();
      expect(await offsetInScroller(scoring, main)).toBeLessThan(80);
      expect(problems).toEqual([]);
    });
  }

  test('a deep link lands on its section, and Home links to the rules', async ({ page }) => {
    const problems = watchHygiene(page);
    await page.setViewportSize(PHONE);
    await page.goto('/rules#military');
    const main = page.locator('[data-rules-page]');
    const military = page.locator('#military');
    await expect(military).toBeInViewport();
    const offset = await offsetInScroller(military, main);
    expect(offset).toBeGreaterThanOrEqual(-2);
    expect(offset).toBeLessThan(80);
    await expect(military.locator('[data-rules-diagram="military"]')).toBeVisible();

    await page.goto('/');
    await page.locator('[data-control="home-rules"]').click();
    await expect(page).toHaveURL(/\/rules$/);
    await expect(main).toBeVisible();
    expect(problems).toEqual([]);
  });
});

/** The game as the page holds it: what must not change while the rules are open. */
async function gameState(page: Page): Promise<{ url: string; version: number; view: string }> {
  return page.evaluate(() => {
    const hook = window.__ajayeb;
    const view = hook?.view();
    const room = hook?.room();
    if (!view || !room) throw new Error('no game on screen');
    return { url: location.href, version: room.version, view: JSON.stringify(view) };
  });
}

test.describe('rules in a game', { tag: ['@mobile'] }, () => {
  const cases = [
    { name: 'a side panel at 1280', vp: DESKTOP, kind: 'drawer' },
    { name: 'a bottom sheet at 360', vp: PHONE, kind: 'sheet' },
    { name: "a bottom sheet on the phone's own screen", vp: null, kind: 'sheet' }
  ] as const;

  for (const c of cases) {
    test(`the game menu opens the rules as ${c.name}`, async ({ page }, info) => {
      // The desktop project sets each viewport itself; the mobile project
      // runs only the phone's own screen, with touch.
      test.skip(
        (info.project.name === 'mobile') !== (c.vp === null),
        'one case per project and viewport'
      );
      const problems = watchHygiene(page);
      if (c.vp) await page.setViewportSize(c.vp);
      const driver = await openHotseatRoom(page, 'circus-maximus');
      await driver.nextTurn();
      const before = await gameState(page);

      await driver.visible('[data-control="menu"]').click();
      await page.locator('[data-control="menu-rules"]').click();
      const overlay = page.locator(`[data-rules-overlay="${c.kind}"]`);
      await expect(overlay).toBeVisible();
      await expect(page.locator('[data-rules-panel] [data-rules-title]')).toBeVisible();
      expect(page.url()).toBe(before.url);

      // The panel's own scroller: the side panel itself, or the sheet's panel.
      const scroller =
        c.kind === 'drawer' ? overlay : page.locator('dialog[open] .sheet-panel').first();

      if (c.kind === 'drawer') {
        // The board stays visible and usable beside the panel: an accessible
        // card is not covered by it.
        const card = page.locator('.card-structure [data-card][data-accessible]').first();
        await expect(card).toBeInViewport();
        const covered = await card.evaluate((el) => {
          const r = el.getBoundingClientRect();
          const hit = document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2);
          return !hit || !el.contains(hit);
        });
        expect(covered, 'the side panel covers no takeable card').toBe(false);
      }

      // A contents chip jumps to its section inside the panel.
      await overlay.locator('[data-rules-toc-item="military"]').click();
      const military = overlay.locator('[data-rules-section="military"]');
      await expect(military).toBeInViewport();
      await expect
        .poll(async () => offsetInScroller(military, scroller))
        .toBeLessThan(c.kind === 'drawer' ? 220 : 260);

      // The panel scrolls on its own; the page does not scroll sideways.
      const top = await scroller.evaluate((el) => el.scrollTop);
      await scroller.evaluate((el) => el.scrollBy({ top: 600, behavior: 'instant' }));
      await expect.poll(() => scroller.evaluate((el) => el.scrollTop)).toBeGreaterThan(top);
      expect(
        await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)
      ).toBeLessThanOrEqual(0);

      await overlay.locator('[data-control="rules-close"]').click();
      await expect(overlay).toBeHidden();
      expect(await gameState(page)).toEqual(before);

      // Escape closes it too.
      await driver.visible('[data-control="menu"]').click();
      await page.locator('[data-control="menu-rules"]').click();
      await expect(overlay).toBeVisible();
      await page.keyboard.press('Escape');
      await expect(overlay).toBeHidden();
      expect(await gameState(page)).toEqual(before);
      expect(problems).toEqual([]);
    });
  }
});
