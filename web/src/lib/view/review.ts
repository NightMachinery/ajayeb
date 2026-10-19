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
import { tokenName } from '$lib/components/choices/token-text';
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
  /** Whether a better move was found (the verdict is a lesson). */
  readonly better: boolean;
  readonly entries: readonly ReviewDecisionLines[];
}

export interface ReviewDecisionLines {
  readonly ply: number;
  readonly played: string;
  readonly verdict: string;
  readonly better: boolean;
}

/** The parts of an action key: construct:<card>, discard:<card>, wonder:<wonder>:<card>. */
function parseKey(key: string): { kind: string; card?: string; wonder?: string } {
  const [kind = '', a, b] = key.split(':');
  if (kind === 'wonder' || kind === 'draft') return { kind, wonder: a, card: b };
  return { kind, card: a };
}

/** One step of a multi-action turn, as a clause inside "Your two moves: ...". */
function stepText(key: string): string {
  const k = parseKey(key);
  switch (k.kind) {
    case 'construct':
      return t('hints.review.played.step.build', { card: cardName(k.card ?? '') });
    case 'discard':
      return t('hints.review.played.step.discard', { card: cardName(k.card ?? '') });
    case 'wonder': {
      const wonder = wonderName(k.wonder ?? '');
      return k.card
        ? t('hints.review.played.step.wonderWith', { wonder, card: cardName(k.card) })
        : t('hints.review.played.step.wonder', { wonder });
    }
    default:
      return t('hints.review.played.step.other');
  }
}

/** "You built Baths." (or "Your two moves: ...") for the move or turn a review is about. */
export function playedText(key: string, names?: ReviewNames): string {
  const parts = key.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 2) {
    return t('hints.review.played.twoMoves', {
      first: stepText(parts[0] ?? ''),
      second: stepText(parts[1] ?? '')
    });
  }
  if (parts.length > 2) {
    return t('hints.review.played.manyMoves', {
      count: parts.length,
      steps: parts.map(stepText).join(t('hints.review.played.stepSeparator'))
    });
  }
  const k = parseKey(parts[0] ?? key);
  switch (k.kind) {
    case 'token':
      return t('hints.review.played.token', { token: tokenName(k.card ?? '') });
    case 'destroy':
      return t('hints.review.played.destroy', { card: cardName(k.card ?? '') });
    case 'revive':
      return t('hints.review.played.revive', { card: cardName(k.card ?? '') });
    case 'draft':
      return t('hints.review.played.draft', { wonder: wonderName(k.wonder ?? '') });
    case 'starter':
      return t('hints.review.played.starter', { name: starterName(k.card, names) });
    case 'construct':
      return t('hints.review.played.build', { card: cardName(k.card ?? '') });
    case 'discard':
      return t('hints.review.played.discard', { card: cardName(k.card ?? '') });
    case 'wonder': {
      const wonder = wonderName(k.wonder ?? '');
      return k.card
        ? t('hints.review.played.wonderWith', { wonder, card: cardName(k.card) })
        : t('hints.review.played.wonder', { wonder });
    }
    default:
      return t('hints.review.played.other');
  }
}

/**
 * "Building Walls" for the better move, as the subject of a sentence. When
 * `playedKey` is given, same-wonder/different-card and same-card/different-wonder
 * comparisons are phrased so the distinction is explicit.
 */
