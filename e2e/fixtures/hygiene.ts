// Page hygiene for every spec: console errors, uncaught exceptions, failed
// requests and HTTP error answers, collected as one list of problems that a
// spec expects to stay empty.

import type { Page, Request } from '@playwright/test';

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

function isCancelledStaticAsset(page: Page, req: Request): boolean {
  // Replaced images and idle prefetches may be abandoned. Match only the
  // browsers' cancellation codes, never connection failures or HTTP errors.
  const error = req.failure()?.errorText;
  if (
    !['Load request cancelled', 'net::ERR_ABORTED', 'NS_BINDING_ABORTED'].includes(error ?? '') ||
    req.method() !== 'GET'
  ) {
    return false;
  }
  try {
    const url = new URL(req.url());
    return (
      url.origin === new URL(page.url()).origin &&
      /^\/(?:art|fonts|icons|sfx|_app\/immutable)\//.test(url.pathname)
    );
  } catch {
    return false;
  }
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
  page.on('requestfailed', (req) => {
    if (isCancelledStaticAsset(page, req)) return;
    problems.push(`request failed: ${req.url()} (${req.failure()?.errorText ?? 'unknown'})`);
  });
  page.on('response', (res) => {
    if (res.status() >= 400) problems.push(`HTTP ${res.status()}: ${res.url()}`);
  });
  return problems;
}
