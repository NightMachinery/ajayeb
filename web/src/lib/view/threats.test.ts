import { describe, expect, it } from 'vitest';
import { protocolFixtureNames, protocolFixtures } from '$lib/wire/fixtures.gen';
import type { View } from '$lib/wire';
import {
  militaryDanger,
  militaryEndangered,
  militaryLeader,
  militaryMargin,
  nextLooting,
  scienceDanger
} from './threats';

/** A real recorded view, by the scenario/viewer name a protocol fixture
 *  carries (see server/internal/engine/testdata/protocol/). */
function fixture(name: (typeof protocolFixtureNames)[number]): View {
  const view = protocolFixtures[protocolFixtureNames.indexOf(name)];
  if (!view) throw new Error(`no protocol fixture named ${name}`);
  return view;
}

describe('militaryLeader / militaryEndangered / militaryMargin', () => {
  it('is null/null/0 when the pawn is centred (age-1-start)', () => {
    const { military } = fixture('age-1-start-seat0');
    expect(military.position).toBe(0);
    expect(militaryLeader(military)).toBeNull();
    expect(militaryEndangered(military)).toBeNull();
    expect(militaryMargin(military)).toBe(0);
  });

  it('leads toward seat 1 when the position is positive (starter-chosen, +3)', () => {
    const { military } = fixture('starter-chosen-seat0');
    expect(military.position).toBe(3);
    expect(militaryLeader(military)).toBe(0);
    expect(militaryEndangered(military)).toBe(1);
    expect(militaryMargin(military)).toBe(3);
  });
});

describe('militaryDanger', () => {
  it('is false below the threshold even for the endangered seat (position +3)', () => {
    const { military } = fixture('starter-chosen-seat0');
    expect(militaryDanger(military, 1)).toBe(false);
    expect(militaryDanger(military, 0)).toBe(false);
  });

  it('is true for the endangered seat at the threshold (looting fixture, position +6)', () => {
    const { military } = fixture('looting-seat0');
    expect(military.position).toBe(6);
    expect(militaryDanger(military, 1)).toBe(true);
    // The leader's own capital is never the one in danger.
    expect(militaryDanger(military, 0)).toBe(false);
  });
});

describe('nextLooting', () => {
  it('is null when the pawn is centred (no direction to look ahead in)', () => {
    const { military } = fixture('age-1-start-seat0');
    expect(nextLooting(military)).toBeNull();
  });

  it('finds the nearest still-present token ahead of the pawn (position +1)', () => {
    const { military } = fixture('card-destroyed-seat0');
    expect(military.position).toBe(1);
    const next = nextLooting(military);
    expect(next).not.toBeNull();
    expect(next?.at).toBe(3);
    expect(next?.value).toBe(2);
    expect(next?.present).toBe(true);
  });

  it('skips a token already claimed and finds the next one (position +3, the one at 3 is gone)', () => {
    const { military } = fixture('starter-chosen-seat0');
    const claimed = military.looting.find((l) => l.at === 3);
    expect(claimed?.present).toBe(false);
    const next = nextLooting(military);
    expect(next?.at).toBe(6);
    expect(next?.value).toBe(5);
  });

  it('is null once every token ahead of the pawn is already claimed (position +6)', () => {
    const { military } = fixture('looting-seat0');
    expect(nextLooting(military)).toBeNull();
  });
});

describe('scienceDanger', () => {
  it('is false with no symbols (age-1-start)', () => {
    expect(scienceDanger(fixture('age-1-start-seat0').players[0].science)).toBe(false);
  });

  it('is false with a single distinct symbol (library-token-taken)', () => {
    const { science } = fixture('library-token-taken-seat0').players[0];
    expect(science.distinct).toBe(1);
    expect(scienceDanger(science)).toBe(false);
  });

  it('is true once a scientific victory has been reached (game-over-science, 6 distinct)', () => {
    const { science } = fixture('game-over-science-seat0').players[0];
    expect(science.distinct).toBe(6);
    expect(scienceDanger(science)).toBe(true);
  });

  it('is true exactly at the danger threshold (5 distinct, the "one pair away" case)', () => {
    // No recorded fixture stops at exactly 5 (a real game passes straight
    // through to 6, which ends it), so this boundary value is constructed
    // directly rather than drawn from a fixture.
    expect(scienceDanger({ symbols: [], distinct: 5 })).toBe(true);
    expect(scienceDanger({ symbols: [], distinct: 4 })).toBe(false);
  });
});
