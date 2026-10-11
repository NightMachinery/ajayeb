// Display mappers for a seat's wonders: how many are built out of how many
// can still count (the phone strip's wonder button, "1/4"), and which of
// them the view's own choices let the player build right now (the
// buildable mark on that button, in the wonder sheet and in the desktop
// city panel). Nothing here decides what is legal: a wonder is buildable
// exactly when a `wonder` choice naming it is among the view's choices.

import type { View, WonderView } from '$lib/wire';

export type Seat = 0 | 1;

/** Built wonders over the wonders still in play for one seat. */
export interface WonderTally {
  built: number;
  /** The seat's wonders minus any that left the game unbuilt (the seventh
   *  wonder rule), so the count never asks for a wonder that cannot be built. */
  total: number;
}

/** How many of a seat's wonders are built, out of those still in play. */
export function wonderTally(player: { readonly wonders: readonly WonderView[] }): WonderTally {
  let built = 0;
  let total = 0;
  for (const w of player.wonders) {
    if (w.removed) continue;
    total += 1;
    if (w.built) built += 1;
  }
  return { built, total };
}

/**
 * The wonders this seat can build now: those named by a `wonder` choice of
 * the view, when the seat is the one to move. The choices always belong to
 * the seat to move (and are empty for a viewer who cannot act), so the
 * other seat gets an empty set.
 */
export function buildableWonders(
  view: Pick<View, 'choices' | 'toMove'>,
  seat: Seat
): ReadonlySet<string> {
  const out = new Set<string>();
  if (view.toMove !== seat) return out;
  for (const c of view.choices) {
    if (c.kind === 'wonder' && c.wonder !== undefined) out.add(c.wonder);
  }
  return out;
}
