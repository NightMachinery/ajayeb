import { describe, expect, it } from 'vitest';
import type { View } from '$lib/wire';
import { protocolFixtures } from '$lib/wire/fixtures.gen';
import {
  cardBySlug,
  cardSlugs,
  catalogHash,
  catalogMatches,
  isCardSlug,
  tokenBySlug,
  tokenSlugs,
  wonderBySlug,
  wonderSlugs
} from './index';

describe('catalog accessors', () => {
  it('finds a card, a wonder and a token by slug', () => {
    const lumber = cardBySlug('lumber-yard');
    expect(lumber).toMatchObject({
      id: 1,
      slug: 'lumber-yard',
      age: 1,
      color: 'brown',
      back: 'age1'
    });
    expect(lumber?.effects).toEqual([{ kind: 'produce', n: 1, resource: 'wood' }]);
    expect(wonderBySlug('pyramids')).toMatchObject({ id: 1, slug: 'pyramids' });
    expect(tokenBySlug('agriculture')).toMatchObject({ id: 1, slug: 'agriculture' });
  });

  it('returns undefined for unknown slugs and inherited keys', () => {
    for (const slug of ['', 'nope', 'constructor', '__proto__', 'toString', 'pyramids']) {
      expect(cardBySlug(slug), slug).toBeUndefined();
    }
    expect(wonderBySlug('lumber-yard')).toBeUndefined();
    expect(tokenBySlug('constructor')).toBeUndefined();
    expect(isCardSlug('hasOwnProperty')).toBe(false);
  });

  it('holds the whole base game, ids unique and in order', () => {
    expect(cardSlugs).toHaveLength(73);
    expect(wonderSlugs).toHaveLength(12);
    expect(tokenSlugs).toHaveLength(10);
    expect(cardSlugs.map((s) => cardBySlug(s)?.id)).toEqual(cardSlugs.map((_, i) => i + 1));
    expect(wonderSlugs.map((s) => wonderBySlug(s)?.id)).toEqual(wonderSlugs.map((_, i) => i + 1));
    expect(tokenSlugs.map((s) => tokenBySlug(s)?.id)).toEqual(tokenSlugs.map((_, i) => i + 1));
  });

  it('links chains both ways', () => {
    let links = 0;
    for (const slug of cardSlugs) {
      const card = cardBySlug(slug)!;
      if (card.chainTo) {
        links++;
        const target = cardBySlug(card.chainTo);
        expect(target?.chainFrom, slug).toBe(slug);
        expect(target?.chainIn, slug).toBe(card.chainOut);
      }
      if (card.chainFrom) expect(cardBySlug(card.chainFrom)?.chainTo, slug).toBe(slug);
    }
    expect(links).toBeGreaterThan(10);
  });

  it('gives guilds their own back and colour', () => {
    const guilds = cardSlugs.map((s) => cardBySlug(s)!).filter((c) => c.back === 'guild');
    expect(guilds).toHaveLength(7);
    for (const g of guilds) {
      expect(g.color).toBe('purple');
      expect(g.age).toBe(3);
    }
  });

  it('matches the hash the server puts in every view', () => {
    expect(catalogHash).toMatch(/^[0-9a-f]{16}$/);
    for (const view of protocolFixtures) expect(catalogMatches(view.catalogHash)).toBe(true);
    expect(catalogMatches('0000000000000000')).toBe(false);
  });

  it('knows every card, wonder and token the recorded views mention', () => {
    for (const view of protocolFixtures as readonly View[]) {
      for (const slot of view.structure) if (slot.card) expect(cardBySlug(slot.card)).toBeDefined();
      for (const p of view.players) {
        for (const c of p.city) expect(cardBySlug(c), c).toBeDefined();
        for (const w of p.wonders) expect(wonderBySlug(w.wonder), w.wonder).toBeDefined();
        for (const tok of p.tokens) expect(tokenBySlug(tok), tok).toBeDefined();
      }
      for (const tok of view.boardTokens) expect(tokenBySlug(tok), tok).toBeDefined();
    }
  });
});
