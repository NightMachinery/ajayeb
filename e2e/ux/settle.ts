import { expect, type Page } from '@playwright/test';
import { quiet } from '../fixtures/layout-probes';
import type { Screen } from './screens';

/** Wait for the represented screen to be painted, before any probe runs. */
export async function settleCapture(page: Page, screen: Screen): Promise<void> {
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
  await page.waitForFunction(
    () => {
      const modal = document.querySelector('dialog:modal');
      return [...document.images].every((image) => {
        if ((modal && !modal.contains(image)) || !image.checkVisibility()) return true;
        const r = image.getBoundingClientRect();
        return (
          r.bottom <= 0 ||
          r.top >= innerHeight ||
          r.right <= 0 ||
          r.left >= innerWidth ||
          image.complete
        );
      });
    },
    null,
    { timeout: 15_000 }
  );
  await page.evaluate(async () => {
    const modal = document.querySelector('dialog:modal');
    await Promise.all(
      [...document.images]
        .filter((image) => {
          if ((modal && !modal.contains(image)) || !image.checkVisibility()) return false;
          const r = image.getBoundingClientRect();
          return r.bottom > 0 && r.top < innerHeight && r.right > 0 && r.left < innerWidth;
        })
        .map((image) => image.decode())
    );
  });
  await quiet(page);
  if (tooltip) await expect(page.locator('#ajayeb-tooltip')).toBeVisible();
}
