// Per-card hints (the hints design, M10.HS2): pure display mappers over the
// server's SlotView.hints, HintsView.discard and PendingView.cardHints. The
// server decides every fact (a pair, a win, looting, price shifts, savings,
// the last producer); this file only picks which pip a card shows, says
// whose fact it is from this screen's point of view, and words the sheet
// lines. No rules: nothing here derives a cost, a symbol count or a move.
//
// The pip budget: one pip per card on its top-end corner, the highest of
// the priority table below, and a hairline "more" dot when the card carries
// another hint. The pip's tooltip lists every hint on that card, in priority
// order.
//
//   own win, opponent win, own pair, opponent pair, own loot, opponent loot
//
// (The gift chevron of the tempo family slots in between pair and loot with
// a later slice.)
//
// Point of view (the perspective): a seat's own screen draws its facts solid
// and the opponent's outlined in the opponent's crest colour; a hot-seat
// device does the same for the seat to move, tinted with that seat's crest;
// a spectator sees each seat's pips in that seat's crest tint, never solid
// or outlined. Shape carries the meaning and fill the side, so colour is
// never the only cue.
//
// Every hint follows the families this screen shows (visibleFamilies: the
// game's settings minus the local hide), and board pips also respect the
// local "hide hint marks on the board".

import type {
  CardHints,
  DiscardHintView,
  HintCode,
  HintFamily,
  Mode,
  PendingCardHint,
  HintSettings,
  PriceShift,
  RoomInfo,
  SlotView,
  View
} from '$lib/wire';
import { cardBySlug, wonderBySlug } from '$lib/catalog';
import { seatNames } from '$lib/components/choices/room';
import { visibleFamilies, type LocalHide } from '$lib/hints/families';
import { cardName, resourceName, scienceName } from '$lib/components/card/effect-text';
import { joinPhrases, t } from '$lib/i18n';

export type Seat = 0 | 1;

/** Which seat's facts this screen shows as its own. */
export type Perspective =
  /** A player's own screen (online, against the bot). */
  | { readonly kind: 'seat'; readonly seat: Seat }
  /** A shared hot-seat device: the seat to move. */
  | { readonly kind: 'hotseat'; readonly seat: Seat }
  /** A spectator: `seat` is only the reference seat for ordering (the active one). */
  | { readonly kind: 'spectator'; readonly seat: Seat };

/** How a pip is filled: solid (the viewer's), outlined (the opponent's), or a crest tint (a spectator). */
export type PipStyle = 'solid' | 'outline' | 'tint';
/** The colour family of a pip: gold for a player's own, a crest for the rest. */
export type PipTone = 'gold' | 'sun' | 'moon';

/** The family each pip code belongs to. */
export const CODE_FAMILY: Readonly<Record<HintCode, HintFamily>> = {
  win: 'threats',
  pair: 'science',
  loot: 'military'
};

/** The pip priority, highest first: [code, side]. */
export const PIP_PRIORITY: readonly (readonly [HintCode, 'own' | 'opp'])[] = [
  ['win', 'own'],
  ['win', 'opp'],
  ['pair', 'own'],
  ['pair', 'opp'],
  ['loot', 'own'],
  ['loot', 'opp']
];

/** One hint a card carries, from this screen's point of view. */
export interface PipEntry {
  readonly code: HintCode;
  /** The seat the fact helps. */
  readonly seat: Seat;
  /** The perspective seat's fact, or the other seat's. */
  readonly side: 'own' | 'opp';
  readonly style: PipStyle;
  readonly tone: PipTone;
  /** The tooltip line, already translated. */
  readonly text: string;
}

/** A card's pip: the top entry, and every entry in priority order. */
export interface CardPip {
  readonly top: PipEntry;
  readonly entries: readonly PipEntry[];
  /** More than one hint: the hairline dot beside the pip. */
  readonly more: boolean;
}

/** Display names of the two seats, and whether each one is "you" on this screen. */
export interface HintNames {
  readonly names: readonly [string, string];
  /** The seat this screen speaks to as "you", or null (a hot-seat device, a spectator). */
  readonly you: Seat | null;
}

/** The point of view of a screen: the view, the room mode and the seat its badges show. */
export function perspectiveOf(
  view: Pick<View, 'viewer' | 'active'>,
  mode: Mode | undefined
): Perspective {
  const seat: Seat = (view.viewer ?? view.active) === 1 ? 1 : 0;
  if (view.viewer === undefined) return { kind: 'spectator', seat };
  if (mode === 'hotseat') return { kind: 'hotseat', seat };
  return { kind: 'seat', seat };
}

