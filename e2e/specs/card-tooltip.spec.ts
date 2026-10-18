import { expect, type Locator, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { quiet, settle } from '../fixtures/layout-probes';
import { watchHygiene } from '../fixtures/hygiene';

const bubble = (page: Page) => page.locator('[data-tooltip-layer] [data-tooltip-bubble]');

async function show(page: Page, anchor: Locator, touch: boolean): Promise<void> {
  await anchor.scrollIntoViewIfNeeded();
  if (!touch) {
    await anchor.hover({ position: { x: 4, y: 4 } });
    return;
  }
  const box = await anchor.boundingBox();
  if (!box) throw new Error('no tooltip anchor');
  const cdp = await page.context().newCDPSession(page);
  try {
    await cdp.send('Input.dispatchTouchEvent', {
      type: 'touchStart',
      touchPoints: [{ x: box.x + 4, y: box.y + box.height / 2 }]
    });
    await page.waitForTimeout(650);
    await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] });
  } finally {
    await cdp.detach();
  }
}

for (const locale of ['en', 'fa'] as const) {
  test.describe(`whole-card tooltips in ${locale}`, { tag: '@mobile' }, () => {
    test.use({ appLocale: locale });

    test('art shows the summary, the inner symbol wins, and focus describes the button', async ({
      page
    }, info) => {
      const problems = watchHygiene(page);
      const touch = info.project.name === 'mobile';
      await page.setViewportSize(
        touch ? { width: 360, height: 740 } : { width: 1280, height: 800 }
      );
      await page.goto('/dev/gallery/tooltips');
      const card = page.getByTestId('tip-card');
      await show(page, card.locator('.art'), touch);
      await expect(bubble(page)).toContainText(
        locale === 'en' ? 'Baths: Civilian building, Age I.' : 'ساختمان مدنی'
      );
      await expect(bubble(page)).toContainText(
        locale === 'en' ? 'Printed cost:' : 'هزینه‌ی چاپ‌شده:'
      );
      await expect(page.locator('dialog[open]')).toHaveCount(0);
      const rect = await bubble(page).boundingBox();
      const vp = page.viewportSize()!;
      expect(rect).not.toBeNull();
      expect(rect!.x).toBeGreaterThanOrEqual(0);
      expect(rect!.x + rect!.width).toBeLessThanOrEqual(vp.width);
      expect(rect!.y).toBeGreaterThanOrEqual(0);
      expect(rect!.y + rect!.height).toBeLessThanOrEqual(vp.height);
      await page.keyboard.press('Escape');
      await show(page, card.locator('[data-chain="drop"]'), touch);
      await expect(
        page.locator('[data-tooltip-layer] [data-card-preview="aqueduct"]')
      ).toBeVisible();
      await expect(bubble(page)).not.toContainText(
        locale === 'en' ? 'Printed cost:' : 'هزینه‌ی چاپ‌شده:'
      );
      await page.keyboard.press('Escape');
      if (!touch) {
        await page.keyboard.press('Tab');
        await card.focus();
        await expect(card).toHaveAttribute('aria-describedby', 'ajayeb-tooltip');
        await expect
          .poll(async () =>
            ((await bubble(page).textContent()) ?? '').replace(/[\u200e\u200f\u2066-\u2069]/gu, '')
          )
          .toContain(
            locale === 'en'
              ? 'Baths: Civilian building, Age I.'
              : 'گرمابه: ساختمان مدنی، دوران یکم.'
          );
        await page.keyboard.press('Escape');
        await expect(card).not.toHaveAttribute('aria-describedby');
      }
      expect(problems).toEqual([]);
    });

    test('a draft wonder has an art summary and keeps its inner effect tooltip', async ({
      page
    }, info) => {
      const touch = info.project.name === 'mobile';
      await page.setViewportSize(
        touch ? { width: 390, height: 844 } : { width: 1280, height: 800 }
      );
      await page.goto('/dev/fixture/draft-round-1-seat0');
      await settle(page);
      const wonder = page.locator('[data-wonder-offer] .wonder-tile').first();
      await show(page, wonder.locator(':scope > span').first(), touch);
      await expect(bubble(page)).toContainText(locale === 'en' ? ': wonder.' : ': شگفتی.');
      await expect(bubble(page)).toContainText(
        locale === 'en' ? 'Printed cost:' : 'هزینه‌ی چاپ‌شده:'
      );
      await expect(page.locator('dialog[open]')).toHaveCount(0);
      await page.keyboard.press('Escape');
      await show(page, wonder.locator('[data-effect]').first(), touch);
      await expect(bubble(page)).not.toContainText(locale === 'en' ? ': wonder.' : ': شگفتی.');
      await page.keyboard.press('Escape');
    });

    test('a Mausoleum rebuild shows the explicit free choice payment', async ({ page }, info) => {
      const touch = info.project.name === 'mobile';
      await page.setViewportSize(
        touch ? { width: 390, height: 844 } : { width: 1280, height: 800 }
      );
      for (const path of [
        '/dev/fixture/build-from-discard-seat0',
        '/dev/gallery?choices=discard'
      ]) {
        await page.goto(path);
        const card = page.locator('dialog[open] [data-discard-choice] [data-card="baths"]');
        await expect(card).toBeVisible();
        if (path.includes('/fixture/')) await settle(page);
        else {
          await page.evaluate(() => document.fonts.ready.then(() => undefined));
          await quiet(page);
        }
        await show(page, card.locator('.art'), touch);
        await expect(bubble(page)).toContainText(
          locale === 'en' ? 'You now: free.' : 'اکنون برای شما:'
        );
        await expect(bubble(page)).not.toContainText(
          locale === 'en' ? 'Printed cost:' : 'هزینه‌ی چاپ‌شده:'
        );
        if (path.includes('/fixture/')) {
          const payment = await page.evaluate(
            () =>
              window.__ajayeb
                ?.view()
                ?.choices.find((c) => c.kind === 'revive' && c.card === 'baths')?.cost
          );
          expect(payment).toMatchObject({ total: 0, affordable: true, chainFree: false });
        }
        await page.keyboard.press('Escape');
      }
    });

    test('live cards show both server quotes and cities disclose keyboard-readable cards lazily', async ({
      page
    }, info) => {
      const problems = watchHygiene(page);
      const touch = info.project.name === 'mobile';
      await page.setViewportSize(
        touch ? { width: 390, height: 844 } : { width: 1280, height: 800 }
      );
      await page.goto('/dev/fixture/costs-seat0');
      await expect(page.locator('.card-structure')).toBeVisible();
      await settle(page);
      const card = page.locator('.card-structure [data-card][data-accessible]').first();
      const slug = await card.getAttribute('data-card');
      const row = await page.evaluate(
        (slug) => window.__ajayeb?.view()?.cardTooltips.find((r) => r.card === slug),
        slug
      );
      expect(row).toBeTruthy();
      await show(page, card.locator('.art'), touch);
      await expect(bubble(page)).toContainText(locale === 'en' ? 'now:' : 'اکنون');
      const lines = await bubble(page).locator('[data-tooltip-lines] li').allTextContents();
      const quotes = lines.filter((line) =>
        locale === 'en' ? line.includes('now:') : line.includes('اکنون')
      );
      expect(quotes).toHaveLength(2);
      for (const cost of row!.costs) {
        if (cost.total > 0)
          expect(quotes.join(' ')).toContain(new Intl.NumberFormat(locale).format(cost.total));
      }
      await page.keyboard.press('Escape');
      if (touch) {
        await page.locator('[data-control="open-city"]').first().click();
      }
      const city = page.locator('.city-panel').filter({ visible: true }).first();
      await expect(city).toBeVisible();
      const closedCount = await page.locator('*').count();
      const disclosure = city.locator('[data-city-cards]').first();
      await expect(disclosure.locator('[data-tooltip-card]')).toHaveCount(0);
      await disclosure.locator('summary').click();
      const cityCard = disclosure.locator('[data-tooltip-card]').first();
      await expect(cityCard).toBeVisible();
      await expect(cityCard).toHaveAttribute('tabindex', '0');
      const openedCount = await page.locator('*').count();
      console.log(
        `${info.project.name} ${locale} city DOM: closed ${closedCount}, open ${openedCount}`
      );
      await show(page, cityCard.locator('.art'), touch);
      await expect(bubble(page)).toContainText(locale === 'en' ? 'Built by ' : 'ساخته‌ی');
      await expect(bubble(page)).toContainText(
        locale === 'en' ? 'Printed cost:' : 'هزینه‌ی چاپ‌شده:'
      );
      await page.keyboard.press('Escape');
      await disclosure.locator('summary').click();
      await expect(disclosure.locator('[data-tooltip-card]')).toHaveCount(0);
      expect(problems).toEqual([]);
    });
  });
}

