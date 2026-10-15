// The app-shell service worker (M10.SH1, web/src/service-worker.ts): it
// installs and controls the page, the fonts load from content-hashed URLs,
// the app opens offline from its cache with its own offline state, the
// server's endpoints never come from a cache, and the stable names stay
// stable. Every other spec blocks the worker (fixtures/base.ts); this one
// allows it. Each test gets a fresh context, so a fresh registration.

import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { openHotseatRoom } from '../fixtures/scenario';

test.use({ serviceWorkers: 'allow' });

/** Loads Home, waits for the worker to be active, and reloads once so the
 *  page is certainly loaded through it. */
async function installWorker(page: Page): Promise<void> {
  await page.goto('/');
  await page.evaluate(async () => {
    await navigator.serviceWorker.ready;
  });
  await page.reload();
  await expect
    .poll(() => page.evaluate(() => navigator.serviceWorker.controller?.scriptURL ?? null))
    .toMatch(/\/service-worker\.js$/);
}

/** Every key in every cache of the page's origin, as a path with its query. */
async function cachedPaths(page: Page): Promise<Record<string, string[]>> {
  return page.evaluate(async () => {
    const out: Record<string, string[]> = {};
    for (const name of await caches.keys()) {
      const cache = await caches.open(name);
      out[name] = (await cache.keys()).map((req) => {
        const url = new URL(req.url);
        return url.pathname + url.search;
      });
    }
    return out;
  });
}

const HASHED_FONT = /\/_app\/immutable\/assets\/(inter|fraunces)-\d00\.[\w-]+\.woff2$/;

