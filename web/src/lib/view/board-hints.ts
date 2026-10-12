// Board-level hints (the hints design, M10.HS4): pure display mappers over
// the server's PlayerView.hints, WonderView.effect and View.hints (the
// science pairs, the looting tokens, the token values, the box tokens, the
// last card of the age and the next starter). The server decides every
// fact; this file only words it from this screen's point of view, for the
// science tracker's slots, the military track's capitals, pawn and looting
// tokens, the wonder rows, chips and sheet, the progress tokens, the age
// chip and the age interstitial.
//
// Each part follows its family (the families this screen shows), and the
// marks on the board (the hourglass, the ready mark, the Law laurel) also
// follow the local hide's board switch. Nothing here is shown once the game
// is over: the server sends no hints then, and the callers pass no screen.

import type {
  HintFamily,
  LootingView,
  OutcomeView,
  PairCardView,
  PlayerHintsView,
  PublicPlace,
  ScienceSymbol,
  SymbolState,
  TokenValueView,
  View,
  WonderEffectView,
  WonderView
} from '$lib/wire';
import { cardBySlug } from '$lib/catalog';
import { cardName, joinList, scienceName, wonderName } from '$lib/components/card/effect-text';
import { tokenName } from '$lib/components/game/names';
import { t } from '$lib/i18n';
import { CAPITAL_NUMERAL } from './threats';
import {
  pipLook,
  type HintNames,
  type HintScreen,
  type PipStyle,
  type PipTone,
  type Seat
} from './hints';
import { outcomeTerms, signed, type OutcomeTerm } from './outcome';

function other(seat: Seat): Seat {
  return seat === 0 ? 1 : 0;
}

function asSeat(n: number | undefined): Seat {
  return n === 1 ? 1 : 0;
}

function isYou(seat: Seat, names: HintNames): boolean {
  return names.you === seat;
}

// --- Where a card is ---------------------------------------------------------

/** A card's public place, in words: "face up on the board", "in Moon's city". */
export function placeText(
  card: string,
  pc: Pick<PairCardView, 'place' | 'seat' | 'inPlay'>,
  names: HintNames
): string {
  const place: PublicPlace = pc.place;
  switch (place) {
    case 'faceUp':
      return t('hints.place.faceUp');
    case 'city':
    case 'tucked': {
      const seat = asSeat(pc.seat);
      if (place === 'tucked') return t('hints.place.tucked');
      return isYou(seat, names)
        ? t('hints.place.cityYou')
        : t('hints.place.cityNamed', { name: names.names[seat] });
    }
    case 'discard':
      return t('hints.place.discard');
    case 'unseenThisAge':
      return pc.inPlay
        ? t('hints.place.unseenThisAgeOdds', { num: pc.inPlay.num, den: pc.inPlay.den })
        : t('hints.place.unseenThisAge');
    case 'unseenLaterAge': {
      const age = cardBySlug(card)?.age ?? 0;
      return pc.inPlay
        ? t('hints.place.laterAgeOdds', { age, num: pc.inPlay.num, den: pc.inPlay.den })
        : t('hints.place.laterAge', { age });
    }
    case 'removedKnown':
      return t('hints.place.removed');
  }
  return '';
}

// --- The science tracker (symbol-status, science-reach) ----------------------

/** One slot of the science tracker: its state and its tooltip. */
export interface SymbolSlotHint {
  readonly state: SymbolState;
  readonly tip: string;
}

/** The science tracker's hints for one seat. */
export interface ScienceHints {
  readonly slots: Readonly<Partial<Record<ScienceSymbol, SymbolSlotHint>>>;
  /** The counter's tooltip: what the seat still needs, and whether it can come. */
  readonly reach: string;
}

function stateText(symbol: ScienceSymbol, state: SymbolState, open: number): string {
  if (symbol === 'scales') {
    switch (state) {
      case 'owned':
      case 'paired':
        return t('hints.science.scalesOwned');
      case 'open':
        return t('hints.science.scalesOpen');
      default:
        return t('hints.science.scalesDead');
    }
  }
  switch (state) {
    case 'paired':
      return t('hints.science.paired');
    case 'live':
      return t('hints.science.live');
    case 'owned':
      return t('hints.science.owned');
    case 'open':
      return t('hints.science.open', { count: open });
    case 'dead':
      return t('hints.science.dead');
  }
}

/**
 * The tracker's hints for `seat`: each slot's state, and a tooltip naming
 * the symbol, its state and where each of its two cards is. Undefined when
 * the science family is not shown or the seat has no hints.
 */
