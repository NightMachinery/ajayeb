// Vibration through navigator.vibrate (Android browsers; iOS has none, so
// there every call is a silent no-op). The Settings screen shows the
// Haptics switch only where hapticsSupported() is true.
//
// Timing (M10.animations-events): a vibration starts with the animation
// step it belongs to (anim/plan.ts hapticOffset), in the timeline the
// runner plays for the frame on screen (anim/records.ts latestPlanned, set
// at the frame's capture, just before fx/index.ts asks for the vibration):
// the take's when the card lands, the military one with the pawn's first
// step, the looting one with the looting token's flip. With Animation off,
// or a frame that does not animate, it starts at once. A new vibration
// cancels one still waiting.

import { latestPlanned } from './anim/records';
import { hapticOffset } from './anim/plan';

/** Whether this browser offers navigator.vibrate. */
export function hapticsSupported(): boolean {
  return typeof navigator !== 'undefined' && typeof navigator.vibrate === 'function';
}

let waiting: ReturnType<typeof setTimeout> | null = null;

function buzz(pattern: readonly number[]): void {
  try {
    navigator.vibrate([...pattern]);
  } catch {
    // Blocked (no user activation yet, or a permissions policy): ignore.
  }
}

/** When `pattern` should start, in ms from now: its step in the frame's timeline. */
export function hapticDelay(pattern: readonly number[]): number {
  const timeline = latestPlanned();
  return timeline ? hapticOffset(timeline, pattern) : 0;
}

/** Vibrates with a pattern (ms on, off, on ...), at its step, or does nothing. */
export function vibrate(pattern: readonly number[]): void {
  if (!hapticsSupported()) return;
  if (waiting !== null) clearTimeout(waiting);
  waiting = null;
  const delay = hapticDelay(pattern);
  if (delay <= 0) {
    buzz(pattern);
    return;
  }
  waiting = setTimeout(() => {
    waiting = null;
    buzz(pattern);
  }, delay);
}
