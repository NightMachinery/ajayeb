// Persian messages for the hint section of Settings. Partial; the rest of
// the hints fall back to English. See ./index.ts.

import type { hints as en } from '../en/hints';
import type { Translation } from '../types';

export const hints = {
  'hints.family.prices': 'قیمت‌ها',
  'hints.family.economy': 'اقتصاد',
  'hints.family.science': 'علم',
  'hints.family.military': 'نظامی',
  'hints.family.threats': 'تهدیدها',
  'hints.family.tempo': 'ضرباهنگ',
  'hints.family.wonders': 'شگفتی‌ها',
  'hints.family.tokens': 'نشان‌های پیشرفت',
  'hints.family.scoring': 'امتیازشماری',
  'hints.family.counting': 'شمارش کارت',
  'hints.family.recap': 'مرور',
  'hints.family.advisor': 'مشاور',

  'hints.title': 'راهنماها',
  'hints.on': 'روشن',
  'hints.off': 'خاموش',
  'hints.saving': 'در حال ذخیره…',

  'hints.summary.all': 'همه‌ی راهنماها روشن',
  'hints.summary.none': 'بدون راهنما',
  'hints.summary.allBut': 'همه‌ی راهنماها روشن به جز {families}',
  'hints.summary.only': 'فقط {families}',
  'hints.lobby.title': 'راهنماهای این بازی',
  'hints.menu': 'راهنماها',
  'hints.sheet.title': 'راهنماهای این بازی',

  'hints.local.title': 'روی صفحه‌ی من',
  'hints.local.intro':
    'راهنماها را فقط روی همین صفحه پنهان کنید. راهنمایی که بازی خاموش کرده هرگز نمایش داده نمی‌شود.',
  'hints.local.boardMarks': 'پنهان کردن نشانه‌های راهنما روی صفحه‌ی بازی',
  'hints.local.boardMarksHint': 'برگه‌ها همچنان راهنماها را نشان می‌دهند.',

  'hints.settings.title': 'راهنماها',
  'hints.settings.tips': 'نکته‌های بار اول',
  'hints.settings.tipsHint':
    'هر نشانه‌ی راهنما را نخستین باری که روی این دستگاه ظاهر می‌شود توضیح می‌دهد.',
  'hints.settings.tipsReset': 'نمایش دوباره‌ی نکته‌ها',
  'hints.settings.tipsResetDone': 'نکته‌ها دوباره نمایش داده می‌شوند.'
} as const satisfies Translation<typeof en>;
