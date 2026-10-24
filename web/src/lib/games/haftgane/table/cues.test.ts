// The Haftgane cue mapper: every event kind, the no-replay rule, the cue
// budget on the engine's own seven-seat batches
// (server/internal/haftgane/rules/testdata/views), and the decisions that
// notify one viewer alone.
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import type { Table } from '$lib/wire/haftgane.gen';
import type { TableEvent, TableState } from '$lib/wire/platform.gen';
import { tableFrame } from '../session/test-frames';
import {
  CUE_BUDGET,
  CUE_SPACING_MS,
  HAPTIC_CONFLICT,
  HAPTIC_REVEAL,
  isLive,
  planTableCues,
  type CueTable
} from './cues';

const blank: TableEvent = {
  kind: '',
  seat: 0,
  other: 0,
  age: 0,
  turn: 0,
  play: '',
  design: '',
  stage: 0,
  side: '',
  left: 0,
  right: 0,
  printed: 0,
  delta: 0,
  source: '',
  outcome: '',
  pass: ''
};
const ev = (kind: string, fields: Partial<TableEvent> = {}): TableEvent => ({
  ...blank,
  kind,
  ...fields
});

/** Seat 0's view of a three-seat table (seats 0 and 1 human, 2 a bot); null is a spectator. */
function frame(overrides: Partial<TableState> = {}, seat: number | null = 0): TableState {
  return tableFrame({ revision: 10, ...overrides }, seat);
}

/** The cue names `events` make for `seat` after a quiet frame. */
function names(events: TableEvent[], seat: number | null = 0, extra: Partial<TableState> = {}) {
  const prev = frame({}, seat);
  const next = frame({ revision: 11, batch: 1, events, ...extra }, seat);
  return planTableCues(next, prev).cues.map((c) => c.cue);
}

function haptic(events: TableEvent[], seat: number | null = 0) {
  return planTableCues(frame({ revision: 11, events }, seat), frame({}, seat)).haptic;
}

const point = (kind: string, id: string, required: number[], confirmed: number[] = []) => ({
  ref: { id, revision: 1 },
  kind,
  required,
  confirmed
});

