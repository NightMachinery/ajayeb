// Haptics timing (M10.animations-events): a vibration starts with its step
// in the timeline the runner plays for the frame on screen.
import { afterEach, describe, expect, it, vi } from 'vitest';
import { setPlanned } from './anim/records';
import { planTimeline, TAKE_MS, PAWN_STEP_MS, EMPTY_TIMELINE } from './anim/plan';
import { scaleTimeline } from './anim/speed';
import { HAPTIC_LOOT, HAPTIC_MILITARY, HAPTIC_TAKE } from './cues';
import { hapticDelay, vibrate } from './haptics';
import type { GameEvent } from '$lib/wire';

function ev(kind: GameEvent['kind'], fields: Partial<GameEvent> = {}): GameEvent {
  return { kind, private: false, ...fields };
}

const prev = {
  code: 'ROOM',
  version: 10,
  events: [],
  view: {
    players: [{ coins: 9 }, { coins: 9 }],
    military: { looting: [{ at: 3, present: true }] }
  }
};
const next = {
  code: 'ROOM',
  version: 11,
  events: [
    ev('build', { seat: 0, card: 'garrison' }),
    ev('pawn', { seat: 0, from: 1, to: 3 }),
    ev('loot', { seat: 1, amount: 2, value: 2 })
  ],
  view: {
    players: [{ coins: 9 }, { coins: 7 }],
    military: { looting: [{ at: 3, present: false }] }
  }
};

afterEach(() => {
  vi.useRealTimers();
  vi.unstubAllGlobals();
  setPlanned(0, EMPTY_TIMELINE);
});

describe('haptics timing', () => {
  it('times each pattern by its step, at the speed the frame plays', () => {
    setPlanned(11, planTimeline(next, prev));
    expect(hapticDelay(HAPTIC_TAKE)).toBe(TAKE_MS);
    expect(hapticDelay(HAPTIC_MILITARY)).toBe(TAKE_MS);
    expect(hapticDelay(HAPTIC_LOOT)).toBe(TAKE_MS + 2 * PAWN_STEP_MS);
    setPlanned(11, scaleTimeline(planTimeline(next, prev), 'relaxed'));
    expect(hapticDelay(HAPTIC_LOOT)).toBe(Math.round((TAKE_MS + 2 * PAWN_STEP_MS) * 1.6));
  });

  it('with nothing animating, it vibrates at once', () => {
    setPlanned(11, EMPTY_TIMELINE);
    expect(hapticDelay(HAPTIC_LOOT)).toBe(0);
  });

  it('vibrates when the step comes, and a newer vibration replaces a waiting one', () => {
    vi.useFakeTimers();
    const calls: number[][] = [];
    vi.stubGlobal('navigator', { vibrate: (p: number[]) => calls.push(p) });
    setPlanned(11, planTimeline(next, prev));
    vibrate(HAPTIC_LOOT);
    expect(calls).toEqual([]);
    vi.advanceTimersByTime(TAKE_MS + 2 * PAWN_STEP_MS);
    expect(calls).toEqual([[...HAPTIC_LOOT]]);
    vibrate(HAPTIC_TAKE);
    setPlanned(12, EMPTY_TIMELINE);
    vibrate(HAPTIC_MILITARY);
    vi.advanceTimersByTime(1000);
    expect(calls).toEqual([[...HAPTIC_LOOT], [...HAPTIC_MILITARY]]);
  });
});
