import { readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import type { Plugin, ProxyOptions } from 'vite';
import { defineConfig } from 'vitest/config';
import { DEV_ONLY_MARK, devOnlyFiles, type Manifest } from './src/lib/sw/dev-only.ts';

// In development the Go server (server/cmd/ajayeb) listens on 127.0.0.1:38951.
// Vite forwards the WebSocket, the HTTP API and the health check to it, so the
// browser only ever talks to the Vite origin. AJAYEB_SERVER points elsewhere.
const target = process.env.AJAYEB_SERVER ?? 'http://127.0.0.1:38951';

// No changeOrigin: the server then sees the Host the page was loaded from,
// which is what its WebSocket origin check compares the Origin header with.
const proxy: Record<string, ProxyOptions> = {
  '/ws': { target, ws: true },
  '/api': { target },
  '/health': { target }
};

// The service worker leaves the /dev pages' own build files out of its
// install precache (src/lib/sw/dev-only.ts; web/README.md, "App shell").
// SvelteKit builds the client and then the worker inside the server build's
// writeBundle, so this hook, placed after SvelteKit's, reads the client
// manifest and the generated route nodes, and writes the list into the
// built worker before the adapter copies the output to build/.
function devPagesOutOfPrecache(): Plugin {
  let root = '';
  let server = false;
  return {
    name: 'ajayeb-dev-pages-out-of-precache',
    apply: 'build',
    configResolved(config) {
      root = config.root;
      server = Boolean(config.build.ssr);
    },
    writeBundle: {
      sequential: true,
      handler() {
        // SvelteKit's nested client build loads this config too.
        if (!server) return;
        const client = join(root, '.svelte-kit/output/client');
        const manifest = JSON.parse(
          readFileSync(join(client, '.vite/manifest.json'), 'utf8')
        ) as Manifest;
        const files = devOnlyFiles(manifest, (key) => readFileSync(join(root, key), 'utf8')).map(
          (entry) => entry.file
        );
        const worker = join(client, 'service-worker.js');
        const code = readFileSync(worker, 'utf8');
        const mark = new RegExp(`(["'\`])${DEV_ONLY_MARK}\\1`, 'g');
        const found = code.match(mark)?.length ?? 0;
        if (found !== 1)
          throw new Error(`service-worker.js holds ${found} copies of ${DEV_ONLY_MARK}, not 1`);
        writeFileSync(
          worker,
          code.replace(mark, () => JSON.stringify(JSON.stringify(files)))
        );
      }
    }
  };
}

export default defineConfig({
  plugins: [tailwindcss(), sveltekit(), devPagesOutOfPrecache()],
  server: { port: 38952, strictPort: true, proxy },
  preview: { port: 38953, strictPort: true, proxy },
  build: { reportCompressedSize: false },
  test: {
    include: ['src/**/*.test.ts'],
    environment: 'node',
    // One worker thread keeps the memory footprint small (docs/dev/ram-policy.md).
    pool: 'threads',
    maxWorkers: 1
  }
});