describe('every event kind', () => {
  it('age-start sounds the new Age for everyone', () => {
    expect(names([ev('age-start', { age: 2 })])).toEqual(['ageStart']);
    expect(names([ev('age-start', { age: 2 })], null)).toEqual(['ageStart']);
  });

  it('hands-passed slides the new hand in for a seat, not for a spectator', () => {
    expect(names([ev('hands-passed', { pass: 'left' })])).toEqual(['slide']);
    expect(names([ev('hands-passed', { pass: 'left' })], null)).toEqual([]);
  });

  it('side-chosen is the reveal, once for all seats', () => {
    const sides = [0, 1, 2].map((seat) => ev('side-chosen', { seat, side: 'day' }));
    expect(names(sides)).toEqual(['flip']);
    expect(names(sides, null)).toEqual(['flip']);
  });

  it('military is one drum for every conflict of the batch', () => {
    const wars = [
      ev('military', { seat: 0, other: 1, outcome: 'seat-wins' }),
      ev('military', { seat: 1, other: 2, outcome: 'tie' }),
      ev('military', { seat: 2, other: 0, outcome: 'other-wins' })
    ];
    expect(names(wars)).toEqual(['drum']);
    expect(names(wars, null)).toEqual(['drum']);
  });

  it('game-over is a fanfare for a winner, a defeat for another seat, a chime for a spectator', () => {
    const over = [ev('game-over')];
    const result = (winners: number[]) => ({
      view: { ...frame().view, result: { winners } } as TableState['view']
    });
    expect(names(over, 0, result([0, 2]))).toEqual(['fanfare']);
    expect(names(over, 1, result([0, 2]))).toEqual(['defeat']);
    expect(names(over, null, result([0, 2]))).toEqual(['chime']);
  });

  it('coins sound only for the viewer who gains them', () => {
    expect(names([ev('coins', { seat: 0, delta: 3, source: 'sale' })])).toEqual(['coin']);
    expect(names([ev('coins', { seat: 1, delta: 3, source: 'sale' })])).toEqual([]);
    expect(names([ev('coins', { seat: 0, delta: 0, source: 'design' })])).toEqual([]);
    expect(names([ev('coins', { seat: 0, delta: 3 })], null)).toEqual([]);
  });

  it('play is the reveal, plus what the viewer did', () => {
    const others = [ev('play', { seat: 1, play: 'build' }), ev('play', { seat: 2, play: 'sell' })];
    expect(names(others)).toEqual(['flip']);
    expect(names([ev('play', { seat: 0, play: 'build' }), ...others])).toEqual(['flip', 'slide']);
    expect(names([ev('play', { seat: 0, play: 'stage', stage: 1 }), ...others])).toEqual([
      'flip',
      'wonder'
    ]);
    expect(names([ev('play', { seat: 0, play: 'sell' }), ...others])).toEqual(['flip', 'coin']);
    expect(names([ev('play', { seat: 0, play: 'build' }), ...others], null)).toEqual(['flip']);
  });

  it('a neighbour paying the viewer for a trade is a coin', () => {
    // Seat 2's left neighbour is seat 0; seat 1's right neighbour is seat 0.
    expect(names([ev('play', { seat: 2, play: 'build', left: 2 })])).toEqual(['flip', 'coin']);
    expect(names([ev('play', { seat: 1, play: 'build', right: 1 })])).toEqual(['flip', 'coin']);
    expect(names([ev('play', { seat: 1, play: 'build', left: 2 })])).toEqual(['flip']);
    // The viewer paying is not a gain.
    expect(names([ev('play', { seat: 0, play: 'stage', left: 2 })])).toEqual(['flip', 'wonder']);
  });

  it('leftover-discarded adds nothing to the reveal it ends', () => {
    expect(names([ev('leftover-discarded', { seat: 0 })])).toEqual([]);
    expect(
      names([ev('play', { seat: 1, play: 'build' }), ev('leftover-discarded', { seat: 0 })])
    ).toEqual(['flip']);
  });

  it('card-kept is an extra turn for the viewer who kept it', () => {
    expect(names([ev('card-kept', { seat: 0 })])).toEqual(['extraTurn']);
    expect(names([ev('card-kept', { seat: 1 })])).toEqual([]);
    expect(names([ev('card-kept', { seat: 0 })], null)).toEqual([]);
  });

  it('discard-build rebuilds, discard-build-nothing ticks', () => {
    expect(names([ev('discard-build', { seat: 1, design: 'amber' })])).toEqual(['rebuild']);
    expect(names([ev('discard-build', { seat: 1, design: 'amber' })], null)).toEqual(['rebuild']);
    expect(names([ev('discard-build-nothing', { seat: 1 })])).toEqual(['tick']);
  });
});

describe('no replay', () => {
  const events = [ev('play', { seat: 1, play: 'build' }), ev('military', { seat: 0, other: 1 })];
  const next = frame({ revision: 11, events });

  it('a live frame plays', () => {
    expect(isLive(next, frame())).toBe(true);
    expect(planTableCues(next, frame()).cues).not.toEqual([]);
  });

  it('a frame with history, or the first frame, plays nothing', () => {
    expect(planTableCues({ ...next, history: [{ batch: 1, events }] }, frame())).toEqual({
      cues: [],
      haptic: null
    });
    expect(planTableCues(next, null)).toEqual({ cues: [], haptic: null });
  });

  it('another room, another seat or no new revision plays nothing', () => {
    expect(planTableCues(next, frame({ code: 'TBLBBB' })).cues).toEqual([]);
    expect(planTableCues(next, frame({}, 1)).cues).toEqual([]);
    expect(planTableCues(next, frame({}, null)).cues).toEqual([]);
    expect(planTableCues(next, frame({ revision: 11 })).cues).toEqual([]);
    expect(planTableCues(next, frame({ revision: 12 })).cues).toEqual([]);
  });
});

