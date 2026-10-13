// The move review on the result screen (the advisor hint family, M10.F3).
// When the opponent's move ends the game, the viewer's review of their own
// last move shows in that move's summary (view/review.ts
// reviewForSummary). When the viewer's own move ends it, no summary
// follows, so the result screen shows View.review itself: this decides
// when. Display only: the review is the server's.

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
