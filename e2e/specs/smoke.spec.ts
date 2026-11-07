// Smoke test: the server is up and the home page is healthy, in both system
// colour schemes, on the desktop and mobile projects. `make e2e-smoke` runs
// it (and `verify` does, through that target).

import AxeBuilder from '@axe-core/playwright';
import { EventEmitter } from 'node:events';
import { expect, type Page } from '@playwright/test';
import { test } from '../fixtures/base';
import { watchHygiene } from '../fixtures/hygiene';

// WCAG 2.2 AA, the level the UX plan targets.
const AXE_TAGS = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa'];

test.describe('smoke', { tag: ['@smoke', '@mobile'] }, () => {
  test('hygiene ignores only the two WebKit notices and still rejects unrelated errors', () => {
    const notices = [
      "The Content Security Policy 'default-src 'self'; script-src 'self' 'sha256-test'; object-src 'none'' was delivered in report-only mode, but does not specify a 'report-to'; the policy will have no effect. Please either add a 'report-to' directive, or deliver the policy via the 'Content-Security-Policy' header.",
      'Viewport argument key "interactive-widget" not recognized and ignored.'
    ];
    const errors = [
      'unrelated WebKit application error',
      "The Content Security Policy 'default-src 'self'; script-src 'self'' blocked a script.",
      'Viewport argument key "another-key" not recognized and ignored.'
    ];
    // Browser-free event sources exercise the real collector even on a host
    // where WebKit cannot launch. Chromium must retain the notices as errors.
    for (const browser of ['webkit', 'chromium', undefined]) {
      const events = new EventEmitter();
      const page = Object.assign(events, {
        context: () => ({
          browser: () => (browser ? { browserType: () => ({ name: () => browser }) } : null)
        })
      }) as unknown as Page;
      const problems = watchHygiene(page);
      for (const text of [...notices, ...errors]) {
        events.emit('console', { type: () => 'error', text: () => text });
      }
      const expected = browser === 'webkit' ? errors : [...notices, ...errors];
      expect(problems).toEqual(expected.map((text) => `console error: ${text}`));
      expect(() => expect(problems).toEqual([])).toThrow();
      events.emit('pageerror', new Error(notices[0]));
      expect(problems.at(-1)).toBe(`uncaught: ${notices[0]}`);
    }
  });

  test('GET /health reports ok', async ({ request }) => {
    const res = await request.get('/health');
    expect(res.ok()).toBe(true);
    expect(await res.json()).toMatchObject({ ok: true });
  });

  test('hygiene ignores only cancelled same-origin static assets', () => {
    const origin = 'https://game.example';
    const cancellations = ['Load request cancelled', 'net::ERR_ABORTED', 'NS_BINDING_ABORTED'];
    for (const browser of ['webkit', 'chromium', 'firefox', undefined]) {
      const events = new EventEmitter();
      const page = Object.assign(events, {
        url: () => `${origin}/room/ABC123`,
        context: () => ({
          browser: () => (browser ? { browserType: () => ({ name: () => browser }) } : null)
        })
      }) as unknown as Page;
      const problems = watchHygiene(page);
      const fail = (url: string, errorText: string | undefined, method = 'GET') => {
        events.emit('requestfailed', {
          url: () => url,
          method: () => method,
          failure: () => (errorText ? { errorText } : null)
        });
      };
      for (const error of cancellations) {
        for (const path of [
          '/art/cards/arsenal.avif?v=1234',
          '/fonts/text.woff2',
          '/icons/icon-192.png',
          '/sfx/coin.m4a?v=1234',
          '/_app/immutable/assets/page.css',
          '/_app/immutable/chunks/page.js'
        ]) {
          fail(`${origin}${path}`, error);
        }
      }
      expect(problems).toEqual([]);

      const expected: string[] = [];
      const mustFail = (url: string, error: string | undefined, method = 'GET') => {
        fail(url, error, method);
        expected.push(`request failed: ${url} (${error ?? 'unknown'})`);
      };
      for (const error of cancellations) {
        for (const path of ['/api/rooms', '/api/art/picture.avif', '/health', '/ws', '/room/X']) {
          mustFail(`${origin}${path}`, error);
        }
        mustFail('wss://game.example/ws', error);
        mustFail('https://other.example/art/cards/arsenal.avif', error);
        mustFail('http://game.example/art/cards/arsenal.avif', error);
        mustFail('https://game.example:8443/art/cards/arsenal.avif', error);
        mustFail(`${origin}/art/cards/arsenal.avif`, error, 'POST');
      }
      for (const error of ['net::ERR_CONNECTION_RESET', 'Load request failed', undefined]) {
        mustFail(`${origin}/art/cards/arsenal.avif`, error);
      }
      events.emit('response', { status: () => 404, url: () => `${origin}/art/missing.avif` });
      expected.push(`HTTP 404: ${origin}/art/missing.avif`);
      expect(problems).toEqual(expected);
    }
  });

  for (const colorScheme of ['light', 'dark'] as const) {
    test.describe(`home with a ${colorScheme} system theme`, () => {
      test.use({ colorScheme });

      test('renders, fits the viewport, logs no errors and passes axe', async ({ page }) => {
        const problems = watchHygiene(page);

        await page.goto('/');
        await expect(page).toHaveTitle(/Ajayeb/);
        await expect(page.getByRole('heading', { level: 1, name: 'Ajayeb' })).toBeVisible();
        // A new visitor gets light whatever the system scheme
        // (M18.installable-and-fullscreen).
        await expect(page.locator('html')).toHaveAttribute('data-theme', 'light');
        await page.evaluate(async () => {
          await document.fonts.ready;
        });

        const { sw, iw } = await page.evaluate(() => ({
          sw: document.documentElement.scrollWidth,
          iw: window.innerWidth
        }));
        expect(sw, `horizontal scroll: scrollWidth ${sw} > innerWidth ${iw}`).toBeLessThanOrEqual(
          iw
        );

        const axe = await new AxeBuilder({ page }).withTags(AXE_TAGS).analyze();
        const blocking = axe.violations.filter(
          (v) => v.impact === 'serious' || v.impact === 'critical'
        );
        const report = blocking
          .map(
            (v) =>
              `${v.impact} ${v.id}: ${v.help} at ${v.nodes.map((n) => n.target.join(' ')).join(', ')}`
          )
          .join('\n');
        expect(blocking, `serious or critical axe violations:\n${report}`).toEqual([]);

        expect(problems, 'console errors, page errors or failed requests').toEqual([]);
      });
    });
  }
});
