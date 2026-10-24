// M18's acceptance playthroughs that no earlier Haftgane spec holds
// (M18.table-playthroughs; docs/testing.md lists which spec covers each
// acceptance line).
//
// The creator alone with six bots, once under each end policy: Home's New
// table at seven seats, Start, and a spectator joins the running table.
// The creator plays every point it is asked to (a card build when one is
// offered, else a sale, else the first choice), so nothing depends on the
// deal, through all three Ages to the score sheet, which both pages show
// with a row for every seat. Every frame each page received passes the
// pending-ownership oracle, nothing logs a console error or a failed
// request, and the frames and the room record are written for the Go leak
// oracle (TestScoresBrowserFrames in server/internal/tableroom), which
// checks them frame by frame, seated player and spectator, under both
// policies.
//
// The temporary admin made by absence (@slow, left out of CI): the
// creator and a second human start a table with bots, and the creator's
// tab closes. Five minutes later, the server's AwayLimit (a constant: no
// test clock or development endpoint changes it, and the e2e server gets
// no backdoor), the second human becomes the temporary admin, sees the
// role tag, and uses a seat power, while the frames give them no
// promotion. The creator comes back, and the temporary admin is demoted
// at once. The made-at-once case, when the last real admin steps down,
// is in haftgane-table.full.spec.ts; the clock-driven rule is also
// covered in Go (TestRolesAndTemporaryAdmin in server/internal/tableroom,
// TestTemporaryAdmin in server/internal/roles).
import { expect, type BrowserContext, type Page } from '@playwright/test';
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import type { TableState } from '../../web/src/lib/wire/platform.gen';
import { test } from '../fixtures/base';
import { dataDir } from '../fixtures/global-setup';
import { watchHygiene } from '../fixtures/hygiene';
import { createTableFromHome, lobbyReady } from '../fixtures/lobby';
import { newDeviceContext } from '../fixtures/online';
import {
  choose,
  dismissMilitary,
  latest,
  openBotTable,
  openMenu,
  pendingOwnership,
  ready,
  seatAction,
  traffic
} from '../fixtures/table';

const SEATS = 7;

/**
 * Plays the page's seat until the game is over: at each point that waits
 * for this seat, a card build when one is offered (no stage, so no wonder
 * decision follows), else a sale, else the point's first choice (a side).
 */
async function playToEnd(page: Page): Promise<TableState> {
  for (;;) {
    const f = await latest(page);
    if (f.status === 'finished') return f;
    const me = f.you.seat!;
    if (f.point?.required.includes(me) && !f.point.confirmed.includes(me)) {
      const cs = f.view.you!.choices;
      const pick =
        cs.find((c) => c.kind === 'build') ?? cs.find((c) => c.kind === 'sell') ?? cs[0]!;
      await choose(page, pick.key);
      await choose(page, 'confirm');
    }
    await expect
      .poll(async () => {
        const g = await latest(page);
        return g.batch !== f.batch || g.status === 'finished';
      })
      .toBe(true);
  }
}

/** The score sheet after the last conflicts: a row for every seat, with its total. */
async function checkSheet(page: Page, f: TableState): Promise<void> {
  await dismissMilitary(page);
  const sheet = page.locator('[data-score-sheet]');
  await expect(sheet).toBeVisible();
  await expect(sheet.locator('[data-score-row]')).toHaveCount(SEATS);
  for (const [seat, score] of f.view.result!.scores.entries())
    await expect(sheet.locator(`[data-score-row="${seat}"]`)).toHaveAttribute(
      'data-total',
      String(score.total)
    );
}

async function named(context: BrowserContext, name: string): Promise<Page> {
  await context.addInitScript((n) => localStorage.setItem('ajayeb:name', n), name);
  return context.newPage();
}

