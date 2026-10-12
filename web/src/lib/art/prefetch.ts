// Fetching the game's art ahead of time, while the browser is idle
// (docs/art/bitmaps.md "Prefetch").
//
// The full plan is the whole game's approved art, the part needed soonest
// first: the current age's cards as the card sheet shows them, the next
// age's (thumbnail, then the card sheet's file), the remaining ages, the
// guilds, the wonders, the progress tokens, the card backs and the victory
// scenes. It is built from complete public lists only: the catalog of an
// age, never which cards were dealt, removed or lie face down, which is
// hidden information the view never carries. So the plan is a function of
// the age alone.
//
// Each item is what the frame that shows the art renders (lookup.ts
// artSource): a fixed `src`, or a srcset with the same `sizes` string the
// frame passes. Preloading through an Image given the same srcset and sizes
// lets the browser run its own candidate selection, so the file it fetches
// is exactly the one the frame asks for later, and the cache hits.
//
// With Save-Data on, or a 2G connection, the plan shrinks to the small
// one: the next age's backs and card thumbnails, within about 200 KB.
// Only approved art is ever requested: an id with no entry in the index is
// skipped, so with no art approved the plan is empty.

import { cardBySlug, cardSlugs, tokenSlugs, wonderSlugs } from '$lib/catalog';
import type { Back } from '$lib/wire';
import { art } from './available.gen';
import {
  artBytes,
  artId,
  artSource,
  backArtId,
  cardArtSizes,
  FULL_CARD_WIDTH,
  pickArt,
  tokenArtSizes,
  WONDER_ART_SIZES,
  type ArtIndex,
  type ArtSize,
  type ArtSource
} from './lookup';

/** The budget of the small plan (Save-Data, 2G), in bytes. */
export const PREFETCH_BUDGET = 200 * 1024;

/** What a variant counts against the budget when the index has no byte
 *  count for it: the asset table's card thumbnail budget. */
const UNKNOWN_BYTES = 10 * 1024;

const ALL_BACKS: readonly Back[] = ['age1', 'age2', 'age3', 'guild'];

/** The decks dealt in an age (1 to 3); the guilds come with Age III. */
function backsOfAge(age: number): Back[] {
  switch (age) {
    case 1:
      return ['age1'];
    case 2:
      return ['age2'];
    case 3:
      return ['age3', 'guild'];
    default:
      return [];
  }
}

/** The backs of the age after `age` (0 is the wonder draft, before Age I). */
export function nextBacks(age: number): Back[] {
  return backsOfAge(age + 1);
}

/** Every card of these decks, in catalog order, as art ids. */
function cardIdsOf(backs: readonly Back[]): string[] {
  return cardSlugs
    .filter((slug) => {
      const back = cardBySlug(slug)?.back;
      return back !== undefined && backs.includes(back);
    })
    .map((slug) => artId('card', slug));
}

/** The art ids of the small plan during `age`: the next age's backs, then
 *  its cards (and the guilds before Age III), in catalog order. */
export function nextAgeArtIds(age: number): { backs: string[]; cards: string[] } {
  const backs = nextBacks(age);
  return { backs: backs.map(backArtId), cards: cardIdsOf(backs) };
}

/**
 * The small plan's files, in order, within `budget` bytes: each back's
 * default size, then each card's `cardSize` (the board shows the
 * thumbnail). Ids with no approved art are skipped; the first file that
 * would overrun the budget ends the plan.
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

/** The parts of the full plan, in the order they are fetched. */
export type PrefetchPart =
  'current' | 'next' | 'remaining' | 'guilds' | 'wonders' | 'tokens' | 'backs' | 'victory';

/** One file of the full plan: what its frame renders, and what it is. */
export interface PrefetchItem extends ArtSource {
  readonly id: string;
  readonly part: PrefetchPart;
}

/** The victory scenes the result screen can show (lookup.ts victoryArtId). */
const VICTORY_IDS = ['victory/military', 'victory/science', 'victory/civilian', 'victory/draw'];

/**
 * The whole game's art in fetch order during `age` (0 is the wonder
 * draft): the current age's cards at the card sheet's file, then their
 * thumbnails; the next age's cards, thumbnail then card sheet file; the
 * other ages the same way; the guilds (with Age III when that is the
 * current or the next age); every wonder (thumbnail, then the large tile);
 * every token (thumbnail, then the medallion's srcset); the backs; the
 * victory scenes. A file is listed once, and only approved art is.
 */
export function fullPrefetchPlan(
  age: number,
  { index = art }: { index?: ArtIndex } = {}
): PrefetchItem[] {
  const items: PrefetchItem[] = [];
  const seen = new Set<string>();
  const add = (id: string, part: PrefetchPart, source: ArtSource | undefined) => {
    if (!source) return;
    const key = sourceKey(source);
    if (seen.has(key)) return;
    seen.add(key);
    items.push({ ...source, id, part });
  };
  // The two frames of a card: the board's thumbnail, and the card sheet's
  // srcset (CardArt's full tier at the sheet's width).
  const thumb = (id: string) => artSource(id, 'thumb', undefined, index);
  const sheet = (id: string) => artSource(id, 'default', cardArtSizes(FULL_CARD_WIDTH), index);
  const cards = (part: PrefetchPart, backs: readonly Back[], sheetFirst: boolean) => {
    const ids = cardIdsOf(backs);
    for (const frame of sheetFirst ? [sheet, thumb] : [thumb, sheet]) {
      for (const id of ids) add(id, part, frame(id));
    }
  };

  const current = backsOfAge(age);
  const next = backsOfAge(age + 1);
  cards('current', current, true);
  cards('next', next, false);
  const done = new Set<Back>([...current, ...next]);
  for (const back of ['age1', 'age2', 'age3'] as const) {
    if (!done.has(back)) cards('remaining', [back], false);
  }
  if (!done.has('guild')) cards('guilds', ['guild'], false);
  for (const slug of wonderSlugs) {
    const id = artId('wonder', slug);
    add(id, 'wonders', artSource(id, 'thumb', undefined, index));
    add(id, 'wonders', artSource(id, 'default', WONDER_ART_SIZES, index));
  }
  for (const slug of tokenSlugs) {
    const id = artId('token', slug);
    add(id, 'tokens', artSource(id, 'thumb', undefined, index));
    add(id, 'tokens', artSource(id, 'thumb', tokenArtSizes('md'), index));
  }
  for (const back of ALL_BACKS) {
    add(backArtId(back), 'backs', artSource(backArtId(back), 'default', undefined, index));
  }
  for (const id of VICTORY_IDS) add(id, 'victory', artSource(id, 'default', undefined, index));
  return items;
}

