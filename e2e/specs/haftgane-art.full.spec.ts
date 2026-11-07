// Live art acceptance with optional review captures. Never seed a guild draw:
// the complete public art catalog is independent of what the room dealt.
import { mkdirSync } from 'node:fs';
import path from 'node:path';
import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { choose, dismissMilitary, latest, ready } from '../fixtures/table';
import { createTableFromHome, lobbyReady } from '../fixtures/lobby';
import { newDeviceContext } from '../fixtures/online';
import { hgBoards, hgDesigns, type HgBoard } from '../../web/src/lib/wire/haftgane-catalog.gen';

async function images(page: Page) {
  await expect
    .poll(() =>
      page.locator('img').evaluateAll((els) =>
        els
          .filter((el) => {
            const r = el.getBoundingClientRect();
            return r.width && r.height && r.top < innerHeight && r.bottom > 0;
          })
          .every((el) => el instanceof HTMLImageElement && el.complete && el.naturalWidth > 0)
      )
    )
    .toBe(true);
}

test.describe('Haftgane art', { tag: ['@full', '@haftgane-table'] }, () => {
  test('approved card and intact wonder art through all three Ages', async ({
    page,
    browser
  }, info) => {
    test.setTimeout(240_000);
    const problems = watchHygiene(page);
    const requests: string[] = [];
    page.on('request', (r) => {
      if (r.url().includes('/art/')) requests.push(r.url());
    });
    // Control the other four players so they preserve the one Age II
    // double-resource design needed by our first day-side stage. Every day
    // stage costs two of one raw resource, and each matching double producer
    // is in the five-player deck and costs one coin. Selling banks that coin.
    // At five seats every packet reaches us by turn five; the others can
    // always sell another card before then. Thus we build the producer by
    // Age II turn five and a stage by Age III turn one, for every deal.
    const contexts = await Promise.all(
      Array.from({ length: 4 }, () => newDeviceContext(browser, info))
    );
    try {
      const peers = await Promise.all(
        contexts.map(async (ctx, i) => {
          await ctx.addInitScript((n) => localStorage.setItem('ajayeb:name', `Partner ${n}`), i);
          return ctx.newPage();
        })
      );
      const code = await createTableFromHome(page, 'Painter');
      await page.locator('[data-act="lobby-seats"]').selectOption('5');
      await lobbyReady(page);
      await page.locator('[data-act="lobby-seating"]').selectOption('lobby');
      await lobbyReady(page);
      await page.locator('[data-act="lobby-sides"]').selectOption('day');
      await lobbyReady(page);
      for (const peer of peers) {
        await peer.goto(`/play/${code}`);
        await lobbyReady(peer);
      }
      await expect
        .poll(
          async () =>
            (await page.evaluate(() => window.__ajayebTable!.lobby()))!.seats.filter(
              (s) => s.kind === 'human'
            ).length
        )
        .toBe(5);
      await choose(page, 'lobby-start');
      const start = await latest(page);
      const board = (hgBoards as Record<string, HgBoard>)[
        start.view.seats[start.you.seat!]!.board
      ]!;
      const required = board.day[0]!.cost.res!;
      expect(required).toHaveLength(2);
      expect(required[0]).toBe(required[1]);
      const producer = Object.entries(hgDesigns).find(
        ([, d]) =>
          d.age === 2 &&
          d.effects.some((fx) => fx.kind === 'produce' && fx.n === 2 && fx.res?.[0] === required[0])
      );
      expect(producer).toBeDefined();
      expect(producer![1].cost).toEqual({ coins: 1 });
      const resourceCard = producer![0];
      let productionBuilt = false;
      const seen = new Set<number>();
      let tucked = false;
      const out = process.env.HG_ART_SHOTS_DIR;
      if (out) mkdirSync(out, { recursive: true });
      for (let move = 0; move < 25; move++) {
        await ready(page);
        await dismissMilitary(page);
        const f = await latest(page);
        if (f.status === 'finished') break;
        if (f.view.phase === 'turn' && !seen.has(f.view.age)) {
          seen.add(f.view.age);
          if (f.view.age === 3) expect(f.view.seats[f.you.seat!]!.stages).toBeGreaterThan(0);
          const variants = out
            ? ([
                ['light', 390, 844],
                ['dark', 390, 844],
                ['light', 844, 390],
                ['dark', 844, 390],
                ['light', 1280, 800],
                ['dark', 1280, 800]
              ] as const)
            : ([['light', 390, 844]] as const);
          for (const [theme, width, height] of variants) {
            await page.setViewportSize({ width, height });
            await page.evaluate((value) => {
              document.documentElement.dataset.theme = value;
            }, theme);
            await images(page);
            expect(await page.locator('[data-card-art] img').count()).toBeGreaterThan(0);
            expect(
              await page.evaluate(() => document.documentElement.scrollWidth > innerWidth + 1)
            ).toBe(false);
            if (out)
              await page.screenshot({
                path: path.join(out, `age-${f.view.age}-${width}x${height}-${theme}.png`)
              });
            await page.locator(`[data-act="city:${f.you.seat}"]`).click();
            const board = page.locator('[data-wonder-board]');
            await expect(board).toHaveAttribute(
              'data-wonder-board',
              f.view.seats[f.you.seat!]!.board
            );
            await expect(board.locator('.painting img')).toBeVisible();
            await images(page);
            const picture = await board
              .locator('.painting img')
              .evaluate((img: HTMLImageElement) => ({
                fit: getComputedStyle(img).objectFit,
                ratio: Number(img.getAttribute('width')) / Number(img.getAttribute('height'))
              }));
            expect(picture).toEqual({ fit: 'contain', ratio: 1.5 });
            await expect(board.locator('[data-stage-built="true"]')).toHaveCount(
              f.view.seats[f.you.seat!]!.stages
            );
            expect(await board.locator('.tuck').count()).toBe(f.view.seats[f.you.seat!]!.stages);
            if (out)
              await page.screenshot({
                path: path.join(out, `wonder-age-${f.view.age}-${width}x${height}-${theme}.png`)
              });
            await page.locator('[data-act="city-close"]').click();
          }
        }
        const choices = f.view.you!.choices;
        const pick =
          choices.find((c) => c.kind === 'stage') ??
          choices.find((c) => c.kind === 'build' && c.design === resourceCard) ??
          choices.find((c) => c.kind === 'sell') ??
          choices[0]!;
        await choose(page, pick.key);
        if (pick.kind === 'side' || pick.kind === 'build-discarded') {
          await page.locator('[data-special-confirm]').click();
        } else await choose(page, 'confirm');
        if (pick.kind === 'stage') tucked = true;
        if (pick.kind === 'build' && pick.design === resourceCard) productionBuilt = true;
        for (const peer of peers) {
          const other = await latest(peer);
          await dismissMilitary(peer);
          const sale = other.view.you!.choices.find(
            (c) => c.kind === 'sell' && (productionBuilt || c.design !== resourceCard)
          );
          expect(
            sale,
            'a partner can preserve the required producer until it reaches the viewer'
          ).toBeDefined();
          await choose(peer, sale!.key);
          await choose(peer, 'confirm');
        }
        await expect.poll(async () => (await latest(page)).point?.ref.id).not.toBe(f.point!.ref.id);
      }
      expect([...seen]).toEqual([1, 2, 3]);
      expect(tucked, 'the live game built a stage with a concealed identity').toBe(true);
      expect(requests.some((url) => url.includes('/art/backs/guild'))).toBe(false);
      expect(problems).toEqual([]);
    } finally {
      await Promise.all(contexts.map((ctx) => ctx.close()));
    }
  });
});

