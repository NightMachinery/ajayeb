// The planner's event animations (M10.animations-events): looting, progress
// tokens and the Great Library, the science pair, destroy and rebuild, the
// age deal, the draft and the end, with their sound and vibration timing.
import { describe, expect, it } from 'vitest';
import type { GameEvent } from '$lib/wire';
import { HAPTIC_LOOT, HAPTIC_MILITARY, HAPTIC_TAKE } from '../cues';
import {
  cueOffsets,
  DEAL_EACH_MS,
  DEAL_FADE_MS,
  DEAL_LEAD_MS,
  DEAL_STAGGER_MS,
  DESTROY_MS,
  DRAFT_MS,
  DRAFT_STAGGER_MS,
  DRAW_MS,
  DRAW_STAGGER_MS,
  EMPTY_TIMELINE,
  GLOW_MS,
  hapticOffset,
  HOLD_MS,
  LAND_MS,
  LOOT_FLIP_MS,
  PAWN_STEP_MS,
  planTimeline,
  REBUILD_MS,
  TAKE_MS,
  TOKEN_MS,
  VICTORY_MS,
  type PlanFrame,
  type PlanView,
  type Step
} from './plan';

function ev(kind: GameEvent['kind'], fields: Partial<GameEvent> = {}): GameEvent {
  return { kind, private: false, ...fields };
}

function view(coins: [number, number], extra: Partial<PlanView> = {}): PlanView {
  return { players: [{ coins: coins[0] }, { coins: coins[1] }], ...extra };
}

function frame(version: number, events: GameEvent[], v: PlanView = view([7, 7])): PlanFrame {
  return { code: 'ROOM', version, events, view: v };
}

const before = frame(10, [], view([7, 7]));

function only<K extends Step['kind']>(steps: Step[], kind: K): Extract<Step, { kind: K }>[] {
  return steps.filter((s): s is Extract<Step, { kind: K }> => s.kind === kind);
}

describe('looting', () => {
  // looting-double: four shields push the pawn from 2 to 6, past seat 1's
  // tokens at 3 (2 coins) and 6 (5 coins).
  const looting = (present: boolean) => ({
    looting: [
      { at: -6, present: true },
      { at: -3, present: true },
      { at: 3, present },
      { at: 6, present }
    ]
  });
  const events = [
    ev('build', { seat: 0, card: 'arsenal' }),
    ev('pay', { seat: 0, amount: 3 }),
    ev('pawn', { seat: 0, from: 2, to: 6 }),
    ev('loot', { seat: 1, amount: 2, value: 2 }),
    ev('loot', { seat: 1, amount: 5, value: 5 })
  ];
  const prev = frame(10, [], view([40, 20], { military: looting(true) }));

  it('flips each token as the pawn reaches it, and drops the victim’s coins there', () => {
    const t = planTimeline(frame(11, events, view([37, 13], { military: looting(false) })), prev);
    expect(only(t.steps, 'loot')).toEqual([
      {
        kind: 'loot',
        seat: 1,
        cell: 3,
        amount: 2,
        from: 20,
        to: 18,
        at: TAKE_MS + 1 * PAWN_STEP_MS,
        dur: expect.any(Number),
        flip: LOOT_FLIP_MS,
        fade: expect.any(Number),
        tween: expect.any(Number)
      },
      {
        kind: 'loot',
        seat: 1,
        cell: 6,
        amount: 5,
        from: 18,
        to: 13,
        at: TAKE_MS + 4 * PAWN_STEP_MS,
        dur: expect.any(Number),
        flip: LOOT_FLIP_MS,
        fade: expect.any(Number),
        tween: expect.any(Number)
      }
    ]);
    // The payer's coins float as before; the victim's only with its tokens.
    expect(only(t.steps, 'coins').map((s) => s.seat)).toEqual([0]);
  });

  it('a victim paid by a trade in the same move: its float stops short of the looting', () => {
    const trade = [
      ev('build', { seat: 0, card: 'arsenal' }),
      ev('pay', { seat: 0, amount: 4, toOpponent: 3 }),
      ev('pawn', { seat: 0, from: 2, to: 3 }),
      ev('loot', { seat: 1, amount: 2, value: 2 })
    ];
    const lootAt3 = {
      looting: [
        { at: 3, present: false },
        { at: 6, present: true }
      ]
    };
    const t = planTimeline(
      frame(11, trade, view([36, 21], { military: lootAt3 })),
      frame(10, [], view([40, 20], { military: looting(true) }))
    );
    expect(only(t.steps, 'coins').find((s) => s.seat === 1)).toMatchObject({ from: 20, to: 23 });
    expect(only(t.steps, 'loot')).toMatchObject([{ seat: 1, from: 23, to: 21, cell: 3 }]);
  });

  it('without the views it still flips, after the pawn', () => {
    const t = planTimeline(
      { code: 'ROOM', version: 11, events },
      { code: 'ROOM', version: 10, events: [] }
    );
    const pawn = only(t.steps, 'pawn')[0]!;
    expect(only(t.steps, 'loot').map((s) => s.at)).toEqual([
      pawn.at + pawn.dur,
      pawn.at + pawn.dur + LOOT_FLIP_MS
    ]);
  });

  it('times the drum by the pawn and looting by the first flip', () => {
    const t = planTimeline(frame(11, events, view([37, 13], { military: looting(false) })), prev);
    const first = only(t.steps, 'loot')[0]!;
    expect(cueOffsets(t).drum).toBe(TAKE_MS);
    expect(cueOffsets(t).loot).toBe(first.at);
    expect(hapticOffset(t, HAPTIC_LOOT)).toBe(first.at);
    expect(hapticOffset(t, HAPTIC_MILITARY)).toBe(TAKE_MS);
  });
});

