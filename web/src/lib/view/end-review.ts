// The move review on the result screen (the advisor hint family, M10.F3).
// The result screen's final-move section carries the viewer's review
// (M11.gameover-last-move). When the opponent's move ended the game, it is
// the review of the viewer's move before it, found as for a move summary
// (view/review.ts reviewForSummary). When the viewer's own move ended it,
// it is View.review itself: this decides when. Display only: the review is
// the server's.

import type { HistoryEntry, ReviewView } from '$lib/wire';

/**
 * The review the result screen shows: View.review when the viewer's own
 * move ended the game, that is when every action after the reviewed one
 * (its token pick, a choice it opened) is the viewer's, and the last one
 * is too. Null for a forfeit (no move ended the game), for a spectator
 * (`seat` undefined) and when the other side moved after the reviewed
 * move, whose summary carries the review instead.
 */
export function endReview(
  review: ReviewView | undefined,
  seat: number | undefined,
  history: readonly HistoryEntry[],
  /** How the game ended (the result screen's outcome kind). */
  victory: string | undefined
): ReviewView | null {
  if (!review || seat === undefined || victory === 'forfeit') return null;
  const last = history[history.length - 1];
  if (!last || last.seat !== seat || last.ply < review.ply) return null;
  for (const e of history) if (e.ply > review.ply && e.seat !== seat) return null;
  return review;
}
