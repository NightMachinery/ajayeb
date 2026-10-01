import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import type { ProxyOptions } from 'vite';
import { defineConfig } from 'vitest/config';

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

export default defineConfig({
  plugins: [tailwindcss(), sveltekit()],
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
