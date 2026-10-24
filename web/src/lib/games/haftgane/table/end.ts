// What the end-of-Age military dialog and the score sheet read from a
// frame, without rules: the conflicts each end of Age reported, a seat's
// side of one, how many tokens a seat took in them, the score rows in the
// server's rank order and the verdict's winners. Every shield count, token
// and score comes from the server; nothing here adds up a total, compares
// shields or decides a winner.

import type { Result, Score, Seat } from '$lib/wire/haftgane.gen';
import type { TableEvent, TableHistoryEntry } from '$lib/wire/platform.gen';

export type ConflictOutcome = 'tie' | 'seat-wins' | 'other-wins';

/** One adjacent pair's conflict, as its `military` event reports it:
 *  `other` is `seat`'s left neighbour. */
export interface Conflict {
  seat: number;
  other: number;
  outcome: ConflictOutcome;
}

/** The conflicts of one end of Age, in seat order. */
export interface AgeConflicts {
  age: number;
  /** The history batch that reported them. */
  batch: number;
  conflicts: Conflict[];
}

function outcomeOf(e: TableEvent): ConflictOutcome | null {
  return e.outcome === 'tie' || e.outcome === 'seat-wins' || e.outcome === 'other-wins'
    ? e.outcome
    : null;
}

/** The conflicts one batch's events report, in seat order. */
export function conflictsIn(events: readonly TableEvent[]): Conflict[] {
  const out: Conflict[] = [];
  for (const e of events) {
    const outcome = e.kind === 'military' ? outcomeOf(e) : null;
    if (outcome) out.push({ seat: e.seat, other: e.other, outcome });
  }
  return out.sort((a, b) => a.seat - b.seat);
}

/**
 * Every end of Age's conflicts that the history reports up to and including
 * `batch` (the batch of the frame on screen), oldest first, one entry per
 * Age. A history that runs ahead of the screen is cut at the screen's batch.
 */
export function conflictsByAge(
  history: readonly TableHistoryEntry[],
  batch: number
): AgeConflicts[] {
  const byAge = new Map<number, AgeConflicts>();
  for (const entry of history) {
    if (entry.batch > batch) continue;
    const conflicts = conflictsIn(entry.events);
    const age = entry.events.find((e) => e.kind === 'military')?.age ?? 0;
    if (conflicts.length > 0 && age > 0) byAge.set(age, { age, batch: entry.batch, conflicts });
  }
  return [...byAge.values()].sort((a, b) => a.age - b.age);
}

export type SideResult = 'victory' | 'defeat' | 'tie';

/** How a conflict went for one of its two seats. */
export function resultFor(c: Conflict, seat: number): SideResult | null {
  if (seat !== c.seat && seat !== c.other) return null;
  if (c.outcome === 'tie') return 'tie';
  const seatWon = c.outcome === 'seat-wins';
  return seatWon === (seat === c.seat) ? 'victory' : 'defeat';
}

/** The conflict between two seats, in either order. */
export function conflictBetween(
  conflicts: readonly Conflict[],
  a: number,
  b: number
): Conflict | undefined {
  return conflicts.find((c) => (c.seat === a && c.other === b) || (c.seat === b && c.other === a));
}

/** The seat that won a conflict, or null for a tie. */
export function winnerOf(c: Conflict): number | null {
  return c.outcome === 'seat-wins' ? c.seat : c.outcome === 'other-wins' ? c.other : null;
}

/** The victories and defeats a seat took in these conflicts: a count of
 *  the server's outcomes, not their points. */
export function tokensTaken(
  conflicts: readonly Conflict[],
  seat: number
): { victories: number; defeats: number } {
  let victories = 0;
  let defeats = 0;
  for (const c of conflicts) {
    const r = resultFor(c, seat);
    if (r === 'victory') victories++;
    if (r === 'defeat') defeats++;
  }
  return { victories, defeats };
}

// The score sheet ---------------------------------------------------------------

/** The score sheet's columns, in score-pad order, each with its glyph. */
export const SCORE_COLUMNS = [
  { key: 'wonder', glyph: 'wonder' },
  { key: 'treasure', glyph: 'coin' },
  { key: 'military', glyph: 'conflict-victory' },
  { key: 'blue', glyph: 'category-blue-sm' },
  { key: 'yellow', glyph: 'category-yellow-sm' },
  { key: 'green', glyph: 'category-green-sm' },
  { key: 'purple', glyph: 'category-purple-sm' },
  { key: 'total', glyph: 'vp-laurel' }
] as const satisfies readonly { key: keyof Score; glyph: string }[];

export type ScoreColumn = (typeof SCORE_COLUMNS)[number]['key'];

export interface ScoreRow {
  seat: number;
  /** The server's rank: 1 for every winner. */
  rank: number;
  score: Score;
  winner: boolean;
}

/** One row per seat, in the server's rank order, seats in order within a
 *  shared rank. */
export function scoreRows(result: Result): ScoreRow[] {
  return result.scores
    .map((score, seat) => ({
      seat,
      rank: result.rank[seat] ?? 0,
      score,
      winner: result.winners.includes(seat)
    }))
    .sort((a, b) => a.rank - b.rank || a.seat - b.seat);
}

/**
 * The verdict's subject: the viewer alone, another seat alone, or the seats
 * that share the victory (the viewer among them or not), all from the
 * server's winners.
 */
export type Verdict =
  | { kind: 'you' }
  | { kind: 'one'; seat: number }
  | { kind: 'shared'; seats: number[]; you: boolean };

export function verdictOf(result: Result, you: number | null): Verdict | null {
  const winners = [...result.winners].sort((a, b) => a - b);
  if (winners.length === 0) return null;
  if (winners.length === 1)
    return winners[0] === you ? { kind: 'you' } : { kind: 'one', seat: winners[0]! };
  return {
    kind: 'shared',
    seats: winners.filter((s) => s !== you),
    you: you !== null && winners.includes(you)
  };
}

/**
 * A tie at the top, as the score sheet's sentence states it. The view does
 * not carry it yet (requested as `result.tieBreak`): `coins` when coins
 * decided between equal totals, `shared` when coins were equal too, with
 * the seats that tied on total.
 */
export interface TieBreak {
  kind: 'coins' | 'shared';
  seats: number[];
}

/** A seat's final coins, from its public city. */
export function coinsOf(seats: readonly Seat[], seat: number): number {
  return seats.find((s) => s.seat === seat)?.coins ?? 0;
}
