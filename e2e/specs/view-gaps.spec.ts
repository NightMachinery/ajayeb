// The server's payment/undo/discard projections through the real UI.
import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { currentView, logLines, openHotseatRoom, playKey } from '../fixtures/scenario';

async function recordedEffects(page: Page) {
  return page.evaluate(() => {
    const fx = (window as unknown as { __ajayebFx?: { cues(): unknown[]; haptics(): number[][] } })
      .__ajayebFx;
    return { cues: fx?.cues() ?? [], haptics: fx?.haptics() ?? [] };
  });
}

test.describe('view gaps', { tag: ['@mobile'] }, () => {
  test('the disabled Undo control explains the server reason on pointer, focus and touch', async ({
    page
  }, testInfo) => {
    const problems = watchHygiene(page);
    const driver = await openHotseatRoom(page, null, 4343);
    await driver.visible('[data-control="menu"]').click();
    const item = page.locator('[data-game-menu]:popover-open [data-control="menu-undo"]');
    const anchor = page.locator('[data-game-menu]:popover-open [data-undo-tooltip]');
    await expect(item).toBeDisabled();
    const version = (await driver.snapshot()).version;
    const reason = 'No move of yours to take back yet.';
    await expect(item.locator('[data-undo-reason]')).toHaveText(reason);
    if (testInfo.project.name === 'mobile') {
      // Playwright normally refuses to target disabled controls; force still
      // sends a real touch gesture, which must explain rather than act.
      await item.tap({ force: true });
    } else {
      await item.hover();
    }
    await expect(page.getByRole('tooltip')).toContainText(reason);
    await page.screenshot({ path: testInfo.outputPath('undo-disabled-tooltip.png') });
    await page.keyboard.press('Escape');
    await expect(page.getByRole('tooltip')).toHaveCount(0);
    // Native disabled controls cannot focus. The tooltip's focusable wrapper
    // gives keyboard users the same reason while preserving disabled semantics.
    await page.keyboard.press('Tab');
    await anchor.focus();
    await expect(page.getByRole('tooltip')).toContainText(reason);
    await expect(anchor).toHaveAttribute('aria-describedby', 'ajayeb-tooltip');
    await page.keyboard.press('Escape');
    await page.keyboard.press('Escape');
    await expect(page.locator('[data-game-menu]:popover-open')).toHaveCount(0);
    expect((await driver.snapshot()).version).toBe(version);
    const view = await currentView(page);
    await playKey(driver, view.choices[0]!.key);
    await driver.visible('[data-control="menu"]').click();
    await expect(
      page.locator('[data-game-menu]:popover-open [data-control="menu-undo"]')
    ).toBeEnabled();
    await expect(page.locator('[data-game-menu]:popover-open [data-undo-reason]')).toHaveCount(0);
    expect(problems).toEqual([]);
  });

  test('reload, resync, undo and rejoin restore the whole log without replay cues', async ({
    page
  }) => {
    const problems = watchHygiene(page);
    let historyFrames = 0;
    page.on('websocket', (socket) =>
      socket.on('framereceived', ({ payload }) => {
        const frame = JSON.parse(payload.toString()) as { t?: string; history?: unknown[] };
        if (frame.t === 'state' && frame.history) historyFrames++;
      })
    );
    await page.addInitScript(() => {
      const Native = window.WebSocket;
      const sockets: WebSocket[] = [];
      window.WebSocket = class extends Native {
        constructor(url: string | URL, protocols?: string | string[]) {
          super(url, protocols);
          sockets.push(this);
        }
      };
      Object.defineProperty(window, '__viewGapsSend', {
        value: (frame: string) => {
          const socket = sockets.find((s) => s.readyState === Native.OPEN);
          if (!socket) throw new Error('no live game socket');
          socket.send(frame);
        }
      });
    });
    await page.goto('/settings');
    await page.locator('input[data-setting="sound"]').check();
    const driver = await openHotseatRoom(page, null, 4343);
    for (let i = 0; i < 12; i++) {
      const view = await currentView(page);
      await playKey(driver, view.choices.find((c) => c.kind !== 'wonder')!.key);
    }
    const url = page.url();
    const before = await logLines(driver);
    const history = await page.evaluate(() => window.__ajayeb?.history());
    expect(history).toHaveLength(12);
    await page.reload();
    await driver.waitReady();
    expect(await logLines(driver)).toEqual(before);
    expect(await recordedEffects(page)).toEqual({ cues: [], haptics: [] });
    expect(await page.evaluate(() => window.__ajayeb?.history())).toEqual(history);
    const count = historyFrames;
    await page.evaluate(() => {
      const room = window.__ajayeb?.room();
      window.__viewGapsSend?.(JSON.stringify({ t: 'resync', code: room?.code }));
    });
    await expect.poll(() => historyFrames).toBe(count + 1);
    await driver.waitReady();
    expect(await logLines(driver)).toEqual(before);
    expect(await recordedEffects(page)).toEqual({ cues: [], haptics: [] });
    const version = (await driver.snapshot()).version;
    await page.keyboard.press('u');
    await driver.waitAdvance(version);
    expect(await page.evaluate(() => window.__ajayeb?.history())).toEqual(history!.slice(0, -1));
    expect(await recordedEffects(page)).toEqual({ cues: [], haptics: [] });
    const undone = await logLines(driver);
    expect(undone.length).toBeLessThan(before.length);
    await page.goto('/');
    await page.goto(url);
    await driver.waitReady();
    expect(await logLines(driver)).toEqual(undone);
    expect(await recordedEffects(page)).toEqual({ cues: [], haptics: [] });
    expect(problems).toEqual([]);
  });
});

declare global {
  interface Window {
    __viewGapsSend?: (frame: string) => void;
  }
}
