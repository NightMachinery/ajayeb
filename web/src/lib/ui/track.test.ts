import { describe, expect, it } from 'vitest';
import { trackEnds, trackMirrored, trackPercent } from './track';

describe('the military track for the side the viewer sits on', () => {
  it('draws the view as it is for the Sun at the left (and spectators)', () => {
    expect(trackEnds(0)).toEqual([0, 1]);
    expect(trackMirrored(0)).toBe(false);
    expect(trackPercent(-9, 0)).toBe(0); // the Sun's capital, left
    expect(trackPercent(0, 0)).toBe(50);
    expect(trackPercent(9, 0)).toBe(100); // the Moon's capital, right
    expect(trackPercent(3, 0)).toBeCloseTo(66.667, 2);
  });

  it('mirrors it for the Moon at the left, so the Moon capital is on its side', () => {
    expect(trackEnds(1)).toEqual([1, 0]);
    expect(trackMirrored(1)).toBe(true);
    expect(trackPercent(9, 1)).toBe(0); // the Moon's capital, now left
    expect(trackPercent(-9, 1)).toBe(100); // the Sun's capital, now right
    expect(trackPercent(0, 1)).toBe(50); // the centre stays put
  });

  it('moves the pawn and the looting tokens to the mirrored spot', () => {
    // A pawn pushed 4 toward the Moon's capital sits left of centre for the
    // Moon, as far from the middle as it is for the Sun.
    for (const at of [-6, -3, 3, 6, 4, -1]) {
      expect(trackPercent(at, 1)).toBeCloseTo(100 - trackPercent(at, 0), 10);
    }
    expect(trackPercent(4, 1)).toBeLessThan(50);
    expect(trackPercent(6, 1)).toBeCloseTo(16.667, 2); // the Moon's 5-coin token
    expect(trackPercent(-6, 1)).toBeCloseTo(83.333, 2); // the Sun's 5-coin token
  });
});
