// Decoded CC0 sound files, fetched only after a user gesture unlocks audio.
// Each cue falls back immediately to its synth while loading or after failure.
// Nothing waits for a network request in the game's presentation path.

import { CUES, type Cue } from './cues';
import type { SoundVolume } from '$lib/ui/settings.svelte';

/** The master gain of each volume step. */
export const VOLUME_GAIN: Readonly<Record<SoundVolume, number>> = {
  quiet: 0.25,
  medium: 0.5,
  loud: 0.85
};

type Ctx = AudioContext;

let ctx: Ctx | null = null;
let noise: AudioBuffer | null = null;
const buffers = new Map<Cue, AudioBuffer>();
let loading: Promise<void> | null = null;
let sounds: (typeof import('./sounds.gen'))['SOUND_FILES'] | null = null;
const active: {
  start: number;
  end: number;
  stopAt: (at: number) => void;
  disconnect: () => void;
}[] = [];

/** Warm a small decoded cache once, after unlock. Two downloads at a time. */
export async function preloadAudio(): Promise<void> {
  const c = ctx;
  if (!c) return;
  if (c.state === 'suspended') {
    try {
      await c.resume();
    } catch {
      return;
    }
  }
  if (c.state !== 'running') return;
  if (!loading) {
    loading = (async () => {
      // Keep URL metadata out of the initial page payload too.
      const files = (await import('./sounds.gen')).SOUND_FILES;
      sounds = files;
      const queue = [...CUES];
      const worker = async () => {
        for (let cue = queue.shift(); cue; cue = queue.shift()) {
          try {
            const response = await fetch(files[cue].url);
            if (!response.ok) continue;
            const buffer = await c.decodeAudioData(await response.arrayBuffer());
            buffers.set(cue, buffer);
          } catch {
            // Failed files keep their synth; no rejected promise escapes.
          }
        }
      };
      await Promise.all([worker(), worker()]);
    })().catch(() => undefined);
  }
  return loading;
}

function audioContextClass(): (new () => AudioContext) | null {
  const g = globalThis as unknown as {
    AudioContext?: new () => AudioContext;
    webkitAudioContext?: new () => AudioContext;
  };
  return g.AudioContext ?? g.webkitAudioContext ?? null;
}

/**
 * Creates or resumes the AudioContext. Call it from inside a user gesture;
 * it is cheap to call on every one.
 */
export function unlockAudio(): void {
  try {
    if (!ctx) {
      const AC = audioContextClass();
      if (!AC) return;
      ctx = new AC();
    }
    if (ctx.state === 'suspended') {
      void ctx
        .resume()
        .then(() => preloadAudio())
        .catch(() => undefined);
    } else {
      void preloadAudio();
    }
  } catch {
    ctx = null; // a blocked or broken WebAudio stays silent
  }
}

/** Whether play() can make a sound now. */
export function audioReady(): boolean {
  return ctx !== null && ctx.state === 'running';
}

function noiseBuffer(c: Ctx): AudioBuffer {
  if (!noise || noise.sampleRate !== c.sampleRate) {
    const length = Math.floor(c.sampleRate * 0.4);
    noise = c.createBuffer(1, length, c.sampleRate);
    const data = noise.getChannelData(0);
    // A fixed linear congruential sequence: the same hiss every time.
    let x = 12345;
    for (let i = 0; i < length; i++) {
      x = (x * 1103515245 + 12345) & 0x7fffffff;
      data[i] = x / 0x3fffffff - 1;
    }
  }
  return noise;
}

/** An attack-decay envelope on a fresh gain node, connected to `out`. */
function env(c: Ctx, out: AudioNode, t: number, peak: number, attack: number, decay: number) {
  const g = c.createGain();
  g.gain.setValueAtTime(0.0001, t);
  g.gain.exponentialRampToValueAtTime(peak, t + attack);
  g.gain.exponentialRampToValueAtTime(0.0001, t + attack + decay);
  g.connect(out);
  return g;
}

