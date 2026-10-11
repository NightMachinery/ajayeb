import { describe, expect, it } from 'vitest';
import {
  CAPITAL_SPACE,
  TRACK_CELLS,
  TRACK_ZONES,
  cellCenter,
  zoneBoundaries,
  zoneMiddle,
  zoneSpan
} from './track-cells';

const cell = 100 / TRACK_CELLS;

describe('cellCenter', () => {
  it('puts the centre space in the middle', () => {
    expect(cellCenter(0, 0)).toBeCloseTo(50);
    expect(cellCenter(0, 1)).toBeCloseTo(50);
  });

  it('puts each capital in the middle of an end space', () => {
    expect(cellCenter(-CAPITAL_SPACE, 0)).toBeCloseTo(cell / 2);
    expect(cellCenter(CAPITAL_SPACE, 0)).toBeCloseTo(100 - cell / 2);
  });

  it('mirrors the track for a player at the Moon', () => {
    expect(cellCenter(3, 1)).toBeCloseTo(cellCenter(-3, 0));
    expect(cellCenter(6, 1)).toBeCloseTo(100 - cellCenter(6, 0));
  });

  it('spaces the cells evenly', () => {
    expect(cellCenter(4, 0) - cellCenter(3, 0)).toBeCloseTo(cell);
  });
});

describe('zoneSpan and zoneMiddle', () => {
  it('covers spaces 3 to 5 for the 5-point zone, on the right with the Sun at the left', () => {
    const zone = TRACK_ZONES[1]!;
    expect(zone.points).toBe(5);
    const span = zoneSpan(zone, 1, 0);
    expect(span.start).toBeCloseTo(cellCenter(3, 0) - cell / 2);
    expect(span.end).toBeCloseTo(cellCenter(5, 0) + cell / 2);
    expect(zoneMiddle(zone, 1, 0)).toBeCloseTo(cellCenter(4, 0));
  });

  it('keeps start before end on the left side and when mirrored', () => {
    for (const side of [-1, 1] as const) {
      for (const left of [0, 1] as const) {
        const span = zoneSpan(TRACK_ZONES[2]!, side, left);
        expect(span.start).toBeLessThan(span.end);
        expect(span.end - span.start).toBeCloseTo(3 * cell);
      }
    }
  });
});

describe('zoneBoundaries', () => {
  it('marks eight lines, symmetric about the centre', () => {
    const lines = zoneBoundaries(0);
    expect(lines).toHaveLength(8);
    for (let i = 0; i < 4; i++) {
      expect(lines[i]! + lines[7 - i]!).toBeCloseTo(100);
    }
    // The first line on the right is the edge between the centre and space 1.
    expect(lines[4]).toBeCloseTo(50 + cell / 2);
  });

  it('is the same drawing when mirrored', () => {
    const a = zoneBoundaries(0);
    const b = zoneBoundaries(1);
    a.forEach((x, i) => expect(b[i]).toBeCloseTo(x));
  });
});
