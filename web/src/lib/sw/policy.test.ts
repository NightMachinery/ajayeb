import { describe, expect, it } from 'vitest';
import {
  ART_CACHE,
  SHELL_KEY,
  GAME_LOCK,
  gameClientIds,
  gameWindows,
  isGamePath,
  overflow,
  shellCacheName,
  staleCaches,
  strategyFor,
  updateHeldBack,
  type RequestFacts
} from './policy';

const ORIGIN = 'https://ajayeb.example';
const PRECACHED = new Set(['/favicon.svg', '/manifest.webmanifest', '/icons/icon-192.png']);

function req(path: string, over: Partial<RequestFacts> = {}): RequestFacts {
  return { method: 'GET', url: ORIGIN + path, mode: 'cors', range: false, ...over };
}

function strategy(path: string, over: Partial<RequestFacts> = {}) {
  return strategyFor(req(path, over), ORIGIN, PRECACHED);
}

describe('strategyFor', () => {
  it('never touches the server endpoints, page load or not', () => {
    for (const path of [
      '/ws',
      '/api',
      '/api/rooms',
      '/api/dev/rooms/ABCD',
      '/health',
      '/service-worker.js'
    ]) {
      expect(strategy(path), path).toBe('bypass');
      expect(strategy(path, { mode: 'navigate' }), path).toBe('bypass');
    }
  });

  it('does not mistake a path that only starts like an endpoint', () => {
    expect(strategy('/apiary', { mode: 'navigate' })).toBe('navigate');
    expect(strategy('/wsx', { mode: 'navigate' })).toBe('navigate');
  });

  it('leaves other origins, non-GET and range requests alone', () => {
    expect(strategyFor(req('/'), 'https://other.example', PRECACHED)).toBe('bypass');
    expect(strategy('/_app/immutable/x.js', { method: 'POST' })).toBe('bypass');
    expect(strategy('/art/cards/a.avif?v=ab12', { range: true })).toBe('bypass');
    expect(strategyFor({ ...req('/'), url: 'not a url' }, ORIGIN, PRECACHED)).toBe('bypass');
  });

  it('handles page loads of app routes, not of files', () => {
    for (const path of ['/', '/play/ABCD', '/rules', '/settings', '/dev/gallery']) {
      expect(strategy(path, { mode: 'navigate' }), path).toBe('navigate');
    }
    for (const path of ['/favicon.svg', '/art/cards/a.avif?v=1', '/_app/version.json']) {
      expect(strategy(path, { mode: 'navigate' }), path).toBe('bypass');
    }
  });

  it('serves build files and precached statics from the cache', () => {
    expect(strategy('/_app/immutable/assets/inter-400.Ab12Cd34.woff2')).toBe('immutable');
    expect(strategy('/_app/immutable/entry/start.x.js')).toBe('immutable');
    expect(strategy('/favicon.svg')).toBe('precached');
    expect(strategy('/manifest.webmanifest')).toBe('precached');
    expect(strategy('/icons/icon-192.png')).toBe('precached');
  });

  it('leaves the version file and unlisted statics to the network', () => {
    expect(strategy('/_app/version.json')).toBe('bypass');
    expect(strategy('/fonts/OFL-Inter.txt')).toBe('bypass');
    expect(strategy('/favicon.svg?x=1')).toBe('bypass');
  });

  it('caches sounds only on demand by content hash, leaving ranges and navigation alone', () => {
    expect(strategy('/sfx/coin.m4a?v=0123abcd')).toBe('sound');
    expect(strategy('/sfx/coin.m4a')).toBe('bypass');
    expect(strategy('/sfx/coin.m4a?v=')).toBe('bypass');
    expect(strategy('/sfx/coin.m4a?v=0123abcd', { range: true })).toBe('bypass');
    expect(strategy('/sfx/coin.m4a?v=0123abcd', { mode: 'navigate' })).toBe('bypass');
  });

  it('caches art only by its content-hashed URL', () => {
    expect(strategy('/art/cards/baths.avif?v=0123abcd')).toBe('art');
    expect(strategy('/art/cards/baths.avif')).toBe('bypass');
    expect(strategy('/art/cards/baths.avif?v=')).toBe('bypass');
  });
});

