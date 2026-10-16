import { overflow } from './policy';

type ArtCache = Pick<Cache, 'match' | 'delete' | 'put' | 'keys'>;

/** Cache lookup and LRU refresh share one transaction queue. A lookup must
 * not observe the temporary deletion that moves a cached hit to the end. */
export function artCache(open: () => Promise<ArtCache>, maxEntries: number) {
  let pending = Promise.resolve();

  function transaction<T>(work: (cache: ArtCache) => Promise<T>): Promise<T> {
    const result = pending.then(async () => work(await open()));
    pending = result.then(
      () => undefined,
      () => undefined
    );
    return result;
  }

  async function remember(cache: ArtCache, request: Request, response: Response): Promise<void> {
    await cache.delete(request);
    await cache.put(request, response);
    for (const key of overflow(await cache.keys(), maxEntries)) await cache.delete(key);
  }

  return {
    match(request: Request): Promise<Response | undefined> {
      return transaction(async (cache) => {
        const hit = await cache.match(request, { ignoreVary: true });
        if (hit) {
          // A failed refresh must not discard an already readable response.
          await remember(cache, request, hit.clone()).catch(() => undefined);
        }
        return hit;
      });
    },
    remember(request: Request, response: Response): Promise<void> {
      return transaction((cache) => remember(cache, request, response)).catch(() => undefined);
    }
  };
}