describe('the budget', () => {
  function fixture(name: string): Table {
    const url = new URL(
      `../../../../../../server/internal/haftgane/rules/testdata/views/${name}.json`,
      import.meta.url
    );
    return JSON.parse(readFileSync(url, 'utf8')) as Table;
  }

  /** A fixture as its viewer's live frame, after a turn point every seat had confirmed. */
  function live(name: string): { next: CueTable; prev: CueTable } {
    const t = fixture(name);
    const seats = t.seats.map((_, seat) => ({ seat, kind: seat === 0 ? 'human' : 'bot' }));
    const all = t.seats.map((_, seat) => seat);
    const you = t.you ? { seat: t.you.seat } : {};
    const view = { seats: t.seats, result: t.result };
    const next: CueTable = {
      code: 'T',
      revision: 2,
      events: t.events.map((e) => ({ ...e, pass: '' })),
      you,
      seats,
      point: t.point ? point(t.kind, t.point, t.required) : undefined,
      view
    };
    const prev: CueTable = {
      ...next,
      revision: 1,
      events: [],
      point: point('turn', 'before', all, all)
    };
    return { next, prev };
  }

  it('seven seats revealing at once make one flip, and the viewer at most two more', () => {
    for (const name of [
      'discard-seat',
      'last-seat',
      'over-hidden-seat',
      'over-reveal-spectator',
      'turn-seat',
      'turn-spectator'
    ]) {
      const { next, prev } = live(name);
      const plan = planTableCues(next, prev);
      expect(plan.cues.length, name).toBeLessThanOrEqual(CUE_BUDGET);
      expect(plan.cues.map((c) => c.delayMs)).toEqual(plan.cues.map((_, i) => i * CUE_SPACING_MS));
      expect(new Set(plan.cues.map((c) => c.cue)).size).toBe(plan.cues.length);
    }
  });

  it('the batches sound as the table reads them', () => {
    const cues = (name: string) => {
      const { next, prev } = live(name);
      return planTableCues(next, prev).cues.map((c) => c.cue);
    };
    // Seat 2 staged a wonder, and the discard pile is theirs to build from.
    expect(cues('discard-seat')).toEqual(['flip', 'wonder', 'yourTurn']);
    // Seat 2 built from the pile; seat 4 now decides its kept last card.
    expect(cues('last-seat')).toEqual(['rebuild', 'yourTurn']);
    // The last batch: the reveal, the conflicts, and seat 0 did not win.
    expect(cues('over-hidden-seat')).toEqual(['flip', 'drum', 'defeat']);
    expect(cues('over-reveal-spectator')).toEqual(['flip', 'drum', 'chime']);
    // The sides, then Age I's deal.
    expect(cues('turn-seat')).toEqual(['flip', 'ageStart']);
    expect(cues('turn-spectator')).toEqual(['flip', 'ageStart']);
  });

  it('a busy frame keeps the shared moments first, in their playing order', () => {
    const events = [
      ev('play', { seat: 0, play: 'sell' }),
      ev('play', { seat: 1, play: 'build', right: 2 }),
      ev('coins', { seat: 0, delta: 3 }),
      ev('military', { seat: 0, other: 1 }),
      ev('age-start', { age: 2 }),
      ev('card-kept', { seat: 0 })
    ];
    expect(names(events)).toEqual(['flip', 'drum', 'ageStart']);
  });
});

