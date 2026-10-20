// Page hygiene for every spec: console errors, uncaught exceptions, failed
// requests and HTTP error answers, collected as one list of problems that a
// spec expects to stay empty.

import type { Page } from '@playwright/test';

function isWebKitNotice(text: string): boolean {
  // WebKit reports these browser notices as errors. The test server's CSP is
  // deliberately report-only; production enforces it. Keep actual violations.
  return (
    (text.startsWith("The Content Security Policy 'default-src 'self'; script-src 'self'") &&
      text.includes(
        "' was delivered in report-only mode, but does not specify a 'report-to'; the policy will have no effect."
      )) ||
    text.startsWith('Viewport argument key "interactive-widget" not recognized and ignored.')
  );
}

/** Starts collecting a page's problems; the returned array fills as they happen. */
export function watchHygiene(page: Page): string[] {
  const problems: string[] = [];
  const webkit = page.context().browser()?.browserType().name() === 'webkit';
  page.on('console', (msg) => {
    if (msg.type() !== 'error') return;
    const text = msg.text();
    if (webkit && isWebKitNotice(text)) return;
    problems.push(`console error: ${text}`);
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
