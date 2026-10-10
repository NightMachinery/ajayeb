import { describe, expect, it } from 'vitest';
import { ICON_NAMES, iconSpriteByteLength, iconSpriteMarkup } from './sprite';

// Importing sprite.ts already throws if a name in names.ts has no source
// file, or a source file is missing from names.ts (see sprite.ts), so a
// successful import is itself a pass of that consistency check.

describe('icon sprite', () => {
  it('stays within the 40 KB raw budget', () => {
    expect(iconSpriteByteLength).toBeLessThanOrEqual(40 * 1024);
  });

  it('has one <symbol> per name in ICON_NAMES, each with a viewBox', () => {
    for (const name of ICON_NAMES) {
      const symbol = new RegExp(`<symbol id="i-${name}" viewBox="[^"]+">`).exec(iconSpriteMarkup);
      expect(symbol, `missing or malformed <symbol> for "${name}"`).not.toBeNull();
    }
  });

  it('never embeds <text>, raster images, filters or gradients', () => {
    expect(iconSpriteMarkup).not.toMatch(
      /<text\b|<image\b|<filter\b|<linearGradient\b|<radialGradient\b/
    );
  });

  it('has no duplicate names', () => {
    expect(new Set(ICON_NAMES).size).toBe(ICON_NAMES.length);
  });
});
