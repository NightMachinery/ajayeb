// The theme's default (M18.installable-and-fullscreen): someone who never
// chose gets light, and a stored choice is kept. src/app-bootstrap.test.ts
// checks the same for the first paint.
import { afterEach, describe, expect, it, vi } from 'vitest';
import { DEFAULT_THEME_PREF, readStoredPref, THEME_STORAGE_KEY } from './theme.svelte';

function stubStorage(value: string | null | 'blocked'): void {
  vi.stubGlobal('localStorage', {
    getItem(key: string): string | null {
      if (value === 'blocked') throw new Error('storage blocked');
      return key === THEME_STORAGE_KEY ? value : null;
    },
    setItem(): void {}
  });
}

describe('the theme preference', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('is light for someone who never chose', () => {
    expect(DEFAULT_THEME_PREF).toBe('light');
    for (const value of [null, 'blocked', 'sepia'] as const) {
      stubStorage(value);
      expect(readStoredPref()).toBe('light');
    }
  });

  it('keeps a stored choice', () => {
    for (const value of ['light', 'dark', 'system'] as const) {
      stubStorage(value);
      expect(readStoredPref()).toBe(value);
    }
  });
});
