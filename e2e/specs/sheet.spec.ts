// Swipe down to dismiss a bottom sheet (web/src/lib/components/shell/sheet-drag.ts).
// The owner found that pulling the card sheet down reloaded the page instead
// of closing the sheet. The swipe is played with synthetic touch events on
// the sheet's panel, since Playwright has no touch drag of its own; the
// listener only reads clientY and timeStamp, which these carry.
import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { openHotseatRoom } from '../fixtures/scenario';

async function swipe(page: Page, distance: number, steps: number): Promise<void> {
  await page.evaluate(
    ({ distance, steps }) => {
      const panel = document.querySelector<HTMLElement>('dialog[open] .sheet-panel');
      if (!panel) throw new Error('no open sheet panel');
      const box = panel.getBoundingClientRect();
      const x = box.left + box.width / 2;
      const y0 = box.top + 40;
      const fire = (type: string, y: number, ended = false): void => {
        const touch = new Touch({ identifier: 1, target: panel, clientX: x, clientY: y });
        panel.dispatchEvent(
          new TouchEvent(type, {
            bubbles: true,
            cancelable: true,
            touches: ended ? [] : [touch],
            targetTouches: ended ? [] : [touch],
            changedTouches: [touch]
          })
        );
      };
      fire('touchstart', y0);
      for (let i = 1; i <= steps; i++) fire('touchmove', y0 + (distance * i) / steps);
      fire('touchend', y0 + distance, true);
    },
    { distance, steps }
  );
}

test.describe('bottom sheet', { tag: ['@mobile'] }, () => {
  test('a long swipe down closes the card sheet; a short one does not', async ({ page }, info) => {
    test.skip(info.project.name !== 'mobile', 'the swipe exists on the phone layout only');
    const problems = watchHygiene(page);
    await openHotseatRoom(page, 'circus-maximus');
    const card = page.locator('.card-structure [data-card][data-accessible]').first();
    const sheet = page.locator('dialog[open] [data-card-sheet]');

    await card.click();
    await expect(sheet).toBeVisible();
    await swipe(page, 20, 4);
    await expect(sheet).toBeVisible();

    await swipe(page, 260, 8);
    await expect(page.locator('dialog[open]')).toHaveCount(0);

    // The page was not reloaded: the same room is still on screen.
    await card.click();
    await expect(sheet).toBeVisible();
    expect(problems).toEqual([]);
  });
});
