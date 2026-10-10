// Affordability badges: one of five display states for a cost the server
// already worked out (CostView). Display only. The server has decided
// whether the seat can pay and how; this only reads the breakdown's fields
// and picks the badge that tells it at a glance.
//
//   chain   free through a chain symbol (CostView.chainFree)
//   free    affordable, nothing to pay
//   own     affordable, pays only the card's printed coins (no trade)
//   trade   affordable, buys resource units from the bank
//   short   not affordable (discard and wonder may still be possible)

import type { CostView, SlotView, View } from '$lib/wire';

export const AFFORD_STATES = ['chain', 'free', 'own', 'trade', 'short'] as const;
export type AffordState = (typeof AFFORD_STATES)[number];

export interface Afford {
  state: AffordState;
  /** The coins this payment costs in total (CostView.total). */
  coins: number;
  /** Coins missing when the state is short, else 0. */
  shortfall: number;
  /** The card that makes a chain build free, when there is one. */
  chainFrom?: string;
}

/** The badge for one seat's cost breakdown. */
export function affordOf(cost: CostView): Afford {
  const base = { coins: cost.total, shortfall: cost.shortfall };
  if (cost.chainFree) {
    return cost.chainFrom === undefined
      ? { state: 'chain', ...base }
      : { state: 'chain', ...base, chainFrom: cost.chainFrom };
  }
  if (!cost.affordable) return { state: 'short', ...base };
  if (cost.trade.length > 0) return { state: 'trade', ...base };
  if (cost.total === 0) return { state: 'free', ...base };
  return { state: 'own', ...base };
}

/**
 * The badge of a structure slot for a seat, or undefined when the slot has
 * no cost to show (face down, covered or empty: the server only sends costs
 * for accessible face-up cards).
 */
export function slotAfford(slot: SlotView, seat: number | undefined): Afford | undefined {
  if (seat === undefined || !slot.costs) return undefined;
  const cost = slot.costs[seat];
  return cost ? affordOf(cost) : undefined;
}

/**
 * The seat whose costs a viewer's badges show: the viewer's own seat, or
 * for a spectator the seat whose turn it is.
 */
export function badgeSeat(view: Pick<View, 'viewer' | 'active'>): number {
  return view.viewer ?? view.active;
}

/** Whether badges are dimmed: it is not this seat's move (shown for planning). */
export function badgesDimmed(view: Pick<View, 'toMove'>, seat: number): boolean {
  return view.toMove !== seat;
}
