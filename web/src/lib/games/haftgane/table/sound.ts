// The table's sound and vibration (docs/art/sound.md, "Haftgane"). The
// plan comes from cues.ts; this plays it with the app's player, Duel's
// too: the same CC0 files and synthesized fallbacks, the same Sound,
// volume and Haptics settings read as the frame shows, and the same test
// hook (window.__ajayebFx), which records a cue only while Sound is on.
//
// The player is its own chunk (sound-player.ts), loaded once a table shows
// with Sound on, or when the menu turns Sound on; with Sound off it never
// loads. The audio files still wait for a gesture to unlock audio.
// Vibration needs no player: navigator.vibrate, while Haptics is on.

import { settings } from '$lib/ui/settings.svelte';
import { planTableCues, type CueTable } from './cues';

type Player = typeof import('./sound-player');
let player: Promise<Player> | null = null;

function load(): Promise<Player> {
  player ??= import('./sound-player').catch((err: unknown) => {
    // Offline before the chunk arrived: try again on the next frame.
    player = null;
    throw err;
  });
  return player;
}

/** Arms the player while Sound is on; `unlock` from inside a gesture. */
export function armTableSound(unlock = false): void {
  if (!settings.value.sound) return;
  load().then(
    (p) => p.arm(unlock),
    () => undefined
  );
}

/** Turns Sound on or off from the table, as the Settings screen does. */
export function setTableSound(on: boolean): void {
  settings.set('sound', on);
  armTableSound(true);
}

function vibrate(pattern: readonly number[]): void {
  if (typeof navigator === 'undefined' || typeof navigator.vibrate !== 'function') return;
  try {
    navigator.vibrate([...pattern]);
  } catch {
    // Blocked (no user activation yet, or a permissions policy): ignore.
  }
}

/** Sounds and vibrates `next`, the frame the table shows after `prev`. */
export function playTableSound(next: CueTable & { batch: number }, prev: CueTable | null): void {
  const { cues, haptic } = planTableCues(next, prev);
  if (haptic && settings.value.haptics) vibrate(haptic);
  if (cues.length === 0 || !settings.value.sound) return;
  load().then(
    (p) => p.play(cues, next.batch),
    () => undefined
  );
}
