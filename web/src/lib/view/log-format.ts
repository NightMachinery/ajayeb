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
// A line that names a card, wonder or progress token also carries `parts`:
// the same sentence cut into plain runs and the named items, in reading
// order, so the LogSheet can give each name its detail tooltip. The cut is
// made in the translated message at the items' placeholders, before any
// value is filled in (i18n tSlots), so it follows each locale's word order
// and a player's name, whatever it contains, is never taken for an item.
//
// Player names are not in the view (PlayerView has no `name`; that is
// RoomInfo.seats[].name, a different message), so callers pass them in
// as data to interpolate, not as translated UI copy.

import { cardName, wonderName } from '$lib/components/card/effect-text';
import { ageLabel, t, tSlots, type MessageKey, type MessageParams } from '$lib/i18n';
import type { GameEvent, ScienceSymbol } from '$lib/wire';

export type SeatNames = readonly [string, string];

/** A card, wonder or progress token a log line names, with its name. */
export interface LogItem {
  readonly kind: 'card' | 'wonder' | 'token';
  readonly slug: string;
  readonly text: string;
}

/** A piece of a log line: a plain run of text, or a named item. */
export type LogPart = string | LogItem;

export interface LogLine {
  readonly kind: GameEvent['kind'];
  readonly text: string;
  readonly seat?: 0 | 1;
  /** The text cut around the items it names; absent when it names none. */
  readonly parts?: readonly LogPart[];
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

const card = (slug: string): LogItem => ({ kind: 'card', slug, text: cardName(slug) });
const wonder = (slug: string): LogItem => ({ kind: 'wonder', slug, text: wonderName(slug) });
const token = (slug: string): LogItem => ({ kind: 'token', slug, text: tokenName(slug) });

/**
 * A message that names items: its text with their names filled in, and the
 * same text cut into plain runs and the items. `items` gives the item for
 * each of the message's item placeholders, `params` the rest.
 */
function named<K extends MessageKey, S extends keyof MessageParams<K> & string>(
  key: K,
  params: Omit<MessageParams<K>, S>,
  items: Readonly<Record<S, LogItem>>
): { text: string; parts: LogPart[] } {
  const slots = Object.keys(items) as S[];
  const names = Object.fromEntries(slots.map((slot) => [slot, items[slot].text]));
  const { text, pieces } = tSlots(key, { ...params, ...names } as MessageParams<K>, slots);
  return {
    text,
    parts: pieces.map((piece) => (typeof piece === 'string' ? piece : items[piece.slot as S]))
  };
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
        ...named('board.log.draft', { name }, { wonder: wonder(event.wonder) })
      };
    case 'draftRound':
      if (event.round !== 2) return null;
      return { kind: event.kind, text: t('board.log.draftRound2') };
    case 'ageStart':
      return { kind: event.kind, text: t('board.log.ageStart', { age: ageLabel(event.age ?? 0) }) };
    case 'build':
      if (!event.card) return null;
      return {
        kind: event.kind,
        seat: event.seat as 0 | 1 | undefined,
        ...named('board.log.build', { name }, { card: card(event.card) })
      };
    case 'discard':
      if (!event.card) return null;
      return {
        kind: event.kind,
        seat: event.seat as 0 | 1 | undefined,
        ...named(
          'board.log.discard',
          { name, amount: event.amount ?? 0 },
          { card: card(event.card) }
        )
      };
    case 'wonder':
      if (!event.wonder || !event.card) return null;
      return {
        kind: event.kind,
        seat: event.seat as 0 | 1 | undefined,
        ...named(
          'board.log.wonder',
          { name },
          { wonder: wonder(event.wonder), card: card(event.card) }
        )
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
        text: t('board.log.pawn', {
          name: names[towardMoon ? 1 : 0] || t(towardMoon ? 'board.crest.moon' : 'board.crest.sun')
        })
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
        ...named('board.log.token', { name }, { token: token(event.token) })
      };
    case 'libraryDraw':
      return {
        kind: event.kind,
        seat: event.seat as 0 | 1 | undefined,
        ...named('board.log.libraryDraw', { name }, { wonder: wonder('great-library') })
      };
    case 'destroy':
      if (!event.card) return null;
      return {
        kind: event.kind,
        seat: event.seat as 0 | 1 | undefined,
        ...named('board.log.destroy', { name }, { card: card(event.card) })
      };
    case 'rebuild':
      if (!event.card) return null;
      return {
        kind: event.kind,
        seat: event.seat as 0 | 1 | undefined,
        ...named('board.log.rebuild', { name }, { card: card(event.card) })
      };
    case 'wonderRemoved':
      if (!event.wonder) return null;
      return {
        kind: event.kind,
        seat: event.seat as 0 | 1 | undefined,
        ...named('board.log.wonderRemoved', { name }, { wonder: wonder(event.wonder) })
      };
    case 'effectSkipped':
      return {
        kind: event.kind,
        seat: event.seat as 0 | 1 | undefined,
        ...skipped(event, names, name)
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
 * wonder (or card) that opened it, which the line names. Falls back to the
 * plain line when the event names neither.
 */
function skipped(
  event: GameEvent,
  names: SeatNames,
  name: string
): { text: string; parts?: LogPart[] } {
  const source = event.wonder ? wonder(event.wonder) : event.card ? card(event.card) : undefined;
  const opponent = seatName(names, event.seat === 0 ? 1 : event.seat === 1 ? 0 : undefined);
  switch (event.skipped) {
    case 'destroyCard':
      if (source && opponent)
        return named('board.log.skipped.destroyCard', { opponent }, { source });
      break;
    case 'buildFromDiscard':
      if (source) return named('board.log.skipped.buildFromDiscard', {}, { source });
      break;
    case 'pickBoardToken':
      return { text: t('board.log.skipped.pickBoardToken') };
    case 'pickLibraryToken':
      if (source) return named('board.log.skipped.pickLibraryToken', {}, { source });
      break;
    default:
      break;
  }
  return { text: t('board.log.effectSkipped', { name }) };
}

/**
 * The whole event stream (every state update's events, concatenated in
 * order), grouped by the age each event happened in. An ageStart event
 * opens its own new group, which its own formatted line then joins. Groups
 * with nothing to show (every event in them formatted to null) are left
 * out.
 */
export function groupLogByAge(
  events: readonly GameEvent[],
  names: SeatNames,
  currentAge = 0
): LogAgeGroup[] {
  const groups: LogAgeGroup[] = [{ age: initialLogAge(events, currentAge), lines: [] }];
  for (const event of events) {
    if (event.kind === 'ageStart' && event.age !== undefined) {
      groups.push({ age: event.age, lines: [] });
    }
    const line = formatLogEvent(event, names);
    if (line) groups[groups.length - 1]!.lines.push(line);
  }
  return groups.filter((group) => group.lines.length > 0);
}

/** A scenario can start in any age without an ageStart event. Use the
 * displayed age if no boundary is recorded, or the age preceding the
 * first recorded boundary. Actual draft events always identify age zero. */
export function initialLogAge(events: readonly GameEvent[], currentAge: number): number {
  for (const event of events) {
    if (event.kind === 'draft' || event.kind === 'draftRound') return 0;
    if (event.kind === 'ageStart' && event.age !== undefined) {
      return Math.max(0, event.age - 1);
    }
  }
  return currentAge;
}
