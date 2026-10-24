// Every design and board of the catalog has an English name, keyed by its
// slug, and the table names nothing the catalog lacks.
import { describe, expect, it } from 'vitest';
import { hgBoardSlugs, hgDesignSlugs } from '$lib/wire/haftgane-catalog.gen';
import { names } from './en/names';

describe('Haftgane names', () => {
  it('names every design and board of the catalog, and nothing else', () => {
    const cards = Object.keys(names).filter((k) => k.startsWith('hg.card.'));
    expect(cards).toEqual(hgDesignSlugs.map((s) => `hg.card.${s}`));
    for (const s of hgBoardSlugs) {
      expect(names).toHaveProperty([`hg.board.${s}`]);
      expect(names).toHaveProperty([`hg.wonder.${s}`]);
    }
  });
});
