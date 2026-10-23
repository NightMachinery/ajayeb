// The art pipeline's game namespaces (tools/art/lib.mjs "Games"), run on a
// throwaway fixture tree (AJAYEB_ART_ROOT) and run directory: a Haftgane
// asset is described, built into its own manifest, approved, processed and
// indexed beside a Duel asset of the same slug, and nothing aimed at the
// wrong game's manifest can drop a Duel asset or replace a Duel file. The
// real repository's art is never touched. The processing step needs
// ImageMagick and avifenc and is skipped without them (it runs in
// `make verify-quick` on the art machine); every other step runs anywhere.
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import {
  copyFileSync,
  existsSync,
  mkdirSync,
  mkdtempSync,
  readdirSync,
  readFileSync,
  rmSync,
  writeFileSync
} from 'node:fs';
import { tmpdir } from 'node:os';
import { dirname, join, relative } from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { afterAll, beforeAll, describe, expect, it } from 'vitest';

const repo = fileURLToPath(new URL('../../../../', import.meta.url));
const tool = (name: string) => join(repo, 'tools', 'art', name);

const haveImageTools = ['magick', 'avifenc'].every(
  (cmd) => spawnSync(cmd, ['-version'], { stdio: 'ignore' }).status === 0
);

let root = '';
let run = '';

/** Runs one art tool on the fixture tree. */
function art(script: string, args: string[], env: Record<string, string> = {}) {
  const base = { ...process.env };
  delete base.AJAYEB_ART_MANIFEST;
  delete base.AJAYEB_ART_GAME;
  const r = spawnSync(process.execPath, [tool(script), ...args], {
    encoding: 'utf8',
    env: { ...base, AJAYEB_ART_ROOT: root, AJAYEB_RUN_DIR: run, ...env }
  });
  return { status: r.status, out: `${r.stdout}${r.stderr}` };
}

function write(rel: string, data: string | Buffer) {
  const file = join(root, rel);
  mkdirSync(dirname(file), { recursive: true });
  writeFileSync(file, data);
}

const readJson = (rel: string) => JSON.parse(readFileSync(join(root, rel), 'utf8'));
const sha = (buf: Buffer) => createHash('sha256').update(buf).digest('hex');

/** Every file under the fixture tree's Duel art, by path, with its hash. */
function duelFiles(): Record<string, string> {
  const out: Record<string, string> = {};
  const walk = (dir: string) => {
    if (!existsSync(dir)) return;
    for (const e of readdirSync(dir, { withFileTypes: true })) {
      const p = join(dir, e.name);
      if (e.isDirectory()) {
        if (e.name !== 'haftgane') walk(p);
      } else out[relative(root, p)] = sha(readFileSync(p));
    }
  };
  walk(join(root, 'art', 'originals'));
  walk(join(root, 'web', 'static', 'art'));
  out['art/manifest.json'] = sha(readFileSync(join(root, 'art', 'manifest.json')));
  return out;
}

function duelAsset(slug: string, status: string) {
  return {
    id: `card/${slug}`,
    kind: 'card',
    out: `cards/${slug}`,
    gen: '1536x1024',
    crop: '4:3',
    subject: `Duel's ${slug}`,
    accent: 'brown',
    status,
    attempt: 1,
    promptAmend: '',
    bytes: {},
    ...(status === 'approved' ? { approvedAttempt: 1 } : {})
  };
}

let duelBefore: Record<string, string> = {};

