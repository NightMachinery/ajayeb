import { describe, expect, it } from 'vitest';
import type { GameEvent, HistoryEntry, RoomLogEntry } from '$lib/wire';
import { formatRoomLogEntry, groupGameLog } from './log';

const names = ['Sara', 'Omid'] as const;

function ev(e: Partial<GameEvent> & Pick<GameEvent, 'kind'>): GameEvent {
  return { private: false, ...e };
}

function change(
  at: number,
  seat: 0 | 1 | undefined,
  on: RoomLogEntry['on'],
  off: RoomLogEntry['off']
): RoomLogEntry {
  return seat === undefined ? { kind: 'hints', at, on, off } : { kind: 'hints', at, seat, on, off };
}

const history: HistoryEntry[] = [
  { ply: 0, seat: 0, events: [ev({ kind: 'draft', seat: 0, wonder: 'pyramids' })] },
  {
    ply: 1,
    seat: 1,
    events: [ev({ kind: 'draft', seat: 1, wonder: 'sphinx' }), ev({ kind: 'ageStart', age: 1 })]
  },
  { ply: 2, seat: 0, events: [ev({ kind: 'build', seat: 0, card: 'baths' })] }
];

describe('formatRoomLogEntry', () => {
  it('names who changed what', () => {
    expect(formatRoomLogEntry(change(0, 1, ['advisor'], ['prices', 'threats']), names)).toBe(
      'Omid changed the hints: turned on Advisor; turned off Prices and Threats.'
    );
    expect(formatRoomLogEntry(change(0, undefined, [], ['recap']), names)).toBe(
      'The hints changed: turned off Recap.'
    );
  });
});

describe('groupGameLog', () => {
  it('keeps a mid-game scenario and its room changes out of the wonder draft', () => {
    const scenario: HistoryEntry[] = [
      { ply: 0, seat: 0, events: [ev({ kind: 'build', card: 'rostrum', seat: 0 })] }
    ];
    const groups = groupGameLog(scenario, [change(0, 0, [], ['prices'])], names, 1, 2);
    expect(groups.map((g) => g.age)).toEqual([2]);
    expect(groups[0]!.lines.map((line) => line.kind)).toEqual(['roomHints', 'build']);
  });
  it('places each change before the action of its ply, in that ply’s age', () => {
    const groups = groupGameLog(
      history,
      [
        change(2, 0, [], ['prices']),
        change(0, 0, ['advisor'], []),
        change(3, undefined, ['prices'], [])
      ],
      names
    );
    expect(groups.map((g) => g.age)).toEqual([0, 1]);
    expect(groups[0]!.lines.map((l) => l.kind)).toEqual(['roomHints', 'draft', 'draft']);
    expect(groups[1]!.lines.map((l) => l.kind)).toEqual([
      'ageStart',
      'roomHints',
      'build',
      'roomHints'
    ]);
    expect(groups[1]!.lines[1]!.text).toBe('Sara changed the hints: turned off Prices.');
  });

  it('stops at the ply on screen, room log included', () => {
    const groups = groupGameLog(
      history,
      [change(1, 1, [], ['tempo']), change(2, 1, [], ['science'])],
      names,
      1
    );
    expect(groups.flatMap((g) => g.lines.map((l) => l.kind))).toEqual(['draft', 'roomHints']);
  });

  it('shows the changes of a game with no move yet', () => {
    const groups = groupGameLog([], [change(0, 0, [], ['wonders'])], names);
    expect(groups).toEqual([
      {
        age: 0,
        lines: [{ kind: 'roomHints', seat: 0, text: 'Sara changed the hints: turned off Wonders.' }]
      }
    ]);
  });
});
