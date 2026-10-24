// The end of an Age and of the game over live tables (M18.age-military-and-
// scores): Home -> lobby -> Start with bots, at three, five and seven seats,
// in English on the light theme and in Persian (right to left) on the dark
// one. The human sells a card every turn; the capture stops at the end of
// Age I (the military dialog), and at the end of the game on the score
// sheet, the final board and the revealed cards, each at the four review
// viewports. Every capture is probed: no horizontal scroll, no control
// under 44 px, no glyph without a tooltip, the panel inside the viewport,
// and nothing animating with Animation off. The gallery's focused end
// dialogs add the tie-break sentences, since a tie at the top is rare in a
// live game.
import { expect, type Page } from '@playwright/test';
import { mkdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { quiet } from '../fixtures/layout-probes';
import { choose, dismissMilitary, latest, openBotTable } from '../fixtures/table';

const VIEWPORTS = [
  [360, 740],
  [390, 844],
  [844, 390],
  [1280, 800]
] as const;
const out = process.env.HG_SCORES_SHOTS_DIR ?? '../.cache/e2e/hg-scores-shots';

/** What a capture must not show, for the element under `selector`. */
async function probe(page: Page, selector: string) {
  return page.locator(selector).evaluate((root) => {
    const box = (root.closest('dialog') ?? root).getBoundingClientRect();
    const small = [...root.querySelectorAll<HTMLElement>('button:not(:disabled), select, a[href]')]
      .filter((e) => !e.closest('[inert]') && e.checkVisibility())
      .map((e) => ({
        key: e.dataset['act'] ?? e.textContent?.trim().slice(0, 20),
        w: Math.round(e.getBoundingClientRect().width),
        h: Math.round(e.getBoundingClientRect().height)
      }))
      .filter((e) => e.w < 44 || e.h < 44);
    const untipped = [...root.querySelectorAll('svg, [role="img"]')]
      .filter((e) => {
        if (e.parentElement?.closest('svg') || e.closest('[data-tooltip]')) return false;
        if (!e.checkVisibility()) return false;
        const button = e.closest('button');
        return !(button && [...button.childNodes].some((n) => n.textContent?.trim()));
      })
      .map((e) => e.outerHTML.slice(0, 100));
    const scrollers = [root, ...root.querySelectorAll<HTMLElement>('[data-sheet-body]')];
    return {
      overflow:
        document.documentElement.scrollWidth > innerWidth + 1 ||
        scrollers.some((e) => e.scrollWidth > e.clientWidth + 1),
      small,
      untipped,
      outside:
        box.x < -1 || box.y < -1 || box.right > innerWidth + 1 || box.bottom > innerHeight + 1,
      animated: document
        .getAnimations()
        .filter(
          (a) => a.playState === 'running' && Number(a.effect?.getComputedTiming().duration) > 1
        ).length
    };
  });
}

const CLEAN = { overflow: false, small: [], untipped: [], outside: false, animated: 0 };

/** Sells a card each turn until the military dialog opens or the game ends. */
async function playToDialog(page: Page): Promise<void> {
  for (let step = 0; step < 40; step++) {
    if (await page.locator('[data-age-military]').isVisible()) return;
    const f = await latest(page);
    if (f.status === 'finished') return;
    const me = f.you.seat!;
    if (f.point?.required.includes(me) && !f.point.confirmed.includes(me)) {
      const cs = f.view.you!.choices;
      await choose(page, (cs.find((c) => c.kind === 'sell') ?? cs[0]!).key);
      await choose(page, 'confirm');
    }
    await expect
      .poll(async () => {
        const g = await latest(page);
        return g.batch !== f.batch || g.status === 'finished';
      })
      .toBe(true);
  }
  throw new Error('no end of Age within 40 steps');
}

for (const seats of [3, 5, 7]) {
  for (const [locale, theme] of [
    ['en', 'light'],
    ['fa', 'dark']
  ] as const) {
    test.describe(`live Haftgane end ${seats}p ${locale} ${theme}`, () => {
      test.use({ appLocale: locale });
      test('the end of Age I and of the game over a live table', async ({ page }) => {
        test.setTimeout(600_000);
        mkdirSync(out, { recursive: true });
        const problems = watchHygiene(page);
        await page.addInitScript((t) => localStorage.setItem('ajayeb:theme', t), theme);
        const code = await openBotTable(page, seats, 'Player');
        await expect(page.locator('html')).toHaveAttribute('dir', locale === 'fa' ? 'rtl' : 'ltr');
        await expect(page.locator('html')).toHaveAttribute('data-theme', theme);
        const probes: Record<string, unknown>[] = [];
        const capture = async (state: string, selector: string, before?: () => Promise<void>) => {
          for (const [width, height] of VIEWPORTS) {
            await page.setViewportSize({ width, height });
            if (before) await before();
            await quiet(page);
            const p = await probe(page, selector);
            probes.push({ code, seats, state, width, height, locale, theme, ...p });
            await page.screenshot({
              path: path.join(out, `${seats}p-${state}-${width}x${height}-${locale}-${theme}.png`)
            });
            expect.soft(p, `${state} at ${width}x${height}`).toEqual(CLEAN);
          }
        };
        // A sheet with a footer scrolls its body; one without scrolls the panel.
        const top = (selector: string) => async () => {
          await page
            .locator(selector)
            .evaluate((panel) =>
              (panel.querySelector('[data-sheet-body]') ?? panel).scrollTo(0, 0)
            );
        };

        await playToDialog(page);
        await expect(page.locator('[data-age-military]')).toHaveAttribute('data-age-military', '1');
        await capture('military', '[data-age-military]', top('[data-age-military]'));
        // The whole conflict list at the tallest seat count, scrolled down.
        await capture('military-end', '[data-age-military]', async () => {
          await page
            .locator('[data-age-military] [data-sheet-body]')
            .evaluate((body) => body.scrollTo(0, body.scrollHeight));
        });
        await dismissMilitary(page);

        for (let age = 2; age <= 3; age++) {
          await playToDialog(page);
          await expect(page.locator('[data-age-military]')).toHaveAttribute(
            'data-age-military',
            String(age)
          );
          if (age === 3) break;
          await dismissMilitary(page);
        }
        expect((await latest(page)).status).toBe('finished');
        await dismissMilitary(page);
        await capture('scores', '[data-score-sheet]', top('[data-score-sheet]'));
        await page.locator('[data-score-sheet] [data-act="final-board"]').click();
        await capture('final-board', '[data-end-actions]');
        await page.locator('[data-act="open-reveal"]').click();
        await capture('reveal', '[data-final-reveal]', top('[data-final-reveal]'));
        await page.locator('[data-act="reveal-close"]').click();

        writeFileSync(
          path.join(out, `probes-${seats}p-${locale}-${theme}.json`),
          JSON.stringify({ probes, problems }, null, 2)
        );
        expect(problems).toEqual([]);
      });
    });
  }
}

// A tie at the top (result.tieBreak) is rare in a live game, so the gallery
// shows the tie-break sentences, with made-up data.
for (const [locale, theme] of [
  ['en', 'light'],
  ['fa', 'dark']
] as const) {
  test.describe(`gallery Haftgane end ${locale} ${theme}`, () => {
    test.use({ appLocale: locale });
    test('the tie-break sentences on the score sheet', async ({ page }) => {
      mkdirSync(out, { recursive: true });
      const problems = watchHygiene(page);
      await page.addInitScript((t) => localStorage.setItem('ajayeb:theme', t), theme);
      for (const kind of ['tie', 'shared'] as const) {
        await page.goto(`/dev/gallery?hgEnd=${kind}&seats=7`);
        const sheet = page.locator('[data-score-sheet]');
        await expect(sheet).toBeVisible();
        await expect(sheet.locator('[data-tie-break]')).toHaveAttribute(
          'data-tie-break',
          kind === 'tie' ? 'coins' : 'shared'
        );
        for (const [width, height] of VIEWPORTS) {
          await page.setViewportSize({ width, height });
          await quiet(page);
          const p = await probe(page, '[data-score-sheet]');
          await page.screenshot({
            path: path.join(out, `gallery-${kind}-${width}x${height}-${locale}-${theme}.png`)
          });
          expect.soft(p, `${kind} at ${width}x${height}`).toEqual(CLEAN);
        }
      }
      expect(problems).toEqual([]);
    });
  });
}
