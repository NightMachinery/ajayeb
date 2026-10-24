// The app-shell service worker (M10.SH1, web/src/service-worker.ts): it
// installs and controls the page, the fonts load from content-hashed URLs,
// install leaves out only the files the /dev pages alone load, the app opens
// offline from its cache with its own offline state, the server's endpoints
// never come from a cache, the stable names stay stable, and a deploy in
// mid-game (a new worker, the old build's files gone from the server)
// leaves an open table every file it may still import, from the old
// worker's cache, while the new worker waits. Every other
// spec blocks the worker (fixtures/base.ts); this one allows it. Each test
// gets a fresh context, so a fresh registration.

import { existsSync, readFileSync } from 'node:fs';
import { expect, type Page } from '@playwright/test';
import { devOnlyFiles, type Manifest } from '../../web/src/lib/sw/dev-only';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';
import { openHotseatRoom } from '../fixtures/scenario';
import { openBotTable } from '../fixtures/table';

test.use({ serviceWorkers: 'allow' });

/** Loads Home, waits for the worker to be active, and reloads once so the
 *  page is certainly loaded through it. */
async function installWorker(page: Page): Promise<{ first: number; repeat: number }> {
  await page.goto('/');
  await expect(page.getByRole('heading', { level: 1, name: 'Ajayeb' })).toBeVisible();
  await page.evaluate(() => document.fonts.ready.then(() => undefined));
  const transfer = () =>
    page.evaluate(() =>
      [
        ...performance.getEntriesByType('navigation'),
        ...performance.getEntriesByType('resource')
      ].reduce((sum, entry) => sum + (entry as PerformanceResourceTiming).transferSize, 0)
    );
  await page.evaluate(async () => {
    await navigator.serviceWorker.ready;
  });
  const first = await transfer();
  await page.reload();
  await expect
    .poll(() => page.evaluate(() => navigator.serviceWorker.controller?.scriptURL ?? null))
    .toMatch(/\/service-worker\.js$/);
  await page.evaluate(() => document.fonts.ready.then(() => undefined));
  return { first, repeat: await transfer() };
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

const HASHED_FONT = /\/_app\/immutable\/assets\/(inter|fraunces|vazirmatn)-\d00\.[\w-]+\.woff2$/;

const WEB = new URL('../../web/', import.meta.url);

/** The served build's files (the worker's `build` list) and those only the
 *  /dev pages load, from the client manifest SvelteKit leaves beside the
 *  build, the same way vite.config.ts computes the list for the worker. */
function clientBuild(): { files: string[]; devOnly: Set<string> } {
  const at = (path: string) => new URL(path, WEB);
  const manifest = JSON.parse(
    readFileSync(at('.svelte-kit/output/client/.vite/manifest.json'), 'utf8')
  ) as Manifest;
  const files = [
    ...new Set(
      Object.values(manifest).flatMap((chunk) => [
        chunk.file,
        ...(chunk.css ?? []),
        ...(chunk.assets ?? [])
      ])
    )
  ].sort();
  // Another build's manifest names hashed files this build does not have.
  expect(
    files.filter((file) => !existsSync(at(`build/${file}`))),
    'the client manifest belongs to the served build; rebuild web'
  ).toEqual([]);
  const devOnly = devOnlyFiles(manifest, (key) => readFileSync(at(key), 'utf8'));
  return { files, devOnly: new Set(devOnly.map((entry) => entry.file)) };
}

test.describe('app shell', { tag: '@mobile' }, () => {
  test('the worker installs, controls the page after one reload and serves the fonts by hashed URL', async ({
    page
  }, testInfo) => {
    const problems = watchHygiene(page);
    const fonts: { url: string; fromWorker: boolean }[] = [];
    page.on('response', (res) => {
      if (res.url().endsWith('.woff2')) {
        fonts.push({ url: res.url(), fromWorker: res.fromServiceWorker() });
      }
    });

    const transfer = await installWorker(page);
    const precache = await page.evaluate(async () => {
      const name = (await caches.keys()).find((key) => key.startsWith('ajayeb-shell-'));
      if (!name) throw new Error('shell cache missing');
      const cache = await caches.open(name);
      const keys = await cache.keys();
      let bytes = 0;
      for (const key of keys) bytes += (await (await cache.match(key))!.arrayBuffer()).byteLength;
      return { files: keys.length, bytes };
    });
    const measurement = { project: testInfo.project.name, precache, pageTransferBytes: transfer };
    console.log('offline-shell measurement', JSON.stringify(measurement));
    await testInfo.attach('shell-measurement', {
      body: JSON.stringify(measurement),
      contentType: 'application/json'
    });
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
    const cachePaths = await cachedPaths(page);
    const shell = Object.entries(cachePaths).find(([name]) => name.startsWith('ajayeb-shell-'));
    expect(shell, 'a versioned shell cache').toBeDefined();
    const keys = shell?.[1] ?? [];
    for (const path of [
      '/shell.html',
      '/favicon.svg',
      '/manifest.webmanifest',
      '/icons/icon-192.png'
    ]) {
      expect(keys).toContain(path);
    }
    // Inter and Fraunces (four files) plus Vazirmatn 400, 600 and 700.
    expect(keys.filter((k) => HASHED_FONT.test(k))).toHaveLength(7);
    expect(keys.some((k) => k.startsWith('/_app/immutable/entry/'))).toBe(true);
    expect(keys.filter((k) => k.startsWith('/art/') || k.startsWith('/fonts/'))).toEqual([]);

    expect(problems, 'console errors, page errors or failed requests').toEqual([]);
  });

  test('install leaves out the files only the /dev pages load, and nothing a real route loads', async ({
    page
  }) => {
    const build = clientBuild();
    await installWorker(page);
    const shell = Object.entries(await cachedPaths(page)).find(([name]) =>
      name.startsWith('ajayeb-shell-')
    );
    const precached = new Set(shell?.[1] ?? []);
    console.log(
      'precache-plan',
      JSON.stringify({
        buildFiles: build.files.length,
        devOnly: build.devOnly.size,
        precached: precached.size
      })
    );

    // Every build file a real route can load, statically or lazily, is
    // installed, including those a dev page shares.
    expect(
      build.files.filter((file) => !build.devOnly.has(file) && !precached.has(`/${file}`)),
      'build files a real route loads that install left out'
    ).toEqual([]);
    // No file only the dev pages load is.
    expect(
      [...build.devOnly].filter((file) => precached.has(`/${file}`)),
      'files only the /dev pages load, installed anyway'
    ).toEqual([]);

    // Independently of the manifest: every build file the real routes load
    // on a full page load comes from the install precache.
    const loaded = new Set<string>();
    page.on('response', (res) => {
      const path = new URL(res.url()).pathname;
      if (path.startsWith('/_app/immutable/')) loaded.add(path);
    });
    for (const path of ['/', '/rules', '/settings']) {
      await page.goto(path);
      await expect(page.locator('h1')).toBeVisible();
    }
    // A dev page opens the room; its own load is the one measured.
    await openHotseatRoom(page, null);
    const room = new URL(page.url()).pathname;
    loaded.clear();
    await page.goto(room);
    await page.waitForFunction(() => window.__ajayeb?.ready() === true);
    await page.waitForLoadState('networkidle');
    expect(loaded.size, 'the room page loads build files').toBeGreaterThan(0);
    const real = [...loaded];
    expect(
      real.filter((path) => !precached.has(path)),
      'build files a real route loaded that install left out'
    ).toEqual([]);

    // The gallery's own route nodes are left out, and the page still opens
    // through the worker, which fetches them from the network.
    loaded.clear();
    await page.goto('/dev/gallery');
    await expect(page.getByTestId('gallery-viewport')).toBeVisible();
    const left = [...loaded].filter(
      (path) => path.startsWith('/_app/immutable/nodes/') && !precached.has(path)
    );
    expect(left.length, 'the dev layout and the gallery page nodes').toBeGreaterThanOrEqual(2);
    for (const path of left) expect(build.devOnly.has(path.slice(1)), path).toBe(true);
  });

  test('a deploy in mid-game leaves the open table every lazy import it may need', async ({
    page,
    context
  }) => {
    test.setTimeout(120_000);
    const build = clientBuild();
    await installWorker(page);
    const old = context.serviceWorkers()[0]!;
    await openBotTable(page, 3, 'Ana');
    await expect
      .poll(() => page.evaluate(() => navigator.serviceWorker.controller?.state ?? null))
      .toBe('activated');

    // The deploy: the server now has another build, with a new version,
    // and none of this build's files any more, for the page or the worker
    // that installed it. Only the next worker fetches what it installs.
    const reached: string[] = [];
    await context.route('**/*', async (route) => {
      const request = route.request();
      const path = new URL(request.url()).pathname;
      const worker = request.serviceWorker();
      if (path === '/_app/version.json')
        return route.fulfill({ contentType: 'application/json', body: '{"version":"next"}' });
      if (path.startsWith('/_app/') && (worker === null || worker === old)) {
        reached.push(`${worker ? 'worker' : 'page'} ${path}`);
        return route.fulfill({ status: 404, body: 'gone' });
      }
      return route.fallback();
    });
    const loaded: string[] = [];
    page.on('response', (res) => {
      const path = new URL(res.url()).pathname;
      if (path.startsWith('/_app/immutable/') && res.ok() && res.fromServiceWorker())
        loaded.push(path);
    });

    // The control: a file this build never had misses the cache, and the
    // worker's own fetch meets the server that no longer has the old build.
    const probe = '/_app/immutable/chunks/not-in-this-build.js';
    expect(await page.evaluate(async (p) => (await fetch(p)).status, probe)).toBe(404);
    expect(reached).toEqual([`worker ${probe}`]);
    reached.length = 0;

    // The next worker installs and waits: the open game page holds it back,
    // so the old one, and its cache, stay. (The browser fetches a worker
    // script outside the test's routing, so the next worker is this build's
    // script under a new URL, which the browser installs as a new version.)
    await page.evaluate(async () => {
      await navigator.serviceWorker.register('/service-worker.js?build=next');
    });
    await expect
      .poll(() =>
        page.evaluate(
          async () => (await navigator.serviceWorker.getRegistration())?.waiting?.state ?? null
        )
      )
      .toBe('installed');
    expect(
      await page.evaluate(
        async () => (await navigator.serviceWorker.getRegistration())?.waiting?.scriptURL ?? null
      )
    ).toMatch(/\/service-worker\.js\?build=next$/);

    // A lazy import the table makes on demand: the rules overlay.
    loaded.length = 0;
    await page.locator('[data-act="table-menu"]').click();
    await page.locator('[data-act="rules"]').click();
    const overlay = page.locator('[data-rules-overlay]');
    await expect(overlay.locator('[data-rules-panel="haftgane"]')).toBeVisible();
    expect(loaded.length, 'the overlay loads its own files').toBeGreaterThan(0);
    await overlay.locator('[data-control="rules-close"]').click();

    // And every file of the build a real route could still import.
    const missing = await page.evaluate(
      async (files) => {
        const out: string[] = [];
        for (const file of files) {
          const res = await fetch(`/${file}`);
          if (!res.ok) out.push(`${file} ${res.status}`);
        }
        return out;
      },
      build.files.filter((file) => !build.devOnly.has(file))
    );
    expect(missing, 'build files the open page could not load after the deploy').toEqual([]);
    expect(reached, 'requests for the old build that reached the server').toEqual([]);
    expect(
      await page.evaluate(() => navigator.serviceWorker.controller?.scriptURL ?? null)
    ).toMatch(/\/service-worker\.js$/);
    await expect(page.locator('[data-table-screen]')).toBeVisible();
    await context.unrouteAll({ behavior: 'ignoreErrors' });
  });

  test('offline, Home and a room page open from the cache with their offline state', async ({
    page,
    context
  }, testInfo) => {
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
    await page.setViewportSize(
      testInfo.project.name === 'mobile'
        ? { width: 360, height: 740 }
        : { width: 1280, height: 800 }
    );
    for (const theme of ['light', 'dark']) {
      await page.evaluate((value) => localStorage.setItem('ajayeb:theme', value), theme);
      await page.reload();
      await expect(page.locator('html')).toHaveAttribute('data-theme', theme);
      await expect(page.locator('[data-offline-notice]')).toBeVisible();
      // A lazy image's decode() can stay pending while offline. Give
      // already cached artwork time to paint without waiting indefinitely.
      await page.evaluate(async () => {
        await Promise.race([
          (async () => {
            await document.fonts.ready;
            await Promise.all([...document.images].map((image) => image.decode().catch(() => {})));
            await new Promise((resolve) =>
              requestAnimationFrame(() => requestAnimationFrame(resolve))
            );
          })(),
          new Promise((resolve) => setTimeout(resolve, 1000))
        ]);
      });
      await page.screenshot({ path: testInfo.outputPath(`home-offline-${theme}.png`) });
    }

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