beforeAll(() => {
  root = mkdtempSync(join(tmpdir(), 'ajayeb-art-games-'));
  run = join(root, 'run');
  mkdirSync(join(root, 'art'), { recursive: true });
  copyFileSync(join(repo, 'art', 'style.md'), join(root, 'art', 'style.md'));
  // Duel: an approved Clay Pit and Lumber Yard, shipped, and a Stone Pit
  // still in review.
  write(
    'art/manifest.json',
    JSON.stringify(
      {
        generatedFrom: 'tools/art/manifest.mjs',
        assets: {
          'card/clay-pit': duelAsset('clay-pit', 'approved'),
          'card/lumber-yard': duelAsset('lumber-yard', 'approved'),
          'card/stone-pit': duelAsset('stone-pit', 'generated')
        }
      },
      null,
      2
    ) + '\n'
  );
  write('art/originals/card/clay-pit.webp', "Duel's clay pit original");
  for (const slug of ['clay-pit', 'lumber-yard']) {
    write(`web/static/art/cards/${slug}.avif`, `Duel ${slug} 480`);
    write(`web/static/art/cards/${slug}@thumb.avif`, `Duel ${slug} 240`);
  }
  // Haftgane: its own Clay Pit, and two aliases of Duel paintings.
  write(
    'art/haftgane/subjects.json',
    JSON.stringify({
      'card/clay-pit': { subject: 'a potter at a kiln beside a clay pit', accent: 'brown' }
    })
  );
  write(
    'art/haftgane/aliases.json',
    JSON.stringify({
      'card/lumber-yard': 'duel:card/lumber-yard',
      'card/stone-pit': 'duel:card/stone-pit'
    })
  );
  duelBefore = duelFiles();
});

afterAll(() => {
  if (root) rmSync(root, { recursive: true, force: true });
});

