/// <reference no-default-lib="true"/>
/// <reference lib="esnext" />
/// <reference lib="webworker" />
/// <reference types="@sveltejs/kit" />

// The app-shell service worker, served as /service-worker.js (a stable name;
// SvelteKit builds and registers it). It lets the app open without a
// network: playing itself needs the server, and the page then shows its own
// offline state. The cache policy by URL is lib/sw/policy.ts; web/README.md
// ("App shell") explains it.
//
// - Install precaches this version: every _app/immutable file (code, CSS,
//   the content-hashed fonts), the stable-named statics svelte.config.js
//   lets through (favicon, manifest, app icons) and the shell, index.html.
// - Page loads go network first. The cached shell answers only when the
//   network fails, takes longer than NAVIGATION_TIMEOUT_MS or answers with a
//   gateway error; a page the network answers is never replaced by it.
// - Art is cached as it is shown, by its ?v= URL, in a bounded cache that
//   outlives versions. It is never precached: the app's idle prefetch
//   (lib/art/prefetch.ts) already warms it (D24).
// - /ws, /api/* (and /api/dev/*), /health and everything else are left to
//   the browser untouched.
// - Updates: a new version takes over at once unless a game page is open,
//   because a page mid-game keeps loading the code it started with. Then it
//   waits, and the next full page load that no other open game page holds
//   back hands over to it (updateHeldBack in the policy). Game pages say
//   so by holding a shared Web Lock (lib/sw/game-lock.ts).

import { build, files, version } from '$service-worker';
import {
  ART_CACHE,
  ART_CACHE_MAX_ENTRIES,
  NAVIGATION_TIMEOUT_MS,
  SHELL_KEY,
  SKIP_WAITING,
  gameClientIds,
  isGamePage,
  isGatewayError,
  overflow,
  shellCacheName,
  staleCaches,
  strategyFor,
  updateHeldBack,
  type OpenWindow
} from '$lib/sw/policy';

const sw = self as unknown as ServiceWorkerGlobalScope;
const SHELL_CACHE = shellCacheName(version);
const PRECACHED = new Set(files);

/** The app's open windows, controlled by any version or none, and which of
 *  them show a game: those holding the game lock, or, where the worker
 *  cannot read locks, those loaded at a game URL. */
async function openWindows(): Promise<OpenWindow[]> {
  const windows = await sw.clients.matchAll({ type: 'window', includeUncontrolled: true });
  let games: Set<string> | null = null;
  try {
    const { held } = await sw.navigator.locks.query();
    games = gameClientIds(held ?? []);
  } catch {
    // No Web Locks here: fall back to the URL each window was loaded at.
  }
  return windows.map((client) => ({
    focused: client.focused,
    game: games ? games.has(client.id) : isGamePage(client.url)
  }));
}

sw.addEventListener('install', (event) => {
  event.waitUntil(
    (async () => {
      const cache = await caches.open(SHELL_CACHE);
      // Hashed build files may come from the HTTP cache; the stable names
      // and the shell are revalidated, so they match this version.
      await cache.addAll(build);
      await cache.addAll(files.map((path) => new Request(path, { cache: 'no-cache' })));
      await cache.add(new Request(SHELL_KEY, { cache: 'no-cache' }));
      if (!updateHeldBack(await openWindows(), false)) await sw.skipWaiting();
    })()
  );
});

sw.addEventListener('activate', (event) => {
  event.waitUntil(
    (async () => {
      for (const name of staleCaches(await caches.keys(), SHELL_CACHE)) {
        await caches.delete(name);
      }
      // Pages opened before the first install are served from now on too.
      await sw.clients.claim();
    })()
  );
});

sw.addEventListener('message', (event) => {
  const data: unknown = event.data;
  if (typeof data === 'object' && data !== null && 'type' in data && data.type === SKIP_WAITING) {
    void sw.skipWaiting();
  }
});

sw.addEventListener('fetch', (event) => {
  const request = event.request;
  const strategy = strategyFor(
    {
      method: request.method,
      url: request.url,
      mode: request.mode,
      range: request.headers.has('range')
    },
    sw.location.origin,
    PRECACHED
  );
  switch (strategy) {
    case 'navigate':
      event.waitUntil(handOver());
      event.respondWith(navigate(request));
      break;
    case 'immutable':
    case 'precached':
      event.respondWith(fromShellCache(request));
      break;
    case 'art':
      event.respondWith(art(event));
      break;
    case 'bypass':
      break;
  }
});

/** A waiting new version takes over on a page load no other open game page
 *  holds back. This page itself loads from the network either way. */
async function handOver(): Promise<void> {
  const waiting = sw.registration.waiting;
  if (!waiting || updateHeldBack(await openWindows(), true)) return;
  waiting.postMessage({ type: SKIP_WAITING });
}

async function cachedShell(): Promise<Response | undefined> {
  const cache = await caches.open(SHELL_CACHE);
  return cache.match(SHELL_KEY);
}

async function navigate(request: Request): Promise<Response> {
  const network = fetch(request);
  // If the timeout wins, a later failure must not surface as unhandled.
  network.catch(() => undefined);
  let timer: ReturnType<typeof setTimeout> | undefined;
  const timeout = new Promise<'timeout'>((resolve) => {
    timer = setTimeout(() => resolve('timeout'), NAVIGATION_TIMEOUT_MS);
  });
  try {
    const first = await Promise.race([network, timeout]);
    if (first !== 'timeout' && !isGatewayError(first.status)) return first;
  } catch {
    // Offline or refused: the cached shell below.
  } finally {
    clearTimeout(timer);
  }
  const shell = await cachedShell();
  // No shell yet (the very first install is still running): the network's
  // own answer, or its error, exactly as without a worker.
  return shell ?? network;
}

async function fromShellCache(request: Request): Promise<Response> {
  const cache = await caches.open(SHELL_CACHE);
  const hit = await cache.match(request, { ignoreVary: true });
  return hit ?? fetch(request);
}

async function art(event: FetchEvent): Promise<Response> {
  const request = event.request;
  const cache = await caches.open(ART_CACHE);
  const hit = await cache.match(request, { ignoreVary: true });
  if (hit) {
    // Putting it again moves it to the end of the cache's key order, which
    // is what makes the trim below drop the least recently used first.
    event.waitUntil(cache.put(request, hit.clone()).catch(() => undefined));
    return hit;
  }
  const response = await fetch(request);
  if (response.status === 200) {
    event.waitUntil(
      cache
        .put(request, response.clone())
        .then(async () => {
          for (const key of overflow(await cache.keys(), ART_CACHE_MAX_ENTRIES)) {
            await cache.delete(key);
          }
        })
        .catch(() => undefined)
    );
  }
  return response;
}
