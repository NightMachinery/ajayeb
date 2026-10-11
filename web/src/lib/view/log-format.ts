// log-format.ts: turns the server's structured GameEvents into the text the
// LogSheet shows, grouped by age. Pure display mapping, never a rules
// decision: it reads only the fields each event kind already carries (see
// view.gen.ts's GameEvent doc comment). Building and wonder names come
// from the translated 'card.name.*' and 'wonder.name.*' messages (through
// components/card/effect-text.ts). Four purely mechanical kinds (pay, reveal,
// coins, gameOver) are left out of the text log: their effect already shows
// up in the line for the move that caused them (a build, a discard, a
// wonder, a token), or, for gameOver, in the GameOver screen itself.
//
// Player names are not in the view (PlayerView has no `name`; that is
// RoomInfo.seats[].name, a different message), so callers pass them in
// as data to interpolate, not as translated UI copy.

import { cardName, wonderName } from '$lib/components/card/effect-text';
import { t } from '$lib/i18n';
import type { GameEvent, ScienceSymbol } from '$lib/wire';

export type SeatNames = readonly [string, string];

export interface LogLine {
  readonly kind: GameEvent['kind'];
  readonly text: string;
  readonly seat?: 0 | 1;
}

export interface LogAgeGroup {
  /** 0 is the wonder draft, before the first ageStart event. */
  readonly age: number;
  readonly lines: LogLine[];
}

function seatName(names: SeatNames, seat: number | undefined): string {
  return seat === 0 || seat === 1 ? names[seat] : '';
}

