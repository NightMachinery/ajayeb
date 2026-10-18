import { expect } from '@playwright/test';
import { test } from '../fixtures/base';
import { SOUND_FILES } from '../../web/src/lib/fx/sounds.gen';

test(
  'sound files: lazy after a gesture, all AAC files decode, gallery has both versions',
  { tag: ['@mobile'] },
  async ({ page }, info) => {
    const errors: string[] = [];
    page.on('pageerror', (error) => errors.push(error.message));
    const requests: string[] = [];
    page.on('request', (request) => {
      if (new URL(request.url()).pathname.startsWith('/sfx/')) requests.push(request.url());
    });
    await page.goto('/dev/gallery');
    await expect(page.locator('[data-sound-board]')).toBeAttached();
    expect(requests).toEqual([]);
    for (const cue of Object.keys(SOUND_FILES)) {
      await expect(page.locator(`[data-sound-cue="${cue}"] [data-cue]`)).toHaveCount(1);
      await expect(page.locator(`[data-synth="${cue}"]`)).toHaveCount(1);
    }
    await page.locator('[data-cue="coin"]').click();
    await expect(page.locator('[data-cue="coin"]')).toBeEnabled();
    expect(new Set(requests).size).toBe(Object.keys(SOUND_FILES).length);
    const decoded = await page.evaluate(async (files) => {
      const context = new AudioContext();
      try {
        const results = [];
        for (const [cue, { url }] of Object.entries(files)) {
          const response = await fetch(url);
          const buffer = await context.decodeAudioData(await response.arrayBuffer());
          results.push({ cue, duration: buffer.duration, channels: buffer.numberOfChannels });
        }
        return {
          aac: document.createElement('audio').canPlayType('audio/mp4; codecs="mp4a.40.2"'),
          results
        };
      } finally {
        await context.close();
      }
    }, SOUND_FILES);
    await info.attach('aac-decode', {
      body: JSON.stringify(decoded),
      contentType: 'application/json'
    });
    console.log('AAC decode:', JSON.stringify(decoded));
    expect(decoded.aac).not.toBe('');
    expect(decoded.results).toHaveLength(17);
    for (const result of decoded.results) {
      expect(result.channels, result.cue).toBe(1);
      expect(result.duration, result.cue).toBeGreaterThan(0);
      expect(result.duration, result.cue).toBeLessThan(3.1);
    }
    await page.locator('[data-synth="coin"]').click();
    expect(errors).toEqual([]);
  }
);

test(
  'gallery reports an unavailable file and still offers its synth',
  { tag: ['@mobile'] },
  async ({ page }) => {
    await page.route('**/sfx/coin.m4a?*', (route) => route.fulfill({ status: 404, body: '' }));
    await page.goto('/dev/gallery');
    await page.locator('[data-cue="coin"]').click();
    const status = page.locator('[data-sound-cue="coin"] [role="status"]');
    await expect(status).toContainText('Sound file unavailable');
    await page.locator('[data-synth="coin"]').click();
    await expect(status).toHaveCount(0);
  }
);

test.describe('sound runtime cache', () => {
  test.use({ serviceWorkers: 'allow' });
  test(
    'installation leaves sounds unfetched; first audition caches them for offline use',
    { tag: ['@mobile'] },
    async ({ page, context }) => {
      await page.goto('/dev/gallery');
      await page.evaluate(async () => {
        await navigator.serviceWorker.ready;
        if (!navigator.serviceWorker.controller) {
          await new Promise<void>((resolve) =>
            navigator.serviceWorker.addEventListener('controllerchange', () => resolve(), {
              once: true
            })
          );
        }
      });
      expect(
        await page.evaluate(async () => {
          const matches = [];
          for (const name of await caches.keys()) {
            const cache = await caches.open(name);
            matches.push(
              ...(await cache.keys()).filter((r) => new URL(r.url).pathname.startsWith('/sfx/'))
            );
          }
          return matches.length;
        })
      ).toBe(0);
      await page.locator('[data-cue="coin"]').click();
      await expect
        .poll(() =>
          page.evaluate(async () => (await (await caches.open('ajayeb-sound')).keys()).length)
        )
        .toBe(17);
      await context.setOffline(true);
      try {
        expect(
          await page.evaluate(async (files) => {
            for (const { url } of Object.values(files)) {
              const response = await fetch(url);
              if (!response.ok || (await response.arrayBuffer()).byteLength === 0) return false;
            }
            return true;
          }, SOUND_FILES)
        ).toBe(true);
      } finally {
        await context.setOffline(false);
      }
    }
  );
});

test(
  'gallery sound controls fit phone and desktop widths',
  { tag: ['@mobile'] },
  async ({ page }, info) => {
    await page.goto('/dev/gallery');
    const board = page.locator('[data-sound-board]');
    for (const viewport of [
      { width: 360, height: 740 },
      { width: 390, height: 844 },
      { width: 740, height: 360 },
      { width: 1280, height: 800 }
    ]) {
      await page.setViewportSize(viewport);
      await board.scrollIntoViewIfNeeded();
      expect(await board.evaluate((node) => node.scrollWidth <= node.clientWidth)).toBe(true);
      await info.attach(`sound-board-${viewport.width}x${viewport.height}`, {
        body: await board.screenshot({
          path: info.outputPath(`sound-board-${viewport.width}x${viewport.height}.png`)
        }),
        contentType: 'image/png'
      });
    }
  }
);