function tone(
  c: Ctx,
  out: AudioNode,
  t: number,
  type: OscillatorType,
  freq: number,
  peak: number,
  attack: number,
  decay: number,
  endFreq?: number
): void {
  const o = c.createOscillator();
  o.type = type;
  o.frequency.setValueAtTime(freq, t);
  if (endFreq !== undefined) o.frequency.exponentialRampToValueAtTime(endFreq, t + attack + decay);
  o.connect(env(c, out, t, peak, attack, decay));
  o.start(t);
  o.stop(t + attack + decay + 0.02);
}

function hiss(
  c: Ctx,
  out: AudioNode,
  t: number,
  filter: BiquadFilterType,
  freq: number,
  endFreq: number,
  q: number,
  peak: number,
  attack: number,
  decay: number
): void {
  const src = c.createBufferSource();
  src.buffer = noiseBuffer(c);
  const f = c.createBiquadFilter();
  f.type = filter;
  f.Q.value = q;
  f.frequency.setValueAtTime(freq, t);
  f.frequency.exponentialRampToValueAtTime(endFreq, t + attack + decay);
  src.connect(f);
  f.connect(env(c, out, t, peak, attack, decay));
  src.start(t);
  src.stop(t + attack + decay + 0.02);
}

/** Each cue, drawn into `out` starting at time t. */
const VOICES: Readonly<Record<Cue, (c: Ctx, out: AudioNode, t: number) => void>> = {
  // A card sliding over felt: band-passed noise sweeping down.
  slide: (c, out, t) => hiss(c, out, t, 'bandpass', 2400, 700, 1.2, 0.5, 0.015, 0.14),
  // Two small metal partials, a second strike just after the first.
  coin: (c, out, t) => {
    tone(c, out, t, 'triangle', 2093, 0.22, 0.003, 0.12);
    tone(c, out, t, 'sine', 3136, 0.1, 0.003, 0.08);
    tone(c, out, t + 0.06, 'triangle', 2637, 0.16, 0.003, 0.14);
  },
  // A crisp snap of card stock.
  flip: (c, out, t) => {
    hiss(c, out, t, 'highpass', 3000, 1800, 0.7, 0.35, 0.004, 0.05);
    tone(c, out, t, 'sine', 640, 0.08, 0.004, 0.05, 420);
  },
  // A low skin with a falling pitch, and a soft thump of noise.
  drum: (c, out, t) => {
    tone(c, out, t, 'sine', 120, 0.7, 0.005, 0.32, 52);
    hiss(c, out, t, 'lowpass', 900, 200, 0.8, 0.25, 0.004, 0.09);
    tone(c, out, t + 0.16, 'sine', 100, 0.35, 0.005, 0.22, 50);
  },
  // A bright fifth, its upper note a beat late, ringing out.
  chime: (c, out, t) => {
    tone(c, out, t, 'sine', 1046.5, 0.22, 0.005, 0.7);
    tone(c, out, t + 0.07, 'sine', 1568, 0.16, 0.005, 0.65);
    tone(c, out, t, 'sine', 2093, 0.04, 0.005, 0.3);
  },
  // A rising major arpeggio, the last note held.
  fanfare: (c, out, t) => {
    const notes = [523.25, 659.25, 783.99, 1046.5];
    notes.forEach((f, i) => {
      const last = i === notes.length - 1;
      tone(c, out, t + i * 0.12, 'triangle', f, 0.24, 0.01, last ? 0.9 : 0.18);
      tone(c, out, t + i * 0.12, 'sine', f * 2, 0.05, 0.01, last ? 0.6 : 0.12);
    });
  },
  // A tiny wooden tick.
  tick: (c, out, t) => tone(c, out, t, 'sine', 1300, 0.12, 0.002, 0.03, 900),
  wonder: (c, out, t) => {
    tone(c, out, t, 'triangle', 196, 0.3, 0.008, 0.75);
    tone(c, out, t, 'sine', 392, 0.12, 0.008, 0.65);
  },
  token: (c, out, t) => tone(c, out, t, 'triangle', 720, 0.2, 0.003, 0.12, 450),
  loot: (c, out, t) => {
    for (let i = 0; i < 3; i++)
      tone(c, out, t + i * 0.09, 'triangle', 2100 - i * 300, 0.18, 0.003, 0.16);
  },
  ageStart: (c, out, t) => hiss(c, out, t, 'bandpass', 1800, 700, 1, 0.3, 0.04, 0.65),
  defeat: (c, out, t) => {
    [523.25, 440, 349.23].forEach((f, i) => tone(c, out, t + i * 0.18, 'sine', f, 0.2, 0.02, 0.6));
  },
  yourTurn: (c, out, t) => tone(c, out, t, 'sine', 880, 0.12, 0.005, 0.18),
  destroy: (c, out, t) => hiss(c, out, t, 'highpass', 1900, 700, 0.7, 0.3, 0.01, 0.28),
  rebuild: (c, out, t) => hiss(c, out, t, 'bandpass', 700, 2400, 1, 0.3, 0.04, 0.25),
  libraryDraw: (c, out, t) => hiss(c, out, t, 'bandpass', 1200, 2200, 0.7, 0.3, 0.06, 0.6),
  extraTurn: (c, out, t) => {
    tone(c, out, t, 'sine', 880, 0.18, 0.005, 0.2);
    tone(c, out, t + 0.12, 'sine', 1174.66, 0.15, 0.005, 0.25);
  }
};

