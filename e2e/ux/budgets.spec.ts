import { gzipSync } from 'node:zlib';
import { expect } from '@playwright/test';
import { test } from '../fixtures/base';
import { recordGame, createDevRoom } from '../fixtures/ws-game';
import { type Budget } from './report';

// Separate from the matrix, so every budget is measured once per invocation.
// A --grep screen filter intentionally omits this test for quick reshoots.
test('run budgets', async ({ browser, baseURL }, info) => {
  test.setTimeout(180_000);
  if (!baseURL) throw new Error('ux requires local baseURL');
  const budgets: Budget[] = [];
  const jsContext = await browser.newContext({ serviceWorkers: 'block' });
  try {
    const page = await jsContext.newPage();
    const scripts = new Map<string, Promise<number>>();
    page.on('response', (response) => {
      if (response.request().resourceType() === 'script')
        scripts.set(
          response.url(),
          response.body().then((body) => gzipSync(body).length)
        );
    });
    await page.goto(baseURL);
    await expect(page.locator('h1')).toBeVisible();
    await page.waitForLoadState('networkidle');
    const sizes = await Promise.all(
      [...scripts]
        .sort(([a], [b]) => a.localeCompare(b))
        .map(async ([url, size]) => ({ file: new URL(url).pathname, gzipBytes: await size }))
    );
    budgets.push({
      name: 'home-initial-js-gzip',
      value: sizes.reduce((n, s) => n + s.gzipBytes, 0),
      limit: 200_000,
      unit: 'bytes',
      details: sizes
    });
  } finally {
    await jsContext.close();
  }

  const recording = await recordGame(baseURL, '11');
  for (const viewport of [
    { width: 1280, height: 800 },
    { width: 390, height: 844 }
  ]) {
    const context = await browser.newContext({
      viewport,
      isMobile: viewport.width === 390,
      hasTouch: viewport.width === 390,
      serviceWorkers: 'block'
    });
    try {
      await context.addInitScript(() => {
        localStorage.setItem('ajayeb:name', 'Player');
        localStorage.setItem(
          'ajayeb:settings',
          JSON.stringify({
            animationSpeed: 'off',
            hotseatPassDevice: false,
            showOpponentMove: false
          })
        );
        localStorage.setItem(
          'ajayeb:tips',
          JSON.stringify({ off: true, seen: [], tutorial: { offer: 'declined' } })
        );
      });
      const room = await createDevRoom(baseURL, {
        mode: 'hotseat',
        seed: recording.seed,
        actions: recording.keys.slice(0, recording.marks.age3)
      });
      const page = await context.newPage();
      await page.goto(new URL(room.path, baseURL).href);
      await page.waitForFunction(
        () =>
          window.__ajayeb?.ready() &&
          window.__ajayeb.view()?.age === 3 &&
          window.__ajayeb.you()?.seats.length === 2
      );
      await page.evaluate(() => document.fonts.ready.then(() => undefined));
      await page.evaluate(async () => {
        await new Promise<void>((resolve) =>
          requestAnimationFrame(() => requestAnimationFrame(() => resolve()))
        );
      });
      const details = await page.evaluate(() => ({
        nodes: document.querySelectorAll('*').length,
        cards: window.__ajayeb?.view()?.players.map((p) => p.city.length),
        slots: window.__ajayeb?.view()?.structure.filter((s) => s.state !== 'empty').length
      }));
      budgets.push({
        name: `age3-dom-${viewport.width}x${viewport.height}`,
        value: details.nodes,
        limit: 2500,
        unit: 'nodes',
        details
      });
    } finally {
      await context.close();
    }
  }
  const lcp: (number | null)[] = [];
  for (let load = 0; load < 3; load++) {
    const context = await browser.newContext({
      viewport: { width: 390, height: 844 },
      isMobile: true,
      hasTouch: true,
      serviceWorkers: 'block'
    });
    try {
      const page = await context.newPage();
      const cdp = await context.newCDPSession(page);
      await cdp.send('Network.enable');
      await cdp.send('Network.setCacheDisabled', { cacheDisabled: true });
      await cdp.send('Emulation.setCPUThrottlingRate', { rate: 4 });
      // Slow 4G: 1.6 Mbps down, 750 Kbps up, 150ms latency (bytes/s in CDP).
      await cdp.send('Network.emulateNetworkConditions', {
        offline: false,
        latency: 150,
        downloadThroughput: 1_600_000 / 8,
        uploadThroughput: 750_000 / 8,
        connectionType: 'cellular4g'
      });
      await page.addInitScript(() => {
        localStorage.setItem('ajayeb:settings', JSON.stringify({ animationSpeed: 'off' }));
        new PerformanceObserver((list) => {
          const entries = list.getEntries();
          (window as unknown as { __uxLCP: number }).__uxLCP = entries.at(-1)?.startTime ?? 0;
        }).observe({ type: 'largest-contentful-paint', buffered: true });
      });
      await page.goto(baseURL, { waitUntil: 'load', timeout: 45_000 });
      await expect(page.locator('h1')).toBeVisible();
      // Closed sheets contain lazy images that deliberately never load.
      // Wait only for painted initial content, without interacting (which ends LCP).
      await page.waitForFunction(
        () =>
          [...document.images].every((image) => {
            if (!image.checkVisibility()) return true;
            const r = image.getBoundingClientRect();
            return (
              r.bottom <= 0 ||
              r.top >= innerHeight ||
              r.right <= 0 ||
              r.left >= innerWidth ||
              image.complete
            );
          }),
        null,
        { timeout: 15_000 }
      );
      await page.evaluate(() => document.fonts.ready.then(() => undefined));
      await page.waitForTimeout(1000);
      lcp.push(
        await page.evaluate(() => (window as unknown as { __uxLCP?: number }).__uxLCP ?? null)
      );
    } finally {
      await context.close();
    }
  }
  const measured = lcp.filter((n): n is number => n !== null).sort((a, b) => a - b);
  budgets.push({
    name: 'home-lcp-mobile-slow4g-median',
    value: measured.length === 3 ? measured[1]! / 1000 : null,
    limit: 2.5,
    unit: 'seconds',
    details: {
      samplesMilliseconds: lcp,
      cpuSlowdown: 4,
      downloadBitsPerSecond: 1_600_000,
      uploadBitsPerSecond: 750_000,
      latencyMilliseconds: 150,
      coldContexts: true,
      cacheDisabled: true
    }
  });
  await info.attach('ux-budgets', {
    body: JSON.stringify(budgets),
    contentType: 'application/json'
  });
});
