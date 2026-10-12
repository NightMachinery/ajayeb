// The tooltip layer (web/src/lib/ui/tooltip.svelte.ts and
// components/shell/TooltipLayer.svelte), on its gallery page
// /dev/gallery/tooltips.
//
// Desktop: a hover shows after the delay, keyboard focus shows at once and
// describes the anchor, Escape closes, and the innermost anchor wins (a
// chain chip inside a card shows the chip's stacked tooltip, the rest of the
// card the card's own).
//
// Mobile (touch, model B): a tap on a passive element shows; a long-press on
// an actionable card shows its tooltip without opening the card sheet; the
// next tap closes it without acting; a plain tap still acts. Touches are
// real CDP touch events (Input.dispatchTouchEvent), so the page sees touch
// pointer events, as on a phone.

import AxeBuilder from '@axe-core/playwright';
import { expect, type Locator, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';

const URL = '/dev/gallery/tooltips';
const AXE_TAGS = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa'];

function bubble(page: Page): Locator {
  return page.locator('[data-tooltip-layer] [role="tooltip"]');
}

async function center(locator: Locator): Promise<{ x: number; y: number }> {
  const box = await locator.boundingBox();
  if (!box) throw new Error('element has no box');
  return { x: box.x + box.width / 2, y: box.y + box.height / 2 };
}

/** A finger resting on the element for `ms`, then lifting. */
async function longPress(page: Page, locator: Locator, ms = 800): Promise<void> {
  const { x, y } = await center(locator);
  const cdp = await page.context().newCDPSession(page);
  try {
    await cdp.send('Input.dispatchTouchEvent', {
      type: 'touchStart',
      touchPoints: [{ x, y }]
    });
    await page.waitForTimeout(ms);
    await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] });
  } finally {
    await cdp.detach();
  }
}

/** A short tap at the element's centre, as a real touch. */
async function tap(page: Page, locator: Locator): Promise<void> {
  const { x, y } = await center(locator);
  await page.touchscreen.tap(x, y);
}

test('the main gallery links the tooltip gallery', async ({ page }) => {
  await page.goto('/dev/gallery');
  await page.getByTestId('gallery-tooltips-link').click();
  await page.waitForURL((url) => url.pathname === URL);
  await expect(page.getByRole('heading', { level: 1, name: 'Tooltips' })).toBeVisible();
});

