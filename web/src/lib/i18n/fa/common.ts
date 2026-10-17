// Persian messages shared by the whole app (the shell, error pages, the
// developer pages). Complete: every key of ../en/common.ts. Kept in Latin
// script on purpose: 'app.name' (the wordmark, under the Persian name that
// 'app.nameNative' shows as decoration) and the proper nouns in 'app.legal'
// (7 Wonders Duel, Repos Production, Asmodee), 'app.foss' (MIT) and
// 'app.source' (GitHub). See ./index.ts.

import type { common as en } from '../en/common';
import type { Translation } from '../types';

export const common = {
  'text.sentence': '{text}.',
  'text.join': '{first} {second}',
  'text.phrases': '{first}. {second}',
  'text.detail': '{label}: {detail}',

  'app.name': 'Ajayeb',
  // The Persian name, shown as decoration next to the Latin one in every locale.
  'app.nameNative': 'عجایب',
  'app.tagline': 'نبرد عجایب برای دو نفر',
  'app.legal':
    'عجایب پیاده‌سازی غیررسمی و غیرتجاری قوانین بازی 7 Wonders Duel، اثر آنتوان بوزا و برونو کاتالا (Repos Production)، به دست علاقه‌مندان آن است. این برنامه وابسته به Repos Production یا Asmodee نیست و تأیید آن‌ها را ندارد؛ 7 Wonders Duel علامت تجاری آن‌هاست.',
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
  'coins.count': { other: '{count} سکه' },

  'dev.title': 'صفحه‌های توسعه‌دهندگان',
  'dev.intro': 'این صفحه‌ها فقط در نسخه‌های توسعه هستند. هنوز چیزی اینجا فهرست نشده است.',

  'dev.gallery.title': 'نگارخانه',
  'dev.gallery.intro':
    'همه‌ی آیکون‌های دست‌ساز و اجزای پایه‌ی رابط (components/shell/*)، برای بررسی چشمی در هر دو پوسته و در پهنای گوشی و رایانه. هر بخش از برنامه که آماده شود، قسمت خودش را پایین‌تر می‌افزاید.',
  'dev.gallery.viewport': 'پنجره‌ی نمایش {width}×{height}',
  'dev.gallery.icons': 'آیکون‌ها ({count})',
  'dev.gallery.shell': 'اجزای پایه‌ی رابط',
  'dev.gallery.cards': 'کارت‌ها',
  'dev.gallery.board': 'صفحه‌ی بازی',
  'dev.gallery.screens': 'صفحه‌ها',
  'dev.gallery.ornaments': 'نقش‌ونگارها، مسیر و لوگو',
  'dev.gallery.ornaments.track': 'خانه‌های مسیر نظامی در پهنای گوشی',
  'dev.gallery.ornaments.capitals':
    'پایتخت‌ها در اندازه‌های شانزده، بیست، بیست‌وچهار و چهل‌وهشت پیکسل',
  'dev.gallery.ornaments.card': 'شمسه‌ی گوشه‌ی کارت و کتیبه‌ی نوار',
  'dev.gallery.ornaments.sheet': 'جداکننده و حاشیه‌ی پنل',
  'dev.gallery.ornaments.table': 'کاشی گره: آزمون درز و بافت میز',
  'dev.gallery.ornaments.logo': 'لوگوی نوشتاری در دویست پیکسل و آیکون سایت',
  'dev.gallery.sound': 'صفحه‌ی صداها',
  'dev.gallery.sound.intro':
    'برای هر صدا یک دکمه، با بلندی صدایی که اینجا انتخاب می‌کنید (لازم نیست تنظیم «صدا» روشن باشد).',
  'dev.gallery.sound.volume': 'بلندی صدا',
  'dev.gallery.sound.cue.slide': 'لغزیدن کارت',
  'dev.gallery.sound.cue.coin': 'جرینگ سکه',
  'dev.gallery.sound.cue.flip': 'رو شدن کارت',
  'dev.gallery.sound.cue.drum': 'طبل جنگ',
  'dev.gallery.sound.cue.chime': 'زنگ علم',
  'dev.gallery.sound.cue.fanfare': 'شیپور پیروزی',
  'dev.gallery.sound.cue.tick': 'تیک گزینش',
  'dev.gallery.sound.all': 'پخش یک حرکت شلوغ',
  'dev.gallery.sound.allHint':
    'ساختنی که سکه می‌پردازد، مهره را جلو می‌برد و غارت می‌کند، همه در یک قاب، همان‌طور که بازی پخشش می‌کند.'
} as const satisfies Translation<typeof en>;
