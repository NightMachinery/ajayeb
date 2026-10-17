import { afterEach, describe, expect, it } from 'vitest';
import { fa } from './fa';
import {
  dirFor,
  en,
  formatNumber,
  formatSigned,
  fullStop,
  joinPhrases,
  labelDetail,
  sentences,
  i18n,
  MINUS_SIGN,
  interpolate,
  isFallbackText,
  isLocale,
  pluralCategory,
  selectPlural,
  stripBidiMarks,
  t,
  type MessageKey
} from './index';

/** Runs fn with keys taken out of the Persian table, as if they were not
 *  translated yet, so the English fallback can be tested on a complete table. */
function untranslated(keys: readonly MessageKey[], fn: () => void): void {
  const table = fa as Record<string, unknown>;
  const saved = keys.map((key) => [key, table[key]] as const);
  for (const key of keys) delete table[key];
  try {
    fn();
  } finally {
    for (const [key, value] of saved) if (value !== undefined) table[key] = value;
  }
}

describe('t', () => {
  it('returns a plain message', () => {
    expect(t('app.tagline')).toBe('A duel of wonders for two');
  });

  it('fills string params', () => {
    expect(t('turn.player', { name: 'Roshan' })).toBe("Roshan's turn");
  });

  it('merges every area table (see ./en/*.ts) into one message set', () => {
    expect(t('shell.close')).toBe('Close');
    expect(t('shell.confirmArmed', { label: 'Build for 4 coins' })).toBe(
      'Confirm: Build for 4 coins'
    );
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
    expect(isLocale('fa')).toBe(true);
    expect(isLocale('xx')).toBe(false);
    expect(isLocale(3)).toBe(false);
  });
});

describe('formatSigned', () => {
  it('writes the English minus as U+2212, never a hyphen', () => {
    expect(MINUS_SIGN).toBe('\u2212');
    expect(formatSigned(-5, undefined, 'en')).toBe('\u22125');
    expect(formatSigned(-1200, undefined, 'en')).toBe('\u22121,200');
    expect(formatSigned(-5, undefined, 'en')).not.toContain('-');
  });

  it('keeps the plus and zero rules of signDisplay', () => {
    expect(formatSigned(3, { signDisplay: 'exceptZero' }, 'en')).toBe('+3');
    expect(formatSigned(0, { signDisplay: 'exceptZero' }, 'en')).toBe('0');
    expect(formatSigned(-2, { signDisplay: 'exceptZero' }, 'en')).toBe('\u22122');
    expect(formatSigned(4, { signDisplay: 'negative' }, 'en')).toBe('4');
  });

  it("leaves a locale's own minus sign and digits alone (Persian)", () => {
    const own = new Intl.NumberFormat('fa').format(-5);
    expect(formatSigned(-5, undefined, 'fa')).toBe(own);
    expect(formatSigned(-5, undefined, 'fa')).not.toContain('-');
    expect(formatSigned(-5, undefined, 'fa')).toContain('\u06f5');
  });
});

describe('the Persian table', () => {
  afterEach(() => i18n.set('en'));

  const START = '\u200e\u2066';
  const END = '\u2069\u200e';

  it('is right to left once chosen', () => {
    i18n.set('fa');
    expect(i18n.locale).toBe('fa');
    expect(i18n.dir).toBe('rtl');
  });

  it('uses its own wording, with Persian digits', () => {
    i18n.set('fa');
    expect(t('home.settings.title')).toBe('تنظیمات');
    // A string param is isolated (U+2068 ... U+2069) with its own direction.
    expect(t('turn.player', { name: 'Omid!' })).toBe('نوبت \u2068Omid!\u2069');
    expect(t('coins.count', { count: 3 })).toBe('۳ سکه');
    expect(t('cards.left', { count: 0 })).toBe('کارتی نمانده');
    expect(t('board.age', { age: 2 })).toBe('دوران ۲');
  });

  it('translates every key but the ones kept English on purpose', () => {
    // docs/i18n/persian.md lists them; a new English key needs its Persian too.
    const missing = Object.keys(en).filter((k) => !(k in fa));
    expect(missing).toEqual(['card.ageNumeral.1', 'card.ageNumeral.2', 'card.ageNumeral.3']);
  });

  it('falls back to English per key, marked left to right', () => {
    i18n.set('fa');
    untranslated(['game.card.faceDown', 'debug.hiddenTokens'], () => {
      expect(t('game.card.faceDown')).toBe(START + en['game.card.faceDown'] + END);
      // A fallback's plural form follows English rules; its digits stay Persian.
      expect(t('debug.hiddenTokens', { count: 0 })).toBe(START + '۰ hidden tokens' + END);
      expect(t('debug.hiddenTokens', { count: 1 })).toBe(START + '۱ hidden token' + END);
    });
    expect(t('game.card.faceDown')).toBe(fa['game.card.faceDown']);
  });

  it('joins words left in English the English way', () => {
    i18n.set('fa');
    untranslated(['card.resource.glass', 'card.resource.papyrus'], () => {
      const words = [t('card.resource.glass'), t('card.resource.papyrus')];
      expect(words.every(isFallbackText)).toBe(true);
    });
    expect(isFallbackText(t('card.resource.glass'))).toBe(false);
    expect(isFallbackText(t('home.settings.title'))).toBe(false);
  });

  it('ends an English fallback sentence inside its marks', () => {
    i18n.set('fa');
    untranslated(['card.effect.buildFromDiscard'], () => {
      const english = t('card.effect.buildFromDiscard');
      expect(fullStop(english)).toBe(START + en['card.effect.buildFromDiscard'] + '.' + END);
      expect(fullStop(t('home.settings.title'))).toBe('تنظیمات.');
      expect(stripBidiMarks(fullStop(english))).toBe(en['card.effect.buildFromDiscard'] + '.');
    });
  });

  it('joins bilingual sentences without moving fallback punctuation or doubling stops', () => {
    i18n.set('fa');
    untranslated(['card.effect.buildFromDiscard'], () => {
      const english = t('card.effect.buildFromDiscard');
      const ended = fullStop(english);
      expect(sentences([english, ended, ''])).toBe(ended + ' ' + ended);
      expect(joinPhrases([ended, t('home.settings.title')], 'text.join')).toBe(ended + ' تنظیمات');
      expect(fullStop(labelDetail(t('home.settings.title'), english))).toBe(
        labelDetail(t('home.settings.title'), english).slice(0, -END.length) + '.' + END
      );
      expect(stripBidiMarks(labelDetail(t('home.settings.title'), english))).toBe(
        'تنظیمات: ' + en['card.effect.buildFromDiscard']
      );
    });
  });

  it('leaves English untouched', () => {
    expect(t('game.card.faceDown')).toBe(en['game.card.faceDown']);
    expect(t('home.settings.title')).toBe('Settings');
  });

  it('only holds keys that English has', () => {
    const extra = Object.keys(fa).filter((k) => !(k in en));
    expect(extra).toEqual([]);
  });

  it('keeps every placeholder of the English message', () => {
    const names = (m: unknown): string[] =>
      [...JSON.stringify(m).matchAll(/\{(\w+)\}/g)].map((x) => x[1]!).sort();
    for (const [key, message] of Object.entries(fa)) {
      const english = en[key as keyof typeof en];
      expect(new Set(names(message)), key).toEqual(new Set(names(english)));
    }
  });
});
