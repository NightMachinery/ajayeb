import { describe, expect, it } from 'vitest';
import { fixtureNames, fixtures } from '$lib/wire/fixtures.gen';
import type { GameEvent, HistoryEntry, State } from '$lib/wire';
import { entriesSinceViewer, summarize, summarizeSince, summaryViewer } from './move-summary';

function ev(e: Partial<GameEvent> & Pick<GameEvent, 'kind'>): GameEvent {
  return { private: false, ...e };
}

function entry(ply: number, seat: 0 | 1, events: GameEvent[]): HistoryEntry {
  return { ply, seat, events };
}

/** The recorded join frame of an online room, whose history holds eight moves. */
function joinHistory(): readonly HistoryEntry[] {
  const frame = fixtures[fixtureNames.indexOf('state-join-history')] as State | undefined;
  if (!frame?.history) throw new Error('no recorded join history');
  return frame.history;
}

describe('summaryViewer', () => {
  it('is the seat to move in hot-seat, your own seat otherwise, none for a spectator', () => {
    expect(summaryViewer('hotseat', [0, 1], 1)).toBe(1);
    expect(summaryViewer('hotseat', [0, 1], undefined)).toBeNull();
    expect(summaryViewer('online', [1], 0)).toBe(1);
    expect(summaryViewer('ai', [0], 1)).toBe(0);
    expect(summaryViewer('online', [], 0)).toBeNull();
  });
});

describe('entriesSinceViewer', () => {
  const h = [
    entry(0, 0, [ev({ kind: 'build', seat: 0, card: 'a' })]),
    entry(1, 1, [ev({ kind: 'build', seat: 1, card: 'b' })]),
    entry(2, 1, [ev({ kind: 'build', seat: 1, card: 'c' })])
  ];
  it("takes the other side's entries after the viewer's last one", () => {
    expect(entriesSinceViewer(h, 0).map((e) => e.ply)).toEqual([1, 2]);
  });
  it('is empty when the viewer moved last', () => {
    expect(entriesSinceViewer(h, 1)).toEqual([]);
    expect(summarizeSince(h, 1)).toBeNull();
  });
  it('takes the whole history when the viewer never moved', () => {
    expect(entriesSinceViewer(h.slice(1), 0)).toHaveLength(2);
  });
  it('is empty for an empty history', () => {
    expect(summarizeSince([], 0)).toBeNull();
  });
});

