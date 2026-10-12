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

const shipped = new Map<string, string>();
for (const [id, entry] of Object.entries(art)) {
  // CSS names the file without the index's ?v= content hash (the server
  // then revalidates it, see docs/art/bitmaps.md "Caching").
  for (const v of Object.values(entry.variants)) shipped.set(v.path.split('?')[0] ?? '', id);
}

describe('table textures', () => {
  it('names only approved texture art', () => {
    const urls = [...tokens.matchAll(/url\(\s*['"]?([^'")]+)['"]?\s*\)/g)].map((m) => m[1] ?? '');
    expect(urls.length).toBeGreaterThan(0);
    for (const url of urls) {
      expect(shipped.get(url), url).toMatch(/^texture\//);
    }
  });

  it('gives the light theme the paper tile and the dark theme the night tile', () => {
    expect(declared(':root', '--table-texture')).toBe("url('/art/ui/paper.avif')");
    expect(shipped.get('/art/ui/paper.avif')).toBe('texture/paper');
    expect(declared(":root[data-theme='dark']", '--table-texture')).toBe(
      "url('/art/ui/night.avif')"
    );
    expect(shipped.get('/art/ui/night.avif')).toBe('texture/night');
  });

  it('draws the texture once, on the page, at the tile size', () => {
    const uses = appCss.match(/var\(--table-texture\)/g) ?? [];
    expect(uses).toHaveLength(1);
    expect(appCss).toMatch(/background-size:\s*auto,\s*512px 512px/);
  });
});
