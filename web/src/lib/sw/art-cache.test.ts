import { describe, expect, it } from 'vitest';
import { artCache } from './art-cache';

function deferred() {
  let resolve = () => {};
  const promise = new Promise<void>((done) => (resolve = done));
  return { promise, resolve };
}

function memoryCache() {
  const entries = new Map<string, Response>();
  const cache = {
    async match(request: Request) {
      return entries.get(request.url)?.clone();
    },
    async delete(request: Request) {
      return entries.delete(request.url);
    },
    async put(request: Request, response: Response) {
      entries.set(request.url, response.clone());
    },
    async keys() {
      return [...entries.keys()].map((url) => new Request(url));
    }
  };
  return { entries, cache };
}

const request = (name: string) => new Request(`https://ajayeb.example/art/${name}.avif?v=1234`);

describe('runtime art cache transactions', () => {
  it('keeps a concurrent offline lookup behind the delete/put refresh gap', async () => {
    const { entries, cache } = memoryCache();
    const image = request('hero');
    entries.set(image.url, new Response('cached image'));
    const deleted = deferred();
    const release = deferred();
    const remove = cache.delete;
    let pause = true;
    cache.delete = async (key) => {
      const result = await remove(key);
      if (pause) {
        pause = false;
        deleted.resolve();
        await release.promise;
      }
      return result;
    };
    const art = artCache(async () => cache, 150);
    const first = art.match(image);
    await deleted.promise;
    expect(entries.has(image.url)).toBe(false);
    const second = art.match(image);
    let secondSettled = false;
    void second.then(() => (secondSettled = true));
    await Promise.resolve();
    await Promise.resolve();
    expect(secondSettled, 'lookup must wait while refresh has removed the key').toBe(false);
    release.resolve();
    expect(await (await first)?.text()).toBe('cached image');
    expect(await (await second)?.text()).toBe('cached image');
  });

  it('refreshes hits before enforcing the least recently used entry limit', async () => {
    const { cache } = memoryCache();
    const art = artCache(async () => cache, 2);
    await art.remember(request('a'), new Response('a'));
    await art.remember(request('b'), new Response('b'));
    await art.match(request('a'));
    await art.remember(request('c'), new Response('c'));
    expect((await cache.keys()).map((key) => key.url)).toEqual([
      request('a').url,
      request('c').url
    ]);
  });

  it('returns a readable hit and recovers its queue if a cache write fails', async () => {
    const { entries, cache } = memoryCache();
    entries.set(request('a').url, new Response('cached'));
    const put = cache.put;
    cache.put = async () => {
      throw new Error('quota exceeded');
    };
    const art = artCache(async () => cache, 2);
    expect(await (await art.match(request('a')))?.text()).toBe('cached');
    cache.put = put;
    await art.remember(request('b'), new Response('new'));
    expect(await (await art.match(request('b')))?.text()).toBe('new');
  });
});
