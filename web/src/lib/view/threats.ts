// Pure mappers over the server view's military and science data: whether a
// seat is in danger of losing its capital or of a scientific victory (the
// threat glows of the UX design), and the military track's lead for the
// aria-label and the compact top-bar variant. No rules: everything here
// reads fields the view already computed (military.position, each looting
// token, science.distinct) and never recomputes what the engine owns.
//
// With the board hints (M10.HS4) the glows read the server's facts about
// each seat (PlayerView.hints): how many shields the opponent still needs
// for a capital and whether it can still gather them, whether the missing
// science symbols can still come, and the wins a seat would have if it
// were its turn now. The glow then has four levels (Glow): none; calm, a
// hairline that says "close, but it cannot happen"; soft, the warning;
// and strong, the opponent can win with one action now. Without them (the
// family off, or the local hide's board switch) the glows fall back to the
// coarse rules above them, soft or none.
//
// A warning speaks of what may still happen. Once the game is over (by any
// victory, the end of Age III included) or the condition has been met (the
// pawn in a capital, six distinct symbols) there is nothing left to warn
// about, so the danger checks answer false; the result screen says what
// happened instead.

import { t } from '$lib/i18n';
import type {
  HintFamily,
  LootingView,
  MilitaryView,
  Phase,
  PlayerHintsView,
  ScienceView,
  ThreatView,
  VictoryKind,
  View
} from '$lib/wire';
import type { HintNames } from './hints';
import { threatList } from './outcome';

export type Seat = 0 | 1;

/** The pawn is this many spaces or more from the middle before a capital is
 *  called "in danger" (the UX design's threat glow threshold). */
export const MILITARY_DANGER_MARGIN = 6;

/** A player with this many distinct science symbols or more is one pair
 *  away from a scientific victory (which needs 6). */
export const SCIENCE_DANGER_DISTINCT = 5;

/** The distinct science symbols of a scientific victory (the printed rule,
 *  docs/rules/military-science.md): from here on it has happened. */
export const SCIENCE_VICTORY_DISTINCT = 6;

/** The pawn's distance from the centre at a capital: a military victory. */
export const MILITARY_VICTORY_MARGIN = 9;

/** Whether the game in this phase is over, so no warning applies any more. */
export function gameEnded(phase: Phase | undefined): boolean {
  return phase === 'gameOver';
}

/** The seat whose capital the conflict pawn currently sits closer to
 *  (`military.position` is positive toward seat 1's capital), or null when
 *  the pawn is centred. */
export function militaryLeader(military: MilitaryView): Seat | null {
  if (military.position > 0) return 0;
  if (military.position < 0) return 1;
  return null;
}

/** The seat whose capital the pawn threatens: the leader's opponent. */
export function militaryEndangered(military: MilitaryView): Seat | null {
  const leader = militaryLeader(military);
  return leader === null ? null : leader === 0 ? 1 : 0;
}

/** How far the pawn has moved from the centre, 0 to 9. */
export function militaryMargin(military: MilitaryView): number {
  return Math.abs(military.position);
}

/** Whether this seat's capital is close enough to the pawn to glow as a
 *  threat: it is the endangered seat, and the margin has reached the
 *  danger threshold but not the capital itself. Never once the game has
 *  ended (`ended`). */
export function militaryDanger(military: MilitaryView, seat: Seat, ended = false): boolean {
  if (ended) return false;
  const margin = militaryMargin(military);
  return (
    militaryEndangered(military) === seat &&
    margin >= MILITARY_DANGER_MARGIN &&
    margin < MILITARY_VICTORY_MARGIN
  );
}

/** The next looting token the pawn would reach if it kept moving the way it
 *  is leaning: the present token on the leader's side whose space is just
 *  beyond the pawn's current position. Null when the pawn is centred or
 *  when nothing stands between it and the trailing seat's capital. */
export function nextLooting(military: MilitaryView): LootingView | null {
  const direction = Math.sign(military.position);
  if (direction === 0) return null;
  const margin = militaryMargin(military);
  let nearest: LootingView | null = null;
  for (const token of military.looting) {
    if (!token.present) continue;
    if (Math.sign(token.at) !== direction) continue;
    if (Math.abs(token.at) <= margin) continue;
    if (nearest === null || Math.abs(token.at) < Math.abs(nearest.at)) nearest = token;
  }
  return nearest;
}

/** A player with 5 distinct science symbols is one symbol from winning
 *  outright, and the UX design glows their strip as a warning. Not with 6
 *  (the victory has happened) and never once the game has ended. */
export function scienceDanger(science: ScienceView, ended = false): boolean {
  if (ended) return false;
  return science.distinct >= SCIENCE_DANGER_DISTINCT && science.distinct < SCIENCE_VICTORY_DISTINCT;
}

// --- Glow levels from the board hints (M10.HS4) ----------------------------

/** How strongly a strip warns: not at all, a calm hairline (close, but it
 *  can no longer happen), the warning, or the strong warning (a win with
 *  one action exists now). */
export type Glow = 'none' | 'calm' | 'soft' | 'strong';

/** The opponent is this many shields or fewer from a capital: "close"
 *  (the same as the pawn 6 or more spaces from the centre). */
export const CAPITAL_CLOSE = 3;

