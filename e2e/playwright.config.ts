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

import { defineConfig, devices } from '@playwright/test';

const CI = !!process.env.CI;
const PORT = 38954;
const ORIGIN = `http://127.0.0.1:${PORT}`;

// Local HTTP must never go through a proxy. Extend whatever is already set
// (the runner, the webServer and the browser all inherit process.env).
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
  timeout: 180_000,
  expect: { timeout: 10_000 },
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
    actionTimeout: 20_000,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'off',
    launchOptions: {
      args: ['--disable-extensions', '--disable-background-networking', '--mute-audio']
    }
  },
  webServer: {
    command: serverCommand,
    url: `${ORIGIN}/health`,
    reuseExistingServer: false,
    timeout: 30_000,
    stdout: 'ignore',
    stderr: 'pipe',
    gracefulShutdown: { signal: 'SIGTERM', timeout: 5_000 }
  },
  projects: [
    { name: 'desktop', use: { ...devices['Desktop Chrome'] } },
    { name: 'mobile', use: { ...devices['Pixel 7'] }, grep: /@mobile/ }
  ]
});
