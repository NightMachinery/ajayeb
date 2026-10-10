import { describe, expect, it } from 'vitest';
import { protocolFixtureNames, protocolFixtures } from '$lib/wire/fixtures.gen';
import type { View } from '$lib/wire';
import { structureLayout, slotRect } from '$lib/view/geometry';
import { fitStructure, tierFor, type StructureShape } from './fit';
import { structureGap, viewportClass } from './breakpoints.svelte';

function fixture(name: (typeof protocolFixtureNames)[number]): View {
  const i = protocolFixtureNames.indexOf(name);
  const view = protocolFixtures[i];
  if (!view) throw new Error(`no fixture ${name}`);
  return view as View;
}

// The structures of the three ages, from real recorded views (full ones:
// every slot still holds a card, or keeps its place when empty).
const AGE_VIEWS = {
  1: fixture('age-1-start-seat0'),
  2: fixture('starter-chosen-seat0'),
  3: fixture('looting-seat1')
} as const;

const shapes = {
  1: structureLayout(AGE_VIEWS[1].structure),
  2: structureLayout(AGE_VIEWS[2].structure),
  3: structureLayout(AGE_VIEWS[3].structure)
};

describe('structure shapes from recorded views', () => {
  it('Age I is 6 cards wide and 5 rows tall', () => {
    expect(AGE_VIEWS[1].age).toBe(1);
    expect(shapes[1]).toMatchObject({ cols: 6, rows: 5 });
  });
  it('Age II is 6 cards wide and 5 rows tall', () => {
    expect(AGE_VIEWS[2].age).toBe(2);
    expect(shapes[2]).toMatchObject({ cols: 6, rows: 5 });
  });
  it('Age III is 4 cards wide and 7 rows tall', () => {
    expect(AGE_VIEWS[3].age).toBe(3);
    expect(shapes[3]).toMatchObject({ cols: 4, rows: 7 });
  });
});

// The design's results table: viewport, structure box, gap, and the
// expected Age I/II and Age III card size and step.
interface Row {
  vp: string;
  W: number;
  H: number;
  gap: number;
  age12: [number, number, number | null];
  age3: [number, number, number];
}

const TABLE: Row[] = [
  { vp: '360×740', W: 344, H: 448, gap: 4, age12: [54, 83, 0.62], age3: [82, 127, 0.42] },
  { vp: '390×844', W: 374, H: 471, gap: 4, age12: [59, 91, 0.62], age3: [87, 134, 0.42] },
  { vp: '430×932', W: 414, H: 547, gap: 4, age12: [66, 101, 0.62], age3: [100, 155, 0.42] },
  { vp: '844×390', W: 450, H: 325, gap: 4, age12: [72, 111, 0.48], age3: [60, 92, 0.42] },
  { vp: '740×360', W: 424, H: 316, gap: 4, age12: [67, 104, 0.5], age3: [58, 90, 0.42] },
  { vp: '768×1024', W: 736, H: 528, gap: 6, age12: [117, 181, 0.48], age3: [97, 150, 0.42] },
  { vp: '1280×800', W: 648, H: 616, gap: 8, age12: [101, 156, 0.62], age3: [113, 175, 0.42] },
  { vp: '1920×1080', W: 1112, H: 896, gap: 8, age12: [132, 204, null], age3: [132, 204, 0.5] }
];

function expectNear(actual: number, expected: number, tolerance: number, what: string) {
  expect(Math.abs(actual - expected), `${what}: ${actual} vs ${expected}`).toBeLessThanOrEqual(
    tolerance
  );
}

function expectFits(W: number, H: number, shape: StructureShape, gap: number) {
  const fit = fitStructure(W, H, shape, { gap });
  expect(fit.width).toBeLessThanOrEqual(W);
  expect(fit.height).toBeLessThanOrEqual(H + 1e-9);
  return fit;
}

describe('fitStructure reproduces the design table within 2 px', () => {
  for (const row of TABLE) {
    for (const age of [1, 2, 3] as const) {
      it(`${row.vp}, Age ${age}`, () => {
        const fit = expectFits(row.W, row.H, shapes[age], row.gap);
        const [w, h, step] = age === 3 ? row.age3 : row.age12;
        expectNear(fit.w, w, 2, 'w');
        expectNear(fit.h, h, 2, 'h');
        if (step !== null) expectNear(fit.step, step, 0.015, 'step');
        expect(fit.step).toBeGreaterThanOrEqual(0.42);
        expect(fit.step).toBeLessThanOrEqual(0.62);
      });
    }
  }
});

