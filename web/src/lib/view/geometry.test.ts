import { describe, expect, it } from 'vitest';
import { protocolFixtureNames, protocolFixtures } from '$lib/wire/fixtures.gen';
import type { View } from '$lib/wire';
import { overlappingSlots, slotPlace, slotRect, structureLayout } from './geometry';

function fixture(name: (typeof protocolFixtureNames)[number]): View {
  const view = protocolFixtures[protocolFixtureNames.indexOf(name)];
  if (!view) throw new Error(`no fixture ${name}`);
  return view as View;
}

const ages = [
  fixture('age-1-start-seat0'),
  fixture('starter-chosen-seat0'),
  fixture('looting-seat1')
];

describe('structureLayout', () => {
  it('is a 1×1 shape for an empty structure (the draft)', () => {
    expect(structureLayout(fixture('draft-round-1-seat0').structure)).toEqual({
      cols: 1,
      rows: 1,
      minCol: 1,
      minRow: 1
    });
  });

  it('starts at row 1, and at column 1 except for the narrower Age III', () => {
    expect(structureLayout(ages[0]!.structure)).toMatchObject({ minCol: 1, minRow: 1 });
    expect(structureLayout(ages[1]!.structure)).toMatchObject({ minCol: 1, minRow: 1 });
    // Age III's widest rows hold four cards, at cols 3 to 9.
    expect(structureLayout(ages[2]!.structure)).toMatchObject({ minCol: 3, minRow: 1, cols: 4 });
  });
});

describe('slotPlace', () => {
  it('puts the first Age I card half a card in from the left, on top', () => {
    const view = ages[0]!;
    const layout = structureLayout(view.structure);
    // Age I's top row is two cards at cols 5 and 7 of an 11-column pyramid.
    expect(slotPlace(view.structure[0]!, layout)).toEqual({ x: 2, y: 0, z: 1 });
    expect(slotPlace(view.structure[1]!, layout)).toEqual({ x: 3, y: 0, z: 1 });
    // The bottom row spans the whole width and is drawn over everything.
    const bottom = view.structure.filter((s) => s.row === 5);
    expect(bottom.map((s) => slotPlace(s, layout).x)).toEqual([0, 1, 2, 3, 4, 5]);
    expect(new Set(bottom.map((s) => slotPlace(s, layout).z))).toEqual(new Set([5]));
  });

  it('draws lower rows over higher ones', () => {
    for (const view of ages) {
      const layout = structureLayout(view.structure);
      for (const s of view.structure) {
        expect(slotPlace(s, layout).z).toBe(s.row);
      }
    }
  });
});

describe('slotRect', () => {
  it('offsets rows by step·h and neighbours by w + gap', () => {
    const view = ages[0]!;
    const layout = structureLayout(view.structure);
    const fit = { w: 54, h: 83, step: 0.62, gap: 4 };
    const a = slotRect(view.structure[14]!, layout, fit); // row 5, col 1
    const b = slotRect(view.structure[15]!, layout, fit); // row 5, col 3
    expect(a).toEqual({ x: 0, y: 4 * 0.62 * 83, w: 54, h: 83 });
    expect(b.x - a.x).toBe(58);
  });

  it('keeps every slot inside the structure size fitStructure reports', () => {
    for (const view of ages) {
      const layout = structureLayout(view.structure);
      const fit = { w: 60, h: 92, step: 0.5, gap: 4 };
      const width = layout.cols * fit.w + (layout.cols - 1) * fit.gap;
      const height = fit.h + (layout.rows - 1) * fit.step * fit.h;
      for (const s of view.structure) {
        const r = slotRect(s, layout, fit);
        expect(r.x).toBeGreaterThanOrEqual(0);
        expect(r.x + r.w).toBeLessThanOrEqual(width);
        expect(r.y + r.h).toBeLessThanOrEqual(height + 1e-9);
      }
    }
  });
});

describe('overlappingSlots', () => {
  it('matches the server cover lists at the start of each age', () => {
    // At the start of an age nothing has been taken, so the cards drawn over
    // a slot are exactly the ones the server says cover it.
    for (const view of [fixture('age-1-start-seat0'), fixture('starter-chosen-seat0')]) {
      for (const s of view.structure) {
        expect(overlappingSlots(s, view.structure).sort()).toEqual([...s.coveredBy].sort());
      }
    }
  });
});
