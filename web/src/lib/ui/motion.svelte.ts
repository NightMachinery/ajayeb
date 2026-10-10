// The effective animation speed, for FX code (a later slice) to scale its
// durations by: the Settings sheet's animationSpeed preference, forced to
// 'off' whenever the OS itself asks for reduced motion (svelte/motion's
// prefersReducedMotion). The OS always wins, because a motion-sensitive
// viewer never chose the app's default; app.css's own
// `@media (prefers-reduced-motion: reduce)` block already collapses CSS
// transitions and animations the same way, so WAAPI/JS-driven FX (ms())
// needs the same override made explicit here.
//
// settings.svelte.ts owns the stored preference; this module only computes
// what to do with it. effectiveSpeed() and scaleMs() are plain functions so
// they can be tested without mocking the browser's matchMedia.

import { prefersReducedMotion } from 'svelte/motion';
import { settings, SettingsStore, type AnimationSpeed } from './settings.svelte';

export { ANIMATION_SPEEDS, type AnimationSpeed } from './settings.svelte';

/** Duration multiplier per speed. 'off' always resolves to exactly 0 in
 *  scaleMs(), regardless of this table, so a future edit here cannot
 *  reintroduce a non-zero "off". */
const SPEED_SCALE: Readonly<Record<AnimationSpeed, number>> = {
  normal: 1,
  fast: 0.6,
  relaxed: 1.6,
  off: 0
};

/** The OS's reduced-motion preference overrides the app setting: it can only
 *  force 'off', never un-force an app choice of 'off'. */
export function effectiveSpeed(preference: AnimationSpeed, reduced: boolean): AnimationSpeed {
  return reduced ? 'off' : preference;
}

/** A base duration in ms, scaled for a speed. Always exactly 0 for 'off',
 *  so a caller never needs to special-case it before animating. */
export function scaleMs(base: number, speed: AnimationSpeed): number {
  return speed === 'off' ? 0 : Math.round(base * SPEED_SCALE[speed]);
}

export interface MotionOptions {
  /** Defaults to the app's one SettingsStore; a test passes its own, so it
   *  can change a preference without touching the shared singleton. */
  settings?: SettingsStore;
}

export class Motion {
  // Defaulted at declaration (not just in the constructor body) so the
  // $derived fields below, which read it, never see "used before its
  // initialization": TypeScript's definite-assignment check looks at field
  // declaration order, not at $derived's actual lazy (read-time) evaluation.
  #settings: SettingsStore = settings;

  constructor(opts: MotionOptions = {}) {
    if (opts.settings) this.#settings = opts.settings;
  }

  /** True when the OS asks for reduced motion (svelte/motion's MediaQuery;
   *  a Node/SSR stub always reports false, so tests see the plain setting). */
  reduced: boolean = $derived(prefersReducedMotion.current);
  /** The app's own Settings choice, before the OS override. */
  preference: AnimationSpeed = $derived(this.#settings.value.animationSpeed);
  /** What FX code should actually use. */
  effective: AnimationSpeed = $derived(effectiveSpeed(this.preference, this.reduced));
  /** Whether the Settings screen should show its "forced off" note: the OS
   *  is overriding a preference that is not already 'off' itself. */
  forcedOff: boolean = $derived(this.reduced && this.preference !== 'off');

  /** Scales a base duration (ms) by the effective speed. */
  ms(base: number): number {
    return scaleMs(base, this.effective);
  }

  /** Mirrors the effective speed onto <html data-motion>, which app.css
   *  already reads to force every CSS transition/animation to ~0 (the same
   *  attribute Settings' "forced off" state and the OS media query both
   *  drive). DOM-only; call it from an effect, as theme.apply() is called. */
  apply(root: HTMLElement = document.documentElement): void {
    if (this.effective === 'off') root.dataset['motion'] = 'reduce';
    else delete root.dataset['motion'];
  }
}

/** The app's one Motion instance, derived from the real settings store. */
export const motion = new Motion();
