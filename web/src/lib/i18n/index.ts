// i18n: typed message tables, t(), and Intl helpers.
//
//   t('app.tagline')                        // 'A duel of wonders for two'
//   t('turn.player', { name: 'Roshan' })    // "Roshan's turn"
//   t('cards.left', { count: 3 })           // '3 cards left'
//
// Keys and placeholders are checked at compile time against the English
// tables, which define the message set and are the only complete ones.
// Another locale's table (./fa/ for Persian) may leave keys out
// (PartialLocaleTable below): t() falls back to English per key, and in a
// right-to-left locale marks the English text as left to right (see t()),
// so its punctuation stays where English puts it.
// Numbers go through Intl.NumberFormat and plural forms through
// Intl.PluralRules, so a locale with its own digits or plural rules needs
// no code changes.

import { board } from './en/board';
import { card } from './en/card';
import { choices } from './en/choices';
import { common } from './en/common';
import { game } from './en/game';
import { hints } from './en/hints';
import { home } from './en/home';
import { online } from './en/online';
import { rules } from './en/rules';
import { session } from './en/session';
import { shell } from './en/shell';
import { fa } from './fa';
import { dirFor, i18n, type Locale } from './locale.svelte';
import type { Message, PluralForms } from './types';

export { i18n, dirFor, isLocale, LOCALES, type Dir, type Locale } from './locale.svelte';
export type { Message, Messages, PluralForms } from './types';

/** The English message set. New areas are spread in here, one file per area
 *  under ./en/ (see ./en/shell.ts for the convention). Each area owns its own
 *  file; a component slice adds keys only to the file(s) it owns. */
export const en = {
  ...common,
  ...session,
  ...shell,
  ...card,
  ...board,
  ...home,
  ...game,
  ...choices,
  ...online,
  ...hints,
  ...rules
} as const;

export type MessageKey = keyof typeof en;

/** A complete table: every key, any wording. */
export type LocaleTable = {
  readonly [K in MessageKey]: (typeof en)[K] extends string ? string : PluralForms;
};

/** Another locale's table: any subset of the keys (t() falls back to English). */
export type PartialLocaleTable = Partial<LocaleTable>;

const tables: Record<Locale, PartialLocaleTable> = { en, fa };

// Compile-time parameter checking --------------------------------------------

type Placeholders<S> = S extends `${string}{${infer Name}}${infer Rest}`
  ? Name | Placeholders<Rest>
  : never;

type TextsOf<M> = M extends string ? M : M[keyof M];

type ParamNames<K extends MessageKey> =
  Placeholders<TextsOf<(typeof en)[K]>> | ((typeof en)[K] extends string ? never : 'count');

/** The params a message needs: count is a number, the rest strings or numbers. */
export type MessageParams<K extends MessageKey> = {
  [P in ParamNames<K>]: P extends 'count' ? number : string | number;
};

type ParamsArg<K extends MessageKey> = [ParamNames<K>] extends [never]
  ? []
  : [params: MessageParams<K>];

type AnyParams = Readonly<Record<string, string | number>>;

// Intl helpers ---------------------------------------------------------------

const numberFormats = new Map<string, Intl.NumberFormat>();
const pluralRules = new Map<string, Intl.PluralRules>();

function numberFormat(locale: string, options?: Intl.NumberFormatOptions): Intl.NumberFormat {
  const cacheKey = `${locale}|${options ? JSON.stringify(options) : ''}`;
  let format = numberFormats.get(cacheKey);
  if (!format) {
    format = new Intl.NumberFormat(locale, options);
    numberFormats.set(cacheKey, format);
  }
  return format;
}

/** Formats a number for the locale (default: the current one). */
export function formatNumber(
  value: number,
  options?: Intl.NumberFormatOptions,
  locale: string = i18n.locale
): string {
  return numberFormat(locale, options).format(value);
}

/** The typographic minus sign, U+2212. */
export const MINUS_SIGN = '\u2212';

/**
 * Formats a signed number for display: Intl's digits and sign rules, with
 * the one minus sign the game shows (U+2212) where the locale writes an
 * ASCII hyphen-minus (English does). A locale whose own minus sign is
 * something else (Persian's U+2212 with direction marks, for one) keeps it.
 * Every signed number on screen (a coin lost, a looting token, an outcome
 * delta) goes through this, so the board never mixes two minus signs.
 * Pass signDisplay for the plus: 'exceptZero' gives "+3", "−1", "0".
 */
export function formatSigned(
  value: number,
  options?: Intl.NumberFormatOptions,
  locale: string = i18n.locale
): string {
  return numberFormat(locale, options)
    .formatToParts(value)
    .map((part) => (part.type === 'minusSign' && part.value === '-' ? MINUS_SIGN : part.value))
    .join('');
}

/** The CLDR plural category of a count in the locale (default: the current one). */
export function pluralCategory(count: number, locale: string = i18n.locale): Intl.LDMLPluralRule {
  let rules = pluralRules.get(locale);
  if (!rules) {
    rules = new Intl.PluralRules(locale);
    pluralRules.set(locale, rules);
  }
  return rules.select(count);
}

