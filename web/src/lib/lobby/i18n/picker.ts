// Small Home-only labels. They never join the generic shell's English table.
import { i18n } from '$lib/i18n';
import { lobby as en } from './picker-en';
import { lobby as fa } from './picker-fa';
export function t(key: keyof typeof en): string {
  return (i18n.locale === 'fa' ? fa : en)[key];
}
