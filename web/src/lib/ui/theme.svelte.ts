// The colour theme: the viewer's preference (system, light or dark) and the
// theme it resolves to. The inline bootstrap in src/app.html applies the
// stored preference before the first paint; this class takes over after
// start-up, follows the system setting while the preference is 'system', and
// keeps <html> in step (see apply()).

import { MediaQuery } from 'svelte/reactivity';

export const THEME_PREFS = ['system', 'light', 'dark'] as const;
export type ThemePref = (typeof THEME_PREFS)[number];
export type ResolvedTheme = 'light' | 'dark';

/** The localStorage key. src/app.html reads the same key. */
export const THEME_STORAGE_KEY = 'ajayeb:theme';

/** Browser chrome colours: the --table token of each theme, as hex. Keep in
 *  sync with tokens.css and the theme-color metas in src/app.html. */
export const THEME_COLORS: Readonly<Record<ResolvedTheme, string>> = {
  light: '#faf1df',
  dark: '#0a1936'
};

export function isThemePref(value: unknown): value is ThemePref {
  return typeof value === 'string' && (THEME_PREFS as readonly string[]).includes(value);
}

function readStoredPref(): ThemePref {
  try {
    const stored = globalThis.localStorage?.getItem(THEME_STORAGE_KEY);
    return isThemePref(stored) ? stored : 'system';
  } catch {
    return 'system'; // storage blocked: private mode or a sandboxed frame
  }
}

function storePref(pref: ThemePref): void {
  try {
    globalThis.localStorage?.setItem(THEME_STORAGE_KEY, pref);
  } catch {
    // Storage blocked; the choice still holds for this page.
  }
}

class Theme {
  #systemDark = new MediaQuery('(prefers-color-scheme: dark)', false);

  pref = $state<ThemePref>(readStoredPref());
  resolved: ResolvedTheme = $derived(
    this.pref === 'system' ? (this.#systemDark.current ? 'dark' : 'light') : this.pref
  );

  set(pref: ThemePref): void {
    this.pref = pref;
    storePref(pref);
  }

  /** Writes the resolved theme to <html>. Call it from an effect; it reads
   *  the reactive state, so the effect re-runs on every change. */
  apply(): void {
    const theme = this.resolved;
    const pref = this.pref;
    const root = document.documentElement;
    root.dataset.theme = theme;
    root.dataset.themePref = pref;
    root.classList.toggle('dark', theme === 'dark');
    root.style.colorScheme = theme;
    // The metas carry media queries for the system case; an explicit choice
    // overrides both, so the browser chrome matches the page.
    const metas = document.querySelectorAll<HTMLMetaElement>('meta[name="theme-color"]');
    for (const meta of metas) {
      const media = meta.media.includes('dark') ? 'dark' : 'light';
      meta.content = THEME_COLORS[pref === 'system' ? media : theme];
    }
  }
}

export const theme = new Theme();