describe('takeable cards on a 360×740 phone', () => {
  // The phone's structure box from the design (344×448), in every age, from
  // the recorded views' own slots: every accessible card is at least 44×44
  // CSS px and lies fully inside the box.
  for (const age of [1, 2, 3] as const) {
    it(`Age ${age}: every takeable card is at least 44×44 and inside the box`, () => {
      const view = AGE_VIEWS[age];
      const layout = shapes[age];
      const fit = expectFits(344, 448, layout, 4);
      const takeable = view.structure.filter((s) => s.accessible && s.state === 'faceUp');
      expect(takeable.length).toBeGreaterThan(0);
      for (const slot of takeable) {
        const rect = slotRect(slot, layout, fit);
        expect(rect.w).toBeGreaterThanOrEqual(44);
        expect(rect.h).toBeGreaterThanOrEqual(44);
        expect(rect.x + rect.w).toBeLessThanOrEqual(344);
        expect(rect.y + rect.h).toBeLessThanOrEqual(448);
      }
    });
  }

  it('stays at least 47 px wide down to a 320 px screen', () => {
    for (const age of [1, 2, 3] as const) {
      const fit = expectFits(320 - 16, 448, shapes[age], 4);
      expect(fit.w).toBeGreaterThanOrEqual(47);
    }
  });
});

describe('fitStructure rules', () => {
  const ageI = { cols: 6, rows: 5 };

  it('raises the step to stepMax when the width binds and the height has room', () => {
    const fit = fitStructure(344, 1000, ageI);
    expect(fit.bound).toBe('width');
    expect(fit.step).toBe(0.62);
  });

  it('lowers the step before shrinking the card when the height binds', () => {
    const roomy = fitStructure(344, 448, ageI);
    const tight = fitStructure(344, 260, ageI);
    expect(tight.w).toBe(roomy.w);
    expect(tight.step).toBeLessThan(roomy.step);
    expect(tight.height).toBeLessThanOrEqual(260);
  });

  it('shrinks the card at stepMin once the height cannot give more', () => {
    const fit = fitStructure(344, 150, ageI);
    expect(fit.bound).toBe('height');
    expect(fit.step).toBe(0.42);
    expect(fit.w).toBeLessThan(54);
    expect(fit.height).toBeLessThanOrEqual(150);
  });

  it('caps the card width at wMax and uses the preferred step', () => {
    const fit = fitStructure(2000, 2000, ageI);
    expect(fit.bound).toBe('cap');
    expect(fit.w).toBe(132);
    expect(fit.step).toBe(0.5);
  });

  it('never returns a step outside [stepMin, stepMax]', () => {
    for (let W = 200; W <= 1400; W += 37) {
      for (let H = 120; H <= 1100; H += 41) {
        for (const shape of [shapes[1], shapes[3]]) {
          const fit = fitStructure(W, H, shape);
          expect(fit.step).toBeGreaterThanOrEqual(0.42);
          expect(fit.step).toBeLessThanOrEqual(0.62);
          expect(fit.width).toBeLessThanOrEqual(W);
          expect(fit.height).toBeLessThanOrEqual(H + 1e-9);
        }
      }
    }
  });

  it('returns a zero size for a box with no room', () => {
    expect(fitStructure(0, 0, ageI)).toMatchObject({ w: 0, h: 0, width: 0, height: 0 });
  });

  it('picks the tier from the card width', () => {
    expect(tierFor(54)).toBe('mini');
    expect(tierFor(71)).toBe('mini');
    expect(tierFor(72)).toBe('medium');
    expect(tierFor(139)).toBe('medium');
    expect(tierFor(140)).toBe('full');
    expect(fitStructure(344, 448, shapes[1]).tier).toBe('mini');
    expect(fitStructure(344, 448, shapes[3]).tier).toBe('medium');
  });
});

describe('viewport classes', () => {
  it('maps the size queries to a class and a gap', () => {
    const none = { phoneLandscape: false, wide: false, tablet: false };
    expect(viewportClass(none)).toBe('phone-portrait');
    expect(viewportClass({ ...none, tablet: true })).toBe('tablet');
    expect(viewportClass({ ...none, tablet: true, wide: true })).toBe('desktop');
    expect(viewportClass({ phoneLandscape: true, wide: true, tablet: true })).toBe(
      'phone-landscape'
    );
    expect(structureGap('phone-portrait')).toBe(4);
    expect(structureGap('phone-landscape')).toBe(4);
    expect(structureGap('tablet')).toBe(6);
    expect(structureGap('desktop')).toBe(8);
  });
});
