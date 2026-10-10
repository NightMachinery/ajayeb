// Playwright configuration for Ajayeb's end-to-end tests.
//
// Run through the guard, never bare: `make e2e`, `make e2e-smoke`, or
// `python3 tools/heavy.py run --kind browser -- pnpm -C e2e exec playwright test`.
// The guard's browser lock keeps it to one Chrome on the machine.
//
// Locally the tests use the installed Google Chrome (channel 'chrome', no
// browser download). On CI (CI=true) they use Playwright's bundled Chromium,
// installed by `playwright install --with-deps chromium`.
//
// Paths are relative to this file's directory, which is also the webServer's
// working directory. Every output goes under ../.cache/e2e/ (git ignores it).
//
// E2E_BASE_URL switches the whole config to production-smoke mode (`make
// e2e-prod`, docs/testing.md "Production smoke"): it becomes `baseURL`
// instead of the local server's origin, no `webServer` is started (the site
// is already running), and every project's `grep` is forced to `@prod`, so
// that is the only thing that can select a spec to run against a real site.
// This has to be done per project, not just once at the top level: Playwright
// resolves each project's effective `grep` as "this project's own `grep` if
// it set one, else the top-level config's", not as an AND of the two, so a
// project that already names its own `grep` (here, `mobile`'s `/@mobile/`)
// would otherwise keep matching every `@mobile` spec regardless of the
// top-level one, defeating it silently. (`--grep` from the command line is
// different: Playwright truly ANDs that one in on top of a project's `grep`,
// which is how `make e2e-smoke`'s `--grep '@smoke'` already narrows both
// projects at once. The forcing here does not rely on that, since a bare
// `playwright test` with E2E_BASE_URL set passes no `--grep` at all.) A spec
// that needs a dev endpoint or a `/dev` route (every spec but
// `prod.smoke.spec.ts`) is never tagged `@prod`, so it can never run this
// way. Without E2E_BASE_URL nothing here changes.

import { defineConfig, devices } from '@playwright/test';

const CI = !!process.env.CI;
const PORT = 38954;
const PROD_BASE_URL = process.env.E2E_BASE_URL;
const ORIGIN = PROD_BASE_URL ?? `http://127.0.0.1:${PORT}`;

// Local HTTP must never go through a proxy. Extend whatever is already set
// (the runner, the webServer and the browser all inherit process.env). This
// is harmless when ORIGIN is a real remote host too: it only ever adds
// entries, never removes whatever lets that traffic through.
for (const key of ['NO_PROXY', 'no_proxy'] as const) {
  const have = (process.env[key] ?? '').split(',').filter(Boolean);
  process.env[key] = [...new Set([...have, '127.0.0.1', 'localhost'])].join(',');
}

const serverCommand = [
  // The prerequisite check and the data wipe must run before the server
  // starts; see the header of fixtures/global-setup.ts for why this is not
  // the `globalSetup` option.
  'node fixtures/global-setup.ts &&',
  'exec ../server/bin/ajayeb',
  `--addr 127.0.0.1:${PORT}`,
  '--data ../.cache/e2e/data',
  '--static ../web/build',
  '--dev --dev-endpoints',
  '--bot-delay 0',
  '--seed-base 1000'
].join(' ');

export default defineConfig({
  testDir: './specs',
  workers: 1,
  fullyParallel: false,
  forbidOnly: CI,
  retries: CI ? 1 : 0,
  // A real network round trip is slower than the loopback server, so the
  // prod run gets more room: 10 minutes per test instead of 3, and 30s
  // instead of 10s for a single `expect`.
  timeout: PROD_BASE_URL ? 600_000 : 180_000,
  expect: { timeout: PROD_BASE_URL ? 30_000 : 10_000 },
  // Forced, and (see the header comment) repeated on every project below so
  // a project's own `grep` cannot quietly take over instead.
  ...(PROD_BASE_URL ? { grep: /@prod/ } : {}),
  outputDir: '../.cache/e2e/results',
  reporter: [
    ['list'],
    ['html', { open: 'never', outputFolder: '../.cache/e2e/report' }],
    ...(CI ? ([['github']] as const) : [])
  ],
  use: {
    baseURL: ORIGIN,
    channel: CI ? undefined : 'chrome',
    headless: true,
    reducedMotion: 'reduce',
    // A click that cannot land (a covered control) fails here, with its
    // call log, instead of waiting out the whole test.
    actionTimeout: PROD_BASE_URL ? 40_000 : 20_000,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'off',
    launchOptions: {
      args: ['--disable-extensions', '--disable-background-networking', '--mute-audio']
    }
  },
  // No webServer in production-smoke mode: the site named by E2E_BASE_URL is
  // already running, and nothing here may start a local one in its place.
  ...(PROD_BASE_URL
    ? {}
    : {
        webServer: {
          command: serverCommand,
          url: `${ORIGIN}/health`,
          reuseExistingServer: false,
          timeout: 30_000,
          stdout: 'ignore',
          stderr: 'pipe',
          gracefulShutdown: { signal: 'SIGTERM', timeout: 5_000 }
        }
      }),
  projects: [
    // Unset, this project's grep falls back to the top-level config.grep
    // above, so the @prod forcing already reaches it with no extra code.
    { name: 'desktop', use: { ...devices['Desktop Chrome'] } },
    // This project sets its own grep, so production-smoke mode must repeat
    // the @prod forcing here too: see the header comment above. It is ANDed
    // with the project's normal @mobile filter (both lookaheads must match),
    // not replaced by it, so a spec needs both tags to run here against a
    // real site. prod.smoke.spec.ts carries only @prod, so by default this
    // project runs none of it and a full `make e2e-prod` stays desktop-only;
    // tag a test @mobile too, once it is confirmed to also pass this way, to
    // let it run here as well.
    {
      name: 'mobile',
      use: { ...devices['Pixel 7'] },
      grep: PROD_BASE_URL ? /(?=.*@prod)(?=.*@mobile)/ : /@mobile/
    }
  ]
});