/** The seat a screen speaks to as "you": only a player's own screen. */
export function youSeat(p: Perspective): Seat | null {
  return p.kind === 'seat' ? p.seat : null;
}

function other(seat: Seat): Seat {
  return seat === 0 ? 1 : 0;
}

function crest(seat: Seat): PipTone {
  return seat === 0 ? 'sun' : 'moon';
}

/** The fill and colour of a pip for `seat` from perspective `p`. */
export function pipLook(p: Perspective, seat: Seat): { style: PipStyle; tone: PipTone } {
  if (p.kind === 'spectator') return { style: 'tint', tone: crest(seat) };
  if (seat !== p.seat) return { style: 'outline', tone: crest(seat) };
  return { style: 'solid', tone: p.kind === 'hotseat' ? crest(seat) : 'gold' };
}

function rank(code: HintCode, side: 'own' | 'opp'): number {
  const i = PIP_PRIORITY.findIndex(([c, s]) => c === code && s === side);
  return i < 0 ? PIP_PRIORITY.length : i;
}

/** The tooltip line of one pip entry. */
export function pipText(code: HintCode, seat: Seat, hints: CardHints, names: HintNames): string {
  const you = names.you === seat;
  const name = names.names[seat];
  const otherName = names.names[other(seat)];
  switch (code) {
    case 'win': {
      const law = hints.effect?.wins?.via === 'token:law';
      if (you) return t(law ? 'hints.pip.win.lawYou' : 'hints.pip.win.you');
      return t(law ? 'hints.pip.win.lawNamed' : 'hints.pip.win.named', { name });
    }
    case 'pair': {
      const symbol = hints.effect?.science ? scienceName(hints.effect.science.symbol) : '';
      return you
        ? t('hints.pip.pair.you', { symbol })
        : t('hints.pip.pair.named', { name, symbol });
    }
    case 'loot': {
      const lost = (hints.effect?.military?.loot ?? []).reduce((n, l) => n + l.lost, 0);
      if (you) return t('hints.pip.loot.you', { name: otherName, count: lost });
      if (names.you === other(seat)) return t('hints.pip.loot.youLose', { name, count: lost });
      return t('hints.pip.loot.named', { name, other: otherName, count: lost });
    }
  }
}

/**
 * The pip a card shows: every code of both seats' hints whose family this
 * screen shows, from perspective `p`, in priority order; undefined when
 * there is none.
 */
export function cardPip(
  hints: readonly [CardHints, CardHints] | undefined,
  p: Perspective,
  families: ReadonlySet<HintFamily>,
  names: HintNames
): CardPip | undefined {
  if (!hints) return undefined;
  const entries: (PipEntry & { rank: number })[] = [];
  for (const seat of [0, 1] as const) {
    const ch = hints[seat];
    for (const code of ch.codes) {
      if (!families.has(CODE_FAMILY[code])) continue;
      const side = seat === p.seat ? 'own' : 'opp';
      entries.push({
        code,
        seat,
        side,
        ...pipLook(p, seat),
        text: pipText(code, seat, ch, names),
        rank: rank(code, side)
      });
    }
  }
  if (entries.length === 0) return undefined;
  entries.sort((a, b) => a.rank - b.rank);
  const clean: PipEntry[] = entries.map((e) => {
    const { rank: r, ...entry } = e;
    void r;
    return entry;
  });
  return { top: clean[0]!, entries: clean, more: clean.length > 1 };
}

/** A slot's pip, for a face-up card with hints. */
export function slotPip(
  slot: SlotView,
  p: Perspective,
  families: ReadonlySet<HintFamily>,
  names: HintNames
): CardPip | undefined {
  return slot.state === 'faceUp' ? cardPip(slot.hints, p, families, names) : undefined;
}

/** The pip's whole tooltip: one line per hint, in priority order. */
export function pipTooltip(pip: CardPip): string {
  return pip.entries.map((e) => e.text).join('\n');
}

/** The ARIA suffix for a card with a pip: what its hints say, in words. */
export function pipLabel(pip: CardPip | undefined): string {
  return pip ? joinPhrases(pip.entries.map((e) => e.text)) : '';
}

// --- Sheet lines ---------------------------------------------------------

