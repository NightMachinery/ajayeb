// The turn recap's facts (docs/haftgane/phone-layout.md, section 4): what
// each seat did in one resolved batch and the coins that moved, read from
// the batch's events as the server filtered them for this viewer, and the
// seat and role changes the room log dates to that turn.
//
// Nothing is added up. A coin change is one event's own number: a seat's
// printed cost and the trade it paid each neighbour come from its play, a
// sale's or a card's income from its coins event, and a receipt is the
// payer's number read from the seat it went to (a seat's left neighbour is
// the next seat and its right the one before, as model.ts seats the ring).
// A face-down card stays unnamed because the server names a sold card, a
// card under a wonder and a discarded last card only to its owner.

import type { PlatformLogEntry, TableEvent, TableHistoryEntry } from '$lib/wire/platform.gen';

/** What a seat did in the batch. `design` is '' where the viewer may not know it. */
export type RecapAct =
  | { kind: 'build'; design: string }
  | { kind: 'build-discarded'; design: string }
  | { kind: 'stage'; stage: number; design: string }
  | { kind: 'sell'; design: string }
  | { kind: 'discard-nothing' }
  | { kind: 'side'; side: string };

/** One movement of coins for a seat, as one event states it. */
export type CoinChange =
  | { kind: 'bank'; amount: number }
  | { kind: 'paid'; seat: number; amount: number }
  | { kind: 'received'; seat: number; amount: number }
  | { kind: 'sale'; amount: number }
  | { kind: 'card'; design: string; amount: number }
  | { kind: 'stage'; stage: number; amount: number };

export interface RecapLine {
  seat: number;
  /** Null for a seat that only received coins (another seat's extra turn). */
  act: RecapAct | null;
  coins: CoinChange[];
  /** The seat keeps its last card for an extra turn: the card for its owner, else ''. */
  kept: string | null;
  /** The viewer's own last card, which went to the discard pile. */
  leftover: string | null;
}

export interface Recap {
  /** The batch it recaps: the history entry's number. */
  batch: number;
  /** The Age and turn, when the batch passed hands (turns one to five). */
  turn: { age: number; turn: number } | null;
  /** The Age whose conflicts the batch resolved. */
  ageEnd: number | null;
  /** The Age the batch dealt. */
  ageStart: number | null;
  /** Where hands went, when they passed. */
  pass: 'left' | 'right' | null;
  gameOver: boolean;
  /** One line per seat with something to say, in ring order from the anchor. */
  lines: RecapLine[];
  /** The seat switches, resignations and role changes made during the turn. */
  changes: PlatformLogEntry[];
}

const TURN_KINDS: ReadonlySet<string> = new Set([
  'play',
  'discard-build',
  'discard-build-nothing',
  'side-chosen'
]);
const CHANGE_KINDS: ReadonlySet<string> = new Set(['seat', 'role', 'resign']);

/** Whether a batch's events are a turn to recap: plays, a discard build or the sides. */
export function recapWorthy(events: readonly TableEvent[]): boolean {
  return events.some((e) => TURN_KINDS.has(e.kind));
}

/** The seats in ring order from the anchor: the anchor, its left neighbour, and on. */
export function ringOrder(count: number, anchor: number): number[] {
  return Array.from({ length: count }, (_, i) => (anchor + i) % count);
}

function actOf(e: TableEvent): RecapAct | null {
  switch (e.kind) {
    case 'play':
      if (e.play === 'build') return { kind: 'build', design: e.design };
      if (e.play === 'stage') return { kind: 'stage', stage: e.stage, design: e.design };
      if (e.play === 'sell') return { kind: 'sell', design: e.design };
      return null;
    case 'discard-build':
      return { kind: 'build-discarded', design: e.design };
    case 'discard-build-nothing':
      return { kind: 'discard-nothing' };
    case 'side-chosen':
      return { kind: 'side', side: e.side };
  }
  return null;
}

function incomeOf(e: TableEvent): CoinChange | null {
  if (e.kind !== 'coins' || e.delta === 0) return null;
  switch (e.source) {
    case 'sale':
      return { kind: 'sale', amount: e.delta };
    case 'design':
      return { kind: 'card', design: e.design, amount: e.delta };
    case 'stage':
      return { kind: 'stage', stage: e.stage, amount: e.delta };
  }
  return null;
}