export function moveText(key: string, playedKey?: string, names?: ReviewNames): string {
  const k = parseKey(key);
  const p = playedKey ? parseKey(playedKey) : undefined;
  switch (k.kind) {
    case 'token':
      return t('hints.review.move.token', { token: tokenName(k.card ?? '') });
    case 'destroy':
      return t('hints.review.move.destroy', { card: cardName(k.card ?? '') });
    case 'revive':
      return t('hints.review.move.revive', { card: cardName(k.card ?? '') });
    case 'draft':
      return t('hints.review.move.draft', { wonder: wonderName(k.wonder ?? '') });
    case 'starter':
      return t('hints.review.move.starter', { name: starterName(k.card, names) });
    case 'construct':
      return t('hints.review.move.build', { card: cardName(k.card ?? '') });
    case 'discard':
      return t('hints.review.move.discard', { card: cardName(k.card ?? '') });
    case 'wonder': {
      const wonder = wonderName(k.wonder ?? '');
      if (!k.card) return t('hints.review.move.wonder', { wonder });
      const card = cardName(k.card);
      if (p?.kind === 'wonder' && p.wonder === k.wonder && p.card && p.card !== k.card) {
        return t('hints.review.move.wonderCardInstead', { wonder, card });
      }
      if (
        p?.kind === 'wonder' &&
        p.card &&
        p.card === k.card &&
        p.wonder &&
        p.wonder !== k.wonder
      ) {
        return t('hints.review.move.otherWonderWithCard', { wonder, card });
      }
      return t('hints.review.move.wonderWith', { wonder, card });
    }
    default:
      return t('hints.review.move.other');
  }
}

function starterName(seat: string | undefined, names?: ReviewNames): string {
  return names
    ? names.names[seat === '1' ? 1 : 0]
    : t('hints.review.starter.player', { count: seat === '1' ? 2 : 1 });
}

function reasonSentence(review: ReviewView, move: string, opp: string): string {
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
      // The points are Greedy's estimate. A two-stage review adds that the
      // search also rated the move better (it does not vouch for the
      // number); a deep review has no points, only the search's verdict, so
      // it names no number.
      if (review.strength === 'deep') return t('hints.review.pointsSearch', { move });
      if (review.strength === 'twoStage') {
        return t('hints.review.pointsConfirmed', { move, count: Math.max(1, review.gap) });
      }
      return t('hints.review.points', { move, count: Math.max(1, review.gap) });
  }
}

/** The verdict: why the better move was better, or empty when no better move was found. */
export function verdictText(review: ReviewView, names: ReviewNames): string {
  if (!review.better) return '';
  const playedParts = review.played.trim().split(/\s+/).filter(Boolean);
  const betterParts = review.better.trim().split(/\s+/).filter(Boolean);
  let stepIdx = 0;
  let playedKey = playedParts[0] ?? review.played;
  let betterKey = betterParts[0] ?? review.better;
  if (playedParts.length > 1 && betterParts.length > 1) {
    stepIdx = betterParts.length - 1;
    for (let i = 0; i < Math.min(playedParts.length, betterParts.length); i++) {
      if (betterParts[i] !== playedParts[i]) {
        stepIdx = i;
        break;
      }
    }
    playedKey = playedParts[stepIdx] ?? playedKey;
    betterKey = betterParts[stepIdx] ?? betterKey;
  }
  const move = moveText(betterKey, playedKey, names);
  const opp = names.names[names.seat === 0 ? 1 : 0];
  const verdict = reasonSentence(review, move, opp);
  if (playedParts.length > 1) {
    return stepIdx === 0
      ? t('hints.review.turn.first', { verdict })
      : t('hints.review.turn.extra', { verdict });
  }
  return verdict;
}

/** The review's lines. */
export function reviewLines(review: ReviewView, names: ReviewNames): ReviewLines {
  // The fallback supports an older server or a recorded fixture. Live
  // reviews carry single-action entries, each explained independently.
  const entries = review.entries?.length
    ? review.entries.map((entry) => ({
        ply: entry.ply,
        played: playedText(entry.played, names),
        verdict: verdictText({ ...entry, entries: [], strength: review.strength }, names),
        better: !!entry.better
      }))
    : [
        {
          ply: review.ply,
          played: playedText(review.played, names),
          verdict: verdictText(review, names),
          better: !!review.better
        }
      ];
  return {
    better: entries.some((entry) => entry.better),
    entries
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
