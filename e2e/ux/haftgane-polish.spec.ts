// Haftgane's table polish (M18.table-polish), over live tables started from
// Home's lobby, the page's user and bots. In English, tables of 3, 5 and 7
// seats; in Persian, one table of 7 seats for the right-to-left set. Both
// themes follow the emulated system, so no reload is needed between them.
//
// On the 7-seat tables, at the first turn: the card sheet's primary button
// names the selected outcome (build, stage or sale, as the server keys it);
// the own city sheet says once that nothing is built, with no "none 0"
// row; and a bot seat's admin menu offers the other frozen bot levels.
// When Age I ends, the military dialog breaks no name inside a word and
// gives each name a tooltip. On every table, at Age II's last turn with a
// card drafted, each hand card is at least 4.5rem wide (44 px on a short
// landscape screen) and its pick tag fits on it, at 360x740, 390x844,
// 844x390 and 1280x800; on the 7-seat tables the own city then counts no
// empty colour. Every state is saved as a screenshot in HG_POLISH_SHOTS_DIR.
import { expect, type Page } from '@playwright/test';
import { mkdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import type { Choice } from '../../web/src/lib/wire/haftgane.gen';
import type { TableState } from '../../web/src/lib/wire/platform.gen';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { quiet } from '../fixtures/layout-probes';
import { choose, latest, openBotTable, openMenu, ready } from '../fixtures/table';

const phone = [360, 740] as const;
const desktop = [1280, 800] as const;
const handSizes = [phone, [390, 844], [844, 390], desktop] as const;
/** A colour row that counts its zero, "none 0", in either digit set. */
const counted0 = (row: string) => /(^|\s)[0۰]$/.test(row);

/** Plays the page's seat until the frame satisfies `done`: a stage when one
 *  is offered, else a building, else the first choice; bots play the rest.
 *  A discard search is confirmed in its own picker. */
async function playUntil(
  page: Page,
  done: (f: TableState) => boolean,
  before?: (f: TableState) => Promise<void>
): Promise<TableState> {
  for (let i = 0; i < 120; i++) {
    const f = await latest(page);
    if (done(f) || f.status === 'finished') return f;
    const me = f.you.seat!;
    const ref = f.point!.ref;
    if (f.point!.required.includes(me) && !f.pending?.confirm) {
      await before?.(f);
      const cs = f.view.you!.choices;
      const pick =
        (f.view.phase === 'sides' ? cs.find((c) => c.key === 'side:day') : undefined) ??
        (f.view.turn > 1 ? cs.find((c) => c.kind === 'stage') : undefined) ??
        cs.find((c) => c.kind === 'build') ??
        cs[0]!;
      if (f.view.phase === 'discard-build') {
        await page.locator(`[data-discard-picker] [data-act="${pick.key}"]`).first().click();
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
  throw new Error('the game did not reach the moment');
}

for (const [locale, tables] of [
  ['en', [3, 5, 7]],
  ['fa', [7]]
] as const)
  for (const seats of tables)
    test.describe(`haftgane table polish ${locale} ${seats} seats`, () => {
      test.use({ appLocale: locale });
      test('card sheet, city rows, bot levels, military names and the last-turn hand', async ({
        page
      }) => {
        test.setTimeout(900_000);
        const out = process.env.HG_POLISH_SHOTS_DIR ?? '../.cache/e2e/hg-polish-shots';
        mkdirSync(out, { recursive: true });
        const problems = watchHygiene(page);
        const report: Record<string, unknown>[] = [];
        const player = locale === 'fa' ? 'روشن' : 'Roshan';
        await page.addInitScript((n) => {
          localStorage.setItem('ajayeb:name', n);
          // The shots switch themes through the emulated system scheme.
          localStorage.setItem('ajayeb:theme', 'system');
        }, player);
        await page.setViewportSize({ width: 390, height: 844 });
        await openBotTable(page, seats, player);
        await expect(page.locator('html')).toHaveAttribute('dir', locale === 'fa' ? 'rtl' : 'ltr');

        /** Both themes at each size, with `focus` scrolled into view. */
        async function shots(
          id: string,
          sizes: readonly (readonly [number, number])[],
          check?: (label: string) => Promise<void>,
          focus?: string
        ): Promise<void> {
          for (const scheme of ['light', 'dark'] as const) {
            await page.emulateMedia({ colorScheme: scheme });
            for (const [width, height] of sizes) {
              await page.setViewportSize({ width, height });
              if (focus) await page.locator(focus).first().scrollIntoViewIfNeeded();
              await quiet(page);
              const label = `${id}-${seats}p-${width}x${height}-${locale}-${scheme}`;
              await page.screenshot({ path: path.join(out, `${label}.png`) });
              await check?.(label);
            }
          }
          await page.emulateMedia({ colorScheme: 'light' });
          await page.setViewportSize({ width: 390, height: 844 });
        }

        /** Drafts `c` on its card's sheet and captures the footer, whose
         *  primary button must be that choice. */
        async function sheetShot(c: Choice): Promise<void> {
          if (!(await page.locator('[data-card-sheet]').isVisible()))
            await page.locator(`[data-act="hand:${c.design}"]`).first().click();
          await page.locator(`[data-card-sheet] [data-pick="${c.key}"]`).click();
          await ready(page);
          await shots(`sheet-${c.kind}`, [phone, desktop], async (label) => {
            const primary = await page
              .locator('[data-card-sheet] .sheet-footer button.bg-accent')
              .evaluateAll((els) => els.map((el) => el.getAttribute('data-act')));
            report.push({ label, primary });
            expect.soft(primary, label).toEqual([c.key]);
          });
        }
        // The first turn on a 7-seat table that offers a stage drafts it.
        let staged = seats !== 7;
        async function stageShot(f: TableState): Promise<void> {
          const c = f.view.you!.choices.find((x) => x.kind === 'stage');
          if (staged || f.view.phase !== 'turn' || !c) return;
          staged = true;
          await sheetShot(c);
          await page.locator('[data-act="pick-close"]').click();
        }

        const first = await playUntil(page, (f) => f.view.phase === 'turn');
        const me = first.you.seat!;
        if (seats === 7) {
          // 1. The card sheet's primary button follows the selected outcome.
          const cs = first.view.you!.choices;
          const design =
            first.view.you!.hand.find((d) =>
              cs.some((c) => c.design === d && c.kind === 'stage')
            ) ?? first.view.you!.hand.find((d) => cs.some((c) => c.design === d))!;
          for (const kind of ['sell', 'stage', 'build'] as const) {
            const c = cs.find((x) => x.design === design && x.kind === kind);
            if (c) await sheetShot(c);
          }
          await page.locator('[data-act="pick-close"]').click();

          // 4. Empty colour rows on the own city sheet.
          await page.locator(`[data-act="city:${me}"]`).first().click();
          await expect(page.locator(`[data-city-sheet="${me}"]`)).toBeVisible();
          await shots(
            'city-empty',
            [phone, desktop],
            async (label) => {
              const rows = await page
                .locator('[data-city-sheet] [data-city-row]')
                .evaluateAll((els) => els.map((el) => (el as HTMLElement).innerText.trim()));
              report.push({ label, rows });
              expect.soft(rows.filter(counted0), label).toEqual([]);
              expect.soft(await page.locator('[data-city-empty]').count(), label).toBe(1);
            },
            '[data-city-sheet] [data-city-rows]'
          );
          await page.locator('[data-act="city-close"]').click();

          // 5. A bot seat's admin menu offers the other bot levels.
          const botSeat = first.seats.find((s) => s.kind === 'bot')!;
          const bot = botSeat.seat;
          await openMenu(page);
          await page.locator(`[data-act="seat-menu:${bot}"]`).click();
          const menu = page.locator(`[data-seat-menu="${bot}"]`);
          await expect(menu).toBeVisible();
          await shots('bot-levels', [phone, desktop], async (label) => {
            const levels = await menu
              .locator(`[data-act^="seat-bot:${bot}:"]`)
              .evaluateAll((els) => els.map((el) => el.getAttribute('data-act')));
            report.push({ label, levels });
            expect.soft(levels.length, label).toBeGreaterThan(0);
            expect.soft(levels, label).not.toContain(`seat-bot:${bot}:${botSeat.bot ?? ''}`);
          });
          await menu.locator('[data-act="seat-menu-close"]').click();
          await expect(menu).toBeHidden();
          await page.locator('[data-act="menu-close"]').click();
          await expect(page.locator('[data-table-menu]')).toBeHidden();
        }

        // 3. Age I's conflicts: no name broken inside a word.
        await playUntil(page, (f) => f.view.age === 2, stageShot);
        if (seats === 7) {
          await expect(page.locator('[data-age-military]')).toBeVisible();
          await shots(
            'military',
            [phone, desktop],
            async (label) => {
              const broken = await page
                .locator('[data-age-military] [data-conflict] .name')
                .evaluateAll((els) =>
                  els.flatMap((el) => {
                    // A word drawn on two lines is broken inside the word.
                    const out: string[] = [];
                    const walk = document.createTreeWalker(el, NodeFilter.SHOW_TEXT);
                    for (let text = walk.nextNode(); text; text = walk.nextNode())
                      for (const w of (text.textContent ?? '').matchAll(/\S+/g)) {
                        const r = document.createRange();
                        r.setStart(text, w.index);
                        r.setEnd(text, w.index + w[0].length);
                        const lines = new Set(
                          [...r.getClientRects()].map((b) => Math.round(b.top))
                        );
                        if (lines.size > 1) out.push(w[0]);
                      }
                    return out;
                  })
                );
              const tipped = await page
                .locator('[data-age-military] [data-conflict] .name')
                .evaluateAll((els) => els.every((el) => el.closest('[data-tooltip]') !== null));
              report.push({ label, broken, tipped });
              expect.soft(broken, label).toEqual([]);
              expect.soft(tipped, label).toBe(true);
            },
            '[data-age-military] [data-conflict]'
          );
        }
        await page.locator('[data-age-military] [data-act="military-close"]').click();

        // 2. Age II's last turn, a card drafted: the hand stays usable.
        const last = await playUntil(page, (f) => f.view.age === 2 && f.view.turn === 6, stageShot);
        const pick =
          last.view.you!.choices.find((c) => c.kind === 'build') ?? last.view.you!.choices[0]!;
        await choose(page, pick.key);
        await expect(page.locator('[data-hand] [data-pick-tag]')).toHaveCount(1);
        await shots('hand-last', handSizes, async (label) => {
          const cards = await page.locator('[data-hand] button[data-card]').evaluateAll((els) =>
            els.map((el) => {
              const r = el.getBoundingClientRect();
              const tag = el.querySelector('[data-pick-tag]')?.getBoundingClientRect();
              return {
                width: Math.round(r.width),
                height: Math.round(r.height),
                tagFits: !tag || (tag.left >= r.left - 0.5 && tag.right <= r.right + 0.5)
              };
            })
          );
          report.push({ label, cards });
          // A short landscape screen holds one row of up to seven cards;
          // elsewhere each card keeps the 4.5rem that fits its tags.
          const least = page.viewportSize()!.height > 512 ? 72 : 44;
          for (const c of cards) {
            expect.soft(c.width, label).toBeGreaterThanOrEqual(least);
            expect.soft(c.height, label).toBeGreaterThanOrEqual(44);
            expect.soft(c.tagFits, label).toBe(true);
          }
        });

        // 4. Mid-game, an empty colour is dimmed and uncounted.
        if (seats === 7) {
          await page.locator(`[data-act="city:${me}"]`).first().click();
          await expect(page.locator(`[data-city-sheet="${me}"]`)).toBeVisible();
          await shots(
            'city-mid',
            [phone, desktop],
            async (label) => {
              const rows = await page
                .locator('[data-city-sheet] [data-city-row]')
                .evaluateAll((els) => els.map((el) => (el as HTMLElement).innerText.trim()));
              report.push({ label, rows });
              expect.soft(rows.filter(counted0), label).toEqual([]);
            },
            '[data-city-sheet] [data-city-rows]'
          );
          await page.locator('[data-act="city-close"]').click();
        }

        writeFileSync(
          path.join(out, `report-${locale}-${seats}p.json`),
          JSON.stringify({ report, problems }, null, 2)
        );
        expect(problems).toEqual([]);
      });
    });
