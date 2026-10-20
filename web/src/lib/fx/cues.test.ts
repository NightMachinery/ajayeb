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
      'slide',
      'wonder'
    ]);
    expect(eventCues(ev('pay', { seat: 0, amount: 3 }))).toEqual(['coin']);
    expect(eventCues(ev('coins', { seat: 1, amount: -2 }))).toEqual(['coin']);
    expect(eventCues(ev('reveal', { slot: 4, card: 'baths' }))).toEqual(['flip']);
    expect(eventCues(ev('pawn', { seat: 0, from: 0, to: 2 }))).toEqual(['drum']);
    expect(eventCues(ev('loot', { seat: 1, amount: 2 }))).toEqual(['loot']);
    expect(eventCues(ev('pair', { seat: 0, symbol: 'quill' }))).toEqual(['chime']);
    expect(eventCues(ev('token', { seat: 0, token: 'law' }))).toEqual(['token']);
    expect(eventCues(ev('gameOver', { victory: 'civilian', winner: 0 }))).toEqual(['chime']);
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
    for (const kind of ['effectSkipped', 'extraTurnLost', 'wonderRemoved', 'resign'] as const) {
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
      { cue: 'drum', delayMs: CUE_SPACING_MS },
      { cue: 'loot', delayMs: 2 * CUE_SPACING_MS }
    ]);
    expect(plan.haptic).toEqual(HAPTIC_LOOT);
  });

  it('plays the fanfare at the end of a game', () => {
    const plan = planCues(
      {
        ...frame(11, [ev('build', { card: 'walls' }), ev('pawn'), ev('gameOver', { winner: 0 })]),
        room: { mode: 'hotseat' }
      },
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

function online(version: number, active: number, events: GameEvent[] = [], viewer = 0): CueFrame {
  return {
    ...frame(version, events),
    room: { mode: 'online' },
    you: { seats: [viewer] },
    view: { viewer, active, toMove: active, phase: 'play', playingAgain: false }
  };
}

const names = (next: CueFrame, prev: CueFrame | null = before) =>
  planCues(next, prev).cues.map((c) => c.cue);

describe('distinct event cues', () => {
  it.each([
    ['wonder', ['slide', 'wonder']],
    ['token', ['token']],
    ['loot', ['loot']],
    ['ageStart', ['ageStart']],
    ['destroy', ['destroy']],
    ['rebuild', ['rebuild']],
    ['libraryDraw', ['libraryDraw']],
    ['extraTurn', ['extraTurn']],
    ['draftRound', ['flip']]
  ] as const)('%s has its own cue', (kind, expected) => {
    expect(eventCues(ev(kind))).toEqual(expected);
    expect(names(frame(11, [ev(kind)]))).toEqual(expected);
  });

  it('deduplicates three library draws without reading their hidden tokens', () => {
    const events = [0, 1, 2].map((place) => ev('libraryDraw', { seat: 1, place }));
    expect(names(frame(11, events))).toEqual(['libraryDraw']);
  });

  it('drops routine cues deterministically and keeps three milestones in event order', () => {
    const events = [
      ev('wonder'),
      ev('pay', { amount: 3 }),
      ev('libraryDraw'),
      ev('libraryDraw'),
      ev('extraTurn'),
      ev('reveal')
    ];
    const plan = planCues(frame(11, events), before);
    expect(plan.cues).toEqual([
      { cue: 'wonder', delayMs: 0 },
      { cue: 'libraryDraw', delayMs: 90 },
      { cue: 'extraTurn', delayMs: 180 }
    ]);
    expect(planCues(frame(11, events), before)).toEqual(plan);
    // A pair, a payment and a token must not make four overlapping sounds.
    expect(
      names(frame(11, [ev('build'), ev('coins', { amount: 2 }), ev('pair'), ev('token')]))
    ).toEqual(['slide', 'chime', 'token']);
  });

  it('sounds the new age as it opens on the starter choice, not again for its answer', () => {
    const pending = {
      ...online(11, 1, [ev('ageStart')]),
      view: { ...online(11, 1).view, phase: 'chooseStarter', toMove: 0 }
    };
    expect(names(pending)).toEqual(['ageStart']);
    expect(names(online(12, 0, [ev('starterChosen')]), pending)).toEqual(['tick']);
    expect(names(frame(11, [ev('ageStart'), ev('draftRound'), ev('starterChosen')]))).toEqual([
      'ageStart'
    ]);
  });
});

describe('viewer-aware game endings', () => {
  it.each(['online', 'ai'])('%s plays defeat only for its losing player', (mode) => {
    for (const viewer of [0, 1]) {
      for (const winner of [0, 1]) {
        const next = { ...online(11, 0, [ev('gameOver', { winner })], viewer), room: { mode } };
        expect(names(next)).toEqual([viewer === winner ? 'fanfare' : 'defeat']);
      }
    }
  });

  it('hot-seat celebrates either winner, while shared endings and spectators are neutral', () => {
    for (const winner of [0, 1]) {
      const next = online(11, 0, [ev('gameOver', { winner })]);
      expect(names({ ...next, room: { mode: 'hotseat' } })).toEqual(['fanfare']);
      expect(names({ ...next, you: { seats: [] } })).toEqual(['chime']);
      expect(names({ ...next, view: { active: 0 } })).toEqual(['chime']);
    }
    expect(names(online(11, 0, [ev('gameOver', { victory: 'civilian' })]))).toEqual(['chime']);
    const busy = online(11, 0, [ev('wonder'), ev('libraryDraw'), ev('extraTurn'), ev('gameOver')]);
    expect(names(busy)).toContain('chime');
  });

  it('uses the gameOver after a resignation, not a second resignation cue', () => {
    expect(
      names(
        online(11, 0, [
          ev('resign', { seat: 0 }),
          ev('gameOver', { winner: 1, victory: 'forfeit' })
        ])
      )
    ).toEqual(['defeat']);
  });
});

describe('online turn alert', () => {
  const rival = online(10, 1);
  const handoff = online(11, 0, [ev('build', { seat: 1, card: 'baths' })]);

  it('alerts either seat after a rival completed their turn', () => {
    expect(names(handoff, rival)).toEqual(['slide', 'yourTurn']);
    expect(names(online(11, 1, [ev('discard', { seat: 0 })], 1), online(10, 0, [], 1))).toEqual([
      'slide',
      'yourTurn'
    ]);
  });

  it('alerts after the rival finishes a pending token choice, not when that choice opens', () => {
    const pending = {
      ...online(11, 1, [ev('pair', { seat: 1 })]),
      view: { ...rival.view, phase: 'pickBoardToken' }
    };
    expect(names(pending, rival)).not.toContain('yourTurn');
    expect(names(online(12, 0, [ev('token', { seat: 1 })]), pending)).toEqual([
      'token',
      'yourTurn'
    ]);
    expect(
      names({ ...handoff, view: { ...handoff.view, phase: 'pickBoardToken' } }, rival)
    ).not.toContain('yourTurn');
  });

  it('excludes extra turns, hot-seat, AI, spectators, seat changes and stale snapshots', () => {
    expect(
      names({ ...handoff, view: { ...handoff.view, playingAgain: true } }, rival)
    ).not.toContain('yourTurn');
    expect(names(handoff, online(10, 0))).not.toContain('yourTurn');
    for (const mode of ['hotseat', 'ai']) {
      expect(names({ ...handoff, room: { mode } }, { ...rival, room: { mode } })).not.toContain(
        'yourTurn'
      );
    }
    expect(names({ ...handoff, you: { seats: [] } }, rival)).not.toContain('yourTurn');
    expect(names(handoff, { ...rival, view: { ...rival.view, viewer: 1 } })).not.toContain(
      'yourTurn'
    );
    expect(names(handoff, null)).toEqual([]);
    expect(names({ ...handoff, code: 'OTHER' }, rival)).toEqual([]);
    expect(names({ ...handoff, version: 9 }, rival)).toEqual([]);
    expect(names({ ...handoff, version: 10 }, rival)).toEqual([]);
    expect(names({ ...handoff, history: [] }, rival)).toEqual([]);
    expect(names({ ...handoff, events: [] }, rival)).toEqual([]);
  });

  it('does not mistake a starter chooser or draft transition for a rival turn', () => {
    for (const phase of ['chooseStarter', 'wonderDraft']) {
      expect(names(handoff, { ...rival, view: { ...rival.view, phase } })).not.toContain(
        'yourTurn'
      );
    }
    for (const kind of ['starterChosen', 'ageStart'] as const) {
      expect(names({ ...handoff, events: [ev(kind)] }, rival)).not.toContain('yourTurn');
    }
  });
});
