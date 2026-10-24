// Haftgane in Persian (M18.persian-messages), over one live 7-seat table
// started from Home's lobby: the page's user and six bots, sides chosen by
// each player, and a second device that watches. Every table screen is
// captured at 360x740 and 1280x800 in both themes (the theme follows the
// emulated system): the lobby, the side choice, the table and its card
// sheet, the cities, the game menu and the admin's seat menus, the rules,
// the recap, Seen hands, the spectator's view, the admin's confirmation,
// the watching view of a former player (the admin seats the watching
// device in a bot's seat in Age II, then gives the seat back to a bot), the
// resignation's confirmation (asked, then taken back), the end-of-Age
// conflicts, the score sheet, the final board and the revealed cards, and
// a special decision whenever the deal gives the user one (a discard
// search, a kept card, the last card).
//
// On each capture, and in the tooltips of the first turn's table, the spec
// collects the Latin text on screen outside the people's own names
// (dir="auto") and the room code; there must be none. Every screenshot and
// the report go to HG_PERSIAN_SHOTS_DIR.
import { expect, type Page } from '@playwright/test';
import { mkdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import type { TableState } from '../../web/src/lib/wire/platform.gen';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { quiet } from '../fixtures/layout-probes';
import { lobbyReady } from '../fixtures/lobby';
import { newDeviceContext } from '../fixtures/online';
import {
  choose,
  dismissMilitary,
  dismissRecap,
  latest,
  openBotTable,
  openMenu,
  ready,
  seatAction
} from '../fixtures/table';

const sizes = [
  [360, 740],
  [1280, 800]
] as const;

/** Latin that Persian keeps on purpose (docs/i18n/persian.md): the
 *  wordmark and the QR code's name. */
const kept = new Set(['Ajayeb', 'QR']);

/** The Latin runs on screen, and the Latin commas and semicolons next to
 *  Persian letters (Persian writes «،» and «؛»): in the visible text and the
 *  accessible names and descriptions, outside user text (dir="auto"). The
 *  room code is a code, taken out before the search: it may start with a
 *  digit, which a run does not. */
async function latinOnScreen(page: Page, code: string): Promise<string[]> {
  const found = await page.evaluate((code) => {
    const out = new Set<string>();
    const run = /[A-Za-z][A-Za-z0-9 .,'’:()-]*|[\u0600-\u06ff\u200c]+\s*[,;]\s*[\u0600-\u06ff]+/g;
    const runs = (text: string | null | undefined) =>
      (code ? (text ?? '').replaceAll(code, ' ') : (text ?? '')).match(run) ?? [];
    const user = (el: Element) => el.closest('[dir="auto"]') !== null;
    const walk = document.createTreeWalker(document.body, NodeFilter.SHOW_TEXT);
    for (let n = walk.nextNode(); n; n = walk.nextNode()) {
      const el = n.parentElement;
      if (!el || user(el) || el.closest('script, style, noscript')) continue;
      if (!el.checkVisibility({ visibilityProperty: true })) continue;
      for (const m of runs(n.textContent)) out.add(m.trim());
    }
    for (const el of document.querySelectorAll('[aria-label], [alt], [aria-description]')) {
      if (user(el) || !el.checkVisibility({ visibilityProperty: true })) continue;
      for (const a of ['aria-label', 'alt', 'aria-description'])
        for (const m of runs(el.getAttribute(a))) out.add(`${a}: ${m.trim()}`);
    }
    return [...out];
  }, code);
  return found.filter((s) => !kept.has(s.replace(/^[a-z-]+: /, '')));
}

/** The same in the tooltips of every anchor on screen, hovered one by one
 *  on a fine pointer. */
async function latinInTooltips(page: Page, code: string): Promise<string[]> {
  const out = new Set<string>();
  const anchors = page.locator('[data-tooltip]');
  const bubble = page.locator('[data-tooltip-layer] [data-tooltip-bubble]');
  const count = Math.min(await anchors.count(), 160);
  for (let i = 0; i < count; i++) {
    const a = anchors.nth(i);
    if (!(await a.isVisible())) continue;
    await a.hover({ timeout: 2_000 }).catch(() => undefined);
    if (!(await bubble.isVisible().catch(() => false)))
      await bubble.waitFor({ state: 'visible', timeout: 1_500 }).catch(() => undefined);
    if (await bubble.isVisible()) {
      const latin = await bubble.evaluate((b, code) => {
        const runs: string[] = [];
        const walk = document.createTreeWalker(b, NodeFilter.SHOW_TEXT);
        for (let n = walk.nextNode(); n; n = walk.nextNode())
          if (!n.parentElement?.closest('[dir="auto"]'))
            runs.push(
              ...((n.textContent ?? '')
                .replaceAll(code || '\u0000', ' ')
                .match(
                  /[A-Za-z][A-Za-z0-9 .,'’:()-]*|[\u0600-\u06ff\u200c]+\s*[,;]\s*[\u0600-\u06ff]+/g
                ) ?? [])
            );
        return runs.map((r) => r.trim());
      }, code);
      for (const s of latin) if (!kept.has(s)) out.add(s);
    }
    await page.mouse.move(0, 0);
  }
  return [...out];
}

test.describe('haftgane in Persian', () => {
  test.use({ appLocale: 'fa' });
  test('every table screen, from the lobby to the final board', async ({ page, browser }, info) => {
    test.setTimeout(1_200_000);
    const out = process.env.HG_PERSIAN_SHOTS_DIR ?? '../.cache/e2e/hg-persian-shots';
    mkdirSync(out, { recursive: true });
    const problems = watchHygiene(page);
    const report: { id: string; latin: string[] }[] = [];
    const seen = new Set<string>();
    const player = 'روشن';
    await page.addInitScript((n) => localStorage.setItem('ajayeb:name', n), player);
    await page.setViewportSize({ width: 390, height: 844 });
    let code = '';

    /** Both themes at both sizes, with `focus` scrolled into view, and the
     *  Latin left on screen in each. */
    async function shots(id: string, focus?: string, p: Page = page): Promise<void> {
      seen.add(id);
      code ||= new URL(p.url()).pathname.split('/').at(-1) ?? '';
      for (const scheme of ['light', 'dark'] as const) {
        await p.emulateMedia({ colorScheme: scheme });
        for (const [width, height] of sizes) {
          await p.setViewportSize({ width, height });
          if (focus) await p.locator(focus).first().scrollIntoViewIfNeeded();
          await quiet(p);
          const label = `${id}-${width}x${height}-${scheme}`;
          await p.screenshot({ path: path.join(out, `${label}.png`) });
          const latin = await latinOnScreen(p, code);
          report.push({ id: label, latin });
          expect.soft(latin, label).toEqual([]);
        }
      }
      await p.emulateMedia({ colorScheme: 'light' });
      await p.setViewportSize({ width: 390, height: 844 });
    }

    /** Captures a special decision the deal gives the user, once each. */
    async function special(f: TableState): Promise<void> {
      const you = f.view.you;
      if (!you) return;
      if (f.view.phase === 'discard-build' && f.point!.required.includes(f.you.seat!)) {
        if (!seen.has('special-discard')) {
          await expect(page.locator('[data-discard-picker]')).toBeVisible();
          await shots('special-discard', '[data-discard-picker]');
        }
      } else if (f.view.phase === 'last-card' && f.point!.required.includes(f.you.seat!)) {
        if (!seen.has('special-last')) await shots('special-last');
      } else if (f.view.phase === 'turn' && f.view.turn === 6 && !seen.has('special-sixth')) {
        await shots('special-sixth', '[data-hand]');
      }
      if (you.kept && !seen.has('special-kept')) await shots('special-kept', '[data-hand]');
    }

    /** Plays the user's seat, a stage when one is offered after the first
     *  turn, else a building, until the frame satisfies `done`, or until
     *  the end-of-Age conflicts open when `military` asks for them. */
    async function playUntil(
      done: (f: TableState) => boolean,
      military = false
    ): Promise<TableState> {
      for (let i = 0; i < 150; i++) {
        await dismissRecap(page);
        if (!military) await dismissMilitary(page);
        const f = await latest(page);
        if (done(f) || f.status === 'finished') return f;
        const me = f.you.seat;
        if (
          me !== undefined &&
          me !== null &&
          f.point?.required.includes(me) &&
          !f.pending?.confirm
        ) {
          await special(f);
          const cs = f.view.you!.choices;
          const pick =
            (f.view.phase === 'sides' ? cs.find((c) => c.key === 'side:night') : undefined) ??
            (f.view.turn > 1 ? cs.find((c) => c.kind === 'stage') : undefined) ??
            cs.find((c) => c.kind === 'build') ??
            cs[0]!;
          if (f.view.phase === 'sides' || f.view.phase === 'discard-build') {
            await page.locator(`[data-act="${pick.key}"]`).first().click();
            await ready(page);
            await page.locator('[data-special-confirm]').click();
          } else {
            await choose(page, pick.key);
            await choose(page, 'confirm');
          }
        }
        const ref = f.point?.ref;
        await expect
          .poll(async () => {
            const g = await latest(page);
            if (military && (await page.locator('[data-age-military]').isVisible())) return true;
            return (
              g.status === 'finished' ||
              g.point?.ref.id !== ref?.id ||
              g.point?.ref.revision !== ref?.revision
            );
          })
          .toBe(true);
        if (military && (await page.locator('[data-age-military]').isVisible()))
          return latest(page);
      }
      throw new Error('the game did not reach the moment');
    }

    // The lobby, with sides chosen by each player.
    code = await openBotTable(page, 7, player, async () => {
      await page.locator('[data-act="lobby-sides"]').selectOption('choose');
      await lobbyReady(page);
      await shots('lobby', '[data-lobby]');
    });
    await expect(page.locator('html')).toHaveAttribute('dir', 'rtl');

    // The side choice.
    await expect(page.locator('[data-side-picker]')).toBeVisible();
    await shots('sides', '[data-side-picker]');
    await playUntil((f) => f.view.phase === 'turn');

    // A second device watches.
    const watcher = await newDeviceContext(browser, info);
    await watcher.addInitScript(() => {
      const s = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}');
      localStorage.setItem('ajayeb:settings', JSON.stringify({ ...s, locale: 'fa' }));
      localStorage.setItem('ajayeb:name', 'مهمان');
    });
    const spectator = await watcher.newPage();
    const spectatorProblems = watchHygiene(spectator);
    await spectator.goto(`/play/${code}`);
    await latest(spectator);
    await expect(spectator.locator('[data-table-screen]')).toBeVisible();
    await shots('spectator', undefined, spectator);

    // The first turn: the table, its tooltips, the card sheet, the cities.
    const first = await latest(page);
    const me = first.you.seat!;
    await shots('table');
    await page.setViewportSize({ width: 1280, height: 800 });
    const tips = await latinInTooltips(page, code);
    report.push({ id: 'tooltips-table-1280x800', latin: tips });
    expect.soft(tips, 'tooltips').toEqual([]);
    await page.setViewportSize({ width: 390, height: 844 });

    const cs = first.view.you!.choices;
    const draft = cs.find((c) => c.kind === 'build' && c.left + c.right > 0) ?? cs[0]!;
    await page.locator(`[data-act="hand:${draft.design}"]`).first().click();
    await expect(page.locator('[data-card-sheet]')).toBeVisible();
    const pickButton = page.locator(`[data-card-sheet] [data-pick="${draft.key}"]`);
    if (await pickButton.isVisible()) {
      await pickButton.click();
      await ready(page);
    }
    await shots('sheet', '[data-card-sheet]');
    await page.locator('[data-act="pick-close"]').click();

    await page.locator(`[data-act="city:${me}"]`).first().click();
    await expect(page.locator(`[data-city-sheet="${me}"]`)).toBeVisible();
    await shots('city-own', '[data-city-sheet]');
    const other = first.seats.find((s) => s.seat !== me)!.seat;
    await page.locator(`[data-act="city-tab:${other}"]`).click();
    await expect(page.locator(`[data-city-sheet="${other}"]`)).toBeVisible();
    await shots('city-other', '[data-city-sheet]');
    await page.locator('[data-act="city-close"]').click();

    // The game menu, a bot's seat menu, the spectators, the rules.
    await openMenu(page);
    await shots('menu', '[data-table-menu]');
    await page.locator(`[data-act="seat-menu:${other}"]`).click();
    await expect(page.locator(`[data-seat-menu="${other}"]`)).toBeVisible();
    await shots('seat-menu', `[data-seat-menu="${other}"]`);
    await page.locator('[data-act="seat-menu-close"]').click();
    await page.locator('[data-act="menu-close"]').click();
    await expect(page.locator('[data-table-menu]')).toBeHidden();
    const watching = page.locator('[data-act="spectators"]');
    if (await watching.isVisible()) {
      await watching.click();
      await shots('spectators');
      await page.keyboard.press('Escape');
    }
    await openMenu(page);
    await page.locator('[data-table-menu] [data-act="rules"]').click();
    await shots('rules');
    await page.keyboard.press('Escape');
    await page
      .locator('[data-act="menu-close"]')
      .click({ timeout: 2_000 })
      .catch(() => undefined);

    // The recap of the first turn, and Seen hands a few turns later.
    await playUntil((f) => f.view.phase === 'turn' && f.view.turn === 2);
    await page.locator('[data-act="recap"]').click();
    await expect(page.locator('[data-recap-layer]')).toBeVisible();
    await shots('recap', '[data-recap-layer]');
    await dismissRecap(page);
    await playUntil((f) => f.view.phase === 'turn' && f.view.turn === 4);
    await page.locator('[data-act="seen-hands"]').first().click();
    await shots('seen');
    await page.locator('[data-act="seen-close"]').click();

    // The conflicts at the end of Age I.
    await playUntil(() => false, true);
    await expect(page.locator('[data-age-military]')).toBeVisible();
    await shots('military', '[data-age-military]');
    await page.locator('[data-age-military] [data-act="military-close"]').click();

    // Age II: the admin seats the watching device in a bot's seat, then
    // gives it back to a bot, so the device watches as a former player
    // while the game waits for the admin. Then the admin's own resignation
    // is asked and taken back.
    await playUntil((f) => f.view.age === 2 && f.view.phase === 'turn');
    const bot = (await latest(page)).seats.find((s) => s.kind === 'bot' && s.seat !== other)!.seat;
    const botMenu = page.locator(`[data-seat-menu="${bot}"]`);
    await openMenu(page);
    await page.locator(`[data-act="seat-menu:${bot}"]`).click();
    await botMenu.locator(`[data-act^="seat-user:${bot}:"]`).first().click();
    await expect(botMenu.locator('[data-admin-confirm]')).toBeVisible();
    await shots('admin-confirm', `[data-seat-menu="${bot}"] [data-admin-confirm]`);
    await botMenu.locator('[data-act="admin-confirm"]').click();
    await expect(botMenu).toBeHidden();
    await expect.poll(async () => (await latest(spectator)).you.seat ?? null).toBe(bot);
    await seatAction(page, bot, `seat-bot:${bot}:normal`);
    await expect.poll(async () => (await latest(spectator)).you.seat ?? null).toBeNull();
    await expect(spectator.locator('[data-table-screen]')).toBeVisible();
    // Seated for a moment, the device was shown Age I's conflicts.
    await dismissMilitary(spectator);
    await shots('watching', undefined, spectator);
    await openMenu(page);
    await page.locator('[data-table-menu] [data-act="resign"]').click();
    await expect(page.locator('[data-table-menu] [data-admin-confirm]')).toBeVisible();
    await shots('resign-confirm', '[data-table-menu] [data-admin-confirm]');
    await page.locator('[data-table-menu] [data-act="admin-back"]').click();
    await page.locator('[data-act="menu-close"]').click();
    await expect(page.locator('[data-table-menu]')).toBeHidden();

    // The end: the last conflicts, the scores, the board, the revealed cards.
    await playUntil(() => false, true);
    while ((await latest(page)).status !== 'finished') {
      await page.locator('[data-age-military] [data-act="military-close"]').click();
      await playUntil(() => false, true);
    }
    await expect(page.locator('[data-age-military]')).toBeVisible();
    await shots('military-final', '[data-age-military]');
    await page.locator('[data-age-military] [data-act="military-close"]').click();
    await expect(page.locator('[data-score-sheet]')).toBeVisible();
    await shots('scores', '[data-score-sheet]');
    await page.locator('[data-score-sheet] [data-act="final-board"]').click();
    await expect(page.locator('[data-score-sheet]')).toHaveCount(0);
    await shots('final');
    await page.locator('[data-act="open-reveal"]').click();
    await shots('reveal');
    await page.locator('[data-act="reveal-close"]').click();

    await watcher.close();
    writeFileSync(
      path.join(out, 'report-persian.json'),
      JSON.stringify({ seen: [...seen], report, problems, spectatorProblems }, null, 2)
    );
    expect(problems).toEqual([]);
  });
});
