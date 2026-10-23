// M14 source revision 46a540d. Keep cold execution separate from SW install.
import { existsSync, readFileSync, writeFileSync } from 'node:fs';
import { gzipSync } from 'node:zlib';
import { expect, type Browser, type BrowserContext } from '@playwright/test';
import { test } from '../fixtures/base';
import { createDevRoom } from '../fixtures/ws-game';

const golden = new URL('../fixtures/platform-baselines.json', import.meta.url);
const update = process.env.UPDATE_PLATFORM_BASELINES === '1';
type Script = { path: string; gzipBytes: number };
type Entry = { gzipBytes: number; scripts: Script[] };
type Resource = { path: string; bodyBytes: number; transferBytes: number };
type Baseline = {
  sourceRevision: string;
  homeLimit: number;
  entryAllowance: number;
  home: Entry;
  duel: Entry;
  install: {
    files: number;
    bodyBytes: number;
    transferBytes: number;
    workerScriptBodyBytes: number;
    resources: Resource[];
  };
};

async function preferences(context: BrowserContext) {
  await context.addInitScript(() => {
    localStorage.setItem('ajayeb:name', 'Fixture');
    localStorage.setItem(
      'ajayeb:settings',
      JSON.stringify({ animationSpeed: 'off', showOpponentMove: false })
    );
    localStorage.setItem(
      'ajayeb:tips',
      JSON.stringify({ off: true, seen: [], tutorial: { offer: 'declined' } })
    );
  });
}

async function entry(browser: Browser, url: string, duel: boolean): Promise<Entry> {
  const context = await browser.newContext({ serviceWorkers: 'block' });
  try {
    await preferences(context);
    const page = await context.newPage();
    const cdp = await context.newCDPSession(page);
    await cdp.send('Network.enable');
    await cdp.send('Network.setCacheDisabled', { cacheDisabled: true });
    const scripts = new Map<string, Promise<number>>();
    page.on('response', (response) => {
      if (response.request().resourceType() === 'script')
        scripts.set(
          new URL(response.url()).pathname,
          response.body().then((body) => gzipSync(body).length)
        );
    });
    await page.goto(url);
    if (duel)
      await page.waitForFunction(
        () => window.__ajayeb?.ready() && window.__ajayeb.you()?.seats.length === 2
      );
    else await expect(page.locator('h1')).toBeVisible();
    await page.waitForLoadState('networkidle');
    const details = await Promise.all(
      [...scripts]
        .sort(([a], [b]) => a.localeCompare(b))
        .map(async ([path, size]) => ({ path, gzipBytes: await size }))
    );
    expect(details.length).toBeGreaterThan(0);
    expect(await page.evaluate(() => navigator.serviceWorker.controller)).toBeNull();
    return {
      gzipBytes: details.reduce((sum, script) => sum + script.gzipBytes, 0),
      scripts: details
    };
  } finally {
    await context.close();
  }
}

async function install(browser: Browser, baseURL: string): Promise<Baseline['install']> {
  // A JSON document runs no app JS and warms no build files. Register the
  // actual worker here, so its install fetches start with an empty HTTP cache.
  const context = await browser.newContext({ serviceWorkers: 'allow' });
  try {
    const page = await context.newPage();
    await page.goto(new URL('/health', baseURL).href);
    const pending: Promise<Resource>[] = [];
    context.on('requestfinished', (request) => {
      if (request.serviceWorker())
        pending.push(
          (async () => {
            const response = await request.response();
            if (!response || response.status() !== 200)
              throw new Error(`install fetch failed: ${request.url()}`);
            const size = await request.sizes();
            const path = new URL(request.url()).pathname;
            // Chromium reports a zero network body for the registration
            // script, but its body buffer works; cache.addAll has the reverse
            // limitation. Both responses are uncompressed on this local server.
            expect(response.headers()['content-encoding']).toBeUndefined();
            const bodyBytes =
              path === '/service-worker.js'
                ? (await response.body()).length
                : size.responseBodySize;
            return { path, bodyBytes, transferBytes: bodyBytes + size.responseHeadersSize };
          })()
        );
    });
    const cached = await page.evaluate(async () => {
      await navigator.serviceWorker.register('/service-worker.js');
      await navigator.serviceWorker.ready;
      const name = (await caches.keys()).find((name) => name.startsWith('ajayeb-shell-'));
      if (!name) throw new Error('worker did not populate its shell cache');
      const cache = await caches.open(name);
      const keys = await cache.keys();
      let bodyBytes = 0;
      for (const key of keys)
        bodyBytes += (await (await cache.match(key))!.arrayBuffer()).byteLength;
      return { files: keys.length, bodyBytes };
    });
    await expect.poll(() => pending.length).toBe(cached.files + 1);
    const resources = (await Promise.all(pending)).sort((a, b) => a.path.localeCompare(b.path));
    // Chromium exposes the registration fetch as a worker request too.
    // It is the one install request that is not itself put in the shell cache.
    const script = resources.find((resource) => resource.path === '/service-worker.js');
    expect(script, 'include the actual registration-script transfer').toBeDefined();
    // Local static responses are uncompressed: verify network body counts
    // independently against the real installed cache, not response.body(),
    // which Chromium exposes as empty for cache.addAll subrequests.
    expect(resources.reduce((sum, r) => sum + r.bodyBytes, 0) - script!.bodyBytes).toBe(
      cached.bodyBytes
    );
    for (const resource of resources) expect(resource.bodyBytes).toBeGreaterThan(0);
    return {
      files: cached.files,
      bodyBytes: resources.reduce((sum, r) => sum + r.bodyBytes, 0),
      transferBytes: resources.reduce((sum, r) => sum + r.transferBytes, 0),
      workerScriptBodyBytes: script!.bodyBytes,
      resources
    };
  } finally {
    await context.close();
  }
}

test('platform entry bytes and cold worker install baseline', async ({
  browser,
  baseURL
}, info) => {
  if (!baseURL) throw new Error('local baseURL required');
  const room = await createDevRoom(baseURL, { mode: 'hotseat', seed: '11' });
  const measured: Baseline = {
    sourceRevision: '46a540d',
    homeLimit: 200_000,
    entryAllowance: 5_000,
    home: await entry(browser, baseURL, false),
    duel: await entry(browser, new URL(room.path, baseURL).href, true),
    install: await install(browser, baseURL)
  };
  if (update) writeFileSync(golden, JSON.stringify(measured, null, 2) + '\n');
  if (!existsSync(golden)) throw new Error('record baselines deliberately (docs/testing.md)');
  const baseline = JSON.parse(readFileSync(golden, 'utf8')) as Baseline;
  await info.attach('platform-baselines', {
    body: JSON.stringify(measured, null, 2),
    contentType: 'application/json'
  });
  console.log(
    'platform-baselines',
    JSON.stringify({
      home: measured.home.gzipBytes,
      duel: measured.duel.gzipBytes,
      installFiles: measured.install.files,
      installBodyBytes: measured.install.bodyBytes,
      installTransferBytes: measured.install.transferBytes,
      workerScriptBodyBytes: measured.install.workerScriptBodyBytes
    })
  );
  expect(measured.home.gzipBytes).toBeLessThanOrEqual(baseline.homeLimit);
  expect(measured.home.gzipBytes).toBeLessThanOrEqual(
    baseline.home.gzipBytes + baseline.entryAllowance
  );
  expect(measured.duel.gzipBytes).toBeLessThanOrEqual(
    baseline.duel.gzipBytes + baseline.entryAllowance
  );
  expect(measured.install.files).toBeGreaterThan(0);
  expect(measured.install.transferBytes).toBeGreaterThan(0);
});