export function scienceHints(
  view: Pick<View, 'players' | 'hints'>,
  seat: Seat,
  families: ReadonlySet<HintFamily>,
  names: HintNames
): ScienceHints | undefined {
  const ph = view.players[seat].hints;
  if (!families.has('science') || !ph || ph.symbols.length === 0) return undefined;
  const pairs = view.hints?.pairs ?? [];
  const slots: Partial<Record<ScienceSymbol, SymbolSlotHint>> = {};
  for (const sh of ph.symbols) {
    const parts = [
      t('hints.science.symbol', { symbol: t(`board.science.${sh.symbol}`) }),
      stateText(sh.symbol, sh.state, sh.openCards)
    ];
    const pair = pairs.find((p) => p.symbol === sh.symbol);
    for (const pc of pair?.cards ?? []) {
      parts.push(
        t('hints.science.card', { card: cardName(pc.card), place: placeText(pc.card, pc, names) })
      );
    }
    slots[sh.symbol] = { state: sh.state, tip: parts.join(' ') };
  }
  return { slots, reach: reachText(ph, seat, names) };
}

/** The counter's tooltip (science-reach). */
export function reachText(ph: PlayerHintsView, seat: Seat, names: HintNames): string {
  const you = isYou(seat, names);
  const name = names.names[seat];
  if (ph.need <= 0) return '';
  const missing = ph.symbols.filter((s) => s.state === 'open').map((s) => scienceName(s.symbol));
  let text: string;
  if (!ph.reachable) {
    text = you ? t('hints.science.unreachableYou') : t('hints.science.unreachableNamed', { name });
  } else {
    const list = joinList(missing, 'disjunction');
    text = you
      ? t('hints.science.needYou', { count: ph.need, list })
      : t('hints.science.needNamed', { count: ph.need, list, name });
  }
  if (ph.lawWins) {
    text += ' ' + (you ? t('hints.science.lawYou') : t('hints.science.lawNamed', { name }));
  }
  return text;
}

// --- The military track (capital-distance, capital-safe, looting) -----------

/** One capital's hint: how far the opponent is, and whether it is safe. */
export interface CapitalHint {
  /** Shields the opponent still needs, shown at CAPITAL_NUMERAL or fewer. */
  readonly toCapital: number;
  readonly numeral: boolean;
  readonly safe: boolean;
  /** The capital's tooltip line about it. */
  readonly text: string;
}

/** The military track's hints: both capitals, and each looting token's line. */
export interface TrackHints {
  readonly capitals: readonly [CapitalHint | undefined, CapitalHint | undefined];
  /** By looting space: the coins its victim would lose now. */
  readonly wouldLose: ReadonlyMap<number, number>;
  readonly names: HintNames;
}

function capitalHint(
  ph: PlayerHintsView | undefined,
  seat: Seat,
  names: HintNames
): CapitalHint | undefined {
  if (!ph) return undefined;
  const attacker = other(seat);
  const name = names.names[attacker];
  const capital = isYou(seat, names)
    ? t('hints.capital.yours')
    : t('hints.capital.named', { name: names.names[seat] });
  let text = isYou(attacker, names)
    ? t('hints.capital.needYou', { count: ph.toCapital, capital })
    : t('hints.capital.needNamed', { count: ph.toCapital, capital, name });
  if (ph.capitalSafe) {
    text +=
      ' ' +
      (isYou(attacker, names)
        ? t('hints.capital.safeYou')
        : t('hints.capital.safeNamed', { name }));
  }
  return {
    toCapital: ph.toCapital,
    numeral: ph.toCapital <= CAPITAL_NUMERAL && !ph.capitalSafe,
    safe: ph.capitalSafe,
    text
  };
}

/** The track's hints, or undefined when the military family is not shown. */
export function trackHints(
  view: Pick<View, 'players' | 'hints'>,
  families: ReadonlySet<HintFamily>,
  names: HintNames
): TrackHints | undefined {
  if (!families.has('military')) return undefined;
  const [a, b] = [view.players[0].hints, view.players[1].hints];
  if (!a && !b) return undefined;
  return {
    capitals: [capitalHint(a, 0, names), capitalHint(b, 1, names)],
    wouldLose: new Map((view.hints?.looting ?? []).map((l) => [l.at, l.wouldLose])),
    names
  };
}

