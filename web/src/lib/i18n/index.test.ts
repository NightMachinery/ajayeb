import { describe, expect, it } from 'vitest';
import {
  dirFor,
  formatNumber,
  i18n,
  interpolate,
  isLocale,
  pluralCategory,
  selectPlural,
  t
} from './index';

describe('t', () => {
  it('returns a plain message', () => {
    expect(t('app.tagline')).toBe('A duel of wonders for two');
  });

  it('fills string params', () => {
    expect(t('turn.player', { name: 'Roshan' })).toBe("Roshan's turn");
  });

  it('chooses the plural form from count', () => {
    expect(t('coins.count', { count: 1 })).toBe('1 coin');
    expect(t('coins.count', { count: 0 })).toBe('0 coins');
    expect(t('coins.count', { count: 7 })).toBe('7 coins');
  });

  it('prefers an exact =n form over the plural category', () => {
    expect(t('cards.left', { count: 0 })).toBe('No cards left');
    expect(t('cards.left', { count: 1 })).toBe('1 card left');
    expect(t('cards.left', { count: 14 })).toBe('14 cards left');
  });

  it('formats numbers for the locale', () => {
    expect(t('coins.count', { count: 12000 })).toBe('12,000 coins');
  });

  it('returns the key for an unknown key', () => {
    // @ts-expect-error unknown keys do not compile
    expect(t('no.such.key')).toBe('no.such.key');
  });

  it('checks params at compile time', () => {
    // @ts-expect-error a message with placeholders needs its params
    expect(t('turn.player')).toBe("{name}'s turn");
    // @ts-expect-error a message without placeholders takes no params
    expect(t('app.name', { name: 'x' })).toBe('Ajayeb');
    // @ts-expect-error count must be a number
    expect(t('coins.count', { count: '2' })).toBe('2 coins');
  });
});

describe('interpolate', () => {
  it('replaces every placeholder and repeats', () => {
    expect(interpolate('{a} and {b}, then {a}', { a: 'x', b: 2 }, 'en')).toBe('x and 2, then x');
  });

  it('leaves a placeholder without a param in place', () => {
    expect(interpolate('{a} {missing}', { a: 1 }, 'en')).toBe('1 {missing}');
  });

  it('returns the text unchanged without params', () => {
    expect(interpolate('{a}', undefined, 'en')).toBe('{a}');
  });
});

describe('plural helpers', () => {
  it('uses CLDR categories of the given locale', () => {
    expect(pluralCategory(1, 'en')).toBe('one');
    expect(pluralCategory(2, 'en')).toBe('other');
    // Arabic has six categories; checks that the locale is passed through.
    expect(pluralCategory(2, 'ar')).toBe('two');
    expect(pluralCategory(3, 'ar')).toBe('few');
  });

  it('falls back to other when a category has no form', () => {
    expect(selectPlural({ one: 'one', other: 'many' }, 3, 'ar')).toBe('many');
  });

  it('formats numbers per locale', () => {
    expect(formatNumber(1234.5, undefined, 'en')).toBe('1,234.5');
    expect(formatNumber(1234, undefined, 'fa')).toBe('۱٬۲۳۴');
  });
});

describe('locale state', () => {
  it('starts in English, left to right', () => {
    expect(i18n.locale).toBe('en');
    expect(i18n.dir).toBe('ltr');
  });

  it('knows the direction of right-to-left languages', () => {
    expect(dirFor('fa')).toBe('rtl');
    expect(dirFor('fa-IR')).toBe('rtl');
    expect(dirFor('ar_EG')).toBe('rtl');
    expect(dirFor('en-GB')).toBe('ltr');
  });

  it('accepts only locales with a table', () => {
    expect(isLocale('en')).toBe(true);
    expect(isLocale('xx')).toBe(false);
    expect(isLocale(3)).toBe(false);
  });
});
