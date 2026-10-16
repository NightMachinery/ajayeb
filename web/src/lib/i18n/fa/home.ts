// Persian messages for the Home and setup screens and Settings. Partial:
// the /dev/gallery demo strings stay English. See ./index.ts.

import type { home as en } from '../en/home';
import type { Translation } from '../types';

export const home = {
  'home.modes': 'حالت‌های بازی',
  'home.playAi': 'بازی با هوش مصنوعی',
  'home.playAiHint': 'در برابر رایانه',
  'home.passAndPlay': 'دست‌به‌دست',
  'home.passAndPlayHint': 'دو بازیکن، یک دستگاه',
  'home.playOnline': 'بازی آنلاین',
  'home.playOnlineHint': 'دوستتان را با یک پیوند دعوت کنید',
  'home.soon': 'به‌زودی',
  'home.comingSoon':
    'به‌زودی. بازی هنوز در دست ساخت است و هر حالت به محض آماده شدن همین‌جا باز می‌شود.',

  'home.continueLabel': 'ادامه‌ی بازی',
  'home.continueEntry': 'ادامه: {mode} · {code}',
  'home.mode.online': 'آنلاین',
  'home.mode.hotseat': 'دست‌به‌دست',
  'home.mode.ai': 'با هوش مصنوعی',
  'home.settings': 'تنظیمات',

  'home.joinCode': 'پیوستن با کد',

  'home.ai.title': 'حریفتان را انتخاب کنید',
  'home.ai.levelLegend': 'سطح دشواری',
  'home.ai.apprentice': 'شاگرد',
  'home.ai.apprenticeHint': 'تازه دارد یاد می‌گیرد',
  'home.ai.scholar': 'دانشمند',
  'home.ai.scholarHint': 'بازیکنی کاربلد',
  'home.ai.master': 'استاد',
  'home.ai.masterHint': 'بی‌رحم',
  'home.ai.seatLegend': 'نشان شما',
  'home.ai.seatRandom': 'تصادفی',
  'home.ai.seatSun': 'خورشید',
  'home.ai.seatMoon': 'ماه',
  'home.ai.firstRandomNote': 'آغازکننده به قید قرعه تعیین می‌شود.',
  'home.undo.toggle': 'اجازه‌ی برگرداندن حرکت',
  'home.undo.note': 'برای کل بازی ثابت است: پس از شروع بازی دیگر تغییر نمی‌کند.',
  'home.ai.start': 'شروع بازی',

  'home.hotseat.title': 'دست‌به‌دست',
  'home.hotseat.player1Label': 'بازیکن ۱ (نشان خورشید)',
  'home.hotseat.player1Default': 'بازیکن ۱',
  'home.hotseat.player2Label': 'بازیکن ۲ (نشان ماه)',
  'home.hotseat.player2Default': 'بازیکن ۲',
  'home.hotseat.firstRandomNote': 'آغازکننده به قید قرعه تعیین می‌شود.',
  'home.hotseat.passDeviceToggle': 'صفحه‌ی تحویل دستگاه',
  'home.hotseat.passDeviceHint':
    'میان نوبت‌ها، صفحه‌ای نام بازیکن بعدی را می‌گوید و حرکت قبلی را خلاصه می‌کند.',
  'home.hotseat.orientationLegend': 'جهت صفحه',
  'home.hotseat.orientationFace': 'رو به بازیکن نوبت‌دار',
  'home.hotseat.orientationFixed': 'ثابت',
  'home.hotseat.start': 'شروع بازی',

  'home.settings.title': 'تنظیمات',
  'home.settings.languageLegend': 'زبان',
  'home.settings.animationSpeedLegend': 'سرعت پویانمایی',
  'home.settings.speedNormal': 'عادی',
  'home.settings.speedFast': 'تند',
  'home.settings.speedRelaxed': 'آرام',
  'home.settings.speedOff': 'خاموش',
  'home.settings.confirmMovesLegend': 'تأیید حرکت‌ها',
  'home.settings.confirmOnlineOnly': 'فقط آنلاین',
  'home.settings.confirmAlways': 'همیشه',
  'home.settings.confirmNever': 'هرگز',
  'home.settings.colorPatterns': 'الگوهای رنگی',
  'home.settings.colorPatternsHint': 'به رنگ هر کارت الگویی می‌افزاید، برای بازیکنان کوررنگ.',
  'home.settings.cardTextLegend': 'متن کارت',
  'home.settings.cardTextNormal': 'عادی',
  'home.settings.cardTextLarge': 'درشت',
  'home.settings.sound': 'صدا',
  'home.settings.soundHint': 'صداهای کوتاه برای کارت‌ها، سکه‌ها، مهره و علم.',
  'home.settings.volumeLegend': 'بلندی صدا',
  'home.settings.volumeQuiet': 'آهسته',
  'home.settings.volumeMedium': 'متوسط',
  'home.settings.volumeLoud': 'بلند',
  'home.settings.showOpponentMove': 'نمایش جزئیات حرکت حریف',
  'home.settings.showOpponentMoveHint':
    'پس از حرکت حریف، پنجره‌ای کارهایش را فهرست می‌کند. با هر ضربه بسته می‌شود.',
  'home.settings.haptics': 'لرزش',
  'home.settings.hapticsHint': 'لرزشی سبک هنگام برداشتن کارت، و قوی‌تر برای نظامی.',

  'home.pass.title': 'دستگاه را به {name} بدهید',
  'home.pass.ready': 'آماده‌ام',
  'home.pass.undo': 'برگرداندن حرکت {name}'
} as const satisfies Translation<typeof en>;