describe('caches', () => {
  it('deletes only older shell caches', () => {
    const current = shellCacheName('200');
    expect(
      staleCaches([shellCacheName('100'), current, ART_CACHE, 'someone-else'], current)
    ).toEqual([shellCacheName('100')]);
  });

  it('trims the art cache to its newest entries', () => {
    expect(overflow(['a', 'b', 'c', 'd'], 2)).toEqual(['a', 'b']);
    expect(overflow(['a', 'b'], 2)).toEqual([]);
  });
});

describe('updates', () => {
  it('finds the game pages by their lock', () => {
    expect(
      gameClientIds([
        { name: GAME_LOCK, clientId: 'a' },
        { name: 'other', clientId: 'b' },
        { name: GAME_LOCK, clientId: 'c' }
      ])
    ).toEqual(new Set(['a', 'c']));
    expect(isGamePath('/play/ABCD')).toBe(true);
    expect(isGamePath('/')).toBe(false);
  });

  it('holds updates for SPA game entry even when the document loaded at Home', () => {
    // Client.url is the old Home URL after SPA entry. Only the game lock
    // says which of the two open windows now displays a game.
    const clients = [
      { id: 'game', loadUrl: '/' },
      { id: 'home', loadUrl: '/' }
    ];
    const windows = gameWindows(
      clients.map((client) => client.id),
      { held: [{ name: GAME_LOCK, clientId: 'game' }], pending: [] }
    );
    expect(windows).toEqual([{ game: true }, { game: false }]);
    expect(updateHeldBack(windows)).toBe(true);
  });

  it('holds updates if locks cannot be queried or a game lock has no client ID', () => {
    for (const held of [undefined, [{ name: GAME_LOCK }], [{ name: GAME_LOCK, clientId: '' }]]) {
      expect(gameClientIds(held)).toBeNull();
      expect(
        updateHeldBack(gameWindows(['loaded-at-home'], held ? { held, pending: [] } : undefined))
      ).toBe(true);
      expect(updateHeldBack(gameWindows([], held ? { held, pending: [] } : undefined))).toBe(false);
    }
    expect(updateHeldBack(gameWindows([''], { held: [], pending: [] }))).toBe(true);
    expect(updateHeldBack(gameWindows(['home'], { held: [], pending: [] }))).toBe(false);
  });

  it('treats pending game locks as intent, including unidentified owners', () => {
    const snapshot = { held: [], pending: [{ name: GAME_LOCK, clientId: 'spa-game' }] };
    expect(gameWindows(['spa-game', 'home'], snapshot)).toEqual([{ game: true }, { game: false }]);
    expect(updateHeldBack(gameWindows(['spa-game', 'home'], snapshot))).toBe(true);
    expect(
      updateHeldBack(gameWindows(['home'], { held: [], pending: [{ name: GAME_LOCK }] }))
    ).toBe(true);
    expect(updateHeldBack(gameWindows(['home'], { held: [] }))).toBe(true);
  });

  it('holds every window when lock owners cannot be matched to worker client IDs', () => {
    for (const snapshot of [
      { held: [{ name: GAME_LOCK, clientId: 'lock-owner' }], pending: [] },
      { held: [], pending: [{ name: GAME_LOCK, clientId: 'lock-owner' }] }
    ]) {
      // Browsers may expose different identifiers through the two APIs.
      const windows = gameWindows(['worker-window-a', 'worker-window-b'], snapshot);
      expect(windows).toEqual([{ game: true }, { game: true }]);
      expect(updateHeldBack(windows)).toBe(true);
    }
  });

  it('holds an update back while any game page is open', () => {
    expect(updateHeldBack([{ game: false }, { game: false }])).toBe(false);
    expect(updateHeldBack([{ game: false }, { game: true }])).toBe(true);
    expect(updateHeldBack([{ game: true }, { game: true }])).toBe(true);
    expect(updateHeldBack([])).toBe(false);
  });
});

describe('offline shell identity', () => {
  it('uses the generic client shell rather than prerendered Home', () => {
    expect(SHELL_KEY).toBe('/shell.html');
    expect(strategy(SHELL_KEY)).toBe('bypass');
    expect(strategy('/play/ABCDEF', { mode: 'navigate' })).toBe('navigate');
  });
});
