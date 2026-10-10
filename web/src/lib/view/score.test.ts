import { describe, expect, it } from 'vitest';
import {
  fixtureNames,
  fixtures,
  protocolFixtureNames,
  protocolFixtures
} from '$lib/wire/fixtures.gen';
import type { ResultView, View } from '$lib/wire';
import { finalScores, outcomeOf, SCORE_CATEGORIES, scoreRows } from './score';

/** A real recorded view, by its protocol fixture name. */
function fixture(name: (typeof protocolFixtureNames)[number]): View {
  const view = protocolFixtures[protocolFixtureNames.indexOf(name)];
  if (!view) throw new Error(`no protocol fixture named ${name}`);
  return view;
}

function resultOf(view: View): ResultView {
  if (!view.result) throw new Error('the view has no result');
  return view.result;
}

describe('scoreRows', () => {
  it('lists the eight categories in sheet order, then the total', () => {
    const rows = scoreRows(resultOf(fixture('game-over-science-seat0')).scores);
    expect(rows.map((r) => r.category)).toEqual([...SCORE_CATEGORIES, 'total']);
  });

  it('copies the server numbers by seat and marks who is ahead (science win)', () => {
    const rows = scoreRows(resultOf(fixture('game-over-science-seat0')).scores);
    const byCategory = Object.fromEntries(rows.map((r) => [r.category, r]));
    expect(byCategory['green']).toEqual({ category: 'green', values: [8, 0], ahead: 0 });
    expect(byCategory['coins']).toEqual({ category: 'coins', values: [10, 2], ahead: 0 });
    expect(byCategory['blue']).toEqual({ category: 'blue', values: [0, 0], ahead: null });
    expect(byCategory['total']).toEqual({ category: 'total', values: [18, 2], ahead: 0 });
  });

  it('reads the military points of a military win', () => {
    const rows = scoreRows(resultOf(fixture('game-over-military-hidden-seat1')).scores);
    const military = rows.find((r) => r.category === 'military');
    expect(military?.values).toEqual([10, 0]);
    expect(rows.at(-1)?.values).toEqual([22, 2]);
  });

  it('marks no seat ahead on equal lines (shared victory)', () => {
    const rows = scoreRows(resultOf(fixture('game-over-shared-seat0')).scores);
    expect(rows.every((r) => r.ahead === null)).toBe(true);
  });
});

describe('outcomeOf', () => {
  it('names a civilian winner without a tie-break when the totals differ', () => {
    const outcome = outcomeOf(resultOf(fixture('game-over-civilian-seat0')));
    expect(outcome).toEqual({ kind: 'civilian', winner: 0, totals: [3, 0], tieBreak: false });
  });

  it('reads a scientific supremacy', () => {
    const outcome = outcomeOf(resultOf(fixture('game-over-science-seat0')));
    expect(outcome.kind).toBe('science');
    expect(outcome.winner).toBe(0);
    expect(outcome.tieBreak).toBe(false);
  });

  it('reads a military supremacy, the same for every viewer', () => {
    for (const name of [
      'game-over-military-hidden-seat0',
      'game-over-military-hidden-seat1',
      'game-over-military-hidden-spectator'
    ] as const) {
      const outcome = outcomeOf(resultOf(fixture(name)));
      expect(outcome.kind).toBe('military');
      expect(outcome.winner).toBe(0);
    }
  });

  it('reads a shared victory with no winner', () => {
    const outcome = outcomeOf(resultOf(fixture('game-over-shared-seat0')));
    expect(outcome).toEqual({ kind: 'shared', winner: null, totals: [3, 3], tieBreak: false });
  });

  it('flags a civilian tie the server broke (equal totals, a named winner)', () => {
    // No recorded view freezes the tie-blue scenario's end, so one real
    // game-over view is given equal totals and more blue points for seat 1,
    // as the engine's tie-blue scenario produces.
    const real = resultOf(fixture('game-over-civilian-seat0'));
    const tied: ResultView = {
      ...real,
      winner: 1,
      scores: [
        { ...real.scores[0], blue: 2, total: 5 },
        { ...real.scores[1], blue: 5, total: 5 }
      ]
    };
    expect(outcomeOf(tied)).toEqual({
      kind: 'civilian',
      winner: 1,
      totals: [5, 5],
      tieBreak: true
    });
  });
});

describe('finalScores', () => {
  it('prefers the result over the preview once the game is over', () => {
    const view = fixture('game-over-science-seat0');
    expect(finalScores(view)).toBe(view.result?.scores);
  });

  it('falls back to the running preview before the end', () => {
    const view = fixture('age-1-start-seat0');
    expect(finalScores(view)).toBe(view.scorePreview);
  });

  it('reads the recorded game-over state frame', () => {
    const frame = fixtures[fixtureNames.indexOf('state-game-over')];
    if (frame?.t !== 'state') throw new Error('state-game-over is not a state frame');
    const outcome = outcomeOf(resultOf(frame.view));
    expect(outcome.kind).toBe('military');
    expect(scoreRows(finalScores(frame.view)).at(-1)?.values).toEqual([22, 2]);
  });
});
