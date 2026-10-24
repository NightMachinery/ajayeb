// When Settings and the game menus offer Install, the iOS steps and
// Fullscreen (M18.installable-and-fullscreen).
import { describe, expect, it } from 'vitest';
import { fullscreenShown } from './fullscreen.svelte';
import { installOffer, isIos, runsInstalled, type InstallWindow } from './install.svelte';

const page = (standalone: boolean) => ({
  matchMedia: (q: string) => ({ matches: standalone && q === '(display-mode: standalone)' })
});
const browserTab: InstallWindow = { ...page(false), navigator: { maxTouchPoints: 0 } };
const androidChrome: InstallWindow = { ...page(false), navigator: { maxTouchPoints: 5 } };
const iphoneSafari: InstallWindow = {
  ...page(false),
  navigator: { standalone: false, maxTouchPoints: 5 }
};
const iphoneHomeScreen: InstallWindow = {
  ...page(false),
  navigator: { standalone: true, maxTouchPoints: 5 }
};
const macSafari: InstallWindow = {
  ...page(false),
  navigator: { standalone: false, maxTouchPoints: 0 }
};
const installedChrome: InstallWindow = { ...page(true), navigator: { maxTouchPoints: 5 } };

describe('the Install row', () => {
  it('offers the button only while the browser holds an install prompt', () => {
    expect(installOffer(false, true, false)).toBe('prompt');
    expect(installOffer(false, false, false)).toBeNull();
  });

  it('shows the Add to Home Screen steps on iOS, which has no prompt', () => {
    expect(installOffer(false, false, true)).toBe('ios');
  });

  it('shows nothing once the app runs installed', () => {
    expect(installOffer(true, true, false)).toBeNull();
    expect(installOffer(true, false, true)).toBeNull();
  });

  it('knows an installed app by its display mode or an iOS home screen launch', () => {
    expect(runsInstalled(installedChrome)).toBe(true);
    expect(runsInstalled(iphoneHomeScreen)).toBe(true);
    expect(runsInstalled(browserTab)).toBe(false);
    expect(runsInstalled(iphoneSafari)).toBe(false);
    expect(runsInstalled({ navigator: {} })).toBe(false);
  });

  it('takes only a touch device with navigator.standalone for iOS', () => {
    expect(isIos(iphoneSafari)).toBe(true);
    expect(isIos(iphoneHomeScreen)).toBe(true);
    expect(isIos(macSafari)).toBe(false);
    expect(isIos(androidChrome)).toBe(false);
    expect(isIos(browserTab)).toBe(false);
  });
});

describe('the Fullscreen toggle', () => {
  const request = () => Promise.resolve();

  it('shows where the page may use the Fullscreen API', () => {
    expect(
      fullscreenShown({ fullscreenEnabled: true, documentElement: { requestFullscreen: request } })
    ).toBe(true);
  });

  it('hides on iPhone Safari, which has fullscreen only for video', () => {
    expect(fullscreenShown({ documentElement: {} })).toBe(false);
  });

  it('hides where a frame or policy forbids it, and without a document', () => {
    expect(
      fullscreenShown({ fullscreenEnabled: false, documentElement: { requestFullscreen: request } })
    ).toBe(false);
    expect(fullscreenShown(undefined)).toBe(false);
  });
});