describe('decisions of one viewer', () => {
  it('a special decision opening for the viewer is their turn, not for anyone else', () => {
    const open = (seat: number | null) =>
      planTableCues(
        frame(
          {
            revision: 11,
            events: [ev('play', { seat: 1, play: 'stage' })],
            point: point('discard-build', 'd6', [1])
          },
          seat
        ),
        frame({ point: point('turn', 't6', [0, 1, 2], [0, 1, 2]) }, seat)
      ).cues.map((c) => c.cue);
    expect(open(1)).toEqual(['flip', 'wonder', 'yourTurn']);
    expect(open(0)).toEqual(['flip']);
    expect(open(null)).toEqual(['flip']);
  });

  it('the last card kept is the extra turn alone', () => {
    const prev = frame({ point: point('turn', 't6', [0, 1, 2], [0, 1, 2]) });
    const next = frame({
      revision: 11,
      events: [ev('card-kept', { seat: 0 })],
      point: point('last-card', 'l', [0])
    });
    expect(planTableCues(next, prev).cues.map((c) => c.cue)).toEqual(['extraTurn']);
  });

  it('the table waiting on the viewer alone is their turn, when someone waiting is a person', () => {
    const at = (confirmed: number[], revision: number) =>
      frame({ revision, point: point('turn', 't2', [0, 1, 2], confirmed) });
    // Seat 1 (a person) confirms last but the viewer: their turn.
    expect(planTableCues(at([1, 2], 12), at([2], 11)).cues.map((c) => c.cue)).toEqual(['yourTurn']);
    // Already the last before this frame, or not the last: nothing. Having
    // confirmed is only the confirmation's tick.
    expect(planTableCues(at([1, 2], 12), at([1, 2], 11)).cues).toEqual([]);
    expect(planTableCues(at([2], 12), at([], 11)).cues).toEqual([]);
    expect(planTableCues(at([0, 1, 2], 12), at([1, 2], 11)).cues.map((c) => c.cue)).toEqual([
      'tick'
    ]);
    // Only bots wait with the viewer: a table of bots always waits on them.
    const bots = (confirmed: number[], revision: number) =>
      frame({
        revision,
        seats: [
          { seat: 0, kind: 'human', userId: 'u0' },
          { seat: 1, kind: 'bot', bot: 'easy' },
          { seat: 2, kind: 'bot', bot: 'easy' }
        ],
        point: point('turn', 't2', [0, 1, 2], confirmed)
      });
    expect(planTableCues(bots([1, 2], 12), bots([2], 11)).cues).toEqual([]);
  });
});

describe('the viewer confirming', () => {
  const at = (confirmed: number[], revision: number, id = 't2') =>
    frame({ revision, point: point('turn', id, [0, 1, 2], confirmed) });

  it('ticks once the table shows their confirmation, with a tap', () => {
    const plan = planTableCues(at([0], 12), at([], 11));
    expect(plan.cues.map((c) => c.cue)).toEqual(['tick']);
    expect(plan.haptic).toEqual(HAPTIC_REVEAL);
  });

  it('stays quiet for a withdrawal, another seat, a spectator or a new point', () => {
    expect(planTableCues(at([], 12), at([0], 11)).cues).toEqual([]);
    expect(planTableCues(at([2], 12), at([], 11)).cues).toEqual([]);
    const watch = (confirmed: number[], revision: number) =>
      frame({ revision, point: point('turn', 't2', [0, 1, 2], confirmed) }, null);
    expect(planTableCues(watch([0], 12), watch([], 11)).cues).toEqual([]);
    expect(planTableCues(at([0], 12, 't3'), at([], 11)).cues).toEqual([]);
  });
});

describe('vibration', () => {
  it('a conflict of the viewer is the military pattern', () => {
    expect(haptic([ev('military', { seat: 2, other: 0 })])).toEqual(HAPTIC_CONFLICT);
    expect(haptic([ev('military', { seat: 0, other: 1 })])).toEqual(HAPTIC_CONFLICT);
  });

  it('a reveal or a decision of a seated viewer is a tap; a spectator never vibrates', () => {
    expect(haptic([ev('play', { seat: 1, play: 'build' })])).toEqual(HAPTIC_REVEAL);
    expect(haptic([ev('discard-build-nothing', { seat: 1 })])).toEqual(HAPTIC_REVEAL);
    expect(haptic([ev('military', { seat: 1, other: 2 })])).toBeNull();
    expect(haptic([ev('age-start')])).toBeNull();
    expect(haptic([ev('play', { seat: 1, play: 'build' })], null)).toBeNull();
    expect(haptic([ev('military', { seat: 0, other: 1 })], null)).toBeNull();
  });
});
