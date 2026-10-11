// Vibration through navigator.vibrate (Android browsers; iOS has none, so
// there every call is a silent no-op). The Settings screen shows the
// Haptics switch only where hapticsSupported() is true.

/** Whether this browser offers navigator.vibrate. */
export function hapticsSupported(): boolean {
  return typeof navigator !== 'undefined' && typeof navigator.vibrate === 'function';
}

/** Vibrates with a pattern (ms on, off, on ...), or does nothing. */
export function vibrate(pattern: readonly number[]): void {
  if (!hapticsSupported()) return;
  try {
    navigator.vibrate([...pattern]);
  } catch {
    // Blocked (no user activation yet, or a permissions policy): ignore.
  }
}
