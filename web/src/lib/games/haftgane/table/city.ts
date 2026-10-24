// What the city sheet and the Seen hands sheet derive from a frame, without
// rules: the seats in ring order for the tabs, a city's cards by colour, and
// the notebook grouped by Age with the point ids decoded for headings. Every
// count, bound and score is the server's; this only sorts and groups what
// the view holds.

import type { Notebook, PileLook, Seat, SeenEntry } from '$lib/wire/haftgane.gen';
import type { HgColour } from '$lib/wire/haftgane-catalog.gen';
import { COLOURS, design, ring, TURNS_PER_AGE } from './model';

/** Every seat from the anchor round the table: the anchor, its left
 *  neighbour, the far seats and its right neighbour, as the tabs list them. */
export function ringOrder(seats: readonly Seat[]): Seat[] {
  const r = ring(seats);
  return r ? [r.anchor, r.left, ...r.far, r.right] : [...seats];
}

/** One of the city sheet's ring tabs. */
export interface RingTab {
  seat: number;
  /** The tab's label: "You", or the seat's name. */
  name: string;
  /** Name and position, already translated. */
  tip: string;
  bot: boolean;
}

/** One row of a city: a colour and its cards in the order the view lists them. */
export interface CityRow {
  colour: HgColour;
  cards: string[];
}

/** A city's cards by colour, every colour present, in COLOURS order. */
export function cityRows(built: readonly string[]): CityRow[] {
  return COLOURS.map((colour) => ({
    colour,
    cards: built.filter((slug) => design(slug)?.colour === colour)
  }));
}

// The notebook -----------------------------------------------------------------

/** A point id decoded: `age-2-turn-3`, `age-2-turn-6-discard-build` or
 *  `age-2-last-card` (docs/protocol.md). The last card counts as turn 7. */
export interface PointParts {
  age: number;
  turn: number;
  kind: 'turn' | 'discard-build' | 'last-card';
}

export function pointParts(id: string): PointParts | null {
  const m = /^age-([1-3])-(?:turn-([1-6])(-discard-build)?|(last-card))$/.exec(id);
  if (!m) return null;
  const age = Number(m[1]);
  if (m[4]) return { age, turn: TURNS_PER_AGE + 1, kind: 'last-card' };
  return { age, turn: Number(m[2]), kind: m[3] ? 'discard-build' : 'turn' };
}

/** One entry of an Age's page: a hand held, or a look at the pile. */
export type NotebookItem =
  | { kind: 'hand'; parts: PointParts; seen: SeenEntry }
  | { kind: 'pile'; parts: PointParts; look: PileLook };

export interface NotebookAge {
  age: number;
  items: NotebookItem[];
}

/**
 * The notebook by Age, the current Age first and then back to Age I, each
 * Age's entries in the order they happened (a pile look after the hand of
 * its turn). The current Age is listed even before it holds anything.
 */
export function notebookAges(notebook: Notebook, current: number): NotebookAge[] {
  const items: NotebookItem[] = [];
  for (const seen of notebook.seen) {
    const parts = pointParts(seen.point);
    if (parts) items.push({ kind: 'hand', parts, seen });
  }
  for (const look of notebook.pileLooks) {
    const parts = pointParts(look.point);
    if (parts) items.push({ kind: 'pile', parts, look });
  }
  const ages = new Set(items.map((i) => i.parts.age));
  if (current >= 1 && current <= 3) ages.add(current);
  const order = (i: NotebookItem) => i.parts.turn * 2 + (i.kind === 'pile' ? 1 : 0);
  return [...ages]
    .sort((a, b) => (a === current ? -1 : b === current ? 1 : b - a))
    .map((age) => ({
      age,
      items: items.filter((i) => i.parts.age === age).sort((a, b) => order(a) - order(b))
    }));
}

/** What a statistic filters the notebook's cards to. */
export type SeenFilter = { colour: HgColour } | { design: string };

export function matchesFilter(slug: string, filter: SeenFilter | null): boolean {
  if (!filter) return true;
  return 'colour' in filter ? design(slug)?.colour === filter.colour : slug === filter.design;
}

/** An entry shows under a filter when one of its cards matches it. */
export function itemMatches(item: NotebookItem, filter: SeenFilter | null): boolean {
  if (!filter) return true;
  const cards = item.kind === 'hand' ? item.seen.hand : item.look.cards.map((c) => c.design);
  return cards.some((slug) => matchesFilter(slug, filter));
}
