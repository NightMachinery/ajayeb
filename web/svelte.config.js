import adapter from '@sveltejs/adapter-static';

/** @type {import('@sveltejs/kit').Config} */
const config = {
  compilerOptions: {
    // Runes mode everywhere in our code. Libraries in node_modules decide for
    // themselves. This can go once Svelte makes runes the only mode.
    runes: ({ filename }) => (filename.split(/[/\\]/).includes('node_modules') ? undefined : true)
  },
  kit: {
    // Home prerenders to index.html. Other routes use a separate generic
    // client shell, so a game deep link never hydrates a Home document.
    adapter: adapter({ fallback: 'shell.html' }),
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
