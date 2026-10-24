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
//   the content-hashed fonts) except those only the /dev pages load, the
//   stable-named statics svelte.config.js lets through (favicon, manifest,
//   app icons) and the generic shell, shell.html.
// - Page loads go network first. The cached shell answers only when the
//   network fails or takes longer than NAVIGATION_TIMEOUT_MS; a network response is never replaced by it.
// - Art is cached as it is shown, by its ?v= URL, in a bounded cache that
//   outlives versions. It is never precached: the app's idle prefetch
//   (lib/art/prefetch.ts) already warms it (D24).
// - Sounds use their own bounded runtime cache by ?v= URL. Only an unlocked
//   player fetches them, so installing the worker never downloads audio.
// - /ws, /api/* (and /api/dev/*), /health and everything else are left to
//   the browser untouched.
// - Updates: a new version takes over on a full page load with no game open,
//   because a page mid-game keeps loading the code it started with. Then it
//   waits, and the next full page load that no other open game page holds
//   back hands over to it (updateHeldBack in the policy). Game pages say
//   so by holding a shared Web Lock (lib/sw/game-lock.ts).

import { base, build, files, version } from '$service-worker';
import { networkFirst } from '$lib/sw/navigation';
import { artCache } from '$lib/sw/art-cache';
import { DEV_ONLY_MARK } from '$lib/sw/dev-only';
import {
  ART_CACHE,
  ART_CACHE_MAX_ENTRIES,
  SOUND_CACHE,
  SOUND_CACHE_MAX_ENTRIES,
  NAVIGATION_TIMEOUT_MS,
  SHELL_KEY,
  SKIP_WAITING,
  gameWindows,
  shellCacheName,
  staleCaches,
  strategyFor,
  updateHeldBack,
  type OpenWindow
} from '$lib/sw/policy';

const sw = self as unknown as ServiceWorkerGlobalScope;
const SHELL_CACHE = shellCacheName(version);
const PRECACHED = new Set(files);

// The build files only the /dev pages load (lib/sw/dev-only.ts) stay out of
// install; a dev page fetches them from the network. vite.config.ts writes
// their list, as JSON, over the mark in the built worker. Under `vite dev`
// the build list is empty, so there is nothing to leave out.
const DEV_ONLY = new Set(
  (import.meta.env.DEV ? [] : (JSON.parse(DEV_ONLY_MARK) as string[])).map(
    (file) => `${base}/${file}`
  )
);
const INSTALL = build.filter((path) => !DEV_ONLY.has(path));

/** The app's open windows, controlled by any version or none. When locks
 *  cannot be queried, every open window holds updates back: its load URL
 *  cannot prove that client-side navigation has not opened a game. */
async function openWindows(): Promise<OpenWindow[]> {
  const windows = await sw.clients.matchAll({ type: 'window', includeUncontrolled: true });
  let snapshot: LockManagerSnapshot | undefined;
  try {
    snapshot = await sw.navigator.locks.query();
  } catch {
    // No Web Locks: use the browser's normal close-all-tabs lifecycle.
  }
  return gameWindows(
    windows.map((client) => client.id),
    snapshot
  );
}

sw.addEventListener('install', (event) => {
  event.waitUntil(
    (async () => {
      const cache = await caches.open(SHELL_CACHE);
      // Hashed build files may come from the HTTP cache; the stable names
      // and the shell are revalidated, so they match this version.
      await cache.addAll(INSTALL);
      await cache.addAll(files.map((path) => new Request(path, { cache: 'no-cache' })));
      await cache.add(new Request(SHELL_KEY, { cache: 'no-cache' }));
      // First installs activate normally. Updates wait for a full load.
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
    event.waitUntil(
      (async () => {
        if (!updateHeldBack(await openWindows())) await sw.skipWaiting();
      })()
    );
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
      event.respondWith(runtime(event, runtimeArt));
      break;
    case 'sound':
      event.respondWith(runtime(event, runtimeSound));
      break;
    case 'bypass':
      break;
  }
});

/** A waiting new version takes over on a page load no other open game page
 *  holds back. This page itself loads from the network either way. */
async function handOver(): Promise<void> {
  const waiting = sw.registration.waiting;
  if (!waiting || updateHeldBack(await openWindows())) return;
  waiting.postMessage({ type: SKIP_WAITING });
}

async function cachedShell(): Promise<Response | undefined> {
  const cache = await caches.open(SHELL_CACHE);
  return cache.match(SHELL_KEY);
}

function navigate(request: Request): Promise<Response> {
  return networkFirst(() => fetch(request), cachedShell, NAVIGATION_TIMEOUT_MS);
}

async function fromShellCache(request: Request): Promise<Response> {
  const cache = await caches.open(SHELL_CACHE);
  const hit = await cache.match(request, { ignoreVary: true });
  return hit ?? fetch(request);
}

// Reads share the LRU transaction queue, so a concurrent offline request
// cannot observe the temporary deletion used to move a hit to the end.
const runtimeArt = artCache(() => caches.open(ART_CACHE), ART_CACHE_MAX_ENTRIES);

const runtimeSound = artCache(() => caches.open(SOUND_CACHE), SOUND_CACHE_MAX_ENTRIES);

async function runtime(event: FetchEvent, cache: ReturnType<typeof artCache>): Promise<Response> {
  const request = event.request;
  const hit = await cache.match(request);
  if (hit) return hit;
  const response = await fetch(request);
  if (response.status === 200) event.waitUntil(cache.remember(request, response.clone()));
  return response;
}
