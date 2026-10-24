// The part of the table's sound that loads only once Sound is on (sound.ts
// imports it lazily): the app's player, $lib/fx, which Duel loads with its
// game screen. It imports names that player already exports, so Duel's
// chunks gain nothing.

import { installFx, playRecordedCues, unlockAudio } from '$lib/fx';
import type { PlannedCue } from './cues';

/** Installs the gesture that unlocks audio, and the test hook; unlocks now if asked. */
export function arm(unlock: boolean): void {
  installFx();
  if (unlock) unlockAudio();
}

/** Plays and records a frame's cues, tagged with its batch. */
export function play(cues: readonly PlannedCue[], batch: number): void {
  playRecordedCues(cues, batch);
}
