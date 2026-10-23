// Lobby-only words load with the lobby or its New table sheet. Home carries
// just its picker labels; Duel's cold entry never carries this message area.
import { i18n, formatNumber } from '$lib/i18n';
import { lobby as en } from './en';
import { lobby as fa } from './fa';
type Key = keyof typeof en;
type Slots<S> = S extends `${string}{${infer P}}${infer Rest}` ? P | Slots<Rest> : never;
type Args<K extends Key> = [Slots<(typeof en)[K]>] extends [never]
  ? []
  : [params: { [P in Slots<(typeof en)[K]>]: string | number }];
export function t<K extends Key>(key: K, ...args: Args<K>): string {
  const table = i18n.locale === 'fa' ? fa : en;
  // Keep interpolation local so a lobby-only caller adds no root chunk export.
  // The parity tests cover the root t() contract for numbers and RTL isolates.
  return table[key].replace(/\{(\w+)\}/g, (placeholder, name: string) => {
    const value = (args[0] as Record<string, string | number> | undefined)?.[name];
    if (value === undefined) return placeholder;
    if (typeof value === 'number') return formatNumber(value);
    return i18n.locale === 'fa' ? '\u2068' + value + '\u2069' : value;
  });
}
