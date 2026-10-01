// Prepares a clean e2e run: checks that the server binary and the web build
// exist, then empties the server's data directory.
//
// Playwright starts the webServer before it runs `globalSetup` (see
// createGlobalSetupTasks in playwright/lib/runner), so a check made there
// would come after the server had already failed or had already opened the
// data directory. That is why playwright.config.ts runs this file as the
// first step of the webServer command (`node fixtures/global-setup.ts &&
// exec ../server/bin/ajayeb ...`): its exit status decides whether the server
// starts at all. Node 24 runs this TypeScript file directly (type stripping).
//
// It deliberately builds nothing. `make e2e` builds both pieces first; a raw
// `playwright test` gets a clear message instead of a slow surprise build.

import { existsSync, mkdirSync, rmSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const e2eDir = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const repoDir = path.resolve(e2eDir, '..');

/** The server's --data directory for e2e runs. Keep in sync with the config. */
export const dataDir = path.join(repoDir, '.cache', 'e2e', 'data');

/** Files the e2e server needs, relative to the repository root. */
const required = ['server/bin/ajayeb', 'web/build/index.html'];

export default function globalSetup(): void {
  const missing = required.filter((rel) => !existsSync(path.join(repoDir, rel)));
  if (missing.length > 0) {
    throw new Error(
      `e2e: missing ${missing.join(' and ')}.\n` +
        'Build them first, from the repository root: make build-server build-web-test\n' +
        '(make e2e and make e2e-smoke do this for you; E2E_BUILD=0 skips it.)'
    );
  }
  rmSync(dataDir, { recursive: true, force: true });
  mkdirSync(dataDir, { recursive: true });
}

if (process.argv[1] && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    globalSetup();
  } catch (err) {
    console.error(err instanceof Error ? err.message : err);
    process.exit(1);
  }
}
