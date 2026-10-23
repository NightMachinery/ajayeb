// Haftgane's rules reference (M18.rules-reference): the /rules/haftgane page
// and its in-game overlay (web/src/lib/games/haftgane/rules/). Tagged
// @haftgane-rules and not @smoke, so it stays out of `make verify`; run it
// with `make e2e GREP=@haftgane-rules`.
//
// - The page lists every section with its deep-link id, has no horizontal
//   scroll on a 390 px phone or a 1280 px desktop, and its contents link to
//   each section.
// - A deep link (/rules/haftgane#military) lands on its section, with the
//   conflict diagram and the rule's text in it.
// - openHaftganeRules() opens the overlay at a section, as a bottom sheet on
//   a phone and a side panel on a desktop. No Haftgane screen calls it yet,
//   so the dev gallery's button does.

import { expect, type Locator, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';

/** The section ids of games/haftgane/rules/sections.ts, in order. */
const SECTION_IDS = [
  'goal',
  'terms',
  'resources',
  'setup',
  'age',
  'actions',
  'costs',
  'trade',
  'turn-order',
  'military',
  'science',
  'commerce',
  'guilds',
  'wonders',
  'scoring',
  'player-counts'
];

const PHONE = { width: 390, height: 844 };
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

test.describe('Haftgane rules page', { tag: ['@haftgane-rules'] }, () => {
  for (const vp of [PHONE, DESKTOP]) {
    test(`/rules/haftgane at ${vp.width} has every section and no horizontal scroll`, async ({
      page
    }) => {
      const problems = watchHygiene(page);
      await page.setViewportSize(vp);
      await page.goto('/rules/haftgane');
      const main = page.locator('[data-rules-page="haftgane"]');
      await expect(main).toBeVisible();

      const ids = await main
        .locator('[data-rules-section]')
        .evaluateAll((els) => els.map((el) => el.id));
      expect(ids).toEqual(SECTION_IDS);
      await expect(page.locator('[data-rules-toc="list"] [data-rules-toc-item]')).toHaveCount(
        SECTION_IDS.length
      );
      await expect(main.locator('[data-rules-wonder]')).toHaveCount(7);
      expect(await horizontalOverflow(page)).toBeLessThanOrEqual(0);

      // The contents link to each section: following one lands on it. (One
      // with enough page below it to reach the top of the scroller.)
      await page.locator('[data-rules-toc-item="guilds"]').click();
      await expect(page).toHaveURL(/\/rules\/haftgane#guilds$/);
      const guilds = page.locator('#guilds');
      await expect(guilds).toBeInViewport();
      await expect.poll(() => offsetInScroller(guilds, main)).toBeLessThan(80);
      expect(problems).toEqual([]);
    });
  }

  test('a deep link lands on its section, with its diagram and text', async ({ page }) => {
    const problems = watchHygiene(page);
    await page.setViewportSize(PHONE);
    await page.goto('/rules/haftgane#military');
    const main = page.locator('[data-rules-page="haftgane"]');
    const military = page.locator('#military');
    await expect(military).toBeInViewport();
    const offset = await offsetInScroller(military, main);
    expect(offset).toBeGreaterThanOrEqual(-2);
    expect(offset).toBeLessThan(80);

    await expect(military.getByRole('heading', { name: 'Military conflicts' })).toBeVisible();
    const diagram = military.locator('[data-rules-diagram="conflicts"]');
    await expect(diagram).toBeVisible();
    await expect(diagram.locator('[data-conflict="defeat"]')).toBeVisible();
    await expect(diagram.locator('[data-conflict="victory"]')).toBeVisible();
    await expect(military).toContainText('take a victory token, worth 1 point in Age I');

    // A second deep link, opened afresh, lands on its own section.
    await page.goto('/rules/haftgane#trade');
    const trade = page.locator('#trade');
    await expect(trade).toBeInViewport();
    await expect(trade.locator('[data-rules-diagram="trade"]')).toBeVisible();
    await expect(trade).toContainText('West Trading Post');
    expect(problems).toEqual([]);
  });
});

test.describe('Haftgane rules overlay', { tag: ['@haftgane-rules'] }, () => {
  const cases = [
    { vp: PHONE, kind: 'sheet' },
    { vp: DESKTOP, kind: 'drawer' }
  ] as const;

  for (const c of cases) {
    test(`openHaftganeRules('trade') opens a ${c.kind} at ${c.vp.width}`, async ({ page }) => {
      const problems = watchHygiene(page);
      await page.setViewportSize(c.vp);
      await page.goto('/dev/gallery');
      const open = page.locator('[data-testid="gallery-haftgane-rules-open-trade"]');
      await open.scrollIntoViewIfNeeded();
      await open.click();

      const overlay = page.locator(`[data-rules-overlay="${c.kind}"]`);
      await expect(overlay).toBeVisible();
      await expect(overlay.locator('[data-rules-panel="haftgane"] [data-rules-title]')).toHaveText(
        'Haftgane rules'
      );
      await expect(overlay.locator('[data-rules-section="trade"]')).toBeInViewport();
      expect(
        await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)
      ).toBeLessThanOrEqual(0);

      await overlay.locator('[data-control="rules-close"]').click();
      await expect(overlay).toBeHidden();
      expect(problems).toEqual([]);
    });
  }
});
