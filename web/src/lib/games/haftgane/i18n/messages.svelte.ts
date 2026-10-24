// Haftgane's message tables and ht(), its t(). They live with the game, not
// in $lib/i18n, so Haftgane's words load only with Haftgane's lazy chunks
// and Home and Duel never carry them (docs/haftgane/platform-architecture.md,
// "Protocol and lazy client entry").
//
//   ht('hg.rules.military.title')                  // 'Military conflicts'
//   ht('hg.rules.diagram.shields', { count: 4 })   // '4 shields'
//
// ht() keeps the root t()'s contract ($lib/i18n/index.ts): keys and
// placeholders checked at compile time against the English tables, numbers
// through Intl, plural forms through Intl.PluralRules, a string param
// isolated in a right-to-left locale, and a key the locale's table leaves
// out falling back to English per key, marked left to right. It follows
// the root's locale (i18n.locale) and uses only the root exports that the
// root chunk already shares (t, i18n, formatNumber): any other import would
// add an export to that chunk and so change Home's and Duel's bytes. The
// few lines that repeat the root's logic are checked against it in
// messages.test.ts, so the two cannot drift apart.
//
// The English tables are the complete ones, one file per area under ./en/.
// The Persian table (./fa/) loads on first use in Persian, the way the
// root's does; until a key is translated it falls back to English.

import { formatNumber, i18n, type Locale, type Messages, type PluralForms } from '$lib/i18n';
import { city } from './en/city';
import { end } from './en/end';
import { names } from './en/names';
import { rules } from './en/rules';
import { screen } from './en/screen';
import { table } from './en/table';

/** Haftgane's English message set. */
export const hgEn = {
  ...city,
  ...end,
  ...names,
  ...rules,
  ...screen,
  ...table
} as const;

export type HgKey = keyof typeof hgEn;

/** Another locale's table: any subset of the keys. */
export type HgPartialTable = {
  readonly [K in HgKey]?: (typeof hgEn)[K] extends string ? string : PluralForms;
};

// Compile-time parameter checking, as in $lib/i18n/index.ts ------------------

type Placeholders<S> = S extends `${string}{${infer Name}}${infer Rest}`
  ? Name | Placeholders<Rest>
  : never;

type TextsOf<M> = M extends string ? M : M[keyof M];

type ParamNames<K extends HgKey> =
  Placeholders<TextsOf<(typeof hgEn)[K]>> | ((typeof hgEn)[K] extends string ? never : 'count');

export type HgParams<K extends HgKey> = {
  [P in ParamNames<K>]: P extends 'count' ? number : string | number;
};

type ParamsArg<K extends HgKey> = [ParamNames<K>] extends [never] ? [] : [params: HgParams<K>];

type AnyParams = Readonly<Record<string, string | number>>;

// The Persian table, loaded on first use -------------------------------------

class HgTables {
  fa = $state.raw<HgPartialTable | undefined>(undefined);
}

const tables = new HgTables();
let loading: Promise<void> | undefined;

let loadPersian = (): Promise<HgPartialTable> => import('./fa').then(({ fa }) => fa);

/** For tests: replaces the Persian table loader and forgets the loaded table. */
export function setHgPersianLoader(next: (() => Promise<HgPartialTable>) | undefined): void {
  loadPersian = next ?? (() => import('./fa').then(({ fa }) => fa));
  loading = undefined;
  tables.fa = undefined;
}

/** Starts loading the Persian table (once; a failed load is retried). */
export function loadHgMessages(locale: Locale): Promise<void> {
  if (locale === 'en' || tables.fa) return Promise.resolve();
  loading ??= loadPersian().then(
    (table) => {
      tables.fa = table;
    },
    () => {
      loading = undefined;
    }
  );
  return loading;
}

function tableFor(locale: Locale): HgPartialTable | undefined {
  if (locale === 'en') return hgEn;
  // Reading the table here makes a component re-render once it arrives.
  // The load writes state from a promise, never during the render.
  if (!tables.fa) void loadHgMessages(locale);
  return tables.fa;
}

// The root's helpers, repeated (see the header) -----------------------------

/** One Intl.PluralRules per locale (a plain cache, never reactive state). */
const pluralRules: Record<string, Intl.PluralRules> = {};

function selectPlural(forms: PluralForms, count: number, locale: string): string {
  const exact = forms[`=${count}`];
  if (exact !== undefined) return exact;
  const rules = (pluralRules[locale] ??= new Intl.PluralRules(locale));
  return forms[rules.select(count)] ?? forms.other;
}

// Built from code points: the Svelte compiler warns on a bidirectional
// control character in a .svelte.ts string literal, escaped or not.
/** U+2068 FIRST STRONG ISOLATE and U+2069 POP DIRECTIONAL ISOLATE. */
const FSI = String.fromCharCode(0x2068);
const PDI = String.fromCharCode(0x2069);

/** An English fallback in right-to-left text: a left-to-right isolate
 *  between two LEFT-TO-RIGHT MARKs, as the root t() writes it. */
const FALLBACK_START = String.fromCharCode(0x200e, 0x2066);
const FALLBACK_END = String.fromCharCode(0x2069, 0x200e);

/** Replaces each {name} with its param: numbers through Intl, and a string
 *  isolated with its own direction in a right-to-left locale. */
export function hgInterpolate(text: string, params: AnyParams | undefined, rtl: boolean): string {
  if (!params) return text;
  return text.replace(/\{(\w+)\}/g, (placeholder, name: string) => {
    const value = params[name];
    if (value === undefined) return placeholder;
    if (typeof value === 'number') return formatNumber(value);
    return rtl ? FSI + value + PDI : value;
  });
}

// ht() -----------------------------------------------------------------------

/**
 * A Haftgane message in the current locale, with its params filled in. A
 * key the locale's table leaves out falls back to English, as in t().
 */
export function ht<K extends HgKey>(key: K, ...args: ParamsArg<K>): string {
  const params = args[0] as AnyParams | undefined;
  const locale = i18n.locale;
  const own = (tableFor(locale) as Messages | undefined)?.[key];
  const message = own ?? (hgEn as Messages)[key];
  if (message === undefined) return key;
  const fallback = own === undefined && locale !== 'en';
  const text =
    typeof message === 'string'
      ? message
      : selectPlural(message, Number(params?.['count'] ?? Number.NaN), fallback ? 'en' : locale);
  const rtl = i18n.dir === 'rtl';
  const filled = hgInterpolate(text, params, rtl);
  return fallback && rtl ? FALLBACK_START + filled + FALLBACK_END : filled;
}
