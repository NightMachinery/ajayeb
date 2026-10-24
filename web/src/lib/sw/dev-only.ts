// Which build files only the /dev pages load, so that the service worker can
// leave them out of its install precache. Pure, so vitest checks it on small
// manifests (dev-only.test.ts); vite.config.ts runs it on each build and
// writes the result into the built worker, and the shell e2e runs it again
// to check the installed cache.
//
// The input is Vite's client manifest, which is also where SvelteKit takes
// the worker's `build` list from, so both name the same files. Its entries
// are SvelteKit's two entry scripts and its route nodes (one per page,
// layout or error component). A route node is a dev node when the module
// SvelteKit generated for it names a file under src/routes/dev/. A file is
// dev-only when a dev node reaches it and no other entry does, so a chunk a
// dev page shares with a real route stays precached. Reaching follows
// static and dynamic imports and collects each chunk's CSS and assets, but
// never enters another entry: the app script imports every node lazily, and
// each node is judged on its own.

/** In the source, the worker's placeholder for the list. vite.config.ts
 *  replaces the string literal holding it with the list as JSON. */
export const DEV_ONLY_MARK = 'ajayeb:dev-only-build-files';

/** The routes directory of the dev pages, relative to the web root. */
export const DEV_ROUTES = 'src/routes/dev/';

/** One entry of Vite's client manifest (.vite/manifest.json). */
export interface ManifestChunk {
  file: string;
  isEntry?: boolean;
  /** Not read: a lazy chunk is reached through its importers. */
  isDynamicEntry?: boolean;
  imports?: string[];
  dynamicImports?: string[];
  css?: string[];
  assets?: string[];
}

/** Vite's client manifest, keyed by source path relative to the web root. */
export type Manifest = Record<string, ManifestChunk>;

/** A dev-only file and the dev route files whose nodes load it. */
export interface DevOnlyFile {
  file: string;
  routes: string[];
}

/** The generated route node modules, under SvelteKit's output directory. */
const NODE = /(^|\/)generated\/client-optimized\/nodes\/\d+\.js$/;

/** The specifiers of a module's `import ... from` and `export ... from`
 *  statements: SvelteKit generates each node as a few of these. */
export function staticSpecifiers(source: string): string[] {
  return [...source.matchAll(/\bfrom\s*(["'])(.+?)\1/g)].map((match) => match[2] ?? '');
}

/** `spec` resolved against the directory of `key`, both relative to the
 *  web root. A path that climbs above the root loses its leading `..`s,
 *  which is harmless here: such files are never under DEV_ROUTES. */
function resolveFrom(key: string, spec: string): string {
  const parts = key.split('/').slice(0, -1);
  for (const part of spec.split('/')) {
    if (part === '..') parts.pop();
    else if (part !== '.' && part !== '') parts.push(part);
  }
  return parts.join('/');
}

/** The files a route node names under DEV_ROUTES; empty for any other
 *  entry. `read` returns a manifest key's generated source. */
function devRouteFiles(key: string, read: (key: string) => string): string[] {
  if (!NODE.test(key)) return [];
  return staticSpecifiers(read(key))
    .map((spec) => resolveFrom(key, spec))
    .filter((file) => file.startsWith(DEV_ROUTES));
}

/** Every file the entry `root` loads: its chunks, their CSS and assets. */
function reach(manifest: Manifest, root: string): Set<string> {
  const files = new Set<string>();
  const seen = new Set([root]);
  const stack = [root];
  for (let key = stack.pop(); key !== undefined; key = stack.pop()) {
    const chunk = manifest[key];
    if (!chunk) throw new Error(`build manifest: ${key} is imported but has no entry`);
    files.add(chunk.file);
    for (const file of [...(chunk.css ?? []), ...(chunk.assets ?? [])]) files.add(file);
    for (const next of [...(chunk.imports ?? []), ...(chunk.dynamicImports ?? [])]) {
      if (seen.has(next) || manifest[next]?.isEntry) continue;
      seen.add(next);
      stack.push(next);
    }
  }
  return files;
}

/** The build files only dev nodes load, sorted, each with the dev route
 *  files that load it. `read` returns a route node's generated source. */
export function devOnlyFiles(manifest: Manifest, read: (key: string) => string): DevOnlyFile[] {
  const app = new Set<string>();
  const dev = new Map<string, Set<string>>();
  for (const [key, chunk] of Object.entries(manifest)) {
    if (!chunk.isEntry) continue;
    const routes = devRouteFiles(key, read);
    for (const file of reach(manifest, key)) {
      if (routes.length === 0) app.add(file);
      else for (const route of routes) dev.set(file, (dev.get(file) ?? new Set()).add(route));
    }
  }
  return [...dev]
    .filter(([file]) => !app.has(file))
    .map(([file, routes]) => ({ file, routes: [...routes].sort() }))
    .sort((a, b) => a.file.localeCompare(b.file));
}
