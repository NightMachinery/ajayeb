// The end of each Age and of the game over a real table from Home, once
// under each end policy (M18.age-military-and-scores). One human opens a
// three-seat table, Start gives the other seats bots, and a spectator joins
// the running table. The human sells a card every turn, which ends the game
// fastest, and the spec checks, against the server's own frames:
//
// - at each end of Age, the military dialog of the human and the spectator:
//   every conflict and its outcome from the frame's military events, the
//   shields from the frame's seats, the human's own two conflicts, and each
//   seat's military points; after a reload it opens again;
// - at the end, the score sheet after the last conflicts: the verdict, the
//   rows in the server's rank order, and every category of every seat; a
//   row opens that seat's city over it;
// - the log download: the file under the revealed end, and under the hidden
//   end no button, the note, and the endpoint's refusal;
// - the final board with its buttons, and the revealed cards when shown;
// - nothing animates with Animation off, and no console error or failed
//   request happens.
//
// The frames each page received and the room record are written for the
// Go leak oracle (server/internal/tableroom/scores_browser_test.go), which
// checks them frame by frame for the seated player and the spectator.
import { expect, type Locator, type Page } from '@playwright/test';
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs';
import path from 'node:path';
import { test } from '../fixtures/base';
import { dataDir } from '../fixtures/global-setup';
import { watchHygiene } from '../fixtures/hygiene';
import { lobbyReady } from '../fixtures/lobby';
import { newDeviceContext } from '../fixtures/online';
import {
  choose,
  dismissMilitary,
  latest,
  openBotTable,
  pendingOwnership,
  ready,
  traffic,
  type Traffic
} from '../fixtures/table';
import type { TableEvent, TableState } from '../../web/src/lib/wire/platform.gen';

const SEATS = 3;
const COLUMNS = [
  'wonder',
  'treasure',
  'military',
  'blue',
  'yellow',
  'green',
  'purple',
  'total'
] as const;

function militaryOf(events: readonly TableEvent[]): TableEvent[] {
  return events.filter((e) => e.kind === 'military');
}

/**
 * The events of the frame that resolved this batch, as the page received
 * it. Only that frame carries them: the frames that follow in the same
 * batch (the next point's confirmations) send none.
 */
function resolvedEvents(log: readonly Traffic[], batch: number): TableEvent[] {
  for (const { dir, msg } of log) {
    if (dir !== 'recv' || msg.t !== 'table-state') continue;
    const f = msg as unknown as TableState;
    if (f.batch === batch && f.events.length > 0) return f.events;
  }
  return [];
}

interface AgeEnd {
  frame: TableState;
  events: TableEvent[];
}

/** The seat's side of a conflict, as the dialog names it. */
function sideOf(e: TableEvent, seat: number): string {
  if (e.outcome === 'tie') return 'tie';
  const seatWins = e.outcome === 'seat-wins';
  return (e.seat === seat) === seatWins ? 'victory' : 'defeat';
}

async function running(page: Page): Promise<number> {
  return page.evaluate(
    () =>
      document.getAnimations().filter((a) => {
        const timing = a.effect?.getComputedTiming();
        return (
          a.playState === 'running' && typeof timing?.duration === 'number' && timing.duration > 1
        );
      }).length
  );
}

/** Checks the open military dialog against the batch that ended the Age. */
async function checkMilitary(page: Page, { frame: f, events }: AgeEnd): Promise<void> {
  expect(events).toHaveLength(SEATS);
  const dialog = page.locator('[data-age-military]');
  await expect(dialog).toBeVisible();
  await expect(dialog).toHaveAttribute('data-age-military', String(events[0]!.age));
  await expect(dialog.locator('[data-conflict]')).toHaveCount(events.length);
  const seat = (n: number) => f.view.seats[n]!;
  for (const e of events) {
    const row = dialog.locator(`[data-conflict="${e.seat}:${e.other}"]`);
    await expect(row).toHaveAttribute('data-outcome', e.outcome);
    await expect(row.locator('[data-shields]')).toHaveAttribute(
      'data-shields',
      `${seat(e.seat).shields}:${seat(e.other).shields}`
    );
  }
  for (const s of f.view.seats) {
    await expect(dialog.locator(`[data-military-total="${s.seat}"]`)).toHaveAttribute(
      'data-points',
      String(s.score.military)
    );
  }
  const me = f.you.seat;
  if (me === undefined) {
    await expect(dialog.locator('[data-my-conflict]')).toHaveCount(0);
    await expect(dialog.locator('[data-my-military]')).toHaveCount(0);
  } else {
    for (const relation of ['left', 'right'] as const) {
      const them = f.view.seats.find((s) => s.relation === relation)!.seat;
      const e = events.find(
        (x) => (x.seat === me && x.other === them) || (x.seat === them && x.other === me)
      )!;
      const card = dialog.locator(`[data-my-conflict="${relation}"]`);
      await expect(card).toHaveAttribute('data-result', sideOf(e, me));
      await expect(card.locator('[data-shields]')).toHaveAttribute(
        'data-shields',
        `${seat(me).shields}:${seat(them).shields}`
      );
    }
    await expect(dialog.locator('[data-my-military]')).toHaveAttribute(
      'data-my-military',
      String(seat(me).score.military)
    );
  }
  expect(await running(page)).toBe(0);
}

