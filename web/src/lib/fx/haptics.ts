// Vibration through navigator.vibrate (Android browsers; iOS has none, so
// there every call is a silent no-op). The Settings screen shows the
// Haptics switch only where hapticsSupported() is true.
//
// Timing (M10.animations-events): a vibration starts with the animation
// step it belongs to (anim/plan.ts hapticOffset), in the timeline the
// runner plays for the frame on screen (the explicit timeline passed by fx/index.ts, set
// at the frame's capture, just before fx/index.ts asks for the vibration):
// the take's when the card lands, the military one with the pawn's first
// step, the looting one with the looting token's flip. With Animation off,
// or a frame that does not animate, it starts at once. A new vibration
// cancels one still waiting.

import { cancelHaptic, waitHaptic } from './haptic-pending';
export { cancelHaptic } from './haptic-pending';
import { settings } from '$lib/ui/settings.svelte';
import type { Timeline } from './anim/plan';
import { hapticOffset } from './anim/plan';

/** Whether this browser offers navigator.vibrate. */
export function hapticsSupported(): boolean {
  return typeof navigator !== 'undefined' && typeof navigator.vibrate === 'function';
}

function buzz(pattern: readonly number[]): void {
  try {
    navigator.vibrate([...pattern]);
  } catch {
    // Blocked (no user activation yet, or a permissions policy): ignore.
  }
}

/** When `pattern` should start, in ms from now: its step in the frame's timeline. */
export function hapticDelay(pattern: readonly number[], timeline: Timeline | null): number {
  return timeline ? hapticOffset(timeline, pattern) : 0;
}

/** The one scheduling path, including Animation off (an immediate vibration).
 *  Record only a vibration that would actually play, after the final setting check. */
export function scheduleHaptic(
  pattern: readonly number[] | null,
  timeline: Timeline | null,
  onplay: (pattern: readonly number[]) => void
): void {
  cancelHaptic();
  if (!pattern || !settings.value.haptics || !hapticsSupported()) return;
  const play = () => {
    if (!settings.value.haptics) return;
    onplay(pattern);
    buzz(pattern);
  };
  const delay = hapticDelay(pattern, timeline);
  if (delay <= 0) play();
  else waitHaptic(play, delay);
}
