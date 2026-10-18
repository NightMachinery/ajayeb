import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { CUES } from './cues';
import { SOUND_FILES } from './sounds.gen';

const param = () => ({
  value: 0,
  setValueAtTime: vi.fn(),
  exponentialRampToValueAtTime: vi.fn()
});

function context() {
  const sources: {
    buffer: unknown;
    start: ReturnType<typeof vi.fn>;
    stop: ReturnType<typeof vi.fn>;
  }[] = [];
  const gains: { gain: ReturnType<typeof param>; disconnect: ReturnType<typeof vi.fn> }[] = [];
  const c = {
    state: 'running',
    sampleRate: 48000,
    currentTime: 10,
    destination: {},
    resume: vi.fn(async () => {
      c.state = 'running';
    }),
    decodeAudioData: vi.fn(async (data: ArrayBuffer) => {
      void data;
      return { duration: 2.3 };
    }),
    createGain: () => {
      const g = { gain: param(), connect: vi.fn(), disconnect: vi.fn() };
      gains.push(g);
      return g;
    },
    createOscillator: vi.fn(() => ({
      frequency: param(),
      connect: vi.fn(),
      start: vi.fn(),
      stop: vi.fn()
    })),
    createBuffer: () => ({ sampleRate: 48000, getChannelData: () => new Float32Array(19200) }),
    createBiquadFilter: () => ({ Q: param(), frequency: param(), connect: vi.fn() }),
    createBufferSource: () => {
      const source = { buffer: null as unknown, connect: vi.fn(), start: vi.fn(), stop: vi.fn() };
      sources.push(source);
      return source;
    }
  };
  return { c, sources, gains };
}

beforeEach(() => {
  vi.resetModules();
  vi.useFakeTimers();
});
afterEach(() => {
  vi.runAllTimers();
  vi.useRealTimers();
  vi.unstubAllGlobals();
});

async function setup() {
  const state = context();
  const constructed = vi.fn();
  vi.stubGlobal(
    'AudioContext',
    class {
      constructor() {
        constructed();
        return state.c;
      }
    }
  );
  const fetcher = vi.fn(async (url: string) => {
    const id = CUES.findIndex((cue) => SOUND_FILES[cue].url === url);
    return { ok: true, arrayBuffer: async () => new Uint8Array([id]).buffer };
  });
  vi.stubGlobal('fetch', fetcher);
  return { ...state, fetcher, constructed, sfx: await import('./sfx') };
}

