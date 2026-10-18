import { afterEach, describe, expect, it, vi } from 'vitest';
import { settings } from '$lib/ui/settings.svelte';
import { playFrameFx, playRecordedCues } from './index';
import { cancelHaptic } from './haptics';
import { setPlanned } from './anim/records';
import { EMPTY_TIMELINE, planTimeline, type PlanFrame } from './anim/plan';
import { DealCarry } from './anim/backlog';
import { AnimRunner } from './anim/runner';
import { retimeCues, scaleTimeline } from './anim/speed';
import { HAPTIC_TAKE, planCues } from './cues';
import { play } from './sfx';

vi.mock('./sfx', () => ({ play: vi.fn(), unlockAudio: vi.fn(), VOLUME_GAIN: {} }));

const prev = { code: 'ROOM', version: 1, events: [] };
const next = {
  code: 'ROOM',
  version: 2,
  events: [{ kind: 'build' as const, private: false, seat: 0, card: 'baths' }]
};

afterEach(() => {
  cancelHaptic();
  settings.update({ sound: false, haptics: true, animationSpeed: 'normal' });
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
    playRecordedCues([{ cue: 'token', delayMs: 120 }], 2, true);
    expect(hook.cues().at(-1)).toEqual({ cue: 'token', delayMs: 120, version: 2, replay: true });
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

describe('frame metadata and audio gating', () => {
  it('passes existing room, seat and view metadata into personal cues and replay records', () => {
    vi.useFakeTimers();
    settings.update({ sound: true, haptics: false });
    const base = {
      room: { mode: 'online' },
      you: { seats: [0] },
      view: { viewer: 0, active: 1, toMove: 1, phase: 'play', playingAgain: false }
    };
    setPlanned(2, EMPTY_TIMELINE);
    vi.mocked(play).mockClear();
    playFrameFx(
      { ...next, ...base, view: { ...base.view, active: 0, toMove: 0 } },
      { ...prev, ...base }
    );
    vi.advanceTimersByTime(90);
    expect(play).toHaveBeenCalledWith('yourTurn', {
      delayMs: 0,
      volume: settings.value.soundVolume
    });
    vi.mocked(play).mockClear();
    playFrameFx(
      { ...next, ...base, events: [{ kind: 'gameOver', private: false, winner: 1 }] },
      { ...prev, ...base }
    );
    expect(play).toHaveBeenCalledExactlyOnceWith('defeat', {
      delayMs: 0,
      volume: settings.value.soundVolume
    });
    playRecordedCues([{ cue: 'defeat', delayMs: 0 }], 2, true);
    // Recording is shared for live frames and replays, including new cue names.
    expect(play).toHaveBeenCalledTimes(2);
  });

  it('does not call the player with Sound off or an initial/history frame', () => {
    vi.mocked(play).mockClear();
    settings.update({ sound: false });
    playFrameFx(next, prev);
    settings.update({ sound: true });
    playFrameFx(next, null);
    playFrameFx({ ...next, history: [] }, prev);
    expect(play).not.toHaveBeenCalled();
  });
});

describe('actual delayed personal notification delivery', () => {
  function handoff(speed: 'normal' | 'relaxed' = 'normal') {
    vi.useFakeTimers();
    vi.setSystemTime(0);
    settings.update({ sound: true, haptics: false, animationSpeed: speed });
    const base = {
      room: { mode: 'online' },
      you: { seats: [0] },
      view: {
        viewer: 0,
        active: 1,
        toMove: 1,
        phase: 'play',
        age: 1,
        playingAgain: false,
        players: [{ coins: 7 }, { coins: 7 }]
      }
    };
    const previous = { ...prev, ...base };
    const incoming = {
      ...next,
      ...base,
      events: [{ kind: 'discard' as const, private: false, seat: 1, card: 'baths', amount: 2 }],
      view: { ...base.view, active: 0, toMove: 0, players: [{ coins: 7 }, { coins: 9 }] }
    };
    const timeline = scaleTimeline(planTimeline(incoming, previous), speed);
    setPlanned(2, timeline);
    // This stub is the audio delivery boundary, not the intended-cue hook:
    // a future alert must never reach WebAudio's uncancellable start queue.
    const audio: { cue: string; time: number; delay: number }[] = [];
    vi.mocked(play).mockClear();
    vi.mocked(play).mockImplementation((cue, opts) => {
      audio.push({ cue, time: Date.now(), delay: opts?.delayMs ?? 0 });
      return true;
    });
    playFrameFx(incoming, previous);
    return { incoming, timeline, audio, alerts: () => audio.filter((a) => a.cue === 'yourTurn') };
  }

  it.each(['normal', 'relaxed'] as const)(
    'delivers at input readiness, with no future audio start (%s)',
    (speed) => {
      const { timeline, audio, alerts } = handoff(speed);
      expect(timeline.totalMs).toBeGreaterThan(timeline.blockingMs);
      expect(audio.map((a) => a.cue)).toEqual(['slide', 'coin']);
      expect(alerts()).toEqual([]);
      vi.advanceTimersByTime(timeline.blockingMs - 1);
      expect(alerts()).toEqual([]);
      vi.advanceTimersByTime(1);
      expect(alerts()).toEqual([{ cue: 'yourTurn', time: timeline.blockingMs, delay: 0 }]);
    }
  );

  it.each(['new move', 'leave', 'mute'] as const)('invalidates a pending alert on %s', (cause) => {
    const { incoming, alerts } = handoff('relaxed');
    vi.advanceTimersByTime(100);
    if (cause === 'new move') {
      setPlanned(3, EMPTY_TIMELINE);
      playFrameFx(
        {
          ...incoming,
          version: 3,
          events: [{ kind: 'build', private: false, seat: 0, card: 'baths' }],
          view: { ...incoming.view, active: 1, toMove: 1 }
        },
        incoming
      );
    } else if (cause === 'leave') {
      // Presenter.clear/destroy and route exit call the runner's finish hook.
      new AnimRunner({ speed: () => 'normal' }).finish();
    } else {
      settings.set('sound', false);
      settings.set('sound', true); // Re-enabling must not restore an old notification.
    }
    vi.advanceTimersByTime(3000);
    expect(alerts()).toEqual([]);
  });

  it('does not repeat a personal notification in a visual replay', () => {
    const { alerts } = handoff();
    new AnimRunner({ speed: () => 'normal' }).finish();
    playRecordedCues([{ cue: 'yourTurn', delayMs: 90 }], 2, true);
    vi.advanceTimersByTime(3000);
    expect(alerts()).toEqual([]);
  });
});

describe('age cue follows the played deal', () => {
  const before: PlanFrame = {
    code: 'AGE',
    version: 1,
    events: [],
    view: { phase: 'chooseStarter', age: 2, players: [] }
  };
  const answer: PlanFrame = {
    code: 'AGE',
    version: 4,
    events: [{ kind: 'build', private: false, seat: 0, card: 'baths' }],
    view: { phase: 'play', age: 2, players: [] }
  };

  it('a coalesced starter answer sounds its recovered deal in both live and replay plans', () => {
    settings.update({ sound: true, haptics: false, animationSpeed: 'normal' });
    const timeline = planTimeline(answer, before);
    setPlanned(answer.version, timeline);
    vi.mocked(play).mockClear();
    playFrameFx(answer, before);
    expect(vi.mocked(play).mock.calls.filter(([cue]) => cue === 'ageStart')).toEqual([
      ['ageStart', { delayMs: 120, volume: settings.value.soundVolume }]
    ]);
    // The runner stores cues from the same played timeline for its replay.
    const cues = planCues(
      answer,
      before,
      timeline.steps.some((s) => s.kind === 'deal')
    ).cues;
    playRecordedCues(retimeCues(cues, timeline), answer.version, true);
    expect(vi.mocked(play).mock.calls.filter(([cue]) => cue === 'ageStart')).toHaveLength(2);
  });

  it('a queued opening is silent until the carried deal actually plays, exactly once', () => {
    settings.update({ sound: true, haptics: false, animationSpeed: 'normal' });
    const carry = new DealCarry();
    const start = {
      ...before,
      version: 2,
      events: [{ kind: 'starterChosen' as const, private: false }]
    };
    start.view = { ...before.view!, phase: 'play' };
    const queued = carry.plan(start, before, planTimeline(start, before), 1);
    setPlanned(2, queued);
    vi.mocked(play).mockClear();
    playFrameFx(start, before);
    expect(vi.mocked(play).mock.calls.some(([cue]) => cue === 'ageStart')).toBe(false);
    const timeline = carry.plan(answer, start, planTimeline(answer, start), 0);
    setPlanned(answer.version, timeline);
    playFrameFx(answer, start);
    expect(vi.mocked(play).mock.calls.filter(([cue]) => cue === 'ageStart')).toHaveLength(1);
    const cues = planCues(
      answer,
      start,
      timeline.steps.some((s) => s.kind === 'deal')
    ).cues;
    playRecordedCues(retimeCues(cues, timeline), answer.version, true);
    expect(vi.mocked(play).mock.calls.filter(([cue]) => cue === 'ageStart')).toHaveLength(2);
  });

  it('can flush a carried deal on an eventless same-version frame without replaying other cues', () => {
    const eventless = { ...answer, version: before.version, events: [] };
    expect(planCues(eventless, before, true)).toEqual({
      cues: [{ cue: 'ageStart', delayMs: 0 }],
      haptic: null
    });
    expect(planCues(eventless, null, true).cues).toEqual([]);
    expect(planCues({ ...eventless, history: [] }, before, true).cues).toEqual([]);
    expect(planCues({ ...eventless, version: 0 }, before, true).cues).toEqual([]);
  });

  it('Animation off retains coalesced age feedback, while initial/history/undo frames stay silent', () => {
    settings.update({ sound: true, haptics: false, animationSpeed: 'off' });
    setPlanned(answer.version, EMPTY_TIMELINE);
    vi.mocked(play).mockClear();
    playFrameFx(answer, before);
    expect(vi.mocked(play).mock.calls.filter(([cue]) => cue === 'ageStart')).toHaveLength(1);
    vi.mocked(play).mockClear();
    playFrameFx(answer, null);
    playFrameFx({ ...answer, history: [] }, before);
    playFrameFx({ ...answer, version: 0 }, before);
    expect(play).not.toHaveBeenCalled();
  });
});
