// The card sheet's signals from the owner's phone playtest
// (M18.card-sheet-signals), over live Duel rooms against a bot:
//
// - the Build a wonder button stands out and names the wonder when the
//   view's choices let the card go under one now, and stays plain when
//   they do not;
// - the sheet lists the hint marks the card shows in its corner, with the
//   same lines as the mark's tooltip;
// - on a touch screen, the first time a mark shows, a one-time tip points
//   at it and says to press and hold it; never on a fine pointer, and not
//   again once closed;
// - the rules' hint legend, reached from that tip's "More in Rules".
//
// Each screen is captured light and dark at 390x844 and 1280x800 (the tip
// at 390x844 only), and the tip once more in Persian, into
// CARD_SIGNALS_SHOTS_DIR (or the test's output directory).
import { expect, type Browser, type Page, type TestInfo } from '@playwright/test';
import { installMotionOff, test } from '../fixtures/base';
import { GameDriver } from '../fixtures/driver';
import { currentView } from '../fixtures/scenario';
import { createDevRoom } from '../fixtures/ws-game';
import { settleCapture } from './settle';

const PHONE = { width: 390, height: 844 };
const DESKTOP = { width: 1280, height: 800 };
const HINTS = '[data-rules-section="hints"]';
const TIPS_ON = JSON.stringify({ off: false, seen: [], tutorial: { offer: 'declined' } });

function dir(testInfo: TestInfo): string {
  return process.env.CARD_SIGNALS_SHOTS_DIR ?? testInfo.outputPath();
}

/** Both themes at the given sizes. The theme follows the system here, so
 *  emulating the colour scheme switches it. */
async function shots(
  page: Page,
  testInfo: TestInfo,
  id: string,
  sizes: readonly { width: number; height: number }[] = [PHONE, DESKTOP],
  schemes: readonly ('light' | 'dark')[] = ['light', 'dark'],
  focus?: string
): Promise<void> {
  for (const scheme of schemes) {
    await page.emulateMedia({ colorScheme: scheme });
    for (const size of sizes) {
      await page.setViewportSize(size);
      // A resize moves the layout: bring the subject back to the top, once
      // more after the settle, since late fonts and images can move it.
      const place = async () => {
        if (focus)
          await page
            .locator(focus)
            .filter({ visible: true })
            .first()
            .evaluate((el) => el.scrollIntoView({ block: 'start', behavior: 'instant' }));
      };
      await place();
      await settleCapture(page, 'card-buildable');
      await place();
      await page.screenshot({
        path: `${dir(testInfo)}/${id}-${size.width}x${size.height}-${scheme}.png`
      });
    }
  }
}

/** A live room against a bot, seat 0 to move. */
async function room(page: Page, origin: string, body: Parameters<typeof createDevRoom>[1]) {
  const r = await createDevRoom(origin, body);
  await page.goto(r.path);
  const driver = new GameDriver(page);
  await driver.waitReady();
  return driver;
}

async function touchPage(browser: Browser, testInfo: TestInfo, locale?: string): Promise<Page> {
  const context = await browser.newContext({
    baseURL: testInfo.project.use.baseURL,
    viewport: PHONE,
    isMobile: true,
    hasTouch: true
  });
  await installMotionOff(context);
  await context.addInitScript(
    ({ tips, locale }) => {
      localStorage.setItem('ajayeb:theme', 'system');
      localStorage.setItem('ajayeb:name', 'Player');
      if (!sessionStorage.getItem('signals-tips')) {
        sessionStorage.setItem('signals-tips', '1');
        localStorage.setItem('ajayeb:tips', tips);
      }
      if (locale) {
        const s = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}') as object;
        localStorage.setItem('ajayeb:settings', JSON.stringify({ ...s, locale }));
      }
    },
    { tips: TIPS_ON, locale }
  );
  return context.newPage();
}

