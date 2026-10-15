// The page side of the service worker's update rule (src/service-worker.ts):
// while a page shows a game, it holds the shared Web Lock GAME_LOCK, and a
// new version of the worker waits for it. The browser releases the lock by
// itself when the page goes away, a reload included.

import { GAME_LOCK } from './policy';

/** Takes the game lock and returns the function that lets it go. Does
 *  nothing where Web Locks are missing (the worker then goes by URL). */
export function holdGameLock(): () => void {
  if (typeof navigator === 'undefined' || !('locks' in navigator)) return () => {};
  let release = (): void => {};
  const held = new Promise<void>((resolve) => {
    release = resolve;
  });
  navigator.locks.request(GAME_LOCK, { mode: 'shared' }, () => held).catch(() => undefined);
  return release;
}
