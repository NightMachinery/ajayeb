// Display bindings only. Ages, names and rules still belong to the public
// catalog; aliases point at approved paintings without copying their bytes.
import { hgBoardSlugs, hgDesignSlugs, hgDesigns } from '$lib/wire/haftgane-catalog.gen';
import type { PrefetchCatalog } from '$lib/art/prefetch';

export const cardArtId = (slug: string): string => `card/${slug.replace(/-age-\d$/, '')}`;
export const wonderArtId = (board: string): string => `wonder/${board}`;

// The public Age, never a concealed design or its category, selects a back.
// There currently are no individual concealed-card surfaces on the table.
export function concealedBackId(age: number): string | undefined {
  return age >= 1 && age <= 3 && Number.isInteger(age) ? `back/age${age}` : undefined;
}

// Keep these identical to the actual frames. The largest card preview is
// 140 CSS px; the board panel painting is half its width, capped at 360 px.
export const CARD_ART_SIZES = '140px';
export const WONDER_ART_SIZES = '(max-width: 48rem) 50vw, 360px';
export const WONDER_STRIP_SIZES = '48px';

export function prefetchCatalog(age: number): PrefetchCatalog {
  return {
    decksOfStage: (stage) => (stage >= 1 && stage <= 3 ? [`age${stage}`] : []),
    // No earlier Ages: current and later public decks only. Guilds belong
    // to the whole Age III catalog, independently of the drawn guild subset.
    decks: [1, 2, 3].filter((a) => a >= age).map((a) => ({ deck: `age${a}`, part: 'remaining' })),
    cardIds: (decks) =>
      hgDesignSlugs.filter((slug) => decks.includes(`age${hgDesigns[slug].age}`)).map(cardArtId),
    backId: (deck) => (deck === 'age1' ? 'back/age1' : deck === 'age2' ? 'back/age2' : 'back/age3'),
    rest: [
      {
        part: 'wonders',
        ids: hgBoardSlugs.map(wonderArtId),
        frames: [
          { size: 'default', sizes: WONDER_STRIP_SIZES },
          { size: 'default', sizes: WONDER_ART_SIZES }
        ]
      },
      {
        part: 'backs',
        ids: [1, 2, 3].filter((a) => a >= age).map((a) => concealedBackId(a)!),
        frames: [{ size: 'default' }]
      }
    ]
  };
}
