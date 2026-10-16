// Persian messages for the entry points of the rules reference. Partial:
// the rules text itself falls back to English. See ./index.ts.

import type { rules as en } from '../en/rules';
import type { Translation } from '../types';

export const rules = {
  'rules.title': 'قوانین',
  'rules.pageTitle': 'روش بازی',
  'rules.toc': 'فهرست',
  'rules.close': 'بستن قوانین',
  'rules.backHome': 'خانه',
  'rules.homeLink': 'قوانین',
  'rules.menu': 'قوانین',
  'rules.openPage': 'باز کردن قوانین در یک صفحه‌ی کامل'
} as const satisfies Translation<typeof en>;