describe('summarize', () => {
  it('reads the recorded join history: the opponent built a Tavern and a card turned up', () => {
    const s = summarizeSince(joinHistory(), 0);
    expect(s).not.toBeNull();
    expect(s!.seat).toBe(1);
    expect(s!.plies).toEqual([7]);
    expect(s!.actions).toEqual([{ ply: 7, kind: 'build', seat: 1, card: 'tavern' }]);
    // The Tavern's coins went to seat 1, nothing to seat 0.
    expect(s!.coins[0]).toBe(0);
    expect(s!.coins[1]).toBeGreaterThan(0);
    expect(s!.revealed).toEqual(['wood-reserve']);
  });

  it("consolidates an extra turn's moves and nets the coins without counting twice", () => {
    const h = [
      entry(10, 0, [ev({ kind: 'build', seat: 0, card: 'x' })]),
      entry(11, 1, [
        ev({ kind: 'wonder', seat: 1, wonder: 'piraeus', card: 'baths' }),
        ev({ kind: 'pay', seat: 1, amount: 5, toOpponent: 2 }),
        ev({ kind: 'extraTurn', seat: 1 }),
        ev({ kind: 'reveal', slot: 3, card: 'quarry' })
      ]),
      entry(12, 1, [
        ev({ kind: 'build', seat: 1, card: 'barracks' }),
        ev({ kind: 'pawn', seat: 1, from: 0, to: 1 }),
        ev({ kind: 'pawn', seat: 1, from: 1, to: 3 }),
        ev({ kind: 'loot', seat: 0, amount: 2, value: 2 })
      ]),
      entry(13, 1, [
        ev({ kind: 'discard', seat: 1, card: 'quarry', amount: 3 }),
        ev({ kind: 'coins', seat: 1, amount: -1, balance: 4, token: 'law' })
      ])
    ];
    const s = summarizeSince(h, 0)!;
    expect(s.plies).toEqual([11, 12, 13]);
    expect(s.actions.map((a) => [a.kind, a.wonder ?? a.card])).toEqual([
      ['wonder', 'piraeus'],
      ['build', 'barracks'],
      ['discard', 'quarry']
    ]);
    // Seat 1: -5 paid, +3 discard, -1 coins. Seat 0: +2 trade, -2 looted.
    expect(s.coins).toEqual([0, -3]);
    expect(s.paid).toEqual([0, 5]);
    expect(s.paidToOpponent).toEqual([0, 2]);
    expect(s.pawn).toEqual({ from: 0, to: 3 });
    expect(s.loot).toEqual([{ seat: 0, amount: 2 }]);
    expect(s.revealed).toEqual(['quarry']);
    expect(s.extraTurns).toBe(1);
    expect(s.gameOver).toBeNull();
  });

  it('collects science, tokens, destroyed cards, ages and the end of the game', () => {
    const s = summarize(
      [
        entry(20, 1, [
          ev({ kind: 'build', seat: 1, card: 'library' }),
          ev({ kind: 'pair', seat: 1, symbol: 'wheel' })
        ]),
        entry(21, 1, [ev({ kind: 'token', seat: 1, token: 'law', fromLibrary: false })]),
        entry(22, 1, [
          ev({ kind: 'wonder', seat: 1, wonder: 'statue-of-zeus', card: 'tavern' }),
          ev({ kind: 'effectSkipped', seat: 1, wonder: 'statue-of-zeus', skipped: 'destroyCard' })
        ]),
        entry(23, 1, [
          ev({ kind: 'destroy', seat: 1, card: 'quarry' }),
          ev({ kind: 'wonderRemoved', seat: 0, wonder: 'colossus' }),
          ev({ kind: 'ageStart', age: 3 })
        ]),
        entry(24, 1, [
          ev({ kind: 'resign', seat: 1 }),
          ev({ kind: 'gameOver', victory: 'forfeit', winner: 0 })
        ])
      ],
      1
    );
    expect(s.pairs).toEqual([{ seat: 1, symbol: 'wheel' }]);
    expect(s.tokens).toEqual([{ seat: 1, token: 'law', fromLibrary: false }]);
    expect(s.skipped).toEqual([{ seat: 1, kind: 'destroyCard' }]);
    // The destroy event names the destroyer; the card was seat 0's.
    expect(s.destroyed).toEqual([{ seat: 0, card: 'quarry' }]);
    expect(s.wondersRemoved).toEqual([{ seat: 0, wonder: 'colossus' }]);
    expect(s.agesStarted).toEqual([3]);
    expect(s.actions.map((a) => a.kind)).toEqual(['build', 'token', 'wonder', 'destroy', 'resign']);
    expect(s.gameOver).toEqual({ victory: 'forfeit', winner: 0 });
  });

  it("keeps a draft's automatic last wonder with the seat it went to", () => {
    const s = summarize(
      [
        entry(2, 1, [
          ev({ kind: 'draft', seat: 1, wonder: 'hanging-gardens' }),
          ev({ kind: 'draft', seat: 0, wonder: 'pyramids', auto: true }),
          ev({ kind: 'draftRound', round: 2 })
        ])
      ],
      1
    );
    expect(s.actions).toEqual([
      { ply: 2, kind: 'draft', seat: 1, wonder: 'hanging-gardens' },
      { ply: 2, kind: 'draft', seat: 0, wonder: 'pyramids', auto: true }
    ]);
  });

  it('reads a starter choice', () => {
    const s = summarize([entry(40, 0, [ev({ kind: 'starterChosen', seat: 0, starter: 1 })])], 0);
    expect(s.actions).toEqual([{ ply: 40, kind: 'starterChosen', seat: 0, starter: 1 }]);
  });
});
