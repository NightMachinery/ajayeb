import { describe, expect, it } from 'vitest';
import { DEV_ROUTES, devOnlyFiles, staticSpecifiers, type Manifest } from './dev-only';

const NODES = '.svelte-kit/generated/client-optimized/nodes';
const ROUTES = '../../../../src/routes';

// Route node sources in the shapes SvelteKit generates them.
const SOURCES: Record<string, string> = {
  [`${NODES}/0.js`]: `import * as universal from "${ROUTES}/+layout.ts";\nexport { universal };\nexport { default as component } from "${ROUTES}/+layout.svelte";`,
  [`${NODES}/1.js`]: `export { default as component } from "${ROUTES}/+error.svelte";`,
  [`${NODES}/2.js`]: `import * as universal from "${ROUTES}/dev/+layout.ts";\nexport { universal };\nexport { default as component } from "../../../../../node_modules/@sveltejs/kit/src/runtime/components/svelte-5/layout.svelte";`,
  [`${NODES}/3.js`]: `export { default as component } from "${ROUTES}/+page.svelte";`,
  [`${NODES}/4.js`]: `export { default as component } from "${ROUTES}/dev/gallery/+page.svelte";`,
  [`${NODES}/5.js`]: `export { default as component } from "${ROUTES}/devices/+page.svelte";`
};

function read(key: string): string {
  const source = SOURCES[key];
  if (source === undefined) throw new Error(`read ${key}`);
  return source;
}

const node = (n: number) => `${NODES}/${n}.js`;

/** A small client manifest: Home (node 3) and a dev gallery (node 4) share
 *  some chunks and each has its own, static and lazy. */
function manifest(): Manifest {
  return {
    '../node_modules/@sveltejs/kit/src/runtime/client/entry.js': {
      file: '_app/immutable/entry/start.js',
      isEntry: true,
      imports: ['_runtime.js']
    },
    '.svelte-kit/generated/client-optimized/app.js': {
      file: '_app/immutable/entry/app.js',
      isEntry: true,
      imports: ['_runtime.js'],
      dynamicImports: [node(0), node(1), node(2), node(3), node(4), node(5)]
    },
    [node(0)]: {
      file: '_app/immutable/nodes/0.js',
      isEntry: true,
      imports: ['_runtime.js'],
      css: ['_app/immutable/assets/0.css'],
      assets: ['_app/immutable/assets/inter.woff2']
    },
    [node(1)]: { file: '_app/immutable/nodes/1.js', isEntry: true, imports: ['_runtime.js'] },
    [node(2)]: { file: '_app/immutable/nodes/2.js', isEntry: true, imports: ['_flags.js'] },
    [node(3)]: {
      file: '_app/immutable/nodes/3.js',
      isEntry: true,
      imports: ['_runtime.js', '_card.js'],
      dynamicImports: ['src/lib/game/lazy.ts']
    },
    [node(4)]: {
      file: '_app/immutable/nodes/4.js',
      isEntry: true,
      imports: ['_runtime.js', '_card.js', '_fixtures.js', '_flags.js'],
      dynamicImports: ['src/lib/game/lazy.ts', 'src/lib/dev/leaves.ts'],
      css: ['_app/immutable/assets/4.css'],
      assets: ['_app/immutable/assets/inter.woff2', '_app/immutable/assets/specimen.png']
    },
    [node(5)]: { file: '_app/immutable/nodes/5.js', isEntry: true, imports: ['_runtime.js'] },
    '_runtime.js': { file: '_app/immutable/chunks/runtime.js' },
    '_card.js': { file: '_app/immutable/chunks/card.js', css: ['_app/immutable/assets/card.css'] },
    '_flags.js': { file: '_app/immutable/chunks/flags.js' },
    '_fixtures.js': { file: '_app/immutable/chunks/fixtures.js', imports: ['_runtime.js'] },
    'src/lib/game/lazy.ts': {
      file: '_app/immutable/chunks/lazy.js',
      isDynamicEntry: true,
      imports: ['_card.js']
    },
    'src/lib/dev/leaves.ts': {
      file: '_app/immutable/chunks/leaves.js',
      isDynamicEntry: true,
      imports: ['_fixtures.js', '_leafart.js'],
      css: ['_app/immutable/assets/leaves.css']
    },
    '_leafart.js': { file: '_app/immutable/chunks/leafart.js' }
  };
}

