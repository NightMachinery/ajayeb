// The service worker's cache policy, as pure functions so vitest can check
// them without a worker (policy.test.ts). src/service-worker.ts applies it.
//
// What the worker does with a request, by strategy:
//   - bypass: nothing. It never calls respondWith, so the browser goes to
//     the network exactly as it would with no worker. The socket (/ws), the
//     HTTP API (/api/*, /api/dev/* included), the health check, the worker
//     script itself, other origins, non-GET requests and range requests.
//   - navigate: an HTML page load. Network first; the cached shell
//     (shell.html) only when the network fails or times out.
//   - immutable: a content-hashed build file under /_app/immutable/. Cache
//     first from this version's precache, then the network.
//   - precached: a stable-named static file the worker precached (the
//     favicon, the web manifest, the app icons). Cache first, then the
//     network; a new version precaches them again.
//   - art: bitmap art requested with its ?v= content hash. Cache first from
//     a bounded runtime cache that outlives versions, then the network.

/** The prefix of the versioned shell cache; the version follows it. */
export const SHELL_CACHE_PREFIX = 'ajayeb-shell-';
/** The runtime art cache. It is not versioned: art URLs carry their hash. */
export const ART_CACHE = 'ajayeb-art';
/** At most this many art responses stay cached (the oldest-used go first). */
export const ART_CACHE_MAX_ENTRIES = 150;
/** How long a page load waits for the network before the cached shell. */
export const NAVIGATION_TIMEOUT_MS = 4000;
/** The generic client shell, independent of prerendered Home. Offline game
 *  deep links must not receive Home's HTML or hydration data. */
export const SHELL_KEY = '/shell.html';
/** The message an old worker sends a waiting one to make it take over. */
export const SKIP_WAITING = 'ajayeb:skip-waiting';

export type Strategy = 'bypass' | 'navigate' | 'immutable' | 'precached' | 'art';

export interface RequestFacts {
  method: string;
  url: string;
  /** Request.mode: 'navigate' for a page load. */
  mode: string;
  /** Whether the request carries a Range header (audio and video do). */
  range: boolean;
}

/** The server's own endpoints, which the worker never answers. */
function isServerEndpoint(path: string): boolean {
  return (
    path === '/ws' ||
    path.startsWith('/ws/') ||
    path === '/api' ||
    path.startsWith('/api/') ||
    path === '/health' ||
    path === '/service-worker.js'
  );
}

/** A path the client router renders (/, /play/<code>, /rules, ...): not a
 *  build file, not art and no file name with an extension. */
function isAppRoute(path: string): boolean {
  if (path.startsWith('/_app/') || path.startsWith('/art/')) return false;
  const last = path.slice(path.lastIndexOf('/') + 1);
  return !last.includes('.');
}

/**
 * Which strategy the worker uses for a request. `origin` is the worker's
 * own origin, and `precached` holds the paths of the stable-named static
 * files it precached.
 */
export function strategyFor(
  req: RequestFacts,
  origin: string,
  precached: ReadonlySet<string>
): Strategy {
  if (req.method !== 'GET' || req.range) return 'bypass';
  let url: URL;
  try {
    url = new URL(req.url);
  } catch {
    return 'bypass';
  }
  if (url.origin !== origin) return 'bypass';
  const path = url.pathname;
  if (isServerEndpoint(path)) return 'bypass';
  // A page load of an app route. Opening a file in a tab (an image, the
  // manifest) is left to the browser: the shell is no answer to it.
  if (req.mode === 'navigate') return isAppRoute(path) ? 'navigate' : 'bypass';
  if (path.startsWith('/_app/immutable/')) return 'immutable';
  if (path.startsWith('/art/') && url.searchParams.get('v')) return 'art';
  if (url.search === '' && precached.has(path)) return 'precached';
  return 'bypass';
}

/** The name of a version's shell cache. */
export function shellCacheName(version: string): string {
  return SHELL_CACHE_PREFIX + version;
}

/** The caches a newly active worker deletes: older shell caches, never the
 *  art cache and never a cache of another app on the same origin. */
export function staleCaches(names: readonly string[], current: string): string[] {
  return names.filter((name) => name.startsWith(SHELL_CACHE_PREFIX) && name !== current);
}

/** The Web Lock every game page holds, shared, while it shows a game
 *  (game-lock.ts). The worker reads who holds it to know which open
 *  windows are mid-game: a window's URL as the worker sees it (Client.url)
 *  is the URL the page was loaded at, and the client router moves on from
 *  there without telling the worker. */
export const GAME_LOCK = 'ajayeb:game-page';

/** A path mid-game: /play/<code>. */
export function isGamePath(path: string): boolean {
  return /^\/play\/[^/]/.test(path);
}

/** The ids of the clients holding the game lock, from LockManager.query(). */
export function gameClientIds(
  held: readonly { name?: string | undefined; clientId?: string | undefined }[] | undefined
): Set<string> | null {
  if (!held) return null;
  const ids = new Set<string>();
  for (const lock of held) {
    if (lock.name !== GAME_LOCK) continue;
    // A game lock without an owner cannot prove which window is safe.
    if (!lock.clientId) return null;
    ids.add(lock.clientId);
  }
  return ids;
}

/** An open window of the app, as the worker sees it. */
export interface OpenWindow {
  /** Whether it shows a game (it holds the game lock). */
  game: boolean;
}

/** Infer current game windows from their locks, never their load URLs.
 *  Missing query data or an unidentified lock makes every window unknown,
 *  which holds an update back until the browser's normal activation. */
export function gameWindows(
  clientIds: readonly string[],
  snapshot:
    | {
        held?: readonly { name?: string | undefined; clientId?: string | undefined }[];
        pending?: readonly { name?: string | undefined; clientId?: string | undefined }[];
      }
    | undefined
): OpenWindow[] {
  // Pending shared locks are game intent too: request() is asynchronous.
  const games =
    snapshot?.held && snapshot.pending
      ? gameClientIds([...snapshot.held, ...snapshot.pending])
      : null;
  // Web Locks owner IDs and Service Worker Client IDs need not match.
  // An unmatched owner cannot identify a safe window, so hold every one.
  const unknownOwner = games === null || [...games].some((owner) => !clientIds.includes(owner));
  return clientIds.map((id) => ({ game: unknownOwner || !id || games?.has(id) === true }));
}

/** Any open game page holds a waiting version back. Never guess which
 *  window a navigation replaces: keeping an update waiting is safer than
 *  changing the chunks underneath a second game tab. */
export function updateHeldBack(open: readonly OpenWindow[]): boolean {
  return open.some((w) => w.game);
}

/**
 * The art cache's least recently used keys to delete so that at most `max`
 * remain. `keys` is in Cache.keys() order, which is insertion order; the
 * worker re-puts an entry on every hit, which moves it to the end.
 */
export function overflow<T>(keys: readonly T[], max: number): T[] {
  return keys.length > max ? keys.slice(0, keys.length - max) : [];
}