describe('a game namespace beside Duel', () => {
  it("builds Haftgane's manifest in its own file, from its own subjects", () => {
    const r = art('manifest.mjs', ['--game', 'haftgane']);
    expect(r.status, r.out).toBe(0);
    const m = readJson('art/haftgane/manifest.json');
    expect(m.game).toBe('haftgane');
    expect(Object.keys(m.assets)).toEqual(['card/clay-pit']);
    expect(m.assets['card/clay-pit']).toMatchObject({
      status: 'todo',
      out: 'haftgane/cards/clay-pit',
      subject: 'a potter at a kiln beside a clay pit'
    });
    expect(duelFiles()).toEqual(duelBefore);
  });

  it("refuses to rebuild, review, process or index another game's manifest", () => {
    const duelManifest = join(root, 'art', 'manifest.json');
    const haftganeManifest = join(root, 'art', 'haftgane', 'manifest.json');
    write(
      'decisions.jsonl',
      '{"id":"card/clay-pit","attempt":1,"verdict":"approve","reasons":["x"]}\n'
    );
    const wrong = [
      art('manifest.mjs', ['--game', 'haftgane', '--manifest', duelManifest]),
      art('manifest.mjs', [], { AJAYEB_ART_MANIFEST: haftganeManifest }),
      art('apply-review.mjs', [
        join(root, 'decisions.jsonl'),
        '--game',
        'haftgane',
        '--manifest',
        duelManifest
      ]),
      art('process.mjs', [
        '--game',
        'haftgane',
        '--manifest',
        duelManifest,
        '--id',
        'card/clay-pit'
      ]),
      art('index.mjs', [
        '--game',
        'haftgane',
        '--manifest',
        duelManifest,
        '--out',
        join(root, 'x.ts')
      ]),
      art('generate.mjs', ['--game', 'haftgane', '--manifest', duelManifest, '--dry-run'])
    ];
    for (const r of wrong.slice(0, 5)) {
      expect(r.status, r.out).not.toBe(0);
      expect(r.out).toMatch(/is the (duel|haftgane) manifest, not (haftgane|duel)'s/);
    }
    // generate.mjs takes no --manifest: it refuses the flag itself.
    expect(wrong[5]!.status).not.toBe(0);
    expect(duelFiles()).toEqual(duelBefore);
    expect(existsSync(join(root, 'x.ts'))).toBe(false);
  });

  it('drops approved art only when told to', () => {
    const copy = join(root, 'scratch-manifest.json');
    const m = readJson('art/haftgane/manifest.json');
    m.assets['card/loom'] = { ...m.assets['card/clay-pit'], id: 'card/loom', status: 'approved' };
    writeFileSync(copy, JSON.stringify(m, null, 2) + '\n');
    const before = readFileSync(copy, 'utf8');
    const refused = art('manifest.mjs', ['--game', 'haftgane', '--manifest', copy]);
    expect(refused.status, refused.out).not.toBe(0);
    expect(refused.out).toContain('would drop 1 approved asset');
    expect(readFileSync(copy, 'utf8')).toBe(before);
    const dropped = art('manifest.mjs', [
      '--game',
      'haftgane',
      '--manifest',
      copy,
      '--drop-approved'
    ]);
    expect(dropped.status, dropped.out).toBe(0);
    expect(Object.keys(JSON.parse(readFileSync(copy, 'utf8')).assets)).toEqual(['card/clay-pit']);
  });

  it('rejects an alias that also has a subject, or names nothing', () => {
    const subjects = readFileSync(join(root, 'art', 'haftgane', 'subjects.json'), 'utf8');
    const aliases = readFileSync(join(root, 'art', 'haftgane', 'aliases.json'), 'utf8');
    try {
      write('art/haftgane/aliases.json', JSON.stringify({ 'card/clay-pit': 'duel:card/clay-pit' }));
      const both = art('manifest.mjs', ['--game', 'haftgane', '--check']);
      expect(both.status).not.toBe(0);
      expect(both.out).toContain('has a subject and an alias');
      write('art/haftgane/aliases.json', JSON.stringify({ 'card/loom': 'duel:card/loom' }));
      const nothing = art('manifest.mjs', ['--game', 'haftgane', '--check']);
      expect(nothing.status).not.toBe(0);
      expect(nothing.out).toContain('not in the duel manifest');
    } finally {
      write('art/haftgane/subjects.json', subjects);
      write('art/haftgane/aliases.json', aliases);
    }
  });

  it(
    'generates, approves and ships a Haftgane Clay Pit without touching Duel’s',
    { timeout: 60_000 },
    () => {
      const dry = art('generate.mjs', [
        '--game',
        'haftgane',
        '--ids',
        'card/clay-pit',
        '--dry-run'
      ]);
      expect(dry.status, dry.out).toBe(0);
      expect(dry.out).toContain('a potter at a kiln beside a clay pit');

      // One attempt, as generate.mjs would leave it, in Haftgane's raw directory.
      const raw = join(run, 'art-work', 'haftgane', 'raw', 'card', 'clay-pit', '1.png');
      mkdirSync(dirname(raw), { recursive: true });
      if (haveImageTools) {
        const made = spawnSync('magick', [
          '-size',
          '768x512',
          'xc:#efe2c8',
          '-fill',
          '#8a5a2b',
          '-draw',
          'circle 384,256 384,120',
          `png:${raw}`
        ]);
        expect(made.status).toBe(0);
      } else {
        writeFileSync(raw, 'a stand-in attempt');
      }
      const m = readJson('art/haftgane/manifest.json');
      Object.assign(m.assets['card/clay-pit'], { status: 'generated', attempt: 1 });
      write('art/haftgane/manifest.json', JSON.stringify(m, null, 2) + '\n');

      const review = art('apply-review.mjs', [join(root, 'decisions.jsonl'), '--game', 'haftgane']);
      expect(review.status, review.out).toBe(0);
      expect(readJson('art/haftgane/manifest.json').assets['card/clay-pit']).toMatchObject({
        status: 'approved',
        approvedAttempt: 1
      });

      const shipped = ['clay-pit.avif', 'clay-pit@thumb.avif'].map((f) =>
        join(root, 'web', 'static', 'art', 'haftgane', 'cards', f)
      );
      if (haveImageTools) {
        const processed = art('process.mjs', ['--game', 'haftgane', '--id', 'card/clay-pit']);
        expect(processed.status, processed.out).toBe(0);
        expect(
          existsSync(join(root, 'art', 'originals', 'haftgane', 'card', 'clay-pit.webp'))
        ).toBe(true);
        expect(readJson('art/haftgane/manifest.json').assets['card/clay-pit'].bytes).toHaveProperty(
          ['clay-pit.avif']
        );
      } else {
        // What process.mjs would ship, so the index step below still runs.
        for (const f of shipped) {
          mkdirSync(dirname(f), { recursive: true });
          writeFileSync(f, `Haftgane ${f}`);
        }
      }
      for (const f of shipped) expect(existsSync(f), f).toBe(true);
      expect(duelFiles()).toEqual(duelBefore);
    }
  );

  it('indexes Haftgane’s own art under its URL prefix, and an alias with the Duel painting’s own paths', () => {
    const out = join(root, 'available-haftgane.gen.ts');
    const r = art('index.mjs', ['--game', 'haftgane', '--out', out]);
    expect(r.status, r.out).toBe(0);
    const ts = readFileSync(out, 'utf8');
    const hashOf = (rel: string) => sha(readFileSync(join(root, rel))).slice(0, 10);
    expect(ts).toContain(
      `/art/haftgane/cards/clay-pit.avif?v=${hashOf('web/static/art/haftgane/cards/clay-pit.avif')}`
    );
    expect(ts).toContain(
      `/art/cards/lumber-yard.avif?v=${hashOf('web/static/art/cards/lumber-yard.avif')}`
    );
    expect(ts).toContain(
      `/art/cards/lumber-yard@thumb.avif?v=${hashOf('web/static/art/cards/lumber-yard@thumb.avif')}`
    );
    // Duel's own Clay Pit is never Haftgane's, and an alias of art still in
    // review has no entry (its placeholder draws).
    expect(ts).not.toContain('/art/cards/clay-pit');
    expect(ts).not.toContain('stone-pit');
    expect(r.out).toContain('card/stone-pit: alias of duel:card/stone-pit, not approved yet');
    expect(duelFiles()).toEqual(duelBefore);
  });
});

interface Size {
  readonly suffix: string;
}
interface Dirs {
  readonly name: string;
  readonly originals: string;
  readonly shipped: string;
  readonly raw: string;
}
interface ArtLib {
  setGame(name: string): void;
  gameDirs(name: string): Dirs;
  originalPath(id: string, root?: string): string;
  shippedPath(id: string, size: Size, root?: string): string;
  shippedUrlPath(dirs: Dirs, id: string, size: Size): string;
  rawAttemptPath(id: string, attempt: number): string;
  qualifiedId(game: string, id: string): string;
  parseQualifiedId(q: string): { game: string; id: string } | null;
  KIND_SPEC: Record<string, { readonly sizes: readonly Size[] }>;
}

describe('the game paths', () => {
  it('give every Duel asset a different file and URL in Haftgane’s namespace', async () => {
    const lib = (await import(/* @vite-ignore */ pathToFileURL(tool('lib.mjs')).href)) as ArtLib;
    const duel = lib.gameDirs('duel');
    const haft = lib.gameDirs('haftgane');
    const ids = Object.keys(
      JSON.parse(readFileSync(join(repo, 'art', 'manifest.json'), 'utf8')).assets
    );
    expect(ids.length).toBeGreaterThan(100);
    const under = (p: string, dir: string) => p.startsWith(dir + '/');
    for (const id of ids) {
      expect(lib.originalPath(id, haft.originals)).not.toBe(lib.originalPath(id, duel.originals));
      expect(under(lib.originalPath(id, haft.originals), join(duel.originals, 'haftgane'))).toBe(
        true
      );
      for (const size of lib.KIND_SPEC[id.slice(0, id.indexOf('/'))]!.sizes) {
        const d = lib.shippedPath(id, size, duel.shipped);
        const h = lib.shippedPath(id, size, haft.shipped);
        expect(h).not.toBe(d);
        expect(under(h, join(duel.shipped, 'haftgane'))).toBe(true);
        expect(lib.shippedUrlPath(haft, id, size)).toBe(
          lib.shippedUrlPath(duel, id, size).replace(/^\/art\//, '/art/haftgane/')
        );
      }
    }
    lib.setGame('haftgane');
    expect(under(lib.rawAttemptPath('card/clay-pit', 1), haft.raw)).toBe(true);
    lib.setGame('duel');
    expect(under(lib.rawAttemptPath('card/clay-pit', 1), duel.raw)).toBe(true);
    expect(under(haft.raw, duel.raw)).toBe(false);
    expect(lib.parseQualifiedId(lib.qualifiedId('duel', 'card/lumber-yard'))).toEqual({
      game: 'duel',
      id: 'card/lumber-yard'
    });
    expect(lib.parseQualifiedId('chess:card/lumber-yard')).toBeNull();
    expect(lib.parseQualifiedId('card/lumber-yard')).toBeNull();
  });
});
