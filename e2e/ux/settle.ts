import { expect, type Page } from '@playwright/test';
import { quiet } from '../fixtures/layout-probes';

/** Wait for the represented screen to be painted, before any probe runs. */
export async function settleCapture(page: Page, screen: string): Promise<void> {
  const tooltip = screen.startsWith('tooltip-');
  if (!tooltip) {
    await page.mouse.move(0, 0);
    await expect(page.locator('[data-tooltip-open="hover"]')).toBeHidden();
  }
  // Keep the tutorial's intended target focus, and intentional tooltip focus.
  if (!tooltip && !screen.startsWith('tutorial-')) {
    await page.evaluate(() => (document.activeElement as HTMLElement | null)?.blur());
    await expect(page.locator('#ajayeb-tooltip')).toBeHidden();
  }
  await page.evaluate(() => document.fonts.ready.then(() => undefined));
  // Force style resolution before checking running entry transitions. A visible
  // dialog can still have opacity 0 from @starting-style, even with Animation off.
  await page.waitForFunction(() => {
    const modal = document.querySelector('dialog:modal');
    return (
      !modal ||
      (modal.checkVisibility({ checkOpacity: true, checkVisibilityCSS: true }) &&
        getComputedStyle(modal).opacity === '1')
    );
  });
  await quiet(page);

  const required =
    screen === 'settings'
      ? '[data-setting="language"] label'
      : screen === 'join'
        ? '[data-code-input]'
        : screen === 'tutorial-card-sheet'
          ? '[data-coach-note][data-coach-step="buildFree"]'
          : screen === 'tutorial-outro'
            ? '[data-coach][data-coach-step="tokens"][data-placed]'
            : null;
  if (required) {
    await page.waitForFunction((selector) => {
      const el = document.querySelector(selector);
      return el?.checkVisibility({ checkOpacity: true, checkVisibilityCSS: true });
    }, required);
  }
  if (screen === 'tutorial-card-sheet') {
    await expect(page.locator('dialog[open] [data-coach-target]')).toBeVisible();
    await expect(page.locator('dialog[open] [data-coach-target]')).toBeEnabled();
  }
  await page.waitForFunction(captureImagesReady, false, { timeout: 15_000 });
  await page.evaluate(captureImagesReady, true);
  await quiet(page);
  if (tooltip) await expect(page.locator('#ajayeb-tooltip')).toBeVisible();
}

/** Lazy images clipped out by a sheet's scroll body need not load. A box
 * inside the layout viewport alone is insufficient, notably below a footer. */
async function captureImagesReady(decode: boolean): Promise<boolean> {
  const modal = document.querySelector('dialog:modal');
  const visible = [...document.images].filter((image) => {
    if ((modal && !modal.contains(image)) || !image.checkVisibility()) return false;
    const r = image.getBoundingClientRect();
    let left = Math.max(0, r.left),
      right = Math.min(innerWidth, r.right);
    let top = Math.max(0, r.top),
      bottom = Math.min(innerHeight, r.bottom);
    for (let parent = image.parentElement; parent; parent = parent.parentElement) {
      const css = getComputedStyle(parent),
        box = parent.getBoundingClientRect();
      if (css.overflowX !== 'visible') {
        left = Math.max(left, box.left);
        right = Math.min(right, box.right);
      }
      if (css.overflowY !== 'visible') {
        top = Math.max(top, box.top);
        bottom = Math.min(bottom, box.bottom);
      }
    }
    return right > left && bottom > top;
  });
  if (decode) await Promise.all(visible.map((image) => image.decode()));
  return visible.every((image) => image.complete);
}
