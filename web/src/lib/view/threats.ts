// Pure mappers over the server view's military and science data: whether a
// seat is in danger of losing its capital or of a scientific victory (the
// threat glows of the UX design), and the military track's lead for the
// aria-label and the compact top-bar variant. No rules: everything here
// reads fields the view already computed (military.position, each looting
// token, science.distinct) and never recomputes what the engine owns.
//
// A warning speaks of what may still happen. Once the game is over (by any
// victory, the end of Age III included) or the condition has been met (the
// pawn in a capital, six distinct symbols) there is nothing left to warn
// about, so the danger checks answer false; the result screen says what
// happened instead.

import type { LootingView, MilitaryView, Phase, ScienceView } from '$lib/wire';

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
