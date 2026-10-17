import { afterEach, describe, expect, it, vi } from 'vitest';
import { settings } from '$lib/ui/settings.svelte';
import { playFrameFx, playRecordedCues } from './index';
import { cancelHaptic } from './haptics';
import { setPlanned } from './anim/records';
import { EMPTY_TIMELINE, planTimeline } from './anim/plan';
import { HAPTIC_TAKE } from './cues';

vi.mock('./sfx', () => ({ play: vi.fn(), unlockAudio: vi.fn(), VOLUME_GAIN: {} }));

const prev = { code: 'ROOM', version: 1, events: [] };
const next = {
  code: 'ROOM',
  version: 2,
  events: [{ kind: 'build' as const, private: false, seat: 0, card: 'baths' }]
};

afterEach(() => {
  cancelHaptic();
  settings.update({ sound: false, haptics: true });
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

describe('frame FX and replay recording', () => {
  it('records replay cues with a marker only while Sound is on', () => {
    vi.stubGlobal('window', { addEventListener: vi.fn() });
    settings.update({ sound: true });
    playRecordedCues([{ cue: 'slide', delayMs: 30 }], 2, true);
    const hook = window.__ajayebFx!;
    expect(hook.cues().at(-1)).toEqual({ cue: 'slide', delayMs: 30, version: 2, replay: true });
    const count = hook.cues().length;
    settings.update({ sound: false });
    playRecordedCues([{ cue: 'slide', delayMs: 30 }], 2, true);
    expect(hook.cues()).toHaveLength(count);
  });

  it('schedules exactly one step-timed vibration; Animation off vibrates once immediately', () => {
    vi.useFakeTimers();
    const buzz = vi.fn();
    vi.stubGlobal('navigator', { vibrate: buzz });
    setPlanned(2, planTimeline(next, prev));
    playFrameFx(next, prev);
    expect(buzz).not.toHaveBeenCalled();
    vi.runAllTimers();
    expect(buzz).toHaveBeenCalledExactlyOnceWith([...HAPTIC_TAKE]);
    buzz.mockClear();
    setPlanned(2, EMPTY_TIMELINE);
    playFrameFx(next, prev);
    expect(buzz).toHaveBeenCalledExactlyOnceWith([...HAPTIC_TAKE]);
    vi.runAllTimers();
    expect(buzz).toHaveBeenCalledTimes(1);
  });
});