export interface PlayOptions {
  /** Start this many ms from now (the animation slice retimes cues with it). */
  delayMs?: number;
  volume?: SoundVolume;
  /** Gallery comparison: require a file or force its synthesized fallback. */
  source?: 'synth' | 'file';
}

/** Plays one cue if unlocked; returns whether playback started. Safe anywhere. */
export function play(cue: Cue, opts: PlayOptions = {}): boolean {
  if (!ctx || ctx.state !== 'running') return false;
  try {
    const c = ctx;
    const buffer = opts.source === 'synth' ? undefined : buffers.get(cue);
    if (opts.source === 'file' && !buffer) return false;
    const master = c.createGain();
    master.gain.value = VOLUME_GAIN[opts.volume ?? 'medium'];
    master.connect(c.destination);
    const t = c.currentTime + Math.max(0, opts.delayMs ?? 0) / 1000 + 0.005;
    const duration = buffer?.duration ?? 1.5;
    if (buffer && sounds) master.gain.value *= sounds[cue].gain;
    // Keep at most three overlapping cues, including scheduled ones.
    for (let i = active.length - 1; i >= 0; i--) {
      if (active[i]!.end <= c.currentTime) active.splice(i, 1);
    }
    const end = t + duration;
    const starts = [t, ...active.filter((v) => v.start > t && v.start < end).map((v) => v.start)];
    for (const start of starts) {
      const overlapping = active.filter((v) => v.start <= start && v.end > start);
      while (overlapping.length >= 3) overlapping.shift()!.stopAt(start);
    }
    let source: AudioBufferSourceNode | undefined;
    if (buffer) {
      source = c.createBufferSource();
      source.buffer = buffer;
      source.connect(master);
      source.start(t);
    } else {
      VOICES[cue](c, master, t);
    }
    const voice = {
      start: t,
      end,
      stopAt: (at: number) => {
        // Planning a later cue must not silence a voice that is still waiting
        // to start. WebAudio applies this cut at the actual overlap time.
        voice.end = Math.min(voice.end, at);
        master.gain.setValueAtTime(0, voice.end);
        source?.stop(voice.end);
      },
      disconnect: () => {
        clearTimeout(timer);
        source?.stop();
        master.disconnect();
        const i = active.indexOf(voice);
        if (i >= 0) active.splice(i, 1);
      }
    };
    active.push(voice);
    const timer = setTimeout(
      voice.disconnect,
      Math.max(0, opts.delayMs ?? 0) + duration * 1000 + 100
    );
    return true;
  } catch {
    // A failing sound never breaks the game.
    return false;
  }
}
