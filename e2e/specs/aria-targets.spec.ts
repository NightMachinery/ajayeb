// Interaction regressions for the M11 screen review's semantics and targets.
import AxeBuilder from '@axe-core/playwright';
import { expect, type Locator } from '@playwright/test';
import { test } from '../fixtures/base';
import { GameDriver } from '../fixtures/driver';
import { reachScreen } from '../ux/screens';
import { settleCapture } from '../ux/settle';

async function targetSize(control: Locator): Promise<void> {
  const box = await control.boundingBox();
  expect(box).not.toBeNull();
  expect(box!.width).toBeGreaterThanOrEqual(44);
  expect(box!.height).toBeGreaterThanOrEqual(44);
}

for (const theme of ['light', 'dark'] as const) {
  test.describe(`accessible controls in ${theme}`, () => {
    test.beforeEach(async ({ page }) => {
      await page.addInitScript((theme) => {
        localStorage.setItem('ajayeb:theme', theme);
        localStorage.setItem('ajayeb:name', 'Player');
      }, theme);
    });

    test('city bands have a valid name and waiting draft scrolls with keys', async ({
      page,
      baseURL
    }) => {
      if (!baseURL) throw new Error('aria targets requires local baseURL');
      await page.setViewportSize({ width: 1280, height: 800 });
      await reachScreen(page, baseURL, 'board-age1');
      const bands = page
        .locator('.city-panel')
        .filter({ visible: true })
        .locator('[role="group"][data-fx-city]');
      await expect(bands).toHaveCount(2);
      for (const band of await bands.all()) {
        await expect(band).toHaveRole('group');
        await expect(band).toHaveAccessibleName('City');
      }
      expect(
        (await new AxeBuilder({ page }).withRules(['aria-prohibited-attr']).analyze()).violations
      ).toEqual([]);

      await page.setViewportSize({ width: 740, height: 360 });
      await reachScreen(page, baseURL, 'wonder-draft');
      const draft = page.locator('[data-draft][data-round]');
      await expect(page.locator('[data-draft] [data-act]')).toHaveCount(0);
      await expect(draft).toHaveAccessibleName('Wonder draft');
      await page.locator('[data-control="menu"]').focus();
      // Reach the region through the real tab order, without a pointer.
      for (let i = 0; i < 15 && !(await draft.evaluate((el) => el === document.activeElement)); i++)
        await page.keyboard.press('Tab');
      await expect(draft).toBeFocused();
      expect(await draft.evaluate((el) => el.scrollHeight > el.clientHeight)).toBe(true);
      await page.keyboard.press('End');
      await expect.poll(() => draft.evaluate((el) => el.scrollTop)).toBeGreaterThan(0);
      expect(
        (await new AxeBuilder({ page }).withRules(['scrollable-region-focusable']).analyze())
          .violations
      ).toEqual([]);
    });

    for (const viewport of [
      { width: 360, height: 740 },
      { width: 390, height: 844 },
      { width: 740, height: 360 }
    ]) {
      test(`strip and disclosure targets at ${viewport.width}x${viewport.height}`, async ({
        page,
        baseURL
      }) => {
        if (!baseURL) throw new Error('aria targets requires local baseURL');
        await page.setViewportSize(viewport);
        await reachScreen(page, baseURL, 'board-age1');
        for (const selector of ['open-city', 'wonders', 'device-link']) {
          const controls = page.locator(`[data-control="${selector}"]`).filter({ visible: true });
          expect(await controls.count()).toBeGreaterThan(0);
          for (const control of await controls.all()) await targetSize(control);
        }
        await reachScreen(page, baseURL, 'board-age3');
        const names = page
          .locator('.player-strip [data-control="open-city"] [dir="auto"]')
          .filter({ visible: true });
        for (const name of await names.all()) {
          expect(await name.evaluate((el) => el.scrollWidth - el.clientWidth)).toBeLessThanOrEqual(
            0
          );
        }
        await reachScreen(page, baseURL, 'card-buildable');
        await settleCapture(page, 'card-buildable');
        const card = await page
          .locator('dialog[open] [data-card-sheet]')
          .getAttribute('data-card-sheet');
        if (!card) throw new Error('missing card sheet');
        const panel = page.locator('dialog[open] [data-after-pick]');
        const summary = panel.locator('summary');
        await targetSize(summary);
        const wasOpen = await panel.evaluate((el) => (el as HTMLDetailsElement).open);
        const box = (await summary.boundingBox())!;
        // The padding near the far edge used to look clickable but did nothing.
        await summary.click({ position: { x: box.width - 8, y: 8 } });
        await expect(panel).toHaveJSProperty('open', !wasOpen);
        await summary.focus();
        await page.keyboard.press('Enter');
        await expect(panel).toHaveJSProperty('open', wasOpen);
        await page.keyboard.press('Space');
        await expect(panel).toHaveJSProperty('open', !wasOpen);
        expect(await page.evaluate(() => localStorage.getItem('ajayeb:afterPickOpen'))).toBe(
          String(!wasOpen)
        );
        await page.reload();
        await new GameDriver(page).openCard(card);
        await expect(page.locator('dialog[open] [data-after-pick]')).toHaveJSProperty(
          'open',
          !wasOpen
        );
        await reachScreen(page, baseURL, 'wonder-flow');
        for (const cost of await page.locator('[data-wonder-step] summary').all())
          await targetSize(cost);
      });
    }

    test('native controls theme unchecked states and keep keyboard selection', async ({
      page,
      baseURL
    }, info) => {
      if (!baseURL) throw new Error('aria targets requires local baseURL');
      await page.setViewportSize({ width: 390, height: 844 });
      await reachScreen(page, baseURL, 'settings');
      for (const label of await page.locator('dialog[open] [role="radiogroup"] label').all())
        await targetSize(label);
      const checkbox = page.locator('[data-setting="show-opponent-move"]');
      await expect(checkbox).not.toBeChecked();
      const unchecked = await checkbox.evaluate((el) => {
        const surface = document.createElement('span');
        surface.style.background = 'var(--panel-sunken)';
        document.body.append(surface);
        const expected = getComputedStyle(surface).backgroundColor;
        surface.remove();
        return {
          appearance: getComputedStyle(el).appearance,
          background: getComputedStyle(el).backgroundColor,
          expected
        };
      });
      expect(unchecked.appearance).toBe('none');
      expect(unchecked.background).toBe(unchecked.expected);
      await checkbox.focus();
      await page.screenshot({ path: info.outputPath('unchecked-checkbox.png') });
      await page.keyboard.press('Space');
      await expect(checkbox).toBeChecked();
      expect(await checkbox.evaluate((el) => getComputedStyle(el).backgroundColor)).not.toBe(
        unchecked.background
      );
      await page.keyboard.press('Escape');
      await reachScreen(page, baseURL, 'new-game');
      const radios = page.locator('dialog[open] input[name="ai-level"]');
      const rival = page.locator('dialog[open] input[name="ai-level"]:not(:checked)').first();
      const radioStyle = await rival.evaluate((el) => ({
        appearance: getComputedStyle(el).appearance,
        background: getComputedStyle(el).backgroundColor
      }));
      expect(radioStyle.appearance).toBe('none');
      expect(radioStyle.background).toBe(unchecked.background);
      await radios.first().focus();
      await page.screenshot({ path: info.outputPath('native-radios.png') });
      await page.keyboard.press('ArrowDown');
      await expect(radios.nth(1)).toBeChecked();
      await page.emulateMedia({ forcedColors: 'active' });
      expect(await radios.first().evaluate((el) => getComputedStyle(el).appearance)).toBe('auto');
      await expect(radios.nth(1)).toBeChecked();
      for (const label of await page.locator('dialog[open] [role="radiogroup"] label').all())
        await targetSize(label);
    });
  });
}
