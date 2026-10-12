// The recap family's client half (the hints design, 2.11, M10.HS5): what
// changed since a player's own last move, from the snapshots this screen
// saw. Today that is price-changed-tick: the trade prices the other side's
// moves changed ("Clay: was 2, now 3. Moon built Clay Pit."), which the
// opponent move summary lists. The board's trace of the emptied slot and
// the spark on a freshly revealed card (last-move-trace) read revealedSince.
//
// Display only: the prices are the server's (PlayerView.tradePrices),
// compared with the ones the same seat saw when its own last move ended;
// the cause is read from public build events and the presentation
// catalog's printed production, never worked out from the rules.
//
// A baseline is kept per seat, so a hot-seat device compares each player
// with what that player saw. Every baseline restarts, with no tick, when
// an age begins (the new structure makes old prices meaningless), after an
// undo or a resync (a frame with the history: nothing happened just now)
// and in another room.

import { cardBySlug } from '$lib/catalog';
import type { GameEvent, HistoryEntry, Resource, ResourceCounts, State } from '$lib/wire';
import type { Seat } from './move-summary';

const RESOURCES = [
  'wood',
  'clay',
  'stone',
  'glass',
  'papyrus'
] as const satisfies readonly Resource[];

/** One seat's baseline: its trade prices when its own last move ended, and
 *  the events of the other side's moves since. */
export interface PriceBase {
  readonly prices: ResourceCounts;
  readonly events: readonly GameEvent[];
}

/** What the tracker keeps between frames. */
export interface RecapState {
  readonly code: string | null;
  readonly version: number;
  readonly age: number;
  /** By seat. */
  readonly base: readonly [PriceBase | null, PriceBase | null];
}

export const EMPTY_RECAP: RecapState = { code: null, version: -1, age: -1, base: [null, null] };

/** The parts of a state frame the tracker reads. */
export interface RecapFrame {
  code: State['code'];
  version: State['version'];
  events: State['events'];
  history?: State['history'];
  view: Pick<State['view'], 'age' | 'players'>;
}

/** One trade price that changed for the viewer since their last move. */
export interface PriceTick {
  readonly resource: Resource;
  readonly from: number;
  readonly to: number;
  /** The other side's build that raised it, when one card explains it. */
  readonly cause?: { readonly seat: Seat; readonly card: string };
}

function fresh(view: RecapFrame['view']): [PriceBase, PriceBase] {
  return [
    { prices: { ...view.players[0].tradePrices }, events: [] },
    { prices: { ...view.players[1].tradePrices }, events: [] }
  ];
}

/** The seats that took an action in these events. */
function movers(events: readonly GameEvent[]): Set<Seat> {
  const out = new Set<Seat>();
  for (const e of events) {
    switch (e.kind) {
      case 'draft':
      case 'build':
      case 'discard':
      case 'wonder':
      case 'starterChosen':
      case 'token':
      case 'destroy':
      case 'rebuild':
        if (e.seat === 0 || e.seat === 1) out.add(e.seat);
        break;
      default:
        break;
    }
  }
  return out;
}

/**
 * The tracker after one frame. A live frame (same room, newer version, no
 * history, same age) adds its events to each seat's baseline, then a seat
 * that moved in it starts again from the prices it now sees: its own moves
 * never tick, and its ticks end with its move. Any other frame restarts
 * every baseline.
 */
export function nextRecap(prev: RecapState, f: RecapFrame): RecapState {
  const live =
    prev.code === f.code &&
    f.version > prev.version &&
    !f.history &&
    prev.age === f.view.age &&
    prev.base[0] !== null &&
    prev.base[1] !== null;
  if (!live) {
    return { code: f.code, version: f.version, age: f.view.age, base: fresh(f.view) };
  }
  const moved = movers(f.events);
  const now = fresh(f.view);
  const base = ([0, 1] as const).map((seat) => {
    const b = prev.base[seat]!;
    return moved.has(seat) ? now[seat] : { prices: b.prices, events: [...b.events, ...f.events] };
  }) as [PriceBase, PriceBase];
  return { code: f.code, version: f.version, age: f.view.age, base };
}

/** Whether a card prints production of resource (brown and grey cards). */
function produces(card: string, resource: Resource): boolean {
  return (cardBySlug(card)?.effects ?? []).some(
    (e) => e.kind === 'produce' && e.resource === resource
  );
}

/**
 * The trade prices of seat that changed since its last move, in resource
 * order, each with the other side's build that explains it when exactly
 * one card of the events produces that resource.
 */
export function priceTicks(
  state: RecapState,
  view: Pick<State['view'], 'players'>,
  seat: Seat
): PriceTick[] {
  const base = state.base[seat];
  if (!base) return [];
  const now = view.players[seat].tradePrices;
  const other: Seat = seat === 0 ? 1 : 0;
  const ticks: PriceTick[] = [];
  for (const resource of RESOURCES) {
    const from = base.prices[resource];
    const to = now[resource];
    if (from === to) continue;
    const builds = base.events.filter(
      (e) =>
        (e.kind === 'build' || e.kind === 'rebuild') &&
        e.seat === other &&
        !!e.card &&
        produces(e.card, resource)
    );
    const card = builds.length === 1 ? builds[0]!.card : undefined;
    ticks.push(
      card ? { resource, from, to, cause: { seat: other, card } } : { resource, from, to }
    );
  }
  return ticks;
}

/**
 * Whether a summary of the other side's moves is the current one: nothing
 * of the viewer's came after it, so the ticks (changes since the viewer's
 * last move) belong to it.
 */
export function isCurrentRun(
  summary: { readonly seat: Seat; readonly plies: readonly number[] } | null,
  history: readonly HistoryEntry[]
): boolean {
  const last = history[history.length - 1];
  return !!summary && !!last && last.seat === summary.seat && summary.plies.includes(last.ply);
}

/**
 * The cards turned face up since the viewer's last move, from the reveal
 * events of the baseline: the board's spark marks them (last-move-trace).
 */
export function revealedSince(state: RecapState, seat: Seat): string[] {
  const base = state.base[seat];
  if (!base) return [];
  const out: string[] = [];
  for (const e of base.events) {
    if (e.kind === 'reveal' && e.card && !out.includes(e.card)) out.push(e.card);
  }
  return out;
}
