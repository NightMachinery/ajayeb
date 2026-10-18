import { readFileSync } from 'node:fs';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { cardBySlug, cardSlugs, tokenSlugs, wonderSlugs } from '$lib/catalog';
import { art } from './available.gen';
import {
  cardArtSizes,
  FULL_CARD_WIDTH,
  pickArt,
  tokenArtSizes,
  WONDER_ART_SIZES,
  type ArtIndex
} from './lookup';
import {
  artPrefetchPlan,
  fullPrefetchPlan,
  nextAgeArtIds,
  nextBacks,
  PREFETCH_BUDGET,
  prefetchImages,
  prefetchPlan,
  wantsSmallPlan,
  type PrefetchItem
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

// A small index with one approved asset of each part of the full plan.
function variant(path: string, width: number, height: number, bytes = 1000) {
  return { path, width, height, bytes };
}
const smallIndex: ArtIndex = {
  'card/lumber-yard': cardEntry('lumber-yard', 8000), // Age I
  'card/sawmill': cardEntry('sawmill', 8000), // Age II
  'card/arsenal': cardEntry('arsenal', 8000), // Age III
  'card/merchants-guild': cardEntry('merchants-guild', 8000), // guild
  'wonder/pyramids': {
    kind: 'wonder',
    variants: {
      default: variant('/art/wonders/pyramids.avif', 960, 640),
      thumb: variant('/art/wonders/pyramids@thumb.avif', 480, 320),
      '1440': variant('/art/wonders/pyramids@1440.avif', 1440, 960)
    }
  },
  'token/law': {
    kind: 'token',
    variants: {
      default: variant('/art/tokens/law.avif', 256, 256),
      thumb: variant('/art/tokens/law@thumb.avif', 96, 96)
    }
  },
  'back/age2': { kind: 'back', variants: { default: variant('/art/backs/age2.avif', 320, 495) } },
  'victory/science': {
    kind: 'victory',
    variants: { default: variant('/art/victory/science.avif', 960, 540) }
  }
};

const sheetOf = (slug: string) => ({
  src: `/art/cards/${slug}.avif`,
  srcset: `/art/cards/${slug}@thumb.avif 240w, /art/cards/${slug}.avif 480w`,
  sizes: '140px'
});
const thumbOf = (slug: string) => ({ src: `/art/cards/${slug}@thumb.avif` });
const strip = ({ src, srcset, sizes }: PrefetchItem) => (srcset ? { src, srcset, sizes } : { src });

describe('fullPrefetchPlan', () => {
  it('checks the catalog slugs this test relies on', () => {
    expect(cardBySlug('lumber-yard')?.back).toBe('age1');
    expect(cardBySlug('sawmill')?.back).toBe('age2');
    expect(cardBySlug('arsenal')?.back).toBe('age3');
    expect(cardBySlug('merchants-guild')?.back).toBe('guild');
  });

  it('takes the current age at the card sheet’s file first, then the next age, the rest, wonders, tokens, backs, scenes', () => {
    const plan = fullPrefetchPlan(1, { index: smallIndex });
    expect(plan.map((it) => `${it.part} ${it.id}`)).toEqual([
      'current card/lumber-yard',
      'current card/lumber-yard',
      'next card/sawmill',
      'next card/sawmill',
      'remaining card/arsenal',
      'remaining card/arsenal',
      'guilds card/merchants-guild',
      'guilds card/merchants-guild',
      'wonders wonder/pyramids',
      'wonders wonder/pyramids',
      'tokens token/law',
      'tokens token/law',
      'backs back/age2',
      'victory victory/science'
    ]);
    expect(plan.slice(0, 4).map(strip)).toEqual([
      sheetOf('lumber-yard'),
      thumbOf('lumber-yard'),
      thumbOf('sawmill'),
      sheetOf('sawmill')
    ]);
    expect(plan.slice(8, 12).map(strip)).toEqual([
      { src: '/art/wonders/pyramids@thumb.avif' },
      {
        src: '/art/wonders/pyramids.avif',
        srcset:
          '/art/wonders/pyramids@thumb.avif 480w, /art/wonders/pyramids.avif 960w, /art/wonders/pyramids@1440.avif 1440w',
        sizes: WONDER_ART_SIZES
      },
      { src: '/art/tokens/law@thumb.avif' },
      {
        src: '/art/tokens/law@thumb.avif',
        srcset: '/art/tokens/law@thumb.avif 96w, /art/tokens/law.avif 256w',
        sizes: '36px'
      }
    ]);
  });

  it('puts the guilds with Age III when it is the next or the current age', () => {
    const parts = (age: number) =>
      fullPrefetchPlan(age, { index: smallIndex })
        .filter((it) => it.id.startsWith('card/'))
        .map((it) => `${it.part} ${it.id}`);
    expect(parts(2)).toEqual([
      'current card/sawmill',
      'current card/sawmill',
      'next card/arsenal',
      'next card/merchants-guild',
      'next card/arsenal',
      'next card/merchants-guild',
      'remaining card/lumber-yard',
      'remaining card/lumber-yard'
    ]);
    expect(parts(3).slice(0, 4)).toEqual([
      'current card/arsenal',
      'current card/merchants-guild',
      'current card/arsenal',
      'current card/merchants-guild'
    ]);
    expect(parts(3).slice(4)).toEqual([
      'remaining card/lumber-yard',
      'remaining card/lumber-yard',
      'remaining card/sawmill',
      'remaining card/sawmill'
    ]);
  });

  it('starts the wonder draft with Age I', () => {
    const plan = fullPrefetchPlan(0, { index: smallIndex });
    expect(plan[0]).toMatchObject({ part: 'next', id: 'card/lumber-yard' });
    expect(plan.some((it) => it.part === 'current')).toBe(false);
  });

  it('is empty when no art is approved', () => {
    for (const age of [0, 1, 2, 3]) expect(fullPrefetchPlan(age, { index: {} })).toEqual([]);
  });

  it('lists every approved file of the game once, whatever the age', () => {
    const files = (age: number) => new Set(fullPrefetchPlan(age).map((it) => it.src + it.srcset));
    const all = files(1);
    expect(all.size).toBe(fullPrefetchPlan(1).length);
    for (const age of [0, 2, 3]) expect(files(age)).toEqual(all);
  });

  it('reads complete public lists only: every card of every age, every wonder and token', () => {
    // A plan built from what was dealt or removed would miss some of these.
    for (const age of [0, 1, 2, 3]) {
      const ids = new Set(fullPrefetchPlan(age).map((it) => it.id));
      const everything = [
        ...cardSlugs.map((s) => `card/${s}`),
        ...wonderSlugs.map((s) => `wonder/${s}`),
        ...tokenSlugs.map((s) => `token/${s}`)
      ];
      for (const id of everything) expect(ids.has(id), id).toBe(id in art);
    }
  });

  it('never looks at the game state: the plan and its component depend on the age alone', () => {
    // The plan takes an age and an index; the component a single prop, the
    // age, and neither imports the session, the view or a wire value.
    expect(fullPrefetchPlan.length).toBe(1);
    const sources = ['./prefetch.ts', './ArtPrefetch.svelte'].map((f) =>
      readFileSync(new URL(f, import.meta.url), 'utf8')
    );
    for (const src of sources) {
      const imports = [...src.matchAll(/from '([^']+)'/g)].map((m) => m[1]);
      for (const from of imports) {
        expect(from, from).toMatch(
          /^(\$lib\/catalog|\.\/available\.gen|\.\/lookup|\.\/prefetch|\$lib\/wire)$/
        );
      }
      expect(src).not.toMatch(/import \{[^}]*\b(View|Room|Session)\b/);
    }
    const props = /interface Props \{([^}]*)\}/.exec(sources[1] ?? '')?.[1] ?? '';
    expect([...props.matchAll(/^\s*(\w+)\??:/gm)].map((m) => m[1])).toEqual(['age']);
  });

  it('preloads with the very strings the frames pass', () => {
    const plan = fullPrefetchPlan(1);
    const sheet = plan.find((it) => it.part === 'current' && it.srcset);
    expect(sheet?.sizes).toBe(cardArtSizes(FULL_CARD_WIDTH));
    expect(sheet?.srcset).toBe(pickArt(sheet!.id)?.srcset);
    expect(plan.find((it) => it.part === 'wonders' && it.srcset)?.sizes).toBe(WONDER_ART_SIZES);
    expect(plan.find((it) => it.part === 'tokens' && it.srcset)?.sizes).toBe(tokenArtSizes('md'));
    // The card sheet draws its full-tier card FULL_CARD_WIDTH wide.
    const cardSheet = readFileSync(
      new URL('../components/game/CardSheet.svelte', import.meta.url),
      'utf8'
    );
    const widths = [...cardSheet.matchAll(/tier="full"\s+width=\{(\d+)\}/g)].map((m) =>
      Number(m[1])
    );
    expect(widths.length).toBeGreaterThan(0);
    for (const w of widths) expect(w).toBe(FULL_CARD_WIDTH);
  });
});

