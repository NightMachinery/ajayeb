import { describe, expect, it } from 'vitest';
import { ICON_NAMES, iconMarkup, iconSourceByteLength, iconSources } from './sources';

describe('original icon sources', () => {
  it('keeps the complete source set within the 40 KB raw budget', () => {
    expect(iconSourceByteLength).toBeLessThanOrEqual(40 * 1024);
    expect(new Set(ICON_NAMES).size).toBe(ICON_NAMES.length);
  });

  it('preserves each source viewBox and paint without use shadow trees', () => {
    for (const name of ICON_NAMES) {
      const source = iconSources[name];
      expect(source.viewBox).toMatch(/^0 0 \d+ \d+$/);
      expect(source.inner).not.toMatch(
        /<use\b|<text\b|<image\b|<filter\b|<linearGradient\b|<radialGradient\b/
      );
      const unscoped = iconMarkup(name, 'a').replace(/icon-61-/g, '');
      expect(unscoped).toBe(source.inner);
    }
  });

  it('gives repeated clipped icons distinct, internally complete paint references', () => {
    const clipped = ICON_NAMES.filter((name) => iconSources[name].inner.includes('id="'));
    expect(clipped).toHaveLength(5);
    for (const name of clipped) {
      const first = iconMarkup(name, ':a');
      const second = iconMarkup(name, 'a');
      const ids = (markup: string) => [...markup.matchAll(/\bid="([^"]+)"/g)].map((m) => m[1]);
      expect(ids(first).some((id) => ids(second).includes(id))).toBe(false);
      for (const markup of [first, second]) {
        const references = [...markup.matchAll(/url\(#([^)]+)\)/g)].map((m) => m[1]);
        expect(references.length).toBeGreaterThan(0);
        for (const reference of references) expect(ids(markup)).toContain(reference);
      }
    }
  });
});
