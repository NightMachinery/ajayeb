// Where a coach mark goes (the tutorial design, section 2.4): the tooltip
// layer's placement (ui/tooltip-place.ts) with two more rules. The bubble
// never overlaps an element target, and covers at most 40% of a region
// target (the structure, the draft offer). When neither above nor below the
// target leaves room, it is pinned to the viewport edge farther from the
// target, with no arrow; the ring alone marks the target then. Pure, in
// viewport pixels, unit-tested in place.test.ts.

import { placeTooltip, type Rect, type Size } from '$lib/ui/tooltip-place';

export type CoachPlacement = 'above' | 'below' | 'pinned';

export interface CoachPlaceInput {
  anchor: Rect;
  bubble: Size;
  bounds: Rect;
  /** The anchor is a region: the bubble may cover up to `maxCover` of it. */
  region: boolean;
  margin?: number;
  gap?: number;
  /** The largest share of a region the bubble may cover. */
  maxCover?: number;
}

export interface CoachPlace {
  placement: CoachPlacement;
  x: number;
  y: number;
  /** The arrow's centre along the bubble's edge, from its left; absent when pinned. */
  arrowX?: number;
}

/** The area two boxes share. */
export function overlapArea(a: Rect, b: Rect): number {
  const w = Math.min(a.x + a.width, b.x + b.width) - Math.max(a.x, b.x);
  const h = Math.min(a.y + a.height, b.y + b.height) - Math.max(a.y, b.y);
  return w > 0 && h > 0 ? w * h : 0;
}

/** The share of the anchor a bubble at (x, y) covers. */
function cover(anchor: Rect, bubble: Size, x: number, y: number): number {
  const area = anchor.width * anchor.height;
  if (area <= 0) return 0;
  return overlapArea(anchor, { x, y, width: bubble.width, height: bubble.height }) / area;
}

export function placeCoach(input: CoachPlaceInput): CoachPlace {
  const margin = input.margin ?? 8;
  const gap = input.gap ?? 10;
  const maxCover = input.maxCover ?? 0.4;
  const { anchor, bubble, bounds, region } = input;
  const ok = (x: number, y: number): boolean =>
    region ? cover(anchor, bubble, x, y) <= maxCover : cover(anchor, bubble, x, y) === 0;

  // Above or below, as a tooltip would go (it prefers above).
  const p = placeTooltip({ anchor, bubble, bounds, margin, gap });
  if (ok(p.bubble.x, p.bubble.y)) {
    return {
      placement: p.side === 'top' ? 'above' : 'below',
      x: p.bubble.x,
      y: p.bubble.y,
      arrowX: p.arrowX
    };
  }

  // Pinned to the edge farther from the target, else the nearer one.
  const top = bounds.y + margin;
  const bottom = bounds.y + bounds.height - margin - bubble.height;
  const anchorMid = anchor.y + anchor.height / 2;
  const boundsMid = bounds.y + bounds.height / 2;
  const far = anchorMid >= boundsMid ? top : bottom;
  const near = far === top ? bottom : top;
  const x = p.bubble.x;
  if (ok(x, far)) return { placement: 'pinned', x, y: far };
  if (ok(x, near)) return { placement: 'pinned', x, y: near };
  // Neither edge keeps clear (a target taller than the screen): the edge
  // where the bubble covers the least of it.
  const y = cover(anchor, bubble, x, near) < cover(anchor, bubble, x, far) ? near : far;
  return { placement: 'pinned', x, y };
}