/** One line of the card sheet's hint section. */
export interface HintLine {
  /** A stable id for tests and styling (data-hint-line). */
  readonly id: string;
  readonly text: string;
  /** A pip glyph beside the line, when the line is a pip's fact. */
  readonly code?: HintCode;
}

/** The hint lines of one seat, under its header ("For you", "For Moon"). */
export interface HintSection {
  readonly seat: Seat;
  readonly side: 'own' | 'opp';
  readonly style: PipStyle;
  readonly tone: PipTone;
  readonly title: string;
  readonly lines: readonly HintLine[];
}

function priceLine(ps: PriceShift, names: HintNames): HintLine {
  const resource = resourceName(ps.resource);
  const seat = ps.seat === 1 ? 1 : 0;
  const text =
    names.you === seat
      ? t('hints.line.price.you', { resource, from: ps.from, to: ps.to })
      : t('hints.line.price.named', {
          name: names.names[seat],
          resource,
          from: ps.from,
          to: ps.to
        });
  return { id: `price-${ps.seat}-${ps.resource}`, text };
}

/** Whether the card's catalog colour is one whose "worth now" line is shown. */
function worthCard(slug: string): boolean {
  const c = cardBySlug(slug)?.color;
  return c === 'yellow' || c === 'purple';
}

/**
 * The lines of one seat's hints for card `slug`: what building it would do
 * for that seat, as the server computed it, each under its family. With
 * `free` set the build costs nothing (the Mausoleum's rebuild), so a win
 * is never "cannot be paid for now", although the server's preview of a
 * rebuild leaves `effect.paid` false.
 */
export function seatLines(
  slug: string,
  seat: Seat,
  ch: CardHints,
  families: ReadonlySet<HintFamily>,
  names: HintNames,
  free = false
): HintLine[] {
  const out: HintLine[] = [];
  const otherSeat = other(seat);
  const e = ch.effect ?? { paid: false, coins: 0, scoreDelta: 0, oppScoreDelta: 0 };
  if (families.has('threats') && e.wins) {
    const kind = t(`hints.victory.${e.wins.kind}`);
    out.push({
      id: 'win',
      code: 'win',
      text:
        e.wins.via === 'token:law'
          ? t('hints.line.winLaw')
          : e.paid || free
            ? t('hints.line.win', { kind })
            : t('hints.line.winShort', { kind })
    });
  }
  if (families.has('science') && e.science) {
    const symbol = scienceName(e.science.symbol);
    if (e.science.kind === 'pair') {
      out.push(
        e.science.tokenOnBoard
          ? { id: 'pair', code: 'pair', text: t('hints.line.pair', { symbol }) }
          : { id: 'pair', text: t('hints.line.pairNoToken', { symbol }) }
      );
    } else {
      out.push({
        id: 'symbol',
        text: t('hints.line.newSymbol', { symbol, count: e.science.distinctAfter })
      });
    }
  }
  if (families.has('military') && e.military) {
    const m = e.military;
    const spaces = Math.abs(m.to - m.from);
    out.push({
      id: 'pawn',
      text:
        names.you === otherSeat
          ? t('hints.line.pawnYou', { count: spaces })
          : t('hints.line.pawn', { name: names.names[otherSeat], count: spaces })
    });
    const lost = m.loot.reduce((n, l) => n + l.lost, 0);
    if (m.loot.length > 0) {
      out.push({
        id: 'loot',
        code: 'loot',
        text:
          names.you === otherSeat
            ? t('hints.line.lootYou', { count: lost })
            : t('hints.line.loot', { name: names.names[otherSeat], count: lost })
      });
    }
  }
  if (families.has('economy')) {
    for (const ps of ch.priceShift) out.push(priceLine(ps, names));
    if (ch.savesInView !== undefined) {
      out.push({
        id: 'saves',
        text:
          ch.savesInView > 0
            ? t('hints.line.saves', { count: ch.savesInView })
            : t('hints.line.savesNothing')
      });
    }
  }
  if (families.has('counting') && ch.lastProducer) {
    const produce = cardBySlug(slug)?.effects.find((x) => x.kind === 'produce')?.resource;
    if (produce) {
      const resource = resourceName(produce);
      out.push({
        id: 'last',
        text:
          ch.lastProducer === 'lastCertain'
            ? t('hints.line.lastCertain', { resource })
            : t('hints.line.lastVisible', { resource })
      });
    }
  }
  if (families.has('scoring') && worthCard(slug) && (e.coins !== 0 || e.scoreDelta !== 0)) {
    out.push({ id: 'worth', text: t('hints.line.worth', { count: e.coins, vp: e.scoreDelta }) });
  }
  return out;
}