test.describe('tooltips on a fine pointer', () => {
  test.beforeEach(({}, info) => {
    test.skip(info.project.name !== 'desktop', 'hover and keyboard focus need a mouse');
  });

  test('a hover shows after the delay; Escape closes', async ({ page }) => {
    const problems = watchHygiene(page);
    await page.goto(URL);
    const text = page.getByTestId('tip-text');
    await text.hover();
    // Not at once: the hover delay is 450 ms.
    expect(await bubble(page).count()).toBe(0);
    await expect(bubble(page)).toBeVisible();
    await expect(bubble(page)).toHaveText(/Victory points/);

    await page.keyboard.press('Escape');
    await expect(bubble(page)).toHaveCount(0);

    // Leaving the anchor hides it too.
    await page.getByTestId('tip-params').hover();
    await expect(bubble(page)).toHaveText(/produce 2/i);
    await page.getByRole('heading', { level: 1 }).hover();
    await expect(bubble(page)).toHaveCount(0);
    expect(problems).toEqual([]);
  });

  test('keyboard focus shows at once and describes the anchor', async ({ page }) => {
    const problems = watchHygiene(page);
    await page.goto(URL);
    const card = page.getByTestId('tip-card');
    for (let i = 0; i < 12; i++) {
      await page.keyboard.press('Tab');
      if (await card.evaluate((el) => el === document.activeElement)) break;
    }
    await expect(card).toBeFocused();
    await expect(bubble(page)).toBeVisible();
    await expect(bubble(page)).toHaveText(/Baths/);
    await expect(card).toHaveAttribute('aria-describedby', /ajayeb-tooltip/);
    await expect(page.locator('#ajayeb-tooltip')).toHaveCount(1);

    await page.keyboard.press('Escape');
    await expect(bubble(page)).toHaveCount(0);
    await expect(card).not.toHaveAttribute('aria-describedby', /ajayeb-tooltip/);
    // Escape closed the tooltip, not anything else: the card is still focused.
    await expect(card).toBeFocused();
    expect(problems).toEqual([]);
  });

  test('the innermost anchor wins, and a chain chip stacks the card it makes free', async ({
    page
  }) => {
    const problems = watchHygiene(page);
    await page.goto(URL);
    const card = page.getByTestId('tip-card');
    const chip = card.locator('[data-chain="drop"]');

    await chip.hover();
    await expect(bubble(page)).toHaveText(/makes Aqueduct free/);
    const stack = page.locator('[data-tooltip-layer] [data-tooltip-stack]');
    await expect(stack).toBeVisible();
    await expect(stack.locator('[data-card-preview="aqueduct"]')).toBeVisible();

    // The same card, off the chip: the card's own tooltip, with no stack.
    const box = await card.boundingBox();
    if (!box) throw new Error('no card box');
    await page.mouse.move(box.x + box.width / 2, box.y + box.height * 0.75);
    await expect(bubble(page)).toHaveText(/Baths/);
    await expect(stack).toHaveCount(0);

    // A hover tooltip does not block a click: the card still acts.
    await card.click();
    await expect(page.getByTestId('tip-card-sheet')).toBeVisible();
    expect(problems).toEqual([]);
  });

  test('the dismiss layer takes the focus, keeps Tab inside and gives it back', async ({
    page
  }) => {
    const problems = watchHygiene(page);
    await page.goto(URL);
    const open = page.getByTestId('tip-dismiss-open');
    const layer = page.locator('[data-dismiss-layer]');
    const close = page.getByTestId('tip-dismiss-close');
    await open.focus();
    await page.keyboard.press('Enter');
    await expect(layer).toBeVisible();
    await expect(layer).toBeFocused();
    // Tab and Shift+Tab stay on the panel's one button.
    await page.keyboard.press('Tab');
    await expect(close).toBeFocused();
    await page.keyboard.press('Tab');
    await expect(close).toBeFocused();
    await page.keyboard.press('Shift+Tab');
    await expect(close).toBeFocused();
    // Escape closes it, and the focus goes back to the button that opened it.
    await page.keyboard.press('Escape');
    await expect(layer).toHaveCount(0);
    await expect(open).toBeFocused();
    // So does the panel's own Close button.
    await page.keyboard.press('Enter');
    await expect(layer).toBeFocused();
    await page.keyboard.press('Tab');
    await page.keyboard.press('Enter');
    await expect(layer).toHaveCount(0);
    await expect(open).toBeFocused();
    expect(problems).toEqual([]);
  });

  test('the bubble stays inside the viewport at the edges, and passes axe while open', async ({
    page
  }) => {
    await page.goto(`${URL}?edges`);
    for (const id of ['edge-tl', 'edge-tr', 'edge-bl', 'edge-br']) {
      await page.locator(`[data-tip-demo="${id}"] [data-chain]`).hover();
      await expect(page.locator('[data-tooltip-layer] [data-tooltip-stack]')).toBeVisible();
      const vp = page.viewportSize();
      if (!vp) throw new Error('no viewport');
      for (const part of ['[data-tooltip-bubble]', '[data-tooltip-stack]']) {
        const box = await page.locator(`[data-tooltip-layer] ${part}`).boundingBox();
        if (!box) throw new Error(`no ${part} box`);
        expect(box.x).toBeGreaterThanOrEqual(0);
        expect(box.y).toBeGreaterThanOrEqual(0);
        expect(box.x + box.width).toBeLessThanOrEqual(vp.width);
        expect(box.y + box.height).toBeLessThanOrEqual(vp.height);
      }
    }
    const group = page.locator('[data-tooltip-layer] .tooltip-group[data-placed]');
    await expect(group).toHaveCount(1);
    // Axe reads the colours as drawn, so it runs only once the fade-in has
    // ended: mid-fade the text is part transparent, and its contrast fails.
    await group.evaluate((el) =>
      Promise.all(el.getAnimations({ subtree: true }).map((a) => a.finished))
    );
    await expect.poll(() => group.evaluate((el) => getComputedStyle(el).opacity)).toBe('1');
    const results = await new AxeBuilder({ page })
      .withTags(AXE_TAGS)
      .include('[data-tooltip-layer]')
      .analyze();
    expect(
      results.violations.map(
        (v) => `${v.id}: ${v.nodes.map((n) => n.html + ' ' + n.failureSummary).join(' | ')}`
      )
    ).toEqual([]);
  });
});

