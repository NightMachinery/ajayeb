import { readFileSync, readdirSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { describe, expect, it } from 'vitest';
import { art } from '$lib/art/available-haftgane.gen';
import { art as duel } from '$lib/art/available.gen';
import { fullPrefetchPlanFor } from '$lib/art/prefetch';
import { hgDesignSlugs, hgDesigns, hgBoardSlugs } from '$lib/wire/haftgane-catalog.gen';
import { tableFrame } from '../session/test-frames';
import { cardArtId, concealedBackId, prefetchCatalog, wonderArtId } from './catalog';
import { haftganePrefetchPlan } from './prefetch';

const root = new URL('../../../../../../', import.meta.url);
const json = (path: string) => JSON.parse(readFileSync(new URL(path, root), 'utf8'));
const aliases = json('art/haftgane/aliases.json') as Record<string, string>;
const subjects = json('art/haftgane/subjects.json') as Record<string, unknown>;
const productionPlan = readFileSync(new URL('docs/haftgane/art-reuse.md', root), 'utf8')
  .split('## Card mapping: new paintings')[1]!
  .split('## Wonder boards')[0]!;
const planned = new Set([...productionPlan.matchAll(/\(`([^`]+)`/g)].map((m) => `card/${m[1]}`));
const manifest = json('art/haftgane/manifest.json') as {
  assets: Record<string, { status: string }>;
};

describe('Haftgane approved art bindings', () => {
  it('covers every public design with approved art or its recorded production subject', () => {
    for (const slug of hgDesignSlugs) {
      const id = cardArtId(slug);
      expect(aliases[id] || subjects[id] || planned.has(id), slug).toBeTruthy();
      if (aliases[id] || manifest.assets[id]?.status === 'approved')
        expect(art[id], slug).toBeDefined();
      else
        expect(
          !!subjects[id] || planned.has(id),
          `recorded procedural placeholder for ${slug}`
        ).toBe(true);
    }
  });

  it('reuses Duel entries, ids and content-hashed paths without a copied file', () => {
    for (const [id, target] of Object.entries(aliases)) {
      expect(target.startsWith('duel:')).toBe(true);
      expect(art[id], id).toEqual(duel[target.slice(5)]);
      for (const v of Object.values(art[id]!.variants)) expect(v.path).toMatch(/\?v=[a-f0-9]{10}$/);
    }
    const reused = hgDesignSlugs.filter((slug) => aliases[cardArtId(slug)]);
    expect(reused).toHaveLength(58);
    expect(new Set(reused.map((slug) => aliases[cardArtId(slug)])).size).toBe(55);
    for (const b of hgBoardSlugs) expect(art[wonderArtId(b)]?.variants['1440']).toBeDefined();
    expect(aliases['card/courthouse']).toBe('duel:card/courthouse');
    expect(aliases['card/east-trading-post']).toBe(aliases['card/west-trading-post']);
  });
});

describe('public Age prefetch', () => {
  it('gives two viewers with different private hands the identical plan in every Age', () => {
    for (const age of [1, 2, 3]) {
      const a = tableFrame({}, 0);
      const b = tableFrame({}, 1);
      a.view.age = b.view.age = age;
      a.view.you!.hand = ['altar', 'ore-vein'];
      a.view.you!.tucked = ['workers-guild'];
      b.view.you!.hand = ['spies-guild', 'palace'];
      b.view.you!.kept = 'scientists-guild';
      expect(haftganePrefetchPlan(a.view.age)).toEqual(haftganePrefetchPlan(b.view.age));
      expect(haftganePrefetchPlan(age, { saveData: true })).toEqual(
        haftganePrefetchPlan(b.view.age, { saveData: true })
      );
    }
  });

  it('includes the complete current and later public catalogs, and no earlier decks', () => {
    for (const age of [1, 2, 3]) {
      const ids = fullPrefetchPlanFor(prefetchCatalog(age), age, { index: art }).map(
        (item) => item.id
      );
      for (const slug of hgDesignSlugs.filter((s) => hgDesigns[s].age >= age)) {
        const entry = art[cardArtId(slug)];
        if (entry) {
          const plan = fullPrefetchPlanFor(prefetchCatalog(age), age, { index: art });
          expect(
            plan.map((p) => p.src),
            slug
          ).toContain(entry.variants.default?.path);
        }
      }
      const catalog = prefetchCatalog(age);
      expect(catalog.decks.map((d) => d.deck)).toEqual(
        [1, 2, 3].filter((a) => a >= age).map((a) => `age${a}`)
      );
      expect(catalog.cardIds(['age3'])).toContain('card/scientists-guild');
      expect(ids).not.toContain('back/guild');
    }
  });

  it('never selects a guild back for concealed cards or from table code', () => {
    expect([1, 2, 3].map(concealedBackId)).toEqual(['back/age1', 'back/age2', 'back/age3']);
    for (const age of [-1, 0, 4, 1.5, NaN]) expect(concealedBackId(age)).toBeUndefined();
    expect(art['back/guild']).toBeUndefined();
    const dir = new URL('../table/', import.meta.url);
    for (const file of readdirSync(dir).filter((f) => /\.(svelte|ts)$/.test(f))) {
      const source = readFileSync(new URL(file, dir), 'utf8');
      expect(source, file).not.toMatch(/back\/guild|backArtId|back-[gG]uild|available\.gen/);
    }
    const mounted = readFileSync(new URL('../table/TableScreen.svelte', import.meta.url), 'utf8');
    expect(mounted).toContain('<ArtPrefetch age={view.age} />');
    // No frame or private-state dependency may be added to the plan module.
    const plan = readFileSync(fileURLToPath(new URL('./prefetch.ts', import.meta.url)), 'utf8');
    expect(plan).not.toMatch(/from ['"][^'"]*(session|presenter|model|platform|\/haftgane\.gen)/);
  });
});
