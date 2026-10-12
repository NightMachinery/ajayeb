// The table textures are the one place the app names bitmap art in CSS
// rather than through the generated index (tokens.css --table-texture, drawn
// on the page in app.css). These tests hold them to the same rule as every
// other frame: only approved, shipped art, one tile per theme.
import { readFileSync } from 'node:fs';
import { describe, expect, it } from 'vitest';
import { art } from './available.gen';

const tokens = readFileSync(new URL('../styles/tokens.css', import.meta.url), 'utf8');
const appCss = readFileSync(new URL('../../app.css', import.meta.url), 'utf8');

/** The value of a custom property in the first rule whose selector matches. */
function declared(selector: string, prop: string): string | undefined {
  const at = tokens.indexOf(`${selector} {`);
  if (at < 0) return undefined;
  const block = tokens.slice(at, tokens.indexOf('\n}', at));
  return new RegExp(`${prop}:\\s*([^;]+);`).exec(block)?.[1]?.trim();
}

// CSS names each tile by the index's own path, ?v= content hash included,
// so the server caches it as immutable and a reprocessed tile reaches every
// reader at once (docs/art/bitmaps.md "Caching"); a stale hash fails here.
const shipped = new Map<string, string>();
for (const [id, entry] of Object.entries(art)) {
  for (const v of Object.values(entry.variants)) shipped.set(v.path, id);
}
/** The index's path of a texture's one shipped file. */
const tile = (id: string) => Object.values(art[id]?.variants ?? {})[0]?.path;

describe('table textures', () => {
  it('names only approved texture art', () => {
    const urls = [...tokens.matchAll(/url\(\s*['"]?([^'")]+)['"]?\s*\)/g)].map((m) => m[1] ?? '');
    expect(urls.length).toBeGreaterThan(0);
    for (const url of urls) {
      expect(shipped.get(url), url).toMatch(/^texture\//);
    }
  });

  it('gives the light theme the paper tile and the dark theme the night tile', () => {
    expect(tile('texture/paper')).toMatch(/^\/art\/ui\/paper\.avif\?v=[0-9a-f]+$/);
    expect(tile('texture/night')).toMatch(/^\/art\/ui\/night\.avif\?v=[0-9a-f]+$/);
    expect(declared(':root', '--table-texture')).toBe(`url('${tile('texture/paper')}')`);
    expect(declared(":root[data-theme='dark']", '--table-texture')).toBe(
      `url('${tile('texture/night')}')`
    );
  });

  it('draws the texture once, on the page, at the tile size', () => {
    const uses = appCss.match(/var\(--table-texture\)/g) ?? [];
    expect(uses).toHaveLength(1);
    expect(appCss).toMatch(/background-size:\s*auto,\s*512px 512px/);
  });
});
