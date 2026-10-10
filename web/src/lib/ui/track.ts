// Where things sit along the military track, for the side of the board the
// viewer sits on. The view gives the pawn's position and each looting
// token's space as -9 to 9, positive toward seat 1's capital. The board
// shows "you" at the bottom or on the left (UX design, decision 6), and
// your capital at the left end of the track, so for a player at the Moon
// (seat 1) the track is drawn mirrored. A spectator and the Sun see it as
// the view states it. Display only: nothing here moves the pawn.

export type Seat = 0 | 1;

/** The furthest a position goes from the centre: a capital. */
export const TRACK_END = 9;

/**
 * How far along the bar a track position sits, in percent from the left
 * end, when `left` is the seat whose capital is drawn at the left end.
 */
export function trackPercent(position: number, left: Seat): number {
  const fromLeft = left === 0 ? position : -position;
  return ((fromLeft + TRACK_END) / (2 * TRACK_END)) * 100;
}

/** The seats whose capitals are drawn at the left and the right end. */
export function trackEnds(left: Seat): readonly [Seat, Seat] {
  return left === 0 ? [0, 1] : [1, 0];
}

/** Whether the track is drawn mirrored from the view's own orientation. */
export function trackMirrored(left: Seat): boolean {
  return left === 1;
}
