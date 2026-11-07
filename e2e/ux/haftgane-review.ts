// Shared M20 capture adapter for the existing Haftgane UX specs. Product
// findings are observations; a missing screen still fails the Playwright test.
import fs from 'node:fs/promises';
import path from 'node:path';
import AxeBuilder from '@axe-core/playwright';
import { expect, type Page, type TestInfo } from '@playwright/test';
import { probePage } from '../../tools/chrome/probe.mjs';
import { settleCapture } from './settle';
import { axeTags, findings, shotsDir, type Capture } from './report';

export const reviewSeats = [3, 4, 5, 6, 7] as const;
// These are the seat counts in the existing real-engine gallery captures.
export const reviewFixtureSeats = [3, 5, 7] as const;
export const reviewVariants = [
  ['en', 'light'],
  ['en', 'dark'],
  ['fa', 'light'],
  ['fa', 'dark']
] as const;
export const reviewViewports = [
  { width: 390, height: 844 },
  { width: 1280, height: 800 }
] as const;

export function reviewHygiene(page: Page) {
  const consoleErrors: string[] = [],
    exceptions: string[] = [],
    failedRequests: string[] = [];
  page.on('console', (m) => {
    if (m.type() === 'error' || m.type() === 'assert') consoleErrors.push(m.text());
  });
  page.on('pageerror', (e) => exceptions.push(e.message));
  page.on('requestfailed', (r) => {
    if (r.failure()?.errorText !== 'net::ERR_ABORTED')
      failedRequests.push(`${r.failure()?.errorText} ${new URL(r.url()).pathname}`);
  });
  page.on('response', (r) => {
    if (r.status() >= 400) failedRequests.push(`HTTP ${r.status()} ${new URL(r.url()).pathname}`);
  });
  // Drain after each shot: an error belongs to the screen that produced it,
  // rather than appearing again on every later shot of this live game.
  return () => ({
    consoleErrors: [...new Set(consoleErrors.splice(0))].sort(),
    exceptions: [...new Set(exceptions.splice(0))].sort(),
    failedRequests: [...new Set(failedRequests.splice(0))].sort()
  });
}

export async function reviewCapture(
  page: Page,
  info: TestInfo,
  screen: string,
  seats: number,
  hygiene: ReturnType<typeof reviewHygiene>,
  scope = 'document',
  sourceKind: 'live' | 'fixture' = 'live'
): Promise<void> {
  await settleCapture(page, screen);
  const {
    lang: locale,
    dir: direction,
    theme
  } = await page.locator('html').evaluate((el) => ({
    lang: el.getAttribute('lang')!,
    dir: el.getAttribute('dir')!,
    theme: el.dataset.theme!
  }));
  expect(['ltr', 'rtl']).toContain(direction);
  const vp = page.viewportSize()!;
  const viewport = `${vp.width}x${vp.height}`;
  const file = `${screen}__${seats}p__${viewport}__${theme}__${locale}.png`;
  const raw = await page.evaluate(probePage);
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
  const builder = new AxeBuilder({ page }).withTags(axeTags);
  if (scope !== 'document') builder.include(scope);
  const axe = (await builder.analyze()).violations
    .filter((v) => v.impact === 'critical' || v.impact === 'serious')
    .map((v) => ({
      id: v.id,
      impact: v.impact,
      help: v.help,
      targets: v.nodes.map((n) => n.target.join(' ')).sort()
    }));
  const url = new URL(page.url());
  const capture: Capture = {
    screen,
    seats,
    direction,
    sourceKind,
    viewport,
    theme,
    locale,
    file,
    source: url.pathname.replace(/\/play\/[A-Z0-9]+$/, '/play/<room>') + url.search,
    axeScope: scope,
    probe,
    axe,
    ...hygiene(),
    failures: []
  };
  capture.failures = findings(capture);
  // M20 reports recommended-size misses as P2, retaining the actual counts.
  capture.failures = capture.failures.map((f) => ({ ...f, severity: Math.min(f.severity, 2) }));
  await fs.mkdir(shotsDir, { recursive: true });
  await page.screenshot({ path: path.join(shotsDir, file), animations: 'disabled' });
  await info.attach('ux-capture', {
    body: JSON.stringify(capture),
    contentType: 'application/json'
  });
}
