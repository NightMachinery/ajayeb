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
// kind is recorded, which is what "would play" means. A personal turn
// notice records its plan, even if a later frame or mute cancels delivery.
// The same hook lists
// what the animation runner did (fx/anim/records.ts): each frame's swap
// time and the moment it let input go, and each animation's start and end.
//
// When the frame animates (fx/anim), its cues sound with their animations
// (anim/speed.ts): the slide when the card lands, and so on. With
// Animation off they keep cues.ts's own spacing.

import { settings } from '$lib/ui/settings.svelte';
import { retimeCues } from './anim/speed';
import {
  animRecords,
  clearRecords,
  frameRecords,
  plannedFor,
  type AnimRecord,
  type FrameRecord
} from './anim/records';
import { planCues, type Cue, type CueFrame } from './cues';
import { scheduleHaptic } from './haptics';
import { play, unlockAudio } from './sfx';
import { cancelTurnCue, waitTurnCue } from './turn-pending';

export { CUES, type Cue } from './cues';
export { play, unlockAudio, VOLUME_GAIN } from './sfx';
export { hapticsSupported } from './haptics';
export type { AnimRecord, FrameRecord } from './anim/records';

export interface RecordedCue {
  cue: Cue;
  delayMs: number;
  /** The version of the frame that made it. */
  version: number;
  /** Whether the cue belongs to a visual replay rather than a new frame. */
  replay: boolean;
}

export interface AjayebFxHook {
  /** The cues recorded since the page loaded or the last clear(), oldest first. */
  cues(): RecordedCue[];
  /** The vibration patterns recorded likewise. */
  haptics(): number[][];
  /** The frames the animation runner was handed, oldest first. */
  frames(): FrameRecord[];
  /** The animations it started, oldest first. */
  anims(): AnimRecord[];
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
  frames: frameRecords,
  anims: animRecords,
  clear: () => {
    recordedCues.length = 0;
    recordedHaptics.length = 0;
    clearRecords();
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
  window.addEventListener('pagehide', cancelTurnCue);
  window.addEventListener('pointerdown', onGesture, { capture: true, passive: true });
  window.addEventListener('keydown', onGesture, { capture: true, passive: true });
}

/** Shared sound playback and recording for live frames and visual replays. */
export function playRecordedCues(
  cues: readonly { cue: Cue; delayMs: number }[],
  version: number,
  replay = false
): void {
  install();
  const { sound, soundVolume } = settings.value;
  if (!sound) return;
  for (const { cue, delayMs } of cues) {
    // A visual replay cannot hand over the live turn.
    if (cue === 'yourTurn' && replay) continue;
    push(recordedCues, { cue, delayMs, version, replay });
    if (cue === 'yourTurn') {
      waitTurnCue(() => {
        if (settings.value.sound) play(cue, { delayMs: 0, volume: settings.value.soundVolume });
      }, delayMs);
    } else {
      play(cue, { delayMs, volume: soundVolume });
    }
  }
}

/** Plays a frame's cues and schedules its single vibration at the matching step. */
export function playFrameFx(next: CueFrame, prev: CueFrame | null): void {
  cancelTurnCue();
  const timeline = plannedFor(next.version);
  const playedDeal =
    timeline && settings.value.animationSpeed !== 'off'
      ? timeline.steps.some((s) => s.kind === 'deal')
      : undefined;
  const plan = planCues(next, prev, playedDeal);
  playRecordedCues(timeline ? retimeCues(plan.cues, timeline) : plan.cues, next.version);
  scheduleHaptic(plan.haptic, timeline, (pattern) => push(recordedHaptics, [...pattern]));
}

/** Installs the gesture unlock early (the Settings screen calls it). */
export function installFx(): void {
  install();
}
