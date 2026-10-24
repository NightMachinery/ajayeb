// Haftgane's admin controls, resignation and spectator view geometry
// (M18.admin-controls-resign-and-spectator): one live table each of 3, 5
// and 7 seats, started from Home's lobby by its creator with bots; a
// friend joins and the creator seats them in a bot's seat, so the table
// has an admin, a non-admin player and a spectator (who follows a seat),
// each on a device of their own. In English and Persian, in both themes,
// every page reloads and the run captures:
//
// - the admin: the table with its role tag, the game menu (seats with
//   Manage, the spectators with role buttons, Stop being admin, Resign),
//   the seat menu on the player's seat and its confirmation, and the seat
//   menu on a bot's seat;
// - the player: the table, the game menu and the resignation's
//   confirmation;
// - the spectator: the table with the spectator panel, and the game menu;
// - at the end, the player's seat given to a bot: the former player's
//   spectator panel.
//
// At 360x740, 390x844, 844x390 and 1280x800 the probe finds no
// horizontal overflow or scroller, no control under 44 px, the menu inside
// the screen, no glyph without a tooltip, nothing animating with
// Animation off and nothing in the top bar running into its neighbour
// (each child's box and content against the others'); and no page logs a console error or failed request. Each
// state is saved as a screenshot for the contact sheet, in
// HG_ADMIN_SHOTS_DIR. A last test checks the dev gallery's leaves.
import { expect, type Browser, type BrowserContext, type Page } from '@playwright/test';
import { mkdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import type { TableState } from '../../web/src/lib/wire/platform.gen';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { quiet } from '../fixtures/layout-probes';
import { newDeviceContext } from '../fixtures/online';
import { choose, latest, openBotTable, openMenu, ready, seatAction } from '../fixtures/table';

const sizes = [
  [360, 740],
  [390, 844],
  [844, 390],
  [1280, 800]
] as const;
const variants = [
  ['en', 'light'],
  ['en', 'dark'],
  ['fa', 'light'],
  ['fa', 'dark']
] as const;
const out = process.env.HG_ADMIN_SHOTS_DIR ?? '../.cache/e2e/hg-admin-shots';

interface Probe {
  overflow: string[];
  small: { act: string; width: number; height: number }[];
  outside: boolean;
  untipped: string[];
  animated: number;
  crowded: string[];
}

const clean: Probe = {
  overflow: [],
  small: [],
  outside: false,
  untipped: [],
  animated: 0,
  crowded: []
};

async function probe(page: Page, selector: string): Promise<Probe> {
  return page.locator(selector).evaluate((root) => {
    const frame = root.closest('dialog') ?? root;
    const box = frame.getBoundingClientRect();
    const panel = root.getBoundingClientRect();
    const shown = (el: Element) =>
      el.getClientRects().length > 0 &&
      el.checkVisibility({ opacityProperty: true, visibilityProperty: true });
    const describe = (el: Element) =>
      [...el.attributes]
        .filter((a) => a.name.startsWith('data-'))
        .map((a) => `[${a.name}=${a.value}]`)
        .join('') || el.tagName.toLowerCase();
    // Horizontal overflow: the page, the table, the root, any horizontal
    // scroller inside it, and anything past the root's sides.
    const overflow: string[] = [];
    if (document.documentElement.scrollWidth > innerWidth + 1) overflow.push('page');
    const table = document.querySelector('[data-table-screen]');
    if (table && table.scrollWidth > table.clientWidth + 1) overflow.push('table');
    if (root.scrollWidth > root.clientWidth + 1) overflow.push('root');
    for (const el of root.querySelectorAll<HTMLElement>('*')) {
      if (!shown(el)) continue;
      const style = getComputedStyle(el);
      if (
        (style.overflowX === 'auto' || style.overflowX === 'scroll') &&
        el.scrollWidth > el.clientWidth + 1
      )
        overflow.push(`scroller ${describe(el)}`);
      const r = el.getBoundingClientRect();
      if (r.width > 0 && (r.left < panel.left - 1 || r.right > panel.right + 1))
        overflow.push(`past the side ${describe(el)}`);
    }
    const small = [...root.querySelectorAll<HTMLElement>('button:not(:disabled), [role="tab"]')]
      .filter((el) => !el.closest('[inert]') && shown(el))
      .map((el) => ({
        act: el.dataset['act'] ?? el.tagName,
        width: Math.round(el.getBoundingClientRect().width),
        height: Math.round(el.getBoundingClientRect().height)
      }))
      .filter((r) => r.width < 43.9 || r.height < 43.9);
    const untipped = [...root.querySelectorAll('svg, [role="img"]')]
      .filter(
        (el) =>
          !el.parentElement?.closest('svg') &&
          !el.closest('[data-tooltip]') &&
          !el.closest('[inert]') &&
          shown(el)
      )
      .map((el) => describe(el.parentElement ?? el));
    // The top bar (the root's, else the page's): no child, with all of its
    // content, may reach into another, so a label that runs past its own
    // box (or is clipped there) counts too.
    const crowded: string[] = [];
    const bar = root.querySelector('[data-top-bar]') ?? document.querySelector('[data-top-bar]');
    if (bar && shown(bar)) {
      const name = (el: Element) =>
        el.getAttribute('data-act') ?? (el.className || el.tagName.toLowerCase());
      const extent = (el: Element) => {
        const r = el.getBoundingClientRect();
        const range = document.createRange();
        range.selectNodeContents(el);
        const c = range.getBoundingClientRect();
        if (c.width === 0 || c.height === 0) return r;
        return {
          left: Math.min(r.left, c.left),
          right: Math.max(r.right, c.right),
          top: Math.min(r.top, c.top),
          bottom: Math.max(r.bottom, c.bottom)
        };
      };
      const kids = [...bar.children].filter(shown).map((el) => ({ el, box: extent(el) }));
      for (const [i, a] of kids.entries())
        for (const b of kids.slice(i + 1)) {
          const across = Math.min(a.box.right, b.box.right) - Math.max(a.box.left, b.box.left);
          const down = Math.min(a.box.bottom, b.box.bottom) - Math.max(a.box.top, b.box.top);
          if (across > 0.5 && down > 0.5) crowded.push(`${name(a.el)} over ${name(b.el)}`);
        }
    }
    return {
      overflow: overflow.slice(0, 10),
      small,
      outside:
        box.x < -1 || box.y < -1 || box.right > innerWidth + 1 || box.bottom > innerHeight + 1,
      untipped: untipped.slice(0, 10),
      animated: document
        .getAnimations()
        .filter(
          (a) => a.playState === 'running' && Number(a.effect?.getComputedTiming().duration) > 1
        ).length,
      crowded
    };
  });
}

async function named(browser: Browser, info: Parameters<typeof newDeviceContext>[1], name: string) {
  const context = await newDeviceContext(browser, info);
  await context.addInitScript((n) => localStorage.setItem('ajayeb:name', n), name);
  const page = await context.newPage();
  await page.setViewportSize({ width: 390, height: 844 });
  return { context, page };
}

async function setVariant(page: Page, locale: string, theme: string): Promise<void> {
  await page.evaluate(
    ({ l, t }) => {
      const settings = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}');
      localStorage.setItem('ajayeb:settings', JSON.stringify({ ...settings, locale: l }));
      localStorage.setItem('ajayeb:theme', t);
    },
    { l: locale, t: theme }
  );
  await page.setViewportSize({ width: 390, height: 844 });
  await page.reload();
  await ready(page);
  await expect(page.locator('html')).toHaveAttribute('data-theme', theme);
  await expect(page.locator('html')).toHaveAttribute('dir', locale === 'fa' ? 'rtl' : 'ltr');
}

