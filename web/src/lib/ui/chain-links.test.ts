import { describe, expect, it } from 'vitest';
import { cardBySlug, cardSlugs } from '$lib/catalog';
import type { Chain } from '$lib/wire';
import { chainSource, chainTarget, chainTooltip } from './chain-links';

const CHAINS = [
  ...new Set(cardSlugs.flatMap((slug) => [cardBySlug(slug)?.chainOut].filter(Boolean)))
] as Chain[];

describe('chain links', () => {
  it('links every chain symbol to the cards the catalog names', () => {
    expect(CHAINS.length).toBeGreaterThan(10);
    for (const chain of CHAINS) {
      const source = chainSource(chain);
      const target = chainTarget(chain);
      expect(source, chain).toBeDefined();
      expect(target, chain).toBeDefined();
      expect(cardBySlug(source!)?.chainTo).toBe(target);
      expect(cardBySlug(target!)?.chainFrom).toBe(source);
    }
  });

  it('Baths (drop) makes Aqueduct free, and Aqueduct stacks Baths', () => {
    const out = chainTooltip('drop', 'out');
    expect(out.text).toContain('Aqueduct');
    expect(out.stack).toEqual([expect.objectContaining({ kind: 'card', slug: 'aqueduct' })]);
    const into = chainTooltip('drop', 'in');
    expect(into.text).toContain('Baths');
    expect(into.stack).toEqual([expect.objectContaining({ kind: 'card', slug: 'baths' })]);
    expect(out.rulesId).toBe('chains');
  });
});