describe('progress tokens', () => {
  it('a token taken from the board flies to the taker, the coins it brings after it', () => {
    const t = planTimeline(
      frame(
        11,
        [
          ev('token', { seat: 0, token: 'agriculture', fromLibrary: false }),
          ev('coins', { seat: 0, amount: 6 })
        ],
        view([13, 7])
      ),
      before
    );
    expect(t.steps[0]).toEqual({
      kind: 'token',
      seat: 0,
      token: 'agriculture',
      library: false,
      at: 0,
      dur: TOKEN_MS
    });
    expect(only(t.steps, 'coins')).toMatchObject([{ seat: 0, at: TOKEN_MS, from: 7, to: 13 }]);
    expect(t.blockingMs).toBe(Math.min(TOKEN_MS - LAND_MS, HOLD_MS));
    expect(cueOffsets(t).token).toBe(TOKEN_MS);
  });

  it('a token from the Great Library says so', () => {
    const t = planTimeline(
      frame(11, [ev('token', { seat: 1, token: 'law', fromLibrary: true })]),
      before
    );
    expect(t.steps[0]).toMatchObject({ kind: 'token', library: true });
  });

  it('the Great Library deals its three tokens one after another after the tuck', () => {
    const drawer = [
      ev('wonder', { seat: 0, wonder: 'great-library', card: 'rostrum' }),
      ev('libraryDraw', { seat: 0, place: 0, token: 'law', private: true }),
      ev('libraryDraw', { seat: 0, place: 1, token: 'philosophy', private: true }),
      ev('libraryDraw', { seat: 0, place: 2, token: 'strategy', private: true })
    ];
    const t = planTimeline(frame(11, drawer), before);
    expect(only(t.steps, 'draw')).toMatchObject([
      { seat: 0, place: 0, token: 'law', at: TAKE_MS, each: DRAW_MS },
      { seat: 0, place: 1, token: 'philosophy', at: TAKE_MS + DRAW_STAGGER_MS },
      { seat: 0, place: 2, token: 'strategy', at: TAKE_MS + 2 * DRAW_STAGGER_MS }
    ]);
  });

  it('never names a drawn token to a viewer whose events do not', () => {
    // The opponent's (and a spectator's) events: the drawer and the place only.
    const others = [
      ev('wonder', { seat: 0, wonder: 'great-library', card: 'rostrum' }),
      ev('libraryDraw', { seat: 0, place: 0 }),
      ev('libraryDraw', { seat: 0, place: 1 }),
      ev('libraryDraw', { seat: 0, place: 2 })
    ];
    const t = planTimeline(frame(11, others), before);
    const draws = only(t.steps, 'draw');
    expect(draws.map((d) => d.place)).toEqual([0, 1, 2]);
    for (const d of draws) expect('token' in d).toBe(false);
    const text = JSON.stringify(t);
    for (const token of ['law', 'philosophy', 'strategy']) expect(text).not.toContain(token);
  });
});

