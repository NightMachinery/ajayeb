// Required viewport review for compact diagrams and the same physical card
// layout in English and Persian. Screenshots are test artifacts only.
import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { openHotseatRoom, playKey } from '../fixtures/scenario';
import { settle } from '../fixtures/layout-probes';
import { watchHygiene } from '../fixtures/hygiene';

test.beforeEach(async ({ page }) => {
  await page.addInitScript(() => {
    const current = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}') as Record<
      string,
      unknown
    >;
    localStorage.setItem(
      'ajayeb:settings',
      JSON.stringify({ ...current, hotseatPassDevice: false })
    );
    localStorage.setItem('ajayeb:name', 'Layout');
  });
});

const VIEWPORTS = [
  { width: 360, height: 740 },
  { width: 390, height: 844 },
  { width: 740, height: 360 },
  { width: 1280, height: 800 }
];

async function diagramFits(page: Page, name: string): Promise<void> {
  const diagram = page.locator(`[data-tooltip-layer] [data-rules-diagram="${name}"]`);
  await expect(diagram).toBeVisible();
  await expect(page.locator('[data-tooltip-layer] .tooltip-group[data-placed]')).toHaveCSS(
    'opacity',
    '1'
  );
  await page.evaluate(() => document.fonts.ready);
  await page.evaluate(
    () =>
      new Promise<void>((resolve) =>
        requestAnimationFrame(() => requestAnimationFrame(() => resolve()))
      )
  );
  const parts = await diagram.locator('svg, [data-swatch]').evaluateAll((els) =>
    els.map((el) => {
      const box = el.getBoundingClientRect();
      const cs = getComputedStyle(el);
      return {
        tag: el.tagName,
        width: box.width,
        height: box.height,
        visibility: cs.visibility,
        opacity: cs.opacity
      };
    })
  );
  expect(
    parts.filter(
      (p) => p.width < 1 || p.height < 1 || p.visibility !== 'visible' || p.opacity === '0'
    )
  ).toEqual([]);
  const bubble = page.locator('[data-tooltip-layer] [data-tooltip-bubble]');
  const vp = page.viewportSize();
  if (!vp) throw new Error('missing viewport');
  const stack = page.locator('[data-tooltip-layer] [data-tooltip-stack]');
  for (const part of [bubble, ...((await stack.count()) ? [stack] : [])]) {
    const box = await part.boundingBox();
    if (!box) throw new Error('missing tooltip part');
    expect(box.x).toBeGreaterThanOrEqual(0);
    expect(box.y).toBeGreaterThanOrEqual(0);
    expect(box.x + box.width).toBeLessThanOrEqual(vp.width + 1);
    expect(box.y + box.height).toBeLessThanOrEqual(vp.height + 1);
  }
}

for (const locale of ['en', 'fa'] as const) {
  test.describe(`review follow-up layouts ${locale}`, () => {
    test.use({ appLocale: locale });
    for (const viewport of VIEWPORTS) {
      test(`${viewport.width}x${viewport.height}`, async ({ page }, info) => {
        const problems = watchHygiene(page);
        await page.setViewportSize(viewport);
        await page.goto('/dev/gallery/tooltips');
        for (const [sample, selector, name] of [
          ['chain-status', '[data-chain]', 'chain'],
          ['guild-effect', '[data-effect]', 'guild'],
          ['seventh-wonder', '[data-control="wonders"]', 'seventh']
        ]) {
          const anchor = page.locator(`[data-tip-demo="${sample}"] ${selector}`);
          await anchor.hover();
          await diagramFits(page, name!);
          await page.screenshot({
            path: info.outputPath(`${name}-${locale}-${viewport.width}.png`)
          });
          await page.keyboard.press('Escape');
        }
        const driver = await openHotseatRoom(page, 'extra-turn-lost');
        await driver.nextTurn();
        await page.locator('.card-structure [data-card][data-accessible]').first().click();
        const sheet = page.locator('dialog[open] [data-card-sheet]');
        await expect(sheet).toBeVisible();
        expect(
          await sheet
            .locator('.card-face')
            .first()
            .evaluate((el) => getComputedStyle(el).direction)
        ).toBe('ltr');
        await sheet.locator('[data-act^="discard:"]').hover();
        await diagramFits(page, 'discard');
        await page.screenshot({
          path: info.outputPath(`card-sheet-${locale}-${viewport.width}.png`)
        });
        await page.keyboard.press('Escape');
        await page.keyboard.press('Escape');
        await playKey(driver, 'discard:theater');
        const why = page.locator('[data-starter-choice] > p').first();
        await why.hover();
        await diagramFits(page, 'starter');
        await page.screenshot({ path: info.outputPath(`starter-${locale}-${viewport.width}.png`) });
        await page.keyboard.press('Escape');
        await playKey(driver, 'starter:0');
        await settle(page);
        const row = page.locator('.progress-token-row').filter({ visible: true }).first();
        expect(await row.evaluate((el) => getComputedStyle(el).direction)).toBe('ltr');
        await page.screenshot({ path: info.outputPath(`board-${locale}-${viewport.width}.png`) });
        expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(
          true
        );
        expect(problems).toEqual([]);
      });
    }
  });
}

test('a literal centred-pawn ruling never offers the starter-choice diagram', async ({
  page,
  baseURL
}) => {
  if (!baseURL) throw new Error('no baseURL');
  const response = await page.request.post(`${baseURL}/api/dev/rooms`, {
    data: {
      mode: 'hotseat',
      seed: '7',
      scenario: 'extra-turn-lost',
      actions: ['discard:theater'],
      options: { centerPawn: 'lastActiveStarts' }
    }
  });
  expect(response.ok()).toBe(true);
  const room = (await response.json()) as { path: string };
  await page.goto(room.path);
  await settle(page);
  expect(await page.evaluate(() => window.__ajayeb?.view()?.phase)).toBe('play');
  await expect(page.locator('[data-starter-choice]')).toHaveCount(0);
  const age = page.locator('[data-age-tip]');
  if (await age.count()) {
    await age.hover();
    await expect(page.locator('[data-tooltip-layer] [data-rules-diagram="starter"]')).toHaveCount(
      0
    );
  }
});