describe('the small plan', () => {
  it('is kept for Save-Data and 2G connections, and only then', () => {
    expect(wantsSmallPlan({ saveData: true })).toBe(true);
    expect(wantsSmallPlan({ saveData: false, effectiveType: '2g' })).toBe(true);
    expect(wantsSmallPlan({ effectiveType: 'slow-2g' })).toBe(true);
    expect(wantsSmallPlan({ saveData: false, effectiveType: '3g' })).toBe(false);
    expect(wantsSmallPlan({ effectiveType: '4g' })).toBe(false);
    expect(wantsSmallPlan(undefined)).toBe(false);
    expect(wantsSmallPlan(null)).toBe(false);
  });

  it('is the next age’s backs and thumbnails within the budget', () => {
    for (const age of [0, 1, 2, 3]) {
      const small = artPrefetchPlan(age, { connection: { saveData: true } });
      expect(small).toEqual(prefetchPlan(age).map((src) => ({ src })));
      expect(small.every((it) => it.srcset === undefined)).toBe(true);
      expect(artPrefetchPlan(age, { connection: { effectiveType: '2g' } })).toEqual(small);
    }
    expect(artPrefetchPlan(1, { connection: { effectiveType: '4g' } })).toEqual(
      fullPrefetchPlan(1)
    );
    expect(artPrefetchPlan(1)).toEqual(fullPrefetchPlan(1));
  });
});

