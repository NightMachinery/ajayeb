// score.ts: the game-over screen's display mappers. They re-shape the
// server's ResultView (the final score sheet, the victory kind and the
// winner) into the rows and the headline the ScoreTable and GameOver
// render. Nothing here scores anything: every number is a field the engine
// already computed, and the winner is the one the server named.

import type { Pair, ResultView, ScoreView, View } from '$lib/wire';

export type Seat = 0 | 1;

/** The score sheet's categories, in the order of the printed sheet. */
export const SCORE_CATEGORIES = [
  'blue',
  'green',
  'yellow',
  'purple',
  'wonders',
  'tokens',
  'coins',
  'military'
] as const satisfies readonly (keyof ScoreView)[];

export type ScoreCategory = (typeof SCORE_CATEGORIES)[number];

export interface ScoreRow {
  /** A score sheet category, or the total line. */
  category: ScoreCategory | 'total';
  /** The points by seat. */
  values: [number, number];
  /** The seat with more points on this line, or null when they are equal. */
  ahead: Seat | null;
}

/** The headline kinds: the three victories, plus a shared civilian one. */
export type OutcomeKind = 'civilian' | 'military' | 'science' | 'shared';

export interface Outcome {
  kind: OutcomeKind;
  /** The winner the server named; null for a shared victory. */
  winner: Seat | null;
  /** The totals by seat. */
  totals: [number, number];
  /**
   * A civilian victory with equal totals: the server broke the tie (on blue
   * points), so the screen says why the winner won.
   */
  tieBreak: boolean;
}

function ahead(a: number, b: number): Seat | null {
  if (a > b) return 0;
  if (b > a) return 1;
  return null;
}

/** The rows of the score table: one per category, then the total. */
export function scoreRows(scores: Pair<ScoreView>): ScoreRow[] {
  const [a, b] = scores;
  const rows: ScoreRow[] = SCORE_CATEGORIES.map((category) => ({
    category,
    values: [a[category], b[category]],
    ahead: ahead(a[category], b[category])
  }));
  rows.push({ category: 'total', values: [a.total, b.total], ahead: ahead(a.total, b.total) });
  return rows;
}

/** The headline of a finished game, from the server's result. */
export function outcomeOf(result: ResultView): Outcome {
  const totals: [number, number] = [result.scores[0].total, result.scores[1].total];
  const winner = result.shared || result.winner === undefined ? null : (result.winner as Seat);
  const kind: OutcomeKind = result.shared ? 'shared' : result.victory;
  return {
    kind,
    winner,
    totals,
    tieBreak: kind === 'civilian' && winner !== null && totals[0] === totals[1]
  };
}

/**
 * The score sheet to show for a view: the final one once the game is over,
 * otherwise the server's running preview.
 */
export function finalScores(view: Pick<View, 'result' | 'scorePreview'>): Pair<ScoreView> {
  return view.result?.scores ?? view.scorePreview;
}
