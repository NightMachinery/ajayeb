// Shared helpers for the Haftgane table specs: the page's table hook, the
// waits on it, choosing through the table screen (a choice of a hand card
// appears once that card is selected; a seat control sits in the game
// menu), and the pending-ownership oracle, which checks every table frame
// a page received against the picks the page itself sent.

import { expect, type Page } from '@playwright/test';
import type { LobbyState, PointRef, TableState } from '../../web/src/lib/wire/platform.gen';
import { createTableFromHome, lobbyReady } from './lobby';

export interface TableHook {
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

/** Waits until the page shows the latest frame and takes input. */
export async function ready(page: Page): Promise<void> {
  await expect.poll(() => page.evaluate(() => window.__ajayebTable?.ready() ?? false)).toBe(true);
}

export async function latest(page: Page): Promise<TableState> {
  await ready(page);
  const frame = await page.evaluate(() => window.__ajayebTable!.latest());
  expect(frame).not.toBeNull();
  return frame!;
}

/**
 * Clicks the control with this data-act. A choice of a hand card shows in
 * the pick list once its card is selected, so the card is tapped first;
 * a seat control is in the game menu, so the menu is opened first.
 */
export async function choose(page: Page, act: string): Promise<void> {
  await ready(page);
  const target = page.locator(`[data-act="${act}"]`);
  if (!(await target.isVisible())) {
    const design = await page.evaluate(
      (key) =>
        window.__ajayebTable
          ?.latest()
          ?.view.you?.choices.find((c) => c.key === key && c.kind !== 'build-discarded')?.design ??
        '',
      act
    );
    const openSheet = page.locator('[data-card-sheet]');
    if (
      (await openSheet.isVisible()) &&
      (await openSheet.getAttribute('data-card-sheet')) !== design
    )
      await page.locator('[data-act="pick-close"]').click();
    const card = page.locator(`[data-act="hand:${design}"]`).first();
    if (
      design &&
      (await card.isVisible()) &&
      !(await page.locator('[data-card-sheet]').isVisible())
    )
      await card.click();
    else if (/^(seat-|undo$|resign$)/.test(act))
      await page.locator('[data-act="table-menu"]').click();
  }
  const draft = page.locator(`[data-card-sheet] [data-pick="${act}"]`);
  if (await draft.isVisible()) {
    await draft.click();
    await ready(page);
    await page.locator('[data-act="pick-close"]').click();
  } else await target.click();
  await ready(page);
}

export async function expectPending(page: Page, key: string, confirm: boolean): Promise<void> {
  const pending = page.locator('[data-pending]');
  await expect(pending).toHaveAttribute('data-pending', key);
  await expect(pending).toHaveAttribute('data-pending-confirm', String(confirm));
  expect((await latest(page)).pending).toEqual({ key, confirm });
}

/** One message on a page's sockets, either way, in order. */
export interface Traffic {
  dir: 'sent' | 'recv';
  msg: Record<string, unknown>;
}

/** Records every message on a page's game sockets, across reloads and reconnects. */
export function traffic(page: Page): Traffic[] {
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
export function pendingOwnership(name: string, log: readonly Traffic[]): string[] {
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
export function contaminate(
  log: readonly Traffic[],
  i: number,
  pending: TableState['pending']
): Traffic[] {
  return log.map((t, j) => (j === i ? { dir: t.dir, msg: { ...t.msg, pending } } : t));
}

/** The indexes of the table frames a page received. */
export function tableStates(log: readonly Traffic[]): number[] {
  return log.flatMap((t, i) => (t.dir === 'recv' && t.msg.t === 'table-state' ? [i] : []));
}

/**
 * Opens a table of this many seats for the page's user alone, the way a
 * player does: Home's New table form under this name, the lobby set to
 * the seat count, and Start, which gives every open seat a bot. Returns
 * the room code once the first table frame is on screen.
 */
export async function openBotTable(page: Page, seats: number, name = 'Host'): Promise<string> {
  const code = await createTableFromHome(page, name);
  const count = page.locator('[data-act="lobby-seats"]');
  if ((await count.inputValue()) !== String(seats)) {
    await count.selectOption(String(seats));
    await lobbyReady(page);
  }
  await expect(page.locator('[data-lobby] [data-seat]')).toHaveCount(seats);
  await choose(page, 'lobby-start');
  await expect(page.locator('[data-table-screen]')).toBeVisible();
  await ready(page);
  return code;
}
