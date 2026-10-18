// Persian messages for the Home and setup screens
// (components/screens/Home, NewGameSheet, HotseatSheet, Settings,
// PassDevice). Complete: every key of ../en/home.ts, including the
// /dev/gallery demo strings. 'home.settings.languageEn' stays 'English' on
// purpose: each language is named in itself. See ./index.ts.

import type { home as en } from '../en/home';
import type { Translation } from '../types';

export const home = {
  // Home: the game modes.
  'home.loading': 'در حال آماده‌سازی…',
  'home.modes': 'حالت‌های بازی',
  'home.playAi': 'بازی با هوش مصنوعی',
  'home.playAiHint': 'در برابر رایانه',
  'home.passAndPlay': 'دست‌به‌دست',
  'home.passAndPlayHint': 'دو بازیکن، یک دستگاه',
  'home.playOnline': 'بازی آنلاین',
  'home.playOnlineHint': 'دوستتان را با یک پیوند دعوت کنید',
  'home.soon': 'به‌زودی',
  'home.comingSoon':
    'به‌زودی. بازی هنوز در دست ساخت است و هر حالت به‌محض آماده شدن همین‌جا باز می‌شود.',

  // Home: the saved-rooms "Continue" list and the Settings entry point.
  'home.continueLabel': 'ادامه‌ی بازی',
  'home.continueEntry': 'ادامه: {mode} · {code}',
  'home.continueAge': 'دوران {age}',
  'home.continueDraft': 'گزینش شگفتی‌ها',
  'home.continueDetails': '{level} · {stage}',
  'home.mode.online': 'آنلاین',
  'home.mode.hotseat': 'دست‌به‌دست',
  'home.mode.ai': 'با هوش مصنوعی',
  'home.settings': 'تنظیمات',

  // Home: joining an online game by its code.
  'home.joinCode': 'پیوستن با کد',

  // NewGameSheet (vs AI).
  'home.ai.title': 'حریفتان را انتخاب کنید',
  'home.ai.levelLegend': 'سطح دشواری',
  'home.ai.apprentice': 'شاگرد',
  'home.ai.apprenticeHint': 'تازه دارد یاد می‌گیرد',
  'home.ai.scholar': 'دانشمند',
  'home.ai.scholarHint': 'بازیکنی کاربلد',
  'home.ai.master': 'استاد',
  'home.ai.masterHint': 'بی‌رحم',
  'home.ai.seatLegend': 'نماد شما',
  'home.ai.seatRandom': 'تصادفی',
  'home.ai.seatSun': 'خورشید',
  'home.ai.seatMoon': 'ماه',
  'home.ai.firstRandomNote': 'آغازکننده به قید قرعه تعیین می‌شود.',
  // The "Allow undo" switch of the bot and hot-seat sheets (both).
  'home.undo.toggle': 'اجازه‌ی برگرداندن حرکت',
  'home.undo.note': 'برای کل بازی ثابت است: پس از شروع بازی دیگر تغییر نمی‌کند.',
  'home.ai.start': 'شروع بازی',

  // HotseatSheet (pass and play).
  'home.hotseat.title': 'دست‌به‌دست',
  'home.hotseat.player1Label': 'بازیکن یکم (نماد خورشید)',
  'home.hotseat.player1Default': 'بازیکن یکم',
  'home.hotseat.player2Label': 'بازیکن دوم (نماد ماه)',
  'home.hotseat.player2Default': 'بازیکن دوم',
  'home.hotseat.firstRandomNote': 'آغازکننده به قید قرعه تعیین می‌شود.',
  'home.hotseat.passDeviceToggle': 'صفحه‌ی تحویل دستگاه',
  'home.hotseat.passDeviceHint':
    'میان نوبت‌ها، صفحه‌ای نام بازیکن بعدی را می‌گوید و حرکت قبلی را خلاصه می‌کند.',
  'home.hotseat.orientationLegend': 'جهت صفحه',
  'home.hotseat.orientationFace': 'رو به بازیکن نوبت‌دار',
  'home.hotseat.orientationFixed': 'ثابت',
  'home.hotseat.start': 'شروع بازی',

  // Settings (the sheet and the /settings route share these).
  'home.settings.title': 'تنظیمات',
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
  'home.settings.hapticsHint': 'لرزشی سبک هنگام برداشتن کارت، و قوی‌تر برای کارت‌های نظامی.',
  // The language row. Each language is named in itself, in every locale.
  'home.settings.languageLegend': 'زبان',
  'home.settings.languageEn': 'English',
  'home.settings.languageFa': 'فارسی',

  // PassDevice (the hot-seat interstitial).
  'home.pass.title': 'دستگاه را به {name} بدهید',
  'home.pass.ready': 'آماده‌ام',
  'home.pass.undo': 'برگرداندن حرکت {name}',

  // /dev/gallery demo content for this slice's screens.
  'home.demo.continueCode': 'نمونه‌ی بازی‌های ذخیره‌شده',
  'home.demo.passSummary1': 'داریوش انبار الوار را ساخت (چهار سکه پرداخت)',
  'home.demo.passSummary2': 'مهره یک خانه به سوی شما آمد'
} as const satisfies Translation<typeof en>;
