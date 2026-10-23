// Haftgane's message tables and ht() (messages.svelte.ts): the English
// tables, the per-key English fallback in Persian, the lazily loaded
// Persian table, and parity with the root t() whose few helpers ht()
// repeats instead of importing.
import { afterEach, describe, expect, it } from 'vitest';
import { en, i18n, interpolate, t } from '$lib/i18n';
import { fa as rootFa } from '$lib/i18n/fa';
import {
  hgEn,
  hgInterpolate,
  ht,
  loadHgMessages,
  setHgPersianLoader,
  type HgPartialTable
} from './messages.svelte';

afterEach(async () => {
  await i18n.set('en');
  setHgPersianLoader(undefined);
});

describe('ht', () => {
  it('returns a plain message', () => {
    expect(ht('hg.rules.military.title')).toBe('Military conflicts');
    expect(ht('hg.wonder.rhodos')).toBe('The Colossus');
  });

  it('fills params and chooses the plural form from count', () => {
    expect(ht('hg.rules.diagram.age', { age: 'II' })).toBe('Age II');
    expect(ht('hg.rules.diagram.shields', { count: 1 })).toBe('1 shield');
    expect(ht('hg.rules.diagram.shields', { count: 4 })).toBe('4 shields');
  });

  it('keeps every key under its own prefix, apart from the root tables', () => {
    for (const key of Object.keys(hgEn)) {
      expect(key).toMatch(/^hg\./);
      expect(key in en, key).toBe(false);
    }
  });

  it('writes no em or en dash', () => {
    for (const message of Object.values(hgEn)) {
      const forms = typeof message === 'string' ? [message] : Object.values(message);
      for (const form of forms) expect(form).not.toMatch(/[–—]/);
    }
  });
});

describe('ht in Persian', () => {
  it('falls back to English per key, marked left to right exactly as t() marks it', async () => {
    await i18n.set('fa');
    await loadHgMessages('fa');
    // The root's Persian table translates 'rules.toc'; take it out, as an
    // untranslated key, and compare the two fallbacks of the same English.
    const table = rootFa as Record<string, unknown>;
    const saved = table['rules.toc'];
    delete table['rules.toc'];
    try {
      expect(en['rules.toc']).toBe(hgEn['hg.rules.toc']);
      expect(ht('hg.rules.toc')).toBe(t('rules.toc'));
      expect(ht('hg.rules.toc')).not.toBe('Contents');
    } finally {
      table['rules.toc'] = saved;
    }
  });

  it('chooses an English fallback plural by English rules, with Persian digits', async () => {
    await i18n.set('fa');
    expect(ht('hg.rules.diagram.shields', { count: 1 })).toContain('۱ shield');
  });

  it('interpolates as the root interpolate() does', async () => {
    const params = { name: 'Omid!', count: 12 };
    const text = '{name} has {count}, {missing}';
    expect(hgInterpolate(text, params, false)).toBe(interpolate(text, params, 'en'));
    await i18n.set('fa');
    expect(hgInterpolate(text, params, true)).toBe(interpolate(text, params, 'fa'));
  });

  it('loads its Persian table on first use and then uses it', async () => {
    const translated: HgPartialTable = { 'hg.rules.toc': 'فهرست' };
    setHgPersianLoader(async () => translated);
    await i18n.set('fa');
    await loadHgMessages('fa');
    expect(ht('hg.rules.toc')).toBe('فهرست');
    expect(ht('hg.rules.close')).toContain('Close the rules');
  });

  it('tries the Persian table again after a failed load', async () => {
    let calls = 0;
    setHgPersianLoader(async () => {
      calls += 1;
      if (calls === 1) throw new Error('offline');
      return { 'hg.rules.toc': 'فهرست' };
    });
    await i18n.set('fa');
    await loadHgMessages('fa');
    expect(ht('hg.rules.toc')).toContain('Contents');
    await loadHgMessages('fa');
    expect(ht('hg.rules.toc')).toBe('فهرست');
    expect(calls).toBe(2);
  });
});
