// The current locale and its text direction, as reactive state. Reading
// i18n.locale (t() does) inside a component or effect makes it re-run when
// the locale changes.

import type { PartialLocaleTable } from './index';

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

/** A locale changes only after its messages are ready. Failed loads keep the
 * current language; a later English request cancels a pending Persian switch. */
export class LocaleState {
  locale = $state<Locale>('en');
  dir: Dir = $derived(dirFor(this.locale));
  table = $state.raw<PartialLocaleTable | undefined>(undefined);
  #request = 0;
  #loading: Promise<PartialLocaleTable> | undefined;
  #load: () => Promise<PartialLocaleTable>;

  constructor(load: () => Promise<PartialLocaleTable> = () => import('./fa').then(({ fa }) => fa)) {
    this.#load = load;
  }

  async set(locale: Locale): Promise<void> {
    const request = ++this.#request;
    if (locale === 'en' || this.table) {
      this.locale = locale;
      return;
    }
    try {
      this.#loading ??= this.#load();
      const table = await this.#loading;
      this.table = table;
      if (request === this.#request) this.locale = locale;
    } catch {
      // Stay in the working language, and permit a later retry.
      this.#loading = undefined;
    }
  }
}

export const i18n = new LocaleState();
