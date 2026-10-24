// The end-of-Age and score sheet model over the engine's own game-over view
// fixture (server/internal/haftgane/rules/testdata/views): the conflicts of
// a history, each seat's side of them, the rows in rank order and the
// verdict, all read from what the server sent.
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import type { Result, Table } from '$lib/wire/haftgane.gen';
import type { TableEvent, TableHistoryEntry } from '$lib/wire/platform.gen';
import {
  conflictBetween,
  conflictsByAge,
  conflictsIn,
  resultFor,
  scoreRows,
  shieldsFor,
  tieBreakOf,
  tokensTaken,
  verdictOf,
  winnerOf
} from './end';

function fixture(name: string): Table {
  return JSON.parse(
    readFileSync(
      new URL(
        `../../../../../../server/internal/haftgane/rules/testdata/views/${name}.json`,
        import.meta.url
      ),
      'utf8'
    )
  ) as Table;
}

const over = fixture('over-reveal-spectator');
const asTable = (events: Table['events']): TableEvent[] => events.map((e) => ({ ...e, pass: '' }));

function military(age: number, seat: number, other: number, outcome: string): TableEvent {
  return { ...asTable(over.events)[0]!, kind: 'military', seat, other, age, outcome };
}

describe('conflicts', () => {
  it('reads every adjacent pair of the last Age from the server events', () => {
    const cs = conflictsIn(asTable(over.events));
    expect(cs).toHaveLength(7);
    expect(cs.map((c) => [c.seat, c.other])).toEqual([
      [0, 1],
      [1, 2],
      [2, 3],
      [3, 4],
      [4, 5],
      [5, 6],
      [6, 0]
    ]);
    expect(cs[5]!.outcome).toBe('tie');
  });

  it('gives each seat its side of a conflict and names the winner', () => {
    const [first, second] = conflictsIn(asTable(over.events));
    expect(resultFor(first!, 0)).toBe('victory');
    expect(resultFor(first!, 1)).toBe('defeat');
    expect(resultFor(first!, 2)).toBeNull();
    expect(winnerOf(first!)).toBe(0);
    expect(resultFor(second!, 2)).toBe('victory');
    expect(winnerOf(second!)).toBe(2);
    const tie = conflictBetween(conflictsIn(asTable(over.events)), 6, 5)!;
    expect(resultFor(tie, 5)).toBe('tie');
    expect(winnerOf(tie)).toBeNull();
  });

  it('carries the shields each side compared and the victory token worth', () => {
    const cs = conflictsIn(asTable(over.events));
    expect(cs[0]).toMatchObject({ seatShields: 3, otherShields: 0, worth: 5 });
    expect(shieldsFor(cs[0]!, 1)).toEqual({ own: 0, them: 3 });
    expect(shieldsFor(cs[0]!, 0)).toEqual({ own: 3, them: 0 });
    expect(cs[5]).toMatchObject({ outcome: 'tie', seatShields: 2, otherShields: 2, worth: 0 });
  });

  it('counts the tokens a seat took, not their points', () => {
    const cs = conflictsIn(asTable(over.events));
    expect(tokensTaken(cs, 0)).toEqual({ victories: 2, defeats: 0 });
    expect(tokensTaken(cs, 1)).toEqual({ victories: 0, defeats: 2 });
    expect(tokensTaken(cs, 5)).toEqual({ victories: 1, defeats: 0 });
    expect(tokensTaken(cs, 6)).toEqual({ victories: 0, defeats: 1 });
  });

  it('keeps one entry per Age, cut at the batch on screen', () => {
    const history: TableHistoryEntry[] = [
      { batch: 0, events: [] },
      { batch: 6, events: [military(1, 0, 1, 'tie'), military(1, 1, 2, 'seat-wins')] },
      { batch: 12, events: [military(2, 1, 2, 'other-wins'), military(2, 0, 1, 'tie')] }
    ];
    const all = conflictsByAge(history, 12);
    expect(all.map((a) => [a.age, a.batch])).toEqual([
      [1, 6],
      [2, 12]
    ]);
    expect(all[1]!.conflicts.map((c) => c.seat)).toEqual([0, 1]);
    expect(conflictsByAge(history, 11).map((a) => a.age)).toEqual([1]);
    expect(conflictsByAge(history, 5)).toEqual([]);
  });
});

describe('score sheet', () => {
  const result = over.result!;

  it('orders rows by the server rank', () => {
    const rows = scoreRows(result);
    expect(rows.map((r) => r.seat)).toEqual([2, 4, 5, 0, 6, 3, 1]);
    expect(rows[0]!.winner).toBe(true);
    expect(rows[0]!.score.total).toBe(33);
    expect(rows.filter((r) => r.winner)).toHaveLength(1);
  });

  it('keeps seat order inside a shared rank', () => {
    const shared: Result = { ...result, rank: [1, 3, 1], winners: [0, 2] };
    expect(scoreRows({ ...shared, scores: result.scores.slice(0, 3) }).map((r) => r.seat)).toEqual([
      0, 2, 1
    ]);
  });

  it('states the verdict from the winners alone', () => {
    expect(verdictOf(result, 2)).toEqual({ kind: 'you' });
    expect(verdictOf(result, 0)).toEqual({ kind: 'one', seat: 2 });
    expect(verdictOf(result, null)).toEqual({ kind: 'one', seat: 2 });
    const shared: Result = { ...result, winners: [4, 2] };
    expect(verdictOf(shared, 2)).toEqual({ kind: 'shared', seats: [4], you: true });
    expect(verdictOf(shared, 0)).toEqual({ kind: 'shared', seats: [2, 4], you: false });
    expect(verdictOf({ ...result, winners: [] }, 0)).toBeNull();
  });

  it('states the tie at the top only as the server sends it', () => {
    expect(result.tieBreak).toBe('');
    expect(tieBreakOf(result)).toBeUndefined();
    expect(tieBreakOf({ ...result, tieBreak: 'coins', tied: [2, 4] })).toEqual({
      kind: 'coins',
      seats: [2, 4]
    });
    expect(tieBreakOf({ ...result, tieBreak: 'shared', tied: [0, 2, 4] })).toEqual({
      kind: 'shared',
      seats: [0, 2, 4]
    });
  });
});