/** Checks the open score sheet against the frame's result. */
async function checkScores(page: Page, f: TableState): Promise<Locator> {
  const r = f.view.result!;
  const sheet = page.locator('[data-score-sheet]');
  await expect(sheet).toBeVisible();
  const me = f.you.seat;
  const verdict =
    me !== undefined && r.winners.includes(me)
      ? r.winners.length === 1
        ? 'you'
        : 'shared'
      : r.winners.length === 1
        ? 'one'
        : 'shared';
  await expect(sheet.locator('[data-verdict]')).toHaveAttribute('data-verdict', verdict);
  const rows = sheet.locator('[data-score-row]');
  await expect(rows).toHaveCount(SEATS);
  const order = r.rank
    .map((rank, seat) => ({ rank, seat }))
    .sort((a, b) => a.rank - b.rank || a.seat - b.seat)
    .map((x) => String(x.seat));
  expect(
    await rows.evaluateAll((els) => els.map((el) => el.getAttribute('data-score-row')))
  ).toEqual(order);
  for (const [seat, score] of r.scores.entries()) {
    const row = sheet.locator(`[data-score-row="${seat}"]`);
    await expect(row).toHaveAttribute('data-rank', String(r.rank[seat]));
    await expect(row).toHaveAttribute('data-total', String(score.total));
    for (const col of COLUMNS) {
      await expect(row.locator(`td[data-col="${col}"]`)).toHaveAttribute(
        'data-value',
        String(score[col])
      );
    }
  }
  expect(await running(page)).toBe(0);
  return sheet;
}

/**
 * Plays the human's seat until a batch after `since` reports conflicts,
 * selling the first card it can each time. The bots confirm at once, so
 * this seat's confirmation resolves each point.
 */
