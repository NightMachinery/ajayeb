// Looking up approved bitmap art (AGENTS.md "Assets", docs/art/bitmaps.md).
//
// The generated index (available.gen.ts, written by tools/art/index.mjs)
// lists only approved art, each id with its shipped AVIF variants. These
// helpers answer "which file, if any, should this frame show?" and nothing
// else: an id with no entry, or an entry with no usable variant, gives
// undefined, and the caller draws its procedural placeholder. So the app
// never asks for a file that was not shipped, and as soon as an asset is
// approved and the index regenerated, it shows up with no code change.
//
// Every function takes the index as an optional last argument (the
// generated one by default), so tests and the dev gallery can pass their
// own.

import type { OutcomeKind } from '$lib/view/score';
import type { Back, BotLevel, SeatInfo } from '$lib/wire';
import { art, type ArtEntry, type ArtVariant } from './available.gen';

export type ArtIndex = { readonly [id: string]: ArtEntry };

/** The asset kinds of docs/art/bitmaps.md's asset table. */
export type ArtKind =
  'card' | 'wonder' | 'token' | 'back' | 'texture' | 'hero' | 'avatar' | 'victory' | 'icon';

/** A shipped size: 'default' for a kind's main size, otherwise its suffix. */
export type ArtSize = 'default' | 'thumb' | (string & {});

/** One frame's choice of file: the preferred variant, plus every variant as
 *  a srcset (width descriptors, narrowest first) for frames that pass
 *  `sizes`. Paths are root-absolute (the app is served at the site root). */
export interface ArtPick {
  readonly id: string;
  readonly src: string;
  readonly width: number;
  readonly height: number;
  readonly srcset: string;
}

/** The art id of an asset: "<kind>/<slug>", as the index keys it. */
export function artId(kind: ArtKind, slug: string): string {
  return `${kind}/${slug}`;
}

/** The art id of a card back. */
export function backArtId(back: Back): string {
  return artId('back', back);
}

/** The avatar slug of each bot level (the persona names of game/names.ts). */
const AVATAR_OF: Record<BotLevel, string> = {
  easy: 'apprentice',
  normal: 'scholar',
  hard: 'master'
};

/** The art id of a bot level's avatar. */
export function avatarArtId(level: BotLevel): string {
  return artId('avatar', AVATAR_OF[level]);
}

/** The bot level of a seat, when a bot plays it: the avatar that seat's
 *  name shows beside it. Undefined for a human or open seat, or no room. */
export function seatBotLevel(
  room: { readonly seats: readonly Pick<SeatInfo, 'seat' | 'kind' | 'bot'>[] } | null | undefined,
  seat: number
): BotLevel | undefined {
  const info = room?.seats.find((s) => s.seat === seat);
  return info?.kind === 'bot' ? info.bot : undefined;
}

/** How a game ended, as the result screen names it ('shared' is a draw on
 *  points and on blue cards). */
export type VictoryKind = OutcomeKind;

/** The victory scenes to try for each way a game ends, in order. A draw
 *  shows its own scene once one is approved, and the civilian one until
 *  then (both players reached the end of Age III). A forfeit has no scene
 *  of its own and shows the civilian one, the most neutral of the three. */
const VICTORY_SCENES: Record<VictoryKind, readonly [string, ...string[]]> = {
  civilian: ['victory/civilian'],
  military: ['victory/military'],
  science: ['victory/science'],
  shared: ['victory/draw', 'victory/civilian'],
  forfeit: ['victory/civilian']
};

/** The art id of the scene for how a game ended: the first candidate with
 *  approved art, else the first candidate (whose placeholder then shows). */
export function victoryArtId(kind: VictoryKind, index: ArtIndex = art): string {
  const candidates = VICTORY_SCENES[kind];
  return candidates.find((id) => hasApprovedArt(id, index)) ?? candidates[0];
}

function usable(v: ArtVariant | undefined): v is ArtVariant {
  return v !== undefined && typeof v.path === 'string' && v.path !== '' && v.width > 0;
}

/** The entry for an id, or undefined. Own keys only, so an id such as
 *  "constructor" is never mistaken for art. */
export function artEntry(id: string, index: ArtIndex = art): ArtEntry | undefined {
  return Object.hasOwn(index, id) ? index[id] : undefined;
}

/** Whether approved art exists for an id. */
export function hasApprovedArt(id: string, index: ArtIndex = art): boolean {
  return pickArt(id, 'default', index) !== undefined;
}

/**
 * The file a frame should show for an id, preferring the `size` variant,
 * then the kind's default size, then the widest variant there is. Undefined
 * when the id has no approved art (the caller draws its placeholder).
 */
export function pickArt(
  id: string,
  size: ArtSize = 'default',
  index: ArtIndex = art
): ArtPick | undefined {
  const entry = artEntry(id, index);
  if (!entry) return undefined;
  const variants = Object.keys(entry.variants)
    .map((k) => entry.variants[k])
    .filter(usable)
    .sort((a, b) => a.width - b.width);
  const own = (k: string) => (Object.hasOwn(entry.variants, k) ? entry.variants[k] : undefined);
  const widest = variants[variants.length - 1];
  const preferred = [own(size), own('default'), widest].find(usable);
  if (!preferred) return undefined;
  return {
    id,
    src: preferred.path,
    width: preferred.width,
    height: preferred.height,
    srcset: variants.map((v) => `${v.path} ${v.width}w`).join(', ')
  };
}

/** The shipped byte count of one variant, or undefined when the index has
 *  none (an entry written before its bytes were recorded). */
export function artBytes(
  id: string,
  size: ArtSize = 'default',
  index: ArtIndex = art
): number | undefined {
  const entry = artEntry(id, index);
  const v = entry && Object.hasOwn(entry.variants, size) ? entry.variants[size] : undefined;
  return usable(v) && v.bytes > 0 ? v.bytes : undefined;
}
