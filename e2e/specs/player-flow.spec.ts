// M11.player-flow: returning players, first-time tips and native dialog focus.
import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { GameDriver } from '../fixtures/driver';
import { FrameLog, joinByLink, newDeviceContext } from '../fixtures/online';

const POPUP = '[data-move-summary]';
const TIP_IDS = [
  'win',
  'pair',
  'denyPair',
  'winOpp',
  'loot',
  'lawLaurel',
  'ghostLoot',
  'hourglass',
  'freeChain',
  'guildBack',
  'track',
  'science'
];

async function configure(page: Page, summary: boolean, tips = false): Promise<void> {
  await page.addInitScript(
    ({ summary, tips, seen }) => {
      const current = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}');
      localStorage.setItem(
        'ajayeb:settings',
        JSON.stringify({
          ...current,
          showOpponentMove: summary,
          confirmMoves: 'never',
          hotseatPassDevice: false
        })
      );
      localStorage.setItem(
        'ajayeb:tips',
        JSON.stringify({ off: !tips, seen, tutorial: { offer: 'declined' } })
      );
    },
    { summary, tips, seen: TIP_IDS }
  );
}

async function discard(driver: GameDriver): Promise<void> {
  const snap = await driver.nextTurn();
  const choice = snap.choices.find((c) => c.kind === 'discard');
  if (!choice) throw Error('no discard in scenario');
  await driver.perform({ flow: 'discard', choice }, snap);
  await driver.waitAdvance(snap.version);
}

// Exercise a hidden page which still receives WebSocket frames, unlike a
// frozen page. The visibility event itself is the real app's wake entry point.
async function visible(page: Page, show: boolean): Promise<void> {
  await page.evaluate((show) => {
    Object.defineProperty(document, 'visibilityState', {
      configurable: true,
      get: () => (show ? 'visible' : 'hidden')
    });
    document.dispatchEvent(new Event('visibilitychange'));
  }, show);
}

