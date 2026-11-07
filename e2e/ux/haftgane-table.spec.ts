import { installMotionOff } from '../fixtures/base';
import { reviewCapture, reviewHygiene, reviewSeats, reviewViewports } from './haftgane-review';
// Haftgane's table screen geometry (M18.table-screen): tables of 3 to 7
// seats, the page's user and bots, at 360x740, 390x844, 844x390 and
// 1280x800, in both themes, left to right and right to left. At each size,
// with a full hand and nothing selected, two frames after the resize, the
// probe finds no horizontal overflow, no control under 44 px, no seat,
// hand card or bar cut off by the screen's edges, no glyph without a
// tooltip and nothing animating with Animation off. Then the page plays a
// turn, and the moment the next hand is on screen nothing animates; a last
// test is the positive control, where the same moment with Animation on
// counts the hand's deal. The page logs no console error or failed
// request. Each state is saved as a screenshot for the contact sheet, in
// HG_TABLE_SHOTS_DIR.
import { expect, type Page } from '@playwright/test';
import { mkdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { useSettings } from '../fixtures/anim';
import { choose, dismissMilitary, latest, openBotTable, openMenu, ready } from '../fixtures/table';

const sizes = [
  [360, 740],
  [390, 844],
  [844, 390],
  [1280, 800]
] as const;

interface Probe {
  overflow: boolean;
  small: { act: string; width: number; height: number }[];
  cut: string[];
  untipped: string[];
  animated: number;
}

async function probe(page: Page): Promise<Probe> {
  return page.evaluate(() => {
    const root = document.querySelector<HTMLElement>('[data-table-screen]')!;
    const describe = (el: Element) =>
      [...el.attributes]
        .filter((a) => a.name.startsWith('data-'))
        .map((a) => `[${a.name}=${a.value}]`)
        .join('') || el.tagName.toLowerCase();
    const shown = (el: Element) =>
      el.getClientRects().length > 0 &&
      el.checkVisibility({ opacityProperty: true, visibilityProperty: true });
    // The part of an element left once every clipping ancestor and the
    // viewport have cut it.
    const cutOff = (el: Element): boolean => {
      const r = el.getBoundingClientRect();
      let { left, right, top, bottom } = r;
      left = Math.max(left, 0);
      top = Math.max(top, 0);
      right = Math.min(right, innerWidth);
      bottom = Math.min(bottom, innerHeight);
      for (let a = el.parentElement; a; a = a.parentElement) {
        const style = getComputedStyle(a);
        if (style.overflowX !== 'visible' || style.overflowY !== 'visible') {
          const c = a.getBoundingClientRect();
          left = Math.max(left, c.left);
          right = Math.min(right, c.right);
          top = Math.max(top, c.top);
          bottom = Math.min(bottom, c.bottom);
        }
      }
      return right - left < r.width - 1 || bottom - top < r.height - 1;
    };
    const controls = [
      ...root.querySelectorAll<HTMLElement>('button, a, select, input, [role="button"]')
    ].filter(shown);
    const parts = [
      ...root.querySelectorAll('[data-seat], [data-hand] [data-card], [data-turn-bar]'),
      ...controls
    ].filter(shown);
    const untipped: string[] = [];
    for (const el of root.querySelectorAll('svg, [role="img"]')) {
      if (el.parentElement?.closest('svg') || !shown(el)) continue;
      if (el.closest('[data-tooltip]')) continue;
      const button = el.closest('button');
      if (button && [...button.childNodes].some((n) => n.textContent?.trim())) continue;
      untipped.push(describe(el.parentElement ?? el));
    }
    return {
      overflow:
        root.scrollWidth > root.clientWidth + 1 ||
        document.documentElement.scrollWidth > innerWidth + 1,
      small: controls
        .map((el) => ({
          act: el.dataset['act'] ?? el.tagName,
          width: Math.round(el.getBoundingClientRect().width),
          height: Math.round(el.getBoundingClientRect().height)
        }))
        .filter((r) => r.width < 43.9 || r.height < 43.9),
      cut: parts.filter(cutOff).map(describe).slice(0, 10),
      untipped: untipped.slice(0, 10),
      animated: document.getAnimations().filter((a) => {
        const timing = a.effect?.getComputedTiming();
        return (
          a.playState === 'running' && typeof timing?.duration === 'number' && timing.duration > 1
        );
      }).length
    };
  });
}

/** Past a side point, if the table has one, to a turn with a full hand. */
async function pastSides(page: Page): Promise<void> {
  const sides = await latest(page);
  if (sides.view.phase !== 'sides') return;
  await choose(page, 'side:day');
  await choose(page, 'confirm');
  await expect.poll(async () => (await latest(page)).point?.ref.id).not.toBe(sides.point!.ref.id);
}

/**
 * Plays the page's first offered choice of the turn and, in the first
 * frame that shows the next hand (the bots decide at once), counts what is
 * still animating: measured at once, not after the motion has settled.
 */
async function animatedAfterTurn(page: Page): Promise<number> {
  const f = await latest(page);
  await choose(page, f.view.you!.choices[0]!.key);
  const hand = await page.locator('[data-hand]').elementHandle();
  await page.locator('[data-act="confirm"]').click();
  return page.evaluate(
    (old) =>
      new Promise<number>((resolve) => {
        const tick = () => {
          const now = document.querySelector('[data-hand]');
          if (!now || now === old) return requestAnimationFrame(tick);
          requestAnimationFrame(() =>
            resolve(
              document.getAnimations().filter((a) => {
                const timing = a.effect?.getComputedTiming();
                return (
                  a.playState === 'running' &&
                  typeof timing?.duration === 'number' &&
                  timing.duration > 1
                );
              }).length
            )
          );
        };
        tick();
      }),
    hand
  );
}

for (const locale of ['en', 'fa'] as const)
  for (const theme of ['light', 'dark'] as const) {
    test.describe(`haftgane table geometry ${locale} ${theme}`, () => {
      test.use({ appLocale: locale });
      test('3 to 7 seats at four viewports', async ({ page }) => {
        test.setTimeout(400_000);
        const out = process.env.HG_TABLE_SHOTS_DIR ?? '../.cache/e2e/hg-table-shots';
        mkdirSync(out, { recursive: true });
        const problems = watchHygiene(page);
        const player = locale === 'fa' ? 'روشن' : 'Roshan';
        await page.addInitScript(
          ({ t, name }) => {
            localStorage.setItem('ajayeb:theme', t);
            localStorage.setItem('ajayeb:name', name);
          },
          { t: theme, name: player }
        );
        const probes = [];
        for (const seats of [3, 4, 5, 6, 7]) {
          await page.setViewportSize({ width: 390, height: 844 });
          await openBotTable(page, seats, player);
          await expect(page.locator('html')).toHaveAttribute('data-theme', theme);
          await expect(page.locator('html')).toHaveAttribute(
            'dir',
            locale === 'fa' ? 'rtl' : 'ltr'
          );
          await pastSides(page);
          const turn = await latest(page);
          const design = turn.view.you!.hand[0]!;
          for (const [width, height] of sizes) {
            await page.setViewportSize({ width, height });
            // The sheet's complete variant matrix lives in haftgane-card-sheet.spec.ts.
            for (const state of ['hand'] as const) {
              const card = page.locator(`[data-act="hand:${design}"]`).first();
              const pressed = (await card.getAttribute('aria-pressed')) === 'true';
              if (state === 'hand' && pressed)
                await page.locator('[data-act="pick-close"]').click();
              await ready(page);
              // Two frames, so the layout has followed the new size; no
              // wait for motion to end, which would hide it.
              await page.evaluate(
                () =>
                  new Promise<void>((resolve) =>
                    requestAnimationFrame(() => requestAnimationFrame(() => resolve()))
                  )
              );
              const p = await probe(page);
              probes.push({ seats, width, height, locale, theme, state, ...p });
              const name = `table-${seats}p-${state}-${width}x${height}-${locale}-${theme}.png`;
              await page.screenshot({ path: path.join(out, name) });
              expect(p, `${seats} seats ${state} ${width}x${height}`).toEqual({
                overflow: false,
                small: [],
                cut: [],
                untipped: [],
                animated: 0
              });
            }
          }
          expect(await animatedAfterTurn(page), `${seats} seats, the next hand`).toBe(0);
        }
        writeFileSync(
          path.join(out, `probes-${locale}-${theme}.json`),
          JSON.stringify({ probes, problems }, null, 2)
        );
        expect(problems).toEqual([]);
      });
    });
  }

test.describe('haftgane table motion control', () => {
  test('with Animation on, the next hand is counted the moment it lands', async ({ page }) => {
    test.setTimeout(120_000);
    const problems = watchHygiene(page);
    await useSettings(page, 'relaxed');
    await page.setViewportSize({ width: 390, height: 844 });
    await openBotTable(page, 3, 'Roshan');
    await pastSides(page);
    expect(await animatedAfterTurn(page)).toBeGreaterThan(0);
    expect(problems).toEqual([]);
  });
});

// One real game per seat count and direction; the review photographs each
// state in both themes and at phone/desktop size without replaying the game.
for (const seats of reviewSeats)
  for (const locale of ['en', 'fa'] as const) {
    test.describe(`haftgane review live ${seats}p ${locale}`, () => {
      test.use({ appLocale: locale, deviceScaleFactor: 1 });
      test('lobby through rematch with shared probes', async ({ browser, baseURL }, info) => {
        test.setTimeout(300_000);
        // Separate touch and pointer contexts are necessary: resizing alone
        // does not emulate the phone's input or its layout viewport.
        for (const viewport of reviewViewports) {
          const context = await browser.newContext({
            viewport,
            baseURL,
            isMobile: viewport.width < 1000,
            hasTouch: viewport.width < 1000,
            deviceScaleFactor: 1,
            serviceWorkers: 'block'
          });
          try {
            await installMotionOff(context);
            await context.addInitScript((locale) => {
              const settings = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}');
              localStorage.setItem('ajayeb:settings', JSON.stringify({ ...settings, locale }));
              localStorage.setItem('ajayeb:theme', 'system');
            }, locale);
            const page = await context.newPage();
            const hygiene = reviewHygiene(page);
            const capture = async (screen: string, scope = 'document') => {
              for (const theme of ['light', 'dark'] as const) {
                await page.emulateMedia({ colorScheme: theme });
                await expect(page.locator('html')).toHaveAttribute('data-theme', theme);
                await expect(page.locator('html')).toHaveAttribute(
                  'dir',
                  locale === 'fa' ? 'rtl' : 'ltr'
                );
                await reviewCapture(page, info, screen, seats, hygiene, scope);
              }
            };
            await openBotTable(page, seats, 'Player', async () => capture('hg-lobby'));
            await pastSides(page);
            const ages = new Set<number>();
            let seenMemory = false;
            for (let step = 0; step < 100; step++) {
              if (await page.locator('[data-age-military]').isVisible()) {
                const age = await page
                  .locator('[data-age-military]')
                  .getAttribute('data-age-military');
                await capture(`hg-end-age${age}`);
                await dismissMilitary(page);
              }
              const f = await latest(page);
              if (f.status === 'finished') break;
              if (f.view.phase === 'turn' && !ages.has(f.view.age)) {
                ages.add(f.view.age);
                await capture(`hg-table-age${f.view.age}`);
                if (seats === 7 && f.view.age === 3) {
                  const nodes = await page.locator('*').count();
                  await info.attach('ux-budgets', {
                    contentType: 'application/json',
                    body: JSON.stringify([
                      {
                        name: `haftgane-age3-dom-7p-${viewport.width}x${viewport.height}-${locale}`,
                        value: nodes,
                        limit: 2500,
                        unit: 'nodes',
                        details: { seats, age: 3, locale }
                      }
                    ])
                  });
                }
                const card = f.view.you!.hand[0]!;
                await page.locator(`[data-act="hand:${card}"]`).first().click();
                await expect(page.locator('[data-card-sheet]')).toBeVisible();
                await capture(`hg-live-card-age${f.view.age}`);
                await page.locator('[data-act="pick-close"]').click();
                await page.locator('[data-act="seen-hands"]').click();
                await expect(page.locator('[data-seen-sheet]')).toBeVisible();
                await capture(f.view.age === 1 ? 'hg-seen-empty' : `hg-seen-age${f.view.age}`);
                await page.locator('[data-act="seen-close"]').click();
                if (f.view.age === 2) {
                  for (const relation of ['self', 'left', ...(seats > 3 ? ['other'] : [])]) {
                    const seat =
                      relation === 'self'
                        ? f.you.seat!
                        : f.view.seats.find((s) => s.relation === relation)!.seat;
                    await page.locator(`[data-act="city:${seat}"]`).first().click();
                    await expect(page.locator('[data-city-sheet]')).toBeVisible();
                    await capture(`hg-city-${relation}`);
                    await page.locator('[data-act="city-close"]').click();
                  }
                  await openMenu(page);
                  await page.locator('[data-act="rules"]').click();
                  await expect(page.locator('[data-rules-overlay]')).toBeVisible();
                  await capture('hg-rules-game');
                  await page.locator('[data-control="rules-close"]').click();
                  if (await page.locator('[data-act="menu-close"]').isVisible())
                    await page.locator('[data-act="menu-close"]').click();
                }
              }
              if (!seenMemory && f.view.age === 1 && f.view.turn >= 3) {
                seenMemory = true;
                await page.locator('[data-act="seen-hands"]').click();
                await expect(page.locator('[data-seen-sheet]')).toBeVisible();
                await capture('hg-seen-memory');
                await page.locator('[data-act="seen-close"]').click();
              }
              const ref = f.point!.ref;
              if (f.point!.required.includes(f.you.seat!) && !f.pending?.confirm) {
                const cs = f.view.you!.choices;
                const pick =
                  (f.view.phase === 'sides' ? cs.find((c) => c.key === 'side:day') : undefined) ??
                  cs.find((c) => c.kind === 'stage') ??
                  cs.find((c) => c.kind === 'build') ??
                  cs[0]!;
                if (f.view.phase === 'discard-build') {
                  await page
                    .locator(`[data-discard-picker] [data-act="${pick.key}"]`)
                    .first()
                    .click();
                  await ready(page);
                  await page.locator('[data-discard-picker] [data-special-confirm]').click();
                } else {
                  await choose(page, pick.key);
                  await choose(page, 'confirm');
                }
              }
              await expect
                .poll(async () => {
                  const g = await latest(page);
                  return (
                    g.status === 'finished' ||
                    g.point?.ref.id !== ref.id ||
                    g.point?.ref.revision !== ref.revision
                  );
                })
                .toBe(true);
            }
            expect([...ages]).toEqual([1, 2, 3]);
            expect((await latest(page)).status).toBe('finished');
            await dismissMilitary(page);
            await expect(page.locator('[data-score-sheet]')).toBeVisible();
            await capture('hg-final-scores');
            await page.locator('[data-act="final-board"]').click();
            await capture('hg-final-board');
            await page.locator('[data-act="lobby-rematch"]').click();
            await expect(page.locator('[data-lobby]')).toBeVisible();
            await capture('hg-rematch-lobby');
            await choose(page, 'lobby-start');
            await expect(page.locator('[data-table-screen]')).toBeVisible();
            await pastSides(page);
            await capture('hg-rematch-table');
            await page.goto('/rules/haftgane#trade');
            await expect(page.locator('[data-rules-page="haftgane"]')).toBeVisible();
            await capture('hg-rules-page');
          } finally {
            await context.close();
          }
        }
      });
    });
  }
