// The generated art index against the files it names (tools/art/index.mjs):
// every path carries the content hash of the shipped file it points at,
// the byte counts are the files' own, and running the generator again
// changes nothing, Prettier formatting included (`index.mjs --check`).
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';
import { art } from './available.gen';

const repo = new URL('../../../../', import.meta.url);
const staticDir = new URL('web/static/', repo);

describe('the art index', () => {
  it('names each shipped file by its content hash, with its real size', () => {
    let files = 0;
    for (const [id, entry] of Object.entries(art)) {
      for (const v of Object.values(entry.variants)) {
        const m = /^\/(art\/[a-z]+\/[a-z0-9@-]+\.avif)\?v=([0-9a-f]{10})$/.exec(v.path);
        expect(m, `${id}: ${v.path}`).not.toBeNull();
        const body = readFileSync(new URL(m![1]!, staticDir));
        expect(createHash('sha256').update(body).digest('hex').slice(0, 10), v.path).toBe(m![2]);
        expect(v.bytes, v.path).toBe(body.length);
        files++;
      }
    }
    expect(files).toBeGreaterThan(0);
  });

  it('is what tools/art/index.mjs writes today', () => {
    const script = fileURLToPath(new URL('tools/art/index.mjs', repo));
    const out = execFileSync(process.execPath, [script, '--check'], { encoding: 'utf8' });
    expect(out).toContain('is up to date');
  }, 30_000);
});
