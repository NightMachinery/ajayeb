import { describe, expect, it } from 'vitest';
import { protocolFixtureNames, protocolFixtures } from '$lib/wire/fixtures.gen';
import type { View } from '$lib/wire';
import { buildableWonders, wonderTally } from './wonders';

function fixture(name: (typeof protocolFixtureNames)[number]): View {
  const view = protocolFixtures[protocolFixtureNames.indexOf(name)];
  if (!view) throw new Error(`no protocol fixture named ${name}`);
  return view;
}

describe('wonderTally', () => {
  it('counts none built out of four at the start of Age I', () => {
    const view = fixture('age-1-start-seat0');
    expect(wonderTally(view.players[0])).toEqual({ built: 0, total: 4 });
    expect(wonderTally(view.players[1])).toEqual({ built: 0, total: 4 });
  });

  it('counts a built wonder (appian-economy: seat 0 has built one)', () => {
    expect(wonderTally(fixture('appian-economy-seat0').players[0])).toEqual({
      built: 1,
      total: 4
    });
  });

  it('leaves a wonder out of the total once it left the game (seventh wonder)', () => {
    const view = fixture('seventh-wonder-seat0');
    expect(wonderTally(view.players[0])).toEqual({ built: 4, total: 4 });
    // Seat 1 built three, and its fourth left the game unbuilt.
    expect(wonderTally(view.players[1])).toEqual({ built: 3, total: 3 });
  });

  it('counts what has been drafted so far during the draft', () => {
    const view = fixture('draft-round-1-seat0');
    expect(wonderTally(view.players[0])).toEqual({ built: 0, total: 1 });
    expect(wonderTally(view.players[1])).toEqual({ built: 0, total: 0 });
  });
});

describe('buildableWonders', () => {
  it("names the wonders of the view's wonder choices for the seat to move", () => {
    const view = fixture('appian-economy-seat0');
    expect(view.toMove).toBe(0);
    expect([...buildableWonders(view, 0)].sort()).toEqual([
      'colossus',
      'great-lighthouse',
      'pyramids'
    ]);
  });

  it('is empty for the seat that is not to move', () => {
    expect(buildableWonders(fixture('appian-economy-seat0'), 1).size).toBe(0);
  });

  it('is empty for a viewer whose view offers no choices', () => {
    // Seat 1's view of the same position: seat 0 is to move.
    expect(buildableWonders(fixture('appian-economy-seat1'), 0).size).toBe(0);
  });

  it('is empty when no wonder can be built (age-1-start: none affordable)', () => {
    expect(buildableWonders(fixture('age-1-start-seat0'), 0).size).toBe(0);
  });

  it('reads only wonder choices, never constructs or discards', () => {
    const view: Pick<View, 'choices' | 'toMove'> = {
      toMove: 1,
      choices: [
        { key: 'construct:baths', kind: 'construct', card: 'baths' },
        { key: 'discard:baths', kind: 'discard', card: 'baths' },
        { key: 'wonder:piraeus:baths', kind: 'wonder', card: 'baths', wonder: 'piraeus' }
      ]
    };
    expect([...buildableWonders(view, 1)]).toEqual(['piraeus']);
  });
});
