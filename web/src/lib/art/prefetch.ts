// Fetching the next age's art ahead of time (UX design §4, "Asset table":
// "the next age's thumbnails are preloaded during the current age, about
// 200 KB or less per age").
//
// The client cannot know which cards the next age will deal (that is hidden
// information, and the view never says), so the plan is every approved card
// of the next age's backs, in catalog order, after the backs themselves,
// cut off at the byte budget. Only approved art is ever requested: an id
// with no entry in the index is skipped, so with no art approved the plan is
// empty and nothing is fetched.

import { cardSlugs, cardBySlug } from '$lib/catalog';
import type { Back } from '$lib/wire';
import { art } from './available.gen';
import { artBytes, artId, backArtId, pickArt, type ArtIndex, type ArtSize } from './lookup';

/** The budget of one age's prefetch, in bytes. */
export const PREFETCH_BUDGET = 200 * 1024;

/** What a variant counts against the budget when the index has no byte
 *  count for it: the asset table's card thumbnail budget. */
const UNKNOWN_BYTES = 10 * 1024;

/** The backs of the age after `age` (0 is the wonder draft, before Age I);
 *  the guilds come with Age III. */
export function nextBacks(age: number): Back[] {
  switch (age) {
    case 0:
      return ['age1'];
    case 1:
      return ['age2'];
    case 2:
      return ['age3', 'guild'];
    default:
      return [];
  }
}

/** The art ids worth fetching during `age`: the next age's backs, then its
 *  cards (and the guilds before Age III), in catalog order. */
export function nextAgeArtIds(age: number): { backs: string[]; cards: string[] } {
  const backs = nextBacks(age);
  return {
    backs: backs.map(backArtId),
    cards: cardSlugs
      .filter((slug) => {
        const back = cardBySlug(slug)?.back;
        return back !== undefined && backs.includes(back);
      })
      .map((slug) => artId('card', slug))
  };
}

/**
 * The files to fetch during `age`, in order, within `budget` bytes: each
 * back's default size, then each card's `cardSize` (the board's medium tier
 * shows the thumbnail). Ids with no approved art are skipped; the first file
 * that would overrun the budget ends the plan.
 */
export function prefetchPlan(
  age: number,
  {
    cardSize = 'thumb',
    budget = PREFETCH_BUDGET,
    index = art
  }: { cardSize?: ArtSize; budget?: number; index?: ArtIndex } = {}
): string[] {
  const { backs, cards } = nextAgeArtIds(age);
  const wanted: { id: string; size: ArtSize }[] = [
    ...backs.map((id) => ({ id, size: 'default' })),
    ...cards.map((id) => ({ id, size: cardSize }))
  ];
  const plan: string[] = [];
  let spent = 0;
  for (const { id, size } of wanted) {
    const pick = pickArt(id, size, index);
    if (!pick || plan.includes(pick.src)) continue;
    const bytes = artBytes(id, size, index) ?? UNKNOWN_BYTES;
    if (spent + bytes > budget) break;
    spent += bytes;
    plan.push(pick.src);
  }
  return plan;
}

// Files this page has already asked for, so a remount or an undo across an
// age boundary does not fetch them again.
const requested = new Set<string>();

/**
 * Fetches the files at low priority when the browser is idle, one at a time
 * so they never compete with the visible art. Returns a function that stops
 * whatever has not started yet. A no-op outside the browser.
 */
export function prefetchImages(paths: readonly string[]): () => void {
  if (typeof window === 'undefined' || typeof Image === 'undefined') return () => {};
  const queue = paths.filter((p) => !requested.has(p));
  let stopped = false;
  let handle: number | undefined;
  const idle = (fn: () => void) =>
    typeof window.requestIdleCallback === 'function'
      ? window.requestIdleCallback(fn, { timeout: 4000 })
      : window.setTimeout(fn, 1500);
  const cancelIdle = (h: number) =>
    typeof window.cancelIdleCallback === 'function'
      ? window.cancelIdleCallback(h)
      : window.clearTimeout(h);

  const next = () => {
    handle = undefined;
    const path = queue.shift();
    if (stopped || path === undefined) return;
    requested.add(path);
    const img = new Image();
    img.decoding = 'async';
    img.fetchPriority = 'low';
    // Failed or not, move on to the next one when the browser is idle again.
    img.onload = img.onerror = () => {
      if (!stopped) handle = idle(next);
    };
    img.src = path;
  };
  handle = idle(next);

  return () => {
    stopped = true;
    if (handle !== undefined) cancelIdle(handle);
  };
}