/** What `navigator.connection` reports (the Network Information API, in
 *  the browsers that have it). */
export interface ConnectionInfo {
  readonly saveData?: boolean;
  readonly effectiveType?: string;
}

/** Whether to keep to the small plan: the user asked to save data, or the
 *  connection is 2G or slower. */
export function wantsSmallPlan(connection: ConnectionInfo | null | undefined): boolean {
  if (!connection) return false;
  return (
    connection.saveData === true ||
    connection.effectiveType === 'slow-2g' ||
    connection.effectiveType === '2g'
  );
}

/** The plan to fetch during `age`: the full plan, or the small one when the
 *  connection asks to save data. */
export function artPrefetchPlan(
  age: number,
  { connection, index = art }: { connection?: ConnectionInfo | null; index?: ArtIndex } = {}
): ArtSource[] {
  if (wantsSmallPlan(connection)) return prefetchPlan(age, { index }).map((src) => ({ src }));
  return fullPrefetchPlan(age, { index });
}

/** This browser's connection, when it says. */
export function browserConnection(): ConnectionInfo | undefined {
  if (typeof navigator === 'undefined') return undefined;
  return (navigator as Navigator & { connection?: ConnectionInfo }).connection;
}

/** One key per distinct request: a srcset with other sizes may pick
 *  another file, so the sizes are part of it. */
function sourceKey(s: ArtSource): string {
  return s.srcset ? `${s.srcset}|${s.sizes ?? ''}` : s.src;
}

// What this page has already asked for, so a remount, a new age or an undo
// across an age boundary does not fetch it again.
const requested = new Set<string>();

/**
 * Fetches the files at low priority while the browser is idle, one at a
 * time so they never compete with the visible art, starting once the next
 * two frames have painted. Each goes through an Image given the item's
 * srcset and sizes, so the browser picks the file the frame will. It
 * pauses while the page is hidden (and on pagehide) and goes on when it is
 * shown again. Returns a function that stops whatever has not started
 * yet. A no-op outside the browser.
 */
export function prefetchImages(items: readonly (ArtSource | string)[]): () => void {
  if (typeof window === 'undefined' || typeof Image === 'undefined') return () => {};
  const queue = items
    .map((it) => (typeof it === 'string' ? { src: it } : it))
    .filter((it) => !requested.has(sourceKey(it)));
  if (queue.length === 0) return () => {};
  let stopped = false;
  let busy = false;
  let handle: number | undefined;
  let frame: number | undefined;
  const idle = (fn: () => void) =>
    typeof window.requestIdleCallback === 'function'
      ? window.requestIdleCallback(fn, { timeout: 4000 })
      : window.setTimeout(fn, 1500);
  const cancelIdle = (h: number) =>
    typeof window.cancelIdleCallback === 'function'
      ? window.cancelIdleCallback(h)
      : window.clearTimeout(h);
  const hidden = () => document.visibilityState === 'hidden';

  const schedule = () => {
    if (stopped || busy || frame !== undefined || handle !== undefined || hidden()) return;
    handle = idle(next);
  };
  const next = () => {
    handle = undefined;
    if (stopped || hidden()) return;
    let item = queue.shift();
    while (item !== undefined && requested.has(sourceKey(item))) item = queue.shift();
    if (item === undefined) return;
    requested.add(sourceKey(item));
    busy = true;
    const img = new Image();
    img.decoding = 'async';
    img.fetchPriority = 'low';
    // Loaded or failed, go on with the next one when the browser is idle.
    img.onload = img.onerror = () => {
      busy = false;
      schedule();
    };
    // sizes before srcset, and src last, as markup gives them.
    if (item.srcset) {
      img.sizes = item.sizes ?? '';
      img.srcset = item.srcset;
    }
    img.src = item.src;
  };
  const pause = () => {
    if (handle !== undefined) cancelIdle(handle);
    handle = undefined;
  };
  const onVisibility = () => (hidden() ? pause() : schedule());
  document.addEventListener('visibilitychange', onVisibility);
  window.addEventListener('pagehide', pause);
  window.addEventListener('pageshow', schedule);
  // Start after the board's first paint: two frames, then the idle wait.
  frame = window.requestAnimationFrame(() => {
    frame = window.requestAnimationFrame(() => {
      frame = undefined;
      schedule();
    });
  });

  return () => {
    stopped = true;
    if (frame !== undefined) window.cancelAnimationFrame(frame);
    frame = undefined;
    pause();
    document.removeEventListener('visibilitychange', onVisibility);
    window.removeEventListener('pagehide', pause);
    window.removeEventListener('pageshow', schedule);
  };
}
