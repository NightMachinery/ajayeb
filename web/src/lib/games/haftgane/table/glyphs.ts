// The vector drawings the table screen draws: Haftgane's own
// (../icons/svg/) and the Duel drawings it reuses by reference
// (docs/haftgane/art-reuse.md; the vector lint refuses copies).
//
// The Duel drawings load under their own module id (the extra `hg` query),
// so they join Haftgane's lazy chunk while Duel's chunks keep their own
// copies untouched: importing the very ids Duel's icon sources import would
// make the bundler move them into a chunk both games share, which changes
// Duel's bytes. Vite reads any query holding `raw` as a raw import.

const own = import.meta.glob<string>('../icons/svg/*.svg', {
  query: '?raw',
  import: 'default',
  eager: true
});

const duel = import.meta.glob<string>(
  [
    '../../../icons/svg/coin.svg',
    '../../../icons/svg/shield.svg',
    '../../../icons/svg/vp-laurel.svg',
    '../../../icons/svg/wonder.svg',
    '../../../icons/svg/trade.svg',
    '../../../icons/svg/choice.svg',
    '../../../icons/svg/badge-check.svg',
    '../../../icons/svg/badge-lock.svg',
    '../../../icons/svg/badge-chain.svg',
    '../../../icons/svg/build-from-discard.svg',
    '../../../icons/svg/resource-*.svg',
    '../../../icons/svg/category-*.svg',
    '../../../icons/svg/chain-{barrel,book,column,drop,helmet,horseshoe,lamp,lyre,mask,target,tower}.svg',
    '../../../icons/svg/science-{mortar,scales,quill}.svg'
  ],
  { query: '?hg&raw', import: 'default', eager: true }
);

export interface GlyphSource {
  viewBox: string;
  inner: string;
}

function parse(raw: string): GlyphSource {
  const open = /<svg\b[^>]*>/.exec(raw);
  const viewBox = open ? (/viewBox="([^"]+)"/.exec(open[0])?.[1] ?? '0 0 24 24') : '0 0 24 24';
  const start = open ? open.index + open[0].length : 0;
  return { viewBox, inner: raw.slice(start, raw.lastIndexOf('</svg>')).trim() };
}

const sources = new Map<string, GlyphSource>();
for (const [path, raw] of [...Object.entries(duel), ...Object.entries(own)]) {
  const name = /\/([^/]+)\.svg$/.exec(path)?.[1];
  if (name) sources.set(name, parse(raw));
}

/** A drawing by file name, or undefined when there is none. */
export function glyphSource(name: string): GlyphSource | undefined {
  return sources.get(name);
}

/** Every drawing's name, for the gallery and the tests. */
export function glyphNames(): string[] {
  return [...sources.keys()].sort();
}

/** Original markup only; paint ids scoped to one instance, as Icon.svelte does. */
export function glyphMarkup(source: GlyphSource, scope: string): string {
  if (!source.inner.includes('id="')) return source.inner;
  const prefix = `hg-${Array.from(scope, (c) => c.codePointAt(0)!.toString(16)).join('-')}-`;
  return source.inner
    .replace(/\bid="([^"]+)"/g, (_, id: string) => `id="${prefix}${id}"`)
    .replace(/url\(#([^)]+)\)/g, (_, id: string) => `url(#${prefix}${id})`);
}
