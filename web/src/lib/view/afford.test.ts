import { describe, expect, it } from 'vitest';
import { protocolFixtureNames, protocolFixtures } from '$lib/wire/fixtures.gen';
import type { CostView, View } from '$lib/wire';
import {
  AFFORD_STATES,
  affordOf,
  badgeSeat,
  badgesDimmed,
  slotAfford,
  type AffordState
} from './afford';

function fixture(name: (typeof protocolFixtureNames)[number]): View {
  const view = protocolFixtures[protocolFixtureNames.indexOf(name)];
  if (!view) throw new Error(`no fixture ${name}`);
  return view as View;
}

function slotCost(view: View, card: string, seat: number): CostView {
  const slot = view.structure.find((s) => s.card === card);
  const cost = slot?.costs?.[seat];
  if (!cost) throw new Error(`no cost for ${card}`);
  return cost;
}

describe('affordOf, from recorded views', () => {
  const reveal = fixture('age-1-reveal-seat0');
  const costs = fixture('costs-seat0');

  it('chain: a chain-free build names the card that chains it', () => {
    expect(affordOf(slotCost(costs, 'statue', 0))).toEqual({
      state: 'chain',
      coins: 0,
      shortfall: 0,
      chainFrom: 'theater'
    });
  });

  it('free: nothing to pay', () => {
    expect(affordOf(slotCost(reveal, 'quarry', 0))).toMatchObject({ state: 'free', coins: 0 });
  });

  it('own: only the printed coins', () => {
    expect(affordOf(slotCost(reveal, 'stone-pit', 0))).toMatchObject({ state: 'own', coins: 1 });
  });

  it('trade: buys units, total is what it costs', () => {
    expect(affordOf(slotCost(reveal, 'stable', 0))).toMatchObject({ state: 'trade', coins: 2 });
  });

  it('short: not affordable, with the shortfall', () => {
    expect(affordOf(slotCost(costs, 'parade-ground', 0))).toEqual({
      state: 'short',
      coins: 10,
      shortfall: 1
    });
  });

  it('agrees with the choice lists: affordable builds are choices, the rest are disabled', () => {
    // Every recorded view of the seat to move: a slot whose badge is not
    // "short" has a construct choice, and a "short" one has none.
    let checked = 0;
    for (const view of protocolFixtures as View[]) {
      if (view.phase !== 'play' || view.viewer === undefined || view.toMove !== view.viewer) {
        continue;
      }
      for (const slot of view.structure) {
        const afford = slotAfford(slot, view.viewer);
        if (!afford || !slot.card) continue;
        const buildable = view.choices.some((c) => c.key === `construct:${slot.card}`);
        expect(buildable, `${slot.card}: ${afford.state}`).toBe(afford.state !== 'short');
        checked++;
      }
    }
    expect(checked).toBeGreaterThan(10);
  });

  it('meets all five states across the recorded views', () => {
    const seen = new Set<AffordState>();
    for (const view of protocolFixtures as View[]) {
      for (const slot of view.structure) {
        for (const seat of [0, 1]) {
          const afford = slotAfford(slot, seat);
          if (afford) seen.add(afford.state);
        }
      }
    }
    expect(seen).toEqual(new Set(AFFORD_STATES));
  });
});

describe('slotAfford', () => {
  const view = fixture('age-1-start-seat0');

  it('is undefined for covered and face-down slots and for no seat', () => {
    const covered = view.structure.find((s) => !s.accessible && s.state === 'faceUp')!;
    const faceDown = view.structure.find((s) => s.state === 'faceDown')!;
    const open = view.structure.find((s) => s.accessible)!;
    expect(slotAfford(covered, 0)).toBeUndefined();
    expect(slotAfford(faceDown, 0)).toBeUndefined();
    expect(slotAfford(open, undefined)).toBeUndefined();
    expect(slotAfford(open, 0)).toBeDefined();
  });
});

describe('badge seat and dimming', () => {
  it('uses the viewer, or the active seat for a spectator', () => {
    expect(badgeSeat(fixture('age-1-start-seat1'))).toBe(1);
    const spectator = fixture('age-1-start-spectator');
    expect(spectator.viewer).toBeUndefined();
    expect(badgeSeat(spectator)).toBe(spectator.active);
  });

  it('dims when it is not that seat to move', () => {
    const view = fixture('age-1-start-seat0');
    expect(badgesDimmed(view, 0)).toBe(view.toMove !== 0);
    expect(badgesDimmed(view, 1)).toBe(view.toMove !== 1);
  });
});
