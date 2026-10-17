import { describe, expect, it } from 'vitest';
import type { TutorialState } from '$lib/wire';
import { railsKey, tutorialAllow } from './gate';

const guided: TutorialState = {
  script: 'first-game',
  status: 'guided',
  left: false,
  turn: 2,
  turns: 10,
  lessons: [],
  action: { id: 'buildFree', key: 'construct:lumber-yard', card: 'lumber-yard' }
};

describe('the tutorial rails', () => {
  it('leave only the guided key open during a guided turn', () => {
    const allow = tutorialAllow(guided, false);
    expect(allow('construct:lumber-yard')).toBe(true);
    expect(allow('discard:lumber-yard')).toBe(false);
    expect(allow('construct:stone-pit')).toBe(false);
    expect(railsKey(guided, false)).toBe('construct:lumber-yard');
  });

  it('are off while paused, outside a tutorial, in the free part and with no action', () => {
    for (const allow of [
      tutorialAllow(guided, true),
      tutorialAllow(undefined, false),
      tutorialAllow({ ...guided, status: 'free' }, false),
      tutorialAllow({ ...guided, status: 'complete' }, false),
      tutorialAllow({ ...guided, action: undefined }, false)
    ]) {
      expect(allow('construct:stone-pit')).toBe(true);
    }
  });
});
