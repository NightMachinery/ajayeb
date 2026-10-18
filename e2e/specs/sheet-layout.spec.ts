// Geometry regressions for the M11 sheet, coach and strip findings.
import { expect, type Locator, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { currentView } from '../fixtures/scenario';
import { reachScreen } from '../ux/screens';
import { settleCapture } from '../ux/settle';

async function usable(control: Locator): Promise<void> {
  expect(
    await control.evaluate((el) => {
      const r = el.getBoundingClientRect();
      const panel = el.closest('.sheet-panel')!.getBoundingClientRect();
      const hit = document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2);
      return {
        width: r.width,
        height: r.height,
        inside: r.top >= panel.top && r.bottom <= panel.bottom,
        hit: !!hit && el.contains(hit)
      };
    })
  ).toMatchObject({
    inside: true,
    hit: true,
    width: expect.any(Number),
    height: expect.any(Number)
  });
  const r = await control.boundingBox();
  expect(r!.width).toBeGreaterThanOrEqual(44);
  expect(r!.height).toBeGreaterThanOrEqual(44);
}

async function cardControls(page: Page): Promise<void> {
  const controls = page.locator(
    'dialog[open] button[data-act], dialog[open] [data-control="wonder-step"]'
  );
  expect(await controls.count()).toBeGreaterThanOrEqual(2);
  for (const control of await controls.all()) await usable(control);
  const blocked = page.getByRole('button', { name: /^Build for/ }).filter({ visible: true });
  for (const control of await blocked.all()) await usable(control);
}

