// Names the game screen shows: progress tokens by engine id, and the two
// players' display names from the room (user text, never translated), with
// the crest names as the fallback for an empty seat name. A bot seat is
// named by its persona, from its level: Apprentice (easy), Scholar (normal)
// and Master (hard).

import { t } from '$lib/i18n';
import type { BotLevel, RoomInfo } from '$lib/wire';

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

/** A bot's persona name for its level. */
export function personaName(level: BotLevel): string {
  switch (level) {
    case 'easy':
      return t('game.persona.easy');
    case 'normal':
      return t('game.persona.normal');
    case 'hard':
      return t('game.persona.hard');
  }
}

/** Both seats' display names, by seat; a bot seat's is its persona. */
export function seatNames(room: Pick<RoomInfo, 'seats'> | null | undefined): [string, string] {
  const name = (seat: 0 | 1) => {
    const info = room?.seats.find((s) => s.seat === seat);
    if (info?.kind === 'bot' && info.bot !== undefined) return personaName(info.bot);
    const given = info?.name?.trim();
    return given || t(seat === 0 ? 'board.crest.sun' : 'board.crest.moon');
  };
  return [name(0), name(1)];
}

/** The seat a bot plays in this room, if one does. */
export function botSeat(room: Pick<RoomInfo, 'seats'> | null | undefined): 0 | 1 | undefined {
  const seat = room?.seats.find((s) => s.kind === 'bot')?.seat;
  return seat === 0 || seat === 1 ? seat : undefined;
}