test.describe('app shell', { tag: '@mobile' }, () => {
  test('the worker installs, controls the page after one reload and serves the fonts by hashed URL', async ({
    page
  }) => {
    const problems = watchHygiene(page);
    const fonts: { url: string; fromWorker: boolean }[] = [];
    page.on('response', (res) => {
      if (res.url().endsWith('.woff2')) {
        fonts.push({ url: res.url(), fromWorker: res.fromServiceWorker() });
      }
    });

    await installWorker(page);
    await expect(page.getByRole('heading', { level: 1, name: 'Ajayeb' })).toBeVisible();
    await page.evaluate(async () => {
      await document.fonts.ready;
    });

    expect(fonts.length, 'Home loads its fonts').toBeGreaterThan(0);
    for (const font of fonts) expect(font.url).toMatch(HASHED_FONT);
    // The reload's fonts came from the worker's precache.
    expect(fonts.some((f) => f.fromWorker)).toBe(true);
    const loaded = await page.evaluate(() =>
      [...document.fonts].filter((f) => f.status === 'loaded').map((f) => f.family)
    );
    expect(loaded.join(' ')).toMatch(/Inter/);
    expect(loaded.join(' ')).toMatch(/Fraunces/);

    // The precache: the build, the stable statics and the shell; no art.
    const caches = await cachedPaths(page);
    const shell = Object.entries(caches).find(([name]) => name.startsWith('ajayeb-shell-'));
    expect(shell, 'a versioned shell cache').toBeDefined();
    const keys = shell?.[1] ?? [];
    for (const path of ['/', '/favicon.svg', '/manifest.webmanifest', '/icons/icon-192.png']) {
      expect(keys).toContain(path);
    }
    expect(keys.filter((k) => HASHED_FONT.test(k))).toHaveLength(4);
    expect(keys.some((k) => k.startsWith('/_app/immutable/entry/'))).toBe(true);
    expect(keys.filter((k) => k.startsWith('/art/') || k.startsWith('/fonts/'))).toEqual([]);

    expect(problems, 'console errors, page errors or failed requests').toEqual([]);
  });

  test('offline, Home and a room page open from the cache with their offline state', async ({
    page,
    context
  }) => {
    // A room this browser played, opened while online.
    await installWorker(page);
    await openHotseatRoom(page, null);
    const roomPath = new URL(page.url()).pathname;
    await page.goto('/');
    await expect(page.locator('[data-offline-notice]')).toHaveCount(0);

    await context.setOffline(true);
    await page.reload();
    await expect(page.getByRole('heading', { level: 1, name: 'Ajayeb' })).toBeVisible();
    await expect(page.locator('[data-offline-notice]')).toBeVisible();
    await expect(page.locator('[data-offline-notice]')).toHaveText(/offline/i);

    // A full load of the room page: the shell from the cache, then the
    // session's own offline state where the game would be.
    await page.goto(roomPath);
    await expect(page.locator('[data-conn]')).toHaveAttribute('data-conn', 'offline');
    await expect(page.locator('[data-conn]')).toHaveText(/offline/i);
    // The game page has its own state; the shell's notice stays off there.
    await expect(page.locator('[data-offline-notice]')).toHaveCount(0);

    // Back online, the room plays again.
    await context.setOffline(false);
    await expect(page.locator('[data-conn]')).toHaveCount(0, { timeout: 20_000 });
  });

  test('the server endpoints are never answered from a cache', async ({ page, context }) => {
    await installWorker(page);
    const fromWorker: string[] = [];
    page.on('response', (res) => {
      const path = new URL(res.url()).pathname;
      if (/^\/(api|ws|health)(\/|$)/.test(path) && res.fromServiceWorker()) fromWorker.push(path);
    });

    // A room's socket opens through a controlled page.
    await openHotseatRoom(page, null);

    const online = await page.evaluate(async () => {
      const health = await fetch('/health');
      const scenarios = await fetch('/api/dev/scenarios');
      return [health.status, scenarios.status];
    });
    expect(online).toEqual([200, 200]);

    await context.setOffline(true);
    const offline = await page.evaluate(async () => {
      const tryFetch = (path: string) =>
        fetch(path).then(
          (res) => `answered ${res.status}`,
          () => 'failed'
        );
      return [await tryFetch('/health'), await tryFetch('/api/dev/scenarios')];
    });
    expect(offline, 'offline, nothing answers the endpoints').toEqual(['failed', 'failed']);
    await context.setOffline(false);

    expect(fromWorker).toEqual([]);
    const cached = Object.values(await cachedPaths(page)).flat();
    expect(cached.filter((path) => /^\/(api|ws|health)(\/|\?|$)/.test(path))).toEqual([]);
  });

  test('the worker, the manifest, the favicon and the icons keep their stable names', async ({
    page,
    request
  }) => {
    await installWorker(page);
    // Paths only: the page may hold these as absolute URLs.
    const head = await page.evaluate(() => {
      const path = (href: string | null | undefined) =>
        href ? new URL(href, location.href).pathname : null;
      const link = (rel: string) =>
        document.querySelector(`link[rel="${rel}"]`)?.getAttribute('href');
      return {
        icon: path(link('icon')),
        touch: path(link('apple-touch-icon')),
        manifest: path(link('manifest')),
        worker: path(navigator.serviceWorker.controller?.scriptURL)
      };
    });
    expect(head).toEqual({
      icon: '/favicon.svg',
      touch: '/icons/apple-touch-icon.png',
      manifest: '/manifest.webmanifest',
      worker: '/service-worker.js'
    });

    const manifest = (await (await request.get('/manifest.webmanifest')).json()) as {
      icons: { src: string }[];
    };
    expect(manifest.icons.map((i) => i.src)).toEqual([
      '/icons/icon-192.png',
      '/icons/icon-512.png'
    ]);
    for (const path of [
      '/service-worker.js',
      '/favicon.svg',
      '/icons/apple-touch-icon.png',
      '/icons/icon-192.png',
      '/icons/icon-512.png'
    ]) {
      const res = await request.get(path);
      expect(res.status(), path).toBe(200);
      // Revalidated on every load, never cached for long under a stable name.
      expect(res.headers()['cache-control'], path).toMatch(/no-cache/);
    }
  });
});
