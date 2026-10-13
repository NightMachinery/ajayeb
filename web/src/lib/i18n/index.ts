// i18n: typed message tables, t(), and Intl helpers.
//
//   t('app.tagline')                        // 'A duel of wonders for two'
//   t('turn.player', { name: 'Roshan' })    // "Roshan's turn"
//   t('cards.left', { count: 3 })           // '3 cards left'
//
// Keys and placeholders are checked at compile time against the English
// tables, which define the message set. Other locales must provide the same
// keys (LocaleTable below), so a missing translation is a type error.
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
import { i18n, type Locale } from './locale.svelte';
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

/** The shape every other locale's table must have: every key, any wording. */
export type LocaleTable = {
  readonly [K in MessageKey]: (typeof en)[K] extends string ? string : PluralForms;
};

const tables: Record<Locale, LocaleTable> = { en };

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

/**
 * Replaces each {name} with its param, formatting numbers for the locale.
 * A placeholder without a param is left as it is, so the gap shows.
 */
export function interpolate(
  text: string,
  params: AnyParams | undefined,
  locale: string = i18n.locale
): string {
  if (!params) return text;
  return text.replace(/\{(\w+)\}/g, (placeholder, name: string) => {
    const value = params[name];
    if (value === undefined) return placeholder;
    return typeof value === 'number' ? formatNumber(value, undefined, locale) : value;
  });
}

// t() ------------------------------------------------------------------------

/**
 * The message for a key in the current locale, with its params filled in.
 * An unknown key (only possible through a cast) returns the key itself.
 */
export function t<K extends MessageKey>(key: K, ...args: ParamsArg<K>): string {
  const locale = i18n.locale;
  const params = args[0] as AnyParams | undefined;
  const message: Message | undefined = tables[locale][key] ?? en[key];
  if (message === undefined) return key;
  const text =
    typeof message === 'string'
      ? message
      : selectPlural(message, Number(params?.['count'] ?? Number.NaN), locale);
  return interpolate(text, params, locale);
}
