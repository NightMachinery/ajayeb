import adapter from '@sveltejs/adapter-static';
import { execFileSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import { readFileSync, readdirSync } from 'node:fs';

// Clean commits use their revision; dirty trees and source archives also hash
// shipped inputs so a changed build always gets a changed version.
function contentVersion() {
  const hash = createHash('sha256');
  for (const root of ['src', 'static']) {
    const base = new URL(`./${root}/`, import.meta.url);
    for (const file of readdirSync(base, { recursive: true, withFileTypes: true })
      .filter((file) => file.isFile())
      .map((file) => new URL(`${file.parentPath}/${file.name}`, 'file:'))
      .sort((a, b) => a.href.localeCompare(b.href))) {
      hash.update(file.href.slice(base.href.length));
      hash.update(readFileSync(file));
    }
  }
  for (const file of [
    'package.json',
    'svelte.config.js',
    'vite.config.ts',
    '../pnpm-lock.yaml',
    'tsconfig.json'
  ])
    hash.update(readFileSync(new URL(file, import.meta.url)));
  return hash.digest('hex');
}

function buildVersion() {
  try {
    const git = (args) =>
      execFileSync('git', args, {
        cwd: new URL('.', import.meta.url),
        stdio: ['ignore', 'pipe', 'ignore'],
        encoding: 'utf8'
      }).trim();
    const revision = git(['rev-parse', 'HEAD']);
    return git(['status', '--porcelain', '--', '.']) ? `${revision}-${contentVersion()}` : revision;
  } catch {
    return contentVersion();
  }
}

/** @type {import('@sveltejs/kit').Config} */
const config = {
  compilerOptions: {
    // Runes mode everywhere in our code. Libraries in node_modules decide for
    // themselves. This can go once Svelte makes runes the only mode.
    runes: ({ filename }) => (filename.split(/[/\\]/).includes('node_modules') ? undefined : true)
  },
  kit: {
    version: { name: buildVersion() },
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
