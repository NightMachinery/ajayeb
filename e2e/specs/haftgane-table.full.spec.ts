// A Haftgane table over the real server (started with -haftgane-tables) and
// its development screen: the creator opens a four-seat lobby, a second
// human joins it from another browser, the creator starts it with bots in
// the empty seats, and a spectator joins the running table. Then:
//
// - the creator confirms a pick, reloads, finds it still confirmed, then
//   withdraws it, changes it and confirms the new one;
// - the second human drafts a pick, changes it, drops off the network and
//   comes back to find the changed pick still pending, and confirms it last,
//   which resolves the point;
// - the creator gives the second human's seat to a bot, and that tab
//   becomes a spectator's view.
//
// The client leak boundary: until the point resolves, neither pick the
// second human drafted appears anywhere in the creator's or the
// spectator's page (their WebSocket frames, DOM, local and session
// storage, console, or the table hook's dump). And every table frame any
// of the three pages received shows as its pending pick exactly its own:
// the one its own acknowledged submissions set, or none.
//
// A second test leaves the table for Home and comes back within the same
// page: the requests after the return go on from the page's seq, so the
// table applies them instead of taking them for repeats.
import { expect, type BrowserContext, type Page } from '@playwright/test';
import type { LobbyState, PointRef, TableState } from '../../web/src/lib/wire/platform.gen';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { FrameLog, newDeviceContext } from '../fixtures/online';
import { createTableFromHome } from '../fixtures/lobby';

interface TableHook {
  table(): TableState | null;
  latest(): TableState | null;
  lobby(): LobbyState | null;
  conn(): string;
  pending(): { t: string; seq: number } | null;
  notice(): { kind: string; code: string } | null;
  history(): readonly unknown[];
  ready(): boolean;
}

declare global {
  interface Window {
    __ajayebTable?: TableHook;
  }
}

async function named(context: BrowserContext, name: string): Promise<Page> {
  await context.addInitScript((n) => localStorage.setItem('ajayeb:name', n), name);
  return context.newPage();
}

/** Waits until the page shows the latest frame and takes input. */
async function ready(page: Page): Promise<void> {
  await expect.poll(() => page.evaluate(() => window.__ajayebTable?.ready() ?? false)).toBe(true);
}

async function latest(page: Page): Promise<TableState> {
  await ready(page);
  const frame = await page.evaluate(() => window.__ajayebTable!.latest());
  expect(frame).not.toBeNull();
  return frame!;
}

async function choose(page: Page, act: string): Promise<void> {
  await ready(page);
  await page.locator(`[data-act="${act}"]`).click();
  await ready(page);
}

async function expectPending(page: Page, key: string, confirm: boolean): Promise<void> {
  const pending = page.locator('[data-pending]');
  await expect(pending).toHaveAttribute('data-pending', key);
  await expect(pending).toHaveAttribute('data-pending-confirm', String(confirm));
  expect((await latest(page)).pending).toEqual({ key, confirm });
}

/** Everything a page holds that a pick could leak into. */
async function holdings(page: Page, frames: FrameLog, consoleLines: string[]): Promise<string> {
  const held = await page.evaluate(() => {
    const hook = window.__ajayebTable;
    return JSON.stringify({
      local: { ...localStorage },
      session: { ...sessionStorage },
      hook: hook && {
        table: hook.table(),
        latest: hook.latest(),
        lobby: hook.lobby(),
        history: hook.history()
      }
    });
  });
  return [held, await page.content(), ...frames.frames.map((f) => f.text), ...consoleLines].join(
    '\n'
  );
}

/** One message on a page's sockets, either way, in order. */
interface Traffic {
  dir: 'sent' | 'recv';
  msg: Record<string, unknown>;
}