/** The capital shows the shields the opponent still needs at or under this. */
export const CAPITAL_NUMERAL = 5;

/** What the glows read from the board hints: the families this screen
 *  shows, and each seat's PlayerView.hints. */
export interface GlowHints {
  readonly families: ReadonlySet<HintFamily>;
  readonly hints: readonly [PlayerHintsView | undefined, PlayerHintsView | undefined];
}

/**
 * The glow inputs of a screen: none without a hint screen, once the game is
 * over (view.result), or with the local hide's board switch off (glow
 * upgrades are board marks).
 */
export function glowHints(
  view: Pick<View, 'players' | 'result'> | null | undefined,
  screen: { readonly families: ReadonlySet<HintFamily>; readonly boardMarks: boolean } | undefined
): GlowHints | undefined {
  if (!view || view.result || !screen || !screen.boardMarks) return undefined;
  return { families: screen.families, hints: [view.players[0].hints, view.players[1].hints] };
}

/** The wins of one kind in a seat's threat list. */
export function winsOf(hints: PlayerHintsView | undefined, kind: VictoryKind): ThreatView[] {
  return (hints?.threats ?? []).filter((th) => th.kind === kind);
}

function otherSeat(seat: Seat): Seat {
  return seat === 0 ? 1 : 0;
}

/**
 * The military glow of `seat`'s strip. Strong when the opponent could take
 * the capital with one action now (threats); with the military hints, soft
 * when the opponent is CAPITAL_CLOSE shields away or closer, calm when it
 * can no longer gather them (capital-safe); without them, the margin rule
 * of militaryDanger.
 */
export function militaryGlow(
  military: MilitaryView,
  seat: Seat,
  ended = false,
  gh?: GlowHints
): Glow {
  if (ended || militaryMargin(military) >= MILITARY_VICTORY_MARGIN) return 'none';
  if (gh?.families.has('threats') && winsOf(gh.hints[otherSeat(seat)], 'military').length > 0) {
    return 'strong';
  }
  const own = gh?.families.has('military') ? gh.hints[seat] : undefined;
  if (own) {
    if (own.toCapital > CAPITAL_CLOSE) return 'none';
    return own.capitalSafe ? 'calm' : 'soft';
  }
  return militaryDanger(military, seat) ? 'soft' : 'none';
}

/**
 * The science glow of `seat`'s strip (the seat that may win by science).
 * Strong when it could win by science with one action now (threats); at 5
 * different symbols, soft, or calm when the science hints say the missing
 * symbols can no longer come (science-reach).
 */
export function scienceGlow(science: ScienceView, seat: Seat, ended = false, gh?: GlowHints): Glow {
  if (ended || science.distinct >= SCIENCE_VICTORY_DISTINCT) return 'none';
  if (gh?.families.has('threats') && winsOf(gh.hints[seat], 'science').length > 0) {
    return 'strong';
  }
  if (science.distinct < SCIENCE_DANGER_DISTINCT) return 'none';
  const own = gh?.families.has('science') ? gh.hints[seat] : undefined;
  if (own && !own.reachable) return 'calm';
  return 'soft';
}

const GLOW_RANK: Readonly<Record<Glow, number>> = { none: 0, calm: 1, soft: 2, strong: 3 };

/** The stronger of two glows. */
export function strongerGlow(a: Glow, b: Glow): Glow {
  return GLOW_RANK[a] >= GLOW_RANK[b] ? a : b;
}

/** Whether a glow is a warning (soft or strong), not calm or none. */
export function glowWarns(g: Glow): boolean {
  return g === 'soft' || g === 'strong';
}

// --- The threats list (the city sheet's "Threats", M10.F3) -----------------

/** One line of a seat's threats: the wins of one kind, with what gives them. */
export interface ThreatLine {
  readonly kind: VictoryKind;
  readonly text: string;
}

const THREAT_KINDS = ['military', 'science', 'civilian'] as const satisfies readonly VictoryKind[];

/**
 * The full threats list of `seat` (PlayerHintsView.threats: the wins it
 * would have with one action if it were its turn now), one line per kind
 * of victory, each naming the cards or wonders that give it. Empty while
 * the threats family is hidden on this screen, when the view sends no list
 * (a pending choice) or once the game is over (no hints then).
 */
export function threatLines(
  hints: PlayerHintsView | undefined,
  seat: Seat,
  families: ReadonlySet<HintFamily>,
  names: HintNames
): ThreatLine[] {
  if (!families.has('threats')) return [];
  const you = names.you === seat;
  const name = names.names[seat];
  const out: ThreatLine[] = [];
  for (const kind of THREAT_KINDS) {
    const wins = winsOf(hints, kind);
    if (wins.length === 0) continue;
    const list = threatList(wins);
    let text: string;
    switch (kind) {
      case 'military':
        text = you
          ? t('hints.threats.militaryYou', { list })
          : t('hints.threats.military', { name, list });
        break;
      case 'science':
        text = you
          ? t('hints.threats.scienceYou', { list })
          : t('hints.threats.science', { name, list });
        break;
      case 'civilian':
        text = you
          ? t('hints.threats.civilianYou', { list })
          : t('hints.threats.civilian', { name, list });
        break;
    }
    out.push({ kind, text });
  }
  return out;
}