test.describe('returning player', { tag: ['@mobile'] }, () => {
  test('unseen online moves appear after reconnect, reload and wake; seen moves stay closed', async ({
    browser
  }, info) => {
    const a = await newDeviceContext(browser, info);
    const b = await newDeviceContext(browser, info);
    try {
      const pageA = await a.newPage();
      const pageB = await b.newPage();
      const frames = new FrameLog(pageA, 'returning');
      await configure(pageA, true);
      await configure(pageB, false);
      await pageA.addInitScript(() => localStorage.setItem('ajayeb:name', 'First'));
      await pageA.goto('/dev/new?mode=online&seed=7&scenario=circus-maximus');
      await pageA.waitForURL(/\/play\/[A-Z0-9]+$/);
      await joinByLink(pageB, pageA.url(), 'Second');
      const da = new GameDriver(pageA),
        db = new GameDriver(pageB);
      await da.waitReady();
      await db.waitReady();
      await discard(da);
      await a.setOffline(true);
      await expect(pageA.locator('[data-turn]').filter({ visible: true }).first()).toHaveAttribute(
        'data-turn',
        /^(reconnecting|offline)$/
      );
      await discard(db);
      await a.setOffline(false);
      await expect(pageA.locator(POPUP)).toBeVisible();
      await expect(pageA.locator('[data-summary-away]')).toHaveText('While you were away');
      await pageA.keyboard.press('Escape');
      await pageA.reload();
      await da.waitReady();
      await expect(pageA.locator(POPUP)).toHaveCount(0);

      // Incoming hidden frames must not move the stored seen marker.
      await discard(da);
      await visible(pageA, false);
      await discard(db);
      const latest = await db.snapshot();
      await expect.poll(async () => (await da.snapshot()).ply).toBe(latest.ply);
      await pageA.reload();
      await da.waitReady();
      await expect(pageA.locator(POPUP)).toBeVisible();
      await expect(pageA.locator('[data-summary-away]')).toBeVisible();
      await pageA.keyboard.press('Escape');

      // Waking without reconnect also opens the unseen received move.
      await discard(da);
      await visible(pageA, false);
      await discard(db);
      await expect.poll(async () => (await da.snapshot()).ply).toBe((await db.snapshot()).ply);
      await expect(pageA.locator(POPUP)).toHaveCount(0);
      // Hold presentation bookkeeping until a presence-only frame arrives.
      // This makes the same-ply race deterministic, before the seen-ply rAF.
      await pageA.evaluate(() => {
        const request = window.requestAnimationFrame;
        const cancel = window.cancelAnimationFrame;
        const pending = new Map<number, FrameRequestCallback>();
        let next = -1;
        window.requestAnimationFrame = (callback) => {
          const id = next--;
          pending.set(id, callback);
          return id;
        };
        window.cancelAnimationFrame = (id) => {
          if (id < 0) pending.delete(id);
          else cancel.call(window, id);
        };
        window.addEventListener(
          'ajayeb-test-resume-frames',
          () => {
            window.requestAnimationFrame = request;
            window.cancelAnimationFrame = cancel;
            for (const callback of pending.values()) request.call(window, callback);
          },
          { once: true }
        );
      });
      try {
        await visible(pageA, true);
        await expect(pageA.locator('[data-summary-away]')).toBeVisible();
        const ply = (await da.snapshot()).ply;
        await b.setOffline(true);
        await pageA.waitForFunction(() => window.__ajayeb?.room()?.seats[1]?.connected === false);
        const presence = JSON.parse(frames.frames.filter((f) => f.version !== null).at(-1)!.text);
        expect(presence.events).toEqual([]);
        expect(presence.view.ply).toBe(ply);
        await expect(pageA.locator('[data-summary-away]')).toHaveText('While you were away');
      } finally {
        await pageA.evaluate(() => window.dispatchEvent(new Event('ajayeb-test-resume-frames')));
        await b.setOffline(false);
      }
      await pageA.keyboard.press('Escape');
      await visible(pageA, false);
      await visible(pageA, true);
      await expect(pageA.locator(POPUP)).toHaveCount(0);

      // The user's display preference applies to returning frames as well.
      await pageA.locator('[data-control="menu"]').filter({ visible: true }).click();
      await pageA.locator('[data-control="menu-settings"]').filter({ visible: true }).click();
      await pageA.locator('dialog[open] [data-setting="show-opponent-move"]').setChecked(false);
      await pageA.keyboard.press('Escape');
      await discard(da);
      await a.setOffline(true);
      await expect(pageA.locator('[data-turn]').filter({ visible: true }).first()).toHaveAttribute(
        'data-turn',
        /^(reconnecting|offline)$/
      );
      await discard(db);
      await a.setOffline(false);
      await da.waitReady();
      await expect.poll(async () => (await da.snapshot()).ply).toBe((await db.snapshot()).ply);
      await expect(pageA.locator(POPUP)).toHaveCount(0);
      await pageA.locator('[data-control="move-summary"]').filter({ visible: true }).click();
      await expect(pageA.locator(POPUP)).toBeVisible();
    } finally {
      await a.setOffline(false);
      await a.close();
      await b.close();
    }
  });
});