test.describe('tooltips on touch', { tag: ['@mobile'] }, () => {
  test.beforeEach(({}, info) => {
    test.skip(info.project.name !== 'mobile', 'touch needs the phone project');
  });

  test('a tap on a passive element shows; the next tap closes it and is swallowed', async ({
    page
  }) => {
    const problems = watchHygiene(page);
    await page.goto(URL);
    await tap(page, page.getByTestId('tip-text'));
    await expect(bubble(page)).toHaveText(/Victory points/);

    // A tap on the Build button only closes the tooltip.
    await tap(page, page.getByTestId('tip-button'));
    await expect(bubble(page)).toHaveCount(0);
    await expect(page.getByTestId('tip-button-pressed')).toHaveText(/0 times/);
    // The button still acts on the next tap.
    await tap(page, page.getByTestId('tip-button'));
    await expect(page.getByTestId('tip-button-pressed')).toHaveText(/1 times/);
    expect(problems).toEqual([]);
  });

  test('a long-press on an actionable card shows its tooltip without opening the sheet', async ({
    page
  }) => {
    const problems = watchHygiene(page);
    await page.goto(URL);
    const card = page.getByTestId('tip-card');
    const opened = page.getByTestId('tip-card-opened');
    await card.scrollIntoViewIfNeeded();

    await longPress(page, card.locator('.art'));
    await expect(bubble(page)).toHaveText(/Baths/);
    await expect(page.locator('dialog[open]')).toHaveCount(0);
    await expect(opened).toHaveText(/opened 0 times/);

    // The next tap closes it and does not act.
    await tap(page, card);
    await expect(bubble(page)).toHaveCount(0);
    await expect(page.locator('dialog[open]')).toHaveCount(0);
    await expect(opened).toHaveText(/opened 0 times/);

    // A long-press on the chain chip inside it: the chip's stacked tooltip.
    await longPress(page, card.locator('[data-chain="drop"]'));
    await expect(bubble(page)).toHaveText(/makes Aqueduct free/);
    await expect(
      page.locator('[data-tooltip-layer] [data-tooltip-stack] [data-card-preview="aqueduct"]')
    ).toBeVisible();
    await tap(page, page.getByRole('heading', { level: 1 }));
    await expect(bubble(page)).toHaveCount(0);
    await expect(opened).toHaveText(/opened 0 times/);

    // A plain tap still opens the card.
    await tap(page, card);
    await expect(page.getByTestId('tip-card-sheet')).toBeVisible();
    await expect(opened).toHaveText(/opened 1 times/);
    expect(problems).toEqual([]);
  });

  test('the stacked tooltip fits a phone screen', async ({ page }) => {
    await page.goto(`${URL}?edges`);
    for (const id of ['edge-tl', 'edge-br']) {
      await tap(page, page.locator(`[data-tip-demo="${id}"] [data-chain]`));
      const stack = page.locator('[data-tooltip-layer] [data-tooltip-stack]');
      await expect(stack).toBeVisible();
      const vp = page.viewportSize();
      if (!vp) throw new Error('no viewport');
      for (const part of ['[data-tooltip-bubble]', '[data-tooltip-stack]']) {
        const box = await page.locator(`[data-tooltip-layer] ${part}`).boundingBox();
        if (!box) throw new Error(`no ${part} box`);
        expect(box.x).toBeGreaterThanOrEqual(0);
        expect(box.y).toBeGreaterThanOrEqual(0);
        expect(box.x + box.width).toBeLessThanOrEqual(vp.width);
        expect(box.y + box.height).toBeLessThanOrEqual(vp.height);
      }
      // One tap closes the bubble and its stack together.
      await tap(page, page.getByRole('heading', { level: 1 }));
      await expect(stack).toHaveCount(0);
      await expect(bubble(page)).toHaveCount(0);
    }
  });

  test('the dismiss layer closes on any tap, and the tap does nothing else', async ({ page }) => {
    const problems = watchHygiene(page);
    await page.goto(URL);
    const open = page.getByTestId('tip-dismiss-open');
    const panel = page.getByTestId('tip-dismiss-panel');
    await open.scrollIntoViewIfNeeded();
    await tap(page, open);
    await expect(panel).toBeVisible();
    // A tap on the very button that opened it: the panel closes, and the
    // swallowed tap does not open it again.
    await tap(page, open);
    await expect(panel).toHaveCount(0);
    await page.waitForTimeout(300);
    await expect(panel).toHaveCount(0);
    // The button itself still works.
    await tap(page, open);
    await expect(panel).toBeVisible();
    expect(problems).toEqual([]);
  });
});