describe('science pairs', () => {
  it('a pair glows in the taker’s tracker once the card lands, with the chime', () => {
    const t = planTimeline(
      frame(11, [
        ev('build', { seat: 0, card: 'library' }),
        ev('pair', { seat: 0, symbol: 'quill' }),
        ev('pair', { seat: 0, symbol: 'quill' })
      ]),
      before
    );
    expect(only(t.steps, 'glow')).toEqual([
      { kind: 'glow', seat: 0, symbol: 'quill', at: TAKE_MS, dur: GLOW_MS }
    ]);
    expect(cueOffsets(t).chime).toBe(TAKE_MS);
  });
});

describe('destroy and rebuild', () => {
  it('a destroyed card leaves the opponent’s city at once', () => {
    const t = planTimeline(frame(11, [ev('destroy', { seat: 0, card: 'quarry' })]), before);
    expect(t.steps).toEqual([{ kind: 'destroy', seat: 0, card: 'quarry', at: 0, dur: DESTROY_MS }]);
    expect(t.blockingMs).toBe(HOLD_MS);
    expect(cueOffsets(t).destroy).toBe(0);
  });

  it('a card rebuilt from the discard pile flies to the city, the slide as it lands', () => {
    const t = planTimeline(frame(11, [ev('rebuild', { seat: 0, card: 'baths' })]), before);
    expect(t.steps).toEqual([{ kind: 'rebuild', seat: 0, card: 'baths', at: 0, dur: REBUILD_MS }]);
    expect(cueOffsets(t).rebuild).toBe(REBUILD_MS);
    expect(hapticOffset(t, HAPTIC_TAKE)).toBe(REBUILD_MS);
  });
});

describe('the age deal', () => {
  const structure = (n: number) => Array.from({ length: n }, () => ({ state: 'faceDown' }));

  it('deals every card of the new structure, staggered, beside the last card’s take', () => {
    const t = planTimeline(
      frame(
        11,
        [ev('discard', { seat: 0, card: 'theater', amount: 2 }), ev('ageStart', { age: 2 })],
        view([9, 7], { structure: structure(20) })
      ),
      before
    );
    const deal = only(t.steps, 'deal')[0]!;
    expect(deal).toEqual({
      kind: 'deal',
      age: 2,
      at: 0,
      dur: DEAL_LEAD_MS + 19 * DEAL_STAGGER_MS + DEAL_EACH_MS,
      fade: DEAL_FADE_MS,
      lead: DEAL_LEAD_MS,
      each: DEAL_EACH_MS,
      stagger: DEAL_STAGGER_MS
    });
    // About 20 cards at 30 to 40 ms each, and input back with the take.
    expect(DEAL_STAGGER_MS).toBeGreaterThanOrEqual(30);
    expect(DEAL_STAGGER_MS).toBeLessThanOrEqual(40);
    expect(t.blockingMs).toBe(HOLD_MS);
  });

  it('the first age, after the draft, deals with no take and holds no input', () => {
    const t = planTimeline(
      frame(11, [ev('ageStart', { age: 1 })], view([7, 7], { structure: structure(20) })),
      before
    );
    expect(t.steps.map((s) => s.kind)).toEqual(['deal']);
    expect(t.blockingMs).toBe(0);
  });

  it('an age that opens on the starter choice deals as the choice is answered, with no fade', () => {
    const opened = planTimeline(
      frame(
        11,
        [ev('discard', { seat: 0, card: 'theater', amount: 2 }), ev('ageStart', { age: 2 })],
        view([9, 7], { structure: structure(20), phase: 'chooseStarter', age: 2 })
      ),
      before
    );
    // The dialog covers the board: the discard's take and coins, no deal.
    expect(opened.steps.map((s) => s.kind)).toEqual(['take', 'coins']);
    const answered = planTimeline(
      frame(
        12,
        [ev('starterChosen', { seat: 1, starter: 0 })],
        view([9, 7], { structure: structure(20), phase: 'play', age: 2 })
      ),
      frame(11, [], view([9, 7], { structure: structure(20), phase: 'chooseStarter', age: 2 }))
    );
    const deal = only(answered.steps, 'deal')[0]!;
    expect(deal).toMatchObject({ kind: 'deal', age: 2, at: 0, fade: 0, lead: DEAL_LEAD_MS });
    expect(answered.blockingMs).toBe(0);
  });
});

