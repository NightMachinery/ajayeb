import { describe, expect, it } from 'vitest';
import { overlapArea, placeCoach } from './place';

const bounds = { x: 0, y: 0, width: 360, height: 740 };
const bubble = { width: 320, height: 140 };

describe('the coach mark placement', () => {
  it('goes above an element target when there is room, else below', () => {
    const above = placeCoach({
      anchor: { x: 100, y: 500, width: 60, height: 80 },
      bubble,
      bounds,
      region: false
    });
    expect(above.placement).toBe('above');
    expect(above.y + bubble.height).toBeLessThanOrEqual(500);
    expect(above.arrowX).toBeDefined();
    const below = placeCoach({
      anchor: { x: 100, y: 60, width: 60, height: 80 },
      bubble,
      bounds,
      region: false
    });
    expect(below.placement).toBe('below');
    expect(below.y).toBeGreaterThanOrEqual(140);
  });

  it('is pinned to the far edge, with no arrow, when neither side has room', () => {
    // A phone held sideways: a tall target in the middle.
    const short = { x: 0, y: 0, width: 740, height: 360 };
    const anchor = { x: 300, y: 120, width: 60, height: 200 };
    const p = placeCoach({ anchor, bubble, bounds: short, region: false });
    expect(p.placement).toBe('pinned');
    expect(p.arrowX).toBeUndefined();
    // The anchor's middle (220) is below the screen's (180): the top edge.
    expect(p.y).toBe(8);
  });

  it('never overlaps an element target when an edge keeps clear', () => {
    const short = { x: 0, y: 0, width: 740, height: 360 };
    const anchor = { x: 300, y: 150, width: 60, height: 60 };
    const p = placeCoach({ anchor, bubble, bounds: short, region: false });
    const box = { x: p.x, y: p.y, width: bubble.width, height: bubble.height };
    expect(overlapArea(anchor, box)).toBe(0);
  });

  it('covers at most 40% of a region target', () => {
    // The structure fills most of a phone: no room above or below.
    const anchor = { x: 8, y: 120, width: 344, height: 520 };
    const p = placeCoach({ anchor, bubble, bounds, region: true });
    const box = { x: p.x, y: p.y, width: bubble.width, height: bubble.height };
    expect(overlapArea(anchor, box) / (anchor.width * anchor.height)).toBeLessThanOrEqual(0.4);
  });
});