async function playToConflicts(
  page: Page,
  log: readonly Traffic[],
  since: number
): Promise<AgeEnd> {
  for (;;) {
    const f = await latest(page);
    for (let b = since + 1; b <= f.batch; b++) {
      const events = militaryOf(resolvedEvents(log, b));
      if (events.length > 0) {
        expect(b, 'no batch resolves while the dialog is open').toBe(f.batch);
        return { frame: f, events };
      }
    }
    expect(f.status, 'the game ended without the last conflicts').toBe('playing');
    const me = f.you.seat!;
    if (f.point?.required.includes(me) && !f.point.confirmed.includes(me)) {
      const cs = f.view.you!.choices;
      const pick = cs.find((c) => c.kind === 'sell') ?? cs[0]!;
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

/** Waits until the page shows this batch, and reads its conflicts from its log. */
async function atBatch(page: Page, log: readonly Traffic[], batch: number): Promise<AgeEnd> {
  await expect.poll(async () => (await latest(page)).batch).toBe(batch);
  await expect.poll(() => militaryOf(resolvedEvents(log, batch)).length).toBe(SEATS);
  return { frame: await latest(page), events: militaryOf(resolvedEvents(log, batch)) };
}

for (const reveal of [true, false]) {
  const policy = reveal ? 'revealed' : 'hidden';
  test(`the end of each Age and the scores under the ${policy} end @full @haftgane-scores`, async ({
    page,
    browser
  }, info) => {
    test.setTimeout(900_000);
    const out = process.env.HG_SCORES_LIVE_DIR ?? '../.cache/e2e/hg-scores-live';
    mkdirSync(out, { recursive: true });
    await page.addInitScript(() => localStorage.setItem('ajayeb:name', 'Ana'));
    const hygiene = [watchHygiene(page)];
    const hostLog = traffic(page);

    const watcherContext = await newDeviceContext(browser, info);
    try {
      await watcherContext.addInitScript(() => localStorage.setItem('ajayeb:name', 'Watcher'));
      const watcher = await watcherContext.newPage();
      hygiene.push(watchHygiene(watcher));
      const watcherLog = traffic(watcher);

      // Home's New table, the end policy set in the lobby, then Start.
      const code = await openBotTable(page, SEATS, 'Ana', async () => {
        const select = page.locator('[data-act="lobby-reveal"]');
        if ((await select.inputValue()) !== String(reveal)) {
          await select.selectOption(String(reveal));
          await lobbyReady(page);
        }
        await expect(select).toHaveValue(String(reveal));
      });
      expect((await latest(page)).finalReveal).toBe(reveal);
      await watcher.goto(`/play/${code}`);
      await ready(watcher);
      await expect(watcher.locator('[data-watching]')).toBeVisible();

      let since = (await latest(page)).batch;
      for (let age = 1; age <= 3; age++) {
        const ended = await playToConflicts(page, hostLog, since);
        const f = ended.frame;
        since = f.batch;
        expect(ended.events[0]!.age, `the conflicts of Age ${age}`).toBe(age);
        await checkMilitary(page, ended);
        await checkMilitary(watcher, await atBatch(watcher, watcherLog, f.batch));
        if (age === 2 || (age === 3 && reveal)) {
          // A reload at the deciding moment opens the dialog again, from
          // the history the join frame carries.
          await page.reload();
          await ready(page);
          await checkMilitary(page, { frame: await latest(page), events: ended.events });
        }
        if (age < 3) {
          // The next Age waits behind the dialog; the score sheet never shows.
          await expect(page.locator('[data-score-sheet]')).toHaveCount(0);
          await dismissMilitary(page);
          await expect(page.locator('[data-age-military]')).toHaveCount(0);
          // The table menu reopens the same conflicts.
          await page.locator('[data-act="table-menu"]').click();
          await page.locator('[data-act="menu-military"]').click();
          await expect(page.locator('[data-age-military]')).toHaveAttribute(
            'data-age-military',
            String(age)
          );
          await dismissMilitary(page);
          continue;
        }
        expect(f.status).toBe('finished');
        expect(f.view.result).toBeTruthy();
        // The last conflicts come first, then the score sheet.
        await expect(page.locator('[data-score-sheet]')).toHaveCount(0);
        await dismissMilitary(page);
        await dismissMilitary(watcher);
      }

      const end = await latest(page);
      expect(end.view.canDownload).toBe(reveal);
      const sheet = await checkScores(page, end);
      await checkScores(watcher, await latest(watcher));

      // A row opens that seat's city over the sheet, which stays open.
      const other = end.view.seats.find((s) => s.seat !== end.you.seat)!.seat;
      await sheet.locator(`[data-act="score-city:${other}"]`).click();
      await expect(page.locator(`[data-city-sheet="${other}"]`)).toBeVisible();
      await page.locator('[data-act="city-close"]').click();
      await expect(page.locator('[data-city-sheet]')).toHaveCount(0);
      await expect(sheet).toBeVisible();

      if (reveal) {
        const [download] = await Promise.all([
          page.waitForEvent('download'),
          sheet.locator('[data-act="download-log"]').click()
        ]);
        const log = JSON.parse(readFileSync((await download.path())!, 'utf8')) as {
          format: string;
          batches: unknown[];
        };
        expect(log.format).toBe('ajayeb-log/2');
        expect(log.batches.length).toBeGreaterThan(0);
      } else {
        await expect(sheet.locator('[data-act="download-log"]')).toHaveCount(0);
        await expect(sheet.locator('[data-log-hidden]')).toBeVisible();
        const refused = await page.request.get(`/api/rooms/${code}/log`);
        expect(refused.status()).toBe(403);
        expect(((await refused.json()) as { code: string }).code).toBe('log-hidden');
      }

      // The final board, with the controls that reopen the end.
      await sheet.locator('[data-act="final-board"]').click();
      await expect(page.locator('[data-score-sheet]')).toHaveCount(0);
      const actions = page.locator('[data-end-actions]');
      await expect(actions).toBeVisible();
      await expect(actions.locator('[data-act="open-reveal"]')).toHaveCount(reveal ? 1 : 0);
      if (reveal) {
        await actions.locator('[data-act="open-reveal"]').click();
        const shown = page.locator('[data-final-reveal]');
        await expect(shown).toBeVisible();
        const pile = shown.locator('[data-reveal-part="pile"] [data-card]');
        const counts = await pile.evaluateAll((els) =>
          els.map((el) => Number(el.getAttribute('data-count') ?? '1'))
        );
        expect(counts.reduce((a, b) => a + b, 0)).toBe(end.view.reveal!.pile.length);
        await expect(shown.locator('[data-reveal-part="box"] [data-card]')).toHaveCount(
          end.view.reveal!.box.length
        );
        await shown.locator('[data-act="reveal-close"]').click();
        await expect(shown).toHaveCount(0);
      }
      await actions.locator('[data-act="open-military"]').click();
      await expect(page.locator('[data-age-military]')).toHaveAttribute('data-age-military', '3');
      await dismissMilitary(page);
      await actions.locator('[data-act="open-scores"]').click();
      await checkScores(page, end);

      for (const [name, log] of [
        ['the player', hostLog],
        ['the spectator', watcherLog]
      ] as const) {
        expect(pendingOwnership(name, log)).toEqual([]);
      }
      for (const problems of hygiene) expect(problems).toEqual([]);

      const record = JSON.parse(readFileSync(path.join(dataDir, 'rooms', `${code}.json`), 'utf8'));
      writeFileSync(
        path.join(out, `frames-${policy}.json`),
        JSON.stringify({ record, logs: [hostLog, watcherLog] })
      );
    } finally {
      await watcherContext.close();
    }
  });
}
