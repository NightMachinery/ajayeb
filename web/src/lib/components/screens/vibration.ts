// Whether Settings offers the Vibration switch: only on a device that both
// can vibrate and has a touch screen. Desktop Chrome has navigator.vibrate
// too, but a desktop never vibrates, so a switch there would do nothing.

/** The parts of navigator the rule reads. */
export interface VibrationNavigator {
  vibrate?: unknown;
  maxTouchPoints?: number;
}

export function vibrationShown(nav: VibrationNavigator | undefined): boolean {
  return typeof nav?.vibrate === 'function' && (nav.maxTouchPoints ?? 0) > 0;
}