/** A looting token's tooltip (looting-token-tooltip). */
export function lootText(
  loot: LootingView,
  wouldLose: number | undefined,
  names: HintNames
): string {
  if (!loot.present) return t('hints.loot.taken');
  const victim = asSeat(loot.seat);
  const name = names.names[victim];
  let text = isYou(victim, names)
    ? t('hints.loot.triggerYou', { count: loot.value })
    : t('hints.loot.triggerNamed', { count: loot.value, name });
  if (wouldLose !== undefined && wouldLose < loot.value) {
    text +=
      ' ' +
      (isYou(victim, names)
        ? t('hints.loot.clampYou', { count: wouldLose })
        : t('hints.loot.clampNamed', { count: wouldLose, name }));
  }
  return text;
}

// --- Wonders (wonder-effect, wonder-ready-glyph, wonder-race-seventh) --------

/** A wonder's hints: what building it would do now, and its marks. */
export interface WonderHint {
  /** What building it would do now, as outcome terms (the owner's point of view). */
  readonly terms: readonly OutcomeTerm[];
  /** The Great Library: the tokens it could draw, named. */
  readonly note?: string;
  /** The race for the last wonder: the hourglass, solid when the owner can afford it. */
  readonly hourglass?: { readonly solid: boolean; readonly tip: string };
  /** Affordable at today's prices, though no choice builds it now. */
  readonly ready?: { readonly style: PipStyle; readonly tone: PipTone; readonly tip: string };
}

/** The seat that owns a wonder in the view, or undefined. */
export function wonderOwner(view: Pick<View, 'players'>, slug: string): Seat | undefined {
  if (view.players[0].wonders.some((w) => w.wonder === slug)) return 0;
  if (view.players[1].wonders.some((w) => w.wonder === slug)) return 1;
  return undefined;
}

/**
 * A wonder effect as an outcome, so the outcome line words it: the coins
 * after (now plus the change), the points, the pawn, a win, the extra turn
 * and the choice it opens.
 */
export function effectOutcome(
  e: WonderEffectView,
  coinsNow: readonly [number, number]
): OutcomeView {
  return {
    coins: [coinsNow[0] + e.coins[0], coinsNow[1] + e.coins[1]],
    score: [e.score[0], e.score[1]],
    ...(e.pawn ? { pawn: e.pawn } : {}),
    ...(e.wins ? { wins: e.wins } : {}),
    ...(e.opens ? { opens: e.opens } : {}),
    ...(e.extraTurn !== 'none'
      ? { tempo: { extraTurn: 'gain', source: e.extraTurn, lastCard: false } }
      : {})
  };
}

/**
 * A wonder's hints for the wonder rows, chips and sheet. `buildable` is
 * whether a choice of the view builds it now (the decider's own glow);
 * the ready mark is for the other cases.
 */
export function wonderHint(
  view: Pick<View, 'players' | 'hints' | 'age'>,
  w: WonderView,
  seat: Seat,
  screen: HintScreen,
  buildable = false
): WonderHint | undefined {
  const f = screen.families;
  if (!f.has('wonders') || w.built || w.removed) return undefined;
  const names = screen.names;
  const coins: [number, number] = [view.players[0].coins, view.players[1].coins];
  const terms: OutcomeTerm[] = [];
  if (w.effect) {
    terms.push(
      ...outcomeTerms(effectOutcome(w.effect, coins), {
        mover: seat,
        coins,
        age: view.age,
        families: f,
        names
      })
    );
  }
  // great-library-odds: the tokens it could draw, named (owner Q1 B), for
  // the wonder sheet and the row's spoken label.
  let note: string | undefined;
  const box = view.hints?.boxTokens ?? [];
  if (w.effect?.opens === 'pickLibraryToken' && f.has('tokens') && box.length > 0) {
    note = t('hints.wonder.library', {
      count: w.effect.pool,
      total: box.length,
      list: joinList(box.map(tokenName), 'conjunction')
    });
  }
  const owner = isYou(seat, names) ? undefined : names.names[seat];
  const affordable = w.cost?.affordable ?? false;
  let hourglass: WonderHint['hourglass'];
  if ((view.hints?.wondersLeft ?? 0) === 1) {
    const parts = [t('hints.wonder.lastSlot')];
    if (w.effect?.removes) {
      const removed = w.effect.removes;
      const removedOwner = wonderOwner(view, removed) ?? other(seat);
      parts.push(
        isYou(removedOwner, names)
          ? t('hints.wonder.removesYour', { wonder: wonderName(removed) })
          : t('hints.wonder.removes', {
              wonder: wonderName(removed),
              name: names.names[removedOwner]
            })
      );
    }
    if (affordable) {
      parts.push(
        owner ? t('hints.wonder.affordNamed', { name: owner }) : t('hints.wonder.affordYou')
      );
    }
    hourglass = { solid: affordable, tip: parts.join(' ') };
  }
  let ready: WonderHint['ready'];
  if (affordable && !buildable && screen.boardMarks) {
    ready = {
      ...pipLook(screen.perspective, seat),
      tip: owner ? t('hints.wonder.readyNamed', { name: owner }) : t('hints.wonder.readyYou')
    };
  }
  if (terms.length === 0 && !hourglass && !ready && !note) return undefined;
  return {
    terms,
    ...(note ? { note } : {}),
    ...(hourglass && screen.boardMarks ? { hourglass } : {}),
    ...(ready ? { ready } : {})
  };
}

