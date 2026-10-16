import { describe, expect, it } from 'vitest';
import type { GameEvent } from '$lib/wire';
import {
  BLOCK_CAP_MS,
  cueOffsets,
  EMPTY_TIMELINE,
  FLIP_DELAY_MS,
  FLIP_MS,
  FLIP_STAGGER_MS,
  FLOAT_MS,
  PAWN_STEP_MS,
  planTimeline,
  TAKE_MS,
  TWEEN_MS,
  type PlanFrame
} from './plan';

function ev(kind: GameEvent['kind'], fields: Partial<GameEvent> = {}): GameEvent {
  return { kind, private: false, ...fields };
}

function frame(
  version: number,
  events: GameEvent[] = [],
  coins: [number, number] = [7, 7],
  code = 'ROOM'
): PlanFrame {
  return { code, version, events, view: { players: [{ coins: coins[0] }, { coins: coins[1] }] } };
}

const before = frame(10);

describe('planTimeline: live events only', () => {
  it('plans nothing without a frame before it, across rooms, or for an old version', () => {
    const build = [ev('build', { seat: 0, card: 'baths' })];
    expect(planTimeline(frame(11, build), null)).toBe(EMPTY_TIMELINE);
    expect(planTimeline(frame(11, build, [7, 7], 'OTHER'), before)).toBe(EMPTY_TIMELINE);
    expect(planTimeline(frame(10, build), before)).toBe(EMPTY_TIMELINE);
    expect(planTimeline(frame(9, build), before)).toBe(EMPTY_TIMELINE);
  });

  it('plans nothing for a frame with no events, or only bookkeeping ones', () => {
    expect(planTimeline(frame(11), before)).toBe(EMPTY_TIMELINE);
    expect(planTimeline(frame(11, [ev('extraTurn', { seat: 0 })]), before)).toBe(EMPTY_TIMELINE);
  });
});

