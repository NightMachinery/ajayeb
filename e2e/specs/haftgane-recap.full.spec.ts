// Haftgane's reveal and turn recap over the real server, from Home's lobby
// (M18.reveal-and-recap). The creator opens a three-seat table alone with
// bots, with Settings' "Show opponent move" on, and a friend joins the
// running table as a spectator. The creator plays turn after turn, building,
// selling and staging in turn, through the end of Age I and one turn of
// Age II, and after each batch the spec checks, against that batch's events
// as each page received them:
//
// - the creator's recap opens by itself over the board: one line per seat
//   that acted, in ring order from the creator, each with what it did, the
//   card when the page may know it, and every coin change with its sign;
//   where hands went; the live region speaks it;
// - once closed, each seat's reveal tag hangs on the board and the recap
//   reopens from the top bar; the new hand carries the side it came from;
// - the spectator's panel shows the same recap, with no card named for a
//   sale, a wonder stage or a discarded last card, and the board's tags;
// - at the end of the Age the recap comes first and the military dialog
//   only after it closes;
// - a role change made between two turns is in the next turn's recap;
// - every table frame each page received names no face-down card of
//   another seat, in its events, its history or a seat's last play, gives
//   no seat a private field and no page another seat's private view;
// - nothing animates with Animation off, and no console error or failed
//   request happens.
//
// Nothing depends on a particular deal: every seat, card and coin checked
// is read from the frames.
import { expect, type BrowserContext, type Locator, type Page } from '@playwright/test';
import type { TableEvent, TableState } from '../../web/src/lib/wire/platform.gen';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { newDeviceContext } from '../fixtures/online';
import {
  choose,
  dismissMilitary,
  dismissRecap,
  latest,
  menuAction,
  openBotTable,
  pendingOwnership,
  ready,
  tableStates,
  traffic,
  type Traffic
} from '../fixtures/table';

const SEATS = 3;

/** The fields a public seat may carry. */
const PUBLIC_SEAT = new Set([
  'seat',
  'board',
  'side',
  'stages',
  'built',
  'coins',
  'victories',
  'defeats',
  'shields',
  'handSize',
  'hasKept',
  'relative',
  'relation',
  'last',
  'score',
  'science'
]);

/** The plays whose card stays face down for everyone but its owner. */
const FACE_DOWN = new Set(['stage', 'sell']);

/** A face-down card of another seat that this event names. */
function namesHidden(e: TableEvent, viewer: number | undefined): boolean {
  if (e.seat === viewer || e.design === '') return false;
  if (e.kind === 'play') return FACE_DOWN.has(e.play);
  return e.kind === 'leftover-discarded' || e.kind === 'card-kept';
}

/**
 * The per-frame leak oracle: no event, history entry or last play names
 * another seat's face-down card, no seat carries a private field, and the
 * private view is the viewer's own seat's.
 */
function frameLeaks(name: string, log: readonly Traffic[]): string[] {
  const out: string[] = [];
  for (const i of tableStates(log)) {
    const f = log[i]!.msg as unknown as TableState;
    const viewer = f.you.seat;
    const events = [...f.events, ...(f.history ?? []).flatMap((h) => h.events)];
    for (const e of events) {
      if (namesHidden(e, viewer))
        out.push(`${name} ${i}: ${e.kind} of seat ${e.seat} names ${e.design}`);
    }
    for (const s of f.view.seats) {
      const extra = Object.keys(s).filter((k) => !PUBLIC_SEAT.has(k));
      if (extra.length) out.push(`${name} ${i}: seat ${s.seat} carries ${extra.join(', ')}`);
      if (s.seat !== viewer && s.last && FACE_DOWN.has(s.last.kind) && s.last.design !== '')
        out.push(`${name} ${i}: seat ${s.seat}'s last ${s.last.kind} names ${s.last.design}`);
    }
    if (f.view.you && f.view.you.seat !== viewer)
      out.push(`${name} ${i}: private view of seat ${f.view.you.seat}`);
  }
  return out;
}

/** What a seat's recap line must say: its act, its card when known, its coin changes. */
interface Line {
  act: string;
  card: string | null;
  coins: string[];
}

const ACTS: Record<string, (e: TableEvent) => string | null> = {
  play: (e) => (['build', 'stage', 'sell'].includes(e.play) ? e.play : null),
  'discard-build': () => 'build-discarded',
  'discard-build-nothing': () => 'discard-nothing',
  'side-chosen': () => 'side'
};
const INCOME: Record<string, string> = { sale: 'sale', design: 'card', stage: 'stage' };

/**
 * Each seat's recap line, read straight from a batch's events: a seat's
 * left neighbour is the next seat and its right the one before; a coin
 * change is one event's own number, a payment negative.
 */
