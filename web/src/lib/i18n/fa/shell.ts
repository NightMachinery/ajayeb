// Persian messages for the shell primitives (components/shell/*). Partial:
// the /dev/gallery demo strings stay English. See ./index.ts.

import type { shell as en } from '../en/shell';
import type { Translation } from '../types';

export const shell = {
  'shell.close': 'بستن',
  'shell.dismiss': 'بستن',
  'shell.confirmArmed': 'تأیید: {label}',
  'shell.tapAgainToConfirm': 'برای تأیید دوباره بزنید',
  'shell.offline': 'آفلاین هستید. بازی‌ها به اتصال نیاز دارند.'
} as const satisfies Translation<typeof en>;
