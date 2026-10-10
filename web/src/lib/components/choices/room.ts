// Who is who, for the draft, the choice panels and the game-over screen.
// Their props are fixed ({view, canAct, onact} and the like), so the seat
// names and the room mode come from the page's game context when there is
// one, and fall back to the crest names (Sun, Moon) when there is none, as
// in the gallery. Display only: nothing here decides anything.

import { t } from '$lib/i18n';
import { getGameContext, type GameContext } from '$lib/session/context';
import type { Mode, RoomInfo } from '$lib/wire';

export type Seat = 0 | 1;
export type SeatNames = readonly [string, string];

/**
 * The page's game context, or null outside a room page. Call it while the
 * component initialises, as with any context lookup.
 */
export function tryGameContext(): GameContext | null {
  try {
    return getGameContext();
  } catch {
    return null;
  }
}

/** The crest name of a seat: seat 0 is the Sun, seat 1 the Moon. */
export function crestName(seat: Seat): string {
  return seat === 0 ? t('board.crest.sun') : t('board.crest.moon');
}

/** The players' display names by seat, with the crest name for a blank one. */
export function seatNames(room: RoomInfo | null | undefined): SeatNames {
  const nameOf = (seat: Seat): string => {
    const name = room?.seats.find((s) => s.seat === seat)?.name?.trim();
    return name ? name : crestName(seat);
  };
  return [nameOf(0), nameOf(1)];
}

/** The other seat. */
export function otherSeat(seat: Seat): Seat {
  return seat === 0 ? 1 : 0;
}

/** A wire seat number as a Seat (the engine only ever sends 0 or 1). */
export function asSeat(n: number): Seat {
  return n === 1 ? 1 : 0;
}

/**
 * Whether messages about this seat should say "you": the view belongs to
 * that seat and the device is not shared. A hot-seat device holds both
 * seats, so it always names the players.
 */
export function isYou(seat: number, viewer: number | undefined, mode: Mode | undefined): boolean {
  return mode !== 'hotseat' && viewer === seat;
}