describe('planTimeline: per event kind', () => {
  it('build: a take to the city that holds input until it lands', () => {
    const t = planTimeline(frame(11, [ev('build', { seat: 1, card: 'baths' })]), before);
    expect(t.steps).toEqual([
      { kind: 'take', seat: 1, card: 'baths', dest: 'city', at: 0, dur: TAKE_MS }
    ]);
    expect(t.blockingMs).toBe(TAKE_MS);
    expect(t.totalMs).toBe(TAKE_MS);
  });

  it('discard: a take to the discard and the coins it brings', () => {
    const t = planTimeline(
      frame(11, [ev('discard', { seat: 0, card: 'baths', amount: 2 })], [9, 7]),
      before
    );
    expect(t.steps).toEqual([
      { kind: 'take', seat: 0, card: 'baths', dest: 'discard', at: 0, dur: TAKE_MS },
      {
        kind: 'coins',
        seat: 0,
        from: 7,
        to: 9,
        at: TAKE_MS,
        dur: FLOAT_MS,
        tween: TWEEN_MS
      }
    ]);
  });

  it('wonder: a take that tucks the card under the wonder', () => {
    const t = planTimeline(
      frame(11, [ev('wonder', { seat: 0, wonder: 'pyramids', card: 'baths' })]),
      before
    );
    expect(t.steps[0]).toMatchObject({ kind: 'take', dest: 'wonder', wonder: 'pyramids' });
  });

  it('reveal: flips after the take lands, staggered', () => {
    const t = planTimeline(
      frame(11, [
        ev('build', { seat: 0, card: 'baths' }),
        ev('reveal', { slot: 3, card: 'quarry' }),
        ev('reveal', { slot: 4, card: 'garrison' })
      ]),
      before
    );
    expect(t.steps.filter((s) => s.kind === 'flip')).toEqual([
      { kind: 'flip', slot: 3, card: 'quarry', at: TAKE_MS + FLIP_DELAY_MS, dur: FLIP_MS },
      {
        kind: 'flip',
        slot: 4,
        card: 'garrison',
        at: TAKE_MS + FLIP_DELAY_MS + FLIP_STAGGER_MS,
        dur: FLIP_MS
      }
    ]);
    expect(t.blockingMs).toBe(TAKE_MS);
    expect(t.totalMs).toBe(TAKE_MS + FLIP_DELAY_MS + FLIP_STAGGER_MS + FLIP_MS);
  });

  it('reveal: ignores a reveal that came before a new age was laid out', () => {
    const t = planTimeline(
      frame(11, [
        ev('build', { seat: 0, card: 'baths' }),
        ev('reveal', { slot: 3, card: 'quarry' }),
        ev('ageStart', { age: 2 })
      ]),
      before
    );
    expect(t.steps.map((s) => s.kind)).toEqual(['take']);
  });

  it('pay: the payer, and the opponent when part of it went there', () => {
    const t = planTimeline(
      frame(
        11,
        [ev('build', { seat: 0, card: 'baths' }), ev('pay', { seat: 0, amount: 4, toOpponent: 2 })],
        [3, 9]
      ),
      before
    );
    const coins = t.steps.filter((s) => s.kind === 'coins');
    expect(coins).toMatchObject([
      { seat: 0, from: 7, to: 3, at: TAKE_MS },
      { seat: 1, from: 7, to: 9, at: TAKE_MS }
    ]);
  });

  it('coins: one step per seat, with the views as the counter ends', () => {
    const t = planTimeline(
      frame(
        11,
        [
          ev('build', { seat: 1, card: 'tavern' }),
          ev('coins', { seat: 1, amount: 4, balance: 11 }),
          ev('coins', { seat: 1, amount: 1, balance: 12 })
        ],
        [7, 12]
      ),
      before
    );
    expect(t.steps.filter((s) => s.kind === 'coins')).toMatchObject([{ seat: 1, from: 7, to: 12 }]);
  });

  it('coins: nothing when the counter ends where it started', () => {
    const t = planTimeline(
      frame(
        11,
        [ev('build', { seat: 0, card: 'baths' }), ev('pay', { seat: 0, amount: 2 })],
        [7, 7]
      ),
      before
    );
    expect(t.steps.map((s) => s.kind)).toEqual(['take']);
  });

  it('coins: a frame with no take floats at once and holds no input', () => {
    const t = planTimeline(frame(11, [ev('coins', { seat: 0, amount: 3 })], [10, 7]), before);
    expect(t.steps).toMatchObject([{ kind: 'coins', at: 0 }]);
    expect(t.blockingMs).toBe(0);
  });

  it('pawn: walks cell by cell, after the take', () => {
    const t = planTimeline(
      frame(11, [
        ev('build', { seat: 0, card: 'garrison' }),
        ev('pawn', { seat: 0, from: 1, to: -2 })
      ]),
      before
    );
    expect(t.steps[1]).toEqual({
      kind: 'pawn',
      seat: 0,
      from: 1,
      to: -2,
      at: TAKE_MS,
      dur: 3 * PAWN_STEP_MS
    });
  });

  it('pawn: several pushes in one frame make one walk; none when it ends where it began', () => {
    const t = planTimeline(
      frame(11, [
        ev('build', { seat: 0, card: 'garrison' }),
        ev('pawn', { seat: 0, from: 0, to: 2 }),
        ev('pawn', { seat: 0, from: 2, to: 3 })
      ]),
      before
    );
    expect(t.steps[1]).toMatchObject({ kind: 'pawn', from: 0, to: 3, dur: 3 * PAWN_STEP_MS });
    const none = planTimeline(
      frame(11, [
        ev('build', { seat: 0, card: 'garrison' }),
        ev('pawn', { seat: 0, from: 2, to: 2 })
      ]),
      before
    );
    expect(none.steps.map((s) => s.kind)).toEqual(['take']);
  });

  it('leaves the other slice its events: looting, tokens, destroy, the draft, the end', () => {
    for (const e of [
      ev('loot', { seat: 1, amount: 2, value: 2 }),
      ev('token', { seat: 0, token: 'law' }),
      ev('destroy', { seat: 0, card: 'quarry' }),
      ev('draft', { seat: 0, wonder: 'pyramids' }),
      ev('gameOver', { victory: 'civilian', winner: 0 })
    ]) {
      expect(planTimeline(frame(11, [e]), before), e.kind).toBe(EMPTY_TIMELINE);
    }
  });

  it('never holds input longer than the cap', () => {
    expect(TAKE_MS).toBeLessThanOrEqual(BLOCK_CAP_MS);
  });
});

describe('cueOffsets', () => {
  it('times each cue kind by its first step', () => {
    const t = planTimeline(
      frame(
        11,
        [
          ev('build', { seat: 0, card: 'garrison' }),
          ev('pay', { seat: 0, amount: 2 }),
          ev('pawn', { seat: 0, from: 0, to: 1 }),
          ev('reveal', { slot: 2, card: 'quarry' })
        ],
        [5, 7]
      ),
      before
    );
    expect(cueOffsets(t)).toEqual({
      slide: TAKE_MS,
      flip: TAKE_MS + FLIP_DELAY_MS,
      coin: TAKE_MS,
      drum: TAKE_MS
    });
    expect(cueOffsets(EMPTY_TIMELINE)).toEqual({});
  });
});
