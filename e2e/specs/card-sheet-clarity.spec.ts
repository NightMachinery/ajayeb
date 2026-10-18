// Card-sheet details remain readable beside the reserved action footer.
import { expect } from '@playwright/test';
import { test } from '../fixtures/base';
import { GameDriver } from '../fixtures/driver';
import { currentView } from '../fixtures/scenario';
import { createDevRoom } from '../fixtures/ws-game';
import { openTutorialRoom, snapshot, tutorialScript, walkTutorial } from '../fixtures/tutorial';
import { reachScreen } from '../ux/screens';
import { settleCapture } from '../ux/settle';

for (const locale of ['en', 'fa'] as const) {
  test.describe(`card-sheet clarity in ${locale}`, { tag: '@mobile' }, () => {
    test.use({ appLocale: locale });
    test.beforeEach(async ({ page }) => {
      await page.addInitScript(() => {
        localStorage.setItem('ajayeb:name', 'Player');
        localStorage.setItem('ajayeb:afterPickOpen', 'true');
      });
    });

    test('short landscape shows one cost and shortfall beside the face; the fade follows scrolling', async ({
      page,
      baseURL,
      isMobile
    }) => {
      if (!baseURL) throw new Error('no baseURL');
      await page.setViewportSize({ width: 740, height: 360 });
      await reachScreen(page, baseURL, 'card-unaffordable');
      await settleCapture(page, 'card-unaffordable');
      const sheet = page.locator('dialog[open] [data-card-sheet]');
      await expect(sheet.locator('[data-cost-breakdown]')).toHaveCount(1);
      await expect(sheet.locator('[data-reason="unaffordable"]')).toHaveCount(1);
      await expect(sheet.locator('p[data-reason]')).toHaveCount(0);
      await expect(sheet.locator('[data-hint-note="short-by-discard"]')).toBeVisible();
      expect((await sheet.boundingBox())!.width).toBeGreaterThan(600);
      const geometry = await sheet.evaluate((el) => {
        const body = el.querySelector('[data-sheet-body]')!.getBoundingClientRect();
        const face = el.querySelector('.card-preview')!.getBoundingClientRect();
        const cost = el.querySelector('[data-cost-breakdown]')!.getBoundingClientRect();
        return {
          costVisible: cost.top >= body.top && cost.bottom <= body.bottom - 20,
          beside: cost.x >= face.right || cost.right <= face.x
        };
      });
      expect(geometry).toEqual({ costVisible: true, beside: true });
      const body = sheet.locator('[data-sheet-body]');
      await expect(body).toHaveAttribute('data-scroll-more', '');
      await body.focus();
      await page.keyboard.press('End');
      await expect(body).not.toHaveAttribute('data-scroll-more');
      await expect
        .poll(() => body.evaluate((el) => el.scrollHeight - el.clientHeight - el.scrollTop))
        .toBeLessThanOrEqual(1);
      // Let native keyboard scrolling settle before reversing direction.
      await page.waitForTimeout(150);
      // Desktop tests native PageUp; mobile verifies the cue on an explicit scroll.
      if (isMobile)
        await body.evaluate((el) => {
          el.scrollTop = 0;
        });
      else await body.press('PageUp');
      await expect.poll(() => body.evaluate((el) => el.scrollTop)).toBe(0);
      await expect(body).toHaveAttribute('data-scroll-more', '');
      const notes = sheet.locator('[data-hint-notes="discard"] li');
      for (const note of await notes.all())
        await expect(note).toHaveText(
          locale === 'en' ? /^If you discard it,/ : /^اگر آن را کنار بگذارید،/
        );
    });

    test('short landscape keeps a real paid total clear of the outcome footer', async ({
      page,
      baseURL
    }) => {
      if (!baseURL) throw new Error('no baseURL');
      await page.setViewportSize({ width: 740, height: 360 });
      const room = await createDevRoom(baseURL, {
        mode: 'hotseat',
        seed: '7',
        scenario: 'mausoleum'
      });
      await page.goto(room.path);
      const driver = new GameDriver(page);
      await driver.waitReady();
      await driver.openCard('rostrum');
      await settleCapture(page, 'card-buildable');
      const sheet = page.locator('dialog[open] [data-card-sheet]');
      expect((await sheet.boundingBox())!.width).toBeGreaterThan(600);
      await expect(sheet.locator('[data-outcome]')).toHaveCount(2);
      expect(
        await sheet.evaluate((el) => {
          const body = el.querySelector('[data-sheet-body]')!.getBoundingClientRect();
          const cost = el.querySelector('[data-cost-breakdown]')!.getBoundingClientRect();
          return cost.top >= body.top && cost.bottom <= body.bottom - 20;
        })
      ).toBe(true);
    });

    test('the expanded desktop disclosure fits or has a scroll-edge cue', async ({
      page,
      baseURL
    }) => {
      if (!baseURL) throw new Error('no baseURL');
      await page.setViewportSize({ width: 1280, height: 800 });
      await reachScreen(page, baseURL, 'card-buildable');
      await settleCapture(page, 'card-buildable');
      const body = page.locator('dialog[open] [data-sheet-body]');
      await expect(body.locator('[data-after-pick]')).toHaveAttribute('open', '');
      expect(
        await body.evaluate((el) => {
          const bottom = el.getBoundingClientRect().bottom;
          const line = el.querySelector('[data-after-line]')!.getBoundingClientRect();
          return line.bottom <= bottom || el.hasAttribute('data-scroll-more');
        })
      ).toBe(true);
      await body.evaluate((el) => {
        el.scrollTop = el.scrollHeight;
      });
      await expect(body).not.toHaveAttribute('data-scroll-more');
    });

    test('Age I tutorial free play has no Mausoleum advice when the coach is absent', async ({
      page,
      baseURL,
      isMobile
    }) => {
      if (!baseURL) throw new Error('no baseURL');
      const script = await tutorialScript(baseURL);
      await openTutorialRoom(page, baseURL, script.keys);
      await walkTutorial(page, { touch: isMobile, until: 'playOn' });
      const state = await snapshot(page);
      expect(state.tutorial?.status).toBe('free');
      expect(state.bubble).toBeNull();
      expect(state.note).toBeNull();
      const view = await currentView(page);
      expect(view.age).toBe(1);
      expect(
        view.players.some((p) => p.wonders.some((w) => w.wonder === 'mausoleum' && !w.built))
      ).toBe(true);
      await new GameDriver(page).openCard('workshop');
      await expect(page.locator('dialog[open] [data-hint-notes="discard"]')).toHaveCount(0);
    });
  });
}
