import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
  compilerOptions: {
    // Runes mode everywhere in our code. Libraries in node_modules decide for
    // themselves. This can go once Svelte makes runes the only mode.
    runes: ({ filename }) => (filename.split(/[/\\]/).includes('node_modules') ? undefined : true)
  },
  kit: {
    // A static single-page app. Every unknown path gets index.html, and the
    // client router takes over. The Go server embeds this build and serves
    // index.html for any path that is not a file, /ws, /api or /health.
    adapter: adapter({ fallback: 'index.html' }),
    // src/service-worker.ts, built to /service-worker.js and registered by
    // SvelteKit. `files` decides which files of static/ it precaches: only
    // the stable-named ones a home screen or the shell needs. Art is cached
    // as it is shown, never precached (D24), and the font licences are
    // documents, not app files.
    serviceWorker: {
      files: (file) =>
        file === 'favicon.svg' || file === 'manifest.webmanifest' || file.startsWith('icons/')
    }
  }
};

export default config;
