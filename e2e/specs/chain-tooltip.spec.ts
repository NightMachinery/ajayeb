import { expect, type Locator, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { quiet, settle } from '../fixtures/layout-probes';
import { watchHygiene } from '../fixtures/hygiene';

const bubble = (page: Page) => page.locator('[data-tooltip-layer] [data-tooltip-bubble]');

async function show(page: Page, anchor: Locator, touch: boolean): Promise<void> {
  await anchor.scrollIntoViewIfNeeded();
  if (!touch) {
    await anchor.hover();
    return;
  }
  const box = await anchor.boundingBox();
  if (!box) throw new Error('missing chain anchor');
  const cdp = await page.context().newCDPSession(page);
  try {
    await cdp.send('Input.dispatchTouchEvent', {
      type: 'touchStart',
      touchPoints: [{ x: box.x + box.width / 2, y: box.y + box.height / 2 }]
    });
    await page.waitForTimeout(650);
    await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] });
  } finally {
    await cdp.detach();
  }
}

async function fitsBesideCard(page: Page, anchor: Locator): Promise<void> {
  await expect(bubble(page)).toBeVisible();
  await quiet(page);
  const tip = await bubble(page).boundingBox();
  const card = await anchor.locator('xpath=ancestor::*[@data-tooltip-card][1]').boundingBox();
  const vp = page.viewportSize()!;
  expect(tip).not.toBeNull();
  expect(card).not.toBeNull();
  expect(tip!.x).toBeGreaterThanOrEqual(0);
  expect(tip!.y).toBeGreaterThanOrEqual(0);
  expect(tip!.x + tip!.width).toBeLessThanOrEqual(vp.width + 1);
  expect(tip!.y + tip!.height).toBeLessThanOrEqual(vp.height + 1);
  const overlap =
    Math.max(0, Math.min(tip!.x + tip!.width, card!.x + card!.width) - Math.max(tip!.x, card!.x)) *
    Math.max(0, Math.min(tip!.y + tip!.height, card!.y + card!.height) - Math.max(tip!.y, card!.y));
  expect(overlap).toBe(0);
  expect(await page.evaluate(() => document.documentElement.scrollWidth > innerWidth)).toBe(false);
}

async function openCity(page: Page, touch: boolean, seat: 0 | 1): Promise<Locator> {
  if (touch) {
    await page.locator(`.seat-panel[data-seat="${seat}"] [data-control="open-city"]`).click();
    await page.getByRole('tab').nth(seat).click();
  }
  return touch
    ? page.locator('dialog[open] .city-panel')
    : page.locator(`.seat-panel[data-seat="${seat}"] .city-panel`);
}

