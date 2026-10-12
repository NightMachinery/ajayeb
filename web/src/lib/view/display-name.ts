// display-name.ts: the name a seat shows (docs/protocol.md, SeatInfo.nameNo).
// When a player's name equals one that was in the game first, the server
// numbers it (2, 3 and so on, stored with the room so it never changes when
// people leave), and every place a seat's name shows writes the number
// after it: "Sara" and "Sara 2". Pure display mapping: the server decides
// the numbers, this only writes them.

import { formatNumber, t } from '$lib/i18n';
import type { SeatInfo } from '$lib/wire';

/** A display name with its number, when it has one (nameNo 2 and up). */
export function numberedName(name: string, nameNo: number | undefined): string {
  if (!nameNo || nameNo < 2 || !Number.isInteger(nameNo)) return name;
  return t('online.identity.numbered', { name, number: formatNumber(nameNo) });
}

/**
 * A seat's display name: its name (trimmed) with its number, or the
 * fallback (a crest name, a bot's persona) when it has no name.
 */
export function seatDisplayName(
  info: Pick<SeatInfo, 'name' | 'nameNo'> | null | undefined,
  fallback: string
): string {
  const name = info?.name?.trim();
  return name ? numberedName(name, info?.nameNo) : fallback;
}

/**
 * Whether a seat is this connection's own and should be marked as such: it
 * holds the seat, and it is not a hot-seat device (which holds both seats,
 * so the mark would say nothing).
 */
export function isSelfSeat(
  info: Pick<SeatInfo, 'you'> | null | undefined,
  mode: string | null | undefined
): boolean {
  return !!info?.you && mode !== 'hotseat';
}
