// The move review (the advisor hint family; the hints design, owner answer
// Q2): after a player's move the server says, to that player only, whether
// a clearly better move existed and why (View.review). The client shows it
// inside the summary of the opponent's next move, so a player learns from
// the move they just made without ever being told what to play.
//
// Display mapping only: the server chose the better move and the reason
// from structured codes; this file turns them into one short sentence and
// decides which summary the review belongs to. It computes nothing about
// the rules.

import { cardName, scienceName, wonderName } from '$lib/components/card/effect-text';
import { t } from '$lib/i18n';
import type { HistoryEntry, ReviewView } from '$lib/wire';
import type { MoveSummary, Seat } from './move-summary';

/** The words a review needs: the seats' display names. */
export interface ReviewNames {
  readonly names: readonly [string, string];
  /** The seat the review is for (the viewer). */
  readonly seat: Seat;
}

/** A review as the summary shows it: what was played, and what was better. */
export interface ReviewLines {
  /** "You discarded Walls." */
  readonly played: string;
  /** "Building Walls would have won the game.", or the quiet "no better move" line. */
  readonly verdict: string;
  /** Whether a better move was found (the verdict is a lesson, not the quiet line). */
  readonly better: boolean;
}

/** The parts of an action key: construct:<card>, discard:<card>, wonder:<wonder>:<card>. */
function parseKey(key: string): { kind: string; card?: string; wonder?: string } {
  const [kind = '', a, b] = key.split(':');
  if (kind === 'wonder') return { kind, wonder: a, card: b };
  return { kind, card: a };
}

/** "You built Baths." for the move a review is about. */
export function playedText(key: string): string {
  const k = parseKey(key);
  switch (k.kind) {
    case 'construct':
      return t('hints.review.played.build', { card: cardName(k.card ?? '') });
    case 'discard':
      return t('hints.review.played.discard', { card: cardName(k.card ?? '') });
    case 'wonder':
      return t('hints.review.played.wonder', { wonder: wonderName(k.wonder ?? '') });
    default:
      return t('hints.review.played.other');
  }
}

/** "Building Walls" for the better move, as the subject of a sentence. */
export function moveText(key: string): string {
  const k = parseKey(key);
  switch (k.kind) {
    case 'construct':
      return t('hints.review.move.build', { card: cardName(k.card ?? '') });
    case 'discard':
      return t('hints.review.move.discard', { card: cardName(k.card ?? '') });
    case 'wonder':
      return t('hints.review.move.wonder', { wonder: wonderName(k.wonder ?? '') });
    default:
      return t('hints.review.move.other');
  }
}

/** The verdict: why the better move was better, or the quiet line. */
export function verdictText(review: ReviewView, names: ReviewNames): string {
  if (!review.better) return t('hints.review.none');
  const move = moveText(review.better);
  const opp = names.names[names.seat === 0 ? 1 : 0];
  switch (review.reason) {
    case 'wins':
      return t('hints.review.wins', { move });
    case 'stopsWin': {
      const th = review.threat;
      if (th?.kind === 'military') return t('hints.review.stopsMilitary', { move, name: opp });
      const with_ = th?.wonder ? wonderName(th.wonder) : th?.card ? cardName(th.card) : undefined;
      return with_
        ? t('hints.review.stopsWinWith', { move, name: opp, card: with_ })
        : t('hints.review.stopsWin', { move, name: opp });
    }
    case 'pair':
      return t('hints.review.pair', {
        move,
        symbol: review.symbol ? scienceName(review.symbol) : ''
      });
    case 'deniesPair':
      return t('hints.review.deniesPair', {
        move,
        name: opp,
        symbol: review.symbol ? scienceName(review.symbol) : ''
      });
    case 'loot':
      return t('hints.review.loot', { move, name: opp, count: review.coins ?? 0 });
    case 'points':
    case 'none':
    default:
      return t('hints.review.points', { move, count: Math.max(1, review.gap) });
  }
}

/** The review's two lines. */
export function reviewLines(review: ReviewView, names: ReviewNames): ReviewLines {
  return {
    played: playedText(review.played),
    verdict: verdictText(review, names),
    better: !!review.better
  };
}

/**
 * The review a summary of the other side's moves carries: the viewer's own
 * review when the move it reviews is the viewer's last move before those
 * moves (the other side did not move between the two). Null otherwise: the
 * viewer has moved since (the summary is an older one), or the review is
 * of a move before an earlier summary.
 */
export function reviewForSummary(
  review: ReviewView | undefined,
  summary: MoveSummary | null,
  history: readonly HistoryEntry[]
): ReviewView | null {
  if (!review || !summary) return null;
  const first = summary.plies[0];
  if (first === undefined || review.ply >= first) return null;
  for (const e of history) {
    if (e.ply > review.ply && e.ply < first && e.seat === summary.seat) return null;
  }
  return review;
}