for (const locale of ['en', 'fa'] as const) {
  test.describe(`linked chain tooltips ${locale}`, { tag: '@mobile' }, () => {
    test.use({ appLocale: locale });

    test('standalone discard gallery receives linked metadata from its own view', async ({
      page
    }, info) => {
      const touch = info.project.name === 'mobile';
      await page.setViewportSize(
        touch ? { width: 390, height: 844 } : { width: 1280, height: 800 }
      );
      await page.goto('/dev/gallery?choices=discard');
      const anchor = page.locator(
        'dialog[open] [data-discard-choice] [data-card="baths"] [data-chain-direction="out"]'
      );
      await expect(anchor).toBeVisible();
      await page.evaluate(() => document.fonts.ready.then(() => undefined));
      await quiet(page);
      await show(page, anchor, touch);
      await expect(bubble(page)).toContainText(
        locale === 'en' ? 'Aqueduct: Civilian building, Age II.' : 'ساختمان مدنی'
      );
      await expect(bubble(page)).toContainText(
        locale === 'en' ? 'Without the chain for you now:' : 'اکنون بدون زنجیره برای شما:'
      );
      await expect(bubble(page)).toContainText(
        locale === 'en' ? 'trading needed.' : 'خرید منبع از بانک'
      );
      await fitsBesideCard(page, anchor);
      await page.keyboard.press('Escape');
    });

    test('uses public details with hints hidden, the actual city owner, and reverse badge preview', async ({
      page
    }, info) => {
      const touch = info.project.name === 'mobile';
      const problems = watchHygiene(page);
      await page.setViewportSize(
        touch ? { width: 360, height: 740 } : { width: 1280, height: 800 }
      );
      await page.addInitScript(() => {
        localStorage.setItem(
          'ajayeb:hintsHidden',
          JSON.stringify({ families: ['counting'], boardMarks: false })
        );
      });
      await page.goto('/dev/fixture/chain-tooltips-seat0');
      await settle(page);
      const row = await page.evaluate(() =>
        window.__ajayeb!.view()!.cardTooltips.find((r) => r.card === 'baths')
      );
      expect(row?.chainOut?.costs?.map((c) => c.total)).toEqual([9, 4]);
      const city = await openCity(page, touch, 0);
      // A colour's disclosure itself carries data-city-cards, before its cards mount.
      const disclosure = city.locator('details[data-city-cards="blue"]');
      await disclosure.locator('summary').click();
      const out = city.locator('[data-card="baths"] [data-chain-direction="out"]');
      await show(page, out, touch);
      await expect(bubble(page)).toContainText(
        locale === 'en' ? 'Aqueduct: Civilian building, Age II.' : 'ساختمان مدنی'
      );
      await expect(bubble(page)).toContainText(
        locale === 'en' ? 'Without the chain for you now: 9 coins (2 coins short).' : '۹ سکه'
      );
      await expect(bubble(page)).toContainText(
        locale === 'en' ? 'Resource trading needed.' : 'به خرید منبع از بانک نیاز دارد.'
      );
      await expect(bubble(page)).toContainText(
        locale === 'en' ? 'Face up in the structure, available now.' : 'اکنون در دسترس است.'
      );
      await fitsBesideCard(page, out);
      await page.keyboard.press('Escape');
      if (touch) await page.keyboard.press('Escape');

      const rival = await openCity(page, touch, 1);
      await rival.locator('details[data-city-cards="green"] summary').click();
      const rivalOut = rival.locator('[data-card="scriptorium"] [data-chain-direction="out"]');
      await show(page, rivalOut, touch);
      const rivalRow = await page.evaluate(() =>
        window.__ajayeb!.view()!.cardTooltips.find((r) => r.card === 'scriptorium')
      );
      expect(rivalRow?.chainOut?.costs?.[1].total).toBe(4);
      await expect(bubble(page)).toContainText(
        locale === 'en' ? 'Library: Scientific building, Age II.' : 'ساختمان علمی'
      );
      const rivalName = await page.evaluate(() => window.__ajayeb!.room()!.seats[1]!.name);
      await expect
        .poll(async () =>
          ((await bubble(page).textContent()) ?? '').replace(/[\u200e\u200f\u2066-\u2069]/gu, '')
        )
        .toContain(
          locale === 'en'
            ? `Without the chain for ${rivalName} now: 4 coins (3 coins short).`
            : `اکنون بدون زنجیره برای ${rivalName}: ۴ سکه (۳ سکه کم است).`
        );
      await expect(bubble(page)).not.toContainText(locale === 'en' ? 'for you now:' : 'برای شما:');
      await fitsBesideCard(page, rivalOut);
      await page.keyboard.press('Escape');
      if (touch) await page.keyboard.press('Escape');

      const target = page.locator('.card-structure [data-card="aqueduct"]');
      for (const anchor of [
        target.locator('[data-chain-direction="in"]'),
        target.locator('[data-afford="chain"]')
      ]) {
        await show(page, anchor, touch);
        await expect(bubble(page)).toContainText(
          locale === 'en' ? 'Baths: Civilian building, Age I.' : 'گرمابه'
        );
        await expect(bubble(page)).toContainText(locale === 'en' ? '3 victory points' : '۳ امتیاز');
        await expect(bubble(page)).toContainText(locale === 'en' ? 'Built by' : 'ساخته‌ی');
        await expect(bubble(page)).not.toContainText(
          locale === 'en' ? 'Without the chain' : 'بدون زنجیره'
        );
        await fitsBesideCard(page, anchor);
        await expect(page.locator('dialog[open]')).toHaveCount(0);
        await page.keyboard.press('Escape');
      }
      // Long-press release inspects nothing; an ordinary following tap still opens the full tier.
      if (touch) await target.locator('.art').tap();
      else await target.locator('.art').click();
      const full = page.locator('dialog[open] [data-tier="full"] [data-chain-direction="in"]');
      await show(page, full, touch);
      await expect(bubble(page)).toContainText(
        locale === 'en' ? 'Baths: Civilian building, Age I.' : 'گرمابه'
      );
      await fitsBesideCard(page, full);
      await page.keyboard.press('Escape');
      expect(problems).toEqual([]);
    });
  });
}