function expectedLines(events: readonly TableEvent[]): Map<number, Line> {
  const lines = new Map<number, Line>();
  const at = (seat: number): Line => {
    let l = lines.get(seat);
    if (!l) lines.set(seat, (l = { act: 'none', card: null, coins: [] }));
    return l;
  };
  const left = (s: number) => (s + 1) % SEATS;
  const right = (s: number) => (s + SEATS - 1) % SEATS;
  for (const e of events) {
    const act = ACTS[e.kind]?.(e);
    if (act && at(e.seat).act === 'none') {
      at(e.seat).act = act;
      at(e.seat).card = act === 'discard-nothing' || act === 'side' ? null : e.design || null;
    }
    // A kept or discarded last card gives its seat a line of its own.
    if (e.kind === 'card-kept' || (e.kind === 'leftover-discarded' && e.design)) at(e.seat);
    if (e.kind === 'play') {
      if (e.printed > 0) at(e.seat).coins.push(`bank:${-e.printed}`);
      if (e.left > 0) {
        at(e.seat).coins.push(`paid:${-e.left}`);
        at(left(e.seat)).coins.push(`received:${e.left}`);
      }
      if (e.right > 0) {
        at(e.seat).coins.push(`paid:${-e.right}`);
        at(right(e.seat)).coins.push(`received:${e.right}`);
      }
    }
    if (e.kind === 'coins' && e.delta !== 0 && INCOME[e.source])
      at(e.seat).coins.push(`${INCOME[e.source]}:${e.delta}`);
  }
  for (const l of lines.values()) l.coins.sort();
  return lines;
}

/** The recap lines a page shows inside `root`, read from the DOM. */
async function shownLines(root: Locator): Promise<Map<number, Line>> {
  const rows = await root.locator('[data-recap-line]').evaluateAll((els) =>
    els.map((el) => ({
      seat: Number(el.getAttribute('data-recap-line')),
      act: el.getAttribute('data-recap-act') ?? '',
      card: el.getAttribute('data-recap-card'),
      coins: [...el.querySelectorAll('[data-coin]')]
        .map((c) => `${c.getAttribute('data-coin')}:${c.getAttribute('data-coin-amount')}`)
        .sort()
    }))
  );
  return new Map(rows.map((r) => [r.seat, { act: r.act, card: r.card, coins: r.coins }]));
}

/** The resolved batch's events, as the page holds them in its history. */
async function batchEvents(page: Page, batch: number): Promise<TableEvent[]> {
  const entry = await page.evaluate(
    (b) => window.__ajayebTable!.history().find((h) => (h as { batch: number }).batch === b),
    batch
  );
  expect(entry, `batch ${batch} is in the history`).toBeDefined();
  return (entry as { events: TableEvent[] }).events;
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

async function named(context: BrowserContext, name: string, recap: boolean): Promise<Page> {
  await context.addInitScript(
    ({ n, on }) => {
      localStorage.setItem('ajayeb:name', n);
      if (!on) return;
      let current: Record<string, unknown> = {};
      try {
        current = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}') as typeof current;
      } catch {
        current = {};
      }
      localStorage.setItem(
        'ajayeb:settings',
        JSON.stringify({ ...current, showOpponentMove: true })
      );
    },
    { n: name, on: recap }
  );
  return context.newPage();
}

/** The choice kinds the creator tries in turn, so sales and stages come up too. */
const ROTATION = ['build', 'sell', 'stage'] as const;