test.describe('touch flow', { tag: ['@mobile'] }, () => {
  test.use({ viewport: { width: 390, height: 844 }, hasTouch: true, isMobile: true });
  test('a first-time tip preserves the intended card tap, and the new-age tip expires before Age III', async ({
    page
  }) => {
    await configure(page, false, true);
    await page.goto('/dev/new?mode=hotseat&seed=7&scenario=circus-maximus');
    await page.waitForURL(/\/play\/[A-Z0-9]+$/);
    await expect(page.locator('[data-first-tip="newAge"]')).toBeVisible();
    await page.locator('.card-structure [data-card="rostrum"]').tap();
    await expect(page.locator('[data-first-tip]')).toHaveCount(0);
    await expect(page.locator('dialog[open] [data-card-sheet="rostrum"]')).toBeVisible();
    // A fresh context would know none of the new age, yet its moment is past.
    await page.goto('/dev/new?mode=hotseat&seed=7&scenario=science-supremacy');
    await page.waitForURL(/\/play\/[A-Z0-9]+$/);
    await page.waitForFunction(() => window.__ajayeb?.ready());
    await expect(page.locator('[data-first-tip="newAge"]')).toHaveCount(0);
  });

  test('freeChain does not cover its card on touch, and concept tips do not replay late in a game', async ({
    page
  }) => {
    await configure(page, false, true);
    await page.addInitScript(() => {
      localStorage.setItem(
        'ajayeb:tips',
        JSON.stringify({ off: false, seen: ['pair'], tutorial: { offer: 'declined' } })
      );
    });
    await page.goto('/dev/new?mode=hotseat&seed=7&scenario=science-pair');
    await page.waitForURL(/\/play\/[A-Z0-9]+$/);
    const panel = page.locator('[data-dismiss-layer]').filter({
      has: page.locator('[data-first-tip="freeChain"]')
    });
    const card = page.locator('.card-structure [data-card="library"]');
    await expect(panel).toBeVisible();
    await expect(card).toBeVisible();
    const [panelBox, cardBox] = await Promise.all([panel.boundingBox(), card.boundingBox()]);
    expect(panelBox).not.toBeNull();
    expect(cardBox).not.toBeNull();
    if (panelBox && cardBox) {
      const disjointVertically =
        panelBox.y >= cardBox.y + cardBox.height - 1 ||
        panelBox.y + panelBox.height <= cardBox.y + 1;
      expect(disjointVertically, 'freeChain bubble must not cover the card face').toBe(true);
    }
    await card.tap();
    await expect(page.locator('[data-first-tip]')).toHaveCount(0);
    await expect(page.locator('dialog[open] [data-card-sheet="library"]')).toBeVisible();

    // At 5/6 science symbols in Age III, the first-science-symbol and pawn-left-centre
    // concept tips do not replay out of context.
    await page.evaluate(() => {
      localStorage.setItem(
        'ajayeb:tips',
        JSON.stringify({
          off: false,
          seen: ['win', 'guildBack'],
          tutorial: { offer: 'declined' }
        })
      );
    });
    await page.goto('/dev/new?mode=hotseat&seed=7&scenario=science-supremacy');
    await page.waitForURL(/\/play\/[A-Z0-9]+$/);
    await page.waitForFunction(() => window.__ajayeb?.ready());
    await expect(page.locator('[data-first-tip="science"]')).toHaveCount(0);
    await expect(page.locator('[data-first-tip="track"]')).toHaveCount(0);
  });

  test('Escape restores native-sheet focus quietly, the next touch acts and keyboard tips still work', async ({
    page
  }) => {
    await page.goto('/');
    const settings = page.getByRole('button', { name: 'Settings', exact: true });
    await settings.click();
    await expect(page.locator('dialog[open]')).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(page.locator('dialog[open]')).toHaveCount(0);
    await expect(settings).toBeFocused();
    await expect(page.locator('[data-tooltip-layer] [role="tooltip"]')).toHaveCount(0);
    // A deliberate new keyboard focus still gets a tooltip.
    await page.keyboard.press('Tab');
    await page.keyboard.press('Shift+Tab');
    await expect(settings).toBeFocused();
    await expect(page.locator('[data-tooltip-layer] [role="tooltip"]')).toBeVisible();
    await page.keyboard.press('Escape');
    await page.locator('[data-control="home-tutorial"]').tap();
    await page.waitForURL(/\/play\/[A-Z0-9]+$/);
    await expect(page.locator('[data-coach]')).toBeVisible();
  });
});
