// Persian messages of the connection layer (src/lib/session): connection
// states, the server's reject and error codes, and the temporary debug board
// with the dev room pages. Complete: every key of ../en/session.ts. The
// mode values in 'dev.new.badMode' (mode=hotseat, ai, online) are URL
// parameters and stay in Latin script. See ./index.ts.

import type { session as en } from '../en/session';
import type { Translation } from '../types';

export const session = {
  'session.conn.connecting': 'در حال اتصال…',
  'session.conn.open': 'متصل',
  'session.conn.reconnecting': 'در حال اتصال دوباره…',
  'session.conn.offline': 'آفلاین. در حال تلاش برای اتصال دوباره…',
  'session.pendingSlow': 'در انتظار سرور…',
  'session.catalogStale':
    'سرور با داده‌های کارت تازه‌تری از آنچه این صفحه نشان می‌دهد بازی می‌کند. برای به‌روزرسانی، صفحه را دوباره بارگذاری کنید.',

  'session.reject.stale': 'بازی پیش از رسیدن حرکت شما جلو رفت. لطفاً دوباره انتخاب کنید.',
  'session.reject.illegal': 'این حرکت اکنون مجاز نیست.',
  'session.reject.notYourTurn': 'نوبت شما نیست.',
  'session.reject.rate': 'درخواست‌های هم‌زمان بیش از حد است. لحظه‌ای صبر کنید و دوباره تلاش کنید.',
  'session.reject.badRequest': 'سرور نتوانست از این درخواست استفاده کند.',
  'session.reject.notAdmin': 'فقط بازیکنی که بازی را ساخته است می‌تواند راهنماهای آن را تغییر دهد.',
  'session.reject.gameOver': 'بازی تمام شده است، پس راهنماهایش دیگر تغییر نمی‌کنند.',
  'session.reject.undoOff': 'برگرداندن در این بازی خاموش است.',

  'session.error.roomArchived':
    'این بازی ذخیره‌شده پس از به‌روزرسانی سرور پایان یافته است و دیگر نمی‌توان آن را ادامه داد.',
  'session.error.roomNotFound': 'هیچ بازی‌ای با این کد نیست.',
  'session.error.roomFull': 'این بازی اتصال دیگری نمی‌پذیرد.',
  'session.error.forbidden': 'نمی‌توانید این کار را در این بازی انجام دهید.',
  'session.error.badHello': 'سرور نتوانست این مرورگر را شناسایی کند. صفحه را دوباره بارگذاری کنید.',
  'session.error.protocol':
    'نسخه‌ی تازه‌تری از بازی منتشر شده است. برای ادامه، صفحه را دوباره بارگذاری کنید.',
  'session.error.busy': 'سرور اکنون پر است. چند دقیقه‌ی دیگر دوباره تلاش کنید.',
  'session.error.badClaim':
    'این پیوند دستگاه دیگر کار نمی‌کند. از بازیکن پیوند تازه‌ای بخواهید، یا بازی را بدون آن باز کنید.',
  'session.error.unknown': 'مشکلی پیش آمد.',

  'debug.title': 'اتاق {code}',
  'debug.note': 'نمای ساده‌ی اشکال‌زدایی بازی. صفحه‌ی واقعی بازی جای آن را می‌گیرد.',
  'debug.loading': 'در انتظار بازی…',
  'debug.connection': 'اتصال',
  'debug.mode': 'حالت',
  'debug.status': 'وضعیت',
  'debug.version': 'نسخه',
  'debug.phase': 'مرحله',
  'debug.age': 'دوران',
  'debug.toMove': 'نوبت حرکت',
  'debug.active': 'جایگاه فعال',
  'debug.extraTurn': 'نوبت دوباره',
  'debug.yes': 'بله',
  'debug.no': 'خیر',
  'debug.none': 'هیچ',
  'debug.seat': 'جایگاه {seat}',
  'debug.you': 'شما',
  'debug.coins': 'سکه‌ها',
  'debug.points': 'امتیازها',
  'debug.military': 'مهره‌ی نظامی',
  'debug.city': 'شهر',
  'debug.wonders': 'شگفتی‌ها',
  'debug.built': 'ساخته‌شده',
  'debug.tokens': 'نشان‌ها',
  'debug.boardTokens': 'نشان‌های پیشرفت روی صفحه‌ی بازی',
  'debug.discard': 'کارت‌های کنارگذاشته',
  'debug.pending': 'انتخاب در انتظار',
  'debug.hiddenTokens': { other: '{count} نشان پنهان' },
  'debug.structure': 'ساختار',
  'debug.accessible': 'برداشتنی',
  'debug.choices': 'گزینه‌ها',
  'debug.noChoices': 'اکنون چیزی برای انتخاب نیست.',
  'debug.cost': { '=0': 'رایگان', other: '{count} سکه می‌پردازد' },
  'debug.gain': { other: '{count} سکه می‌گیرد' },
  'debug.undo': 'برگرداندن',
  'debug.resync': 'تازه‌سازی',
  'debug.result': 'نتیجه',
  'debug.winner': 'برنده',
  'debug.shared': 'پیروزی مشترک',
  'debug.rematch': 'درخواست بازی دوباره',
  'debug.toRematch': 'رفتن به بازی دوباره',
  'debug.thinking': 'جایگاه {seat} در حال فکر کردن است…',

  'dev.new.title': 'اتاق تازه',
  'dev.new.creating': 'در حال ساختن اتاق {mode}…',
  'dev.new.failed': 'ساختن اتاق ممکن نشد: {code}',
  'dev.new.badMode': 'حالت ناشناخته. از mode=hotseat، ai یا online استفاده کنید.',
  'dev.fixture.title': 'نمونه‌های ضبط‌شده',
  'dev.fixture.unknown': 'هیچ نمونه‌ی ضبط‌شده‌ای به نام {name} نیست.'
} as const satisfies Translation<typeof en>;
