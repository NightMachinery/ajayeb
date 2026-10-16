import { afterEach, describe, expect, it } from 'vitest';
import { i18n, t } from '$lib/i18n';
import { longestWord } from './card-style';

describe('longestWord', () => {
  afterEach(() => i18n.set('en'));

  it('counts the letters of the longest word', () => {
    expect(longestWord('Great Library')).toBe(7);
    expect(longestWord('')).toBe(1);
  });

  it('does not count the direction marks of a right-to-left locale', () => {
    const english = longestWord(t('card.name.workshop'));
    i18n.set('fa');
    expect(t('card.name.workshop')).not.toBe('Workshop');
    expect(longestWord(t('card.name.workshop'))).toBe(english);
  });
});
