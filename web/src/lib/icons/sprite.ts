// Builds one inline SVG sprite from the hand-authored sources in ./svg/*.svg,
// each a standalone <svg viewBox="...">...</svg> file. This module turns
// every source into a <symbol id="i-<name>"> and exports the sprite markup as
// a string, so +layout.svelte can inline it once (see IconSprite.svelte).
// Icon.svelte then references a symbol with <use href="#i-<name>"/>.
//
// `?raw` makes Vite inline each file's text at build time; `eager: true`
// avoids a dynamic import per icon, since the whole set is small (see the
// budget check below).

import { ICON_NAMES, type IconName } from './names';

const sources = import.meta.glob('./svg/*.svg', {
  query: '?raw',
  import: 'default',
  eager: true
}) as Record<string, string>;

function nameFromPath(path: string): string {
  const match = /\/([^/]+)\.svg$/.exec(path);
  if (!match?.[1]) throw new Error(`icons/sprite: unexpected glob path "${path}"`);
  return match[1];
}

/** Pulls the viewBox and the markup between the opening and closing <svg> tags. */
function parseSource(path: string, raw: string): { viewBox: string; inner: string } {
  const open = /<svg\b[^>]*>/.exec(raw);
  const closeAt = raw.lastIndexOf('</svg>');
  if (!open || closeAt < 0) {
    throw new Error(`icons/sprite: "${path}" is not a standalone <svg>...</svg> document`);
  }
  const viewBox = /viewBox="([^"]+)"/.exec(open[0])?.[1];
  if (!viewBox) throw new Error(`icons/sprite: "${path}" has no viewBox`);
  const inner = raw.slice(open.index + open[0].length, closeAt).trim();
  return { viewBox, inner };
}

const bySource = new Map<string, string>();
for (const [path, raw] of Object.entries(sources)) {
  bySource.set(nameFromPath(path), raw);
}

const missing = ICON_NAMES.filter((name) => !bySource.has(name));
if (missing.length > 0) {
  throw new Error(`icons/sprite: no svg source for: ${missing.join(', ')}`);
}
const extra = [...bySource.keys()].filter(
  (name) => !(ICON_NAMES as readonly string[]).includes(name)
);
if (extra.length > 0) {
  throw new Error(`icons/sprite: svg source(s) not listed in names.ts: ${extra.join(', ')}`);
}

const symbols = ICON_NAMES.map((name) => {
  const raw = bySource.get(name);
  if (!raw) throw new Error(`icons/sprite: unreachable, missing "${name}"`); // guarded above
  const { viewBox, inner } = parseSource(name, raw);
  return `<symbol id="i-${name}" viewBox="${viewBox}">${inner}</symbol>`;
}).join('');

/** The whole sprite as one hidden, inert <svg>: every icon as a <symbol>. */
export const iconSpriteMarkup = `<svg aria-hidden="true" focusable="false" style="position:absolute;width:0;height:0;overflow:hidden" xmlns="http://www.w3.org/2000/svg">${symbols}</svg>`;

/** Raw byte size of the sprite markup, for the 40 KB budget (see icons.test.ts). */
export const iconSpriteByteLength = new TextEncoder().encode(iconSpriteMarkup).byteLength;

export { ICON_NAMES, type IconName };
