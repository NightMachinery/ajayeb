// Every submitted key comes from an actual sheet control. The two humans
// keep the point open long enough to withdraw; three bots fill the table.
import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { newDeviceContext } from '../fixtures/online';
import { createTableFromHome, lobbyReady } from '../fixtures/lobby';
import { latest, pendingOwnership, ready, traffic } from '../fixtures/table';
import { watchHygiene } from '../fixtures/hygiene';
import type { Choice } from '../../web/src/lib/wire/haftgane.gen';

async function open(page: Page, c: Choice): Promise<void> {
  await ready(page);
  await page.locator(`[data-act="hand:${c.design}"]`).first().click();
  await expect(page.locator('[data-card-sheet]')).toBeVisible();
  if (c.kind === 'build') {
    const options = page.locator('[data-card-sheet] [data-pick^="build:"]');
    if ((await options.count()) > 1 && (await options.last().getAttribute('data-pick')) === c.key)
      await expect(options.last()).toHaveAttribute('aria-checked', 'false');
  }
  await page.locator(`[data-pick="${c.key}"]`).click();
  await ready(page);
}
async function commit(page: Page, key: string): Promise<void> {
  await page.locator(`[data-card-sheet] .sheet-footer [data-act="${key}"]`).click();
  await expect(page.locator('[data-card-sheet]')).toHaveCount(0);
  await ready(page);
}

test('five seats play payments, stages, sales and Change through the sheet @full @haftgane-sheet', async ({
  browser
}, info) => {
  test.setTimeout(300_000);
  const aCtx = await newDeviceContext(browser, info);
  const bCtx = await newDeviceContext(browser, info);
  const cCtx = await newDeviceContext(browser, info);
  try {
    for (const [ctx, name] of [
      [aCtx, 'Ana'],
      [bCtx, 'Bo'],
      [cCtx, 'Cy']
    ] as const) {
      await ctx.addInitScript((n) => {
        localStorage.setItem('ajayeb:name', n);
        const settings = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}');
        localStorage.setItem(
          'ajayeb:settings',
          JSON.stringify({ ...settings, confirmMoves: 'never' })
        );
      }, name);
    }
    const a = await aCtx.newPage(),
      b = await bCtx.newPage(),
      c = await cCtx.newPage();
    const logs = [traffic(a), traffic(b), traffic(c)];
    const problems = [watchHygiene(a), watchHygiene(b), watchHygiene(c)];
    const code = await createTableFromHome(a, 'Ana');
    await a.locator('[data-act="lobby-seats"]').selectOption('5');
    await lobbyReady(a);
    await b.goto(`/play/${code}`);
    await expect
      .poll(
        async () =>
          (await a.evaluate(() => window.__ajayebTable!.lobby()))?.seats.filter(
            (s) => s.kind === 'human'
          ).length
      )
      .toBe(2);
    await a.locator('[data-act="lobby-start"]').click();
    await latest(a);
    await latest(b);
    await c.goto(`/play/${code}`);
    await latest(c);
    let sold = false,
      staged = false,
      changed = false,
      nonDefault = false,
      turns = 0;
    for (
      let step = 0;
      step < 24 && !(sold && staged && changed && nonDefault && turns >= 3);
      step++
    ) {
      const before = await latest(a);
      if (before.status === 'finished') break;
      for (const page of [a, b]) {
        const frame = await latest(page);
        if (frame.you.seat === undefined || !frame.point?.required.includes(frame.you.seat))
          continue;
        const cs = frame.view.you!.choices;
        const grouped = cs
          .filter((x) => x.kind === 'build')
          .filter((x) => cs.filter((y) => y.kind === 'build' && y.design === x.design).length > 1);
        const options = grouped.filter((x) => x.design === grouped[0]?.design);
        let pick = options.length > 1 && !nonDefault ? options.at(-1) : undefined;
        const multi = !!pick;
        pick ??= !staged && turns > 0 ? cs.find((x) => x.kind === 'stage') : undefined;
        pick ??= !sold ? cs.find((x) => x.kind === 'sell') : undefined;
        pick ??= cs.find((x) => x.kind === 'build') ?? cs[0];
        expect(pick).toBeDefined();
        if (pick!.kind === 'side' || pick!.kind === 'build-discarded') {
          await page.locator(`[data-act="${pick!.key}"]`).click();
          await ready(page);
          await page.locator('[data-act="confirm"]').click();
          await ready(page);
          continue;
        }
        await open(page, pick!);
        if (multi) {
          const defaultKey = await page
            .locator('[data-card-sheet] .sheet-footer [data-act^="build:"]')
            .getAttribute('data-act');
          expect(defaultKey).toBe(pick!.key);
          nonDefault = true;
        }
        await commit(page, pick!.key);
        if (
          !changed &&
          page === a &&
          before.point?.required.includes((await latest(b)).you.seat!)
        ) {
          const waiting = await latest(a);
          await expect(page.locator('[data-pending]')).toHaveAttribute(
            'data-pending-confirm',
            'true'
          );
          await page.locator('[data-act="withdraw"]').click();
          await ready(page);
          await expect(page.locator('[data-card-sheet]')).toBeVisible();
          expect((await latest(a)).pending?.confirm).toBe(false);
          const other = waiting.view.you!.choices.find(
            (x) => x.design === pick!.design && x.key !== pick!.key
          )!;
          await page.locator(`[data-pick="${other.key}"]`).click();
          await ready(page);
          await commit(page, other.key);
          changed = true;
          pick = other;
        }
        sold ||= pick!.kind === 'sell';
        staged ||= pick!.kind === 'stage';
      }
      await expect.poll(async () => (await latest(a)).point?.ref.id).not.toBe(before.point?.ref.id);
      turns++;
    }
    expect({ sold, staged, changed, nonDefault, severalTurns: turns >= 3 }).toEqual({
      sold: true,
      staged: true,
      changed: true,
      nonDefault: true,
      severalTurns: true
    });
    logs.forEach((log, i) => expect(pendingOwnership(`page-${i}`, log)).toEqual([]));
    expect(problems.flat()).toEqual([]);
  } finally {
    await aCtx.close();
    await bCtx.close();
    await cCtx.close();
  }
});
