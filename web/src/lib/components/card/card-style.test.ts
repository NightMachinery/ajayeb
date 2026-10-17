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
    i18n.set('fa');
    // The age numerals stay English in Persian, so t() marks them left to right.
    const numeral = t('card.ageNumeral.3');
    expect(numeral).not.toBe('III');
    expect(longestWord(numeral)).toBe(3);
  });
});
