import { describe, expect, it } from 'vitest';
import type { GameEvent } from '$lib/wire';
import { CUE_SPACING_MS, planCues } from '../cues';
import { EMPTY_TIMELINE, FLIP_DELAY_MS, planTimeline, TAKE_MS, type PlanFrame } from './plan';
import { retimeCues, scaleTimeline, timelineFor } from './speed';

function ev(kind: GameEvent['kind'], fields: Partial<GameEvent> = {}): GameEvent {
  return { kind, private: false, ...fields };
}

const before: PlanFrame = {
  code: 'ROOM',
  version: 10,
  events: [],
  view: { players: [{ coins: 7 }, { coins: 7 }] }
};
const move: PlanFrame = {
  code: 'ROOM',
  version: 11,
  events: [
    ev('build', { seat: 0, card: 'garrison' }),
    ev('pay', { seat: 0, amount: 2 }),
    ev('pawn', { seat: 0, from: 0, to: 1 }),
    ev('reveal', { slot: 2, card: 'quarry' })
  ],
  view: { players: [{ coins: 5 }, { coins: 7 }] }
};

describe('scaleTimeline and timelineFor', () => {
  it('Off gives the empty timeline: nothing animates, nothing holds input', () => {
    expect(timelineFor(move, before, 'off')).toBe(EMPTY_TIMELINE);
    expect(scaleTimeline(planTimeline(move, before), 'off')).toBe(EMPTY_TIMELINE);
  });

  it('normal keeps the planned times', () => {
    expect(timelineFor(move, before, 'normal')).toEqual(planTimeline(move, before));
  });

  it('fast and relaxed scale every start, length and the hold', () => {
    const base = planTimeline(move, before);
    for (const [speed, factor] of [
      ['fast', 0.6],
      ['relaxed', 1.6]
    ] as const) {
      const t = timelineFor(move, before, speed);
      expect(t.blockingMs).toBe(Math.round(base.blockingMs * factor));
      expect(t.totalMs).toBe(Math.round(base.totalMs * factor));
      t.steps.forEach((s, i) => {
        const b = base.steps[i];
        expect(s.at, `${speed} ${s.kind} at`).toBe(Math.round((b?.at ?? 0) * factor));
        expect(s.dur, `${speed} ${s.kind} dur`).toBe(Math.round((b?.dur ?? 0) * factor));
      });
    }
  });

  it('a frame that replays history (cuePrev gives null) plans nothing at any speed', () => {
    expect(timelineFor(move, null, 'normal')).toBe(EMPTY_TIMELINE);
  });
});

describe('retimeCues', () => {
  it('moves each cue to its step: slide on landing, coin with the float, drum, flip', () => {
    const cues = planCues(move, before).cues;
    expect(cues.map((c) => c.cue)).toEqual(['slide', 'coin', 'drum', 'flip']);
    const timed = retimeCues(cues, timelineFor(move, before, 'normal'));
    expect(timed).toEqual([
      { cue: 'slide', delayMs: TAKE_MS },
      { cue: 'coin', delayMs: TAKE_MS },
      { cue: 'drum', delayMs: TAKE_MS },
      { cue: 'flip', delayMs: TAKE_MS + FLIP_DELAY_MS }
    ]);
  });

  it('follows the speed', () => {
    const cues = planCues(move, before).cues;
    const timed = retimeCues(cues, timelineFor(move, before, 'relaxed'));
    expect(timed[0]).toEqual({ cue: 'slide', delayMs: Math.round(TAKE_MS * 1.6) });
  });

  it('keeps a cue with no step of its own after the cue before it', () => {
    const science: PlanFrame = {
      ...before,
      version: 11,
      events: [ev('build', { seat: 0, card: 'scriptorium' })]
    };
    const cues = planCues(science, before).cues;
    expect(cues.map((c) => c.cue)).toEqual(['slide', 'chime']);
    expect(retimeCues(cues, timelineFor(science, before, 'normal'))).toEqual([
      { cue: 'slide', delayMs: TAKE_MS },
      { cue: 'chime', delayMs: TAKE_MS + CUE_SPACING_MS }
    ]);
  });

  it('with Animation off, the cues keep their own spacing', () => {
    const cues = planCues(move, before).cues;
    expect(retimeCues(cues, timelineFor(move, before, 'off'))).toEqual(cues);
  });
});
