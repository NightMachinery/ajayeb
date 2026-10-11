import { describe, expect, it } from 'vitest';
import type { GameEvent } from '$lib/wire';
import {
  CUE_SPACING_MS,
  CUES,
  eventCues,
  eventsHaptic,
  HAPTIC_LOOT,
  HAPTIC_MILITARY,
  HAPTIC_TAKE,
  isLive,
  planCues,
  type CueFrame
} from './cues';

function ev(kind: GameEvent['kind'], fields: Partial<GameEvent> = {}): GameEvent {
  return { kind, private: false, ...fields };
}

function frame(version: number, events: GameEvent[] = [], code = 'ROOM'): CueFrame {
  return { code, version, events };
}

const before = frame(10);

describe('eventCues', () => {
  it('maps each event kind to its cue', () => {
    expect(eventCues(ev('build', { seat: 0, card: 'lumber-yard' }))).toEqual(['slide']);
    expect(eventCues(ev('wonder', { seat: 0, wonder: 'pyramids', card: 'baths' }))).toEqual([
      'slide'
    ]);
    expect(eventCues(ev('pay', { seat: 0, amount: 3 }))).toEqual(['coin']);
    expect(eventCues(ev('coins', { seat: 1, amount: -2 }))).toEqual(['coin']);
    expect(eventCues(ev('reveal', { slot: 4, card: 'baths' }))).toEqual(['flip']);
    expect(eventCues(ev('pawn', { seat: 0, from: 0, to: 2 }))).toEqual(['drum']);
    expect(eventCues(ev('loot', { seat: 1, amount: 2 }))).toEqual(['drum']);
    expect(eventCues(ev('pair', { seat: 0, symbol: 'quill' }))).toEqual(['chime']);
    expect(eventCues(ev('token', { seat: 0, token: 'law' }))).toEqual(['chime']);
    expect(eventCues(ev('gameOver', { victory: 'civilian', winner: 0 }))).toEqual(['fanfare']);
    expect(eventCues(ev('draft', { seat: 0, wonder: 'pyramids' }))).toEqual(['tick']);
  });

  it('adds a chime to a science card, judged by its catalog colour', () => {
    expect(eventCues(ev('build', { seat: 0, card: 'scriptorium' }))).toEqual(['slide', 'chime']);
    expect(eventCues(ev('build', { seat: 0, card: 'not-a-card' }))).toEqual(['slide']);
  });

  it('chinks coins for a discard only when it pays something', () => {
    expect(eventCues(ev('discard', { seat: 0, card: 'baths', amount: 3 }))).toEqual([
      'slide',
      'coin'
    ]);
    expect(eventCues(ev('discard', { seat: 0, card: 'baths', amount: 0 }))).toEqual(['slide']);
    expect(eventCues(ev('pay', { seat: 0, amount: 0 }))).toEqual([]);
  });

  it('stays silent for bookkeeping events', () => {
    for (const kind of ['ageStart', 'draftRound', 'extraTurn', 'effectSkipped'] as const) {
      expect(eventCues(ev(kind)), kind).toEqual([]);
    }
  });

  it('only names known cues', () => {
    const kinds: GameEvent['kind'][] = [
      'build',
      'discard',
      'pay',
      'reveal',
      'pawn',
      'pair',
      'gameOver',
      'draft'
    ];
    for (const kind of kinds) {
      for (const cue of eventCues(ev(kind, { amount: 1 }))) expect(CUES).toContain(cue);
    }
  });
});

describe('isLive', () => {
  it('is false for the first frame of a room', () => {
    expect(isLive(frame(11, [ev('build')]), null)).toBe(false);
  });

  it('is false for another room, an undo or an empty frame', () => {
    expect(isLive(frame(11, [ev('build')], 'OTHER'), before)).toBe(false);
    expect(isLive(frame(9, [ev('build')]), before)).toBe(false);
    expect(isLive(frame(10, [ev('build')]), before)).toBe(false);
    expect(isLive(frame(11), before)).toBe(false);
  });

  it('is true for the next frame of the same room with events', () => {
    expect(isLive(frame(11, [ev('build')]), before)).toBe(true);
  });
});

describe('planCues', () => {
  it('plays nothing for the initial snapshot or history', () => {
    const busy = [ev('build', { card: 'baths' }), ev('pawn'), ev('gameOver')];
    expect(planCues(frame(1, busy), null)).toEqual({ cues: [], haptic: null });
    expect(planCues(frame(5, busy), before)).toEqual({ cues: [], haptic: null });
  });

  it('plays each kind at most once, in first-event order, spaced', () => {
    // A build that pays, pushes the pawn and loots twice.
    const plan = planCues(
      frame(11, [
        ev('build', { seat: 0, card: 'arsenal' }),
        ev('pay', { seat: 0, amount: 3 }),
        ev('pawn', { seat: 0, from: 2, to: 6 }),
        ev('loot', { seat: 1, amount: 2 }),
        ev('coins', { seat: 1, amount: -2 }),
        ev('loot', { seat: 1, amount: 5 }),
        ev('coins', { seat: 1, amount: -5 })
      ]),
      before
    );
    expect(plan.cues).toEqual([
      { cue: 'slide', delayMs: 0 },
      { cue: 'coin', delayMs: CUE_SPACING_MS },
      { cue: 'drum', delayMs: 2 * CUE_SPACING_MS }
    ]);
    expect(plan.haptic).toEqual(HAPTIC_LOOT);
  });

  it('plays the fanfare at the end of a game', () => {
    const plan = planCues(
      frame(11, [ev('build', { card: 'walls' }), ev('pawn'), ev('gameOver', { winner: 0 })]),
      before
    );
    expect(plan.cues.map((c) => c.cue)).toEqual(['slide', 'drum', 'fanfare']);
  });
});

describe('eventsHaptic', () => {
  it('buzzes briefly for a take, more for military, most for looting', () => {
    expect(eventsHaptic([ev('build')])).toEqual(HAPTIC_TAKE);
    expect(eventsHaptic([ev('draft')])).toEqual(HAPTIC_TAKE);
    expect(eventsHaptic([ev('build'), ev('pawn')])).toEqual(HAPTIC_MILITARY);
    expect(eventsHaptic([ev('build'), ev('pawn'), ev('loot')])).toEqual(HAPTIC_LOOT);
    expect(HAPTIC_TAKE).toEqual([8]);
  });

  it('stays still for events that take nothing', () => {
    expect(eventsHaptic([ev('token'), ev('pair')])).toBeNull();
    expect(eventsHaptic([])).toBeNull();
  });
});