function tokenName(slug: string): string {
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

function scienceSymbolName(symbol: ScienceSymbol): string {
  switch (symbol) {
    case 'globe':
      return t('board.science.globe');
    case 'scales':
      return t('board.science.scales');
    case 'sundial':
      return t('board.science.sundial');
    case 'mortar':
      return t('board.science.mortar');
    case 'set-square':
      return t('board.science.set-square');
    case 'quill':
      return t('board.science.quill');
    case 'wheel':
      return t('board.science.wheel');
  }
}

/**
 * One event, translated to one log line, or null for a kind this log does
 * not show on its own (see the file comment).
 */
export function formatLogEvent(event: GameEvent, names: SeatNames): LogLine | null {
  const name = seatName(names, event.seat);
  switch (event.kind) {
    case 'draft':
      if (!event.wonder) return null;
      return {
        kind: event.kind,
        seat: event.seat as 0 | 1 | undefined,
        text: t('board.log.draft', { name, wonder: wonderName(event.wonder) })
      };
    case 'draftRound':
      if (event.round !== 2) return null;
      return { kind: event.kind, text: t('board.log.draftRound2') };
    case 'ageStart':
      return { kind: event.kind, text: t('board.log.ageStart', { age: event.age ?? 0 }) };
    case 'build':
      if (!event.card) return null;
      return {
        kind: event.kind,
        seat: event.seat as 0 | 1 | undefined,
        text: t('board.log.build', { name, card: cardName(event.card) })
      };
    case 'discard':
      if (!event.card) return null;
      return {
        kind: event.kind,
        seat: event.seat as 0 | 1 | undefined,
        text: t('board.log.discard', {
          name,
          card: cardName(event.card),
          amount: event.amount ?? 0
        })
      };
    case 'wonder':
      if (!event.wonder || !event.card) return null;
      return {
        kind: event.kind,
        seat: event.seat as 0 | 1 | undefined,
        text: t('board.log.wonder', {
          name,
          wonder: wonderName(event.wonder),
          card: cardName(event.card)
        })
      };
    case 'pay':
    case 'reveal':
    case 'coins':
    case 'gameOver':
      return null;
    case 'pawn': {
      if (event.to === undefined) return null;
      const towardMoon = event.to > (event.from ?? 0);
      return {
        kind: event.kind,
        text: t('board.log.pawn', { crest: t(towardMoon ? 'board.crest.moon' : 'board.crest.sun') })
      };
    }
    case 'loot':
      return {
        kind: event.kind,
        seat: event.seat as 0 | 1 | undefined,
        text: t('board.log.loot', { name, amount: event.amount ?? 0 })
      };
    case 'pair':
      if (!event.symbol) return null;
      return {
        kind: event.kind,
        seat: event.seat as 0 | 1 | undefined,
        text: t('board.log.pair', { name, symbol: scienceSymbolName(event.symbol) })
      };
    case 'token':
      if (!event.token) return null;
      return {
        kind: event.kind,
        seat: event.seat as 0 | 1 | undefined,
        text: t('board.log.token', { name, token: tokenName(event.token) })
      };
    case 'libraryDraw':
      return {
        kind: event.kind,
        seat: event.seat as 0 | 1 | undefined,
        text: t('board.log.libraryDraw', { name })
      };
    case 'destroy':
      if (!event.card) return null;
      return {
        kind: event.kind,
        seat: event.seat as 0 | 1 | undefined,
        text: t('board.log.destroy', { name, card: cardName(event.card) })
      };
    case 'rebuild':
      if (!event.card) return null;
      return {
        kind: event.kind,
        seat: event.seat as 0 | 1 | undefined,
        text: t('board.log.rebuild', { name, card: cardName(event.card) })
      };
    case 'wonderRemoved':
      if (!event.wonder) return null;
      return {
        kind: event.kind,
        seat: event.seat as 0 | 1 | undefined,
        text: t('board.log.wonderRemoved', { name, wonder: wonderName(event.wonder) })
      };
    case 'effectSkipped':
      return {
        kind: event.kind,
        seat: event.seat as 0 | 1 | undefined,
        text: skippedText(event, names, name)
      };
    case 'extraTurn':
      return {
        kind: event.kind,
        seat: event.seat as 0 | 1 | undefined,
        text: t('board.log.extraTurn', { name })
      };
    case 'extraTurnLost':
      return {
        kind: event.kind,
        seat: event.seat as 0 | 1 | undefined,
        text: t('board.log.extraTurnLost', { name })
      };
    case 'starterChosen':
      if (event.starter === undefined) return null;
      return {
        kind: event.kind,
        seat: event.seat as 0 | 1 | undefined,
        text: t('board.log.starterChosen', { name, starter: seatName(names, event.starter) })
      };
    case 'resign':
      return {
        kind: event.kind,
        seat: event.seat as 0 | 1 | undefined,
        text: t('board.log.resign', { name })
      };
    default:
      return null;
  }
}

/**
 * Why a choice was skipped, from the choice it would have opened and the
 * wonder (or card) that opened it. Falls back to the plain line when the
 * event names neither.
 */
function skippedText(event: GameEvent, names: SeatNames, name: string): string {
  const source = event.wonder
    ? wonderName(event.wonder)
    : event.card
      ? cardName(event.card)
      : undefined;
  const opponent = seatName(names, event.seat === 0 ? 1 : event.seat === 1 ? 0 : undefined);
  switch (event.skipped) {
    case 'destroyCard':
      if (source && opponent) return t('board.log.skipped.destroyCard', { source, opponent });
      break;
    case 'buildFromDiscard':
      if (source) return t('board.log.skipped.buildFromDiscard', { source });
      break;
    case 'pickBoardToken':
      return t('board.log.skipped.pickBoardToken');
    case 'pickLibraryToken':
      if (source) return t('board.log.skipped.pickLibraryToken', { source });
      break;
    default:
      break;
  }
  return t('board.log.effectSkipped', { name });
}

/**
 * The whole event stream (every state update's events, concatenated in
 * order), grouped by the age each event happened in. An ageStart event
 * opens its own new group, which its own formatted line then joins. Groups
 * with nothing to show (every event in them formatted to null) are left
 * out.
 */
export function groupLogByAge(events: readonly GameEvent[], names: SeatNames): LogAgeGroup[] {
  const groups: LogAgeGroup[] = [{ age: 0, lines: [] }];
  for (const event of events) {
    if (event.kind === 'ageStart' && event.age !== undefined) {
      groups.push({ age: event.age, lines: [] });
    }
    const line = formatLogEvent(event, names);
    if (line) groups[groups.length - 1]!.lines.push(line);
  }
  return groups.filter((group) => group.lines.length > 0);
}