for (const theme of ['light', 'dark'] as const) {
  test.describe(`sheet layout in ${theme}`, { tag: '@mobile' }, () => {
    test.beforeEach(async ({ page }) => {
      await page.addInitScript((theme) => {
        localStorage.setItem('ajayeb:theme', theme);
        localStorage.setItem('ajayeb:name', 'Player');
        const settings = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}');
        localStorage.setItem(
          'ajayeb:settings',
          JSON.stringify({ ...settings, confirmMoves: 'always', hotseatPassDevice: false })
        );
      }, theme);
    });

    test('short card actions stay visible through scroll, confirmation and wonder steps', async ({
      page,
      baseURL
    }) => {
      if (!baseURL) throw new Error('sheet layout requires local baseURL');
      const problems = watchHygiene(page);
      await page.setViewportSize({ width: 740, height: 360 });
      for (const screen of ['card-buildable', 'card-unaffordable'] as const) {
        await reachScreen(page, baseURL, screen);
        await settleCapture(page, screen);
        await cardControls(page);
        const body = page.locator('dialog[open] [data-sheet-body]');
        await body.focus();
        await page.keyboard.press('End');
        await expect.poll(() => body.evaluate((el) => el.scrollTop)).toBeGreaterThan(0);
        await expect
          .poll(() =>
            body.evaluate((el) => Math.abs(el.scrollHeight - el.clientHeight - el.scrollTop))
          )
          .toBeLessThanOrEqual(1);
        await cardControls(page);
        const act = page.locator('dialog[open] button[data-act]:enabled').first();
        await act.click();
        await expect(act).toHaveAttribute('data-armed', '');
        await cardControls(page);
        await page.locator('dialog[open] [data-control="wonder-step"]').click();
        await expect(page.locator('dialog[open] [data-wonder-step]')).toBeVisible();
        await expect(page.locator('dialog[open] [data-sheet-footer]')).toHaveCount(0);
        const cost = page.locator('dialog[open] [data-wonder-option] details').first();
        await cost.locator('summary').click();
        await expect(cost).toHaveAttribute('open', '');
        await page.locator('dialog[open] [data-control="wonder-back"]').click();
        await cardControls(page);
        await page.mouse.move(0, 0);
        await page.keyboard.press('Escape');
        await expect(page.locator('dialog[open]')).toHaveCount(0);
      }
      // The reverse flow reserves the same action area for one wonder
      // confirmation, with no stale Build/Discard controls from the card.
      await reachScreen(page, baseURL, 'board-age2');
      const view = await currentView(page);
      const wonder = view.choices.find((choice) => choice.kind === 'wonder');
      expect(wonder?.wonder).toBeTruthy();
      await page
        .locator(`.seat-panel[data-seat="${view.toMove}"] [data-control="wonders"]`)
        .filter({ visible: true })
        .click();
      await page.locator(`dialog[open] [data-wonder="${wonder!.wonder}"]`).click();
      await page.locator('dialog[open] [data-control="wonder-build"]').click();
      await page.locator(`.card-structure [data-card="${wonder!.card}"][data-marked]`).click();
      const pick = page.locator(`dialog[open] [data-act="${wonder!.key}"]`);
      await usable(pick);
      await expect(page.locator('dialog[open] button[data-act]')).toHaveCount(1);
      await pick.click();
      await expect(pick).toHaveAttribute('data-armed', '');
      await usable(pick);
      expect(problems).toEqual([]);
    });

    test('choice action covers the bottom strip while every option remains reachable', async ({
      page,
      baseURL
    }) => {
      if (!baseURL) throw new Error('sheet layout requires local baseURL');
      await page.setViewportSize({ width: 740, height: 360 });
      for (const screen of ['choice-great-library', 'choice-progress-token'] as const) {
        await reachScreen(page, baseURL, screen);
        await settleCapture(page, screen);
        const panel = page.locator('dialog[open] .sheet-panel');
        for (const fraction of [0, 0.5, 1]) {
          await panel.evaluate((el, fraction) => {
            el.scrollTop = fraction * (el.scrollHeight - el.clientHeight);
          }, fraction);
          expect(
            await panel.evaluate((el) => {
              const r = el.getBoundingClientRect();
              const commit = el.querySelector('[data-commit]')!;
              return [0.15, 0.5, 0.85].every((fraction) => {
                const hit = document.elementFromPoint(r.x + r.width * fraction, r.bottom - 2);
                return !!hit && commit.contains(hit);
              });
            })
          ).toBe(true);
        }
        const last = page.locator('dialog[open] [data-pick]').last();
        await last.focus();
        await expect(last).toBeFocused();
        await usable(last);
        await page.keyboard.press('Space');
        await expect(last).toHaveAttribute('aria-pressed', 'true');
        const commit = page.locator('dialog[open] [data-commit] button');
        await usable(commit);
        await commit.click();
        await expect(commit).toHaveAttribute('data-armed', '');
        await usable(commit);
      }
    });

    test('phone coach clears the topbar', async ({ page, baseURL }) => {
      if (!baseURL) throw new Error('sheet layout requires local baseURL');
      for (const viewport of [
        { width: 360, height: 740 },
        { width: 390, height: 844 }
      ]) {
        await page.setViewportSize(viewport);
        await reachScreen(page, baseURL, 'tutorial-first');
        await settleCapture(page, 'tutorial-first');
        const bar = await page.locator('.top-bar').boundingBox();
        const coach = await page.locator('[data-coach]').boundingBox();
        expect(coach!.y).toBeGreaterThanOrEqual(bar!.y + bar!.height + 8 - 1);
      }
    });

    test('phone hint pips clear neighboring chips', async ({ page, baseURL }) => {
      if (!baseURL) throw new Error('sheet layout requires local baseURL');
      for (const viewport of [
        { width: 360, height: 740 },
        { width: 390, height: 844 }
      ]) {
        await page.setViewportSize(viewport);
        await reachScreen(page, baseURL, 'board-age2');
        await settleCapture(page, 'board-age2');
        const geometry = await page.evaluate(() => {
          const strips = [...document.querySelectorAll('.player-strip')].filter(
            (el) => el.getBoundingClientRect().width > 0
          );
          let pips = 0;
          const overlaps: string[] = [];
          const intersects = (a: DOMRect, b: DOMRect) =>
            Math.min(a.right, b.right) > Math.max(a.left, b.left) &&
            Math.min(a.bottom, b.bottom) > Math.max(a.top, b.top);
          for (const strip of strips) {
            const sr = strip.getBoundingClientRect();
            for (const chip of strip.querySelectorAll('[data-resource]')) {
              // Structural selector also identifies the original baseline pip.
              const pip = chip.querySelector('span.absolute[aria-hidden="true"]');
              if (!pip) continue;
              pips++;
              const r = pip.getBoundingClientRect();
              // Leave two pixels for the pip shadow beside a chip border.
              const clearance = new DOMRect(r.x - 2, r.y - 2, r.width + 4, r.height + 4);
              for (const neighbor of strip.querySelectorAll('[data-resource]'))
                if (neighbor !== chip && intersects(clearance, neighbor.getBoundingClientRect()))
                  overlaps.push('resource');
            }
            for (const pip of strip.querySelectorAll('[data-ready-dot]')) {
              pips++;
              const r = pip.getBoundingClientRect();
              if (
                r.right > sr.right - 3 ||
                r.left < sr.left + 3 ||
                r.top < sr.top + 3 ||
                r.bottom > sr.bottom - 3
              )
                overlaps.push('wonder');
            }
          }
          return { pips, overlaps };
        });
        expect(geometry.pips).toBeGreaterThan(0);
        expect(geometry.overlaps).toEqual([]);
      }
    });

    test('a pinned tutorial action keeps its coach note in the scrolling body', async ({
      page,
      baseURL
    }) => {
      if (!baseURL) throw new Error('sheet layout requires local baseURL');
      await page.setViewportSize({ width: 740, height: 360 });
      await reachScreen(page, baseURL, 'tutorial-card-sheet');
      await settleCapture(page, 'tutorial-card-sheet');
      const geometry = await page.locator('dialog[open] [data-coach-note]').evaluate((note) => {
        const r = note.getBoundingClientRect();
        const body = note.closest('[data-sheet-body]')!.getBoundingClientRect();
        return { inside: r.top >= body.top && r.bottom <= body.bottom, detailHeight: body.height };
      });
      expect(geometry.inside).toBe(true);
      expect(geometry.detailHeight).toBeGreaterThanOrEqual(44);
      await usable(page.locator('dialog[open] [data-coach-target]'));
    });

    test('a scrolled phone detail body scrolls before a downward drag dismisses it', async ({
      page,
      baseURL
    }) => {
      if (!baseURL) throw new Error('sheet layout requires local baseURL');
      await page.setViewportSize({ width: 360, height: 740 });
      await reachScreen(page, baseURL, 'card-unaffordable');
      await settleCapture(page, 'card-unaffordable');
      const body = page.locator('dialog[open] [data-sheet-body]');
      await body.evaluate((el) => {
        el.scrollTop = el.scrollHeight;
      });
      expect(await body.evaluate((el) => el.scrollTop)).toBeGreaterThan(0);
      const drag = async () => {
        await body.evaluate((el) => {
          const target = el.firstElementChild!;
          const r = el.getBoundingClientRect();
          const fire = (type: string, dy: number, ended = false) => {
            const touch = new Touch({
              identifier: 1,
              target,
              clientX: r.x + 30,
              clientY: r.y + 20 + dy
            });
            target.dispatchEvent(
              new TouchEvent(type, {
                bubbles: true,
                cancelable: true,
                touches: ended ? [] : [touch],
                changedTouches: [touch]
              })
            );
          };
          fire('touchstart', 0);
          fire('touchmove', 180);
          fire('touchend', 180, true);
        });
      };
      await drag();
      await expect(page.locator('dialog[open]')).toHaveCount(1);
      await body.evaluate((el) => {
        el.scrollTop = 0;
      });
      await drag();
      await expect(page.locator('dialog[open]')).toHaveCount(0);
    });
  });
}
