import { describe, expect, it } from 'vitest';
import { fixtureNames, fixtures } from '$lib/wire/fixtures.gen';
import type { GameEvent, HistoryEntry, State } from '$lib/wire';
import {
  entriesSinceViewer,
  entryLines,
  finalTurn,
  lastPly,
  latestSummary,
  summarize,
  summarizeSince,
  summaryTrigger,
  summaryViewer,
  type TriggerFrame
} from './move-summary';

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

describe('lines', () => {
  it('gives each move of an extra turn its own line, with what it paid and brought', () => {
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
      ])
    ];
    const s = summarizeSince(h, 0)!;
    expect(s.lines.map((l) => [l.kind, l.ply])).toEqual([
      ['wonder', 11],
      ['extraTurn', 11],
      ['build', 12]
    ]);
    const [wonder, , build] = s.lines;
    expect(wonder).toMatchObject({ wonder: 'piraeus', card: 'baths', paid: 5, toOpponent: 2 });
    expect(wonder!.pawn).toBeNull();
    expect(build).toMatchObject({
      card: 'barracks',
      paid: 0,
      pawn: { from: 0, to: 3 },
      loot: [{ seat: 0, amount: 2, value: 2 }]
    });
  });

  it("puts a bot's reply and the choice it opened in one summary", () => {
    const h = [
      entry(30, 0, [ev({ kind: 'build', seat: 0, card: 'stable' })]),
      entry(31, 1, [
        ev({ kind: 'build', seat: 1, card: 'library' }),
        ev({ kind: 'pay', seat: 1, amount: 1, toOpponent: 0 }),
        ev({ kind: 'pair', seat: 1, symbol: 'quill' })
      ]),
      entry(32, 1, [
        ev({ kind: 'token', seat: 1, token: 'law', fromLibrary: false }),
        ev({ kind: 'coins', seat: 1, amount: 0, balance: 4, token: 'law' })
      ])
    ];
    const s = summarizeSince(h, 0)!;
    expect(s.plies).toEqual([31, 32]);
    expect(lastPly(s)).toBe(32);
    expect(s.lines.map((l) => l.kind)).toEqual(['build', 'token']);
    expect(s.lines[0]).toMatchObject({ card: 'library', paid: 1, pairs: ['quill'] });
    expect(s.lines[1]).toMatchObject({ token: 'law', fromLibrary: false });
  });

  it('gives an age change, a removed wonder, a skipped choice and the end their own lines', () => {
    const lines = entryLines(
      entry(50, 1, [
        ev({ kind: 'wonder', seat: 1, wonder: 'statue-of-zeus', card: 'tavern' }),
        ev({ kind: 'wonderRemoved', seat: 0, wonder: 'colossus' }),
        ev({ kind: 'effectSkipped', seat: 1, wonder: 'statue-of-zeus', skipped: 'destroyCard' }),
        ev({ kind: 'discard', seat: 1, card: 'quarry', amount: 3 }),
        ev({ kind: 'coins', seat: 0, amount: -3, balance: 1, wonder: 'appian-way' }),
        ev({ kind: 'ageStart', age: 2 }),
        ev({ kind: 'gameOver', victory: 'civilian', winner: 1 })
      ])
    );
    expect(lines.map((l) => l.kind)).toEqual([
      'wonder',
      'wonderRemoved',
      'skipped',
      'discard',
      'ageStart',
      'gameOver'
    ]);
    expect(lines[1]).toMatchObject({ seat: 0, wonder: 'colossus' });
    expect(lines[2]).toMatchObject({ skipped: 'destroyCard', wonder: 'statue-of-zeus' });
    expect(lines[3]).toMatchObject({ amount: 3, coins: [-3, 0] });
    expect(lines[4]).toMatchObject({ age: 2 });
    expect(lines[5]).toMatchObject({ victory: 'civilian', winner: 1 });
  });
});

describe('latestSummary', () => {
  const h = [
    entry(0, 0, [ev({ kind: 'build', seat: 0, card: 'a' })]),
    entry(1, 1, [ev({ kind: 'build', seat: 1, card: 'b' })]),
    entry(2, 1, [ev({ kind: 'build', seat: 1, card: 'c' })]),
    entry(3, 0, [ev({ kind: 'build', seat: 0, card: 'd' })])
  ];
  it("keeps the other side's last run after the viewer has moved since", () => {
    expect(summarizeSince(h, 0)).toBeNull();
    expect(latestSummary(h, 0)!.plies).toEqual([1, 2]);
  });
  it("is the moves since the viewer's own last one when the other side moved last", () => {
    expect(latestSummary(h, 1)!.plies).toEqual([3]);
  });
  it('is null before the other side has moved', () => {
    expect(latestSummary(h.slice(0, 1), 0)).toBeNull();
    expect(latestSummary([], 1)).toBeNull();
  });
  it('follows the hot-seat viewer from one player to the next', () => {
    // Seat 1 is to move after ply 0: it sees seat 0's move; after ply 2,
    // seat 0 is to move and sees seat 1's two moves.
    const v1 = summaryViewer('hotseat', [0, 1], 1)!;
    expect(latestSummary(h.slice(0, 1), v1)!.plies).toEqual([0]);
    const v0 = summaryViewer('hotseat', [0, 1], 0)!;
    expect(latestSummary(h.slice(0, 3), v0)!.plies).toEqual([1, 2]);
  });
});

