// Persian messages shared by the whole app. A partial table: a key left out
// here falls back to English (../en/common.ts), so only the shell a player
// meets first is translated yet. See ./index.ts.

import type { common as en } from '../en/common';
import type { Translation } from '../types';

export const common = {
  'text.sentence': '{text}.',
  'text.join': '{first} {second}',
  'text.phrases': '{first}. {second}',
  'text.detail': '{label}: {detail}',

  'app.tagline': 'نبرد عجایب برای دو نفر',
  'app.legal':
    'عجایب پیاده‌سازی غیررسمی و غیرتجاری قوانین بازی 7 Wonders Duel، اثر آنتوان بوزا و برونو کاتالا (Repos Production)، به دست علاقه‌مندان آن است. این برنامه وابسته به Repos Production یا Asmodee نیست و تأیید آن‌ها را ندارد؛ 7 Wonders Duel نشان تجاری آن‌هاست.',
  'app.getGame': 'نسخه‌ی اصلی بازی را از ناشرش تهیه کنید',
  'app.foss': 'خود عجایب نرم‌افزاری آزاد و متن‌باز است که با مجوز MIT منتشر شده است.',
  'app.source': 'کد منبع در GitHub',
  'app.home': 'بازگشت به صفحه‌ی اول',

  'error.notFound': 'این صفحه وجود ندارد.',
  'error.generic': 'مشکلی پیش آمد.',
  'error.status': 'خطای {status}',

  'theme.label': 'پوسته',
  'theme.system': 'سیستم',
  'theme.light': 'روشن',
  'theme.dark': 'تیره',

  'turn.player': 'نوبت {name}',
  // Persian keeps the noun singular after a number, so one form serves all.
  'cards.left': { '=0': 'کارتی نمانده', other: '{count} کارت مانده' },
  'coins.count': { other: '{count} سکه' }
} as const satisfies Translation<typeof en>;
