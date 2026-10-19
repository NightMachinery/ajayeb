// Import the original vector sources once at build time. Each rendered icon
// gets its own shapes, avoiding a document-wide sprite and SVG use shadow trees.
import { ICON_NAMES, type IconName } from './names';

const sources = import.meta.glob('./svg/*.svg', {
  query: '?raw',
  import: 'default',
  eager: true
}) as Record<string, string>;

function nameFromPath(path: string): string {
  const match = /\/([^/]+)\.svg$/.exec(path);
  if (!match?.[1]) throw new Error(`icons/sources: unexpected glob path "${path}"`);
  return match[1];
}

/** Pulls the viewBox and the markup between the opening and closing <svg> tags. */
function parseSource(path: string, raw: string): { viewBox: string; inner: string } {
  const open = /<svg\b[^>]*>/.exec(raw);
  const closeAt = raw.lastIndexOf('</svg>');
  if (!open || closeAt < 0) {
    throw new Error(`icons/sources: "${path}" is not a standalone <svg>...</svg> document`);
  }
  const viewBox = /viewBox="([^"]+)"/.exec(open[0])?.[1];
  if (!viewBox) throw new Error(`icons/sources: "${path}" has no viewBox`);
  const inner = raw.slice(open.index + open[0].length, closeAt).trim();
  return { viewBox, inner };
}

const bySource = new Map<string, string>();
for (const [path, raw] of Object.entries(sources)) {
  bySource.set(nameFromPath(path), raw);
}

const missing = ICON_NAMES.filter((name) => !bySource.has(name));
if (missing.length > 0) {
  throw new Error(`icons/sources: no svg source for: ${missing.join(', ')}`);
}
const extra = [...bySource.keys()].filter(
  (name) => !(ICON_NAMES as readonly string[]).includes(name)
);
if (extra.length > 0) {
  throw new Error(`icons/sources: svg source(s) not listed in names.ts: ${extra.join(', ')}`);
}

export interface IconSource {
  viewBox: string;
  inner: string;
}

export const iconSources = Object.fromEntries(
  ICON_NAMES.map((name) => {
    const raw = bySource.get(name);
    if (!raw) throw new Error(`icons/sources: missing "${name}"`);
    return [name, parseSource(name, raw)];
  })
) as Record<IconName, IconSource>;

/** Original markup only, never user or server content. Scope local paint ids
 *  to a component instance so repeated clipped icons cannot reference each other. */
export function iconMarkup(name: IconName, scope: string): string {
  const { inner } = iconSources[name];
  if (!inner.includes('id="')) return inner;
  // Hex code points produce a URL-safe, unambiguous encoding of Svelte's id.
  const prefix = `icon-${Array.from(scope, (c) => c.codePointAt(0)!.toString(16)).join('-')}-`;
  return inner
    .replace(/\bid="([^"]+)"/g, (_, id: string) => `id="${prefix}${id}"`)
    .replace(/url\(#([^)]+)\)/g, (_, id: string) => `url(#${prefix}${id})`);
}

/** Keep the original source set inside its existing 40 KB raw budget. */
export const iconSourceByteLength = new TextEncoder().encode(
  [...bySource.values()].join('')
).byteLength;

export { ICON_NAMES, type IconName };
