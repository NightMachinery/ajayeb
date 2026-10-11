// Sound and haptics for the frames the Presenter shows. playFrameFx() is
// the one call site's entry point (session/presenter.svelte.ts calls it
// after each swap): it plans the frame's cues (cues.ts), plays them if
// Sound is on (sfx.ts) and vibrates if Haptics is on (haptics.ts), both
// read from the app's settings at the moment the frame shows.
//
// The test hook: window.__ajayebFx records what would play, whether or not
// an AudioContext exists (CI has no audio device, and headless Chrome
// never unlocks one without a real gesture). A cue is recorded when Sound
// is on, a vibration when Haptics is on; with a setting off nothing of its
// kind is recorded, which is what "would play" means.

import { settings } from '$lib/ui/settings.svelte';
import { planCues, type Cue, type CueFrame } from './cues';
import { hapticsSupported, vibrate } from './haptics';
import { play, unlockAudio } from './sfx';

export { CUES, type Cue } from './cues';
export { play, unlockAudio, VOLUME_GAIN } from './sfx';
export { hapticsSupported } from './haptics';

export interface RecordedCue {
  cue: Cue;
  delayMs: number;
  /** The version of the frame that made it. */
  version: number;
}

export interface AjayebFxHook {
  /** The cues recorded since the page loaded or the last clear(), oldest first. */
  cues(): RecordedCue[];
  /** The vibration patterns recorded likewise. */
  haptics(): number[][];
  clear(): void;
}

declare global {
  interface Window {
    __ajayebFx?: AjayebFxHook;
  }
}

const LIMIT = 500;
const recordedCues: RecordedCue[] = [];
const recordedHaptics: number[][] = [];

function push<T>(list: T[], item: T): void {
  list.push(item);
  if (list.length > LIMIT) list.splice(0, list.length - LIMIT);
}

const hook: AjayebFxHook = {
  cues: () => recordedCues.map((c) => ({ ...c })),
  haptics: () => recordedHaptics.map((h) => [...h]),
  clear: () => {
    recordedCues.length = 0;
    recordedHaptics.length = 0;
  }
};

let installed = false;

/**
 * Installs the test hook and the gesture listener that unlocks audio. Runs
 * once, on the first frame; nothing here touches audio until Sound is on.
 */
function install(): void {
  if (installed || typeof window === 'undefined') return;
  installed = true;
  if (!window.__ajayebFx) {
    Object.defineProperty(window, '__ajayebFx', { value: hook, configurable: true });
  }
  const onGesture = () => {
    if (settings.value.sound) unlockAudio();
  };
  window.addEventListener('pointerdown', onGesture, { capture: true, passive: true });
  window.addEventListener('keydown', onGesture, { capture: true, passive: true });
}

/** Plays the sound and haptics of the frame `next`, shown after `prev`. */
export function playFrameFx(next: CueFrame, prev: CueFrame | null): void {
  install();
  const plan = planCues(next, prev);
  const { sound, soundVolume, haptics } = settings.value;
  if (sound) {
    for (const { cue, delayMs } of plan.cues) {
      push(recordedCues, { cue, delayMs, version: next.version });
      play(cue, { delayMs, volume: soundVolume });
    }
  }
  if (haptics && plan.haptic && hapticsSupported()) {
    push(recordedHaptics, [...plan.haptic]);
    vibrate(plan.haptic);
  }
}

/** Installs the gesture unlock early (the Settings screen calls it). */
export function installFx(): void {
  install();
}