// All seven real headers, including the longest name in both locales, at
// the smallest supported phone width. A single text range must stay on one
// line and inside its panel, so nowrap cannot hide an overflowing name.
for (const [locale, longest] of [
  ['en', 'Halikarnassos'],
  ['fa', 'هالیکارناسوس']
] as const) {
  test.describe(`wonder names ${locale}`, { tag: ['@full', '@haftgane-table'] }, () => {
    test.use({ appLocale: locale });
    test('all board names stay unbroken at 360px', async ({ page }) => {
      const problems = watchHygiene(page);
      await page.setViewportSize({ width: 360, height: 844 });
      await page.addInitScript(() => localStorage.setItem('ajayeb:name', 'Painter'));
      await page.goto('/dev/gallery');
      const boards = page.locator('[data-gallery-wonder-boards] [data-wonder-board]');
      await expect(boards).toHaveCount(7);
      await expect(boards.locator('[data-wonder-name]').filter({ hasText: longest })).toHaveCount(
        1
      );
      await page.evaluate(() => document.fonts.ready);
      for (const board of await boards.all()) {
        await board.scrollIntoViewIfNeeded();
        const name = board.locator('[data-wonder-name]');
        const probe = await name.evaluate((el) => {
          const range = document.createRange();
          range.selectNodeContents(el);
          const rects = [...range.getClientRects()].filter((r) => r.width > 0 && r.height > 0);
          const box = el.closest('[data-wonder-board]')!.getBoundingClientRect();
          return {
            text: el.textContent,
            lines: new Set(rects.map((r) => Math.round(r.top))).size,
            inside: rects.every((r) => r.left >= box.left && r.right <= box.right)
          };
        });
        expect(probe.lines, probe.text ?? '').toBe(1);
        expect(probe.inside, probe.text ?? '').toBe(true);
      }
      expect(problems).toEqual([]);
    });
  });
}
