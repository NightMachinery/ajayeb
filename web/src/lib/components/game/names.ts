// Names the game screen shows: progress tokens by engine id, and the two
// players' display names from the room (user text, never translated), with
// the crest names as the fallback for an empty seat name.

import { t } from '$lib/i18n';
import type { RoomInfo } from '$lib/wire';

/** A progress token's translated name, or the slug for an unknown one. */
export function tokenName(slug: string): string {
  switch (slug) {
    case 'agriculture':
      return t('token.agriculture');
    case 'architecture':
      return t('token.architecture');
    case 'economy':
      return t('token.economy');
    case 'law':
      return t('token.law');
    case 'masonry':
      return t('token.masonry');
    case 'mathematics':
      return t('token.mathematics');
    case 'philosophy':
      return t('token.philosophy');
    case 'strategy':
      return t('token.strategy');
    case 'theology':
      return t('token.theology');
    case 'urbanism':
      return t('token.urbanism');
    default:
      return slug;
  }
}

/** Both seats' display names, by seat. */
export function seatNames(room: Pick<RoomInfo, 'seats'> | null | undefined): [string, string] {
  const name = (seat: 0 | 1) => {
    const given = room?.seats.find((s) => s.seat === seat)?.name;
    return given || t(seat === 0 ? 'board.crest.sun' : 'board.crest.moon');
  };
  return [name(0), name(1)];
}
