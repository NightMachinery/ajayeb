import fs from 'node:fs/promises';
import path from 'node:path';
import AxeBuilder from '@axe-core/playwright';
import { expect } from '@playwright/test';
import { test } from '../fixtures/base';
import { probePage } from '../../tools/chrome/probe.mjs';
import { reachScreen, screenIds } from './screens';
import { settleCapture } from './settle';
import { axeTags, findings, shotsDir, type Capture } from './report';

const viewports = [
  { width: 360, height: 740 },
  { width: 390, height: 844 },
  { width: 740, height: 360 },
  { width: 1280, height: 800 }
];
for (const viewport of viewports) {
  for (const theme of ['light', 'dark'] as const) {
    for (const locale of ['en', 'fa'] as const) {
      if (locale === 'fa' && (theme !== 'light' || ![390, 1280].includes(viewport.width))) continue;
      const vp = `${viewport.width}x${viewport.height}`;
      const mobile = viewport.width !== 1280;
      test.describe(`${vp} ${theme} ${locale}`, () => {
        test.use({
          viewport,
          isMobile: mobile,
          hasTouch: mobile,
          deviceScaleFactor: 1,
          appLocale: locale
        });
        for (const screen of screenIds) {
          test(screen, async ({ page, baseURL }, info) => {
            if (!baseURL) throw new Error('ux requires local baseURL');
            const consoleErrors: string[] = [],
              exceptions: string[] = [],
              failedRequests: string[] = [];
            page.on('console', (msg) => {
              if (msg.type() === 'error' || msg.type() === 'assert') consoleErrors.push(msg.text());
            });
            page.on('pageerror', (error) => exceptions.push(error.message));
            page.on('requestfailed', (request) => {
              if (request.failure()?.errorText !== 'net::ERR_ABORTED')
                failedRequests.push(
                  `${request.failure()?.errorText} ${new URL(request.url()).pathname}`
                );
            });
            page.on('response', (response) => {
              if (response.status() >= 400)
                failedRequests.push(
                  `HTTP ${response.status()} ${new URL(response.url()).pathname}`
                );
            });
            await page.addInitScript(
              ({ theme, passDevice }) => {
                localStorage.setItem('ajayeb:theme', theme);
                localStorage.setItem('ajayeb:name', 'Player');
                const settings = JSON.parse(localStorage.getItem('ajayeb:settings') ?? '{}');
                localStorage.setItem(
                  'ajayeb:settings',
                  JSON.stringify({
                    ...settings,
                    confirmMoves: 'never',
                    hotseatPassDevice: passDevice,
                    sound: false,
                    haptics: false
                  })
                );
              },
              { theme, passDevice: screen === 'pass-device' }
            );
            await reachScreen(page, baseURL, screen);
            await expect(page.locator('html')).toHaveAttribute('data-theme', theme);
            await expect(page.locator('html')).toHaveAttribute('lang', locale);
            await expect(page.locator('html')).toHaveAttribute('data-motion', 'reduce');
            await settleCapture(page, screen);
            const raw = await page.evaluate(probePage);
            // Keep identity, room URL and incidental text out of the report.
            const {
              iw,
              ih,
              viewportWidth,
              viewportHeight,
              sw,
              sh,
              hscroll,
              vscroll,
              domNodes,
              overflow,
              viewportOutside,
              covered,
              tap
            } = raw;
            const probe = {
              iw,
              ih,
              viewportWidth,
              viewportHeight,
              sw,
              sh,
              hscroll,
              vscroll,
              domNodes,
              overflow,
              viewportOutside,
              covered,
              tap
            };
            const axeScope = screen === 'discard-pile' ? 'dialog[open]' : 'document';
            const builder = new AxeBuilder({ page }).withTags(axeTags);
            // Only this gallery specimen has unrelated inert demonstration UI.
            if (screen === 'discard-pile') builder.include(axeScope);
            const axe = (await builder.analyze()).violations
              .filter((v) => v.impact === 'critical' || v.impact === 'serious')
              .map((v) => ({
                id: v.id,
                impact: v.impact,
                help: v.help,
                targets: v.nodes.map((n) => n.target.join(' ')).sort()
              }));
            const url = new URL(page.url());
            const source = url.pathname.replace(/\/play\/[A-Z0-9]+$/, '/play/<room>') + url.search;
            const file = `${screen}__${vp}__${theme}__${locale}.png`;
            await fs.mkdir(shotsDir, { recursive: true });
            await page.screenshot({ path: path.join(shotsDir, file), animations: 'disabled' });
            const capture: Capture = {
              screen,
              viewport: vp,
              theme,
              locale,
              file,
              source,
              axeScope,
              probe,
              axe,
              consoleErrors: [...new Set(consoleErrors)].sort(),
              exceptions: [...new Set(exceptions)].sort(),
              failedRequests: [...new Set(failedRequests)].sort(),
              failures: []
            };
            capture.failures = findings(capture);
            await info.attach('ux-capture', {
              body: JSON.stringify(capture),
              contentType: 'application/json'
            });
          });
        }
      });
    }
  }
}
