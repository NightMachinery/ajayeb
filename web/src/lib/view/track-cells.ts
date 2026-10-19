// Geometry of the military track drawn as its 19 printed spaces: the
// centre, nine spaces toward each capital, the capital being the ninth
// (docs/rules/military-science.md, "The track and its zones"). The view
// gives positions as -9 to 9, positive toward seat 1's capital; the seat at
// the bottom (or on the left) of the board has its capital at the left end,
// so for a player at the Moon the track is mirrored (as in ui/track.ts).
//
// Display only. The zone values are the printed board's decoration (the
// live score is always the view's scorePreview), and whether a looting
// token is still there is the view's own `present` flag.

export type Seat = 0 | 1;

/** The track's spaces: the centre and nine toward each capital. */
export const TRACK_CELLS = 19;

/** A capital's distance from the centre. */
export const CAPITAL_SPACE = 9;

/** One printed zone on each side of the centre: its spaces and its value. */
export interface TrackZone {
  /** The first and last space of the zone, counted from the centre. */
  first: number;
  last: number;
  /** The victory points it is worth to the player ahead at the end. */
  points: number;
}

/** The printed zones, nearest the centre first. */
export const TRACK_ZONES: readonly TrackZone[] = [
  { first: 1, last: 2, points: 2 },
  { first: 3, last: 5, points: 5 },
  { first: 6, last: 8, points: 10 }
];

/** A position as counted from the left end's side: positive toward the right. */
function fromLeft(position: number, left: Seat): number {
  return left === 0 ? position : -position;
}

/** The centre of a space, in percent of the track's width from its left end. */
export function cellCenter(position: number, left: Seat): number {
  return ((fromLeft(position, left) + CAPITAL_SPACE + 0.5) / TRACK_CELLS) * 100;
}

/** A span of the track, in percent from its left end. */
export interface Span {
  start: number;
  end: number;
}

/**
 * Where one zone lies on one side: `side` is +1 for the side toward seat
 * 1's capital (positive positions) and -1 for the side toward seat 0's.
 */
export function zoneSpan(zone: TrackZone, side: 1 | -1, left: Seat): Span {
  const a = cellCenter(side * (zone.first - 0.5), left);
  const b = cellCenter(side * (zone.last + 0.5), left);
  return { start: Math.min(a, b), end: Math.max(a, b) };
}

/** The middle of a zone on one side, in percent (where its value is printed). */
export function zoneMiddle(zone: TrackZone, side: 1 | -1, left: Seat): number {
  const span = zoneSpan(zone, side, left);
  return (span.start + span.end) / 2;
}

/**
 * The lines between zones on both sides, in percent, left to right: after
 * the centre, between the zones, and before each capital.
 */
export function zoneBoundaries(left: Seat): number[] {
  const out: number[] = [];
  for (const side of [-1, 1] as const) {
    for (const zone of TRACK_ZONES) {
      out.push(cellCenter(side * (zone.first - 0.5), left));
    }
    out.push(cellCenter(side * (CAPITAL_SPACE - 0.5), left));
  }
  return out.sort((a, b) => a - b);
}

/** A run of equal spaces drawn as one strip of niches: a zone on one side, or the centre. */
export interface CellRun extends Span {
  /** The zone's value, 0 for the centre. */
  points: number;
  /** How many spaces the run holds. */
  count: number;
}

/**
 * The 17 spaces between the capitals as seven runs, left to right: each
 * zone on each side, and the centre space alone. A run's spaces share one
 * look (the same niche art), so one element can draw them all.
 */
export function cellRuns(left: Seat): CellRun[] {
  const runs: CellRun[] = [
    { points: 0, count: 1, start: cellCenter(-0.5, left), end: cellCenter(0.5, left) }
  ];
  for (const side of [-1, 1] as const) {
    for (const zone of TRACK_ZONES) {
      runs.push({
        points: zone.points,
        count: zone.last - zone.first + 1,
        ...zoneSpan(zone, side, left)
      });
    }
  }
  return runs.sort((a, b) => a.start - b.start);
}