test.describe('Haftgane reveal and recap', { tag: ['@full', '@mobile', '@haftgane-recap'] }, () => {
  test('the reveal, the recap and the hand pass for a player and a spectator', async ({
    browser
  }, info) => {
    test.setTimeout(420_000);
    const contexts: BrowserContext[] = [];
    try {
      for (let i = 0; i < 2; i++) contexts.push(await newDeviceContext(browser, info));
      const a = await named(contexts[0]!, 'Ana', true);
      const b = await named(contexts[1]!, 'Bo', false);
      const hygiene = [a, b].map(watchHygiene);
      const logs = [a, b].map(traffic);

      const code = await openBotTable(a, SEATS, 'Ana');
      const me = (await latest(a)).you.seat!;
      await b.goto(`/play/${code}`);
      await expect(b.locator('[data-spectator-panel]')).toBeVisible();
      const userB = (await latest(b)).you.userId;
      // Before the first turn there is nothing to recap.
      await expect(a.locator('[data-act="recap"]')).toHaveCount(0);
      await expect(b.locator('[data-spectator-panel] [data-turn-recap]')).toHaveCount(0);

      let turns = 0;
      let ageEnds = 0;
      let promoted = false;
      let changeChecked = false;
      for (let round = 0; ageEnds === 0 || turns < 8; round++) {
        expect(round, 'the end of Age I came in time').toBeLessThan(16);
        const f = await latest(a);
        expect(f.status).toBe('playing');
        const required = f.point?.required.includes(me) && !f.point.confirmed.includes(me);
        if (required) {
          const cs = f.view.you!.choices;
          const kind = ROTATION[turns % ROTATION.length];
          const pick =
            cs.find((c) => c.kind === kind) ?? cs.find((c) => c.kind === 'build') ?? cs[0]!;
          await choose(a, pick.key);
          // Choosing starts a new decision: the tags leave the board.
          await expect(a.locator('[data-board-reveal]')).toHaveCount(0);
          await choose(a, 'confirm');
        }
        await expect.poll(async () => (await latest(a)).batch).toBeGreaterThan(f.batch);
        await ready(a);
        const batch = (await latest(a)).batch;
        await expect.poll(async () => (await latest(b)).batch).toBe(batch);
        await ready(b);

        const mine = await batchEvents(a, batch);
        const theirs = await batchEvents(b, batch);
        const ageEnd = mine.some((e) => e.kind === 'military');
        if (!mine.some((e) => e.kind in ACTS)) {
          // Nothing to recap: no recap opens, and conflicts show at once.
          await expect(a.locator('[data-recap-layer]')).toHaveCount(0);
          if (ageEnd) ageEnds++;
          await dismissMilitary(a);
          continue;
        }
        turns++;

        // The creator's recap opens by itself, before any military dialog.
        const layer = a.locator('[data-recap-layer]');
        const recap = layer.locator(`[data-turn-recap="${batch}"]`);
        await expect(recap).toBeVisible();
        await expect(a.locator('[data-age-military]')).toHaveCount(0);
        const want = expectedLines(mine);
        expect(await shownLines(recap)).toEqual(want);
        await expect(recap.locator('[data-recap-line]').first()).toHaveAttribute(
          'data-recap-line',
          String(me)
        );
        const passed = mine.find((e) => e.kind === 'hands-passed');
        if (passed) {
          await expect(recap.locator('[data-recap-closing]')).toHaveAttribute(
            'data-recap-closing',
            passed.pass
          );
        }
        // The creator's own discarded last card is named to the creator.
        const leftover = mine.find((e) => e.kind === 'leftover-discarded' && e.seat === me);
        if (leftover?.design) {
          await expect(
            recap.locator(`[data-recap-line="${me}"] [data-recap-last]`).first()
          ).toBeVisible();
        }
        await expect(a.locator('[data-recap-live]')).not.toHaveText('');
        const change = promoted && !changeChecked;
        if (change) await expect(recap.locator('[data-recap-change="role"]')).toHaveText(/Bo/);
        // It was seen as it opened: no unseen dot.
        await expect(a.locator('[data-act="recap"]')).not.toHaveAttribute('data-unseen');
        expect(await running(a), 'animations with Animation off').toBe(0);

        // The spectator's panel: the same lines, with no face-down card.
        const panel = b.locator(`[data-spectator-panel] [data-turn-recap="${batch}"]`);
        await expect(panel).toBeVisible();
        const watched = await shownLines(panel);
        expect(watched).toEqual(expectedLines(theirs));
        expect([...watched.keys()].sort()).toEqual([...want.keys()].sort());
        for (const [seat, line] of watched) {
          expect(line.act).toBe(want.get(seat)!.act);
          expect(line.coins).toEqual(want.get(seat)!.coins);
          if (FACE_DOWN.has(line.act)) expect(line.card, `seat ${seat}'s ${line.act}`).toBeNull();
        }
        if (change) {
          await expect(panel.locator('[data-recap-change="role"]')).toHaveText(/Bo/);
          changeChecked = true;
        }
        await expect(b.locator('[data-recap-live]')).not.toHaveText('');
        await expect(b.locator('[data-board-reveal]')).toHaveCount(
          [...want.values()].filter((l) => l.act !== 'none').length
        );
        expect(await running(b), 'animations with Animation off').toBe(0);

        // Closed, the military dialog follows at the end of an Age; then
        // the tags show on the board and the recap reopens on request.
        await dismissRecap(a);
        if (ageEnd) {
          ageEnds++;
          await expect(a.locator('[data-age-military]')).toBeVisible();
          await dismissMilitary(a);
        }
        for (const [seat, line] of want) {
          if (line.act === 'none') continue;
          await expect(
            a.locator(`[data-board-reveal="${seat}"] [data-reveal-tag]`)
          ).toHaveAttribute('data-reveal-act', line.act);
        }
        const dealt = passed
          ? passed.pass
          : mine.some((e) => e.kind === 'age-start')
            ? 'deal'
            : null;
        if (dealt && (await latest(a)).view.you?.hand.length)
          await expect(a.locator('[data-deal]')).toHaveAttribute('data-deal', dealt);
        await a.locator('[data-act="recap"]').click();
        await expect(recap).toBeVisible();
        await dismissRecap(a);

        // Between two turns, the creator makes the spectator an admin.
        if (turns === 2 && !promoted) {
          await menuAction(a, `role-promote:${userB}`);
          await a.locator('[data-act="menu-close"]').click();
          promoted = true;
        }
      }
      expect(changeChecked, 'the role change was recapped').toBe(true);

      const names = ['creator', 'spectator'];
      for (const [i, log] of logs.entries()) {
        expect(pendingOwnership(names[i]!, log)).toEqual([]);
        expect(frameLeaks(names[i]!, log)).toEqual([]);
      }
      for (const problems of hygiene) expect(problems).toEqual([]);
    } finally {
      for (const ctx of contexts) await ctx.close();
    }
  });
});