test.describe(
  'Haftgane playthroughs',
  { tag: ['@full', '@mobile', '@haftgane-playthrough'] },
  () => {
    for (const reveal of [true, false]) {
      const policy = reveal ? 'revealed' : 'hidden';
      test(`the creator alone with six bots and a spectator, to the ${policy} end`, async ({
        page,
        browser
      }, info) => {
        test.setTimeout(600_000);
        const out = process.env.HG_PLAYTHROUGH_LIVE_DIR ?? '../.cache/e2e/hg-playthrough-live';
        mkdirSync(out, { recursive: true });
        await page.addInitScript(() => localStorage.setItem('ajayeb:name', 'Ana'));
        const hygiene = [watchHygiene(page)];
        const hostLog = traffic(page);
        const watcherContext = await newDeviceContext(browser, info);
        try {
          const watcher = await named(watcherContext, 'Watcher');
          hygiene.push(watchHygiene(watcher));
          const watcherLog = traffic(watcher);

          const code = await openBotTable(page, SEATS, 'Ana', async () => {
            const select = page.locator('[data-act="lobby-reveal"]');
            if ((await select.inputValue()) !== String(reveal)) {
              await select.selectOption(String(reveal));
              await lobbyReady(page);
            }
            await expect(select).toHaveValue(String(reveal));
          });
          const start = await latest(page);
          expect(start.startingHumanCount).toBe(1);
          expect(start.finalReveal).toBe(reveal);
          expect(start.seats.filter((s) => s.kind === 'bot')).toHaveLength(SEATS - 1);
          await watcher.goto(`/play/${code}`);
          await expect(watcher.locator('[data-watching]')).toBeVisible();
          expect((await latest(watcher)).you.seat).toBeUndefined();

          const end = await playToEnd(page);
          expect(end.view.result!.scores).toHaveLength(SEATS);
          expect(end.view.canDownload).toBe(reveal);
          await checkSheet(page, end);
          await expect.poll(async () => (await latest(watcher)).status).toBe('finished');
          await checkSheet(watcher, await latest(watcher));

          for (const [name, log] of [
            ['the creator', hostLog],
            ['the spectator', watcherLog]
          ] as const)
            expect(pendingOwnership(name, log)).toEqual([]);
          for (const problems of hygiene) expect(problems).toEqual([]);

          const record = JSON.parse(
            readFileSync(path.join(dataDir, 'rooms', `${code}.json`), 'utf8')
          );
          writeFileSync(
            path.join(out, `frames-${SEATS}-${policy}-${info.project.name}.json`),
            JSON.stringify({ record, logs: [hostLog, watcherLog] })
          );
        } finally {
          await watcherContext.close();
        }
      });
    }

    test('a temporary admin is made after five minutes away and demoted on return @slow', async ({
      browser
    }, info) => {
      test.setTimeout(600_000);
      const contexts: BrowserContext[] = [];
      try {
        for (let i = 0; i < 2; i++) contexts.push(await newDeviceContext(browser, info));
        const a = await named(contexts[0]!, 'Ana');
        const b = await named(contexts[1]!, 'Bo');
        const hygieneB = watchHygiene(b);
        const logB = traffic(b);

        const code = await createTableFromHome(a, 'Ana');
        await b.goto(`/play/${code}`);
        await expect
          .poll(() =>
            a.evaluate(
              () => window.__ajayebTable?.lobby()?.seats.filter((s) => s.kind === 'human').length
            )
          )
          .toBe(2);
        await choose(a, 'lobby-start');
        const start = await latest(b);
        const userA = (await latest(a)).you.userId;
        const userB = start.you.userId;
        const seatB = start.you.seat!;
        const bot = start.seats.find((s) => s.kind === 'bot')!;

        // The creator's tab closes; the room sees them away.
        await a.close();
        const away = Date.now();
        await expect
          .poll(async () => (await latest(b)).humans.find((h) => h.userId === userA)?.connected)
          .toBe(false);
        expect((await latest(b)).humans.find((h) => h.userId === userB)?.temporary).toBe(false);

        // Five minutes on, the connected seated human takes the role.
        await expect
          .poll(async () => (await latest(b)).humans.find((h) => h.userId === userB)?.temporary, {
            timeout: 420_000,
            intervals: [5_000]
          })
          .toBe(true);
        expect(Date.now() - away, 'the role waited out the absence').toBeGreaterThan(290_000);
        await expect(
          b.locator(`[data-seat="${seatB}"] [data-role-tag="temporary"]`).first()
        ).toBeVisible();
        const made = await latest(b);
        expect(made.you.canPromote).toBeUndefined();
        expect(made.humans.find((h) => h.userId === userB)?.admin).toBe(false);

        // The seat powers: a bot's seat takes another level.
        const level = made.botLevels.find((l) => l !== bot.bot)!;
        await seatAction(b, bot.seat, `seat-bot:${bot.seat}:${level}`);
        await expect
          .poll(async () => (await latest(b)).seats[bot.seat])
          .toMatchObject({ kind: 'bot', bot: level });
        // No promotion: the away creator's seat menu offers seat powers only.
        const seatA = made.seats.find((s) => s.userId === userA)!.seat;
        await openMenu(b);
        await b.locator(`[data-act="seat-menu:${seatA}"]`).click();
        const menu = b.locator(`[data-seat-menu="${seatA}"]`);
        await expect(menu.locator(`[data-act^="seat-bot:${seatA}:"]`).first()).toBeVisible();
        await expect(menu.locator('[data-act^="role-"]')).toHaveCount(0);
        await b.locator('[data-act="seat-menu-close"]').click();
        await b.locator('[data-act="menu-close"]').click();

        // The creator comes back: the temporary admin is demoted at once.
        const back = await contexts[0]!.newPage();
        await back.goto(`/play/${code}`);
        await ready(back);
        await expect
          .poll(async () => (await latest(b)).humans.find((h) => h.userId === userB)?.temporary)
          .toBe(false);
        await expect(b.locator('[data-role-tag="temporary"]')).toHaveCount(0);
        expect((await latest(back)).humans.find((h) => h.userId === userA)?.admin).toBe(true);

        expect(pendingOwnership('the second human', logB)).toEqual([]);
        expect(hygieneB).toEqual([]);
      } finally {
        for (const context of contexts) await context.close();
      }
    });
  }
);
