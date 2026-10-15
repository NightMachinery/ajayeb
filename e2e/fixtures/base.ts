// The shared `test` every spec imports instead of '@playwright/test'
// directly. Its only job is to make the default `context` (and so `page`)
// fixture start with the game's own Animation setting forced off, the
// opponent move summary popup off and the first-time tips off.
//
// This replaces playwright.config.ts's old `use.reducedMotion: 'reduce'`.
// The client no longer reads the OS's prefers-reduced-motion at all (D21,
// docs/dev/decisions.md: animation follows the game's own setting, not the
// OS), so that config option stopped affecting app behaviour; forcing the
// game's own setting instead keeps the suite's pacing fast and deterministic.
//
// `newDeviceContext` (fixtures/online.ts) creates its extra browser contexts
// by hand with `browser.newContext()`, which bypasses this fixture, so it
// calls `installMotionOff` itself.
//
// It also blocks the app's service worker (M10.SH1, src/service-worker.ts)
// for every spec: the `serviceWorkers` option defaults to 'block' here, and
// `installMotionOff` blocks registration in a hand-made context the same way
// Playwright's own 'block' does. So no ordinary spec ever runs behind the
// worker's cache. shell.spec.ts, which tests the worker, sets
// `test.use({ serviceWorkers: 'allow' })`.

import { test as base, type BrowserContext } from '@playwright/test';

/** The Settings sheet's persisted preferences (web/src/lib/ui/settings.svelte.ts). */
const SETTINGS_KEY = 'ajayeb:settings';
/** A sessionStorage flag so the override applies at most once per tab:
 *  sessionStorage survives a reload of the same tab but not a new one, so a
 *  spec that changes a setting itself and reloads keeps its own choice,
 *  while a fresh tab still starts with Animation off. */
const GUARD_KEY = 'ajayeb:e2e-motion-off-applied';
/** The first-time tips' device state (web/src/lib/ui/tips.svelte.ts). */
const TIPS_KEY = 'ajayeb:tips';

/**
 * Runs in the page, before any app script, on every navigation. Merges
 * `animationSpeed: 'off'` and `showOpponentMove: false` into whatever JSON
 * is already under settingsKey (or starts fresh if there is none or it fails
 * to parse, the same way the app's own SettingsStore drops a value it cannot
 * read), then sets the guard so a later reload of this same tab leaves the
 * settings alone.
 */
function installMotionOffScript(opts: {
  settingsKey: string;
  guardKey: string;
  tipsKey: string;
}): void {
  const { settingsKey, guardKey, tipsKey } = opts;
  try {
    if (window.sessionStorage.getItem(guardKey)) return;
    window.sessionStorage.setItem(guardKey, '1');
    let current: Record<string, unknown> = {};
    try {
      const raw = window.localStorage.getItem(settingsKey);
      if (raw) current = JSON.parse(raw) as Record<string, unknown>;
    } catch {
      current = {};
    }
    current['animationSpeed'] = 'off';
    // The opponent move summary popup (M10.S03b) closes on any tap and
    // swallows it, which would eat a driver's next click after every move
    // of the other side. Specs about it (move-summary.spec.ts) turn it on.
    current['showOpponentMove'] = false;
    window.localStorage.setItem(settingsKey, JSON.stringify(current));
    // A first-time tip (M10.HS5) is a callout that swallows the next tap
    // elsewhere, like the summary. Specs about tips turn them back on.
    window.localStorage.setItem(tipsKey, JSON.stringify({ off: true, seen: [] }));
  } catch {
    // Storage may be unavailable; nothing to force, and nothing to crash on.
  }
}

async function addMotionOff(context: BrowserContext): Promise<void> {
  await context.addInitScript(installMotionOffScript, {
    settingsKey: SETTINGS_KEY,
    guardKey: GUARD_KEY,
    tipsKey: TIPS_KEY
  });
}

/** Runs in the page before any app script: service worker registration
 *  becomes a no-op, which is what Playwright's `serviceWorkers: 'block'`
 *  does too (without its console warning). */
function blockServiceWorkersScript(): void {
  if (navigator.serviceWorker) {
    navigator.serviceWorker.register = () => new Promise<ServiceWorkerRegistration>(() => {});
  }
}

/** Adds the init scripts to a context created outside the `context` fixture
 *  below (online.ts's extra browser tabs for a second player or a
 *  spectator): Animation off, and no service worker, which such a context
 *  would otherwise allow. Call it right after `newContext()`, before the
 *  first `page.goto`, the same way an init script must be added before it. */
export async function installMotionOff(context: BrowserContext): Promise<void> {
  await addMotionOff(context);
  await context.addInitScript(blockServiceWorkersScript);
}

/** Every spec imports `test` (and `test.describe`, `test.use`, and so on)
 *  from here instead of from '@playwright/test', so its default `context`,
 *  and therefore its default `page`, starts with Animation off. `expect`
 *  needs no override: specs keep importing it from '@playwright/test'. */
export const test = base.extend({
  // The default for every spec; a spec that tests the worker allows it.
  serviceWorkers: 'block',
  context: async ({ context }, use) => {
    await addMotionOff(context);
    await use(context);
  }
});