describe('summaryTrigger', () => {
  function frame(over: Partial<TriggerFrame> & { toMove?: number; phase?: string }): TriggerFrame {
    const { toMove, phase, ...rest } = over;
    return {
      code: 'ROOM',
      version: 5,
      events: [ev({ kind: 'build', seat: 1, card: 'b' })],
      room: { mode: 'ai' },
      you: { seats: [0] },
      view: {
        toMove: 'toMove' in over ? toMove : 0,
        phase: (phase ?? 'play') as TriggerFrame['view']['phase']
      },
      ...rest
    };
  }
  const prev = frame({ version: 4, events: [], toMove: 1 });
  const h = [
    entry(0, 0, [ev({ kind: 'build', seat: 0, card: 'a' })]),
    entry(1, 1, [ev({ kind: 'build', seat: 1, card: 'b' })])
  ];

  it("opens on a live frame that hands the turn back after the bot's move", () => {
    expect(summaryTrigger(prev, frame({}), h)).toBe(1);
  });
  it('waits while the bot still has a choice or an extra turn to take', () => {
    expect(summaryTrigger(prev, frame({ toMove: 1 }), h)).toBeNull();
  });
  it("never opens for the viewer's own move", () => {
    expect(summaryTrigger(prev, frame({ toMove: 1 }), h.slice(0, 1))).toBeNull();
  });
  it('never opens on history: a join, a resync or an undo', () => {
    expect(summaryTrigger(null, frame({}), h)).toBeNull();
    expect(summaryTrigger(prev, frame({ history: h }), h)).toBeNull();
    expect(summaryTrigger(frame({ code: 'OTHER', version: 4 }), frame({}), h)).toBeNull();
    expect(summaryTrigger(frame({ version: 5 }), frame({}), h)).toBeNull();
  });
  it('opens unseen history on reload, reconnect and wake, only once', () => {
    expect(summaryTrigger(null, frame({ history: h }), h, 0)).toBe(1);
    expect(summaryTrigger(prev, frame({ history: h }), h, 0)).toBe(1);
    expect(summaryTrigger(prev, frame({ events: [] }), h, 0)).toBe(1);
    expect(summaryTrigger(null, frame({ history: h }), h, 1)).toBeNull();
    expect(summaryTrigger(null, frame({ history: h, toMove: 1 }), h, 0)).toBeNull();
  });
  it('never opens on a frame without events (a presence change, a thinking frame)', () => {
    expect(summaryTrigger(prev, frame({ events: [] }), h)).toBeNull();
  });
  it('never opens during the wonder draft', () => {
    expect(summaryTrigger(prev, frame({ phase: 'wonderDraft' }), h)).toBeNull();
  });
  it('never opens at the end of the game: the result screen shows the final turn', () => {
    const over = frame({ toMove: undefined, phase: 'gameOver' });
    expect(summaryTrigger(prev, over, h)).toBeNull();
    expect(summaryTrigger(prev, over, h.slice(0, 1))).toBeNull();
    expect(
      summaryTrigger(null, frame({ history: h, toMove: undefined, phase: 'gameOver' }), h, 0)
    ).toBeNull();
  });
  it('opens in hot-seat for the player now to move, about the one who moved before', () => {
    const hot = { room: { mode: 'hotseat' as const }, you: { seats: [0, 1] } };
    expect(summaryTrigger(prev, frame({ ...hot, toMove: 0 }), h)).toBe(1);
    // An extra turn: the same player is to move again.
    expect(summaryTrigger(prev, frame({ ...hot, toMove: 1 }), h)).toBeNull();
    // Nobody is to move at the end of a hot-seat game.
    expect(
      summaryTrigger(prev, frame({ ...hot, toMove: undefined, phase: 'gameOver' }), h)
    ).toBeNull();
  });
  it('never opens for a spectator', () => {
    expect(
      summaryTrigger(prev, frame({ room: { mode: 'online' }, you: { seats: [] } }), h)
    ).toBeNull();
  });
});

describe('finalTurn', () => {
  it("takes the last seat's whole run: a wonder's extra turn and the choices in it", () => {
    const s = finalTurn([
      entry(30, 0, [ev({ kind: 'build', seat: 0, card: 'a' })]),
      entry(31, 1, [
        ev({ kind: 'wonder', seat: 1, wonder: 'piraeus', card: 'b' }),
        ev({ kind: 'extraTurn', seat: 1 })
      ]),
      entry(32, 1, [
        ev({ kind: 'build', seat: 1, card: 'library' }),
        ev({ kind: 'pair', seat: 1, symbol: 'wheel' })
      ]),
      entry(33, 1, [
        ev({ kind: 'token', seat: 1, token: 'law', fromLibrary: false }),
        ev({ kind: 'gameOver', victory: 'science', winner: 1 })
      ])
    ])!;
    expect(s.seat).toBe(1);
    expect(s.plies).toEqual([31, 32, 33]);
    expect(s.actions.map((a) => a.kind)).toEqual(['wonder', 'build', 'token']);
    expect(s.gameOver).toEqual({ victory: 'science', winner: 1 });
    expect(s.lines.map((l) => l.kind)).toContain('gameOver');
  });
  it('is one move when the other side moved just before', () => {
    const s = finalTurn([
      entry(0, 0, [ev({ kind: 'build', seat: 0, card: 'a' })]),
      entry(1, 1, [ev({ kind: 'build', seat: 1, card: 'b' })]),
      entry(2, 0, [
        ev({ kind: 'discard', seat: 0, card: 'c', amount: 3 }),
        ev({ kind: 'gameOver', victory: 'civilian', winner: 1 })
      ])
    ])!;
    expect(s.seat).toBe(0);
    expect(s.plies).toEqual([2]);
  });
  it('takes the whole history when one seat made all of it', () => {
    expect(finalTurn([entry(0, 1, [ev({ kind: 'build', seat: 1, card: 'a' })])])!.plies).toEqual([
      0
    ]);
  });
  it('is null for an empty history', () => {
    expect(finalTurn([])).toBeNull();
  });
});