test(
  'covered-card and back inspection taps survive tooltips on real touch',
  { tag: '@mobile' },
  async ({ page }, info) => {
    test.skip(info.project.name !== 'mobile', 'real touch inspection uses the phone project');
    await page.setViewportSize({ width: 390, height: 844 });
    for (const specimen of [
      {
        fixture: 'age-1-start-seat0',
        selector: '.slot[data-state="faceUp"]:not([data-accessible]) .art',
        kind: 'card'
      },
      {
        fixture: 'age-1-start-seat0',
        selector: '.slot[data-state="faceUp"]:not([data-accessible]) [data-cost-pip]',
        kind: 'symbol'
      },
      {
        fixture: 'age-1-start-seat0',
        selector: '.slot[data-state="faceDown"] .card-back',
        kind: 'back'
      }
    ]) {
      await page.goto(`/dev/fixture/${specimen.fixture}`);
      await settle(page);
      const point = await page.evaluate((selector) => {
        for (const el of document.querySelectorAll(selector)) {
          const r = el.getBoundingClientRect();
          // Only a painted part that this element really receives, rather
          // than a centre covered by another card in the structure.
          for (let y = r.y + 2; y < Math.min(r.bottom, innerHeight) - 1; y += 3) {
            for (let x = r.x + 2; x < Math.min(r.right, innerWidth) - 1; x += 3) {
              if (el.contains(document.elementFromPoint(x, y))) return { x, y };
            }
          }
        }
        return null;
      }, specimen.selector);
      expect(point, `a visible ${specimen.kind} inspection point`).not.toBeNull();
      const cdp = await page.context().newCDPSession(page);
      try {
        await cdp.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [point!] });
        await page.waitForTimeout(650);
        await expect(bubble(page)).toBeVisible();
        if (specimen.kind === 'card') await expect(bubble(page)).toContainText('Age');
        if (specimen.kind === 'back') await expect(bubble(page)).toContainText('Turns face up');
        if (specimen.kind === 'symbol') await expect(bubble(page)).not.toContainText('Age');
        await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] });
        await expect(page.locator('dialog[open]')).toHaveCount(0);
        // The first tap dismisses only; the next ordinary tap still inspects.
        await page.touchscreen.tap(point!.x, point!.y);
        await expect(bubble(page)).toHaveCount(0);
        await expect(page.locator('dialog[open]')).toHaveCount(0);
        await page.touchscreen.tap(point!.x, point!.y);
        await expect(page.locator('dialog[open] [data-card-sheet]')).toBeVisible();
        if (specimen.kind === 'back') {
          await page.setViewportSize({ width: 740, height: 360 });
          const back = page.locator('dialog[open] [data-card-sheet] .card-back');
          await show(page, back, true);
          await expect(bubble(page)).toContainText('Turns face up');
          const a = await back.boundingBox();
          const b = await bubble(page).boundingBox();
          if (!a || !b) throw new Error('short-phone back or tooltip has no box');
          expect(b.x).toBeGreaterThanOrEqual(0);
          expect(b.y).toBeGreaterThanOrEqual(0);
          expect(b.x + b.width).toBeLessThanOrEqual(740);
          expect(b.y + b.height).toBeLessThanOrEqual(360);
          expect(
            a.x < b.x + b.width &&
              b.x < a.x + a.width &&
              a.y < b.y + b.height &&
              b.y < a.y + a.height
          ).toBe(false);
          await info.attach('short-phone-back', {
            body: await page.screenshot(),
            contentType: 'image/png'
          });
        }
      } finally {
        await cdp.detach();
      }
    }
  }
);
