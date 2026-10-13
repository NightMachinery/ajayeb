// The recap family's client half (the hints design, 2.11, M10.HS5): what
// changed since a player's own last move, from the snapshots this screen
// saw. price-changed-tick: the trade prices the other side's moves changed
// ("Clay: was 2, now 3. Moon built Clay Pit."), which the opponent move
// summary lists, and the board (M10.F3) marks with a caret on the strip's
// trade-price chip and a "Was 4, now 5" line in a card's badge tooltip
// (cardPriceTick). last-move-trace: the slots the other side's moves
// emptied (emptiedSince) and the cards they turned face up
// (revealedSince), which the structure marks until the seat's own move
// ends. recapBoard gathers what the board draws, and the game screen
// hands it down through a context (setRecapBoard, getRecapBoard).
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

import { getContext, setContext } from 'svelte';
import { cardBySlug } from '$lib/catalog';
import { cardName } from '$lib/components/card/effect-text';
import { t } from '$lib/i18n';
import type { GameEvent, HistoryEntry, Resource, ResourceCounts, SlotView, State } from '$lib/wire';
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
  /** Its cost (CostView.total) of each face-up card then, by card. */
  readonly costs?: Readonly<Record<string, number>>;
  /** The structure slots the moves since emptied, in the order they went. */
  readonly emptied?: readonly number[];
}

/** What the tracker keeps between frames. */
export interface RecapState {
  readonly code: string | null;
  readonly version: number;
  readonly age: number;
  /** By seat. */
  readonly base: readonly [PriceBase | null, PriceBase | null];
  /** The slots that held a card in the last frame (for the emptied slots). */
  readonly filled?: readonly number[];
}

export const EMPTY_RECAP: RecapState = { code: null, version: -1, age: -1, base: [null, null] };

/** The parts of a state frame the tracker reads. */
export interface RecapFrame {
  code: State['code'];
  version: State['version'];
  events: State['events'];
  history?: State['history'];
  view: Pick<State['view'], 'age' | 'players'> & { structure?: State['view']['structure'] };
}

/** One trade price that changed for the viewer since their last move. */
export interface PriceTick {
  readonly resource: Resource;
  readonly from: number;
  readonly to: number;
  /** The other side's build that raised it, when one card explains it. */
  readonly cause?: { readonly seat: Seat; readonly card: string };
}

/** A seat's cost of each face-up card in the structure, by card. */
function costsOf(structure: readonly SlotView[] | undefined, seat: Seat): Record<string, number> {
  const out: Record<string, number> = {};
  for (const s of structure ?? []) {
    const cost = s.costs?.[seat];
    if (s.state === 'faceUp' && s.card && cost) out[s.card] = cost.total;
  }
  return out;
}

/** The slots that hold a card. */
function filledOf(structure: readonly SlotView[] | undefined): number[] {
  return (structure ?? []).filter((s) => s.state !== 'empty').map((s) => s.index);
}

