// A seat's display name and its place relative to the viewer, for labels
// and tooltips. A human's name is user text (shown with dir="auto"); a bot
// takes its level's persona name from the root tables, as Duel's bots do.

import { formatNumber, t } from '$lib/i18n';
import type { Seat } from '$lib/wire/haftgane.gen';
import type { TableState } from '$lib/wire/platform.gen';
import { ht } from '../i18n/messages.svelte';
import { humanAt } from './model';

/** A bot level's persona name. */
export function persona(level: string | undefined): string {
  switch (level) {
    case 'easy':
      return t('game.persona.easy');
    case 'normal':
      return t('game.persona.normal');
    case 'hard':
      return t('game.persona.hard');
    default:
      return ht('hg.screen.botName');
  }
}

/** A name and its number among those who share it ("Apprentice 2"). */
function numbered(name: string, n: number): string {
  return n > 1 ? `${name} ${formatNumber(n)}` : name;
}

/** A human's display name, numbered among those who share it. */
export function humanName(h: { name: string; nameNo: number }): string {
  return numbered(h.name, h.nameNo);
}

/** A seat's display name. Bots of one level share a persona name, so each
 *  after the first in seat order takes a number, as humans' names do. */
export function seatName(frame: TableState, seat: number): string {
  const p = frame.seats[seat];
  if (p?.kind === 'bot') {
    const same = frame.seats.filter((s) => s.seat <= seat && s.kind === 'bot' && s.bot === p.bot);
    return numbered(persona(p.bot), same.length);
  }
  const h = humanAt(frame, seat);
  if (h) return humanName(h);
  return ht('hg.screen.seat', { seat: seat + 1 });
}

export function isBot(frame: TableState, seat: number): boolean {
  return frame.seats[seat]?.kind === 'bot';
}

/** Where a seat sits from the viewer: "your left neighbour", "2 seats to
 *  your right", or its number for a spectator. */
export function where(seat: Seat, count: number): string {
  switch (seat.relation) {
    case 'self':
      return ht('hg.screen.where.self');
    case 'left':
      return ht('hg.screen.where.left');
    case 'right':
      return ht('hg.screen.where.right');
    case 'other':
      return seat.relative <= count - seat.relative
        ? ht('hg.screen.where.farLeft', { count: seat.relative })
        : ht('hg.screen.where.farRight', { count: count - seat.relative });
    default:
      return ht('hg.screen.where.seat', { seat: seat.seat + 1 });
  }
}
