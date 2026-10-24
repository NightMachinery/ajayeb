// Persian messages for the table session (../en/table.ts): refused requests,
// this tab's seat, and the development screen. See ./index.ts.

import type { Translation } from '$lib/i18n/types';
import type { table as en } from '../en/table';

export const table = {
  'hg.table.reject.decision-closed': 'این تصمیم دیگر بسته شده است.',
  'hg.table.reject.illegal': 'این انتخاب اکنون برای شما باز نیست.',
  'hg.table.reject.stale': 'پیش از رسیدن درخواست شما، بازی جلو رفت. دوباره انتخاب کنید.',
  'hg.table.reject.bad-request': 'سرور نتوانست از این درخواست استفاده کند.',
  'hg.table.reject.not-admin': 'این کار فقط از مدیران این میز برمی‌آید.',
  'hg.table.reject.forbidden': 'اکنون نمی‌توانید این کار را در این میز انجام دهید.',
  'hg.table.reject.rate': 'درخواست‌ها بیش از اندازه است. کمی صبر کنید و دوباره تلاش کنید.',
  'hg.table.reject.storage': 'سرور نتوانست این را ذخیره کند. دوباره تلاش کنید.',
  'hg.table.reject.timeout': 'پاسخی نیامد. دوباره تلاش کنید.',
  'hg.table.reject.unknown': 'این درخواست پذیرفته نشد.',

  'hg.table.seat.watching': 'دیگر جایگاهی ندارید. این میز را تماشا می‌کنید.',
  'hg.table.seat.seated': 'اکنون در جایگاه {seat} بازی می‌کنید.',

  'hg.table.dev.lobby': 'لابی {code}',
  'hg.table.dev.table': 'میز {code}',
  'hg.table.dev.seats': 'جایگاه‌ها',
  'hg.table.dev.seat': 'جایگاه {seat}',
  'hg.table.dev.open': 'خالی',
  'hg.table.dev.reserved': 'رزروشده',
  'hg.table.dev.bot': 'ربات ({level})',
  'hg.table.dev.you': 'شما',
  'hg.table.dev.away': 'غایب',
  'hg.table.dev.admin': 'مدیر',
  'hg.table.dev.temporary': 'مدیر موقت',
  'hg.table.dev.start': 'شروع، با پر کردن جایگاه‌های خالی با ربات',
  'hg.table.dev.point': 'تصمیم {id}',
  'hg.table.dev.decide': 'انتخاب شما',
  'hg.table.dev.waiting': 'در انتظار جایگاه‌های دیگر',
  'hg.table.dev.watching': 'شما تماشا می‌کنید.',
  'hg.table.dev.picked': 'انتخاب شما: {key}',
  'hg.table.dev.confirmed': 'تأیید شد',
  'hg.table.dev.confirm': 'تأیید',
  'hg.table.dev.change': 'تغییر',
  'hg.table.dev.undo': 'برگرداندن',
  'hg.table.dev.resign': 'واگذاری',
  'hg.table.dev.toBot': 'سپردن جایگاه {seat} به ربات',
  'hg.table.dev.seatUser': 'نشاندن {name} در جایگاه {seat}',
  'hg.table.dev.finished': 'بازی تمام شد.',
  'hg.table.dev.dismiss': 'بستن',
  'hg.table.dev.create.title': 'میز تازه',
  'hg.table.dev.create.creating': 'در حال ساختن لابی با {count} جایگاه…',
  'hg.table.dev.create.failed': 'لابی ساخته نشد: {code}',
  // The URL parameter stays in Latin: it is typed as it is.
  'hg.table.dev.create.badSeats': 'از seats=3 تا seats=7 استفاده کنید.'
} as const satisfies Translation<typeof en>;
