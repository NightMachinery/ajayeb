// Haftgane's reveal and turn recap geometry (M18.reveal-and-recap): one
// live table each of 3, 5 and 7 seats, started from Home's lobby by its
// creator with bots, Settings' "Show opponent move" on, and a friend
// watching, each on a device of their own. In English and Persian, in both
// themes, both pages reload and the creator plays a turn (a reload restates
// no reveal, so each variant reveals one live), and the run captures:
//
// - the player: the recap over the board as it opens by itself, then the
//   board with every seat's reveal tag once it closes;
// - the spectator: the board with the tags, and the spectator panel with
//   the same recap.
//
// At 360x740, 390x844, 844x390 and 1280x800 the probe finds no
// horizontal overflow or scroller, no control under 44 px, the recap
// inside the screen, no glyph without a tooltip, nothing animating with
// Animation off and nothing in the top bar running into its neighbour
// (each child's box and content against the others'); and no page logs a console error or failed request. Each
// state is saved as a screenshot for the contact sheet, in
// HG_RECAP_SHOTS_DIR. A last test checks the dev gallery's leaves.
import { expect, type Browser, type BrowserContext, type Page } from '@playwright/test';
import { mkdirSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { quiet } from '../fixtures/layout-probes';
import { newDeviceContext } from '../fixtures/online';
import {
  choose,
  dismissMilitary,
  dismissRecap,
  latest,
  openBotTable,
  ready
} from '../fixtures/table';

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
const out = process.env.HG_RECAP_SHOTS_DIR ?? '../.cache/e2e/hg-recap-shots';

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
    const box = root.getBoundingClientRect();
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
      if (r.width > 0 && (r.left < box.left - 1 || r.right > box.right + 1))
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

async function named(
  browser: Browser,
  info: Parameters<typeof newDeviceContext>[1],
  name: string,
  recap: boolean
) {
  const context = await newDeviceContext(browser, info);
  await context.addInitScript(
    ({ n, on }) => {
      localStorage.setItem('ajayeb:name', n);
      if (!on) return;
      const settings = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}');
      localStorage.setItem(
        'ajayeb:settings',
        JSON.stringify({ ...settings, showOpponentMove: true })
      );
    },
    { n: name, on: recap }
  );
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

/**
 * The creator plays one turn, preferring a card bought from a neighbour,
 * then a sale, then a building, so the recap has coins to show.
 */
async function playTurn(page: Page): Promise<number> {
  await dismissMilitary(page);
  const f = await latest(page);
  const cs = f.view.you!.choices;
  const pick =
    cs.find((c) => c.kind === 'build' && c.left + c.right > 0) ??
    cs.find((c) => c.kind === 'sell') ??
    cs.find((c) => c.kind === 'build') ??
    cs[0]!;
  await choose(page, pick.key);
  await choose(page, 'confirm');
  await expect.poll(async () => (await latest(page)).batch).toBeGreaterThan(f.batch);
  await ready(page);
  return (await latest(page)).batch;
}

for (const seats of [3, 5, 7] as const) {
  test.describe(`haftgane recap geometry ${seats} seats`, () => {
    test('player and spectator, four variants, four viewports', async ({ browser }, info) => {
      test.setTimeout(900_000);
      mkdirSync(out, { recursive: true });
      const contexts: BrowserContext[] = [];
      try {
        const player = await named(browser, info, 'Roshan', true);
        const watcher = await named(browser, info, 'Sara', false);
        contexts.push(player.context, watcher.context);
        const [a, c] = [player.page, watcher.page];
        const problems = [a, c].map(watchHygiene);
        const probes: Record<string, unknown>[] = [];

        const code = await openBotTable(a, seats, 'Roshan');
        await c.goto(`/play/${code}`);
        await expect(c.locator('[data-spectator-panel]')).toBeVisible();

        async function capture(
          page: Page,
          who: string,
          state: string,
          selector: string
        ): Promise<void> {
          for (const [width, height] of sizes) {
            await page.setViewportSize({ width, height });
            await page.locator(selector).evaluate((root) => root.scrollTo(0, 0));
            await quiet(page);
            const p = await probe(page, selector);
            const v = await page.evaluate(() => [
              document.documentElement.lang,
              document.documentElement.dataset['theme']
            ]);
            const base = `recap-${seats}p-${who}-${state}-${width}x${height}-${v[0]}-${v[1]}`;
            probes.push({ seats, who, state, width, height, locale: v[0], theme: v[1], ...p });
            await page.screenshot({ path: path.join(out, `${base}.png`) });
            const scrolls = await page
              .locator(selector)
              .evaluate((root) => root.scrollHeight > root.clientHeight + 1);
            if (width === 390 && scrolls) {
              await page.locator(selector).evaluate((root) => root.scrollTo(0, root.scrollHeight));
              await quiet(page);
              await page.screenshot({ path: path.join(out, `${base}-scrolled.png`) });
            }
            expect(p, `${seats} seats ${who} ${state} ${width}x${height} ${v}`).toEqual(clean);
          }
          await page.setViewportSize({ width: 390, height: 844 });
        }

        for (const [locale, theme] of variants) {
          for (const p of [a, c]) await setVariant(p, locale, theme);
          const batch = await playTurn(a);
          await expect.poll(async () => (await latest(c)).batch).toBe(batch);
          await ready(c);

          // The player: the recap as it opened, then the tagged board.
          const layer = a.locator('[data-recap-layer]');
          await expect(layer.locator(`[data-turn-recap="${batch}"]`)).toBeVisible();
          await capture(a, 'player', 'recap', '[data-recap-layer]');
          await dismissRecap(a);
          await expect(a.locator('[data-board-reveal]').first()).toBeAttached();
          await capture(a, 'player', 'board', '[data-table-screen]');

          // The spectator: the tagged board and the panel's recap.
          await expect(c.locator('[data-board-reveal]').first()).toBeAttached();
          await capture(c, 'spectator', 'board', '[data-table-screen]');
          await expect(
            c.locator(`[data-spectator-panel] [data-turn-recap="${batch}"]`)
          ).toBeVisible();
          await capture(c, 'spectator', 'panel', '[data-spectator-panel]');
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

test.describe('haftgane recap gallery', () => {
  test('the gallery shows the reveal leaves and opens the recap', async ({ page }) => {
    mkdirSync(out, { recursive: true });
    const problems = watchHygiene(page);
    await page.setViewportSize({ width: 390, height: 844 });
    await page.goto('/dev/gallery');
    const section = page.locator('[data-gallery-reveal]');
    await section.scrollIntoViewIfNeeded();
    await expect(section.locator('[data-turn-recap]')).toHaveCount(2);
    await expect(section.locator('[data-recap-change]')).toHaveCount(2 * 2);
    await quiet(page);
    await section.screenshot({ path: path.join(out, 'gallery-reveal-390x844.png') });
    // The section is taller than the screen, so only the overlay must fit.
    expect({ ...(await probe(page, '[data-gallery-reveal]')), outside: false }).toEqual(clean);
    await section.locator('[data-act="recap"]').click();
    await expect(page.locator('[data-recap-layer]')).toBeVisible();
    await quiet(page);
    expect(await probe(page, '[data-recap-layer]')).toEqual(clean);
    await page.screenshot({ path: path.join(out, 'gallery-recap-layer-390x844.png') });
    await page.keyboard.press('Escape');
    await expect(page.locator('[data-recap-layer]')).toHaveCount(0);
    expect(problems).toEqual([]);
  });
});