/** Records every message on a page's game sockets, across reloads and reconnects. */
function traffic(page: Page): Traffic[] {
  const log: Traffic[] = [];
  page.on('websocket', (ws) => {
    if (!new URL(ws.url()).pathname.endsWith('/ws')) return;
    const take =
      (dir: Traffic['dir']) =>
      ({ payload }: { payload: string | Buffer }) => {
        try {
          const text = typeof payload === 'string' ? payload : payload.toString('utf8');
          log.push({ dir, msg: JSON.parse(text) as Record<string, unknown> });
        } catch {
          /* not JSON: no game message */
        }
      };
    ws.on('framesent', take('sent'));
    ws.on('framereceived', take('recv'));
  });
  return log;
}

interface OwnPick {
  key: string;
  confirm: boolean;
  point: PointRef;
}

/**
 * The pending-pick ownership oracle. It models a page's own pick from what
 * the page sent, never from the frames' pending field: each table-pick under
 * its hello's client id and seq, taken as the page's own pick when a frame
 * acknowledges that seq, and gone when the point changes or the page holds
 * no seat. Every table frame's pending must be exactly that pick, or absent
 * when there is none. It returns what disagrees.
 */
function pendingOwnership(name: string, log: readonly Traffic[]): string[] {
  const problems: string[] = [];
  const sent = new Map<string, OwnPick>();
  let clientId = '';
  let own: OwnPick | null = null;
  log.forEach(({ dir, msg }, i) => {
    if (dir === 'sent') {
      if (msg.t === 'hello') clientId = String(msg.clientId);
      if (msg.t === 'table-pick') {
        const pick = msg as unknown as {
          seq: number;
          key: string;
          confirm: boolean;
          point: PointRef;
        };
        sent.set(`${clientId}:${pick.seq}`, {
          key: pick.key,
          confirm: pick.confirm,
          point: pick.point
        });
      }
      return;
    }
    if (msg.t !== 'table-state') return;
    const f = msg as unknown as TableState;
    const acked = f.ack && sent.get(`${f.ack.clientId}:${f.ack.seq}`);
    if (acked) own = acked;
    const ref = f.point?.ref;
    if (
      own &&
      (f.you.seat === undefined || ref?.id !== own.point.id || ref.revision !== own.point.revision)
    ) {
      own = null;
    }
    const want = own ? { key: own.key, confirm: own.confirm } : null;
    const got = f.pending ?? null;
    if (JSON.stringify(got) !== JSON.stringify(want)) {
      problems.push(
        `${name} message ${i}: pending ${JSON.stringify(got)}, its own ${JSON.stringify(want)}`
      );
    }
  });
  return problems;
}

/** The log with message i's pending pick replaced: a contaminated frame. */
function contaminate(
  log: readonly Traffic[],
  i: number,
  pending: TableState['pending']
): Traffic[] {
  return log.map((t, j) => (j === i ? { dir: t.dir, msg: { ...t.msg, pending } } : t));
}

/** Follows a link inside the page, as the app's own links do: no reload. */
async function follow(page: Page, href: string): Promise<void> {
  await page.evaluate((to) => {
    const a = document.createElement('a');
    a.href = to;
    document.body.append(a);
    a.click();
    a.remove();
  }, href);
}

function consoleOf(page: Page): string[] {
  const lines: string[] = [];
  page.on('console', (msg) => lines.push(msg.text()));
  return lines;
}

