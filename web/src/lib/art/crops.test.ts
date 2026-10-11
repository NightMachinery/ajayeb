import { describe, expect, it } from 'vitest';
import { wonderSlugs } from '$lib/catalog';
import { wonderArtCrop } from './crops';

describe('wonder art crops', () => {
  it('anchors the Colossus and the Mausoleum near the top, with the sky fade', () => {
    for (const slug of ['colossus', 'mausoleum']) {
      const crop = wonderArtCrop(slug);
      expect(crop.feather, slug).toBe('sky');
      const y = Number(/^50% (\d+)%$/.exec(crop.position)?.[1]);
      expect(y, slug).toBeLessThan(25);
    }
  });

  it('centres every other wonder with the soft fade', () => {
    for (const slug of wonderSlugs.filter((s) => s !== 'colossus' && s !== 'mausoleum')) {
      expect(wonderArtCrop(slug), slug).toEqual({ position: '50% 50%', feather: 'soft' });
    }
  });

  it('only names real wonders', () => {
    expect(wonderArtCrop('constructor')).toEqual({ position: '50% 50%', feather: 'soft' });
  });
});