/** The header of a seat's section. */
export function sectionTitle(seat: Seat, names: HintNames): string {
  return names.you === seat
    ? t('hints.section.you')
    : t('hints.section.named', { name: names.names[seat] });
}

/**
 * The card sheet's hint sections: the perspective seat's lines first, then
 * the opponent's, each only when it has a line.
 */
export function cardSections(
  slug: string,
  hints: readonly [CardHints, CardHints] | undefined,
  p: Perspective,
  families: ReadonlySet<HintFamily>,
  names: HintNames
): HintSection[] {
  if (!hints) return [];
  const out: HintSection[] = [];
  for (const seat of [p.seat, other(p.seat)] as const) {
    const lines = seatLines(slug, seat, hints[seat], families, names);
    if (lines.length === 0) continue;
    out.push({
      seat,
      side: seat === p.seat ? 'own' : 'opp',
      ...pipLook(p, seat),
      title: sectionTitle(seat, names),
      lines
    });
  }
  return out;
}

// --- Card sheet notes beside the actions -----------------------------------

/** Whether a seat has an open Mausoleum: an unbuilt wonder with a cost and the catalog's rebuild effect. */
export function mausoleumOpen(view: Pick<View, 'players'>, seat: Seat): boolean {
  return view.players[seat].wonders.some(
    (w) =>
      !w.built &&
      !w.removed &&
      w.cost !== undefined &&
      (wonderBySlug(w.wonder)?.effects.some((e) => e.kind === 'build-from-discard') ?? false)
  );
}

/**
 * The muted notes under the discard button (discard-denies, mausoleum-feed):
 * what discarding the card keeps from the opponent, and who could rebuild it.
 */
export function discardNotes(
  slot: SlotView,
  view: Pick<View, 'players'>,
  seat: Seat,
  families: ReadonlySet<HintFamily>,
  names: HintNames
): HintLine[] {
  const out: HintLine[] = [];
  const opp = other(seat);
  const oppName = names.names[opp];
  const oppMausoleum = families.has('wonders') && mausoleumOpen(view, opp);
  const mine = slot.costs?.[seat];
  const theirs = slot.costs?.[opp];
  if (families.has('prices') && theirs && mine) {
    if (theirs.chainFree) {
      out.push({
        id: 'denies',
        text: oppMausoleum
          ? t('hints.note.deniesChainForNow', { name: oppName })
          : t('hints.note.deniesChain', { name: oppName })
      });
    } else if (theirs.affordable && !mine.affordable) {
      out.push({
        id: 'denies',
        text: oppMausoleum
          ? t('hints.note.deniesCheapForNow', { name: oppName, count: theirs.total })
          : t('hints.note.deniesCheap', { name: oppName, count: theirs.total })
      });
    }
  }
  if (oppMausoleum && !out.some((l) => l.id === 'denies')) {
    out.push({ id: 'mausoleum', text: t('hints.note.mausoleumOpp', { name: oppName }) });
  }
  if (families.has('wonders') && mausoleumOpen(view, seat)) {
    out.push({
      id: 'mausoleumOwn',
      text:
        names.you === seat
          ? t('hints.note.mausoleumYou')
          : t('hints.note.mausoleumNamed', { name: names.names[seat] })
    });
  }
  return out;
}

/**
 * deny-route: on a card that carries the opponent's pip, one note per
 * action: building it takes it from them, and a tucked card leaves the game.
 */
export function denyNotes(
  hints: readonly [CardHints, CardHints] | undefined,
  seat: Seat,
  families: ReadonlySet<HintFamily>,
  names: HintNames
): { build?: string; wonder?: string } {
  const opp = other(seat);
  const codes = hints?.[opp].codes.filter((c) => families.has(CODE_FAMILY[c])) ?? [];
  if (codes.length === 0) return {};
  const name = names.names[opp];
  return {
    build: t('hints.note.denyBuild', { name }),
    wonder: t('hints.note.denyTuck', { name })
  };
}

/**
 * short-by-discard: a short build that a discard's coins would cover at
 * today's prices (an advisory, since the discard uses the turn).
 */
