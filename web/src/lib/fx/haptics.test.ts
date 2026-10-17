// Haptics timing (M10.animations-events): a vibration starts with its step
// in the timeline the runner plays for the frame on screen.
import { afterEach, describe, expect, it, vi } from 'vitest';
import { latestPlanned, setPlanned } from './anim/records';
import { planTimeline, TAKE_MS, PAWN_STEP_MS, EMPTY_TIMELINE } from './anim/plan';
import { scaleTimeline } from './anim/speed';
import { HAPTIC_LOOT, HAPTIC_MILITARY, HAPTIC_TAKE } from './cues';
import { cancelHaptic, hapticDelay, scheduleHaptic } from './haptics';
import { settings } from '$lib/ui/settings.svelte';
import { AnimRunner } from './anim/runner';
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
  cancelHaptic();
  settings.update({ haptics: true });
  vi.useRealTimers();
  vi.unstubAllGlobals();
  setPlanned(0, EMPTY_TIMELINE);
});

describe('haptics timing', () => {
  it.each(['finish', 'hurry/clear/undo capture'])('cancels a delayed vibration on %s', (way) => {
    vi.useFakeTimers();
    const buzz = vi.fn();
    vi.stubGlobal('navigator', { vibrate: buzz });
    setPlanned(11, planTimeline(next, prev));
    scheduleHaptic(HAPTIC_LOOT, latestPlanned(), () => {});
    const runner = new AnimRunner({ speed: () => 'normal', root: () => null });
    if (way === 'finish') runner.finish();
    else runner.capture(next, null);
    vi.runAllTimers();
    expect(buzz).not.toHaveBeenCalled();
  });

  it.each(['switch off then on', 'timeout recheck'])(
    'does not play a disabled vibration: %s',
    (way) => {
      vi.useFakeTimers();
      const buzz = vi.fn();
      const record = vi.fn();
      vi.stubGlobal('navigator', { vibrate: buzz });
      scheduleHaptic(HAPTIC_LOOT, planTimeline(next, prev), record);
      if (way === 'switch off then on') {
        settings.update({ haptics: false });
        settings.update({ haptics: true });
      } else settings.value.haptics = false;
      vi.runAllTimers();
      expect(buzz).not.toHaveBeenCalled();
      expect(record).not.toHaveBeenCalled();
    }
  );

  it('times each pattern by its step, at the speed the frame plays', () => {
    setPlanned(11, planTimeline(next, prev));
    expect(hapticDelay(HAPTIC_TAKE, latestPlanned())).toBe(TAKE_MS);
    expect(hapticDelay(HAPTIC_MILITARY, latestPlanned())).toBe(TAKE_MS);
    expect(hapticDelay(HAPTIC_LOOT, latestPlanned())).toBe(TAKE_MS + 2 * PAWN_STEP_MS);
    setPlanned(11, scaleTimeline(planTimeline(next, prev), 'relaxed'));
    expect(hapticDelay(HAPTIC_LOOT, latestPlanned())).toBe(
      Math.round((TAKE_MS + 2 * PAWN_STEP_MS) * 1.6)
    );
  });

  it('with nothing animating, it vibrates at once', () => {
    setPlanned(11, EMPTY_TIMELINE);
    expect(hapticDelay(HAPTIC_LOOT, latestPlanned())).toBe(0);
  });

  it('vibrates when the step comes, and a newer vibration replaces a waiting one', () => {
    vi.useFakeTimers();
    const calls: number[][] = [];
    vi.stubGlobal('navigator', { vibrate: (p: number[]) => calls.push(p) });
    setPlanned(11, planTimeline(next, prev));
    scheduleHaptic(HAPTIC_LOOT, latestPlanned(), () => {});
    expect(calls).toEqual([]);
    vi.advanceTimersByTime(TAKE_MS + 2 * PAWN_STEP_MS);
    expect(calls).toEqual([[...HAPTIC_LOOT]]);
    scheduleHaptic(HAPTIC_TAKE, latestPlanned(), () => {});
    setPlanned(12, EMPTY_TIMELINE);
    scheduleHaptic(HAPTIC_MILITARY, latestPlanned(), () => {});
    vi.advanceTimersByTime(1000);
    expect(calls).toEqual([[...HAPTIC_LOOT], [...HAPTIC_MILITARY]]);
  });
});