describe('devOnlyFiles', () => {
  it("lists the dev nodes' own chunks, CSS and assets, with the routes that load them", () => {
    expect(devOnlyFiles(manifest(), read)).toEqual([
      { file: '_app/immutable/assets/4.css', routes: ['src/routes/dev/gallery/+page.svelte'] },
      { file: '_app/immutable/assets/leaves.css', routes: ['src/routes/dev/gallery/+page.svelte'] },
      {
        file: '_app/immutable/assets/specimen.png',
        routes: ['src/routes/dev/gallery/+page.svelte']
      },
      {
        file: '_app/immutable/chunks/fixtures.js',
        routes: ['src/routes/dev/gallery/+page.svelte']
      },
      {
        file: '_app/immutable/chunks/flags.js',
        routes: ['src/routes/dev/+layout.ts', 'src/routes/dev/gallery/+page.svelte']
      },
      { file: '_app/immutable/chunks/leafart.js', routes: ['src/routes/dev/gallery/+page.svelte'] },
      { file: '_app/immutable/chunks/leaves.js', routes: ['src/routes/dev/gallery/+page.svelte'] },
      { file: '_app/immutable/nodes/2.js', routes: ['src/routes/dev/+layout.ts'] },
      { file: '_app/immutable/nodes/4.js', routes: ['src/routes/dev/gallery/+page.svelte'] }
    ]);
  });

  it('keeps everything a real route loads, statically or lazily, even when a dev page shares it', () => {
    const devOnly = new Set(devOnlyFiles(manifest(), read).map((entry) => entry.file));
    for (const shared of [
      '_app/immutable/chunks/runtime.js',
      '_app/immutable/chunks/card.js',
      '_app/immutable/assets/card.css',
      '_app/immutable/chunks/lazy.js',
      '_app/immutable/assets/inter.woff2',
      '_app/immutable/entry/app.js',
      '_app/immutable/nodes/5.js'
    ])
      expect(devOnly.has(shared), shared).toBe(false);
  });

  it('keeps a chunk a real route reaches only lazily through a chunk it shares with a dev page', () => {
    const m = manifest();
    m['src/lib/game/lazy.ts']!.dynamicImports = ['src/lib/dev/leaves.ts'];
    const devOnly = devOnlyFiles(m, read).map((entry) => entry.file);
    expect(devOnly).not.toContain('_app/immutable/chunks/leaves.js');
    expect(devOnly).not.toContain('_app/immutable/chunks/fixtures.js');
    expect(devOnly).toContain('_app/immutable/nodes/4.js');
  });

  it('does not count the app script, which imports every node lazily, as reaching the dev nodes', () => {
    const devOnly = devOnlyFiles(manifest(), read).map((entry) => entry.file);
    expect(devOnly).toContain('_app/immutable/nodes/2.js');
    expect(devOnly).toContain('_app/immutable/nodes/4.js');
  });

  it('reads only route nodes, and a route outside src/routes/dev/ is a real route', () => {
    const asked: string[] = [];
    devOnlyFiles(manifest(), (key) => {
      asked.push(key);
      return read(key);
    });
    expect(asked.sort()).toEqual([0, 1, 2, 3, 4, 5].map(node));
    expect(DEV_ROUTES).toBe('src/routes/dev/');
  });

  it('fails on a manifest that imports a key it does not have', () => {
    const m = manifest();
    m[node(3)]!.imports = ['_missing.js'];
    expect(() => devOnlyFiles(m, read)).toThrow(/_missing\.js/);
  });
});

describe('staticSpecifiers', () => {
  it("finds the route files in SvelteKit's generated node modules", () => {
    expect(staticSpecifiers(read(node(0)))).toEqual([
      `${ROUTES}/+layout.ts`,
      `${ROUTES}/+layout.svelte`
    ]);
    expect(staticSpecifiers(`export { default as component } from '${ROUTES}/x.svelte'`)).toEqual([
      `${ROUTES}/x.svelte`
    ]);
  });
});