export function shortByDiscard(
  shortfall: number,
  discardValue: number,
  families: ReadonlySet<HintFamily>
): string | undefined {
  if (!families.has('economy') || shortfall <= 0 || shortfall > discardValue) return undefined;
  return t('hints.note.shortByDiscard', { count: discardValue });
}

// --- The Mausoleum's pool and the destroy choice ---------------------------

/** The discard pile's hints for one seat (the Mausoleum's owner), by card slug. */
export function discardHintsFor(
  view: Pick<View, 'hints'>,
  seat: Seat
): ReadonlyMap<string, DiscardHintView> {
  const out = new Map<string, DiscardHintView>();
  for (const d of view.hints?.discard ?? []) if (d.seat === seat) out.set(d.card, d);
  return out;
}

/** The seats whose Mausoleum pool the view describes. */
export function discardSeats(view: Pick<View, 'hints'>): Seat[] {
  const seats = new Set<Seat>();
  for (const d of view.hints?.discard ?? []) seats.add(d.seat === 1 ? 1 : 0);
  return [...seats].sort();
}

/** The lines for rebuilding one discarded card, with its worth for the seat. */
export function discardLines(
  d: DiscardHintView,
  families: ReadonlySet<HintFamily>,
  names: HintNames
): HintLine[] {
  const seat: Seat = d.seat === 1 ? 1 : 0;
  const lines = seatLines(d.card, seat, d.hints, families, names, true);
  if (families.has('wonders') && d.total > 0 && !d.chainFree) {
    lines.push({ id: 'worthCoins', text: t('hints.line.worthCoins', { count: d.total }) });
  }
  return lines;
}

/** The pip of a discarded card for its Mausoleum owner, from perspective `p`. */
export function discardPip(
  d: DiscardHintView,
  p: Perspective,
  families: ReadonlySet<HintFamily>,
  names: HintNames
): CardPip | undefined {
  const seat: Seat = d.seat === 1 ? 1 : 0;
  const empty: CardHints = { ...d.hints, codes: [] };
  const pair: [CardHints, CardHints] = seat === 0 ? [d.hints, empty] : [empty, d.hints];
  return cardPip(pair, p, families, names);
}

/**
 * The pips of the discard pile (mausoleum-pool-hint): for each discarded
 * card that the server describes for a Mausoleum owner, the highest pip of
 * both seats' entries (discardPip, merged), by card slug. None with the
 * local hide's board switch off, as on the board.
 */
export function discardPips(
  view: Pick<View, 'hints'>,
  screen: HintScreen | undefined
): ReadonlyMap<string, CardPip> {
  const out = new Map<string, CardPip>();
  if (!screen || !screen.boardMarks) return out;
  const bySlug = new Map<string, DiscardHintView[]>();
  for (const d of view.hints?.discard ?? []) bySlug.set(d.card, [...(bySlug.get(d.card) ?? []), d]);
  for (const [slug, list] of bySlug) {
    const pips = list
      .map((d) => discardPip(d, screen.perspective, screen.families, screen.names))
      .filter((p): p is CardPip => p !== undefined);
    if (pips.length === 0) continue;
    const entries = pips
      .flatMap((p) => p.entries)
      .sort((a, b) => rank(a.code, a.side) - rank(b.code, b.side));
    out.set(slug, { top: entries[0]!, entries, more: entries.length > 1 });
  }
  return out;
}

/** The lines under one destroy candidate (destroy-price-drop). */
export function destroyLines(
  h: PendingCardHint,
  victim: Seat,
  families: ReadonlySet<HintFamily>,
  names: HintNames
): HintLine[] {
  if (!families.has('economy')) return [];
  const out: HintLine[] = h.priceShift.map((ps) => priceLine(ps, names));
  out.push({
    id: 'saves',
    text:
      h.savesInView > 0
        ? t('hints.line.saves', { count: h.savesInView })
        : t('hints.line.savesNothing')
  });
  if (h.onlySource) {
    const produce = cardBySlug(h.card)?.effects.find((x) => x.kind === 'produce')?.resource;
    if (produce) {
      out.push({
        id: 'only',
        text:
          names.you === victim
            ? t('hints.line.onlySourceYou', { resource: resourceName(produce) })
            : t('hints.line.onlySource', {
                name: names.names[victim],
                resource: resourceName(produce)
              })
      });
    }
  }
  return out;
}

