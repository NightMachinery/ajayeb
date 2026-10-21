// Audit the recorded measurements themselves; runtime budget gates live
// in e2e/ux/platform-baselines.spec.ts and allow the documented 5 KB growth.
import { readFileSync } from 'node:fs';
import { expect, it } from 'vitest';

type Entry = { gzipBytes: number; scripts: { path: string; gzipBytes: number }[] };
type Measurements = {
  homeLimit: number;
  entryAllowance: number;
  home: Entry;
  duel: Entry;
  install: {
    files: number;
    bodyBytes: number;
    transferBytes: number;
    workerScriptBodyBytes: number;
    resources: { path: string; bodyBytes: number; transferBytes: number }[];
  };
};

it('keeps cold-entry and install baseline totals consistent with their per-file measurements', () => {
  const value = JSON.parse(
    readFileSync(new URL('../../e2e/fixtures/platform-baselines.json', import.meta.url), 'utf8')
  ) as Measurements;
  expect(value.homeLimit).toBe(200_000);
  expect(value.entryAllowance).toBe(5_000);
  for (const entry of [value.home, value.duel]) {
    expect(entry.scripts.length).toBeGreaterThan(0);
    expect(new Set(entry.scripts.map((s) => s.path)).size).toBe(entry.scripts.length);
    expect(entry.gzipBytes).toBe(entry.scripts.reduce((sum, s) => sum + s.gzipBytes, 0));
    for (const script of entry.scripts) expect(script.gzipBytes).toBeGreaterThan(0);
  }
  expect(value.home.gzipBytes).toBeLessThanOrEqual(value.homeLimit);
  const install = value.install;
  expect(install.files + 1).toBe(install.resources.length);
  expect(install.files).toBeGreaterThan(0);
  expect(install.bodyBytes).toBe(install.resources.reduce((sum, r) => sum + r.bodyBytes, 0));
  expect(install.transferBytes).toBe(
    install.resources.reduce((sum, r) => sum + r.transferBytes, 0)
  );
  expect(install.workerScriptBodyBytes).toBeGreaterThan(0);
  expect(install.workerScriptBodyBytes).toBe(
    install.resources.find((r) => r.path === '/service-worker.js')?.bodyBytes
  );
});