describe('sound file player', () => {
  it('does no audio work before a gesture, then fetches and decodes each hashed cue once', async () => {
    const { sfx, fetcher, c, constructed } = await setup();
    sfx.play('coin');
    await sfx.preloadAudio();
    expect(fetcher).not.toHaveBeenCalled();
    expect(constructed).not.toHaveBeenCalled();
    expect(sfx.audioReady()).toBe(false);
    sfx.unlockAudio();
    await sfx.preloadAudio();
    sfx.unlockAudio();
    await sfx.preloadAudio();
    expect(constructed).toHaveBeenCalledOnce();
    expect(fetcher).toHaveBeenCalledTimes(CUES.length);
    expect(c.decodeAudioData).toHaveBeenCalledTimes(CUES.length);
    expect(fetcher.mock.calls.map(([url]) => url).sort()).toEqual(
      CUES.map((cue) => SOUND_FILES[cue].url).sort()
    );
  });

  it('falls back while loading and after a fetch or decode failure without replaying a late file', async () => {
    const { sfx, fetcher, c, sources } = await setup();
    let finish!: (value: { ok: boolean; arrayBuffer: () => Promise<ArrayBuffer> }) => void;
    fetcher.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finish = resolve;
        })
    );
    c.decodeAudioData.mockRejectedValue(new Error('unsupported file'));
    sfx.unlockAudio();
    sfx.play('coin');
    expect(c.createOscillator).toHaveBeenCalled();
    expect(sources).toHaveLength(0);
    await vi.waitFor(() => expect(finish).toBeDefined());
    finish({ ok: false, arrayBuffer: async () => new ArrayBuffer(0) });
    await sfx.preloadAudio();
    const count = c.createOscillator.mock.calls.length;
    sfx.play('coin');
    expect(c.createOscillator.mock.calls.length).toBeGreaterThan(count);
    expect(sources).toHaveLength(0);
    expect(fetcher).toHaveBeenCalledTimes(CUES.length);
  });

  it('never substitutes synth for a file-only audition that failed', async () => {
    const { sfx, fetcher, c, sources } = await setup();
    fetcher.mockResolvedValue({ ok: false, arrayBuffer: async () => new ArrayBuffer(0) });
    sfx.unlockAudio();
    await sfx.preloadAudio();
    expect(sfx.play('coin', { source: 'file' })).toBe(false);
    expect(c.createOscillator).not.toHaveBeenCalled();
    expect(sources).toHaveLength(0);
    expect(sfx.play('coin')).toBe(true);
    expect(c.createOscillator).toHaveBeenCalled();
  });

  it('auditions the selected cue buffer rather than a different cached file', async () => {
    const { sfx, c, sources } = await setup();
    const coin = { duration: 0.7 };
    c.decodeAudioData.mockImplementation(async (data) =>
      new Uint8Array(data)[0] === CUES.indexOf('coin') ? coin : { duration: 0.2 }
    );
    sfx.unlockAudio();
    await sfx.preloadAudio();
    expect(sfx.play('coin', { source: 'file' })).toBe(true);
    expect(sources.at(-1)!.buffer).toBe(coin);
  });

  it('plays the buffer with the volume step and delay, and keeps long files connected to the end', async () => {
    const { sfx, sources, gains, c } = await setup();
    sfx.unlockAudio();
    await sfx.preloadAudio();
    sfx.play('defeat', { volume: 'quiet', delayMs: 90 });
    expect(sources[0]!.start).toHaveBeenCalledWith(10.095);
    expect(gains[0]!.gain.value).toBe(sfx.VOLUME_GAIN.quiet);
    expect(c.createOscillator).not.toHaveBeenCalled();
    vi.advanceTimersByTime(2200);
    expect(gains[0]!.disconnect).not.toHaveBeenCalled();
    vi.advanceTimersByTime(400);
    expect(gains[0]!.disconnect).toHaveBeenCalledOnce();
  });

  it('can audition every synth even with files loaded, and caps overlapping voices at three', async () => {
    const { sfx, c, sources, gains } = await setup();
    sfx.unlockAudio();
    await sfx.preloadAudio();
    for (const cue of CUES) expect(() => sfx.play(cue, { source: 'synth' })).not.toThrow();
    expect(c.createOscillator).toHaveBeenCalled();
    vi.runAllTimers();
    const before = gains.length;
    for (const cue of ['coin', 'coin', 'coin', 'coin'] as const) sfx.play(cue);
    expect(sources.at(-4)!.stop).toHaveBeenCalled();
    expect(gains[before]!.disconnect).not.toHaveBeenCalled();
    expect(gains[before]!.gain.setValueAtTime).toHaveBeenCalledWith(0, 10.005);
    expect(gains.slice(before + 1).every((g) => g.disconnect.mock.calls.length === 0)).toBe(true);
  });

  it('cuts at the future overlap without silencing earlier playback', async () => {
    const { sfx, sources, gains } = await setup();
    sfx.unlockAudio();
    await sfx.preloadAudio();
    for (const delayMs of [0, 90, 180, 270]) sfx.play('coin', { delayMs });
    expect(sources[0]!.start).toHaveBeenCalledWith(10.005);
    expect(sources[0]!.stop).toHaveBeenCalledOnce();
    const cut = sources[0]!.stop.mock.calls[0]![0] as number;
    expect(cut).toBeCloseTo(10.275);
    expect(gains[0]!.gain.setValueAtTime).toHaveBeenCalledWith(0, cut);
    expect(gains[0]!.disconnect).not.toHaveBeenCalled();
    expect(sources.slice(1).every((source) => source.stop.mock.calls.length === 0)).toBe(true);
  });

  it('does not drop spaced cues that will not overlap', async () => {
    const { sfx, sources } = await setup();
    sfx.unlockAudio();
    await sfx.preloadAudio();
    for (let i = 0; i < 4; i++) sfx.play('coin', { delayMs: i * 3000 });
    expect(sources.every((source) => source.stop.mock.calls.length === 0)).toBe(true);
  });

  it('silently handles a browser refusing resume', async () => {
    const { sfx, c, fetcher } = await setup();
    c.state = 'suspended';
    c.resume.mockRejectedValue(new Error('gesture denied'));
    sfx.unlockAudio();
    await sfx.preloadAudio();
    expect(sfx.audioReady()).toBe(false);
    expect(fetcher).not.toHaveBeenCalled();
  });
});