test.describe('card sheet signals', () => {
  test.beforeEach(async ({ page }) => {
    await page.addInitScript(() => {
      localStorage.setItem('ajayeb:theme', 'system');
      localStorage.setItem('ajayeb:name', 'Player');
    });
  });

  test('the wonder button and the marks follow the view', async ({ page, baseURL }, testInfo) => {
    test.setTimeout(300_000);
    if (!baseURL) throw new Error('no baseURL');

    // A card that pairs a science symbol, with coins for every wonder.
    const driver = await room(page, baseURL, {
      mode: 'ai',
      bot: 'easy',
      seed: '7',
      scenario: 'science-pair'
    });
    const view = await currentView(page);
    const ready = view.choices.filter((c) => c.kind === 'wonder' && c.card === 'library');
    expect(ready.length).toBeGreaterThan(0);
    await driver.openCard('library');
    const sheet = page.locator('dialog[open] [data-card-sheet="library"]');
    const button = page.locator('dialog[open] [data-control="wonder-step"]');
    await expect(button).toHaveAttribute('data-wonder-ready', ready.map((c) => c.wonder).join(' '));
    // The sheet lists the corner's mark, line for line.
    const lines = sheet.locator('[data-card-marks] [data-hint-line]');
    await expect(lines.first()).toBeVisible();
    await expect(sheet.locator('[data-card-marks] [data-hint-line="pair"]')).toHaveCount(1);
    await shots(page, testInfo, 'sheet-ready');

    // The same mark's tooltip on the board says the same lines.
    await page.keyboard.press('Escape');
    await expect(page.locator('dialog[open] [data-card-sheet]')).toHaveCount(0);
    const pip = page.locator('.card-structure [data-card="library"] [data-hint]');
    await pip.hover();
    const bubble = page.locator('[data-tooltip-layer] [data-tooltip-bubble] [data-hint-lines]');
    await expect(bubble.locator('[data-hint-line="pair"]')).toHaveCount(1);
    await page.mouse.move(0, 0);

    // A card no wonder can take now: the button stays plain.
    const poor = await room(page, baseURL, {
      mode: 'hotseat',
      seed: '7',
      actions: [
        'draft:pyramids',
        'draft:sphinx',
        'draft:colossus',
        'draft:hanging-gardens',
        'draft:appian-way',
        'draft:great-library',
        'construct:wood-reserve',
        'construct:workshop',
        'construct:scriptorium',
        'construct:palisade',
        'construct:baths',
        'construct:clay-pit'
      ]
    });
    const v2 = await currentView(page);
    const plain = v2.structure.find(
      (s) =>
        s.state === 'faceUp' &&
        s.accessible &&
        !v2.choices.some((c) => c.kind === 'wonder' && c.card === s.card) &&
        v2.disabled.some((d) => d.kind === 'wonder' && d.card === s.card)
    )?.card;
    expect(plain, 'a card with only disabled wonder choices').toBeTruthy();
    await poor.openCard(plain!);
    await expect(page.locator('dialog[open] [data-control="wonder-step"]')).not.toHaveAttribute(
      'data-wonder-ready'
    );
    await shots(page, testInfo, 'sheet-plain');

    // A card whose shields take a looting token: its loss mark, listed.
    const loot = await room(page, baseURL, {
      mode: 'ai',
      bot: 'easy',
      seed: '7',
      scenario: 'looting-double'
    });
    await loot.openCard('arsenal');
    await expect(
      page.locator('dialog[open] [data-card-marks] [data-hint-line="loot"]')
    ).toBeVisible();
    await shots(page, testInfo, 'sheet-loot');
  });

  test('a one-time tip on touch says to press and hold a mark, and leads to the legend', async ({
    page,
    browser,
    baseURL
  }, testInfo) => {
    test.setTimeout(300_000);
    if (!baseURL) throw new Error('no baseURL');

    // A fine pointer, tips on: the pair's own tip, never the long-press one.
    await page.addInitScript((tips) => {
      if (!sessionStorage.getItem('signals-tips')) {
        sessionStorage.setItem('signals-tips', '1');
        localStorage.setItem('ajayeb:tips', tips);
      }
    }, TIPS_ON);
    await room(page, baseURL, { mode: 'ai', bot: 'easy', seed: '7', scenario: 'science-pair' });
    await expect(page.locator('[data-first-tip="pair"]')).toBeVisible();
    await expect(page.locator('[data-first-tip="longPress"]')).toHaveCount(0);

    for (const locale of [undefined, 'fa']) {
      const touch = await touchPage(browser, testInfo, locale);
      // A handler that throws fails quietly on the page (the tip's More once did).
      const errors: string[] = [];
      touch.on('pageerror', (e) => errors.push(e.message));
      try {
        await room(touch, baseURL, {
          mode: 'ai',
          bot: 'easy',
          seed: '7',
          scenario: 'science-pair'
        });
        const tip = touch.locator('[data-first-tip="longPress"]');
        await expect(tip).toBeVisible();
        const id = locale ? `tip-${locale}` : 'tip';
        await shots(touch, testInfo, id, [PHONE], locale ? ['light'] : ['light', 'dark']);
        if (locale) continue;

        // More in Rules opens the legend of every mark.
        await tip.locator('[data-tip-more]').click();
        const legend = touch.locator('[data-rules-overlay] [data-hint-legend]');
        await expect(legend).toBeVisible();
        await shots(touch, testInfo, 'legend-touch', [PHONE], undefined, HINTS);

        // Seen once: after a reload the pair's own tip comes instead.
        await touch.reload();
        await new GameDriver(touch).waitReady();
        await expect(touch.locator('[data-first-tip="pair"]')).toBeVisible();
        await expect(tip).toHaveCount(0);
        expect(errors).toEqual([]);
      } finally {
        await touch.context().close();
      }
    }

    // The legend from the rules screen's own contents, at both sizes.
    await page.goto('/rules');
    await page.locator('[data-rules-toc-item="hints"]').filter({ visible: true }).first().click();
    await expect(page.locator('[data-hint-legend]')).toBeVisible();
    await shots(page, testInfo, 'legend', undefined, undefined, HINTS);
  });
});
