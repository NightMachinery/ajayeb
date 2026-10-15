import { describe, expect, it } from 'vitest';
import {
  ART_CACHE,
  GAME_LOCK,
  gameClientIds,
  isGamePage,
  isGamePath,
  isGatewayError,
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
  it('knows a game page', () => {
    expect(isGamePage(ORIGIN + '/play/ABCD')).toBe(true);
    expect(isGamePage(ORIGIN + '/play/ABCD#claim=x')).toBe(true);
    expect(isGamePage(ORIGIN + '/play/')).toBe(false);
    expect(isGamePage(ORIGIN + '/')).toBe(false);
    expect(isGamePage(ORIGIN + '/rules')).toBe(false);
    expect(isGamePage('nonsense')).toBe(false);
  });

  it('finds the game pages by their lock', () => {
    expect(
      gameClientIds([
        { name: GAME_LOCK, clientId: 'a' },
        { name: 'other', clientId: 'b' },
        { name: GAME_LOCK, clientId: 'c' },
        { name: GAME_LOCK }
      ])
    ).toEqual(new Set(['a', 'c']));
    expect(isGamePath('/play/ABCD')).toBe(true);
    expect(isGamePath('/')).toBe(false);
  });

  const win = (game: boolean, focused = false) => ({ game, focused });

  it('holds a fresh install back while any game page is open', () => {
    expect(updateHeldBack([win(false), win(false, true)], false)).toBe(false);
    expect(updateHeldBack([win(false), win(true, true)], false)).toBe(true);
    expect(updateHeldBack([], false)).toBe(false);
  });

  it('lets a page load hand over unless another game page is open', () => {
    // A second tab loads Home while a game plays in the background.
    expect(updateHeldBack([win(true)], true)).toBe(true);
    // The game's own tab, focused, reloads or loads Home.
    expect(updateHeldBack([win(true, true)], true)).toBe(false);
    // Another game stays open in the background.
    expect(updateHeldBack([win(true, true), win(true)], true)).toBe(true);
    expect(updateHeldBack([win(false), win(false, true)], true)).toBe(false);
  });

  it('treats gateway errors as an unreachable server', () => {
    expect([502, 503, 504].every(isGatewayError)).toBe(true);
    expect([200, 404, 500].some(isGatewayError)).toBe(false);
  });
});