test.describe('Haftgane table', { tag: ['@full', '@haftgane-table'] }, () => {
  test('picks change, survive a reload and a network blip, and stay private', async ({
    browser
  }, info) => {
    const contexts: BrowserContext[] = [];
    const device = async () => {
      const c = await newDeviceContext(browser, info);
      contexts.push(c);
      return c;
    };
    try {
      const [ctxA, ctxB, ctxC] = [await device(), await device(), await device()];
      const a = await named(ctxA, 'Ana');
      const b = await named(ctxB, 'Bo');
      const c = await named(ctxC, 'Cy');
      const problemsA = watchHygiene(a);
      const problemsC = watchHygiene(c);
      const framesA = new FrameLog(a, 'creator');
      const framesC = new FrameLog(c, 'spectator');
      const trafficA = traffic(a);
      const trafficB = traffic(b);
      const trafficC = traffic(c);
      const consoleA = consoleOf(a);
      const consoleC = consoleOf(c);

      // The lobby: the creator holds the first slot, the second human takes
      // the next open one by joining.
      await createTableFromHome(a);
      await a.waitForURL(/\/play\/[A-Z0-9]+$/);
      const code = a.url().split('/').at(-1)!;
      await b.goto(`/play/${code}`);
      await expect
        .poll(() =>
          a.evaluate(
            () => window.__ajayebTable?.lobby()?.seats.filter((s) => s.kind === 'human').length
          )
        )
        .toBe(2);
      await choose(a, 'lobby-start');

      const startA = await latest(a);
      const startB = await latest(b);
      expect(startA.seats.map((s) => s.kind).sort()).toEqual(['bot', 'bot', 'human', 'human']);
      const seatA = startA.you.seat!;
      const seatB = startB.you.seat!;
      expect(seatA).not.toBe(seatB);
      const point = startA.point!.ref;

      // A spectator joins the running table.
      await c.goto(`/play/${code}`);
      await expect(c.locator('[data-watching]')).toBeVisible();
      expect((await latest(c)).you.seat).toBeUndefined();

      // The creator confirms a pick, and a reload restores it confirmed.
      const keysA = startA.view.you!.choices.map((ch) => ch.key);
      expect(keysA.length).toBeGreaterThan(1);
      const [firstA, secondA] = keysA as [string, string];
      await choose(a, firstA);
      await expectPending(a, firstA, false);
      await choose(a, 'confirm');
      await expectPending(a, firstA, true);
      await a.reload();
      await expectPending(a, firstA, true);
      await expect(a.locator(`[data-seat="${seatA}"]`)).toHaveAttribute('data-confirmed', 'true');

      // It withdraws, changes and confirms again.
      await choose(a, 'withdraw');
      await expectPending(a, firstA, false);
      await choose(a, secondA);
      await expectPending(a, secondA, false);
      await choose(a, 'confirm');
      await expectPending(a, secondA, true);

      // The second human drafts two picks no page of the others has any
      // reason to name, the second replacing the first.
      const seen = (await holdings(a, framesA, consoleA)) + (await holdings(c, framesC, consoleC));
      const privateKeys = startB.view
        .you!.choices.map((ch) => ch.key)
        .filter((k) => !seen.includes(k));
      expect(privateKeys.length, 'two of seat B choices never shown to A or C').toBeGreaterThan(1);
      const [firstB, secondB] = privateKeys as [string, string];
      await choose(b, firstB);
      await expectPending(b, firstB, false);
      await choose(b, secondB);
      await expectPending(b, secondB, false);

      // A network blip: the reconnect's frame still holds the draft.
      await ctxB.setOffline(true);
      await expect.poll(() => b.evaluate(() => window.__ajayebTable!.conn())).not.toBe('open');
      await ctxB.setOffline(false);
      await expectPending(b, secondB, false);
      expect((await latest(b)).point!.ref).toEqual(point);

      // The leak boundary, while both drafts are private.
      for (const [name, page, frames, lines] of [
        ['creator', a, framesA, consoleA],
        ['spectator', c, framesC, consoleC]
      ] as const) {
        const held = await holdings(page, frames, lines);
        for (const key of [firstB, secondB]) {
          expect(held.includes(key), `${name} holds seat ${seatB}'s pending ${key}`).toBe(false);
        }
      }
      expect(pendingOwnership('creator', trafficA)).toEqual([]);
      expect(pendingOwnership('spectator', trafficC)).toEqual([]);
      await expect(a.locator(`[data-seat="${seatB}"]`)).toHaveAttribute('data-confirmed', 'false');

      // The second human confirms last: the point resolves for everyone.
      await choose(b, 'confirm');
      for (const page of [a, b, c]) {
        await expect.poll(async () => (await latest(page)).point?.ref.id).not.toBe(point.id);
        expect((await latest(page)).batch).toBe(1);
        expect((await page.evaluate(() => window.__ajayebTable!.history())).length).toBe(2);
      }

      // The creator gives the second human's seat to a bot: that tab now
      // watches, and the new frame's history replaced its old view.
      await choose(a, `seat-bot:${seatB}`);
      await expect(b.locator(`[data-notice="watching"]`)).toBeVisible();
      await expect(b.locator('[data-watching]')).toBeVisible();
      await expect(b.locator('[data-pending]')).toHaveCount(0);
      const watched = await latest(b);
      expect(watched.you.seat).toBeUndefined();
      expect(watched.view.you ?? null).toBeNull();
      expect(watched.seats[seatB]!.kind).toBe('bot');
      expect((await latest(a)).you.seat).toBe(seatA);

      // Every frame of the whole game showed each page its own pick only.
      for (const [name, log] of [
        ['creator', trafficA],
        ['second human', trafficB],
        ['spectator', trafficC]
      ] as const) {
        expect(pendingOwnership(name, log)).toEqual([]);
      }
      // Canaries: the oracle reports a frame that shows the creator the
      // second human's draft, its own pick with the wrong confirmation, or
      // the spectator any pick.
      const states = (log: Traffic[]) =>
        log.flatMap((t, i) => (t.dir === 'recv' && t.msg.t === 'table-state' ? [i] : []));
      const ownA = states(trafficA).find((i) => trafficA[i]!.msg.pending);
      expect(ownA, 'a creator frame with its own pending pick').toBeDefined();
      const mine = trafficA[ownA!]!.msg.pending as { key: string; confirm: boolean };
      for (const canary of [
        contaminate(trafficA, ownA!, { key: firstB, confirm: false }),
        contaminate(trafficA, ownA!, { key: mine.key, confirm: !mine.confirm }),
        contaminate(trafficC, states(trafficC).at(-1)!, { key: firstB, confirm: false })
      ]) {
        expect(pendingOwnership('canary', canary)).not.toEqual([]);
      }

      expect(problemsA).toEqual([]);
      expect(problemsC).toEqual([]);
    } finally {
      for (const context of contexts) await context.close();
    }
  });

  test('a return from Home in the same page goes on with its request seqs', async ({ page }) => {
    await page.addInitScript(() => localStorage.setItem('ajayeb:name', 'Ana'));
    const log = traffic(page);
    await createTableFromHome(page);
    await page.waitForURL(/\/play\/[A-Z0-9]+$/);
    const path = new URL(page.url()).pathname;
    await choose(page, 'lobby-start');
    const [first, second] = (await latest(page)).view.you!.choices.map((ch) => ch.key) as [
      string,
      string
    ];
    await choose(page, first);
    await expectPending(page, first, false);

    // Home and back by the app's own navigation: the page, and with it the
    // client id, stays.
    await page.evaluate(() => {
      (window as unknown as { samePage?: boolean }).samePage = true;
    });
    await follow(page, '/');
    await page.waitForURL(/\/$/);
    await follow(page, path);
    await page.waitForURL(new RegExp(`${path}$`));
    // The server restores the pending pick, and the new one is applied.
    await expectPending(page, first, false);
    await choose(page, second);
    await expectPending(page, second, false);
    expect(await page.evaluate(() => (window as unknown as { samePage?: boolean }).samePage)).toBe(
      true
    );

    const sent = log.filter((t) => t.dir === 'sent').map((t) => t.msg);
    expect(new Set(sent.filter((m) => m.t === 'hello').map((m) => m.clientId)).size).toBe(1);
    // Start, the pick before leaving and the one after it.
    const seqs = sent
      .filter((m) => /^(table|lobby)-/.test(String(m.t)) && typeof m.seq === 'number')
      .map((m) => m.seq as number);
    expect(seqs.length).toBe(3);
    expect(seqs, 'each request a new, higher seq').toEqual(
      [...new Set(seqs)].sort((x, y) => x - y)
    );
    expect(pendingOwnership('the page', log)).toEqual([]);
  });
});
