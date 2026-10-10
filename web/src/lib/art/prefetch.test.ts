import { describe, expect, it } from 'vitest';
import { cardBySlug } from '$lib/catalog';
import type { ArtIndex } from './lookup';
import {
  nextAgeArtIds,
  nextBacks,
  PREFETCH_BUDGET,
  prefetchImages,
  prefetchPlan
} from './prefetch';

function cardEntry(slug: string, thumbBytes: number) {
  return {
    kind: 'card',
    variants: {
      default: { path: `/art/cards/${slug}.avif`, width: 480, height: 360, bytes: 30000 },
      thumb: { path: `/art/cards/${slug}@thumb.avif`, width: 240, height: 180, bytes: thumbBytes }
    }
  };
}

describe('nextAgeArtIds', () => {
  it('follows the draft with Age I, Age I with Age II, Age II with Age III and the guilds', () => {
    expect(nextBacks(0)).toEqual(['age1']);
    expect(nextBacks(1)).toEqual(['age2']);
    expect(nextBacks(2)).toEqual(['age3', 'guild']);
    expect(nextBacks(3)).toEqual([]);
  });

  it('lists the next age’s backs and cards, and nothing after Age III', () => {
    const age1 = nextAgeArtIds(1);
    expect(age1.backs).toEqual(['back/age2']);
    expect(age1.cards).toHaveLength(23);
    for (const id of age1.cards) {
      expect(cardBySlug(id.slice('card/'.length))?.back).toBe('age2');
    }
    const age2 = nextAgeArtIds(2);
    expect(age2.backs).toEqual(['back/age3', 'back/guild']);
    expect(age2.cards).toHaveLength(27);
    expect(nextAgeArtIds(3)).toEqual({ backs: [], cards: [] });
  });
});

describe('prefetchPlan', () => {
  it('is empty when no art is approved', () => {
    for (const age of [0, 1, 2, 3]) expect(prefetchPlan(age, { index: {} })).toEqual([]);
  });

  it('fetches only approved art of the next age, backs first, thumbnails for cards', () => {
    const index: ArtIndex = {
      'card/lumber-yard': cardEntry('lumber-yard', 8000), // Age I
      'card/sawmill': cardEntry('sawmill', 8000), // Age II
      'back/age2': {
        kind: 'back',
        variants: {
          default: { path: '/art/backs/age2.avif', width: 320, height: 495, bytes: 25000 }
        }
      }
    };
    expect(prefetchPlan(1, { index })).toEqual([
      '/art/backs/age2.avif',
      '/art/cards/sawmill@thumb.avif'
    ]);
    expect(prefetchPlan(0, { index })).toEqual(['/art/cards/lumber-yard@thumb.avif']);
    expect(prefetchPlan(3, { index })).toEqual([]);
  });

  it('stops at the byte budget, counting unknown sizes as a thumbnail budget', () => {
    const age2 = nextAgeArtIds(1).cards.map((id) => id.slice('card/'.length));
    const index: ArtIndex = Object.fromEntries(
      age2.map((slug) => [`card/${slug}`, cardEntry(slug, 0)])
    );
    // 23 cards at an assumed 10 KB each: 20 fit in 200 KB.
    expect(PREFETCH_BUDGET).toBe(200 * 1024);
    expect(prefetchPlan(1, { index })).toHaveLength(20);
    expect(prefetchPlan(1, { index, budget: 25 * 1024 })).toHaveLength(2);
  });
});

describe('prefetchImages', () => {
  it('does nothing outside the browser', () => {
    const stop = prefetchImages(['/art/cards/sawmill@thumb.avif']);
    expect(typeof stop).toBe('function');
    stop();
  });
});