/**
 * The recap of one batch for a table of `count` seats, seen from `anchor`
 * (the viewer's seat, or the seat a spectator follows). `since` is the batch
 * of the turn recapped before it: the room log entries made after that
 * turn resolved and before this one did are this turn's changes.
 */
export function buildRecap(
  entry: TableHistoryEntry,
  count: number,
  anchor: number,
  since: number,
  log: readonly PlatformLogEntry[]
): Recap {
  const events = entry.events;
  const left = (s: number) => (s + 1) % count;
  const right = (s: number) => (s + count - 1) % count;
  const lines: RecapLine[] = [];
  for (const seat of ringOrder(count, anchor)) {
    let act: RecapAct | null = null;
    const coins: CoinChange[] = [];
    let kept: string | null = null;
    let leftover: string | null = null;
    for (const e of events) {
      if (e.seat === seat && act === null && e.kind !== 'coins') act = actOf(e);
      if (e.kind === 'play' && e.seat === seat) {
        if (e.printed > 0) coins.push({ kind: 'bank', amount: e.printed });
        if (e.left > 0) coins.push({ kind: 'paid', seat: left(seat), amount: e.left });
        if (e.right > 0) coins.push({ kind: 'paid', seat: right(seat), amount: e.right });
      }
      if (e.kind === 'card-kept' && e.seat === seat) kept = e.design;
      if (e.kind === 'leftover-discarded' && e.seat === seat && e.design) leftover = e.design;
    }
    for (const e of events) {
      if (e.kind !== 'play' || e.seat === seat) continue;
      if (e.left > 0 && left(e.seat) === seat)
        coins.push({ kind: 'received', seat: e.seat, amount: e.left });
      if (e.right > 0 && right(e.seat) === seat)
        coins.push({ kind: 'received', seat: e.seat, amount: e.right });
    }
    for (const e of events) {
      if (e.seat !== seat) continue;
      const income = incomeOf(e);
      if (income) coins.push(income);
    }
    if (act || coins.length > 0 || kept !== null || leftover !== null) {
      lines.push({ seat, act, coins, kept, leftover });
    }
  }
  const passed = events.find((e) => e.kind === 'hands-passed');
  const military = events.find((e) => e.kind === 'military');
  const start = events.find((e) => e.kind === 'age-start');
  return {
    batch: entry.batch,
    turn: passed ? { age: passed.age, turn: passed.turn } : null,
    ageEnd: military ? military.age : null,
    ageStart: start ? start.age : null,
    pass: passed?.pass === 'left' || passed?.pass === 'right' ? passed.pass : null,
    gameOver: events.some((e) => e.kind === 'game-over'),
    lines,
    changes: log.filter((l) => CHANGE_KINDS.has(l.kind) && l.at >= since && l.at < entry.batch)
  };
}

/** The newest turn in the history, recapped; null before the first. */
export function latestRecap(
  history: readonly TableHistoryEntry[],
  count: number,
  anchor: number,
  log: readonly PlatformLogEntry[]
): Recap | null {
  let at = -1;
  for (let i = history.length - 1; i >= 0; i--) {
    if (recapWorthy(history[i]!.events)) {
      at = i;
      break;
    }
  }
  if (at < 0 || count < 3) return null;
  let since = 0;
  for (let i = at - 1; i >= 0; i--) {
    if (recapWorthy(history[i]!.events)) {
      since = history[i]!.batch;
      break;
    }
  }
  return buildRecap(history[at]!, count, anchor, since, log);
}

/** The line of one seat, if the recap has one. */
export function lineOf(recap: Recap | null, seat: number): RecapLine | undefined {
  return recap?.lines.find((l) => l.seat === seat);
}

/** The sale's own coins for a seat that sold, for its "Sold +3" tag. */
export function saleOf(line: RecapLine): number | null {
  const sale = line.coins.find((c) => c.kind === 'sale');
  return sale ? sale.amount : null;
}

/** How the new hand arrives with the batch on screen: from the neighbour who passed it, dealt for a new Age, or not at all. */
export function dealOf(events: readonly TableEvent[]): 'left' | 'right' | 'deal' | null {
  const passed = events.find((e) => e.kind === 'hands-passed');
  if (passed?.pass === 'left' || passed?.pass === 'right') return passed.pass;
  return events.some((e) => e.kind === 'age-start') ? 'deal' : null;
}