/** Picks the plural form for a count: an exact '=n' form first, then the category, then 'other'. */
export function selectPlural(
  forms: PluralForms,
  count: number,
  locale: string = i18n.locale
): string {
  const exact = forms[`=${count}`];
  if (exact !== undefined) return exact;
  return forms[pluralCategory(count, locale)] ?? forms.other;
}

/** U+2068 FIRST STRONG ISOLATE and U+2069 POP DIRECTIONAL ISOLATE. */
const FSI = '\u2068';
const PDI = '\u2069';

/**
 * Replaces each {name} with its param, formatting numbers for the locale.
 * In a right-to-left locale a string param (a player's name, a card name)
 * is isolated with its own direction, so a Latin name's punctuation or
 * digits never trade places with the Persian words around it.
 * A placeholder without a param is left as it is, so the gap shows.
 */
export function interpolate(
  text: string,
  params: AnyParams | undefined,
  locale: string = i18n.locale
): string {
  if (!params) return text;
  const isolate = dirFor(locale) === 'rtl';
  return text.replace(/\{(\w+)\}/g, (placeholder, name: string) => {
    const value = params[name];
    if (value === undefined) return placeholder;
    if (typeof value === 'number') return formatNumber(value, undefined, locale);
    return isolate ? FSI + value + PDI : value;
  });
}

// t() ------------------------------------------------------------------------

/**
 * How an English fallback sits in right-to-left text: a left-to-right
 * isolate (U+2066 ... U+2069), so the punctuation and any right-to-left
 * param inside it keep their English places, between two LEFT-TO-RIGHT
 * MARKs (U+200E), so that fallbacks joined into one string (". " between
 * effect sentences) still read left to right instead of being reordered as
 * neutral runs of a right-to-left paragraph.
 */
const FALLBACK_START = '\u200e\u2066';
const FALLBACK_END = '\u2069\u200e';

/**
 * The message for a key in the current locale, with its params filled in.
 * A key the locale's table leaves out falls back to English: its plural
 * form is chosen by English rules (the forms are English ones), its numbers
 * still use the locale's digits, and in a right-to-left locale the text is
 * marked left to right (FALLBACK_START above). An unknown key (only possible through
 * a cast) returns the key itself.
 */
export function t<K extends MessageKey>(key: K, ...args: ParamsArg<K>): string {
  const locale = i18n.locale;
  const params = args[0] as AnyParams | undefined;
  const own: Message | undefined = tables[locale][key];
  const message: Message | undefined = own ?? en[key];
  if (message === undefined) return key;
  const fallback = own === undefined && locale !== 'en';
  const text =
    typeof message === 'string'
      ? message
      : selectPlural(message, Number(params?.['count'] ?? Number.NaN), fallback ? 'en' : locale);
  const filled = interpolate(text, params, locale);
  return fallback && dirFor(locale) === 'rtl' ? FALLBACK_START + filled + FALLBACK_END : filled;
}

/** Whether t() returned this text as an English fallback in a right-to-left
 *  locale (so a list of such words is joined the English way). */
export function isFallbackText(text: string): boolean {
  return text.startsWith(FALLBACK_START);
}

/** Ends a sentence with a full stop. After an English fallback the stop goes
 *  inside its left-to-right marks, so right-to-left text does not move it to
 *  the start of the English sentence. */
export function fullStop(text: string): string {
  // The fallback keeps English punctuation as well as English word order.
  const fallbackEnd = text.endsWith(FALLBACK_END);
  const core = fallbackEnd ? text.slice(0, -FALLBACK_END.length) : text;
  const pattern = fallbackEnd
    ? en['text.sentence']
    : (tables[i18n.locale]['text.sentence'] ?? en['text.sentence']);
  const result = interpolate(pattern, { text: core }, 'en');
  return fallbackEnd ? result + FALLBACK_END : result;
}

/** Join already translated text using the locale's punctuation templates.
 *  Its existing isolates stay intact; no new isolate changes a fallback's direction. */
export function joinPhrases(
  parts: readonly string[],
  separator: 'text.join' | 'text.phrases' = 'text.phrases'
): string {
  const pattern = tables[i18n.locale][separator] ?? en[separator];
  return parts
    .filter(Boolean)
    .reduce(
      (first, second) => (first ? interpolate(pattern, { first, second }, 'en') : second),
      ''
    );
}

export function labelDetail(label: string, detail: string): string {
  return detail
    ? interpolate(tables[i18n.locale]['text.detail'] ?? en['text.detail'], { label, detail }, 'en')
    : label;
}

/** End phrases once and join sentences, including RTL English fallbacks. */
export function sentences(parts: readonly string[]): string {
  const ended = parts
    .map((p) => p.trim())
    .filter(Boolean)
    .map((p) => (/[.!?…؟]$/.test(stripBidiMarks(p)) ? p : fullStop(p)));
  return joinPhrases(ended, 'text.join');
}

/** U+200E, U+200F and U+2066 to U+2069: the invisible direction marks that
 *  t() and interpolate() add in a right-to-left locale. */
const BIDI_MARKS = /[\u200e\u200f\u2066-\u2069]/g;

/** The text without its invisible direction marks, for counting letters. */
export function stripBidiMarks(text: string): string {
  return text.replace(BIDI_MARKS, '');
}