// A fake browser: idle callbacks and frames run when the test says, and
// each Image records what it was given.
function fakeBrowser() {
  const idles: (() => void)[] = [];
  const frames: (() => void)[] = [];
  const images: FakeImage[] = [];
  const listeners: Record<string, (() => void)[]> = {};
  const doc = {
    visibilityState: 'visible',
    addEventListener: (t: string, f: () => void) => (listeners[t] ??= []).push(f),
    removeEventListener: (t: string, f: () => void) =>
      (listeners[t] = (listeners[t] ?? []).filter((g) => g !== f))
  };
  class FakeImage {
    decoding = '';
    fetchPriority = '';
    sizes = '';
    srcset = '';
    src = '';
    onload: (() => void) | null = null;
    onerror: (() => void) | null = null;
    constructor() {
      images.push(this);
    }
  }
  const win = {
    requestIdleCallback: (f: () => void) => idles.push(f),
    cancelIdleCallback: (h: number) => {
      idles[h - 1] = () => {};
    },
    requestAnimationFrame: (f: () => void) => frames.push(f),
    cancelAnimationFrame: (h: number) => {
      frames[h - 1] = () => {};
    },
    addEventListener: doc.addEventListener,
    removeEventListener: doc.removeEventListener
  };
  vi.stubGlobal('window', win);
  vi.stubGlobal('document', doc);
  vi.stubGlobal('Image', FakeImage);
  const drain = (q: (() => void)[]) => {
    const fns = q.splice(0, q.length);
    for (const f of fns) f();
  };
  return {
    images,
    paint: () => {
      drain(frames);
      drain(frames);
    },
    idle: () => drain(idles),
    pendingIdle: () => idles.length,
    setHidden: (hidden: boolean) => {
      doc.visibilityState = hidden ? 'hidden' : 'visible';
      for (const f of listeners['visibilitychange'] ?? []) f();
    }
  };
}

describe('prefetchImages in a browser', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('fetches one file at a time, at low priority, after the first paint, with the srcset and sizes', () => {
    const b = fakeBrowser();
    const stop = prefetchImages([
      { src: '/t1/a.avif', srcset: '/t1/a.avif 240w, /t1/b.avif 480w', sizes: '140px' },
      { src: '/t1/c.avif' }
    ]);
    b.idle();
    expect(b.images).toHaveLength(0);
    b.paint();
    b.idle();
    expect(b.images).toHaveLength(1);
    expect(b.images[0]).toMatchObject({
      src: '/t1/a.avif',
      srcset: '/t1/a.avif 240w, /t1/b.avif 480w',
      sizes: '140px',
      fetchPriority: 'low',
      decoding: 'async'
    });
    b.idle();
    expect(b.images).toHaveLength(1);
    b.images[0]!.onload?.();
    b.idle();
    expect(b.images).toHaveLength(2);
    expect(b.images[1]).toMatchObject({ src: '/t1/c.avif', srcset: '' });
    stop();
  });

  it('pauses while the page is hidden, stops when told, and skips what this page already asked for', () => {
    const b = fakeBrowser();
    const stop = prefetchImages(['/t2/a.avif', '/t2/b.avif', '/t2/c.avif']);
    b.paint();
    b.setHidden(true);
    b.idle();
    expect(b.images).toHaveLength(0);
    b.setHidden(false);
    b.idle();
    expect(b.images.map((i) => i.src)).toEqual(['/t2/a.avif']);
    stop();
    b.images[0]!.onload?.();
    b.idle();
    expect(b.images).toHaveLength(1);

    // A new plan (the next age) skips the file already fetched.
    const again = prefetchImages(['/t2/a.avif', '/t2/b.avif']);
    b.paint();
    b.idle();
    expect(b.images.map((i) => i.src)).toEqual(['/t2/a.avif', '/t2/b.avif']);
    again();
  });
});
