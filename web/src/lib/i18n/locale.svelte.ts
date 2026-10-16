// The current locale and its text direction, as reactive state. Reading
// i18n.locale (t() does) inside a component or effect makes it re-run when
// the locale changes.

/** Locales with a message table. English (en) is the complete one; every
 *  other table may be partial, and t() falls back to English per key. */
export const LOCALES = ['en', 'fa'] as const;
export type Locale = (typeof LOCALES)[number];

export type Dir = 'ltr' | 'rtl';

const RTL_LANGUAGES = new Set(['ar', 'dv', 'fa', 'he', 'ku', 'ps', 'sd', 'ug', 'ur', 'yi']);

/** The text direction of a BCP 47 language tag, from its language subtag. */
export function dirFor(tag: string): Dir {
  const language = tag.toLowerCase().split(/[-_]/)[0] ?? '';
  return RTL_LANGUAGES.has(language) ? 'rtl' : 'ltr';
}

export function isLocale(value: unknown): value is Locale {
  return typeof value === 'string' && (LOCALES as readonly string[]).includes(value);
}

class LocaleState {
  locale = $state<Locale>('en');
  dir: Dir = $derived(dirFor(this.locale));

  set(locale: Locale): void {
    this.locale = locale;
  }
}

export const i18n = new LocaleState();
