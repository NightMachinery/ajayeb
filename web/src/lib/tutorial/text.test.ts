import { describe, expect, it } from 'vitest';
import type { TutorialState } from '$lib/wire';
import { coachStep, CLOSED_UI } from './steps';
import { coachText, lockedReason } from './text';

const guided = (action: TutorialState['action'], lessons: TutorialState['lessons'] = []) =>
  ({
    script: 'first-game',
    status: 'guided',
    left: false,
    turn: 3,
    turns: 10,
    lessons,
    action
  }) satisfies TutorialState;

function step(t: TutorialState, sheet: string | null = null) {
  const s = coachStep({
    tutorial: t,
    code: 'R',
    ply: 1,
    paused: false,
    acked: () => false,
    ui: { ...CLOSED_UI, sheet: sheet ? { card: sheet, step: 'card' } : null }
  });
  if (!s) throw new Error('no step');
  return s;
}

describe('the coach texts', () => {
  it('word the chrome, the tap step and the lesson with its names', () => {
    const t = guided({ id: 'buildFree', key: 'construct:lumber-yard', card: 'lumber-yard' });
    const tap = coachText(step(t), 'Apprentice');
    expect(tap.title).toBe('Tutorial');
    expect(tap.progress).toBe('Lesson 3 of 10');
    expect(tap.text).toBe('Tap Lumber Yard.');
    const act = coachText(step(t, 'lumber-yard'), 'Apprentice');
    expect(act.text).toContain('Lumber Yard costs nothing');
    expect(act.text).toContain('wood');
  });

  it('name the resource a trade buys and the bot', () => {
    const t = guided({ id: 'trade', key: 'construct:baths', card: 'baths' });
    const text = coachText(step(t, 'baths'), 'Apprentice').text;
    expect(text).toContain('stone');
    expect(text).toContain('Apprentice');
  });

  it('give an unknown lesson no text, and no progress outside the guided part', () => {
    const t = { ...guided(undefined, [{ id: 'someday' }]), status: 'free' as const, turn: 0 };
    const w = coachText(step(t), 'Apprentice');
    expect(w.text).toBe('');
    expect(w.progress).toBeNull();
  });

  it('give the locked buttons their reason', () => {
    expect(lockedReason(guided({ id: 'trade', key: 'construct:baths', card: 'baths' }))).toBe(
      'Tutorial: this lesson builds Baths.'
    );
    expect(
      lockedReason(guided({ id: 'discard', key: 'discard:stone-pit', card: 'stone-pit' }))
    ).toBe('Tutorial: this lesson discards Stone Pit.');
    expect(
      lockedReason(
        guided({
          id: 'buildWonder',
          key: 'wonder:great-lighthouse:glassworks',
          card: 'glassworks',
          wonder: 'great-lighthouse'
        })
      )
    ).toBe('Tutorial: this lesson builds Great Lighthouse with Glassworks.');
    expect(lockedReason(guided({ id: 'draftPick', key: 'draft:piraeus', wonder: 'piraeus' }))).toBe(
      'Tutorial: this lesson takes Piraeus.'
    );
  });
});