function fresh(view: RecapFrame['view']): [PriceBase, PriceBase] {
  return [
    {
      prices: { ...view.players[0].tradePrices },
      events: [],
      costs: costsOf(view.structure, 0),
      emptied: []
    },
    {
      prices: { ...view.players[1].tradePrices },
      events: [],
      costs: costsOf(view.structure, 1),
      emptied: []
    }
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
  const filled = filledOf(f.view.structure);
  if (!live) {
    return { code: f.code, version: f.version, age: f.view.age, base: fresh(f.view), filled };
  }
  const moved = movers(f.events);
  const now = fresh(f.view);
  const nowFilled = new Set(filled);
  const gone = (prev.filled ?? []).filter((i) => !nowFilled.has(i));
  const base = ([0, 1] as const).map((seat): PriceBase => {
    const b = prev.base[seat]!;
    if (moved.has(seat)) return now[seat];
    return {
      prices: b.prices,
      events: [...b.events, ...f.events],
      costs: b.costs ?? {},
      emptied: [...(b.emptied ?? []), ...gone]
    };
  });
  return {
    code: f.code,
    version: f.version,
    age: f.view.age,
    base: base as [PriceBase, PriceBase],
    filled
  };
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

/**
 * The structure slots the other side's moves emptied since the seat's last
 * move: the board draws a faint trace there (last-move-trace).
 */
export function emptiedSince(state: RecapState, seat: Seat): number[] {
  return [...(state.base[seat]?.emptied ?? [])];
}

/** A card whose price for the seat changed since its last move. */
export interface CardPriceTick {
  readonly from: number;
  readonly to: number;
  /** The other side's build that explains it, when one card does. */
  readonly cause?: { readonly seat: Seat; readonly card: string };
}

/**
 * How the seat's cost of `card` changed since its last move: the server's
 * total then and now (CostView.total; never computed here), with the one
 * build that explains it when every trade price that moved names the same
 * card. Undefined when the price is the same, or the card was not face up
 * when the seat's last move ended.
 */
export function cardPriceTick(
  state: RecapState,
  view: Pick<State['view'], 'players' | 'structure'>,
  seat: Seat,
  card: string
): CardPriceTick | undefined {
  const from = state.base[seat]?.costs?.[card];
  const now = view.structure.find((s) => s.state === 'faceUp' && s.card === card)?.costs?.[seat];
  if (from === undefined || !now || now.total === from) return undefined;
  const ticks = priceTicks(state, view, seat);
  const causes = new Set(ticks.map((tk) => tk.cause?.card));
  const first = ticks[0]?.cause;
  return causes.size === 1 && first
    ? { from, to: now.total, cause: first }
    : { from, to: now.total };
}

/** The badge tooltip line of a card price change: "Was 4, now 5: Moon built Clay Pit." */
export function cardPriceText(tick: CardPriceTick, names: readonly [string, string]): string {
  return tick.cause
    ? t('hints.recap.cardPriceBuilt', {
        from: tick.from,
        to: tick.to,
        name: names[tick.cause.seat],
        card: cardName(tick.cause.card)
      })
    : t('hints.recap.cardPrice', { from: tick.from, to: tick.to });
}

/** The trade-price chip's tooltip line for one tick ("Was 2, now 3."). */
export function priceTickText(tick: PriceTick, names: readonly [string, string]): string {
  return tick.cause
    ? t('hints.recap.chipBuilt', {
        from: tick.from,
        to: tick.to,
        name: names[tick.cause.seat],
        card: cardName(tick.cause.card)
      })
    : t('hints.recap.chip', { from: tick.from, to: tick.to });
}

// --- What the board draws (M10.F3) -------------------------------------------

/** A trade-price chip's caret: which way the price went, and its tooltip line. */
export interface RecapChip {
  readonly up: boolean;
  readonly text: string;
}

/** The recap's board marks for the viewer's seat, until its move ends. */
export interface RecapBoard {
  /** The seat they are for (the viewer, or the seat to move on a hot-seat device). */
  readonly seat: Seat;
  /** That seat's trade prices now: the resource chips that show them carry the carets. */
  readonly prices: ResourceCounts;
  /** Its changed trade prices, by resource. */
  readonly chips: ReadonlyMap<Resource, RecapChip>;
  /** The slots the other side emptied, and the cards it turned face up. */
  readonly emptied: ReadonlySet<number>;
  readonly revealed: ReadonlySet<string>;
  /** The badge tooltip line of a card whose price changed, by card. */
  readonly cardPrice: ReadonlyMap<string, string>;
  /** Whether the board's marks show (carets, traces): off with the local
   *  hide's board switch, when only the tooltip lines remain. */
  readonly marks: boolean;
}

/**
 * The board's recap for `seat`: every change since its last move, worded
 * with the seat names. Null without a seat (a spectator).
 */
export function recapBoard(
  state: RecapState,
  view: Pick<State['view'], 'players' | 'structure'>,
  seat: Seat | null,
  names: readonly [string, string],
  marks: boolean
): RecapBoard | null {
  if (seat === null || !state.base[seat]) return null;
  const chips = new Map(
    priceTicks(state, view, seat).map(
      (tk) => [tk.resource, { up: tk.to > tk.from, text: priceTickText(tk, names) }] as const
    )
  );
  const cardPrice = new Map<string, string>();
  for (const s of view.structure) {
    if (s.state !== 'faceUp' || !s.card || !s.accessible) continue;
    const tick = cardPriceTick(state, view, seat, s.card);
    if (tick) cardPrice.set(s.card, cardPriceText(tick, names));
  }
  return {
    seat,
    prices: view.players[seat].tradePrices,
    chips,
    emptied: new Set(emptiedSince(state, seat)),
    revealed: new Set(revealedSince(state, seat)),
    cardPrice,
    marks
  };
}

const RECAP_KEY = Symbol('recap-board');

/** Hands the board's recap down to the strips and the structure (the game screen). */
export function setRecapBoard(get: () => RecapBoard | null): void {
  setContext(RECAP_KEY, get);
}

/** The board's recap, from the game screen's context; null outside a game. */
export function getRecapBoard(): () => RecapBoard | null {
  return getContext<(() => RecapBoard | null) | undefined>(RECAP_KEY) ?? (() => null);
}