for (const seats of [3, 5, 7] as const) {
  test.describe(`haftgane admin geometry ${seats} seats`, () => {
    test('admin, player and spectator, four variants, four viewports', async ({
      browser
    }, info) => {
      test.setTimeout(900_000);
      mkdirSync(out, { recursive: true });
      const contexts: BrowserContext[] = [];
      try {
        const admin = await named(browser, info, 'Roshan');
        const player = await named(browser, info, 'Kian');
        const watcher = await named(browser, info, 'Sara');
        contexts.push(admin.context, player.context, watcher.context);
        const [a, b, c] = [admin.page, player.page, watcher.page];
        const problems = [a, b, c].map(watchHygiene);
        const probes: Record<string, unknown>[] = [];

        const code = await openBotTable(a, seats, 'Roshan');
        for (const p of [b, c]) {
          await p.goto(`/play/${code}`);
          await expect(p.locator('[data-spectator-panel]')).toBeVisible();
        }
        const start: TableState = await latest(a);
        const userB = (await latest(b)).you.userId;
        const seatA = start.you.seat!;
        const seatB = start.seats.find((s) => s.kind === 'bot')!.seat;
        const botSeat = start.seats.find((s) => s.kind === 'bot' && s.seat !== seatB)!.seat;
        await seatAction(a, seatB, `seat-user:${seatB}:${userB}`);
        await a.locator('[data-act="menu-close"]').click();
        await expect.poll(async () => (await latest(b)).you.seat).toBe(seatB);
        // One turn played, so the spectator panel has a last turn to list.
        const keyA = (await latest(a)).view.you!.choices[0]!.key;
        await choose(a, keyA);
        await choose(a, 'confirm');
        // The player kept the bot's pick; confirm one if it is not confirmed.
        const kept = await latest(b);
        if (kept.batch === 0 && !kept.pending?.confirm) {
          await choose(b, kept.view.you!.choices[0]!.key);
          await choose(b, 'confirm');
        }
        await expect.poll(async () => (await latest(c)).batch).toBeGreaterThan(0);
        const followed = (seatA + 1) % seats;

        async function capture(
          page: Page,
          who: string,
          state: string,
          selector: string
        ): Promise<void> {
          for (const [width, height] of sizes) {
            await page.setViewportSize({ width, height });
            await page
              .locator(selector)
              .evaluate((root) => (root.querySelector('[data-sheet-body]') ?? root).scrollTo(0, 0));
            await quiet(page);
            const p = await probe(page, selector);
            const v = await page.evaluate(() => [
              document.documentElement.lang,
              document.documentElement.dataset['theme']
            ]);
            const base = `admin-${seats}p-${who}-${state}-${width}x${height}-${v[0]}-${v[1]}`;
            probes.push({ seats, who, state, width, height, locale: v[0], theme: v[1], ...p });
            await page.screenshot({ path: path.join(out, `${base}.png`) });
            const scrolls = await page.locator(selector).evaluate((root) => {
              const body = root.querySelector('[data-sheet-body]') ?? root;
              return body.scrollHeight > body.clientHeight + 1;
            });
            if (width === 390 && scrolls) {
              await page.locator(selector).evaluate((root) => {
                const body = root.querySelector('[data-sheet-body]') ?? root;
                body.scrollTo(0, body.scrollHeight);
              });
              await quiet(page);
              await page.screenshot({ path: path.join(out, `${base}-scrolled.png`) });
            }
            expect(p, `${seats} seats ${who} ${state} ${width}x${height} ${v}`).toEqual(clean);
          }
          await page.setViewportSize({ width: 390, height: 844 });
        }

        const menu = '[data-table-menu] [role="dialog"]';
        for (const [locale, theme] of variants) {
          for (const p of [a, b, c]) await setVariant(p, locale, theme);

          // The admin.
          await capture(a, 'admin', 'table', '[data-table-screen]');
          await openMenu(a);
          await capture(a, 'admin', 'menu', menu);
          await a.locator(`[data-act="seat-menu:${seatB}"]`).click();
          await expect(a.locator(`[data-seat-menu="${seatB}"]`)).toBeVisible();
          await capture(a, 'admin', 'seat-player', '[data-seat-menu]');
          await a.locator(`[data-act="seat-bot:${seatB}:easy"]`).click();
          await expect(a.locator('[data-seat-menu] [data-admin-confirm]')).toBeVisible();
          await capture(a, 'admin', 'seat-confirm', '[data-seat-menu]');
          await a.locator('[data-act="admin-back"]').click();
          await a.locator('[data-act="seat-menu-close"]').click();
          await a.locator(`[data-act="seat-menu:${botSeat}"]`).click();
          await expect(a.locator(`[data-seat-menu="${botSeat}"]`)).toBeVisible();
          await capture(a, 'admin', 'seat-bot', '[data-seat-menu]');
          await a.locator('[data-act="seat-menu-close"]').click();
          await a.locator('[data-act="menu-close"]').click();

          // The player.
          await capture(b, 'player', 'table', '[data-table-screen]');
          await openMenu(b);
          await capture(b, 'player', 'menu', menu);
          await b.locator('[data-table-menu] [data-act="resign"]').click();
          await expect(b.locator('[data-table-menu] [data-admin-confirm]')).toBeVisible();
          await capture(b, 'player', 'resign', menu);
          await b.locator('[data-act="admin-back"]').click();
          await b.locator('[data-act="menu-close"]').click();

          // The spectator, following the seat after the creator's.
          await c.locator(`[data-act="follow:${followed}"]`).click();
          await expect(c.locator('[data-neighbour="left"]')).toHaveAttribute(
            'data-seat',
            String((followed + 1) % seats)
          );
          await capture(c, 'spectator', 'table', '[data-table-screen]');
          await capture(c, 'spectator', 'panel', '[data-spectator-panel]');
          await openMenu(c);
          await capture(c, 'spectator', 'menu', menu);
          await c.locator('[data-act="menu-close"]').click();
        }

        // The former player, once a bot plays their seat.
        await seatAction(a, seatB, `seat-bot:${seatB}:easy`);
        await expect(b.locator(`[data-former-seat="${seatB}"]`)).toBeVisible();
        for (const [locale, theme] of variants) {
          await setVariant(b, locale, theme);
          await expect(b.locator(`[data-former-seat="${seatB}"]`)).toBeVisible();
          await capture(b, 'former', 'table', '[data-table-screen]');
        }

        writeFileSync(
          path.join(out, `probes-${seats}p.json`),
          JSON.stringify({ probes, problems }, null, 2)
        );
        for (const list of problems) expect(list).toEqual([]);
      } finally {
        for (const context of contexts) await context.close();
      }
    });
  });
}

