import { describe, expect, it } from 'vitest';
import { clear, overlaps, type Box } from './toast-lane';

const box = (left: number, top: number, w: number, h: number): Box => ({
  left,
  top,
  right: left + w, // physical-ok: viewport coordinates
  bottom: top + h
});

describe('toast lanes', () => {
  it('counts a shared area as an overlap, never a shared edge', () => {
    expect(overlaps(box(0, 0, 10, 10), box(5, 5, 10, 10))).toBe(true);
    expect(overlaps(box(0, 0, 10, 10), box(10, 0, 10, 10))).toBe(false);
    expect(overlaps(box(0, 0, 10, 10), box(0, 10.4, 10, 10))).toBe(false);
  });
  it('is clear of every obstacle or not at all', () => {
    const cards = [box(0, 50, 40, 60), box(50, 50, 40, 60)];
    expect(clear(box(0, 0, 90, 48), cards)).toBe(true);
    expect(clear(box(0, 0, 90, 52), cards)).toBe(false);
  });
  it('lets an empty list sit anywhere', () => {
    expect(clear(box(0, 60, 90, 0), [box(0, 50, 40, 60)])).toBe(true);
  });
});