/** The destroy choice's hints, by candidate slug. */
export function destroyHintsBySlug(pending: View['pending']): ReadonlyMap<string, PendingCardHint> {
  return new Map((pending?.cardHints ?? []).map((h) => [h.card, h]));
}

/** A card's name with its top hint, for the compact lists of the discard sheet. */
export function cardWithHint(slug: string, lines: readonly HintLine[]): string {
  return lines[0]
    ? t('hints.cardWith', { card: cardName(slug), hint: lines[0].text })
    : cardName(slug);
}

// --- This screen ------------------------------------------------------------

/** What a screen needs to draw hints: its point of view, the families it shows, and the names. */
export interface HintScreen {
  readonly perspective: Perspective;
  /** Families this screen shows (the game's settings minus the local hide). */
  readonly families: ReadonlySet<HintFamily>;
  /** Whether hint marks may show on the board (the local hide's board switch). */
  readonly boardMarks: boolean;
  readonly names: HintNames;
}

/** A screen's hints, from the view, the room's mode, the families shown and the seat names. */
export function hintScreen(
  view: Pick<View, 'viewer' | 'active'>,
  mode: Mode | undefined,
  families: ReadonlySet<HintFamily>,
  boardMarks: boolean,
  names: readonly [string, string]
): HintScreen {
  const perspective = perspectiveOf(view, mode);
  return { perspective, families, boardMarks, names: { names, you: youSeat(perspective) } };
}

/** The pips a card shows in wonder-first mode (tuck-deny): only the opponent's. */
export function denyPip(pip: CardPip | undefined): CardPip | undefined {
  if (!pip) return undefined;
  const entries = pip.entries.filter((e) => e.side === 'opp');
  return entries[0] ? { top: entries[0], entries, more: entries.length > 1 } : undefined;
}

/** The parts of a Session a screen's hints read. */
export interface HintSession {
  readonly view: View | null;
  readonly room: { readonly mode: Mode; readonly seats: RoomInfo['seats'] } | null;
  readonly hints: HintSettings | null;
}

/**
 * A room page's hint display: from its session and this device's local
 * hide. None without a view, and none once the game is over (no hint is
 * shown after the fact).
 */
export function sessionHintScreen(
  session: HintSession | undefined,
  hide: LocalHide
): HintScreen | undefined {
  const view = session?.view;
  if (!session || !view || view.result) return undefined;
  return hintScreen(
    view,
    session.room?.mode,
    visibleFamilies(session.hints, hide),
    !hide.boardMarks,
    seatNames(session.room as RoomInfo | null)
  );
}

/**
 * The Mausoleum pool as hint sections, one per discarded card that has
 * something to say, titled with the card's name: for seat's pool, in the
 * order of `cards`.
 */
export function discardSections(
  view: Pick<View, 'hints'>,
  cards: readonly string[],
  seat: Seat,
  screen: HintScreen
): HintSection[] {
  const pool = discardHintsFor(view, seat);
  const out: HintSection[] = [];
  for (const slug of cards) {
    const d = pool.get(slug);
    if (!d) continue;
    const lines = discardLines(d, screen.families, screen.names);
    if (lines.length === 0) continue;
    out.push({
      seat,
      side: seat === screen.perspective.seat ? 'own' : 'opp',
      ...pipLook(screen.perspective, seat),
      title: cardName(slug),
      lines
    });
  }
  return out;
}

/** The destroy choice's candidates as hint sections, titled with each card's name. */
export function destroySections(
  pending: View['pending'],
  cards: readonly string[],
  screen: HintScreen
): HintSection[] {
  if (!pending || pending.kind !== 'destroyCard') return [];
  const destroyer: Seat = pending.seat === 1 ? 1 : 0;
  const victim = other(destroyer);
  const byCard = destroyHintsBySlug(pending);
  const out: HintSection[] = [];
  for (const slug of cards) {
    const h = byCard.get(slug);
    if (!h) continue;
    const lines = destroyLines(h, victim, screen.families, screen.names);
    if (lines.length === 0) continue;
    out.push({
      seat: destroyer,
      side: destroyer === screen.perspective.seat ? 'own' : 'opp',
      ...pipLook(screen.perspective, destroyer),
      title: cardName(slug),
      lines
    });
  }
  return out;
}

/** The heading of a seat's Mausoleum pool. */
export function poolTitle(seat: Seat, names: HintNames): string {
  return names.you === seat
    ? t('hints.discard.poolYou')
    : t('hints.discard.poolNamed', { name: names.names[seat] });
}
