// Page hygiene for every spec: console errors, uncaught exceptions, failed
// requests and HTTP error answers, collected as one list of problems that a
// spec expects to stay empty.

import type { Page } from '@playwright/test';

/** Starts collecting a page's problems; the returned array fills as they happen. */
export function watchHygiene(page: Page): string[] {
  const problems: string[] = [];
  page.on('console', (msg) => {
    if (msg.type() === 'error') problems.push(`console error: ${msg.text()}`);
  });
  page.on('pageerror', (err) => problems.push(`uncaught: ${err.message}`));
  page.on('requestfailed', (req) =>
    problems.push(`request failed: ${req.url()} (${req.failure()?.errorText ?? 'unknown'})`)
  );
  page.on('response', (res) => {
    if (res.status() >= 400) problems.push(`HTTP ${res.status()}: ${res.url()}`);
  });
  return problems;
}