describe('the draft', () => {
  it('a pick and the round’s last wonder fly one after the other, the tick as the first lands', () => {
    const t = planTimeline(
      frame(11, [
        ev('draft', { seat: 1, wonder: 'pyramids' }),
        ev('draft', { seat: 0, wonder: 'colossus', auto: true })
      ]),
      before
    );
    expect(t.steps).toEqual([
      { kind: 'draft', seat: 1, wonder: 'pyramids', at: 0, dur: DRAFT_MS },
      { kind: 'draft', seat: 0, wonder: 'colossus', at: DRAFT_STAGGER_MS, dur: DRAFT_MS }
    ]);
    expect(cueOffsets(t).tick).toBe(DRAFT_MS);
    expect(hapticOffset(t, HAPTIC_TAKE)).toBe(DRAFT_MS);
    expect(t.blockingMs).toBe(HOLD_MS);
  });
});

describe('victory', () => {
  it('the banner comes after the take, with the fanfare', () => {
    const t = planTimeline(
      frame(
        11,
        [
          ev('discard', { seat: 0, card: 'academy', amount: 4 }),
          ev('gameOver', { victory: 'civilian', winner: 0 })
        ],
        view([13, 0])
      ),
      before
    );
    const banner = only(t.steps, 'victory')[0]!;
    expect(banner).toEqual({
      kind: 'victory',
      victory: 'civilian',
      winner: 0,
      at: TAKE_MS,
      dur: VICTORY_MS
    });
    expect(cueOffsets(t).fanfare).toBe(TAKE_MS);
  });

  it('a military supremacy waits for the pawn to reach the capital', () => {
    const t = planTimeline(
      frame(11, [
        ev('build', { seat: 0, card: 'arsenal' }),
        ev('pawn', { seat: 0, from: 7, to: 9 }),
        ev('gameOver', { victory: 'military', winner: 0 })
      ]),
      before
    );
    expect(only(t.steps, 'victory')[0]).toMatchObject({ at: TAKE_MS + 2 * PAWN_STEP_MS });
  });

  it('a shared victory has no winner', () => {
    const t = planTimeline(frame(11, [ev('gameOver', { victory: 'civilian' })]), before);
    expect(t.steps).toEqual([{ kind: 'victory', victory: 'civilian', at: 0, dur: VICTORY_MS }]);
  });
});

describe('every lead', () => {
  it('holds input at most a take’s flight, never past the budget', () => {
    for (const e of [
      ev('build', { seat: 0, card: 'baths' }),
      ev('rebuild', { seat: 0, card: 'baths' }),
      ev('destroy', { seat: 0, card: 'quarry' }),
      ev('token', { seat: 0, token: 'law' }),
      ev('draft', { seat: 0, wonder: 'pyramids' })
    ]) {
      const t = planTimeline(frame(11, [e]), before);
      expect(t.blockingMs, e.kind).toBeGreaterThan(0);
      expect(t.blockingMs, e.kind).toBeLessThanOrEqual(HOLD_MS);
    }
  });

  it('plans nothing for a frame that replays history', () => {
    expect(planTimeline(frame(11, [ev('gameOver', { winner: 0 })]), null)).toBe(EMPTY_TIMELINE);
  });

  it('a vibration with no step of its own starts at once', () => {
    expect(hapticOffset(EMPTY_TIMELINE, HAPTIC_TAKE)).toBe(0);
    expect(hapticOffset(EMPTY_TIMELINE, HAPTIC_LOOT)).toBe(0);
  });
});
