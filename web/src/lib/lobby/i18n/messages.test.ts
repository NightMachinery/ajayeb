import { afterEach, expect, it } from 'vitest';
import { i18n } from '$lib/i18n';
import { lobby as en } from './en';
import { lobby as fa } from './fa';
import { t } from './messages';
import { t as pickerT } from './picker';
afterEach(() => {
  i18n.locale = 'en';
});
it('has every Persian lobby key and uses the shared RTL and Intl interpolation', () => {
  expect(Object.keys(fa).sort()).toEqual(Object.keys(en).sort());
  expect(t('lobby.seat', { seat: 4 })).toBe('Seat 4');
  i18n.locale = 'fa';
  expect(t('lobby.seat', { seat: 4 })).toContain('۴');
  expect(t('lobby.manage', { name: 'Guest' })).toContain('\u2068Guest\u2069');
  i18n.locale = 'en';
  for (const key of ['lobby.duel', 'lobby.haftgane', 'lobby.new'] as const)
    expect(t(key)).toBe(pickerT(key));
});

it('preserves every message placeholder in Persian', () => {
  const slots = (text: string) => [...text.matchAll(/\{(\w+)\}/g)].map((m) => m[1]).sort();
  for (const key of Object.keys(en) as (keyof typeof en)[])
    expect(slots(fa[key]), key).toEqual(slots(en[key]));
});