test.describe('haftgane admin gallery', () => {
  test('the gallery shows the admin leaves and opens both seat menus', async ({ page }) => {
    mkdirSync(out, { recursive: true });
    const problems = watchHygiene(page);
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto('/dev/gallery');
    const section = page.locator('[data-gallery-admin]');
    await section.scrollIntoViewIfNeeded();
    await expect(section.locator('[data-role-tag]')).toHaveCount(6 + 2 + 2 + 2);
    await expect(section.locator('[data-spectator-panel] [data-act^="follow:"]')).toHaveCount(7);
    await expect(section.locator('[data-previous-seat]')).toHaveCount(2);
    await quiet(page);
    await section.screenshot({ path: path.join(out, 'gallery-admin-390x844.png') });
    await section.locator('[data-spectator-panel]').scrollIntoViewIfNeeded();
    await quiet(page);
    expect(await probe(page, '[data-gallery-admin] [data-spectator-panel]')).toEqual(clean);
    const opens = section.locator(':scope > .row button');
    for (const [i, seat] of [
      [0, 2],
      [1, 3]
    ] as const) {
      await opens.nth(i).click();
      await expect(page.locator(`[data-seat-menu="${seat}"]`)).toBeVisible();
      await quiet(page);
      expect(await probe(page, '[data-seat-menu]')).toEqual(clean);
      await page.screenshot({ path: path.join(out, `gallery-seat-menu-${seat}-390x844.png`) });
      await page.locator('[data-act="seat-menu-close"]').click();
      await expect(page.locator('[data-seat-menu]')).toHaveCount(0);
    }
    expect(problems).toEqual([]);
  });
});