/** A seat's wonder hints by wonder slug. */
export function seatWonderHints(
  view: Pick<View, 'players' | 'hints' | 'age'>,
  seat: Seat,
  screen: HintScreen,
  buildable?: ReadonlySet<string>
): ReadonlyMap<string, WonderHint> {
  const out = new Map<string, WonderHint>();
  for (const w of view.players[seat].wonders) {
    const h = wonderHint(view, w, seat, screen, buildable?.has(w.wonder) ?? false);
    if (h) out.set(w.wonder, h);
  }
  return out;
}

/** The wonder button's marks for a seat: the hourglass and the ready ring. */
export interface WonderButtonHint {
  readonly hourglass: boolean;
  /** Wonders the seat could afford at today's prices (the ready mark). */
  readonly ready: number;
  readonly tip: string;
}

/** The phone strip's wonder button hints for `seat`. */
export function wonderButtonHint(
  view: Pick<View, 'players' | 'hints' | 'age'>,
  seat: Seat,
  screen: HintScreen,
  buildable?: ReadonlySet<string>
): WonderButtonHint | undefined {
  const hints = [...seatWonderHints(view, seat, screen, buildable).values()];
  const hourglass = hints.some((h) => h.hourglass);
  const ready = hints.filter((h) => h.ready).length;
  if (!hourglass && ready === 0) return undefined;
  const name = isYou(seat, screen.names) ? undefined : screen.names.names[seat];
  const parts: string[] = [];
  if (hourglass) parts.push(t('hints.wonder.lastSlot'));
  if (ready > 0) {
    parts.push(
      name
        ? t('hints.wonder.readyCountNamed', { count: ready, name })
        : t('hints.wonder.readyCountYou', { count: ready })
    );
  }
  return { hourglass, ready, tip: parts.join(' ') };
}

/** The wonder sheet's lines about the race and the age's end. */
export function wonderSheetLines(
  view: Pick<View, 'players' | 'hints' | 'structure' | 'phase'>,
  families: ReadonlySet<HintFamily>
): string[] {
  const out: string[] = [];
  const left = view.hints?.wondersLeft;
  if (families.has('wonders') && left !== undefined && left <= 2) {
    const unbuilt = [...view.players[0].wonders, ...view.players[1].wonders].filter(
      (w) => !w.built && !w.removed
    ).length;
    out.push(t('hints.wonder.slotsLeft', { count: left, unbuilt }));
  }
  if (families.has('tempo') && view.phase === 'play') {
    const r = view.structure.filter((s) => s.state !== 'empty').length;
    if (r > 0 && r <= 3) out.push(t('hints.age.cardsLeft', { count: r }));
  }
  return out;
}

// --- Progress tokens (token-value-now, token-deny-note, law-token-marks) -----

/** A board token's hints: its value for each seat, and the Law laurel. */
export interface TokenHint {
  /** One line per seat, the perspective seat first. */
  readonly lines: readonly string[];
  /** The Law laurel: Law would be this seat's sixth symbol. */
  readonly mark?: { readonly style: PipStyle; readonly tone: PipTone; readonly tip: string };
}

/** One seat's value of a token, in words. */
export function tokenValueText(v: TokenValueView, names: HintNames): string {
  if (v.wins) return t('hints.token.wins');
  const parts: string[] = [];
  if (v.coins > 0) parts.push(t('hints.token.coins', { count: v.coins }));
  if (v.vp !== 0) parts.push(t('hints.token.vp', { delta: signed(v.vp) }));
  switch (v.token) {
    case 'masonry':
    case 'architecture':
      parts.push(
        v.savings > 0 ? t('hints.token.saves', { count: v.savings }) : t('hints.token.savesNothing')
      );
      break;
    case 'economy':
      parts.push(
        t('hints.token.economy', { count: v.savings, name: names.names[other(asSeat(v.seat))] })
      );
      break;
    case 'urbanism':
      parts.push(t('hints.token.chains', { count: v.count }));
      break;
    case 'theology':
      parts.push(t('hints.token.theology', { count: v.count }));
      break;
    case 'strategy':
      parts.push(t('hints.token.strategy'));
      break;
  }
  return parts.length > 0 ? joinList(parts, 'conjunction') : t('hints.token.nothing');
}

