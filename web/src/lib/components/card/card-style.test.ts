import { describe, expect, it } from 'vitest';
import { longestWord } from './card-style';

describe('longestWord', () => {
  it('counts the letters of the longest word', () => {
    expect(longestWord('Great Library')).toBe(7);
    expect(longestWord('')).toBe(1);
  });

  it('does not count the direction marks of a right-to-left locale', () => {
    // How t() returns an English fallback in a right-to-left locale.
    expect(longestWord('\u200e\u2066Workshop\u2069\u200e')).toBe(8);
    expect(longestWord('\u200e\u2066Great Library\u2069\u200e')).toBe(7);
  });
});