/** The hints of board token `slug`, from perspective `screen`. */
export function tokenHint(
  view: Pick<View, 'players' | 'hints'>,
  slug: string,
  screen: HintScreen
): TokenHint | undefined {
  const names = screen.names;
  const lines: string[] = [];
  if (screen.families.has('tokens')) {
    const p = screen.perspective.seat;
    for (const seat of [p, other(p)] as const) {
      const v = view.hints?.tokens.find((x) => x.token === slug && x.seat === seat);
      if (!v) continue;
      const value = tokenValueText(v, names);
      lines.push(
        isYou(seat, names)
          ? t('hints.token.forYou', { value })
          : t('hints.token.forNamed', { name: names.names[seat], value })
      );
    }
  }
  let mark: TokenHint['mark'];
  if (slug === 'law' && screen.families.has('science') && screen.boardMarks) {
    for (const seat of [screen.perspective.seat, other(screen.perspective.seat)] as const) {
      if (!view.players[seat].hints?.lawWins) continue;
      mark = {
        ...pipLook(screen.perspective, seat),
        tip: isYou(seat, names)
          ? t('hints.science.lawYou')
          : t('hints.science.lawNamed', { name: names.names[seat] })
      };
      break;
    }
  }
  if (lines.length === 0 && !mark) return undefined;
  return { lines, ...(mark ? { mark } : {}) };
}

// --- The age chip (age-countdown, next-starter) -------------------------------

/** The age chip's tooltip: who takes the last card, and who starts next. */
export function ageTip(
  view: Pick<View, 'hints' | 'age' | 'phase'>,
  cardsLeft: number,
  families: ReadonlySet<HintFamily>,
  names: HintNames
): string | undefined {
  if (!families.has('tempo') || !view.hints || view.age === 0) return undefined;
  const parts: string[] = [];
  const last = view.hints.lastCardTo;
  if (last !== undefined) {
    const seat = asSeat(last);
    parts.push(
      isYou(seat, names)
        ? t('hints.age.lastYou')
        : t('hints.age.lastNamed', { name: names.names[seat] })
    );
    if (cardsLeft <= 2) parts.push(t('hints.age.lastLost'));
  }
  const ns = view.hints.nextStarter;
  if (ns && view.age < 3) {
    const age = view.age + 1;
    if (ns.chooser !== undefined) {
      const seat = asSeat(ns.chooser);
      parts.push(
        isYou(seat, names)
          ? t('hints.age.chooserYou', { age })
          : t('hints.age.chooserNamed', { name: names.names[seat], age })
      );
    } else if (ns.lastActiveStarts) {
      parts.push(t('hints.age.lastStarts', { age }));
    } else {
      parts.push(t('hints.age.lastChooses', { age }));
    }
  }
  return parts.length > 0 ? parts.join(' ') : undefined;
}

// --- The age interstitial (symbol-status's dead pairs) ------------------------

/**
 * The pairs an ended age killed: a seat holds one card of a symbol, and the
 * other card was set aside in an age that is over. One line per such pair.
 */
export function deadPairLines(
  view: Pick<View, 'players' | 'hints'>,
  endedAge: number,
  families: ReadonlySet<HintFamily>,
  names: HintNames
): string[] {
  if (!families.has('science')) return [];
  const out: string[] = [];
  for (const pair of view.hints?.pairs ?? []) {
    const gone = pair.cards.find(
      (c) => c.place === 'removedKnown' && cardBySlug(c.card)?.age === endedAge
    );
    const held = pair.cards.find((c) => c.place === 'city');
    if (!gone || !held) continue;
    const seat = asSeat(held.seat);
    out.push(
      isYou(seat, names)
        ? t('hints.age.deadPairYou', { card: cardName(gone.card), own: cardName(held.card) })
        : t('hints.age.deadPairNamed', {
            card: cardName(gone.card),
            own: cardName(held.card),
            name: names.names[seat]
          })
    );
  }
  return out;
}
